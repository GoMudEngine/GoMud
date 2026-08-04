# Quest Definition Unification (5c-pre) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** one struct, one parse for quest YAML — `internal/quests` owns the full
explicitly-tagged definition; `internal/questengine` consumes it and stops
touching files — proven behavior-identical by an equivalence harness over all
79 live files before anything old is deleted.

**Architecture:** move the trigger type family down into `internal/quests`
(questengine already imports quests — no cycle), leave type ALIASES behind in
questengine so its ~35 importers and its evaluation tests compile unchanged,
replace the engine's load-time struct mutation (`questId`/`trigId`) with an
engine-side index wrapper, and rewrite `questengine.LoadDataFiles` to build
its index from `quests.GetAllQuests()`.

**Tech stack:** Go, yaml.v2 via `internal/fileloader` (which calls
`Validate()` on every loaded file — `fileloader.go:106` — so validation moved
into `(*Quest).Validate()` enforces at boot with no extra wiring).

**Verified contracts the plan relies on** (re-verify only if a step fails):
- `quests.GetAllQuests() []Quest` returns value copies (`quests.go:204`) —
  safe to hold read-only; slices share backing arrays, and nothing mutates
  defs after load once the `questId`/`trigId` mutation is removed.
- Boot order `main.go:1740` `quests.LoadDataFiles()` →
  `main.go:1746` `questengine.LoadDataFiles()` — already consumption-ready.
- Engine internals live in `internal/questengine/engine.go` (trigger index +
  `evaluate`/`executeActions` use `t.trigId` at lines 94/103/104/179/193).
- Drift gate: `boot_smoke_test.go`, env `DOGMUD_BOOT_SMOKE=1`, baseline map
  `knownSilentlyIgnoredKeys` (~28 entries; live run says 26 distinct, new: 0).
- Windows note: run everything from the main loop (subagents are
  shell-denied); repo-root chdir + `configs.ReloadConfig()` for any test that
  touches the live data tree (precedent: `internal/dialogue/save_test.go`'s
  `overrideDataFilesDir`, `internal/web/auth_test.go`).

---

### Task 1: Equivalence harness (RED)

**Files:**
- Create: `internal/quests/unification_equivalence_test.go`

The harness carries LOCAL copies of both old struct shapes (tag-less
`oldQuest` mirroring today's `quests.Quest` binding; snake_case-tagged
`oldQuestDef` mirroring `questengine.QuestDef`), parses every live quest file
three ways, and asserts the NEW unified `quests.Quest` reproduces both. Local
copies make the harness permanent — it survives the deletion of the old
production structs.

- [ ] **Step 1: Write the harness.** Local old-shape structs are verbatim
  copies of the CURRENT `internal/quests/quests.go:25-74` structs (keep them
  tag-less — that IS the old binding) and the CURRENT
  `internal/questengine/types.go` `QuestDef` tree (keep snake_case tags;
  drop only the unexported `questId`/`trigId`). Body:

