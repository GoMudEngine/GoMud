# Corpse-Based Loot Redesign — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Route mob loot into an owned, lootable **corpse container** (never the room floor), gated by ownership + a short timeout, with party loot modes (FFA/round-robin/leader-hold) and a shared party gold pool.

**Architecture:** Embed the existing `rooms.Container` struct into `rooms.Corpse` as a `Loot` field; reroute `dropMobLootAndSetCorpse` to fill it instead of `room.AddItem`/`room.Gold`. Add ownership + mode metadata on the corpse (stamped at spawn) and a loot-mode + gold-pool on the in-memory `parties.Party`. Reuse the existing chest get/put wiring, the highest-damager + same-room-party kill-credit patterns, and the corpse decay/prune loop. **Mob corpses only — player corpses are never touched.**

**Tech Stack:** Go (GoMud engine). Unit tests via `go test` (packages `internal/rooms`, `internal/parties`, `internal/hooks`, `internal/actions` already have `_test.go` files). Boot-validation via `go build -o gomud_smoke.exe .` + local run. Multi-party behavior via the `/playtest` harness.

**Spec:** `docs/superpowers/specs/2026-07-07-corpse-loot-redesign-design.md`

---

## Verified code facts (2026-07-07)

- **`rooms.Corpse`** (`internal/rooms/corpse.go:8-21`) — carries a full `characters.Character` copy + `MobId`/`UserId`/`RoundCreated`/`Prunable`/`WasCharmed`/`CorpseName`. `Update()` (33-51) sets `Prunable` on decay.
- **`rooms.Container`** (`internal/rooms/container.go:8-16`) — `Items []items.Item`, `Gold int`, `AddItem`/`RemoveItem`/`FindItem`/`Count`. Reuse verbatim.
- **`dropMobLootAndSetCorpse(m *mobs.Mob, room *rooms.Room)`** (`internal/hooks/Death_MobLoot.go:25`) — drops carried items (`ShouldDrop(100)`), equipped items (skip `NeverDrops`, gate `ShouldDrop(m.ItemDropChance)`), gold (`room.Gold += m.Character.Gold`) to the floor, then `room.AddCorpse(...)` (88-95). Whole block suppressed by `buffs.PermaGear`. Called from `Death_MobInstanceCleanup.go:59` **before** the instance is destroyed, so `m.Character.PlayerDamage` is live here.
- **`Character.PlayerDamage map[int]int`** (`internal/characters/character.go:279`) — key=userId, value=cumulative damage.
- **Kill-credit patterns:** highest-damager loop `MobDeath_BountyClaim.go:42-48`; same-room-party expansion `buildKillerSet` `MobDeath_FactionRep.go:165-189` (iterates `party.UserIds`, includes members whose `Character.RoomId == evt.RoomId`).
- **`Room.AddCorpse(c Corpse)`** (`rooms.go:195`); **`RemoveCorpse(c Corpse) bool`** (199, matches MobId+UserId+Name+RoundCreated); **`FindCorpse(name) (Corpse, bool)`** (1189, returns a VALUE copy); corpses live in the `r.Corpses []Corpse` slice.
- **Prune/decay loop `UpdateCorpses(roundNow)`** (`rooms.go:221-247`) — `if corpse.Prunable {` block (232) is the drop-on-decay hook; corpse is written back via `r.Corpses[idx] = corpse`.
- **`get.go`** (`internal/usercommands/get.go`) — container branch 311-385 (value-copy `container := room.Containers[name]` → mutate → write back `room.Containers[name] = container`; gold prefix-match `go`/`gol`/`gold`; `container.FindItem(rest)` → `user.Character.StoreItem` → `container.RemoveItem` → `events.ItemOwnership`). Corpse non-pickup guard 514-517 (`room.FindCorpse(rest)` → "You can't pick up corpses").
- **`parties.Party`** (`internal/parties/parties.go:34-70`) — `LeaderUserId int`, `UserIds []int`, `partyId int`; `Get(userId) *Party` (138), `IsLeader(userId) bool` (175), `IsMember` (238), `GetMembers() []int` (308), `Leave(userId) bool` (211), `Disband()` (299).
- **`party.go`** (`internal/usercommands/party.go`) — `Party(...)` dispatch 20-99 (`partyCommand := strings.ToLower(args[0])`); `cmdPartyLeave` 406-478, `cmdPartyDisband` 480-507, `cmdPartyKick` 509-534, `cmdPartyPromote` 536-564, `cmdPartyAccept`/join.
- **Salvage:** `usercommands/salvage.go` `startCorpseSalvage(user, corpse)` (125; guards at 128/133/139-143); resolver `actions/salvage.go salvageCorpse(...)` removes at 144 (`room.RemoveCorpse(target)`).
- **Raise:** `internal/hooks/companion_summon.go:63-114` — corpse-selection loop (skips `Prunable`, `UserId != 0`, `WasCharmed`), removes inline via `room.Corpses = append(...)` at ~111. Spell fields `SummonRequiresCorpse`/`SummonMinCorpsePool` (`spells.go:48-49`).
- **`Room.AddItem(item, stash bool)`** (`rooms.go:1055`), **`Room.Gold`** (`rooms.go:98`).
- **Config:** `Death` struct `config.gameplay.go:38-39` (`CorpsesEnabled`, `CorpseDecayTime`); defaults applied ~86 (`CorpseDecayTime` → `1 hour`).

