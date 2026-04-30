package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mutators"
)

func TestRegenMultFromSpecs_Empty(t *testing.T) {
	if got := regenMultFromSpecs(); got != 1.0 {
		t.Errorf("regenMultFromSpecs() = %v, want 1.0", got)
	}
}

func TestRegenMultFromSpecs_NilSpec(t *testing.T) {
	if got := regenMultFromSpecs(nil); got != 1.0 {
		t.Errorf("nil spec contributed multiplier %v, want 1.0", got)
	}
}

func TestRegenMultFromSpecs_ZeroMultIgnored(t *testing.T) {
	spec := &mutators.MutatorSpec{MutatorId: "noregen"}
	if got := regenMultFromSpecs(spec); got != 1.0 {
		t.Errorf("zero-multiplier spec contributed %v, want 1.0", got)
	}
}

func TestRegenMultFromSpecs_SingleSpec(t *testing.T) {
	spec := &mutators.MutatorSpec{MutatorId: "sanctuary", RegenMultiplier: 5.0}
	if got := regenMultFromSpecs(spec); got != 5.0 {
		t.Errorf("regenMultFromSpecs(sanctuary) = %v, want 5.0", got)
	}
}

func TestRegenMultFromSpecs_StacksMultiplicatively(t *testing.T) {
	a := &mutators.MutatorSpec{MutatorId: "sanctuary", RegenMultiplier: 5.0}
	b := &mutators.MutatorSpec{MutatorId: "sanctuary2", RegenMultiplier: 5.0}
	if got := regenMultFromSpecs(a, b); got != 25.0 {
		t.Errorf("regenMultFromSpecs(5.0, 5.0) = %v, want 25.0", got)
	}
}

func TestRegenMultFromSpecs_MixedZeroAndNonZero(t *testing.T) {
	a := &mutators.MutatorSpec{MutatorId: "sanctuary", RegenMultiplier: 5.0}
	b := &mutators.MutatorSpec{MutatorId: "noregen"}
	if got := regenMultFromSpecs(a, b); got != 5.0 {
		t.Errorf("regenMultFromSpecs(5.0, 0) = %v, want 5.0", got)
	}
}

func TestRoomRegenMultiplier_NilRoom(t *testing.T) {
	if got := roomRegenMultiplier(nil); got != 1.0 {
		t.Errorf("nil room got %v, want 1.0", got)
	}
}
