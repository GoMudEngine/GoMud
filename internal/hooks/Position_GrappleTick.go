// Position_GrappleTick.go fires once per round per active grapple
// pair. For each character that's currently a controller (IsController),
// it looks up the partner via GrappleData.Partner, validates the pair,
// rolls an opposed Strength+grappling check, dispatches the result
// through position.ResolveOutcome, and applies the resulting transition
// (advance / degrade / reversal / escape / hold) via TransitionPair.
//
// Iterates the controller side only — the controlled side is processed
// transitively through its controller. Skips solo Turtles (no partner)
// and any pair that fails ValidateGrapplePair (logged; consistency
// checker will force-break in T8).
//
// Chunk 4b-fixup: replaces the chunk-4b ControlLevel drift-needle math
// with outcome-driven dispatch. ControlLevel is sunset in T18.
// Messaging (emitOutcomeMessages) is wired in T16.
package hooks

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// processGrappleTick is the NewRound event listener. Iterates all
// active players + mobs; for each character that's a controller,
// resolves the partner and processes the pair.
func processGrappleTick(e events.Event) events.ListenerReturn {
	for _, u := range users.GetAllActiveUsers() {
		if u == nil || u.Character == nil {
			continue
		}
		if !u.Character.IsController() {
			continue
		}
		partner := resolvePartner(u.Character)
		if partner == nil {
			continue
		}
		if err := position.ValidateGrapplePair(u.Character, partner); err != nil {
			mudlog.Warn("Position_GrappleTick: invalid pair", "user", u.UserId, "err", err)
			continue
		}
		processGrapplePair(u.Character, partner)
	}
	for _, mobInstId := range mobs.GetAllMobInstanceIds() {
		m := mobs.GetInstance(mobInstId)
		if m == nil {
			continue
		}
		if !m.Character.IsController() {
			continue
		}
		partner := resolvePartner(&m.Character)
		if partner == nil {
			continue
		}
		if err := position.ValidateGrapplePair(&m.Character, partner); err != nil {
			mudlog.Warn("Position_GrappleTick: invalid pair", "mob", m.InstanceId, "err", err)
			continue
		}
		processGrapplePair(&m.Character, partner)
	}
	return events.Continue
}

// resolvePartner looks up a character's grapple partner via
// GrappleData. Returns nil if there's no partner reference or the
// partner can't be found in the live user/mob tables.
func resolvePartner(c *characters.Character) *characters.Character {
	if c.Position == nil {
		return nil
	}
	d, ok := c.Position.GrappleData()
	if !ok || d.Partner.IsZero() {
		return nil
	}
	if d.Partner.UserId > 0 {
		if u := users.GetByUserId(d.Partner.UserId); u != nil {
			return u.Character
		}
	}
	if d.Partner.MobInstanceId > 0 {
		if m := mobs.GetInstance(d.Partner.MobInstanceId); m != nil {
			return &m.Character
		}
	}
	return nil
}

// processGrapplePair runs the per-round drift roll, dispatches the
// result through position.ResolveOutcome, applies the resulting
// transition (advance / degrade / reversal / escape / hold), updates
// the LastDriftRoll snapshot for chunk-4d submission tick, and
// applies stamina cost.
//
// Chunk 4b-fixup: replaces the chunk-4b ControlLevel drift-needle
// math. ControlLevel is sunset entirely; the per-round outcome IS
// the position change.
func processGrapplePair(controller, controlled *characters.Character) {
	cfg := configs.GetBalanceConfig()

	// Score formula unchanged from chunk 4b:
	//   controller: (Str + WeaponCombat) × stamina × encumbrance
	//   controlled: (Str + WeaponCombat + 0.5·Dex + body.EscapeModifier)
	//               × stamina × encumbrance
	ctrlBase := float64(controller.Stats.Strength.Value) +
		float64(controller.GetSkillLevel(skills.WeaponCombat))
	cdBase := float64(controlled.Stats.Strength.Value) +
		float64(controlled.GetSkillLevel(skills.WeaponCombat)) +
		0.5*float64(controlled.Stats.Dexterity.Value) +
		escapeModifierFromBody(controlled)

	ctrlScore := ctrlBase *
		grappleStaminaMultiplier(controller, cfg) *
		grappleEncumbranceMultiplier(controller, cfg)
	cdScore := cdBase *
		grappleStaminaMultiplier(controlled, cfg) *
		grappleEncumbranceMultiplier(controlled, cfg)

	_, margin, atkRoll, defRoll := dice.OpposedRollStat(ctrlScore, cdScore)

	// LastDriftRoll snapshot for chunk-4d Position_SubmissionTick.
	currentRound := util.GetRoundCount()
	snap := characters.DriftRollSnapshot{
		Round:          currentRound,
		MarginAttacker: margin,
		AttackerZScore: atkRoll.ZScore,
		DefenderZScore: defRoll.ZScore,
	}
	controller.LastDriftRoll = snap
	controlled.LastDriftRoll = snap

	// Compute signed z used by ResolveOutcome.
	z := 0.0
	if atkRoll.StdDev > 0 {
		z = margin / atkRoll.StdDev
	}

	source := controller.Position.State()
	defenderPosture := controlled.Position.State()
	outcome := position.ResolveOutcome(source, z, defenderPosture)

	// Apply outcome via TransitionPair when position changes.
	switch outcome.Kind {
	case position.OutcomeAdvance:
		applyAdvanceOrEscape(controller, controlled, outcome.Target,
			position.TriggerPositionAdvance)
	case position.OutcomeDegrade:
		applyAdvanceOrEscape(controller, controlled, outcome.Target,
			position.TriggerPositionDegrade)
	case position.OutcomeReversal:
		applyReversal(controller, controlled, outcome.Target)
	case position.OutcomeEscape:
		applyAdvanceOrEscape(controller, controlled, position.Standing,
			position.TriggerControlledEscape)
		// Clear per-grapple cooldowns on full escape — next grapple
		// starts fresh.
		controller.PerGrappleMessageCooldowns = map[string]bool{}
		controlled.PerGrappleMessageCooldowns = map[string]bool{}
	case position.OutcomeHold:
		// No transition. Stamina drains; flavor handled below.
	}

	// Stamina cost unchanged.
	applyGrappleStaminaCost(controller, controlled, cfg)
	fireStaminaWarningIfLow(controller)
	fireStaminaWarningIfLow(controlled)

	// Messaging — T16 wires this. Stub for now so the test in T15
	// asserts call-through without rendering.
	emitOutcomeMessages(controller, controlled, outcome)
}

