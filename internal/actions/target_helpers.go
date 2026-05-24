package actions

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// ResolveEngagedTarget returns the actor that the given actor is currently
// engaged with in combat, or nil if there is no engaged target. Polymorphic
// across user and mob actors: consults Character.CurrentCombatTarget() which
// reads from the CombatPhase FSM and falls back to the legacy Aggro field,
// making it correct for both player actors (EngagedTarget path) and mob
// actors (Aggro path).
//
// Returns nil if:
//   - the actor's Character is nil,
//   - the actor has no current combat target, or
//   - the target's instance can no longer be found in the world (race
//     condition: target left between state-read and registry lookup).
func ResolveEngagedTarget(actor Actor, room *rooms.Room) Actor {
	char := actor.GetCharacter()
	if char == nil {
		return nil
	}
	target := char.CurrentCombatTarget()
	if target.MobInstanceId > 0 {
		if victim := mobs.GetInstance(target.MobInstanceId); victim != nil {
			return NewMobActorInRoom(victim, room)
		}
		return nil
	}
	if target.UserId > 0 {
		if victim := users.GetByUserId(target.UserId); victim != nil {
			return NewUserActorInRoom(victim, room)
		}
		return nil
	}
	return nil
}
