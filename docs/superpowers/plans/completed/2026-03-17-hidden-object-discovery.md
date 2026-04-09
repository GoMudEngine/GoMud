# Hidden Object Discovery System — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add hidden nouns/containers to rooms, consolidate tracking/foraging/search under one "search" skill with gaussian rolls, and migrate existing players.

**Architecture:** New `Search` skill replaces `Tracking` + `Foraging`. Room YAML gains `hidden_nouns` map and Container gains `Hidden` bool. Character gains `Discoveries` map for persistent per-room discovery state. All three commands (search, track, forage) use `dice.RollStat()` against tier difficulty targets (125/135/175).

**Tech Stack:** Go, YAML data files, goja JS scripting, ANSI terminal output

**Spec:** `docs/superpowers/specs/2026-03-17-hidden-object-discovery-design.md`

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `internal/skills/skills.go` | Modify | Add Search, remove Tracking/Foraging from all maps |
| `internal/characters/character.go` | Modify | Add Discoveries field, helpers, migration in Validate() |
| `internal/characters/character_test.go` | Modify | Tests for discoveries + migration |
| `internal/rooms/rooms.go` | Modify | Add HiddenNouns field to Room struct, FindHiddenNoun() |
| `internal/rooms/container.go` | Modify | Add Hidden field to Container |
| `internal/rooms/rooms_test.go` | Modify | Tests for hidden noun/container YAML parsing |
| `internal/usercommands/skill.search.go` | Modify | Full rewrite: gaussian rolls, tier discovery, bug fixes |
| `internal/usercommands/skill.track.go` | Modify | Replace tier system with gaussian rolls |
| `internal/usercommands/skill.forage.go` | Modify | Replace percentage roll with gaussian rolls |
| `internal/usercommands/look.go` | Modify | Hidden noun description append, hidden container filter |
| `_datafiles/world/dogmud/templates/help/search.template` | Modify | Updated help text |
| `_datafiles/world/dogmud/templates/help/track.template` | Modify | Note search skill |
| `_datafiles/world/dogmud/templates/help/foraging.template` | Modify | Note search skill |
| `_datafiles/world/dogmud/templates/help/skills.template` | Modify | Replace tracking/foraging with search |
| `docs/schemas/room.md` | Modify | Document hidden_nouns + container hidden |
| `internal/rooms/context.md` | Modify | HiddenNoun convention docs |
| `internal/usercommands/context.md` | Modify | Search roll mechanics docs |
| `internal/skills/context.md` | Modify | Search skill consolidation docs |

---

## Task 1: Add Search Skill to Skills Package

**Files:**
- Modify: `internal/skills/skills.go:26-49` (constants), `:54-102` (Professions), `:267-284` (SkillPrimaryStats), `:295-317` (SkillProgressionMultipliers), `:353-382` (init)

- [ ] **Step 1: Add Search constant, remove Tracking and Foraging**

In `internal/skills/skills.go`, add `Search` constant and remove `Tracking` and `Foraging`:

```go
// Line ~40-42: Replace these:
//   Tracking  SkillTag = `tracking`
//   Foraging  SkillTag = `foraging`
// With:
Search SkillTag = `search`
```

- [ ] **Step 2: Update Professions map**

Replace `Tracking` and `Foraging` references in the Professions map (lines 54-102):

```go
"ranger":      {RangedCombat, Search},      // was Tracking
"survivalist": {Search},                     // was Foraging, Tracking
```

- [ ] **Step 3: Update SkillPrimaryStats map**

In SkillPrimaryStats (lines 267-284), remove `"tracking"` and `"foraging"` entries, add `"search"`:

```go
"search": "perception",
// DELETE: "tracking": "perception",
// DELETE: "foraging": "perception",
```

- [ ] **Step 4: Update SkillProgressionMultipliers map**

In SkillProgressionMultipliers (lines 295-317), remove `Tracking` and `Foraging`, add `Search`:

```go
Search: 2.0,
// DELETE: Tracking: 2.0,
// DELETE: Foraging: 2.0,
```

- [ ] **Step 5: Update init() registration list**

