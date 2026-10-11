package contract

import (
	"encoding/json"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/adapter"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"os"
	"testing"
)

func TestTencentRealRoamingContracts(t *testing.T) {
	b, err := os.ReadFile("tencent-roam.anonymized.json")
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		C2C   adapter.TencentC2CRoam   `json:"c2c"`
		Group adapter.TencentGroupRoam `json:"group"`
	}
	if err = json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	if len(v.C2C.Messages) == 0 || len(v.Group.Messages) == 0 {
		t.Fatal("empty capture")
	}
	for _, m := range v.C2C.Messages {
		if m.Flags&8 != 0 {
			continue
		}
		e, err := m.Envelope("")
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := domain.ParseTencentMessage(e.Payload)
		if err != nil || parsed.OccurredAt != m.Time || parsed.SourceMessageID != m.Key {
			t.Fatalf("C2C contract %v %+v", err, parsed)
		}
	}
	for _, m := range v.Group.Messages {
		if m.Place != 0 || m.System != 0 {
			continue
		}
		e, err := m.Envelope(v.Group.Group)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := domain.ParseTencentMessage(e.Payload)
		if err != nil || parsed.SourceSeq != m.Seq || parsed.ConversationType != domain.ConversationGroup {
			t.Fatalf("group contract %v %+v", err, parsed)
		}
	}
}
