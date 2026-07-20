package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Bash(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Must be in combat to use bash; silently skip if not in combat.
	if !mob.Character.IsInCombat() {
		return true, nil
	}

	// Delegate core bash logic to the shared action.
	bashResult := actions.ExecuteBash(&actions.MobActor{Mob: mob, Room: room})

	// Any early-exit condition: silently return.
	if !bashResult.Executed {
		return true, nil
	}

	// Format and send darkness-aware messages.
	target := bashResult.Target
	result := bashResult.MoveResult
	mobName := mob.Character.Name
	dmgDesc := combat.GetDamageDescription(result.Damage, result.TargetMaxHP)

	// Look up the target player record for darkness-aware personal messaging.
	var targetUser *users.UserRecord
	if target.UserId > 0 {
		targetUser = users.GetByUserId(target.UserId)
	}
	canSee := targetUser == nil || canSeeInDark(targetUser, room)

	// Natural bashers (elementals, golems) slam instead of shield-bashing.
	bashLabel := "shield bash"
	bashVerb := "bashes"
	bashWith := "with their shield"
	if sp := species.GetSpecies(mob.Character.SpeciesId); sp != nil && sp.NaturalBash {
		bashLabel = "crushing slam"
		bashVerb = "slams into"
		bashWith = "with tremendous force"
	}

	if result.Hit {
		if result.KnockedDown {
			if targetUser != nil {
				if canSee {
					targetUser.SendText(messaging.CategoryBash, fmt.Sprintf(`<ansi fg="mobname">%s</ansi>'s <ansi fg="yellow-bold">%s</ansi> knocks you to the ground! (<ansi fg="damage">%s</ansi>)`, mobName, bashLabel, dmgDesc))
				} else {
					targetUser.SendText(messaging.CategoryBash, fmt.Sprintf(`Something's <ansi fg="yellow-bold">%s</ansi> knocks you to the ground! (<ansi fg="damage">%s</ansi>)`, bashLabel, dmgDesc))
				}
			}
			room.SendTextVisual(messaging.CategoryBash,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi>'s <ansi fg="yellow-bold">%s</ansi> knocks <ansi fg="username">%s</ansi> to the ground!`, mobName, bashLabel, target.Name),
				target.UserId)
		} else {
			if targetUser != nil {
				if canSee {
					targetUser.SendText(messaging.CategoryBash, fmt.Sprintf(`<ansi fg="mobname">%s</ansi>'s <ansi fg="yellow-bold">%s</ansi> strikes you! (<ansi fg="damage">%s</ansi>)`, mobName, bashLabel, dmgDesc))
				} else {
					targetUser.SendText(messaging.CategoryBash, fmt.Sprintf(`Something's <ansi fg="yellow-bold">%s</ansi> strikes you! (<ansi fg="damage">%s</ansi>)`, bashLabel, dmgDesc))
				}
			}
			room.SendTextVisual(messaging.CategoryBash,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> %s <ansi fg="username">%s</ansi> %s!`, mobName, bashVerb, target.Name, bashWith),
				target.UserId)
		}
	} else {
		if targetUser != nil {
			if canSee {
				targetUser.SendText(messaging.CategoryBash, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> attempts a %s, but misses!`, mobName, bashLabel))
			} else {
				targetUser.SendText(messaging.CategoryBash, fmt.Sprintf(`Something attempts a %s, but misses!`, bashLabel))
			}
		}
		room.SendTextVisual(messaging.CategoryBash,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> attempts to bash <ansi fg="username">%s</ansi>, but misses!`, mobName, target.Name),
			target.UserId)
	}

	return true, nil
}
