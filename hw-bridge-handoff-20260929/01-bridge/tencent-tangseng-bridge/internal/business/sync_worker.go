package business

import (
	"context"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"log"
	"time"
)

// RunSync performs the business synchronization required by the handoff plan.
// It runs on the existing single instance; it owns no lease or leader role.
func (s *Service) RunSync(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		if err := s.Accounts.FlushStatus(ctx); err != nil {
			log.Printf("account status notification pending class=%T", err)
		}
		if err := s.Accounts.ReconcileRegistrations(ctx); err != nil {
			log.Printf("registration reconciliation pending class=%T", err)
		}
		s.syncActiveAccounts(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (s *Service) syncActiveAccounts(ctx context.Context) {
	identities, err := s.Store.ListIdentities(ctx)
	if err != nil {
		log.Printf("business sync list failed: %v", err)
		return
	}
	for _, id := range identities {
		if ctx.Err() != nil {
			return
		}
		if id.Source != domain.SourceTencent || id.Target != domain.SourceTang || id.EntityType != "user" {
			continue
		}
		r, e := s.Accounts.Profile(ctx, id.SourceID)
		if e != nil {
			continue
		}
		work, cancel := context.WithTimeout(ctx, 3*time.Minute)
		if r.Status == "deleting" {
			e = s.DeleteAccount(work, id.SourceID)
			cancel()
			if e != nil {
				log.Printf("account cleanup pending owner=%s error=%v", id.SourceID, e)
			}
			continue
		}
		if r.Status != "confirmed" {
			cancel()
			continue
		}
		result, e := s.SyncAccount(work, id.SourceID)
		if e == nil {
			var n int
			n, e = s.SyncTencent(work, id.SourceID)
			result.Accepted += n
		}
		if e == nil {
			e = s.syncAccountDetails(work, id)
		}
		cancel()
		if e != nil {
			log.Printf("business sync pending owner=%s error=%v", id.SourceID, e)
		} else if result.Accepted > 0 {
			log.Printf("business sync complete owner=%s channels=%d accepted=%d", id.SourceID, result.Channels, result.Accepted)
		}
	}
}
func (s *Service) syncAccountDetails(ctx context.Context, id domain.Identity) error {
	if err := s.expireFriendRequests(ctx, id.SourceID, time.Now().Unix()); err != nil {
		return err
	}
	if err := s.clearGroupUnreadDaily(ctx, id); err != nil {
		return err
	}
	if err := s.Accounts.RenewDue(ctx, id.SourceID); err != nil {
		return err
	}
	key := "profile-refresh:" + id.SourceID
	c, err := s.cursor(ctx, key, time.Now().Unix())
	if err != nil {
		return err
	}
	if time.Now().Unix()-c.Value >= 6*3600 {
		if err = s.SyncFriends(ctx, id); err != nil {
			return err
		}
		if err = s.ReconcileFriends(ctx, id); err != nil {
			return err
		}
		c.Value = time.Now().Unix()
		if err = s.Store.PutCursor(ctx, c); err != nil {
			return err
		}
	}
	// Receipt commands are transient. Refresh message extensions periodically
	// for known channels, without creating synthetic group read receipts.
	key = "extra-refresh:" + id.SourceID
	c, err = s.cursor(ctx, key, time.Now().Unix())
	if err != nil {
		return err
	}
	if time.Now().Unix()-c.Value < 10*60 {
		return nil
	}
	friends, err := s.Store.ListFriendStates(ctx, id.SourceID)
	if err != nil {
		return err
	}
	for _, f := range friends {
		if f.Status != "active" {
			continue
		}
		if _, err = s.SyncMessageExtras(ctx, id.SourceID, f.Peer, 1); err != nil {
			return err
		}
	}
	groups, err := s.Store.ListGroupStates(ctx)
	if err != nil {
		return err
	}
	for _, g := range groups {
		if g.Status == 2 {
			continue
		}
		for _, m := range g.Members {
			if m.UID == id.TargetID && m.Deleted == 0 {
				if _, err = s.SyncMessageExtras(ctx, id.SourceID, g.Number, 2); err != nil {
					return err
				}
				break
			}
		}
	}
	c.Value = time.Now().Unix()
	return s.Store.PutCursor(ctx, c)
}
