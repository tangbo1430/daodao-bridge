package adapter

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestUserRequestRefreshesOnlyExpiredActorOnce(t *testing.T) {
	refresh, calls := 0, 0
	denied := false
	tokens := &sync.Map{}
	tokens.Store("fixture-owner", "old")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if denied {
			w.WriteHeader(403)
			return
		}
		if r.Header.Get("token") != "fresh" {
			w.WriteHeader(401)
			return
		}
		fmt.Fprint(w, `{"status":200}`)
	}))
	defer srv.Close()
	c := &TangSengClient{BaseURL: srv.URL, Token: "admin-must-not-be-used", UserTokens: tokens, RefreshUser: func(_ context.Context, uid string) error {
		if uid != "fixture-owner" {
			t.Fatal("wrong actor refreshed")
		}
		refresh++
		tokens.Store(uid, "fresh")
		return nil
	}}
	if err := c.UserRequest(context.Background(), "fixture-owner", "GET", "/fixture", nil, nil); err != nil || refresh != 1 || calls != 2 {
		t.Fatalf("refresh=%d calls=%d error=%v", refresh, calls, err)
	}
	denied = true
	if err := c.UserRequest(context.Background(), "fixture-owner", "GET", "/fixture", nil, nil); err == nil || refresh != 1 {
		t.Fatal("permission rejection refreshed credentials")
	}
}