// applyAdvanceOrEscape fires position.TransitionPair with controller
// and controlled in their existing roles. Used for advances,
// degrades, and full escapes (which all keep the role assignment).
func applyAdvanceOrEscape(controller, controlled *characters.Character,
	target position.State, trigger string) {
	if err := position.TransitionPair(controller, controlled, target,
		state.TransitionReason{Trigger: trigger}); err != nil {
		mudlog.Warn("Position_GrappleTick: TransitionPair failed",
			"controller_user", controller.GetUserId(),
			"controller_mob", controller.GetMobInstanceId(),
			"target", target, "trigger", trigger, "err", err)
	}
}

// applyReversal swaps roles when transitioning. The former defender
// becomes the new controller; former controller becomes the new
// controlled. position.TransitionPair takes (controller, controlled)
// args, so we swap them at the call site.
func applyReversal(formerController, formerControlled *characters.Character,
	target position.State) {
	if err := position.TransitionPair(formerControlled, formerController, target,
		state.TransitionReason{Trigger: position.TriggerReversal}); err != nil {
		mudlog.Warn("Position_GrappleTick: reversal TransitionPair failed",
			"former_controller_user", formerController.GetUserId(),
			"former_controlled_user", formerControlled.GetUserId(),
			"target", target, "err", err)
	}
}

// emitOutcomeMessages is wired to grapplemessaging.RenderOutcome
// in T16. Stub here so T15 compiles + runs.
func emitOutcomeMessages(controller, controlled *characters.Character,
	outcome position.Outcome) {
	// Wired in T16.
}

// grappleStaminaMultiplier returns a penalty multiplier in (1-max, 1].
// Steeper than the combat ResourcePenalty curve: grappling cardio-
// stresses faster than stand-up fighting (default 0.60 max vs 0.28).
func grappleStaminaMultiplier(c *characters.Character, cfg configs.Balance) float64 {
	if c.StaminaMax.Value <= 0 {
		return 1.0
	}
	s := float64(c.Stamina) / float64(c.StaminaMax.Value)
	if s < 0 {
		s = 0
	}
	if s > 1 {
		s = 1
	}
	maxPen := float64(cfg.GrappleStaminaPenaltyMax)
	curve := float64(cfg.GrappleStaminaPenaltyCurve)
	return 1.0 - maxPen*math.Pow(1.0-s, curve)
}

// grappleEncumbranceMultiplier returns a penalty multiplier in
// (1-max, 1]. No penalty at ≤50% load; ramps to max at ≥200% (crushed).
func grappleEncumbranceMultiplier(c *characters.Character, cfg configs.Balance) float64 {
	cap := c.CarryCapacity()
	if cap <= 0 {
		return 1.0
	}
	load := c.GetCarriedWeight()
	e := load / cap
	if e < 0 {
		e = 0
	}
	threshold := e - 0.5
	if threshold < 0 {
		return 1.0
	}
	normalized := threshold / 1.5
	if normalized > 1 {
		normalized = 1
	}
	maxPen := float64(cfg.GrappleEncumbrancePenaltyMax)
	curve := float64(cfg.GrappleEncumbrancePenaltyCurve)
	return 1.0 - maxPen*math.Pow(normalized, curve)
}

// escapeModifierFromBody reads the controlled character's body slot
// armor for the EscapeModifier field on ItemSpec. Mirrors the legacy
// CheckGroundedEscape helper from chunk 2.
func escapeModifierFromBody(c *characters.Character) float64 {
	bodyItem := c.Equipment.Body
	if bodyItem.ItemId == 0 {
		return 0.0
	}
	spec := items.GetItemSpec(bodyItem.ItemId)
	if spec == nil {
		return 0.0
	}
	return spec.EscapeModifier
}

// applyGrappleStaminaCost deducts asymmetric per-round stamina from
// both sides. Cost can drive stamina to 0; the character keeps
// grappling (the penalty curve maxes out, which is the intended
// "smother" feedback loop).
func applyGrappleStaminaCost(controller, controlled *characters.Character, cfg configs.Balance) {
	base := float64(cfg.GrappleStaminaCostPerRound)
	ctrlCost := int(math.Round(base * float64(cfg.GrappleControllerCostMultiplier)))
	cdCost := int(math.Round(base * float64(cfg.GrappleControlledCostMultiplier)))
	controller.Stamina -= ctrlCost
	if controller.Stamina < 0 {
		controller.Stamina = 0
	}
	controlled.Stamina -= cdCost
	if controlled.Stamina < 0 {
		controlled.Stamina = 0
	}
}

// fireGradientMessages / fireTransitionMessages / fireStaminaWarningIfLow
// live in Position_Messaging.go (T7).

func init() {
	events.RegisterListener(events.NewRound{}, processGrappleTick)
}
