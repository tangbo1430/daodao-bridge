package business

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/account"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
	"net/http"
	"net/url"
	"strconv"
)

type deletionProgress struct {
	Groups       []string        `json:"groups"`
	GroupExits   map[string]bool `json:"group_exits"`
	GroupMirrors map[string]bool `json:"group_mirrors"`
	FriendsDone  bool            `json:"friends_done"`
	ProfileDone  bool            `json:"profile_done"`
}

// DeleteAccount uses the user's token throughout. Each completed native step is
// persisted so the existing Outbox can resume after a restart or mirror failure.
func (s *Service) DeleteAccount(ctx context.Context, owner string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.Store.GetCursor(ctx, "account:"+owner)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return err
	}
	if state.Aux == "deleted" {
		return s.Accounts.EraseRegistrationCredential(ctx, owner)
	}
	r, err := s.Accounts.BeginDeletion(ctx, owner)
	if err != nil { // A native Tencent account may never have enabled the bridge.
		_, lookup := s.Store.FindIdentity(ctx, domain.SourceTencent, owner)
		if errors.Is(lookup, store.ErrNotFound) {
			return nil
		}
		return err
	}
	key := "account-delete:" + owner
	checkpoint, e := s.Store.GetCursor(ctx, key)
	p := deletionProgress{}
	if e != nil && !errors.Is(e, store.ErrNotFound) {
		return e
	}
	if checkpoint.Aux != "" {
		if err = json.Unmarshal([]byte(checkpoint.Aux), &p); err != nil {
			return err
		}
	}
	persist := func() error {
		b, e := json.Marshal(p)
		if e != nil {
			return e
		}
		checkpoint.Key = key
		checkpoint.Aux = string(b)
		return s.Store.PutCursor(ctx, checkpoint)
	}
	if p.GroupExits == nil {
		p.GroupExits = map[string]bool{}
		p.GroupMirrors = map[string]bool{}
		seen := map[string]bool{}
		groups, e := s.Store.ListGroupStates(ctx)
		if e != nil {
			return e
		}
		for _, g := range groups {
			for _, m := range g.Members {
				if m.UID == r.Identity.TargetID {
					seen[g.Number] = true
				}
			}
		}
		// Recent conversations supplement the mapping for groups joined while
		// the bridge was offline; never acknowledge messages during deletion.
		convs, e := s.Tang.SyncConversations(ctx, r.Identity.TargetID, 0, "", "hwbridge-delete-"+owner)
		if e != nil {
			return e
		}
		for _, c := range convs.Conversations {
			if c.Type == 2 {
				seen[c.Channel] = true
			}
		}
		for group := range seen {
			p.Groups = append(p.Groups, group)
		}
		if err = persist(); err != nil {
			return err
		}
	}
	for _, group := range p.Groups {
		if !p.GroupExits[group] {
			// The detail endpoint reports membership, allowing a retry after an
			// exit succeeded but its checkpoint could not be written.
			var detail struct {
				Quit   int `json:"quit"`
				Status int `json:"status"`
			}
			path := "/v1/groups/" + url.PathEscape(group)
			if err = s.Tang.UserRequest(ctx, r.Identity.TargetID, http.MethodGet, path, nil, &detail); err != nil {
				return err
			}
			if detail.Quit == 0 && detail.Status != 2 {
				if err = s.Tang.UserRequest(ctx, r.Identity.TargetID, http.MethodPost, path+"/exit", nil, nil); err != nil {
					return err
				}
			}
			p.GroupExits[group] = true
			if err = persist(); err != nil {
				return err
			}
		}
		if !p.GroupMirrors[group] {
			g, e := s.Store.GetGroupState(ctx, group)
			if e != nil && !errors.Is(e, store.ErrNotFound) {
				return e
			}
			if e == nil && g.Status != 2 && g.MirrorReady {
				wasOwner := false
				for _, m := range g.Members {
					if m.UID == r.Identity.TargetID && m.Role == 1 {
						wasOwner = true
					}
				}
				if wasOwner {
					// Native exit may elect a new owner or dissolve the group. Read
					// the authoritative result and transfer BEFORE removing the old owner.
					if _, e = s.syncGroup(ctx, r.Identity, group, ""); e != nil {
						return e
					}
					p.GroupMirrors[group] = true
					if e = persist(); e != nil {
						return e
					}
					continue
				}
				members, e := s.Tencent.GroupMembers(ctx, g.TencentID)
				if e != nil {
					return e
				}
				for _, uid := range members {
					if uid == owner {
						if e = s.Tencent.DeleteGroupMembers(ctx, g.TencentID, []string{owner}); e != nil {
							return e
						}
					}
				}
				kept := g.Members[:0]
				for _, m := range g.Members {
					if m.UID != r.Identity.TargetID {
						kept = append(kept, m)
					}
				}
				g.Members = kept
				if e = s.Store.PutGroupState(ctx, g); e != nil {
					return e
				}
			}
			p.GroupMirrors[group] = true
			if err = persist(); err != nil {
				return err
			}
		}
	}
	if !p.FriendsDone {
		version := int64(0)
		for page := 0; page < 100; page++ {
			var friends []struct {
				UID     string `json:"uid"`
				Version int64  `json:"version"`
				Deleted int    `json:"is_deleted"`
			}
			if err = s.Tang.UserRequest(ctx, r.Identity.TargetID, http.MethodGet, "/v1/friend/sync?api_version=1&limit=200&version="+strconv.FormatInt(version, 10), nil, &friends); err != nil {
				return err
			}
			next := version
			for _, f := range friends {
				if f.Version > next {
					next = f.Version
				}
				if f.UID != "" && f.UID != "fileHelper" && f.Deleted == 0 {
					if err = s.Tang.FriendDelete(ctx, r.Identity.TargetID, f.UID); err != nil {
						return err
					}
				}
			}
			if len(friends) < 200 {
				p.FriendsDone = true
				break
			}
			if next <= version {
				return fmt.Errorf("delete friend cursor stalled")
			}
			version = next
		}
		if !p.FriendsDone {
			return fmt.Errorf("delete friend page limit exceeded")
		}
		if err = persist(); err != nil {
			return err
		}
	}
	if !p.ProfileDone {
		if err = s.Tang.UserRequest(ctx, r.Identity.TargetID, http.MethodPut, "/v1/user/current", map[string]any{"name": "已注销用户"}, nil); err != nil {
			return err
		}
		if err = s.Accounts.UpdateProfile(ctx, account.Request{TencentUserID: owner, DisplayName: "已注销用户"}); err != nil {
			return err
		}
		p.ProfileDone = true
		if err = persist(); err != nil {
			return err
		}
	}
	friends, err := s.Store.ListFriendStates(ctx, owner)
	if err != nil {
		return err
	}
	for _, f := range friends {
		f.Status = "deleted"
		f.TokenCipher = ""
		f.VercodeCipher = ""
		if err = s.Store.PutFriendState(ctx, f); err != nil {
			return err
		}
	}
	return s.Accounts.Deactivate(ctx, owner, "deleted")
}
