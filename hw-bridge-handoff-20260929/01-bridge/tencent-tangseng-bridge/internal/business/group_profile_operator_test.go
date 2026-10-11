package business

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
)

func TestObservedProfileCallbacksAreConsumedWithoutReverseWrite(t *testing.T) {
	s, f, owner, _ := groupService(t)
	ctx := context.Background()
	raw, err := os.ReadFile("../../test/contract/group-profile-operator-observed-20261002.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Callbacks []map[string]any `json:"callbacks"`
	}
	if err = json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Callbacks) < 5 {
		t.Fatal("observed callback fixture incomplete")
	}
	for _, p := range fixture.Callbacks {
		operator := owner
		if p["Operator_Account"] == "fixture-admin" {
			operator = s.Tencent.AdminAccount
		}
		p["Operator_Account"] = operator
		for _, field := range []string{"Name", "Notification", "FaceUrl"} {
			if value, ok := p[field].(string); ok && operator == owner {
				if err = s.Store.PutCursor(ctx, domain.SyncCursor{Key: profileEchoKey("dd_fixture-group", operator, field, value), Value: int64(p["EventTime"].(float64)) - 1000}); err != nil {
					t.Fatal(err)
				}
			}
		}
		payload, _ := json.Marshal(p)
		if err = s.Handle(ctx, domain.BusinessEvent{Source: domain.SourceTencent, Command: "Group.CallbackAfterGroupInfoChanged", Payload: payload}); err != nil {
			t.Fatal(err)
		}
	}
	if len(f.calls) != 0 {
		t.Fatal("real callback shape caused reverse writes")
	}
}

func TestGroupProfileAuthenticatedOperatorAndEcho(t *testing.T) {
	s, f, owner, peer := groupService(t)
	ctx := context.Background()
	if _, err := s.Group(ctx, "create", GroupRequest{Owner: owner, Members: []string{peer}, RequestID: "operator"}); err != nil {
		t.Fatal(err)
	}
	f.calls = nil
	if _, err := s.Group(ctx, "update", GroupRequest{Owner: owner, GroupID: "dd_fixture-group", Field: "name", Value: "Changed"}); err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	for _, c := range f.calls {
		if c.Path == "/v4/group_open_http_svc/modify_group_base_info" {
			body = c.Body
		}
	}
	if body["From_Account"] != owner || body["Name"] != "Changed" || body["Introduction"] != nil {
		t.Fatalf("incorrect attribution: %v", body)
	}
	raw, _ := json.Marshal(map[string]any{"GroupId": "dd_fixture-group", "Operator_Account": owner, "Name": "Changed", "EventTime": time.Now().UnixMilli()})
	f.calls = nil
	e := domain.BusinessEvent{Source: domain.SourceTencent, Command: "Group.CallbackAfterGroupInfoChanged", Payload: raw}
	for i := 0; i < 2; i++ {
		if err := s.Handle(ctx, e); err != nil {
			t.Fatal(err)
		}
	}
	if len(f.calls) != 0 {
		t.Fatalf("mirror callback caused upstream calls: %v", f.calls)
	}
	if _, err := s.Group(ctx, "sync", GroupRequest{Owner: owner, GroupID: "dd_fixture-group"}); err != nil {
		t.Fatal(err)
	}
	for _, c := range f.calls {
		if c.Path == "/v4/group_open_http_svc/modify_group_base_info" {
			t.Fatal("repeat sync emitted another notice")
		}
	}
}

