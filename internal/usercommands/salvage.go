package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
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

	// Find item in backpack (not equipped — must unequip first)
	itm, source, found := user.Character.FindItem(rest)
	if !found {
		user.SendText(fmt.Sprintf(
			`<ansi fg="red">You don't have "%s".</ansi>`, rest))
		return true, nil
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

	// Station or tool check
	hasTool := userHasSalvageKit(user)
	usesKit := false

	if hasSalvageReturns && recipe == nil {
		// Tagged items always require tool
		if !hasTool {
			user.SendText(`<ansi fg="red">You need a salvage kit to break that down.</ansi>`)
			return true, nil
		}
		usesKit = true
	} else if recipe != nil && recipe.Station != "" && room.Station != recipe.Station {
		if !hasTool {
			user.SendText(fmt.Sprintf(
				`<ansi fg="red">You need a %s to salvage that, or a salvage kit.</ansi>`,
				strings.ReplaceAll(recipe.Station, "_", " ")))
			return true, nil
		}
		usesKit = true
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
	user.Character.SetMiscData("salvage_uses_kit", usesKit)
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

// userHasSalvageKit checks whether the player has a salvage kit in their
// backpack or component bag.
func userHasSalvageKit(user *users.UserRecord) bool {
	for _, itm := range user.Character.Items {
		if itm.GetSpec().ComponentTag == "salvage-kit" {
			return true
		}
	}
	for _, itm := range user.Character.ComponentItems {
		if itm.GetSpec().ComponentTag == "salvage-kit" {
			return true
		}
	}
	return false
}

// NOTE: Salvage resolution happens in hooks/NewRound_UserRoundTick.go resolveSalvage()