---

## Task 1: Add the Loot container + HasLoot + ownership/mode fields to Corpse

**Files:**
- Modify: `internal/rooms/corpse.go`
- Test: `internal/rooms/corpse_test.go` (create)

- [ ] **Step 1: Write the failing test.**

```go
package rooms

import "testing"

func TestCorpse_HasLoot(t *testing.T) {
	var c Corpse
	if c.HasLoot() {
		t.Fatal("empty corpse should not report loot")
	}
	c.Loot.Gold = 5
	if !c.HasLoot() {
		t.Fatal("corpse with gold should report loot")
	}
	c.Loot.Gold = 0
	c.Loot.AddItem(items.New(1)) // any valid itemid
	if !c.HasLoot() {
		t.Fatal("corpse with an item should report loot")
	}
}
```

- [ ] **Step 2: Run it — expect FAIL** (`HasLoot`/`Loot` undefined).

Run: `go test ./internal/rooms/ -run TestCorpse_HasLoot`
Expected: compile error / FAIL.

- [ ] **Step 3: Add the fields + method.** In `internal/rooms/corpse.go`, extend the struct and add `HasLoot`:

```go
type Corpse struct {
	UserId       int
	MobId        int
	Character    characters.Character
	RoundCreated uint64
	Prunable     bool
	WasCharmed   bool

	CorpseName        string
	CorpseDescription string

	// Corpse-loot redesign (2026-07-07): mob loot lives here, not on the floor.
	Loot            Container // items + gold looted from the dead mob
	OwnerUserIds    []int     // who may loot before RoundOwnedUntil (empty = anyone)
	LootMode        string    // "ffa" | "roundrobin" | "leaderhold" (party mode snapshot; "" = solo/ffa)
	RoundOwnedUntil uint64    // round at which ownership opens to free-for-all
	RRAssignee      []int     // round-robin only: parallel to Loot.Items, itemIdx -> ownerUserId (0 = unassigned)
}

// HasLoot reports whether the corpse still holds any items or gold.
// Gates salvage / raise / drop-on-decay.
func (c *Corpse) HasLoot() bool {
	return len(c.Loot.Items) > 0 || c.Loot.Gold > 0
}
```

Add the `items` import if needed (the file already imports `characters` + `gametime`).

- [ ] **Step 4: Run the test — expect PASS.**

