package navlink

import (
	"crypto/tls"
	"strings"
	"time"

	"github.com/kalifun/navlink/extend"
	"github.com/kalifun/navlink/internal/gerrors"
)

// FleetOptions configures per-AGV subscriptions when Config.Fleet is set.
type FleetOptions struct {
	// SubscribeState subscribes per-AGV state topics when tracked (default true).
	SubscribeState bool
	// SubscribeVisualization subscribes per-AGV visualization when tracked.
	SubscribeVisualization bool
	// AutoTrackFromConnection tracks on ONLINE and untracks on OFFLINE/CONNECTIONBROKEN.
	AutoTrackFromConnection bool
}

// DefaultFleetOptions returns the recommended fleet defaults.
func DefaultFleetOptions() FleetOptions {
	return FleetOptions{
		SubscribeState:          true,
		SubscribeVisualization:  false,
		AutoTrackFromConnection: true,
	}
}

// DefaultOrderQoS is used for order / instantActions (and non-viz subscribe)
// when Config.QoS is nil.
const DefaultOrderQoS byte = 1

// QoSOf returns a pointer for Config.QoS. Passing 0 means real MQTT QoS 0.
func QoSOf(q byte) *byte {
	v := q
	return &v
}

// LastWill is an MQTT last-will message (optional).
type LastWill struct {
	Topic   string
	Payload []byte
	QoS     byte
	Retain  bool
}

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

	// QoS is the MQTT QoS for publishes and subscriptions.
	// nil = library defaults (order/instantActions publish 1; visualization subscribe 0).
	// A pointer to 0 is a real QoS 0 (not "unset").
	QoS *byte

	KeepAlive      time.Duration
	ConnectTimeout time.Duration
	CleanSession   bool
	AutoReconnect  bool
	TLS            *tls.Config
	Will           *LastWill

	// InboundQueueSize is the paho-callback → worker queue length (default 256).
	InboundQueueSize int
	// OnInboundDrop is called when the inbound queue is full (viz may be dropped;
	// state/connection back-pressure after this hook).
	OnInboundDrop func(topic string)

	// RestoreSubscriptionsOnReconnect re-subscribes VDA topics after MQTT reconnect.
	// Default true. Applies when the transport implements ReconnectAware (built-in MQTT,
	// testkit FakeBroker). FleetSession uses Restore; other subscriptions are recreated.
	RestoreSubscriptionsOnReconnect *bool

	// StrictIdentity validates payload manufacturer/serial against the topic (default true).
	StrictIdentity *bool

	// Transport injects a custom transport (tests / shared connection).
	// When nil, an MQTT transport is created from Broker settings.
	Transport Transport

	// Fleet enables fleet tracking when non-nil.
	// Pass &DefaultFleetOptions() or a customized FleetOptions.
	// An all-zero FleetOptions is treated as DefaultFleetOptions().
	Fleet *FleetOptions

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
	// OnHandlerError is called when a typed/raw handler returns an error.
	OnHandlerError  HandlerErrorHandler
	OnTransportUp   func()
	OnTransportDown func(error)
	// OnSubscriptionsRestored is called after reconnect restore (nil err = success).
	OnSubscriptionsRestored func(error)
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
	if c.Will != nil {
		if strings.TrimSpace(c.Will.Topic) == "" || c.Will.QoS > 2 {
			return gerrors.WillMessageError
		}
	}
	return nil
}

func (c Config) publishQoS() byte {
	if c.QoS == nil {
		return DefaultOrderQoS
	}
	return *c.QoS
}

func (c Config) subscribeQoSExplicit() (qos byte, explicit bool) {
	if c.QoS == nil {
		return DefaultOrderQoS, false
	}
	return *c.QoS, true
}
