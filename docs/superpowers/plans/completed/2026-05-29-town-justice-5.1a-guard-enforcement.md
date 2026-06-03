# Town Justice 5.1a — Guard Enforcement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Guards detect "wanted" players (bad faction rep, open faction bounty, or fresh unresolved crime against their faction/allies) and escalate warn → attack, via a shared `internal/justice` substrate driven by a per-round enforcement tick.

**Architecture:** New `internal/justice` package holds an ordered `Severity` enum, a pure `Verdict` decision (reusable by 5.2 bounty hunters), and a `RunGuardEnforcement` action with warn-grace memory in mob MiscData. A per-round hook in `NewRound_MobRoundTick.go` runs it for `guard`-group mobs. No changes to the crimes/factions/bounties substrate (already built) and none to the crime call sites (intervention is reactive-via-tick).

**Tech Stack:** Go; reads `factions`/`crimes`/`bounties`/`opinions`; plain `t.Error` tests with package function-seams (the established Phase-1 pattern).

**Spec:** `docs/superpowers/specs/2026-05-29-town-justice-5.1a-guard-enforcement-design.md`

**Verified facts (use, don't re-derive):**
- `factions.TierFor(faction string, userId int) opinions.Tier`; `factions.FactionsForMob(mob *mobs.Mob) []string`; `factions.GetDefinition(f) *Definition` with `.Allies []string`.
- `opinions.Tier` is an ordered iota: `TierHostile=0`, `TierCold=1`, `TierNeutral=2`, `TierWarm=3`, `TierFriendly=4` (more hostile = lower).
- `crimes.AllForFaction(factionId string, includeResolved bool) []*crimes.Crime`; `Crime.Perpetrator` is `crimes.Perpetrator{Type crimes.PerpType, Id int}` with `crimes.PerpPlayer`; `Crime.Round uint64`.
- `bounties.OpenAgainstPlayer(userId int) []*bounties.Bounty`; `Bounty.Issuer` is `bounties.Issuer{Type bounties.IssuerType, Id string}` with `bounties.IssuerFaction`.
- `characters.Character.SetMiscData(key string, value any)` / `GetMiscData(key string) any`.
- `mob.Command(string)` queues an input; `room.GetPlayers(rooms.FindAll) []int`; `users.GetByUserId(id) *users.UserRecord`; `buffs.NoAggroTarget`; `user.Character.IsHidden()`.
- `actions.FormatSayText(name, text string, isSneaking bool, nameColor, textColor string) string`; `room.SendText(cat messaging.Category, txt string)` — for synchronous mob speech (the merchantSay pattern, so a scheduled guard's warning is never queue-delayed).
- Per-round tick loop: `internal/hooks/NewRound_MobRoundTick.go` ~line 90 iterates `mobs.GetAllMobInstanceIds()`, with `mob`, `room` (`rooms.LoadRoom(mob.Character.RoomId)`, may be nil), and `roundCount` in scope; `tickMobRecomputeGoals(mob, roundCount)` is called at line 109.

---

## File Structure

| File | Responsibility |
|------|----------------|
| `internal/configs/config.balance.go` | `GuardWarnGraceRounds`, `JusticeCrimeLookbackRounds` fields |
| `internal/configs/config.balance.mobs.go` | defaults (50, 1000) |
| `internal/justice/justice.go` (new) | `Severity` enum, `Verdict` decision, read-seams |
| `internal/justice/enforce.go` (new) | `RunGuardEnforcement`, `EnforceAction`, pure `resolveWarn` helper, `guardSay` |
| `internal/justice/justice_test.go` (new) | `Verdict` table tests via seams |
| `internal/justice/enforce_test.go` (new) | `resolveWarn` unit tests + `RunGuardEnforcement` integration |
| `internal/hooks/NewRound_MobRoundTick.go` | per-round guard enforcement call + `isGuardMob` helper |
| `internal/hooks/NewRound_MobRoundTick_GuardEnforce_test.go` (new) | `isGuardMob` gate test |

---

### Task 1: Config knobs + defaults

**Files:** Modify `internal/configs/config.balance.go`, `internal/configs/config.balance.mobs.go`

- [ ] **Step 1: Add fields** — in `config.balance.go`, after the goal-pruning knobs (`GoalAbandonDormantRounds`):

```go
	// GuardWarnGraceRounds is how many rounds a warned (Cold-rep) player may
	// remain present before a guard escalates the warning to an attack. 5.1a.
	GuardWarnGraceRounds ConfigInt `yaml:"GuardWarnGraceRounds"`

	// JusticeCrimeLookbackRounds is the recency window for the unresolved-crime
	// "wanted" signal used by guard enforcement. 5.1a.
	JusticeCrimeLookbackRounds ConfigInt `yaml:"JusticeCrimeLookbackRounds"`
```

- [ ] **Step 2: Defaults** — in `config.balance.mobs.go`, after the `GoalAbandonDormantRounds` guard:

```go
	if b.GuardWarnGraceRounds < 1 {
		b.GuardWarnGraceRounds = 50
	}
	if b.JusticeCrimeLookbackRounds < 1 {
		b.JusticeCrimeLookbackRounds = 1000
	}
```

- [ ] **Step 3: Build** — Run `go build ./internal/configs/`. Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add internal/configs/config.balance.go internal/configs/config.balance.mobs.go
git commit -m "feat(config): GuardWarnGraceRounds + JusticeCrimeLookbackRounds

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: `internal/justice` — Severity + Verdict

**Files:** Create `internal/justice/justice.go`, `internal/justice/justice_test.go`

- [ ] **Step 1: Write the failing tests** — `internal/justice/justice_test.go`:

```go
package justice

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/opinions"
)

// installSeams sets the read-seams to controllable fakes and returns a restore.
func installSeams(t *testing.T) func() {
	t.Helper()
	origRep, origAllies, origBounty, origCrime, origNow, origLook :=
		repTierFn, alliesFn, openFactionBountyFn, unresolvedCrimeFn, nowRoundFn, lookbackFn
	return func() {
		repTierFn, alliesFn, openFactionBountyFn, unresolvedCrimeFn, nowRoundFn, lookbackFn =
			origRep, origAllies, origBounty, origCrime, origNow, origLook
	}
}

func TestVerdict_NeutralRep_None(t *testing.T) {
	defer installSeams(t)()
	repTierFn = func(string, int) opinions.Tier { return opinions.TierNeutral }
	alliesFn = func(string) []string { return nil }
	openFactionBountyFn = func(int, map[string]bool) bool { return false }
	unresolvedCrimeFn = func(string, int, uint64, uint64) bool { return false }
	nowRoundFn = func() uint64 { return 1000 }
	lookbackFn = func() uint64 { return 1000 }
	if got := Verdict([]string{"thornwall_guards"}, 17); got != SeverityNone {
		t.Errorf("got %v, want None", got)
	}
}

func TestVerdict_ColdRep_Warn(t *testing.T) {
	defer installSeams(t)()
	repTierFn = func(string, int) opinions.Tier { return opinions.TierCold }
	alliesFn = func(string) []string { return nil }
	openFactionBountyFn = func(int, map[string]bool) bool { return false }
	unresolvedCrimeFn = func(string, int, uint64, uint64) bool { return false }
	nowRoundFn = func() uint64 { return 1000 }
	lookbackFn = func() uint64 { return 1000 }
	if got := Verdict([]string{"thornwall_guards"}, 17); got != SeverityWarn {
		t.Errorf("got %v, want Warn", got)
	}
}

func TestVerdict_HostileRep_Attack(t *testing.T) {
	defer installSeams(t)()
	repTierFn = func(string, int) opinions.Tier { return opinions.TierHostile }
	alliesFn = func(string) []string { return nil }
	openFactionBountyFn = func(int, map[string]bool) bool { return false }
	unresolvedCrimeFn = func(string, int, uint64, uint64) bool { return false }
	nowRoundFn = func() uint64 { return 1000 }
	lookbackFn = func() uint64 { return 1000 }
	if got := Verdict([]string{"thornwall_guards"}, 17); got != SeverityAttack {
		t.Errorf("got %v, want Attack", got)
	}
}

func TestVerdict_OpenBounty_Attack(t *testing.T) {
	defer installSeams(t)()
	repTierFn = func(string, int) opinions.Tier { return opinions.TierNeutral }
	alliesFn = func(string) []string { return nil }
	openFactionBountyFn = func(int, map[string]bool) bool { return true }
	unresolvedCrimeFn = func(string, int, uint64, uint64) bool { return false }
	nowRoundFn = func() uint64 { return 1000 }
	lookbackFn = func() uint64 { return 1000 }
	if got := Verdict([]string{"thornwall_guards"}, 17); got != SeverityAttack {
		t.Errorf("got %v, want Attack (bounty)", got)
	}
}

func TestVerdict_AllyCrime_Attack(t *testing.T) {
	defer installSeams(t)()
	// Guard faction neutral, but an ALLY (citizens) has an unresolved crime.
	repTierFn = func(string, int) opinions.Tier { return opinions.TierNeutral }
	alliesFn = func(f string) []string {
		if f == "thornwall_guards" {
			return []string{"thornwall_citizens"}
		}
		return nil
	}
	openFactionBountyFn = func(int, map[string]bool) bool { return false }
	unresolvedCrimeFn = func(faction string, _ int, _ uint64, _ uint64) bool {
		return faction == "thornwall_citizens"
	}
	nowRoundFn = func() uint64 { return 1000 }
	lookbackFn = func() uint64 { return 1000 }
	if got := Verdict([]string{"thornwall_guards"}, 17); got != SeverityAttack {
		t.Errorf("got %v, want Attack (ally crime)", got)
	}
}
```

- [ ] **Step 2: Run, expect FAIL** — `go test ./internal/justice/` → undefined `Verdict`/`Severity`/seams.

- [ ] **Step 3: Implement `internal/justice/justice.go`**

```go
// Package justice computes whether a player is "wanted" by a faction and how
// severely, from the existing crime/rep/bounty substrate. The decision
// (Verdict) is pure and reusable (5.2 bounty hunters); the enforcement action
// lives in enforce.go.
package justice

import (
	"github.com/GoMudEngine/GoMud/internal/bounties"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crimes"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/opinions"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Severity is ordered: a future Arrest rung (5.1c) slots between Warn and
// Attack without reshaping callers. Compare with >.
type Severity int

const (
	SeverityNone   Severity = iota // citizen / not wanted
	SeverityWarn                   // wanted, mild — verbal warning
	SeverityArrest                 // RESERVED for 5.1c; not produced here
	SeverityAttack                 // wanted, severe — engage
)

// Read-seams — production wiring below; tests override.
var (
	repTierFn = func(faction string, userId int) opinions.Tier {
		return factions.TierFor(faction, userId)
	}
	alliesFn = func(faction string) []string {
		d := factions.GetDefinition(faction)
		if d == nil {
			return nil
		}
		return d.Allies
	}
	openFactionBountyFn = func(userId int, factionSet map[string]bool) bool {
		for _, b := range bounties.OpenAgainstPlayer(userId) {
			if b.Issuer.Type == bounties.IssuerFaction && factionSet[b.Issuer.Id] {
				return true
			}
		}
		return false
	}
	unresolvedCrimeFn = func(faction string, userId int, lookback, now uint64) bool {
		for _, c := range crimes.AllForFaction(faction, false) {
			if c.Perpetrator.Type != crimes.PerpPlayer || c.Perpetrator.Id != userId {
				continue
			}
			if now >= c.Round && now-c.Round <= lookback {
				return true
			}
		}
		return false
	}
	nowRoundFn = util.GetRoundCount
	lookbackFn = func() uint64 {
		v := configs.GetBalanceConfig().JusticeCrimeLookbackRounds
		if v < 1 {
			return 1000
		}
		return uint64(v)
	}
)

// Verdict returns how a guard belonging to guardFactions should treat a
// player, taking the most severe of bounty / rep-tier / unresolved-crime
// signals across the guard factions and their declared allies.
func Verdict(guardFactions []string, userId int) Severity {
	set := map[string]bool{}
	for _, f := range guardFactions {
		set[f] = true
		for _, a := range alliesFn(f) {
			set[a] = true
		}
	}
	if openFactionBountyFn(userId, set) {
		return SeverityAttack
	}
	now := nowRoundFn()
	lookback := lookbackFn()
	worst := SeverityNone
	for f := range set {
		switch repTierFn(f, userId) {
		case opinions.TierHostile:
			return SeverityAttack
		case opinions.TierCold:
			if worst < SeverityWarn {
				worst = SeverityWarn
			}
		}
		if unresolvedCrimeFn(f, userId, lookback, now) {
			return SeverityAttack
		}
	}
	return worst
}
```

- [ ] **Step 4: Run, expect PASS** — `go test ./internal/justice/`.

- [ ] **Step 5: Commit**

```bash
git add internal/justice/justice.go internal/justice/justice_test.go
git commit -m "feat(justice): Severity + Verdict wanted-decision

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: `internal/justice` — RunGuardEnforcement

**Files:** Create `internal/justice/enforce.go`, `internal/justice/enforce_test.go`

- [ ] **Step 1: Write the failing tests** — `internal/justice/enforce_test.go`:

```go
package justice

import "testing"

func TestResolveWarn_FirstSighting_WarnsAndStamps(t *testing.T) {
	out := resolveWarn(false, 0, 1000, 50)
	if out != warnOutcomeWarn {
		t.Errorf("got %v, want warn", out)
	}
}

func TestResolveWarn_WithinGrace_NoOp(t *testing.T) {
	out := resolveWarn(true, 1000, 1020, 50) // 20 < 50
	if out != warnOutcomeNone {
		t.Errorf("got %v, want none (within grace)", out)
	}
}

func TestResolveWarn_PastGrace_Escalates(t *testing.T) {
	out := resolveWarn(true, 1000, 1060, 50) // 60 >= 50
	if out != warnOutcomeAttack {
		t.Errorf("got %v, want attack (past grace)", out)
	}
}

func TestMiscDataRound_ParsesNumericKinds(t *testing.T) {
	if r, ok := miscDataRound(map[string]any{"k": uint64(42)}, "k"); !ok || r != 42 {
		t.Errorf("uint64: got %d,%v", r, ok)
	}
	if r, ok := miscDataRound(map[string]any{"k": 42}, "k"); !ok || r != 42 {
		t.Errorf("int: got %d,%v", r, ok)
	}
	if _, ok := miscDataRound(map[string]any{}, "k"); ok {
		t.Error("missing key should return ok=false")
	}
}
```

- [ ] **Step 2: Run, expect FAIL** — `go test ./internal/justice/ -run 'ResolveWarn|MiscDataRound'`.

- [ ] **Step 3: Implement `internal/justice/enforce.go`**

```go
package justice

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// EnforceAction reports one enforcement decision (returned for tests/telemetry).
type EnforceAction struct {
	UserId    int
	Severity  Severity
	Escalated bool // a prior warning escalated to attack this tick
}

type warnOutcome int

const (
	warnOutcomeNone warnOutcome = iota
	warnOutcomeWarn
	warnOutcomeAttack
)

// resolveWarn is the pure escalation decision for a Warn-severity player.
func resolveWarn(alreadyWarned bool, warnedRound, nowRound, grace uint64) warnOutcome {
	if !alreadyWarned {
		return warnOutcomeWarn
	}
	if nowRound >= warnedRound && nowRound-warnedRound >= grace {
		return warnOutcomeAttack
	}
	return warnOutcomeNone
}

// miscDataRound reads a round value stored in MiscData under key, tolerating
// the numeric kinds a YAML round-trip can produce.
func miscDataRound(misc map[string]any, key string) (uint64, bool) {
	v, ok := misc[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case uint64:
		return n, true
	case int64:
		return uint64(n), true
	case int:
		return uint64(n), true
	case float64:
		return uint64(n), true
	}
	return 0, false
}

func warnGraceRounds() uint64 {
	v := configs.GetBalanceConfig().GuardWarnGraceRounds
	if v < 1 {
		return 50
	}
	return uint64(v)
}

// RunGuardEnforcement scans players in the room and applies warn/attack
// against wanted players for this guard, managing warn-grace memory in the
// guard's MiscData. Returns the actions taken (for tests). Both the per-round
// tick (now) and a future protection-faction btree action (later) call this.
func RunGuardEnforcement(mob *mobs.Mob, room *rooms.Room, nowRound uint64) []EnforceAction {
	if mob == nil || room == nil || mob.Character.IsInCombat() {
		return nil
	}
	guardFactions := factions.FactionsForMob(mob)
	if len(guardFactions) == 0 {
		return nil
	}
	grace := warnGraceRounds()
	var actions []EnforceAction

	for _, uid := range room.GetPlayers(rooms.FindAll) {
		user := users.GetByUserId(uid)
		if user == nil {
			continue
		}
		if user.Character.HasBuffFlag(buffs.NoAggroTarget) ||
			user.Character.IsHidden() || user.Character.Health < 1 {
			continue
		}

		sev := Verdict(guardFactions, uid)
		switch sev {
		case SeverityAttack:
			mob.Command(fmt.Sprintf("attack @%d", uid))
			actions = append(actions, EnforceAction{uid, SeverityAttack, false})
		case SeverityWarn:
			key := fmt.Sprintf("justice_warned_%d", uid)
			warnedRound, warned := miscDataRound(mob.Character.MiscData, key)
			switch resolveWarn(warned, warnedRound, nowRound, grace) {
			case warnOutcomeWarn:
				guardSay(room, mob, "Move along — you're not welcome here.")
				mob.Character.SetMiscData(key, nowRound)
				actions = append(actions, EnforceAction{uid, SeverityWarn, false})
			case warnOutcomeAttack:
				mob.Command(fmt.Sprintf("attack @%d", uid))
				actions = append(actions, EnforceAction{uid, SeverityAttack, true})
			}
		}
	}
	return actions
}

// guardSay broadcasts a guard's line synchronously (the merchantSay pattern) so
// a scheduled guard's warning is never delayed by the mob command queue.
func guardSay(room *rooms.Room, mob *mobs.Mob, line string) {
	actor := &actions.MobActor{Mob: mob, Room: room}
	result := actions.Say(actor, line)
	room.SendText(messaging.CategorySpeech,
		actions.FormatSayText(mob.Character.Name, result.Text, false, "mobname", "saytext-mob"))
}
```

Note: `mob.Character.MiscData` is the underlying `map[string]any`; if it can be nil on a fresh mob, `miscDataRound` returns `(0,false)` for a nil map (safe), and `SetMiscData` allocates it.

- [ ] **Step 4: Run, expect PASS** — `go test ./internal/justice/`.

- [ ] **Step 5: Commit**

```bash
git add internal/justice/enforce.go internal/justice/enforce_test.go
git commit -m "feat(justice): RunGuardEnforcement warn/attack with grace memory

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: Per-round enforcement tick

**Files:** Modify `internal/hooks/NewRound_MobRoundTick.go`; create `internal/hooks/NewRound_MobRoundTick_GuardEnforce_test.go`

- [ ] **Step 1: Write the failing test** — `internal/hooks/NewRound_MobRoundTick_GuardEnforce_test.go`:

```go
package hooks

import "testing"

func TestIsGuardMob(t *testing.T) {
	if !isGuardMob([]string{"humanoid", "guard", "thornwall_guards"}) {
		t.Error("expected guard group to be detected")
	}
	if isGuardMob([]string{"humanoid", "merchant"}) {
		t.Error("non-guard must not be detected")
	}
	if isGuardMob(nil) {
		t.Error("nil groups must not be detected")
	}
}
```

- [ ] **Step 2: Run, expect FAIL** — `go test ./internal/hooks/ -run TestIsGuardMob` → undefined `isGuardMob`.

- [ ] **Step 3: Add the helper + wire the tick** — in `NewRound_MobRoundTick.go`, add the import `"github.com/GoMudEngine/GoMud/internal/justice"`, add the helper near `tickMobRecomputeGoals`:

```go
// isGuardMob reports whether a mob's groups include the law-enforcement
// "guard" marker (5.1a town justice).
func isGuardMob(groups []string) bool {
	for _, g := range groups {
		if g == "guard" {
			return true
		}
	}
	return false
}
```

Then, in the per-mob loop, immediately after the `tickMobRecomputeGoals(mob, roundCount)` line, add:

```go
		if room != nil && isGuardMob(mob.Groups) {
			justice.RunGuardEnforcement(mob, room, roundCount)
		}
```

(`mob.Groups` is the mob's group slice; `room` and `roundCount` are already in scope at this point in the loop.)

- [ ] **Step 4: Run, expect PASS** — `go test ./internal/hooks/ -run 'TestIsGuardMob|RecomputeGoals'`. (Scope with `-run`; the package has a known pre-existing flaky `TestHandlePlayerFoldCasting_*` — re-run if hit.)

- [ ] **Step 5: Build** — `go build ./...`. Expected: no output.

- [ ] **Step 6: Commit**

```bash
git add internal/hooks/NewRound_MobRoundTick.go internal/hooks/NewRound_MobRoundTick_GuardEnforce_test.go
git commit -m "feat(justice): per-round guard enforcement tick (guard group)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: Verification

**Files:** none (verification only)

- [ ] **Step 1: Confirm guard mobs carry the `guard` group** — Run:
  `grep -rl "  - guard$" _datafiles/world/dogmud/mobs/` — confirm city_guard (106), city_gate_guard (92), guard_captain_velk (94), constable_drunn (335) appear. If any lacks the `guard` group, add it (single list line) and note it in the commit.

- [ ] **Step 2: Build + vet** — `go build ./...` and `go vet ./internal/justice/ ./internal/hooks/`. Expected: clean.

- [ ] **Step 3: Tests** — `go test ./internal/justice/... ./internal/hooks/ -run 'Justice|Verdict|ResolveWarn|MiscData|IsGuardMob|RecomputeGoals'` and `go test ./internal/configs/`. Expected: ok.

- [ ] **Step 4: Boot smoke (instance wipe per SOP)**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . > /tmp/51a_boot.log 2>&1 &
```
Poll for `Server Ready` or a panic; confirm `Config name="Balance.GuardWarnGraceRounds" value=50` and `...JusticeCrimeLookbackRounds value=1000`, no `panic`/`fatal`. Then `taskkill //IM "GoMud.exe" //F`.

---

## Notes for the implementer

- **No substrate or crime-site changes.** `factions`/`crimes`/`bounties` are already built; crime-in-progress intervention is reactive-via-tick (a fresh crime makes the player wanted on the guard's next round). Do not touch combat/steal/attack code.
- **Citizenship needs no work** — `peacefulquest` was retired in chunk 1.2; a Neutral+ player is `SeverityNone` (left alone).
- **Warn must be synchronous** (`guardSay`, not `mob.Command("say")`): guards are scheduled mobs, and a queued say can be delayed (the same bug fixed for `sell`/`offer` on 2026-05-29).
- **Import direction:** `justice` imports factions/crimes/bounties/opinions/mobs/rooms/users/actions/buffs/configs/messaging/util; none import `justice`, and `hooks` importing `justice` adds no cycle.
- **Forward-compat (do not build now):** `SeverityArrest` is reserved for 5.1c; `RunGuardEnforcement` is the reusable seam a Phase-4 `protection-faction` btree action can call later; the bounty filter is guard-scoped so 5.2 hunters can add an all-issuers query.
- **Test coverage strategy:** the decision logic is fully unit-tested at the pure level — `Verdict` (Task 2, via read-seams), `resolveWarn` + `miscDataRound` (Task 3). `RunGuardEnforcement`'s loop is thin glue over those plus `room.GetPlayers`/`users.GetByUserId`; it returns `[]EnforceAction` so it CAN be integration-tested with `rooms.SeedRoomsForTest` + a seeded user if the users-package test seam is readily available. If seeding a user cross-package proves high-cost, rely on the pure-unit tests + the boot smoke + a manual in-game check (warn on a Cold-rep player, attack on Hostile/bounty) rather than forcing a brittle integration harness — note which path you took in the Task 3 commit.
