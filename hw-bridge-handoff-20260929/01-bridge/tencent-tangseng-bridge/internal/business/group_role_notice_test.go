package business

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/engine"
)

func TestRoleNoticeDurableOnceAndNoBootstrapReplay(t *testing.T) {
	s, f, owner, peer := groupService(t)
	ctx := context.Background()
	id, _ := s.owner(ctx, owner)
	req := GroupRequest{Owner: owner, Members: []string{peer}, RequestID: "role-test"}
	if _, err := s.Group(ctx, "create", req); err != nil {
		t.Fatal(err)
	}
	if stats, _ := s.Store.Stats(ctx); stats.OutboxPending != 0 {
		t.Fatal("initial mirror generated role notice")
	}
	f.members[1].Role = 2
	f.members[1].Version = 3
	for i := 0; i < 3; i++ {
		if _, err := s.Group(ctx, "sync", GroupRequest{Owner: owner, GroupID: "dd_fixture-group"}); err != nil {
			t.Fatal(err)
		}
	}
	jobs, err := s.Store.ClaimDueOutbox(ctx, 10)
	if err != nil || len(jobs) != 1 {
		t.Fatalf("jobs=%d err=%v", len(jobs), err)
	}
	b := *jobs[0].Message.Business
	// Snapshot loss must retain the durable job without reporting delivery success.
	g, _ := s.Store.GetGroupState(ctx, "fixture-group")
	stale := g
	stale.Members = append([]domain.GroupMember(nil), g.Members...)
	for i := range stale.Members {
		if stale.Members[i].UID == peer {
			stale.Members[i].Version = 2
		}
	}
	s.Store.PutGroupState(ctx, stale)
	if err = s.Handle(ctx, b); err == nil {
		t.Fatal("notice sent before snapshot committed")
	}
	s.Store.PutGroupState(ctx, g)
	// A failed delivery does not advance the sent checkpoint.
	s.Tang.UserTokens.Delete(id.TargetID)
	if err = s.Handle(ctx, b); err == nil {
		t.Fatal("missing credential accepted")
	}
	if _, err = s.Store.GetCursor(ctx, b.EventID); err == nil {
		t.Fatal("failed send recorded successful")
	}
	s.Tang.UserTokens.Store(id.TargetID, "owner-fixture-token")
	s.Tang.SendEndpoint = "/v1/message/send"
	before := len(f.calls)
	for i := 0; i < 2; i++ {
		if err = s.Handle(ctx, b); err != nil {
			t.Fatal(err)
		}
	}
	sent := f.calls[before:]
	if len(sent) != 1 || sent[0].Token != "owner-fixture-token" {
		t.Fatalf("unexpected sends %v", sent)
	}
	p := sent[0].Body["payload"].(map[string]any)
	if p["type"] != float64(2000) || p["content"] != "Peer已被设置为群管理员" || p["bridge_event_id"] != b.EventID {
		t.Fatal(p)
	}
	// Every PC(2) receiver consumes the same notice without a reverse job.
	e := engine.Engine{Store: s.Store, BusinessHandler: s.Handle}
	for _, receiver := range []string{id.TargetID, "second-huawei"} {
		raw, _ := json.Marshal(map[string]any{"message_id": "notice-native", "from_uid": id.TargetID, "to_uid": receiver, "channel_id": "fixture-group", "channel_type": 2, "payload": p})
		if _, err = e.Accept(ctx, domain.EventEnvelope{Source: domain.SourceTang, Command: "msg.notify", EventID: "echo-" + receiver, Payload: raw}); err != nil {
			t.Fatal(err)
		}
	}
	if stats, _ := s.Store.Stats(ctx); stats.OutboxPending != 1 {
		t.Fatal("echo created a second job")
	}
}

func TestRoleNoticeDemotionSupersededAndRejectedChanges(t *testing.T) {
	s, f, owner, peer := groupService(t)
	ctx := context.Background()
	s.Group(ctx, "create", GroupRequest{Owner: owner, Members: []string{peer}, RequestID: "roles"})
	// A request whose native state did not change must not produce a success notice.
	s.Group(ctx, "update", GroupRequest{Owner: owner, GroupID: "dd_fixture-group", Field: "manager", Members: []string{peer}})
	if stats, _ := s.Store.Stats(ctx); stats.OutboxPending != 0 {
		t.Fatal("unchanged role notified")
	}
	for i, role := range []int{2, 0} {
		f.members[1].Role = role
		f.members[1].Version = int64(3 + i)
		if _, err := s.Group(ctx, "sync", GroupRequest{Owner: owner, GroupID: "dd_fixture-group"}); err != nil {
			t.Fatal(err)
		}
	}
	before := len(f.calls)
	s.Tang.SendEndpoint = "/v1/message/send"
	e := engine.Engine{Store: s.Store, BusinessHandler: s.Handle}
	for i := 0; i < 2; i++ {
		if count, err := e.ProcessDue(ctx, 10); err != nil || count != 1 {
			t.Fatal(count, err)
		}
	}
	if len(f.calls) != before+1 {
		t.Fatal("superseded promotion should not be sent")
	}
	p := f.calls[before].Body["payload"].(map[string]any)
	if p["content"] != "Peer的群管理员身份已取消" {
		t.Fatal(p)
	}
}
