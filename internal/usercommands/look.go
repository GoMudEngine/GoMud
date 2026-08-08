package usercommands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mapper"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Look(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	secretLook := flags.Has(events.CmdSecretly)

	visibility := room.GetVisibility()

	if visibility < 1 {
		if !user.Character.HasFlagFromAnySource(buffs.NightVision) {
			user.SendText(messaging.CategorySystem, `You can't see anything!`)
			return true, nil
		}
	}

	isSneaking := user.Character.IsHidden()

	// trim off some fluff
	if len(rest) > 2 {
		if rest[0:3] == `at ` {
			rest = rest[3:]
		}
	}
	if len(rest) > 3 {
		if rest[0:4] == `the ` {
			rest = rest[4:]
		}
	}

	lookAt := rest

	events.AddToQueue(events.Looking{
		UserId: user.UserId,
		RoomId: room.RoomId,
		Target: lookAt,
		Hidden: isSneaking,
	})

	// Handle an ordinary look with no target
	if len(lookAt) == 0 {

		if !secretLook && !isSneaking {
			room.SendTextVisual(messaging.CategoryMobEmote,
				fmt.Sprintf(`<ansi fg="username">%s</ansi> is looking around.`, user.Character.Name),
				user.UserId,
			)

			// Make it a "secret looks" now because we don't want another look message sent out by the lookRoom() func
			secretLook = true
		}
		lookRoom(user, room.RoomId, secretLook || isSneaking)
		return true, nil
	}

	//
	// look for any mobs, players, npcs
	//

	target, err := actions.ResolveTargetActor(room, lookAt)
	if err == nil {

		// Track perception use when examining a target
		user.Character.OnStatUse("perception", user.UserId)

		if target.IsPlayer() {

			u := target.(*actions.UserActor).User

			if !isSneaking {
				u.SendText(messaging.CategoryMobEmote,
					fmt.Sprintf(`<ansi fg="username">%s</ansi> is looking at you.`, user.Character.Name),
				)

				room.SendTextVisual(messaging.CategoryMobEmote,
					fmt.Sprintf(`<ansi fg="username">%s</ansi> is looking at <ansi fg="username">%s</ansi>.`, user.Character.Name, u.Character.Name),
					u.UserId)
			}

			descTxt, _ := templates.Process("character/description", u.Character, user.UserId)
			user.SendText(messaging.CategoryRoomDescription, descTxt)

			itemNames := []string{}
			for _, item := range u.Character.Items {
				itemNames = append(itemNames, item.DisplayName())
			}

			invData := map[string]any{
				`Equipment`: &u.Character.Equipment,
				`ItemNames`: itemNames,
			}

			inventoryTxt, _ := templates.Process("character/inventory-look", invData, user.UserId)
			user.SendText(messaging.CategoryRoomDescription, inventoryTxt)

		} else {

			m := target.(*actions.MobActor).Mob

			if !isSneaking {
				targetName := m.Character.GetMobName(0).String()
				room.SendTextVisual(messaging.CategoryMobEmote,
					fmt.Sprintf(`<ansi fg="username">%s</ansi> is looking at %s.`, user.Character.Name, targetName),
					user.UserId,
				)
			}

			descTxt, _ := templates.Process("character/description", &m.Character, user.UserId)
			user.SendText(messaging.CategoryRoomDescription, descTxt)

			itemNames := []string{}
			for _, item := range m.Character.Items {
				itemNames = append(itemNames, item.DisplayName())
			}

			invData := map[string]any{
				`Equipment`:     &m.Character.Equipment,
				`ItemNames`:     itemNames,
				`HideEquipment`: m.HideEquipmentSlots,
			}

			inventoryTxt, _ := templates.Process("character/inventory-look", invData, user.UserId)
			user.SendText(messaging.CategoryRoomDescription, inventoryTxt)
		}

		return true, nil

	}
	// fall through to container / noun / pet lookup branches below

	if room.MatchesSealedCrate(strings.ToLower(lookAt)) {
		user.SendText(messaging.CategoryRoomDescription, `A heavy iron-banded shipping crate sits at the roadside, its lid`+
			` latched shut and its sides marked with the caravan's burned-in seal.`+
			` The wood is weather-stained and the latch is recently oiled.`)
		return true, nil
	}

	containerName := room.FindContainerByName(lookAt)
	if containerName != `` {
		if container, exists := room.Containers[containerName]; exists && container.Hidden {
			if user == nil || !user.Character.HasDiscovery(room.RoomId, containerName) {
				containerName = `` // Treat as not found
			}
		}
	}
	if containerName != `` {

		itemNames := []string{}
		itemNamesFormatted := []string{}

		container := room.Containers[containerName]

		if container.Lock.IsLocked() {
			user.SendText(messaging.CategorySystem, ``)
			user.SendText(messaging.CategorySystem, fmt.Sprintf(`The <ansi fg="container">%s</ansi> is locked.`, containerName))
			user.SendText(messaging.CategorySystem, ``)
			return true, nil
		}

		if container.Gold > 0 {
			itemNames = append(itemNames, fmt.Sprintf(`%d gold`, container.Gold))
			itemNamesFormatted = append(itemNamesFormatted, fmt.Sprintf(`<ansi fg="gold">%d gold</ansi>`, container.Gold))
		}

		for _, item := range container.Items {
			if !item.IsValid() {
				room.RemoveItem(item, false)
				continue
			}

			itemNames = append(itemNames, item.Name())
			itemNamesFormatted = append(itemNamesFormatted, fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, item.DisplayName()))
		}

		if len(container.Recipes) > 0 {

			user.SendText(messaging.CategorySystem, ``)
			user.SendText(messaging.CategorySystem, fmt.Sprintf(`You can <ansi fg="command">use</ansi> the <ansi fg="container">%s</ansi> if you put the following objects inside:`, containerName))

			for finalItemId, recipeList := range container.Recipes {

				neededItems := map[int]int{}

				for _, inputItemId := range recipeList {
					neededItems[inputItemId] += 1
				}

				user.SendText(messaging.CategorySystem, ``)

				finalItem := items.New(finalItemId)
				user.SendText(messaging.CategorySystem, fmt.Sprintf(`    <ansi fg="230">To receive 1 <ansi fg="itemname">%s</ansi>:</ansi> `, finalItem.DisplayName()))

				for inputItemId, qtyNeeded := range neededItems {
					tmpItem := items.New(inputItemId)
					totalContained := container.Count(inputItemId)
					colorClass := "8" // None fulfilled
					if totalContained == qtyNeeded {
						colorClass = "10"
					} else if totalContained > 0 {
						colorClass = "3"
					}
					user.SendText(messaging.CategorySystem, fmt.Sprintf(`        <ansi fg="%s">[%d/%d]</ansi> <ansi fg="itemname">%s</ansi>`, colorClass, totalContained, qtyNeeded, tmpItem.DisplayName()))
				}

			}

		}

		chestStuff := map[string]any{
			`ItemNames`:          itemNames,
			`ItemNamesFormatted`: itemNamesFormatted,
		}

		textOut, _ := templates.Process("descriptions/insidecontainer", chestStuff, user.UserId)

		user.SendText(messaging.CategoryRoomDescription, ``)
		user.SendText(messaging.CategoryRoomDescription, textOut)
		user.SendText(messaging.CategoryRoomDescription, ``)

		return true, nil
	}

	//
	// Check room exits
	//
	exitName, lookRoomId := room.FindExitByName(lookAt)
	// If nothing found, consider directional aliases
	if exitName == `` {

		if alias := keywords.TryDirectionAlias(lookAt); alias != lookAt {
			exitName, lookRoomId = room.FindExitByName(alias)
			if exitName != `` {
				lookAt = alias
			}
		}
	}

	if exitName != `` {

		if visibility < 2 {

			if !user.Character.HasFlagFromAnySource(buffs.NightVision) {
				biome := room.GetBiome()
				if !biome.IsLit() {
					user.SendText(messaging.CategorySystem, `It's too dark to see anything in that direction.`)
					return true, nil
				}
			}

		}

		exitInfo, _ := room.GetExitInfo(exitName)
		if exitInfo.Lock.IsLocked() {
			user.SendText(messaging.CategorySystem, fmt.Sprintf("The %s exit is locked.", exitName))
			return true, nil
		}

		user.SendText(messaging.CategorySystem, fmt.Sprintf("You peer toward the %s.", exitName))
		if !isSneaking {
			room.SendTextVisual(messaging.CategoryMobEmote, fmt.Sprintf(`<ansi fg="username">%s</ansi> peers toward the %s.`, user.Character.Name, exitName), user.UserId)
		}

		lookRoom(user, lookRoomId, secretLook || isSneaking)

		return true, nil

	}

	// If the input is a recognized direction alias but no exit exists,
	// stop here — never fall through to item/mob matching.
	if alias := keywords.TryDirectionAlias(lookAt); alias != lookAt {
		user.SendText(messaging.CategorySystem, "There is no exit in that direction.")
		return true, nil
	}

	//
	// Check for anything in their backpack they might want to look at
	//
	lookItem, lookDestination, foundItem := user.Character.FindItem(lookAt)

	if foundItem {

		user.SendText(messaging.CategoryRoomDescription, ``)

		user.SendText(messaging.CategoryRoomDescription,
			fmt.Sprintf(`You look at the <ansi fg="item">%s</ansi> %s:`, lookItem.DisplayName(), lookDestination),
		)

		user.SendText(messaging.CategoryRoomDescription, ``)

		if !isSneaking {
			room.SendTextVisual(messaging.CategoryMobEmote,
				fmt.Sprintf(`<ansi fg="username">%s</ansi> is admiring their <ansi fg="item">%s</ansi>.`, user.Character.Name, lookItem.DisplayName()),
				user.UserId,
			)
		}

		user.SendText(messaging.CategoryRoomDescription,
			util.SplitStringNL(lookItem.GetLongDescription(), 80),
		)

		// Show potion aging info based on alchemy skill
		lookSpec := lookItem.GetSpec()
		if lookSpec.Aging.HasAging() && lookItem.CraftedRound > 0 {
			elapsed := util.GetRoundCount() - lookItem.CraftedRound
			bottleMult := lookItem.BottleMultiplier
			if bottleMult <= 0 {
				bottleMult = lookSpec.BottleAgingMultiplier
			}
			effSpeed := items.CalcEffectiveAgingSpeed(bottleMult, lookItem.CraftSkill)
			phase, _ := items.GetAgingPhase(elapsed, lookSpec.Aging, effSpeed)
			alchSkill := user.Character.GetSkillLevel(skills.Alchemy)
			desc := items.GetPhaseDescription(
				phase, alchSkill, elapsed, lookSpec.Aging, effSpeed)
			if desc != "" {
				user.SendText(messaging.CategoryRoomDescription,
					fmt.Sprintf(` - <ansi fg="yellow-bold">%s</ansi>`, desc),
				)
			}
		}

		user.SendText(messaging.CategoryRoomDescription, ``)

		return true, nil
	}

	//
	// Look for any nouns in the room info
	//
	foundNoun, foundDesc := room.FindNoun(lookAt)
	if foundNoun == "" && user != nil {
		// Check discovered hidden nouns
		if key, hn, ok := room.FindHiddenNoun(lookAt); ok {
			if user.Character.HasDiscovery(room.RoomId, key) {
				foundNoun = key
				foundDesc = hn.Description
			}
		}
	}
	if len(foundNoun) > 0 {

		user.SendText(messaging.CategoryRoomDescription, ``)

		user.SendText(messaging.CategoryRoomDescription,
			fmt.Sprintf(`You look at the <ansi fg="noun">%s</ansi>:`, foundNoun),
		)

		user.SendText(messaging.CategoryRoomDescription, ``)

		if !isSneaking {

			// Noun highlighting is universal (2026-06-12). It was formerly
			// gated on the room.nouns role permission or a pet with the
			// SeeNouns buff flag — but the default user role can never hold
			// permissions and no dogmud-world buff carries see-nouns, so the
			// gate made the feature admin-only by accident. Discoverability
			// for everyone beats a vestigial perk.
			renderNouns := true

			if renderNouns && len(room.Nouns) > 0 {
				for noun, _ := range room.Nouns {
					foundDesc = strings.Replace(foundDesc, noun, `<ansi fg="noun">`+noun+`</ansi>`, 1)
				}
			}

			room.SendTextVisual(messaging.CategoryMobEmote,
				fmt.Sprintf(`<ansi fg="username">%s</ansi> is examining the <ansi fg="noun">%s</ansi>.`, user.Character.Name, foundNoun),
				user.UserId,
			)
		}

		user.SendText(messaging.CategoryRoomDescription, util.SplitStringNL(foundDesc, 80))

		user.SendText(messaging.CategoryRoomDescription, ``)

		return true, nil
	}

	//
	// Look for any pets in the room
	//
	petUserId := room.FindByPetName(rest)
	if petUserId == 0 && rest == `pet` && user.Character.Pet.Exists() {
		petUserId = user.UserId
	}
	if petUserId > 0 {
		if petUser := users.GetByUserId(petUserId); petUser != nil {

			user.SendText(messaging.CategoryRoomDescription, fmt.Sprintf(`You look at %s`, petUser.Character.Pet.DisplayName()))

			room.SendTextVisual(messaging.CategoryMobEmote, fmt.Sprintf(`<ansi fg="username">%s</ansi> is looking at %s.`, user.Character.Name, petUser.Character.Pet.DisplayName()), user.UserId)

			textOut, _ := templates.Process("character/pet", petUser, user.UserId)
			user.SendText(messaging.CategoryRoomDescription, textOut)

			return true, nil
		}
	}

	if len(room.Corpses) > 0 {

		mobCorpseLookup := map[string]int{}
		mobCorpses := []string{}

		playerCorpseLookup := map[string]int{}
		playerCorpses := []string{}
		for idx, c := range room.Corpses {
			if c.Prunable {
				continue
			}

			if c.MobId > 0 {
				name := c.DisplayName()
				if _, ok := mobCorpseLookup[name]; !ok {
					mobCorpseLookup[name] = idx
					mobCorpses = append(mobCorpses, name)
				}
			}

			if c.UserId > 0 {
				name := c.Character.Name + ` corpse`
				if _, ok := playerCorpseLookup[name]; !ok {
					playerCorpseLookup[name] = idx
					playerCorpses = append(playerCorpses, name)
				}
			}
		}

		if corpse, corpseFound := room.FindCorpse(rest); corpseFound {

			corpseColor := `mob-corpse`
			if corpse.UserId > 0 {
				corpseColor = `user-corpse`
			}

			user.SendText(messaging.CategoryRoomDescription, fmt.Sprintf(`You look at the <ansi fg="%s">%s</ansi>.`, corpseColor, corpse.DisplayName()))
			room.SendTextVisual(messaging.CategoryMobEmote, fmt.Sprintf(`<ansi fg="username">%s</ansi> is looking at the <ansi fg="%s">%s</ansi>.`, user.Character.Name, corpseColor, corpse.DisplayName()), user.UserId)

			if corpse.CorpseDescription != "" {
				user.SendText(messaging.CategoryRoomDescription, corpse.CorpseDescription)
			} else {
				descTxt, _ := templates.Process("character/description-corpse", &corpse.Character, user.UserId)
				user.SendText(messaging.CategoryRoomDescription, descTxt)
			}

			// Corpse-loot redesign (2026-07-07): show what can be looted,
			// mirroring the room-container listing.
			if corpse.HasLoot() {

				itemNames := []string{}
				itemNamesFormatted := []string{}

				if corpse.Loot.Gold > 0 {
					itemNames = append(itemNames, fmt.Sprintf(`%d gold`, corpse.Loot.Gold))
					itemNamesFormatted = append(itemNamesFormatted, fmt.Sprintf(`<ansi fg="gold">%d gold</ansi>`, corpse.Loot.Gold))
				}

				for _, item := range corpse.Loot.Items {
					if !item.IsValid() {
						continue
					}
					itemNames = append(itemNames, item.Name())
					itemNamesFormatted = append(itemNamesFormatted, fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, item.DisplayName()))
				}

				corpseStuff := map[string]any{
					`ItemNames`:          itemNames,
					`ItemNamesFormatted`: itemNamesFormatted,
				}

				textOut, _ := templates.Process("descriptions/insidecontainer", corpseStuff, user.UserId)
				user.SendText(messaging.CategoryRoomDescription, textOut)
			}

			return true, nil

		}

	}

	// Nothing found
	user.SendText(messaging.CategorySystem, "Look at what???")

	return true, nil

}

