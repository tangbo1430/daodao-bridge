package engine

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
)

type fakeTarget struct {
	sent    int
	ensured int
}

type failingTarget struct{}

func (failingTarget) EnsureUser(_ context.Context, i domain.Identity) (domain.Identity, error) {
	return i, nil
}
func (failingTarget) EnsureGroup(_ context.Context, i domain.Identity) (domain.Identity, error) {
	return i, nil
}
func (failingTarget) SendText(context.Context, domain.MessageEvent, domain.Identity, domain.Identity, string) (string, error) {
	return "", errors.New("temporary send failure")
}

func (f *fakeTarget) EnsureUser(_ context.Context, i domain.Identity) (domain.Identity, error) {
	f.ensured++
	return i, nil
}
func (f *fakeTarget) EnsureGroup(_ context.Context, i domain.Identity) (domain.Identity, error) {
	f.ensured++
	return i, nil
}
func (f *fakeTarget) SendText(_ context.Context, _ domain.MessageEvent, _, _ domain.Identity, _ string) (string, error) {
	f.sent++
	return "target-1", nil
}

func TestAcceptIsIdempotentAndProcessesText(t *testing.T) {
	st := store.NewMemoryStore()
	target := &fakeTarget{}
	e := &Engine{Store: st, Tang: target, MaxAttempts: 3}
	payload := json.RawMessage(`{"CallbackCommand":"C2C.CallbackAfterSendMsg","From_Account":"10001","To_Account":"10002","MsgSeq":1,"MsgRandom":2,"MsgBody":[{"MsgType":"TIMTextElem","MsgContent":{"Text":"hello"}}]}`)
	env := domain.EventEnvelope{EventID: "e1", Source: domain.SourceTencent, Command: "C2C.CallbackAfterSendMsg", IdempotencyKey: "k1", Payload: payload}
	accepted, err := e.Accept(context.Background(), env)
	if err != nil || !accepted {
		t.Fatalf("accept: %v %v", accepted, err)
	}
	accepted, err = e.Accept(context.Background(), env)
	if err != nil || accepted {
		t.Fatalf("duplicate should be ignored: %v %v", accepted, err)
	}
	if _, err := e.ProcessDue(context.Background(), 10); err != nil {
		t.Fatal(err)
	}
	if target.sent != 1 {
		t.Fatalf("sent=%d", target.sent)
	}
	stats, _ := st.Stats(context.Background())
	if stats.MessagesDone != 1 {
		t.Fatalf("stats=%+v", stats)
	}
}

func TestDuplicateCallbackRepairsMissingOutbox(t *testing.T) {
	ctx := context.Background()
	st := store.NewMemoryStore()
	env := domain.EventEnvelope{
		EventID: "old-event", Source: domain.SourceTencent, Command: "C2C.CallbackAfterSendMsg",
		IdempotencyKey: "old-key",
		Payload:        json.RawMessage(`{"CallbackCommand":"C2C.CallbackAfterSendMsg","From_Account":"10001","To_Account":"10002","MsgKey":"message-1","MsgBody":[{"MsgType":"TIMTextElem","MsgContent":{"Text":"hello"}}]}`),
	}
	if accepted, err := st.AcceptInbox(ctx, env); err != nil || !accepted {
		t.Fatalf("prepare orphan inbox: accepted=%v err=%v", accepted, err)
	}
	e := &Engine{Store: st, Tang: &fakeTarget{}}
	if accepted, err := e.Accept(ctx, env); err != nil || accepted {
		t.Fatalf("duplicate repair: accepted=%v err=%v", accepted, err)
	}
	stats, err := st.Stats(ctx)
	if err != nil || stats.OutboxPending != 1 {
		t.Fatalf("missing repaired job: stats=%+v err=%v", stats, err)
	}
	if accepted, err := e.Accept(ctx, env); err != nil || accepted {
		t.Fatalf("second duplicate: accepted=%v err=%v", accepted, err)
	}
	stats, _ = st.Stats(ctx)
	if stats.OutboxPending != 1 {
		t.Fatalf("duplicate created another job: %+v", stats)
	}
}

