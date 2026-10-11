package business

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecallActorPermissionAndDuplicate(t *testing.T) {
	s, calls, owner, peer := fixture(t)
	ctx := context.Background()
	id, _ := s.owner(ctx, owner)
	v := domain.MessageRecord{Key: "fixture", Source: domain.SourceTencent, TencentKey: "key", TencentFrom: owner, TencentTo: peer, TangID: "2104980470256111616", TangClientNo: "hwb_fixture", TangFrom: id.TargetID, TangTo: peer, ChannelType: 1}
	_ = s.Store.PutMessageRecord(ctx, v)
	if err := s.Recall(ctx, owner, "key"); err != nil {
		t.Fatal(err)
	}
	if err := s.Recall(ctx, owner, "key"); err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 1 || (*calls)[0].Path != "/v1/message/revoke" || (*calls)[0].Token != "owner-fixture-token" {
		t.Fatalf("calls %+v", *calls)
	}
	v.Key = "other"
	v.TencentKey = "other-key"
	v.TencentFrom = peer
	_ = s.Store.PutMessageRecord(ctx, v)
	if s.Recall(ctx, owner, "other-key") == nil {
		t.Fatal("recalled someone else's C2C message")
	}
	if s.Recall(ctx, owner, "missing") == nil {
		t.Fatal("missing mapping reported success")
	}
}
func TestReadUsesRecipientTokenAndOnlyReadTimeBoundary(t *testing.T) {
	s, calls, owner, peer := fixture(t)
	ctx := context.Background()
	id, _ := s.owner(ctx, owner)
	for i := int64(1); i <= 2; i++ {
		_ = s.Store.PutMessageRecord(ctx, domain.MessageRecord{NeedReceipt: true, Key: fmt.Sprint(i), Source: domain.SourceTang, TencentKey: fmt.Sprint(i), TencentFrom: peer, TencentTo: owner, TangID: fmt.Sprint(i), TangFrom: peer, TangTo: id.TargetID, ChannelType: 1, OccurredAt: i * 100})
	}
	if err := s.Read(ctx, owner, peer, 100); err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 2 || (*calls)[0].Token != "owner-fixture-token" || (*calls)[0].Path != "/v1/message/readed" || (*calls)[1].Path != "/v1/coversation/clearUnread" {
		t.Fatalf("calls %+v", *calls)
	}
	ids := (*calls)[0].Body["message_ids"].([]any)
	if len(ids) != 1 || ids[0] != "1" {
		t.Fatal("marked newer message read")
	}
	v, _ := s.Store.FindMessageRecord(ctx, domain.SourceTang, "2")
	if v.ReadSynced {
		t.Fatal("newer receipt marked delivered")
	}
}
func TestExtraSyncRecallReadCursorAndFailure(t *testing.T) {
	s, _, owner, peer := fixture(t)
	ctx := context.Background()
	id, _ := s.owner(ctx, owner)
	_ = s.Store.PutMessageRecord(ctx, domain.MessageRecord{Key: "incoming", Source: domain.SourceTang, TencentKey: "native-key", TencentFrom: peer, TencentTo: owner, TangID: "2104980470256111616", TangFrom: peer, TangTo: id.TargetID, ChannelType: 1})
	_ = s.Store.PutMessageRecord(ctx, domain.MessageRecord{Key: "outgoing", Source: domain.SourceTencent, TencentKey: "key2", TencentFrom: owner, TencentTo: peer, TangID: "2104980470256111617", TangFrom: id.TargetID, TangTo: peer, ChannelType: 1, NeedReceipt: true, OccurredAt: 100})
	failed := true
	recalls, reads := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1/message/extra/sync" {
			if r.Header.Get("token") != "owner-fixture-token" {
				t.Error("wrong actor")
			}
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["source"] != "hwbridge" {
				t.Error("shared another client's extra cursor")
			}
			fmt.Fprint(w, `[{"message_id":2104980470256111616,"revoke":1,"extra_version":2},{"message_id_str":"2104980470256111617","readed_count":1,"extra_version":3}]`)
			return
		}
		if strings.HasSuffix(r.URL.Path, "admin_msgwithdraw") {
			recalls++
		}
		if strings.HasSuffix(r.URL.Path, "admin_set_msg_read") {
			reads++
		}
		if failed {
			fmt.Fprint(w, `{"ActionStatus":"FAIL","ErrorCode":10007}`)
			return
		}
		fmt.Fprint(w, `{"ActionStatus":"OK","ErrorCode":0}`)
	}))
	defer server.Close()
	s.Tang.BaseURL = server.URL
	s.Tencent.BaseURL = server.URL
	if _, err := s.SyncMessageExtras(ctx, owner, peer, 1); err == nil {
		t.Fatal("upstream refusal reported success")
	}
	if _, err := s.Store.GetCursor(ctx, "extra:"+owner+":1:"+peer); err != store.ErrNotFound {
		t.Fatal("advanced cursor before applying change")
	}
	failed = false
	if n, err := s.SyncMessageExtras(ctx, owner, peer, 1); err != nil || n != 2 {
		t.Fatalf("n %d err %v", n, err)
	}
	cursor, _ := s.Store.GetCursor(ctx, "extra:"+owner+":1:"+peer)
	if cursor.Value != 3 {
		t.Fatalf("cursor %+v", cursor)
	}
	if recalls != 2 || reads != 1 {
		t.Fatalf("calls %d %d", recalls, reads)
	}
}
