# DOGMud Test Coverage Audit

**Last regenerated: 2026-07-20**
**Baseline command:** `go test -coverprofile=cover.out ./...` (whole module, not just `./internal/...`)
**Repo total: 42.3% of statements** · 4,880 test functions across 675 `*_test.go` files · 122 packages

> **Previous revision was 2026-02-28 and had gone badly stale** — it described 57 packages
> (the repo now has 122), listed two packages that no longer exist, and marked as TODO a
> large amount of work that has since shipped. This is a full regeneration. See
> §0 for what changed, because the delta is genuinely good news.

Companion document: [`TECH_DEBT_AUDIT_2026-07-20.md`](TECH_DEBT_AUDIT_2026-07-20.md) — the
broader codebase audit. Testing findings there and here are cross-referenced.

---

## §0 — What changed since 2026-02-28

Every Tier-1 package improved, most of them dramatically. The Feb doc's headline figure was
"Tier 1 weighted average ~24% (critical gap)."

| Package | Feb 2026 | Jul 2026 | Δ |
|---------|---------:|---------:|------:|
| combat | 3.3% | **42.4%** | +39.1 |
| dice | 69.7% | **83.9%** | +14.2 |
| characters | 27.3% | **44.2%** | +16.9 |
| mutations | 52.1% | **76.3%** | +24.2 |
| crafting | 57.0% | **69.5%** | +12.5 |
| spells | 0.0% | **50.3%** | +50.3 |
| items | 0.0% | **65.9%** | +65.9 |
| mobs | 0.0% | **59.1%** | +59.1 |
| hooks | 0.0% | **45.1%** | +45.1 |
| buffs | 25.7% | **67.5%** | +41.8 |
| dialogue | 0.0% | **60.4%** | +60.4 |
| skills | 0.0% | **54.2%** | +54.2 |
| rooms | 5.2% | **43.7%** | +38.5 |
| species | 0.0% | **25.0%** | +25.0 |
| **quests** | 0.0% | **0.0%** | — |
| **enchantments** | 0.0% | **0.0%** | — |

**Tier 1 unweighted mean: 24% → 60.4%.** That is the single most important number in this
document. The Feb plan worked.

Also shipped since Feb:
- **10 of 12** Stage 40.3 integration scenarios (all were TODO).
- **6 of 10** Stage 40.4 items (mostly TODO).
- The `seedRegistry` pattern spread from 2 packages to **12**.
- CI now runs `-race` + coverage on every PR and master push (Feb doc listed this as TODO).

Two Tier-2 packages did not move off zero: **`quests`** and **`enchantments`**. Both are
addressed in §4.

---

## §1 — Coverage by package (regenerated 2026-07-20)

Distribution across 122 packages:

| Band | Count |
|------|------:|
| 0% | 31 |
| 0–25% | 13 |
| 25–50% | 23 |
| 50–75% | 26 |
| 75–100% | 29 |

### 1a. Zero coverage — no test file at all

Excludes `cmd/generate`, `tools/*`, and the root package (build/CLI utilities, out of scope).

**Tier 2 — should not be zero:**
`internal/quests` ⚠️ (see §4.1 — has a test file that covers nothing) ·
`internal/enchantments` ⚠️ (see §4.2)

**Tier 3 — infrastructure, ranked by risk:**
`internal/connections` · `internal/plugins` · `internal/statmods` · `internal/fileloader` ·
`internal/mudlog` · `internal/keywords` · `internal/conversationadapter` · `internal/llm` ·
`internal/copyover` · `internal/mutators` · `internal/pets` · `internal/audio` ·
`internal/colorpatterns` · `internal/exit` · `internal/flags` · `internal/stats` ·
`internal/suggestions` · `internal/integrations/discord`

**Modules (an entire tree the Feb doc never covered):**
`modules/achievements` · `modules/leaderboards` · `modules/cleanup` · `modules/follow` ·
`modules/time` · `modules/webhelp`

> Note the split: `internal/achievements` (the logic) is at 39.4%, but `modules/achievements`
> (the wiring that hooks it into the server) is at 0%. Same for leaderboards. The math is
> tested; *whether it's actually plugged in* is not.

### 1b. Under 25% — tested but thin

