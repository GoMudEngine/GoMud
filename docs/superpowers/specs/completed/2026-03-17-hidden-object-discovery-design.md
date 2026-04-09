# Hidden Object Discovery System — Design Spec

**Date:** 2026-03-17
**Status:** Approved

## Overview

Add a hidden object discovery mechanism to DOGMud. Players with sufficient
Perception and Search skill can find hidden nouns, hidden containers, and
hidden contents within rooms. Discoveries persist permanently per-character.

This work also consolidates the tracking, foraging, and search commands
under a single new "search" skill, replaces hard stat thresholds with
gaussian dice rolls, and fixes several existing bugs in the search command.

## Goals

- Give content designers a way to hide nouns and containers in rooms that
  require active effort to discover
- Reward Perception investment and Search skill progression
- Consolidate three overlapping Perception-based commands under one skill
- Migrate existing players cleanly with no lost progression
- Establish the `hidden_nouns` YAML pattern for future description overlays
  (seasonal, weather, etc.)

## Non-Goals

- Room object hiding by players (player-placed hidden objects)
- Automated discovery without using the search command
- Visual/UI changes beyond text output

---

## 1. New "Search" Skill

### Skill Registration

Add `Search` to the skills package as a new `SkillTag` constant. It replaces
`Tracking` and `Foraging`, which are removed from the registry.

| Property | Value |
|----------|-------|
| Tag | `search` |
| Governing stat | Perception |
| Soft cap | 50 (same as all skills) |
| Progression multiplier | 2.0 (same as tracking/foraging had) |
| Starting rank | 1 (for new characters) |

### Commands Using Search Skill

All three commands trigger `OnSkillUse("search")` and
`CheckSkillProgression("search", ...)` on every use:

- **`search`** — find hidden things in the current room
- **`track`** — find creature/player trails
- **`forage`** — gather resources (herbs, wood, ore, food)

Each command retains its own gameplay logic. Only the skill they progress
is unified.

### Skill Migration (Existing Players)

Migration runs in `Character.Validate()`, **before** `ensureAllSkills()`.
This is critical — `ensureAllSkills` will add `search` at rank 1 if missing,
which would mask the old skill data. The migration guard checks for the
presence of old skills (not the absence of the new one):

1. If character has `tracking` or `foraging` in their skill map:
   - Set `search` rank = `max(tracking rank, foraging rank)`
   - If result < 1, set to 1
   - Sum `SkillUseCount` entries for tracking + foraging into search
   - Remove `tracking` and `foraging` from skill map and use count map
2. Migration is idempotent — safe to run on every load (old keys won't
   exist after first run)

Rationale for max (not sum): players should not lose progress, but summing
ranks would give an unfair jumpstart. Max preserves the best investment.

---

## 2. Roll Mechanics

### Combined Score

```
searchScore = Perception + SkillMultiplier(searchRank) * scaleFactor
```

`SkillMultiplier` follows the existing sqrt curve:
`mult = base + (max - base) * sqrt(rank / softCap)` with config defaults
base=1.0, max=3.0.

The `scaleFactor` should be tuned so that the skill contribution is
meaningful but doesn't dwarf Perception. Recommend starting at 25.0,
giving a range of 25 (rank 0) to 75 (rank 50) added to Perception.

### Per-Discovery Rolls

Each discoverable thing in the room gets its own independent roll:

```
roll = dice.RollStat(float64(searchScore))
```

The roll result is `roll.Value` (float64). It is compared against the tier
difficulty target for that discovery type. If `roll.Value >= target`, the
discovery succeeds.

Note: `dice.RollStat` takes `float64`, so `searchScore` must be cast.
`RollResult.Value` is the field name (not `Result`).

### Tier Difficulty Targets

| Tier | Target | Discovers |
|------|--------|-----------|
| 1 | 125 | Secret exits, hidden containers |
| 2 | 135 | Stashed items, hidden players/mobs |
| 3 | 175 | Hidden nouns, hidden contents within visible nouns |

