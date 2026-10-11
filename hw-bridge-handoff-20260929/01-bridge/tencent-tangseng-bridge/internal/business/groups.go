package business

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/adapter"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/media"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type GroupRequest struct {
	Owner     string   `json:"tencent_user_id"`
	GroupID   string   `json:"group_id"`
	RequestID string   `json:"request_id"`
	Name      string   `json:"name"`
	Notice    string   `json:"notice"`
	Members   []string `json:"members"`
	Field     string   `json:"field"`
	Value     string   `json:"value"`
}

func groupNumber(id string) (string, error) {
	if !strings.HasPrefix(id, "dd_") || len(id) <= 3 {
		return "", fail(400, "该群暂不支持此操作")
	}
	return strings.TrimPrefix(id, "dd_"), nil
}
func (s *Service) tangUser(ctx context.Context, uid string) (string, error) {
	if uid == "fileHelper" {
		return "", fail(400, "该联系人暂不支持此操作")
	}
	if identity, err := s.Store.FindIdentity(ctx, domain.SourceTencent, uid); err == nil && identity.EntityType == "user" {
		return identity.TargetID, nil
	} else if err != nil && !errors.Is(err, store.ErrNotFound) {
		return "", err
	}
	if identity, err := s.Store.FindIdentity(ctx, domain.SourceTang, uid); err == nil && identity.EntityType == "user" && identity.TargetID == uid {
		return uid, nil
	}
	return "", fail(409, "成员聊天服务尚未就绪，请稍后重试")
}
func (s *Service) Group(ctx context.Context, action string, req GroupRequest) (domain.GroupState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, err := s.owner(ctx, req.Owner)
	if err != nil {
		return domain.GroupState{}, err
	}
	members := []string{}
	seen := map[string]bool{}
	for _, member := range req.Members {
		uid, err := s.tangUser(ctx, member)
		if err != nil {
			return domain.GroupState{}, err
		}
		if uid != id.TargetID && !seen[uid] {
			seen[uid] = true
			members = append(members, uid)
		}
	}
	if action == "create" {
		if req.RequestID == "" || len(req.RequestID) > 128 || len(members) == 0 {
			return domain.GroupState{}, fail(400, "request_id 和群成员不能为空")
		}
		groups, err := s.Store.ListGroupStates(ctx)
		if err != nil {
			return domain.GroupState{}, err
		}
		for _, g := range groups {
			if g.RequestOwner == req.Owner && g.RequestID == req.RequestID {
				if err = s.finishInvites(ctx, id, &g); err != nil {
					return g, err
				}
				return s.syncGroup(ctx, id, g.Number, "")
			}
		}
		// TangSeng only allows friends in the initial creation request. Remaining
		// members are invited after creation using the same actor, as specified.
		var friends []struct {
			UID       string `json:"uid"`
			Deleted   int    `json:"is_deleted"`
			BeDeleted int    `json:"be_deleted"`
			Version   int64  `json:"version"`
		}
		version := int64(0)
		friendSet := map[string]bool{}
		for page := 0; page < 100; page++ {
			if err = s.Tang.UserRequest(ctx, id.TargetID, http.MethodGet, "/v1/friend/sync?api_version=1&limit=200&version="+strconv.FormatInt(version, 10), nil, &friends); err != nil {
				return domain.GroupState{}, err
			}
			next := version
			for _, f := range friends {
				if f.Deleted == 0 && f.BeDeleted == 0 {
					friendSet[f.UID] = true
				}
				if f.Version > next {
					next = f.Version
				}
			}
			if len(friends) < 200 {
				break
			}
			if next <= version {
				return domain.GroupState{}, fmt.Errorf("friend cursor stalled")
			}
			version = next
		}
		initial, remaining := []string{}, []string{}
		for _, uid := range members {
			if friendSet[uid] {
				initial = append(initial, uid)
			} else {
				remaining = append(remaining, uid)
			}
		}
		if len(initial) == 0 {
			return domain.GroupState{}, fail(409, "请至少选择一位好友建群")
		}
		var created struct {
			Number string `json:"group_no"`
		}
		if err = s.Tang.UserRequest(ctx, id.TargetID, http.MethodPost, "/v1/group/create", map[string]any{"name": req.Name, "members": initial}, &created); err != nil {
			return domain.GroupState{}, err
		}
		if created.Number == "" {
			return domain.GroupState{}, fmt.Errorf("group create response missing group_no")
		}
		g := domain.GroupState{Number: created.Number, TencentID: "dd_" + created.Number, Status: 1, RequestOwner: req.Owner, RequestID: req.RequestID, PendingInvites: remaining}
		if err = s.Store.PutGroupState(ctx, g); err != nil {
			return g, err
		}
		if err = s.finishInvites(ctx, id, &g); err != nil {
			return g, err
		}
		return s.syncGroup(ctx, id, g.Number, "")
	}
	number, err := groupNumber(req.GroupID)
	if err != nil {
		return domain.GroupState{}, err
	}
	path := "/v1/groups/" + url.PathEscape(number)
	switch action {
	case "invite", "remove":
		if len(members) == 0 {
			return domain.GroupState{}, fail(400, "群成员不能为空")
		}
		method := http.MethodPost
		if action == "remove" {
			method = http.MethodDelete
		}
		if err = s.Tang.UserRequest(ctx, id.TargetID, method, path+"/members", map[string]any{"members": members}, nil); err != nil {
			return domain.GroupState{}, err
		}
	case "exit":
		if err = s.Tang.UserRequest(ctx, id.TargetID, http.MethodPost, path+"/exit", nil, nil); err != nil {
			return domain.GroupState{}, err
		}
		return s.syncGroup(ctx, id, number, "")
	case "disband":
		g, err := s.Store.GetGroupState(ctx, number)
		if err != nil {
			return g, err
		}
		if g.Status == 2 && !g.MirrorReady {
			return g, nil
		}
		if g.Status != 2 {
			var current domain.GroupState
			if err = s.Tang.UserRequest(ctx, id.TargetID, http.MethodGet, path, nil, &current); err != nil {
				return g, err
			}
			if current.Number != number {
				return g, fmt.Errorf("group detail identifier mismatch")
			}
			if current.Status != 2 {
				if err = s.Tang.UserRequest(ctx, id.TargetID, http.MethodDelete, path+"/disband", nil, nil); err != nil {
					return g, err
				}
			}
		}
		if err = s.Tencent.DestroyMirrorGroup(ctx, g.TencentID); err != nil {
			return g, err
		}
		g.Status = 2
		g.MirrorReady = false
		return g, s.Store.PutGroupState(ctx, g)
	case "update":
		switch req.Field {
		case "name", "notice":
			err = s.Tang.UserRequest(ctx, id.TargetID, http.MethodPut, path, map[string]string{req.Field: req.Value}, nil)
		case "avatar":
			if s.Media == nil {
				return domain.GroupState{}, fail(503, "头像同步服务未配置")
			}
			err = s.Media.UploadAvatar(ctx, id.TargetID, number, req.Value)
		case "invite":
			if req.Value != "0" && req.Value != "1" {
				return domain.GroupState{}, fail(400, "邀请确认参数无效")
			}
			err = s.Tang.UserRequest(ctx, id.TargetID, http.MethodPut, path, map[string]string{"invite": req.Value}, nil)
		case "manager", "member":
			if len(members) == 0 {
				return domain.GroupState{}, fail(400, "群成员不能为空")
			}
			method := http.MethodPost
			if req.Field == "member" {
				method = http.MethodDelete
			}
			err = s.Tang.UserRequest(ctx, id.TargetID, method, path+"/managers", members, nil)
		case "mute_member":
			if len(members) != 1 {
				return domain.GroupState{}, fail(400, "请选择一位群成员")
			}
			seconds, parseErr := strconv.Atoi(req.Value)
			if parseErr != nil {
				return domain.GroupState{}, fail(400, "禁言时长无效")
			}
			key := 0
			for index, duration := range []int{60, 600, 3600, 86400, 604800, 2592000} {
				if seconds == duration {
					key = index + 1
				}
			}
			if seconds != 0 && key == 0 {
				return domain.GroupState{}, fail(400, "可选择禁言 1 分钟、10 分钟、1 小时、1 天、7 天或 30 天")
			}
			action := 0
			if seconds > 0 {
				action = 1
			}
			err = s.Tang.UserRequest(ctx, id.TargetID, http.MethodPost, path+"/forbidden_with_member", map[string]any{"member_uid": members[0], "action": action, "key": key}, nil)
		case "forbidden":
			if req.Value != "0" && req.Value != "1" {
				return domain.GroupState{}, fail(400, "禁言参数无效")
			}
			err = s.Tang.UserRequest(ctx, id.TargetID, http.MethodPost, path+"/forbidden/"+req.Value, nil, nil)
		case "owner":
			var uid string
			uid, err = s.tangUser(ctx, req.Value)
			if err == nil {
				err = s.Tang.UserRequest(ctx, id.TargetID, http.MethodPost, path+"/transfer/"+url.PathEscape(uid), nil, nil)
			}
		default:
			return domain.GroupState{}, fail(400, "暂不支持此群资料修改")
		}
		if err != nil {
			return domain.GroupState{}, err
		}
	case "sync":
	default:
		return domain.GroupState{}, fail(400, "未知群操作")
	}
	var change groupProfileChange
	if action == "update" {
		change = newGroupProfileChange(id.TargetID, req.Field, req.Value)
	}
	return s.syncGroupWithChange(ctx, id, number, "", change)
}

