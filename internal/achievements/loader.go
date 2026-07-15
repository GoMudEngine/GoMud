package achievements

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

var validStats = map[string]bool{
	"strength": true, "dexterity": true, "perception": true,
	"vitality": true, "willpower": true, "charisma": true, "any": true,
}

// thresholdTypes need a positive threshold (as opposed to a stat/skill/token param).
var thresholdTypes = map[string]bool{
	"mob_kills": true, "pvp_kills": true, "deaths": true, "gold_total": true,
	"rooms_explored": true, "quests_completed": true, "mutation_count": true,
	"item_rarity": true, "achievement_points": true,
}

// allTriggerTypes is the full fixed vocabulary.
var allTriggerTypes = func() map[string]bool {
	m := map[string]bool{"stat_reached": true, "skill_reached": true, "quest_completed": true}
	for k := range thresholdTypes {
		m[k] = true
	}
	return m
}()

// validateDefinition enforces the achievement schema; the loader panics on any error.
func validateDefinition(d Definition, fileBase string) error {
	if d.Id == "" {
		return fmt.Errorf("achievement in %q has no id", fileBase)
	}
	if d.Id != fileBase {
		return fmt.Errorf("achievement %q: filename base %q must equal id", d.Id, fileBase)
	}
	if d.Name == "" {
		return fmt.Errorf("achievement %q: missing name", d.Id)
	}
	if !validCategories[d.Category] {
		return fmt.Errorf("achievement %q: invalid category %q", d.Id, d.Category)
	}
	if d.Points < 0 {
		return fmt.Errorf("achievement %q: points must be >= 0", d.Id)
	}
	if !allTriggerTypes[d.Trigger.Type] {
		return fmt.Errorf("achievement %q: unknown trigger type %q", d.Id, d.Trigger.Type)
	}
	switch d.Trigger.Type {
	case "stat_reached":
		if !validStats[d.Trigger.Stat] {
			return fmt.Errorf("achievement %q: stat_reached needs a valid stat (or 'any'), got %q", d.Id, d.Trigger.Stat)
		}
		if d.Trigger.Threshold <= 0 {
			return fmt.Errorf("achievement %q: stat_reached needs threshold > 0", d.Id)
		}
	case "skill_reached":
		if d.Trigger.Skill == "" {
			return fmt.Errorf("achievement %q: skill_reached needs a skill (or 'any')", d.Id)
		}
		if d.Trigger.Threshold <= 0 {
			return fmt.Errorf("achievement %q: skill_reached needs threshold > 0", d.Id)
		}
	case "quest_completed":
		if d.Trigger.Token == "" {
			return fmt.Errorf("achievement %q: quest_completed needs a token", d.Id)
		}
	default: // threshold types
		if d.Trigger.Threshold <= 0 {
			return fmt.Errorf("achievement %q: %s needs threshold > 0", d.Id, d.Trigger.Type)
		}
	}
	return nil
}

// LoadDataFiles loads and validates every achievement YAML under
// <DataFiles>/achievements. It PANICS on any malformed definition so the pre-push
// boot test catches it. A missing directory is not an error (no achievements yet).
func LoadDataFiles() {
	start := time.Now()

	registry = map[string]Definition{}
	registryOrder = nil

	dataPath := filepath.FromSlash(configs.GetFilePathsConfig().DataFiles.String() + `/achievements`)

	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		mudlog.Info("achievements.LoadDataFiles()", "loadedCount", 0, "note", "no achievements dir")
		return
	}

	err := filepath.Walk(dataPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}
		raw, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		var d Definition
		if uerr := yaml.Unmarshal(raw, &d); uerr != nil {
			return fmt.Errorf("achievement %s: %w", path, uerr)
		}
		base := strings.TrimSuffix(filepath.Base(path), ".yaml")
		if verr := validateDefinition(d, base); verr != nil {
			return verr
		}
		if _, dup := registry[d.Id]; dup {
			return fmt.Errorf("duplicate achievement id %q (%s)", d.Id, path)
		}
		registry[d.Id] = d
		registryOrder = append(registryOrder, d.Id)
		return nil
	})
	if err != nil {
		panic(fmt.Errorf("achievements.LoadDataFiles: %w", err))
	}

	mudlog.Info("achievements.LoadDataFiles()", "loadedCount", len(registry), "Time Taken", time.Since(start))
}