No tier 4 — the gaussian roll system provides natural scaling. Higher
stats and skill ranks improve odds continuously without needing a
discrete bonus tier.

### Z-Score Effects

Standard z-score thresholds apply via `dice.RollStat`:
- `ZScore >= 2.0` — critical success (could trigger bonus flavor text)
- `ZScore <= -2.0` — fumble (could trigger a funny failure message)

### Cooldown

2-round cooldown on search retained (unchanged).

---

## 3. Room Data Model

### Hidden Nouns — New YAML Field

```yaml
hidden_nouns:
  compartment:
    description: "A narrow compartment tucked behind a loose panel.
      Inside you find dust and old parchment scraps."
    hidden_description: "You notice the edges of a loose panel in
      the wall, revealing a narrow compartment behind it."
```

- `description` — what `look <noun>` shows after discovery (same role
  as regular noun descriptions)
- `hidden_description` — appended to room description output for
  players who have discovered this noun

### Room Struct Addition

```go
HiddenNouns map[string]HiddenNoun `yaml:"hidden_nouns,omitempty" instance:"skip"`
```

Marked `instance:"skip"` so hidden nouns are template-driven, not
overridden by instance saves.

### HiddenNoun Struct (New)

```go
// HiddenNoun represents a discoverable noun in a room that is invisible
// until found via the search command (tier 3).
type HiddenNoun struct {
    Description       string `yaml:"description"`
    HiddenDescription string `yaml:"hidden_description"`
}
```

Lives in the rooms package alongside the existing Container struct.

### Hidden Containers — New Field on Container

```go
type Container struct {
    Lock         gamelock.Lock `yaml:"lock,omitempty"`
    Items        []items.Item  `yaml:"items,omitempty"`
    Gold         int           `yaml:"gold,omitempty"`
    DespawnRound uint64        `yaml:"despawnround,omitempty"`
    Recipes      map[int][]int `yaml:"recipes,omitempty,flow"`
    Hidden       bool          `yaml:"hidden,omitempty"` // NEW
}
```

Hidden containers behave like regular containers but are invisible
(no `look`, `open`, or listing) until discovered via tier-1 search roll.
Discovery is persisted the same way as hidden nouns.

**Instance save warning:** The `Containers` field is NOT marked
`instance:"skip"` — it is instance-persisted. When adding `hidden: true`
to a container in a room template, any existing instance save for that
room will lack the field (defaulting to `false`), overriding the template.
Per the project SOP: delete stale instance saves after editing templates.

**Container filtering:** `FindContainerByName()` and all container
interaction commands (`look`, `open`, `get`, `put`) must filter out
hidden containers that the player has not discovered. The simplest
approach: add a `FindVisibleContainerByName(name, discoveries)` wrapper
or pass the player's discoveries into the existing lookup. All container
command call sites must be updated.

### Authoring Convention

Hidden nouns that represent "hidden contents within a visible noun"
(e.g., a compartment behind a bookshelf) have no formal parent link.
The relationship is expressed through prose in `hidden_description`.
This keeps the data model simple. The convention must be documented in
context.md files and the content generation guide.

---

## 4. Character Discoveries

### New Field on Character

```go
Discoveries map[int][]string `yaml:"discoveries,omitempty"`
```

Key is room ID, value is a slice of discovered noun/container keys.

### Helper Methods

```go
// HasDiscovery returns true if the player has discovered the given
// noun/container in the specified room.
func (c *Character) HasDiscovery(roomId int, key string) bool

// AddDiscovery records that the player has discovered the given
// noun/container. No-op if already discovered.
func (c *Character) AddDiscovery(roomId int, key string)
```

### Persistence

The `Discoveries` map persists to character YAML automatically via the
existing save system. No additional persistence code needed.

### Flushing

