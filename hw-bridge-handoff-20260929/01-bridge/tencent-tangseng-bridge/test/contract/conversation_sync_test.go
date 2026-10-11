package contract

import (
	"encoding/json"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/adapter"
	"os"
	"testing"
)

func TestDeployedConversationSyncContract(t *testing.T) {
	b, err := os.ReadFile("conversation-sync.anonymized.json")
	if err != nil {
		t.Fatal(err)
	}
	var capture struct {
		Conversation adapter.TangConversations   `json:"conversation"`
		Channel      adapter.TangChannelMessages `json:"channel"`
	}
	if err = json.Unmarshal(b, &capture); err != nil {
		t.Fatal(err)
	}
	if capture.Conversation.UID != "fixture-owner" || len(capture.Conversation.Conversations) != 1 {
		t.Fatal("conversation wrapper changed")
	}
	if capture.Conversation.Conversations[0].Version != 1790704541000000000 {
		t.Fatal("version precision lost")
	}
	if len(capture.Channel.Messages) == 0 || capture.Channel.Messages[0].MessageID() != "900000000000000020" {
		t.Fatal("native identifier precision lost")
	}
}
