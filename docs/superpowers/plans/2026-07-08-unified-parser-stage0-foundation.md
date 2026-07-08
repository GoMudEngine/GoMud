# Unified Parser Seam — Stage 0 (Foundation) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the `internal/parser` package — a greedy multi-word target
resolver (`Resolve` primitive + `ResolveItem`/`ResolveActor` helpers + kind
adapters wrapping the existing `Find*` functions) — fully unit-tested, with **no
command migrated yet.**

**Architecture:** A command declares which target `Kind`s it wants; the package
tokenizes the input, tries the **longest multi-word span first**, and dispatches
each candidate span to a per-`Kind` adapter that wraps an existing resolver
(`room.FindNoun`, `room.FindOnFloor`, `room.FindContainerByName`,
`room.FindCorpseIndex`, `room.FindByName`, `character.FindItem`, …). The greedy
engine is dependency-injected so its algorithm is tested against fake adapters in
isolation; the real adapters are tested against seeded fixtures.

**Tech Stack:** Go, `testify` (assert/require), the existing
`Seed*ForTest`/`NewTestUser` helpers in `internal/{items,mobs,rooms,users}`.

**Spec:** `docs/superpowers/specs/2026-07-08-unified-parser-seam-design.md`

**Refinement vs spec:** the spec listed `KindBackpackItem` + `KindEquippedItem`
separately; `character.FindItem` already returns a combined backpack+equipped
pool with a source string, so this stage implements a single `KindInventoryItem`
adapter over it. All other kinds match the spec.

---

## File Structure

- `internal/parser/parser.go` — `Kind`, `Scope`, `Match`, the greedy-span engine
  (`resolveWith`), and the public `Resolve`.
- `internal/parser/adapters.go` — one `adapter` func per `Kind`, each wrapping an
  existing resolver, plus the `Kind → adapter` registry.
- `internal/parser/helpers.go` — the composed helpers `ResolveItem`,
  `ResolveActor`, and `splitOnConnective`.
- `internal/parser/parser_test.go` — engine tests (fake adapters), plus
  `TestMain` + `seedParserTest` scaffolding.
- `internal/parser/adapters_test.go` — per-adapter tests against seeded fixtures.
- `internal/parser/helpers_test.go` — composed-helper tests.

---

## Task 1: Core types + greedy-span engine (pure, fake adapters)

**Files:**
- Create: `internal/parser/parser.go`
- Test: `internal/parser/parser_test.go`

- [ ] **Step 1: Write the failing test**

`internal/parser/parser_test.go`:

```go
package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// exactAdapter returns a Match only when the candidate equals want.
func exactAdapter(want string, kind Kind) adapter {
	return func(_ Scope, candidate string) (Match, bool) {
		if candidate == want {
			return Match{Kind: kind, Name: candidate}, true
		}
		return Match{}, false
	}
}

func TestResolveWith_LongestSpanWins(t *testing.T) {
	// Both a 2-word and a 1-word adapter can match; the 2-word span must win.
	adapters := []adapter{
		exactAdapter("hare paths", KindNoun),
		exactAdapter("hare", KindNoun),
	}
	m, ok := resolveWith([]string{"hare", "paths"}, Scope{}, adapters)
	require.True(t, ok)
	assert.Equal(t, "hare paths", m.Name)
}

func TestResolveWith_FallsBackToShorterSpan(t *testing.T) {
	adapters := []adapter{exactAdapter("hare", KindNoun)}
	m, ok := resolveWith([]string{"hare", "zzz"}, Scope{}, adapters)
	require.True(t, ok)
	assert.Equal(t, "hare", m.Name) // "hare zzz" misses, "hare" hits
}

func TestResolveWith_KindPriorityBreaksTies(t *testing.T) {
	// Two adapters match the SAME span; the first in order wins.
	adapters := []adapter{
		exactAdapter("bank clerk", KindMob),
		exactAdapter("bank clerk", KindNoun),
	}
	m, ok := resolveWith([]string{"bank", "clerk"}, Scope{}, adapters)
	require.True(t, ok)
	assert.Equal(t, KindMob, m.Kind)
}

func TestResolveWith_NoMatch(t *testing.T) {
	adapters := []adapter{exactAdapter("nope", KindNoun)}
	_, ok := resolveWith([]string{"totally", "absent"}, Scope{}, adapters)
	assert.False(t, ok)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/parser/ -run TestResolveWith -v`
