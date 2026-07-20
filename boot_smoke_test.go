package main

import (
	"os"
	"runtime/debug"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/quests"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
)

// bootSmokeEnvVar gates the boot smoke test.
//
// It is opt-in rather than always-on for two reasons: it loads the entire world
// (~40s, and materially longer under -race), and it populates every package
// global in the process, which would leak into any other test added to package
// main later. CI runs it in a dedicated step; see
// .github/actions/codegen-and-test/action.yml.
const bootSmokeEnvVar = `DOGMUD_BOOT_SMOKE`

// TestSmoke_ServerBootsCleanWithRealData loads every YAML data file through the
// real boot path and fails if anything is malformed.
//
// This automates the manual step in CLAUDE.md's Pre-Push SOP: "Boot the server
// locally and confirm it starts cleanly past data-file loading." That step
// exists because `go build` and `go vet` cannot see content errors — a
// filename/name-field mismatch, an invalid trigger event, an ID collision or a
// bad cross-reference only surfaces when the loaders run, and they deliberately
// panic when they find one. Until now nothing verified that automatically; it
// depended on a human remembering to boot the server before pushing.
//
// It calls loadAllDataFiles, the same function main() calls, so it exercises
// the real ordering and the dependency-injected cross-validators (schedule and
// patrol room/path checks, conversation pair checks, faction holding cells,
// mutation graph, species body parts) rather than a reimplementation that could
// drift from boot.
//
// This is also the prerequisite for the yaml.v2 -> v3 migration (audit finding
// 5.1): that swap changes decoding behaviour across every content file, and is
// not safe to attempt without an automated check that the whole tree still
// loads.
func TestSmoke_ServerBootsCleanWithRealData(t *testing.T) {
	if os.Getenv(bootSmokeEnvVar) == `` {
		t.Skipf("set %s=1 to run the full data-file boot smoke test (~40s)", bootSmokeEnvVar)
	}

	// "LOW" maps to slog Warn, which suppresses the very noisy per-file and
	// colour-palette Info logging the loaders emit.
	mudlog.SetupLogger(nil, `LOW`, ``, false)

	if err := configs.ReloadConfig(); err != nil {
		t.Fatalf("ReloadConfig failed — cannot boot without config: %v", err)
	}

	// The loaders panic by design on malformed content (fail fast at boot
	// rather than silently at runtime). Capture that and report it as a test
	// failure with the stack, which is far more useful than a raw panic.
	var bootPanic any
	var bootStack string

	start := time.Now()
	func() {
		defer func() {
			if r := recover(); r != nil {
				bootPanic = r
				bootStack = string(debug.Stack())
			}
		}()
		loadAllDataFiles(false)
	}()
	elapsed := time.Since(start)

	if bootPanic != nil {
		t.Fatalf("loading real data files panicked — the server would not boot:\n\n%v\n\n%s",
			bootPanic, bootStack)
	}

	t.Logf("world loaded in %s", elapsed)

	// A clean run is necessary but not sufficient: a loader that silently found
	// nothing (wrong path, empty dir) would also "not panic". Assert each major
	// category actually produced content.
	categories := []struct {
		name  string
		count int
	}{
		{"rooms", len(rooms.GetAllRoomIds())},
		{"mob templates", len(mobs.AllMobTemplates())},
		{"spells", len(spells.GetAllSpells())},
		{"buffs", len(buffs.GetAllBuffIds())},
		{"quests", len(quests.GetAllQuests())},
		{"crafting recipes", len(crafting.GetAll())},
	}

	for _, c := range categories {
		if c.count == 0 {
			t.Errorf("%s: loaded 0 entries — the loader ran without error but found nothing, "+
				"which usually means a wrong or empty data path", c.name)
			continue
		}
		t.Logf("%-18s %d loaded", c.name, c.count)
	}
}
