package mqtt

import (
	"context"
	"sync"

	"github.com/kalifun/navlink/errors"
	"github.com/sirupsen/logrus"
)

// MQTTSubscription implements MQTT subscription with proper lifecycle management
type MQTTSubscription struct {
	topic     string
	transport *Transport
	mu        sync.RWMutex
	active    bool
}

func (ms *MQTTSubscription) Unsubscribe(ctx context.Context) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if !ms.active {
		return errors.SubscriptionNotActiveF.Args(ms.topic)
	}

	if ms.transport.client == nil || !ms.transport.client.IsConnected() {
		return errors.ClientNotConnected
	}

	ms.transport.logger.WithFields(logrus.Fields{
		"transport_id": ms.transport.id,
		"topic":        ms.topic,
	}).Info("Unsubscribing from MQTT topic")

	token := ms.transport.client.Unsubscribe(ms.topic)
	if token.Wait() && token.Error() != nil {
		ms.transport.logger.WithError(token.Error()).Error("MQTT unsubscribe failed")
		return errors.UnsubscribeFailedF.Args(ms.topic, token.Error())
	}

	ms.transport.removeSubscription(ms.topic)
	ms.active = false

	ms.transport.logger.WithFields(logrus.Fields{
		"transport_id": ms.transport.id,
		"topic":        ms.topic,
	}).Info("Successfully unsubscribed from MQTT topic")
	return nil
}

func (ms *MQTTSubscription) Topic() string {
	return ms.topic
}
