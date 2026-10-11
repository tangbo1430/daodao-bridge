package business

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/account"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/adapter"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/engine"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type call struct {
	Path, Method, Token string
	Body                map[string]any
}

func TestFileHelperCannotBecomeContactOrGroupMember(t *testing.T) {
	s, calls, owner, _ := fixture(t)
	ctx := context.Background()
	before := len(*calls)
	if err := s.SaveSearch(ctx, owner, adapter.SearchUser{UID: "fileHelper"}); err == nil {
		t.Fatal("file helper saved as a contact")
	}
	if _, err := s.Friend(ctx, owner, "fileHelper", "apply", "", ""); err == nil {
		t.Fatal("file helper accepted as a friend")
	}
	if _, err := s.tangUser(ctx, "fileHelper"); err == nil {
		t.Fatal("file helper accepted as a group member")
	}
	if len(*calls) != before {
		t.Fatal("excluded system account caused upstream writes")
	}
}

func fixture(t *testing.T) (*Service, *[]call, string, string) {
	t.Helper()
	calls := []call{}
	ctx := context.Background()
	st := store.NewMemoryStore()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		calls = append(calls, call{r.URL.Path, r.Method, r.Header.Get("token"), body})
		w.Header().Set("Content-Type", "application/json")
		if strings.HasPrefix(r.URL.Path, "/v4/") {
			fmt.Fprint(w, `{"ActionStatus":"OK","ErrorCode":0,"ResultItem":[{"ResultCode":0}]}`)
			return
		}
		if r.URL.Path == "/v1/user/search" {
			fmt.Fprint(w, `[{"uid":"peer-fixture","name":"Fixture","vercode":"search-proof"}]`)
			return
		}
		fmt.Fprint(w, `{}`)
	}))
	t.Cleanup(upstream.Close)
	accounts := account.New(nil)
	accounts.SetStore(st)
	record, err := accounts.Reserve(ctx, account.Request{TencentUserID: "owner-fixture"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = accounts.Confirm(ctx, record.Identity.SourceID); err != nil {
		t.Fatal(err)
	}
	peer := "peer-fixture"
	if err = st.UpsertIdentity(ctx, domain.Identity{Source: domain.SourceTang, SourceID: peer, Target: domain.SourceTencent, TargetID: peer, EntityType: "user", Shadow: true}); err != nil {
		t.Fatal(err)
	}
	tang := &adapter.TangSengClient{BaseURL: upstream.URL, Token: "must-not-use-admin", UserTokens: &sync.Map{}}
	tang.UserTokens.Store(record.Identity.TargetID, "owner-fixture-token")
	tim := &adapter.TencentClient{BaseURL: upstream.URL, SDKAppID: 1, SecretKey: "fixture-only", AdminAccount: "fixture-admin"}
	return New(st, accounts, tang, tim, "fixture-key"), &calls, record.Identity.SourceID, peer
}

func TestFriendLifecycleAndIdempotentMirrorCallbacks(t *testing.T) {
	s, calls, owner, peer := fixture(t)
	ctx := context.Background()
	token, _ := s.Box.Encrypt("request-fixture-token")
	if err := s.Store.PutFriendState(ctx, domain.FriendState{Owner: owner, Peer: peer, Status: "pending_in", TokenCipher: token, ExpiresAt: time.Now().Add(time.Hour).Unix()}); err != nil {
		t.Fatal(err)
	}
	rows, err := s.Requests(ctx, owner)
	if err != nil || len(rows) != 1 {
		t.Fatalf("requests=%v err=%v", rows, err)
	}
	raw, _ := json.Marshal(rows)
	if strings.Contains(string(raw), "token") {
		t.Fatal("credentials exposed to APK")
	}
	for _, action := range []string{"accept", "accept", "block", "block", "unblock", "delete", "delete"} {
		if _, err = s.Friend(ctx, owner, peer, action, "", ""); err != nil {
			t.Fatalf("%s: %v", action, err)
		}
	}
	want := []string{"/v1/friend/sure", "/v4/sns/friend_add", "/v1/user/blacklist/peer-fixture", "/v1/user/blacklist/peer-fixture", "/v1/friends/peer-fixture", "/v4/sns/friend_delete"}
	if len(*calls) != len(want) {
		t.Fatalf("calls=%+v", *calls)
	}
	for i, c := range *calls {
		if c.Path != want[i] {
			t.Fatalf("call %d: %+v", i, c)
		}
		if strings.HasPrefix(c.Path, "/v1/") && c.Token != "owner-fixture-token" {
			t.Fatal("wrong actor token")
		}
	}
	body := (*calls)[1].Body
	if body["ForceAddFlags"] != float64(1) || body["AddType"] != "Add_Type_Both" {
		t.Fatalf("friend_add contract %v", body)
	}
}
func TestExpiredRequestSendsNewApplicationWithoutAcceptingMirror(t *testing.T) {
	s, calls, owner, peer := fixture(t)
	ctx := context.Background()
	cipher, _ := s.Box.Encrypt("expired")
	_ = s.Store.PutFriendState(ctx, domain.FriendState{Owner: owner, Peer: peer, Status: "pending_in", TokenCipher: cipher, ExpiresAt: time.Now().Add(-time.Hour).Unix()})
	v, err := s.Friend(ctx, owner, peer, "accept", "", "")
	if err != nil || v.Status != "pending_out" {
		t.Fatalf("state=%s err=%v", v.Status, err)
	}
	for _, c := range *calls {
		if c.Path == "/v1/friend/sure" || c.Path == "/v4/sns/friend_add" {
			t.Fatalf("expired application incorrectly accepted: %+v", c)
		}
	}
}
func TestMutationsRejectMissingActorToken(t *testing.T) {
	s, calls, owner, peer := fixture(t)
	ctx := context.Background()
	record, _ := s.Accounts.Profile(ctx, owner)
	s.Tang.UserTokens.Delete(record.Identity.TargetID)
	if _, err := s.Friend(ctx, owner, peer, "apply", "hello", "proof"); err == nil {
		t.Fatal("admin fallback allowed")
	}
	if len(*calls) != 0 {
		t.Fatalf("remote write made without actor: %+v", *calls)
	}
}
func TestTencentNormalFriendCallbackDoesNotReachTang(t *testing.T) {
	s, calls, _, _ := fixture(t)
	err := s.Handle(context.Background(), domain.BusinessEvent{Source: domain.SourceTencent, Command: "Sns.CallbackFriendDelete", Payload: json.RawMessage(`{"PairList":[{"From_Account":"native-a","To_Account":"native-b"}]}`)})
	if err != nil || len(*calls) != 0 {
		t.Fatalf("native friendship changed: %v %+v", err, *calls)
	}
}
func TestPCCommandUsesParamAndDurableOutbox(t *testing.T) {
	s, calls, owner, peer := fixture(t)
	ctx := context.Background()
	r, _ := s.Accounts.Profile(ctx, owner)
	payload, _ := json.Marshal(map[string]any{"to_uid": r.Identity.TargetID, "from_uid": "", "channel_type": 8, "payload": map[string]any{"type": 99, "cmd": "friendAccept", "param": map[string]any{"from_uid": peer, "to_uid": r.Identity.TargetID, "from_name": "Fixture"}}})
	e := &engine.Engine{Store: s.Store, BusinessHandler: s.Handle}
	envelope := domain.EventEnvelope{Source: domain.SourceTang, Command: "msg.notify", EventID: "fixture-accept", IdempotencyKey: "fixture-accept", Payload: payload}
	accepted, err := e.Accept(ctx, envelope)
	if err != nil || !accepted {
		t.Fatal(err)
	}
	if len(*calls) != 0 {
		t.Fatal("HTTP side effects before outbox processing")
	}
	if _, err = e.ProcessDue(ctx, 10); err != nil {
		t.Fatal(err)
	}
	v, _ := s.Store.GetFriendState(ctx, owner, peer)
	if v.Status != "active" {
		t.Fatalf("status %s", v.Status)
	}
	count := len(*calls)
	_, _ = e.Accept(ctx, envelope)
	_, _ = e.ProcessDue(ctx, 10)
	if len(*calls) != count {
		t.Fatal("redelivery duplicated operations")
	}
}
func TestRefusalAndSearchDoNotReactivateOldRequest(t *testing.T) {
	s, _, owner, peer := fixture(t)
	ctx := context.Background()
	_ = s.Store.PutFriendState(ctx, domain.FriendState{Owner: owner, Peer: peer, Status: "pending_in"})
	if _, err := s.Friend(ctx, owner, peer, "refuse", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveSearch(ctx, owner, adapter.SearchUser{UID: peer, Vercode: "new-proof"}); err != nil {
		t.Fatal(err)
	}
	v, _ := s.Store.GetFriendState(ctx, owner, peer)
	if v.Status != "refused" {
		t.Fatalf("search changed status %s", v.Status)
	}
	if strings.Contains(v.VercodeCipher, "new-proof") {
		t.Fatal("plaintext vercode persisted")
	}
}
