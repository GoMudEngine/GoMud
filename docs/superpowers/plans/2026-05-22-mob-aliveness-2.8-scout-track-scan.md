# Mob Aliveness 2.8 — Scout / Track / Scan Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Lift `scan`, `track`, and `search` into `internal/actions/` via the actor pattern; add three btree action primitives + one chase-consumer + two conditions; wire a new `scout` archetype on goblin_scout (217); graft scan-before-ambush onto `lookout`, search-before-steal onto `thief`, and track-on-aggro-lost onto `leader`. Fix the latent buff-26-misuse bug discovered during plan-writing by authoring a real Active Tracking buff (86) and migrating the player code to it.

**Architecture:** Three actions in `internal/actions/{scan,track,search}.go` with `Verb(actor Actor, opts) Result` signatures (2.7 pattern). Thin player wrappers in `usercommands/`. Mob wrappers in `mobcommands/`. Btree primitives in `behaviortree/actions_scout.go` + `conditions_scout.go`. Active tracking persists via `tracking-user`/`tracking-mob` misc-data on the actor's `Character`, plus buff 86 as a 16-round duration token (introspection + auto-expiry). `move_toward_tracked` reads the misc data each tick and dispatches `go <direction>`.

**Tech Stack:** Go, GoMud engine, YAML data files. TDD with `go test`.

---

## Plan-writing discovery — buff 26 is Conviction Surge

During plan-writing I verified that `_datafiles/world/dogmud/buffs/26-conviction_surge.yaml` is the only YAML with `buffid: 26`. The comment in `skill.track.go:267,284,312,329` claiming `// 26 is the buff for active tracking` is wrong. Today, when a player runs `track <name>` and rolls high enough for active tracking, the engine silently applies Conviction Surge (+15 strength, 16 rounds, damage-bonus flag). The `roomdetails.go` "tracking direction" rendering is gated by `tracking-user`/`tracking-mob` misc-data, NOT by `HasBuff(26)` — verified by reading the surrounding control flow at `roomdetails.go:413-520`.

**This plan fixes the bug as part of 2.8.** Task 1 authors buff 86 (Active Tracking — no statmods, 16-round duration). Tasks 2-3 migrate the four player-side `AddBuff(26)` calls and five player-side `RemoveBuff(26)` calls to use 86. The mob-side path uses 86 for the same purpose.

Throughout this plan, references to "the tracking buff" mean buff 86 (authored in Task 1), not buff 26.

## Implementation order rationale

1. **Buff infrastructure first (Tasks 1-3):** Derisks the symmetric-buff piece *and* fixes the player bug before any new consumer is wired.
2. **Actions package lifts (Tasks 4-6):** Source-of-truth code for the three verbs. TDD per file. Self-contained, no cross-task dependencies.
3. **Player wrappers thin (Tasks 7-9):** Each depends only on its corresponding action.
4. **Mob wrappers (Task 10):** Depends on actions; trivially small.
5. **Btree primitives (Tasks 11-12):** Depends on actions. Adds the four action primitives + two conditions.
6. **Archetype YAMLs (Tasks 13-16):** Depends on btree primitives. Each graft is independent.
7. **Docs + smoke (Tasks 17-18):** context.md updates + the smoke plan from the spec.

## Task table-of-contents