func (s *Service) finishInvites(ctx context.Context, actor domain.Identity, g *domain.GroupState) error {
	if len(g.PendingInvites) == 0 {
		return nil
	}
	if err := s.Tang.UserRequest(ctx, actor.TargetID, http.MethodPost, "/v1/groups/"+url.PathEscape(g.Number)+"/members", map[string]any{"members": g.PendingInvites}, nil); err != nil {
		return err
	}
	g.PendingInvites = nil
	return s.Store.PutGroupState(ctx, *g)
}
func truncateUTF8(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	value = value[:limit]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value
}
func (s *Service) syncGroup(ctx context.Context, actor domain.Identity, number, speaker string, mentioned ...string) (result domain.GroupState, retErr error) {
	return s.syncGroupWithChange(ctx, actor, number, speaker, groupProfileChange{}, mentioned...)
}

func (s *Service) syncGroupWithChange(ctx context.Context, actor domain.Identity, number, speaker string, change groupProfileChange, mentioned ...string) (result domain.GroupState, retErr error) {
	step := "load-state"
	defer func() {
		if retErr != nil {
			retErr = &GroupError{Step: step, Err: retErr}
		}
	}()
	old, err := s.Store.GetGroupState(ctx, number)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return old, err
	}
	var g domain.GroupState
	path := "/v1/groups/" + url.PathEscape(number)
	step = "read-tang-group"
	if err = s.Tang.UserRequest(ctx, actor.TargetID, http.MethodGet, path, nil, &g); err != nil {
		return old, err
	}
	if g.Number == "" || g.Number != number {
		return old, fmt.Errorf("group detail identifier mismatch")
	}
	g.TencentID = "dd_" + number
	g.RequestID = old.RequestID
	g.RequestOwner = old.RequestOwner
	g.PendingInvites = old.PendingInvites
	g.Members = nil
	if g.Status == 2 {
		if old.MirrorReady {
			if err = s.Tencent.DestroyMirrorGroup(ctx, g.TencentID); err != nil {
				return g, err
			}
		}
		return g, s.Store.PutGroupState(ctx, g)
	}
	version := int64(0)
	step = "read-tang-members"
	byUID := map[string]domain.GroupMember{}
	for page := 0; page < 100; page++ {
		var rows []domain.GroupMember
		memberPath := path + "/membersync?limit=1000&version=" + strconv.FormatInt(version, 10)
		if g.GroupType == 1 {
			memberPath = path + "/members?limit=1000&page=" + strconv.Itoa(page+1)
		}
		if err = s.Tang.UserRequest(ctx, actor.TargetID, http.MethodGet, memberPath, nil, &rows); err != nil {
			return g, err
		}
		next := version
		for _, m := range rows {
			byUID[m.UID] = m
			if m.Version > next {
				next = m.Version
			}
		}
		if len(rows) < 1000 {
			break
		}
		if g.GroupType != 1 && next <= version {
			return g, fmt.Errorf("member cursor stalled")
		}
		version = next
		if page == 99 {
			return g, fmt.Errorf("group member page limit exceeded")
		}
	}
	if member, ok := byUID[actor.TargetID]; !ok || member.Deleted != 0 {
		wasMember := false
		for _, m := range old.Members {
			if m.UID == actor.TargetID && m.Huawei && m.Deleted == 0 {
				wasMember = true
			}
		}
		if !wasMember {
			return old, fail(403, "当前账号不在群内")
		}
	}
	oldMembers := map[string]domain.GroupMember{}
	for _, m := range old.Members {
		oldMembers[m.UID] = m
	}
	groupOwner := ""
	step = "import-tencent-members"
	selected := []string{}
	active := map[string]bool{speaker: true}
	active[change.NativeOperator] = true
	for _, uid := range mentioned {
		active[uid] = true
	}
	for _, m := range byUID {
		if m.UID == "" || m.UID == "fileHelper" || m.Deleted != 0 {
			continue
		}
		if hw, e := s.Store.FindIdentityByTarget(ctx, domain.SourceTang, m.UID); e == nil && hw.EntityType == "user" && hw.SourceID != hw.TargetID {
			m.TencentID = hw.SourceID
			m.Huawei = true
		}
		previous := oldMembers[m.UID]
		if m.Huawei || m.Role > 0 || len(byUID) <= 100 || active[m.UID] || previous.TencentID != "" {
			if m.TencentID == "" {
				identity := domain.Identity{Source: domain.SourceTang, SourceID: m.UID, Target: domain.SourceTencent, TargetID: m.UID, EntityType: "user", DisplayName: m.Name, Shadow: true}
				// Member names may be group-local remarks; retain the account
				// profile and its public avatar instead of replacing it with
				// this partial member record on every conversation sync.
				profile, lookupErr := s.Store.FindIdentity(ctx, domain.SourceTang, m.UID)
				if lookupErr != nil && !errors.Is(lookupErr, store.ErrNotFound) {
					return g, lookupErr
				}
				if profile != nil {
					identity.AvatarURL = profile.AvatarURL
					if profile.DisplayName != "" {
						identity.DisplayName = profile.DisplayName
					}
				}
				if identity.AvatarURL == "" && s.Media != nil {
					identity.AvatarURL, err = s.Media.AvatarURL(ctx, "users", m.UID)
					if err != nil {
						return g, err
					}
				}
				if _, err = s.Tencent.EnsureUser(ctx, identity); err != nil {
					return g, err
				}
				if err = s.Store.UpsertIdentity(ctx, identity); err != nil {
					return g, err
				}
				m.TencentID = m.UID
			}
			selected = append(selected, m.TencentID)
			if m.Role == 1 {
				groupOwner = m.TencentID
			}
		}
		g.Members = append(g.Members, m)
	}
	if groupOwner == "" {
		return g, fmt.Errorf("group owner missing from member sync")
	}
	// Fetch the initial avatar before creating the mirror. Supplying FaceUrl
	// on create avoids a synthetic administrator "changed avatar" message.
	avatar, avatarAvailable := "", false
	if s.Media != nil {
		step = "read-tang-group-avatar"
		avatar, err = s.Media.AvatarURL(ctx, "groups", number)
		if err != nil && !errors.Is(err, media.ErrGroupAvatarNotFound) {
			return g, err
		}
		avatarAvailable = err == nil
	}
	if !old.MirrorReady {
		step = "create-tencent-mirror"
		if err = s.rememberGroupAvatarInitializer(ctx, g, groupOwner); err != nil {
			return g, err
		}
		create := map[string]any{"GroupId": g.TencentID, "Type": "Public", "Owner_Account": groupOwner, "Name": truncateUTF8(g.Name, 30), "Notification": truncateUTF8(g.Notice, 300), "ApplyJoinOption": "DisableApply"}
		if avatarAvailable {
			create["FaceUrl"] = avatar
		}
		if err = s.Tencent.CreateMirrorGroup(ctx, create); err != nil {
			return g, err
		}
	}
	removed := []string{}
	step = "read-tencent-members"
	currentProfile, err := s.Tencent.GroupProfile(ctx, g.TencentID)
	if err != nil {
		return g, err
	}
	capacity, currentOwner := currentProfile.Max, currentProfile.Owner
	// Mandatory members are never evicted to make room for shadow speakers.
	// Optional shadows are selected deterministically, with current mentions
	// and speakers ahead of previously mirrored inactive members.
	if len(selected) > capacity {
		mandatory := 0
		for _, m := range g.Members {
			if m.Huawei || m.Role > 0 {
				mandatory++
			}
		}
		if mandatory > capacity {
			return g, fail(409, "腾讯镜像群容量不足，华为成员及群管理成员超过上限")
		}
		rank := func(m domain.GroupMember) int {
			if m.Huawei || m.Role > 0 {
				return 0
			}
			if active[m.UID] {
				return 1
			}
			if oldMembers[m.UID].TencentID != "" {
				return 2
			}
			return 3
		}
		sort.Slice(g.Members, func(i, j int) bool {
			a, b := rank(g.Members[i]), rank(g.Members[j])
			if a != b {
				return a < b
			}
			return g.Members[i].UID < g.Members[j].UID
		})
		selected = nil
		for i := range g.Members {
			m := &g.Members[i]
			if m.TencentID == "" {
				continue
			}
			if len(selected) < capacity {
				selected = append(selected, m.TencentID)
			} else {
				m.TencentID = ""
			}
		}
	}
	existingMembers, err := s.Tencent.GroupMemberProfiles(ctx, g.TencentID)
	if err != nil {
		return g, err
	}
	wanted := map[string]bool{}
	currentMembers := map[string]adapter.TencentGroupMember{}
	for _, uid := range selected {
		wanted[uid] = true
	}
	for _, member := range existingMembers {
		currentMembers[member.ID] = member
		if !wanted[member.ID] && member.ID != currentOwner {
			removed = append(removed, member.ID)
		}
	}
	if len(removed) > 0 {
		if err = s.Tencent.DeleteGroupMembers(ctx, g.TencentID, removed); err != nil {
			return g, err
		}
	}
	step = "add-tencent-members"
	added := []string{}
	for _, uid := range selected {
		if _, ok := currentMembers[uid]; !ok {
			added = append(added, uid)
		}
	}
	if err = s.Tencent.AddGroupMembers(ctx, g.TencentID, added); err != nil {
		return g, err
	}
	if currentOwner != groupOwner {
		if err = s.Tencent.GroupRequest(ctx, "change_group_owner", map[string]any{"GroupId": g.TencentID, "NewOwner_Account": groupOwner}, nil); err != nil {
			return g, err
		}
	}
	if currentOwner != "" && !wanted[currentOwner] {
		if err = s.Tencent.DeleteGroupMembers(ctx, g.TencentID, []string{currentOwner}); err != nil {
			return g, err
		}
	}
	mute := "Off"
	step = "update-tencent-group"
	if g.Forbidden != 0 || g.Status == 0 {
		mute = "On"
	}
	// Compare the actual projection, not a cached last attempt: retries and
	// echoed callbacks must not emit new group tips when nothing has changed.
	groupProfile := map[string]any{"GroupId": g.TencentID}
	for _, field := range []struct{ key, want, have string }{
		{"Name", truncateUTF8(g.Name, 30), currentProfile.Name},
		{"Notification", truncateUTF8(g.Notice, 300), currentProfile.Notification},
		{"MuteAllMember", mute, currentProfile.MuteAll},
	} {
		if field.want != field.have {
			groupProfile[field.key] = field.want
		}
	}
	if s.Media != nil {
		// New native groups may not have an avatar object yet. Preserve the
		// existing projection and retry on the next sync, without blocking roles,
		// mute state or group creation. Other failures must still be reported.
		if !avatarAvailable {
			avatar = currentProfile.FaceURL
		}
		if avatar != currentProfile.FaceURL {
			groupProfile["FaceUrl"] = avatar
		}
		g.Avatar = avatar
	} else {
		g.Avatar = currentProfile.FaceURL
	}
	if len(groupProfile) > 1 {
		// Native group avatars may be generated asynchronously (initial 404).
		// Attribute only that first fill to the recorded creation identity;
		// explicit authenticated avatar edits keep their own operator.
		if currentProfile.FaceURL == "" && groupProfile["FaceUrl"] != nil && !change.Avatar {
			initial, lookup := s.Store.GetCursor(ctx, "group-avatar-initializer:"+g.Number)
			if lookup != nil && !errors.Is(lookup, store.ErrNotFound) {
				return g, lookup
			}
			if lookup == nil && initial.Aux != "" {
				if err = s.projectGroupProfile(ctx, g, map[string]any{"GroupId": g.TencentID, "FaceUrl": groupProfile["FaceUrl"]}, groupProfileChange{NativeOperator: initial.Aux, Avatar: true}); err != nil {
					return g, err
				}
				delete(groupProfile, "FaceUrl")
			}
		}
		if err = s.projectGroupProfile(ctx, g, groupProfile, change); err != nil {
			return g, err
		}
	}
	for _, m := range g.Members {
		step = "update-tencent-member"
		if m.TencentID == "" || m.Role == 1 {
			continue
		}
		role := "Member"
		if m.Role == 2 {
			role = "Admin"
		}
		current, exists := currentMembers[m.TencentID]
		if !exists {
			current.Role = "Member"
		} // add_group_member creates ordinary members.
		if current.Role != role {
			if err = s.Tencent.GroupRequest(ctx, "modify_group_member_info", map[string]any{"GroupId": g.TencentID, "Member_Account": m.TencentID, "Role": role}, nil); err != nil {
				return g, err
			}
		}
		now := time.Now().Unix()
		seconds := m.MutedUntil - now
		if seconds < 0 {
			seconds = 0
		}
		if seconds > 2592000 {
			seconds = 2592000
		}
		if memberMuteChanged(now, seconds, current.MutedUntil()) {
			if err = s.Tencent.GroupRequest(ctx, "forbid_send_msg", map[string]any{"GroupId": g.TencentID, "Members_Account": []string{m.TencentID}, "ShutUpTime": seconds}, nil); err != nil {
				return g, err
			}
		}
	}
	g.MirrorReady = true
	step = "queue-role-notice"
	if err = s.queueGroupRoleNotices(ctx, actor, old, g); err != nil {
		return g, err
	}
	step = "persist-mirror"
	if err = s.Store.UpsertIdentity(ctx, domain.Identity{Source: domain.SourceTang, SourceID: g.Number, Target: domain.SourceTencent, TargetID: g.TencentID, EntityType: "group", DisplayName: g.Name, Shadow: true}); err != nil {
		return g, err
	}
	if err = s.Store.PutGroupState(ctx, g); err != nil {
		return g, err
	}
	step = "refresh-member-avatar"
	return g, s.refreshOwnMemberAvatar(ctx, actor, g)
}

