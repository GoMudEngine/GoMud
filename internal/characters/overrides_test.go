package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/stats"
)

func TestApplyMobOverrides_HealthMax(t *testing.T) {
	c := &Character{
		HealthMax: stats.StatInfo{Base: 1, Value: 100},
	}
	c.Health = 100
	ApplyMobOverrides(c, 1500, 0, 0)
	if c.HealthMax.Value != 1500 {
		t.Errorf("HealthMax.Value = %d, want 1500", c.HealthMax.Value)
	}
	if c.Health != 1500 {
		t.Errorf("Health = %d, want 1500 (filled to max)", c.Health)
	}
}

func TestApplyMobOverrides_StaminaMax(t *testing.T) {
	c := &Character{
		StaminaMax: stats.StatInfo{Base: 1, Value: 50},
	}
	c.Stamina = 50
	ApplyMobOverrides(c, 0, 9999, 0)
	if c.StaminaMax.Value != 9999 {
		t.Errorf("StaminaMax.Value = %d, want 9999", c.StaminaMax.Value)
	}
	if c.Stamina != 9999 {
		t.Errorf("Stamina = %d, want 9999 (filled to max)", c.Stamina)
	}
}

func TestApplyMobOverrides_CarryCapacity(t *testing.T) {
	c := &Character{}
	c.Stats.Strength.ValueAdj = 10 // would yield ~6.5 from default calc
	ApplyMobOverrides(c, 0, 0, 5000)
	if got := c.CarryCapacity(); got != 5000 {
		t.Errorf("CarryCapacity() = %v, want 5000", got)
	}
}

func TestApplyMobOverrides_NoOverrides_PreservesDefaults(t *testing.T) {
	c := &Character{
		HealthMax:  stats.StatInfo{Value: 100},
		StaminaMax: stats.StatInfo{Value: 50},
	}
	c.Health = 100
	c.Stamina = 50
	c.Stats.Strength.ValueAdj = 100
	ApplyMobOverrides(c, 0, 0, 0)
	if c.HealthMax.Value != 100 || c.Health != 100 {
		t.Error("zero override should not modify HP")
	}
	if c.StaminaMax.Value != 50 || c.Stamina != 50 {
		t.Error("zero override should not modify SP")
	}
	// CarryCapacity should fall through to Strength-derived calc
	got := c.CarryCapacity()
	if got <= 0 {
		t.Errorf("CarryCapacity() with no override = %v, expected non-zero from Strength", got)
	}
}
