package media

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Synthetic timing fixture models the native upload handler's observed source
// order: notify friends, then set is_upload_avatar, then return success.
func TestFirstAvatarRefreshAfterNativeCommitAndRetry(t *testing.T) {
	for _, failRefresh := range []bool{false, true} {
		t.Run(fmt.Sprint(failRefresh), func(t *testing.T) {
			uploaded, posts := 0, 0
			seen := []int{}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/v1/users/fixture-owner" {
					fmt.Fprintf(w, `{"uid":"fixture-owner","is_upload_avatar":%d}`, uploaded)
					return
				}
				if r.Method == "GET" {
					w.Header().Set("Content-Type", "image/png")
					_ = png.Encode(w, image.NewRGBA(image.Rect(0, 0, 1, 1)))
					return
				}
				if r.Header.Get("token") != "fixture-owner-token" {
					t.Error("wrong actor")
				}
				if err := r.ParseMultipartForm(1 << 20); err != nil {
					t.Error(err)
				}
				defer r.MultipartForm.RemoveAll()
				file, _, err := r.FormFile("file")
				if err != nil {
					t.Error(err)
				} else {
					if _, err := png.Decode(file); err != nil {
						t.Error(err)
					}
					file.Close()
				}
				posts++
				if failRefresh && posts == 2 {
					http.Error(w, "retry", 503)
					return
				}
				seen = append(seen, uploaded)
				uploaded = 1
				w.Write([]byte(`{"status":200}`))
			}))
			defer srv.Close()
			s := &Service{Store: store.NewMemoryStore(), TangBase: srv.URL, Token: func(string) (string, bool) { return "fixture-owner-token", true }}
			source := srv.URL + "/v1/users/fixture-source/avatar"
			err := s.UploadAvatar(context.Background(), "fixture-owner", "", source)
			if (err != nil) != failRefresh {
				t.Fatal("unexpected refresh result", err)
			}
			if err = s.UploadAvatar(context.Background(), "fixture-owner", "", source); err != nil {
				t.Fatal(err)
			}
			want := 2
			if failRefresh {
				want = 3
			}
			if posts != want || len(seen) != 2 || seen[0] != 0 || seen[1] != 1 {
				t.Fatalf("posts=%d notification states=%v", posts, seen)
			}
			if err = s.UploadAvatar(context.Background(), "fixture-owner", "", source); err != nil || posts != want {
				t.Fatal("duplicate refresh", err, posts)
			}
		})
	}
}

func TestMissingGroupAvatarDoesNotMaskOtherFailures(t *testing.T) {
	status := http.StatusNotFound
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status) }))
	defer srv.Close()
	s := &Service{Store: store.NewMemoryStore(), TangBase: srv.URL}
	for _, test := range []struct {
		kind     string
		code     int
		optional bool
	}{
		{"groups", 404, true}, {"users", 404, false}, {"groups", 403, false}, {"groups", 500, false},
	} {
		status = test.code
		_, err := s.AvatarURL(context.Background(), test.kind, "fixture")
		if err == nil || errors.Is(err, ErrGroupAvatarNotFound) != test.optional {
			t.Fatalf("kind=%s status=%d error=%v", test.kind, status, err)
		}
	}
}

func TestAvatarContentFingerprintAndRejectedUpload(t *testing.T) {
	shade := uint8(20)
	uploads := 0
	reject := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/v1/users/fixture-owner" {
			w.Write([]byte(`{"uid":"fixture-owner","is_upload_avatar":1}`))
			return
		}
		if r.Method == "GET" {
			var b bytes.Buffer
			im := image.NewRGBA(image.Rect(0, 0, 1, 1))
			im.Set(0, 0, color.RGBA{R: shade, A: 255})
			_ = png.Encode(&b, im)
			w.Header().Set("Content-Type", "image/png")
			w.Write(b.Bytes())
			return
		}
		if r.Header.Get("token") != "fixture-owner-token" {
			t.Error("wrong avatar actor")
		}
		uploads++
		if reject {
			w.Write([]byte(`{"status":400}`))
		} else {
			w.Write([]byte(`{"status":200}`))
		}
	}))
	defer srv.Close()
	s := &Service{Store: store.NewMemoryStore(), TangBase: srv.URL, Token: func(string) (string, bool) { return "fixture-owner-token", true }}
	ctx := context.Background()
	source := srv.URL + "/v1/users/fixture-source/avatar"
	if err := s.UploadAvatar(ctx, "fixture-owner", "", source); err != nil {
		t.Fatal(err)
	}
	if err := s.UploadAvatar(ctx, "fixture-owner", "", source); err != nil || uploads != 1 {
		t.Fatalf("duplicate uploaded: %d %v", uploads, err)
	}
	shade = 40
	reject = true
	if err := s.UploadAvatar(ctx, "fixture-owner", "", source); err == nil {
		t.Fatal("business rejection accepted")
	}
	reject = false
	if err := s.UploadAvatar(ctx, "fixture-owner", "", source); err != nil || uploads != 3 {
		t.Fatalf("changed content not retried: %d %v", uploads, err)
	}
}
