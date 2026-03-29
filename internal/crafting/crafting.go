// Package crafting implements the data-driven recipe and crafting framework (Stage 13.1).
// New recipes require only a YAML file in _datafiles/world/dogmud/recipes/<skill>/<id>.yaml.
package crafting

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/pkg/errors"
)

// RecipeIngredient describes a single ingredient requirement for a recipe.
type RecipeIngredient struct {
	ItemTag  string `yaml:"item_tag"`
	Quantity int    `yaml:"quantity"`
}

// RecipeOutput describes the item produced by a successful craft.
type RecipeOutput struct {
	ItemId   int `yaml:"item_id"`
	Quantity int `yaml:"quantity"`
}

// RecipeSpec is the data-driven definition of a crafting recipe loaded from YAML.
type RecipeSpec struct {
	RecipeId       string             `yaml:"id"`
	Name           string             `yaml:"name"`
	Skill          string             `yaml:"skill"`
	SkillMinimum   int                `yaml:"skill_minimum"`
	Station        string             `yaml:"station"`        // "" = no station required
	TimeRounds     int                `yaml:"time_rounds"`
	Ingredients    []RecipeIngredient `yaml:"ingredients"`
	Output         RecipeOutput       `yaml:"output"`
	TargetType     string             `yaml:"target_type,omitempty"`  // equipment type consumed as enchanting input
	EnchantType    string             `yaml:"enchant_type,omitempty"` // enchantment ID to apply to target
	SuccessMessage string             `yaml:"success_message"`
	FailureMessage string             `yaml:"failure_message"`
}

// Id implements fileloader.Loadable.
func (r *RecipeSpec) Id() string { return r.RecipeId }

// Filepath implements fileloader.Loadable.
// Returns "skill/id.yaml" using the OS path separator so the fileloader path check passes.
func (r *RecipeSpec) Filepath() string {
	return util.FilePath(r.Skill + "/" + r.RecipeId + ".yaml")
}

// Validate implements fileloader.Loadable.
func (r *RecipeSpec) Validate() error {
	if r.RecipeId == "" {
		return fmt.Errorf("recipe id cannot be empty")
	}
	if r.Name == "" {
		return fmt.Errorf("recipe %q: name cannot be empty", r.RecipeId)
	}
	if r.Skill == "" {
		return fmt.Errorf("recipe %q: skill cannot be empty", r.RecipeId)
	}
	// Enchanting recipes use target_type instead of output
	if r.Output.ItemId < 1 && r.EnchantType == "" {
		return fmt.Errorf("recipe %q: output.item_id must be > 0 (or enchant_type must be set)", r.RecipeId)
	}
	return nil
}

// Package-level registry, populated by LoadRecipeFiles.
var allRecipes map[string]*RecipeSpec

// LoadRecipeFiles reads all YAML files from the recipes/ data directory and
// populates the in-memory registry. Called once at startup from main.go.
func LoadRecipeFiles() {
	start := time.Now()

	dataPath := string(configs.GetFilePathsConfig().DataFiles) + `/recipes`
	tmpAll, err := fileloader.LoadAllFlatFiles[string, *RecipeSpec](dataPath)
	if err != nil {
		panic(errors.Wrap(err, `filepath: `+dataPath))
	}

	allRecipes = tmpAll
	mudlog.Info("crafting.LoadRecipeFiles()", "loadedCount", len(allRecipes), "Time Taken", time.Since(start))
}

// GetRecipe returns the RecipeSpec for a given id, or nil if not found.
func GetRecipe(id string) *RecipeSpec {
	if allRecipes == nil {
		return nil
	}
	return allRecipes[id]
}

// GetAll returns the full recipe registry map.
func GetAll() map[string]*RecipeSpec {
	return allRecipes
}

// FindRecipeByName does a case-insensitive search across recipe names.
// Prefers exact matches over substring matches.
func FindRecipeByName(name string) *RecipeSpec {
	lower := strings.ToLower(name)

	// First pass: exact match
	for _, r := range allRecipes {
		if strings.ToLower(r.Name) == lower {
			return r
		}
	}

	// Second pass: substring match
	for _, r := range allRecipes {
		if strings.Contains(strings.ToLower(r.Name), lower) {
			return r
		}
	}
	return nil
}

