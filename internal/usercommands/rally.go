package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Rally(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	result := actions.ExecuteRally(&actions.UserActor{User: user, Room: room})

	if result.Crafting {
		user.SendText(messaging.CategorySystem, `<ansi fg="red">You can't rally while focused on your work. Finish or be interrupted first.</ansi>`)
		return true, nil
	}
	if result.AlreadyActive {
		user.SendText(messaging.CategorySystem, "You're already rallied — save it for when it matters.")
		return true, nil
	}
	if result.OnCooldown {
		user.SendText(messaging.CategorySystem, "You need a moment to recover before attempting another special move.")
		return true, nil
	}
	if !result.Executed {
		return true, nil
	}

	user.SendText(messaging.CategorySystem, `<ansi fg="cyan-bold">You rally your allies with an inspiring shout that steadies their resolve!</ansi>`)
	room.SendTextVisual(messaging.CategoryRally,
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
			memberUser.SendText(messaging.CategorySystem,
				fmt.Sprintf(`<ansi fg="cyan-bold"><ansi fg="username">%s</ansi>'s rallying cry steadies your nerves!</ansi>`, user.Character.Name))
			applyRallyToCompanions(memberUser, room, result.Bonus, result.Duration)
		}
	}

	// Fan out to caster's own companions in the room.
	applyRallyToCompanions(user, room, result.Bonus, result.Duration)

	// Resonant Larynx (shout-stacking): the same breath also looses a war cry,
	// under the rally cooldown already paid. ApplyWarcryEffect applies the war
	// cry to the caster; fan it to the same allies the rally reached.
	if mutations.HasMutationFlag(user.Character.Mutations, "shout-stacking") {
		wb, wd := actions.ApplyWarcryEffect(user.Character)
		user.SendText(messaging.CategorySystem, `<ansi fg="red-bold">Your layered voice looses a thunderous war cry in the same breath!</ansi>`)
		room.SendTextVisual(messaging.CategoryWarcry,
			fmt.Sprintf(`<ansi fg="red-bold"><ansi fg="username">%s</ansi>'s cry carries a war cry within it!</ansi>`, user.Character.Name),
			user.UserId,
		)
		if party := parties.Get(user.UserId); party != nil {
			for _, memberId := range party.GetMembers() {
				if memberId == user.UserId {
					continue
				}
				memberUser := users.GetByUserId(memberId)
				if memberUser == nil || memberUser.Character.RoomId != user.Character.RoomId {
					continue
				}
				memberUser.Character.AddCondition(characters.ConditionWarcry, wd, wb, "warcry")
				memberUser.Character.AddBuff(79, false)
				applyWarcryToCompanions(memberUser, room, wb, wd)
			}
		}
		applyWarcryToCompanions(user, room, wb, wd)
	}

	// Rhetoric progression is awarded inside actions.ExecuteRally,
	// so the mob wrapper gets it too.

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
