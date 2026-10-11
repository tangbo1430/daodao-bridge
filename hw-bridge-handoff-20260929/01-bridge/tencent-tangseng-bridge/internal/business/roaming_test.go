package business

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTencentC2CRoamingPaginationOrderAndInboxFailure(t *testing.T) {
	s, _, owner, peer := fixture(t)
	ctx := context.Background()
	started := time.Now().Unix() - 15
	received := []string{}
	fail := true
	requests := 0
	s.Accept = func(_ context.Context, e domain.EventEnvelope) (bool, error) {
		m, err := domain.ParseTencentMessage(e.Payload)
		if err != nil {
			return false, err
		}
		if m.SourceMessageID == "fixture-new" && fail {
			return false, fmt.Errorf("fixture database failure")
		}
		received = append(received, m.SourceMessageID)
		return true, nil
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		requests++
		if body["Operator_Account"] != owner || body["Peer_Account"] != peer {
			t.Error("wrong conversation")
		}
		items := []any{}
		key, when, complete := "fixture-new", started+5, 0
		if body["LastMsgKey"] == "fixture-new" {
			key, when, complete = "fixture-old", started+1, 1
			if body["MaxTime"] != float64(started+5) {
				t.Error("continuation ignored LastMsgTime")
			}
		}
		items = append(items, map[string]any{"From_Account": owner, "To_Account": peer, "MsgKey": key, "MsgSeq": when, "MsgTimeStamp": when, "MsgBody": []any{map[string]any{"MsgType": "TIMTextElem", "MsgContent": map[string]string{"Text": "fixture"}}}})
		_ = json.NewEncoder(w).Encode(map[string]any{"ActionStatus": "OK", "Complete": complete, "LastMsgTime": when, "LastMsgKey": key, "MsgList": items})
	}))
	defer srv.Close()
	s.Tencent.BaseURL = srv.URL
	if _, err := s.roamC2C(ctx, owner, peer, started); err == nil {
		t.Fatal("inbox failure concealed")
	}
	if _, err := s.Store.GetCursor(ctx, "tim-c2c:"+domain.StableID(owner, peer)); err == nil {
		t.Fatal("cursor committed before all acceptance")
	}
	if len(received) != 1 || received[0] != "fixture-old" {
		t.Fatalf("delivery order %v", received)
	}
	fail = false
	received = nil
	if _, err := s.roamC2C(ctx, owner, peer, started); err != nil {
		t.Fatal(err)
	}
	if len(received) != 2 || received[0] != "fixture-old" || received[1] != "fixture-new" || requests != 4 {
		t.Fatalf("received=%v requests=%d", received, requests)
	}
}

func TestReadReceiptDisabledOnlyClearsUnread(t *testing.T) {
	s, calls, owner, peer := fixture(t)
	ctx := context.Background()
	r, _ := s.Accounts.Profile(ctx, owner)
	_ = s.Store.PutMessageRecord(ctx, domain.MessageRecord{Key: "fixture-disabled-receipt", TencentFrom: peer, TencentTo: owner, TangFrom: peer, TangTo: r.Identity.TargetID, TangID: "fixture-message", ChannelType: 1, OccurredAt: 100, Source: domain.SourceTang})
	if err := s.Read(ctx, owner, peer, 100); err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 1 || (*calls)[0].Path != "/v1/coversation/clearUnread" {
		t.Fatalf("receipt-disabled message reported read: %+v", *calls)
	}
}
