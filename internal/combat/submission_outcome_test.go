package combat_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/stretchr/testify/assert"
)

// setupMountForOutcome puts a + b in a Mount grapple with a as
// controller (InControl) and b as controlled (Controlled).
// Uses the proven TransitionPair → MutateGrappleControlLevel
// pattern from Position_SubmissionTick_test.go.
func setupMountForOutcome(t *testing.T) (controller, controlled *characters.Character) {
	t.Helper()
	a := characters.New()
	a.SetUserId(1)
	b := characters.New()
	b.SetUserId(2)

	a.Stats.Strength.Base = 100
	b.Stats.Strength.Base = 100
	a.Stats.Dexterity.Base = 100
	b.Stats.Dexterity.Base = 100
	a.Stamina = 100
	b.Stamina = 100
	a.StaminaMax.Base = 100
	b.StaminaMax.Base = 100
	a.StaminaMax.Value = 100
	b.StaminaMax.Value = 100

	if err := position.TransitionPair(a, b, position.Clinch,
		state.TransitionReason{Trigger: position.TriggerGrappleEntry}); err != nil {
		t.Fatalf("TransitionPair → Clinch failed: %v", err)
	}
	if err := position.TransitionPair(a, b, position.Mount,
		state.TransitionReason{Trigger: position.TriggerTakedownMount}); err != nil {
		t.Fatalf("TransitionPair → Mount failed: %v", err)
	}

	a.Position.MutateGrappleControlLevel(position.InControl)
	b.Position.MutateGrappleControlLevel(position.Controlled)

	return a, b
}

// setupMountWithMobDefender creates a Mount grapple where the
// defender is a mob character (UserId=0, MobInstanceId=1). This is
// required for death-cascade tests: Die() for players goes through the
// full Dead → Respawning → Alive cycle synchronously (leaving them
// alive again), while mobs stay in the Dead state after Die().
func setupMountWithMobDefender(t *testing.T) (controller, controlled *characters.Character) {
	t.Helper()
	a := characters.New()
	a.SetUserId(1)

	// Build a mob-like defender: UserId==0, MobInstanceId!=0.
	// characters.New() gives UserId==0 by default; set MobInstanceId
	// so Die() takes the mob path (Dead only, no respawn).
	b := characters.New()
	// UserId stays 0 (mob); set MobInstanceId so Die() skips respawn.
	b.MobInstanceId = 1
	b.IsMob = true

	a.Stats.Strength.Base = 100
	b.Stats.Strength.Base = 100
	a.Stats.Dexterity.Base = 100
	b.Stats.Dexterity.Base = 100
	a.Stamina = 100
	b.Stamina = 100
	a.StaminaMax.Base = 100
	b.StaminaMax.Base = 100
	a.StaminaMax.Value = 100
	b.StaminaMax.Value = 100

	if err := position.TransitionPair(a, b, position.Clinch,
		state.TransitionReason{Trigger: position.TriggerGrappleEntry}); err != nil {
		t.Fatalf("TransitionPair → Clinch failed: %v", err)
	}
	if err := position.TransitionPair(a, b, position.Mount,
		state.TransitionReason{Trigger: position.TriggerTakedownMount}); err != nil {
		t.Fatalf("TransitionPair → Mount failed: %v", err)
	}

	a.Position.MutateGrappleControlLevel(position.InControl)
	b.Position.MutateGrappleControlLevel(position.Controlled)

	return a, b
}

func TestResolveSubmissionOutcome_BadTierKnocksAttempterProne(t *testing.T) {
	atk, def := setupMountForOutcome(t)
	result := combat.SubmissionAttemptResult{
		Tier:    combat.SubTierBad,
		SubType: position.SubArmbar,
	}
	combat.ResolveSubmissionOutcome(atk, def, result, combat.RoleTop)
	assert.True(t, atk.IsProne(), "attempter should be prone after bad-tier sub")
	assert.False(t, atk.IsGrappling(), "pair should have broken after bad tier")
}

func TestResolveSubmissionOutcome_NeutralTierNoOp(t *testing.T) {
	atk, def := setupMountForOutcome(t)
	preState := atk.Position.State()
	result := combat.SubmissionAttemptResult{
		Tier:    combat.SubTierNeutral,
		SubType: position.SubArmbar,
	}
	combat.ResolveSubmissionOutcome(atk, def, result, combat.RoleTop)
	assert.Equal(t, preState, atk.Position.State(),
		"position should be unchanged on neutral tier")
}

