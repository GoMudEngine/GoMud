package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// wireCompanionAssist registers an Attackers-change subscription on
// every character. When the inbound Attackers list grows, the handler
// checks whether the character is a charmed companion. If so, and if
// the owner (and sibling companions) are Idle, they are directed to
// attack the new threat.
//
// This replaces the polling-based CompanionAutoTarget scan that ran
// in NewRound_DoCombat once per round. The new path fires reactively
// the moment a new attacker is recorded on the companion's CombatPhase,
// rather than waiting until the next round tick.
//
// Behavioral parity with the old polling path:
//   - Same AutoAssist flag check (companion entry on owner's character)
//   - Same grace-period check (NoAggroTarget buff on owner)
//   - Owner's own "attack owner's current target" path preserved via
//     the existing NewRound_DoCombat_helpers.go#handleCompanionOwnerAssist
//     for MvP attacks (which fires before the round loop, not here)
//   - Sibling companions in the same room are also directed to assist
//
// Task 18 will remove CompanionAutoTarget from aggro_helpers.go.
// Until then, both paths may fire: the reactive (this file) fires when
// the attacker is first recorded; the polling path fires the next round
// tick. Duplicate attack commands are benign — the second attempt hits
// an already-fighting companion or is vetoed.
func wireCompanionAssist(c *characters.Character) {
	prevLen := 0
	c.CombatPhase.SubscribeAttackersChange(func(attackers []state.ActorRef) {
		// Only react when the list GROWS (new attacker added).
		// Shrinks (attacker removed on death/flee) are not assists events.
		cur := len(attackers)
		if cur <= prevLen {
			prevLen = cur
			return
		}
		prevLen = cur

		// Identify the companion mob owning this character.
		mob := findMobOwningCharacter(c)
		if mob == nil {
			return // player character or unmapped transient character
		}
		if !mob.Character.IsCharmed() {
			return // not a companion
		}

		ownerId := mob.Character.GetCharmedUserId()
		if ownerId == 0 {
			return
		}

		owner := users.GetByUserId(ownerId)
		if owner == nil {
			return
		}

		// Grace-period: if the owner has the respawn-grace buff, no mob
		// should be targeting them. Companions stand down to avoid pulling
		// the mob into a fight before grace expires.
		if owner.Character.HasBuffFlag(buffs.NoAggroTarget) {
			return
		}

		// AutoAssist gate: check the companion entry on the owner.
		comp := owner.Character.GetCompanionByInstanceId(mob.InstanceId)
		if comp == nil || !comp.AutoAssist {
			return
		}

		// Pick the newest inbound attacker (last in the list) as the
		// engagement target for the owner. The companion itself may
		// already be engaging this attacker; the owner is the one we're
		// pulling in here.
		newAttacker := attackers[len(attackers)-1]
		attackCmd := actorRefToAttackCmd(newAttacker)
		if attackCmd == "" {
			return
		}

		// Owner: if Idle, direct them to attack. Use CombatPhase predicate
		// (IsInCombat) so we read from the state machine, not legacy Aggro.
		if !owner.Character.IsInCombat() {
			owner.Command(fmt.Sprintf("attack %s", attackCmd))
		}

		// Sibling companions in the same room: direct any other Idle,
		// AutoAssist-flagged companions to join the fight.
		ownerRoom := rooms.LoadRoom(owner.Character.RoomId)
		if ownerRoom == nil {
			return
		}
		for _, instanceId := range ownerRoom.GetMobs(rooms.FindCharmed) {
			if instanceId == mob.InstanceId {
				continue // the companion that was attacked — already handling its own fight
			}
			sibling := mobs.GetInstance(instanceId)
			if sibling == nil {
				continue
			}
			if !sibling.Character.IsCharmed(ownerId) {
				continue // not our owner's companion
			}
			if sibling.Character.IsInCombat() {
				continue // already fighting
			}
			sibComp := owner.Character.GetCompanionByInstanceId(sibling.InstanceId)
			if sibComp == nil || !sibComp.AutoAssist {
				continue
			}
			sibling.Command(fmt.Sprintf("attack %s", attackCmd))
		}
	})
}

// actorRefToAttackCmd converts an ActorRef to the mob `attack` command
// argument format used throughout the engine:
//
//	Player → "attack @<userId>"
//	Mob    → "attack #<instanceId>"
func actorRefToAttackCmd(ref state.ActorRef) string {
	if ref.IsPlayer() {
		return fmt.Sprintf("@%d", ref.UserId)
	}
	if ref.IsMob() {
		return fmt.Sprintf("#%d", ref.MobInstanceId)
	}
	return ""
}

func init() {
	characters.OnCharacterCreated(wireCompanionAssist)
}
