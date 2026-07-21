package navlink_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/kalifun/vda5050-types-go/state"

	"github.com/kalifun/navlink"
)

func TestEventBusL1AndCustom(t *testing.T) {
	mem := &memoryTransport{}
	client, err := navlink.New(navlink.Config{
		Interface: "uagv",
		Version:   "v2",
		Transport: mem,
		Bus:       navlink.NewMemoryEventBus(),
	})
	if err != nil {
		t.Fatal(err)
	}

	var gotSerial string
	client.OnState(func(ctx context.Context, env navlink.Envelope, st *state.State) error {
		gotSerial = env.AGV.SerialNumber
		_ = client.Emit("rcs.demo.derived", env.AGV.SerialNumber)
		return nil
	})

	var custom any
	_, err = client.Subscribe("rcs.demo.derived", func(ctx context.Context, payload any) error {
		custom = payload
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Stop(ctx)

	payload, _ := json.Marshal(map[string]any{
		"headerId": 1, "timestamp": "2026-07-21T00:00:00.000Z", "version": "v2",
		"manufacturer": "M", "serialNumber": "S1",
		"orderId": "", "orderUpdateId": 0, "lastNodeId": "N1", "lastNodeSequenceId": 0,
		"nodeStates": []any{}, "edgeStates": []any{}, "actionStates": []any{},
		"batteryState":  map[string]any{"batteryCharge": 80.0, "charging": false},
		"operatingMode": "AUTOMATIC", "errors": []any{},
		"safetyState": map[string]any{"eStop": "NONE", "fieldViolation": false},
	})
	if err := mem.Publish(ctx, "uagv/v2/M/S1/state", payload, navlink.PublishOptions{}); err != nil {
		t.Fatal(err)
	}
	if gotSerial != "S1" || custom != "S1" {
		t.Fatalf("serial=%q custom=%v", gotSerial, custom)
	}
}

func TestEventBusDecodeFailed(t *testing.T) {
	mem := &memoryTransport{}
	client, err := navlink.New(navlink.Config{
		Interface: "uagv",
		Version:   "v2",
		Transport: mem,
		Bus:       navlink.NewMemoryEventBus(),
	})
	if err != nil {
		t.Fatal(err)
	}
	client.OnState(func(ctx context.Context, env navlink.Envelope, st *state.State) error {
		t.Fatal("should not run")
		return nil
	})
	var saw bool
	_, _ = client.Subscribe(navlink.EventDecodeFailed, func(ctx context.Context, payload any) error {
		saw = true
		return nil
	})

	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Stop(ctx)

	_ = mem.Publish(ctx, "uagv/v2/M/S1/state", []byte(`{`), navlink.PublishOptions{})
	if !saw {
		t.Fatal("expected decode failed event")
	}
}
