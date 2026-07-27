package questengine

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dialogue"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/quests"
	"gopkg.in/yaml.v2"
)

// Package-level engine instance
var globalEngine *Engine

// GetEngine returns the global quest engine instance.
func GetEngine() *Engine {
	if globalEngine == nil {
		globalEngine = NewEngine()
	}
	return globalEngine
}

// LoadDataFiles builds the trigger index from the quest definitions already
// loaded by quests.LoadDataFiles (main.go boot order guarantees it ran first).
// Since the 5c-pre unification this package performs NO file I/O — the quest
// file parse (and its Validate()-at-load enforcement, plus flag registration)
// lives entirely in internal/quests.
func LoadDataFiles() {
	start := time.Now()

	globalEngine = NewEngine()
	all := quests.GetAllQuests()
	for i := range all {
		globalEngine.RegisterQuest(&all[i])
	}

	ValidateAllFlags()

	mudlog.Info("questengine.LoadDataFiles()", "loadedCount", len(globalEngine.quests), "Time Taken", time.Since(start))
}

// dialogueEntry pairs a parsed DialogueFile with the mob ID from the filename.
type dialogueEntry struct {
	mobId int
	df    *dialogue.DialogueFile
}

// loadAllDialogueFiles walks the dialogue directory and parses every YAML file.
func loadAllDialogueFiles(basePath string) []dialogueEntry {
	var entries []dialogueEntry

	zoneDirs, err := os.ReadDir(basePath)
	if err != nil {
		// No dialogue directory is fine (e.g., tests).
		return entries
	}

	for _, zoneDir := range zoneDirs {
		if !zoneDir.IsDir() {
			continue
		}
		zonePath := basePath + "/" + zoneDir.Name()
		files, err := os.ReadDir(zonePath)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".yaml") {
				continue
			}
			fullPath := zonePath + "/" + f.Name()
			data, err := os.ReadFile(fullPath)
			if err != nil {
				continue
			}
			var df dialogue.DialogueFile
			if err := yaml.Unmarshal(data, &df); err != nil {
				continue
			}
			entries = append(entries, dialogueEntry{mobId: df.MobId, df: &df})
		}
	}

	return entries
}

// ValidateAllFlags scans all quest engine triggers and dialogue files for quest
// flag references and panics if any reference an undeclared flag key or value.
// This runs at startup so typos are caught before they reach players.
func ValidateAllFlags() {
	var errors []string

	// Scan quest engine triggers for flag references.
	for _, q := range globalEngine.quests {
		for i, t := range q.Triggers {
			for k, v := range t.Conditions.HasFlag {
				if err := quests.ValidateFlag(k, v); err != nil {
					errors = append(errors, fmt.Sprintf("quest %d trigger %d has_flag: %s", q.QuestId, i, err))
				}
			}
			for k, v := range t.Conditions.MissingFlag {
				if err := quests.ValidateFlag(k, v); err != nil {
					errors = append(errors, fmt.Sprintf("quest %d trigger %d missing_flag: %s", q.QuestId, i, err))
				}
			}
			for j, a := range t.Actions {
				if a.SetFlag != nil {
					if err := quests.ValidateFlag(a.SetFlag.Key, a.SetFlag.Value); err != nil {
						errors = append(errors, fmt.Sprintf("quest %d trigger %d action %d set_flag: %s", q.QuestId, i, j, err))
					}
				}
			}
		}
	}

	// Scan dialogue files for flag references.
	dialoguePath := configs.GetFilePathsConfig().DataFiles.String() + `/dialogue`
	dialogueFiles := loadAllDialogueFiles(dialoguePath)
	for _, entry := range dialogueFiles {
		refs, sets := dialogue.CollectFlagReferences(entry.df)
		for _, ref := range refs {
			if err := quests.ValidateFlag(ref.Key, ref.Value); err != nil {
				errors = append(errors, fmt.Sprintf("dialogue mob %d %s: %s", entry.mobId, ref.Source, err))
			}
		}
		for _, set := range sets {
			if err := quests.ValidateFlag(set.Key, set.Value); err != nil {
				errors = append(errors, fmt.Sprintf("dialogue mob %d %s setsQuestFlag: %s", entry.mobId, set.Source, err))
			}
		}
	}

	if len(errors) > 0 {
		panic(fmt.Sprintf("Quest flag validation failed (%d errors):\n  %s", len(errors), strings.Join(errors, "\n  ")))
	}

	mudlog.Info("ValidateAllFlags()", "msg", "all quest flag references validated")
}
