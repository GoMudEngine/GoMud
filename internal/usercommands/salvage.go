package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Salvage handles the `salvage` command — breaks down items for materials.
func Salvage(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	rest = strings.TrimSpace(rest)

	if rest == "" {
		user.SendText(`<ansi fg="command">salvage <item></ansi> - Break down an item for materials.`)
		return true, nil
	}

	// Already busy?
	if user.Character.IsCrafting() {
		user.SendText(`<ansi fg="red">You're already busy working on something.</ansi>`)
		return true, nil
	}

	// Try inventory item first; fall through to room corpses if no
	// inventory match.
	itm, source, found := user.Character.FindItem(rest)
	if !found {
		corpse, corpseFound := room.FindCorpse(rest)
		if !corpseFound {
			user.SendText(fmt.Sprintf(
				`<ansi fg="red">You don't have "%s" and there's no corpse of that name here.</ansi>`, rest))
			return true, nil
		}
		return startCorpseSalvage(user, corpse)
	}

	// Require item to be in backpack, not equipped
	if source != "in your backpack" {
		user.SendText(`<ansi fg="red">You need to remove that before you can salvage it.</ansi>`)
		return true, nil
	}

	spec := itm.GetSpec()

	// Spoiled/declining potions can be salvaged for binding paste
	isSpoiledPotion := false
	if spec.Type == items.Potion && spec.Aging.HasAging() && itm.CraftedRound > 0 {
		currentRound := util.GetRoundCount()
		var elapsed uint64
		if currentRound >= itm.CraftedRound {
			elapsed = currentRound - itm.CraftedRound
		}
		bottleMult := itm.BottleMultiplier
		if bottleMult <= 0 {
			bottleMult = spec.BottleAgingMultiplier
		}
		effSpeed := items.CalcEffectiveAgingSpeed(bottleMult, itm.CraftSkill)
		phase, _ := items.GetAgingPhase(elapsed, spec.Aging, effSpeed)
		if phase == items.PhaseSpoiled || phase == items.PhaseDeclining {
			isSpoiledPotion = true
		}
	}

	// Determine salvage source: recipe reverse-lookup or tagged returns
	recipe := crafting.GetRecipeByOutputItemId(spec.ItemId)
	hasSalvageReturns := len(spec.SalvageReturns) > 0

	if recipe == nil && !hasSalvageReturns && !isSpoiledPotion {
		user.SendText(`<ansi fg="red">You can't find anything useful to salvage from that.</ansi>`)
		return true, nil
	}

	// Calculate rounds based on ingredient gold value
	bal := configs.GetBalanceConfig()
	var totalGold int
	if isSpoiledPotion {
		totalGold = 1 // Quick salvage — just scraping residue
	} else if recipe != nil {
		totalGold = crafting.CalcIngredientGoldValue(recipe.Ingredients)
	} else {
		totalGold = crafting.CalcSalvageReturnGoldValue(spec.SalvageReturns)
	}
	rounds := crafting.CalcSalvageRounds(totalGold,
		int(bal.SalvageGoldPerRound), int(bal.SalvageMaxRounds))

	// Store salvage target info for resolution
	user.Character.SetMiscData("salvage_item_uuid", itm.UUID.String())
	user.Character.SetMiscData("salvage_spoiled_potion", isSpoiledPotion)

	// Start multi-round salvage activity using CraftingState
	user.Character.CraftingState = &characters.CraftingState{
		RecipeId:    fmt.Sprintf("salvage:%d", spec.ItemId),
		RoundsTotal: rounds,
	}

	user.SendText(fmt.Sprintf(
		`<ansi fg="yellow">You begin carefully disassembling the <ansi fg="itemname">%s</ansi>...</ansi>`,
		itm.DisplayName()))

	return true, nil
}

// startCorpseSalvage initiates a corpse salvage activity. Called from
// Salvage when the player's argument matches a room corpse rather than
// an inventory item.
func startCorpseSalvage(user *users.UserRecord, corpse rooms.Corpse) (bool, error) {

	// Player corpses are out of scope for v1.
	if corpse.MobId <= 0 {
		user.SendText(`<ansi fg="red">You can't bring yourself to salvage that.</ansi>`)
		return true, nil
	}

	mobSpec := mobs.GetMobSpec(mobs.MobId(corpse.MobId))
	if mobSpec == nil {
		user.SendText(`<ansi fg="red">Something is wrong with that corpse.</ansi>`)
		return true, nil
	}

	returns := crafting.LookupCorpseSalvage(mobSpec.Groups)
	if len(returns) == 0 {
		user.SendText(`<ansi fg="red">There's nothing useful to recover here.</ansi>`)
		return true, nil
	}

	bal := configs.GetBalanceConfig()
	totalGold := crafting.CalcSalvageReturnGoldValue(returns)
	rounds := crafting.CalcSalvageRounds(totalGold,
		int(bal.SalvageGoldPerRound), int(bal.SalvageMaxRounds))

	// Stash corpse identity for the resolver. mobid + roundCreated
	// uniquely identifies the corpse within the room. Store as int to
	// avoid type-assertion issues if MiscData ever round-trips through
	// YAML (uint64 can come back coerced).
	user.Character.SetMiscData("salvage_corpse_round_created", int(corpse.RoundCreated))
	user.Character.SetMiscData("salvage_corpse_name", corpse.Character.Name)

	user.Character.CraftingState = &characters.CraftingState{
		RecipeId:    fmt.Sprintf("salvage-corpse:%d", corpse.MobId),
		RoundsTotal: rounds,
	}

	user.SendText(fmt.Sprintf(
		`<ansi fg="yellow">You begin carefully working over the <ansi fg="mobname">%s corpse</ansi>...</ansi>`,
		corpse.Character.Name))

	return true, nil
}

// NOTE: Salvage resolution happens in hooks/NewRound_UserRoundTick.go resolveSalvage()
