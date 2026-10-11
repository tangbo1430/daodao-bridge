package business

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
	"strings"
	"time"
)

func (s *Service) messageChannel(ctx context.Context, owner, channel string, ct int) (domain.Identity, error) {
	id, err := s.owner(ctx, owner)
	if err != nil {
		return id, err
	}
	if ct == 1 {
		peer, e := s.Store.FindIdentity(ctx, domain.SourceTang, channel)
		if e != nil || peer.SourceID != peer.TargetID {
			return id, fail(404, "联系人资料尚未就绪，请稍后重试")
		}
		return id, nil
	}
	if ct != 2 {
		return id, fail(400, "频道类型无效")
	}
	g, e := s.Store.GetGroupState(ctx, channel)
	if e != nil {
		return id, fail(404, "群资料尚未就绪，请稍后重试")
	}
	for _, m := range g.Members {
		if m.UID == id.TargetID && m.Deleted == 0 {
			return id, nil
		}
	}
	return id, fail(403, "当前账号不在群中")
}

func (s *Service) Recall(ctx context.Context, owner, key string) error {
	return s.recall(ctx, owner, key, false)
}
func (s *Service) RecallBoth(ctx context.Context, owner, key string) error {
	return s.recall(ctx, owner, key, true)
}
func (s *Service) recall(ctx context.Context, owner, key string, mirrorTencent bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, err := s.owner(ctx, owner)
	if err != nil {
		return err
	}
	rows, err := s.Store.FindMessageRecords(ctx, domain.SourceTencent, key)
	if errors.Is(err, store.ErrNotFound) {
		return fail(404, "撤回消息映射不存在")
	}
	if err != nil {
		return err
	}
	for _, v := range rows {
		if v.ChannelType == 1 && v.TencentFrom != owner {
			return fail(403, "只能撤回自己发送的单聊消息")
		}
		if v.Revoked {
			continue
		}
		channel := v.TangTo
		if v.ChannelType == 1 && channel == id.TargetID {
			channel = v.TangFrom
		}
		if _, err = s.messageChannel(ctx, owner, channel, v.ChannelType); err != nil {
			return err
		}
		if err = s.Tang.RecallMessage(ctx, id.TargetID, channel, v.ChannelType, v.TangID, v.TangClientNo); err != nil {
			return err
		}
		if mirrorTencent {
			if err = s.Tencent.RecallMessage(ctx, v); err != nil {
				return err
			}
		}
		v.Revoked = true
		if err = s.Store.PutMessageRecord(ctx, v); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Read(ctx context.Context, owner, peer string, before int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, err := s.messageChannel(ctx, owner, peer, 1)
	if err != nil {
		return err
	}
	if before <= 0 {
		return fail(400, "已读时间无效")
	}
	for page := 0; page < 100; page++ {
		rows, e := s.Store.ListMessageRecords(ctx, owner, peer, before)
		if e != nil {
			return e
		}
		ids := []string{}
		for _, v := range rows {
			if v.Source == domain.SourceTang && v.NeedReceipt {
				ids = append(ids, v.TangID)
			}
		}
		if err = s.Tang.ReadMessages(ctx, id.TargetID, peer, ids); err != nil {
			return err
		}
		for _, v := range rows {
			v.ReadSynced = true
			if err = s.Store.PutMessageRecord(ctx, v); err != nil {
				return err
			}
		}
		if len(rows) < 100 {
			return s.Tang.ClearUnread(ctx, id.TargetID, peer, 1)
		}
	}
	return fmt.Errorf("read synchronization page limit reached")
}

func (s *Service) SyncMessageExtras(ctx context.Context, owner, channel string, ct int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, err := s.messageChannel(ctx, owner, channel, ct)
	if err != nil {
		return 0, err
	}
	key := "extra:" + owner + ":" + fmt.Sprint(ct) + ":" + channel
	cursor, err := s.Store.GetCursor(ctx, key)
	if errors.Is(err, store.ErrNotFound) {
		cursor = domain.SyncCursor{Key: key, StartedAt: time.Now().Unix()}
	} else if err != nil {
		return 0, err
	}
	processed := 0
	for page := 0; page < 100; page++ {
		extra, e := s.Tang.MessageExtras(ctx, id.TargetID, channel, ct, cursor.Value)
		if e != nil {
			return processed, e
		}
		max := cursor.Value
		for _, change := range extra {
			if change.Version > max {
				max = change.Version
			}
			v, e := s.Store.FindMessageRecord(ctx, domain.SourceTang, change.MessageID())
			if errors.Is(e, store.ErrNotFound) {
				continue
			}
			if e != nil {
				return processed, e
			}
			if ct != v.ChannelType || ct == 2 && channel != v.TangTo || ct == 1 && !((v.TangFrom == id.TargetID && v.TangTo == channel) || (v.TangTo == id.TargetID && v.TangFrom == channel)) {
				return processed, fail(409, "消息频道与映射不一致")
			}
			changed := false
			if (change.Revoke > 0 || change.MutualDeleted > 0) && !v.Revoked {
				if e = s.Tencent.RecallMessage(ctx, v); e != nil {
					return processed, e
				}
				v.Revoked = true
				changed = true
			}
			if ct == 1 && v.Source == domain.SourceTencent && v.NeedReceipt && (!v.ReadSynced || !v.TencentReceiptSynced) && (change.Readed > 0 || change.ReadCount > 0) {
				if e = s.Tencent.MarkRead(ctx, v); e != nil {
					return processed, e
				}
				v.ReadSynced = true
				v.TencentReceiptSynced = true
				changed = true
			}
			if changed {
				if e = s.Store.PutMessageRecord(ctx, v); e != nil {
					return processed, e
				}
				processed++
			}
		}
		if len(extra) > 0 && max <= cursor.Value {
			return processed, fmt.Errorf("message extra cursor did not advance")
		}
		cursor.Value = max
		if err = s.Store.PutCursor(ctx, cursor); err != nil {
			return processed, err
		}
		if len(extra) < 100 {
			return processed, nil
		}
	}
	return processed, fmt.Errorf("message extra page limit reached")
}

func (s *Service) messageStateEvent(ctx context.Context, e domain.BusinessEvent) error {
	if e.Source == domain.SourceTang {
		id, err := s.Store.FindIdentityByTarget(ctx, domain.SourceTang, e.Owner)
		if err != nil {
			return err
		}
		var p struct {
			Param struct {
				Channel string      `json:"channel_id"`
				Type    int         `json:"channel_type"`
				ID      json.Number `json:"message_id"`
			} `json:"param"`
		}
		if err = json.Unmarshal(e.Payload, &p); err != nil {
			return err
		}
		if p.Param.Channel == "" {
			p.Param.Channel = e.ChannelID
			p.Param.Type = e.ChannelType
		}
		if p.Param.Channel == "" || p.Param.Type == 0 {
			// Upgrade-compatible recovery for jobs already persisted before
			// envelope routing was retained. Never guess a recipient/channel.
			original, lookupErr := s.Store.FindTangInboxEvent(ctx, e.EventID, e.Owner)
			if lookupErr == nil {
				recovered, parseErr := domain.ParseBusinessEvent(original)
				if parseErr != nil {
					return parseErr
				}
				if recovered != nil && recovered.Command == e.Command && recovered.Owner == e.Owner {
					if p.Param.Channel == "" {
						p.Param.Channel = recovered.ChannelID
					}
					if p.Param.Type == 0 {
						p.Param.Type = recovered.ChannelType
					}
				}
			} else if !errors.Is(lookupErr, store.ErrNotFound) {
				return lookupErr
			}
		}
		if p.Param.Channel == "" || (p.Param.Type != 1 && p.Param.Type != 2) {
			return fail(400, "消息状态指令缺少有效频道")
		}
		_, err = s.SyncMessageExtras(ctx, id.SourceID, p.Param.Channel, p.Param.Type)
		return err
	}
	var p struct {
		From     string `json:"From_Account"`
		To       string `json:"To_Account"`
		Key      string `json:"MsgKey"`
		Owner    string `json:"Operator_Account"`
		Group    string `json:"GroupId"`
		Reader   string `json:"Report_Account"`
		Peer     string `json:"Peer_Account"`
		ReadTime int64  `json:"LastReadTime"`
		Seqs     []struct {
			Seq int64 `json:"MsgSeq"`
		} `json:"MsgSeqList"`
	}
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return err
	}
	if e.Command == "C2C.CallbackAfterMsgReport" {
		if _, err := s.owner(ctx, p.Reader); err != nil {
			return nil
		}
		if mapped, err := s.Store.FindIdentity(ctx, domain.SourceTang, p.Peer); err != nil || mapped.TargetID != p.Peer {
			return nil
		}
		return s.Read(ctx, p.Reader, p.Peer, p.ReadTime)
	}
	if e.Command == "C2C.CallbackAfterMsgWithDraw" {
		if _, err := s.owner(ctx, p.From); err != nil {
			return nil
		}
		if mapped, err := s.Store.FindIdentity(ctx, domain.SourceTang, p.To); err != nil || mapped.TargetID != p.To {
			return nil
		}
		return s.Recall(ctx, p.From, p.Key)
	}
	if !strings.HasPrefix(p.Group, "dd_") {
		return nil
	}
	for _, seq := range p.Seqs {
		key := fmt.Sprintf("%s:%d", p.Group, seq.Seq)
		v, err := s.Store.FindMessageRecord(ctx, domain.SourceTencent, key)
		if err != nil {
			return err
		}
		if v.Revoked {
			continue
		}
		if p.Owner == s.Tencent.AdminAccount {
			return fail(409, "群管理员撤回需要真实操作人通知")
		}
		if err = s.Recall(ctx, p.Owner, key); err != nil {
			return err
		}
	}
	return nil
}
