package navlink_test

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"

	"github.com/kalifun/vda5050-types-go/state"

	"github.com/kalifun/navlink"
	"github.com/kalifun/navlink/testkit"
	"github.com/kalifun/navlink/topic"
)

func TestHeaderSequencePolicy(t *testing.T) {
	p := navlink.NewHeaderSequencePolicy()
	agv := navlink.Identity{Manufacturer: "M", SerialNumber: "S"}
	ch := topic.ChannelState

	if d := p.Classify(agv, ch, 1); d != navlink.InboundAccept {
		t.Fatalf("first=%s", d)
	}
	if d := p.Classify(agv, ch, 1); d != navlink.InboundDuplicate {
		t.Fatalf("dup=%s", d)
	}
	if d := p.Classify(agv, ch, 0); d != navlink.InboundStale {
		t.Fatalf("stale=%s", d)
	}
	if d := p.Classify(agv, ch, 2); d != navlink.InboundAccept {
		t.Fatalf("next=%s", d)
	}
}

func TestInboundPolicyAnnotatesEnvelope(t *testing.T) {
	broker := testkit.NewFakeBroker()
	policy := navlink.NewHeaderSequencePolicy()
	client, err := navlink.New(navlink.Config{
		Interface:     "uagv",
		Version:       "v2",
		Transport:     broker,
		InboundPolicy: policy,
	})
	if err != nil {
		t.Fatal(err)
	}

	var last atomic.Value
	client.OnState(func(ctx context.Context, env navlink.Envelope, st *state.State) error {
		last.Store(env)
		return nil
	})

	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Stop(ctx)

	payload := mustStateJSON(t, 5)
	topicName := "uagv/v2/M/S1/state"
	if err := broker.Inject(ctx, topicName, payload); err != nil {
		t.Fatal(err)
	}
	env1 := last.Load().(navlink.Envelope)
	if env1.InboundDisposition != navlink.InboundAccept {
		t.Fatalf("disp=%q meta=%v", env1.InboundDisposition, env1.Meta)
	}
	if env1.Meta[navlink.MetaInboundDisposition] != string(navlink.InboundAccept) {
		t.Fatalf("meta=%v", env1.Meta)
	}

	if err := broker.Inject(ctx, topicName, payload); err != nil {
		t.Fatal(err)
	}
	env2 := last.Load().(navlink.Envelope)
	if env2.InboundDisposition != navlink.InboundDuplicate {
		t.Fatalf("disp=%q", env2.InboundDisposition)
	}
}

func mustStateJSON(t *testing.T, headerID uint32) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"headerId": headerID, "timestamp": "2026-07-21T00:00:00.000Z", "version": "v2",
		"manufacturer": "M", "serialNumber": "S1",
		"orderId": "", "orderUpdateId": 0, "lastNodeId": "N1", "lastNodeSequenceId": 0,
		"nodeStates": []any{}, "edgeStates": []any{}, "actionStates": []any{},
		"batteryState":  map[string]any{"batteryCharge": 80.0, "charging": false},
		"operatingMode": "AUTOMATIC", "errors": []any{},
		"safetyState": map[string]any{"eStop": "NONE", "fieldViolation": false},
	})
	if err != nil {
		t.Fatal(err)
	}
	return b
}
