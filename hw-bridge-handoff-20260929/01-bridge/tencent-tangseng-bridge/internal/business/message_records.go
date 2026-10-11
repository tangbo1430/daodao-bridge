package business

import (
	"context"
	"errors"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
	"strings"
)

func (s *Service) RecordDelivery(ctx context.Context, event domain.MessageEvent, from, to domain.Identity, target, marker string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := domain.MessageRecord{Key: domain.StableID("message-record", string(event.Source), event.SourceMessageID), Source: event.Source, ChannelType: 1, OccurredAt: event.OccurredAt}
	v.NeedReceipt = event.NeedReceipt
	abstract := []rune(event.Content)
	if len(abstract) > 100 {
		abstract = abstract[:100]
	}
	v.Abstract = string(abstract)
	if v.OccurredAt > 100000000000 {
		v.OccurredAt /= 1000
	}
	if event.ConversationType == domain.ConversationGroup {
		v.ChannelType = 2
	}
	if event.Source == domain.SourceTencent {
		v.TencentKey = strings.Split(event.SourceMessageID, "#part")[0]
		v.TencentSDKID = event.SourceSDKID
		v.TencentSeq = event.SourceSeq
		if v.TencentSDKID != "" {
			if err := s.Store.PutCursor(ctx, domain.SyncCursor{Key: "message-sdk:" + domain.StableID(v.TencentSDKID), Aux: v.TencentKey}); err != nil {
				return err
			}
		}
		v.TencentFrom = event.FromAccount
		v.TencentTo = to.SourceID
		v.TangFrom = from.TargetID
		v.TangTo = to.TargetID
		v.TangID = target
		v.TangClientNo = "hwb_" + domain.StableID(marker, event.SourceMessageID)
	} else {
		v.TangID = event.SourceMessageID
		v.TangSeq = event.SourceSeq
		v.TangClientNo = event.ClientMsgNo
		v.TangFrom = event.FromAccount
		v.TangTo = to.SourceID
		v.TencentKey = target
		v.TencentFrom = from.TargetID
		if event.MirrorSender != "" {
			v.TencentFrom = event.MirrorSender
		}
		v.TencentTo = to.TargetID
	}
	old, err := s.Store.FindMessageRecord(ctx, domain.SourceTang, v.TangID)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return err
	}
	if err == nil {
		if v.TencentSDKID == "" {
			v.TencentSDKID = old.TencentSDKID
		}
		if v.TencentSeq == 0 {
			v.TencentSeq = old.TencentSeq
		}
		v.ReadSynced = old.ReadSynced
		v.TencentReceiptSynced = old.TencentReceiptSynced
		v.Revoked = old.Revoked
		if v.TangSeq == 0 {
			v.TangSeq = old.TangSeq
		}
	}
	return s.Store.PutMessageRecord(ctx, v)
}