Run: `go test ./internal/rooms/ -run TestCorpse_HasLoot`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
git add internal/rooms/corpse.go internal/rooms/corpse_test.go
git commit -m "feat(loot): Corpse gains embedded Loot container + HasLoot + owner/mode fields"
```

---

## Task 2: Route mob loot into the corpse instead of the floor

**Files:**
- Modify: `internal/hooks/Death_MobLoot.go`
- Test: `internal/hooks/Death_MobLoot_test.go` (create)

- [ ] **Step 1: Write the failing test.** Build a mob with carried items + gold, call `dropMobLootAndSetCorpse`, assert the room floor is empty and the new corpse's `Loot` holds them.

```go
func TestDropMobLoot_GoesIntoCorpse(t *testing.T) {
	room := rooms.NewRoom(0)
	m := mobs.NewMobById(1, 0) // any base mob id present in test data
	m.Character.Items = []items.Item{items.New(1)}
	m.Character.Gold = 42

	dropMobLootAndSetCorpse(m, room)

	if len(room.Items) != 0 || room.Gold != 0 {
		t.Fatalf("loot leaked to the floor: items=%d gold=%d", len(room.Items), room.Gold)
	}
	if len(room.Corpses) != 1 {
		t.Fatalf("expected 1 corpse, got %d", len(room.Corpses))
	}
	c := room.Corpses[0]
	if c.Loot.Gold != 42 || len(c.Loot.Items) != 1 {
		t.Fatalf("corpse loot wrong: gold=%d items=%d", c.Loot.Gold, len(c.Loot.Items))
	}
}
```

(Adjust `mobs.NewMobById` / `rooms.NewRoom` to the real constructors — grep the existing `_test.go` files in those packages for the exact helpers, e.g. `salvage_test.go` builds corpses/rooms.)

- [ ] **Step 2: Run — expect FAIL** (loot still on floor).

Run: `go test ./internal/hooks/ -run TestDropMobLoot_GoesIntoCorpse`
Expected: FAIL (floor not empty).

- [ ] **Step 3: Reroute the loot.** In `internal/hooks/Death_MobLoot.go`, replace the three `room.AddItem(...)` / `room.Gold += ...` sinks with a locally-built `loot rooms.Container`, keeping the PermaGear / NeverDrops / ShouldDrop gating untouched. Then attach it to the corpse:

```go
func dropMobLootAndSetCorpse(m *mobs.Mob, room *rooms.Room) {
	currentRound := util.GetRoundCount()

	var loot rooms.Container

	if !m.Character.HasBuffFlag(buffs.PermaGear) {
		for _, item := range m.Character.Items {
			if !item.ShouldDrop(100) {
				continue
			}
			loot.AddItem(item)
		}
		for _, item := range m.Character.Equipment.GetAllItems() {
			if item.GetSpec().NeverDrops {
				continue
			}
			if !item.ShouldDrop(m.ItemDropChance) {
				continue
			}
			loot.AddItem(item)
		}
		if m.Character.Gold > 0 {
			loot.Gold += m.Character.Gold
		}
	}

	config := configs.GetGamePlayConfig()
	if config.Death.CorpsesEnabled {
		room.AddCorpse(rooms.Corpse{
			MobId:             int(m.MobId),
			Character:         m.Character,
			RoundCreated:      currentRound,
			WasCharmed:        m.Character.IsCharmed() || m.Character.EverCharmed,
			CorpseName:        m.CorpseName,
			CorpseDescription: m.CorpseDescription,
			Loot:              loot,
			// Owner/mode/timeout stamped in Task 7.
		})
	}
	// NOTE: if CorpsesEnabled is false, there is no corpse to hold loot.
	// Fall back to the old floor-drop in that case (see Step 3b).
}
```

- [ ] **Step 3b: Preserve floor-drop when corpses are disabled.** Wrap the old per-item `room.SendTextVisual(...)` + `room.AddItem(item, false)` + `room.Gold += ...` block in an `else` for `if config.Death.CorpsesEnabled { ... } else { /* old floor drop from `loot` */ }`, so a server with `CorpsesEnabled:false` keeps today's behavior. Drop from the built `loot` container: `for _, it := range loot.Items { room.AddItem(it, false) }` + `room.Gold += loot.Gold`, with the existing loot messages + dark-room clatter.

- [ ] **Step 4: Run — expect PASS.** Also run the whole hooks package to catch regressions.

Run: `go test ./internal/hooks/ -run TestDropMobLoot_GoesIntoCorpse && go test ./internal/hooks/`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
git add internal/hooks/Death_MobLoot.go internal/hooks/Death_MobLoot_test.go
git commit -m "feat(loot): mob loot fills the corpse container, not the room floor (floor-drop kept when corpses disabled)"
```

---

## Task 3: Loot a corpse — look / loot / get from corpse (get.go)

**Files:**
- Modify: `internal/rooms/rooms.go` (add a mutate-in-place corpse helper)
- Modify: `internal/usercommands/get.go`
- Modify: `internal/usercommands/look.go` (show corpse loot on `look <corpse>`)

- [ ] **Step 1: Add a corpse write-back helper to `rooms.go`.** `FindCorpse` returns a value copy, so looting needs index access. Add:

```go
// FindCorpseIndex returns the slice index of the first non-prunable corpse
// matching searchName, or -1. Callers mutate r.Corpses[idx] directly.
func (r *Room) FindCorpseIndex(searchName string) int {
	// Mirror FindCorpse's matching (rooms.go:1189) but return the index.
	for idx := range r.Corpses {
		if r.Corpses[idx].Prunable {
			continue
		}
		if strings.EqualFold(r.Corpses[idx].DisplayName(), searchName) ||
			strings.Contains(strings.ToLower(r.Corpses[idx].DisplayName()), strings.ToLower(searchName)) {
			return idx
		}
	}
	return -1
}
```

