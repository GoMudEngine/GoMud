# Achievements / accolades Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** YAML-authored achievements, unlocked by a poll over each player's state, shown via an `achievements` command (+ web page), with an achievement-points leaderboard and a ~23-item starter set — including "The Pinnacle" for acquiring legendary-tier gear.

**Architecture:** `internal/achievements` owns the `Definition`/`Trigger` types, the boot-validated loader, and the pure `Evaluate(trigger, *Character) bool`. A `NewRound` **hook** polls online players and unlocks; an `achievements` **usercommand** displays them; the **leaderboards** module gains a points board; a thin **module** adds only the web page (last task). Per-character state lives in `Character.Achievements map[string]uint64`.

**Refinement vs spec:** poll = hook, command = usercommand (no plugin needed for those); only the web page uses a minimal `modules/achievements/` plugin.

**Tech Stack:** Go, GoMud loader/event/command layers, testify, YAML.

**Spec:** `docs/superpowers/specs/2026-07-15-achievements-design.md`

---

## File Structure

- `internal/characters/character.go` — `Achievements` field + `HasAchievement`/`GrantAchievement`.
- `internal/achievements/achievements.go` — `Definition`, `Trigger`, registry, points helpers.
- `internal/achievements/evaluate.go` — `Evaluate(trigger, *characters.Character) bool`.
- `internal/achievements/loader.go` — `LoadDataFiles()` (world data, validate + panic).
- `internal/achievements/*_test.go` — evaluator + loader + points tests.
- `internal/hooks/Achievements_Poll.go` — `NewRound` poll → grant + announce.
- `internal/usercommands/achievements.go` (+ help template + registration).
- `modules/leaderboards/leaderboards.go` — `LB_Achievements` points board.
- `modules/achievements/achievements.go` (+ files/) — web page only (last).
- `_datafiles/world/dogmud/achievements/*.yaml` — starter set (~23).
- `main.go` — `achievements.LoadDataFiles()` in the data-load sequence.
- `PATCH_NOTES.md`, `docs/PATH_TO_1.0.md`.

---

## Task 1: `Character.Achievements` field + helpers

**Files:**
- Modify: `internal/characters/character.go`
- Test: `internal/characters/achievements_test.go` (create)

- [ ] **Step 1: Write the failing test**

Create `internal/characters/achievements_test.go`:

```go
package characters

import "testing"

func TestCharacterAchievements(t *testing.T) {
	c := &Character{}
	if c.HasAchievement("first-blood") {
		t.Error("fresh character has no achievements")
	}
	c.GrantAchievement("first-blood", 42)
	if !c.HasAchievement("first-blood") {
		t.Error("granted achievement should be present")
	}
	if c.Achievements["first-blood"] != 42 {
		t.Errorf("unlock round = %d, want 42", c.Achievements["first-blood"])
	}
	// Idempotent: re-grant keeps the original round.
	c.GrantAchievement("first-blood", 99)
	if c.Achievements["first-blood"] != 42 {
		t.Errorf("re-grant should not overwrite the original unlock round")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/characters/ -run TestCharacterAchievements -v`
Expected: FAIL — `Achievements` / `HasAchievement` / `GrantAchievement` undefined.

- [ ] **Step 3: Add the field + helpers**

In `internal/characters/character.go`, add to the `Character` struct near `VisitedRooms`:

```go
	Achievements map[string]uint64 `yaml:"achievements,omitempty"` // achievement id -> round unlocked
```

Add the helpers (near the MiscData helpers):

```go
// HasAchievement reports whether this character has unlocked the given achievement.
func (c *Character) HasAchievement(id string) bool {
	_, ok := c.Achievements[id]
	return ok
}

// GrantAchievement records an achievement unlock at the given round. Idempotent:
// a re-grant keeps the original unlock round.
func (c *Character) GrantAchievement(id string, round uint64) {
	if c.Achievements == nil {
		c.Achievements = make(map[string]uint64)
	}
	if _, ok := c.Achievements[id]; !ok {
		c.Achievements[id] = round
	}
}
```

- [ ] **Step 4: Run test + commit**

