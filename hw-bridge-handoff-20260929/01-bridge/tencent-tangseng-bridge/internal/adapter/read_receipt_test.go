package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
)

func TestReadReceiptUsesMessageAPIAndBoundsUnreadClearing(t *testing.T) {
	raw, err := os.ReadFile("../../test/contract/tencent-read-receipt.anonymized.json")
	if err != nil {
		t.Fatal(err)
	}
	var contract struct {
		Request map[string]any `json:"receipt_request"`
	}
	if err = json.Unmarshal(raw, &contract); err != nil {
		t.Fatal(err)
	}
	for _, failure := range []string{"", "receipt", "unread"} {
		t.Run(failure, func(t *testing.T) {
			calls := []string{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				command := "unread"
				if strings.HasSuffix(r.URL.Path, "c2c_msg_read_receipt") {
					command = "receipt"
					if !reflect.DeepEqual(body, contract.Request) {
						t.Errorf("receipt body %+v", body)
					}
				} else if strings.HasSuffix(r.URL.Path, "admin_set_msg_read") {
					if body["Report_Account"] != "fixture-peer" || body["Peer_Account"] != "fixture-owner" || body["MsgReadTime"] != float64(1790776228) {
						t.Errorf("unread boundary %+v", body)
					}
				} else {
					t.Errorf("unexpected endpoint %s", r.URL.Path)
				}
				calls = append(calls, command)
				if command == failure {
					fmt.Fprint(w, `{"ActionStatus":"FAIL","ErrorCode":20001}`)
				} else {
					fmt.Fprint(w, `{"ActionStatus":"OK","ErrorCode":0}`)
				}
			}))
			defer server.Close()
			client := TencentClient{SDKAppID: 1, SecretKey: "fixture-only", AdminAccount: "fixture-admin", BaseURL: server.URL}
			record := domain.MessageRecord{TencentFrom: "fixture-owner", TencentTo: "fixture-peer", TencentKey: "111_222_1790776228", ChannelType: 1, OccurredAt: 1790776228}
			err := client.MarkRead(context.Background(), record)
			if (err != nil) != (failure != "") {
				t.Fatalf("failure=%s err=%v", failure, err)
			}
			want := []string{"receipt", "unread"}
			if failure == "receipt" {
				want = []string{"receipt"}
			}
			if !reflect.DeepEqual(calls, want) {
				t.Fatalf("calls %+v", calls)
			}
			calls = nil
			record.ChannelType = 2
			if client.MarkRead(context.Background(), record) == nil || len(calls) != 0 {
				t.Fatal("group used C2C receipt endpoint")
			}
		})
	}
}
