package activity_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
)

// --- AC-001 through AC-012: Basic transitions ---

func TestAC_001_FreeToCasting(t *testing.T) {
	m := activity.NewMachine()
	err := m.TransitionToCasting(
		activity.CastingData{SpellId: "fireball", FoldsNeeded: 3},
		state.TransitionReason{Trigger: activity.TriggerCastBegin},
	)
	if err != nil {
		t.Fatalf("expected transition to succeed, got %v", err)
	}
	if m.State() != activity.Casting {
		t.Errorf("expected Casting, got %v", m.State())
	}
}

func TestAC_002_FreeToCrafting(t *testing.T) {
	m := activity.NewMachine()
	err := m.TransitionToCrafting(
		activity.CraftingData{RecipeId: "iron_dagger", RoundsTotal: 4},
		state.TransitionReason{Trigger: activity.TriggerCraftBegin},
	)
	if err != nil {
		t.Fatalf("expected transition to succeed, got %v", err)
	}
	if m.State() != activity.Crafting {
		t.Errorf("expected Crafting, got %v", m.State())
	}
}

func TestAC_003_FreeToSalvaging(t *testing.T) {
	m := activity.NewMachine()
	err := m.TransitionToSalvaging(
		activity.SalvagingData{ItemUuid: "abc-123", RoundsTotal: 3},
		state.TransitionReason{Trigger: activity.TriggerSalvageBegin},
	)
	if err != nil {
		t.Fatalf("expected transition to succeed, got %v", err)
	}
	if m.State() != activity.Salvaging {
		t.Errorf("expected Salvaging, got %v", m.State())
	}
}

func TestAC_004_CastingToFreeOnComplete(t *testing.T) {
	m := activity.NewMachine()
	_ = m.TransitionToCasting(activity.CastingData{SpellId: "fireball"},
		state.TransitionReason{Trigger: activity.TriggerCastBegin})
	err := m.TransitionToFree(state.TransitionReason{Trigger: activity.TriggerCastComplete})
	if err != nil {
		t.Fatalf("expected transition to succeed, got %v", err)
	}
	if m.State() != activity.Free {
		t.Errorf("expected Free, got %v", m.State())
	}
}

func TestAC_005_CraftingToFreeOnComplete(t *testing.T) {
	m := activity.NewMachine()
	_ = m.TransitionToCrafting(activity.CraftingData{RecipeId: "iron_dagger"},
		state.TransitionReason{Trigger: activity.TriggerCraftBegin})
	err := m.TransitionToFree(state.TransitionReason{Trigger: activity.TriggerCraftComplete})
	if err != nil {
		t.Fatalf("expected transition to succeed, got %v", err)
	}
	if m.State() != activity.Free {
		t.Errorf("expected Free, got %v", m.State())
	}
}

func TestAC_006_SalvagingToFreeOnComplete(t *testing.T) {
	m := activity.NewMachine()
	_ = m.TransitionToSalvaging(activity.SalvagingData{ItemUuid: "abc-123"},
		state.TransitionReason{Trigger: activity.TriggerSalvageBegin})
	err := m.TransitionToFree(state.TransitionReason{Trigger: activity.TriggerSalvageComplete})
	if err != nil {
		t.Fatalf("expected transition to succeed, got %v", err)
	}
	if m.State() != activity.Free {
		t.Errorf("expected Free, got %v", m.State())
	}
}

func TestAC_007_CastingToFreeOnCancel(t *testing.T) {
	m := activity.NewMachine()
	_ = m.TransitionToCasting(activity.CastingData{SpellId: "fireball"},
		state.TransitionReason{Trigger: activity.TriggerCastBegin})
	err := m.TransitionToFree(state.TransitionReason{Trigger: activity.TriggerCastCancel})
	if err != nil {
		t.Fatalf("cancel should succeed, got %v", err)
	}
	if m.State() != activity.Free {
		t.Errorf("expected Free after cancel, got %v", m.State())
	}
}

func TestAC_008_CraftingToFreeOnCancel(t *testing.T) {
	m := activity.NewMachine()
	_ = m.TransitionToCrafting(activity.CraftingData{RecipeId: "iron_dagger"},
		state.TransitionReason{Trigger: activity.TriggerCraftBegin})
	err := m.TransitionToFree(state.TransitionReason{Trigger: activity.TriggerCraftCancel})
	if err != nil {
		t.Fatalf("cancel should succeed, got %v", err)
	}
	if m.State() != activity.Free {
		t.Errorf("expected Free after cancel, got %v", m.State())
	}
}

func TestAC_009_SalvagingToFreeOnCancel(t *testing.T) {
	m := activity.NewMachine()
	_ = m.TransitionToSalvaging(activity.SalvagingData{ItemUuid: "abc-123"},
		state.TransitionReason{Trigger: activity.TriggerSalvageBegin})
	err := m.TransitionToFree(state.TransitionReason{Trigger: activity.TriggerSalvageCancel})
	if err != nil {
		t.Fatalf("cancel should succeed, got %v", err)
	}
	if m.State() != activity.Free {
		t.Errorf("expected Free after cancel, got %v", m.State())
	}
}

func TestAC_010_StartsInFree(t *testing.T) {
	m := activity.NewMachine()
	if m.State() != activity.Free {
		t.Errorf("expected Free, got %v", m.State())
	}
	if !m.IsFree() {
		t.Errorf("IsFree() = false, want true")
	}
	if m.IsActing() {
		t.Errorf("IsActing() = true, want false")
	}
}

