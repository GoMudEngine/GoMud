package catalog

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestUpgradeGear_PredicateAlwaysFalse(t *testing.T) {
	mob := &mobs.Mob{}
	mob.Character.Gold = 99999
	g := &goals.Goal{Type: "upgrade-gear"}
	if upgradeGearPredicate(g, mob) {
		t.Errorf("upgrade-gear predicate must always be false (perpetual drive)")
	}
}

func TestUpgradeGear_PredicateNilMob(t *testing.T) {
	if upgradeGearPredicate(&goals.Goal{Type: "upgrade-gear"}, nil) {
		t.Errorf("predicate must be false for nil mob")
	}
}

func TestUpgradeGear_ContextScore_FloorWhenBroke(t *testing.T) {
	mob := &mobs.Mob{} // 0 gold, no items, not in combat
	g := &goals.Goal{Type: "upgrade-gear"}
	if got := upgradeGearContextScore(g, mob); got != 1.0 {
		t.Errorf("context score = %v, want floor 1.0 (broke + no loot)", got)
	}
}

func TestUpgradeGear_ContextScore_ElevatedWhenGold(t *testing.T) {
	mob := &mobs.Mob{}
	mob.Character.Gold = 1000 // well above default reserve 50
	g := &goals.Goal{Type: "upgrade-gear"}
	if got := upgradeGearContextScore(g, mob); got != 2.5 {
		t.Errorf("context score = %v, want 2.5 (spendable gold, idle)", got)
	}
}

func TestUpgradeGear_ContextScore_FloorInCombat(t *testing.T) {
	mob := &mobs.Mob{}
	mob.Character.Gold = 1000
	mob.Character.Aggro = &characters.Aggro{} // in combat
	g := &goals.Goal{Type: "upgrade-gear"}
	if got := upgradeGearContextScore(g, mob); got != 1.0 {
		t.Errorf("context score = %v, want floor 1.0 while in combat", got)
	}
}

func TestUpgradeGear_ContextScore_NilMob(t *testing.T) {
	if got := upgradeGearContextScore(&goals.Goal{}, nil); got != 0 {
		t.Errorf("context score = %v, want 0 for nil mob", got)
	}
}
