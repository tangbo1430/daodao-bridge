package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/account"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/adapter"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/business"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFriendListDoesNotExposeApplicationCredentials(t *testing.T) {
	ctx := context.Background()
	st := store.NewMemoryStore()
	accounts := account.New(nil)
	accounts.SetStore(st)
	_, _ = accounts.Reserve(ctx, account.Request{TencentUserID: "fixture-owner"})
	_, _ = accounts.Confirm(ctx, "fixture-owner")
	_ = st.PutFriendState(ctx, domain.FriendState{Owner: "fixture-owner", Peer: "fixture-peer", Status: "pending_in", TokenCipher: "never-expose", VercodeCipher: "never-expose-code"})
	svc := business.New(st, accounts, &adapter.TangSengClient{}, &adapter.TencentClient{}, "test-key")
	api := &Server{Business: svc, SharedSecret: "fixture-secret"}
	req := httptest.NewRequest(http.MethodGet, "/api/bridge/friend_requests?tencent_user_id=fixture-owner", nil)
	mac := hmac.New(sha256.New, []byte("fixture-secret"))
	req.Header.Set("X-Bridge-Signature", hex.EncodeToString(mac.Sum(nil)))
	rec := httptest.NewRecorder()
	api.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "fixture-peer") || strings.Contains(rec.Body.String(), "never-expose") {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}
func TestFriendMutationRequiresInternalSignature(t *testing.T) {
	api := &Server{SharedSecret: "fixture"}
	for _, action := range []string{"apply", "accept", "refuse", "delete", "block", "unblock"} {
		req := httptest.NewRequest(http.MethodPost, "/api/bridge/friend_"+action, strings.NewReader(`{"tencent_user_id":"other-user","target_tsdd_uid":"peer"}`))
		rec := httptest.NewRecorder()
		api.Handler().ServeHTTP(rec, req)
		if rec.Code != 401 {
			t.Fatalf("%s returned %d", action, rec.Code)
		}
	}
}
