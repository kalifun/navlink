package mqtt

import (
	"context"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/kalifun/navlink/gerrors"
)

const defaultTokenWait = 30 * time.Second

// waitToken waits for an MQTT token, honoring ctx cancel/deadline.
// Returns TimeoutError when the default/absolute wait elapses without ctx error;
// otherwise returns ctx.Err() when the context ends first.
func waitToken(ctx context.Context, token pahomqtt.Token) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	timeout := defaultTokenWait
	if deadline, ok := ctx.Deadline(); ok {
		d := time.Until(deadline)
		if d <= 0 {
			return context.DeadlineExceeded
		}
		if d < timeout {
			timeout = d
		}
	}

	done := make(chan struct{})
	go func() {
		token.Wait()
		close(done)
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		if err := ctx.Err(); err != nil {
			return err
		}
		return gerrors.TimeoutError
	}
}
