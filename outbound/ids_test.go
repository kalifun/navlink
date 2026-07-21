package outbound_test

import (
	"testing"

	"github.com/kalifun/navlink/outbound"
)

func TestMemoryHeaderIDsMonotonic(t *testing.T) {
	p := outbound.NewMemoryHeaderIDs()
	var prev uint32
	for i := 0; i < 5; i++ {
		id, err := p.Next()
		if err != nil {
			t.Fatal(err)
		}
		if id <= prev {
			t.Fatalf("headerId not monotonic: prev=%d got=%d", prev, id)
		}
		prev = id
	}
}

func TestOrderUpdateGetNextCommit(t *testing.T) {
	s := outbound.NewMemoryOrderUpdateIDs()

	a, err := s.GetNext("order-1")
	if err != nil || a != 1 {
		t.Fatalf("GetNext = %d, %v", a, err)
	}
	// Retry before commit returns same pending id.
	b, err := s.GetNext("order-1")
	if err != nil || b != 1 {
		t.Fatalf("pending GetNext = %d, %v", b, err)
	}
	if err := s.Commit("order-1", 1); err != nil {
		t.Fatal(err)
	}
	c, err := s.GetNext("order-1")
	if err != nil || c != 2 {
		t.Fatalf("after commit GetNext = %d, %v", c, err)
	}
}

func TestOrderUpdateSyncFromVehicleNeverRewinds(t *testing.T) {
	s := outbound.NewMemoryOrderUpdateIDs()
	id, err := s.GetNext("order-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Commit("order-1", id); err != nil {
		t.Fatal(err)
	}
	// Advance by issuing and committing 2, 3
	for _, want := range []uint32{2, 3} {
		n, err := s.GetNext("order-1")
		if err != nil || n != want {
			t.Fatalf("GetNext want %d got %d err %v", want, n, err)
		}
		if err := s.Commit("order-1", n); err != nil {
			t.Fatal(err)
		}
	}
	if s.LastCommitted("order-1") != 3 {
		t.Fatalf("committed=%d", s.LastCommitted("order-1"))
	}

	// Vehicle lags at 1 — must not rewind.
	s.SyncFromVehicle("order-1", 1)
	if s.LastCommitted("order-1") != 3 {
		t.Fatalf("rewind detected: committed=%d", s.LastCommitted("order-1"))
	}
	next, err := s.GetNext("order-1")
	if err != nil || next != 4 {
		t.Fatalf("after lag sync GetNext=%d err=%v", next, err)
	}

	// Vehicle ahead raises watermark.
	s.SyncFromVehicle("order-1", 10)
	if s.LastCommitted("order-1") != 10 {
		t.Fatalf("ahead sync committed=%d", s.LastCommitted("order-1"))
	}
	// Pending 4 is obsolete after watermark 10.
	next, err = s.GetNext("order-1")
	if err != nil || next != 11 {
		t.Fatalf("after ahead sync GetNext=%d err=%v", next, err)
	}
}

func TestMemoryActionIDsIndependentPerAGV(t *testing.T) {
	a := outbound.NewMemoryActionIDs()
	id1, err := a.Next("M", "S1")
	if err != nil || id1 != "1" {
		t.Fatalf("S1 first=%q err=%v", id1, err)
	}
	id2, err := a.Next("M", "S2")
	if err != nil || id2 != "1" {
		t.Fatalf("S2 first=%q err=%v", id2, err)
	}
	id3, err := a.Next("M", "S1")
	if err != nil || id3 != "2" {
		t.Fatalf("S1 second=%q err=%v", id3, err)
	}
}
