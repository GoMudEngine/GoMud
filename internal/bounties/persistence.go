package bounties

import (
	"path/filepath"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	_ "gopkg.in/yaml.v3" // Pre-imported for T3 YAML marshaling
)

var (
	registry   *Registry
	registryMu sync.RWMutex
	saveMu     sync.Mutex // serializes disk writes (Windows file-lock safety)
)

func registryFilePath() string {
	return filepath.Join(string(configs.GetFilePathsConfig().DataFiles), "world", "dogmud", "bounties.yaml")
}
