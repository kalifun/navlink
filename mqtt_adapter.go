package navlink

import (
	"context"
	"errors"

	"github.com/kalifun/navlink/internal/mqtt"
)

// mqttTransport adapts mqtt.Transport to navlink.Transport without import cycles.
type mqttTransport struct {
	inner *mqtt.Transport
}

func newMQTTTransport(cfg mqtt.Config) *mqttTransport {
	return &mqttTransport{inner: mqtt.New(cfg)}
}

func (t *mqttTransport) Start(ctx context.Context) error {
	return t.inner.Start(ctx)
}

func (t *mqttTransport) Stop(ctx context.Context) error {
	return t.inner.Stop(ctx)
}

func (t *mqttTransport) Publish(ctx context.Context, topic string, payload []byte, opts PublishOptions) error {
	err := t.inner.Publish(ctx, topic, payload, opts.QoS, opts.Retain)
	var att *mqtt.AttemptedError
	if errors.As(err, &att) {
		return MarkPublishAttempted(att.Err)
	}
	return err
}

func (t *mqttTransport) Subscribe(ctx context.Context, filter string, handler RawHandler) (Unsubscribe, error) {
	unsub, err := t.inner.Subscribe(ctx, filter, mqtt.Handler(handler))
	if err != nil {
		return nil, err
	}
	return Unsubscribe(unsub), nil
}

func (t *mqttTransport) SetOnReconnect(fn func()) {
	t.inner.SetOnReconnect(fn)
}

func (t *mqttTransport) SetOnConnectionLost(fn func(error)) {
	t.inner.SetOnConnectionLost(fn)
}

func (t *mqttTransport) Connected() bool {
	return t.inner.Connected()
}
