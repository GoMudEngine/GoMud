package actions

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
)

// CraftResult describes the outcome of an InitiateCraft call.
// Callers are responsible for all player-facing messaging.
type CraftResult struct {
	// Initiated is true when multi-round crafting has been started
	// (Activity machine transitioned to Crafting).
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
	// ForeignComponent is true when the recipe requires self-crafted
	// components (RequireOwnComponents) and a matching ingredient in the
	// actor's pools was made by someone else (or has no maker at all).
	ForeignComponent bool
	// AlreadyCrafting is true when the character already has an active
	// CraftingState.
	AlreadyCrafting bool

	// Descriptive data filled in on all non-error paths (for messaging).
	RecipeName           string
	SkillName            string
	SkillLevel           int
	SkillMinimum         int
	TimeRounds           int // recipe.TimeRounds — for duration-description messaging
	StationNeeded        string
	MissingTag           string
	ForeignComponentName string // name of the offending component (ForeignComponent only)
	OutputName           string // display name of the produced item (immediate-complete only)
	SuccessMsg           string // recipe.SuccessMessage
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

	// ── Self-crafted-component gate (require_own_components) ─────────────────
	if ownOk, offendingName := crafting.CheckOwnComponents(recipe, char.Items, char.ComponentItems, char.Name); !ownOk {
		res.ForeignComponentName = offendingName
		res.ForeignComponent = true
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
		newItem.CraftSkill = skillLevel
		// Maker's mark — same policy as the async completion path
		// (crafting.ShouldStampMakerName): components stamp regardless of
		// Type so require_own_components provenance works for
		// TimeRounds<=0 sub-recipes too.
		if crafting.ShouldStampMakerName(newItem.CraftSkill, newItem.GetSpec()) {
			newItem.MakerName = char.Name
		}
		char.StoreItem(newItem)
		res.OutputName = newItem.DisplayName()
		res.ImmediateComplete = true
		return res
	}

	// ── Start multi-round crafting ────────────────────────────────────────────
	craftData := activity.CraftingData{
		RecipeId:    recipe.RecipeId,
		RoundsTotal: recipe.TimeRounds,
	}
	actorRef := state.ActorRef{
		UserId:        actor.GetUserId(),
		MobInstanceId: actor.GetMobInstanceId(),
	}
	if err := char.Activity.TransitionToCrafting(
		craftData,
		state.TransitionReason{
			Trigger: activity.TriggerCraftBegin,
			Actor:   actorRef,
		},
	); err != nil {
		res.AlreadyCrafting = true
		return res
	}

	res.Initiated = true
	return res
}
