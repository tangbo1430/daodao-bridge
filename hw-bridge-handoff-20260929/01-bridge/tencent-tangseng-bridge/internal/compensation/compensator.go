package compensation

import (
	"context"
	"sync"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
)

type Fetcher interface {
	Fetch(context.Context, string, int) ([]domain.EventEnvelope, string, error)
}

type Acceptor interface {
	Accept(context.Context, domain.EventEnvelope) (bool, error)
}

type Compensator struct {
	Fetcher  Fetcher
	Acceptor Acceptor
	Limit    int
	mu       sync.Mutex
	cursors  map[string]string
}

func New(fetcher Fetcher, acceptor Acceptor, limit int) *Compensator {
	if limit < 1 {
		limit = 100
	}
	return &Compensator{Fetcher: fetcher, Acceptor: acceptor, Limit: limit, cursors: make(map[string]string)}
}

func (c *Compensator) RunOnce(ctx context.Context, partition string) (int, error) {
	if c.Fetcher == nil || c.Acceptor == nil {
		return 0, nil
	}
	c.mu.Lock()
	cursor := c.cursors[partition]
	c.mu.Unlock()
	events, next, err := c.Fetcher.Fetch(ctx, cursor, c.Limit)
	if err != nil {
		return 0, err
	}
	accepted := 0
	for _, event := range events {
		ok, err := c.Acceptor.Accept(ctx, event)
		if err != nil {
			return accepted, err
		}
		if ok {
			accepted++
		}
	}
	if next != "" {
		c.mu.Lock()
		c.cursors[partition] = next
		c.mu.Unlock()
	}
	return accepted, nil
}
