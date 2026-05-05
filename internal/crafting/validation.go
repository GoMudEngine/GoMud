package crafting

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// ValidateRecipeIngredientTags ensures every recipe ingredient resolves
// to AT LEAST ONE item carrying the recipe's Skill in its
// VendorCategories list. Multiple items can share a ComponentTag
// (e.g. four different bottle items all tagged "bottle"); the recipe
// validates as long as some item with that tag is sourceable from the
// recipe's discipline. Ingredients with no canonical item (no ItemSpec
// has matching ComponentTag) are logged as a warning but don't error
// — those are legitimate "raw concept" tags (e.g. a flavor recipe that
// hasn't been wired to a specific item).
func ValidateRecipeIngredientTags(
	recipes map[string]*RecipeSpec,
	specs map[int]*items.ItemSpec,
) error {
	// Index items by ComponentTag — collect ALL items per tag, not just one.
	// Map iteration order in Go is randomized, so single-item indexing
	// caused intermittent boot panics whenever an alchemy-only bottle won
	// the slot for a recipe that needed a jewelcrafting-tagged bottle.
	byTag := map[string][]*items.ItemSpec{}
	for _, s := range specs {
		if s == nil || s.ComponentTag == "" {
			continue
		}
		byTag[s.ComponentTag] = append(byTag[s.ComponentTag], s)
	}

	type fault struct {
		recipeId string
		itemTag  string
		skill    string
		why      string
	}
	var faults []fault

	for id, r := range recipes {
		if r == nil {
			continue
		}
		for _, ing := range r.Ingredients {
			candidates, ok := byTag[ing.ItemTag]
			if !ok {
				mudlog.Warn("crafting.ValidateRecipeIngredientTags",
					"warning", "recipe ingredient has no canonical item",
					"recipe", id, "itemTag", ing.ItemTag)
				continue
			}
			matched := false
			for _, spec := range candidates {
				if slices.Contains(spec.VendorCategories, r.Skill) {
					matched = true
					break
				}
			}
			if !matched {
				names := make([]string, 0, len(candidates))
				for _, spec := range candidates {
					names = append(names, fmt.Sprintf("%d (%q)", spec.ItemId, spec.Name))
				}
				faults = append(faults, fault{
					recipeId: id,
					itemTag:  ing.ItemTag,
					skill:    r.Skill,
					why: fmt.Sprintf("no item with tag %q has %q in vendor_categories; candidates: %s",
						ing.ItemTag, r.Skill, strings.Join(names, ", ")),
				})
			}
		}
	}

	if len(faults) == 0 {
		return nil
	}

	sort.Slice(faults, func(i, j int) bool { return faults[i].recipeId < faults[j].recipeId })
	var b strings.Builder
	fmt.Fprintf(&b, "recipe ingredients with missing vendor_categories tags (%d):\n", len(faults))
	for _, f := range faults {
		fmt.Fprintf(&b, "  - recipe %q ingredient %q (%s): %s\n",
			f.recipeId, f.itemTag, f.skill, f.why)
	}
	return fmt.Errorf("%s", b.String())
}
