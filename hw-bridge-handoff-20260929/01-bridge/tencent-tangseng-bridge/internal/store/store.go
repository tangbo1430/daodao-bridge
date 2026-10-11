package store

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
)

var ErrDuplicate = errors.New("duplicate inbox event")
var ErrNotFound = errors.New("store record not found")
var ErrAlreadyDelivered = errors.New("message mapping is already delivered")
var ErrAmbiguousDelivery = errors.New("delivery outcome is ambiguous; verify the destination before replay")

type Store interface {
	PutMessageRecord(context.Context, domain.MessageRecord) error
	FindMessageRecord(context.Context, domain.Source, string) (domain.MessageRecord, error)
	FindMessageRecords(context.Context, domain.Source, string) ([]domain.MessageRecord, error)
	ListMessageRecords(context.Context, string, string, int64) ([]domain.MessageRecord, error)
	GetCursor(context.Context, string) (domain.SyncCursor, error)
	ListCursors(context.Context, string) ([]domain.SyncCursor, error)
	PutCursor(context.Context, domain.SyncCursor) error
	GetMedia(context.Context, string) (domain.MediaObject, error)
	PutMedia(context.Context, domain.MediaObject) error
	GetGroupState(context.Context, string) (domain.GroupState, error)
	PutGroupState(context.Context, domain.GroupState) error
	ListGroupStates(context.Context) ([]domain.GroupState, error)
	GetFriendState(context.Context, string, string) (domain.FriendState, error)
	PutFriendState(context.Context, domain.FriendState) error
	ListFriendStates(context.Context, string) ([]domain.FriendState, error)
	AcceptInbox(context.Context, domain.EventEnvelope) (bool, error)
	FindTangInboxEvent(context.Context, string, string) (domain.EventEnvelope, error)
	AcceptMessage(context.Context, domain.EventEnvelope, domain.OutboxJob) (bool, error)
	FindIdentity(context.Context, domain.Source, string) (*domain.Identity, error)
	FindIdentityByTarget(context.Context, domain.Source, string) (*domain.Identity, error)
	ListIdentities(context.Context) ([]domain.Identity, error)
	UpsertFriendLink(context.Context, domain.Source, string, string, string) error
	UpsertIdentity(context.Context, domain.Identity) error
	FindMessage(context.Context, domain.Source, string) (*domain.MessageMapping, error)
	PutMessage(context.Context, domain.MessageMapping) error
	CreateOutbox(context.Context, domain.OutboxJob) error
	ClaimDueOutbox(context.Context, int) ([]domain.OutboxJob, error)
	CompleteOutbox(context.Context, string, domain.MessageMapping) error
	FailOutbox(context.Context, string, string, bool) error
	ListDeadLetters(context.Context, int) ([]DeadLetter, error)
	RetryDeadLetter(context.Context, string) error
	Stats(context.Context) (Stats, error)
}

type Stats struct {
	InboxReceived int64 `json:"inbox_received"`
	OutboxPending int64 `json:"outbox_pending"`
	OutboxDead    int64 `json:"outbox_dead"`
	MessagesDone  int64 `json:"messages_done"`
}

type DeadLetter struct {
	JobID           string        `json:"job_id"`
	EventID         string        `json:"event_id"`
	Source          domain.Source `json:"source"`
	SourceMessageID string        `json:"source_message_id"`
	ConversationID  string        `json:"conversation_id"`
	Target          domain.Source `json:"target"`
	Attempts        int           `json:"attempts"`
	LastError       string        `json:"last_error"`
	CreatedAt       int64         `json:"created_at"`
	UpdatedAt       int64         `json:"updated_at"`
}

func ambiguousDelivery(message string) bool {
	message = strings.ToLower(message)
	for _, fragment := range []string{"timeout", "deadline exceeded", "connection reset", "broken pipe", "unexpected eof"} {
		if strings.Contains(message, fragment) {
			return true
		}
	}
	return false
}

func limitError(message string) string {
	message = strings.ToValidUTF8(message, "�")
	runes := []rune(message)
	if len(runes) > 1000 {
		return string(runes[:1000])
	}
	return message
}

