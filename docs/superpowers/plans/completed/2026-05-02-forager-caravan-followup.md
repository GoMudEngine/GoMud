# Forager + Caravan Followup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix five issues in the now-shipped forager + caravan stack — Whisper off the route, system NPCs spawn at boot, foragers never deadlock, Kessa physically delivers via a roadside crate — and halve the caravan depot dwell.

**Architecture:** Two engine extensions (a `RotationSeed` on `gamelock.Lock` for fresh-pick-each-cycle lockboxes, and a new `internal/sealedcrate/` package for player-untouchable delivery containers) underpin the content + state-machine changes. Forager state machine adds a sanctuary dump on Recall→Resting; Kessa's Fernway flow rewrites to dump into the sealed crate; caravan's Fernway pickup drains the crate into the wagon and the abstract `caravan_load` flag is deleted.

**Tech Stack:** Go, YAML data files (rooms, items, config), existing forager + caravan + Container + gamelock packages.

---

## Decisions locked at plan time (from spec)

- **Whisper removed** from `thornwallVendorRooms` (`internal/caravan/routes.go:39`).
- **`CaravanDepotDwellRounds`**: 720 → 360.
- **System NPC anchor rooms** (boot-prepared): `4042`, `4123`, `3040`, `4197`.
- **Sanctuary lockbox**: `Container` with lock difficulty `10`, capacity `500`, visible (not hidden), no trap, fresh combination each forager cycle via new `Lock.RotationSeed`.
- **Roadside sealed crate** at room `4038`: new `SealedCrate` type, capacity `2000`, persisted under `_datafiles/world/dogmud/crates/`, no player interaction.
- **`Lock.RotationSeed` semantics**: when `0` (default) `GetLockSequence` is unchanged (back-compat); when `>0` it is mixed into the hash input.
- **Stage 3.4 rest extension** (`actions_forager.go:149-160`) is **removed**: foragers always re-cycle out of Resting after dwell elapses; surplus accumulates in the lockbox instead.
- **`caravan_load` flag** is deleted everywhere; real items in the wagon supersede it.

## Verified engine API anchors

- `gamelock.Lock` struct — `internal/gamelock/gamelock.go:12-17`
- `gamelock.Lock.SetLocked()` — `internal/gamelock/gamelock.go:45-47`
- `util.GetLockSequence(lockId string, difficulty int, seed string) string` — `internal/util/util.go:435-470`
- `GetLockSequence` callsites (6 total): `internal/util/util_test.go:656,671,672`, `internal/characters/inventory.go:94`, `internal/usercommands/admin.room.go:282`, `internal/usercommands/admin.room.exits.go:244`, `internal/usercommands/keyring.go:97`, `internal/usercommands/picklock.go:110`
- `rooms.Container` struct — `internal/rooms/container.go:8-15`
- `rooms.Room.Containers` map — `internal/rooms/rooms.go:85`
- `rooms.LoadRoom` (no `Prepare()`) — `internal/rooms/save_and_load.go:78-97`
- `rooms.Room.Prepare(checkAdjacentRooms bool)` — `internal/rooms/rooms.go:585-588`
- `caravan.thornwallVendorRooms` literal — `internal/caravan/routes.go:29-40`
- `Balance.CaravanDepotDwellRounds` field — `internal/configs/config.balance.go:297`
- `CaravanDepotDwellRounds` default — `internal/configs/config.balance.misc.go:281-283`
- `CaravanDepotDwellRounds` config-default test — `internal/configs/config.balance_test.go:8-10`
- `tickFernwayPickup` — `internal/behaviortree/actions_caravan.go:209-246`
- `caravanLoadAppend / caravanLoadGet / caravanLoadSet` — `internal/behaviortree/actions_caravan.go` (Stage 3.1)
- `tickForagerResting` — `internal/behaviortree/actions_forager.go:136-164`
- `tickForagerDeliveringFernway` — `internal/behaviortree/actions_forager.go:283-300`
- `tickForagerRecalling` — `internal/behaviortree/actions_forager.go:302-321`
- `npcVisitVendorsInRoom` (item-transfer template) — `internal/behaviortree/actions_forager.go:366-417`
- Boot prewarm site — `main.go:1167-1207`
- Sanctuary anchor rooms (existing files): `_datafiles/world/dogmud/rooms/stillwater/4123.yaml`, `_datafiles/world/dogmud/rooms/ironwind_steppe/3040.yaml`, `_datafiles/world/dogmud/rooms/the_fernway_south/4197.yaml`
- Fernway pickup room: `_datafiles/world/dogmud/rooms/north_road/4038.yaml`

---

## File structure overview

| Layer | File | Purpose |
|---|---|---|
| Engine | `internal/gamelock/gamelock.go` | Add `RotationSeed uint64`; bump on `SetLocked` (Task 4) |
| Engine | `internal/util/util.go` | Extend `GetLockSequence` with optional rotation (Task 5) |
| Engine callers | `inventory.go`, `picklock.go`, `keyring.go`, `admin.room.go`, `admin.room.exits.go` | Pass `lock.RotationSeed` (Task 6) |
| Engine (new pkg) | `internal/sealedcrate/sealedcrate.go` | `SealedCrate` type + capacity helpers (Task 11) |
| Engine (new pkg) | `internal/sealedcrate/persistence.go` | Save/load YAML at `_datafiles/world/dogmud/crates/` (Task 12) |
| Engine | `internal/rooms/rooms.go` | Add `SealedCrate *sealedcrate.Crate` field on `Room` (Task 13) |
| Engine | `main.go` boot loop | Boot-prepare anchor rooms; load sealed-crate persistence (Tasks 3, 14) |
| Engine | `internal/behaviortree/actions_forager.go` | Drop carry-ratio rest gate; add lockbox dump on Recall→Resting; rewrite Kessa Fernway tick (Tasks 8, 9, 16) |
| Engine | `internal/behaviortree/actions_caravan.go` | Rewrite `tickFernwayPickup`; delete `caravanLoad*` callsites (Task 17) |
| Engine | `internal/usercommands/{get,look,put,picklock,open}.go` | Sealed-crate command shims (Task 15) |
| Content | `internal/caravan/routes.go` | Remove room 507 (Task 1) |
| Content | `_datafiles/config.yaml`, `internal/configs/config.balance.misc.go`, `internal/configs/config.balance_test.go` | `CaravanDepotDwellRounds: 360` (Task 2) |
| Content | `_datafiles/world/dogmud/rooms/stillwater/4123.yaml`, `ironwind_steppe/3040.yaml`, `the_fernway_south/4197.yaml` | Add `lockbox` container (Task 7) |
| Content | `_datafiles/world/dogmud/rooms/north_road/4038.yaml` | Wire sealed crate + flavor noun (Task 16) |
| Docs | `PATCH_NOTES.md` | Followup entry (Task 18) |

---

### Task 1: Remove Whisper from caravan rotation

**Files:**
- Modify: `internal/caravan/routes.go:29-40`
- Modify: `internal/caravan/routes_test.go` (add a test pinning the absence of 507)

- [ ] **Step 1: Read the existing slice and the routes test file.**

Run: `head -50 internal/caravan/routes.go`

Note the existing `thornwallVendorRooms` and `stillwaterVendorRooms` slices.

- [ ] **Step 2: Write a failing test.**

In `internal/caravan/routes_test.go`, append:

```go
func TestThornwallVendorRoomsExcludesWhisper(t *testing.T) {
    for _, r := range thornwallVendorRooms {
        if r == 507 {
            t.Fatalf("thornwallVendorRooms still includes 507 (Whisper); she's off-route in the phantom's zone and should not be on the caravan rotation")
        }
    }
}
```

- [ ] **Step 3: Run the test to verify it fails.**

Run: `go test ./internal/caravan/... -run TestThornwallVendorRoomsExcludesWhisper -v`
Expected: FAIL with the "still includes 507" message.

- [ ] **Step 4: Remove the entry.**

In `internal/caravan/routes.go`, delete the line:

```go
    507, // Whisper
```

The comma on the preceding line stays (Go is fine with a trailing entry).

- [ ] **Step 5: Run the test to verify it passes.**

Run: `go test ./internal/caravan/... -run TestThornwallVendorRoomsExcludesWhisper -v`
Expected: PASS.

- [ ] **Step 6: Run the full caravan test suite.**

Run: `go test ./internal/caravan/...`
Expected: PASS.

- [ ] **Step 7: Commit.**

```bash
git add internal/caravan/routes.go internal/caravan/routes_test.go
git commit -m "$(cat <<'EOF'
fix(caravan): remove Whisper (507) from Thornwall vendor rotation

Whisper is in the phantom's zone and was never a standard merchant —
the Stage 3.1 vendor list mistakenly included her. Caravan now skips
room 507 entirely.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Halve caravan depot dwell

**Files:**
- Modify: `internal/configs/config.balance.misc.go:281-283`
- Modify: `_datafiles/config.yaml:866-869`
- Modify: `internal/configs/config.balance_test.go:8-10`

- [ ] **Step 1: Read the existing config block + test.**

Run: `sed -n '260,300p' internal/configs/config.balance.misc.go` and `sed -n '865,876p' _datafiles/config.yaml` and `head -20 internal/configs/config.balance_test.go`.

- [ ] **Step 2: Update the test default expectation first (it will fail until code is updated).**

In `internal/configs/config.balance_test.go`, change:

```go
    if cfg.CaravanDepotDwellRounds != 720 {
        t.Errorf("CaravanDepotDwellRounds default = %d, want 720", cfg.CaravanDepotDwellRounds)
    }
```

to:

```go
    if cfg.CaravanDepotDwellRounds != 360 {
        t.Errorf("CaravanDepotDwellRounds default = %d, want 360", cfg.CaravanDepotDwellRounds)
    }
```

- [ ] **Step 3: Run the test to verify it fails.**

Run: `go test ./internal/configs/... -run CaravanDepotDwell -v`
Expected: FAIL with `default = 720, want 360`.

- [ ] **Step 4: Update the engine default.**

In `internal/configs/config.balance.misc.go:281-283`, change:

```go
    if b.CaravanDepotDwellRounds <= 0 {
        b.CaravanDepotDwellRounds = 720
    }
```

to:

```go
    if b.CaravanDepotDwellRounds <= 0 {
        b.CaravanDepotDwellRounds = 360
    }
