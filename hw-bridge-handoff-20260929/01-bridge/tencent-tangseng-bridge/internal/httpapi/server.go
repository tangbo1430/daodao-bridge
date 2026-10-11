package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/account"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/adapter"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/business"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/engine"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
)

type Server struct {
	Media           http.Handler
	MediaSource     http.Handler
	Business        *business.Service
	Engine          *engine.Engine
	Store           store.Store
	SharedSecret    string
	TencentSDKAppID int
	Tencent         *adapter.TencentClient
	Tang            *adapter.TangSengClient
	BridgePeerID    string
	Accounts        *account.Service
	// TestSendText is an internal-only smoke-test hook wired to the PC(2)
	// session manager by main. It is protected by the normal HMAC middleware.
	TestSendText func(context.Context, string, string, string) (string, error)
	received     atomic.Int64
	accepted     atomic.Int64
	failed       atomic.Int64
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	if s.Media != nil {
		mux.Handle("/api/bridge/media", s.Media)
	}
	if s.MediaSource != nil {
		mux.Handle("/api/bridge/media-source", s.MediaSource)
	}
	mux.HandleFunc("/healthz", s.health)
	mux.HandleFunc("/readyz", s.ready)
	mux.HandleFunc("/metrics", s.metrics)
	mux.HandleFunc("/internal/v1/events/tencent", s.receiveTencent)
	mux.HandleFunc("/cb/tim", s.receiveTIM)
	mux.HandleFunc("/api/v1/account/reserve", s.accountReserve)
	mux.HandleFunc("/api/v1/account/confirm", s.accountConfirm)
	mux.HandleFunc("/api/v1/account/cancel", s.accountCancel)
	mux.HandleFunc("/api/v1/account/profile", s.accountProfile)
	mux.HandleFunc("/api/v1/account/recheck", s.accountRecheck)
	mux.HandleFunc("/api/v1/account/provision", s.bridgeProvision)
	for _, action := range []string{"prepare", "commit", "abort"} {
		mux.HandleFunc("/api/v1/account/"+action, s.registration)
	}
	mux.HandleFunc("/api/v1/im/bridge/config", s.bridgeConfig)
	mux.HandleFunc("/api/v1/im/bridge/provision", s.bridgeProvision)
	mux.HandleFunc("/api/v1/im/bridge/disable", s.bridgeDisable)
	mux.HandleFunc("/api/v1/admin/test/send-tangseng", s.testSendTangSeng)
	mux.HandleFunc("/api/bridge/search_user", s.searchUser)
	mux.HandleFunc("/api/bridge/message_extra_sync", s.messageState)
	mux.HandleFunc("/api/bridge/session_sync", s.sessionSync)
	mux.HandleFunc("/api/bridge/tencent_roam_sync", s.sessionSync)
	mux.HandleFunc("/api/bridge/message_read", s.messageState)
	mux.HandleFunc("/api/bridge/message_recall_notify", s.messageState)
	mux.HandleFunc("/api/bridge/message_recall", s.messageState)
	mux.HandleFunc("/api/bridge/profile_sync", s.profileSync)
	for _, action := range []string{"apply", "accept", "refuse", "delete", "block", "unblock"} {
		mux.HandleFunc("/api/bridge/friend_"+action, s.friendAction)
	}
	mux.HandleFunc("/api/bridge/friend_requests", s.friendRequests)
	for _, action := range []string{"create", "invite", "remove", "exit", "disband", "update", "sync"} {
		mux.HandleFunc("/api/bridge/group_"+action, s.groupAction)
	}
	mux.HandleFunc("/internal/v1/events/tangseng", s.receiveTang)
	mux.HandleFunc("/internal/v1/events/tangseng/webhook", s.receiveTangWebhook)
	mux.HandleFunc("/admin/replay/", s.replay)
	mux.HandleFunc("/admin/dead-letter/", s.retryDead)
	return requestLogger(s.auth(mux))
}

