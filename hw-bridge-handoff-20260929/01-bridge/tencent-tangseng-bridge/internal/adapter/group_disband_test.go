package adapter

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDestroyMirrorOnlyAcceptsSuccessOrAlreadyAbsent(t *testing.T) {
	for _, code := range []int{0, 10010, 10007, 10002} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				status := "FAIL"
				if code == 0 {
					status = "OK"
				}
				fmt.Fprintf(w, `{"ActionStatus":%q,"ErrorCode":%d}`, status, code)
			}))
			defer server.Close()
			c := TencentClient{SDKAppID: 1, SecretKey: "fixture-only", AdminAccount: "fixture-admin", BaseURL: server.URL}
			err := c.DestroyMirrorGroup(context.Background(), "dd_fixture-group")
			if (err == nil) != (code == 0 || code == 10010) {
				t.Fatalf("code %d err %v", code, err)
			}
		})
	}
}
