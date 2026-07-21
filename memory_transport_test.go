package navlink_test

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/kalifun/navlink"
)

type failTransport struct{}

func (t *failTransport) Start(ctx context.Context) error { return nil }
func (t *failTransport) Stop(ctx context.Context) error  { return nil }
func (t *failTransport) Publish(ctx context.Context, topic string, payload []byte, opts navlink.PublishOptions) error {
	return errors.New("publish failed")
}
func (t *failTransport) Subscribe(ctx context.Context, filter string, handler navlink.RawHandler) (navlink.Unsubscribe, error) {
	return func(ctx context.Context) error { return nil }, nil
}

// memoryTransport is an in-process Transport for unit tests.
type memoryTransport struct {
	mu      sync.Mutex
	running bool
	subs    []memSub
	published []pub
}

type memSub struct {
	filter  string
	handler navlink.RawHandler
}

type pub struct {
	topic   string
	payload []byte
}

func (t *memoryTransport) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.running = true
	return nil
}

func (t *memoryTransport) Stop(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.running = false
	t.subs = nil
	return nil
}

func (t *memoryTransport) Publish(ctx context.Context, topic string, payload []byte, opts navlink.PublishOptions) error {
	t.mu.Lock()
	t.published = append(t.published, pub{topic: topic, payload: append([]byte(nil), payload...)})
	subs := append([]memSub(nil), t.subs...)
	t.mu.Unlock()

	for _, s := range subs {
		if topicMatch(s.filter, topic) {
			if err := s.handler(ctx, topic, payload); err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *memoryTransport) Subscribe(ctx context.Context, filter string, handler navlink.RawHandler) (navlink.Unsubscribe, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.subs = append(t.subs, memSub{filter: filter, handler: handler})
	return func(ctx context.Context) error { return nil }, nil
}

func topicMatch(filter, topic string) bool {
	fp := strings.Split(filter, "/")
	tp := strings.Split(topic, "/")
	if len(fp) != len(tp) {
		return false
	}
	for i := range fp {
		if fp[i] == "+" || fp[i] == "#" {
			continue
		}
		if fp[i] != tp[i] {
			return false
		}
	}
	return true
}
