package items

import (
	"slices"
	"testing"
)

func TestValidProcTriggers(t *testing.T) {
	got := ValidProcTriggers()
	for _, want := range []string{"on_hit", "on_kill", "on_block", "on_grapple", "on_spell_hit"} {
		if !slices.Contains(got, want) {
			t.Errorf("missing trigger %q in %v", want, got)
		}
	}
	if !slices.IsSorted(got) {
		t.Errorf("triggers must be sorted: %v", got)
	}
}

func TestValidProcEffects(t *testing.T) {
	got := ValidProcEffects()
	for _, want := range []string{"lifesteal", "steal_pool", "aoe_stun", "apply_condition"} {
		if !slices.Contains(got, want) {
			t.Errorf("missing effect %q in %v", want, got)
		}
	}
}
