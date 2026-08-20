package navlink

import (
	"context"
	"sync"
	"time"

	"github.com/kalifun/navlink/extend"
	"github.com/kalifun/navlink/gerrors"
	"github.com/kalifun/navlink/mqtt"
	"github.com/kalifun/navlink/outbound"
	"github.com/kalifun/navlink/session"
	"github.com/kalifun/navlink/topic"
)

// Client is the VDA5050 protocol access entrypoint.
type Client struct {
	cfg        Config
	transport  Transport
	topics     topic.Resolver
	builder    *outbound.Builder
	fleet      *session.FleetSession
	extensions *extend.Registry
	bus        EventBus

	mu        sync.RWMutex
	started   bool
	unsubs    []Unsubscribe
	busUnsubs []Unsubscribe

	wantState bool
	wantConn  bool
	wantViz   bool
	wantFS    bool

	stateHandlers []StateHandler
	connHandlers  []ConnectionHandler
	vizHandlers   []VisualizationHandler
	fsHandlers    []FactsheetHandler
	topicHandlers []topicSub

	onTransportUp           func()
	onTransportDown         func(error)
	onHandlerError          HandlerErrorHandler
	onSubscriptionsRestored func(error)
}

type topicSub struct {
	filter  string
	handler TopicHandler
}

// New validates config and constructs a Client (does not connect).
func New(cfg Config) (*Client, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	c := &Client{
		cfg: cfg,
		topics: topic.Resolver{
			Interface: cfg.Interface,
			Version:   cfg.Version,
		},
		builder:                 outbound.NewBuilder(cfg.headerVersion()),
		extensions:              cfg.Extensions,
		bus:                     cfg.Bus,
		onTransportUp:           cfg.OnTransportUp,
		onTransportDown:         cfg.OnTransportDown,
		onHandlerError:          cfg.OnHandlerError,
		onSubscriptionsRestored: cfg.OnSubscriptionsRestored,
	}

	if cfg.Transport != nil {
		c.transport = cfg.Transport
	} else {
		keepAlive := cfg.KeepAlive
		if keepAlive == 0 {
			keepAlive = 30 * time.Second
		}
		subQoS, explicit := cfg.subscribeQoSExplicit()
		var will *mqtt.Will
		if cfg.Will != nil {
			will = &mqtt.Will{
				Topic:   cfg.Will.Topic,
				Payload: cfg.Will.Payload,
				QoS:     cfg.Will.QoS,
				Retain:  cfg.Will.Retain,
			}
		}
		c.transport = newMQTTTransport(mqtt.Config{
			Broker:           cfg.Broker,
			ClientID:         cfg.ClientID,
			Username:         cfg.Username,
			Password:         cfg.Password,
			QoS:              subQoS,
			QoSExplicit:      explicit,
			KeepAlive:        keepAlive,
			ConnectTimeout:   cfg.ConnectTimeout,
			CleanSession:     cfg.CleanSession,
			AutoReconnect:    cfg.AutoReconnect,
			TLS:              cfg.TLS,
			Will:             will,
			InboundQueueSize: cfg.InboundQueueSize,
			OnInboundDrop:    cfg.OnInboundDrop,
		})
	}

	if cfg.Fleet != nil {
		opts := *cfg.Fleet
		if opts == (session.Options{}) {
			opts = session.DefaultOptions()
		}
		c.fleet = session.NewFleetSession(c.topics, c.fleetSubscribe, opts)
	}

	if cfg.restoreOnReconnect() {
		if ra, ok := c.transport.(ReconnectAware); ok {
			ra.SetOnReconnect(c.handleReconnect)
		}
	}
	if la, ok := c.transport.(ConnectionLostAware); ok {
		la.SetOnConnectionLost(func(err error) {
			c.emitTransportDown(err)
		})
	}

	return c, nil
}

// Topics returns the TopicResolver bound to this client.
func (c *Client) Topics() topic.Resolver { return c.topics }

// Transport returns the byte-level transport.
// Use this for non-VDA application MQTT; do not mix raw traffic into AGV helpers.
func (c *Client) Transport() Transport { return c.transport }

// OnState registers a typed state handler (may be called before Start).
// With an EventBus attached, the handler is registered on EventStateReceived.
func (c *Client) OnState(h StateHandler) {
	if h == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.wantState = true
	if c.bus != nil {
		_, _ = c.subscribeTypedLocked(EventStateReceived, wrapState(h))
		return
	}
	c.stateHandlers = append(c.stateHandlers, h)
}

// OnConnection registers a typed connection handler.
func (c *Client) OnConnection(h ConnectionHandler) {
	if h == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.wantConn = true
	if c.bus != nil {
		_, _ = c.subscribeTypedLocked(EventConnectionChanged, wrapConnection(h))
		return
	}
	c.connHandlers = append(c.connHandlers, h)
}

