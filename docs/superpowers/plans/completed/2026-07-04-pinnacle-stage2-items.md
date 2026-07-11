# Pinnacle Items Stage 2: The Nine Items — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Author the nine pinnacle item YAMLs, the permanent-haste worn
buff, and the two sentient-item voice files, on top of the merged Stage 1
primitives — live-verifiable via admin `item spawn` and a harness playtest.

**Architecture:** Two small engine additions first (a `staminamax` statmod
key; item-driven aggro-pull + on_kill voice emission), then pure content:
item YAMLs in a new `items/materials-40000/` directory, one buff YAML, two
voice YAMLs. Mechanical fields are fully specified below; descriptions are
authored creative work written to a brief, in world.md voice.

**Tech Stack:** Go (2 engine tasks), YAML data files, admin smoke commands,
GoMud playtest harness.

**Spec:** `docs/superpowers/specs/2026-07-04-pinnacle-chase-items-design.md`
(sections 5.1-5.9). **Author reference:** `docs/schemas/pinnacle-items.md`.

**Branch:** `feature/pinnacle-stage2-items` off `master`.

```bash
git checkout master && git checkout -b feature/pinnacle-stage2-items
```

---

## Locked IDs (allocated via id_inventory 2026-07-04 — do NOT reassign)

| ID | Item | File (in `_datafiles/world/dogmud/items/materials-40000/`) |
|---|---|---|
| 40181 | Phial of Second Birth | `40181-phial_of_second_birth.yaml` (**hardcoded in drink.go**) |
| 40182 | Vitalis Bandolier | `40182-vitalis_bandolier.yaml` |
| 40183 | The Blackrazor | `40183-the_blackrazor.yaml` |
| 40184 | Wayfarer's Bottomless Pack | `40184-wayfarers_bottomless_pack.yaml` |
| 40185 | Aegis of Mockery | `40185-aegis_of_mockery.yaml` |
| 40186 | Thornwall Harness | `40186-thornwall_harness.yaml` |
| 40187 | Seething Prism | `40187-seething_prism.yaml` |
| 40188 | Zephyr Treads | `40188-zephyr_treads.yaml` |
| 40189 | Staff of the Hollow Choir | `40189-staff_of_the_hollow_choir.yaml` |

Buff **98** = Zephyr's Alacrity (permanent haste). Buff 99 reserved spare.
Voices: `blackrazor`, `aegis` at `_datafiles/world/dogmud/itemvoices/`.

Filenames follow `ConvertForFilename()` (lowercase, a-z/0-9, apostrophes
dropped, other chars → underscore) — e.g. `Wayfarer's Bottomless Pack` →
`wayfarers_bottomless_pack`. **Verify against the loader on first boot.**

## Calibration context (current ceilings, verified 2026-07-04)

