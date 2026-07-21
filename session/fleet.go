package session

import (
	"context"
	"sync"

	"github.com/kalifun/vda5050-types-go/connection"

	"github.com/kalifun/navlink/topic"
)

// AGV is the protocol identity used by FleetSession.
type AGV struct {
	Manufacturer string
	SerialNumber string
}

func (a AGV) key() string { return a.Manufacturer + "/" + a.SerialNumber }

// Options configures fleet-level subscription behaviour.
type Options struct {
	// SubscribeState subscribes per-AGV state topics when tracked (default true).
	SubscribeState bool
	// SubscribeVisualization subscribes per-AGV visualization when tracked.
	SubscribeVisualization bool
	// AutoTrackFromConnection tracks on ONLINE and untracks on OFFLINE/CONNECTIONBROKEN.
	AutoTrackFromConnection bool
}

// DefaultOptions returns the recommended fleet defaults.
func DefaultOptions() Options {
	return Options{
		SubscribeState:          true,
		SubscribeVisualization:  false,
		AutoTrackFromConnection: true,
	}
}

// SubscribeFunc subscribes a topic filter using the client's inbound dispatcher.
type SubscribeFunc func(ctx context.Context, filter string) (Unsubscribe, error)

// Unsubscribe cancels a subscription.
type Unsubscribe func(ctx context.Context) error

// OnlineHandler is called when an AGV becomes tracked/online.
type OnlineHandler func(agv AGV)

// OfflineHandler is called when an AGV is untracked/offline.
type OfflineHandler func(agv AGV)

// FleetSession manages connection wildcard subscription and per-AGV state/viz
// subscriptions.
//
// Common traps covered here:
//   - Retained connection: subscribe connection once at Start; ONLINE triggers Track.
//   - Reconnect / CleanSession: call Restore to re-subscribe connection + tracked AGVs
//     without emitting duplicate online/offline hooks.
type FleetSession struct {
	topics    topic.Resolver
	subscribe SubscribeFunc
	opts      Options

	mu        sync.Mutex
	started   bool
	connUnsub Unsubscribe
	tracked   map[string]*trackedAGV
	onOnline  []OnlineHandler
	onOffline []OfflineHandler
}

type trackedAGV struct {
	agv    AGV
	unsubs []Unsubscribe
}

// NewFleetSession creates a fleet session (does not subscribe until Start).
func NewFleetSession(topics topic.Resolver, subscribe SubscribeFunc, opts Options) *FleetSession {
	return &FleetSession{
		topics:    topics,
		subscribe: subscribe,
		opts:      opts,
		tracked:   make(map[string]*trackedAGV),
	}
}

// OnAGVOnline registers a hook invoked after an AGV is tracked.
func (f *FleetSession) OnAGVOnline(h OnlineHandler) {
	if h == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.onOnline = append(f.onOnline, h)
}

// OnAGVOffline registers a hook invoked after an AGV is untracked.
func (f *FleetSession) OnAGVOffline(h OfflineHandler) {
	if h == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.onOffline = append(f.onOffline, h)
}

// Start subscribes the fleet connection wildcard once.
func (f *FleetSession) Start(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.started {
		return nil
	}
	unsub, err := f.subscribe(ctx, f.topics.Wildcard(topic.ChannelConnection))
	if err != nil {
		return err
	}
	f.connUnsub = unsub
	f.started = true
	return nil
}

