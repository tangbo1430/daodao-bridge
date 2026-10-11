package session

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWuKongJSONRPCRoundTrip(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		var frame rpcFrame
		if conn.ReadJSON(&frame) != nil || frame.Method != "connect" {
			return
		}
		_ = conn.WriteJSON(rpcFrame{JSONRPC: "2.0", ID: frame.ID, Result: mustJSON(map[string]any{"reasonCode": 1})})
		if conn.ReadJSON(&frame) != nil || frame.Method != "send" {
			return
		}
		var sendParams map[string]json.RawMessage
		if json.Unmarshal(frame.Params, &sendParams) != nil {
			return
		}
		var encoded string
		if json.Unmarshal(sendParams["payload"], &encoded) != nil {
			t.Errorf("send payload is not base64 string: %s", sendParams["payload"])
			return
		}
		payload, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			t.Errorf("decode payload: %v", err)
			return
		}
		var textPayload map[string]any
		if json.Unmarshal(payload, &textPayload) != nil || textPayload["content"] != "hi" {
			t.Errorf("unexpected send payload: %s", payload)
			return
		}
		_ = conn.WriteJSON(rpcFrame{JSONRPC: "2.0", ID: frame.ID, Result: mustJSON(map[string]any{"reasonCode": 1, "messageId": "m-target"})})
		incoming := mustJSON(map[string]any{"type": 1, "content": "hello"})
		_ = conn.WriteJSON(rpcFrame{JSONRPC: "2.0", Method: "recv", Params: mustJSON(map[string]any{"messageId": "m-in", "messageSeq": 7, "channelId": "peer", "channelType": 1, "fromUid": "peer", "payload": base64.StdEncoding.EncodeToString(incoming)})})
		var ack rpcFrame
		if conn.ReadJSON(&ack) != nil || ack.Method != "recvack" || ack.ID != "" || ack.JSONRPC != "2.0" {
			t.Errorf("unexpected recvack: %+v", ack)
		}
	}))
	defer server.Close()
	wsURL, _ := url.Parse(server.URL)
	wsURL.Scheme = "ws"
	s := New(Config{URL: wsURL.String(), UID: "u1", Token: "token", PingInterval: time.Hour})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := s.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	target, err := s.SendText(ctx, "peer", 1, "hi", "client-1", true)
	if err != nil {
		t.Fatal(err)
	}
	if target != "m-target" {
		t.Fatalf("target=%s", target)
	}
	select {
	case event := <-s.Events():
		if event.MessageID != "m-in" || event.MessageSeq != 7 {
			t.Fatalf("event=%+v", event)
		}
		if err := s.Ack(ctx, event); err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	_ = s.Close()
}
