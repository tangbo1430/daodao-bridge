package domain

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strings"
)

type MessagePart struct {
	UserID    string `json:"user_id,omitempty"`
	Avatar    string `json:"avatar,omitempty"`
	Vercode   string `json:"vercode,omitempty"`
	Kind      string `json:"kind"`
	Text      string `json:"text,omitempty"`
	URL       string `json:"url,omitempty"`
	Cover     string `json:"cover,omitempty"`
	Name      string `json:"name,omitempty"`
	Size      int64  `json:"size,omitempty"`
	CoverSize int64  `json:"cover_size,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	Seconds   int    `json:"seconds,omitempty"`
	Format    int    `json:"format,omitempty"`
}
type MediaObject struct {
	SHA256    string `json:"sha256"`
	SourceURL string `json:"source_url"`
	ObjectURL string `json:"object_url"`
	MIME      string `json:"mime"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
}

func TencentParts(raw json.RawMessage) []MessagePart {
	var body struct {
		Elements []struct {
			Type    string          `json:"MsgType"`
			Content json.RawMessage `json:"MsgContent"`
		} `json:"MsgBody"`
	}
	if json.Unmarshal(raw, &body) != nil {
		return nil
	}
	parts := []MessagePart{}
	hasMedia := false
	for _, e := range body.Elements {
		var v struct {
			Text, URL, FileName, VideoURL, ThumbURL                   string
			Size, FileSize, VideoSize, ThumbSize                      int64
			Second, VideoSecond, ThumbWidth, ThumbHeight, ImageFormat int
			ImageInfoArray                                            []struct {
				Type, Width, Height int
				Size                int64
				URL                 string
			}
			Index               int
			Data, Desc, Ext     string
			Latitude, Longitude float64
		}
		if json.Unmarshal(e.Content, &v) != nil {
			continue
		}
		part := MessagePart{Kind: "text"}
		switch e.Type {
		case "TIMTextElem":
			part.Text = NormalizeEmojiText(v.Text)
		case "TIMFaceElem":
			part.Text = NormalizeEmojiText(faceContent(v.Index, v.Data))
		case "TIMImageElem":
			part.Kind = "image"
			part.Format = v.ImageFormat
			for _, img := range v.ImageInfoArray {
				if part.URL == "" || img.Type == 1 {
					part.URL = img.URL
					part.Size = img.Size
					part.Width = img.Width
					part.Height = img.Height
				}
				if img.Type == 1 {
					break
				}
			}
			hasMedia = true
		case "TIMFileElem":
			part.Kind = "file"
			part.URL = v.URL
			part.Size = v.FileSize
			part.Name = v.FileName
			hasMedia = true
		case "TIMSoundElem":
			part.Kind = "audio"
			part.URL = v.URL
			part.Size = v.Size
			part.Seconds = v.Second
			hasMedia = true
		case "TIMVideoFileElem":
			part.Kind = "video"
			part.URL = v.VideoURL
			part.Cover = v.ThumbURL
			part.Size = v.VideoSize
			part.CoverSize = v.ThumbSize
			part.Width = v.ThumbWidth
			part.Height = v.ThumbHeight
			part.Seconds = v.VideoSecond
			hasMedia = true
		case "TIMCustomElem":
			var card struct{ BusinessID, UserID, Name, Avatar string }
			if json.Unmarshal([]byte(v.Data), &card) == nil && card.BusinessID == "send_person_card" && card.UserID != "" {
				part = MessagePart{Kind: "card", UserID: card.UserID, Name: card.Name, Avatar: card.Avatar}
				hasMedia = true
			} else {
				part.Text = customContent(v.Data, v.Desc, v.Ext)
			}
		case "TIMLocationElem":
			part.Text = fmt.Sprintf("[位置] %s (%g, %g)", v.Desc, v.Latitude, v.Longitude)
			hasMedia = true
		default:
			part.Kind = "tip"
			part.Text = "[暂不支持的消息类型:" + e.Type + "]"
			hasMedia = true
		}
		if part.Kind != "text" && part.Kind != "tip" && part.Kind != "card" && part.URL == "" {
			part = MessagePart{Kind: "tip", Text: "[媒体地址缺失:" + e.Type + "]"}
		}
		parts = append(parts, part)
	}
	if !hasMedia {
		return nil
	}
	return parts
}
func TangParts(raw json.RawMessage) ([]MessagePart, string) {
	var v struct {
		Type     int    `json:"type"`
		Flame    int    `json:"flame"`
		URL      string `json:"url"`
		Cover    string `json:"cover"`
		Content  string `json:"content"`
		Name     string `json:"name"`
		Width    int    `json:"width"`
		Height   int    `json:"height"`
		Size     int64  `json:"size"`
		TimeTrad int    `json:"timeTrad"`
		Second   int    `json:"second"`
		UID      string `json:"uid"`
		Avatar   string `json:"avatar"`
		Vercode  string `json:"vercode"`
	}
	if json.Unmarshal(raw, &v) != nil {
		return nil, ""
	}
	p := MessagePart{URL: v.URL, Cover: v.Cover, Name: v.Name, Size: v.Size, Width: v.Width, Height: v.Height, Seconds: v.Second}
	if v.Flame == 1 {
		p = MessagePart{Kind: "tip", Text: "对方发送了一条阅后即焚消息，当前版本暂不支持查看"}
	} else {
		switch v.Type {
		case 0, 1:
			return nil, ""
		case 2:
			p.Kind = "image"
			p.Format = 1
		case 3:
			p.Kind = "image"
			p.Format = 2
		case 4:
			p.Kind = "audio"
			p.Seconds = v.TimeTrad
		case 5:
			p.Kind = "video"
		case 8:
			p.Kind = "file"
		case 7:
			p = MessagePart{Kind: "card", UserID: v.UID, Name: v.Name, Avatar: v.Avatar, Vercode: v.Vercode}
		case 2000:
			p.Kind = "tip"
			p.Text = v.Content
		case 6:
			p.Kind = "text"
			p.Text = "[位置] " + v.Content
		case 11:
			p.Kind = "text"
			p.Text = mergedSummary(raw)
		case 12, 13:
			p.Kind = "text"
			p.Text = "[表情]"
		case 14:
			p.Kind = "text"
			p.Text = html.UnescapeString(htmlTags.ReplaceAllString(v.Content, ""))
			if strings.TrimSpace(p.Text) == "" {
				p.Text = "[富文本]"
			}
		default:
			p.Kind = "tip"
			p.Text = "[暂不支持此消息]"
		}
	}
	label := map[string]string{"image": "[图片]", "audio": "[语音]", "video": "[视频]", "file": "[文件]", "card": "[名片]"}[p.Kind]
	if p.Kind == "tip" || p.Kind == "text" {
		label = p.Text
	}
	return []MessagePart{p}, label
}

