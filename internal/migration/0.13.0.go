package migration

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

// Description:
// Seeds warren faction rep for any user holding the legacy "2-end"
// quest token (The Warren Compact). Without this migration, players
// who completed quest 2 pre-deploy would lose their hostility
// immunity from warren mobs after the peacefulquest field is removed
// in chunk 1.2.
//
// Idempotent — only seeds rep when the user has no existing entry
// for the warren faction (rep == default_rep, which is -25 for the
// warren definition shipped in T9).
func migrate_SeedWarrenRepFromQuestToken() error {

	c := configs.GetConfig()

	usersGlob := filepath.Join(string(c.FilePaths.DataFiles), "users", "*.yaml")
	matches, err := filepath.Glob(usersGlob)
	if err != nil {
		return err
	}

	mudlog.Info("Migration 0.13.0", "message", "Seeding warren faction rep from legacy 2-end quest token")

	warrenDef := factions.GetDefinition("warren")
	if warrenDef == nil {
		// LoadAllDefinitions wasn't called yet, or warren faction
		// wasn't authored. Nothing to do.
		mudlog.Warn("Migration 0.13.0", "message", "warren faction not loaded; skipping")
		return nil
	}

	for _, userFilePath := range matches {
		if filepath.Base(userFilePath) == "users.idx" {
			continue
		}

		mudlog.Info("Migration 0.13.0", "file", userFilePath, "message", "checking user file")

		data, err := os.ReadFile(userFilePath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", userFilePath, err)
		}

		var userMap map[string]interface{}
		if err := yaml.Unmarshal(data, &userMap); err != nil {
			return fmt.Errorf("failed to parse %s: %w", userFilePath, err)
		}

		// Top-level userid.
		userIdRaw, hasUserId := userMap["userid"]
		if !hasUserId {
			mudlog.Info("Migration 0.13.0", "file", userFilePath, "message", "no userid, skipping")
			continue
		}
		userId, ok := userIdRaw.(int)
		if !ok {
			mudlog.Warn("Migration 0.13.0", "file", userFilePath, "message", "userid not an int, skipping")
			continue
		}
		if userId <= 0 {
			continue
		}

		charData, hasChar := userMap["character"].(map[interface{}]interface{})
		if !hasChar {
			mudlog.Info("Migration 0.13.0", "file", userFilePath, "message", "no character data, skipping")
			continue
		}

		questProgressRaw, hasQP := charData["questprogress"].(map[interface{}]interface{})
		if !hasQP {
			mudlog.Info("Migration 0.13.0", "file", userFilePath, "message", "no questprogress, skipping")
			continue
		}

		// Look for quest 2 with step "end".
		stepRaw, hasQuest2 := questProgressRaw[2]
		if !hasQuest2 {
			continue
		}
		step, ok := stepRaw.(string)
		if !ok || step != "end" {
			continue
		}

		// Idempotency: only seed if the user has no rep entry yet.
		// GetRep returns the faction's DefaultRep when no row exists.
		if existing := factions.GetRep("warren", userId); existing != warrenDef.DefaultRep {
			mudlog.Info("Migration 0.13.0", "file", userFilePath,
				"message", fmt.Sprintf("warren rep already at %d, skipping seed", existing))
			continue
		}

		factions.BumpRep("warren", userId, 50)
		mudlog.Info("Migration 0.13.0", "file", userFilePath,
			"message", fmt.Sprintf("seeded warren rep +50 for userId %d", userId))
	}

	mudlog.Info("Migration 0.13.0", "message", "Warren rep seeding complete!")
	return nil
}
