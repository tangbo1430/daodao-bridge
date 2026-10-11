package account

import (
	"context"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/adapter"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestConcurrentExpiredTokenRefreshAndDailyCheckpoint(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		fmt.Fprint(w, `{"uid":"native-fixture","token":"new-fixture-token"}`)
	}))
	defer server.Close()
	ctx := context.Background()
	st := store.NewMemoryStore()
	client := &adapter.TangSengClient{BaseURL: server.URL, UserTokens: &sync.Map{}, ShadowPassword: "fixture-password"}
	s := New(client)
	s.SetStore(st)
	id := domain.Identity{Source: domain.SourceTencent, SourceID: "fixture-owner", Target: domain.SourceTang, TargetID: "native-fixture", EntityType: "user"}
	st.UpsertIdentity(ctx, id)
	s.records[id.SourceID] = Record{Identity: id, Status: "confirmed"}
	client.UserTokens.Store(id.TargetID, "old-fixture-token")
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if e := s.RefreshIfToken(ctx, id.TargetID, "old-fixture-token"); e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("parallel refresh count=%d", calls.Load())
	}
	login, e := st.GetCursor(ctx, "account-login:"+id.SourceID)
	if e != nil || login.Value < time.Now().Unix()-10 {
		t.Fatal(login, e)
	}
	if e = s.RenewDue(ctx, id.SourceID); e != nil {
		t.Fatal(e)
	}
	if calls.Load() != 1 {
		t.Fatal("recent login was renewed")
	}
	st.PutCursor(ctx, domain.SyncCursor{Key: "account-renew-day:" + id.SourceID})
	st.PutCursor(ctx, domain.SyncCursor{Key: "account-login:" + id.SourceID, Value: time.Now().Add(-8 * 24 * time.Hour).Unix()})
	if e = s.RenewDue(ctx, id.SourceID); e != nil {
		t.Fatal(e)
	}
	if calls.Load() != 2 {
		t.Fatal("overdue login not renewed")
	}
	if e = s.RenewDue(ctx, id.SourceID); e != nil || calls.Load() != 2 {
		t.Fatal("daily checkpoint ignored", e)
	}
}

func TestBannedAccountCannotBeProvisionedOrRestored(t *testing.T) {
	ctx := context.Background()
	st := store.NewMemoryStore()
	s := New(nil)
	s.SetStore(st)
	r, err := s.Reserve(ctx, Request{TencentUserID: "ban-fixture"})
	if err != nil {
		t.Fatal(err)
	}
	s.Confirm(ctx, r.Identity.SourceID)
	if err = s.ConnectionFailure(ctx, r.Identity.TargetID, 19); err != nil {
		t.Fatal(err)
	}
	s = New(nil)
	s.SetStore(st)
	s.StartProxy = func(context.Context, domain.Identity) error { t.Fatal("banned proxy started"); return nil }
	if err = s.Restore(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Reserve(ctx, Request{TencentUserID: r.Identity.SourceID}); err == nil {
		t.Fatal("ban bypass")
	}
	if _, err = s.Confirm(ctx, r.Identity.SourceID); err == nil {
		t.Fatal("confirm bypass")
	}
	p, err := s.Profile(ctx, r.Identity.SourceID)
	if err != nil || p.Status != "banned" {
		t.Fatal(p, err)
	}
}
