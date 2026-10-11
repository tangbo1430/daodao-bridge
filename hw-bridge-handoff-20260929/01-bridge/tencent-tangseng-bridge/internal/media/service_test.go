package media

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var zeroTime time.Time

func TestMediaStreamsPastBusinessWriteDeadline(t *testing.T) {
	// Keep upstream open after its first byte: a media download must start before
	// the complete object is available and outlive the normal API write deadline.
	release := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "8")
		fmt.Fprint(w, "f")
		w.(http.Flusher).Flush()
		<-release
		fmt.Fprint(w, "ixture!")
	}))
	defer upstream.Close()
	id := fmt.Sprintf("%x", sha256.Sum256([]byte("fixture!")))
	s := &Service{Store: store.NewMemoryStore(), TangBase: upstream.URL, Secret: "fixture-only"}
	if err := s.Store.PutMedia(context.Background(), domain.MediaObject{SHA256: id, ObjectURL: "file/preview/fixture", Size: 8, MIME: "video/mp4", Name: "fixture.mp4"}); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(s)
	server.Config.WriteTimeout = 10 * time.Millisecond
	server.Start()
	defer server.Close()
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(server.URL + "/?id=" + id + "&sig=" + s.signature(id))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	time.Sleep(30 * time.Millisecond)
	close(release)
	data, err := io.ReadAll(resp.Body)
	if err != nil || string(data) != "fixture!" {
		t.Fatalf("stream truncated by business deadline: %q %v", data, err)
	}
}

func TestTransferUploadsAsActorAndServesSignedRange(t *testing.T) {
	uploads := 0
	content := "fixture-media-binary"
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/file/upload" {
			uploads++
			if r.Header.Get("token") != "actor-only-token" {
				t.Error("wrong upload actor")
			}
			f, _, err := r.FormFile("file")
			if err != nil {
				t.Error(err)
				http.Error(w, "bad", 400)
				return
			}
			defer f.Close()
			data, _ := io.ReadAll(f)
			if string(data) != content {
				t.Error("wrong upload content")
			}
			if !strings.HasPrefix(r.URL.Query().Get("path"), "/1/peer/") {
				t.Error("wrong upload channel")
			}
			fmt.Fprint(w, `{"path":"file/preview/chat/1/peer/uploaded.bin"}`)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/v1/file/preview/") {
			if r.Header.Get("token") != "" {
				t.Error("credentials leaked to preview")
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			http.ServeContent(w, r, "fixture.bin", zeroTime, strings.NewReader(content))
			return
		}
		http.NotFound(w, r)
	}))
	defer upstream.Close()
	s := &Service{Store: store.NewMemoryStore(), TangBase: upstream.URL, PublicBase: "https://app.invalid/api/bridge/media", Secret: "fixture-only", Token: func(uid string) (string, bool) { return "actor-only-token", uid == "actor" }}
	event := domain.MessageEvent{Source: domain.SourceTencent, ConversationType: domain.ConversationC2C, Parts: []domain.MessagePart{{Kind: "file", URL: upstream.URL + "/v1/file/preview/source", Name: "fixture.bin"}}}
	first, err := s.Prepare(context.Background(), event, domain.Identity{TargetID: "actor"}, domain.Identity{TargetID: "peer"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(first.Parts[0].URL, "file/preview/") || first.Parts[0].Size != int64(len(content)) {
		t.Fatalf("%+v", first)
	}
	_, err = s.Prepare(context.Background(), event, domain.Identity{TargetID: "actor"}, domain.Identity{TargetID: "peer"})
	if err != nil || uploads != 1 {
		t.Fatalf("cache: uploads %d err %v", uploads, err)
	}
	incoming := domain.MessageEvent{Source: domain.SourceTang, ToAccount: "actor", ConversationType: domain.ConversationC2C, Parts: []domain.MessagePart{{Kind: "file", URL: first.Parts[0].URL}}}
	converted, err := s.Prepare(context.Background(), incoming, domain.Identity{}, domain.Identity{})
	if err != nil {
		t.Fatal(err)
	}
	uri := converted.Parts[0].URL
	if !strings.HasPrefix(uri, s.PublicBase) || strings.Contains(uri, "actor-only-token") {
		t.Fatalf("unsafe public link %s", uri)
	}
	req := httptest.NewRequest("GET", uri, nil)
	req.Header.Set("Range", "bytes=0-6")
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != 206 || rec.Body.String() != "fixture" || rec.Header().Get("Content-Range") == "" {
		t.Fatalf("range %d %s", rec.Code, rec.Body.String())
	}
	head := httptest.NewRecorder()
	s.ServeHTTP(head, httptest.NewRequest("HEAD", uri, nil))
	if head.Code != 200 || head.Body.Len() != 0 || head.Header().Get("Content-Length") != "20" {
		t.Fatalf("head %d %v", head.Code, head.Header())
	}
	bad := httptest.NewRecorder()
	s.ServeHTTP(bad, httptest.NewRequest("GET", uri+"x", nil))
	if bad.Code != 403 {
		t.Fatalf("signature bypass %d", bad.Code)
	}
}
func TestRejectsPrivateTargetsTraversalAndOversize(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "redirect") {
			http.Redirect(w, r, "http://169.254.169.254/latest/meta-data", 302)
			return
		}
		w.Header().Set("Content-Length", "104857601")
	}))
	defer upstream.Close()
	s := &Service{TangBase: upstream.URL}
	for _, address := range []string{"file:///etc/passwd", "http://127.0.0.1/secrets", upstream.URL + "/v1/user/token", "file/preview/../../user", "file/preview/large", "file/preview/redirect"} {
		if f, err := s.download(context.Background(), address); err == nil {
			f.Cleanup()
			t.Fatalf("accepted %s", address)
		}
	}
}
func TestPayloadCarriesNoInternalAddress(t *testing.T) {
	part := domain.MessagePart{Kind: "video", URL: "https://app.invalid/media?id=a", Cover: "https://app.invalid/media?id=b", Width: 1920, Height: 1080, Size: 43356721, Seconds: 31, CoverSize: 100}
	raw, _ := json.Marshal(domain.TencentElements([]domain.MessagePart{part}))
	for _, v := range []string{"TIMVideoFileElem", "VideoDownloadFlag", "ThumbDownloadFlag", "43356721", "1920"} {
		if !strings.Contains(string(raw), v) {
			t.Fatalf("missing %s in %s", v, raw)
		}
	}
}
