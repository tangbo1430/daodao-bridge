package engine

import (
	"context"
	"errors"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
	"testing"
)

type noticeTarget struct {
	fakeTarget
	messages []domain.MessageEvent
	from, to []string
	err      error
}

func (t *noticeTarget) SendText(_ context.Context, m domain.MessageEvent, f, to domain.Identity, _ string) (string, error) {
	t.messages = append(t.messages, m)
	t.from = append(t.from, f.TargetID)
	t.to = append(t.to, to.TargetID)
	return "notice-target", t.err
}
func failedJob() domain.OutboxJob {
	return domain.OutboxJob{ID: "fixture-job", EventID: "fixture-event", Target: domain.SourceTang, Message: domain.MessageEvent{Source: domain.SourceTencent, SourceMessageID: "fixture-message", FromAccount: "author", ToAccount: "peer", ConversationID: "peer", ConversationType: domain.ConversationC2C, Content: "hello"}}
}

func TestPermanentFailureNoticeDurableIdempotentAndPrivate(t *testing.T) {
	for _, group := range []bool{false, true} {
		t.Run(map[bool]string{true: "group", false: "c2c"}[group], func(t *testing.T) {
			ctx := context.Background()
			st := store.NewMemoryStore()
			target := &noticeTarget{}
			e := &Engine{Store: st, Tencent: target, TencentNoticeSender: "system"}
			job := failedJob()
			if group {
				job.Message.ConversationType = domain.ConversationGroup
				job.Message.ConversationID = "dd_group"
			}
			st.CreateOutbox(ctx, job)
			for i := 0; i < 2; i++ {
				e.fail(ctx, job, domain.WuKongFailure(4))
			}
			stats, _ := st.Stats(ctx)
			if stats.OutboxDead != 1 || stats.OutboxPending != 1 {
				t.Fatalf("%+v", stats)
			}
			// Simulate restart using the same durable store and no transient state.
			e = &Engine{Store: st, Tencent: target}
			if _, err := e.ProcessDue(ctx, 10); err != nil {
				t.Fatal(err)
			}
			if len(target.messages) != 1 || target.to[0] != "author" || target.messages[0].ConversationType != domain.ConversationC2C {
				t.Fatal(target.messages)
			}
			wantFrom := "peer"
			if group {
				wantFrom = "system"
			}
			if target.from[0] != wantFrom {
				t.Fatal(target.from)
			}
			if target.messages[0].Parts[0].Kind != "tip" || target.messages[0].BridgeMarker == "" {
				t.Fatal("missing safe tip metadata")
			}
		})
	}
}

func TestNoticeFailureNeverMarksNotifiedOrRecursivelyNotifies(t *testing.T) {
	ctx := context.Background()
	st := store.NewMemoryStore()
	e := &Engine{Store: st, Tencent: &noticeTarget{err: domain.WuKongFailure(4)}, MaxAttempts: 1}
	job := failedJob()
	st.CreateOutbox(ctx, job)
	e.fail(ctx, job, domain.WuKongFailure(4))
	e.ProcessDue(ctx, 10)
	stats, _ := st.Stats(ctx)
	if stats.MessagesDone != 0 || stats.OutboxDead != 2 || stats.OutboxPending != 0 {
		t.Fatalf("%+v", stats)
	}
}

func TestAmbiguousNetworkFailureDoesNotReportNotDelivered(t *testing.T) {
	ctx := context.Background()
	st := store.NewMemoryStore()
	e := &Engine{Store: st, MaxAttempts: 1}
	job := failedJob()
	st.CreateOutbox(ctx, job)
	e.fail(ctx, job, errors.New("send timeout"))
	stats, _ := st.Stats(ctx)
	if stats.OutboxPending != 0 || stats.OutboxDead != 1 {
		t.Fatalf("%+v", stats)
	}
}

func TestMediaFailureNotifiesOnlyAfterRetryBudget(t *testing.T) {
	ctx := context.Background()
	st := store.NewMemoryStore()
	e := &Engine{Store: st, MaxAttempts: 2}
	job := failedJob()
	st.CreateOutbox(ctx, job)
	err := &domain.DeliveryError{Reason: "media_transfer", Public: "媒体处理失败", Media: true}
	e.fail(ctx, job, err)
	stats, _ := st.Stats(ctx)
	if stats.OutboxPending != 1 {
		t.Fatal(stats)
	}
	job.Attempts = 1
	e.fail(ctx, job, err)
	stats, _ = st.Stats(ctx)
	if stats.OutboxDead != 1 || stats.OutboxPending != 1 {
		t.Fatal(stats)
	}
}
