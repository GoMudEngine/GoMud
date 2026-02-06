package migration

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

// Description:
// Migrates user character stat names from old system (speed, smarts, mysticism)
// to new system (dexterity, willpower, charisma) and adds stamina/conviction fields.
func migrate_UserStatsRename() error {

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
		// Old stat names
		Speed     StatInfo `yaml:"speed,omitempty"`
		Smarts    StatInfo `yaml:"smarts,omitempty"`
		Mysticism StatInfo `yaml:"mysticism,omitempty"`
	}

	type Character struct {
		Stats      Stats `yaml:"stats,omitempty"`
		Stamina    int   `yaml:"stamina,omitempty"`
		Conviction int   `yaml:"conviction,omitempty"`
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

	mudlog.Info("Migration 0.10.0", "message", "Migrating user files for new stat system")

	for _, userFilePath := range matches {

		// Skip the index file
		if filepath.Base(userFilePath) == "users.idx" {
			continue
		}

		mudlog.Info("Migration 0.10.0", "file", userFilePath, "message", "checking user file")

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
			mudlog.Info("Migration 0.10.0", "file", userFilePath, "message", "no character data, skipping")
			continue
		}

		// Check if stats exist
		statsData, hasStats := charData["stats"].(map[interface{}]interface{})
		if !hasStats {
			mudlog.Info("Migration 0.10.0", "file", userFilePath, "message", "no stats data, skipping")
			continue
		}

		changed := false

		// Migrate old stat names to new ones
		migrations := map[string]string{
			"speed":     "dexterity",
			"smarts":    "willpower",
			"mysticism": "charisma",
		}

		for oldName, newName := range migrations {
			if oldStat, exists := statsData[oldName]; exists {
				mudlog.Info("Migration 0.10.0", "file", userFilePath, "message", fmt.Sprintf("renaming stat %s -> %s", oldName, newName))
				statsData[newName] = oldStat
				delete(statsData, oldName)
				changed = true
			}
		}

		// Initialize stamina and conviction if missing
		if _, hasStamina := charData["stamina"]; !hasStamina {
			mudlog.Info("Migration 0.10.0", "file", userFilePath, "message", "adding stamina field (initialized to 0)")
			charData["stamina"] = 0
			changed = true
		}

		if _, hasConviction := charData["conviction"]; !hasConviction {
			mudlog.Info("Migration 0.10.0", "file", userFilePath, "message", "adding conviction field (initialized to 0)")
			charData["conviction"] = 0
			changed = true
		}

		// Save if changed
		if changed {
			mudlog.Info("Migration 0.10.0", "file", userFilePath, "message", "saving updated file")

			newData, err := yaml.Marshal(userMap)
			if err != nil {
				return fmt.Errorf("failed to marshal %s: %w", userFilePath, err)
			}

			if err := os.WriteFile(userFilePath, newData, 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", userFilePath, err)
			}
		} else {
			mudlog.Info("Migration 0.10.0", "file", userFilePath, "message", "no changes needed")
		}
	}

	mudlog.Info("Migration 0.10.0", "message", "User file migration complete!")
	return nil
}
