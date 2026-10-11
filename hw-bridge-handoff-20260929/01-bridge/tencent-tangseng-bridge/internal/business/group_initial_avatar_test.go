package business

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/media"
)

func TestInitialAvatarIsCreatedInlineOrUsesRememberedOwner(t *testing.T) {
	for _, delayed := range []bool{false, true} {
		t.Run(map[bool]string{false: "inline", true: "delayed-404"}[delayed], func(t *testing.T) {
			s, f, owner, peer := groupService(t)
			ctx := context.Background()
			// Mirror observer is a Huawei member; native creator is another user.
			f.members[0].Role = 0
			f.members[1].Role = 1
			ready := !delayed
			png, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+aN1cAAAAASUVORK5CYII=")
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "/groups/") && !ready {
					w.WriteHeader(404)
					return
				}
				w.Header().Set("Content-Type", "image/png")
				_, _ = w.Write(png)
			}))
			defer srv.Close()
			s.Media = &media.Service{Store: s.Store, TangBase: srv.URL, PublicBase: "https://example.test/api/bridge/media", Secret: "fixture-only"}
			g, err := s.Group(ctx, "sync", GroupRequest{Owner: owner, GroupID: "dd_fixture-group"})
			if err != nil {
				t.Fatal(err)
			}
			createFound := false
			for _, c := range f.calls {
				if c.Path == "/v4/group_open_http_svc/create_group" {
					createFound = true
					_, hasAvatar := c.Body["FaceUrl"]
					if hasAvatar == delayed {
						t.Fatalf("unexpected create avatar: delayed=%v", delayed)
					}
				}
				if c.Path == "/v4/group_open_http_svc/modify_group_base_info" && c.Body["FaceUrl"] != nil {
					t.Fatal("creation emitted synthetic avatar modification")
				}
			}
			if !createFound {
				t.Fatal("no create")
			}
			f.calls = nil
			ready = true
			for i := 0; i < 3; i++ {
				g, err = s.Group(ctx, "sync", GroupRequest{Owner: owner, GroupID: "dd_fixture-group"})
				if err != nil {
					t.Fatal(err)
				}
			}
			patches := 0
			for _, c := range f.calls {
				if c.Path == "/v4/group_open_http_svc/modify_group_base_info" && c.Body["FaceUrl"] != nil {
					patches++
					if c.Body["From_Account"] != peer {
						t.Fatalf("observer/admin used instead of creation owner: %v", c.Body["From_Account"])
					}
				}
			}
			want := 0
			if delayed {
				want = 1
			}
			if patches != want || g.Avatar == "" {
				t.Fatalf("patches=%d want=%d", patches, want)
			}
		})
	}
}

func TestInitialAvatarFailureDoesNotCreateIncompleteMirror(t *testing.T) {
	s, f, owner, peer := groupService(t)
	// Avoid unrelated profile avatar reads while testing the group image failure.
	ctx := context.Background()
	id, _ := s.Store.FindIdentity(ctx, "tangseng", peer)
	id.AvatarURL = "https://example.test/peer.png"
	_ = s.Store.UpsertIdentity(ctx, *id)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) }))
	defer srv.Close()
	s.Media = &media.Service{Store: s.Store, TangBase: srv.URL, PublicBase: "https://example.test/media", Secret: "fixture-only"}
	if _, err := s.Group(ctx, "sync", GroupRequest{Owner: owner, GroupID: "dd_fixture-group"}); err == nil {
		t.Fatal("image failure hidden")
	}
	for _, c := range f.calls {
		if c.Path == "/v4/group_open_http_svc/create_group" {
			t.Fatal("incomplete mirror created after image error")
		}
	}
}
