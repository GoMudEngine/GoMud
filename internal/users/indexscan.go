package users

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

// UserFileScan holds the fields a lightweight pass over a user file yields.
// The user index and the character index are both built from these at
// startup, without paying for a full UserRecord unmarshal per file.
type UserFileScan struct {
	UserId        int
	Username      string
	CharacterName string
}

// userFileScanFields is the minimal unmarshal target for a scan. The file
// is still lexed in full, but decoding into this instead of a full
// UserRecord is measurably cheaper and builds no throwaway record - see
// BenchmarkScanVsFullUnmarshal.
type userFileScanFields struct {
	UserId    int    `yaml:"userid"`
	Username  string `yaml:"username"`
	Character struct {
		Name string `yaml:"name"`
	} `yaml:"character"`
}

// ScanUserFiles runs a lightweight scan over the configured users directory.
func ScanUserFiles() []UserFileScan {
	basePath := util.FilePath(string(configs.GetFilePathsConfig().DataFiles), `/`, `users`)
	return scanUserFilesInDir(basePath)
}

// scanUserFilesInDir reads the userid, username, and active character name
// of every user file under basePath. Alt files are never opened. Files that
// cannot be read or parsed are skipped with a warning instead of aborting
// the scan, and anomalies that usually mean hand-edited data (duplicate
// userids, duplicate usernames, a numeric filename that disagrees with the
// userid inside the file) are logged so they get noticed.
func scanUserFilesInDir(basePath string) []UserFileScan {

	results := []UserFileScan{}
	seenIds := make(map[int]string)
	seenNames := make(map[string]string)

	filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {

		if err != nil {
			mudlog.Warn("ScanUserFiles", "path", path, "walk_error", err)
			return nil
		}

		if info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, `.yaml`) || strings.HasSuffix(path, `.alts.yaml`) {
			return nil
		}

		fileBytes, err := os.ReadFile(path)
		if err != nil {
			mudlog.Warn("ScanUserFiles", "path", path, "read_error", err)
			return nil
		}

		var scanned userFileScanFields
		if err := yaml.Unmarshal(fileBytes, &scanned); err != nil {
			mudlog.Warn("ScanUserFiles", "path", path, "unmarshal_error", err)
			return nil
		}

		if scanned.UserId < 1 || scanned.Username == `` {
			mudlog.Warn("ScanUserFiles", "info", "skipping user file missing userid or username", "path", path)
			return nil
		}

		if fileId, convErr := strconv.Atoi(strings.TrimSuffix(filepath.Base(path), `.yaml`)); convErr == nil && fileId != scanned.UserId {
			mudlog.Warn("ScanUserFiles", "info", "filename does not match userid in file", "path", path, "userid", scanned.UserId)
		}

		if otherPath, ok := seenIds[scanned.UserId]; ok {
			mudlog.Warn("ScanUserFiles", "info", "duplicate userid", "userid", scanned.UserId, "path", path, "otherpath", otherPath)
		}
		lowerName := strings.ToLower(scanned.Username)
		if otherPath, ok := seenNames[lowerName]; ok {
			mudlog.Warn("ScanUserFiles", "info", "duplicate username", "username", scanned.Username, "path", path, "otherpath", otherPath)
		}
		seenIds[scanned.UserId] = path
		seenNames[lowerName] = path

		results = append(results, UserFileScan{
			UserId:        scanned.UserId,
			Username:      scanned.Username,
			CharacterName: scanned.Character.Name,
		})

		return nil
	})

	return results
}
