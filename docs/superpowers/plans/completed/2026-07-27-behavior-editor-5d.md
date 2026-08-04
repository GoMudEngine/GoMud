# Behavior-Tree Editor (5d) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** sixth `/build` tab — Behaviors — structured recursive editing of
archetypes / per-mob trees / room trees, archetype lifecycle with reference
guard, live cache-busting saves, plus the mob-rename behavior-file move.

**Architecture:** writer + validation + reference scan live in
`internal/behaviortree` (it imports mobs, so it can walk templates; the
mob-rename hook is a callback var on mobs, the `AttackRejectedTryMobBehavior`
seam pattern in reverse). `Build.Behavior.*` GMCP behind a `behaviorDeps`
seam; `behaviors.js` recursive collapsible editor.

**Verified contracts** (re-verify only if a step fails):
- `compileNode` (loader.go:60) IS the structural validator — unknown
  type/check/do/mod, missing children/child/check/do all error with a node
  path. Save-time validation = `LoadTreeFromBytes` on the writer's own
  marshal output + the new event-vocabulary check.
- One authoring type serves all three families: `TreeDef{Tree NodeDef,
  GoalWeights, DefaultGoals}` (types.go:82); tree files just leave the
  extras empty.
- `NodeDef.Params` is `yaml:",inline"` — unknown node keys land there and
  round-trip. encoding/json has NO inline: on the wire Params travels as an
  explicit `"params"` object; the server re-inlines on yaml marshal. The
  client emits `{type, event, check, do, mod, note, child, children,
  params:{}}`.
- Path helpers exported: `GetBehaviorPath(mobId, zone, name)`,
  `GetRoomBehaviorPath(roomId, zone)`, `GetArchetypePath(name)`
  (helpers.go:51/74/86).
- Engine caches: `LoadTree/LoadRoomTree/LoadArchetype` replace the positive
  entry AND clear the negative (engine.go) — so SAVE eviction = reload from
  the just-written file. DELETE needs new `Evict*` methods (remove positive
  + goal maps, set the negative).
- Archetype shift tables: `shiftEligibleFrom` / `shiftTargetWhitelist` (+
  the mutation pull table) in archetype_shift.go — archetype-name references
  the delete guard must check.
- `mobs` cannot import `behaviortree` (behaviortree imports mobs) — hence
  the callback-var seam for the rename move.
- yaml.v2 marshals map keys sorted → the inline-params fixed point holds.
- Windows: subagents shell-denied; run commands from the main loop. Branch
  `feature/behavior-editor-5d` off master.

---

### Task 1: Types groundwork + round-trip proof (TDD)

**Files:**
- Modify: `internal/behaviortree/types.go`, `internal/behaviortree/loader.go`
- Create: `internal/behaviortree/roundtrip_test.go`

- [ ] **Step 1: Failing round-trip test** — the 5b/5c fixed-point pattern
  over EVERY live behavior file (glob `behaviors/**/*.yaml` under the
  configured world; expect ≥26 archetypes + per-mob + room trees; repo-root
  chdir helper à la `internal/quests/unification_equivalence_test.go`):
  parse `TreeDef` → marshal → reparse → re-marshal byte-identical; node
  count survives (recursive counter); the multiset of check/do/event values
  survives. Also a unit case proving a `note:` key on a node round-trips
  and does NOT leak into `cleanParams` output.
