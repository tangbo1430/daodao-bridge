package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// TangMessage preserves 64-bit IDs and the original payload for the normal Inbox.
type TangMessage struct {
	Setting     uint8           `json:"setting"`
	ID          json.Number     `json:"message_id"`
	IDString    string          `json:"message_idstr"`
	Seq         int64           `json:"message_seq"`
	ClientNo    string          `json:"client_msg_no"`
	From        string          `json:"from_uid"`
	To          string          `json:"to_uid,omitempty"`
	Channel     string          `json:"channel_id"`
	ChannelType int             `json:"channel_type"`
	Timestamp   int64           `json:"timestamp"`
	Deleted     int             `json:"is_deleted"`
	Revoke      int             `json:"revoke"`
	Extra       *MessageExtra   `json:"message_extra,omitempty"`
	Payload     json.RawMessage `json:"payload"`
}

func (m TangMessage) MessageID() string {
	if m.IDString != "" {
		return m.IDString
	}
	return m.ID.String()
}

type TangConversation struct {
	Channel   string        `json:"channel_id"`
	Type      int           `json:"channel_type"`
	Unread    int           `json:"unread"`
	Mute      int           `json:"mute"`
	Stick     int           `json:"stick"`
	Timestamp int64         `json:"timestamp"`
	LastSeq   int64         `json:"last_msg_seq"`
	OffsetSeq int64         `json:"offset_msg_seq"`
	Version   int64         `json:"version"`
	Recents   []TangMessage `json:"recents"`
}
type TangConversations struct {
	UID           string             `json:"uid"`
	Conversations []TangConversation `json:"conversations"`
}

func (c *TangSengClient) SyncConversations(ctx context.Context, actor string, version int64, seqs, device string) (TangConversations, error) {
	var out TangConversations
	err := c.UserRequest(ctx, actor, http.MethodPost, "/v1/conversation/sync", map[string]any{"version": version, "last_msg_seqs": seqs, "msg_count": 20, "device_uuid": device}, &out)
	if err == nil && out.UID != actor {
		err = fmt.Errorf("conversation response actor mismatch")
	}
	return out, err
}
func (c *TangSengClient) AckConversations(ctx context.Context, actor, device string, cmdVersion int64) error {
	return c.UserRequest(ctx, actor, http.MethodPost, "/v1/conversation/syncack", map[string]any{"cmd_version": cmdVersion, "device_uuid": device}, nil)
}

type TangChannelMessages struct {
	Start    int64         `json:"start_message_seq"`
	End      int64         `json:"end_message_seq"`
	PullMode int           `json:"pull_mode"`
	More     int           `json:"more"`
	Messages []TangMessage `json:"messages"`
}

func (c *TangSengClient) SyncChannel(ctx context.Context, actor, channel string, ct int, start, end int64, device string) (TangChannelMessages, error) {
	var out TangChannelMessages
	err := c.UserRequest(ctx, actor, http.MethodPost, "/v1/message/channel/sync", map[string]any{"channel_id": channel, "channel_type": ct, "start_message_seq": start, "end_message_seq": end, "limit": 100, "pull_mode": 1, "device_uuid": device}, &out)
	return out, err
}

func (c *TangSengClient) SyncCommands(ctx context.Context, actor string, seq int64) ([]TangMessage, error) {
	var out []TangMessage
	err := c.UserRequest(ctx, actor, http.MethodPost, "/v1/message/sync", map[string]any{"max_message_seq": seq, "limit": 100}, &out)
	return out, err
}
func (c *TangSengClient) AckCommands(ctx context.Context, actor string, seq int64) error {
	return c.UserRequest(ctx, actor, http.MethodPost, "/v1/message/syncack/"+strconv.FormatInt(seq, 10), nil, nil)
}

func (c *TangSengClient) LatestChannel(ctx context.Context, actor, channel string, ct int, device string) (TangChannelMessages, error) {
	var out TangChannelMessages
	err := c.UserRequest(ctx, actor, http.MethodPost, "/v1/message/channel/sync", map[string]any{"channel_id": channel, "channel_type": ct, "start_message_seq": 0, "end_message_seq": 0, "limit": 1, "pull_mode": 0, "device_uuid": device}, &out)
	return out, err
}
