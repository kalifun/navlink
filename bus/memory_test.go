package bus_test

import (
	"context"
	"testing"

	"github.com/kalifun/navlink/bus"
)

func TestMemoryMultiSubscribeOrder(t *testing.T) {
	b := bus.NewMemory()
	var order []int
	_, _ = b.Subscribe("e", func(ctx context.Context, payload any) error {
		order = append(order, 1)
		return nil
	})
	_, _ = b.Subscribe("e", func(ctx context.Context, payload any) error {
		order = append(order, 2)
		return nil
	})
	if err := b.Publish(context.Background(), "e", "x"); err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Fatalf("order=%v", order)
	}
}

func TestMemoryUnsubscribe(t *testing.T) {
	b := bus.NewMemory()
	var n int
	unsub, _ := b.Subscribe("e", func(ctx context.Context, payload any) error {
		n++
		return nil
	})
	_ = unsub(context.Background())
	_ = b.Publish(context.Background(), "e", nil)
	if n != 0 {
		t.Fatalf("n=%d", n)
	}
}
