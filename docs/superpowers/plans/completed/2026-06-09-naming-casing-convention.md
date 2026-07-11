# Game-wide Title-Case Naming Convention — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make every player-visible name and title use one canonical smart-Title-Case form, applied through a single formatter, with a load-time panic validator so the inconsistency can't reappear.

**Architecture:** A pure leaf package `internal/casing` exposes `Title(s)` (smart title case: minor words lowercased mid-name, internal capitals preserved, idempotent). It is the single source of truth, used by (a) the runtime display/title paths, (b) a one-time data sweep that canonicalizes stored display names, and (c) a load-time validator that panics on any non-canonical stored display name. Lookup keys, tags, filenames, and ids are never touched.

**Tech Stack:** Go (stdlib only — `strings`, `unicode`); existing YAML loaders; `mudlog`/`panic` startup-validator pattern.

**Spec:** `docs/superpowers/specs/2026-06-09-naming-casing-convention-design.md`

---

## File Structure

- `internal/casing/casing.go` — NEW. `Title(s string) string` + minor-word set. Pure, no deps.
- `internal/casing/casing_test.go` — NEW. Unit tests for `Title`.
- `internal/skills/skills.go` — MODIFY. `GetMutationTier` returns lowercase words; `GetTitle` wraps result in `casing.Title`.
- `internal/characters/formattedname.go` — MODIFY. Remove local `titleCase`; mob-name display path calls `casing.Title`.
- `internal/<loaders>/*.go` — MODIFY (Task 6). Add a name-validator call after each display-name-bearing loader (mobs, items, rooms, spells, buffs).
- `tools/casing_sweep/main.go` — NEW (Task 5). One-time data sweep that canonicalizes stored display-name lines, reusing `casing.Title`.
- Data files under `_datafiles/world/dogmud/` — MODIFY by the sweep (display-name values only).

---

## Task 1: `internal/casing` package (the keystone formatter)

**Files:**
- Create: `internal/casing/casing.go`
- Test: `internal/casing/casing_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/casing/casing_test.go`:

```go
package casing

import "testing"

func TestTitle(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"olen", "Olen"},
		{"temple priest olen", "Temple Priest Olen"},
		{"city guard", "City Guard"},
		{"iron sword", "Iron Sword"},
		{"captain of the guard", "Captain of the Guard"},
		{"tome of the deep", "Tome of the Deep"},
		{"ascendant master warrior", "Ascendant Master Warrior"},
		// Minor word as FIRST or LAST word is still capitalized.
		{"the warren", "The Warren"},
		{"keeper of the", "Keeper of The"},
		// Internal capitals preserved (proper names / mixed case).
		{"Olen", "Olen"},
		{"McGregor the brave", "McGregor the Brave"},
		// Idempotent on already-canonical input.
		{"Captain of the Guard", "Captain of the Guard"},
		// Whitespace collapses to single spaces between words.
		{"valley   rat", "Valley Rat"},
		// Leading/trailing whitespace trimmed.
		{"  lich  ", "Lich"},
		// All-minor-word name: first+last cap rule.
		{"a", "A"},
	}
	for _, c := range cases {
		if got := Title(c.in); got != c.want {
			t.Errorf("Title(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestTitleIdempotent(t *testing.T) {
	for _, s := range []string{"temple priest olen", "captain of the guard", "iron sword", "the warren"} {
		once := Title(s)
		if twice := Title(once); twice != once {
			t.Errorf("Title not idempotent: Title(%q)=%q then Title(that)=%q", s, once, twice)
		}
	}
}
```

- [ ] **Step 2: Run the tests; verify they fail to compile (package missing)**

Run: `go test ./internal/casing/ -run TestTitle -v`
Expected: build failure — `package casing` / `Title` undefined.

- [ ] **Step 3: Implement `casing.Title`**

Create `internal/casing/casing.go`:

