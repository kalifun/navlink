package testkit

import (
	"context"
	"sync"

	"github.com/kalifun/navlink"
)

// Recorder captures publish/subscribe activity for assertions.
type Recorder struct {
	mu          sync.Mutex
	published   []PublishedMessage
	subscribed  []string
	unsubscribed []string
}

// Published returns recorded publishes.
func (r *Recorder) Published() []PublishedMessage {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]PublishedMessage, len(r.published))
	copy(out, r.published)
	return out
}

// SubscribedFilters returns filters that were subscribed.
func (r *Recorder) SubscribedFilters() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.subscribed))
	copy(out, r.subscribed)
	return out
}

// RecordingTransport wraps a Transport and records pub/sub calls.
type RecordingTransport struct {
	Inner navlink.Transport
	Rec   *Recorder
}

// NewRecordingTransport wraps inner with a new Recorder when rec is nil.
func NewRecordingTransport(inner navlink.Transport, rec *Recorder) *RecordingTransport {
	if rec == nil {
		rec = &Recorder{}
	}
	return &RecordingTransport{Inner: inner, Rec: rec}
}

func (t *RecordingTransport) Start(ctx context.Context) error {
	return t.Inner.Start(ctx)
}

func (t *RecordingTransport) Stop(ctx context.Context) error {
	return t.Inner.Stop(ctx)
}

func (t *RecordingTransport) Publish(ctx context.Context, topic string, payload []byte, opts navlink.PublishOptions) error {
	err := t.Inner.Publish(ctx, topic, payload, opts)
	t.Rec.mu.Lock()
	t.Rec.published = append(t.Rec.published, PublishedMessage{
		Topic:   topic,
		Payload: append([]byte(nil), payload...),
		Opts:    opts,
	})
	t.Rec.mu.Unlock()
	return err
}

func (t *RecordingTransport) Subscribe(ctx context.Context, filter string, handler navlink.RawHandler) (navlink.Unsubscribe, error) {
	unsub, err := t.Inner.Subscribe(ctx, filter, handler)
	if err != nil {
		return nil, err
	}
	t.Rec.mu.Lock()
	t.Rec.subscribed = append(t.Rec.subscribed, filter)
	t.Rec.mu.Unlock()
	return func(ctx context.Context) error {
		t.Rec.mu.Lock()
		t.Rec.unsubscribed = append(t.Rec.unsubscribed, filter)
		t.Rec.mu.Unlock()
		return unsub(ctx)
	}, nil
}
