# Mob Aliveness 2.10 — PvM/MvP/PvP/MvM Parity Audit (Design)

**Status:** Approved (brainstorming) — ready for `writing-plans`
**Roadmap chunk:** 2.10 (Phase 2 — Tactical fill-in, closeout)
**Size:** M (likely two sessions)
**Branch:** `feature/mob-aliveness-2.10-parity-audit`

## Goal

Close out Phase 2 of the mob aliveness roadmap by walking every player and
mob command, classifying each by parity verdict, fixing concrete gaps that
are quick, surfacing the rest for triage, and bundling the known
Companion-Phase-5 mutation-actives-on-mobs gap.

This is the chunk that "catches verbs we didn't think to add" before Phase 3
(routine layer) and Phase 4 (strategic layer) start dispatching tactical
verbs from new contexts.

## Non-goals

- Forced parity for every player command. Many commands are correctly
  one-sided (bank, character, inbox, password, etc.) — orthogonality is a
  valid verdict.
- Companion-control player-facing UI for triggering specific mob abilities.
  Per `feedback_companion_autonomy`, companions are intentionally
  autonomous; the only player-facing control surface is the existing
  `companion <name> assist on/off` posture toggle. This is a permanent
  design decision, not a deferral.
- Live in-combat smoke testing. Following the chunk 2.9 precedent,
  in-game smoke testing is deferred to the user post-chunk.

## What's in scope

1. **Audit walk** of all non-admin, non-meta player commands (~95
   candidates after filtering) and all 64 mob commands, producing two
   classification tables in this chunk's followup-PR description and the
   review doc.
2. **Mutation\_\* actions-lift** — lift the six `mutation_*` player
   commands into the `internal/actions/` package, add symmetric mob
   wrappers, add one btree action `try_mutation_active`. Closes the
   Companion Phase 5 "mutations on mobs" half.
3. **Quick patches inline** — ≤30-LoC fixes for concrete gaps, one
   commit each.
4. **Deferred-gap review doc** — separate doc surfaced for the user to
   triage before any memory entries land.
5. **`selljunk` deletion** — phantom verb, zero callers, used as the
   case study for the new "delete divergent verb" verdict.

## Classification scheme

Each command audited gets exactly one of six verdicts:

| Verdict | Meaning | Action |
|---|---|---|
| **Equivalent** | The other side has a working counterpart (possibly under a different file name, e.g., `skill.cast` ↔ `cast`). | No code change. Verify by reading both. |
| **Orthogonal** | One-sided by design — including verbs whose only real use is driving player-progression mechanics (quests, character sheet, skills, etc.) even when a literal mob equivalent is technically wireable. | No code change. Note rationale in table. |
| **Never-relevant** | The other side has no concept the command would apply to (e.g., `pvp`, `alias`, `macros`). | No code change. |
| **Gap: patch inline** | Concrete missing verb; lift is ≤30 LoC net change, no new config knob, no new gameplay decision, no multi-subsystem ripple. | One `fix(parity): <verb> — <desc>` commit per gap. |
| **Gap: delete divergent verb** | The asymmetric side is dead code or earns its keep insufficiently. Verified by grepping for live callers in Go and YAML; if none, delete is the answer. | One `chore(parity): drop dead <verb>` commit per gap. `selljunk` is the case study. |
| **Gap: defer** | Lift is larger, ambiguous, or needs a design decision. | Add to the deferred-gap review doc, no code change in this chunk. |

**Out of audit scope** (skipped entirely, not classified):
- `admin.*` commands — operator tools, not character verbs.
- Meta shells (`default`, `noop`, `usercommands`, `mobcommands`,
  `enchant_slot`, `helpfile_completeness_test`).

Filtered surface: player ~95 candidates, mob ~62 candidates (excludes the
two meta shells).

## Mutation\_\* actions-lift

### Six commands to lift

| Player command | LoC | Effect summary | Targeting |
|---|---|---|---|
| `mutation_blinding_flash` | 83 | Blinds nearby mobs via opposed Wil + UnarmedCombat roll; self-blind aftermath | AoE on room mobs |
| `mutation_blinding_spit` | 114 | Single-target blind via opposed roll | Single target |
| `mutation_healing_gel` | 64 | Self-heal (small, scales off skill) | Self |
| `mutation_pacifism_aura` | 70 | Adds Pacified condition to mobs that fail to resist | AoE on room mobs |
| `mutation_sonic_shout` | 105 | Damage + Stunned condition | AoE on room mobs |
| `mutation_toxic_bite` | 169 | Damage + Poisoned condition | Single target |
| **Total** | **605** | | |

### Package layout

```
internal/actions/
  mutation_blinding_flash.go      ~80 lines
  mutation_blinding_spit.go       ~110 lines
  mutation_healing_gel.go         ~60 lines
  mutation_pacifism_aura.go       ~70 lines
  mutation_sonic_shout.go         ~100 lines
  mutation_toxic_bite.go          ~160 lines
  mutation_helpers.go             ~60 lines
  mutation_blinding_flash_test.go
  mutation_blinding_spit_test.go
  mutation_healing_gel_test.go
  mutation_pacifism_aura_test.go
  mutation_sonic_shout_test.go
  mutation_toxic_bite_test.go
```

`mutation_helpers.go` extracts the five-step preamble shared by five of the
six (mutation-presence check → in-combat gate → cooldown try → stamina gate
→ score calc). Healing-gel skips the in-combat gate; helpers expose a
preamble variant for that. Pattern follows the existing
`internal/actions/combat_helpers.go`.

### Per-mutation function signature

```go
func TriggerBlindingFlash(actor Actor, opts MutationOpts) MutationResult
```

Where:

```go
type MutationOpts struct {
    TargetActor Actor // nil for self / AoE mutations
    // per-mutation knobs only added if a real caller needs them; YAGNI by default
}

type MutationResult struct {
    Triggered     bool
    BlockReason   string  // "no-mutation", "not-in-combat", "on-cooldown",
                          // "low-stamina", "no-target" — empty when Triggered
    AffectedCount int     // number of mobs/targets affected (AoE)
    // No []string Messages: the action fires user/room messages directly
    // via the Actor interface, matching the established pattern in
    // forage.go / salvage.go / cast.go. Wrappers do not re-format text.
}
```

### Wrappers

- `internal/usercommands/mutation_*.go` — collapse to ~20 lines each.
  Total: ~480 LoC deleted from `usercommands/`.
- `internal/mobcommands/mutation_*.go` — new ~20-line wrappers. Total:
  ~120 LoC added to `mobcommands/`. Registered in `mobCommands` map.