func memberMuteChanged(now, seconds, currentUntil int64) bool {
	// Tencent can return the cancellation timestamp instead of zero.
	if seconds == 0 {
		return currentUntil > now
	}
	if currentUntil <= now {
		return true
	}
	delta := currentUntil - (now + seconds)
	// REST round-trip/second rounding must not repeatedly renew a mute.
	return delta < -5 || delta > 5
}

func (s *Service) ResolveGroupMessage(ctx context.Context, event domain.MessageEvent) (*domain.Identity, error) {
	if event.Source == domain.SourceTencent {
		no, err := groupNumber(event.ConversationID)
		if err != nil {
			return nil, err
		}
		g, err := s.Store.GetGroupState(ctx, no)
		if err != nil {
			return nil, err
		}
		if g.Status == 2 {
			return nil, fail(409, "群已解散")
		}
		return &domain.Identity{Source: domain.SourceTencent, SourceID: g.TencentID, Target: domain.SourceTang, TargetID: no, EntityType: "group"}, nil
	}
	actor, err := s.Store.FindIdentityByTarget(ctx, domain.SourceTang, event.ToAccount)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	g, err := s.Store.GetGroupState(ctx, event.ConversationID)
	needSync := errors.Is(err, store.ErrNotFound) || !g.MirrorReady
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}
	knownSender := false
	missingMention := map[string]bool{}
	if event.Mention != nil {
		for _, uid := range event.Mention.UIDs {
			missingMention[uid] = true
		}
	}
	for _, m := range g.Members {
		if m.TencentID != "" && m.Deleted == 0 {
			delete(missingMention, m.UID)
		}
		if m.UID == event.FromAccount && m.TencentID != "" && m.Deleted == 0 {
			knownSender = true
		}
	}
	if needSync || !knownSender || len(missingMention) > 0 {
		mentioned := []string{}
		for uid := range missingMention {
			mentioned = append(mentioned, uid)
		}
		g, err = s.syncGroup(ctx, *actor, event.ConversationID, event.FromAccount, mentioned...)
		if err != nil {
			return nil, err
		}
	}
	if g.Status == 2 {
		return nil, fail(409, "群已解散")
	}
	return &domain.Identity{Source: domain.SourceTang, SourceID: g.Number, Target: domain.SourceTencent, TargetID: g.TencentID, EntityType: "group", DisplayName: g.Name, AvatarURL: g.Avatar}, nil
}

