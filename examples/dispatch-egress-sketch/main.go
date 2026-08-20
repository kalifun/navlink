// Dispatch egress sketch: GetNext → fill → Publish → Record / return / fence
// from ClassifyPublish. Timeout is Uncertain — same IDs must not be reused.
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
	switch navlink.ClassifyPublish(err) {
	case navlink.PublishOutcomeAccepted:
		ledger.RecordSuccessfulPublish(mfr, sn, orderID, res.HeaderID, res.OrderUpdateID)
		return nil
	case navlink.PublishOutcomeNotStarted:
		return fmt.Errorf("not started (same ids reusable header=%d update=%d): %w", headerID, updateID, err)
	case navlink.PublishOutcomeUncertain:
		ledger.Fence(headerID, updateID)
		return fmt.Errorf("uncertain (do not reuse header=%d update=%d): %w", headerID, updateID, err)
	default:
		return err
	}
}

type fakeLedger struct {
	nextHeader uint32
	nextUpdate uint32
	recorded   []string
	fenced     []string
}

func (l *fakeLedger) GetNext(mfr, sn, orderID string) (headerID, orderUpdateID uint32) {
	return l.nextHeader, l.nextUpdate
}

func (l *fakeLedger) RecordSuccessfulPublish(mfr, sn, orderID string, headerID, orderUpdateID uint32) {
	l.recorded = append(l.recorded, fmt.Sprintf("%s/%s %s h=%d u=%d", mfr, sn, orderID, headerID, orderUpdateID))
	l.nextHeader = headerID + 1
	l.nextUpdate = orderUpdateID + 1
}

func (l *fakeLedger) Fence(headerID, orderUpdateID uint32) {
	l.fenced = append(l.fenced, fmt.Sprintf("h=%d u=%d", headerID, orderUpdateID))
	l.nextHeader = headerID + 1
	l.nextUpdate = orderUpdateID + 1
}