```go
// Package casing is the single source of truth for player-visible display
// casing. Title applies smart title case to a name or title string.
//
// It MUST only be used on player-visible name/title strings — never on lookup
// keys, noun tags, filenames, ids, or anything used for parsing/matching.
package casing

import (
	"strings"
	"unicode"
)

// minorWords are lower-cased when they appear as an interior word (not the
// first or last word). Title-case convention for articles, coordinating
// conjunctions, and short prepositions.
var minorWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "as": {}, "at": {}, "but": {}, "by": {},
	"for": {}, "from": {}, "in": {}, "nor": {}, "of": {}, "on": {}, "or": {},
	"the": {}, "to": {}, "with": {},
}

// Title returns s in smart title case. Each whitespace-separated word's first
// rune is upper-cased, except interior minor words which are lower-cased. The
// first and last words are always capitalized. Characters after the first rune
// of a word are left untouched, so existing internal capitals (e.g. "McGregor",
// "Olen") are preserved and the function is idempotent on canonical input.
func Title(s string) string {
	words := strings.Fields(s) // trims + collapses whitespace
	last := len(words) - 1
	for i, w := range words {
		lower := strings.ToLower(w)
		if i != 0 && i != last {
			if _, isMinor := minorWords[lower]; isMinor {
				words[i] = lower
				continue
			}
		}
		words[i] = upperFirst(w)
	}
	return strings.Join(words, " ")
}

// upperFirst upper-cases the first rune of w, leaving the remainder unchanged.
func upperFirst(w string) string {
	if w == "" {
		return w
	}
	r := []rune(w)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
```

- [ ] **Step 4: Run the tests; verify they pass**

Run: `go test ./internal/casing/ -v`
Expected: PASS (both tests).

- [ ] **Step 5: Commit**

```bash
git add internal/casing/casing.go internal/casing/casing_test.go
git commit -m "feat(casing): smart Title-Case formatter (single source of truth)"
```

---

## Task 2: Fix player titles (`skills.GetTitle`)

**Files:**
- Modify: `internal/skills/skills.go` (`GetMutationTier` ~115-129, `GetTitle` ~247-260)
- Test: `internal/skills/skills_test.go` (add a test; create if absent)

- [ ] **Step 1: Write the failing test**