Run: `go test ./internal/characters/ -run TestCharacterAchievements -v` → PASS
```bash
git add internal/characters/character.go internal/characters/achievements_test.go
git commit -m "feat(achievements): Character.Achievements storage + helpers"
```

---

## Task 2: `internal/achievements` types + `Evaluate` (the core)

**Files:**
- Create: `internal/achievements/achievements.go` (types + registry stub)
- Create: `internal/achievements/evaluate.go`
- Test: `internal/achievements/evaluate_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/achievements/evaluate_test.go`:

```go
package achievements

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
)

func charWith(fn func(*characters.Character)) *characters.Character {
	c := &characters.Character{}
	fn(c)
	return c
}

func TestEvaluate_MobKills(t *testing.T) {
	c := charWith(func(c *characters.Character) { c.KD.TotalKills = 100 })
	if !Evaluate(Trigger{Type: "mob_kills", Threshold: 100}, c, 0) {
		t.Error("100 kills should meet threshold 100")
	}
	if Evaluate(Trigger{Type: "mob_kills", Threshold: 101}, c, 0) {
		t.Error("100 kills should not meet threshold 101")
	}
}

func TestEvaluate_GoldTotal(t *testing.T) {
	c := charWith(func(c *characters.Character) { c.Gold = 600; c.Bank = 500 })
	if !Evaluate(Trigger{Type: "gold_total", Threshold: 1000}, c, 0) {
		t.Error("600 purse + 500 bank = 1100 should meet 1000")
	}
}

func TestEvaluate_StatReachedAny(t *testing.T) {
	c := &characters.Character{}
	c.Stats.Strength.ValueAdj = 155
	if !Evaluate(Trigger{Type: "stat_reached", Stat: "any", Threshold: 150}, c, 0) {
		t.Error("a 155 stat should satisfy any>=150")
	}
	if Evaluate(Trigger{Type: "stat_reached", Stat: "dexterity", Threshold: 150}, c, 0) {
		t.Error("dexterity is 0; should not satisfy 150")
	}
	if !Evaluate(Trigger{Type: "stat_reached", Stat: "strength", Threshold: 150}, c, 0) {
		t.Error("strength 155 should satisfy strength>=150")
	}
}

func TestEvaluate_MutationCount(t *testing.T) {
	c := charWith(func(c *characters.Character) { c.Mutations = map[string]int{"a": 1, "b": 2} })
	if !Evaluate(Trigger{Type: "mutation_count", Threshold: 2}, c, 0) {
		t.Error("2 mutations should meet threshold 2")
	}
}

func TestEvaluate_AchievementPoints(t *testing.T) {
	c := &characters.Character{}
	// earnedPoints passed in (last arg) — the meta trigger reads it, not char state.
	if !Evaluate(Trigger{Type: "achievement_points", Threshold: 50}, c, 60) {
		t.Error("60 earned points should meet 50")
	}
	if Evaluate(Trigger{Type: "achievement_points", Threshold: 50}, c, 40) {
		t.Error("40 earned points should not meet 50")
	}
}

func TestEvaluate_UnknownType(t *testing.T) {
	if Evaluate(Trigger{Type: "nonsense", Threshold: 1}, &characters.Character{}, 0) {
		t.Error("unknown trigger type should never satisfy")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/achievements/ -run TestEvaluate -v`
Expected: FAIL — package/`Trigger`/`Evaluate` undefined.

- [ ] **Step 3: Add the types**

Create `internal/achievements/achievements.go`:

```go
package achievements

// Category is the grouping an achievement belongs to.
const (
	CategoryCombat      = "combat"
	CategoryExploration = "exploration"
	CategoryWealth      = "wealth"
	CategoryProgression = "progression"
	CategoryQuests      = "quests"
)

// validCategories is the allowed set (loader validation).
var validCategories = map[string]bool{
	CategoryCombat: true, CategoryExploration: true, CategoryWealth: true,
	CategoryProgression: true, CategoryQuests: true,
}

// Trigger is the fixed-vocabulary unlock condition for an achievement.
type Trigger struct {
	Type      string `yaml:"type"`
	Threshold int    `yaml:"threshold,omitempty"` // count/value types
	Stat      string `yaml:"stat,omitempty"`      // stat_reached (a primary stat name or "any")
	Skill     string `yaml:"skill,omitempty"`     // skill_reached (a skill name or "any")
	Token     string `yaml:"token,omitempty"`     // quest_completed (a quest token)
}

// Definition is one authored achievement.
type Definition struct {
	Id          string  `yaml:"id"`
	Name        string  `yaml:"name"`
	Description string  `yaml:"description"`
	Category    string  `yaml:"category"`
	Points      int     `yaml:"points"`
	Trigger     Trigger `yaml:"trigger"`
}

// registry holds the loaded definitions, keyed by id, plus a stable ordering.
var (
	registry      = map[string]Definition{}
	registryOrder []string
)

// All returns the loaded definitions in load order.
func All() []Definition {
	out := make([]Definition, 0, len(registryOrder))
	for _, id := range registryOrder {
		out = append(out, registry[id])
	}
	return out
}

// Get returns a definition by id.
func Get(id string) (Definition, bool) { d, ok := registry[id]; return d, ok }

// PointsFor sums the points of the given unlocked achievement ids.
func PointsFor(ids map[string]uint64) int {
	total := 0
	for id := range ids {
		if d, ok := registry[id]; ok {
			total += d.Points
		}
	}
	return total
}
```

- [ ] **Step 4: Add the evaluator**

Create `internal/achievements/evaluate.go`:

```go
package achievements

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

// pinnacleEquipTypes are the wearable/wieldable item types that count for the
// item_rarity ("acquired a pinnacle item") trigger — excludes components,
// materials, potions, and quest junk so a raw high-rarity reagent doesn't count.
var pinnacleEquipTypes = map[items.ItemType]bool{
	items.Weapon: true, items.Offhand: true, items.Head: true, items.Neck: true,
	items.Shoulders: true, items.Body: true, items.Back: true, items.Belt: true,
	items.Wrist: true, items.Gloves: true, items.Ring: true, items.Legs: true,
	items.Feet: true,
}

// Evaluate reports whether trigger t is satisfied by character c. earnedPoints is
// the character's current total achievement points (only used by the meta
// achievement_points trigger). Pure — no mutation, no side effects.
func Evaluate(t Trigger, c *characters.Character, earnedPoints int) bool {
	switch t.Type {
	case "mob_kills":
		return c.KD.TotalKills >= t.Threshold
	case "pvp_kills":
		return c.KD.TotalPvpKills >= t.Threshold
	case "deaths":
		return c.KD.TotalDeaths >= t.Threshold
	case "gold_total":
		return c.Gold+c.Bank >= t.Threshold
	case "mutation_count":
		return len(c.Mutations) >= t.Threshold
	case "rooms_explored":
		return roomsExplored(c) >= t.Threshold
	case "quest_completed":
		return c.HasQuest(t.Token)
	case "quests_completed":
		return completedQuestCount(c) >= t.Threshold
	case "stat_reached":
		return statValue(c, t.Stat) >= t.Threshold
	case "skill_reached":
		return skillValue(c, t.Skill) >= t.Threshold
	case "item_rarity":
		return ownsPinnacleItem(c, t.Threshold)
	case "achievement_points":
		return earnedPoints >= t.Threshold
	}
	return false
}

func roomsExplored(c *characters.Character) int {
	total := 0
	for _, ids := range c.VisitedRooms {
		total += len(ids)
	}
	return total
}

// completedQuestCount counts quests whose progress token ends in "-end".
func completedQuestCount(c *characters.Character) int {
	n := 0
	for _, token := range c.QuestProgress {
		if len(token) >= 4 && token[len(token)-4:] == "-end" {
			n++
		}
	}
	return n
}

func statValue(c *characters.Character, name string) int {
	switch name {
	case "strength":
		return c.Stats.Strength.ValueAdj
	case "dexterity":
		return c.Stats.Dexterity.ValueAdj
	case "perception":
		return c.Stats.Perception.ValueAdj
	case "vitality":
		return c.Stats.Vitality.ValueAdj
	case "willpower":
		return c.Stats.Willpower.ValueAdj
	case "charisma":
		return c.Stats.Charisma.ValueAdj
	case "any":
		best := 0
		for _, v := range []int{
			c.Stats.Strength.ValueAdj, c.Stats.Dexterity.ValueAdj, c.Stats.Perception.ValueAdj,
			c.Stats.Vitality.ValueAdj, c.Stats.Willpower.ValueAdj, c.Stats.Charisma.ValueAdj,
		} {
			if v > best {
				best = v
			}
		}
		return best
	}
	return 0
}

func skillValue(c *characters.Character, name string) int {
	if name == "any" {
		best := 0
		for _, tag := range skills.GetAllSkillNames() {
			if lvl := c.GetSkillLevel(tag); lvl > best {
				best = lvl
			}
		}
		return best
	}
	return c.GetSkillLevel(skills.SkillTag(name))
}

// ownsPinnacleItem reports whether the character holds any equipment-type item at
// or above the given rarity tier (backpack + equipped + bank storage).
func ownsPinnacleItem(c *characters.Character, minRarity int) bool {
	qualifies := func(it items.Item) bool {
		spec := it.GetSpec()
		return pinnacleEquipTypes[spec.Type] && spec.RarityTier >= minRarity
	}
	for _, it := range c.Items {
		if qualifies(it) {
			return true
		}
	}
	for _, it := range c.Equipment.GetAllItems() {
		if qualifies(it) {
			return true
		}
	}
	// Note: bank storage lives on the UserRecord, not the Character; the poll
	// passes an already-merged holdings view OR we skip storage here. See Task 5.
	return false
}
```

