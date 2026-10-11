package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Source string

const (
	SourceTencent Source = "tencent"
	SourceTang    Source = "tangseng"
)

type EventEnvelope struct {
	EventID        string          `json:"event_id"`
	TraceID        string          `json:"trace_id"`
	Source         Source          `json:"source"`
	Command        string          `json:"command"`
	OccurredAt     int64           `json:"occurred_at"`
	IdempotencyKey string          `json:"idempotency_key"`
	Payload        json.RawMessage `json:"payload"`
}

type ConversationType string

const (
	ConversationC2C   ConversationType = "c2c"
	ConversationGroup ConversationType = "group"
)

type MessageEvent struct {
	FailureNotice    *FailureNotice   `json:"failure_notice,omitempty"`
	Mention          *Mention         `json:"mention,omitempty"`
	Reply            *ReplyReference  `json:"reply,omitempty"`
	SourceSDKID      string           `json:"source_sdk_id,omitempty"`
	MirrorSender     string           `json:"mirror_sender,omitempty"`
	NeedReceipt      bool             `json:"need_receipt,omitempty"`
	SourceSeq        int64            `json:"source_seq,omitempty"`
	ClientMsgNo      string           `json:"client_msg_no,omitempty"`
	Parts            []MessagePart    `json:"parts,omitempty"`
	Business         *BusinessEvent   `json:"business,omitempty"`
	Source           Source           `json:"source"`
	SourceMessageID  string           `json:"source_message_id"`
	ConversationID   string           `json:"conversation_id"`
	ConversationType ConversationType `json:"conversation_type"`
	FromAccount      string           `json:"from_account"`
	ToAccount        string           `json:"to_account,omitempty"`
	Content          string           `json:"content"`
	OccurredAt       int64            `json:"occurred_at"`
	BridgeMarker     string           `json:"bridge_marker,omitempty"`
}

type Identity struct {
	Source      Source `json:"source"`
	SourceID    string `json:"source_id"`
	Target      Source `json:"target"`
	TargetID    string `json:"target_id"`
	EntityType  string `json:"entity_type,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	Shadow      bool   `json:"shadow"`
}

type MessageMapping struct {
	Source          Source `json:"source"`
	SourceMessageID string `json:"source_message_id"`
	Target          Source `json:"target"`
	TargetMessageID string `json:"target_message_id"`
	Status          string `json:"status"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
}

type OutboxJob struct {
	ID            string       `json:"id"`
	EventID       string       `json:"event_id"`
	Target        Source       `json:"target"`
	Message       MessageEvent `json:"message"`
	Status        string       `json:"status"`
	Attempts      int          `json:"attempts"`
	NextAttemptAt int64        `json:"next_attempt_at"`
	LastError     string       `json:"last_error,omitempty"`
	CreatedAt     int64        `json:"created_at"`
	UpdatedAt     int64        `json:"updated_at"`
}

func NewEventID() string {
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}

func NewTraceID() string {
	return fmt.Sprintf("tr_%d", time.Now().UnixNano())
}

func StableID(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func ShadowID(source Source, id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	prefix := "tx_"
	if source == SourceTang {
		prefix = "dd_"
	}
	if strings.HasPrefix(id, prefix) {
		return id
	}
	shadow := prefix + id
	if len(shadow) > 32 {
		shadow = shadow[:32]
	}
	return shadow
}

func MessageKey(source Source, command, messageID string) string {
	if messageID == "" {
		messageID = StableID(command, strconv.FormatInt(time.Now().UnixNano(), 10))
	}
	return string(source) + ":" + command + ":" + messageID
}
