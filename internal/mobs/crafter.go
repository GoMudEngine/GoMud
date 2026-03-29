package mobs

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// CraftResult describes the outcome of a crafter mob's tick, so the calling
// hook can emit room messages and world events without import cycles.
type CraftResult struct {
	Success      bool
	RecipeName   string
	OutputItemId int
	SkillMinimum int
	MobName      string
	Zone         string
}

// TickMobCraft handles autonomous crafting for crafter mobs. Crafting only
// fires on the material restock tick (every CrafterMaterialRestockRate rounds):
// materials arrive, and the mob immediately attempts one craft from them.
// Returns a non-nil CraftResult only when a craft is attempted.
func TickMobCraft(mob *Mob) *CraftResult {
	if !mob.Crafter {
		return nil
	}
	b := configs.GetBalanceConfig()
	if !bool(b.CrafterEnabled) {
		return nil
	}
	if mob.Character.Aggro != nil {
		return nil
	}

	roundCount := util.GetRoundCount()

	// Initialize restock timer on first call
	restockRate := uint64(b.CrafterMaterialRestockRate)
	if restockRate == 0 {
		return nil
	}
	if mob.crafterLastRestockRound == 0 {
		mob.crafterLastRestockRound = roundCount
		return nil
	}

	// Only act on the restock tick
	if roundCount-mob.crafterLastRestockRound < restockRate {
		return nil
	}
	mob.crafterLastRestockRound = roundCount

	// Restock materials
	for _, itemId := range mob.CrafterRestockMaterials {
		itm := items.New(itemId)
		if itm.ItemId > 0 {
			mob.Character.StoreItem(itm)
		}
	}

	// Pick a recipe and attempt it immediately
	recipe := pickEligibleRecipe(mob)
	if recipe == nil {
		return nil
	}

	// Roll success
	skillLevel := mob.Character.GetSkillLevel(skills.SkillTag(recipe.Skill))
	chance := crafting.CalcSuccessChance(skillLevel, recipe.SkillMinimum)

	result := &CraftResult{
		RecipeName:   recipe.Name,
		OutputItemId: recipe.Output.ItemId,
		SkillMinimum: recipe.SkillMinimum,
		MobName:      mob.Character.Name,
		Zone:         mob.Character.Zone,
	}

	// Consume ingredients regardless of success
	// Mobs don't have a component bag, so pass an empty slice as the second arg.
	backpack := mob.Character.GetAllBackpackItems()
	remaining, _ := crafting.ConsumeIngredients(backpack, []items.Item{}, recipe)
	mob.Character.Items = remaining

	if util.Rand(100) < chance {
		result.Success = true
		// Stock the output item in the mob's shop
		if recipe.Output.ItemId > 0 {
			for i := 0; i < recipe.Output.Quantity; i++ {
				mob.Character.Shop.StockItem(recipe.Output.ItemId)
			}
		}
		// Skill progression
		mob.Character.OnSkillUse(recipe.Skill, 0)
	}

	return result
}

// pickEligibleRecipe finds a random recipe the mob can attempt.
func pickEligibleRecipe(mob *Mob) *crafting.RecipeSpec {
	backpack := mob.Character.GetAllBackpackItems()
	var eligible []*crafting.RecipeSpec

	for _, recipeId := range mob.CrafterRecipeIds {
		recipe := crafting.GetRecipe(recipeId)
		if recipe == nil {
			continue
		}
		// Must match the mob's craft skill
		if recipe.Skill != mob.CrafterSkill {
			continue
		}
		// Skill minimum check
		skillLevel := mob.Character.GetSkillLevel(skills.SkillTag(recipe.Skill))
		if skillLevel < recipe.SkillMinimum {
			continue
		}
		// Ingredient check (mobs don't have a component bag)
		ok, _ := crafting.HasIngredients(backpack, []items.Item{}, recipe)
		if !ok {
			continue
		}
		eligible = append(eligible, recipe)
	}

	if len(eligible) == 0 {
		return nil
	}
	return eligible[util.Rand(len(eligible))]
}
