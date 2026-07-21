package navlink_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kalifun/navlink"
	"github.com/kalifun/navlink/gerrors"
	"github.com/kalifun/vda5050-types-go/order"
)

func TestPublishErrorClassifiers(t *testing.T) {
	if !navlink.PublishAccepted(nil) {
		t.Fatal("nil should be accepted")
	}
	if navlink.PublishAccepted(gerrors.PublishFailed) {
		t.Fatal("PublishFailed must not be accepted")
	}
	if !navlink.IsPublishNotStarted(gerrors.ClientNotStarted) {
		t.Fatal("ClientNotStarted")
	}
	if !navlink.IsPublishNotStarted(gerrors.MQTTTransportNotRunning) {
		t.Fatal("MQTTTransportNotRunning")
	}
	if !navlink.IsPublishTimeout(gerrors.TimeoutError) {
		t.Fatal("TimeoutError")
	}
	if !navlink.IsPublishTimeout(context.DeadlineExceeded) {
		t.Fatal("DeadlineExceeded")
	}
	if navlink.IsPublishTimeout(context.Canceled) {
		t.Fatal("Canceled is not timeout")
	}
	if !navlink.IsPublishCanceled(context.Canceled) {
		t.Fatal("Canceled")
	}
	if navlink.IsPublishCanceled(context.DeadlineExceeded) {
		t.Fatal("DeadlineExceeded is not cancel")
	}
	if !navlink.IsPublishQoSRejected(gerrors.QosNotSupported.With("qos", byte(9))) {
		t.Fatal("QosNotSupported")
	}
	if !navlink.IsPublishBrokerRejected(gerrors.PublishFailed.With("cause", "x")) {
		t.Fatal("PublishFailed")
	}
}

func TestPublishOrderNilRejected(t *testing.T) {
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

	_, err = client.AGV("M", "S").PublishOrder(ctx, nil)
	if err == nil || navlink.PublishAccepted(err) {
		t.Fatalf("nil order must not be accepted: %v", err)
	}
	if len(mem.published) != 0 {
		t.Fatalf("published=%d", len(mem.published))
	}

	_, err = client.AGV("M", "S").PublishInstantActions(ctx, nil)
	if err == nil || navlink.PublishAccepted(err) {
		t.Fatalf("nil instantActions must not be accepted: %v", err)
	}
}

func TestPublishOrderNotStarted(t *testing.T) {
	client, err := navlink.New(navlink.Config{
		Interface: "uagv",
		Version:   "v2",
		Transport: &memoryTransport{},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.AGV("M", "S").PublishOrder(context.Background(), &order.Order{OrderId: "o"})
	if !navlink.IsPublishNotStarted(err) {
		t.Fatalf("err=%v", err)
	}
	if !errors.Is(err, gerrors.ClientNotStarted) {
		t.Fatalf("want ClientNotStarted, got %v", err)
	}
}

func TestPublishOrderRejectsBadQoS(t *testing.T) {
	mem := &memoryTransport{}
	client, err := navlink.New(navlink.Config{
		Interface: "uagv",
		Version:   "v2",
		Transport: mem,
		QoS:       9,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Stop(ctx)

	ord := &order.Order{
		OrderId:       "o",
		OrderUpdateId: 1,
		Nodes:         []order.Node{},
		Edges:         []order.Edge{},
	}
	ord.HeaderId = 1
	_, err = client.AGV("M", "S").PublishOrder(ctx, ord)
	if !navlink.IsPublishQoSRejected(err) {
		t.Fatalf("err=%v", err)
	}
	if len(mem.published) != 0 {
		t.Fatalf("must not publish on bad qos: %d", len(mem.published))
	}
}
