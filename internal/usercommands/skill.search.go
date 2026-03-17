package usercommands

import (
	"fmt"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
Search Skill
Uses Perception + Search skill rank to find hidden things.
Per-discovery gaussian rolls against tier difficulty targets:

	Tier 1 (target 125): Secret exits, hidden containers
	Tier 2 (target 135): Stashed items, hidden players/mobs
	Tier 3 (target 175): Hidden nouns, hidden contents
*/
func Search(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if !user.Character.TryCooldown(`search`, "2 rounds") {
		user.SendText(
			fmt.Sprintf("You need to wait %d more rounds to do that again.", user.Character.GetCooldown(`search`)),
		)
		return true, fmt.Errorf("you're doing that too often")
	}

	// Compute search score: Perception + skill bonus
	searchRank := user.Character.GetSkillLevel(skills.Search)
	searchScore := float64(user.Character.Stats.Perception.ValueAdj) +
		combat.SkillMultiplier(searchRank)*25.0

	user.SendText("You snoop around for a bit...\n")
	room.SendText(
		fmt.Sprintf(`<ansi fg="username">%s</ansi> is snooping around.`, user.Character.Name),
		user.UserId,
	)

	rolledAgainstSomething := false

	// ── Tier 1 (target 125): Secret exits ────────────────────────
	for exitName, exitInfo := range room.Exits {
		if !exitInfo.Secret {
			continue
		}
		rolledAgainstSomething = true
		roll := dice.RollStat(searchScore)
		if roll.Value >= 125 {
			user.SendText(fmt.Sprintf(`You found a secret exit: <ansi fg="secret-exit">%s</ansi>`, exitName))
		}
	}

	// ── Tier 1 (target 125): Hidden containers ──────────────────
	for containerName, container := range room.Containers {
		if !container.Hidden {
			continue
		}
		if user.Character.HasDiscovery(room.RoomId, containerName) {
			continue
		}
		rolledAgainstSomething = true
		roll := dice.RollStat(searchScore)
		if roll.Value >= 125 {
			user.Character.AddDiscovery(room.RoomId, containerName)
			user.SendText(fmt.Sprintf(`You discover a hidden <ansi fg="container">%s</ansi>!`, containerName))
		}
	}

	// ── Tier 2 (target 135): Stashed items ──────────────────────
	stashedItems := []string{}
	for _, item := range room.Stash {
		if !item.IsValid() {
			room.RemoveItem(item, true)
			continue
		}
		rolledAgainstSomething = true
		roll := dice.RollStat(searchScore)
		if roll.Value >= 135 {
			name := item.DisplayName() + ` <ansi fg="item-stashed">(stashed)</ansi>`
			stashedItems = append(stashedItems, name)
		}
	}

	if len(stashedItems) > 0 {
		groundDetails := map[string]any{
			`GroundStuff`: stashedItems,
			`IsDark`:      room.GetBiome().IsDark(),
			`IsNight`:     gametime.IsNight(),
		}
		textOut, _ := templates.Process("descriptions/ontheground", groundDetails, user.UserId)
		user.SendText(textOut)
	}

	// ── Tier 2 (target 135): Hidden players ─────────────────────
	hiddenPlayers := []string{}
	for _, pId := range room.GetPlayers() {
		if pId == user.UserId {
			continue
		}
		p := users.GetByUserId(pId)
		if p == nil || !p.Character.HasBuffFlag(buffs.Hidden) {
			continue
		}
		rolledAgainstSomething = true
		roll := dice.RollStat(searchScore)
		if roll.Value >= 135 {
			hiddenPlayers = append(hiddenPlayers, p.Character.Name+` <ansi fg="black-bold">(hiding)</ansi>`)
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

	// ── Tier 2 (target 135): Hidden mobs ────────────────────────
	hiddenMobs := []string{}
	for _, mId := range room.GetMobs() {
		mob := mobs.GetInstance(mId)
		if mob == nil || !mob.Character.HasBuffFlag(buffs.Hidden) {
			continue
		}
		rolledAgainstSomething = true
		roll := dice.RollStat(searchScore)
		if roll.Value >= 135 {
			hiddenMobs = append(hiddenMobs, mob.Character.Name+` <ansi fg="black-bold">(hiding)</ansi>`)
		}
	}

	if len(hiddenMobs) > 0 {
		details := rooms.GetDetails(room, user)
		details.VisibleMobs = []string{}
		for _, name := range hiddenMobs {
			details.VisibleMobs = append(details.VisibleMobs,
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

	// ── Tier 3 (target 175): Hidden nouns ───────────────────────
	// Sort keys for deterministic output order
	hiddenNounKeys := make([]string, 0, len(room.HiddenNouns))
	for k := range room.HiddenNouns {
		hiddenNounKeys = append(hiddenNounKeys, k)
	}
	sort.Strings(hiddenNounKeys)

	for _, nounKey := range hiddenNounKeys {
		if user.Character.HasDiscovery(room.RoomId, nounKey) {
			continue
		}
		hiddenNoun := room.HiddenNouns[nounKey]
		rolledAgainstSomething = true
		roll := dice.RollStat(searchScore)
		if roll.Value >= 175 {
			user.Character.AddDiscovery(room.RoomId, nounKey)
			user.SendText(fmt.Sprintf(`You discover something: <ansi fg="noun">%s</ansi>`, nounKey))
			user.SendText(hiddenNoun.HiddenDescription)
		}
	}

	// ── Skill progression (anti-botting gate) ───────────────────
	if rolledAgainstSomething {
		user.Character.CheckSkillProgression(string(skills.Search), user.UserId, 1.0)
	}

	return true, nil
}
