# Quest Editor (Admin Web-Building 5c) — Design

**Goal:** a fifth `/build` mode — Quests — giving admins full-parity authoring
of quest files: identity, steps, rewards, flag declarations, and the complete
trigger DSL, plus create/delete with a reference guard. Saves are validated
against the same rules the boot loader enforces (and more), and go live
without a reboot.

## The architectural fact that shapes everything: dual parsing

Every quest YAML is parsed **twice by two different parsers**:

1. **`internal/quests`** (yaml.v2, mostly TAG-LESS fields) — drives the
   player-facing quest log and **pays rewards**
   (`internal/hooks/Quest_HandleQuestUpdate.go` reads
   `quests.GetQuest(...).Rewards`). Tag-less binding means the rewards block
   only loads from **no-underscore keys**: `questid`, `gold`, `itemid`,
   `buffid`, `skillinfo`, `spellid`, `playermessage`, `roommessage`,
   `roomid`. Tagged exceptions take snake_case: `stat_info`, `recipe_info`,
   `item_info`, `rep_faction`, `rep_amount`.
2. **`internal/questengine`** (`QuestDef` in `types.go`, fully snake_case
   tagged) — owns steps' `map_target`, `secret`, and the entire `triggers:`
   DSL. Its `QuestRewards` struct is **vestigial** (nothing reads it;
   verified), as are `chain_quest` (no reader — real chaining is the rewards
   `questid`) and `linear` (no reader; 4 files set it to no effect).

**Consequence for the writer:** the canonical emitted form is
questengine-shaped YAML for everything EXCEPT the rewards block, which must
use the quests-package no-underscore keys. A writer that naively marshals
`QuestDef` would emit `item_id:` and silently un-pay every reward it touches
— the exact silent-no-op class the reward-key gotcha memo warns about.

**Decision:** re-tag `questengine.QuestRewards` to the no-underscore keys so
both parsers agree on one canonical form forever (nothing reads the engine
copy, so this changes no behavior; it kills the divergent-vocabulary class).
Drop the dead `chain_quest` and `linear` fields from `QuestDef` (compiler
sweep confirms no readers) and strip `linear:` from the 4 files in the
canonicalization pass.

## Server side

### Authoring struct and writer — `internal/questengine/save.go`

`QuestDef` is the fuller shape, so the writer lives in questengine:

