package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
)

type TencentRoamMessage struct {
	From        string          `json:"From_Account"`
	To          string          `json:"To_Account"`
	Key         string          `json:"MsgKey"`
	Seq         int64           `json:"MsgSeq"`
	Random      int64           `json:"MsgRandom"`
	Time        int64           `json:"MsgTimeStamp"`
	Flags       int             `json:"MsgFlagBits"`
	Place       int             `json:"IsPlaceMsg"`
	System      int             `json:"IsSystemMsg"`
	Read        int             `json:"IsPeerRead"`
	NeedReceipt int             `json:"IsNeedReadReceipt"`
	Body        json.RawMessage `json:"MsgBody"`
	Custom      string          `json:"CloudCustomData"`
}

func (m TencentRoamMessage) Envelope(group string) (domain.EventEnvelope, error) {
	command := "C2C.CallbackAfterSendMsg"
	key := m.Key
	if group != "" {
		command = "Group.CallbackAfterSendMsg"
		key = fmt.Sprintf("%s:%d", group, m.Seq)
	}
	if key == "" || m.From == "" || m.Time <= 0 {
		return domain.EventEnvelope{}, fmt.Errorf("roaming message identifiers missing")
	}
	b, err := json.Marshal(map[string]any{"CallbackCommand": command, "From_Account": m.From, "To_Account": m.To, "GroupId": group, "MsgKey": m.Key, "MsgSeq": m.Seq, "MsgRandom": m.Random, "MsgTime": m.Time, "MsgBody": m.Body, "CloudCustomData": m.Custom})
	return domain.EventEnvelope{Source: domain.SourceTencent, Command: command, EventID: domain.StableID("tim-event", command, key), IdempotencyKey: domain.MessageKey(domain.SourceTencent, command, key), OccurredAt: m.Time, Payload: b}, err
}

type TencentC2CRoam struct {
	Complete int                  `json:"Complete"`
	LastTime int64                `json:"LastMsgTime"`
	LastKey  string               `json:"LastMsgKey"`
	Messages []TencentRoamMessage `json:"MsgList"`
}

func (c TencentClient) RoamC2C(ctx context.Context, owner, peer string, min, max int64, last string) (TencentC2CRoam, error) {
	var out TencentC2CRoam
	body := map[string]any{"Operator_Account": owner, "Peer_Account": peer, "MinTime": min, "MaxTime": max, "MaxCnt": 100}
	if last != "" {
		body["LastMsgKey"] = last
	}
	err := c.BusinessRequest(ctx, "openim", "admin_getroammsg", body, &out)
	return out, err
}

type TencentGroupRoam struct {
	Group    string               `json:"GroupId"`
	Finished int                  `json:"IsFinished"`
	Messages []TencentRoamMessage `json:"RspMsgList"`
}

func (c TencentClient) RoamGroup(ctx context.Context, group string, maxSeq int64) (TencentGroupRoam, error) {
	var out TencentGroupRoam
	body := map[string]any{"GroupId": group, "ReqMsgNumber": 20, "WithRecalledMsg": 1}
	if maxSeq > 0 {
		body["ReqMsgSeq"] = maxSeq
	}
	err := c.GroupRequest(ctx, "group_msg_get_simple", body, &out)
	if err == nil && out.Group != group {
		err = fmt.Errorf("group history response mismatch")
	}
	return out, err
}
