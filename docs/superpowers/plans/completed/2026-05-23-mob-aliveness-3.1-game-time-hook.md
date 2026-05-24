# Mob Aliveness 3.1 — Game-time Hook Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend the existing `time_of_day` btree condition with a `range:` parameter so chunk 3.2 schedules can author hour-precise gates like "smith works 9-17".

**Architecture:** Single-file modification (`internal/behaviortree/conditions_state.go`). Adds a string-format hour-range parser + wrap-around comparator. Existing binary `period: day` / `period: night` form stays intact for backward compatibility. New `range:` takes precedence when both are set.

**Tech Stack:** Go 1.24, existing `internal/gametime/` package, existing `internal/behaviortree/` condition registry, `internal/util/SetRoundCountForTest` for deterministic test time.

**Spec:** `docs/superpowers/specs/2026-05-23-mob-aliveness-3.1-game-time-hook-design.md`

**Branch:** `feature/mob-aliveness-3.1-game-time-hook` (already created; spec committed as `af0e5dfa`).

---

## Stage map

| Stage | Task | Description |
|---|---|---|
| 1 | T1 | Extend `condTimeOfDay` with `range:` parameter + parser + tests |
| 2 | T2 | Update `context.md` documentation |
| 3 | T3 | Roadmap closeout (mark 3.1 Done) |

3 tasks. Truly small chunk.

---

## Task 1: Extend `condTimeOfDay` with `range:` parameter

**Files:**
- Modify: `internal/behaviortree/conditions_state.go` (extend `condTimeOfDay` at line 64; add `parseHourRange` + `inHourRange` helpers)
- Create: `internal/behaviortree/conditions_state_test.go` (does not currently exist)

