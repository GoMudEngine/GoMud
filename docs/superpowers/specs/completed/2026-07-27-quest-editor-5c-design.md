# Quest Editor (Admin Web-Building 5c) — Design

> **Revised 2026-07-27** after the 5c-pre unification landed (`d305ab221`).
> The original draft spent its hardest sections working around quest YAML's
> dual parse; that machinery is gone. See
> `2026-07-27-quest-unification-5c-pre-design.md` for what changed underneath.

**Goal:** a fifth `/build` mode — Quests — giving admins full-parity authoring
of quest files: identity, steps, rewards, flag declarations, and the complete
trigger DSL, plus create/delete with a reference guard. Saves are validated
against the boot loader's own rules (and more), and go live without a reboot.

## Foundation (already true, courtesy of 5c-pre)

- `quests.Quest` is THE definition — one explicitly-tagged parse, trigger DSL
  included. The editor writer just marshals it; the silent-unpaid-reward
  class is structurally dead.
- **Marshal is a proven fixed point** over all 79 live files
  (`internal/quests/roundtrip_test.go`), with section-survival and
  rewards-identity guards. The writer inherits this as its safety net.
- `(*Quest).Validate()` (events vocabulary incl. `command_issued`, non-empty
  actions, duplicate/empty step ids, flag declarations, own-quest grant
  tokens) runs on **every** fileloader parse — a file the writer emits cannot
  boot-break the server for those rule classes.
- questengine holds no state of its own beyond the trigger index built from
  `quests.GetAllQuests()`; rebuilding it is one cheap call
  (`questengine.LoadDataFiles()`, ~60ms for all 79).
- Dead fields (`linear`, `chain_quest`) no longer exist; 29 dead `linear:`
  lines are already stripped from content.

## Server side

### Writer — `internal/quests/save.go`

The mob/dialogue-editor writer pattern, minus all format gymnastics:

- `SaveQuest(q Quest) error` — run `q.Validate()`, marshal, write to
  `quests/<id>-<ConvertForFilename(name)>.yaml` (a rename moves the file:
  old-path-computed-first, mob editor's pattern), then swap the package
  cache entry and re-register the quest's flags.
- `CreateNewQuestFile(name string) (int, error)` — next free id from the
  loaded cache max; skeleton with one `start` and one `end` step, zero
  triggers; boot-safe by construction (must pass `Validate()`).
- `DeleteQuest(questId int) error` — removes file + cache entry + flag
  registrations. Reference guard lives at the gmcp layer (below), which
  refuses before calling this.

**Engine re-index:** `internal/quests` cannot call questengine (import
direction). The gmcp handler calls `questengine.LoadDataFiles()` after every
successful Save/Create/Delete — a full cheap rebuild, no new plumbing, and
the index can never drift from the cache. Edits are live immediately: the
next `Notify` walks the new index, and the quest log / dialogue-editor token
enums read `quests.GetAllQuests()` fresh.

**One-time churn caveat:** authored files are not in canonical marshal form
(key order, indentation). The FIRST editor save of an old quest produces a
one-time formatting diff; every save after that is minimal (fixed point).
Same trade the room/mob/dialogue editors made — accepted, not fought.

### Validation — `Validate()` plus injected registry checks

`(*Quest).Validate()` covers the structural rules. The gmcp layer adds
registry-backed checks via an injected `questValidators` struct (5b pattern —
handler tests run with no world loaded):

- cross-quest tokens in `has`/`missing`/`quest_granted`-token/rewards
  `questid` resolve to real quest steps
- `mob`/`item`/`room`/`buff` ids exist (trigger filters, conditions,
  `map_target`, `npc_say` speakers, `spawn_*`, `teleport`, `apply_buff`)
- `teach_spell`/rewards `spellid` is a real spell; skill names, stat names,
  recipe ids, faction slugs valid
- `set_flag`/`has_flag`/`missing_flag` keys+values declared — this is
  `ValidateAllFlags`' boot PANIC moved to save time, so a bad save is
  refused instead of taking the next boot down
- warnings (save succeeds, amber list via `BuildResult.Warnings`): a step no
  trigger and no dialogue file grants; an `npc_say` mob with no dialogue
  file; `map_target` pointing at a room no trigger references (likely typo).

### Reference guard for delete

Refuse deletion while anything references the quest, reporting each
reference verbatim: dialogue files (`grantsQuest`, `questRequired`,
`questExcluded`, `questFlagRequired/Excluded`, `setsQuestFlag` — walked via
the 5b dialogue loader), other quests' triggers/conditions/rewards
`questid`, and achievements YAML if it names quest tokens.

### GMCP — `Build.Quest.*` behind a `questDeps` seam

`List` (id, name, secret, repeatable, step/trigger counts; hides id ≥
1000000), `Get` (full Quest + enums), `Update`, `Create`, `Delete`. Enums
payload: event/condition/action vocabularies with one-line descriptions,
quest tokens (per-step, with quest names — same shape the dialogue editor
uses), flag keys, and pickers for mobs, buffs, spells, skills, stats,
recipes, factions (items ride the existing login-time `Build.Items`
prefetch). Routed through `GMCPBuildOp` on MainWorker behind `requireAdmin`.

## Client side — `quests.js`, fifth mode

List panel: search by id/name, badges (secret / repeatable / N steps /
N triggers), `+ New Quest`.

Inspector sections:

1. **Identity** — name, description, secret, repeatable + cooldown rounds.
2. **Steps** — ordered list (↑/↓; order is the progression order), each
   row: id, description, hint, map target (zone→room picker, mirroring the
   mob editor's test-spawn picker; -1 = deliberate NO-marker sentinel —
   suppresses trigger inference, per questengine.ResolveQuestTarget —
   surfaced as a checkbox. An earlier draft mislabeled -1 "quest giver").
3. **Rewards** — the canonical fields with pickers/datalists; the panel
   never shows raw yaml keys.
4. **Flags** — key / allowed values / description rows; the editor reminds
   that dialogue `setsQuestFlag` must match these.
5. **Triggers** — ordered collapsible rows (5b's whole-row toggle shell with
   ▸/▾). Each: event select that shows only that event's filter fields
   (room/mob/item/skill/command/topic/noun/verb/quest_token), a conditions
   drawer (has/missing tokens with the token datalist, in_room, has_item,
   flags, gold, masterwork), and an **ordered actions list** — one sub-form
   per action type, chosen from a typed dropdown. `npc_say` lines get their
   own mini-list (delay/text/speaker/emote); `sequence` nests lines +
   `on_complete` actions exactly one level (matching the engine).

Wiring in `build.html`: Quests tab, mode routing, `Build.Result` dispatch,
script tag. Same browser-cache caveat as every static-js change (hard
refresh after deploy).

## Testing & gates

TDD throughout: fake `questDeps` for handlers, injected validators, writer
cache-contract test (save swaps cache + re-registers flags; delete removes
both), refusal test per validation rule. The existing equivalence harness
and round-trip fixed-point tests must stay green untouched. Full suite +
vet + panic-mode boot with instance wipe before handoff. Editing tooling ⇒
no adversarial content playtest owed; the user browser gate covers: edit a
step hint and see it live in the quest log, add a trigger and fire it
in-game without a reboot, trip a refusal (unknown token / undeclared flag),
delete guard naming a dialogue reference, create + immediately grant a new
quest.

## Out of scope

The pinned general admin help page (after the epic); trigger *simulation*
(dry-running a trigger against a fake player); editing achievements;
`1000000-generic_quest.yaml` (template file — hidden from the list).
