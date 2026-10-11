package account

import (
	"context"
	"errors"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
	"time"
)

// Persistent account state is itself the pending notification. A failed PHP
// delivery never advances the acknowledged version or changes Tencent status.
func (s *Service) FlushStatus(ctx context.Context) error {
	if s.NotifyStatus == nil || s.Store == nil {
		return nil
	}
	rows, err := s.Store.ListCursors(ctx, "account:")
	if err != nil {
		return err
	}
	for _, r := range rows {
		switch r.Aux {
		case "confirmed", "banned", "tenant_disabled", "deleted":
		default:
			continue
		}
		id := r.Key[len("account:"):]
		version := r.Value
		// Old confirmed cursors have value=0. Give their initial projection
		// a stable version without modifying history or account mappings.
		if version <= 0 {
			version = r.StartedAt
			if version <= 0 {
				version = 1
			}
		}
		ack, e := s.Store.GetCursor(ctx, "account-status-ack:"+id)
		if e != nil && !errors.Is(e, store.ErrNotFound) {
			return e
		}
		if ack.Value >= version && ack.Aux == r.Aux {
			continue
		}
		if e = s.NotifyStatus(ctx, id, r.Aux, version); e != nil {
			return e
		}
		if e = s.Store.PutCursor(ctx, domain.SyncCursor{Key: "account-status-ack:" + id, Value: version, Aux: r.Aux}); e != nil {
			return e
		}
	}
	return nil
}

// Recheck is an explicit authenticated operation. No automatic retry is used
// to bypass native bans; both native login and a PC(2) handshake must succeed.
func (s *Service) Recheck(ctx context.Context, id string) (Record, error) {
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()
	r, err := s.Profile(ctx, id)
	if err != nil {
		return r, err
	}
	if r.Status != "banned" && r.Status != "tenant_disabled" {
		return r, fmt.Errorf("account does not require native status verification")
	}
	if s.Tang == nil || s.VerifyConnection == nil {
		return r, fmt.Errorf("native status verifier unavailable")
	}
	_, err = s.Tang.Reauthenticate(ctx, r.Identity)
	if err != nil {
		return r, err
	}
	if err = s.VerifyConnection(ctx, r.Identity.TargetID); err != nil {
		s.Tang.UserTokens.Delete(r.Identity.TargetID)
		if s.StopProxy != nil {
			_ = s.StopProxy(ctx, r.Identity)
		}
		return r, err
	}
	if err = s.persistLogin(ctx, id); err != nil {
		return r, err
	}
	// Compare while locked so a concurrent ban/delete cannot be overwritten.
	s.mu.Lock()
	current := s.records[id]
	if current.Status != r.Status {
		s.mu.Unlock()
		return r, fmt.Errorf("account state changed during verification")
	}
	state, e := s.Store.GetCursor(ctx, "account:"+id)
	if e == nil && state.Aux == r.Status {
		state.Aux = "confirmed"
		state.Value = time.Now().UnixNano()
		e = s.Store.PutCursor(ctx, state)
	} else if e == nil {
		e = fmt.Errorf("native account state changed")
	}
	if e != nil {
		s.mu.Unlock()
		return r, e
	}
	r.Status = "confirmed"
	r.LastLoginAt = time.Now().Unix()
	s.records[id] = r
	s.mu.Unlock()
	if s.StartProxy != nil {
		err = s.StartProxy(ctx, r.Identity)
	}
	return r, err
}
