package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Assist attacks whatever a party member is currently fighting.
// Usage: assist          — assist party leader
//
//	assist <name>   — assist a specific party member
func Assist(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	partyInfo := parties.Get(user.UserId)
	if partyInfo == nil {
		user.SendTextLegacy(`You are not in a party.`)
		return true, nil
	}

	// Determine which party member to assist
	var assistTarget *users.UserRecord

	if rest == `` {
		// Default: assist party leader
		if partyInfo.IsLeader(user.UserId) {
			// Leader with no argument: assist first party member who is fighting
			for _, memberId := range partyInfo.UserIds {
				if memberId == user.UserId {
					continue
				}
				if member := users.GetByUserId(memberId); member != nil {
					if member.Character.IsInCombat() && member.Character.RoomId == user.Character.RoomId {
						assistTarget = member
						break
					}
				}
			}
		} else {
			assistTarget = users.GetByUserId(partyInfo.LeaderUserId)
		}
	} else {
		// Assist a named party member
		for _, memberId := range partyInfo.UserIds {
			if memberId == user.UserId {
				continue
			}
			if member := users.GetByUserId(memberId); member != nil {
				if matchesName(member.Character.Name, rest) {
					assistTarget = member
					break
				}
			}
		}
	}

	if assistTarget == nil {
		user.SendTextLegacy(`No party member found to assist.`)
		return true, nil
	}

	if !assistTarget.Character.IsInCombat() {
		user.SendTextLegacy(fmt.Sprintf(`<ansi fg="username">%s</ansi> is not fighting anyone.`, assistTarget.Character.Name))
		return true, nil
	}

	if assistTarget.Character.RoomId != user.Character.RoomId {
		user.SendTextLegacy(fmt.Sprintf(`<ansi fg="username">%s</ansi> is not here.`, assistTarget.Character.Name))
		return true, nil
	}

	// Attack whatever they're fighting
	if mobInstId := assistTarget.Character.EngagedTarget().MobInstanceId; mobInstId > 0 {
		if m := mobs.GetInstance(mobInstId); m != nil {
			return Attack(fmt.Sprintf("#%d", mobInstId), user, room, flags)
		}
	}

	if targetUserId := assistTarget.Character.EngagedTarget().UserId; targetUserId > 0 {
		if p := users.GetByUserId(targetUserId); p != nil {
			return Attack(fmt.Sprintf("@%d", targetUserId), user, room, flags)
		}
	}

	user.SendTextLegacy(fmt.Sprintf(`<ansi fg="username">%s</ansi>'s target is no longer here.`, assistTarget.Character.Name))
	return true, nil
}

// matchesName does a case-insensitive prefix match on a character name.
func matchesName(charName string, input string) bool {
	if len(input) > len(charName) {
		return false
	}
	for i := range input {
		if (input[i] | 0x20) != (charName[i] | 0x20) {
			return false
		}
	}
	return true
}
