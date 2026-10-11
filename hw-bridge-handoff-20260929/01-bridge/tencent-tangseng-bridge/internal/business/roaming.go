package business

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/adapter"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
	"sort"
	"time"
)

// SyncTencent only scans known bridge conversations. Ordinary Tencent chats
// are never fetched, rewritten, or imported into TangSeng.
func (s *Service) SyncTencent(ctx context.Context, owner string) (int, error) {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	id, err := s.owner(ctx, owner)
	if err != nil {
		return 0, err
	}
	if s.Accept == nil {
		return 0, fail(503, "同步收件接口未配置")
	}
	state, err := s.cursor(ctx, "account:"+owner, time.Now().Unix())
	if err != nil {
		return 0, err
	}
	if state.StartedAt == 0 {
		state.StartedAt = time.Now().Unix()
	}
	if state.Aux == "" {
		state.Aux = "confirmed"
	}
	if err = s.Store.PutCursor(ctx, state); err != nil {
		return 0, err
	}
	peers := map[string]bool{}
	friends, err := s.Store.ListFriendStates(ctx, owner)
	if err != nil {
		return 0, err
	}
	for _, f := range friends {
		peers[f.Peer] = true
	}
	convs, e := s.Store.GetCursor(ctx, "conversations:"+owner)
	if e != nil && !errors.Is(e, store.ErrNotFound) {
		return 0, e
	}
	if convs.Aux != "" {
		var channels map[string]adapter.TangConversation
		if err = json.Unmarshal([]byte(convs.Aux), &channels); err != nil {
			return 0, err
		}
		for _, c := range channels {
			if c.Type == 1 {
				peers[c.Channel] = true
			}
		}
	}
	count := 0
	for peer := range peers {
		mapped, e := s.Store.FindIdentity(ctx, domain.SourceTang, peer)
		if errors.Is(e, store.ErrNotFound) {
			continue
		}
		if e != nil {
			return count, e
		}
		if mapped.TargetID != peer {
			continue
		}
		n, e := s.roamC2C(ctx, owner, peer, state.StartedAt)
		count += n
		if e != nil {
			return count, e
		}
	}
	groups, err := s.Store.ListGroupStates(ctx)
	if err != nil {
		return count, err
	}
	for _, g := range groups {
		if g.Status == 2 || !g.MirrorReady {
			continue
		}
		member := false
		for _, m := range g.Members {
			if m.UID == id.TargetID && m.Deleted == 0 {
				member = true
			}
		}
		if !member {
			continue
		}
		n, e := s.roamGroup(ctx, g.TencentID, state.StartedAt)
		count += n
		if e != nil {
			return count, e
		}
	}
	return count, nil
}

func (s *Service) roamC2C(ctx context.Context, owner, peer string, started int64) (int, error) {
	c, err := s.cursor(ctx, "tim-c2c:"+domain.StableID(owner, peer), started)
	if err != nil {
		return 0, err
	}
	begin := c.Value - 60
	if begin < started {
		begin = started
	}
	until := time.Now().Unix() - 2
	count := 0
	// Fixed small windows bound memory while preserving oldest-first delivery.
	for window := 0; begin <= until && window < 288; window++ {
		end := begin + 300
		if end > until {
			end = until
		}
		max, last := end, ""
		messages := []adapter.TencentRoamMessage{}
		complete := false
		for page := 0; page < 100; page++ {
			batch, e := s.Tencent.RoamC2C(ctx, owner, peer, begin, max, last)
			if e != nil {
				return count, e
			}
			messages = append(messages, batch.Messages...)
			if batch.Complete == 1 {
				complete = true
				break
			}
			if batch.LastKey == "" || batch.LastTime < begin || batch.LastTime > max || (batch.LastTime == max && batch.LastKey == last) {
				return count, fmt.Errorf("Tencent C2C cursor stalled")
			}
			max, last = batch.LastTime, batch.LastKey
		}
		if !complete {
			return count, fmt.Errorf("Tencent C2C page limit exceeded")
		}
		sort.SliceStable(messages, func(i, j int) bool {
			if messages[i].Time == messages[j].Time {
				return messages[i].Seq < messages[j].Seq
			}
			return messages[i].Time < messages[j].Time
		})
		for _, m := range messages {
			if m.Time < started || m.Time < begin || m.Time > end {
				continue
			}
			if !((m.From == owner && m.To == peer) || (m.From == peer && m.To == owner)) {
				return count, fmt.Errorf("Tencent C2C history actor mismatch")
			}
			if m.Flags&8 != 0 { // Never re-send recalled content.
				if m.From == owner {
					if v, e := s.Store.FindMessageRecord(ctx, domain.SourceTencent, m.Key); e == nil && !v.Revoked {
						if e = s.Recall(ctx, owner, m.Key); e != nil {
							return count, e
						}
					}
				}
				continue
			}
			if m.From != owner {
				if m.Read == 1 {
					if e := s.syncRoamingRead(ctx, owner, peer, m.Key); e != nil {
						return count, e
					}
				}
				continue
			} // Incoming mirror history is already delivered.
			envelope, e := m.Envelope("")
			if e != nil {
				return count, e
			}
			ok, e := s.Accept(ctx, envelope)
			if e != nil {
				return count, e
			}
			if ok {
				count++
			}
		}
		c.Value = end
		if err = s.Store.PutCursor(ctx, c); err != nil {
			return count, err
		}
		begin = end + 1
	}
	if begin <= until {
		return count, fmt.Errorf("Tencent C2C catch-up window limit reached; cursor retained")
	}
	return count, nil
}

func (s *Service) roamGroup(ctx context.Context, group string, started int64) (int, error) {
	c, err := s.cursor(ctx, "tim-group:"+group, started)
	if err != nil {
		return 0, err
	}
	max, highest := int64(0), c.Value
	messages := []adapter.TencentRoamMessage{}
	complete := false
	for page := 0; page < 1000; page++ {
		batch, e := s.Tencent.RoamGroup(ctx, group, max)
		if e != nil {
			return 0, e
		}
		lowest := int64(0)
		stop := batch.Finished == 2 || len(batch.Messages) == 0
		for _, m := range batch.Messages {
			if lowest == 0 || m.Seq < lowest {
				lowest = m.Seq
			}
			if m.Seq > highest {
				highest = m.Seq
			}
			if m.Seq <= c.Value || (m.Time > 0 && m.Time < started) {
				stop = true
				continue
			}
			messages = append(messages, m)
		}
		if stop || lowest <= 1 {
			complete = true
			break
		}
		if lowest == 0 || (max > 0 && lowest > max) {
			return 0, fmt.Errorf("Tencent group history cursor stalled")
		}
		max = lowest - 1
	}
	if !complete {
		return 0, fmt.Errorf("Tencent group history page limit exceeded")
	}
	sort.SliceStable(messages, func(i, j int) bool { return messages[i].Seq < messages[j].Seq })
	count := 0
	for _, m := range messages {
		if m.Place != 0 || m.System != 0 || len(m.Body) == 0 || string(m.Body) == "null" {
			continue
		}
		if _, e := s.owner(ctx, m.From); e != nil {
			continue
		}
		envelope, e := m.Envelope(group)
		if e != nil {
			return count, e
		}
		ok, e := s.Accept(ctx, envelope)
		if e != nil {
			return count, e
		}
		if ok {
			count++
		}
	}
	c.Value = highest
	return count, s.Store.PutCursor(ctx, c)
}
