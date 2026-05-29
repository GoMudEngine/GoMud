# Mob Aliveness 4.6 — Goal Satisfaction & Pruning Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire the unused `IsSatisfied`/`IsExpired` primitives into a throttled per-mob prune sweep, plus dormancy-based abandonment, so mob goals don't accumulate forever.

**Architecture:** A new batch `goals.Prune` evaluates every goal on a mob template under one lock — removing satisfied/expired goals and abandoning goals whose context score has been ~0 for `GoalAbandonDormantRounds` (tracked via a new per-goal `DormantSinceRound`). The existing per-round goals tick calls `Prune` on a staggered cadence before `Recompute`.

**Tech Stack:** Go, testify-free table tests (this package uses plain `t.Error`), YAML persistence.

**Spec:** `docs/superpowers/specs/2026-05-29-mob-aliveness-4.6-goal-pruning-design.md`

---

## File Structure

| File | Change |
|------|--------|
| `internal/goals/types.go` | Add `DormantSinceRound uint64` to `Goal` |
| `internal/configs/config.balance.go` | Add `GoalPruneIntervalRounds`, `GoalAbandonDormantRounds` |
| `internal/configs/config.balance.mobs.go` | Defaults (50, 600) |
| `internal/goals/prune.go` (new) | `Prune`, `PruneReason`, `PruneRecord`, `abandonDormantRoundsConfig` |
| `internal/goals/prune_test.go` (new) | Prune behavior tests |
| `internal/hooks/NewRound_MobRoundTick.go` | `shouldPruneGoals` gate + call `Prune` in `tickMobRecomputeGoals` |
| `internal/hooks/NewRound_MobRoundTick_RecomputeGoals_test.go` | `shouldPruneGoals` test |

---

### Task 1: Add `DormantSinceRound` to Goal

**Files:**
- Modify: `internal/goals/types.go`
- Test: `internal/goals/prune_test.go` (new — first test lands here)

- [ ] **Step 1: Write the failing test**

Create `internal/goals/prune_test.go`:

```go
package goals

import (
	"testing"
	"time"
)

func TestGoal_DormantSinceRound_Persists(t *testing.T) {
	ClearCache()
	resetRegistry()
	mg := &MobGoals{MobId: 99201, NextGoalId: 2, Goals: []*Goal{
		{Id: "g1", Type: "revenge", Priority: 70, CreatedAt: time.Now().UTC(), DormantSinceRound: 4242},
	}}
	cacheStoreForTest("dormant-mob", mg)
	if err := saveToDisk(99201, "dormant-mob"); err != nil {
		t.Fatalf("save: %v", err)
	}
	ClearCache()
	got := GoalsOf(99201, "dormant-mob")
	if len(got) != 1 {
		t.Fatalf("want 1 goal, got %d", len(got))
	}
	if got[0].DormantSinceRound != 4242 {
		t.Errorf("DormantSinceRound not persisted: got %d", got[0].DormantSinceRound)
	}
}
```

- [ ] **Step 2: Run it — expect FAIL (compile error)**

Run: `go test ./internal/goals/ -run TestGoal_DormantSinceRound_Persists`
Expected: FAIL — `unknown field 'DormantSinceRound' in struct literal`.

- [ ] **Step 3: Add the field**

In `internal/goals/types.go`, in the `Goal` struct, after the `ExpiresAt` line:

```go
	ExpiresAt         time.Time      `yaml:"expires_at,omitempty"`
	DormantSinceRound uint64         `yaml:"dormant_since_round,omitempty"` // chunk 4.6 — round dormancy began; 0 = live
```

- [ ] **Step 4: Run it — expect PASS**

Run: `go test ./internal/goals/ -run TestGoal_DormantSinceRound_Persists`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/goals/types.go internal/goals/prune_test.go
git commit -m "feat(goals): add Goal.DormantSinceRound for 4.6 pruning

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: Config knobs + defaults

**Files:**
- Modify: `internal/configs/config.balance.go`
- Modify: `internal/configs/config.balance.mobs.go`

- [ ] **Step 1: Add the fields**

