package business

import (
	"context"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCommandQueueResetDoesNotBlockNewMessages(t *testing.T) {
	s, _, owner, _ := fixture(t)
	ctx := context.Background()
	id, _ := s.owner(ctx, owner)
	_ = s.Store.PutCursor(ctx, domain.SyncCursor{Key: "commands:" + owner, Value: 8})
	accepted, acked := 0, 0
	s.Accept = func(context.Context, domain.EventEnvelope) (bool, error) { accepted++; return true, nil }
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/message/sync":
			fmt.Fprintf(w, `[{"message_idstr":"900000000000000032","message_seq":2,"from_uid":"fixture-peer","channel_id":"%s","channel_type":1,"timestamp":%d,"payload":{"type":99,"cmd":"syncMessageExtra"}}]`, id.TargetID, time.Now().Unix())
		case "/v1/message/syncack/2":
			acked++
			fmt.Fprint(w, `{}`)
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	s.Tang.BaseURL = srv.URL
	n, seq, err := s.syncCommands(ctx, id, time.Now().Unix()-60)
	if err != nil || n != 1 || seq != 2 || accepted != 1 || acked != 1 {
		t.Fatalf("count=%d seq=%d accepted=%d acked=%d error=%v", n, seq, accepted, acked, err)
	}
	c, _ := s.Store.GetCursor(ctx, "commands:"+owner)
	if c.Value != 2 {
		t.Fatal(c)
	}
}
