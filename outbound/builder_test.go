package outbound_test

import (
	"testing"
	"time"

	"github.com/kalifun/vda5050-types-go/instant_actions"
	"github.com/kalifun/vda5050-types-go/order"

	"github.com/kalifun/navlink/outbound"
)

func TestBuilderSameVersionForOrderAndInstantActions(t *testing.T) {
	fixed := time.Date(2026, 7, 21, 4, 5, 6, 7_000_000, time.UTC)
	b := outbound.NewBuilder("v2.0.0", outbound.NewMemoryHeaderIDs(), nil, nil)
	b.SetClock(func() time.Time { return fixed })

	o := &order.Order{OrderId: "o1"}
	ia := &instant_actions.InstantActions{}

	if _, err := b.PrepareOrder(o, "M", "S"); err != nil {
		t.Fatal(err)
	}
	if err := b.PrepareInstantActions(ia, "M", "S"); err != nil {
		t.Fatal(err)
	}
	if o.Version != "v2.0.0" || ia.Version != "v2.0.0" {
		t.Fatalf("versions order=%q ia=%q", o.Version, ia.Version)
	}
	wantTS := "2026-07-21T04:05:06.007Z"
	if o.Timestamp != wantTS || ia.Timestamp != wantTS {
		t.Fatalf("timestamps order=%q ia=%q", o.Timestamp, ia.Timestamp)
	}
	if o.HeaderId == 0 || ia.HeaderId == 0 || o.HeaderId == ia.HeaderId {
		t.Fatalf("headerIds should be distinct monotonic: order=%d ia=%d", o.HeaderId, ia.HeaderId)
	}
}

func TestBuilderAllocatesOrderUpdateId(t *testing.T) {
	store := outbound.NewMemoryOrderUpdateIDs()
	b := outbound.NewBuilder("v2", outbound.NewMemoryHeaderIDs(), store, nil)
	o := &order.Order{OrderId: "o1"}
	commitID, err := b.PrepareOrder(o, "M", "S")
	if err != nil {
		t.Fatal(err)
	}
	if o.OrderUpdateId != 1 || commitID != 1 {
		t.Fatalf("updateId=%d commitID=%d", o.OrderUpdateId, commitID)
	}
	if err := b.CommitOrderUpdate(o.OrderId, commitID); err != nil {
		t.Fatal(err)
	}
	o2 := &order.Order{OrderId: "o1"}
	commitID, err = b.PrepareOrder(o2, "M", "S")
	if err != nil {
		t.Fatal(err)
	}
	if o2.OrderUpdateId != 2 || commitID != 2 {
		t.Fatalf("second updateId=%d commitID=%d", o2.OrderUpdateId, commitID)
	}
}
