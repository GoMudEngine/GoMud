package actions

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

// SalvageOptions identifies the salvage target.
//
//   - TargetCorpse: salvage an eligible corpse in the room. Default
//     mode is "first eligible". When TargetCorpseMobId is also set
//     (non-zero), filter for the specific corpse with that MobId +
//     RoundCreated (used by the player path, which started the
//     activity against a specific corpse).
//   - TargetItemUuid: salvage a specific item from actor inventory
//     by UUID. Used by the player path (resolved on the final
//     activity tick from SalvagingData.ItemUuid).
//   - SpoiledPotion: hint that this is a spoiled-potion salvage —
//     overrides recipe lookup and yields binding paste. Set by the
//     player wrapper when starting the activity.
//
// Exactly one of TargetCorpse or TargetItemUuid!="" should be set.
type SalvageOptions struct {
	TargetCorpse             bool
	TargetCorpseMobId        int    // 0 = first eligible; non-zero = specific corpse
	TargetCorpseRoundCreated uint64 // disambiguator paired with TargetCorpseMobId
	TargetItemUuid           string
	SpoiledPotion            bool
}

// SalvageResult is the structured outcome of one salvage tick.
type SalvageResult struct {
	Succeeded    bool
	MaterialIds  []int
	Reason       string
	RollHappened bool
}

// Salvage runs one tick of the salvage roll. Single-tick by design
// — player-side multi-round UX wraps this via the Activity machine
// + per-tick hook in NewRound_UserRoundTick.go. UserActor emits
// per-tick progress text; MobActor silent. Skill progression via
// actor.OnSkillUse("salvage").
func Salvage(actor Actor, opts SalvageOptions) SalvageResult {
	result := SalvageResult{}

	char := actor.GetCharacter()
	room := actor.GetRoom()
	if char == nil || room == nil {
		result.Reason = "no character or room"
		return result
	}

	if !opts.TargetCorpse && opts.TargetItemUuid == "" {
		result.Reason = "no target"
		return result
	}

	bal := configs.GetBalanceConfig()
	salvageSkill := char.GetSkillLevel(skills.Salvage)
	chance := crafting.CalcSalvageChance(salvageSkill,
		float64(bal.SalvageMinChance),
		float64(bal.SalvageMaxChance),
		int(bal.SalvageSoftCap))

	if opts.TargetCorpse {
		return salvageCorpse(actor, room, opts, chance)
	}
	return salvageItem(actor, opts.TargetItemUuid, opts.SpoiledPotion, chance)
}

