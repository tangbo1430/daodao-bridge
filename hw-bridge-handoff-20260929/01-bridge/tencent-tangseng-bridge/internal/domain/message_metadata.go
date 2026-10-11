package domain

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Mention struct {
	All  int      `json:"all"`
	UIDs []string `json:"uids,omitempty"`
}
type ReplyReference struct {
	ID      string `json:"id"`
	Seq     int64  `json:"seq"`
	Time    int64  `json:"time"`
	From    string `json:"from"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Type    int    `json:"type"`
}

func TencentMetadata(raw []byte) (*Mention, *ReplyReference, string) {
	var p struct {
		SDKID  string `json:"MsgId"`
		Custom string `json:"CloudCustomData"`
		At     []struct {
			All int    `json:"GroupAtAllFlag"`
			UID string `json:"GroupAt_Account"`
		} `json:"GroupAtInfo"`
	}
	if json.Unmarshal(raw, &p) != nil {
		return nil, nil, ""
	}
	var mention *Mention
	if len(p.At) > 0 {
		mention = &Mention{}
		for _, a := range p.At {
			if a.All == 1 {
				mention.All = 1
			}
			if a.UID != "" {
				mention.UIDs = append(mention.UIDs, a.UID)
			}
		}
	}
	var custom struct {
		Reply *struct {
			ID      string `json:"messageID"`
			Seq     int64  `json:"messageSequence"`
			Time    int64  `json:"messageTime"`
			From    string `json:"messageSender"`
			Summary string `json:"messageAbstract"`
			Type    int    `json:"messageType"`
		} `json:"messageReply"`
	}
	var reply *ReplyReference
	if json.Unmarshal([]byte(p.Custom), &custom) == nil && custom.Reply != nil {
		r := custom.Reply
		reply = &ReplyReference{ID: r.ID, Seq: r.Seq, Time: r.Time, From: r.From, Name: r.From, Summary: r.Summary, Type: r.Type}
	}
	return mention, reply, p.SDKID
}
func TangMetadata(raw []byte) (*Mention, *ReplyReference) {
	var p struct {
		Flame   int      `json:"flame"`
		Mention *Mention `json:"mention"`
		Reply   *struct {
			ID      json.Number     `json:"message_id"`
			Seq     int64           `json:"message_seq"`
			From    string          `json:"from_uid"`
			Name    string          `json:"from_name"`
			Payload json.RawMessage `json:"payload"`
		} `json:"reply"`
	}
	if json.Unmarshal(raw, &p) != nil || p.Flame == 1 {
		return nil, nil
	}
	var reply *ReplyReference
	if p.Reply != nil {
		r := p.Reply
		_, summary := TangParts(r.Payload)
		if summary == "" {
			var v struct {
				Content string `json:"content"`
			}
			_ = json.Unmarshal(r.Payload, &v)
			summary = v.Content
		}
		reply = &ReplyReference{ID: r.ID.String(), Seq: r.Seq, From: r.From, Name: r.Name, Summary: summary, Type: 1}
	}
	return p.Mention, reply
}
func (e MessageEvent) TangPayload() map[string]any {
	p := map[string]any{"type": 1, "content": NormalizeEmojiText(e.Content)}
	if len(e.Parts) > 0 {
		p = e.Parts[0].TangPayload()
	}
	if e.BridgeMarker != "" {
		p["bridge_event_id"] = e.BridgeMarker
		p["bridge_source"] = string(e.Source)
	}
	if e.Mention != nil {
		p["mention"] = e.Mention
	}
	if e.Reply != nil {
		r := e.Reply
		p["reply"] = map[string]any{"message_id": r.ID, "message_seq": r.Seq, "from_uid": r.From, "from_name": r.Name, "payload": map[string]any{"type": 1, "content": r.Summary}}
	}
	return p
}
func (e MessageEvent) TencentReply() map[string]any {
	if e.Reply == nil {
		return nil
	}
	r := e.Reply
	return map[string]any{"messageID": r.ID, "messageAbstract": r.Summary, "messageSender": r.From, "messageType": r.Type, "messageTime": r.Time, "messageSequence": r.Seq, "version": 1}
}
func (e *MessageEvent) PrependText(prefix string) {
	e.Content = prefix + e.Content
	if len(e.Parts) > 0 {
		e.Parts = append([]MessagePart{{Kind: "text", Text: prefix}}, e.Parts...)
	}
}
func (e *MessageEvent) FallbackReply() {
	if e.Reply == nil {
		return
	}
	r := e.Reply
	name := strings.TrimSpace(r.Name)
	if name == "" {
		name = r.From
	}
	if name == "" {
		name = "对方"
	}
	summary := []rune(r.Summary)
	if len(summary) > 100 {
		summary = summary[:100]
	}
	e.PrependText(fmt.Sprintf("「回复 %s：%s」\n", name, string(summary)))
	e.Reply = nil
}