Melee `damage_multiplier` ceiling 1.50 (Heavy Greatsword); staff
`spell_damage_multiplier` ceiling 1.80 (Edrin's); body `physical_mitigation`
ceiling 16 (Hull-Plate Cuirass); `blockrating` ceiling 22 (Windstone Aegis);
back `weight_reduction` ceiling 0.50. Crash Site BIS uses `rarity_tier: 82`
(beyond-vendor marker) — all nine use it. All nine set `not_salable: true`
(spec guardrail 4: the sink must not leak through vendors). Haste is a buff
FLAG (`flags: [haste]`) → 1.5x swing count (`HasteSwingMultiplier`), binary.

## Description briefs — creative authoring rules (apply to every item task)

Each `description:` is a folded block scalar (`>-`), ~68-72 col wrap,
world.md voice (post-crash, Chrysalis, the grey material lore where noted),
NO mechanical numbers, NO lore-boundary leaks (never name the ship/moons
outside revelation-gated content). Each task gives a 2-3 sentence brief;
the implementer writes 5-8 lines of prose to it. Also author `namesimple`
(one noun) and, where given, `displayname`.

---

### Task 1: Engine — `staminamax` statmod key

The Zephyr Treads need a real stamina-pool boost. Only `healthmax` has a
statmod key today; `StaminaMax.Mods` reads no statmods
(internal/characters/validate.go:90-93).

**Files:**
- Modify: `internal/statmods/statmods.go` (add key beside `HealthMax` at :32)
- Modify: `internal/characters/validate.go` (StaminaMax.Mods term, ~:90)
- Test: `internal/characters/staminamax_statmod_test.go` (new)

- [ ] **Step 1: Write the failing test**

```go
package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

func TestStaminaMaxStatmod(t *testing.T) {
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		999950: {ItemId: 999950, Name: "wind boots", Type: items.Feet,
			Statmods: map[string]int{"staminamax": 60}},
	})()

	c := New()
	c.Validate()
	base := c.StaminaMax.Value

	c.Equipment.Feet = items.New(999950)
	c.Validate()
	if c.StaminaMax.Value != base+60 {
		t.Fatalf("expected StaminaMax %d (+60), got %d", base+60, c.StaminaMax.Value)
	}
}
```

(Verify the ItemSpec statmods field name — it may be `StatMods
statmods.StatMods` with yaml `statmods`; match the real field. Verify
`items.Feet` type constant. If `New()`+`Validate()` needs stat seeding for
a nonzero base, use the `.Base` technique from
pool_reservation_pinnacle_test.go — the assertion is the +60 delta.)

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/characters/ -run TestStaminaMaxStatmod -v`
Expected: FAIL (delta is 0 — key unread)

- [ ] **Step 3: Implement**

`internal/statmods/statmods.go`, beside `HealthMax`:

```go
	StaminaMax StatName = `staminamax`
```

`internal/characters/validate.go`, in the `StaminaMax.Mods` computation
(mirror the HealthMax pattern exactly):

```go
	c.StaminaMax.Mods = int(rb.StaminaBase) +
		c.StatMod(string(statmods.StaminaMax)) +
		c.Stats.Strength.ValueAdj*int(rb.StaminaPerStrength) +
		c.Stats.Willpower.ValueAdj*int(rb.StaminaPerWillpower) +
		c.Stats.Vitality.ValueAdj*int(rb.StaminaPerVitality)
```

(YAGNI: do NOT add `convictionmax` — nothing needs it.)

- [ ] **Step 4: Run tests**

Run: `go test ./internal/characters/ ./internal/statmods/ -count=1` → green;
`go build ./...` clean.

- [ ] **Step 5: Commit**

```bash
git add internal/statmods/ internal/characters/
git commit -m "feat(pinnacle): staminamax statmod key (Zephyr Treads SP boost)"
```

---

### Task 2: Engine — item aggro-pull (`taunt_pull`) + on_kill voice lines

Two small wirings on existing primitives. (a) The Aegis's chatter must
mechanically pull aggro: `ForceTauntAggro`
(internal/characters/taunt_hold.go:15) exists but nothing item-driven calls
it. (b) The Blackrazor's on_kill lines are authored-but-silent: nothing
emits the `on_kill` voice event.

**Files:**
- Modify: `internal/items/itemspec.go` (one field)
- Modify: `internal/hooks/pinnacle_tick.go` (tickVoices taunt-pull)
- Modify: `internal/hooks/MobDeath_ItemProcs.go` (on_kill voice emission)
- Test: `internal/hooks/pinnacle_tick_test.go`, `internal/hooks/item_procs_test.go` (append)

- [ ] **Step 1: ItemSpec field**

In itemspec.go's pinnacle block:

```go
	TauntPull bool `yaml:"taunt_pull,omitempty"` // sentient chatter on_taunt also pulls the bearer's target's aggro (Aegis)
```

No validation needed (bool).

- [ ] **Step 2: Write the failing taunt-pull test**

Append to pinnacle_tick_test.go (reuse the room/mob fixture pattern from
TestProcAoeStun tests — SetInstanceForTest + room.AddMob; seed a voice with
an on_taunt line; seed the item with VoiceId + TauntPull; put the bearer in
combat with the mob via SetAggro but make the MOB aggro a DIFFERENT userId):

```go
func TestTickVoices_TauntPull(t *testing.T) {
	defer seedAllRegistries()()
	defer itemvoices.SeedVoicesForTest(map[string]*itemvoices.VoiceSpec{
		"testshield": {VoiceId: "testshield", Lines: map[string][]string{
			"on_taunt": {"Your mother smells of elderberries!"},
		}},
	})()
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		999951: {ItemId: 999951, Name: "mock shield", Type: items.Offhand,
			VoiceId: "testshield", TauntPull: true},
	})()
	// fixture: user in room with mob; user.Character.SetAggro(0, mobInstanceId, ...);
	// mob.Character aggro set to a DIFFERENT user id (e.g. 2).
	// Call the taunt-pull path deterministically (see Step 3's extracted func)
	// and assert mob.Character.Aggro.UserId == bearer's userId afterwards.
}
```

(The 15% chatter roll makes tickVoices non-deterministic — extract the pull
into `applyTauntPull(user *users.UserRecord, spec items.ItemSpec)` called
by tickVoices after an on_taunt line emits, and unit-test applyTauntPull
directly, mirroring how pickVoiceEvent was extracted. Fill in the fixture
per the existing aoe_stun tests.)

- [ ] **Step 3: Implement**

In pinnacle_tick.go, after an `on_taunt` line is emitted in tickVoices:

```go
		// The Aegis: the insult is also mechanical — pull the bearer's
		// current target onto the bearer (tank loop: taunt the mob off
		// your ally). Uses the taunt-hold plumbing so reactive re-aggro
		// can't immediately flip it back.
		if event == "on_taunt" && spec.TauntPull {
			applyTauntPull(user, spec)
		}
