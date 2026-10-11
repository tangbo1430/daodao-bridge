package contract

import (
	"encoding/base64"
	"encoding/json"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"os"
	"testing"
)

// Captured from existing production inbox on 2026-09-29; IDs are consistently
// pseudonymized, content removed and request credentials replaced. No real
// credentials or application text are present in these fixtures.
func TestCapturedFriendCommands(t *testing.T) {
	for _, name := range []string{"tangseng-friend-request.anonymized.json", "tangseng-friend-accept.anonymized.json"} {
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, encoded := range []bool{false, true} {
			data := raw
			if encoded {
				var outer map[string]json.RawMessage
				if err = json.Unmarshal(raw, &outer); err != nil {
					t.Fatal(err)
				}
				outer["payload"], _ = json.Marshal(base64.StdEncoding.EncodeToString(outer["payload"]))
				data, _ = json.Marshal(outer)
			}
			event, err := domain.ParseBusinessEvent(domain.EventEnvelope{Source: domain.SourceTang, Command: "msg.notify", Payload: data})
			if err != nil || event == nil || event.Owner == "" {
				t.Fatalf("%s: %+v %v", name, event, err)
			}
			if event.Command != "friendRequest" && event.Command != "friendAccept" {
				t.Fatalf("command %q", event.Command)
			}
		}
	}
}
