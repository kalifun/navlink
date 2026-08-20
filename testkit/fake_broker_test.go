package testkit_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/kalifun/vda5050-types-go/order"
	"github.com/kalifun/vda5050-types-go/state"

	"github.com/kalifun/navlink"
	"github.com/kalifun/navlink/gerrors"
	"github.com/kalifun/navlink/testkit"
)

func TestFakeBrokerInjectState(t *testing.T) {
	broker := testkit.NewFakeBroker()
	client, err := navlink.New(navlink.Config{
		Interface: "uagv",
		Version:   "v2",
		Transport: broker,
	})
	if err != nil {
		t.Fatal(err)
	}

	var got string
	client.OnState(func(ctx context.Context, env navlink.Envelope, st *state.State) error {
		got = st.LastNodeId
		return nil
	})

	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Stop(ctx)

	payload, _ := json.Marshal(map[string]any{
		"headerId": 1, "timestamp": "2026-07-21T00:00:00.000Z", "version": "v2",
		"manufacturer": "M", "serialNumber": "S1",
		"orderId": "", "orderUpdateId": 0, "lastNodeId": "N7", "lastNodeSequenceId": 0,
		"nodeStates": []any{}, "edgeStates": []any{}, "actionStates": []any{},
		"batteryState":  map[string]any{"batteryCharge": 80.0, "charging": false},
		"operatingMode": "AUTOMATIC", "errors": []any{},
		"safetyState": map[string]any{"eStop": "NONE", "fieldViolation": false},
	})
	if err := broker.Inject(ctx, "uagv/v2/M/S1/state", payload); err != nil {
		t.Fatal(err)
	}
	if got != "N7" {
		t.Fatalf("got=%q", got)
	}
	if len(broker.Published()) != 0 {
		t.Fatal("Inject must not record as Publish")
	}
}

func TestFakeBrokerRecordsPublishOrder(t *testing.T) {
	broker := testkit.NewFakeBroker()
	client, err := navlink.New(navlink.Config{
		Interface: "uagv",
		Version:   "v2",
		Transport: broker,
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
	ord.HeaderId = 1
	if _, err := client.AGV("M", "S1").PublishOrder(ctx, ord); err != nil {
		t.Fatal(err)
	}
	pubs := broker.Published()
	if len(pubs) != 1 || pubs[0].Topic != "uagv/v2/M/S1/order" {
		t.Fatalf("pubs=%+v", pubs)
	}
}

func TestFakeBrokerFailNextPublish(t *testing.T) {
	broker := testkit.NewFakeBroker()
	client, err := navlink.New(navlink.Config{
		Interface: "uagv",
		Version:   "v2",
		Transport: broker,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Stop(ctx)

	broker.FailNextPublish(gerrors.PublishFailed)
	ord := &order.Order{OrderId: "o1", OrderUpdateId: 1, Nodes: []order.Node{}, Edges: []order.Edge{}}
	ord.HeaderId = 1
	_, err = client.AGV("M", "S1").PublishOrder(ctx, ord)
	if navlink.ClassifyPublish(err) != navlink.PublishOutcomeNotStarted {
		t.Fatalf("outcome=%s err=%v", navlink.ClassifyPublish(err), err)
	}
	if len(broker.Published()) != 0 {
		t.Fatal("fail must not record")
	}
}

func TestFakeBrokerHangNextPublish(t *testing.T) {
	broker := testkit.NewFakeBroker()
	client, err := navlink.New(navlink.Config{
		Interface: "uagv",
		Version:   "v2",
		Transport: broker,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Stop(ctx)

	broker.HangNextPublish(time.Millisecond)
	ord := &order.Order{OrderId: "o1", OrderUpdateId: 1, Nodes: []order.Node{}, Edges: []order.Edge{}}
	ord.HeaderId = 1
	_, err = client.AGV("M", "S1").PublishOrder(ctx, ord)
	if navlink.ClassifyPublish(err) != navlink.PublishOutcomeUncertain {
		t.Fatalf("outcome=%s err=%v", navlink.ClassifyPublish(err), err)
	}
	if len(broker.Published()) != 0 {
		t.Fatal("hang must not record")
	}
}

func TestRecordingTransport(t *testing.T) {
	inner := testkit.NewFakeBroker()
	rec := &testkit.Recorder{}
	rt := testkit.NewRecordingTransport(inner, rec)

	ctx := context.Background()
	_ = rt.Start(ctx)
	defer rt.Stop(ctx)

	unsub, err := rt.Subscribe(ctx, "a/b", func(ctx context.Context, topic string, payload []byte) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = rt.Publish(ctx, "a/b", []byte("x"), navlink.PublishOptions{})
	_ = unsub(ctx)

	if len(rec.SubscribedFilters()) != 1 || rec.SubscribedFilters()[0] != "a/b" {
		t.Fatalf("subs=%v", rec.SubscribedFilters())
	}
	if len(rec.Published()) != 1 {
		t.Fatalf("pubs=%d", len(rec.Published()))
	}
}
