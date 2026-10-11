package proxy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
)

// Connector isolates the actual WuKong/TangSeng PC protocol. The deployable pool
// handles lifecycle, reconnects and event handoff without assuming a client SDK.
type Connector interface {
	Connect(context.Context, string) (Connection, error)
}

type Connection interface {
	Receive(context.Context) (domain.EventEnvelope, error)
	Close() error
}

type EventHandler func(context.Context, domain.EventEnvelope) error

type Pool struct {
	Connector   Connector
	OnEvent     EventHandler
	MaxSessions int
	mu          sync.Mutex
	accounts    map[string]struct{}
}

func NewPool(connector Connector, onEvent EventHandler, maxSessions int) *Pool {
	if maxSessions < 1 {
		maxSessions = 500
	}
	return &Pool{Connector: connector, OnEvent: onEvent, MaxSessions: maxSessions, accounts: make(map[string]struct{})}
}

func (p *Pool) AddAccount(account string) error {
	if account == "" {
		return fmt.Errorf("proxy account is empty")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.accounts[account]; exists {
		return nil
	}
	if len(p.accounts) >= p.MaxSessions {
		return fmt.Errorf("proxy pool capacity %d reached", p.MaxSessions)
	}
	p.accounts[account] = struct{}{}
	return nil
}

func (p *Pool) Run(ctx context.Context) {
	p.mu.Lock()
	accounts := make([]string, 0, len(p.accounts))
	for account := range p.accounts {
		accounts = append(accounts, account)
	}
	p.mu.Unlock()
	for _, account := range accounts {
		go p.runAccount(ctx, account)
	}
}

func (p *Pool) runAccount(ctx context.Context, account string) {
	backoff := time.Second
	for ctx.Err() == nil {
		if p.Connector == nil {
			return
		}
		connection, err := p.Connector.Connect(ctx, account)
		if err != nil {
			if !sleep(ctx, backoff) {
				return
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second
		for ctx.Err() == nil {
			event, receiveErr := connection.Receive(ctx)
			if receiveErr != nil {
				_ = connection.Close()
				break
			}
			if p.OnEvent != nil {
				_ = p.OnEvent(ctx, event)
			}
		}
	}
}

func sleep(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
