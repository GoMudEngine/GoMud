package actions

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

// CraftResult describes the outcome of an InitiateCraft call.
// Callers are responsible for all player-facing messaging.
type CraftResult struct {
	// Initiated is true when multi-round crafting has been started
	// (CraftingState set on the character).
	Initiated bool
	// ImmediateComplete is true when the recipe had TimeRounds <= 0 and was
	// completed in a single call.
	ImmediateComplete bool
	// RecipeNotFound is true when no recipe matched the given name.
	RecipeNotFound bool
	// RecipeNotKnown is true when the actor's character doesn't have the recipe
	// in their KnownRecipes map.
	RecipeNotKnown bool
	// SkillTooLow is true when the actor's skill rank is below the recipe
	// minimum.
	SkillTooLow bool
	// WrongStation is true when the recipe requires a station the current room
	// does not provide.
	WrongStation bool
	// MissingIngredients is true when the actor lacks one or more ingredients.
	MissingIngredients bool
	// AlreadyCrafting is true when the character already has an active
	// CraftingState.
	AlreadyCrafting bool

	// Descriptive data filled in on all non-error paths (for messaging).
	RecipeName    string
	SkillName     string
	SkillLevel    int
	SkillMinimum  int
	TimeRounds    int    // recipe.TimeRounds — for duration-description messaging
	StationNeeded string
	MissingTag    string
	OutputName    string // display name of the produced item (immediate-complete only)
	SuccessMsg    string // recipe.SuccessMessage
}

// InitiateCraft attempts to begin (or immediately complete) a crafting
// operation for actor using the recipe identified by recipeName.
//
// Enchanting recipes are intentionally NOT handled here — that path requires
// player-specific target disambiguation and stays in the user command wrapper.
//
// Callers are responsible for:
//   - Skill progression (OnSkillUse)
//   - Quest engine notifications
//   - All player-facing text
func InitiateCraft(actor Actor, recipeName string) CraftResult {
	char := actor.GetCharacter()
	room := actor.GetRoom()

	// ── Already crafting? ─────────────────────────────────────────────────────
	if char.IsCrafting() {
		return CraftResult{AlreadyCrafting: true}
	}

	// ── Recipe lookup ─────────────────────────────────────────────────────────
	recipe := crafting.FindRecipeByName(recipeName)
	if recipe == nil {
		return CraftResult{RecipeNotFound: true}
	}

	res := CraftResult{
		RecipeName:   recipe.Name,
		SkillName:    recipe.Skill,
		SkillMinimum: recipe.SkillMinimum,
		TimeRounds:   recipe.TimeRounds,
		SuccessMsg:   recipe.SuccessMessage,
	}

	// ── Known-recipe gate ─────────────────────────────────────────────────────
	if !char.HasRecipe(recipe.RecipeId) {
		res.RecipeNotKnown = true
		return res
	}

	// ── Skill level gate ──────────────────────────────────────────────────────
	skillLevel := char.GetSkillLevel(skills.SkillTag(recipe.Skill))
	res.SkillLevel = skillLevel
	if skillLevel < recipe.SkillMinimum {
		res.SkillTooLow = true
		return res
	}

	// ── Station check ─────────────────────────────────────────────────────────
	if recipe.Station != "" && room.Station != recipe.Station {
		res.StationNeeded = strings.ReplaceAll(recipe.Station, "_", " ")
		res.WrongStation = true
		return res
	}

	// ── Ingredient check ──────────────────────────────────────────────────────
	ok, missingTag := crafting.HasIngredients(char.Items, char.ComponentItems, recipe)
	if !ok {
		res.MissingTag = missingTag
		res.MissingIngredients = true
		return res
	}

	// ── Enchanting recipes: caller handles these (user-only complexity) ───────
	// We only proceed for normal crafting recipes here.
	if crafting.IsEnchantingRecipe(recipe) {
		// Return as if recipe not found so the user wrapper can take over.
		// Mob callers simply won't request enchanting recipes.
		return CraftResult{RecipeNotFound: true}
	}

	// ── Immediate completion (TimeRounds <= 0) ────────────────────────────────
	if recipe.TimeRounds <= 0 {
		char.Items, char.ComponentItems = crafting.ConsumeIngredients(
			char.Items, char.ComponentItems, recipe)
		newItem := items.New(recipe.Output.ItemId)
		char.StoreItem(newItem)
		res.OutputName = newItem.DisplayName()
		res.ImmediateComplete = true
		return res
	}

	// ── Start multi-round crafting ────────────────────────────────────────────
	char.CraftingState = &characters.CraftingState{
		RecipeId:    recipe.RecipeId,
		RoundsTotal: recipe.TimeRounds,
	}
	res.Initiated = true
	return res
}
