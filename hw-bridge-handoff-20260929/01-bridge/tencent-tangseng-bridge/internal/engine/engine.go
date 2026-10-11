package engine

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/adapter"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
)

type Engine struct {
	TencentNoticeSender string
	RecordDelivery      func(context.Context, domain.MessageEvent, domain.Identity, domain.Identity, string, string) error
	PrepareMessage      func(context.Context, domain.MessageEvent, domain.Identity, domain.Identity) (domain.MessageEvent, error)
	BusinessHandler     func(context.Context, domain.BusinessEvent) error
	GroupResolver       func(context.Context, domain.MessageEvent) (*domain.Identity, error)
	TencentScope        func(context.Context, domain.MessageEvent) (bool, error)
	Store               store.Store
	Tencent             adapter.Target
	Tang                adapter.Target
	Queue               interface {
		Enqueue(context.Context, string) error
	}
	MaxAttempts int
}

func (e *Engine) Accept(ctx context.Context, envelope domain.EventEnvelope) (bool, error) {
	if envelope.EventID == "" {
		envelope.EventID = domain.NewEventID()
	}
	if envelope.TraceID == "" {
		envelope.TraceID = domain.NewTraceID()
	}
	if envelope.OccurredAt == 0 {
		envelope.OccurredAt = time.Now().Unix()
	}
	if envelope.Source != domain.SourceTencent && envelope.Source != domain.SourceTang {
		return false, fmt.Errorf("unsupported event source %q", envelope.Source)
	}
	if len(envelope.Payload) == 0 {
		return false, fmt.Errorf("event payload is required")
	}
	if envelope.IdempotencyKey == "" {
		envelope.IdempotencyKey = domain.MessageKey(envelope.Source, envelope.Command, envelope.EventID)
	}
	if e.BusinessHandler != nil {
		business, err := domain.ParseBusinessEvent(envelope)
		if err != nil {
			return false, err
		}
		if business != nil {
			// Commands concern the receiving account, unlike globally deduplicated group messages.
			if business.Owner != "" {
				envelope.IdempotencyKey += ":" + business.Owner
			}
			job := e.job(envelope, domain.MessageEvent{Source: envelope.Source, SourceMessageID: envelope.EventID, FromAccount: business.Owner, Business: business}, "")
			return e.Store.AcceptMessage(ctx, envelope, job)
		}
	}
	// The inbox is also the durable landing zone for relation, recall, read
	// receipt and group-management callbacks. Those handlers are added in the
	// later A+ phases; accepting them here must not create a bogus text job.
	if !messageCommand(envelope) {
		return e.Store.AcceptInbox(ctx, envelope)
	}
	message, err := parse(envelope)
	if err != nil {
		if errors.Is(err, domain.ErrIgnoredMessage) {
			return e.Store.AcceptInbox(ctx, envelope)
		}
		if envelope.Source == domain.SourceTang && ignorableTangSync(envelope.Payload) {
			// WuKong sends conversation/message sync notifications through the
			// same recv channel. They have no sender, channel, or text and are
			// acknowledged as control traffic rather than becoming dead text
			// relay jobs.
			return e.Store.AcceptInbox(ctx, envelope)
		}
		return e.Store.AcceptMessage(ctx, envelope, e.job(envelope, domain.MessageEvent{Source: envelope.Source, SourceMessageID: envelope.EventID}, err.Error()))
	}
	if message.BridgeMarker != "" && strings.Contains(message.BridgeMarker, "bridge") {
		return e.Store.AcceptInbox(ctx, envelope)
	}
	if message.FromAccount == "fileHelper" || message.ToAccount == "fileHelper" || message.ConversationID == "fileHelper" {
		return e.Store.AcceptInbox(ctx, envelope)
	}
	// Only explicitly mapped dd_ groups belong to the cross-system bridge.
	if message.Source == domain.SourceTencent && message.ConversationType == domain.ConversationGroup && !strings.HasPrefix(message.ConversationID, "dd_") {
		return e.Store.AcceptInbox(ctx, envelope)
	}
	if message.Source == domain.SourceTencent && e.TencentScope != nil {
		allowed, scopeErr := e.TencentScope(ctx, message)
		if scopeErr != nil {
			return false, scopeErr
		}
		if !allowed {
			return e.Store.AcceptInbox(ctx, envelope)
		}
	}
	// WuKong delivers a sender's own message back to that sender's PC(2)
	// session.  The proxy envelope uses the session owner as ToAccount, so a
	// same From/To pair is a local echo rather than a new cross-system message.
	// A second session (the shadow account) receives the real peer delivery and
	// is the only event that should create the Tencent relay job.
	if message.Source == domain.SourceTang && message.FromAccount == message.ToAccount {
		return e.Store.AcceptInbox(ctx, envelope)
	}
	// A TangSeng recv can echo a message sent by one of our own shadow
	// accounts. Those messages are delivery receipts of the bridge's send,
	// not new user messages. Drop them before creating an opposite-direction
	// job; otherwise the two shadow sessions can bounce the same text forever.
	if message.Source == domain.SourceTang {
		if shadow, lookupErr := e.Store.FindIdentityByTarget(ctx, domain.SourceTang, message.FromAccount); lookupErr == nil {
			// A TangSeng user can also be present in the contact/search
			// projection.  That reverse projection is not evidence that the
			// sender is one of our own shadow accounts.  Only suppress the
			// event when the mapping points at a different Tencent owner;
			// self-mappings (tang UID == Tencent UID) are real users and must
			// continue to the Tencent relay.
			if shadow.SourceID != message.FromAccount {
				return e.Store.AcceptInbox(ctx, envelope)
			}
		}
	}
	job := e.job(envelope, message, "")
	accepted, err := e.Store.AcceptMessage(ctx, envelope, job)
	if err != nil {
		return false, err
	}
	if e.Queue != nil {
		_ = e.Queue.Enqueue(ctx, job.ID)
	}
	return accepted, nil
}

