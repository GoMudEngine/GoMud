package hooks_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/GoMudEngine/GoMud/internal/state/combatphase"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	_ "github.com/GoMudEngine/GoMud/internal/hooks" // wire init() observers
)

// TestActivityCascadeOnLifeDead verifies that the activity_life_dead
// observer cancels any active Activity when Life transitions Alive→Dead.
// Covers AC-035.
func TestActivityCascadeOnLifeDead(t *testing.T) {
	c := characters.New()
	_ = c.Activity.TransitionToCasting(
		activity.CastingData{SpellId: "fireball", FoldsAccumulated: 2},
		state.TransitionReason{Trigger: activity.TriggerCastBegin},
	)
	if !c.IsCasting() {
		t.Fatal("setup: expected character to be casting")
	}

	_ = c.Life.TransitionToDead(
		life.DeadData{},
		state.TransitionReason{Trigger: life.TriggerSuicide},
	)

	if c.IsCasting() {
		t.Errorf("Activity should have cascaded to Free on Life Dead")
	}
	if !c.IsFree() {
		t.Errorf("Activity state = %v, expected Free", c.Activity.State())
	}
}

// TestCombatEntryVetoedWhenCrafting verifies that TransitionToEngaging is
// vetoed (returns a non-nil error) while the character is Crafting, and that
// the Activity remains Crafting afterwards. The activity_self veto in
// CombatPhase_Vetoes.go provides this protection — no separate cascade is
// needed. Covers AC-038.
func TestCombatEntryVetoedWhenCrafting(t *testing.T) {
	c := characters.New()
	_ = c.Activity.TransitionToCrafting(
		activity.CraftingData{RecipeId: "iron_dagger", RoundsTotal: 4},
		state.TransitionReason{Trigger: activity.TriggerCraftBegin},
	)
	if !c.IsCrafting() {
		t.Fatal("setup: expected character to be crafting")
	}

	err := c.CombatPhase.TransitionToEngaging(
		combatphase.EngagingData{Target: state.ActorRef{UserId: 999}},
		state.TransitionReason{Trigger: combatphase.TriggerAttackCommand},
	)

	if err == nil {
		t.Error("TransitionToEngaging should be vetoed while crafting; got nil error")
	}
	if !c.IsCrafting() {
		t.Errorf(
			"Activity should remain Crafting after vetoed combat entry; got %v",
			c.Activity.State(),
		)
	}
}

// TestCombatEntryAllowedWhenCasting verifies that TransitionToEngaging
// SUCCEEDS while the character is Casting. Per the chunk-3 per-activity
// policy (Q2C from brainstorm — covers AC-018), casting is itself a
// combat action and is exempt from the activity_self veto in
// CombatPhase_Vetoes.go. The cast continues through combat entry;
// damage triggers a separate concentration-break path. Only Crafting
// and Salvaging block combat entry.
func TestCombatEntryAllowedWhenCasting(t *testing.T) {
	c := characters.New()
	_ = c.Activity.TransitionToCasting(
		activity.CastingData{SpellId: "fireball"},
		state.TransitionReason{Trigger: activity.TriggerCastBegin},
	)
	if !c.IsCasting() {
		t.Fatal("setup: expected character to be casting")
	}

	err := c.CombatPhase.TransitionToEngaging(
		combatphase.EngagingData{Target: state.ActorRef{UserId: 999}},
		state.TransitionReason{Trigger: combatphase.TriggerAttackCommand},
	)

	if err != nil {
		t.Errorf("TransitionToEngaging should succeed while casting; got %v", err)
	}
	if !c.IsCasting() {
		t.Errorf(
			"Activity should remain Casting after combat entry (no cascade-cancel for casting); got %v",
			c.Activity.State(),
		)
	}
}
