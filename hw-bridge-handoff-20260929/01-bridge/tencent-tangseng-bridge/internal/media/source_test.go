package media

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
)

func TestSourceTicketDoesNotExposeInternalOrigin(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") != "bytes=0-0" {
			t.Error("source resolution requested full object")
		}
		if r.Header.Get("token") != "" || r.Header.Get("X-Bridge-Signature") != "" {
			t.Error("credential leaked")
		}
		w.WriteHeader(206)
		fmt.Fprint(w, "x")
	}))
	defer upstream.Close()
	id := strings.Repeat("a", 64)
	s := &Service{Store: store.NewMemoryStore(), TangBase: upstream.URL, Secret: "fixture-only"}
	if err := s.Store.PutMedia(context.Background(), domain.MediaObject{SHA256: id, ObjectURL: "file/preview/fixture", Size: 1}); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	s.ServeSource(rec, httptest.NewRequest("GET", "/?id="+id+"&sig="+s.signature(id), nil))
	if rec.Code != 200 || strings.Contains(rec.Body.String(), upstream.URL) {
		t.Fatalf("unsafe response: %d", rec.Code)
	}
	var out map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	mac := hmac.New(sha256.New, []byte(s.Secret))
	mac.Write([]byte("hw-bridge-media-source-v1\n" + out["payload"]))
	if hex.EncodeToString(mac.Sum(nil)) != out["signature"] || !strings.Contains(out["payload"], `"mode":"bridge"`) {
		t.Fatal("unsigned or invalid fallback")
	}
	bad := httptest.NewRecorder()
	s.ServeSource(bad, httptest.NewRequest("GET", "/?id="+id+"&sig=bad", nil))
	if bad.Code != 403 {
		t.Fatal("source accessible without capability")
	}
}
