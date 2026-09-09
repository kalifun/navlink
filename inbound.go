package navlink

import (
	"context"
	"encoding/json"

	vda5050 "github.com/kalifun/vda5050-types-go"
	"github.com/kalifun/vda5050-types-go/connection"
	"github.com/kalifun/vda5050-types-go/factsheet"
	"github.com/kalifun/vda5050-types-go/state"
	"github.com/kalifun/vda5050-types-go/visualization"

	"github.com/kalifun/navlink/internal/gerrors"
	"github.com/kalifun/navlink/internal/session"
	"github.com/kalifun/navlink/topic"
)

func (c *Client) onRawMessage(ctx context.Context, rawTopic string, payload []byte) error {
	receivedAt, dispatchedAt := envelopeTimes(ctx)
	parsed, err := c.topics.Parse(rawTopic)
	if err != nil {
		env := Envelope{Topic: rawTopic, Raw: payload, ReceivedAt: receivedAt, DispatchedAt: dispatchedAt}
		c.reportDecode(ctx, env, err)
		c.noteInbound(env)
		return nil
	}

	env := Envelope{
		AGV: Identity{
			Manufacturer: parsed.Manufacturer,
			SerialNumber: parsed.SerialNumber,
		},
		Topic:        rawTopic,
		Channel:      parsed.Channel,
		Raw:          payload,
		ReceivedAt:   receivedAt,
		DispatchedAt: dispatchedAt,
		Meta:         Meta{},
	}
	defer func() { c.noteInbound(env) }()
	if c.cfg.IdentityMapper != nil {
		env.RobotID = c.cfg.IdentityMapper(parsed.Manufacturer, parsed.SerialNumber)
	}
	if c.extensions != nil {
		meta, err := c.extensions.Apply(string(parsed.Channel), payload)
		if err != nil {
			c.reportDecode(ctx, env, gerrors.NewDecodeFailedWithArgs(err.Error()))
			return nil
		}
		env.Meta = Meta(meta)
	}

	header, herr := extractHeader(payload)
	if herr == nil && header != nil {
		env.Header = HeaderSummary{
			HeaderID:     header.HeaderId,
			Timestamp:    header.Timestamp,
			Version:      header.Version,
			Manufacturer: header.Manufacturer,
			SerialNumber: header.SerialNumber,
		}
		if err := c.checkIdentity(env); err != nil {
			c.reportDecode(ctx, env, err)
			return nil
		}
		c.applyInboundPolicy(&env)
	}

	switch parsed.Channel {
	case topic.ChannelState:
		var msg state.State
		if err := json.Unmarshal(payload, &msg); err != nil {
			c.reportDecode(ctx, env, gerrors.NewDecodeFailedWithArgs(err.Error()))
			return nil
		}
		return c.finishHandler(env, c.invokeState(ctx, env, &msg))
	case topic.ChannelConnection:
		var msg connection.Connection
		if err := json.Unmarshal(payload, &msg); err != nil {
			c.reportDecode(ctx, env, gerrors.NewDecodeFailedWithArgs(err.Error()))
			return nil
		}
		return c.finishHandler(env, c.invokeConnection(ctx, env, &msg))
	case topic.ChannelVisualization:
		var msg visualization.Visualization
		if err := json.Unmarshal(payload, &msg); err != nil {
			c.reportDecode(ctx, env, gerrors.NewDecodeFailedWithArgs(err.Error()))
			return nil
		}
		return c.finishHandler(env, c.invokeVisualization(ctx, env, &msg))
	case topic.ChannelFactsheet:
		var msg factsheet.Factsheet
		if err := json.Unmarshal(payload, &msg); err != nil {
			c.reportDecode(ctx, env, gerrors.NewDecodeFailedWithArgs(err.Error()))
			return nil
		}
		return c.finishHandler(env, c.invokeFactsheet(ctx, env, &msg))
	default:
		return nil
	}
}

func (c *Client) finishHandler(env Envelope, err error) error {
	if err != nil {
		c.reportHandlerError(env, err)
	}
	return err
}