// Stop untracks all AGVs and drops the connection subscription.
func (f *FleetSession) Stop(ctx context.Context) error {
	f.mu.Lock()
	agvs := make([]AGV, 0, len(f.tracked))
	for _, e := range f.tracked {
		agvs = append(agvs, e.agv)
	}
	f.mu.Unlock()

	var firstErr error
	for _, agv := range agvs {
		if err := f.Untrack(ctx, agv); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if f.connUnsub != nil {
		if err := f.connUnsub(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
		f.connUnsub = nil
	}
	f.started = false
	return firstErr
}

// Track subscribes per-AGV channels according to Options.
func (f *FleetSession) Track(ctx context.Context, agv AGV) error {
	return f.track(ctx, agv, true)
}

// Untrack removes per-AGV subscriptions.
func (f *FleetSession) Untrack(ctx context.Context, agv AGV) error {
	return f.untrack(ctx, agv, true)
}

// HandleConnection drives Track/Untrack from connection state when enabled.
func (f *FleetSession) HandleConnection(ctx context.Context, agv AGV, state connection.ConnectionState) error {
	if !f.opts.AutoTrackFromConnection {
		return nil
	}
	switch state {
	case connection.Online:
		return f.Track(ctx, agv)
	case connection.Offline, connection.ConnectionBroken:
		return f.Untrack(ctx, agv)
	default:
		return nil
	}
}

// Restore re-subscribes connection and all tracked AGVs after reconnect.
// Online/offline hooks are not fired again.
func (f *FleetSession) Restore(ctx context.Context) error {
	f.mu.Lock()
	if !f.started && len(f.tracked) == 0 {
		f.mu.Unlock()
		return nil
	}
	agvs := make([]AGV, 0, len(f.tracked))
	for _, e := range f.tracked {
		agvs = append(agvs, e.agv)
		for i := len(e.unsubs) - 1; i >= 0; i-- {
			_ = e.unsubs[i](ctx)
		}
	}
	f.tracked = make(map[string]*trackedAGV)
	if f.connUnsub != nil {
		_ = f.connUnsub(ctx)
		f.connUnsub = nil
	}
	f.started = false
	f.mu.Unlock()

	if err := f.Start(ctx); err != nil {
		return err
	}
	for _, agv := range agvs {
		if err := f.track(ctx, agv, false); err != nil {
			return err
		}
	}
	return nil
}

// Tracked returns a snapshot of currently tracked AGVs.
func (f *FleetSession) Tracked() []AGV {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]AGV, 0, len(f.tracked))
	for _, e := range f.tracked {
		out = append(out, e.agv)
	}
	return out
}

func (f *FleetSession) track(ctx context.Context, agv AGV, notify bool) error {
	f.mu.Lock()
	if _, exists := f.tracked[agv.key()]; exists {
		f.mu.Unlock()
		return nil
	}
	f.mu.Unlock()

	var unsubs []Unsubscribe
	for _, ch := range f.channels() {
		filter := f.topics.Build(agv.Manufacturer, agv.SerialNumber, ch)
		unsub, err := f.subscribe(ctx, filter)
		if err != nil {
			for i := len(unsubs) - 1; i >= 0; i-- {
				_ = unsubs[i](ctx)
			}
			return err
		}
		unsubs = append(unsubs, unsub)
	}

	f.mu.Lock()
	if _, exists := f.tracked[agv.key()]; exists {
		f.mu.Unlock()
		for i := len(unsubs) - 1; i >= 0; i-- {
			_ = unsubs[i](ctx)
		}
		return nil
	}
	f.tracked[agv.key()] = &trackedAGV{agv: agv, unsubs: unsubs}
	var hooks []OnlineHandler
	if notify {
		hooks = append([]OnlineHandler(nil), f.onOnline...)
	}
	f.mu.Unlock()

	for _, h := range hooks {
		h(agv)
	}
	return nil
}

func (f *FleetSession) untrack(ctx context.Context, agv AGV, notify bool) error {
	f.mu.Lock()
	entry, ok := f.tracked[agv.key()]
	if !ok {
		f.mu.Unlock()
		return nil
	}
	delete(f.tracked, agv.key())
	unsubs := entry.unsubs
	var hooks []OfflineHandler
	if notify {
		hooks = append([]OfflineHandler(nil), f.onOffline...)
	}
	f.mu.Unlock()

	var firstErr error
	for i := len(unsubs) - 1; i >= 0; i-- {
		if err := unsubs[i](ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	for _, h := range hooks {
		h(agv)
	}
	return firstErr
}

func (f *FleetSession) channels() []topic.Channel {
	var ch []topic.Channel
	if f.opts.SubscribeState {
		ch = append(ch, topic.ChannelState)
	}
	if f.opts.SubscribeVisualization {
		ch = append(ch, topic.ChannelVisualization)
	}
	return ch
}
