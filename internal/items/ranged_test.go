package items

import (
	"testing"

	"gopkg.in/yaml.v2"
)

func TestItemSpec_RangedFields(t *testing.T) {
	spec := ItemSpec{Subtype: Shooting, AmmoTag: "bolts", MinStrength: 90}
	if spec.AmmoTag != "bolts" || spec.MinStrength != 90 {
		t.Fatalf("ranged spec fields not settable: %+v", spec)
	}
}

func TestItem_LoadedState(t *testing.T) {
	itm := Item{ItemId: 1, Loaded: true}
	if !itm.Loaded {
		t.Error("Loaded field must persist on the instance")
	}
}

func TestItem_IsRangedWeapon(t *testing.T) {
	ranged := Item{ItemId: 1, Spec: &ItemSpec{ItemId: 1, Type: Weapon, Subtype: Shooting}}
	if !ranged.IsRangedWeapon() {
		t.Error("shooting-subtype weapon must report IsRangedWeapon")
	}
	melee := Item{ItemId: 2, Spec: &ItemSpec{ItemId: 2, Type: Weapon, Subtype: Slashing}}
	if melee.IsRangedWeapon() {
		t.Error("slashing weapon must not report IsRangedWeapon")
	}
	var none Item
	if none.IsRangedWeapon() {
		t.Error("zero item must not report IsRangedWeapon")
	}
}

func TestAmmoType(t *testing.T) {
	bundle := Item{ItemId: 3, Uses: 20, Spec: &ItemSpec{ItemId: 3, Type: Ammo, AmmoTag: "arrows"}}
	if bundle.GetSpec().Type != Ammo || bundle.GetSpec().AmmoTag != "arrows" {
		t.Errorf("ammo bundle spec: %+v", bundle.GetSpec())
	}
}

// TestItem_Loaded_YamlRoundTrip verifies that the Loaded field survives a
// YAML marshal/unmarshal cycle — proving instance persistence.
func TestItem_Loaded_YamlRoundTrip(t *testing.T) {
	orig := Item{ItemId: 4, Loaded: true}
	data, err := yaml.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var got Item
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !got.Loaded {
		t.Errorf("Loaded=true did not survive yaml round-trip; marshalled:\n%s", data)
	}
}
