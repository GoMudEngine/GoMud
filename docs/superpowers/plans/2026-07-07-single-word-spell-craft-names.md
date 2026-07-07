# Single-Word Spell/Craft Invocation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let players invoke every spell and craft recipe by a short single-word alias (`cast ward`), while keeping the canonical hyphenated ids (zero ripple), and upgrade the `cast` parser to also accept full multi-word names (`cast conviction ward`).

**Architecture:** Add an additive `Aliases []string` field to `SpellData` and `RecipeSpec`; build an alias index at load with a uniqueness validator; a shared resolver (`id → alias → display-name`) used by the `cast` command, the bare-spell dispatcher fallback, and craft's `FindRecipeByName`. Upgrade the `cast` parser to greedy longest-match (what `craft` already does). Then curate + add collision-free aliases to all 59 spell + 126 recipe YAMLs.

**Tech Stack:** Go (GoMud). Unit tests via `go test` (`internal/spells`, `internal/crafting`, `internal/usercommands` have `_test.go`). Boot-validation + in-game `cast`/`craft` checks.

**Spec:** `docs/superpowers/specs/2026-07-07-single-word-spell-craft-names-design.md`

---

## Verified code facts (2026-07-07)

- **`SpellData`** (`internal/spells/spells.go:19`): `SpellId string` (yaml `spellid`), `Name string`, etc. `allSpells map[string]*SpellData` keyed by spellid. `GetSpell(id)` (:141) = exact `allSpells[id]`. `FindSpellByName(name)` (:148) = matches lowercased `Name` exact, then prefix — **so multi-word display names ("conviction ward") already resolve here.** `LoadSpellFiles()` (:324) loads + asserts casing + sets `allSpells` — the index-build/validate hook.
- **`cast` parser** (`internal/usercommands/skill.cast.go`): `parts := strings.SplitN(rest, " ", 2)` → `spellName = parts[0]` (first token only), `targetName = parts[1]`. Resolves `GetSpell(spellName)` then `FindSpellByName(spellName)`. **The first-token split is the multi-word blocker.**
- **Bare-spell dispatcher fallback** (`internal/usercommands/usercommands.go:522`): `if user.Character.HasSpell(cmd) { castCmd := cmd; if rest != "" { castCmd += " " + rest }; return Cast(castCmd, ...) }`. `HasSpell(cmd)` is keyed by spellid — so a bare alias needs this gate made alias-aware.
- **`RecipeSpec`** (`internal/crafting/crafting.go:32`): `RecipeId string` (yaml `id`), `Name string`. `allRecipes map[string]*RecipeSpec` keyed by RecipeId. `FindRecipeByName(name)` (:126) = lowercased `Name` exact, then substring. `LoadRecipeFiles()` (:~85) sets `allRecipes` — the recipe index/validate hook.
- **`craft` parser** (`internal/usercommands/craft.go:32-45`): ALREADY greedy — tries `FindRecipeByName(rest)`, then progressively shorter candidates (recipe vs trailing item-target). So once `FindRecipeByName` matches aliases, `craft <alias> <item>` works.
- **`spells` list** (`internal/usercommands/spells.go:73`): iterates `user.Character.GetSpells()` → `spells.GetSpell(spellId)` for display — where the alias gets surfaced.
- Validator precedent: quest-flag `ValidateAllFlags` panics at startup on collisions.

---

## Task 1: Spell alias field + index + resolver + validator

**Files:**
- Modify: `internal/spells/spells.go`
- Test: `internal/spells/spells_alias_test.go` (create)

- [ ] **Step 1: Write failing tests.**

```go
func TestSpellAliasResolution(t *testing.T) {
	// Requires a couple of loaded spells with aliases; if the test harness can
	// load real spell files, use conviction-ward (alias "ward"); else inject
	// into allSpells via a small test seam. Grep spells_test.go for how it seeds.
	if ResolveSpellId("conviction-ward") != "conviction-ward" {
		t.Fatal("canonical id must resolve to itself")
	}
	if ResolveSpellId("ward") != "conviction-ward" {
		t.Fatal("alias must resolve to canonical id")
	}
	if ResolveSpellId("conviction ward") != "conviction-ward" {
		t.Fatal("multi-word display name must resolve to canonical id")
	}
	if ResolveSpellId("nonesuch") != "" {
		t.Fatal("unknown must resolve to empty")
	}
}
```

- [ ] **Step 2: Run — expect FAIL** (`ResolveSpellId` undefined).

