package contract

import (
	_ "embed"
	"encoding/json"
	"strings"
	"testing"
)

// These are intentionally small redacted fixtures. They validate the shape
// of the contract harness; they do not claim that the remote protocol is
// confirmed. Real frames must replace them only after the Phase 0 checklist
// records the endpoint, time and source.

//go:embed tencent-c2c-after-send.json
var tencentFixture []byte

//go:embed wukong-connect.json
var connectFixture []byte

//go:embed wukong-recv.json
var recvFixture []byte

func TestRedactedFixturesHaveRequiredShapes(t *testing.T) {
	var tim struct {
		CallbackCommand string `json:"CallbackCommand"`
		SdkAppid        int    `json:"SdkAppid"`
		MsgBody         []struct {
			MsgType string `json:"MsgType"`
		} `json:"MsgBody"`
	}
	decode(t, tencentFixture, &tim)
	if tim.CallbackCommand != "C2C.CallbackAfterSendMsg" || tim.SdkAppid == 0 || len(tim.MsgBody) == 0 {
		t.Fatalf("Tencent fixture is incomplete: %+v", tim)
	}

	var connect struct {
		Method string `json:"method"`
		Params struct {
			UID        string `json:"uid"`
			DeviceFlag int    `json:"deviceFlag"`
			DeviceID   string `json:"deviceId"`
		} `json:"params"`
	}
	decode(t, connectFixture, &connect)
	if connect.Method != "connect" || connect.Params.UID == "" || connect.Params.DeviceFlag != 2 || connect.Params.DeviceID == "" {
		t.Fatalf("WuKong connect fixture is incomplete: %+v", connect)
	}

	var recv struct {
		Method string `json:"method"`
		Params struct {
			MessageID string `json:"messageId"`
			ChannelID string `json:"channelId"`
			FromUID   string `json:"fromUid"`
		} `json:"params"`
	}
	decode(t, recvFixture, &recv)
	if recv.Method != "recv" || recv.Params.MessageID == "" || recv.Params.ChannelID == "" || recv.Params.FromUID == "" {
		t.Fatalf("WuKong recv fixture is incomplete: %+v", recv)
	}

	for _, fixture := range [][]byte{tencentFixture, connectFixture, recvFixture} {
		text := strings.ToLower(string(fixture))
		for _, forbidden := range []string{"password", "secretkey", "usersig", "authorization"} {
			if strings.Contains(text, forbidden) && !strings.Contains(text, "redacted") {
				t.Fatalf("fixture appears to contain an unredacted credential: %q", forbidden)
			}
		}
	}
}

func decode(t *testing.T, data []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("invalid fixture JSON: %v", err)
	}
}
