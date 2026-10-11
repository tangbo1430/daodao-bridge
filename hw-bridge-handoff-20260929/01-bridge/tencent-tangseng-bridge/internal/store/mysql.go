package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
)

type MySQLStore struct{ DB *sql.DB }

func NewMySQLStore(db *sql.DB) *MySQLStore { return &MySQLStore{DB: db} }

func (s *MySQLStore) AcceptInbox(ctx context.Context, event domain.EventEnvelope) (bool, error) {
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return false, err
	}
	result, err := s.DB.ExecContext(ctx, `INSERT IGNORE INTO bridge_inbox_event (event_id,idempotency_key,trace_id,source_system,command_name,occurred_at,payload_json,status,created_at) VALUES (?,?,?,?,?,?,?,?,?)`, event.EventID, event.IdempotencyKey, event.TraceID, event.Source, event.Command, event.OccurredAt, payload, "accepted", time.Now().Unix())
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	return count == 1, err
}

func (s *MySQLStore) AcceptMessage(ctx context.Context, event domain.EventEnvelope, job domain.OutboxJob) (bool, error) {
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return false, err
	}
	message, err := json.Marshal(job.Message)
	if err != nil {
		return false, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT IGNORE INTO bridge_inbox_event (event_id,idempotency_key,trace_id,source_system,command_name,occurred_at,payload_json,status,created_at) VALUES (?,?,?,?,?,?,?,?,?)`, event.EventID, event.IdempotencyKey, event.TraceID, event.Source, event.Command, event.OccurredAt, payload, "accepted", time.Now().Unix())
	if err != nil {
		return false, err
	}
	accepted, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	// INSERT IGNORE preserves a completed or dead job on redelivery, while
	// restoring a missing job for an inbox row created by an older release.
	_, err = tx.ExecContext(ctx, `INSERT IGNORE INTO bridge_outbox_job (id,event_id,target_system,message_json,status,attempts,next_attempt_at,last_error,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?)`, job.ID, job.EventID, job.Target, message, job.Status, job.Attempts, job.NextAttemptAt, job.LastError, job.CreatedAt, job.UpdatedAt)
	if err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return accepted == 1, nil
}

func (s *MySQLStore) FindIdentity(ctx context.Context, source domain.Source, id string) (*domain.Identity, error) {
	var value domain.Identity
	err := s.DB.QueryRowContext(ctx, `SELECT source_system,source_id,target_system,target_id,entity_type,display_name,avatar_url,shadow FROM bridge_identity_mapping WHERE source_system=? AND source_id=? LIMIT 1`, source, id).Scan(&value.Source, &value.SourceID, &value.Target, &value.TargetID, &value.EntityType, &value.DisplayName, &value.AvatarURL, &value.Shadow)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &value, err
}

func (s *MySQLStore) FindIdentityByTarget(ctx context.Context, target domain.Source, id string) (*domain.Identity, error) {
	var value domain.Identity
	err := s.DB.QueryRowContext(ctx, `SELECT source_system,source_id,target_system,target_id,entity_type,display_name,avatar_url,shadow FROM bridge_identity_mapping WHERE target_system=? AND target_id=? LIMIT 1`, target, id).Scan(&value.Source, &value.SourceID, &value.Target, &value.TargetID, &value.EntityType, &value.DisplayName, &value.AvatarURL, &value.Shadow)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &value, err
}

func (s *MySQLStore) UpsertIdentity(ctx context.Context, value domain.Identity) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO bridge_identity_mapping (source_system,source_id,target_system,target_id,entity_type,display_name,avatar_url,shadow,status,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE target_id=VALUES(target_id),display_name=VALUES(display_name),avatar_url=VALUES(avatar_url),updated_at=VALUES(updated_at)`, value.Source, value.SourceID, value.Target, value.TargetID, value.EntityType, value.DisplayName, value.AvatarURL, value.Shadow, "active", time.Now().Unix(), time.Now().Unix())
	return err
}

func (s *MySQLStore) ListIdentities(ctx context.Context) ([]domain.Identity, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT source_system,source_id,target_system,target_id,entity_type,display_name,avatar_url,shadow FROM bridge_identity_mapping WHERE entity_type='user' AND status='active'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]domain.Identity, 0)
	for rows.Next() {
		var value domain.Identity
		if err := rows.Scan(&value.Source, &value.SourceID, &value.Target, &value.TargetID, &value.EntityType, &value.DisplayName, &value.AvatarURL, &value.Shadow); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (s *MySQLStore) UpsertFriendLink(ctx context.Context, source domain.Source, owner, peer, status string) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO friend_link (source_system,owner_id,peer_id,relation_type,status,updated_at) VALUES (?,?,?,?,?,?) ON DUPLICATE KEY UPDATE status=VALUES(status),updated_at=VALUES(updated_at)`, source, owner, peer, "friend", status, time.Now().Unix())
	return err
}

func (s *MySQLStore) FindMessage(ctx context.Context, source domain.Source, id string) (*domain.MessageMapping, error) {
	var value domain.MessageMapping
	err := s.DB.QueryRowContext(ctx, `SELECT source_system,source_message_id,target_system,target_message_id,status,created_at,updated_at FROM bridge_message_mapping WHERE source_system=? AND source_message_id=? LIMIT 1`, source, id).Scan(&value.Source, &value.SourceMessageID, &value.Target, &value.TargetMessageID, &value.Status, &value.CreatedAt, &value.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &value, err
}

func (s *MySQLStore) PutMessage(ctx context.Context, value domain.MessageMapping) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO bridge_message_mapping (source_system,source_message_id,target_system,target_message_id,status,created_at,updated_at) VALUES (?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE target_message_id=VALUES(target_message_id),status=VALUES(status),updated_at=VALUES(updated_at)`, value.Source, value.SourceMessageID, value.Target, value.TargetMessageID, value.Status, value.CreatedAt, value.UpdatedAt)
	return err
}

