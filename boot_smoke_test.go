package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/dialogue"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/quests"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
	yaml "gopkg.in/yaml.v2"
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

// TestSmoke_AllDialogueFilesParse eagerly parses every dialogue file.
//
// Dialogue is the one major content type loadAllDataFiles does NOT cover: it is
// lazy-loaded per mob on first interaction (dialogue.Load). That laziness is
// what made the documented "bare-scalar list field mutes a whole NPC" incident
// so expensive — internal/dialogue/loader.go discards the ENTIRE file on any
// unmarshal error and permanently caches a nil sentinel, so a single malformed
// field silently mutes that NPC for the life of the process, and nothing at
// boot or in CI notices. It only surfaces when a player talks to them.
//
// Parsing every file up front turns that from a silent production failure into
// a CI failure. Note this deliberately checks parsing only, mirroring exactly
// what dialogue.Load does — it is not a semantic validation of the content.
func TestSmoke_AllDialogueFilesParse(t *testing.T) {
	if os.Getenv(bootSmokeEnvVar) == `` {
		t.Skipf("set %s=1 to run the dialogue parse sweep", bootSmokeEnvVar)
	}

	root := filepath.Join(`_datafiles`, `world`, `dogmud`, `dialogue`)
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("dialogue directory not found at %s: %v", root, err)
	}

	var checked, failed int

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, `.yaml`) {
			return nil
		}

		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Errorf("%s: unreadable: %v", path, readErr)
			failed++
			return nil
		}

		var df dialogue.DialogueFile
		if unmarshalErr := yaml.Unmarshal(raw, &df); unmarshalErr != nil {
			// This is exactly the condition that mutes an NPC at runtime.
			t.Errorf("%s: would mute this NPC at runtime — dialogue.Load discards the whole "+
				"file and caches a nil sentinel on this error:\n    %v", path, unmarshalErr)
			failed++
			return nil
		}

		checked++
		return nil
	})
	if err != nil {
		t.Fatalf("walking dialogue tree: %v", err)
	}

	if checked == 0 {
		t.Fatal("parsed 0 dialogue files — wrong path?")
	}
	t.Logf("dialogue files parsed cleanly: %d (failed: %d)", checked, failed)
}

// knownSilentlyIgnoredKeys is the accepted baseline of YAML keys that appear in
// content files but map to no field on their target type, so the lenient
// decoder discards them.
//
// This exists as a DRIFT GATE, not an approval. Flipping the loaders to strict
// decoding outright would fail the boot with 3,213 violations, the vast bulk of
// them benign: `coord` (2,459) is authored room data the engine abandoned in
// favour of crawled exit-delta positions, `level` (612) is legacy from the
// level/XP system being removed, and most quest entries are an artifact of quest
// YAML being parsed by two type systems (the legacy `quests` package and the
// newer `questengine`), each seeing the other's fields as unknown.
//
// Baselining the distinct field/type pairs means a NEW mistyped or unsupported
// key — the `hostile:` failure mode, where an authored value silently does
// nothing — fails CI immediately, without requiring the existing backlog to be
// cleared first.
//
// Several entries below are genuine content bugs worth fixing, listed in the
// audit doc rather than silently accepted:
//   - `item_id` on quests.QuestReward: CLAUDE.md documents that reward keys are
//     no-underscore (`itemid`); the snake_case form silently no-ops. This is a
//     live instance of that footgun.
//   - `scriptag` on mobs.Mob: almost certainly a typo for `scripttag`.
//   - `visible` / `sequential` / `expireMessage` on buffs.BuffSpec.
//   - `cooldown` on rooms.SpawnInfo (118x) and `zone` on exit.RoomExit (136x):
//     authored values doing nothing.
//
// To clear an entry: fix the content (or add the field to the struct), confirm
// the count drops, and delete the line.
var knownSilentlyIgnoredKeys = map[string]bool{
	"coord|rooms.Room":                       true,
	"level|characters.Character":             true,
	"zone|exit.RoomExit":                     true,
	"cooldown|rooms.SpawnInfo":               true,
	"triggers|quests.Quest":                  true,
	"playermessage|questengine.QuestRewards": true,
	"roommessage|questengine.QuestRewards":   true,
	"linear|quests.Quest":                    true,
	"itemid|questengine.QuestRewards":        true,
	"skillinfo|questengine.QuestRewards":     true,
	"map_target|quests.QuestStep":            true,
	"items|mobs.Mob":                         true,
	"rep_faction|questengine.QuestRewards":   true,
	"rep_amount|questengine.QuestRewards":    true,
	"repeatable|questengine.QuestDef":        true,
	"cooldown_rounds|questengine.QuestDef":   true,
	"tactics|characters.Character":           true,
	"recipe_info|questengine.QuestRewards":   true,
	"long|rooms.Container":                   true,
	"item_info|questengine.QuestRewards":     true,
	"triggers|quests.QuestStep":              true,
	"triggers|questengine.QuestStep":         true,
	"spellid|questengine.QuestRewards":       true,
	"scriptag|mobs.Mob":                      true,
	"allow_recall|rooms.Room":                true,
	"visible|buffs.BuffSpec":                 true,
	"sequential|buffs.BuffSpec":              true,
	"expireMessage|buffs.BuffSpec":           true,
	"buffid|questengine.QuestRewards":        true,
}

var unknownKeyRe = regexp.MustCompile(`field ([A-Za-z_0-9]+) not found in type ([A-Za-z_.]+)`)

// TestSmoke_NoNewSilentlyIgnoredYAMLKeys fails when content introduces a YAML
// key that maps to no field on its target type and is not already baselined.
//
// This closes the `hostile:` incident class going forward: an authored value
// that silently does nothing now fails CI instead of shipping to production and
// being discovered months later.
func TestSmoke_NoNewSilentlyIgnoredYAMLKeys(t *testing.T) {
	if os.Getenv(bootSmokeEnvVar) == `` {
		t.Skipf("set %s=1 to run the unknown-YAML-key drift gate", bootSmokeEnvVar)
	}

	mudlog.SetupLogger(nil, `LOW`, ``, false)
	if err := configs.ReloadConfig(); err != nil {
		t.Fatalf("ReloadConfig failed: %v", err)
	}

	var mu sync.Mutex
	found := map[string][]string{} // "field|type" -> example paths

	fileloader.StrictDecodeProbe = func(path string, err error) {
		mu.Lock()
		defer mu.Unlock()
		for _, m := range unknownKeyRe.FindAllStringSubmatch(err.Error(), -1) {
			key := m[1] + "|" + m[2]
			if len(found[key]) < 3 {
				found[key] = append(found[key], path)
			}
		}
	}
	defer func() { fileloader.StrictDecodeProbe = nil }()

	func() {
		defer func() { _ = recover() }() // boot failures are the other test's job
		loadAllDataFiles(false)
	}()

	var novel []string
	for key := range found {
		if !knownSilentlyIgnoredKeys[key] {
			novel = append(novel, key)
		}
	}
	sort.Strings(novel)

	for _, key := range novel {
		t.Errorf("new silently-ignored YAML key %q — this value is authored but does nothing, "+
			"because no field on that type matches it. Fix the key, add the field, or (if "+
			"deliberate) add it to knownSilentlyIgnoredKeys.\n    e.g. %v",
			key, found[key])
	}

	t.Logf("distinct silently-ignored keys: %d (baselined: %d, new: %d)",
		len(found), len(knownSilentlyIgnoredKeys), len(novel))
}
