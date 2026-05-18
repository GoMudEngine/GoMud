# Chunk 4d documentation + helpfile audit

Produced 2026-05-18 to feed Task 21 (helpfile updates) and Task 22
(context.md updates) of the chunk-4d Submission Rework plan.

Scope: identify helpfiles and context.md files that must add, update,
or cross-reference the new submission/surrender system introduced by
commits `b94ccf55` through `232b5091` (T1-T19).

Survey date: 2026-05-18. Branch: `feature/mob-aliveness-1.3-crimes`.

---

## Status table (chunk 4d tasks through T19)

| Task | Status | Commit |
|------|--------|--------|
| T1 — Position submission mapping | Done | `b94ccf55` |
| T2 — SubmissionPolicy + SurrenderPolicy | Done | `9607a51f` |
| T3 — Balance config knobs (7 new) | Done | `ce777d1a` |
| T4 — Sub roll + tier classification | Done | `85e20c0d` |
| T5 — Drift-roll snapshot | Done | `401f15c9` |
| T6 — Position_SubmissionTick observer | Done | `6d675c13` |
| T7 — Outcome resolver + policy matrix | Done | `9e7fa8f5` |
| T8 — NoDeprogression + GoldLossFraction | Done | `24d7d994` |
| T9 — Broken-limb buff (id 83) | Done | `94072e20` |
| T10 — Submission-stunned buff (id 84) | Done | `ac8266e1` |
| T11 — Submission attempt + resolution messaging | Done | `482baef2` |
| T12 — MobSpec policy fields + archetype defaults | Done | `17f19bf1` |
| T13 — Btree primitives | Done | `128fa048` |
| T14 — Selective mob YAML overrides | Done | `46cc251d` |
| T15 — set submission / set surrender commands | Done | `69350a86` |
| T16 — Status display additions | Done | `dbc21d7a` |
| T17 — Helpfiles (submission + surrender) | Done | `510d0225` |
| T18 — Sunset legacy submit command + helpers | Done | `4830f825` |
| T19 — Behavior Matrix PB-301..PB-341 | Done | `232b5091` |
| T20 — Doc audit (this file) | In progress | — |
| T21 — Helpfile updates | Pending | — |
| T22 — context.md sweep | Pending | — |
| T23 — Roadmap closeout | Pending | — |

---

## Files reviewed — summary table

### Helpfiles

| File | Keyword match | Verdict |
|------|--------------|---------|
| `submission.template` | entire file (T17) | KEEP-AS-IS |
| `surrender.template` | entire file (T17) | KEEP-AS-IS |
| `grapple.template` | See Also references `help submit` | UPDATE — broken cross-ref |
| `special.template` | lists `submit` as active command | UPDATE — DELETE stale entry |
| `combat.template` | grapple section, See Also | UPDATE — ADD submission links |
| `attack.template` | grapple section, See Also | UPDATE — ADD submission link |
| `unarmed-combat.template` | grapple Progression section | UPDATE — ADD submission note |
| `conditions.template` | stub only | UPDATE — ADD broken-limb entry |
| `status.template` | no submission content | UPDATE — ADD policy display note |
| `set.template` | no submission content | UPDATE — ADD set submission/surrender |
| `death.template` | no submission content | UPDATE — ADD NoDeprogression note |
| `reach.template` | grapple reach (T9 content) | KEEP-AS-IS |
| `kick.template` | grapple knee section | KEEP-AS-IS |
| `stand.template` | not grapple-specific | KEEP-AS-IS |
| `bash.template` | not grapple-specific | KEEP-AS-IS |
| `trip.template` | not grapple-specific | KEEP-AS-IS |
| `strength.template` | grapple reference only | KEEP-AS-IS |

### Context.md files