- Both wrappers do exactly: parse `rest`, build `MutationOpts`, call
  `actions.TriggerXxx(actor.From(self), opts)`, return result.

### Robustness contract

To prevent player↔mob drift on future mutation-mechanic changes, the
contract between `actions/` and wrappers is explicit:

**The `actions.TriggerXxx` function owns ALL of:**
- Mutation-presence check (`mutations.HasMutation(...)`)
- In-combat gate (where applicable)
- Cooldown try (`special-move` bucket, shared with other special moves)
- Stamina gate + consumption
- Attacker/defender score calculations
- Effect application (condition adds, damage, healing)
- AoE iteration
- Player/room message emission
- Skill-use event emission

**The wrapper owns ONLY:**
- Parsing `rest` (none of the six commands currently take args; this is
  forward-compat slack)
- Building `MutationOpts`
- Calling the action
- Translating `BlockReason` into the perspective-specific terminal text
  if the action's default doesn't fit (it should, in v1)

**Wrappers never reimplement any of the action's logic.** Tests live next
to the action, not the wrapper. Wrapper tests are minimal smoke (verifies
the call routes through; one assertion per wrapper).

### Btree integration

New btree action: `try_mutation_active`. Two argument forms:

```yaml
# Single explicit key — preferred
- type: try_mutation_active
  key: blinding-flash

# Ordered preference list — first available wins
- type: try_mutation_active
  keys: [healing-gel, blinding-flash, sonic-shout]
```

Per-call dispatch: for each candidate key in order:
1. Mob has the mutation? (skip if not)
2. Mob's special-move cooldown free? (skip if not)
3. Mob has enough stamina? (skip if not)
4. Fire `actions.TriggerXxx`, return `Success`.

If no candidate fires, return `Failure` so the parent selector can try the
next branch.

**Validation:** btree loader rejects any `try_mutation_active` node with
neither `key` nor `keys` set. Reason: implicit "use any mutation the mob
has" creates non-deterministic behavior tied to Go map iteration order.
Forcing explicit enumeration makes priority an author decision, not an
accident. This rejection is documented in the btree action's helpfile.

### Known limitation: runtime-evolved mutations don't auto-flow into btrees

If a mob (including a companion) evolves a new active mutation at runtime
that isn't listed in its archetype's `try_mutation_active` nodes, that
mutation will never fire in combat. This is not a chunk-2.10 fix; it
becomes a deferred-followup memory entry. Future options for that
followup (sketched in the memory entry, not committed here):

- Btree loader auto-augments `try_mutation_active` nodes from the mob
  template's `mutations:` field at load time (won't catch runtime
  evolution).
- New `try_any_active_mutation` action enumerates the mob's *current*
  mutations at tick time, in a deterministic order (rarity-descending,
  evolution-order, or author-tagged priority).
- Mutation-grant code writes evolved keys into a mob-scoped MiscData list
  the btree action reads.

## What this chunk does NOT touch

- Mutation acquisition / scoring / rarity (chunk 2.5 territory).
- The `incorporeal` mutation (chunk 2.2a) — no active command, only
  passive scaling.
- Cooldown sharing semantics — the `special-move` cooldown bucket stays
  shared across all special moves per-character. Mobs and players use
  the same bucket scheme.

## Quick-patch convention

A patch qualifies as "quick" (inline-fix-this-chunk) when **all** of:

- ≤30 LoC net change
- No new config knob
- No new gameplay decision
- No multi-subsystem ripple

Anything else → deferred-gap review doc.

**Examples to ground the rule:**

| Example gap | Verdict | Reason |
|---|---|---|
| Mob wrapper that calls an existing actions function | Quick | One small file, mechanical |
| Missing helpfile section | Quick | Doc-only |
| Renaming a divergent file name for consistency, no behavior change | Quick | Mechanical |
| Wiring a new btree primitive (e.g., `try_throw`) | Defer | Touches btree engine, needs args design |
| Bulk-sell command for players (mob has `selljunk`, but mob `selljunk` itself is dead code) | Delete | Drop the mob side; no player-side wire needed |
| Active-command crafting audit (cross-cutting `IsCrafting` issue) | Defer | Cross-cutting, has its own memory entry already |

## Commit shape

Single feature branch `feature/mob-aliveness-2.10-parity-audit`. Commits:

| Step | Commit prefix | Notes |
|---|---|---|
| Audit walk | (no commits — only spec doc updates) | Classification tables fill in via subagent runs |
| Mutation\_\* actions-lift | `refactor(actions): lift mutation_* commands into actions package` | One commit for the lift |
| Mutation\_\* mob wrappers + btree action | `feat(mobcommands): mutation_* mob wrappers` + `feat(btree): try_mutation_active action` | Two commits |
| `selljunk` deletion | `chore(parity): drop dead selljunk mob command` | One commit (includes test deletion) |
| Quick patches | `fix(parity): <verb> — <one-line desc>` | One commit per gap, atomic for clean revert |
| Deferred-list review doc | `docs(2.10): deferred parity gaps for review` | One commit; doc kept as historical record |
| Post-review memory writes | `chore(memory): log 2.10 deferred parity gaps` | Batch commit after user triages |
| Roadmap update | `docs(2.10): mark mob-aliveness 2.10 Done` | Final commit |

Git history reads as a chronological inventory of what shifted; any single
parity fix can be reverted cleanly if smoke surfaces a regression.

## Deferred-gap review workflow

After the audit + quick-patch + mutation\_\* lift land, a review doc is
written to:

```
docs/superpowers/specs/2026-05-23-mob-aliveness-2.10-deferred-gaps-review.md
```

Per-entry template:

```markdown
### <verb-name>
- **Direction:** mob-side missing | player-side missing | both-sides need design
- **Surface:** what the command does today on the present side
- **Why deferred:** ≤30 LoC didn't fit / needs new config / needs gameplay decision / ambiguous
- **Sketch of fix:** 2-3 sentence proposal
- **Proposed verdict:** patch-as-followup-chunk | memory-entry-only | wontfix | drop-the-divergent-side | needs-your-call
- **Estimated size:** S / M / L
```

**Handoff:** I surface the doc as a single message asking for one of
`{accept-proposed-verdict, change-verdict, drop-entirely, fix-now-anyway}`
per entry. Inline annotations or in-thread responses both work.

**Per-verdict actions after triage:**

