package outbound

import (
	"time"

	vda5050 "github.com/kalifun/vda5050-types-go"
	"github.com/kalifun/vda5050-types-go/instant_actions"
	"github.com/kalifun/vda5050-types-go/order"

	"github.com/kalifun/navlink/gerrors"
)

// TimestampLayout is RFC3339 with millisecond precision in UTC (Zulu).
// Aligned with common platform vda5050_timestamp formatting.
const TimestampLayout = "2006-01-02T15:04:05.000Z"

// Builder fills outbound VDA5050 headers consistently for one Client.
type Builder struct {
	// Version is written into ProtocolHeader.Version for Order and InstantActions.
	Version string

	HeaderIDs    HeaderIdProvider
	OrderUpdates OrderUpdateIdStore
	ActionIDs    ActionIdAllocator

	now func() time.Time
}

// NewBuilder constructs a Builder. now may be nil (defaults to time.Now UTC).
func NewBuilder(version string, headerIDs HeaderIdProvider, orderUpdates OrderUpdateIdStore, actionIDs ActionIdAllocator) *Builder {
	return &Builder{
		Version:      version,
		HeaderIDs:    headerIDs,
		OrderUpdates: orderUpdates,
		ActionIDs:    actionIDs,
		now:          func() time.Time { return time.Now().UTC() },
	}
}

// SetClock overrides the timestamp clock (tests).
func (b *Builder) SetClock(now func() time.Time) {
	if now != nil {
		b.now = now
	}
}

// ApplyHeader sets manufacturer/serial/version/timestamp and optionally headerId.
func (b *Builder) ApplyHeader(h *vda5050.ProtocolHeader, manufacturer, serial string) error {
	if h == nil {
		return gerrors.NewIdAllocationFailedWithArgs("nil protocol header")
	}
	if h.Manufacturer == "" {
		h.Manufacturer = manufacturer
	}
	if h.SerialNumber == "" {
		h.SerialNumber = serial
	}
	// Same Client policy for all outbound message types.
	h.Version = b.Version
	h.Timestamp = b.now().Format(TimestampLayout)

	if b.HeaderIDs != nil {
		id, err := b.HeaderIDs.Next()
		if err != nil {
			return gerrors.NewIdAllocationFailedWithArgs(err.Error())
		}
		h.HeaderId = id
	}
	return nil
}

// PrepareOrder applies header fields and allocates orderUpdateId when the store
// is configured and the caller left OrderUpdateId at zero.
// Returns the update id that must be Commit'ed after a successful publish
// (zero when the store was not used for this call).
func (b *Builder) PrepareOrder(o *order.Order, manufacturer, serial string) (commitUpdateID uint32, err error) {
	if o == nil {
		return 0, nil
	}
	if err := b.ApplyHeader(&o.ProtocolHeader, manufacturer, serial); err != nil {
		return 0, err
	}
	if b.OrderUpdates == nil || o.OrderUpdateId != 0 {
		return 0, nil
	}
	id, err := b.OrderUpdates.GetNext(o.OrderId)
	if err != nil {
		return 0, gerrors.NewIdAllocationFailedWithArgs(err.Error())
	}
	o.OrderUpdateId = id
	return id, nil
}

// CommitOrderUpdate commits a previously prepared orderUpdateId.
func (b *Builder) CommitOrderUpdate(orderID string, id uint32) error {
	if b.OrderUpdates == nil || id == 0 {
		return nil
	}
	if err := b.OrderUpdates.Commit(orderID, id); err != nil {
		return gerrors.NewIdAllocationFailedWithArgs(err.Error())
	}
	return nil
}

// PrepareInstantActions applies header fields shared with Order.
func (b *Builder) PrepareInstantActions(ia *instant_actions.InstantActions, manufacturer, serial string) error {
	if ia == nil {
		return nil
	}
	return b.ApplyHeader(&ia.ProtocolHeader, manufacturer, serial)
}

// NextActionID allocates an actionId when ActionIDs is configured.
func (b *Builder) NextActionID(manufacturer, serial string) (string, error) {
	if b.ActionIDs == nil {
		return "", gerrors.NewIdAllocationFailedWithArgs("ActionIDs allocator is not configured")
	}
	id, err := b.ActionIDs.Next(manufacturer, serial)
	if err != nil {
		return "", gerrors.NewIdAllocationFailedWithArgs(err.Error())
	}
	return id, nil
}
