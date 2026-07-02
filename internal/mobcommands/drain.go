package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Drain is the vampire's lifesteal attack — it saps the target's vitality,
// applies a bleed condition, and heals the attacker for a fraction of the
// damage dealt.
func Drain(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Must be in combat to use drain; silently skip if not in combat.
	if !mob.Character.IsInCombat() {
		return true, nil
	}

	res := actions.ExecuteDrain(&actions.MobActor{Mob: mob, Room: room})

	// Identity/anatomy refusal: silently swallow so the btree can fall through.
	if res.NotLifeDrainer {
		return true, nil
	}
	// Any other early-exit condition (OnCooldown, NoTarget): silently return.
	if !res.Executed {
		return true, nil
	}

	// Format and send darkness-aware messages.
	target := res.Target
	result := res.MoveResult
	mobName := mob.Character.Name
	dmgDesc := combat.GetDamageDescription(result.Damage, result.TargetMaxHP)

	// Look up target player record for darkness-aware personal messaging.
	var targetUser *users.UserRecord
	if target.UserId > 0 {
		targetUser = users.GetByUserId(target.UserId)
	}
	canSee := targetUser == nil || canSeeInDark(targetUser, room)

	if result.Hit {
		if targetUser != nil {
			if canSee {
				targetUser.SendText(messaging.CategoryHitNaturalSharp, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> plunges into you, sapping your vitality and leeching your life-force! (<ansi fg="damage">%s</ansi>)`, mobName, dmgDesc))
			} else {
				targetUser.SendText(messaging.CategoryHitNaturalSharp, fmt.Sprintf(`Something plunges into you, sapping your vitality and leeching your life-force! (<ansi fg="damage">%s</ansi>)`, dmgDesc))
			}
		}
		sendRoomText(room, messaging.CategoryHitNaturalSharp,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> plunges into <ansi fg="username">%s</ansi> and leeches their vitality!`, mobName, target.Name),
			target.UserId)

		// Note: the mob heals itself; no message needed (player cannot see the
		// vampire's inner restoration without additional flavor investment).
	} else {
		if targetUser != nil {
			if canSee {
				targetUser.SendText(messaging.CategoryHitNaturalSharp, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> reaches for you hungrily, but misses!`, mobName))
			} else {
				targetUser.SendText(messaging.CategoryHitNaturalSharp, `Something reaches for you hungrily, but misses!`)
			}
		}
		sendRoomText(room, messaging.CategoryHitNaturalSharp,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> reaches for <ansi fg="username">%s</ansi> hungrily, but misses!`, mobName, target.Name),
			target.UserId)
	}

	return true, nil
}
