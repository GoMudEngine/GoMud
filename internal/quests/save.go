package quests

// The quest writer (admin web-building 5c). Everything mutating here runs on
// MainWorker (GMCPBuildOp), matching the mob writer's threading model — the
// package deliberately carries no mutex. Marshal output is the proven fixed
// point (roundtrip_test.go), so repeat saves diff minimally; the FIRST save
// of a hand-authored file produces a one-time canonicalization diff.

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"gopkg.in/yaml.v2"
)

// questsDataRoot is a test seam (the mobs.mobsDataRoot pattern): overriding
// the package var redirects file I/O without touching the real config.
var questsDataRoot = func() string {
	return configs.GetFilePathsConfig().DataFiles.String() + `/quests`
}

func flagKey(questId int, key string) string {
	return fmt.Sprintf("%d-%s", questId, key)
}

// SaveQuest validates and writes a quest definition, updating the cache and
// flag registry. Renames move the file — the old path is computed from the
// CACHED entry before the swap, so a rename can't strand a duplicate that
// would boot as a second copy.
func SaveQuest(q Quest) error {
	if err := q.Validate(); err != nil {
		return err
	}
	out, err := yaml.Marshal(q)
	if err != nil {
		return err
	}

	dir := questsDataRoot()
	oldPath := ""
	var oldFlags []QuestFlagDef
	if prev, ok := quests[q.QuestId]; ok {
		oldPath = filepath.Join(dir, prev.Filename())
		oldFlags = prev.Flags
	}
	newPath := filepath.Join(dir, q.Filename())

	if err := os.WriteFile(newPath, out, 0644); err != nil {
		return err
	}
	if oldPath != "" && oldPath != newPath {
		if err := os.Remove(oldPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	// Swap cache + re-register flags (dropping declarations the new
	// definition no longer carries).
	for _, f := range oldFlags {
		delete(flagRegistry, flagKey(q.QuestId, f.Key))
	}
	cp := q
	quests[q.QuestId] = &cp
	RegisterFlags(q.QuestId, q.Flags)
	return nil
}

// CreateNewQuestFile allocates the next free id past the cache max and
// persists a minimal boot-safe skeleton (start + end steps, no triggers).
func CreateNewQuestFile(name string) (int, error) {
	if name == "" {
		name = "New Quest"
	}
	id := 0
	for qid := range quests {
		if qid > id {
			id = qid
		}
	}
	id++

	q := Quest{
		QuestId:     id,
		Name:        name,
		Description: "An unfinished quest.",
		Steps: []QuestStep{
			{Id: "start", Description: "This quest has begun."},
			{Id: "end", Description: "This quest is complete."},
		},
	}
	if err := SaveQuest(q); err != nil {
		return 0, err
	}
	return id, nil
}

// DeleteQuest removes the quest's file, cache entry, and flag registrations.
// Reference guarding (dialogue files, other quests) is the caller's job —
// the gmcp layer refuses before ever calling this. Idempotent when the file
// is already gone.
func DeleteQuest(questId int) error {
	prev, ok := quests[questId]
	if !ok {
		return fmt.Errorf("quest %d is not loaded", questId)
	}
	p := filepath.Join(questsDataRoot(), prev.Filename())
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, f := range prev.Flags {
		delete(flagRegistry, flagKey(questId, f.Key))
	}
	delete(quests, questId)
	return nil
}
