# Bounty Hunting 5.2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** NPCs hunt wanted players (a scaled, geared bounty-hunter dispatched when a player's bounty crosses a threshold, who tracks them down and claims the kill — clearing the record like serving a sentence), plus authored standing bounties players can claim on notable NPCs.

**Architecture:** A global per-round **dispatch sweep** (`internal/bountyhunter`) spawns/scales/telegraphs one hunter per over-threshold player and owns the hunt lifecycle. Per-hunter **pursuit** reuses the 4.x strategic layer — a `hunt_bounty_target` goal + planner driving `pathto`/engage via the `bounty_hunter` archetype's `try_goal_planner`. Claim reuses the existing `PlayerDeath_BountyResolve` guard-claim path (hunter carries the issuer faction) plus an extracted `justice.ClearFactionRecord`. Gear reuses the planar-oasis affix path (`GenerateAffixedItem`+`Wear`).

**Tech Stack:** Go (`internal/bountyhunter`, `internal/goals`, `internal/planners`, `internal/justice`, `internal/hooks`, `internal/configs`), YAML data (mob/item/archetype/faction/bounty seed), Go `testing`.

**Spec:** `docs/superpowers/specs/2026-05-30-mob-aliveness-5.2-bounty-hunting-design.md`

**Conventions for every task:**
- Build: `go build ./...`
- Boot smoke (per CLAUDE.md SOP): `rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*` then run the server binary and confirm it loads past data files without panic (`factions.LoadAllDefinitions`, `mobs.LoadDataFiles() loadedCount`, `bounties` load, `Server Ready`). To run non-interactively: `go build -o /tmp/dogmud_bt.exe . && timeout 40 /tmp/dogmud_bt.exe 2>&1 | grep -iE "loadedCount|panic|fatal|Server Ready" | head -30`.
- Each task ends green + committed. Stage only the files the task lists (the working tree has unrelated runtime churn — never `git add -A`).

---

## Task 1: Config knobs

**Files:**
- Modify: `internal/configs/config.balance.go` (add fields near the `// ── BOUNTIES ──` group)
- Modify: `internal/configs/config.balance.misc.go` (add defaults in `validateMisc()` near the bounty defaults)
- Test: `internal/configs/config_balance_test.go` (create if absent; else add to the existing balance test file — grep for one first)

- [ ] **Step 1: Write the failing test**

Add (match the package clause of existing config tests — likely `package configs`):

```go
func TestBalance_BountyHunterDefaults(t *testing.T) {
	b := &Balance{}
	b.validateMisc()
	if b.BountyHunterGoldThreshold != 500 {
		t.Fatalf("BountyHunterGoldThreshold = %d, want 500", int(b.BountyHunterGoldThreshold))
	}
	if b.BountyHunterBaseStatpool != 250 {
		t.Fatalf("BountyHunterBaseStatpool = %d, want 250", int(b.BountyHunterBaseStatpool))
	}
	if b.BountyHunterStatpoolPerGold != 0.25 {
		t.Fatalf("BountyHunterStatpoolPerGold = %v, want 0.25", float64(b.BountyHunterStatpoolPerGold))
	}
	if b.BountyHunterMinStatpool != 300 || b.BountyHunterMaxStatpool != 500 {
		t.Fatalf("min/max statpool = %d/%d, want 300/500", int(b.BountyHunterMinStatpool), int(b.BountyHunterMaxStatpool))
	}
	if b.BountyHunterRepathRounds != 5 {
		t.Fatalf("BountyHunterRepathRounds = %d, want 5", int(b.BountyHunterRepathRounds))
	}
	if b.BountyHunterRedispatchCooldown != 500 {
		t.Fatalf("BountyHunterRedispatchCooldown = %d, want 500", int(b.BountyHunterRedispatchCooldown))
	}
	if b.BountyHunterGearGoldDivisor != 5 {
		t.Fatalf("BountyHunterGearGoldDivisor = %d, want 5", int(b.BountyHunterGearGoldDivisor))
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/configs/ -run TestBalance_BountyHunterDefaults -v`
Expected: FAIL — undefined fields.

- [ ] **Step 3: Add the struct fields**

In `internal/configs/config.balance.go`, after the `BountyGoldFloor` field, add:

```go
	// ── BOUNTY HUNTING (5.2) ────────────────────────────────────────────
	BountyHunterGoldThreshold     ConfigInt   `yaml:"BountyHunterGoldThreshold"`     // Single open bounty >= this dispatches a hunter (default 500)
	BountyHunterBaseStatpool      ConfigInt   `yaml:"BountyHunterBaseStatpool"`      // Base of hunter scaled statpool (default 250)
	BountyHunterStatpoolPerGold   ConfigFloat `yaml:"BountyHunterStatpoolPerGold"`   // Statpool added per gold of triggering bounty (default 0.25)
	BountyHunterMinStatpool       ConfigInt   `yaml:"BountyHunterMinStatpool"`       // Clamp floor for hunter statpool (default 300)
	BountyHunterMaxStatpool       ConfigInt   `yaml:"BountyHunterMaxStatpool"`       // Clamp ceiling for hunter statpool (default 500)
	BountyHunterRepathRounds      ConfigInt   `yaml:"BountyHunterRepathRounds"`      // Rounds between pursuit re-paths (default 5)
	BountyHunterRedispatchCooldown ConfigInt  `yaml:"BountyHunterRedispatchCooldown"` // Rounds before re-dispatch after a hunter dies (default 500)
	BountyHunterGearGoldDivisor   ConfigInt   `yaml:"BountyHunterGearGoldDivisor"`   // gearGold = statpool / this, fed to GenerateAffixedItem (default 5)
```