Expected: FAIL — `undefined: adapter`, `undefined: resolveWith`, `undefined: Match`, etc. (package does not compile yet).

- [ ] **Step 3: Write the minimal implementation**

`internal/parser/parser.go`:

```go
// Package parser is DOGMud's shared command target-resolution seam. Commands
// declare which Kinds of target they want; the package tokenizes the input,
// tries the longest multi-word span first, and dispatches each candidate span
// to a per-Kind adapter that wraps an existing resolver. See
// docs/superpowers/specs/2026-07-08-unified-parser-seam-design.md.
package parser

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

type Kind int

const (
	KindMob Kind = iota
	KindPlayer
	KindPet
	KindFloorItem
	KindInventoryItem // backpack + equipped, via character.FindItem
	KindComponentItem // component bag
	KindPotionItem    // bandolier
	KindRoomContainer
	KindCorpse
	KindNoun
	KindExit
)

// Scope is the search context. Room is required; User is required for
// inventory/component/potion kinds and ownership-sensitive lookups.
type Scope struct {
	User *users.UserRecord
	Room *rooms.Room
}

// Match is the typed result. Only the fields relevant to Kind are populated.
type Match struct {
	Kind          Kind
	Name          string     // canonical resolved name (for messaging)
	Item          items.Item // item-ish kinds
	Source        string     // free-form source note ("in your backpack", "wielded")
	MobInstanceId int        // KindMob
	UserId        int        // KindPlayer / KindPet
	ContainerName string     // KindRoomContainer
	CorpseIdx     int        // KindCorpse
	Leftover      string     // unconsumed tokens (used by later stages, e.g. recipes)
}

// adapter resolves a single candidate span to a Match. ok=false means no match.
type adapter func(s Scope, candidate string) (Match, bool)

// resolveWith runs the greedy longest-span algorithm: it tries the full token
// span first, then drops trailing tokens one at a time. At each span length it
// tries the adapters in order; the first hit wins. This makes a 2-word match
// ("bank clerk") beat a 1-word match ("bank"), and adapter order breaks ties
// within one span length.
func resolveWith(tokens []string, s Scope, adapters []adapter) (Match, bool) {
	for l := len(tokens); l >= 1; l-- {
		candidate := strings.Join(tokens[:l], " ")
		for _, a := range adapters {
			if m, ok := a(s, candidate); ok {
				return m, true
			}
		}
	}
	return Match{}, false
}

// tokenize splits raw command input, honoring quotes and lower-casing so
// matching is case-insensitive.
func tokenize(input string) []string {
	return util.SplitButRespectQuotes(strings.ToLower(strings.TrimSpace(input)))
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/parser/ -run TestResolveWith -v`
Expected: PASS (all four subtests).

- [ ] **Step 5: Commit**

```bash
git add internal/parser/parser.go internal/parser/parser_test.go
git commit -m "feat(parser): greedy longest-span resolution engine + core types"
```

---

## Task 2: Test scaffolding (seeded Scope)

Adapter tests need a `Scope` backed by a seeded item registry, one mob, and a
room/user. This task adds a `TestMain` + `seedParserTest` helper, mirroring the
pattern already used in `internal/usercommands/usercommands_test.go`.

**Files:**
- Modify: `internal/parser/parser_test.go` (add `TestMain` + `seedParserTest`)

