package business

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
)

type groupRoleNotice struct {
	Group   string `json:"group"`
	Actor   string `json:"actor"`
	Member  string `json:"member"`
	Name    string `json:"name"`
	Role    int    `json:"role"`
	Version int64  `json:"version"`
}

// Native manager endpoints emit memberUpdate, but no visible system message.
// Queue before advancing the group snapshot so a failed snapshot write cannot
// lose the notice. Existing inbox/outbox keys make that ordering retry-safe.
func (s *Service) queueGroupRoleNotices(ctx context.Context, actor domain.Identity, old, current domain.GroupState) error {
	if !old.MirrorReady || current.Status != 1 {
		return nil
	}
	previous := map[string]domain.GroupMember{}
	for _, m := range old.Members {
		previous[m.UID] = m
	}
	for _, m := range current.Members {
		before, ok := previous[m.UID]
		if !ok || before.Deleted != 0 || m.Deleted != 0 || before.Role == m.Role {
			continue
		}
		if !((before.Role == 0 && m.Role == 2) || (before.Role == 2 && m.Role == 0)) {
			continue
		}
		key := "bridge-group-role:" + domain.StableID(current.Number, m.UID, strconv.FormatInt(m.Version, 10), strconv.Itoa(before.Role), strconv.Itoa(m.Role))
		sender := actor.TargetID
		// Prefer the actual group owner when it is a managed Huawei identity.
		// Never use a service administrator or a native user's borrowed identity.
		for _, owner := range current.Members {
			if owner.Role == 1 && owner.Deleted == 0 && owner.Huawei {
				sender = owner.UID
				break
			}
		}
		payload, err := json.Marshal(groupRoleNotice{current.Number, sender, m.UID, m.Name, m.Role, m.Version})
		if err != nil {
			return err
		}
		e := domain.EventEnvelope{Source: domain.SourceTencent, Command: "Bridge.GroupRoleNotice", EventID: key, IdempotencyKey: key, Payload: payload, OccurredAt: time.Now().Unix()}
		b := &domain.BusinessEvent{Source: e.Source, Command: e.Command, EventID: key, Payload: payload}
		job := domain.OutboxJob{ID: domain.StableID("outbox", key), EventID: key, Target: domain.SourceTang, Status: "pending", CreatedAt: time.Now().UnixNano(), NextAttemptAt: time.Now().Unix(), Message: domain.MessageEvent{Source: e.Source, SourceMessageID: key, Business: b}}
		if _, err = s.Store.AcceptMessage(ctx, e, job); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) sendGroupRoleNotice(ctx context.Context, e domain.BusinessEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var n groupRoleNotice
	if err := json.Unmarshal(e.Payload, &n); err != nil {
		return err
	}
	if !strings.HasPrefix(e.EventID, "bridge-group-role:") || n.Group == "" || n.Actor == "" || n.Member == "" || (n.Role != 0 && n.Role != 2) {
		return fmt.Errorf("invalid role notice")
	}
	if _, err := s.Store.GetCursor(ctx, e.EventID); err == nil {
		return nil
	} else if !errors.Is(err, store.ErrNotFound) {
		return err
	}
	g, err := s.Store.GetGroupState(ctx, n.Group)
	if err != nil {
		return err
	}
	if g.Status != 1 {
		return nil
	}
	found := false
	for _, m := range g.Members {
		if m.UID != n.Member {
			continue
		}
		if m.Version < n.Version {
			return fmt.Errorf("role snapshot not committed")
		}
		if m.Deleted != 0 || m.Role != n.Role {
			return nil
		} // Superseded before delivery.
		found = true
	}
	if !found {
		return nil
	}
	actor, err := s.Store.FindIdentityByTarget(ctx, domain.SourceTang, n.Actor)
	if err != nil {
		return err
	}
	if _, err = s.owner(ctx, actor.SourceID); err != nil {
		return err
	}
	name := strings.TrimSpace(n.Name)
	if name == "" {
		name = "该成员"
	}
	text := name + "已被设置为群管理员"
	if n.Role == 0 {
		text = name + "的群管理员身份已取消"
	}
	msg := domain.MessageEvent{Source: domain.SourceTencent, SourceMessageID: e.EventID, ConversationType: domain.ConversationGroup, ConversationID: n.Group, Content: text, Parts: []domain.MessagePart{{Kind: "tip", Text: text}}, BridgeMarker: e.EventID}
	id, err := s.Tang.SendText(ctx, msg, *actor, domain.Identity{TargetID: n.Group}, e.EventID)
	if err != nil {
		return err
	}
	return s.Store.PutCursor(ctx, domain.SyncCursor{Key: e.EventID, Value: time.Now().Unix(), Aux: id})
}
