# Quest Editor (5c) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** the fifth `/build` mode — full-parity quest authoring (identity,
steps, rewards, flags, complete trigger DSL) with create/delete behind a
reference guard, live without reboot.

**Architecture:** writer + registry-validation in `internal/quests` (the
unified parse from 5c-pre is the only data model); `Build.Quest.*` GMCP
behind a `questDeps` seam; engine re-index via one cheap
`questengine.LoadDataFiles()` call from the gmcp layer after each successful
mutation; `quests.js` fifth tab mirroring the dialogue editor's collapsible
patterns.

**Verified contracts** (re-verify only if a step fails):
- `quests` package: cache is package var `quests map[int]*Quest`
  (`quests.go:254`), `RegisterFlags` overwrites per `"%d-%s"` key,
  `GetFlagRegistry()` exported, `Quest.Filename()` =
  `"%d-%s.yaml"` via `ConvertForFilename(Name)`, `(*Quest).Validate()`
  enforces structural rules on every parse. No package mutex — all mutation
  happens on MainWorker (same model the mob writer relies on).
- Enum sources: `buffs.GetAllBuffIds()` + `GetBuffSpec(id).Name`;
  `spells.GetAllSpells() map[string]*SpellData` (`.Name`);
  `crafting.GetAll() map[string]*RecipeSpec` (key = recipe id);
  `factions.AllDefinitions()`; skill tags = the `SkillTag` constants block
  in `internal/skills/skills.go:29+` (verify the full set at
  implementation); stats are the six fixed names; quest tokens/flags mirror
  `collectDialogueEnums` (gmcp.Dialogue.go:136).
- Client reuses existing verbs for pickers: `Build.Mob.List` (mob pickers),
  `Build.Room.List` per zone (map_target picker — the mob test-spawn
  pattern), `window.Builder.itemRows` (items, stashed at login).
- Achievements never name quest tokens (only `quests_completed` counters) —
  excluded from the delete guard.
- Windows: subagents are shell-denied; run all commands from the main loop.
  Branch: `feature/quest-editor-5c` off master.

---

### Task 1: Writer — `internal/quests/save.go` (TDD)

**Files:**
- Create: `internal/quests/save.go`, `internal/quests/save_test.go`

- [ ] **Step 1: Failing tests.** `save_test.go` needs a temp-dir seam. The
  quests loader reads `configs.GetFilePathsConfig().DataFiles`; mirror the
  mobs pattern (`internal/mobs/save_test.go:25`) with a package var:
  in `save.go` define `var questsDataRoot = func() string { return configs.GetFilePathsConfig().DataFiles.String() + "/quests" }`
  and use it everywhere; tests override it. Tests (each seeds the cache
  directly like `seedMob`, cleans up via `t.Cleanup`):

```go
func TestSaveQuest_WritesSwapsCacheAndFlags(t *testing.T)
// valid quest with a flag decl → file exists at dir/<id>-<name>.yaml,
// GetQuest(id) returns the new value, GetFlagRegistry() has "<id>-<key>".

func TestSaveQuest_RenameMovesFile(t *testing.T)
// save under name A, save same id under name B → old file gone, new exists.

func TestSaveQuest_RefusesInvalid(t *testing.T)
// quest with duplicate step ids → error, nothing written, cache untouched.

func TestCreateNewQuestFile_SkeletonIsBootSafe(t *testing.T)
// returns max-cache-id+1; file exists; GetQuest(id).Validate() == nil;
// skeleton has steps [start, end], zero triggers.

func TestDeleteQuest_RemovesFileCacheAndFlags(t *testing.T)
// after delete: file gone, GetQuest(id) nil, flag keys gone.
// Also idempotent when the file is already missing (mob delete pattern).
```

- [ ] **Step 2: RED** — `go test ./internal/quests/ -run "SaveQuest|CreateNewQuestFile|DeleteQuest" -count=1` fails to compile.
- [ ] **Step 3: Implement.**

