package business

import (
	"context"
	"encoding/json"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"os"
	"strings"
	"testing"
)

func TestObservedGroupSettingNoticesDoNotDuplicateSDKTips(t *testing.T) {
	s, f, owner, _ := groupService(t)
	ctx := context.Background()
	id, _ := s.owner(ctx, owner)
	raw, err := os.ReadFile("../../test/contract/group-update-observed-20261002.json")
	if err != nil {
		t.Fatal(err)
	}
	var observed struct {
		Payloads []json.RawMessage `json:"payloads"`
	}
	if err = json.Unmarshal(raw, &observed); err != nil {
		t.Fatal(err)
	}
	for i, payload := range observed.Payloads {
		e := domain.BusinessEvent{Source: domain.SourceTang, Command: "groupSystem", Owner: id.TargetID, ChannelID: "fixture-group", EventID: "proxy-a", NativeMessageID: string(rune('a' + i)), Payload: payload}
		if err = s.Handle(ctx, e); err != nil {
			t.Fatal(err)
		}
		e.EventID = "proxy-b"
		if err = s.Handle(ctx, e); err != nil {
			t.Fatal(err)
		}
	}
	notices := 0
	for _, c := range f.calls {
		if c.Path != "/v4/group_open_http_svc/send_group_msg" {
			continue
		}
		notices++
		b, _ := json.Marshal(c.Body)
		if !strings.Contains(string(b), "bridge_tip") || strings.Contains(string(b), "{0}") || strings.Contains(string(b), "开启了群禁言") || strings.Contains(string(b), "修改群名") {
			t.Fatalf("unexpected notice: %s", b)
		}
	}
	if notices != 2 {
		t.Fatalf("expected exactly two invitation setting notices, got %d", notices)
	}
}

func TestPendingGroupInvitationOnlyNotifiesVisibleHuaweiUser(t *testing.T) {
	s, f, owner, peer := groupService(t)
	ctx := context.Background()
	id, _ := s.owner(ctx, owner)
	if _, err := s.Group(ctx, "create", GroupRequest{Owner: owner, Members: []string{peer}, RequestID: "notice-fixture"}); err != nil {
		t.Fatal(err)
	}
	for _, visible := range []string{"another-user", id.TargetID} {
		before := len(f.calls)
		payload, _ := json.Marshal(map[string]any{"type": 1009, "visibles": []string{visible}, "invite_no": "fixture-invite"})
		if err := s.Handle(ctx, domain.BusinessEvent{Source: domain.SourceTang, Command: "groupSystem", ChannelID: "fixture-group", Owner: id.TargetID, EventID: "notice-" + visible, Payload: payload}); err != nil {
			t.Fatal(err)
		}
		notices := 0
		for _, c := range f.calls[before:] {
			if c.Path == "/v4/openim/sendmsg" {
				notices++
				if c.Body["To_Account"] != owner {
					t.Fatal("notice sent to wrong recipient")
				}
				b, _ := json.Marshal(c.Body)
				if string(b) == "" {
					t.Fatal("empty notice")
				}
			}
		}
		want := 0
		if visible == id.TargetID {
			want = 1
		}
		if notices != want {
			t.Fatalf("visible=%s notices=%d", visible, notices)
		}
	}
}
