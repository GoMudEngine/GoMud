package questengine

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dialogue"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/quests"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// Id implements fileloader.Loadable.
func (q *QuestDef) Id() int { return q.QuestId }

// Filepath implements fileloader.Loadable.
func (q *QuestDef) Filepath() string {
	return util.FilePath(fmt.Sprintf("%d-%s.yaml", q.QuestId, util.ConvertForFilename(q.Name)))
}

// Validate implements fileloader.Loadable.
func (q *QuestDef) Validate() error {
	if q.QuestId < 1 {
		return fmt.Errorf("quest id must be > 0")
	}
	if q.Name == "" {
		return fmt.Errorf("quest %d: name cannot be empty", q.QuestId)
	}
	if len(q.Steps) == 0 {
		return fmt.Errorf("quest %d (%s): must have at least one step", q.QuestId, q.Name)
	}

	validEvents := map[string]bool{
		"room_enter": true, "item_give": true, "skill_use": true,
		"mob_death": true, "command": true, "item_gain": true,
		"dialogue": true, "quest_granted": true, "room_interact": true,
		"command_issued": true,
	}

	for i, t := range q.Triggers {
		if t.Event == "" {
			return fmt.Errorf("quest %d (%s): trigger %d has no event", q.QuestId, q.Name, i)
		}
		if !validEvents[t.Event] {
			return fmt.Errorf("quest %d (%s): trigger %d has invalid event %q", q.QuestId, q.Name, i, t.Event)
		}
		if len(t.Actions) == 0 {
			return fmt.Errorf("quest %d (%s): trigger %d has no actions", q.QuestId, q.Name, i)
		}
	}

	// Check for duplicate step IDs
	seen := make(map[string]bool)
	for _, s := range q.Steps {
		if s.Id == "" {
			return fmt.Errorf("quest %d (%s): step has empty id", q.QuestId, q.Name)
		}
		if seen[s.Id] {
			return fmt.Errorf("quest %d (%s): duplicate step id %q", q.QuestId, q.Name, s.Id)
		}
		seen[s.Id] = true
	}

	// Validate flag declarations
	flagKeys := make(map[string]bool)
	for _, f := range q.Flags {
		if f.Key == "" {
			return fmt.Errorf("quest %d (%s): flag has empty key", q.QuestId, q.Name)
		}
		if len(f.Values) == 0 {
			return fmt.Errorf("quest %d (%s): flag %q has no allowed values", q.QuestId, q.Name, f.Key)
		}
		fullKey := fmt.Sprintf("%d-%s", q.QuestId, f.Key)
		if flagKeys[fullKey] {
			return fmt.Errorf("quest %d (%s): duplicate flag key %q", q.QuestId, q.Name, f.Key)
		}
		flagKeys[fullKey] = true
	}

	// Validate grant tokens reference valid steps (only for this quest's own tokens)
	for i, t := range q.Triggers {
		for _, a := range t.Actions {
			if a.Grant != "" {
				parts := strings.SplitN(a.Grant, "-", 2)
				if len(parts) == 2 {
					// Only validate if the grant is for THIS quest
					questIdStr := fmt.Sprintf("%d", q.QuestId)
					if parts[0] == questIdStr {
						stepId := parts[1]
						if !seen[stepId] {
							return fmt.Errorf("quest %d (%s): trigger %d grants unknown step %q",
								q.QuestId, q.Name, i, a.Grant)
						}
					}
				}
			}
		}
	}

	return nil
}

// Package-level engine instance
var globalEngine *Engine

// GetEngine returns the global quest engine instance.
func GetEngine() *Engine {
	if globalEngine == nil {
		globalEngine = NewEngine()
	}
	return globalEngine
}

// LoadDataFiles reads all expanded quest YAML files and populates the global engine.
func LoadDataFiles() {
	start := time.Now()

	dataPath := configs.GetFilePathsConfig().DataFiles.String() + `/quests`
	tmpQuests, err := fileloader.LoadAllFlatFiles[int, *QuestDef](dataPath)
	if err != nil {
		panic(errors.Wrap(err, `filepath: `+dataPath))
	}

	globalEngine = NewEngine()
	for _, q := range tmpQuests {
		globalEngine.RegisterQuest(q)
	}

	for _, q := range tmpQuests {
		for _, f := range q.Flags {
			quests.RegisterFlags(q.QuestId, []quests.QuestFlagDef{{Key: f.Key, Values: f.Values, Description: f.Description}})
		}
	}

	ValidateAllFlags()

	mudlog.Info("questengine.LoadDataFiles()", "loadedCount", len(tmpQuests), "Time Taken", time.Since(start))
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