- [ ] **Step 1: Add the scaffolding (no behavior test yet — this is fixture code)**

Append to `internal/parser/parser_test.go`:

```go
import (
	"os"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)
	os.Exit(m.Run())
}

// seedParserTest seeds a minimal world (items, one mob, a room with a player)
// and returns a Scope plus a cleanup func. Room 1 contains: the player Aliceia
// (user 1), a Skeleton mob (instance 100), and whatever the caller adds to the
// returned room's Items/Nouns/Containers/Corpses.
func seedParserTest(t *testing.T) (Scope, func()) {
	t.Helper()

	cleanKw := keywords.SeedKeywordsForTest()

	cleanItems := items.SeedItemsForTest(map[int]*items.ItemSpec{
		10001: {ItemId: 10001, Name: "Iron Sword", NameSimple: "Sword", Type: items.Weapon, Value: 100},
		40100: {ItemId: 40100, Name: "lake iron nodule", NameSimple: "nodule", Type: items.Object, Value: 5},
		30001: {ItemId: 30001, Name: "Healing Potion", Type: items.Potion, Uses: 3, Value: 25},
	})

	cleanMobs := mobs.SeedMobsForTest(
		map[int]*mobs.Mob{1: {MobId: 1, Zone: "TestZone", Character: characters.Character{Name: "Skeleton"}}},
		map[int]*mobs.Mob{100: {MobId: 1, InstanceId: 100, HomeRoomId: 1,
			Character: characters.Character{Name: "Skeleton", RoomId: 1, Buffs: buffs.New(), Cooldowns: map[string]int{}}}},
	)

	u := users.NewTestUser(1, "alice", "Aliceia", 1001)
	u.Character.RoomId = 1
	cleanUsers := users.SeedUsersForTest(map[int]*users.UserRecord{1: u})

	room := &rooms.Room{
		RoomId: 1, Zone: "TestZone", Title: "Test Room", Biome: "default",
		Exits: map[string]exit.RoomExit{"north": {RoomId: 2}},
	}
	cleanRooms := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{1: room},
		map[string]*rooms.ZoneConfig{"TestZone": {Name: "TestZone", RoomId: 1, RoomIds: map[int]struct{}{1: {}}}},
	)
	room.AddPlayer(1)
	room.AddMob(100)

	cleanBiomes := rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{
		"default": {BiomeId: "default", Name: "Default", Symbol: ".", LitArea: true, MovementCost: 1.0},
	})

	scope := Scope{User: u, Room: room}
	cleanup := func() {
		cleanBiomes()
		cleanRooms()
		cleanUsers()
		cleanMobs()
		cleanItems()
		cleanKw()
	}
	return scope, cleanup
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go test ./internal/parser/ -run TestResolveWith -v`
Expected: PASS (existing engine tests still green; scaffolding compiles).
If a `Seed*ForTest` signature differs, grep the reference usage in
`internal/usercommands/usercommands_test.go` (lines ~76–214) and match it.

- [ ] **Step 3: Commit**

```bash
git add internal/parser/parser_test.go
git commit -m "test(parser): seeded Scope scaffolding for adapter tests"
```

---

## Task 3: Room-only adapters (Noun, Exit, Container, Corpse)

**Files:**
- Create: `internal/parser/adapters.go`
- Test: `internal/parser/adapters_test.go`

- [ ] **Step 1: Write the failing test**

`internal/parser/adapters_test.go`:

```go
package parser

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNounAdapter_MultiWord(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	s.Room.Nouns = map[string]string{"hare paths": "Faint trails worn by hares."}

	m, ok := nounAdapter(s, "hare paths")
	require.True(t, ok)
	assert.Equal(t, KindNoun, m.Kind)
	assert.Equal(t, "hare paths", m.Name)

	_, ok = nounAdapter(s, "dragon")
	assert.False(t, ok)
}

func TestExitAdapter(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	m, ok := exitAdapter(s, "north")
	require.True(t, ok)
	assert.Equal(t, KindExit, m.Kind)
	assert.Equal(t, "north", m.Name)
}

func TestContainerAdapter(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	s.Room.Containers = map[string]rooms.Container{
		"wooden chest": {Items: []items.Item{items.New(10001)}},
	}
	m, ok := containerAdapter(s, "wooden chest")
	require.True(t, ok)
	assert.Equal(t, KindRoomContainer, m.Kind)
	assert.Equal(t, "wooden chest", m.ContainerName)
}

func TestCorpseAdapter(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	s.Room.Corpses = []rooms.Corpse{{
		MobId:     1,
		Character: characters.Character{Name: "Skeleton"},
		Loot:      rooms.Container{Items: []items.Item{items.New(10001)}},
	}}
	m, ok := corpseAdapter(s, "corpse")
	require.True(t, ok)
	assert.Equal(t, KindCorpse, m.Kind)
	assert.Equal(t, 0, m.CorpseIdx)
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/parser/ -run 'TestNounAdapter|TestExitAdapter|TestContainerAdapter|TestCorpseAdapter' -v`
Expected: FAIL — `undefined: nounAdapter` etc.

- [ ] **Step 3: Write the implementation**

`internal/parser/adapters.go`:

```go
package parser

func nounAdapter(s Scope, candidate string) (Match, bool) {
	if s.Room == nil {
		return Match{}, false
	}
	found, _ := s.Room.FindNoun(candidate)
	if found == "" {
		return Match{}, false
	}
	return Match{Kind: KindNoun, Name: found}, true
}

func exitAdapter(s Scope, candidate string) (Match, bool) {
	if s.Room == nil {
		return Match{}, false
	}
	if _, ok := s.Room.Exits[candidate]; ok {
		return Match{Kind: KindExit, Name: candidate}, true
	}
	return Match{}, false
}

func containerAdapter(s Scope, candidate string) (Match, bool) {
	if s.Room == nil {
		return Match{}, false
	}
	name := s.Room.FindContainerByName(candidate)
	if name == "" {
		return Match{}, false
	}
	return Match{Kind: KindRoomContainer, Name: name, ContainerName: name}, true
}

func corpseAdapter(s Scope, candidate string) (Match, bool) {
	if s.Room == nil {
		return Match{}, false
	}
	idx := s.Room.FindCorpseIndex(candidate)
	if idx < 0 {
		return Match{}, false
	}
	return Match{Kind: KindCorpse, Name: s.Room.Corpses[idx].DisplayName(), CorpseIdx: idx}, true
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/parser/ -run 'TestNounAdapter|TestExitAdapter|TestContainerAdapter|TestCorpseAdapter' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/parser/adapters.go internal/parser/adapters_test.go
git commit -m "feat(parser): room-scoped adapters (noun, exit, container, corpse)"
```

---

## Task 4: Item/inventory adapters (Floor, Inventory, Component, Potion)

**Files:**
- Modify: `internal/parser/adapters.go`
- Test: `internal/parser/adapters_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/parser/adapters_test.go`:

```go
func TestFloorItemAdapter(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	s.Room.AddItem(items.New(40100), false) // "lake iron nodule"

	m, ok := floorItemAdapter(s, "lake iron nodule")
	require.True(t, ok)
	assert.Equal(t, KindFloorItem, m.Kind)
	assert.Equal(t, 40100, m.Item.ItemId)
}

func TestInventoryItemAdapter(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	s.User.Character.StoreItem(items.New(10001)) // Iron Sword to backpack

	m, ok := inventoryItemAdapter(s, "iron sword")
	require.True(t, ok)
	assert.Equal(t, KindInventoryItem, m.Kind)
	assert.Equal(t, 10001, m.Item.ItemId)
}

func TestPotionItemAdapter(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	s.User.Character.PotionItems = append(s.User.Character.PotionItems, items.New(30001))

	m, ok := potionItemAdapter(s, "healing potion")
	require.True(t, ok)
	assert.Equal(t, KindPotionItem, m.Kind)
	assert.Equal(t, 30001, m.Item.ItemId)
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/parser/ -run 'TestFloorItemAdapter|TestInventoryItemAdapter|TestPotionItemAdapter' -v`
Expected: FAIL — `undefined: floorItemAdapter` etc.

