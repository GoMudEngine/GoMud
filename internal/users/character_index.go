package users

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

// foldCharacterName is the canonical map key for character-name lookups.
func foldCharacterName(name string) string {
	return strings.ToLower(name)
}

// characterScanRecord is the minimal decode target for the character-name
// scan — only the owning userId and the character's name are read.
type characterScanRecord struct {
	UserId    int `yaml:"userid"`
	Character struct {
		Name string `yaml:"name"`
	} `yaml:"character"`
}

// altScanRecord is the minimal decode target for one entry of an
// <userId>.alts.yaml list.
type altScanRecord struct {
	Name string `yaml:"name"`
}

// CharacterNameIndex builds a case-folded character-name -> userId map (mains
// + alts) in ONE minimal-decode pass over the users directory.
//
// Use this for bulk lookups (e.g. the boot-time mob-name collision audit,
// which used to call CharacterNameSearch — a full directory scan — once per
// mob template, making boot O(mobs x users): ~34k YAML decodes and ~21-47s of
// silent boot time). Building the index costs one scan; lookups are free.
// For a single one-off lookup, CharacterNameSearch remains fine.
func CharacterNameIndex() map[string]int {
	basePath := util.FilePath(string(configs.GetFilePathsConfig().DataFiles), `/`, `users`)
	return characterNameIndexFromDir(basePath)
}

func characterNameIndexFromDir(basePath string) map[string]int {

	index := map[string]int{}

	filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		name := info.Name()
		if !strings.HasSuffix(name, `.yaml`) || strings.HasSuffix(name, `.alts.yaml`) {
			return nil
		}

		fileBytes, err := os.ReadFile(path)
		if err != nil {
			mudlog.Warn("CharacterNameIndex", "warning", "unreadable user file, skipping", "path", path, "error", err)
			return nil
		}

		var rec characterScanRecord
		if err := yaml.Unmarshal(fileBytes, &rec); err != nil {
			mudlog.Warn("CharacterNameIndex", "warning", "malformed user file, skipping", "path", path, "error", err)
			return nil
		}
		if rec.UserId <= 0 {
			return nil
		}
		if rec.Character.Name != `` {
			index[foldCharacterName(rec.Character.Name)] = rec.UserId
		}

		// Alt characters live beside the main save as <userId>.alts.yaml.
		altsPath := strings.TrimSuffix(path, `.yaml`) + `.alts.yaml`
		altBytes, err := os.ReadFile(altsPath)
		if err != nil {
			return nil // no alts file — the common case
		}
		var alts []altScanRecord
		if err := yaml.Unmarshal(altBytes, &alts); err != nil {
			mudlog.Warn("CharacterNameIndex", "warning", "malformed alts file, skipping", "path", altsPath, "error", err)
			return nil
		}
		for _, alt := range alts {
			if alt.Name != `` {
				index[foldCharacterName(alt.Name)] = rec.UserId
			}
		}
		return nil
	})

	return index
}
