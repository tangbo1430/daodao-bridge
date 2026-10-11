package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"time"
)

func (s *MemoryStore) GetMedia(_ context.Context, id string) (domain.MediaObject, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.media[id]
	if !ok {
		return v, ErrNotFound
	}
	return v, nil
}
func (s *MemoryStore) PutMedia(_ context.Context, v domain.MediaObject) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.media == nil {
		s.media = map[string]domain.MediaObject{}
	}
	s.media[v.SHA256] = v
	return nil
}
func (s *MySQLStore) GetMedia(ctx context.Context, id string) (domain.MediaObject, error) {
	var raw string
	var v domain.MediaObject
	err := s.DB.QueryRowContext(ctx, "SELECT object_json FROM media_cache WHERE sha256=? AND status='ready' AND object_json IS NOT NULL", id).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return v, ErrNotFound
	}
	if err != nil {
		return v, err
	}
	err = json.Unmarshal([]byte(raw), &v)
	return v, err
}
func (s *MySQLStore) PutMedia(ctx context.Context, v domain.MediaObject) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	_, err = s.DB.ExecContext(ctx, "INSERT INTO media_cache(sha256,source_url,object_url,media_type,size_bytes,status,created_at,updated_at,object_json) VALUES(?,?,?,?,?,'ready',?,?,?) ON DUPLICATE KEY UPDATE object_url=VALUES(object_url),object_json=VALUES(object_json),status='ready',updated_at=VALUES(updated_at)", v.SHA256, v.SourceURL, v.ObjectURL, v.MIME, v.Size, now, now, string(raw))
	return err
}
