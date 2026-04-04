package usercommands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Craft handles the `craft` and `craft list` commands (Stage 13.1).
func Craft(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	rest = strings.TrimSpace(rest)

	// ── craft / craft list ────────────────────────────────────────────────────
	if rest == "" || strings.ToLower(rest) == "list" {
		return craftList(user, room), nil
	}

	// ── Enchanting: needs player-specific disambiguation before delegating ─────
	// Peek at the recipe first to route enchanting separately.
	recipe := crafting.FindRecipeByName(rest)
	if recipe != nil && crafting.IsEnchantingRecipe(recipe) {
		return craftEnchanting(rest, recipe, user, room)
	}

	// ── Normal craft path: delegate to shared action ──────────────────────────
	actor := &actions.UserActor{User: user, Room: room}
	result := actions.InitiateCraft(actor, rest)

	switch {
	case result.RecipeNotFound:
		user.SendText(fmt.Sprintf(
			`<ansi fg="red">No recipe found for "%s". Type <ansi fg="cyan-bold">craft list</ansi> to see available recipes.</ansi>`,
			rest))
		return true, nil

	case result.RecipeNotKnown:
		user.SendText(`<ansi fg="red">You don't know that recipe yet. Keep crafting to discover new ones!</ansi>`)
		return true, nil

	case result.AlreadyCrafting:
		user.SendText(`<ansi fg="red">You are already working on something. Finish or be interrupted first.</ansi>`)
		return true, nil

	case result.SkillTooLow:
		user.SendText(fmt.Sprintf(
			`<ansi fg="red">Your %s skill is too low (requires %d, you have %d).</ansi>`,
			result.SkillName, result.SkillMinimum, result.SkillLevel))
		return true, nil

	case result.WrongStation:
		user.SendText(fmt.Sprintf(
			`<ansi fg="red">You need to be at a %s to craft that.</ansi>`,
			result.StationNeeded))
		return true, nil

	case result.MissingIngredients:
		user.SendText(fmt.Sprintf(`<ansi fg="red">You are missing: %s.</ansi>`, result.MissingTag))
		return true, nil

	case result.ImmediateComplete:
		user.SendText(fmt.Sprintf(`<ansi fg="green">%s</ansi>`, result.SuccessMsg))
		return true, nil

	case result.Initiated:
		user.SendText(fmt.Sprintf(
			`<ansi fg="yellow">You begin crafting %s... (%s)</ansi>`,
			result.RecipeName, craftTimeDesc(result.TimeRounds)))

		// Quest engine: command notification
		bridge := questengine.NewGameBridge(user, room.RoomId)
		questengine.GetEngine().Notify("command", questengine.EventDetails{
			UserId:  user.UserId,
			RoomId:  room.RoomId,
			Command: "craft",
		}, bridge, bridge)
		return true, nil
	}

	return true, nil
}

