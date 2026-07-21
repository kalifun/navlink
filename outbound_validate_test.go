package navlink_test

import (
	"context"
	"testing"

	"github.com/kalifun/vda5050-types-go/instant_actions"
	"github.com/kalifun/vda5050-types-go/order"

	"github.com/kalifun/navlink"
	"github.com/kalifun/navlink/gerrors"
)

func TestOutboundValidationRejectsBadOrder(t *testing.T) {
	mem := &memoryTransport{}
	client, err := navlink.New(navlink.Config{
		Interface: "uagv",
		Version:   "v2",
		Transport: mem,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Stop(ctx)

	_, err = client.AGV("M", "S").PublishOrder(ctx, &order.Order{
		OrderId:       "o1",
		OrderUpdateId: 1,
		Nodes:         []order.Node{},
		Edges:         []order.Edge{},
	})
	if !navlink.IsPublishValidationFailed(err) {
		t.Fatalf("want validation failed, got %v", err)
	}
	if !navlink.IsOutboundValidationFailed(err) {
		t.Fatal(err)
	}
	if len(mem.published) != 0 {
		t.Fatalf("must not publish: %d", len(mem.published))
	}
}

func TestOutboundValidationAcceptsGoodOrder(t *testing.T) {
	mem := &memoryTransport{}
	client, err := navlink.New(navlink.Config{
		Interface: "uagv",
		Version:   "v2",
		Transport: mem,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Stop(ctx)

	ord := &order.Order{OrderId: "o1", OrderUpdateId: 1, Nodes: []order.Node{}, Edges: []order.Edge{}}
	ord.HeaderId = 3
	if _, err := client.AGV("M", "S").PublishOrder(ctx, ord); err != nil {
		t.Fatal(err)
	}
	if len(mem.published) != 1 {
		t.Fatalf("published=%d", len(mem.published))
	}
}

func TestOutboundValidationIdentityMismatch(t *testing.T) {
	mem := &memoryTransport{}
	client, err := navlink.New(navlink.Config{
		Interface: "uagv",
		Version:   "v2",
		Transport: mem,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	_ = client.Start(ctx)
	defer client.Stop(ctx)

	ord := &order.Order{OrderId: "o1", OrderUpdateId: 1, Nodes: []order.Node{}, Edges: []order.Edge{}}
	ord.HeaderId = 1
	ord.Manufacturer = "OTHER"
	_, err = client.AGV("M", "S").PublishOrder(ctx, ord)
	if !navlink.IsOutboundValidationFailed(err) {
		t.Fatalf("err=%v", err)
	}
}

func TestOutboundValidationDisabled(t *testing.T) {
	mem := &memoryTransport{}
	client, err := navlink.New(navlink.Config{
		Interface:          "uagv",
		Version:            "v2",
		Transport:          mem,
		OutboundValidation: &navlink.OutboundValidation{Disabled: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	_ = client.Start(ctx)
	defer client.Stop(ctx)

	// headerId 0 allowed when disabled
	if _, err := client.AGV("M", "S").PublishOrder(ctx, &order.Order{
		OrderId: "o", OrderUpdateId: 0, Nodes: []order.Node{}, Edges: []order.Edge{},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestOutboundValidationMissingActionID(t *testing.T) {
	mem := &memoryTransport{}
	client, err := navlink.New(navlink.Config{
		Interface: "uagv",
		Version:   "v2",
		Transport: mem,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	_ = client.Start(ctx)
	defer client.Stop(ctx)

	ia := &instant_actions.InstantActions{
		Actions: []instant_actions.InstantAction{{ActionType: "cancelOrder"}},
	}
	ia.HeaderId = 1
	_, err = client.AGV("M", "S").PublishInstantActions(ctx, ia)
	if !navlink.IsOutboundValidationFailed(err) {
		t.Fatalf("err=%v want %v", err, gerrors.OutboundValidationFailed)
	}
}