```go
// SaveQuest validates and writes a quest definition, updating the cache and
// flag registry. Renames move the file (old path computed BEFORE the cache
// swap). Marshal form is the proven fixed point (roundtrip_test.go), so
// repeat saves diff minimally.
func SaveQuest(q Quest) error {
	if err := q.Validate(); err != nil {
		return err
	}
	out, err := yaml.Marshal(q)
	if err != nil {
		return err
	}
	dir := questsDataRoot()
	oldPath := ""
	if prev, ok := quests[q.QuestId]; ok {
		oldPath = filepath.Join(dir, prev.Filename())
	}
	newPath := filepath.Join(dir, q.Filename())
	if err := os.WriteFile(newPath, out, 0644); err != nil {
		return err
	}
	if oldPath != "" && oldPath != newPath {
		if err := os.Remove(oldPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	cp := q
	quests[q.QuestId] = &cp
	RegisterFlags(q.QuestId, q.Flags)
	return nil
}
```

  `CreateNewQuestFile(name string) (int, error)`: next id = max cache key
  +1 (min 1); skeleton
  `Quest{QuestId: id, Name: name, Description: "An unfinished quest.", Steps: []QuestStep{{Id: "start", Description: "…"}, {Id: "end", Description: "…"}}}`;
  persist via `SaveQuest`. `DeleteQuest(questId int) error`: compute path
  from cached Filename, `os.Remove` (swallow IsNotExist), delete cache
  entry, delete this quest's keys from the flag registry (iterate
  `q.Flags`, `delete(flagRegistry, fmt.Sprintf("%d-%s", …))`).
  Rewrite `LoadDataFiles`'s dataPath to use `questsDataRoot()` so the seam
  covers loads too.
- [ ] **Step 4: GREEN** + `go test ./internal/quests/ -count=1` (harness +
  round-trip must stay green). **Step 5: Commit**
  `feat(quests): SaveQuest/CreateNewQuestFile/DeleteQuest writer with cache+flag contract`

### Task 2: Registry validation — `internal/quests/validate_refs.go` (TDD)

**Files:**
- Create: `internal/quests/validate_refs.go`, `internal/quests/validate_refs_test.go`

- [ ] **Step 1: Failing tests** (permissive-validators helper like 5b's;
  one test per rule, errsContaining/warnsContaining helpers):
  refusals — foreign token in `has`/`missing`/trigger `quest_token`/rewards
  `questid` that StepExists rejects; own-quest token naming a step not in
  the INCOMING q.Steps; unknown mob/item/room/buff id anywhere they appear
  (trigger filters, conditions has_item/missing_item/in_room, map_target>0,
  npc_say mob + line speakers, spawn_mob/spawn_item id+room, teleport,
  apply_buff, lock/unlock room, rewards itemid/buffid/roomid);
  unknown spell (teach_spell, rewards spellid), skill (train_skill, rewards
  skillinfo parts, skill_use event filter), stat (train_stat, rewards
  stat_info parts), recipe (learn_recipe, rewards recipe_info parts),
  faction (bump_rep, rewards rep_faction); flag key/value not declared in
  the incoming q.Flags NOR FlagDeclared (has_flag/missing_flag/set_flag).
  Warnings — a step id no own-trigger grants and DialogueGrants rejects;
  npc_say mob where MobHasDialogue is false; step MapTarget>0 that no
  trigger's room field references.
- [ ] **Step 2: RED.** **Step 3: Implement**