func (s *Server) searchUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, 405, "GET is required")
		return
	}
	if s.Tang == nil || s.Accounts == nil || s.Store == nil || s.Tencent == nil {
		writeError(w, 503, "bridge relation service is not configured")
		return
	}
	owner := strings.TrimSpace(r.URL.Query().Get("tencent_user_id"))
	keyword := strings.TrimSpace(r.URL.Query().Get("keyword"))
	if owner == "" || keyword == "" {
		writeError(w, 400, "tencent_user_id and keyword are required")
		return
	}
	if _, err := s.Accounts.Profile(r.Context(), owner); err != nil {
		writeError(w, 404, err.Error())
		return
	}
	users, err := s.Tang.SearchUser(r.Context(), keyword)
	if err != nil {
		writeError(w, 502, err.Error())
		return
	}
	result := make([]map[string]any, 0, len(users))
	for _, user := range users {
		uid := strings.TrimSpace(user.UID)
		if uid == "" || uid == "fileHelper" {
			continue
		}
		name := user.Name
		if name == "" {
			name = user.Nick
		}
		identity := domain.Identity{Source: domain.SourceTang, SourceID: uid, Target: domain.SourceTencent, TargetID: uid, EntityType: "user", DisplayName: name, AvatarURL: firstNonEmpty(user.FaceURL, user.Avatar), Shadow: true}
		if s.Business != nil && s.Business.Media != nil {
			publicAvatar, avatarErr := s.Business.Media.AvatarURL(r.Context(), "users", uid)
			if avatarErr != nil {
				s.businessError(w, avatarErr)
				return
			}
			identity.AvatarURL = publicAvatar
			user.FaceURL = publicAvatar
			user.Avatar = publicAvatar
		}
		if ensured, e := s.Tencent.EnsureUser(r.Context(), identity); e != nil {
			writeError(w, 502, e.Error())
			return
		} else {
			identity = ensured
		}
		if e := s.Store.UpsertIdentity(r.Context(), identity); e != nil {
			writeError(w, 500, e.Error())
			return
		}
		// Keep only the TangSeng -> Tencent contact projection here.  The
		// dispatcher can resolve the reverse edge with FindIdentityByTarget.
		// Persisting a second Tencent -> TangSeng self-mapping makes account
		// restore mistake the searched real user for a provisioned Huawei
		// shadow account and can suppress that user's replies as loop traffic.
		if s.Business == nil {
			_ = s.Store.UpsertFriendLink(r.Context(), domain.SourceTencent, owner, uid, "discovered")
		}
		if s.Business != nil {
			if e := s.Business.SaveSearch(r.Context(), owner, user); e != nil {
				s.businessError(w, e)
				return
			}
		}
		result = append(result, map[string]any{"user_id": uid, "nick": name, "face_url": identity.AvatarURL, "vercode": user.Vercode})
	}
	writeJSON(w, 200, map[string]any{"users": result})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (s *Server) testSendTangSeng(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "POST is required")
		return
	}
	if s.TestSendText == nil {
		writeError(w, 503, "test sender is not configured")
		return
	}
	var req struct {
		TencentUserID string `json:"tencent_user_id"`
		ToTangSengUID string `json:"to_tangseng_uid"`
		Text          string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.TencentUserID) == "" || strings.TrimSpace(req.ToTangSengUID) == "" || req.Text == "" {
		writeError(w, 400, "tencent_user_id, to_tangseng_uid and text are required")
		return
	}
	id, err := s.Accounts.Profile(r.Context(), strings.TrimSpace(req.TencentUserID))
	if err != nil {
		writeError(w, 404, err.Error())
		return
	}
	if id.Status != "confirmed" {
		writeError(w, 409, "account is not confirmed")
		return
	}
	messageID, err := s.TestSendText(r.Context(), id.Identity.TargetID, strings.TrimSpace(req.ToTangSengUID), req.Text)
	if err != nil {
		writeError(w, 502, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"sent": true, "message_id": messageID, "from_tangseng_uid": id.Identity.TargetID, "to_tangseng_uid": strings.TrimSpace(req.ToTangSengUID)})
}

