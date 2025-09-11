package core

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SessionMeta represents session metadata
type SessionMeta struct {
	ID          string            `json:"id"`
	TransportID string            `json:"transportId"`
	CreatedAt   time.Time         `json:"createdAt"`
	LastSeen    time.Time         `json:"lastSeen"`
	Metadata    map[string]string `json:"metadata"`
}

type SubscriptionMeta struct {
	Topic      string    `json:"topic"`
	QoS        byte      `json:"qos"`
	Subscribed time.Time `json:"subscribed"`
}

// InMemorySessionStore implements SessionStore using in-memory storage
type InMemorySessionStore struct {
	sessions      map[string]SessionMeta
	subscriptions map[string]map[string]SubscriptionMeta // session -> topic -> subscription
	mu            sync.RWMutex
}

func (ss *InMemorySessionStore) SaveSession(ctx context.Context, id string, meta SessionMeta) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.sessions[id] = meta
	return nil
}

func (ss *InMemorySessionStore) GetSession(ctx context.Context, id string) (SessionMeta, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	meta, exists := ss.sessions[id]
	if !exists {
		return SessionMeta{}, fmt.Errorf("session not found: %s", id)
	}

	return meta, nil
}

func (ss *InMemorySessionStore) DeleteSession(ctx context.Context, id string) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	delete(ss.sessions, id)
	delete(ss.subscriptions, id)
	return nil
}

func (ss *InMemorySessionStore) ListSessions(ctx context.Context) ([]SessionMeta, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	metas := make([]SessionMeta, 0, len(ss.sessions))

	for _, meta := range ss.sessions {
		metas = append(metas, meta)
	}

	return metas, nil
}

func (ss *InMemorySessionStore) AddSubscription(ctx context.Context, sessionID string, sub SubscriptionMeta) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if _, exists := ss.sessions[sessionID]; !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if ss.subscriptions[sessionID] == nil {
		ss.subscriptions[sessionID] = make(map[string]SubscriptionMeta)
	}

	ss.subscriptions[sessionID][sub.Topic] = sub
	return nil
}

func (ss *InMemorySessionStore) RemoveSubscription(ctx context.Context, sessionID string, topic string) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if subs, exists := ss.subscriptions[sessionID]; exists {
		delete(subs, topic)
	}

	return nil
}

func (ss *InMemorySessionStore) ListSubscriptions(ctx context.Context, sessionID string) ([]SubscriptionMeta, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	if _, exists := ss.sessions[sessionID]; !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	subs := ss.subscriptions[sessionID]
	subscriptions := make([]SubscriptionMeta, 0, len(subs))
	for _, sub := range subs {
		subscriptions = append(subscriptions, sub)
	}
	return subscriptions, nil
}