- [ ] **Step 1: Read the current `condTimeOfDay` and surrounding context**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
sed -n '60,80p' internal/behaviortree/conditions_state.go
```

Confirm the current signature and binary `period:` switch. The function is the entry point we're extending; the new `range:` parameter handling goes BEFORE the existing `period:` switch (range takes precedence per spec).

Also check what test-helper patterns exist in the package:

```bash
grep -l "func newTestMobInst\|func newTestContext\|util.SetRoundCountForTest" internal/behaviortree/*_test.go | head -5
```

`util.SetRoundCountForTest(r uint64)` exists in `internal/util/util.go:128` — use it to make `gametime.GetDate()` deterministic in tests.

- [ ] **Step 2: Write the failing tests**

Create `internal/behaviortree/conditions_state_test.go`:

```go
package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// timeOfDayRoundForHour24 returns a round number such that gametime.GetDate()
// reports Hour24 == hour. Uses the current TimingConfig.RoundsPerDay so it
// stays correct if the default changes.
//
// At the production default RoundsPerDay=20, this gives ~0.83 rounds per
// hour, so the math floors precisely: hour 0 -> round 0, hour 6 -> round 5,
// hour 12 -> round 10, hour 18 -> round 15.
func timeOfDayRoundForHour24(hour int) uint64 {
	c := configs.GetTimingConfig()
	roundsPerDay := uint64(c.RoundsPerDay)
	// roundOfDay = floor(hour * roundsPerDay / 24); the +1 nudges us past
	// integer-boundary edges so hour 9 reads as exactly hour 9, not hour 8.
	return (uint64(hour)*roundsPerDay)/24 + 1
}

// setTestHour pins the global round count so subsequent gametime queries
// resolve to a known hour. Returns a cleanup func the caller defers.
func setTestHour(t *testing.T, hour int) func() {
	t.Helper()
	prev := util.GetRoundCount()
	util.SetRoundCountForTest(timeOfDayRoundForHour24(hour))
	return func() { util.SetRoundCountForTest(prev) }
}

func TestCondTimeOfDay_BinaryPeriod_Day(t *testing.T) {
	defer setTestHour(t, 12)() // noon

	res := condTimeOfDay(map[string]any{"period": "day"}, nil)
	if res != Success {
		t.Errorf("expected Success at noon with period=day, got %v", res)
	}
}

func TestCondTimeOfDay_BinaryPeriod_Night(t *testing.T) {
	defer setTestHour(t, 0)() // midnight

	res := condTimeOfDay(map[string]any{"period": "night"}, nil)
	if res != Success {
		t.Errorf("expected Success at midnight with period=night, got %v", res)
	}
}

func TestCondTimeOfDay_Range_BasicMatch(t *testing.T) {
	defer setTestHour(t, 10)() // 10am

	res := condTimeOfDay(map[string]any{"range": "9-17"}, nil)
	if res != Success {
		t.Errorf("expected Success at 10am with range=9-17, got %v", res)
	}
}

func TestCondTimeOfDay_Range_BeforeWindow(t *testing.T) {
	defer setTestHour(t, 8)() // 8am

	res := condTimeOfDay(map[string]any{"range": "9-17"}, nil)
	if res != Failure {
		t.Errorf("expected Failure at 8am with range=9-17, got %v", res)
	}
}

func TestCondTimeOfDay_Range_AtExclusiveEnd(t *testing.T) {
	defer setTestHour(t, 17)() // 5pm — exclusive end, should fail

	res := condTimeOfDay(map[string]any{"range": "9-17"}, nil)
	if res != Failure {
		t.Errorf("expected Failure at 17:00 (exclusive end) with range=9-17, got %v", res)
	}
}

func TestCondTimeOfDay_Range_WrapAroundMidnight_Success(t *testing.T) {
	tests := []int{23, 0, 5}
	for _, hour := range tests {
		t.Run("", func(t *testing.T) {
			defer setTestHour(t, hour)()

			res := condTimeOfDay(map[string]any{"range": "22-6"}, nil)
			if res != Success {
				t.Errorf("expected Success at hour %d with range=22-6 (wrap), got %v", hour, res)
			}
		})
	}
}

func TestCondTimeOfDay_Range_WrapAroundMidnight_Failure(t *testing.T) {
	tests := []int{6, 7, 21}
	for _, hour := range tests {
		t.Run("", func(t *testing.T) {
			defer setTestHour(t, hour)()

			res := condTimeOfDay(map[string]any{"range": "22-6"}, nil)
			if res != Failure {
				t.Errorf("expected Failure at hour %d with range=22-6 (wrap), got %v", hour, res)
			}
		})
	}
}

func TestCondTimeOfDay_Range_FullDay_AlwaysSuccess(t *testing.T) {
	defer setTestHour(t, 0)()

	res := condTimeOfDay(map[string]any{"range": "0-24"}, nil)
	if res != Success {
		t.Errorf("expected Success with range=0-24 (full day), got %v", res)
	}
}

func TestCondTimeOfDay_Range_EmptyRange_AlwaysFailure(t *testing.T) {
	defer setTestHour(t, 5)()

	res := condTimeOfDay(map[string]any{"range": "5-5"}, nil)
	if res != Failure {
		t.Errorf("expected Failure with range=5-5 (empty), got %v", res)
	}
}

func TestCondTimeOfDay_Range_Empty_FallsBackToPeriod(t *testing.T) {
	defer setTestHour(t, 12)() // noon

	res := condTimeOfDay(map[string]any{"range": "", "period": "day"}, nil)
	if res != Success {
		t.Errorf("expected Success: empty range should fall back to period=day, got %v", res)
	}
}

func TestCondTimeOfDay_Range_Malformed_AbcReturnsFailure(t *testing.T) {
	defer setTestHour(t, 12)()

	res := condTimeOfDay(map[string]any{"range": "abc"}, nil)
	if res != Failure {
		t.Errorf("expected Failure with malformed range=abc, got %v", res)
	}
}

func TestCondTimeOfDay_Range_OutOfBounds_25(t *testing.T) {
	defer setTestHour(t, 12)()

	res := condTimeOfDay(map[string]any{"range": "9-25"}, nil)
	if res != Failure {
		t.Errorf("expected Failure with out-of-bounds range=9-25, got %v", res)
	}
}

func TestCondTimeOfDay_Range_MissingEnd(t *testing.T) {
	defer setTestHour(t, 12)()

	res := condTimeOfDay(map[string]any{"range": "9"}, nil)
	if res != Failure {
		t.Errorf("expected Failure with malformed range=9 (missing end), got %v", res)
	}
}

func TestCondTimeOfDay_Range_NegativeStart(t *testing.T) {
	defer setTestHour(t, 12)()

	res := condTimeOfDay(map[string]any{"range": "-1-17"}, nil)
	if res != Failure {
		t.Errorf("expected Failure with negative-start range=-1-17, got %v", res)
	}
}

func TestCondTimeOfDay_Range_TakesPrecedenceOverPeriod(t *testing.T) {
	// At midnight, period=day would Fail, but range=22-6 (wrap) wins → Success.
	defer setTestHour(t, 0)()

	res := condTimeOfDay(map[string]any{
		"period": "day",
		"range":  "22-6",
	}, nil)
	if res != Success {
		t.Errorf("expected Success: range takes precedence over period, got %v", res)
	}

	// And the reverse: at 10am, range=22-6 would Fail; range wins → Failure.
	defer setTestHour(t, 10)()

	res = condTimeOfDay(map[string]any{
		"period": "day",
		"range":  "22-6",
	}, nil)
	if res != Failure {
		t.Errorf("expected Failure at 10am: range=22-6 wins over period=day, got %v", res)
	}
}
```

- [ ] **Step 3: Run tests, expect failures**

```bash
go test ./internal/behaviortree/ -run TestCondTimeOfDay -v 2>&1 | tail -30
```

Expected: most range-related tests FAIL because `condTimeOfDay` doesn't yet recognize the `range:` parameter. The two binary-period tests (Day, Night) should PASS — they test existing behavior.

- [ ] **Step 4: Extend `condTimeOfDay` with `range:` handling**

Replace the existing `condTimeOfDay` in `internal/behaviortree/conditions_state.go` (around line 64) with the extended version. Add the two helper functions (`parseHourRange`, `inHourRange`) near the bottom of the file or just below `condTimeOfDay`.

```go
// loggedTimeOfDayMisconfigs tracks already-logged misconfigured range
// strings so warnings don't spam every btree tick. Lock-free reads via
// sync.Map are fine for this low-cardinality map.
var loggedTimeOfDayMisconfigs sync.Map

func condTimeOfDay(params map[string]any, ctx *EvalContext) Result {
	rangeStr := getStringParam(params, "range")

	// New: range parameter takes precedence over period when both set.
	if rangeStr != "" {
		start, end, valid := parseHourRange(rangeStr)
		if !valid {
			// Log once per malformed range string; subsequent ticks stay silent.
			if _, already := loggedTimeOfDayMisconfigs.LoadOrStore("err:"+rangeStr, true); !already {
				mudlog.Error("time_of_day",
					"error", "invalid `range` parameter",
					"value", rangeStr)
			}
			return Failure
		}

		// Edge: empty range (start == end) always Failure.
		if start == end {
			if _, already := loggedTimeOfDayMisconfigs.LoadOrStore("empty:"+rangeStr, true); !already {
				mudlog.Warn("time_of_day",
					"warn", "empty `range` (start == end) always returns Failure",
					"value", rangeStr)
			}
			return Failure
		}

		// Edge: full-day range (0-24) always Success.
		if start == 0 && end == 24 {
			if _, already := loggedTimeOfDayMisconfigs.LoadOrStore("full:"+rangeStr, true); !already {
				mudlog.Warn("time_of_day",
					"warn", "full-day `range` (0-24) always returns Success — consider removing the gate",
					"value", rangeStr)
			}
			return Success
		}

		hour := gametime.GetDate().Hour24
		if inHourRange(hour, start, end) {
			return Success
		}
		return Failure
	}

	// Existing binary form (unchanged).
	period := getStringParam(params, "period")
	isNight := gametime.IsNight()
	switch strings.ToLower(period) {
	case "night":
		if isNight {
			return Success
		}
	case "day":
		if !isNight {
			return Success
		}
	}
	return Failure
}

// parseHourRange parses a "start-end" hour string. Hours are 0-23 24h
// format, with end=24 allowed to mean "end of day". Returns (start, end,
// true) on success; returns (0, 0, false) on any malformed input.
//
// Wrap-around (start > end, e.g., "22-6") is allowed and signaled by the
// caller treating the inequality differently via inHourRange.
func parseHourRange(s string) (int, int, bool) {
	parts := strings.SplitN(s, "-", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	startStr := strings.TrimSpace(parts[0])
	endStr := strings.TrimSpace(parts[1])
	if startStr == "" || endStr == "" {
		return 0, 0, false
	}
	start, err := strconv.Atoi(startStr)
	if err != nil {
		return 0, 0, false
	}
	end, err := strconv.Atoi(endStr)
	if err != nil {
		return 0, 0, false
	}
	if start < 0 || start > 23 {
		return 0, 0, false
	}
	if end < 0 || end > 24 {
		return 0, 0, false
	}
	return start, end, true
}

// inHourRange reports whether hour falls within [start, end). Handles
// wrap-around when start > end (e.g., 22-6 covers 22, 23, 0, 1, 2, 3, 4, 5).
func inHourRange(hour, start, end int) bool {
	if start <= end {
		return hour >= start && hour < end
	}
	// Wrap-around: hour is in range if at or after start, OR strictly before end.
	return hour >= start || hour < end
}
```

Add the new imports at the top of the file (after existing imports — `strings` and `gametime` and `util` are already there):

```go
import (
	"strconv"   // NEW
	"strings"
	"sync"      // NEW

	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/mudlog"  // NEW
	"github.com/GoMudEngine/GoMud/internal/util"
)
```

Verify `mudlog` is the right import path by checking how other condition files use it:

```bash
grep -l "mudlog" internal/behaviortree/*.go | head -3
grep "mudlog" internal/behaviortree/actions_mutation.go | head -3
```

If `actions_mutation.go` uses `mudlog.Error("try_mutation_active", ...)`, the same pattern applies here.

- [ ] **Step 5: Run tests, expect PASS**

```bash
go test ./internal/behaviortree/ -run TestCondTimeOfDay -v 2>&1 | tail -40
```

Expected: all 15+ TestCondTimeOfDay_* tests pass.

- [ ] **Step 6: Run the full behaviortree package + build**

```bash
go test ./internal/behaviortree/ 2>&1 | tail -5
go build ./...
```

Expected: PASS, no regressions, clean build.

- [ ] **Step 7: Commit**

```bash
git add internal/behaviortree/conditions_state.go \
        internal/behaviortree/conditions_state_test.go
git commit -m "$(cat <<'EOF'
feat(btree): time_of_day condition supports hour ranges

Extends the existing time_of_day condition with a `range:` parameter
for hour-precise time gating that chunk 3.2 schedules will need.

  range: "9-17"   -> 9 <= hour24 < 17  (working hours)
  range: "22-6"   -> 22 <= hour24 OR hour24 < 6  (wraps midnight)
  range: "0-24"   -> always Success + warning (likely misconfig)
  range: "5-5"    -> always Failure + warning (empty)

Existing `period: day` / `period: night` binary form unchanged.
When both `period:` and `range:` are set on the same node, range
takes precedence.

Malformed range strings (non-numeric, out-of-bounds, missing end)
log an error once per misconfig and return Failure; empty + full-day
ranges log a warning once and match deterministically (always-fail /
always-pass). Misconfig dedup via sync.Map so warnings don't spam.

Resolves chunk 3.1 from the mob aliveness roadmap.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Update `context.md` documentation

**Files:**
- Modify: `internal/behaviortree/context.md` (the `time_of_day` row in the conditions table around line ~?? — find by grep)

- [ ] **Step 1: Find the current `time_of_day` row**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
grep -n "time_of_day" internal/behaviortree/context.md
```

Note the line number for the conditions-table row.

- [ ] **Step 2: Read the row + surrounding rows to understand the table format**

```bash
grep -B 2 -A 2 "time_of_day" internal/behaviortree/context.md
```

The existing row is: `| time_of_day | period ("day" or "night") | In-game time of day. |`

- [ ] **Step 3: Update the row**

Replace the row with:

```markdown
| `time_of_day` | `period` ("day" or "night") OR `range` ("`<start_hour>-<end_hour>`", 24h format, e.g., `"9-17"`; wraps midnight when start > end). When both set, `range` takes precedence. | In-game time of day. Hour comparisons use `[start, end)` semantics (inclusive start, exclusive end). Empty range (`"5-5"`) always Failure; full-day range (`"0-24"`) always Success — both log a warning once. |
```

If the row format doesn't fit cleanly into a single line, split into a multi-line description but keep the table column structure intact.

- [ ] **Step 4: Verify the table still renders**

```bash
# Visual check that the table is still valid markdown:
sed -n '/^| `time_of_day`/,/^|/p' internal/behaviortree/context.md | head -5
```

If the table looks malformed (broken pipes, columns out of alignment), fix the whitespace.

- [ ] **Step 5: Commit**

```bash
git add internal/behaviortree/context.md
git commit -m "$(cat <<'EOF'
docs(btree): document time_of_day range parameter

Updates the conditions-table row to reflect the new `range:` parameter
added in the chunk 3.1 implementation. Documents precedence (range
wins over period when both set), inclusive-start / exclusive-end
semantics, and the empty/full-day edge-case behavior.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Roadmap closeout

**Files:**
- Modify: `MOB_ALIVENESS_ROADMAP.md`

- [ ] **Step 1: Update progress tracker row**

Find:

```markdown
| 3.1 | Routine | Game-time hook | S | — | Not started |
```

Change `Not started` → `Done`.

- [ ] **Step 2: Update chunk mini-brief status line**

Find:

```markdown
### 3.1 Game-time hook
**Status:** Not started • **Size:** S
```

Change to:

```markdown
### 3.1 Game-time hook
**Status:** Done (2026-05-23) • **Size:** S
```

- [ ] **Step 3: Append a `**Shipped:**` bullet**

After the existing `- **Why:** ...` line in the 3.1 section, add:

```markdown
- **Shipped:** Extended the existing `time_of_day` btree condition (`internal/behaviortree/conditions_state.go:64`) with a `range:` parameter for hour-precise time gating that chunk 3.2 schedules will use. Most of the roadmap requirements were already in place: `util.GetRoundCount()` provides the time tick, `gametime.IsNight()` + `GameDate.Night` provide the day/night flag, `configs.GetTimingConfig().RoundsPerDay` provides configurable day length, and `modules/time/time.go` already gives players a `time` command. The only new code is the `range:` parser + wrap-around comparator + tests. Single-task chunk; net ~80 LoC + ~180 LoC tests. Spec at `docs/superpowers/specs/2026-05-23-mob-aliveness-3.1-game-time-hook-design.md`, plan at `docs/superpowers/plans/2026-05-23-mob-aliveness-3.1-game-time-hook.md`. No in-game smoke required — pure condition primitive exercised only by tests in this chunk; chunk 3.2's schedule consumers will exercise it in-context.
```

- [ ] **Step 4: Update the roll-up line**

Find the current roll-up (probably reads `**Roll-up:** 18 / 41 done • 0 in progress • 23 not started.` after the chunk 2.10-followups closeout).

Read the current line:

```bash
grep -n "^\*\*Roll-up:" MOB_ALIVENESS_ROADMAP.md
```

Increment done by 1, decrement not-started by 1. Verify the math against the current state before committing.

- [ ] **Step 5: Commit**

```bash
git add MOB_ALIVENESS_ROADMAP.md
git commit -m "$(cat <<'EOF'
docs(3.1): mark mob-aliveness 3.1 Done

Phase 3 of the mob aliveness roadmap kicked off. Chunk 3.1 was a
small extension of existing infrastructure (the gametime package and
the time_of_day btree condition were already mostly in place).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Self-review checklist

**Spec coverage:**
- [x] Extend `time_of_day` condition with `range:` parameter — Task 1
- [x] Hour-range parser with wrap-around semantics — Task 1 (`parseHourRange` + `inHourRange`)
- [x] Validation: empty / full-day / malformed / out-of-bounds — Task 1 test cases + production code
- [x] Precedence: range wins over period when both set — Task 1 test + production code
- [x] Misconfig dedup via sync.Map — Task 1
- [x] Backward compat: existing `period: day/night` binary form unchanged — Task 1 regression tests
- [x] Documentation update in `context.md` — Task 2
- [x] Roadmap closeout — Task 3

**Placeholder scan:**
- No "TBD" / "TODO" / "fill in details"
- Test code is complete and runnable as written
- Commit messages use heredoc + Co-Authored-By
- The `mudlog` import line is verified via grep in Task 1 Step 4 (instructions to confirm before committing)

**Type consistency:**
- `parseHourRange` signature `(s string) (int, int, bool)` used consistently
- `inHourRange` signature `(hour, start, end int) bool` used consistently
- `loggedTimeOfDayMisconfigs sync.Map` referenced by the deduplication logic
- Test helper `setTestHour` returns a cleanup func, used via `defer setTestHour(t, hour)()` pattern consistently

Issues found inline: none.

---

## Execution handoff

**Plan complete and saved to `docs/superpowers/plans/2026-05-23-mob-aliveness-3.1-game-time-hook.md`.** Two execution options:

**1. Subagent-Driven (recommended)** — fresh subagent per task, review between tasks. Even for a 3-task chunk this gives the spec-compliance check, code-quality check, and isolated test execution.

**2. Inline Execution** — execute tasks in this session using executing-plans, batch execution with checkpoints. Reasonable for a chunk this small.

Which approach?
