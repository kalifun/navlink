package outbound

import (
	"time"

	vda5050 "github.com/kalifun/vda5050-types-go"
	"github.com/kalifun/vda5050-types-go/instant_actions"
	"github.com/kalifun/vda5050-types-go/order"
)

// TimestampLayout is RFC3339 with millisecond precision in UTC (Zulu).
const TimestampLayout = "2006-01-02T15:04:05.000Z"

// Builder applies wire-format defaults for outbound VDA5050 messages.
//
// navlink is an execution endpoint: headerId / orderUpdateId / actionId are
// owned by the caller. This builder only fills identity blanks,
// ProtocolHeader.Version, and timestamp.
type Builder struct {
	Version string
	now     func() time.Time
}

// NewBuilder constructs a Builder. Version is written into every outbound header.
func NewBuilder(version string) *Builder {
	return &Builder{
		Version: version,
		now:     func() time.Time { return time.Now().UTC() },
	}
}

// SetClock overrides the timestamp clock (tests).
func (b *Builder) SetClock(now func() time.Time) {
	if now != nil {
		b.now = now
	}
}

// ApplyHeader sets manufacturer/serial when empty, and always sets version + timestamp.
// HeaderId is never allocated here.
func (b *Builder) ApplyHeader(h *vda5050.ProtocolHeader, manufacturer, serial string) {
	if h == nil {
		return
	}
	if h.Manufacturer == "" {
		h.Manufacturer = manufacturer
	}
	if h.SerialNumber == "" {
		h.SerialNumber = serial
	}
	h.Version = b.Version
	h.Timestamp = b.now().Format(TimestampLayout)
}

// PrepareOrder applies wire defaults. Caller must set OrderId / OrderUpdateId / HeaderId.
func (b *Builder) PrepareOrder(o *order.Order, manufacturer, serial string) {
	if o == nil {
		return
	}
	b.ApplyHeader(&o.ProtocolHeader, manufacturer, serial)
}

// PrepareInstantActions applies wire defaults. Caller must set HeaderId and actionIds.
func (b *Builder) PrepareInstantActions(ia *instant_actions.InstantActions, manufacturer, serial string) {
	if ia == nil {
		return
	}
	b.ApplyHeader(&ia.ProtocolHeader, manufacturer, serial)
}