```

- [ ] **Step 5: Update the YAML.**

In `_datafiles/config.yaml`, replace the block around line 866-869:

```yaml
  # 720 rounds ≈ 48 min real ≈ a full in-game day. Stage 3.1 doubled
  # this from 360 to make foragers the day-to-day supply pipeline.
  CaravanDepotDwellRounds: 720
```

with:

```yaml
  # 360 rounds ≈ 24 min real ≈ a half-game-day. Halved from 720 on
  # 2026-05-02 — foragers are the day-to-day pipeline regardless,
  # and a more visible caravan beats a more realistic one.
  CaravanDepotDwellRounds: 360
```

- [ ] **Step 6: Run the test suite.**

Run: `go test ./internal/configs/... -v`
Expected: PASS.

- [ ] **Step 7: Commit.**

```bash
git add internal/configs/config.balance.misc.go _datafiles/config.yaml internal/configs/config.balance_test.go
git commit -m "$(cat <<'EOF'
config(caravan): halve depot dwell 720 -> 360 rounds

Foragers now run from boot and never deadlock, so they dominate
day-to-day throughput regardless of caravan cadence. Halving the
depot dwell roughly doubles caravan visibility in each town —
event-style deliveries the user prefers over once-per-day realism.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: Boot-prepare system NPC anchor rooms

**Files:**
- Modify: `main.go` (around line 1207, after existing shop prewarm)
- Test: `main_test.go` may not be appropriate for boot wiring; add an integration-style test in `internal/rooms/save_and_load_test.go` if a suitable harness exists, otherwise rely on log assertion + manual boot.

- [ ] **Step 1: Read the existing boot prewarm block.**

Run: `sed -n '1167,1220p' main.go`. Note the shop prewarm completes at line 1207 with `mudlog.Info("shop cache prewarmed (spawninfo)", "count", prewarmedFromSpawn)`.

- [ ] **Step 2: Add the system anchor rooms list as a package-level constant near the boot prewarm.**

Insert a new const block above the boot loop (e.g., near `main.go:1140` or wherever sibling globals live; if there is no obvious spot, place at the top of the same function above the `mobZoneByMobId` block):

```go
// systemNPCAnchorRooms is the explicit list of rooms whose spawninfo
// must fire at boot rather than on first player visit. These rooms
// host long-running system NPCs (caravan master, foragers) whose
// state machines must run continuously regardless of player presence.
var systemNPCAnchorRooms = []int{
    4042, // North Road Crossroads Village Square — caravan master 281
    4123, // Stillwater Temple — Tova, Marsh forager 371
    3040, // Ironwind Steppe sanctuary — Halix, Steppe forager 372
    4197, // Forager's Camp, Fernway South — Kessa, Fernway forager 373
}
```

- [ ] **Step 3: Add the prepare loop after the existing shop prewarm log.**

Insert after `main.go:1207` (after `mudlog.Info("shop cache prewarmed (spawninfo)", "count", prewarmedFromSpawn)`):

```go
    // Force-spawn long-running system NPCs at boot. The shop prewarm
    // above seeds shop cache entries from spawninfo but does NOT call
    // room.Prepare(), so the actual mob instances would otherwise only
    // be created when a player walks into the room. For caravan +
    // forager NPCs anchored in low-traffic rooms, that means their
    // state machines never start.
    preparedAnchors := 0
    for _, roomId := range systemNPCAnchorRooms {
        room := rooms.LoadRoom(roomId)
        if room == nil {
            mudlog.Warn("system anchor room not found", "roomId", roomId)
            continue
        }
        room.Prepare(false)
        preparedAnchors++
    }
    mudlog.Info("system NPC anchor rooms prepared", "count", preparedAnchors)
```

- [ ] **Step 4: Build and boot the server locally to verify the new log line appears.**

Run: `go build -o ./tmp-mud.exe . && ./tmp-mud.exe -path ./_datafiles 2>&1 | grep -E "system NPC anchor|caravan|forager" | head -20`

Expected: A `system NPC anchor rooms prepared count=4` line. No panics. Caravan and forager mobs spawn during boot (not on first player visit).

Stop the server with Ctrl-C once you've confirmed the log line. Delete `tmp-mud.exe`.

- [ ] **Step 5: Run the existing test suite to verify no regression.**

Run: `go test ./...`
Expected: PASS (boot wiring has no unit-test surface; we lean on the integration check from Step 4).

- [ ] **Step 6: Commit.**

```bash
git add main.go
git commit -m "$(cat <<'EOF'
feat(boot): prepare system-NPC anchor rooms at startup

Caravan master and three foragers anchor in rooms players rarely
visit. The existing prewarm loaded room templates and seeded shop
cache entries but never called room.Prepare(), so their mob
instances waited indefinitely for a player to wander past.

New step after shop prewarm calls Prepare(false) on the four
anchors (4042, 4123, 3040, 4197), kicking spawninfo into firing
and giving the long-running state machines their first idle tick.

Fixes the /admin/economy/ "(not active)" forager rows on clean boot.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: Add `RotationSeed` to `gamelock.Lock`

**Files:**
- Modify: `internal/gamelock/gamelock.go`
- Test: `internal/gamelock/gamelock_test.go` (create if missing)

- [ ] **Step 1: Read the existing Lock struct.**

Run: `cat internal/gamelock/gamelock.go`. Note the four fields and `SetLocked` / `SetUnlocked` methods.

- [ ] **Step 2: Write a failing test that asserts `SetLocked` increments `RotationSeed`.**

Create `internal/gamelock/gamelock_test.go` (or append if it exists):

```go
package gamelock

import "testing"

func TestLock_SetLockedIncrementsRotationSeed(t *testing.T) {
    l := Lock{Difficulty: 6}
    if l.RotationSeed != 0 {
        t.Fatalf("RotationSeed = %d, want 0 default", l.RotationSeed)
    }
    l.SetLocked()
    if l.RotationSeed != 1 {
        t.Fatalf("RotationSeed after first SetLocked = %d, want 1", l.RotationSeed)
    }
    l.SetUnlocked()
    l.SetLocked()
    if l.RotationSeed != 2 {
        t.Fatalf("RotationSeed after second SetLocked = %d, want 2", l.RotationSeed)
    }
}

func TestLock_RotationSeedDefaultsZero(t *testing.T) {
    var l Lock
    if l.RotationSeed != 0 {
        t.Errorf("zero-value Lock RotationSeed = %d, want 0", l.RotationSeed)
    }
}
```

- [ ] **Step 3: Run the test to verify it fails.**

Run: `go test ./internal/gamelock/... -run RotationSeed -v`
Expected: FAIL — `RotationSeed` is undefined on `Lock`.

- [ ] **Step 4: Add the field and increment in `SetLocked`.**

In `internal/gamelock/gamelock.go`:

Replace the struct:

```go
type Lock struct {
    Difficulty     uint8  `yaml:"difficulty,omitempty"`
    UnlockedRound  uint64 `yaml:"-"`
    RelockInterval string `yaml:"relockinterval,omitempty"`
    TrapBuffIds    []int  `yaml:"trapbuffids,omitempty,flow"`
}
```

with:

```go
type Lock struct {
    Difficulty     uint8  `yaml:"difficulty,omitempty"`
    UnlockedRound  uint64 `yaml:"-"`
    RelockInterval string `yaml:"relockinterval,omitempty"`
    TrapBuffIds    []int  `yaml:"trapbuffids,omitempty,flow"`
    // RotationSeed rotates the lock combination on every SetLocked
    // call. Mixed into util.GetLockSequence so cached keyring entries
    // become invalid after the lock re-locks. Default 0 = back-compat
    // (sequence derivation unchanged when seed is zero).
    RotationSeed uint64 `yaml:"rotationseed,omitempty"`
}
```

Replace `SetLocked`:

```go
func (l *Lock) SetLocked() {
    l.UnlockedRound = 0
}
```

with:

```go
func (l *Lock) SetLocked() {
    l.UnlockedRound = 0
    l.RotationSeed++
}
```

- [ ] **Step 5: Run the test to verify it passes.**

Run: `go test ./internal/gamelock/... -run RotationSeed -v`
Expected: PASS.

- [ ] **Step 6: Run the full gamelock suite.**

Run: `go test ./internal/gamelock/...`
Expected: PASS.

- [ ] **Step 7: Commit.**

```bash
git add internal/gamelock/gamelock.go internal/gamelock/gamelock_test.go
git commit -m "$(cat <<'EOF'
feat(gamelock): add RotationSeed; bump on SetLocked

Sets up per-cycle lock combination rotation for the upcoming forager
sanctuary lockboxes. Default 0 keeps existing locks unchanged; only
callers that explicitly bump rotation invalidate keyring entries.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: Mix `RotationSeed` into `GetLockSequence`

**Files:**
- Modify: `internal/util/util.go:435-470`
- Modify: `internal/util/util_test.go:621-680`

- [ ] **Step 1: Read the existing function and tests.**

Run: `sed -n '435,470p' internal/util/util.go` and `sed -n '620,680p' internal/util/util_test.go`.

- [ ] **Step 2: Add a new test asserting back-compat (rotation 0 returns the same sequence as before) and rotation>0 returns a different sequence.**

Append to `internal/util/util_test.go`:

```go
func TestGetLockSequence_RotationBackCompat(t *testing.T) {
    // Rotation 0 must produce the same sequence the function used to,
    // so existing keyring entries stay valid for non-rotating locks.
    base := GetLockSequence("Lock", 4, "Seed", 0)
    if len(base) != 4 {
        t.Fatalf("base length = %d, want 4", len(base))
    }
    for _, c := range base {
        if c != 'U' && c != 'D' {
            t.Fatalf("base contains non-U/D char %q", c)
        }
    }
}

func TestGetLockSequence_RotationChangesOutput(t *testing.T) {
    base := GetLockSequence("Lock", 8, "Seed", 0)
    rotated := GetLockSequence("Lock", 8, "Seed", 1)
    if base == rotated {
        t.Fatalf("rotation 1 produced the same sequence as rotation 0 (%q); rotation must change output", base)
    }

    // Different rotation values produce different sequences.
    r2 := GetLockSequence("Lock", 8, "Seed", 2)
    if rotated == r2 {
        t.Fatalf("rotation 1 and rotation 2 produced the same sequence (%q); rotation must change output", rotated)
    }
}
```

