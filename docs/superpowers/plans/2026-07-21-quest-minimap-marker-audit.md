# Quest Minimap-Marker Audit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give every quest step either a minimap marker or an explicit audited "no marker" (`map_target: -1`) decision, and add a CI gate so no future quest step ships un-audited.

**Architecture:** Reuse the per-step `map_target int` field with a new `-1` = deliberate-none sentinel. Fix `ResolveQuestTarget` so `-1` yields no marker and suppresses inference. Add an engine accessor to iterate all quests, then a `TestSmoke_*` gate asserting every non-`end` step resolves a marker or is `-1`. Then audit all 79 quest YAMLs to satisfy the gate, recording each step's disposition in a coverage ledger.

**Tech Stack:** Go 1.25, `internal/questengine`, `boot_smoke_test.go` (package main), YAML content under `_datafiles/world/dogmud/quests/`.

Spec: `docs/superpowers/specs/2026-07-21-quest-minimap-marker-audit-design.md`

---

## File structure

- `internal/questengine/map_target.go` — modify `ResolveQuestTarget` for the `-1` sentinel.
- `internal/questengine/map_target_test.go` — add the sentinel unit test.
- `internal/questengine/engine.go` — add `AllQuests()` iterator accessor.
- `boot_smoke_test.go` (repo root) — add `TestSmoke_EveryQuestStepHasMarkerDecision`.
- `_datafiles/world/dogmud/quests/*.yaml` — the audit edits.
- `docs/superpowers/specs/2026-07-21-quest-minimap-marker-audit-design.md` — append the coverage ledger.

---

### Task 1: `-1` sentinel in the resolver

**Files:**
- Modify: `internal/questengine/map_target.go:26-34`
- Test: `internal/questengine/map_target_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/questengine/map_target_test.go`:

```go
// A step with map_target -1 means "deliberately no marker" — it must resolve to
// 0 (no marker) AND suppress room_enter inference, so a stray inferable target
// does not re-introduce a marker the author explicitly removed.
func TestResolveQuestTarget_DeliberateNoneSentinel(t *testing.T) {
	e := NewEngine()
	e.RegisterQuest(&QuestDef{
		QuestId: 9998,
		Steps:   []QuestStep{{Id: "nowhere", MapTarget: -1}},
		Triggers: []TriggerDef{
			// A room_enter gated on the step that WOULD infer room 4242 —
			// proves -1 overrides inference.
			{Event: "room_enter", Room: 4242, Conditions: Conditions{Has: []string{"9998-nowhere"}}},
		},
	})
	if got := e.ResolveQuestTarget(9998, "nowhere"); got != 0 {
		t.Fatalf("map_target -1 must resolve to 0 (no marker, inference suppressed); got %d", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/questengine/ -run TestResolveQuestTarget_DeliberateNoneSentinel -v`
Expected: FAIL — current code returns `-1` (the `!= 0` branch) or `4242` (inference), not `0`.

- [ ] **Step 3: Implement the resolver change**

In `internal/questengine/map_target.go`, replace the step-1 block (currently
`if step.MapTarget != 0 { return step.MapTarget }`) with:

```go
	// 1. Explicit map_target on the current step.
	for _, step := range q.Steps {
		if step.Id == currentStep {
			if step.MapTarget > 0 {
				return step.MapTarget
			}
			if step.MapTarget == -1 {
				return 0 // deliberate no-marker: do NOT fall through to inference
			}
			break // map_target == 0 → fall through to room_enter inference
		}
	}
```