Note: `ownsPinnacleItem` scans `c.Items` + equipped. Bank storage (`ItemStorage`) is on the
`UserRecord`, not the `Character`. Keep the Character-only scan here (backpack + equipped);
the spec's "storage" scan is a nice-to-have that would require passing the UserRecord — do
NOT add it in this task to keep `Evaluate` Character-pure. (If storage coverage is wanted,
handle it in the poll by also checking `user.ItemStorage` — deferred.)

Confirm exact symbols before implementing (codegraph): `items.ItemType` constants
(`items.Weapon`, `items.Head`, …), `items.ItemSpec.RarityTier`, `items.ItemSpec.Type`,
`characters.Character.Equipment.GetAllItems()`, `characters.Character.KD.TotalKills`,
`skills.GetAllSkillNames()`, `skills.SkillTag`. Adjust field/const names to match.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/achievements/ -run TestEvaluate -v`
Expected: PASS (all).

- [ ] **Step 6: Commit**

```bash
git add internal/achievements/achievements.go internal/achievements/evaluate.go internal/achievements/evaluate_test.go
git commit -m "feat(achievements): Definition/Trigger types + pure Evaluate core"
```

---

## Task 3: Loader (`LoadDataFiles`) with boot validation

**Files:**
- Create: `internal/achievements/loader.go`
- Test: `internal/achievements/loader_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/achievements/loader_test.go`:

```go
package achievements

import "testing"

