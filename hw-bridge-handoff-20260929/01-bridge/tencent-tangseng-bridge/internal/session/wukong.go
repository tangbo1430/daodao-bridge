package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
)

// Config contains the per-user WuKong IM PC connection settings. The bridge
// never changes WuKong IM; it speaks the public JSON-RPC websocket protocol.
type Config struct {
	URL           string
	UID           string
	Token         string
	DeviceID      string
	DeviceFlag    int
	PingInterval  time.Duration
	PongWait      time.Duration
	WriteDeadline time.Duration
	HTTPHeader    http.Header
}

func (c Config) withDefaults() Config {
	if c.DeviceFlag == 0 {
		c.DeviceFlag = 2
	}
	if c.PingInterval <= 0 {
		c.PingInterval = 30 * time.Second
	}
	if c.PongWait <= 0 {
		c.PongWait = 10 * time.Second
	}
	if c.WriteDeadline <= 0 {
		c.WriteDeadline = 5 * time.Second
	}
	if c.DeviceID == "" {
		c.DeviceID = "hwbridge-" + c.UID
	}
	return c
}

type rpcFrame struct {
	JSONRPC string          `json:"jsonrpc,omitempty"`
	Method  string          `json:"method,omitempty"`
	ID      string          `json:"id,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type recvParams struct {
	Setting     json.RawMessage `json:"setting"`
	MessageID   string          `json:"messageId"`
	MessageSeq  int64           `json:"messageSeq"`
	ClientMsgNo string          `json:"clientMsgNo"`
	Timestamp   int64           `json:"timestamp"`
	ChannelID   string          `json:"channelId"`
	ChannelType int             `json:"channelType"`
	FromUID     string          `json:"fromUid"`
	Payload     json.RawMessage `json:"payload"`
}

// Event is the decoded server-side recv frame. Payload is kept intact so the
// converter can apply the same rules to websocket and webhook events.
type Event struct {
	NeedReceipt bool
	MessageID   string
	MessageSeq  int64
	ClientMsgNo string
	Timestamp   int64
	ChannelID   string
	ChannelType int
	FromUID     string
	Payload     json.RawMessage
}

type pendingResponse struct {
	frame rpcFrame
	err   error
}

// Session is one WuKong PC(2) websocket connection. A Session is safe for
// concurrent SendText/Ack calls; all reads are serialized by readLoop.
type Session struct {
	cfg       Config
	dialer    *websocket.Dialer
	conn      *websocket.Conn
	writeMu   sync.Mutex
	pendingMu sync.Mutex
	pending   map[string]chan pendingResponse
	events    chan Event
	done      chan struct{}
	closeOnce sync.Once
	seq       atomic.Uint64
	errMu     sync.Mutex
	readErr   error
}

func New(cfg Config) *Session {
	cfg = cfg.withDefaults()
	return &Session{cfg: cfg, dialer: websocket.DefaultDialer, pending: make(map[string]chan pendingResponse), events: make(chan Event, 128), done: make(chan struct{})}
}

func (s *Session) Config() Config { return s.cfg }

// Connect dials and completes the JSON-RPC connect handshake. Once it
// returns, recv frames are available through Events and SendText can be used.
func (s *Session) Connect(ctx context.Context) error {
	if s.cfg.URL == "" || s.cfg.UID == "" || s.cfg.Token == "" {
		return fmt.Errorf("wuKong websocket URL, uid and token are required")
	}
	conn, _, err := s.dialer.DialContext(ctx, s.cfg.URL, s.cfg.HTTPHeader)
	if err != nil {
		return err
	}
	s.conn = conn
	go s.readLoop()
	params := map[string]any{"uid": s.cfg.UID, "token": s.cfg.Token, "deviceFlag": s.cfg.DeviceFlag, "deviceId": s.cfg.DeviceID, "clientTimestamp": time.Now().UnixMilli(), "version": 4}
	frame, err := s.call(ctx, "connect", params)
	if err != nil {
		_ = s.Close()
		return err
	}
	var result struct {
		ReasonCode int    `json:"reasonCode"`
		Reason     string `json:"reason"`
		Message    string `json:"message"`
	}
	if err := json.Unmarshal(frame.Result, &result); err != nil {
		_ = s.Close()
		return fmt.Errorf("wuKong connect response missing result: %w", err)
	}
	if result.ReasonCode != 1 {
		_ = s.Close()
		return domain.WuKongFailure(result.ReasonCode)
	}
	go s.pingLoop()
	return nil
}

func first(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func (s *Session) nextID(prefix string) string { return fmt.Sprintf("%s-%d", prefix, s.seq.Add(1)) }

func (s *Session) call(ctx context.Context, method string, params any) (rpcFrame, error) {
	id := s.nextID(method)
	ch := make(chan pendingResponse, 1)
	s.pendingMu.Lock()
	s.pending[id] = ch
	s.pendingMu.Unlock()
	defer func() { s.pendingMu.Lock(); delete(s.pending, id); s.pendingMu.Unlock() }()
	if err := s.writeFrame(rpcFrame{Method: method, ID: id, Params: mustJSON(params)}); err != nil {
		return rpcFrame{}, err
	}
	select {
	case result := <-ch:
		if result.frame.Error != nil {
			return rpcFrame{}, fmt.Errorf("wuKong %s failed: code=%d message=%s", method, result.frame.Error.Code, result.frame.Error.Message)
		}
		return result.frame, result.err
	case <-ctx.Done():
		return rpcFrame{}, ctx.Err()
	case <-s.done:
		return rpcFrame{}, fmt.Errorf("wuKong session closed")
	}
}

func mustJSON(v any) json.RawMessage { data, _ := json.Marshal(v); return data }

func (s *Session) writeFrame(frame rpcFrame) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if s.conn == nil {
		return fmt.Errorf("wuKong session is not connected")
	}
	if frame.JSONRPC == "" {
		frame.JSONRPC = "2.0"
	}
	_ = s.conn.SetWriteDeadline(time.Now().Add(s.cfg.WriteDeadline))
	return s.conn.WriteJSON(frame)
}

func (s *Session) readLoop() {
	defer func() { _ = s.Close(); close(s.events) }()
	for {
		var frame rpcFrame
		if err := s.conn.ReadJSON(&frame); err != nil {
			s.errMu.Lock()
			s.readErr = err
			s.errMu.Unlock()
			return
		}
		if frame.ID != "" {
			s.pendingMu.Lock()
			ch := s.pending[frame.ID]
			s.pendingMu.Unlock()
			if ch != nil {
				ch <- pendingResponse{frame: frame}
				continue
			}
		}
		if frame.Method == "ping" {
			// WuKong normally uses client initiated JSON-RPC pings. Be tolerant
			// of a server initiated ping as well and answer its request id.
			_ = s.writeFrame(rpcFrame{ID: frame.ID, Result: mustJSON(nil)})
			continue
		}
		if frame.Method == "pong" {
			continue
		}
		if frame.Method == "recv" {
			var p recvParams
			if json.Unmarshal(frame.Params, &p) == nil {
				select {
				case s.events <- Event{NeedReceipt: receiptSetting(p.Setting), MessageID: p.MessageID, MessageSeq: p.MessageSeq, ClientMsgNo: p.ClientMsgNo, Timestamp: p.Timestamp, ChannelID: p.ChannelID, ChannelType: p.ChannelType, FromUID: p.FromUID, Payload: p.Payload}:
				case <-s.done:
					return
				}
			}
		}
	}
}

func (s *Session) pingLoop() {
	ticker := time.NewTicker(s.cfg.PingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), s.cfg.PongWait)
			_, err := s.call(ctx, "ping", map[string]any{})
			cancel()
			if err != nil {
				_ = s.Close()
				return
			}
		case <-s.done:
			return
		}
	}
}

func (s *Session) Events() <-chan Event { return s.events }

func (s *Session) Ack(ctx context.Context, event Event) error {
	if event.MessageID == "" {
		return fmt.Errorf("message id is required")
	}
	return s.writeFrame(rpcFrame{Method: "recvack", Params: mustJSON(map[string]any{"messageId": event.MessageID, "messageSeq": event.MessageSeq})})
}

func (s *Session) SendText(ctx context.Context, channelID string, channelType int, content, clientMsgNo string, receipt bool) (string, error) {
	return s.SendPayload(ctx, channelID, channelType, mustJSON(map[string]any{"type": 1, "content": content}), clientMsgNo, receipt)
}
func (s *Session) SendPayload(ctx context.Context, channelID string, channelType int, payload json.RawMessage, clientMsgNo string, receipt bool) (string, error) {
	if clientMsgNo == "" {
		clientMsgNo = s.nextID("msg")
	}
	params := map[string]any{"clientMsgNo": clientMsgNo, "channelId": channelID, "channelType": channelType, "payload": []byte(payload), "header": map[string]any{"redDot": true}, "setting": map[string]any{"receipt": receipt}}
	frame, err := s.call(ctx, "send", params)
	if err != nil {
		return "", err
	}
	var result struct {
		ReasonCode int    `json:"reasonCode"`
		Reason     string `json:"reason"`
		Message    string `json:"message"`
	}
	if err := json.Unmarshal(frame.Result, &result); err != nil {
		return "", fmt.Errorf("wuKong send response missing result: %w", err)
	}
	if result.ReasonCode != 1 {
		return "", domain.WuKongFailure(result.ReasonCode)
	}
	var value struct {
		MessageID   string `json:"messageId"`
		ClientMsgNo string `json:"clientMsgNo"`
		MessageSeq  int64  `json:"messageSeq"`
	}
	_ = json.Unmarshal(frame.Result, &value)
	if value.MessageID != "" {
		return value.MessageID, nil
	}
	if value.ClientMsgNo != "" {
		return value.ClientMsgNo, nil
	}
	return clientMsgNo, nil
}

func (s *Session) Close() error {
	s.closeOnce.Do(func() {
		close(s.done)
		if s.conn != nil {
			_ = s.conn.Close()
		}
	})
	return nil
}

func (s *Session) Done() <-chan struct{} { return s.done }

func (s *Session) Err() error {
	s.errMu.Lock()
	defer s.errMu.Unlock()
	if s.readErr != nil {
		return s.readErr
	}
	return errors.New("session closed")
}
