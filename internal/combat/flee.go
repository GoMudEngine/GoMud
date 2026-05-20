package combat

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// FleeBlocker identifies the first opponent that successfully blocked
// a flee attempt. Exactly one of UserId / MobInstanceId is non-zero,
// matching the standard `state.ActorRef` convention.
type FleeBlocker struct {
	Name          string
	UserId        int
	MobInstanceId int
}

// IsPlayer reports whether the blocker is a player.
func (b FleeBlocker) IsPlayer() bool { return b.UserId != 0 }

// ResolveFleeBlockers walks every combatant in `room` currently
// targeting `fleer`, rolls each one's Dex+UnarmedCombat×25 against
// the fleer's Dex+Skullduggery×25 (with a 0.5× prone penalty when
// the fleer is Prone or Supine), and returns the first blocker
// whose roll wins. Returns nil if nobody blocks.
//
// Works for player and mob fleers alike: the helper reads
// fleer.GetUserId() and fleer.GetMobInstanceId() to identify which
// combatants are actually targeting the fleer. Both player and mob
// blockers are walked.
//
// The grapple-block precondition (IsStandingGrapple || IsGroundGrapple
// on the fleer) is NOT checked here — callers handle it because the
// messaging differs (player direct vs mob room broadcast). This
// helper is purely the opposed-roll resolver.
//
// Replaces two duplicated opposed-roll loops:
//   - internal/hooks/NewRound_DoCombat_helpers.go:handlePlayerFlee
//   - internal/mobcommands/flee.go:Flee
//
// Side effect of the refactor: the player-flee path's player-blocker
// loop had a variable-shadowing typo (the inner `for _, userId := range`
// shadowed the outer fleer's userId, so the in-loop "is this player
// targeting me" check degenerated to "is this player targeting
// themselves" — always false in practice). Routing through this
// helper restores the intended PvP-blocker behavior.
func ResolveFleeBlockers(fleer *characters.Character, room *rooms.Room) *FleeBlocker {
	if fleer == nil || room == nil {
		return nil
	}

	fleerUid := fleer.GetUserId()
	fleerMid := fleer.GetMobInstanceId()

	// Prone penalty: Supine and Prone both halve the fleer's score
	// (legacy enum couldn't distinguish, and the design intent is
	// "you're slower because you're on the floor" — applies equally).
	pronePenalty := 1.0
	if fleer.IsProne() || fleer.IsSupine() {
		pronePenalty = 0.5
	}
	fleeScore := float64(fleer.Stats.Dexterity.ValueAdj+
		fleer.GetSkillLevel(skills.Skullduggery)*25) * pronePenalty

	// Mobs first, then players. Both loops use FindFighting to scope
	// to combatants only; the in-loop targeting filter narrows to
	// "fighting the fleer specifically".
	for _, mobInstId := range room.GetMobs(rooms.FindFighting) {
		m := mobs.GetInstance(mobInstId)
		if m == nil {
			continue
		}
		if m.InstanceId == fleerMid {
			continue // a mob doesn't block its own flee
		}
		if !mobTargetsFleer(m, fleerUid, fleerMid) {
			continue
		}
		blockScore := float64(m.Character.Stats.Dexterity.ValueAdj +
			m.Character.GetSkillLevel(skills.UnarmedCombat)*25)
		success, _, _, _ := dice.OpposedRollStat(fleeScore, blockScore)
		if !success {
			return &FleeBlocker{
				Name:          m.Character.Name,
				MobInstanceId: m.InstanceId,
			}
		}
	}

	for _, uId := range room.GetPlayers(rooms.FindFighting) {
		if uId == fleerUid {
			continue // a player doesn't block their own flee
		}
		u := users.GetByUserId(uId)
		if u == nil {
			continue
		}
		if !playerTargetsFleer(u, fleerUid, fleerMid) {
			continue
		}
		blockScore := float64(u.Character.Stats.Dexterity.ValueAdj +
			u.Character.GetSkillLevel(skills.UnarmedCombat)*25)
		success, _, _, _ := dice.OpposedRollStat(fleeScore, blockScore)
		if !success {
			return &FleeBlocker{
				Name:   u.Character.Name,
				UserId: u.UserId,
			}
		}
	}

	return nil
}

// mobTargetsFleer reports whether mob m's aggro points at the fleer
// (player or mob).
func mobTargetsFleer(m *mobs.Mob, fleerUid, fleerMid int) bool {
	if m == nil || m.Character.Aggro == nil {
		return false
	}
	if fleerUid > 0 && m.Character.Aggro.UserId == fleerUid {
		return true
	}
	if fleerMid > 0 && m.Character.Aggro.MobInstanceId == fleerMid {
		return true
	}
	return false
}

// playerTargetsFleer reports whether player u's current combat
// target is the fleer (player or mob).
func playerTargetsFleer(u *users.UserRecord, fleerUid, fleerMid int) bool {
	if u == nil || !u.Character.IsInCombat() {
		return false
	}
	target := u.Character.CurrentCombatTarget()
	if fleerUid > 0 && target.UserId == fleerUid {
		return true
	}
	if fleerMid > 0 && target.MobInstanceId == fleerMid {
		return true
	}
	return false
}
