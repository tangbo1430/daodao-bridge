package business

import (
	"context"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeleteAccountResumesAfterMirrorFailureAndErasesCredentials(t *testing.T) {
	s, _, owner, _ := fixture(t)
	ctx := context.Background()
	r, _ := s.Accounts.Profile(ctx, owner)
	s.Accounts.Tang = s.Tang
	exits, stops, profiles, removals := 0, 0, 0, 0
	failMirror := true
	s.Accounts.StopProxy = func(context.Context, domain.Identity) error { stops++; return nil }
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if rq.URL.Path[:4] == "/v1/" && rq.Header.Get("token") != "owner-fixture-token" {
			t.Error("wrong deletion actor")
		}
		switch rq.URL.Path {
		case "/v1/conversation/sync":
			fmt.Fprintf(w, `{"uid":%q,"conversations":[{"channel_id":"fixture-group","channel_type":2}]}`, r.Identity.TargetID)
		case "/v1/groups/fixture-group":
			fmt.Fprint(w, `{"quit":0,"status":1}`)
		case "/v1/groups/fixture-group/exit":
			exits++
			fmt.Fprint(w, `{}`)
		case "/v4/group_open_http_svc/get_group_member_info":
			if failMirror {
				w.WriteHeader(502)
				return
			}
			fmt.Fprintf(w, `{"ActionStatus":"OK","MemberList":[{"Member_Account":%q}]}`, owner)
		case "/v4/group_open_http_svc/delete_group_member":
			removals++
			fmt.Fprint(w, `{"ActionStatus":"OK"}`)
		case "/v1/friend/sync":
			fmt.Fprint(w, `[{"uid":"fixture-friend","version":2},{"uid":"fileHelper","version":3}]`)
		case "/v1/friends/fixture-friend":
			fmt.Fprint(w, `{}`)
		case "/v1/user/current":
			profiles++
			fmt.Fprint(w, `{}`)
		default:
			t.Errorf("unexpected deletion API %s", rq.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	s.Tang.BaseURL = server.URL
	s.Tencent.BaseURL = server.URL
	_ = s.Store.PutGroupState(ctx, domain.GroupState{Number: "fixture-group", TencentID: "dd_fixture-group", Status: 1, MirrorReady: true, Members: []domain.GroupMember{{UID: r.Identity.TargetID}}})
	_ = s.Store.PutFriendState(ctx, domain.FriendState{Owner: owner, Peer: "fixture-friend", Status: "active", TokenCipher: "stored", VercodeCipher: "stored"})
	if err := s.DeleteAccount(ctx, owner); err == nil {
		t.Fatal("mirror failure concealed")
	}
	state, _ := s.Store.GetCursor(ctx, "account:"+owner)
	if state.Aux != "deleting" || exits != 1 || stops != 1 {
		t.Fatalf("state=%+v exits=%d stops=%d", state, exits, stops)
	}
	if _, ok := s.Tang.UserTokens.Load(r.Identity.TargetID); !ok {
		t.Fatal("cleanup lost its user token")
	}
	failMirror = false
	if err := s.DeleteAccount(ctx, owner); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteAccount(ctx, owner); err != nil {
		t.Fatal(err)
	}
	state, _ = s.Store.GetCursor(ctx, "account:"+owner)
	if state.Aux != "deleted" || exits != 1 || profiles != 1 || removals != 1 {
		t.Fatalf("state=%+v exits=%d profiles=%d removals=%d", state, exits, profiles, removals)
	}
	if _, ok := s.Tang.UserTokens.Load(r.Identity.TargetID); ok {
		t.Fatal("deleted user token retained")
	}
	f, _ := s.Store.GetFriendState(ctx, owner, "fixture-friend")
	if f.TokenCipher != "" || f.VercodeCipher != "" || f.Status != "deleted" {
		t.Fatalf("friend credentials retained: %+v", f)
	}
	if err := s.Accounts.Cancel(ctx, owner); err == nil {
		t.Fatal("deletion reverted to cancelled")
	}
}
