package navlink

import (
	"context"
	"time"
)

type inboundReceivedAtKey struct{}

// InboundSlowCause says which inbound phase exceeded Config.SlowInbound.
type InboundSlowCause int

const (
	// InboundSlowQueue: time in the MQTT inbound queue before the worker ran.
	InboundSlowQueue InboundSlowCause = iota
	// InboundSlowHandler: time spent in decode + On* / OnTopic.
	InboundSlowHandler
)

func (c InboundSlowCause) String() string {
	switch c {
	case InboundSlowQueue:
		return "queue"
	case InboundSlowHandler:
		return "handler"
	default:
		return "unknown"
	}
}

func contextWithInboundReceivedAt(ctx context.Context, receivedAt time.Time) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if receivedAt.IsZero() {
		return ctx
	}
	return context.WithValue(ctx, inboundReceivedAtKey{}, receivedAt)
}

func envelopeTimes(ctx context.Context) (receivedAt, dispatchedAt time.Time) {
	dispatchedAt = time.Now().UTC()
	if ctx != nil {
		if t, ok := ctx.Value(inboundReceivedAtKey{}).(time.Time); ok && !t.IsZero() {
			return t, dispatchedAt
		}
	}
	return dispatchedAt, dispatchedAt
}

func (c *Client) noteInbound(env Envelope) {
	if c == nil || c.cfg.OnSlowInbound == nil || c.cfg.SlowInbound <= 0 {
		return
	}
	if w := env.QueueWait(); w >= c.cfg.SlowInbound {
		c.cfg.OnSlowInbound(env, InboundSlowQueue, w)
	}
	if !env.DispatchedAt.IsZero() {
		if d := time.Since(env.DispatchedAt); d >= c.cfg.SlowInbound {
			c.cfg.OnSlowInbound(env, InboundSlowHandler, d)
		}
	}
}
