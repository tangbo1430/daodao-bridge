package domain

import (
	"encoding/json"
	"testing"
)

func TestGroupSequenceKeysAreScopedAndDistinct(t *testing.T) {
	seen := map[string]bool{}
	for _, raw := range []string{`{"CallbackCommand":"Group.CallbackAfterSendMsg","GroupId":"dd_a","From_Account":"owner","MsgSeq":1,"MsgBody":[{"MsgType":"TIMTextElem","MsgContent":{"Text":"one"}}]}`, `{"CallbackCommand":"Group.CallbackAfterSendMsg","GroupId":"dd_a","From_Account":"owner","MsgSeq":2,"MsgBody":[{"MsgType":"TIMTextElem","MsgContent":{"Text":"two"}}]}`, `{"CallbackCommand":"Group.CallbackAfterSendMsg","GroupId":"dd_b","From_Account":"owner","MsgSeq":1,"MsgBody":[{"MsgType":"TIMTextElem","MsgContent":{"Text":"three"}}]}`} {
		m, err := ParseTencentMessage(json.RawMessage(raw))
		if err != nil {
			t.Fatal(err)
		}
		if seen[m.SourceMessageID] {
			t.Fatal("group message collision")
		}
		seen[m.SourceMessageID] = true
	}
}
func TestMemberUpdateUsesDocumentedCommand(t *testing.T) {
	event, err := ParseBusinessEvent(EventEnvelope{Source: SourceTang, Command: "msg.notify", Payload: json.RawMessage(`{"to_uid":"hw-owner","payload":{"type":99,"cmd":"memberUpdate","param":{"group_no":"fixture-group"}}}`)})
	if err != nil || event == nil || event.Command != "memberUpdate" {
		t.Fatalf("%+v %v", event, err)
	}
}
