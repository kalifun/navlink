package main

import (
	"context"
	"fmt"
	"time"

	"github.com/kalifun/navlink/pkg/core"
	"github.com/kalifun/navlink/pkg/transport/mqtt"
)

// Example usage of the MQTT transport
func ExampleMQTTTransport() {
	// Create MQTT configuration
	config := mqtt.MQTTConfig{
		Broker:               "tcp://localhost:1883",
		ClientID:             "navlink-client-1",
		Username:             "username",
		Password:             "password",
		QoS:                  1,
		CleanSession:         true,
		KeepAlive:            30,
		ConnectTimeout:       30 * time.Second,
		MaxReconnectInterval: 10 * time.Minute,
		AutoReconnect:        true,
	}

	// Create MQTT transport
	transport := mqtt.NewTransport("mqtt-transport-1", config)

	// Start the transport
	ctx := context.Background()
	if err := transport.Start(ctx); err != nil {
		fmt.Printf("Failed to start MQTT transport: %v\n", err)
		return
	}
	defer transport.Stop(ctx)

	// Subscribe to a topic
	handler := func(ctx context.Context, msg *core.Message) error {
		fmt.Printf("Received message on topic %s: %s\n", msg.Topic, string(msg.Payload))
		return nil
	}

	subscription, err := transport.Subscribe(ctx, "test/topic", handler)
	if err != nil {
		fmt.Printf("Failed to subscribe: %v\n", err)
		return
	}
	defer subscription.Unsubscribe(ctx)

	// Publish a message
	payload := []byte("Hello, MQTT!")
	opts := core.PublishOptions{
		QoS:     1,
		Retain:  false,
		TimeOut: 10 * time.Second,
	}

	if err := transport.Publish(ctx, "test/topic", payload, opts); err != nil {
		fmt.Printf("Failed to publish: %v\n", err)
		return
	}

	fmt.Println("MQTT transport example completed successfully")
}

// Example using the factory
func ExampleMQTTTransportFactory() {
	// Create configuration map
	config := map[string]interface{}{
		"broker":               "tcp://localhost:1883",
		"clientId":             "navlink-client-2",
		"username":             "username",
		"password":             "password",
		"qos":                  1,
		"cleanSession":         true,
		"keepAlive":            30,
		"connectTimeout":       30,
		"maxReconnectInterval": 600,
		"autoReconnect":        true,
		"tlsConfig": map[string]interface{}{
			"caFile":   "/path/to/ca.crt",
			"certFile": "/path/to/client.crt",
			"keyFile":  "/path/to/client.key",
			"insecure": false,
		},
		"willMessage": map[string]interface{}{
			"topic":    "client/status",
			"qos":      1,
			"retained": true,
			"payload":  "offline",
		},
	}

	// Create transport using factory
	factory := mqtt.NewTransportFactory()
	transport, err := factory.CreateTransport("mqtt-transport-2", config)
	if err != nil {
		fmt.Printf("Failed to create MQTT transport: %v\n", err)
		return
	}

	// Use the transport
	ctx := context.Background()
	if err := transport.Start(ctx); err != nil {
		fmt.Printf("Failed to start MQTT transport: %v\n", err)
		return
	}
	defer transport.Stop(ctx)

	fmt.Println("MQTT transport factory example completed successfully")
}
