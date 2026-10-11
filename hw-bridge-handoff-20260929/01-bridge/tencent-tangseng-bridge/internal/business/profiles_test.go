package business

import (
	"context"
	"encoding/json"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/engine"
	"testing"
)

func TestProfileRunsAsOwnerAndPersistsAfterSuccess(t *testing.T) {
	s, calls, owner, _ := fixture(t)
	ctx := context.Background()
	payload, _ := json.Marshal(map[string]string{"tencent_user_id": owner, "display_name": "Fixture Updated"})
	e := &engine.Engine{Store: s.Store, BusinessHandler: s.Handle}
	_, err := e.Accept(ctx, domain.EventEnvelope{Source: domain.SourceTencent, Command: "Profile.Sync", EventID: "profile-fixture", IdempotencyKey: "profile-fixture", Payload: payload})
	if err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 0 {
		t.Fatal("profile mutation before durable queue")
	}
	if _, err = e.ProcessDue(ctx, 10); err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 1 || (*calls)[0].Path != "/v1/user/current" || (*calls)[0].Token != "owner-fixture-token" || (*calls)[0].Body["name"] != "Fixture Updated" {
		t.Fatalf("calls %+v", *calls)
	}
	r, _ := s.Accounts.Profile(ctx, owner)
	if r.Identity.DisplayName != "Fixture Updated" {
		t.Fatal("cache not updated")
	}
	saved, _ := s.Store.FindIdentity(ctx, domain.SourceTencent, owner)
	if saved.DisplayName != "Fixture Updated" {
		t.Fatal("profile not persisted")
	}
	if err = s.Profile(ctx, owner, "Fixture Updated", ""); err != nil || len(*calls) != 1 {
		t.Fatal("duplicate profile mutation")
	}
}
