# Dialogue Editor (Admin Web-Building Sub-Project 5b) — Design

**Date:** 2026-07-27
**Status:** Approved design, pre-implementation
**Epic:** Admin web-building. Sub-project 5 was decomposed during
brainstorming: **5a NPC greetings** (shipped, merged `345ccb9e7`) fixed the
`DialogueFile` struct and closed the unknown-key gate gap; this is **5b**, the
editor built against that now-truthful struct. Quest and behavior-tree
editors (5c/5d) remain deferred.
**Predecessor specs:** `2026-07-23-mob-authoring-3-design.md`,
`2026-07-25-npc-greetings-5a-design.md`

## Goal

Author a mob's **dialogue** from the `/build` admin page — greetings,
keyword patterns, the conversation tree with its quest gates, and memory
config — without hand-editing YAML, and with every mechanical dialogue SOP
from CLAUDE.md enforced at save instead of remembered by a human.

Dialogue is the largest content type in the game (302 files, ~25k lines) and
the most footgun-dense: a single bare-scalar list field silently mutes an
entire NPC, a missing `-end` token re-offers a completed quest, a grant node
in the wrong file position is shadowed by a lore trigger. An editor that
writes structurally valid YAML eliminates the first class outright and turns
the rest into refusals and warnings.

Scope decisions (user-confirmed):

- **Launched from the mob editor** — a "Dialogue…" button on the mob form
  opens a dedicated full-panel editor for that mob. Dialogue is 1:1 with
  mobs; the Mobs tab's zone-filtered list is the entry point, and no second
  list is built.
- **Full parity, one build**: greetings, patterns, tree (root + variants +
  nodes), and memory in a single spec and build — no half-editor state where
  an NPC's patterns are editable and its tree is not.
- Tree nodes render as an **ordered, draggable list**, not a graph canvas —
  see §4.

## Background: shape and stats

Measured across the live world (302 files):

| section | files carrying it |
|---|---|
| `patterns:` | 298 |
| `tree:` | 286 |
| `greetings:` | 186 (live since 5a) |
| `memory:` | 106 |
| `defaultMood:` | 299 |

`Pattern` and `TreeNode` (`internal/dialogue/types.go`) share the quest-gate
field family: `questRequired`/`questExcluded` (string lists), `grantsQuest`,
`requiresItem`/`givesItem` (item ids), `questFlagRequired`/`questFlagExcluded`
(map[string]string), `setsQuestFlag`, `bumpsRep`, `givesGold`,
`masterworkRequired`, `moodChange`. Patterns add `keywords`/`moods`/
`responses`; tree nodes add `id`/`triggers`/`requires`/`unlocks`/`text`/
`hints`. `Tree.Root` carries text/hints plus quest-gated `Variants`.

### The loader contract the writer must honour

`dialogue.Load(mobId, zone)` (`loader.go:21`):

- Path: `dialogue/<zoneNameSanitize(zone)>/<mobId>.yaml`. The filename is the
  mob id and nothing else — a mismatch silently mutes the NPC (documented
  gotcha). The package has its **own private `zoneNameSanitize`**; the writer
  must call that same function, not a copy from another package, so writer
  and loader cannot diverge. (Precedent: the mobs↔rooms sanitizer-agreement
  test from the zone-rename work; a matching pin is cheap here.)
- `dialogueCache` keyed `"%d:%s"` (mobId:zone) — a save must **replace the
  cache entry in place** so live NPCs serve the edit immediately. No
  invalidation exists today.
- **`nilSentinel`**: when `Load` finds no file, it caches "no dialogue
  forever" under the same key. **Create must clear the sentinel** or a
  newly authored dialogue file is invisible until reboot. This is the
  writer-side gotcha most likely to burn someone; it gets its own test.
- On parse error the loader nil-sentinels the whole file — the "mute NPC"
  incident class. The editor writes marshalled-from-struct YAML, so this
  class cannot occur through the editor; the strict gate (5a) catches it
  arriving by hand.

`validateQuestExclusions` (runtime, warn-only) already exists in the loader;
§2 supersedes it with a save-time refusal, and it stays as the backstop for
hand-edited files.

## 1. Writer seam — `internal/dialogue/save.go` (net-new)

- `SaveDialogueFile(df DialogueFile) error` — validate (§2), marshal, write
  to the loader's exact path, update `dialogueCache`, clear `nilSentinel`.
- `CreateNewDialogueFile(mobId int, zone string) error` — refuse if a file
  exists; write a minimal skeleton (defaultMood neutral, one empty-keyword
  catch-all pattern); clear the sentinel.
- `DeleteDialogueFile(mobId int, zone string) error` — remove the file, drop
  the cache entry, **set** the sentinel (the mob now genuinely has no
  dialogue).
- `ValidateDialogueFile(df) (errors []string, warnings []string)` — the §2
  rules, separated so the GMCP layer can surface warnings without blocking.

Saves are canonicalizing: yaml.v2 re-marshal reflows prose scalars and
reorders nothing semantically. A mass round-trip test (§6) proves field-level
identity over all 302 live files — and per the room-468 lesson, nobody judges
canonicalization by reading a positional diff.

## 2. Save-time SOP enforcement

The point of the editor. Every mechanical rule from CLAUDE.md's dialogue
SOPs becomes machine-checked:

| rule (source) | enforcement |
|---|---|
| `grantsQuest` ⇒ `questExcluded` contains the granted token AND its quest's `-end` token (re-grant SOP) | **refuse** |
| quest-granting node/pattern ⇒ `quest` and `task` present in triggers/keywords (quest NPC SOP) | **refuse** |
| grant nodes FIRST under `tree.nodes` — matching is file-order; a lore trigger shadows a later gated grant | **refuse**, showing the offending order |
| no semicolons in `text:`/`hints:`/`responses:` (command separator) | **refuse** |
| quest tokens must be real quests; flag keys/values must match a quest's declared `flags:`; item ids must exist | **pickers** from real registries (`quests.GetAllQuests`, questengine `QuestDef.Flags` via `AllQuests()`, `items.GetItemSpec`) — untypeable, and re-validated server-side |
| `moods` / `moodChange` / `defaultMood` | picker from the engine's mood set + values observed in content |
| `expiryPeriod` set (SOP: only for genuinely timed quests) | **warn** |
| trigger word appears in no hint, no node text, and not the root (discoverability SOP: "undiscoverable triggers are broken triggers") | **warn, listing each undiscoverable trigger** |
| `requires` used where `questRequired` would do (memory-expiry brick risk) | **warn** |
| hints in narrator voice, NPC text first person | not machine-checkable — the hints field carries inline guidance text instead |

The discoverability warning is the sleeper: today that rule lives in a
human's memory and its violations surface in playtests weeks later.

Warnings return alongside success and render in the panel; errors block the
save naming the exact node/pattern.

## 3. GMCP protocol — `Build.Dialogue.*`

Admin-gated and MainWorker-routed exactly as the other editors, behind a
`dialogueDeps` seam (load / save / create / delete / enums accessors) so
handlers unit-test against fakes.

| verb | payload | reply |
|---|---|---|
| `Build.Dialogue.Get` | `{mobId, zone}` | `Build.Dialogue` (full file + enums) — distinguishes "no file" (offer Create) from file content |
| `Build.Dialogue.Update` | full file | `Build.Result` + `warnings []string` |
| `Build.Dialogue.Create` | `{mobId, zone}` | `Build.Result` + fresh detail |
| `Build.Dialogue.Delete` | `{mobId, zone}` | `Build.Result` (confirm client-side; no blocker scan — see §5) |

Enums: quest tokens (with titles), per-quest declared flags (key → allowed
values), mood list, and the item picker rides the existing `Build.Item.List`
prefetch from the spawn editor.

`BuildResult` gains `Warnings []string` — first editor to use it; harmless
for the others.

## 4. UI — dedicated panel from the mob form

A **Dialogue…** button on the mob editor (with an indicator whether the mob
has dialogue) swaps the panel to the dialogue editor for that mob; Close
returns to the mob form.

Sections, top to bottom:

1. **Identity & mood** — mobId/zone (read-only), defaultMood picker, memory
   expiryPeriod (with the SOP warning inline).
2. **Greetings** — list of `{text, moods}` rows (the 5a shape).
3. **Patterns** — collapsible rows summarised by keywords; per-row form with
   keywords/moods/responses lists and the quest-gate drawer.
4. **Tree** — root text/hints; quest-gated root variants; then the **node
   list: ordered and draggable**, each node a collapsible form (id, triggers,
   requires/unlocks, text, hints, quest-gate drawer).

The node list is deliberately a list, not a graph canvas: in this engine
**order is semantics** — matching walks `tree.nodes` in file order, which is
why grant nodes must come first. A graph view would hide the single most
important property of the data. `requires`/`unlocks` render as node-id
pickers scoped to this file, and unknown references refuse at save.

Quest-gate drawer (shared component between patterns, nodes, and root
variants): grantsQuest picker, questRequired/Excluded token pickers,
flag-gate pickers driven by the selected quest's declared flags, item
give/require pickers, bumpsRep, givesGold, masterworkRequired.

## 5. Delete semantics

No blocker scan. Dialogue is referenced *by* its mob — nothing else points
at a dialogue file — and the mob-delete scan (sub-project 3) already checks
the reverse direction. Deleting dialogue mutes the NPC and nothing else;
a client-side confirm naming that consequence is sufficient.

## 6. Testing & verification gate

**Unit** (`internal/dialogue`): one test per §2 refusal rule and per warning;
writer path/cache/sentinel behaviour — save updates cache, create clears the
sentinel (the test that fails if the writer forgets), delete sets it;
sanitizer agreement pin.

**Round-trip, all 302 live files**: load → `SaveDialogueFile` to a temp
mirror → reparse → `reflect.DeepEqual` on every section against the original
parse. Proves the writer is lossless over the entire live corpus before it
ever touches a real file.

**Handlers** (`modules/gmcp`, fake deps): get/update/create/delete round-trip;
non-admin silently refused; warnings surfaced on success.

**Gates**: strict dialogue sweep stays green (baseline still empty, greetings
count still ≥186); full suite; boot under `MapConsistencyEnforce: panic`.

**Browser gate (user)**: drive mob → Dialogue…, edit a pattern, reorder tree
nodes, attempt each refusal (semicolon, grant without end-token, grant node
out of order), see the discoverability warning fire, save, and `talk` to the
NPC in game to confirm the edit is live without a reboot.

**Content adversarial playtest: N/A** — this is tooling, not player-facing
content (precedent: spawn and zone editors). Content written *with* the
editor gets playtested per the normal content SOP when it ships.

## Out of scope

- Quest editor (5c) and behavior-tree editor (5d) — deferred.
- Authoring aids beyond validation (prose suggestions, voice linting).
- Changes to the mood system, memory engine, or matching semantics.
- Bulk operations across dialogue files.
- The in-game lazy-load behaviour — unchanged; the editor rides it.
