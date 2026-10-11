package adapter

import (
	"encoding/json"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"strings"
)

func (c TencentClient) offlinePush(event domain.MessageEvent, from, to domain.Identity) map[string]any {
	if len(event.Parts) == 1 && event.Parts[0].Kind == "tip" {
		return map[string]any{"PushFlag": 1}
	}
	chatType, sender, title, avatar := 1, from.TargetID, from.DisplayName, from.AvatarURL
	if event.ConversationType == domain.ConversationGroup {
		chatType, sender, title, avatar = 2, to.TargetID, to.DisplayName, to.AvatarURL
	}
	if title == "" {
		title = sender
	}
	desc := event.Content
	for _, p := range event.Parts {
		switch p.Kind {
		case "image":
			desc = "[图片]"
		case "audio":
			desc = "[语音]"
		case "video":
			desc = "[视频]"
		case "file":
			desc = "[文件]"
		case "card":
			desc = "[名片]"
		default:
			continue
		}
		break
	}
	desc = pushText(domain.NormalizeEmojiText(desc), 100)
	title = pushText(title, 40)
	// The bundled OfflinePushExtBusinessInfo uses sender for BOTH user and
	// group IDs; group routing must not use the message author's user ID.
	ext, _ := json.Marshal(map[string]any{"entity": map[string]any{"version": 1, "action": 1, "chatType": chatType, "sender": sender, "nickname": title, "faceUrl": avatar, "content": desc, "sendTime": event.OccurredAt}})
	info := map[string]any{"PushFlag": 0, "Title": title, "Desc": desc, "Ext": string(ext)}
	if len(c.PushAndroidInfo) > 0 {
		info["AndroidInfo"] = c.PushAndroidInfo
	}
	return info
}

func pushText(s string, limit int) string {
	r := []rune(strings.Join(strings.Fields(s), " "))
	if len(r) > limit {
		return string(r[:limit]) + "…"
	}
	return string(r)
}
