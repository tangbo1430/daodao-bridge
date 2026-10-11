package business

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFriendExpiryErasesSecretsAndReapplyStillWorks(t *testing.T) {
	s, _, owner, peer := fixture(t)
	ctx := context.Background()
	now := time.Now().Unix()
	s.Store.PutFriendState(ctx, domain.FriendState{Owner: owner, Peer: peer, Status: "pending_in", ExpiresAt: now - 1, TokenCipher: "old", VercodeCipher: "old"})
	if err := s.expireFriendRequests(ctx, owner, now); err != nil {
		t.Fatal(err)
	}
	f, _ := s.Store.GetFriendState(ctx, owner, peer)
	if f.Status != "expired_in" || f.TokenCipher != "" || f.VercodeCipher != "" {
		t.Fatal(f)
	}
	f, err := s.Friend(ctx, owner, peer, "accept", "", "")
	if err != nil || f.Status != "pending_out" || f.ExpiresAt < now+29*86400 {
		t.Fatal(f, err)
	}
}

func TestNativeGroupNoticeContentDedupAndPrivateVisibility(t *testing.T) {
	s, f, owner, _ := groupService(t)
	ctx := context.Background()
	id, _ := s.owner(ctx, owner)
	e := domain.BusinessEvent{Source: domain.SourceTang, Command: "groupSystem", Owner: id.TargetID, ChannelID: "fixture-group", EventID: "copy-a", NativeMessageID: "native-notice-fixture", Payload: json.RawMessage(`{"type":1002,"content":"Fixture邀请{0}加入群聊","extra":[{"name":"Peer"}]}`)}
	if err := s.Handle(ctx, e); err != nil {
		t.Fatal(err)
	}
	e.EventID = "copy-b"
	if err := s.Handle(ctx, e); err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, c := range f.calls {
		if c.Path == "/v4/group_open_http_svc/send_group_msg" {
			count++
			b, _ := json.Marshal(c.Body)
			if !strings.Contains(string(b), "Fixture邀请Peer加入群聊") || c.Body["OfflinePushInfo"].(map[string]any)["PushFlag"] != float64(1) {
				t.Fatal("incorrect native notice")
			}
		}
	}
	if count != 1 {
		t.Fatal("group notice duplicate", count)
	}
	before := len(f.calls)
	e.NativeMessageID = "invitation-fixture"
	e.Payload = json.RawMessage(`{"type":1009,"visibles":["someone-else"]}`)
	if err := s.Handle(ctx, e); err != nil {
		t.Fatal(err)
	}
	for _, c := range f.calls[before:] {
		if strings.Contains(c.Path, "sendmsg") || strings.Contains(c.Path, "send_group_msg") {
			t.Fatal("private invitation leaked")
		}
	}
	before = len(f.calls)
	e.NativeMessageID = "unknown-fixture"
	e.Payload = json.RawMessage(`{"type":1004}`)
	if err := s.Handle(ctx, e); err != nil {
		t.Fatal(err)
	}
	for _, c := range f.calls[before:] {
		if strings.Contains(c.Path, "sendmsg") {
			t.Fatal("unknown numeric event invented removal")
		}
	}
}

func TestGroupUnreadDailyOnlyActiveMembershipAndNoReceipt(t *testing.T) {
	s, f, owner, peer := groupService(t)
	ctx := context.Background()
	id, _ := s.owner(ctx, owner)
	if _, err := s.Group(ctx, "sync", GroupRequest{Owner: owner, GroupID: "dd_fixture-group"}); err != nil {
		t.Fatal(err)
	}
	s.Store.PutGroupState(ctx, domain.GroupState{Number: "other", MirrorReady: true, Status: 1, Members: []domain.GroupMember{{UID: peer}}})
	start := len(f.calls)
	for i := 0; i < 2; i++ {
		if err := s.clearGroupUnreadDaily(ctx, id); err != nil {
			t.Fatal(err)
		}
	}
	clears := 0
	for _, c := range f.calls[start:] {
		if c.Path == "/v1/coversation/clearUnread" {
			clears++
			if c.Token != "owner-fixture-token" || c.Body["channel_type"] != float64(2) {
				t.Fatal(c)
			}
		}
		if c.Path == "/v1/message/readed" {
			t.Fatal("group read receipt emitted")
		}
	}
	if clears != 1 {
		t.Fatalf("clear calls=%d", clears)
	}
}

