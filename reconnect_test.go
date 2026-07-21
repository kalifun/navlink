package navlink_test

import (
	"context"
	"testing"

	"github.com/kalifun/vda5050-types-go/state"

	"github.com/kalifun/navlink"
	"github.com/kalifun/navlink/session"
	"github.com/kalifun/navlink/testkit"
)

func TestReconnectRestoresSubscriptions(t *testing.T) {
	broker := testkit.NewFakeBroker()
	client, err := navlink.New(navlink.Config{
		Interface: "uagv",
		Version:   "v2",
		Transport: broker,
	})
	if err != nil {
		t.Fatal(err)
	}
	client.OnState(func(ctx context.Context, env navlink.Envelope, st *state.State) error {
		return nil
	})

	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Stop(ctx)

	if !containsFilter(broker.Filters(), "uagv/v2/+/+/state") {
		t.Fatalf("before reconnect filters=%v", broker.Filters())
	}

	broker.SimulateReconnect()

	if !containsFilter(broker.Filters(), "uagv/v2/+/+/state") {
		t.Fatalf("after reconnect filters=%v", broker.Filters())
	}
}

func TestReconnectRestoresFleetTracked(t *testing.T) {
	broker := testkit.NewFakeBroker()
	opts := session.DefaultOptions()
	opts.AutoTrackFromConnection = false
	client, err := navlink.New(navlink.Config{
		Interface: "uagv",
		Version:   "v2",
		Transport: broker,
		Fleet:     &opts,
	})
	if err != nil {
		t.Fatal(err)
	}
	client.OnState(func(ctx context.Context, env navlink.Envelope, st *state.State) error {
		return nil
	})

	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Stop(ctx)

	if err := client.Track(ctx, "M", "S1"); err != nil {
		t.Fatal(err)
	}
	if !containsFilter(broker.Filters(), "uagv/v2/M/S1/state") {
		t.Fatalf("tracked filters=%v", broker.Filters())
	}

	broker.SimulateReconnect()

	if !containsFilter(broker.Filters(), "uagv/v2/+/+/connection") {
		t.Fatalf("connection missing after reconnect: %v", broker.Filters())
	}
	if !containsFilter(broker.Filters(), "uagv/v2/M/S1/state") {
		t.Fatalf("tracked state missing after reconnect: %v", broker.Filters())
	}
}

func containsFilter(filters []string, want string) bool {
	for _, f := range filters {
		if f == want {
			return true
		}
	}
	return false
}