To reset discoveries (e.g., after major content changes), flush the
`Discoveries` field on affected characters. No built-in decay mechanism
— manual flush only, as needed.

---

## 5. Room Description Integration

### Look Command Changes

When a player uses `look` in a room:

1. Render the base room description (unchanged)
2. Check `user.Character.Discoveries[room.RoomId]`
3. For each discovered key that matches a `hidden_nouns` entry, append
   the `hidden_description` text as a new paragraph after the base
   description
4. For each discovered key that matches a hidden container, include
   that container in the normal container listing

### Look Noun Changes

When a player uses `look <noun>`:

1. Check regular `room.Nouns` first (unchanged)
2. If no match, check `room.HiddenNouns`:
   - If player has discovery → show `description`
   - If player does not have discovery → no match (as if noun doesn't
     exist)
3. If no match, check containers (unchanged, but now includes hidden
   containers the player has discovered)

### Undiscovered State

- Hidden nouns: completely invisible. Not in room description, not
  interactable via `look <noun>`, no visual hint.
- Hidden containers: not listed, not interactable via `look`/`open`.

### Ordering

When a room has multiple discovered hidden nouns, their
`hidden_description` paragraphs are appended in **sorted key order**
(alphabetical by noun key). Go map iteration is non-deterministic, so
the implementation must sort keys before rendering.

### Edge Cases

- **No hidden nouns/containers in room:** search produces no tier-3
  results. No special message — the player just sees results (or lack
  thereof) from lower tiers.
- **All hidden things already discovered:** search skips discovered items
  and does NOT trigger skill progression. This prevents AFK botting by
  repeatedly searching a fully-explored room. The player sees "You snoop
  around for a bit..." but gains nothing.
- **Hidden container with a lock:** discovery reveals the container, but
  the lock still applies normally. Player must pick or use a key after
  discovering it.

---

## 6. Search Command Refactor

### Replace Stat Thresholds with Rolls

Remove the old `skillLevel` calculation based on hard Perception
thresholds. Replace with:

```go
searchScore := perceptionAdj + int(combat.SkillMultiplier(searchRank) * 25.0)
```

Each discovery type rolls `dice.RollStat(searchScore)` against its tier
target independently.

### Fix Existing Bugs

0. **Typo** — comment at line 18 says "Searcg Skill", fix to "Search Skill".
1. **Stashed items gate** — currently at `skillLevel > 2` (tier 3+).
   Should be tier 2 (target 135). Move to the correct roll check.
2. **Hidden mob detection** — line 135 appends to `hiddenPlayers` slice
   instead of `hiddenMobs`. Fix the variable name.
3. **Hidden mob lookup** — line 127 calls `users.GetByUserId(mId)` for
   mob instance IDs. Should use `mobs.GetInstance(mId)`, which returns
   `*mobs.Mob` (not `*users.UserRecord`). The subsequent
   `.Character.HasBuffFlag()` call works the same way since both types
   have a `Character` field.

### Tier 3 Implementation

Fill the empty placeholder (lines 171-174) with hidden noun discovery:

```go
// Tier 3: Hidden nouns and hidden contents
for nounKey, hiddenNoun := range room.HiddenNouns {
    if user.Character.HasDiscovery(room.RoomId, nounKey) {
        continue
    }
    roll := dice.RollStat(float64(searchScore))
    if roll.Value >= 175 {
        user.Character.AddDiscovery(room.RoomId, nounKey)
        user.SendText(fmt.Sprintf(
            "You discover something: <ansi fg=\"noun\">%s</ansi>",
            nounKey))
        user.SendText(hiddenNoun.HiddenDescription)
    }
}
```

### Skill Progression

Only trigger progression if the room contained at least one undiscovered
thing to roll against (secret exit, hidden container, stashed item,
hidden player/mob, or hidden noun). If everything in the room has already
been discovered or there is nothing to find, skip the progression call.
This prevents AFK botting by repeatedly searching a fully-explored room.