func (s *MySQLStore) CreateOutbox(ctx context.Context, value domain.OutboxJob) error {
	payload, err := json.Marshal(value.Message)
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx, `INSERT IGNORE INTO bridge_outbox_job (id,event_id,target_system,message_json,status,attempts,next_attempt_at,last_error,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?)`, value.ID, value.EventID, value.Target, payload, value.Status, value.Attempts, value.NextAttemptAt, value.LastError, value.CreatedAt, value.UpdatedAt)
	return err
}

func (s *MySQLStore) ClaimDueOutbox(ctx context.Context, limit int) ([]domain.OutboxJob, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin outbox transaction: %w", err)
	}
	defer tx.Rollback()
	if limit < 1 {
		limit = 1
	}
	if limit > 1000 {
		limit = 1000
	}
	now := time.Now().Unix()
	// A process can stop after claiming a job and before completing or
	// rescheduling it. Recover that lease so a restart never leaves the message
	// permanently stuck in `processing`.
	if _, err := tx.ExecContext(ctx, `UPDATE bridge_outbox_job SET status='retry',next_attempt_at=?,last_error=CASE WHEN last_error='' THEN 'recovered stale processing job' ELSE last_error END,updated_at=? WHERE status='processing' AND updated_at<?`, now, now, now-60); err != nil {
		return nil, fmt.Errorf("recover stale outbox jobs: %w", err)
	}
	// Keep the LIMIT literal in this statement. Some MySQL deployments reject
	// prepared LIMIT placeholders and close the connection instead of returning
	// a useful SQL error.
	// Do not claim a later event while an older event in the same source
	// conversation is unfinished. This prevents a transient Tencent REST error
	// from looking like a missing middle message while newer messages overtake
	// it.
	query := fmt.Sprintf(`SELECT j.id,j.event_id,j.target_system,j.message_json,j.status,j.attempts,j.next_attempt_at,j.last_error,j.created_at,j.updated_at
		FROM bridge_outbox_job j
		WHERE j.status IN ('pending','retry') AND j.next_attempt_at<=?
		  AND NOT EXISTS (
			SELECT 1 FROM bridge_outbox_job older
			WHERE older.status IN ('pending','retry','processing')
			  AND (older.created_at<j.created_at OR (older.created_at=j.created_at AND older.id<j.id))
			  AND JSON_UNQUOTE(JSON_EXTRACT(older.message_json,'$.source'))=JSON_UNQUOTE(JSON_EXTRACT(j.message_json,'$.source'))
			  AND JSON_UNQUOTE(JSON_EXTRACT(older.message_json,'$.conversation_id'))=JSON_UNQUOTE(JSON_EXTRACT(j.message_json,'$.conversation_id'))
		  )
		ORDER BY j.created_at,j.id LIMIT %d FOR UPDATE`, limit)
	rows, err := tx.QueryContext(ctx, query, now)
	if err != nil {
		return nil, fmt.Errorf("query due outbox: %w", err)
	}
	defer rows.Close()
	jobs := make([]domain.OutboxJob, 0, limit)
	for rows.Next() {
		var job domain.OutboxJob
		var raw []byte
		if err := rows.Scan(&job.ID, &job.EventID, &job.Target, &raw, &job.Status, &job.Attempts, &job.NextAttemptAt, &job.LastError, &job.CreatedAt, &job.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan outbox row: %w", err)
		}
		if err := json.Unmarshal(raw, &job.Message); err != nil {
			return nil, fmt.Errorf("decode outbox row: %w", err)
		}
		jobs = append(jobs, job)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close outbox rows: %w", err)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate outbox rows: %w", err)
	}
	for i := range jobs {
		jobs[i].Status = "processing"
		jobs[i].UpdatedAt = time.Now().Unix()
		if _, err := tx.ExecContext(ctx, `UPDATE bridge_outbox_job SET status='processing',updated_at=? WHERE id=?`, jobs[i].UpdatedAt, jobs[i].ID); err != nil {
			return nil, fmt.Errorf("mark outbox processing: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit outbox transaction: %w", err)
	}
	return jobs, nil
}

