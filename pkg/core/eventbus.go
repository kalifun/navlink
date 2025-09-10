package core

import (
	"context"
	"sync"
)

type MemoryEventBus struct {
	subscribers map[string][]EventHandler
	mu          sync.RWMutex
	context     context.Context
	cancel      context.CancelFunc
}

func (e *MemoryEventBus) Publish(ctx context.Context, topic string, msg DomainMessage) error {
	e.mu.RLock()
	handlers := e.subscribers[topic]
	e.mu.RUnlock()

	handlersCopy := make([]EventHandler, len(handlers))
	copy(handlersCopy, handlers)

	// Dispatch to all handlers concurrently
	var wg sync.WaitGroup
	for _, handler := range handlersCopy {
		wg.Add(1)
		go func(eh EventHandler) {
			defer wg.Done()
			if err := eh(ctx, msg); err != nil {
				// Log error but don't fail the publish
				// TODO: Add proper error logging
			}
		}(handler)
	}
	wg.Wait()
	return nil
}

func (e *MemoryEventBus) Subscribe(ctx context.Context, topic string, handler EventHandler) (EventSubscription, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.subscribers[topic] = append(e.subscribers[topic], handler)

	return &MemorySubscription{
		topic:   topic,
		handler: handler,
		bus:     e,
		ctx:     ctx,
		cancel:  e.cancel,
	}, nil
}

type MemorySubscription struct {
	topic   string
	handler EventHandler
	bus     *MemoryEventBus
	ctx     context.Context
	cancel  context.CancelFunc
}

func (s *MemorySubscription) Unsubscribe(ctx context.Context) error {
	s.bus.mu.Lock()
	defer s.bus.mu.Unlock()

	handlers := s.bus.subscribers[s.topic]
	// For MVP, we'll clear all handlers for this topic
	// TODO: Implement proper handler identification for multiple subscribers
	if len(handlers) > 0 {
		s.bus.subscribers[s.topic] = handlers[:0]
	}
	return nil
}

func (s *MemorySubscription) Topic() string {
	return s.topic
}