type MemoryStore struct {
	messageRecords map[string]domain.MessageRecord
	cursors        map[string]domain.SyncCursor
	media          map[string]domain.MediaObject
	mu             sync.Mutex
	inbox          map[string]domain.EventEnvelope
	identities     map[string]domain.Identity
	messages       map[string]domain.MessageMapping
	outbox         map[string]domain.OutboxJob
	friends        map[string]string
	friendStates   map[string]domain.FriendState
	groupStates    map[string]domain.GroupState
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		friendStates: make(map[string]domain.FriendState),
		groupStates:  make(map[string]domain.GroupState),
		inbox:        make(map[string]domain.EventEnvelope), identities: make(map[string]domain.Identity),
		messages: make(map[string]domain.MessageMapping), outbox: make(map[string]domain.OutboxJob), friends: make(map[string]string),
	}
}

func (s *MemoryStore) UpsertFriendLink(_ context.Context, source domain.Source, owner, peer, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.friends[string(source)+":"+owner+":"+peer] = status
	return nil
}

func (s *MemoryStore) AcceptInbox(_ context.Context, event domain.EventEnvelope) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.inbox[event.IdempotencyKey]; ok {
		return false, nil
	}
	s.inbox[event.IdempotencyKey] = event
	return true, nil
}

// AcceptMessage keeps the durable inbox record and its relay job together.
// A duplicate callback also repairs an inbox record left without an outbox
// job by an older Bridge version.
func (s *MemoryStore) AcceptMessage(_ context.Context, event domain.EventEnvelope, job domain.OutboxJob) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, duplicate := s.inbox[event.IdempotencyKey]
	if !duplicate {
		s.inbox[event.IdempotencyKey] = event
	}
	if _, exists := s.outbox[job.ID]; !exists {
		s.outbox[job.ID] = job
	}
	return !duplicate, nil
}

func identityKey(source domain.Source, id string) string { return string(source) + ":" + id }

func (s *MemoryStore) FindIdentity(_ context.Context, source domain.Source, id string) (*domain.Identity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.identities[identityKey(source, id)]
	if !ok {
		return nil, ErrNotFound
	}
	copy := value
	return &copy, nil
}

func (s *MemoryStore) FindIdentityByTarget(_ context.Context, target domain.Source, id string) (*domain.Identity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, value := range s.identities {
		if value.Target == target && value.TargetID == id {
			copy := value
			return &copy, nil
		}
	}
	return nil, ErrNotFound
}

func (s *MemoryStore) UpsertIdentity(_ context.Context, value domain.Identity) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.identities[identityKey(value.Source, value.SourceID)] = value
	return nil
}

func (s *MemoryStore) ListIdentities(_ context.Context) ([]domain.Identity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	values := make([]domain.Identity, 0, len(s.identities))
	for _, value := range s.identities {
		values = append(values, value)
	}
	return values, nil
}

func (s *MemoryStore) FindMessage(_ context.Context, source domain.Source, id string) (*domain.MessageMapping, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.messages[string(source)+":"+id]
	if !ok {
		return nil, ErrNotFound
	}
	copy := value
	return &copy, nil
}

func (s *MemoryStore) PutMessage(_ context.Context, value domain.MessageMapping) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages[string(value.Source)+":"+value.SourceMessageID] = value
	return nil
}

func (s *MemoryStore) CreateOutbox(_ context.Context, value domain.OutboxJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if value.CreatedAt == 0 {
		value.CreatedAt = time.Now().Unix()
	}
	if value.NextAttemptAt == 0 {
		value.NextAttemptAt = time.Now().Unix()
	}
	if value.Status == "" {
		value.Status = "pending"
	}
	if _, exists := s.outbox[value.ID]; exists {
		return nil
	}
	s.outbox[value.ID] = value
	return nil
}

