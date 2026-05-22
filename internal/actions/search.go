package actions

import (
	"fmt"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// SearchOptions is intentionally empty v1 — in-room search is the only
// mode. Reserved for future "search container" path.
type SearchOptions struct{}

// SearchStashedItem represents a stashed item discovered by Tier 2.
type SearchStashedItem struct {
	ItemId      int
	DisplayName string
}

// SearchResult is the structured outcome.
type SearchResult struct {
	HiddenExitsFound      []string           // Tier 1 — player flavor
	HiddenContainersFound []string           // Tier 1 — player flavor
	StashedItemsFound     []SearchStashedItem
	HiddenPlayersFound    []int // Tier 2 — user ids
	HiddenMobsFound       []int // Tier 2 — mob instance ids
	HiddenNounsFound      []string // Tier 3 — player flavor

	OnCooldown bool
	Reason     string
}

// Search rolls Perception+Search per discovery candidate in the room.
// UserActor receives template-rendered output; MobActor is silent
// (no broadcast, no template). Cooldown is shared with the player path
// (2 rounds on the "search" key).
func Search(actor Actor, opts SearchOptions) SearchResult {
	result := SearchResult{}
	char := actor.GetCharacter()
	room := actor.GetRoom()
	if char == nil || room == nil {
		return result
	}

	if !char.TryCooldown("search", "2 rounds") {
		result.OnCooldown = true
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem,
				fmt.Sprintf("You need to wait %d more rounds to do that again.",
					char.GetCooldown("search")))
		}
		return result
	}

	searchScore := CalcSearchScore(char)

	if actor.IsPlayer() {
		actor.SendText(messaging.CategorySystem, "You snoop around for a bit...\n")
		room.SendTextVisual(messaging.CategoryMobEmote,
			fmt.Sprintf(`<ansi fg="username">%s</ansi> is snooping around.`, char.Name),
			actor.GetUserId(),
		)
	}

	rolledAgainstSomething := false

	// ── Tier 1 (target 125): Secret exits ────────────────────────
	for exitName, exitInfo := range room.Exits {
		if !exitInfo.Secret {
			continue
		}
		rolledAgainstSomething = true
		roll := dice.RollStat(searchScore)
		if roll.Value >= 125.0 {
			result.HiddenExitsFound = append(result.HiddenExitsFound, exitName)
			if actor.IsPlayer() {
				actor.SendText(messaging.CategorySystem,
					fmt.Sprintf(`You found a secret exit: <ansi fg="secret-exit">%s</ansi>`, exitName))
			}
		}
	}

	// ── Tier 1 (target 125): Hidden containers ──────────────────
	for containerName, container := range room.Containers {
		if !container.Hidden {
			continue
		}
		if char.HasDiscovery(room.RoomId, containerName) {
			continue
		}
		rolledAgainstSomething = true
		roll := dice.RollStat(searchScore)
		if roll.Value >= 125.0 {
			char.AddDiscovery(room.RoomId, containerName)
			result.HiddenContainersFound = append(result.HiddenContainersFound, containerName)
			if actor.IsPlayer() {
				actor.SendText(messaging.CategorySystem,
					fmt.Sprintf(`You discover a hidden <ansi fg="container">%s</ansi>!`, containerName))
			}
		}
	}

	// ── Tier 2 (target 135): Stashed items ──────────────────────
	stashedNames := []string{}
	for _, item := range room.Stash {
		if !item.IsValid() {
			room.RemoveItem(item, true)
			continue
		}
		rolledAgainstSomething = true
		roll := dice.RollStat(searchScore)
		if roll.Value >= 135.0 {
			result.StashedItemsFound = append(result.StashedItemsFound, SearchStashedItem{
				ItemId:      item.ItemId,
				DisplayName: item.DisplayName(),
			})
			if actor.IsPlayer() {
				stashedNames = append(stashedNames, item.DisplayName()+` <ansi fg="item-stashed">(stashed)</ansi>`)
			}
		}
	}
	if actor.IsPlayer() && len(stashedNames) > 0 {
		details := map[string]any{
			"GroundStuff": stashedNames,
			"IsDark":      room.GetBiome().IsDark(),
			"IsNight":     gametime.IsNight(),
		}
		text, _ := templates.Process("descriptions/ontheground", details, actor.GetUserId())
		actor.SendText(messaging.CategorySystem, text)
	}

	// ── Tier 2 (target 135): Hidden players ─────────────────────
	hiddenPlayerNames := []string{}
	for _, pId := range room.GetPlayers() {
		if pId == actor.GetUserId() {
			continue
		}
		p := users.GetByUserId(pId)
		if p == nil || !p.Character.IsHidden() {
			continue
		}
		rolledAgainstSomething = true
		roll := dice.RollStat(searchScore)
		if roll.Value >= 135.0 {
			result.HiddenPlayersFound = append(result.HiddenPlayersFound, pId)
			if actor.IsPlayer() {
				hiddenPlayerNames = append(hiddenPlayerNames,
					p.Character.Name+` <ansi fg="black-bold">(hiding)</ansi>`)
			}
		}
	}
	if actor.IsPlayer() && len(hiddenPlayerNames) > 0 {
		details := rooms.GetDetails(room, users.GetByUserId(actor.GetUserId()))
		details.VisiblePlayers = []string{}
		for _, name := range hiddenPlayerNames {
			details.VisiblePlayers = append(details.VisiblePlayers,
				characters.FormattedName{Name: name, Type: "username", Suffix: "hidden"}.String())
		}
		text, _ := templates.Process("descriptions/who", details, actor.GetUserId())
		actor.SendText(messaging.CategorySystem, text)
	}

	// ── Tier 2 (target 135): Hidden mobs ────────────────────────
	hiddenMobNames := []string{}
	for _, mId := range room.GetMobs() {
		m := mobs.GetInstance(mId)
		if m == nil || !m.Character.IsHidden() {
			continue
		}
		rolledAgainstSomething = true
		roll := dice.RollStat(searchScore)
		if roll.Value >= 135.0 {
			result.HiddenMobsFound = append(result.HiddenMobsFound, mId)
			if actor.IsPlayer() {
				hiddenMobNames = append(hiddenMobNames,
					m.Character.Name+` <ansi fg="black-bold">(hiding)</ansi>`)
			}
		}
	}
	if actor.IsPlayer() && len(hiddenMobNames) > 0 {
		details := rooms.GetDetails(room, users.GetByUserId(actor.GetUserId()))
		details.VisibleMobs = []string{}
		for _, name := range hiddenMobNames {
			details.VisibleMobs = append(details.VisibleMobs,
				characters.FormattedName{Name: name, Type: "mob", Suffix: "hidden"}.String())
		}
		text, _ := templates.Process("descriptions/who", details, actor.GetUserId())
		actor.SendText(messaging.CategorySystem, text)
	}

	// ── Tier 3 (target 175): Hidden nouns ───────────────────────
	// Sort keys for deterministic output order.
	nounKeys := make([]string, 0, len(room.HiddenNouns))
	for k := range room.HiddenNouns {
		nounKeys = append(nounKeys, k)
	}
	sort.Strings(nounKeys)

	for _, nounKey := range nounKeys {
		if char.HasDiscovery(room.RoomId, nounKey) {
			continue
		}
		hiddenNoun := room.HiddenNouns[nounKey]
		rolledAgainstSomething = true
		roll := dice.RollStat(searchScore)
		if roll.Value >= 175.0 {
			char.AddDiscovery(room.RoomId, nounKey)
			result.HiddenNounsFound = append(result.HiddenNounsFound, nounKey)
			if actor.IsPlayer() {
				actor.SendText(messaging.CategorySystem,
					fmt.Sprintf(`You discover something: <ansi fg="noun">%s</ansi>`, nounKey))
				actor.SendText(messaging.CategorySystem, hiddenNoun.HiddenDescription)
			}
		}
	}

	// ── Skill progression (anti-botting gate) ───────────────────
	if rolledAgainstSomething {
		actor.OnSkillUse(string(skills.Search))
	}

	return result
}
