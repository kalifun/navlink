package mqtt

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestEnqueueStampsReceivedAtBeforeDispatch(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	tr := &Transport{
		inbound:        make(chan inbound, 1),
		dispatchCtx:    ctx,
		dispatchCancel: cancel,
	}
	tr.dispatchWG.Add(1)
	go tr.runDispatch(ctx)
	t.Cleanup(func() {
		cancel()
		tr.dispatchWG.Wait()
	})

	var (
		mu         sync.Mutex
		receivedAt time.Time
		started    = make(chan struct{})
		done       = make(chan struct{})
	)
	handler := func(ctx context.Context, topic string, payload []byte, stamped time.Time) error {
		close(started)
		time.Sleep(40 * time.Millisecond)
		mu.Lock()
		receivedAt = stamped
		mu.Unlock()
		close(done)
		return nil
	}

	before := time.Now().UTC()
	tr.enqueue("uagv/v2/M/S1/state", []byte(`{}`), handler)
	select {
	case <-started:
	case <-t.Context().Done():
		t.Fatal("handler never started")
	}
	select {
	case <-done:
	case <-t.Context().Done():
		t.Fatal("handler never finished")
	}
	mu.Lock()
	got := receivedAt
	mu.Unlock()
	if got.IsZero() {
		t.Fatal("missing stamp")
	}
	if got.Before(before.Add(-time.Second)) || got.After(before.Add(time.Second)) {
		t.Fatalf("receivedAt=%s before=%s", got, before)
	}
	if time.Since(got) < 30*time.Millisecond {
		t.Fatalf("stamp looks like dispatch time: %s ago", time.Since(got))
	}
}
