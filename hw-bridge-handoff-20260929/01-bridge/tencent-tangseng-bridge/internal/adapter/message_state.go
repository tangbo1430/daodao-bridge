package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type MessageExtra struct {
	ID            json.Number `json:"message_id"`
	IDString      string      `json:"message_id_str"`
	Revoke        int         `json:"revoke"`
	Revoker       string      `json:"revoker"`
	MutualDeleted int         `json:"is_mutual_deleted"`
	Readed        int         `json:"readed"`
	ReadCount     int         `json:"readed_count"`
	Version       int64       `json:"extra_version"`
}

func (v MessageExtra) MessageID() string {
	if v.IDString != "" {
		return v.IDString
	}
	return v.ID.String()
}
func (c *TangSengClient) MessageExtras(ctx context.Context, actor, channel string, ct int, version int64) ([]MessageExtra, error) {
	var out []MessageExtra
	err := c.UserRequest(ctx, actor, http.MethodPost, "/v1/message/extra/sync", map[string]any{"channel_id": channel, "channel_type": ct, "extra_version": version, "source": "hwbridge", "limit": 100}, &out)
	return out, err
}
func (c *TangSengClient) RecallMessage(ctx context.Context, actor, channel string, ct int, id, clientNo string) error {
	if id == "" || clientNo == "" {
		return fmt.Errorf("native message identifiers missing")
	}
	q := url.Values{"message_id": {id}, "client_msg_no": {clientNo}, "channel_id": {channel}, "channel_type": {strconv.Itoa(ct)}}
	return c.UserRequest(ctx, actor, http.MethodPost, "/v1/message/revoke?"+q.Encode(), nil, nil)
}
func (c *TangSengClient) ReadMessages(ctx context.Context, actor, channel string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return c.UserRequest(ctx, actor, http.MethodPost, "/v1/message/readed", map[string]any{"channel_id": channel, "channel_type": 1, "message_ids": ids}, nil)
}
func (c *TangSengClient) ClearUnread(ctx context.Context, actor, channel string, ct int) error {
	return c.UserRequest(ctx, actor, http.MethodPut, "/v1/coversation/clearUnread", map[string]any{"channel_id": channel, "channel_type": ct, "unread": 0}, nil)
}
func (c TencentClient) RecallMessage(ctx context.Context, v domain.MessageRecord) error {
	if v.TencentKey == "" {
		return fmt.Errorf("Tencent message key missing")
	}
	if v.ChannelType == 1 {
		return c.BusinessRequest(ctx, "openim", "admin_msgwithdraw", map[string]any{"From_Account": v.TencentFrom, "To_Account": v.TencentTo, "MsgKey": v.TencentKey}, nil)
	}
	i := strings.LastIndex(v.TencentKey, ":")
	if i < 0 {
		return fmt.Errorf("Tencent group message sequence missing")
	}
	seq, err := strconv.ParseInt(v.TencentKey[i+1:], 10, 64)
	if err != nil || seq <= 0 {
		return fmt.Errorf("Tencent group sequence invalid")
	}
	var out struct {
		Results []struct {
			Seq  int64 `json:"MsgSeq"`
			Code int   `json:"RetCode"`
		} `json:"RecallRetList"`
	}
	if err = c.GroupRequest(ctx, "group_msg_recall", map[string]any{"GroupId": v.TencentTo, "MsgSeqList": []any{map[string]any{"MsgSeq": seq}}}, &out); err != nil {
		return err
	}
	for _, v := range out.Results {
		if v.Seq == seq && v.Code == 0 {
			return nil
		}
	}
	return fmt.Errorf("Tencent group recall did not confirm message sequence %d", seq)
}
func (c TencentClient) MarkRead(ctx context.Context, v domain.MessageRecord) error {
	if v.ChannelType != 1 || v.TencentKey == "" || v.TencentFrom == "" || v.TencentTo == "" || v.OccurredAt <= 0 {
		return fmt.Errorf("C2C read receipt mapping incomplete")
	}
	// Clearing the conversation unread counter does not acknowledge an
	// individual message. Both APIs must succeed before recording completion.
	if err := c.BusinessRequest(ctx, "openim", "c2c_msg_read_receipt", map[string]any{
		"Operator_Account": v.TencentTo, "Peer_Account": v.TencentFrom,
		"C2CMsgInfo": []any{map[string]any{"From_Account": v.TencentFrom, "To_Account": v.TencentTo, "MsgKey": v.TencentKey}},
	}, nil); err != nil {
		return err
	}
	return c.BusinessRequest(ctx, "openim", "admin_set_msg_read", map[string]any{
		"Report_Account": v.TencentTo, "Peer_Account": v.TencentFrom, "MsgReadTime": v.OccurredAt,
	}, nil)
}
