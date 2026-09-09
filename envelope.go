package navlink

import (
	"time"

	"github.com/kalifun/navlink/topic"
)

// Meta holds vendor extension fields produced by ExtensionRegistry (P1+).
type Meta map[string]any

// HeaderSummary is a small decode of common VDA5050 header fields.
type HeaderSummary struct {
	HeaderID     uint32
	Timestamp    string
	Version      string
	Manufacturer string
	SerialNumber string
}

// Envelope is the inbound message shell around a typed VDA5050 payload.
type Envelope struct {
	AGV     Identity
	Topic   string
	Channel topic.Channel
	Raw     []byte
	// ReceivedAt is when the MQTT callback enqueued the payload (UTC).
	// Transports without a queue (FakeBroker) set it to the same instant as DispatchedAt.
	ReceivedAt time.Time
	// DispatchedAt is when the inbound worker started handling the message (UTC).
	DispatchedAt time.Time
	Header       HeaderSummary
	Meta         Meta
	RobotID      string // filled when Config.IdentityMapper is set

	// InboundDisposition is set when Config.InboundPolicy is configured.
	// Empty means unclassified (default accept-all).
	InboundDisposition InboundDisposition
}

// QueueWait is DispatchedAt − ReceivedAt: time spent in the inbound queue.
// Zero if either timestamp is unset, or if the result would be negative.
func (e Envelope) QueueWait() time.Duration {
	if e.ReceivedAt.IsZero() || e.DispatchedAt.IsZero() {
		return 0
	}
	d := e.DispatchedAt.Sub(e.ReceivedAt)
	if d < 0 {
		return 0
	}
	return d
}