Run: `go test ./internal/spells/ -run TestSpellAliasResolution`
Expected: FAIL.

- [ ] **Step 3: Add the field, index, resolver, validator.**
  - Add to `SpellData` (near `Name`): `Aliases []string \`yaml:"aliases,omitempty"\`` (lowercase single-word invocation forms; primary first).
  - Add a package var `spellsByAlias map[string]*SpellData`.
  - In `LoadSpellFiles()`, after `allSpells = tmpAllSpells`, build the index + validate:

```go
spellsByAlias = make(map[string]*SpellData, len(allSpells))
for _, s := range allSpells {
	for _, a := range s.Aliases {
		a = strings.ToLower(strings.TrimSpace(a))
		if a == "" {
			continue
		}
		if _, clash := allSpells[a]; clash {
			panic(fmt.Sprintf("spell alias %q (on %s) collides with a spellid", a, s.SpellId))
		}
		if other, dup := spellsByAlias[a]; dup {
			panic(fmt.Sprintf("duplicate spell alias %q on %s and %s", a, other.SpellId, s.SpellId))
		}
		spellsByAlias[a] = s
	}
}
```
  - Add the resolver:

```go
// ResolveSpell resolves an input token to a spell by canonical id, then alias,
// then full display name (case-insensitive). Returns nil if none match.
func ResolveSpell(token string) *SpellData {
	token = strings.ToLower(strings.TrimSpace(token))
	if sd, ok := allSpells[token]; ok {
		return sd
	}
	if sd, ok := spellsByAlias[token]; ok {
		return sd
	}
	return FindSpellByName(token)
}

// ResolveSpellId returns the canonical spellid for a token (id/alias/name), or "".
func ResolveSpellId(token string) string {
	if sd := ResolveSpell(token); sd != nil {
		return sd.SpellId
	}
	return ""
}
```
Add `fmt`/`strings` imports if missing.

- [ ] **Step 4: Run — expect PASS** (+ the spells package).

