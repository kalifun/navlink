package mqtt

import (
	"context"
	"sync"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/kalifun/navlink/gerrors"
)

// Config holds MQTT connection settings.
type Config struct {
	Broker        string
	ClientID      string
	Username      string
	Password      string
	QoS           byte
	KeepAlive     time.Duration
	CleanSession  bool
	AutoReconnect bool

	// OnReconnect is called after a successful reconnect (not the initial connect).
	OnReconnect func()
}

// Handler handles a raw MQTT message.
type Handler func(ctx context.Context, topic string, payload []byte) error

// Unsubscribe cancels a subscription.
type Unsubscribe func(ctx context.Context) error

// Transport is a thin MQTT byte transport. It does not understand VDA5050.
type Transport struct {
	cfg    Config
	client pahomqtt.Client

	mu           sync.RWMutex
	running      bool
	wasConnected bool
	onReconnect  func()
}

// New creates an MQTT transport (does not connect).
func New(cfg Config) *Transport {
	return &Transport{cfg: cfg, onReconnect: cfg.OnReconnect}
}

// SetOnReconnect sets the reconnect callback (overrides Config.OnReconnect).
func (t *Transport) SetOnReconnect(fn func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onReconnect = fn
}

// Start connects to the broker.
func (t *Transport) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.running {
		return gerrors.MQTTTransportAlreadyRunning
	}
	if t.cfg.Broker == "" || t.cfg.ClientID == "" {
		return gerrors.NewConfigurationErrorWithArgs("broker and clientId are required")
	}

	keepAlive := t.cfg.KeepAlive
	if keepAlive == 0 {
		keepAlive = 30 * time.Second
	}

	opts := pahomqtt.NewClientOptions()
	opts.AddBroker(t.cfg.Broker)
	opts.SetClientID(t.cfg.ClientID)
	opts.SetUsername(t.cfg.Username)
	opts.SetPassword(t.cfg.Password)
	opts.SetKeepAlive(keepAlive)
	opts.SetCleanSession(t.cfg.CleanSession)
	opts.SetAutoReconnect(t.cfg.AutoReconnect)
	opts.SetConnectRetry(t.cfg.AutoReconnect)
	opts.SetOnConnectHandler(t.onConnect)

	t.client = pahomqtt.NewClient(opts)
	token := t.client.Connect()
	if !token.WaitTimeout(30 * time.Second) {
		return gerrors.TimeoutError
	}
	if err := token.Error(); err != nil {
		return gerrors.ConnectionFailed.With("cause", err.Error())
	}

	t.running = true
	return nil
}

func (t *Transport) onConnect(_ pahomqtt.Client) {
	t.mu.Lock()
	first := !t.wasConnected
	t.wasConnected = true
	handler := t.onReconnect
	t.mu.Unlock()
	if !first && handler != nil {
		handler()
	}
}

// Stop disconnects from the broker.
func (t *Transport) Stop(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running {
		return nil
	}
	t.client.Disconnect(250)
	t.running = false
	t.wasConnected = false
	return nil
}

// Publish sends a payload to topic.
func (t *Transport) Publish(ctx context.Context, topic string, payload []byte, qos byte, retain bool) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.running {
		return gerrors.MQTTTransportNotRunning
	}
	if qos == 0 {
		qos = t.cfg.QoS
	}
	token := t.client.Publish(topic, qos, retain, payload)
	if !token.WaitTimeout(30 * time.Second) {
		return gerrors.TimeoutError
	}
	if err := token.Error(); err != nil {
		return gerrors.PublishFailed.With("cause", err.Error())
	}
	return nil
}

// Subscribe registers a handler for a topic filter.
func (t *Transport) Subscribe(ctx context.Context, filter string, handler Handler) (Unsubscribe, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running {
		return nil, gerrors.MQTTTransportNotRunning
	}

	cb := func(_ pahomqtt.Client, msg pahomqtt.Message) {
		_ = handler(context.Background(), msg.Topic(), msg.Payload())
	}
	token := t.client.Subscribe(filter, t.cfg.QoS, cb)
	if !token.WaitTimeout(30 * time.Second) {
		return nil, gerrors.TimeoutError
	}
	if err := token.Error(); err != nil {
		return nil, gerrors.SubscriptionFailed.With("cause", err.Error())
	}

	return func(ctx context.Context) error {
		t.mu.Lock()
		defer t.mu.Unlock()
		if !t.running {
			return nil
		}
		token := t.client.Unsubscribe(filter)
		if !token.WaitTimeout(30 * time.Second) {
			return gerrors.TimeoutError
		}
		if err := token.Error(); err != nil {
			return gerrors.NewUnsubscribeFailedWithArgs(filter, err)
		}
		return nil
	}, nil
}
