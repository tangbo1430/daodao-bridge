package adapter

import (
	"context"
	"encoding/json"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Synthetic contract based on the bundled APK OfflinePushExtBusinessInfo and
// Tencent OfflinePushInfo REST schema, not a captured vendor notification.
func TestSendOfflinePushAPKRoutingAndTipSuppression(t *testing.T) {
	for _, kind := range []string{"text", "image", "audio", "video", "file", "tip"} {
		for _, group := range []bool{false, true} {
			t.Run(kind+map[bool]string{true: "-group", false: "-c2c"}[group], func(t *testing.T) {
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var body map[string]any
					json.NewDecoder(r.Body).Decode(&body)
					push, ok := body["OfflinePushInfo"].(map[string]any)
					if !ok {
						t.Error("no push")
						return
					}
					if kind == "tip" {
						if push["PushFlag"] != float64(1) {
							t.Error(push)
						}
					} else {
						var ext struct {
							Entity struct {
								Version, Action, ChatType          int
								Sender, Nickname, FaceURL, Content string
							}
						}
						if err := json.Unmarshal([]byte(push["Ext"].(string)), &ext); err != nil {
							t.Error(err)
						}
						wantID, wantType := "native-peer", 1
						if group {
							wantID, wantType = "dd_fixture", 2
						}
						if ext.Entity.Version != 1 || ext.Entity.Action != 1 || ext.Entity.ChatType != wantType || ext.Entity.Sender != wantID {
							t.Errorf("%+v", ext)
						}
						if kind != "text" && ext.Entity.Content == "secret-url" {
							t.Error("media URL in push")
						}
					}
					w.Write([]byte(`{"ActionStatus":"OK","ErrorCode":0,"MsgKey":"fixture-key"}`))
				}))
				defer s.Close()
				c := TencentClient{SDKAppID: 1, SecretKey: "fixture-secret", AdminAccount: "fixture-admin", BaseURL: s.URL}
				event := domain.MessageEvent{Source: domain.SourceTang, Content: "secret-url", Parts: []domain.MessagePart{{Kind: kind, Text: "test"}}, ConversationType: domain.ConversationC2C}
				if group {
					event.ConversationType = domain.ConversationGroup
				}
				_, err := c.SendText(context.Background(), event, domain.Identity{TargetID: "native-peer", DisplayName: "sender"}, domain.Identity{TargetID: "dd_fixture", DisplayName: "group"}, "fixture-marker")
				if err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
