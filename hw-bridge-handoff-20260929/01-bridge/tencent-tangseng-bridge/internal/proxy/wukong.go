package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/session"
)

// EventEnvelope converts a WuKong recv frame to the common TangSeng inbox
// shape. It preserves the native payload and adds only routing metadata.
func EventEnvelope(uid string, event session.Event) domain.EventEnvelope {
	messageID := event.MessageID
	if messageID == "" {
		messageID = event.ClientMsgNo
	}
	if messageID == "" && event.MessageSeq != 0 {
		messageID = fmt.Sprintf("seq-%d", event.MessageSeq)
	}
	if messageID == "" {
		messageID = domain.StableID(uid, event.ChannelID, fmt.Sprint(event.Timestamp))
	}
	toUID := uid
	payload, _ := json.Marshal(map[string]any{"need_receipt": event.NeedReceipt, "message_id": messageID, "message_seq": event.MessageSeq, "client_msg_no": event.ClientMsgNo, "occurred_at": event.Timestamp, "channel_id": event.ChannelID, "channel_type": event.ChannelType, "from_uid": event.FromUID, "to_uid": toUID, "payload": json.RawMessage(event.Payload)})
	return domain.EventEnvelope{EventID: domain.StableID("tangseng-ws", uid, messageID), TraceID: domain.NewTraceID(), Source: domain.SourceTang, Command: "msg.notify", OccurredAt: event.Timestamp, IdempotencyKey: domain.MessageKey(domain.SourceTang, "msg.notify", messageID), Payload: payload}
}

// WuKongManager owns one PC(2) session per TangSeng shadow account. Sessions
// are created lazily after account login and are safe to use from the outbox
// workers. A later deployment can shard this manager behind the leader lease.
type WuKongManager struct {
	RefreshUser       func(context.Context, string, string) error
	ConnectionFailure func(context.Context, string, int) error
	WSURL             string
	DevicePrefix      string
	mu                sync.Mutex
	sessions          map[string]*session.Session
	tokens            map[string]string
	deviceIDs         map[string]string
	receivers         map[string]context.CancelFunc
}

func NewWuKongManager(wsURL string) *WuKongManager {
	return &WuKongManager{WSURL: wsURL, DevicePrefix: "hwbridge-", sessions: make(map[string]*session.Session), tokens: make(map[string]string), receivers: make(map[string]context.CancelFunc)}
}

func (m *WuKongManager) SetToken(uid, token string) {
	m.mu.Lock()
	if old := m.tokens[uid]; old != "" && old != token {
		if s := m.sessions[uid]; s != nil {
			_ = s.Close()
			delete(m.sessions, uid)
		}
	}
	m.tokens[uid] = token
	m.mu.Unlock()
}

func (m *WuKongManager) SetDeviceID(uid, deviceID string) {
	m.mu.Lock()
	if m.deviceIDs == nil {
		m.deviceIDs = make(map[string]string)
	}
	m.deviceIDs[uid] = deviceID
	m.mu.Unlock()
}

func (m *WuKongManager) session(ctx context.Context, uid string) (*session.Session, error) {
	m.mu.Lock()
	if existing := m.sessions[uid]; existing != nil {
		select {
		case <-existing.Done():
			delete(m.sessions, uid)
		default:
			m.mu.Unlock()
			return existing, nil
		}
	}
	token := m.tokens[uid]
	deviceID := m.deviceIDs[uid]
	m.mu.Unlock()
	if token == "" {
		return nil, fmt.Errorf("WuKong token is missing for %s", uid)
	}
	if deviceID == "" {
		deviceID = m.DevicePrefix + uid
	}
	s := session.New(session.Config{URL: m.WSURL, UID: uid, Token: token, DeviceID: deviceID})
	if err := s.Connect(ctx); err != nil {
		var failure *domain.DeliveryError
		if errors.As(err, &failure) && (failure.Code == 19 || failure.Code == 25) && m.ConnectionFailure != nil {
			if stateErr := m.ConnectionFailure(ctx, uid, failure.Code); stateErr != nil {
				return nil, fmt.Errorf("connection state: %w", stateErr)
			}
		}
		return nil, err
	}
	m.mu.Lock()
	m.sessions[uid] = s
	m.mu.Unlock()
	return s, nil
}

func (m *WuKongManager) SendText(ctx context.Context, uid, channelID string, channelType int, content, clientMsgNo string) (string, error) {
	if m == nil || m.WSURL == "" {
		return "", fmt.Errorf("WuKong proxy URL is not configured")
	}
	s, err := m.session(ctx, uid)
	if err != nil {
		return "", err
	}
	return s.SendText(ctx, channelID, channelType, content, clientMsgNo, true)
}