func TestReverseProvisionedIdentityUsesSourceIDOnTarget(t *testing.T) {
	st := store.NewMemoryStore()
	if err := st.UpsertIdentity(context.Background(), domain.Identity{Source: domain.SourceTencent, SourceID: "64833", Target: domain.SourceTang, TargetID: "tang-u1", EntityType: "user", Shadow: true}); err != nil {
		t.Fatal(err)
	}
	e := &Engine{Store: st}
	got, err := e.ensureIdentity(context.Background(), domain.SourceTang, "tang-u1", "user")
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != domain.SourceTang || got.Target != domain.SourceTencent || got.TargetID != "64833" {
		t.Fatalf("unexpected mapping: %+v", got)
	}
}

func TestTangSelfMappingIsNotMistakenForShadowEcho(t *testing.T) {
	st := store.NewMemoryStore()
	// This is the contact projection created by a TangSeng search.  The
	// source and target IDs are equal, so it represents the real Tang user,
	// not a Tencent-owned shadow account.
	if err := st.UpsertIdentity(context.Background(), domain.Identity{Source: domain.SourceTang, SourceID: "tang-real", Target: domain.SourceTencent, TargetID: "tang-real", EntityType: "user", Shadow: true}); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertIdentity(context.Background(), domain.Identity{Source: domain.SourceTencent, SourceID: "74629", Target: domain.SourceTang, TargetID: "shadow-74629", EntityType: "user", Shadow: true}); err != nil {
		t.Fatal(err)
	}
	e := &Engine{Store: st}
	payload := json.RawMessage(`{"message_id":"m1","from_uid":"tang-real","to_uid":"shadow-74629","channel_id":"tang-real","channel_type":1,"payload":"eyJjb250ZW50IjoiaGkiLCJ0eXBlIjoxfQ=="}`)
	accepted, err := e.Accept(context.Background(), domain.EventEnvelope{EventID: "t1", Source: domain.SourceTang, Command: "msg.notify", IdempotencyKey: "tk1", Payload: payload})
	if err != nil || !accepted {
		t.Fatalf("accept: %v %v", accepted, err)
	}
	stats, _ := st.Stats(context.Background())
	if stats.OutboxPending != 1 {
		t.Fatalf("expected reverse relay job, stats=%+v", stats)
	}
}

func TestTangSenderLocalEchoIsIgnored(t *testing.T) {
	st := store.NewMemoryStore()
	e := &Engine{Store: st}
	payload := json.RawMessage(`{"message_id":"echo-1","from_uid":"tang-real","to_uid":"tang-real","channel_id":"shadow-74629","channel_type":1,"payload":"eyJjb250ZW50IjoiaGkiLCJ0eXBlIjoxfQ=="}`)
	accepted, err := e.Accept(context.Background(), domain.EventEnvelope{EventID: "echo-event", Source: domain.SourceTang, Command: "msg.notify", IdempotencyKey: "echo-key", Payload: payload})
	if err != nil || !accepted {
		t.Fatalf("accept: %v %v", accepted, err)
	}
	stats, _ := st.Stats(context.Background())
	if stats.OutboxPending != 0 {
		t.Fatalf("local echo created a relay job: stats=%+v", stats)
	}
}

func TestProcessDueReportsRelayFailure(t *testing.T) {
	st := store.NewMemoryStore()
	e := &Engine{Store: st, Tang: failingTarget{}, MaxAttempts: 3}
	payload := json.RawMessage(`{"CallbackCommand":"C2C.CallbackAfterSendMsg","From_Account":"10001","To_Account":"10002","MsgBody":[{"MsgType":"TIMTextElem","MsgContent":{"Text":"hello"}}]}`)
	accepted, err := e.Accept(context.Background(), domain.EventEnvelope{EventID: "failed-event", Source: domain.SourceTencent, Command: "C2C.CallbackAfterSendMsg", IdempotencyKey: "failed-key", Payload: payload})
	if err != nil || !accepted {
		t.Fatalf("accept: %v %v", accepted, err)
	}
	if _, err := e.ProcessDue(context.Background(), 10); err == nil || !strings.Contains(err.Error(), "temporary send failure") {
		t.Fatalf("expected relay failure to be observable, got %v", err)
	}
}
