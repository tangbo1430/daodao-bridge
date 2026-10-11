package contract

import (
	"encoding/json"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"os"
	"testing"
)

func TestCapturedGroupSystemEventsHaveNoSender(t *testing.T) {
	raw, err := os.ReadFile("tangseng-group-create.anonymized.json")
	if err != nil {
		t.Fatal(err)
	}
	var events []json.RawMessage
	if err = json.Unmarshal(raw, &events); err != nil {
		t.Fatal(err)
	}
	for _, payload := range events {
		event, err := domain.ParseBusinessEvent(domain.EventEnvelope{Source: domain.SourceTang, Command: "msg.notify", Payload: payload})
		if err != nil || event == nil || event.Owner == "" {
			t.Fatalf("missing structured command %+v %v", event, err)
		}
		if event.Command != "groupSystem" && event.Command != "groupAvatarUpdate" {
			t.Fatalf("unexpected command %s", event.Command)
		}
	}
}
func TestCapturedSuperGroupUsesMemberPages(t *testing.T) {
	raw, err := os.ReadFile("tangseng-group-members.anonymized.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Status int             `json:"status"`
		Body   json.RawMessage `json:"body"`
	}
	if err = json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(err)
	}
	var group domain.GroupState
	var members []domain.GroupMember
	if len(rows) != 3 {
		t.Fatal("incomplete capture")
	}
	if err = json.Unmarshal(rows[0].Body, &group); err != nil {
		t.Fatal(err)
	}
	if group.GroupType != 1 || rows[1].Status != 400 || rows[2].Status != 200 {
		t.Fatal("contract changed")
	}
	if err = json.Unmarshal(rows[2].Body, &members); err != nil {
		t.Fatal(err)
	}
	if len(members) != 2 || members[0].Role != 1 || members[0].UID == "" {
		t.Fatalf("members %+v", members)
	}
}
