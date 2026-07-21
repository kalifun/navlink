package navlink

import (
	"context"

	"github.com/kalifun/vda5050-types-go/connection"
	"github.com/kalifun/vda5050-types-go/factsheet"
	"github.com/kalifun/vda5050-types-go/state"
	"github.com/kalifun/vda5050-types-go/visualization"

	"github.com/kalifun/navlink/bus"
)

// L1 protocol event names (stable; do not rename casually).
const (
	EventStateReceived         = "vda.state.received"
	EventConnectionChanged     = "vda.connection.changed"
	EventVisualizationReceived = "vda.visualization.received"
	EventFactsheetReceived     = "vda.factsheet.received"
	EventDecodeFailed          = "vda.decode.failed"
)

// EventHandler handles a bus event payload.
type EventHandler func(ctx context.Context, payload any) error

// EventBus is the optional protocol/custom event surface.
type EventBus interface {
	Publish(ctx context.Context, event string, payload any) error
	Subscribe(event string, h EventHandler) (Unsubscribe, error)
}

// StateEvent is the payload for EventStateReceived.
type StateEvent struct {
	Envelope Envelope
	State    *state.State
}

// ConnectionEvent is the payload for EventConnectionChanged.
type ConnectionEvent struct {
	Envelope   Envelope
	Connection *connection.Connection
}

// VisualizationEvent is the payload for EventVisualizationReceived.
type VisualizationEvent struct {
	Envelope      Envelope
	Visualization *visualization.Visualization
}

// FactsheetEvent is the payload for EventFactsheetReceived.
type FactsheetEvent struct {
	Envelope  Envelope
	Factsheet *factsheet.Factsheet
}

// DecodeFailedEvent is the payload for EventDecodeFailed.
type DecodeFailedEvent struct {
	Envelope Envelope
	Err      error
}

type eventBusShim struct {
	inner bus.Bus
}

// NewMemoryEventBus returns a process-local synchronous EventBus.
func NewMemoryEventBus() EventBus {
	return eventBusShim{inner: bus.NewMemory()}
}

func (s eventBusShim) Publish(ctx context.Context, event string, payload any) error {
	return s.inner.Publish(ctx, event, payload)
}

func (s eventBusShim) Subscribe(event string, h EventHandler) (Unsubscribe, error) {
	unsub, err := s.inner.Subscribe(event, bus.Handler(h))
	if err != nil {
		return nil, err
	}
	if unsub == nil {
		return nil, nil
	}
	return Unsubscribe(unsub), nil
}
