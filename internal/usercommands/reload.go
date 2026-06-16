package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/language"
	"github.com/GoMudEngine/GoMud/internal/mapper"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
* Role Permissions:
* reload <items|biomes|translations|mapcache>   (Admin — data-file reload)
* reload                                         (All — chamber a ranged weapon)
 */
func Reload(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Admin data-file reload retains the `reload <subcommand>` form, gated by
	// role permission. Anything else (bare `reload`, or a non-admin user) falls
	// through to the player-facing ranged-weapon reload below.
	if rest != "" && user.HasRolePermission("reload", true) {
		switch strings.ToLower(rest) {
		case `items`:
			items.LoadDataFiles()
			user.SendText(messaging.CategorySystem, `Items reloaded.`)
			return true, nil
		case `biomes`:
			rooms.LoadBiomeDataFiles()
			user.SendText(messaging.CategorySystem, `Biomes reloaded.`)
			return true, nil
		case `translations`:
			ok := language.ReloadTranslation()
			if !ok {
				user.SendText(messaging.CategorySystem, `Translations reload failed.`)
			} else {
				user.SendText(messaging.CategorySystem, `Translations reloaded.`)
			}
			return true, nil
		case `mapcache`:
			mapper.ClearCache()
			user.SendText(messaging.CategorySystem, `Mapper cache cleared. Next 'map' command will rebuild from current room data.`)
			return true, nil
		case `help`:
			infoOutput, _ := templates.Process("admincommands/help/command.reload", nil, user.UserId)
			user.SendText(messaging.CategorySystem, infoOutput)
			return true, nil
		default:
			user.SendText(messaging.CategorySystem, `Unknown reload command. See <ansi fg="command">reload help</ansi>.`)
			return true, nil
		}
	}

	// Player-facing ranged-weapon reload.
	res := actions.ExecuteReload(&actions.UserActor{User: user, Room: room})

	switch {
	case res.Crafting:
		user.SendText(messaging.CategorySystem, "You're too busy to reload right now.")
		return true, nil

	case res.NoWeapon:
		user.SendText(messaging.CategorySystem, "You don't have a ranged weapon equipped.")
		return true, nil

	case res.AlreadyLoaded:
		user.SendText(messaging.CategorySystem,
			fmt.Sprintf(`Your <ansi fg="itemname">%s</ansi> is already loaded.`, res.WeaponName))
		return true, nil

	case res.NoAmmo:
		user.SendText(messaging.CategorySystem,
			fmt.Sprintf(`You have no <ansi fg="item">%s</ansi> left to load your <ansi fg="itemname">%s</ansi> with.`,
				res.AmmoTag, res.WeaponName))
		return true, nil

	case res.OnCooldown:
		user.SendText(messaging.CategorySystem, "You need a moment before you can reload.")
		return true, nil
	}

	// Success.
	user.SendText(messaging.CategorySystem,
		fmt.Sprintf(`You ready your <ansi fg="itemname">%s</ansi>.`, res.WeaponName))

	if res.BundleEmptied {
		user.SendText(messaging.CategorySystem,
			fmt.Sprintf(`That was the last of your <ansi fg="itemname">%s</ansi>.`, res.AmmoName))
	}

	room.SendText(messaging.CategoryRoomDescription,
		fmt.Sprintf(`<ansi fg="username">%s</ansi> readies their <ansi fg="itemname">%s</ansi>.`,
			user.Character.Name, res.WeaponName),
		user.UserId)

	// Notify the quest engine of a successful `reload` so quests can gate a step
	// on it (Spoke G's "reload when the pouch runs low" beat). Mirrors
	// forage/drink/throw/cast/shoot.
	bridge := questengine.NewGameBridge(user, room.RoomId)
	questengine.GetEngine().Notify("command", questengine.EventDetails{
		UserId:  user.UserId,
		RoomId:  room.RoomId,
		Command: "reload",
	}, bridge, bridge)

	return true, nil
}
