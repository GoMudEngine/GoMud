package dialogue

// WalkAllFiles reads every dialogue file from disk (5c: the quest editor's
// reference guard and grant-token index need the whole set). It deliberately
// bypasses dialogueCache/nilSentinel — a bulk read-only sweep must not
// populate (or clobber) the lazy per-mob cache the live game maintains.

import (
	"os"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"gopkg.in/yaml.v2"
)

// WalkAllFiles calls fn for every parseable dialogue file under
// {DataFiles}/dialogue/<zone>/<mobId>.yaml. Unparseable files are skipped
// (they are mute in-game anyway; the strict sweep owns reporting them).
func WalkAllFiles(fn func(mobId int, zone string, df *DialogueFile)) {
	basePath := string(configs.GetFilePathsConfig().DataFiles) + `/dialogue`

	zoneDirs, err := os.ReadDir(basePath)
	if err != nil {
		return // no dialogue directory is fine (e.g. tests)
	}
	for _, zoneDir := range zoneDirs {
		if !zoneDir.IsDir() {
			continue
		}
		zonePath := basePath + `/` + zoneDir.Name()
		files, err := os.ReadDir(zonePath)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), `.yaml`) {
				continue
			}
			data, err := os.ReadFile(zonePath + `/` + f.Name())
			if err != nil {
				continue
			}
			var df DialogueFile
			if err := yaml.Unmarshal(data, &df); err != nil {
				continue
			}
			fn(df.MobId, zoneDir.Name(), &df)
		}
	}
}