- `SaveQuestDef(q QuestDef) error` — validate, marshal, write to
  `quests/<id>-<ConvertForFilename(name)>.yaml` (rename moves the file, mob
  editor's old-path-first pattern), then **swap BOTH caches**: the
  questengine engine's trigger index and the `internal/quests` package cache
  (a small exported re-load hook in `internal/quests`, fed by re-parsing the
  file just written — never by converting structs in memory, so the two
  parsers stay the only source of truth).
- `CreateNewQuestFile(name string) (int, error)` — next free id from the
  loaded cache max (id_inventory convention), skeleton with one `start` and
  one `end` step and zero triggers; boot-safe by construction.
- `DeleteQuestDef(questId int) error` — refuses while referenced (guard
  below), removes file + both cache entries.

### Validation — extend the loader's own rules

Save-time validation calls the existing `validateQuestDef` (events
vocabulary incl. `command_issued`, non-empty actions, duplicate/empty step
ids, flag declarations, own-quest grant tokens) and extends it with
registry-backed checks, **injected** per the 5b pattern so gmcp handler
tests run without a world:

- cross-quest tokens in `has`/`missing`/`quest_granted`/rewards `questid`
  resolve to real quest steps
- `mob`/`item`/`room`/`buff` ids exist (triggers, conditions, actions,
  `map_target`, `npc_say`, `spawn_*`, `teleport`, `apply_buff`)
- `teach_spell`/rewards `spellid` is a real spell; skills, stats, recipe
  ids, faction slugs valid
- `set_flag`/`has_flag`/`missing_flag` keys+values declared (the boot
  panic, moved to save time)
- warnings (save succeeds, listed): a step no trigger grants and no
  dialogue file grants; `map_target` room in a different zone than the
  trigger rooms (likely typo); an `npc_say` mob that has no dialogue file.

### Reference guard for delete

Refuse deletion while anything references the quest, reporting each
reference verbatim: dialogue files (`grantsQuest`, `questRequired`,
`questExcluded`, `questFlagRequired/Excluded`, `setsQuestFlag` — via the 5b
dialogue loader), other quests' triggers/conditions/rewards, and
achievements YAML if it names quest tokens.

### GMCP — `Build.Quest.*` behind a `questDeps` seam

`List` (id, name, secret, repeatable, step/trigger counts), `Get` (full
QuestDef + rewards + enums), `Update`, `Create`, `Delete`. Enums payload:
event/condition/action vocabularies with one-line descriptions, quest
tokens, flag keys, moods of reward-relevant registries (items via the
existing login-time prefetch, mobs, buffs, spells, skills, stats, recipes,
factions). Routed through `GMCPBuildOp` on MainWorker behind `requireAdmin`,
like every other Build verb. `BuildResult.Warnings` reused from 5b.

## Client side — `quests.js`, fifth mode

List panel: search by id/name, badges (secret / repeatable / N steps /
N triggers), `+ New Quest`.

Inspector sections:

1. **Identity** — name, description, secret, repeatable + cooldown rounds.
2. **Steps** — ordered list (↑/↓; order is the progression order), each
   row: id, description, hint, map target (zone→room picker, mirroring the
   mob editor's test-spawn picker; -1 = "quest giver" sentinel surfaced as
   a checkbox).
3. **Rewards** — the 13 canonical fields with pickers/datalists; the panel
   never shows raw yaml keys, so the no-underscore/snake_case split becomes
   invisible to authors.
4. **Flags** — key / allowed values / description rows; the editor reminds
   that dialogue `setsQuestFlag` must match these.
5. **Triggers** — ordered collapsible rows (5b's whole-row toggle shell).
   Each: event select that shows only that event's filter fields
   (room/mob/item/skill/command/topic/noun/verb/quest_token), a conditions
   drawer (has/missing tokens with the token datalist, in_room, has_item,
   flags, gold, masterwork), and an **ordered actions list** — one sub-form
   per action type, chosen from a typed dropdown. `npc_say` lines get their
   own mini-list (delay/text/speaker/emote); `sequence` nests lines +
   `on_complete` actions exactly one level (matching the engine).

Wiring in `build.html`: Quests tab, mode routing, `Build.Result` dispatch,
script tag. Same browser-cache caveat as every static-js change (hard
refresh after deploy).

## Canonicalization sweep + round-trip proof

The 5b proof pattern over all 79 quest files: marshal must be a **fixed
point** (second marshal byte-identical) and section-count guards
(steps/triggers/actions/flags preserved), plus a rewards-equivalence check
against the `internal/quests` parse so the writer can never silently un-pay
a reward. The sweep strips the 4 dead `linear:` lines. Any file the writer
would reorder/canonicalize is committed as churn ONCE here, so future saves
diff cleanly.

## Testing & gates

TDD throughout: fake `questDeps` for handlers, injected validators,
writer cache-contract test, round-trip fixed-point over the live tree,
refusal test per validation rule. Full suite + vet + panic-mode boot with
instance wipe before handoff. Editing tooling ⇒ no adversarial content
playtest owed; the user browser gate covers: edit a step hint and see it
live in the quest log, add a trigger and fire it in-game, trip a refusal
(unknown token / undeclared flag / snake_case never visible), delete guard
naming a dialogue reference, create + immediately grant a new quest.

## Out of scope

The pinned general admin help page (after the epic); trigger *simulation*
(dry-running a trigger against a fake player); editing achievements;
`1000000-generic_quest.yaml` (template file — the list hides id ≥ 1000000).