| User verdict | Action |
|---|---|
| `accept-proposed-verdict` | Carry through the proposed verdict |
| `change-verdict` | Adjust per user feedback |
| `drop-entirely` | No memory entry, no code change |
| `fix-now-anyway` | Pull back into this chunk; add the appropriate commit |
| `patch-as-followup-chunk` | New `project_*.md` memory file + MEMORY.md table row |
| `memory-entry-only` | Memory file only, no roadmap entry |
| `wontfix` | No memory entry; rationale stays in the review doc |
| `drop-the-divergent-side` | Triggers one more `chore(parity): drop dead <verb>` commit in this chunk |

The review doc itself stays committed under `specs/` as the historical
record of what was triaged.

## Testing

| Surface | Test location | What it covers |
|---|---|---|
| `actions.TriggerXxx` × 6 mutations | `internal/actions/mutation_*_test.go` | Presence gate, in-combat gate, cooldown, stamina, scoring, effect application, AoE iteration, skill-use emission. Mirrors `forage_test.go` / `salvage_test.go` shape. |
| Player wrapper `usercommands/mutation_*.go` | Inline smoke if not already present | Minimal — verify call routes through |
| Mob wrapper `mobcommands/mutation_*.go` | New `mobcommands/mutation_*_test.go` smoke | Minimal — verify mobcommand dispatcher fires |
| Btree `try_mutation_active` action | New `internal/behaviortree/actions_mutation_test.go` | Single-key path, ordered-keys path, load-time rejection of no-key node, Success/Failure semantics |
| Audit classification table | Spec doc only | The table itself is the deliverable |
| Quick patches | Per-patch, when behavior changes | No test for pure renames or deletions |
| `selljunk` deletion | `mobcommands_test.go:TestSelljunk` deleted alongside implementation | No replacement |

**Manual smoke test (user-driven, post-chunk):** spawn a test mob with
`mutations: { blinding-flash: 1 }`, give its btree a node
`try_mutation_active: blinding-flash`, attack it, verify the flash fires
and blinds the player. Patch notes will include this checklist.

**Not tested (deferred):**
- Live in-combat mutation firing on a real server (manual smoke,
  per chunk 2.9 precedent).
- Multi-mob AoE rendering verification (chunk-6 messaging plumbing).
- Btree authoring patterns for which mob should get which mutation in
  its archetype (content-pass territory).

## Memory entries this chunk produces

Created during brainstorming (before chunk starts):
- `feedback_companion_autonomy.md` — design rule that companions are
  intentionally autonomous, never orderable. Linked from MEMORY.md.
- `project_dismiss_companion_gear_helpfile.md` — followup to update
  `help dismiss` warning about gear loss. Linked from MEMORY.md.

To be created during/after chunk (after user triages review doc):
- `project_mutation_active_runtime_evolution_btree.md` — runtime-evolved
  mutations don't auto-flow into btree dispatch. Three sketched fix
  paths documented.
- One `project_*.md` per deferred-gap that the user verdicted as
  `patch-as-followup-chunk` or `memory-entry-only`.
- Any updates to MEMORY.md needed to mark 2.10 done and adjust Companion
  Phase 5's status (close the mutations-on-mobs half, leave only the
  permanently-wontfix UI half if anything remains).

## Open questions

None at spec time — all clarifications captured in conversation, scope
locked.

## References

- Roadmap: `MOB_ALIVENESS_ROADMAP.md` chunk 2.10
- Precedent specs: `docs/superpowers/specs/2026-05-22-mob-aliveness-2.9-mob-forage-salvage-design.md`,
  `docs/superpowers/specs/completed/2026-04-04-mob-player-parity-design.md`
- Combat-quadrant parity log: `project_pvm_mvp_parity_gaps.md` (largely
  closed by April 2026 combat unification)
- Active-command crafting audit (deferred sibling):
  `project_active_command_crafting_audit.md`
- Companion autonomy design rule: `feedback_companion_autonomy.md`
- Actions-lift precedent: chunk 2.1 (`actions.Buy`), chunk 2.9
  (`actions.Forage`, `actions.Salvage`)
- Actor abstraction: `internal/actions/actor.go`,
  `internal/actions/actor_user.go`, `internal/actions/actor_mob.go`

## Player-side audit table

Walked the ~95 non-admin, non-meta player commands in
`internal/usercommands/`. The classification scheme is documented in
the spec's "Classification scheme" section. Counterpart lookup used
both literal name matches and purpose-based cross-reference against
`internal/mobcommands/`.

