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

func Warcry(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	result := actions.ExecuteWarcry(&actions.UserActor{User: user, Room: room})

	if result.Crafting {
		user.SendText(messaging.CategorySystem, `<ansi fg="red">You can't muster a warcry while focused on your work. Finish or be interrupted first.</ansi>`)
		return true, nil
	}

	if result.AlreadyActive {
		user.SendText(messaging.CategorySystem, "Your warcry still echoes — you can't shout it louder.")
		return true, nil
	}

	if result.OnCooldown {
		user.SendText(messaging.CategorySystem, "You need a moment to recover before attempting another special move.")
		return true, nil
	}

	user.SendText(messaging.CategorySystem, `<ansi fg="red-bold">You let out a thunderous warcry that ignites the fighting spirit of your allies!</ansi>`)
	room.SendTextVisual(messaging.CategoryWarcry,
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
					memberUser.SendText(messaging.CategorySystem,
						fmt.Sprintf(`<ansi fg="red-bold"><ansi fg="username">%s</ansi>'s warcry stirs your blood!</ansi>`, user.Character.Name))

					// Apply to this party member's companions in the room
					applyWarcryToCompanions(memberUser, room, result.Bonus, result.Duration)
				}
			}
		}
	}

	// Apply to caster's own companions in the room
	applyWarcryToCompanions(user, room, result.Bonus, result.Duration)

	// Resonant Larynx (shout-stacking): the same breath also looses a rally,
	// under the war-cry cooldown already paid. ApplyRallyEffect applies the
	// rally to the caster; fan it to the same allies the war cry reached.
	if mutations.HasMutationFlag(user.Character.Mutations, "shout-stacking") {
		rb, rd := actions.ApplyRallyEffect(user.Character)
		user.SendText(messaging.CategorySystem, `<ansi fg="cyan-bold">Your layered voice weaves a rallying cry into the same breath!</ansi>`)
		room.SendTextVisual(messaging.CategoryRally,
			fmt.Sprintf(`<ansi fg="cyan-bold"><ansi fg="username">%s</ansi>'s cry carries a rally within it!</ansi>`, user.Character.Name),
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
				memberUser.Character.AddCondition(characters.ConditionRally, rd, rb, "rally")
				memberUser.Character.AddBuff(80, false)
				applyRallyToCompanions(memberUser, room, rb, rd)
			}
		}
		applyRallyToCompanions(user, room, rb, rd)
	}

	// Rhetoric progression is awarded inside actions.ExecuteWarcry,
	// so the mob wrapper gets it too.

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
