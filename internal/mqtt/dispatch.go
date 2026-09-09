package mqtt

import (
	"context"
	"time"
)

// DropReason explains an OnInboundDrop callback.
type DropReason int

const (
	// DropDiscarded: message was dropped (visualization under backpressure).
	DropDiscarded DropReason = iota
	// DropBackpressured: queue full; enqueue will block until space or stop.
	DropBackpressured
)

func (t *Transport) startDispatchLocked() {
	qsize := t.cfg.InboundQueueSize
	if qsize <= 0 {
		qsize = defaultInboundQueue
	}
	t.qsize = qsize
	dispatchCtx, cancel := context.WithCancel(context.Background())
	t.dispatchCtx = dispatchCtx
	t.dispatchCancel = cancel
	t.connLatest = make(map[string]inbound)
	t.connWake = make(chan struct{}, 1)
	t.topicQ = make(chan inbound, qsize)
	t.shards = make(map[string]chan inbound)

	t.dispatchWG.Go(func() { t.runConnectionCoalesce(dispatchCtx) })
	t.dispatchWG.Go(func() { t.runQueue(dispatchCtx, t.topicQ) })
}

func (t *Transport) runQueue(ctx context.Context, ch <-chan inbound) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			t.dispatchOne(ctx, msg)
		}
	}
}

// runConnectionCoalesce delivers at most one pending connection per AGV;
// newer connection states overwrite older ones still waiting (latest wins).
func (t *Transport) runConnectionCoalesce(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.connWake:
			for {
				msg, ok := t.takeConn()
				if !ok {
					break
				}
				t.dispatchOne(ctx, msg)
			}
		}
	}
}

func (t *Transport) dispatchOne(ctx context.Context, msg inbound) {
	err := msg.handler(ctx, msg.topic, msg.payload, msg.receivedAt)
	if err == nil {
		return
	}
	t.mu.RLock()
	h := t.cfg.OnHandlerError
	t.mu.RUnlock()
	if h != nil {
		h(msg.topic, err)
	}
}

func (t *Transport) takeConn() (inbound, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for k, v := range t.connLatest {
		delete(t.connLatest, k)
		return v, true
	}
	return inbound{}, false
}

func (t *Transport) enqueueConnection(key string, msg inbound) {
	if key == "" {
		key = msg.topic
	}
	t.mu.Lock()
	if t.connLatest == nil {
		t.mu.Unlock()
		return
	}
	t.connLatest[key] = msg
	wake := t.connWake
	t.mu.Unlock()
	if wake == nil {
		return
	}
	select {
	case wake <- struct{}{}:
	default:
	}
}

func (t *Transport) queueFor(topic string) chan inbound {
	lane, key := classifyInbound(topic)
	switch lane {
	case laneConnection:
		return nil // handled by enqueueConnection
	case laneTopic:
		t.mu.RLock()
		ch := t.topicQ
		t.mu.RUnlock()
		return ch
	default:
		return t.agvQueue(key)
	}
}

func (t *Transport) agvQueue(key string) chan inbound {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.shards == nil || t.dispatchCtx == nil {
		return nil
	}
	if ch, ok := t.shards[key]; ok {
		return ch
	}
	ch := make(chan inbound, t.qsize)
	t.shards[key] = ch
	ctx := t.dispatchCtx
	t.dispatchWG.Go(func() { t.runQueue(ctx, ch) })
	return ch
}

func (t *Transport) enqueue(topic string, payload []byte, handler Handler) {
	msg := inbound{
		topic:      topic,
		payload:    payload,
		handler:    handler,
		receivedAt: time.Now().UTC(),
	}

	lane, key := classifyInbound(topic)
	if lane == laneConnection {
		t.enqueueConnection(key, msg)
		return
	}

	t.mu.RLock()
	drop := t.cfg.OnInboundDrop
	done := ctxDone(t.dispatchCtx)
	t.mu.RUnlock()

	ch := t.queueFor(topic)
	if ch == nil {
		return
	}
	select {
	case ch <- msg:
		return
	default:
	}

	reason := DropBackpressured
	if droppableTopic(topic) {
		reason = DropDiscarded
	}
	if drop != nil {
		drop(topic, reason)
	}
	if reason == DropDiscarded {
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
