package business

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
)

func TestExplicitReceiptFailureKeepsCursorAndLegacyFlagDoesNotSkip(t *testing.T) {
	s, _, owner, peer := fixture(t)
	ctx := context.Background()
	id, _ := s.owner(ctx, owner)
	record := domain.MessageRecord{Key: "out", Source: domain.SourceTencent, TencentKey: "1_2_100", TencentFrom: owner, TencentTo: peer, TangID: "123", TangFrom: id.TargetID, TangTo: peer, ChannelType: 1, OccurredAt: 100, NeedReceipt: true, ReadSynced: true}
	_ = s.Store.PutMessageRecord(ctx, record)
	failReceipt := true
	receipts, clears := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/message/extra/sync" {
			var p struct {
				Version int64 `json:"extra_version"`
			}
			_ = json.NewDecoder(r.Body).Decode(&p)
			if p.Version >= 7 {
				fmt.Fprint(w, `[]`)
			} else {
				fmt.Fprint(w, `[{"message_id_str":"123","readed_count":1,"extra_version":7}]`)
			}
			return
		}
		if strings.HasSuffix(r.URL.Path, "c2c_msg_read_receipt") {
			receipts++
			if failReceipt {
				fmt.Fprint(w, `{"ActionStatus":"FAIL","ErrorCode":20001}`)
				return
			}
		}
		if strings.HasSuffix(r.URL.Path, "admin_set_msg_read") {
			clears++
		}
		fmt.Fprint(w, `{"ActionStatus":"OK","ErrorCode":0}`)
	}))
	defer server.Close()
	s.Tang.BaseURL = server.URL
	s.Tencent.BaseURL = server.URL
	if _, err := s.SyncMessageExtras(ctx, owner, peer, 1); err == nil {
		t.Fatal("receipt refusal hidden")
	}
	v, _ := s.Store.FindMessageRecord(ctx, domain.SourceTang, "123")
	if v.TencentReceiptSynced || clears != 0 {
		t.Fatal("failed receipt recorded or unread counter cleared")
	}
	if _, err := s.Store.GetCursor(ctx, "extra:"+owner+":1:"+peer); err != store.ErrNotFound {
		t.Fatal("cursor advanced on failed receipt")
	}
	failReceipt = false
	if n, err := s.SyncMessageExtras(ctx, owner, peer, 1); err != nil || n != 1 {
		t.Fatalf("retry %d %v", n, err)
	}
	v, _ = s.Store.FindMessageRecord(ctx, domain.SourceTang, "123")
	if !v.ReadSynced || !v.TencentReceiptSynced {
		t.Fatal("explicit receipt completion missing")
	}
	if n, err := s.SyncMessageExtras(ctx, owner, peer, 1); err != nil || n != 0 || receipts != 2 || clears != 1 {
		t.Fatalf("duplicate %d %v calls %d/%d", n, err, receipts, clears)
	}
}

func TestDisabledReceiptNeverSendsTencentReceipt(t *testing.T) {
	s, _, owner, peer := fixture(t)
	ctx := context.Background()
	id, _ := s.owner(ctx, owner)
	_ = s.Store.PutMessageRecord(ctx, domain.MessageRecord{Key: "off", Source: domain.SourceTencent, TencentKey: "key", TencentFrom: owner, TencentTo: peer, TangID: "123", TangFrom: id.TargetID, TangTo: peer, ChannelType: 1, OccurredAt: 100, NeedReceipt: false})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/message/extra/sync" {
			t.Errorf("unexpected receipt API %s", r.URL.Path)
		}
		fmt.Fprint(w, `[{"message_id_str":"123","readed_count":1,"extra_version":7}]`)
	}))
	defer server.Close()
	s.Tang.BaseURL = server.URL
	s.Tencent.BaseURL = server.URL
	if n, err := s.SyncMessageExtras(ctx, owner, peer, 1); err != nil || n != 0 {
		t.Fatalf("disabled %d %v", n, err)
	}
}