- [ ] **Step 4: Add the defaults**

In `internal/configs/config.balance.misc.go` `validateMisc()`, after the `BountyGoldFloor` default block, add:

```go
	// ── BOUNTY HUNTING (5.2) ────────────────────────────────────────────
	if b.BountyHunterGoldThreshold <= 0 {
		b.BountyHunterGoldThreshold = 500
	}
	if b.BountyHunterBaseStatpool <= 0 {
		b.BountyHunterBaseStatpool = 250
	}
	if b.BountyHunterStatpoolPerGold <= 0 {
		b.BountyHunterStatpoolPerGold = 0.25
	}
	if b.BountyHunterMinStatpool <= 0 {
		b.BountyHunterMinStatpool = 300
	}
	if b.BountyHunterMaxStatpool <= 0 {
		b.BountyHunterMaxStatpool = 500
	}
	if b.BountyHunterRepathRounds <= 0 {
		b.BountyHunterRepathRounds = 5
	}
	if b.BountyHunterRedispatchCooldown <= 0 {
		b.BountyHunterRedispatchCooldown = 500
	}
	if b.BountyHunterGearGoldDivisor <= 0 {
		b.BountyHunterGearGoldDivisor = 5
	}
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `go test ./internal/configs/ -run TestBalance_BountyHunterDefaults -v` → PASS. Then `go build ./...`.

- [ ] **Step 6: Commit**

```bash
git add internal/configs/config.balance.go internal/configs/config.balance.misc.go internal/configs/config_balance_test.go
git commit -m "feat(config): bounty-hunter balance knobs (5.2)"
```

---

## Task 2: Extract `justice.ClearFactionRecord`

The clearing logic in `ResolveDetention` (resolve crimes for faction+allies, withdraw bounties, reset rep floor) must be reusable by the bounty-kill path. Extract it to an exported function and have `ResolveDetention` call it.

**Files:**
- Modify: `internal/justice/arrest.go`
- Test: `internal/justice/arrest_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestClearFactionRecord_ResolvesCrimesWithdrawsBountiesResetsRep(t *testing.T) {
	// Seed seams: one unresolved crime + one open bounty for faction "f" against user 7,
	// rep below floor. Mirror the setup used by existing ResolveDetention tests
	// (grep arrest_test.go for how aCrimesForFactionFn/aOpenBountiesFn/aResolveCrimeFn/
	// aWithdrawFn/aGetRepFn/aSetRepFn/aRepResetFn/alliesFn are overridden) and assert
	// ClearFactionRecord("f", 7) calls resolve on the crime, withdraw on the bounty,
	// and sets rep to the floor.
	// (Use the existing test scaffold's recorded-call capture pattern.)
}
```
(Author this against the actual seam-override scaffold present in `arrest_test.go` — the existing `ResolveDetention` tests already wire every seam; copy that setup and assert the three effects fire for the faction + its allies.)

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/justice/ -run TestClearFactionRecord -v` → FAIL (undefined `ClearFactionRecord`).

- [ ] **Step 3: Extract the function**

In `internal/justice/arrest.go`, lift the crime-resolve + bounty-withdraw + rep-reset block out of `ResolveDetention` into:

```go
// ClearFactionRecord clears a player's standing with a faction and its allies:
// resolves their unresolved crimes, withdraws the faction set's open bounties
// against them, and resets rep to the floor where currently below it. Shared by
// serving a sentence (ResolveDetention) and a bounty-hunter claim (5.2).
func ClearFactionRecord(faction string, userId int) {
	factionSet := map[string]bool{faction: true}
	for _, a := range alliesFn(faction) {
		factionSet[a] = true
	}
	for f := range factionSet {
		for _, c := range aCrimesForFactionFn(f, false) {
			if c.Perpetrator.Type == crimes.PerpPlayer && c.Perpetrator.Id == userId {
				aResolveCrimeFn(f, c.Id, "record cleared")
			}
		}
	}
	for _, b := range aOpenBountiesFn(userId) {
		if b.Issuer.Type == bounties.IssuerFaction && factionSet[b.Issuer.Id] {
			aWithdrawFn(b.Id)
		}
	}
	floor := aRepResetFn()
	for f := range factionSet {
		if aGetRepFn(f, userId) < floor {
			aSetRepFn(f, userId, floor)
		}
	}
}
```

Then replace that block in `ResolveDetention` with a single call `ClearFactionRecord(faction, userId)` (keep the `faction` read, buff removal, miscdata clear, and the release-room move exactly as they are).

- [ ] **Step 4: Run tests**

Run: `go test ./internal/justice/ -v` → all PASS (existing ResolveDetention tests still pass + new one). `go build ./...`.

- [ ] **Step 5: Commit**

```bash
git add internal/justice/arrest.go internal/justice/arrest_test.go
git commit -m "refactor(justice): extract ClearFactionRecord for reuse by bounty kills"
```

---

## Task 3: Bounty-hunter claim clears the record on player death

Extend `PlayerDeath_BountyResolve` so when a hunter mob (carrying the issuer faction — the existing `killGuard` path) claims a player bounty, it ALSO clears the player's record with that faction (death pays the debt).

**Files:**
- Modify: `internal/hooks/PlayerDeath_BountyResolve.go`
- Test: `internal/hooks/PlayerDeath_BountyResolve_test.go` (add to existing if present)

- [ ] **Step 1: Write the failing test**