In `internal/configs/config.balance.go`, immediately after the `GoalSelectTickEnabled` field (the end of the goal-selection block, ~line 362):

```go
	// GoalPruneIntervalRounds is how often (in rounds) the per-mob goal
	// prune sweep runs. Staggered per mob to avoid a synchronized spike.
	// chunk 4.6.
	GoalPruneIntervalRounds ConfigInt `yaml:"GoalPruneIntervalRounds"`

	// GoalAbandonDormantRounds is how many consecutive rounds a goal's
	// context score may stay at ~0 before it is abandoned (pruned).
	// chunk 4.6.
	GoalAbandonDormantRounds ConfigInt `yaml:"GoalAbandonDormantRounds"`
```

- [ ] **Step 2: Add defaults**

In `internal/configs/config.balance.mobs.go`, in the `// ── GOAL SELECTION ──` block, after the `GoalSelectTickEnabled` guard (~line 138):

```go
	if b.GoalPruneIntervalRounds < 1 {
		b.GoalPruneIntervalRounds = 50
	}
	if b.GoalAbandonDormantRounds < 1 {
		b.GoalAbandonDormantRounds = 600
	}
```

- [ ] **Step 3: Build to verify it compiles**

Run: `go build ./internal/configs/`
Expected: no output (success).

- [ ] **Step 4: Commit**

```bash
git add internal/configs/config.balance.go internal/configs/config.balance.mobs.go
git commit -m "feat(config): add GoalPruneIntervalRounds + GoalAbandonDormantRounds

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: `goals.Prune` batch sweep

**Files:**
- Create: `internal/goals/prune.go`
- Test: `internal/goals/prune_test.go` (extend)

- [ ] **Step 1: Write the failing tests**

Append to `internal/goals/prune_test.go` (the `import` block already has `testing` and `time`; add `"github.com/GoMudEngine/GoMud/internal/mobs"`):

```go
// pruneCtxScore lets tests drive a goal type's context score.
var pruneCtxScore = 1.0

func registerPruneType(t *testing.T) {
	t.Helper()
	resetRegistry()
	RegisterGoalType("prune-sat", GoalTypeMeta{
		Predicate:    func(g *Goal, m *mobs.Mob) bool { return true },
		ContextScore: func(g *Goal, m *mobs.Mob) float64 { return 1.0 },
	})
	RegisterGoalType("prune-live", GoalTypeMeta{
		Predicate:    func(g *Goal, m *mobs.Mob) bool { return false },
		ContextScore: func(g *Goal, m *mobs.Mob) float64 { return pruneCtxScore },
	})
}

func seedGoals(t *testing.T, mobId int, name string, gs ...*Goal) {
	t.Helper()
	ClearCache()
	mg := &MobGoals{MobId: mobId, NextGoalId: len(gs) + 1, Goals: gs}
	cacheStoreForTest(name, mg)
	if err := saveToDisk(mobId, name); err != nil {
		t.Fatalf("seed save: %v", err)
	}
}

func TestPrune_RemovesSatisfied(t *testing.T) {
	registerPruneType(t)
	seedGoals(t, 99301, "p-sat",
		&Goal{Id: "g1", Type: "prune-sat", Priority: 50, CreatedAt: time.Now().UTC()})
	recs := Prune(99301, "p-sat", nil, time.Now().UTC(), 1000)
	if len(recs) != 1 || recs[0].Reason != ReasonSatisfied {
		t.Fatalf("want 1 satisfied record, got %+v", recs)
	}
	if len(GoalsOf(99301, "p-sat")) != 0 {
		t.Error("satisfied goal not removed")
	}
}

func TestPrune_RemovesExpired(t *testing.T) {
	registerPruneType(t)
	seedGoals(t, 99302, "p-exp",
		&Goal{Id: "g1", Type: "prune-live", Priority: 50, CreatedAt: time.Now().UTC(),
			ExpiresAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)})
	recs := Prune(99302, "p-exp", nil, time.Now().UTC(), 1000)
	if len(recs) != 1 || recs[0].Reason != ReasonExpired {
		t.Fatalf("want 1 expired record, got %+v", recs)
	}
}

