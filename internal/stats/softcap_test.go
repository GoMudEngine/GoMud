package stats

import "testing"

// Recalculate must be the identity: ValueAdj tracks Value with no
// compression. Before 2026-08-02 it compressed anything above 150, which
// silently applied to the resource pools too (they are StatInfo as well).
func TestRecalculateIsIdentity(t *testing.T) {
	for _, raw := range []int{0, 1, 85, 100, 105, 149, 150, 151, 165, 195, 280, 411, 530, 1000} {
		si := StatInfo{Base: raw}
		si.Recalculate()
		if si.Value != raw {
			t.Errorf("Base=%d: Value=%d, want %d", raw, si.Value, raw)
		}
		if si.ValueAdj != raw {
			t.Errorf("Base=%d: ValueAdj=%d, want %d (compression must be gone)", raw, si.ValueAdj, raw)
		}
	}
}

// Training and Mods must still sum into Value.
func TestRecalculateSumsComponents(t *testing.T) {
	si := StatInfo{Base: 85, Training: 326, Mods: 10}
	si.Recalculate()
	if si.Value != 421 || si.ValueAdj != 421 {
		t.Errorf("got Value=%d ValueAdj=%d, want 421/421", si.Value, si.ValueAdj)
	}
	if si.Racial != 85 {
		t.Errorf("Racial=%d, want 85", si.Racial)
	}
}

// A pool-sized value must pass through untouched. HealthMax/StaminaMax/
// ConvictionMax/ActionPointsMax are StatInfo and set Mods, so this is the
// path that was silently costing every character ~40% of every pool.
func TestPoolSizedValuesAreNotCompressed(t *testing.T) {
	si := StatInfo{Mods: 530}
	si.Recalculate()
	if si.ValueAdj != 530 {
		t.Errorf("pool Mods=530 -> ValueAdj=%d, want 530", si.ValueAdj)
	}
	ap := StatInfo{Mods: 200}
	ap.Recalculate()
	if ap.ValueAdj != 200 {
		t.Errorf("ActionPointsMax 200 -> ValueAdj=%d, want 200", ap.ValueAdj)
	}
}
