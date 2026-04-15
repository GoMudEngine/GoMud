package questengine_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"gopkg.in/yaml.v2"
)

// TestAllQuestStepsHaveHints verifies every QuestStep in every loaded quest
// definition has a non-empty Hint field. Missing hints mean players will see
// blank text when they use the hint command to get guidance.
func TestAllQuestStepsHaveHints(t *testing.T) {
	dataPath := configs.GetFilePathsConfig().DataFiles.String() + "/quests"

	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		t.Skipf("quest data path does not exist: %s", dataPath)
	}

	quests, err := fileloader.LoadAllFlatFiles[int, *questengine.QuestDef](dataPath)
	if err != nil {
		t.Fatalf("failed to load quest definitions: %v", err)
	}

	if len(quests) == 0 {
		t.Skip("no quest files found, skipping hint coverage check")
	}

	var violations []string
	for _, q := range quests {
		for _, step := range q.Steps {
			if step.Hint == "" {
				violations = append(violations,
					fmt.Sprintf("quest %d (%s): step %q has no hint", q.QuestId, q.Name, step.Id))
			}
		}
	}

	if len(violations) > 0 {
		t.Errorf("found %d quest steps without hints:\n  %s",
			len(violations), strings.Join(violations, "\n  "))
	}
}

// dialogueFileForTest is a minimal struct used to parse dialogue YAML files
// just enough to check text and hints fields for self-references.
type dialogueFileForTest struct {
	Tree struct {
		Root struct {
			Text  string `yaml:"text"`
			Hints string `yaml:"hints"`
		} `yaml:"root"`
		Nodes []struct {
			Text  string `yaml:"text"`
			Hints string `yaml:"hints"`
		} `yaml:"nodes"`
	} `yaml:"tree"`
	Patterns []struct {
		Text  string `yaml:"text"`
		Hints string `yaml:"hints"`
	} `yaml:"patterns"`
}

// skipWords are common words that should not be checked as name components
// when detecting self-references in dialogue.
var skipWords = map[string]bool{
	"the": true, "old": true, "a": true, "an": true, "of": true,
	"in": true, "on": true, "at": true, "to": true, "for": true,
	"and": true, "with": true,
}

// significantWords splits a name into words worth checking for self-reference.
// It skips common filler words and very short words (length <= 2).
func significantWords(name string) []string {
	parts := strings.Fields(strings.ToLower(name))
	var out []string
	for _, w := range parts {
		if len(w) > 2 && !skipWords[w] {
			out = append(out, w)
		}
	}
	return out
}

// TestDialogueSelfReference verifies no NPC dialogue file contains the NPC's
// own name in its text or hints fields. NPCs should always speak in first
// person; third-person self-references indicate copy-paste errors or broken
// voice.
func TestDialogueSelfReference(t *testing.T) {
	dataPath := configs.GetFilePathsConfig().DataFiles.String() + "/dialogue"

	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		t.Skipf("dialogue data path does not exist: %s", dataPath)
	}

	var yamlFiles []string
	err := filepath.Walk(dataPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".yaml") {
			yamlFiles = append(yamlFiles, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk dialogue directory: %v", err)
	}

	if len(yamlFiles) == 0 {
		t.Skip("no dialogue files found, skipping self-reference check")
	}

	skippedAll := true
	var violations []string

	for _, path := range yamlFiles {
		base := filepath.Base(path)
		stem := strings.TrimSuffix(base, ".yaml")
		mobId, err := strconv.Atoi(stem)
		if err != nil {
			// Filename is not a plain integer mob ID — skip silently.
			continue
		}

		mobSpec := mobs.GetMobSpec(mobs.MobId(mobId))
		if mobSpec == nil {
			// Mob spec not loaded in test environment — skip this file.
			continue
		}

		skippedAll = false
		nameWords := significantWords(mobSpec.Character.Name)
		if len(nameWords) == 0 {
			continue
		}

		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Errorf("failed to read %s: %v", path, readErr)
			continue
		}

		var df dialogueFileForTest
		if parseErr := yaml.Unmarshal(raw, &df); parseErr != nil {
			// Malformed YAML — not our concern here, skip.
			continue
		}

		// Collect all text/hints strings from the file.
		var fields []string
		fields = append(fields, df.Tree.Root.Text, df.Tree.Root.Hints)
		for _, node := range df.Tree.Nodes {
			fields = append(fields, node.Text, node.Hints)
		}
		for _, pat := range df.Patterns {
			fields = append(fields, pat.Text, pat.Hints)
		}

		for _, word := range nameWords {
			for _, field := range fields {
				if field == "" {
					continue
				}
				if strings.Contains(strings.ToLower(field), word) {
					violations = append(violations,
						fmt.Sprintf("%s (mob %d, name %q): contains own name word %q in text/hints",
							path, mobId, mobSpec.Character.Name, word))
					break
				}
			}
		}
	}

	if skippedAll {
		t.Skip("mob specs not loaded in test environment; skipping dialogue self-reference check")
	}

	if len(violations) > 0 {
		t.Errorf("found %d dialogue self-reference violations:\n  %s",
			len(violations), strings.Join(violations, "\n  "))
	}
}
