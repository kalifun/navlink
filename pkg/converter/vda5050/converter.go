package vda5050

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"

    "github.com/kalifun/navlink/pkg/types"
    "github.com/kalifun/vda5050-types-go"
    "github.com/kalifun/vda5050-types-go/connection"
    "github.com/kalifun/vda5050-types-go/factsheet"
    "github.com/kalifun/vda5050-types-go/instant_actions"
    "github.com/kalifun/vda5050-types-go/order"
    "github.com/kalifun/vda5050-types-go/state"
    "github.com/kalifun/vda5050-types-go/visualization"
)

// VDA5050Converter implements the core.Converter interface for VDA5050 messages.
type VDA5050Converter struct{}

func (c *VDA5050Converter) Start(ctx context.Context) error {
    // No-op for now; placeholder for future resource init
    return nil
}

func (c *VDA5050Converter) Stop(ctx context.Context) error {
    // No-op for now; placeholder for future cleanup
    return nil
}

// ToDomain converts a transport message to a domain message
func (c *VDA5050Converter) ToDomain(ctx context.Context, tmsg *types.TransportMessage) (*types.DomainMessage, error) {
	parts := strings.Split(tmsg.Topic, "/")
	if len(parts) < 5 {
		return nil, fmt.Errorf("invalid VDA5050 topic format: %s, expected at least 5 parts", tmsg.Topic)
	}
	header, err := extractHeader(tmsg.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to extract VDA5050 header: %w", err)
	}
	if header == nil {
		return nil, fmt.Errorf("extracted header is nil")
	}

	domainHeader := types.DomainHeader{
		HeaderId:     header.HeaderId,
		Timestamp:    header.Timestamp,
		Version:      parts[1],
		Manufacturer: parts[2],
		SerialNumber: parts[3],
	}

	topicType := types.DomainMessageType(parts[len(parts)-1])

	// Validate and unmarshal payload based on topic type to ensure it's a valid message
	switch topicType {
	case types.DomainMessageTypeOrder:
		var msg order.Order
		if err := json.Unmarshal(tmsg.Payload, &msg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal order message: %w", err)
		}
	case types.DomainMessageTypeInstantAction:
		var msg instant_actions.InstantActions
		if err := json.Unmarshal(tmsg.Payload, &msg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal instant action message: %w", err)
		}
	case types.DomainMessageTypeState:
		var msg state.State
		if err := json.Unmarshal(tmsg.Payload, &msg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal state message: %w", err)
		}
	case types.DomainMessageTypeVisualization:
		var msg visualization.Visualization
		if err := json.Unmarshal(tmsg.Payload, &msg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal visualization message: %w", err)
		}
	case types.DomainMessageTypeConnection:
		var msg connection.Connection
		if err := json.Unmarshal(tmsg.Payload, &msg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal connection message: %w", err)
		}
	case types.DomainMessageTypeFactsheet:
		var msg factsheet.Factsheet
		if err := json.Unmarshal(tmsg.Payload, &msg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal factsheet message: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown topic type: %s", topicType)
	}

	dmsg := &types.DomainMessage{
		Header:  domainHeader,
		Payload: tmsg.Payload, // Keep the raw payload
		Type:    topicType,
	}

	return dmsg, nil
}

// FromDomain converts a domain message to a transport publish message
func (c *VDA5050Converter) FromDomain(ctx context.Context, dmsg *types.DomainMessage) (*types.TransportPublish, error) {
    // Re-construct the topic from the domain message header
    topic := fmt.Sprintf("uagv/%s/%s/%s/%s", dmsg.Header.Version, dmsg.Header.Manufacturer, dmsg.Header.SerialNumber, dmsg.Type)

    return &types.TransportPublish{
        Topic:   topic,
        Payload: dmsg.Payload,
    }, nil
}

// GetSupportedTypes returns the message types this converter supports
func (c *VDA5050Converter) GetSupportedTypes() []string {
    return []string{
        string(types.DomainMessageTypeOrder),
        string(types.DomainMessageTypeInstantAction),
        string(types.DomainMessageTypeState),
        string(types.DomainMessageTypeVisualization),
        string(types.DomainMessageTypeConnection),
        string(types.DomainMessageTypeFactsheet),
    }
}

// Validate validates a domain message
func (c *VDA5050Converter) Validate(ctx context.Context, dmsg *types.DomainMessage) error {
    if dmsg == nil {
        return fmt.Errorf("domain message cannot be nil")
    }
    if dmsg.Payload == nil {
        return fmt.Errorf("domain message payload cannot be nil")
    }

    switch dmsg.Type {
    case types.DomainMessageTypeOrder:
        var msg order.Order
        if err := json.Unmarshal(dmsg.Payload, &msg); err != nil {
            return fmt.Errorf("invalid order payload: %w", err)
        }
    case types.DomainMessageTypeInstantAction:
        var msg instant_actions.InstantActions
        if err := json.Unmarshal(dmsg.Payload, &msg); err != nil {
            return fmt.Errorf("invalid instantActions payload: %w", err)
        }
    case types.DomainMessageTypeState:
        var msg state.State
        if err := json.Unmarshal(dmsg.Payload, &msg); err != nil {
            return fmt.Errorf("invalid state payload: %w", err)
        }
    case types.DomainMessageTypeVisualization:
        var msg visualization.Visualization
        if err := json.Unmarshal(dmsg.Payload, &msg); err != nil {
            return fmt.Errorf("invalid visualization payload: %w", err)
        }
    case types.DomainMessageTypeConnection:
        var msg connection.Connection
        if err := json.Unmarshal(dmsg.Payload, &msg); err != nil {
            return fmt.Errorf("invalid connection payload: %w", err)
        }
    case types.DomainMessageTypeFactsheet:
        var msg factsheet.Factsheet
        if err := json.Unmarshal(dmsg.Payload, &msg); err != nil {
            return fmt.Errorf("invalid factsheet payload: %w", err)
        }
    default:
        return fmt.Errorf("unsupported domain message type: %s", dmsg.Type)
    }
    return nil
}

func extractHeader(payload []byte) (*vda5050.ProtocolHeader, error) {
	var hearder vda5050.ProtocolHeader
	if err := json.Unmarshal(payload, &hearder); err != nil {
		return nil, err
	}
	return &hearder, nil
}
