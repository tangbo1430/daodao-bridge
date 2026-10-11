package engine

import (
	"context"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"time"
)

func (e *Engine) queueFailure(ctx context.Context, job domain.OutboxJob, failure *domain.DeliveryError) error {
	id := domain.StableID("failure-notice-v1", job.ID)
	sender := job.Message.ToAccount
	text := failure.Public
	if job.Message.ConversationType == domain.ConversationGroup {
		// Failure is private to its author, never broadcast to all group members.
		if job.Message.Source == domain.SourceTencent {
			sender = e.TencentNoticeSender
		}
		text = "群消息：" + text
	}
	message := domain.MessageEvent{Source: job.Message.Source, SourceMessageID: id,
		ConversationID: "failure:" + job.ID, ConversationType: domain.ConversationC2C,
		Content: text, Parts: []domain.MessagePart{{Kind: "tip", Text: text}}, OccurredAt: time.Now().Unix(),
		FailureNotice: &domain.FailureNotice{OriginalJob: job.ID, Reason: failure.Reason, Sender: sender, Recipient: job.Message.FromAccount}}
	return e.Store.CreateOutbox(ctx, domain.OutboxJob{ID: id, EventID: id, Target: job.Message.Source, Message: message,
		Status: "pending", CreatedAt: time.Now().UnixNano(), NextAttemptAt: time.Now().Unix()})
}

func (e *Engine) sendFailure(ctx context.Context, job domain.OutboxJob) error {
	n := job.Message.FailureNotice
	if n.Sender == "" || n.Recipient == "" {
		return e.fail(ctx, job, fmt.Errorf("failure notice identity unavailable"))
	}
	target := e.Tencent
	if job.Target == domain.SourceTang {
		target = e.Tang
	}
	if target == nil {
		return e.fail(ctx, job, fmt.Errorf("failure notice adapter unavailable"))
	}
	message := job.Message
	message.FromAccount, message.ToAccount, message.ConversationID = n.Sender, n.Recipient, n.Recipient
	message.BridgeMarker = "bridge-failure:" + job.ID
	id, err := target.SendText(ctx, message, domain.Identity{TargetID: n.Sender}, domain.Identity{TargetID: n.Recipient}, message.BridgeMarker)
	if err != nil {
		return e.fail(ctx, job, err)
	}
	return e.Store.CompleteOutbox(ctx, job.ID, domain.MessageMapping{Source: job.Message.Source, SourceMessageID: job.Message.SourceMessageID,
		Target: job.Target, TargetMessageID: id, Status: "done", CreatedAt: job.CreatedAt, UpdatedAt: time.Now().Unix()})
}
