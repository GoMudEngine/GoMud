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

func Rally(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	result := actions.ExecuteRally(&actions.UserActor{User: user, Room: room})

	if result.Crafting {
		user.SendText(`<ansi fg="red">You can't rally while focused on your work. Finish or be interrupted first.</ansi>`)
		return true, nil
	}
	if result.OnCooldown {
		user.SendText("You need a moment to recover before attempting another special move.")
		return true, nil
	}
	if !result.Executed {
		return true, nil
	}

	user.SendText(`<ansi fg="cyan-bold">You rally your allies with an inspiring shout that steadies their resolve!</ansi>`)
	room.SendTextVisual(
		fmt.Sprintf(`<ansi fg="cyan-bold"><ansi fg="username">%s</ansi> rallies everyone with an inspiring shout!</ansi>`, user.Character.Name),
		user.UserId,
	)

	// Fan out to party members in the room.
	if party := parties.Get(user.UserId); party != nil {
		for _, memberId := range party.GetMembers() {
			if memberId == user.UserId {
				continue
			}
			memberUser := users.GetByUserId(memberId)
			if memberUser == nil || memberUser.Character.RoomId != user.Character.RoomId {
				continue
			}
			memberUser.Character.AddCondition(characters.ConditionRally, result.Duration, result.Bonus, "rally")
			memberUser.Character.AddBuff(80, false)
			memberUser.SendText(
				fmt.Sprintf(`<ansi fg="cyan-bold"><ansi fg="username">%s</ansi>'s rallying cry steadies your nerves!</ansi>`, user.Character.Name))
			applyRallyToCompanions(memberUser, room, result.Bonus, result.Duration)
		}
	}

	// Fan out to caster's own companions in the room.
	applyRallyToCompanions(user, room, result.Bonus, result.Duration)

	// Rhetoric skill progression.
	if user.Character.Aggro != nil {
		user.Character.OnSkillUse(string(skills.Rhetoric), user.UserId)
	} else if util.Rand(100) < 50 {
		user.Character.OnSkillUse(string(skills.Rhetoric), user.UserId)
	}

	return true, nil
}

func applyRallyToCompanions(owner *users.UserRecord, room *rooms.Room, bonus float64, duration int) {
	for _, mobInstId := range owner.Character.GetCharmIds() {
		mob := mobs.GetInstance(mobInstId)
		if mob == nil {
			continue
		}
		if mob.Character.RoomId != owner.Character.RoomId {
			continue
		}
		mob.Character.AddCondition(characters.ConditionRally, duration, bonus, "rally")
		mob.Character.AddBuff(80, false)
	}
}
