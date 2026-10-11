package httpapi

import (
	"encoding/json"
	"errors"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/adapter"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/business"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"log"
	"net/http"
	"strings"
)

func (s *Server) friendAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "POST is required")
		return
	}
	if s.Business == nil {
		writeError(w, 503, "friend service unavailable")
		return
	}
	var req struct {
		Owner   string `json:"tencent_user_id"`
		Peer    string `json:"target_tsdd_uid"`
		Remark  string `json:"remark"`
		Vercode string `json:"vercode"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil || strings.TrimSpace(req.Owner) == "" || strings.TrimSpace(req.Peer) == "" {
		writeError(w, 400, "tencent_user_id and target_tsdd_uid are required")
		return
	}
	action := strings.TrimPrefix(r.URL.Path, "/api/bridge/friend_")
	result, err := s.Business.Friend(r.Context(), strings.TrimSpace(req.Owner), strings.TrimSpace(req.Peer), action, req.Remark, req.Vercode)
	if err != nil {
		s.businessError(w, err)
		return
	}
	// Never serialize encrypted application credentials into an APK response.
	msg := "操作成功"
	if result.Status == "pending_out" {
		msg = "已发送好友申请，等待对方验证"
	}
	writeJSON(w, 200, map[string]any{"code": 200, "msg": msg, "status": result.Status, "blocked": result.Blocked, "accepted": true})
}
func (s *Server) friendRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, 405, "GET is required")
		return
	}
	if s.Business == nil {
		writeError(w, 503, "friend service unavailable")
		return
	}
	rows, err := s.Business.Requests(r.Context(), r.URL.Query().Get("tencent_user_id"))
	if err != nil {
		s.businessError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"code": 200, "requests": rows})
}
func (s *Server) businessError(w http.ResponseWriter, err error) {
	status := 502
	var businessErr *business.Error
	if errors.As(err, &businessErr) {
		status = businessErr.Status
	}
	// Upstream errors may contain response details. Only our curated business
	// errors are safe to display through PHP to an Android user.
	message := "操作失败，请稍后重试"
	if businessErr != nil {
		message = businessErr.Message
	}
	body := map[string]any{"code": status, "msg": message}
	var delivery *domain.DeliveryError
	if errors.As(err, &delivery) {
		if delivery.Permanent {
			status = 409
		} else if delivery.Auth {
			status = 401
		}
		body["code"], body["msg"], body["reason"] = status, delivery.Public, delivery.Reason
	}
	var groupErr *business.GroupError
	if errors.As(err, &groupErr) {
		body["step"] = groupErr.Step
		log.Printf("bridge business failure step=%s class=%T", groupErr.Step, groupErr.Err)
	}
	var tencentErr *adapter.TencentError
	if errors.As(err, &tencentErr) {
		body["upstream_code"] = tencentErr.Code
		log.Printf("bridge Tencent failure operation=%s code=%d", tencentErr.Operation, tencentErr.Code)
	}
	writeJSON(w, status, body)
}
