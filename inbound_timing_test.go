package navlink

import (
	"context"
	"testing"
	"time"
)

type timingTransport struct{}

func (timingTransport) Start(context.Context) error { return nil }
func (timingTransport) Stop(context.Context) error  { return nil }
func (timingTransport) Publish(context.Context, string, []byte, PublishOptions) error {
	return nil
}
func (timingTransport) Subscribe(context.Context, string, RawHandler) (Unsubscribe, error) {
	return func(context.Context) error { return nil }, nil
}

func TestEnvelopeQueueWait(t *testing.T) {
	recv := time.Date(2026, 9, 9, 1, 2, 3, 0, time.UTC)
	env := Envelope{ReceivedAt: recv, DispatchedAt: recv.Add(1500 * time.Millisecond)}
	if env.QueueWait() != 1500*time.Millisecond {
		t.Fatalf("QueueWait=%s", env.QueueWait())
	}
	if (Envelope{}).QueueWait() != 0 {
		t.Fatal("zero envelope")
	}
}

func TestDispatchTopicUsesEnqueueTime(t *testing.T) {
	c, err := New(Config{Interface: "uagv", Version: "v2", Transport: timingTransport{}})
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()
	if err := c.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Stop(ctx) })

	enqueued := time.Now().UTC().Add(-1500 * time.Millisecond)
	ctx = contextWithInboundReceivedAt(ctx, enqueued)
	var env Envelope
	if err := c.dispatchTopic(ctx, "uagv/v2/M/S1/order", []byte(`{}`), func(ctx context.Context, e Envelope) error {
		env = e
		time.Sleep(20 * time.Millisecond)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !env.ReceivedAt.Equal(enqueued) {
		t.Fatalf("ReceivedAt=%s want %s", env.ReceivedAt, enqueued)
	}
	if env.DispatchedAt.Before(enqueued) {
		t.Fatalf("DispatchedAt=%s before enqueue", env.DispatchedAt)
	}
	if env.QueueWait() < time.Second {
		t.Fatalf("QueueWait=%s", env.QueueWait())
	}
}

func TestSlowInboundReportsQueueAndHandler(t *testing.T) {
	c, err := New(Config{Interface: "uagv", Version: "v2", Transport: timingTransport{}})
	if err != nil {
		t.Fatal(err)
	}
	var causes []InboundSlowCause
	c.cfg.SlowInbound = 10 * time.Millisecond
	c.cfg.OnSlowInbound = func(env Envelope, cause InboundSlowCause, d time.Duration) {
		causes = append(causes, cause)
	}
	ctx := t.Context()
	if err := c.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Stop(ctx) })

	enqueued := time.Now().UTC().Add(-50 * time.Millisecond)
	ctx = contextWithInboundReceivedAt(ctx, enqueued)
	if err := c.dispatchTopic(ctx, "uagv/v2/M/S1/order", []byte(`{}`), func(ctx context.Context, e Envelope) error {
		time.Sleep(25 * time.Millisecond)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(causes) < 2 {
		t.Fatalf("causes=%v", causes)
	}
	if causes[0] != InboundSlowQueue || causes[1] != InboundSlowHandler {
		t.Fatalf("causes=%v", causes)
	}
}
