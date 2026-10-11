package domain

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrIgnoredMessage = errors.New("message is control-only")

type tencentElement struct {
	MsgType    string `json:"MsgType"`
	MsgContent struct {
		Text            string `json:"Text"`
		CloudCustomData string `json:"CloudCustomData"`
		Index           int    `json:"Index"`
		Data            string `json:"Data"`
		Desc            string `json:"Desc"`
		Ext             string `json:"Ext"`
	} `json:"MsgContent"`
}

type tencentPayload struct {
	CallbackCommand string           `json:"CallbackCommand"`
	SdkAppid        any              `json:"SdkAppid"`
	FromAccount     string           `json:"From_Account"`
	ToAccount       string           `json:"To_Account"`
	GroupID         string           `json:"GroupId"`
	MsgKey          string           `json:"MsgKey"`
	MsgSeq          int64            `json:"MsgSeq"`
	MsgRandom       int64            `json:"MsgRandom"`
	MsgTime         int64            `json:"MsgTime"`
	EventTime       int64            `json:"EventTime"`
	MsgBody         []tencentElement `json:"MsgBody"`
	CloudCustomData string           `json:"CloudCustomData"`
}

type tangPayload struct {
	NeedReceipt   bool            `json:"need_receipt"`
	MessageSeq    int64           `json:"message_seq"`
	EventType     string          `json:"event_type"`
	MessageID     any             `json:"message_id"`
	MsgID         string          `json:"msg_id"`
	ClientMsgNo   string          `json:"client_msg_no"`
	FromUID       string          `json:"from_uid"`
	ToUID         string          `json:"to_uid"`
	ChannelID     string          `json:"channel_id"`
	ChannelType   int             `json:"channel_type"`
	Content       string          `json:"content"`
	Text          string          `json:"text"`
	OccurredAt    int64           `json:"occurred_at"`
	BridgeSource  string          `json:"bridge_source"`
	BridgeEventID string          `json:"bridge_event_id"`
	Payload       json.RawMessage `json:"payload"`
}

func ParseTencentMessage(payload json.RawMessage) (MessageEvent, error) {
	var p tencentPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return MessageEvent{}, fmt.Errorf("decode Tencent payload: %w", err)
	}
	if p.CallbackCommand != "C2C.CallbackAfterSendMsg" && p.CallbackCommand != "Group.CallbackAfterSendMsg" {
		return MessageEvent{}, fmt.Errorf("unsupported Tencent command %q", p.CallbackCommand)
	}
	if p.FromAccount == "" {
		return MessageEvent{}, fmt.Errorf("Tencent From_Account is required")
	}
	if len(p.MsgBody) == 0 {
		return MessageEvent{}, fmt.Errorf("Tencent MsgBody is empty")
	}
	onlyTyping := true
	for _, elem := range p.MsgBody {
		var custom struct {
			BusinessID string `json:"businessID"`
		}
		if elem.MsgType != "TIMCustomElem" || json.Unmarshal([]byte(elem.MsgContent.Data), &custom) != nil || custom.BusinessID != "user_typing_status" {
			onlyTyping = false
		}
	}
	if onlyTyping {
		return MessageEvent{}, ErrIgnoredMessage
	}
	var text strings.Builder
	marker := p.CloudCustomData
	for _, element := range p.MsgBody {
		switch element.MsgType {
		case "TIMTextElem":
			text.WriteString(element.MsgContent.Text)
			if element.MsgContent.CloudCustomData != "" {
				marker = element.MsgContent.CloudCustomData
			}
		case "TIMFaceElem":
			text.WriteString(normalizeEmojiText(faceContent(element.MsgContent.Index, element.MsgContent.Data)))
		case "TIMCustomElem":
			text.WriteString(normalizeEmojiText(customContent(element.MsgContent.Data, element.MsgContent.Desc, element.MsgContent.Ext)))
		default:
			// Do not silently drop a message whose media converter is not yet
			// enabled.  It remains durable and visible to the peer as a clear
			// placeholder until the media phase is deployed.
			text.WriteString("[暂不支持的消息类型:" + element.MsgType + "]")
		}
	}
	content := normalizeEmojiText(text.String())
	parts := TencentParts(payload)
	if len(parts) > 0 {
		content = PartsSummary(parts)
	}
	if content == "" {
		return MessageEvent{}, fmt.Errorf("Tencent event contains no text element")
	}
	messageID := p.MsgKey
	if p.GroupID != "" && p.MsgSeq > 0 {
		messageID = fmt.Sprintf("%s:%d", p.GroupID, p.MsgSeq)
	}
	if messageID == "" {
		messageID = fmt.Sprintf("%d:%d", p.MsgSeq, p.MsgRandom)
	}
	if messageID == ":" || messageID == "0:0" {
		h := sha256.Sum256(payload)
		messageID = hex.EncodeToString(h[:])
	}
	conversationType := ConversationC2C
	conversationID := p.ToAccount
	if p.GroupID != "" || p.CallbackCommand == "Group.CallbackAfterSendMsg" {
		conversationType = ConversationGroup
		conversationID = p.GroupID
	}
	if conversationID == "" {
		return MessageEvent{}, fmt.Errorf("Tencent conversation target is required")
	}
	when := p.MsgTime
	if when == 0 {
		when = p.EventTime
		if when > 100000000000 {
			when /= 1000
		}
	}
	if when == 0 {
		when = time.Now().Unix()
	}
	mention, reply, sdkID := TencentMetadata(payload)
	return MessageEvent{
		Mention: mention, Reply: reply, SourceSDKID: sdkID,
		SourceSeq: p.MsgSeq,
		Source:    SourceTencent, SourceMessageID: messageID, ConversationID: conversationID,
		ConversationType: conversationType, FromAccount: p.FromAccount, ToAccount: p.ToAccount,
		Content: content, Parts: parts, OccurredAt: when, BridgeMarker: marker,
	}, nil
}