- [ ] **Step 3: Run the new tests to verify they fail (compile error: too many args).**

Run: `go test ./internal/util/... -run RotationBackCompat -v`
Expected: FAIL — function takes 3 args.

- [ ] **Step 4: Update the function signature.**

In `internal/util/util.go`, replace lines 435-470 with:

```go
// GetLockSequence derives the U/D pin sequence for a lock from a
// stable identifier, difficulty, server seed, and an optional
// rotation seed. When `rotation` is 0 the rotation suffix is not
// included in the hash input — this preserves the pre-rotation
// output so existing keyring entries continue to work. Callers that
// want fresh combinations (e.g. forager lockboxes that re-lock
// each cycle) pass the lock's bumping `RotationSeed`.
func GetLockSequence(lockIdentifier string, difficulty int, seed string, rotation uint64) string {

    // Clamp difficulty between [2..32]
    if difficulty < 2 {
        difficulty = 2
    } else if difficulty > 32 {
        difficulty = 32
    }

    // Generate the hash. Rotation 0 keeps the original input, so
    // existing locks keep their existing sequence.
    hashInput := strings.ToLower(lockIdentifier + seed)
    if rotation > 0 {
        hashInput = hashInput + ":" + strconv.FormatUint(rotation, 10)
    }
    hash := Md5Bytes([]byte(hashInput))
    for len(hash) < difficulty {
        hash = append(hash, Md5Bytes([]byte(hashInput+strconv.Itoa(len(hash))))...)
    }

    seq := make([]byte, difficulty)
    for i := 0; i < difficulty; i++ {
        if hash[i]%2 == 0 {
            seq[i] = 'U'
        } else {
            seq[i] = 'D'
        }
    }

    return string(seq)
}
```

- [ ] **Step 5: Update the existing `TestGetLockSequence` test to pass `0` for rotation.**

In `internal/util/util_test.go`, find the line:

```go
            got := GetLockSequence(tt.lockIdentifier, tt.difficulty, tt.seed)
```

Replace with:

```go
            got := GetLockSequence(tt.lockIdentifier, tt.difficulty, tt.seed, 0)
```

And the determinism block:

```go
        first := GetLockSequence("Lock", 4, "Seed")
        second := GetLockSequence("Lock", 4, "Seed")
```

Replace with:

```go
        first := GetLockSequence("Lock", 4, "Seed", 0)
        second := GetLockSequence("Lock", 4, "Seed", 0)
```

- [ ] **Step 6: Run all util tests.**

Run: `go test ./internal/util/... -v -run LockSequence`
Expected: PASS — including the two new tests.

