// Dispatch egress sketch: GetNext → fill → Publish → Record only if PublishAccepted.
//
// This is documentation-as-code for new warehouses. The "ledger" is in-memory;
// real dispatchers swap in their own ID watermark / unique-publish bookkeeping.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/kalifun/vda5050-types-go/order"

	"github.com/kalifun/navlink"
	"github.com/kalifun/navlink/testkit"
)

func main() {
	ctx := context.Background()
	broker := testkit.NewFakeBroker()
	client, err := navlink.New(navlink.Config{
		Interface: "uagv",
		Version:   "v2",
		Transport: broker,
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := client.Start(ctx); err != nil {
		log.Fatal(err)
	}
	defer client.Stop(ctx)

	ledger := &fakeLedger{nextHeader: 1, nextUpdate: 1}
	if err := publishOrderOnce(ctx, client, ledger, "M", "S1", "order-42"); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("recorded publishes: %v\n", ledger.recorded)
	fmt.Printf("broker saw %d message(s)\n", len(broker.Published()))
}

func publishOrderOnce(ctx context.Context, client *navlink.Client, ledger *fakeLedger, mfr, sn, orderID string) error {
	headerID, updateID := ledger.GetNext(mfr, sn, orderID)

	ord := &order.Order{
		OrderId:       orderID,
		OrderUpdateId: updateID,
		Nodes:         []order.Node{},
		Edges:         []order.Edge{},
	}
	ord.HeaderId = headerID

	res, err := client.AGV(mfr, sn).PublishOrder(ctx, ord)
	switch {
	case navlink.PublishAccepted(err):
		// MQTT QoS handshake OK — not "vehicle accepted order".
		ledger.RecordSuccessfulPublish(mfr, sn, orderID, res.HeaderID, res.OrderUpdateID)
		return nil
	case navlink.IsPublishValidationFailed(err):
		// Bad packet: fix fields; do not bump watermark / do not Record.
		return fmt.Errorf("validation: %w", err)
	case navlink.IsPublishNotStarted(err):
		return fmt.Errorf("client not ready: %w", err)
	case navlink.IsPublishTimeout(err), navlink.IsPublishCanceled(err), navlink.IsPublishBrokerRejected(err):
		// Same IDs may be retried — ledger was not advanced on Record.
		return fmt.Errorf("transport (retry same ids header=%d update=%d): %w", headerID, updateID, err)
	default:
		return err
	}
}

// fakeLedger stands in for dispatcher ID bookkeeping.
type fakeLedger struct {
	nextHeader uint32
	nextUpdate uint32
	recorded   []string
}

func (l *fakeLedger) GetNext(mfr, sn, orderID string) (headerID, orderUpdateID uint32) {
	h, u := l.nextHeader, l.nextUpdate
	return h, u
}

func (l *fakeLedger) RecordSuccessfulPublish(mfr, sn, orderID string, headerID, orderUpdateID uint32) {
	l.recorded = append(l.recorded, fmt.Sprintf("%s/%s %s h=%d u=%d", mfr, sn, orderID, headerID, orderUpdateID))
	l.nextHeader = headerID + 1
	l.nextUpdate = orderUpdateID + 1
}