func (c *Client) dispatchTopic(ctx context.Context, rawTopic string, payload []byte, h TopicHandler) error {
	receivedAt, dispatchedAt := envelopeTimes(ctx)
	env := Envelope{
		Topic:        rawTopic,
		Raw:          payload,
		ReceivedAt:   receivedAt,
		DispatchedAt: dispatchedAt,
		Meta:         Meta{},
	}
	if parsed, err := c.topics.Parse(rawTopic); err == nil {
		env.AGV = Identity{Manufacturer: parsed.Manufacturer, SerialNumber: parsed.SerialNumber}
		env.Channel = parsed.Channel
		if c.cfg.IdentityMapper != nil {
			env.RobotID = c.cfg.IdentityMapper(parsed.Manufacturer, parsed.SerialNumber)
		}
	}
	err := c.finishHandler(env, h(ctx, env))
	c.noteInbound(env)
	return err
}

func (c *Client) checkIdentity(env Envelope) error {
	if !c.cfg.strictIdentity() {
		return nil
	}
	if env.Header.Manufacturer == "" && env.Header.SerialNumber == "" {
		return nil
	}
	if env.Header.Manufacturer != env.AGV.Manufacturer || env.Header.SerialNumber != env.AGV.SerialNumber {
		return gerrors.IdentityMismatch
	}
	return nil
}

func (c *Client) applyInboundPolicy(env *Envelope) {
	if c.cfg.InboundPolicy == nil || env == nil {
		return
	}
	d := c.cfg.InboundPolicy.Classify(env.AGV, env.Channel, env.Header.HeaderID)
	if d == "" {
		d = InboundAccept
	}
	env.InboundDisposition = d
	if env.Meta == nil {
		env.Meta = Meta{}
	}
	env.Meta[MetaInboundDisposition] = string(d)
}

func (c *Client) reportDecode(ctx context.Context, env Envelope, err error) {
	_ = c.publishEvent(ctx, EventDecodeFailed, DecodeFailedEvent{Envelope: env, Err: err})
	if c.cfg.OnDecodeError != nil {
		c.cfg.OnDecodeError(env, err)
	}
}

func (c *Client) invokeState(ctx context.Context, env Envelope, msg *state.State) error {
	c.mu.RLock()
	bus := c.bus
	handlers := append([]StateHandler(nil), c.stateHandlers...)
	c.mu.RUnlock()
	if bus != nil {
		return bus.Publish(ctx, EventStateReceived, StateEvent{Envelope: env, State: msg})
	}
	for _, h := range handlers {
		if err := h(ctx, env, msg); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) invokeConnection(ctx context.Context, env Envelope, msg *connection.Connection) error {
	// Never Track/Subscribe here: that blocks the inbound path. Opt-in AutoTrack
	// is queued onto a dedicated worker (see scheduleAutoTrack).
	c.scheduleAutoTrack(env, msg.ConnectionState)
	c.mu.RLock()
	bus := c.bus
	handlers := append([]ConnectionHandler(nil), c.connHandlers...)
	c.mu.RUnlock()
	if bus != nil {
		return bus.Publish(ctx, EventConnectionChanged, ConnectionEvent{Envelope: env, Connection: msg})
	}
	for _, h := range handlers {
		if err := h(ctx, env, msg); err != nil {
			return err
		}
	}
	return nil
}

func sessionAGV(id Identity) session.AGV {
	return session.AGV{Manufacturer: id.Manufacturer, SerialNumber: id.SerialNumber}
}

func (c *Client) invokeVisualization(ctx context.Context, env Envelope, msg *visualization.Visualization) error {
	c.mu.RLock()
	bus := c.bus
	handlers := append([]VisualizationHandler(nil), c.vizHandlers...)
	c.mu.RUnlock()
	if bus != nil {
		return bus.Publish(ctx, EventVisualizationReceived, VisualizationEvent{Envelope: env, Visualization: msg})
	}
	for _, h := range handlers {
		if err := h(ctx, env, msg); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) invokeFactsheet(ctx context.Context, env Envelope, msg *factsheet.Factsheet) error {
	c.mu.RLock()
	bus := c.bus
	handlers := append([]FactsheetHandler(nil), c.fsHandlers...)
	c.mu.RUnlock()
	if bus != nil {
		return bus.Publish(ctx, EventFactsheetReceived, FactsheetEvent{Envelope: env, Factsheet: msg})
	}
	for _, h := range handlers {
		if err := h(ctx, env, msg); err != nil {
			return err
		}
	}
	return nil
}

func extractHeader(payload []byte) (*vda5050.ProtocolHeader, error) {
	var h vda5050.ProtocolHeader
	if err := json.Unmarshal(payload, &h); err != nil {
		return nil, err
	}
	return &h, nil
}
