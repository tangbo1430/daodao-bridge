package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseTencentTextC2C(t *testing.T) {
	payload := json.RawMessage(`{"CallbackCommand":"C2C.CallbackAfterSendMsg","From_Account":"10001","To_Account":"10002","MsgSeq":12,"MsgRandom":34,"MsgBody":[{"MsgType":"TIMTextElem","MsgContent":{"Text":"hello"}}]}`)
	event, err := ParseTencentMessage(payload)
	if err != nil {
		t.Fatal(err)
	}
	if event.Source != SourceTencent || event.ConversationType != ConversationC2C || event.Content != "hello" {
		t.Fatalf("unexpected event: %+v", event)
	}
}

func TestParseTencentFaceElemIsRelayable(t *testing.T) {
	payload := json.RawMessage(`{"CallbackCommand":"C2C.CallbackAfterSendMsg","From_Account":"10001","To_Account":"10002","MsgBody":[{"MsgType":"TIMFaceElem","MsgContent":{"Index":2,"Data":"[TUIEmoji_Expect]"}}]}`)
	event, err := ParseTencentMessage(payload)
	if err != nil {
		t.Fatal(err)
	}
	if event.Content != "🤔" {
		t.Fatalf("expected normalized emoji, got %q", event.Content)
	}
}

func TestParseTencentCustomElemIsRelayable(t *testing.T) {
	payload := json.RawMessage(`{"CallbackCommand":"C2C.CallbackAfterSendMsg","From_Account":"10001","To_Account":"10002","MsgBody":[{"MsgType":"TIMCustomElem","MsgContent":{"Data":"{\"content\":\"custom text\"}"}}]}`)
	event, err := ParseTencentMessage(payload)
	if err != nil {
		t.Fatal(err)
	}
	if event.Content != "custom text" {
		t.Fatalf("expected custom content, got %q", event.Content)
	}
}

func TestParseTencentMixedTextAndEmojiPreservesOrder(t *testing.T) {
	payload := json.RawMessage(`{"CallbackCommand":"C2C.CallbackAfterSendMsg","From_Account":"10001","To_Account":"10002","MsgBody":[{"MsgType":"TIMTextElem","MsgContent":{"Text":"看这里 "}},{"MsgType":"TIMFaceElem","MsgContent":{"Index":2,"Data":"[TUIEmoji_Expect]"}},{"MsgType":"TIMTextElem","MsgContent":{"Text":" 好吗"}}]}`)
	event, err := ParseTencentMessage(payload)
	if err != nil {
		t.Fatal(err)
	}
	if event.Content != "看这里 🤔 好吗" {
		t.Fatalf("expected mixed content, got %q", event.Content)
	}
}

func TestParseTencentUnsupportedElemGetsPlaceholder(t *testing.T) {
	payload := json.RawMessage(`{"CallbackCommand":"C2C.CallbackAfterSendMsg","From_Account":"10001","To_Account":"10002","MsgBody":[{"MsgType":"TIMImageElem","MsgContent":{}}]}`)
	event, err := ParseTencentMessage(payload)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(event.Content, "TIMImageElem") {
		t.Fatalf("expected placeholder, got %q", event.Content)
	}
}

func TestShadowID(t *testing.T) {
	if got := ShadowID(SourceTencent, "10001"); got != "tx_10001" {
		t.Fatal(got)
	}
	if got := ShadowID(SourceTang, "42"); got != "dd_42" {
		t.Fatal(got)
	}
}

func TestParseTangWebhookMessage(t *testing.T) {
	payload := json.RawMessage(`{"message_id":12345,"from_uid":"u1","to_uid":"u2","channel_id":"u2","channel_type":1,"payload":{"type":1,"content":"hello","bridge_source":"tencent","bridge_event_id":"e1"}}`)
	event, err := ParseTangMessage(payload)
	if err != nil {
		t.Fatal(err)
	}
	if event.Source != SourceTang || event.SourceMessageID != "12345" || event.Content != "hello" || event.BridgeMarker == "" {
		t.Fatalf("unexpected event: %+v", event)
	}
}

func TestParseTangMessageDetectsBridgeClientMessageNumber(t *testing.T) {
	payload := json.RawMessage(`{"message_id":"m1","client_msg_no":"hwb_event_1","from_uid":"shadow","to_uid":"real","channel_id":"real","channel_type":1,"payload":{"type":1,"content":"hello"}}`)
	event, err := ParseTangMessage(payload)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(event.BridgeMarker, "bridge") {
		t.Fatalf("expected bridge marker, got %q", event.BridgeMarker)
	}
}

func TestParseTencentMessageReadsRootCloudCustomData(t *testing.T) {
	payload := json.RawMessage(`{"CallbackCommand":"C2C.CallbackAfterSendMsg","From_Account":"shadow","To_Account":"user","MsgKey":"m1","CloudCustomData":"{\"bridge_source\":\"tangseng\"}","MsgBody":[{"MsgType":"TIMTextElem","MsgContent":{"Text":"hello"}}]}`)
	event, err := ParseTencentMessage(payload)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(event.BridgeMarker, "bridge") {
		t.Fatalf("expected root cloud marker, got %q", event.BridgeMarker)
	}
}