Add to `internal/skills/skills_test.go` (create the file with `package skills` + imports if it doesn't exist):

```go
func TestGetTitle_IsTitleCased(t *testing.T) {
	// No mutations, low skills, Strength-dominant → "scrub warrior" canonicalized.
	var s stats.Statistics
	s.Strength.Value = 200
	s.Dexterity.Value = 100
	s.Perception.Value = 100
	s.Vitality.Value = 100
	s.Willpower.Value = 100
	s.Charisma.Value = 100

	got := GetTitle(map[string]int{}, map[string]int{}, s)
	if got != "Scrub Warrior" {
		t.Errorf("GetTitle = %q, want %q", got, "Scrub Warrior")
	}
}
```

(Confirm the exact `stats.Statistics` field shape by reading `internal/stats`; adjust the struct literal if fields differ. The assertion — Title-Cased output — is the point.)

- [ ] **Step 2: Run the test; verify it fails**

Run: `go test ./internal/skills/ -run TestGetTitle_IsTitleCased -v`
Expected: FAIL — got `"scrub warrior"`, want `"Scrub Warrior"`.

- [ ] **Step 3: Lowercase the mutation-tier source words**

In `internal/skills/skills.go`, change `GetMutationTier` return values to lowercase so all three title components share one casing before formatting:

```go
func GetMutationTier(owned map[string]int) string {
	load := mutations.GetMutationLoad(owned)
	switch {
	case load >= 50:
		return "exalted"
	case load >= 30:
		return "ascendant"
	case load >= 15:
		return "evolved"
	case load >= 1:
		return "awakened"
	default:
		return ""
	}
}
```

- [ ] **Step 4: Wrap the assembled title in `casing.Title`**

Add the import `"github.com/GoMudEngine/GoMud/internal/casing"` to `skills.go`, then change the end of `GetTitle`:

```go
	return casing.Title(strings.Join(parts, " "))
```

- [ ] **Step 5: Run the test; verify it passes + no import cycle**

Run: `go test ./internal/skills/ -run TestGetTitle_IsTitleCased -v && go build ./...`
Expected: PASS; build clean. (If `internal/casing` importing fails with a cycle, it won't — `casing` has no deps.)

- [ ] **Step 6: Commit**

```bash
git add internal/skills/skills.go internal/skills/skills_test.go
git commit -m "fix(skills): Title-Case player titles via casing.Title; lowercase tier source words"
```

---

## Task 3: Route mob-name display through `casing.Title`

**Files:**
- Modify: `internal/characters/formattedname.go` (remove local `titleCase` ~49-58; call `casing.Title` at ~64-66)
- Test: `internal/characters/formattedname_test.go` (create if absent)

- [ ] **Step 1: Write the failing test**

Add to `internal/characters/formattedname_test.go`:

```go
func TestFormattedName_MobUsesSmartTitleCase(t *testing.T) {
	f := FormattedName{Name: "captain of the guard", Type: "mobname"}
	out := f.String()
	if !strings.Contains(out, "Captain of the Guard") {
		t.Errorf("mob name not smart-title-cased; got %q", out)
	}
	if strings.Contains(out, "Of The") {
		t.Errorf("minor words should be lowercased; got %q", out)
	}
}
```

- [ ] **Step 2: Run the test; verify it fails**

Run: `go test ./internal/characters/ -run TestFormattedName_MobUsesSmartTitleCase -v`
Expected: FAIL — current `titleCase` produces "Captain Of The Guard".

- [ ] **Step 3: Replace local `titleCase` with `casing.Title`**

In `internal/characters/formattedname.go`: delete the local `titleCase` function (lines ~49-58), add import `"github.com/GoMudEngine/GoMud/internal/casing"`, and change the mob branch in `String()`:

```go
	name := f.Name
	// Smart Title-Case mob names for display (single source of truth).
	if strings.HasPrefix(f.Type, `mobname`) {
		name = casing.Title(name)
	}
```

- [ ] **Step 4: Run the test; verify it passes + package green**

Run: `go test ./internal/characters/ -run TestFormattedName_MobUsesSmartTitleCase -v && go test ./internal/characters/`
Expected: PASS; no other characters tests break.

- [ ] **Step 5: Commit**

```bash
git add internal/characters/formattedname.go internal/characters/formattedname_test.go
git commit -m "refactor(characters): mob-name display uses casing.Title (smart, shared)"
```

---

## Task 4: Race display + locate the title/race concatenation surface

**Files:**
- Modify: the surface(s) that show race and/or concatenate race + title.
- Test: ad-hoc (display assertion in the relevant package if one exists).

- [ ] **Step 1: Find every surface that displays race or concatenates race + title**

Run:
```bash
grep -rn "Species()\|\.Race\b\|GetTitle(" internal/ modules/ --include=*.go | grep -iv _test
```
Record each hit. The reported "human Ascendant master…" string is produced where race + `GetTitle` are concatenated (candidates: a `who`/`score`/`look` command template, `modules/gmcp/gmcp.Char.go` `Char.Info`).

- [ ] **Step 2: Title-Case race at each display surface**

Wherever `Character.Species()` (`internal/characters/description.go:154`) feeds a player-visible field, wrap it: `casing.Title(c.Species())`. Do NOT change the stored species `Name` (lowercase) — only the display call sites. Example for GMCP `Char.Info`:

```go
Race: casing.Title(user.Character.Species()),
```

- [ ] **Step 3: Verify the concatenated string reads "Human Ascendant Master Warrior"**

Add/adjust a test in the package that owns the concatenation if one exists; otherwise verify by `go build ./...` and a boot-time spot check (Task 8). The title itself is already canonical from Task 2; this step only fixes race casing.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "fix: Title-Case race at player-visible display surfaces"
```

---

## Task 5: One-time data sweep — canonicalize stored display names

**Files:**
- Create: `tools/casing_sweep/main.go`
- Modify (by running it): display-name values in `_datafiles/world/dogmud/{mobs,items,rooms,spells,buffs}/**.yaml`

- [ ] **Step 1: Pin the exact display-name field(s) per loader (discovery)**

Run and record results — this determines the sweep's allowlist and is the critical names-only boundary:
```bash
grep -rn "yaml:\"name\|yaml:\"title\|yaml:\"displayname" internal/items/itemspec.go internal/rooms/rooms.go internal/spells/spells.go internal/buffs/buffspec.go internal/mobs/*.go
```
Known so far: mob → `character.name`; room → `title`; buff → `name`; spell → `name`; item → **`displayname` when set, else `name`** (item `Name` is also used for matching — confirm before deciding whether to canonicalize item `name` or only `displayname`). Write the final field list into the tool as `targetFields` per directory. **Lookup/keyword/tag/`namesimple` fields are excluded.**

- [ ] **Step 2: Write the sweep tool (targeted line edits, comment-preserving)**

Create `tools/casing_sweep/main.go`. It walks each data dir, and for each file rewrites ONLY the value of an allowlisted display-name line via `casing.Title`, leaving all other lines/comments/formatting intact (no YAML marshal round-trip — that would strip comments). Supports `-dry` to print a diff without writing.

```go
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/casing"
)

// dir -> set of top-level YAML keys whose VALUE is a player-visible display name.
// Derived in Step 1. Keys NOT listed here are never touched.
var targets = map[string][]string{
	`_datafiles/world/dogmud/mobs`:   {"name"},  // under character: — see keyDepth note
	`_datafiles/world/dogmud/rooms`:  {"title"},
	`_datafiles/world/dogmud/buffs`:  {"name"},
	`_datafiles/world/dogmud/spells`: {"name"},
	`_datafiles/world/dogmud/items`:  {"displayname"}, // confirm in Step 1
}

// line like:  `  name: temple priest olen`  or  `name: "temple priest olen"`
func keyLineRe(key string) *regexp.Regexp {
	return regexp.MustCompile(`^(\s*` + regexp.QuoteMeta(key) + `:\s*)(.+?)(\s*)$`)
}

func canonicalizeValue(raw string) (string, bool) {
	v := strings.TrimSpace(raw)
	quoted := false
	if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
		quoted = true
		v = v[1 : len(v)-1]
	}
	c := casing.Title(v)
	if c == v {
		return raw, false
	}
	if quoted {
		return `"` + c + `"`, true
	}
	return c, true
}

func main() {
	dry := flag.Bool("dry", false, "print changes without writing")
	flag.Parse()
	for dir, keys := range targets {
		res := keyLineMatchers(keys)
		_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(p, ".yaml") {
				return nil
			}
			processFile(p, res, *dry)
			return nil
		})
	}
}

func keyLineMatchers(keys []string) []*regexp.Regexp {
	out := make([]*regexp.Regexp, 0, len(keys))
	for _, k := range keys {
		out = append(out, keyLineRe(k))
	}
	return out
}

func processFile(path string, res []*regexp.Regexp, dry bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	changed := false
	for i, line := range lines {
		for _, re := range res {
			m := re.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			newVal, did := canonicalizeValue(m[2])
			if !did {
				continue
			}
			if dry {
				fmt.Printf("%s\n  - %s\n  + %s%s\n", path, m[2], m[1], newVal)
			}
			lines[i] = m[1] + newVal
			changed = true
			break
		}
	}
	if changed && !dry {
		_ = os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
	}
}
```

> **Mob name depth note:** mob `name:` lives under `character:` (indented). The regex matches any indentation, so it will also match a top-level `name:` if one exists. In Step 1, confirm mob files have exactly one display `name:` (the character name) and no colliding top-level `name:`. If a collision exists, tighten the mob matcher to require the `character:`-block indentation.

- [ ] **Step 3: Dry-run and review the diff**

Run: `go run ./tools/casing_sweep -dry | tee /tmp/casing-sweep.txt`
Inspect: confirm only display names change, proper names are preserved (e.g. `Olen` stays `Olen`), no lookup/tag lines appear. Eyeball a sample across each dir.

- [ ] **Step 4: Apply the sweep**

Run: `go run ./tools/casing_sweep`
Then: `git diff --stat` and skim `git diff` for anything that touched a non-name line.

- [ ] **Step 5: Build + boot-load smoke (no validator yet)**

Run: `go build ./... && rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*`
Then boot (`go run .`), confirm clean data load + `Server Ready`, kill it.

- [ ] **Step 6: Commit**

```bash
git add tools/casing_sweep/main.go _datafiles/world/dogmud/
git commit -m "chore(casing): one-time sweep canonicalizing stored display names"
```

---

## Task 6: Load-time panic validator (the guardrail)

**Files:**
- Create: `internal/casing/validate.go` — a shared assert helper.
- Modify: each loader (mobs, items, rooms, spells, buffs) to call it after load.
- Test: `internal/casing/validate_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/casing/validate_test.go`:

```go
package casing

import "testing"

func TestAssertCanonical(t *testing.T) {
	// Canonical input: no panic.
	AssertCanonical("Temple Priest Olen", "mob", "371-tova.yaml")
	AssertCanonical("Captain of the Guard", "mob", "x.yaml")

	// Non-canonical: panics.
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on non-canonical name")
		}
	}()
	AssertCanonical("temple priest olen", "mob", "371-tova.yaml")
}
```

- [ ] **Step 2: Run the test; verify it fails**

Run: `go test ./internal/casing/ -run TestAssertCanonical -v`
Expected: FAIL — `AssertCanonical` undefined.

- [ ] **Step 3: Implement the validator helper**

Create `internal/casing/validate.go`:

```go
package casing

import "fmt"

// AssertCanonical panics if name is not already in canonical Title form. Call
// it from data loaders on player-visible display-name fields so non-canonical
// authored names fail fast at startup (caught by the pre-push boot test).
func AssertCanonical(name, kind, source string) {
	if got := Title(name); got != name {
		panic(fmt.Sprintf(
			"casing: non-canonical %s name in %s: %q (expected %q)",
			kind, source, name, got))
	}
}
```

- [ ] **Step 4: Run the test; verify it passes**

Run: `go test ./internal/casing/ -run TestAssertCanonical -v`
Expected: PASS.

- [ ] **Step 5: Wire the validator into each loader**

After each loader populates its registry (mobs, items, rooms, spells, buffs), iterate the loaded specs and call `casing.AssertCanonical(displayName, kind, sourceFile)` on the display-name field pinned in Task 5 Step 1. Use the loader's existing source-path/id for the `source` arg. Example shape (adapt to each loader's actual map + field):

```go
// in buffs.LoadDataFiles(), after `buffs = tmpBuffs`:
for id, b := range buffs {
	casing.AssertCanonical(b.Name, "buff", fmt.Sprintf("%d", id))
}
```

Do this for: `internal/mobs` (character name), `internal/items` (the field chosen in Task 5), `internal/rooms` (title), `internal/spells` (name), `internal/buffs` (name). Skip empty strings (`if name == "" { continue }`).

- [ ] **Step 6: Build + boot; verify validator passes on the swept data**

Run: `go build ./... && go run .` → confirm `Server Ready`, no casing panic. Kill server.
(If it panics, the offending file/value is printed — fix that name and re-run. This is the validator doing its job.)

- [ ] **Step 7: Commit**

```bash
git add internal/casing/validate.go internal/casing/validate_test.go internal/{mobs,items,rooms,spells,buffs}/
git commit -m "feat(casing): load-time panic validator on display names (guardrail)"
```

---

## Task 7: Prose / template literal audit

**Files:**
- Modify: room descriptions, combat-message templates, dialogue `text`/`hints` that hand-type an entity name with stray casing.

- [ ] **Step 1: Find candidate hand-typed names / stray casing**

Run (heuristic — flags multi-word Capitalized phrases and known role words in prose):
```bash
grep -rniE "temple priest|city guard|guard captain|captain of the" _datafiles/world/dogmud/rooms _datafiles/world/dogmud/dialogue _datafiles/combat-messages 2>/dev/null | head -60
```
Also skim combat-message templates for literal creature/role names (most use `{source}`/`{target}`/`{itemname}` tokens, which already carry canonical values — only literals need fixing).

- [ ] **Step 2: Fix literals to match canonical names; leave sentence prose alone**

For each hit that NAMES a specific entity (e.g., a room description that literally says "the temple priest"), decide: is it referring to the entity as a proper display name (→ canonicalize) or as generic prose ("a priest tends the altar" — leave it)? Only canonicalize literal references to a specific named/titled entity. Do NOT title-case ordinary sentence prose.

- [ ] **Step 3: Build + boot smoke**

Run: `go build ./... && go run .` → `Server Ready`, kill.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/
git commit -m "content(casing): canonicalize hand-typed entity names in prose/templates"
```

---

## Task 8: Full verification + push

- [ ] **Step 1: Full test suite**

Run: `go test ./...`
Expected: all green.

- [ ] **Step 2: Boot test (pre-push SOP)**

Run: `rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*` then `go run .`
Expected: clean load, `Server Ready`, no casing panic. Kill server + clear ports afterward.

- [ ] **Step 3: Targeting + display spot-check (manual)**

Connect locally and verify: `look`, `who`, a combat round, `inventory`, and a named NPC all show consistent Title-Case; and that targeting still works — `attack temple priest`, `get iron sword` (proves keys/tags were NOT canonicalized). Also check the web/GMCP client shows the same casing.

- [ ] **Step 4: PATCH_NOTES + push per SOP**

Add a PATCH_NOTES entry (player-facing: "Names and titles now use consistent capitalization throughout"). Confirm `Logging.LogToFile: false`. Commit PATCH_NOTES, then push per the normal SOP.

```bash
git add PATCH_NOTES.md
git commit -m "docs: PATCH_NOTES for game-wide naming casing convention"
```

---

## Notes / decisions captured from the spec

- **Names only, never keys:** the sweep + validator touch only the pinned display-name field(s) per loader. Lookup/keyword/tag/`namesimple`/filename/id fields are untouched, so targeting and parsing are unaffected. This is the #1 risk — guard it in Task 5 Step 1 and Task 8 Step 3.
- **Idempotence** lets the validator use `name == Title(name)` and lets display re-apply `Title` harmlessly.
- **Out of scope:** non-human mob attack types/messaging (separate spec); title-casing prose sentences; renaming keys/tags/ids.
