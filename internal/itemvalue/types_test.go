package itemvalue

import "testing"

func TestSlotName_Values(t *testing.T) {
	// Smoke-check that the slot constants have the exact
	// Worn-struct field names. If someone renames a field on
	// characters.Worn, this test won't catch it directly, but
	// it locks the expected naming convention.
	cases := map[SlotName]string{
		SlotWeapon:    "Weapon",
		SlotOffhand:   "Offhand",
		SlotRing:      "Ring",
		SlotRing2:     "Ring2",
		SlotWrist1:    "Wrist1",
		SlotWrist2:    "Wrist2",
		SlotExtraArm1: "ExtraArm1",
		SlotTail:      "Tail",
	}
	for got, want := range cases {
		if string(got) != want {
			t.Errorf("SlotName = %q, want %q", string(got), want)
		}
	}
}

func TestWeightProfile_ZeroValue(t *testing.T) {
	var p WeightProfile
	if p.Name != "" {
		t.Errorf("zero-value Name = %q, want empty", p.Name)
	}
	if p.PhysicalDamageWeight != 0 {
		t.Errorf("zero-value PhysicalDamageWeight = %f, want 0", p.PhysicalDamageWeight)
	}
}

func TestSwapDelta_ZeroValue(t *testing.T) {
	var d SwapDelta
	if d.Score != 0 {
		t.Errorf("zero-value Score = %f, want 0", d.Score)
	}
	if d.Slot != "" {
		t.Errorf("zero-value Slot = %q, want empty", d.Slot)
	}
	if d.Displaced != nil {
		t.Errorf("zero-value Displaced = %v, want nil", d.Displaced)
	}
}
