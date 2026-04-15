package usercommands

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Warcry(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	cfg := configs.GetBalanceConfig()
	if !user.Character.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		user.SendText("You need a moment to recover before attempting another special move.")
		return true, nil
	}

	// Calculate magnitude: 0.05 + 0.15 * sqrt((rhetoric/75) * (charisma/175))
	rhetoric := float64(user.Character.GetSkillLevel(skills.Rhetoric))
	charisma := float64(user.Character.Stats.Charisma.ValueAdj)
	bonus := 0.05 + 0.15*math.Sqrt((rhetoric/75.0)*(charisma/175.0))
	if bonus < 0.05 {
		bonus = 0.05
	}
	if bonus > 0.20 {
		bonus = 0.20
	}

	duration := 25

	// Apply to self
	user.Character.AddCondition(characters.ConditionWarcry, duration, bonus, "warcry")
	user.Character.AddBuff(79, false)

	user.SendText(`<ansi fg="red-bold">You let out a thunderous warcry that ignites the fighting spirit of your allies!</ansi>`)
	room.SendTextVisual(
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
					memberUser.Character.AddCondition(characters.ConditionWarcry, duration, bonus, "warcry")
					memberUser.Character.AddBuff(79, false)
					memberUser.SendText(
						fmt.Sprintf(`<ansi fg="red-bold"><ansi fg="username">%s</ansi>'s warcry stirs your blood!</ansi>`, user.Character.Name))

					// Apply to this party member's companions in the room
					applyWarcryToCompanions(memberUser, room, bonus, duration)
				}
			}
		}
	}

	// Apply to caster's own companions in the room
	applyWarcryToCompanions(user, room, bonus, duration)

	// Skill and stat progression
	// OnSkillUse handles rhetoric progression + charisma stat use internally.
	// In combat: always fire. Out of combat: 50% chance (soft incentive).
	if user.Character.Aggro != nil {
		user.Character.OnSkillUse(string(skills.Rhetoric), user.UserId)
	} else if util.Rand(100) < 50 {
		user.Character.OnSkillUse(string(skills.Rhetoric), user.UserId)
	}

	// Set combat wait if in combat
	if user.Character.Aggro != nil {
		user.Character.Aggro.RoundsWaiting = 1
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
