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

// TestActivityCascadeOnCombatEntryForCrafting verifies that entering
// combat (Idle→Engaging) cancels a Crafting activity. Covers AC-038.
func TestActivityCascadeOnCombatEntryForCrafting(t *testing.T) {
	c := characters.New()
	_ = c.Activity.TransitionToCrafting(
		activity.CraftingData{RecipeId: "iron_dagger", RoundsTotal: 4},
		state.TransitionReason{Trigger: activity.TriggerCraftBegin},
	)
	if !c.IsCrafting() {
		t.Fatal("setup: expected character to be crafting")
	}

	// TransitionToEngaging takes (EngagingData, TransitionReason).
	// No vetoes are registered in tests, so the transition succeeds.
	_ = c.CombatPhase.TransitionToEngaging(
		combatphase.EngagingData{Target: state.ActorRef{UserId: 999}},
		state.TransitionReason{Trigger: combatphase.TriggerAttackCommand},
	)

	if !c.IsFree() {
		t.Errorf(
			"Activity should cascade to Free on Combat Phase Engaging; got %v",
			c.Activity.State(),
		)
	}
}

// TestActivityCastingNotCanceledByCombatEntry verifies that entering
// combat (Idle→Engaging) does NOT cancel a Casting activity — casting
// is exempt per the per-activity interrupt policy. Covers AC-018.
func TestActivityCastingNotCanceledByCombatEntry(t *testing.T) {
	c := characters.New()
	_ = c.Activity.TransitionToCasting(
		activity.CastingData{SpellId: "fireball"},
		state.TransitionReason{Trigger: activity.TriggerCastBegin},
	)
	if !c.IsCasting() {
		t.Fatal("setup: expected character to be casting")
	}

	_ = c.CombatPhase.TransitionToEngaging(
		combatphase.EngagingData{Target: state.ActorRef{UserId: 999}},
		state.TransitionReason{Trigger: combatphase.TriggerAttackCommand},
	)

	if !c.IsCasting() {
		t.Errorf(
			"Casting should NOT cascade to Free on Combat Phase Engaging; got %v",
			c.Activity.State(),
		)
	}
}