```go
if rolledAgainstSomething {
    user.Character.CheckSkillProgression("search", user.UserId, 1.0)
}
```

---

## 7. Track Command Rework

The tracking command currently uses hard skill-level tiers (1-4) from
`GetSkillLevel(skills.Tracking)`. Rework to use the unified gaussian
roll system with the same `searchScore` formula.

### Roll-Based Tier Mapping

One `dice.RollStat(float64(searchScore))` roll per `track` use. The roll
value determines how much detail the player receives:

| Roll Value | Information Revealed |
|-----------|---------------------|
| >= 125 | Most recent visitor only (name + trail strength) |
| >= 135 | All recent visitors (names + trail strengths) |
| >= 175 | All visitors + exit directions + targeted tracking |

Active tracking (old tier 4) is gated at the 175 threshold as well.
The `track <target>` syntax for targeted/active tracking requires
beating 175 on the roll.

### Behavioral Changes

- Remove the `GetSkillLevel(skills.Tracking)` check entirely
- Remove the `skillLevel == 0` gate — any character can attempt tracking
  (the roll determines success)
- Replace `skills.Tracking` references with `skills.Search`
- Fire `CheckSkillProgression("search", ...)` on every use
- Cooldown unchanged (1 round)
- Active tracking buff (buff 26) still applied on successful tier-3
  targeted track

### Trail Strength

`trailStrengthToString()` and `findExited()` helpers are unchanged.
The roll determines *whether* you see information, not *what quality*
the trail data is — trail strength is still based on how recently the
visitor passed through.

---

## 8. Forage Command Rework

The forage command currently uses a flat percentage roll:
`20 + (skillRank * SkillWeight * 5) + ceil(Perception/10)`, capped at 90%.
Rework to use the unified gaussian roll system.

### Roll Mechanics

```
searchScore = Perception + SkillMultiplier(searchRank) * 25.0
roll = dice.RollStat(float64(searchScore))
```

Success if `roll.Value >= biome difficulty target`. Each biome has a
base difficulty:

| Biome | Difficulty | Rationale |
|-------|-----------|-----------|
| farmland | 110 | Cultivated, easy pickings |
| forest | 120 | Abundant but need to know where to look |
| land | 125 | Generic terrain, moderate |
| swamp | 130 | Harder to find usable materials |
| shore | 135 | Limited yields |
| mountains | 140 | Harsh terrain, sparse resources |
| cliffs | 145 | Dangerous and sparse |
| cave | 135 | Dark but concentrated deposits |

These values are initial estimates — tune after playtesting. Could be
stored as a field on the Biome struct or in a config map.

### Rare Yields

On a critical success (z-score >= 2.0), the player could find a rarer
item from a secondary yield table. This is optional flavor — implement
only if biome yield tables are expanded to have common/rare tiers.

### Behavioral Changes

- Remove `GetSkillLevel(skills.Foraging)` and the old percentage formula
- Replace `skills.Foraging` references with `skills.Search`
- Fire `CheckSkillProgression("search", ...)` on every use
- The `forageYields` biome map is unchanged — still maps biome → item IDs
- Cooldown unchanged (6 rounds)
- `OnSkillUse("foraging", ...)` call at line 75 replaced with
  `CheckSkillProgression("search", ...)`

---

## 9. Skill Migration Details

### Skills Package Changes

- Add: `Search SkillTag = "search"` constant
- Remove: `Tracking` and `Foraging` constants
- Update `AllSkills()`, `StarterSkills()`, starting skill sets
- Update `skillStatMap` / `SkillPrimaryStats`: `"search": "perception"`,
  remove `"tracking"` and `"foraging"` entries
- Update `skillProgressionMultiplier`: `Search: 2.0`
- Update `Professions` map: replace `Tracking`/`Foraging` with `Search`
  in ranger/survivalist profession entries
- Update `totalSkills` constant (currently 17, will be 16 after removing
  two skills and adding one)