| File | Verdict | Scope |
|------|---------|-------|
| `internal/combat/context.md` | UPDATE | File-map table still lists `AttemptSubmission` (deleted in T18); add Submission System section |
| `internal/state/position/context.md` | UPDATE | "Next chunks: 4d" still shows as pending; update to shipped |
| `internal/characters/context.md` | UPDATE | Add SubmissionPolicy + SurrenderPolicy field docs |
| `internal/hooks/context.md` | UPDATE | Add Position_SubmissionTick to observer registry |
| `internal/state/life/context.md` | UPDATE | Add NoDeprogression + GoldLossFraction to DeadData struct doc |
| `internal/configs/context.md` | UPDATE | Add 6 chunk-4d balance knobs subsection (no entry yet) |
| `internal/behaviortree/context.md` | UPDATE | Add 3 chunk-4d btree primitives to position table |
| `internal/buffs/context.md` | UPDATE | Add broken-limb (id 83) + submission-stunned (id 84) entries |
| `docs/TEST_COVERAGE_AUDIT.md` | UPDATE | Remove stale AttemptSubmission / ApplySubmission* rows |

---

## Per-helpfile findings

### `submission.template` — KEEP-AS-IS

T17 authored this file correctly. Covers all four policies
(mercy/subdue/cripple/lethal), position-dependent submission types,
`set submission` usage, and See Also linking to surrender, grapple,
and reach. Nothing stale; no chunk-4d content is missing.

---

### `surrender.template` — KEEP-AS-IS

T17 authored this file correctly. Covers the realism caveat (the
tap is a signal, not a guarantee), the three surrender policies
(never/always/auto-tap-below), and `set surrender` usage.
Nothing stale.

---

### `grapple.template` — UPDATE

**Issue:** The See Also block (line 41) still reads:
```
  help trip</ansi>, <ansi fg="command">help submit</ansi>,
```
The `submit` command was deleted in T18. `submit.template` no longer
exists. This is a broken cross-reference — `help submit` will resolve
to nothing or a fallback at runtime.

**T21 action:** Replace `help submit` with `help submission` and add
`help surrender` while the block is open. Suggested replacement:

```
<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help attack</ansi>,
  <ansi fg="command">help kick</ansi>, <ansi fg="command">help trip</ansi>,
  <ansi fg="command">help submission</ansi>, <ansi fg="command">help surrender</ansi>,
  <ansi fg="command">help reach</ansi>
```

This is the highest-priority helpfile fix in T21 — a live broken link.

---

### `special.template` — UPDATE

**Issue:** Line 13 lists `submit` as a currently available special
move:
```
  submit    - Attempts to force a restrained opponent into submission.
```
The `submit` command was deleted in T18. Submissions are now
engine-fired automatically on the sub tick — no player command
triggers them.

**T21 action:** Delete the `submit` line. Optionally add a one-line
explanatory note so players who remember the old command aren't
confused. Suggested replacement block:

```
<ansi fg="yellow">Special moves that share this cooldown:</ansi>

  <ansi fg="command">bash</ansi>      - A powerful melee strike that can stagger your opponent.
  <ansi fg="command">kick</ansi>      - A fast kick aimed to disorient or knock back an enemy.
  <ansi fg="command">trip</ansi>      - Attempts to knock an opponent off their feet.
  <ansi fg="command">grapple</ansi>   - Grabs and restrains an opponent.
  <ansi fg="command">cast</ansi>      - Initiates spellcasting. Counts as a special move on
              initiation.

Submissions (chokes, joint locks) fire automatically when you have
dominant control in a grapple — no command needed. See
<ansi fg="command">help submission</ansi>.
```

---

### `combat.template` — UPDATE

**Why:** The Special Combat Moves section and See Also line have no
mention of the submission system. A player reading `help combat` for
a tactical overview after being subbed has nowhere to go.

**T21 action:** Add `help submission` and `help surrender` to the
See Also block, and add a one-line entry to the Special Combat Moves
section:

