package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

func TestHomunculusStatPool(t *testing.T) {
	if got := homunculusStatPool(500, 4.0); got != 2000 {
		t.Fatalf("500*4 = %d, want 2000", got)
	}
	if got := homunculusStatPool(0, 4.0); got != 4 {
		t.Fatalf("floored craftSum(1)*4 = %d, want 4", got)
	}
}

func TestHomunculusCraftSumAndInheritance(t *testing.T) {
	c := characters.New()
	// New() floors every skill to level 1, so measure the delta from investment.
	baseline := homunculusCraftSum(c)

	c.Skills[string(skills.WeaponCombat)] = 40 // combat — inherited
	c.Skills[string(skills.Spellcasting)] = 15 // combat — inherited
	c.Skills["blacksmithing"] = 60             // craft — drives pool, NOT inherited
	c.Skills["alchemy"] = 50                   // craft
	c.Skills["salvage"] = 30                   // craft

	// The three crafts replaced their level-1 baseline entries.
	want := baseline - 3 + 60 + 50 + 30
	if got := homunculusCraftSum(c); got != want {
		t.Fatalf("craftSum = %d, want %d", got, want)
	}

	inh := inheritedHomunculusSkills(c)
	if _, ok := inh["blacksmithing"]; ok {
		t.Fatal("crafting skills must NOT be inherited by the homunculus")
	}
	if inh[string(skills.WeaponCombat)] != 40 {
		t.Fatalf("weapon-combat should be inherited at 40, got %d", inh[string(skills.WeaponCombat)])
	}
	if inh[string(skills.Spellcasting)] != 15 {
		t.Fatalf("spellcasting should be inherited at 15, got %d", inh[string(skills.Spellcasting)])
	}
}