```

```go
// applyTauntPull forces the bearer's current combat target (a mob) to
// aggro the bearer, if it isn't already fighting them.
func applyTauntPull(user *users.UserRecord, spec items.ItemSpec) {
	c := user.Character
	if c.Aggro == nil || c.Aggro.MobInstanceId <= 0 {
		return
	}
	mob := mobs.GetInstance(c.Aggro.MobInstanceId)
	if mob == nil || mob.IsNonCombatant() {
		return
	}
	if mob.Character.Aggro != nil && mob.Character.Aggro.UserId == user.UserId {
		return // already fighting the bearer
	}
	holdRounds := int(configs.GetBalanceConfig().TauntHoldRounds)
	mob.Character.ForceTauntAggro(user.UserId, 0, holdRounds)
}
```

(Verify `ForceTauntAggro(userId, mobInstanceId, holdRounds)` signature at
taunt_hold.go:15-23 and the `TauntHoldRounds` knob name; verify
`c.Aggro.MobInstanceId` field name against the Aggro struct.)

- [ ] **Step 4: on_kill voice emission**

In MobDeath_ItemProcs.go, after the on_kill proc dispatch, emit the
weapon's on_kill line (paced by the same chatter cooldown so a kill spree
doesn't spam — reuse the `pinnacle_voice_next_round` key via the same
helper tickVoices uses; extract a tiny shared
`tryEmitVoice(user, room, spec, event) bool` if needed):

```go
		// Sentient weapons savor the kill (paced by the chatter cooldown).
		wspec := user.Character.Equipment.Weapon.GetSpec()
		if wspec.VoiceId != "" {
			tryEmitVoice(user, nil, wspec, "on_kill")
		}
```

Write `tryEmitVoice` in pinnacle_tick.go: checks `pinnacle_voice_next_round`
gate, emits via emitVoiceLine-style formatting with NO fallback (silent when
no authored line), arms the cooldown on emit, returns whether it emitted.
Refactor tickVoices to use it where sensible WITHOUT changing tickVoices
behavior (same one-line-per-round + chance gate semantics). Add a test:
seeded on_kill line emits once, second immediate kill is cooldown-silent.

- [ ] **Step 5: Run + commit**

Run: `go test ./internal/hooks/ ./internal/items/ -count=1` green;
`go build ./...` clean.

```bash
git add internal/items/ internal/hooks/
git commit -m "feat(pinnacle): taunt_pull item flag + on_kill voice emission"
```

---

### Task 3: Buff 98 (Zephyr's Alacrity) + Zephyr Treads (40188)

**Files:**
- Create: `_datafiles/world/dogmud/buffs/98-zephyrs_alacrity.yaml`
- Create: `_datafiles/world/dogmud/items/materials-40000/40188-zephyr_treads.yaml`

- [ ] **Step 1: Buff YAML** (worn-buff pattern per the lantern precedent —
buffs.go re-applies worn buffs on Validate when missing):

```yaml
buffid: 98
name: Zephyr's Alacrity
description: The world drags half a beat behind your feet.
triggerrate: 5 real minutes
triggercount: 1
flags:
  - haste
