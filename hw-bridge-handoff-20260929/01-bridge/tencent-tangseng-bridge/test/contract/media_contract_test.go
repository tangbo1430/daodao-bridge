package contract

import (
	"encoding/json"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"os"
	"testing"
)

func TestCapturedTangMediaFields(t *testing.T) {
	raw, err := os.ReadFile("tangseng-media.anonymized.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		ID      string          `json:"message_id"`
		Payload json.RawMessage `json:"payload"`
	}
	if err = json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		envelope, _ := json.Marshal(map[string]any{"message_id": row.ID, "from_uid": "native-peer", "to_uid": "shadow-owner", "channel_id": "native-peer", "channel_type": 1, "payload": row.Payload})
		event, e := domain.ParseTangMessage(envelope)
		if e != nil || len(event.Parts) != 1 {
			t.Fatalf("%s %v", row.ID, e)
		}
		if event.Parts[0].Kind == "video" && (event.Parts[0].Seconds != 31 || event.Parts[0].Size != 43356721 || event.Parts[0].Cover == "") {
			t.Fatalf("video fields lost %+v", event.Parts[0])
		}
	}
}
func TestFlameNeverCopiesOriginalOrMedia(t *testing.T) {
	event, err := domain.ParseTangMessage(json.RawMessage(`{"message_id":2104980126914580481,"from_uid":"a","to_uid":"b","channel_id":"a","channel_type":1,"payload":{"type":2,"url":"private.jpg","content":"must-never-copy","flame":1}}`))
	if err != nil {
		t.Fatal(err)
	}
	if event.SourceMessageID != "2104980126914580481" || len(event.Parts) != 1 || event.Parts[0].Kind != "tip" || event.Parts[0].URL != "" {
		t.Fatalf("%+v", event)
	}
}
