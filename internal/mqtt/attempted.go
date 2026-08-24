package mqtt

// AttemptedError wraps a publish wait failure after MQTT Publish was invoked.
// The packet may already be in flight; callers should treat this as uncertain.
type AttemptedError struct {
	Err error
}

func (e *AttemptedError) Error() string {
	if e == nil || e.Err == nil {
		return "mqtt publish attempted"
	}
	return e.Err.Error()
}

func (e *AttemptedError) Unwrap() error { return e.Err }

func attempted(err error) error {
	if err == nil {
		return nil
	}
	return &AttemptedError{Err: err}
}