func TestAC_011_CastingDataAccessible(t *testing.T) {
	m := activity.NewMachine()
	_ = m.TransitionToCasting(
		activity.CastingData{SpellId: "fireball", FoldsNeeded: 5, ConvictionSpent: 12},
		state.TransitionReason{Trigger: activity.TriggerCastBegin},
	)
	d, ok := m.CastingData()
	if !ok {
		t.Fatal("expected casting data to be available")
	}
	if d.SpellId != "fireball" || d.FoldsNeeded != 5 || d.ConvictionSpent != 12 {
		t.Errorf("data lost in transition: %+v", d)
	}
}

func TestAC_012_DataClearedOnReturnToFree(t *testing.T) {
	m := activity.NewMachine()
	_ = m.TransitionToCasting(activity.CastingData{SpellId: "fireball"},
		state.TransitionReason{Trigger: activity.TriggerCastBegin})
	_ = m.TransitionToFree(state.TransitionReason{Trigger: activity.TriggerCastComplete})
	if _, ok := m.CastingData(); ok {
		t.Errorf("CastingData() should return ok=false when state is Free")
	}
}

// --- AC-013 through AC-024: Per-activity interrupt policy ---

func TestAC_013_CastingConcentrationBreakOnDamage(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

func TestAC_014_CraftingHardCancelOnDamage(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

func TestAC_015_SalvagingHardCancelOnDamage(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

func TestAC_016_CraftingCancelOnCombatEntry(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

func TestAC_017_SalvagingCancelOnCombatEntry(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

func TestAC_018_CastingNotInterruptedByCombatEntry(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

func TestAC_019_CraftingCancelOnMovement(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

func TestAC_020_SalvagingCancelOnMovement(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

func TestAC_021_CastingNotCanceledByMovement(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

func TestAC_022_ConvictionRefundOnCastCancel(t *testing.T) {
	t.Skip("integration test — verified in Task 9 cancel command tests")
}

func TestAC_023_NoRefundOnCraftCancel(t *testing.T) {
	t.Skip("integration test — verified in Task 9 cancel command tests")
}

func TestAC_024_NoRefundOnSalvageCancel(t *testing.T) {
	t.Skip("integration test — verified in Task 9 cancel command tests")
}

// --- AC-025 through AC-028: Cross-activity start veto ---

func TestAC_025_CastingToCastingFails(t *testing.T) {
	m := activity.NewMachine()
	_ = m.TransitionToCasting(activity.CastingData{SpellId: "fireball"},
		state.TransitionReason{Trigger: activity.TriggerCastBegin})
	err := m.TransitionToCasting(activity.CastingData{SpellId: "icebolt"},
		state.TransitionReason{Trigger: activity.TriggerCastBegin})
	if err == nil {
		t.Fatal("expected error transitioning Casting→Casting")
	}
	if m.State() != activity.Casting {
		t.Errorf("state should remain Casting on failed transition")
	}
}

func TestAC_026_CraftingToCastingFails(t *testing.T) {
	m := activity.NewMachine()
	_ = m.TransitionToCrafting(activity.CraftingData{RecipeId: "iron_dagger"},
		state.TransitionReason{Trigger: activity.TriggerCraftBegin})
	err := m.TransitionToCasting(activity.CastingData{SpellId: "fireball"},
		state.TransitionReason{Trigger: activity.TriggerCastBegin})
	if err == nil {
		t.Fatal("expected error transitioning Crafting→Casting directly")
	}
}

func TestAC_027_CastingToCraftingFails(t *testing.T) {
	m := activity.NewMachine()
	_ = m.TransitionToCasting(activity.CastingData{SpellId: "fireball"},
		state.TransitionReason{Trigger: activity.TriggerCastBegin})
	err := m.TransitionToCrafting(activity.CraftingData{RecipeId: "iron_dagger"},
		state.TransitionReason{Trigger: activity.TriggerCraftBegin})
	if err == nil {
		t.Fatal("expected error transitioning Casting→Crafting directly")
	}
}

func TestAC_028_SalvagingToCraftingFails(t *testing.T) {
	m := activity.NewMachine()
	_ = m.TransitionToSalvaging(activity.SalvagingData{ItemUuid: "abc-123"},
		state.TransitionReason{Trigger: activity.TriggerSalvageBegin})
	err := m.TransitionToCrafting(activity.CraftingData{RecipeId: "iron_dagger"},
		state.TransitionReason{Trigger: activity.TriggerCraftBegin})
	if err == nil {
		t.Fatal("expected error transitioning Salvaging→Crafting directly")
	}
}

// --- AC-029 through AC-034: Mob/player parity ---

func TestAC_029_MobCraftCancelsOnCombatEntry(t *testing.T) {
	t.Skip("integration test — verified in Task 5/7 cascade + migration tests")
}

func TestAC_030_PlayerCraftCancelsOnDamage(t *testing.T) {
	t.Skip("integration test — verified in Task 5/7 cascade + migration tests")
}

func TestAC_031_MobCraftCancelsOnDamage(t *testing.T) {
	t.Skip("integration test — verified in Task 5/7 cascade + migration tests")
}

func TestAC_032_PlayerCastConcentrationBreakSurvivable(t *testing.T) {
	t.Skip("integration test — verified in Task 5/6 cascade + migration tests")
}

func TestAC_033_MobCastConcentrationBreakSurvivable(t *testing.T) {
	t.Skip("integration test — verified in Task 5/6 cascade + migration tests")
}

func TestAC_034_BtreeCancelActivityTriggersFree(t *testing.T) {
	t.Skip("integration test — verified in Task 9 btree primitive tests")
}

// --- AC-035 through AC-038: Cascade verification ---

func TestAC_035_LifeDeadCascadesActivityFree(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

func TestAC_036_MovementCascadesCraftingFree(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

func TestAC_037_MovementCascadesSalvagingFree(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

func TestAC_038_CombatPhaseEngagingCascadesCraftingFree(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}
