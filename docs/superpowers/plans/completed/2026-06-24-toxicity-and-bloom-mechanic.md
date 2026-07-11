# Toxicity Overhaul + Bloom Mechanic Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the inert toxicity system real, visible, and dangerous (Part A), then
build the Bloom drug mechanic on top of it (Part B) — Bloom Wafer → all-pools
Communion high → Crash + toxicity spike + addiction + mutation acceleration →
withdrawal → Ysolde detox.

**Architecture:** Real Go surface (unlike the content districts). Toxicity logic lives
in `internal/characters/resources.go` + the regen tick in
`internal/hooks/NewRound_AutoHeal.go`; visibility touches the status template, the
prompt-token system, and GMCP `Char.Vitals`. Per/Dex penalties apply via NEW effective-
stat accessors (no choke point exists today — rolls read `.Stats.X.ValueAdj` directly).
Bloom is mostly DATA (the wafer item + buffs 90–92) plus a thin Go layer: a consume
hook + a `NewRound_Bloom` lifecycle hook (buffs have no onEnd hook, so the crash/
withdrawal/addiction-decay run from a per-round hook) + a `BloomAddiction` field +
Ysolde's detox effect. Reuse buff statmods/flags + the mutation-progression rails.

**Tech Stack:** Go (GoMud engine), `go test ./...`, DOGMud world YAML (buffs/items),
codegraph MCP for symbol verification, the `/playtest` harness for the loop smoke.

**Spec:** `docs/superpowers/specs/2026-06-24-toxicity-and-bloom-mechanic-design.md`.

---

## Conventions for every task (READ FIRST)

