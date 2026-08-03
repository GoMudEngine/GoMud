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
	RecipeId             string             `yaml:"id"`
	Name                 string             `yaml:"name"`
	Aliases              []string           `yaml:"aliases,omitempty"` // short single-word invocation forms
	Skill                string             `yaml:"skill"`
	SkillMinimum         int                `yaml:"skill_minimum"`
	RequireOwnComponents bool               `yaml:"require_own_components,omitempty"` // crafted-component ingredients must carry the crafter's MakerName
	LearnOnly            bool               `yaml:"learn_only,omitempty"`             // excluded from craft-discovery; taught only via quest learn_recipe (or admin learn)
	Station              string             `yaml:"station"`                          // "" = no station required
	TimeRounds           int                `yaml:"time_rounds"`
	Ingredients          []RecipeIngredient `yaml:"ingredients"`
	Output               RecipeOutput       `yaml:"output"`
	TargetType           string             `yaml:"target_type,omitempty"`  // equipment type consumed as enchanting input
	EnchantType          string             `yaml:"enchant_type,omitempty"` // enchantment ID to apply to target
	SuccessMessage       string             `yaml:"success_message"`
	FailureMessage       string             `yaml:"failure_message"`
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

	// Validate alias uniqueness within the recipe namespace and ensure no alias
	// collides with any RecipeId (panic on violation, mirroring the spell validator).
	seen := make(map[string]string, len(allRecipes)) // alias -> recipeId
	for _, r := range allRecipes {
		for _, a := range r.Aliases {
			a = strings.ToLower(strings.TrimSpace(a))
			if a == "" || a == r.RecipeId {
				continue // blank, or a redundant self-alias (already resolves via the id)
			}
			if _, clash := allRecipes[a]; clash {
				panic(fmt.Sprintf("recipe alias %q (on %s) collides with a recipe id", a, r.RecipeId))
			}
			if other, dup := seen[a]; dup {
				panic(fmt.Sprintf("duplicate recipe alias %q on %s and %s", a, other, r.RecipeId))
			}
			seen[a] = r.RecipeId
		}
	}

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

// recipeByOutputId is a lazy-built index from output item ID to recipe.
var recipeByOutputId map[int]*RecipeSpec

// GetRecipeByOutputItemId returns the recipe that produces the given item ID,
// or nil if no recipe outputs that item. Builds an index on first call.
func GetRecipeByOutputItemId(itemId int) *RecipeSpec {
	if recipeByOutputId == nil {
		recipeByOutputId = make(map[int]*RecipeSpec)
		for _, r := range allRecipes {
			if r.Output.ItemId > 0 {
				recipeByOutputId[r.Output.ItemId] = r
			}
		}
	}
	return recipeByOutputId[itemId]
}

// FindRecipesByName does a case-insensitive, TIERED search across recipes and
// returns every match from the highest-priority tier that has any:
//
//  1. exact recipe name
//  2. exact declared alias
//  3. substring of the recipe name
//
// Results are deterministically ordered — tightest name first (shortest name
// containing the query), alphabetical on ties — so callers that take the first
// element get the closest match every time. The old single-result search
// returned the first hit of a MAP iteration, which made `craft cloak`
// non-deterministic between cattail-down-cloak and wool-cloak (2026-08-03).
func FindRecipesByName(name string) []*RecipeSpec {
	lower := strings.ToLower(strings.TrimSpace(name))
	if lower == "" {
		return nil
	}

	var exact, alias, sub []*RecipeSpec
	for _, r := range allRecipes {
		rn := strings.ToLower(r.Name)
		if rn == lower {
			exact = append(exact, r)
			continue
		}
		aliased := false
		for _, a := range r.Aliases {
			if strings.ToLower(strings.TrimSpace(a)) == lower {
				alias = append(alias, r)
				aliased = true
				break
			}
		}
		if aliased {
			continue
		}
		if strings.Contains(rn, lower) {
			sub = append(sub, r)
		}
	}

	pick := exact
	if len(pick) == 0 {
		pick = alias
	}
	if len(pick) == 0 {
		pick = sub
	}
	sort.Slice(pick, func(i, j int) bool {
		if len(pick[i].Name) != len(pick[j].Name) {
			return len(pick[i].Name) < len(pick[j].Name)
		}
		return pick[i].Name < pick[j].Name
	})
	return pick
}

