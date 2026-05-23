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
// snooping emote + "you find X" message; MobActor SendText is a
// no-op (silent).
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

	if actor.IsPlayer() {
		actor.SendText(messaging.CategorySystem,
			`You crouch low and begin searching the ground carefully...`)
		room.SendTextVisual(messaging.CategoryMobEmote,
			fmt.Sprintf(`<ansi fg="username">%s</ansi> is searching the ground for something.`,
				char.Name),
			actor.GetUserId(),
		)
	}

	coreResult := forager.ForageCore(forager.ForageAttempt{
		Biome:       biome.BiomeId,
		SearchScore: searchScore,
		AtNight:     gametime.IsNight(),
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
