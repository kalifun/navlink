package facts_test

import (
	"testing"

	"github.com/kalifun/vda5050-types-go/state"

	"github.com/kalifun/navlink/facts"
)

func TestApplyLastNodeAndOrder(t *testing.T) {
	prev := &state.State{LastNodeId: "N1", OrderId: "o1", OrderUpdateId: 1}
	next := &state.State{LastNodeId: "N2", OrderId: "o2", OrderUpdateId: 2}
	got := facts.Apply(prev, next)
	kinds := map[facts.Kind]bool{}
	for _, f := range got {
		kinds[f.Kind] = true
	}
	for _, want := range []facts.Kind{
		facts.KindLastNodeChanged,
		facts.KindOrderIdChanged,
		facts.KindOrderUpdateChanged,
	} {
		if !kinds[want] {
			t.Fatalf("missing %s in %#v", want, got)
		}
	}
}

func TestApplyActionTransition(t *testing.T) {
	prev := &state.State{
		ActionStates: []state.ActionState{
			{ActionId: "a1", ActionStatus: state.ActionWaiting},
		},
	}
	next := &state.State{
		ActionStates: []state.ActionState{
			{ActionId: "a1", ActionStatus: state.ActionRunning},
		},
	}
	got := facts.Apply(prev, next)
	if len(got) != 1 || got[0].Kind != facts.KindActionTransitioned {
		t.Fatalf("got=%#v", got)
	}
	if got[0].PrevStatus != state.ActionWaiting || got[0].NextStatus != state.ActionRunning {
		t.Fatalf("fact=%#v", got[0])
	}
}

func TestApplyNilPrev(t *testing.T) {
	next := &state.State{LastNodeId: "N1", OrderId: "o1"}
	got := facts.Apply(nil, next)
	if len(got) < 2 {
		t.Fatalf("got=%#v", got)
	}
}