func (s *MemoryStore) ClaimDueOutbox(_ context.Context, limit int) ([]domain.OutboxJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().Unix()
	all := make([]domain.OutboxJob, 0, len(s.outbox))
	for id, value := range s.outbox {
		if value.Status == "processing" && value.UpdatedAt < now-60 {
			value.Status = "retry"
			value.NextAttemptAt = now
			value.LastError = "recovered stale processing job"
			value.UpdatedAt = now
			s.outbox[id] = value
		}
		all = append(all, value)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].CreatedAt == all[j].CreatedAt {
			return all[i].ID < all[j].ID
		}
		return all[i].CreatedAt < all[j].CreatedAt
	})
	jobs := make([]domain.OutboxJob, 0, limit)
	blocked := make(map[string]bool)
	for _, value := range all {
		if value.Status != "pending" && value.Status != "retry" && value.Status != "processing" {
			continue
		}
		conversation := string(value.Message.Source) + ":" + value.Message.ConversationID
		if blocked[conversation] {
			continue
		}
		blocked[conversation] = true
		if len(jobs) >= limit || value.Status == "processing" || value.NextAttemptAt > now {
			continue
		}
		value.Status = "processing"
		value.UpdatedAt = now
		s.outbox[value.ID] = value
		jobs = append(jobs, value)
	}
	return jobs, nil
}

func (s *MemoryStore) CompleteOutbox(_ context.Context, id string, mapping domain.MessageMapping) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.outbox[id]
	if !ok {
		return ErrNotFound
	}
	job.Status, job.UpdatedAt = "done", time.Now().Unix()
	s.outbox[id] = job
	s.messages[string(mapping.Source)+":"+mapping.SourceMessageID] = mapping
	return nil
}

func (s *MemoryStore) FailOutbox(_ context.Context, id, message string, dead bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	message = limitError(message)
	job, ok := s.outbox[id]
	if !ok {
		return ErrNotFound
	}
	job.Attempts++
	job.LastError, job.UpdatedAt = message, time.Now().Unix()
	if dead {
		job.Status, job.NextAttemptAt = "dead", 0
	} else {
		job.Status = "retry"
		delays := []int64{1, 3, 10, 30, 120}
		i := job.Attempts - 1
		if i >= len(delays) {
			i = len(delays) - 1
		}
		job.NextAttemptAt = time.Now().Unix() + delays[i]
	}
	s.outbox[id] = job
	return nil
}

func (s *MemoryStore) RetryDeadLetter(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.outbox[id]
	if !ok {
		return ErrNotFound
	}
	if job.Status != "dead" {
		return ErrNotFound
	}
	if mapping, ok := s.messages[string(job.Message.Source)+":"+job.Message.SourceMessageID]; ok && mapping.Status == "done" {
		return ErrAlreadyDelivered
	}
	if ambiguousDelivery(job.LastError) {
		return ErrAmbiguousDelivery
	}
	job.Status, job.NextAttemptAt, job.UpdatedAt = "retry", time.Now().Unix(), time.Now().Unix()
	s.outbox[id] = job
	return nil
}

func (s *MemoryStore) ListDeadLetters(_ context.Context, limit int) ([]DeadLetter, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit < 1 || limit > 500 {
		limit = 100
	}
	values := make([]DeadLetter, 0)
	for _, job := range s.outbox {
		if job.Status != "dead" {
			continue
		}
		values = append(values, DeadLetter{JobID: job.ID, EventID: job.EventID, Source: job.Message.Source, SourceMessageID: job.Message.SourceMessageID, ConversationID: job.Message.ConversationID, Target: job.Target, Attempts: job.Attempts, LastError: job.LastError, CreatedAt: job.CreatedAt, UpdatedAt: job.UpdatedAt})
	}
	sort.Slice(values, func(i, j int) bool { return values[i].UpdatedAt > values[j].UpdatedAt })
	if len(values) > limit {
		values = values[:limit]
	}
	return values, nil
}

func (s *MemoryStore) Stats(_ context.Context) (Stats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stats := Stats{InboxReceived: int64(len(s.inbox))}
	for _, job := range s.outbox {
		switch job.Status {
		case "pending", "retry", "processing":
			stats.OutboxPending++
		case "dead":
			stats.OutboxDead++
		}
	}
	for _, message := range s.messages {
		if message.Status == "done" {
			stats.MessagesDone++
		}
	}
	return stats, nil
}