In the Special Combat Moves table, after `taunt`:
```
  <ansi fg="command">grapple</ansi> - Grabs and restrains an opponent.
```
After the existing list and the 5-round cooldown line, add:
```
Submissions (joint locks, chokes) fire automatically when
your control in a grapple is dominant. See
<ansi fg="command">help submission</ansi>.
```

In the See Also block, add after `help prone`:
```
  <ansi fg="command">help grapple</ansi>, <ansi fg="command">help submission</ansi>,
  <ansi fg="command">help surrender</ansi>
```

Keep additions brief — this is an overview file.

---

### `attack.template` — UPDATE

**Why:** The See Also already lists `help grapple` and `help reach`.
The submission system is the natural next step once a player is in a
grapple, but there's no pointer here.

**T21 action:** Add `help submission` to the See Also block. No body
changes needed — the Weapon Reach section already says "see help
grapple" and submission is one step further in that chain.

Suggested See Also update:
```
<ansi fg="yellow">See Also:</ansi>

  <ansi fg="command">help combat</ansi>, <ansi fg="command">help defense</ansi>,
  <ansi fg="command">help flee</ansi>, <ansi fg="command">help bash</ansi>
  <ansi fg="command">help trip</ansi>, <ansi fg="command">help kick</ansi>,
  <ansi fg="command">help prone</ansi>, <ansi fg="command">help stamina</ansi>
  <ansi fg="command">help grapple</ansi>, <ansi fg="command">help reach</ansi>,
  <ansi fg="command">help submission</ansi>
```

---

### `unarmed-combat.template` — UPDATE

**Why:** The Progression section lists `kick`, `trip`, and `grapple`
as skills that train unarmed-combat. Grappling is now the gateway to
the submission system. A pointer closes the discovery loop for
unarmed fighters (who are most likely to be grappling).

**T21 action:** Add one pointer bullet in the Progression section
after the grapple line:

```
  - Successful submissions earned from dominant grapple control
    also credit the skill. See <ansi fg="command">help submission</ansi>.
```

Alternatively, add `help submission` to the existing See Also block.
Either approach is sufficient — the See Also is cheaper.

---

### `conditions.template` — UPDATE

**Why:** The file is currently a stub (4 lines, no condition list).
Chunk 4d adds a new persistent debuff — broken-limb (id 83) — that
players will encounter after a cripple submission. When a player
types `conditions` and sees "broken arm," they have no way to learn
what it does or how long it lasts without this file.

**T21 action:** Expand the stub to at least mention the broken-limb
condition. Suggested minimal addition:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">conditions</ansi>

The <ansi fg="command">conditions</ansi> command lists current
afflictions and debuffs.

<ansi fg="yellow">━━━ Notable Conditions ━━━</ansi>

  <ansi fg="stat">Broken Limb:</ansi>
    A limb broken during a cripple submission. Reduces combat
    effectiveness for an extended period. Recovers over time.
    Cannot be removed early by normal healing.

  <ansi fg="stat">Prone:</ansi>    On the ground, face-down. See <ansi fg="command">help prone</ansi>.
  <ansi fg="stat">Stunned:</ansi>  Brief stagger (one round, clears automatically).

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help submission</ansi>,
  <ansi fg="command">help prone</ansi>
```

Note: expanding this file fully is 4f work; the minimal broken-limb
entry is the T21 scope.

---

### `status.template` — UPDATE

**Why:** T16 added submission + surrender policy display to the
`status` command output. The `status` helpfile has no mention of
these fields. A player who sees "Submission: subdue" in their status
sheet and types `help status` gets no explanation.

**T21 action:** Add a Grapple Policies subsection after the Resource
Pools section:

```
<ansi fg="yellow">Grapple Policies: </ansi>

  <ansi fg="command">Submission</ansi> - What you do when you lock a submission.
    (mercy / subdue / cripple / lethal)
    See <ansi fg="command">help submission</ansi>.

  <ansi fg="command">Surrender</ansi>  - Whether you tap out when caught in a sub.
    (never / always / auto-tap-below N%)
    See <ansi fg="command">help surrender</ansi>.
