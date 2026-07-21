package navlink

import (
	"context"
	"encoding/json"
	"time"

	vda5050 "github.com/kalifun/vda5050-types-go"
	"github.com/kalifun/vda5050-types-go/instant_actions"
	"github.com/kalifun/vda5050-types-go/order"

	"github.com/kalifun/navlink/topic"
)

// AGVHandle publishes typed outbound messages to one AGV.
type AGVHandle struct {
	client       *Client
	manufacturer string
	serial       string
}

// PublishOrder publishes an order after filling topic and basic header fields.
func (a *AGVHandle) PublishOrder(ctx context.Context, o *order.Order) error {
	if err := a.client.requireStarted(); err != nil {
		return err
	}
	if o == nil {
		return nil
	}
	a.fillHeader(&o.ProtocolHeader)
	payload, err := json.Marshal(o)
	if err != nil {
		return err
	}
	t := a.client.topics.Build(a.manufacturer, a.serial, topic.ChannelOrder)
	return a.client.transport.Publish(ctx, t, payload, PublishOptions{QoS: a.client.cfg.qos()})
}

// PublishInstantActions publishes instantActions after filling topic and basic header fields.
func (a *AGVHandle) PublishInstantActions(ctx context.Context, ia *instant_actions.InstantActions) error {
	if err := a.client.requireStarted(); err != nil {
		return err
	}
	if ia == nil {
		return nil
	}
	a.fillHeader(&ia.ProtocolHeader)
	payload, err := json.Marshal(ia)
	if err != nil {
		return err
	}
	t := a.client.topics.Build(a.manufacturer, a.serial, topic.ChannelInstantActions)
	return a.client.transport.Publish(ctx, t, payload, PublishOptions{QoS: a.client.cfg.qos()})
}

func (a *AGVHandle) fillHeader(h *vda5050.ProtocolHeader) {
	if h.Manufacturer == "" {
		h.Manufacturer = a.manufacturer
	}
	if h.SerialNumber == "" {
		h.SerialNumber = a.serial
	}
	if h.Version == "" {
		h.Version = a.client.cfg.Version
	}
	if h.Timestamp == "" {
		h.Timestamp = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	}
}
