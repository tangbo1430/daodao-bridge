package httpapi

import (
	"encoding/json"
	"errors"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/account"
	"net/http"
)

func (s *Server) registration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "POST is required")
		return
	}
	if s.Accounts == nil {
		writeError(w, 503, "注册服务未配置")
		return
	}
	var err error
	var result any = map[string]any{}
	if r.URL.Path == "/api/v1/account/prepare" {
		var q account.PrepareRequest
		if json.NewDecoder(r.Body).Decode(&q) != nil {
			writeError(w, 400, "注册参数无效")
			return
		}
		var v account.Reservation
		v, err = s.Accounts.Prepare(r.Context(), q)
		result = map[string]any{"reserve_id": v.ID, "tsdd_uid": v.UID, "app_id": v.AppID, "status": v.Status}
	} else {
		var q struct {
			ID  string `json:"reserve_id"`
			TIM string `json:"tim_uid"`
		}
		if json.NewDecoder(r.Body).Decode(&q) != nil {
			writeError(w, 400, "注册参数无效")
			return
		}
		if r.URL.Path == "/api/v1/account/commit" {
			err = s.Accounts.CommitRegistration(r.Context(), q.ID, q.TIM)
		} else {
			err = s.Accounts.AbortRegistration(r.Context(), q.ID)
		}
	}
	if err != nil {
		code, msg := 1005, "注册处理未完成，请稍后重试"
		var known *account.RegistrationError
		if errors.As(err, &known) {
			code, msg = known.Code, known.Message
		}
		writeJSON(w, 409, map[string]any{"code": code, "msg": msg})
		return
	}
	writeJSON(w, 200, map[string]any{"code": 0, "data": result})
}
