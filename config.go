package navlink

import (
	"strings"
	"time"

	"github.com/kalifun/navlink/extend"
	"github.com/kalifun/navlink/gerrors"
	"github.com/kalifun/navlink/session"
)

// Config configures a navlink Client.
type Config struct {
	// Broker is the MQTT broker URL, e.g. tcp://localhost:1883.
	// Required unless Transport is provided.
	Broker string

	// ClientID is the MQTT client id. Required unless Transport is provided.
	ClientID string

	Username string
	Password string

	// Interface is the VDA topic prefix (e.g. uagv / vda5050). Required.
	Interface string
	// Version is the VDA topic version segment (e.g. v2 / v2.0.0). Required.
	Version string

	// HeaderVersion is written into outbound ProtocolHeader.Version for Order and
	// InstantActions. Empty means Version is used (same Client policy for both).
	HeaderVersion string

	// Manufacturer / SerialNumber optionally pin subscriptions to one AGV.
	// Empty means fleet-level `+` wildcards for channels with handlers.
	Manufacturer string
	SerialNumber string

	QoS           byte
	KeepAlive     time.Duration
	CleanSession  bool
	AutoReconnect bool

	// RestoreSubscriptionsOnReconnect re-subscribes VDA topics after MQTT reconnect.
	// Default true. Applies when the transport implements ReconnectAware (built-in MQTT,
	// testkit FakeBroker). FleetSession uses Restore; other subscriptions are recreated.
	RestoreSubscriptionsOnReconnect *bool

	// StrictIdentity validates payload manufacturer/serial against the topic (default true).
	StrictIdentity *bool

	// Transport injects a custom transport (tests / shared connection).
	// When nil, an MQTT transport is created from Broker settings.
	Transport Transport

	// Fleet enables FleetSession when non-nil.
	// Pass &session.DefaultOptions() or a customized session.Options.
	// An all-zero Options value is treated as DefaultOptions().
	Fleet *session.Options

	// Extensions fills Envelope.Meta from vendor fields (optional).
	Extensions *extend.Registry

	// Bus is an optional EventBus (same as Client.UseEventBus).
	Bus EventBus

	// IdentityMapper optionally fills Envelope.RobotID.
	IdentityMapper IdentityMapper

	// OutboundValidation configures light pre-publish checks. Nil = enabled defaults
	// (reject headerId 0, empty orderId, orderUpdateId 0, empty actionId, identity mismatch).
	OutboundValidation *OutboundValidation

	// InboundPolicy optionally classifies inbound headerId (Accept/Stale/Duplicate).
	// Nil = accept-all. Classification is annotated on Envelope; messages are not dropped.
	InboundPolicy InboundPolicy

	// OnDecodeError is called when decode or identity checks fail.
	OnDecodeError DecodeErrorHandler
}

func (c Config) headerVersion() string {
	if strings.TrimSpace(c.HeaderVersion) != "" {
		return c.HeaderVersion
	}
	return c.Version
}

func (c Config) strictIdentity() bool {
	if c.StrictIdentity == nil {
		return true
	}
	return *c.StrictIdentity
}

func (c Config) restoreOnReconnect() bool {
	if c.RestoreSubscriptionsOnReconnect == nil {
		return true
	}
	return *c.RestoreSubscriptionsOnReconnect
}

func (c Config) validate() error {
	if strings.TrimSpace(c.Interface) == "" {
		return gerrors.NewInvalidConfigWithArgs("Interface is required")
	}
	if strings.TrimSpace(c.Version) == "" {
		return gerrors.NewInvalidConfigWithArgs("Version is required")
	}
	if c.Transport == nil {
		if strings.TrimSpace(c.Broker) == "" {
			return gerrors.NewInvalidConfigWithArgs("Broker is required when Transport is nil")
		}
		if strings.TrimSpace(c.ClientID) == "" {
			return gerrors.NewInvalidConfigWithArgs("ClientID is required when Transport is nil")
		}
	}
	return nil
}

func (c Config) qos() byte {
	if c.QoS == 0 {
		return 0
	}
	return c.QoS
}
