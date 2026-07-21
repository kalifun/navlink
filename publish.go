package navlink

import (
	"context"
	"errors"

	"github.com/kalifun/navlink/gerrors"
	"github.com/kalifun/navlink/topic"
)

// PublishResult is a summary of what was actually handed to Transport.Publish.
// It is for reconciliation / logging only: navlink does not allocate IDs and does
// not RecordSuccessfulPublish — the orchestration layer decides that from err.
type PublishResult struct {
	Topic          string
	Channel        topic.Channel
	Manufacturer   string
	SerialNumber   string
	QoS            byte
	Payload        []byte
	HeaderID       uint32
	OrderID        string
	OrderUpdateID  uint32
	ActionIDs      []string
}

// PublishAccepted reports whether the MQTT broker accepted the publish for the
// configured QoS (token wait completed without error). Equivalent to err == nil.
// This is not vehicle-side order acceptance — that still comes from inbound state.
// Platforms should call RecordSuccessfulPublish only when this is true.
func PublishAccepted(err error) bool {
	return err == nil
}

// IsPublishNotStarted reports ClientNotStarted (or transport not running).
func IsPublishNotStarted(err error) bool {
	return errors.Is(err, gerrors.ClientNotStarted) ||
		errors.Is(err, gerrors.MQTTTransportNotRunning)
}

// IsPublishTimeout reports a publish wait timeout or context deadline.
// context.Canceled is not a timeout; use IsPublishCanceled.
func IsPublishTimeout(err error) bool {
	return errors.Is(err, gerrors.TimeoutError) ||
		errors.Is(err, context.DeadlineExceeded)
}

// IsPublishCanceled reports context cancellation (not a timeout).
func IsPublishCanceled(err error) bool {
	return errors.Is(err, context.Canceled)
}

// IsPublishQoSRejected reports an unsupported QoS level.
func IsPublishQoSRejected(err error) bool {
	return errors.Is(err, gerrors.QosNotSupported)
}

// IsPublishBrokerRejected reports broker/token rejection after the wait completed.
func IsPublishBrokerRejected(err error) bool {
	return errors.Is(err, gerrors.PublishFailed)
}

func validatePublishQoS(qos byte) error {
	if qos > 2 {
		return gerrors.QosNotSupported.With("qos", qos)
	}
	return nil
}
