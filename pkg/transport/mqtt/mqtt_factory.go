package mqtt

import (
	"fmt"
	"time"

	"github.com/kalifun/navlink/errors"
	"github.com/kalifun/navlink/pkg/core"
)

// TransportFactory creates MQTT transport instances
type TransportFactory struct{}

// NewTransportFactory creates a new MQTT transport factory
func NewTransportFactory() *TransportFactory {
	return &TransportFactory{}
}

// CreateTransport creates a new MQTT transport instance
func (f *TransportFactory) CreateTransport(id string, config map[string]interface{}) (core.Transport, error) {
	mqttConfig, err := parseMQTTConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MQTT config: %w", err)
	}

	return NewTransport(id, mqttConfig), nil
}

// parseMQTTConfig parses configuration map into MQTTConfig
func parseMQTTConfig(config map[string]interface{}) (MQTTConfig, error) {
	var mqttConfig MQTTConfig

	if broker, ok := config["broker"].(string); ok {
		mqttConfig.Broker = broker
	} else {
		return MQTTConfig{}, errors.ConfigurationErrorF.Args("broker is required and must be a string")
	}

	if clientID, ok := config["clientId"].(string); ok {
		mqttConfig.ClientID = clientID
	} else {
		return MQTTConfig{}, errors.ConfigurationErrorF.Args("clientId is required and must be a string")
	}

	if username, ok := config["username"].(string); ok {
		mqttConfig.Username = username
	}

	if password, ok := config["password"].(string); ok {
		mqttConfig.Password = password
	}

	if qos, ok := config["qos"].(int); ok {
		if qos >= 0 && qos <= 2 {
			mqttConfig.QoS = byte(qos)
		} else {
			return MQTTConfig{}, errors.ConfigurationErrorF.Args("QoS must be 0, 1, or 2")
		}
	} else {
		mqttConfig.QoS = 1 // Default QoS
	}

	if cleanSession, ok := config["cleanSession"].(bool); ok {
		mqttConfig.CleanSession = cleanSession
	} else {
		mqttConfig.CleanSession = true // Default clean session
	}

	if keepAlive, ok := config["keepAlive"].(int); ok && keepAlive > 0 {
		mqttConfig.KeepAlive = uint16(keepAlive)
	} else {
		mqttConfig.KeepAlive = 30 // Default 30 seconds
	}

	if connectTimeout, ok := config["connectTimeout"].(int); ok && connectTimeout > 0 {
		mqttConfig.ConnectTimeout = time.Duration(connectTimeout) * time.Second
	} else {
		mqttConfig.ConnectTimeout = 30 * time.Second // Default 30 seconds
	}

	if maxReconnectInterval, ok := config["maxReconnectInterval"].(int); ok && maxReconnectInterval > 0 {
		mqttConfig.MaxReconnectInterval = time.Duration(maxReconnectInterval) * time.Second
	} else {
		mqttConfig.MaxReconnectInterval = 10 * time.Minute // Default 10 minutes
	}

	if autoReconnect, ok := config["autoReconnect"].(bool); ok {
		mqttConfig.AutoReconnect = autoReconnect
	} else {
		mqttConfig.AutoReconnect = true // Default auto reconnect
	}

	// Parse TLS config if provided
	if tlsConfig, ok := config["tlsConfig"].(map[string]interface{}); ok {
		mqttConfig.TLSConfig = &TLSConfig{}
		if caFile, ok := tlsConfig["caFile"].(string); ok {
			mqttConfig.TLSConfig.CAFile = caFile
		}
		if certFile, ok := tlsConfig["certFile"].(string); ok {
			mqttConfig.TLSConfig.CertFile = certFile
		}
		if keyFile, ok := tlsConfig["keyFile"].(string); ok {
			mqttConfig.TLSConfig.KeyFile = keyFile
		}
		if insecure, ok := tlsConfig["insecure"].(bool); ok {
			mqttConfig.TLSConfig.Insecure = insecure
		}
	}

	// Parse will message if provided
	if willMessage, ok := config["willMessage"].(map[string]interface{}); ok {
		mqttConfig.WillMessage = &WillMessage{}
		if topic, ok := willMessage["topic"].(string); ok {
			mqttConfig.WillMessage.Topic = topic
		}
		if qos, ok := willMessage["qos"].(int); ok && qos >= 0 && qos <= 2 {
			mqttConfig.WillMessage.QoS = byte(qos)
		}
		if retained, ok := willMessage["retained"].(bool); ok {
			mqttConfig.WillMessage.Retained = retained
		}
		if payload, ok := willMessage["payload"].(string); ok {
			mqttConfig.WillMessage.Payload = payload
		}
	}

	return mqttConfig, nil
}