func ignorableTangSync(payload json.RawMessage) bool {
	var envelope struct {
		Payload json.RawMessage `json:"payload"`
	}
	if json.Unmarshal(payload, &envelope) != nil || len(envelope.Payload) == 0 {
		return false
	}
	var encoded string
	if json.Unmarshal(envelope.Payload, &encoded) == nil {
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return false
		}
		envelope.Payload = decoded
	}
	var control struct {
		Cmd  string `json:"cmd"`
		Type int    `json:"type"`
	}
	if json.Unmarshal(envelope.Payload, &control) != nil {
		return false
	}
	return control.Type == 99 || strings.HasPrefix(control.Cmd, "sync")
}

func messageCommand(envelope domain.EventEnvelope) bool {
	if envelope.Source == domain.SourceTencent {
		switch envelope.Command {
		case "C2C.CallbackAfterSendMsg", "Group.CallbackAfterSendMsg":
			return true
		}
		return false
	}
	return envelope.Command == "msg.notify" || envelope.Command == "send"
}

func parse(envelope domain.EventEnvelope) (domain.MessageEvent, error) {
	if envelope.Source == domain.SourceTencent {
		return domain.ParseTencentMessage(envelope.Payload)
	}
	return domain.ParseTangMessage(envelope.Payload)
}

func (e *Engine) job(envelope domain.EventEnvelope, message domain.MessageEvent, parseError string) domain.OutboxJob {
	jobID := domain.StableID("outbox", envelope.IdempotencyKey)
	// Nanosecond ordering is required because chat clients routinely send
	// several messages within one second. The database uses this value only as
	// an ordering key; retry timestamps remain Unix seconds.
	job := domain.OutboxJob{ID: jobID, EventID: envelope.EventID, Target: opposite(envelope.Source), Message: message, Status: "pending", CreatedAt: time.Now().UnixNano(), NextAttemptAt: time.Now().Unix()}
	if parseError != "" {
		job.LastError = parseError
	}
	return job
}

func opposite(source domain.Source) domain.Source {
	if source == domain.SourceTencent {
		return domain.SourceTang
	}
	return domain.SourceTencent
}

