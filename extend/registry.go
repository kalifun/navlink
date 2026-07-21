package extend

import "fmt"

// Meta is a bag of extension fields attached to an inbound Envelope.
// Key naming and semantics are defined by the consumer, not by navlink.
type Meta map[string]any

// Extractor pulls consumer-defined fields from a raw payload for a VDA channel.
// Returning a nil/empty Meta is not an error.
//
// Vendor / OEM extractors belong in the consuming project (dispatcher, sim,
// tool). navlink only provides the registration hook.
type Extractor func(channel string, raw []byte) (Meta, error)

// Registry runs registered extractors and merges Meta maps.
type Registry struct {
	extractors []Extractor
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register appends an extractor (order = registration order).
func (r *Registry) Register(e Extractor) {
	if e == nil || r == nil {
		return
	}
	r.extractors = append(r.extractors, e)
}

// Apply runs all extractors and merges results. Later keys overwrite earlier ones.
func (r *Registry) Apply(channel string, raw []byte) (Meta, error) {
	if r == nil || len(r.extractors) == 0 {
		return Meta{}, nil
	}
	out := Meta{}
	for i, e := range r.extractors {
		meta, err := e(channel, raw)
		if err != nil {
			return nil, fmt.Errorf("extension extractor %d: %w", i, err)
		}
		for k, v := range meta {
			out[k] = v
		}
	}
	return out, nil
}
