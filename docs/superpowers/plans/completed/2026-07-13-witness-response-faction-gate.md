# Witness-Response Faction Gate Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop unaligned bystander NPCs (e.g. Drillmaster Vorn) from fleeing/seeking revenge when a player attacks a factionless target (training dummy, wildlife, monster), by gating the witness-response seeder on the victim's faction — the same rule the crime record already uses.

**Architecture:** The witness-response seeder `aggressiveActionToRevenge` (fires on `PlayerAttackedMob`) currently reacts to *every* mob in the room. Gate it on the attacked victim's registered factions: resolve the victim's faction ids via a new pure/injectable helper; if the victim is factionless, return early (no reaction); otherwise seed the victim plus only the same-faction, non-`AutoAggro` witnesses (reusing `crimes.WitnessesInRoom`). No new mob flag; no change to combat-flee or the crime record.

**Tech Stack:** Go. Packages: `internal/seeders`, `internal/crimes`, `internal/factions`, `internal/mobs`, `internal/rooms`.

Spec: `docs/superpowers/specs/2026-07-13-witness-response-faction-gate-design.md`

---

## File structure

- `internal/seeders/faction_gate.go` (new) — the pure, injectable `registeredFactionIds` helper. Small, single responsibility, unit-testable without the faction registry or the live mob-instance map.
- `internal/seeders/faction_gate_test.go` (new) — unit tests for the helper.
- `internal/seeders/aggressive_action_to_revenge.go` (modify) — apply the gate.
- `internal/seeders/aggressive_action_to_revenge_test.go` (modify) — add a no-panic test for the factionless-victim early-return path (mirrors the existing no-panic tests; full instance-map integration is covered by boot/live smoke, consistent with the codebase's deferral pattern in `friend_killed_to_revenge_test.go:92`).

---

## Task 1: Pure faction-resolution helper

**Files:**
- Create: `internal/seeders/faction_gate.go`
- Test: `internal/seeders/faction_gate_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/seeders/faction_gate_test.go`:

```go
package seeders

import "testing"

func TestRegisteredFactionIds(t *testing.T) {
	// isFaction stub: only "thornwall_citizens" and "bandits" are real factions.
	isFaction := func(g string) bool {
		return g == "thornwall_citizens" || g == "bandits"
	}

	tests := []struct {
		name   string
		groups []string
		want   []string
	}{
		{"factionless (only non-faction groups)", []string{"construct", "humanoid"}, []string{}},
		{"empty groups", nil, []string{}},
		{"one faction among non-factions", []string{"humanoid", "thornwall_citizens"}, []string{"thornwall_citizens"}},
		{"multiple factions", []string{"bandits", "coulee_folk", "thornwall_citizens"}, []string{"bandits", "thornwall_citizens"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := registeredFactionIds(tc.groups, isFaction)
			if len(got) != len(tc.want) {
				t.Fatalf("registeredFactionIds(%v) = %v, want %v", tc.groups, got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("registeredFactionIds(%v) = %v, want %v", tc.groups, got, tc.want)
				}
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/seeders/ -run TestRegisteredFactionIds -v`
Expected: FAIL — `undefined: registeredFactionIds`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/seeders/faction_gate.go`:

```go
package seeders

// registeredFactionIds returns the subset of groups that are registered
// factions, per the isFaction predicate. Kept pure + injectable so the gate
// decision is unit-testable without loading the faction registry or the live
// mob-instance map. Callers pass factions.GetDefinition(g) != nil as isFaction.
func registeredFactionIds(groups []string, isFaction func(string) bool) []string {
	out := make([]string, 0, len(groups))
	for _, g := range groups {
		if isFaction(g) {
			out = append(out, g)
		}
	}
	return out
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/seeders/ -run TestRegisteredFactionIds -v`
Expected: PASS (all 4 sub-tests).

- [ ] **Step 5: Commit**

```bash
git add internal/seeders/faction_gate.go internal/seeders/faction_gate_test.go
git commit -m "feat(seeders): pure registeredFactionIds helper for the witness faction gate"
```

---

## Task 2: Gate the witness-response seeder on the victim's faction

**Files:**
- Modify: `internal/seeders/aggressive_action_to_revenge.go`
- Test: `internal/seeders/aggressive_action_to_revenge_test.go`

Current body of `aggressiveActionToRevenge` (for reference) resolves `attackedMob`, returns early on `attackedMob == nil` and on `attackedMob.AutoAggro`, then calls `seedWitnessResponse(attackedMob, aggressiveVictimRevengePriority)` and loops `room.GetMobs()` seeding every non-`AutoAggro` witness at `aggressiveWitnessRevengePriority`.

- [ ] **Step 1: Write the failing test**

Add to `internal/seeders/aggressive_action_to_revenge_test.go` (a no-panic guard for the new early-return path — mirrors the existing zero-field no-panic tests in that file):

```go
func TestAggressiveActionToRevenge_UnknownMob_NoPanic_WithGate(t *testing.T) {
	// MobInstanceId that resolves to no instance: the function must return
	// cleanly (attackedMob == nil) even with the faction gate in place.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("aggressiveActionToRevenge panicked: %v", r)
		}
	}()
	aggressiveActionToRevenge(events.PlayerAttackedMob{UserId: 5, MobInstanceId: 987654})
}
```

- [ ] **Step 2: Run test to verify it passes against current code, then confirm the gate keeps it passing**

Run: `go test ./internal/seeders/ -run TestAggressiveActionToRevenge -v`
Expected: PASS (this guards behavior; it passes before and after the change — its job is to catch a panic introduced by the new imports/logic). If `events` is already imported in the test file (it is — existing tests use `events.PlayerAttackedMob`), no import change is needed.

- [ ] **Step 3: Apply the faction gate**

Edit `internal/seeders/aggressive_action_to_revenge.go`. Add imports `"github.com/GoMudEngine/GoMud/internal/crimes"` and `"github.com/GoMudEngine/GoMud/internal/factions"` to the import block. Replace the body from the `seedWitnessResponse(attackedMob, ...)` line through the end of the witness loop with:

```go
	// Gate on the VICTIM's faction, mirroring the (already faction-gated) crime
	// record: attacking a factionless target (training dummy, wildlife, monster,
	// unaligned tutorial mob) is not a social crime, so no bystander reacts.
	victimFactions := registeredFactionIds(attackedMob.Groups, func(g string) bool {
		return factions.GetDefinition(g) != nil
	})
	if len(victimFactions) == 0 {
		return
	}

	// Classify and respond for the attacked mob at victim priority.
	seedWitnessResponse(attackedMob, pa.UserId, aggressiveVictimRevengePriority)

	// Seed revenge into witnesses who SHARE a faction with the victim (same rule
	// crimes.WitnessesInRoom applies). Skip AutoAggro witnesses — they already
	// attack on sight, so revenge is redundant noise.
	room := rooms.LoadRoom(attackedMob.Character.RoomId)
	if room == nil {
		return
	}
	for _, witnessInstId := range crimes.WitnessesInRoom(victimFactions, room, attackedMob.InstanceId) {
		witness := mobs.GetInstance(witnessInstId)
		if witness == nil || witness.AutoAggro {
			continue
		}
		seedWitnessResponse(witness, pa.UserId, aggressiveWitnessRevengePriority)
	}