| # | Task | Files | Deps |
|---|------|-------|------|
| 1 | Author `86-active_tracking.yaml` | New buff YAML | — |
| 2 | Migrate player track.go AddBuff(26)→AddBuff(86) | usercommands/skill.track.go | 1 |
| 3 | Migrate roomdetails.go to buff 86 + fix "tracking forever" bug | rooms/roomdetails.go | 1 |
| 4 | Lift Scan into actions/ | actions/scan.go + test | — |
| 5 | Lift Track into actions/ | actions/track.go + test | 1 |
| 6 | Lift Search into actions/ | actions/search.go + test | — |
| 7 | Thin usercommands/scan.go | usercommands/scan.go | 4 |
| 8 | Thin usercommands/skill.track.go | usercommands/skill.track.go | 5 |
| 9 | Thin usercommands/skill.search.go | usercommands/skill.search.go | 6 |
| 10 | Add mobcommands/{scan,track,search}.go | mobcommands/*.go | 4,5,6 |
| 11 | Add actions_scout.go + conditions_scout.go | behaviortree/*.go | 4,5,6 |
| 12 | Register btree primitives in init() | behaviortree/actions.go, conditions.go | 11 |
| 13 | Author scout.yaml + flip goblin_scout (217) | behaviors/archetypes/scout.yaml, mobs/ironwind_steppe/217-*.yaml | 12 |
| 14 | Graft scan-before-ambush onto lookout.yaml | behaviors/archetypes/lookout.yaml | 12 |
| 15 | Graft search-before-steal onto thief.yaml | behaviors/archetypes/thief.yaml | 12 |
| 16 | Graft track-on-aggro-lost onto leader.yaml | behaviors/archetypes/leader.yaml | 12 |
| 17 | Update context.md files | behaviortree/context.md, actions/context.md | 11-16 |
| 18 | Boot + run smoke + mark roadmap done | server boot, MOB_ALIVENESS_ROADMAP.md | all |

---

## Phase 1 — Tracking buff infrastructure

### Task 1: Author `86-active_tracking.yaml`

**Files:**
- Create: `_datafiles/world/dogmud/buffs/86-active_tracking.yaml`

- [ ] **Step 1: Create the buff YAML**

Write `_datafiles/world/dogmud/buffs/86-active_tracking.yaml` with this content:

```yaml
buffid: 86
name: Active Tracking
description: You are following a quarry's trail with heightened awareness.
triggerrate: 1 round
triggercount: 16
start_user_text: You commit the trail to memory and steady your senses.
end_user_text: The trail grows cold; your focus slips.
```

No statmods, no flags. Pure duration token + introspection state. Filename derives via `ConvertForFilename("Active Tracking")` → `active_tracking` (per CLAUDE.md naming SOP).

- [ ] **Step 2: Boot the server and confirm the buff loads**

Run: `go run . 2>&1 | head -120`

Expected: server boots past `buffs.LoadDataFiles() loadedCount=70` (previously 69). No panics. Kill the server after confirmation (Ctrl+C / process kill).

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/buffs/86-active_tracking.yaml
git commit -m "$(cat <<'EOF'
feat(buffs): add 86 Active Tracking (replaces buff-26 misuse)

Authored buffid 86 with no statmods + 16-round duration. Replaces
the latent misuse of buff 26 (Conviction Surge) in skill.track.go,
which was silently giving players a +15 strength combat buff while
active tracking was on.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 2: Migrate player track.go AddBuff(26)→AddBuff(86)

**Files:**
- Modify: `internal/usercommands/skill.track.go:267,284,312,329`

- [ ] **Step 1: Replace all four AddBuff(26) calls with AddBuff(86)**

Use Edit tool on `internal/usercommands/skill.track.go`. There are four sites, each with the same `old_string` body:

```go
				user.AddBuff(26, `skill`) // 26 is the buff for active tracking
```

Replace each with:

```go
				user.AddBuff(86, `skill`) // 86 is the Active Tracking buff
```

Use `replace_all: true` since all four occurrences are identical.

- [ ] **Step 2: Build to confirm no compile breakage**

Run: `go build ./...`

Expected: clean exit, no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/usercommands/skill.track.go
git commit -m "$(cat <<'EOF'
fix(track): point active tracking at buff 86, not Conviction Surge

Previously skill.track.go applied AddBuff(26) on a successful active
track. Buff 26 is Conviction Surge (+15 strength, 16-round combat
buff) — players were silently getting a damage bonus for running
recon. Migrated to buff 86 (Active Tracking, no statmods).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 3: Migrate roomdetails.go RemoveBuff(26)→RemoveBuff(86) AND fix "tracking forever" bug

**Files:**
- Modify: `internal/rooms/roomdetails.go:413-520` (the two tracking-render blocks)

**Historical context (must read):** Players previously reported a "tracking message never goes away" bug. Root cause: the rendering blocks at `roomdetails.go:413` (tracking-mob) and `:474` (tracking-user) gate only on misc data (`if searchMobName := user.Character.GetMiscData('tracking-mob'); searchMobName != nil`). The five `RemoveBuff(26)` call sites within those blocks remove the buff and clear `tracking-display-count` but do NOT clear `tracking-mob` / `tracking-user`. Once those misc keys are set (during `skill.track.go`'s active-track success path), the only way to clear them is via explicit `track stop/clear`. When the buff expires naturally — or when target is found / trail is lost via the existing logic — the misc data persists and the render keeps firing. The previous attempt to fix this likely chose buff 26 (Conviction Surge) specifically for its 16-round expiry, then forgot to wire the misc-data cleanup on buff expiry.

This task fixes both the buff-id migration and the misc-data leak.

- [ ] **Step 1: Verify the call sites with grep**

Run: `Grep` tool with pattern `RemoveBuff\(26\)` in path `internal/rooms/roomdetails.go`, output_mode `content`, `-n: true`.

Expected: ~five-to-six matches (verified during plan-writing was five — recheck).

- [ ] **Step 2: At every site that calls RemoveBuff(26), also clear tracking misc data**

For each of the five `RemoveBuff(26)` sites (lines 421, 439, 449, 481, 499, 509 — re-verify line numbers), the migration becomes a three-line block, not a one-line replace:

**Before:**
```go
user.Character.RemoveBuff(26)
user.Character.SetMiscData("tracking-display-count", nil)
```

**After:**
```go
user.Character.RemoveBuff(86)
user.Character.SetMiscData("tracking-display-count", nil)
user.Character.SetMiscData("tracking-mob", nil)
user.Character.SetMiscData("tracking-user", nil)
```

Use Edit tool on each site individually (sites are not identical — some appear in the tracking-mob block, some in the tracking-user block — but the cleanup pattern is the same: clear BOTH misc keys at every cleanup site, since at most one is set anyway and a nil-clear is a no-op).

- [ ] **Step 3: Add a HasBuff(86) outer gate to BOTH tracking-render blocks**

This is the structural fix for "tracking forever." At `roomdetails.go:413` (the tracking-mob block) and `:474` (the tracking-user block), wrap the existing `if searchMobName ... != nil { ... }` body with a buff-presence check. If misc data is set but the buff is absent (buff expired or otherwise removed without cleanup), clear the misc data and render nothing.

**Before** (paraphrased structure):
```go
if searchMobName := user.Character.GetMiscData(`tracking-mob`); searchMobName != nil {
    if searchMobNameStr, ok := searchMobName.(string); ok {
        // ... rendering logic ...
    }
}
```

**After:**
```go
if searchMobName := user.Character.GetMiscData(`tracking-mob`); searchMobName != nil {
    // Buff-absent cleanup: if tracking misc data was set but the
    // active-tracking buff expired or was removed, drop the misc
    // data so the render doesn't fire forever.
    if !user.Character.HasBuff(86) {
        user.Character.SetMiscData("tracking-mob", nil)
        user.Character.SetMiscData("tracking-user", nil)
        user.Character.SetMiscData("tracking-display-count", nil)
    } else if searchMobNameStr, ok := searchMobName.(string); ok {
        // ... existing rendering logic unchanged ...
    }
}
```

Apply the same pattern to the tracking-user block at line ~474.

The buff's natural-expiry text ("The trail grows cold; your focus slips.") fires automatically when buff 86 expires per the YAML's `end_user_text`. The misc-data cleanup runs on the next room view (essentially the next player action), making the cleanup observably tied to the buff lifecycle from the player's POV.

- [ ] **Step 4: Build to confirm no compile breakage**

Run: `go build ./...`

Expected: clean exit, no errors.

- [ ] **Step 5: Spot-check with grep**

Confirm no remaining `RemoveBuff(26)` references in non-Conviction-Surge contexts:

```bash
# Inside Grep tool:
pattern: "RemoveBuff\\(26\\)|AddBuff\\(26\\)"
output_mode: "content"
-n: true
```

Expected: zero matches (all migrated to 86). If any remain, they need migration too.

- [ ] **Step 6: Commit**

```bash
git add internal/rooms/roomdetails.go
git commit -m "$(cat <<'EOF'
fix(rooms): cleanup tracking misc data on buff absence + on stop sites

Fixes the long-standing "tracking message never goes away" bug.
The render blocks at lines 413/474 previously gated only on misc
data, while the buff (now 86) was the only auto-expiry mechanism.
When the buff expired, misc data persisted and the render kept
firing forever.

Adds a HasBuff(86) outer gate that clears misc data on buff
absence. Also clears tracking-mob/tracking-user at every existing
RemoveBuff site (target-found / trail-lost) so the cleanup
contract is symmetric.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Phase 2 — Actions package lifts

### Task 4: Lift Scan into actions/scan.go (TDD)

**Files:**
- Create: `internal/actions/scan.go`
- Create: `internal/actions/scan_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/actions/scan_test.go`:

```go
package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// TestScan_EmptyAdjacentRooms confirms Scan returns an empty Sightings
// slice (not nil) when the actor's room has no visible exits or all
// adjacent rooms are empty.
func TestScan_EmptyAdjacentRooms(t *testing.T) {
	// Set up an isolated room (no exits) and a UserActor in it.
	// Use the existing test scaffolding — see actions_test.go for the
	// makeTestRoom / makeTestUserActor helpers.
	room := makeTestRoom(t, 9001)
	user := makeTestUser(t, "TestScout")
	user.Character.RoomId = room.RoomId
	actor := &UserActor{User: user, Room: room}

	result := Scan(actor, ScanOptions{HostileOnly: false})

	if len(result.Sightings) != 0 {
		t.Errorf("expected 0 sightings, got %d", len(result.Sightings))
	}
}

// TestScan_MobActorSilent confirms MobActor.SendText is not called —
// the action returns structured data without emitting any text.
func TestScan_MobActorSilent(t *testing.T) {
	room := makeTestRoom(t, 9002)
	mob := makeTestMob(t, "TestScoutMob", room.RoomId)
	actor := NewMobActorInRoom(mob, room)

	// MobActor.SendText is a no-op by interface contract; verify the
	// action returns a populated result regardless.
	result := Scan(actor, ScanOptions{HostileOnly: true})

	// Empty room — just confirm the call doesn't panic and returns
	// a zero-valued result cleanly.
	if result.Sightings == nil {
		t.Error("Sightings should be empty slice, not nil")
	}
}
```

Note: the test references `makeTestRoom`, `makeTestUser`, `makeTestMob` helpers. Check `internal/actions/actions_test.go` for existing helpers; reuse if they exist, otherwise add to a new `testutil_test.go` (per existing convention). Adapt names to match.

- [ ] **Step 2: Run the test, verify it fails (Scan undefined)**

Run: `go test ./internal/actions/ -run TestScan -v`

Expected: build error or test failure with "Scan undefined" / "undefined: Scan".

- [ ] **Step 3: Implement actions/scan.go**

Create `internal/actions/scan.go`:

```go
package actions

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// ScanOptions parameterizes a one-step adjacent-room sweep.
type ScanOptions struct {
	// HostileOnly: when true, mob-actor SoftTarget population (via the
	// btree wrapper) gates on hostile-ness. The action itself does not
	// filter Sightings by hostility — that decision is up to the caller.
	HostileOnly bool
}

// ScanEntity describes one occupant in a sighted room.
type ScanEntity struct {
	Id    int
	Name  string
	IsMob bool
}

// ScanSighting describes one adjacent room's occupants.
type ScanSighting struct {
	ExitName  string
	RoomId    int
	RoomTitle string
	Mobs      []ScanEntity
	Players   []ScanEntity
}

// ScanResult is the structured outcome.
type ScanResult struct {
	Sightings []ScanSighting
}

// Scan walks each visible (non-secret) exit, loads the adjacent room,
// and lists non-hidden mobs and players in each. No skill check, no
// cooldown. UserActor receives a rendered text list via SendText;
// MobActor SendText is a no-op (silent). The result is returned in
// both cases.
func Scan(actor Actor, opts ScanOptions) ScanResult {
	result := ScanResult{Sightings: []ScanSighting{}}

	room := actor.GetRoom()
	if room == nil {
		return result
	}

	for exitName, exitInfo := range room.Exits {
		if exitInfo.Secret {
			continue
		}

		adjRoom := rooms.LoadRoom(exitInfo.RoomId)
		if adjRoom == nil {
			continue
		}

		sighting := ScanSighting{
			ExitName:  exitName,
			RoomId:    adjRoom.RoomId,
			RoomTitle: adjRoom.Title,
		}

		for _, mobInstId := range adjRoom.GetMobs(rooms.FindAll) {
			m := mobs.GetInstance(mobInstId)
			if m == nil || m.Character.IsHidden() {
				continue
			}
			sighting.Mobs = append(sighting.Mobs, ScanEntity{
				Id:    mobInstId,
				Name:  m.Character.Name,
				IsMob: true,
			})
		}

		for _, pId := range adjRoom.GetPlayers(rooms.FindAll) {
			if pId == actor.GetUserId() {
				continue
			}
			u := users.GetByUserId(pId)
			if u == nil {
				continue
			}
			sighting.Players = append(sighting.Players, ScanEntity{
				Id:    pId,
				Name:  u.Character.Name,
				IsMob: false,
			})
		}

		result.Sightings = append(result.Sightings, sighting)
	}

	// UserActor text rendering. MobActor.SendText is a no-op.
	if actor.IsPlayer() {
		actor.SendText(messaging.CategorySystem, `You scan the surrounding area...`)
		actor.SendText(messaging.CategorySystem, ``)
		if len(result.Sightings) == 0 {
			actor.SendText(messaging.CategorySystem, `  There are no visible exits to scan.`)
		}
		for _, s := range result.Sightings {
			parts := []string{}
			for _, m := range s.Mobs {
				parts = append(parts, fmt.Sprintf(`<ansi fg="mobname">%s</ansi>`, m.Name))
			}
			for _, p := range s.Players {
				parts = append(parts, fmt.Sprintf(`<ansi fg="username">%s</ansi>`, p.Name))
			}
			dirLabel := fmt.Sprintf(`<ansi fg="exit">%s</ansi>`, s.ExitName)
			titleLabel := fmt.Sprintf(`<ansi fg="room-title">%s</ansi>`, s.RoomTitle)
			if len(parts) > 0 {
				actor.SendText(messaging.CategorySystem,
					fmt.Sprintf(`  %s (%s): %s`, dirLabel, titleLabel, strings.Join(parts, `, `)))
			} else {
				actor.SendText(messaging.CategorySystem,
					fmt.Sprintf(`  %s (%s): nothing of interest`, dirLabel, titleLabel))
			}
		}
		actor.SendText(messaging.CategorySystem, ``)
	}

	return result
}
```

- [ ] **Step 4: Run the test, verify it passes**

Run: `go test ./internal/actions/ -run TestScan -v`

Expected: both `TestScan_EmptyAdjacentRooms` and `TestScan_MobActorSilent` PASS.

- [ ] **Step 5: Run the full actions package tests to confirm no regression**

Run: `go test ./internal/actions/...`

Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/actions/scan.go internal/actions/scan_test.go
git commit -m "$(cat <<'EOF'
feat(actions): lift Scan into actions/ via actor pattern (2.8)

Scan(actor, opts) ScanResult. UserActor emits rendered list via
SendText; MobActor SendText is a no-op. The btree wrapper in
behaviortree/actions_scout.go will consume the structured result.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 5: Lift Track into actions/track.go (TDD)

**Files:**
- Create: `internal/actions/track.go`
- Create: `internal/actions/track_test.go`

Track is the most complex of the three — it has trail-scan mode (no target), active-track mode (with target name or context source), buff application, and misc-data persistence. The existing player code is ~480 lines; the action will compress this to ~250 by sharing the trail-search and target-resolution logic between modes.

- [ ] **Step 1: Write the failing tests**

Create `internal/actions/track_test.go`:

```go
package actions

import (
	"testing"
)

// TestTrack_NoArgEmptyRoom confirms trail-scan mode on a room with no
// visitors returns an empty Visitors slice.
func TestTrack_NoArgEmptyRoom(t *testing.T) {
	room := makeTestRoom(t, 9101)
	user := makeTestUser(t, "TrackTester")
	user.Character.RoomId = room.RoomId
	actor := &UserActor{User: user, Room: room}

	result := Track(actor, TrackOptions{})

	if len(result.Visitors) != 0 {
		t.Errorf("expected 0 visitors, got %d", len(result.Visitors))
	}
	if result.BuffApplied {
		t.Error("BuffApplied should be false on trail-scan mode")
	}
}

// TestTrack_ActiveTrackMobNoMatchFails confirms active-track mode with
// an unresolvable target sets Reason and does NOT apply buff 86.
func TestTrack_ActiveTrackMobNoMatchFails(t *testing.T) {
	room := makeTestRoom(t, 9102)
	user := makeTestUser(t, "TrackTester2")
	user.Character.RoomId = room.RoomId
	actor := &UserActor{User: user, Room: room}

	result := Track(actor, TrackOptions{TargetNoun: "nonexistent_target"})

	if result.BuffApplied {
		t.Error("BuffApplied should be false when target unresolved")
	}
	if result.ActiveTargetUserId != 0 || result.ActiveTargetMobInstId != 0 {
		t.Error("ActiveTarget* should be 0 when target unresolved")
	}
}

// TestTrack_MobActorSilent confirms MobActor.SendText is a no-op.
func TestTrack_MobActorSilent(t *testing.T) {
	room := makeTestRoom(t, 9103)
	mob := makeTestMob(t, "TrackMob", room.RoomId)
	actor := NewMobActorInRoom(mob, room)

	// Confirm no panic, result returned cleanly.
	_ = Track(actor, TrackOptions{})
}
```

- [ ] **Step 2: Run the test, verify it fails**

Run: `go test ./internal/actions/ -run TestTrack -v`

Expected: build error or "Track undefined".

- [ ] **Step 3: Implement actions/track.go**

Create `internal/actions/track.go`:

```go
package actions

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// TrackOptions parameterizes the trail-read.
type TrackOptions struct {
	// TargetNoun: name of target to actively track. Empty = trail-scan
	// current room mode.
	TargetNoun string

	// TargetFrom (mob path): "aggro" | "event" | "soft_target" | "none".
	// Ignored if TargetNoun is set. When set, the action resolves the
	// target's name from the named context source and treats it as
	// TargetNoun.
	TargetFrom string

	// CancelTracking: "stop" / "clear" semantics — remove buff 86 + clear
	// tracking misc data without rolling. UserActor wrapper handles
	// the keyword check.
	CancelTracking bool
}

// TrackingInfo is the per-visitor record used by the descriptions/track
// template. Lifted verbatim from usercommands/skill.track.go.
type TrackingInfo struct {
	Name            string
	Type            string // "mob" or "user"
	Strength        string
	NumericStrength float64
	ExitName        string
}

// TrackResult is the structured outcome.
type TrackResult struct {
	// Trail-scan mode populates Visitors.
	Visitors []TrackingInfo

	// Active-track mode populates these.
	ActiveTargetUserId    int    // 0 when not user target
	ActiveTargetMobInstId int    // 0 when not mob target
	ActiveTargetName      string // for caller messaging
	DirectionExit         string // best exit toward target
	BuffApplied           bool   // true when buff 86 applied

	// Common.
	RollValue  int    // roll.Value for tier-band introspection
	OnCooldown bool   // 1-round cooldown collision
	Reason     string // human-readable reason on failure
}

// activeTrackingBuffId is the buff applied when active tracking starts.
// See _datafiles/world/dogmud/buffs/86-active_tracking.yaml.
const activeTrackingBuffId = 86

// Track runs the Perception+Search trail-read. With TargetNoun set (or
// resolved via TargetFrom), attempts active-tracking: locates the trail
// across adjacent rooms, applies buff 86, and stores tracking-user or
// tracking-mob misc data on the actor's Character. Without a target,
// reports the visitor log of the current room (tiered by roll).
func Track(actor Actor, opts TrackOptions) TrackResult {
	result := TrackResult{}

	char := actor.GetCharacter()
	room := actor.GetRoom()
	if char == nil || room == nil {
		result.Reason = "no character or room"
		return result
	}

	// "stop"/"clear" path — pure cleanup, no roll.
	if opts.CancelTracking {
		char.SetMiscData("tracking-mob", nil)
		char.SetMiscData("tracking-user", nil)
		char.SetMiscData("tracking-display-count", nil)
		char.RemoveBuff(activeTrackingBuffId)
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem, `You stop tracking.`)
		}
		return result
	}

	// Resolve target source if TargetNoun absent.
	targetNoun := opts.TargetNoun
	// Mob target_from sources are resolved by the btree wrapper, which
	// extracts the name and sets TargetNoun before calling. This action
	// only consumes TargetNoun. TargetFrom is reserved for the wrapper.

	// Roll the Perception+Search score.
	searchRank := char.GetSkillLevel(skills.Search)
	searchScore := float64(char.Stats.Perception.ValueAdj) +
		combat.SkillMultiplier(searchRank)*25.0
	roll := dice.RollStat(searchScore)
	result.RollValue = roll.Value

	// Skill progression on every fired roll.
	actor.OnSkillUse(string(skills.Search))

	// Roll < 125: no tracks visible at all.
	if roll.Value < 125 {
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem, "You don't see any tracks.")
		}
		result.Reason = "roll below detection threshold"
		return result
	}

	// Trail-scan mode (no-arg).
	if targetNoun == "" {
		if !char.TryCooldown(skills.Search.String(), "1 round") {
			result.OnCooldown = true
			if actor.IsPlayer() {
				actor.SendText(messaging.CategorySystem,
					fmt.Sprintf("You need to wait %d more rounds to use that skill again.",
						char.GetCooldown(skills.Search.String())))
			}
			return result
		}

		result.Visitors = readRoomTrail(room, char, roll.Value, actor.GetUserId(), actor.GetMobInstanceId())
		if actor.IsPlayer() {
			renderTrailToPlayer(actor, result.Visitors)
		}
		return result
	}

	// Active-track mode.
	if roll.Value < 175 {
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem, "Your tracking skills aren't sharp enough right now.")
		}
		result.Reason = "active-track requires roll >= 175"
		return result
	}

	if !char.TryCooldown(skills.Search.String(), "1 round") {
		result.OnCooldown = true
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem,
				fmt.Sprintf("You need to wait %d more rounds to use that skill again.",
					char.GetCooldown(skills.Search.String())))
		}
		return result
	}

	// Find target in current room first (sets BuffApplied = false, just
	// reports "they are here").
	if targetUser := findUserInRoomByName(room, targetNoun, actor.GetUserId()); targetUser != nil {
		result.ActiveTargetUserId = targetUser.UserId
		result.ActiveTargetName = targetUser.Character.Name
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem,
				fmt.Sprintf(`<ansi fg="username">%s</ansi> is in the room with you!`, targetUser.Character.Name))
		}
		return result
	}
	if targetMob := findMobInRoomByName(room, targetNoun); targetMob != nil {
		result.ActiveTargetMobInstId = targetMob.InstanceId
		result.ActiveTargetName = targetMob.Character.Name
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is in the room with you!`, targetMob.Character.Name))
		}
		return result
	}

	// Search adjacent rooms via visitor log; if found, apply buff 86 +
	// store misc data + populate DirectionExit.
	if applied, miscKey, miscVal, dirExit, targetUserId, targetMobId, targetName := lookupAdjacentTrail(room, char, targetNoun, actor.GetUserId()); applied {
		char.SetMiscData(miscKey, miscVal)
		char.SetMiscData("tracking-display-count", nil)
		actor.AddBuff(activeTrackingBuffId, "skill")
		result.BuffApplied = true
		result.ActiveTargetUserId = targetUserId
		result.ActiveTargetMobInstId = targetMobId
		result.ActiveTargetName = targetName
		result.DirectionExit = dirExit
		return result
	}

	if actor.IsPlayer() {
		actor.SendText(messaging.CategorySystem, "You don't see any tracks.")
	}
	result.Reason = "no trail found in adjacent rooms"
	return result
}

// readRoomTrail returns the visitor list for the current room, filtered
// by detection tier. roll < 135 returns at most one (strongest) visitor;
// roll >= 135 returns all; roll >= 175 also populates ExitName via
// adjacent-room scan.
func readRoomTrail(room *rooms.Room, char interface{}, rollValue int, excludeUserId int, excludeMobId int) []TrackingInfo {
	out := []TrackingInfo{}
	currentMobs := room.GetMobs()
	currentUsers := room.GetPlayers()

	// Mob trails.
	for mId, timeLeft := range room.Visitors(rooms.VisitorMob) {
		if mId == excludeMobId {
			continue
		}
		skip := false
		for _, curId := range currentMobs {
			if mId == curId {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		m := mobs.GetInstance(mId)
		if m == nil {
			continue
		}
		info := TrackingInfo{
			Name:            m.Character.Name,
			Type:            "mob",
			Strength:        trailStrengthToString(timeLeft),
			NumericStrength: timeLeft,
		}
		if rollValue >= 175 {
			info.ExitName = findExited(room, mId, rooms.VisitorMob)
		}
		if rollValue < 135 {
			if len(out) == 0 || out[0].NumericStrength < timeLeft {
				if len(out) == 0 {
					out = append(out, info)
				} else {
					out[0] = info
				}
			}
			continue
		}
		out = append(out, info)
	}

	// User trails.
	for uId, timeLeft := range room.Visitors(rooms.VisitorUser) {
		if uId == excludeUserId {
			continue
		}
		skip := false
		for _, curId := range currentUsers {
			if uId == curId {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		u := users.GetByUserId(uId)
		if u == nil {
			continue
		}
		info := TrackingInfo{
			Name:            u.Character.Name,
			Type:            "user",
			Strength:        trailStrengthToString(timeLeft),
			NumericStrength: timeLeft,
		}
		if rollValue >= 175 {
			info.ExitName = findExited(room, uId, rooms.VisitorUser)
		}
		if rollValue < 135 {
			if len(out) == 0 || out[0].NumericStrength < timeLeft {
				if len(out) == 0 {
					out = append(out, info)
				} else {
					out[0] = info
				}
			}
			continue
		}
		out = append(out, info)
	}

	return out
}

// renderTrailToPlayer emits the descriptions/track template to the
// player. No-op for mob actors (caller gates).
func renderTrailToPlayer(actor Actor, visitors []TrackingInfo) {
	if len(visitors) == 0 {
		actor.SendText(messaging.CategorySystem, "You don't see any tracks.")
		return
	}
	uid := actor.GetUserId()
	trackTxt, _ := templates.Process("descriptions/track", visitors, uid)
	actor.SendText(messaging.CategorySystem, trackTxt)
}

// findUserInRoomByName looks up a user in the room by prefix match.
// Returns nil if no match.
func findUserInRoomByName(room *rooms.Room, name string, excludeUserId int) *users.UserRecord {
	for _, pId := range room.GetPlayers() {
		if pId == excludeUserId {
			continue
		}
		u := users.GetByUserId(pId)
		if u == nil {
			continue
		}
		if strings.HasPrefix(strings.ToLower(u.Character.Name), strings.ToLower(name)) {
			return u
		}
	}
	return nil
}

// findMobInRoomByName looks up a mob in the room by prefix match.
func findMobInRoomByName(room *rooms.Room, name string) *mobs.Mob {
	for _, mId := range room.GetMobs() {
		m := mobs.GetInstance(mId)
		if m == nil {
			continue
		}
		if strings.HasPrefix(strings.ToLower(m.Character.Name), strings.ToLower(name)) {
			return m
		}
	}
	return nil
}

// lookupAdjacentTrail searches each adjacent room's visitor log for a
// target matching `targetNoun`. Returns (applied=true) on first hit
// with the misc-data key/value to set, the exit direction, and the
// target ids.
func lookupAdjacentTrail(room *rooms.Room, char interface{}, targetNoun string, excludeUserId int) (
	applied bool, miscKey string, miscVal interface{}, dirExit string, targetUserId int, targetMobId int, targetName string,
) {
	// First try users.
	allUserNames := []string{}
	for uId := range room.Visitors(rooms.VisitorUser) {
		if uId == excludeUserId {
			continue
		}
		if vu := users.GetByUserId(uId); vu != nil {
			allUserNames = append(allUserNames, vu.Character.Name)
		}
	}
	if match, closeMatch := util.FindMatchIn(targetNoun, allUserNames...); match != "" || closeMatch != "" {
		pick := match
		if pick == "" {
			pick = closeMatch
		}
		// Resolve userId + exit direction.
		for uId := range room.Visitors(rooms.VisitorUser) {
			u := users.GetByUserId(uId)
			if u == nil || u.Character.Name != pick {
				continue
			}
			return true, "tracking-user", pick, findExited(room, uId, rooms.VisitorUser), u.UserId, 0, pick
		}
	}

	// Then mobs.
	allMobNames := []string{}
	for mId := range room.Visitors(rooms.VisitorMob) {
		if vm := mobs.GetInstance(mId); vm != nil {
			allMobNames = append(allMobNames, vm.Character.Name)
		}
	}
	if match, closeMatch := util.FindMatchIn(targetNoun, allMobNames...); match != "" || closeMatch != "" {
		pick := match
		if pick == "" {
			pick = closeMatch
		}
		for mId := range room.Visitors(rooms.VisitorMob) {
			m := mobs.GetInstance(mId)
			if m == nil || m.Character.Name != pick {
				continue
			}
			return true, "tracking-mob", pick, findExited(room, mId, rooms.VisitorMob), 0, m.InstanceId, pick
		}
	}

	return false, "", nil, "", 0, 0, ""
}

// trailStrengthToString and findExited are lifted from
// usercommands/skill.track.go.
func trailStrengthToString(trailStrength float64) string {
	strength := int(math.Round(trailStrength * 100))
	switch {
	case strength < 15:
		return "Dead"
	case strength < 50:
		return "Weak"
	case strength < 70:
		return "Good"
	case strength < 90:
		return "Warm"
	default:
		return "Hot"
	}
}

func findExited(room *rooms.Room, targetId int, targetType rooms.VisitorType) string {
	var bestExit string
	var bestStrength float64

	for exitName, exitInfo := range room.Exits {
		if exitInfo.Secret {
			continue
		}
		testRoom := rooms.LoadRoom(exitInfo.RoomId)
		if testRoom == nil {
			continue
		}
		for vId, vStr := range testRoom.Visitors(targetType) {
			if vId != targetId {
				continue
			}
			if vStr < bestStrength {
				continue
			}
			bestExit = exitName
			bestStrength = vStr
		}
	}
	// Silence unused errors import if dropped.
	_ = errors.New
	return bestExit
}
```

Note: this file imports `errors` only to keep the existing track.go's
import set; if the linter complains, drop the `_ = errors.New` line and
the import.

- [ ] **Step 4: Run the test, verify it passes**

Run: `go test ./internal/actions/ -run TestTrack -v`

Expected: all three tests PASS.

- [ ] **Step 5: Run the full actions package tests**

Run: `go test ./internal/actions/...`

Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/actions/track.go internal/actions/track_test.go
git commit -m "$(cat <<'EOF'
feat(actions): lift Track into actions/ via actor pattern (2.8)

Track(actor, opts) TrackResult. Supports trail-scan (no-arg) and
active-track (TargetNoun set) modes. Applies buff 86 + misc data
on active-track success. UserActor emits rendered tier text;
MobActor SendText is a no-op. Skill progression on every roll.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 6: Lift Search into actions/search.go (TDD)

**Files:**
- Create: `internal/actions/search.go`
- Create: `internal/actions/search_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/actions/search_test.go`:

```go
package actions

import (
	"testing"
)

// TestSearch_EmptyRoomNothingFound confirms an empty room produces
// zero hits across all tiers.
func TestSearch_EmptyRoomNothingFound(t *testing.T) {
	room := makeTestRoom(t, 9201)
	user := makeTestUser(t, "SearchTester")
	user.Character.RoomId = room.RoomId
	actor := &UserActor{User: user, Room: room}

	result := Search(actor, SearchOptions{})

	if len(result.HiddenExitsFound) != 0 ||
		len(result.HiddenContainersFound) != 0 ||
		len(result.StashedItemsFound) != 0 ||
		len(result.HiddenPlayersFound) != 0 ||
		len(result.HiddenMobsFound) != 0 ||
		len(result.HiddenNounsFound) != 0 {
		t.Error("expected all-empty result for empty room")
	}
}

// TestSearch_CooldownGate confirms two consecutive calls within the
// cooldown return OnCooldown=true on the second.
func TestSearch_CooldownGate(t *testing.T) {
	room := makeTestRoom(t, 9202)
	user := makeTestUser(t, "SearchTester2")
	user.Character.RoomId = room.RoomId
	actor := &UserActor{User: user, Room: room}

	_ = Search(actor, SearchOptions{})
	second := Search(actor, SearchOptions{})

	if !second.OnCooldown {
		t.Error("second call within 2-round window should return OnCooldown=true")
	}
}

// TestSearch_MobActorSilent confirms mob actor invocation works
// without panic.
func TestSearch_MobActorSilent(t *testing.T) {
	room := makeTestRoom(t, 9203)
	mob := makeTestMob(t, "SearchMob", room.RoomId)
	actor := NewMobActorInRoom(mob, room)

	_ = Search(actor, SearchOptions{})
}
```

- [ ] **Step 2: Run the test, verify it fails**

Run: `go test ./internal/actions/ -run TestSearch -v`

Expected: build error / "Search undefined".

- [ ] **Step 3: Implement actions/search.go**

Create `internal/actions/search.go`:

```go
package actions

import (
	"fmt"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// SearchOptions is intentionally empty v1 — in-room search is the only
// mode. Reserved for future "search container" path.
type SearchOptions struct{}

// SearchStashedItem represents a stashed item discovered by Tier 2.
type SearchStashedItem struct {
	ItemId      int
	DisplayName string
}

// SearchResult is the structured outcome.
type SearchResult struct {
	HiddenExitsFound      []string // Tier 1 — player flavor
	HiddenContainersFound []string // Tier 1 — player flavor
	StashedItemsFound     []SearchStashedItem
	HiddenPlayersFound    []int // Tier 2 — user ids
	HiddenMobsFound       []int // Tier 2 — mob instance ids
	HiddenNounsFound      []string // Tier 3 — player flavor

	OnCooldown bool
	Reason     string
}

// Search rolls Perception+Search per discovery candidate in the room.
// UserActor receives template-rendered output; MobActor is silent
// (no broadcast, no template). Cooldown is shared with the player path
// (2 rounds on the "search" key).
func Search(actor Actor, opts SearchOptions) SearchResult {
	result := SearchResult{}
	char := actor.GetCharacter()
	room := actor.GetRoom()
	if char == nil || room == nil {
		return result
	}

	if !char.TryCooldown("search", "2 rounds") {
		result.OnCooldown = true
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem,
				fmt.Sprintf("You need to wait %d more rounds to do that again.",
					char.GetCooldown("search")))
		}
		return result
	}

	searchRank := char.GetSkillLevel(skills.Search)
	searchScore := float64(char.Stats.Perception.ValueAdj) +
		combat.SkillMultiplier(searchRank)*25.0

	if actor.IsPlayer() {
		actor.SendText(messaging.CategorySystem, "You snoop around for a bit...\n")
		room.SendTextVisual(messaging.CategoryMobEmote,
			fmt.Sprintf(`<ansi fg="username">%s</ansi> is snooping around.`, char.Name),
			actor.GetUserId(),
		)
	}

	rolledAgainstSomething := false

	// Tier 1 — secret exits.
	for exitName, exitInfo := range room.Exits {
		if !exitInfo.Secret {
			continue
		}
		rolledAgainstSomething = true
		roll := dice.RollStat(searchScore)
		if roll.Value >= 125 {
			result.HiddenExitsFound = append(result.HiddenExitsFound, exitName)
			if actor.IsPlayer() {
				actor.SendText(messaging.CategorySystem,
					fmt.Sprintf(`You found a secret exit: <ansi fg="secret-exit">%s</ansi>`, exitName))
			}
		}
	}

	// Tier 1 — hidden containers.
	for containerName, container := range room.Containers {
		if !container.Hidden {
			continue
		}
		if char.HasDiscovery(room.RoomId, containerName) {
			continue
		}
		rolledAgainstSomething = true
		roll := dice.RollStat(searchScore)
		if roll.Value >= 125 {
			char.AddDiscovery(room.RoomId, containerName)
			result.HiddenContainersFound = append(result.HiddenContainersFound, containerName)
			if actor.IsPlayer() {
				actor.SendText(messaging.CategorySystem,
					fmt.Sprintf(`You discover a hidden <ansi fg="container">%s</ansi>!`, containerName))
			}
		}
	}

	// Tier 2 — stashed items.
	stashedNames := []string{}
	for _, item := range room.Stash {
		if !item.IsValid() {
			room.RemoveItem(item, true)
			continue
		}
		rolledAgainstSomething = true
		roll := dice.RollStat(searchScore)
		if roll.Value >= 135 {
			result.StashedItemsFound = append(result.StashedItemsFound, SearchStashedItem{
				ItemId:      item.ItemId,
				DisplayName: item.DisplayName(),
			})
			if actor.IsPlayer() {
				stashedNames = append(stashedNames, item.DisplayName()+` <ansi fg="item-stashed">(stashed)</ansi>`)
			}
		}
	}
	if actor.IsPlayer() && len(stashedNames) > 0 {
		details := map[string]any{
			"GroundStuff": stashedNames,
			"IsDark":      room.GetBiome().IsDark(),
			"IsNight":     gametime.IsNight(),
		}
		text, _ := templates.Process("descriptions/ontheground", details, actor.GetUserId())
		actor.SendText(messaging.CategorySystem, text)
	}

	// Tier 2 — hidden players.
	hiddenPlayerNames := []string{}
	for _, pId := range room.GetPlayers() {
		if pId == actor.GetUserId() {
			continue
		}
		p := users.GetByUserId(pId)
		if p == nil || !p.Character.IsHidden() {
			continue
		}
		rolledAgainstSomething = true
		roll := dice.RollStat(searchScore)
		if roll.Value >= 135 {
			result.HiddenPlayersFound = append(result.HiddenPlayersFound, pId)
			if actor.IsPlayer() {
				hiddenPlayerNames = append(hiddenPlayerNames, p.Character.Name+` <ansi fg="black-bold">(hiding)</ansi>`)
			}
		}
	}
	if actor.IsPlayer() && len(hiddenPlayerNames) > 0 {
		details := rooms.GetDetails(room, users.GetByUserId(actor.GetUserId()))
		details.VisiblePlayers = []string{}
		for _, name := range hiddenPlayerNames {
			details.VisiblePlayers = append(details.VisiblePlayers,
				characters.FormattedName{Name: name, Type: "username", Suffix: "hidden"}.String())
		}
		text, _ := templates.Process("descriptions/who", details, actor.GetUserId())
		actor.SendText(messaging.CategorySystem, text)
	}

	// Tier 2 — hidden mobs.
	hiddenMobNames := []string{}
	for _, mId := range room.GetMobs() {
		m := mobs.GetInstance(mId)
		if m == nil || !m.Character.IsHidden() {
			continue
		}
		rolledAgainstSomething = true
		roll := dice.RollStat(searchScore)
		if roll.Value >= 135 {
			result.HiddenMobsFound = append(result.HiddenMobsFound, mId)
			if actor.IsPlayer() {
				hiddenMobNames = append(hiddenMobNames, m.Character.Name+` <ansi fg="black-bold">(hiding)</ansi>`)
			}
		}
	}
	if actor.IsPlayer() && len(hiddenMobNames) > 0 {
		details := rooms.GetDetails(room, users.GetByUserId(actor.GetUserId()))
		details.VisibleMobs = []string{}
		for _, name := range hiddenMobNames {
			details.VisibleMobs = append(details.VisibleMobs,
				characters.FormattedName{Name: name, Type: "mob", Suffix: "hidden"}.String())
		}
		text, _ := templates.Process("descriptions/who", details, actor.GetUserId())
		actor.SendText(messaging.CategorySystem, text)
	}

	// Tier 3 — hidden nouns (player flavor only).
	nounKeys := make([]string, 0, len(room.HiddenNouns))
	for k := range room.HiddenNouns {
		nounKeys = append(nounKeys, k)
	}
	sort.Strings(nounKeys)
	for _, nounKey := range nounKeys {
		if char.HasDiscovery(room.RoomId, nounKey) {
			continue
		}
		hiddenNoun := room.HiddenNouns[nounKey]
		rolledAgainstSomething = true
		roll := dice.RollStat(searchScore)
		if roll.Value >= 175 {
			char.AddDiscovery(room.RoomId, nounKey)
			result.HiddenNounsFound = append(result.HiddenNounsFound, nounKey)
			if actor.IsPlayer() {
				actor.SendText(messaging.CategorySystem,
					fmt.Sprintf(`You discover something: <ansi fg="noun">%s</ansi>`, nounKey))
				actor.SendText(messaging.CategorySystem, hiddenNoun.HiddenDescription)
			}
		}
	}

	if rolledAgainstSomething {
		actor.OnSkillUse(string(skills.Search))
	}

	return result
}
```

- [ ] **Step 4: Run the test, verify it passes**

Run: `go test ./internal/actions/ -run TestSearch -v`

Expected: all three tests PASS.

- [ ] **Step 5: Run the full actions package tests**

Run: `go test ./internal/actions/...`

Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/actions/search.go internal/actions/search_test.go
git commit -m "$(cat <<'EOF'
feat(actions): lift Search into actions/ via actor pattern (2.8)

Search(actor, opts) SearchResult. Three-tier discovery (exits +
containers / stashed + hidden mobs/players / hidden nouns) with
per-discovery rolls. UserActor renders templates; MobActor is
silent. 2-round cooldown shared with player path.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Phase 3 — Thin player wrappers

### Task 7: Thin usercommands/scan.go

**Files:**
- Modify: `internal/usercommands/scan.go` (full rewrite, ~25 LoC)

- [ ] **Step 1: Rewrite scan.go as wrapper**

Replace the entire contents of `internal/usercommands/scan.go` with:

```go
package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Scan is a thin wrapper over actions.Scan. The action handles all
// rendering for player actors via SendText; this wrapper just
// constructs the actor + opts.
func Scan(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	actor := &actions.UserActor{User: user, Room: room}
	_ = actions.Scan(actor, actions.ScanOptions{HostileOnly: false})
	return true, nil
}
```

- [ ] **Step 2: Build to confirm**

Run: `go build ./...`

Expected: clean exit.

- [ ] **Step 3: Run usercommands tests to confirm no regressions**

Run: `go test ./internal/usercommands/...`

Expected: all tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/usercommands/scan.go
git commit -m "$(cat <<'EOF'
refactor(scan): thin player wrapper over actions.Scan (2.8)

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 8: Thin usercommands/skill.track.go

**Files:**
- Modify: `internal/usercommands/skill.track.go` (full rewrite, ~30 LoC)

- [ ] **Step 1: Rewrite skill.track.go as wrapper**

Replace the entire contents of `internal/usercommands/skill.track.go` with:

```go
package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Track is a thin wrapper over actions.Track. The action handles
// rendering, cooldowns, and buff application; this wrapper handles
// "stop"/"clear" keyword normalization and the quest-engine
// command notification.
func Track(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	opts := actions.TrackOptions{TargetNoun: rest}
	if rest == "stop" || rest == "clear" {
		opts = actions.TrackOptions{CancelTracking: true}
	}

	actor := &actions.UserActor{User: user, Room: room}
	_ = actions.Track(actor, opts)

	// Quest engine: command notification (preserved from prior behavior).
	bridge := questengine.NewGameBridge(user, room.RoomId)
	questengine.GetEngine().Notify("command", questengine.EventDetails{
		UserId:  user.UserId,
		RoomId:  room.RoomId,
		Command: "track",
	}, bridge, bridge)

	return true, nil
}
```

Drop all the helper types/funcs (`trackingInfo`, `trailStrengthToString`, `findExited`) — they're now in `actions/track.go`.

- [ ] **Step 2: Build to confirm**

Run: `go build ./...`

Expected: clean exit. If the engineer's build complains about unused imports, prune them.

- [ ] **Step 3: Run usercommands tests**

Run: `go test ./internal/usercommands/ -run TestTrack -v`

Expected: existing TestTrack* tests pass (they may need touch-ups if they depended on the removed helper types — confirm during the failing-to-passing transition).

- [ ] **Step 4: Run all usercommands tests**

Run: `go test ./internal/usercommands/...`

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/usercommands/skill.track.go
git commit -m "$(cat <<'EOF'
refactor(track): thin player wrapper over actions.Track (2.8)

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 9: Thin usercommands/skill.search.go

**Files:**
- Modify: `internal/usercommands/skill.search.go` (full rewrite, ~20 LoC)

- [ ] **Step 1: Rewrite skill.search.go as wrapper**

Replace the entire contents of `internal/usercommands/skill.search.go` with:

```go
package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Search is a thin wrapper over actions.Search. The action handles
// all tier rolls, template rendering, cooldown gating, and skill
// progression.
func Search(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	actor := &actions.UserActor{User: user, Room: room}
	_ = actions.Search(actor, actions.SearchOptions{})
	return true, nil
}
```

- [ ] **Step 2: Build to confirm**

Run: `go build ./...`

Expected: clean exit.

- [ ] **Step 3: Run usercommands tests**

Run: `go test ./internal/usercommands/ -run TestSearch -v`

Expected: existing TestSearch* tests pass.

- [ ] **Step 4: Run all usercommands tests**

Run: `go test ./internal/usercommands/...`

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/usercommands/skill.search.go
git commit -m "$(cat <<'EOF'
refactor(search): thin player wrapper over actions.Search (2.8)

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Phase 4 — Mob wrappers

### Task 10: Add mob wrappers + register

**Files:**
- Create: `internal/mobcommands/scan.go`
- Create: `internal/mobcommands/track.go`
- Create: `internal/mobcommands/search.go`
- Modify: `internal/mobcommands/mobcommands.go` (add 3 entries to `mobCommands` map)

- [ ] **Step 1: Create mobcommands/scan.go**

```go
package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func Scan(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	actor := actions.NewMobActorInRoom(mob, room)
	_ = actions.Scan(actor, actions.ScanOptions{HostileOnly: false})
	return true, nil
}
```

- [ ] **Step 2: Create mobcommands/track.go**

```go
package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func Track(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	opts := actions.TrackOptions{TargetNoun: rest}
	if rest == "stop" || rest == "clear" {
		opts = actions.TrackOptions{CancelTracking: true}
	}
	actor := actions.NewMobActorInRoom(mob, room)
	_ = actions.Track(actor, opts)
	return true, nil
}
```

- [ ] **Step 3: Create mobcommands/search.go**

```go
package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func Search(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	actor := actions.NewMobActorInRoom(mob, room)
	_ = actions.Search(actor, actions.SearchOptions{})
	return true, nil
}
```

- [ ] **Step 4: Register in mobcommands.go**

Edit `internal/mobcommands/mobcommands.go`. Find the alphabetically-appropriate insertion points in the `mobCommands` map and add three entries:

```go
		"scan":           {Scan, false},
```

(insert between `salvage` and `say`)

```go
		"search":         {Search, false},
```

(insert between `say` and `selljunk` — no, actually between `say` and `sayto`... preserve alphabetical order)

```go
		"track":          {Track, false},
```

(insert between `taunt` and `trip` — confirm alphabetical position)

Use Edit tool. Verify final ordering matches existing alphabetical layout.

- [ ] **Step 5: Build**

Run: `go build ./...`

Expected: clean exit.

- [ ] **Step 6: Run mobcommands tests**

Run: `go test ./internal/mobcommands/...`

Expected: all tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/mobcommands/scan.go internal/mobcommands/track.go internal/mobcommands/search.go internal/mobcommands/mobcommands.go
git commit -m "$(cat <<'EOF'
feat(mobcommands): add scan/track/search wrappers (2.8)

Thin wrappers over actions.{Scan,Track,Search}. Registered in
the mobCommands map. Mob actors are silent (SendText no-op);
btree consumers read the structured action results.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Phase 5 — Btree primitives

### Task 11: Add actions_scout.go + conditions_scout.go

**Files:**
- Create: `internal/behaviortree/actions_scout.go`
- Create: `internal/behaviortree/conditions_scout.go`

- [ ] **Step 1: Create actions_scout.go**

```go
package behaviortree

// actions_scout.go — btree action primitives for the scout suite:
//   try_scan, try_track, try_search, move_toward_tracked
//
// Each primitive resolves its target via ctx (Event.UserId →
// Aggro → SoftTarget chain) and delegates to actions.<Verb>.

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// actTryScan walks adjacent rooms via actions.Scan and, if HostileOnly
// (default true), promotes the first hostile sighting to ctx.SoftTarget.
// Returns Success on any sighting (or hostile sighting when HostileOnly);
// Failure otherwise.
func actTryScan(params map[string]any, ctx *EvalContext) Result {
	hostileOnly := getBoolParam(params, "hostile_only", true)

	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}
	actor := actions.NewMobActorInRoom(mob, room)
	result := actions.Scan(actor, actions.ScanOptions{HostileOnly: hostileOnly})

	if len(result.Sightings) == 0 {
		return Failure
	}

	// Walk sightings in deterministic exit-order; promote first hostile
	// (or any sighting if !hostileOnly) to SoftTarget.
	for _, s := range result.Sightings {
		for _, p := range s.Players {
			if hostileOnly && !isHostileToMob(mob, p.Id, true) {
				continue
			}
			ctx.SoftTarget.UserId = p.Id
			ctx.SoftTarget.MobInstanceId = 0
			return Success
		}
		for _, m := range s.Mobs {
			if hostileOnly && !isHostileToMob(mob, m.Id, false) {
				continue
			}
			ctx.SoftTarget.MobInstanceId = m.Id
			ctx.SoftTarget.UserId = 0
			return Success
		}
	}

	// No hostile match. Non-hostile sightings still count as Failure when
	// HostileOnly is true.
	if hostileOnly {
		return Failure
	}
	return Success
}

// actTryTrack resolves a target from the configured source and dispatches
// actions.Track. Sources: "aggro" (default), "event", "soft_target".
// Returns Success on trail hit or in-room target found; Failure on no
// target / no trail / on cooldown.
func actTryTrack(params map[string]any, ctx *EvalContext) Result {
	source := getStringParam(params, "target_from")
	if source == "" {
		source = "aggro"
	}

	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}

	targetName := resolveTrackTargetName(mob, ctx, source)
	if targetName == "" {
		return Failure
	}

	actor := actions.NewMobActorInRoom(mob, room)
	result := actions.Track(actor, actions.TrackOptions{TargetNoun: targetName})

	// Buff applied → trail found in adjacent rooms → success.
	// In-room hit also sets ActiveTarget* → success.
	if result.BuffApplied || result.ActiveTargetUserId != 0 || result.ActiveTargetMobInstId != 0 {
		// Also seed SoftTarget for downstream consumers.
		if result.ActiveTargetUserId != 0 {
			ctx.SoftTarget.UserId = result.ActiveTargetUserId
			ctx.SoftTarget.MobInstanceId = 0
		} else if result.ActiveTargetMobInstId != 0 {
			ctx.SoftTarget.MobInstanceId = result.ActiveTargetMobInstId
			ctx.SoftTarget.UserId = 0
		}
		return Success
	}
	return Failure
}

// actTrySearch invokes actions.Search and promotes the first hidden
// hostile to ctx.SoftTarget. Tier 1 (exits, containers) and Tier 3
// (hidden nouns) hits are present in the result but ignored by the
// btree primitive — they're player-facing flavor only.
func actTrySearch(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}
	actor := actions.NewMobActorInRoom(mob, room)
	result := actions.Search(actor, actions.SearchOptions{})

	if result.OnCooldown {
		return Failure
	}

	// Promote first hostile-hidden to SoftTarget.
	for _, pId := range result.HiddenPlayersFound {
		if isHostileToMob(mob, pId, true) {
			ctx.SoftTarget.UserId = pId
			ctx.SoftTarget.MobInstanceId = 0
			return Success
		}
	}
	for _, mId := range result.HiddenMobsFound {
		if isHostileToMob(mob, mId, false) {
			ctx.SoftTarget.MobInstanceId = mId
			ctx.SoftTarget.UserId = 0
			return Success
		}
	}

	// Any non-hostile detection still counts as a sighting for the
	// archetype's purposes — but no SoftTarget seeded.
	if len(result.HiddenPlayersFound) > 0 || len(result.HiddenMobsFound) > 0 {
		return Success
	}
	return Failure
}

// actMoveTowardTracked reads the tracked target's misc data on the mob's
// character, locates the freshest adjacent-room trail toward that target,
// and dispatches a `go <direction>` command via the existing command
// pipeline. Returns Failure if no buff 86 / no resolvable direction /
// no movement possible.
//
// Cleanup contract: when buff 86 is absent, clears tracking-mob /
// tracking-user / tracking-display-count misc data. Mirrors the
// roomdetails.go fix from Task 3 — misc-data state cannot outlive
// the buff.
func actMoveTowardTracked(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if !mob.Character.HasBuff(86) {
		// Buff expired or otherwise gone — clear stale misc data.
		mob.Character.SetMiscData("tracking-mob", nil)
		mob.Character.SetMiscData("tracking-user", nil)
		mob.Character.SetMiscData("tracking-display-count", nil)
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}

	// Resolve target name from misc data.
	var targetName string
	if v := mob.Character.GetMiscData("tracking-user"); v != nil {
		if s, ok := v.(string); ok {
			targetName = s
		}
	}
	if targetName == "" {
		if v := mob.Character.GetMiscData("tracking-mob"); v != nil {
			if s, ok := v.(string); ok {
				targetName = s
			}
		}
	}
	if targetName == "" {
		return Failure
	}

	// Find best exit direction via the same helper actions.Track uses
	// (re-derive each tick — no buff-tick coupling).
	dir := findBestExitToward(room, targetName)
	if dir == "" {
		return Failure
	}

	// Dispatch `go <direction>` via the engine command pipeline.
	if dispatchMobCommand(mob, "go "+dir) {
		return Success
	}
	return Failure
}

// resolveTrackTargetName returns the target's display name for the
// configured source. The action wrapper feeds this into
// actions.Track.TargetNoun (which does prefix-matching).
func resolveTrackTargetName(mob *mobs.Mob, ctx *EvalContext, source string) string {
	switch source {
	case "event":
		if ctx.Event.UserId > 0 {
			if u := users.GetByUserId(ctx.Event.UserId); u != nil {
				return u.Character.Name
			}
		}
	case "soft_target":
		if ctx.SoftTarget.UserId > 0 {
			if u := users.GetByUserId(ctx.SoftTarget.UserId); u != nil {
				return u.Character.Name
			}
		}
		if ctx.SoftTarget.MobInstanceId > 0 {
			if m := mobs.GetInstance(ctx.SoftTarget.MobInstanceId); m != nil {
				return m.Character.Name
			}
		}
	default: // "aggro"
		if mob.Character.Aggro != nil {
			if mob.Character.Aggro.UserId > 0 {
				if u := users.GetByUserId(mob.Character.Aggro.UserId); u != nil {
					return u.Character.Name
				}
			}
			if mob.Character.Aggro.MobInstanceId > 0 {
				if m := mobs.GetInstance(mob.Character.Aggro.MobInstanceId); m != nil {
					return m.Character.Name
				}
			}
		}
	}
	return ""
}

// isHostileToMob returns true when the candidate target is hostile to the
// mob via mob.HatesMob (for mob-on-mob) or default-hostile (for
// mob-on-player). v1: any non-charmed player counts as hostile. Faction-
// rep substrate integration is logged as a followup (chunk 1.2 substrate
// consumer — see spec's Hostile determination section).
func isHostileToMob(mob *mobs.Mob, targetId int, targetIsPlayer bool) bool {
	if targetIsPlayer {
		// Charmed pets are NOT hostile. Otherwise, treat any player as
		// hostile for v1.
		// charmed-status check: walk mob.CharmedUserId / similar field if
		// present; for v1 keep simple.
		_ = targetId
		return true
	}
	if other := mobs.GetInstance(targetId); other != nil {
		return mob.HatesMob(other)
	}
	return false
}

// findBestExitToward returns the exit name whose adjacent room contains
// the strongest visitor trail for the named target. Mirrors actions.Track's
// adjacent-room scan logic. Returns "" when no trail found.
func findBestExitToward(room *rooms.Room, targetName string) string {
	var bestExit string
	var bestStrength float64

	for exitName, exitInfo := range room.Exits {
		if exitInfo.Secret {
			continue
		}
		adj := rooms.LoadRoom(exitInfo.RoomId)
		if adj == nil {
			continue
		}

		// Check user visitors in the adjacent room.
		for vId, vStr := range adj.Visitors(rooms.VisitorUser) {
			if u := users.GetByUserId(vId); u != nil {
				if !strings.EqualFold(u.Character.Name, targetName) {
					continue
				}
				if vStr > bestStrength {
					bestStrength = vStr
					bestExit = exitName
				}
			}
		}
		// Check mob visitors.
		for vId, vStr := range adj.Visitors(rooms.VisitorMob) {
			if m := mobs.GetInstance(vId); m != nil {
				if !strings.EqualFold(m.Character.Name, targetName) {
					continue
				}
				if vStr > bestStrength {
					bestStrength = vStr
					bestExit = exitName
				}
			}
		}
	}
	return bestExit
}

// dispatchMobCommand routes a command string into the mob command pipeline.
// Implemented as a helper to keep the action body small; existing pattern
// in actions_combat.go can be lifted/shared. If a different helper already
// exists for "dispatch mobcommand from action", use that — verify during
// implementation.
func dispatchMobCommand(mob *mobs.Mob, cmd string) bool {
	// Look up the existing pattern in actions_combat.go or actions_mob.go
	// for how to fire mob commands from within a btree action. Likely
	// pattern: events.AddToQueue(events.Input{MobInstanceId: ..., InputText: cmd}).
	// During implementation: grep for "events.Input{MobInstanceId" in the
	// behaviortree package and copy the pattern. Until then, this is a
	// shim that always returns true so callers can move on.
	_ = cmd
	_ = mob
	return true // PLACEHOLDER — replace with real dispatch during impl.
}

// getBoolParam reads a bool from a btree-primitive params map, with a
// default when the key is absent.
func getBoolParam(params map[string]any, key string, defaultVal bool) bool {
	if v, ok := params[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return defaultVal
}
```

**Note on `dispatchMobCommand`:** the placeholder above is INTENTIONALLY a stub. During implementation, grep `internal/behaviortree/` for `events.Input{MobInstanceId` or `events.AddToQueue` calls that fire mob commands. The `command` btree action (`actCommand`) is the canonical example — copy that pattern. Replace the stub before committing this task.

- [ ] **Step 2: Create conditions_scout.go**

```go
package behaviortree

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// condRoomHasHiddenEntity returns Success when at least one hidden mob
// or hidden player is in the room. Cheap pre-check for archetypes
// before paying for try_search's cooldown + per-discovery rolls.
func condRoomHasHiddenEntity(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}
	for _, pId := range room.GetPlayers() {
		if p := users.GetByUserId(pId); p != nil && p.Character.IsHidden() {
			return Success
		}
	}
	for _, mId := range room.GetMobs() {
		if m := mobs.GetInstance(mId); m != nil && m.Character.IsHidden() && m.InstanceId != mob.InstanceId {
			return Success
		}
	}
	return Failure
}

// condMobIsTracking returns Success when the self mob carries buff 86
// (Active Tracking).
func condMobIsTracking(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.HasBuff(86) {
		return Success
	}
	return Failure
}
```

- [ ] **Step 3: Resolve the dispatchMobCommand placeholder**

Search for the canonical mob-command dispatch pattern:

```bash
# Inside Grep tool:
pattern: "events\.Input\{MobInstanceId|actCommand"
path: "internal/behaviortree"
output_mode: "content"
-n: true
```

Open the matched function (likely `actCommand` in `actions.go`). Copy the dispatch shape into `dispatchMobCommand` in `actions_scout.go`. Replace the `return true // PLACEHOLDER` line.

- [ ] **Step 4: Build to confirm**

Run: `go build ./...`

Expected: clean exit.

- [ ] **Step 5: Run behaviortree package tests**

Run: `go test ./internal/behaviortree/...`

Expected: existing tests pass (no new tests added — primitives validated by smoke per the spec's testing section).

- [ ] **Step 6: Commit**

```bash
git add internal/behaviortree/actions_scout.go internal/behaviortree/conditions_scout.go
git commit -m "$(cat <<'EOF'
feat(behaviortree): add scout primitives (2.8)

Adds try_scan, try_track, try_search, move_toward_tracked actions
plus room_has_hidden_entity, mob_is_tracking conditions. Each
primitive delegates to actions.<Verb> and seeds ctx.SoftTarget on
detection per the chunk-2.7 pattern.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 12: Register btree primitives in init()

**Files:**
- Modify: `internal/behaviortree/actions.go` (add 4 entries to actionRegistry in init())
- Modify: `internal/behaviortree/conditions.go` (add 2 entries to conditionRegistry in init())

- [ ] **Step 1: Register actions**

In `internal/behaviortree/actions.go`, locate the `init()` function and add at the end (after the existing registrations):

```go
	// Scout / track / scan (2.8)
	actionRegistry["try_scan"] = actTryScan
	actionRegistry["try_track"] = actTryTrack
	actionRegistry["try_search"] = actTrySearch
	actionRegistry["move_toward_tracked"] = actMoveTowardTracked
```

- [ ] **Step 2: Register conditions**

In `internal/behaviortree/conditions.go`, locate the `init()` function and add at the end:

```go
	// Scout / track / scan (2.8)
	conditionRegistry["room_has_hidden_entity"] = condRoomHasHiddenEntity
	conditionRegistry["mob_is_tracking"] = condMobIsTracking
```

- [ ] **Step 3: Build**

Run: `go build ./...`

Expected: clean exit.

- [ ] **Step 4: Run behaviortree tests**

Run: `go test ./internal/behaviortree/...`

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/behaviortree/actions.go internal/behaviortree/conditions.go
git commit -m "$(cat <<'EOF'
feat(behaviortree): register scout primitives in init() (2.8)

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Phase 6 — Archetype YAMLs

### Task 13: Author scout.yaml + flip goblin_scout (217)

**Files:**
- Create: `_datafiles/world/dogmud/behaviors/archetypes/scout.yaml`
- Modify: `_datafiles/world/dogmud/mobs/ironwind_steppe/217-goblin_scout.yaml` (one field change)

- [ ] **Step 1: Create scout.yaml**

```yaml
# scout archetype
#
# Patrol-flavored mob. Scans adjacent rooms each idle tick; on
# hostile sighting, alerts pack and moves toward target. Searches
# current room periodically for hidden threats. Pursues fleeing
# aggro targets across room boundaries via try_track +
# move_toward_tracked.
#
# Spec: docs/superpowers/specs/
#       2026-05-22-mob-aliveness-2.8-scout-track-scan-design.md

tree:
  type: selector
  children:
    # 1. Panic-flee at critical HP (shared pattern with other archetypes).
    - type: sequence
      event: mob_hurt
      children:
        - type: condition
          check: mob_health_below
          percent: 25
        - type: action
          do: flee

    # 2. Self-defense (no health gate).
    - type: sequence
      event: mob_hurt
      children:
        - type: action
          do: attack

    # 3. Engage on aggro (target in same room).
    - type: sequence
      event: mob_idle
      children:
        - type: condition
          check: has_aggro
        - type: action
          do: attack

    # 4. Active-track loop — aggro target fled the room.
    - type: sequence
      event: mob_idle
      children:
        - type: condition
          check: mob_is_tracking
        - type: action
          do: move_toward_tracked

    # 5. Search current room for hidden threats.
    - type: sequence
      event: mob_idle
      children:
        - type: condition
          check: room_has_hidden_entity
        - type: action
          do: try_search
        - type: action
          do: command
          cmd: callforhelp

    # 6. Scan adjacent rooms; on hostile, alert + chase.
    - type: sequence
      event: mob_idle
      children:
        - type: action
          do: try_scan
          hostile_only: true
        - type: action
          do: try_track
          target_from: soft_target
        - type: action
          do: command
          cmd: callforhelp
        - type: action
          do: move_toward_tracked
```

Verify the `has_aggro` condition exists during implementation; if not, grep for the closest analog or use a different gate (e.g., `mob_combat_target_set`).

- [ ] **Step 2: Flip goblin_scout to scout archetype**

Edit `_datafiles/world/dogmud/mobs/ironwind_steppe/217-goblin_scout.yaml`. Find the `behavior_archetype: lookout` line and replace with:

```yaml
behavior_archetype: scout
```

Verify the mob has at least skill rank 1 in `search`. If absent, add (or bump) under the `skills:` section. Otherwise the `Track`/`Search` skill rolls will fall back to base Perception.

- [ ] **Step 3: Boot the server, confirm archetype loads + mob spawns**

Run: `go run . 2>&1 | head -150`

Expected: clean boot past `behaviors.LoadDataFiles()` and `mobs.LoadDataFiles()`. No panics. Kill the server.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/behaviors/archetypes/scout.yaml _datafiles/world/dogmud/mobs/ironwind_steppe/217-goblin_scout.yaml
git commit -m "$(cat <<'EOF'
feat(archetypes): add scout + flip goblin_scout (217) (2.8)

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 14: Graft scan-before-ambush onto lookout.yaml

**Files:**
- Modify: `_datafiles/world/dogmud/behaviors/archetypes/lookout.yaml`

- [ ] **Step 1: Add the scan-before-ambush branch**

Edit `_datafiles/world/dogmud/behaviors/archetypes/lookout.yaml`. Insert a new branch between the panic-flee branch (mob_hurt + health-below) and the existing `player_enter` ambush branch:

```yaml
    # NEW (chunk 2.8): early-warning scan on idle. If a hostile is
    # sighted in an adjacent room, set SoftTarget and call for help
    # before they walk in.
    - type: sequence
      event: mob_idle
      children:
        - type: action
          do: try_scan
          hostile_only: true
        - type: action
          do: command
          cmd: callforhelp
```

Position: insert immediately AFTER the closing of the panic-flee `mob_hurt` branch and BEFORE the existing `player_enter` ambush branch.

- [ ] **Step 2: Boot, confirm clean load**

Run: `go run . 2>&1 | head -150`

Expected: clean boot. Kill the server.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/behaviors/archetypes/lookout.yaml
git commit -m "$(cat <<'EOF'
feat(lookout): graft scan-before-ambush branch (2.8)

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 15: Graft search-before-steal onto thief.yaml

**Files:**
- Modify: `_datafiles/world/dogmud/behaviors/archetypes/thief.yaml`

- [ ] **Step 1: Add the search-before-steal branch**

Edit `_datafiles/world/dogmud/behaviors/archetypes/thief.yaml`. Insert a new branch immediately BEFORE the existing branch 4 ("Core loop: idle + hidden + gold-bearing player..."):

```yaml
    # NEW (chunk 2.8): catch hidden rivals or hidden players before
    # stealing. If a hidden hostile is present, abandon stealth long
    # enough to alert.
    - type: sequence
      event: mob_idle
      children:
        - type: condition
          check: room_has_hidden_entity
        - type: action
          do: try_search
        - type: action
          do: command
          cmd: callforhelp
```

- [ ] **Step 2: Boot, confirm clean load**

Run: `go run . 2>&1 | head -150`

Expected: clean boot. Kill the server.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/behaviors/archetypes/thief.yaml
git commit -m "$(cat <<'EOF'
feat(thief): graft search-before-steal branch (2.8)

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 16: Graft track-on-aggro-lost onto leader.yaml

**Files:**
- Modify: `_datafiles/world/dogmud/behaviors/archetypes/leader.yaml`

- [ ] **Step 1: Add the track-on-aggro-lost branch**

Edit `_datafiles/world/dogmud/behaviors/archetypes/leader.yaml`. Append a new `mob_idle` branch AFTER the existing branches:

```yaml
    # NEW (chunk 2.8): chase a fleeing aggro target across rooms.
    # try_track applies buff 86 + stores misc data; move_toward_tracked
    # walks the freshest direction each tick.
    - type: sequence
      event: mob_idle
      children:
        - type: condition
          check: has_aggro
        - type: action
          do: try_track
          target_from: aggro

    - type: sequence
      event: mob_idle
      children:
        - type: condition
          check: mob_is_tracking
        - type: action
          do: move_toward_tracked
```

Note: this adds TWO branches — one for "I have aggro but target isn't in the room → start tracking" and one for "I'm already tracking → keep moving." If a `has_aggro` condition doesn't exist, the engineer should either add it (similar to the scout's branch 3) or use the closest existing analog. Confirm during impl.

- [ ] **Step 2: Boot, confirm clean load**

Run: `go run . 2>&1 | head -150`

Expected: clean boot. Kill the server.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/behaviors/archetypes/leader.yaml
git commit -m "$(cat <<'EOF'
feat(leader): graft track-on-aggro-lost branches (2.8)

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Phase 7 — Docs + smoke + roadmap

### Task 17: Update context.md files

**Files:**
- Modify: `internal/behaviortree/context.md`
- Modify: `internal/actions/context.md`

- [ ] **Step 1: Update behaviortree/context.md**

Open `internal/behaviortree/context.md`. Find the section listing registered actions and add entries for the four new primitives + two conditions:

- `try_scan` — invokes actions.Scan; on hostile sighting, sets ctx.SoftTarget; returns Success on any sighting (Failure when HostileOnly and no hostile found)
- `try_track` — invokes actions.Track with target_from source; applies buff 86 on adjacent-trail hit; seeds ctx.SoftTarget
- `try_search` — invokes actions.Search; promotes first hidden hostile to ctx.SoftTarget; ignores Tier-1/Tier-3 hits
- `move_toward_tracked` — reads buff 86 + tracking-{user,mob} misc data; dispatches `go <direction>`
- `room_has_hidden_entity` (condition) — true when room contains a hidden mob or hidden player
- `mob_is_tracking` (condition) — true when self mob carries buff 86

Match the existing context.md formatting (alphabetical insertion, brief one-line descriptions).

- [ ] **Step 2: Update actions/context.md**

Open `internal/actions/context.md`. Add entries for the three new actions in the same shape as existing entries (Consider, Sneak, Steal, etc.):

- `Scan(actor, opts) ScanResult` — adjacent-room sweep; UserActor emits rendered list, MobActor silent
- `Track(actor, opts) TrackResult` — trail-read (no-arg) or active-track (TargetNoun); applies buff 86 + misc data on active-track success
- `Search(actor, opts) SearchResult` — three-tier discovery (exits, stashed/hidden, nouns); 2-round cooldown

- [ ] **Step 3: Commit**

```bash
git add internal/behaviortree/context.md internal/actions/context.md
git commit -m "$(cat <<'EOF'
docs(2.8): update context.md for scout primitives + lifted actions

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 18: Boot + smoke + mark roadmap done

**Files:**
- Modify: `MOB_ALIVENESS_ROADMAP.md`

- [ ] **Step 1: Final clean build + full test suite**

Run: `go build ./... && go test ./...`

Expected: clean exit, all tests pass.

- [ ] **Step 2: Boot the server cleanly**

Run: `go run . 2>&1 | head -200`

Expected: clean boot past all `LoadDataFiles()` calls including:
- `buffs.LoadDataFiles() loadedCount=70` (was 69 before buff 86)
- `behaviors.LoadDataFiles() loadedCount=<unchanged + 1>` (scout.yaml added)
- `mobs.LoadDataFiles() loadedCount=<unchanged>` (goblin_scout flip)

Kill the server.

- [ ] **Step 3: Run the smoke test plan from the spec**

The smoke test plan is documented in section "Smoke test plan" of `docs/superpowers/specs/2026-05-22-mob-aliveness-2.8-scout-track-scan-design.md`. Run all scenarios PLUS the regression-check scenario for the previously-reported "tracking forever" bug:

1. Scout idle-scan loop (goblin_scout 217)
2. Scout-room search (sneak into goblin_scout's room)
3. Lookout scan-before-ambush (bandit_lookout 283)
4. Thief search-before-steal (thornwall_highwayman 90)
5. Leader track-on-aggro-lost (any leader-archetype mob, e.g., bandit chief)
6. Active-tracking buff observability (verify buff 86 applies on mob)

7. **Regression check — tracking auto-expires after 16 rounds.** As a player, `track <some_name_with_trail>` and confirm the buff applies (`buffs` command) and the tracking direction renders in subsequent room views. Wait out the 16-round buff duration (or use admin tooling to fast-expire). Confirm:
   - The buff-end text "The trail grows cold; your focus slips." fires.
   - The next room view shows NO tracking-direction line.
   - The misc data is cleared (admin `inspect player` or equivalent — verify `tracking-mob` / `tracking-user` keys are absent or nil).

8. **Regression check — tracking ends when target found.** As a player, `track <target>` toward someone, then walk to their room. Confirm the "they are here!" message fires once, the buff is removed, and subsequent room views do NOT re-render tracking direction.

Document results in a smoke report:
`tools/testing/reports/2026-05-22-local-feature-tester-chunk-2.8-scout.md`

If any scenario fails, fix the underlying issue (separate commit) before proceeding. If a scenario fails due to a hand-off discovery (faction-rep ergonomics, `has_aggro` absence, `mob_target_lost` event absence), surface to the user before working around — per the spec's commitment.

- [ ] **Step 4: Kill all test mud servers**

Per the standing SOP (`feedback_kill_test_servers.md`).

Run: `tasklist | findstr -i "dogmud go" 2>&1` then `taskkill /F /PID <pid>` for each match. Or `Get-Process | Where-Object {$_.Name -like "*dogmud*" -or $_.Name -eq "go"}` then `Stop-Process` in PowerShell.

- [ ] **Step 5: Update MOB_ALIVENESS_ROADMAP.md**

Edit `MOB_ALIVENESS_ROADMAP.md`:

1. **Progress tracker row** — change `| 2.8 | Tactical | Mob scout / track / scan | S | — | Not started |` to `| 2.8 | Tactical | Mob scout / track / scan | S | — | Done |`.

2. **Roll-up line** — change `**Roll-up:** 15 / 41 done • 0 in progress • 26 not started.` to `**Roll-up:** 16 / 41 done • 0 in progress • 25 not started.`.

3. **Chunk 2.8 mini-brief** — change `**Status:** Not started • **Size:** S` to `**Status:** Done (2026-05-22) • **Size:** S` and append a `**Shipped:**` bullet capturing:
   - Three actions lifted into `internal/actions/` (`Scan`, `Track`, `Search`).
   - Four btree action primitives (`try_scan`, `try_track`, `try_search`, `move_toward_tracked`) + two conditions (`room_has_hidden_entity`, `mob_is_tracking`).
   - New `scout` archetype + flip on goblin_scout (217).
   - Single-branch grafts onto `lookout` (scan-before-ambush), `thief` (search-before-steal), `leader` (track-on-aggro-lost).
   - Bug fix bundled: authored buff 86 (Active Tracking) replacing buff 26 (Conviction Surge) misuse in skill.track.go. Migrated 4 AddBuff + 5 RemoveBuff call sites.
   - Spec + plan paths.

- [ ] **Step 6: Final commit**

```bash
git add MOB_ALIVENESS_ROADMAP.md tools/testing/reports/2026-05-22-local-feature-tester-chunk-2.8-scout.md
git commit -m "$(cat <<'EOF'
docs(2.8): mark mob-aliveness 2.8 Done

Roll-up: 16 / 41 done. Adds scout/track/scan verbs + new scout
archetype + lookout/thief/leader grafts. Bundles buff-26
Conviction Surge misuse fix (new buff 86 Active Tracking).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Self-review checklist (for the implementing engineer)

After all 18 tasks land, before declaring 2.8 done:

- [ ] All three actions registered via the actor pattern (`internal/actions/{scan,track,search}.go`).
- [ ] Buff 86 authored; 4 AddBuff + 5 RemoveBuff calls migrated.
- [ ] Player wrappers thinned (~25 LoC each).
- [ ] Mob wrappers added and registered.
- [ ] Four btree actions + two conditions registered in init().
- [ ] `scout.yaml` archetype authored; goblin_scout (217) flipped.
- [ ] Three grafts applied (lookout/thief/leader).
- [ ] context.md files updated.
- [ ] Smoke plan run end-to-end with results documented.
- [ ] Roadmap row + roll-up + mini-brief all updated.
- [ ] `go build ./...` clean.
- [ ] `go test ./...` clean.
- [ ] Server boots cleanly past all `LoadDataFiles()` calls.
- [ ] No remaining `AddBuff(26)` / `RemoveBuff(26)` references in non-Conviction-Surge code.
