package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

// Description:
// One-time exploit remediation. Player fyttyn ground raw vitality to 411 via
// a since-fixed exploit; under the (now-removed) stat soft cap that raw value
// was being silently compressed to an effective 280. Removing the soft cap
// elsewhere in this release would otherwise hand fyttyn back the 131 points
// that were never legitimately earned. This migration freezes fyttyn's raw
// vitality at the value that was actually in play instead.
//
// Scoped to fyttyn ONLY. Other players who ended up above the old 150 soft
// cap (Duard, pruuk, Deios) earned those stats legitimately — the compression
// was a bug taxing them, so removing it correctly hands them back their small
// gain, and this migration must not touch their saves.
//
// Idempotent: a save whose vitality total is already at or below the frozen
// target is left untouched, so re-running (or running against an
// already-migrated save) is a no-op. A tree with no fyttyn account (e.g. a
// local dev tree) is not an error.
func migrate_FreezeExploitedVitality(dryRun bool) error {
	c := configs.GetConfig()
	return freezeExploitedVitalityInDir(filepath.Join(string(c.FilePaths.DataFiles), "users"), dryRun)
}

// targetVitalityTotal is the frozen total: fyttyn's effective vitality under
// the old soft cap (raw base 85 + training 326 = 411, compressed to 280).
const targetVitalityTotal = 280

// freezeExploitedVitalityInDir is the testable core — operates on any users
// directory so tests can run it against disposable fixtures rather than the
// real (gitignored) datafiles tree.
func freezeExploitedVitalityInDir(usersDir string, dryRun bool) error {
	mode := "APPLY"
	if dryRun {
		mode = "DRY-RUN"
	}

	matches, err := filepath.Glob(filepath.Join(usersDir, "*.yaml"))
	if err != nil {
		return err
	}

	mudlog.Info("Migration 0.16.0", "message", "Freezing fyttyn's exploited vitality", "mode", mode)

	found := false
	for _, path := range matches {
		if filepath.Base(path) == "users.idx" {
			continue
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		// Parse as a generic map to preserve every other field untouched,
		// mirroring migrate_UserStatsRename (0.10.0) and
		// migrate_ReclassifyPlayerMutations (0.14.0).
		var userMap map[string]interface{}
		if err := yaml.Unmarshal(raw, &userMap); err != nil {
			mudlog.Warn("Migration 0.16.0", "file", filepath.Base(path), "error", err)
			continue
		}

		username, _ := userMap["username"].(string)
		if !strings.EqualFold(username, "fyttyn") {
			continue
		}
		found = true

		charData, ok := userMap["character"].(map[interface{}]interface{})
		if !ok {
			mudlog.Warn("Migration 0.16.0", "file", filepath.Base(path), "message", "fyttyn save has no character data, skipping")
			continue
		}
		statsData, ok := charData["stats"].(map[interface{}]interface{})
		if !ok {
			mudlog.Warn("Migration 0.16.0", "file", filepath.Base(path), "message", "fyttyn save has no stats data, skipping")
			continue
		}
		vitalityData, ok := statsData["vitality"].(map[interface{}]interface{})
		if !ok {
			mudlog.Warn("Migration 0.16.0", "file", filepath.Base(path), "message", "fyttyn save has no vitality stat, skipping")
			continue
		}

		base, _ := vitalityData["base"].(int)
		training, _ := vitalityData["training"].(int)
		total := base + training

		if total <= targetVitalityTotal {
			mudlog.Info("Migration 0.16.0", "file", filepath.Base(path), "username", username,
				"total", total, "target", targetVitalityTotal,
				"message", "vitality already at or below frozen target, skipping")
			continue
		}

		newTraining := targetVitalityTotal - base
		if newTraining < 0 {
			mudlog.Error("Migration 0.16.0", "file", filepath.Base(path), "username", username,
				"base", base, "target", targetVitalityTotal,
				"message", "base alone exceeds frozen target; leaving file untouched to avoid a negative training value")
			continue
		}

		mudlog.Info("Migration 0.16.0", "file", filepath.Base(path), "username", username,
			"oldTotal", total, "newTotal", targetVitalityTotal, "mode", mode,
			"message", "freezing exploited vitality")

		if dryRun {
			continue
		}

		vitalityData["training"] = newTraining

		out, err := yaml.Marshal(userMap)
		if err != nil {
			return fmt.Errorf("failed to marshal %s: %w", path, err)
		}
		if err := os.WriteFile(path, out, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}
	}

	if !found {
		mudlog.Info("Migration 0.16.0", "message", "no fyttyn account found in this datafiles tree — nothing to do")
	}

	return nil
}
