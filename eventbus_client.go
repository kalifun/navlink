package navlink

import (
	"context"

	"github.com/kalifun/navlink/gerrors"
)

// UseEventBus attaches a bus and migrates any already-registered On* handlers onto it.
// After this call, On* registers as bus subscribers; inbound messages Publish to the bus.
func (c *Client) UseEventBus(bus EventBus) {
	if bus == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bus = bus

	for _, h := range c.stateHandlers {
		c.subscribeTypedLocked(EventStateReceived, wrapState(h))
	}
	c.stateHandlers = nil
	for _, h := range c.connHandlers {
		c.subscribeTypedLocked(EventConnectionChanged, wrapConnection(h))
	}
	c.connHandlers = nil
	for _, h := range c.vizHandlers {
		c.subscribeTypedLocked(EventVisualizationReceived, wrapVisualization(h))
	}
	c.vizHandlers = nil
	for _, h := range c.fsHandlers {
		c.subscribeTypedLocked(EventFactsheetReceived, wrapFactsheet(h))
	}
	c.fsHandlers = nil
}

// EventBus returns the attached bus, if any.
func (c *Client) EventBus() EventBus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.bus
}

// Emit publishes a custom (or L1) event. Requires an attached EventBus.
func (c *Client) Emit(event string, payload any) error {
	c.mu.RLock()
	bus := c.bus
	c.mu.RUnlock()
	if bus == nil {
		return gerrors.NewInvalidConfigWithArgs("EventBus is not configured")
	}
	return bus.Publish(context.Background(), event, payload)
}

// Subscribe registers a handler for an event name (L1 or platform custom).
// Subscribing to an L1 event also marks the corresponding MQTT channel as wanted.
func (c *Client) Subscribe(event string, h EventHandler) (Unsubscribe, error) {
	if h == nil {
		return nil, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.markEventWantedLocked(event)
	if c.bus == nil {
		return nil, gerrors.NewInvalidConfigWithArgs("EventBus is not configured")
	}
	return c.subscribeTypedLocked(event, h)
}

func (c *Client) subscribeTypedLocked(event string, h EventHandler) (Unsubscribe, error) {
	unsub, err := c.bus.Subscribe(event, h)
	if err != nil {
		return nil, err
	}
	if unsub != nil {
		c.busUnsubs = append(c.busUnsubs, unsub)
	}
	return unsub, nil
}

func (c *Client) markEventWantedLocked(event string) {
	switch event {
	case EventStateReceived:
		c.wantState = true
	case EventConnectionChanged:
		c.wantConn = true
	case EventVisualizationReceived:
		c.wantViz = true
	case EventFactsheetReceived:
		c.wantFS = true
	}
}

func wrapState(h StateHandler) EventHandler {
	return func(ctx context.Context, payload any) error {
		ev, ok := payload.(StateEvent)
		if !ok {
			return nil
		}
		return h(ctx, ev.Envelope, ev.State)
	}
}

func wrapConnection(h ConnectionHandler) EventHandler {
	return func(ctx context.Context, payload any) error {
		ev, ok := payload.(ConnectionEvent)
		if !ok {
			return nil
		}
		return h(ctx, ev.Envelope, ev.Connection)
	}
}

func wrapVisualization(h VisualizationHandler) EventHandler {
	return func(ctx context.Context, payload any) error {
		ev, ok := payload.(VisualizationEvent)
		if !ok {
			return nil
		}
		return h(ctx, ev.Envelope, ev.Visualization)
	}
}

func wrapFactsheet(h FactsheetHandler) EventHandler {
	return func(ctx context.Context, payload any) error {
		ev, ok := payload.(FactsheetEvent)
		if !ok {
			return nil
		}
		return h(ctx, ev.Envelope, ev.Factsheet)
	}
}

func (c *Client) publishEvent(ctx context.Context, event string, payload any) error {
	c.mu.RLock()
	bus := c.bus
	c.mu.RUnlock()
	if bus == nil {
		return nil
	}
	return bus.Publish(ctx, event, payload)
}