In init() (lines 353-382), replace `Tracking` and `Foraging` with `Search` in the explicit skill list:

```go
for _, sk := range []SkillTag{
    Cast,
    WeaponCombat, UnarmedCombat, RangedCombat, Spellcasting, Rhetoric,
    FirstAid, Stealth, Search, Bartering,  // Search replaces Tracking
    Blacksmithing, Alchemy, Tailoring, Cooking, Jewelcrafting, Enchanting,
    // Foraging removed
} {
```

- [ ] **Step 6: Fix all compile errors from removed constants**

Search the codebase for `skills.Tracking` and `skills.Foraging` references. Update each to `skills.Search`. Key files:
- `internal/usercommands/skill.track.go` — all `skills.Tracking` refs
- `internal/usercommands/skill.forage.go` — all `skills.Foraging` refs
- `internal/usercommands/skill.search.go` — if any
- `internal/characters/character_test.go` — test references

- [ ] **Step 7: Build and verify**

Run: `go build ./...`
Expected: Clean build (no errors)

- [ ] **Step 8: Run existing tests**

Run: `go test ./internal/skills/... ./internal/characters/...`
Expected: All pass (some tests may need updating for removed skills)

- [ ] **Step 9: Commit**

```
git add internal/skills/skills.go internal/usercommands/skill.track.go \
  internal/usercommands/skill.forage.go internal/characters/character_test.go
git commit -m "refactor: consolidate tracking+foraging into search skill"
```

---

## Task 2: Character Discoveries Field + Skill Migration

**Files:**
- Modify: `internal/characters/character.go:80-112` (struct), `:2385-2442` (Validate)
- Modify: `internal/characters/character_test.go`

- [ ] **Step 1: Write tests for HasDiscovery/AddDiscovery**

In `internal/characters/character_test.go`, add:

```go
func TestCharacter_Discoveries(t *testing.T) {
    c := New()

    // No discoveries initially
    assert.False(t, c.HasDiscovery(100, "compartment"))

    // Add a discovery
    c.AddDiscovery(100, "compartment")
    assert.True(t, c.HasDiscovery(100, "compartment"))

    // Idempotent — adding again doesn't duplicate
    c.AddDiscovery(100, "compartment")
    assert.Equal(t, 1, len(c.Discoveries[100]))

    // Different room, different namespace
    assert.False(t, c.HasDiscovery(200, "compartment"))

    // Multiple discoveries in same room
    c.AddDiscovery(100, "chest")
    assert.True(t, c.HasDiscovery(100, "chest"))
    assert.Equal(t, 2, len(c.Discoveries[100]))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/characters/ -run TestCharacter_Discoveries -v`
Expected: FAIL — `HasDiscovery` and `AddDiscovery` not defined

- [ ] **Step 3: Add Discoveries field to Character struct**

In `internal/characters/character.go`, add after the `MiscData` field (~line 97):

```go
Discoveries  map[int][]string               `yaml:"discoveries,omitempty"`
```

Initialize in `New()` (~line 140, alongside other map inits):

```go
Discoveries:    make(map[int][]string),
```

- [ ] **Step 4: Implement HasDiscovery and AddDiscovery**

Add to `internal/characters/character.go`:

```go
// HasDiscovery returns true if the player has discovered the given
// noun/container key in the specified room.
func (c *Character) HasDiscovery(roomId int, key string) bool {
    if c.Discoveries == nil {
        return false
    }
    for _, k := range c.Discoveries[roomId] {
        if k == key {
            return true
        }
    }
    return false
}

// AddDiscovery records a discovery. No-op if already discovered.
func (c *Character) AddDiscovery(roomId int, key string) {
    if c.HasDiscovery(roomId, key) {
        return
    }
    if c.Discoveries == nil {
        c.Discoveries = make(map[int][]string)
    }
    c.Discoveries[roomId] = append(c.Discoveries[roomId], key)
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/characters/ -run TestCharacter_Discoveries -v`
Expected: PASS

- [ ] **Step 6: Write skill migration test**