// craftEnchanting handles the enchanting sub-path of craft, which requires
// player-specific target disambiguation not available to mob actors.
func craftEnchanting(rest string, recipe *crafting.RecipeSpec, user *users.UserRecord, room *rooms.Room) (bool, error) {
	// Known-recipe gate
	if !user.Character.HasRecipe(recipe.RecipeId) {
		user.SendText(`<ansi fg="red">You don't know that recipe yet. Keep crafting to discover new ones!</ansi>`)
		return true, nil
	}

	// Already crafting?
	if user.Character.IsCrafting() {
		user.SendText(`<ansi fg="red">You are already working on something. Finish or be interrupted first.</ansi>`)
		return true, nil
	}

	// Skill gate
	skillLevel := user.Character.GetSkillLevel(skills.SkillTag(recipe.Skill))
	if skillLevel < recipe.SkillMinimum {
		user.SendText(fmt.Sprintf(
			`<ansi fg="red">Your %s skill is too low (requires %d, you have %d).</ansi>`,
			recipe.Skill, recipe.SkillMinimum, skillLevel))
		return true, nil
	}

	// Station check
	if recipe.Station != "" && room.Station != recipe.Station {
		user.SendText(fmt.Sprintf(
			`<ansi fg="red">You need to be at a %s to craft that.</ansi>`,
			strings.ReplaceAll(recipe.Station, "_", " ")))
		return true, nil
	}

	// Ingredient check
	ok, missing := crafting.HasIngredients(user.Character.Items, user.Character.ComponentItems, recipe)
	if !ok {
		user.SendText(fmt.Sprintf(`<ansi fg="red">You are missing: %s.</ansi>`, missing))
		return true, nil
	}

	// Target item disambiguation — strip the recipe name from the input
	// to get the optional item specifier. The recipe name may contain spaces
	// (e.g., "carapace ward"), so we can't just split on the first space.
	specifier := ""
	recipeName := strings.ToLower(recipe.Name)
	restLower := strings.ToLower(strings.TrimSpace(rest))
	if strings.HasPrefix(restLower, recipeName) {
		specifier = strings.TrimSpace(rest[len(recipeName):])
	} else if strings.HasPrefix(restLower, strings.ToLower(recipe.RecipeId)) {
		specifier = strings.TrimSpace(rest[len(recipe.RecipeId):])
	}

	eq := user.Character.Equipment
	equipSlots := []crafting.EquipmentSlot{
		{Item: eq.Weapon, Label: "wielded"},
		{Item: eq.Offhand, Label: "offhand"},
		{Item: eq.ExtraArm1, Label: "extra arm 1"},
		{Item: eq.ExtraArm2, Label: "extra arm 2"},
		{Item: eq.ExtraArm3, Label: "extra arm 3"},
		{Item: eq.ExtraArm4, Label: "extra arm 4"},
		{Item: eq.Head, Label: "worn - head"},
		{Item: eq.Neck, Label: "worn - neck"},
		{Item: eq.Shoulders, Label: "worn - shoulders"},
		{Item: eq.Body, Label: "worn - body"},
		{Item: eq.Back, Label: "worn - back"},
		{Item: eq.Belt, Label: "worn - belt"},
		{Item: eq.Wrist1, Label: "worn - wrist"},
		{Item: eq.Wrist2, Label: "worn - wrist"},
		{Item: eq.ExtraWrist1, Label: "extra wrist 1"},
		{Item: eq.ExtraWrist2, Label: "extra wrist 2"},
		{Item: eq.ExtraWrist3, Label: "extra wrist 3"},
		{Item: eq.ExtraWrist4, Label: "extra wrist 4"},
		{Item: eq.Gloves, Label: "worn - gloves"},
		{Item: eq.Ring, Label: "worn - ring"},
		{Item: eq.Ring2, Label: "worn - ring"},
		{Item: eq.Legs, Label: "worn - legs"},
		{Item: eq.Feet, Label: "worn - feet"},
	}
	candidates := crafting.FindTargetItems(user.Character.Items, equipSlots, recipe.TargetType, specifier)

	if len(candidates) == 0 {
		user.SendText(fmt.Sprintf(
			`<ansi fg="red">You need a %s in your inventory or equipment to enchant.</ansi>`,
			strings.ReplaceAll(recipe.TargetType, "_", " ")))
		return true, nil
	}

	if len(candidates) > 1 && specifier == "" {
		user.SendText(`<ansi fg="yellow">Which item do you want to enchant?</ansi>`)
		for i, c := range candidates {
			slotInfo := ""
			if c.SourceLabel != "" {
				slotInfo = fmt.Sprintf(` <ansi fg="yellow">(%s)</ansi>`, c.SourceLabel)
			}
			user.SendText(fmt.Sprintf(`  <ansi fg="white-bold">[%d]</ansi> <ansi fg="itemname">%s</ansi>%s`, i+1, c.Item.DisplayName(), slotInfo))
		}
		user.SendText(`  <ansi fg="white-bold">[0]</ansi> Cancel`)
		user.SendText(`<ansi fg="yellow">Specify which item: craft ` + recipe.Name + ` <item name></ansi>`)
		return true, nil
	}

	// Safety: complete immediately if time_rounds <= 0
	if recipe.TimeRounds <= 0 {
		completeCraft(user, recipe)
		return true, nil
	}

	// Start multi-round enchanting
	user.Character.CraftingState = &characters.CraftingState{
		RecipeId:    recipe.RecipeId,
		RoundsTotal: recipe.TimeRounds,
	}
	user.SendText(fmt.Sprintf(
		`<ansi fg="yellow">You begin crafting %s... (%s)</ansi>`,
		recipe.Name, craftTimeDesc(recipe.TimeRounds)))

	// Quest engine: command notification
	bridge := questengine.NewGameBridge(user, room.RoomId)
	questengine.GetEngine().Notify("command", questengine.EventDetails{
		UserId:  user.UserId,
		RoomId:  room.RoomId,
		Command: "craft",
	}, bridge, bridge)

	return true, nil
}