```

Also add `help submission` and `help surrender` to the Related
section at the bottom.

---

### `set.template` — UPDATE

**Why:** T15 added `set submission` and `set surrender` subcommands.
The `set` helpfile lists all settings but has no entry for these two.
A player typing `set submission` after reading the submission helpfile
would expect `help set` to list it.

**T21 action:** Add two entries to the Usage section. Suggested
placement after the `set wimpy` entry:

```
  <ansi fg="command">set submission &lt;mercy|subdue|cripple|lethal&gt;</ansi>
  Sets what happens when you lock a submission on an opponent.
  Type <ansi fg="command">set submission</ansi> alone to see current policy.
  See <ansi fg="command">help submission</ansi>.

  <ansi fg="command">set surrender &lt;never|always|auto-tap-below N&gt;</ansi>
  Sets whether you tap out when caught in a submission.
  Type <ansi fg="command">set surrender</ansi> alone to see current policy.
  See <ansi fg="command">help surrender</ansi>.
```

---

### `death.template` — UPDATE

**Why:** The current text says "One of your core attributes will
permanently weaken slightly." This is accurate for standard deaths.
But chunk 4d introduces NoDeprogression: subdue and cripple
submissions use the death cascade WITHOUT stat decay. A player who
gets subdued and wakes up at the temple without any stat loss and no
understanding of why will be confused (or incorrectly conclude the
death system is broken).

**T21 action:** Expand the Death Penalties section to note the
submission exception:

```
<ansi fg="yellow-bold">Death Penalties:</ansi>
  - One of your core attributes will permanently weaken slightly.
  - Skills you haven't practiced recently may grow rusty.
  - You'll be transported home with all pools restored.

<ansi fg="yellow-bold">Submission Exception:</ansi>
  If you were knocked out by a grapple submission (subdue or
  cripple), you wake at the temple with NO stat decay — your
  opponent chose not to kill you. A cripple submission leaves
  a broken limb that persists for around an hour of play.
  See <ansi fg="command">help submission</ansi>.
```

---

### `reach.template` — KEEP-AS-IS

T9 (chunk 4c) authored this file. No submission references are
needed here — reach and submissions are parallel grapple systems
that don't interact with each other's helpfiles. The existing
See Also (grapple, unarmed-combat, weapon-combat) is correct.

---

### `kick.template` — KEEP-AS-IS

The Knee section describes the grapple-control variant. Adding a
submission pointer here would be premature; the player discovers the
submission system via `help grapple` → `help submission`. Keep the
kick file focused on kick mechanics.

---

### `stand.template`, `bash.template`, `trip.template` — KEEP-AS-IS

None interact with the submission system. Keep.

---

### `strength.template` — KEEP-AS-IS

References grapples in the effects paragraph. Accurate and not
requiring a submission pointer — strength is the roll stat for
initiating a grapple, not for submission eligibility.

---

## Per-context.md findings

### `internal/combat/context.md` — UPDATE

**Issue 1 (stale file-map):** Line 817 of the file-map table still
lists `AttemptSubmission` in the `grapple.go` contents column:

```
| `combat/grapple.go` | `AttemptGrapple`, `ApplyGrappleResult`,
  `CheckClinchProgression`, `CheckGroundedEscape`,
  `ApplyPositionProgression`, `IsThirdPartyAttack`,
  `AttemptSubmission` |
