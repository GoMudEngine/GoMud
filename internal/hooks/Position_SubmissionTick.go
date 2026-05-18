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
// Eligibility rules:
//   - Top (controller): drift MarginAttacker > alpha AND top-attack
//     subs are available at this position+control-level.
//   - Bottom (controlled): (-MarginAttacker) > alpha OR
//     DefenderZScore >= critZ, AND bottom-attack subs are available.
//
// At most one side passes per round — the drift roll has one winner.
// If both sides qualify (wide alpha + crit shortcut edge case),
// the side with the larger absolute z-score wins the tiebreak.
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

	cfg := configs.GetBalanceConfig()
	alpha := float64(cfg.SubmissionAttemptAlpha)
	critZ := float64(cfg.SubmissionAttemptCritZ)

	posState := controller.Position.State()
	ctrlData, ok := controller.Position.GrappleData()
	if !ok {
		return combat.RoleTop, false
	}

	// Top eligibility: controller won drift roll big AND top subs available.
	// IsTopSubEligible now takes IsControllerRole bool.
	topOK := false
	if position.IsTopSubEligible(posState, ctrlData.IsControllerRole) && snap.MarginAttacker > alpha {
		topOK = true
	}

	// Bottom eligibility: defender won drift roll big OR crit-defended,
	// AND bottom subs available at this position+level.
	bottomOK := false
	bottomMargin := -snap.MarginAttacker // defender margin is the inverse
	cdData, cdOK := controlled.Position.GrappleData()
	if cdOK {
		if position.IsBottomSubEligible(posState, cdData.IsControllerRole) {
			if bottomMargin > alpha || snap.DefenderZScore >= critZ {
				bottomOK = true
			}
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
