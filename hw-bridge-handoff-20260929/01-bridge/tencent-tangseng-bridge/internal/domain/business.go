package domain

import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

// BusinessEvent shares the existing durable outbox with chat messages.
type BusinessEvent struct {
	Source          Source          `json:"source"`
	Command         string          `json:"command"`
	Owner           string          `json:"owner"`
	EventID         string          `json:"event_id"`
	NativeMessageID string          `json:"native_message_id,omitempty"`
	ChannelID       string          `json:"channel_id,omitempty"`
	ChannelType     int             `json:"channel_type,omitempty"`
	Payload         json.RawMessage `json:"payload"`
}

func ParseBusinessEvent(e EventEnvelope) (*BusinessEvent, error) {
	b := &BusinessEvent{Source: e.Source, Command: e.Command, EventID: e.EventID, Payload: e.Payload}
	if e.Source == SourceTencent {
		switch e.Command {
		case "C2C.CallbackAfterMsgWithDraw", "C2C.CallbackAfterMsgReport", "Group.CallbackAfterRecallMsg":
			return b, nil
		case "Sns.CallbackFriendAdd", "Sns.CallbackFriendDelete", "Sns.CallbackBlackListAdd", "Sns.CallbackBlackListDelete", "Profile.Sync", "Bridge.AccountDeleted":
			return b, nil
		case "Group.CallbackAfterNewMemberJoin", "Group.CallbackAfterMemberExit", "Group.CallbackAfterGroupDestroyed", "Group.CallbackAfterGroupInfoChanged", "Group.CallbackAfterChangeGroupOwner", "Group.CallbackAfterMemberFieldChanged":
			return b, nil
		}
		return nil, nil
	}
	var outer struct {
		MessageID   string          `json:"message_id"`
		To          string          `json:"to_uid"`
		ChannelID   string          `json:"channel_id"`
		ChannelType int             `json:"channel_type"`
		Payload     json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(e.Payload, &outer); err != nil {
		return nil, err
	}
	b.NativeMessageID = outer.MessageID
	var encoded string
	if json.Unmarshal(outer.Payload, &encoded) == nil {
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, err
		}
		outer.Payload = decoded
	}
	var command struct {
		Type          int    `json:"type"`
		Cmd           string `json:"cmd"`
		BridgeEventID string `json:"bridge_event_id"`
	}
	if len(outer.Payload) == 0 || json.Unmarshal(outer.Payload, &command) != nil {
		return nil, nil
	}
	// Locally generated role notices already have a Tencent system notification.
	// Let the normal message marker guard consume their PC(2) echoes as inbox only.
	if command.Type == 2000 && strings.HasPrefix(command.BridgeEventID, "bridge-group-role:") {
		return nil, nil
	}
	if ((command.Type >= 1001 && command.Type <= 1022) || command.Type == 2000) && outer.ChannelType == 2 {
		b.Command = "groupSystem"
		b.Owner = outer.To
		b.ChannelID = outer.ChannelID
		b.Payload = outer.Payload
		return b, nil
	}
	if command.Type != 99 {
		return nil, nil
	}
	switch command.Cmd {
	case "friendRequest", "friendAccept", "friendDeleted", "channelUpdate", "userUpdate", "userAvatarUpdate", "memberUpdate", "groupAvatarUpdate", "messageRevoke", "syncMessageExtra", "syncConversation", "syncMessage":
		b.Command = command.Cmd
		b.Owner = outer.To
		b.Payload = outer.Payload
		if command.Cmd == "messageRevoke" || command.Cmd == "syncMessageExtra" {
			// The deployed non-persistent command has no param object.
			// Its routing channel lives in the WuKong receive envelope.
			b.ChannelID = outer.ChannelID
			b.ChannelType = outer.ChannelType
		}
		return b, nil
	}
	return nil, nil
}

type FriendState struct {
	Owner         string `json:"owner"`
	Peer          string `json:"peer"`
	Status        string `json:"status"`
	Name          string `json:"name,omitempty"`
	Avatar        string `json:"avatar,omitempty"`
	Remark        string `json:"remark,omitempty"`
	TokenCipher   string `json:"token_cipher,omitempty"`
	VercodeCipher string `json:"vercode_cipher,omitempty"`
	RequestID     string `json:"request_id,omitempty"`
	ExpiresAt     int64  `json:"expires_at,omitempty"`
	UpdatedAt     int64  `json:"updated_at"`
	Blocked       bool   `json:"blocked"`
}