Run: `go test ./internal/spells/`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
git add internal/spells/spells.go internal/spells/spells_alias_test.go
git commit -m "feat(qol): spell Aliases field + alias index + ResolveSpell resolver + uniqueness validator"
```

---

## Task 2: Cast greedy-parser + alias-aware bare-spell fallback

**Files:**
- Modify: `internal/usercommands/skill.cast.go`
- Modify: `internal/usercommands/usercommands.go` (the bare-spell fallback ~line 522)

- [ ] **Step 1: Greedy longest-match in `skill.cast.go`.** Replace the `SplitN(rest, " ", 2)` first-token parse with progressive shortening: split `rest` into words; try the whole thing as a spell, then drop the last word and retry, until `spells.ResolveSpell(candidate)` matches — the leftover trailing words are the target.

```go
words := strings.Fields(rest)
var spellInfo *spells.SpellData
targetName := ""
for n := len(words); n >= 1; n-- {
	candidate := strings.Join(words[:n], " ")
	if sd := spells.ResolveSpell(candidate); sd != nil {
		spellInfo = sd
		targetName = strings.TrimSpace(strings.Join(words[n:], " "))
		break
	}
}
if spellInfo == nil {
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="red">No spell found for "%s".</ansi>`, rest))
	return true, nil
}
spellName := spellInfo.SpellId // downstream InitiateCast/lookup use the canonical id
```
Keep the rest of `Cast` (skill check, InitiateCast) using `spellName`/`spellInfo` as before — but pass the CANONICAL `spellInfo.SpellId` downstream so InitiateCast's re-lookup hits. Strip a leading `on`/`at` from `targetName` if the existing target resolver doesn't already (grep how targetName is consumed; likely fine).

- [ ] **Step 2: Alias-aware bare-spell fallback in `usercommands.go`.** At ~line 522, make the gate resolve aliases to a canonical spellid before `HasSpell`:

```go
if spellId := spells.ResolveSpellId(cmd); spellId != "" && user.Character.HasSpell(spellId) {
	castCmd := cmd
	if len(rest) > 0 {
		castCmd += ` ` + rest
	}
	return Cast(castCmd, user, room, flags)
}
```
(So bare `ward goblin` → `ResolveSpellId("ward")` = "conviction-ward" → HasSpell true → `Cast("ward goblin")` → the greedy parser resolves it.) Confirm `spells` is imported in usercommands.go.

- [ ] **Step 3: TDD where possible + build.** Add a `skill.cast` unit test only if the harness supports it cheaply (Cast needs a full UserRecord/room); otherwise rely on the Task 1 resolver test + build + the in-game check in Task 8. Run:

Run: `go build -o gomud_smoke.exe . && go vet ./internal/usercommands/ && echo OK`
Expected: `OK`.

- [ ] **Step 4: Commit.**

```bash
git add internal/usercommands/skill.cast.go internal/usercommands/usercommands.go
git commit -m "feat(qol): cast greedy longest-match parser + alias-aware bare-spell invocation"
```

---

## Task 3: Recipe alias field + resolver + validator

**Files:**
- Modify: `internal/crafting/crafting.go`
- Test: `internal/crafting/crafting_alias_test.go` (create)

- [ ] **Step 1: Write failing test.**

```go
func TestRecipeAliasResolution(t *testing.T) {
	// Seed a recipe with an alias (grep crafting_test.go / how allRecipes is set
	// in tests) or load real files; assert FindRecipeByName matches the alias.
	r := FindRecipeByName("quench") // alias for anti-corrosion-quench
	if r == nil || r.RecipeId != "anti-corrosion-quench" {
		t.Fatalf("alias did not resolve: %+v", r)
	}
}
```

- [ ] **Step 2: Run — expect FAIL.**

Run: `go test ./internal/crafting/ -run TestRecipeAliasResolution`
Expected: FAIL.

- [ ] **Step 3: Add the field + alias matching + validator.**
  - Add to `RecipeSpec`: `Aliases []string \`yaml:"aliases,omitempty"\``.
  - In `FindRecipeByName`, add an alias pass (after the exact-Name pass, before substring): match any recipe whose `Aliases` contains the lowercased input.
  - In `LoadRecipeFiles()` (after `allRecipes = tmpAll`), validate alias uniqueness within the recipe namespace + no collision with any `RecipeId` — panic on violation (mirror Task 1's validator).

- [ ] **Step 4: Run — expect PASS.**

Run: `go test ./internal/crafting/`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
git add internal/crafting/crafting.go internal/crafting/crafting_alias_test.go
git commit -m "feat(qol): recipe Aliases field + FindRecipeByName alias match + uniqueness validator"
```

---

## Task 4: Curate the alias mapping (spells + recipes) — REVIEW GATE

**Files:**
- Create: `docs/superpowers/notes/2026-07-07-invocation-alias-map.md` (the mapping table, for review)

- [ ] **Step 1: Generate the spell alias map (59).** For each spell, pick a natural
  lowercase single word, unique within the spell namespace and not equal to any
  spellid. Examples: `conviction-ward`→`ward`, `chrysalis-glow`→`glow`,
  `kinetic-shove`→`shove`, `conjure-fire`→`fire`, `chrysalis-regeneration`→`regen`,
  `identify`→`id`, `heal`→`heal`. Resolve collisions deliberately (e.g. if two
  spells both want `shield`, give the second a distinct word). List every spell:
  `spellid | display name | proposed alias`.

- [ ] **Step 2: Generate the recipe alias map (126)**, same rules, unique within the
  recipe namespace. (Spell and recipe aliases may overlap — separate commands.)

- [ ] **Step 3: Write both tables to the notes file and PRESENT TO THE USER for
  review.** The player approves/edits the alias words before they're written into
  YAMLs (the spec calls for a player-reviewed list). Apply any edits.

- [ ] **Step 4: Commit the approved map.**

```bash
git add docs/superpowers/notes/2026-07-07-invocation-alias-map.md
git commit -m "docs(qol): approved single-word alias map for spells + recipes"
```

---

## Task 5: Apply spell aliases to the 59 spell YAMLs

**Files:**
- Modify: all `_datafiles/world/dogmud/spells/*.yaml`

- [ ] **Step 1: Add `aliases:` to each spell YAML** per the approved map, e.g. to
  `conviction-ward.yaml`:

```yaml
spellid: conviction-ward
name: Conviction Ward
aliases:
  - ward
```
Add the block after `name:`. (Scriptable: for each `spellid → alias`, insert the
`aliases:` block — but validate each file parses.)

- [ ] **Step 2: Validate all parse.**

Run: `python -c "import glob,yaml; [yaml.safe_load(open(f)) for f in glob.glob('_datafiles/world/dogmud/spells/*.yaml')]" && echo OK`
Expected: `OK`.

- [ ] **Step 3: Boot — the alias validator must pass (no duplicate/collision panic).**

Run: `rm -rf _datafiles/world/dogmud/mobs.instances/* && go build -o gomud_smoke.exe .` then boot and confirm `spells.loadAllSpells() loadedCount=59` with **no panic**. (A collision would panic here — that's the validator working.)

- [ ] **Step 4: Commit.**

```bash
git add _datafiles/world/dogmud/spells/
git commit -m "content(qol): single-word aliases on all 59 spells"
```

---

## Task 6: Apply recipe aliases to the 126 recipe YAMLs

**Files:**
- Modify: all `_datafiles/world/dogmud/recipes/**/*.yaml`

- [ ] **Step 1: Add `aliases:` to each recipe YAML** per the approved map (after
  `name:`).

- [ ] **Step 2: Validate all parse.**

Run: `python -c "import glob,yaml; [yaml.safe_load(open(f)) for f in glob.glob('_datafiles/world/dogmud/recipes/**/*.yaml', recursive=True)]" && echo OK`
Expected: `OK`.

- [ ] **Step 3: Boot — recipe validator passes.**

Run: boot the smoke server; confirm `crafting.LoadRecipeFiles() loadedCount=126` with no panic.

- [ ] **Step 4: Commit.**

```bash
git add _datafiles/world/dogmud/recipes/
git commit -m "content(qol): single-word aliases on all 126 recipes"
```

---

## Task 7: Discoverability — show the alias in `spells` list + help

**Files:**
- Modify: `internal/usercommands/spells.go`
- Modify: the `spells`/`help` templates if the list is templated (grep)

- [ ] **Step 1: Surface the alias in the `spells` list.** In `spells.go` where each
  known spell is rendered (`spells.GetSpell(spellId)`), append the primary alias,
  e.g. `Conviction Ward (ward)`. If rendering is templated
  (`_datafiles/world/dogmud/templates/`), pass the alias into the template data.

- [ ] **Step 2: (Optional) per-spell help.** If `help <spell>` renders spell data,
  include the alias line. Skip if it complicates the help-completeness test.

- [ ] **Step 3: Build + verify the help-completeness test still passes.**

Run: `go build -o gomud_smoke.exe . && go test ./internal/usercommands/ 2>&1 | tail -3`
Expected: PASS.

- [ ] **Step 4: Commit.**

```bash
git add internal/usercommands/spells.go
git commit -m "feat(qol): show spell alias in the spells list"
```

---

## Task 8: Full verification + memory

- [ ] **Step 1: Full suite + clean boot.**

Run: `go test ./... 2>&1 | grep -vE "^ok|no test files" | head` (no new failures)
then `rm -rf _datafiles/world/dogmud/mobs.instances/* && go build -o gomud_smoke.exe .`, boot, confirm spells 59 / recipes 126 loaded, no panic, `ValidateZoneConsistency errors=0`.

- [ ] **Step 2: In-game verification** (mudagent, an admin caster char — use
  smoketester; grant/learn a spell if needed). Confirm all forms cast:
  - `cast ward` (alias), `cast conviction-ward` (canonical), `cast conviction ward`
    (multi-word display) — all cast Conviction Ward.
  - Bare `ward` (no `cast`) casts it too (dispatcher fallback).
  - `cast ward goblin` targets the goblin.
  - `craft <alias>` resolves a recipe (with a known recipe).
  - The `spells` list shows the alias.
  - **Deliberate-collision check:** temporarily add a duplicate alias to two spells
    → boot → confirm it PANICS (validator works) → revert.

- [ ] **Step 3: Update memory.** Mark [[project_single_word_spell_craft_names]] BUILT;
  note the alias approach (canonical ids untouched, zero migration), the
  ResolveSpell resolver, the cast greedy parser + dispatcher fallback, and that a
  prod push + pre-push SOP is owed (user pushes).

- [ ] **Step 4: Commit any doc/memory changes.**

```bash
git add docs/
git commit -m "docs(qol): single-word invocation verification notes"
```

---

## Notes for the executor

- **Zero ripple is the invariant:** never change a `spellid` or recipe `id`, a
  filename, or a player-save key. Aliases are purely additive.
- **Separate namespaces:** spell aliases and recipe aliases are validated
  independently; the same word may be a spell alias and a recipe alias.
- **The validator is the safety net:** a duplicate/colliding alias must PANIC at
  load (Task 1/3), so Tasks 5/6's boot step is the real collision check for the
  curated list.
- **Downstream canonical id:** after the greedy parse, hand `spellInfo.SpellId`
  (canonical) to InitiateCast so the existing spellbook/cooldown/quest plumbing
  (all keyed by spellid) is untouched.
