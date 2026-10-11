package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"sort"
	"strings"
	"time"
)

func (s *MemoryStore) ListCursors(_ context.Context, prefix string) ([]domain.SyncCursor, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []domain.SyncCursor{}
	for k, v := range s.cursors {
		if strings.HasPrefix(k, prefix) {
			out = append(out, v)
		}
	}
	return out, nil
}

func (s *MySQLStore) ListCursors(ctx context.Context, prefix string) ([]domain.SyncCursor, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT cursor_json FROM bridge_business_cursor WHERE LEFT(scope_key,?)=?`, len(prefix), prefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.SyncCursor{}
	for rows.Next() {
		var raw []byte
		var v domain.SyncCursor
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *MemoryStore) PutMessageRecord(_ context.Context, v domain.MessageRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.messageRecords == nil {
		s.messageRecords = make(map[string]domain.MessageRecord)
	}
	s.messageRecords[v.Key] = v
	return nil
}
func (s *MemoryStore) FindMessageRecord(_ context.Context, system domain.Source, key string) (domain.MessageRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, v := range s.messageRecords {
		if system == domain.SourceTang && v.TangID == key || system == domain.SourceTencent && v.TencentKey == key {
			return v, nil
		}
	}
	return domain.MessageRecord{}, ErrNotFound
}
func (s *MemoryStore) FindMessageRecords(_ context.Context, system domain.Source, key string) ([]domain.MessageRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []domain.MessageRecord{}
	for _, v := range s.messageRecords {
		if system == domain.SourceTang && v.TangID == key || system == domain.SourceTencent && v.TencentKey == key {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil, ErrNotFound
	}
	return out, nil
}
func (s *MySQLStore) FindMessageRecords(ctx context.Context, system domain.Source, key string) ([]domain.MessageRecord, error) {
	column := "tencent_key"
	if system == domain.SourceTang {
		column = "tang_id"
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT record_json FROM bridge_message_record WHERE `+column+`=?`, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.MessageRecord{}
	for rows.Next() {
		var b []byte
		var v domain.MessageRecord
		if err = rows.Scan(&b); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(b, &v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, ErrNotFound
	}
	return out, nil
}
func (s *MemoryStore) ListMessageRecords(_ context.Context, owner, peer string, before int64) ([]domain.MessageRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []domain.MessageRecord{}
	for _, v := range s.messageRecords {
		if v.ChannelType == 1 && v.TencentTo == owner && v.TencentFrom == peer && v.OccurredAt <= before && !v.ReadSynced && !v.Revoked {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].OccurredAt < out[j].OccurredAt })
	if len(out) > 100 {
		out = out[:100]
	}
	return out, nil
}
func (s *MemoryStore) GetCursor(_ context.Context, key string) (domain.SyncCursor, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.cursors[key]
	if !ok {
		return v, ErrNotFound
	}
	return v, nil
}
func (s *MemoryStore) PutCursor(_ context.Context, v domain.SyncCursor) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cursors == nil {
		s.cursors = make(map[string]domain.SyncCursor)
	}
	s.cursors[v.Key] = v
	return nil
}
func (s *MySQLStore) PutMessageRecord(ctx context.Context, v domain.MessageRecord) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx, `INSERT INTO bridge_message_record (record_key,tencent_key,tang_id,tencent_from,tencent_to,channel_type,occurred_at,read_synced,revoked,record_json) VALUES (?,?,?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE tencent_key=VALUES(tencent_key),tang_id=VALUES(tang_id),read_synced=VALUES(read_synced),revoked=VALUES(revoked),record_json=VALUES(record_json)`, v.Key, v.TencentKey, v.TangID, v.TencentFrom, v.TencentTo, v.ChannelType, v.OccurredAt, v.ReadSynced, v.Revoked, b)
	return err
}
func (s *MySQLStore) FindMessageRecord(ctx context.Context, system domain.Source, key string) (domain.MessageRecord, error) {
	column := "tencent_key"
	if system == domain.SourceTang {
		column = "tang_id"
	}
	var b []byte
	var v domain.MessageRecord
	err := s.DB.QueryRowContext(ctx, `SELECT record_json FROM bridge_message_record WHERE `+column+`=? LIMIT 1`, key).Scan(&b)
	if err == sql.ErrNoRows {
		return v, ErrNotFound
	}
	if err != nil {
		return v, err
	}
	err = json.Unmarshal(b, &v)
	return v, err
}
func (s *MySQLStore) ListMessageRecords(ctx context.Context, owner, peer string, before int64) ([]domain.MessageRecord, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT record_json FROM bridge_message_record WHERE tencent_to=? AND tencent_from=? AND channel_type=1 AND occurred_at<=? AND read_synced=0 AND revoked=0 ORDER BY occurred_at LIMIT 100`, owner, peer, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.MessageRecord{}
	for rows.Next() {
		var b []byte
		var v domain.MessageRecord
		if err = rows.Scan(&b); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(b, &v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *MySQLStore) GetCursor(ctx context.Context, key string) (domain.SyncCursor, error) {
	var b []byte
	var v domain.SyncCursor
	err := s.DB.QueryRowContext(ctx, `SELECT cursor_json FROM bridge_business_cursor WHERE scope_key=?`, key).Scan(&b)
	if err == sql.ErrNoRows {
		return v, ErrNotFound
	}
	if err != nil {
		return v, err
	}
	err = json.Unmarshal(b, &v)
	return v, err
}
func (s *MySQLStore) PutCursor(ctx context.Context, v domain.SyncCursor) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx, `INSERT INTO bridge_business_cursor (scope_key,cursor_json,updated_at) VALUES (?,?,?) ON DUPLICATE KEY UPDATE cursor_json=VALUES(cursor_json),updated_at=VALUES(updated_at)`, v.Key, b, time.Now().Unix())
	return err
}
