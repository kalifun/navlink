package mqtt

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func startTestDispatch(t *testing.T, cfg Config) *Transport {
	t.Helper()
	tr := New(cfg)
	tr.mu.Lock()
	tr.startDispatchLocked()
	tr.mu.Unlock()
	t.Cleanup(func() {
		tr.shutdownDispatch()
	})
	return tr
}

func TestEnqueueStampsReceivedAtBeforeDispatch(t *testing.T) {
	tr := startTestDispatch(t, Config{InboundQueueSize: 4})

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

func TestAGVShardsDoNotBlockEachOther(t *testing.T) {
	tr := startTestDispatch(t, Config{InboundQueueSize: 8})

	block := make(chan struct{})
	s1Started := make(chan struct{})
	var s2Count atomic.Int32

	s1 := func(ctx context.Context, topic string, payload []byte, receivedAt time.Time) error {
		close(s1Started)
		<-block
		return nil
	}
	s2 := func(ctx context.Context, topic string, payload []byte, receivedAt time.Time) error {
		s2Count.Add(1)
		return nil
	}

	tr.enqueue("uagv/v2/M/S1/state", []byte(`1`), s1)
	select {
	case <-s1Started:
	case <-time.After(2 * time.Second):
		t.Fatal("S1 handler did not start")
	}

	for range 5 {
		tr.enqueue("uagv/v2/M/S2/state", []byte(`2`), s2)
	}

	deadline := time.After(2 * time.Second)
	for s2Count.Load() < 5 {
		select {
		case <-deadline:
			t.Fatalf("S2 blocked by S1; count=%d", s2Count.Load())
		case <-time.After(5 * time.Millisecond):
		}
	}
	close(block)
}

func TestConnectionLaneDoesNotStarveAGVState(t *testing.T) {
	tr := startTestDispatch(t, Config{InboundQueueSize: 8})

	block := make(chan struct{})
	connStarted := make(chan struct{})
	var stateCount atomic.Int32

	connH := func(ctx context.Context, topic string, payload []byte, receivedAt time.Time) error {
		close(connStarted)
		<-block
		return nil
	}
	stateH := func(ctx context.Context, topic string, payload []byte, receivedAt time.Time) error {
		stateCount.Add(1)
		return nil
	}

	tr.enqueue("uagv/v2/M/S1/connection", []byte(`c`), connH)
	select {
	case <-connStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("connection handler did not start")
	}

	for range 5 {
		tr.enqueue("uagv/v2/M/S1/state", []byte(`s`), stateH)
	}

	deadline := time.After(2 * time.Second)
	for stateCount.Load() < 5 {
		select {
		case <-deadline:
			t.Fatalf("state starved by connection; count=%d", stateCount.Load())
		case <-time.After(5 * time.Millisecond):
		}
	}
	close(block)
}

func TestTopicLaneDoesNotStarveAGVState(t *testing.T) {
	tr := startTestDispatch(t, Config{InboundQueueSize: 8})

	block := make(chan struct{})
	topicStarted := make(chan struct{})
	var stateCount atomic.Int32

	topicH := func(ctx context.Context, topic string, payload []byte, receivedAt time.Time) error {
		close(topicStarted)
		<-block
		return nil
	}
	stateH := func(ctx context.Context, topic string, payload []byte, receivedAt time.Time) error {
		stateCount.Add(1)
		return nil
	}

	tr.enqueue("fleet/directory/refresh", []byte(`x`), topicH)
	select {
	case <-topicStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("topic handler did not start")
	}

	for range 5 {
		tr.enqueue("uagv/v2/M/S1/state", []byte(`s`), stateH)
	}

	deadline := time.After(2 * time.Second)
	for stateCount.Load() < 5 {
		select {
		case <-deadline:
			t.Fatalf("state starved by OnTopic lane; count=%d", stateCount.Load())
		case <-time.After(5 * time.Millisecond):
		}
	}
	close(block)
}

func TestSameAGVStatePreservesOrder(t *testing.T) {
	tr := startTestDispatch(t, Config{InboundQueueSize: 16})

	var (
		mu   sync.Mutex
		got  []string
		done = make(chan struct{})
	)
	handler := func(ctx context.Context, topic string, payload []byte, receivedAt time.Time) error {
		mu.Lock()
		got = append(got, string(payload))
		n := len(got)
		mu.Unlock()
		if n == 3 {
			close(done)
		}
		return nil
	}

	tr.enqueue("uagv/v2/M/S1/state", []byte(`a`), handler)
	tr.enqueue("uagv/v2/M/S1/state", []byte(`b`), handler)
	tr.enqueue("uagv/v2/M/S1/state", []byte(`c`), handler)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("order=%v", got)
	}
}

