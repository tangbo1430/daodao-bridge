package httpapi

import (
	"encoding/json"
	"net/http"
)

func (s *Server) accountRecheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "POST is required")
		return
	}
	if s.Accounts == nil {
		writeError(w, 503, "账号服务暂不可用")
		return
	}
	var q struct {
		TIMUID string `json:"tim_uid"`
	}
	if json.NewDecoder(r.Body).Decode(&q) != nil || q.TIMUID == "" {
		writeError(w, 400, "账号标识无效")
		return
	}
	result, err := s.Accounts.Recheck(r.Context(), q.TIMUID)
	if err != nil {
		writeJSON(w, 409, map[string]any{"code": 409, "msg": "原生账号状态尚未通过验证，请确认解封后重试"})
		return
	}
	writeJSON(w, 200, map[string]any{"code": 0, "data": result})
}