| Coverage | Package | Note |
|---------:|---------|------|
| 1.7% | `internal/web` | Admin dashboard backend; only `auth_test.go` exists |
| 4.6% | `internal/gametime` | Calendar/clock math — pure logic, should be far higher |
| 6.4% | `modules/gmcp` | Client protocol; touches every room move |
| 11.6% | `modules/weather` | Root pkg thin; subpackages are 88–99% |
| 11.7% | `internal/term` | Terminal/ANSI handling |
| 14.5% | `internal/events` | **Core event bus** — see §4.3 |
| 17.0% | `internal/inputhandlers` | Login/telnet state machine |
| 18.5% | `internal/migration` | Version upgrade rewriters |
| 18.8% | `internal/conversations` | NPC↔NPC exchange engine |
| 18.8% | `modules/playtest` | Test harness beacons |
| 19.3% | `internal/bountyhunter` | |
| 24.4% | `internal/goals/catalog` | |

### 1c. 25–50%

`species` 25.0 · `ferry` 26.3 · `gamelock` 30.8 · `state/perception` 30.8 ·
`usercommands` 31.3 · `modules/weather/engine` 31.9 · `worldevents` 36.6 ·
`modules/auctions` 37.2 · `state/activity` 38.3 · `mobcommands` 38.4 ·
`achievements` 39.4 · `planners` 41.7 · `questengine` 42.1 · `combat` 42.4 ·
`devtools` 43.3 · `rooms` 43.7 · `characters` 44.2 · `hooks` 45.1 · `actions` 47.4 ·
`parties` 48.2 · `seeders` 48.3 · `users` 48.5 · `templates` 48.9

### 1d. 50–75%

`spells` 50.3 · `state/life` 51.9 · `itemvalue` 53.8 · `skills` 54.2 · `itemvoices` 54.5 ·
`state/awareness` 55.7 · `behaviortree` 57.7 · `mobs` 59.1 · `caravan` 59.8 ·
`forager` 59.9 · `dialogue` 60.4 · `prompt` 60.5 · `state/position` 62.2 ⚠️ (see §3.2) ·
`justice` 63.4 · `items` 65.9 · `buffs` 67.5 · `state/presence` 68.2 · `crafting` 69.5 ·
`opinions` 70.6 · `textutil` 70.6 · `configs` 71.1 · `uuid` 72.1 · `parser` 72.6 ·
`warehouse` 72.8 · `util` 73.0 · `shops` 74.1

### 1e. 75%+ — the healthy tail

`mutations` 76.3 · `markdown` 78.0 · `facts` 78.2 · `language` 78.2 · `economy/health` 80.1 ·
`factions` 80.2 · `state/combatphase` 81.2 · `crimes` 83.5 · `dice` 83.9 ·
`state/control` 84.7 · `version` 86.5 · `grapplemessaging` 86.7 · `banner` 87.5 ·
`modules/weather/content` 87.8 · `sealedcrate` 88.9 · `goals` 89.0 · `messaging` 90.6 ·
`state` 92.9 · `modules/weather/seasons` 93.0 · `casing` 94.1 · `modules/weather/sim` 94.1 ·
`relationships` 95.0 · `modules/weather/crawler` 98.8 · `badinputtracker` 100 ·
`channels` 100 · `economy` 100

### 1f. Packages removed since Feb

`internal/clans` (no Go files) · `internal/scripting` (deleted). Remove from any tracking.

---

## §2 — Tier classification (updated for 122 packages)

The Feb tier model still works conceptually. What was missing is that ~45 packages created
since then were never sorted into it.

### Tier 1 — Critical (core gameplay)
Existing: `combat`, `dice`, `characters`, `mutations`, `crafting`, `spells`, `items`,
`mobs`, `hooks`, `buffs`

**Newly classified Tier 1:**
- **`internal/state`** (+ `position`, `control`, `activity`, `life`, `combatphase`,
  `presence`, `awareness`, `perception`) — the authoritative combat-position/lifecycle state
  machine. It *absorbed* functions the Feb doc listed under `combat/grapple.go`
  (`CheckClinchProgression`, `CheckGroundedEscape`). Called from `combat`, `actions`,
  `behaviortree`, `hooks`, `users`.
- **`internal/behaviortree`** — the entire mob AI decision layer. At 57.7% with ~55 test
  files, this is an undocumented success story rather than a gap.
- **`internal/questengine`** — owns the quest state machine (triggers/conditions/actions).
  Core to all quest content and the newcomer tutorial.

### Tier 2 — Important (gameplay quality)
Existing: `quests`, `dialogue`, `enchantments`, `skills`, `species`, `rooms`

**Newly classified Tier 2:** `shops`, `economy`, `economy/health`, `warehouse`, `caravan`,
`ferry` (living economy) · `guilds`, `achievements` (retention features) · `justice`,
`bounties`, `bountyhunter`, `factions` (crime/reputation) · `goals`, `planners`, `seeders`
(AI motivation layer) · `itemvalue` (upgrade scoring — took over from `items.IsBetterThan`) ·
`modules/gmcp`, `modules/weather`

