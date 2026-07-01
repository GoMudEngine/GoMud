# Crash Site Interior — Plan B2 (Interior Content + Endgame Mechanics)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the full 30-room / 3-stage Crash Site Interior finale on top of B1's instance foundation — the Chrysalis-suppression aura, the mutation-scour reward, the one-time revelation, two boss-tier wardens, instance-scaled loot + legendary reagents, and the trap-dungeon content.

**Architecture:** Two small engine mechanics land first (a `Dampened` buff-flag suppression aura checked at the spell + mutation chokepoints; a scour-potion delivery special-cased in `drink.go`), then everything else is data (buffs, mutators, items, mobs, 30 rooms, one quest) that reuses shipped machinery: the instance system (B1), the `mutators → playerbuffids` hazard path (#21), `loot_pool` + `character.items` drops (#21), and the `room_interact` quest engine (Q76). Lore boundary **releases** in stage 7.3c — the buried thing is finally named in the records archive.

**Tech Stack:** Go 1.2x (GoMud engine); YAML data files (rooms/mobs/items/buffs/mutators/quests/dialogue); the codegraph MCP for symbol verification.

**Reference spec:** `docs/superpowers/specs/2026-07-01-crash-site-interior-design.md` (§5 the 30-room / 3-stage layout; §5b suppression aura; §6 loot + scour; §3 revelation).

**Depends on:** B1 (merged `526870f48`) — the instanced `crash_site_interior` zone (entry_room 6373, stubs 6373–6375), the Threshold-Keeper (9553), the disc-gated `open_instance_portal`. Also Plan A (`bc0252ab`) — `(*Character).ScourMutations(charges int)` already exists.

---

## ID Allocation (verified free, 2026-07-01)

| Kind | Block | Use |
|------|-------|-----|
| Rooms | **6376–6402** (27 new; +6373–6375 built = **30**) | 7.3a 6373–6381 · 7.3b 6382–6392 · 7.3c 6393–6402 |
| Mobs | **9554–9565** | constructs 9554–9560 · Warden-Prime 9561 · Core Guardian 9562 · (buffer 9563–9565) |
| Materials (40xxx) | **40169–40176** | reagents (warden-core, oracle-shard, hull relics) + trophies |
| Consumable (30xxx) | **30067** | the mutation-scour potion |
| Weapons (10xxx) | **10047–10049** | tech-relic loot_pool base items |
| Armor (20xxx) | **20091–20095** | tech-relic loot_pool base items |
| Buffs | **95–98** | 95 Dampened-aura · 96 Hull Discharge (deep) · 97 Arc Trap (defusable) · 98 warden on-hit (optional) |
| Mutators | string IDs | `hull_suppression`, `hull_discharge_deep` |
| Quest | **77** | "The Truth" (one-step revelation) |

> Before creating any YAML, re-run `python tools/id_inventory.py` to confirm nothing shifted. Item IDs are GLOBAL across sub-folders (the #21 20084 collision) — check the max across ALL `armor-20000/*` and `weapons-10000/*`.

---

## File Structure

**Engine (Phase 1–2):**
- `internal/buffs/buffspec.go` — add `Dampened Flag = `dampened`` to the flag enum.
- `internal/configs/config.balance.misc.go` — add `CrashSiteSuppressionFactor float64` (default 0.35).
- `internal/mutations/mutations.go` — add `DampenBonus(raw, factor float64) float64` helper.
- `internal/hooks/combat_shared_helpers.go` — gate `rawDmg` on `Dampened` (spell chokepoint).
- `internal/characters/combat.go` — gate the physical `GetDamageMultiplier` mutation call on `Dampened`.
- `internal/usercommands/drink.go` — special-case the scour-potion item ID → `ScourMutations`.
- Tests: `internal/mutations/dampen_test.go`, `internal/hooks/suppression_test.go` (or nearest existing test file), `internal/usercommands/drink_scour_test.go`.

**Data (Phase 3–7):**
- `_datafiles/world/dogmud/buffs/95-hull_dampening.yaml`, `96-hull_discharge_deep.yaml`, `97-arc_trap.yaml`, (`98-warden_overload.yaml`).
- `_datafiles/world/dogmud/mutators/hull_suppression.yaml`, `hull_discharge_deep.yaml`.
- `_datafiles/world/dogmud/items/materials-40000/40169..40176-*.yaml`; `consumables-30000/30067-*.yaml`; `weapons-10000/10047..10049-*.yaml`; `armor-20000/20091..20095-*.yaml`.
- `_datafiles/world/dogmud/mobs/crash_site_interior/9554..9562-*.yaml`.
- `_datafiles/world/dogmud/rooms/crash_site_interior/6376..6402.yaml` (+ edits to built 6373–6375 for exits/mutators/spawns).
- `_datafiles/world/dogmud/quests/77-the_truth.yaml`.
- Dialogue edits: `mobs/.../9553` behavior or dialogue (Threshold-Keeper reaction) + 1–2 existing NPCs (`questRequired: ["77-end"]`).

---

# PHASE 1 — Engine: the Chrysalis-suppression aura

**Design (verified by investigation):** a new buff flag `Dampened`, applied to every interior room via a mutator (`hull_suppression` → `playerbuffids: [95]`), checked at exactly two code chokepoints — the unified spell-damage function (covers players *and* mobs) and the physical damage-mutation multiplier. Raw stats are suppressed by the aura buff's own negative `statmods` (Willpower/Charisma, the belief stats). Config knob `CrashSiteSuppressionFactor` (0.35) sets how far suppressed effects collapse toward baseline.

### Task 1.1: Add the `Dampened` buff flag

**Files:** Modify `internal/buffs/buffspec.go` (the `Flag` enum, near line 86 after `ConditionMirror`).

- [ ] **Step 1: Add the flag constant.** In the `const (...)` flag block, after `ConditionMirror Flag = `condition-mirror``, add:

```go
	Dampened        Flag = `dampened` // #22 crash-site: Chrysalis suppression — mutation/spell power scaled down
```

- [ ] **Step 2: Build to verify it compiles.**

Run: `go build ./internal/buffs/`
Expected: exit 0.

- [ ] **Step 3: Commit.**

```bash
git add internal/buffs/buffspec.go
git commit -m "feat(crash-site): add Dampened buff flag for the suppression aura"
```

### Task 1.2: Add the suppression config knob

**Files:** Modify `internal/configs/config.balance.misc.go` (near `LootBudgetScalar`/`InstanceStatPoolCap`, ~line 275–280).

- [ ] **Step 1: Add the field** to the balance config struct (match the surrounding field style — tag + default handling):

```go
	// CrashSiteSuppressionFactor: inside the buried hull (#22), Chrysalis-granted
	// spell power and mutation combat bonuses are scaled to this fraction (the
	// "your gifts fail here" endgame twist). 0 = fully suppressed, 1 = no effect.
	CrashSiteSuppressionFactor ConfigFloat `yaml:"crashsitesuppressionfactor"`
```

- [ ] **Step 2: Add the default.** Find where the other Balance floats are defaulted (the `Validate()`/`SetDefaults` path for this struct) and add:

```go
	if c.CrashSiteSuppressionFactor == 0 {
		c.CrashSiteSuppressionFactor = 0.35
	}
```

> Verify the exact struct + default pattern with `codegraph_node` on `LootBudgetScalar` first — match its `ConfigFloat` type and defaulting convention exactly. If `ConfigFloat` isn't the type used, mirror whatever `LootBudgetScalar` uses.

- [ ] **Step 3: Build.** Run: `go build ./internal/configs/` — expect exit 0.
- [ ] **Step 4: Commit.**

```bash
git add internal/configs/config.balance.misc.go
git commit -m "feat(crash-site): add CrashSiteSuppressionFactor balance knob (default 0.35)"
```

### Task 1.3: Add the `DampenBonus` mutation helper (TDD)

**Files:** Create `internal/mutations/dampen_test.go`; modify `internal/mutations/mutations.go`.

- [ ] **Step 1: Write the failing test.** Create `internal/mutations/dampen_test.go`:

```go
package mutations

import "testing"

func TestDampenBonus(t *testing.T) {
	cases := []struct {
		name   string
		raw    float64
		factor float64
		want   float64
	}{
		{"no bonus stays put", 1.0, 0.35, 1.0},
		{"below baseline untouched", 0.8, 0.35, 0.8},
		{"bonus collapses toward baseline", 2.0, 0.35, 1.35}, // 1 + (2-1)*0.35
		{"full factor is a no-op", 1.5, 1.0, 1.5},
		{"zero factor removes the bonus", 1.5, 0.0, 1.0},
	}
	for _, c := range cases {
		if got := DampenBonus(c.raw, c.factor); got != c.want {
			t.Errorf("%s: DampenBonus(%v,%v)=%v want %v", c.name, c.raw, c.factor, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run it — verify it fails.** Run: `go test ./internal/mutations/ -run TestDampenBonus` — expect FAIL (`DampenBonus` undefined).

- [ ] **Step 3: Implement.** Add to `internal/mutations/mutations.go` (near `sumEffects`, ~line 350):

```go
// DampenBonus pulls a mutation MULTIPLIER's bonus toward the 1.0 baseline by
// `factor` (0 = remove the bonus entirely, 1 = untouched). Values at or below
// 1.0 (no bonus / a penalty) are returned unchanged. Used by the #22 crash-site
// suppression aura at the mutation call sites where the bearer is Dampened.
func DampenBonus(raw, factor float64) float64 {
	if raw <= 1.0 {
		return raw
	}
	return 1.0 + (raw-1.0)*factor
}
```

- [ ] **Step 4: Run — verify it passes.** Run: `go test ./internal/mutations/ -run TestDampenBonus` — expect PASS.
- [ ] **Step 5: Commit.**

```bash
git add internal/mutations/mutations.go internal/mutations/dampen_test.go
git commit -m "feat(crash-site): DampenBonus helper — collapse a mutation multiplier's bonus"
```

### Task 1.4: Gate spell damage on `Dampened` (the clean chokepoint)

**Files:** Modify `internal/hooks/combat_shared_helpers.go` (`calcSpellDamageForCharacter`, the `rawDmg` computation ~line 40).

- [ ] **Step 1: Verify the site.** `codegraph_node calcSpellDamageForCharacter --includeCode` — confirm `rawDmg` is an `int` computed near line 40 and `caster *characters.Character` is a parameter.

- [ ] **Step 2: Insert the gate** immediately after `rawDmg` is computed (before the weapon/gear multiplier lines):

```go
	// #22 crash-site: inside the buried hull, belief-driven power is suppressed.
	if caster != nil && caster.HasBuffFlag(buffs.Dampened) {
		factor := float64(configs.GetBalanceConfig().CrashSiteSuppressionFactor.Float())
		rawDmg = int(float64(rawDmg) * factor)
		if rawDmg < 1 {
			rawDmg = 1
		}
	}
```

> Match the existing imports/accessors: confirm how the file already reads balance config (grep the file for `GetBalanceConfig`) and how `HasBuffFlag` + `buffs` are referenced elsewhere in `internal/hooks`. If `.Float()` isn't how `ConfigFloat` is read, use the accessor `LootBudgetScalar` uses.

- [ ] **Step 3: Build.** Run: `go build ./internal/hooks/` — expect exit 0.

- [ ] **Step 4: Add a focused test.** In the nearest existing hooks test file (or create `internal/hooks/suppression_test.go`), assert that a caster with the `Dampened` buff flag produces lower spell damage than one without, at the same stats. Model setup on an existing spell-damage test (find one with `codegraph_search calcSpellDamage` → its test). If no test harness exists for this path, add a minimal one constructing a `*characters.Character`, adding a buff carrying `Dampened`, and comparing `calcSpellDamageForCharacter` output with/without. Run it, expect PASS.

- [ ] **Step 5: Commit.**

```bash
git add internal/hooks/combat_shared_helpers.go internal/hooks/suppression_test.go
git commit -m "feat(crash-site): suppress spell damage for Dampened casters"
```

### Task 1.5: Gate the physical mutation damage-multiplier on `Dampened`

**Files:** Modify `internal/characters/combat.go` (the call site using `mutations.GetDamageMultiplier(c.Mutations)`).

- [ ] **Step 1: Locate the site.** `codegraph_callers GetDamageMultiplier` → find the call in `internal/characters/combat.go` where `c` (the character) is in scope. Confirm the exact expression (e.g. `dm := mutations.GetDamageMultiplier(c.Mutations)`).

- [ ] **Step 2: Wrap the result** so a Dampened bearer's mutation damage bonus collapses:

```go
	dm := mutations.GetDamageMultiplier(c.Mutations)
	if c.HasBuffFlag(buffs.Dampened) {
		dm = mutations.DampenBonus(dm, configs.GetBalanceConfig().CrashSiteSuppressionFactor.Float())
	}
```

(Adapt the variable name to the actual code; keep the downstream use of the value identical.)

- [ ] **Step 3: Build.** Run: `go build ./internal/characters/` — expect exit 0.
- [ ] **Step 4: Boot-compile the whole tree.** Run: `go build ./...` — expect exit 0.
- [ ] **Step 5: Commit.**

```bash
git add internal/characters/combat.go
git commit -m "feat(crash-site): suppress mutation damage multiplier for Dampened bearers"
```

> **Scope note (documented, not a gap):** raw stats (Str/Dex/etc.) are suppressed by the aura buff's negative `statmods` (Task 3.1), and spell + physical-mutation power by the two gates above. Deeper per-getter suppression (natural armor, every resistance, regen) is deliberately deferred — these three levers already deliver the "your gifts fail here" feel without threading a scale through all ~30 context-free mutation getters. If playtest says it's not felt enough, extend by wrapping additional getters with `DampenBonus` the same way.

---

# PHASE 2 — Engine: the mutation-scour potion delivery

**Design (verified):** `(*Character).ScourMutations(charges int)` exists. JS scripting is gone, so deliver via a **potion special-cased by item ID in `drink.go`**, exactly like the Bloom Wafer (40108) block at `drink.go:218`. Deliberate (`drink` verb) → safe from accidental wipes; portable reward dropped in the medical bay (7.3b).

### Task 2.1: The scour-potion item

**Files:** Create `_datafiles/world/dogmud/items/consumables-30000/30067-catalyst_of_unmaking.yaml`.

- [ ] **Step 1: Create the item.** Model on an existing potion/consumable (read `consumables-30000/30066-*.yaml` for the schema). Name must be canonical Title Case, no leading article (`casing.Title`). Filename = `30067-catalyst_of_unmaking.yaml`.

```yaml
itemid: 30067
name: Catalyst of Unmaking
description: |
  A heavy phial of something the buried place made — clear, and colder than
  cold, and utterly still. Where every other draught you have carried tugs
  faintly at the change in your blood, this one is silent, a held breath. To
  drink it is to be, for one moment, only what you were born as: nothing added,
  nothing woken. What comes back afterward comes back hungrier.
type: potion
subtype: drinkable
uses: 1
not_salable: true
```

- [ ] **Step 2: YAML lint.** Run: `python3 -c "import yaml; yaml.safe_load(open('_datafiles/world/dogmud/items/consumables-30000/30067-catalyst_of_unmaking.yaml',encoding='utf-8')); print('OK')"` — expect OK.
- [ ] **Step 3: Commit.**

```bash
git add _datafiles/world/dogmud/items/consumables-30000/30067-catalyst_of_unmaking.yaml
git commit -m "feat(crash-site): the Catalyst of Unmaking (mutation-scour potion, item 30067)"
```

### Task 2.2: Wire the scour on drink (TDD)

**Files:** Modify `internal/usercommands/drink.go` (add an item-ID const + a special-case block mirroring the Bloom Wafer at ~line 218–257).

- [ ] **Step 1: Verify the pattern.** Read `internal/usercommands/drink.go` around the `bloomWaferItemId = 40108` const (~line 24) and its handling block (~218). Confirm how it acquires the target `user`, calls `user.Character.BloomSeedNewMutation(...)`, and messages the player.

- [ ] **Step 2: Write the failing test.** Create/extend `internal/usercommands/drink_scour_test.go` — a character with ≥2 mutations, after drinking item 30067, has `len(Character.Mutations)` reduced to only species intrinsics and `Character.MutationRerollBonus > 0`. If `drink.go` is hard to unit-test directly, instead write the test at the `ScourMutations` boundary already covered — but prefer asserting the drink path routes to it. Model on any existing `internal/usercommands/*_test.go`. Run it, expect FAIL.

- [ ] **Step 3: Implement.** Add the const near `bloomWaferItemId`:

```go
	catalystOfUnmakingItemId = 30067 // #22 crash-site: scours all mutations, biases the re-acquisition toward rare
```

Add the special-case block (mirror the Bloom Wafer block's shape — same guard on the drunk item's ID, same user acquisition):

```go
	if itemSpec.ItemId == catalystOfUnmakingItemId {
		charges := configs.GetConfig()... // if a knob exists; else a literal below
		user.Character.ScourMutations(scourRerollCharges)
		user.SendText(`<ansi fg="magenta">You drink the Catalyst. For one breath you are only what you were born as — every woken thing in your blood goes still and gone. Then the cold lets go, and the hunger comes back stronger than before.</ansi>`)
		// consume the potion + return, matching the Bloom Wafer flow
		return true, nil
	}
```

Define `const scourRerollCharges = 3` near the top (the number of rare-biased re-acquisition charges granted). Match the EXACT consume/return signature the surrounding `drink.go` handlers use (the Bloom Wafer block is the template — copy its consume + return shape verbatim, only swapping the effect).

- [ ] **Step 4: Run the test — expect PASS.** Run: `go test ./internal/usercommands/ -run Scour`.
- [ ] **Step 5: Build all.** Run: `go build ./...` — expect exit 0.
- [ ] **Step 6: Commit.**

```bash
git add internal/usercommands/drink.go internal/usercommands/drink_scour_test.go
git commit -m "feat(crash-site): drinking the Catalyst of Unmaking scours mutations (reroll x3)"
```

---

# PHASE 3 — Data: buffs + mutators (the aura + deep hazards)

### Task 3.1: The `hull_dampening` aura buff (95)

**Files:** Create `_datafiles/world/dogmud/buffs/95-hull_dampening.yaml`.

- [ ] **Step 1: Create the buff.** Carries the `dampened` flag + negative belief-stat statmods (Willpower/Charisma) for the raw-stat side of suppression. Long `triggercount` (the mutator re-applies each round, refreshing it). No tick damage (it's a debuff, not a DoT).

```yaml
buffid: 95
name: Hull Dampening
description: The buried place presses the change in your blood back down. Your gifts feel distant, your certainty thin.
triggerrate: 1 round
triggercount: 5
flags:
  - dampened
statmods:
  willpower: -30
  charisma: -30
start_user_text: The air here is wrong in a way you feel in your blood — the change in you goes quiet, and your gifts pull away like a tide going out.
start_room_text: "{source} falters, as if something in them had briefly forgotten itself."
end_user_text: The pressure eases and the change stirs back to life in your blood.
```

- [ ] **Step 2: Filename check.** Confirm `ConvertForFilename("Hull Dampening")` = `hull_dampening` → file `95-hull_dampening.yaml`. Correct.
- [ ] **Step 3: YAML lint.** `python3 -c "import yaml; yaml.safe_load(open('_datafiles/world/dogmud/buffs/95-hull_dampening.yaml',encoding='utf-8')); print('OK')"`.
- [ ] **Step 4: Commit** (batch with 3.2–3.4).

### Task 3.2: Deep hazard buffs (96 discharge, 97 arc-trap)

**Files:** Create `buffs/96-hull_discharge_deep.yaml`, `buffs/97-arc_trap.yaml`. Model on buff 94 (Cold Discharge). Stronger than #21 (`tick_percent: -0.06`) since this is the endgame — use `-0.09`.

```yaml
# 96-hull_discharge_deep.yaml
buffid: 96
name: Hull Discharge
description: Deep in the buried place the dead machines still bite, harder than they did above.
triggerrate: 1 round
triggercount: 3
start_user_text: The dead metal wakes just enough to hate you, and a hard arc slams up through your bones.
start_room_text: "{source} convulses as the hull arcs against them."
end_user_text: The arcing fades — for now.
trigger_user_text: A savage arc bites up through you.
trigger_room_text: "{source} shudders as the hull arcs against them."
tick_pool: health
tick_percent: -0.09
tick_variance: 0.03
tick_min: 8
```

```yaml
# 97-arc_trap.yaml — fired by a defusable trapped exit (lock.trapbuffids)
buffid: 97
name: Arc Trap
description: A dormant ward, tripped — a lash of stored violence from the dead machinery.
triggerrate: 1 round
triggercount: 2
start_user_text: Something you crossed was still armed. A ward discharges in a white lash and the pain arrives a half-beat later.
start_room_text: "{source} trips a dormant ward and the corridor flares white."
end_user_text: The seared feeling dulls.
trigger_user_text: The ward's charge crawls through you again.
trigger_room_text: "{source} spasms as the ward's charge finds them."
tick_pool: health
tick_percent: -0.11
tick_variance: 0.04
tick_min: 10
```

- [ ] **Step 1:** Create both files. **Step 2:** YAML-lint both. **Step 3:** Commit with 3.1/3.3.

### Task 3.3: The mutators

**Files:** Create `mutators/hull_suppression.yaml`, `mutators/hull_discharge_deep.yaml`. Model on `mutators/hull_discharge.yaml`.

```yaml
# hull_suppression.yaml — applied to EVERY interior room → the aura
mutatorid: hull_suppression
descriptionmodifier:
  behavior: append
  text: The change in your blood goes quiet here, as though the place itself refused it.
playerbuffids: [95]
```

```yaml
# hull_discharge_deep.yaml — applied to hazard rooms only
mutatorid: hull_discharge_deep
descriptionmodifier:
  behavior: append
  text: The dead machinery in the walls crawls and stings, and something bites up through anyone who lingers.
playerbuffids: [96]
```

- [ ] **Step 1:** Create both. **Step 2:** YAML-lint. **Step 3:** Commit.

```bash
git add _datafiles/world/dogmud/buffs/95-hull_dampening.yaml _datafiles/world/dogmud/buffs/96-hull_discharge_deep.yaml _datafiles/world/dogmud/buffs/97-arc_trap.yaml _datafiles/world/dogmud/mutators/hull_suppression.yaml _datafiles/world/dogmud/mutators/hull_discharge_deep.yaml
git commit -m "feat(crash-site): suppression + deep-hazard buffs (95-97) and mutators"
```

---

# PHASE 4 — Content: items (reagents, tech-relic loot bases, trophies)

**Design:** legendary-craft **reagents** (the pinnacle-craft economy seed, `grey-relic`-family tags extending #21's 40166) drop as fixed `character.items` on constructs/bosses; **tech-relic loot bases** (weapons/armor) are the affix-scaled `loot_pool` bases the Keeper's gold rolls into; **trophies** are proof-of-kill flavor. Non-techie descriptions throughout.

### Task 4.1: Legendary reagents + trophies (40169–40176)

**Files:** Create under `items/materials-40000/`. Names canonical Title Case, no leading article. Every reagent carries a `component_tag`.

| ID | name | component_tag | role |
|----|------|--------------|------|
| 40169 | Warden Core | warden-core | dropped by wardens; pinnacle-craft reagent |
| 40170 | Oracle Shard | oracle-shard | from the records/oracle-stones; reagent |
| 40171 | Hull Filament | hull-filament | grey-material reagent |
| 40172 | Coldlight Cell | coldlight-cell | the emergency-light source; reagent |
| 40173 | Sealed Medical Relic | medical-relic | from the medical bay; reagent |
| 40174 | Warden-Prime Casing | grey-relic | rare (Warden-Prime trophy/reagent) |
| 40175 | Core Guardian Heart | grey-relic | ultra-rare (Core Guardian trophy/reagent) |
| 40176 | Fragment of the Sky-Before | — | quest/lore trophy (records; `not_salable: true`) |

- [ ] **Step 1:** Create each YAML (model on `40166` from #21 for a reagent; set `is_component: true` + `component_tag` on reagents so they auto-route to the component bag). Example (40169):

```yaml
itemid: 40169
name: Warden Core
description: |
  A fist-sized knot of the grey material, still faintly warm, that sat at the
  heart of one of the buried place's dead guardians. Something in it has not
  entirely stopped. The smiths who chase the pinnacle of their craft would give
  a great deal for one, and blanch to hold it.
type: material
is_component: true
component_tag: warden-core
```

- [ ] **Step 2:** YAML-lint all eight. **Step 3:** `python tools/id_inventory.py --type items` — confirm no collisions. **Step 4:** Commit.

### Task 4.2: Tech-relic loot bases (weapons 10047–10049, armor 20091–20095)

**Files:** Create under `items/weapons-10000/` and `items/armor-20000/`. These are the `loot_pool` bases the instance affixes (like the Oasis's 10027/20072). Give them Oasis-parity base stats + non-techie "relic" flavor. Cover slots UNSERVED elsewhere where possible (check what the Oasis + #21 Sentinel already provide; avoid duplicating BIS slots — but affix-scaled bases can overlap, they're randomized).

| ID | name | slot/type | note |
|----|------|-----------|------|
| 10047 | Warden's Lash | weapon (flail/whip subtype) | melee base |
| 10048 | Coldlight Lance | weapon (spear/caster subtype) | consider `wand`/`staff` for a caster base |
| 10049 | Relic Sidearm | weapon (ranged) | ranged base |
| 20091 | Hull-Plate Cuirass | armor body | |
| 20092 | Grey Warden Helm | armor head | |
| 20093 | Filament Weave Gloves | armor gloves | |
| 20094 | Coldlight Mantle | armor back/shoulders | |
| 20095 | Warden-Step Greaves | armor legs | |

- [ ] **Step 1:** Create each, modeling weapons on an existing endgame weapon (e.g. `10046` Ironhorn Warbow from #21) and armor on `20090` (Greyfield Striders). Set base `damage_multiplier`/mitigation to Oasis-parity. **Step 2:** YAML-lint. **Step 3:** id_inventory collision check. **Step 4:** Commit.

```bash
git add _datafiles/world/dogmud/items/materials-40000/4016*.yaml _datafiles/world/dogmud/items/materials-40000/4017*.yaml _datafiles/world/dogmud/items/weapons-10000/1004*.yaml _datafiles/world/dogmud/items/armor-20000/2009*.yaml
git commit -m "feat(crash-site): legendary reagents, trophies, and tech-relic loot bases"
```

---

# PHASE 5 — Content: mobs (constructs + two bosses)

**Design (verified):** re-species OFF orb (basedamage 0) → **earth_elemental (37, bd 6)** for the "made-thing" feel + thick-hide/iron-constitution intrinsics. Statpool difficulty is set by the **room `spawninfo` multiplier** (Phase 6), NOT the mob YAML — so mob YAML `statpool` here is a nominal floor only. Bosses get `loot_pool` (affix gear) + `character.items` (guaranteed reagents). Model every file on the #21 Sentinel (`mobs/eastern_highlands/9552-the_sentinel.yaml`).

### Task 5.1: Trash constructs (9554–9560)

**Files:** Create under `mobs/crash_site_interior/`. ~5–7 construct templates (maintenance drones, sentry units, ambulatory wardens). All `speciesid: 37`, `archetype: fighting`, `hostile: true`, `groups: [construct]`, `maxwander: 0` (or small patrol), non-techie names/descriptions. Nominal `statpool` (the instance multiplier overrides — but set ~300 as a floor). Give constructs `loot_pool` for affix drops + a low-chance reagent in `character.items`.

Exemplar (9554):

```yaml
mobid: 9554
zone: Crash Site Interior
behavior_archetype: aggro
aiprofile: brute
archetype: fighting
hostile: true
statpool: 300
maxwander: 0
groups: [construct]
idlecommands:
  - 'emote drifts on a column of cold light, its blank face turning as if it half-remembers what a face is for.'
  - ''
combatcommands:
  - 'emote strikes with the flat certainty of a thing that has never once doubted.'
  - ''
character:
  name: Hull Warden
  description: |
    A drifting shape of the grey material, limbed and eyeless, that keeps the
    dead corridors as it has kept them since before there were people to keep
    them from. It is patient. It is not cruel. It simply will not let you pass.
  speciesid: 37
  level: 1
  gold: 0
  items:
    - itemid: 40171
      dropchance: 20
loot_pool:
  - 20091
itemdropchance: 60
```

- [ ] **Step 1:** Create 5–7 construct templates (vary names/species-flavor; a couple can be `speciesid: 35` flesh_golem-analog "heavier" units, bd 8, for variety). **Step 2:** Confirm zone-folder name `crash_site_interior` matches `ConvertForFilename("Crash Site Interior")` (it does). **Step 3:** YAML-lint all. **Step 4:** id_inventory mobs check. **Step 5:** Commit.

### Task 5.2: Warden-Prime (mid-boss, 9561) + Core Guardian (final boss, 9562)

**Files:** Create `mobs/crash_site_interior/9561-warden_prime.yaml`, `9562-the_core_guardian.yaml`. Model on the Sentinel: `behavior_archetype: leader`, `aiprofile: brute`, `submission_policy: lethal`, `surrender_policy: never`, `spawnmutations: [large]`, `speciesid: 37`, `itemdropchance: 100`. Rich guaranteed reagent + trophy drops + a fat `loot_pool`.

Core Guardian `character.items` (guaranteed climax loot):

```yaml
  items:
    - itemid: 40175   # Core Guardian Heart (ultra-rare reagent/trophy)
      dropchance: 100
    - itemid: 40169   # Warden Core
      dropchance: 100
    - itemid: 30067   # Catalyst of Unmaking (the scour potion) — guaranteed from the final boss
      dropchance: 100
    - itemid: 40170   # Oracle Shard
      dropchance: 60
loot_pool:
  - 10047
  - 10048
  - 20091
  - 20092
```

Warden-Prime drops `40174` (Warden-Prime Casing, ~40%), a reagent, and a smaller loot_pool.

> **Design decision (scour delivery):** the Catalyst (30067) is a **guaranteed Core Guardian drop** — you must beat the whole dungeon to earn a scour. The medical-bay chamber (7.3b) is FLAVOR/foreshadowing only ("where the change was studied"), not a second scour source. This keeps the scour behind the full run and avoids a room-interact accidental-wipe hazard.

- [ ] **Step 1:** Create both boss files. **Step 2:** YAML-lint. **Step 3:** id_inventory check. **Step 4:** Commit.

```bash
git add _datafiles/world/dogmud/mobs/crash_site_interior/
git commit -m "feat(crash-site): construct wardens + Warden-Prime + Core Guardian bosses"
```

---

# PHASE 6 — Content: the 30 rooms (3 stages)

**Design:** rooms 6373–6402. **Every room** carries `mutators: [- mutatorid: hull_suppression]` (the aura). Hazard rooms ALSO carry `hull_discharge_deep`. Instance difficulty comes from each room's `spawninfo` **statpool multiplier** (trash 1–2, Warden-Prime 4, Core Guardian 6). Non-cartesian zone (maze OK). Cold-blue-white light prose; **lore boundary HELD until the 7.3c records room, where it RELEASES**. Continue the grey-corridor voice from B1's 6373–6375.

> **B1 fix carried in:** the interior is currently pitch-dark. Give the **entry room 6373** (and at least the main-path rooms) a light source so players can see — either bake ambient light into the zone (preferred: a room-level light flag/biome that renders lit) OR place a `coldlight` fixture. Simplest: set the main corridors' biome/description to a self-lit "cold emergency lighting" and confirm `look` renders (test in Phase 8). If the engine requires a lightsource item/buff for a room to be visible, apply an always-on room light (e.g. a `hull_suppression`-sibling mutator granting `buffs.EmitsLight`, or a fixture item). Resolve the exact mechanism in Step 1 below before authoring all 30.

### Task 6.0: Resolve the lighting mechanism

- [ ] **Step 1:** Determine how a room is made visible without a player light source. Check `zone-config.yaml`/room schema for a `dark`/`light`/`lit` field or biome behavior (`codegraph_search` for the darkness check that produced "You can't see anything!" — likely a room/biome `GetLighting` or `Dark()` path). Pick the mechanism (room field vs. ambient mutator granting `EmitsLight`). Document it in one sentence and use it consistently for lit rooms. Deep/hazard rooms MAY stay dark for tension if the party carries light — but the main path should be traversable. **Step 2:** Commit the finding as a comment in the zone-config or the plan's execution notes.

### Task 6.1: Stage 7.3a — The Breached Section (6373–6381, 9 rooms)

**Files:** Edit built `6373.yaml`, `6374.yaml`, `6375.yaml`; create `6376.yaml`–`6381.yaml`.

Room table (exits are compass; `→` = leads to). Linear-with-one-branch on-ramp:

| ID | title | exits | mutators | spawninfo (mob×mult) | notes |
|----|-------|-------|----------|----------------------|-------|
| 6373 | The Breach | E→6374 | hull_suppression | — | entry (built; add light + mutator) |
| 6374 | The Grey Corridor | W→6373, E→6375 | hull_suppression | 9554×1 | built; add spawn |
| 6375 | The Sealed Chamber | W→6374, E→6376 | hull_suppression | — | built; loot cache (containers) |
| 6376 | Cold-Light Junction | W→6375, E→6377, S→6378 | hull_suppression | 9554×1 | branch |
| 6377 | Storage Recess | W→6376 | hull_suppression | — | optional loot room (dead-end) |
| 6378 | The Navigation Alcove | N→6376, E→6379 | hull_suppression | — | **orbital display: "four shapes, one damaged"** noun beat |
| 6379 | Arced Passage | W→6378, E→6380 | hull_suppression, **hull_discharge_deep** | 9555×1 | first hazard room |
| 6380 | Warden Post | W→6379, E→6381 | hull_suppression | 9554×2, 9556×1 | first construct pack |
| 6381 | The Descent | W→6380, DOWN→6382 | hull_suppression | — | lift/bulkhead to 7.3b |

- [ ] **Step 1:** Edit 6373 (add light mechanism + `mutators`), 6374/6375 (add `mutators` + spawns, keep prose). **Step 2:** Create 6376–6381 with full non-techie prose + the `symbol`/`display` nouns for 6378. **Step 3:** YAML-lint all. **Step 4:** Commit.

### Task 6.2: Stage 7.3b — The Ruined Decks (6382–6392, 11 rooms)

**Files:** Create `6382.yaml`–`6392.yaml`. The trap heart: maze-y, warden packs, peak hazard density, 2–3 optional side rooms, the medical bay + fabrication bay anchors, Warden-Prime gate.

| ID | title | exits | mutators | spawninfo | notes |
|----|-------|-------|----------|-----------|-------|
| 6382 | Lower Landing | UP→6381, N→6383, E→6386 | hull_suppression | 9554×2 | hub |
| 6383 | Buckled Deck | S→6382, N→6384 | hull_suppression, hull_discharge_deep | — | hazard corridor |
| 6384 | The Medical Bay | S→6383, E→6385 | hull_suppression | 9557×1 | **anchor: "where the change was studied"** — chrysalis-native foreshadow noun; flavor only |
| 6385 | Specimen Recess | W→6384 | hull_suppression | — | optional loot (reagents) |
| 6386 | Trapped Conduit | W→6382, E→6387 | hull_suppression | — | **defusable trapped exit** (east lock.trapbuffids [97]) |
| 6387 | The Fabrication Bay | W→6386, N→6388 | hull_suppression | 9558×1, 9554×1 | **anchor: reagent-rich loot** |
| 6388 | Collapsed Junction | S→6387, E→6389, N→6391 | hull_suppression, hull_discharge_deep | 9555×2 | maze node |
| 6389 | Warden Nest | W→6388, E→6390 | hull_suppression | 9554×2, 9556×2 | heavy pack |
| 6390 | Optional: The Silent Vault | W→6389 | hull_suppression | 9559×1 | risk/reward (a lone tough warden + rich loot) |
| 6391 | The Prime's Approach | S→6388, N→6392 | hull_suppression, hull_discharge_deep | — | gauntlet corridor |
| 6392 | The Warden-Prime's Hold | S→6391, UP/N→6393 | hull_suppression | **9561×4** (Warden-Prime) | **MID-BOSS**; gates 7.3c |

- [ ] **Step 1:** Create all 11 with prose. Author the medical-bay noun to *foreshadow* the truth (the change was studied here) WITHOUT naming it yet — boundary still held. **Step 2:** Add the defusable trap on 6386→E (schema: `exits: { east: { roomid: 6387, lock: { difficulty: <high>, trapbuffids: [97] } } }` — verify exact lock schema against a #21 trapped exit). **Step 3:** YAML-lint. **Step 4:** Commit.

### Task 6.3: Stage 7.3c — The Command Section (6393–6402, 10 rooms)

**Files:** Create `6393.yaml`–`6402.yaml`. The climax: heaviest defenses, Core Guardian, THE REVELATION (records archive — lore boundary RELEASES), command deck, signal array, sealed shuttle bay.

| ID | title | exits | mutators | spawninfo | notes |
|----|-------|-------|----------|-----------|-------|
| 6393 | The Command Approach | S→6392, N→6394 | hull_suppression, hull_discharge_deep | 9555×2, 9556×1 | |
| 6394 | The Sealed Threshold | S→6393, N→6395 | hull_suppression | — | ominous quiet before the boss |
| 6395 | The Core Guardian's Vault | S→6394, N→6396 | hull_suppression | **9562×6** (Core Guardian) | **FINAL BOSS** |
| 6396 | The Command Deck | S→6395, E→6397, W→6398 | hull_suppression | — | hub of the deep interior |
| 6397 | The Records Archive | W→6396 | hull_suppression | — | **THE REVELATION** (Quest 77 `room_interact`; oracle-stones; boundary RELEASES) |
| 6398 | The Oracle Gallery | E→6396, N→6399 | hull_suppression | — | the "wall of light — the sky-before" beat |
| 6399 | The Signal Array | S→6398, E→6400 | hull_suppression | — | **the moons hook** — reachable, understood, NOT activatable |
| 6400 | The Sealed Shuttle Bay | W→6399 | hull_suppression | — | the deferred shuttle stub (locked, "not yet") |
| 6401 | Optional: The Drift Vault | (off 6396, e.g. 6396 gains an exit) | hull_suppression | 9559×1 | final risk/reward loot |
| 6402 | Optional: The Long Dark | (off 6393 or 6388) | hull_suppression, hull_discharge_deep | 9560×1 | secret/optional |

> Adjust the two optional rooms' attach points so every room is reachable and exits reciprocal (non_cartesian relaxes coordinate checks but keep exits sane). Confirm the entry (6373) → … → 6395 (boss) → 6397 (records) main path is unbroken.

- [ ] **Step 1:** Create all 10. **7.3c is where the buried thing is NAMED** — the records/oracle prose delivers: crashed colony vessel from another world; the Chrysalis is native to Gaius and infected the colonists; the bloodline "immunity" was genetic chance; three more ships wait in orbit ("the moons"). Signal array = understood-but-deferred. **Step 2:** YAML-lint all. **Step 3:** Commit.

```bash
git add _datafiles/world/dogmud/rooms/crash_site_interior/
git commit -m "feat(crash-site): the 30-room 3-stage interior (7.3a/b/c) — the finale, boundary releases"
```

---

# PHASE 7 — Content: the revelation quest (77) + NPC reactions

**Design (verified — zero engine work):** a one-step quest whose `room_interact` in the records archive (6397) grants `77-end` once (gated `missing: ["77-end"]`), fires the scripted revelation `send_text`, and optionally `give_item: 40176` (Fragment of the Sky-Before). A few existing NPCs get a reactive dialogue node gated `questRequired: ["77-end"]`. Reuses the Q76 template exactly.

### Task 7.1: Quest 77 "The Truth"

**Files:** Create `_datafiles/world/dogmud/quests/77-the_truth.yaml`. Model on `quests/76-the_disc.yaml`.

- [ ] **Step 1:** Create the quest. One step (`77-end`). The `room_interact` trigger:

```yaml
questid: 77
name: The Truth
# one-step revelation; granted by reading the records archive
triggers:
  - event: room_interact
    room: 6397
    noun: records
    conditions:
      missing: ["77-end"]
    actions:
      - grant: "77-end"
      - give_item: 40176
      - send_text: |
          <the scripted revelation prose — the crashed colony vessel, the
          Chrysalis native to Gaius, the bloodline's immunity as chance, the
          three ships waiting in orbit. Threshold of the whole game's mystery.
          Write it as the oracle-stones speaking / the wall of light showing.>
      - room_text: "{source} goes very still before the wall of moving light, and does not speak for a long moment."
```

> The `noun: records` must exist on room 6397 (add `records`, and ideally `oracle-stones`, nouns pointing at the archive). Add a SECOND `room_interact` on `noun: oracle-stones` (or `archive`) with `has: ["77-end"]` giving re-readable flavor text (so returning players can re-read without re-firing the one-time beat).

- [ ] **Step 2:** Validate flags/tokens. **Step 3:** YAML-lint. **Step 4:** Confirm no `flags:` block is referenced without declaration (this quest uses tokens, not quest-flags, so none needed). **Step 5:** Commit.

### Task 7.2: NPC reactions to `truth-known`

**Files:** Edit the Threshold-Keeper (`behaviors/eastern_highlands/9553-*.yaml` OR a dialogue file if she has one) + 1–2 existing NPCs (e.g. a Greenford or New Plymouth authority/temple figure).

- [ ] **Step 1:** Add to the Threshold-Keeper a reactive line for returning players who now know — gated on the token. If she's behavior-tree-only, add a `player_ask`/greeting branch conditioned on the quest token (verify the behavior-tree condition for "player has quest token" exists; if not, use a dialogue node instead). A safe minimal approach: give her a dialogue file with a root variant `questRequired: ["77-end"]` ("You have the look of someone who went all the way in. Most don't come back with their eyes like that.").
- [ ] **Step 2:** Add a top-of-list reactive tree node to ONE established NPC (pick a temple/authority NPC in New Plymouth or Greenford) with `questRequired: ["77-end"]`, triggers including `quest`/`task` per SOP, first-person voice: "You carry something behind your eyes now. Be careful who you show it to." This is the seeded "you have a heretic's eyes" beat.
- [ ] **Step 3:** Keep it to 2–3 NPCs total (spec says SEED, don't build the hunted-heretic system). **Step 4:** YAML-lint any edited files. **Step 5:** Commit.

```bash
git add _datafiles/world/dogmud/quests/77-the_truth.yaml _datafiles/world/dogmud/mobs/eastern_highlands/9553-*.yaml <edited-npc-files>
git commit -m "feat(crash-site): Quest 77 The Truth — the revelation + seeded heretic reactions"
```

---

# PHASE 8 — Integration: boot, harness E2E, docs/memory

### Task 8.1: Clean boot verification

- [ ] **Step 1:** Nuke instance saves: `rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*`.
- [ ] **Step 2:** `go build -o C:/tmp/dogmud-cs.exe .` — expect exit 0.
- [ ] **Step 3:** Boot and watch for clean load: rooms ~1319 (was 1292 + 27), mobs ~589, items/buffs counts up, `ValidateAllFlags OK`, `ValidateZoneConsistency errors=0 mode=panic` (crash_site is non_cartesian so it's skipped, but EH edits must stay clean), **0 panics/casing errors**. Grep the boot log for `panic:|casing|did not end in|loadedCount`.
- [ ] **Step 4:** Fix any load-time failures (casing, filename mismatch, undeclared quest flag, bad mutator ref) and re-boot until clean.

### Task 8.2: Harness end-to-end run

- [ ] **Step 1:** Boot with AI port (55555). Drive the `mudagent` harness as smoketester (has the Attuned Disc + gold; already at EH 6372 or send there).
- [ ] **Step 2:** `ask keeper crash 300` → enter the portal → verify:
  - **Lighting:** `look` in 6373 renders (not "You can't see anything!").
  - **Suppression:** cast a damage spell inside vs. a known baseline — confirm reduced damage (or at least that the Hull Dampening buff shows on `status`/`buffs` and Willpower is lowered). Confirm mutations feel suppressed (physical mutation-damage bearer hits softer).
  - **Combat:** a construct pack is a real threat at 300g; the Warden-Prime (6392) and Core Guardian (6395) are hard but beatable by a geared master (smoketester is buffed — expect them tough-but-winnable; note if trivial or impossible).
  - **Hazards:** a `hull_discharge_deep` room ticks damage; the 6386 trapped exit fires the arc trap on a failed pick and can be `defuse`d.
  - **Loot:** the Core Guardian drops the Catalyst (30067) + reagents; affix gear rolls from loot_pool.
  - **Scour:** `drink catalyst of unmaking` → mutations cleared to intrinsics, `MutationRerollBonus` set (check via `status`/mutation display, or re-acquire and confirm rare bias).
  - **Revelation:** `examine records` in 6397 fires the one-time beat once; a second `examine` gives re-read flavor, not a re-grant.
- [ ] **Step 2b:** Record findings in `tools/playtest/reports/2026-07-01-local-crash-site-b2.md`.
- [ ] **Step 3:** Tear down harness + server.

### Task 8.3: Calibration note + docs/memory

- [ ] **Step 1:** From the harness run, note any tuning owed (statpool multipliers, hazard tick %, suppression factor). Fold the crash-site numbers into the **owed arc-wide calibration pass** (Cascade #20 + EH #21 + #22) rather than perfecting now — the smoketester is OP, so "trivial" readings are expected; flag anything that reads *impossible* or *broken*.
- [ ] **Step 2:** Update `docs/superpowers/plans/` progress + the memory files: `project_zone_expansion_redesign.md` (add a "#22 PLAN B2 COMPLETE" block — 30 rooms, suppression aura, scour, revelation, bosses, loot; lore boundary released) and `MEMORY.md` top status. Mark the lore boundary as RELEASED for #22 interior.
- [ ] **Step 3:** Merge `feature/crash-site-b2` → master `--no-ff`.

---

## Self-Review (completed inline)

- **Spec coverage:** §5 30-room/3-stage → Phase 6 ✅; §5b suppression aura → Phase 1 ✅; §6 loot + reagents → Phases 4–5 ✅; §6 mutation-scour → Phase 2 + Task 5.2 delivery ✅; §3 revelation + `truth-known` + seeded consequence → Phase 7 ✅; §4 signal array/shuttle deferred hook → rooms 6399/6400 ✅; construct `damage_multiplier`/species fix → Task 5.1 (species 37) ✅.
- **Deferred (per spec, correctly NOT built):** the full hunted-heretic system (seeded only, 7.2); signal-array activation + shuttle + moon mega-zones (stub rooms only); arc combat calibration (folded into the owed pass, 8.3).
- **Type/name consistency:** buff flag `dampened` (enum `Dampened`) used identically in 1.1/1.4/1.5/3.1; config `CrashSiteSuppressionFactor` (0.35) in 1.2/1.4/1.5; scour item 30067 in 2.1/2.2/5.2; quest token `77-end` in 7.1/7.2; species 37 throughout Phase 5. IDs cross-checked against the allocation table.
- **Known open decision resolved in-plan:** lighting mechanism (Task 6.0 must resolve before authoring all rooms); scour delivery chosen as a guaranteed Core-Guardian potion drop (Task 5.2), medical bay is flavor.

## Execution Handoff

Plan saved to `docs/superpowers/plans/2026-07-01-crash-site-B2-interior-content.md`. Two execution options:

1. **Subagent-Driven (recommended)** — fresh subagent per task, two-stage review between tasks, fast iteration. Content phases (4–7) can pre-allocate ID blocks per the table for safe parallel dispatch.
2. **Inline Execution** — batch execution with checkpoints in this session.