```go
// QuestValidators are the registry checks the pure validator cannot own —
// injected (5b pattern) so tests and worldless callers can fake them.
type QuestValidators struct {
	StepExists     func(token string) bool // foreign "<qid>-<step>" resolves
	MobExists      func(id int) bool
	ItemExists     func(id int) bool
	RoomExists     func(id int) bool
	BuffExists     func(id int) bool
	SpellExists    func(id string) bool
	SkillExists    func(name string) bool
	StatExists     func(name string) bool
	RecipeExists   func(id string) bool
	FactionExists  func(id string) bool
	FlagDeclared   func(key, value string) bool // foreign quests' flags
	DialogueGrants func(token string) bool      // any dialogue file grants it
	MobHasDialogue func(mobId int) bool
}

// ValidateQuestRefs checks every id/token/name the quest references against
// the live registries. Own-quest tokens and flags validate against the
// INCOMING definition (they may change in the same save). Returns refusals
// and warnings; structural rules are (*Quest).Validate()'s job, not ours.
func ValidateQuestRefs(q Quest, v QuestValidators) (errs []string, warns []string)
```

  Implementation walks steps/rewards/flags/triggers (conditions + actions,
  recursing into `sequence.on_complete` one level). Own-token test:
  `strings.HasPrefix(tok, fmt.Sprintf("%d-", q.QuestId))` → step must be in
  q.Steps; else `v.StepExists`. Own-flag test: key in q.Flags (+ value in
  its Values) → ok; else `v.FlagDeclared`. Every error names its location
  ("trigger 3 action 2 spawn_mob: mob 999 does not exist").
- [ ] **Step 4: GREEN + commit** `feat(quests): ValidateQuestRefs — registry-backed save-time checks (boot flag panic becomes a refusal)`

### Task 3: GMCP — `modules/gmcp/gmcp.Quest.go` + wiring (TDD)

**Files:**
- Create: `modules/gmcp/gmcp.Quest.go`, `modules/gmcp/gmcp.Quest_test.go`
- Modify: `modules/gmcp/gmcp.go` (routing), `modules/gmcp/gmcp.Build.go` (dispatch)

- [ ] **Step 1: Failing handler tests** with `fakeQuestWorld` (mirror
  `fakeMobWorld`): update-refuses-invalid-saves-nothing (bad registry ref via
  strict fake validators), update-with-warning-saves-and-returns-Warnings,
  get/list shapes (list hides id ≥ 1000000), create returns fresh detail,
  delete-refused-while-referenced (fake references func returns one entry),
  clean delete calls deps.del + deps.reindex once.
- [ ] **Step 2: RED.** **Step 3: Implement.**

```go
type questDeps struct {
	load       func(id int) *quests.Quest
	all        func() []quests.Quest
	save       func(q quests.Quest) error
	create     func(name string) (int, error)
	del        func(id int) error
	references func(id int) []questRefEntry // delete guard (Task 4)
	reindex    func()                       // questengine.LoadDataFiles
	validators quests.QuestValidators
}
```

  Handlers: `buildQuestList` (rows: id, name, secret, repeatable,
  stepCount, triggerCount), `buildQuestGet` (Quest + `collectQuestEnums()`),
  `buildQuestUpdate` (Validate() → ValidateQuestRefs → refuse on errs, save,
  reindex, return Warnings), `buildQuestCreate` (create, reindex, re-send
  fresh detail — dialogue Create pattern), `buildQuestDelete` (references
  non-empty → refusal listing each verbatim; else del + reindex).
  `collectQuestEnums()`: quest tokens (per-step, `"%d-%s"` + quest name —
  lift from `collectDialogueEnums`), flag registry
  (`quests.GetFlagRegistry()`), and three static vocabulary tables carried
  in this file, each entry `{key, description}`:
  EVENTS (10: room_enter, item_give, skill_use, mob_death, command,
  command_issued, item_gain, dialogue, quest_granted, room_interact — with
  the command-vs-command_issued distinction spelled out), CONDITIONS (9:
  has, missing, in_room, has_item, missing_item, has_flag, missing_flag,
  has_gold, has_masterwork), ACTIONS (24: grant, consume_item, give_item,
  give_gold, charge_gold, npc_say, send_text, room_text, spawn_mob,
  spawn_item, lock_exits, unlock_exits, teach_spell, train_skill,
  train_stat, learn_recipe, apply_buff, teleport, give_mutation, set_flag,
  sequence, bump_rep, declare_bounty — plus per-action picker enums: buffs
  `{id,name}` via GetAllBuffIds/GetBuffSpec, spells `{id,name}` via
  GetAllSpells, recipes via crafting.GetAll keys, factions via
  factions.AllDefinitions, skills via the SkillTag constants, stats fixed).
  `realQuestDeps()` wires the live registries; `DialogueGrants` and
  `MobHasDialogue` come from the Task 4 dialogue walker;
  `reindex: questengine.LoadDataFiles`.
