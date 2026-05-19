// Position_SubmissionTick.go fires once per round per active grapple
// pair, after Position_GrappleTick has stashed the drift snapshot.
// Checks each side of the pair for sub-attempt eligibility (top
// attack from the controller, bottom-attack reversal from the
// controlled side); if eligible, rolls a fresh opposed sub roll and
// logs the result. Outcome application (full resolve via
// combat.ResolveSubmissionOutcome) is stubbed here — T7 lands the
// resolver and replaces the stub.
//
// init() registration order: "Position_SubmissionTick.go" sorts
// alphabetically after "Position_GrappleTick.go" within the hooks
// package, so this listener registers AFTER processGrappleTick.
// Both are NewRound listeners; the event system fires them in
// registration order, ensuring the drift snapshot is fresh when
// this observer reads it.
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// processSubmissionTick is the NewRound event listener. Iterates
// active players + mobs; for each pair where a sub attempt is
// eligible, fires the sub roll. Runs AFTER processGrappleTick so
// LastDriftRoll is fresh.
func processSubmissionTick(e events.Event) events.ListenerReturn {
	for _, u := range users.GetAllActiveUsers() {
		if u == nil || u.Character == nil {
			continue
		}
		processSubmissionTickForChar(u.Character)
	}
	for _, mobInstId := range mobs.GetAllMobInstanceIds() {
		m := mobs.GetInstance(mobInstId)
		if m == nil {
			continue
		}
		processSubmissionTickForChar(&m.Character)
	}
	return events.Continue
}

// processSubmissionTickForChar handles a single character's
// sub-tick opportunity. Only processes the controller side of each
// pair — the controlled side is handled transitively (EvaluateSubAttempt
// reads both sides' eligibility from the same snapshot). Each pair is
// processed exactly once.
func processSubmissionTickForChar(c *characters.Character) {
	if c == nil || c.Position == nil {
		return
	}
	if !c.IsController() {
		return
	}
	partner := resolvePartner(c)
	if partner == nil {
		return
	}

	// Chunk 4e §8: always reset the sub-interrupt accumulator for the
	// controller at the end of this tick, regardless of whether a sub
	// fires. The accumulator was written by the damage pipeline this
	// round (T7) and must not bleed into the next round. Using defer
	// ensures the reset fires even on early-return paths (no sub pool,
	// not eligible, etc.).
	//
	// The attempter might be c or partner, determined below. We reset
	// both c and partner defensively — the partner's own tick will
	// never fire (only controllers are processed here), so this is
	// the only opportunity to clear partner's accumulator.
	defer func() {
		c.SubInterruptDamageThisRound = 0
		if partner != nil {
			partner.SubInterruptDamageThisRound = 0
		}
	}()

	role, eligible := EvaluateSubAttempt(c, partner)
	if !eligible {
		return
	}

	var attempter, recipient *characters.Character
	var subPool []position.SubmissionType

	switch role {
	case combat.RoleTop:
		attempter, recipient = c, partner
		subPool = position.TopSubmissionsForPosition(c.Position.State())
	case combat.RoleBottom:
		attempter, recipient = partner, c
		subPool = position.BottomSubmissionsForPosition(c.Position.State())
	}

	if len(subPool) == 0 {
		return
	}

	subType := pickSubmissionRoundRobin(attempter, subPool)
	result := combat.RollSubmissionAttempt(attempter, recipient, subType)

	// Chunk 4e §8: if the submitter took qualifying third-party damage
	// this round, force Bad-tier outcome. The damage was accumulated by
	// chunk4eAccumulateSubInterruptDamage in the combat pipeline (T7).
	// Models "a kick to the ribs while applying an armbar = you lose the
	// armbar AND your position."
	if attempter.SubInterruptDamageThisRound > 0 {
		mudlog.Debug("Position_SubmissionTick: sub interrupted by 3rd-party damage",
			"submitter_user", attempter.GetUserId(),
			"submitter_mob", attempter.GetMobInstanceId(),
			"damage", attempter.SubInterruptDamageThisRound,
			"original_tier", result.Tier,
		)
		result.Tier = combat.SubTierBad
	}

	mudlog.Debug("SubmissionTick",
		"role", role,
		"attempter", attempter.Name,
		"recipient", recipient.Name,
		"sub", subType,
		"tier", result.Tier,
		"atkZ", result.AttackerZScore,
	)

	combat.ResolveSubmissionOutcome(attempter, recipient, result, role)
}

