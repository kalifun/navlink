package bus

import (
	"context"
	"sync"
)

// Handler consumes one event payload.
type Handler func(ctx context.Context, payload any) error

// Unsubscribe removes a subscription.
type Unsubscribe func(ctx context.Context) error

// Bus is a process-local event pub/sub.
type Bus interface {
	Publish(ctx context.Context, event string, payload any) error
	Subscribe(event string, h Handler) (Unsubscribe, error)
}

// Memory is a synchronous in-process Bus.
// Publish invokes subscribers in registration order and never drops messages.
type Memory struct {
	mu     sync.Mutex
	nextID uint64
	subs   map[string][]subscription
}

type subscription struct {
	id uint64
	h  Handler
}

// NewMemory creates an empty Memory bus.
func NewMemory() *Memory {
	return &Memory{subs: make(map[string][]subscription)}
}

// Publish delivers payload to all subscribers of event, in order.
func (m *Memory) Publish(ctx context.Context, event string, payload any) error {
	m.mu.Lock()
	subs := append([]subscription(nil), m.subs[event]...)
	m.mu.Unlock()

	for _, s := range subs {
		if err := s.h(ctx, payload); err != nil {
			return err
		}
	}
	return nil
}

// Subscribe registers a handler for event.
func (m *Memory) Subscribe(event string, h Handler) (Unsubscribe, error) {
	if h == nil {
		return nil, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	id := m.nextID
	m.subs[event] = append(m.subs[event], subscription{id: id, h: h})

	return func(ctx context.Context) error {
		m.mu.Lock()
		defer m.mu.Unlock()
		list := m.subs[event]
		for i, s := range list {
			if s.id == id {
				m.subs[event] = append(list[:i], list[i+1:]...)
				break
			}
		}
		return nil
	}, nil
}
