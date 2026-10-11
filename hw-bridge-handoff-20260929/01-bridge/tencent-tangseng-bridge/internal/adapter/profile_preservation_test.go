package adapter

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

func TestPartialAccountImportDoesNotClearProfile(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		fmt.Fprint(w, `{"ActionStatus":"OK","ErrorCode":0}`)
	}))
	defer server.Close()
	c := TencentClient{BaseURL: server.URL, SDKAppID: 1, SecretKey: "fixture-only", AdminAccount: "admin"}
	_, err := c.EnsureUser(context.Background(), domain.Identity{TargetID: "fixture-peer"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := body["FaceUrl"]; ok {
		t.Fatal("empty avatar would overwrite existing profile")
	}
	if _, ok := body["Nick"]; ok {
		t.Fatal("empty nickname would overwrite existing profile")
	}
	_, err = c.EnsureUser(context.Background(), domain.Identity{TargetID: "fixture-peer", AvatarURL: "https://example.test/avatar.png", DisplayName: "Name"})
	if err != nil || body["FaceUrl"] != "https://example.test/avatar.png" || body["Nick"] != "Name" {
		t.Fatalf("profile omitted: %v %v", body, err)
	}
}

func TestDeleteMissingFriendRequiresVerifiedTargetAbsence(t *testing.T) {
	for _, tc := range []struct {
		owner, peer string
		ok          bool
	}{
		{"Imported", "NotImported", true}, {"NotImported", "Imported", false},
		{"Imported", "Imported", false}, {"Imported", "", false},
	} {
		t.Run(tc.owner+"-"+tc.peer, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "friend_delete") {
					fmt.Fprint(w, `{"ActionStatus":"OK","ErrorCode":0,"ResultItem":[{"To_Account":"peer","ResultCode":30003}]}`)
				} else {
					fmt.Fprintf(w, `{"ActionStatus":"OK","ErrorCode":0,"ResultItem":[{"UserID":"owner","ResultCode":0,"AccountStatus":%q},{"UserID":"peer","ResultCode":0,"AccountStatus":%q}]}`, tc.owner, tc.peer)
				}
			}))
			defer server.Close()
			c := TencentClient{BaseURL: server.URL, SDKAppID: 1, SecretKey: "fixture-only", AdminAccount: "admin"}
			if err := c.DeleteFriend(context.Background(), "owner", "peer"); (err == nil) != tc.ok {
				t.Fatalf("result %v", err)
			}
		})
	}
}