func TestPrune_AbandonsDormantPastThreshold(t *testing.T) {
	registerPruneType(t)
	pruneCtxScore = 0.0 // dormant
	defer func() { pruneCtxScore = 1.0 }()
	// DormantSinceRound already old enough: 1000 - 100 = 900 >= 600 default.
	seedGoals(t, 99303, "p-dorm",
		&Goal{Id: "g1", Type: "prune-live", Priority: 50, CreatedAt: time.Now().UTC(),
			DormantSinceRound: 100})
	recs := Prune(99303, "p-dorm", nil, time.Now().UTC(), 1000)
	if len(recs) != 1 || recs[0].Reason != ReasonAbandoned {
		t.Fatalf("want 1 abandoned record, got %+v", recs)
	}
}

func TestPrune_StampsDormancyButKeepsUnderThreshold(t *testing.T) {
	registerPruneType(t)
	pruneCtxScore = 0.0
	defer func() { pruneCtxScore = 1.0 }()
	seedGoals(t, 99304, "p-stamp",
		&Goal{Id: "g1", Type: "prune-live", Priority: 50, CreatedAt: time.Now().UTC()})
	recs := Prune(99304, "p-stamp", nil, time.Now().UTC(), 1000)
	if len(recs) != 0 {
		t.Fatalf("want no removals, got %+v", recs)
	}
	got := GoalsOf(99304, "p-stamp")
	if len(got) != 1 || got[0].DormantSinceRound != 1000 {
		t.Fatalf("want DormantSinceRound stamped to 1000, got %+v", got)
	}
}

func TestPrune_ClearsDormancyWhenLiveAgain(t *testing.T) {
	registerPruneType(t)
	pruneCtxScore = 1.0 // live
	seedGoals(t, 99305, "p-clear",
		&Goal{Id: "g1", Type: "prune-live", Priority: 50, CreatedAt: time.Now().UTC(),
			DormantSinceRound: 100})
	recs := Prune(99305, "p-clear", nil, time.Now().UTC(), 1000)
	if len(recs) != 0 {
		t.Fatalf("want no removals, got %+v", recs)
	}
	got := GoalsOf(99305, "p-clear")
	if len(got) != 1 || got[0].DormantSinceRound != 0 {
		t.Fatalf("want DormantSinceRound cleared, got %+v", got)
	}
}

func TestPrune_BatchRemovesMultiple(t *testing.T) {
	registerPruneType(t)
	seedGoals(t, 99306, "p-batch",
		&Goal{Id: "g1", Type: "prune-sat", Priority: 50, CreatedAt: time.Now().UTC()},
		&Goal{Id: "g2", Type: "prune-live", Priority: 40, CreatedAt: time.Now().UTC(),
			ExpiresAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)})
	recs := Prune(99306, "p-batch", nil, time.Now().UTC(), 1000)
	if len(recs) != 2 {
		t.Fatalf("want 2 removals, got %+v", recs)
	}
	if len(GoalsOf(99306, "p-batch")) != 0 {
		t.Error("expected all dead goals removed")
	}
}
```

- [ ] **Step 2: Run them — expect FAIL**

Run: `go test ./internal/goals/ -run TestPrune`
Expected: FAIL — `undefined: Prune`, `undefined: ReasonSatisfied`, etc.

- [ ] **Step 3: Implement `prune.go`**

Create `internal/goals/prune.go`:

```go
package goals

