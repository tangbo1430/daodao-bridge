package account

import (
	"context"
	"errors"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
	"time"
)

func (s *Service) persistLogin(ctx context.Context, id string) error {
	if s.Store == nil {
		return nil
	}
	return s.Store.PutCursor(ctx, domain.SyncCursor{Key: "account-login:" + id, Value: time.Now().Unix()})
}

// RenewDue uses the single-instance business worker and persists only successful checks.
func (s *Service) RenewDue(ctx context.Context, id string) error {
	if s.Store == nil {
		return nil
	}
	now := time.Now().UTC()
	key := "account-renew-day:" + id
	day, err := s.Store.GetCursor(ctx, key)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return err
	}
	if day.Aux == now.Format("2006-01-02") {
		return nil
	}
	login, err := s.Store.GetCursor(ctx, "account-login:"+id)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return err
	}
	if now.Unix()-login.Value >= 7*86400 {
		r, e := s.Profile(ctx, id)
		if e != nil {
			return e
		}
		if r.Status != "confirmed" {
			return nil
		}
		if e = s.Refresh(ctx, r.Identity.TargetID); e != nil {
			return e
		}
	}
	return s.Store.PutCursor(ctx, domain.SyncCursor{Key: key, Aux: now.Format("2006-01-02"), Value: now.Unix()})
}

// A send code 19 can be a CHANNEL ban. Only connection rejection changes account state.
func (s *Service) ConnectionFailure(ctx context.Context, uid string, code int) error {
	if code != 19 && code != 25 {
		return nil
	}
	id, err := s.Store.FindIdentityByTarget(ctx, domain.SourceTang, uid)
	if err != nil {
		return err
	}
	status := "banned"
	if code == 25 {
		status = "tenant_disabled"
	}
	return s.Deactivate(ctx, id.SourceID, status)
}