- **Branch:** `feature/toxicity-and-bloom` (from `master` in Task 0). Commit per task;
  trailer `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.
- **TDD + real tests:** this is code — every Go task writes a failing `go test` first,
  then implements. Gate each task on `go test ./internal/... ./modules/...` (or the
  specific package) **green** AND a clean server boot (`go run .` →
  `grep -nE "ERROR:.*PANIC|fatal error:" /tmp/tb_boot.log` none; kill after).
- **Controller drives all shell** (subagents Write/Edit + report; controller runs
  `go test`, boot, git). Subagents that touch Go MUST use **codegraph** to verify
  symbols/signatures before editing (CLAUDE.md) — the plan gives search targets.
- **No hard numbers in player-facing text** (descriptive tiers; `status` is the
  deliberate mechanical exception). **All balance magnitudes = config knobs.**
- **Pinned anchors (verified 2026-06-24):**
  - `Character.Toxicity float64` (`internal/characters/character.go:112`); methods
    `GetToxicityMax`, `AddToxicity`, `GetToxicityPenalties` in
    `internal/characters/resources.go:259-298`.
  - Toxicity decay block: `internal/hooks/NewRound_AutoHeal.go:63-69`; the sole
    penalty consumer discards Per/Dex at `:114` (`toxRegenMult, _, _`).
  - Config: `internal/configs/config.balance.go:464-466` (ToxicityBaseMax 100,
    DecayPerTick 1.0, VitalityScale 5); defaults set in
    `internal/configs/config.balance.combat.go:206-213`.
  - Stats read directly as `.Stats.Perception.ValueAdj` / `.Stats.Dexterity.ValueAdj`
    (`internal/stats/stats.go:23`) at: `actions/cast.go:296`, `actions/combat_bash.go:97`,
    `actions/combat_drain.go:98`, `actions/combat_fire.go:48,188,195` — **plus any
    others codegraph finds**; NO effective-stat accessor exists.
  - Buffs: `internal/buffs/buffs.go` (Expired() via `TriggersLeft<=0`; no onEnd hook).
    Apply via `Character.AddBuff(id, source)` / `AddBuffScaled(id, mult)` (see
    `drink.go:163-165`). Buff YAMLs in `_datafiles/world/dogmud/buffs/`; **next-free
    id 90** → Bloom uses 90/91/92.
  - Mutations: `Character.Mutations map[string]int` (id→level,
    `character.go:249`); pkg `internal/mutations/mutations.go` (`GetAll`, `GetMutation`,
    `GetMutationLoad`, `HasMutation`).
  - GMCP vitals flow via `events.CharacterVitalsChanged` →
    `modules/gmcp/gmcp.Char.go:147` (vitalsChangedHandler).
  - Ysolde = mob `9323` (`mobs/new_plymouth_common/9323-ysolde.yaml`).

---

# PART A — Toxicity overhaul (the foundation)

## Task 0: Branch + baseline
- [ ] **Step 1:** `git checkout master && git checkout -b feature/toxicity-and-bloom`.
- [ ] **Step 2:** `go build ./... && go test ./internal/characters/... 2>&1 | tail -5` — green baseline.
- [ ] **Step 3:** Boot once (`go run . > /tmp/tb_boot.log 2>&1 &`; confirm Server Ready, no panic; kill).

## Task 1: Config knobs (Toxicity + Bloom balance block)
**Files:** Modify `internal/configs/config.balance.go` (struct + the `Balance` block), `internal/configs/config.balance.combat.go` (defaults), `_datafiles/config.yaml` (document the knobs).
- [ ] **Step 1: Add fields** to the Balance struct near the existing Toxicity knobs (`config.balance.go:464`):
```go
ToxicitySicknessDamagePct ConfigFloat `yaml:"ToxicitySicknessDamagePct"` // % max-HP/tick acute harm at top band (default 0.02)
ToxicityHighDecaySlowMult ConfigFloat `yaml:"ToxicityHighDecaySlowMult"` // decay multiplier when toxicity >= 75% (default 0.5 = clears slower)
// Bloom
BloomAddictionPerDose      ConfigInt   `yaml:"BloomAddictionPerDose"`      // default 1
BloomAddictionDecayRounds  ConfigInt   `yaml:"BloomAddictionDecayRounds"`  // rounds of abstinence per -1 addiction (default 300)
BloomWithdrawalOnsetRounds ConfigInt   `yaml:"BloomWithdrawalOnsetRounds"` // abstinence rounds before withdrawal (default 60)
BloomMutationAdvanceChance ConfigFloat `yaml:"BloomMutationAdvanceChance"` // chance/dose to advance strongest mutation (default 0.50)
BloomNewMutationChance     ConfigFloat `yaml:"BloomNewMutationChance"`     // chance/dose to instead grant a new mutation (default 0.10)
BloomCommunionRounds       ConfigInt   `yaml:"BloomCommunionRounds"`       // high duration (default 30)
BloomCrashRoundsMult       ConfigFloat `yaml:"BloomCrashRoundsMult"`       // crash duration = communion * this (default 2.5)
```
- [ ] **Step 2: Defaults** in `config.balance.combat.go` (mirror the existing `if b.ToxicityX <= 0 { b.ToxicityX = ... }` block): set each to the default above when `<= 0`.
- [ ] **Step 3: Boot** — config logs the new knobs (grep `/tmp/tb_boot.log` for `BloomCommunionRounds`); no panic.
- [ ] **Step 4: Commit** — `feat(toxicity): add toxicity-sickness + Bloom balance config knobs`.

## Task 2: `AddToxicity` clamp fix + route drink.go through it
**Files:** Modify `internal/characters/resources.go:266-273`; `internal/usercommands/drink.go:120-122`; Test `internal/characters/resources_test.go`.
- [ ] **Step 1: Failing test** — `TestAddToxicity_ClampsToMaxAndReturnsTrue`: set `c.Toxicity = max-1`, `c.AddToxicity(10)` returns true and `c.Toxicity == GetToxicityMax()` (clamped, not rejected). Run → fails (current code rejects).
- [ ] **Step 2: Implement** — change `AddToxicity` to clamp:
```go
func (c *Character) AddToxicity(amount float64) bool {
	c.Toxicity += amount
	if max := c.GetToxicityMax(); c.Toxicity > max {
		c.Toxicity = max
	}
	if c.Toxicity < 0 {
		c.Toxicity = 0
	}
	return true
}
```
- [ ] **Step 3:** Route `drink.go` through it — replace `user.Character.Toxicity += float64(itemSpec.Toxicity)` (line ~121) and the spoiled path (line ~67) with `user.Character.AddToxicity(float64(...))`.
- [ ] **Step 4:** `go test ./internal/characters/...` green; boot clean.
- [ ] **Step 5: Commit** — `fix(toxicity): AddToxicity clamps to max; drink routes through it`.

## Task 3: Effective-stat accessors — wire the dead Per/Dex penalties
**Files:** Create `internal/characters/effective_stats.go`; Modify the roll sites; Test `internal/characters/effective_stats_test.go`.
- [ ] **Step 1: Failing test** — `TestEffectivePerception_ToxicityPenalty`: a char at ≥90% toxicity has `GetEffectivePerception() == int(float64(Stats.Perception.ValueAdj)*0.80)` (the perceptionMult from `GetToxicityPenalties`), and `GetEffectiveDexterity()` applies the dexMult; at 0 toxicity both equal `ValueAdj`. Run → fails (functions undefined).
- [ ] **Step 2: Implement** the accessors:
```go
// GetEffectivePerception returns Perception adjusted for transient effects (toxicity).
func (c *Character) GetEffectivePerception() int {
	_, perMult, _ := c.GetToxicityPenalties()
	return int(float64(c.Stats.Perception.ValueAdj) * perMult)
}
// GetEffectiveDexterity returns Dexterity adjusted for transient effects (toxicity).
func (c *Character) GetEffectiveDexterity() int {
	_, _, dexMult := c.GetToxicityPenalties()
	return int(float64(c.Stats.Dexterity.ValueAdj) * dexMult)
}
```
- [ ] **Step 3: Route the combat-relevant roll sites** through the accessors. Use codegraph (`codegraph_search`/`codegraph_callers`) to find ALL reads of `.Stats.Perception.ValueAdj` and `.Stats.Dexterity.ValueAdj` on a PLAYER/attacker-or-defender character, and replace with `GetEffectivePerception()/GetEffectiveDexterity()` at the COMBAT/CAST/FIRE/DEFENSE sites (known: `actions/cast.go:296`, `actions/combat_bash.go:97` [defense], `actions/combat_drain.go:98` [defense], `actions/combat_fire.go:48,188,195`). DO NOT change stat-training/progression or display reads — only the live combat/skill rolls. (Mobs use the same Character type, so mobs at toxicity also get the penalty — acceptable/desirable.)
- [ ] **Step 4:** `go test ./internal/... ./internal/actions/...` green (existing combat tests still pass — the accessor returns ValueAdj at 0 toxicity, so no behavior change for un-poisoned chars); boot clean.
- [ ] **Step 5: Commit** — `feat(toxicity): effective Per/Dex accessors apply toxicity penalty to combat rolls`.

## Task 4: Acute high-toxicity harm (the "shortened life") + slowed high decay
**Files:** Modify `internal/hooks/NewRound_AutoHeal.go` (the toxicity block ~63-69 + near the death check ~57); Test `internal/hooks/` (or a characters-level helper test).
- [ ] **Step 1:** Extract the harm into a testable helper on Character, `internal/characters/resources.go`:
```go
// ToxicitySicknessDamage returns the acute HP damage to apply this tick from high
// toxicity (0 below the top band). Percentage-of-max-HP, scaled by how deep into the
// >=90% band the character is.
func (c *Character) ToxicitySicknessDamage() int {
	max := c.GetToxicityMax()
	if max <= 0 { return 0 }
	ratio := c.Toxicity / max
	if ratio < 0.90 { return 0 }
	bal := configs.GetBalanceConfig()
	over := (ratio - 0.90) / 0.10 // 0..1+ within/over the top band
	dmg := float64(c.GetHealthMax()) * float64(bal.ToxicitySicknessDamagePct) * (1.0 + over)
	if dmg < 1 { dmg = 1 }
	return int(dmg)
}
```
  (VERIFY `GetHealthMax()` is the correct max-HP accessor via codegraph; adjust name if needed.)
- [ ] **Step 2: Failing test** — `TestToxicitySicknessDamage`: 0 below 90%; > 0 at/above 90%; scales upward past max. Implement helper to pass.
- [ ] **Step 3: Wire into the regen tick** (`NewRound_AutoHeal.go`): after decay, `if d := user.Character.ToxicitySicknessDamage(); d > 0 { user.Character.Health -= d; <descriptive "the poison turns in you" message, throttled> }`. The existing `Health < 1` death check just below routes lethal toxicity through `Die(... TriggerHealthZero)` — so sustained max-toxicity can kill (canon). Also apply the slowed-high-decay: when `ratio >= 0.75`, multiply the decay by `bal.ToxicityHighDecaySlowMult`.
- [ ] **Step 4:** `go test ./...` green; boot clean.
- [ ] **Step 5: Commit** — `feat(toxicity): acute high-toxicity HP harm + slowed clearance at high levels`.

## Task 5: Threshold-crossing messages (band-gated)
**Files:** Modify `internal/hooks/NewRound_AutoHeal.go`; add a band helper to `internal/characters/resources.go`; Test the band helper.
- [ ] **Step 1:** Add `func (c *Character) ToxicityBand() int` (0=clear <50%, 1=queasy, 2=sick ≥75%, 3=critical ≥90%) + test. The regen tick compares the band before/after the toxicity change this tick (track prior via a transient — recompute from a stored `lastToxicityBand` on Character, NOT persisted: a runtime field, or compute pre/post within the tick since both decay and additions happen in-tick). Simplest: compute band at start of the tick vs after decay/harm; if it changed, emit the message.
- [ ] **Step 2:** On band INCREASE emit the matching descriptive line (1: "A faint nausea settles in and will not quite lift." / 2: "Your hands have a fine tremor now and your sight swims at the edges." / 3: "Your whole body is in revolt — sweat, shakes, the taste of metal."). On DECREASE, a relief line. `user.SendText(...)`. No raw numbers.
- [ ] **Step 3:** `go test ./internal/characters/...` (band helper) green; boot clean; (manual: messages are smoke-tested in Task 13).
- [ ] **Step 4: Commit** — `feat(toxicity): descriptive threshold-crossing messages`.

## Task 6: Visibility — status sheet + {tox} prompt token + GMCP vitals
**Files:** Modify the status template (find via codegraph/grep — the `.:Vitals` / `.:Info` sheet; likely `_datafiles/.../templates/.../status.template` or rendered in `internal/usercommands/status.go`); the prompt-token renderer (find the `{enc}` handler); `modules/gmcp/gmcp.Char.go` (vitals payload).
- [ ] **Step 1: Status sheet** — add a Toxicity entry to the `status` output showing the descriptive band (`ToxicityBand()` → clear/queasy/sick/poisoned/critical), colored like the encumbrance tiers. Locate the exact template/render site via codegraph (`codegraph_search "status"` + grep the Vitals box).
- [ ] **Step 2: `{tox}` prompt token** — find the prompt-token switch that handles `{enc}`/`{hp}` (grep `"{enc}"` / the prompt package) and add a `{tox}` case rendering the colored band. Document it in the prompt help if there's a token list.
- [ ] **Step 3: GMCP vitals** — add toxicity (current + max, or band string) to the `Char.Vitals` GMCP payload in `gmcp.Char.go` so the web client can show it; ensure a toxicity change triggers a vitals refresh (the regen tick already changes toxicity each round — confirm `CharacterVitalsChanged` fires or add a lightweight refresh; do NOT spam — piggyback on existing per-round vitals).
- [ ] **Step 4:** Boot clean; `go test ./modules/gmcp/...` green; (web/visual confirmed in Task 13 smoke).
- [ ] **Step 5: Commit** — `feat(toxicity): visibility — status sheet, {tox} prompt token, GMCP vitals`.

> **Part A is independently shippable here** (toxicity now works for ALL potions). Part B builds on it.

---

# PART B — the Bloom mechanic

## Task 7: `Character.BloomAddiction` field + save/load
**Files:** Modify `internal/characters/character.go` (field near `Toxicity:112`); Test `internal/characters/`.
- [ ] **Step 1: Failing test** — `TestBloomAddiction_Persists`: a Character with `BloomAddiction=3` round-trips through YAML marshal/unmarshal. Run → fails (no field).
- [ ] **Step 2: Implement** — add `BloomAddiction int `yaml:"bloom_addiction,omitempty"`` and a transient `BloomLastDoseRound int `yaml:"-"`` (runtime). Add helpers: `AddBloomAddiction(n int)`, `BloomAddictionTier() int` (0 none / 1 hooked / 2 dependent / 3 enslaved — for descriptive display).
- [ ] **Step 3:** `go test ./internal/characters/...` green; boot clean.
- [ ] **Step 4: Commit** — `feat(bloom): BloomAddiction character field + helpers`.

## Task 8: Bloom buffs (90/91/92) + the Bloom Wafer item (data)
**Files:** Create `_datafiles/world/dogmud/buffs/90-bloom_communion.yaml`, `91-bloom_crash.yaml`, `92-bloom_withdrawal.yaml`; Create the Bloom Wafer item `_datafiles/world/dogmud/items/materials-40000/40108-bloom_wafer.yaml`.
- [ ] **Step 1:** READ existing buffs for structure: a pool-regen/heal buff (e.g. `54-*`..`60-*`), a stat-mod buff, and `74-chrysalis_catalyst.yaml`. Author:
  - **90 Bloom Communion** — duration from `BloomCommunionRounds` (the consume hook scales it; author a baseline triggercount): strong all-pool regen multipliers + a small positive all-stat statmod + a buff flag granting fear/pain-debuff immunity (find the existing immunity flag via codegraph; if none, the consume hook can cancel/period-suppress — keep to a flag if one exists, else document as deferred). Euphoric apply/description text.
  - **91 Bloom Crash** — longer: negative pool-regen + negative all-stat statmods. Grim text.
  - **92 Bloom Withdrawal** — negative regen/stat statmods + craving flavor; the hook scales its severity.
- [ ] **Step 2:** Author **40108 Bloom Wafer** (model on a consumable; `type: object`, an eat/drink consumable subtype that routes through `drink`/`eat`; `not_salable: true`; a HIGH `toxicity:` value, e.g. 35; thin-iridescent-wafer description). Set `vendor_categories` only if salable — it is NOT salable, so omit (NotSalable skips the validator).
- [ ] **Step 3: Boot** — `buffs.LoadDataFiles` +3, `items.LoadDataFiles` +1; no panic / no ValidateVendorCategories issue (not_salable).
- [ ] **Step 4: Commit** — `feat(bloom): Communion/Crash/Withdrawal buffs (90-92) + Bloom Wafer item (40108)`.

## Task 9: Mutation acceleration (advance strongest; seed random mid/high-tier)
**Files:** Create `internal/characters/bloom_mutation.go`; Test `internal/characters/bloom_mutation_test.go`. Use the `internal/mutations` pkg.
- [ ] **Step 1: Failing tests** — `TestBloomAdvance_StrongestUnderCap` (a char with `{extra-arms:1, camo-skin:3}` advances camo-skin→4 when under cap); `TestBloomAdvance_RespectsCap` (at cap, does not exceed — advances the next-strongest under cap, or no-ops); `TestBloomSeed_WhenNoMutations` (a char with empty Mutations gains exactly one mutation, drawn from the mid/high-tier weighted pool).
- [ ] **Step 2: Implement** `func (c *Character) BloomAdvanceMutation(rng ...)`:
  - If `len(c.Mutations) == 0`: pick a random mutation weighted toward mid/high-tier (derive tier from `mutations.GetMutationLoad(map[string]int{id:1})` or a spec field — VERIFY the tier signal via codegraph in the `mutations` pkg; higher load ⇒ higher tier; exclude trivial/cosmetic), set `c.Mutations[id] = 1` (respecting `CanApplyTo` species + slot caps via the existing `validateMutationSlots` rules).
  - Else: find the highest-level entry (ties → deterministic by id sort); look up its cap (`mutations.GetMutation(id)` max level — VERIFY the cap field); if under cap, `c.Mutations[id]++`; else advance the next-highest under cap; if all capped, grant a new random mid/high-tier mutation instead.
  - Keep it a single clean mutation-map mutation so a future "clear all + reroll" composes (spec §B5 / [[project-moon-crash-remort-potion]]).
- [ ] **Step 3:** `go test ./internal/characters/...` green; boot clean.
- [ ] **Step 4: Commit** — `feat(bloom): mutation acceleration — advance strongest cap-aware, seed mid/high-tier`.

## Task 10: The dose hook — consuming a Bloom Wafer
**Files:** Modify the consume path (`internal/usercommands/drink.go` and/or `eat.go` — route the Bloom Wafer specially) OR add a small handler; Test where practical.
- [ ] **Step 1:** When the consumed item is the Bloom Wafer (id 40108 — gate on the itemid or a spec flag like `is_bloom`/a `drug` subtype; prefer a spec bool `is_drug: true` + a `drug_*` data block if clean, else gate on itemid as v1): instead of the normal potion buff path, the dose hook runs:
  - `AddBuffScaled(90, communionDurationMult)` (Communion; scale duration by `BloomCommunionRounds`).
  - `AddToxicity(float64(itemSpec.Toxicity))` (the high spike).
  - `AddBloomAddiction(bal.BloomAddictionPerDose)`; set `BloomLastDoseRound = util.GetRoundCount()`.
  - `if rng < BloomMutationAdvanceChance { BloomAdvanceMutation() ; <descriptive message> }` (and the smaller `BloomNewMutationChance` branch for forcing a new one).
  - A euphoric consume message.
- [ ] **Step 2:** Ensure the normal toxicity/quest-notify drink flow still applies appropriately (the wafer still counts as consumed). Keep the special-case minimal + well-commented.
- [ ] **Step 3: Boot + manual reasoning test** (full behavior smoke in Task 13); `go test ./...` green.
- [ ] **Step 4: Commit** — `feat(bloom): dosing applies Communion + toxicity spike + addiction + mutation roll`.

## Task 11: `NewRound_Bloom` lifecycle hook — crash, withdrawal, addiction decay
**Files:** Create `internal/hooks/NewRound_Bloom.go` (model on `NewRound_AutoHeal.go`); register it where round hooks register (find via codegraph — the NewRound hook registration). Test the pure helpers on Character.
- [ ] **Step 1: Failing tests** (on Character helpers the hook calls): `TestBloomCrashOnCommunionEnd` (a char whose Communion just expired gets Crash applied once); `TestWithdrawalOnsetAfterAbstinence` (addicted char with `roundsSinceDose >= BloomWithdrawalOnsetRounds` gains Withdrawal scaled by addiction tier); `TestAddictionDecayOnAbstinence` (addiction drops by 1 every `BloomAddictionDecayRounds`).
- [ ] **Step 2: Implement** the per-round hook (iterate active players, like AutoHeal): detect Communion→absent transition (track via a `HadBloomCommunion` transient or check buff presence vs a last-seen flag) → apply Crash (`AddBuffScaled(91, crashMult)`); if `BloomAddiction>0` and rounds-since-dose ≥ onset → ensure Withdrawal (92) is active, severity scaled by `BloomAddictionTier()`; decay addiction on the configured cadence; emit throttled craving messages. Dosing (Task 10) resets the abstinence clock.
- [ ] **Step 3:** `go test ./...` green; boot clean (hook registered + runs).
- [ ] **Step 4: Commit** — `feat(bloom): per-round lifecycle — crash, withdrawal onset/scaling, addiction decay`.

## Task 12: Ysolde detox (brutal-fast vs cold-turkey)
**Files:** Author Ysolde detox — a cure item she gives or a dialogue-triggered effect. Modify `mobs/new_plymouth_common/9323-ysolde.yaml` + `dialogue/new_plymouth_common/9323.yaml`; possibly a "Bloom Detox" effect (reuse buff 91-style or a new buff 93).
- [ ] **Step 1:** Decide the mechanism (cleanest for v1): a **detox via dialogue** — `ask ysolde about bloom`/`detox` triggers (when `BloomAddiction>0`) an effect that: applies a heavy immediate `AddToxicity` spike (into the danger band — the purge) + a strong pools debuff buff (new **93 Bloom Detox**, a multi-round pool/regen penalty) AND steps `BloomAddiction` down hard (large decrement or schedule-to-zero over a short rough window). This realizes "fast but brutal upfront" vs cold-turkey's "slow but mild." Implement the effect trigger via the dialogue→action path (find how dialogue fires engine effects/buffs via codegraph; if dialogue can't run Go, give Ysolde a **cure item** "Ysolde's Purge" that, when drunk, runs the same effect through the consume hook).
- [ ] **Step 2:** Author the 93 Bloom Detox buff (heavy pool/regen debuff, duration). Ysolde dialogue: discoverable `detox`/`bloom`/`sick` topic, first-person, that offers/administers it. NO quest gating in v1 (the Bloom Trail quest can gate/discount later).
- [ ] **Step 3: Boot** — buffs +1; dialogue + mob load; `go test ./...` green.
- [ ] **Step 4: Commit** — `feat(bloom): Ysolde detox — brutal-upfront fast kick vs slow cold turkey`.

## Task 13: Loop smoke (playtest) + final review + merge
- [ ] **Step 1: `/playtest local feature-tester`** (or manual telnet via the harness bridge) as `smoketester`: give yourself a Bloom Wafer (admin/inventory edit), `drink`/`eat` it → confirm the **Communion** (pools surge, message), watch toxicity climb (status/`{tox}`/threshold message), let it **Crash**, re-dose to climb addiction, abstain to trigger **Withdrawal**, then **Ysolde detox** → confirm brutal-upfront + addiction drop. Confirm mutation advance fired (check `status`/mutations). Confirm toxicity is now VISIBLE (status + GMCP/web).
- [ ] **Step 2: Triage** — fix blocking issues inline (`fix(bloom): …`); log cosmetics.
- [ ] **Step 3: Final gates** — `go test ./...` green; clean boot (`ValidateZoneConsistency errors=0`, buffs/items/mobs load, no panic).
- [ ] **Step 4: Code-review pass** — dispatch a code-quality reviewer over the Go diff (toxicity accessors, the two NewRound hooks, the dose hook, mutation advance) per the subagent-driven review stage; fix findings.
- [ ] **Step 5: Merge** — `git checkout master && git merge --no-ff feature/toxicity-and-bloom -m "Merge: Toxicity overhaul + Bloom mechanic"`. **Do NOT push** (HELD).
- [ ] **Step 6: Update memory** — toxicity+Bloom done+merged; NEXT = the **Bloom Trail questline** (its own spec→plan→build; uses this mechanic).

---

## Self-Review (completed during planning)

- **Spec coverage:** A2 visibility → Task 6; A3 wire Per/Dex → Task 3; A4 acute harm →
  Task 4; A5 decay/AddToxicity → Tasks 2+4; threshold messages → Task 5; B1 wafer +
  B2 communion + B3 crash → Tasks 8+10+11; B4 addiction → Tasks 7+10+11; B5 mutation
  accel (+ random mid/high seed + moon-potion composability) → Task 9; B6 withdrawal →
  Task 11; B7 toxicity cost → Part A (Task 4); B8 escape (cold-turkey via Task 11 decay
  + Ysolde detox) → Task 12; config knobs → Task 1; tests/smoke → every task + Task 13.
- **Placeholder scan:** new units (config, AddToxicity, accessors, sickness helper,
  band helper, addiction field, mutation advance) have literal code; integration edits
  (roll sites, status template, prompt token, GMCP, dialogue-effect, NewRound
  registration) carry explicit codegraph search targets because exact lines shift —
  the established DOGMud pattern. No TBD/TODO.
- **Consistency:** `GetEffectivePerception/Dexterity`, `ToxicitySicknessDamage`,
  `ToxicityBand`, `BloomAddiction`/`AddBloomAddiction`/`BloomAddictionTier`,
  `BloomAdvanceMutation`, buffs 90/91/92(+93 detox), item 40108, Ysolde 9323, config
  knob names — used identically across tasks.
- **Risk notes:** Task 3 (routing combat roll sites) is the highest-regression task —
  the accessor returns `ValueAdj` at 0 toxicity so un-poisoned behavior is unchanged;
  existing combat tests are the safety net. Task 6 (status template/prompt/GMCP) and
  Task 12 (dialogue→effect) need codegraph discovery of the exact mechanism before
  editing. Fear/pain-immunity flag (Task 8) may not exist — degrade gracefully
  (document deferred) rather than block.