func (e *Engine) ProcessDue(ctx context.Context, limit int) (int, error) {
	jobs, err := e.Store.ClaimDueOutbox(ctx, limit)
	if err != nil {
		return 0, err
	}
	processed := 0
	var firstErr error
	for _, job := range jobs {
		if err := e.processJob(ctx, job); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("outbox job %s source=%s target=%s message=%s failed: %w", job.ID, job.Message.Source, job.Target, job.Message.SourceMessageID, err)
			}
			continue
		}
		processed++
	}
	return processed, firstErr
}

func (e *Engine) processJob(ctx context.Context, job domain.OutboxJob) error {
	if job.Message.FailureNotice != nil {
		return e.sendFailure(ctx, job)
	}
	if job.Message.Business != nil {
		if e.BusinessHandler == nil {
			return e.fail(ctx, job, fmt.Errorf("business handler unavailable"))
		}
		if err := e.BusinessHandler(ctx, *job.Message.Business); err != nil {
			return e.fail(ctx, job, err)
		}
		return e.Store.CompleteOutbox(ctx, job.ID, domain.MessageMapping{Source: job.Message.Source, SourceMessageID: "business:" + job.ID, Target: job.Target, Status: "done", CreatedAt: job.CreatedAt, UpdatedAt: time.Now().Unix()})
	}
	if job.Message.Content == "" {
		return e.fail(ctx, job, fmt.Errorf("message content is empty"))
	}
	targetID := job.Message.ConversationID
	var to *domain.Identity
	var err error
	if job.Message.ConversationType == domain.ConversationGroup {
		if e.GroupResolver == nil {
			return e.fail(ctx, job, fmt.Errorf("group service unavailable"))
		}
		to, err = e.GroupResolver(ctx, job.Message)
	} else {
		if job.Message.ToAccount != "" {
			targetID = job.Message.ToAccount
		}
		to, err = e.ensureIdentity(ctx, job.Message.Source, targetID, "user")
	}
	if err != nil {
		return e.fail(ctx, job, err)
	}
	from, err := e.ensureIdentity(ctx, job.Message.Source, job.Message.FromAccount, "user")
	if err != nil {
		return e.fail(ctx, job, err)
	}
	var target adapter.Target
	if job.Target == domain.SourceTencent {
		target = e.Tencent
	} else {
		target = e.Tang
	}
	if target == nil {
		return e.fail(ctx, job, fmt.Errorf("target adapter %s is not configured", job.Target))
	}
	marker := job.EventID
	message := job.Message
	if e.PrepareMessage != nil {
		message, err = e.PrepareMessage(ctx, message, *from, *to)
		if err != nil {
			for _, part := range job.Message.Parts {
				if part.Kind == "image" || part.Kind == "audio" || part.Kind == "video" || part.Kind == "file" {
					err = &domain.DeliveryError{Reason: "media_transfer", Public: "媒体未送达：文件处理失败，请重新发送", Media: true}
					break
				}
			}
			return e.fail(ctx, job, err)
		}
	}
	var targetMessageID string
	if job.Target == domain.SourceTang && len(message.Parts) > 1 {
		for i, part := range message.Parts {
			child := message
			child.SourceMessageID = fmt.Sprintf("%s#part%d", message.SourceMessageID, i)
			child.Parts = []domain.MessagePart{part}
			if old, lookup := e.Store.FindMessage(ctx, child.Source, child.SourceMessageID); lookup == nil && old.Status == "done" {
				if i == 0 {
					targetMessageID = old.TargetMessageID
				}
				continue
			}
			sent, sendErr := target.SendText(ctx, child, *from, *to, marker)
			if sendErr != nil {
				return e.fail(ctx, job, sendErr)
			}
			if i == 0 {
				targetMessageID = sent
			}
			if e.RecordDelivery != nil {
				if recordErr := e.RecordDelivery(ctx, child, *from, *to, sent, marker); recordErr != nil {
					return e.fail(ctx, job, recordErr)
				}
			}
			if saveErr := e.Store.PutMessage(ctx, domain.MessageMapping{Source: child.Source, SourceMessageID: child.SourceMessageID, Target: job.Target, TargetMessageID: sent, Status: "done", CreatedAt: job.CreatedAt, UpdatedAt: time.Now().Unix()}); saveErr != nil {
				return e.fail(ctx, job, saveErr)
			}
		}
	} else {
		targetMessageID, err = target.SendText(ctx, message, *from, *to, marker)
	}
	if err != nil {
		return e.fail(ctx, job, err)
	}
	mapping := domain.MessageMapping{Source: job.Message.Source, SourceMessageID: job.Message.SourceMessageID, Target: job.Target, TargetMessageID: targetMessageID, Status: "done", CreatedAt: job.CreatedAt, UpdatedAt: time.Now().Unix()}
	if e.RecordDelivery != nil && !(job.Target == domain.SourceTang && len(message.Parts) > 1) {
		if err = e.RecordDelivery(ctx, message, *from, *to, targetMessageID, marker); err != nil {
			return e.fail(ctx, job, err)
		}
	}
	return e.Store.CompleteOutbox(ctx, job.ID, mapping)
}

