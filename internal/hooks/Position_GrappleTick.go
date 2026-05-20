// Position_GrappleTick.go fires once per round per active grapple
// pair. For each character that's currently grappling, it looks up
// the partner via GrappleData.Partner, validates the pair, rolls an
// opposed Strength+grappling check, dispatches the result through
// position.ResolveOutcome, and applies the resulting transition
// (advance / degrade / reversal / escape / hold) via TransitionPair.
//
// Chunk 4b-fixup-2 T8: replaces the IsController()-filtered single-side
// iteration with pair-aware deduplication. Both sides of the pair are
// marked seen so the partner loop skips them — each pair is processed
// exactly once per round. This fixes the symmetric Clinch regression
// where IsController() returned false for both sides and the drift roll
// never fired.
//
// The drift-roll attacker-arg is chosen by determineDriftAttacker:
// priority is Control state (most Controlling-leaning wins), then
// IsAggressor, then iteration order.
//
// Chunk 4b-fixup: replaces the chunk-4b ControlLevel drift-needle math
// with outcome-driven dispatch. ControlLevel is sunset in T18.
// Messaging (emitOutcomeMessages) is wired in T16.
package hooks

import (
	"fmt"
	"math"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/grapplemessaging"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/control"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

var (
	// grappleOutcomesLib holds the parsed template library. Loaded
	// lazily on first use via grappleLibOnce — avoids mudlog nil-pointer
	// panics during test package init (mudlog.SetupLogger hasn't run yet).
	grappleOutcomesLib *grapplemessaging.Library
	grappleLibOnce     sync.Once
)

// loadGrappleLib ensures the library is loaded exactly once. Callers
// that need the library should call this; after the first call the
// sync.Once is a cheap no-op.
func loadGrappleLib() *grapplemessaging.Library {
	grappleLibOnce.Do(func() {
		lib, err := grapplemessaging.Load(
			"_datafiles/world/dogmud/messaging/grapple_outcomes.yaml")
		if err != nil {
			mudlog.Error("Position_GrappleTick: failed to load grapple_outcomes.yaml",
				"err", err)
			// Use an empty library so render calls return debug strings.
			lib = &grapplemessaging.Library{
				Advancements: map[string]grapplemessaging.TemplateTriad{},
				Degradations: map[string]grapplemessaging.TemplateTriad{},
				Reversals:    map[string]grapplemessaging.TemplateTriad{},
				Escapes:      map[string]grapplemessaging.TemplateTriad{},
				Holds:        map[string]grapplemessaging.TemplateTriad{},
				StrikingApex: map[string][]string{},
			}
		}
		errs := grapplemessaging.ValidateCompleteness(lib)
		for _, e := range errs {
			mudlog.Warn("Position_GrappleTick: grapple_outcomes.yaml incomplete",
				"violation", e)
		}
		grappleOutcomesLib = lib
	})
	return grappleOutcomesLib
}

// determineDriftAttacker picks which side is the drift-roll's
// attacker-arg. Priority:
//  1. Whoever has the more controller-leaning Control state.
//  2. Tiebreaker: whoever has IsAggressor=true.
//  3. Final fallback: lhs (the iteration order side).
func determineDriftAttacker(lhs, rhs *characters.Character) (attacker, defender *characters.Character) {
	if lhs.Control == nil || rhs.Control == nil {
		return lhs, rhs
	}
	lhsRank := controlRank(lhs.Control.State())
	rhsRank := controlRank(rhs.Control.State())
	if lhsRank < rhsRank {
		return lhs, rhs
	}
	if rhsRank < lhsRank {
		return rhs, lhs
	}
	// Tie: prefer aggressor
	if lhs.Position != nil {
		if d, ok := lhs.Position.GrappleData(); ok && d.IsAggressor {
			return lhs, rhs
		}
	}
	if rhs.Position != nil {
		if d, ok := rhs.Position.GrappleData(); ok && d.IsAggressor {
			return rhs, lhs
		}
	}
	// Last resort: caller order
	return lhs, rhs
}

// controlRank maps control states to a priority score for attacker
// selection. 0 = Controlling (most attacker-favored), 1 = Neutral,
// 2 = Controlled. Transient states get the rank of their nearest
// stable state (defensive — transients shouldn't be observed between
// ticks).
func controlRank(s control.State) int {
	switch s {
	case control.Controlling, control.LosingControl:
		return 0
	case control.Neutral:
		return 1
	case control.BecomingControlled, control.Controlled:
		return 2
	}
	return 1
}

// processGrapplePairFromIteration is called by processGrappleTick;
// determines which side is the controller arg, then delegates to
// processGrapplePair (existing).
func processGrapplePairFromIteration(lhs, rhs *characters.Character) {
	controller, controlled := determineDriftAttacker(lhs, rhs)
	processGrapplePair(controller, controlled)
}

// processGrappleTick is the NewRound event listener. Iterates all
// active players + mobs; for each character that's grappling,
// resolves the partner and processes the pair exactly once (deduped
// via a seen map).
//
// Chunk 4b-fixup-2 T8: replaced the IsController() filter with
// pair-aware deduplication. Both sides are marked seen when a pair
// is processed so the partner's loop entry is skipped. This fixes
// symmetric Clinch where IsController() returned false for both sides.
func processGrappleTick(e events.Event) events.ListenerReturn {
	seen := map[state.ActorRef]bool{}

	for _, u := range users.GetAllActiveUsers() {
		if u == nil || u.Character == nil {
			continue
		}
		myRef := state.ActorRef{UserId: u.UserId}
		if seen[myRef] {
			continue
		}
		if u.Character.Position == nil {
			continue
		}
		if !u.Character.Position.IsGrappling() {
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
		partnerRef := state.ActorRef{
			UserId:        partner.GetUserId(),
			MobInstanceId: partner.GetMobInstanceId(),
		}
		seen[myRef] = true
		seen[partnerRef] = true

		processGrapplePairFromIteration(u.Character, partner)
	}

	for _, mobInstId := range mobs.GetAllMobInstanceIds() {
		m := mobs.GetInstance(mobInstId)
		if m == nil {
			continue
		}
		myRef := state.ActorRef{MobInstanceId: m.InstanceId}
		if seen[myRef] {
			continue
		}
		if m.Character.Position == nil {
			continue
		}
		if !m.Character.Position.IsGrappling() {
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
		partnerRef := state.ActorRef{
			UserId:        partner.GetUserId(),
			MobInstanceId: partner.GetMobInstanceId(),
		}
		seen[myRef] = true
		seen[partnerRef] = true

		processGrapplePairFromIteration(&m.Character, partner)
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

	// Score formula (spec 2026-05-19 §3):
	//   score = (0.7·Str + 0.3·Dex + skill_coef·UnarmedCombat)
	//           × stamina_mult × encumbrance_mult
	//   skill_coef = 2.2 (aggressor) or 2.0 (defender)
	//
	// IsAggressor is set on GrappleData by ApplyGrappleResult's
	// markAggressor call (chunk 4b-fixup-2 T5). It persists for the
	// lifetime of the grapple — even after reversals, the original
	// initiator keeps the bonus.
	ctrlScore := grappleScore(controller, isAggressorSide(controller), cfg)
	cdScore := grappleScore(controlled, isAggressorSide(controlled), cfg)

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

	// Chunk 4b-fixup-2 T9: apply ControlLevel shifts based on drift z.
	applyControlShift(controller, controlled, z)

	// Messaging — wired in T16.
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

// emitOutcomeMessages picks a template triad for the outcome and
// sends the appropriate speaker variant to each side + the room.
// Cooldown map lives on Character.PerGrappleMessageCooldowns;
// PickTemplate handles within-grapple variety.
func emitOutcomeMessages(controller, controlled *characters.Character,
	outcome position.Outcome) {
	lib := loadGrappleLib()
	if lib == nil {
		return
	}

	// Initialize cooldown maps if nil (chunk-4b created them on
	// transitions; defensive init here).
	if controller.PerGrappleMessageCooldowns == nil {
		controller.PerGrappleMessageCooldowns = map[string]bool{}
	}
	if controlled.PerGrappleMessageCooldowns == nil {
		controlled.PerGrappleMessageCooldowns = map[string]bool{}
	}

	// Pick template triad based on outcome kind + source/target.
	var (
		triad grapplemessaging.TemplateTriad
		key   string
		found bool
	)

	switch outcome.Kind {
	case position.OutcomeAdvance:
		key = stateMessagingName(outcome.Source) + "_to_" + stateMessagingName(outcome.Target)
		triad, found = lib.Advancements[key]
	case position.OutcomeDegrade:
		key = stateMessagingName(outcome.Source) + "_to_" + stateMessagingName(outcome.Target)
		triad, found = lib.Degradations[key]
	case position.OutcomeReversal:
		key = stateMessagingName(outcome.Source) + "_reverse"
		triad, found = lib.Reversals[key]
		if !found {
			key = "generic_reverse"
			triad, found = lib.Reversals[key]
		}
	case position.OutcomeEscape:
		key = "generic_escape"
		triad, found = lib.Escapes[key]
	case position.OutcomeHold:
		emitHoldFlavor(controller, controlled, outcome)
		emitStrikingApexFlavor(controller, controlled, outcome)
		return
	}

	if !found {
		mudlog.Warn("Position_GrappleTick: missing message key",
			"kind", outcome.Kind, "key", key)
		return
	}

	controllerName := characterDisplayName(controller)
	controlledName := characterDisplayName(controlled)

	if msg := grapplemessaging.PickTemplate(triad.Controller,
		controller.PerGrappleMessageCooldowns, key+":ctrl"); msg != "" {
		sendToCharacter(controller,
			grapplemessaging.RenderTemplate(msg, controllerName, controlledName))
	}
	if msg := grapplemessaging.PickTemplate(triad.Controlled,
		controlled.PerGrappleMessageCooldowns, key+":cd"); msg != "" {
		sendToCharacter(controlled,
			grapplemessaging.RenderTemplate(msg, controllerName, controlledName))
	}
	if msg := grapplemessaging.PickTemplate(triad.Observers,
		controller.PerGrappleMessageCooldowns, key+":obs"); msg != "" {
		broadcastToRoomExcluding(controller, controlled,
			grapplemessaging.RenderTemplate(msg, controllerName, controlledName))
	}
}

// emitHoldFlavor sends sparse hold-round flavor every ~4 rounds.
// Tracked via a "hold_last_round" key in the last-round map.
func emitHoldFlavor(controller, controlled *characters.Character,
	outcome position.Outcome) {
	const holdEmitEveryRounds = uint64(4)
	round := util.GetRoundCount()
	lastKey := "hold_last_round:" + stateMessagingName(outcome.Source)
	lastRound := uint64(0)
	if controller.PerGrappleMessageCooldownsLastRound != nil {
		if v, ok := controller.PerGrappleMessageCooldownsLastRound[lastKey]; ok {
			lastRound = v
		}
	}
	if round-lastRound < holdEmitEveryRounds {
		return
	}
	// Mark for next time.
	if controller.PerGrappleMessageCooldownsLastRound == nil {
		controller.PerGrappleMessageCooldownsLastRound = map[string]uint64{}
	}
	controller.PerGrappleMessageCooldownsLastRound[lastKey] = round

	// Pick hold key by source state.
	key := holdKeyForState(outcome.Source)
	lib := loadGrappleLib()
	if lib == nil {
		return
	}
	triad, found := lib.Holds[key]
	if !found {
		return
	}
	controllerName := characterDisplayName(controller)
	controlledName := characterDisplayName(controlled)

	if msg := grapplemessaging.PickTemplate(triad.Controller,
		controller.PerGrappleMessageCooldowns, "hold:"+key+":ctrl"); msg != "" {
		sendToCharacter(controller,
			grapplemessaging.RenderTemplate(msg, controllerName, controlledName))
	}
	if msg := grapplemessaging.PickTemplate(triad.Controlled,
		controlled.PerGrappleMessageCooldowns, "hold:"+key+":cd"); msg != "" {
		sendToCharacter(controlled,
			grapplemessaging.RenderTemplate(msg, controllerName, controlledName))
	}
	if msg := grapplemessaging.PickTemplate(triad.Observers,
		controller.PerGrappleMessageCooldowns, "hold:"+key+":obs"); msg != "" {
		broadcastToRoomExcluding(controller, controlled,
			grapplemessaging.RenderTemplate(msg, controllerName, controlledName))
	}
}

// emitStrikingApexFlavor fires Mount-strike flavor on Hold rounds at
// Mount. Single-speaker list — visible to controller only; observers
// see the combat damage text from the combat system.
func emitStrikingApexFlavor(controller, controlled *characters.Character,
	outcome position.Outcome) {
	if outcome.Source != position.Mount {
		return
	}
	lib := loadGrappleLib()
	if lib == nil {
		return
	}
	pool, found := lib.StrikingApex["mount_strike_flavor"]
	if !found {
		return
	}
	controllerName := characterDisplayName(controller)
	controlledName := characterDisplayName(controlled)
	msg := grapplemessaging.PickTemplate(pool,
		controller.PerGrappleMessageCooldowns, "apex:mount_strike")
	if msg != "" {
		sendToCharacter(controller,
			grapplemessaging.RenderTemplate(msg, controllerName, controlledName))
	}
}

// stateMessagingName converts a position.State to its canonical
// snake_case YAML key segment.
func stateMessagingName(s position.State) string {
	switch s {
	case position.Clinch:
		return "clinch"
	case position.BackStanding:
		return "backstanding"
	case position.Mount:
		return "mount"
	case position.SideControl:
		return "sidecontrol"
	case position.KneeOnBelly:
		return "kob"
	case position.NorthSouth:
		return "ns"
	case position.Crucifix:
		return "crucifix"
	case position.BackGround:
		return "background"
	case position.HalfGuard:
		return "halfguard"
	case position.Guard:
		return "guard"
	case position.Turtle:
		return "turtle"
	case position.Standing:
		return "standing"
	}
	return "unknown"
}

// holdKeyForState picks the right hold-template key for a source state.
func holdKeyForState(s position.State) string {
	switch s {
	case position.Clinch:
		return "clinch_hold"
	case position.Guard:
		return "guard_hold"
	case position.Turtle:
		return "turtle_hold"
	case position.BackStanding:
		return "backstanding_hold"
	default:
		// Everything else is a ground position.
		return "ground_hold_generic"
	}
}

// characterDisplayName returns the name wrapped with the appropriate
// ANSI color tag (username for players, mobname for mobs).
func characterDisplayName(c *characters.Character) string {
	if c.GetUserId() > 0 {
		return fmt.Sprintf(`<ansi fg="username">%s</ansi>`, c.Name)
	}
	return fmt.Sprintf(`<ansi fg="mobname">%s</ansi>`, c.Name)
}

// sendToCharacter sends a text message to a player character.
// For mobs, no-op — NPCs don't have a stdout. Uses the existing
// userForCharacter helper from Position_Messaging.go (same package).
func sendToCharacter(c *characters.Character, msg string) {
	if u := userForCharacter(c); u != nil {
		u.SendText(messaging.CategoryGrappleFlow, msg)
	}
	// Mobs don't receive text.
}

// broadcastToRoomExcluding sends a text message to every user in the
// room except controller and controlled. Follows the pattern from
// sendSubmissionTriple in Position_Messaging.go.
func broadcastToRoomExcluding(controller, controlled *characters.Character,
	msg string) {
	roomId := 0
	if controller != nil {
		roomId = controller.RoomId
	}
	if roomId == 0 && controlled != nil {
		roomId = controlled.RoomId
	}
	if roomId == 0 {
		return
	}
	r := rooms.LoadRoom(roomId)
	if r == nil {
		return
	}
	var excludeIds []int
	if controller != nil && controller.GetUserId() > 0 {
		excludeIds = append(excludeIds, controller.GetUserId())
	}
	if controlled != nil && controlled.GetUserId() > 0 {
		excludeIds = append(excludeIds, controlled.GetUserId())
	}
	// COMPANION-NAME-LEAK SIBLING FIX (T11-followup): grapple prose names
	// both grapplers (controllerName/controlledName rendered into msg by
	// callers in this file) — route through SendTextVisual so infrared
	// observers see anonymized text and blind observers don't get a free
	// identification. Same class as companion_follow.go:55.
	switch len(excludeIds) {
	case 0:
		r.SendTextVisual(messaging.CategoryGrappleFlow, msg)
	case 1:
		r.SendTextVisual(messaging.CategoryGrappleFlow, msg, excludeIds[0])
	default:
		r.SendTextVisual(messaging.CategoryGrappleFlow, msg, excludeIds[0], excludeIds[1])
	}
}

// grappleScore computes one side's per-round drift score. Spec
// 2026-05-19 §3. Symmetric formula for both sides; aggressor gets a
// 10% bonus on the skill term (initiative edge).
//
//	score = (0.7·Str + 0.3·Dex + skill_coef·UnarmedCombat)
//	        × stamina_multiplier × encumbrance_multiplier
//
// where skill_coef = 2.2 for the aggressor side, 2.0 for the defender.
// Returns 0 for a nil character (defensive — callers should never
// pass nil but the function shouldn't panic on it).
func grappleScore(c *characters.Character, isAggressor bool, cfg configs.Balance) float64 {
	if c == nil {
		return 0
	}
	strVal := float64(c.Stats.Strength.Value)
	dexVal := float64(c.Stats.Dexterity.Value)
	skill := float64(c.GetSkillLevel(skills.UnarmedCombat))

	skillCoef := 2.0
	if isAggressor {
		skillCoef = 2.2
	}

	base := 0.7*strVal + 0.3*dexVal + skillCoef*skill
	return base * grappleStaminaMultiplier(c, cfg) * grappleEncumbranceMultiplier(c, cfg)
}

// isAggressorSide returns true if this character's GrappleData
// has IsAggressor=true. Set once by ApplyGrappleResult.markAggressor
// (chunk 4b-fixup-2 T5) when the grapple is initiated; persists
// for the grapple's lifetime regardless of subsequent reversals.
// Returns false defensively for nil/Position-less characters.
func isAggressorSide(c *characters.Character) bool {
	if c == nil || c.Position == nil {
		return false
	}
	d, ok := c.Position.GrappleData()
	if !ok {
		return false
	}
	return d.IsAggressor
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

// applyControlShift updates both sides' ControlLevel state based on
// drift roll outcome. Chunk 4b-fixup-2 T9 spec §5:
//   |z| < 0.5      → no shift
//   0.5 ≤ |z| < 1.5 → 1 step
//   |z| ≥ 1.5      → 2 steps
// Winner shifts toward Controlling; loser shifts toward Controlled.
func applyControlShift(controller, controlled *characters.Character, z float64) {
	absZ := z
	if absZ < 0 {
		absZ = -absZ
	}
	var steps int
	switch {
	case absZ < 0.5:
		return // no shift
	case absZ < 1.5:
		steps = 1
	default:
		steps = 2
	}

	if z > 0 {
		// Controller wins — shifts toward Controlling
		shiftControl(controller, +steps)
		shiftControl(controlled, -steps)
	} else {
		// Controlled wins — shifts toward Controlling
		shiftControl(controlled, +steps)
		shiftControl(controller, -steps)
	}
}

// shiftControl moves the given character's Control state by `steps`
// stable-state ranks. Positive steps = toward Controlling; negative
// = toward Controlled. Caps at the endpoints.
func shiftControl(c *characters.Character, steps int) {
	if c == nil || c.Control == nil {
		return
	}
	// Lazily bind Control machine to this character's ActorRef so the
	// boundary-cross callback can route gradient messages back to the
	// right character via characterFromRef. NewMachine() leaves self
	// at zero-value; without this, emitGradientMessage silently skips.
	if c.Control.Self().IsZero() {
		c.Control.SetSelf(state.ActorRef{
			UserId:        c.GetUserId(),
			MobInstanceId: c.GetMobInstanceId(),
		})
	}
	currentRank := controlRank(c.Control.State())
	newRank := currentRank - steps // toward Controlling means lower rank
	if newRank < 0 {
		newRank = 0
	}
	if newRank > 2 {
		newRank = 2
	}
	if newRank == currentRank {
		return
	}
	reason := state.TransitionReason{Trigger: control.TriggerDriftLoss}
	if newRank < currentRank {
		reason.Trigger = control.TriggerDriftWin
	}
	switch newRank {
	case 0:
		_ = c.Control.TransitionToControlling(reason)
	case 1:
		_ = c.Control.TransitionToNeutral(reason)
	case 2:
		_ = c.Control.TransitionToControlled(reason)
	}
}

// fireStaminaWarningIfLow (called at lines 219-220) and submission
// messaging live in Position_Messaging.go (T6/T7/T17).

func init() {
	events.RegisterListener(events.NewRound{}, processGrappleTick)
	// Template library loaded lazily on first use via loadGrappleLib()
	// to avoid mudlog nil-pointer panics during test package init.

	// Chunk 4b-fixup-2 T13: register the ControlLevel boundary-cross
	// callback to fire gradient messaging when characters cross state
	// boundaries. emitGradientMessage nil-checks grappleOutcomesLib so
	// it is safe to call even if the library has not loaded yet.
	control.RegisterBoundaryCrossCallback(func(self state.ActorRef, transient control.State, from control.State, to control.State, r state.TransitionReason) {
		emitGradientMessage(self, transient, from, to)
	})
}

// emitGradientMessage fires the appropriate gradient flavor when
// a character's ControlLevel state crosses a boundary same-tick.
// Resolves the gradient key from (transient, from, to) and dispatches
// the Self/Partner/Observers triad with name substitution + cooldown.
func emitGradientMessage(self state.ActorRef, transient control.State, from control.State, to control.State) {
	if grappleOutcomesLib == nil {
		return
	}
	key := gradientKeyForCrossing(transient, from, to)
	if key == "" {
		return
	}
	triad, ok := grappleOutcomesLib.Gradients[key]
	if !ok {
		mudlog.Warn("Position_GrappleTick: missing gradient key", "key", key)
		return
	}

	// Resolve self + partner characters.
	selfChar := characterFromRef(self)
	if selfChar == nil {
		return
	}
	partner := resolvePartner(selfChar)
	if partner == nil {
		return
	}

	if selfChar.PerGrappleMessageCooldowns == nil {
		selfChar.PerGrappleMessageCooldowns = map[string]bool{}
	}
	if partner.PerGrappleMessageCooldowns == nil {
		partner.PerGrappleMessageCooldowns = map[string]bool{}
	}

	selfName := characterDisplayName(selfChar)
	partnerName := characterDisplayName(partner)

	// Gradient templates use {controllerName} for "self" (the character
	// whose state changed) and {controlledName} for "partner" in
	// observer templates. The substitution matches what the YAML
	// authoring used (selfName fills {controllerName}; partnerName
	// fills {controlledName}).
	if msg := grapplemessaging.PickTemplate(triad.Self, selfChar.PerGrappleMessageCooldowns, "gradient:"+key+":self"); msg != "" {
		sendToCharacter(selfChar, grapplemessaging.RenderTemplate(msg, selfName, partnerName))
	}
	if msg := grapplemessaging.PickTemplate(triad.Partner, partner.PerGrappleMessageCooldowns, "gradient:"+key+":partner"); msg != "" {
		sendToCharacter(partner, grapplemessaging.RenderTemplate(msg, selfName, partnerName))
	}
	if msg := grapplemessaging.PickTemplate(triad.Observers, selfChar.PerGrappleMessageCooldowns, "gradient:"+key+":obs"); msg != "" {
		broadcastToRoomExcluding(selfChar, partner, grapplemessaging.RenderTemplate(msg, selfName, partnerName))
	}
}

// gradientKeyForCrossing returns the YAML key for a boundary crossing.
// Spec §7:
//
//	upper_boundary_down: Controlling → Neutral
//	upper_boundary_up:   Neutral → Controlling
//	lower_boundary_down: Neutral → Controlled
//	lower_boundary_up:   Controlled → Neutral
func gradientKeyForCrossing(transient control.State, from control.State, to control.State) string {
	if transient == control.LosingControl {
		if from == control.Controlling && to == control.Neutral {
			return "upper_boundary_down"
		}
		if from == control.Neutral && to == control.Controlling {
			return "upper_boundary_up"
		}
	}
	if transient == control.BecomingControlled {
		if from == control.Neutral && to == control.Controlled {
			return "lower_boundary_down"
		}
		if from == control.Controlled && to == control.Neutral {
			return "lower_boundary_up"
		}
	}
	return ""
}

// characterFromRef resolves a state.ActorRef to its Character pointer.
// Helper for emitGradientMessage; mirrors the resolvePartner pattern.
func characterFromRef(ref state.ActorRef) *characters.Character {
	if ref.UserId > 0 {
		if u := users.GetByUserId(ref.UserId); u != nil {
			return u.Character
		}
	}
	if ref.MobInstanceId > 0 {
		if m := mobs.GetInstance(ref.MobInstanceId); m != nil {
			return &m.Character
		}
	}
	return nil
}
