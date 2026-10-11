package domain

import "fmt"

// DeliveryError carries only curated public text. Native response bodies and
// signed URLs must never become chat content or persisted error messages.
type DeliveryError struct {
	Reason    string
	Code      int
	Public    string
	Permanent bool
	Auth      bool
	Media     bool
}

func (e *DeliveryError) Error() string {
	return fmt.Sprintf("delivery %s code=%d: %s", e.Reason, e.Code, e.Public)
}

// WuKongFailure follows the handoff specification section 7.3. Unknown codes
// and system/rate errors remain retryable; a timeout is not a negative ACK.
func WuKongFailure(code int) *DeliveryError {
	e := &DeliveryError{Reason: "temporary", Code: code, Public: "服务暂时不可用，请稍后重试"}
	switch code {
	case 2:
		e.Reason, e.Public, e.Auth = "authentication", "登录状态已失效，请重新登录", true
	case 3:
		e.Reason, e.Public, e.Permanent = "not_group_member", "消息未送达：你已不在该群中", true
	case 4:
		e.Reason, e.Public, e.Permanent = "recipient_rejected", "消息已发出，但被对方拒收", true
	case 13:
		e.Reason, e.Public, e.Permanent = "permission_denied", "消息未送达：请确认好友关系、群成员身份或群发言权限", true
	case 19:
		e.Reason, e.Public, e.Permanent = "banned", "消息未送达：账号或频道已被停用", true
	case 24:
		e.Reason, e.Public, e.Permanent = "group_disbanded", "消息未送达：群已解散", true
	case 25:
		e.Reason, e.Public, e.Permanent = "send_forbidden", "暂时无法发送：已被禁言或服务已停用", true
	}
	return e
}

type FailureNotice struct {
	OriginalJob string `json:"original_job"`
	Reason      string `json:"reason"`
	Sender      string `json:"sender"`
	Recipient   string `json:"recipient"`
}
