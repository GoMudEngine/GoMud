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
// Migrates user character files from raceid to speciesid
func migrate_RaceToSpecies() error {

	c := configs.GetConfig()

	usersGlob := filepath.Join(string(c.FilePaths.DataFiles), "users", "*.yaml")
	matches, err := filepath.Glob(usersGlob)

	if err != nil {
		return err
	}

	mudlog.Info("Migration 0.12.0", "message", "Renaming raceid to speciesid in user files")

	for _, userFilePath := range matches {

		if filepath.Base(userFilePath) == "users.idx" {
			continue
		}

		mudlog.Info("Migration 0.12.0", "file", userFilePath, "message", "checking user file")

		data, err := os.ReadFile(userFilePath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", userFilePath, err)
		}

		var userMap map[string]interface{}
		if err := yaml.Unmarshal(data, &userMap); err != nil {
			return fmt.Errorf("failed to parse %s: %w", userFilePath, err)
		}

		charData, hasChar := userMap["character"].(map[interface{}]interface{})
		if !hasChar {
			mudlog.Info("Migration 0.12.0", "file", userFilePath, "message", "no character data, skipping")
			continue
		}

		raceId, hasRaceId := charData["raceid"]
		if !hasRaceId {
			mudlog.Info("Migration 0.12.0", "file", userFilePath, "message", "no raceid found, skipping")
			continue
		}

		charData["speciesid"] = raceId
		delete(charData, "raceid")

		mudlog.Info("Migration 0.12.0", "file", userFilePath, "message", fmt.Sprintf("renamed raceid (%v) to speciesid", raceId))

		newData, err := yaml.Marshal(userMap)
		if err != nil {
			return fmt.Errorf("failed to marshal %s: %w", userFilePath, err)
		}

		if err := os.WriteFile(userFilePath, newData, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", userFilePath, err)
		}
	}

	mudlog.Info("Migration 0.12.0", "message", "Race to species migration complete!")
	return nil
}
