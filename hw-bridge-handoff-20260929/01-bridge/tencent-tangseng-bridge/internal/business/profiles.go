package business

import (
	"context"
	"errors"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/account"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Friend commands contain a partial profile. In particular, friendAccept's
// from_name belongs to the accepter, which may be the Huawei owner.
func (s *Service) friendIdentity(ctx context.Context, peer, name string, previous domain.FriendState) (domain.Identity, error) {
	v := domain.Identity{Source: domain.SourceTang, SourceID: peer, Target: domain.SourceTencent, TargetID: peer, EntityType: "user", Shadow: true, DisplayName: previous.Name, AvatarURL: previous.Avatar}
	mapped, err := s.Store.FindIdentity(ctx, domain.SourceTang, peer)
	if err == nil {
		if mapped.EntityType != "user" || mapped.TargetID != peer {
			return v, fail(409, "联系人资料异常，请刷新后重试")
		}
		if mapped.DisplayName != "" {
			v.DisplayName = mapped.DisplayName
		}
		if mapped.AvatarURL != "" {
			v.AvatarURL = mapped.AvatarURL
		}
	} else if !errors.Is(err, store.ErrNotFound) {
		return v, err
	}
	if name != "" {
		v.DisplayName = name
	}
	if s.Media != nil {
		v.AvatarURL, err = s.Media.AvatarURL(ctx, "users", peer)
		if err != nil {
			return v, err
		}
	}
	return v, nil
}

// Profile commands can target a group-only shadow, which is absent from the
// owner's friend list. Fetch it under that owner's Tang identity.
func (s *Service) SyncShadowProfile(ctx context.Context, actor domain.Identity, uid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if hw, err := s.Store.FindIdentityByTarget(ctx, domain.SourceTang, uid); err == nil && hw.SourceID != hw.TargetID {
		return nil // Huawei profiles are authoritative on the PHP side.
	}
	mapped, err := s.Store.FindIdentity(ctx, domain.SourceTang, uid)
	if errors.Is(err, store.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if mapped.EntityType != "user" {
		return nil
	}
	var profile struct {
		UID  string `json:"uid"`
		Name string `json:"name"`
	}
	if err = s.Tang.UserRequest(ctx, actor.TargetID, http.MethodGet, "/v1/users/"+url.PathEscape(uid), nil, &profile); err != nil {
		return err
	}
	if profile.UID != "" && profile.UID != uid {
		return fail(502, "资料账号不匹配")
	}
	if profile.Name != "" {
		mapped.DisplayName = profile.Name
	}
	if s.Media != nil {
		mapped.AvatarURL, err = s.Media.AvatarURL(ctx, "users", uid)
		if err != nil {
			return err
		}
	}
	if _, err = s.Tencent.EnsureUser(ctx, *mapped); err != nil {
		return err
	}
	if err = s.Tencent.UpdateProfile(ctx, mapped.TargetID, mapped.DisplayName, mapped.AvatarURL); err != nil {
		return err
	}
	return s.Store.UpsertIdentity(ctx, *mapped)
}

func (s *Service) Profile(ctx context.Context, owner, name, avatar string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, err := s.owner(ctx, owner)
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	nameChanged := name != "" && name != id.DisplayName
	var avatarRefresh domain.SyncCursor
	refreshAvatar := false
	if avatar != "" && !strings.Contains(avatar, "/head_pic/head_pic1.png") {
		if s.Media == nil {
			return fail(503, "头像同步服务未配置")
		}
		if err = s.Media.UploadAvatar(ctx, id.TargetID, "", avatar); err != nil {
			return err
		}
		id.AvatarURL = avatar
		// uploadAvatar emits userAvatarUpdate, but clients also cache the
		// channel profile. Refresh that profile AFTER the bytes were uploaded.
		// Persist this separately so a failed notification is retried without
		// uploading the same image again, including an unchanged source URL.
		upload, e := s.Store.GetCursor(ctx, "avatar-upload:"+domain.StableID(id.TargetID, ""))
		if e != nil {
			return e
		}
		key := "avatar-profile-refresh:" + id.TargetID
		previous, e := s.Store.GetCursor(ctx, key)
		if e != nil && !errors.Is(e, store.ErrNotFound) {
			return e
		}
		refreshAvatar = upload.Aux != "" && upload.Aux != previous.Aux
		avatarRefresh = domain.SyncCursor{Key: key, Aux: upload.Aux, Value: time.Now().Unix()}
	}
	if nameChanged || refreshAvatar {
		currentName := id.DisplayName
		if name != "" {
			currentName = name
		}
		if currentName == "" {
			return fail(409, "资料尚未就绪，请稍后重试")
		}
		// The native owner API sends channelUpdate to friends; never forge a
		// system command or impersonate a native recipient to clear its cache.
		if err = s.Tang.UserRequest(ctx, id.TargetID, http.MethodPut, "/v1/user/current", map[string]string{"name": currentName}, nil); err != nil {
			return err
		}
		if refreshAvatar {
			if err = s.Store.PutCursor(ctx, avatarRefresh); err != nil {
				return err
			}
		}
		id.DisplayName = currentName
	}
	if err = s.Accounts.UpdateProfile(ctx, account.Request{TencentUserID: owner, DisplayName: id.DisplayName, AvatarURL: id.AvatarURL}); err != nil {
		return err
	}
	return s.refreshProfileGroups(ctx, id)
}