statmods:
  dexterity: 8
start_user_text: "The treads pulse — the world slows around your stride."
end_user_text: "Your stride settles back to mortal pace."
```

- [ ] **Step 2: Item YAML** (mechanical fields exact; description to brief):

```yaml
itemid: 40188
name: Zephyr Treads
namesimple: treads
description: >-
  [BRIEF: wind-quick boots stitched with quicksilver soles and windlace
  bindings; the air itself seems reluctant to slow them; faint storm-scent.
  Pinnacle craft — masterwork feel, no numbers, no lore leaks.]
type: feet
subtype: wearable
physical_mitigation: 3
statmods:
  staminamax: 90
  dexterity: 5
wornbuffids:
  - 98
weight: 1.0
rarity_tier: 82
value: 5000
not_salable: true
```

(Replace the BRIEF with authored prose per the description rules.)

- [ ] **Step 3: Boot smoke**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go build -o gomud_smoke.exe . && ./gomud_smoke.exe
```
Confirm clean load (buff + item counts up by 1 each, no panic — this also
proves the new `materials-40000/` directory is walked by the loader), then
kill the server and delete the exe. In-game (optional but preferred if the
harness admin account is handy): `item spawn 40188`, `get treads`, `wear
treads`, confirm the haste start text + `status` shows the SP boost.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/buffs/98-zephyrs_alacrity.yaml "_datafiles/world/dogmud/items/materials-40000/"
git commit -m "content(pinnacle): Zephyr Treads + permanent haste worn buff"
```

---

### Task 4: The Blackrazor (40183) + blackrazor voice

**Files:**
- Create: `_datafiles/world/dogmud/items/materials-40000/40183-the_blackrazor.yaml`
- Create: `_datafiles/world/dogmud/itemvoices/blackrazor.yaml`

- [ ] **Step 1: Item YAML**

```yaml
itemid: 40183
name: The Blackrazor
namesimple: blackrazor
description: >-
  [BRIEF: a two-handed greatsword edged in volcanic obsidian set over a
  void-quenched core; it drinks light the way it drinks everything else;
  the grip is always slightly warm, like something breathing. Ominous,
  hungry, quasi-legal masterwork. No numbers, no lore leaks.]
type: weapon
hands: 2
subtype: slashing
damage:
  basedamage: 10
  variance: 4
damage_multiplier: 3.75
speedmultiplier: 0.9
parryrating: 6
staminacost: 9
reserve_health_pct: 0.25
hunger_rounds: 50
hunger_drain_pct: 0.01
procs:
  - trigger: on_hit
    chance: 100
    effect: lifesteal
    params:
      ratio: 0.25
voice_id: blackrazor
statmods:
  strength: 6
weight: 6.0
rarity_tier: 82
value: 8000
not_salable: true
```

- [ ] **Step 2: Voice YAML** — author 4-6 lines per event, first-person,
hungry, darkly funny, distinct voice (it is ancient, vain, and starving).
Events to author: `on_equip`, `on_unequip`, `on_kill`, `on_idle`,
`on_hunger_warning`, `on_hunger_feeding`, `on_taunt`. (Skip `on_grudge` —
nothing emits it for weapons yet.) Shape:

```yaml
voiceid: blackrazor
lines:
  on_equip:
    - "Ahhhh. A hand again. Do keep it moving."
    - "You will do. For now."
    # + 2-4 more
  on_kill:
    - "Yes... YES. Another."
    - "It drinks well tonight."
    # + 2-4 more
  # ... every listed event, >=4 lines each