func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/healthz") || strings.HasPrefix(r.URL.Path, "/readyz") || strings.HasPrefix(r.URL.Path, "/metrics") {
			next.ServeHTTP(w, r)
			return
		}
		if s.SharedSecret == "" {
			writeError(w, http.StatusServiceUnavailable, "bridge shared secret is not configured")
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
		if err != nil {
			writeError(w, 400, "invalid request body")
			return
		}
		r.Body = io.NopCloser(strings.NewReader(string(body)))
		if r.URL.Path == "/cb/tim" || strings.HasPrefix(r.URL.Path, "/api/v1/account/") {
			if !s.verifyTIM(r, body) {
				writeError(w, http.StatusUnauthorized, "invalid Tencent bridge signature")
				return
			}
			switch r.URL.Path {
			case "/api/v1/account/prepare", "/api/v1/account/commit", "/api/v1/account/abort", "/api/v1/account/recheck":
				if s.Store == nil {
					writeError(w, 503, "nonce store unavailable")
					return
				}
				nonce := strings.TrimSpace(r.Header.Get("X-HWB-Nonce"))
				if len(nonce) < 16 || len(nonce) > 128 {
					writeError(w, 401, "invalid nonce")
					return
				}
				key := domain.StableID("hwb-account-nonce", nonce)
				fresh, err := s.Store.AcceptInbox(r.Context(), domain.EventEnvelope{EventID: key, IdempotencyKey: key, TraceID: key, Source: domain.SourceTencent, Command: "internal.account.nonce", OccurredAt: time.Now().Unix(), Payload: json.RawMessage(`{}`)})
				if err != nil {
					writeError(w, 503, "nonce store unavailable")
					return
				}
				if !fresh {
					writeError(w, 409, "request nonce already used")
					return
				}
			}
			r.Body = io.NopCloser(strings.NewReader(string(body)))
			next.ServeHTTP(w, r)
			return
		}
		provided := strings.TrimSpace(r.Header.Get("X-Bridge-Signature"))
		if provided == "" {
			writeError(w, http.StatusUnauthorized, "X-Bridge-Signature is required")
			return
		}
		h := hmac.New(sha256.New, []byte(s.SharedSecret))
		h.Write(body)
		expected := hex.EncodeToString(h.Sum(nil))
		if !hmac.Equal([]byte(strings.TrimPrefix(provided, "sha256=")), []byte(expected)) {
			writeError(w, http.StatusUnauthorized, "invalid bridge signature")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) accountReserve(w http.ResponseWriter, r *http.Request) {
	if s.Accounts == nil {
		writeError(w, 503, "account service is not configured")
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, 405, "POST is required")
		return
	}
	var req account.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "invalid request")
		return
	}
	value, err := s.Accounts.Reserve(r.Context(), req)
	if err != nil {
		writeError(w, 502, err.Error())
		return
	}
	writeJSON(w, 200, value)
}
func (s *Server) accountConfirm(w http.ResponseWriter, r *http.Request) {
	if s.Accounts == nil {
		writeError(w, 503, "account service is not configured")
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST is required")
		return
	}
	id := r.URL.Query().Get("tencent_user_id")
	if id == "" {
		var req account.Request
		_ = json.NewDecoder(r.Body).Decode(&req)
		id = req.TencentUserID
	}
	value, err := s.Accounts.Confirm(r.Context(), id)
	if err != nil {
		writeError(w, 404, err.Error())
		return
	}
	writeJSON(w, 200, value)
}
func (s *Server) accountCancel(w http.ResponseWriter, r *http.Request) {
	if s.Accounts == nil {
		writeError(w, 503, "account service is not configured")
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST is required")
		return
	}
	id := r.URL.Query().Get("tencent_user_id")
	if id == "" {
		var req account.Request
		_ = json.NewDecoder(r.Body).Decode(&req)
		id = req.TencentUserID
	}
	if err := s.Accounts.Cancel(r.Context(), id); err != nil {
		writeError(w, 404, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "cancelled"})
}
func (s *Server) accountProfile(w http.ResponseWriter, r *http.Request) {
	if s.Accounts == nil {
		writeError(w, 503, "account service is not configured")
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST is required")
		return
	}
	id := r.URL.Query().Get("tencent_user_id")
	value, err := s.Accounts.Profile(r.Context(), id)
	if err != nil {
		writeError(w, 404, err.Error())
		return
	}
	writeJSON(w, 200, value)
}

// bridgeConfig is the only endpoint the APK needs. It is intentionally
// callable by the PHP backend with the internal HWB signature; the Tencent
// secret is never returned.
func (s *Server) bridgeConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, 405, "GET is required")
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("tencent_user_id"))
	if id == "" {
		writeError(w, 400, "tencent_user_id is required")
		return
	}
	if s.Accounts == nil || s.Tencent == nil {
		writeError(w, 503, "bridge account service is not configured")
		return
	}
	record, err := s.Accounts.Profile(r.Context(), id)
	if err != nil {
		writeJSON(w, 200, map[string]any{"bridge_enabled": false, "tencent_user_id": id, "sdk_app_id": s.TencentSDKAppID})
		return
	}
	sig, err := s.Tencent.UserSig(id)
	if err != nil {
		writeError(w, 503, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"bridge_enabled": record.Status == "confirmed", "bridge_peer_id": s.BridgePeerID, "target": "tangseng", "sdk_app_id": s.TencentSDKAppID, "tencent_user_id": id, "user_sig": sig, "expires_in": s.Tencent.SigTTL, "shadow_uid": record.Identity.TargetID})
}

