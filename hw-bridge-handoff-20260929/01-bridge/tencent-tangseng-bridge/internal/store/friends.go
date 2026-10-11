package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"sort"
	"time"
)

func (s *MemoryStore) GetFriendState(_ context.Context, owner, peer string) (domain.FriendState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.friendStates[owner+":"+peer]
	if !ok {
		return v, ErrNotFound
	}
	return v, nil
}
func (s *MemoryStore) PutFriendState(_ context.Context, v domain.FriendState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	v.UpdatedAt = time.Now().Unix()
	s.friendStates[v.Owner+":"+v.Peer] = v
	return nil
}
func (s *MemoryStore) ListFriendStates(_ context.Context, owner string) ([]domain.FriendState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []domain.FriendState{}
	for _, v := range s.friendStates {
		if v.Owner == owner {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt > out[j].UpdatedAt })
	return out, nil
}
func (s *MySQLStore) GetFriendState(ctx context.Context, owner, peer string) (domain.FriendState, error) {
	var v domain.FriendState
	var data sql.NullString
	var status string
	err := s.DB.QueryRowContext(ctx, `SELECT status,friend_state_json FROM friend_link WHERE source_system='tencent' AND owner_id=? AND peer_id=? AND relation_type='friend'`, owner, peer).Scan(&status, &data)
	if err == sql.ErrNoRows {
		return v, ErrNotFound
	}
	if err != nil {
		return v, err
	}
	if data.Valid {
		if err = json.Unmarshal([]byte(data.String), &v); err != nil {
			return v, err
		}
	}
	v.Owner = owner
	v.Peer = peer
	v.Status = status
	return v, nil
}
func (s *MySQLStore) PutFriendState(ctx context.Context, v domain.FriendState) error {
	v.UpdatedAt = time.Now().Unix()
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx, `INSERT INTO friend_link (source_system,owner_id,peer_id,relation_type,status,updated_at,friend_state_json) VALUES ('tencent',?,?,'friend',?,?,?) ON DUPLICATE KEY UPDATE status=VALUES(status),updated_at=VALUES(updated_at),friend_state_json=VALUES(friend_state_json)`, v.Owner, v.Peer, v.Status, v.UpdatedAt, data)
	return err
}
func (s *MySQLStore) ListFriendStates(ctx context.Context, owner string) ([]domain.FriendState, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT peer_id,status,friend_state_json,updated_at FROM friend_link WHERE source_system='tencent' AND owner_id=? AND relation_type='friend' ORDER BY updated_at DESC`, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.FriendState{}
	for rows.Next() {
		var peer, status string
		var data sql.NullString
		var updated int64
		if err = rows.Scan(&peer, &status, &data, &updated); err != nil {
			return nil, err
		}
		var v domain.FriendState
		if data.Valid {
			if err = json.Unmarshal([]byte(data.String), &v); err != nil {
				return nil, err
			}
		}
		v.Owner = owner
		v.Peer = peer
		v.Status = status
		v.UpdatedAt = updated
		out = append(out, v)
	}
	return out, rows.Err()
}