import (
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// PruneReason categorizes why a goal was removed by Prune.
type PruneReason string

const (
	ReasonSatisfied PruneReason = "satisfied"
	ReasonExpired   PruneReason = "expired"
	ReasonAbandoned PruneReason = "abandoned-stale"
)

// PruneRecord reports one removed goal (returned for logging/tests).
type PruneRecord struct {
	GoalId string
	Type   string
	Reason PruneReason
}

// Prune evaluates every goal on the mob template and removes those that
// are satisfied, expired, or abandoned-stale (context score ~0 for at
// least GoalAbandonDormantRounds). Per-goal DormantSinceRound stamps are
// updated in place. Persists once if anything changed and triggers one
// Recompute if any goal was removed. Safe with a nil mob.
func Prune(mobId int, namesimple string, mob *mobs.Mob, now time.Time, nowRound uint64) []PruneRecord {
	mg := loadOrLazyInit(mobId, namesimple)
	abandonRounds := abandonDormantRoundsConfig()

	cacheMu.Lock()
	var records []PruneRecord
	removeIds := map[string]bool{}
	changed := false

	for _, g := range mg.Goals {
		if IsSatisfied(g, mob) {
			records = append(records, PruneRecord{g.Id, g.Type, ReasonSatisfied})
			removeIds[g.Id] = true
			continue
		}
		if IsExpired(g, now) {
			records = append(records, PruneRecord{g.Id, g.Type, ReasonExpired})
			removeIds[g.Id] = true
			continue
		}
		if effectiveContextMod(g, mob) > 0 {
			if g.DormantSinceRound != 0 {
				g.DormantSinceRound = 0
				changed = true
			}
			continue
		}
		// Dormant (score == 0).
		if g.DormantSinceRound == 0 {
			g.DormantSinceRound = nowRound
			changed = true
		}
		if abandonRounds > 0 && nowRound >= g.DormantSinceRound &&
			nowRound-g.DormantSinceRound >= uint64(abandonRounds) {
			records = append(records, PruneRecord{g.Id, g.Type, ReasonAbandoned})
			removeIds[g.Id] = true
		}
	}

	if len(removeIds) > 0 {
		out := mg.Goals[:0:0]
		for _, g := range mg.Goals {
			if !removeIds[g.Id] {
				out = append(out, g)
			}
		}
		mg.Goals = out
		if removeIds[mg.CurrentGoalId] {
			mg.CurrentGoalId = ""
			mg.CurrentSinceRound = 0
			mg.LastSwitchRound = 0
		}
		changed = true
	}
	nameByMobId[mobId] = namesimple
	cacheMu.Unlock()

	if changed {
		if err := saveToDisk(mobId, namesimple); err != nil {
			mudlog.Warn("goals.Prune: save failed", "mob_id", mobId, "error", err)
		}
	}
	for _, r := range records {
		mudlog.Debug("goals.prune",
			"mob_id", mobId,
			"goal", r.GoalId,
			"type", r.Type,
			"reason", string(r.Reason),
			"round", nowRound)
	}
	if len(removeIds) > 0 {
		Recompute(mobId, namesimple, mob, nowRound)
	}
	return records
}

// abandonDormantRoundsConfig reads GoalAbandonDormantRounds, defaulting
// to 600 if unset/invalid (mirrors the config validate default).
func abandonDormantRoundsConfig() int {
	v := int(configs.GetBalanceConfig().GoalAbandonDormantRounds)
	if v < 1 {
		v = 600
	}
	return v
}
```

- [ ] **Step 4: Run them — expect PASS**

Run: `go test ./internal/goals/ -run TestPrune`
Expected: PASS (all six).

- [ ] **Step 5: Run the whole goals package (no regressions)**

Run: `go test ./internal/goals/...`
Expected: ok.

- [ ] **Step 6: Commit**

```bash
git add internal/goals/prune.go internal/goals/prune_test.go
git commit -m "feat(goals): Prune sweep — satisfied/expired/abandoned-stale

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: Tick integration (throttle + call Prune)

**Files:**
- Modify: `internal/hooks/NewRound_MobRoundTick.go`
- Test: `internal/hooks/NewRound_MobRoundTick_RecomputeGoals_test.go`

- [ ] **Step 1: Write the failing test for the cadence gate**

Append to `internal/hooks/NewRound_MobRoundTick_RecomputeGoals_test.go`:

```go
func TestShouldPruneGoals_CadenceAndStagger(t *testing.T) {
	// interval 0 disables.
	if shouldPruneGoals(100, 5, 0) {
		t.Error("interval 0 must disable pruning")
	}
	// (nowRound + templateId) % interval == 0 → prune.
	// nowRound=45, templateId=5, interval=50 → 50%50==0 → true.
	if !shouldPruneGoals(45, 5, 50) {
		t.Error("expected prune on cadence boundary")
	}
	// nowRound=46, templateId=5, interval=50 → 51%50==1 → false.
	if shouldPruneGoals(46, 5, 50) {
		t.Error("expected no prune off cadence")
	}
}
```

- [ ] **Step 2: Run it — expect FAIL**

Run: `go test ./internal/hooks/ -run TestShouldPruneGoals_CadenceAndStagger`
Expected: FAIL — `undefined: shouldPruneGoals`.

- [ ] **Step 3: Add the gate helper and wire the tick**

In `internal/hooks/NewRound_MobRoundTick.go`, add the helper (near `tickMobRecomputeGoals`):

```go
// shouldPruneGoals gates the 4.6 prune sweep: runs every `interval`
// rounds, staggered per mob template so all mobs don't prune on the same
// tick. interval <= 0 disables pruning.
func shouldPruneGoals(nowRound uint64, templateId, interval int) bool {
	if interval <= 0 {
		return false
	}
	return (nowRound+uint64(templateId))%uint64(interval) == 0
}
```

Then, inside `tickMobRecomputeGoals`, between the `GoalsOf` empty-check and the `Recompute` call, insert the prune call:

```go
	if len(goals.GoalsOf(templateId, name)) == 0 {
		return // cheap path: no goals to select among
	}
	if shouldPruneGoals(nowRound, templateId, int(configs.GetBalanceConfig().GoalPruneIntervalRounds)) {
		goals.Prune(templateId, name, mob, time.Now().UTC(), nowRound)
	}
	goals.Recompute(templateId, name, mob, nowRound)
```

Ensure `time` is imported in the file (add `"time"` to the import block if not present).

- [ ] **Step 4: Run the gate test + the existing recompute tests — expect PASS**

Run: `go test ./internal/hooks/ -run 'TestShouldPruneGoals|RecomputeGoals'`
Expected: PASS.

- [ ] **Step 5: Build**

Run: `go build ./...`
Expected: no output.

- [ ] **Step 6: Commit**

```bash
git add internal/hooks/NewRound_MobRoundTick.go internal/hooks/NewRound_MobRoundTick_RecomputeGoals_test.go
git commit -m "feat(goals): run Prune on staggered cadence in the goals tick

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: Verification

**Files:** none (verification only)

- [ ] **Step 1: Build**

Run: `go build ./...`
Expected: no output.

- [ ] **Step 2: Run goals + hooks suites**

Run: `go test ./internal/goals/... ./internal/hooks/`
Expected: ok for both (note: `hooks` has a known pre-existing flaky fold-casting test unrelated to goals — if `TestHandlePlayerFoldCasting_*` panics on a nil SpellBook, re-run `go test ./internal/hooks/` to confirm it's the flake, not this change).

- [ ] **Step 3: Boot smoke (wipe instance saves per SOP)**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . > /tmp/4.6_boot.log 2>&1 &
```
Poll the log for `Server Ready` or a panic; confirm `Config name="Balance.GoalPruneIntervalRounds" value=50` and `...GoalAbandonDormantRounds value=600` appear, the goal files load without panic, and there is no `panic`/`fatal`. Then stop:

```bash
taskkill //IM "GoMud.exe" //F
```

---

## Notes for the implementer

- **Dormancy keys on context score, not selection.** `effectiveContextMod(g, mob)` returns 1.0 when a type has no `ContextScore` registered, so such goals never go dormant — only satisfied/expired prune them. That is intended.
- **No per-tick churn:** `DormantSinceRound` changes only on live↔dormant transitions, so the file is written only on transitions or removals.
- **Double Recompute is fine:** `Prune` triggers a Recompute when it removes goals, and the tick calls `Recompute` again right after — the second call is a no-op when nothing changed (idempotent by design).
- **Ongoing goals** (`protection-mob`/`protection-faction`, predicate always false) are retired via dormancy once their target is dead (their ContextScore returns 0), matching the existing `// 4.6's pruning sweep` comment.