// OnVisualization registers a typed visualization handler.
func (c *Client) OnVisualization(h VisualizationHandler) {
	if h == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.wantViz = true
	if c.bus != nil {
		_, _ = c.subscribeTypedLocked(EventVisualizationReceived, wrapVisualization(h))
		return
	}
	c.vizHandlers = append(c.vizHandlers, h)
}

// OnFactsheet registers a typed factsheet handler.
func (c *Client) OnFactsheet(h FactsheetHandler) {
	if h == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.wantFS = true
	if c.bus != nil {
		_, _ = c.subscribeTypedLocked(EventFactsheetReceived, wrapFactsheet(h))
		return
	}
	c.fsHandlers = append(c.fsHandlers, h)
}

// OnTopic registers a raw/escape-hatch topic filter handler.
func (c *Client) OnTopic(filter string, h TopicHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.topicHandlers = append(c.topicHandlers, topicSub{filter: filter, handler: h})
}

// OnAGVOnline registers a FleetSession online hook (no-op when Fleet is disabled).
func (c *Client) OnAGVOnline(h func(Identity)) {
	if c.fleet == nil || h == nil {
		return
	}
	c.fleet.OnAGVOnline(func(agv session.AGV) {
		h(Identity{Manufacturer: agv.Manufacturer, SerialNumber: agv.SerialNumber})
	})
}

// OnAGVOffline registers a FleetSession offline hook (no-op when Fleet is disabled).
func (c *Client) OnAGVOffline(h func(Identity)) {
	if c.fleet == nil || h == nil {
		return
	}
	c.fleet.OnAGVOffline(func(agv session.AGV) {
		h(Identity{Manufacturer: agv.Manufacturer, SerialNumber: agv.SerialNumber})
	})
}

// Track manually tracks an AGV (subscribes per-AGV channels). Requires Fleet.
func (c *Client) Track(ctx context.Context, manufacturer, serial string) error {
	if c.fleet == nil {
		return gerrors.NewInvalidConfigWithArgs("Fleet session is not enabled")
	}
	if err := c.requireStarted(); err != nil {
		return err
	}
	return c.fleet.Track(ctx, session.AGV{Manufacturer: manufacturer, SerialNumber: serial})
}

// Untrack stops per-AGV subscriptions. Requires Fleet.
func (c *Client) Untrack(ctx context.Context, manufacturer, serial string) error {
	if c.fleet == nil {
		return gerrors.NewInvalidConfigWithArgs("Fleet session is not enabled")
	}
	if err := c.requireStarted(); err != nil {
		return err
	}
	return c.fleet.Untrack(ctx, session.AGV{Manufacturer: manufacturer, SerialNumber: serial})
}

// RestoreFleet re-subscribes fleet connection and tracked AGVs after reconnect.
func (c *Client) RestoreFleet(ctx context.Context) error {
	if c.fleet == nil {
		return nil
	}
	if err := c.requireStarted(); err != nil {
		return err
	}
	return c.fleet.Restore(ctx)
}

// AGV returns a per-vehicle outbound handle.
func (c *Client) AGV(manufacturer, serial string) *AGVHandle {
	return &AGVHandle{client: c, manufacturer: manufacturer, serial: serial}
}

// Start connects the transport and establishes subscriptions for registered handlers.
func (c *Client) Start(ctx context.Context) error {
	c.mu.Lock()

	if c.started {
		c.mu.Unlock()
		return gerrors.ClientAlreadyStarted
	}
	if err := c.transport.Start(ctx); err != nil {
		c.mu.Unlock()
		return err
	}

	if err := c.subscribeLocked(ctx); err != nil {
		_ = c.transport.Stop(ctx)
		c.mu.Unlock()
		return err
	}

	c.started = true
	up := c.onTransportUp
	c.mu.Unlock()
	if up != nil {
		up()
	}
	return nil
}

