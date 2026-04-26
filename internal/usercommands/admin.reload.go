package usercommands

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/language"
	"github.com/GoMudEngine/GoMud/internal/mapper"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
* Role Permissions:
* reload 				(All)
 */
func Reload(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if rest == "" {
		infoOutput, _ := templates.Process("admincommands/help/command.reload", nil, user.UserId)
		user.SendText(infoOutput)
		return true, nil
	}

	switch strings.ToLower(rest) {
	case `items`:
		items.LoadDataFiles()
		user.SendText(`Items reloaded.`)
	case `biomes`:
		rooms.LoadBiomeDataFiles()
		user.SendText(`Biomes reloaded.`)
	case `translations`:
		ok := language.ReloadTranslation()
		if !ok {
			user.SendText(`Translations reload failed.`)
		} else {
			user.SendText(`Translations reloaded.`)
		}
	case `mapcache`:
		mapper.ClearCache()
		user.SendText(`Mapper cache cleared. Next 'map' command will rebuild from current room data.`)
	default:
		user.SendText(`Unknown reload command.`)
	}
	return true, nil
}
