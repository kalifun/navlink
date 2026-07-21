package navlink

import "context"

// PublishOptions controls MQTT publish behaviour.
type PublishOptions struct {
	QoS    byte
	Retain bool
}

// RawHandler handles a transport-level message (topic + payload).
type RawHandler func(ctx context.Context, topic string, payload []byte) error

// Unsubscribe cancels a subscription.
type Unsubscribe func(ctx context.Context) error

// Transport is the byte-level pub/sub boundary (MQTT, memory, etc.).
// VDA typed APIs live on Client; raw MQTT must not be mixed into AGV helpers.
type Transport interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Publish(ctx context.Context, topic string, payload []byte, opts PublishOptions) error
	Subscribe(ctx context.Context, filter string, handler RawHandler) (Unsubscribe, error)
}

// ReconnectAware is implemented by transports that can signal reconnects.
// The handler is invoked after a successful reconnect, not on the initial connect.
type ReconnectAware interface {
	SetOnReconnect(fn func())
}