Add a test asserting that in the `killGuard` branch, after a successful `TryClaim`, `justice.ClearFactionRecord(issuerFaction, targetUserId)` is invoked. The existing test file already exercises `attributeBountyKill`/the resolve closure — extend it: stub a hunter mob killer belonging to faction "f", an open faction-"f" bounty on the target, and assert (via a justice seam or a recorded call) that the faction record is cleared. (If `ClearFactionRecord` isn't easily stubbable from hooks tests, assert behavior through the justice seams the test can set, mirroring how other hook tests observe justice effects.)

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/hooks/ -run BountyResolve -v` → FAIL.

- [ ] **Step 3: Implement**

In `internal/hooks/PlayerDeath_BountyResolve.go`, in the `case killGuard:` branch, after the successful `TryClaim` + gold transfer, add a record-clear for faction-issued bounties:

```go
				case killGuard:
					inst := mobs.GetInstance(bk.mobInstanceId)
					if inst == nil {
						bounties.MarkExpired(b.Id)
						continue
					}
					if _, ok := bounties.TryClaim(b.Id, knowledge.MobSubject(int(inst.MobId))); ok {
						inst.Character.Gold += b.GoldReward
						// 5.2: a faction-dispatched hunter's kill clears the
						// player's record with that faction (death pays the
						// debt — same clearing as serving a sentence).
						if issuerFaction != "" {
							justice.ClearFactionRecord(issuerFaction, userId)
						}
					} else {
						bounties.MarkExpired(b.Id)
					}
```

Add the `internal/justice` import if not present. (No import cycle: hooks already imports justice.)

- [ ] **Step 4: Run tests + build**

Run: `go test ./internal/hooks/ -v` → PASS. `go build ./...`.

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/PlayerDeath_BountyResolve.go internal/hooks/PlayerDeath_BountyResolve_test.go
git commit -m "feat(bounty): hunter kill clears the player's faction record (5.2)"
```

---

## Task 4: `hunt_bounty_target` goal type

**Files:**
- Create: `internal/goals/catalog/hunt_bounty_target.go`
- Test: `internal/goals/catalog/hunt_bounty_target_test.go`

- [ ] **Step 1: Write the failing test**

```go
package catalog

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
)

func TestHuntBountyTarget_Registered(t *testing.T) {
	meta, ok := goals.LookupGoalType("hunt_bounty_target")
	if !ok {
		t.Fatalf("hunt_bounty_target not registered")
	}
	if meta.Predicate == nil {
		t.Fatalf("hunt_bounty_target needs a predicate")
	}
}
```
(If the registry accessor is named differently than `LookupGoalType`, grep `internal/goals/registry.go` and match it — e.g. `GoalTypeMeta(type)`.)

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/goals/catalog/ -run TestHuntBountyTarget -v` → FAIL.

- [ ] **Step 3: Implement the goal type**

```go
package catalog

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/bounties"
	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// hunt_bounty_target: a dispatched bounty hunter's reason to exist. Params carry
// the target player's userId and the triggering bounty id (stamped at dispatch).
// The goal is satisfied (drops out) once that bounty is no longer open.
func init() {
	goals.RegisterGoalType("hunt_bounty_target", goals.GoalTypeMeta{
		Predicate:    huntBountyPredicate,
		ContextScore: huntBountyContextScore,
		Params: []goals.ParamSchema{
			{Key: "target_user_id", Required: true, GoType: "int"},
			{Key: "bounty_id", Required: true, GoType: "int"},
		},
	})
}

// huntBountyPredicate: goal is ACTIVE (not satisfied) while the triggering
// bounty is still open. Returns true when the goal still applies.
func huntBountyPredicate(g *goals.Goal, mob goals.GoalMob) bool {
	bid := paramIntOr(g, "bounty_id", 0)
	if bid == 0 {
		return false
	}
	b := bounties.Get(bid)
	return b != nil && b.Status == bounties.StatusOpen
}

// huntBountyContextScore: dominant while the target is online; this is the
// hunter's singular purpose.
func huntBountyContextScore(g *goals.Goal, mob goals.GoalMob) float64 {
	uid := paramIntOr(g, "target_user_id", 0)
	if uid == 0 || users.GetByUserId(uid) == nil {
		return 0
	}
	_ = knowledge.PlayerSubject(uid) // (target subject available to consumers)
	_ = strconv.Itoa                  // keep imports honest if unused; remove if lint flags
	return 100
}
```

IMPORTANT, verify against the real signatures before finalizing:
- The `Predicate`/`ContextScore` function signatures (`PredicateFn`, `ContextScoreFn` in `internal/goals/types.go`) — match their exact parameter types (the mob param type may be `*mobs.Mob` or an interface like `goals.GoalMob`). Adjust the function signatures to match. If predicates take `(*Goal)` only, drop the mob param.
- `paramIntOr` — the catalog package already has this helper (used by `revenge_mob.go`). Reuse it; do not redefine.
- Remove the `_ = strconv.Itoa` / `knowledge` lines if they cause unused-import lint; they're only there as a reminder that the target subject is `knowledge.PlayerSubject(uid)`.

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/goals/catalog/ -run TestHuntBountyTarget -v` → PASS. `go build ./...`.

- [ ] **Step 5: Commit**

```bash
git add internal/goals/catalog/hunt_bounty_target.go internal/goals/catalog/hunt_bounty_target_test.go
git commit -m "feat(goals): hunt_bounty_target goal type (5.2)"
```

---

## Task 5: `hunt_bounty_target` planner

**Files:**
- Create: `internal/planners/hunt_bounty_target.go`
- Test: `internal/planners/hunt_bounty_target_test.go`

- [ ] **Step 1: Write the failing test**

```go
package planners

import "testing"

func TestHuntBountyTarget_Registered(t *testing.T) {
	if LookupPlanner("hunt_bounty_target") == nil {
		t.Fatalf("hunt_bounty_target planner not registered")
	}
}
```
Plus a behavioral test for the pure decision helper added in Step 3 (`huntDecision`): jailed target → hold (empty command, Running); same room → "attack @<uid>"; different room → "pathto <room>". Example:

```go
func TestHuntDecision(t *testing.T) {
	// jailed
	if cmd, _ := huntDecision(true, 10, 10, 200, 7); cmd != "" {
		t.Fatalf("jailed target must hold (empty command), got %q", cmd)
	}
	// same room
	if cmd, _ := huntDecision(false, 10, 10, 200, 7); cmd != "attack @7" {
		t.Fatalf("same room should attack, got %q", cmd)
	}
	// pursue
	if cmd, _ := huntDecision(false, 10, 25, 200, 7); cmd != "pathto 25" {
		t.Fatalf("pursue should pathto target room, got %q", cmd)
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/planners/ -run "TestHuntBountyTarget|TestHuntDecision" -v` → FAIL.

- [ ] **Step 3: Implement the planner + pure decision helper**

```go
package planners

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func init() {
	RegisterPlanner("hunt_bounty_target", huntBountyTargetPlanner)
}

// huntDecision is the pure pursuit decision (unit-testable):
//   - jailed target           → hold: empty command (loiter; never enter cell/engage)
//   - hunter in target's room  → "attack @<uid>"
//   - else                     → "pathto <targetRoom>" (closing chase)
func huntDecision(targetJailed bool, hunterRoom, targetRoom, _repathRounds, targetUserId int) (string, BTreeStatus) {
	if targetJailed {
		return "", StatusRunning
	}
	if hunterRoom == targetRoom {
		return "attack @" + strconv.Itoa(targetUserId), StatusRunning
	}
	return "pathto " + strconv.Itoa(targetRoom), StatusRunning
}

func huntBountyTargetPlanner(mob *mobs.Mob, goal *goals.Goal) PlanResult {
	if mob == nil {
		return PlanResult{Status: StatusFailure}
	}
	uid := goalParamIntOr(goal, "target_user_id", 0)
	if uid == 0 {
		return PlanResult{Status: StatusFailure}
	}
	u := users.GetByUserId(uid)
	if u == nil {
		// Target offline — hold; the dispatch manager suspends/ends the hunt.
		return PlanResult{Status: StatusRunning}
	}
	jailed := u.Character.HasBuffFlag(buffs.Jailed)
	cmd, status := huntDecision(
		jailed,
		mob.Character.RoomId,
		u.Character.RoomId,
		0,
		uid,
	)
	return PlanResult{Command: cmd, Status: status}
}
```

Verify before finalizing:
- `buffs.Jailed` is the correct flag constant (grep `internal/buffs/` for `Jailed`). If the jailed state is detected differently (e.g. a `no-aggro-target` flag or the `jail_until_round` MiscData), use `HasBuffFlag(buffs.Jailed)` if it exists, else check `mob`-side `GetMiscData("jail_until_round")` on the *target* user. The jailed buff is id 88; confirm a named `buffs.Jailed` flag exists, otherwise gate on the target's `jail_until_round` MiscData being present.
- `goalParamIntOr`, `LookupPlanner`, `StatusRunning`/`StatusFailure`, `PlanResult` all exist in the planners package (used by `wealth_gold.go`).
- Confirm `pathto <roomid>` works cross-zone for a mob (the caravan/patrol system does inter-zone movement). If `pathto` is in-zone only, note it as a follow-up and have the planner fall back to stepping toward the target's zone; for v1 in-zone pursuit is acceptable but log the limitation.

- [ ] **Step 4: Run tests + build**

Run: `go test ./internal/planners/ -v` → PASS. `go build ./...`.

- [ ] **Step 5: Commit**

```bash
git add internal/planners/hunt_bounty_target.go internal/planners/hunt_bounty_target_test.go
git commit -m "feat(planners): hunt_bounty_target pursuit planner (5.2)"
```

---

## Task 6: Hunter base gear items (content)

Author 8 dedicated "bounty hunter" base items (one per slot). Base specs are modest; affixes (generated at spawn) carry the scaled power. Run `python tools/id_inventory.py --type items` first to get the next free item ids; substitute the assigned ids below (placeholder range `309xx` shown — replace with real free ids).

**Files (create, one per item, under `_datafiles/world/dogmud/items/`):**
- weapon, body, head, legs, feet, gloves, neck, back — 8 files.

- [ ] **Step 1: Reserve ids**

Run `python tools/id_inventory.py --type items` and note 8 free ids. Use them consistently below.

- [ ] **Step 2: Author the 8 items**

For each, follow the existing item YAML schema (read `docs/schemas/item.md` + an existing weapon and armor item for exact fields). Example weapon (substitute real id):

```yaml
itemid: 30900
name: a bounty hunter's blade
itemtype: Weapon
subtype: Sword
description: >
  A businesslike blade with a worn grip and a notched edge that has
  clearly answered for more than one fugitive. Plain, balanced, lethal.
rarity_tier: 60
damage_multiplier: 1.0
hands: 1
value: 0
```

Example armor piece (body; substitute real id):

```yaml
itemid: 30901
name: a bounty hunter's coat
itemtype: Body
description: >
  A long coat of boiled leather reinforced with steel plates, scarred
  by old fights and cinched for hard travel.
rarity_tier: 60
physical_mitigation: 8
value: 0
```

Repeat for head/legs/feet/gloves/neck/back with slot-appropriate `itemtype` and modest base mitigation/stats. Keep names in the cohesive "a bounty hunter's X" family. Verify each `itemtype`/`subtype` against the schema so it loads.

- [ ] **Step 3: Boot smoke**

Wipe instances, boot. Expect `itemspec.LoadDataFiles() itemLoadedCount` increased by 8, no panic.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/items/
git commit -m "content(bounty): bounty hunter gear base items (5.2)"
```

---

## Task 7: `bounty_hunter` archetype + hunter mob template

**Files:**
- Create: `_datafiles/world/dogmud/behaviors/archetypes/bounty_hunter.yaml`
- Create: `_datafiles/world/dogmud/mobs/<zone>/<id>-bounty_hunter.yaml` (pick a sensible home zone/id via `id_inventory.py --type mobs`; the template is spawned dynamically so its room is nominal)

- [ ] **Step 1: Author the archetype**

Model on `scout.yaml` + `generic_fighter.yaml` (read both). Structure: a `mob_combat_round` selector with the standard combat cascade, a `mob_idle` branch that runs `try_goal_planner`, a panic-flee at a hard HP floor, and `default_goals` empty (the hunt goal is added dynamically at dispatch — do NOT seed a default hunt goal). Example skeleton (fill combat children to match generic_fighter):

```yaml
tree:
  type: selector
  children:
    # combat round: standard cascade (copy from generic_fighter.yaml)
    - type: sequence
      event: mob_combat_round
      children: [ ... copy generic_fighter combat children ... ]
    # idle: drive the strategic planner (pursuit)
    - type: sequence
      event: mob_idle
      children:
        - type: action
          name: try_goal_planner
goal_weights:
  hunt_bounty_target: 5.0
```

- [ ] **Step 2: Author the hunter mob template**

```yaml
mobid: <free id>
zone: <home zone>
archetype: fighting
behavior_archetype: bounty_hunter
statpool: 300            # nominal; overridden via forceStatPool at dispatch
hostile: false           # engages only its contract target via the planner
charm_immune: true
itemdropchance: 3        # ~3% independent per worn piece (5.2 gear)
groups:
  - humanoid
loot_pool:               # the 8 base items from Task 6 (real ids)
  - 30900
  - 30901
  # ... 6 more
character:
  name: a bounty hunter
  description: >
    A lean, hard-eyed figure in travel-worn leathers, a notched blade at
    the hip and a folded contract tucked into one glove. They move like
    someone who has done this many times and expects to again.
  speciesid: 1
  gold: 0
  stats:
    strength: { training: 10 }
    vitality: { training: 8 }
    perception: { training: 8 }
    willpower: { training: 6 }
```

(The issuer faction group is added dynamically at dispatch — Task 8 — so the `killGuard` claim path credits it. Do NOT hardcode a faction here.)

- [ ] **Step 3: Boot smoke**

Wipe instances, boot. Expect the `bounty_hunter` archetype loads (no unknown-archetype panic), mob count +1, clean boot.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/behaviors/archetypes/bounty_hunter.yaml _datafiles/world/dogmud/mobs/
git commit -m "content(bounty): bounty_hunter archetype + hunter mob template (5.2)"
```

---

## Task 8: Dispatch manager — spawn + scale + gear (`internal/bountyhunter`)

**Files:**
- Create: `internal/bountyhunter/bountyhunter.go` (registry + scaling + spawn)
- Create: `internal/bountyhunter/bountyhunter_test.go`

- [ ] **Step 1: Write the failing test (pure scaling)**

```go
package bountyhunter

import "testing"

func TestScaledStatpool(t *testing.T) {
	// clamp(250 + gold*0.25, 300, 500)
	cases := []struct{ gold, want int }{
		{300, 325}, {600, 400}, {850, 462}, {1000, 500}, {100, 300},
	}
	for _, c := range cases {
		got := scaledStatpool(c.gold, 250, 0.25, 300, 500)
		if got != c.want {
			t.Fatalf("scaledStatpool(%d) = %d, want %d", c.gold, got, c.want)
		}
	}
}

func TestGearGold(t *testing.T) {
	if g := gearGold(500, 5); g != 100 {
		t.Fatalf("gearGold(500,5) = %d, want 100", g)
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/bountyhunter/ -run "TestScaledStatpool|TestGearGold" -v` → FAIL.

- [ ] **Step 3: Implement scaling + spawn**

```go
package bountyhunter

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// hunterMobTemplateId is the bounty_hunter mob template id authored in Task 7.
const hunterMobTemplateId = 0 // TODO-AUTHOR: set to the real template id from Task 7

func scaledStatpool(gold, base int, perGold float64, min, max int) int {
	v := int(math.Round(float64(base) + float64(gold)*perGold))
	if v < min {
		v = min
	}
	if v > max {
		v = max
	}
	return v
}

func gearGold(statpool, divisor int) int {
	if divisor <= 0 {
		divisor = 1
	}
	return statpool / divisor
}

// spawnHunter creates a scaled, geared hunter at the issuer faction's seat,
// tags it with the issuer faction (so the kill-claim path credits it), adds the
// hunt goal, places it in the world, and telegraphs the target. Returns the
// hunter instance id (0 on failure).
func spawnHunter(targetUserId, bountyId, bountyGold int, issuerFaction string) int {
	bal := configs.GetBalanceConfig()
	def := factions.GetDefinition(issuerFaction)
	if def == nil || def.ReleaseRoom == 0 {
		return 0 // no seat to spawn from
	}
	seat := def.ReleaseRoom

	statpool := scaledStatpool(bountyGold,
		int(bal.BountyHunterBaseStatpool), float64(bal.BountyHunterStatpoolPerGold),
		int(bal.BountyHunterMinStatpool), int(bal.BountyHunterMaxStatpool))

	hunter := mobs.NewMobByIdFresh(mobs.MobId(hunterMobTemplateId), seat, statpool)
	if hunter == nil {
		return 0
	}

	// Tag with issuer faction so PlayerDeath_BountyResolve's killGuard path
	// credits the hunter + ClearFactionRecord fires.
	hunter.Groups = append(hunter.Groups, issuerFaction)

	// Affix-scaled gear (mirrors rooms.go:786 instance loot path).
	gg := gearGold(statpool, int(bal.BountyHunterGearGoldDivisor))
	scalar := float64(bal.LootBudgetScalar)
	for _, baseId := range hunter.LootPool {
		affixed := items.GenerateAffixedItem(baseId, gg, scalar)
		if affixed.ItemId > 0 {
			hunter.Character.Wear(affixed)
		}
	}

	// Add the hunt goal with target params (use the goals public Add API).
	goals.Add(int(hunter.MobId), hunter.Character.Name, goals.Goal{
		Type:     "hunt_bounty_target",
		Priority: 100,
		Params: map[string]any{
			"target_user_id": targetUserId,
			"bounty_id":      bountyId,
		},
	})

	// Place in the world + register.
	if room := rooms.LoadRoom(seat); room != nil {
		room.AddMob(hunter.InstanceId)
	}

	// Telegraph.
	if u := users.GetByUserId(targetUserId); u != nil {
		u.SendText(messaging.CategorySystem,
			"Word reaches you that a hunter has taken the contract on your head.")
	}

	return hunter.InstanceId
}

var _ = fmt.Sprintf // remove if unused
```

Verify before finalizing:
- `goals.Add` signature and `Goal` struct fields (`Type`, `Priority`, `Params`) — grep `internal/goals/`; the admin `/goal add` command (`internal/usercommands/admin.goal.go`) shows the real Add path. Match it (it may take `(mobId int, namesimple string, ...)` or build a Goal differently). Adjust the call.
- Confirm `hunter.Groups` is the right field to append a faction to and that `factions.FactionsForMob` reads it (it does, per the citizen-tagging work).
- Confirm `mobs.MobId` conversion + `NewMobByIdFresh(mobId, homeRoom, forceStatPool)` arg order.
- Set `hunterMobTemplateId` to the real id from Task 7.

- [ ] **Step 4: Run tests + build**

Run: `go test ./internal/bountyhunter/ -v` → PASS. `go build ./...`.

- [ ] **Step 5: Commit**

```bash
git add internal/bountyhunter/bountyhunter.go internal/bountyhunter/bountyhunter_test.go
git commit -m "feat(bountyhunter): scaled+geared hunter spawn (5.2)"
```

---

## Task 9: Dispatch sweep + lifecycle + round-tick wiring

**Files:**
- Create: `internal/bountyhunter/dispatch.go` (active-hunt registry + per-round sweep)
- Create: `internal/bountyhunter/dispatch_test.go`
- Modify: a `NewRound` hook to call the sweep once per round (see Step 4)

- [ ] **Step 1: Write the failing test (trigger decision)**

```go
package bountyhunter

import "testing"

func TestShouldDispatch(t *testing.T) {
	// over threshold, no active hunt, off cooldown → true
	if !shouldDispatch(600, 500, false, 0, 1000, 500) {
		t.Fatalf("expected dispatch when over threshold, idle, off cooldown")
	}
	// below threshold → false
	if shouldDispatch(400, 500, false, 0, 1000, 500) {
		t.Fatalf("below threshold must not dispatch")
	}
	// active hunt already → false
	if shouldDispatch(600, 500, true, 0, 1000, 500) {
		t.Fatalf("active hunt must not double-dispatch")
	}
	// within cooldown (lastKilled 800, now 1000, cd 500 → 200 < 500) → false
	if shouldDispatch(600, 500, false, 800, 1000, 500) {
		t.Fatalf("within redispatch cooldown must not dispatch")
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/bountyhunter/ -run TestShouldDispatch -v` → FAIL.

- [ ] **Step 3: Implement registry + sweep**

```go
package bountyhunter

import (
	"github.com/GoMudEngine/GoMud/internal/bounties"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

type hunt struct {
	hunterInstanceId int
	bountyId         int
	issuerFaction    string
}

var (
	activeHunts   = map[int]*hunt{} // keyed by target userId
	lastKilledFor = map[int]uint64{} // userId → round a hunter was last killed
)

// shouldDispatch is the pure trigger decision (unit-testable).
func shouldDispatch(maxBountyGold, threshold int, hasActive bool, lastKilledRound, nowRound uint64, cooldown uint64) bool {
	if maxBountyGold < threshold {
		return false
	}
	if hasActive {
		return false
	}
	if lastKilledRound > 0 && nowRound >= lastKilledRound && nowRound-lastKilledRound < cooldown {
		return false
	}
	return true
}

// RunDispatchSweep runs once per round: dispatches hunters for newly-eligible
// players and reaps finished/called-off hunts.
func RunDispatchSweep(nowRound uint64) {
	bal := configs.GetBalanceConfig()
	threshold := int(bal.BountyHunterGoldThreshold)
	cooldown := uint64(bal.BountyHunterRedispatchCooldown)

	// 1. Reap: end hunts whose bounty closed or whose hunter died.
	for uid, h := range activeHunts {
		b := bounties.Get(h.bountyId)
		bountyClosed := b == nil || b.Status != bounties.StatusOpen
		hunterDead := mobs.GetInstance(h.hunterInstanceId) == nil
		if bountyClosed {
			// Called off (player paid/served/it expired or the hunter claimed).
			despawnHunter(h.hunterInstanceId)
			delete(activeHunts, uid)
		} else if hunterDead {
			// Reprieve: bounty stands; start cooldown.
			lastKilledFor[uid] = nowRound
			delete(activeHunts, uid)
		}
	}

	// 2. Dispatch: scan open bounties, group max gold per target player.
	type cand struct{ gold, bountyId int; faction string }
	best := map[int]cand{} // userId → strongest open faction bounty
	for _, b := range bounties.AllOpen() {
		if b.Target.Type != knowledgeSubjectPlayer() || b.Issuer.Type != bounties.IssuerFaction {
			continue
		}
		uid := b.Target.Id
		if c, ok := best[uid]; !ok || b.GoldReward > c.gold {
			best[uid] = cand{b.GoldReward, b.Id, b.Issuer.Id}
		}
	}
	for uid, c := range best {
		_, hasActive := activeHunts[uid]
		if !shouldDispatch(c.gold, threshold, hasActive, lastKilledFor[uid], nowRound, cooldown) {
			continue
		}
		id := spawnHunter(uid, c.bountyId, c.gold, c.faction)
		if id > 0 {
			activeHunts[uid] = &hunt{hunterInstanceId: id, bountyId: c.bountyId, issuerFaction: c.faction}
		}
	}
}

func despawnHunter(instanceId int) {
	// Remove the hunter from its room + the instance registry. Use the same
	// despawn path mobs use elsewhere (grep for how summoned/temporary mobs are
	// removed — e.g. rooms.RemoveMob + mobs.DestroyInstance). Implement to the
	// real API.
}
```

Verify before finalizing:
- Replace `knowledgeSubjectPlayer()` with the real constant `knowledge.SubjectPlayer` (import `internal/knowledge`); it's written as a helper here only to avoid an import placeholder — use the real enum.
- Implement `despawnHunter` against the real removal API: grep for how transient mobs are removed (e.g. `rooms.RemoveMob(instanceId)` + a `mobs` destroy/dealloc). Match the existing despawn pattern (look at companion dismiss or summon expiry).
- Confirm `bounties.Get`, `bounties.AllOpen`, `mobs.GetInstance` signatures (from the substrate map).

- [ ] **Step 4: Wire the sweep into the round tick**

In `internal/hooks/NewRound_MobRoundTick.go`, **before** the per-mob loop (so it runs once per round, not per mob), add a single call:

```go
	bountyhunter.RunDispatchSweep(roundCount)
```
Add the `internal/bountyhunter` import. (Confirm `roundCount`/the round number is in scope at the top of the handler; it is used later in the per-mob loop.) If a once-per-round placement is cleaner in a different `NewRound_*.go` handler, use that — the requirement is exactly-once-per-round.

- [ ] **Step 5: Run tests + build + boot smoke**

Run: `go test ./internal/bountyhunter/ -v` → PASS. `go build ./...`. Boot smoke → clean (no hunter dispatched at boot since no over-threshold bounties exist yet).

- [ ] **Step 6: Commit**

```bash
git add internal/bountyhunter/dispatch.go internal/bountyhunter/dispatch_test.go internal/hooks/NewRound_MobRoundTick.go
git commit -m "feat(bountyhunter): per-round dispatch sweep + hunt lifecycle (5.2)"
```

---

## Task 10: Half B — standing-bounty seed (player-claimable NPC bounties)

**Files:**
- Create: `_datafiles/world/dogmud/bounties.standing.yaml` (committed seed list)
- Create: `internal/bounties/standing.go` (loader, idempotent)
- Create: `internal/bounties/standing_test.go`
- Modify: `main.go` (call the loader at boot, after bounties + mobs load)

- [ ] **Step 1: Write the failing test (idempotent declare)**

```go
func TestSeedStanding_Idempotent(t *testing.T) {
	// With a fake declare/openForTarget seam: first SeedStanding declares N;
	// second SeedStanding (same input, bounties now open) declares 0.
	// Assert declare-call count == N after first, still N after second.
}
```
(Author against the bounties package internals — use the existing `statpoolForTest` seam pattern + a declare counter; mirror how `bounties` unit tests stub declaration.)

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/bounties/ -run TestSeedStanding -v` → FAIL.

- [ ] **Step 3: Author the seed file**

```yaml
# Standing kill-bounties on notable hostile NPCs, declared at boot (idempotent).
# Players claim by killing the target (MobDeath_BountyClaim credits the highest
# damager). Issuer = a faction; reward auto-computed from target statpool unless
# gold/rep overridden.
- target_mob_id: <chrysalis phantom mob id>
  issuer_faction: thornwall_guards
  reason: "Standing contract: the Chrysalis Phantom"
- target_mob_id: 286            # Soren, north_road bandit boss
  issuer_faction: thornwall_guards
  reason: "Standing contract: the bandit Soren"
# add a few more notable bandits as desired
```
(Look up the Chrysalis Phantom mob id and confirm Soren=286 via `id_inventory`/grep.)

- [ ] **Step 4: Implement the loader**

```go
package bounties

import (
	"os"
	"path/filepath"

	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

type standingEntry struct {
	TargetMobId   int    `yaml:"target_mob_id"`
	IssuerFaction string `yaml:"issuer_faction"`
	Reason        string `yaml:"reason"`
}

// SeedStanding declares standing kill-bounties from the committed seed file,
// idempotently — it skips any target that already has an open bounty.
func SeedStanding(dataFilesPath string) {
	path := filepath.Join(dataFilesPath, "bounties.standing.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			mudlog.Warn("bounties.SeedStanding: read", "path", path, "error", err)
		}
		return
	}
	var entries []standingEntry
	if err := yaml.Unmarshal(raw, &entries); err != nil {
		mudlog.Warn("bounties.SeedStanding: unmarshal", "path", path, "error", err)
		return
	}
	declared := 0
	for _, e := range entries {
		if e.TargetMobId == 0 || e.IssuerFaction == "" {
			continue
		}
		target := knowledge.MobSubject(e.TargetMobId)
		if len(OpenForTarget(target)) > 0 {
			continue // already open — idempotent
		}
		_, derr := Declare(FactionIssuer(e.IssuerFaction), target, ConditionKill, 0,
			DeclareOpts{DeclaredReason: e.Reason})
		if derr != nil {
			mudlog.Warn("bounties.SeedStanding: declare", "mob", e.TargetMobId, "error", derr)
			continue
		}
		declared++
	}
	mudlog.Info("bounties.SeedStanding", "declared", declared, "entries", len(entries))
}
```

Verify: the bounties package's data-files base path (how other bounties persistence resolves its dir — match it for `dataFilesPath`); `Declare`/`OpenForTarget`/`FactionIssuer`/`MobSubject`/`ConditionKill`/`DeclareOpts` signatures (from the substrate map).

- [ ] **Step 5: Wire into boot**

In `main.go`, after the bounties registry + mobs are loaded (and after factions load, since the issuer must resolve), add `bounties.SeedStanding(<dataFilesPath>)`. Use the same data-files path expression main.go passes to other loaders.

- [ ] **Step 6: Run tests + build + boot smoke**

Run: `go test ./internal/bounties/ -v` → PASS. `go build ./...`. Boot smoke twice (wipe + boot, then boot again without wiping the bounties registry) → second boot logs `declared=0` (idempotent); no duplicate bounties.

- [ ] **Step 7: Commit**

```bash
git add _datafiles/world/dogmud/bounties.standing.yaml internal/bounties/standing.go internal/bounties/standing_test.go main.go
git commit -m "feat(bounties): standing player-claimable NPC bounties seed (5.2 Half B)"
```

---

## Task 11: Docs, roadmap, full regression + boot smoke

**Files:**
- Create: `internal/bountyhunter/context.md`
- Modify: `internal/justice/context.md` (note `ClearFactionRecord` is now shared with bounty kills)
- Modify: `MOB_ALIVENESS_ROADMAP.md` (mark 5.2 status + summary)

- [ ] **Step 1: Write `internal/bountyhunter/context.md`**

Document: the package's role (dispatch sweep + hunt lifecycle), the trigger (single bounty ≥ threshold, one-per-player, re-dispatch cooldown), spawn (seat = issuer `release_room`, `forceStatPool` scaling, affix gear via the oasis path), pursuit (the `hunt_bounty_target` goal/planner — pathfind/engage, jailed-hold), claim (issuer-faction tag → `PlayerDeath_BountyResolve` killGuard path → `ClearFactionRecord`), and the config knobs. Note the jailed-target safety (planner hold + the Jailed buff's no-aggro-target net).

- [ ] **Step 2: Update `internal/justice/context.md`**

Add a line: `ClearFactionRecord(faction, userId)` is the shared record-clearing used by both serving a sentence (`ResolveDetention`) and a bounty-hunter kill (5.2).

- [ ] **Step 3: Update the roadmap**

In `MOB_ALIVENESS_ROADMAP.md`, set chunk 5.2 status to shipped (2026-05-30) with a one-paragraph summary (Half A NPC-hunts-wanted-player + Half B player-claimable NPC bounties; not PvP). Note the deferred followups: disguise-kit evasion; criminal-NPC-commits-crime-then-hunted (NPC-vs-NPC).

- [ ] **Step 4: Full regression + boot smoke**

```bash
go build ./...
go test ./... 2>&1 | grep -iE "FAIL|panic" ; echo "exit grep (empty = all pass)"
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go build -o /tmp/dogmud_52.exe . && timeout 40 /tmp/dogmud_52.exe 2>&1 | grep -iE "loadedCount|panic|fatal|Server Ready" | head -30
```
Expect: all tests pass; clean boot; `bounties.SeedStanding declared=...` logged; `bounty_hunter` archetype + hunter template + 8 items load.

- [ ] **Step 5: Commit**

```bash
git add internal/bountyhunter/context.md internal/justice/context.md MOB_ALIVENESS_ROADMAP.md
git commit -m "docs(bountyhunter): context + roadmap for 5.2 bounty hunting"
```

---

## In-game smoke test plan (deferred to user)

1. **Hunter dispatch + pursuit + kill:** commit a murder (single bounty ≥ 500) → telegraph fires → a scaled, geared hunter spawns at the issuer's station and closes in across rooms → it catches and kills you → your record with that faction clears (no immediate re-dispatch).
2. **Kill the hunter:** trigger a hunter, defeat it → reprieve; confirm no new hunter until `BountyHunterRedispatchCooldown` passes and only if still ≥ threshold.
3. **Jail escape:** trigger a hunter → flee to town → get arrested → confirm the hunter cannot reach/kill you in the cell (planner hold + Jailed no-aggro-target), and serving/paying calls off the hunt (bounty withdrawn).
4. **Gear scaling + drops:** inspect a high-bounty vs low-bounty hunter (tougher bounty → better gear); kill several hunters and confirm rare (~3%/piece) gear drops.
5. **Half B:** read a standing bounty (e.g. the Chrysalis Phantom) on a board / `bounty list`, kill the target, receive the gold + faction rep.
```
