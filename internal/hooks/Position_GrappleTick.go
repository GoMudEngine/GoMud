// Position_GrappleTick.go fires once per round per active grapple
// pair. For each character that's currently a controller (IsController),
// it looks up the partner via GrappleData.Partner, validates the pair,
// rolls an opposed Strength+grappling check, computes ControlLevel
// drift (via position.MarginToDelta), applies asymmetric stamina cost,
// and fires a threshold transition when either side hits Controlled.
//
// Iterates the controller side only — the controlled side is processed
// transitively through its controller. Skips solo Turtles (no partner)
// and any pair that fails ValidateGrapplePair (logged; consistency
// checker will force-break in T8).
//
// Messaging hooks (fireGradientMessages / fireTransitionMessages /
// fireStaminaWarningIfLow) are no-op stubs here; Task 7 lands the real
// implementations in Position_Messaging.go so that work is purely
// additive (no changes to this file).
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

// processGrapplePair fires the opposed roll, computes drift, updates
// both sides' ControlLevels, applies asymmetric stamina cost, and
// triggers an escape transition when either side hits Controlled.
func processGrapplePair(controller, controlled *characters.Character) {
	cfg := configs.GetBalanceConfig()

	// Score formula:
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

	// OpposedRollStat returns (success, margin, attackRoll, defenseRoll).
	// We translate margin into a "margin z-score" using the attacker's
	// stddev (same value used to spread both rolls) so MarginToDelta's
	// |z| buckets map cleanly to the spec table.
	_, margin, atkRoll, _ := dice.OpposedRollStat(ctrlScore, cdScore)
	marginZ := 0.0
	if atkRoll.StdDev > 0 {
		marginZ = margin / atkRoll.StdDev
	}
	delta := position.MarginToDelta(math.Abs(marginZ))

	if delta == 0 {
		applyGrappleStaminaCost(controller, controlled, cfg)
		fireStaminaWarningIfLow(controller)
		fireStaminaWarningIfLow(controlled)
		return
	}

	// Positive margin = controller won → controlled drifts toward
	// Controlled (rank up), controller drifts toward InControl (rank down).
	var ctrlDelta, cdDelta int
	if margin > 0 {
		ctrlDelta = -delta
		cdDelta = +delta
	} else {
		ctrlDelta = +delta
		cdDelta = -delta
	}

	ctrlData, _ := controller.Position.GrappleData()
	cdData, _ := controlled.Position.GrappleData()

	newCtrl := position.ShiftControl(ctrlData.ControlLevel, ctrlDelta)
	newCd := position.ShiftControl(cdData.ControlLevel, cdDelta)

	updateControlLevel(controller, newCtrl)
	updateControlLevel(controlled, newCd)

	fireGradientMessages(controller, ctrlData.ControlLevel, newCtrl)
	fireGradientMessages(controlled, cdData.ControlLevel, newCd)

	applyGrappleStaminaCost(controller, controlled, cfg)
	fireStaminaWarningIfLow(controller)
	fireStaminaWarningIfLow(controlled)

	// Threshold check: either side hitting Controlled triggers escape
	// to the position's default escape target. Reset per-grapple
	// message cooldowns so the next grapple starts with a clean slate.
	if newCtrl == position.Controlled || newCd == position.Controlled {
		escapeTarget := position.DefaultEscapeTarget(controller.Position.State())
		_ = position.TransitionPair(
			controller, controlled,
			escapeTarget,
			state.TransitionReason{Trigger: position.TriggerControlThresholdCrossed},
		)
		controller.PerGrappleMessageCooldowns = map[string]bool{}
		controlled.PerGrappleMessageCooldowns = map[string]bool{}
		fireTransitionMessages(controller, controlled, escapeTarget)
	}
}

// updateControlLevel mutates the character's GrappleData ControlLevel
// without re-transitioning the FSM state. The transition table forbids
// e.g. Mount→Mount, so per-round control shifts can't go through
// TransitionTo. See position.Machine.MutateGrappleControlLevel.
func updateControlLevel(c *characters.Character, newLevel position.ControlLevel) {
	if c.Position == nil {
		return
	}
	c.Position.MutateGrappleControlLevel(newLevel)
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
