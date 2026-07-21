package outbound_test

import (
	"testing"
	"time"

	"github.com/kalifun/vda5050-types-go/instant_actions"
	"github.com/kalifun/vda5050-types-go/order"

	"github.com/kalifun/navlink/outbound"
)

func TestBuilderFillsVersionTimestampNotHeaderID(t *testing.T) {
	fixed := time.Date(2026, 7, 21, 4, 5, 6, 7_000_000, time.UTC)
	b := outbound.NewBuilder("v2.0.0")
	b.SetClock(func() time.Time { return fixed })

	o := &order.Order{OrderId: "o1", OrderUpdateId: 3}
	o.HeaderId = 9

	b.PrepareOrder(o, "M", "S")
	if o.Version != "v2.0.0" || o.Timestamp != "2026-07-21T04:05:06.007Z" {
		t.Fatalf("version/ts = %q %q", o.Version, o.Timestamp)
	}
	if o.HeaderId != 9 || o.OrderUpdateId != 3 {
		t.Fatalf("must not rewrite caller IDs: header=%d update=%d", o.HeaderId, o.OrderUpdateId)
	}
	if o.Manufacturer != "M" || o.SerialNumber != "S" {
		t.Fatalf("identity=%s/%s", o.Manufacturer, o.SerialNumber)
	}

	ia := &instant_actions.InstantActions{}
	ia.HeaderId = 10
	b.PrepareInstantActions(ia, "M", "S")
	if ia.Version != "v2.0.0" || ia.HeaderId != 10 {
		t.Fatalf("ia version=%q header=%d", ia.Version, ia.HeaderId)
	}
}