func TestFriendReconcileSkipsNativeTencentAndPendingRelationships(t *testing.T) {
	s, _, owner, peer := fixture(t)
	ctx := context.Background()
	id, _ := s.owner(ctx, owner)
	for _, uid := range []string{"extra-shadow", "pending-shadow"} {
		s.Store.UpsertIdentity(ctx, domain.Identity{Source: domain.SourceTang, SourceID: uid, Target: domain.SourceTencent, TargetID: uid, EntityType: "user", Shadow: true})
	}
	s.Store.PutFriendState(ctx, domain.FriendState{Owner: owner, Peer: "pending-shadow", Status: "pending_in"})
	adds, deletes := []string{}, []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		json.NewDecoder(r.Body).Decode(&b)
		switch r.URL.Path {
		case "/v1/friend/sync":
			fmt.Fprintf(w, `[{"uid":%q,"version":1}]`, peer)
		case "/v4/sns/friend_get":
			if b["From_Account"] == owner {
				fmt.Fprint(w, `{"ActionStatus":"OK","CompleteFlag":1,"UserDataItem":[{"To_Account":"native-tencent"},{"To_Account":"extra-shadow"},{"To_Account":"pending-shadow"}]}`)
			} else {
				fmt.Fprint(w, `{"ActionStatus":"OK","CompleteFlag":1,"UserDataItem":[]}`)
			}
		case "/v4/sns/friend_add":
			adds = append(adds, b["AddFriendItem"].([]any)[0].(map[string]any)["To_Account"].(string))
			fmt.Fprint(w, `{"ActionStatus":"OK","ResultItem":[{"ResultCode":0}]}`)
		case "/v4/sns/friend_delete":
			deletes = append(deletes, b["To_Account"].([]any)[0].(string))
			fmt.Fprint(w, `{"ActionStatus":"OK","ResultItem":[{"ResultCode":0}]}`)
		default:
			t.Error(r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	s.Tang.BaseURL = server.URL
	s.Tencent.BaseURL = server.URL
	if err := s.ReconcileFriends(ctx, id); err != nil {
		t.Fatal(err)
	}
	if len(adds) != 1 || adds[0] != peer || len(deletes) != 1 || deletes[0] != "extra-shadow" {
		t.Fatalf("adds=%v deletes=%v", adds, deletes)
	}
}

func TestMemberRoleCallbackUsesModifiedAccountAndActor(t *testing.T) {
	s, f, owner, peer := groupService(t)
	ctx := context.Background()
	raw, _ := json.Marshal(map[string]any{"GroupId": "dd_fixture-group", "Operator_Account": owner, "Modified_Account": peer, "Role": "Admin"})
	if err := s.Handle(ctx, domain.BusinessEvent{Source: domain.SourceTencent, Command: "Group.CallbackAfterMemberFieldChanged", Payload: raw}); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range f.calls {
		if c.Path == "/v1/groups/fixture-group/managers" {
			found = true
			if c.Token != "owner-fixture-token" || c.Body["members"].([]any)[0] != peer {
				t.Fatal(c)
			}
		}
	}
	if !found {
		t.Fatal("role change not forwarded")
	}
}

func TestDeletingOwnerTransfersMirrorBeforeRemoval(t *testing.T) {
	s, f, owner, peer := groupService(t)
	ctx := context.Background()
	if _, err := s.Group(ctx, "sync", GroupRequest{Owner: owner, GroupID: "dd_fixture-group"}); err != nil {
		t.Fatal(err)
	}
	// Snapshot returned after native exit: the surviving member became owner.
	f.members = []domain.GroupMember{{UID: peer, Role: 1, Name: "new owner", Version: 3}}
	start := len(f.calls)
	if err := s.DeleteAccount(ctx, owner); err != nil {
		t.Fatal(err)
	}
	transfer, remove := -1, -1
	for i, c := range f.calls[start:] {
		if c.Path == "/v4/group_open_http_svc/change_group_owner" {
			transfer = i
			if c.Body["NewOwner_Account"] != peer {
				t.Fatal(c)
			}
		}
		if c.Path == "/v4/group_open_http_svc/delete_group_member" {
			remove = i
		}
	}
	if transfer < 0 || remove <= transfer {
		t.Fatalf("transfer=%d removal=%d", transfer, remove)
	}
}