// craftList prints all known recipes grouped by skill with craftability indicators.
func craftList(user *users.UserRecord, room *rooms.Room) bool {
	all := crafting.GetAll()
	if len(all) == 0 {
		user.SendText(`<ansi fg="yellow">No crafting recipes are currently available.</ansi>`)
		return true
	}

	// Filter to only known recipes
	known := make(map[string]*crafting.RecipeSpec)
	for id, r := range all {
		if user.Character.HasRecipe(id) {
			known[id] = r
		}
	}

	if len(known) == 0 {
		user.SendText(`<ansi fg="yellow">You haven't discovered any crafting recipes yet.</ansi>`)
		return true
	}

	// Collect unique skill names sorted alphabetically
	skillSet := make(map[string]struct{})
	for _, r := range known {
		skillSet[r.Skill] = struct{}{}
	}
	skillNames := make([]string, 0, len(skillSet))
	for sk := range skillSet {
		skillNames = append(skillNames, sk)
	}
	sort.Strings(skillNames)

	user.SendText(``)
	user.SendText(`<ansi fg="cyan-bold"> .:. Crafting Recipes .:.</ansi>`)

	totalKnown := 0
	grandTotal := 0

	for _, skillName := range skillNames {
		skillLevel := user.Character.GetSkillLevel(skills.SkillTag(skillName))

		// Count known vs total for this skill
		allForSkill := crafting.GetAllForSkill(skillName)
		knownCount := 0
		for _, r := range allForSkill {
			if user.Character.HasRecipe(r.RecipeId) {
				knownCount++
			}
		}
		totalKnown += knownCount
		grandTotal += len(allForSkill)

		completionDesc := recipeCompletionTier(knownCount, len(allForSkill))

		user.SendText(``)
		user.SendText(fmt.Sprintf(
			`<ansi fg="yellow">%s</ansi> <ansi fg="white">(%s)</ansi> — <ansi fg="black-bold">%s</ansi>`,
			titleCase(strings.ReplaceAll(skillName, "-", " ")), skills.GetSkillRankDescription(skillLevel), completionDesc))

		recipes := crafting.GetAllForSkill(skillName)
		for _, r := range recipes {
			if !user.Character.HasRecipe(r.RecipeId) {
				continue
			}
			indicator, reason := recipeStatus(user, room, r, skillLevel)
			ingredientList := ingredientSummary(r)
			stationStr := ""
			if r.Station != "" {
				stationStr = fmt.Sprintf(" [%s]", strings.ReplaceAll(r.Station, "_", " "))
			}
			if reason != "" {
				user.SendText(fmt.Sprintf(
					`  <ansi fg="red">[%s]</ansi> <ansi fg="white">%-22s</ansi> — %s  <ansi fg="red">%s</ansi><ansi fg="dark-cyan">%s, %s</ansi>`,
					indicator, r.Name, ingredientList, reason, stationStr, craftTimeDesc(r.TimeRounds)))
			} else {
				user.SendText(fmt.Sprintf(
					`  <ansi fg="green">[%s]</ansi> <ansi fg="white">%-22s</ansi> — %s  <ansi fg="dark-cyan">%s, %s</ansi>`,
					indicator, r.Name, ingredientList, stationStr, craftTimeDesc(r.TimeRounds)))
			}
		}
	}

	// Overall completion
	overallDesc := recipeCompletionTier(totalKnown, grandTotal)
	user.SendText(``)
	user.SendText(fmt.Sprintf(`<ansi fg="cyan">Overall recipe knowledge: %s</ansi>`, overallDesc))
	user.SendText(``)
	return true
}

// recipeCompletionTier returns a descriptive tier for how many recipes
// are known out of a total. No hard numbers shown to the player.
func recipeCompletionTier(known, total int) string {
	if total <= 0 {
		return "unknown"
	}
	pct := float64(known) / float64(total) * 100
	switch {
	case pct < 15:
		return "a handful of recipes"
	case pct < 35:
		return "a modest collection"
	case pct < 60:
		return "a solid repertoire"
	case pct < 85:
		return "an extensive catalog"
	default:
		return "near-complete mastery"
	}
}

// recipeStatus returns the indicator character and a blocking reason string.
// indicator is "✓" if craftable, "✗" otherwise. reason is "" if craftable.
func recipeStatus(user *users.UserRecord, room *rooms.Room, r *crafting.RecipeSpec, skillLevel int) (string, string) {
	if skillLevel < r.SkillMinimum {
		return "X", fmt.Sprintf("%s skill required", skills.GetSkillRankDescription(r.SkillMinimum))
	}
	if r.Station != "" && room.Station != r.Station {
		return "X", fmt.Sprintf("need %s", strings.ReplaceAll(r.Station, "_", " "))
	}
	ok, missing := crafting.HasIngredients(user.Character.Items, user.Character.ComponentItems, r)
	if !ok {
		return "X", fmt.Sprintf("missing %s", missing)
	}
	return "V", ""
}

// ingredientSummary returns a short comma-separated ingredient list.
func ingredientSummary(r *crafting.RecipeSpec) string {
	parts := make([]string, 0, len(r.Ingredients))
	for _, ing := range r.Ingredients {
		parts = append(parts, fmt.Sprintf("%dx %s", ing.Quantity, ing.ItemTag))
	}
	return strings.Join(parts, ", ")
}

// completeCraft resolves a craft instantly (used when time_rounds <= 0).
func completeCraft(user *users.UserRecord, recipe *crafting.RecipeSpec) {
	user.Character.Items, user.Character.ComponentItems = crafting.ConsumeIngredients(user.Character.Items, user.Character.ComponentItems, recipe)
	newItem := items.New(recipe.Output.ItemId)
	user.Character.StoreItem(newItem)
	user.SendText(fmt.Sprintf(`<ansi fg="green">%s</ansi>`, recipe.SuccessMessage))
}

// titleCase capitalises the first letter of each space-separated word.
func titleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// craftTimeDesc returns a qualitative description for crafting duration.
func craftTimeDesc(rounds int) string {
	switch {
	case rounds <= 1:
		return "instant"
	case rounds <= 3:
		return "quick"
	case rounds <= 6:
		return "moderate"
	case rounds <= 10:
		return "lengthy"
	default:
		return "prolonged"
	}
}
