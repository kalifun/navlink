package mqtt

import (
	"context"
	"crypto/tls"
	"strings"
	"sync"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/kalifun/navlink/internal/gerrors"
)

const defaultInboundQueue = 256

// Will is an MQTT last-will message.
type Will struct {
	Topic   string
	Payload []byte
	QoS     byte
	Retain  bool
}

// Config holds MQTT connection settings.
type Config struct {
	Broker         string
	ClientID       string
	Username       string
	Password       string
	QoS            byte
	QoSExplicit    bool // true: use QoS for every subscribe, including visualization
	KeepAlive      time.Duration
	ConnectTimeout time.Duration
	CleanSession   bool
	AutoReconnect  bool
	TLS            *tls.Config
	Will           *Will

	InboundQueueSize int
	OnInboundDrop    func(topic string)
	OnHandlerError   func(topic string, err error)
	OnConnectionLost func(error)
	OnReconnect      func()
}

// Handler handles a raw MQTT message.
type Handler func(ctx context.Context, topic string, payload []byte) error

// Unsubscribe cancels a subscription.
type Unsubscribe func(ctx context.Context) error

type inbound struct {
	topic   string
	payload []byte
	handler Handler
}

// Transport is a thin MQTT byte transport. It does not understand VDA5050.
type Transport struct {
	cfg    Config
	client pahomqtt.Client

	mu             sync.RWMutex
	running        bool
	stopping       bool
	wasConnected   bool
	onReconnect    func()
	onLost         func(error)
	inbound        chan inbound
	dispatchCtx    context.Context
	dispatchCancel context.CancelFunc
	dispatchWG     sync.WaitGroup
}

// New creates an MQTT transport (does not connect).
func New(cfg Config) *Transport {
	return &Transport{
		cfg:         cfg,
		onReconnect: cfg.OnReconnect,
		onLost:      cfg.OnConnectionLost,
	}
}

// SetOnReconnect sets the reconnect callback (overrides Config.OnReconnect).
func (t *Transport) SetOnReconnect(fn func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onReconnect = fn
}

// SetOnConnectionLost sets the unexpected-disconnect callback.
func (t *Transport) SetOnConnectionLost(fn func(error)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onLost = fn
}

// Connected reports whether the MQTT client is currently connected.
func (t *Transport) Connected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.running && t.client != nil && t.client.IsConnected()
}

// Start connects to the broker.
func (t *Transport) Start(ctx context.Context) error {
	t.mu.Lock()
	if t.running {
		t.mu.Unlock()
		return gerrors.MQTTTransportAlreadyRunning
	}
	if t.cfg.Broker == "" || t.cfg.ClientID == "" {
		t.mu.Unlock()
		return gerrors.NewConfigurationErrorWithArgs("broker and clientId are required")
	}
	if err := validateWill(t.cfg.Will); err != nil {
		t.mu.Unlock()
		return err
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
	opts.SetConnectionLostHandler(t.onConnectionLost)
	if t.cfg.ConnectTimeout > 0 {
		opts.SetConnectTimeout(t.cfg.ConnectTimeout)
	}
	if t.cfg.TLS != nil {
		opts.SetTLSConfig(t.cfg.TLS)
	}
	if t.cfg.Will != nil {
		opts.SetBinaryWill(t.cfg.Will.Topic, t.cfg.Will.Payload, t.cfg.Will.QoS, t.cfg.Will.Retain)
	}

	qsize := t.cfg.InboundQueueSize
	if qsize <= 0 {
		qsize = defaultInboundQueue
	}
	t.inbound = make(chan inbound, qsize)
	dispatchCtx, cancel := context.WithCancel(context.Background())
	t.dispatchCtx = dispatchCtx
	t.dispatchCancel = cancel
	t.stopping = false
	t.client = pahomqtt.NewClient(opts)
	t.dispatchWG.Add(1)
	go t.runDispatch(dispatchCtx)
	t.mu.Unlock()

	token := t.client.Connect()
	if err := waitToken(ctx, token); err != nil {
		t.shutdownDispatch()
		return err
	}
	if err := token.Error(); err != nil {
		t.shutdownDispatch()
		return gerrors.ConnectionFailed.With("cause", err.Error())
	}

	t.mu.Lock()
	t.running = true
	t.mu.Unlock()
	return nil
}

func validateWill(w *Will) error {
	if w == nil {
		return nil
	}
	if strings.TrimSpace(w.Topic) == "" {
		return gerrors.WillMessageError
	}
	if w.QoS > 2 {
		return gerrors.WillMessageError
	}
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

func (t *Transport) onConnectionLost(_ pahomqtt.Client, err error) {
	t.mu.RLock()
	stopping := t.stopping
	handler := t.onLost
	t.mu.RUnlock()
	if stopping || handler == nil {
		return
	}
	handler(err)
}

func (t *Transport) shutdownDispatch() {
	t.mu.Lock()
	cancel := t.dispatchCancel
	t.dispatchCancel = nil
	t.dispatchCtx = nil
	t.client = nil
	t.running = false
	t.wasConnected = false
	t.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	t.dispatchWG.Wait()
}

// Stop disconnects from the broker.
func (t *Transport) Stop(ctx context.Context) error {
	t.mu.Lock()
	if !t.running {
		t.mu.Unlock()
		return nil
	}
	t.stopping = true
	t.running = false
	t.wasConnected = false
	cancel := t.dispatchCancel
	t.dispatchCancel = nil
	t.dispatchCtx = nil
	client := t.client
	t.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if client != nil {
		client.Disconnect(250)
	}
	t.dispatchWG.Wait()
	return nil
}

func (t *Transport) runDispatch(ctx context.Context) {
	defer t.dispatchWG.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-t.inbound:
			if !ok {
				return
			}
			err := msg.handler(ctx, msg.topic, msg.payload)
			if err == nil {
				continue
			}
			t.mu.RLock()
			h := t.cfg.OnHandlerError
			t.mu.RUnlock()
			if h != nil {
				h(msg.topic, err)
			}
		}
	}
}