func TestInboundDropReasons(t *testing.T) {
	var (
		mu      sync.Mutex
		reasons []DropReason
		topics  []string
	)
	tr := startTestDispatch(t, Config{
		InboundQueueSize: 1,
		OnInboundDrop: func(topic string, reason DropReason) {
			mu.Lock()
			topics = append(topics, topic)
			reasons = append(reasons, reason)
			mu.Unlock()
		},
	})

	block := make(chan struct{})
	started := make(chan struct{})
	var startOnce sync.Once
	hold := func(ctx context.Context, topic string, payload []byte, receivedAt time.Time) error {
		startOnce.Do(func() { close(started) })
		<-block
		return nil
	}

	// Fill viz shard: one in-flight + one queued.
	tr.enqueue("uagv/v2/M/S1/visualization", []byte(`1`), hold)
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("viz handler did not start")
	}
	tr.enqueue("uagv/v2/M/S1/visualization", []byte(`2`), hold) // fills buffer
	tr.enqueue("uagv/v2/M/S1/visualization", []byte(`3`), hold) // should drop

	deadline := time.After(2 * time.Second)
	for {
		mu.Lock()
		n := len(reasons)
		mu.Unlock()
		if n >= 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("expected DropDiscarded for viz")
		case <-time.After(5 * time.Millisecond):
		}
	}
	mu.Lock()
	if reasons[0] != DropDiscarded {
		t.Fatalf("viz reason=%v", reasons[0])
	}
	mu.Unlock()

	// State backpressure: fill S2 shard then observe Backpressured (non-blocking check via hook).
	s2block := make(chan struct{})
	s2started := make(chan struct{})
	var s2once sync.Once
	s2hold := func(ctx context.Context, topic string, payload []byte, receivedAt time.Time) error {
		s2once.Do(func() { close(s2started) })
		<-s2block
		return nil
	}
	tr.enqueue("uagv/v2/M/S2/state", []byte(`1`), s2hold)
	select {
	case <-s2started:
	case <-time.After(2 * time.Second):
		t.Fatal("state handler did not start")
	}
	tr.enqueue("uagv/v2/M/S2/state", []byte(`2`), s2hold) // buffer full

	backpressured := make(chan struct{})
	go func() {
		tr.enqueue("uagv/v2/M/S2/state", []byte(`3`), s2hold)
		close(backpressured)
	}()

	deadline = time.After(2 * time.Second)
	for {
		mu.Lock()
		hasBP := false
		for _, r := range reasons {
			if r == DropBackpressured {
				hasBP = true
				break
			}
		}
		mu.Unlock()
		if hasBP {
			break
		}
		select {
		case <-deadline:
			t.Fatal("expected DropBackpressured for state")
		case <-time.After(5 * time.Millisecond):
		}
	}

	close(block)
	close(s2block)
	select {
	case <-backpressured:
	case <-time.After(2 * time.Second):
		t.Fatal("blocked enqueue did not finish after drain")
	}
}

func TestConnectionCoalesceLatestWins(t *testing.T) {
	tr := startTestDispatch(t, Config{InboundQueueSize: 8})

	var (
		mu      sync.Mutex
		got     []string
		started = make(chan struct{})
		block   = make(chan struct{})
	)
	handler := func(ctx context.Context, topic string, payload []byte, receivedAt time.Time) error {
		mu.Lock()
		got = append(got, string(payload))
		n := len(got)
		mu.Unlock()
		if n == 1 {
			close(started)
			<-block
		}
		return nil
	}

	tr.enqueue("uagv/v2/M/S1/connection", []byte(`online-1`), handler)
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("first connection handler did not start")
	}

	tr.enqueue("uagv/v2/M/S1/connection", []byte(`offline`), handler)
	tr.enqueue("uagv/v2/M/S1/connection", []byte(`online-2`), handler)
	close(block)

	deadline := time.After(2 * time.Second)
	for {
		mu.Lock()
		n := len(got)
		snapshot := append([]string(nil), got...)
		mu.Unlock()
		if n >= 2 {
			if n != 2 || snapshot[0] != "online-1" || snapshot[1] != "online-2" {
				t.Fatalf("got=%v want [online-1 online-2]", snapshot)
			}
			return
		}
		select {
		case <-deadline:
			t.Fatalf("got=%v", snapshot)
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func TestConnectionCoalesceKeepsDistinctAGVs(t *testing.T) {
	tr := startTestDispatch(t, Config{InboundQueueSize: 8})

	var (
		mu   sync.Mutex
		got  []string
		done = make(chan struct{})
	)
	handler := func(ctx context.Context, topic string, payload []byte, receivedAt time.Time) error {
		mu.Lock()
		got = append(got, string(payload))
		n := len(got)
		mu.Unlock()
		if n == 2 {
			close(done)
		}
		return nil
	}

	tr.enqueue("uagv/v2/M/S1/connection", []byte(`s1`), handler)
	tr.enqueue("uagv/v2/M/S2/connection", []byte(`s2`), handler)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 2 {
		t.Fatalf("got=%v", got)
	}
	seen := map[string]bool{got[0]: true, got[1]: true}
	if !seen["s1"] || !seen["s2"] {
		t.Fatalf("got=%v", got)
	}
}
