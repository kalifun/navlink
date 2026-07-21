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
	AGV        Identity
	Topic      string
	Channel    topic.Channel
	Raw        []byte
	ReceivedAt time.Time
	Header     HeaderSummary
	Meta       Meta
	RobotID    string // filled when Config.IdentityMapper is set
}
