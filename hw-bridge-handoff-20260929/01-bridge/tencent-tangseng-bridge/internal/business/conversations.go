package business

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/adapter"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/proxy"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/session"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
	"sort"
	"strconv"
	"strings"
	"time"
)

type SyncResult struct {
	Accepted  int   `json:"accepted"`
	Channels  int   `json:"channels"`
	StartedAt int64 `json:"started_at"`
}

func channelCursorKey(owner, channel string, ct int) string {
	return "channel:" + domain.StableID(owner, channel, strconv.Itoa(ct))
}
func (s *Service) cursor(ctx context.Context, key string, started int64) (domain.SyncCursor, error) {
	c, err := s.Store.GetCursor(ctx, key)
	if errors.Is(err, store.ErrNotFound) {
		return domain.SyncCursor{Key: key, StartedAt: started}, nil
	}
	return c, err
}

// SyncAccount complements PC(2) delivery using the same Inbox and message keys.
// The initial cutover is persisted before reading any remote history.
func (s *Service) SyncAccount(ctx context.Context, owner string) (SyncResult, error) {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	out := SyncResult{}
	id, err := s.owner(ctx, owner)
	if err != nil {
		return out, err
	}
	if s.Accept == nil {
		return out, fail(503, "同步收件接口未配置")
	}
	accountState, err := s.cursor(ctx, "account:"+owner, time.Now().Unix())
	if err != nil {
		return out, err
	}
	if accountState.StartedAt == 0 {
		accountState.StartedAt = time.Now().Unix()
	}
	if accountState.Aux == "" {
		accountState.Aux = "confirmed"
	}
	if err = s.Store.PutCursor(ctx, accountState); err != nil {
		return out, err
	}
	out.StartedAt = accountState.StartedAt
	cursor, err := s.cursor(ctx, "conversations:"+owner, out.StartedAt)
	if err != nil {
		return out, err
	}
	channels := map[string]adapter.TangConversation{}
	if cursor.Aux != "" {
		if err = json.Unmarshal([]byte(cursor.Aux), &channels); err != nil {
			return out, err
		}
	}
	seqs := []string{}
	for key, c := range channels {
		cc, e := s.cursor(ctx, channelCursorKey(owner, c.Channel, c.Type), out.StartedAt)
		if e != nil {
			return out, e
		}
		seqs = append(seqs, key+":"+strconv.FormatInt(cc.Value, 10))
	}
	sort.Strings(seqs)
	device := "hwbridge-sync-" + owner
	response, err := s.Tang.SyncConversations(ctx, id.TargetID, cursor.Value, strings.Join(seqs, "|"), device)
	if err != nil {
		return out, err
	}
	version := cursor.Value
	for _, c := range response.Conversations {
		if c.Channel == "" || c.Type < 1 || c.Type > 2 {
			continue
		}
		c.Recents = nil
		channels[c.Channel+":"+strconv.Itoa(c.Type)] = c
		if c.Version > version {
			version = c.Version
		}
	}
	// Large TangSeng groups may not maintain recent conversations. Their
	// persisted member mapping is therefore also a synchronization source.
	groups, err := s.Store.ListGroupStates(ctx)
	if err != nil {
		return out, err
	}
	for _, g := range groups {
		key := g.Number + ":2"
		if g.Status == 2 {
			delete(channels, key)
			continue
		}
		present := false
		for _, c := range response.Conversations {
			if c.Channel == g.Number && c.Type == 2 {
				present = true
				break
			}
		}
		if present {
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
		latest, e := s.Tang.LatestChannel(ctx, id.TargetID, g.Number, 2, device)
		if e != nil {
			return out, e
		}
		for _, m := range latest.Messages {
			if m.Channel != g.Number || m.ChannelType != 2 {
				return out, fmt.Errorf("group channel response mismatch")
			}
			previous := channels[key]
			if m.Seq >= previous.LastSeq {
				previous.Channel = g.Number
				previous.Type = 2
				previous.LastSeq = m.Seq
				previous.Timestamp = m.Timestamp
				channels[key] = previous
			}
		}
	}
	// Save the pending channel set without advancing its version. A later
	// request can resume a channel even if the upstream cache no longer lists it.
	metadata, err := json.Marshal(channels)
	if err != nil {
		return out, err
	}
	cursor.Aux = string(metadata)
	if err = s.Store.PutCursor(ctx, cursor); err != nil {
		return out, err
	}
	for key, c := range channels {
		if c.Type == 2 {
			g, e := s.Group(ctx, "sync", GroupRequest{Owner: owner, GroupID: "dd_" + c.Channel})
			var denied *Error
			if e != nil {
				if errors.As(e, &denied) && denied.Status == 403 {
					delete(channels, key)
					continue
				}
				return out, e
			}
			member := false
			for _, m := range g.Members {
				if m.UID == id.TargetID && m.Deleted == 0 {
					member = true
				}
			}
			if g.Status == 2 || !member {
				delete(channels, key)
				continue
			}
		}
		count, e := s.syncChannel(ctx, id, c, out.StartedAt, device)
		out.Accepted += count
		if e != nil {
			return out, e
		}
		out.Channels++
	}
	count, cmdSeq, err := s.syncCommands(ctx, id, out.StartedAt)
	out.Accepted += count
	if err != nil {
		return out, err
	}
	// All messages/commands are durable before confirming the server cursor.
	if err = s.Tang.AckConversations(ctx, id.TargetID, device, cmdSeq); err != nil {
		return out, err
	}
	cursor.Value = version
	metadata, err = json.Marshal(channels)
	if err != nil {
		return out, err
	}
	cursor.Aux = string(metadata)
	if err = s.Store.PutCursor(ctx, cursor); err != nil {
		return out, err
	}
	return out, nil
}

func (s *Service) syncChannel(ctx context.Context, id domain.Identity, c adapter.TangConversation, started int64, device string) (int, error) {
	cc, err := s.cursor(ctx, channelCursorKey(id.SourceID, c.Channel, c.Type), started)
	if err != nil {
		return 0, err
	}
	if cc.Value == 0 && c.Timestamp < started {
		cc.Value = c.LastSeq
		return 0, s.Store.PutCursor(ctx, cc)
	}
	if c.OffsetSeq > cc.Value {
		cc.Value = c.OffsetSeq
	}
	accepted := 0
	for page := 0; cc.Value < c.LastSeq && page < 100; page++ {
		batch, e := s.Tang.SyncChannel(ctx, id.TargetID, c.Channel, c.Type, cc.Value+1, c.LastSeq+1, device)
		if e != nil {
			return accepted, e
		}
		sort.SliceStable(batch.Messages, func(i, j int) bool { return batch.Messages[i].Seq < batch.Messages[j].Seq })
		next := cc.Value
		for _, m := range batch.Messages {
			if m.Channel != c.Channel || m.ChannelType != c.Type {
				return accepted, fmt.Errorf("channel sync response mismatch")
			}
			if m.Seq <= cc.Value || m.Seq > c.LastSeq {
				continue
			}
			if m.Seq > next {
				next = m.Seq
			}
			if m.Timestamp < started || m.Deleted != 0 || m.Revoke != 0 || (m.Extra != nil && (m.Extra.Revoke != 0 || m.Extra.MutualDeleted != 0)) {
				continue
			}
			if err = s.hydrateTangRecord(ctx, m); err != nil {
				return accepted, err
			}
			ok, e := s.acceptTang(ctx, id, m)
			if e != nil {
				return accepted, e
			}
			if ok {
				accepted++
			}
		}
		// The deployed server can return more=0 on a full page with later
		// sequence numbers outstanding. Use the conversation watermark instead.
		if next == cc.Value {
			if len(batch.Messages) != 0 {
				return accepted, fmt.Errorf("channel sync cursor did not advance")
			}
			next = c.LastSeq // Empty range consists of deleted/expired messages.
		}
		cc.Value = next
		if err = s.Store.PutCursor(ctx, cc); err != nil {
			return accepted, err
		}
	}
	if cc.Value < c.LastSeq {
		return accepted, fmt.Errorf("channel sync page limit exceeded")
	}
	return accepted, nil
}

func (s *Service) acceptTang(ctx context.Context, id domain.Identity, m adapter.TangMessage) (bool, error) {
	if m.MessageID() == "" {
		return false, fmt.Errorf("sync message ID missing")
	}
	return s.Accept(ctx, proxy.EventEnvelope(id.TargetID, session.Event{NeedReceipt: m.Setting&128 != 0, MessageID: m.MessageID(), MessageSeq: m.Seq, ClientMsgNo: m.ClientNo, FromUID: m.From, ChannelID: m.Channel, ChannelType: m.ChannelType, Timestamp: m.Timestamp, Payload: m.Payload}))
}

func (s *Service) hydrateTangRecord(ctx context.Context, m adapter.TangMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, err := s.Store.FindMessageRecord(ctx, domain.SourceTang, m.MessageID())
	if errors.Is(err, store.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if v.TangSeq != 0 && v.TangClientNo != "" {
		return nil
	}
	v.TangSeq = m.Seq
	if m.ClientNo != "" {
		v.TangClientNo = m.ClientNo
	}
	return s.Store.PutMessageRecord(ctx, v)
}

func (s *Service) syncCommands(ctx context.Context, id domain.Identity, started int64) (int, int64, error) {
	c, err := s.cursor(ctx, "commands:"+id.SourceID, started)
	if err != nil {
		return 0, 0, err
	}
	count := 0
	for page := 0; page < 100; page++ {
		items, e := s.Tang.SyncCommands(ctx, id.TargetID, c.Value)
		if e != nil {
			return count, c.Value, e
		}
		// The deployed command queue can restart its sequence after it has
		// been drained. Its message IDs, not this queue offset, are durable IDs.
		next := int64(0)
		for _, m := range items {
			if m.Seq > next {
				next = m.Seq
			}
			var payload struct {
				Type int `json:"type"`
			}
			if e = json.Unmarshal(m.Payload, &payload); e != nil {
				return count, c.Value, e
			}
			// Do not re-enact old relation commands or import earlier history.
			// Current relation state is obtained through the dedicated sync API.
			if m.Timestamp < started {
				continue
			}
			if m.Deleted != 0 || m.Revoke != 0 {
				continue
			}
			ok, e := s.acceptTang(ctx, id, m)
			if e != nil {
				return count, c.Value, e
			}
			if ok {
				count++
			}
		}
		if len(items) == 0 {
			return count, c.Value, nil
		}
		if next <= 0 {
			return count, c.Value, fmt.Errorf("command sync sequence missing")
		}
		c.Value = next
		if e = s.Store.PutCursor(ctx, c); e != nil {
			return count, c.Value, e
		}
		if next > 0 {
			if e = s.Tang.AckCommands(ctx, id.TargetID, next); e != nil {
				return count, c.Value, e
			}
		}
		if len(items) < 100 {
			return count, c.Value, nil
		}
	}
	return count, c.Value, fmt.Errorf("command sync page limit exceeded")
}