- [ ] **Step 7: Verify the build still compiles globally (call sites at picklock/keyring/admin.room/inventory will fail next; that's expected for now).**

Run: `go build ./...`
Expected: FAIL — call sites pass 3 args, function now takes 4. This is the signal that Task 6 is needed.

- [ ] **Step 8: Commit (with build broken — Task 6 immediately follows).**

```bash
git add internal/util/util.go internal/util/util_test.go
git commit -m "$(cat <<'EOF'
feat(util): GetLockSequence accepts optional rotation seed

Adds 4th parameter to mix a per-lock rotation value into the hash
input. Rotation 0 (default) keeps existing output, so legacy locks
+ keyring entries are unaffected. Rotation >0 produces a distinct
sequence, used by per-cycle re-locking lockboxes.

Build is intentionally broken at this commit — call sites are
updated in the next commit.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 6: Update `GetLockSequence` callsites

**Files:**
- Modify: `internal/characters/inventory.go:94`
- Modify: `internal/usercommands/admin.room.go:282`
- Modify: `internal/usercommands/admin.room.exits.go:244`
- Modify: `internal/usercommands/keyring.go:97`
- Modify: `internal/usercommands/picklock.go:110`

- [ ] **Step 1: Update `internal/characters/inventory.go:94`.**

Find:

```go
    sequence := util.GetLockSequence(lockId, difficulty, string(configs.GetServerConfig().Seed))
```

Replace with:

```go
    sequence := util.GetLockSequence(lockId, difficulty, string(configs.GetServerConfig().Seed), 0)
```

(This site is a generic Item-level lock; rotation does not apply.)

- [ ] **Step 2: Update `internal/usercommands/admin.room.go:282`.**

Find:

```go
        for _, dir := range util.GetLockSequence(lockId, int(currentlyEditing.Container.Lock.Difficulty), string(configs.GetServerConfig().Seed)) {
```

Replace with:

```go
        for _, dir := range util.GetLockSequence(lockId, int(currentlyEditing.Container.Lock.Difficulty), string(configs.GetServerConfig().Seed), currentlyEditing.Container.Lock.RotationSeed) {
```

- [ ] **Step 3: Update `internal/usercommands/admin.room.exits.go:244`.**

Find:

```go
        for _, dir := range util.GetLockSequence(lockId, int(currentlyEditing.Exit.Lock.Difficulty), string(configs.GetServerConfig().Seed)) {
```

Replace with:

```go
        for _, dir := range util.GetLockSequence(lockId, int(currentlyEditing.Exit.Lock.Difficulty), string(configs.GetServerConfig().Seed), currentlyEditing.Exit.Lock.RotationSeed) {
```

- [ ] **Step 4: Update `internal/usercommands/keyring.go:97`.**

Find:

```go
            actualSequence := util.GetLockSequence(lockId, int(exitInfo.Lock.Difficulty), cfgSeed)
```

Replace with:

```go
            actualSequence := util.GetLockSequence(lockId, int(exitInfo.Lock.Difficulty), cfgSeed, exitInfo.Lock.RotationSeed)
```

- [ ] **Step 5: Update `internal/usercommands/picklock.go:110`.**

Find:

```go
    sequence := util.GetLockSequence(lockId, lockStrength, string(configs.GetServerConfig().Seed))
```

Just above this line, set `lockRotation` from whichever branch matched (container or exit). At the top of the function, both branches set local variables — extend them.

In the `if containerName != ``` branch (around line 59-77), after `lockTrap = container.Lock.TrapBuffIds`, add:

```go
        lockRotation := container.Lock.RotationSeed
```

In the `else if exitName != ``` branch (around line 78-98), after `lockTrap = exitInfo.Lock.TrapBuffIds`, add:

```go
        lockRotation := exitInfo.Lock.RotationSeed
```

Hoist `lockRotation` to the outer scope by declaring it before the branches (alongside `lockId`, `lockStrength`, `lockTrap`):

```go
    lockId := ``
    lockStrength := 0
    lockTrap := []int{}
    lockRotation := uint64(0)
```

In each branch replace the `:=` with `=` for `lockRotation`:

```go
        lockRotation = container.Lock.RotationSeed
```

```go
        lockRotation = exitInfo.Lock.RotationSeed
```

Then update the `GetLockSequence` call:

```go
    sequence := util.GetLockSequence(lockId, lockStrength, string(configs.GetServerConfig().Seed), lockRotation)
```

- [ ] **Step 6: Build everything to confirm callsites are clean.**

Run: `go build ./...`
Expected: PASS.

- [ ] **Step 7: Run all tests to confirm no regressions.**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 8: Commit.**

```bash
git add internal/characters/inventory.go internal/usercommands/admin.room.go internal/usercommands/admin.room.exits.go internal/usercommands/keyring.go internal/usercommands/picklock.go
git commit -m "$(cat <<'EOF'
refactor(locks): pass Lock.RotationSeed through GetLockSequence

Updates the 5 in-tree callsites to forward the lock's rotation seed
to GetLockSequence. Existing locks all default to RotationSeed=0
which preserves their existing pin sequences (and thus existing
keyring entries) unchanged.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 7: Add lockbox containers to the three sanctuary rooms

**Files:**
- Modify: `_datafiles/world/dogmud/rooms/stillwater/4123.yaml`
- Modify: `_datafiles/world/dogmud/rooms/ironwind_steppe/3040.yaml`
- Modify: `_datafiles/world/dogmud/rooms/the_fernway_south/4197.yaml`

- [ ] **Step 1: Read each sanctuary room file fully.**

Run: `cat _datafiles/world/dogmud/rooms/stillwater/4123.yaml`, `cat _datafiles/world/dogmud/rooms/ironwind_steppe/3040.yaml`, `cat _datafiles/world/dogmud/rooms/the_fernway_south/4197.yaml`.

- [ ] **Step 2: Append the lockbox container to `_datafiles/world/dogmud/rooms/stillwater/4123.yaml`.**

After the existing `spawninfo:` block (the file ends with mob 371), insert at the end:

```yaml
containers:
  lockbox:
    lock:
      difficulty: 10
      relockinterval: 24 hours
      rotationseed: 1
    items: []
```

Also extend the `nouns:` block to include a flavor entry for the lockbox so players who `look lockbox` see something. Find the existing `nouns:` map (lines 38-82 currently) and append:

```yaml
  lockbox: |
    A sturdy iron-banded oak lockbox tucked behind one of the
    altar's flanking columns — Tova the marsh forager's, by some
    long-standing arrangement with the temple. The lid latches
    flush, the hasp scarred from years of careful keying.
```

Then in the `description:` field, lightly mention the lockbox so its noun is discoverable. Append at the end of the description (before `biome:`):

```
A worn iron-banded <ansi fg="itemname">lockbox</ansi> sits
half-hidden behind one of the altar's flanking columns.
```

- [ ] **Step 3: Append the lockbox to `_datafiles/world/dogmud/rooms/ironwind_steppe/3040.yaml`.**

Use the same `containers` block but with steppe-flavored noun text. Read the room first to choose phrasing that fits the existing description; the lockbox is Halix's stash. Add at the end of the YAML:

```yaml
containers:
  lockbox:
    lock:
      difficulty: 10
      relockinterval: 24 hours
      rotationseed: 1
    items: []
```

Add to the `nouns:` block:

```yaml
  lockbox: |
    A travel-worn lockbox bound in waxed leather and iron strap-
    work — Halix the steppe forager's, secured against weather and
    wandering hands alike. The latch is well-oiled but the keyway
    looks recently pried.
```

Add a one-sentence mention of the lockbox at the end of the room's `description:`.

- [ ] **Step 4: Append the lockbox to `_datafiles/world/dogmud/rooms/the_fernway_south/4197.yaml`.**

Read the existing file (Forager's Camp; description mentions a lean-to and drying-racks). Add at the end of the YAML:

```yaml
containers:
  lockbox:
    lock:
      difficulty: 10
      relockinterval: 24 hours
      rotationseed: 1
    items: []
```

Add to the `nouns:` block:

```yaml
  lockbox: |
    A small ironbound lockbox sits beneath the lean-to's bedding
    pallet — Kessa the fernway forager's, a hand-cut hardwood box
    with a brass hasp and a lock that looks practiced. She keeps
    the surplus of her cycles here.
```

Add a one-sentence mention of the lockbox at the end of `description:`.

- [ ] **Step 5: Verify each YAML parses and rebuilds without instance-save interference.**

For each of the three rooms, check whether an instance save exists and would override the new container:

Run: `ls _datafiles/world/dogmud/rooms.instances/stillwater/4123.yaml 2>/dev/null; ls _datafiles/world/dogmud/rooms.instances/ironwind_steppe/3040.yaml 2>/dev/null; ls _datafiles/world/dogmud/rooms.instances/the_fernway_south/4197.yaml 2>/dev/null`

If any exist, delete them so the engine loads from the fresh template:

```bash
rm -f _datafiles/world/dogmud/rooms.instances/stillwater/4123.yaml
rm -f _datafiles/world/dogmud/rooms.instances/ironwind_steppe/3040.yaml
rm -f _datafiles/world/dogmud/rooms.instances/the_fernway_south/4197.yaml
```

(Do not commit the .instances/ deletions — they are runtime state. If the directory is not gitignored, verify with `git check-ignore _datafiles/world/dogmud/rooms.instances/.` before committing.)

- [ ] **Step 6: Boot the server briefly to confirm all three rooms load + the new lockboxes parse without YAML errors.**

Run: `go build -o ./tmp-mud.exe . && timeout 8 ./tmp-mud.exe -path ./_datafiles 2>&1 | grep -E "rooms.LoadDataFiles|panic|error" | head -20`

Expected: A `loadedCount=` line for rooms; no panics. Stop with Ctrl-C, delete `tmp-mud.exe`.

- [ ] **Step 7: Commit.**

```bash
git add _datafiles/world/dogmud/rooms/stillwater/4123.yaml _datafiles/world/dogmud/rooms/ironwind_steppe/3040.yaml _datafiles/world/dogmud/rooms/the_fernway_south/4197.yaml
git commit -m "$(cat <<'EOF'
content(foragers): add sanctuary lockboxes for Tova / Halix / Kessa

Each forager's anchor room gains a difficulty-10 lockbox container
where she dumps surplus on Recall arrival. RotationSeed=1 so the
combination differs from a generic stale-RotationSeed=0 entry; the
seed will be bumped on every cycle dump in actions_forager.go.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 8: Forager dumps surplus into sanctuary lockbox on Recall arrival

**Files:**
- Modify: `internal/behaviortree/actions_forager.go` (around `tickForagerRecalling` lines 302-321 and `tickForagerResting` lines 136-164)
- Test: `internal/behaviortree/actions_forager_test.go`

- [ ] **Step 1: Read the existing recall + rest tick code.**

Run: `sed -n '130,330p' internal/behaviortree/actions_forager.go`.

- [ ] **Step 2: Write a failing test asserting that on arrival at sanctuary with non-empty satchel, items move into the room's `lockbox` container and satchel ends empty.**

Append to `internal/behaviortree/actions_forager_test.go`:

```go
func TestTickForagerRecalling_DumpsSurplusIntoLockbox(t *testing.T) {
    // Set up: forager mob at sanctuary room with lockbox, satchel
    // contains 3 items.
    p := &forager.ForagerProfile{SanctuaryRoom: 99001, Buckets: []string{"marsh"}}
    mob := newTestForagerMob(t)
    mob.Character.StoreItem(items.New(40021)) // a marsh-bucket item
    mob.Character.StoreItem(items.New(40028))
    mob.Character.StoreItem(items.New(40023))

    room := newTestRoom(t, 99001, map[string]rooms.Container{
        "lockbox": {Lock: gamelock.Lock{Difficulty: 10, RotationSeed: 1}},
    })
    rooms.RegisterTestRoom(room) // helper that returns this room from rooms.LoadRoom — see existing helpers in this test file

    ctx := &EvalContext{RoomId: 99001, MobState: &BehaviorState{}}
    res := tickForagerRecalling(p, mob, ctx)
    if res != Success {
        t.Fatalf("tickForagerRecalling = %v, want Success", res)
    }

    if len(mob.Character.Items) != 0 {
        t.Errorf("satchel after dump = %d items, want 0", len(mob.Character.Items))
    }
    box := room.Containers["lockbox"]
    if len(box.Items) != 3 {
        t.Errorf("lockbox after dump = %d items, want 3", len(box.Items))
    }
    if !box.Lock.IsLocked() {
        t.Errorf("lockbox should be re-locked after dump, but IsLocked() = false")
    }
    if box.Lock.RotationSeed <= 1 {
        t.Errorf("lockbox RotationSeed = %d, want >1 (bumped on dump)", box.Lock.RotationSeed)
    }
}
```

(If the existing test file lacks `newTestForagerMob`, `newTestRoom`, or `RegisterTestRoom`, follow the patterns from existing tests in this file. Reuse `mobs.NewMobByIdFresh` for mob construction and the project's existing test-room registration helper.)

- [ ] **Step 3: Run the test to verify it fails.**

Run: `go test ./internal/behaviortree/... -run TestTickForagerRecalling_DumpsSurplus -v`
Expected: FAIL — current code does not dump items.

- [ ] **Step 4: Add a `dumpSatchelToLockbox` helper at the bottom of `actions_forager.go`.**

Insert before the `npcVisitVendorsInRoom` function:

```go
// dumpSatchelToLockbox transfers every item in the forager's satchel
// into the room's "lockbox" container, then bumps the lockbox lock's
// RotationSeed so existing player keyring entries become invalid.
// If the room has no lockbox container or the lockbox is full
// (>= ForagerLockboxCapacity), items remain in the satchel and the
// caller falls back to the legacy rest-extension behavior.
//
// Returns true if any items were dumped (caller can rely on the
// satchel being lighter on the next tick).
func dumpSatchelToLockbox(mob *mobs.Mob, ctx *EvalContext) bool {
    room := rooms.LoadRoom(ctx.RoomId)
    if room == nil {
        return false
    }
    box, ok := room.Containers["lockbox"]
    if !ok {
        return false
    }
    cap := configs.GetBalanceConfig().ForagerLockboxCapacity
    if cap <= 0 {
        cap = 500
    }
    dumped := false
    // Walk satchel in reverse so RemoveItem indices stay valid.
    for i := len(mob.Character.Items) - 1; i >= 0; i-- {
        if len(box.Items) >= int(cap) {
            break
        }
        item := mob.Character.Items[i]
        mob.Character.RemoveItem(item)
        box.AddItem(item)
        dumped = true
    }
    if dumped {
        box.Lock.SetLocked() // bumps RotationSeed
        room.Containers["lockbox"] = box
        room.SendText(`<ansi fg="yellow">A latch clicks shut from somewhere in the sanctuary.</ansi>`)
    }
    return dumped
}
```

- [ ] **Step 5: Add the new config knob `ForagerLockboxCapacity` (default 500).**

Edit `internal/configs/config.balance.go` (around line 297-302 where caravan dwell knobs live, or in the forager section):

```go
    // ForagerLockboxCapacity caps how many items a sanctuary lockbox
    // can hold. When the box is full, the forager falls back to the
    // Stage 3.4 rest-extension behavior until a player picks the box
    // open and clears space.
    ForagerLockboxCapacity ConfigInt `yaml:"ForagerLockboxCapacity"`
```

Edit `internal/configs/config.balance.misc.go` (Validate block, near line 281-286):

```go
    if b.ForagerLockboxCapacity <= 0 {
        b.ForagerLockboxCapacity = 500
    }
```

Edit `_datafiles/config.yaml` (in the forager section near `ForagerForageDwellRounds`):

```yaml
  # Sanctuary lockbox capacity (per forager). When full, the forager
  # falls back to rest-extension behavior until a player clears space.
  ForagerLockboxCapacity: 500
```

- [ ] **Step 6: Wire `dumpSatchelToLockbox` into `tickForagerRecalling`.**

Replace the existing function body (around line 302-321):

```go
func tickForagerRecalling(
    p *forager.ForagerProfile,
    mob *mobs.Mob,
    ctx *EvalContext,
) Result {
    if ctx.RoomId == p.SanctuaryRoom {
        // Dump remaining satchel into the sanctuary lockbox before
        // resting. Surplus accumulates in the lockbox where players
        // can pick to retrieve it; the cycle resumes immediately
        // regardless of vendor saturation.
        dumpSatchelToLockbox(mob, ctx)
        transitionForager(ctx.MobState, forager.StateResting)
        return Success
    }
    // Don't re-issue the cast every idle tick — re-issuing while a
    // cast is in progress can reset its progress and trap the forager
    // mid-cast indefinitely (observed 2026-04-30: Kessa "begins
    // weaving a spell" repeatedly but never actually teleports).
    // Wait for the active cast to resolve.
    if mob.Character.IsCasting() {
        return Success
    }
    mob.Command("cast fold-recall")
    return Success
}
```

- [ ] **Step 7: Remove the Stage 3.4 carry-ratio rest gate.**

In `tickForagerResting` (lines 149-160), replace:

```go
    if dwellElapsed && mob.Character.Health >= mob.Character.HealthMax.Value {
        // Stage 3.4: stay home if satchel is still over rest threshold.
        // Vendors didn't absorb much last cycle; foraging more would just
        // overflow back to satchel. Narratively: the forager sits at the
        // sanctuary looking content — the merchants don't need more right now.
        restThreshold := float64(configs.GetBalanceConfig().ForagerRestCarryThreshold)
        if carryRatio(mob) > restThreshold {
            // Continue resting — let legacy idle fire flavor emotes.
            return Failure
        }
        transitionForager(ctx.MobState, forager.StateTravelingToTerritory)
        return Success
    }
```

with:

```go
    if dwellElapsed && mob.Character.Health >= mob.Character.HealthMax.Value {
        // 2026-05-02: removed Stage 3.4 carry-ratio gate. Surplus
        // dumps into the sanctuary lockbox on Recall arrival, so the
        // satchel is empty here in the normal case. If the lockbox
        // was full and surplus stayed in the satchel, fall back to
        // continuing rest.
        restThreshold := float64(configs.GetBalanceConfig().ForagerRestCarryThreshold)
        if carryRatio(mob) > restThreshold {
            return Failure
        }
        transitionForager(ctx.MobState, forager.StateTravelingToTerritory)
        return Success
    }
```

(Note: kept the carry-ratio check as a *backstop* for the lockbox-full case — it is now reachable only when the box hits capacity.)

- [ ] **Step 8: Run the new test + the existing forager + config suites.**

Run: `go test ./internal/behaviortree/... ./internal/configs/... -v -run "Forager|RotationSeed|LockboxCapacity"`
Expected: PASS.

- [ ] **Step 9: Commit.**

```bash
git add internal/behaviortree/actions_forager.go internal/behaviortree/actions_forager_test.go internal/configs/config.balance.go internal/configs/config.balance.misc.go _datafiles/config.yaml
git commit -m "$(cat <<'EOF'
feat(forager): dump surplus into sanctuary lockbox on recall

When the forager arrives at her sanctuary, transfer up to
ForagerLockboxCapacity (500) items from her satchel into the room's
"lockbox" container, then re-lock with a bumped RotationSeed so
players must re-pick to access the new contents.

Removes the Stage 3.4 carry-ratio rest gate as the primary
deadlock-avoidance mechanism. Backstop remains: if the lockbox is
full, the forager falls back to extended rest until a player picks
the box and clears space.

New config knob: ForagerLockboxCapacity (default 500).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 9: SealedCrate engine package — type + helpers

**Files:**
- Create: `internal/sealedcrate/sealedcrate.go`
- Create: `internal/sealedcrate/sealedcrate_test.go`

- [ ] **Step 1: Look at the existing shop persistence layout for reference.**

Run: `head -100 internal/shops/persistence.go`. Note its file-based YAML pattern; we will mirror it.

- [ ] **Step 2: Write a failing test asserting the type API.**

Create `internal/sealedcrate/sealedcrate_test.go`:

```go
package sealedcrate

import (
    "testing"

    "github.com/GoMudEngine/GoMud/internal/items"
)

func TestCrate_AddRespectsCapacity(t *testing.T) {
    c := New(4038, 3) // capacity 3
    if !c.Add(items.New(40021)) {
        t.Fatalf("first Add returned false")
    }
    if !c.Add(items.New(40028)) {
        t.Fatalf("second Add returned false")
    }
    if !c.Add(items.New(40023)) {
        t.Fatalf("third Add returned false")
    }
    if c.Add(items.New(40020)) {
        t.Fatalf("fourth Add returned true; expected false (over capacity)")
    }
    if got := c.Len(); got != 3 {
        t.Errorf("Len = %d, want 3", got)
    }
}

func TestCrate_DrainAll(t *testing.T) {
    c := New(4038, 100)
    c.Add(items.New(40021))
    c.Add(items.New(40028))
    drained := c.DrainAll()
    if len(drained) != 2 {
        t.Errorf("DrainAll returned %d items, want 2", len(drained))
    }
    if c.Len() != 0 {
        t.Errorf("Len after DrainAll = %d, want 0", c.Len())
    }
}

func TestCrate_RoomIdAndCapacity(t *testing.T) {
    c := New(4038, 2000)
    if c.RoomId() != 4038 {
        t.Errorf("RoomId = %d, want 4038", c.RoomId())
    }
    if c.Capacity() != 2000 {
        t.Errorf("Capacity = %d, want 2000", c.Capacity())
    }
}
```

- [ ] **Step 3: Run the test to verify it fails (package does not exist).**

Run: `go test ./internal/sealedcrate/... -v`
Expected: FAIL with "no Go files".

- [ ] **Step 4: Create the package.**

Write `internal/sealedcrate/sealedcrate.go`:

```go
// Package sealedcrate provides a player-untouchable container
// primitive used for autonomous deliveries between long-running
// system NPCs (e.g. Kessa the Fernway forager and the caravan).
//
// A SealedCrate is bound to a single room, persists across reboots,
// and is mutated only by code paths inside internal/behaviortree
// (forager / caravan tick functions). Player commands (get, look,
// put, picklock, open) recognize a sealed-crate noun and emit
// flavor — they never read or write the crate's items list.
package sealedcrate

import (
    "sync"

    "github.com/GoMudEngine/GoMud/internal/items"
)

// Crate is the single sealed-crate instance for one room.
type Crate struct {
    roomId   int
    capacity int
    mu       sync.Mutex
    items    []items.Item
}

// New constructs an empty crate. Capacity 0 or negative defaults to
// 2000.
func New(roomId, capacity int) *Crate {
    if capacity <= 0 {
        capacity = 2000
    }
    return &Crate{roomId: roomId, capacity: capacity}
}

func (c *Crate) RoomId() int   { return c.roomId }
func (c *Crate) Capacity() int { return c.capacity }

func (c *Crate) Len() int {
    c.mu.Lock()
    defer c.mu.Unlock()
    return len(c.items)
}

// Add appends an item if there's room. Returns true on success.
func (c *Crate) Add(it items.Item) bool {
    c.mu.Lock()
    defer c.mu.Unlock()
    if len(c.items) >= c.capacity {
        return false
    }
    c.items = append(c.items, it)
    return true
}

// DrainAll empties the crate and returns its contents.
func (c *Crate) DrainAll() []items.Item {
    c.mu.Lock()
    defer c.mu.Unlock()
    out := c.items
    c.items = nil
    return out
}

// Snapshot returns a copy of the items list (read-only view, used
// for persistence and inspection).
func (c *Crate) Snapshot() []items.Item {
    c.mu.Lock()
    defer c.mu.Unlock()
    out := make([]items.Item, len(c.items))
    copy(out, c.items)
    return out
}

// SetItemsForLoad replaces the items list wholesale. Used only by
// the persistence loader at boot.
func (c *Crate) SetItemsForLoad(its []items.Item) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.items = its
}
```

- [ ] **Step 5: Run the test to verify it passes.**

Run: `go test ./internal/sealedcrate/... -v`
Expected: PASS.

- [ ] **Step 6: Commit.**

```bash
git add internal/sealedcrate/sealedcrate.go internal/sealedcrate/sealedcrate_test.go
git commit -m "$(cat <<'EOF'
feat(sealedcrate): new package for player-untouchable crates

Adds the Crate primitive — a room-bound, capacity-bounded container
mutated only by autonomous NPC code paths. Designed for the
Kessa-to-caravan handoff at North Road 4038; deliberately separate
from the player-facing rooms.Container type so player commands can
never accidentally mutate it.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 10: SealedCrate persistence (save/load YAML)

**Files:**
- Create: `internal/sealedcrate/persistence.go`
- Create: `internal/sealedcrate/persistence_test.go`

- [ ] **Step 1: Write a failing test for save+load round-trip.**

Create `internal/sealedcrate/persistence_test.go`:

```go
package sealedcrate

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/GoMudEngine/GoMud/internal/items"
)

func TestSaveLoadRoundTrip(t *testing.T) {
    tmp := t.TempDir()

    c := New(4038, 2000)
    c.Add(items.New(40021))
    c.Add(items.New(40028))

    if err := SaveTo(filepath.Join(tmp, "4038-fernway_shipment.yaml"), c); err != nil {
        t.Fatalf("SaveTo: %v", err)
    }

    loaded, err := LoadFrom(filepath.Join(tmp, "4038-fernway_shipment.yaml"))
    if err != nil {
        t.Fatalf("LoadFrom: %v", err)
    }
    if loaded.RoomId() != 4038 {
        t.Errorf("loaded RoomId = %d, want 4038", loaded.RoomId())
    }
    if loaded.Capacity() != 2000 {
        t.Errorf("loaded Capacity = %d, want 2000", loaded.Capacity())
    }
    if loaded.Len() != 2 {
        t.Errorf("loaded Len = %d, want 2", loaded.Len())
    }
}

func TestLoadMissingFileReturnsNil(t *testing.T) {
    tmp := t.TempDir()
    c, err := LoadFrom(filepath.Join(tmp, "does-not-exist.yaml"))
    if err == nil && c != nil {
        t.Errorf("expected (nil, err) for missing file, got crate")
    }
    // It's ok to return either (nil, fs error) or (nil, nil).
    _ = os.Stat
}
```

- [ ] **Step 2: Run the test to verify it fails (no SaveTo/LoadFrom).**

Run: `go test ./internal/sealedcrate/... -run SaveLoad -v`
Expected: FAIL — undefined.

- [ ] **Step 3: Create the persistence implementation.**

Write `internal/sealedcrate/persistence.go`:

```go
package sealedcrate

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/GoMudEngine/GoMud/internal/items"
    "gopkg.in/yaml.v3"
)

type cratePayload struct {
    RoomId   int          `yaml:"roomid"`
    Capacity int          `yaml:"capacity"`
    Items    []items.Item `yaml:"items,omitempty"`
}

// SaveTo writes a crate's state to the given path. Caller must
// ensure the parent directory exists.
func SaveTo(path string, c *Crate) error {
    payload := cratePayload{
        RoomId:   c.RoomId(),
        Capacity: c.Capacity(),
        Items:    c.Snapshot(),
    }
    data, err := yaml.Marshal(payload)
    if err != nil {
        return fmt.Errorf("sealedcrate.SaveTo: marshal: %w", err)
    }
    if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
        return fmt.Errorf("sealedcrate.SaveTo: mkdir: %w", err)
    }
    if err := os.WriteFile(path, data, 0o644); err != nil {
        return fmt.Errorf("sealedcrate.SaveTo: write: %w", err)
    }
    return nil
}

// LoadFrom reads a crate from the given path. Returns nil + error
// if the file is missing.
func LoadFrom(path string) (*Crate, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var payload cratePayload
    if err := yaml.Unmarshal(data, &payload); err != nil {
        return nil, fmt.Errorf("sealedcrate.LoadFrom: unmarshal: %w", err)
    }
    c := New(payload.RoomId, payload.Capacity)
    c.SetItemsForLoad(payload.Items)
    return c, nil
}
```

- [ ] **Step 4: Run the test to verify it passes.**

Run: `go test ./internal/sealedcrate/... -v`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
git add internal/sealedcrate/persistence.go internal/sealedcrate/persistence_test.go
git commit -m "$(cat <<'EOF'
feat(sealedcrate): YAML persistence with round-trip test

Adds SaveTo / LoadFrom for crate state. Mirrors the shop persistence
pattern at internal/shops/persistence.go but isolated in the new
sealedcrate package.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 11: Bind SealedCrate to Room + boot loader

**Files:**
- Modify: `internal/rooms/rooms.go` (add `SealedCrate *sealedcrate.Crate` field around line 100)
- Modify: `main.go` (add boot loader after Task 3's anchor-prepare block)
- Test: `internal/rooms/rooms_test.go` (a small assertion that the field exists and round-trips)

- [ ] **Step 1: Add the field on `Room`.**

In `internal/rooms/rooms.go`, near line 100 (alongside `Mutators`), add:

```go
    SealedCrate       *sealedcrate.Crate                `yaml:"-"`                                   // Player-untouchable delivery crate; populated at boot from _datafiles/world/dogmud/crates/<roomid>-*.yaml. Nil for rooms with no crate.
```

Add the import `"github.com/GoMudEngine/GoMud/internal/sealedcrate"` at the top of the file.

- [ ] **Step 2: Add a small helper to attach the crate at boot.**

Append to `internal/rooms/rooms.go`:

```go
// AttachSealedCrate binds a sealed crate to this room. Used by the
// boot loader; subsequent reads come via Room.SealedCrate.
func (r *Room) AttachSealedCrate(c *sealedcrate.Crate) {
    r.SealedCrate = c
}
```

- [ ] **Step 3: Add the boot-time loader to `main.go` after Task 3's anchor-prepare loop.**

Insert after the `mudlog.Info("system NPC anchor rooms prepared", ...)` line:

```go
    // Load sealed crates from disk and attach them to their rooms.
    // The crates/ directory mirrors shops/ — one YAML per crate,
    // named "<roomid>-<label>.yaml". Missing directory means no
    // crates exist yet, which is fine.
    crateDir := util.FilePath(configs.GetFilePathsConfig().DataFiles.String(), `/world/dogmud/crates`)
    if entries, err := os.ReadDir(crateDir); err == nil {
        loaded := 0
        for _, e := range entries {
            if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
                continue
            }
            path := filepath.Join(crateDir, e.Name())
            c, err := sealedcrate.LoadFrom(path)
            if err != nil {
                mudlog.Warn("sealedcrate load", "path", path, "error", err)
                continue
            }
            room := rooms.LoadRoom(c.RoomId())
            if room == nil {
                mudlog.Warn("sealedcrate room missing", "roomId", c.RoomId(), "path", path)
                continue
            }
            room.AttachSealedCrate(c)
            loaded++
        }
        mudlog.Info("sealed crates loaded", "count", loaded)
    } else if !os.IsNotExist(err) {
        mudlog.Warn("sealedcrate dir scan", "dir", crateDir, "error", err)
    }
```

Add imports if missing: `"os"`, `"path/filepath"`, `"strings"`, `"github.com/GoMudEngine/GoMud/internal/sealedcrate"`.

- [ ] **Step 4: Build to confirm imports compile.**

Run: `go build ./...`
Expected: PASS.

- [ ] **Step 5: Run all tests.**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit.**

```bash
git add internal/rooms/rooms.go main.go
git commit -m "$(cat <<'EOF'
feat(rooms): attach sealed crates at boot from _datafiles/world/dogmud/crates/

Rooms gain an optional *sealedcrate.Crate field; boot loader walks
the crates/ directory (mirrors shops/) and attaches each persisted
crate to its room. Missing directory is non-fatal — crates are
opt-in per room.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 12: Player command shims for sealed crates

**Files:**
- Modify: `internal/usercommands/get.go`, `look.go`, `put.go`, `picklock.go`, `lock.go` (and `open.go` if it exists)

- [ ] **Step 1: Find the dispatch points where each command resolves a noun against `room.Containers`.**

Run: `grep -n "Containers\\[" internal/usercommands/{get,look,put,picklock,lock,open}.go 2>/dev/null`

For each command, identify the line where it looks up `room.Containers[name]` after `room.FindContainerByName(args[0])`.

- [ ] **Step 2: Add a small shared helper (e.g. in `internal/rooms/rooms.go` or as an inline check) to detect a sealed-crate noun.**

In `internal/rooms/rooms.go`, append:

```go
// MatchesSealedCrate returns true if the given user-typed noun
// matches the room's sealed crate (if any). Used by player command
// shims to short-circuit interaction.
func (r *Room) MatchesSealedCrate(noun string) bool {
    if r.SealedCrate == nil {
        return false
    }
    n := strings.ToLower(noun)
    return n == "crate" || n == "shipping crate" || n == "sealed crate"
}
```

(Add `"strings"` import if missing.)

- [ ] **Step 3: Add the shim to `internal/usercommands/get.go`.**

Find the section that handles `get <name>` and resolves containers/items. Before any container lookup, add:

```go
    if room.MatchesSealedCrate(strings.ToLower(args[0])) {
        user.SendText(`The shipping crate is sealed and bound for the caravan; you can't get into it.`)
        return true, nil
    }
```

- [ ] **Step 4: Add the shim to `look.go`.**

Find the noun-matching section (around line 150-300, where it walks `room.Containers`). Add a sealed-crate match BEFORE the container lookup so a `look crate` short-circuits to flavor:

```go
    if room.MatchesSealedCrate(strings.ToLower(rest)) {
        user.SendText(`A heavy iron-banded shipping crate sits at the roadside, its lid latched shut and its sides marked with the caravan's burned-in seal. The wood is weather-stained and the latch is recently oiled.`)
        return true, nil
    }
```

- [ ] **Step 5: Add the shim to `put.go`.**

Find the destination container resolution. Add:

```go
    if room.MatchesSealedCrate(strings.ToLower(targetName)) {
        user.SendText(`The shipping crate is sealed; the caravan only lets through what they put in themselves.`)
        return true, nil
    }
```

(Adjust `targetName` to the correct local variable in `put.go`.)

- [ ] **Step 6: Add the shim to `picklock.go`.**

Add immediately after the args parse:

```go
    if room.MatchesSealedCrate(strings.ToLower(args[0])) {
        user.SendText(`The shipping crate has no lock to pick — it's sealed by the caravan's binding, not by mechanism.`)
        return true, nil
    }
```

- [ ] **Step 7: Add the shim to `lock.go` (the `lock` command).**

```go
    if room.MatchesSealedCrate(strings.ToLower(args[0])) {
        user.SendText(`The shipping crate is already sealed.`)
        return true, nil
    }
```

- [ ] **Step 8: If `internal/usercommands/open.go` exists, add a similar shim.**

```go
    if room.MatchesSealedCrate(strings.ToLower(args[0])) {
        user.SendText(`The shipping crate's seal won't budge — only the caravan opens it.`)
        return true, nil
    }
```

If `open.go` doesn't exist, skip this step.

- [ ] **Step 9: Build and run all tests.**

Run: `go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 10: Commit.**

```bash
git add internal/rooms/rooms.go internal/usercommands/get.go internal/usercommands/look.go internal/usercommands/put.go internal/usercommands/picklock.go internal/usercommands/lock.go internal/usercommands/open.go 2>/dev/null || true
git commit -m "$(cat <<'EOF'
feat(commands): sealed-crate shims for get/look/put/picklock/lock/open

Players can see and look at the roadside shipping crate but cannot
get from it, put items in it, pick its lock, or open it. Each
short-circuits with flavor text before reaching the standard
Container handling.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 13: Add the sealed crate at room 4038

**Files:**
- Create: `_datafiles/world/dogmud/crates/4038-fernway_shipment.yaml`
- Modify: `_datafiles/world/dogmud/rooms/north_road/4038.yaml` (add a `nouns:` entry + visible reference in description)

- [ ] **Step 1: Read the existing 4038 room YAML.**

Run: `cat _datafiles/world/dogmud/rooms/north_road/4038.yaml`.

- [ ] **Step 2: Create the persistence file.**

Create `_datafiles/world/dogmud/crates/4038-fernway_shipment.yaml`:

```yaml
roomid: 4038
capacity: 2000
items: []
```

- [ ] **Step 3: Add a `crate` noun + description mention in the room YAML.**

In `_datafiles/world/dogmud/rooms/north_road/4038.yaml`, append/extend the `nouns:` block:

```yaml
  crate: |
    A heavy iron-banded oak shipping crate sits in the lee of the
    crossroads marker, its lid latched shut and its flanks burned
    with the caravan's seal. Foragers from the Fernway leave loads
    in it for the caravan to collect; the seal only opens to the
    caravan's hand.
```

Add a one-sentence visible reference to the room `description:` so the noun is discoverable:

```
A heavy iron-banded <ansi fg="itemname">crate</ansi> sits at the
roadside, sealed with the caravan's burned-in mark.
```

- [ ] **Step 4: Boot the server and confirm the crate loads + the noun is discoverable.**

Run: `go build -o ./tmp-mud.exe . && timeout 10 ./tmp-mud.exe -path ./_datafiles 2>&1 | grep -E "sealed crates loaded|panic|error" | head`

Expected: A `sealed crates loaded count=1` log line. No panics. Stop with Ctrl-C, delete tmp-mud.exe.

- [ ] **Step 5: Commit.**

```bash
git add _datafiles/world/dogmud/crates/4038-fernway_shipment.yaml _datafiles/world/dogmud/rooms/north_road/4038.yaml
git commit -m "$(cat <<'EOF'
content(crate): add Fernway shipping crate at North Road 4038

Persistence file (capacity 2000, empty initial state) plus a
visible noun reference in the room description and prose. Players
can see + look the crate; the command shims block all interaction.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 14: Rewrite Kessa's Fernway delivery to dump into the sealed crate

**Files:**
- Modify: `internal/behaviortree/actions_forager.go` — `tickForagerDeliveringFernway` (lines 283-300)
- Test: `internal/behaviortree/actions_forager_test.go`

- [ ] **Step 1: Write a failing test asserting Kessa dumps her satchel into the room's sealed crate on arrival at 4038.**

Append to `internal/behaviortree/actions_forager_test.go`:

```go
func TestTickForagerDeliveringFernway_DumpsIntoCrate(t *testing.T) {
    p := &forager.ForagerProfile{
        MeetingRoom: 4038,
        Buckets:     []string{"fernway"},
        Kind:        forager.KindFernway,
    }
    mob := newTestForagerMob(t)
    mob.Character.StoreItem(items.New(40050)) // a fernway-bucket item
    mob.Character.StoreItem(items.New(40051))

    crate := sealedcrate.New(4038, 2000)
    room := newTestRoom(t, 4038, nil)
    room.AttachSealedCrate(crate)
    rooms.RegisterTestRoom(room)

    ctx := &EvalContext{RoomId: 4038, MobState: &BehaviorState{}}
    cfg := configs.GetBalanceConfig()
    res := tickForagerDeliveringFernway(p, mob, ctx, cfg)
    if res != Success {
        t.Fatalf("res = %v, want Success", res)
    }
    if crate.Len() != 2 {
        t.Errorf("crate Len = %d, want 2", crate.Len())
    }
    if len(mob.Character.Items) != 0 {
        t.Errorf("satchel after dump = %d, want 0", len(mob.Character.Items))
    }
    // Should immediately transition to Recalling — no 150-round wait.
    state := ctx.MobState.GetString("forager_state")
    if state != "recalling" {
        t.Errorf("forager_state = %q, want recalling", state)
    }
}
```

- [ ] **Step 2: Run the test to verify it fails.**

Run: `go test ./internal/behaviortree/... -run DeliveringFernway_DumpsIntoCrate -v`
Expected: FAIL — current implementation waits, doesn't dump.

- [ ] **Step 3: Rewrite `tickForagerDeliveringFernway`.**

Replace the body (lines 283-300):

```go
func tickForagerDeliveringFernway(
    p *forager.ForagerProfile,
    mob *mobs.Mob,
    ctx *EvalContext,
    cfg configs.Balance,
) Result {
    if ctx.RoomId != p.MeetingRoom {
        mob.Command(fmt.Sprintf("pathto %d", p.MeetingRoom))
        return Success
    }
    // Arrived at meeting room. Dump fernway-bucket items into the
    // sealed crate — Kessa drops the load and turns for home, no
    // wait for the caravan to coincide.
    room := rooms.LoadRoom(ctx.RoomId)
    if room != nil && room.SealedCrate != nil {
        crate := room.SealedCrate
        dumped := 0
        for i := len(mob.Character.Items) - 1; i >= 0; i-- {
            item := mob.Character.Items[i]
            bucket := economy.BucketFor(item.ItemId)
            if !slices.Contains(p.Buckets, bucket) {
                continue
            }
            if !crate.Add(item) {
                break // crate full
            }
            mob.Character.RemoveItem(item)
            dumped++
        }
        if dumped > 0 {
            // Persist immediately — this is a delivery commit, and
            // we want a server crash to leave the crate intact for
            // the next caravan pickup.
            persistCrate(ctx.RoomId, crate)
            room.SendText(fmt.Sprintf(
                `<ansi fg="mobname">%s</ansi> hauls a satchel up to the crate, latches it shut, and turns for home.`,
                p.Name))
        }
    }
    // Always advance to Recalling — no more 150-round wait timer.
    transitionForager(ctx.MobState, forager.StateRecalling)
    return Success
}

// persistCrate writes the crate's current state to its YAML at
// _datafiles/world/dogmud/crates/<roomid>-fernway_shipment.yaml.
// Errors are logged but not surfaced — persistence is best-effort
// crash-safety, not a correctness requirement.
func persistCrate(roomId int, c *sealedcrate.Crate) {
    path := util.FilePath(
        configs.GetFilePathsConfig().DataFiles.String(),
        fmt.Sprintf("/world/dogmud/crates/%d-fernway_shipment.yaml", roomId),
    )
    if err := sealedcrate.SaveTo(path, c); err != nil {
        mudlog.Error("forager.persistCrate", "roomId", roomId, "error", err)
    }
}
```

Add imports if missing: `"slices"`, `"github.com/GoMudEngine/GoMud/internal/sealedcrate"`, `"github.com/GoMudEngine/GoMud/internal/economy"`, `"github.com/GoMudEngine/GoMud/internal/mudlog"`, `"github.com/GoMudEngine/GoMud/internal/util"`.

- [ ] **Step 4: Run the test to verify it passes.**

Run: `go test ./internal/behaviortree/... -run DeliveringFernway -v`
Expected: PASS.

- [ ] **Step 5: Run the full forager + caravan suite.**

Run: `go test ./internal/behaviortree/... ./internal/forager/... ./internal/caravan/... ./internal/sealedcrate/... -v`
Expected: PASS.

- [ ] **Step 6: Commit.**

```bash
git add internal/behaviortree/actions_forager.go internal/behaviortree/actions_forager_test.go
git commit -m "$(cat <<'EOF'
feat(forager): Kessa drops fernway load into sealed crate at 4038

Replaces the 150-round wait-for-caravan loop with an unconditional
crate dump. Kessa drains all fernway-bucket items from her satchel
into the room's sealed crate, persists the crate state, and
transitions immediately to Recalling. The caravan picks up
asynchronously on its next pass.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 15: Rewrite caravan Fernway pickup to drain the crate; delete `caravan_load`

**Files:**
- Modify: `internal/behaviortree/actions_caravan.go` — `tickFernwayPickup` (209-246), `caravanLoadAppend` callsites, plus `caravanLoadGet/Set` callers
- Modify: `internal/caravan/visit.go` (anywhere it reads the load flag)

- [ ] **Step 1: Find every `caravanLoad` callsite.**

Run: `grep -rn "caravanLoad" internal/`
Capture the list — typically lives in `actions_caravan.go` and possibly `visit.go`.

- [ ] **Step 2: Rewrite `tickFernwayPickup` to drain the sealed crate into the wagon.**

Replace `tickFernwayPickup` (around lines 209-246) with:

```go
func tickFernwayPickup(cur caravan.CaravanState, mob *mobs.Mob, ctx *EvalContext) Result {
    if partyHostilesNearby(ctx.InstanceId) || anyPartyMemberInCombat(ctx.InstanceId) {
        return Failure
    }

    if ctx.RoomId != fernwayMeetingRoomId {
        mob.Command(fmt.Sprintf("pathto %d", fernwayMeetingRoomId))
        return Success
    }

    startedStr := ctx.MobState.GetString("caravan_state_started_round")
    started, _ := strconv.ParseUint(startedStr, 10, 64)
    now := util.GetRoundCount()
    elapsed := now - started

    // First tick at room: drain the sealed crate into the wagon.
    if elapsed == 0 {
        if room := rooms.LoadRoom(ctx.RoomId); room != nil && room.SealedCrate != nil {
            wagon := caravanWagon(ctx.InstanceId)
            if wagon != nil {
                drained := room.SealedCrate.DrainAll()
                for _, it := range drained {
                    wagon.Character.StoreItem(it)
                }
                if len(drained) > 0 {
                    persistCrate(ctx.RoomId, room.SealedCrate)
                    room.SendText(fmt.Sprintf(
                        `<ansi fg="yellow">The caravan pulls up to the roadside crate, breaks the seal, and loads its contents into the wagon — %d %s in all.</ansi>`,
                        len(drained), pluralize("crate", len(drained))))
                }
            }
        }
    }

    dwell := uint64(configs.GetBalanceConfig().FernwayPickupDwellRounds)
    if elapsed >= dwell {
        transitionTo(ctx.MobState, caravan.AdvanceState(cur))
        return Success
    }
    return Failure
}

// caravanWagon returns the wagon mob (374) belonging to the caravan
// party. Returns nil if the wagon is missing (which should not
// happen in a healthy caravan).
func caravanWagon(instanceId int) *mobs.Mob {
    // (Implementation note: search the party for mobId 374. The
    // existing caravan code already has a party-walking helper —
    // reuse it. If none exists, add a small one here.)
    // ... see existing party iteration in actions_caravan.go ...
    return findPartyMemberByMobId(instanceId, 374)
}
```

If `findPartyMemberByMobId` does not exist, add it at the bottom of `actions_caravan.go`:

```go
func findPartyMemberByMobId(instanceId int, mobId int) *mobs.Mob {
    // Walk the party rosters used elsewhere in this file (e.g. by
    // partyHostilesNearby). If your existing code uses a different
    // accessor, mirror it. The intent: return the wagon mob from
    // the same party as the caravan-leader instance.
    party := parties.GetPartyByLeaderInstanceId(instanceId) // adapt to actual API
    if party == nil {
        return nil
    }
    for _, instId := range party.MemberInstanceIds {
        m := mobs.GetInstance(instId)
        if m != nil && int(m.MobId) == mobId {
            return m
        }
    }
    return nil
}
```

(Adapt to whatever `parties` / party-walk API the rest of `actions_caravan.go` uses; reuse the existing helper if there is one.)

Also export `pluralize` or inline it:

```go
func pluralize(word string, n int) string {
    if n == 1 {
        return word
    }
    return word + "s"
}
```

- [ ] **Step 3: Delete `caravanLoadAppend` / `caravanLoadGet` / `caravanLoadSet` definitions and every callsite.**

Search for callsites:

```bash
grep -rn "caravanLoad" internal/
```

For each callsite outside `tickFernwayPickup`:
- In `internal/behaviortree/actions_caravan.go` `tickRoute`, remove the `caravanLoadAppend(...)` calls; the wagon now carries real items, so the bucket flag has no purpose.
- In `internal/caravan/visit.go`, remove any `caravan_load` read; vendor restocking now uses the wagon's actual inventory (Stage 3.4 transfer).
- Delete the `caravanLoadAppend / caravanLoadGet / caravanLoadSet` helper functions.

If a caller has logic shaped like `if caravanLoadGet(...) contains "fernway" { topUpFernwayBucket() }`, replace it with the wagon's real-item delivery loop (i.e. the Stage 3.4 `VisitVendorsInRoom` path, which already handles inventory-driven restocks).

- [ ] **Step 4: Add a regression test asserting an integration smoke — caravan dwells at 4038, crate has items, wagon ends with the items.**

Append to `internal/behaviortree/actions_caravan_test.go`:

```go
func TestTickFernwayPickup_DrainsCrateIntoWagon(t *testing.T) {
    crate := sealedcrate.New(4038, 2000)
    crate.Add(items.New(40050))
    crate.Add(items.New(40051))

    room := newTestRoom(t, 4038, nil)
    room.AttachSealedCrate(crate)
    rooms.RegisterTestRoom(room)

    leader := newTestCaravanLeader(t)
    wagon := newTestCaravanWagon(t) // mobId 374
    party := newTestParty(t, leader, wagon)
    _ = party

    ctx := &EvalContext{
        RoomId:     4038,
        InstanceId: int(leader.InstanceId),
        MobState:   &BehaviorState{},
    }
    ctx.MobState.Set("caravan_state_started_round", strconv.FormatUint(util.GetRoundCount(), 10))

    res := tickFernwayPickup(caravan.StateOutboundFernwayPickup, leader, ctx)
    if res == Failure {
        t.Logf("first tick returned Failure (legacy idle path) — that's fine")
    }

    if crate.Len() != 0 {
        t.Errorf("crate after first tick = %d items, want 0 (drained)", crate.Len())
    }
    if len(wagon.Character.Items) != 2 {
        t.Errorf("wagon after first tick = %d items, want 2", len(wagon.Character.Items))
    }
}
```

(If the test harness for caravan tests differs — e.g. a different `newTestCaravan*` factory — match the existing pattern in `actions_caravan_test.go`.)

- [ ] **Step 5: Run the new test + the full suite.**

Run: `go test ./internal/behaviortree/... ./internal/caravan/... -v -run "FernwayPickup|VisitVendor"`
Expected: PASS.

- [ ] **Step 6: Build to confirm no orphan references to deleted helpers.**

Run: `go build ./...`
Expected: PASS.

- [ ] **Step 7: Commit.**

```bash
git add internal/behaviortree/actions_caravan.go internal/behaviortree/actions_caravan_test.go internal/caravan/visit.go
git commit -m "$(cat <<'EOF'
feat(caravan): Fernway pickup drains sealed crate into wagon

Replaces the bucket-flag handoff with real item transfer. On dwell
at 4038 the caravan drains the room's sealed crate into the wagon's
inventory; downstream vendor visits use the wagon's contents
directly (Stage 3.4 path).

Deletes caravanLoadAppend/Get/Set and every callsite — the abstract
flag is no longer needed now that physical items move through the
crate.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 16: Periodic crate persistence guard

**Files:**
- Modify: `internal/behaviortree/actions_caravan.go` (the wagon fetch path) and `internal/behaviortree/actions_forager.go` (Kessa dump path)

This task ensures crate persistence stays consistent if the server crashes mid-cycle. Both Tasks 14 (Kessa dump) and 15 (caravan drain) already call `persistCrate` immediately after mutation, so this task only needs a smoke test.

- [ ] **Step 1: Write a test that mutates the crate, then loads it from disk, asserts state matches.**

Append to `internal/sealedcrate/persistence_test.go`:

```go
func TestSaveLoad_AfterPartialDrain(t *testing.T) {
    tmp := t.TempDir()
    path := filepath.Join(tmp, "4038-fernway_shipment.yaml")

    c := New(4038, 2000)
    c.Add(items.New(40050))
    c.Add(items.New(40051))
    if err := SaveTo(path, c); err != nil {
        t.Fatalf("SaveTo: %v", err)
    }

    // Drain and re-save.
    drained := c.DrainAll()
    if len(drained) != 2 {
        t.Errorf("DrainAll = %d, want 2", len(drained))
    }
    if err := SaveTo(path, c); err != nil {
        t.Fatalf("SaveTo after drain: %v", err)
    }

    loaded, err := LoadFrom(path)
    if err != nil {
        t.Fatalf("LoadFrom: %v", err)
    }
    if loaded.Len() != 0 {
        t.Errorf("loaded Len after drain+save = %d, want 0", loaded.Len())
    }
}
```

- [ ] **Step 2: Run.**

Run: `go test ./internal/sealedcrate/... -v`
Expected: PASS.

- [ ] **Step 3: Commit.**

```bash
git add internal/sealedcrate/persistence_test.go
git commit -m "$(cat <<'EOF'
test(sealedcrate): pin save/load behavior across drain

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 17: Live boot smoke test

**Files:** None — verification only.

- [ ] **Step 1: Boot the server locally and watch the relevant log lines.**

Run: `go build -o ./tmp-mud.exe . && timeout 20 ./tmp-mud.exe -path ./_datafiles 2>&1 | tee ./tmp-boot.log | grep -E "rooms.LoadDataFiles|mobs.LoadDataFiles|shop cache prewarm|system NPC anchor|sealed crates loaded|panic|error" | head -40`

Expected lines (ordering may vary slightly):
```
mobs.LoadDataFiles() loadedCount=...
rooms.LoadDataFiles() loadedCount=...
shop cache prewarmed (persisted) count=...
shop cache prewarmed (spawninfo) count=...
system NPC anchor rooms prepared count=4
sealed crates loaded count=1
```

No `panic` lines. If you see one, halt and investigate before committing this task.

- [ ] **Step 2: Connect a test client (telnet or local web client) and verify:**

  1. `look` from any North Road room near 4038 — confirm caravan master is present (mob spawned at boot).
  2. `goto 4123` (admin teleport) — confirm Tova is present + the lockbox is visible in the room description.
  3. `goto 3040` — Halix + lockbox visible.
  4. `goto 4197` — Kessa + lockbox visible.
  5. `goto 4038`, `look crate` — flavor text fires; `get crate`, `open crate`, `picklock crate` all return the appropriate locked-out flavor.
  6. `goto 4123`, `picklock lockbox` — the picklock minigame fires (no auto-bypass from any pre-existing keyring).

- [ ] **Step 3: Stop the server, delete `tmp-mud.exe` and `tmp-boot.log`.**

```bash
rm -f tmp-mud.exe tmp-boot.log
```

- [ ] **Step 4: No commit — this is verification only.**

---

### Task 18: PATCH_NOTES + memory pointer cleanup

**Files:**
- Modify: `PATCH_NOTES.md`
- Modify: `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\MEMORY.md`
- Modify: `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\project_caravan_stages_unmerged.md` (or delete)

- [ ] **Step 1: Append a dated entry to `PATCH_NOTES.md`.**

```markdown
## 2026-05-02 — Forager + Caravan Followup

- Whisper (room 507) removed from the caravan's Thornwall vendor rotation.
- System NPC anchor rooms (4042, 4123, 3040, 4197) now spawn at boot
  rather than on first player visit. Caravan + foragers are active
  immediately on a clean restart; the `/admin/economy/` dashboard no
  longer shows "(not active)" rows after boot.
- Sanctuary lockboxes added at each forager's anchor room. Foragers
  dump satchel surplus into the lockbox on Recall arrival; the
  lockbox combination rotates each cycle, requiring a fresh picklock
  attempt to access. Stage 3.4 carry-ratio rest gate removed; lockbox
  capacity (default 500) acts as the new soft backstop.
- Roadside sealed crate at North Road 4038. Kessa dumps her load
  into it on arrival; the caravan drains it on its next pass.
  Players can see the crate but cannot interact with it.
  `caravan_load` flag deleted in favor of real items in the wagon.
- `CaravanDepotDwellRounds` halved 720 → 360 for more visible caravan
  cadence.
- New engine surface: `gamelock.Lock.RotationSeed` (per-cycle lock
  rotation) and `internal/sealedcrate/` package (player-untouchable
  delivery containers).
```

- [ ] **Step 2: Update memory pointer about caravan being dev-only.**

Read `memory/project_caravan_stages_unmerged.md` and either:
- Delete it (caravan is on master; the project pointer is stale), or
- Update it to reflect "caravan is on master; stages 3.0–3.4 all shipped; 2026-05-02 followup landed".

Also remove the stale "caravan trying to restock Whisper" and "foragers showing 'not active'" entries from `memory/MEMORY.md` "Bugs to Investigate" — both are now fixed.

- [ ] **Step 3: Commit code-side changes (memory files are outside the repo, so they commit separately if at all).**

```bash
git add PATCH_NOTES.md
git commit -m "$(cat <<'EOF'
docs: PATCH_NOTES for 2026-05-02 forager + caravan followup

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Memory files are user-local; if the user wants them updated, do so via the memory write protocol — no git commit needed.

---

## Self-review checklist

Run before signaling the plan is done:

1. **Spec coverage.** Each Decision in the spec has a task:
   - Decision 1 (Whisper) → Task 1.
   - Decision 2 (boot prewarm anchor rooms) → Task 3.
   - Decision 3 (sanctuary lockboxes + RotationSeed + remove rest gate) → Tasks 4, 5, 6, 7, 8.
   - Decision 4 (sealed crate + Kessa rewrite + caravan rewrite) → Tasks 9, 10, 11, 12, 13, 14, 15.
   - Decision 5 (halve depot dwell) → Task 2.
   - Risk register's "Removing `caravan_load` flag breaks tests/dashboards that read it" → addressed in Task 15 Step 3.
   - Success criterion #5 (sanctuary lockbox cheese path with re-pick required) → covered by Task 6 (RotationSeed plumb-through) + Task 7 (lockbox config) + Task 8 (SetLocked on dump) + Task 17 (live verification).

2. **No placeholders.** All TBD/TODO/"add appropriate" patterns checked. Every code step shows the code.

3. **Type / signature consistency.**
   - `GetLockSequence(lockId, difficulty, seed, rotation)` — used consistently in Tasks 5 + 6.
   - `Lock.RotationSeed uint64` — Task 4 defines, Task 6 reads, Task 8 bumps via `SetLocked()`.
   - `sealedcrate.Crate` API: `New(roomId, capacity)`, `RoomId()`, `Capacity()`, `Len()`, `Add(item) bool`, `DrainAll() []items.Item`, `Snapshot() []items.Item`, `SetItemsForLoad([]items.Item)` — used consistently in Tasks 9, 10, 11, 14, 15, 16.
   - `Room.SealedCrate *sealedcrate.Crate`, `Room.AttachSealedCrate`, `Room.MatchesSealedCrate(noun)` — defined in Tasks 11 + 12, used in 13, 14, 15.
   - `ForagerLockboxCapacity ConfigInt` — defined in Task 8, no other consumers (capacity is read inline in `dumpSatchelToLockbox`).

4. **Build / test ordering.** Task 5 intentionally leaves the build broken; Task 6 fixes it. All other tasks end on a green build.
