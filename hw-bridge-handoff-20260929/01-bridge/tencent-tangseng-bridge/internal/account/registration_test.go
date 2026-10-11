package account

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/adapter"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type registrationFixture struct {
	Registered   int
	Logins       int
	Password     string
	Invite       string
	LoseResponse bool
	Reject       bool
}

func TestRegistrationRejectedRetryReconciliationAndDeletionErase(t *testing.T) {
	s, f := registrationService(t)
	ctx := context.Background()
	q := prepareFixture()
	f.Reject = true
	if _, e := s.Prepare(ctx, q); e == nil {
		t.Fatal("refusal accepted")
	}
	f.Reject = false
	q.RequestID = "fixture-request-new"
	r, e := s.Prepare(ctx, q)
	if e != nil || f.Registered != 2 {
		t.Fatal("definite refusal locked username", e)
	}
	r.CreatedAt = time.Now().Unix() - 900
	r.UpdatedAt = r.CreatedAt
	b, _ := json.Marshal(r)
	s.Store.PutCursor(ctx, domain.SyncCursor{Key: "registration:" + r.ID, Aux: string(b)})
	lookups := 0
	s.RegistrationLookup = func(context.Context, string, string) (string, error) { lookups++; return "fixture-tim", nil }
	if e = s.ReconcileRegistrations(ctx); e != nil {
		t.Fatal(e)
	}
	if e = s.ReconcileRegistrations(ctx); e != nil || lookups != 1 {
		t.Fatal("scan interval/idempotency", e, lookups)
	}
	if e = s.Deactivate(ctx, "fixture-tim", "deleted"); e != nil {
		t.Fatal(e)
	}
	r, e = s.reservation(ctx, r.ID)
	if e != nil || r.Cipher != "" || r.Status != "deleted" {
		t.Fatal("credential retained", e)
	}
	if _, found, e := s.RegisteredCredential(ctx, "fixture-tim"); !found || e == nil {
		t.Fatal("deleted account fell back to legacy")
	}
}

