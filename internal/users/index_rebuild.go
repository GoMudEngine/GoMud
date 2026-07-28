package users

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

// indexScanRecord is the minimal decode target for the index rebuild scan —
// unmarshalling the full UserRecord just to extract two fields was the
// dominant cost of the old rebuild (upstream #638).
type indexScanRecord struct {
	UserId   int    `yaml:"userid"`
	Username string `yaml:"username"`
}

// rebuildFromDir re-indexes every user file under basePath with one
// minimal-decode scan and a single atomic write (temp file + rename).
//
// Hardening (ported from upstream #638): a malformed or unreadable file is
// skipped with a warning instead of aborting the walk — the old
// SearchOfflineUsers-based rebuild returned the error, silently dropping every
// user after the bad file from the index. Duplicate userids/usernames and
// filename/content mismatches are logged; the first record wins.
func (idx *UserIndex) rebuildFromDir(basePath string) error {

	records := make([]IndexUserRecord, 0, 64)
	seenIds := make(map[int]string)
	seenNames := make(map[string]int)
	highest := 0

	filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			mudlog.Warn("UserIndex", "warning", "walk error, skipping", "path", path, "error", err)
			return nil
		}
		if info.IsDir() {
			return nil
		}
		name := info.Name()
		if !strings.HasSuffix(name, `.yaml`) || strings.HasSuffix(name, `.alts.yaml`) {
			return nil
		}

		fileBytes, err := os.ReadFile(path)
		if err != nil {
			mudlog.Warn("UserIndex", "warning", "unreadable user file, skipping", "path", path, "error", err)
			return nil
		}

		var rec indexScanRecord
		if err := yaml.Unmarshal(fileBytes, &rec); err != nil {
			mudlog.Warn("UserIndex", "warning", "malformed user file, skipping", "path", path, "error", err)
			return nil
		}
		if rec.UserId <= 0 || rec.Username == `` {
			mudlog.Warn("UserIndex", "warning", "user file missing userid/username, skipping", "path", path)
			return nil
		}

		username := strings.ToLower(rec.Username)
		if firstName, dup := seenIds[rec.UserId]; dup {
			mudlog.Warn("UserIndex", "warning", "duplicate userid, skipping", "path", path, "userId", rec.UserId, "firstUsername", firstName)
			return nil
		}
		if firstId, dup := seenNames[username]; dup {
			mudlog.Warn("UserIndex", "warning", "duplicate username, skipping", "path", path, "username", username, "firstUserId", firstId)
			return nil
		}

		// Modern files are named <userid>.yaml — flag content that disagrees.
		if base := strings.TrimSuffix(name, `.yaml`); base != `` {
			if fnameId, convErr := strconv.Atoi(base); convErr == nil && fnameId != rec.UserId {
				mudlog.Warn("UserIndex", "warning", "filename does not match userid in content", "path", path, "userId", rec.UserId)
			}
		}

		seenIds[rec.UserId] = username
		seenNames[username] = rec.UserId

		newRecord := IndexUserRecord{UserID: int64(rec.UserId)}
		copy(newRecord.Username[:], username)
		records = append(records, newRecord)

		if rec.UserId > highest {
			highest = rec.UserId
		}
		return nil
	})

	idx.metaData = IndexMetaData{
		MetaDataSize: FixedHeaderTotalLength,
		IndexVersion: IndexVersion,
		RecordCount:  uint64(len(records)),
		RecordSize:   IndexRecordSizeV1,
	}

	if err := idx.writeCompleteIndexAtomic(records); err != nil {
		return err
	}

	if highest > idx.highestUserId {
		idx.highestUserId = highest
	}
	return nil
}

// writeCompleteIndexAtomic writes header + records to a temp file in the same
// directory, then renames it over the index — a crash mid-write can never
// leave a truncated index behind.
func (idx *UserIndex) writeCompleteIndexAtomic(records []IndexUserRecord) error {
	tmpName := idx.Filename + `.tmp`

	f, err := os.Create(tmpName)
	if err != nil {
		return err
	}

	writeErr := func() error {
		headerBytes, err := idx.metaData.Format()
		if err != nil {
			return err
		}
		if _, err := f.Write(headerBytes); err != nil {
			return err
		}
		for _, rec := range records {
			if _, err := f.Write(rec.Username[:]); err != nil {
				return err
			}
			if err := binary.Write(f, binary.LittleEndian, rec.UserID); err != nil {
				return err
			}
			if _, err := f.Write([]byte{IndexLineTerminatorV1}); err != nil {
				return err
			}
		}
		return f.Sync()
	}()

	if closeErr := f.Close(); writeErr == nil {
		writeErr = closeErr
	}
	if writeErr != nil {
		os.Remove(tmpName)
		return writeErr
	}

	return os.Rename(tmpName, idx.Filename)
}

// Rebuild re-indexes all user files from the configured users directory.
func (idx *UserIndex) Rebuild() error {
	basePath := util.FilePath(string(configs.GetFilePathsConfig().DataFiles), `/`, `users`)
	return idx.rebuildFromDir(basePath)
}
