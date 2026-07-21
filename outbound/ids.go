package outbound

import (
	"fmt"
	"strconv"
	"sync"
)

// HeaderIdProvider allocates monotonic headerId values for master-side outbound
// Order / InstantActions messages.
type HeaderIdProvider interface {
	Next() (uint32, error)
}

// OrderUpdateIdStore manages per-orderId orderUpdateId values.
//
// Typical flow: GetNext → publish → Commit on success.
// SyncFromVehicle observes vehicle-reported ids and must never rewind an
// already issued number when the vehicle lags behind.
type OrderUpdateIdStore interface {
	GetNext(orderID string) (uint32, error)
	Commit(orderID string, id uint32) error
	SyncFromVehicle(orderID string, observed uint32)
}

// ActionIdAllocator allocates monotonic actionId strings per AGV.
// It is independent of HeaderIdProvider.
type ActionIdAllocator interface {
	Next(manufacturer, serial string) (string, error)
}

// MemoryHeaderIDs is a process-local HeaderIdProvider.
type MemoryHeaderIDs struct {
	mu   sync.Mutex
	next uint32
}

// NewMemoryHeaderIDs starts allocating from 1.
func NewMemoryHeaderIDs() *MemoryHeaderIDs {
	return &MemoryHeaderIDs{next: 1}
}

// Next returns the next monotonic headerId.
func (p *MemoryHeaderIDs) Next() (uint32, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.next == 0 {
		p.next = 1
	}
	id := p.next
	p.next++
	return id, nil
}

// MemoryOrderUpdateIDs is a process-local OrderUpdateIdStore.
type MemoryOrderUpdateIDs struct {
	mu        sync.Mutex
	committed map[string]uint32 // last successfully committed id
	pending   map[string]uint32 // allocated but not yet committed
}

// NewMemoryOrderUpdateIDs creates an empty in-memory store.
func NewMemoryOrderUpdateIDs() *MemoryOrderUpdateIDs {
	return &MemoryOrderUpdateIDs{
		committed: make(map[string]uint32),
		pending:   make(map[string]uint32),
	}
}

// GetNext returns the next orderUpdateId for orderID.
// If a previous GetNext was not committed, the same pending id is returned.
func (s *MemoryOrderUpdateIDs) GetNext(orderID string) (uint32, error) {
	if orderID == "" {
		return 0, fmt.Errorf("orderId is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if id, ok := s.pending[orderID]; ok {
		return id, nil
	}
	next := s.committed[orderID] + 1
	s.pending[orderID] = next
	return next, nil
}

// Commit records a successfully published orderUpdateId.
func (s *MemoryOrderUpdateIDs) Commit(orderID string, id uint32) error {
	if orderID == "" {
		return fmt.Errorf("orderId is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if pending, ok := s.pending[orderID]; ok && pending != id {
		return fmt.Errorf("commit id %d does not match pending %d for order %q", id, pending, orderID)
	}
	delete(s.pending, orderID)
	if id > s.committed[orderID] {
		s.committed[orderID] = id
	}
	return nil
}

// SyncFromVehicle raises the committed watermark when the vehicle reports a
// higher orderUpdateId. A lagging vehicle must not rewind issued numbers.
func (s *MemoryOrderUpdateIDs) SyncFromVehicle(orderID string, observed uint32) {
	if orderID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if observed > s.committed[orderID] {
		s.committed[orderID] = observed
	}
	if pending, ok := s.pending[orderID]; ok && pending <= s.committed[orderID] {
		delete(s.pending, orderID)
	}
}

// LastCommitted returns the last committed id (for tests).
func (s *MemoryOrderUpdateIDs) LastCommitted(orderID string) uint32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.committed[orderID]
}

// MemoryActionIDs is a process-local ActionIdAllocator.
type MemoryActionIDs struct {
	mu   sync.Mutex
	next map[string]uint64
}

// NewMemoryActionIDs creates an empty in-memory allocator.
func NewMemoryActionIDs() *MemoryActionIDs {
	return &MemoryActionIDs{next: make(map[string]uint64)}
}

// Next returns the next actionId for the AGV as a decimal string.
func (a *MemoryActionIDs) Next(manufacturer, serial string) (string, error) {
	key := manufacturer + "/" + serial
	a.mu.Lock()
	defer a.mu.Unlock()
	a.next[key]++
	return strconv.FormatUint(a.next[key], 10), nil
}