// faceContent accepts both the standard TIMFaceElem Data string and the
// TUIKit convention where the data is already an encoded [TUIEmoji_*] token.
// A deterministic fallback keeps the event relayable when a new face index is
// introduced by the client.
func faceContent(index int, data string) string {
	if strings.TrimSpace(data) != "" {
		return data
	}
	if index > 0 {
		return fmt.Sprintf("[表情#%d]", index)
	}
	return "[表情]"
}

func customContent(data, desc, ext string) string {
	data = strings.TrimSpace(data)
	if data == "" {
		data = strings.TrimSpace(desc)
	}
	if data == "" {
		data = strings.TrimSpace(ext)
	}
	if data == "" {
		return "[自定义消息]"
	}
	// Some TUIKit custom elements wrap the visible text in JSON.  Extract it
	// without rejecting the event when the custom payload belongs to another
	// plugin.
	var value map[string]any
	if json.Unmarshal([]byte(data), &value) == nil {
		for _, key := range []string{"text", "content", "desc", "description", "emoji"} {
			if v, ok := value[key].(string); ok && strings.TrimSpace(v) != "" {
				return v
			}
		}
		if desc != "" {
			return desc
		}
		return "[暂不支持的自定义消息]"
	}
	return data
}

// normalizeEmojiText converts the TUIKit text-token emoji format to ordinary
// Unicode so TangSeng and Tencent clients that do not share TUIKit's image
// bundle still display the message. Unknown tokens remain readable with a
// Chinese fallback instead of being discarded.
func normalizeEmojiText(content string) string {
	for token, replacement := range tuiEmojiMap {
		content = strings.ReplaceAll(content, token, replacement)
	}
	// Keep unknown TUIEmoji tokens visible rather than leaking a renderer-only
	// marker that looks like a missing message on the receiving client.
	for {
		start := strings.Index(content, "[TUIEmoji_")
		if start < 0 {
			break
		}
		end := strings.Index(content[start:], "]")
		if end < 0 {
			break
		}
		end += start
		name := strings.TrimPrefix(strings.TrimSuffix(content[start:end+1], "]"), "[TUIEmoji_")
		content = content[:start] + "[表情:" + name + "]" + content[end+1:]
	}
	return strings.TrimSpace(content)
}

// NormalizeEmojiText is shared by both relay directions.  It is exported so
// the REST/WebSocket adapters can normalize a token that originated outside
// the Tencent callback path as well.
func NormalizeEmojiText(content string) string {
	return normalizeEmojiText(content)
}

