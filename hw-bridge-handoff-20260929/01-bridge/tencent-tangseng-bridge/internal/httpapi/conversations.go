package httpapi

import (
	"encoding/json"
	"net/http"
)

func (s *Server) sessionSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "POST is required")
		return
	}
	if s.Business == nil {
		writeError(w, 503, "session service unavailable")
		return
	}
	var req struct {
		Owner string `json:"tencent_user_id"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.Owner == "" {
		writeError(w, 400, "tencent_user_id is required")
		return
	}
	if r.URL.Path == "/api/bridge/tencent_roam_sync" {
		count, err := s.Business.SyncTencent(r.Context(), req.Owner)
		if err != nil {
			s.businessError(w, err)
			return
		}
		writeJSON(w, 200, map[string]any{"code": 200, "accepted": count})
		return
	}
	result, err := s.Business.SyncAccount(r.Context(), req.Owner)
	if err != nil {
		s.businessError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"code": 200, "data": result})
}