```

`AttemptSubmission`, `ApplySubmissionSuccess`,
`ApplySubmissionFailure`, and `SubmissionResult` were ALL deleted in
T18. This entry is stale.

**T22 action (Issue 1):** Remove `AttemptSubmission` from the
grapple.go row. The four deleted symbols should not appear there.

**Issue 2 (no submission section):** Chunk 4d adds
`Position_SubmissionTick.go` (observer), `combat/submission.go`
(roll + tier + outcome resolver), and a policy matrix. None are
documented in this file. The file's chunk-4c section ("Weapon Reach
Utility") is the model.

**T22 action (Issue 2):** Add a new "Submission System (chunk 4d)"
section after the existing "Weapon Reach Utility (chunk 4c)" block.
Cover:

1. Overview: automatic, engine-fired per-round tick; no player
   command.
2. Key files:
   - `Position_SubmissionTick.go` — observer that fires once per
     grapple round; reads the drift-roll z-score snapshot on
     `Character.LastDriftRollZ`.
   - `combat/submission.go` — `RollSubmissionAttempt`, tier
     classification (bad / neutral / success / crit), and
     `ResolveSubmissionOutcome`.
3. Four outcome tiers:
   - **Bad** (z < SubBadZThreshold): attempter falls Prone, sub
     attempt backfires.
   - **Neutral**: no outcome; continue grapple.
   - **Success**: apply attempter's SubmissionPolicy.
   - **Crit** (z >= SubCritZThreshold): apply policy + apply
     submission-stunned buff (id 84) to defender.
4. Policy matrix: attempter's SubmissionPolicy × defender's
   SurrenderPolicy → final outcome. Key rule: only `mercy` policy
   honors the tap.
5. Life cascade flags: `NoDeprogression` (subdue/cripple skip stat
   decay), `GoldLossFraction` (fraction of gold transferred).
6. New buffs: broken-limb (id 83, cripple tier only), submission-
   stunned (id 84, crit tier, 1 round).
7. Balance knobs: see `internal/configs/context.md` chunk-4d table.

---

### `internal/state/position/context.md` — UPDATE

**Issue:** The "Next chunks" paragraph in the Status section (line
42) still reads:

```
Next chunks: 4d — submissions engine (chokes / joint-locks gated
by ControlLevel thresholds). 4e — player command parity ...
```

Chunk 4d has shipped. This language is stale.

**T22 action:** Rewrite the "Next chunks" entry for 4d to "4d shipped"
language, following the same pattern as the 4c entry above it:

```
**4d shipped:** Automatic submission system. Fires from
`Position_SubmissionTick.go` once per grapple round when the
drift-roll z-score exceeds `SubmissionAttemptAlpha`. Four-tier
resolution (bad / neutral / success / crit) consumes attempter's
`SubmissionPolicy` and defender's `SurrenderPolicy`. Life cascade
extended with `NoDeprogression` + `GoldLossFraction` flags. New
buffs: broken-limb (id 83) for cripple, submission-stunned (id 84)
for crit. See `internal/combat/context.md` for the integration.
```

Also update Intentional Simplification #7 from "No submission
system. Submissions ... chunk 4d adds the submission engine" to
"Submission system shipped in chunk 4d."

---

### `internal/characters/context.md` — UPDATE

**Why:** T2 added `SubmissionPolicy` and `SurrenderPolicy` as named
fields (and enums) on `Character`. T5 added `LastDriftRollZ` float64
(the snapshot the sub tick reads). None of these are documented in
this file.

**T22 action:** Add a "Chunk-4d submission fields" note to the
Character struct section. Near the existing combat-state fields
(position, grapple controller), add:

```
**Chunk-4d submission fields (T2, T5):**
- `SubmissionPolicy` — enum (mercy/subdue/cripple/lethal); default
  `subdue`. Set by `set submission` command. Applied by the sub tick
  when the attempter locks a submission.
- `SurrenderPolicy` — enum (never/always/auto-tap-below N); default
  `auto-tap-below 15`. Applied when defender is the controlled party.
- `LastDriftRollZ` — float64 snapshot of the most recent per-round
  grapple drift z-score. Read by `Position_SubmissionTick` to decide
  whether a submission window opens this round.
