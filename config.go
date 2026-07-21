package navlink

import (
	"strings"
	"time"

	"github.com/kalifun/navlink/gerrors"
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

	// Manufacturer / SerialNumber optionally pin subscriptions to one AGV.
	// Empty means fleet-level `+` wildcards for channels with handlers.
	Manufacturer string
	SerialNumber string

	QoS           byte
	KeepAlive     time.Duration
	CleanSession  bool
	AutoReconnect bool

	// StrictIdentity validates payload manufacturer/serial against the topic (default true).
	StrictIdentity *bool

	// Transport injects a custom transport (tests / shared connection).
	// When nil, an MQTT transport is created from Broker settings.
	Transport Transport

	// IdentityMapper optionally fills Envelope.RobotID.
	IdentityMapper IdentityMapper

	// OnDecodeError is called when decode or identity checks fail.
	OnDecodeError DecodeErrorHandler
}

func (c Config) strictIdentity() bool {
	if c.StrictIdentity == nil {
		return true
	}
	return *c.StrictIdentity
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
