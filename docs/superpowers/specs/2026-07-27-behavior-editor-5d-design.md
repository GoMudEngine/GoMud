# Behavior-Tree Editor (Admin Web-Building 5d) — Design

**Goal:** a sixth `/build` tab — Behaviors — giving admins structured editing
of all three behavior-tree families: the 26 shared **archetypes**
(edit / create / delete / apply-to-NPCs), **per-mob trees**, and **room
trees**, with a recursive node editor, registry-backed validation, and saves
that go live immediately by busting the engine's caches.

## System facts (verified)

- **Three file families**, one resolution order (helpers.go:151): a per-mob
  tree (`behaviors/<zone>/<mobId>-<convertedName>.yaml`) wins; else the mob's
  `behavior_archetype` names `behaviors/archetypes/<name>.yaml`; room trees
  live at `behaviors/rooms/<zone>/<roomId>.yaml`.
- **NodeDef is recursive**: `type` ∈ selector / sequence / condition /
  action / decorator; optional `event` gate; `check` / `do` name a registry
  entry; decorator `mod` ∈ cooldown / repeat / invert / random / delay with
  `child`; branch nodes carry `children`; everything else rides an INLINE
  params map (`yaml:",inline"`). Archetype files add `goal_weights` and
  `default_goals`.
- **Registries**: 67 conditions, 89 actions, registered at init. They declare
  NO param schemas — params are freeform per check/action.
- **Events are matched by raw string equality with no boot validation** — a
  typo'd `event:` is a silently dead node today. Observed vocabulary:
  mob_combat_round, mob_hurt, mob_idle, mob_die, packmate_hurt,
  heard_callforhelp, player_enter, player_give, player_attack_rejected (the
  authoritative list is assembled from the dispatch sites at implementation
  and pinned by an anti-drift test — the AddPeriod lesson).
- **The engine caches everything, including negatives**: trees/noTree (by
  mob id), roomTrees/noRoomTree, archetypes/noArchetype, plus per-archetype
  goal-weight/default-goal maps. engine.go carries a literal
  `TODO(hot-reload): bust cache on file change` — this editor IS that
  hot-reload; every save/create/delete must evict the positive entry AND the
  negative entry AND the goal maps, or edits silently do nothing until
  reboot.
- **Latent bug folded into scope:** per-mob tree filenames embed the mob's
  NAME. The mob editor's rename moves the mob file but not the behavior
  file, so a renamed mob's tree silently stops resolving (falls back to the
  archetype, no error). The mob-editor rename path learns to move the
  behavior file alongside.
- **Comment loss hits hardest here.** Behavior files are the most
  comment-dense content in the repo (design rationale lives in `#`
  comments), and a marshal-based writer drops them all. Compensations, both
  required: (1) new OPTIONAL `notes:` string on the file root and `note:` on
  NodeDef, surfaced as text fields — durable homes for rationale going
  forward; (2) when the raw on-disk file contains comment lines, the panel
  shows a standing warning on first open: "this file carries hand-written
  comments; the first editor save drops them — move anything worth keeping
  into notes fields first."

## Server side

### Writer + validation — `internal/behaviortree/save.go`

