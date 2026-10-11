package adapter

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFriendAddResultCodes(t *testing.T) {
	for _, tc := range []struct {
		Code           int
		Force, Success bool
	}{{0, true, true}, {30015, true, true}, {30539, false, true}, {30539, true, false}, {30515, true, false}, {30516, false, false}, {30525, false, false}} {
		t.Run(fmt.Sprintf("%d-%t", tc.Code, tc.Force), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, `{"ActionStatus":"OK","ErrorCode":0,"ResultItem":[{"ResultCode":%d}]}`, tc.Code)
			}))
			defer server.Close()
			c := TencentClient{BaseURL: server.URL, SDKAppID: 1, SecretKey: "fixture", AdminAccount: "admin"}
			err := c.FriendAdd(context.Background(), "from", "to", "", tc.Force)
			if (err == nil) != tc.Success {
				t.Fatalf("result %v", err)
			}
		})
	}
}