// salvageCorpse handles the corpse-target path. Finds the target
// corpse (specific by MobId+RoundCreated, or first eligible),
// rolls returns, removes the corpse, stores materials.
func salvageCorpse(actor Actor, room *rooms.Room, opts SalvageOptions, chance float64) SalvageResult {
	result := SalvageResult{}

	var target rooms.Corpse
	found := false
	for _, c := range room.Corpses {
		if c.Prunable {
			continue
		}
		// Player corpses are out of scope.
		if c.MobId <= 0 {
			continue
		}
		// Specific-corpse filter (player path).
		if opts.TargetCorpseMobId != 0 {
			if c.MobId != opts.TargetCorpseMobId ||
				c.RoundCreated != opts.TargetCorpseRoundCreated {
				continue
			}
		}
		mobSpec := mobs.GetMobSpec(mobs.MobId(c.MobId))
		if mobSpec == nil {
			continue
		}
		if len(crafting.LookupCorpseSalvage(mobSpec.Groups)) > 0 {
			target = c
			found = true
			break
		}
	}

	if !found {
		result.Reason = "no eligible corpse"
		return result
	}

	result.RollHappened = true

	mobSpec := mobs.GetMobSpec(mobs.MobId(target.MobId))
	returns := crafting.LookupCorpseSalvage(mobSpec.Groups)
	recovered := crafting.RollSalvageReturnsFromSpec(returns, chance)

	room.RemoveCorpse(target)
	actor.OnSkillUse(string(skills.Salvage))

	storeRecovered(actor, recovered, &result)

	if actor.IsPlayer() {
		if len(recovered) > 0 {
			actor.SendText(messaging.CategorySystem, fmt.Sprintf(
				`<ansi fg="green">You salvage the <ansi fg="mobname">%s corpse</ansi> and recover: %s.</ansi>`,
				target.Character.Name,
				formatRecovered(recovered)))
		} else {
			actor.SendText(messaging.CategorySystem, fmt.Sprintf(
				`<ansi fg="red">You attempt to salvage the <ansi fg="mobname">%s corpse</ansi> but recover nothing useful.</ansi>`,
				target.Character.Name))
		}
	} else if room.PlayerCt() > 0 {
		// Mob-side room flavor (matches existing mobcommands/salvage.go).
		room.SendTextVisual(messaging.CategoryMobIdle, fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi> kneels by the carcass and cuts strips of hide from it.`,
			actor.GetName()))
	}

	result.Succeeded = true
	return result
}

// salvageItem handles the item-target path (by UUID). Mirrors the
// logic of the prior resolveSalvageFromData in
// hooks/NewRound_UserRoundTick.go, adapted for the actor interface.
func salvageItem(actor Actor, uuid string, spoiledPotion bool, chance float64) SalvageResult {
	result := SalvageResult{}
	char := actor.GetCharacter()

	// Find item in inventory by UUID.
	var targetItem items.Item
	found := false
	for _, itm := range char.Items {
		if itm.UUID.String() == uuid {
			targetItem = itm
			found = true
			break
		}
	}
	if !found {
		if actor.IsPlayer() {
			actor.SendText(messaging.CategoryError,
				`<ansi fg="red">The item you were salvaging is no longer in your backpack.</ansi>`)
		}
		result.Reason = "item not found"
		return result
	}

	itemId := targetItem.ItemId
	spec := items.GetItemSpec(itemId)
	if spec == nil {
		if actor.IsPlayer() {
			actor.SendText(messaging.CategoryError,
				`<ansi fg="red">Something went wrong with your salvage attempt.</ansi>`)
		}
		result.Reason = "spec not found"
		return result
	}

	// Roll returns from spoiled-potion branch, recipe lookup, or
	// tagged salvage_returns.
	var recovered []crafting.RecipeIngredient
	if spoiledPotion {
		qty := 1
		if chance > 0.5 {
			qty = 2
		}
		recovered = []crafting.RecipeIngredient{
			{ItemTag: "binding-paste", Quantity: qty},
		}
	} else {
		recipe := crafting.GetRecipeByOutputItemId(itemId)
		if recipe != nil {
			recovered = crafting.RollSalvageReturns(recipe.Ingredients, chance)
		} else if len(spec.SalvageReturns) > 0 {
			recovered = crafting.RollSalvageReturnsFromSpec(spec.SalvageReturns, chance)
		}
	}

	result.RollHappened = true

	// Always destroy the item (matches existing behavior).
	char.RemoveItem(targetItem)
	actor.OnSkillUse(string(skills.Salvage))

	storeRecovered(actor, recovered, &result)

	if actor.IsPlayer() {
		if len(recovered) > 0 {
			actor.SendText(messaging.CategorySystem, fmt.Sprintf(
				`<ansi fg="green">You salvage the <ansi fg="itemname">%s</ansi> and recover: %s.</ansi>`,
				targetItem.DisplayName(),
				formatRecovered(recovered)))
		} else {
			actor.SendText(messaging.CategorySystem, fmt.Sprintf(
				`<ansi fg="red">You attempt to salvage the <ansi fg="itemname">%s</ansi> but recover nothing useful.</ansi>`,
				targetItem.DisplayName()))
		}
	}

	result.Succeeded = true
	return result
}

// storeRecovered creates the material items and stores them in the
// actor's inventory, populating result.MaterialIds.
func storeRecovered(actor Actor, recovered []crafting.RecipeIngredient, result *SalvageResult) {
	char := actor.GetCharacter()
	for _, ing := range recovered {
		for i := 0; i < ing.Quantity; i++ {
			matSpec := items.FindSpecByComponentTag(ing.ItemTag)
			if matSpec == nil {
				continue
			}
			newItem := items.New(matSpec.ItemId)
			char.StoreItem(newItem)
			result.MaterialIds = append(result.MaterialIds, matSpec.ItemId)
		}
	}
}

// formatRecovered builds the comma-separated yield list for player
// flavor text. Matches the format used by the prior player path in
// hooks/NewRound_UserRoundTick.go.
func formatRecovered(recovered []crafting.RecipeIngredient) string {
	parts := make([]string, 0, len(recovered))
	for _, ing := range recovered {
		parts = append(parts, fmt.Sprintf("%dx %s", ing.Quantity, ing.ItemTag))
	}
	return strings.Join(parts, ", ")
}