func TestResolveSubmissionOutcome_SuccessMercyReleases(t *testing.T) {
	atk, def := setupMountForOutcome(t)
	atk.SubmissionPolicy = characters.PolicyMercy
	result := combat.SubmissionAttemptResult{
		Tier:    combat.SubTierSuccess,
		SubType: position.SubArmbar,
	}
	combat.ResolveSubmissionOutcome(atk, def, result, combat.RoleTop)
	assert.False(t, atk.IsGrappling(), "attacker should no longer be grappling on mercy")
	assert.False(t, def.IsGrappling(), "defender should no longer be grappling on mercy")
	assert.True(t, def.IsAlive(), "defender should be alive after mercy release")
}

func TestResolveSubmissionOutcome_SuccessSubdueKillsDefender(t *testing.T) {
	// Use a mob defender so Die() stays Dead (doesn't respawn).
	// Player characters go through Dead → Respawning → Alive
	// synchronously, so IsAlive() would return true after Die().
	atk, def := setupMountWithMobDefender(t)
	atk.SubmissionPolicy = characters.PolicySubdue
	result := combat.SubmissionAttemptResult{
		Tier:    combat.SubTierSuccess,
		SubType: position.SubArmbar,
	}
	combat.ResolveSubmissionOutcome(atk, def, result, combat.RoleTop)
	// Subdue: enters death cascade (NoDeprogression flag lands in T8).
	// For T7 verify Die() was called — mob stays Dead.
	assert.False(t, def.IsAlive(),
		"defender should enter death cascade on subdue policy")
}

func TestResolveSubmissionOutcome_SuccessCrippleArmbar(t *testing.T) {
	atk, def := setupMountWithMobDefender(t)
	atk.SubmissionPolicy = characters.PolicyCripple
	def.Gold = 100
	result := combat.SubmissionAttemptResult{
		Tier:    combat.SubTierSuccess,
		SubType: position.SubArmbar,
	}
	combat.ResolveSubmissionOutcome(atk, def, result, combat.RoleTop)
	// Cripple with arm sub: death cascade + broken-limb buff (T9 stub).
	// For T7: verify the cascade fired.
	assert.False(t, def.IsAlive(),
		"defender should enter death cascade on cripple+arm policy")
}

func TestResolveSubmissionOutcome_SuccessCrippleChokeDegradesToSubdue(t *testing.T) {
	atk, def := setupMountWithMobDefender(t)
	atk.SubmissionPolicy = characters.PolicyCripple
	// RNC is a choke — CrippleBodyPart returns ""; should degrade to subdue.
	result := combat.SubmissionAttemptResult{
		Tier:    combat.SubTierSuccess,
		SubType: position.SubRNC,
	}
	combat.ResolveSubmissionOutcome(atk, def, result, combat.RoleTop)
	// Choke can't break limbs → degrades to subdue; cascade still fires.
	assert.False(t, def.IsAlive(),
		"defender should enter death cascade when cripple degrades to subdue on choke")
}

func TestResolveSubmissionOutcome_SuccessLethal(t *testing.T) {
	atk, def := setupMountWithMobDefender(t)
	atk.SubmissionPolicy = characters.PolicyLethal
	result := combat.SubmissionAttemptResult{
		Tier:    combat.SubTierSuccess,
		SubType: position.SubArmbar,
	}
	combat.ResolveSubmissionOutcome(atk, def, result, combat.RoleTop)
	// Lethal: full death cascade (NoDeprogression=false, distinguished in T8).
	assert.False(t, def.IsAlive(),
		"defender should enter full death cascade on lethal policy")
}

func TestResolveSubmissionOutcome_CritMercyAppliesStunnedStub(t *testing.T) {
	atk, def := setupMountForOutcome(t)
	atk.SubmissionPolicy = characters.PolicyMercy
	result := combat.SubmissionAttemptResult{
		Tier:    combat.SubTierCrit,
		SubType: position.SubArmbar,
	}
	combat.ResolveSubmissionOutcome(atk, def, result, combat.RoleTop)
	// Crit + mercy: clean release; applyStunnedBuff is a T10 stub.
	// For T7: verify the mercy path still fires (defender alive, grapple broken).
	assert.False(t, atk.IsGrappling(), "grapple broken on crit+mercy")
	assert.True(t, def.IsAlive(), "defender alive after crit+mercy release")
}
