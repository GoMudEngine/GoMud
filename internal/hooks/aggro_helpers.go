package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// ValidateAggro checks if a character's aggro target still exists and is
// alive. If the target is gone or dead, EndAggro() is called and false is
// returned. Returns false (without calling EndAggro) when Aggro is already nil.
func ValidateAggro(char *characters.Character) bool {
	if char.Aggro == nil {
		return false
	}

	if char.Aggro.MobInstanceId > 0 {
		target := mobs.GetInstance(char.Aggro.MobInstanceId)
		if target == nil || target.Character.Health < 1 {
			char.EndAggro()
			return false
		}
	}

	if char.Aggro.UserId > 0 {
		target := users.GetByUserId(char.Aggro.UserId)
		if target == nil || target.Character.Health < 1 {
			char.EndAggro()
			return false
		}
	}

	return true
}

// RetargetOrEnd clears the character's current aggro and scans the room for a
// new target that is already attacking us (by userId or mobInstanceId). For
// charmed companions the scan also considers mobs attacking the charm owner.
// Returns true if a new target was found and SetAggro was called.
func RetargetOrEnd(char *characters.Character, room *rooms.Room,
	userId int, mobInstanceId int) bool {

	char.EndAggro()

	// Scan mobs in the room that are currently fighting.
	for _, instId := range room.GetMobs(rooms.FindFighting) {
		attackingMob := mobs.GetInstance(instId)
		if attackingMob == nil || attackingMob.Character.Aggro == nil {
			continue
		}

		aggro := attackingMob.Character.Aggro

		// Is this mob attacking us (player or mob)?
		if (userId > 0 && aggro.UserId == userId) ||
			(mobInstanceId > 0 && aggro.MobInstanceId == mobInstanceId) {
			char.SetAggro(0, attackingMob.InstanceId, characters.DefaultAttack)
			return true
		}
	}

	// For charmed companions: also retarget mobs attacking the charm owner.
	if char.IsCharmed() {
		ownerId := char.GetCharmedUserId()
		if ownerId > 0 {
			for _, instId := range room.GetMobs(rooms.FindFighting) {
				attackingMob := mobs.GetInstance(instId)
				if attackingMob == nil || attackingMob.Character.Aggro == nil {
					continue
				}
				if attackingMob.Character.Aggro.UserId == ownerId {
					char.SetAggro(0, attackingMob.InstanceId, characters.DefaultAttack)
					return true
				}
			}
		}
	}

	// Scan players in the room that are currently fighting.
	for _, pId := range room.GetPlayers(rooms.FindFighting) {
		attackingPlayer := users.GetByUserId(pId)
		if attackingPlayer == nil || attackingPlayer.Character.Aggro == nil {
			continue
		}

		aggro := attackingPlayer.Character.Aggro

		// Is this player attacking us?
		if (userId > 0 && aggro.UserId == userId) ||
			(mobInstanceId > 0 && aggro.MobInstanceId == mobInstanceId) {
			char.SetAggro(attackingPlayer.UserId, 0, characters.DefaultAttack)
			return true
		}
	}

	return false
}

// CompanionAutoTarget checks whether an idle charmed mob should enter combat
// to defend its owner. It does nothing if the mob is already fighting or is
// not charmed. AutoAssist must be set on the companion entry for it to act.
//
// Priority:
//  1. If the owner is already fighting, attack the owner's current target.
//  2. Otherwise scan the room for mobs that are attacking the owner and
//     engage the first one found.
func CompanionAutoTarget(mob *mobs.Mob, room *rooms.Room) {
	// Already fighting — nothing to do.
	if mob.Character.Aggro != nil {
		return
	}

	if !mob.Character.IsCharmed() {
		return
	}

	ownerId := mob.Character.GetCharmedUserId()
	if ownerId == 0 {
		return
	}

	owner := users.GetByUserId(ownerId)
	if owner == nil {
		return
	}

	// Verify AutoAssist flag on the companion entry.
	comp := owner.Character.GetCompanionByInstanceId(mob.InstanceId)
	if comp == nil || !comp.AutoAssist {
		return
	}

	// If owner is fighting, join their fight immediately.
	if owner.Character.Aggro != nil {
		ownerAggro := owner.Character.Aggro
		if ownerAggro.UserId > 0 {
			mob.Command(fmt.Sprintf("attack @%d", ownerAggro.UserId))
			return
		}
		if ownerAggro.MobInstanceId > 0 {
			mob.Command(fmt.Sprintf("attack #%d", ownerAggro.MobInstanceId))
			return
		}
	}

	// Owner is idle — scan room for mobs attacking the owner and intercept.
	for _, instId := range room.GetMobs(rooms.FindFighting) {
		attackingMob := mobs.GetInstance(instId)
		if attackingMob == nil || attackingMob.Character.Aggro == nil {
			continue
		}
		if attackingMob.Character.Aggro.UserId == ownerId {
			mob.Command(fmt.Sprintf("attack #%d", attackingMob.InstanceId))
			return
		}
	}
}
