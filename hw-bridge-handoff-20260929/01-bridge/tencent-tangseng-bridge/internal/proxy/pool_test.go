package proxy

import "testing"

func TestPoolCapacity(t *testing.T) {
	p := NewPool(nil, nil, 1)
	if err := p.AddAccount("a"); err != nil {
		t.Fatal(err)
	}
	if err := p.AddAccount("b"); err == nil {
		t.Fatal("expected capacity error")
	}
}