func (t *Transport) enqueue(topic string, payload []byte, handler Handler) {
	t.mu.RLock()
	ch := t.inbound
	drop := t.cfg.OnInboundDrop
	done := ctxDone(t.dispatchCtx)
	t.mu.RUnlock()
	if ch == nil {
		return
	}
	msg := inbound{
		topic:   topic,
		payload: payload,
		handler: handler,
	}
	select {
	case ch <- msg:
		return
	default:
	}
	if drop != nil {
		drop(topic)
	}
	if strings.HasSuffix(topic, "/visualization") {
		return
	}
	select {
	case ch <- msg:
	case <-done:
	}
}

func ctxDone(ctx context.Context) <-chan struct{} {
	if ctx == nil {
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	return ctx.Done()
}

// Publish sends a payload to topic. qos 0 is a real MQTT QoS 0 (not "use default").
func (t *Transport) Publish(ctx context.Context, topic string, payload []byte, qos byte, retain bool) error {
	t.mu.RLock()
	if !t.running || t.client == nil {
		t.mu.RUnlock()
		return gerrors.MQTTTransportNotRunning
	}
	client := t.client
	t.mu.RUnlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	token := client.Publish(topic, qos, retain, payload)
	if err := waitToken(ctx, token); err != nil {
		return attempted(err)
	}
	if err := token.Error(); err != nil {
		return gerrors.PublishFailed.With("cause", err.Error()).With("topic", topic)
	}
	return nil
}

// Subscribe registers a handler for a topic filter.
// The handler runs on an inbound worker, not on the paho callback goroutine.
func (t *Transport) Subscribe(ctx context.Context, filter string, handler Handler) (Unsubscribe, error) {
	t.mu.RLock()
	if !t.running || t.client == nil {
		t.mu.RUnlock()
		return nil, gerrors.MQTTTransportNotRunning
	}
	client := t.client
	qos := t.subscribeQoS(filter)
	t.mu.RUnlock()

	cb := func(_ pahomqtt.Client, msg pahomqtt.Message) {
		payload := append([]byte(nil), msg.Payload()...)
		t.enqueue(msg.Topic(), payload, handler)
	}
	token := client.Subscribe(filter, qos, cb)
	if err := waitToken(ctx, token); err != nil {
		return nil, err
	}
	if err := token.Error(); err != nil {
		return nil, gerrors.SubscriptionFailed.With("cause", err.Error())
	}

	return func(ctx context.Context) error {
		t.mu.RLock()
		if !t.running || t.client == nil {
			t.mu.RUnlock()
			return nil
		}
		client := t.client
		t.mu.RUnlock()
		token := client.Unsubscribe(filter)
		if err := waitToken(ctx, token); err != nil {
			return err
		}
		if err := token.Error(); err != nil {
			return gerrors.NewUnsubscribeFailedWithArgs(filter, err)
		}
		return nil
	}, nil
}

func (t *Transport) subscribeQoS(filter string) byte {
	if t.cfg.QoSExplicit {
		return t.cfg.QoS
	}
	if strings.HasSuffix(filter, "/visualization") {
		return 0
	}
	if t.cfg.QoS == 0 {
		return 1
	}
	return t.cfg.QoS
}
