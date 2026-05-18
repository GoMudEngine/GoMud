# Combat State — Chunk 4d: Submission Rework Design

> **Side quest from mob aliveness chunk 2.7.** Sub-chunk 4d of the
> combat-state-machines redesign (master spec:
> `docs/superpowers/specs/2026-05-13-combat-state-machines-design.md`).
> Fourth of six sub-chunks in the rich-grapple expansion (master-spec
> section "3a. Position rich-grapple expansion"):
>
>   - 4a (shipped) — Position FSM scaffold.
>   - 4b (shipped) — Control-axis mechanics + cutover.
>   - 4c (shipped) — Position × Weapon Utility (reach model).
>   - **4d (this spec)** — Submission system rework. Per-round
>     opportunistic submission attempts gated on the chunk-4b control
>     drift roll. Position-driven submission types. Policy-driven
>     outcome resolution (no per-round prompts) reusing the existing
>     death cascade with a no-deprogression flag. Sunsets the legacy
>     `submit` command + `AttemptSubmission` / `ApplySubmissionSuccess`
>     / `ApplySubmissionFailure` helpers.
>   - 4e — Third-party interaction asymmetries.
>   - 4f — Balance pass + flavor text + full-stack smoke.
>
> **Aliveness paused for the duration** of chunks 1-6.

## Goal

Make grapple positions feel consequential by giving the controller side
real submission threats per round, and make those threats land via a
character-policy resolution model that survives DOGMud's 4-second
combat tempo without per-round prompts.

The current `submit` system is a single player-typed special-attack
that produces a binary success/fail roll, with a hardcoded "yield if
defender HP < 25%" branch and a fall-through "resist takes 2× Str
damage." NPCs never yield. It doesn't differentiate submission types,
doesn't leverage the 14-state Position machine for position-flavored
narrative, doesn't model permanent consequences (broken limbs,
chokeouts), and doesn't give either side meaningful agency around the
outcome.

4d replaces this with an opportunistic per-round model that piggybacks
on the chunk-4b control-axis tick:

1. When a controller wins their per-round drift roll by a margin
   exceeding an alpha threshold AND their current Position has
   associated submissions, a **submission attempt opportunity** opens
   for that round.
2. A **separate sub-roll** then determines what happens this round:
   - **Bad** (margin below failure threshold): defender exploits the
     overcommit and **escapes the grapple entirely** (pair breaks to
     Standing; controller falls Prone).
   - **Neutral** (failed sub but not catastrophically): controller
     keeps the position and ControlLevel, sub didn't lock in.
   - **Success** (sub locks): apply the controller's pre-set
     **submission policy** → outcome.
   - **Critical success** (z-score > crit threshold): sub locks AND
     defender is **stunned next round** (cannot attack / defend; gets
     hit for free).
3. Outcome resolution uses **policy-driven** decisions on both sides
   — no prompts. Controllers (player or mob) have a `submission
   policy` set ahead of time (`mercy` / `subdue` / `cripple` /
   `lethal`). Defenders have a `surrender policy` (`auto-tap-below
   <hp%>` / `never-tap` / `always-tap`). Realism framing: "your nature
   already decided what you'll do here; the system plays it out."
4. The two harshest outcomes (`subdue`, `cripple`) reuse the existing
   **death cascade** with a `NoDeprogression` flag — defender wakes up
   in the temple with no stat decay, possibly with a persistent
   broken-limb debuff (cripple). `lethal` is the full death path. `mercy`
   is a clean release with a brief recovery debuff.

End-state: submissions are an organic per-round threat that emerges
from the chunk-4b control drift, position type drives the submission
narrative (Crucifix→armbar, BackGround→RNC, Mount→americana, etc.),
and the consequences are real but proportional to the aggressor's
chosen brutality.

## Non-goals (4d)

- **Bottom-position submissions from Controlled side.** 4d models
  only the *controller* (InControl side) attempting submissions. Some
  positions (Guard, HalfGuard) have meaningful bottom-game submissions
  in BJJ; in the FSM model the Guard-bottom *is* the controller of
  that pair-state, so the formal model already covers this — the
  position-to-sub mapping table just needs to include bottom-Guard
  subs (triangle, armbar from guard, omoplata). No special "reverse
  attacker" plumbing.
- **Multi-attacker grapples / 2-on-1 submissions.** Master-spec
  out-of-scope.
