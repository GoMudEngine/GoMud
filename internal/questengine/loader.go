package questengine

import (
	"fmt"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/pkg/errors"
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

	mudlog.Info("questengine.LoadDataFiles()", "loadedCount", len(tmpQuests), "Time Taken", time.Since(start))
}
