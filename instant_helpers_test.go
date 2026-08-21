package navlink_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/kalifun/navlink"
	"github.com/kalifun/navlink/testkit"
	"github.com/kalifun/vda5050-types-go"
)

func startHelperClient(t *testing.T) (*navlink.Client, *testkit.FakeBroker, context.Context) {
	t.Helper()
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
	t.Cleanup(func() { _ = client.Stop(ctx) })
	return client, broker, ctx
}

func TestStdInstantHelpersPublishActionType(t *testing.T) {
	client, broker, ctx := startHelperClient(t)
	h := client.AGV("M", "S1")

	tests := []struct {
		name      string
		call      func() (navlink.PublishResult, error)
		wantType  string
		wantActID string
	}{
		{"startPause", func() (navlink.PublishResult, error) { return h.StartPause(ctx, 11, "p1") }, vda5050.ActionStartPause, "p1"},
		{"stopPause", func() (navlink.PublishResult, error) { return h.StopPause(ctx, 12, "p2") }, vda5050.ActionStopPause, "p2"},
		{"stateRequest", func() (navlink.PublishResult, error) { return h.StateRequest(ctx, 13, "s1") }, vda5050.ActionStateRequest, "s1"},
		{"factsheetRequest", func() (navlink.PublishResult, error) { return h.FactsheetRequest(ctx, 14, "f1") }, vda5050.ActionFactsheetRequest, "f1"},
		{"startCharging", func() (navlink.PublishResult, error) { return h.StartCharging(ctx, 15, "c1") }, vda5050.ActionStartCharging, "c1"},
		{"stopCharging", func() (navlink.PublishResult, error) { return h.StopCharging(ctx, 16, "c2") }, vda5050.ActionStopCharging, "c2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := len(broker.Published())
			res, err := tt.call()
			if err != nil {
				t.Fatal(err)
			}
			if res.ActionIDs[0] != tt.wantActID {
				t.Fatalf("result=%+v", res)
			}
			pubs := broker.Published()
			if len(pubs) != before+1 {
				t.Fatalf("published count=%d want %d", len(pubs), before+1)
			}
			got := pubs[len(pubs)-1]
			if got.Topic != "uagv/v2/M/S1/instantActions" {
				t.Fatalf("topic=%s", got.Topic)
			}
			a0 := firstAction(t, got.Payload)
			if a0["actionType"] != tt.wantType || a0["actionId"] != tt.wantActID {
				t.Fatalf("action=%v", a0)
			}
			if a0["blockingType"] != string(vda5050.Hard) {
				t.Fatalf("blockingType=%v", a0["blockingType"])
			}
			if _, ok := a0["actionParameters"]; ok {
				t.Fatalf("unexpected parameters: %v", a0["actionParameters"])
			}
		})
	}
}

func TestInitPositionPublishesOfficialParameters(t *testing.T) {
	client, broker, ctx := startHelperClient(t)
	res, err := client.AGV("M", "S1").InitPosition(ctx, 21, "init-1", navlink.InitPositionParams{
		X: 1.25, Y: -2.5, Theta: 0.5, MapID: "map-a", LastNodeID: "n3",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.HeaderID != 21 || res.ActionIDs[0] != "init-1" {
		t.Fatalf("result=%+v", res)
	}
	a0 := firstAction(t, broker.Published()[0].Payload)
	if a0["actionType"] != vda5050.ActionInitPosition {
		t.Fatalf("actionType=%v", a0["actionType"])
	}
	raw, ok := a0["actionParameters"].([]any)
	if !ok || len(raw) != 5 {
		t.Fatalf("actionParameters=%v", a0["actionParameters"])
	}
	got := map[string]any{}
	order := make([]string, 0, 5)
	for _, p := range raw {
		m := p.(map[string]any)
		k := m["key"].(string)
		got[k] = m["value"]
		order = append(order, k)
	}
	wantKeys := []string{"x", "y", "theta", "mapId", "lastNodeId"}
	if len(order) != 5 {
		t.Fatalf("keys=%v", order)
	}
	for i, k := range wantKeys {
		if order[i] != k {
			t.Fatalf("param order=%v want %v", order, wantKeys)
		}
	}
	if got["x"].(float64) != 1.25 || got["y"].(float64) != -2.5 || got["theta"].(float64) != 0.5 {
		t.Fatalf("numeric params=%v", got)
	}
	if got["mapId"] != "map-a" || got["lastNodeId"] != "n3" {
		t.Fatalf("id params=%v", got)
	}
}

func TestStdInstantHelperEmptyActionIDDoesNotPublish(t *testing.T) {
	client, broker, ctx := startHelperClient(t)
	_, err := client.AGV("M", "S1").StartPause(ctx, 1, "")
	if navlink.ClassifyPublish(err) != navlink.PublishOutcomeNotStarted {
		t.Fatalf("outcome=%s err=%v", navlink.ClassifyPublish(err), err)
	}
	if len(broker.Published()) != 0 {
		t.Fatalf("published=%v", broker.Published())
	}
}

func TestInitPositionMissingIDsDoesNotPublish(t *testing.T) {
	client, broker, ctx := startHelperClient(t)
	h := client.AGV("M", "S1")

	cases := []struct {
		name string
		p    navlink.InitPositionParams
	}{
		{"empty mapId", navlink.InitPositionParams{MapID: "", LastNodeID: "n1"}},
		{"blank mapId", navlink.InitPositionParams{MapID: "  ", LastNodeID: "n1"}},
		{"empty lastNodeId", navlink.InitPositionParams{MapID: "m", LastNodeID: ""}},
		{"blank lastNodeId", navlink.InitPositionParams{MapID: "m", LastNodeID: "  "}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.InitPosition(ctx, 3, "a1", tt.p)
			if navlink.ClassifyPublish(err) != navlink.PublishOutcomeNotStarted {
				t.Fatalf("outcome=%s err=%v", navlink.ClassifyPublish(err), err)
			}
			if len(broker.Published()) != 0 {
				t.Fatalf("published=%v", broker.Published())
			}
		})
	}
}

func firstAction(t *testing.T, payload []byte) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatal(err)
	}
	actions, _ := body["actions"].([]any)
	if len(actions) != 1 {
		t.Fatalf("actions=%v", body["actions"])
	}
	return actions[0].(map[string]any)
}