var tuiEmojiMap = map[string]string{
	"[TUIEmoji_Smile]": "😀", "[TUIEmoji_Expect]": "🤔", "[TUIEmoji_Blink]": "😉", "[TUIEmoji_Guffaw]": "😂",
	"[TUIEmoji_KindSmile]": "😊", "[TUIEmoji_Haha]": "😄", "[TUIEmoji_Cheerful]": "😃", "[TUIEmoji_Speechless]": "😶",
	"[TUIEmoji_Amazed]": "😮", "[TUIEmoji_Sorrow]": "😢", "[TUIEmoji_Complacent]": "😌", "[TUIEmoji_Silly]": "😜",
	"[TUIEmoji_Lustful]": "😍", "[TUIEmoji_Giggle]": "🤭", "[TUIEmoji_Kiss]": "😘", "[TUIEmoji_Wail]": "😭",
	"[TUIEmoji_TearsLaugh]": "🤣", "[TUIEmoji_Trapped]": "😱", "[TUIEmoji_Mask]": "😷", "[TUIEmoji_Fear]": "😨",
	"[TUIEmoji_BareTeeth]": "😁", "[TUIEmoji_FlareUp]": "😤", "[TUIEmoji_Yawn]": "🥱", "[TUIEmoji_Tact]": "🤐",
	"[TUIEmoji_Stareyes]": "🤩", "[TUIEmoji_ShutUp]": "🤫", "[TUIEmoji_Sigh]": "😮‍💨", "[TUIEmoji_Hehe]": "😏",
	"[TUIEmoji_Silent]": "😑", "[TUIEmoji_Surprised]": "😲", "[TUIEmoji_Askance]": "🙄", "[TUIEmoji_Ok]": "👌",
	"[TUIEmoji_Shit]": "💩", "[TUIEmoji_Monster]": "👹", "[TUIEmoji_Daemon]": "👿", "[TUIEmoji_Rage]": "😡",
	"[TUIEmoji_Fool]": "🤡", "[TUIEmoji_Pig]": "🐷", "[TUIEmoji_Cow]": "🐮", "[TUIEmoji_Ai]": "🤖",
	"[TUIEmoji_Skull]": "💀", "[TUIEmoji_Bombs]": "💣", "[TUIEmoji_Coffee]": "☕", "[TUIEmoji_Cake]": "🍰",
	"[TUIEmoji_Beer]": "🍻", "[TUIEmoji_Flower]": "🌹", "[TUIEmoji_Watermelon]": "🍉", "[TUIEmoji_Rich]": "🤑",
	"[TUIEmoji_Heart]": "❤️", "[TUIEmoji_Moon]": "🌙", "[TUIEmoji_Sun]": "☀️", "[TUIEmoji_Star]": "⭐",
	"[TUIEmoji_RedPacket]": "🧧", "[TUIEmoji_Celebrate]": "🎉", "[TUIEmoji_Bless]": "🙏", "[TUIEmoji_Fortune]": "🧧",
	"[TUIEmoji_Convinced]": "💪", "[TUIEmoji_Prohibit]": "🚫", "[TUIEmoji_666]": "🤘", "[TUIEmoji_857]": "👏",
	"[TUIEmoji_Knife]": "🔪", "[TUIEmoji_Like]": "👍",
}

func ParseTangMessage(payload json.RawMessage) (MessageEvent, error) {
	var p tangPayload
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.UseNumber()
	if err := decoder.Decode(&p); err != nil {
		return MessageEvent{}, fmt.Errorf("decode TangSeng payload: %w", err)
	}
	messageID := fmt.Sprint(p.MessageID)
	if messageID == "<nil>" {
		messageID = ""
	}
	if messageID == "" {
		messageID = p.MsgID
	}
	if messageID == "" || p.FromUID == "" || p.ChannelID == "" {
		return MessageEvent{}, fmt.Errorf("TangSeng message_id, from_uid and channel_id are required")
	}
	content := p.Content
	if content == "" {
		content = p.Text
	}
	if strings.HasPrefix(p.ClientMsgNo, "hwb_") {
		p.BridgeSource = "bridge"
		p.BridgeEventID = p.ClientMsgNo
	}
	var payloadMap map[string]any
	payloadRaw := p.Payload
	var encodedPayload string
	if len(payloadRaw) > 0 && json.Unmarshal(payloadRaw, &encodedPayload) == nil && encodedPayload != "" {
		if decoded, decodeErr := base64.StdEncoding.DecodeString(encodedPayload); decodeErr == nil {
			payloadRaw = decoded
		}
	}
	if len(payloadRaw) > 0 && json.Unmarshal(payloadRaw, &payloadMap) == nil {
		if kind, ok := payloadMap["type"].(float64); ok && (kind == 97 || kind == 98 || kind == 99 || kind == 20) {
			return MessageEvent{}, ErrIgnoredMessage
		}
		if content == "" {
			if value, ok := payloadMap["content"].(string); ok {
				content = value
			}
		}
		if p.BridgeSource == "" {
			if value, ok := payloadMap["bridge_source"].(string); ok {
				p.BridgeSource = value
			}
		}
		if p.BridgeEventID == "" {
			if value, ok := payloadMap["bridge_event_id"].(string); ok {
				p.BridgeEventID = value
			}
		}
	}
	parts, summary := TangParts(payloadRaw)
	if len(parts) > 0 {
		content = summary
	}
	if content == "" {
		return MessageEvent{}, fmt.Errorf("TangSeng message content is required")
	}
	conversationType := ConversationC2C
	if p.ChannelType == 2 || p.ToUID == "" && p.ChannelType != 1 {
		conversationType = ConversationGroup
	}
	when := p.OccurredAt
	if when == 0 {
		when = time.Now().Unix()
	}
	mention, reply := TangMetadata(payloadRaw)
	return MessageEvent{
		Mention: mention, Reply: reply,
		NeedReceipt: p.NeedReceipt, Source: SourceTang, SourceMessageID: messageID, ConversationID: p.ChannelID,
		SourceSeq: p.MessageSeq, ClientMsgNo: p.ClientMsgNo,
		ConversationType: conversationType, FromAccount: p.FromUID, ToAccount: p.ToUID,
		Content: content, Parts: parts, OccurredAt: when,
		BridgeMarker: strings.TrimSpace(p.BridgeSource + ":" + p.BridgeEventID),
	}, nil
}
