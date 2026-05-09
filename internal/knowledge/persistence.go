package knowledge

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"
	_ "gopkg.in/yaml.v3" // Pre-imported for T3 YAML marshaling
)

var (
	knowledgeCache   = make(map[int]*ObserverFile)
	knowledgeCacheMu sync.RWMutex
	saveMu           sync.Mutex // serializes disk writes (Windows file-lock safety)
)

func knowledgeBaseDir() string {
	return filepath.Join(string(configs.GetFilePathsConfig().DataFiles), "world", "dogmud", "knowledge")
}

// observerFilePath returns the absolute path for the given observer mob template id.
// The mobName is used in the filename for human readability; mismatch is
// tolerated (filename is not the lookup key).
func observerFilePath(mobId int, mobName string) string {
	return filepath.Join(knowledgeBaseDir(),
		fmt.Sprintf("%d-%s.yaml", mobId, util.ConvertForFilename(mobName)))
}
