package crafting

// Stage 4a pinnacle-crafting backbone: integration test of the assembly
// self-craft gate against the REAL shipped recipe + item data files.
//
// Unlike crafting_own_components_test.go (which uses synthetic specs), this
// test loads the actual _datafiles/world/dogmud/recipes and item YAMLs so a
// regression in the shipped data — a dropped require_own_components /
// learn_only flag, a mistagged ingredient, a renamed component — fails here
// deterministically, mirroring the boot-time checks.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"gopkg.in/yaml.v3"
)

// repoDataRoot walks up from the test's working directory to locate
// _datafiles/world/dogmud (tests run from the package dir).
func repoDataRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot get working directory: %v", err)
	}
	for {
		candidate := filepath.Join(dir, "_datafiles", "world", "dogmud")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("cannot find _datafiles from %s", dir)
		}
		dir = parent
	}
}

// loadRealRecipes reads every recipe YAML under recipes/ into RecipeSpecs.
func loadRealRecipes(t *testing.T, root string) map[string]*RecipeSpec {
	t.Helper()
	out := map[string]*RecipeSpec{}
	recipeDir := filepath.Join(root, "recipes")
	err := filepath.Walk(recipeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".yaml") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var r RecipeSpec
		if err := yaml.Unmarshal(data, &r); err != nil {
			t.Fatalf("unmarshal recipe %s: %v", path, err)
		}
		out[r.RecipeId] = &r
		return nil
	})
	if err != nil {
		t.Fatalf("walk recipes: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("no recipes loaded — check path")
	}
	return out
}

// miniSpec is the subset of item fields the assembly gate + tag validation
// need. Unmarshalling the full items.ItemSpec risks tripping custom
// unmarshalers on nested types, so we read a flat projection and rebuild.
type miniSpec struct {
	ItemId           int      `yaml:"itemid"`
	Name             string   `yaml:"name"`
	Type             string   `yaml:"type"`
	ComponentTag     string   `yaml:"component_tag"`
	IsComponent      bool     `yaml:"is_component"`
	VendorCategories []string `yaml:"vendor_categories"`
}

// loadRealItemSpecs reads every item YAML under items/ into items.ItemSpecs
// (projected to the fields under test).
func loadRealItemSpecs(t *testing.T, root string) map[int]*items.ItemSpec {
	t.Helper()
	out := map[int]*items.ItemSpec{}
	itemDir := filepath.Join(root, "items")
	err := filepath.Walk(itemDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".yaml") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var m miniSpec
		if err := yaml.Unmarshal(data, &m); err != nil {
			return nil // skip anything that isn't a flat item spec
		}
		if m.ItemId == 0 {
			return nil
		}
		out[m.ItemId] = &items.ItemSpec{
			ItemId:           m.ItemId,
			Name:             m.Name,
			Type:             items.ItemType(m.Type),
			ComponentTag:     m.ComponentTag,
			IsComponent:      m.IsComponent,
			VendorCategories: m.VendorCategories,
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk items: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("no item specs loaded — check path")
	}
	return out
}

// the nine shipped Stage-4a assembly recipe slugs.
var stage4aAssemblies = []string{
	"assemble-the-blackrazor",
	"assemble-aegis-of-mockery",
	"assemble-thornwall-harness",
	"assemble-wayfarers-pack",
	"assemble-zephyr-treads",
	"assemble-phial-of-second-birth",
	"assemble-vitalis-bandolier",
	"assemble-seething-prism",
	"assemble-hollow-choir-staff",
}