- [ ] **Step 2: RED** (note leaks into params today).
- [ ] **Step 3:** `NodeDef` gains `Note string
  \`yaml:"note,omitempty" json:"note,omitempty"\``; `TreeDef` gains
  `Notes string \`yaml:"notes,omitempty" json:"notes,omitempty"\``; add
  `"note"` to `knownFields` (loader.go:12) so it never reaches runtime
  params. Mirror json tags onto every yaml-tagged field of `NodeDef` /
  `TreeDef` / `GoalDefault` (`Params` gets `json:"params,omitempty"` — the
  wire shape; regex-assisted like 5c's 112 tags).
- [ ] **Step 4: GREEN** + full behaviortree suite. **Step 5: Commit**
  `feat(btree): note/notes fields + json wire tags + marshal fixed-point proof`

### Task 2: Event vocabulary, pinned (TDD)

**Files:**
- Create: `internal/behaviortree/events.go`, `internal/behaviortree/events_test.go`

- [ ] **Step 1:** Assemble the authoritative list:
  `grep -rn 'EventType:\s*"' internal/ modules/ --include="*.go" | grep -v _test`
  plus the event strings observed in live YAML (mob_combat_round, mob_hurt,
  mob_idle, mob_die, packmate_hurt, heard_callforhelp, player_enter,
  player_give, player_attack_rejected, ...grep is authoritative, this list
  is not).
- [ ] **Step 2:** `var KnownBehaviorEvents = map[string]bool{...}` +
  `func EventNames() []string` (sorted). Anti-drift test: walks the repo's
  Go source for `EventType: "..."` literals (chdir helper) and asserts
  every fired event is in the map AND every map entry is fired somewhere —
  both directions, so the vocabulary can't rot (the AddPeriod lesson).
  Second test: every `event:` value in live behavior YAML is in the map.
- [ ] **Step 3: RED → GREEN → Commit**
  `feat(btree): pinned behavior event vocabulary + anti-drift gate`

### Task 3: Writer + cache contract (TDD)

**Files:**
- Create: `internal/behaviortree/save.go`, `internal/behaviortree/save_test.go`
- Modify: `internal/behaviortree/engine.go` (Evict methods)

- [ ] **Step 1: Failing tests** (CONFIG_PATH override helper, the
  `internal/dialogue/save_test.go` pattern):

```go
TestSaveArchetype_WritesValidatesAndReloadsCache  // file exists; engine.GetArchetype* fresh; negative cleared
TestSaveArchetype_RefusesUnknownDo                // loader error surfaces verbatim; nothing written
TestSaveArchetype_RefusesUnknownEvent             // the NEW vocabulary check
TestDeleteArchetype_EvictsAndSetsNegative         // file gone; positive gone; noArchetype[name] true; goal maps cleared
TestSaveMobTree_PathUsesLiveMobName               // GetBehaviorPath(mobId, zone, name)
TestCreateMobTree_SeedsFromArchetype              // tree deep-equal (by re-marshal) to the archetype's
TestSaveRoomTree_RoundTripsAndReloads
TestRawFileHasHandComments                        // helper: true for '#' comment lines, false for clean marshal output
```

- [ ] **Step 2: RED.** **Step 3: Implement.** Validation order in every
  save: marshal → `LoadTreeFromBytes(out)` (structural + registry) → walk
  NodeDefs for `Event != "" && !KnownBehaviorEvents[event]` (refusal) →
  write → reload into engine (`LoadArchetype`/`LoadTree`/`LoadRoomTree` —
  they already clear negatives). Warnings returned alongside: empty
  composite, an event used by no other live tree. New engine methods:

```go
func (e *Engine) EvictArchetype(name string) {
	e.mu.Lock()
	delete(e.archetypes, name)
	delete(e.archetypeGoalWeights, name)
	delete(e.archetypeDefaultGoals, name)
	e.noArchetype[name] = true
	e.mu.Unlock()
}
// EvictTree(mobId) / EvictRoomTree(roomId): delete positive, set negative.
```

  `CreateArchetype(name)` refuses an existing file; skeleton = one selector
  with a single `note:`-bearing example condition/action pair that compiles.
  `DeleteMobTree(mobId)` / `DeleteRoomTree(roomId)`: remove + Evict (the
  negative cache is CORRECT after delete — that's the fallback semantics).
- [ ] **Step 4: GREEN + commit** `feat(btree): writer with cache+negative-cache contract`

### Task 4: Mob-rename moves the behavior file (TDD)

**Files:**
- Modify: `internal/mobs/save.go` (+ test), `internal/behaviortree/engine.go` init (hook assignment)

- [ ] **Step 1: Failing test** in `internal/mobs/save_test.go`: set a fake
  `OnMobFileRename` recorder, save a rename via `SaveMobSpec`, assert the
  hook fired with (mobId, zone, oldName, newName). Separate behaviortree
  test: the real hook moves `behaviors/<zone>/<id>-<old>.yaml` →
  `<id>-<new>.yaml` and re-loads the tree cache.
- [ ] **Step 2:** `mobs` declares
  `var OnMobFileRename func(mobId int, zone, oldName, newName string)`;
  `SaveMobSpec`'s rename branch calls it (nil-safe) AFTER the mob file move
  succeeds. `behaviortree` init assigns the mover (silently no-op when no
  behavior file exists). Same hook fires on ZONE change too (path embeds
  zone).
- [ ] **Step 3: GREEN + commit** `fix(btree): mob rename/re-zone moves the behavior file (was silently orphaning it)`

### Task 5: Archetype reference guard (TDD)

**Files:**
- Create: `internal/behaviortree/references.go` (+ test)

- [ ] `ArchetypeReferences(name string) []string` — verbatim strings from:
  every mob template with `BehaviorArchetype == name`
  (`mobs.AllMobTemplates()`), membership in `shiftEligibleFrom` /
  `shiftTargetWhitelist` / the mutation pull table ("the shift system names
  this archetype — removing it strands shifted mobs"), and
  `validateAutoAggroBehaviorGates`-style auto-aggro mobs whose only gate is
  this archetype. Tests fake nothing — seed a template via the mobs test
  helpers. RED → GREEN → commit.

### Task 6: Registry exports + GMCP handlers (TDD)

**Files:**
- Modify: `internal/behaviortree/conditions.go` / `actions.go`
  (`ConditionNames()` / `ActionNames()`, sorted)
- Create: `modules/gmcp/gmcp.Behavior.go` (+ `_test.go`)
- Modify: `modules/gmcp/gmcp.go` (routing), `modules/gmcp/gmcp.Build.go` (dispatch)

- [ ] `behaviorDeps` seam mirroring `questDeps`. Verbs:
  `Build.Behavior.List` → `Build.Behaviors` payload
  `{archetypes:[{name, usedBy, hasHandComments}], mobTrees:[{mobId, mobName,
  zone}], roomTrees:[{roomId, zone, title}]}`;
  `Build.Behavior.Get {kind, name|mobId|roomId}` → `Build.Behavior`
  `{kind, key, file TreeDef, hasHandComments, usedBy, enums}` — enums:
  node types, decorator mods (with their param key: rounds/times/percent),
  `ConditionNames()`, `ActionNames()`, `EventNames()`, archetype names,
  goal-type names; `Update`, `Create {kind, name|mobId+fromArchetype|roomId}`,
  `Delete` (archetype delete refuses on `ArchetypeReferences`, listed
  verbatim in `BehaviorRefs`). BuildResult gains `BehaviorRefs []string`.
  Fake-deps tests per handler: refuse-invalid-saves-nothing, warning-only
  saves, guard blocks, create-from-archetype seeds. RED → GREEN → wire
  routing/dispatch (the `Build.Quest.*` block is the template) → commit.

### Task 7: Client — `behaviors.js` + wiring

**Files:**
- Create: `_datafiles/html/public/static/js/behaviors.js`
- Modify: `_datafiles/html/public/build.html`, `_datafiles/html/public/static/js/mobs.js`

- [ ] **Step 1: The recursive node editor.** One function, `nodeRow(box,
  def, depth)`, built on the 5b/5c collapsible shell: summary
  `type · event? · check/do/mod`; body = type select (reshapes the row's
  fields on change), event select ((none) + vocabulary), check/do datalist
  picker (shown per type), decorator mod select + params (rounds / times /
  percent shown per mod) + single child slot, `note` text field, generic
  params key/value rows, and for selector/sequence an ORDERED children list
  (`↑`/`↓`, add-child type picker, "children run in ORDER" rule line).
  Gather mirrors the wire shape (`params` object; empty fields omitted).
  Guard depth with a soft cap (~12) that warns rather than refuses.
- [ ] **Step 2: Lists + detail.** Three collapsible sections with search;
  archetype detail = notes + goal-weights rows + default-goals rows + tree
  root + used-by list (each entry a jump: switch to mobs mode +
  `Build.Mob.Get`); mob-tree detail shows "overrides archetype <x>" when
  the mob has one; room-tree detail names the room. Create flows per spec.
  Comment-loss banner when `hasHandComments`. Save/Delete via the shared
  far-edge layout; refusals verbatim into `#bt-errors`, warnings amber.
- [ ] **Step 3: mobs.js jump** — beside the behavior_archetype select, an
  "Edit tree…" button → switch to behaviors mode + `Build.Behavior.Get
  {kind:"archetype", name}` (and a "Mob tree…" button when a per-mob tree
  exists — `Build.Behavior.List` data cached on `window.Builder`).
- [ ] **Step 4: build.html** — sixth tab, `behaviorlist` sidebar div (reuse
  the shared list CSS selectors), routing block, mode entry fires
  `Build.Behavior.List`, script tag. `node --check` + inline-script parse
  check. Commit
  `feat(build): Behaviors tab — recursive tree editor (5d UI)`

### Task 8: Verification gate

- [ ] Full suite; gofmt/vet; drift gate (`DOGMUD_BOOT_SMOKE=1`, 11/11
  new: 0); panic-mode boot after instance wipe.
- [ ] Headless E2E (extend the quest_e2e plumbing): list → get
  `generic_fighter` → save unchanged → ok (+ hasHandComments true) →
  refusals (unknown `do`, bogus `event`, childless selector) → create
  archetype → delete it clean → delete `generic_fighter` REFUSED with
  references. Temp-admin dance as before, reverted.
- [ ] PATCH_NOTES staff-voice entry; memory update (epic file + MEMORY.md).
- [ ] **User browser gate:** edit an archetype node → watch a live mob
  change behavior without reboot; create a per-mob tree from an archetype →
  verify override; delete it → verify fallback; trip the refusals; see the
  comment-loss banner on generic_fighter; rename a mob with a per-mob tree
  → file follows. First-save comment loss on hand-authored files is
  EXPECTED — say so in the handoff. Merge on the user's word.

---

## Self-review notes

- Save-eviction uses the engine's own Load* (already clears negatives);
  only DELETE needs the new Evict* methods. The cache-contract tests pin
  both directions.
- The seed-from-archetype copy is done by re-marshal/re-parse of TreeDef —
  never by sharing NodeDef slices (the shallow-copy gotcha).
- `usedBy` on the List payload is computed per request from
  `mobs.AllMobTemplates()` — no cache to go stale.
- The event anti-drift test asserting BOTH directions means adding a new
  engine event without updating the vocabulary fails CI, and so does a
  vocabulary entry nothing fires.
