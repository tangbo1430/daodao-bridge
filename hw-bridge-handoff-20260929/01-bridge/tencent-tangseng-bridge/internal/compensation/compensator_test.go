package compensation

import (
	"context"
	"testing"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
)

type fakeFetcher struct{ calls int }

func (f *fakeFetcher) Fetch(_ context.Context, cursor string, _ int) ([]domain.EventEnvelope, string, error) {
	f.calls++
	return []domain.EventEnvelope{{EventID: "e1", IdempotencyKey: "k1", Source: domain.SourceTang, Payload: []byte(`{"message_id":"m","from_uid":"a","channel_id":"b","content":"x"}`)}}, cursor + "next", nil
}

type fakeAcceptor struct{ count int }

func (f *fakeAcceptor) Accept(_ context.Context, _ domain.EventEnvelope) (bool, error) {
	f.count++
	return true, nil
}

func TestCompensatorAdvancesCursorAfterAccept(t *testing.T) {
	fetcher := &fakeFetcher{}
	acceptor := &fakeAcceptor{}
	c := New(fetcher, acceptor, 10)
	if _, err := c.RunOnce(context.Background(), "account-a"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.RunOnce(context.Background(), "account-a"); err != nil {
		t.Fatal(err)
	}
	if fetcher.calls != 2 || acceptor.count != 2 {
		t.Fatalf("calls=%d accepted=%d", fetcher.calls, acceptor.count)
	}
}
