package httpapi

import (
	"encoding/json"
	"net/http"
)

func (s *Server) messageState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "POST is required")
		return
	}
	if s.Business == nil {
		writeError(w, 503, "message service unavailable")
		return
	}
	var req struct {
		Owner   string `json:"tencent_user_id"`
		Key     string `json:"message_key"`
		Channel string `json:"channel_id"`
		Type    int    `json:"channel_type"`
		Before  int64  `json:"last_read_time"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.Owner == "" {
		writeError(w, 400, "tencent_user_id is required")
		return
	}
	var err error
	count := 0
	switch r.URL.Path {
	case "/api/bridge/message_recall":
		err = s.Business.RecallBoth(r.Context(), req.Owner, req.Key)
	case "/api/bridge/message_recall_notify":
		err = s.Business.Recall(r.Context(), req.Owner, req.Key)
	case "/api/bridge/message_read":
		err = s.Business.Read(r.Context(), req.Owner, req.Channel, req.Before)
	default:
		count, err = s.Business.SyncMessageExtras(r.Context(), req.Owner, req.Channel, req.Type)
	}
	if err != nil {
		s.businessError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"code": 200, "processed": count})
}