func TestValidateDefinition(t *testing.T) {
	good := Definition{Id: "first-blood", Name: "First Blood", Description: "d", Category: "combat", Points: 5, Trigger: Trigger{Type: "mob_kills", Threshold: 1}}
	if err := validateDefinition(good, "first-blood"); err != nil {
		t.Errorf("good def should validate: %v", err)
	}

	cases := []struct {
		name string
		d    Definition
		file string
	}{
		{"unknown type", Definition{Id: "x", Name: "n", Category: "combat", Trigger: Trigger{Type: "bogus"}}, "x"},
		{"bad category", Definition{Id: "x", Name: "n", Category: "nope", Trigger: Trigger{Type: "mob_kills", Threshold: 1}}, "x"},
		{"stat missing param", Definition{Id: "x", Name: "n", Category: "progression", Trigger: Trigger{Type: "stat_reached", Threshold: 1}}, "x"},
		{"bad stat name", Definition{Id: "x", Name: "n", Category: "progression", Trigger: Trigger{Type: "stat_reached", Stat: "wisdom", Threshold: 1}}, "x"},
		{"quest missing token", Definition{Id: "x", Name: "n", Category: "quests", Trigger: Trigger{Type: "quest_completed"}}, "x"},
		{"filename mismatch", Definition{Id: "x", Name: "n", Category: "combat", Trigger: Trigger{Type: "mob_kills", Threshold: 1}}, "y"},
		{"negative points", Definition{Id: "x", Name: "n", Category: "combat", Points: -1, Trigger: Trigger{Type: "mob_kills", Threshold: 1}}, "x"},
	}
	for _, tc := range cases {
		if err := validateDefinition(tc.d, tc.file); err == nil {
			t.Errorf("%s: expected validation error, got nil", tc.name)
		}
	}

	// "any" is a valid stat/skill.
	anyStat := Definition{Id: "x", Name: "n", Category: "progression", Trigger: Trigger{Type: "stat_reached", Stat: "any", Threshold: 130}}
	if err := validateDefinition(anyStat, "x"); err != nil {
		t.Errorf("stat 'any' should validate: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/achievements/ -run TestValidateDefinition -v`
Expected: FAIL — `validateDefinition` undefined.

- [ ] **Step 3: Implement loader + validation**

Create `internal/achievements/loader.go`. Model the file-walk + panic-on-error on
`internal/quests/quests.go`'s `LoadDataFiles` (read it first). Core:

```go
package achievements

import "fmt"

var validStats = map[string]bool{
	"strength": true, "dexterity": true, "perception": true,
	"vitality": true, "willpower": true, "charisma": true, "any": true,
}

// paramTypes need a positive threshold.
var thresholdTypes = map[string]bool{
	"mob_kills": true, "pvp_kills": true, "deaths": true, "gold_total": true,
	"rooms_explored": true, "quests_completed": true, "mutation_count": true,
	"item_rarity": true, "achievement_points": true, "stat_reached": true, "skill_reached": true,
}

func validateDefinition(d Definition, fileBase string) error {
	if d.Id == "" {
		return fmt.Errorf("achievement in %q has no id", fileBase)
	}
	if d.Id != fileBase {
		return fmt.Errorf("achievement %q: filename base %q must equal id", d.Id, fileBase)
	}
	if d.Name == "" {
		return fmt.Errorf("achievement %q: missing name", d.Id)
	}
	if !validCategories[d.Category] {
		return fmt.Errorf("achievement %q: invalid category %q", d.Id, d.Category)
	}
	if d.Points < 0 {
		return fmt.Errorf("achievement %q: points must be >= 0", d.Id)
	}
	if !thresholdTypes[d.Trigger.Type] {
		return fmt.Errorf("achievement %q: unknown trigger type %q", d.Id, d.Trigger.Type)
	}
	switch d.Trigger.Type {
	case "stat_reached":
		if !validStats[d.Trigger.Stat] {
			return fmt.Errorf("achievement %q: stat_reached needs a valid stat (or 'any'), got %q", d.Id, d.Trigger.Stat)
		}
	case "skill_reached":
		if d.Trigger.Skill == "" {
			return fmt.Errorf("achievement %q: skill_reached needs a skill (or 'any')", d.Id)
		}
	case "quest_completed":
		if d.Trigger.Token == "" {
			return fmt.Errorf("achievement %q: quest_completed needs a token", d.Id)
		}
	default:
		if d.Trigger.Threshold <= 0 {
			return fmt.Errorf("achievement %q: %s needs threshold > 0", d.Id, d.Trigger.Type)
		}
	}
	return nil
}
```

Remove the `stats` import if unused (the literal `validStats` set above suffices — drop the
import line). Then add `LoadDataFiles()` that walks `_datafiles/world/dogmud/achievements/`
(use the same path/util helpers as `quests.LoadDataFiles`), unmarshals each `.yaml` into a
`Definition`, calls `validateDefinition(d, fileBaseWithoutExt)`, **panics** on any error
(matches the loader SOP), and populates `registry`/`registryOrder`. Log a
`mudlog.Info("achievements.LoadDataFiles()", "loadedCount", n, ...)` line like the others.

- [ ] **Step 4: Run test + commit**

Run: `go test ./internal/achievements/ -run TestValidateDefinition -v` → PASS
Run: `go build ./internal/achievements/` → success
```bash
git add internal/achievements/loader.go internal/achievements/loader_test.go
git commit -m "feat(achievements): boot-validated YAML loader"
```

---

## Task 4: Boot wiring

**Files:** Modify `main.go`

- [ ] **Step 1: Wire the loader**

In `main.go`, next to the existing `quests.LoadDataFiles()` call, add:

```go
	achievements.LoadDataFiles()
```

Add the import `"github.com/GoMudEngine/GoMud/internal/achievements"`.

- [ ] **Step 2: Build + commit**

Run: `go build ./...` → success
```bash
git add main.go
git commit -m "feat(achievements): load definitions at boot"
```

---

## Task 5: Poll hook — grant + private announce

**Files:**
- Create: `internal/hooks/Achievements_Poll.go`
- Modify: `internal/hooks/hooks.go` (register the listener — mirror how other NewRound hooks register)
- Test: `internal/hooks/Achievements_Poll_test.go`

- [ ] **Step 1: Write the failing test (pure grant helper)**

Extract the grant decision into a pure helper so it's testable without the full event/user
machinery. Create `internal/hooks/Achievements_Poll_test.go`:

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/achievements"
	"github.com/GoMudEngine/GoMud/internal/characters"
)

func TestNewlyEarnedAchievements(t *testing.T) {
	defs := []achievements.Definition{
		{Id: "kills-1", Category: "combat", Points: 5, Trigger: achievements.Trigger{Type: "mob_kills", Threshold: 1}},
		{Id: "kills-100", Category: "combat", Points: 10, Trigger: achievements.Trigger{Type: "mob_kills", Threshold: 100}},
	}
	c := &characters.Character{}
	c.KD.TotalKills = 5
	c.GrantAchievement("kills-1", 1) // already earned

	earned := newlyEarnedAchievements(defs, c, 10)
	if len(earned) != 0 {
		t.Fatalf("kills-1 already earned, kills-100 not met at 5 kills; want 0 new, got %d", len(earned))
	}

	c.KD.TotalKills = 100
	earned = newlyEarnedAchievements(defs, c, 20)
	if len(earned) != 1 || earned[0].Id != "kills-100" {
		t.Fatalf("want [kills-100], got %+v", earned)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/hooks/ -run TestNewlyEarnedAchievements -v`
Expected: FAIL — `newlyEarnedAchievements` undefined.

- [ ] **Step 3: Implement the hook**

Create `internal/hooks/Achievements_Poll.go`:

```go
package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/achievements"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// newlyEarnedAchievements returns the definitions the character now meets but
// hasn't unlocked yet. earnedPoints is the character's current total (for the
// achievement_points meta trigger). Pure.
func newlyEarnedAchievements(defs []achievements.Definition, c *characters.Character, earnedPoints int) []achievements.Definition {
	var earned []achievements.Definition
	for _, d := range defs {
		if c.HasAchievement(d.Id) {
			continue
		}
		if achievements.Evaluate(d.Trigger, c, earnedPoints) {
			earned = append(earned, d)
		}
	}
	return earned
}

// CheckAchievements polls online players and grants newly-earned achievements.
// Gated to a modest interval (AchievementPollRounds) to bound cost.
func CheckAchievements(e events.Event) events.ListenerReturn {
	interval := uint64(configs.GetBalanceConfig().AchievementPollRounds)
	if interval <= 0 {
		interval = 10
	}
	round := util.GetRoundCount()
	if round%interval != 0 {
		return events.Continue
	}

	defs := achievements.All()
	if len(defs) == 0 {
		return events.Continue
	}

	for _, u := range users.GetAllActiveUsers() {
		if u.Permission == users.PermissionAdmin || u.Character.IsAiControlled() { // match leaderboard exclusion
			continue
		}
		earnedPoints := achievements.PointsFor(u.Character.Achievements)
		earned := newlyEarnedAchievements(defs, u.Character, earnedPoints)
		if len(earned) == 0 {
			continue
		}
		for _, d := range earned {
			u.Character.GrantAchievement(d.Id, round)
			u.SendText(messaging.CategorySystem,
				fmt.Sprintf(`<ansi fg="yellow-bold">*** Achievement unlocked: %s ***</ansi>`, d.Name))
			u.SendText(messaging.CategorySystem,
				fmt.Sprintf(`<ansi fg="white">%s  (+%d points)</ansi>`, d.Description, d.Points))
		}
		users.SaveUser(*u)
	}
	return events.Continue
}
```

Register it: in `internal/hooks/hooks.go` (or wherever `RegisterListeners` wires NewRound
handlers), add `events.RegisterListener(events.NewRound{}, CheckAchievements)`. Read that
file first to match the exact registration style. Confirm the admin/AI exclusion accessors
against `modules/leaderboards/leaderboards.go`'s `Update()` (use the SAME checks it uses —
adjust `u.Permission == ...` / `IsAiControlled()` to the real API).

Add config `AchievementPollRounds` (ConfigInt, default 10) to `config.balance.go` +
`config.balance.misc.go` validator (`if <= 0 { = 10 }`), same as `MailSendCooldownRounds`.

- [ ] **Step 4: Run test + build + commit**

Run: `go test ./internal/hooks/ -run TestNewlyEarnedAchievements -v` → PASS
Run: `go build ./...` → success
```bash
git add internal/hooks/Achievements_Poll.go internal/hooks/hooks.go internal/hooks/Achievements_Poll_test.go internal/configs/
git commit -m "feat(achievements): NewRound poll grants + privately announces unlocks"
```

---

## Task 6: `achievements` command

**Files:**
- Create: `internal/usercommands/achievements.go`
- Modify: `internal/usercommands/usercommands.go` (register)
- Create: `_datafiles/world/dogmud/templates/help/achievements.template` (+ copy to `default/`)

- [ ] **Step 1: Implement the command**

Create `internal/usercommands/achievements.go`: `Achievements(rest, user, room, flags)` that
lists the user's earned + locked achievements grouped by category, with a progress hint for
locked ones and a total (count + points via `achievements.PointsFor`). Use
`achievements.All()` for the full list and `user.Character.HasAchievement(id)`. A
`progressHint(def, character)` helper renders e.g. `mob_kills 42/100`. Keep it ANSI-styled,
80-col wrapped, no emoji.

- [ ] **Step 2: Register + help file**

In `usercommands.go`: `` `achievements`: {Achievements, true, true, false}, ``. Create the
help template (mirror `inbox.template` style) and copy to both `dogmud/` and `default/`
help dirs (the helpfile-completeness test requires the dogmud one).

- [ ] **Step 3: Build + full usercommands test + commit**

Run: `go build ./... && go test ./internal/usercommands/` → PASS (help completeness satisfied)
```bash
git add internal/usercommands/achievements.go internal/usercommands/usercommands.go _datafiles/world/dogmud/templates/help/achievements.template _datafiles/world/default/templates/help/achievements.template
git commit -m "feat(achievements): 'achievements' command"
```

---

## Task 7: Achievement-points leaderboard board

**Files:** Modify `modules/leaderboards/leaderboards.go`

- [ ] **Step 1: Add the board**

Read `modules/leaderboards/leaderboards.go` fully. Add an `LB_Achievements leaderboardData`
field (mirroring `LB_Power`), reset it in `loadLBs`, and in `Update()` compute each
character's achievement points via `achievements.PointsFor(char.Achievements)` and
`Consider(...)` it into the board, alongside the existing Power computation. Add it to
`getCurrentLeaderboards()`. Follow the exact Power pattern.

- [ ] **Step 2: Build + commit**

Run: `go build ./modules/leaderboards/` → success
```bash
git add modules/leaderboards/leaderboards.go
git commit -m "feat(leaderboards): achievement-points board"
```

---

## Task 8: Starter content (~23 achievements)

**Files:** Create `_datafiles/world/dogmud/achievements/<id>.yaml` (one per achievement)

- [ ] **Step 1: Author the set**

Create one YAML per the spec §9 ladder (filename = id). Example
(`_datafiles/world/dogmud/achievements/first-blood.yaml`):

```yaml
id: first-blood
name: First Blood
description: Defeat your first foe.
category: combat
points: 5
trigger:
  type: mob_kills
  threshold: 1
```

Author all: first-blood, cutting-teeth (mob_kills 100), veteran (mob_kills 1000), duelist
(pvp_kills 10), hard-to-kill (deaths 10); wanderer/pathfinder/cartographer (rooms_explored
25/100/250); coin-purse/well-off/magnate (gold_total 1000/10000/100000); honed/formidable
(stat_reached any 130/160), skilled (skill_reached any 25), twice-touched/chimeric
(mutation_count 2/5), the-pinnacle (item_rarity 82, points ~40); errand-runner/adventurer/
hero-of-the-realm (quests_completed 1/5/15); decorated/legend (achievement_points 50/150).
Keep points a coherent ladder (small milestones 5–10, big ones 25–40, meta 50). Follow the
no-hard-numbers-in-prose rule in `description` (describe the feat, thresholds live in the
trigger + the command's progress hint).

- [ ] **Step 2: Commit**

```bash
git add _datafiles/world/dogmud/achievements/
git commit -m "content(achievements): ~23-item starter set across five categories"
```

---

## Task 9: Web page (minimal module) — LAST

**Files:** Create `modules/achievements/achievements.go` (+ `files/achievements.html`)

- [ ] **Step 1: Minimal plugin for the web page**

Mirror `modules/leaderboards/`'s web wiring: a plugin that registers
`plug.Web.WebPage("Achievements", "/achievements", "achievements.html", true, webAchievementData)`,
where `webAchievementData` returns all definitions grouped by category with the viewing
user's earned/locked state. Reuse `internal/achievements.All()` + the user's `Achievements`
map. Author `files/achievements.html` mirroring `leaderboards.html`.

- [ ] **Step 2: Build + commit**

Run: `go build ./...` → success
```bash
git add modules/achievements/
git commit -m "feat(achievements): web page"
```

---

## Task 10: Boot smoke test + docs

- [ ] **Step 1: Full build + touched-package tests**

Run: `go build ./... && go test ./internal/achievements/ ./internal/characters/ ./internal/hooks/ ./internal/usercommands/ ./modules/leaderboards/`
Expected: all `ok`.

- [ ] **Step 2: Boot smoke test (pre-push SOP)**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```

Run `go run .`; confirm `achievements.LoadDataFiles() loadedCount=~23` and `Server Ready`
with **no panic** (the loader validates every starter def), then stop.

- [ ] **Step 3: Patch note + roadmap**

Prepend a `PATCH_NOTES.md` entry (player-facing: new `achievements` command, accolades to
earn, points leaderboard). In `docs/PATH_TO_1.0.md` §3, mark **Achievements / accolades** ✅
with a dated note referencing this plan + spec; note guilds/clans remains as the other §3 item.

- [ ] **Step 4: Commit**

```bash
git add PATCH_NOTES.md docs/PATH_TO_1.0.md
git commit -m "docs(achievements): patch notes + roadmap"
```

---

## Notes for the implementer

- **Verify symbols with codegraph before Task 2/5/7** — this plan names several APIs
  (`items.ItemSpec.RarityTier`, `items.ItemType` constants, `Character.KD.TotalKills/TotalPvpKills/TotalDeaths`,
  `Character.Equipment.GetAllItems`, `skills.GetAllSkillNames`, the leaderboard admin/AI
  exclusion, `users.SaveUser`, `util.GetRoundCount`). Confirm exact names/receivers and
  adjust; do not assume.
- **Loader parity:** read `internal/quests/quests.go` `LoadDataFiles` for the exact file-walk
  + path + panic conventions and mirror them (the boot test depends on the panic behavior).
- **No emoji** in any player-facing text (client ASCII gap) — use ANSI color, not glyphs.
- **Admin/AI exclusion** must match the leaderboard's exactly so the two agree on who's ranked.
- **`Evaluate` stays Character-pure** (no UserRecord) — bank-storage scanning for `item_rarity`
  is deferred; backpack + equipped is the v1 scope.
```
