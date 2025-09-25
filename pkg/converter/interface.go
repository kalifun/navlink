package converter

import (
    "context"

    "github.com/kalifun/navlink/pkg/types"
)

// Converter handles protocol conversion between transport messages and domain messages
type Converter interface {
    // ToDomain converts a transport message to a domain message
    ToDomain(ctx context.Context, tmsg *types.TransportMessage) (*types.DomainMessage, error)

    // FromDomain converts a domain message to a transport publish message
    FromDomain(ctx context.Context, dmsg *types.DomainMessage) (*types.TransportPublish, error)

    // GetSupportedTypes returns the message types this converter supports
    GetSupportedTypes() []string

    // Validate validates a domain message
    Validate(ctx context.Context, dmsg *types.DomainMessage) error
}

// ConverterFactory creates converter instances
type ConverterFactory func(config map[string]interface{}) (Converter, error)
