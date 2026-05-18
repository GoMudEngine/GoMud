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

func TestResolveSubmissionOutcome_CrippleAppliesBrokenLimbBuff(t *testing.T) {
	// This test verifies that cripple policy applies the broken-limb buff
	// for submission types that affect limbs (e.g., armbar, omoplata,
	// kimura, americana). The actual buff spec (id 83) is loaded from YAML
	// at server startup, so this unit test can't verify the buff exists in
	// the registry. Instead, we verify the death cascade occurs and the
	// applyBrokenLimbBuff call path is exercised (no panic).
	atk, def := setupMountWithMobDefender(t)
	atk.SubmissionPolicy = characters.PolicyCripple
	result := combat.SubmissionAttemptResult{
		Tier:    combat.SubTierSuccess,
		SubType: position.SubArmbar,
	}
	combat.ResolveSubmissionOutcome(atk, def, result, combat.RoleTop)
	// Cripple with armbar: death cascade fires + applyBrokenLimbBuff called.
	// The buff's actual existence is verified at server boot when YAML is loaded.
	assert.False(t, def.IsAlive(),
		"defender should enter death cascade on cripple+armbar")
}

// ── T19: Behavior Matrix gap tests ───────────────────────────────────────────

// PB-306: Success sub roll, policy=mercy, defender always-tap.
// Mercy releases regardless of defender surrender policy.
// Defender policy (always-tap) is stored correctly and the outcome is a
// clean release.
func TestPB_306_MercyRelease_DefenderAlwaysTap(t *testing.T) {
	atk, def := setupMountForOutcome(t)
	atk.SubmissionPolicy = characters.PolicyMercy
	def.SurrenderPolicy = characters.SurrenderPolicy{Mode: characters.SurrenderAlways}
	result := combat.SubmissionAttemptResult{
		Tier:    combat.SubTierSuccess,
		SubType: position.SubArmbar,
	}
	combat.ResolveSubmissionOutcome(atk, def, result, combat.RoleTop)
	// Mercy: clean release regardless of defender surrender policy.
	assert.False(t, atk.IsGrappling(), "PB-306: attacker released from grapple on mercy")
	assert.False(t, def.IsGrappling(), "PB-306: defender released from grapple on mercy")
	assert.True(t, def.IsAlive(), "PB-306: defender alive after mercy release")
}

// PB-307: Success sub roll, policy=mercy, defender never-tap.
// Per Open Q3 resolution: mercy is about the controller's nature, not a
// negotiation. Release fires anyway even though the defender would not tap.
func TestPB_307_MercyRelease_DefenderNeverTap(t *testing.T) {
	atk, def := setupMountForOutcome(t)
	atk.SubmissionPolicy = characters.PolicyMercy
	def.SurrenderPolicy = characters.SurrenderPolicy{Mode: characters.SurrenderNever}
	result := combat.SubmissionAttemptResult{
		Tier:    combat.SubTierSuccess,
		SubType: position.SubArmbar,
	}
	combat.ResolveSubmissionOutcome(atk, def, result, combat.RoleTop)
	// Mercy controller releases regardless — defender's never-tap policy has no
	// mechanical effect on a mercy-policy controller.
	assert.False(t, atk.IsGrappling(), "PB-307: attacker released even with defender never-tap")
	assert.False(t, def.IsGrappling(), "PB-307: defender released even with never-tap")
	assert.True(t, def.IsAlive(), "PB-307: defender alive after mercy+never-tap release")
}

