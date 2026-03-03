package usercommands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
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

	// ── craft <name> ──────────────────────────────────────────────────────────
	recipe := crafting.FindRecipeByName(rest)
	if recipe == nil {
		user.SendText(fmt.Sprintf(`<ansi fg="red">No recipe found for "%s". Type <ansi fg="cyan-bold">craft list</ansi> to see available recipes.</ansi>`, rest))
		return true, nil
	}

	// Known-recipe gate (Stage 31.1)
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
	ok, missing := crafting.HasIngredients(user.Character.Items, recipe)
	if !ok {
		user.SendText(fmt.Sprintf(`<ansi fg="red">You are missing: %s.</ansi>`, missing))
		return true, nil
	}

	// Enchanting: target item check
	if crafting.IsEnchantingRecipe(recipe) {
		_, found := crafting.FindTargetItem(user.Character.Items, recipe.TargetType)
		if !found {
			user.SendText(fmt.Sprintf(
				`<ansi fg="red">You need a %s in your inventory to enchant.</ansi>`,
				strings.ReplaceAll(recipe.TargetType, "_", " ")))
			return true, nil
		}
	}

	// Safety: complete immediately if time_rounds <= 0
	if recipe.TimeRounds <= 0 {
		completeCraft(user, recipe)
		return true, nil
	}

	// Start crafting
	user.Character.CraftingState = &characters.CraftingState{
		RecipeId:    recipe.RecipeId,
		RoundsTotal: recipe.TimeRounds,
	}
	user.SendText(fmt.Sprintf(
		`<ansi fg="yellow">You begin crafting %s... (%s)</ansi>`,
		recipe.Name, craftTimeDesc(recipe.TimeRounds)))

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

	for _, skillName := range skillNames {
		skillLevel := user.Character.GetSkillLevel(skills.SkillTag(skillName))
		user.SendText(``)
		user.SendText(fmt.Sprintf(
			`<ansi fg="yellow">%s</ansi> <ansi fg="white">(%s)</ansi>`,
			titleCase(strings.ReplaceAll(skillName, "-", " ")), skills.GetSkillRankDescription(skillLevel)))

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

	user.SendText(``)
	return true
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
	ok, missing := crafting.HasIngredients(user.Character.Items, r)
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
	user.Character.Items = crafting.ConsumeIngredients(user.Character.Items, recipe)
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
