package navlink

import (
	"context"

	"github.com/kalifun/navlink/mqtt"
)

// mqttTransport adapts mqtt.Transport to navlink.Transport without import cycles.
// Errors already come from gerrors inside mqtt; no remapping here.
type mqttTransport struct {
	inner *mqtt.Transport
}

func newMQTTTransport(cfg mqtt.Config) Transport {
	return &mqttTransport{inner: mqtt.New(cfg)}
}

func (t *mqttTransport) Start(ctx context.Context) error {
	return t.inner.Start(ctx)
}

func (t *mqttTransport) Stop(ctx context.Context) error {
	return t.inner.Stop(ctx)
}

func (t *mqttTransport) Publish(ctx context.Context, topic string, payload []byte, opts PublishOptions) error {
	return t.inner.Publish(ctx, topic, payload, opts.QoS, opts.Retain)
}

func (t *mqttTransport) Subscribe(ctx context.Context, filter string, handler RawHandler) (Unsubscribe, error) {
	unsub, err := t.inner.Subscribe(ctx, filter, mqtt.Handler(handler))
	if err != nil {
		return nil, err
	}
	return Unsubscribe(unsub), nil
}