### Tier 3 — Utility / infrastructure
Everything else, including `relationships`, `opinions`, `knowledge`, `facts`, `crimes`,
`mutators`, `conversations`, `parser`, `messaging`, `textutil`, `casing`, and the remaining
`modules/*`.

### Coverage targets

| Tier | Target | Current mean | Rationale |
|------|-------:|-------------:|-----------|
| Tier 1 | 85% | **60.4%** | Bugs here are game-breaking. Mostly pure math/state transitions. *(Target lowered from Feb's 95% — see note.)* |
| Tier 2 | 70% | ~45% | Important features; some integration complexity. *(Lowered from 85%.)* |
| Tier 3 | 50% | varies | Test pure logic; skip raw I/O and network wrappers. *(Lowered from 60%.)* |

> **On lowering the targets:** the Feb targets (95/85/60) were aspirational and never had a
> path to being met. A 95% target on `hooks` is not achievable without restructuring the
> event system. Targets that are permanently missed stop functioning as targets. These
> numbers are set where sustained progress is realistic — raise them once hit.

---

## §3 — Test suite quality

Coverage percentage measures *statements executed*, not *behavior verified*. This section
covers what the percentage can't see. **This is the most actionable part of the document.**

### 3.1 Skipped tests: 211 skip sites across 4,880 test functions (4.3%)

Concentration is extreme — three files hold 60% of all skips:

| File | Test funcs | Skips | % skipped |
|------|-----------:|------:|----------:|
| `internal/state/position/position_test.go` | 124 | **87** | 70% |
| `internal/state/activity/activity_test.go` | — | 22 | — |
| `internal/state/life/life_test.go` | — | 15 | — |

**Classification:**

**(a) Migration-checklist shells (~130 sites) — the dominant category.** Empty test bodies
that skip with a pointer to where the behavior was allegedly verified during a migration:
`t.Skip("integration test — verified in Task 5 cascade tests")`. These were a plan tracker
rendered as test functions. They inflate the apparent test count and make
`internal/state/position` look tested when 70% of that file asserts nothing.
**Recommendation: delete them.** The migration is finished; a skipped test is worse than no
test because it reads as coverage. Keep any that name a *specific, existing* test elsewhere.

**(b) ⚠️ False assurance — VERIFIED PROBLEM.** 15 skip messages say
`"covered by control_test.go:TestX (add if missing)"`. **I checked all 8 distinct referenced
tests. Every one is missing:**

```
MISSING: TestInitialControlForPair_HalfGuard      MISSING: TestInitialControlForPair_NorthSouth
MISSING: TestInitialControlForPair_BackStanding   MISSING: TestInitialControlForPair_Crucifix
MISSING: TestInitialControlForPair_SideControl    MISSING: TestInitialControlForPair_Turtle
MISSING: TestInitialControlForPair_KneeOnBelly    MISSING: TestInitialControlForPair_BackGround
```

Eight grapple positions — half guard, back standing, side control, knee-on-belly,
north-south, crucifix, turtle, back-ground — have **no control-initialization test anywhere
in the repo**, while the skip messages assert they are covered. The "(add if missing)"
phrasing shows the author never confirmed. This is the highest-value finding in §3.

**(c) Legitimate-permanent.** Genuine constraints — keep, no action:
- `internal/actions/actions_test.go:238` — real import cycle (usercommands and mobcommands
  both import actions).
- `internal/migration/reclassify_test.go:21` — skips when `_archive/prod-users` absent.
- `internal/templates/process_test.go` (7 sites) — skips when a template file isn't on disk.

**(d) Fixture-blocked — the real backlog.** Tests that would work if fixtures existed:
- `internal/characters/godfunc_refactor_test.go:289–318` (5 sites) — all
  `"BLOCKED: requires items.LoadDataFiles() which needs configs init"`. Fixing the configs-init
  seam unblocks 5 tests at once.
- `internal/itemvalue/delta_test.go` (6 sites) — needs global item registry + balance config.
- `internal/mobcommands/gearup_test.go` (3), `internal/usercommands/suicide_cleanup_test.go` (4).

**(e) Probabilistic self-skip — a hidden flake class.** Six sites in
`internal/actions/combat_drain_test.go`, `combat_throttle_test.go`, and
`internal/hooks/spell_drainarea_test.go` read:
`t.Skip("no hit observed in 100 attempts — probabilistic test; re-run if flaky")`.
These **silently pass when the RNG doesn't cooperate**. If a change made the hit rate zero,
these tests would skip rather than fail — the exact scenario they exist to catch.
**Recommendation:** raise the attempt count until the miss probability is negligible, or
seed the RNG deterministically, and make exhaustion a `t.Fatal` rather than a skip.

**(f) Stale-debt — cheap wins with a paper trail.**
- `internal/hooks/Death_PlayerRespawn_test.go:55` — *"Die() now revives userId-0 characters;
  the soft-lock discriminator changed — re-audit new-player death."* Honestly flagged,
  unaudited since. Either the revival is correct (assert it) or it's a new-player-death bug.
- `internal/behaviortree/conditions_state_test.go:77` — *"NightHours=0 in test environment —
  IsNight() always false, nothing to assert."* The test environment defeats the test; seed
  the config instead.

### 3.2 Tests that cannot fail

All quoted below were read and verified during this audit.

**Zero assertions** — `internal/mobs/mobs_test.go:869, :1030`:
```go
func TestSleep(t *testing.T) {
	mob := &Mob{InstanceId: 42}
	// Should not panic
	mob.Sleep(1)
}
```
Passes if `Sleep` is a no-op or corrupts state. `TestAddBuff` is identical in shape — nothing
verifies a buff event was queued.

**Assertion unreachable behind `recover()`** — `internal/mobs/mobs_test.go:1057`:
```go
defer func() { recover() }()
result := TickMobCraft(mob)
assert.Nil(t, result)      // never reached if TickMobCraft panics
```
A panicking implementation passes exactly as cleanly as a correct one. (Narrow — only 2 sites
repo-wide, also `TestGetAngryCommand`'s no-species subtest at `:370`.)

**Tests the thing it isn't named after** — `internal/rooms/corpse_test.go:173`,
`TestCorpseUpdate_GametimeIntegration`. Never constructs a `Corpse` or calls `Update()`. Its
own comments are leftover scaffolding: *"If you have a need to test how gametime transforms
RoundCreated + decayRate, you might do something like..."* Provides zero coverage of the type
it's filed under.

**Name promises more than the assertion delivers** — `internal/mobs/mobs_test.go:777`,
`TestValidateCallsCharacterValidate` only checks `assert.NoError`. Delete the delegation call
entirely and it still passes.

**Comment contradicts the assertion** — `internal/combat/damage_pipeline_test.go:202`,
`TestMitigationCap` says it verifies the cap equals 0.75 but only asserts `0 < got <= 1.0`.
An implementation returning 0.99 for every channel — breaking the per-channel-configurable
contract — passes.

### 3.3 What's genuinely strong

Worth stating plainly, because it sets the bar:
- `internal/dice/dice_test.go` — Monte Carlo distribution tests (10k–100k iterations) against
  expected mean/stdDev/crit-rate with explicit tolerances, plus `Example*` funcs and benchmarks.
- `internal/characters/regression_test.go` — each test names the stage/bug it guards;
  `TestRegression_AlignmentFullyRemoved` uses reflection to assert a field stayed deleted.
- `internal/questengine/engine_test.go` — asserts exact granted-quest sets, gold amounts, and
  `Handled` results across chained triggers, with a circular-dependency guard.
- `internal/behaviortree`, `internal/goals`, `internal/items`, `modules/auctions` — sampled
  broadly, no weak tests found; exact value assertions throughout.

### 3.4 Parallelism — deliberately absent, and correctly so

`t.Parallel()` appears in exactly **1** file — and that file documents why it's unsafe
(`internal/goals/prune_test.go:10-14`: *"package-global driven by serial tests via
defer-restore. Do NOT mark tests in this file t.Parallel() while this global exists"*).

Given the package-global architecture, broad `t.Parallel()` adoption would be unsafe without
the dependency-injection refactor described in the tech-debt audit §4.6. **This is the right
trade-off, not a gap.** If wall-time becomes a problem, parallelize *across* packages
(`go test ./...` already does) rather than within them.

---

## §4 — Specific gaps worth acting on

### 4.1 ⚠️ `internal/quests`: a passing test suite that covers 0.0%

`quests_test.go` **runs and passes** (`ok internal/quests 1.015s`) yet reports **0.0% of
statements**. The three tests unmarshal YAML into `Quest`/`QuestReward` structs — that
executes `gopkg.in/yaml.v2` library code, not `quests.go`.

**Those tests are good and should stay.** They are struct-tag regression guards — precisely
the class that would have caught the documented `hostile:` incident (a yaml tag on an
unexported field silently no-op'd for two months on prod).

**The lesson: coverage % is the wrong metric for schema packages.** Don't "fix" this by
deleting the tests.

**The actual gap:** all 14 exported functions in `quests.go` are untested, including pure
string/token logic that is trivially testable:
`IsTokenAfter` (:103) · `PartsToToken` (:153) · `TokenToParts` (:157) · `GetQuest` (:173) ·
`ValidateFlag` (:219) · `GetQuestCt` (:93) · `GetFlagRegistry` (:235)

`IsTokenAfter` and `TokenToParts` decide quest-step ordering across all quest content. They
are ~50 lines of pure string manipulation with zero tests.

### 4.2 ⚠️ `internal/enchantments`: 0% coverage

Zero real tests. Its only non-source file is `test_helpers.go`, holding
`SeedEnchantmentsForTest`.

This package is boot-critical (loaded at `main.go:62`) and drives
`characters/migrate_enchantments.go`, `usercommands/skill.disenchant.go`,
`hooks/NewRound_UserRoundTick.go`. Its `copyStatMods` function is exactly the
shallow-copy-shared-pointer bug class already documented as a past incident.

**Action:** test `copyStatMods` for genuine deep-copy independence.

> **Correction (2026-07-20).** An earlier revision of this document called the
> `test_helpers.go` filename a smell and recommended renaming it to `helpers_test.go`
> so it would not compile into the production binary. **That recommendation was wrong
> and would break the build.** `SeedEnchantmentsForTest` is consumed *across* packages —
> e.g. `internal/characters/pool_reservation_pinnacle_test.go:99` — and Go does not
> export identifiers from `_test.go` files to other packages. The non-`_test.go` name is
> a deliberate, necessary trade-off, and it is the consistent convention across all 12
> seed-helper packages. The observation that this code ships in the binary is true; the
> conclusion drawn from it was not.

### 4.3 `internal/events` at 14.5% — the bus everything rides

Per the tech-debt audit §1.1, `DoListeners` is the single dispatch point for every combat
round, quest event, and command execution — with no panic recovery. It is 14.5% covered.

### 4.4 Regression guards for documented past incidents

| Incident | Guarded? |
|---|---|
| yaml tag on unexported field (`hostile:`, 2 months on prod) | ✅ `internal/mobs/legacy_hostile_test.go` — `TestLegacyHostileYAMLBackcompat` |
| shallow-copy shared pointers on mob spawn | ⚠️ Partial — `internal/mutations/opposition_test.go` touches it; no guard on `newMobByIdInternal`'s copy |
| quest reward YAML keys are no-underscore (`itemid` not `item_id`) | ✅ `internal/quests/quests_test.go` (the 0%-coverage suite from §4.1 — it's doing real work) |
| filename must match name field (startup panic) | ⚠️ Indirect only, via `ConvertForFilename` tests in `util`/`caravan`/`warehouse` |
| **dialogue bare-scalar list field mutes whole NPC** (`questRequired: "X"` vs `["X"]` → yaml.v2 nils the entire file) | ❌ **UNGUARDED** — zero test files mention `questRequired` |

The last row is the highest-priority missing regression test in the repo: the failure is
**silent**, kills an entire NPC's dialogue, and lazy-loading hides it from the boot test.
See tech-debt audit §5.1 — migrating to yaml.v3 with `KnownFields(true)` would catch this
class structurally, but a regression test should land regardless.

### 4.5 Untested code where a live defect was already found

Four of the eight Tier-0 bugs in the tech-debt audit sit in code with **zero test coverage**.
That is the concrete argument for this document:

| Bug | Location | Package coverage |
|-----|----------|-----------------:|
| Players loot locked containers | `usercommands/get.go:451` | 31.3% (this path: 0) |
| Mobs mint gold via `put` | `mobcommands/put.go:74` | 38.4% (this path: 0) |
| `GetAuctionHistory` slice panic | `modules/auctions/auctions.go:1183` | 37.2% (this method: 0) |
| Instant crafts skip skill progression | `usercommands/craft.go:121` | 31.3% (this path: 0) |

Two further landmines found in zero-coverage code during the same sweep:
- **`statmods.StatMods.Add`** (`statmods.go:53`) — `if s == nil { s = make(StatMods) }`
  reassigns the local parameter, so it never escapes to the caller. `Add` on a nil map is a
  **silent no-op**; the bonus vanishes with no panic. All current callers happen to
  pre-allocate, but nothing enforces that for the next enchant/affix feature. Worse,
  `items_test.go:242` (`TestEnchantUnEnchant`) calls this exact path and only asserts
  `Damage.BaseDamage` — never the resulting stat bonus — so the test **masks** the landmine.
- **`connections.Kick` vs `Remove`** (`connections.go:107-149`) — `Kick` (used on `/quit`,
  duplicate-login, admin kick) closes the connection but never `delete()`s it from
  `netConnections`, unlike `Remove`. Zero test files in the package, so nothing guards the
  asymmetry; stale entries linger until an unrelated failed read/write cleans them up.

### 4.6 Large source files with no proportional test presence

| File | Lines | Package coverage |
|------|------:|-----------------:|
| `internal/rooms/rooms.go` | 2,762 | 43.7% |
| `internal/hooks/spell_resolution.go` | 1,478 | 45.1% |
| `internal/mapper/mapper.go` | 1,189 | 24.9% |
| `modules/gmcp/gmcp.Char.go` | 1,042 | 6.4% |
| `internal/util/util.go` | 1,076 | 73.0% |
| `modules/auctions/auctions.go` | 1,195 | 37.2% |

`modules/gmcp` at 6.4% is the standout: `GetCharNode` is a 356-line branch-by-string function
that builds every GMCP payload the web client consumes, and it fires on every room move.

---

## §5 — Testability barriers (status update)

### Barrier 1: Global singleton registries — ✅ SOLVED
The Feb recommendation (Option A, `seedRegistry`) didn't just get adopted, it became house
style. `Seed<X>ForTest` helpers now exist in **12 packages**: mutations, crafting, spells,
items, buffs, mobs, enchantments, rooms, species, keywords, mutators, users — exceeding the
original ask (spells/items/mobs/buffs) by four.

### Barrier 2: Interleaved logic and side effects — ✅ MOSTLY SOLVED
The "extract ~10 pure helpers" recommendation was followed almost literally.
`internal/hooks/combat_shared_helpers.go` now has exactly 10 top-level helpers
(`calcSpellDamageForCharacter`, `checkConcentrationBreak`, `tryWeaponBreak`,
`applyCritEffects`, `simulateFoldRound`, `calcFoldConvictionCost`, `clearCastingActivity`,
`cancelCraftOrSalvageOnDamage`, `cancelDamageBuffs`, `processFoldRound`), several individually
tested. Caveat: `spell_resolution.go` has *grown* to 1,478 lines despite the extraction.

### Barrier 3: Embedded RNG — ⚠️ STILL TRUE
No RNG-injection refactor. Option C (statistical testing) remains dominant and works well
(`TestAttemptGrapple_Statistical`). **But see §3.1(e)** — the probabilistic self-skip pattern
is the failure mode of relying on RNG without seeding.

### Barrier 4: File I/O in constructors — ⚠️ STILL TRUE
Option A (construct test objects directly) remains standard, now reinforced by the
seedRegistry spread. Note this is what blocks the 5 `godfunc_refactor_test.go` tests
(§3.1(d)) — `items.LoadDataFiles()` requires configs init.

---

## §6 — Test patterns in use

Patterns 1–8 from the Feb revision are all still active and correct: `seedRegistry`, factory
helpers (`makeItem`/`buildOwned`), `testify/assert`, statistical N-iteration verification,
table-driven tests, float-epsilon comparison, monotonicity/curve verification, and global
state save/restore.

**New — Pattern 9: the standardized `test_helpers.go` file.** Twelve packages now export a
single `Seed<Package>ForTest(data) func()` returning a cleanup closure, with a consistent
docstring. This is the repo-wide generalization of patterns 1 + 8 and is now more prescriptive
than either.

**Convention note:** keep the `test_helpers.go` name (no `_test.go` suffix). That is
deliberate, not an oversight: these seeders are imported by *other* packages' tests, and Go
does not export identifiers from `_test.go` files across package boundaries. The cost is
that the helper compiles into the production binary; the benefit is cross-package seeding,
which is the entire point of the pattern. See the correction note in §4.2.

---

## §7 — Stage 40.x checklist status

### 40.3 — Integration scenarios: 10 of 12 DONE (all were TODO)

| # | Scenario | Status |
|---|----------|--------|
| 1 | Full melee attack round | ✅ `TestIntegration_CombatLifecycle` (combat/integration_combat_test.go:21) |
| 2 | Spell cast → resolve → damage | ✅ `TestCalcSpellDamage_SpellPowerAmplifies` + hooks_test.go battery |
| 3 | Grapple sequence | ✅ combat/submission_test.go, grapple_test.go — **note:** clinch/ground progression moved to `internal/state/position` |
| 4 | Defense resolution (best-of-all) | ✅ `TestIntegration_DefenseAndMitigation` (:162) |
| 5 | Resource depletion → penalty curve | ✅ `TestIntegration_CombatStaminaDepletion` (:72) |
| 6 | Skill progression over N uses | ✅ `TestIntegration_SkillProgressionSimulated` |
| 7 | Stat progression over N uses | ✅ `TestIntegration_StatProgressionSimulated` |
| 8 | Crafting loop | ✅ `TestIntegration_CraftingFullLoop` |
| 9 | Mutation acquisition + stacking | ◐ Acquisition covered (`TestRollAcquisition`); stacking only via unit-tested `GetMutationLoad`/`HasConflict` |
| 10 | Buff application + expiry | ✅ buffs/buffs_test.go:303,757,831 |
| 11 | Item comparison chain | ◐ `TestIsBetterThan` only — richer logic moved to `internal/itemvalue` |
| 12 | Mob AI move selection | ◐ Dispatch covered; deeper logic migrated to `internal/behaviortree` |

### 40.4 — Regression / refactor / CI

| # | Task | Status |
|---|------|--------|
| 1 | Extract pure helpers from hooks | ✅ 10 helpers in combat_shared_helpers.go |
| 2 | Hooks unit tests for extracted helpers | ✅ 60+ test files in hooks/ |
| 3 | seedRegistry for spells | ✅ |
| 4 | seedRegistry for items | ✅ (Feb doc said "N/A" — now exists) |
| 5 | seedRegistry for mobs | ✅ `mobs.SeedMobsForTest` |
| 6 | Regression: damage pipeline edge cases | ✅ |
| 7 | Regression: mitigation cap enforcement | ⚠️ Exists but is weak — see §3.2 |
| 8 | Regression: defense floor | ✅ `TestRegression_DefenseFloorAlwaysApplies` |
| 9 | CI coverage gate (Tier 1 < 90%) | ⚠️ **Shipped differently** — global **28%** floor on total repo, no per-tier logic |
| 10 | Smoke test in CI | ✅ `go test ./...` on every PR + master push |

**On #9:** the shipped gate is a flat 28% repo-wide floor. With the repo now at 42.3%, that
floor is 14 points of slack — it cannot detect a significant regression. It also soft-passes
(`exit 0`) if `coverage.out` is missing. Options: raise the floor to ~40% to match reality,
or implement the per-tier gate as originally specified. The former is a two-character change
and captures most of the value.

### 40.2 — Still deferred
`ChanceToTame` (combat/calculations.go:140) remains untested. **`PowerRanking` no longer
exists** — replaced by `PowerScore` (calculations.go:19), a full redesign consuming the real
damage pipeline. Section E of the old doc referenced several deleted functions; all
file:line references in this document were re-verified on 2026-07-20.

---

## §8 — Next 16 tests to write

Ranked by risk-prevented ÷ effort. Each is actionable without further investigation.

The first four are **fix-plus-test**, not test-only: the audits turned up live defects in
untested code, and writing the test without the fix leaves a red suite. Full diagnosis for
each is in the tech-debt audit's Tier 0.

| # | Target | Why | Assert | Effort |
|---|--------|-----|--------|:-----:|
| 1 | **`usercommands.Get` container lock check** (`get.go:451`) | **LIVE, PLAYER-REACHABLE BUG** — `get.go` has zero lock checks while every sibling command has one; players loot locked chests without picking (tech-debt §0.6) | Locked container rejects `get gold/item from` with an "is locked" message; balances and contents unchanged | S |
| 2 | `mobcommands.Put` gold path | **Live bug** — credits container without debiting mob, skips lock check (tech-debt §0.1) | Mob gold decreases by exactly the deposited amount; locked container refuses | S |
| 3 | `auctions.GetAuctionHistory` (`auctions.go:1183`) | **Latent panic** — `PastAuctions[len-n : n]` is `[7:3]` for n=3, len=10. Dormant only because the sole caller passes 0 (tech-debt §0.7) | `GetAuctionHistory(3)` on a 10-item history returns the last 3 without panicking | S |
| 4 | `statmods.StatMods.Add` nil-receiver (`statmods.go:53`) | `if s == nil { s = make(...) }` reassigns the local param — `Add` on a nil map is a **silent no-op**, bonus vanishes. 0% coverage | Either fix (return the map / require pre-allocation) or test-document the no-op; also assert `items_test.go:242`'s enchant actually applies a stat bonus, which it never checks today | S |
| 5 | The 8 missing `TestInitialControlForPair_*` (§3.1b) | Skip messages claim coverage that doesn't exist; 8 grapple positions unverified | Correct initial control value per position (half guard, back standing, side control, knee-on-belly, north-south, crucifix, turtle, back-ground) | S |
| 6 | `dialogue` bare-scalar regression (§4.4) | Only documented incident with **no** guard; silent, kills whole NPC file | Walk real `_datafiles/**/dialogue/*.yaml` asserting zero unmarshal errors; plus a unit case feeding `questRequired: 34-end` (bare scalar) asserting it errors rather than nilling the file | S |
| 7 | `quests.IsTokenAfter` / `TokenToParts` / `PartsToToken` (§4.1) | Pure string logic gating quest-step ordering for all quest content; 0% | Round-trip `PartsToToken`↔`TokenToParts`; ordering incl. malformed tokens and skip-ahead/backward rejection | S |
| 8 | Un-skip `usercommands/suicide_cleanup_test.go` (4 tests) | Cheapest win in the audit — the file's own header says to use `seedAllRegistries()`, which **already exists unused in the same package** at `usercommands_test.go:76` | Death/respawn cleanup correctness | S |
| 9 | Un-skip `characters/godfunc_refactor_test.go` `TestWear_*` (5 tests) | Blocked on `items.LoadDataFiles()`, but `items.SeedItemsForTest` already exists and is unused here. Equip-slot logic has zero coverage | Empty-slot, slot-swap, 2H-displaces-offhand, wrong-type rejection, multi-arm routing | S |
| 10 | `enchantments.copyStatMods` (§4.2) | Boot-critical, 0%, known shallow-copy bug class | Mutating the copy does not mutate the source | S |
| 11 | `statmods.Get` | 0%, pure logic, feeds the combat-math stack | Multi-stat sum; unknown name returns 0, not panic | S |
| 12 | Fix `TestMitigationCap` (§3.2) | Existing test can't catch the bug it names | Each channel returns its *configured* cap, not merely 0<x≤1 | S |
| 13 | Replace zero-assertion `TestSleep`/`TestAddBuff` (§3.2) | Currently pass if methods are no-ops | Sleep sets the buff flag; AddBuff queues the expected event | S |
| 14 | `hooks.transferPartialGold` credit side (`Death_PlayerCorpse.go:70`) | Debit half is tested; **credit half never executes** — the only test passes a zero `ActorRef{}`, so conservation was never checked. Fires on every subdue/cripple death | Killer's gold increases by exactly `loss` for both the player-killer and mob-killer branches | M |
| 15 | `connections` registry concurrency | Every player session; concurrency-heavy → real `-race` beneficiary. Note `Kick` never `delete()`s from the map while `Remove` does — no guard on that asymmetry | Add/Get/Remove round-trip; concurrent Add+Remove doesn't corrupt the registry | M |
| 16 | Data-file boot test (tech-debt §6.2) | Automates the manual Pre-Push SOP; prerequisite for the yaml.v3 migration | `mobs`/`quests`/`rooms`/`dialogue` loaders return zero errors and `loadedCount > 0` against real `_datafiles` | M |

**Deliberately ranked lower** (high blast radius, but genuinely large effort):
`applyControlShift` (`hooks/Position_GrappleTick.go:744`) fires every round of every grapple
and has no direct test — but the 35 dead skip references pointing at it must be untangled
first, and the fixture pattern needs establishing from `TestProcessGrapplePair_StashesDriftSnapshot`.
Real **L** effort. Same for `modules/gmcp.GetCharNode` (6.4% package coverage, 356-line
function on every room move) and `internal/hooks/spell_resolution.go`, which has **no dedicated
test file at all** despite `hooks` having 58 other test files.

**Also do (not tests, but test-suite hygiene):**
- Delete the ~130 migration-checklist skip shells (§3.1a) — they misrepresent coverage.
- Fix the 6 probabilistic self-skips (§3.1e) — seed the RNG; make exhaustion fail, not skip.
- Rename `internal/enchantments/test_helpers.go` → `helpers_test.go` (§4.2).
- Raise the CI coverage floor from 28% to ~40% (§7 #9) and make a missing `coverage.out`
  a failure rather than a pass.

---

## Appendix — how this document was generated

Coverage: `go test -coverprofile ./...` run 2026-07-20 on the full module; per-package
figures extracted from the run log. Suite green (exit 0). `-race` could not be run locally
(no C toolchain — `-race` requires cgo); CI runs it on Linux.

Every file:line in this document was re-verified on 2026-07-20. Skip counts, the 8 missing
`TestInitialControlForPair_*` tests, the `quests` 0%-with-passing-tests anomaly, the
zero-assertion tests, and the `enchantments` helper-suffix issue were each confirmed by
direct inspection rather than inference.

**Regenerate §1 whenever this doc is revisited** — it is a snapshot and will drift. The
qualitative sections (§3–§6) age much more slowly.
