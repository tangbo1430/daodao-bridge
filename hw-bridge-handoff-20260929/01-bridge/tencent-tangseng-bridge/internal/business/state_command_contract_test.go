package business

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
)

func TestNativeExtraCommandUsesEnvelopeAndRecoversOldJob(t *testing.T) {
	for _, oldJob := range []bool{false, true} {
		t.Run(fmt.Sprint(oldJob), func(t *testing.T) {
			s, _, owner, peer := fixture(t)
			ctx := context.Background()
			id, _ := s.owner(ctx, owner)
			raw, err := os.ReadFile("../../test/contract/message-extra-envelope.anonymized.json")
			if err != nil {
				t.Fatal(err)
			}
			raw = []byte(strings.ReplaceAll(strings.ReplaceAll(string(raw), "fixture-shadow", id.TargetID), "fixture-peer", peer))
			envelope := domain.EventEnvelope{Source: domain.SourceTang, Command: "msg.notify", EventID: "contract-state", IdempotencyKey: "contract-state", Payload: raw}
			_, _ = s.Store.AcceptInbox(ctx, envelope)
			event, err := domain.ParseBusinessEvent(envelope)
			if err != nil || event == nil || event.ChannelID != peer || event.ChannelType != 1 {
				t.Fatalf("routing %+v %v", event, err)
			}
			if oldJob {
				event.ChannelID = ""
				event.ChannelType = 0
			}
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				if r.URL.Path != "/v1/message/extra/sync" || body["channel_id"] != peer || body["channel_type"] != float64(1) || r.Header.Get("token") != "owner-fixture-token" {
					t.Errorf("request %s %+v", r.URL.Path, body)
				}
				fmt.Fprint(w, "[]")
			}))
			defer server.Close()
			s.Tang.BaseURL = server.URL
			if err = s.Handle(ctx, *event); err != nil {
				t.Fatal(err)
			}
			if requests != 1 {
				t.Fatalf("requests %d", requests)
			}
			if _, err = s.Store.FindTangInboxEvent(ctx, envelope.EventID, "another-owner"); err == nil {
				t.Fatal("routing crossed proxy owner")
			}
		})
	}
}

func TestMissingStateChannelIsStillRejected(t *testing.T) {
	s, calls, owner, _ := fixture(t)
	id, _ := s.owner(context.Background(), owner)
	err := s.Handle(context.Background(), domain.BusinessEvent{Source: domain.SourceTang, Command: "syncMessageExtra", Owner: id.TargetID, EventID: "missing", Payload: json.RawMessage(`{"cmd":"syncMessageExtra","type":99}`)})
	if err == nil || len(*calls) != 0 {
		t.Fatalf("guessed a missing channel: %v %+v", err, *calls)
	}
}