```go
package quests

// Verbatim old-shape locals (abridged here — copy the real structs):
//   oldQuest / oldQuestReward / oldQuestStep   from quests.go pre-change
//   oldQuestDef / oldTriggerDef / oldConditions / oldActionDef (+ params)
//   from questengine/types.go pre-change

func TestUnification_EquivalentToBothOldParses(t *testing.T) {
	chdirRepoRootForTest(t) // same pattern as dialogue save_test helper

	dir := configs.GetFilePathsConfig().DataFiles.String() + `/quests`
	files, err := filepath.Glob(dir + `/*.yaml`)
	if err != nil || len(files) == 0 {
		t.Fatalf("no quest files found under %s: %v", dir, err)
	}

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		var oq oldQuest
		var od oldQuestDef
		var nq Quest
		if err := yaml.Unmarshal(data, &oq); err != nil {
			t.Fatalf("%s: old quests parse: %v", f, err)
		}
		if err := yaml.Unmarshal(data, &od); err != nil {
			t.Fatalf("%s: old questengine parse: %v", f, err)
		}
		if err := yaml.Unmarshal(data, &nq); err != nil {
			t.Fatalf("%s: new unified parse: %v", f, err)
		}

		// Everything the old quests parse populated (the REWARD-PAYING copy):
		if nq.QuestId != oq.QuestId || nq.Name != oq.Name ||
			nq.Secret != oq.Secret || nq.Repeatable != oq.Repeatable ||
			nq.CooldownRounds != oq.CooldownRounds {
			t.Errorf("%s: identity mismatch vs old quests parse", f)
		}
		if !rewardsEqualOld(nq.Rewards, oq.Rewards) {
			t.Errorf("%s: REWARDS mismatch vs old quests parse:\nnew=%+v\nold=%+v", f, nq.Rewards, oq.Rewards)
		}
		// Everything the old questengine parse populated:
		if len(nq.Steps) != len(od.Steps) || len(nq.Triggers) != len(od.Triggers) {
			t.Fatalf("%s: step/trigger count mismatch vs old questengine parse", f)
		}
		for i := range od.Steps {
			if nq.Steps[i].Id != od.Steps[i].Id || nq.Steps[i].MapTarget != od.Steps[i].MapTarget ||
				nq.Steps[i].Hint != od.Steps[i].Hint || nq.Steps[i].Description != od.Steps[i].Description {
				t.Errorf("%s: step %d mismatch", f, i)
			}
		}
		for i := range od.Triggers {
			if !triggersEqualOld(nq.Triggers[i], od.Triggers[i]) {
				t.Errorf("%s: trigger %d mismatch", f, i)
			}
		}
		if len(nq.Flags) != len(od.Flags) {
			t.Errorf("%s: flag count mismatch", f)
		}
	}
}
```

  `rewardsEqualOld` / `triggersEqualOld` compare field-by-field (the trigger
  comparison marshals both sides back to yaml and compares strings — cheap
  deep-equal that ignores the old/new type difference). Also assert total
  file count `>= 79` so a glob mistake can't vacuously pass.

- [ ] **Step 2: Run it — expect RED as a COMPILE failure** (`nq.Triggers`,
  `Steps[i].MapTarget` don't exist on `quests.Quest` yet):
  `go test ./internal/quests/ -run Unification -count=1` → `undefined` /
  unknown-field compile errors. If it compiles, the harness is wrong — stop.

- [ ] **Step 3: Commit** `test(quests): equivalence harness for the 5c-pre unification (RED)`
  — commit the failing-to-compile test on the feature branch only if the repo
  convention allows broken intermediate states; otherwise hold the commit and
  fold it into Task 2's commit. (Working on branch `feature/quest-unification-5c-pre`,
  created from master at task start.)

### Task 2: The unified struct + aliases + engine index (GREEN)

**Files:**
- Modify: `internal/quests/quests.go` (struct tags + new fields)
- Create: `internal/quests/triggers.go` (the moved type family)
- Modify: `internal/questengine/types.go` (becomes aliases + engine-local types)
- Modify: `internal/questengine/engine.go` (index wrapper)
- Modify: `internal/questengine/loader.go` (no more file I/O)
- Modify: `internal/questengine/engine_test.go` lines 82 and 410 (delete the
  two `Linear: false,` literals — the field dies)

- [ ] **Step 1: Explicit tags + new fields on `quests.Quest`.** Replace the
  struct block in `quests.go` (current lines 25-74):

```go
type QuestFlagDef struct {
	Key         string   `yaml:"key"`
	Values      []string `yaml:"values"`
	Description string   `yaml:"description,omitempty"`
}

// QuestReward — every key is EXPLICITLY tagged with the key that has always
// bound it. The no-underscore keys are historical yaml.v2 tag-less binding
// (lowercased field name, no underscore handling), now pinned by tag; the
// snake_case five were tagged all along. This vocabulary is canonical — the
// 5c editor writer marshals exactly this.
type QuestReward struct {
	QuestId       string `yaml:"questid,omitempty"`
	Gold          int    `yaml:"gold,omitempty"`
	ItemId        int    `yaml:"itemid,omitempty"`
	BuffId        int    `yaml:"buffid,omitempty"`
	SkillInfo     string `yaml:"skillinfo,omitempty"`
	StatInfo      string `yaml:"stat_info,omitempty"`
	RecipeInfo    string `yaml:"recipe_info,omitempty"`
	ItemInfo      string `yaml:"item_info,omitempty"`
	SpellId       string `yaml:"spellid,omitempty"`
	PlayerMessage string `yaml:"playermessage,omitempty"`
	RoomMessage   string `yaml:"roommessage,omitempty"`
	RoomId        int    `yaml:"roomid,omitempty"`
	RepFaction    string `yaml:"rep_faction,omitempty"`
	RepAmount     int    `yaml:"rep_amount,omitempty"`
}

type Quest struct {
	QuestId        int            `yaml:"questid"`
	Name           string         `yaml:"name"`
	Description    string         `yaml:"description,omitempty"`
	Secret         bool           `yaml:"secret,omitempty"`
	Steps          []QuestStep    `yaml:"steps"`
	Rewards        QuestReward    `yaml:"rewards,omitempty"`
	Triggers       []TriggerDef   `yaml:"triggers,omitempty"`
	Flags          []QuestFlagDef `yaml:"flags,omitempty"`
	Repeatable     bool           `yaml:"repeatable,omitempty"`
	CooldownRounds int            `yaml:"cooldown_rounds,omitempty"`
}

type QuestStep struct {
	Id          string `yaml:"id"`
	Description string `yaml:"description,omitempty"`
	Hint        string `yaml:"hint,omitempty"`
	MapTarget   int    `yaml:"map_target,omitempty"`
}
```

  Note what is deliberately ABSENT: `linear` (dead — no reader),
  `chain_quest` (dead), and any snake_case reward alias.

- [ ] **Step 2: Move the trigger family.** Create
  `internal/quests/triggers.go`: cut `TriggerDef`, `Conditions`, `ActionDef`,
  `BumpRepDef`, `DeclareBountyDef`, `QuestFlagAction`, `NpcSayDef`,
  `SayLineDef`, `SpawnDef`, `ExitLock`, `SkillDef`, `StatDef`, `RecipeDef`,
  `SequenceDef` from `internal/questengine/types.go` verbatim, changing only:
  package clause → `package quests`, and DELETE the two unexported fields on
  `TriggerDef` (`questId int` / `trigId string`) and their comment — the
  engine wrapper replaces them. `EventDetails` and `NotifyResult` do NOT move
  (engine-facing, not definition data).

- [ ] **Step 3: Aliases in questengine.** `types.go` shrinks to:

```go
package questengine

import "github.com/GoMudEngine/GoMud/internal/quests"

// The definition types live in internal/quests (single owner of the quest
// file parse — 5c-pre unification). These aliases keep questengine's API and
// its evaluation tests source-compatible.
type (
	QuestFlagDef  = quests.QuestFlagDef
	QuestDef      = quests.Quest
	QuestStep     = quests.QuestStep
	QuestRewards  = quests.QuestReward
	TriggerDef    = quests.TriggerDef
	Conditions    = quests.Conditions
	ActionDef     = quests.ActionDef
	BumpRepDef    = quests.BumpRepDef
	DeclareBountyDef = quests.DeclareBountyDef
	QuestFlagAction  = quests.QuestFlagAction
	NpcSayDef     = quests.NpcSayDef
	SayLineDef    = quests.SayLineDef
	SpawnDef      = quests.SpawnDef
	ExitLock      = quests.ExitLock
	SkillDef      = quests.SkillDef
	StatDef       = quests.StatDef
	RecipeDef     = quests.RecipeDef
	SequenceDef   = quests.SequenceDef
)

// EventDetails / NotifyResult stay here — they describe engine invocations,
// not quest definitions. (moved from the old types.go bottom, unchanged)
```

- [ ] **Step 4: Engine index wrapper.** In `engine.go`:

```go
// indexedTrigger pairs a definition trigger with the engine-assigned identity
// the evaluator needs (visit tracking, tracing). Replaces the old pattern of
// mutating unexported fields ON the definition struct at register time —
// definitions are now immutable shared data owned by internal/quests.
type indexedTrigger struct {
	def     *TriggerDef
	questId int
	trigId  string
}

type Engine struct {
	quests       map[int]*QuestDef
	triggerIndex map[string][]*indexedTrigger
}

func (e *Engine) RegisterQuest(q *QuestDef) {
	e.quests[q.QuestId] = q
	for i := range q.Triggers {
		e.triggerIndex[q.Triggers[i].Event] = append(e.triggerIndex[q.Triggers[i].Event],
			&indexedTrigger{def: &q.Triggers[i], questId: q.QuestId, trigId: fmt.Sprintf("q%d-t%d", q.QuestId, i)})
	}
}
```

  Then mechanical fixes in `evaluate`/`executeActions`/`matchTriggerFields`:
  `t.def.Conditions`, `t.def.Actions`, `matchTriggerFields(t.def, details)`,
  `t.trigId` stays (now on the wrapper). `executeActions` takes
  `*indexedTrigger`. `NewEngine` allocates the new map type. The compiler
  enumerates every remaining reference — fix until `go build ./...` is clean
  (the dead-code-sweep rule: let the compiler find them, don't grep).

- [ ] **Step 5: Loader stops reading files.** Replace
  `questengine.LoadDataFiles` (loader.go:121-144) body:

```go
func LoadDataFiles() {
	start := time.Now()

	globalEngine = NewEngine()
	all := quests.GetAllQuests()
	for i := range all {
		globalEngine.RegisterQuest(&all[i])
	}

	ValidateAllFlags()

	mudlog.Info("questengine.LoadDataFiles()", "loadedCount", len(globalEngine.quests), "Time Taken", time.Since(start))
}
```

  Delete the `RegisterFlags` loop (quests.LoadDataFiles already registers
  the same `q.Flags`) and the now-unused fileloader/errors imports.
  `validateQuestDef` moves in Task 3 — for THIS step, leave it in place but
  unreferenced is not allowed by the compiler, so move its body now if the
  compiler demands, or keep a temporary call from nothing (prefer: do Task 3
  immediately after; the two tasks land as consecutive commits either way).
  `&all[i]` takes stable addresses into one backing array — do NOT use
  `for _, q := range` + `&q`.

- [ ] **Step 6: Delete the two `Linear: false,` test literals**
  (engine_test.go:82, engine_test.go:410).

- [ ] **Step 7: Compile + run the harness and both packages' suites:**
  `go build ./... && go test ./internal/quests/ ./internal/questengine/ -count=1`
  Expected: harness GREEN over every live file; questengine evaluation tests
  pass with no assertion changes (aliases guarantee source compatibility).

- [ ] **Step 8: Commit** `refactor(quests): single parse — definitions unified into internal/quests (5c-pre)`

### Task 3: Validation moves to `(*Quest).Validate()`

**Files:**
- Modify: `internal/quests/quests.go` (`Validate()` currently `return nil`, line 80)
- Modify: `internal/questengine/loader.go` (delete `validateQuestDef`)
- Create: `internal/quests/validate_test.go`
- Modify: `internal/questengine/loader_test.go` (its 12 `QuestDef{}` validation
  cases move to the new quests test file, s/QuestDef/Quest/ — assertions unchanged)

- [ ] **Step 1: Write the failing tests.** Port every `validateQuestDef` case
  from `loader_test.go` into `internal/quests/validate_test.go` against
  `(q *Quest).Validate()`: empty id, empty name, no steps, invalid event,
  trigger with no actions, duplicate step ids, empty step id, empty flag
  key / no flag values / duplicate flag key, own-quest grant of unknown step,
  plus the valid-quest happy path. One test per rule, table style matching the
  existing file.
- [ ] **Step 2: RED:** `go test ./internal/quests/ -run Validate -count=1`
  fails (Validate still returns nil).
- [ ] **Step 3: Move the body.** `validateQuestDef(q *QuestDef) error`
  (questengine/loader.go:25-107) becomes `(q *Quest) Validate() error` in
  quests.go verbatim (it references only fields that moved with the struct).
  Delete the original and its callers (none remain after Task 2's loader
  rewrite). fileloader now enforces it on every parse (fileloader.go:106).
- [ ] **Step 4: GREEN + full packages:**
  `go test ./internal/quests/ ./internal/questengine/ -count=1`
- [ ] **Step 5: Commit** `refactor(quests): boot validation lives on (*Quest).Validate, enforced by fileloader`

### Task 4: Marshal fixed-point guard (the 5c writer's safety net)

**Files:**
- Create: `internal/quests/roundtrip_test.go`

- [ ] **Step 1: Write the test** (5b pattern —
  `internal/dialogue/save_test.go`'s `TestWriter_RoundTripsEveryLiveFile` is
  the model): for every live quest file, `parse → yaml.Marshal → parse →
  yaml.Marshal` and assert the two marshals are byte-identical ("would churn
  on every save" failure message), plus count guards
  (steps/triggers/actions/flags survive) and rewards-nonzero preservation
  (any reward field non-zero in parse #1 is non-zero in parse #2).
- [ ] **Step 2: Run it.** Marshal fixed-point should hold immediately (we
  only parse and re-marshal our own output — authored files are NOT required
  to match the canonical form, only to survive it). If a file fails, diagnose
  before proceeding: it means a tag mistake in Task 2, not acceptable churn.
- [ ] **Step 3: Commit** `test(quests): marshal fixed-point round-trip over all live quest files`

### Task 5: Content sweep — dead `linear:` lines + step-trigger probe

**Files:**
- Modify: `_datafiles/world/dogmud/quests/21-newcomers_path.yaml` (line 5),
  `28-waking_to_gaius.yaml` (line 8), `29-two_roads.yaml` (line 28),
  `30-the_awakening.yaml` (line 19)

- [ ] **Step 1: Verify placement, then strip.** For each of the four files,
  confirm the `linear:` line is top-level (29-two_roads.yaml's is at line 28
  — eyeball that it isn't nested inside a step before deleting). Delete the
  line + any attached comment. No other edits — these files must not be
  canonicalized by hand.
- [ ] **Step 2: Settle the `triggers|*.QuestStep` baseline entries.** Run the
  drift gate and print the recorded example paths for those two keys (the
  probe stores up to 3 paths per key — temporarily `t.Logf` the `found` map
  for them). If a live file authors `triggers:` inside a step, that content
  is read by NOTHING — move it to top-level `triggers:` (content bugfix,
  separate mini-commit naming the quest) — the earlier repo grep found no
  such file, so the expected outcome is: entries are stale, clear them in
  Task 6.
- [ ] **Step 3: Boot-check** (`go build -o` scratch + run past
  `quests.LoadDataFiles`/`questengine.LoadDataFiles` log lines, then kill).
- [ ] **Step 4: Commit** `chore(quests): strip dead linear: lines (no reader; 5c-pre)`

### Task 6: Drift-gate baseline shrink

**Files:**
- Modify: `boot_smoke_test.go` (`knownSilentlyIgnoredKeys`)

- [ ] **Step 1: Delete the 17 quest-artifact entries** (every
  `*|questengine.QuestRewards`, `*|questengine.QuestDef`,
  `*|questengine.QuestStep`, plus `triggers|quests.Quest`,
  `triggers|quests.QuestStep`, `linear|quests.Quest`,
  `map_target|quests.QuestStep`). Update the file's header comment: the
  dual-parse paragraph becomes past tense with a pointer to the 5c-pre spec.
- [ ] **Step 2: Run the gate:**
  `DOGMUD_BOOT_SMOKE=1 go test -run TestSmoke_NoNewSilentlyIgnoredYAMLKeys -count=1 -v .`
  Expected: PASS, `new: 0`, distinct count ~9-11. Any novel key here is a tag
  mistake from Task 2 — fix the tag, never re-baseline it.
- [ ] **Step 3: Commit** `test(smoke): drift-gate baseline shrinks 28→11 — quest dual-parse artifacts retired`

### Task 7: Full gates + bookkeeping

- [ ] **Step 1:** `go test ./... -count=1` (full suite),
  `gofmt -l internal/quests internal/questengine` (empty),
  `go vet ./internal/quests/... ./internal/questengine/...`
- [ ] **Step 2: Panic-mode boot** after the instance-save wipe
  (`rm -rf _datafiles/world/dogmud/{mobs,rooms}.instances/*` — never
  shops/guilds/moderation), watch for both quest load lines + workers
  started + zero panics.
- [ ] **Step 3: PATCH_NOTES.md** — short staff-voice entry (engine
  housekeeping: one source of truth for quest files; no player-visible
  change).
- [ ] **Step 4: Merge** to master per the session pattern (--no-ff, message
  file — `git merge -F -` does NOT read stdin), delete the feature branch,
  update memory (epic topic file + MEMORY.md unpushed count).

---

## Self-review notes

- Task 2 Step 5's ordering wart (validateQuestDef unreferenced between Tasks
  2 and 3) is called out in-step: Go tolerates an unreferenced package-level
  func, so Task 2 compiles with `validateQuestDef` still present; Task 3
  deletes it. No broken intermediate commit.
- The alias `QuestRewards = quests.QuestReward` intentionally maps the old
  engine name onto the surviving struct so any stray reference compiles
  against the REAL rewards rather than failing obscurely; the compiler sweep
  in Task 2 confirms whether any such reference exists at all (expected:
  none outside tests).
- `1000000-generic_quest.yaml` flows through the harness like every other
  file; nothing special-cases it in 5c-pre.