func (s *Server) bridgeProvision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "POST is required")
		return
	}
	if s.Accounts == nil {
		writeError(w, 503, "bridge account service is not configured")
		return
	}
	var req account.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "invalid request")
		return
	}
	record, err := s.Accounts.Reserve(r.Context(), req)
	if err != nil {
		writeError(w, 502, err.Error())
		return
	}
	record, err = s.Accounts.Confirm(r.Context(), req.TencentUserID)
	if err != nil {
		writeError(w, 502, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"bridge_enabled": true, "record": record})
}

func (s *Server) bridgeDisable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "POST is required")
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("tencent_user_id"))
	if id == "" {
		writeError(w, 400, "tencent_user_id is required")
		return
	}
	if s.Accounts == nil {
		writeError(w, 503, "bridge account service is not configured")
		return
	}
	if err := s.Accounts.Cancel(r.Context(), id); err != nil {
		writeError(w, 404, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"bridge_enabled": false, "tencent_user_id": id})
}

func (s *Server) verifyTIM(r *http.Request, body []byte) bool {
	ts, nonce, provided := strings.TrimSpace(r.Header.Get("X-HWB-Timestamp")), strings.TrimSpace(r.Header.Get("X-HWB-Nonce")), strings.TrimSpace(r.Header.Get("X-HWB-Sign"))
	if ts == "" || nonce == "" || provided == "" {
		return false
	}
	value, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return false
	}
	if value > 1e12 {
		value /= 1000
	}
	if delta := time.Now().Unix() - value; delta > 300 || delta < -300 {
		return false
	}
	h := hmac.New(sha256.New, []byte(s.SharedSecret))
	_, _ = h.Write([]byte(ts))
	_, _ = h.Write([]byte("\n"))
	_, _ = h.Write([]byte(nonce))
	_, _ = h.Write([]byte("\n"))
	_, _ = h.Write(body)
	expected := hex.EncodeToString(h.Sum(nil))
	provided = strings.TrimPrefix(provided, "sha256=")
	return hmac.Equal([]byte(strings.ToLower(provided)), []byte(expected))
}

func (s *Server) receiveTencent(w http.ResponseWriter, r *http.Request) {
	s.receive(w, r, domain.SourceTencent)
}

// receiveTIM accepts the raw Tencent callback body. PHP stores that body
// before forwarding it, so a successful 200 means the event is durably in the
// Bridge inbox rather than merely parsed in memory.
func (s *Server) receiveTIM(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST is required")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	if err != nil {
		writeError(w, 400, "invalid request body")
		return
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		writeError(w, 400, "invalid Tencent callback JSON")
		return
	}
	command, _ := raw["CallbackCommand"].(string)
	if command == "" {
		command, _ = raw["callback_command"].(string)
	}
	if s.TencentSDKAppID > 0 {
		if appID := firstInt64(raw, "SdkAppid", "SDKAppID"); appID != 0 && int(appID) != s.TencentSDKAppID {
			writeError(w, http.StatusBadRequest, "SdkAppid does not match bridge configuration")
			return
		}
	}
	if command == "" {
		writeError(w, 400, "CallbackCommand is required")
		return
	}
	messageID := firstString(raw, "MsgKey", "MsgId", "MsgID", "MessageId")
	if command == "Group.CallbackAfterSendMsg" && firstInt64(raw, "MsgSeq") > 0 {
		messageID = fmt.Sprintf("%s:%d", firstString(raw, "GroupId"), firstInt64(raw, "MsgSeq"))
	}
	if messageID == "" {
		messageID = domain.StableID("tim", command, string(body))
	}
	eventID := domain.StableID("tim-event", command, messageID)
	occurred := firstInt64(raw, "MsgTime", "Timestamp", "EventTime")
	if occurred == 0 {
		occurred = time.Now().Unix()
	}
	envelope := domain.EventEnvelope{EventID: eventID, TraceID: r.Header.Get("X-Trace-ID"), Source: domain.SourceTencent, Command: command, OccurredAt: occurred, IdempotencyKey: domain.MessageKey(domain.SourceTencent, command, messageID), Payload: json.RawMessage(body)}
	accepted, err := s.Engine.Accept(r.Context(), envelope)
	if err != nil {
		s.failed.Add(1)
		writeError(w, 500, err.Error())
		return
	}
	s.received.Add(1)
	if accepted {
		s.accepted.Add(1)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ActionStatus": "OK", "ErrorCode": 0, "accepted": accepted, "event_id": eventID})
}

