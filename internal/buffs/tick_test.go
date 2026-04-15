package buffs

import (
	"testing"
)

func TestComputeTickAmount_HealNoScaling(t *testing.T) {
	amount := ComputeTickAmount(200, 0.05, 0, 1, 1.0)
	if amount != 10 {
		t.Errorf("expected 10, got %d", amount)
	}
}

func TestComputeTickAmount_DamageNoScaling(t *testing.T) {
	amount := ComputeTickAmount(200, -0.08, 0, 3, 1.0)
	if amount != -16 {
		t.Errorf("expected -16, got %d", amount)
	}
}

func TestComputeTickAmount_MinFloor(t *testing.T) {
	amount := ComputeTickAmount(10, 0.05, 0, 1, 1.0)
	if amount != 1 {
		t.Errorf("expected 1, got %d", amount)
	}
}

func TestComputeTickAmount_DamageMinFloor(t *testing.T) {
	amount := ComputeTickAmount(10, -0.05, 0, 3, 1.0)
	if amount != -3 {
		t.Errorf("expected -3, got %d", amount)
	}
}

func TestComputeTickAmount_WithSpellScaling(t *testing.T) {
	amount := ComputeTickAmount(200, 0.05, 0, 1, 2.0)
	if amount != 20 {
		t.Errorf("expected 20, got %d", amount)
	}
}

func TestComputeTickAmount_ZeroPercent(t *testing.T) {
	amount := ComputeTickAmount(200, 0, 0, 1, 1.0)
	if amount != 0 {
		t.Errorf("expected 0, got %d", amount)
	}
}
