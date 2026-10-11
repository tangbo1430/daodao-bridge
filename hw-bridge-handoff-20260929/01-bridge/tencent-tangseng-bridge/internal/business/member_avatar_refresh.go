package business

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
)

func memberAvatarRefreshKey(group, uid string) string {
	return "member-avatar-refresh:" + domain.StableID(group, uid)
}

func activeAvatarMember(g domain.GroupState, uid string) bool {
	if !g.MirrorReady || g.Status == 2 {
		return false
	}
	for _, m := range g.Members {
		if m.UID == uid && m.Huawei && m.Deleted == 0 {
			return true
		}
	}
	return false
}

func (s *Service) pendingMemberAvatar(ctx context.Context, group, uid string) (string, error) {
	upload, err := s.Store.GetCursor(ctx, "avatar-upload:"+domain.StableID(uid, ""))
	if errors.Is(err, store.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if upload.Aux == "" {
		return "", nil
	}
	previous, err := s.Store.GetCursor(ctx, memberAvatarRefreshKey(group, uid))
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return "", err
	}
	if previous.Aux == upload.Aux {
		return "", nil
	}
	return upload.Aux, nil
}

// Caller has just fetched authoritative members using this user's identity.
// The native self-member API accepts an empty update, advances member version
// and emits memberUpdate. No remark, role or avatar path is overwritten.
func (s *Service) refreshOwnMemberAvatar(ctx context.Context, actor domain.Identity, g domain.GroupState) error {
	if !activeAvatarMember(g, actor.TargetID) {
		return nil
	}
	hash, err := s.pendingMemberAvatar(ctx, g.Number, actor.TargetID)
	if err != nil || hash == "" {
		return err
	}
	path := "/v1/groups/" + url.PathEscape(g.Number) + "/members/" + url.PathEscape(actor.TargetID)
	if err = s.Tang.UserRequest(ctx, actor.TargetID, http.MethodPut, path, map[string]any{}, nil); err != nil {
		return err
	}
	return s.Store.PutCursor(ctx, domain.SyncCursor{Key: memberAvatarRefreshKey(g.Number, actor.TargetID), Aux: hash, Value: time.Now().Unix()})
}

// Profile changes also refresh existing group rosters. A new group is covered
// by syncGroup; a failed request leaves its checkpoint pending for later sync.
func (s *Service) refreshProfileGroups(ctx context.Context, actor domain.Identity) error {
	groups, err := s.Store.ListGroupStates(ctx)
	if err != nil {
		return err
	}
	var failures []error
	for _, g := range groups {
		if !activeAvatarMember(g, actor.TargetID) {
			continue
		}
		hash, lookup := s.pendingMemberAvatar(ctx, g.Number, actor.TargetID)
		if lookup != nil {
			failures = append(failures, lookup)
			continue
		}
		if hash == "" {
			continue
		}
		// Recheck native membership/status before any write, rather than trust
		// a stale local roster following a kick, exit or disband.
		if _, err = s.syncGroup(ctx, actor, g.Number, ""); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}
