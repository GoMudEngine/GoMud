# Quest Definition Unification (5c-pre) — Design

**Goal:** collapse the two independent parses of quest YAML into one struct
with one loader, so the 5c quest editor writes against a single source of
truth — and so the tag-less-binding landmine class (the no-underscore
reward-key gotcha) is structurally dead before an editor can step on it.

This is a refactor with an equivalence proof, not a behavior change. No
player-visible behavior moves.

## Current state (verified)

- **Two parsers, same files.** `internal/quests` parses every quest file
  into `Quest` (yaml.v2, mostly TAG-LESS fields — binding is the lowercased
  field name with no underscore handling) and **pays rewards** from that
  copy (`internal/hooks/Quest_HandleQuestUpdate.go`). `internal/questengine`
  re-parses the same files into `QuestDef` (fully snake_case tagged) and
  owns steps' `map_target` and the entire `triggers:` DSL.
- **Boot order already correct for consolidation:** `main.go:1740`
  `quests.LoadDataFiles()` then `main.go:1746` `questengine.LoadDataFiles()`.
- **No import cycle risk:** questengine already imports quests (bridge.go,
  loader.go), so quests is the lower package. The 12 importers of quests use
  its lookup API (`GetQuest`, `GetAllQuests`, token helpers); the ~35
  importers of questengine call event-notify entry points (`Notify`,
  `GetEngine`, `EventDetails`), not the definition types.
- **Everything goes through `internal/fileloader`** (yaml.v2 + the
  `StrictDecodeProbe` drift gate).
- **Dead weight (verified no readers):** `QuestDef.Linear` (4 files author
  `linear:` to no effect), `QuestRewards.ChainQuest`, and the entire
  `questengine.QuestRewards` struct (nothing reads the engine's copy of
  rewards).
- **The drift-gate baseline is 60% quest artifacts:** 17 of its 28 entries
  exist only because each parser sees the other's fields as unknown keys
  (all the `*|questengine.QuestRewards` entries, `triggers|quests.Quest`,
  `map_target|quests.QuestStep`, `repeatable|questengine.QuestDef`, etc.).
  Current live run: 26 distinct keys, `new: 0` — two baselined entries are
  already stale.

## Target architecture

### `internal/quests` owns the one definition

`Quest` becomes the full shape, every field **explicitly tagged** with the
key that binds it TODAY (no more tag-less inference):

- Existing fields keep their current effective keys: rewards stay
  `questid`, `gold`, `itemid`, `buffid`, `skillinfo`, `spellid`,
  `playermessage`, `roommessage`, `roomid` (no-underscore — pinned by tag
  now instead of by accident) plus the tagged five (`stat_info`,
  `recipe_info`, `item_info`, `rep_faction`, `rep_amount`).
- `QuestStep` gains `MapTarget int` (`map_target`).
- `Quest` gains `Triggers []TriggerDef` and `Linear` is NOT carried over.
- The definition types move from questengine to quests: `TriggerDef`,
  `Conditions`, `ActionDef` and its parameter structs (`NpcSayDef`,
  `SayLineDef`, `SpawnDef`, `ExitLock`, `SkillDef`, `StatDef`, `RecipeDef`,
  `BuffDef`, `SequenceDef`, `QuestFlagAction`, `BumpRepDef`,
  `DeclareBountyDef`). Mechanical package move; questengine references them
  as `quests.TriggerDef` etc.
- `validateQuestDef`'s checks (event vocabulary incl. `command_issued`,
  non-empty actions, duplicate/empty step ids, flag declarations, own-quest
  grant tokens) move to `internal/quests` as `(*Quest).Validate()` — the
  fileloader interface hook quests already implements — so validation runs
  at parse time for every future caller including the 5c writer.

### `internal/questengine` stops parsing files

- `questengine.LoadDataFiles()` builds its trigger index from
  `quests.GetAllQuests()` — no file I/O.
- The internal `questId`/`trigId` fields currently smuggled onto
  `TriggerDef` during load become an engine-side index wrapper
  (`indexedTrigger{def *quests.TriggerDef; questId int; trigId string}`) —
  the definition structs stay pure data.
- `QuestDef`, `questengine.QuestStep`, `questengine.QuestRewards`,
  `questengine.QuestFlagDef` are deleted. The compiler enumerates every
  consumer (the dead-code-sweep rule); expected fallout is confined to
  questengine internals and their tests.
- Evaluation logic (engine, conditions, actions, guards, bridge, map_target
  inference) does not change — only where its input comes from.

### API compatibility

`quests.GetQuest`, `GetAllQuests`, token helpers, and `Quest.Rewards`
consumers (hooks, usercommands, gmcp, seeders, characters, users) keep
their exact signatures. `questengine.Notify`/`GetEngine`/`EventDetails` and
all ~35 call sites untouched. `main.go` boot order unchanged (the second
call now just builds the index).

## The equivalence proof (non-negotiable, lands FIRST)

Before any old path is deleted, a test harness parses **all 79 live quest
files** three ways — old `quests.Quest`, old `questengine.QuestDef`, new
unified `Quest` — and asserts:

- every field the old quests parse populated is identical in the new parse
  (rewards especially: this is the reward-payment path on prod content);
- every field the old questengine parse populated (steps + map_target,
  secret, repeatable/cooldown, flags, full trigger tree) is identical;
- file count identical, no quest gains or loses steps/triggers/actions.

The harness pins the transition commit; after the old structs are deleted
it reduces to a marshal-fixed-point round-trip guard (the 5b pattern) that
stays forever as the 5c writer's safety net.

## Drift-gate and content cleanup

- Clear the 17 quest-artifact baseline entries; the baseline shrinks ~28 →
  ~11 and the gate keeps `new: 0`.
- Strip the 4 dead `linear:` lines from quest files.
- `triggers|*.QuestStep` baseline entries: the probe's recorded example
  paths locate whichever file authors `triggers:` inside a step (neither
  parser reads it — if the content is real, it's a broken quest to fix by
  moving the trigger top-level; if the entries are stale, they just clear).
  Current live-run evidence suggests stale, to be confirmed by the probe.

## Testing & gates

TDD throughout. The equivalence harness is the centerpiece; beyond it:
`(*Quest).Validate` unit tests per rule, engine-index tests proving trigger
evaluation output is unchanged for a representative quest (the existing
questengine test suite already covers evaluation — it must pass unmodified),
full suite + vet, drift gate run (`DOGMUD_BOOT_SMOKE=1`, expect `new: 0`
with the shrunken baseline), panic-mode boot with instance wipe. No content
playtest owed (no behavior change); no browser gate (no UI).

## Out of scope

The 5c editor itself (its spec slims down on top of this); any yaml library
change; trigger evaluation changes; the pinned general admin help page.
