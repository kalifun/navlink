package navlink_test

import (
	"context"
	"strings"
	"sync"

	"github.com/kalifun/navlink"
)

// hangStateSubscribeTransport blocks Subscribe on per-AGV state filters until hangState is closed.
type hangStateSubscribeTransport struct {
	memoryTransport
	hangState    <-chan struct{}
	enteredState chan struct{}
	once         sync.Once
}

func (t *hangStateSubscribeTransport) Subscribe(ctx context.Context, filter string, handler navlink.RawHandler) (navlink.Unsubscribe, error) {
	if strings.Contains(filter, "/state") && !strings.Contains(filter, "+") {
		t.once.Do(func() {
			close(t.enteredState)
		})
		select {
		case <-t.hangState:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return t.memoryTransport.Subscribe(ctx, filter, handler)
}
