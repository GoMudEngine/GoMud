package facts

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"
	_ "gopkg.in/yaml.v3" // pre-imported for Task 4 YAML calls
)

var (
	registry      *Registry
	registryMu    sync.RWMutex
	registrySaveMu sync.Mutex // serializes registry disk writes

	awarenessCache    = make(map[int]*Awareness)
	awarenessCacheMu  sync.RWMutex
	awarenessSaveMu   sync.Mutex // serializes awareness disk writes
)

func registryFilePath() string {
	return filepath.Join(string(configs.GetFilePathsConfig().DataFiles), "world", "dogmud", "facts.yaml")
}

func awarenessFilePath(mobId int, mobName string) string {
	return filepath.Join(string(configs.GetFilePathsConfig().DataFiles),
		"world", "dogmud", "facts.awareness",
		fmt.Sprintf("%d-%s.yaml", mobId, util.ConvertForFilename(mobName)))
}
