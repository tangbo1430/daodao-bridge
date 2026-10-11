package adapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
)

func TestTangSengMockRegisterAndSend(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/user/usernameregister":
			if body["username"] == nil || body["password"] == nil {
				t.Errorf("register body missing fields: %#v", body)
			}
			_, _ = w.Write([]byte(`{"data":{"uid":"mock-uid-1","token":"mock-user-token"}}`))
		case "/v1/group/create":
			_, _ = w.Write([]byte(`{"group_no":"mock-group-1"}`))
		case "/v1/message/send":
			if body["token"] != "mock-user-token" {
				t.Errorf("unexpected sender token: %#v", body["token"])
			}
			payload, _ := body["payload"].(map[string]any)
			if payload["content"] != "hello mock" {
				t.Errorf("unexpected payload: %#v", payload)
			}
			_, _ = w.Write([]byte(`{"message_id":"mock-message-1"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := TangSengClient{BaseURL: server.URL, Token: "mock-service-token", UserEndpoint: "/v1/user/usernameregister", GroupEndpoint: "/v1/group/create", SendEndpoint: "/v1/message/send", ShadowPassword: "mock-password", Manufacturer: "bridge", UserTokens: &sync.Map{}}
	identity, err := client.EnsureUser(context.Background(), domain.Identity{Source: domain.SourceTencent, SourceID: "tencent-user", TargetID: "tx_tencent-user", DisplayName: "Mock User"})
	if err != nil {
		t.Fatal(err)
	}
	if identity.TargetID != "mock-uid-1" {
		t.Fatalf("unexpected uid: %s", identity.TargetID)
	}
	messageID, err := client.SendText(context.Background(), domain.MessageEvent{Source: domain.SourceTencent, SourceMessageID: "source-1", ConversationID: "mock-uid-1", ConversationType: domain.ConversationC2C, FromAccount: "tencent-user", Content: "hello mock"}, identity, domain.Identity{TargetID: "mock-uid-2"}, "event-1")
	if err != nil {
		t.Fatal(err)
	}
	if messageID != "mock-message-1" {
		t.Fatalf("unexpected message id: %s", messageID)
	}
	if !strings.Contains(strings.Join(paths, ","), "/v1/user/usernameregister") {
		t.Fatal(paths)
	}
}

func TestTangSengLoginStoresPC2Token(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/user/usernamelogin" {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["username"] != "ceshijishu888" || body["flag"] != float64(2) {
			t.Errorf("unexpected login body: %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"uid":"real-test-uid","token":"real-test-token"}}`))
	}))
	defer server.Close()
	tokens := &sync.Map{}
	client := TangSengClient{BaseURL: server.URL, LoginEndpoint: "/v1/user/usernamelogin", Username: "ceshijishu888", Password: "123456", UserTokens: tokens}
	uid, err := client.Login(context.Background())
	if err != nil || uid != "real-test-uid" {
		t.Fatalf("login failed uid=%s err=%v", uid, err)
	}
	value, ok := tokens.Load(uid)
	if !ok || value != "real-test-token" {
		t.Fatalf("token not stored: %#v", value)
	}
}

func TestParseSearchUsersSingleExactMatch(t *testing.T) {
	users, err := parseSearchUsers(json.RawMessage(`{"data":{"uid":"f6e7a19da487461597c1d6563019a2ba","name":"温暖晨曦","vercode":"v@1"},"exist":1}`))
	if err != nil || len(users) != 1 {
		t.Fatalf("unexpected search result users=%#v err=%v", users, err)
	}
	if users[0].UID != "f6e7a19da487461597c1d6563019a2ba" || users[0].Vercode != "v@1" {
		t.Fatalf("unexpected user: %#v", users[0])
	}
}

func TestEnsureUserUsesStableUsernameAfterRestore(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["username"] != "tx_27723" {
			t.Errorf("restore used wrong username: %#v", body["username"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"uid":"38a51c7d74a4480b9325e455bdbbcf7b","token":"restored-token"}}`))
	}))
	defer server.Close()

	client := TangSengClient{
		BaseURL:        server.URL,
		UserEndpoint:   "/v1/user/usernameregister",
		ShadowPassword: "12345678",
		UserTokens:     &sync.Map{},
	}
	identity, err := client.EnsureUser(context.Background(), domain.Identity{
		Source:   domain.SourceTencent,
		SourceID: "27723",
		// This is the persisted TangSeng UID from the first provision.
		TargetID: "38a51c7d74a4480b9325e455bdbbcf7b",
	})
	if err != nil {
		t.Fatal(err)
	}
	if identity.TargetID != "38a51c7d74a4480b9325e455bdbbcf7b" {
		t.Fatalf("unexpected restored uid: %s", identity.TargetID)
	}
}

func TestStableMessageRandomIsDeterministic(t *testing.T) {
	first := stableMessageRandom("message-1", "event-1", "from", "to")
	second := stableMessageRandom("message-1", "event-1", "from", "to")
	if first <= 0 || first != second {
		t.Fatalf("unstable message random: first=%d second=%d", first, second)
	}
	if first == stableMessageRandom("message-2", "event-1", "from", "to") {
		t.Fatal("different source messages unexpectedly share the same random value")
	}
}