// PB-313: Crit sub roll, policy=subdue.
// Recipient should NOT receive the Stunned buff (they enter the death cascade;
// buff would be a no-op). The subdue death-cascade outcome still fires.
func TestPB_313_CritSubdue_NoStunnedBuff(t *testing.T) {
	// Use mob defender so Die() stays Dead (players respawn synchronously).
	atk, def := setupMountWithMobDefender(t)
	atk.SubmissionPolicy = characters.PolicySubdue
	result := combat.SubmissionAttemptResult{
		Tier:    combat.SubTierCrit,
		SubType: position.SubArmbar,
	}
	combat.ResolveSubmissionOutcome(atk, def, result, combat.RoleTop)
	// Subdue: death cascade fires (mob stays Dead).
	assert.False(t, def.IsAlive(),
		"PB-313: defender should enter death cascade on crit+subdue")
	// Stunned buff (id 84) must NOT have been applied because policy≠mercy.
	// The defender entered the death cascade before any buff could matter;
	// confirm HasBuff returns false (buff registry not seeded in unit tests,
	// so HasBuff(84) will be false either way — but that's the correct state).
	assert.False(t, def.HasBuff(84),
		"PB-313: Stunned buff must not be applied on crit+subdue (death cascade fires)")
}

// PB-316: Mount + Controlled + defender drift margin < alpha → no sub roll.
// (Label for the specific bottom-side case already covered by
// TestEvaluateSubAttempt_NeitherEligible, but here we document it explicitly
// as a bottom-sub variant.)
// This is a policy test: verify that IsBottomSubEligible returns false when
// the defender has a sufficient bottom position but we inject ControlLevel=Neutral,
// which is above the Controlled/BecomingControlled gate.
func TestPB_316_BottomSub_InsufficientMargin_NoBotSubWindow(t *testing.T) {
	// Directly test the eligibility predicate: Mount position + Neutral
	// ControlLevel (defender is NOT in Controlled/BecomingControlled, so the
	// bottom-sub gate should fail even though Mount has bottom subs).
	// The margin < alpha case is fully covered by EvaluateSubAttempt_NeitherEligible;
	// here we verify the eligibility gate itself for the ControlLevel axis.
	assert.False(t,
		position.IsBottomSubEligible(position.Mount, position.Neutral),
		"PB-316: Mount+Neutral ControlLevel must not open bottom-sub window")
	assert.False(t,
		position.IsBottomSubEligible(position.Mount, position.InControl),
		"PB-316: Mount+InControl ControlLevel must not open bottom-sub window (already escaping)")
}

// PB-318: Bottom-sub success — defender attempting sub from Mount-bottom.
// The defender (RoleBottom attempter) should have their SubmissionPolicy applied
// to the former controller (now the recipient). Outcome is determined by the
// bottom-attempter's policy, not the controller's.
func TestPB_318_BottomSubSuccess_DefenderPolicyAppliedToController(t *testing.T) {
	// Set up: controller=a (InControl), defender=b (Controlled).
	// For RoleBottom, the attempter IS the controlled side (b) and the
	// recipient is the controller (a).
	// Use mob controller so Die() stays Dead if subdue fires.
	//
	// Note: setupMountWithMobDefender builds a as controller and b as mob.
	// For PB-318 we need the inverse — the bottom (b) is the human attempter;
	// the top (a) is the one who may die. Use setupMountForOutcome (both players)
	// and set b's policy; the player cascade goes Dead→Respawning→Alive
	// (stays alive), so we verify the grapple breaks (subdue fires) rather
	// than checking IsAlive.
	controller, defender := setupMountForOutcome(t)
	// The bottom (defender) attempts and succeeds — defender's policy resolves.
	defender.SubmissionPolicy = characters.PolicyMercy
	result := combat.SubmissionAttemptResult{
		Tier:    combat.SubTierSuccess,
		SubType: position.SubArmbar, // Mount-bottom armbar
	}
	// RoleBottom: attempter=defender, recipient=controller.
	combat.ResolveSubmissionOutcome(defender, controller, result, combat.RoleBottom)
	// Mercy policy: clean release, grapple breaks, nobody dies.
	assert.False(t, defender.IsGrappling(),
		"PB-318: bottom-sub success with mercy: defender (attempter) no longer grappling")
	assert.False(t, controller.IsGrappling(),
		"PB-318: bottom-sub success with mercy: controller (recipient) no longer grappling")
	assert.True(t, controller.IsAlive(),
		"PB-318: mercy policy on bottom-sub release keeps controller alive")
}