func TestGroupNativeOperatorIsNotReceivingProxy(t *testing.T) {
	s, f, owner, peer := groupService(t)
	ctx := context.Background()
	if _, err := s.Group(ctx, "create", GroupRequest{Owner: owner, Members: []string{peer}, RequestID: "native-op"}); err != nil {
		t.Fatal(err)
	}
	id, _ := s.owner(ctx, owner)
	f.name = "Native change"
	f.calls = nil
	payload, _ := json.Marshal(map[string]any{"type": 1005, "content": "{0}修改群名", "data": map[string]string{"name": f.name}, "extra": []map[string]string{{"uid": peer, "name": "Peer"}}})
	if err := s.Handle(ctx, domain.BusinessEvent{Source: domain.SourceTang, Command: "groupSystem", Owner: id.TargetID, ChannelID: "fixture-group", NativeMessageID: "native-change", Payload: payload}); err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, c := range f.calls {
		if c.Path == "/v4/group_open_http_svc/modify_group_base_info" {
			count++
			if c.Body["From_Account"] != peer {
				t.Fatalf("proxy incorrectly credited: %v", c.Body)
			}
		}
	}
	if count != 1 {
		t.Fatalf("profile updates: %d", count)
	}
	raw, _ := json.Marshal(map[string]any{"GroupId": "dd_fixture-group", "Operator_Account": peer, "Name": f.name, "EventTime": time.Now().UnixMilli()})
	f.calls = nil
	if err := s.Handle(ctx, domain.BusinessEvent{Source: domain.SourceTencent, Command: "Group.CallbackAfterGroupInfoChanged", Payload: raw}); err != nil {
		t.Fatal(err)
	}
	if len(f.calls) != 0 {
		t.Fatal("shadow callback caused reverse write")
	}
}

func TestProfileAttributionDoesNotClaimUnrelatedOrStaleFields(t *testing.T) {
	s, f, owner, _ := groupService(t)
	ctx := context.Background()
	id, _ := s.owner(ctx, owner)
	g := domain.GroupState{TencentID: "dd_fixture-group", Members: []domain.GroupMember{{UID: id.TargetID, TencentID: owner}}}
	profile := map[string]any{"GroupId": g.TencentID, "Name": "Newer name", "FaceUrl": "https://example.invalid/avatar"}
	if err := s.projectGroupProfile(ctx, g, profile, newGroupProfileChange(id.TargetID, "name", "Older name")); err != nil {
		t.Fatal(err)
	}
	for _, c := range f.calls {
		if c.Body["From_Account"] != nil {
			t.Fatal("stale event claimed newer profile")
		}
	}
	f.calls = nil
	if err := s.projectGroupProfile(ctx, g, profile, newGroupProfileChange(id.TargetID, "avatar", "source")); err != nil {
		t.Fatal(err)
	}
	if len(f.calls) != 2 || f.calls[0].Body["From_Account"] != owner || f.calls[0].Body["Name"] != nil || f.calls[1].Body["From_Account"] != nil {
		t.Fatalf("unrelated fields attributed: %v", f.calls)
	}
}

func TestProfileEchoPreservesGenuineFieldsAndExpires(t *testing.T) {
	s, _, owner, _ := groupService(t)
	ctx := context.Background()
	now := time.Now().UnixMilli()
	if err := s.Store.PutCursor(ctx, domain.SyncCursor{Key: profileEchoKey("dd_fixture-group", owner, "Name", "Mirror"), Value: now}); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		offset      int64
		namePresent bool
	}{{0, false}, {-10000, true}, {int64(11 * time.Minute / time.Millisecond), true}} {
		raw, _ := json.Marshal(map[string]any{"GroupId": "dd_fixture-group", "Operator_Account": owner, "Name": "Mirror", "Notification": "Real edit", "EventTime": now + test.offset})
		e, all, err := s.filterProfileEcho(ctx, domain.BusinessEvent{Command: "Group.CallbackAfterGroupInfoChanged", Payload: raw})
		if err != nil || all {
			t.Fatalf("genuine change dropped: %v", err)
		}
		var p map[string]any
		_ = json.Unmarshal(e.Payload, &p)
		_, present := p["Name"]
		if present != test.namePresent || p["Notification"] != "Real edit" {
			t.Fatalf("incorrect filter: %v", p)
		}
	}
}
