package types

import "time"

// DomainHeader contains the VDA5050 protocol header and routing information.
type DomainHeader struct {
	HeaderId     uint32
	Timestamp    string
	Version      string
	Manufacturer string
	SerialNumber string
}

// DomainMessage is the internal representation of a message.
type DomainMessage struct {
	Header  DomainHeader
	Payload []byte
	Type    DomainMessageType
}

// DomainMessageType defines the type of the domain message.
type DomainMessageType string

const (
	DomainMessageTypeOrder         DomainMessageType = "order"
	DomainMessageTypeInstantAction DomainMessageType = "instantActions"
	DomainMessageTypeState         DomainMessageType = "state"
	DomainMessageTypeVisualization DomainMessageType = "visualization"
	DomainMessageTypeConnection    DomainMessageType = "connection"
	DomainMessageTypeFactsheet     DomainMessageType = "factsheet"
	DomainMessageTypeUnknown       DomainMessageType = "unknown"
	DomainMessageTypeWildcard      DomainMessageType = "*"
)

// TransportMessage is the message format used by the transport layer.
type TransportMessage struct {
	Topic   string
	Payload []byte
	Meta    map[string]interface{}
}

// TransportPublish is the message format for publishing messages from the domain.
type TransportPublish struct {
	Topic   string
	Payload []byte
	Opts    interface{} // e.g., PublishOptions
}

// PublishOptions contains options for publishing a message.
type PublishOptions struct {
	QoS     byte
	Retain  bool
	Timeout time.Duration
}
