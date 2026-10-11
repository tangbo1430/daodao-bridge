package business

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/account"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/adapter"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/media"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/security"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Service struct {
	Accept       func(context.Context, domain.EventEnvelope) (bool, error)
	syncMu       sync.Mutex
	peerSettings sync.Map
	Media        *media.Service
	Store        store.Store
	Accounts     *account.Service
	Tang         *adapter.TangSengClient
	Tencent      *adapter.TencentClient
	Box          *security.SecretBox
	mu           sync.Mutex
}

func New(st store.Store, accounts *account.Service, tang *adapter.TangSengClient, tencent *adapter.TencentClient, secret string) *Service {
	s := &Service{Store: st, Accounts: accounts, Tang: tang, Tencent: tencent}
	if secret != "" {
		key := sha256.Sum256([]byte("hw-bridge-friend-state-v1\x00" + secret))
		s.Box, _ = security.NewSecretBox(key[:])
	}
	return s
}

type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string { return e.Message }

func (s *Service) TencentConversation(ctx context.Context, event domain.MessageEvent) (bool, error) {
	if profile, err := s.Accounts.Profile(ctx, event.FromAccount); err != nil || profile.Status != "confirmed" {
		return false, nil
	}
	if event.ConversationType == domain.ConversationGroup {
		return strings.HasPrefix(event.ConversationID, "dd_"), nil
	}
	peer, err := s.Store.FindIdentity(ctx, domain.SourceTang, event.ToAccount)
	if errors.Is(err, store.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return peer.EntityType == "user" && peer.TargetID == event.ToAccount, nil
}

func fail(code int, msg string) error { return &Error{code, msg} }
func (s *Service) owner(ctx context.Context, owner string) (domain.Identity, error) {
	if s.Accounts == nil || s.Tang == nil || s.Tencent == nil || s.Store == nil || s.Box == nil {
		return domain.Identity{}, fail(503, "好友服务暂不可用，请稍后重试")
	}
	r, err := s.Accounts.Profile(ctx, owner)
	if err != nil || r.Status != "confirmed" {
		return domain.Identity{}, fail(403, "聊天服务尚未就绪，请稍后重试")
	}
	return r.Identity, nil
}
func (s *Service) state(ctx context.Context, owner, peer string) (domain.FriendState, error) {
	v, err := s.Store.GetFriendState(ctx, owner, peer)
	if errors.Is(err, store.ErrNotFound) {
		return domain.FriendState{Owner: owner, Peer: peer, Status: "discovered"}, nil
	}
	return v, err
}
func (s *Service) SaveSearch(ctx context.Context, owner string, user adapter.SearchUser) error {
	if user.UID == "fileHelper" {
		return fail(400, "该联系人暂不支持此操作")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, err := s.state(ctx, owner, user.UID)
	if err != nil {
		return err
	}
	if s.Box == nil {
		return fail(503, "好友服务暂不可用，请稍后重试")
	}
	if user.Vercode != "" {
		v.VercodeCipher, err = s.Box.Encrypt(user.Vercode)
		if err != nil {
			return err
		}
	}
	v.Name = user.Name
	if v.Name == "" {
		v.Name = user.Nick
	}
	v.Avatar = user.FaceURL
	if v.Avatar == "" {
		v.Avatar = user.Avatar
	}
	return s.Store.PutFriendState(ctx, v)
}
func (s *Service) Friend(ctx context.Context, owner, peer, action, wording, vercode string) (domain.FriendState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, err := s.owner(ctx, owner)
	if err != nil {
		return domain.FriendState{}, err
	}
	if peer == "" || peer == "fileHelper" || peer == id.TargetID || peer == owner {
		return domain.FriendState{}, fail(400, "好友账号无效")
	}
	// Only a known TangSeng projection may be changed by the public facade.
	mapped, err := s.Store.FindIdentity(ctx, domain.SourceTang, peer)
	if err != nil || mapped.EntityType != "user" || mapped.TargetID != peer {
		return domain.FriendState{}, fail(404, "请先搜索用户")
	}
	v, err := s.state(ctx, owner, peer)
	if err != nil {
		return v, err
	}
	switch action {
	case "apply":
		if v.Status == "active" {
			return v, nil
		}
		if err = s.apply(ctx, id, &v, wording, vercode); err != nil {
			return v, err
		}
	case "accept":
		if v.Status == "expired_in" {
			if err = s.apply(ctx, id, &v, "", ""); err != nil {
				return v, err
			}
			break
		}
		if v.Status == "active" {
			return v, nil
		}
		if v.Status != "pending_in" && v.Status != "accepting" {
			return v, fail(409, "没有可处理的好友申请")
		}
		if v.Status == "pending_in" {
			if v.ExpiresAt <= time.Now().Unix() {
				if err = s.apply(ctx, id, &v, "", ""); err != nil {
					return v, err
				}
				break
			}
			token, e := s.Box.Decrypt(v.TokenCipher)
			if e != nil {
				return v, fmt.Errorf("好友申请凭据无法解密")
			}
			if err = s.Tang.FriendSure(ctx, id.TargetID, token); err != nil {
				var reason *domain.DeliveryError
				if errors.As(err, &reason) && reason.Reason == "friend_request_expired" {
					if err = s.apply(ctx, id, &v, "", ""); err != nil {
						return v, err
					}
					break
				}
				return v, err
			}
			v.Status = "accepting"
			if err = s.Store.PutFriendState(ctx, v); err != nil {
				return v, err
			}
		}
		if err = s.Tencent.AddFriend(ctx, owner, peer); err != nil {
			return v, err
		}
		v.Status = "active"
		v.TokenCipher = ""
	case "refuse":
		if v.Status == "refused" {
			return v, nil
		}
		if v.Status != "pending_in" {
			return v, fail(409, "没有可拒绝的好友申请")
		}
		if err = s.Tang.FriendRefuse(ctx, id.TargetID, peer); err != nil {
			return v, err
		}
		v.Status = "refused"
		v.TokenCipher = ""
	case "delete":
		if v.Status == "deleted" {
			return v, nil
		}
		if v.Status != "deleting" {
			if err = s.Tang.FriendDelete(ctx, id.TargetID, peer); err != nil {
				return v, err
			}
			v.Status = "deleting"
			if err = s.Store.PutFriendState(ctx, v); err != nil {
				return v, err
			}
		}
		if err = s.Tencent.SetFriendEdge(ctx, owner, peer, false); err != nil {
			return v, err
		}
		v.Status = "deleted"
		v.TokenCipher = ""
	case "block", "unblock":
		wanted := action == "block"
		if v.Blocked == wanted {
			return v, nil
		}
		if err = s.Tang.Blacklist(ctx, id.TargetID, peer, wanted); err != nil {
			return v, err
		}
		v.Blocked = wanted
	default:
		return v, fail(400, "未知好友操作")
	}
	return v, s.Store.PutFriendState(ctx, v)
}
func (s *Service) apply(ctx context.Context, id domain.Identity, v *domain.FriendState, wording, vercode string) error {
	if vercode == "" && v.VercodeCipher != "" {
		var err error
		vercode, err = s.Box.Decrypt(v.VercodeCipher)
		if err != nil {
			return fmt.Errorf("搜索凭据无法解密")
		}
	}
	if vercode == "" {
		users, err := s.Tang.SearchUser(ctx, v.Peer)
		if err != nil {
			return err
		}
		for _, u := range users {
			if u.UID == v.Peer {
				vercode = u.Vercode
				break
			}
		}
	}
	if vercode == "" {
		return fail(409, "请通过搜索账号或名片添加")
	}
	if err := s.Tang.FriendApply(ctx, id.TargetID, v.Peer, wording, vercode); err != nil {
		var reason *domain.DeliveryError
		if !errors.As(err, &reason) || reason.Reason != "already_friends" {
			return err
		}
		if err = s.Tencent.AddFriend(ctx, id.SourceID, v.Peer); err != nil {
			return err
		}
		v.Status = "active"
	} else {
		v.Status = "pending_out"
	}
	v.Remark = wording
	v.TokenCipher = ""
	v.ExpiresAt = time.Now().Add(30 * 24 * time.Hour).Unix()
	var err error
	v.VercodeCipher, err = s.Box.Encrypt(vercode)
	return err
}
func (s *Service) Requests(ctx context.Context, owner string) ([]map[string]any, error) {
	if _, err := s.owner(ctx, owner); err != nil {
		return nil, err
	}
	list, err := s.Store.ListFriendStates(ctx, owner)
	if err != nil {
		return nil, err
	}
	out := []map[string]any{}
	for _, v := range list {
		if v.Status == "pending_in" || v.Status == "expired_in" || v.Status == "accepting" || v.RequestID != "" {
			out = append(out, map[string]any{"user_id": v.Peer, "nick": v.Name, "face_url": v.Avatar, "remark": v.Remark, "status": v.Status, "expires_at": v.ExpiresAt})
		}
	}
	return out, nil
}
func (s *Service) Handle(ctx context.Context, e domain.BusinessEvent) error {
	if e.Command == "Bridge.GroupRoleNotice" {
		return s.sendGroupRoleNotice(ctx, e)
	}
	if e.Source == domain.SourceTang && (e.Command == "syncConversation" || e.Command == "syncMessage") {
		id, err := s.Store.FindIdentityByTarget(ctx, domain.SourceTang, e.Owner)
		if err != nil {
			return err
		}
		_, err = s.SyncAccount(ctx, id.SourceID)
		return err
	}
	if e.Source == domain.SourceTencent && e.Command == "Bridge.AccountDeleted" {
		var req struct {
			Owner string `json:"From_Account"`
		}
		if err := json.Unmarshal(e.Payload, &req); err != nil {
			return err
		}
		if req.Owner == "" {
			return fail(400, "注销账号缺失")
		}
		return s.DeleteAccount(ctx, req.Owner)
	}
	if e.Command == "C2C.CallbackAfterMsgWithDraw" || e.Command == "C2C.CallbackAfterMsgReport" || e.Command == "Group.CallbackAfterRecallMsg" || e.Command == "messageRevoke" || e.Command == "syncMessageExtra" {
		return s.messageStateEvent(ctx, e)
	}
	if e.Source == domain.SourceTencent && e.Command == "Profile.Sync" {
		var req account.Request
		if err := json.Unmarshal(e.Payload, &req); err != nil {
			return err
		}
		return s.Profile(ctx, req.TencentUserID, req.DisplayName, req.AvatarURL)
	}
	if strings.HasPrefix(e.Command, "Group.") {
		return s.tencentGroupEvent(ctx, e)
	}
	if e.Command == "channelUpdate" || e.Command == "memberUpdate" || strings.HasPrefix(e.Command, "group") {
		if e.ChannelID != "" {
			return s.groupCommand(ctx, e)
		}
		var command struct {
			Param struct {
				GroupNo     string `json:"group_no"`
				ChannelType int    `json:"channel_type"`
			} `json:"param"`
		}
		if err := json.Unmarshal(e.Payload, &command); err != nil {
			return err
		}
		if command.Param.GroupNo != "" || command.Param.ChannelType == 2 {
			return s.groupCommand(ctx, e)
		}
	}
	if e.Source == domain.SourceTencent {
		return s.tencentEvent(ctx, e)
	}
	id, err := s.Store.FindIdentityByTarget(ctx, domain.SourceTang, e.Owner)
	if err != nil {
		return err
	}
	if id.EntityType != "user" || id.SourceID == id.TargetID {
		return nil
	}
	if _, err = s.owner(ctx, id.SourceID); err != nil {
		return err
	}
	var cmd struct {
		Param json.RawMessage `json:"param"`
	}
	if err = json.Unmarshal(e.Payload, &cmd); err != nil {
		return err
	}
	if len(cmd.Param) == 0 {
		return fmt.Errorf("%s command param missing", e.Command)
	}
	var param struct {
		FromUID     string `json:"from_uid"`
		FromName    string `json:"from_name"`
		ApplyUID    string `json:"apply_uid"`
		UID         string `json:"uid"`
		ToUID       string `json:"to_uid"`
		Name        string `json:"apply_name"`
		Remark      string `json:"remark"`
		Token       string `json:"token"`
		ChannelID   string `json:"channel_id"`
		ChannelType int    `json:"channel_type"`
	}
	if err = json.Unmarshal(cmd.Param, &param); err != nil {
		return err
	}
	switch e.Command {
	case "friendRequest":
		if param.ApplyUID == "" || param.Token == "" || (param.ToUID != "" && param.ToUID != id.TargetID) {
			return fmt.Errorf("invalid friendRequest parameters")
		}
		if hw, lookupErr := s.Store.FindIdentityByTarget(ctx, domain.SourceTang, param.ApplyUID); lookupErr == nil && hw.SourceID != hw.TargetID {
			return nil
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		v, err := s.state(ctx, id.SourceID, param.ApplyUID)
		if err != nil {
			return err
		}
		if v.RequestID == e.EventID && v.Status != "notifying" {
			return nil
		}
		identity, err := s.friendIdentity(ctx, param.ApplyUID, param.Name, v)
		if err != nil {
			return err
		}
		if _, err = s.Tencent.EnsureUser(ctx, identity); err != nil {
			return err
		}
		if err = s.Store.UpsertIdentity(ctx, identity); err != nil {
			return err
		}
		if v.RequestID != e.EventID {
			v.Status = "notifying"
			v.Name = identity.DisplayName
			v.Avatar = identity.AvatarURL
			v.Remark = param.Remark
			v.RequestID = e.EventID
			v.ExpiresAt = time.Now().Add(15 * 24 * time.Hour).Unix()
			v.TokenCipher, err = s.Box.Encrypt(param.Token)
			if err != nil {
				return err
			}
			if err = s.Store.PutFriendState(ctx, v); err != nil {
				return err
			}
		}
		allow, policyErr := s.Tencent.FriendAllowType(ctx, id.SourceID)
		if policyErr != nil {
			return policyErr
		}
		switch allow {
		case "AllowType_Type_NeedConfirm":
			err = s.Tencent.FriendAdd(ctx, param.ApplyUID, id.SourceID, param.Remark, false)
		case "AllowType_Type_DenyAny":
			err = &adapter.TencentError{Code: 30516, Operation: "friend_add"}
		default:
			// ForceAddFlags=0 would auto-accept an AllowAny account. Keep the
			// TangSeng application pending and expose an actionable custom card.
			err = s.Tencent.SendFriendApplication(ctx, param.ApplyUID, id.SourceID, e.EventID, identity.DisplayName, param.Remark)
		}
		if err != nil {
			var remote *adapter.TencentError
			if errors.As(err, &remote) && (remote.Code == 30516 || remote.Code == 30515 || remote.Code == 30525) {
				if err = s.Tang.FriendRefuse(ctx, id.TargetID, param.ApplyUID); err != nil {
					return err
				}
				v.Status = "refused"
				v.TokenCipher = ""
				return s.Store.PutFriendState(ctx, v)
			}
			return err
		}
		v.Status = "pending_in"
		return s.Store.PutFriendState(ctx, v)
	case "friendAccept":
		peer := param.FromUID
		if peer == id.TargetID {
			peer = param.ToUID
		} else if param.ToUID != id.TargetID {
			return fmt.Errorf("friendAccept recipient mismatch")
		}
		if peer == "" {
			return fmt.Errorf("friendAccept peer missing")
		}
		if hw, e := s.Store.FindIdentityByTarget(ctx, domain.SourceTang, peer); e == nil && hw.SourceID != hw.TargetID {
			return nil
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		v, err := s.state(ctx, id.SourceID, peer)
		if err != nil {
			return err
		}
		if v.Status == "active" {
			return nil
		}
		peerName := ""
		if param.FromUID == peer {
			peerName = param.FromName
		}
		identity, err := s.friendIdentity(ctx, peer, peerName, v)
		if err != nil {
			return err
		}
		if _, err = s.Tencent.EnsureUser(ctx, identity); err != nil {
			return err
		}
		if err = s.Store.UpsertIdentity(ctx, identity); err != nil {
			return err
		}
		if err = s.Tencent.AddFriend(ctx, id.SourceID, peer); err != nil {
			return err
		}
		v.Status = "active"
		v.TokenCipher = ""
		v.Name = identity.DisplayName
		v.Avatar = identity.AvatarURL
		return s.Store.PutFriendState(ctx, v)
	case "userUpdate", "userAvatarUpdate":
		uid := param.UID
		if uid == "" {
			uid = param.ChannelID
		}
		if uid != "" {
			return s.SyncShadowProfile(ctx, *id, uid)
		}
		return s.SyncFriends(ctx, *id)
	case "channelUpdate":
		if param.ChannelType == 1 && param.ChannelID != "" {
			s.peerSettings.Delete(id.TargetID + ":" + param.ChannelID)
		}
		// Native profile edits emit channelUpdate, including when the new
		// name equals an older friend snapshot. Refresh the named projection
		// explicitly instead of relying on that snapshot's change detection.
		relationErr := s.SyncFriends(ctx, *id)
		if param.ChannelType == 1 && param.ChannelID != "" {
			f, lookupErr := s.Store.GetFriendState(ctx, id.SourceID, param.ChannelID)
			if lookupErr == nil && (f.Status == "deleted" || f.Status == "peer_deleted") {
				return relationErr
			}
			if lookupErr != nil && !errors.Is(lookupErr, store.ErrNotFound) {
				return errors.Join(relationErr, lookupErr)
			}
			return errors.Join(relationErr, s.SyncShadowProfile(ctx, *id, param.ChannelID))
		}
		return relationErr
	case "friendDeleted":
		return s.SyncFriends(ctx, *id)
	}
	return nil
}
func (s *Service) tencentEvent(ctx context.Context, e domain.BusinessEvent) error {
	var event struct {
		PairList []struct {
			From string `json:"From_Account"`
			To   string `json:"To_Account"`
		} `json:"PairList"`
	}
	if err := json.Unmarshal(e.Payload, &event); err != nil {
		return err
	}
	for _, p := range event.PairList {
		owner, peer := p.From, p.To
		if _, err := s.Accounts.Profile(ctx, owner); err != nil {
			owner, peer = peer, owner
			if _, err = s.Accounts.Profile(ctx, owner); err != nil {
				continue
			}
		}
		mapped, err := s.Store.FindIdentity(ctx, domain.SourceTang, peer)
		if errors.Is(err, store.ErrNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		if mapped.TargetID != peer {
			continue
		}
		action := ""
		switch e.Command {
		case "Sns.CallbackFriendAdd":
			v, err := s.Store.GetFriendState(ctx, owner, peer)
			if errors.Is(err, store.ErrNotFound) {
				continue
			}
			if err != nil {
				return err
			}
			if v.Status != "pending_in" && v.Status != "accepting" {
				continue
			}
			action = "accept"
		case "Sns.CallbackFriendDelete":
			// A deletion emitted under a shadow identity is a mirror echo,
			// never permission to delete the Huawei user's native relation.
			if owner != p.From {
				continue
			}
			action = "delete"
		case "Sns.CallbackBlackListAdd":
			if owner != p.From {
				continue
			}
			action = "block"
		case "Sns.CallbackBlackListDelete":
			if owner != p.From {
				continue
			}
			action = "unblock"
		}
		if action != "" {
			if _, err = s.Friend(ctx, owner, peer, action, "", ""); err != nil {
				return err
			}
		}
	}
	return nil
}

// SyncFriends is a bounded API pagination loop; no periodic reliability work is added.
func (s *Service) SyncFriends(ctx context.Context, id domain.Identity) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	version := int64(0)
	var syncErrors []error
	for page := 0; page < 100; page++ {
		var raw json.RawMessage
		if err := s.Tang.UserRequest(ctx, id.TargetID, http.MethodGet, "/v1/friend/sync?version="+strconv.FormatInt(version, 10)+"&limit=200&api_version=1", nil, &raw); err != nil {
			return err
		}
		var friends []struct {
			UID       string `json:"uid"`
			Name      string `json:"name"`
			Avatar    string `json:"avatar"`
			Version   int64  `json:"version"`
			Deleted   int    `json:"is_deleted"`
			BeDeleted int    `json:"be_deleted"`
			Destroyed int    `json:"is_destroy"`
		}
		if err := json.Unmarshal(raw, &friends); err != nil {
			return fmt.Errorf("friend sync response does not match contract: %w", err)
		}
		next := version
		for _, f := range friends {
			if f.UID == "" {
				return fmt.Errorf("friend sync uid missing")
			}
			if f.Version > next {
				next = f.Version
			}
			if f.UID == "fileHelper" {
				continue
			}
			// Native Huawei-to-Huawei friendships remain Tencent-only.
			if hw, err := s.Store.FindIdentityByTarget(ctx, domain.SourceTang, f.UID); err == nil && hw.SourceID != hw.TargetID {
				continue
			}
			// One failed relation must not prevent later active friends from
			// receiving their avatars and profiles. Report failures after
			// processing the remaining page, so failed items still retry.
			err := func() error {
				v, err := s.state(ctx, id.SourceID, f.UID)
				if err != nil {
					return err
				}
				if f.Deleted != 0 || f.BeDeleted != 0 || f.Destroyed != 0 {
					// Historical Tang tombstones may describe users never
					// mirrored by this Bridge. Do not import or mutate those
					// accounts merely to remove an absent mirror relation.
					_, mappedErr := s.Store.FindIdentity(ctx, domain.SourceTang, f.UID)
					if errors.Is(mappedErr, store.ErrNotFound) {
						v.Status = "peer_deleted"
						v.TokenCipher = ""
						return s.Store.PutFriendState(ctx, v)
					}
					if mappedErr != nil {
						return mappedErr
					}
					if f.Destroyed != 0 {
						if err = s.Tencent.UpdateProfile(ctx, f.UID, "已注销用户", ""); err != nil {
							return err
						}
						mapped, e := s.Store.FindIdentity(ctx, domain.SourceTang, f.UID)
						if e == nil {
							mapped.DisplayName = "已注销用户"
							if e = s.Store.UpsertIdentity(ctx, *mapped); e != nil {
								return e
							}
						}
						if err = s.Store.PutCursor(ctx, domain.SyncCursor{Key: "shadow:" + f.UID, Aux: "deleted", Value: time.Now().Unix()}); err != nil {
							return err
						}
					}
					if f.Deleted != 0 || f.Destroyed != 0 {
						if err = s.Tencent.SetFriendEdge(ctx, id.SourceID, f.UID, false); err != nil {
							return err
						}
					}
					if f.BeDeleted != 0 || f.Destroyed != 0 {
						if err = s.Tencent.SetFriendEdge(ctx, f.UID, id.SourceID, false); err != nil {
							return err
						}
					}
					v.Status = "peer_deleted"
					if f.Deleted != 0 || f.Destroyed != 0 {
						v.Status = "deleted"
					}
					v.TokenCipher = ""
				} else {
					if s.Media != nil {
						avatar, avatarErr := s.Media.AvatarURL(ctx, "users", f.UID)
						if avatarErr != nil {
							return avatarErr
						}
						f.Avatar = avatar
					}
					identity := domain.Identity{Source: domain.SourceTang, SourceID: f.UID, Target: domain.SourceTencent, TargetID: f.UID, EntityType: "user", DisplayName: f.Name, AvatarURL: f.Avatar, Shadow: true}
					if _, err = s.Tencent.EnsureUser(ctx, identity); err != nil {
						return err
					}
					if err = s.Store.UpsertIdentity(ctx, identity); err != nil {
						return err
					}
					if v.Status != "active" {
						if err = s.Tencent.AddFriend(ctx, id.SourceID, f.UID); err != nil {
							return err
						}
					}
					if f.Name != v.Name || f.Avatar != v.Avatar {
						if err = s.Tencent.UpdateProfile(ctx, f.UID, f.Name, f.Avatar); err != nil {
							return err
						}
					}
					v.Status = "active"
					v.TokenCipher = ""
					v.Name = f.Name
					v.Avatar = f.Avatar
				}
				if err = s.Store.PutFriendState(ctx, v); err != nil {
					return err
				}
				return nil
			}()
			if err != nil {
				syncErrors = append(syncErrors, fmt.Errorf("friend sync peer=%s: %w", f.UID, err))
			}
		}
		if len(friends) < 200 {
			return errors.Join(syncErrors...)
		}
		if next <= version {
			return fmt.Errorf("friend sync cursor did not advance")
		}
		version = next
	}
	return fmt.Errorf("friend sync page limit exceeded")
}
