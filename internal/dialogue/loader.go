package dialogue

import (
	"fmt"
	"os"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/conversations"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

var (
	dialogueCache = map[string]*DialogueFile{}
	nilSentinel   = map[string]bool{}
)

// Load lazily reads and caches a mob's dialogue file.
// Returns nil if no file exists for this mob/zone combination.
func Load(mobId int, zone string) *DialogueFile {
	key := fmt.Sprintf("%d:%s", mobId, zone)

	if nilSentinel[key] {
		return nil
	}

	if df, ok := dialogueCache[key]; ok {
		return df
	}

	sanitizedZone := conversations.ZoneNameSanitize(zone)
	dataFiles := string(configs.GetFilePathsConfig().DataFiles)
	path := util.FilePath(dataFiles + `/dialogue/` + sanitizedZone + `/` + fmt.Sprintf("%d", mobId) + `.yaml`)

	if _, err := os.Stat(path); err != nil {
		nilSentinel[key] = true
		return nil
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		mudlog.Error("dialogue.Load()", "error", "Problem reading dialogue file "+path+": "+err.Error())
		nilSentinel[key] = true
		return nil
	}

	var df DialogueFile
	if err := yaml.Unmarshal(bytes, &df); err != nil {
		mudlog.Error("dialogue.Load()", "error", "Problem unmarshalling dialogue file "+path+": "+err.Error())
		nilSentinel[key] = true
		return nil
	}

	validateQuestExclusions(&df, path)

	dialogueCache[key] = &df
	return &df
}

// validateQuestExclusions warns if any grantsQuest node is missing the
// quest's end token from questExcluded. Without the end-token exclusion,
// players who have completed a quest can get it re-offered.
func validateQuestExclusions(df *DialogueFile, path string) {
	checkExclusions := func(label string, grantsQuest string, questExcluded []string) {
		if grantsQuest == "" {
			return
		}
		// Extract quest ID prefix: "10-start" → "10", "14-evidence" → "14"
		idx := strings.Index(grantsQuest, "-")
		if idx < 0 {
			return
		}
		prefix := grantsQuest[:idx]
		endToken := prefix + "-end"

		// If it already grants the end token, no exclusion needed
		if grantsQuest == endToken {
			return
		}

		for _, ex := range questExcluded {
			if ex == endToken {
				return
			}
		}
		mudlog.Warn("dialogue.validateQuestExclusions()", "warning",
			fmt.Sprintf("%s: %s grants %q but questExcluded is missing %q — completed quest can be re-offered",
				path, label, grantsQuest, endToken))
	}

	// Check patterns
	for i, p := range df.Patterns {
		checkExclusions(fmt.Sprintf("pattern[%d]", i), p.GrantsQuest, p.QuestExcluded)
	}

	// Check tree nodes
	if df.Tree != nil {
		for _, n := range df.Tree.Nodes {
			checkExclusions(fmt.Sprintf("node[%s]", n.Id), n.GrantsQuest, n.QuestExcluded)
		}
	}
}
