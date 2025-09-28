package router

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/kalifun/navlink/pkg/core"
	"github.com/kalifun/navlink/pkg/types"
	"github.com/sirupsen/logrus"
)

// Router subscribes to the event bus and routes messages to the appropriate processor.
type Router struct {
	id         string
	eventBus   core.EventBus
	processors map[types.DomainMessageType]core.Processor
	logger     *logrus.Entry
	wg         sync.WaitGroup
	cancel     context.CancelFunc
}

// New creates a new Router.
func New(eventBus core.EventBus, processors []core.Processor) *Router {
	id := fmt.Sprintf("router-%s", uuid.New().String())
	procMap := make(map[types.DomainMessageType]core.Processor)
	for _, p := range processors {
		procMap[p.Type()] = p
	}

	return &Router{
		eventBus:   eventBus,
		id:         id,
		processors: procMap,
		logger:     logrus.WithField("component", id),
	}
}

// ID returns the unique identifier of the router.
func (r *Router) ID() string {
	return r.id
}

// Start begins the routing process.
func (r *Router) Start(ctx context.Context) error {
	ctx, r.cancel = context.WithCancel(ctx)

	for msgType, proc := range r.processors {
		currentProc := proc

		topic := string(msgType)
		msgChan, err := r.eventBus.Subscribe(ctx, topic)
		if err != nil {
			return fmt.Errorf("router failed to subscribe to topic %s: %w", topic, err)
		}
		r.wg.Add(1)
		go func(msgType types.DomainMessageType) {
			defer r.wg.Done()
			r.logger.Infof("Starting processor for message type: %s", msgType)
			for {
				select {
				case <-ctx.Done():
					r.logger.Infof("Stopping processor for message type: %s", msgType)
					return
				case msg, ok := <-msgChan:
					if !ok {
						r.logger.Infof("Channel closed for message type: %s", msgType)
						return
					}

					if err := currentProc.Process(ctx, msg); err != nil {
						r.logger.WithError(err).Errorf("Error processing message %d", msg.Header.HeaderId)
					}

				}
			}
		}(msgType)
	}

	// Also subscribe to the wildcard topic if no specific processor handles it.
	// This is useful for logging unhandled messages.
	if _, exists := r.processors[types.DomainMessageTypeWildcard]; !exists {
		go r.handleWildcard(ctx)
	}

	r.logger.Info("Router started")

	return nil
}

func (r *Router) handleWildcard(ctx context.Context) {
	wildcardChan, err := r.eventBus.Subscribe(ctx, string(types.DomainMessageTypeWildcard))
	if err != nil {
		r.logger.Errorf("Failed to subscribe to wildcard topic: %v", err)
		return
	}

	r.wg.Add(1)
	defer r.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-wildcardChan:
			if !ok {
				return
			}
			// If a specific processor for this type exists, it has already been handled.
			// We only log if there is NO specific processor.
			if _, exists := r.processors[msg.Type]; !exists {
				r.logger.WithField("type", msg.Type).Warn("Received message with no registered processor")
			}
		}
	}
}

// Stop gracefully shuts down the router.
func (r *Router) Stop(ctx context.Context) error {
	if r.cancel != nil {
		r.cancel()
	}
	r.wg.Wait()
	r.logger.Info("Router stopped")
	return nil
}