```go
func TestCharacter_SkillMigration_TrackingAndForaging(t *testing.T) {
    c := New()
    // Simulate old character with tracking=5, foraging=3
    c.Skills["tracking"] = 5
    c.Skills["foraging"] = 3
    c.SkillUseCount["tracking"] = 100
    c.SkillUseCount["foraging"] = 50
    delete(c.Skills, "search") // Remove search so migration triggers

    c.Validate()

    assert.Equal(t, 5, c.Skills["search"], "search should be max(tracking, foraging)")
    assert.Equal(t, 150, c.SkillUseCount["search"], "use counts should be summed")
    assert.Zero(t, c.Skills["tracking"], "tracking should be removed")
    assert.Zero(t, c.Skills["foraging"], "foraging should be removed")
    assert.Zero(t, c.SkillUseCount["tracking"])
    assert.Zero(t, c.SkillUseCount["foraging"])
}

func TestCharacter_SkillMigration_ForagingOnly(t *testing.T) {
    c := New()
    c.Skills["foraging"] = 7
    c.SkillUseCount["foraging"] = 200
    delete(c.Skills, "search")
    delete(c.Skills, "tracking")

    c.Validate()

    assert.Equal(t, 7, c.Skills["search"])
    assert.Equal(t, 200, c.SkillUseCount["search"])
    assert.Zero(t, c.Skills["foraging"])
}

func TestCharacter_SkillMigration_Idempotent(t *testing.T) {
    c := New()
    c.Skills["search"] = 10
    c.SkillUseCount["search"] = 500

    c.Validate()

    assert.Equal(t, 10, c.Skills["search"], "should not change existing search")
    assert.Equal(t, 500, c.SkillUseCount["search"])
}
```

- [ ] **Step 7: Run migration tests to verify they fail**

Run: `go test ./internal/characters/ -run TestCharacter_SkillMigration -v`
Expected: FAIL — migration not yet implemented

- [ ] **Step 8: Implement skill migration in Validate()**

In `internal/characters/character.go`, in `Validate()`, add **before** the `ensureAllSkills` call (~line 2442):

```go
// Migrate tracking/foraging → search (must run before ensureAllSkills)
if _, hasTracking := c.Skills["tracking"]; hasTracking {
    trackRank := c.Skills["tracking"]
    forageRank := c.Skills["foraging"]
    c.Skills["search"] = max(trackRank, forageRank)
    if c.Skills["search"] < 1 {
        c.Skills["search"] = 1
    }
    if c.SkillUseCount == nil {
        c.SkillUseCount = make(map[string]int)
    }
    c.SkillUseCount["search"] = c.SkillUseCount["tracking"] +
        c.SkillUseCount["foraging"]
    delete(c.Skills, "tracking")
    delete(c.Skills, "foraging")
    delete(c.SkillUseCount, "tracking")
    delete(c.SkillUseCount, "foraging")
} else if _, hasForaging := c.Skills["foraging"]; hasForaging {
    c.Skills["search"] = max(c.Skills["foraging"], 1)
    if c.SkillUseCount == nil {
        c.SkillUseCount = make(map[string]int)
    }
    c.SkillUseCount["search"] = c.SkillUseCount["foraging"]
    delete(c.Skills, "foraging")
    delete(c.SkillUseCount, "foraging")
}
```

- [ ] **Step 9: Run all character tests**

Run: `go test ./internal/characters/ -v`
Expected: All pass

- [ ] **Step 10: Commit**

```
git add internal/characters/character.go internal/characters/character_test.go
git commit -m "feat: add character discoveries field + skill migration"
```

---

## Task 3: Room Data Model — HiddenNoun + Container.Hidden

**Files:**
- Modify: `internal/rooms/rooms.go:65-101` (Room struct), `:1624+` (FindNoun)
- Modify: `internal/rooms/container.go` (Container struct)

- [ ] **Step 1: Add HiddenNoun struct to rooms package**

Add to `internal/rooms/rooms.go` (or a new `internal/rooms/hidden_noun.go` if preferred):