```

- [ ] **Step 3: Boot smoke** — same procedure as Task 3 Step 3 (itemvoices
loadedCount goes to 1; dangling-ref validation passes). Optional live:
`item spawn 40183`, wield it, confirm the health reservation clamps the
pool (status), kill something, watch for a kill line.

- [ ] **Step 4: Commit**

```bash
git add "_datafiles/world/dogmud/items/materials-40000/40183-the_blackrazor.yaml" _datafiles/world/dogmud/itemvoices/blackrazor.yaml
git commit -m "content(pinnacle): The Blackrazor + its voice"
```

---

### Task 5: Aegis of Mockery (40185) + aegis voice

**Files:**
- Create: `_datafiles/world/dogmud/items/materials-40000/40185-aegis_of_mockery.yaml`
- Create: `_datafiles/world/dogmud/itemvoices/aegis.yaml`

- [ ] **Step 1: Item YAML**

```yaml
itemid: 40185
name: Aegis of Mockery
namesimple: aegis
description: >-
  [BRIEF: a tall shield with a voice-amber boss that never quite shuts up;
  resonance lacquer carries its insults across a battlefield; dents from a
  hundred fights it talked its way into. The tank's shield — loud, proud,
  indestructible-feeling. No numbers.]
type: offhand
subtype: wearable
blockrating: 30
physical_mitigation: 14
magical_mitigation: 10
conviction_mitigation: 8
procs:
  - trigger: on_block
    chance: 10
    cooldown_rounds: 20
    effect: aoe_stun
voice_id: aegis
taunt_pull: true
weight: 8.0
rarity_tier: 82
value: 7000
not_salable: true
```

- [ ] **Step 2: Voice YAML** — `voiceid: aegis`. Events: `on_equip`,
`on_unequip`, `on_idle`, `on_taunt` (the star — 8+ insult lines, Monty
Python energy per the spec: "Your mother smells of elderberries, and your
footwork embarrasses her further."), `on_kill` (gloating). ≥4 lines per
event except on_taunt (≥8).

- [ ] **Step 3: Boot smoke** (as Task 3). Optional live: spawn/wield, fight
with a partymate-attacked mob nearby, watch taunt line + aggro flip; block
until the stun proc fires (chance 10 — may take a while; the unit tests
already prove it, don't grind).

- [ ] **Step 4: Commit**

```bash
git add "_datafiles/world/dogmud/items/materials-40000/40185-aegis_of_mockery.yaml" _datafiles/world/dogmud/itemvoices/aegis.yaml
git commit -m "content(pinnacle): Aegis of Mockery + its voice"
```

---

### Task 6: Vitalis Bandolier (40182) + Wayfarer's Bottomless Pack (40184)

**Files:**
- Create: `_datafiles/world/dogmud/items/materials-40000/40182-vitalis_bandolier.yaml`
- Create: `_datafiles/world/dogmud/items/materials-40000/40184-wayfarers_bottomless_pack.yaml`

- [ ] **Step 1: Bandolier YAML**

```yaml
itemid: 40182
name: Vitalis Bandolier
namesimple: bandolier
description: >-
  [BRIEF: a chest-worn strap of chrysalis filter-membrane and still-glass
  rosettes; four cradles that hold a potion at the exact moment of its
  peak forever; the wearer's pulse and the potions' shimmer sync up.
  Mageblood fantasy. No numbers.]
type: belt
subtype: wearable
is_bandolier: true
bandolier_capacity: 4
preserves_contents: true
ambient_potions: true
weight: 1.5
rarity_tier: 82
value: 7000
not_salable: true
```

- [ ] **Step 2: Pack YAML**

```yaml
itemid: 40184
name: Wayfarer's Bottomless Pack
namesimple: pack
description: >-
  [BRIEF: folded-space silk over a warden chassis-loom frame; it swallows
  cargo and hands back feathers; travelers' myth made real. Pure QoL joy.
  No numbers.]
