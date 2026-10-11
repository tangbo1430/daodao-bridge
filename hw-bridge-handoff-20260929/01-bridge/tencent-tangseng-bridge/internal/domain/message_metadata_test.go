package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestCardAndMentionConversionKeepsCredentialsOffTencent(t *testing.T) {
	parts, _ := TangParts([]byte(`{"type":7,"uid":"native-fixture","name":"Fixture","avatar":"/avatar","vercode":"private-card-proof"}`))
	if len(parts) != 1 || parts[0].Kind != "card" {
		t.Fatalf("parts=%+v", parts)
	}
	b, _ := json.Marshal(TencentElements(parts))
	if strings.Contains(string(b), "private-card-proof") {
		t.Fatal("card credentials leaked")
	}
	wire, _ := json.Marshal(map[string]any{"MsgBody": TencentElements(parts)})
	back := TencentParts(wire)
	if len(back) != 1 || back[0].UserID != "native-fixture" {
		t.Fatalf("card round trip %+v", back)
	}
	mention, reply := TangMetadata([]byte(`{"mention":{"all":1,"uids":["native-fixture"]},"reply":{"message_id":2104992437091962880,"message_seq":8,"from_uid":"native-fixture","from_name":"Fixture","payload":{"type":1,"content":"quoted"}}}`))
	if mention == nil || mention.All != 1 || reply == nil || reply.ID != "2104992437091962880" {
		t.Fatalf("metadata %+v %+v", mention, reply)
	}
	payload := (MessageEvent{Content: "reply", Reply: reply}).TangPayload()
	if _, ok := payload["reply"].(map[string]any)["message_id"].(string); !ok {
		t.Fatal("Tang reply ID must be a JSON string; numeric IDs crash deployed history decoder")
	}
	mention, reply = TangMetadata([]byte(`{"flame":1,"mention":{"all":1},"reply":{"message_id":1,"payload":{"content":"secret"}}}`))
	if mention != nil || reply != nil {
		t.Fatal("ephemeral metadata leaked")
	}
}
func TestUnsupportedCustomAndMergedEphemeralContentAreNotExposed(t *testing.T) {
	if got := customContent(`{"unknown":"private-json"}`, "[订单]", ""); got != "[订单]" {
		t.Fatal(got)
	}
	_, summary := TangParts([]byte(`{"type":11,"msgs":[{"from_uid":"fixture","payload":{"type":1,"flame":1,"content":"must-not-copy"}}]}`))
	if strings.Contains(summary, "must-not-copy") || !strings.Contains(summary, "阅后即焚") {
		t.Fatal(summary)
	}
	_, err := ParseTencentMessage([]byte(`{"CallbackCommand":"C2C.CallbackAfterSendMsg","From_Account":"a","To_Account":"b","MsgBody":[{"MsgType":"TIMCustomElem","MsgContent":{"Data":"{\"businessID\":\"user_typing_status\"}"}}]}`))
	if !errors.Is(err, ErrIgnoredMessage) {
		t.Fatalf("typing err %v", err)
	}
}