```go
// HiddenNoun represents a discoverable noun in a room that is invisible
// until found via the search command (tier 3).
type HiddenNoun struct {
    Description       string `yaml:"description"`
    HiddenDescription string `yaml:"hidden_description"`
}
```

- [ ] **Step 2: Add HiddenNouns field to Room struct**

In the Room struct (lines 65-101), add after the `Nouns` field:

```go
HiddenNouns   map[string]HiddenNoun         `yaml:"hidden_nouns,omitempty" instance:"skip"`
```

- [ ] **Step 3: Add Hidden field to Container struct**

In `internal/rooms/container.go`, add to the Container struct:

```go
Hidden       bool          `yaml:"hidden,omitempty"`
```

- [ ] **Step 4: Add FindHiddenNoun method**

Add to `internal/rooms/rooms.go`:

```go
// FindHiddenNoun looks up a hidden noun by key. Returns the noun and
// true if found, zero value and false if not.
func (r *Room) FindHiddenNoun(key string) (HiddenNoun, bool) {
    if r.HiddenNouns == nil {
        return HiddenNoun{}, false
    }
    // Try exact match first, then fuzzy
    if hn, ok := r.HiddenNouns[key]; ok {
        return hn, true
    }
    // Build candidate list for fuzzy matching
    keys := make([]string, 0, len(r.HiddenNouns))
    for k := range r.HiddenNouns {
        keys = append(keys, k)
    }
    exact, close := util.FindMatchIn(key, keys...)
    if exact != "" {
        return r.HiddenNouns[exact], true
    }
    if close != "" {
        return r.HiddenNouns[close], true
    }
    return HiddenNoun{}, false
}
```

- [ ] **Step 5: Build and verify**

Run: `go build ./...`
Expected: Clean build

- [ ] **Step 6: Commit**

```
git add internal/rooms/rooms.go internal/rooms/container.go
git commit -m "feat: add HiddenNoun struct and Container.Hidden field"
```

---

## Task 4: Search Command Refactor

**Files:**
- Modify: `internal/usercommands/skill.search.go` (full rewrite)

- [ ] **Step 1: Rewrite search command with gaussian rolls**

Replace the entire body of `skill.search.go` with the new implementation. Key changes:
- Import `dice` and `combat` packages
- Compute `searchScore` from Perception + SkillMultiplier
- Replace hard tier thresholds with per-discovery `dice.RollStat()` rolls
- Fix stashed items gate (was tier 3+, now tier 2 / target 135)
- Fix hidden mob detection bug (`hiddenPlayers` → `hiddenMobs` variable)
- Fix mob lookup (`users.GetByUserId` → `mobs.GetInstance`)
- Add tier 3 hidden noun discovery loop
- Add hidden container discovery at tier 1
- Track `rolledAgainstSomething` flag for anti-botting progression gate
- Fire `CheckSkillProgression("search", ...)` only if rolled against something
- Fix typo in header comment ("Searcg" → "Search")

The search score formula:
```go
searchRank := user.Character.GetSkillLevel(skills.Search)
searchScore := user.Character.Stats.Perception.ValueAdj +
    int(combat.SkillMultiplier(searchRank) * 25.0)
```

- [ ] **Step 2: Build and verify**

Run: `go build ./...`
Expected: Clean build

- [ ] **Step 3: Commit**

```
git add internal/usercommands/skill.search.go
git commit -m "feat: rewrite search command with gaussian rolls + hidden noun discovery"
```

---

## Task 5: Track Command Rework

**Files:**
- Modify: `internal/usercommands/skill.track.go`

- [ ] **Step 1: Replace tier system with gaussian rolls**

Key changes to `skill.track.go`:
- Remove `skillLevel := user.Character.GetSkillLevel(skills.Tracking)` and the `skillLevel == 0` gate
- Compute `searchScore` same as search command
- One `dice.RollStat(float64(searchScore))` roll per use
- Roll value >= 125: show most recent visitor only
- Roll value >= 135: show all visitors
- Roll value >= 175: show exit directions + enable targeted/active tracking
- Replace all `skills.Tracking` references with `skills.Search`
- Fire `CheckSkillProgression("search", ...)` on every use
- The `track <target>` syntax requires the roll to beat 175; if not, show "Your tracking skills aren't sharp enough right now"

