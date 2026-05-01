# Caravan System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Stage-1-party-driven caravan that runs Thornwall↔Stillwater on a 1-game-day cycle, replacing per-mob auto-restock for the served zones and producing a recurring 4052 bandit brawl.

**Architecture:** A new `internal/caravan` package owns route data, state-enum, and vendor-visit logic. A single new btree action `caravan_step` reads the leader's `MobState["caravan_state"]` and dispatches to the package. Player-attack rebuff via a new `Mob.PlayerAttackImmune` flag wired into the existing `IsNonCombatant` gates. Bandit↔caravan combat via a small extension to `lookfortrouble` that scans for mobs whose `groups` intersect the caller's `hates`.

**Tech Stack:** Go, YAML data files (mobs / behavior trees / dialogue / config), behavior tree primitives from Stage 1 (`internal/parties` + `internal/behaviortree`).

---

## Decisions locked at plan time (from spec + scout)

**Caravan crew** — mob IDs 316 (Ketil), 317 (Marta), 318 (Lars). All three live in `_datafiles/world/dogmud/mobs/thornwall_city/`. Zone field: `"Thornwall City"`.

**Depot rooms:**
- Thornwall depot = **room 465** (Market Square, Center). Caravan crew spawninfo lives here.
- Stillwater depot = **room 4109** (North Square). No spawninfo change — caravan only passes through.

**Cycle math** — `RoundSeconds: 4`, `RoundsPerDay: 900`. Default `CaravanDepotDwellRounds: 360` (~24 min real, ~half a game day at each depot). Cycle total ≈ 900 rounds.

**Stillwater vendors** (visit order picked for room-cluster locality, not mathematically optimal — pathfinding handles routing between):

| Order | Room | Mob | Name |
|---|---|---|---|
| 1 | 4102 | 336 | fishmonger Tov Brann |
| 2 | 4103 | 333 | innkeeper Sigrid |
| 3 | 4105 | 341 | storekeeper Wulf |
| 4 | 4106 | 337 | smith Brindle |
| 5 | 4125 | 338 | apothecary Ilsa |
| 6 | 4126 | 340 | pearl-carver Kess |
| 7 | 4135 | 348 | miller Bram |
| 8 | 4143 | 339 | weaver Edda |

**Thornwall vendors** (visit order):

| Order | Room | Mob | Name |
|---|---|---|---|
| 1 | 464 | 103 | food vendor |
| 2 | 470 | 97 | blacksmith Kerra |
| 3 | 471 | 98 | apothecary Voss |
| 4 | 475 | 104 | fence dealer Siv |
| 5 | 480 | 113 | weaver Maren |
| 6 | 481 | 248 | tavern cook Brynn |
| 7 | 482 | 108 | jeweler Tess |
| 8 | 483 | 109 | enchanter Vael |
| 9 | 507 | 273 | Whisper |