// TestPinnacleAssemblyGate_RealData verifies CheckOwnComponents against the
// real assemble-the-blackrazor recipe and the real component ItemSpecs it
// consumes (hungering-guard 40210 + obsidian-edge-resin 40211).
func TestPinnacleAssemblyGate_RealData(t *testing.T) {
	root := repoDataRoot(t)
	recipes := loadRealRecipes(t, root)
	specs := loadRealItemSpecs(t, root)

	recipe := recipes["assemble-the-blackrazor"]
	if recipe == nil {
		t.Fatal("assemble-the-blackrazor recipe not found in shipped data")
	}
	if !recipe.RequireOwnComponents {
		t.Fatal("assemble-the-blackrazor: require_own_components is not set in shipped data")
	}

	// Locate the two crafted-component ingredients this recipe consumes and
	// grab their REAL specs from disk.
	guard := specs[40210]  // hungering-guard
	resin := specs[40211]  // obsidian-edge-resin
	if guard == nil || resin == nil {
		t.Fatalf("real component specs missing: guard=%v resin=%v", guard, resin)
	}
	if !guard.IsComponent || guard.ComponentTag != "hungering-guard" {
		t.Fatalf("40210 not a hungering-guard component: is_component=%v tag=%q", guard.IsComponent, guard.ComponentTag)
	}
	if !resin.IsComponent || resin.ComponentTag != "obsidian-edge-resin" {
		t.Fatalf("40211 not an obsidian-edge-resin component: is_component=%v tag=%q", resin.IsComponent, resin.ComponentTag)
	}

	// Seed the real specs into the items registry so items.New() resolves them.
	restore := items.SeedItemsForTest(map[int]*items.ItemSpec{
		40210: guard,
		40211: resin,
	})
	defer restore()

	const smith = "TestSmith"

	// Case 1: BOTH components made by the crafter → accepted.
	mineGuard := items.New(40210)
	mineGuard.MakerName = smith
	mineResin := items.New(40211)
	mineResin.MakerName = smith
	if ok, offending := CheckOwnComponents(recipe, []items.Item{mineGuard, mineResin}, nil, smith); !ok {
		t.Fatalf("own components rejected: offending=%q", offending)
	}

	// Case 2: one component made by someone else → refused, names that component.
	foreignResin := items.New(40211)
	foreignResin.MakerName = "SomeoneElse"
	ok, offending := CheckOwnComponents(recipe, []items.Item{mineGuard, foreignResin}, nil, smith)
	if ok {
		t.Fatal("foreign component accepted")
	}
	if offending != resin.Name {
		t.Errorf("offending name = %q, want %q", offending, resin.Name)
	}

	// Case 3: a component with no MakerName (unmade / looted) → refused.
	unmadeGuard := items.New(40210)
	if ok, _ := CheckOwnComponents(recipe, []items.Item{unmadeGuard, mineResin}, nil, smith); ok {
		t.Fatal("maker-less component accepted")
	}
}

// TestPinnacleAssemblies_FlagsIntact is a regression guard: every shipped
// assembly recipe must keep BOTH require_own_components and learn_only. If a
// future edit drops either flag, the pinnacle gate silently breaks — this
// fails first.
func TestPinnacleAssemblies_FlagsIntact(t *testing.T) {
	root := repoDataRoot(t)
	recipes := loadRealRecipes(t, root)

	for _, slug := range stage4aAssemblies {
		r := recipes[slug]
		if r == nil {
			t.Errorf("assembly recipe %q missing from shipped data", slug)
			continue
		}
		if !r.RequireOwnComponents {
			t.Errorf("%s: require_own_components must be true", slug)
		}
		if !r.LearnOnly {
			t.Errorf("%s: learn_only must be true", slug)
		}
	}
}

// TestPinnacleAssemblies_IngredientTags mirrors the boot-time
// ValidateRecipeIngredientTags invariant at the Go level for the nine
// assembly recipes: every ingredient tag must resolve to at least one
// ItemSpec whose VendorCategories includes the recipe's Skill.
func TestPinnacleAssemblies_IngredientTags(t *testing.T) {
	root := repoDataRoot(t)
	recipes := loadRealRecipes(t, root)
	specs := loadRealItemSpecs(t, root)

	// Index all items by ComponentTag.
	byTag := map[string][]*items.ItemSpec{}
	for _, s := range specs {
		if s.ComponentTag != "" {
			byTag[s.ComponentTag] = append(byTag[s.ComponentTag], s)
		}
	}

	for _, slug := range stage4aAssemblies {
		r := recipes[slug]
		if r == nil {
			t.Errorf("assembly recipe %q missing", slug)
			continue
		}
		for _, ing := range r.Ingredients {
			candidates := byTag[ing.ItemTag]
			if len(candidates) == 0 {
				t.Errorf("%s: ingredient %q resolves to no item spec", slug, ing.ItemTag)
				continue
			}
			matched := false
			for _, s := range candidates {
				for _, vc := range s.VendorCategories {
					if vc == r.Skill {
						matched = true
						break
					}
				}
				if matched {
					break
				}
			}
			if !matched {
				t.Errorf("%s: ingredient %q has no item with %q in vendor_categories", slug, ing.ItemTag, r.Skill)
			}
		}
	}
}
