package migration

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

// Description:
// Migrates user character stats from fixed racial base (1) to rolled stats (70-130)
// This gives existing characters properly scaled stats with mean 100, stdDev 15
func migrate_RollCharacterStats() error {

	// User file structure (minimal, just what we need)
	type StatInfo struct {
		Training int `yaml:"training,omitempty"`
		Base     int `yaml:"base,omitempty"`
	}

	type Stats struct {
		Strength   StatInfo `yaml:"strength,omitempty"`
		Dexterity  StatInfo `yaml:"dexterity,omitempty"`
		Perception StatInfo `yaml:"perception,omitempty"`
		Vitality   StatInfo `yaml:"vitality,omitempty"`
		Willpower  StatInfo `yaml:"willpower,omitempty"`
		Charisma   StatInfo `yaml:"charisma,omitempty"`
	}

	type Character struct {
		Stats      Stats `yaml:"stats,omitempty"`
		Health     int   `yaml:"health,omitempty"`
		Mana       int   `yaml:"mana,omitempty"`
		Stamina    int   `yaml:"stamina,omitempty"`
		Conviction int   `yaml:"conviction,omitempty"`
		Level      int   `yaml:"level,omitempty"`
	}

	type UserFile struct {
		Character Character `yaml:"character,omitempty"`
	}

	c := configs.GetConfig()

	usersGlob := filepath.Join(string(c.FilePaths.DataFiles), "users", "*.yaml")
	matches, err := filepath.Glob(usersGlob)

	if err != nil {
		return err
	}

	mudlog.Info("Migration 0.11.0", "message", "Rolling stats for existing characters")

	const (
		statMean   = 100.0
		statStdDev = 15.0
		statMin    = 70.0
		statMax    = 130.0
	)

	for _, userFilePath := range matches {

		// Skip the index file
		if filepath.Base(userFilePath) == "users.idx" {
			continue
		}

		mudlog.Info("Migration 0.11.0", "file", userFilePath, "message", "checking user file")

		// Read the user file
		data, err := os.ReadFile(userFilePath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", userFilePath, err)
		}

		// Parse as generic map to preserve all fields
		var userMap map[string]interface{}
		if err := yaml.Unmarshal(data, &userMap); err != nil {
			return fmt.Errorf("failed to parse %s: %w", userFilePath, err)
		}

		// Check if character exists
		charData, hasChar := userMap["character"].(map[interface{}]interface{})
		if !hasChar {
			mudlog.Info("Migration 0.11.0", "file", userFilePath, "message", "no character data, skipping")
			continue
		}

		// Check if stats exist
		statsData, hasStats := charData["stats"].(map[interface{}]interface{})
		if !hasStats {
			mudlog.Info("Migration 0.11.0", "file", userFilePath, "message", "no stats data, skipping")
			continue
		}

		changed := false

		// Check if this character needs stat rolling (base stats are still 1 or very low)
		// We'll check if all stats are below 10, which indicates they haven't been rolled yet
		needsRolling := true
		statNames := []string{"strength", "dexterity", "perception", "vitality", "willpower", "charisma"}

		for _, statName := range statNames {
			if statInfo, exists := statsData[statName].(map[interface{}]interface{}); exists {
				if baseVal, hasBase := statInfo["base"]; hasBase {
					if base, ok := baseVal.(int); ok && base >= 10 {
						needsRolling = false
						break
					}
				}
			}
		}

		if !needsRolling {
			mudlog.Info("Migration 0.11.0", "file", userFilePath, "message", "stats already rolled, skipping")
			continue
		}

		mudlog.Info("Migration 0.11.0", "file", userFilePath, "message", "rolling new stats")

		// Roll 6 stats
		rolledStats := dice.RollStatArray(6, statMean, statStdDev, statMin, statMax)

		// Update each stat with rolled base value
		statNamesWithRolls := map[string]int{
			"strength":   rolledStats[0],
			"dexterity":  rolledStats[1],
			"perception": rolledStats[2],
			"vitality":   rolledStats[3],
			"willpower":  rolledStats[4],
			"charisma":   rolledStats[5],
		}

		for statName, rolledValue := range statNamesWithRolls {
			if statInfo, exists := statsData[statName].(map[interface{}]interface{}); exists {
				oldBase := 1
				if baseVal, hasBase := statInfo["base"]; hasBase {
					if base, ok := baseVal.(int); ok {
						oldBase = base
					}
				}
				statInfo["base"] = rolledValue
				mudlog.Info("Migration 0.11.0", "file", userFilePath, "stat", statName,
					"message", fmt.Sprintf("base: %d -> %d", oldBase, rolledValue))
				changed = true
			} else {
				// Stat doesn't exist, create it
				statsData[statName] = map[interface{}]interface{}{
					"base": rolledValue,
				}
				mudlog.Info("Migration 0.11.0", "file", userFilePath, "stat", statName,
					"message", fmt.Sprintf("created with base: %d", rolledValue))
				changed = true
			}
		}

		// Note: Health/Mana/Stamina/Conviction will be recalculated when the user logs in
		// because Validate() is called which triggers RecalculateStats()

		// Save if changed
		if changed {
			mudlog.Info("Migration 0.11.0", "file", userFilePath, "message", "saving updated file")

			newData, err := yaml.Marshal(userMap)
			if err != nil {
				return fmt.Errorf("failed to marshal %s: %w", userFilePath, err)
			}

			if err := os.WriteFile(userFilePath, newData, 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", userFilePath, err)
			}
		} else {
			mudlog.Info("Migration 0.11.0", "file", userFilePath, "message", "no changes needed")
		}
	}

	mudlog.Info("Migration 0.11.0", "message", "Character stat migration complete!")
	return nil
}
