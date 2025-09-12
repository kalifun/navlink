package mqtt

import (
	"context"
	"fmt"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/kalifun/navlink/errors"
	"github.com/kalifun/navlink/pkg/core"
	"github.com/sirupsen/logrus"
)

type MqttConfig struct {
	Broker               string        `json:"broker" yaml:"broker"`
	ClientID             string        `json:"clientId" yaml:"clientId"`
	Username             string        `json:"username" yaml:"username"`
	Password             string        `json:"password" yaml:"password"`
	QoS                  byte          `json:"qos" yaml:"qos"`
	CleanSession         bool          `json:"cleanSession" yaml:"cleanSession"`
	KeepAlive            uint16        `json:"keepAlive" yaml:"keepAlive"`
	ConnectTimeout       time.Duration `json:"connectTimeout" yaml:"connectTimeout"`
	MaxReconnectInterval time.Duration `json:"maxReconnectInterval" yaml:"maxReconnectInterval"`
	AutoReconnect        bool          `json:"autoReconnect" yaml:"autoReconnect"`
	TLSConfig            *TLSConfig    `json:"tlsConfig,omitempty" yaml:"tlsConfig,omitempty"`
	WillMessage          *WillMessage  `json:"willMessage,omitempty" yaml:"willMessage,omitempty"`
}

type TLSConfig struct {
	CAFile   string `json:"caFile" yaml:"caFile"`
	CertFile string `json:"certFile" yaml:"certFile"`
	KeyFile  string `json:"keyFile" yaml:"keyFile"`
	Insecure bool   `json:"insecure" yaml:"insecure"`
}

type WillMessage struct {
	Topic    string `json:"topic" yaml:"topic"`
	QoS      byte   `json:"qos" yaml:"qos"`
	Retained bool   `json:"retained" yaml:"retained"`
	Payload  string `json:"payload" yaml:"payload"`
}

type MqttTransport struct {
	id             string
	config         MqttConfig
	client         mqtt.Client
	logger         *logrus.Logger
	mu             sync.RWMutex
	running        bool
	handlers       map[string]core.MessageHandler
	subscriptions  map[string]mqtt.Token
	connectChan    chan struct{}
	disconnectChan chan struct{}
	ctx            context.Context
	cancel         context.CancelFunc
}

func (mt *MqttTransport) ID() string {
	return mt.id
}

func (mt *MqttTransport) Start(ctx context.Context) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	if mt.running {
		return errors.MqttTransportAlreadyRunning
	}

	if err := mt.validateConfig(); err != nil {
		return err
	}

	mt.logger.WithFields(logrus.Fields{
		"transport_id": mt.id,
		"broker":       mt.config.Broker,
		"client_id":    mt.config.ClientID,
	}).Info("Starting MQTT transport")

	if err := mt.connect(); err != nil {
		return errors.ConnectionFailed
	}

	mt.running = true
	mt.logger.WithField("transport_id", mt.id).Info("MQTT transport started successfully")

	return nil
}

func (mt *MqttTransport) Stop(ctx context.Context) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	if !mt.running {
		return errors.MqttTransportNotRunning
	}

	mt.logger.WithField("transport_id", mt.id).Info("Stopping MQTT transport")

	mt.cancel()

	if mt.client != nil && mt.client.IsConnected() {
		mt.client.Disconnect(250)
	}

	mt.running = false
	close(mt.disconnectChan)
	mt.logger.WithField("transport_id", mt.id).Info("MQTT transport stopped successfully")
	return nil
}

func (mt *MqttTransport) Publish(ctx context.Context, topic string, payload []byte, opts core.PublishOptions) error {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	if !mt.running {
		return errors.MqttTransportNotRunning
	}

	if mt.client == nil || !mt.client.IsConnected() {
		return errors.ConnectionFailed
	}

	mt.logger.WithFields(logrus.Fields{
		"transport_id": mt.id,
		"topic":        topic,
		"payload_size": len(payload),
		"qos":          opts.QoS,
		"retain":       opts.Retain,
	}).Debug("Publishing MQTT message")

	token := mt.client.Publish(topic, opts.QoS, opts.Retain, payload)
	if opts.TimeOut > 0 {
		select {
		case <-token.Done():
			if token.Error() != nil {
				mt.logger.WithError(token.Error()).Error("MQTT publish failed")
				return errors.PublishFailed
			}
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(opts.TimeOut):
			return errors.TimeoutError
		}
	} else {
		if err := token.Error(); err != nil {
			mt.logger.WithError(err).Error("MQTT publish failed")
			return errors.PublishFailed
		}
	}

	mt.logger.WithFields(logrus.Fields{
		"transport_id": mt.id,
		"topic":        topic,
	}).Debug("MQTT message published successfully")
	return nil
}

