package business

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/media"
)

func TestGroupSyncPreservesCanonicalShadowProfile(t *testing.T) {
	s, f, owner, peer := groupService(t)
	ctx := context.Background()
	mapped, _ := s.Store.FindIdentity(ctx, domain.SourceTang, peer)
	mapped.DisplayName = "Canonical Name"
	mapped.AvatarURL = "https://example.test/current-avatar.png"
	_ = s.Store.UpsertIdentity(ctx, *mapped)
	for i := 0; i < 3; i++ {
		if _, err := s.Group(ctx, "sync", GroupRequest{Owner: owner, GroupID: "dd_fixture-group"}); err != nil {
			t.Fatal(err)
		}
	}
	mapped, _ = s.Store.FindIdentity(ctx, domain.SourceTang, peer)
	if mapped.AvatarURL != "https://example.test/current-avatar.png" || mapped.DisplayName != "Canonical Name" {
		t.Fatalf("profile overwritten: %+v", mapped)
	}
	for _, c := range f.calls {
		if strings.HasSuffix(c.Path, "account_import") && (c.Body["FaceUrl"] != mapped.AvatarURL || c.Body["Nick"] != mapped.DisplayName) {
			t.Fatalf("partial group import: %+v", c.Body)
		}
	}
}

func TestFailedRelationDoesNotBlockLaterFriendProfile(t *testing.T) {
	s, _, owner, peer := fixture(t)
	_ = s.Store.UpsertIdentity(context.Background(), domain.Identity{Source: domain.SourceTang, SourceID: "broken-peer", Target: domain.SourceTencent, TargetID: "broken-peer", EntityType: "user", Shadow: true})
	var updated bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/friend/sync"):
			fmt.Fprintf(w, `[{"uid":"broken-peer","is_deleted":1,"version":1},{"uid":%q,"name":"Current Name","avatar":"https://example.test/new.png","version":2}]`, peer)
		case strings.HasSuffix(r.URL.Path, "friend_delete"):
			fmt.Fprint(w, `{"ActionStatus":"OK","ErrorCode":0,"ResultItem":[{"To_Account":"broken-peer","ResultCode":30007}]}`)
		case strings.HasSuffix(r.URL.Path, "portrait_set"):
			updated = true
			fmt.Fprint(w, `{"ActionStatus":"OK","ErrorCode":0}`)
		default:
			fmt.Fprint(w, `{"ActionStatus":"OK","ErrorCode":0}`)
		}
	}))
	defer server.Close()
	s.Tang.BaseURL = server.URL
	s.Tencent.BaseURL = server.URL
	id, _ := s.owner(context.Background(), owner)
	err := s.SyncFriends(context.Background(), id)
	if err == nil || !strings.Contains(err.Error(), "30007") {
		t.Fatalf("failure hidden: %v", err)
	}
	v, _ := s.Store.GetFriendState(context.Background(), owner, peer)
	if !updated || v.Avatar != "https://example.test/new.png" || v.Status != "active" {
		t.Fatalf("active friend blocked: %+v", v)
	}
}

func TestAvatarCommandUpdatesGroupOnlyShadow(t *testing.T) {
	s, _, owner, peer := fixture(t)
	ctx := context.Background()
	id, _ := s.owner(ctx, owner)
	image, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+aN1cAAAAASUVORK5CYII=")
	profileWrites := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/avatar") {
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(image)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if strings.HasPrefix(r.URL.Path, "/v1/users/") {
			if r.Header.Get("token") != "owner-fixture-token" {
				t.Error("wrong profile actor")
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"uid": peer, "name": "New Shadow Name"})
		} else {
			if strings.HasSuffix(r.URL.Path, "portrait_set") {
				profileWrites++
			}
			fmt.Fprint(w, `{"ActionStatus":"OK","ErrorCode":0}`)
		}
	}))
	defer server.Close()
	s.Tang.BaseURL = server.URL
	s.Tencent.BaseURL = server.URL
	s.Media = &media.Service{Store: s.Store, TangBase: server.URL, PublicBase: "https://example.test/api/bridge/media", Secret: "fixture-only"}
	payload, _ := json.Marshal(map[string]any{"param": map[string]string{"uid": peer}})
	if err := s.Handle(ctx, domain.BusinessEvent{Source: domain.SourceTang, Command: "userAvatarUpdate", Owner: id.TargetID, Payload: payload}); err != nil {
		t.Fatal(err)
	}
	mapped, _ := s.Store.FindIdentity(ctx, domain.SourceTang, peer)
	if profileWrites != 1 || !strings.HasPrefix(mapped.AvatarURL, "https://example.test/api/bridge/media?id=") || mapped.DisplayName != "New Shadow Name" {
		t.Fatalf("profile %+v writes %d", mapped, profileWrites)
	}
	friends, _ := s.Store.ListFriendStates(ctx, owner)
	if len(friends) != 0 {
		t.Fatal("profile event unexpectedly created a friend relation")
	}
}

func TestUnmappedDeletedTangUserDoesNotBlockProfileSync(t *testing.T) {
	s, _, owner, _ := fixture(t)
	writes := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/friend/sync") {
			fmt.Fprint(w, `[{"uid":"unmapped-history","is_deleted":1,"is_destroy":1,"version":1}]`)
		} else {
			writes++
			fmt.Fprint(w, `{"ActionStatus":"OK","ErrorCode":0}`)
		}
	}))
	defer server.Close()
	s.Tang.BaseURL = server.URL
	s.Tencent.BaseURL = server.URL
	id, _ := s.owner(context.Background(), owner)
	if err := s.SyncFriends(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if writes != 0 {
		t.Fatal("unmapped historical tombstone mutated Tencent")
	}
	v, _ := s.Store.GetFriendState(context.Background(), owner, "unmapped-history")
	if v.Status != "peer_deleted" {
		t.Fatalf("state %+v", v)
	}
}