// EvaluateSubAttempt checks whether a sub attempt is eligible for
// either side of a grapple pair based on the chunk-4b drift snapshot
// stashed on Character.LastDriftRoll this round. Returns the role of
// the attempter (top = controller, bottom = controlled) and whether
// eligible.
//
// Eligibility rules (chunk 4b-fixup T20: gate unified to SubWindowOpens):
//   - Top (controller): position.SubWindowOpens(MarginAttacker) AND
//     top-attack subs are available at this position.
//   - Bottom (controlled): position.SubWindowOpens(-MarginAttacker) OR
//     DefenderZScore >= critZ (config-tunable shortcut), AND bottom-
//     attack subs are available at this position.
//
// position.SubWindowOpens uses the canonical |z| >= 1.5 threshold
// (subWindowAlpha in internal/state/position/outcomes.go), replacing
// the former config-driven SubmissionAttemptAlpha (default 1.0).
//
// At most one side passes per round — the drift roll has one winner.
// If both sides qualify (wide gate + crit shortcut edge case), the
// side with the larger absolute z-score wins the tiebreak.
//
// Returns (RoleTop, false) as the zero-value non-eligible result so
// callers can safely ignore the role when eligible==false.
func EvaluateSubAttempt(controller, controlled *characters.Character) (combat.Role, bool) {
	if controller == nil || controller.Position == nil ||
		controlled == nil || controlled.Position == nil {
		return combat.RoleTop, false
	}

	currentRound := util.GetRoundCount()
	snap := controller.LastDriftRoll
	if snap.Round != currentRound {
		return combat.RoleTop, false // stale or missing snapshot
	}

	// critZ remains config-driven (defender crit shortcut is a tunable
	// design knob independent of the sub-window threshold).
	// The sub-window threshold itself is now read from the canonical
	// position.SubWindowOpens gate (|z| >= 1.5) rather than from the
	// old SubmissionAttemptAlpha config field (which defaulted to 1.0).
	// Using the unified gate keeps chunk-4d in lock-step with the
	// chunk 4b-fixup outcome dispatcher.
	cfg := configs.GetBalanceConfig()
	critZ := float64(cfg.SubmissionAttemptCritZ)

	posState := controller.Position.State()

	// Top eligibility: controller won drift roll big AND top subs available.
	// Chunk 4b-fixup-2 T14: pass Control.State() directly so IsTopSubEligible
	// can gate on exactly Controlling (Neutral no longer qualifies).
	topOK := false
	if controller.Control != nil &&
		position.IsTopSubEligible(posState, controller.Control.State()) &&
		position.SubWindowOpens(snap.MarginAttacker) {
		topOK = true
	}

	// Bottom eligibility: defender won drift roll big OR crit-defended,
	// AND bottom subs available at this position+level.
	// Chunk 4b-fixup-2 T14: pass controlled.Control.State() so
	// IsBottomSubEligible gates on exactly Controlled (Neutral no longer
	// qualifies). Nil-guard in case Control hasn't been initialised yet.
	bottomOK := false
	bottomMargin := -snap.MarginAttacker // defender margin is the inverse
	if controlled.Control != nil &&
		position.IsBottomSubEligible(posState, controlled.Control.State()) {
		if position.SubWindowOpens(bottomMargin) || snap.DefenderZScore >= critZ {
			bottomOK = true
		}
	}

	switch {
	case topOK && !bottomOK:
		return combat.RoleTop, true
	case bottomOK && !topOK:
		return combat.RoleBottom, true
	case topOK && bottomOK:
		// Tiebreak: larger absolute z-score wins.
		atkAbsZ := snap.AttackerZScore
		if atkAbsZ < 0 {
			atkAbsZ = -atkAbsZ
		}
		defAbsZ := snap.DefenderZScore
		if defAbsZ < 0 {
			defAbsZ = -defAbsZ
		}
		if atkAbsZ >= defAbsZ {
			return combat.RoleTop, true
		}
		return combat.RoleBottom, true
	default:
		return combat.RoleTop, false
	}
}

// pickSubmissionRoundRobin advances the attempter's
// LastSubmissionAttempted index and returns the next sub from the
// pool. Wraps modulo the pool length so it cycles through without
// hammering the same submission every round. Returns SubNone for an
// empty pool (caller checks before calling).
func pickSubmissionRoundRobin(c *characters.Character, pool []position.SubmissionType) position.SubmissionType {
	if len(pool) == 0 {
		return position.SubNone
	}
	c.LastSubmissionAttempted = (c.LastSubmissionAttempted + 1) % len(pool)
	return pool[c.LastSubmissionAttempted]
}

func init() {
	// processSubmissionTick must run AFTER processGrappleTick so the
	// LastDriftRoll snapshot is fresh. Both are NewRound listeners;
	// init() ordering is filename-alphabetical within the package —
	// "Position_SubmissionTick.go" sorts after "Position_GrappleTick.go"
	// so registration order is correct.
	events.RegisterListener(events.NewRound{}, processSubmissionTick)
}