```

---

### `internal/hooks/context.md` — UPDATE

**Why:** The hooks file documents all active observers (Life_Cascades,
Position_Cascades, etc.). `Position_SubmissionTick.go` is a new
observer added in T6, not listed anywhere.

**T22 action:** Add `Position_SubmissionTick.go` to the observer
registry section. Suggested entry near the existing position
observers:

```
- **Position_SubmissionTick.go** — Fires each grapple round
  (registered on the same per-round tick as
  `Position_GrappleTick.go`). Reads `c.LastDriftRollZ` to check
  whether the drift margin exceeds `SubmissionAttemptAlpha`.
  Calls `combat.EvaluateSubAttempt` which returns a tier; delegates
  to `ResolveSubmissionOutcome` for policy application and messaging.
  Source: T6 (`6d675c13`).
```

---

### `internal/state/life/context.md` — UPDATE

**Why:** The `DeadData` struct documentation (lines 68-80 of the
file) does not mention the two chunk-4d fields added in T8.

**T22 action:** Update the DeadData struct documentation to add the
two new fields:

```go
type DeadData struct {
    Reason           state.TransitionReason
    Killer           state.ActorRef
    DamageMap        map[int]int
    NoDeprogression  bool    // T8: set for subdue/cripple submissions
    GoldLossFraction float64 // T8: fraction of gold to transfer
}
```

Add a note after the DamageMap paragraph:

```
**NoDeprogression** — When `true`, the standard attribute-decay
and skill-rust cascade steps are skipped. Set by the submission
outcome resolver for `subdue` and `cripple` policy outcomes.
This allows players to "die" without permanent character damage
when knocked out rather than killed.

**GoldLossFraction** — Fraction of the defender's carried gold
transferred to the attacker at resolution time. Set alongside
`NoDeprogression` by the submission resolver. Zero for `mercy`
and `lethal` policies (lethal uses normal loot). Default from
config: `SubGoldLossFraction` (0.20).
```

---

### `internal/configs/context.md` — UPDATE

**Why:** The configs context documents balance knobs by chunk. The
chunk-4c section (Weapon Reach) already exists and is the model.
No chunk-4d entry exists despite T3 adding 6 new knobs.

**T22 action:** Add a "Submission System (chunk 4d)" subsection
under the Balance section, after the chunk-4c table:

```
### Submission System (chunk 4d)

Six knobs control when submission windows open and how tiers are
resolved. All live under `Balance` in `_datafiles/config.yaml`.

| Knob | Default | Effect |
|------|---------|--------|
| `SubmissionAttemptAlpha` | 1.0 | Min drift-margin z-score (absolute) for a sub window to open on either side of the grapple. |
| `SubmissionAttemptCritZ` | 2.0 | Defender drift z >= this triggers a bottom-submission window regardless of margin. |
| `SubSkillWeight` | 1.0 | Unarmed-combat skill contribution multiplier in the sub roll. |
| `SubBadZThreshold` | -1.0 | Sub-roll z-score below which the bad tier fires (attempter falls Prone). |
| `SubCritZThreshold` | 2.0 | Sub-roll z-score at or above which the crit tier fires (defender stunned). |
| `SubGoldLossFraction` | 0.20 | Fraction of defender's carried gold transferred to attacker on subdue/cripple. |

