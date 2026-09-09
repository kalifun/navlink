package navlink

// InboundDropReason explains why OnInboundDrop fired.
type InboundDropReason int

const (
	// InboundDropped means the message was discarded (visualization when the
	// inbound queue is full).
	InboundDropped InboundDropReason = iota
	// InboundBackpressured means the queue is full; the MQTT callback will
	// block until space is available (or the transport stops). The message is
	// not discarded.
	InboundBackpressured
)

func (r InboundDropReason) String() string {
	switch r {
	case InboundDropped:
		return "dropped"
	case InboundBackpressured:
		return "backpressured"
	default:
		return "unknown"
	}
}
