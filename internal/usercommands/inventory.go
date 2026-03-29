package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Inventory(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	itemNames := []string{}
	itemNamesFormatted := []string{}

	itemList := []items.Item{}

	typeSearchTerms := map[string]items.ItemType{
		`weapons`:   items.Weapon,
		`offhand`:   items.Offhand,
		`shields`:   items.Offhand,
		`head`:      items.Head,
		`neck`:      items.Neck,
		`body`:      items.Body,
		`armor`:     items.Body,
		`belts`:     items.Belt,
		`gloves`:    items.Gloves,
		`rings`:     items.Ring,
		`legs`:      items.Legs,
		`pants`:     items.Legs,
		`leggings`:  items.Legs,
		`feet`:      items.Feet,
		`potions`:   items.Potion,
		`food`:      items.Food,
		`drinks`:    items.Drink,
		`scrolls`:   items.Scroll,
		`grenades`:  items.Grenade,
		`keys`:      items.Key,
		`gemstones`: items.Gemstone,
	}

	subtypeSearchTerms := map[string]items.ItemSubType{
		`armor`:        items.Wearable,
		`clothing`:     items.Wearable,
		`clothes`:      items.Wearable,
		`wearable`:     items.Wearable,
		`drinks`:       items.Drinkable,
		`drinkable`:    items.Drinkable,
		`food`:         items.Edible,
		`usable`:       items.Usable,
		`throwable`:    items.Throwable,
		`bloudgeoning`: items.Bludgeoning,
		`cleaving`:     items.Cleaving,
		`stabbing`:     items.Stabbing,
		`slashing`:     items.Slashing,
		`shooting`:     items.Shooting,
		`claws`:        items.Claws,
	}

	for _, item := range user.Character.GetAllBackpackItems() {

		foundMatch := false
		if len(rest) > 0 {

			for term, itemType := range typeSearchTerms {
				if strings.HasPrefix(term, rest) {
					if item.GetSpec().Type == itemType {
						itemList = append(itemList, item)
						foundMatch = true
						break
					}
				}
			}

			if foundMatch {
				continue
			}

			for term, itemSubtype := range subtypeSearchTerms {
				if strings.HasPrefix(term, rest) {
					if item.GetSpec().Subtype == itemSubtype {
						itemList = append(itemList, item)
						foundMatch = true
						break
					}
				}
			}

			if foundMatch {
				continue
			}

			//
			// Did not find match, search item name for a possible match.
			//
			for _, part := range util.BreakIntoParts(item.Name()) {
				if strings.HasPrefix(part, rest) {
					itemList = append(itemList, item)
					break
				}

			}

		} else {
			itemList = append(itemList, item)
		}

	}

	// Build stack keys and group identical items
	type stackEntry struct {
		name          string
		nameFormatted string
		count         int
	}
	stackOrder := []string{}
	stacks := map[string]*stackEntry{}

	for _, item := range itemList {
		iSpec := item.GetSpec()

		// Stack key: ItemId + enchant state + uses (for consumables)
		stackKey := fmt.Sprintf("%d|%s|%d|%d", item.ItemId, item.EnchantType, item.EnchantTier, item.Uses)

		if entry, exists := stacks[stackKey]; exists {
			entry.count++
			continue
		}

		iName := item.Name()
		iNameFormatted := fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, item.DisplayName())

		if iSpec.Subtype == items.Drinkable || iSpec.Subtype == items.Edible || iSpec.Subtype == items.Usable || iSpec.Type == items.Lockpicks {
			if iSpec.Uses > 0 {
				iName = fmt.Sprintf(`%s (%d)`, iName, item.Uses)
				iNameFormatted = fmt.Sprintf(`%s <ansi fg="uses-left">(%d)</ansi>`, iNameFormatted, item.Uses)
			}
		}

		stacks[stackKey] = &stackEntry{name: iName, nameFormatted: iNameFormatted, count: 1}
		stackOrder = append(stackOrder, stackKey)
	}

	for _, key := range stackOrder {
		entry := stacks[key]
		if entry.count > 1 {
			itemNames = append(itemNames, fmt.Sprintf(`%s (x%d)`, entry.name, entry.count))
			itemNamesFormatted = append(itemNamesFormatted, fmt.Sprintf(`%s <ansi fg="uses-left">(x%d)</ansi>`, entry.nameFormatted, entry.count))
		} else {
			itemNames = append(itemNames, entry.name)
			itemNamesFormatted = append(itemNamesFormatted, entry.nameFormatted)
		}
	}

	raceInfo := species.GetSpecies(user.Character.SpeciesId)

	diceRoll := raceInfo.Damage.DiceRoll
	if user.Character.Equipment.Weapon.ItemId != 0 {
		iSpec := user.Character.Equipment.Weapon.GetSpec()
		diceRoll = iSpec.Damage.DiceRoll
	}

	invData := map[string]any{
		`Equipment`:          &user.Character.Equipment,
		`ItemNames`:          itemNames,
		`ItemNamesFormatted`: itemNamesFormatted,
		`AttackDamage`:       diceRoll,
		`RaceInfo`:           raceInfo,
		`Searching`:          len(rest) > 0,
		`Count`:              fmt.Sprintf(`(%.1f/%.0f lbs)`, user.Character.GetCarriedWeight(), user.Character.CarryCapacity()),
	}

	tplTxt, _ := templates.Process("character/inventory", invData, user.UserId)
	user.SendText(tplTxt)

	return true, nil
}
