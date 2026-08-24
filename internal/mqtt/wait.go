package mqtt

import (
	"context"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/kalifun/navlink/internal/gerrors"
)

const defaultTokenWait = 30 * time.Second

// waitToken waits for an MQTT token, honoring ctx cancel/deadline.
// The paho WaitTimeout goroutine always finishes within timeout, so a
// timed-out or canceled wait does not leak a token.Wait() goroutine.
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

	type outcome struct {
		completed bool
		err       error
	}
	done := make(chan outcome, 1)
	go func() {
		ok := token.WaitTimeout(timeout)
		done <- outcome{completed: ok, err: token.Error()}
	}()

	select {
	case r := <-done:
		if !r.completed {
			return gerrors.TimeoutError
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
