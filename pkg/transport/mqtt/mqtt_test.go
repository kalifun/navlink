package mqtt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTransportCreation tests creating a new MQTT transport
func TestTransportCreation(t *testing.T) {
	config := MQTTConfig{
		Broker:       "tcp://localhost:1883",
		ClientID:     "test-client",
		Username:     "test-user",
		Password:     "test-pass",
		QoS:          1,
		CleanSession: true,
		KeepAlive:    30,
	}

	transport := NewTransport("test-transport", config)

	assert.NotNil(t, transport)
	assert.Equal(t, "test-transport", transport.ID())
	assert.Equal(t, config, transport.GetConfig())
	assert.False(t, transport.IsRunning())
}

// TestTransportConfigValidation tests configuration validation
func TestTransportConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  MQTTConfig
		wantErr bool
	}{
		{
			name: "Valid config",
			config: MQTTConfig{
				Broker:   "tcp://localhost:1883",
				ClientID: "test-client",
				QoS:      1,
			},
			wantErr: false,
		},
		{
			name: "Missing broker",
			config: MQTTConfig{
				ClientID: "test-client",
				QoS:      1,
			},
			wantErr: true,
		},
		{
			name: "Missing client ID",
			config: MQTTConfig{
				Broker: "tcp://localhost:1883",
				QoS:    1,
			},
			wantErr: true,
		},
		{
			name: "Invalid QoS",
			config: MQTTConfig{
				Broker:   "tcp://localhost:1883",
				ClientID: "test-client",
				QoS:      3,
			},
			wantErr: true,
		},
		{
			name: "Valid QoS values",
			config: MQTTConfig{
				Broker:   "tcp://localhost:1883",
				ClientID: "test-client",
				QoS:      2,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := NewTransport("test", tt.config)
			err := transport.validateConfig()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestMQTTTransportFactory tests the MQTT transport factory
func TestMQTTTransportFactory(t *testing.T) {
	factory := NewTransportFactory()
	assert.NotNil(t, factory)

	config := map[string]interface{}{
		"broker":       "tcp://localhost:1883",
		"clientId":     "factory-client",
		"username":     "factory-user",
		"password":     "factory-pass",
		"qos":          1,
		"cleanSession": true,
		"keepAlive":    30,
	}

	transport, err := factory.CreateTransport("factory-transport", config)
	assert.NoError(t, err)
	assert.NotNil(t, transport)
	assert.Equal(t, "factory-transport", transport.ID())
}
