package business

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
)

func TestNativeChannelUpdateRefreshesProfileDespiteUnchangedFriendSnapshot(t *testing.T) {
	s, _, owner, peer := fixture(t)
	ctx := context.Background()
	id, _ := s.owner(ctx, owner)
	original := "Original Name"
	_ = s.Store.PutFriendState(ctx, domain.FriendState{Owner: owner, Peer: peer, Status: "active", Name: original})
	projected := "Temporary Name"
	explicitWrites := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/v1/friend/sync":
			_ = json.NewEncoder(w).Encode([]any{map[string]any{"uid": peer, "name": original, "version": 1}})
		case r.URL.Path == "/v1/users/"+peer:
			if r.Header.Get("token") != "owner-fixture-token" {
				t.Error("wrong profile actor")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"uid": peer, "name": original})
		case strings.HasSuffix(r.URL.Path, "portrait_set"):
			var b struct {
				Account string `json:"From_Account"`
				Items   []struct {
					Tag   string
					Value string
				} `json:"ProfileItem"`
			}
			_ = json.NewDecoder(r.Body).Decode(&b)
			if b.Account != peer {
				t.Errorf("unexpected profile account %s", b.Account)
			}
			for _, item := range b.Items {
				if item.Tag == "Tag_Profile_IM_Nick" {
					projected = item.Value
				}
			}
			explicitWrites++
			fmt.Fprint(w, `{"ActionStatus":"OK","ErrorCode":0}`)
		default:
			// Importing an existing account must not substitute for portrait_set.
			fmt.Fprint(w, `{"ActionStatus":"OK","ErrorCode":0}`)
		}
	}))
	defer server.Close()
	s.Tang.BaseURL = server.URL
	s.Tencent.BaseURL = server.URL
	raw, err := os.ReadFile("../../test/contract/profile-channel-update.anonymized.json")
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.ReplaceAll(strings.ReplaceAll(string(raw), "fixture-shadow", id.TargetID), "fixture-peer", peer))
	event, err := domain.ParseBusinessEvent(domain.EventEnvelope{Source: domain.SourceTang, Command: "msg.notify", EventID: "profile-restore", Payload: raw})
	if err != nil || event == nil {
		t.Fatalf("parse %v", err)
	}
	s.peerSettings.Store(id.TargetID+":"+peer, "stale receipt setting")
	s.peerSettings.Store(id.TargetID+":other-peer", "unrelated setting")
	if err = s.Handle(ctx, *event); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.peerSettings.Load(id.TargetID + ":" + peer); ok {
		t.Fatal("changed peer receipt setting remained cached")
	}
	if _, ok := s.peerSettings.Load(id.TargetID + ":other-peer"); !ok {
		t.Fatal("unrelated peer setting invalidated")
	}
	if projected != original || explicitWrites != 1 {
		t.Fatalf("profile=%s writes=%d", projected, explicitWrites)
	}
}