func (m *WuKongManager) VerifyConnection(ctx context.Context, uid string) error {
	_, err := m.session(ctx, uid)
	return err
}

func (m *WuKongManager) SendPayload(ctx context.Context, uid, channelID string, channelType int, payload json.RawMessage, clientMsgNo string) (string, error) {
	return m.SendPayloadWithReceipt(ctx, uid, channelID, channelType, payload, clientMsgNo, channelType == 1)
}
func (m *WuKongManager) SendPayloadWithReceipt(ctx context.Context, uid, channelID string, channelType int, payload json.RawMessage, clientMsgNo string, receipt bool) (string, error) {
	if m == nil || m.WSURL == "" {
		return "", fmt.Errorf("WuKong proxy URL is not configured")
	}
	s, err := m.session(ctx, uid)
	if err != nil {
		return "", err
	}
	return s.SendPayload(ctx, channelID, channelType, payload, clientMsgNo, receipt && channelType == 1)
}

func (m *WuKongManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for uid, cancel := range m.receivers {
		cancel()
		delete(m.receivers, uid)
	}
	for uid, s := range m.sessions {
		_ = s.Close()
		delete(m.sessions, uid)
	}
	return nil
}

// StartReceiveLoop owns one reconnecting PC(2) receive loop for a Tang UID.
// Starting it twice is idempotent; the callback is invoked only after the
// caller has durably accepted the event, and RunReceiveLoop then sends ack.
func (m *WuKongManager) StartReceiveLoop(ctx context.Context, uid string, handle func(context.Context, session.Event) error) error {
	if m == nil || uid == "" {
		return fmt.Errorf("uid is required")
	}
	m.mu.Lock()
	if _, ok := m.receivers[uid]; ok {
		m.mu.Unlock()
		return nil
	}
	loopCtx, cancel := context.WithCancel(ctx)
	m.receivers[uid] = cancel
	m.mu.Unlock()
	go func() {
		defer func() {
			m.mu.Lock()
			delete(m.receivers, uid)
			m.mu.Unlock()
		}()
		backoff := time.Second
		authFailures := 0
		for loopCtx.Err() == nil {
			m.mu.Lock()
			attemptedToken := m.tokens[uid]
			m.mu.Unlock()
			err := m.RunReceiveLoop(loopCtx, uid, handle)
			if loopCtx.Err() != nil {
				return
			}
			if err != nil {
				var failure *domain.DeliveryError
				if errors.As(err, &failure) {
					if failure.Code == 19 || failure.Code == 25 {
						return
					}
					if failure.Auth {
						authFailures++
						if authFailures > 1 || m.RefreshUser == nil {
							return
						}
						if refreshErr := m.RefreshUser(loopCtx, uid, attemptedToken); refreshErr != nil {
							return
						}
						continue
					}
				} else {
					authFailures = 0
				}
				select {
				case <-loopCtx.Done():
					return
				case <-time.After(backoff):
				}
				if backoff < 60*time.Second {
					backoff *= 2
					if backoff > 60*time.Second {
						backoff = 60 * time.Second
					}
				}
				continue
			}
			backoff = time.Second
		}
	}()
	return nil
}

func (m *WuKongManager) StopReceiveLoop(uid string) {
	m.mu.Lock()
	delete(m.tokens, uid)
	delete(m.deviceIDs, uid)
	if cancel := m.receivers[uid]; cancel != nil {
		cancel()
		delete(m.receivers, uid)
	}
	if s := m.sessions[uid]; s != nil {
		_ = s.Close()
		delete(m.sessions, uid)
	}
	m.mu.Unlock()
}

// RunReceiveLoop processes recv frames and acknowledges only after the caller
// confirms handling. It is intentionally exposed for the relay worker.
func (m *WuKongManager) RunReceiveLoop(ctx context.Context, uid string, handle func(context.Context, session.Event) error) error {
	s, err := m.session(ctx, uid)
	if err != nil {
		return err
	}
	for {
		select {
		case event, ok := <-s.Events():
			if !ok {
				return fmt.Errorf("session %s event stream closed", uid)
			}
			if err := handle(ctx, event); err != nil {
				// Do not acknowledge a failed durable handoff. WuKong will
				// redeliver it after reconnect and the inbox idempotency key
				// protects against duplicates.
				continue
			}
			if err := s.Ack(ctx, event); err != nil {
				return fmt.Errorf("ack %s: %w", event.MessageID, err)
			}
		case <-ctx.Done():
			return ctx.Err()
		case <-s.Done():
			return fmt.Errorf("session %s closed: %w", uid, s.Err())
		case <-time.After(time.Second):
		}
	}
}
