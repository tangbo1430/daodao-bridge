package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Based on redacted registration[-short-device]-captured-20261001.json.
func TestNamedLoginUsesCompatibleDeviceMetadataAndPreservesNativeUID(t *testing.T) {
	uid := "0123456789abcdef0123456789abcdef"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b struct {
			Device struct {
				ID string `json:"device_id"`
			} `json:"device"`
		}
		json.NewDecoder(r.Body).Decode(&b)
		if len(b.Device.ID) > 40 {
			http.Error(w, `{"msg":"更新用户登录设备失败","status":400}`, 400)
			return
		}
		if b.Device.ID != "hwbridge"+uid {
			t.Error("unstable device metadata", b.Device.ID)
		}
		fmt.Fprintf(w, `{"data":{"uid":%q,"app_id":"fixture-app","token":"fixture-token"}}`, uid)
	}))
	defer server.Close()
	c := TangSengClient{BaseURL: server.URL, LoginEndpoint: "/login"}
	got, e := c.LoginNamedUser(context.Background(), UserCredential{Username: "fixture_user", Password: "fixture", UID: uid})
	if e != nil || got.UID != uid || got.Token != "fixture-token" {
		t.Fatal(got.UID, e)
	}
}
