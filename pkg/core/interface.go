package core

import (
	"context"
	"time"

	"github.com/kalifun/navlink/pkg/types"
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
	Topic   string
	Payload []byte
	Meta    map[string]string
	Time    time.Time
}

// Subscription represents a message subscription
type Subscription interface {
	Unsubscribe(ctx context.Context) error
	Topic() string
}

// Conve// TransportPublish represents a message to be published
type TransportPublish struct {
	Topic   string
	Payload []byte
	Options PublishOptions
}

// EventBus handles internal message routing
type EventBus interface {
	LifecycleComponent
	Publish(ctx context.Context, msg *types.DomainMessage) error
	Subscribe(ctx context.Context, topic string) (<-chan *types.DomainMessage, error)
	Unsubscribe(ctx context.Context, topic string, sub <-chan *types.DomainMessage) error
}

// EventSubscription represents an event subscription
type EventSubscription interface {
	Unsubscribe(ctx context.Context) error
	Topic() string
}

type Processor interface {
	Process(ctx context.Context, msg *types.DomainMessage) error
	Type() string // Returns the message type this processor handles
}

// TransportFactory creates transport instances
type TransportFactory func(config map[string]interface{}) (Transport, error)

// ProcessorFactory creates processor instances
type ProcessorFactory func(config map[string]interface{}) (Processor, error)

type SessionStore interface {
	SaveSession(ctx context.Context, id string, meta SessionMeta) error
	GetSession(ctx context.Context, id string) (SessionMeta, error)
	DeleteSession(ctx context.Context, id string) error
	ListSessions(ctx context.Context) ([]SessionMeta, error)
	AddSubscription(ctx context.Context, sessionID string, sub SubscriptionMeta) error
	RemoveSubscription(ctx context.Context, sessionID, topic string) error
	ListSubscriptions(ctx context.Context, sessionID string) ([]SubscriptionMeta, error)
}