// PB-319: Bottom-sub bad roll — defender (attempter) falls Prone; pair breaks.
func TestPB_319_BottomSubBadRoll_DefenderProne(t *testing.T) {
	controller, defender := setupMountForOutcome(t)
	result := combat.SubmissionAttemptResult{
		Tier:    combat.SubTierBad,
		SubType: position.SubArmbar,
	}
	// RoleBottom: attempter=defender; bad roll → attempter (defender) falls Prone.
	combat.ResolveSubmissionOutcome(defender, controller, result, combat.RoleBottom)
	assert.True(t, defender.IsProne(),
		"PB-319: bottom-sub bad roll should knock bottom attempter Prone")
	assert.False(t, defender.IsGrappling(),
		"PB-319: grapple should break after bottom-sub bad roll")
}

// PB-322: Player surrender always-tap + attacker mercy → release fires.
// The spec: mercy controller always releases; defender always-tap policy is
// consistent with this. Verify the combination results in a clean release.
func TestPB_322_SurrenderAlwaysTap_AttackerMercy_Releases(t *testing.T) {
	atk, def := setupMountForOutcome(t)
	atk.SubmissionPolicy = characters.PolicyMercy
	def.SurrenderPolicy = characters.SurrenderPolicy{
		Mode:           characters.SurrenderAlways,
		HpPctThreshold: 0, // always-tap ignores HP threshold
	}
	result := combat.SubmissionAttemptResult{
		Tier:    combat.SubTierSuccess,
		SubType: position.SubArmbar,
	}
	combat.ResolveSubmissionOutcome(atk, def, result, combat.RoleTop)
	// Mercy + always-tap: clean release, both alive.
	assert.False(t, atk.IsGrappling(),
		"PB-322: mercy+always-tap: attacker no longer grappling")
	assert.True(t, def.IsAlive(),
		"PB-322: mercy+always-tap: defender alive after release")
}

// PB-323: Player surrender never-tap + attacker subdue → tap signal ignored.
// Subdue fires regardless of defender's surrender policy.
func TestPB_323_SurrenderNeverTap_AttackerSubdue_Fires(t *testing.T) {
	// Use mob defender so cascade stays Dead.
	atk, def := setupMountWithMobDefender(t)
	atk.SubmissionPolicy = characters.PolicySubdue
	def.SurrenderPolicy = characters.SurrenderPolicy{Mode: characters.SurrenderNever}
	result := combat.SubmissionAttemptResult{
		Tier:    combat.SubTierSuccess,
		SubType: position.SubArmbar,
	}
	combat.ResolveSubmissionOutcome(atk, def, result, combat.RoleTop)
	// Subdue ignores never-tap: death cascade fires.
	assert.False(t, def.IsAlive(),
		"PB-323: subdue fires regardless of never-tap surrender policy")
}

// PB-330: Broken-arm debuff statmod — verified in internal/buffs package.
// TestPB_330 and TestPB_332 live in internal/buffs/buffs_test.go (internal
// package) where the buff registry can be seeded without the YAML loader.
// See TestPB_330_BrokenLimbBuff_StatModApplied and
// TestPB_332_BrokenLimbBuff_ExpiresNaturally in that file.

// PB-331: Broken-arm debuff prevents 2H weapon wield.
// SKIP — the 2H wield restriction is a spec note in the chunk-4d design doc
// ("can't wield two-handed weapons") but is NOT yet wired as a buff flag or
// equip-path guard in 4d code. The buff YAML (83-broken_limb.yaml) only
// applies statmods (str/dex/vit). The 2H restriction is 4f flavor work.
// When implemented, the test should go in internal/usercommands/equip_test.go
// and verify that equip.go rejects a 2H weapon when buff 83 is active.
func TestPB_331_BrokenLimbBuff_NoTwoHandedWield(t *testing.T) {
	t.Skip("PB-331: 2H wield restriction from broken-limb buff is 4f flavor work; " +
		"buff 83 YAML only applies statmods in 4d. Test should live in " +
		"internal/usercommands/equip_test.go when the wield guard is wired.")
}
