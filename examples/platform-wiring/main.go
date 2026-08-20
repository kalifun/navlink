package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/kalifun/vda5050-types-go/connection"
	"github.com/kalifun/vda5050-types-go/state"

	"github.com/kalifun/navlink"
)

// Central wiring: typed On* for VDA (L1). Platform/domain events use a separate bus.
// Do not UseEventBus(fleetBus) — a shared synchronous bus serializes protocol
// handlers with fleet.robot.* and can deadlock.
func main() {
	platformBus := navlink.NewMemoryEventBus()

	client, err := navlink.New(navlink.Config{
		Broker:    env("NAVLINK_BROKER", "tcp://localhost:1883"),
		ClientID:  env("NAVLINK_CLIENT_ID", "navlink-platform-wiring"),
		Interface: env("NAVLINK_INTERFACE", "uagv"),
		Version:   env("NAVLINK_VERSION", "v2"),
		OnDecodeError: func(env navlink.Envelope, err error) {
			fmt.Printf("L1 decode failed topic=%s err=%v\n", env.Topic, err)
		},
		OnHandlerError: func(env navlink.Envelope, err error) {
			fmt.Printf("L1 handler error topic=%s err=%v\n", env.Topic, err)
		},
	})
	if err != nil {
		fatal(err)
	}

	wire(client, platformBus)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := client.Start(ctx); err != nil {
		fatal(err)
	}
	defer client.Stop(context.Background())
	<-ctx.Done()
}

func wire(client *navlink.Client, platformBus navlink.EventBus) {
	_, _ = platformBus.Subscribe("rcs.ingress.state_seen", func(ctx context.Context, payload any) error {
		fmt.Printf("platform handler: %v\n", payload)
		return nil
	})

	client.OnState(func(ctx context.Context, env navlink.Envelope, st *state.State) error {
		fmt.Printf("L1 state serial=%s lastNode=%s\n", env.AGV.SerialNumber, st.LastNodeId)
		return platformBus.Publish(ctx, "rcs.ingress.state_seen", map[string]any{
			"serial": env.AGV.SerialNumber,
			"node":   st.LastNodeId,
		})
	})
	client.OnConnection(func(ctx context.Context, env navlink.Envelope, c *connection.Connection) error {
		fmt.Printf("L1 connection serial=%s state=%s\n", env.AGV.SerialNumber, c.ConnectionState)
		return nil
	})
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
