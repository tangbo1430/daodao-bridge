package business

import (
	"context"
	"errors"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
	"net/http"
	"net/url"
	"time"
)

func (s *Service) expireFriendRequests(ctx context.Context, owner string, now int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.Store.ListFriendStates(ctx, owner)
	if err != nil {
		return err
	}
	for _, f := range rows {
		expires := f.ExpiresAt
		if f.Status == "pending_out" && expires == 0 && f.UpdatedAt > 0 {
			expires = f.UpdatedAt + 30*86400
		}
		if expires == 0 || expires > now {
			continue
		}
		if f.Status != "pending_in" && f.Status != "pending_out" && f.Status != "notifying" {
			continue
		}
		// Keep incoming expiry recognizable so clicking accept can reapply.
		if f.Status == "pending_in" || f.Status == "notifying" {
			f.Status = "expired_in"
		} else {
			f.Status = "expired_out"
		}
		f.TokenCipher, f.VercodeCipher = "", ""
		f.ExpiresAt = expires
		if err = s.Store.PutFriendState(ctx, f); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) clearGroupUnreadDaily(ctx context.Context, id domain.Identity) error {
	date := time.Now().UTC().Format("2006-01-02")
	groups, err := s.Store.ListGroupStates(ctx)
	if err != nil {
		return err
	}
	for _, g := range groups {
		if !g.MirrorReady || g.Status == 2 {
			continue
		}
		member := false
		for _, m := range g.Members {
			if m.UID == id.TargetID && m.Deleted == 0 {
				member = true
				break
			}
		}
		if !member {
			continue
		}
		key := "group-unread-day:" + id.SourceID + ":" + g.Number
		c, e := s.Store.GetCursor(ctx, key)
		if e != nil && !errors.Is(e, store.ErrNotFound) {
			return e
		}
		if c.Aux == date {
			continue
		}
		// Recheck native membership; stale local mirrors cannot authorize clearing.
		var detail struct {
			Quit   int    `json:"quit"`
			Status int    `json:"status"`
			Number string `json:"group_no"`
		}
		if e = s.Tang.UserRequest(ctx, id.TargetID, http.MethodGet, "/v1/groups/"+url.PathEscape(g.Number), nil, &detail); e != nil {
			return e
		}
		if detail.Number != g.Number || detail.Quit != 0 || detail.Status == 2 {
			continue
		}
		if e = s.Tang.ClearUnread(ctx, id.TargetID, g.Number, 2); e != nil {
			return e
		}
		if e = s.Store.PutCursor(ctx, domain.SyncCursor{Key: key, Aux: date, Value: time.Now().Unix()}); e != nil {
			return e
		}
	}
	return nil
}