func firstString(raw map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := raw[key].(string); ok && strings.TrimSpace(value) != "" {
			return value
		}
		if value, ok := raw[key].(json.Number); ok {
			return value.String()
		}
	}
	return ""
}
func firstInt64(raw map[string]any, keys ...string) int64 {
	for _, key := range keys {
		switch value := raw[key].(type) {
		case float64:
			return int64(value)
		case json.Number:
			if parsed, err := value.Int64(); err == nil {
				return parsed
			}
		case string:
			if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
				return parsed
			}
		}
	}
	return 0
}
func (s *Server) receiveTang(w http.ResponseWriter, r *http.Request) {
	s.receive(w, r, domain.SourceTang)
}

// receiveTangWebhook accepts the native TangSeng msg.notify payload (a JSON
// array of MsgResp objects) and converts each item into the normal inbox
// envelope. It lets the TangSeng server point its webhook directly at Bridge.
func (s *Server) receiveTangWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST is required")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	if err != nil {
		writeError(w, 400, "invalid request body")
		return
	}
	var messages []json.RawMessage
	if err := json.Unmarshal(body, &messages); err != nil {
		messages = []json.RawMessage{json.RawMessage(body)}
	}
	accepted := 0
	for _, message := range messages {
		envelope := domain.EventEnvelope{EventID: domain.StableID("tangseng-webhook", string(message)), Source: domain.SourceTang, Command: "msg.notify", OccurredAt: time.Now().Unix(), IdempotencyKey: domain.StableID("tangseng", string(message)), Payload: message}
		ok, acceptErr := s.Engine.Accept(r.Context(), envelope)
		if acceptErr != nil {
			s.failed.Add(1)
			writeError(w, 500, acceptErr.Error())
			return
		}
		if ok {
			accepted++
		}
	}
	s.received.Add(int64(len(messages)))
	s.accepted.Add(int64(accepted))
	writeJSON(w, http.StatusAccepted, map[string]any{"accepted": accepted, "received": len(messages)})
}

func (s *Server) receive(w http.ResponseWriter, r *http.Request, source domain.Source) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST is required")
		return
	}
	s.received.Add(1)
	var envelope domain.EventEnvelope
	if err := json.NewDecoder(io.LimitReader(r.Body, 8<<20)).Decode(&envelope); err != nil {
		s.failed.Add(1)
		writeError(w, 400, "invalid event JSON")
		return
	}
	if envelope.Source == "" {
		envelope.Source = source
	}
	if envelope.Source != source {
		s.failed.Add(1)
		writeError(w, 400, "event source does not match endpoint")
		return
	}
	accepted, err := s.Engine.Accept(r.Context(), envelope)
	if err != nil {
		s.failed.Add(1)
		writeError(w, 500, err.Error())
		return
	}
	if accepted {
		s.accepted.Add(1)
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"accepted": accepted, "event_id": envelope.EventID})
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	if s.Store == nil || s.Engine == nil {
		writeError(w, 503, "bridge is not ready")
		return
	}
	if _, err := s.Store.Stats(r.Context()); err != nil {
		writeError(w, 503, "store is not ready")
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ready"})
}

func (s *Server) metrics(w http.ResponseWriter, r *http.Request) {
	stats, err := s.Store.Stats(r.Context())
	if err != nil {
		writeError(w, 503, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "bridge_events_received %d\nbridge_events_accepted %d\nbridge_events_failed %d\nbridge_inbox_received %d\nbridge_outbox_pending %d\nbridge_outbox_dead %d\nbridge_messages_done %d\n", s.received.Load(), s.accepted.Load(), s.failed.Load(), stats.InboxReceived, stats.OutboxPending, stats.OutboxDead, stats.MessagesDone)
}

func (s *Server) replay(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "event replay requires an event-specific admin store adapter")
}

func (s *Server) retryDead(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && r.URL.Path == "/admin/dead-letter/" {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		items, err := s.Store.ListDeadLetters(r.Context(), limit)
		if err != nil {
			writeError(w, 503, err.Error())
			return
		}
		writeJSON(w, 200, map[string]any{"dead_letters": items})
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST is required")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/admin/dead-letter/")
	if id == "" {
		writeError(w, 400, "event id is required")
		return
	}
	if err := s.Store.RetryDeadLetter(r.Context(), id); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, store.ErrAlreadyDelivered) || errors.Is(err, store.ErrAmbiguousDelivery) {
			status = http.StatusConflict
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "queued", "job_id": id})
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		_ = strconv.FormatInt(time.Since(started).Milliseconds(), 10)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}