- [ ] **Step 4:** routing in `gmcp.go` beside `Build.Dialogue.*`:
  `` `Build.Quest.List`, `Build.Quest.Get`, `Build.Quest.Update`, `Build.Quest.Create`, `Build.Quest.Delete` ``;
  dispatch cases in `gmcp.Build.go`'s `handleBuildOp` (payloads
  `Build.Quests` for the list, `Build.Quest` for detail, `Build.Result` for
  mutations — mirror the dialogue cases).
- [ ] **Step 5: GREEN + `go build ./...` + commit**
  `feat(build): Build.Quest.* handlers behind questDeps`

### Task 4: Reference guard + dialogue walker (TDD)

**Files:**
- Create: `internal/dialogue/walk.go` (+ test), `modules/gmcp/gmcp.Quest_refs.go` (+ test)

- [ ] **Step 1: Failing tests.** dialogue walker: against a temp dialogue
  tree (the `overrideDataFilesDir` helper from `internal/dialogue/save_test.go`),
  `WalkAllFiles(fn)` visits every zone/mob file with parsed DialogueFile.
  Reference scan: quest 41 referenced by a dialogue file granting
  `41-start` → one entry naming mob+zone+field; by another quest's trigger
  condition `has: [41-end]` → entry; by rewards `questid: 41-start` →
  entry; unreferenced quest → empty.
- [ ] **Step 2: RED.** **Step 3: Implement.** `dialogue.WalkAllFiles(fn
  func(mobId int, zone string, df *DialogueFile))` — exported walk of
  `{DataFiles}/dialogue/<zone>/<mobId>.yaml` (the loader in
  questengine/loader.go:153 is the model; do NOT touch the dialogue cache).
  `scanQuestReferences(questId int) []questRefEntry` walks: every dialogue
  file's gate fields (grantsQuest, questRequired, questExcluded,
  questFlagRequired/Excluded, setsQuestFlag — across patterns, tree nodes,
  root variants) for tokens/flag-keys with the `"<id>-"` prefix; every
  OTHER quest's triggers (has/missing/quest_token/grant), rewards QuestId,
  and flag references. Entry: `{Kind, Where, Detail string}` rendered
  verbatim in the refusal. Also implement `DialogueGrants`/`MobHasDialogue`
  on top of the same walker (built once per validation call, not cached).
- [ ] **Step 4: GREEN + commit** `feat(build): quest delete reference guard + dialogue walker`

### Task 5: Client — list, identity, steps, rewards, flags (`quests.js`)

**Files:**
- Create: `_datafiles/html/public/static/js/quests.js`

- [ ] **Step 1:** `window.Builder.QuestPanel` skeleton mirroring `mobs.js`
  structure (ce/gmcp/toast/field helpers copied per house pattern; panel
  state: rows, search, selectedId, dirty, saving). List: search by id/name,
  badges secret/repeatable/`N steps`/`N triggers`, `+ New Quest` prompt →
  `Build.Quest.Create {name}`.
- [ ] **Step 2:** Inspector sections 1-4. Steps as ORDERED collapsible rows
  (lift `collapsible()` with the ▸/▾ whole-row toggle from dialogue.js —
  identical helper, keep it file-local per house js style): id, description,
  hint textarea, map-target = zone select + per-zone room dropdown
  (`Build.Room.List` → `Build.Rooms`, the mob test-spawn pattern) + a
  "quest giver" checkbox emitting -1. Rewards: one field per canonical
  reward with datalists (items from `window.Builder.itemRows`, buffs/
  spells/recipes/factions/skills/stats from enums). Flags: key/values-chips/
  description rows with the "dialogue setsQuestFlag must match" hint.
  `gatherFile()` emits yaml-name keys exactly (questid, name, description,
  secret, steps[{id,description,hint,map_target}], rewards{...},
  triggers[...], flags[...], repeatable, cooldown_rounds).
