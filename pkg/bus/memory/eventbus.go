package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/kalifun/navlink/pkg/types"
	"github.com/sirupsen/logrus"
)

// MemoryEventBus is an in-memory implementation of the EventBus interface.
type MemoryEventBus struct {
	id          string
	mu          sync.RWMutex
	subscribers map[string][]chan *types.DomainMessage
	ctx         context.Context
	cancel      context.CancelFunc
	logger      *logrus.Entry
}

func (b *MemoryEventBus) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.cancel != nil {
		return fmt.Errorf("event bus %s is already running", b.id)
	}

	b.ctx, b.cancel = context.WithCancel(ctx)
	b.logger.Info("Event bus started")
	return nil
}

func (b *MemoryEventBus) Stop(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.cancel == nil {
		return fmt.Errorf("event bus %s is not running", b.id)
	}

	b.cancel()
	b.cancel = nil // Mark as stopped

	// Close all subscriber channels to signal completion
	for topic, subscribers := range b.subscribers {
		for _, ch := range subscribers {
			close(ch)
		}
		delete(b.subscribers, topic)
	}
	b.logger.Info("Event bus stopped")
	return nil
}

func (b *MemoryEventBus) Publish(ctx context.Context, msg *types.DomainMessage) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.ctx.Err() != nil {
		return fmt.Errorf("event bus is stopped: %w", b.ctx.Err())
	}

	topic := string(msg.Type)
	subscribers, ok := b.subscribers[topic]
	if !ok {
		b.logger.WithField("topic", topic).Debug("No subscribers for topic")
		return nil
	}

	b.logger.WithFields(logrus.Fields{
		"topic": topic,
		"count": len(subscribers),
	}).Debug("Publishing message to subscribers")

	for _, ch := range subscribers {
		// Use a non-blocking send to prevent a slow subscriber from blocking the publisher.
		select {
		case ch <- msg:
		case <-ctx.Done():
			return ctx.Err()
		default:
			b.logger.WithField("topic", topic).Warn("Subscriber channel is full. Message dropped.")
		}
	}

	return nil
}

func (b *MemoryEventBus) Subscribe(ctx context.Context, topic string) (<-chan *types.DomainMessage, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.ctx.Err() != nil {
		return nil, fmt.Errorf("event bus is stopped: %w", b.ctx.Err())
	}

	ch := make(chan *types.DomainMessage, 10)
	b.subscribers[topic] = append(b.subscribers[topic], ch)
	b.logger.WithField("topic", topic).Debug("New subscription added")
	return ch, nil
}

func (b *MemoryEventBus) Unsubscribe(ctx context.Context, topic string, sub <-chan *types.DomainMessage) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.ctx.Err() != nil {
		return fmt.Errorf("event bus is stopped: %w", b.ctx.Err())
	}

	subscribers, ok := b.subscribers[topic]
	if !ok {
		return fmt.Errorf("no subscribers for topic: %s", topic)
	}

	for i, ch := range subscribers {
		if ch == sub {
			close(ch)
			// Remove from slice without preserving order for efficiency
			b.subscribers[topic][i] = b.subscribers[topic][len(b.subscribers[topic])-1]
			b.subscribers[topic] = b.subscribers[topic][:len(b.subscribers[topic])-1]
			b.logger.WithField("topic", topic).Debug("Subscription removed")
			return nil
		}
	}
	return fmt.Errorf("subscription channel not found for topic: %s", topic)
}
