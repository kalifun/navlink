package navlink_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/kalifun/vda5050-types-go/order"
	"github.com/kalifun/vda5050-types-go/state"

	"github.com/kalifun/navlink"
	"github.com/kalifun/navlink/testkit"
	"github.com/kalifun/navlink/topic"
)

func TestClientOnStateTyped(t *testing.T) {
	mem := &memoryTransport{}
	client, err := navlink.New(navlink.Config{
		Interface: "uagv",
		Version:   "v2",
		Transport: mem,
	})
	if err != nil {
		t.Fatal(err)
	}

	var gotSerial string
	var gotNode string
	client.OnState(func(ctx context.Context, env navlink.Envelope, st *state.State) error {
		gotSerial = env.AGV.SerialNumber
		gotNode = st.LastNodeId
		return nil
	})

	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Stop(ctx)

	payload, _ := json.Marshal(map[string]any{
		"headerId":           1,
		"timestamp":          "2026-07-21T00:00:00.000Z",
		"version":            "v2",
		"manufacturer":       "RobotCorp",
		"serialNumber":       "AGV001",
		"orderId":            "",
		"orderUpdateId":      0,
		"lastNodeId":         "N1",
		"lastNodeSequenceId": 0,
		"nodeStates":         []any{},
		"edgeStates":         []any{},
		"actionStates":       []any{},
		"batteryState":       map[string]any{"batteryCharge": 80.0, "charging": false},
		"operatingMode":      "AUTOMATIC",
		"errors":             []any{},
		"safetyState":        map[string]any{"eStop": "NONE", "fieldViolation": false},
	})

	topicStr := client.Topics().Build("RobotCorp", "AGV001", topic.ChannelState)
	if err := mem.Publish(ctx, topicStr, payload, navlink.PublishOptions{}); err != nil {
		t.Fatal(err)
	}

	if gotSerial != "AGV001" || gotNode != "N1" {
		t.Fatalf("got serial=%q node=%q", gotSerial, gotNode)
	}
}

func TestClientDecodeFailureDoesNotPanic(t *testing.T) {
	mem := &memoryTransport{}
	var saw error
	client, err := navlink.New(navlink.Config{
		Interface: "uagv",
		Version:   "v2",
		Transport: mem,
		OnDecodeError: func(env navlink.Envelope, err error) {
			saw = err
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	client.OnState(func(ctx context.Context, env navlink.Envelope, st *state.State) error {
		t.Fatal("handler should not run")
		return nil
	})

	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Stop(ctx)

	topicStr := client.Topics().Build("RobotCorp", "AGV001", topic.ChannelState)
	if err := mem.Publish(ctx, topicStr, []byte(`{`), navlink.PublishOptions{}); err != nil {
		t.Fatal(err)
	}
	if saw == nil {
		t.Fatal("expected decode error callback")
	}
}

func TestClientHandlerErrorCallback(t *testing.T) {
	mem := &memoryTransport{}
	var saw error
	client, err := navlink.New(navlink.Config{
		Interface: "uagv",
		Version:   "v2",
		Transport: mem,
	})
	if err != nil {
		t.Fatal(err)
	}
	client.OnHandlerError(func(env navlink.Envelope, err error) {
		saw = err
	})
	client.OnState(func(ctx context.Context, env navlink.Envelope, st *state.State) error {
		return errors.New("handler boom")
	})

	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Stop(ctx)

	payload, _ := json.Marshal(map[string]any{
		"headerId": 1, "timestamp": "2026-07-21T00:00:00.000Z", "version": "v2",
		"manufacturer": "RobotCorp", "serialNumber": "AGV001",
		"orderId": "", "orderUpdateId": 0, "lastNodeId": "N1", "lastNodeSequenceId": 0,
		"nodeStates": []any{}, "edgeStates": []any{}, "actionStates": []any{},
		"batteryState":  map[string]any{"batteryCharge": 80.0, "charging": false},
		"operatingMode": "AUTOMATIC", "errors": []any{},
		"safetyState": map[string]any{"eStop": "NONE", "fieldViolation": false},
	})
	topicStr := client.Topics().Build("RobotCorp", "AGV001", topic.ChannelState)
	_ = mem.Publish(ctx, topicStr, payload, navlink.PublishOptions{})
	if saw == nil || saw.Error() != "handler boom" {
		t.Fatalf("saw=%v", saw)
	}
}

func TestClientConnected(t *testing.T) {
	broker := testkit.NewFakeBroker()
	client, err := navlink.New(navlink.Config{
		Interface: "uagv",
		Version:   "v2",
		Transport: broker,
	})
	if err != nil {
		t.Fatal(err)
	}
	if client.Connected() {
		t.Fatal("not started")
	}
	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Stop(ctx)
	if !client.Connected() {
		t.Fatal("started should be connected")
	}
}

func TestWillRejected(t *testing.T) {
	_, err := navlink.New(navlink.Config{
		Broker:    "tcp://localhost:1883",
		ClientID:  "c",
		Interface: "uagv",
		Version:   "v2",
		Will:      &navlink.LastWill{Topic: "", Payload: []byte("x")},
	})
	if err == nil {
		t.Fatal("expected will error")
	}
}

func TestPublishOrderBuildsTopic(t *testing.T) {
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

	ord := &order.Order{
		OrderId:       "o-1",
		OrderUpdateId: 1,
		Nodes:         []order.Node{},
		Edges:         []order.Edge{},
	}
	ord.HeaderId = 42
	res, err := client.AGV("RobotCorp", "AGV001").PublishOrder(ctx, ord)
	if err != nil {
		t.Fatal(err)
	}
	if !navlink.PublishAccepted(err) {
		t.Fatal("expected accepted")
	}
	if len(mem.published) != 1 {
		t.Fatalf("published=%d", len(mem.published))
	}
	want := "uagv/v2/RobotCorp/AGV001/order"
	if mem.published[0].topic != want {
		t.Fatalf("topic=%q want %q", mem.published[0].topic, want)
	}
	if res.Topic != want || res.HeaderID != 42 || res.OrderID != "o-1" || res.OrderUpdateID != 1 {
		t.Fatalf("result=%+v", res)
	}
	if res.QoS != navlink.DefaultOrderQoS {
		t.Fatalf("default qos=%d want %d", res.QoS, navlink.DefaultOrderQoS)
	}
	if ord.HeaderId != 42 || ord.OrderUpdateId != 1 || ord.Version != "v2" {
		t.Fatalf("caller IDs/version mutated: header=%d update=%d version=%q",
			ord.HeaderId, ord.OrderUpdateId, ord.Version)
	}
}

func TestPublishOrderExplicitQoS0(t *testing.T) {
	mem := &memoryTransport{}
	client, err := navlink.New(navlink.Config{
		Interface: "uagv",
		Version:   "v2",
		Transport: mem,
		QoS:       navlink.QoSOf(0),
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Stop(ctx)

	ord := &order.Order{OrderId: "o-1", OrderUpdateId: 1, Nodes: []order.Node{}, Edges: []order.Edge{}}
	ord.HeaderId = 1
	res, err := client.AGV("M", "S").PublishOrder(ctx, ord)
	if err != nil {
		t.Fatal(err)
	}
	if res.QoS != 0 {
		t.Fatalf("qos=%d", res.QoS)
	}
}
