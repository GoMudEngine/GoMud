// Package moderation owns durable, admin-facing moderation state: the player
// petition queue and the account/IP ban lists. State persists as YAML under
// _datafiles/moderation/ and is living runtime state (like guilds/shops) — it
// is gitignored, kept on prod, and must NOT be wiped by the instance-save SOP.
package moderation

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

var (
	mu              sync.Mutex
	dataDirOverride string // test hook; empty = use config
)

func moderationDir() string {
	if dataDirOverride != "" {
		return dataDirOverride
	}
	return filepath.FromSlash(configs.GetFilePathsConfig().DataFiles.String() + `/moderation`)
}

// SetDataDirForTest points persistence at dir and returns a restore func.
func SetDataDirForTest(dir string) func() {
	prev := dataDirOverride
	dataDirOverride = dir
	return func() { dataDirOverride = prev }
}

// resetForTest clears all in-memory state (test-only). Task 2 extends this to
// also clear the ban maps.
func resetForTest() {
	mu.Lock()
	defer mu.Unlock()
	petitions = nil
	nextPetitionId = 1
}

// LoadDataFiles loads the moderation stores into memory at boot. Missing or
// malformed files are logged + skipped (runtime state must not crash boot).
// Task 2 extends this to also call loadBans().
func LoadDataFiles() {
	loadPetitions()
	mudlog.Info("moderation.LoadDataFiles()", "petitions", len(petitions))
}

// now is overridable in tests if deterministic timestamps are ever needed.
var now = time.Now
