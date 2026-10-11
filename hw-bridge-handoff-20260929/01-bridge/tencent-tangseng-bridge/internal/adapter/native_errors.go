package adapter

import (
	"errors"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"strings"
)

func nativeFailure(err error) error {
	var status *HTTPStatusError
	if !errors.As(err, &status) {
		return err
	}
	if strings.Contains(status.Message, "该用户已被禁用") {
		return &domain.DeliveryError{Reason: "account_banned", Code: status.Status, Public: "聊天服务已停用，请联系管理员", Permanent: true}
	}
	if strings.Contains(status.Message, "租户已停用") || strings.Contains(status.Message, "租户已被禁用") {
		return &domain.DeliveryError{Reason: "tenant_disabled", Code: status.Status, Public: "聊天服务已停用，请联系管理员", Permanent: true}
	}
	if strings.Contains(status.Message, "已经是好友") {
		return &domain.DeliveryError{Reason: "already_friends", Code: status.Status, Public: "双方已经是好友", Permanent: true}
	}
	if strings.Contains(status.Message, "获取好友申请token的信息失败") {
		return &domain.DeliveryError{Reason: "friend_request_expired", Code: status.Status, Public: "好友申请已过期，请重新申请", Permanent: true}
	}
	// Do not expose raw native msg/response. Only definite HTTP refusal is
	// terminal. In particular 408/429/5xx cannot be classified as business denial.
	switch status.Status {
	case 401:
		return &domain.DeliveryError{Reason: "authentication", Code: 401, Public: "登录状态已失效，请重新登录", Auth: true}
	case 400, 403, 409, 422:
		return &domain.DeliveryError{Reason: "native_rejected", Code: status.Status, Public: "操作被拒绝，请确认好友关系、账号状态或群操作权限", Permanent: true}
	}
	return err
}