- [ ] **Step 2: Build and verify**

Run: `go build ./...`
Expected: Clean build

- [ ] **Step 3: Commit**

```
git add internal/usercommands/skill.track.go
git commit -m "feat: rework track command to use gaussian rolls + search skill"
```

---

## Task 6: Forage Command Rework

**Files:**
- Modify: `internal/usercommands/skill.forage.go`

- [ ] **Step 1: Replace percentage roll with gaussian roll**

Key changes to `skill.forage.go`:
- Remove the old formula: `20 + (forageSkill * 5) + ceil(Perception/10)`
- Compute `searchScore` same as search/track
- Add biome difficulty map:
```go
var forageDifficulty = map[string]float64{
    "farmland":  110,
    "forest":    120,
    "land":      125,
    "swamp":     130,
    "shore":     135,
    "cave":      135,
    "mountains": 140,
    "cliffs":    145,
}
```
- Roll `dice.RollStat(float64(searchScore))` vs biome difficulty
- Replace `skills.Foraging` references with `skills.Search`
- Replace `OnSkillUse("foraging", ...)` with `CheckSkillProgression("search", ...)`

- [ ] **Step 2: Build and verify**

Run: `go build ./...`
Expected: Clean build

- [ ] **Step 3: Commit**

```
git add internal/usercommands/skill.forage.go
git commit -m "feat: rework forage command to use gaussian rolls + search skill"
```

---

## Task 7: Look Command Integration

**Files:**
- Modify: `internal/usercommands/look.go:153` (container lookup), `:317` (noun lookup), `:522-526` (room description)

- [ ] **Step 1: Add hidden noun description to room look output**

After the room description is rendered (~line 526), add logic to append discovered hidden noun descriptions:

```go
// Append discovered hidden noun descriptions
if user != nil && room.HiddenNouns != nil {
    // Sort keys for deterministic order
    hiddenKeys := make([]string, 0, len(room.HiddenNouns))
    for k := range room.HiddenNouns {
        hiddenKeys = append(hiddenKeys, k)
    }
    sort.Strings(hiddenKeys)
    for _, key := range hiddenKeys {
        if user.Character.HasDiscovery(room.RoomId, key) {
            hn := room.HiddenNouns[key]
            if hn.HiddenDescription != "" {
                user.SendText(hn.HiddenDescription)
            }
        }
    }
}
```

- [ ] **Step 2: Add hidden noun lookup to `look <noun>` flow**

At ~line 317, after the regular `FindNoun` call, if no match found, check hidden nouns:

```go
foundNoun, foundDesc := room.FindNoun(lookAt)
if foundNoun == "" && user != nil {
    // Check hidden nouns (only if player has discovered them)
    if hn, ok := room.FindHiddenNoun(lookAt); ok {
        // Find the matching key for discovery check
        for key := range room.HiddenNouns {
            if hn2, _ := room.FindHiddenNoun(key); hn2 == hn {
                if user.Character.HasDiscovery(room.RoomId, key) {
                    foundNoun = key
                    foundDesc = hn.Description
                }
                break
            }
        }
    }
}
```

- [ ] **Step 3: Filter hidden containers from look/open/get**

At ~line 153, where `FindContainerByName` is called, add a discovery check after finding the container:

```go
containerName := room.FindContainerByName(lookAt)
if containerName != "" {
    container := room.Containers[containerName]
    if container.Hidden && (user == nil || !user.Character.HasDiscovery(room.RoomId, containerName)) {
        containerName = "" // Treat as not found
    }
}
```

Search for ALL other call sites of `FindContainerByName` in the codebase and add the same hidden check. Key commands: `open`, `get`, `put`, `close`.

- [ ] **Step 4: Build and verify**

Run: `go build ./...`
Expected: Clean build

- [ ] **Step 5: Commit**

```
git add internal/usercommands/look.go
git commit -m "feat: integrate hidden nouns + containers into look command"
```

---

## Task 8: Help Files + Skill List Updates

