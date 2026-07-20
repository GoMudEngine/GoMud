package users

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/quests"
	"github.com/GoMudEngine/GoMud/internal/util"
)

var (
	ErrBothFilesExist = errors.New("could not migrate due to both file formats existing")
)

//
// Handles user file migration
//

func DoUserMigrations() {

	DoFilenameMigrationV1()
	DoQuestStepMigration()

}

// DoQuestStepMigration checks all players for quest progress on steps that
// no longer exist (removed during quest engine port). Players stuck on
// deleted steps are bumped to "start" so they can redo the quest cleanly.
func DoQuestStepMigration() {
	migrated := 0

	SearchOfflineUsers(func(u *UserRecord) bool {
		if u.Character.QuestProgress == nil {
			return true
		}

		changed := false
		for questId, currentStep := range u.Character.QuestProgress {
			q := quests.GetQuest(quests.PartsToToken(questId, currentStep))
			if q == nil {
				// Quest itself was deleted (e.g., Q17) — remove progress
				delete(u.Character.QuestProgress, questId)
				mudlog.Info("DoQuestStepMigration", "info",
					"Removed deleted quest progress",
					"user", u.Username, "questId", questId, "step", currentStep)
				changed = true
				continue
			}

			// Check if the step still exists in the quest definition
			stepExists := false
			for _, step := range q.Steps {
				if step.Id == currentStep {
					stepExists = true
					break
				}
			}

			if !stepExists {
				// Step was removed — reset to "start"
				u.Character.QuestProgress[questId] = "start"
				mudlog.Info("DoQuestStepMigration", "info",
					"Migrated removed quest step to start",
					"user", u.Username, "questId", questId,
					"oldStep", currentStep, "newStep", "start")
				changed = true
			}
		}

		if changed {
			migrated++
			SaveUser(u)
		}
		return true
	})

	if migrated > 0 {
		mudlog.Info("DoQuestStepMigration", "info",
			"Quest step migration complete", "usersUpdated", migrated)
	}
}

func DoFilenameMigrationV1() error {

	var errorResult error = nil

	SearchOfflineUsers(func(u *UserRecord) bool {

		oldUserPath := util.FilePath(string(configs.GetFilePathsConfig().DataFiles), `/`, `users`, `/`, strings.ToLower(u.Username)+`.yaml`)
		newUserPath := util.FilePath(string(configs.GetFilePathsConfig().DataFiles), `/`, `users`, `/`, strconv.Itoa(u.UserId)+`.yaml`)

		_, err := os.Stat(oldUserPath)
		oldUserPathExists := err == nil

		_, err = os.Stat(newUserPath)
		newUserPathExists := err == nil

		if oldUserPathExists && newUserPathExists {
			errorResult = ErrBothFilesExist
			return false
		}

		if oldUserPathExists {
			err := os.Rename(oldUserPath, newUserPath)
			if err != nil {
				errorResult = err
				return false
			}
		}

		oldAltsFilePath := util.FilePath(string(configs.GetFilePathsConfig().DataFiles), `/users/`, strings.ToLower(u.Username)+`-alts.yaml`)
		newAltsFilePath := util.FilePath(string(configs.GetFilePathsConfig().DataFiles), `/users/`, strconv.Itoa(u.UserId)+`.alts.yaml`)

		_, err = os.Stat(oldAltsFilePath)
		oldAltsPathExists := err == nil

		_, err = os.Stat(newAltsFilePath)
		newAltsPathExists := err == nil

		if oldAltsPathExists && newAltsPathExists {
			errorResult = ErrBothFilesExist
			return false
		}

		if oldAltsPathExists {
			err := os.Rename(oldAltsFilePath, newAltsFilePath)
			if err != nil {
				errorResult = err
				return false
			}
		}

		return true
	})

	return errorResult
}