```

Note: this removes the old `room.GetMobs()` loop and the `if witnessInstId == attackedMob.InstanceId { continue }` self-skip (now handled by passing `attackedMob.InstanceId` as `excludeInstanceId` to `WitnessesInRoom`). The `rooms` import is already present and still used.

- [ ] **Step 4: Run tests + build to verify**

Run: `go test ./internal/seeders/ -v` and `go build ./...`
Expected: build succeeds; all seeder tests PASS (including `TestRegisteredFactionIds` from Task 1 and the no-panic guards). No unused-import errors (`crimes`, `factions`, `mobs`, `rooms` all referenced).

- [ ] **Step 5: Boot smoke (data-load + wiring)**

Run (POSIX/bash): nuke instance saves, boot, watch for readiness with no panic, then kill:

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
(go run . > /tmp/flee_boot.log 2>&1 &) ; \
  c=0; while [ $c -lt 90 ]; do grep -qE "Server Ready|panic:" /tmp/flee_boot.log && break; c=$((c+1)); ping -n 2 127.0.0.1 >/dev/null 2>&1; done ; \
  grep -iE "Server Ready|panic:" /tmp/flee_boot.log | tail -2 ; taskkill //F //IM GoMud.exe
```
Expected: `Server Ready`, no `panic:`.

- [ ] **Step 6: Commit**

```bash
git add internal/seeders/aggressive_action_to_revenge.go internal/seeders/aggressive_action_to_revenge_test.go
git commit -m "fix(seeders): faction-gate witness response so unaligned targets (dummies) trigger no bystander reaction"
```

---

## Task 3: Full-suite verification

**Files:** none (verification only)

- [ ] **Step 1: Run the full test suite**

Run: `go test ./... 2>&1 | grep -E "FAIL" || echo ALL_PASS`
Expected: `ALL_PASS` (no FAIL lines; ~88 packages ok).

- [ ] **Step 2: (Optional) live check**

Boot the server, create a veteran/newbie character, go to the Drill Yard (room 5227) where Drillmaster Vorn (9108) and the Training Dummy (9109) are, `attack dummy`, and confirm Vorn no longer "recoils and cries out, then hurries for the nearest way out." (Manual; the boot smoke + unit tests are the automated gate.)

---

## Self-Review

**1. Spec coverage:**
- Spec "resolve victim's registered-faction Groups" → Task 1 (`registeredFactionIds`) + Task 2 Step 3 (called with `factions.GetDefinition`). ✓
- Spec "factionless victim → return early" → Task 2 Step 3 (`if len(victimFactions) == 0 { return }`). ✓
- Spec "factioned victim → seed victim + only same-faction non-AutoAggro witnesses" → Task 2 Step 3 (`seedWitnessResponse(attackedMob,...)` + `crimes.WitnessesInRoom` loop skipping `AutoAggro`). ✓
- Spec "reuse crimes.WitnessesInRoom / factions.GetDefinition, no import cycle" → Task 2 Step 3 imports. ✓
- Spec "combat-flee + crime record unchanged" → not touched by any task. ✓
- Spec testing (factionless-no-reaction, same/different-faction witness, AutoAggro skip): the DECISION logic is unit-tested via `registeredFactionIds` (Task 1) + the already-tested `crimes.WitnessesInRoom`; the wired path is covered by no-panic + boot/live smoke, matching the codebase's instance-map-integration deferral. ✓

**2. Placeholder scan:** No TBD/TODO; every code step shows complete code. ✓

**3. Type consistency:** `registeredFactionIds(groups []string, isFaction func(string) bool) []string` defined in Task 1, called identically in Task 2. `crimes.WitnessesInRoom(factionIds []string, room *rooms.Room, excludeInstanceId int) []int` and `factions.GetDefinition(factionId string) *Definition` match verified signatures. `mobs.GetInstance`, `attackedMob.Groups`, `attackedMob.InstanceId`, `attackedMob.AutoAggro`, `attackedMob.Character.RoomId`, `pa.UserId` all match existing usage in the file. ✓
