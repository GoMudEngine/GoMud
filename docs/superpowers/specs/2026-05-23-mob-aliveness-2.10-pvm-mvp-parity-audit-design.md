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
