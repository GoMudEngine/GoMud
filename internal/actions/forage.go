package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/forager"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

// ForageOptions parameterizes a forage attempt.
// Empty v1 — biome derives from actor.GetRoom().GetBiome().
type ForageOptions struct{}

// ForageResult is the structured outcome.
type ForageResult struct {
	Found        bool
	ItemId       int
	ItemName     string
	Reason       string
	OnCooldown   bool
	RollHappened bool
}

// Forage runs a Perception+Search forage attempt scoped to the
// actor's current room biome. Cooldown key "forage" shared with
// the player path (6 rounds). UserActor emits the existing
// "you crouch low..." actor-direct message + room broadcast.
// MobActor's actor-direct SendText is a no-op (silent), but the
// room broadcast emits regardless (chunk 3.8) so players can
// observe forager NPCs working in their territory.
func Forage(actor Actor, opts ForageOptions) ForageResult {
	result := ForageResult{}

	char := actor.GetCharacter()
	room := actor.GetRoom()
	if char == nil || room == nil {
		result.Reason = "no character or room"
		return result
	}

	biome := room.GetBiome()
	if _, ok := forager.ForageYields[biome.BiomeId]; !ok {
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem,
				`There is nothing here worth foraging. Try an outdoor area.`)
		}
		result.Reason = "wrong biome"
		return result
	}

	if !char.TryCooldown("forage", "6 rounds") {
		result.OnCooldown = true
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem,
				fmt.Sprintf("You need to wait %d more rounds before you can forage again.",
					char.GetCooldown("forage")))
		}
		return result
	}

	searchScore := CalcSearchScore(char)

	// Actor-direct message: player-only ("you crouch low" reads from
	// the actor's perspective; mobs have no perspective to address).
	if actor.IsPlayer() {
		actor.SendText(messaging.CategorySystem,
			`You crouch low and begin searching the ground carefully...`)
	}
	// Room broadcast: emits for both player and mob actors. Chunk 3.8
	// made this visible for mob foragers (Tova/Halix/Kessa) so players
	// observing their territory can see them working. Username vs
	// mobname ANSI tag picked per actor type.
	nameTag := "mobname"
	if actor.IsPlayer() {
		nameTag = "username"
	}
	room.SendTextVisual(messaging.CategoryMobEmote,
		fmt.Sprintf(`<ansi fg="%s">%s</ansi> is searching the ground for something.`,
			nameTag, char.Name),
		actor.GetUserId(),
	)

	// Zone + weather overlay: player-forage only. Threading these here (and
	// nowhere in the NPC forager path) is the intentional guard against
	// zone/storm-gated ultra-rare reagents leaking into vendor stock via
	// forager NPCs. Weather lookup degrades safely to "" when the weather
	// module isn't installed or the zone is unrecognized.
	weatherType := ""
	if f, ok := GetExportedFunction("GetWeather"); ok {
		if getW, ok := f.(func(string) map[string]any); ok {
			if wx := getW(room.Zone); wx != nil {
				weatherType, _ = wx["type"].(string)
			}
		}
	}

	coreResult := forager.ForageCore(forager.ForageAttempt{
		Biome:       biome.BiomeId,
		SearchScore: searchScore,
		AtNight:     gametime.IsNight(),
		Zone:        room.Zone,
		Weather:     weatherType,
	})

	result.RollHappened = true

	if !coreResult.Found {
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem,
				`You find nothing of use this time.`)
		}
		return result
	}

	newItem := items.New(coreResult.ItemId)
	if !newItem.IsValid() {
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem,
				`You find something, but it crumbles in your hands.`)
		}
		result.Reason = "item invalid"
		return result
	}

	char.StoreItem(newItem)
	if actor.GetUserId() != 0 {
		events.AddToQueue(events.ItemOwnership{
			UserId: actor.GetUserId(),
			Item:   newItem,
			Gained: true,
		})
	}
	actor.OnSkillUse(string(skills.Search))

	if actor.IsPlayer() {
		actor.SendText(messaging.CategorySystem,
			fmt.Sprintf(`You find a <ansi fg="itemname">%s</ansi>.`, newItem.DisplayName()))
	}

	result.Found = true
	result.ItemId = coreResult.ItemId
	result.ItemName = newItem.DisplayName()
	return result
}