**Engine API anchors** (verified by scout):
- `mobs.Mob.Zone` (string) — `internal/mobs/mobs.go:67`
- `mobs.Mob.HasShop() bool` — `internal/mobs/mobs.go:722`
- `mobs.Mob.Hostile bool` — `internal/mobs/mobs.go:74`
- `mobs.Mob.Groups []string` — `internal/mobs/mobs.go:79`
- `mobs.Mob.Hates []string` — `internal/mobs/mobs.go:90`
- `mobs.Mob.HatesMob(m *Mob) bool` — `internal/mobs/mobs.go:854`
- `mobs.Mob.hatesAnyGroup(groups []string) bool` (private) — `internal/mobs/mobs.go:942`
- `mobs.TickMobShopRestock(mob *Mob) bool` — `internal/mobs/crafter.go:88`
- `shops.GetShopInventory(zone string, mobId int, roomId int) *ShopInventory` — `internal/mobs/crafter.go:93` (call site)
- `shops.ShopInventory.Restock() bool` — `internal/shops/shopinventory.go:71`
- `rooms.Room.GetMobs(findTypes ...FindFlag) []int` — `internal/rooms/rooms.go:1103`
- Non-combatant rebuff in `internal/usercommands/attack.go:161`
- `mob.Character.SetAggro(uid, mobInstId int, aggroType AggroType)` — `internal/characters/aggro.go:75` (used in Stage 1's `engageHostilePlayerInRoom`)

**Bandit detune targets:**
- 283 bandit_lookout: statpool 140 → 100 (~28% drop)
- 284 bandit_fighter: statpool TBR-25%
- 285 bandit_caster: statpool TBR-25%
- 286 Soren: statpool TBR-30%

(TBR = "to be read" — task 13 reads each YAML and computes the new value before writing.)

---

## File structure overview

| Layer | File | Purpose |
|---|---|---|
| Engine struct | `internal/mobs/mobs.go` | Add `PlayerAttackImmune bool` field (Task 1) |
| Engine config | `internal/configs/config.balance.go` | Add caravan knobs (Task 2) |
| Engine config | `_datafiles/config.yaml` | Default values for the new knobs (Task 2) |
| Engine integration | `internal/hooks/MobIdle_HandleIdleMobs.go` | Skip auto-restock in served zones (Task 3) |
| Engine integration | `internal/mobcommands/lookfortrouble.go` | Group-hate scan against other mobs (Task 4) |
| Player commands | 9 files in `internal/usercommands/` | Add `PlayerAttackImmune` rebuff (Task 1) |
| Caravan logic | `internal/caravan/routes.go` | Route data (Task 5) |
| Caravan logic | `internal/caravan/state.go` | State enum + transitions (Task 6) |
| Caravan logic | `internal/caravan/visit.go` | Vendor-visit invocation (Task 7) |
| Btree action | `internal/behaviortree/actions_caravan.go` | `caravan_step` action (Task 8) |
| Mob content | `_datafiles/world/dogmud/mobs/thornwall_city/{316,317,318}-*.yaml` | Caravan crew (Tasks 9-11) |
| Btree content | `_datafiles/world/dogmud/behaviors/thornwall_city/{316,317,318}-*.yaml` | Caravan btrees (Tasks 9-11) |
| Dialogue content | `_datafiles/world/dogmud/dialogue/thornwall_city/{316,317,318}-*.yaml` | Crew flavor (Tasks 9-11) |
| Room content | `_datafiles/world/dogmud/rooms/thornwall_city/465.yaml` | Add caravan to spawninfo (Task 12) |
| Mob tuning | `_datafiles/world/dogmud/mobs/north_road/{283,284,285,286}-*.yaml` | Bandit detune + `hates: caravan` (Task 13) |
| Docs | `docs/schemas/{mob,behavior}.md`, `PATCH_NOTES.md` | Document new fields/actions (Task 14) |

---

### Task 1: Add `PlayerAttackImmune` mob flag + wire 9 player commands

**Files:**
- Modify: `internal/mobs/mobs.go` — add field
- Modify: `internal/mobs/mobs_test.go` — test field is read from YAML
- Modify: `internal/usercommands/attack.go:161` — add rebuff check
- Modify: `internal/usercommands/bash.go:42`
- Modify: `internal/usercommands/grapple.go:41`
- Modify: `internal/usercommands/kick.go:43`
- Modify: `internal/usercommands/shoot.go:47`
- Modify: `internal/usercommands/taunt.go:43`
- Modify: `internal/usercommands/throw.go:115`
- Modify: `internal/usercommands/trip.go:42`
- Modify: `internal/usercommands/skill.skullduggery.steal.go:109`
- Test: `internal/usercommands/attack_test.go` (new test for rebuff path)

- [ ] **Step 1: Read the existing field group around `Hostile` and `NonCombatant` to find the YAML tag pattern.**

Run: read `internal/mobs/mobs.go:60-100`. Note the `yaml:"..."` struct tag on neighboring bool fields. The new field must use the exact same convention (lowercase, snake_case key).

- [ ] **Step 2: Add `PlayerAttackImmune` field to the Mob struct.**

In `internal/mobs/mobs.go`, alongside the other bool flags (near `NonCombatant`), add:

```go
PlayerAttackImmune bool `yaml:"player_attack_immune,omitempty"`
```

Match the exact tag style (omitempty etc.) of the neighboring fields you read in Step 1.

- [ ] **Step 3: Build to confirm no struct corruption.**

Run: `go build ./internal/mobs/`
Expected: no output (clean build).

- [ ] **Step 4: Write a failing test in `internal/usercommands/attack_test.go` for the rebuff path.**

Read `internal/usercommands/attack.go:155-180` to see the existing `IsNonCombatant()` rebuff check + the `mobs.FireAttackRejected` callout.

Add a test that seeds a mob with `PlayerAttackImmune: true` (and `NonCombatant: false`), invokes the attack handler against it, and asserts the rebuff message + that combat does not start. Pattern after any existing attack test that hits the non-combatant path. If no such test exists, write a tiny one — exercising at minimum that the function returns early without setting `mob.Character.Aggro`.

- [ ] **Step 5: Run the test to confirm failure.**

Run: `go test ./internal/usercommands/ -run TestAttack -v`
Expected: FAIL on the new test (PlayerAttackImmune flag has no effect yet).

- [ ] **Step 6: Wire the rebuff in attack.go.**

In `internal/usercommands/attack.go`, change:

```go
if m.IsNonCombatant() {
```

to:

```go
if m.IsNonCombatant() || m.PlayerAttackImmune {
```

- [ ] **Step 7: Wire the same change in the other 7 attack-style commands.**

Apply the identical `|| m.PlayerAttackImmune` extension at the cited line in each of:

- `bash.go:42`
- `grapple.go:41`
- `kick.go:43`
- `shoot.go:47`
- `taunt.go:43`
- `throw.go:115`
- `trip.go:42`

For each, read 5 lines of context first to confirm the variable name (`m`, `mob`, `target` — they vary), then apply the matching change.

- [ ] **Step 8: Wire the rebuff in `skill.skullduggery.steal.go:109`.**

Same pattern. The variable might be different — read context first.

- [ ] **Step 9: Run all affected tests.**

Run: `go test ./internal/usercommands/ ./internal/mobs/`
Expected: PASS (including the new test from Step 4).

- [ ] **Step 10: Commit.**

```bash
git add internal/mobs/mobs.go internal/usercommands/attack.go internal/usercommands/bash.go internal/usercommands/grapple.go internal/usercommands/kick.go internal/usercommands/shoot.go internal/usercommands/taunt.go internal/usercommands/throw.go internal/usercommands/trip.go internal/usercommands/skill.skullduggery.steal.go internal/usercommands/attack_test.go
git commit -m "feat(mobs,usercommands): add PlayerAttackImmune flag for caravan crew

New bool field on Mob, defaulted false. When true, the same gate that
rebuffs attacks on non-combatants in attack/bash/grapple/kick/shoot/
taunt/throw/trip and steal also rebuffs PlayerAttackImmune mobs. Used
by Stage 2 caravan crew, who fight bandits but cannot be attacked by
players. Mob-vs-mob attacks pass through unchanged.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: Add caravan config knobs

**Files:**
- Modify: `internal/configs/config.balance.go` — add 2 fields
- Modify: `_datafiles/config.yaml` — add defaults
- Test: `internal/configs/config.balance_test.go` (or append to existing)

- [ ] **Step 1: Read `internal/configs/config.balance.go` to find the existing string-slice / int patterns.**

Note the field declaration style and any `Validate()` defaults applied. New fields must follow identical conventions.

- [ ] **Step 2: Write a failing test asserting the defaults load.**

In `internal/configs/config.balance_test.go` (create if needed), add:

```go
func TestBalanceConfig_CaravanDefaults(t *testing.T) {
    cfg := &Balance{}
    cfg.Validate()
    if cfg.CaravanDepotDwellRounds != 360 {
        t.Errorf("CaravanDepotDwellRounds default = %d, want 360", cfg.CaravanDepotDwellRounds)
    }
    if len(cfg.CaravanServedZones) == 0 {
        t.Error("CaravanServedZones default should not be empty")
    }
    expected := map[string]bool{"Stillwater": true, "Thornwall City": true}
    for _, z := range cfg.CaravanServedZones {
        if !expected[z] {
            t.Errorf("unexpected zone in default CaravanServedZones: %q", z)
        }
        delete(expected, z)
    }
    if len(expected) > 0 {
        t.Errorf("missing default zones: %v", expected)
    }
}
```

- [ ] **Step 3: Run the test, confirm failure.**

Run: `go test ./internal/configs/ -run TestBalanceConfig_CaravanDefaults -v`
Expected: FAIL — fields don't exist yet.

- [ ] **Step 4: Add the two fields to the Balance struct in `internal/configs/config.balance.go`.**

```go
// CaravanServedZones lists zone display names whose vendor mobs do NOT
// auto-restock — they restock only on caravan visit. Mobs in zones not
// in this list keep the legacy per-mob restock tick.
CaravanServedZones []string `yaml:"CaravanServedZones"`

// CaravanDepotDwellRounds is the number of rounds the caravan rests at
// each depot between transit legs. ~360 ≈ 24 min real ≈ half a game day.
CaravanDepotDwellRounds int `yaml:"CaravanDepotDwellRounds"`
```

- [ ] **Step 5: Add defaults in the existing `Validate()` (or equivalent default-setting) function.**

Find the place in `config.balance.go` where existing fields get their defaults (e.g., `if e.RollSpread == 0 { e.RollSpread = 0.15 }`), and add:

```go
if len(e.CaravanServedZones) == 0 {
    e.CaravanServedZones = []string{"Stillwater", "Thornwall City"}
}
if e.CaravanDepotDwellRounds == 0 {
    e.CaravanDepotDwellRounds = 360
}
```

- [ ] **Step 6: Add a helper method for served-zone lookup.**

Append to `config.balance.go`:

```go
// IsCaravanServedZone reports whether the named zone is in the
// CaravanServedZones list. Case-sensitive — match the zone display
// name exactly.
func (e Balance) IsCaravanServedZone(zone string) bool {
    for _, z := range e.CaravanServedZones {
        if z == zone {
            return true
        }
    }
    return false
}
```

- [ ] **Step 7: Run the test, confirm pass.**

Run: `go test ./internal/configs/ -run TestBalanceConfig_CaravanDefaults -v`
Expected: PASS.

- [ ] **Step 8: Add YAML defaults in `_datafiles/config.yaml`.**

Find the `Balance:` section. Add (matching the existing indentation):

```yaml
  # Stage 2 caravan: zones whose vendors restock on caravan visit only
  # (per-mob TickMobShopRestock is suppressed). Vendors in zones not
  # listed here keep the legacy auto-restock tick.
  CaravanServedZones:
    - "Stillwater"
    - "Thornwall City"

  # Rounds the caravan rests at each depot between transit legs.
  # 360 rounds ≈ 24 min real ≈ half an in-game day.
  CaravanDepotDwellRounds: 360
```

- [ ] **Step 9: Boot test — confirm config loads.**

Run: `go build ./...` — expect clean.

- [ ] **Step 10: Commit.**

```bash
git add internal/configs/config.balance.go internal/configs/config.balance_test.go _datafiles/config.yaml
git commit -m "feat(config): add caravan config knobs

CaravanServedZones controls which zones the caravan delivers to —
vendors there skip per-mob auto-restock and depend on caravan visits.
CaravanDepotDwellRounds tunes how long the caravan rests at each
depot before the next leg. Defaults: Stillwater + Thornwall City,
360 rounds (~24 min real, ~half a game day).

IsCaravanServedZone() helper for the hook integration in the next task.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: Suppress auto-restock in caravan-served zones

**Files:**
- Modify: `internal/hooks/MobIdle_HandleIdleMobs.go:55` — guard the call
- Modify: `internal/hooks/hooks_test.go` — add a test for the suppression

- [ ] **Step 1: Read `internal/hooks/MobIdle_HandleIdleMobs.go:50-70` to find the existing `TickMobShopRestock` call.**

Note the surrounding context — the conditional that fires the restock plus the supply-cart flavor message.

- [ ] **Step 2: Write a failing test that asserts the restock is skipped for served zones.**

In `internal/hooks/hooks_test.go`, add:

```go
func TestHandleIdleMobs_SuppressesRestockInCaravanServedZones(t *testing.T) {
    cleanup := seedAllRegistries()
    defer cleanup()

    // Build a merchant mob in a caravan-served zone. The exact mob spec
    // matters less than the zone string + presence of a shop.
    mob := mobs.GetInstance(100)
    if mob == nil {
        t.Fatal("seedAllRegistries should provide instance 100")
    }
    mob.Zone = "Stillwater"
    // Force the mob into a state where TickMobShopRestock would normally
    // fire (set last-restock round to 0, register a shop). Use whatever
    // helpers seedAllRegistries exposes; if none, set the state manually
    // — see internal/mobs/crafter.go TickMobShopRestock for what it
    // checks.

    // Snapshot any shop-state field that Restock() would mutate. Run the
    // idle handler. Verify the snapshot is unchanged (restock did not
    // fire).
}
```

If the existing test scaffolding in `hooks_test.go` doesn't make this easy, write a minimal direct test against `IsCaravanServedZone` plus a docstring comment explaining the integration is tested manually in the smoke test (Task 15). Don't over-engineer the test; the suppression itself is one line of code.

- [ ] **Step 3: Run the test, confirm failure (or "no zone-check exists yet" if you went the lighter route).**

Run: `go test ./internal/hooks/ -run TestHandleIdleMobs_SuppressesRestockInCaravanServedZones -v`
Expected: FAIL.

- [ ] **Step 4: Wire the suppression in `internal/hooks/MobIdle_HandleIdleMobs.go`.**

Find the existing `mobs.TickMobShopRestock(mob)` call (around line 55). Wrap it:

```go
// Stage 2 caravan: vendors in caravan-served zones skip the per-mob
// restock tick — they restock only when the caravan visits.
if !configs.GetBalanceConfig().IsCaravanServedZone(mob.Zone) {
    if mobs.TickMobShopRestock(mob) {
        // ... existing flavor-message block stays inside the if ...
    }
}
```

Make sure the existing flavor-message body that was nested under the `if mobs.TickMobShopRestock(mob)` stays inside the inner `if`. Don't change its content.

- [ ] **Step 5: Run the test, confirm pass.**

Run: `go test ./internal/hooks/ -run TestHandleIdleMobs_SuppressesRestockInCaravanServedZones -v`
Expected: PASS.

- [ ] **Step 6: Run the full hooks test suite to confirm no regression.**

Run: `go test ./internal/hooks/`
Expected: ok.

- [ ] **Step 7: Commit.**

```bash
git add internal/hooks/MobIdle_HandleIdleMobs.go internal/hooks/hooks_test.go
git commit -m "feat(hooks): suppress per-mob auto-restock in caravan-served zones

When a vendor mob's zone is listed in Balance.CaravanServedZones, the
idle handler skips the per-mob TickMobShopRestock call. These vendors
will restock only when the caravan visits them (Task 7-8 wires the
visit invocation). Vendors in non-served zones keep the legacy tick.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: Extend `lookfortrouble` with hostile-mob group-hate scan

**Files:**
- Modify: `internal/mobcommands/lookfortrouble.go` — add the mob scan
- Modify: `internal/mobcommands/lookfortrouble_test.go` (or `mobcommands_test.go`) — test it

- [ ] **Step 1: Read `internal/mobcommands/lookfortrouble.go` end-to-end (it's about 200 lines).**

Note: the existing function builds three lists — `allPotentialTargets`, `nonDownedUserTargets`, `possibleMobTargets` — by scanning players. There's a "look for trouble in mobs they hate" branch starting around line 128 that scans `room.GetMobs()` and uses `HatesSpecies`. We'll extend that branch to also use `hatesAnyGroup` against other mobs' `Groups` slice.

- [ ] **Step 2: Read `internal/mobs/mobs.go:90, 854, 942` to confirm the helper signatures.**

Specifically:
- `Hates []string` — the group names this mob hates
- `func (m *Mob) HatesMob(other *Mob) bool` — public mob-vs-mob hostility check
- `func (r *Mob) hatesAnyGroup(groups []string) bool` — private helper

`HatesMob` likely already calls `hatesAnyGroup(other.Groups)`. Verify by reading. If `HatesMob` already does what we need, the integration in `lookfortrouble` is just "iterate other mobs in the room, append to `possibleMobTargets` when `mob.HatesMob(other)` is true."

- [ ] **Step 3: Write a failing test.**

In `internal/mobcommands/lookfortrouble_test.go` (create if absent), add:

```go
func TestLookForTrouble_TargetsMobByGroupHate(t *testing.T) {
    cleanup := seedAllRegistries() // uses hooks_test.go helper if available;
                                    // otherwise build minimal fake registries here
    defer cleanup()

    // Hostile mob with `hates: [caravan]`. No players in room.
    bandit := newTestMob(t, /* MobId */ 9001, /* Hostile */ true)
    bandit.Hates = []string{"caravan"}

    // Caravan mob with `groups: [caravan]`.
    caravan := newTestMob(t, 9002, false)
    caravan.Groups = []string{"caravan"}

    room := newTestRoom(t, 9999, []*mobs.Mob{bandit, caravan})

    // After lookfortrouble, the bandit should have Aggro on the caravan.
    LookForTrouble("", bandit, room)

    if bandit.Character.Aggro == nil {
        t.Fatal("bandit should have aggro after lookfortrouble; got nil")
    }
    if bandit.Character.Aggro.MobInstanceId != caravan.InstanceId {
        t.Errorf("bandit aggro target = inst %d, want %d (caravan)",
            bandit.Character.Aggro.MobInstanceId, caravan.InstanceId)
    }
}
```

The test helpers `newTestMob` and `newTestRoom` may not exist by those names — adapt to the existing scaffolding in `mobcommands_test.go` / `hooks_test.go`. If creating fakes is a pain, fall back to using `seedAllRegistries` and reading the test mob you get back.

- [ ] **Step 4: Run the test, confirm failure.**

Run: `go test ./internal/mobcommands/ -run TestLookForTrouble_TargetsMobByGroupHate -v`
Expected: FAIL — no group-hate scan exists.

- [ ] **Step 5: Add the hostile-mob scan in `lookfortrouble.go`.**

Find the existing "look for trouble in mobs they hate" block (around line 128). It already iterates `room.GetMobs()` and does `mob.HatesSpecies(...)` against species. We need to also check `mob.HatesMob(other)` (which internally does `hatesAnyGroup(other.Groups)`).

Sketch (apply at the right insertion point — read the surrounding code first):

```go
// Stage 2 caravan: extend hostile-mob scan to also check group hate
// (e.g., bandits with `hates: [caravan]` aggro on caravan mobs).
for _, otherInstId := range room.GetMobs() {
    if otherInstId == mob.InstanceId {
        continue
    }
    other := mobs.GetInstance(otherInstId)
    if other == nil {
        continue
    }
    if mob.HatesMob(other) {
        possibleMobTargets = append(possibleMobTargets, otherInstId)
    }
}
```

If the existing species-hate loop already iterates `room.GetMobs()`, fold the new `HatesMob` check into the same loop rather than duplicating iteration. Keep the change minimal.

- [ ] **Step 6: Run the test, confirm pass.**

Run: `go test ./internal/mobcommands/ -run TestLookForTrouble_TargetsMobByGroupHate -v`
Expected: PASS.

- [ ] **Step 7: Run the full mobcommands suite.**

Run: `go test ./internal/mobcommands/`
Expected: ok.

- [ ] **Step 8: Commit.**

```bash
git add internal/mobcommands/lookfortrouble.go internal/mobcommands/lookfortrouble_test.go
git commit -m "feat(mobcommands): lookfortrouble scans for hostile mobs by group hate

Extends the existing 'look for mobs I hate' branch to also check
mob.HatesMob(other), which uses hatesAnyGroup against the other mob's
Groups field. Lets bandits with 'hates: [caravan]' aggro on caravan
mobs entering their room.

Sets up the Stage 2 caravan-vs-bandit brawl at room 4052: caravan
mobs declare 'groups: [caravan]', bandits declare 'hates: [caravan]'.
Lookout's idle scan finds Ketil, sets aggro, calls help via Stage 1
party_call_help. Camp mobs respond. Caravan party (statted to win)
fights back via party_assist_target.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: Caravan route data (`internal/caravan/routes.go`)

**Files:**
- Create: `internal/caravan/routes.go`
- Create: `internal/caravan/routes_test.go`

- [ ] **Step 1: Write the failing test in `internal/caravan/routes_test.go`.**

```go
package caravan

import "testing"

func TestRoute_OutboundIntegrity(t *testing.T) {
    if OutboundRoute.DepartFromRoomId == 0 {
        t.Error("OutboundRoute.DepartFromRoomId == 0")
    }
    if OutboundRoute.ArriveAtRoomId == 0 {
        t.Error("OutboundRoute.ArriveAtRoomId == 0")
    }
    if OutboundRoute.DepartFromRoomId == OutboundRoute.ArriveAtRoomId {
        t.Errorf("OutboundRoute depart == arrive (%d)", OutboundRoute.DepartFromRoomId)
    }
    if len(OutboundRoute.VendorStopIds) == 0 {
        t.Error("OutboundRoute has no vendor stops")
    }
    seen := map[int]bool{}
    for _, stop := range OutboundRoute.VendorStopIds {
        if stop == 0 {
            t.Error("OutboundRoute has zero vendor stop")
        }
        if seen[stop] {
            t.Errorf("OutboundRoute has duplicate stop %d", stop)
        }
        seen[stop] = true
    }
}

func TestRoute_InboundIntegrity(t *testing.T) {
    if InboundRoute.DepartFromRoomId == 0 {
        t.Error("InboundRoute.DepartFromRoomId == 0")
    }
    if InboundRoute.ArriveAtRoomId == 0 {
        t.Error("InboundRoute.ArriveAtRoomId == 0")
    }
    if InboundRoute.DepartFromRoomId == InboundRoute.ArriveAtRoomId {
        t.Errorf("InboundRoute depart == arrive (%d)", InboundRoute.DepartFromRoomId)
    }
    if len(InboundRoute.VendorStopIds) == 0 {
        t.Error("InboundRoute has no vendor stops")
    }
}

func TestRoute_OutboundAndInboundOpposite(t *testing.T) {
    if OutboundRoute.ArriveAtRoomId != InboundRoute.DepartFromRoomId {
        t.Errorf("Outbound arrives at %d but Inbound departs from %d (should match)",
            OutboundRoute.ArriveAtRoomId, InboundRoute.DepartFromRoomId)
    }
    if InboundRoute.ArriveAtRoomId != OutboundRoute.DepartFromRoomId {
        t.Errorf("Inbound arrives at %d but Outbound departs from %d (should match)",
            InboundRoute.ArriveAtRoomId, OutboundRoute.DepartFromRoomId)
    }
}
```

- [ ] **Step 2: Run, confirm failure.**

Run: `go test ./internal/caravan/`
Expected: FAIL — package doesn't exist yet.

- [ ] **Step 3: Create `internal/caravan/routes.go`.**

```go
// Package caravan owns the route data, state machine, and vendor-visit
// logic for the Stage 2 NPC caravan that runs Thornwall ↔ Stillwater.
//
// See docs/superpowers/specs/2026-04-27-caravan-system-design.md for
// the full spec. Stage 1 NPC party primitives (internal/parties +
// internal/behaviortree party_* actions) drive the crew's coordinated
// movement and combat; this package owns only the route topology, the
// state-enum, and the per-stop restock invocation.
package caravan

// Depot rooms (decided in plan task 0).
const (
    ThornwallDepotRoomId  = 465  // Market Square, Center
    StillwaterDepotRoomId = 4109 // North Square
)

// Stillwater vendor rooms in visit order (cluster-local).
var stillwaterVendorRooms = []int{
    4102, // fishmonger Tov Brann
    4103, // innkeeper Sigrid
    4105, // storekeeper Wulf
    4106, // smith Brindle
    4125, // apothecary Ilsa
    4126, // pearl-carver Kess
    4135, // miller Bram
    4143, // weaver Edda
}

// Thornwall vendor rooms in visit order.
var thornwallVendorRooms = []int{
    464, // food vendor
    470, // blacksmith Kerra
    471, // apothecary Voss
    475, // fence dealer Siv
    480, // weaver Maren
    481, // tavern cook Brynn
    482, // jeweler Tess
    483, // enchanter Vael
    507, // Whisper
}

// Route is one leg of the caravan cycle.
type Route struct {
    // DepartFromRoomId is the depot the caravan starts this leg at.
    DepartFromRoomId int
    // ArriveAtRoomId is the depot the caravan reaches at the end of
    // transit (before vendor visits begin).
    ArriveAtRoomId int
    // VendorStopIds is the ordered list of rooms the caravan visits
    // after arriving. Each room's shop-bearing mobs get a Restock()
    // call when the caravan stops in that room.
    VendorStopIds []int
}

var (
    // OutboundRoute: Thornwall → Stillwater. Visited after arrival.
    OutboundRoute = Route{
        DepartFromRoomId: ThornwallDepotRoomId,
        ArriveAtRoomId:   StillwaterDepotRoomId,
        VendorStopIds:    stillwaterVendorRooms,
    }
    // InboundRoute: Stillwater → Thornwall. Visited after arrival.
    InboundRoute = Route{
        DepartFromRoomId: StillwaterDepotRoomId,
        ArriveAtRoomId:   ThornwallDepotRoomId,
        VendorStopIds:    thornwallVendorRooms,
    }
)
```

- [ ] **Step 4: Run, confirm pass.**

Run: `go test ./internal/caravan/`
Expected: ok.

- [ ] **Step 5: Commit.**

```bash
git add internal/caravan/routes.go internal/caravan/routes_test.go
git commit -m "feat(caravan): add route data for Thornwall<->Stillwater

Hardcoded routes in v1 (spec defers YAML config to Stage 4 when there
are multiple caravans). Two depots: Thornwall (room 465, Market Square
Center) and Stillwater (room 4109, North Square). 8 Stillwater vendor
stops + 9 Thornwall vendor stops, sequenced for room-cluster locality.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: Caravan state machine (`internal/caravan/state.go`)

**Files:**
- Create: `internal/caravan/state.go`
- Create: `internal/caravan/state_test.go`

- [ ] **Step 1: Write the failing test in `internal/caravan/state_test.go`.**

```go
package caravan

import "testing"

func TestCaravanState_NameRoundTrip(t *testing.T) {
    // Every state must have a unique name; ParseState(NameOf(s)) == s.
    for s := StateThornwallDwell; s <= StateThornwallRoute; s++ {
        name := s.Name()
        if name == "" {
            t.Errorf("state %d has empty name", s)
        }
        parsed, ok := ParseState(name)
        if !ok {
            t.Errorf("ParseState(%q) returned ok=false", name)
        }
        if parsed != s {
            t.Errorf("ParseState(%q) = %v, want %v", name, parsed, s)
        }
    }
}

func TestCaravanState_ParseUnknownReturnsFalse(t *testing.T) {
    if _, ok := ParseState("not_a_state"); ok {
        t.Error("ParseState(\"not_a_state\") should return ok=false")
    }
    if _, ok := ParseState(""); ok {
        t.Error("ParseState(\"\") should return ok=false")
    }
}

func TestCaravanState_AdvanceCycles(t *testing.T) {
    // Six states form a cycle that loops back to the first.
    expected := []CaravanState{
        StateOutboundTransit,
        StateStillwaterRoute,
        StateStillwaterDwell,
        StateInboundTransit,
        StateThornwallRoute,
        StateThornwallDwell, // wraps
    }
    cur := StateThornwallDwell
    for i, want := range expected {
        cur = AdvanceState(cur)
        if cur != want {
            t.Errorf("step %d: AdvanceState gave %v, want %v", i, cur, want)
        }
    }
}

func TestCaravanState_Classifiers(t *testing.T) {
    cases := []struct {
        s        CaravanState
        isDwell  bool
        isTransit bool
        isRoute  bool
    }{
        {StateThornwallDwell, true, false, false},
        {StateOutboundTransit, false, true, false},
        {StateStillwaterRoute, false, false, true},
        {StateStillwaterDwell, true, false, false},
        {StateInboundTransit, false, true, false},
        {StateThornwallRoute, false, false, true},
    }
    for _, c := range cases {
        if got := IsDwellState(c.s); got != c.isDwell {
            t.Errorf("IsDwellState(%v) = %v, want %v", c.s, got, c.isDwell)
        }
        if got := IsTransitState(c.s); got != c.isTransit {
            t.Errorf("IsTransitState(%v) = %v, want %v", c.s, got, c.isTransit)
        }
        if got := IsRouteState(c.s); got != c.isRoute {
            t.Errorf("IsRouteState(%v) = %v, want %v", c.s, got, c.isRoute)
        }
    }
}

func TestCaravanState_RouteFor(t *testing.T) {
    if RouteForState(StateOutboundTransit) != &OutboundRoute {
        t.Error("OutboundTransit should map to OutboundRoute")
    }
    if RouteForState(StateStillwaterRoute) != &OutboundRoute {
        t.Error("StillwaterRoute should map to OutboundRoute (continues outbound leg)")
    }
    if RouteForState(StateInboundTransit) != &InboundRoute {
        t.Error("InboundTransit should map to InboundRoute")
    }
    if RouteForState(StateThornwallRoute) != &InboundRoute {
        t.Error("ThornwallRoute should map to InboundRoute")
    }
    if RouteForState(StateThornwallDwell) != nil {
        t.Error("ThornwallDwell should map to no route (nil)")
    }
}
```

- [ ] **Step 2: Run, confirm failure.**

Run: `go test ./internal/caravan/ -run TestCaravanState`
Expected: FAIL.

- [ ] **Step 3: Create `internal/caravan/state.go`.**

```go
package caravan

// CaravanState enumerates the six phases of one caravan cycle.
//
// The cycle is:
//   ThornwallDwell → OutboundTransit → StillwaterRoute →
//   StillwaterDwell → InboundTransit → ThornwallRoute → (back to top)
//
// State transitions are driven by the caravan_step btree action reading
// environmental context (current room, dwell timer, route progress).
// AdvanceState is a pure function; the action decides WHEN to advance.
type CaravanState int

const (
    StateThornwallDwell  CaravanState = iota
    StateOutboundTransit
    StateStillwaterRoute
    StateStillwaterDwell
    StateInboundTransit
    StateThornwallRoute
)

var stateNames = map[CaravanState]string{
    StateThornwallDwell:  "thornwall_dwell",
    StateOutboundTransit: "outbound_transit",
    StateStillwaterRoute: "stillwater_route",
    StateStillwaterDwell: "stillwater_dwell",
    StateInboundTransit:  "inbound_transit",
    StateThornwallRoute:  "thornwall_route",
}

var nameToState = func() map[string]CaravanState {
    m := make(map[string]CaravanState, len(stateNames))
    for s, n := range stateNames {
        m[n] = s
    }
    return m
}()

// Name returns the canonical string for a state, used as the value
// stored in MobState["caravan_state"].
func (s CaravanState) Name() string {
    return stateNames[s]
}

// ParseState reverses Name(). Returns (StateThornwallDwell, false) on
// unknown input — callers should treat !ok as "no state set" and
// default to StateThornwallDwell.
func ParseState(name string) (CaravanState, bool) {
    s, ok := nameToState[name]
    return s, ok
}

// AdvanceState returns the next state in the cycle. After
// StateThornwallRoute it wraps back to StateThornwallDwell.
func AdvanceState(cur CaravanState) CaravanState {
    return (cur + 1) % 6
}

// IsDwellState reports whether the caravan is at a depot waiting for
// the dwell timer to expire.
func IsDwellState(s CaravanState) bool {
    return s == StateThornwallDwell || s == StateStillwaterDwell
}

// IsTransitState reports whether the caravan is in long-haul travel
// between depots.
func IsTransitState(s CaravanState) bool {
    return s == StateOutboundTransit || s == StateInboundTransit
}

// IsRouteState reports whether the caravan is visiting vendor stops
// in the destination town.
func IsRouteState(s CaravanState) bool {
    return s == StateStillwaterRoute || s == StateThornwallRoute
}

// RouteForState returns a pointer to the Route that owns this state's
// transit + visit, or nil for dwell states.
func RouteForState(s CaravanState) *Route {
    switch s {
    case StateOutboundTransit, StateStillwaterRoute:
        return &OutboundRoute
    case StateInboundTransit, StateThornwallRoute:
        return &InboundRoute
    }
    return nil
}
```

- [ ] **Step 4: Run, confirm pass.**

Run: `go test ./internal/caravan/`
Expected: ok.

- [ ] **Step 5: Commit.**

```bash
git add internal/caravan/state.go internal/caravan/state_test.go
git commit -m "feat(caravan): add state enum + transition helpers

Six-state cycle: thornwall_dwell → outbound_transit → stillwater_route
→ stillwater_dwell → inbound_transit → thornwall_route → loop.
AdvanceState is pure; the caravan_step btree action (next task) decides
when to advance based on env context (current room, dwell rounds,
route index).

Helpers: Name/ParseState for MobState round-trip, IsDwell/IsTransit/
IsRoute classifiers, RouteForState for getting the Route pointer
that owns a given state.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 7: Vendor visit logic (`internal/caravan/visit.go`)

**Files:**
- Create: `internal/caravan/visit.go`
- Create: `internal/caravan/visit_test.go`

- [ ] **Step 1: Write the failing test.**

```go
package caravan

import (
    "testing"

    "github.com/GoMudEngine/GoMud/internal/mobs"
    "github.com/GoMudEngine/GoMud/internal/rooms"
)

func TestVisitVendorsInRoom_NoRoomReturnsNil(t *testing.T) {
    // Use a roomid that's guaranteed not to exist.
    if got := VisitVendorsInRoom(99999999); got != nil {
        t.Errorf("VisitVendorsInRoom(missing) = %v, want nil", got)
    }
}

func TestVisitVendorsInRoom_NoShopMobsReturnsNil(t *testing.T) {
    // Seed a test room containing only non-shop mobs. The test
    // scaffolding pattern follows internal/behaviortree's seedTestRoom;
    // adapt as needed.
    cleanup := seedTestRoomWithMobs(t, 7777, "TestZone", []mobs.MobId{1})
    defer cleanup()

    got := VisitVendorsInRoom(7777)
    if got != nil {
        t.Errorf("VisitVendorsInRoom = %v, want nil for room with no shop mobs", got)
    }
}

func TestVisitVendorsInRoom_ReturnsShopMobNames(t *testing.T) {
    // Seed a room with one shop-bearing mob and confirm the name comes
    // back in the result. The Restock() side effect is verified manually
    // in the smoke test (Task 15) — wiring up a real shop inventory in
    // a unit test requires more scaffolding than is worth it here.
    mob := buildShopBearingTestMob(t, "TestShopper")
    cleanup := seedTestRoomWithExistingMobs(t, 7778, "TestZone", []*mobs.Mob{mob})
    defer cleanup()

    got := VisitVendorsInRoom(7778)
    if len(got) != 1 || got[0] != "TestShopper" {
        t.Errorf("VisitVendorsInRoom = %v, want [TestShopper]", got)
    }
}

// Helper sigs the implementer fills in. Pattern after Stage 1 test
// helpers in internal/behaviortree/actions_party_test.go (seedTestRoom,
// makePartyMob).
func seedTestRoomWithMobs(t *testing.T, roomId int, zone string, mobIds []mobs.MobId) func() {
    t.Helper()
    panic("TODO: implement following internal/behaviortree pattern")
}
func seedTestRoomWithExistingMobs(t *testing.T, roomId int, zone string, list []*mobs.Mob) func() {
    t.Helper()
    panic("TODO: implement following internal/behaviortree pattern")
}
func buildShopBearingTestMob(t *testing.T, name string) *mobs.Mob {
    t.Helper()
    panic("TODO: implement — make m.HasShop() return true")
}

// Silence unused imports if rooms isn't referenced by the helpers.
var _ = rooms.LoadRoom
```

Note for the implementer: those three helpers are placeholders — the implementer needs to flesh them out using the same patterns as the `internal/behaviortree/actions_party_test.go` scaffolding (`seedTestRoom`, `makePartyMob`, `mobs.SetInstanceForTest`). The third helper is the only tricky one: `HasShop()` returns `len(m.Character.Shop) > 0`, so seeding `m.Character.Shop = characters.Shop{...}` (or an equivalent stub) makes it return true. The implementer should consult `internal/characters/character.go` for the Shop struct shape if needed.

- [ ] **Step 2: Run, confirm failure (it'll panic on the TODO helpers, which counts).**

Run: `go test ./internal/caravan/ -run TestVisitVendorsInRoom`
Expected: PANIC or FAIL — the package and helpers don't exist.

- [ ] **Step 3: Create `internal/caravan/visit.go`.**

```go
package caravan

import (
    "fmt"

    "github.com/GoMudEngine/GoMud/internal/mobs"
    "github.com/GoMudEngine/GoMud/internal/rooms"
    "github.com/GoMudEngine/GoMud/internal/shops"
)

// VisitVendorsInRoom calls Restock() on every shop-bearing mob in the
// given room and returns the list of mob names that received a delivery
// (for room-flavor message generation).
//
// "Shop-bearing" = HasShop() returns true AND the shops package can
// resolve a shop inventory for (zone, mobId, homeRoomId). A mob with a
// Shop spec but no registered inventory is silently skipped.
//
// Returns nil if the room doesn't exist.
func VisitVendorsInRoom(roomId int) []string {
    room := rooms.LoadRoom(roomId)
    if room == nil {
        return nil
    }
    var visited []string
    for _, instId := range room.GetMobs(rooms.FindAll) {
        mob := mobs.GetInstance(instId)
        if mob == nil || !mob.HasShop() {
            continue
        }
        si := shops.GetShopInventory(mob.Zone, int(mob.MobId), mob.HomeRoomId)
        if si == nil {
            continue
        }
        si.Restock()
        visited = append(visited, mob.Character.Name)
    }
    return visited
}

// FormatDeliveryMessage builds the room-visible flavor text for a
// caravan visit. Returns an empty string if no vendors were visited
// (caller should skip sending in that case).
func FormatDeliveryMessage(visited []string) string {
    if len(visited) == 0 {
        return ""
    }
    if len(visited) == 1 {
        return fmt.Sprintf(
            `<ansi fg="yellow">The caravan crew unloads a crate of supplies; <ansi fg="mobname">%s</ansi> nods their thanks.</ansi>`,
            visited[0])
    }
    // Two or more — list with commas.
    names := visited[0]
    for _, n := range visited[1:] {
        names += ", " + n
    }
    return fmt.Sprintf(
        `<ansi fg="yellow">The caravan crew unloads supplies for: %s.</ansi>`,
        names)
}
```

Note: `Restock()` returns a bool indicating whether anything was added, but for the visit-list we report any shop visit regardless — players see "delivery happened" even if a vendor was already topped off. (Top-off-to-MaxStock means the visit is a no-op for that vendor, but the visible flavor stays consistent.)

- [ ] **Step 4: Implement the test helpers (the three TODO functions in `visit_test.go`).**

Read `internal/behaviortree/actions_party_test.go:30-60` (the `makePartyMob` and `seedTestRoom` helpers from Stage 1). Adapt them:

- `seedTestRoomWithMobs(t, roomId, zone, mobIds)`: creates a fresh `rooms.Room` at roomId in the given zone, instantiates one Mob per mobId via `mobs.GetMobSpec`/`NewMobByIdFresh`, places them in the room. Returns a cleanup that destroys the room and mob instances.
- `seedTestRoomWithExistingMobs(t, roomId, zone, list)`: same but takes already-built `*mobs.Mob` values (used for the shop-bearing one).
- `buildShopBearingTestMob(t, name)`: builds a `*mobs.Mob` with `Character.Name = name`, `Character.Shop = characters.Shop{...}` (one stock entry), `Zone = "TestZone"`, and a unique `InstanceId`. Register via `mobs.SetInstanceForTest`. The `Shop` value just needs to be non-empty so `HasShop()` returns true.

If `shops.GetShopInventory` returns nil for the test mob (because no inventory was registered through the normal load path), that's fine — Step 3's `VisitVendorsInRoom` skips silently and the test should be adjusted to reflect that. In that case, change the third test's assertion to: "returns nil because no registered shop inventory" and remove the test assertion on names. The integration test then is just "iteration logic works without panicking." The full Restock side effect is verified in the smoke test.

- [ ] **Step 5: Run, confirm pass.**

Run: `go test ./internal/caravan/`
Expected: ok.

- [ ] **Step 6: Commit.**

```bash
git add internal/caravan/visit.go internal/caravan/visit_test.go
git commit -m "feat(caravan): add VisitVendorsInRoom + delivery message formatter

VisitVendorsInRoom iterates the room's mobs and calls Restock() on
each shop-bearing one, returning the list of names for flavor-message
formatting. Looks up shop inventory via shops.GetShopInventory(zone,
mobId, homeRoomId) — the existing per-mob restock path's pattern.

FormatDeliveryMessage builds the visible room text for the caravan's
arrival at a vendor stop. Single-vendor rooms get a personalized line;
multi-vendor rooms list comma-separated.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 8: `caravan_step` btree action (`internal/behaviortree/actions_caravan.go`)

**Files:**
- Create: `internal/behaviortree/actions_caravan.go`
- Create: `internal/behaviortree/actions_caravan_test.go`

- [ ] **Step 1: Read Stage 1 reference action and state-machine state-key conventions.**

Read `internal/behaviortree/actions_party.go` (the Stage 1 actions) for:
- How actions look up the calling mob (`mobs.GetInstance(ctx.InstanceId)`)
- How `ctx.MobState.Set(key, value)` and `ctx.MobState.Get(key)` work
- The action signature: `func(params map[string]any, ctx *EvalContext) Result`

Read `internal/behaviortree/types.go` for `EvalContext` and `BehaviorState`.

- [ ] **Step 2: Write the failing test.**

```go
package behaviortree

import (
    "fmt"
    "testing"

    "github.com/GoMudEngine/GoMud/internal/caravan"
    "github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestActCaravanStep_DefaultsToThornwallDwellOnFirstTick(t *testing.T) {
    // A mob with no caravan_state in MobState should be treated as
    // freshly spawned at the Thornwall depot.
    fn := LookupAction("caravan_step")
    if fn == nil {
        t.Fatal("caravan_step not registered")
    }

    mob := buildCaravanLeaderMob(t, /*instanceId*/ 7100, /*roomId*/ caravan.ThornwallDepotRoomId)
    state := NewBehaviorState()

    ctx := &EvalContext{
        InstanceId: 7100,
        RoomId:     caravan.ThornwallDepotRoomId,
        MobState:   state,
    }
    result := fn(nil, ctx)
    if result == Failure {
        t.Errorf("caravan_step returned Failure on first tick; want Success")
    }
    got, _ := state.Get("caravan_state")
    if got != caravan.StateThornwallDwell.Name() {
        t.Errorf("caravan_state after first tick = %q, want %q",
            got, caravan.StateThornwallDwell.Name())
    }
    _ = mob
}

func TestActCaravanStep_AdvancesFromDwellAfterTimerExpires(t *testing.T) {
    fn := LookupAction("caravan_step")

    mob := buildCaravanLeaderMob(t, 7101, caravan.ThornwallDepotRoomId)
    _ = mob
    state := NewBehaviorState()
    state.Set("caravan_state", caravan.StateThornwallDwell.Name())
    // Mark dwell as having started long enough ago that it expires now.
    // Use a sentinel "0" to mean "expired" — adapt to whatever convention
    // the implementation picks.
    state.Set("caravan_state_started_round", "0")

    ctx := &EvalContext{
        InstanceId: 7101,
        RoomId:     caravan.ThornwallDepotRoomId,
        MobState:   state,
    }
    fn(nil, ctx)

    got, _ := state.Get("caravan_state")
    if got != caravan.StateOutboundTransit.Name() {
        t.Errorf("after dwell timer expires, caravan_state = %q, want %q",
            got, caravan.StateOutboundTransit.Name())
    }
}

func TestActCaravanStep_AdvancesFromTransitOnArrival(t *testing.T) {
    fn := LookupAction("caravan_step")

    // Caravan is "in transit outbound" but has reached Stillwater.
    mob := buildCaravanLeaderMob(t, 7102, caravan.StillwaterDepotRoomId)
    _ = mob
    state := NewBehaviorState()
    state.Set("caravan_state", caravan.StateOutboundTransit.Name())

    ctx := &EvalContext{
        InstanceId: 7102,
        RoomId:     caravan.StillwaterDepotRoomId,
        MobState:   state,
    }
    fn(nil, ctx)

    got, _ := state.Get("caravan_state")
    if got != caravan.StateStillwaterRoute.Name() {
        t.Errorf("after transit arrival, caravan_state = %q, want %q",
            got, caravan.StateStillwaterRoute.Name())
    }
}

func TestActCaravanStep_RouteAdvancesIndexAndExitsAfterAllStops(t *testing.T) {
    fn := LookupAction("caravan_step")

    mob := buildCaravanLeaderMob(t, 7103, 4143) // last vendor stop
    _ = mob
    state := NewBehaviorState()
    state.Set("caravan_state", caravan.StateStillwaterRoute.Name())
    // Pretend we've already visited 7 of the 8 stops and are now at the
    // 8th (room 4143). After this tick, route should exit to dwell.
    state.Set("caravan_route_index", fmt.Sprintf("%d", len(caravan.OutboundRoute.VendorStopIds)-1))

    ctx := &EvalContext{
        InstanceId: 7103,
        RoomId:     4143,
        MobState:   state,
    }
    fn(nil, ctx)

    got, _ := state.Get("caravan_state")
    if got != caravan.StateStillwaterDwell.Name() {
        t.Errorf("after final route stop, caravan_state = %q, want %q",
            got, caravan.StateStillwaterDwell.Name())
    }
}

// buildCaravanLeaderMob is the local test scaffolding helper. Pattern
// after makePartyMob in actions_party_test.go.
func buildCaravanLeaderMob(t *testing.T, instanceId int, roomId int) *mobs.Mob {
    t.Helper()
    panic("TODO: implement following actions_party_test.go makePartyMob pattern")
}
```

- [ ] **Step 3: Run, confirm failure.**

Run: `go test ./internal/behaviortree/ -run TestActCaravanStep`
Expected: FAIL or PANIC — action and helper don't exist.

- [ ] **Step 4: Create `internal/behaviortree/actions_caravan.go`.**

```go
package behaviortree

// actions_caravan.go — Stage 2 caravan state-machine btree action.
//
// caravan_step is the single workhorse action that drives the
// continuous Thornwall↔Stillwater caravan cycle. It reads the leader's
// caravan_state from MobState, dispatches to internal/caravan, and
// advances the state when transitions are warranted.
//
// State persistence in MobState (string→string):
//   caravan_state              — current state name (caravan.CaravanState.Name())
//   caravan_state_started_round — round when current state was entered (uint64)
//   caravan_route_index        — index into the current Route's VendorStopIds

import (
    "fmt"
    "strconv"

    "github.com/GoMudEngine/GoMud/internal/caravan"
    "github.com/GoMudEngine/GoMud/internal/configs"
    "github.com/GoMudEngine/GoMud/internal/mobs"
    "github.com/GoMudEngine/GoMud/internal/rooms"
    "github.com/GoMudEngine/GoMud/internal/util"
)

func init() {
    actionRegistry["caravan_step"] = actCaravanStep
}

func actCaravanStep(params map[string]any, ctx *EvalContext) Result {
    if ctx.MobState == nil {
        return Failure
    }
    mob := mobs.GetInstance(ctx.InstanceId)
    if mob == nil {
        return Failure
    }

    cur := readCaravanState(ctx.MobState)

    // Dispatch by category.
    switch {
    case caravan.IsDwellState(cur):
        return tickDwell(cur, mob, ctx)
    case caravan.IsTransitState(cur):
        return tickTransit(cur, mob, ctx)
    case caravan.IsRouteState(cur):
        return tickRoute(cur, mob, ctx)
    }
    return Failure
}

// readCaravanState fetches the current state from MobState, defaulting
// to StateThornwallDwell on first tick (no value set).
func readCaravanState(s *BehaviorState) caravan.CaravanState {
    raw, ok := s.Get("caravan_state")
    if !ok {
        s.Set("caravan_state", caravan.StateThornwallDwell.Name())
        s.Set("caravan_state_started_round", strconv.FormatUint(util.GetRoundCount(), 10))
        s.Set("caravan_route_index", "0")
        return caravan.StateThornwallDwell
    }
    parsed, ok := caravan.ParseState(raw)
    if !ok {
        // Corrupt value — reset.
        s.Set("caravan_state", caravan.StateThornwallDwell.Name())
        return caravan.StateThornwallDwell
    }
    return parsed
}

// transitionTo writes the new state to MobState and resets per-state
// counters (started-round, route index).
func transitionTo(s *BehaviorState, next caravan.CaravanState) {
    s.Set("caravan_state", next.Name())
    s.Set("caravan_state_started_round", strconv.FormatUint(util.GetRoundCount(), 10))
    s.Set("caravan_route_index", "0")
}

// tickDwell: at depot, waiting for the dwell timer to elapse.
func tickDwell(cur caravan.CaravanState, mob *mobs.Mob, ctx *EvalContext) Result {
    startedStr, _ := ctx.MobState.Get("caravan_state_started_round")
    started, _ := strconv.ParseUint(startedStr, 10, 64)
    dwell := uint64(configs.GetBalanceConfig().CaravanDepotDwellRounds)
    if util.GetRoundCount() >= started+dwell {
        transitionTo(ctx.MobState, caravan.AdvanceState(cur))
        return Success
    }
    // Still resting — no-op success so the btree branch consumes mob_idle.
    return Success
}

// tickTransit: walking toward the destination depot. Issues pathto on
// each tick; the engine's path step + mob walk loop handle progress.
// On arrival, transition to the route phase.
func tickTransit(cur caravan.CaravanState, mob *mobs.Mob, ctx *EvalContext) Result {
    route := caravan.RouteForState(cur)
    if route == nil {
        return Failure
    }
    if ctx.RoomId == route.ArriveAtRoomId {
        transitionTo(ctx.MobState, caravan.AdvanceState(cur))
        return Success
    }
    // Issue a pathto each tick — pathto is idempotent if already on the
    // right path.
    mob.Command(fmt.Sprintf("pathto %d", route.ArriveAtRoomId))
    return Success
}

// tickRoute: visiting vendor stops in order. On arrival at the next
// stop, fire VisitVendorsInRoom + emit flavor + advance the index. When
// all stops done, transition to the depot dwell state.
func tickRoute(cur caravan.CaravanState, mob *mobs.Mob, ctx *EvalContext) Result {
    route := caravan.RouteForState(cur)
    if route == nil {
        return Failure
    }
    idxStr, _ := ctx.MobState.Get("caravan_route_index")
    idx, _ := strconv.Atoi(idxStr)
    if idx >= len(route.VendorStopIds) {
        // All stops visited — exit the route phase.
        transitionTo(ctx.MobState, caravan.AdvanceState(cur))
        return Success
    }
    nextRoom := route.VendorStopIds[idx]
    if ctx.RoomId == nextRoom {
        // Arrived at this stop — restock + advance index.
        visited := caravan.VisitVendorsInRoom(nextRoom)
        if msg := caravan.FormatDeliveryMessage(visited); msg != "" {
            if r := rooms.LoadRoom(nextRoom); r != nil {
                r.SendText(msg)
            }
        }
        ctx.MobState.Set("caravan_route_index", strconv.Itoa(idx+1))
        return Success
    }
    // Not at the next stop yet — pathto it.
    mob.Command(fmt.Sprintf("pathto %d", nextRoom))
    return Success
}
```

- [ ] **Step 5: Implement `buildCaravanLeaderMob` in the test file.**

Pattern after `makePartyMob` in `internal/behaviortree/actions_party_test.go:30-60`. Build a `*mobs.Mob` with the given InstanceId, set RoomId to the given roomId, name "Ketil", register via `mobs.SetInstanceForTest`. Return the mob; the test caller registers cleanup via `t.Cleanup(func() { mobs.SetInstanceForTest(instanceId, nil) })`.

- [ ] **Step 6: Run the tests, confirm pass.**

Run: `go test ./internal/behaviortree/ -run TestActCaravanStep -v`
Expected: PASS.

- [ ] **Step 7: Run the full behaviortree suite to confirm no regression.**

Run: `go test ./internal/behaviortree/`
Expected: ok.

- [ ] **Step 8: Commit.**

```bash
git add internal/behaviortree/actions_caravan.go internal/behaviortree/actions_caravan_test.go
git commit -m "feat(behaviortree): add caravan_step btree action

Single workhorse action that drives the Stage 2 caravan state machine.
Reads caravan_state from MobState (defaults to thornwall_dwell on first
tick), dispatches to internal/caravan helpers, advances state on the
right environmental conditions:

- Dwell states: advance when CaravanDepotDwellRounds elapsed since
  state was entered.
- Transit states: pathto destination depot each tick; transition to
  route phase on arrival.
- Route states: pathto next vendor stop; on arrival, fire
  VisitVendorsInRoom + room flavor message + advance route index.
  When all stops visited, transition to depot dwell.

State persistence uses MobState string keys: caravan_state,
caravan_state_started_round, caravan_route_index.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 9: Caravan crew — Ketil (mob 316, leader)

**Files:**
- Create: `_datafiles/world/dogmud/mobs/thornwall_city/316-ketil.yaml`
- Create: `_datafiles/world/dogmud/behaviors/thornwall_city/316-ketil.yaml`
- Create: `_datafiles/world/dogmud/dialogue/thornwall_city/316-ketil.yaml`

**Note on naming:** filename and `name:` field MUST agree per `ConvertForFilename`. "Ketil" → "ketil" → `316-ketil.yaml`. Same name in mob, behavior, and dialogue file.

- [ ] **Step 1: Read an existing thornwall_city mob YAML for the file structure pattern.**

Run: `Read _datafiles/world/dogmud/mobs/thornwall_city/97-blacksmith_kerra.yaml` (or any vendor in the folder). Note the YAML structure — required fields, archetype, statpool, equipment, character block.

- [ ] **Step 2: Read the schema reference for any unfamiliar fields.**

Run: `Read docs/schemas/mob.md`. Confirm field names for `groups`, `non_combatant`, `player_attack_immune` (the new flag).

- [ ] **Step 3: Create `_datafiles/world/dogmud/mobs/thornwall_city/316-ketil.yaml`.**

```yaml
mobid: 316
zone: Thornwall City
archetype: fighting
behavior_archetype: ""           # per-mob btree handles all behavior
statpool: 220                     # leader, statted to anchor a 3v4 win
itemdropchance: 0
groups:
  - caravan
  - merchant_train
hostile: false
non_combatant: false              # crew fights bandits
player_attack_immune: true        # players cannot attack — rebuff like shopkeepers
maxwander: -1                     # -1 = unlimited; caravan crosses zones
charm_immune: true
buffids: []
idlecommands:
  - 'emote checks the harness on the lead horse, frowning at a worn buckle'
  - ''
  - 'emote runs a calloused hand along the wagon rail, listening for cracks'
  - ''
  - 'emote spits in the dust and squints toward the road'
character:
  name: Ketil
  description: |
    A weathered man in his middle years, broad through the shoulders
    and slow to anger but not slow to act. He wears a heavy traveling
    coat of waxed canvas over a coat of mail mended in three places.
    A long sword hangs at his hip and a kite shield rides on his back.
    His eyes track movement on the road with the practiced wariness of
    someone who has run this route more times than he can count.
  speciesid: 1
  level: 1
  gold: 50
  equipment:
    weapon:
      itemid: 50
    offhand:
      itemid: 51
    body:
      itemid: 60
  stats:
    strength:
      training: 30
    dexterity:
      training: 18
    vitality:
      training: 25
    perception:
      training: 15
    willpower:
      training: 10
    charisma:
      training: 12
  skills:
    weapon-combat: 25
    block: 15
```

The exact item IDs (`weapon: 50`, `offhand: 51`, `body: 60`) are placeholders. Pick real items from `_datafiles/world/dogmud/items/` — a longsword, a kite shield, mail body armor. Use `Glob _datafiles/world/dogmud/items/weapons-*/*.yaml` to find candidates, or just leave the equipment block unset for v1 if hand-picking real IDs becomes a rabbit hole. The mob will fight with bare hands using statpool alone, which still wins given a high statpool.

If hand-picking is deferred, REPLACE the equipment block with a comment:

```yaml
# Equipment TBD — defaulting to fists for v1 stat-pool tuning. If
# fights at 4052 don't go the caravan's way, kit Ketil with a real
# weapon/shield/armor from items/.
```

- [ ] **Step 4: Create `_datafiles/world/dogmud/behaviors/thornwall_city/316-ketil.yaml`.**

```yaml
# Caravan master Ketil (316) — Stage 2 caravan leader.
# Drives the state machine via caravan_step. Engages bandits via
# Stage 1 party_assist_target when in combat. Forms the caravan party
# at first idle tick.

tree:
  type: sequence
  children:

    # ── Always: ensure caravan party (idempotent) ─────────────────────
    - type: action
      do: party_ensure_npc_party
      leader_mob_id: 316
      home_room_id: 465      # Thornwall depot

    - type: selector
      children:

        # ── COMBAT: copy the bandit's aggro back at them so we hit
        #    whoever's hitting us. (party_assist_target sets caller's
        #    aggro to leader's; for the leader, leader == self, so the
        #    aggro is whatever was set by the mob_hurt response.)
        - type: sequence
          event: mob_idle
          children:
            - type: condition
              check: party_in_combat
            - type: action
              do: party_assist_target

        # ── DRIVE THE CARAVAN STATE MACHINE
        - type: sequence
          event: mob_idle
          children:
            - type: action
              do: caravan_step

        # ── On hit: counter-attack (sets Aggro on the attacker so the
        #    combat loop engages them; followers will then assist via
        #    party_assist_target on next idle).
        - type: sequence
          event: mob_hurt
          children:
            - type: action
              do: attack
```

- [ ] **Step 5: Create `_datafiles/world/dogmud/dialogue/thornwall_city/316-ketil.yaml`.**

Read an existing simple dialogue YAML for the pattern: e.g.,
`Read _datafiles/world/dogmud/dialogue/thornwall_city/103-food_vendor.yaml`. Then write Ketil's:

```yaml
mobid: 316

patterns:
  - keywords: [hello, hi, greetings, hail]
    text: |
      <ansi fg="mobname">Ketil</ansi> nods at you without breaking
      stride. "Aye. Just got the wagons checked over. Long road today."

  - keywords: [road, trip, journey, where, going]
    text: |
      <ansi fg="mobname">Ketil</ansi> jerks his chin toward the gate.
      "North to Stillwater, then back. Same as always. The bandits on
      the spur road test us every time, and every time they lose.
      Keeps the road honest."

  - keywords: [bandit, bandits, danger, robber]
    text: |
      <ansi fg="mobname">Ketil</ansi> rests a hand on the pommel of his
      sword. "Fools who think a wagon means easy coin. They learn
      otherwise. Marta and Lars earn their wages on every run."

  - keywords: [wagon, supplies, cargo, goods, deliver]
    text: |
      "Bulk goods. Things that move better in volume than a
      shopkeeper can stock alone. Stillwater needs Thornwall iron;
      Thornwall needs Stillwater pearl. We bridge the difference."

  - keywords: [marta, lars, crew, guards]
    text: |
      "<ansi fg="mobname">Marta</ansi> hits like a falling tree.
      <ansi fg="mobname">Lars</ansi> can pin a sparrow at fifty paces.
      I trust them with my life and have, more than once."

defaults:
  - text: |
      <ansi fg="mobname">Ketil</ansi> grunts noncommittally and
      returns his attention to the wagon.
```

- [ ] **Step 6: Sanity-build to confirm YAML loads.**

Run: `go build ./...` and (if available locally) start the server briefly to confirm `mobs.LoadDataFiles() loadedCount=...` doesn't panic on Ketil.

If a local boot is impractical at this stage, defer the boot check to Task 12 when all three crew are wired in.

- [ ] **Step 7: Commit.**

```bash
git add _datafiles/world/dogmud/mobs/thornwall_city/316-ketil.yaml _datafiles/world/dogmud/behaviors/thornwall_city/316-ketil.yaml _datafiles/world/dogmud/dialogue/thornwall_city/316-ketil.yaml
git commit -m "content(thornwall_city): add Ketil, caravan master (mob 316)

Stage 2 caravan crew: leader. statpool 220 (anchors a 3v4 win against
the bandit pack), groups [caravan, merchant_train], player_attack_immune
true, maxwander -1 (caravan crosses zones).

Behavior tree:
- party_ensure_npc_party (forms caravan party with home_room_id 465)
- in combat: party_assist_target
- mob_idle out of combat: caravan_step (state machine)
- mob_hurt: attack (sets Aggro on attacker so followers can assist)

Dialogue covers route, bandits, cargo, crew. Establishes the
wholesaler-arbitrage worldbuilding.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 10: Caravan crew — Marta (mob 317, guard)

**Files:**
- Create: `_datafiles/world/dogmud/mobs/thornwall_city/317-marta.yaml`
- Create: `_datafiles/world/dogmud/behaviors/thornwall_city/317-marta.yaml`
- Create: `_datafiles/world/dogmud/dialogue/thornwall_city/317-marta.yaml`

- [ ] **Step 1: Create `_datafiles/world/dogmud/mobs/thornwall_city/317-marta.yaml`.**

```yaml
mobid: 317
zone: Thornwall City
archetype: fighting
behavior_archetype: ""
statpool: 180
itemdropchance: 0
groups:
  - caravan
  - merchant_train
hostile: false
non_combatant: false
player_attack_immune: true
maxwander: -1
charm_immune: true
buffids: []
idlecommands:
  - 'emote rolls her shoulders, working out road-stiffness'
  - ''
  - 'emote leans on her hammer and watches the road northward'
  - ''
  - 'emote unwraps a strip of dried meat and chews thoughtfully'
character:
  name: Marta
  description: |
    A heavyset woman with arms like cooper's bands, dressed in a
    quilted gambeson and a sturdy leather skirt. A two-handed war
    hammer rests across her back, the head dark with old use. Her
    expression is mild, almost amused — the look of someone who has
    been underestimated her whole life and stopped caring.
  speciesid: 1
  level: 1
  gold: 15
  equipment:
    weapon:
      itemid: 50
    body:
      itemid: 60
  stats:
    strength:
      training: 32
    dexterity:
      training: 12
    vitality:
      training: 28
    perception:
      training: 12
    willpower:
      training: 10
    charisma:
      training: 8
  skills:
    weapon-combat: 22
    unarmed-combat: 18
```

Same equipment-deferral note as Task 9: pick a real two-handed hammer from `items/weapons-*/` if convenient, otherwise leave equipment blank and rely on statpool.

- [ ] **Step 2: Create `_datafiles/world/dogmud/behaviors/thornwall_city/317-marta.yaml`.**

```yaml
# Caravan guard Marta (317) — follower of Ketil.
# Follows the leader through transit, assists on target in combat,
# counter-attacks on hurt.

tree:
  type: sequence
  children:

    # ── Always: ensure caravan party (idempotent) ─────────────────────
    - type: action
      do: party_ensure_npc_party
      leader_mob_id: 316
      home_room_id: 465

    - type: selector
      children:

        # ── COMBAT: assist whoever the leader is targeting
        - type: sequence
          event: mob_idle
          children:
            - type: condition
              check: party_in_combat
            - type: action
              do: party_assist_target

        # ── TRANSIT: follow the leader
        - type: sequence
          event: mob_idle
          children:
            - type: action
              do: party_follow_leader

        # ── On hit: counter-attack
        - type: sequence
          event: mob_hurt
          children:
            - type: action
              do: attack
```

- [ ] **Step 3: Create `_datafiles/world/dogmud/dialogue/thornwall_city/317-marta.yaml`.**

```yaml
mobid: 317

patterns:
  - keywords: [hello, hi, greetings, hail]
    text: |
      <ansi fg="mobname">Marta</ansi> gives you a slow, friendly nod.
      "Hey."

  - keywords: [hammer, weapon, fight]
    text: |
      "Old girl's seen things." She pats the haft. "Most folk who
      look at her think twice. The ones who don't think at all
      stop thinking soon enough."

  - keywords: [bandit, bandits, road]
    text: |
      <ansi fg="mobname">Marta</ansi> shrugs. "Same crew, every run.
      They never seem to learn. Suits me. Keeps me in practice."

  - keywords: [ketil, lars, crew]
    text: |
      "Boss is steady. Lars hits what he aims at. Wouldn't run with
      anyone else."

defaults:
  - text: |
      <ansi fg="mobname">Marta</ansi> gives you a small smile and
      goes back to whatever she was doing.
```

- [ ] **Step 4: Commit.**

```bash
git add _datafiles/world/dogmud/mobs/thornwall_city/317-marta.yaml _datafiles/world/dogmud/behaviors/thornwall_city/317-marta.yaml _datafiles/world/dogmud/dialogue/thornwall_city/317-marta.yaml
git commit -m "content(thornwall_city): add Marta, caravan guard (mob 317)

Stage 2 caravan crew: guard. statpool 180, two-handed hammer.
groups [caravan, merchant_train], player_attack_immune true,
maxwander -1.

Behavior tree: ensure caravan party (member of leader 316), assist
target in combat, follow leader otherwise, counter-attack on hurt.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 11: Caravan crew — Lars (mob 318, guard)

**Files:**
- Create: `_datafiles/world/dogmud/mobs/thornwall_city/318-lars.yaml`
- Create: `_datafiles/world/dogmud/behaviors/thornwall_city/318-lars.yaml`
- Create: `_datafiles/world/dogmud/dialogue/thornwall_city/318-lars.yaml`

- [ ] **Step 1: Create `_datafiles/world/dogmud/mobs/thornwall_city/318-lars.yaml`.**

```yaml
mobid: 318
zone: Thornwall City
archetype: fighting
behavior_archetype: ""
statpool: 170
itemdropchance: 0
groups:
  - caravan
  - merchant_train
hostile: false
non_combatant: false
player_attack_immune: true
maxwander: -1
charm_immune: true
buffids: []
idlecommands:
  - 'emote restrings his bow with quick, practiced motions'
  - ''
  - 'emote scans the rooftops with a habitual, half-attentive sweep'
  - ''
  - 'emote checks the count in his quiver, lips moving silently'
character:
  name: Lars
  description: |
    A lean, sharp-eyed man in dark traveling leathers, a hunting bow
    over one shoulder and a quiver of fletched arrows at his hip. A
    long knife rides on his belt for close work. He has the still,
    economical stance of someone who learned early that small motions
    waste less energy than large ones.
  speciesid: 1
  level: 1
  gold: 12
  equipment:
    weapon:
      itemid: 50
  stats:
    strength:
      training: 18
    dexterity:
      training: 30
    vitality:
      training: 18
    perception:
      training: 25
    willpower:
      training: 12
    charisma:
      training: 10
  skills:
    weapon-combat: 18
    ranged-combat: 24
```

- [ ] **Step 2: Create `_datafiles/world/dogmud/behaviors/thornwall_city/318-lars.yaml`.**

```yaml
# Caravan guard Lars (318) — follower of Ketil.
# Same shape as Marta's btree.

tree:
  type: sequence
  children:

    - type: action
      do: party_ensure_npc_party
      leader_mob_id: 316
      home_room_id: 465

    - type: selector
      children:

        - type: sequence
          event: mob_idle
          children:
            - type: condition
              check: party_in_combat
            - type: action
              do: party_assist_target

        - type: sequence
          event: mob_idle
          children:
            - type: action
              do: party_follow_leader

        - type: sequence
          event: mob_hurt
          children:
            - type: action
              do: attack
```

- [ ] **Step 3: Create `_datafiles/world/dogmud/dialogue/thornwall_city/318-lars.yaml`.**

```yaml
mobid: 318

patterns:
  - keywords: [hello, hi, greetings, hail]
    text: |
      <ansi fg="mobname">Lars</ansi> looks up, makes a brief eye-contact
      acknowledgement, and goes back to checking his bowstring.

  - keywords: [bow, weapon, arrow]
    text: |
      "Yew. Cured slow. The bowyer up in Stillwater knows his trade.
      Not pretty but it shoots true."

  - keywords: [bandit, bandits, road]
    text: |
      <ansi fg="mobname">Lars</ansi> shrugs. "I see them before they
      see us. After that it's just arithmetic."

  - keywords: [ketil, marta, crew]
    text: |
      "Good people. Run a tight wagon. I get paid on time."

defaults:
  - text: |
      <ansi fg="mobname">Lars</ansi> nods politely and returns to his
      gear.
```

- [ ] **Step 4: Commit.**

```bash
git add _datafiles/world/dogmud/mobs/thornwall_city/318-lars.yaml _datafiles/world/dogmud/behaviors/thornwall_city/318-lars.yaml _datafiles/world/dogmud/dialogue/thornwall_city/318-lars.yaml
git commit -m "content(thornwall_city): add Lars, caravan guard (mob 318)

Stage 2 caravan crew: guard, ranged. statpool 170, hunting bow.
groups [caravan, merchant_train], player_attack_immune true,
maxwander -1.

Behavior tree mirrors Marta's: ensure caravan party (member of leader
316), assist target in combat, follow leader otherwise, counter-attack
on hurt.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 12: Add caravan crew to Thornwall depot spawninfo

**Files:**
- Modify: `_datafiles/world/dogmud/rooms/thornwall_city/465.yaml`

- [ ] **Step 1: Read the current spawninfo block.**

Run: `Read _datafiles/world/dogmud/rooms/thornwall_city/465.yaml`. Note any existing spawninfo entries. We will append, not replace.

- [ ] **Step 2: Add the three caravan crew to spawninfo.**

In `_datafiles/world/dogmud/rooms/thornwall_city/465.yaml`, find the `spawninfo:` block. Add the three new entries beneath whatever's already there:

```yaml
spawninfo:
  # ... existing entries ...
  - mobid: 316        # Ketil — caravan master
  - mobid: 317        # Marta — caravan guard
  - mobid: 318        # Lars — caravan guard
```

If `spawninfo:` doesn't yet exist on the room, add the whole block. If `spawninfo` exists but is `[]`/empty, replace the empty marker with the three entries.

- [ ] **Step 3: Delete any stale instance saves for room 465 (per CLAUDE.md SOP).**

Run: check `_datafiles/world/dogmud/rooms.instances/thornwall_city/465.yaml`. If it exists, delete it so the engine loads the fresh template on next boot.

```bash
rm -f _datafiles/world/dogmud/rooms.instances/thornwall_city/465.yaml
```

- [ ] **Step 4: Boot the server locally to confirm clean spawn.**

Per the Pre-Push SOP in CLAUDE.md, boot the server briefly and verify in stdout:
- `mobs.LoadDataFiles() loadedCount=...` — count went up by 3 (Ketil, Marta, Lars)
- No panic during data file loading
- The caravan party forms (no errors from `party_ensure_npc_party`)

If you can connect and run `party admin list-npc`, you should see one party with leader "Ketil (mob 316)" and 3 members.

- [ ] **Step 5: Commit.**

```bash
git add _datafiles/world/dogmud/rooms/thornwall_city/465.yaml
git commit -m "content(thornwall_city): spawn caravan crew at depot (room 465)

Adds mob 316 (Ketil), 317 (Marta), 318 (Lars) to spawninfo for the
Thornwall Market Square Center room — the caravan's home depot. On
boot, the three mobs spawn here and form a party via Stage 1
party_ensure_npc_party. The caravan_step state machine begins in
thornwall_dwell and cycles from there.

Stale instance save (if any) for this room was deleted to force a
clean template load.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 13: Bandit detune + add `hates: caravan`

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/north_road/283-bandit_lookout.yaml`
- Modify: `_datafiles/world/dogmud/mobs/north_road/284-bandit_fighter.yaml`
- Modify: `_datafiles/world/dogmud/mobs/north_road/285-bandit_caster.yaml`
- Modify: `_datafiles/world/dogmud/mobs/north_road/286-soren.yaml`

- [ ] **Step 1: Read all four bandit YAMLs to capture current statpools.**

Use `Read` for each. Record current values. The lookout is at 140 (per spec). The other three need to be looked up.

- [ ] **Step 2: Compute the detuned values.**

| Mob | Current | Drop % | New |
|---|---|---|---|
| 283 lookout | 140 | ~28% | 100 |
| 284 fighter | (read) | 25% | round to nearest 5 |
| 285 caster  | (read) | 25% | round to nearest 5 |
| 286 Soren   | (read) | 30% | round to nearest 5 |

Round results to the nearest 5 for tunability and clean YAML reading.

- [ ] **Step 3: Apply the four edits.**

For each mob YAML:
- Update `statpool: <new value>`
- Add `hates:` field if not present, append `caravan` to it. Format:
  ```yaml
  hates:
    - caravan
  ```
  If the mob already has a `hates` list (existing values for some other group), append `caravan` to it.

- [ ] **Step 4: Delete stale mob instance saves for the four bandits.**

Per CLAUDE.md instance-save SOP:

```bash
rm -f _datafiles/world/dogmud/mobs.instances/north_road/283.yaml
rm -f _datafiles/world/dogmud/mobs.instances/north_road/284.yaml
rm -f _datafiles/world/dogmud/mobs.instances/north_road/285.yaml
rm -f _datafiles/world/dogmud/mobs.instances/north_road/286.yaml
```

If those files don't exist, no-op.

- [ ] **Step 5: Boot the server, verify no panic.**

Run a brief local boot. Watch for `mobs.LoadDataFiles() loadedCount=...` to remain unchanged (no new mobs, just edits) and no panic from YAML parsing.

- [ ] **Step 6: Commit.**

```bash
git add _datafiles/world/dogmud/mobs/north_road/283-bandit_lookout.yaml _datafiles/world/dogmud/mobs/north_road/284-bandit_fighter.yaml _datafiles/world/dogmud/mobs/north_road/285-bandit_caster.yaml _datafiles/world/dogmud/mobs/north_road/286-soren.yaml
git commit -m "tune(bandits): ~25-30% statpool detune + hates caravan

Spec calls for the bandit pack to be a meaningful threat to a 3-person
baseline party (stats around 100) but not a guaranteed wipe for solo
players. Detune across the four mobs:

- 283 bandit_lookout: 140 → 100 (~28%)
- 284 bandit_fighter: ~25%
- 285 bandit_caster:  ~25%
- 286 Soren:          ~30%

All four pick up 'hates: caravan' so the Stage 2 caravan triggers a
brawl when passing through room 4052. Caravan group is declared on the
new caravan crew mobs (316/317/318) via 'groups: [caravan, ...]'.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 14: Schema docs + PATCH_NOTES

**Files:**
- Modify: `docs/schemas/mob.md`
- Modify: `docs/schemas/behavior.md`
- Modify: `PATCH_NOTES.md`

- [ ] **Step 1: Add `player_attack_immune` and `hates` documentation to `docs/schemas/mob.md`.**

Find the section that documents bool flags like `non_combatant` and `hostile`. Add entries for:

- `player_attack_immune` (bool, default false) — when true, the same gates that rebuff attacks on `non_combatant` mobs (attack/bash/grapple/kick/shoot/taunt/throw/trip and steal) also rebuff attacks on this mob. Used by Stage 2 caravan crew, who fight bandits but cannot be attacked by players. Mob-vs-mob attacks pass through unchanged.

If `hates` (the existing field added before Stage 2) is undocumented, add it too: list of group names this mob is hostile toward; combined with the target mob's `groups` field via `lookfortrouble`'s mob-vs-mob scan.

- [ ] **Step 2: Add `caravan_step` documentation to `docs/schemas/behavior.md`.**

Find the existing actions table (the one Stage 1 added entries to). Append:

| Action | Description |
|---|---|
| `caravan_step` | Drives the Stage 2 caravan state machine. Reads the caller's `MobState["caravan_state"]`, dispatches based on state category (dwell / transit / route), advances state on the right environmental conditions (timer expired / arrival / all stops visited). Used only on the caravan leader (Ketil); follower btrees use `party_follow_leader` + `party_assist_target` for transit and combat. State persistence keys: `caravan_state`, `caravan_state_started_round`, `caravan_route_index`. No params. |

- [ ] **Step 3: Add a Stage 2 entry to `PATCH_NOTES.md`.**

Per the Pre-Push SOP in CLAUDE.md, top of the file. Format follows the existing pattern. Sketch:

```markdown
## 2026-04-XX (Stage 2: Caravan System)

- Added the Thornwall ↔ Stillwater caravan: a 3-NPC delivery crew
  (Ketil, Marta, Lars) that walks a continuous loop visiting every
  vendor in both towns and triggering restock on arrival. The cycle
  takes ~1 in-game day (~1 hour real time).
- Vendor mobs in caravan-served zones (Stillwater, Thornwall City) no
  longer auto-restock on a per-mob timer — they restock only when the
  caravan visits. Vendors in non-served zones (Watchers Crossing,
  Sanctum Basin, etc.) keep the legacy auto-restock.
- The caravan crew can be examined and talked to but cannot be attacked
  by players (rebuff like shopkeepers). They will fight bandits along
  the road and have been statted to win the brawl at the North Road
  bandit camp.
- Bandit pack at the camp (lookout, fighter, caster, Soren) has been
  detuned by ~25-30% across the board so the road is challenging but
  passable for solo and small-party players.
- New `Mob.PlayerAttackImmune` flag for "fights enemies but rebuffs
  player attacks." Wired into all attack-style player commands.
- New `caravan_step` btree action drives the cycle.
- New config knobs: `Balance.CaravanServedZones`, `Balance.CaravanDepotDwellRounds`.
```

(Today's date when this lands.)

- [ ] **Step 4: Commit.**

```bash
git add docs/schemas/mob.md docs/schemas/behavior.md PATCH_NOTES.md
git commit -m "docs(schemas,patch-notes): document Stage 2 caravan system

Adds player_attack_immune + hates to mob schema, caravan_step to
behavior schema, and a PATCH_NOTES entry covering the new caravan,
the restock semantics shift, the caravan immunity, and the bandit
detune.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 15: In-game smoke test (manual)

**Files:** none — verification step only.

This task confirms the system end-to-end. Per the spec's verification plan; no commits unless a regression is found. Run the steps in order and log any issues for follow-up tasks.

- [ ] **Step 1: Boot server fresh.**

Restart the server. In the boot log, confirm:
- `mobs.LoadDataFiles() loadedCount=...` — count includes the 3 new caravan mobs
- No panic during data file loading
- No panic from `parties.LoadDataFiles()` or related party startup

- [ ] **Step 2: Confirm caravan party forms.**

Log in as an admin character. Walk to room 465 (Thornwall Market Square Center). Confirm Ketil, Marta, and Lars are present.

Run: `party admin list-npc`

Expected: one entry showing `Leader: Ketil (mob 316)`, 3 members, HomeRoom 465.

- [ ] **Step 3: Wait through one dwell, observe outbound transit.**

Wait at room 465 for the dwell timer (`CaravanDepotDwellRounds = 360` → 24 min real). Observe:
- "Ketil heads west" / similar movement messages
- All three caravan mobs leave together
- They walk through Thornwall, Thornwall Outskirts, Marches Spur Road, North Road toward Stillwater

If you don't want to wait 24 min, drop `CaravanDepotDwellRounds` to e.g. 30 in `_datafiles/config.yaml` for the test, then restore it after.

- [ ] **Step 4: Witness the bandit brawl at room 4052.**

Position yourself at room 4052 before the caravan arrives (or follow them). Confirm:
- Bandit lookout aggros on Ketil ("bandit lookout charges into Ketil")
- Bandit lookout calls help via Stage 1 `party_call_help`
- Camp mobs respond and engage
- Caravan party fights back via `party_assist_target`
- Caravan wins (all 4 bandits dead)
- Caravan continues westward

If the caravan loses or the brawl never starts, debug at this point — likely the `lookfortrouble` group-hate scan isn't firing, or the caravan's stat tuning is too low.

- [ ] **Step 5: Verify Stillwater vendor visits.**

Follow the caravan into Stillwater. At each vendor stop (rooms 4102, 4103, 4105, 4106, 4125, 4126, 4135, 4143):
- Confirm the room flavor message: "The caravan crew unloads a crate of supplies; <vendor> nods their thanks."
- Optional: before the visit, run `list` at the vendor to see stock; after the visit, re-run `list` to see stock topped off.

- [ ] **Step 6: Verify Stillwater dwell, then return.**

Caravan arrives at room 4109 (Stillwater depot). Dwells for `CaravanDepotDwellRounds`. Then transitions to `inbound_transit` and walks back. Bandit camp at 4052 will have respawned — second brawl, caravan wins again.

- [ ] **Step 7: Verify Thornwall vendor visits.**

Same as Step 5 but for Thornwall vendor rooms (464, 470, 471, 475, 480, 481, 482, 483, 507).

- [ ] **Step 8: Confirm cycle restarts.**

Caravan arrives back at room 465. State transitions to `thornwall_dwell`. Cycle is now ready to repeat.

- [ ] **Step 9: Test player-attack rebuff.**

While caravan is at any room, try `attack ketil`. Expected message: "You can't attack Ketil." No combat starts. Confirm `mob.Character.Aggro` is unchanged (use admin observation if available).

Try the same with `bash`, `kick`, `shoot ketil`, `steal ketil`. Each should rebuff.

- [ ] **Step 10: Verify auto-restock suppression.**

Visit a Stillwater vendor (e.g., smith Brindle in room 4106). Buy items repeatedly until stock is empty. Wait > 200 rounds (the legacy restock interval). Confirm stock does NOT replenish. Wait for next caravan visit. Confirm stock tops off at that point.

Visit a non-served-zone vendor (e.g., one in Watchers Crossing). Confirm legacy auto-restock still works there.

- [ ] **Step 11: Caravan wipe + recovery (if comfortable as admin).**

Use admin tools to kill all 3 caravan mobs (`kill ketil`, `kill marta`, `kill lars` if admin allows). Confirm:
- Stage 1 leader-death dissolution fires
- Mobs respawn at room 465 per spawninfo on their normal timers
- New btree ticks form a fresh caravan party
- State defaults to `thornwall_dwell`
- Cycle resumes from there

- [ ] **Step 12: Document any regressions.**

If any step fails, add an entry to `bug_log.txt` (or wherever issues are tracked) with: which step, what was expected, what happened. Open a follow-up task in the plan checklist for the fix.

---

## Self-review checklist

Run this after the plan is committed but before starting implementation.

**Spec coverage:**
- ✅ State machine (spec section "Data model: state machine") → Task 6
- ✅ Cycle ~900 rounds (spec) → Task 2 (config) + Task 6 (state durations)
- ✅ Restock semantics A (spec section "Restock semantics") → Task 3 (suppress) + Task 7 (visit invocation) + Task 8 (visit dispatch)
- ✅ Combat at 4052 group-hate (spec section "Combat at room 4052") → Task 4 + Task 13
- ✅ Player attack rebuff (spec section "Player attack rebuff") → Task 1
- ✅ Caravan crew of 3 (spec) → Tasks 9-11
- ✅ Reuses Stage 1 primitives (spec) → Tasks 9-11 btrees
- ✅ Depot rooms (spec) → locked at plan time + Task 12
- ✅ Vendor route (spec) → Task 5
- ✅ Bandit detune (spec) → Task 13
- ✅ Cross-zone movement (spec note) → smoke test in Task 15
- ✅ Schema docs (spec files-affected) → Task 14
- ✅ Out-of-scope items (spec) — none of them have tasks (correct).

**Type consistency:**
- `Route.DepartFromRoomId`, `Route.ArriveAtRoomId`, `Route.VendorStopIds` — used identically in tasks 5, 6, 8.
- `CaravanState`, `StateThornwallDwell`, etc. — defined in task 6, used in task 8 with matching names.
- `caravan.VisitVendorsInRoom` — defined in task 7, called in task 8 with matching signature `(roomId int) []string`.
- `caravan.FormatDeliveryMessage` — defined in task 7, called in task 8.
- `caravan.RouteForState` — defined in task 6, used in task 8.
- `Balance.IsCaravanServedZone` — defined in task 2, used in task 3.
- `Mob.PlayerAttackImmune` — defined in task 1, referenced in task 9-11 YAMLs.

**Placeholder scan:**
- "TBD" appears in two contexts: (1) bandit detune target values in the task 13 plan ("TBR" → to be read at task time, with explicit instructions for the read; this is a deliberate "look up before writing" step, not a deferred design decision); (2) caravan crew equipment item IDs in tasks 9-11 (with explicit fallback: leave equipment blank if hand-picking real items is a rabbit hole). Both are acceptable — the engineer has clear instructions for either path.
- No "implement later" or "fill in details" outside those.
- Every code step shows the actual code.
- Test steps include actual test code.