- **Mid-round defender escape from a locked submission.** Once the
  sub locks (success roll), the policy resolves; defender can't
  scramble out. The "escape entirely" outcome only fires on a *bad*
  sub roll (defender uses the controller's overcommit).
- **Per-attempt UI prompts.** Explicit non-goal. The chunk-4b smoke
  surfaced that combat is too fast for per-round prompts; this chunk
  commits to the policy model.
- **PvP consent rules / arena duel mode.** PvP submission policy
  defaults are the same as PvE; arena / consent / tournament rules
  are a separate system (out of scope).
- **Surgical submission-type selection by the controller** (e.g.,
  "submit armbar" to pick one). Position drives the type. If the
  player wants a different sub, they change position first.
- **Healing the broken-limb debuff via spells or potions.** 4d
  introduces the debuff and the temple-heal interaction that clears
  it; spell-based heals are 4f flavor.
- **Submission damage to non-grapple targets** (e.g., joint locks
  applied from standing). Submissions require a ground grapple in
  4d.

## Architecture

4d adds one new observer file (`Position_SubmissionTick.go`), one new
policy substrate on Character (`SubmissionPolicy` / `SurrenderPolicy`
enum + fields), a new buff (broken-limb debuff), a `NoDeprogression`
flag on the death cascade, and a position → submission mapping table
in `internal/state/position/`. Sunsets the legacy `submit` command
and the `AttemptSubmission` / `ApplySubmissionSuccess` /
`ApplySubmissionFailure` helpers in `internal/combat/grapple.go`.

### Files

| File | Status | Responsibility |
|------|--------|----------------|
| `internal/state/position/submissions.go` | NEW | `SubmissionType` enum (Armbar / RNC / Triangle / Americana / Kimura / Omoplata / Anaconda); `PositionSubmissions(s State) []SubmissionType` mapping; `IsSubmissionEligible(s State, cl ControlLevel) bool` predicate |
| `internal/state/position/submissions_test.go` | NEW | Mapping + eligibility unit tests across all 14 states × all ControlLevels |
| `internal/characters/submission_policy.go` | NEW | `SubmissionPolicy` + `SurrenderPolicy` enums + `Character.SubPolicy` / `Character.SurrenderPolicy` field accessors + parse-from-string helpers for `set` command + default-by-archetype lookup |
| `internal/characters/submission_policy_test.go` | NEW | Enum parsing, default lookup, edge cases |
| `internal/hooks/Position_SubmissionTick.go` | NEW | Per-round observer: checks each controller for sub-attempt opportunity, rolls separate sub roll, branches on result tier, calls `ResolveSubmissionOutcome` |
| `internal/hooks/Position_SubmissionTick_test.go` | NEW | Integration tests for each tier (bad/neutral/success/crit) × each outcome policy |
| `internal/combat/submission.go` | NEW | `RollSubmissionAttempt(controller, defender) SubmissionAttemptResult`; `ResolveSubmissionOutcome(controller, defender, type, result, policies)` — produces messaging + applies outcome |
| `internal/combat/submission_test.go` | NEW | Roll tier boundaries, policy resolution matrix |
| `internal/buffs/buffs.go` | MODIFY | Register new buff "broken limb" (combat-stat penalty; persists across rest; cleared by temple heal interaction) |
| `_datafiles/world/dogmud/buffs/<id>-broken_limb.yaml` | NEW | Buff YAML with stat penalties + duration semantics |
| `internal/hooks/Life_Cascades.go` | MODIFY | Add `NoDeprogression bool` flag on `life.DeadData`; when set, `Death_PlayerCleanup` skips the stat-decay step |
| `internal/state/life/life.go` | MODIFY | Extend `DeadData` struct with `NoDeprogression bool` field |
| `internal/hooks/Death_PlayerCleanup.go` | MODIFY | Check the flag; skip deprogression when set |
| `internal/hooks/Death_PlayerCorpse.go` | MODIFY | When `NoDeprogression` set: skip full-corpse-loot path; transfer only the configured gold-loss fraction to the aggressor (no item drop) |
| `internal/behaviortree/conditions_submission.go` | NEW | New btree primitives: `mob_can_submit` (controller in sub-eligible pos with InControl), `mob_submission_policy_is <policy>` |
| `internal/behaviortree/conditions_submission_test.go` | NEW | Primitive tests |
| `internal/usercommands/submit.go` | DELETE | Legacy command sunset |
| `internal/mobcommands/submit.go` | DELETE | Mob equivalent sunset |
| `internal/combat/grapple.go` | MODIFY | DELETE `AttemptSubmission`, `ApplySubmissionSuccess`, `ApplySubmissionFailure`, `SubmissionResult` struct |
| `internal/usercommands/set.go` | MODIFY | Add `set submission <policy>` and `set surrender <policy>` subcommands; rendering for current policy display |
| `internal/configs/balance.go` | MODIFY | Add submission-tick config knobs (see Balance config section below) |
| `_datafiles/config.yaml` | MODIFY | Surface the new knobs |
| `_datafiles/world/dogmud/grapple-messages.yaml` | MODIFY | Add submission-attempt messages: opening / escape / neutral / success-by-outcome / crit |
| `_datafiles/world/dogmud/mobs/**/*.yaml` | MODIFY (selective) | Add `submission_policy:` / `surrender_policy:` fields to mob YAMLs where archetype default isn't right (bandits→subdue, predators→subdue, guards→subdue, civilian→mercy, bosses→cripple/lethal) |
| `internal/mobs/mobs.go` | MODIFY | `MobSpec.SubmissionPolicy` / `SurrenderPolicy` fields + default-by-archetype fallback in `newMobByIdInternal` |
| `_datafiles/world/dogmud/templates/help/submit.template` | DELETE | Legacy helpfile |
| `_datafiles/world/dogmud/templates/help/submission.template` | NEW | Replacement: explains the policy model + outcome ladder + position→sub mapping table |
| `_datafiles/world/dogmud/templates/help/surrender.template` | NEW | Defender-side policy explainer |

## Per-round submission tick mechanics

The submission tick is a per-round observer that runs **after** the
chunk-4b control drift tick, so it can read the result of the drift
roll for the current round (margin, z-score, ControlLevel transitions).

### Eligibility gate

For each grapple pair in the room, check the controller side:
1. Position must be sub-eligible (`IsSubmissionEligible(state, cl)`).
   See "Position → submission mapping" below.
2. ControlLevel must be `InControl` OR `LosingControl`. (At Neutral
   or below the controller doesn't have enough positional dominance
   to threaten a sub.)
3. This round's drift-roll margin must exceed
   `SubmissionAttemptAlpha` (config; default `1.0` — about a one-
   standard-deviation win). This is the "alpha threshold" the user
   specified: only strong drift rounds open submission opportunities.

If all three gates pass, fire the sub roll for this round.

### The separate submission roll

A fresh opposed roll, distinct from the chunk-4b drift roll:

```
attackerScore = controller.Strength.ValueAdj + controller.GetSkillLevel(UnarmedCombat) * SubSkillWeight
defenderScore = defender.Strength.ValueAdj + defender.Vitality.ValueAdj
                + defender.GetSkillLevel(UnarmedCombat) * SubSkillWeight
result, margin, atkRoll, defRoll := dice.OpposedRollStat(attackerScore, defenderScore)
```

The result is interpreted in tiers based on z-score and margin:

| Tier | Condition | Result |
|------|-----------|--------|
| **Bad** | `atkRoll.ZScore < SubBadZThreshold` (default `-1.0`) | Defender exploits overcommit → escape entirely. `TransitionPair → Standing`; controller falls `Prone`. (Mirrors `ApplySubmissionFailure` behavior, but now triggered by a per-round opportunistic attempt rather than a player-command.) |
| **Neutral** | `result == false` AND not Bad | Failed attempt; no consequence. Controller stays in position with current ControlLevel. (No damage, no position change.) |
| **Success** | `result == true` AND `atkRoll.ZScore < SubCritZThreshold` (default `2.0`) | Sub locks in. Resolve via policy matrix below. |
| **Critical** | `result == true` AND `atkRoll.ZScore >= SubCritZThreshold` | Sub locks AND defender is **stunned** next round: cannot make attacks, gets a defense penalty. Modeled as a 1-round `Stunned` buff. Then resolve via policy matrix (with a "crit" message flag for narration). |

### Per-round timing

```
Position_GrappleTick (chunk 4b)   → updates ControlLevel + maybe transitions position
Position_SubmissionTick (chunk 4d) → checks eligibility, rolls sub, resolves outcome
NewRound combat resolution         → existing per-swing damage / combat messages
```

The sub tick runs INSIDE the same round as the drift tick — so a
controller can drift toward Crucifix in round N, get drifted INTO
Crucifix in the same round's transition, and immediately have an
armbar window open. This is a feature, not a bug: positional
dominance pays off the round you achieve it.

### Balance knobs (new in `internal/configs/balance.go`)

| Knob | Default | What it controls |
|------|---------|------------------|
| `SubmissionAttemptAlpha` | `1.0` | Minimum drift-roll margin (in std devs) that opens a sub window |
| `SubSkillWeight` | `1.5` | Unarmed-combat skill contribution to the sub roll vs strength |
| `SubBadZThreshold` | `-1.0` | Z-score below which the bad-tier (escape) fires |
| `SubCritZThreshold` | `2.0` | Z-score at or above which the crit-tier (stun) fires |
| `SubGoldLossFraction` | `0.20` | Fraction of carried gold transferred on subdue/cripple |
| `BrokenLimbBuffDuration` | `0` | 0 = until cleared by temple heal action. Future-tuning: a long round count |

## Position → submission mapping

The canonical mapping in `internal/state/position/submissions.go`.
Position drives the submission type; sub type drives the narration
flavor and the broken-limb body-part. Multiple subs per position are
fine — the tick picks one (round-robin? random?) — but each is
locked to a body part for the cripple outcome.

| Position | Sub types | Cripple body part |
|----------|-----------|-------------------|
| Standing / Prone / Supine / Turtle | none | n/a (not sub-eligible) |
| Clinch | none | n/a (standing grapple — no subs in 4d; future: standing arm-drag / guillotine entry) |
| BackStanding | RNC (rear naked choke) | none (choke → unconscious, no limb damage) |
| Mount | Americana / Triangle / Armbar (rotating) | arm (or face/neck for Triangle = no body part, just unconscious) |
| SideControl | Kimura / Americana | arm (shoulder) |
| KneeOnBelly | Armbar | arm |
| NorthSouth | Kimura / Anaconda (choke) | arm OR none (Anaconda = unconscious) |
| Crucifix | Armbar | arm (locked) |
| BackGround | RNC | none (choke) |
| HalfGuard (controller from top) | Kimura | arm |
| Guard (controller from bottom) | Triangle / Armbar / Omoplata | arm / face-neck for Triangle = none |
| Turtle | none in 4d | n/a (Turtle is a defensive curl — back-take submissions from an attacker behind a turtled defender are 4e/4f future work; for now Turtle is not sub-eligible) |

**Default selection** when a position has multiple subs: round-robin
per controller (track last-attempted sub on `Character`). Avoids
hammering the same sub every round; gives a varied narrative.

**Sub types and outcome flavor:**

| Sub | Flavor verb | Outcome mapping |
|-----|-------------|-----------------|
| RNC | "wraps an arm around your neck..." | mercy → release ; subdue → unconscious ; cripple → unconscious (no broken limb — chokes don't break things, fall through to subdue) ; lethal → choke to death |
| Triangle | "wraps their legs around your neck and arm..." | same as RNC; choke variant |
| Anaconda | "rolls and wraps your neck..." | same as RNC |
| Armbar | "isolates and hyper-extends your arm..." | mercy → release ; subdue → unconscious from pain ; cripple → **broken arm** + unconscious ; lethal → broken arm + finishing damage |
| Kimura | "twists your shoulder into an unnatural angle..." | same as Armbar; broken shoulder |
| Americana | "cranks your shoulder backwards..." | same as Armbar; broken shoulder |
| Omoplata | "traps your shoulder with their leg..." | same as Armbar |

Choke variants always degrade `cripple` → `subdue` (you can't break a
limb with a choke). Joint-lock variants do full cripple.

## Controller policy + defender policy

### Controller (`SubmissionPolicy` enum)

| Policy | Behavior when a sub locks |
|--------|---------------------------|
| `mercy` | Release the grapple cleanly. Defender stays standing (or transitions to Standing if was on the ground). Defender takes a brief recovery debuff (~2 rounds of -10% stamina). No gold transfer. No persistent damage. |
| `subdue` (default for most player + neutral mob archetypes) | Choke / hold to unconscious. Defender enters the no-deprogression death path (see Outcome severity section). Aggressor takes `SubGoldLossFraction` of defender's gold. Defender wakes up at temple with no lasting injury (chokes leave you woozy, not broken). |
| `cripple` | Break the joint / take the limb (per the sub type → body part mapping above). Defender enters no-deprogression death path AND wakes up at temple with a persistent **broken limb** debuff that requires a temple-heal action to clear. Gold transfer same as subdue. |
| `lethal` | Apply continuous damage drain (HP loss per round) until defender hits 0 → full death path WITH deprogression + full corpse loot. The aggressor doesn't have to stay in the lock to maintain it; once `lethal` policy + successful sub, the engine queues a damage cascade over the next 2-3 rounds. (Allows the defender's allies to intervene during the locked finish. Implementation detail of how the cascade is queued — direct damage tick observer vs delayed death event — lives in the plan, not the spec.) |

### Defender (`SurrenderPolicy` enum)

| Policy | Behavior when a sub locks |
|--------|---------------------------|
| `auto-tap-below <hp%>` (default for players: `15%`) | Sends a tap signal when the sub locks if HP is below the threshold. Tap signal is honored ONLY by controllers with `mercy` policy. (Acknowledges the realism — you can ask for mercy, but you don't get to demand it.) |
| `never-tap` (default for hostile mobs) | Never sends a tap signal. Just takes whatever the controller delivers. |
| `always-tap` (default for civilian / cowardly archetypes) | Sends a tap signal as soon as a sub locks regardless of HP. Mercy controllers always release; other controllers ignore. |

### Player UX

```
set submission <mercy|subdue|cripple|lethal>
  - Shows current policy if no arg, sets if arg provided
  - Default for new characters: subdue
  - Lethal requires confirmation prompt the first time set
    ("Are you sure? Lethal submissions count as full kills with
    deprogression for the target if they're a player.")

set surrender <auto-tap-at <hp%>|never|always>
  - Defender-side
  - Default for new characters: auto-tap-at 15%
```

`submission` and `surrender` (no args) display the current policy.

### Mob policy storage

In `internal/mobs/mobs.go` add two `MobSpec` fields:

```go
SubmissionPolicy string `yaml:"submission_policy,omitempty"`
SurrenderPolicy  string `yaml:"surrender_policy,omitempty"`
```

If empty in YAML, fall through to the archetype default in
`characters.DefaultSubmissionPolicyForArchetype(archetype)`:

| Archetype | Default sub policy | Default surrender policy |
|-----------|--------------------|--------------------------|
| `predator` | `subdue` (choke + leave alive, the predator wants its meal) | `never-tap` |
| `bandit` (faction-thieves) | `subdue` (knock out + rob) | `never-tap` |
| `guard` / `city_watch` | `subdue` (jail-style respawn) | `never-tap` (guards don't surrender to outlaws) |
| `defensive_caster` | `mercy` (the spellcaster has bigger options — sub is a finisher of last resort) | `auto-tap-below 25%` |
| `tank_taunter` | `subdue` | `never-tap` |
| `leader` / `boss_*` | `cripple` (bosses do real damage) or `lethal` for the most dangerous | `never-tap` |
| `generic_fighter` | `subdue` | `auto-tap-below 10%` |
| `civilian` / `merchant` | `mercy` | `always-tap` |
| `lookout` / `ambusher` | `subdue` | `auto-tap-below 20%` |

Per-mob YAML overrides are simple: `submission_policy: lethal` on a
specific boss, `surrender_policy: always-tap` on a particular
cowardly NPC.

## Outcome severity ladder (death-pipeline reuse)

Critical design move: reuse the existing Life FSM `Alive → Dead → Respawning → Alive`
cascade for `subdue` / `cripple` / `lethal` outcomes. Add a single
`NoDeprogression bool` flag on `life.DeadData` that the
`Death_PlayerCleanup` cascade respects.

| Outcome | Triggers Life-Dead cascade | NoDeprogression flag | Gold loss | Corpse loot | Persistent debuff | Respawn location |
|---------|---------------------------|----------------------|-----------|-------------|-------------------|------------------|
| `mercy + tap honored` | NO | n/a | none | n/a | brief recovery debuff (~2 rounds, -10% stamina) | stays in room, position resets to Standing |
| `subdue` | YES | TRUE | `SubGoldLossFraction` (default 20%) → aggressor | NONE (defender drops no items, just the gold transfer) | none | temple (per existing respawn-room logic) |
| `cripple` | YES | TRUE | `SubGoldLossFraction` → aggressor | NONE | **broken limb** debuff (combat-stat penalty, persists until temple-healed) | temple |
| `lethal` | YES | FALSE | full (existing death loot path) | full (existing) | n/a (you're dead) | temple (or whatever the existing death respawn logic chooses) |

### Death cascade modifications

1. **`life.DeadData`** extends:
   ```go
   type DeadData struct {
       Killer            state.ActorRef
       DamageMap         map[int]int
       NoDeprogression   bool // chunk 4d: skip stat decay
       GoldLossFraction  float64 // chunk 4d: 0 = use existing logic, > 0 = transfer this fraction to Killer instead of full loot
   }
   ```

2. **`Death_PlayerCleanup.go`** wraps existing deprogression logic
   in a guard:
   ```go
   if !data.NoDeprogression {
       applyDeprogression(c)  // existing
   }
   ```

3. **`Death_PlayerCorpse.go`** wraps existing corpse-creation logic:
   ```go
   if data.GoldLossFraction > 0 {
       transferPartialGold(c, data.Killer, data.GoldLossFraction)
       // skip corpse creation
   } else {
       createFullCorpse(c)  // existing
   }
   ```

These two flag-guarded branches are the only Life cascade changes.
Everything else (respawn room selection, the resource reset, etc.)
runs unchanged.

## Broken-limb debuff

A new buff (registered in `internal/buffs/`) with:
- **Statmods**: -25% accuracy with the affected arm's weapon; can't
  wield two-handed weapons; -10% defense; -5% stamina max
- **Duration**: indefinite (cleared by a temple-heal interaction, not
  a round timer). New buff flag `requires_heal_action: true` so the
  existing buff-tick doesn't decrement / expire it.
- **Display**: prominent in `status` output ("Right arm: BROKEN
  (requires temple heal)")
- **Healing**: new admin / temple-priest interaction, modeled as a
  per-room dialogue tree on the temple priest mob. The smoke-tester
  can use an admin `mob heal-injury <inst>` shortcut to clear the
  debuff for testing.

Lifecycle: applied by `ResolveSubmissionOutcome` when policy ==
`cripple` AND sub type maps to a limb body part; persists across rest,
respawn (re-applied on the respawned character — the broken arm
survived the trip to temple), and combat; cleared by the heal
interaction.

## Btree primitives (submission tier)

New primitives for mob AI in `internal/behaviortree/conditions_submission.go`:

| Primitive | Type | What it checks |
|-----------|------|-----------------|
| `mob_can_submit` | condition | Mob is the controller of a sub-eligible grapple (position + ControlLevel match) |
| `mob_submission_policy_is <policy>` | condition | Mob's current `SubmissionPolicy` matches the named value; used in archetype branches that want to fork on policy |

These are deliberately minimal — mob submission attempts fire from
the `Position_SubmissionTick.go` observer (engine-driven, like the
chunk-4b drift tick), not from explicit btree action nodes. The
primitives exist so mob AI can BRANCH around the submission state
("don't trip if I'm controlling a grapple and about to sub" type
heuristics). New action primitives (`mob_attempt_submission`) are
not needed — the engine fires automatically.

## Cutover plan

The submission rework is a hard replacement, not a parallel-write
migration. The old `submit` command + helpers go away in a single
commit at the end of the chunk.

| Phase | Tasks |
|-------|-------|
| Foundation | T1 Position submission mapping + eligibility predicate. T2 Policy enums + Character fields + default lookup. T3 Balance knobs + config.yaml. |
| Mechanics | T4 Submission roll + tier resolution. T5 Per-round submission observer (Position_SubmissionTick). T6 Outcome resolver + policy matrix. T7 NoDeprogression / GoldLossFraction Life cascade modifications. |
| Effects | T8 Broken-limb buff + temple-heal interaction. T9 Stunned buff (1-round, for crit-tier). T10 Submission messages YAML. |
| Mob integration | T11 MobSpec fields + archetype defaults. T12 Btree primitives. T13 Selective mob YAML overrides (bosses, civilians). |
| Player UX | T14 `set submission` / `set surrender` commands. T15 Status display additions. T16 New helpfiles (submission, surrender). |
| Sunset | T17 DELETE `submit` command (user + mob), DELETE helpers in grapple.go (`AttemptSubmission`, `ApplySubmissionSuccess`, `ApplySubmissionFailure`, `SubmissionResult`), DELETE submit.template helpfile, fix compile errors. |
| Tests / docs | T18 Behavior Matrix PB-301..PB-340. T19 Doc audit + updates (T22-style). T20 Helpfile updates (T23-style). T21 Build/test/smoke. T22 Roadmap closeout. |

## Behavior matrix preview

(Drafted in plan, completed in tests during T18.)

| ID | Scenario | Expected |
|----|----------|----------|
| PB-301 | Standing — no sub eligibility | tick fires no sub roll |
| PB-302 | Mount + InControl + drift margin > alpha | sub roll fires |
| PB-303 | Mount + InControl + drift margin < alpha | no sub roll this round |
| PB-304 | Bad sub roll (z < -1.0) | controller falls Prone, pair breaks to Standing |
| PB-305 | Neutral sub roll | no change |
| PB-306 | Success sub roll, policy=mercy, defender taps | clean release, brief recovery debuff |
| PB-307 | Success sub roll, policy=mercy, defender never-tap | no-deprogression death path? OR no-op? — see Open Q3 |
| PB-308 | Success sub roll, policy=subdue, defender never-tap | no-deprogression death, 20% gold to aggressor, temple respawn, no debuff |
| PB-309 | Success sub roll, policy=cripple, sub=Armbar (Mount) | no-deprogression death + broken-arm debuff, gold transfer, temple respawn |
| PB-310 | Success sub roll, policy=cripple, sub=RNC (BackGround) | no-deprogression death (no debuff — chokes don't break) — degrades to subdue |
| PB-311 | Success sub roll, policy=lethal | full death path with deprogression, full corpse loot, temple respawn |
| PB-312 | Crit sub roll, policy=subdue | sub + 1-round Stunned buff on defender, then subdue outcome |
| PB-313 | Position rotation: 3 consecutive sub windows in Mount | round-robin sub types: Americana → Triangle → Armbar |
| PB-320 | Mob policy: predator default subdue | mob fires subdue automatically |
| PB-321 | Mob policy: boss override lethal | mob fires lethal automatically |
| PB-322 | Player set surrender always-tap, attacker mercy | every sub releases on tap |
| PB-323 | Player set surrender never-tap, attacker subdue | tap ignored, sub outcome fires |
| PB-330 | Broken-arm debuff applied, attack swing | -25% accuracy correctly applied |
| PB-331 | Broken-arm debuff, attempt 2H weapon wield | rejected with message |
| PB-332 | Broken-arm debuff cleared by temple heal | debuff removed |
| PB-340 | Legacy `submit` command typed (after sunset) | `unknown command` response |

## Sunset list (chunk 4d)

- `internal/usercommands/submit.go` — DELETE
- `internal/mobcommands/submit.go` — DELETE
- `internal/combat/grapple.go` — DELETE: `AttemptSubmission`,
  `ApplySubmissionSuccess`, `ApplySubmissionFailure`,
  `SubmissionResult` struct
- `_datafiles/world/dogmud/templates/help/submit.template` — DELETE
- `_datafiles/world/dogmud/templates/help/submit.md` (if exists) — DELETE
- `internal/usercommands/usercommands.go` — REMOVE `submit` entry from command registry
- `internal/mobcommands/mobcommands.go` — same
- `internal/configs/config.balance.go` — REMOVE old `SubmissionDuration` /
  `SubmissionResistDamageMult` if present and unused after T17

## Persistence

- `Character.SubmissionPolicy` / `Character.SurrenderPolicy` — serialized
  to YAML user files. Existing users get default values on next load
  (`subdue` / `auto-tap-below 15%`); not a breaking migration.
- `Character.LastSubmissionAttempted` (round-robin tracker) — runtime
  only (`yaml:"-"`), reset on respawn.
- `BrokenLimb` buff — serialized to user YAML, persists across logout/
  login like other buffs. Buff system already handles this.
- Mob policy YAML fields — runtime-loaded from MobSpec, not persisted
  per-instance.

## Open questions / risks

1. **Q1 — Submission tick ordering vs combat resolution.** The sub
   tick fires *between* the drift tick and the per-swing combat
   resolution. If a sub success removes the defender from combat
   (death cascade fires for subdue/cripple/lethal), the existing
   per-swing combat resolution will attempt to swing at a dead/gone
   target. Need to ensure the combat-round handler short-circuits
   when the target enters the Dead state mid-round (chunk-2 cascade
   probably already handles this — verify in T5).

2. **Q2 — Crit stun (Stunned buff) vs subdue/cripple/lethal.** If a
   crit fires AND the policy is subdue/cripple/lethal, the defender
   enters the death cascade — so the Stunned buff is a no-op (target
   is gone). Resolve: only apply the Stunned buff when the OUTCOME
   keeps the defender in combat (mercy outcome, OR... is there ever
   a case?). Probably simpler: skip Stunned when policy ≠ mercy. The
   crit narration still fires regardless ("With brutal precision
   you...!").

3. **Q3 — `mercy + never-tap` interaction.** What happens when a
   merciful controller successfully locks a sub but the defender has
   `never-tap`? Three options:
   - (a) The release fires anyway — controller is just letting them
     up because they're merciful, no input from defender required.
   - (b) Mercy requires a tap to honor; without one, the sub becomes
     a no-op (sub didn't land for resolution purposes) — but the
     position stays.
   - (c) The release still fires, but the defender takes a small
     "wrenched joint" debuff for refusing to tap (lower than cripple,
     no death cascade).
   Recommended: (a) — mercy is about the controller's nature, not a
   negotiation. The defender's never-tap policy just means they
   weren't going to ask, and the controller releases regardless.

4. **Q4 — PvP submission policy.** Two players, both with `lethal`
   policies set, end up in a grapple. Without consent rules, the
   first to win the drift roll + sub roll kills the other with full
   deprogression. Is this OK? For DOGMud's PvP-supportive design,
   probably yes — players who set `lethal` are opting in. But may
   need a one-time warning prompt the first time a player sets
   `lethal` (called out in the player UX section already).

5. **Q5 — Broken-limb interaction with healing spells.** The buff
   is `requires_heal_action: true` — meaning the existing heal-spell
   timer-decrement doesn't expire it. Is there a healing spell that
   SHOULD clear it (e.g., a high-tier `mend bones` spell)? Out of
   4d scope per non-goals; 4f flavor.

6. **Q6 — Stamina cost of sub attempts.** The chunk-4b drift tick
   already debits stamina per round. Should sub attempts cost
   additional stamina? Probably yes (sub attempts are explosive
   bursts of effort); use the existing controller-side stamina drain
   mechanic with a per-attempt multiplier. Default 1.5× normal
   round-stamina cost. Tunable.

7. **Q7 — Multiple controllers in same room.** If two mobs are
   independently controlling two grapples in the same room, each
   sub tick fires per pair. Existing chunk-4b consistency-check
   observer enforces pair validity, so this should "just work" —
   verify in T5 / T18.

8. **Q8 — Sub on a non-player target (mob defender).** Sub outcome
   for a mob defender: subdue → mob dies (no respawn); cripple →
   mob dies with `BrokenLimb` buff (which doesn't matter because
   they're dead — the buff just narrates the manner of death);
   lethal → mob dies (full death cascade with `NoDeprogression
   == false` doesn't matter because mobs don't have deprogression).
   So for mobs, all three "deadly" policies collapse to "mob dies
   + appropriate narration." Fine.

## Resumption criteria (chunk 4d done when)

- Per-round submission attempt opportunity opens on chunk-4b drift
  margin > alpha, fires opposed roll, branches on tier (bad / neutral /
  success / crit) correctly.
- Bad roll escapes the defender from the grapple entirely; controller
  falls Prone.
- Success roll resolves through the controller's pre-set policy
  (mercy / subdue / cripple / lethal) without per-round prompts on
  either side.
- Defender's `surrender` policy honored only by mercy controllers
  (realism: surrender signal exists but isn't enforceable).
- Subdue / cripple outcomes route through the Life cascade with
  `NoDeprogression: true` and `GoldLossFraction: 0.20` (default);
  defender wakes up at temple, no stat decay, partial gold to
  aggressor, optional broken-limb buff for cripple.
- Lethal outcome routes through normal death path (with deprogression
  and full corpse loot).
- Crit roll applies 1-round Stunned buff to defender ONLY for mercy
  outcome.
- `set submission <policy>` and `set surrender <policy>` work for
  players; `submission` / `surrender` no-args display current policy.
- Mob YAML supports `submission_policy:` / `surrender_policy:`
  fields, falling through to archetype defaults.
- 2 new btree primitives shipped: `mob_can_submit`,
  `mob_submission_policy_is`.
- Legacy `submit` command + `AttemptSubmission` / `ApplySubmissionSuccess`
  / `ApplySubmissionFailure` helpers + `SubmissionResult` struct
  deleted; final grep returns zero hits outside comments.
- Behavior Matrix PB-301 through PB-340 PASS or SKIP.
- Chunks 0-4c regression clean.
- Server boots cleanly past data-file loading.

## Out-of-scope / future followup candidates

- Bottom-position submissions from a separately-controlled bottom
  game (covered in 4d via the Guard-as-controller framing, but more
  nuanced bottom-attack states could be added in 4e/4f).
- Multi-attacker grapples + 2-on-1 submissions.
- Healing the broken-limb debuff via spells / potions (4f).
- Position-specific submission narration flavor variations (e.g.,
  Crucifix armbar reads differently from KneeOnBelly armbar) — 4f.
- Voluntary "drag captured opponent" mechanic after a successful
  subdue / cripple — would let bandits actually take prisoners somewhere
  rather than just rob-and-leave.
- Submission-as-progression XP: subbing a tough opponent should pay
  out skill / stat use credits separately from kills.
- Bot-style "always cripple" griefing in PvP — may need a
  cooldown-per-victim or a faction/relationship cap.
- Per-position stamina costs vary (Crucifix armbar is more expensive
  to attempt than Mount americana). 4f.
- `mob heal-injury <inst>` admin command for clearing broken-limb in
  test scenarios (sister to chunk-4c's `mob heal`).