func lookRoom(user *users.UserRecord, roomId int, secretLook bool) {

	room := rooms.LoadRoom(roomId)

	if room == nil {
		return
	}

	// Make sure to prepare the room before anyone looks in if this is the first time someone has dealt with it in a while.
	if room.PlayerCt() < 1 {
		room.Prepare(true)
	}

	if !secretLook {
		// Find the exit back
		lookFromName := room.FindExitTo(user.Character.RoomId)
		if lookFromName == "" {
			room.SendTextVisual(messaging.CategoryMobEmote,
				fmt.Sprintf(`<ansi fg="username">%s</ansi> is looking into the room from somewhere...`, user.Character.Name),
				user.UserId,
			)
		} else {
			room.SendTextVisual(messaging.CategoryMobEmote,
				fmt.Sprintf(`<ansi fg="username">%s</ansi> is looking into the room from the <ansi fg="exit">%s</ansi> exit`, user.Character.Name, lookFromName),
				user.UserId,
			)
		}
	}

	var details rooms.RoomTemplateDetails

	tinyMapOn := user.GetConfigOption(`tinymap`)
	if tinyMapOn == nil {
		tinyMapOn = true
	}

	if user.ScreenReader {
		tinyMapOn = false
	}

	// Web-client sessions already have the graphical Map panel, so the inline
	// ASCII minimap beside room descriptions is redundant and only steals
	// horizontal space from the room text. Suppress it automatically for them.
	if connections.IsWebsocket(user.ConnectionId()) {
		tinyMapOn = false
	}

	if tinyMapOn.(bool) && roomId > 0 {

		zMapper := mapper.GetMapper(room.RoomId)
		if zMapper == nil {

			mudlog.Error("Map", "error", "Could not find mapper for zone:"+room.Zone)

			details = rooms.GetDetails(room, user)

		} else {

			c := mapper.Config{
				ZoomLevel: 1,
				Width:     5,
				Height:    5,
				UserId:    user.UserId,
			}

			c.OverrideSymbol(roomId, '@', `You`)

			output := zMapper.GetLimitedMap(room.RoomId, c)
			tinyMap := []string{}
			tinyMap = append(tinyMap, `╔═════╗`)
			for _, mapLine := range output.Render {
				tinyMap = append(tinyMap, `║`+string(mapLine)+`║`)
			}
			tinyMap = append(tinyMap, `╚═════╝`)
			// This additional check is for ephemeral room copies,
			// which can slightly mess with the map render of the @
			if tinyMap[3][3] != '@' {
				youLine := []rune(tinyMap[3])
				youLine[3] = '@'
				tinyMap[3] = string(youLine)
			}

			legend := output.GetLegend(keywords.GetAllLegendAliases(room.Zone))

			for i := 1; i <= c.Height; i++ {
				for sym, txtLegend := range legend {
					txtLc := strings.ToLower(txtLegend)
					tinyMap[i] = strings.Replace(tinyMap[i], string(sym), fmt.Sprintf(`<ansi fg="map-room"><ansi fg="map-%s" bg="mapbg-%s">%c</ansi></ansi>`, txtLc, txtLc, sym), -1)
				}
			}

			details = rooms.GetDetails(room, user, tinyMap)

		}

	} else {
		details = rooms.GetDetails(room, user)
	}

	textOut, _ := templates.Process("descriptions/room-title", details, user.UserId)
	user.SendText(messaging.CategoryRoomDescription, textOut)

	textOut, _ = templates.Process("descriptions/room", details, user.UserId)
	user.SendText(messaging.CategoryRoomDescription, textOut)

	// Append discovered hidden noun descriptions.
	// No user nil check: command dispatch always supplies a live user, and
	// user is already dereferenced above (mapper config, template calls).
	// The old check here was unreachable and made the earlier dereferences
	// look like bugs to static analysis.
	if room.HiddenNouns != nil {
		hiddenKeys := make([]string, 0, len(room.HiddenNouns))
		for k := range room.HiddenNouns {
			hiddenKeys = append(hiddenKeys, k)
		}
		sort.Strings(hiddenKeys)
		for _, key := range hiddenKeys {
			if user.Character.HasDiscovery(room.RoomId, key) {
				hn := room.HiddenNouns[key]
				if hn.HiddenDescription != "" {
					user.SendText(messaging.CategoryRoomDescription, hn.HiddenDescription)
				}
			}
		}
	}

	signCt := 0
	privateSigns := room.GetPrivateSigns()
	for _, sign := range privateSigns {
		if sign.VisibleUserId == user.UserId {
			signCt++
			textOut, _ = templates.Process("descriptions/rune", sign, user.UserId)
			user.SendText(messaging.CategoryRoomDescription, textOut)
		}
	}

	publicSigns := room.GetPublicSigns()
	for _, sign := range publicSigns {
		signCt++
		textOut, _ = templates.Process("descriptions/sign", sign, user.UserId)
		user.SendText(messaging.CategoryRoomDescription, textOut)
	}

	if signCt > 0 {
		user.SendText(messaging.CategoryRoomDescription, "")
	}

	textOut, _ = templates.Process("descriptions/who", details, user.UserId)
	if len(textOut) > 0 {
		user.SendText(messaging.CategoryRoomDescription, textOut)
	}

	groundStuff := []string{}
	for containerName, container := range room.Containers {

		// Skip hidden containers the user hasn't discovered
		if container.Hidden {
			if user == nil || !user.Character.HasDiscovery(room.RoomId, containerName) {
				continue
			}
		}

		chestName := fmt.Sprintf(`<ansi fg="container">%s</ansi>`, containerName)

		if container.HasLock() {
			if container.Lock.IsLocked() {
				chestName += ` <ansi fg="white">(locked)</ansi>`
			} else {
				chestName += ` <ansi fg="white">(unlocked)</ansi>`
			}
		}

		groundStuff = append(groundStuff, chestName)

	}

	if room.Gold > 0 {
		groundStuff = append(groundStuff, fmt.Sprintf(`<ansi fg="gold">%d gold</ansi>`, room.Gold))
	}

	// Stack identical floor items for display
	type groundStack struct {
		name  string
		count int
	}
	groundStackOrder := []string{}
	groundStacks := map[string]*groundStack{}

	for _, item := range room.Items {
		if !item.IsValid() {
			room.RemoveItem(item, false)
			continue
		}
		key := fmt.Sprintf("%d|%s|%d", item.ItemId, item.EnchantType, item.EnchantTier)
		if entry, exists := groundStacks[key]; exists {
			entry.count++
		} else {
			groundStacks[key] = &groundStack{name: item.DisplayName(), count: 1}
			groundStackOrder = append(groundStackOrder, key)
		}
	}
	for _, key := range groundStackOrder {
		entry := groundStacks[key]
		if entry.count > 1 {
			groundStuff = append(groundStuff, fmt.Sprintf(`%s <ansi fg="uses-left">(x%d)</ansi>`, entry.name, entry.count))
		} else {
			groundStuff = append(groundStuff, entry.name)
		}
	}

	// Find stashed items
	for _, item := range room.Stash {
		if !item.IsValid() {
			room.RemoveItem(item, true)
		}
		if item.StashedBy != user.UserId {
			continue
		}
		name := item.DisplayName() + ` <ansi fg="item-stashed">(stashed)</ansi>`
		groundStuff = append(groundStuff, name)
	}

	groundStuff = append(groundStuff, details.VisibleCorpses...)

	groundDetails := map[string]any{
		`GroundStuff`: groundStuff,
		`IsDark`:      room.GetBiome().IsDark(),
		`IsNight`:     gametime.IsNight(),
	}
	textOut, _ = templates.Process("descriptions/ontheground", groundDetails, user.UserId)
	if len(textOut) > 0 {
		user.SendText(messaging.CategoryRoomDescription, textOut)
	}

	textOut, _ = templates.Process("descriptions/exits", details, user.UserId)
	user.SendText(messaging.CategoryRoomDescription, textOut)

}
