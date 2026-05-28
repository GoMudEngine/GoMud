package seeders

import (
	"sync/atomic"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
)

// fakeEvent implements events.Event for tests — minimal shape.
type fakeEvent struct{ typeName string }

func (f fakeEvent) Type() string { return f.typeName }

func TestRegister_DispatchInvokesRuleForMatchingType(t *testing.T) {
	resetRegistryForTest()
	var called int32
	Register("test-rule", func(events.Event) {
		atomic.AddInt32(&called, 1)
	}, "test-type-A")

	Dispatch(fakeEvent{typeName: "test-type-A"})
	if atomic.LoadInt32(&called) != 1 {
		t.Errorf("rule not invoked on matching event")
	}
}

func TestDispatch_NoRulesForType_NoOp(t *testing.T) {
	resetRegistryForTest()
	// No rules registered. Dispatch must not panic.
	ret := Dispatch(fakeEvent{typeName: "unsubscribed-type"})
	if ret != events.Continue {
		t.Errorf("Dispatch returned %v, want events.Continue", ret)
	}
}

func TestDispatch_OnlyMatchingTypeFires(t *testing.T) {
	resetRegistryForTest()
	var calledA, calledB int32
	Register("rule-A", func(events.Event) { atomic.AddInt32(&calledA, 1) }, "type-A")
	Register("rule-B", func(events.Event) { atomic.AddInt32(&calledB, 1) }, "type-B")

	Dispatch(fakeEvent{typeName: "type-A"})
	if atomic.LoadInt32(&calledA) != 1 {
		t.Errorf("rule-A should have fired")
	}
	if atomic.LoadInt32(&calledB) != 0 {
		t.Errorf("rule-B should NOT have fired on type-A event")
	}
}

func TestDispatch_MultipleRulesForSameType_AllFire(t *testing.T) {
	resetRegistryForTest()
	var calledA, calledB int32
	Register("rule-A", func(events.Event) { atomic.AddInt32(&calledA, 1) }, "shared-type")
	Register("rule-B", func(events.Event) { atomic.AddInt32(&calledB, 1) }, "shared-type")

	Dispatch(fakeEvent{typeName: "shared-type"})
	if atomic.LoadInt32(&calledA) != 1 || atomic.LoadInt32(&calledB) != 1 {
		t.Errorf("both rules should have fired: a=%d b=%d", calledA, calledB)
	}
}

func TestDispatch_PanicInOneRule_OthersStillFire(t *testing.T) {
	resetRegistryForTest()
	var calledAfter int32
	Register("rule-panic", func(events.Event) {
		panic("boom")
	}, "shared-type")
	Register("rule-after", func(events.Event) {
		atomic.AddInt32(&calledAfter, 1)
	}, "shared-type")

	// Must not panic.
	Dispatch(fakeEvent{typeName: "shared-type"})
	if atomic.LoadInt32(&calledAfter) != 1 {
		t.Errorf("rule-after should have fired despite rule-panic")
	}
}

func TestRegister_OneRuleMultipleTypes(t *testing.T) {
	resetRegistryForTest()
	var called int32
	Register("multi-type", func(events.Event) {
		atomic.AddInt32(&called, 1)
	}, "type-X", "type-Y")

	Dispatch(fakeEvent{typeName: "type-X"})
	Dispatch(fakeEvent{typeName: "type-Y"})
	Dispatch(fakeEvent{typeName: "type-Z"})

	if atomic.LoadInt32(&called) != 2 {
		t.Errorf("called=%d, want 2 (X + Y, not Z)", called)
	}
}
