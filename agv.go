package navlink

import (
	"context"
	"encoding/json"
	"fmt"
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

// PublishOrder publishes an order after EnvelopeBuilder fills header / IDs.
func (a *AGVHandle) PublishOrder(ctx context.Context, o *order.Order) error {
	if err := a.client.requireStarted(); err != nil {
		return err
	}
	if o == nil {
		return nil
	}

	commitID, err := a.client.builder.PrepareOrder(o, a.manufacturer, a.serial)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(o)
	if err != nil {
		return err
	}
	t := a.client.topics.Build(a.manufacturer, a.serial, topic.ChannelOrder)
	if err := a.client.transport.Publish(ctx, t, payload, PublishOptions{QoS: a.client.cfg.qos()}); err != nil {
		return err
	}
	return a.client.builder.CommitOrderUpdate(o.OrderId, commitID)
}

// PublishInstantActions publishes instantActions after EnvelopeBuilder fills header fields.
func (a *AGVHandle) PublishInstantActions(ctx context.Context, ia *instant_actions.InstantActions) error {
	if err := a.client.requireStarted(); err != nil {
		return err
	}
	if ia == nil {
		return nil
	}
	if err := a.client.builder.PrepareInstantActions(ia, a.manufacturer, a.serial); err != nil {
		return err
	}
	payload, err := json.Marshal(ia)
	if err != nil {
		return err
	}
	t := a.client.topics.Build(a.manufacturer, a.serial, topic.ChannelInstantActions)
	return a.client.transport.Publish(ctx, t, payload, PublishOptions{QoS: a.client.cfg.qos()})
}

// NextActionID allocates a monotonic actionId for this AGV when ActionIDs is configured.
func (a *AGVHandle) NextActionID() (string, error) {
	return a.client.builder.NextActionID(a.manufacturer, a.serial)
}

// CancelOrder publishes a standard cancelOrder instantAction.
// actionId comes from ActionIDs when configured; otherwise a timestamp-based id is used.
func (a *AGVHandle) CancelOrder(ctx context.Context) error {
	actionID, err := a.NextActionID()
	if err != nil {
		actionID = fmt.Sprintf("cancel-%d", time.Now().UTC().UnixNano())
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
	return a.PublishInstantActions(ctx, ia)
}

// SyncOrderUpdateFromVehicle observes a vehicle-reported orderUpdateId without rewinding.
func (a *AGVHandle) SyncOrderUpdateFromVehicle(orderID string, observed uint32) {
	if a.client.cfg.OrderUpdateIDs == nil {
		return
	}
	a.client.cfg.OrderUpdateIDs.SyncFromVehicle(orderID, observed)
}
