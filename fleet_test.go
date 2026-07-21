package navlink_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/kalifun/vda5050-types-go/state"

	"github.com/kalifun/navlink"
	"github.com/kalifun/navlink/extend"
	"github.com/kalifun/navlink/session"
)

func TestFleetSessionTracksOnConnectionOnline(t *testing.T) {
	mem := &memoryTransport{}
	opts := session.DefaultOptions()
	client, err := navlink.New(navlink.Config{
		Interface: "uagv",
		Version:   "v2",
		Transport: mem,
		Fleet:     &opts,
	})
	if err != nil {
		t.Fatal(err)
	}

	var gotNode string
	client.OnState(func(ctx context.Context, env navlink.Envelope, st *state.State) error {
		gotNode = st.LastNodeId
		return nil
	})

	var online navlink.Identity
	client.OnAGVOnline(func(id navlink.Identity) { online = id })

	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Stop(ctx)

	if !mem.hasFilter("uagv/v2/+/+/connection") {
		t.Fatalf("filters=%v", mem.filters())
	}
	if mem.hasFilter("uagv/v2/+/+/state") {
		t.Fatal("fleet mode must not wildcard-subscribe state")
	}

	connPayload, _ := json.Marshal(map[string]any{
		"headerId": 1, "timestamp": "2026-07-21T00:00:00.000Z", "version": "v2",
		"manufacturer": "M", "serialNumber": "S1", "connectionState": "ONLINE",
	})
	if err := mem.Publish(ctx, "uagv/v2/M/S1/connection", connPayload, navlink.PublishOptions{}); err != nil {
		t.Fatal(err)
	}
	if online.SerialNumber != "S1" {
		t.Fatalf("online=%+v", online)
	}
	if !mem.hasFilter("uagv/v2/M/S1/state") {
		t.Fatalf("expected per-AGV state, filters=%v", mem.filters())
	}

	statePayload, _ := json.Marshal(map[string]any{
		"headerId": 1, "timestamp": "2026-07-21T00:00:00.000Z", "version": "v2",
		"manufacturer": "M", "serialNumber": "S1",
		"orderId": "", "orderUpdateId": 0, "lastNodeId": "N9", "lastNodeSequenceId": 0,
		"nodeStates": []any{}, "edgeStates": []any{}, "actionStates": []any{},
		"batteryState":  map[string]any{"batteryCharge": 80.0, "charging": false},
		"operatingMode": "AUTOMATIC", "errors": []any{},
		"safetyState": map[string]any{"eStop": "NONE", "fieldViolation": false},
	})
	if err := mem.Publish(ctx, "uagv/v2/M/S1/state", statePayload, navlink.PublishOptions{}); err != nil {
		t.Fatal(err)
	}
	if gotNode != "N9" {
		t.Fatalf("gotNode=%q", gotNode)
	}
}

func TestExtensionMetaOnState(t *testing.T) {
	mem := &memoryTransport{}
	reg := extend.NewRegistry()
	// Consumer-owned extractor (not a navlink built-in vendor module).
	reg.Register(func(channel string, raw []byte) (extend.Meta, error) {
		if channel != "state" {
			return nil, nil
		}
		var probe struct {
			Extra string `json:"extraField"`
		}
		if err := json.Unmarshal(raw, &probe); err != nil {
			return nil, err
		}
		if probe.Extra == "" {
			return nil, nil
		}
		return extend.Meta{"ExtraField": probe.Extra}, nil
	})
	client, err := navlink.New(navlink.Config{
		Interface:  "uagv",
		Version:    "v2",
		Transport:  mem,
		Extensions: reg,
	})
	if err != nil {
		t.Fatal(err)
	}

	var reported any
	client.OnState(func(ctx context.Context, env navlink.Envelope, st *state.State) error {
		reported = env.Meta["ExtraField"]
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
		"orderId": "", "orderUpdateId": 0, "lastNodeId": "N1", "lastNodeSequenceId": 0,
		"extraField": "from-consumer",
		"nodeStates": []any{}, "edgeStates": []any{}, "actionStates": []any{},
		"batteryState":  map[string]any{"batteryCharge": 80.0, "charging": false},
		"operatingMode": "AUTOMATIC", "errors": []any{},
		"safetyState": map[string]any{"eStop": "NONE", "fieldViolation": false},
	})
	if err := mem.Publish(ctx, "uagv/v2/M/S1/state", payload, navlink.PublishOptions{}); err != nil {
		t.Fatal(err)
	}
	if reported != "from-consumer" {
		t.Fatalf("reported=%v", reported)
	}
}