func registrationService(t *testing.T) (*Service, *registrationFixture) {
	t.Helper()
	f := &registrationFixture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		json.NewDecoder(r.Body).Decode(&b)
		switch r.URL.Path {
		case "/register":
			f.Registered++
			f.Password, _ = b["password"].(string)
			f.Invite, _ = b["invite_code"].(string)
			if b["username"] != "fixture_user" || b["manufacturer"] != "huawei" || b["device"] != nil || len(f.Password) != 24 {
				t.Error("registration contract", b["username"])
			}
			if f.Reject {
				http.Error(w, `{"msg":"该用户名已存在"}`, 400)
				return
			}
			if f.LoseResponse {
				w.WriteHeader(504)
				return
			}
			fmt.Fprint(w, `{"data":{"uid":"native-registration-fixture","app_id":"app-fixture","token":"registration-token-not-used"}}`)
		case "/login":
			f.Logins++
			if b["password"] != f.Password {
				t.Error("stored password not reused")
			}
			fmt.Fprint(w, `{"uid":"native-registration-fixture","token":"login-fixture-token","app_id":"app-fixture"}`)
		case "/v1/user/current":
			fmt.Fprint(w, `{}`)
		default:
			t.Error(r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(server.Close)
	c := &adapter.TangSengClient{BaseURL: server.URL, UserEndpoint: "/register", LoginEndpoint: "/login", UserTokens: &sync.Map{}}
	s := New(c)
	s.SetStore(store.NewMemoryStore())
	s.ConfigureRegistration("fixture-encryption-key")
	c.Credential = s.RegisteredCredential
	return s, f
}
func prepareFixture() PrepareRequest {
	return PrepareRequest{RequestID: "fixture-request-1", Username: "fixture_user", Nick: "Fixture", Invite: "fixture-invite", ClientIP: "192.0.2.1"}
}

func TestRegistrationUsesServerInviteWithoutExposingItToClient(t *testing.T) {
	s, f := registrationService(t)
	q := prepareFixture()
	q.Invite = ""
	if _, err := s.Prepare(context.Background(), q); err == nil || f.Registered != 0 {
		t.Fatal("missing channel configuration did not fail closed", err)
	}
	s.Tang.InviteCode = "server-only-fixture"
	r, err := s.Prepare(context.Background(), q)
	if err != nil || f.Invite != s.Tang.InviteCode || f.Registered != 1 {
		t.Fatal("server enrollment failed", err)
	}
	s.Tang.InviteCode = "rotated-server-fixture"
	again, err := s.Prepare(context.Background(), q)
	if err != nil || again.ID != r.ID || f.Registered != 1 {
		t.Fatal("configuration rotation changed request identity", err)
	}
	b, _ := json.Marshal(r)
	if strings.Contains(string(b), "server-only-fixture") {
		t.Fatal("channel credential persisted in public reservation")
	}
}

func TestRegistrationPrepareCommitIdempotencyAndEncryptedRecovery(t *testing.T) {
	s, f := registrationService(t)
	ctx := context.Background()
	q := prepareFixture()
	r, err := s.Prepare(ctx, q)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Prepare(ctx, q)
	if err != nil || second.ID != r.ID || f.Registered != 1 || f.Logins != 1 {
		t.Fatal("prepare repeated native mutation", err)
	}
	c, _ := s.Store.GetCursor(ctx, "registration:"+r.ID)
	if strings.Contains(c.Aux, f.Password) || strings.Contains(c.Aux, "login-fixture-token") {
		t.Fatal("plaintext credential persisted")
	}
	if err = s.CommitRegistration(ctx, r.ID, "fixture-tim"); err != nil {
		t.Fatal(err)
	}
	if err = s.CommitRegistration(ctx, r.ID, "fixture-tim"); err != nil {
		t.Fatal(err)
	}
	if err = s.CommitRegistration(ctx, r.ID, "other-tim"); err == nil {
		t.Fatal("mapping reassigned")
	}
	if err = s.AbortRegistration(ctx, r.ID); err == nil {
		t.Fatal("committed account aborted")
	}
	restarted := New(s.Tang)
	restarted.SetStore(s.Store)
	restarted.ConfigureRegistration("fixture-encryption-key")
	s.Tang.Credential = restarted.RegisteredCredential
	if err = restarted.Restore(ctx); err != nil {
		t.Fatal(err)
	}
	if f.Registered != 1 {
		t.Fatal("restart used legacy registration")
	}
	id, _ := s.Store.FindIdentity(ctx, domain.SourceTencent, "fixture-tim")
	if id.TargetID != r.UID {
		t.Fatal("identity changed")
	}
}

func TestRegistrationLostResponseUsesSavedIdentityAndAbortReuse(t *testing.T) {
	s, f := registrationService(t)
	ctx := context.Background()
	q := prepareFixture()
	f.LoseResponse = true
	if _, err := s.Prepare(ctx, q); err == nil {
		t.Fatal("timeout claimed success")
	}
	r, err := s.Prepare(ctx, q)
	if err != nil || f.Registered != 1 || r.UID == "" {
		t.Fatal("lost response recovery", r, err)
	}
	if err = s.AbortRegistration(ctx, r.ID); err != nil {
		t.Fatal(err)
	}
	q.RequestID = "fixture-request-2"
	next, err := s.Prepare(ctx, q)
	if err != nil || next.ID == r.ID || next.UID != r.UID || f.Registered != 1 {
		t.Fatal("abort credential reuse", err)
	}
	if err = s.AbortRegistration(ctx, r.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.CommitRegistration(ctx, next.ID, "fixture-tim"); err != nil {
		t.Fatal("old abort cancelled new generation", err)
	}
}

func TestRegistrationNoTakeoverAndMissingEncryption(t *testing.T) {
	s, f := registrationService(t)
	ctx := context.Background()
	q := prepareFixture()
	f.Reject = true
	if _, err := s.Prepare(ctx, q); err == nil || f.Logins != 0 {
		t.Fatal("existing native account reused")
	}
	s.RegistrationBox = nil
	q.RequestID = "fixture-request-2"
	if _, err := s.Prepare(ctx, q); err == nil || f.Registered != 1 {
		t.Fatal("missing encryption accepted")
	}
}