- `SaveArchetype(name string, file ArchetypeYAML) error`,
  `SaveMobTree(mobId int, file TreeYAML) error` (filename computed from the
  live mob template's name), `SaveRoomTree(roomId int, zone string, file
  TreeYAML) error` — each: validate, marshal, write, then **evict** the
  matching engine caches (positive + negative + goal maps).
- `CreateArchetype(name)` (skeleton: one selector with a commented… noted
  example node), `DeleteArchetype(name)`, `CreateMobTree(mobId,
  fromArchetype string)` (seeded as a COPY of the named archetype's tree —
  the natural authoring flow is "specialize this archetype"),
  `DeleteMobTree(mobId)` (mob falls back to its archetype), room-tree
  create/delete.
- Validation (pure func + injected registry checks, the 5b/5c pattern):
  parse structure via the loader's own node builder; `check`/`do`/`mod`
  must name real registry entries; `event` must be in the pinned
  vocabulary; decorator param sanity (cooldown/delay need a duration,
  repeat a count); branch nodes need children, leaves must not have them.
  Warnings: a node with an event no OTHER tree uses (probable typo that
  passed the vocabulary), an empty selector/sequence.

### Reference guard

Archetype delete refuses while referenced, verbatim: mob templates'
`behavior_archetype`, the mutation archetype-shift mapping
(archetype_shift.go's table — exact source verified at planning), and
`validateAutoAggroBehaviorGates` expectations (an auto-aggro mob whose only
gate lives in the archetype being deleted). Per-mob and room tree deletes
are guarded only by a confirm (their fallback semantics are safe).

### GMCP — `Build.Behavior.*` behind a `behaviorDeps` seam

`List` (three sections: archetypes with used-by counts, mob trees with mob
names, room trees with room titles), `Get` (parsed file + enums), `Update`,
`Create`, `Delete`. Enums: node types, decorator mods, the 67 condition +
89 action names, the pinned event vocabulary, archetype names (for
create-from), and goal-type names for the weights editor. MainWorker via
GMCPBuildOp behind requireAdmin; `BuildResult.Warnings` as before.

## Client side — `behaviors.js`, sixth mode

List panel: three collapsible sections (Archetypes / Mob trees / Room
trees), search, `+ New archetype`, `+ New mob tree` (mob picker +
seed-from-archetype select), `+ New room tree` (zone→room picker).

Detail panel:
- **The recursive node editor**: one collapsible row per node (the 5b/5c
  shell), summary line = `type · event · check/do`; body = type select
  (switching type reshapes the row), event select (vocabulary + "none"),
  check/do datalist pickers over the registries, `note:` text field,
  generic params as key/value rows (registries declare no schemas — the
  hint links each check/do to the behavior schema doc), decorator mod
  select + single child, branch children as an ORDERED nested list with
  add-child (type picker) and ↑/↓ — child order IS evaluation order for
  selectors/sequences and the UI says so.
- Archetype extras: `notes:`, goal-weights rows (goal type → multiplier),
  default-goals rows (type/priority/params).
- Apply-to-NPCs: archetype detail lists the mobs using it (jump links to
  the mob editor); the mob form's existing behavior_archetype dropdown
  gains an "Edit tree…" jump into this tab.
- Comment-loss warning banner when the raw file has `#` comments (server
  sends a `hasHandComments` flag on Get).

Wiring in build.html: sixth tab, routing, script tag; hard-refresh caveat
as always.

## Proofs & gates

- **Round-trip fixed point over every live behavior file** (26 archetypes +
  all per-mob and room trees): marshal is byte-stable on re-marshal, node
  counts survive, `check`/`do`/`event` sets identical. The 5b/5c pattern.
- TDD throughout: writer cache-contract tests (save evicts positive AND
  negative caches; delete restores the negative), validation refusal per
  rule, guard tests, fake `behaviorDeps` handler tests.
- Anti-drift test pinning the event vocabulary against the dispatch sites.
- Headless E2E (the quest_e2e.mjs plumbing): list → get archetype → save
  unchanged → edit a node → save → refusals (unknown do, bogus event,
  childless selector) → create/delete archetype with guard.
- Full suite + vet + drift gate + panic boot. Browser gate: edit an
  archetype node and watch a live mob's behavior change without reboot;
  create a per-mob tree from an archetype and verify it overrides; delete
  it and verify fallback; trip refusals; comment-loss banner on a
  comment-rich file; mob rename moves its behavior file.

## Out of scope

The pinned general admin help page (still after the epic — this editor's
help content lands there); a visual node-graph canvas (the recursive list
IS the tree); live tree-execution tracing/debugging (`questdebug`-style
tooling is its own project); editing `_datafiles` sample trees outside the
configured world.
