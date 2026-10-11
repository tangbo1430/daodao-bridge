package httpapi

import (
	"encoding/json"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/business"
	"net/http"
	"strings"
)

func (s *Server) groupAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "POST is required")
		return
	}
	if s.Business == nil {
		writeError(w, 503, "group service unavailable")
		return
	}
	var req business.GroupRequest
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req) != nil || strings.TrimSpace(req.Owner) == "" || len(req.Members) > 1000 {
		writeError(w, 400, "invalid group request")
		return
	}
	g, err := s.Business.Group(r.Context(), strings.TrimPrefix(r.URL.Path, "/api/bridge/group_"), req)
	if err != nil {
		s.businessError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"code": 200, "msg": "操作成功", "data": map[string]any{"group_id": g.TencentID, "group_no": g.Number, "name": g.Name, "status": g.Status, "mirror_ready": g.MirrorReady, "invite": g.Invite, "forbidden": g.Forbidden}})
}
