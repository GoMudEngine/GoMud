package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// checkSpoiledGrenades scans the player's backpack for spoiled throwable items.
// Each spoiled grenade has a 35% chance to detonate (applying its effect to the
// holder) or fizzle into putrid residue. Grenades in the bandolier are safe —
// they get ejected to the room floor by the regen tick instead.
func checkSpoiledGrenades(user *users.UserRecord, room *rooms.Room) {
	currentRound := util.GetRoundCount()
	detonated := 0
	fizzled := 0
	remaining := make([]items.Item, 0, len(user.Character.Items))

	for _, item := range user.Character.Items {
		spec := item.GetSpec()
		if spec.Subtype != items.Throwable || !spec.Aging.HasAging() || item.CraftedRound == 0 {
			remaining = append(remaining, item)
			continue
		}

		var elapsed uint64
		if currentRound >= item.CraftedRound {
			elapsed = currentRound - item.CraftedRound
		}
		effSpeed := items.CalcEffectiveAgingSpeed(item.BottleMultiplier, item.CraftSkill)
		phase, _ := items.GetAgingPhase(elapsed, spec.Aging, effSpeed)

		if phase != items.PhaseSpoiled {
			remaining = append(remaining, item)
			continue
		}

		// Spoiled grenade — 35% chance to detonate
		if util.Rand(100) < 35 {
			detonated++
			// Apply grenade effect to the holder
			if spec.DamageMultiplier > 0 {
				// Firebomb: self-damage
				rawDmg := combat.CalcRawDamage(
					user.Character.Stats.Dexterity.ValueAdj,
					user.Character.GetSkillLevel(skills.Skullduggery),
					spec.DamageMultiplier,
					combat.ChannelPhysical,
				)
				dmg := int(rawDmg)
				if dmg < 1 {
					dmg = 1
				}
				user.Character.Health -= dmg
				user.SendText(fmt.Sprintf(
					`<ansi fg="red-bold">A <ansi fg="itemname">%s</ansi> in your pack detonates! (%s)</ansi>`,
					item.DisplayName(),
					combat.GetDamageDescription(dmg, user.Character.HealthMax.Value)))
			} else {
				// Debuff grenade: apply buffs to self
				for _, buffId := range spec.BuffIds {
					user.AddBuff(buffId, "grenade-accident")
				}
				user.SendText(fmt.Sprintf(
					`<ansi fg="red-bold">A <ansi fg="itemname">%s</ansi> in your pack goes off!</ansi>`,
					item.DisplayName()))
			}
		} else {
			// Fizzle into putrid residue
			fizzled++
			residue := items.New(40050)
			if residue.ItemId > 0 {
				remaining = append(remaining, residue)
			}
		}
	}

	if detonated > 0 || fizzled > 0 {
		user.Character.Items = remaining
		if fizzled > 0 {
			user.SendText(fmt.Sprintf(
				`<ansi fg="yellow">%d grenade(s) have destabilized and dissolved into putrid residue.</ansi>`,
				fizzled))
		}
		if room != nil {
			room.SendTextVisual(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> fumbles with something in their pack...`,
					user.Character.Name),
				user.UserId)
		}
		events.AddToQueue(events.RedrawPrompt{UserId: user.UserId, OnlyIfChanged: true}, 100)
	}
}

// encumbranceTier returns a descriptive label and ANSI color for the
// player's current weight-to-capacity ratio.
func encumbranceTier(weight, capacity float64) (label, color string) {
	if capacity <= 0 {
		return "crushed", "magenta-bold"
	}
	ratio := weight / capacity
	switch {
	case ratio <= 0.25:
		return "light", "green"
	case ratio <= 0.50:
		return "moderate", "yellow"
	case ratio <= 0.75:
		return "heavy", "red"
	case ratio <= 1.00:
		return "overburdened", "red-bold"
	default:
		return "crushed", "magenta-bold"
	}
}

func Inventory(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Check for spoiled grenades in backpack — risk of detonation
	checkSpoiledGrenades(user, room)

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

	// Build stack keys and group identical items.
	// Two items share a display row iff items.SameStack returns true AND
	// they share the same spoiled status (so spoiled potions don't merge
	// with fresh ones).
	type stackEntry struct {
		name           string
		nameFormatted  string
		count          int
		representative items.Item // first item seen for this stack key
		spoiled        bool
	}
	stackOrder := []string{}
	stacks := map[string]*stackEntry{}

	alchSkill := user.Character.GetSkillLevel(skills.Alchemy)

	isSpoiledItem := func(itm items.Item) bool {
		iSpec := itm.GetSpec()
		if alchSkill < 6 || !iSpec.Aging.HasAging() || itm.CraftedRound == 0 {
			return false
		}
		elapsed := util.GetRoundCount() - itm.CraftedRound
		bMult := itm.BottleMultiplier
		if bMult <= 0 {
			bMult = iSpec.BottleAgingMultiplier
		}
		effSpeed := items.CalcEffectiveAgingSpeed(bMult, itm.CraftSkill)
		phase, _ := items.GetAgingPhase(elapsed, iSpec.Aging, effSpeed)
		return phase == items.PhaseSpoiled
	}

	for _, item := range itemList {
		iSpec := item.GetSpec()
		isSpoiled := isSpoiledItem(item)

		// Look for an existing stack that this item can join.
		// We walk stackOrder to maintain insertion order, and check
		// items.SameStack + spoiled-flag agreement.
		merged := false
		for _, key := range stackOrder {
			entry := stacks[key]
			if entry.spoiled == isSpoiled && items.SameStack(entry.representative, item) {
				entry.count++
				merged = true
				break
			}
		}
		if merged {
			continue
		}

		// New stack entry.
		stackKey := fmt.Sprintf("%d|%d|%d|%v", item.ItemId, item.Uses, item.EnchantTier, isSpoiled)
		// Disambiguate duplicate keys from distinct items (e.g. different
		// BottleMultiplier values with the same ItemId/Uses/EnchantTier).
		for {
			if _, exists := stacks[stackKey]; !exists {
				break
			}
			stackKey += "|x"
		}

		// Use base name for stacked display — enchant adjectives are
		// revealed via look/identify, not the inventory list.
		iName := item.Name()
		iNameFormatted := fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, item.Name())

		if isSpoiled {
			iName = fmt.Sprintf(`%s (turned)`, item.Name())
			iNameFormatted = fmt.Sprintf(`<ansi fg="8">%s (turned)</ansi>`, item.Name())
		} else if iSpec.Subtype == items.Drinkable || iSpec.Subtype == items.Edible || iSpec.Subtype == items.Usable || iSpec.Type == items.Lockpicks {
			if iSpec.Uses > 0 {
				iName = fmt.Sprintf(`%s (%d)`, iName, item.Uses)
				iNameFormatted = fmt.Sprintf(`%s <ansi fg="uses-left">(%d)</ansi>`, iNameFormatted, item.Uses)
			}
		}

		stacks[stackKey] = &stackEntry{
			name:           iName,
			nameFormatted:  iNameFormatted,
			count:          1,
			representative: item,
			spoiled:        isSpoiled,
		}
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

	// Build stacked list of component bag items
	componentNames := []string{}
	componentNamesFormatted := []string{}

	compStackOrder := []string{}
	compStacks := map[string]*stackEntry{}

	for _, item := range user.Character.ComponentItems {
		iSpec := item.GetSpec()

		// Find an existing matching stack using SameStack.
		merged := false
		for _, key := range compStackOrder {
			entry := compStacks[key]
			if items.SameStack(entry.representative, item) {
				entry.count++
				merged = true
				break
			}
		}
		if merged {
			continue
		}

		stackKey := fmt.Sprintf("%d|%d|%d|%d", item.ItemId, item.Uses, item.EnchantTier, len(compStackOrder))

		iName := item.Name()
		iNameFormatted := fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, item.DisplayName())

		if iSpec.Subtype == items.Drinkable || iSpec.Subtype == items.Edible || iSpec.Subtype == items.Usable || iSpec.Type == items.Lockpicks {
			if iSpec.Uses > 0 {
				iName = fmt.Sprintf(`%s (%d)`, iName, item.Uses)
				iNameFormatted = fmt.Sprintf(`%s <ansi fg="uses-left">(%d)</ansi>`, iNameFormatted, item.Uses)
			}
		}

		compStacks[stackKey] = &stackEntry{
			name:           iName,
			nameFormatted:  iNameFormatted,
			count:          1,
			representative: item,
		}
		compStackOrder = append(compStackOrder, stackKey)
	}

	for _, key := range compStackOrder {
		entry := compStacks[key]
		if entry.count > 1 {
			componentNames = append(componentNames, fmt.Sprintf(`%s (x%d)`, entry.name, entry.count))
			componentNamesFormatted = append(componentNamesFormatted, fmt.Sprintf(`%s <ansi fg="uses-left">(x%d)</ansi>`, entry.nameFormatted, entry.count))
		} else {
			componentNames = append(componentNames, entry.name)
			componentNamesFormatted = append(componentNamesFormatted, entry.nameFormatted)
		}
	}

	// Build stacked list of bandolier potion items
	potionNames := []string{}
	potionNamesFormatted := []string{}

	potStackOrder := []string{}
	potStacks := map[string]*stackEntry{}

	for _, item := range user.Character.PotionItems {
		isSpoiled := isSpoiledItem(item)

		// Find an existing matching stack using SameStack + spoiled agreement.
		merged := false
		for _, key := range potStackOrder {
			entry := potStacks[key]
			if entry.spoiled == isSpoiled && items.SameStack(entry.representative, item) {
				entry.count++
				merged = true
				break
			}
		}
		if merged {
			continue
		}

		stackKey := fmt.Sprintf("%d|%d|%d|%v|%d", item.ItemId, item.Uses, item.EnchantTier, isSpoiled, len(potStackOrder))

		iName := item.Name()
		iNameFormatted := fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, item.DisplayName())

		if isSpoiled {
			iName = fmt.Sprintf(`%s (turned)`, item.Name())
			iNameFormatted = fmt.Sprintf(`<ansi fg="8">%s (turned)</ansi>`, item.DisplayName())
		}

		potStacks[stackKey] = &stackEntry{
			name:           iName,
			nameFormatted:  iNameFormatted,
			count:          1,
			representative: item,
			spoiled:        isSpoiled,
		}
		potStackOrder = append(potStackOrder, stackKey)
	}

	for _, key := range potStackOrder {
		entry := potStacks[key]
		if entry.count > 1 {
			potionNames = append(potionNames, fmt.Sprintf(`%s (x%d)`, entry.name, entry.count))
			potionNamesFormatted = append(potionNamesFormatted, fmt.Sprintf(`%s <ansi fg="uses-left">(x%d)</ansi>`, entry.nameFormatted, entry.count))
		} else {
			potionNames = append(potionNames, entry.name)
			potionNamesFormatted = append(potionNamesFormatted, entry.nameFormatted)
		}
	}

	raceInfo := species.GetSpecies(user.Character.SpeciesId)

	diceRoll := raceInfo.Damage.DiceRoll
	if user.Character.Equipment.Weapon.ItemId != 0 {
		iSpec := user.Character.Equipment.Weapon.GetSpec()
		diceRoll = iSpec.Damage.DiceRoll
	}

	encLabel, encColor := encumbranceTier(user.Character.GetCarriedWeight(), user.Character.CarryCapacity())

	invData := map[string]any{
		`Equipment`:               &user.Character.Equipment,
		`ItemNames`:               itemNames,
		`ItemNamesFormatted`:      itemNamesFormatted,
		`ComponentNames`:          componentNames,
		`ComponentNamesFormatted`: componentNamesFormatted,
		`PotionNames`:             potionNames,
		`PotionNamesFormatted`:    potionNamesFormatted,
		`AttackDamage`:            diceRoll,
		`RaceInfo`:                raceInfo,
		`Searching`:               len(rest) > 0,
		`EncumbranceLabel`:        encLabel,
		`EncumbranceColor`:        encColor,
	}

	tplTxt, _ := templates.Process("character/inventory", invData, user.UserId)
	user.SendText(tplTxt)

	return true, nil
}