- [ ] **Step 3:** `node --check` clean. Commit
  `feat(build): quest panel — list, identity, steps, rewards, flags (5c UI part 1)`

### Task 6: Client — triggers section

**Files:**
- Modify: `_datafiles/html/public/static/js/quests.js`

- [ ] **Step 1:** Trigger rows: ordered collapsibles summarizing
  `event · filters · N actions`. Event `<select>` from enums.events (with
  description line under it); filter fields shown per event from a static
  map:

```js
var EVENT_FILTERS = {
  room_enter: ["room"], item_give: ["mob", "item"], skill_use: ["skill"],
  mob_death: ["mob", "room"], command: ["command", "room"],
  command_issued: ["command", "room"], item_gain: ["item"],
  dialogue: ["mob", "topic"], quest_granted: ["quest_token"],
  room_interact: ["room", "noun", "verb"]
};
```

- [ ] **Step 2:** Conditions drawer (collapsed by default): has/missing
  token chip-lists on the token datalist, in_room/has_item/missing_item
  pickers, has_flag/missing_flag key→value rows on the flag enums,
  has_gold/has_masterwork numbers.
- [ ] **Step 3:** Actions as an ORDERED list inside each trigger ("actions
  run in order"), add-action via a typed dropdown over enums.actions. One
  sub-form builder per type, table-driven:

```js
// Each entry: fields in emit order; kind drives the input widget.
// mob/item/room/buff = numeric picker on the matching datalist;
// spell/recipe/faction/skill/stat = string picker; text/num/bool literal.
var ACTION_FORMS = {
  grant:        [{k:"grant", kind:"token"}],
  consume_item: [{k:"consume_item", kind:"item"}],
  give_item:    [{k:"give_item", kind:"item"}],
  give_gold:    [{k:"give_gold", kind:"num"}],
  charge_gold:  [{k:"charge_gold", kind:"num"}],
  npc_say:      [{k:"mob", kind:"mob"}, {k:"lines", kind:"saylines"}],
  send_text:    [{k:"send_text", kind:"text"}],
  room_text:    [{k:"room_text", kind:"text"}],
  spawn_mob:    [{k:"id", kind:"mob"}, {k:"room", kind:"room"}],
  spawn_item:   [{k:"id", kind:"item"}, {k:"room", kind:"room"}],
  lock_exits:   [{k:"room", kind:"room"}, {k:"player_scoped", kind:"bool"}],
  unlock_exits: [{k:"room", kind:"room"}, {k:"player_scoped", kind:"bool"}],
  teach_spell:  [{k:"teach_spell", kind:"spell"}],
  train_skill:  [{k:"skill", kind:"skill"}, {k:"level", kind:"num"}],
  train_stat:   [{k:"stat", kind:"stat"}, {k:"amount", kind:"num"}],
  learn_recipe: [{k:"recipe", kind:"recipe"}],
  apply_buff:   [{k:"buff", kind:"buff"}, {k:"source", kind:"text"}],
  teleport:     [{k:"teleport", kind:"room"}],
  give_mutation:[{k:"give_mutation", kind:"bool"}],
  set_flag:     [{k:"key", kind:"flagkey"}, {k:"value", kind:"text"}],
  bump_rep:     [{k:"faction", kind:"faction"}, {k:"delta", kind:"num"}],
  declare_bounty: "custom",   // dedicated builder: issuer type/id, target,
                              // condition, expiry/gold/rep overrides, reason
  sequence:     "custom"      // delay_between + saylines + nested
                              // on_complete action list (ONE level: the
                              // nested add-action dropdown excludes
                              // "sequence") + lock_message
};
```

  `saylines` = mini ordered list of {delay, text, speaker (mob picker,
  0 = npc_say mob), emote}. Emission: single-key actions nest under their
  yaml key (`{grant: "41-start"}`, `{spawn_mob: {id: …, room: …}}`) —
  gather must produce exactly the `ActionDef` yaml shapes.
- [ ] **Step 4:** Save/refusal/warning plumbing: `Build.Quest.Update` with
  gathered file; `onResult` renders refusals verbatim above the form
  (`#q-errors`), warnings amber persistent (`#q-warnings`); delete confirm
  lists guard refusals verbatim when refused. `node --check` clean. Commit
  `feat(build): quest panel — trigger DSL editor (5c UI part 2)`

### Task 7: `build.html` wiring

**Files:**
- Modify: `_datafiles/html/public/build.html`

- [ ] **Step 1:** Add the Quests tab beside Zones (grep `zones` in the mode
  toggle markup and mirror every occurrence: tab button, `questlist`
  sidebar div, mode switch handling in `setMode`, list-mode conditionals at
  ~1172-1197). Script tag beside `zones.js`. Routing (beside the zone
  lines, CRLF-aware edits):

```js
if (ns === "Build.Quests") { if (QP) QP.render(obj); return; }
if (ns === "Build.Quest") { if (QP) QP.renderDetail(obj); return; }
if (ns === "Build.Result" && window.Builder.mode === "quests") { if (QP) QP.onResult(obj); return; }
if (ns === "Build.Rooms" && window.Builder.mode === "quests") { if (QP) QP.onRoomList(obj); return; }
if (ns === "Build.Mobs" && window.Builder.mode === "quests") { if (QP) QP.onMobList(obj); return; }
```

  Entering the mode fires `Build.Quest.List` + `Build.Mob.List` (mob picker
  rows). Inline-script parse check + `node --check quests.js`.
- [ ] **Step 2: Commit** `feat(build): Quests tab wiring (5c)`

### Task 8: Verification gate

- [ ] `go test ./... -count=1`; gofmt/vet on touched packages;
  `DOGMUD_BOOT_SMOKE=1 go test -run TestSmoke_NoNewSilentlyIgnoredYAMLKeys -count=1 .`
  (expect 11/11, new: 0); panic-mode boot after instance wipe.
- [ ] Headless WS E2E sanity (the mob-editor `mob_e2e.mjs` scratchpad
  pattern): login → `Build.Quest.List` → Get a real quest → Update with an
  unchanged gather → expect ok + zero-churn-after-first-save; Update with a
  bogus token → expect refusal naming it.
- [ ] PATCH_NOTES.md staff-voice entry; memory update (epic topic file +
  MEMORY.md status/counts).
- [ ] Hand the **user browser gate**: edit a step hint → see it live in the
  quest log without reboot; add a trigger (e.g. room_enter + send_text) and
  fire it in-game; trip refusals (unknown token, undeclared flag, bogus mob
  id in npc_say); delete guard names a dialogue reference verbatim; create
  a quest and immediately grant it via `questtoken`. First-save formatting
  churn on an old quest is EXPECTED (one-time canonicalization) — say so in
  the handoff. Merge to master on the user's word.

---

## Self-review notes

- The plan deliberately does NOT add a quests-package mutex: all mutation
  paths run on MainWorker (GMCPBuildOp), matching the mob writer's model.
  If a future caller mutates off-worker, that's its bug to bring a lock.
- `reindex` after every mutation rebuilds the whole trigger index (~60ms);
  wired as a deps func so handler tests assert call-count without touching
  the real engine.
- Rewards `skillinfo`/`stat_info`/`recipe_info`/`item_info` are composite
  strings ("skill:level,…") — validated by splitting in ValidateQuestRefs;
  the client edits them as structured row-pairs and joins on gather.
- `1000000-generic_quest.yaml` is hidden from the list but still loads,
  round-trips, and is protected by the guard like any quest.
