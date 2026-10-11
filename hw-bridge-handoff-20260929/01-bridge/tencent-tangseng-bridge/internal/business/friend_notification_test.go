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
)

func TestIncomingApplicationRespectsTencentAllowPolicy(t *testing.T) {
	for _, mode := range []string{"AllowAny", "NeedConfirm", "DenyAny"} {
		t.Run(mode, func(t *testing.T) {
			s, tangCalls, owner, peer := fixture(t)
			ctx := context.Background()
			r, _ := s.Accounts.Profile(ctx, owner)
			paths := []string{}
			var notified map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				paths = append(paths, r.URL.Path)
				if strings.HasSuffix(r.URL.Path, "portrait_get") {
					fmt.Fprintf(w, `{"ActionStatus":"OK","ErrorCode":0,"UserProfileItem":[{"To_Account":%q,"ResultCode":0,"ProfileItem":[{"Tag":"Tag_Profile_IM_AllowType","Value":%q}]}]}`, owner, "AllowType_Type_"+mode)
					return
				}
				if strings.HasSuffix(r.URL.Path, "sendmsg") {
					_ = json.NewDecoder(r.Body).Decode(&notified)
				}
				fmt.Fprint(w, `{"ActionStatus":"OK","ErrorCode":0,"ResultItem":[{"ResultCode":30539}]}`)
			}))
			defer server.Close()
			s.Tencent.BaseURL = server.URL
			payload, _ := json.Marshal(map[string]any{"param": map[string]string{"apply_uid": peer, "to_uid": r.Identity.TargetID, "apply_name": "Fixture", "remark": "hello", "token": "private-fixture-request"}})
			event := domain.BusinessEvent{Source: domain.SourceTang, Command: "friendRequest", Owner: r.Identity.TargetID, EventID: "fixture-apply", Payload: payload}
			if err := s.Handle(ctx, event); err != nil {
				t.Fatal(err)
			}
			v, _ := s.Store.GetFriendState(ctx, owner, peer)
			want := "pending_in"
			if mode == "DenyAny" {
				want = "refused"
			}
			if v.Status != want {
				t.Fatalf("status %s", v.Status)
			}
			joined := strings.Join(paths, " ")
			if (mode == "NeedConfirm") != strings.Contains(joined, "friend_add") {
				t.Fatalf("unexpected writes: %s", joined)
			}
			if mode == "AllowAny" {
				data, _ := json.Marshal(notified)
				if !strings.Contains(string(data), "bridge_friend_apply") || strings.Contains(string(data), "private-fixture-request") {
					t.Fatalf("invalid card %s", data)
				}
			}
			if mode == "DenyAny" && (len(*tangCalls) != 1 || (*tangCalls)[0].Method != "PUT") {
				t.Fatalf("missing refusal %+v", *tangCalls)
			}
			count := len(paths)
			if err := s.Handle(ctx, event); err != nil {
				t.Fatal(err)
			}
			if len(paths) != count {
				t.Fatal("duplicate request notification")
			}
		})
	}
}

func TestTencentScopeKeepsNativeChatsUnchanged(t *testing.T) {
	s, _, owner, peer := fixture(t)
	ctx := context.Background()
	for _, row := range []struct {
		from, to string
		kind     domain.ConversationType
		allow    bool
	}{
		{owner, peer, domain.ConversationC2C, true}, {owner, "ordinary-user", domain.ConversationC2C, false},
		{peer, owner, domain.ConversationC2C, false}, {owner, "ordinary-group", domain.ConversationGroup, false},
		{owner, "dd_fixture", domain.ConversationGroup, true},
	} {
		got, err := s.TencentConversation(ctx, domain.MessageEvent{FromAccount: row.from, ToAccount: row.to, ConversationID: row.to, ConversationType: row.kind})
		if err != nil || got != row.allow {
			t.Fatalf("%+v: %t %v", row, got, err)
		}
	}
}
