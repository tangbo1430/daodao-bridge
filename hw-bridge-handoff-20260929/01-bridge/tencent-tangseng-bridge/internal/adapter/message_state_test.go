package adapter

import (
	"context"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGroupMessageSequenceAndRecallPerItemResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v4/group_open_http_svc/send_group_msg" {
			fmt.Fprint(w, `{"ActionStatus":"OK","ErrorCode":0,"MsgSeq":42}`)
			return
		}
		fmt.Fprint(w, `{"ActionStatus":"OK","ErrorCode":0,"RecallRetList":[{"MsgSeq":42,"RetCode":10007}]}`)
	}))
	defer server.Close()
	c := TencentClient{BaseURL: server.URL, SDKAppID: 1, SecretKey: "fixture-only", AdminAccount: "fixture-admin"}
	id, err := c.SendText(context.Background(), domain.MessageEvent{Source: domain.SourceTang, SourceMessageID: "native-id", ConversationType: domain.ConversationGroup, Content: "fixture"}, domain.Identity{TargetID: "peer"}, domain.Identity{TargetID: "dd_group"}, "marker")
	if err != nil || id != "dd_group:42" {
		t.Fatalf("id %s err %v", id, err)
	}
	if c.RecallMessage(context.Background(), domain.MessageRecord{TencentKey: id, TencentTo: "dd_group", ChannelType: 2}) == nil {
		t.Fatal("nested permission refusal reported success")
	}
}
