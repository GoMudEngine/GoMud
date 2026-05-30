package factions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

var (
	definitionsMu sync.RWMutex
	definitions   = map[string]*Definition{}
)

// definitionsBaseDir returns the directory holding faction
// definition YAML files. Honors the DOGMUD_FACTIONS_DIR_OVERRIDE
// env var so tests can redirect to a temp dir.
func definitionsBaseDir() string {
	if override := os.Getenv("DOGMUD_FACTIONS_DIR_OVERRIDE"); override != "" {
		return override
	}
	return util.FilePath(
		configs.GetFilePathsConfig().DataFiles.String(), `/`, `factions`,
	)
}

// LoadAllDefinitions reads every YAML file under the factions dir
// and populates the in-memory registry. Validates that every
// allies/enemies reference resolves to a loaded faction; panics on
// unknown reference so authoring errors surface at boot, not at
// runtime.
//
// Idempotent — safe to call multiple times. Subsequent calls
// fully replace the registry contents.
func LoadAllDefinitions() error {
	dir := definitionsBaseDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			definitionsMu.Lock()
			definitions = map[string]*Definition{}
			definitionsMu.Unlock()
			return nil
		}
		return fmt.Errorf("factions.LoadAllDefinitions: readdir %s: %w", dir, err)
	}

	loaded := map[string]*Definition{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			mudlog.Warn("factions.LoadAllDefinitions: read", "path", path, "error", readErr)
			continue
		}
		var def Definition
		if unmErr := yaml.Unmarshal(raw, &def); unmErr != nil {
			mudlog.Warn("factions.LoadAllDefinitions: unmarshal", "path", path, "error", unmErr)
			continue
		}
		if def.FactionId == "" {
			mudlog.Warn("factions.LoadAllDefinitions: missing faction_id", "path", path)
			continue
		}
		loaded[def.FactionId] = &def
	}

	// Validate ally/enemy references.
	for _, def := range loaded {
		for _, ally := range def.Allies {
			if _, ok := loaded[ally]; !ok {
				panic(fmt.Sprintf("factions: faction %q lists unknown ally %q",
					def.FactionId, ally))
			}
		}
		for _, enemy := range def.Enemies {
			if _, ok := loaded[enemy]; !ok {
				panic(fmt.Sprintf("factions: faction %q lists unknown enemy %q",
					def.FactionId, enemy))
			}
		}
	}

	definitionsMu.Lock()
	definitions = loaded
	definitionsMu.Unlock()

	mudlog.Info("factions.LoadAllDefinitions", "loadedCount", len(loaded))
	return nil
}

// GetDefinition returns the immutable Definition for factionId, or
// nil if unknown.
func GetDefinition(factionId string) *Definition {
	definitionsMu.RLock()
	defer definitionsMu.RUnlock()
	return definitions[factionId]
}

// AllDefinitions returns a snapshot of every loaded definition.
// Read-only — callers must not mutate.
func AllDefinitions() []*Definition {
	definitionsMu.RLock()
	defer definitionsMu.RUnlock()
	out := make([]*Definition, 0, len(definitions))
	for _, d := range definitions {
		out = append(out, d)
	}
	return out
}

// ValidateHoldingCells panics if any loaded faction declares a HoldingCellRoom
// that roomExists reports as missing. Cells of 0 (no jail) are skipped. Called
// from main.go after rooms load (DI breaks the factions<-rooms import edge).
func ValidateHoldingCells(roomExists func(roomId int) bool) {
	definitionsMu.RLock()
	defer definitionsMu.RUnlock()
	for _, def := range definitions {
		if def.HoldingCellRoom == 0 {
			continue
		}
		if !roomExists(def.HoldingCellRoom) {
			panic(fmt.Sprintf("factions: faction %q holding_cell_room %d does not exist",
				def.FactionId, def.HoldingCellRoom))
		}
	}
}

// clearRegistryForTest wipes the registry. Test-only seam.
func clearRegistryForTest() {
	definitionsMu.Lock()
	definitions = map[string]*Definition{}
	definitionsMu.Unlock()
}
