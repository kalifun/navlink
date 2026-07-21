package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/kalifun/vda5050-types-go/state"

	"github.com/kalifun/navlink"
)

func main() {
	client, err := navlink.New(navlink.Config{
		Broker:    env("NAVLINK_BROKER", "tcp://localhost:1883"),
		ClientID:  env("NAVLINK_CLIENT_ID", "navlink-subscribe-state"),
		Interface: env("NAVLINK_INTERFACE", "uagv"),
		Version:   env("NAVLINK_VERSION", "v2"),
	})
	if err != nil {
		fatal(err)
	}

	client.OnState(func(ctx context.Context, env navlink.Envelope, st *state.State) error {
		fmt.Printf("%s lastNodeId=%s\n", env.AGV.SerialNumber, st.LastNodeId)
		return nil
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := client.Start(ctx); err != nil {
		fatal(err)
	}
	defer client.Stop(context.Background())

	<-ctx.Done()
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
