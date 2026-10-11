package contract

import (
	"encoding/json"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"os"
	"testing"
)

func TestCapturedMessageStateCommands(t *testing.T) {
	raw, err := os.ReadFile("message-state.anonymized.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture map[string]json.RawMessage
	if err = json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	e, err := domain.ParseBusinessEvent(domain.EventEnvelope{Source: domain.SourceTencent, Command: "C2C.CallbackAfterMsgWithDraw", Payload: fixture["tencent_withdraw"]})
	if err != nil || e == nil || e.Command != "C2C.CallbackAfterMsgWithDraw" {
		t.Fatalf("withdraw %v %v", e, err)
	}
	payload, _ := json.Marshal(map[string]any{"to_uid": "fixture-owner-shadow", "channel_type": 2, "payload": fixture["tang_extra_command"]})
	e, err = domain.ParseBusinessEvent(domain.EventEnvelope{Source: domain.SourceTang, Command: "msg.notify", Payload: payload})
	if err != nil || e == nil || e.Command != "syncMessageExtra" || e.Owner != "fixture-owner-shadow" {
		t.Fatalf("extra %v %v", e, err)
	}
}