- Update both the `Professions` map AND the explicit `init()` registration
  list — skills are registered from both sources

### Character Load Migration

In `Character.Validate()`, **before** `ensureAllSkills()` is called:

```go
// Migrate tracking/foraging → search (must run before ensureAllSkills)
if _, hasTracking := c.Skills["tracking"]; hasTracking {
    trackRank := c.Skills["tracking"]
    forageRank := c.Skills["foraging"]
    c.Skills["search"] = max(trackRank, forageRank)
    if c.Skills["search"] < 1 {
        c.Skills["search"] = 1
    }
    c.SkillUseCount["search"] = c.SkillUseCount["tracking"] +
        c.SkillUseCount["foraging"]
    delete(c.Skills, "tracking")
    delete(c.Skills, "foraging")
    delete(c.SkillUseCount, "tracking")
    delete(c.SkillUseCount, "foraging")
} else if _, hasForaging := c.Skills["foraging"]; hasForaging {
    // Edge case: has foraging but not tracking
    c.Skills["search"] = max(c.Skills["foraging"], 1)
    c.SkillUseCount["search"] = c.SkillUseCount["foraging"]
    delete(c.Skills, "foraging")
    delete(c.SkillUseCount, "foraging")
}
```

### Track and Forage Commands

Update both commands to call `CheckSkillProgression("search", ...)`
instead of their old skill tags. Gameplay logic unchanged.

---

## 10. Content & Documentation Updates

### Schema Documentation

- `docs/schemas/room.md` — add `hidden_nouns` field and container
  `hidden` field documentation
- Include YAML examples for both hidden nouns and hidden containers

### Context Files

- `internal/rooms/context.md` — document the `HiddenNoun` struct,
  `hidden_nouns` field, hidden container field, and the authoring
  convention (hidden contents reference parent nouns via prose, no
  formal parent link)
- `internal/usercommands/context.md` — document search command changes,
  roll mechanics, tier targets
- `internal/skills/context.md` — document search skill consolidation,
  removal of tracking/foraging

### Content Generation Guide

- `docs/CONTENT_GENERATION_GUIDE.md` — add section on authoring hidden
  nouns: when to use them, how to write `hidden_description` text, the
  convention for hidden contents within visible nouns

### Help Files

- Update `search` help file — new skill description, roll-based system
- Update `track` help file — note that it now uses the search skill
- Update `forage` help file — note that it now uses the search skill
- Update skill list help file — replace tracking/foraging with search
- Update or redirect any help files that reference tracking/foraging
- Update helpfile registry with new/changed entries

### Tutorial Area Audit

- Review wilderness guide quest for references to tracking skill
- Update NPC dialogue that mentions tracking or foraging skills
- Ensure the tutorial teaches the search skill appropriately

---

## 11. Testing Strategy

### Unit Tests

- `HasDiscovery` / `AddDiscovery` — basic CRUD, idempotency
- Skill migration — verify max-of-two-ranks logic, use count merging,
  old skill pruning, idempotency
- `HiddenNoun` YAML parsing — verify struct loads correctly from YAML
- Container `Hidden` field — verify parsing and default false
- Search roll logic — verify per-discovery rolls against tier targets
- Track roll logic — verify tier mapping (125/135/175) controls info depth
- Forage roll logic — verify biome difficulty targets and success/failure

### Integration Verification

- Build check: `go build ./...`
- Full test suite: `go test ./internal/...`
- Manual testing:
  - Search in a room with hidden nouns at various Perception/skill levels
  - Verify discovery persists across logout/login
  - Verify `look` shows hidden_description after discovery
  - Verify `look <noun>` works after discovery, fails before
  - Verify hidden containers appear after discovery
  - Verify track command uses gaussian rolls, shows correct tier of info
  - Verify forage command uses gaussian rolls against biome difficulty
  - Verify all three commands fire search skill progression
  - Verify skill migration on an existing character save file