// Stop cancels subscriptions and disconnects the transport.
func (c *Client) Stop(ctx context.Context) error {
	c.mu.Lock()
	if !c.started {
		c.mu.Unlock()
		return nil
	}
	fleet := c.fleet
	unsubs := c.unsubs
	busUnsubs := c.busUnsubs
	c.unsubs = nil
	c.busUnsubs = nil
	transport := c.transport
	c.started = false
	c.mu.Unlock()

	var firstErr error
	if fleet != nil {
		if err := fleet.Stop(ctx); err != nil {
			firstErr = err
		}
	}
	for i := len(unsubs) - 1; i >= 0; i-- {
		if err := unsubs[i](ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	for i := len(busUnsubs) - 1; i >= 0; i-- {
		if busUnsubs[i] == nil {
			continue
		}
		if err := busUnsubs[i](ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if err := transport.Stop(ctx); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

func (c *Client) subscribeLocked(ctx context.Context) error {
	if c.fleet != nil {
		if err := c.fleet.Start(ctx); err != nil {
			return err
		}
		// Factsheet stays optional fleet-wide; state/viz come from Track.
		if c.wantFS {
			filter := c.subscriptionFilter(topic.ChannelFactsheet)
			unsub, err := c.transport.Subscribe(ctx, filter, c.onRawMessage)
			if err != nil {
				return err
			}
			c.unsubs = append(c.unsubs, unsub)
		}
	} else {
		channels := make([]topic.Channel, 0, 4)
		if c.wantState {
			channels = append(channels, topic.ChannelState)
		}
		if c.wantConn {
			channels = append(channels, topic.ChannelConnection)
		}
		if c.wantViz {
			channels = append(channels, topic.ChannelVisualization)
		}
		if c.wantFS {
			channels = append(channels, topic.ChannelFactsheet)
		}

		for _, ch := range channels {
			filter := c.subscriptionFilter(ch)
			unsub, err := c.transport.Subscribe(ctx, filter, c.onRawMessage)
			if err != nil {
				return err
			}
			c.unsubs = append(c.unsubs, unsub)
		}
	}

	for _, sub := range c.topicHandlers {
		handler := sub.handler
		unsub, err := c.transport.Subscribe(ctx, sub.filter, func(ctx context.Context, t string, payload []byte) error {
			return c.dispatchTopic(ctx, t, payload, handler)
		})
		if err != nil {
			return err
		}
		c.unsubs = append(c.unsubs, unsub)
	}
	return nil
}

func (c *Client) fleetSubscribe(ctx context.Context, filter string) (session.Unsubscribe, error) {
	unsub, err := c.transport.Subscribe(ctx, filter, c.onRawMessage)
	if err != nil {
		return nil, err
	}
	return session.Unsubscribe(unsub), nil
}

// handleReconnect restores VDA subscriptions after transport reconnect.
// Stale Unsubscribe handles are discarded; FleetSession.Restore re-tracks AGVs.
func (c *Client) handleReconnect() {
	ctx := context.Background()

	c.mu.Lock()
	if !c.started {
		c.mu.Unlock()
		return
	}
	c.unsubs = nil
	fleet := c.fleet
	c.mu.Unlock()

	var restoreErr error
	if fleet != nil {
		restoreErr = fleet.Restore(ctx)
	}

	c.mu.Lock()
	if !c.started {
		c.mu.Unlock()
		return
	}
	if restoreErr == nil {
		restoreErr = c.subscribeLocked(ctx)
	}
	up := c.onTransportUp
	restored := c.onSubscriptionsRestored
	c.mu.Unlock()

	if restored != nil {
		restored(restoreErr)
	}
	if restoreErr == nil && up != nil {
		up()
	}
}

// Connected reports whether the transport currently has an MQTT connection.
func (c *Client) Connected() bool {
	if cs, ok := c.transport.(ConnectionStatus); ok {
		return cs.Connected()
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}

// OnTransportUp registers a callback after a successful connect or restore.
func (c *Client) OnTransportUp(h func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onTransportUp = h
}

// OnTransportDown registers a callback for unexpected transport disconnects.
func (c *Client) OnTransportDown(h func(error)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onTransportDown = h
}

// OnHandlerError registers a callback when a typed/raw inbound handler returns an error.
func (c *Client) OnHandlerError(h HandlerErrorHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onHandlerError = h
}

// OnSubscriptionsRestored registers a callback after reconnect subscription restore.
func (c *Client) OnSubscriptionsRestored(h func(error)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onSubscriptionsRestored = h
}

func (c *Client) emitTransportDown(err error) {
	c.mu.RLock()
	h := c.onTransportDown
	c.mu.RUnlock()
	if h != nil {
		h(err)
	}
}

func (c *Client) reportHandlerError(env Envelope, err error) {
	if err == nil {
		return
	}
	c.mu.RLock()
	h := c.onHandlerError
	c.mu.RUnlock()
	if h != nil {
		h(env, err)
	}
}

func (c *Client) subscriptionFilter(ch topic.Channel) string {
	mfr := c.cfg.Manufacturer
	sn := c.cfg.SerialNumber
	if mfr == "" {
		mfr = "+"
	}
	if sn == "" {
		sn = "+"
	}
	return c.topics.Build(mfr, sn, ch)
}

func (c *Client) requireStarted() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.started {
		return gerrors.ClientNotStarted
	}
	return nil
}
