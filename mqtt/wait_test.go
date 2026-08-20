package mqtt

import (
	"context"
	"testing"
)

func TestWaitTokenAlreadyCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := waitToken(ctx, nil); err != context.Canceled {
		t.Fatalf("err=%v", err)
	}
}
