package business

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
)

func TestFriendCommandsPreserveAvatarAndUsePeerName(t *testing.T) {
	for _, command := range []string{"friendRequest", "friendAccept"} {
		for _, accepts := range []string{"owner", "peer"} {
			t.Run(command+"/"+accepts, func(t *testing.T) {
				s, _, owner, peer := fixture(t)
				ctx := context.Background()
				actor, _ := s.owner(ctx, owner)
				mapped, _ := s.Store.FindIdentity(ctx, domain.SourceTang, peer)
				mapped.DisplayName = "Peer Name"
				mapped.AvatarURL = "https://example.test/retained.png"
				_ = s.Store.UpsertIdentity(ctx, *mapped)
				_ = s.Store.PutFriendState(ctx, domain.FriendState{Owner: owner, Peer: peer, Status: "deleted", Name: "Peer Name", Avatar: mapped.AvatarURL})
				imports := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.HasSuffix(r.URL.Path, "portrait_get") {
						fmt.Fprintf(w, `{"ActionStatus":"OK","ErrorCode":0,"UserProfileItem":[{"To_Account":%q,"ResultCode":0,"ProfileItem":[{"Tag":"Tag_Profile_IM_AllowType","Value":"AllowType_Type_AllowAny"}]}]}`, owner)
						return
					}
					if strings.HasSuffix(r.URL.Path, "account_import") {
						imports++
						var body map[string]any
						_ = json.NewDecoder(r.Body).Decode(&body)
						if body["Nick"] != "Peer Name" || body["FaceUrl"] != mapped.AvatarURL {
							t.Errorf("partial event corrupted imported profile: %+v", body)
						}
					}
					fmt.Fprint(w, `{"ActionStatus":"OK","ErrorCode":0,"ResultItem":[{"ResultCode":0}]}`)
				}))
				defer server.Close()
				s.Tencent.BaseURL = server.URL
				from, to, name := peer, actor.TargetID, "Peer Name"
				if accepts == "owner" {
					from, to, name = actor.TargetID, peer, "Owner Name"
				}
				payload, _ := json.Marshal(map[string]any{"param": map[string]string{"from_uid": from, "to_uid": to, "from_name": name, "apply_uid": peer, "apply_name": "Peer Name", "token": "fixture-private"}})
				if command == "friendRequest" {
					payload, _ = json.Marshal(map[string]any{"param": map[string]string{"apply_uid": peer, "to_uid": actor.TargetID, "apply_name": "Peer Name", "token": "fixture-private"}})
				}
				event := domain.BusinessEvent{Source: domain.SourceTang, Command: command, Owner: actor.TargetID, EventID: "reapply", Payload: payload}
				if err := s.Handle(ctx, event); err != nil {
					t.Fatal(err)
				}
				result, _ := s.Store.FindIdentity(ctx, domain.SourceTang, peer)
				if result.DisplayName != "Peer Name" || result.AvatarURL != mapped.AvatarURL || imports != 1 {
					t.Fatalf("profile not preserved: %+v imports=%d", result, imports)
				}
			})
		}
	}
}
