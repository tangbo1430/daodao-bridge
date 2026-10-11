package business

import (
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/media"
)

func TestAvatarRefreshAfterUploadRetriesWithoutReupload(t *testing.T) {
	s, _, owner, _ := fixture(t)
	ctx := context.Background()
	id, _ := s.owner(ctx, owner)
	actions := []string{}
	rejectRefresh := true
	shade := uint8(20)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v1/users/"+id.TargetID {
			_ = json.NewEncoder(w).Encode(map[string]any{"uid": id.TargetID, "is_upload_avatar": 1})
			return
		}
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "image/png")
			im := image.NewRGBA(image.Rect(0, 0, 2, 2))
			im.Set(0, 0, color.RGBA{R: shade, A: 255})
			_ = png.Encode(w, im)
			return
		}
		if r.Header.Get("token") != "owner-fixture-token" {
			t.Error("wrong actor")
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1/users/"+id.TargetID+"/avatar" {
			actions = append(actions, "upload")
			_, _ = w.Write([]byte(`{"status":200}`))
			return
		}
		if r.URL.Path != "/v1/user/current" {
			t.Errorf("unexpected mutation %s", r.URL.Path)
		}
		actions = append(actions, "refresh")
		var b map[string]string
		_ = json.NewDecoder(r.Body).Decode(&b)
		if b["name"] != "Fixture Name" || len(b) != 1 {
			t.Errorf("unexpected profile mutation %v", b)
		}
		if rejectRefresh {
			w.WriteHeader(503)
			_, _ = w.Write([]byte(`{"status":503}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":200}`))
	}))
	defer server.Close()
	s.Tang.BaseURL = server.URL
	s.Media = &media.Service{Store: s.Store, TangBase: server.URL, Token: func(string) (string, bool) { return "owner-fixture-token", true }}
	avatar := server.URL + "/v1/users/source/avatar"
	if err := s.Profile(ctx, owner, "Fixture Name", avatar); err == nil {
		t.Fatal("refresh failure hidden")
	}
	if _, err := s.Store.GetCursor(ctx, "avatar-profile-refresh:"+id.TargetID); err == nil {
		t.Fatal("failed refresh marked delivered")
	}
	rejectRefresh = false
	if err := s.Profile(ctx, owner, "Fixture Name", avatar); err != nil {
		t.Fatal(err)
	}
	if err := s.Profile(ctx, owner, "Fixture Name", avatar); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(actions, []string{"upload", "refresh", "refresh"}) {
		t.Fatalf("actions %v", actions)
	}
	shade = 40 // same URL, new content; name unchanged
	if err := s.Profile(ctx, owner, "", avatar); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(actions, []string{"upload", "refresh", "refresh", "upload", "refresh"}) {
		t.Fatalf("actions %v", actions)
	}
	stored, _ := s.Store.FindIdentity(ctx, domain.SourceTencent, owner)
	if stored.DisplayName != "Fixture Name" || stored.AvatarURL != avatar {
		t.Fatal("profile not persisted")
	}
}