Update the doc comment's resolution-order list to note: "`-1` = deliberate no
marker (returns 0, suppresses inference)."

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/questengine/ -run 'TestResolveQuestTarget' -v`
Expected: PASS (new test + existing `TestResolveQuestTarget_*` all green).

- [ ] **Step 5: Commit**

```bash
git add internal/questengine/map_target.go internal/questengine/map_target_test.go
git commit -m "feat(questengine): map_target -1 = deliberate no-marker sentinel"
```

---

### Task 2: Engine iterator + the CI gate (produces the audit worklist)

**Files:**
- Modify: `internal/questengine/engine.go` (add `AllQuests`)
- Modify: `boot_smoke_test.go` (add the gate)

- [ ] **Step 1: Add the `AllQuests` accessor**

In `internal/questengine/engine.go`, after `GetQuest`:

```go
// AllQuests returns every registered quest (unordered). For audits/gates.
func (e *Engine) AllQuests() []*QuestDef {
	out := make([]*QuestDef, 0, len(e.quests))
	for _, q := range e.quests {
		out = append(out, q)
	}
	return out
}
```

(Confirm the map field name is `e.quests` — it is, per `map_target.go:21`.)

- [ ] **Step 2: Write the gate**

First read `boot_smoke_test.go` and reuse the SAME data-load setup its sibling
`TestSmoke_*` gates use (they load the real `_datafiles` tree via the loaders /
a shared helper). Then add:

```go
func TestSmoke_EveryQuestStepHasMarkerDecision(t *testing.T) {
	// <mirror the sibling gates' real-data load here so questengine is populated>
	eng := questengine.GetEngine()
	var violations []string
	for _, q := range eng.AllQuests() {
		if q.QuestId == 1000000 {
			continue // generic reference template, not live content
		}
		for _, step := range q.Steps {
			if step.Id == "end" {
				continue // resolver returns 0 for "end" by design
			}
			if step.MapTarget == -1 {
				continue // audited deliberate no-marker
			}
			if eng.ResolveQuestTarget(q.QuestId, step.Id) > 0 {
				continue // explicit or inferred marker
			}
			violations = append(violations, fmt.Sprintf(
				"  quest %d %q step %q: no marker (map_target 0, no room_enter inference) — set a room target or -1 with a reason comment",
				q.QuestId, q.Name, step.Id))
		}
	}
	if len(violations) > 0 {
		t.Fatalf("%d quest step(s) have no marker decision:\n%s",
			len(violations), strings.Join(violations, "\n"))
	}
}
```

Add `fmt`/`strings`/`questengine` imports if not already present.

- [ ] **Step 3: Run the gate to produce the worklist**

Run: `go test . -run TestSmoke_EveryQuestStepHasMarkerDecision -v 2>&1 | tee /tmp/marker_worklist.txt`
Expected: FAIL, listing every undecided step. **This list is the Task 3 worklist.**
Do NOT commit yet — the gate is red until Task 3 finishes.

---

### Task 3: The audit pass (judgment; batched by quest id)

This is content classification, not codegen — no pre-written YAML here. For each
quest (id order, all 79 except `1000000-generic_quest.yaml`), read its `steps` and
`triggers` and classify **each step**:

- **Already resolves** — explicit `map_target > 0`, OR a `room_enter` trigger whose
  `conditions.has` includes `"{questId}-{stepId}"`. → leave as-is. **Do NOT add a
  redundant explicit `map_target`.**
- **Markable but unresolved** — a single clear destination the mechanism can't infer
  (talk to NPC X / deliver to Z / kill the boss in room Y). → add `map_target: <room>`.
  **Verify the room id** against where that NPC/target/room actually is (grep the mob's
  room in `rooms/*` spawninfo, or the room file) before writing it.
- **Genuinely un-markable** — kill-N-anywhere, gather-from-many, learn-a-command
  tutorial step, or deliberately spoiler-hidden. → `map_target: -1` with an inline
  `# map_target: -1 — <reason>` comment.

- [ ] **Step 1: Work the worklist in id-order batches**

Process in batches (e.g. quests 1–20, 21–46, 48–79). For each quest: read it,
classify every step, apply edits. Record each step in the ledger (Step 3).

- [ ] **Step 2: Re-run the gate after each batch**

Run: `go test . -run TestSmoke_EveryQuestStepHasMarkerDecision 2>&1 | tail -20`
Track the shrinking violation count until it reaches 0.

- [ ] **Step 3: Fill the coverage ledger**

Append to the spec's "Coverage ledger" section, one row per quest:
`| questId name | step: disposition (target N / inferred / -1: reason) | ... |`
so every intentional blank is visible and auditable.

- [ ] **Step 4: Commit the audit edits (safe, additive) in batches**

```bash
git add _datafiles/world/dogmud/quests/<batch>.yaml docs/superpowers/specs/2026-07-21-quest-minimap-marker-audit-design.md
git commit -m "content(quests): minimap-marker audit — <id range>"
```

(Master stays green through these: the gate is not in the suite yet.)

---

### Task 4: Turn the gate green + boot test + land the gate

**Files:** Modify: `boot_smoke_test.go`, `internal/questengine/engine.go`

- [ ] **Step 1: Confirm the gate is green**

Run: `go test . -run TestSmoke_EveryQuestStepHasMarkerDecision -v`
Expected: PASS (0 violations).

- [ ] **Step 2: Full data-file boot test**

Run the built server against the real tree (nuke instance saves first), confirm
zero panics + `quests`/`questengine LoadDataFiles loadedCount=79` + `ValidateZone
errors=0`. (Marker edits are load-bearing YAML — a typo'd `map_target` would still
parse, but a broken step id would not; the boot + gate together catch it.)

- [ ] **Step 3: Commit the accessor + gate + ledger**

```bash
git add internal/questengine/engine.go boot_smoke_test.go docs/superpowers/specs/2026-07-21-quest-minimap-marker-audit-design.md
git commit -m "test(quests): gate every quest step on a marker decision (target or -1)"
```

- [ ] **Step 4: Run the full affected suites**

Run: `go test ./internal/questengine/ . -count=1`
Expected: PASS. Also `gofmt -l` + `go vet ./internal/questengine/ .` clean.

---

### Task 5: Adversarial harness spot-check (content playtest gate)

Marker DATA is server-side; the SVG render is the user's browser check. Verify the
data half for a couple of freshly-marked quests.

- [ ] **Step 1: Drive a local harness session** past a quest whose step got a new
  explicit `map_target`, and capture the `Char.Quests` GMCP — confirm the focused
  quest's `target_room` matches the room we set, and updates as steps advance.

- [ ] **Step 2: Confirm a `-1` step emits no target** — on a step marked `-1`,
  `Char.Quests` `target_room` should be 0/absent (no marker), not a stray room.

- [ ] **Step 3: Note findings**; fix any mismatched room id and re-run the gate.

---

## Self-review

- **Spec coverage:** sentinel (Task 1), resolver change (Task 1), CI gate (Tasks 2/4),
  per-step audit with inference-leaning rule (Task 3), scope/exclusions (Task 2 gate +
  Task 3 rules), coverage ledger (Task 3), harness spot-check (Task 5). All covered.
- **Placeholder scan:** the only non-literal is Task 2 Step 2's "mirror the sibling
  gates' load setup" — unavoidable without duplicating the existing helper here; the
  gate body itself is exact. Task 3 is judgment content, procedure given.
- **Type consistency:** `QuestDef.QuestId`, `QuestStep.Id`/`.MapTarget`, `Conditions.Has`,
  `Engine.AllQuests()`/`.ResolveQuestTarget()`, `e.quests` — all match the verified source.
```