// GetAllForSkill returns all recipes for a given skill, sorted by name.
func GetAllForSkill(skill string) []*RecipeSpec {
	var result []*RecipeSpec
	for _, r := range allRecipes {
		if r.Skill == skill {
			result = append(result, r)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// HasIngredients checks whether inv and componentInv together contain all
// required ingredients for recipe.
// Returns (true, "") on success; (false, firstMissingTag) on failure.
func HasIngredients(inv []items.Item, componentInv []items.Item, recipe *RecipeSpec) (bool, string) {
	counts := make(map[string]int)
	// Count from component bag first
	for _, item := range componentInv {
		spec := item.GetSpec()
		if spec.ComponentTag != "" {
			counts[spec.ComponentTag]++
		}
	}
	// Then from backpack
	for _, item := range inv {
		spec := item.GetSpec()
		if spec.ComponentTag != "" {
			counts[spec.ComponentTag]++
		}
	}
	for _, ing := range recipe.Ingredients {
		if counts[ing.ItemTag] < ing.Quantity {
			return false, ing.ItemTag
		}
	}
	return true, ""
}

// ConsumeIngredients removes the required items from componentInv first, then
// inv, and returns the remainders of both pools.
// Items are matched by ComponentTag; exactly the needed quantity is consumed.
func ConsumeIngredients(inv []items.Item, componentInv []items.Item, recipe *RecipeSpec) ([]items.Item, []items.Item) {
	needed := make(map[string]int)
	for _, ing := range recipe.Ingredients {
		needed[ing.ItemTag] = ing.Quantity
	}

	// Consume from component bag first
	newComponent := make([]items.Item, 0, len(componentInv))
	for _, item := range componentInv {
		spec := item.GetSpec()
		if spec.ComponentTag != "" {
			if remaining := needed[spec.ComponentTag]; remaining > 0 {
				needed[spec.ComponentTag]--
				continue // consume this item
			}
		}
		newComponent = append(newComponent, item)
	}

	// Then from backpack
	newInv := make([]items.Item, 0, len(inv))
	for _, item := range inv {
		spec := item.GetSpec()
		if spec.ComponentTag != "" {
			if remaining := needed[spec.ComponentTag]; remaining > 0 {
				needed[spec.ComponentTag]--
				continue // consume this item
			}
		}
		newInv = append(newInv, item)
	}

	return newInv, newComponent
}

// GetStarterRecipes returns a map of all recipes with SkillMinimum == 0,
// each set to value 1 (known). Used to seed new or existing characters.
func GetStarterRecipes() map[string]int {
	result := make(map[string]int)
	for id, r := range allRecipes {
		if r.SkillMinimum == 0 {
			result[id] = 1
		}
	}
	return result
}

// GetEligibleRecipes returns recipe IDs that the player could discover:
// not already known, the player's skill level meets the recipe's SkillMinimum,
// and the recipe belongs to currentSkill (so blacksmithing can't discover
// enchanting recipes).
func GetEligibleRecipes(knownRecipes map[string]int, skillLevels map[string]int, currentSkill string) []string {
	var eligible []string
	for id, r := range allRecipes {
		if r.Skill != currentSkill {
			continue
		}
		if _, known := knownRecipes[id]; known {
			continue
		}
		if skillLevels[r.Skill] >= r.SkillMinimum {
			eligible = append(eligible, id)
		}
	}
	return eligible
}

// TargetCandidate represents a potential enchanting target.
type TargetCandidate struct {
	BackpackIdx int
	Item        items.Item
	Source      string // "backpack" or "equipped"
	SourceLabel string // e.g. "wielded", "worn - body" (empty for backpack)
}

// EquipmentSlot is a lightweight descriptor of a single worn item slot,
// used by FindTargetItems to avoid an import cycle with the characters package.
type EquipmentSlot struct {
	Item  items.Item
	Label string // e.g. "wielded", "worn - body"
}

// FindTargetItems searches inventory and equipment slots for items matching
// targetType. If specifier is non-empty, filters by item name substring.
// Pass nil (or an empty slice) for slots to search only the backpack.
func FindTargetItems(inv []items.Item, slots []EquipmentSlot, targetType string, specifier string) []TargetCandidate {
	var candidates []TargetCandidate

	for i, item := range inv {
		if item.ItemId < 1 {
			continue
		}
		spec := item.GetSpec()
		if string(spec.Type) != targetType {
			continue
		}
		if specifier != "" && !strings.Contains(strings.ToLower(item.DisplayName()), strings.ToLower(specifier)) {
			continue
		}
		candidates = append(candidates, TargetCandidate{
			BackpackIdx: i,
			Item:        item,
			Source:      "backpack",
			SourceLabel: "",
		})
	}

	for _, slot := range slots {
		if slot.Item.ItemId < 1 {
			continue
		}
		spec := slot.Item.GetSpec()
		if string(spec.Type) != targetType {
			continue
		}
		if specifier != "" && !strings.Contains(strings.ToLower(slot.Item.DisplayName()), strings.ToLower(specifier)) {
			continue
		}
		candidates = append(candidates, TargetCandidate{
			BackpackIdx: -1,
			Item:        slot.Item,
			Source:      "equipped",
			SourceLabel: slot.Label,
		})
	}

	return candidates
}

// FindTargetItem is a backward-compatible wrapper. Returns the first match index.
func FindTargetItem(inv []items.Item, targetType string) (int, bool) {
	candidates := FindTargetItems(inv, nil, targetType, "")
	if len(candidates) > 0 {
		return candidates[0].BackpackIdx, true
	}
	return -1, false
}

// IsEnchantingRecipe returns true if this recipe produces an enchanted item
// rather than a normal crafted output.
func IsEnchantingRecipe(recipe *RecipeSpec) bool {
	return recipe.EnchantType != "" && recipe.TargetType != ""
}

// CalcSuccessChance returns the crafting success percentage clamped to
// [CraftingMinSuccessChance, CraftingMaxSuccessChance].
// Formula: clamp(base + (skillLevel - skillMinimum) * bonusPerLevel, min, max)
func CalcSuccessChance(skillLevel, skillMinimum int) int {
	b := configs.GetBalanceConfig()
	base := int(b.CraftingBaseSuccessChance)
	bonusPerLevel := int(b.CraftingSkillBonusPerLevel)
	minChance := int(b.CraftingMinSuccessChance)
	maxChance := int(b.CraftingMaxSuccessChance)
	chance := base + (skillLevel-skillMinimum)*bonusPerLevel
	if chance < minChance {
		return minChance
	}
	if chance > maxChance {
		return maxChance
	}
	return chance
}