func (s *MySQLStore) CompleteOutbox(ctx context.Context, id string, mapping domain.MessageMapping) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE bridge_outbox_job SET status='done',updated_at=? WHERE id=?`, time.Now().Unix(), id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO bridge_message_mapping (source_system,source_message_id,target_system,target_message_id,status,created_at,updated_at) VALUES (?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE target_message_id=VALUES(target_message_id),status=VALUES(status),updated_at=VALUES(updated_at)`, mapping.Source, mapping.SourceMessageID, mapping.Target, mapping.TargetMessageID, mapping.Status, mapping.CreatedAt, mapping.UpdatedAt); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *MySQLStore) FailOutbox(ctx context.Context, id, message string, dead bool) error {
	message = limitError(message)
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var job domain.OutboxJob
	var payload []byte
	err = tx.QueryRowContext(ctx, `SELECT event_id,message_json,attempts FROM bridge_outbox_job WHERE id=? FOR UPDATE`, id).Scan(&job.EventID, &payload, &job.Attempts)
	if err != nil {
		return err
	}
	job.Attempts++
	// Short retries keep a transient REST/connection error from appearing as a
	// lost chat message. Persistent errors still back off and become dead after
	// MaxAttempts.
	delays := []int64{1, 3, 10, 30, 120}
	index := job.Attempts - 1
	if index >= len(delays) {
		index = len(delays) - 1
	}
	status, next := "retry", time.Now().Unix()+delays[index]
	if dead {
		status, next = "dead", 0
	}
	_, err = tx.ExecContext(ctx, `UPDATE bridge_outbox_job SET status=?,attempts=?,next_attempt_at=?,last_error=?,updated_at=? WHERE id=?`, status, job.Attempts, next, message, time.Now().Unix(), id)
	if err != nil {
		return err
	}
	if dead {
		_, err = tx.ExecContext(ctx, `INSERT INTO bridge_dead_letter (job_id,event_id,reason,payload_json,attempts,status,created_at,retried_at) VALUES (?,?,?,?,?,'dead',?,0) ON DUPLICATE KEY UPDATE reason=VALUES(reason),attempts=VALUES(attempts),status='dead'`, id, job.EventID, message, payload, job.Attempts, time.Now().Unix())
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *MySQLStore) RetryDeadLetter(ctx context.Context, id string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var status, lastError string
	var raw []byte
	err = tx.QueryRowContext(ctx, `SELECT status,last_error,message_json FROM bridge_outbox_job WHERE id=? FOR UPDATE`, id).Scan(&status, &lastError, &raw)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if status != "dead" {
		return ErrNotFound
	}
	var message domain.MessageEvent
	if err := json.Unmarshal(raw, &message); err != nil {
		return err
	}
	var delivered int
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM bridge_message_mapping WHERE source_system=? AND source_message_id=? AND status='done'`, message.Source, message.SourceMessageID).Scan(&delivered)
	if err != nil {
		return err
	}
	if delivered > 0 {
		return ErrAlreadyDelivered
	}
	if ambiguousDelivery(lastError) {
		return ErrAmbiguousDelivery
	}
	now := time.Now().Unix()
	if _, err := tx.ExecContext(ctx, `UPDATE bridge_outbox_job SET status='retry',next_attempt_at=?,updated_at=? WHERE id=?`, now, now, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE bridge_dead_letter SET status='replayed',retried_at=? WHERE job_id=?`, now, id); err != nil {
		return err
	}
	audit, _ := json.Marshal(map[string]string{"job_id": id})
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_log (trace_id,action_name,actor,payload_json,created_at) VALUES (?,?,?,?,?)`, id, "dead_letter_replay", "bridge_admin", audit, now); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *MySQLStore) ListDeadLetters(ctx context.Context, limit int) ([]DeadLetter, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	rows, err := s.DB.QueryContext(ctx, fmt.Sprintf(`SELECT id,event_id,target_system,message_json,attempts,last_error,created_at,updated_at FROM bridge_outbox_job WHERE status='dead' ORDER BY updated_at DESC LIMIT %d`, limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]DeadLetter, 0)
	for rows.Next() {
		var item DeadLetter
		var raw []byte
		if err := rows.Scan(&item.JobID, &item.EventID, &item.Target, &raw, &item.Attempts, &item.LastError, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		var message domain.MessageEvent
		if err := json.Unmarshal(raw, &message); err != nil {
			return nil, err
		}
		item.Source, item.SourceMessageID, item.ConversationID = message.Source, message.SourceMessageID, message.ConversationID
		values = append(values, item)
	}
	return values, rows.Err()
}

func (s *MySQLStore) Stats(ctx context.Context) (Stats, error) {
	var result Stats
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM bridge_inbox_event`).Scan(&result.InboxReceived)
	if err != nil {
		return result, err
	}
	if err = s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM bridge_outbox_job WHERE status IN ('pending','retry','processing')`).Scan(&result.OutboxPending); err != nil {
		return result, err
	}
	if err = s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM bridge_outbox_job WHERE status='dead'`).Scan(&result.OutboxDead); err != nil {
		return result, err
	}
	err = s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM bridge_message_mapping WHERE status='done'`).Scan(&result.MessagesDone)
	return result, err
}

func ValidateMySQL(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("nil MySQL database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return db.PingContext(ctx)
}
