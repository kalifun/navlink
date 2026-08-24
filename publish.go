package navlink

import (
	"context"
	"errors"

	"github.com/kalifun/navlink/internal/gerrors"
	"github.com/kalifun/navlink/topic"
)

// PublishOutcome is the execution-layer three-way result of a publish.
// It answers whether the caller-supplied protocol IDs may be reused.
type PublishOutcome int

const (
	// PublishOutcomeAccepted: broker accepted the publish (QoS handshake).
	// IDs are consumed; platforms may RecordSuccessfulPublish.
	PublishOutcomeAccepted PublishOutcome = iota
	// PublishOutcomeNotStarted: the publish never became a deliverable MQTT packet.
	// IDs may be returned to the pool and reused.
	PublishOutcomeNotStarted
	// PublishOutcomeUncertain: the packet may already be in flight (typical: PUBACK timeout).
	// IDs must not be reused. Fencing / recovery is orchestration.
	PublishOutcomeUncertain
)

func (o PublishOutcome) String() string {
	switch o {
	case PublishOutcomeAccepted:
		return "accepted"
	case PublishOutcomeNotStarted:
		return "not_started"
	case PublishOutcomeUncertain:
		return "uncertain"
	default:
		return "unknown"
	}
}

// PublishResult is a summary of what was actually handed to Transport.Publish.
// It is for reconciliation / logging only: navlink does not allocate IDs and does
// not RecordSuccessfulPublish — the orchestration layer decides that from err.
type PublishResult struct {
	Topic         string
	Channel       topic.Channel
	Manufacturer  string
	SerialNumber  string
	QoS           byte
	Payload       []byte
	HeaderID      uint32
	OrderID       string
	OrderUpdateID uint32
	ActionIDs     []string
}

// ClassifyPublish maps a publish error to Accepted / NotStarted / Uncertain.
// Prefer this on the orchestration path; IsPublish* predicates remain for logs.
func ClassifyPublish(err error) PublishOutcome {
	if err == nil {
		return PublishOutcomeAccepted
	}
	if IsPublishValidationFailed(err) ||
		IsPublishNotStarted(err) ||
		IsPublishQoSRejected(err) ||
		IsPublishBrokerRejected(err) {
		return PublishOutcomeNotStarted
	}
	if errors.Is(err, gerrors.InvalidConfig) {
		return PublishOutcomeNotStarted
	}
	if errors.Is(err, gerrors.TimeoutError) {
		return PublishOutcomeUncertain
	}
	if publishWasAttempted(err) {
		return PublishOutcomeUncertain
	}
	if IsPublishCanceled(err) || errors.Is(err, context.DeadlineExceeded) {
		return PublishOutcomeNotStarted
	}
	return PublishOutcomeUncertain
}

// MarkPublishAttempted records that MQTT Publish was already invoked.
// Timeout / cancel on the wrapped error classify as Uncertain.
func MarkPublishAttempted(err error) error {
	if err == nil {
		return nil
	}
	var already *publishAttemptedError
	if errors.As(err, &already) {
		return err
	}
	return &publishAttemptedError{err: err}
}

type publishAttemptedError struct {
	err error
}

func (e *publishAttemptedError) Error() string { return e.err.Error() }
func (e *publishAttemptedError) Unwrap() error { return e.err }

func publishWasAttempted(err error) bool {
	var e *publishAttemptedError
	return errors.As(err, &e)
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

// IsPublishValidationFailed reports light outbound validation rejection (bad packet).
// Distinct from broker reject — safe to fix the packet; usually not a same-ID retry case.
func IsPublishValidationFailed(err error) bool {
	return IsOutboundValidationFailed(err)
}

func validatePublishQoS(qos byte) error {
	if qos > 2 {
		return gerrors.QosNotSupported.With("qos", qos)
	}
	return nil
}
