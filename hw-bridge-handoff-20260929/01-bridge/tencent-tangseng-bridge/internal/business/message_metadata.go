package business

import (
	"context"
	"errors"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/adapter"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
	"net/http"
	"net/url"
	"strings"
)

func (s *Service) mapMessageUser(ctx context.Context, source domain.Source, uid string) (string, error) {
	if uid == "" {
		return "", nil
	}
	if source == domain.SourceTang {
		hw, err := s.Store.FindIdentityByTarget(ctx, domain.SourceTang, uid)
		if err == nil {
			return hw.SourceID, nil
		}
		if !errors.Is(err, store.ErrNotFound) {
			return "", err
		}
		return uid, nil
	}
	id, err := s.Store.FindIdentity(ctx, domain.SourceTencent, uid)
	if err == nil {
		return id.TargetID, nil
	}
	if !errors.Is(err, store.ErrNotFound) {
		return "", err
	}
	id, err = s.Store.FindIdentity(ctx, domain.SourceTang, uid)
	if err != nil {
		return "", err
	}
	if id.TargetID != uid {
		return "", store.ErrNotFound
	}
	return uid, nil
}

func (s *Service) prepareMetadata(ctx context.Context, e domain.MessageEvent, from, to domain.Identity) (domain.MessageEvent, error) {
	// Copy slices: cached Outbox payloads must not acquire target-specific data.
	e.Parts = append([]domain.MessagePart(nil), e.Parts...)
	if e.ConversationType == domain.ConversationGroup && e.Source == domain.SourceTencent && e.Mention == nil {
		g, err := s.Store.GetGroupState(ctx, to.TargetID)
		if err != nil {
			return e, err
		}
		mention := domain.Mention{}
		if strings.Contains(e.Content, "@所有人") {
			mention.All = 1
		}
		names := map[string][]domain.GroupMember{}
		for _, m := range g.Members {
			if m.Deleted == 0 && m.Name != "" {
				names[m.Name] = append(names[m.Name], m)
			}
		}
		for name, rows := range names {
			if len(rows) == 1 && containsMention(e.Content, name) && rows[0].TencentID != "" {
				mention.UIDs = append(mention.UIDs, rows[0].TencentID)
			}
		}
		if mention.All == 1 || len(mention.UIDs) > 0 {
			e.Mention = &mention
		}
	}
	if e.Mention != nil {
		out := &domain.Mention{All: e.Mention.All}
		seen := map[string]bool{}
		for _, uid := range e.Mention.UIDs {
			mapped, err := s.mapMessageUser(ctx, e.Source, uid)
			if errors.Is(err, store.ErrNotFound) {
				continue
			}
			if err != nil {
				return e, err
			}
			if mapped != "" && !seen[mapped] {
				out.UIDs = append(out.UIDs, mapped)
				seen[mapped] = true
			}
		}
		e.Mention = out
	}
	for i, p := range e.Parts {
		if p.Kind != "card" {
			continue
		}
		original := p.UserID
		mapped, err := s.mapMessageUser(ctx, e.Source, original)
		if errors.Is(err, store.ErrNotFound) || original == "" {
			e.Parts[i] = domain.MessagePart{Kind: "tip", Text: "[名片] " + p.Name + "：该账号尚未开通跨系统聊天"}
			continue
		}
		if err != nil {
			return e, err
		}
		p.UserID = mapped
		if e.Source == domain.SourceTencent {
			actor := from.TargetID
			if hw, lookup := s.Store.FindIdentity(ctx, domain.SourceTencent, original); lookup == nil {
				actor = hw.TargetID
			}
			var profile struct{ Name, Avatar, Vercode string }
			// A missing friend-only vercode must not turn into an administrator call.
			lookup := s.Tang.UserRequest(ctx, actor, http.MethodGet, "/v1/users/"+url.PathEscape(mapped), nil, &profile)
			if lookup == nil {
				p.Vercode = profile.Vercode
				if profile.Name != "" {
					p.Name = profile.Name
				}
			}
			p.Avatar = "/v1/users/" + url.PathEscape(mapped) + "/avatar"
		} else {
			p.Avatar = ""
			if s.Media != nil {
				p.Avatar, err = s.Media.AvatarURL(ctx, "users", original)
				if err != nil {
					return e, err
				}
			}
			if mapped == original {
				identity := domain.Identity{Source: domain.SourceTang, SourceID: original, Target: domain.SourceTencent, TargetID: original, EntityType: "user", DisplayName: p.Name, AvatarURL: p.Avatar, Shadow: true}
				if _, err = s.Tencent.EnsureUser(ctx, identity); err != nil {
					return e, err
				}
				if err = s.Store.UpsertIdentity(ctx, identity); err != nil {
					return e, err
				}
				owners := []string{to.TargetID}
				if e.ConversationType == domain.ConversationGroup {
					owners = nil
					g, lookup := s.Store.GetGroupState(ctx, e.ConversationID)
					if lookup != nil {
						return e, lookup
					}
					for _, m := range g.Members {
						if m.Huawei && m.Deleted == 0 {
							owners = append(owners, m.TencentID)
						}
					}
				}
				for _, owner := range owners {
					if err = s.SaveSearch(ctx, owner, adapter.SearchUser{UID: original, Name: p.Name, Avatar: p.Avatar, Vercode: p.Vercode}); err != nil {
						return e, err
					}
				}
			}
			p.Vercode = "" // Never put Tang friend credentials in a Tencent custom card.
		}
		e.Parts[i] = p
	}
	if e.Reply != nil {
		r := *e.Reply
		e.Reply = &r
		key := r.ID
		if e.Source == domain.SourceTencent && e.ConversationType == domain.ConversationGroup && r.Seq > 0 {
			key = fmt.Sprintf("%s:%d", e.ConversationID, r.Seq)
		}
		if e.Source == domain.SourceTencent && r.ID != "" {
			if alias, err := s.Store.GetCursor(ctx, "message-sdk:"+domain.StableID(r.ID)); err == nil {
				key = alias.Aux
			} else if !errors.Is(err, store.ErrNotFound) {
				return e, err
			}
		}
		v, err := s.Store.FindMessageRecord(ctx, e.Source, key)
		if errors.Is(err, store.ErrNotFound) {
			e.FallbackReply()
		} else if err != nil {
			return e, err
		} else {
			same := false
			if e.Source == domain.SourceTencent {
				same = v.ChannelType == 1 && ((v.TencentFrom == e.FromAccount && v.TencentTo == e.ToAccount) || (v.TencentFrom == e.ToAccount && v.TencentTo == e.FromAccount))
				if e.ConversationType == domain.ConversationGroup {
					same = v.ChannelType == 2 && v.TencentTo == e.ConversationID
				}
			} else {
				same = v.ChannelType == 1 && ((v.TangFrom == e.FromAccount && v.TangTo == e.ToAccount) || (v.TangFrom == e.ToAccount && v.TangTo == e.FromAccount))
				if e.ConversationType == domain.ConversationGroup {
					same = v.ChannelType == 2 && v.TangTo == e.ConversationID
				}
			}
			if !same || v.Revoked {
				e.FallbackReply()
			} else if e.Source == domain.SourceTencent && v.TangSeq > 0 {
				r.ID = v.TangID
				r.Seq = v.TangSeq
				r.From = v.TangFrom
				e.Reply = &r
			} else if e.Source == domain.SourceTang && v.TencentSDKID != "" {
				r.ID = v.TencentSDKID
				r.Seq = v.TencentSeq
				r.From = v.TencentFrom
				r.Time = v.OccurredAt
				r.Type = 1
				e.Reply = &r
			} else {
				e.FallbackReply()
			}
		}
	}
	if e.Source == domain.SourceTang && e.ConversationType == domain.ConversationGroup {
		g, err := s.Store.GetGroupState(ctx, e.ConversationID)
		if err != nil {
			return e, err
		}
		for _, m := range g.Members {
			if m.UID == e.FromAccount && m.Deleted == 0 && m.TencentID == "" {
				e.MirrorSender = s.Tencent.AdminAccount
				e.PrependText("[" + m.Name + "] ")
				break
			}
		}
	}
	return e, nil
}
func containsMention(content, name string) bool {
	needle := "@" + name
	for start := 0; start < len(content); {
		index := strings.Index(content[start:], needle)
		if index < 0 {
			return false
		}
		end := start + index + len(needle)
		if end == len(content) || strings.ContainsRune(" \t\n，,。:：!！", []rune(content[end:])[0]) {
			return true
		}
		start = end
	}
	return false
}

// Called for incoming mirrored history only; never clear unrelated unread
// messages based on a single message's receipt.
func (s *Service) syncRoamingRead(ctx context.Context, owner, peer, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.Store.FindMessageRecords(ctx, domain.SourceTencent, key)
	if errors.Is(err, store.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	id, err := s.owner(ctx, owner)
	if err != nil {
		return err
	}
	for _, v := range rows {
		if v.Source != domain.SourceTang || v.ChannelType != 1 || v.TencentFrom != peer || v.TencentTo != owner || v.ReadSynced || v.Revoked {
			continue
		}
		if v.NeedReceipt {
			if err = s.Tang.ReadMessages(ctx, id.TargetID, peer, []string{v.TangID}); err != nil {
				return err
			}
		}
		v.ReadSynced = true
		if err = s.Store.PutMessageRecord(ctx, v); err != nil {
			return err
		}
	}
	return nil
}
