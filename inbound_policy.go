package navlink

import (
	"sync"

	"github.com/kalifun/navlink/topic"
)

// InboundDisposition is a light headerId classification for packet acceptance.
// navlink does not accept/reject business semantics — platforms decide whether to drop.
type InboundDisposition string

const (
	InboundAccept    InboundDisposition = "accept"
	InboundStale     InboundDisposition = "stale"
	InboundDuplicate InboundDisposition = "duplicate"
)

// MetaInboundDisposition is set on Envelope.Meta when an InboundPolicy is configured.
const MetaInboundDisposition = "navlink.inboundDisposition"

// InboundPolicy classifies inbound messages by headerId for one (mfr, sn, channel).
// Default (nil Config.InboundPolicy) is accept-all — no classification.
type InboundPolicy interface {
	Classify(agv Identity, channel topic.Channel, headerID uint32) InboundDisposition
}

// HeaderSequencePolicy tracks the last accepted headerId per key and classifies
// equal as Duplicate and lower as Stale. It does not drop messages.
type HeaderSequencePolicy struct {
	mu   sync.Mutex
	last map[string]uint32
}

// NewHeaderSequencePolicy returns a policy that updates watermarks on Accept only.
func NewHeaderSequencePolicy() *HeaderSequencePolicy {
	return &HeaderSequencePolicy{last: make(map[string]uint32)}
}

// Classify implements InboundPolicy.
func (p *HeaderSequencePolicy) Classify(agv Identity, channel topic.Channel, headerID uint32) InboundDisposition {
	if p == nil {
		return InboundAccept
	}
	key := agv.Manufacturer + "\x00" + agv.SerialNumber + "\x00" + string(channel)

	p.mu.Lock()
	defer p.mu.Unlock()

	prev, ok := p.last[key]
	if !ok {
		p.last[key] = headerID
		return InboundAccept
	}
	if headerID == prev {
		return InboundDuplicate
	}
	if headerID < prev {
		return InboundStale
	}
	p.last[key] = headerID
	return InboundAccept
}
