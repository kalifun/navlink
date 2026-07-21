package topic

import (
	"fmt"
	"strings"
)

// Channel is the last segment of a VDA5050 MQTT topic.
type Channel string

const (
	ChannelOrder          Channel = "order"
	ChannelInstantActions Channel = "instantActions"
	ChannelState          Channel = "state"
	ChannelVisualization  Channel = "visualization"
	ChannelConnection     Channel = "connection"
	ChannelFactsheet      Channel = "factsheet"
)

// Resolver is the single source of truth for VDA5050 topic build/parse.
type Resolver struct {
	Interface string
	Version   string
}

// ParsedTopic is the identity extracted from a topic string.
type ParsedTopic struct {
	Interface    string
	Version      string
	Manufacturer string
	SerialNumber string
	Channel      Channel
}

// Build constructs `{interface}/{version}/{manufacturer}/{serialNumber}/{channel}`.
func (r Resolver) Build(manufacturer, serial string, ch Channel) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s", r.Interface, r.Version, manufacturer, serial, ch)
}

// Wildcard builds a fleet-level topic with `+` for manufacturer and serial.
func (r Resolver) Wildcard(ch Channel) string {
	return r.Build("+", "+", ch)
}

// Parse splits a VDA5050 topic into identity parts.
func (r Resolver) Parse(topic string) (ParsedTopic, error) {
	parts := strings.Split(topic, "/")
	if len(parts) != 5 {
		return ParsedTopic{}, fmt.Errorf("invalid VDA5050 topic %q: want 5 segments", topic)
	}
	ch := Channel(parts[4])
	if !ValidChannel(ch) {
		return ParsedTopic{}, fmt.Errorf("unknown VDA5050 channel %q in topic %q", parts[4], topic)
	}
	return ParsedTopic{
		Interface:    parts[0],
		Version:      parts[1],
		Manufacturer: parts[2],
		SerialNumber: parts[3],
		Channel:      ch,
	}, nil
}

// ValidChannel reports whether ch is a known VDA5050 channel.
func ValidChannel(ch Channel) bool {
	switch ch {
	case ChannelOrder, ChannelInstantActions, ChannelState,
		ChannelVisualization, ChannelConnection, ChannelFactsheet:
		return true
	default:
		return false
	}
}