func (s *Service) groupCommand(ctx context.Context, e domain.BusinessEvent) error {
	var cmd struct {
		Type    int                        `json:"type"`
		Content string                     `json:"content"`
		Data    map[string]json.RawMessage `json:"data"`
		Extra   []struct {
			UID  string `json:"uid"`
			Name string `json:"name"`
		} `json:"extra"`
		Visibles []string `json:"visibles"`
		Param    struct {
			GroupNo     string `json:"group_no"`
			ChannelID   string `json:"channel_id"`
			ChannelType int    `json:"channel_type"`
		} `json:"param"`
	}
	if err := json.Unmarshal(e.Payload, &cmd); err != nil {
		return err
	}
	number := cmd.Param.GroupNo
	if number == "" {
		number = e.ChannelID
	}
	if number == "" && cmd.Param.ChannelType == 2 {
		number = cmd.Param.ChannelID
	}
	if number == "" {
		return nil
	}
	actor, err := s.Store.FindIdentityByTarget(ctx, domain.SourceTang, e.Owner)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var change groupProfileChange
	// The receiving PC(2) account is not necessarily the person who changed
	// the group. Only the native event's explicit operator may be attributed.
	if cmd.Type == 1005 && len(cmd.Extra) > 0 {
		for _, field := range []string{"name", "notice", "forbidden"} {
			var value string
			if json.Unmarshal(cmd.Data[field], &value) == nil {
				c := newGroupProfileChange(cmd.Extra[0].UID, field, value)
				change.NativeOperator = c.NativeOperator
				if change.Values == nil {
					change.Values = map[string]string{}
				}
				for k, v := range c.Values {
					change.Values[k] = v
				}
			}
		}
	}
	g, err := s.syncGroupWithChange(ctx, *actor, number, "", change)
	if err != nil {
		return err
	}
	switch cmd.Type {
	case 1001, 1002, 1003, 1004, 1008, 1009, 1020, 1021, 2000:
	case 1005:
		// Native invitation confirmation has no equivalent Tencent group
		// notification: the mirror intentionally keeps joining disabled.
		// Name, notice and mute updates already emit SDK notifications; do not
		// duplicate those tips or infer settings by matching localized text.
		if len(cmd.Data["invite"]) == 0 && len(cmd.Data["status"]) == 0 {
			return nil
		}
	default:
		return nil
	}
	visible := len(cmd.Visibles) == 0 && cmd.Type != 1009
	for _, uid := range cmd.Visibles {
		if uid == actor.TargetID {
			visible = true
		}
	}
	if !visible {
		return nil
	}
	// Native content defines the meaning; numeric types alone do not prove
	// removal, especially 1004/1020 on deployments with different versions.
	text := cmd.Content
	for i, person := range cmd.Extra {
		text = strings.ReplaceAll(text, fmt.Sprintf("{%d}", i), person.Name)
	}
	if cmd.Type == 1009 {
		text = "群「" + g.Name + "」的邀请正在等待确认，成员入群后会自动同步。"
	}
	if strings.TrimSpace(text) == "" {
		return nil
	}
	messageID := e.NativeMessageID
	if messageID == "" {
		messageID = e.EventID
	}
	to, kind, scope := g.TencentID, domain.ConversationGroup, "group"
	if len(cmd.Visibles) > 0 || cmd.Type == 1009 || g.Status == 2 || cmd.Type == 1004 || cmd.Type == 1020 {
		to, kind, scope = actor.SourceID, domain.ConversationC2C, actor.SourceID
	}
	key := "group-notice:" + domain.StableID(number, messageID, scope)
	if _, lookup := s.Store.GetCursor(ctx, key); lookup == nil {
		return nil
	} else if !errors.Is(lookup, store.ErrNotFound) {
		return lookup
	}
	msg := domain.MessageEvent{Source: domain.SourceTang, SourceMessageID: key, ConversationType: kind, ConversationID: g.TencentID, Content: text, Parts: []domain.MessagePart{{Kind: "tip", Text: text}}}
	_, err = s.Tencent.SendText(ctx, msg, domain.Identity{TargetID: s.Tencent.AdminAccount}, domain.Identity{TargetID: to}, "bridge-group-notice")
	if err != nil {
		return err
	}
	return s.Store.PutCursor(ctx, domain.SyncCursor{Key: key, Value: time.Now().Unix()})
}