**Files:**
- Modify: `_datafiles/world/dogmud/templates/help/search.template`
- Modify: `_datafiles/world/dogmud/templates/help/track.template`
- Modify: `_datafiles/world/dogmud/templates/help/foraging.template`
- Modify: `_datafiles/world/dogmud/templates/help/skills.template`

- [ ] **Step 1: Update search help file**

Rewrite to describe the new search skill and roll-based system. Mention that it covers searching rooms, and is the governing skill for track and forage as well.

- [ ] **Step 2: Update track help file**

Add note that tracking now uses the Search skill for progression.

- [ ] **Step 3: Update forage help file**

Add note that foraging now uses the Search skill for progression.

- [ ] **Step 4: Update skills list help file**

Replace tracking and foraging entries with a single search entry.

- [ ] **Step 5: Commit**

```
git add _datafiles/world/dogmud/templates/help/
git commit -m "docs: update help files for search skill consolidation"
```

---

## Task 9: Documentation + Context Updates

**Files:**
- Modify: `docs/schemas/room.md`
- Modify: `internal/rooms/context.md`
- Modify: `internal/usercommands/context.md`
- Modify: `internal/skills/context.md`

- [ ] **Step 1: Update room schema**

Add `hidden_nouns` field and `hidden` container field to `docs/schemas/room.md` with YAML examples.

- [ ] **Step 2: Update rooms context.md**

Document the `HiddenNoun` struct, `hidden_nouns` field, hidden container field, and the authoring convention:
- Hidden contents reference parent nouns via prose, no formal parent link
- `description` = what `look <noun>` shows after discovery
- `hidden_description` = appended to room description for discoverers
- `hidden_nouns` is `instance:"skip"` (template-driven)

- [ ] **Step 3: Update usercommands context.md**

Document search command changes: gaussian roll mechanics, tier targets (125/135/175), anti-botting progression gate, track/forage rework summary.

- [ ] **Step 4: Update skills context.md**

Document search skill consolidation: replaces tracking + foraging, same governing stat (Perception), migration details for existing players.

- [ ] **Step 5: Commit**

```
git add docs/schemas/room.md internal/rooms/context.md \
  internal/usercommands/context.md internal/skills/context.md
git commit -m "docs: update context files for hidden object discovery system"
```

---

## Task 10: Tutorial Area Audit

**Files:**
- Audit: `_datafiles/world/dogmud/dialogue/ashwick/259.yaml` (Delia, wilderness guide NPC)
- Audit: any other tutorial NPC dialogue referencing tracking/foraging

- [ ] **Step 1: Search for tracking/foraging references in tutorial content**

```
grep -ri "tracking\|foraging" _datafiles/world/dogmud/dialogue/
grep -ri "tracking\|foraging" _datafiles/world/dogmud/templates/help/
```

- [ ] **Step 2: Update NPC dialogue**

Replace references to "tracking skill" with "search skill" and "foraging skill" with "search skill" in any tutorial NPC dialogue.

- [ ] **Step 3: Verify no broken references**

Build and manually review affected dialogue files.

- [ ] **Step 4: Commit**

```
git add _datafiles/world/dogmud/dialogue/
git commit -m "fix: update tutorial NPC dialogue for search skill consolidation"
```

---

## Task 11: Final Verification

- [ ] **Step 1: Full build**

Run: `go build ./...`
Expected: Clean build

- [ ] **Step 2: Full test suite**

Run: `go test ./internal/...`
Expected: All pass (except pre-existing failures unrelated to this work)

- [ ] **Step 3: Grep for stale references**

```
grep -ri "skills\.Tracking\|skills\.Foraging" internal/
grep -ri "\"tracking\"\|\"foraging\"" internal/skills/
```

Expected: No matches in Go source (only in test fixtures, YAML data, or comments explaining the migration)

- [ ] **Step 4: Update CLAUDE.md if needed**

If any new project conventions were established (e.g., the hidden noun authoring pattern), add them to CLAUDE.md.

- [ ] **Step 5: Final commit**

```
git commit -m "chore: final verification for hidden object discovery system"
```