- [ ] **Step 3: Write the implementation**

Append to `internal/parser/adapters.go`:

```go
import "github.com/GoMudEngine/GoMud/internal/items"

func floorItemAdapter(s Scope, candidate string) (Match, bool) {
	if s.Room == nil {
		return Match{}, false
	}
	it, ok := s.Room.FindOnFloor(candidate, false)
	if !ok {
		return Match{}, false
	}
	return Match{Kind: KindFloorItem, Name: it.Name(), Item: it}, true
}

func inventoryItemAdapter(s Scope, candidate string) (Match, bool) {
	if s.User == nil {
		return Match{}, false
	}
	it, source, ok := s.User.Character.FindItem(candidate)
	if !ok {
		return Match{}, false
	}
	return Match{Kind: KindInventoryItem, Name: it.Name(), Item: it, Source: source}, true
}

// matchInSlice resolves candidate against an item slice via items.FindMatchIn,
// preferring a full match over a partial one.
func matchInSlice(candidate string, list []items.Item) (items.Item, bool) {
	pMatch, fMatch := items.FindMatchIn(candidate, list...)
	if fMatch.ItemId != 0 {
		return fMatch, true
	}
	if pMatch.ItemId != 0 {
		return pMatch, true
	}
	return items.Item{}, false
}

func componentItemAdapter(s Scope, candidate string) (Match, bool) {
	if s.User == nil {
		return Match{}, false
	}
	it, ok := matchInSlice(candidate, s.User.Character.ComponentItems)
	if !ok {
		return Match{}, false
	}
	return Match{Kind: KindComponentItem, Name: it.Name(), Item: it}, true
}

func potionItemAdapter(s Scope, candidate string) (Match, bool) {
	if s.User == nil {
		return Match{}, false
	}
	it, ok := matchInSlice(candidate, s.User.Character.PotionItems)
	if !ok {
		return Match{}, false
	}
	return Match{Kind: KindPotionItem, Name: it.Name(), Item: it}, true
}
```

Note: if `internal/parser/adapters.go` already imports `items` from a prior
task, merge the import rather than adding a duplicate import block.

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/parser/ -run 'Adapter' -v`
Expected: PASS (all adapter tests, including Task 3's).

- [ ] **Step 5: Commit**

```bash
git add internal/parser/adapters.go internal/parser/adapters_test.go
git commit -m "feat(parser): item adapters (floor, inventory, component, potion)"
```

---

## Task 5: Actor adapters (Mob, Player, Pet)

**Files:**
- Modify: `internal/parser/adapters.go`
- Test: `internal/parser/adapters_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/parser/adapters_test.go`:

```go
func TestMobAdapter(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	m, ok := mobAdapter(s, "skeleton")
	require.True(t, ok)
	assert.Equal(t, KindMob, m.Kind)
	assert.Equal(t, 100, m.MobInstanceId)
}

