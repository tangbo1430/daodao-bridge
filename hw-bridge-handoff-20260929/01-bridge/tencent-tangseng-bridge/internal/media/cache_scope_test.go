package media

import (
	"context"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDownloadCacheCannotBypassActorUpload(t *testing.T) {
	uploads := []string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			uploads = append(uploads, r.Header.Get("token"))
			fmt.Fprintf(w, `{"path":"file/preview/%s"}`, r.Header.Get("token"))
			return
		}
		fmt.Fprint(w, "fixture-identical-content")
	}))
	defer srv.Close()
	s := &Service{TangBase: srv.URL, Store: store.NewMemoryStore(), Token: func(uid string) (string, bool) { return uid + "-token", true }}
	ctx := context.Background()
	p := domain.MessagePart{Kind: "file", Name: "fixture.bin", URL: srv.URL + "/v1/file/preview/source"}
	original, err := s.convertObject(ctx, p, "", "", 1, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, owner := range []string{"actor-a", "actor-a", "actor-b"} {
		v, e := s.convertObject(ctx, p, owner, "peer", 1, true)
		if e != nil || v.ObjectURL == original.ObjectURL {
			t.Fatalf("actor upload bypassed: %+v %v", v, e)
		}
	}
	if len(uploads) != 2 || uploads[0] != "actor-a-token" || uploads[1] != "actor-b-token" {
		t.Fatalf("upload cache scope %v", uploads)
	}
	public, _ := s.Store.GetMedia(ctx, original.SHA256)
	if public.ObjectURL != original.ObjectURL {
		t.Fatal("existing viewer capability changed")
	}
}