func (e *Engine) ensureIdentity(ctx context.Context, source domain.Source, id, entityType string) (*domain.Identity, error) {
	if id == "" {
		return nil, fmt.Errorf("identity id is empty")
	}
	if existing, err := e.Store.FindIdentity(ctx, source, id); err == nil {
		return existing, nil
	}
	// Provisioned mappings are stored in the forward direction (Tencent user
	// to Tang UID). Tang recv events carry the Tang UID, so resolve that edge
	// before creating a new shadow account.
	if reverse, err := e.Store.FindIdentityByTarget(ctx, source, id); err == nil {
		mapped := *reverse
		mapped.Source = source
		mapped.SourceID = id
		mapped.Target = opposite(source)
		mapped.TargetID = reverse.SourceID
		return &mapped, nil
	}
	target := opposite(source)
	identity := domain.Identity{Source: source, SourceID: id, Target: target, TargetID: domain.ShadowID(source, id), EntityType: entityType, Shadow: true}
	if source == domain.SourceTang && entityType == "user" {
		identity.TargetID = id
	}
	if err := e.Store.UpsertIdentity(ctx, identity); err != nil {
		return nil, err
	}
	adapterTarget := e.Tang
	if target == domain.SourceTencent {
		adapterTarget = e.Tencent
	}
	if adapterTarget != nil {
		var ensured domain.Identity
		var ensureErr error
		if entityType == "group" {
			ensured, ensureErr = adapterTarget.EnsureGroup(ctx, identity)
		} else {
			ensured, ensureErr = adapterTarget.EnsureUser(ctx, identity)
		}
		if ensureErr != nil {
			return nil, ensureErr
		}
		if ensured.TargetID != "" && ensured.TargetID != identity.TargetID {
			identity.TargetID = ensured.TargetID
			if err := e.Store.UpsertIdentity(ctx, identity); err != nil {
				return nil, err
			}
		}
	}
	return &identity, nil
}

func (e *Engine) fail(ctx context.Context, job domain.OutboxJob, err error) error {
	max := e.MaxAttempts
	if max < 1 {
		max = 5
	}
	dead := job.Attempts+1 >= max
	var delivery *domain.DeliveryError
	if errors.As(err, &delivery) && delivery.Permanent {
		dead = true
	}
	persistCtx := ctx
	var cancel context.CancelFunc
	if ctx.Err() != nil {
		persistCtx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
	}
	if job.Message.FailureNotice == nil && job.Message.Business == nil && delivery != nil && dead && (delivery.Permanent || delivery.Media) {
		// Persist the independent notification before finalizing the original job.
		// A crash between these writes retries an idempotent insertion.
		if noticeErr := e.queueFailure(persistCtx, job, delivery); noticeErr != nil {
			return fmt.Errorf("persist failure notice: %w", noticeErr)
		}
	}
	if persistErr := e.Store.FailOutbox(persistCtx, job.ID, err.Error(), dead); persistErr != nil {
		return fmt.Errorf("%w; persist outbox failure: %v", err, persistErr)
	}
	return err
}

func Encode(v any) json.RawMessage { data, _ := json.Marshal(v); return data }