func TestPlayerAdapter(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	m, ok := playerAdapter(s, "aliceia")
	require.True(t, ok)
	assert.Equal(t, KindPlayer, m.Kind)
	assert.Equal(t, 1, m.UserId)
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/parser/ -run 'TestMobAdapter|TestPlayerAdapter' -v`
Expected: FAIL — `undefined: mobAdapter`.

- [ ] **Step 3: Write the implementation**

Append to `internal/parser/adapters.go`:

```go
func mobAdapter(s Scope, candidate string) (Match, bool) {
	if s.Room == nil {
		return Match{}, false
	}
	_, mobInstanceId := s.Room.FindByName(candidate)
	if mobInstanceId == 0 {
		return Match{}, false
	}
	return Match{Kind: KindMob, Name: candidate, MobInstanceId: mobInstanceId}, true
}

func playerAdapter(s Scope, candidate string) (Match, bool) {
	if s.Room == nil {
		return Match{}, false
	}
	playerId, _ := s.Room.FindByName(candidate)
	if playerId == 0 {
		return Match{}, false
	}
	return Match{Kind: KindPlayer, Name: candidate, UserId: playerId}, true
}

func petAdapter(s Scope, candidate string) (Match, bool) {
	if s.Room == nil {
		return Match{}, false
	}
	playerId := s.Room.FindByPetName(candidate)
	if playerId == 0 {
		return Match{}, false
	}
	return Match{Kind: KindPet, Name: candidate, UserId: playerId}, true
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/parser/ -run Adapter -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/parser/adapters.go internal/parser/adapters_test.go
git commit -m "feat(parser): actor adapters (mob, player, pet)"
```

---

## Task 6: Public `Resolve` + the Kind→adapter registry

**Files:**
- Modify: `internal/parser/parser.go`
- Test: `internal/parser/parser_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/parser/parser_test.go`:

```go
func TestResolve_PicksRequestedKindByPriority(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	s.Room.Nouns = map[string]string{"skeleton": "A pile of old bones."}

	// Asking for Mob before Noun must return the mob, not the same-named noun.
	m, ok := Resolve(s, "skeleton", KindMob, KindNoun)
	require.True(t, ok)
	assert.Equal(t, KindMob, m.Kind)

	// Asking for Noun only returns the noun.
	m, ok = Resolve(s, "skeleton", KindNoun)
	require.True(t, ok)
	assert.Equal(t, KindNoun, m.Kind)
}

func TestResolve_MultiWordFloorItem(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	s.Room.AddItem(items.New(40100), false)

	m, ok := Resolve(s, "lake iron nodule", KindFloorItem)
	require.True(t, ok)
	assert.Equal(t, 40100, m.Item.ItemId)
}

func TestResolve_UnknownKindReturnsFalse(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	_, ok := Resolve(s, "nothing here", KindFloorItem)
	assert.False(t, ok)
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/parser/ -run TestResolve_ -v`
Expected: FAIL — `undefined: Resolve`.

- [ ] **Step 3: Write the implementation**

Append to `internal/parser/parser.go`:

```go
// registry maps each Kind to its adapter.
var registry = map[Kind]adapter{
	KindMob:           mobAdapter,
	KindPlayer:        playerAdapter,
	KindPet:           petAdapter,
	KindFloorItem:     floorItemAdapter,
	KindInventoryItem: inventoryItemAdapter,
	KindComponentItem: componentItemAdapter,
	KindPotionItem:    potionItemAdapter,
	KindRoomContainer: containerAdapter,
	KindCorpse:        corpseAdapter,
	KindNoun:          nounAdapter,
	KindExit:          exitAdapter,
}

// Resolve tokenizes input and greedily resolves the longest multi-word span
// against the requested kinds, in the order given (used as the tie-breaker at a
// given span length). Returns the best Match and whether anything resolved.
func Resolve(s Scope, input string, kinds ...Kind) (Match, bool) {
	tokens := tokenize(input)
	if len(tokens) == 0 || len(kinds) == 0 {
		return Match{}, false
	}
	adapters := make([]adapter, 0, len(kinds))
	for _, k := range kinds {
		if a, ok := registry[k]; ok {
			adapters = append(adapters, a)
		}
	}
	return resolveWith(tokens, s, adapters)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/parser/ -run TestResolve_ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/parser/parser.go internal/parser/parser_test.go
git commit -m "feat(parser): public Resolve + Kind->adapter registry"
```

---

## Task 7: Composed helpers (`ResolveItem`, `ResolveActor`, `splitOnConnective`)

**Files:**
- Create: `internal/parser/helpers.go`
- Test: `internal/parser/helpers_test.go`

- [ ] **Step 1: Write the failing test**

`internal/parser/helpers_test.go`:

```go
package parser

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitOnConnective(t *testing.T) {
	left, right, found := splitOnConnective("dagger to bank clerk", "to")
	require.True(t, found)
	assert.Equal(t, "dagger", left)
	assert.Equal(t, "bank clerk", right)

	_, _, found = splitOnConnective("dagger", "to")
	assert.False(t, found)
}

func TestResolveItem_FromCorpse(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	s.Room.Corpses = []rooms.Corpse{{
		MobId:     1,
		Character: characters.Character{Name: "Skeleton"},
		Loot:      rooms.Container{Items: []items.Item{items.New(10001)}},
	}}

	// "sword corpse" — item from the trailing corpse container, no "from".
	m, ok := ResolveItem(s, "sword corpse")
	require.True(t, ok)
	assert.Equal(t, KindCorpse, m.Kind)     // resolved via the corpse container
	assert.Equal(t, 10001, m.Item.ItemId)   // the looted item
	assert.Equal(t, 0, m.CorpseIdx)
}

func TestResolveItem_FromFloor(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	s.Room.AddItem(items.New(40100), false)

	m, ok := ResolveItem(s, "lake iron nodule")
	require.True(t, ok)
	assert.Equal(t, KindFloorItem, m.Kind)
	assert.Equal(t, 40100, m.Item.ItemId)
}

func TestResolveActor_MultiWordMob(t *testing.T) {
	s, cleanup := seedParserTest(t)
	defer cleanup()
	m, ok := ResolveActor(s, "skeleton", KindMob, KindPlayer)
	require.True(t, ok)
	assert.Equal(t, 100, m.MobInstanceId)
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/parser/ -run 'TestSplitOnConnective|TestResolveItem|TestResolveActor' -v`
Expected: FAIL — `undefined: splitOnConnective`, `undefined: ResolveItem`, `undefined: ResolveActor`.

- [ ] **Step 3: Write the implementation**

`internal/parser/helpers.go`:

```go
package parser

import "strings"

// splitOnConnective splits input on the first standalone occurrence of the
// connective word (e.g. "to"), returning the trimmed left and right sides.
// found=false when the connective is absent.
func splitOnConnective(input, connective string) (left, right string, found bool) {
	tokens := strings.Fields(input)
	for i, tok := range tokens {
		if tok == connective && i > 0 && i < len(tokens)-1 {
			return strings.Join(tokens[:i], " "), strings.Join(tokens[i+1:], " "), true
		}
	}
	return input, "", false
}

// ResolveItem is the shared get/drop/look-item ladder. It resolves an item that
// may live in a trailing container / corpse / pet ("get X from Y" or "get X Y"),
// or on the floor / in inventory. When the item comes from a container-like
// source, the returned Match carries that source (Kind + ContainerName /
// CorpseIdx) plus the item, so the command can still apply its own gates.
func ResolveItem(s Scope, input string) (Match, bool) {
	// Explicit "from <container>".
	if itemPart, containerPart, ok := splitOnConnective(input, "from"); ok {
		if cm, ok2 := Resolve(s, containerPart, KindRoomContainer, KindCorpse, KindPet); ok2 {
			return lootFromContainer(s, cm, itemPart)
		}
		return Match{}, false
	}

	// No "from": try each "<item> <container>" split, from the longest item /
	// shortest container span down. Accept the first split where the container
	// resolves AND the item resolves inside it — validating the item avoids
	// mis-stripping when the typed word differs from the container's canonical
	// name (e.g. "sword corpse" vs. canonical "Skeleton corpse").
	tokens := strings.Fields(input)
	for start := 1; start < len(tokens); start++ {
		itemPart := strings.Join(tokens[:start], " ")
		containerPart := strings.Join(tokens[start:], " ")
		if cm, ok := Resolve(s, containerPart, KindRoomContainer, KindCorpse, KindPet); ok {
			if m, ok2 := lootFromContainer(s, cm, itemPart); ok2 {
				return m, true
			}
		}
	}

	// No container: resolve the item from floor / inventory.
	return Resolve(s, input, KindFloorItem, KindInventoryItem)
}

// lootFromContainer resolves itemName inside the container/corpse identified by
// cm and returns a Match that carries both the item and the source handle.
func lootFromContainer(s Scope, cm Match, itemName string) (Match, bool) {
	switch cm.Kind {
	case KindRoomContainer:
		it, ok := s.Room.Containers[cm.ContainerName].FindItem(itemName)
		if !ok {
			return Match{}, false
		}
		return Match{Kind: KindRoomContainer, Name: it.Name(), Item: it, ContainerName: cm.ContainerName}, true
	case KindCorpse:
		it, ok := s.Room.Corpses[cm.CorpseIdx].Loot.FindItem(itemName)
		if !ok {
			return Match{}, false
		}
		return Match{Kind: KindCorpse, Name: it.Name(), Item: it, CorpseIdx: cm.CorpseIdx}, true
	}
	return Match{}, false
}

// ResolveActor resolves a mob / player / pet target from input.
func ResolveActor(s Scope, input string, kinds ...Kind) (Match, bool) {
	return Resolve(s, input, kinds...)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/parser/ -run 'TestSplitOnConnective|TestResolveItem|TestResolveActor' -v`
Expected: PASS.

- [ ] **Step 5: Run the whole package + build**

Run: `go test ./internal/parser/ -v && go build ./...`
Expected: all PASS, build OK.

- [ ] **Step 6: Commit**

```bash
git add internal/parser/helpers.go internal/parser/helpers_test.go
git commit -m "feat(parser): ResolveItem/ResolveActor/splitOnConnective composed helpers"
```

---

## Definition of Done (Stage 0)

- `internal/parser` builds and `go test ./internal/parser/` is green.
- `Resolve`, `ResolveItem`, `ResolveActor`, `splitOnConnective`, all `Kind`s, and
  their adapters exist and are unit-tested.
- **No command has been migrated** — `internal/usercommands` is untouched, and
  `go build ./...` + the full suite still pass. (Stage 1 does the first migration.)

---

## Self-Review Notes

- **Spec coverage:** Stage 0 of the spec's table = the `internal/parser`
  foundation (`Kind`/`Scope`/`Match`/`Resolve` + adapters + `ResolveItem`/
  `ResolveActor` + `splitOnConnective`). Every one of those is a task here.
- **Refinement recorded:** spec's `KindBackpackItem`+`KindEquippedItem` →
  `KindInventoryItem` (documented in the header) because `character.FindItem`
  already returns the combined pool.
- **Deferred to later stages (not this plan):** any command edits (Stage 1+),
  `spells.ResolveSpell` / `crafting.FindRecipeByName` routing (Stage 6), the
  `Match.Ambiguous` disambiguation hook (the spec left it unbuilt), and
  preposition-stripping beyond the "from" split (added per-command as needed).
- **Known Stage-0 gap (closed in Stage 1):** `lootFromContainer` handles
  room-container and corpse sources; the pet-inventory source (get.go's
  `Pet.FindItem` path) is added when `get` is migrated in Stage 1, since that's
  where the pet-ownership check lives. `KindPet` stays in the registry and
  resolves as an actor; only its loot-from-pet branch is deferred.
- **Signature risk:** if any `Seed*ForTest` / `FindItem` / `FindMatchIn`
  signature differs at implementation time, the reference usages are
  `internal/usercommands/usercommands_test.go` (seeding) and
  `internal/usercommands/get.go` (the ladder this mirrors). Match those.
