package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Warcry(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	result := actions.ExecuteWarcry(&actions.UserActor{User: user, Room: room})

	if result.Crafting {
		user.SendTextLegacy(`<ansi fg="red">You can't muster a warcry while focused on your work. Finish or be interrupted first.</ansi>`)
		return true, nil
	}

	if result.AlreadyActive {
		user.SendTextLegacy("Your warcry still echoes — you can't shout it louder.")
		return true, nil
	}

	if result.OnCooldown {
		user.SendTextLegacy("You need a moment to recover before attempting another special move.")
		return true, nil
	}

	user.SendTextLegacy(`<ansi fg="red-bold">You let out a thunderous warcry that ignites the fighting spirit of your allies!</ansi>`)
	room.SendTextVisualLegacy(
		fmt.Sprintf(`<ansi fg="red-bold"><ansi fg="username">%s</ansi> lets out a thunderous warcry!</ansi>`, user.Character.Name),
		user.UserId,
	)

	// Apply to party members in the room
	if party := parties.Get(user.UserId); party != nil {
		for _, memberId := range party.GetMembers() {
			if memberId == user.UserId {
				continue
			}
			if memberUser := users.GetByUserId(memberId); memberUser != nil {
				if memberUser.Character.RoomId == user.Character.RoomId {
					memberUser.Character.AddCondition(characters.ConditionWarcry, result.Duration, result.Bonus, "warcry")
					memberUser.Character.AddBuff(79, false)
					memberUser.SendTextLegacy(
						fmt.Sprintf(`<ansi fg="red-bold"><ansi fg="username">%s</ansi>'s warcry stirs your blood!</ansi>`, user.Character.Name))

					// Apply to this party member's companions in the room
					applyWarcryToCompanions(memberUser, room, result.Bonus, result.Duration)
				}
			}
		}
	}

	// Apply to caster's own companions in the room
	applyWarcryToCompanions(user, room, result.Bonus, result.Duration)

	// Skill and stat progression
	// OnSkillUse handles rhetoric progression + charisma stat use internally.
	// In combat: always fire. Out of combat: 50% chance (soft incentive).
	if user.Character.IsInCombat() {
		user.Character.OnSkillUse(string(skills.Rhetoric), user.UserId)
	} else if util.Rand(100) < 50 {
		user.Character.OnSkillUse(string(skills.Rhetoric), user.UserId)
	}

	return true, nil
}

func applyWarcryToCompanions(owner *users.UserRecord, room *rooms.Room, bonus float64, duration int) {
	for _, mobInstId := range owner.Character.GetCharmIds() {
		mob := mobs.GetInstance(mobInstId)
		if mob == nil {
			continue
		}
		if mob.Character.RoomId != owner.Character.RoomId {
			continue
		}
		mob.Character.AddCondition(characters.ConditionWarcry, duration, bonus, "warcry")
		mob.Character.AddBuff(79, false)
	}
}
