# NPC Greetings (5a) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the `greetings:` blocks already authored in 186 of 302 dialogue files reach players — an ambient welcome when a player enters the room — and close the gate gap that let an entire authored layer go silently unread.

**Architecture:** `Greeting` becomes a real field on `DialogueFile` (shaped to the existing YAML — zero content files change). Selection is a pure function; frequency rides the existing per-`(mobInstance, player)` in-process memory; the hook lands in `go.go`'s player-arrival section beside the conversation boost, delivering via the mob's own `say`. The dialogue parse sweep gains strict unknown-key detection so the next authored-but-unimplemented field fails CI instead of sleeping for months.

**Tech Stack:** Go, yaml.v2, testify in `internal/dialogue` where present (check the package's existing style first), playtest harness for the content gate.

**Spec:** `docs/superpowers/specs/2026-07-25-npc-greetings-5a-design.md`

---

## Verified reference (do not re-derive)

```go
dialogue.Load(mobId int, zone string) *DialogueFile   // loader.go:21 — lazy, cached
type DialogueFile struct                              // types.go:121 — MobId, Zone, DefaultMood, Patterns, Tree, Memory
dialogue.GetMood(mobInstanceId int, defaultMood string) Mood    // mood.go:8 — Mood is a plain string TYPE, no normalization exists
// engine.go:113 compares by type conversion:  if Mood(m) == currentMood
type PlayerMemory struct                              // memory.go:9 — VisitCount, LastVisitRound, UnlockedNodes, CurrentRootSeen, RecentTopics
dialogue.GetMemory(mobInstanceId, userId int) *PlayerMemory     // memory.go:25; memoryCache keyed (mobInstanceId<<32 | userId)
conversations.isFullyIdle(m MobConversant) bool       // state.go:299 — UNEXPORTED; combat/sleep/conversation/patrol
conversationadapter.AdaptMob(m *mobs.Mob) MobConversant
(m *mobs.Mob) Command(cmd string, waitSeconds ...float64)       // mobs.go:780 idiom
(r *Room) GetMobs(findTypes ...FindFlag) []int
user.Character.IsHidden()                             // go.go:123 sets isSneaking from it
```

The arrival insertion point is `internal/usercommands/go.go` ~702 — the block
commented "Chunk 3.6: player-arrival conversation boost". The greeting hook
goes **immediately before** it (spec §2: welcomed before NPCs talk amongst
themselves). Verify whether `isSneaking` (set at go.go:123) is still in scope
there; if not, call `user.Character.IsHidden()` directly.

**Suppression refinement over the spec (record, don't hide):** the spec's
table says "mid-schedule-activity" suppresses. The conversations predicate
`isFullyIdle` — the engine's existing definition of an unoccupied NPC — covers
combat, sleep, mid-conversation and patrol mid-walk, but not schedule
activities generally. That is the better rule: the greeting prose is written
to be delivered mid-work ("mind the shavings — pull up that stool"), so a
crafting shopkeeper *should* greet. Task 8 updates the spec table to name the
conversations predicate. Sleep segments are still suppressed via the
`Sleeping` buff, which schedule sleep applies.

---

## File Structure

| File | Responsibility |
|------|----------------|
| `boot_smoke_test.go` **(modify)** | strict unknown-key sweep for dialogue files |
| `internal/dialogue/types.go` **(modify)** | `Greeting` type + `DialogueFile.Greetings` |
| `internal/dialogue/greetings.go` **(create)** | `PickGreeting`, `HasGreeted`, `MarkGreeted` |
| `internal/dialogue/greetings_test.go` **(create)** | Unit tests |
| `internal/dialogue/memory.go` **(modify)** | `Greeted` flag on `PlayerMemory` |
| `internal/conversations/state.go` **(modify)** | export `IsFullyIdle` wrapper |
| `internal/usercommands/go.go` **(modify)** | arrival hook |
| one dialogue YAML **(modify)** | rewrite the single root-duplicate greeting |
| `docs/superpowers/specs/2026-07-25-npc-greetings-5a-design.md` **(modify)** | suppression-table refinement |

---

### Task 1: The gate first — strict dialogue sweep (RED on the real bug)

The gate lands **before** the field, so its first run fails on the actual
incident: 186 files reporting `greetings` unknown. That failure is the RED
proving the gate detects the class; Task 2 turns it green.

**Files:** Modify `boot_smoke_test.go`

- [ ] **Step 1: Extend the dialogue sweep with a strict probe**

In `TestSmoke_AllDialogueFilesParse`, after the existing lenient
`yaml.Unmarshal` succeeds for a file, add a strict pass:

```go
		// Lenient decode succeeded — now catch what it silently DROPPED.
		// Unknown keys are not an error under lenient decoding, which is how
		// 186 files' greetings: blocks went unread for months: the drift gate
		// never sees dialogue (lazy-loaded), and this test only checked that
		// unmarshal didn't error. Strict-probe the same bytes and fail on any
		// unknown key not in the baseline.
		var strictProbe dialogue.DialogueFile
		if strictErr := yaml.UnmarshalStrict(raw, &strictProbe); strictErr != nil {
			// unknownKeyRe already exists in this file (boot_smoke_test.go:254,
			// used by the drift gate): `field (...) not found in type (...)`.
			// Reuse it — do not define a near-duplicate.
			for _, m := range unknownKeyRe.FindAllStringSubmatch(strictErr.Error(), -1) {
				key := m[1] + "|" + m[2]
				if !knownIgnoredDialogueKeys[key] {
					t.Errorf("%s: authored key %q maps to no field on dialogue types — "+
						"the value is silently dropped. Add the field, fix the key, or "+
						"(if deliberate) baseline it in knownIgnoredDialogueKeys.", path, key)
					failed++
				}
			}
		}
```

And alongside the test, mirroring the drift-gate shape:

```go
// knownIgnoredDialogueKeys is the accepted baseline of dialogue YAML keys
// ("field|type", matching the drift gate's key shape) that map to no field.
// It starts EMPTY: the one historical entry (greetings, 186 files) was
// resolved by implementing the field, which is the preferred way to clear
// an entry.
var knownIgnoredDialogueKeys = map[string]bool{}
```

- [ ] **Step 2: Run to verify it fails on the real incident**

Run: `DOGMUD_BOOT_SMOKE=1 go test . -run TestSmoke_AllDialogueFilesParse -count=1 2>&1 | tail -5`
Expected: FAIL, with many `authored key "greetings"` errors (186 files).

- [ ] **Step 3: Commit the failing gate SKIPPED-GUARDED**

Do NOT commit a red gate. Hold this change uncommitted; Task 2 makes it green
and they commit together. (Recorded here so the checkbox flow is honest.)

---

### Task 2: The field — `Greeting` + `DialogueFile.Greetings`

**Files:** Modify `internal/dialogue/types.go`

- [ ] **Step 1: Add the types**

In `types.go`, near `Pattern`:

```go
// Greeting is one ambient line an NPC offers when a player arrives in its
// room. Authored in 186 dialogue files since long before the engine read
// them — the struct is shaped to the existing YAML, not the other way round.
type Greeting struct {
	Text  string   `yaml:"text"`
	Moods []string `yaml:"moods,omitempty"`
}
```

And on `DialogueFile` (after `DefaultMood`, matching authored file order):

```go
	Greetings   []Greeting   `yaml:"greetings,omitempty"`
```

- [ ] **Step 2: Run the Task 1 gate — now GREEN**

Run: `DOGMUD_BOOT_SMOKE=1 go test . -run TestSmoke_AllDialogueFilesParse -count=1 2>&1 | tail -3`
Expected: PASS with an empty baseline.

- [ ] **Step 3: Add the coverage-count assertion**

The probe measured exactly 186 files with a non-empty block. Pin it so the
field is not quietly matching a subset. In the same walk in
`TestSmoke_AllDialogueFilesParse`:

```go
		if len(df.Greetings) > 0 {
			greetingFiles++
		}
```

and after the walk:

```go
	// Measured 2026-07-25 by strict-unmarshal probe: 186 of 302 files author
	// a greetings block. If this number drops without content changes, the
	// field has stopped matching the authored shape.
	if greetingFiles != 186 {
		t.Errorf("files with a non-empty greetings block: got %d, want 186", greetingFiles)
	}
```

- [ ] **Step 4: Full run + commit both tasks together**

Run: `DOGMUD_BOOT_SMOKE=1 go test . -run TestSmoke_AllDialogueFilesParse -count=1 -v 2>&1 | tail -4 && go build ./...`
Expected: PASS, build clean.

```bash
git add boot_smoke_test.go internal/dialogue/types.go
git commit -m "feat(dialogue): read the greetings block + strict unknown-key gate"
```

---

### Task 3: Selection — `PickGreeting`

**Files:** Create `internal/dialogue/greetings.go`, `internal/dialogue/greetings_test.go`

- [ ] **Step 1: Write the failing test**

Read `internal/dialogue/engine_test.go` (or the package's existing tests)
first and match their assertion style. Then:

```go
package dialogue

import "testing"

func TestPickGreeting(t *testing.T) {
	gs := []Greeting{
		{Text: "grumpy welcome", Moods: []string{"grumpy"}},
		{Text: "friendly welcome", Moods: []string{"friendly", "cheerful"}},
		{Text: "plain welcome"}, // untagged — the unconditional fallback
	}

	// Mood match wins.
	if text, ok := PickGreeting(gs, "friendly"); !ok || text != "friendly welcome" {
		t.Errorf("friendly: got %q ok=%v", text, ok)
	}
	// Unknown mood falls back to the untagged line.
	if text, ok := PickGreeting(gs, "melancholy"); !ok || text != "plain welcome" {
		t.Errorf("fallback: got %q ok=%v", text, ok)
	}
	// All tagged, none matching, no untagged -> say nothing rather than
	// deliver a line written for a mood the NPC is not in.
	tagged := []Greeting{{Text: "x", Moods: []string{"grumpy"}}}
	if _, ok := PickGreeting(tagged, "friendly"); ok {
		t.Error("no matching mood and no untagged line must yield no greeting")
	}
	// Empty list.
	if _, ok := PickGreeting(nil, "friendly"); ok {
		t.Error("nil greetings must yield no greeting")
	}
}
```

- [ ] **Step 2: Run to verify RED**

Run: `go test ./internal/dialogue/ -run TestPickGreeting -v`
Expected: FAIL — `undefined: PickGreeting`

- [ ] **Step 3: Implement**

Create `internal/dialogue/greetings.go`. NOTE: `Mood` is a plain string TYPE
(mood.go) — `Mood(m)` in engine.go:113 is a type conversion, not a
normalizing call, and no normalization exists anywhere. Compare by direct
conversion, exactly as the pattern matcher does; do not invent normalization
the engine lacks. `GetMood` returns `Mood`, so the parameter is `Mood` (the
test's untyped string constants convert implicitly):

```go
package dialogue

// PickGreeting selects the greeting an NPC offers for its current mood:
// the first whose Moods contains the mood, else the first untagged line,
// else nothing — a line written for a mood the NPC is not in stays unsaid.
// Comparison is direct Mood conversion, the same idiom the pattern matcher
// uses (engine.go:113); no normalization exists in this package.
func PickGreeting(gs []Greeting, currentMood Mood) (string, bool) {
	for _, g := range gs {
		for _, m := range g.Moods {
			if Mood(m) == currentMood {
				return g.Text, true
			}
		}
	}
	for _, g := range gs {
		if len(g.Moods) == 0 {
			return g.Text, true
		}
	}
	return "", false
}
```

- [ ] **Step 4: Verify GREEN + commit**

Run: `go test ./internal/dialogue/ -run TestPickGreeting -v`
Expected: PASS

```bash
git add internal/dialogue/greetings.go internal/dialogue/greetings_test.go
git commit -m "feat(dialogue): mood-aware greeting selection"
```

---

### Task 4: Frequency — the `Greeted` flag

**Files:** Modify `internal/dialogue/memory.go`; extend `greetings.go` + test

- [ ] **Step 1: Write the failing test**

```go
func TestGreetedOncePerInstance(t *testing.T) {
	const userId = 777001
	if HasGreeted(555001, userId) {
		t.Fatal("fresh memory must not read as greeted")
	}
	MarkGreeted(555001, userId)
	if !HasGreeted(555001, userId) {
		t.Error("marked instance must read as greeted")
	}
	// A respawned mob has a NEW instance id — it greets again by design
	// (spec §3): the in-process memory is keyed per instance.
	if HasGreeted(555002, userId) {
		t.Error("a different instance id must greet afresh")
	}
	// A different player is greeted independently.
	if HasGreeted(555001, 777002) {
		t.Error("another player must be greeted independently")
	}
}
```

- [ ] **Step 2: RED**

Run: `go test ./internal/dialogue/ -run TestGreetedOncePerInstance -v`
Expected: FAIL — `undefined: HasGreeted`

- [ ] **Step 3: Implement**

`memory.go`: add to `PlayerMemory`:

```go
	Greeted         bool // this mob instance has greeted this player (5a; in-process, so once per boot)
```

`greetings.go`:

```go
// HasGreeted reports whether this mob instance already greeted this player.
func HasGreeted(mobInstanceId, userId int) bool {
	return GetMemory(mobInstanceId, userId).Greeted
}

// MarkGreeted records the greeting. In-process memory, so the horizon is one
// server boot — and a respawned mob (new instance id) greets afresh, which
// reads correctly for a killed-and-returned shopkeeper.
func MarkGreeted(mobInstanceId, userId int) {
	GetMemory(mobInstanceId, userId).Greeted = true
}
```

- [ ] **Step 4: GREEN + commit**

Run: `go test ./internal/dialogue/ -run TestGreetedOncePerInstance -v && go test ./internal/dialogue/`
Expected: PASS, package green.

```bash
git add internal/dialogue/memory.go internal/dialogue/greetings.go internal/dialogue/greetings_test.go
git commit -m "feat(dialogue): once-per-instance greeting memory"
```

---

### Task 5: Export the idleness predicate

**Files:** Modify `internal/conversations/state.go`

- [ ] **Step 1: Add the exported wrapper**

Beside `isFullyIdle` (state.go:299):

```go
// IsFullyIdle reports whether a mob is unoccupied by the engine's standing
// definition — not in combat, not asleep, not mid-conversation, not walking a
// patrol. Exported for the arrival-greeting hook (5a), which must not invent
// a second idleness definition that would drift from this one.
func IsFullyIdle(m MobConversant) bool {
	return isFullyIdle(m)
}
```

- [ ] **Step 2: Build + package tests + commit**

Run: `go build ./... && go test ./internal/conversations/`
Expected: clean, PASS.

```bash
git add internal/conversations/state.go
git commit -m "feat(conversations): export IsFullyIdle for the greeting hook"
```

---

### Task 6: The arrival hook

**Files:** Modify `internal/usercommands/go.go`

- [ ] **Step 1: Insert the hook**

Immediately BEFORE the "Chunk 3.6: player-arrival conversation boost" block
(~go.go:697), so a player is welcomed before NPCs talk amongst themselves:

```go
			// 5a: NPC greetings. The first unoccupied NPC with an authored
			// greeting for its current mood welcomes the arriving player —
			// once per mob instance per player per boot, at most one greeting
			// per entry, and never for a hidden player: being hailed by name
			// would silently defeat stealth.
			if !user.Character.IsHidden() {
				for _, greeterInstId := range destRoom.GetMobs() {
					gMob := mobs.GetInstance(greeterInstId)
					if gMob == nil {
						continue
					}
					if dialogue.HasGreeted(greeterInstId, user.UserId) {
						continue
					}
					if !conversations.IsFullyIdle(conversationadapter.AdaptMob(gMob)) {
						continue
					}
					df := dialogue.Load(int(gMob.MobId), gMob.Zone)
					if df == nil || len(df.Greetings) == 0 {
						continue
					}
					text, ok := dialogue.PickGreeting(df.Greetings, dialogue.GetMood(greeterInstId, df.DefaultMood))
					if !ok {
						continue
					}
					dialogue.MarkGreeted(greeterInstId, user.UserId)
					gMob.Command(`say ` + text)
					break // at most one greeting per entry (9 two-greeter rooms measured; none higher)
				}
			}
```

Notes for the implementer:
- Verified: `go.go` already imports `conversations` and `mobs`;
  `conversationadapter` is used by the boost block so it is present too. Add
  `dialogue` — it is not currently imported.
- If `isSneaking` (go.go:123) is still in scope here, prefer it over calling
  `IsHidden()` again — read the function to see.
- Delivery via `gMob.Command("say ...")` routes through the mob command queue
  and the speech channel, so `deafen` and channel filtering apply — that is
  the point of not using raw room text.

- [ ] **Step 2: Build + full-package check**

Run: `go build ./... && go test ./internal/usercommands/ -count=1 2>&1 | tail -2`
Expected: clean.

- [ ] **Step 3: Manual smoke (in-game)**

Boot locally, walk a fresh character into Amber Valley's inn (mob 9397,
Hesper Vane — a known greeter with a friendly-mood line). Expect the greeting
ONCE on first entry, silence on re-entry, and the normal tree root on `talk`.

- [ ] **Step 4: Commit**

```bash
git add internal/usercommands/go.go
git commit -m "feat(dialogue): NPCs greet arriving players (5a)"
```

---

### Task 7: The one duplicate greeting

**Files:** One dialogue YAML (identify below)

- [ ] **Step 1: Identify it**

The 07-25 probe found exactly one greeting that duplicates its tree root.
Re-find it (temp test in `internal/dialogue/`, delete after):

```go
func TestTmpFindDuplicate(t *testing.T) {
	// walk ../../_datafiles/world/dogmud/dialogue/*/*.yaml, unmarshal into
	// DialogueFile (Greetings is now a real field), normalize whitespace+case,
	// and t.Logf any file where greeting[0] and Tree.Root.Text contain each
	// other. Expect exactly one hit.
}
```

- [ ] **Step 2: Rewrite the greeting**

Rewrite that file's greeting as a distinct *welcome* in the NPC's established
voice — first person, 80-char wrap, no semicolons (CLAUDE.md dialogue SOPs).
Do not touch the tree root: it is the live self-introduction.

- [ ] **Step 3: Verify + commit**

Run: `DOGMUD_BOOT_SMOKE=1 go test . -run TestSmoke_AllDialogueFilesParse -count=1 2>&1 | tail -3`
Expected: PASS (count still 186).

```bash
git add _datafiles/world/dogmud/dialogue/
git commit -m "content(dialogue): distinct welcome for the one root-duplicate greeting"
```

---

### Task 8: Spec refinement note

**Files:** Modify `docs/superpowers/specs/2026-07-25-npc-greetings-5a-design.md`

- [ ] **Step 1: Update the suppression table**

Replace the "mid-schedule-activity" row with the conversations predicate
(`IsFullyIdle`: combat / asleep / mid-conversation / patrol mid-walk) and add
a line recording WHY: the greeting prose is written to be delivered mid-work
("mind the shavings — pull up that stool"), so a crafting shopkeeper should
greet; schedule *sleep* remains suppressed via the Sleeping buff.

- [ ] **Step 2: Commit**

```bash
git add docs/superpowers/specs/2026-07-25-npc-greetings-5a-design.md
git commit -m "docs(spec): 5a suppression = conversations idleness predicate (recorded refinement)"
```

---

### Task 9: Verification gate + content playtest

- [ ] **Step 1: Full suite / fmt / vet**

Run: `go test ./... -count=1` then `gofmt -l internal modules && go vet ./...`
Expected: all clean.

- [ ] **Step 2: Boot under the prod setting**

`MapConsistencyEnforce: panic` (skip-worktree file — check content, not git
status). Boot; expect zero PANIC lines, `errors=0 warnings=0`.

- [ ] **Step 3: PATCH_NOTES.md**

Dated player-facing entry in the house voice. This one is genuinely
player-visible: 186 townsfolk begin welcoming arrivals. No identifiers, no
numbers.

- [ ] **Step 4: CONTENT ADVERSARIAL PLAYTEST — REQUIRED, not optional**

This changes what 186 NPCs do in front of players; boot-clean does not verify
an experience (CLAUDE.md gate). Write a goals file
(`tools/playtest/goals/npc-greetings.yaml`) directing the tester to: enter and
re-enter greeter rooms as a fresh character; visit at least one two-greeter
room (9 exist); `talk` immediately after being greeted and judge whether
greeting + root text read as coherent or repetitive; try entering while
sneaking; report every line that reads oddly out of context. Run
`/playtest local bug-finder` with that goals file, fix findings, re-run if
needed — only then hand to the user.

- [ ] **Step 5: Commit patch notes; hand the playtest report + feature to the user**

---

## Out of scope

Dialogue editor (5b — next), quest/behavior-tree editors, greeting prose
rewrites beyond the single duplicate, `ExpiryPeriod`-based re-greeting
(considered and rejected in the spec), and any persistence of the Greeted
flag.
