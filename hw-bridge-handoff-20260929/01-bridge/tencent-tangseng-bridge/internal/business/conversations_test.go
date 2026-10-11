package business

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConversationSyncRestartsWithoutSkippingPagesOrImportingOldMessages(t *testing.T) {
	s, _, owner, peer := fixture(t)
	ctx := context.Background()
	id, _ := s.Accounts.Profile(ctx, owner)
	_ = s.Store.PutCursor(ctx, domain.SyncCursor{Key: "account:" + owner, Aux: "confirmed", StartedAt: 100})
	acks, requests := 0, []int64{}
	failInbox := true
	seen := map[string]bool{}
	s.Accept = func(_ context.Context, e domain.EventEnvelope) (bool, error) {
		m, err := domain.ParseTangMessage(e.Payload)
		if err != nil {
			return false, err
		}
		if m.SourceMessageID == "900000000000000003" && failInbox {
			return false, fmt.Errorf("fixture DB unavailable")
		}
		if m.OccurredAt < 100 {
			t.Fatal("pre-cutover history imported")
		}
		ok := !seen[m.SourceMessageID]
		seen[m.SourceMessageID] = true
		return ok, nil
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]json.RawMessage
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("token") != "owner-fixture-token" {
			t.Error("wrong sync token")
		}
		switch r.URL.Path {
		case "/v1/conversation/sync":
			fmt.Fprintf(w, `{"uid":%q,"conversations":[{"channel_id":%q,"channel_type":1,"last_msg_seq":4,"timestamp":200,"version":1790704541000000000}]}`, id.Identity.TargetID, peer)
		case "/v1/message/channel/sync":
			var start int64
			_ = json.Unmarshal(body["start_message_seq"], &start)
			requests = append(requests, start)
			items := []map[string]any{}
			for n := start; n <= 4 && n < start+2; n++ {
				when := 200
				if n == 1 {
					when = 99
				}
				items = append(items, map[string]any{"message_idstr": fmt.Sprintf("90000000000000000%d", n), "message_seq": n, "from_uid": peer, "channel_id": peer, "channel_type": 1, "timestamp": when, "payload": map[string]any{"type": 1, "content": "fixture"}})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"messages": items, "more": 0})
		case "/v1/message/sync":
			fmt.Fprint(w, `[]`)
		case "/v1/conversation/syncack":
			acks++
			fmt.Fprint(w, `{}`)
		default:
			t.Errorf("unexpected sync API %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	s.Tang.BaseURL = srv.URL
	if _, err := s.SyncAccount(ctx, owner); err == nil {
		t.Fatal("inbox failure concealed")
	}
	c, _ := s.Store.GetCursor(ctx, channelCursorKey(owner, peer, 1))
	if c.Value != 2 || acks != 0 {
		t.Fatalf("cursor advanced or ack before durable acceptance: %+v acks=%d", c, acks)
	}
	failInbox = false
	result, err := s.SyncAccount(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	if len(seen) != 3 || result.Accepted != 2 || acks != 1 {
		t.Fatalf("seen=%v result=%+v acks=%d", seen, result, acks)
	}
	c, _ = s.Store.GetCursor(ctx, "conversations:"+owner)
	if c.Value != 1790704541000000000 {
		t.Fatal("64-bit version lost")
	}
	if _, err = s.SyncAccount(ctx, owner); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 3 || requests[0] != 1 || requests[1] != 3 || requests[2] != 3 {
		t.Fatalf("cursor resume %v", requests)
	}
}