type: back
subtype: wearable
weight_reduction: 0.99
weight: 1.0
rarity_tier: 82
value: 5000
not_salable: true
```

- [ ] **Step 3: Boot smoke** (as Task 3). Optional live: wear the bandolier,
`sort`/store 2 potions, confirm ambient buff text after the attunement
window (BandolierAttuneRounds 100 makes live confirmation slow — the unit
tests own this; just confirm storage routing works).

- [ ] **Step 4: Commit**

```bash
git add "_datafiles/world/dogmud/items/materials-40000/40182-vitalis_bandolier.yaml" "_datafiles/world/dogmud/items/materials-40000/40184-wayfarers_bottomless_pack.yaml"
git commit -m "content(pinnacle): Vitalis Bandolier + Wayfarer's Bottomless Pack"
```

---

### Task 7: Thornwall Harness (40186) + Seething Prism (40187)

**Files:**
- Create: `_datafiles/world/dogmud/items/materials-40000/40186-thornwall_harness.yaml`
- Create: `_datafiles/world/dogmud/items/materials-40000/40187-seething_prism.yaml`

- [ ] **Step 1: Harness YAML** (note: apply_condition magnitude is a flat
param, not Strength-scaled — documented deviation from spec 5.5, tuned
flat-high instead; carry the note into the commit body):

```yaml
itemid: 40186
name: Thornwall Harness
namesimple: harness
description: >-
  [BRIEF: riveted spike-plates over cured hide, every barb angled for the
  clinch; wearing it is a threat display; grapplers' pinnacle — anything
  that wraps around you regrets it. No numbers.]
type: body
subtype: wearable
physical_mitigation: 20
magical_mitigation: 8
conviction_mitigation: 6
statmods:
  strength: 6
  vitality: 6
procs:
  - trigger: on_grapple
    chance: 50
    cooldown_rounds: 5
    effect: apply_condition
    params:
      condition: 1
      duration: 6
      magnitude: 14
weight: 11.0
rarity_tier: 82
value: 6000
not_salable: true
```

- [ ] **Step 2: Prism YAML**

```yaml
itemid: 40187
name: Seething Prism
namesimple: prism
description: >-
  [BRIEF: a living crystal on a containment-lattice chain, wet-looking,
  faintly warm, visibly feeding on the wearer; under the skin around it,
  things shift. Deeply quasi-legal; Veyra won't say where the technique
  came from. No numbers.]
type: neck
subtype: wearable
reserve_health_pct: 0.15
reserve_stamina_pct: 0.15
reserve_conviction_pct: 0.15
mutation_tick_interval: 300
mutation_tick_chance: 10
mutation_rarity_floor: 5
weight: 0.5
rarity_tier: 82
value: 7000
not_salable: true
```

- [ ] **Step 3: Boot smoke** (as Task 3). Optional live: wear the prism,
confirm all three pools clamp (status).

- [ ] **Step 4: Commit**

```bash
git add "_datafiles/world/dogmud/items/materials-40000/40186-thornwall_harness.yaml" "_datafiles/world/dogmud/items/materials-40000/40187-seething_prism.yaml"
git commit -m "content(pinnacle): Thornwall Harness + Seething Prism"
```

---

### Task 8: Staff of the Hollow Choir (40189) + Phial of Second Birth (40181)

**Files:**
- Create: `_datafiles/world/dogmud/items/materials-40000/40189-staff_of_the_hollow_choir.yaml`
- Create: `_datafiles/world/dogmud/items/materials-40000/40181-phial_of_second_birth.yaml`

- [ ] **Step 1: Staff YAML**

```yaml
itemid: 40189
name: Staff of the Hollow Choir
namesimple: staff
description: >-
  [BRIEF: a conductor-core stave crowned with choir-focus gems and a
  hollowed voice-box that hums a chord just below hearing; casting through
  it feels like being sung through; it takes the conviction of others and
  gives it to you. No numbers.]
type: weapon
hands: 2
subtype: staff
damage:
  basedamage: 6
  variance: 2
damage_multiplier: 0.80
spell_damage_multiplier: 3.75
parryrating: 12
staminacost: 6
statmods:
  casting: 10
  spellcasting: 5
  manifestation: 5
procs:
  - trigger: on_spell_hit
    chance: 100
    cooldown_rounds: 3
    effect: steal_pool
    params:
      pool: 3
      amount_pct: 0.08
