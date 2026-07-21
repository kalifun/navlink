package navlink

import (
	"context"
	"encoding/json"

	vda5050 "github.com/kalifun/vda5050-types-go"
	"github.com/kalifun/vda5050-types-go/instant_actions"
	"github.com/kalifun/vda5050-types-go/order"

	"github.com/kalifun/navlink/gerrors"
	"github.com/kalifun/navlink/topic"
)

// AGVHandle publishes typed outbound messages to one AGV.
// Callers (dispatcher) must assign headerId / orderUpdateId / actionId before publish.
type AGVHandle struct {
	client       *Client
	manufacturer string
	serial       string
}

// PublishOrder publishes an order. Caller owns HeaderId and OrderUpdateId.
func (a *AGVHandle) PublishOrder(ctx context.Context, o *order.Order) error {
	if err := a.client.requireStarted(); err != nil {
		return err
	}
	if o == nil {
		return nil
	}
	a.client.builder.PrepareOrder(o, a.manufacturer, a.serial)

	payload, err := json.Marshal(o)
	if err != nil {
		return err
	}
	t := a.client.topics.Build(a.manufacturer, a.serial, topic.ChannelOrder)
	return a.client.transport.Publish(ctx, t, payload, PublishOptions{QoS: a.client.cfg.qos()})
}

// PublishInstantActions publishes instantActions. Caller owns HeaderId and actionIds.
func (a *AGVHandle) PublishInstantActions(ctx context.Context, ia *instant_actions.InstantActions) error {
	if err := a.client.requireStarted(); err != nil {
		return err
	}
	if ia == nil {
		return nil
	}
	a.client.builder.PrepareInstantActions(ia, a.manufacturer, a.serial)

	payload, err := json.Marshal(ia)
	if err != nil {
		return err
	}
	t := a.client.topics.Build(a.manufacturer, a.serial, topic.ChannelInstantActions)
	return a.client.transport.Publish(ctx, t, payload, PublishOptions{QoS: a.client.cfg.qos()})
}

// CancelOrder publishes a standard cancelOrder instantAction.
// actionID and headerID must be supplied by the caller (orchestration layer).
func (a *AGVHandle) CancelOrder(ctx context.Context, headerID uint32, actionID string) error {
	if actionID == "" {
		return gerrors.NewInvalidConfigWithArgs("actionId is required")
	}
	ia := &instant_actions.InstantActions{
		Actions: []instant_actions.InstantAction{
			{
				ActionType:   "cancelOrder",
				ActionId:     actionID,
				BlockingType: vda5050.Hard,
			},
		},
	}
	ia.HeaderId = headerID
	return a.PublishInstantActions(ctx, ia)
}
