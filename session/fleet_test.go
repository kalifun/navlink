package session_test

import (
	"context"
	"sync"
	"testing"

	"github.com/kalifun/vda5050-types-go/connection"

	"github.com/kalifun/navlink/session"
	"github.com/kalifun/navlink/topic"
)

type fakeSub struct {
	mu      sync.Mutex
	filters []string
}

func (f *fakeSub) subscribe(ctx context.Context, filter string) (session.Unsubscribe, error) {
	f.mu.Lock()
	f.filters = append(f.filters, filter)
	f.mu.Unlock()
	return func(ctx context.Context) error {
		f.mu.Lock()
		defer f.mu.Unlock()
		for i, x := range f.filters {
			if x == filter {
				f.filters = append(f.filters[:i], f.filters[i+1:]...)
				break
			}
		}
		return nil
	}, nil
}

func (f *fakeSub) has(filter string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, x := range f.filters {
		if x == filter {
			return true
		}
	}
	return false
}

func TestFleetConnectionDrivesTrackUntrack(t *testing.T) {
	sub := &fakeSub{}
	topics := topic.Resolver{Interface: "uagv", Version: "v2"}
	fleet := session.NewFleetSession(topics, sub.subscribe, session.DefaultOptions())

	var online, offline int
	fleet.OnAGVOnline(func(agv session.AGV) { online++ })
	fleet.OnAGVOffline(func(agv session.AGV) { offline++ })

	ctx := context.Background()
	if err := fleet.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if !sub.has("uagv/v2/+/+/connection") {
		t.Fatal("expected connection wildcard")
	}

	agv := session.AGV{Manufacturer: "M", SerialNumber: "S1"}
	if err := fleet.HandleConnection(ctx, agv, connection.Online); err != nil {
		t.Fatal(err)
	}
	if !sub.has("uagv/v2/M/S1/state") {
		t.Fatal("expected per-AGV state sub")
	}
	if online != 1 {
		t.Fatalf("online=%d", online)
	}

	if err := fleet.HandleConnection(ctx, agv, connection.Offline); err != nil {
		t.Fatal(err)
	}
	if sub.has("uagv/v2/M/S1/state") {
		t.Fatal("state sub should be removed")
	}
	if offline != 1 {
		t.Fatalf("offline=%d", offline)
	}
}

func TestFleetRestoreKeepsTrackedWithoutExtraHooks(t *testing.T) {
	sub := &fakeSub{}
	topics := topic.Resolver{Interface: "uagv", Version: "v2"}
	fleet := session.NewFleetSession(topics, sub.subscribe, session.DefaultOptions())
	var online int
	fleet.OnAGVOnline(func(agv session.AGV) { online++ })

	ctx := context.Background()
	_ = fleet.Start(ctx)
	agv := session.AGV{Manufacturer: "M", SerialNumber: "S1"}
	_ = fleet.Track(ctx, agv)
	if online != 1 {
		t.Fatalf("online=%d", online)
	}
	if err := fleet.Restore(ctx); err != nil {
		t.Fatal(err)
	}
	if online != 1 {
		t.Fatalf("restore should not re-fire online, online=%d", online)
	}
	if !sub.has("uagv/v2/+/+/connection") || !sub.has("uagv/v2/M/S1/state") {
		t.Fatalf("filters after restore: %v", sub.filters)
	}
}
