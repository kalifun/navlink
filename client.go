package navlink

import (
	"context"
	"sync"

	"time"

	"github.com/kalifun/navlink/gerrors"
	"github.com/kalifun/navlink/mqtt"
	"github.com/kalifun/navlink/topic"
)

// Client is the VDA5050 protocol access entrypoint.
type Client struct {
	cfg       Config
	transport Transport
	topics    topic.Resolver

	mu      sync.RWMutex
	started bool
	unsubs  []Unsubscribe

	stateHandlers []StateHandler
	connHandlers  []ConnectionHandler
	vizHandlers   []VisualizationHandler
	fsHandlers    []FactsheetHandler
	topicHandlers []topicSub
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
	}

	if cfg.Transport != nil {
		c.transport = cfg.Transport
	} else {
		keepAlive := cfg.KeepAlive
		if keepAlive == 0 {
			keepAlive = 30 * time.Second
		}
		c.transport = newMQTTTransport(mqtt.Config{
			Broker:        cfg.Broker,
			ClientID:      cfg.ClientID,
			Username:      cfg.Username,
			Password:      cfg.Password,
			QoS:           cfg.qos(),
			KeepAlive:     keepAlive,
			CleanSession:  cfg.CleanSession,
			AutoReconnect: cfg.AutoReconnect,
		})
	}

	return c, nil
}

// Topics returns the TopicResolver bound to this client.
func (c *Client) Topics() topic.Resolver { return c.topics }

// OnState registers a typed state handler (may be called before Start).
func (c *Client) OnState(h StateHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stateHandlers = append(c.stateHandlers, h)
}

// OnConnection registers a typed connection handler.
func (c *Client) OnConnection(h ConnectionHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connHandlers = append(c.connHandlers, h)
}

// OnVisualization registers a typed visualization handler.
func (c *Client) OnVisualization(h VisualizationHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.vizHandlers = append(c.vizHandlers, h)
}

// OnFactsheet registers a typed factsheet handler.
func (c *Client) OnFactsheet(h FactsheetHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fsHandlers = append(c.fsHandlers, h)
}

// OnTopic registers a raw/escape-hatch topic filter handler.
func (c *Client) OnTopic(filter string, h TopicHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.topicHandlers = append(c.topicHandlers, topicSub{filter: filter, handler: h})
}

// AGV returns a per-vehicle outbound handle.
func (c *Client) AGV(manufacturer, serial string) *AGVHandle {
	return &AGVHandle{client: c, manufacturer: manufacturer, serial: serial}
}

// Start connects the transport and establishes subscriptions for registered handlers.
func (c *Client) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return gerrors.ClientAlreadyStarted
	}
	if err := c.transport.Start(ctx); err != nil {
		return err
	}

	if err := c.subscribeLocked(ctx); err != nil {
		_ = c.transport.Stop(ctx)
		return err
	}

	c.started = true
	return nil
}

// Stop cancels subscriptions and disconnects the transport.
func (c *Client) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return nil
	}

	var firstErr error
	for i := len(c.unsubs) - 1; i >= 0; i-- {
		if err := c.unsubs[i](ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	c.unsubs = nil

	if err := c.transport.Stop(ctx); err != nil && firstErr == nil {
		firstErr = err
	}
	c.started = false
	return firstErr
}

func (c *Client) subscribeLocked(ctx context.Context) error {
	channels := make([]topic.Channel, 0, 4)
	if len(c.stateHandlers) > 0 {
		channels = append(channels, topic.ChannelState)
	}
	if len(c.connHandlers) > 0 {
		channels = append(channels, topic.ChannelConnection)
	}
	if len(c.vizHandlers) > 0 {
		channels = append(channels, topic.ChannelVisualization)
	}
	if len(c.fsHandlers) > 0 {
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
