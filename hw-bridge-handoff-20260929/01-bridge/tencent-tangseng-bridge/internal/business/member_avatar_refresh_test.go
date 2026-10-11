package business

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
)

func TestMemberAvatarRefreshObservedContractRetryAndDedup(t *testing.T) {
	s, _, owner, _ := fixture(t)
	ctx := context.Background()
	id, _ := s.owner(ctx, owner)
	raw, err := os.ReadFile("../../test/contract/member-avatar-refresh-observed-20261002.json")
	if err != nil {
		t.Fatal(err)
	}
	var observed struct {
		Response json.RawMessage `json:"response"`
	}
	if err = json.Unmarshal(raw, &observed); err != nil {
		t.Fatal(err)
	}
	fail := true
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "PUT" || r.URL.Path != "/v1/groups/fixture-group/members/"+id.TargetID || r.Header.Get("token") != "owner-fixture-token" {
			t.Error("incorrect identity or endpoint")
		}
		var body map[string]any
		if json.NewDecoder(r.Body).Decode(&body) != nil || body == nil || len(body) != 0 {
			t.Error("member business fields overwritten")
		}
		w.Header().Set("Content-Type", "application/json")
		if fail {
			w.WriteHeader(503)
			_, _ = w.Write([]byte(`{"status":503}`))
			return
		}
		_, _ = w.Write(observed.Response)
	}))
	defer srv.Close()
	s.Tang.BaseURL = srv.URL
	g := domain.GroupState{Number: "fixture-group", MirrorReady: true, Status: 1, Members: []domain.GroupMember{{UID: id.TargetID, Huawei: true}}}
	if err = s.refreshOwnMemberAvatar(ctx, id, g); err != nil || calls != 0 {
		t.Fatal("no completed upload must not refresh")
	}
	_ = s.Store.PutCursor(ctx, domain.SyncCursor{Key: "avatar-upload:" + domain.StableID(id.TargetID, ""), Aux: "first-hash"})
	if err = s.refreshOwnMemberAvatar(ctx, id, g); err == nil {
		t.Fatal("failure was swallowed")
	}
	if _, err = s.Store.GetCursor(ctx, memberAvatarRefreshKey(g.Number, id.TargetID)); err == nil {
		t.Fatal("failed notification marked completed")
	}
	fail = false
	for i := 0; i < 3; i++ {
		if err = s.refreshOwnMemberAvatar(ctx, id, g); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 2 {
		t.Fatalf("repeated refresh: %d", calls)
	}
	_ = s.Store.PutCursor(ctx, domain.SyncCursor{Key: "avatar-upload:" + domain.StableID(id.TargetID, ""), Aux: "second-hash"})
	if err = s.refreshOwnMemberAvatar(ctx, id, g); err != nil || calls != 3 {
		t.Fatalf("changed content not refreshed: %v", err)
	}
	g.Members[0].Deleted = 1
	_ = s.Store.PutCursor(ctx, domain.SyncCursor{Key: "avatar-upload:" + domain.StableID(id.TargetID, ""), Aux: "third-hash"})
	if err = s.refreshOwnMemberAvatar(ctx, id, g); err != nil || calls != 3 {
		t.Fatal("departed member refreshed")
	}
	g.Members[0].Deleted = 0
	g.Status = 2
	if err = s.refreshOwnMemberAvatar(ctx, id, g); err != nil || calls != 3 {
		t.Fatal("disbanded group refreshed")
	}
}

func TestProfileRefreshesKnownGroupAndConsumesMemberEcho(t *testing.T) {
	s, f, owner, peer := groupService(t)
	ctx := context.Background()
	id, _ := s.owner(ctx, owner)
	if _, err := s.Group(ctx, "create", GroupRequest{Owner: owner, Members: []string{peer}, RequestID: "member-avatar"}); err != nil {
		t.Fatal(err)
	}
	_ = s.Store.PutCursor(ctx, domain.SyncCursor{Key: "avatar-upload:" + domain.StableID(id.TargetID, ""), Aux: "avatar-hash"})
	f.calls = nil
	if err := s.Profile(ctx, owner, "", ""); err != nil {
		t.Fatal(err)
	}
	payload := json.RawMessage(`{"type":99,"cmd":"memberUpdate","param":{"group_no":"fixture-group"}}`)
	for i := 0; i < 3; i++ {
		if err := s.Handle(ctx, domain.BusinessEvent{Source: domain.SourceTang, Command: "memberUpdate", Owner: id.TargetID, Payload: payload}); err != nil {
			t.Fatal(err)
		}
	}
	refreshes := 0
	for _, c := range f.calls {
		if c.Method == "PUT" && c.Path == "/v1/groups/fixture-group/members/"+id.TargetID {
			refreshes++
			if len(c.Body) != 0 {
				t.Fatal("remark overwritten")
			}
		}
		if c.Path == "/v4/group_open_http_svc/send_group_msg" || c.Path == "/v4/group_open_http_svc/modify_group_base_info" {
			t.Fatal("member avatar echo generated visible group notification")
		}
	}
	if refreshes != 1 {
		t.Fatalf("expected once per image, got %d", refreshes)
	}
}
