package dialogue

// The dialogue writer (5b). No writer existed before this; the loader's
// contract it must honour is documented at each step. Validation is NOT
// called here — the GMCP handler validates with injected registries first
// (the spawn-editor lesson: hardwiring real registries into the save path
// makes it untestable without a loaded world).

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"
	yaml "gopkg.in/yaml.v2"
)

func dialogueFilePath(mobId int, zone string) string {
	dataFiles := string(configs.GetFilePathsConfig().DataFiles)
	return util.FilePath(dataFiles + `/dialogue/` + zoneNameSanitize(zone) + `/` + fmt.Sprintf("%d", mobId) + `.yaml`)
}

// SaveDialogueFile writes the file at the loader's exact path and replaces
// the cache entry in place so live NPCs serve the edit immediately.
func SaveDialogueFile(df DialogueFile) error {
	if df.MobId == 0 || df.Zone == "" {
		return errors.New("dialogue file needs a mob id and zone")
	}
	data, err := yaml.Marshal(&df)
	if err != nil {
		return err
	}
	path := dialogueFilePath(df.MobId, df.Zone)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}
	key := fmt.Sprintf("%d:%s", df.MobId, df.Zone)
	cp := df
	dialogueCache[key] = &cp
	delete(nilSentinel, key) // a save proves the file exists
	return nil
}

// CreateNewDialogueFile writes a minimal skeleton and clears the nil
// sentinel — without that, a mob whose Load already ran stays "dialogueless"
// until reboot no matter what is on disk.
func CreateNewDialogueFile(mobId int, zone string) error {
	if _, err := os.Stat(dialogueFilePath(mobId, zone)); err == nil {
		return fmt.Errorf("mob %d already has a dialogue file", mobId)
	}
	df := DialogueFile{
		MobId: mobId, Zone: zone, DefaultMood: string(MoodNeutral),
		Patterns: []Pattern{{Keywords: []string{""}, Responses: []string{"..."}}},
	}
	return SaveDialogueFile(df)
}

// DeleteDialogueFile removes the file, drops the cache entry, and SETS the
// sentinel: the mob now genuinely has no dialogue.
func DeleteDialogueFile(mobId int, zone string) error {
	path := dialogueFilePath(mobId, zone)
	if err := os.Remove(path); err != nil {
		return err
	}
	key := fmt.Sprintf("%d:%s", mobId, zone)
	delete(dialogueCache, key)
	nilSentinel[key] = true
	return nil
}