var htmlTags = regexp.MustCompile(`<[^>]*>`)

func mergedSummary(raw []byte) string {
	var v struct {
		Msgs []struct {
			From    string `json:"from_uid"`
			Payload struct {
				Type    int    `json:"type"`
				Content string `json:"content"`
				Flame   int    `json:"flame"`
			} `json:"payload"`
		} `json:"msgs"`
		Users []struct {
			UID  string `json:"uid"`
			Name string `json:"name"`
		} `json:"users"`
	}
	if json.Unmarshal(raw, &v) != nil {
		return "[聊天记录]"
	}
	names := map[string]string{}
	for _, u := range v.Users {
		names[u.UID] = u.Name
	}
	out := "[聊天记录]"
	for i, m := range v.Msgs {
		if i == 10 {
			break
		}
		name := names[m.From]
		if name == "" {
			name = "用户"
		}
		summary := m.Payload.Content
		if m.Payload.Flame == 1 {
			summary = "[阅后即焚消息不支持转发]"
		} else if summary == "" {
			summary = map[int]string{2: "[图片]", 3: "[GIF]", 4: "[语音]", 5: "[视频]", 7: "[名片]", 8: "[文件]"}[m.Payload.Type]
			if summary == "" {
				summary = "[消息]"
			}
		}
		r := []rune(summary)
		if len(r) > 100 {
			r = r[:100]
		}
		out += "\n" + name + "：" + string(r)
	}
	return out
}
func (p MessagePart) TangPayload() map[string]any {
	switch p.Kind {
	case "card":
		return map[string]any{"type": 7, "uid": p.UserID, "name": p.Name, "avatar": p.Avatar, "vercode": p.Vercode}
	case "image":
		typ := 2
		if p.Format == 2 {
			typ = 3
		}
		return map[string]any{"type": typ, "url": p.URL, "width": p.Width, "height": p.Height}
	case "file":
		return map[string]any{"type": 8, "url": p.URL, "name": p.Name, "size": p.Size}
	case "audio":
		return map[string]any{"type": 4, "url": p.URL, "timeTrad": p.Seconds}
	case "video":
		return map[string]any{"type": 5, "url": p.URL, "cover": p.Cover, "size": p.Size, "width": p.Width, "height": p.Height, "second": p.Seconds}
	case "tip":
		return map[string]any{"type": 2000, "content": p.Text}
	default:
		return map[string]any{"type": 1, "content": NormalizeEmojiText(p.Text)}
	}
}
func TencentElements(parts []MessagePart) []any {
	out := []any{}
	for _, p := range parts {
		typ := "TIMTextElem"
		content := map[string]any{"Text": p.Text}
		id := StableID(p.URL)[:32]
		switch p.Kind {
		case "card":
			typ = "TIMCustomElem"
			data, _ := json.Marshal(map[string]any{"businessID": "send_person_card", "userID": p.UserID, "name": p.Name, "avatar": p.Avatar})
			content = map[string]any{"Data": string(data), "Desc": "[名片]", "Ext": "send_person_card"}
		case "image":
			typ = "TIMImageElem"
			images := []any{}
			for i := 1; i <= 3; i++ {
				images = append(images, map[string]any{"Type": i, "URL": p.URL, "Width": p.Width, "Height": p.Height, "Size": p.Size})
			}
			format := p.Format
			if format == 0 {
				format = 1
			}
			content = map[string]any{"UUID": id, "ImageFormat": format, "ImageInfoArray": images}
		case "file":
			typ = "TIMFileElem"
			content = map[string]any{"Url": p.URL, "UUID": id, "FileName": p.Name, "FileSize": p.Size, "Download_Flag": 2}
		case "audio":
			typ = "TIMSoundElem"
			content = map[string]any{"Url": p.URL, "UUID": id, "Size": p.Size, "Second": p.Seconds, "Download_Flag": 2}
		case "video":
			typ = "TIMVideoFileElem"
			content = map[string]any{"VideoUrl": p.URL, "VideoUUID": id, "VideoSize": p.Size, "VideoSecond": p.Seconds, "VideoFormat": "mp4", "VideoDownloadFlag": 2, "ThumbUrl": p.Cover, "ThumbUUID": StableID(p.Cover)[:32], "ThumbSize": p.CoverSize, "ThumbWidth": p.Width, "ThumbHeight": p.Height, "ThumbFormat": "JPG", "ThumbDownloadFlag": 2}
		case "tip":
			typ = "TIMCustomElem"
			data, _ := json.Marshal(map[string]any{"businessID": "bridge_tip", "version": 1, "text": p.Text, "content": p.Text})
			content = map[string]any{"Data": string(data), "Desc": p.Text, "Ext": "bridge_tip"}
		}
		out = append(out, map[string]any{"MsgType": typ, "MsgContent": content})
	}
	return out
}
func PartsSummary(parts []MessagePart) string {
	var s strings.Builder
	for _, p := range parts {
		if p.Kind == "text" || p.Kind == "tip" {
			s.WriteString(p.Text)
		} else {
			s.WriteString(map[string]string{"image": "[图片]", "audio": "[语音]", "video": "[视频]", "file": "[文件]", "card": "[名片]"}[p.Kind])
		}
	}
	return s.String()
}