(Match `FindCorpse`'s exact name-building — read `rooms.go:1189` and reuse its `Name + " corpse"` logic so `get sword from wolf corpse` resolves the same way `look` does.)

- [ ] **Step 2: Add the corpse-loot branch to `get.go`.** In the `get <item> from <target>` resolution (near the container branch 311-385), when the trailing target names a corpse (not a room container), operate on `room.Corpses[idx].Loot`:

```go
// After the existing container branch fails to resolve a room Container,
// try a corpse by the same trailing-name.
corpseIdx := room.FindCorpseIndex(containerName) // containerName already parsed off "from <x>"
if corpseIdx >= 0 {
	corpse := &room.Corpses[corpseIdx]

	// Ownership + mode gate (Tasks 8 + 10 fill these in; for Task 3 allow all).
	if !canLootCorpse(user, corpse) { // stub in Task 3: return true
		user.SendText(messaging.CategorySystem, `This isn't your kill.`)
		return true, nil
	}

	// gold
	if isGoldWord(args[0]) {
		amt := corpse.Loot.Gold
		// (parse an explicit amount like the container path does)
		corpse.Loot.Gold -= amt
		grantCorpseGold(user, amt) // Task 3 stub: user.Character.Gold += amt. Task 11 routes to the party pool.
		return true, nil
	}

	matchItem, found := corpse.Loot.FindItem(rest)
	if !found {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You don't see a %s in the %s.`, rest, corpse.DisplayName()))
		return true, nil
	}
	if user.Character.StoreItem(matchItem) {
		events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: matchItem, Gained: true})
		corpse.Loot.RemoveItem(matchItem)
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You take the %s from the %s.`, matchItem.DisplayName(), corpse.DisplayName()))
		sendEncumbranceWarning(user)
	} else {
		user.SendText(messaging.CategorySystem, `You can't carry any more.`)
	}
	return true, nil
}
```

For Task 3, define `canLootCorpse` to always return `true` and `grantCorpseGold` to add straight to `user.Character.Gold` — Tasks 8/10/11 replace these stubs. Because `corpse := &room.Corpses[corpseIdx]` is a pointer into the slice, mutations persist with no write-back needed.

- [ ] **Step 3: Add a bare `loot <corpse>` command** (takes everything the looter is entitled to). Simplest: register `loot` as an alias that expands to iterate the corpse's items calling the same take-logic, then the gold. Put the shared take-logic in a helper `lootCorpseAll(user, room, corpseIdx)` called by both `loot <corpse>` and used by round-robin in Task 10. (Grep `usercommands` for the command-registration table to add `loot`.)

- [ ] **Step 4: Show loot on `look <corpse>`.** In `internal/usercommands/look.go`, where a corpse is examined, append its `Loot` contents (item list + gold) like a chest. Read the existing container-look rendering and mirror it.

- [ ] **Step 5: Build + manual verify.**

Run: `go build -o gomud_smoke.exe . && echo BUILT`
Then boot, kill a mob, confirm: no items on the floor, `look <mob> corpse` shows the loot, `get <item> from <corpse>` and `loot <corpse>` transfer it, `get gold from <corpse>` works.
Expected: loot flows through the corpse; floor stays clean.

- [ ] **Step 6: Commit.**

```bash
git add internal/rooms/rooms.go internal/usercommands/get.go internal/usercommands/look.go
git commit -m "feat(loot): look/loot/get items+gold from a corpse container"
```

---

## Task 4: Drop remaining loot to the floor on corpse decay

**Files:**
- Modify: `internal/rooms/rooms.go` (`UpdateCorpses`)
- Test: `internal/rooms/rooms_corpsedecay_test.go` (create)

- [ ] **Step 1: Write the failing test.** A corpse with loot, forced `Prunable`, run through `UpdateCorpses`, assert the loot is on the floor and the corpse is gone.

```go
func TestUpdateCorpses_DropsLootOnDecay(t *testing.T) {
	room := NewRoom(0)
	c := Corpse{MobId: 1, RoundCreated: 1}
	c.Loot.Gold = 10
	c.Loot.AddItem(items.New(1))
	c.Prunable = true // force decay this tick
	room.AddCorpse(c)

	room.UpdateCorpses(999999999)

	if room.Gold != 10 || len(room.Items) != 1 {
		t.Fatalf("decayed loot not on floor: gold=%d items=%d", room.Gold, len(room.Items))
	}
	if len(room.Corpses) != 0 {
		t.Fatalf("corpse not pruned: %d", len(room.Corpses))
	}
}
```

- [ ] **Step 2: Run — expect FAIL.**

Run: `go test ./internal/rooms/ -run TestUpdateCorpses_DropsLootOnDecay`
Expected: FAIL (loot lost).

- [ ] **Step 3: Drop loot in the prune block.** In `UpdateCorpses` (`rooms.go:221-247`), inside `if corpse.Prunable {` (before appending to `removeIdx`):

```go
if corpse.Prunable {
	if corpse.HasLoot() {
		for _, it := range corpse.Loot.Items {
			r.AddItem(it, false)
		}
		r.Gold += corpse.Loot.Gold
	}
	removeIdx = append(removeIdx, idx)
	// ... existing "crumbles to dust" messages ...
}
```

- [ ] **Step 4: Run — expect PASS** (+ the rooms package).

Run: `go test ./internal/rooms/`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
git add internal/rooms/rooms.go internal/rooms/rooms_corpsedecay_test.go
git commit -m "feat(loot): unlooted corpse loot drops to the floor on decay"
```

---

## Task 5: Salvage + raise refuse a loot-bearing corpse

**Files:**
- Modify: `internal/usercommands/salvage.go` (`startCorpseSalvage`)
- Modify: `internal/actions/salvage.go` (`salvageCorpse` resolver — belt & suspenders)
- Modify: `internal/hooks/companion_summon.go` (raise corpse-selection loop)
- Test: extend `internal/mobcommands/salvage_test.go` or add `internal/usercommands/salvage_test.go`

- [ ] **Step 1: Write the failing test.** A corpse with loot fed to the salvage-start path returns the "pick it clean" refusal and does NOT start the activity.

```go
func TestSalvage_RefusesLootBearingCorpse(t *testing.T) {
	// build a corpse with a valid salvage table AND loot in it
	corpse := buildAnimalCorpse() // reuse the salvage_test helper
	corpse.Loot.Gold = 1
	ok, _ := startCorpseSalvage(testUser, corpse)
	// assert the user got the refusal message and no salvage activity started
	// (inspect testUser's sent messages / MiscData salvage_corpse_round_created absent)
}
```

- [ ] **Step 2: Run — expect FAIL.**

Run: `go test ./internal/usercommands/ -run TestSalvage_RefusesLootBearingCorpse`
Expected: FAIL.

- [ ] **Step 3: Add the guards.**

In `startCorpseSalvage` (`salvage.go`, near the 139-143 empty-table guard):
```go
if corpse.HasLoot() {
	user.SendText(messaging.CategorySystem, `That corpse still has loot on it. Pick it clean first.`)
	return true, nil
}
```

In `salvageCorpse` (`actions/salvage.go`, right before `room.RemoveCorpse(target)` at 144):
```go
if target.HasLoot() {
	return SalvageResult{ /* failure: corpse still holds loot */ }
}
```

In `companion_summon.go` corpse-selection loop (`63-114`), add a filter next to the `WasCharmed` skip:
```go
if c.HasLoot() {
	continue // don't consume a corpse that still holds loot
}
```

- [ ] **Step 4: Run — expect PASS** (+ salvage + hooks packages).

Run: `go test ./internal/usercommands/ ./internal/actions/ ./internal/hooks/`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
git add internal/usercommands/salvage.go internal/actions/salvage.go internal/hooks/companion_summon.go internal/usercommands/salvage_test.go
git commit -m "feat(loot): salvage + raise refuse a corpse that still holds loot"
```

---

## Task 6: Config knob — Death.CorpseLootTimeout

**Files:**
- Modify: `internal/configs/config.gameplay.go`
- Modify: `_datafiles/config.yaml` (document the default)

- [ ] **Step 1: Add the field + default.** In the `Death` struct (`config.gameplay.go:38-39` area):

```go
CorpseLootTimeout ConfigString `yaml:"CorpseLootTimeout"` // real-time duration ownership holds before free-for-all
```

In the defaults block (~line 86, next to the `CorpseDecayTime` default):
```go
if g.Death.CorpseLootTimeout == `` {
	g.Death.CorpseLootTimeout = `4 minutes`
}
```

- [ ] **Step 2: Build to confirm it parses.**

Run: `go build ./internal/configs/ && echo OK`
Expected: `OK`.

- [ ] **Step 3: Commit.**

```bash
git add internal/configs/config.gameplay.go _datafiles/config.yaml
git commit -m "feat(loot): add Death.CorpseLootTimeout config (default 4 minutes)"
```

---

## Task 7: Stamp the owner set + mode + timeout at corpse spawn

**Files:**
- Create: `internal/hooks/corpse_ownership.go` (owner-set helper)
- Modify: `internal/hooks/Death_MobLoot.go` (stamp onto the corpse)
- Test: `internal/hooks/corpse_ownership_test.go` (create)

- [ ] **Step 1: Write the failing test** for the pure owner-set helper.

```go
func TestComputeCorpseOwners_SoloAndParty(t *testing.T) {
	// solo: PlayerDamage has one user, no party -> owner = [that user]
	owners := computeCorpseOwners(map[int]int{7: 100}, 0 /*roomId*/)
	if len(owners) != 1 || owners[0] != 7 {
		t.Fatalf("solo owner wrong: %v", owners)
	}
	// (party case exercised via a fake party in a follow-up if the parties
	//  registry is testable; otherwise cover the highest-damager fallback.)
}
```

- [ ] **Step 2: Run — expect FAIL.**

Run: `go test ./internal/hooks/ -run TestComputeCorpseOwners_SoloAndParty`
Expected: FAIL (undefined).

- [ ] **Step 3: Implement the helper** in `corpse_ownership.go`, reusing the highest-damager + `buildKillerSet` patterns:

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// computeCorpseOwners returns the userIds allowed to loot the corpse before
// the free-for-all timeout: the damagers, plus their same-room party members.
// Empty when a mob/environment killed it (no player owner -> anyone loots).
func computeCorpseOwners(playerDamage map[int]int, roomId int) []int {
	set := map[int]struct{}{}
	for userId := range playerDamage {
		set[userId] = struct{}{}
	}
	for damagerUserId := range playerDamage {
		p := parties.Get(damagerUserId)
		if p == nil {
			continue
		}
		for _, memberId := range p.UserIds {
			if _, ok := set[memberId]; ok {
				continue
			}
			if u := users.GetByUserId(memberId); u != nil && u.Character.RoomId == roomId {
				set[memberId] = struct{}{}
			}
		}
	}
	out := make([]int, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	return out
}

// corpseLootMode returns the party's loot mode for the killer, or "ffa" solo.
func corpseLootMode(playerDamage map[int]int) string {
	for userId := range playerDamage {
		if p := parties.Get(userId); p != nil {
			return p.LootMode // Task 9 adds this field ("" defaults to ffa)
		}
	}
	return "ffa"
}
```

- [ ] **Step 4: Stamp onto the corpse** in `Death_MobLoot.go` (the `AddCorpse` call from Task 2). Compute a `RoundOwnedUntil` from the config duration + `currentRound` (use the same `gametime.GetDate(...).AddPeriod(dur)` pattern `Corpse.Update` uses for decay, but resolve to a round number). Add:

```go
owners := computeCorpseOwners(m.Character.PlayerDamage, room.RoomId)
mode := corpseLootMode(m.Character.PlayerDamage)
ownedUntil := lootTimeoutRound(currentRound, config.Death.CorpseLootTimeout.String())
// ... in the Corpse literal:
OwnerUserIds:    owners,
LootMode:        mode,
RoundOwnedUntil: ownedUntil,
```

Implement `lootTimeoutRound(now uint64, dur string) uint64` mirroring `Corpse.Update`'s `gametime.GetDate(now).AddPeriod(dur)`.

- [ ] **Step 5: Run — expect PASS** (+ hooks package).

Run: `go test ./internal/hooks/`
Expected: PASS.

- [ ] **Step 6: Commit.**

```bash
git add internal/hooks/corpse_ownership.go internal/hooks/corpse_ownership_test.go internal/hooks/Death_MobLoot.go
git commit -m "feat(loot): stamp owner set + loot mode + free-for-all timeout at corpse spawn"
```

---

## Task 8: Enforce ownership + timeout in corpse looting

**Files:**
- Modify: `internal/usercommands/get.go` (replace the Task 3 `canLootCorpse` stub)

- [ ] **Step 1: Implement `canLootCorpse`.**

```go
func canLootCorpse(user *users.UserRecord, corpse *rooms.Corpse) bool {
	// Free-for-all after the timeout, or when there was no player owner.
	if util.GetRoundCount() >= corpse.RoundOwnedUntil || len(corpse.OwnerUserIds) == 0 {
		return true
	}
	for _, id := range corpse.OwnerUserIds {
		if id == user.UserId {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Build + manual/harness verify** two chars: A kills a mob solo → B cannot loot ("This isn't your kill") until ~4 min pass, then B can.

Run: `go build -o gomud_smoke.exe . && echo BUILT` then drive two sessions (see the harness rig in `docs/ENDGAME_COMBAT_TUNING.md` §5 for the mudagent 2-bridge recipe; use two non-partied chars).
Expected: non-owner blocked pre-timeout, allowed post-timeout.

- [ ] **Step 3: Commit.**

```bash
git add internal/usercommands/get.go
git commit -m "feat(loot): gate corpse looting by owner set until the free-for-all timeout"
```

---

## Task 9: Party loot mode field + `party loot <mode>` command

**Files:**
- Modify: `internal/parties/parties.go` (add `LootMode string`)
- Modify: `internal/usercommands/party.go` (`party loot <mode>` subcommand)
- Test: `internal/parties/parties_lootmode_test.go` (create)

- [ ] **Step 1: Write the failing test.**

```go
func TestParty_LootModeDefaultAndSet(t *testing.T) {
	p := New(1)
	if p.LootMode != "" && p.LootMode != "ffa" {
		t.Fatalf("new party should default to ffa/empty, got %q", p.LootMode)
	}
	p.LootMode = "roundrobin"
	if Get(1).LootMode != "roundrobin" {
		t.Fatal("loot mode not persisted on the registry party")
	}
	p.Disband()
}
```

- [ ] **Step 2: Run — expect FAIL.**

Run: `go test ./internal/parties/ -run TestParty_LootModeDefaultAndSet`
Expected: FAIL.

- [ ] **Step 3: Add `LootMode string` to the `Party` struct** (`parties.go:34-70`). Treat `""` as `ffa` everywhere it's read.

- [ ] **Step 4: Add the `party loot <mode>` subcommand** in `party.go` dispatch (mirror `cmdPartyKick`). Leader-only; accept `ffa` / `roundrobin` (aliases `rr`, `round-robin`) / `leaderhold` (alias `leader`); reject others with the valid list; echo the change to the party.

```go
if partyCommand == `loot` {
	return cmdPartyLoot(user, currentParty, rest)
}
```
`cmdPartyLoot` validates `currentParty.IsLeader(user.UserId)`, normalizes `rest` to a canonical mode, sets `currentParty.LootMode`, and announces it.

- [ ] **Step 5: Run — expect PASS.**

Run: `go test ./internal/parties/`
Expected: PASS.

- [ ] **Step 6: Commit.**

```bash
git add internal/parties/parties.go internal/usercommands/party.go internal/parties/parties_lootmode_test.go
git commit -m "feat(loot): party LootMode field + 'party loot <mode>' (leader-only, default ffa)"
```

---

## Task 10: Enforce loot modes (FFA / leader-hold / round-robin)

**Files:**
- Modify: `internal/usercommands/get.go` (mode enforcement in the corpse branch + `loot <corpse>`)
- Create: `internal/rooms/corpse_roundrobin.go` (assignment helper)
- Test: `internal/rooms/corpse_roundrobin_test.go` (create)

- [ ] **Step 1: Write the failing test** for the pure round-robin assignment.

```go
func TestAssignRoundRobin(t *testing.T) {
	// 3 items, 2 members [10,20], cursor start 0 -> [10,20,10], cursor ends at 1
	assignee, cursor := AssignRoundRobin(3, []int{10, 20}, 0)
	want := []int{10, 20, 10}
	for i := range want {
		if assignee[i] != want[i] {
			t.Fatalf("assignee[%d]=%d want %d", i, assignee[i], want[i])
		}
	}
	if cursor != 1 {
		t.Fatalf("cursor=%d want 1", cursor)
	}
}
```

- [ ] **Step 2: Run — expect FAIL.**

Run: `go test ./internal/rooms/ -run TestAssignRoundRobin`
Expected: FAIL.

- [ ] **Step 3: Implement `AssignRoundRobin(nItems int, members []int, startCursor int) (assignee []int, endCursor int)`** in `corpse_roundrobin.go` (round-robin over `members`, returns the parallel assignee slice + the advanced cursor).

- [ ] **Step 4: Wire it in.**
  - Add a per-party round-robin cursor: `RRCursor int` on the `Party` struct (persists across corpses).
  - In Task 7's stamping, when `mode == "roundrobin"`, call `AssignRoundRobin(len(loot.Items), owners, party.RRCursor)`, store the result in `corpse.RRAssignee`, and write the advanced cursor back to the party.
  - In `get.go` corpse branch, enforce by mode:
    - **ffa** (or `""`): any owner may take any item (already the Task 3/8 behavior).
    - **leaderhold**: only `party.IsLeader(user.UserId)` may take items (others get "the leader is holding loot").
    - **roundrobin**: a member may take item `i` only if `corpse.RRAssignee[i] == user.UserId` (or after the free-for-all timeout). Provide `loot pass <corpse>` → reassign the passer's still-present assigned items to the next member (advance through `owners`).
  - `loot <corpse>` respects the same mode (takes only what the looter is entitled to).

- [ ] **Step 5: Run unit test — expect PASS; then build + harness-verify** a 3-char party through each mode (see Task 13).

Run: `go test ./internal/rooms/ && go build -o gomud_smoke.exe . && echo BUILT`
Expected: PASS + BUILT.

- [ ] **Step 6: Commit.**

```bash
git add internal/usercommands/get.go internal/rooms/corpse_roundrobin.go internal/rooms/corpse_roundrobin_test.go internal/parties/parties.go
git commit -m "feat(loot): enforce ffa/leader-hold/round-robin loot modes (with round-robin pass)"
```

---

## Task 11: Shared party gold pool — accrue on loot

**Files:**
- Modify: `internal/parties/parties.go` (`GoldPool int`)
- Modify: `internal/usercommands/get.go` (`grantCorpseGold` routes to the pool in a party)
- Test: `internal/parties/parties_goldpool_test.go` (create)

- [ ] **Step 1: Write the failing test.**

```go
func TestParty_GoldPoolAccrue(t *testing.T) {
	p := New(1)
	p.AddGold(30) // helper to add to pool
	if Get(1).GoldPool != 30 {
		t.Fatalf("pool=%d want 30", Get(1).GoldPool)
	}
	p.Disband()
}
```

- [ ] **Step 2: Run — expect FAIL.**

Run: `go test ./internal/parties/ -run TestParty_GoldPoolAccrue`
Expected: FAIL.

- [ ] **Step 3: Add `GoldPool int` + `AddGold(n int)`** to `Party`. Replace the Task 3 `grantCorpseGold` stub in `get.go`:

```go
func grantCorpseGold(user *users.UserRecord, amt int) {
	if p := parties.Get(user.UserId); p != nil {
		p.AddGold(amt) // into the shared pool
		return
	}
	user.Character.Gold += amt // solo -> straight to purse
}
```

- [ ] **Step 4: Run — expect PASS.**

Run: `go test ./internal/parties/`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
git add internal/parties/parties.go internal/usercommands/get.go internal/parties/parties_goldpool_test.go
git commit -m "feat(loot): corpse gold accrues into a shared party gold pool (solo -> purse)"
```

---

## Task 12: Settle-split the gold pool on membership changes + on demand

**Files:**
- Modify: `internal/parties/parties.go` (`SettleGold()` + call in `Leave`/`Disband`)
- Modify: `internal/usercommands/party.go` (`party gold split` + settle in leave/disband/kick/promote handlers)
- Test: `internal/parties/parties_goldsplit_test.go` (create)

- [ ] **Step 1: Write the failing test** for the split math (even split + remainder handling).

```go
func TestParty_SettleGold_EvenSplitWithRemainder(t *testing.T) {
	p := New(1)          // leader user 1
	p.InvitePlayer(2); p.AcceptInvite(2)
	p.InvitePlayer(3); p.AcceptInvite(3)
	p.GoldPool = 10      // 3 members -> 3 each, remainder 1
	paid := p.SettleGold() // returns map[userId]amount
	total := 0
	for _, v := range paid { total += v }
	if total != 10 {
		t.Fatalf("split lost gold: paid %d of 10", total)
	}
	if p.GoldPool != 0 {
		t.Fatalf("pool not reset: %d", p.GoldPool)
	}
	p.Disband()
}
```

(Decide remainder rule: give the leftover coin(s) to the leader; assert total is conserved.)

- [ ] **Step 2: Run — expect FAIL.**

Run: `go test ./internal/parties/ -run TestParty_SettleGold_EvenSplitWithRemainder`
Expected: FAIL.

- [ ] **Step 3: Implement `SettleGold() map[int]int`** — split `GoldPool` evenly across `UserIds`, remainder to the leader, credit each member's `Character.Gold` (via `users.GetByUserId`), zero the pool, return the payout map. Call it **before** the membership change in `Leave` (211) and at the top of `Disband` (299).

- [ ] **Step 4: Wire the command layer.** In `party.go`, call `currentParty.SettleGold()` at the top of `cmdPartyLeave`, `cmdPartyDisband`, `cmdPartyKick`, and on `cmdPartyAccept` (settle to existing members *before* the joiner is added). Add a `party gold split` subcommand (`cmdPartyGold`) that calls `SettleGold()` on demand and reports each payout. Announce payouts to the party.

- [ ] **Step 5: Run — expect PASS** (+ parties package).

Run: `go test ./internal/parties/`
Expected: PASS.

- [ ] **Step 6: Commit.**

```bash
git add internal/parties/parties.go internal/usercommands/party.go internal/parties/parties_goldsplit_test.go
git commit -m "feat(loot): settle-split the party gold pool on join/leave/disband + 'party gold split'"
```

---

## Task 13: Full verification, help templates, memory

**Files:**
- Create/modify: help templates for `loot` + `party loot` + `party gold` (`_datafiles/world/dogmud/templates/help/`)
- Modify: memory topic file

- [ ] **Step 1: Full suite + clean boot.**

Run: `go test ./... 2>&1 | tail -30` (expect no new failures) then
`rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/* && go build -o gomud_smoke.exe . && echo BUILT`, boot, confirm `errors=0 mode=panic`, no panic.
Expected: green + clean boot.

- [ ] **Step 2: Harness playtest — the multi-party matrix.** Using the 2-3 bridge conductor rig (`docs/ENDGAME_COMBAT_TUNING.md` §5 recipe; quester4/5/6):
  - Solo kill → killer loots; a second non-owner blocked until timeout.
  - Party FFA → any member loots; party round-robin → items split by rotation, `loot pass` reassigns; leader-hold → only leader loots.
  - Corpse gold → pool; `party gold split` pays out; a member `leave` triggers a settle payout.
  - Try to `salvage` / `raise` a loot-bearing corpse → refused; after looting it clean → allowed.
  - Let a corpse decay with loot in it → loot appears on the floor.
  Record a short report under `tools/playtest/reports/`.

- [ ] **Step 3: Help templates.** Add/entend help for `loot`, `party loot`, `party gold` (player-facing, 80-col). Verify the help-completeness test still passes (`go test ./... -run Help` or the relevant test).

- [ ] **Step 4: Update memory.** Mark [[project_loot_drop_redesign_future]] BUILT + record the final command surface + any gotchas found. Note the pre-push SOP is owed (PATCH_NOTES, LogToFile:false, droplet deploy + perf datapoint) — the user pushes.

- [ ] **Step 5: Commit.**

```bash
git add _datafiles/world/dogmud/templates/help/
git commit -m "docs(loot): help for loot / party loot / party gold + build verification"
```

---

## Self-review notes (for the executor)

- **Corpse pointer mutation:** `corpse := &room.Corpses[idx]` mutates in place — do NOT take a value copy (`FindCorpse` returns a copy and would silently drop writes). Use `FindCorpseIndex`.
- **`CorpsesEnabled:false` path** (Task 2 Step 3b) must keep the old floor-drop, or that config disables loot entirely.
- **Owner set empty** = environment/mob kill → free-for-all immediately (Task 8 handles via `len(OwnerUserIds)==0`).
- **Round-robin cursor** lives on the `Party` (persists across corpses), not the corpse.
- **Gold conservation:** `SettleGold` must never lose or mint coins — the remainder-to-leader rule keeps the total exact (Task 12 test asserts it).
- **Mob corpses only:** every corpse guard added here is for mob corpses; player corpses (`UserId != 0`) are already skipped by salvage/raise and never receive a `Loot` container (only `dropMobLootAndSetCorpse` populates `Loot`).
