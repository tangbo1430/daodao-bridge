package account

import (
	"context"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
	"testing"
)

func TestDisabledAndDeletedAccountsDoNotRestartProxy(t *testing.T) {
	ctx := context.Background()
	st := store.NewMemoryStore()
	s := New(nil)
	s.SetStore(st)
	for _, uid := range []string{"disabled-fixture", "deleted-fixture"} {
		if _, err := s.Reserve(ctx, Request{TencentUserID: uid}); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Confirm(ctx, uid); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Cancel(ctx, "disabled-fixture"); err != nil {
		t.Fatal(err)
	}
	if err := s.Deactivate(ctx, "deleted-fixture", "deleted"); err != nil {
		t.Fatal(err)
	}
	restarted := New(nil)
	restarted.SetStore(st)
	if err := restarted.Restore(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.Profile(ctx, "disabled-fixture"); err == nil {
		t.Fatal("disabled account restored")
	}
	if _, err := restarted.Reserve(ctx, Request{TencentUserID: "deleted-fixture"}); err == nil {
		t.Fatal("deleted account resurrected")
	}
	if _, err := restarted.Reserve(ctx, Request{TencentUserID: "disabled-fixture"}); err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.Confirm(ctx, "disabled-fixture"); err != nil {
		t.Fatal(err)
	}
	c, err := st.GetCursor(ctx, "account:disabled-fixture")
	if err != nil || c.Aux != "confirmed" || c.StartedAt == 0 {
		t.Fatalf("cursor %+v err %v", c, err)
	}
}

func TestCancelledAccountCanBeCleanedAfterRestartWithoutProxy(t *testing.T) {
	ctx := context.Background()
	st := store.NewMemoryStore()
	s := New(nil)
	s.SetStore(st)
	_, _ = s.Reserve(ctx, Request{TencentUserID: "cancelled-cleanup-fixture"})
	_ = s.Cancel(ctx, "cancelled-cleanup-fixture")
	s = New(nil)
	s.SetStore(st)
	if err := s.Restore(ctx); err != nil {
		t.Fatal(err)
	}
	s.StartProxy = func(context.Context, domain.Identity) error { t.Fatal("cleanup started proxy"); return nil }
	r, err := s.BeginDeletion(ctx, "cancelled-cleanup-fixture")
	if err != nil || r.Status != "deleting" {
		t.Fatalf("record=%+v err=%v", r, err)
	}
	c, _ := st.GetCursor(ctx, "account:cancelled-cleanup-fixture")
	if c.Aux != "deleting" {
		t.Fatalf("state=%+v", c)
	}
}
