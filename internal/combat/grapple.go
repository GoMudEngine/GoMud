package combat

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

// GrappleResult represents the outcome of a grapple attempt
type GrappleResult struct {
	Success         bool
	Margin          float64
	IsGroundGrapple bool // true when the new grapple position is a ground grapple (SideControl)
	AttackScore     float64
	DefenseScore    float64
	AttackRoll      float64
	DefenseRoll     float64
	PositionPenalty float64 // For defender if prone
	AttackZScore    float64 // For crit detection (Stage 8.4)
	DefenseZScore   float64 // For reference (Stage 8.4)
}

// AttemptGrapple performs a grapple attempt from attacker to defender.
// Returns a GrappleResult with the outcome and details.
//
// Grapple calculation:
// attackScore = attacker.Dex + attacker.CombatSkill + weapon.GrappleModifier
// defenseScore = defender.Dex + defender.CombatSkill
//
// Position modifiers:
// - If defender is prone: defenseScore *= 0.3  (-70% defense when already down)
// - If attacker is prone: attackScore *= 0.5   (-50% offense when attacking from ground)
//
// Position transitions on success:
// - Standing → Clinched
// - Prone → Grounded (direct, skip Clinched)
func AttemptGrapple(attacker *characters.Character, defender *characters.Character) GrappleResult {
	result := GrappleResult{}

	// Base scores: Dex + Combat Skill
	attackerCombatSkill := float64(attacker.GetCombatSkillLevel())
	defenderCombatSkill := float64(defender.GetCombatSkillLevel())

	// Check for 1-round grapple opportunity from prior dodge crit (Stage 8.4)
	opportunityBonus := GetGrappleOpportunityBonus(attacker)

	result.AttackScore = (float64(attacker.GetEffectiveDexterity()) + attackerCombatSkill) * opportunityBonus
	result.DefenseScore = float64(defender.GetEffectiveDexterity()) + defenderCombatSkill

	// Clear opportunity after use if it was active (Stage 8.4)
	if opportunityBonus > 1.0 {
		ClearGrappleOpportunity(attacker)
	}

	// Add weapon grapple modifier (if wielding a weapon)
	if attacker.Equipment.Weapon.ItemId != 0 {
		weaponSpec := items.GetItemSpec(attacker.Equipment.Weapon.ItemId)
		if weaponSpec != nil {
			result.AttackScore += weaponSpec.GrappleModifier
		}
	}

	// Position modifiers
	if defender.IsProne() || defender.IsSupine() {
		// Defender at -70% defense when already down (brutal!)
		result.PositionPenalty = -0.7
		result.DefenseScore *= 0.3
	}

	if attacker.IsProne() || attacker.IsSupine() {
		// Attacker at -50% offense when attacking from ground
		result.AttackScore *= 0.5
	}

	// Opposed roll
	success, margin, attackRoll, defenseRoll := dice.OpposedRollStat(result.AttackScore, result.DefenseScore)

	result.Success = success
	result.Margin = margin
	result.AttackRoll = attackRoll.Value
	result.DefenseRoll = defenseRoll.Value
	result.AttackZScore = attackRoll.ZScore   // Stage 8.4: For crit detection
	result.DefenseZScore = defenseRoll.ZScore // Stage 8.4: For reference

	// Determine whether the new grapple position is a ground grapple.
	if success {
		result.IsGroundGrapple = defender.IsProne() || defender.IsSupine()
	}

	return result
}

// ApplyGrappleResult applies the grapple result to both characters.
// Sets positions and tracks the grapple controller.
//
// Chunk 4b W1 cutover: fires position.TransitionPair onto the new
// FSM in parallel with the legacy CombatPosition + GrappleControllerId
// writes. If the pair transition fails (e.g. invalid source state for
// the chosen target) the legacy fields are NOT updated either, so the
// two views stay consistent across the migration window. Both writes
// disappear in S1 along with CombatPosition itself.
func ApplyGrappleResult(attacker *characters.Character, defender *characters.Character, result GrappleResult, attackerId int) {
	if !result.Success {
		return
	}

	// Pick the FSM target. Either side already on the ground lands the
	// pair directly into SideControl (skips Clinch); both standing →
	// Clinch.
	target := position.Clinch
	if attacker.IsProne() || attacker.IsSupine() ||
		defender.IsProne() || defender.IsSupine() {
		target = position.SideControl
	}

	if err := position.TransitionPair(
		attacker, defender, target,
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	); err != nil {
		mudlog.Warn("ApplyGrappleResult: TransitionPair failed",
			"attacker", attackerId, "target", target, "err", err)
		return
	}

	// Stage 8.3: Mark who is the controller (FSM-driven via IsController()).
	// ConditionGrappleController is sunset; controller identity lives in the
	// FSM GrappleData.ControlLevel field.

	// Chunk 4b-fixup-2 T5: mark attacker as aggressor for drift-roll
	// tiebreaker in symmetric positions.
	markAggressor(attacker)
}

