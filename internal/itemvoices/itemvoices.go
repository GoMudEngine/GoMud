// Package itemvoices loads sentient-item voice definitions: authored line
// pools keyed by event, one YAML per voice at
// _datafiles/world/dogmud/itemvoices/<voiceid>.yaml. Consumed by the
// pinnacle per-round tick (internal/hooks) for items with a voice_id.
package itemvoices

import (
	"fmt"
	"sort"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/pkg/errors"
)

// validVoiceEvents is the fixed set of event keys a voice's Lines map may
// use. Validate() rejects any other key.
var validVoiceEvents = map[string]bool{
	"on_equip":          true,
	"on_unequip":        true,
	"on_kill":           true,
	"on_idle":           true,
	"on_hunger_warning": true,
	"on_hunger_feeding": true,
	"on_taunt":          true,
	"on_grudge":         true,
}

// VoiceSpec is the data-driven definition of a sentient item's voice,
// loaded from YAML. Lines is keyed by event name; each event maps to a
// pool of candidate lines, one of which is chosen at random when the
// event fires.
type VoiceSpec struct {
	VoiceId string              `yaml:"voiceid"`
	Lines   map[string][]string `yaml:"lines"`
}

// Id implements fileloader.Loadable.
func (v *VoiceSpec) Id() string { return v.VoiceId }

// Filepath implements fileloader.Loadable.
func (v *VoiceSpec) Filepath() string {
	return util.FilePath(v.VoiceId + ".yaml")
}

// Validate implements fileloader.Loadable.
func (v *VoiceSpec) Validate() error {
	if v.VoiceId == "" {
		return fmt.Errorf("voiceid cannot be empty")
	}
	for event, lines := range v.Lines {
		if !validVoiceEvents[event] {
			return fmt.Errorf("voice %q: unknown event %q", v.VoiceId, event)
		}
		if len(lines) == 0 {
			return fmt.Errorf("voice %q: event %q has no lines", v.VoiceId, event)
		}
	}
	return nil
}

// Line returns a random line from the event's pool, or "" if the event has
// no authored lines (unknown event or empty pool).
func (v *VoiceSpec) Line(event string) string {
	pool := v.Lines[event]
	if len(pool) == 0 {
		return ""
	}
	return pool[util.Rand(len(pool))]
}

// Package-level registry, populated by LoadDataFiles.
var allVoices map[string]*VoiceSpec

// GetVoice returns the VoiceSpec for a given id, or nil if not found.
func GetVoice(id string) *VoiceSpec {
	if allVoices == nil {
		return nil
	}
	return allVoices[id]
}

// AllVoiceIds returns every loaded voice id, sorted — for the item editor's
// sentient-voice dropdown (so only resolvable voices are offered).
func AllVoiceIds() []string {
	out := make([]string, 0, len(allVoices))
	for id := range allVoices {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// SeedVoicesForTest swaps the registry for a test-supplied map, returning a
// restore func to be deferred by the caller.
func SeedVoicesForTest(m map[string]*VoiceSpec) func() {
	old := allVoices
	allVoices = m
	return func() { allVoices = old }
}

// LoadDataFiles reads all YAML files from the itemvoices/ data directory
// and populates the in-memory registry. Called once at startup from
// main.go. It MUST run AFTER items.LoadDataFiles() so the voice_id
// cross-validation below sees every loaded ItemSpec.
//
// After voices load, every ItemSpec with a non-empty VoiceId must
// reference a loaded voice, or LoadDataFiles panics (dangling voice_id
// references must fail loudly at boot, same as schedule_id / patrol_id
// world validation).
func LoadDataFiles() {
	start := time.Now()

	dataPath := string(configs.GetFilePathsConfig().DataFiles) + `/itemvoices`
	tmpAll, err := fileloader.LoadAllFlatFiles[string, *VoiceSpec](dataPath)
	if err != nil {
		panic(errors.Wrap(err, `filepath: `+dataPath))
	}

	allVoices = tmpAll
	mudlog.Info("itemvoices.LoadDataFiles()", "loadedCount", len(allVoices), "Time Taken", time.Since(start))

	for itemId, spec := range items.GetAllItemSpecsMap() {
		if spec.VoiceId == "" {
			continue
		}
		if GetVoice(spec.VoiceId) == nil {
			panic(fmt.Sprintf("item %d references unknown voice_id %q", itemId, spec.VoiceId))
		}
	}
}
