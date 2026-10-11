package media

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
)

func TestOfficeFilePreservesMIMEAndDownloadBytes(t *testing.T) {
	// Synthetic content; MIME and relative preview path reproduce the observed
	// desktop file contract. Database width is verified separately on MySQL.
	const mime = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	const content = "PK\x03\x04office-file-fixture"
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/file/preview/chat/1/peer/fixture.docx" {
			t.Errorf("unexpected preview path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", mime)
		fmt.Fprint(w, content)
	}))
	defer upstream.Close()
	s := &Service{Store: store.NewMemoryStore(), TangBase: upstream.URL, PublicBase: "https://app.invalid/media", Secret: "fixture-only"}
	message := domain.MessageEvent{Source: domain.SourceTang, Parts: []domain.MessagePart{{Kind: "file", URL: "file/preview/chat/1/peer/fixture.docx", Name: "验收文档.docx"}}}
	prepared, err := s.Prepare(context.Background(), message, domain.Identity{}, domain.Identity{})
	if err != nil {
		t.Fatal(err)
	}
	id := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))
	object, err := s.Store.GetMedia(context.Background(), id)
	if err != nil || object.MIME != mime || object.Name != "验收文档.docx" {
		t.Fatalf("metadata mismatch: %+v %v", object, err)
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("GET", prepared.Parts[0].URL, nil))
	if w.Code != 200 || w.Header().Get("Content-Type") != mime || w.Body.String() != content {
		t.Fatalf("file gateway mismatch: %d %s", w.Code, w.Header().Get("Content-Type"))
	}
}