// IsThirdPartyAttack returns true if the attacker is not involved in the target's grapple.
// This identifies opportunistic attackers targeting grappling fighters.
// Stage 8.5: Third-party grapple vulnerability.
//
// Chunk 4b R2: FSM-driven. Target must be in a grapple state and have a
// Partner recorded in its GrappleData; attacker is third-party iff its
// ActorRef does not match that Partner. Replaces the legacy
// CombatPosition.IsGrapplePosition + GrappleControllerId reads (both
// sunset in S3/S5).
func IsThirdPartyAttack(attacker *characters.Character, target *characters.Character) bool {
	if !target.IsGrappling() {
		return false
	}
	if target.Position == nil {
		return false
	}
	d, ok := target.Position.GrappleData()
	if !ok {
		return false
	}
	if d.Partner.IsZero() {
		// Solo Turtle is the only legal zero-Partner grapple state.
		// Preserve legacy behavior: "no controller ID" → not third
		// party. 4e refines this with proper solo-defender semantics.
		return false
	}
	attackerRef := state.ActorRef{
		UserId:        attacker.GetUserId(),
		MobInstanceId: attacker.GetMobInstanceId(),
	}
	return d.Partner != attackerRef
}

// markAggressor sets IsAggressor=true on the attacker side's
// GrappleData after a successful TransitionPair. Called from
// ApplyGrappleResult. Used as a tiebreaker for the drift roll's
// attacker-arg in symmetric positions (Clinch, HalfGuard, Turtle)
// where both sides start at the same ControlLevel state.
func markAggressor(attacker *characters.Character) {
	if attacker == nil || attacker.Position == nil {
		return
	}
	d, ok := attacker.Position.GrappleData()
	if !ok {
		return
	}
	d.IsAggressor = true
	attacker.Position.SetGrappleData(d)
}

// CritFailureResult represents the outcome of a critical grapple failure (Stage 8.6)
type CritFailureResult struct {
	Message       string // For attacker
	TargetMessage string // For defender
	RoomMessage   string // For observers
}

// HandleGrappleCritFailure handles the consequences of a critical grapple failure.
// Stage 8.6: Attacker falls prone, defender gets reversal opportunity (+15% grapple bonus).
//
// Triggered when grapple fails with z-score < -2.0 (~2-3% chance).
// Chunk 4b W8 cutover: attacker is Standing (grapple entry from
// Standing) and the failure aborts before pair formation, so a direct
// Standing → Prone transition is the right move. Legacy CombatPosition
// + PositionRoundsMin stay in sync via parallel-write until S1.
func HandleGrappleCritFailure(attacker *characters.Character, defender *characters.Character) CritFailureResult {
	result := CritFailureResult{}

	// Position can be nil on a mob instance that ResetForMobInstance() cleared
	// and hasn't re-wired yet (same nil state CalculatePositionString guards).
	// Skip the prone transition rather than panic — the fumble messaging below
	// still conveys the crit failure.
	if attacker.Position != nil {
		if err := attacker.Position.TransitionToProne(
			position.ProneData{MinRecoveryRounds: 2},
			state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward},
		); err != nil {
			mudlog.Warn("HandleGrappleCritFailure: TransitionToProne failed", "err", err)
		}
	}

	// Defender gets grapple opportunity (reuse existing system from Stage 8.4)
	SetGrappleOpportunity(defender)

	// Generate dramatic messages
	result.Message = `<ansi fg="red-bold">You overextend badly and fall to the ground!</ansi>`
	result.TargetMessage = `<ansi fg="yellow-bold">Your opponent overextends and falls - you see an opening!</ansi>`
	result.RoomMessage = `<ansi fg="combat">The failed grapple sends them sprawling!</ansi>`

	return result
}
