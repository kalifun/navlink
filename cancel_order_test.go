package navlink_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/kalifun/navlink"
	"github.com/kalifun/navlink/testkit"
)

func TestCancelOrderPublishesInstantAction(t *testing.T) {
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

	res, err := client.AGV("M", "S1").CancelOrder(ctx, 7, "act-1")
	if err != nil {
		t.Fatal(err)
	}
	if res.HeaderID != 7 || len(res.ActionIDs) != 1 || res.ActionIDs[0] != "act-1" {
		t.Fatalf("result=%+v", res)
	}
	pubs := broker.Published()
	if len(pubs) != 1 || pubs[0].Topic != "uagv/v2/M/S1/instantActions" {
		t.Fatalf("pubs=%+v", pubs)
	}
	var body map[string]any
	if err := json.Unmarshal(pubs[0].Payload, &body); err != nil {
		t.Fatal(err)
	}
	if uint32(body["headerId"].(float64)) != 7 {
		t.Fatalf("headerId=%v", body["headerId"])
	}
	actions, _ := body["actions"].([]any)
	if len(actions) != 1 {
		t.Fatalf("actions=%v", body["actions"])
	}
	a0 := actions[0].(map[string]any)
	if a0["actionType"] != "cancelOrder" || a0["actionId"] != "act-1" {
		t.Fatalf("action=%v", a0)
	}
}