// FindRecipeByName is the deterministic single-result wrapper over
// FindRecipesByName: the tightest match, or nil.
func FindRecipeByName(name string) *RecipeSpec {
	matches := FindRecipesByName(name)
	if len(matches) == 0 {
		return nil
	}
	return matches[0]
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

// componentTagOf returns the ComponentTag of item's spec, or "" if the item
// isn't tagged as a crafting component/material. Shared matcher used by
// HasIngredients, ConsumeIngredients, and CheckOwnComponents so tag-matching
// behavior stays consistent across all three.
func componentTagOf(item items.Item) string {
	return item.GetSpec().ComponentTag
}

// HasIngredients checks whether inv and componentInv together contain all
// required ingredients for recipe.
// Returns (true, "") on success; (false, firstMissingTag) on failure.
func HasIngredients(inv []items.Item, componentInv []items.Item, recipe *RecipeSpec) (bool, string) {
	counts := make(map[string]int)
	// Count from component bag first
	for _, item := range componentInv {
		if tag := componentTagOf(item); tag != "" {
			counts[tag]++
		}
	}
	// Then from backpack
	for _, item := range inv {
		if tag := componentTagOf(item); tag != "" {
			counts[tag]++
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
		if tag := componentTagOf(item); tag != "" {
			if remaining := needed[tag]; remaining > 0 {
				needed[tag]--
				continue // consume this item
			}
		}
		newComponent = append(newComponent, item)
	}

	// Then from backpack
	newInv := make([]items.Item, 0, len(inv))
	for _, item := range inv {
		if tag := componentTagOf(item); tag != "" {
			if remaining := needed[tag]; remaining > 0 {
				needed[tag]--
				continue // consume this item
			}
		}
		newInv = append(newInv, item)
	}

	return newInv, newComponent
}

// PlanStoragePull computes which storage items would complete recipe's
// ingredients given what the actor already holds (inv = backpack, componentInv
// = component bag). Returns the exact storage items to pull and whether pulling
// them makes the recipe craftable. All-or-nothing: if storage can't cover the
// full shortfall, returns (nil, false) — pull nothing. Tag-matching mirrors
// HasIngredients / ConsumeIngredients via componentTagOf.
func PlanStoragePull(recipe *RecipeSpec, inv, componentInv, storage []items.Item) ([]items.Item, bool) {
	shortfall := make(map[string]int)
	for _, ing := range recipe.Ingredients {
		shortfall[ing.ItemTag] = ing.Quantity
	}
	for _, it := range componentInv {
		if t := componentTagOf(it); shortfall[t] > 0 {
			shortfall[t]--
		}
	}
	for _, it := range inv {
		if t := componentTagOf(it); shortfall[t] > 0 {
			shortfall[t]--
		}
	}
	pull := []items.Item{}
	for _, it := range storage {
		t := componentTagOf(it)
		if shortfall[t] > 0 {
			pull = append(pull, it)
			shortfall[t]--
		}
	}
	for _, remaining := range shortfall {
		if remaining > 0 {
			return nil, false
		}
	}
	return pull, true
}

// craftableComponentTags is the set of component_tags that some recipe
// produces as output — i.e. genuinely crafted sub-assemblies. Lazily built
// from allRecipes on first use. Used to scope require_own_components to
// crafted intermediates, exempting drop/forage reagents that are
// IsComponent (for bag routing only) but can never carry a maker's mark
// (e.g. boss-drop Folded-Space Silk, Warden Chassis-Loom).
var craftableComponentTags map[string]bool

// isCraftableComponentTag reports whether tag is produced as the output of
// some known recipe. Non-craftable tags (drop/forage reagents) are exempt
// from the require_own_components maker-mark check in CheckOwnComponents.
func isCraftableComponentTag(tag string) bool {
	if craftableComponentTags == nil {
		craftableComponentTags = make(map[string]bool)
		for _, r := range allRecipes {
			if r.Output.ItemId <= 0 {
				continue
			}
			if ct := componentTagOf(items.New(r.Output.ItemId)); ct != "" {
				craftableComponentTags[ct] = true
			}
		}
	}
	return craftableComponentTags[tag]
}

// ResetCraftableComponentTagsForTest clears the lazy craftableComponentTags
// cache so tests that register recipes via RegisterRecipeForTest (after the
// cache may have already been built by an earlier test in the same binary)
// are picked up on the next isCraftableComponentTag call. Production code
// should never call this.
func ResetCraftableComponentTagsForTest() {
	craftableComponentTags = nil
}

// CheckOwnComponents enforces require_own_components: every ingredient that
// is both (a) a crafted component (IsComponent) AND (b) genuinely craftable
// (some recipe outputs an item with that component_tag) must have been made
// by the crafter. Bulk materials and drop/forage reagents (IsComponent for
// bag-routing only, but never any recipe's output — e.g. boss-drop
// Folded-Space Silk) are exempt. Tag-matching mirrors HasIngredients /
// ConsumeIngredients via componentTagOf.
// Returns (true, "") on success; (false, offendingComponentName) on failure.
// Callers own all player-facing text (same convention as HasIngredients).
//
// NOTE: strict-any-match — if ANY matching-tag component in the pools is
// foreign, the craft refuses even if the crafter also carries their own
// copy of that component. This is deliberate: HasIngredients/ConsumeIngredients
// don't guarantee which matching item gets consumed first, so we can't
// safely assume the crafter's own copy is the one that would be used.
func CheckOwnComponents(recipe *RecipeSpec, inv, componentInv []items.Item, crafterName string) (bool, string) {
	if !recipe.RequireOwnComponents {
		return true, ""
	}

	pools := [][]items.Item{componentInv, inv}
	for _, ing := range recipe.Ingredients {
		if !isCraftableComponentTag(ing.ItemTag) {
			// Drop/forage reagent (is_component only for bag routing) — it can
			// never carry a maker's mark, so require_own_components does not
			// apply to it. Only crafted sub-assemblies are gated.
			continue
		}
		for _, pool := range pools {
			for _, item := range pool {
				if componentTagOf(item) != ing.ItemTag {
					continue
				}
				spec := item.GetSpec()
				if !spec.IsComponent {
					continue // bulk material, not a crafted component — exempt
				}
				if item.MakerName != crafterName {
					return false, spec.Name
				}
			}
		}
	}
	return true, ""
}

// ShouldStampMakerName decides whether a freshly crafted output item gets the
// crafter's MakerName. Skilled crafters (skill 30+) stamp everything except
// ordinary Object-type outputs — but component outputs (IsComponent) stamp
// REGARDLESS of Type, since components are conventionally authored
// `type: object` and require_own_components pinnacle-assembly gating needs
// their provenance (see CheckOwnComponents). Shared by every craft-completion
// path (async round tick and immediate-complete).
func ShouldStampMakerName(craftSkill int, spec items.ItemSpec) bool {
	if craftSkill < 30 {
		return false
	}
	return spec.Type != items.Object || spec.IsComponent
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
		if r.LearnOnly {
			// learn_only recipes are never a discovery candidate — quest-taught
			// (learn_recipe) or admin-learned only.
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

// IsEnchantingRecipe returns true if this recipe produces an enchanted item
// rather than a normal crafted output.
func IsEnchantingRecipe(recipe *RecipeSpec) bool {
	return recipe.EnchantType != "" && recipe.TargetType != ""
}

// RegisterRecipeForTest injects a RecipeSpec into the global registry for
// unit tests that need GetRecipe() to return a spec without loading YAML
// files from disk. The entry persists for the lifetime of the test binary.
func RegisterRecipeForTest(spec *RecipeSpec) {
	if allRecipes == nil {
		allRecipes = map[string]*RecipeSpec{}
	}
	allRecipes[spec.RecipeId] = spec
}

// UnregisterRecipeForTest removes a test-injected recipe so fixtures don't
// leak into other tests in the same binary.
func UnregisterRecipeForTest(recipeId string) {
	delete(allRecipes, recipeId)
	recipeByOutputId = nil
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
