package httpapi

import (
	"encoding/json"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"net/http"
)

func (s *Server) profileSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "POST required")
		return
	}
	if s.Business == nil {
		writeError(w, 503, "profile service unavailable")
		return
	}
	var req struct {
		Owner  string `json:"tencent_user_id"`
		Name   string `json:"display_name"`
		Avatar string `json:"avatar_url"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req) != nil || req.Owner == "" {
		writeError(w, 400, "invalid profile")
		return
	}
	if s.Engine == nil || s.Accounts == nil {
		writeError(w, 503, "profile queue unavailable")
		return
	}
	owner, err := s.Accounts.Profile(r.Context(), req.Owner)
	if err != nil || owner.Status != "confirmed" {
		writeJSON(w, 403, map[string]any{"code": 403, "msg": "聊天服务尚未就绪，请稍后重试"})
		return
	}
	body, _ := json.Marshal(req)
	id := domain.NewEventID()
	_, err = s.Engine.Accept(r.Context(), domain.EventEnvelope{Source: domain.SourceTencent, Command: "Profile.Sync", EventID: id, IdempotencyKey: id, Payload: body})
	if err != nil {
		s.businessError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"code": 200, "status": "queued", "event_id": id})
}
