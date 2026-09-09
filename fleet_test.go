package navlink_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/kalifun/vda5050-types-go/connection"
	"github.com/kalifun/vda5050-types-go/state"

	"github.com/kalifun/navlink"
	"github.com/kalifun/navlink/extend"
)

func TestFleetDefaultDoesNotAutoTrackOnConnection(t *testing.T) {
	mem := &memoryTransport{}
	opts := navlink.DefaultFleetOptions()
	client, err := navlink.New(navlink.Config{
		Interface: "uagv",
		Version:   "v2",
		Transport: mem,
		Fleet:     &opts,
	})
	if err != nil {
		t.Fatal(err)
	}

	var online navlink.Identity
	client.OnAGVOnline(func(id navlink.Identity) { online = id })

	ctx := t.Context()
	if err := client.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Stop(ctx)

	connPayload, _ := json.Marshal(map[string]any{
		"headerId": 1, "timestamp": "2026-07-21T00:00:00.000Z", "version": "v2",
		"manufacturer": "M", "serialNumber": "S1", "connectionState": "ONLINE",
	})
	if err := mem.Publish(ctx, "uagv/v2/M/S1/connection", connPayload, navlink.PublishOptions{}); err != nil {
		t.Fatal(err)
	}
	if online.SerialNumber != "" {
		t.Fatalf("default must not auto-track, online=%+v", online)
	}
	if mem.hasFilter("uagv/v2/M/S1/state") {
		t.Fatalf("expected no per-AGV state subscribe, filters=%v", mem.filters())
	}
}

func TestFleetExplicitTrackSubscribesState(t *testing.T) {
	mem := &memoryTransport{}
	opts := navlink.DefaultFleetOptions()
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

	ctx := t.Context()
	if err := client.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Stop(ctx)

	if err := client.Track(ctx, "M", "S1"); err != nil {
		t.Fatal(err)
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

func TestFleetAutoTrackAsyncDoesNotBlockInbound(t *testing.T) {
	release := make(chan struct{})
	entered := make(chan struct{})
	tr := &hangStateSubscribeTransport{
		memoryTransport: memoryTransport{},
		hangState:       release,
		enteredState:    entered,
	}
	opts := navlink.DefaultFleetOptions()
	opts.AutoTrackFromConnection = true
	var mu sync.Mutex
	var connSerials []string
	client, err := navlink.New(navlink.Config{
		Interface: "uagv",
		Version:   "v2",
		Transport: tr,
		Fleet:     &opts,
		OnHandlerError: func(env navlink.Envelope, err error) {
			t.Logf("handler error topic=%s err=%v", env.Topic, err)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	client.OnConnection(func(ctx context.Context, env navlink.Envelope, msg *connection.Connection) error {
		mu.Lock()
		connSerials = append(connSerials, env.AGV.SerialNumber)
		mu.Unlock()
		return nil
	})

	ctx := t.Context()
	if err := client.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() {
		close(release)
		_ = client.Stop(ctx)
	}()

	connPayload := func(sn string) []byte {
		b, _ := json.Marshal(map[string]any{
			"headerId": 1, "timestamp": "2026-07-21T00:00:00.000Z", "version": "v2",
			"manufacturer": "M", "serialNumber": sn, "connectionState": "ONLINE",
		})
		return b
	}
	if err := tr.Publish(ctx, "uagv/v2/M/S1/connection", connPayload("S1"), navlink.PublishOptions{}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("auto-track Subscribe did not start")
	}

	if err := tr.Publish(ctx, "uagv/v2/M/S2/connection", connPayload("S2"), navlink.PublishOptions{}); err != nil {
		t.Fatal(err)
	}

	deadline := time.After(2 * time.Second)
	for {
		mu.Lock()
		n := len(connSerials)
		mu.Unlock()
		if n >= 2 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("S2 connection blocked while S1 Track hung; serials=%v", connSerials)
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func waitFilter(t *testing.T, mem interface{ hasFilter(string) bool }, filter string) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		if mem.hasFilter(filter) {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("timeout waiting for filter %s", filter)
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func TestFleetAutoTrackOptInEventuallyTracks(t *testing.T) {
	mem := &memoryTransport{}
	opts := navlink.DefaultFleetOptions()
	opts.AutoTrackFromConnection = true
	client, err := navlink.New(navlink.Config{
		Interface: "uagv",
		Version:   "v2",
		Transport: mem,
		Fleet:     &opts,
	})
	if err != nil {
		t.Fatal(err)
	}

	var online navlink.Identity
	var onlineMu sync.Mutex
	client.OnAGVOnline(func(id navlink.Identity) {
		onlineMu.Lock()
		online = id
		onlineMu.Unlock()
	})

	ctx := t.Context()
	if err := client.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Stop(ctx)

	connPayload, _ := json.Marshal(map[string]any{
		"headerId": 1, "timestamp": "2026-07-21T00:00:00.000Z", "version": "v2",
		"manufacturer": "M", "serialNumber": "S1", "connectionState": "ONLINE",
	})
	if err := mem.Publish(ctx, "uagv/v2/M/S1/connection", connPayload, navlink.PublishOptions{}); err != nil {
		t.Fatal(err)
	}
	waitFilter(t, mem, "uagv/v2/M/S1/state")
	deadline := time.After(2 * time.Second)
	for {
		onlineMu.Lock()
		sn := online.SerialNumber
		onlineMu.Unlock()
		if sn == "S1" {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("online=%+v", online)
		case <-time.After(10 * time.Millisecond):
		}
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

	ctx := t.Context()
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