See `internal/combat/context.md` "Submission System" for the full
resolution flow.
```

---

### `internal/behaviortree/context.md` — UPDATE

**Why:** T13 added 3 chunk-4d btree primitives. The position-
primitives table in this file lists the chunk-4a and chunk-4b sets
but has no chunk-4d entries.

**T22 action:** Add a chunk-4d row-group to the position-primitives
table (or a new table if preferred). Verify exact primitive names
against `internal/behaviortree/conditions_position.go` before
committing — the names below are from the T13 commit message:

```
**Submission primitives (3 — chunk 4d):**
| Primitive | Underlying check |
|-----------|------------------|
| `mob_can_attempt_sub` | `IsTopSubEligible(c)` — mob is in a top-submission-eligible position with sufficient control |
| `target_can_attempt_sub` | `IsBottomSubEligible(target)` — target is in a bottom-submission-eligible position |
| `mob_has_sub_window` | `c.LastDriftRollZ` >= `SubmissionAttemptAlpha` — sub window currently open |
```

Verify names against source before committing; the pattern from T13
may differ slightly in casing.

---

### `internal/buffs/context.md` — UPDATE

**Why:** T9 and T10 added two new buffs (broken-limb id 83,
submission-stunned id 84). The buffs context.md has no catalog
section listing DOGMud-specific buffs by ID.

**T22 action:** If the file has a buff-ID registry section, add both
entries. If it does not, add a short DOGMud-specific section:

```
## DOGMud chunk-4d buffs