weight: 4.0
rarity_tier: 82
value: 8000
not_salable: true
```

- [ ] **Step 2: Phial YAML** — **itemid MUST be 40181** (hardcoded in
drink.go). MIRROR the Catalyst of Unmaking's YAML shape exactly for
type/subtype/uses (read
`_datafiles/world/dogmud/items/*/30067-*.yaml` first and copy its
type/subtype/uses/drinkable structure verbatim; only the identity fields
and description differ):

```yaml
itemid: 40181
name: Phial of Second Birth
namesimple: phial
description: >-
  [BRIEF: a crystalline decanter of unmaking distillate cut with
  first-bloom nectar; it smells like rain on nothing; drinking it ends
  every change the Chrysalis ever made and begins exactly one more.
  No numbers, no mechanics-speak.]
type: <copy from 30067>
subtype: <copy from 30067>
uses: 1
weight: 0.3
rarity_tier: 82
value: 6000
not_salable: true
```

(The `<copy from 30067>` markers are a read-then-fill instruction, not
placeholders — the catalyst's exact type/subtype are authoritative and
MUST be copied, not guessed, or drink.go's Drinkable gate won't fire.)

- [ ] **Step 3: Boot smoke** (as Task 3). Live REQUIRED for the phial (it's
the one item whose engine path ships in Stage 1): admin character — grant
2-3 mutations (`chrysalis`/admin mutation tooling or a scour+bloom cycle),
`item spawn 40181`, get, drink, confirm: all mutations cleared, exactly one
new mutation, `mutations` shows it, rarity feels rare-tier.

- [ ] **Step 4: Commit**

```bash
git add "_datafiles/world/dogmud/items/materials-40000/40189-staff_of_the_hollow_choir.yaml" "_datafiles/world/dogmud/items/materials-40000/40181-phial_of_second_birth.yaml"
git commit -m "content(pinnacle): Staff of the Hollow Choir + Phial of Second Birth"
```

---

### Task 9: Full-suite + boot verification + world-critic pass

**Files:**
- Modify (if needed): any YAML the critic flags
- Modify: `docs/schemas/pinnacle-items.md` (append the ID table from this plan's header)

- [ ] **Step 1:** `go test -timeout 300s -count=1 ./...` → green.
- [ ] **Step 2:** Instance-save wipe + boot test (as Task 3 Step 3): items
count +9, buffs +1, itemvoices loadedCount=2, zero panics. Kill server.
- [ ] **Step 3:** World-critic review of all nine descriptions + both voice
files against world.md (voice consistency, lore-boundary leaks, 68-72 col
wrap, no numbers). Fix findings inline.
- [ ] **Step 4:** Append the nine-item ID/file table to
`docs/schemas/pinnacle-items.md` under a "Stage 2 shipped items" heading.
- [ ] **Step 5: Commit**

```bash
git add _datafiles/ docs/schemas/pinnacle-items.md
git commit -m "content(pinnacle): Stage 2 polish — critic fixes + shipped-item registry"
```

---

### Task 10: Harness playtest (live verification)

Run from the coordinator session (not a code subagent): write
`tools/playtest/goals/pinnacle-stage2-items.yaml` with goals per item —
spawn/wear each of the nine via admin, verify: Treads haste text + SP boost;
Blackrazor reservation + lifesteal + hunger warning after idling + kill
line; Aegis taunt line + aggro pull + (unit-proven) stun; Bandolier storage
+ ambient after attunement; Pack weight reduction (encumbrance tier drops);
Harness grapple bleed; Prism triple reservation; Staff CP steal on spell
hit; Phial scour+single-rare-grant. Then:

```
/playtest local feature-tester pinnacle-stage2-items.yaml
```

Triage the report; fix content-level findings on the branch; engine-level
findings get their own review cycle.

---

## Explicitly OUT of Stage 2

- Reagents, drop tables, forage entries (Stage 3).
- Veyra, quests, recipes, components, acquisition of ANY kind (Stage 4) —
  in Stage 2 these items are admin-spawn-only.
- The bandolier "can't drink slotted potions" restriction (deferred at
  Stage 1; attunement is the cost).
- Strength-scaled grapple bleed (flat magnitude param documented above).
- on_grudge emission (no emitter yet; authored lines would be silent).
