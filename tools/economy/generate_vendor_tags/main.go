// Walks all recipe YAMLs and item YAMLs to produce a deterministic
// proposal of vendor_categories per item. Output: a CSV the human
// reviews before the subagent applies it to YAMLs.
//
// Run:
//
//	go run tools/economy/generate_vendor_tags/main.go
//
// Output: tools/economy/generate_vendor_tags/vendor_tag_proposal.csv
package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"
)

const (
	itemsDir   = "_datafiles/world/dogmud/items"
	recipesDir = "_datafiles/world/dogmud/recipes"
	outPath    = "tools/economy/generate_vendor_tags/vendor_tag_proposal.csv"
)

type itemYAML struct {
	ItemId           int      `yaml:"itemid"`
	Name             string   `yaml:"name"`
	Type             string   `yaml:"type"`
	Subtype          string   `yaml:"subtype"`
	Value            int      `yaml:"value"`
	QuestToken       string   `yaml:"questtoken"`
	ComponentTag     string   `yaml:"component_tag"`
	IsComponent      bool     `yaml:"is_component"`
	VendorCategories []string `yaml:"vendor_categories"`
}

type recipeYAML struct {
	Id          string `yaml:"id"`
	Skill       string `yaml:"skill"`
	Ingredients []struct {
		ItemTag  string `yaml:"item_tag"`
		Quantity int    `yaml:"quantity"`
	} `yaml:"ingredients"`
	Output struct {
		ItemId int `yaml:"item_id"`
	} `yaml:"output"`
}

func main() {
	items, err := loadAllItems()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load items: %v\n", err)
		os.Exit(1)
	}
	recipes, err := loadAllRecipes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load recipes: %v\n", err)
		os.Exit(1)
	}

	// Build component_tag → item lookup for resolving recipe ingredients.
	byCompTag := map[string]*itemYAML{}
	for i := range items {
		if items[i].ComponentTag != "" {
			byCompTag[items[i].ComponentTag] = &items[i]
		}
	}

	// proposal[itemId] = set(disciplines)
	proposal := map[int]map[string]bool{}
	source := map[int][]string{} // human-readable provenance

	addTag := func(itemId int, disc, why string) {
		if proposal[itemId] == nil {
			proposal[itemId] = map[string]bool{}
		}
		proposal[itemId][disc] = true
		source[itemId] = append(source[itemId], why)
	}

	// ── Pass 1: ingredient walk ────────────────────────────────
	for _, r := range recipes {
		for _, ing := range r.Ingredients {
			it, ok := byCompTag[ing.ItemTag]
			if !ok {
				continue // ghost tag — caller will surface as warning later
			}
			addTag(it.ItemId, r.Skill, fmt.Sprintf("ingredient of %s recipe %s", r.Skill, r.Id))
		}
		if r.Output.ItemId > 0 {
			addTag(r.Output.ItemId, r.Skill, fmt.Sprintf("output of %s recipe %s", r.Skill, r.Id))
		}
	}

	// ── Pass 2: type-based heuristics for finished goods ───────
	for i := range items {
		it := &items[i]
		if it.Value <= 0 || it.QuestToken != "" {
			continue
		}
		// Skip if already covered by recipe pass.
		if proposal[it.ItemId] != nil && len(proposal[it.ItemId]) > 0 {
			continue
		}
		switch strings.ToLower(it.Type) {
		case "weapon":
			switch strings.ToLower(it.Subtype) {
			case "wand", "sceptre", "staff":
				addTag(it.ItemId, "enchanting", "weapon subtype "+it.Subtype+" → enchanting")
			default:
				addTag(it.ItemId, "blacksmithing", "weapon → blacksmithing")
			}
		case "head", "body", "legs", "feet", "gloves", "shoulders":
			// Metal vs cloth vs leather — heuristic on subtype/name.
			n := strings.ToLower(it.Name)
			sub := strings.ToLower(it.Subtype)
			switch {
			case strings.Contains(n, "leather") || sub == "leather":
				addTag(it.ItemId, "blacksmithing", "leather armor → blacksmithing+tailoring")
				addTag(it.ItemId, "tailoring", "leather armor → blacksmithing+tailoring")
			case strings.Contains(n, "cloth") || strings.Contains(n, "robe") ||
				strings.Contains(n, "hood") || strings.Contains(n, "linen") ||
				sub == "cloth":
				addTag(it.ItemId, "tailoring", "cloth armor → tailoring")
			default:
				addTag(it.ItemId, "blacksmithing", "metal armor → blacksmithing")
			}
		case "neck", "ring":
			addTag(it.ItemId, "jewelcrafting", "jewelry → jewelcrafting")
		case "back", "belt", "wrist":
			addTag(it.ItemId, "tailoring", "accessory → tailoring (review)")
		case "potion":
			addTag(it.ItemId, "alchemy", "potion → alchemy")
		case "food":
			addTag(it.ItemId, "cooking", "food → cooking")
		case "scroll":
			addTag(it.ItemId, "enchanting", "scroll → enchanting")
		case "componentbag", "container":
			addTag(it.ItemId, "tailoring", "bag → tailoring (review)")
		case "object":
			// Can't infer — leave for human review (CSV row will show empty proposal).
		}
	}

	// ── Emit CSV ───────────────────────────────────────────────
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}
	f, err := os.Create(outPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create csv: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()

	_ = w.Write([]string{
		"item_id", "name", "type", "subtype", "component_tag",
		"value", "questtoken", "current_tags", "proposed_tags",
		"needs_review", "provenance",
	})

	sort.Slice(items, func(i, j int) bool { return items[i].ItemId < items[j].ItemId })
	for _, it := range items {
		var proposed []string
		for d := range proposal[it.ItemId] {
			proposed = append(proposed, d)
		}
		sort.Strings(proposed)

		needsReview := ""
		if it.Value > 0 && it.QuestToken == "" && len(proposed) == 0 {
			needsReview = "YES"
		}

		_ = w.Write([]string{
			fmt.Sprintf("%d", it.ItemId),
			it.Name,
			it.Type,
			it.Subtype,
			it.ComponentTag,
			fmt.Sprintf("%d", it.Value),
			it.QuestToken,
			strings.Join(it.VendorCategories, ","),
			strings.Join(proposed, ","),
			needsReview,
			strings.Join(source[it.ItemId], "; "),
		})
	}

	fmt.Printf("Wrote %s with %d rows.\n", outPath, len(items))
}

func loadAllItems() ([]itemYAML, error) {
	var items []itemYAML
	err := filepath.Walk(itemsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var it itemYAML
		if err := yaml.Unmarshal(raw, &it); err != nil {
			fmt.Fprintf(os.Stderr, "WARN: parse %s: %v\n", path, err)
			return nil
		}
		if it.ItemId > 0 {
			items = append(items, it)
		}
		return nil
	})
	return items, err
}

func loadAllRecipes() ([]recipeYAML, error) {
	var recipes []recipeYAML
	err := filepath.Walk(recipesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var r recipeYAML
		if err := yaml.Unmarshal(raw, &r); err != nil {
			fmt.Fprintf(os.Stderr, "WARN: parse %s: %v\n", path, err)
			return nil
		}
		if r.Id != "" && r.Skill != "" {
			recipes = append(recipes, r)
		}
		return nil
	})
	return recipes, err
}
