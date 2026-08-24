package testkit

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/kalifun/navlink"
	"github.com/kalifun/navlink/internal/gerrors"
)

// PublishedMessage is one outbound message captured by FakeBroker.
type PublishedMessage struct {
	Topic   string
	Payload []byte
	Opts    navlink.PublishOptions
}

// FakeBroker is an in-process MQTT stand-in for unit tests.
// It implements navlink.Transport so Client tests need no paho / real broker.
type FakeBroker struct {
	mu          sync.Mutex
	running     bool
	nextID      int
	subs        map[int]sub
	published   []PublishedMessage
	fail        error
	hangFor     time.Duration
	hang        bool
	onReconnect func()
	onLost      func(error)
}

type sub struct {
	filter  string
	handler navlink.RawHandler
}

// NewFakeBroker creates a stopped broker.
func NewFakeBroker() *FakeBroker {
	return &FakeBroker{subs: make(map[int]sub)}
}

// Start marks the broker running.
func (b *FakeBroker) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.running = true
	if b.subs == nil {
		b.subs = make(map[int]sub)
	}
	return nil
}

// Stop clears subscriptions. Published history is kept for assertions until ClearPublished.
func (b *FakeBroker) Stop(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.running = false
	b.subs = make(map[int]sub)
	return nil
}

// FailNextPublish makes the next Publish return err without delivering.
// ClassifyPublish treats this as DefinitelyNotStarted (not in-flight).
func (b *FakeBroker) FailNextPublish(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.fail = err
	b.hang = false
}

// HangNextPublish makes the next Publish block for d then return a timeout
// marked as attempted (Uncertain). d <= 0 waits until ctx is done.
func (b *FakeBroker) HangNextPublish(d time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.hang = true
	b.hangFor = d
	b.fail = nil
}

// Connected implements navlink.ConnectionStatus.
func (b *FakeBroker) Connected() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.running
}

// SetOnConnectionLost implements navlink.ConnectionLostAware.
func (b *FakeBroker) SetOnConnectionLost(fn func(error)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.onLost = fn
}

// SimulateDisconnect fires the connection-lost callback without clearing subs.
func (b *FakeBroker) SimulateDisconnect(err error) {
	b.mu.Lock()
	handler := b.onLost
	b.mu.Unlock()
	if handler != nil {
		handler(err)
	}
}

// Publish records the message and delivers it to matching subscribers (loopback).
func (b *FakeBroker) Publish(ctx context.Context, topic string, payload []byte, opts navlink.PublishOptions) error {
	b.mu.Lock()
	if !b.running {
		b.mu.Unlock()
		return errNotRunning
	}
	fail := b.fail
	hang := b.hang
	hangFor := b.hangFor
	b.fail = nil
	b.hang = false
	b.mu.Unlock()

	if fail != nil {
		return fail
	}
	if hang {
		if hangFor <= 0 {
			<-ctx.Done()
			return navlink.MarkPublishAttempted(ctx.Err())
		}
		timer := time.NewTimer(hangFor)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return navlink.MarkPublishAttempted(ctx.Err())
		case <-timer.C:
			return navlink.MarkPublishAttempted(gerrors.TimeoutError)
		}
	}

	b.mu.Lock()
	cp := append([]byte(nil), payload...)
	b.published = append(b.published, PublishedMessage{Topic: topic, Payload: cp, Opts: opts})
	subs := make([]sub, 0, len(b.subs))
	for _, s := range b.subs {
		subs = append(subs, s)
	}
	b.mu.Unlock()

	for _, s := range subs {
		if matchFilter(s.filter, topic) {
			if err := s.handler(ctx, topic, payload); err != nil {
				return err
			}
		}
	}
	return nil
}

// Subscribe registers a handler for a topic filter.
func (b *FakeBroker) Subscribe(ctx context.Context, filter string, handler navlink.RawHandler) (navlink.Unsubscribe, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.running {
		return nil, errNotRunning
	}
	b.nextID++
	id := b.nextID
	b.subs[id] = sub{filter: filter, handler: handler}
	return func(ctx context.Context) error {
		b.mu.Lock()
		defer b.mu.Unlock()
		delete(b.subs, id)
		return nil
	}, nil
}

// Inject delivers an inbound message to matching subscribers without recording
// it as a client Publish. Use this to simulate AGV → master traffic.
func (b *FakeBroker) Inject(ctx context.Context, topic string, payload []byte) error {
	b.mu.Lock()
	if !b.running {
		b.mu.Unlock()
		return errNotRunning
	}
	subs := make([]sub, 0, len(b.subs))
	for _, s := range b.subs {
		subs = append(subs, s)
	}
	b.mu.Unlock()

	for _, s := range subs {
		if matchFilter(s.filter, topic) {
			if err := s.handler(ctx, topic, payload); err != nil {
				return err
			}
		}
	}
	return nil
}

// Published returns a copy of outbound messages.
func (b *FakeBroker) Published() []PublishedMessage {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]PublishedMessage, len(b.published))
	copy(out, b.published)
	return out
}

// ClearPublished resets the outbound log.
func (b *FakeBroker) ClearPublished() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.published = nil
}

// SetOnReconnect implements navlink.ReconnectAware.
func (b *FakeBroker) SetOnReconnect(fn func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.onReconnect = fn
}

// SimulateReconnect clears subscriptions (as CleanSession reconnect would) and
// invokes the reconnect handler so Client can restore VDA subscriptions.
func (b *FakeBroker) SimulateReconnect() {
	b.mu.Lock()
	b.subs = make(map[int]sub)
	handler := b.onReconnect
	b.mu.Unlock()
	if handler != nil {
		handler()
	}
}

// Filters returns current subscription filters (for assertions).
func (b *FakeBroker) Filters() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]string, 0, len(b.subs))
	for _, s := range b.subs {
		out = append(out, s.filter)
	}
	return out
}

func matchFilter(filter, topic string) bool {
	fp := strings.Split(filter, "/")
	tp := strings.Split(topic, "/")
	if len(fp) != len(tp) {
		return false
	}
	for i := range fp {
		if fp[i] == "+" || fp[i] == "#" {
			continue
		}
		if fp[i] != tp[i] {
			return false
		}
	}
	return true
}

var errNotRunning = errString("testkit: fake broker is not running")

type errString string

func (e errString) Error() string { return string(e) }