func (mt *MqttTransport) Subscribe(ctx context.Context, topic string, handler core.MessageHandler) (core.Subscription, error) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	if !mt.running {
		return nil, errors.MqttTransportNotRunning
	}

	if mt.client == nil || !mt.client.IsConnected() {
		return nil, errors.ConnectionFailed
	}

	mt.logger.WithFields(logrus.Fields{
		"transport_id": mt.id,
		"topic":        topic,
		"qos":          mt.config.QoS,
	}).Info("Subscribing to MQTT topic")

	token := mt.client.Subscribe(topic, mt.config.QoS, func(c mqtt.Client, m mqtt.Message) {
		mt.handleMessage(m, handler)
	})

	if err := token.Error(); err != nil {
		mt.logger.WithError(token.Error()).Error("MQTT subscription failed")
		return nil, errors.SubscriptionFailed
	}

	mt.handlers[topic] = handler
	mt.subscriptions[topic] = token

	sub := &MQTTSubscription{
		topic:     topic,
		transport: mt,
	}

	mt.logger.WithFields(logrus.Fields{
		"transport_id": mt.id,
		"topic":        topic,
	}).Info("Successfully subscribed to MQTT topic")
	return sub, nil
}

// validateConfig validates MQTT configuration
func (mt *MqttTransport) validateConfig() error {
	if mt.config.Broker == "" {
		return errors.ConfigurationError.Args("broker URL is required")
	}

	if mt.config.ClientID == "" {
		return errors.ConfigurationError.Args("client ID is required")
	}

	if mt.config.QoS > 2 {
		return errors.ConfigurationError.Args("QoS must be 0, 1, or 2")
	}

	if mt.config.ConnectTimeout <= 0 {
		mt.config.ConnectTimeout = 30 * time.Second
	}

	if mt.config.MaxReconnectInterval <= 0 {
		mt.config.MaxReconnectInterval = 30 * time.Minute
	}
	return nil
}

func (mt *MqttTransport) connect() error {
	mqttOpts := mqtt.NewClientOptions()
	mqttOpts.AddBroker(mt.config.Broker)
	mqttOpts.SetClientID(mt.config.ClientID)
	mqttOpts.SetUsername(mt.config.Username)
	mqttOpts.SetPassword(mt.config.Password)
	mqttOpts.SetCleanSession(mt.config.CleanSession)
	mqttOpts.SetKeepAlive(time.Duration(mt.config.KeepAlive) * time.Second)
	mqttOpts.SetAutoReconnect(mt.config.AutoReconnect)
	mqttOpts.SetMaxReconnectInterval(mt.config.MaxReconnectInterval)

	if mt.config.WillMessage != nil {
		mqttOpts.SetWill(mt.config.WillMessage.Topic,
			mt.config.WillMessage.Payload,
			mt.config.WillMessage.QoS,
			mt.config.WillMessage.Retained)
	}

	mqttOpts.OnConnect = mt.onConnect
	mqttOpts.OnReconnecting = mt.onReconnecting
	mqttOpts.OnConnectionLost = mt.onConnectionLost

	mt.client = mqtt.NewClient(mqttOpts)
	connectToken := mt.client.Connect()
	if connectToken.WaitTimeout(mt.config.ConnectTimeout) {
		if connectToken.Error() != nil {
			return connectToken.Error()
		}
		return nil
	}

	return errors.ConnectionFailed
}

func (mt *MqttTransport) onConnect(client mqtt.Client) {
	mt.logger.WithFields(logrus.Fields{
		"transport_id": mt.id,
		"broker":       mt.config.Broker,
	}).Info("MQTT connection established")
	select {
	case mt.connectChan <- struct{}{}:
	default:
	}
}

func (mt *MqttTransport) onConnectionLost(client mqtt.Client, err error) {
	mt.logger.WithFields(logrus.Fields{
		"transport_id": mt.id,
		"error":        err.Error(),
	}).Error("MQTT connection lost")

	select {
	case mt.disconnectChan <- struct{}{}:
	default:
	}
}

func (mt *MqttTransport) onReconnecting(client mqtt.Client, opts *mqtt.ClientOptions) {
	mt.logger.WithFields(logrus.Fields{
		"transport_id": mt.id,
		"broker":       mt.config.Broker,
	}).Info("Attempting to reconnect to MQTT broker")
}

// handleMessage processes incoming MQTT messages
func (mt *MqttTransport) handleMessage(msg mqtt.Message, handler core.MessageHandler) {
	mt.logger.WithFields(logrus.Fields{
		"transport_id": mt.id,
		"topic":        msg.Topic(),
		"payload_size": len(msg.Payload()),
		"qos":          msg.Qos(),
		"retained":     msg.Retained(),
	}).Debug("Received MQTT message")

	message := &core.Message{
		Topic:  msg.Topic(),
		Paylod: msg.Payload(),
		Meta: map[string]string{
			"source":     "mqtt",
			"qos":        fmt.Sprintf("%d", msg.Qos()),
			"retained":   fmt.Sprintf("%t", msg.Retained()),
			"message_id": fmt.Sprintf("%d", msg.MessageID()),
		},
		Time: time.Now(),
	}

	go func() {
		if err := handler(mt.ctx, message); err != nil {
			mt.logger.WithFields(logrus.Fields{
				"transport_id": mt.id,
				"topic":        msg.Topic(),
				"error":        err.Error(),
			}).Error("MQTT message handler error")
		}

		msg.Ack()
	}()
}

// removeSubscription removes a subscription from internal tracking
func (mt *MqttTransport) removeSubscription(topic string) {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	delete(mt.handlers, topic)
	delete(mt.subscriptions, topic)
}
