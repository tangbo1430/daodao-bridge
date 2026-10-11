package account

import (
	"context"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"testing"
)

func TestLegacyZeroVersionStatusProjection(t *testing.T) {
	s, _ := registrationService(t)
	ctx := context.Background()
	s.Store.PutCursor(ctx, domain.SyncCursor{Key: "account:fixture-legacy", Aux: "confirmed", StartedAt: 100})
	calls := 0
	s.NotifyStatus = func(_ context.Context, id, state string, version int64) error {
		calls++
		if version != 100 {
			t.Fatal(version)
		}
		return nil
	}
	if e := s.FlushStatus(ctx); e != nil {
		t.Fatal(e)
	}
	if e := s.FlushStatus(ctx); e != nil || calls != 1 {
		t.Fatal("legacy projection retry", e, calls)
	}
	r, _ := s.Store.GetCursor(ctx, "account:fixture-legacy")
	if r.Value != 0 {
		t.Fatal("legacy cursor rewritten")
	}
}

func TestBannedRecheckRequiresHandshakeAndStatusDeliveryRetries(t *testing.T) {
	s, _ := registrationService(t)
	ctx := context.Background()
	r, e := s.Prepare(ctx, prepareFixture())
	if e != nil {
		t.Fatal(e)
	}
	if e = s.CommitRegistration(ctx, r.ID, "fixture-tim"); e != nil {
		t.Fatal(e)
	}
	if e = s.Deactivate(ctx, "fixture-tim", "banned"); e != nil {
		t.Fatal(e)
	}
	s.VerifyConnection = func(context.Context, string) error { return fmt.Errorf("fixture still banned") }
	if _, e = s.Recheck(ctx, "fixture-tim"); e == nil {
		t.Fatal("false unban")
	}
	p, _ := s.Profile(ctx, "fixture-tim")
	if p.Status != "banned" {
		t.Fatal(p.Status)
	}
	failures := true
	calls := 0
	s.NotifyStatus = func(_ context.Context, id, state string, version int64) error {
		calls++
		if failures {
			return fmt.Errorf("fixture PHP offline")
		}
		if state != "banned" && state != "confirmed" {
			t.Fatal(state)
		}
		return nil
	}
	if e = s.FlushStatus(ctx); e == nil {
		t.Fatal("notification failure swallowed")
	}
	if _, e = s.Store.GetCursor(ctx, "account-status-ack:fixture-tim"); e == nil {
		t.Fatal("failed delivery acknowledged")
	}
	failures = false
	if e = s.FlushStatus(ctx); e != nil {
		t.Fatal(e)
	}
	s.FlushStatus(ctx)
	if calls != 2 {
		t.Fatal("duplicate status notification", calls)
	}
	starts := 0
	s.StartProxy = func(context.Context, domain.Identity) error { starts++; return nil }
	s.VerifyConnection = func(context.Context, string) error { return nil }
	if p, e = s.Recheck(ctx, "fixture-tim"); e != nil || p.Status != "confirmed" || starts != 1 {
		t.Fatal(p.Status, e, starts)
	}
	if e = s.FlushStatus(ctx); e != nil || calls != 3 {
		t.Fatal("restore notification missing", e)
	}
}
