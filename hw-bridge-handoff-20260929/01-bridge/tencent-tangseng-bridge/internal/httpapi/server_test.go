package httpapi

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/engine"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
)

func TestSignatureRequired(t *testing.T) {
	st := store.NewMemoryStore()
	api := &Server{Engine: &engine.Engine{Store: st}, Store: st, SharedSecret: "secret"}
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/events/tencent", bytes.NewBufferString(`{"source":"tencent","payload":{}}`))
	res := httptest.NewRecorder()
	api.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d", res.Code)
	}
}

func TestDeadLetterAuditOmitsMessageContentAndBlocksAmbiguousReplay(t *testing.T) {
	ctx := context.Background()
	st := store.NewMemoryStore()
	job := domain.OutboxJob{ID: "dead-1", EventID: "event-1", Target: domain.SourceTencent, Status: "pending", Message: domain.MessageEvent{Source: domain.SourceTang, SourceMessageID: "message-1", ConversationID: "peer", Content: "private message"}}
	if err := st.CreateOutbox(ctx, job); err != nil {
		t.Fatal(err)
	}
	if err := st.FailOutbox(ctx, job.ID, "context deadline exceeded", true); err != nil {
		t.Fatal(err)
	}
	api := &Server{Engine: &engine.Engine{Store: st}, Store: st, SharedSecret: "secret"}
	sign := func(req *http.Request) {
		h := hmac.New(sha256.New, []byte("secret"))
		req.Header.Set("X-Bridge-Signature", "sha256="+hex.EncodeToString(h.Sum(nil)))
	}
	list := httptest.NewRequest(http.MethodGet, "/admin/dead-letter/", nil)
	sign(list)
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, list)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "dead-1") || strings.Contains(response.Body.String(), "private message") {
		t.Fatalf("audit status=%d body=%s", response.Code, response.Body.String())
	}
	replay := httptest.NewRequest(http.MethodPost, "/admin/dead-letter/dead-1", nil)
	sign(replay)
	response = httptest.NewRecorder()
	api.Handler().ServeHTTP(response, replay)
	if response.Code != http.StatusConflict {
		t.Fatalf("ambiguous replay status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestSignedEventAccepted(t *testing.T) {
	st := store.NewMemoryStore()
	api := &Server{Engine: &engine.Engine{Store: st}, Store: st, SharedSecret: "secret"}
	body := []byte(`{"source":"tencent","event_id":"e1","command":"C2C.CallbackAfterSendMsg","payload":{"CallbackCommand":"C2C.CallbackAfterSendMsg","From_Account":"a","To_Account":"b","MsgBody":[{"MsgType":"TIMTextElem","MsgContent":{"Text":"hi"}}]}}`)
	h := hmac.New(sha256.New, []byte("secret"))
	h.Write(body)
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/events/tencent", bytes.NewReader(body))
	req.Header.Set("X-Bridge-Signature", "sha256="+hex.EncodeToString(h.Sum(nil)))
	res := httptest.NewRecorder()
	api.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusAccepted {
		t.Fatalf("code=%d body=%s", res.Code, res.Body.String())
	}
	if _, err := st.Stats(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestRawTencentCallbackAcceptedWithHWBSignature(t *testing.T) {
	st := store.NewMemoryStore()
	api := &Server{Engine: &engine.Engine{Store: st}, Store: st, SharedSecret: "secret"}
	body := []byte(`{"CallbackCommand":"C2C.CallbackAfterSendMsg","From_Account":"a","To_Account":"b","MsgKey":"k1","MsgBody":[{"MsgType":"TIMTextElem","MsgContent":{"Text":"hi"}}]}`)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := "n1"
	h := hmac.New(sha256.New, []byte("secret"))
	h.Write([]byte(ts + "\n" + nonce + "\n"))
	h.Write(body)
	req := httptest.NewRequest(http.MethodPost, "/cb/tim", bytes.NewReader(body))
	req.Header.Set("X-HWB-Timestamp", ts)
	req.Header.Set("X-HWB-Nonce", nonce)
	req.Header.Set("X-HWB-Sign", hex.EncodeToString(h.Sum(nil)))
	res := httptest.NewRecorder()
	api.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", res.Code, res.Body.String())
	}
	stats, _ := st.Stats(context.Background())
	if stats.InboxReceived != 1 {
		t.Fatalf("stats=%+v", stats)
	}
}

func TestNewAccountEndpointsPersistNonceAndRejectReplay(t *testing.T) {
	st := store.NewMemoryStore()
	api := &Server{Store: st, SharedSecret: "fixture-secret"}
	body := []byte(`{}`)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := "fixture-nonce-1234567890"
	h := hmac.New(sha256.New, []byte(api.SharedSecret))
	h.Write([]byte(ts + "\n" + nonce + "\n"))
	h.Write(body)
	for attempt, want := range []int{503, 409} {
		// A restart sees the same durable nonce; even a different endpoint
		// cannot reuse it. No credentials/body are kept in the nonce record.
		api = &Server{Store: st, SharedSecret: "fixture-secret"}
		path := "/api/v1/account/prepare"
		if attempt == 1 {
			path = "/api/v1/account/commit"
		}
		r := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
		r.Header.Set("X-HWB-Timestamp", ts)
		r.Header.Set("X-HWB-Nonce", nonce)
		r.Header.Set("X-HWB-Sign", hex.EncodeToString(h.Sum(nil)))
		w := httptest.NewRecorder()
		api.Handler().ServeHTTP(w, r)
		if w.Code != want {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}
