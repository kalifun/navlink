package facts

import "github.com/kalifun/vda5050-types-go/state"

// Kind identifies a protocol-level fact derived from consecutive states.
// These are not domain/scheduling events.
type Kind string

const (
	KindLastNodeChanged    Kind = "LastNodeChanged"
	KindOrderIdChanged     Kind = "OrderIdChanged"
	KindOrderUpdateChanged Kind = "OrderUpdateChanged"
	KindActionTransitioned Kind = "ActionTransitioned"
)

// Fact is a pure protocol observation between two State messages.
type Fact struct {
	Kind Kind

	// LastNodeChanged
	PrevLastNodeID string
	NextLastNodeID string

	// OrderIdChanged / OrderUpdateChanged
	PrevOrderID       string
	NextOrderID       string
	PrevOrderUpdateID uint32
	NextOrderUpdateID uint32

	// ActionTransitioned
	ActionID   string
	PrevStatus state.ActionStatus
	NextStatus state.ActionStatus
	ActionType string
}

// Apply diffs prev → next and returns protocol facts only.
// prev may be nil (first observation); then only "changed from empty" facts are emitted
// when next has non-empty values that are useful to observe.
func Apply(prev, next *state.State) []Fact {
	if next == nil {
		return nil
	}
	var out []Fact

	prevNode, prevOrder, prevUpdate := "", "", uint32(0)
	if prev != nil {
		prevNode = prev.LastNodeId
		prevOrder = prev.OrderId
		prevUpdate = prev.OrderUpdateId
	}

	if prevNode != next.LastNodeId {
		out = append(out, Fact{
			Kind:           KindLastNodeChanged,
			PrevLastNodeID: prevNode,
			NextLastNodeID: next.LastNodeId,
		})
	}
	if prevOrder != next.OrderId {
		out = append(out, Fact{
			Kind:        KindOrderIdChanged,
			PrevOrderID: prevOrder,
			NextOrderID: next.OrderId,
		})
	}
	if prevUpdate != next.OrderUpdateId {
		out = append(out, Fact{
			Kind:              KindOrderUpdateChanged,
			PrevOrderID:       next.OrderId,
			NextOrderID:       next.OrderId,
			PrevOrderUpdateID: prevUpdate,
			NextOrderUpdateID: next.OrderUpdateId,
		})
	}

	prevActions := map[string]state.ActionState{}
	if prev != nil {
		for _, a := range prev.ActionStates {
			prevActions[a.ActionId] = a
		}
	}
	for _, a := range next.ActionStates {
		old, ok := prevActions[a.ActionId]
		if !ok {
			out = append(out, Fact{
				Kind:       KindActionTransitioned,
				ActionID:   a.ActionId,
				PrevStatus: "",
				NextStatus: a.ActionStatus,
				ActionType: derefStr(a.ActionType),
			})
			continue
		}
		if old.ActionStatus != a.ActionStatus {
			out = append(out, Fact{
				Kind:       KindActionTransitioned,
				ActionID:   a.ActionId,
				PrevStatus: old.ActionStatus,
				NextStatus: a.ActionStatus,
				ActionType: derefStr(a.ActionType),
			})
		}
	}
	return out
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
