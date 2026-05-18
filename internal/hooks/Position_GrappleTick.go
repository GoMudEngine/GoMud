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
	"fmt"
	"math"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/grapplemessaging"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/state"
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
		u.SendText(msg)
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
	switch len(excludeIds) {
	case 0:
		r.SendText(msg)
	case 1:
		r.SendText(msg, excludeIds[0])
	default:
		r.SendText(msg, excludeIds[0], excludeIds[1])
	}
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
	// Template library loaded lazily on first use via loadGrappleLib()
	// to avoid mudlog nil-pointer panics during test package init.
}
