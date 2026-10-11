package store

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
)

// FindTangInboxEvent recovers routing omitted by older persisted business jobs.
// One native command can have several proxy recipients: always bind to owner.
func (s *MemoryStore) FindTangInboxEvent(_ context.Context, eventID, owner string) (domain.EventEnvelope, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.inbox {
		if e.Source != domain.SourceTang || e.EventID != eventID {
			continue
		}
		var routing struct {
			Owner string `json:"to_uid"`
		}
		if json.Unmarshal(e.Payload, &routing) == nil && routing.Owner == owner {
			return e, nil
		}
	}
	return domain.EventEnvelope{}, ErrNotFound
}

func (s *MySQLStore) FindTangInboxEvent(ctx context.Context, eventID, owner string) (domain.EventEnvelope, error) {
	var e domain.EventEnvelope
	var payload []byte
	err := s.DB.QueryRowContext(ctx, `SELECT event_id,idempotency_key,command_name,payload_json,occurred_at FROM bridge_inbox_event WHERE event_id=? AND source_system='tangseng' AND JSON_UNQUOTE(JSON_EXTRACT(payload_json,'$.to_uid'))=? LIMIT 1`, eventID, owner).Scan(&e.EventID, &e.IdempotencyKey, &e.Command, &payload, &e.OccurredAt)
	if err == sql.ErrNoRows {
		return e, ErrNotFound
	}
	e.Source = domain.SourceTang
	e.Payload = json.RawMessage(payload)
	return e, err
}
