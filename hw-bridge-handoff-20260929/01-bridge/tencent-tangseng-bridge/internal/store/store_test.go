package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
)

func TestMemoryStoreClaimsOneMessagePerConversation(t *testing.T) {
	st := NewMemoryStore()
	now := time.Now().Unix()
	for i, id := range []string{"first", "second"} {
		if err := st.CreateOutbox(context.Background(), domain.OutboxJob{
			ID: id, EventID: id, Target: domain.SourceTencent, Status: "pending",
			CreatedAt: int64(i + 1), NextAttemptAt: now,
			Message: domain.MessageEvent{Source: domain.SourceTang, SourceMessageID: id, ConversationID: "same-peer", Content: id},
		}); err != nil {
			t.Fatal(err)
		}
	}
	jobs, err := st.ClaimDueOutbox(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 || jobs[0].ID != "first" {
		t.Fatalf("expected only the oldest conversation job, got %#v", jobs)
	}
	if err := st.CompleteOutbox(context.Background(), "first", domain.MessageMapping{Source: domain.SourceTang, SourceMessageID: "first", Status: "done"}); err != nil {
		t.Fatal(err)
	}
	jobs, err = st.ClaimDueOutbox(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 || jobs[0].ID != "second" {
		t.Fatalf("expected next conversation job after completion, got %#v", jobs)
	}
}

func TestLongRemoteErrorFitsDeadLetterColumns(t *testing.T) {
	st := NewMemoryStore()
	ctx := context.Background()
	job := domain.OutboxJob{ID: "long-error", Target: domain.SourceTencent, Status: "pending", Message: domain.MessageEvent{Source: domain.SourceTang, SourceMessageID: "m1"}}
	if err := st.CreateOutbox(ctx, job); err != nil {
		t.Fatal(err)
	}
	if err := st.FailOutbox(ctx, job.ID, strings.Repeat("失败", 800), true); err != nil {
		t.Fatal(err)
	}
	items, err := st.ListDeadLetters(ctx, 1)
	if err != nil || len(items) != 1 || len([]rune(items[0].LastError)) != 1000 {
		t.Fatalf("truncated error: items=%+v err=%v", items, err)
	}
}

func TestDeadLetterAuditAndReplayGuards(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryStore()
	newJob := func(id string) domain.OutboxJob {
		return domain.OutboxJob{ID: id, EventID: id, Target: domain.SourceTang, Status: "pending", NextAttemptAt: time.Now().Unix(), Message: domain.MessageEvent{Source: domain.SourceTencent, SourceMessageID: id, ConversationID: "peer", Content: "hello"}}
	}
	for _, id := range []string{"delivered", "ambiguous", "safe"} {
		if err := st.CreateOutbox(ctx, newJob(id)); err != nil {
			t.Fatal(err)
		}
	}
	if err := st.FailOutbox(ctx, "delivered", "permission denied", true); err != nil {
		t.Fatal(err)
	}
	if err := st.FailOutbox(ctx, "ambiguous", "context deadline exceeded", true); err != nil {
		t.Fatal(err)
	}
	if err := st.FailOutbox(ctx, "safe", "missing sender token", true); err != nil {
		t.Fatal(err)
	}
	if err := st.PutMessage(ctx, domain.MessageMapping{Source: domain.SourceTencent, SourceMessageID: "delivered", Status: "done"}); err != nil {
		t.Fatal(err)
	}
	items, err := st.ListDeadLetters(ctx, 10)
	if err != nil || len(items) != 3 || items[0].JobID == "" || items[0].SourceMessageID == "" {
		t.Fatalf("dead letter audit: items=%+v err=%v", items, err)
	}
	if err := st.RetryDeadLetter(ctx, "delivered"); !errors.Is(err, ErrAlreadyDelivered) {
		t.Fatalf("delivered replay should be blocked: %v", err)
	}
	if err := st.RetryDeadLetter(ctx, "ambiguous"); !errors.Is(err, ErrAmbiguousDelivery) {
		t.Fatalf("ambiguous replay should be blocked: %v", err)
	}
	if err := st.RetryDeadLetter(ctx, "safe"); err != nil {
		t.Fatalf("safe replay: %v", err)
	}
	stats, _ := st.Stats(ctx)
	if stats.OutboxPending != 1 || stats.OutboxDead != 2 {
		t.Fatalf("replay stats=%+v", stats)
	}
}

func TestMemoryStoreRecoversStaleProcessingJob(t *testing.T) {
	st := NewMemoryStore()
	now := time.Now().Unix()
	st.outbox["stale"] = domain.OutboxJob{
		ID: "stale", Status: "processing", CreatedAt: 1, UpdatedAt: now - 120,
		Message: domain.MessageEvent{Source: domain.SourceTang, ConversationID: "peer", Content: "hello"},
	}
	jobs, err := st.ClaimDueOutbox(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 || jobs[0].ID != "stale" {
		t.Fatalf("stale processing job was not recovered: %#v", jobs)
	}
}
