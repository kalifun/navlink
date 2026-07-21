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
// On success, PublishResult describes the bytes/topic actually sent; on failure
// result may be zero or partially filled (topic/payload prepared before transport).
//
// err == nil (PublishAccepted) means the MQTT QoS handshake succeeded (broker
// accepted the publish). It does not mean the vehicle accepted the order —
// that is still observed via inbound state. Platforms may RecordSuccessfulPublish
// only on accepted publish; navlink never records for them.
func (a *AGVHandle) PublishOrder(ctx context.Context, o *order.Order) (PublishResult, error) {
	var res PublishResult
	if err := a.client.requireStarted(); err != nil {
		return res, err
	}
	if o == nil {
		return res, gerrors.NewInvalidConfigWithArgs("order is nil")
	}
	a.client.builder.PrepareOrder(o, a.manufacturer, a.serial)

	payload, err := json.Marshal(o)
	if err != nil {
		return res, err
	}
	qos := a.client.cfg.qos()
	if err := validatePublishQoS(qos); err != nil {
		return res, err
	}
	ch := topic.ChannelOrder
	t := a.client.topics.Build(a.manufacturer, a.serial, ch)
	res = PublishResult{
		Topic:         t,
		Channel:       ch,
		Manufacturer:  a.manufacturer,
		SerialNumber:  a.serial,
		QoS:           qos,
		Payload:       payload,
		HeaderID:      o.HeaderId,
		OrderID:       o.OrderId,
		OrderUpdateID: o.OrderUpdateId,
	}
	if err := a.client.transport.Publish(ctx, t, payload, PublishOptions{QoS: qos}); err != nil {
		return res, err
	}
	return res, nil
}

// PublishInstantActions publishes instantActions. Caller owns HeaderId and actionIds.
// See PublishOrder for success/failure semantics and PublishResult usage.
func (a *AGVHandle) PublishInstantActions(ctx context.Context, ia *instant_actions.InstantActions) (PublishResult, error) {
	var res PublishResult
	if err := a.client.requireStarted(); err != nil {
		return res, err
	}
	if ia == nil {
		return res, gerrors.NewInvalidConfigWithArgs("instantActions is nil")
	}
	a.client.builder.PrepareInstantActions(ia, a.manufacturer, a.serial)

	payload, err := json.Marshal(ia)
	if err != nil {
		return res, err
	}
	qos := a.client.cfg.qos()
	if err := validatePublishQoS(qos); err != nil {
		return res, err
	}
	ch := topic.ChannelInstantActions
	t := a.client.topics.Build(a.manufacturer, a.serial, ch)
	actionIDs := make([]string, 0, len(ia.Actions))
	for _, act := range ia.Actions {
		actionIDs = append(actionIDs, act.ActionId)
	}
	res = PublishResult{
		Topic:        t,
		Channel:      ch,
		Manufacturer: a.manufacturer,
		SerialNumber: a.serial,
		QoS:          qos,
		Payload:      payload,
		HeaderID:     ia.HeaderId,
		ActionIDs:    actionIDs,
	}
	if err := a.client.transport.Publish(ctx, t, payload, PublishOptions{QoS: qos}); err != nil {
		return res, err
	}
	return res, nil
}

// CancelOrder publishes a standard cancelOrder instantAction.
// actionID and headerID must be supplied by the caller (orchestration layer).
func (a *AGVHandle) CancelOrder(ctx context.Context, headerID uint32, actionID string) (PublishResult, error) {
	if actionID == "" {
		return PublishResult{}, gerrors.NewInvalidConfigWithArgs("actionId is required")
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
