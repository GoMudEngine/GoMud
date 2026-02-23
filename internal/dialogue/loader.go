package dialogue

import (
	"fmt"
	"os"

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
	path := util.FilePath(dataFiles + `/world/dogmud/dialogue/` + sanitizedZone + `/` + fmt.Sprintf("%d", mobId) + `.yaml`)

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

	dialogueCache[key] = &df
	return &df
}
