package business

import (
	"context"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"strings"
	"testing"
)

func TestCardProofIsEncryptedAndOwnerMappingPreserved(t *testing.T) {
	s, _, owner, peer := fixture(t)
	ctx := context.Background()
	id, _ := s.owner(ctx, owner)
	e := domain.MessageEvent{Source: domain.SourceTang, FromAccount: peer, ToAccount: id.TargetID, Content: "[名片]", Parts: []domain.MessagePart{{Kind: "card", UserID: peer, Name: "Fixture", Vercode: "fixture-card-proof"}}}
	got, err := s.prepareMetadata(ctx, e, domain.Identity{TargetID: peer}, domain.Identity{TargetID: owner})
	if err != nil {
		t.Fatal(err)
	}
	if got.Parts[0].Vercode != "" || got.Parts[0].UserID != peer {
		t.Fatalf("card=%+v", got.Parts[0])
	}
	state, _ := s.Store.GetFriendState(ctx, owner, peer)
	plain, err := s.Box.Decrypt(state.VercodeCipher)
	if err != nil || plain != "fixture-card-proof" || strings.Contains(state.VercodeCipher, "fixture-card-proof") {
		t.Fatal("card proof not safely cached")
	}
	if e.Parts[0].Vercode != "fixture-card-proof" {
		t.Fatal("original Outbox payload mutated")
	}
	e.Parts[0].UserID = id.TargetID
	got, err = s.prepareMetadata(ctx, e, domain.Identity{}, domain.Identity{TargetID: owner})
	if err != nil || got.Parts[0].UserID != owner {
		t.Fatalf("owner card=%+v err=%v", got, err)
	}
}
func TestGroupMentionFallbackDoesNotGuessDuplicateNames(t *testing.T) {
	s, _, owner, peer := fixture(t)
	ctx := context.Background()
	id, _ := s.owner(ctx, owner)
	_ = s.Store.PutGroupState(ctx, domain.GroupState{Number: "fixture-group", Members: []domain.GroupMember{{UID: peer, TencentID: peer, Name: "Alex"}, {UID: "same-name", TencentID: "same-name", Name: "Alex"}, {UID: id.TargetID, TencentID: owner, Name: "Owner"}}})
	e := domain.MessageEvent{Source: domain.SourceTencent, ConversationType: domain.ConversationGroup, Content: "@Alex @Owner @所有人 hello"}
	got, err := s.prepareMetadata(ctx, e, id, domain.Identity{TargetID: "fixture-group"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Mention == nil || got.Mention.All != 1 || len(got.Mention.UIDs) != 1 || got.Mention.UIDs[0] != id.TargetID {
		t.Fatalf("mention=%+v", got.Mention)
	}
	if containsMention("@Alexandra hello", "Alex") {
		t.Fatal("partial nickname matched")
	}
}
func TestReplyChecksConversationAndUsesNativeSequence(t *testing.T) {
	s, _, owner, peer := fixture(t)
	ctx := context.Background()
	id, _ := s.owner(ctx, owner)
	_ = s.Store.PutMessageRecord(ctx, domain.MessageRecord{Key: "record", TencentKey: "fixture-key", TencentFrom: owner, TencentTo: peer, TangID: "1234567890123456789", TangSeq: 8, TangFrom: id.TargetID, TangTo: peer, ChannelType: 1})
	e := domain.MessageEvent{Source: domain.SourceTencent, FromAccount: owner, ToAccount: peer, Content: "reply", Reply: &domain.ReplyReference{ID: "fixture-key", Summary: "quoted"}}
	got, err := s.prepareMetadata(ctx, e, id, domain.Identity{TargetID: peer})
	if err != nil || got.Reply == nil || got.Reply.ID != "1234567890123456789" || got.Reply.Seq != 8 {
		t.Fatalf("reply=%+v err=%v", got, err)
	}
	e.ToAccount = "other-conversation"
	got, err = s.prepareMetadata(ctx, e, id, domain.Identity{})
	if err != nil || got.Reply != nil || !strings.Contains(got.Content, "quoted") {
		t.Fatalf("cross-chat reference=%+v err=%v", got, err)
	}
}
func TestFullMirrorKeepsOwnerAndUsesNamedFallback(t *testing.T) {
	s, f, owner, peer := groupService(t)
	ctx := context.Background()
	id, _ := s.owner(ctx, owner)
	f.capacity = 1
	g, err := s.syncGroup(ctx, id, "fixture-group", peer)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range g.Members {
		if m.Huawei && m.TencentID != owner {
			t.Fatal("Huawei member evicted")
		}
		if m.UID == peer && m.TencentID != "" {
			t.Fatal("shadow exceeded capacity")
		}
	}
	e := domain.MessageEvent{Source: domain.SourceTang, ConversationType: domain.ConversationGroup, ConversationID: g.Number, FromAccount: peer, Content: "hello"}
	got, err := s.prepareMetadata(ctx, e, domain.Identity{TargetID: peer}, domain.Identity{TargetID: g.TencentID})
	if err != nil || got.MirrorSender != s.Tencent.AdminAccount || !strings.HasPrefix(got.Content, "[Peer]") {
		t.Fatalf("fallback=%+v err=%v", got, err)
	}
}
func TestRoamingReadDoesNotClearOtherUnreadMessages(t *testing.T) {
	s, calls, owner, peer := fixture(t)
	ctx := context.Background()
	id, _ := s.owner(ctx, owner)
	_ = s.Store.PutMessageRecord(ctx, domain.MessageRecord{Key: "receipt", Source: domain.SourceTang, TencentKey: "fixture-key", TencentFrom: peer, TencentTo: owner, TangID: "123", TangFrom: peer, TangTo: id.TargetID, ChannelType: 1, NeedReceipt: true})
	if err := s.syncRoamingRead(ctx, owner, peer, "fixture-key"); err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 1 || (*calls)[0].Path != "/v1/message/readed" {
		t.Fatalf("unexpected read calls %+v", *calls)
	}
	if err := s.syncRoamingRead(ctx, owner, peer, "fixture-key"); err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 1 {
		t.Fatal("duplicate read receipt")
	}
}