| ID | Name | Duration | Source | Effect |
|----|------|----------|--------|--------|
| 83 | Broken Limb | ~1 hr play (~3600 rounds) | Cripple submission outcome | Reduces combat effectiveness; cannot be dispelled early |
| 84 | Submission Stunned | 1 round | Crit submission tier | Brief combat stagger; clears automatically next round |
```

---

### `docs/TEST_COVERAGE_AUDIT.md` — UPDATE

**Why:** Lines 264-267 of this file list `AttemptSubmission`,
`ApplySubmissionFailure`, and `ApplySubmissionSuccess` as functions
in `grapple.go` that lack test coverage. These functions were deleted
in T18. The rows are stale and will mislead future audit passes.

**T22 action:** Delete (or mark "DELETED T18") the three rows
referencing these symbols:
- `AttemptSubmission` (line 264)
- `ApplySubmissionFailure` (line 266)
- `ApplySubmissionSuccess` (line 267)

The replacement functions (`RollSubmissionAttempt`,
`ResolveSubmissionOutcome`) live in `combat/submission.go` and ARE
tested via the PB-301..PB-341 behavior matrix (T19). The audit could
optionally add a note to that effect.

---

## New documentation surface

### `help policies` — CONSIDER (low priority)

**Verdict:** Not required for T21. The two individual helpfiles
(`help submission`, `help surrender`) are already discoverable via
`help grapple`. A top-level `help policies` that links to both is
a nice-to-have if the policies surface expands (e.g., a future
`set aggression` or `set flee-threshold`), but it's not needed now.

**Track as 4f candidate:** If chunk 4e adds more player-configurable
combat policies, create a `policies.template` hub page at that time.

---

### Broken-limb visibility in `help conditions` — REQUIRED (T21)

As noted in the conditions.template finding above: the broken-limb
buff is a persistent, non-dispellable debuff that players will
encounter after a cripple submission. The `conditions` helpfile is
currently a 4-line stub. A minimal expansion is required in T21 so
players know what the buff is and that it recovers over time.

---

### `help grapple` — submission cross-link — REQUIRED (T21)

The broken `help submit` See Also in `grapple.template` is a live
bug (links to a deleted command). This is the highest-priority T21
fix. See the grapple.template findings above.

---

## Summary counts

| Category | Files reviewed | UPDATE | DELETE | KEEP-AS-IS |
|----------|---------------|--------|--------|------------|
| Helpfiles | 17 | 7 | 0 | 10 |
| Context.md files | 9 | 8 | 0 | 0 (but 1 has stale rows to remove) |

**Helpfiles T21 must touch:** 7
(`grapple.template`, `special.template`, `combat.template`,
`attack.template`, `unarmed-combat.template`,
`conditions.template`, `status.template`, `set.template`,
`death.template`)

Note: that is 9 files counted above but `attack.template` and
`unarmed-combat.template` have minimal changes (one line in See
Also); the substantive rewrites are grapple, special, combat,
conditions, status, set, and death — 7 files with meaningful work.

**Context.md files T22 must touch:** 8
(`internal/combat/context.md`, `internal/state/position/context.md`,
`internal/characters/context.md`, `internal/hooks/context.md`,
`internal/state/life/context.md`, `internal/configs/context.md`,
`internal/behaviortree/context.md`, `internal/buffs/context.md`)
Plus: `docs/TEST_COVERAGE_AUDIT.md` (3 stale rows to delete).

---

## Surprising findings

1. **`grapple.template` has a live broken link.** The `help submit`
   cross-reference in the See Also block points to a command and
   helpfile that were deleted in T18. This is a runtime issue, not
   just a documentation gap — if the engine does a strict helpfile
   lookup for `help submit`, it will 404. Highest-priority T21 fix.

2. **`special.template` actively misleads players.** It lists
   `submit` as a typed command players can use. Players who read
   this and type `submit` mid-grapple will get an error and have no
   idea that submissions are now automatic. Second-highest T21
   priority.

3. **`death.template` doesn't account for NoDeprogression.** Players
   subdued via a submission will wake at the temple with no stat
   decay and ~20% of their gold missing. The death helpfile will
   confuse them: they'll look up "why did I lose gold" and find
   nothing. The NoDeprogression + gold-loss explanation needs to live
   somewhere obvious. `death.template` is the right place.

4. **`conditions.template` is a 4-line stub.** The broken-limb buff
   from cripple submissions is the first persistent, non-dispellable
   combat debuff players will commonly encounter. Without a conditions
   reference, players have no way to learn its duration or that it
   recovers on its own.

5. **`set.template` missing two new subcommands.** `set submission`
   and `set surrender` were added in T15 and are referenced from the
   helpfiles that T17 authored — but the main `set` helpfile doesn't
   list them. Any player who types `help set` to discover available
   settings will miss both policies.

6. **Eight context.md files have zero chunk-4d content.** The
   submission system touches characters, life, hooks, combat,
   configs, buffs, behaviortree, and state/position — all with
   established context.md files — yet none document the new T1-T18
   additions. T22 has a full sweep ahead of it. The largest gap is
   `internal/combat/context.md`, where the file-map table actively
   lists three deleted functions.

7. **TEST_COVERAGE_AUDIT.md has stale rows.** The three deleted
   grapple helpers (`AttemptSubmission`, `ApplySubmissionSuccess`,
   `ApplySubmissionFailure`) appear in a "no test coverage" table
   with file locations that no longer exist. A future audit pass
   would chase these down unnecessarily. T22 should clean them up.

---

## Post-audit notes for T21 and T22 execution

**Voice conventions (match existing helpfiles):**
- No hard numbers in player-facing text for damage. Exception:
  `auto-tap-below N` is a user-configured percentage — showing N
  is correct because the player set it.
- 80-char line wrap throughout.
- Use `━━━` section headers (yellow), `<ansi fg="stat">` for stat
  names, `<ansi fg="command">` for commands. See `submission.template`
  (T17) as the voice reference for the new feature's register.
- Policy names (mercy/subdue/cripple/lethal, never/always) are
  command keywords — render them as `<ansi fg="command">`.

**For context.md updates (T22):** Use present-tense "fully shipped"
voice consistent with chunk-4c and chunk-4b patterns. Reference
commit SHAs where applicable. The DeadData struct update in
`internal/state/life/context.md` is the trickiest — verify
`NoDeprogression` and `GoldLossFraction` field names against
`internal/state/life/life.go` before committing.

**Priority order for T21:**
1. Fix `grapple.template` broken `help submit` link.
2. Remove `submit` from `special.template`.
3. Expand `conditions.template` with broken-limb entry.
4. Add `death.template` NoDeprogression exception note.
5. Add `set submission` / `set surrender` to `set.template`.
6. Add `status.template` policy display note.
7. Add submission links to `combat.template`, `attack.template`,
   `unarmed-combat.template` See Also sections (low effort, low
   urgency).
