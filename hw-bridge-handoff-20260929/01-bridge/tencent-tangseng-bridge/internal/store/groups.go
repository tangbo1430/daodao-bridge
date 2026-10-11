package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"time"
)

func copyGroup(v domain.GroupState) domain.GroupState {
	v.Members = append([]domain.GroupMember(nil), v.Members...)
	v.PendingInvites = append([]string(nil), v.PendingInvites...)
	return v
}
func (s *MemoryStore) GetGroupState(_ context.Context, id string) (domain.GroupState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.groupStates[id]
	if !ok {
		return v, ErrNotFound
	}
	return copyGroup(v), nil
}
func (s *MemoryStore) PutGroupState(_ context.Context, v domain.GroupState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.groupStates[v.Number] = copyGroup(v)
	return nil
}
func (s *MemoryStore) ListGroupStates(_ context.Context) ([]domain.GroupState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []domain.GroupState{}
	for _, v := range s.groupStates {
		out = append(out, copyGroup(v))
	}
	return out, nil
}
func (s *MySQLStore) GetGroupState(ctx context.Context, id string) (domain.GroupState, error) {
	var v domain.GroupState
	var data sql.NullString
	err := s.DB.QueryRowContext(ctx, `SELECT group_state_json FROM group_map WHERE source_system='tangseng' AND source_group_id=?`, id).Scan(&data)
	if err == sql.ErrNoRows || (err == nil && !data.Valid) {
		return v, ErrNotFound
	}
	if err != nil {
		return v, err
	}
	err = json.Unmarshal([]byte(data.String), &v)
	return v, err
}
func (s *MySQLStore) ListGroupStates(ctx context.Context) ([]domain.GroupState, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT group_state_json FROM group_map WHERE source_system='tangseng' AND group_state_json IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.GroupState{}
	for rows.Next() {
		var data string
		var v domain.GroupState
		if err = rows.Scan(&data); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(data), &v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *MySQLStore) PutGroupState(ctx context.Context, v domain.GroupState) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	status := "pending"
	if v.MirrorReady {
		status = "active"
	}
	if v.Status == 2 {
		status = "disbanded"
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO group_map (source_group_id,target_group_id,source_system,target_system,status,created_at,updated_at,group_state_json) VALUES (?,?,'tangseng','tencent',?,?,?,?) ON DUPLICATE KEY UPDATE target_group_id=VALUES(target_group_id),status=VALUES(status),updated_at=VALUES(updated_at),group_state_json=VALUES(group_state_json)`, v.Number, v.TencentID, status, now, now, data)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE group_member_mirror SET status='deleted',updated_at=? WHERE group_source='tangseng' AND group_id=?`, now, v.Number)
	if err != nil {
		return err
	}
	for _, m := range v.Members {
		if m.TencentID == "" {
			continue
		}
		role := "member"
		if m.Role == 1 {
			role = "owner"
		} else if m.Role == 2 {
			role = "admin"
		}
		memberStatus := "active"
		if m.Deleted != 0 {
			memberStatus = "deleted"
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO group_member_mirror (group_source,group_id,member_source,member_id,role,status,updated_at) VALUES ('tangseng',?,'tencent',?,?,?,?) ON DUPLICATE KEY UPDATE role=VALUES(role),status=VALUES(status),updated_at=VALUES(updated_at)`, v.Number, m.TencentID, role, memberStatus, now)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
