package core

import (
	"context"
	"time"
)

// Transport defines the interface for message transport
type Transport interface {
	ID() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Publish(ctx context.Context, topic string, payload []byte, opts PublishOptions) error
	Subscribe(ctx context.Context, topic string, handler MessageHandler) (Subscription, error)
}

// PublishOptions for publishing messages
type PublishOptions struct {
	QoS     byte
	Retain  bool
	TimeOut time.Duration
}

type MessageHandler = func(ctx context.Context, msg *Message) error

// Message represents a transport-level message
type Message struct {
	Topic  string
	Paylod []byte
	Meta   map[string]string
	Time   time.Time
}

// Subscription represents a message subscription
type Subscription interface {
	Unsubscribe(ctx context.Context) error
	Topic() string
}

// Converter handles protocol conversion
type Converter interface {
	ToDomainMsg(ctx context.Context, msg *Message) (*DomainMessage, error)
	FromDomainMsg(ctx context.Context, msg *DomainMessage) (TransportPublish, error)
}

type DomainMessage struct {
	ID        string            // Unique message ID for idempotency
	Type      string            // Domain type: Order/State/Action/Event
	Source    string            // Original source
	Payload   any               // Deserialized payload (dynamic)
	Meta      map[string]string // Metadata (correlationId, tenant, etc.)
	Timestamp time.Time         // Message Timestamp
}

// TransportPublish represents a message to be published
type TransportPublish struct {
	Topic   string
	Payload []byte
	Options PublishOptions
}

// EventBus handles internal message routing
type EventBus interface {
	Publish(ctx context.Context, topic string, msg *DomainMessage) error
	Subscribe(ctx context.Context, topic string, handler EventHandler) (EventSubscription, error)
}

// EventHandler handles domain events
type EventHandler = func(ctx context.Context, msg *DomainMessage) error

// EventSubscription represents an event subscription
type EventSubscription interface {
	Unsubscribe(ctx context.Context) error
	Topic() string
}

type Router interface {
	Route(ctx context.Context, msg *DomainMessage) error
	RegisterProcessor(msgType string, processor Processor)
}

type Processor interface {
	Process(ctx context.Context, msg *DomainMessage) error
	Type() string // Returns the message type this processor handles
}