| Command | Mob counterpart | Verdict | Notes / file refs |
|---------|----------------|---------|-------------------|
| `afk` | (none) | Never-relevant | Session-presence state only; mobs are always "present" |
| `alias` | (none) | Never-relevant | Client input substitution; mobs parse command strings directly |
| `appraise` | (none) | Orthogonal | Player-merchant UX convenience; mobs set prices, don't query them |
| `ask` | `converse` (loose) | Orthogonal | Mob dialogue acquisition is not a design goal; `ask` is player-side dialogue driver |
| `assess` | (none) | Orthogonal | Necromantic corpse evaluation to choose `raise` spell; mob necromancers use scripted cast sequences |
| `assist` | (none) | Orthogonal | Party-join-combat shortcut; mobs use `lookforaid` / `lookfortrouble` / btree `join_combat` equivalents |
| `attack` | `attack` | Equivalent | Both sides resolve via `actions.FindAttackTarget` then `Character.SetAggro`; `mobcommands/attack.go` ↔ `usercommands/attack.go` |
| `bank` | (none) | Never-relevant | Player account management; mobs have no bank account |
| `bash` | `bash` | Equivalent | Both sides call `actions.ExecuteBash`; `mobcommands/bash.go` ↔ `usercommands/bash.go` |
| `biome` | (none) | Never-relevant | Player UI info display; mobs don't inspect biome data |
| `break` | `break` | Equivalent | Both call `Character.EndAggro()`; `mobcommands/break.go` ↔ `usercommands/break.go` |
| `broadcast` | `broadcast` | Equivalent | Both emit global chat; `mobcommands/broadcast.go` ↔ `usercommands/broadcast.go` |
| `bug` | (none) | Never-relevant | Player bug reporting to flat file; mobs can't file bugs |
| `buy` | `buy` | Equivalent | Both route through `actions.Buy`; `mobcommands/buy.go` ↔ `usercommands/buy.go` |
| `cancel` | `cancel` | Equivalent | Both abort casting/crafting/salvage activities; `mobcommands/cancel.go` ↔ `usercommands/cancel.go` |
| `character` | (none) | Never-relevant | Player character sheet display; mobs have no UI |
| `companion` | (none) | Never-relevant | Player companion roster / posture-toggle UI; mobs are companions, they don't manage them |
| `conditions` | (none) | Orthogonal | Player buff display; mobs query buffs via code, no command needed |
| `consider` | `consider` | Equivalent | Both route through `actions.Consider`; `mobcommands/consider.go` ↔ `usercommands/consider.go` |
| `cooldowns` | (none) | Orthogonal | Player cooldown display; mobs check cooldowns in code via `Cooldowns.Try()` |
| `craft` | `craft` | Equivalent | Both create items from recipes; `mobcommands/craft.go` ↔ `usercommands/craft.go` |
| `deletecharacter` | (none) | Never-relevant | Account deletion flow; mobs use `despawn` / `DestroyInstance` |
| `dismiss` | (none) | Orthogonal | Player severs companion bond; there is no mob-dismissing-a-companion design goal |
| `drink` | `drink` | Equivalent | Both consume potions from backpack; `mobcommands/drink.go` ↔ `usercommands/drink.go` |
| `drop` | `drop` | Equivalent | Both drop items to room floor; `mobcommands/drop.go` ↔ `usercommands/drop.go` |
| `eat` | `eat` | Equivalent | Both consume food items from backpack; `mobcommands/eat.go` ↔ `usercommands/eat.go` |
| `emote` | `emote` | Equivalent | Both emit free-form emote text; `mobcommands/emote.go` ↔ `usercommands/emote.go` |
| `equip` | `equip` | Equivalent | Both wield/wear items; `mobcommands/equip.go` ↔ `usercommands/equip.go` |
| `exits` | (none) | Orthogonal | Player room-exit display; mobs navigate by `pathto` and internal graph |
| `flee` | `flee` | Equivalent | Both disengage and run; `mobcommands/flee.go` ↔ `usercommands/flee.go` |
| `gearup` | `gearup` | Equivalent | Both equip best items from backpack; `mobcommands/gearup.go` ↔ `usercommands/gearup.go` |
| `get` | `get` | Equivalent | Both pick items from room; `mobcommands/get.go` ↔ `usercommands/get.go` |
| `give` | `give` | Equivalent | Both transfer items to targets; `mobcommands/give.go` ↔ `usercommands/give.go` |
| `go` | `go` | Equivalent | Both move between rooms via exits; `mobcommands/go.go` ↔ `usercommands/go.go` |
| `grapple` | `grapple` | Equivalent | Both initiate grapple; `mobcommands/grapple.go` ↔ `usercommands/grapple.go` |
| `help` | (none) | Never-relevant | Player help-file browser; mobs don't read help |
| `hint` | (none) | Never-relevant | Player quest-hint display; mobs don't track quests from the player side |
| `history` | (none) | Never-relevant | Player combat-log display; mobs have no session history UI |
| `inbox` | (none) | Never-relevant | Player message inbox; mobs have no inbox |
| `inventory` | (none) | Orthogonal | Player inventory display; mobs query inventory in code, no display needed |
| `keyring` | (none) | Never-relevant | Player key/lock tracking display; mobs don't maintain a keyring |
| `kick` | `kick` | Equivalent | Both call `actions.ExecuteKick`; `mobcommands/kick.go` ↔ `usercommands/kick.go` |
| `killstats` | (none) | Never-relevant | Player kill-count display; mobs don't inspect their own kill history |
| `list` | (none) | Orthogonal | Player shop-listing UI; mobs ARE the shop; merchant behavior is server-side |
| `lock` | (none) | Gap: defer | Mobs can pick locks (`defuse`, `picklock` via skullduggery) but can't lock containers/exits. Mob locking would require btree authoring context. Lift is non-trivial: needs room-exit lock state mutation + mob permission model. |
| `look` | `look` | Equivalent | Both inspect rooms/targets; `mobcommands/look.go` ↔ `usercommands/look.go` |
| `macros` | (none) | Never-relevant | Player macro display; mobs don't use macros |
| `motd` | (none) | Never-relevant | Server MOTD display; player-only |
| `mutation_blinding_flash` | (none yet) | Gap: patch inline | → Stage B lift; six `mutation_*` commands lifted into `actions/` package and mob wrappers added |
| `mutation_blinding_spit` | (none yet) | Gap: patch inline | → Stage B lift |
| `mutation_healing_gel` | (none yet) | Gap: patch inline | → Stage B lift |
| `mutation_pacifism_aura` | (none yet) | Gap: patch inline | → Stage B lift |
| `mutation_sonic_shout` | (none yet) | Gap: patch inline | → Stage B lift |
| `mutation_toxic_bite` | (none yet) | Gap: patch inline | → Stage B lift |
| `mutations` | (none) | Orthogonal | Player mutation-roster display; mobs query mutations in code via `mutations.HasMutation()` |
| `offer` | (none) | Orthogonal | Player pre-sell price-check UI (mob says offer price but doesn't transfer); mobs are the price-givers, not price-askers |
| `online` | (none) | Never-relevant | Server player-list display; player-only meta |
| `party` | (none) | Orthogonal | Player party management UI (invite, kick, leave, list); mobs join parties implicitly as charmed companions, no command surface needed |
| `password` | (none) | Never-relevant | Account credential management; mobs have no passwords |
| `pet` | (none) | Orthogonal | Player affection gesture toward a pet; mobs don't pet other mobs as a command |
| `picklock` | (none) | Gap: defer | Mobs with skullduggery could plausibly pick locks to pursue fleeing players. Lift needs btree action + skill-rank check + room-exit lock mutation. Non-trivial (>30 LoC, needs design decision on which mobs get this). |
| `print` | (none) | Never-relevant | Admin/debug text-echo tool; player UI utility with no mob analog |
| `printline` | (none) | Never-relevant | Admin/debug line-ruler tool (companion to `print`); no mob analog |
| `put` | `put` | Equivalent | Both place items into containers; `mobcommands/put.go` ↔ `usercommands/put.go` |
| `pvp` | (none) | Never-relevant | Player PvP flag display; mobs have no PvP concept |
| `quests` | (none) | Never-relevant | Player quest-log display; mobs don't hold quests (they grant them) |
| `quit` | (none) | Never-relevant | Player session exit; mobs use `despawn` |
| `rally` | `rally` | Equivalent | Both call `actions.ExecuteRally`; mob rally is self-only buff variant; `mobcommands/rally.go` ↔ `usercommands/rally.go` |
| `read` | (none) | Orthogonal | Reads item lore text from player's backpack; mobs don't read item descriptions |
| `remove` | `remove` | Equivalent | Both unequip items; `mobcommands/remove.go` ↔ `usercommands/remove.go` |
| `renameself` | (none) | Never-relevant | Account name-change flow; mobs are named at spawn from template |
| `reply` | (none) | Never-relevant | Player-to-player whisper reply; mobs use `sayto` / `say` |
| `report` | (none) | Orthogonal | Reports player's own HP/SP/CP to party; mobs communicate state internally, no chat-report needed |
| `salvage` | `salvage` | Equivalent | Both call `actions.Salvage`; `mobcommands/salvage.go` ↔ `usercommands/salvage.go` |
| `save` | (none) | Never-relevant | Player character persistence trigger; mobs auto-save via instance saves |
| `say` | `say` | Equivalent | Both emit room speech; `mobcommands/say.go` ↔ `usercommands/say.go` |
| `scan` | `scan` | Equivalent | Both call `actions.Scan`; `mobcommands/scan.go` ↔ `usercommands/scan.go` |
| `sell` | (none) | Orthogonal | Player sells items to merchant; mobs use `selljunk` (being deleted in Stage C) or internal gold-add; no mob-to-shop sale command is a design goal |
| `set` | (none) | Never-relevant | Player client-preference settings (ansi, linewidth, etc.); mobs have no client preferences |
| `setdesc` | (none) | Never-relevant | Player self-description edit; mobs have static descriptions in YAML |
| `sethome` | (none) | Never-relevant | Player home-room preference; mobs use `homeRoomId` field |
| `share` | (none) | Orthogonal | Player splits gold with party; mobs don't initiate gold sharing |
| `shoot` | `shoot` | Equivalent | Both call `actions.ExecuteShoot`; `mobcommands/shoot.go` ↔ `usercommands/shoot.go` |
| `shout` | `shout` | Equivalent | Both emit zone-wide speech; `mobcommands/shout.go` ↔ `usercommands/shout.go` |
| `show` | `show` | Equivalent | Both display an item to a target; `mobcommands/show.go` ↔ `usercommands/show.go` |
| `skill.cast` | `cast` | Equivalent | Both call spell resolution hooks; `mobcommands/cast.go` ↔ `usercommands/skill.cast.go` |
| `skill.disenchant` | (none) | Orthogonal | Removes Chrysalis enchantment at enchanting circle; requires player enchantment ownership model; no mob-enchantment-stripping design goal |
| `skill.forage` | `forage` | Equivalent | Both call `actions.Forage`; `mobcommands/forage.go` ↔ `usercommands/skill.forage.go` |
| `skill.map` | (none) | Never-relevant | Renders ASCII map to player terminal; mobs navigate via pathfinding graph, not map rendering |
| `skill.search` | `search` | Equivalent | Both call `actions.Search`; `mobcommands/search.go` ↔ `usercommands/skill.search.go` |
| `skill.skullduggery.defuse` | `defuse` | Equivalent | Both route through `actions.Defuse`; `mobcommands/defuse.go` ↔ `usercommands/skill.skullduggery.defuse.go` |
| `skill.skullduggery.plant` | `plant` | Equivalent | Both plant items on targets/containers; `mobcommands/plant.go` ↔ `usercommands/skill.skullduggery.plant.go` |
| `skill.skullduggery.shadow` | `shadow` | Equivalent | Both follow a target while hidden; `mobcommands/shadow.go` ↔ `usercommands/skill.skullduggery.shadow.go` |
| `skill.skullduggery.sneak` | `sneak` | Equivalent | Both enter hidden state; `mobcommands/sneak.go` ↔ `usercommands/skill.skullduggery.sneak.go` |
| `skill.skullduggery.steal` | `steal` | Equivalent | Both route through `actions.Steal`; `mobcommands/steal.go` ↔ `usercommands/skill.skullduggery.steal.go` |
| `skill.track` | `track` | Equivalent | Both call `actions.Track`; `mobcommands/track.go` ↔ `usercommands/skill.track.go` |
| `skills` | (none) | Orthogonal | Player skill-rank display; mobs query skill ranks in code |
| `sort` | (none) | Orthogonal | Player sorts component bag / bandolier contents; mobs have no bandolier/bag slot UI |
| `spells` | (none) | Orthogonal | Player spell-list display; mobs pick spells via btree/cooldown logic |
| `stand` | (auto) | Orthogonal | Mobs recover from prone automatically each round via `Character.AttemptRecovery()` in `NewRound_MobRoundTick.go:152`; no explicit mob `stand` command needed |
| `start` | (none) | Never-relevant | New-character creation and tutorial flow; mobs are spawned, not created via a tutorial |
| `stash` | (none) | Orthogonal | Player plants an item in the room disguised as a non-item; thematic player-only deception tool; no mob-stashing design goal |
| `status` | (none) | Never-relevant | Player stat-sheet display; mobs have no UI terminal |
| `storage` | (none) | Never-relevant | Player storage-room item deposit/withdraw; mobs have no storage accounts |
| `suggest` | (none) | Never-relevant | Player suggestion filing to flat file; mobs can't make suggestions |
| `suicide` | `suicide` | Equivalent | Both trigger self-death path; `mobcommands/suicide.go` ↔ `usercommands/suicide.go` |
| `surprise_attack` | `attack` (mob path) | Equivalent | Both sides fire per-weapon surprise attack from hidden state when special-move cooldown is available. Mob path via mobcommands/attack.go:64 + hooks/Awareness_Cascades.go. Player path via usercommands/attack.go's executeSurpriseAttack helper. Unification refactor opportunity tracked separately (see deferred-gaps review). |
| `talk` | `converse` (loose) | Orthogonal | `talk` initiates a dialogue session with an NPC; mobs initiate speech via `converse` / `say` / btree events. Mob-to-player dialogue origination is already handled by those commands; no `talk` mob command needed. |
| `target` | (none) | Orthogonal | Player switches combat target mid-fight; mobs select targets via `lookfortrouble` / `SetAggro` in btree/AI. No explicit `target` command needed: mob target-switching happens in code, not via a command. |
| `taunt` | `taunt` | Equivalent | Both call `actions.ExecuteTaunt`; `mobcommands/taunt.go` ↔ `usercommands/taunt.go` |
| `throw` | (none) | Gap: defer | Throws grenade/throwable items at room hostiles; mobs have no grenade-holding or throwable-item-selection model. Lift needs inventory scan for `Throwable` subtype + AoE resolution + skill check. Non-trivial; needs design decision on which mobs get this. |
| `title` | (none) | Never-relevant | Player-computed title display (skill tier + mutation tier); mobs have no player-facing title UI |
| `trip` | `trip` | Equivalent | Both call `actions.ExecuteTrip`; `mobcommands/trip.go` ↔ `usercommands/trip.go` |
| `unlock` | (none) | Gap: defer | Players unlock exits/containers with keys; mobs currently only interact with locks via `defuse` (trap-disarm) and `picklock` (not yet a mob command). Mob key-carrying + unlock is a more involved design decision. |
| `use` | (none) | Orthogonal | Interacts with room containers / crafting stations; mobs use `craft` directly and don't need a meta `use` dispatcher |
| `warcry` | `warcry` | Equivalent | Both call `actions.ExecuteWarcry`; `mobcommands/warcry.go` ↔ `usercommands/warcry.go` |
| `whisper` | `sayto` (loose) | Orthogonal | `whisper` sends private player-to-player messages; mobs use `sayto` for directed speech to a player, which is emitted as room-visible by design. True private mob whisper is not a design goal. |
| `who` | (none) | Never-relevant | Player online-list display; mobs don't query who is online |
| `zombieact` | (none) | Never-relevant | Player zombie-state flavor emote (dead-man's-land stub); mob zombie behavior is handled by btree archetypes |

**Player-side audit summary:**
- Equivalent: 44 (`attack`, `bash`, `break`, `broadcast`, `buy`, `cancel`,
  `consider`, `craft`, `drink`, `drop`, `eat`, `emote`, `equip`, `flee`,
  `gearup`, `get`, `give`, `go`, `grapple`, `kick`, `look`, `put`,
  `rally`, `remove`, `salvage`, `say`, `scan`, `shoot`, `shout`, `show`,
  `skill.cast`, `skill.forage`, `skill.search`,
  `skill.skullduggery.defuse`, `skill.skullduggery.plant`,
  `skill.skullduggery.shadow`, `skill.skullduggery.sneak`,
  `skill.skullduggery.steal`, `skill.track`, `suicide`, `surprise_attack`,
  `taunt`, `trip`, `warcry`)
- Orthogonal: 28 (`appraise`, `ask`, `assess`, `assist`, `conditions`,
  `cooldowns`, `dismiss`, `exits`, `inventory`, `list`, `mutations`,
  `offer`, `party`, `pet`, `read`, `report`, `sell`, `share`,
  `skill.disenchant`, `skills`, `sort`, `spells`, `stand`, `stash`,
  `talk`, `target`, `use`, `whisper`)
- Never-relevant: 37 (`afk`, `alias`, `bank`, `biome`, `bug`,
  `character`, `companion`, `deletecharacter`, `help`, `hint`, `history`, `inbox`,
  `keyring`, `killstats`, `macros`, `motd`, `online`, `password`,
  `print`, `printline`, `pvp`, `quests`, `quit`, `renameself`, `reply`,
  `save`, `set`, `setdesc`, `sethome`, `skill.map`, `start`, `status`,
  `storage`, `suggest`, `title`, `who`, `zombieact`)
- Gap: patch inline: 6 (`mutation_blinding_flash`, `mutation_blinding_spit`,
  `mutation_healing_gel`, `mutation_pacifism_aura`, `mutation_sonic_shout`,
  `mutation_toxic_bite`) — all → Stage B lift
- Gap: delete divergent verb: 0 (player-side has no dead verbs; `selljunk`
  is mob-side only and handled in Stage C)
- Gap: defer: 4 (`lock`, `picklock`, `throw`, `unlock`)
- **Total: 119** (118 non-admin, non-meta player command files after exclusions; `print.go` defines
  both `print` and `printline`, giving 119 distinct command functions)

## Mob-side audit table

Walked the ~62 mob commands in `internal/mobcommands/`. The classification
scheme is documented in the spec's "Classification scheme" section.
Direction reversed from the player-side: "does the player side have a
counterpart, and if not, is that a gap?"

Note: `darkness.go` is a shared-helper file (exports `sendRoomText`,
`sendAudioRoomText`, `canSeeInDark`, `isExcludedId`) — not a command.
It is excluded from the audit table. `sayto.go` defines three functions:
`SayTo`, `SayToOnly`, and `ReplyTo`. All three are registered commands and
appear as separate rows.

| Command | Player counterpart | Verdict | Notes / file refs |
|---------|-------------------|---------|-------------------|
| `aid` | (none) | Orthogonal | Mob first-aid (`aidskill` spell) targets downed players; player has `assist` (join combat) but no explicit first-aid command. `Aid` is species-gated (`KnowsFirstAid`); player healing is handled by spells/potions, not a command. |
| `attack` | `attack` | Equivalent | Both resolve via `actions.FindAttackTarget` + `Character.SetAggro`; mob supports hidden-state surprise-attack bonus. `mobcommands/attack.go` ↔ `usercommands/attack.go` |
| `bash` | `bash` | Equivalent | Both call `actions.ExecuteBash`; mob wraps darkness-aware messaging. `mobcommands/bash.go` ↔ `usercommands/bash.go` |
| `befriend` | (none) | Orthogonal | Engine plumbing: charms a player as a companion. Player has `companion` UI (posture toggle, dismiss) but cannot initiate a charm — that is a mob-only action by design. Actively used: `divergences.go:124` notes it as mob-AI. |
| `bite` | (none) | Orthogonal | Vampire-exclusive natural attack: physical damage + life-drain via `actions.ExecuteBite`. No player counterpart; design intent is a mob-specific special tied to vampire species. Not a gap — no player character has fangs. |
| `break` | `break` | Equivalent | Both call `Character.EndAggro()` and emit a disengage message. `mobcommands/break.go` ↔ `usercommands/break.go` |
| `broadcast` | `broadcast` | Equivalent | Both emit global chat via `events.Broadcast`. `mobcommands/broadcast.go` ↔ `usercommands/broadcast.go` |
| `buy` | `buy` | Equivalent | Both route through `actions.Buy`. `mobcommands/buy.go` ↔ `usercommands/buy.go` |
| `callforhelp` | (none) | Orthogonal | Mob AI coordination verb: broadcasts a `heard_callforhelp` btree event to nearby same-routine mobs and physically moves allies to the caller's room. Actively used by lookout archetype (verified: `behaviortree/lookout_archetype_test.go`). No player equivalent is a design goal — player party is managed via `party` command. |
| `cancel` | `cancel` | Equivalent | Both abort casting/crafting/salvaging via `Activity.TransitionToFree`. `mobcommands/cancel.go` ↔ `usercommands/cancel.go` |
| `cast` | `skill.cast` | Equivalent | Both route through `actions.InitiateCast` + `Activity.TransitionToCasting`. `mobcommands/cast.go` ↔ `usercommands/skill.cast.go` |
| `charge` | (none) | Orthogonal | Boar/ram-species trip variant: same math as `trip` but distinct flavor text ("charges and slams"). Player `trip` is the species-agnostic equivalent; adding a player `charge` would be redundant. Species-gated by btree archetype dispatch. |
| `consider` | `consider` | Equivalent | Both route through `actions.Consider`. Mob `MobActor.SendText` is a no-op, so math runs silently. `mobcommands/consider.go` ↔ `usercommands/consider.go` |
| `consume` | (none) | Orthogonal | Mob eats a room corpse to gain a `ConditionRegen` buff. Players have no equivalent corpse-eating design goal; food/potions cover the regen niche for players. Flesh-golem variant has enhanced absorption. |
| `converse` | (none) | Orthogonal | Mob-to-mob dialogue origination via the `conversations` subsystem. Player dialogue is player-to-mob only via `ask`/`talk`; mob-to-mob scripted conversation is a world-flavor feature with no player counterpart. |
| `craft` | `craft` | Equivalent | Both route through `actions.InitiateCraft`. `mobcommands/craft.go` ↔ `usercommands/craft.go` |
| `defuse` | `skill.skullduggery.defuse` | Equivalent | Both route through `actions.Defuse`. `mobcommands/defuse.go` ↔ `usercommands/skill.skullduggery.defuse.go` |
| `despawn` | (none) | Orthogonal | Engine plumbing: destroys the mob instance, deletes instance save, cleans room spawn slot. No player equivalent concept (player `quit` / `deletecharacter` are entirely separate flows). |
| `drink` | `drink` | Equivalent | Both consume drinkable items from backpack, apply buff IDs, grapple-blocked. `mobcommands/drink.go` ↔ `usercommands/drink.go` |
| `drop` | `drop` | Equivalent | Both drop items and gold to room floor via `actions.DropItem` / `actions.FloorDropGold`. `mobcommands/drop.go` ↔ `usercommands/drop.go` |
| `eat` | `eat` | Equivalent | Both consume edible items from backpack, apply buff IDs, grapple-blocked. `mobcommands/eat.go` ↔ `usercommands/eat.go` |
| `emote` | `emote` | Equivalent | Both emit free-form emote text via `actions.Emote` + `actions.FormatEmoteText`. `mobcommands/emote.go` ↔ `usercommands/emote.go` |
| `equip` | `equip` | Equivalent | Both equip/wield items via `actions.EquipItem`; mob has same-item no-op guard. `mobcommands/equip.go` ↔ `usercommands/equip.go` |
| `flee` | `flee` | Equivalent | Both use `combat.ResolveFleeBlockers`; grapple-blocked, movement-buff-blocked; random exit selection. `mobcommands/flee.go` ↔ `usercommands/flee.go` |
| `forage` | `skill.forage` | Equivalent | Both route through `actions.Forage`. `mobcommands/forage.go` ↔ `usercommands/skill.forage.go` |
| `gearup` | `gearup` | Equivalent | Both equip best items from backpack via `itemvalue.IsUpgrade`; mob has charmed-mob displaced-item drop logic. `mobcommands/gearup.go` ↔ `usercommands/gearup.go` |
| `get` | `get` | Equivalent | Both pick items and gold from room via `actions.GetItemFromFloor` / `actions.GetGoldFromFloor`. `mobcommands/get.go` ↔ `usercommands/get.go` |
| `give` | `give` | Equivalent | Both transfer items/gold to targets via `actions.GiveItemToChar` / `actions.GiveGoldToChar`; mob supports mob-to-mob give. `mobcommands/give.go` ↔ `usercommands/give.go` |
| `givequest` | (none) | Orthogonal | Engine plumbing: enqueues a `events.Quest` token for target players. Quest granting is an NPC-to-player operation; players receive quests, not give them. |
| `go` | `go` | Equivalent | Both traverse room exits; mob additionally supports `go <roomId>` teleport and NPC party coupled-movement. `mobcommands/go.go` ↔ `usercommands/go.go` |
| `grapple` | `grapple` | Equivalent | Both route through `actions.ExecuteGrapple`. `mobcommands/grapple.go` ↔ `usercommands/grapple.go` |
| `hamstring` | (none) | Orthogonal | Wolf-species natural attack: physical damage + bleed condition via `actions.ExecuteHamstring`. Species-gated; no player character has a wolf's bite-and-rake anatomy. Not a gap — comparable to `bite`. |
| `howl` | `taunt` | Equivalent | Both call `actions.ExecuteTaunt`; `howl` is the wolf-flavored taunt reskin for mob use. `mobcommands/howl.go` ↔ `usercommands/taunt.go` (`taunt` mob command is the generic flavor; `howl` is wolf-specific — both are Equivalent). |
| `kick` | `kick` | Equivalent | Both call `actions.ExecuteKick` with stomp/knee/standard variant detection. `mobcommands/kick.go` ↔ `usercommands/kick.go` |
| `look` | `look` | Equivalent | Both inspect rooms, targets, items, nouns. Mob look suppresses output when sneaking; includes `lookRoom` helper for peering through exits. `mobcommands/look.go` ↔ `usercommands/look.go` |
| `lookforaid` | (none) | Orthogonal | Mob AI scan: charmed mob searches room for downed companions and issues `aid` commands. No player equivalent needed — players use `assist` / `aid` manually. |
| `lookfortrouble` | (none) | Orthogonal | Mob AI aggro-scan: evaluates all players/mobs in room against faction, hate, and auto-aggro rules, then issues `attack`. Core of the hostile mob loop; no player equivalent by design. |
| `pathto` | (none) | Orthogonal | Mob AI pathfinding: uses `mapper.GetPath` to compute a route and stamps `mob.Path`; the movement loop executes it. No player equivalent — players move manually via `go`. |
| `plant` | `skill.skullduggery.plant` | Equivalent | Both route through `actions.Plant`. `mobcommands/plant.go` ↔ `usercommands/skill.skullduggery.plant.go` |
| `portal` | (none) | Orthogonal | Mob-specific temporary-exit creation (`exit.TemporaryRoomExit`). Used by the loot goblin AI; the `portal loot` and `portal home` modes are mob-AI idioms with no player equivalent. Player `cast` can produce portal-like effects via spells, which is the player-side design. |
| `put` | `put` | Equivalent | Both place items into room containers. `mobcommands/put.go` ↔ `usercommands/put.go` |
| `rally` | `rally` | Equivalent | Both call `actions.ExecuteRally`; mob applies self-buff only (no ally fan-out). `mobcommands/rally.go` ↔ `usercommands/rally.go` |
| `remove` | `remove` | Equivalent | Both unequip items via `actions.RemoveEquipment`. `mobcommands/remove.go` ↔ `usercommands/remove.go` |
| `replyto` | `reply` (partial) | Orthogonal | Mob directed reply verb: sends `"<mob> replies to <player>"` with room broadcast. Player `reply` is a private whisper-reply UI mechanism. Purpose differs enough that no direct counterpart is a design goal; mob speech is always room-visible. |
| `salvage` | `salvage` | Equivalent | Both route through `actions.Salvage`. Mob wraps `Activity` transitions explicitly. `mobcommands/salvage.go` ↔ `usercommands/salvage.go` |
| `say` | `say` | Equivalent | Both route through `actions.Say` with darkness-aware audio dispatch. `mobcommands/say.go` ↔ `usercommands/say.go` |
| `sayto` | `whisper` (partial) | Orthogonal | Mob directed speech visible to the room; player `whisper` is private. Player side has no room-visible directed-speech command (just `say` which is undirected). Purposeful asymmetry: mob speech is always observable. No gap. |
| `saytoonly` | (none) | Orthogonal | Mob private-channel directed speech: sends only to the target player, no room broadcast. Used for dialogue-engine responses. No player counterpart by design — players have `whisper`/`reply` for private messages. Distinct from `sayto`. |
| `scan` | `scan` | Equivalent | Both route through `actions.Scan`. `mobcommands/scan.go` ↔ `usercommands/scan.go` |
| `search` | `skill.search` | Equivalent | Both route through `actions.Search`. `mobcommands/search.go` ↔ `usercommands/skill.search.go` |
| `selljunk` | (none) | Gap: delete divergent verb | Converts mob inventory items to 1 gold/item via a light-flash emote. Grep verification: no callers in `_datafiles/` YAML or `internal/` Go (excluding `mobcommands/selljunk.go` and `mobcommands/mobcommands.go` registration). Only surfaces are: `actions/divergences.go:141` (documentation comment), `mobcommands_test.go:1005` (unit test). Zero live behavioral callers. No mob behavior YAML calls `selljunk`. Stage C deletes this file + its test. |
| `shadow` | `skill.skullduggery.shadow` | Equivalent | Both route through `actions.Shadow`. `mobcommands/shadow.go` ↔ `usercommands/skill.skullduggery.shadow.go` |
| `shoot` | `shoot` | Equivalent | Both route through `actions.ExecuteShoot`. `mobcommands/shoot.go` ↔ `usercommands/shoot.go` |
| `shout` | `shout` | Equivalent | Both emit zone-speech with adjacent-room bleed. `mobcommands/shout.go` ↔ `usercommands/shout.go` |
| `show` | `show` | Equivalent | Both display an item from backpack to a target. `mobcommands/show.go` ↔ `usercommands/show.go` |
| `sneak` | `skill.skullduggery.sneak` | Equivalent | Both route through `actions.Sneak`. `mobcommands/sneak.go` ↔ `usercommands/skill.skullduggery.sneak.go` |
| `steal` | `skill.skullduggery.steal` | Equivalent | Both route through `actions.Steal`. `mobcommands/steal.go` ↔ `usercommands/skill.skullduggery.steal.go` |
| `suicide` | `suicide` | Equivalent | Both trigger the Life machine death path (`Character.Die`); mob supports `vanish` variant for admin-style forced removal. `mobcommands/suicide.go` ↔ `usercommands/suicide.go` |
| `taunt` | `taunt` | Equivalent | Both call `actions.ExecuteTaunt`; `taunt` is the generic-flavor mob taunt (used by golems, elementals). `mobcommands/taunt.go` ↔ `usercommands/taunt.go` |
| `track` | `skill.track` | Equivalent | Both route through `actions.Track`. `mobcommands/track.go` ↔ `usercommands/skill.track.go` |
| `trip` | `trip` | Equivalent | Both call `actions.ExecuteTrip`; mob supports tail-sweep variant when mob has the tail mutation. `mobcommands/trip.go` ↔ `usercommands/trip.go` |
| `wander` | (none) | Orthogonal | Mob AI roaming: picks a random adjacent room exit and issues `go`; respects MaxWander, zone restriction, pack-roaming logic. No player equivalent — players choose their own movement. Actively used across most mob YAML files. |
| `warcry` | `warcry` | Equivalent | Both call `actions.ExecuteWarcry`; mob applies self-buff only. `mobcommands/warcry.go` ↔ `usercommands/warcry.go` |

**Mob-side audit summary:**
- Equivalent: 44 (`attack`, `bash`, `break`, `broadcast`, `buy`, `cancel`,
  `cast`, `consider`, `craft`, `defuse`, `drink`, `drop`, `eat`, `emote`,
  `equip`, `flee`, `forage`, `gearup`, `get`, `give`, `go`, `grapple`,
  `howl`, `kick`, `look`, `plant`, `put`, `rally`, `remove`, `salvage`,
  `say`, `scan`, `search`, `shadow`, `shoot`, `shout`, `show`, `sneak`,
  `steal`, `suicide`, `taunt`, `track`, `trip`, `warcry`) — note: `howl`
  and `taunt` are both Equivalent; both call `actions.ExecuteTaunt` with
  different flavor text, and both map to the player-side `taunt` command
- Orthogonal: 18 (`aid`, `befriend`, `bite`, `callforhelp`, `charge`,
  `consume`, `converse`, `despawn`, `givequest`, `hamstring`, `lookforaid`,
  `lookfortrouble`, `pathto`, `portal`, `replyto`, `sayto`, `saytoonly`,
  `wander`)
- Never-relevant: 0 (mob commands have no session-management, UI-display,
  or account-management equivalents — those are all player-side
  Never-relevant entries)
- Gap: patch inline: 0 (no mob-side commands are missing a player
  counterpart that would be a quick lift; the mutation gaps flow the other
  direction — player-side missing mob-side — handled in Stage B)
- Gap: delete divergent verb: 1 (`selljunk` — zero live YAML/Go callers
  confirmed by grep; only `divergences.go` doc comment + unit test remain)
- Gap: defer: 0 (no mob-side commands surfaced a deferred gap in the
  player direction; all asymmetries are Orthogonal by design)
- **Total: 63** (62 command files; `sayto.go` exports three registered
  commands — `SayTo`, `SayToOnly`, `ReplyTo` — counted separately;
  `darkness.go` is a shared-helper file, excluded)
