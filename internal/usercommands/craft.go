package usercommands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Craft handles the `craft` and `craft list` commands (Stage 13.1).
func Craft(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	rest = strings.TrimSpace(rest)

	// ── craft (bare) → only recipes craftable right now, here ─────────────────
	if rest == "" {
		return craftCraftableNow(user, room), nil
	}
	// ── craft list / craft all → full sectioned view ──────────────────────────
	if lc := strings.ToLower(rest); lc == "list" || lc == "all" {
		return craftList(user, room), nil
	}

	// ── Enchanting: needs player-specific disambiguation before delegating ─────
	// Peek at the recipe first to route enchanting separately.
	// Input may be "recipe-name item-name" so try progressively shorter
	// prefixes: "honed-edge knuckles" → try "honed-edge knuckles", then
	// "honed-edge". This handles hyphenated recipe names with an item target.
	recipe := crafting.FindRecipeByName(rest)
	if recipe == nil {
		words := strings.Fields(rest)
		for i := len(words) - 1; i >= 1; i-- {
			candidate := strings.Join(words[:i], " ")
			if r := crafting.FindRecipeByName(candidate); r != nil {
				recipe = r
				break
			}
		}
	}
	if recipe != nil && crafting.IsEnchantingRecipe(recipe) {
		return craftEnchanting(rest, recipe, user, room)
	}

	// Auto-pull from storage: at a storage room that satisfies the recipe's station,
	// draw exactly the missing components from the player's storage so the craft
	// can proceed. Fires wherever you can craft (at the station); your storage is
	// one per-character pool, reachable while crafting. All-or-nothing
	// (PlanStoragePull returns complete=false if storage can't cover it).
	if recipe != nil &&
		(recipe.Station == "" || room.Station == recipe.Station) &&
		user.Character.HasRecipe(recipe.RecipeId) {
		if ok, _ := crafting.HasIngredients(user.Character.Items, user.Character.ComponentItems, recipe); !ok {
			if pull, complete := crafting.PlanStoragePull(recipe, user.Character.Items, user.Character.ComponentItems, user.ItemStorage.GetItems()); complete {
				for _, itm := range pull {
					if user.ItemStorage.RemoveItem(itm) {
						user.Character.StoreItem(itm)
						user.SendText(messaging.CategoryLoot, fmt.Sprintf(`You draw <ansi fg="item">%s</ansi> from storage.`, itm.DisplayName()))
					}
				}
			}
		}
	}

	// ── Normal craft path: delegate to shared action ──────────────────────────
	actor := &actions.UserActor{User: user, Room: room}
	result := actions.InitiateCraft(actor, rest)

	switch {
	case result.RecipeNotFound:
		user.SendText(messaging.CategorySystem, fmt.Sprintf(
			`<ansi fg="red">No recipe found for "%s". Type <ansi fg="cyan-bold">craft list</ansi> to see available recipes.</ansi>`,
			rest))
		return true, nil

	case result.RecipeNotKnown:
		user.SendText(messaging.CategorySystem, `<ansi fg="red">You don't know that recipe yet. Keep crafting to discover new ones!</ansi>`)
		return true, nil

	case result.AlreadyCrafting:
		user.SendText(messaging.CategorySystem, `<ansi fg="red">You are already working on something. Finish or be interrupted first.</ansi>`)
		return true, nil

	case result.SkillTooLow:
		user.SendText(messaging.CategorySystem, fmt.Sprintf(
			`<ansi fg="red">Your %s skill is too low (requires %d, you have %d).</ansi>`,
			result.SkillName, result.SkillMinimum, result.SkillLevel))
		return true, nil

	case result.WrongStation:
		user.SendText(messaging.CategorySystem, fmt.Sprintf(
			`<ansi fg="red">You need to be at a %s to craft that.</ansi>`,
			result.StationNeeded))
		return true, nil

	case result.MissingIngredients:
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="red">You are missing: %s.</ansi>`, result.MissingTag))
		return true, nil

	case result.ForeignComponent:
		user.SendText(messaging.CategorySystem, fmt.Sprintf(
			`<ansi fg="red">The %s must be your own work — it bears another maker's mark.</ansi>`,
			result.ForeignComponentName))
		return true, nil

	case result.ImmediateComplete:
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="green">%s</ansi>`, result.SuccessMsg))
		return true, nil

	case result.Initiated:
		user.SendText(messaging.CategorySystem, fmt.Sprintf(
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
		user.SendText(messaging.CategorySystem, `<ansi fg="red">You don't know that recipe yet. Keep crafting to discover new ones!</ansi>`)
		return true, nil
	}

	// Already crafting?
	if user.Character.IsCrafting() {
		user.SendText(messaging.CategorySystem, `<ansi fg="red">You are already working on something. Finish or be interrupted first.</ansi>`)
		return true, nil
	}

	// Skill gate
	skillLevel := user.Character.GetSkillLevel(skills.SkillTag(recipe.Skill))
	if skillLevel < recipe.SkillMinimum {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(
			`<ansi fg="red">Your %s skill is too low (requires %d, you have %d).</ansi>`,
			recipe.Skill, recipe.SkillMinimum, skillLevel))
		return true, nil
	}

	// Station check
	if recipe.Station != "" && room.Station != recipe.Station {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(
			`<ansi fg="red">You need to be at a %s to craft that.</ansi>`,
			strings.ReplaceAll(recipe.Station, "_", " ")))
		return true, nil
	}

	// Ingredient check
	ok, missing := crafting.HasIngredients(user.Character.Items, user.Character.ComponentItems, recipe)
	if !ok {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="red">You are missing: %s.</ansi>`, missing))
		return true, nil
	}

	// Self-crafted-component check (require_own_components)
	if ownOk, offendingName := crafting.CheckOwnComponents(recipe, user.Character.Items, user.Character.ComponentItems, user.Character.Name); !ownOk {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(
			`<ansi fg="red">The %s must be your own work — it bears another maker's mark.</ansi>`,
			offendingName))
		return true, nil
	}

	// Slot-based target resolution
	// Strip the recipe name from the input to get the optional slot specifier.
	specifier := ""
	recipeName := strings.ToLower(recipe.Name)
	restLower := strings.ToLower(strings.TrimSpace(rest))
	if strings.HasPrefix(restLower, recipeName) {
		specifier = strings.TrimSpace(rest[len(recipeName):])
	} else if strings.HasPrefix(restLower, strings.ToLower(recipe.RecipeId)) {
		specifier = strings.TrimSpace(rest[len(recipe.RecipeId):])
	}

	slotLabel, targetItem, errMsg := resolveEnchantSlot(&user.Character.Equipment, recipe.TargetType, specifier)
	if errMsg != "" {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="red">%s</ansi>`, errMsg))
		return true, nil
	}

	if targetItem == nil {
		user.SendText(messaging.CategorySystem, `<ansi fg="red">Could not find a valid item in that slot.</ansi>`)
		return true, nil
	}

	// Safety: complete immediately if time_rounds <= 0
	if recipe.TimeRounds <= 0 {
		completeCraft(user, recipe)
		return true, nil
	}

	// Start multi-round enchanting with the resolved slot.
	craftData := activity.CraftingData{
		RecipeId:    recipe.RecipeId,
		RoundsTotal: recipe.TimeRounds,
		TargetSlot:  slotLabel,
	}
	if err := user.Character.Activity.TransitionToCrafting(
		craftData,
		state.TransitionReason{
			Trigger: activity.TriggerCraftBegin,
			Actor:   state.ActorRef{UserId: user.UserId},
		},
	); err != nil {
		user.SendText(messaging.CategorySystem, `<ansi fg="red">You are already working on something. Finish or be interrupted first.</ansi>`)
		return true, nil
	}
	user.SendText(messaging.CategorySystem, fmt.Sprintf(
		`<ansi fg="yellow">You begin enchanting <ansi fg="itemname">%s</ansi>... (%s)</ansi>`,
		targetItem.DisplayName(), craftTimeDesc(recipe.TimeRounds)))

	// Quest engine: command notification
	bridge := questengine.NewGameBridge(user, room.RoomId)
	questengine.GetEngine().Notify("command", questengine.EventDetails{
		UserId:  user.UserId,
		RoomId:  room.RoomId,
		Command: "craft",
	}, bridge, bridge)

	return true, nil
}

// classifyRecipe buckets a known recipe for the current room: "ready" (makeable
// now, incl. from this room's storage), "missing" (skill+station OK, lack mats),
// or "locked" (wrong/absent station or skill too low).
func classifyRecipe(user *users.UserRecord, room *rooms.Room, r *crafting.RecipeSpec) string {
	lvl := user.Character.GetSkillLevel(skills.SkillTag(r.Skill))
	if lvl < r.SkillMinimum {
		return "locked"
	}
	if r.Station != "" && room.Station != r.Station {
		return "locked"
	}
	if ok, _ := crafting.HasIngredients(user.Character.Items, user.Character.ComponentItems, r); ok {
		return "ready"
	}
	// Completable by pulling from the player's storage (auto-pulled at craft time).
	if _, complete := crafting.PlanStoragePull(r, user.Character.Items, user.Character.ComponentItems, user.ItemStorage.GetItems()); complete {
		return "ready"
	}
	return "missing"
}

// craftRecipeRow renders a single recipe line (name + ingredients + station +
// time) using the shared craftList row style, WITHOUT the leading [X] indicator.
// Used by the bare-craft "Ready to Craft" view.
func craftRecipeRow(r *crafting.RecipeSpec) string {
	ingredientList := ingredientSummary(r)
	stationStr := ""
	if r.Station != "" {
		stationStr = fmt.Sprintf(" [%s]", strings.ReplaceAll(r.Station, "_", " "))
	}
	displayName := r.Name
	if crafting.IsEnchantingRecipe(r) && r.TargetType != "" {
		displayName = fmt.Sprintf("%s (%s)", r.Name, r.TargetType)
	}
	return fmt.Sprintf(
		`  <ansi fg="green">[V]</ansi> <ansi fg="white">%-26s</ansi> — %s  <ansi fg="dark-cyan">%s, %s</ansi>`,
		displayName, ingredientList, stationStr, craftTimeDesc(r.TimeRounds))
}

// craftCraftableNow prints only the recipes the player can craft right now in
// the current room (materials in hand or completable from this room's storage).
func craftCraftableNow(user *users.UserRecord, room *rooms.Room) bool {
	all := crafting.GetAll()

	// Collect known + ready recipes, grouped by skill for a stable ordering.
	bySkill := make(map[string][]*crafting.RecipeSpec)
	skillSet := make(map[string]struct{})
	ready := 0
	for id, r := range all {
		if !user.Character.HasRecipe(id) {
			continue
		}
		if classifyRecipe(user, room, r) != "ready" {
			continue
		}
		bySkill[r.Skill] = append(bySkill[r.Skill], r)
		skillSet[r.Skill] = struct{}{}
		ready++
	}

	user.SendText(messaging.CategorySystem, ``)
	user.SendText(messaging.CategorySystem, `<ansi fg="green-bold"> .:. Ready to Craft .:.</ansi>`)

	if ready == 0 {
		user.SendText(messaging.CategorySystem, ``)
		user.SendText(messaging.CategorySystem, `<ansi fg="yellow">Nothing you can craft here right now — try <ansi fg="cyan-bold">craft list</ansi> to see everything you know.</ansi>`)
		user.SendText(messaging.CategorySystem, ``)
		return true
	}

	skillNames := make([]string, 0, len(skillSet))
	for sk := range skillSet {
		skillNames = append(skillNames, sk)
	}
	sort.Strings(skillNames)

	for _, skillName := range skillNames {
		recipes := bySkill[skillName]
		sort.SliceStable(recipes, func(i, j int) bool { return recipes[i].Name < recipes[j].Name })
		user.SendText(messaging.CategorySystem, ``)
		user.SendText(messaging.CategorySystem, fmt.Sprintf(
			`<ansi fg="yellow">%s</ansi>`,
			titleCase(strings.ReplaceAll(skillName, "-", " "))))
		for _, r := range recipes {
			user.SendText(messaging.CategorySystem, craftRecipeRow(r))
		}
	}

	user.SendText(messaging.CategorySystem, ``)
	user.SendText(messaging.CategorySystem, `<ansi fg="cyan">Type <ansi fg="cyan-bold">craft list</ansi> to see every recipe you know.</ansi>`)
	user.SendText(messaging.CategorySystem, ``)
	return true
}

// craftList prints all known recipes, sectioned by craftability — Ready to
// craft, Missing ingredients, then Locked — with per-recipe status details.
func craftList(user *users.UserRecord, room *rooms.Room) bool {
	all := crafting.GetAll()
	if len(all) == 0 {
		user.SendText(messaging.CategorySystem, `<ansi fg="yellow">No crafting recipes are currently available.</ansi>`)
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
		user.SendText(messaging.CategorySystem, `<ansi fg="yellow">You haven't discovered any crafting recipes yet.</ansi>`)
		return true
	}

	// Bucket every known recipe by craftability, keyed by skill within the
	// bucket so we can preserve the existing per-skill grouping inside sections.
	type bucket struct {
		bySkill map[string][]*crafting.RecipeSpec
	}
	buckets := map[string]*bucket{
		"ready":   {bySkill: map[string][]*crafting.RecipeSpec{}},
		"missing": {bySkill: map[string][]*crafting.RecipeSpec{}},
		"locked":  {bySkill: map[string][]*crafting.RecipeSpec{}},
	}
	for _, r := range known {
		b := buckets[classifyRecipe(user, room, r)]
		b.bySkill[r.Skill] = append(b.bySkill[r.Skill], r)
	}

	// Overall completion accounting (known vs total across all skills).
	totalKnown := 0
	grandTotal := 0
	{
		countedSkills := make(map[string]struct{})
		for _, r := range known {
			countedSkills[r.Skill] = struct{}{}
		}
		for skillName := range countedSkills {
			allForSkill := crafting.GetAllForSkill(skillName)
			for _, r := range allForSkill {
				grandTotal++
				if user.Character.HasRecipe(r.RecipeId) {
					totalKnown++
				}
			}
		}
	}

	user.SendText(messaging.CategorySystem, ``)
	user.SendText(messaging.CategorySystem, `<ansi fg="cyan-bold"> .:. Crafting Recipes .:.</ansi>`)

	// Ordered sections, actionable ones first.
	sections := []struct {
		key    string
		header string
	}{
		{"ready", `<ansi fg="green-bold">Ready to craft</ansi>`},
		{"missing", `<ansi fg="yellow-bold">Missing ingredients</ansi>`},
		{"locked", `<ansi fg="red-bold">Locked (station or skill)</ansi>`},
	}

	for _, sec := range sections {
		b := buckets[sec.key]
		if len(b.bySkill) == 0 {
			continue
		}
		user.SendText(messaging.CategorySystem, ``)
		user.SendText(messaging.CategorySystem, fmt.Sprintf(` %s`, sec.header))

		// Sort skill names within the section for stable output.
		skillNames := make([]string, 0, len(b.bySkill))
		for sk := range b.bySkill {
			skillNames = append(skillNames, sk)
		}
		sort.Strings(skillNames)

		for _, skillName := range skillNames {
			skillLevel := user.Character.GetSkillLevel(skills.SkillTag(skillName))

			recipes := b.bySkill[skillName]
			sort.SliceStable(recipes, func(i, j int) bool { return recipes[i].Name < recipes[j].Name })

			user.SendText(messaging.CategorySystem, fmt.Sprintf(
				`  <ansi fg="yellow">%s</ansi> <ansi fg="white">(%s)</ansi>`,
				titleCase(strings.ReplaceAll(skillName, "-", " ")), skills.GetSkillRankDescription(skillLevel)))

			for _, r := range recipes {
				indicator, reason := recipeStatus(user, room, r, skillLevel)
				ingredientList := ingredientSummary(r)
				stationStr := ""
				if r.Station != "" {
					stationStr = fmt.Sprintf(" [%s]", strings.ReplaceAll(r.Station, "_", " "))
				}
				// Enchanting recipes target an equipped item; annotate the
				// recipe name with the slot for at-a-glance lookup.
				displayName := r.Name
				if crafting.IsEnchantingRecipe(r) && r.TargetType != "" {
					displayName = fmt.Sprintf("%s (%s)", r.Name, r.TargetType)
				}
				if reason != "" {
					user.SendText(messaging.CategorySystem, fmt.Sprintf(
						`    <ansi fg="red">[%s]</ansi> <ansi fg="white">%-26s</ansi> — %s  <ansi fg="red">%s</ansi><ansi fg="dark-cyan">%s, %s</ansi>`,
						indicator, displayName, ingredientList, reason, stationStr, craftTimeDesc(r.TimeRounds)))
				} else {
					user.SendText(messaging.CategorySystem, fmt.Sprintf(
						`    <ansi fg="green">[%s]</ansi> <ansi fg="white">%-26s</ansi> — %s  <ansi fg="dark-cyan">%s, %s</ansi>`,
						indicator, displayName, ingredientList, stationStr, craftTimeDesc(r.TimeRounds)))
				}
			}
		}
	}

	// Overall completion
	overallDesc := recipeCompletionTier(totalKnown, grandTotal)
	user.SendText(messaging.CategorySystem, ``)
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="cyan">Overall recipe knowledge: %s</ansi>`, overallDesc))
	user.SendText(messaging.CategorySystem, ``)
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
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="green">%s</ansi>`, recipe.SuccessMessage))
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
