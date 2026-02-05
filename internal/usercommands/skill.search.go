package usercommands

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

/*
Searcg Skill
Level 1 - Find secret exits or hidden players/mobs
Level 2 - Find objects stashed in the area
Level 3 - ???
Level 4 - You are always aware of hidden players/mobs in the area

(Lvl 1) <ansi fg="skill">search</ansi> Search for secret exits or hidden players/mobs.
(Lvl 2) <ansi fg="skill">search</ansi> Finds objects that may be hidden in the area.
(Lvl 3) <ansi fg="skill">search</ansi> Finds special/unknown "things of interest" in the area.
(Lvl 4) <ansi fg="skill">search</ansi> Doubles your chance of success when searching.
*/
func Search(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	skillLevel := user.Character.GetSkillLevel(skills.Search)

	if skillLevel == 0 {
		user.SendText("You don't know how to search.")
		return true, fmt.Errorf("you don't know how to search")
	}

	if !user.Character.TryCooldown(skills.Search.String(), "2 rounds") {
		user.SendText(
			fmt.Sprintf("You need to wait %d more rounds to use that skill again.", user.Character.GetCooldown(skills.Search.String())),
		)
		return true, fmt.Errorf("you're doing that too often")
	}

	// Fire an event that a skill has been used
	events.AddToQueue(events.SkillUsed{user.UserId, skills.Search, ``})

	// 10% + 1% for every 2 charisma
	searchOddsIn100 := 10 + int(math.Ceil(float64(user.Character.Stats.Charisma.ValueAdj)/2))

	user.SendText("You snoop around for a bit...\n")
	room.SendText(
		fmt.Sprintf(`<ansi fg="username">%s</ansi> is snooping around.`, user.Character.Name),
		user.UserId,
	)

	// Check room exists
	for exit, exitInfo := range room.Exits {
		if exitInfo.Secret {

			roll := util.Rand(100)

			util.LogRoll(`Secret Exit`, roll, searchOddsIn100)

			if roll < searchOddsIn100 {
				user.SendText(fmt.Sprintf(`You found a secret exit: <ansi fg="secret-exit">%s</ansi>`, exit))
			}
		}
	}

	if skillLevel > 2 {
		// Find stashed items
		stashedItems := []string{}
		for _, item := range room.Stash {
			if !item.IsValid() {
				room.RemoveItem(item, true)
			}
			name := item.DisplayName() + ` <ansi fg="item-stashed">(stashed)</ansi>`
			stashedItems = append(stashedItems, name)
		}

		hiddenPlayers := []string{}

		for _, pId := range room.GetPlayers() {
			if pId == user.UserId {
				continue
			}
			if p := users.GetByUserId(pId); p != nil {

				roll := util.Rand(100)

				util.LogRoll(`Hidden Player`, roll, searchOddsIn100)

				if roll < searchOddsIn100 {
					if p.Character.HasBuffFlag(buffs.Hidden) {
						hiddenPlayers = append(hiddenPlayers, p.Character.Name+` <ansi fg="black-bold">(hiding)</ansi>`)
					}
				}
			}
		}

		if len(hiddenPlayers) > 0 {

			details := rooms.GetDetails(room, user)
			details.VisiblePlayers = []string{}

			for _, name := range hiddenPlayers {
				details.VisiblePlayers = append(details.VisiblePlayers,
					characters.FormattedName{
						Name:   name,
						Type:   `username`,
						Suffix: `hidden`,
					}.String(),
				)
			}

			whoTxt, _ := templates.Process("descriptions/who", details, user.UserId)
			user.SendText(whoTxt)

		}

		hiddenMobs := []string{}

		for _, mId := range room.GetMobs() {
			if m := users.GetByUserId(mId); m != nil {

				roll := util.Rand(100)

				util.LogRoll(`Hidden Mob`, roll, searchOddsIn100)

				if roll < searchOddsIn100 {
					if m.Character.HasBuffFlag(buffs.Hidden) {
						hiddenMobs = append(hiddenPlayers, m.Character.Name+` <ansi fg="black-bold">(hiding)</ansi>`)
					}
				}
			}
		}

		if len(hiddenMobs) > 0 {

			details := rooms.GetDetails(room, user)
			details.VisiblePlayers = []string{}

			for _, name := range hiddenMobs {
				details.VisibleMobs = append(details.VisiblePlayers,
					characters.FormattedName{
						Name:   name,
						Type:   `mob`,
						Suffix: `hidden`,
					}.String(),
				)
			}

			whoTxt, _ := templates.Process("descriptions/who", details, user.UserId)
			user.SendText(whoTxt)

		}

		groundDetails := map[string]any{
			`GroundStuff`: stashedItems,
			`IsDark`:      room.GetBiome().IsDark(),
			`IsNight`:     gametime.IsNight(),
		}

		textOut, _ := templates.Process("descriptions/ontheground", groundDetails, user.UserId)
		user.SendText(textOut)
	}

	if skillLevel >= 3 {
		// Find props

	}

	return true, nil
}
