# Pinnacle Items Stage 1: Engine Primitives — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the generic engine primitives (item procs, pool reservation,
sentient item voices, bandolier ambience, rarity-floored mutations,
self-craft enforcement) that the nine pinnacle items (Stage 2) sit on as
pure YAML.

**Architecture:** Data lives on `ItemSpec` (new yaml fields). Pure
selection/pacing logic lives in the `items` package. All dispatch that
touches characters/rooms/combat lives in `internal/hooks` (which already
imports everything and is imported by nothing) — proc firing from the
unified combat round, grapple tick, spell resolution, and a new pinnacle
per-round tick inside `UserRoundTick`. Per-character mutable state
(cooldowns, last kill, attunement) uses `Character.MiscData` (persisted,
existing accessors). Two config kill switches gate everything.

**Tech Stack:** Go, YAML data files, existing test scaffolding
(`seedRegistry` per package, `seedAllRegistries` in hooks/usercommands).

**Spec:** `docs/superpowers/specs/2026-07-04-pinnacle-chase-items-design.md`

**Verified integration points** (all file:line refs checked 2026-07-04):

| What | Where |
|---|---|
| Combat round orchestrator (all quadrants) | `internal/hooks/NewRound_DoCombat_unified.go:53` `handleCombatRound(atk, def actions.Actor, ...)` |
| Hit + result (incl. `DefenseUsed`) | `rollCombatAttack` (unified.go:131) → `combat.AttackPlayerVsMob` etc.; `AttackResult` |
| Kill attribution | `events.MobDeath` (`internal/events/eventtypes.go:279`, has `PlayerDamage map[int]int`) |
| Grapple per-round | `internal/hooks/Position_GrappleTick.go:254` `processGrapplePair(controller, controlled *characters.Character)` |
| Spell damage calc | `internal/hooks/combat_shared_helpers.go:36` `calcSpellDamageForCharacter(...)`; applied in `spell_resolution.go` `applyMobEffect_*` |
| Pool reservation (enchant precedent) | `internal/characters/validate.go:228` `GetPoolReservation(pool string, poolMax int) int`; clamps CURRENT pool at validate.go:132-159 |
| Worn buffs apply/expire | `internal/characters/buffs.go:199-222` |
| Buff tick loop | `internal/buffs/buffs.go:259` `Trigger()`, fired at `internal/hooks/NewRound_UserRoundTick.go:145` |
| Per-player round tick (per-player loop) | `internal/hooks/NewRound_UserRoundTick.go:112-145` (`user`, `user.Character`, `room` in scope) |
| Drink special cases | `internal/usercommands/drink.go:36` (`catalystOfUnmakingItemId = 30067`), case at :266-277 |
| Potion aging | `internal/items/aging.go:31` `GetAgingPhase`, `:151` `CalcEffectiveAgingSpeed`; elapsed = `util.GetRoundCount() - item.CraftedRound` at 9 call sites |
| Bandolier storage | `Character.PotionItems` (`character.go:141`); `StoreItem` route at `internal/characters/inventory.go:158-165` |
| Craft skill check | `internal/usercommands/craft.go:124-128`; consume at :350 → `crafting.ConsumeIngredients` (`internal/crafting/crafting.go:187`) |
| MakerName stamped | `internal/hooks/NewRound_UserRoundTick.go:413-425` |
| MiscData accessors | `characters.Character.SetMiscData/GetMiscData` (`character.go:497/510`), persisted |
| Mutation weighted pool | `internal/mutations/mutations.go:219` `GetWeightedPool(owned, sp)`; NO rarity floor exists |
| Grant mutation | `internal/characters/character.go:792` `GrantRandomMutation()` |
| Scour | `internal/characters/mutation_scour.go:9` `ScourMutations(charges int)` |
| Stun buff | `AddBuff(84, false)` pattern (`internal/combat/submission_outcome.go:279`) |
| Party members | `parties.Get(userId)`, `(*Party).GetMembers()` |
| Non-combatant | `mob.IsNonCombatant()`, `mob.PlayerAttackImmune` |
| CP write pattern | direct field writes + clamp (`internal/actions/combat_taunt.go:220-222`) |
| Skill statmods | ALREADY WORK: `GetSkillLevel` adds `c.StatMod(skillName)` (`internal/characters/skills.go:177`) — staff skill bonuses are pure YAML, no task needed |
| Config toggles | `internal/configs/config.gameplay.go:3` (`FerriesEnabled` at :24); Balance validation pattern in `config.balance.*.go` |

**Branch:** `feature/pinnacle-stage1-engine-primitives` off `master`.

```bash
git checkout master && git checkout -b feature/pinnacle-stage1-engine-primitives
```

**MiscData key conventions used throughout this plan:**

- `pinnacle_proc_cd_<itemId>_<procIdx>` → uint64 round when cooldown expires
- `pinnacle_last_kill_round` → uint64 round of the character's last kill
- `pinnacle_hunger_anchor` → uint64 round hunger counts from
- `pinnacle_bandolier_attune_round` → uint64 round ambience reactivates
- `pinnacle_bandolier_buffs` → []int buff ids the bandolier applied
- `pinnacle_voice_next_round` → uint64 next round chatter may fire

---

### Task 1: Config knobs

**Files:**
- Modify: `internal/configs/config.gameplay.go` (GamePlay struct, near `FerriesEnabled` line 24)
- Modify: `internal/configs/config.balance.go` (Balance struct) and `internal/configs/config.balance.misc.go` (or wherever `validateMisc` lives — locate with `grep -n "func (b \*Balance) validateMisc" internal/configs/`)
- Test: `internal/configs/configs_pinnacle_test.go`

- [ ] **Step 1: Write the failing test**

```go
package configs

import "testing"

func TestPinnacleConfigDefaults(t *testing.T) {
	b := Balance{}
	b.Validate()
	if b.BandolierAttuneRounds <= 0 {
		t.Fatalf("BandolierAttuneRounds default expected >0, got %d", b.BandolierAttuneRounds)
	}
	if b.SentientChatterCooldownRounds <= 0 {
		t.Fatalf("SentientChatterCooldownRounds default expected >0, got %d", b.SentientChatterCooldownRounds)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/configs/ -run TestPinnacleConfigDefaults -v`
Expected: FAIL (fields undefined — compile error is the failure here)

- [ ] **Step 3: Implement**

In `config.gameplay.go`, after `FerriesEnabled`:

```go
	PinnacleItemsEnabled ConfigBool `yaml:"PinnacleItemsEnabled"` // Master toggle: sentient/ambient/hunger/mutation item ticks
	ItemProcsEnabled     ConfigBool `yaml:"ItemProcsEnabled"`     // Toggle: item proc firing (on_hit/on_block/etc.)
```

In `config.balance.go` Balance struct:

```go
	BandolierAttuneRounds         ConfigInt `yaml:"BandolierAttuneRounds"`         // Rounds of re-attunement after bandolier contents change (default 100)
	SentientChatterCooldownRounds ConfigInt `yaml:"SentientChatterCooldownRounds"` // Min rounds between sentient item lines (default 20)
```

In the misc validate function (same file as other `if b.X <= 0` defaults):

```go
	if b.BandolierAttuneRounds <= 0 {
		b.BandolierAttuneRounds = 100
	}
	if b.SentientChatterCooldownRounds <= 0 {
		b.SentientChatterCooldownRounds = 20
	}
```

In `_datafiles/config.yaml` under GamePlay, add (defaults on so local
smoke tests exercise them; prod can flip off):

```yaml
  PinnacleItemsEnabled: true
  ItemProcsEnabled: true
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/configs/ -run TestPinnacleConfigDefaults -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/configs/ _datafiles/config.yaml
git commit -m "feat(pinnacle): config knobs — PinnacleItemsEnabled, ItemProcsEnabled, tuning defaults"
```

---

### Task 2: ItemSpec fields + load-time validation

**Files:**
- Modify: `internal/items/itemspec.go` (ItemSpec struct — add near `WornBuffIds` line 241; find the spec's `Validate()` method in the same file)
- Test: `internal/items/itemspec_pinnacle_test.go`

- [ ] **Step 1: Write the failing test**

```go
package items

import "testing"

func TestItemProcValidation(t *testing.T) {
	spec := &ItemSpec{
		ItemId: 999901, Name: "test proc item", Type: Weapon,
		Procs: []ItemProc{{Trigger: "on_hit", Chance: 25, Effect: "lifesteal", Params: map[string]float64{"ratio": 0.25}}},
	}
	if err := spec.Validate(); err != nil {
		t.Fatalf("valid proc rejected: %v", err)
	}

	bad := &ItemSpec{
		ItemId: 999902, Name: "bad trigger", Type: Weapon,
		Procs: []ItemProc{{Trigger: "on_sneeze", Chance: 25, Effect: "lifesteal"}},
	}
	if err := bad.Validate(); err == nil {
		t.Fatal("invalid trigger accepted")
	}

	badEffect := &ItemSpec{
		ItemId: 999903, Name: "bad effect", Type: Weapon,
		Procs: []ItemProc{{Trigger: "on_hit", Chance: 25, Effect: "explode"}},
	}
	if err := badEffect.Validate(); err == nil {
		t.Fatal("invalid effect accepted")
	}

	badReserve := &ItemSpec{ItemId: 999904, Name: "bad reserve", Type: Weapon, ReserveHealthPct: 1.5}
	if err := badReserve.Validate(); err == nil {
		t.Fatal("reserve pct > 1 accepted")
	}
}

func TestProcsFor(t *testing.T) {
	spec := &ItemSpec{Procs: []ItemProc{
		{Trigger: "on_hit", Chance: 100, Effect: "lifesteal"},
		{Trigger: "on_block", Chance: 10, Effect: "aoe_stun"},
	}}
	if got := spec.ProcsFor("on_hit"); len(got) != 1 || got[0].Effect != "lifesteal" {
		t.Fatalf("ProcsFor(on_hit) = %+v", got)
	}
	if got := spec.ProcsFor("on_kill"); len(got) != 0 {
		t.Fatalf("expected empty, got %+v", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/items/ -run "TestItemProcValidation|TestProcsFor" -v`
Expected: FAIL (compile error — ItemProc undefined)

- [ ] **Step 3: Implement**

In `itemspec.go`, define next to the ItemSpec struct:

```go
// ItemProc is a data-driven proc an item fires from a combat/round trigger.
// Dispatch lives in internal/hooks (import direction: hooks → items).
type ItemProc struct {
	Trigger        string             `yaml:"trigger"`                   // on_hit | on_kill | on_block | on_grapple | on_spell_hit
	Chance         int                `yaml:"chance"`                    // percent per trigger event (1-100)
	CooldownRounds int                `yaml:"cooldown_rounds,omitempty"` // internal cooldown, 0 = none
	Effect         string             `yaml:"effect"`                    // lifesteal | steal_pool | aoe_stun | apply_condition
	Params         map[string]float64 `yaml:"params,omitempty"`
}

var validProcTriggers = map[string]bool{
	"on_hit": true, "on_kill": true, "on_block": true, "on_grapple": true, "on_spell_hit": true,
}

var validProcEffects = map[string]bool{
	"lifesteal": true, "steal_pool": true, "aoe_stun": true, "apply_condition": true,
}
```

Add fields to ItemSpec (near WornBuffIds):

```go
	Procs                []ItemProc `yaml:"procs,omitempty"`                  // data-driven combat procs
	ReserveHealthPct     float64    `yaml:"reserve_health_pct,omitempty"`     // 0-1 fraction of HealthMax reserved while equipped
	ReserveStaminaPct    float64    `yaml:"reserve_stamina_pct,omitempty"`    // 0-1 fraction of StaminaMax reserved while equipped
	ReserveConvictionPct float64    `yaml:"reserve_conviction_pct,omitempty"` // 0-1 fraction of ConvictionMax reserved while equipped
	PreservesContents    bool       `yaml:"preserves_contents,omitempty"`     // bandolier: contents never age
	AmbientPotions       bool       `yaml:"ambient_potions,omitempty"`        // bandolier: slotted potion buffs always-on at Peak
	MutationTickInterval int        `yaml:"mutation_tick_interval,omitempty"` // rounds between mutation rolls while worn (0 = never)
	MutationTickChance   int        `yaml:"mutation_tick_chance,omitempty"`   // percent chance per roll
	MutationRarityFloor  int        `yaml:"mutation_rarity_floor,omitempty"`  // min mutation rarity in the pool (0 = no floor)
	VoiceId              string     `yaml:"voice_id,omitempty"`               // sentient item voice file id (itemvoices/)
	HungerRounds         int        `yaml:"hunger_rounds,omitempty"`          // rounds without a kill before the item feeds on the wielder (0 = never)
	HungerDrainPct       float64    `yaml:"hunger_drain_pct,omitempty"`       // fraction of HealthMax drained per hungry round
```

Add helper + validation. In the existing `Validate()` method body, append:

```go
	for idx, p := range spec.Procs {
		if !validProcTriggers[p.Trigger] {
			return fmt.Errorf("item %d proc %d: invalid trigger %q", spec.ItemId, idx, p.Trigger)
		}
		if !validProcEffects[p.Effect] {
			return fmt.Errorf("item %d proc %d: invalid effect %q", spec.ItemId, idx, p.Effect)
		}
		if p.Chance < 1 || p.Chance > 100 {
			return fmt.Errorf("item %d proc %d: chance must be 1-100", spec.ItemId, idx)
		}
	}
	for name, v := range map[string]float64{
		"reserve_health_pct": spec.ReserveHealthPct, "reserve_stamina_pct": spec.ReserveStaminaPct, "reserve_conviction_pct": spec.ReserveConvictionPct,
	} {
		if v != 0 && (v < 0 || v >= 1) {
			return fmt.Errorf("item %d: %s must be in [0,1), got %v", spec.ItemId, name, v)
		}
	}
```

(The snippets use `spec` as the receiver — rename to match the real
`Validate()` receiver in itemspec.go if it differs.)

```go
// ProcsFor returns the procs matching a trigger. Cheap; no allocation when empty.
func (i *ItemSpec) ProcsFor(trigger string) []ItemProc {
	if len(i.Procs) == 0 {
		return nil
	}
	var out []ItemProc
	for _, p := range i.Procs {
		if p.Trigger == trigger {
			out = append(out, p)
		}
	}
	return out
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/items/ -v -run "TestItemProcValidation|TestProcsFor"`
Expected: PASS. Also run `go build ./...` — expect clean.

- [ ] **Step 5: Commit**

```bash
git add internal/items/
git commit -m "feat(pinnacle): ItemSpec proc/reserve/ambient/voice/hunger fields + validation"
```

---

### Task 3: Pool reservation from ItemSpec percentages

Mirrors the enchantment reservation exactly: reservations clamp the
CURRENT pool (max unchanged) inside `Validate()`.

**Files:**
- Modify: `internal/characters/validate.go:228` (`GetPoolReservation`)
- Test: `internal/characters/pool_reservation_pinnacle_test.go`

- [ ] **Step 1: Write the failing test**

```go
package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

func TestItemSpecPoolReservation(t *testing.T) {
	defer items.SeedItemSpecsForTest(map[int]*items.ItemSpec{
		999910: {ItemId: 999910, Name: "hungry blade", Type: items.Weapon, Hands: 2, ReserveHealthPct: 0.25},
	})()

	c := New()
	c.HealthMax.Value = 400
	c.Health = 400
	c.Equipment.Weapon = items.New(999910)

	if res := c.GetPoolReservation("health", c.HealthMax.Value); res != 100 {
		t.Fatalf("expected 100 reserved (25%% of 400), got %d", res)
	}
	c.Validate()
	if c.Health != 300 {
		t.Fatalf("expected current health clamped to 300, got %d", c.Health)
	}
}
```

NOTE: `items.SeedItemSpecsForTest` may not exist — the items package
seeds via the unexported map in `items_test.go:13`. If there is no
exported seeder, add one alongside `mutations.SeedMutationsForTest`
(`internal/mutations/test_helpers.go:6` is the pattern):

```go
// internal/items/test_helpers.go
package items

// SeedItemSpecsForTest swaps the spec registry for tests. Returns a restore func.
func SeedItemSpecsForTest(specs map[int]*ItemSpec) func() {
	old := items
	items = specs
	return func() { items = old }
}
```

(Confirm the package-global map name — `items_test.go:13` assigns it;
match that name.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/characters/ -run TestItemSpecPoolReservation -v`
Expected: FAIL — reservation is 0 (only Chrysalis enchantments counted)

- [ ] **Step 3: Implement**

In `GetPoolReservation` (validate.go:228), inside the existing loop over
`c.Equipment.GetAllItems()` (or a parallel loop if the existing one
`continue`s early on non-enchanted items), add the spec-driven path:

```go
	for _, itm := range c.Equipment.GetAllItems() {
		spec := itm.GetSpec()
		if spec == nil {
			continue
		}
		var pct float64
		switch pool {
		case "health":
			pct = spec.ReserveHealthPct
		case "stamina":
			pct = spec.ReserveStaminaPct
		case "conviction":
			pct = spec.ReserveConvictionPct
		}
		if pct > 0 {
			total += int(math.Floor(float64(poolMax) * pct))
		}
	}
```

The existing clamp at validate.go:132-159 already applies the total to
current Health/Stamina/Conviction — no further wiring.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/characters/ -run TestItemSpecPoolReservation -v` → PASS
Run: `go test ./internal/characters/` → all green (regression check on enchant reservations)

- [ ] **Step 5: Commit**

```bash
git add internal/items/test_helpers.go internal/characters/
git commit -m "feat(pinnacle): ItemSpec reserve_*_pct pool reservations (mirrors enchant ReservePool)"
```

---

### Task 4: Proc engine — chance/cooldown gate + lifesteal + on_hit/on_kill dispatch

**Files:**
- Create: `internal/hooks/item_procs.go`
- Create: `internal/hooks/MobDeath_ItemProcs.go`
- Modify: `internal/hooks/NewRound_DoCombat_unified.go` (after the attack result is known in `handleCombatRound`)
- Modify: `internal/hooks/hooks.go` (or wherever listeners register — grep `MobDeath` registrations, e.g. `internal/hooks/MobDeath_BountyClaim.go` sibling files show the pattern)
- Test: `internal/hooks/item_procs_test.go`

- [ ] **Step 1: Write the failing test (gate + lifesteal math)**

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func TestProcGate_CooldownAndChance(t *testing.T) {
	defer seedAllRegistries()()
	c := characters.New()
	p := items.ItemProc{Trigger: "on_hit", Chance: 100, CooldownRounds: 10, Effect: "lifesteal"}

	if !procGateOpen(c, 12345, 0, p) {
		t.Fatal("100%% chance, no cooldown recorded — gate should open")
	}
	markProcCooldown(c, 12345, 0, p) // records now+10
	if procGateOpen(c, 12345, 0, p) {
		t.Fatal("gate should be closed during cooldown")
	}

	zero := items.ItemProc{Trigger: "on_hit", Chance: 0, Effect: "lifesteal"}
	_ = zero // Chance 0 is rejected by validation; gate treats <1 as never
}

func TestProcLifesteal(t *testing.T) {
	defer seedAllRegistries()()
	attacker := characters.New()
	attacker.HealthMax.Value = 200
	attacker.Health = 100

	healed := procLifesteal(attacker, 80, map[string]float64{"ratio": 0.25})
	if healed != 20 {
		t.Fatalf("expected 20 healed (25%% of 80), got %d", healed)
	}
	if attacker.Health != 120 {
		t.Fatalf("expected health 120, got %d", attacker.Health)
	}

	// clamps at max
	attacker.Health = 195
	procLifesteal(attacker, 80, map[string]float64{"ratio": 0.25})
	if attacker.Health != 200 {
		t.Fatalf("expected clamp at 200, got %d", attacker.Health)
	}
	_ = util.GetRoundCount()
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/hooks/ -run "TestProcGate|TestProcLifesteal" -v`
Expected: FAIL (functions undefined)

- [ ] **Step 3: Implement `internal/hooks/item_procs.go`**

```go
package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// procCooldownKey identifies one proc slot on one item for MiscData bookkeeping.
func procCooldownKey(itemId, procIdx int) string {
	return fmt.Sprintf("pinnacle_proc_cd_%d_%d", itemId, procIdx)
}

// procGateOpen rolls chance and checks the cooldown. It does NOT mark the
// cooldown — callers mark only when the effect actually executed.
func procGateOpen(c *characters.Character, itemId, procIdx int, p items.ItemProc) bool {
	if !bool(configs.GetConfig().GamePlay.ItemProcsEnabled) {
		return false
	}
	if p.CooldownRounds > 0 {
		if until, ok := c.GetMiscData(procCooldownKey(itemId, procIdx)).(uint64); ok {
			if util.GetRoundCount() < until {
				return false
			}
		}
	}
	if p.Chance < 100 && util.Rand(100) >= p.Chance {
		return false
	}
	return true
}

func markProcCooldown(c *characters.Character, itemId, procIdx int, p items.ItemProc) {
	if p.CooldownRounds > 0 {
		c.SetMiscData(procCooldownKey(itemId, procIdx), util.GetRoundCount()+uint64(p.CooldownRounds))
	}
}

// procLifesteal heals the attacker for ratio*damage, clamped to HealthMax.
// Returns the amount actually healed.
func procLifesteal(attacker *characters.Character, damage int, params map[string]float64) int {
	ratio := params["ratio"]
	if ratio <= 0 || damage <= 0 {
		return 0
	}
	amt := int(float64(damage) * ratio)
	if amt < 1 {
		amt = 1
	}
	return attacker.Heal(amt)
}
```

NOTE: confirm `characters.Character.Heal(int) int` exists (used at
`internal/actions/combat_drain.go:124` as `char.Heal(healAmt)`). If the
signature differs, match it. Confirm `GetConfig().GamePlay` accessor
name against `configs` package (`GetConfig()` used widely; the ferry
toggle reads show the exact pattern — grep `FerriesEnabled` readers).

MiscData yaml round-trips may deserialize numbers as `int`; make the
cooldown read defensive:

```go
// readMiscRound tolerates int/uint64/float64 from yaml round-tripping.
func readMiscRound(v any) (uint64, bool) {
	switch n := v.(type) {
	case uint64:
		return n, true
	case int:
		return uint64(n), true
	case float64:
		return uint64(n), true
	}
	return 0, false
}
```

Use `readMiscRound` in `procGateOpen` instead of the bare type assert.

- [ ] **Step 4: Run tests** → PASS, then commit

```bash
git add internal/hooks/item_procs.go internal/hooks/item_procs_test.go
git commit -m "feat(pinnacle): proc gate (chance+cooldown via MiscData) + lifesteal effect"
```

- [ ] **Step 5: Write failing dispatch test (on_hit from combat round)**

Add to `item_procs_test.go`:

```go
func TestDispatchOnHitProcs_Lifesteal(t *testing.T) {
	defer seedAllRegistries()()
	defer items.SeedItemSpecsForTest(map[int]*items.ItemSpec{
		999920: {ItemId: 999920, Name: "leech blade", Type: items.Weapon, Hands: 2,
			Procs: []items.ItemProc{{Trigger: "on_hit", Chance: 100, Effect: "lifesteal", Params: map[string]float64{"ratio": 0.25}}}},
	})()

	attacker := characters.New()
	attacker.HealthMax.Value = 200
	attacker.Health = 100
	attacker.Equipment.Weapon = items.New(999920)
	defender := characters.New()

	dispatchItemProcs("on_hit", attacker, defender, nil, 80)

	if attacker.Health != 120 {
		t.Fatalf("on_hit lifesteal expected 120 health, got %d", attacker.Health)
	}
}
```

- [ ] **Step 6: Implement the dispatcher in `item_procs.go`**

```go
// dispatchItemProcs fires all procs on the OWNER's relevant equipment for a
// trigger. owner = whoever the trigger belongs to (attacker for on_hit/
// on_kill/on_spell_hit, defender for on_block, either for on_grapple).
// other may be nil (on_kill after despawn). room may be nil where unknown
// (effects that need a room fetch it lazily via the owner's RoomId).
func dispatchItemProcs(trigger string, owner, other *characters.Character, room *rooms.Room, damage int) {
	if owner == nil {
		return
	}
	for _, itm := range procBearingItems(owner, trigger) {
		spec := itm.GetSpec()
		if spec == nil {
			continue
		}
		for idx, p := range spec.ProcsFor(trigger) {
			if !procGateOpen(owner, itm.ItemId, idx, p) {
				continue
			}
			executed := false
			switch p.Effect {
			case "lifesteal":
				executed = procLifesteal(owner, damage, p.Params) > 0
			case "steal_pool":
				executed = procStealPool(owner, other, p.Params)
			case "aoe_stun":
				executed = procAoeStun(owner, room, p.Params)
			case "apply_condition":
				executed = procApplyCondition(other, p.Params)
			}
			if executed {
				markProcCooldown(owner, itm.ItemId, idx, p)
				sendProcMessage(owner, other, spec, p)
			}
		}
	}
}

// procBearingItems narrows which equipment slots a trigger consults:
// weapon triggers read the weapon; on_block reads the offhand; on_grapple
// reads body armor. Keeps the per-swing cost to 1-2 spec lookups.
func procBearingItems(c *characters.Character, trigger string) []items.Item {
	switch trigger {
	case "on_hit", "on_kill", "on_spell_hit":
		return []items.Item{c.Equipment.Weapon}
	case "on_block":
		return []items.Item{c.Equipment.Offhand}
	case "on_grapple":
		return []items.Item{c.Equipment.Body}
	}
	return nil
}
```

For this task, stub the not-yet-built effects so it compiles (they are
implemented in Tasks 5-7 — these stubs return false, which means "did
not execute", and are replaced by the real implementations):

```go
func procStealPool(owner, other *characters.Character, params map[string]float64) bool { return false }
func procAoeStun(owner *characters.Character, room *rooms.Room, params map[string]float64) bool { return false }
func procApplyCondition(target *characters.Character, params map[string]float64) bool { return false }

// sendProcMessage sends descriptive (no raw numbers) flavor to the owner's
// user if they are a player. Effect-specific text; keep it generic here and
// let voice files (Task 9) carry personality.
func sendProcMessage(owner, other *characters.Character, spec *items.ItemSpec, p items.ItemProc) {
	// Implemented minimally: proc effects already produce combat text via
	// their own channels (heal desc, stun messages). Intentionally quiet here.
}
```

Check `Equipment` field names against `characters.Character` (the worn
selector `GetAllWornItems` exists at buffs.go:199; `Equipment.Weapon`/
`Offhand` confirmed in combat code; `Equipment.Body` — verify exact
field name with `grep -n "Body " internal/characters/equipment*.go`,
it may be `Equipment.Body` or similar; fix to match).

- [ ] **Step 7: Wire on_hit + on_kill**

**on_hit** — in `handleCombatRound` (`NewRound_DoCombat_unified.go:53`),
after the attack result exists (immediately after the
`rollCombatAttack`/damage-bonus phase; find where the result var `res`
holds `.Hit` and `.DamageToTarget`):

```go
	// Pinnacle item procs: attacker's weapon on_hit.
	if res != nil && res.Hit {
		dispatchItemProcs("on_hit", atk.GetCharacter(), def.GetCharacter(), room, res.DamageToTarget)
	}
```

(Match the real variable names in that function — the explorer confirmed
`atk, def actions.Actor` params and an `AttackResult` with `Hit` and
`DamageToTarget`; `actions.Actor` has `GetCharacter()`.)

**on_kill** — new file `internal/hooks/MobDeath_ItemProcs.go`, copying
the registration pattern of `MobDeath_BountyClaim.go`:

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// MobDeathItemProcs fires on_kill procs and records the last-kill round
// (hunger anchor) for every player with damage attribution on the kill.
func MobDeathItemProcs(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.MobDeath)
	if !typeOk {
		return events.Continue
	}
	for uid := range evt.PlayerDamage {
		user := users.GetByUserId(uid)
		if user == nil {
			continue
		}
		user.Character.SetMiscData("pinnacle_last_kill_round", util.GetRoundCount())
		dispatchItemProcs("on_kill", user.Character, nil, nil, 0)
	}
	return events.Continue
}
```

Register it where the sibling MobDeath listeners register (grep
`MobDeath_BountyClaim` registration — likely an `init()` or a central
`RegisterListeners` in the hooks package; copy exactly).

- [ ] **Step 8: Run tests + full package**

Run: `go test ./internal/hooks/ -run TestDispatchOnHitProcs -v` → PASS
Run: `go build ./... && go test ./internal/hooks/ ./internal/combat/` → green

- [ ] **Step 9: Commit**

```bash
git add internal/hooks/
git commit -m "feat(pinnacle): proc dispatcher + on_hit/on_kill wiring, last-kill tracking"
```

---

### Task 5: on_block + aoe_stun effect (party-safe)

**Files:**
- Modify: `internal/hooks/item_procs.go` (replace `procAoeStun` stub)
- Modify: `internal/hooks/NewRound_DoCombat_unified.go` (on_block dispatch)
- Test: `internal/hooks/item_procs_test.go` (append)

- [ ] **Step 1: Write the failing test**

```go
func TestProcAoeStun_PartySafe(t *testing.T) {
	defer seedAllRegistries()()
	// seedAllRegistries seeds buff specs; ensure buff 84 (stun stagger) is
	// present — if the seed map lacks it, extend the seed with:
	// 84: {BuffId: 84, Name: "Submission Stunned", RoundInterval: 1, TriggerCount: 1}
	room, mobIds := makeTestRoomWithMobs(t, 2) // helper below
	owner := characters.New()

	ok := procAoeStun(owner, room, map[string]float64{"stun_rounds": 2})
	if !ok {
		t.Fatal("aoe_stun should execute with hostile mobs present")
	}
	for _, mid := range mobIds {
		m := mobs.GetInstance(mid)
		if m == nil || !m.Character.HasBuff(84) {
			t.Fatalf("mob %d should be stunned", mid)
		}
	}
}
```

Write `makeTestRoomWithMobs` following the existing hooks test
scaffolding (`hooks_test.go:36` `seedAllRegistries`, plus however
existing hooks tests construct rooms+mobs — copy the nearest precedent,
e.g. any `Position_GrappleTick` or combat-round test constructing a room
with `rooms.LoadRoom`-compatible fixtures. If no precedent exists in
hooks tests, construct via the rooms test seeding pattern
`rooms_test.go:16` and `mobs.NewMobById` + add to room).

- [ ] **Step 2: Run to verify it fails** (`procAoeStun` stub returns false)

Run: `go test ./internal/hooks/ -run TestProcAoeStun -v` → FAIL

- [ ] **Step 3: Implement**

```go
// procAoeStun applies the stagger-stun buff (84) to every hostile,
// stun-eligible mob in the owner's room. Party members and players are
// never targeted; non-combatants and attack-immune mobs are skipped.
// Returns true if at least one target was stunned.
func procAoeStun(owner *characters.Character, room *rooms.Room, params map[string]float64) bool {
	if room == nil {
		room = rooms.LoadRoom(owner.RoomId)
		if room == nil {
			return false
		}
	}
	stunned := 0
	for _, mid := range room.GetMobs() {
		mob := mobs.GetInstance(mid)
		if mob == nil || mob.IsNonCombatant() || mob.PlayerAttackImmune {
			continue
		}
		_ = mob.Character.AddBuff(84, false)
		stunned++
	}
	if stunned > 0 {
		room.SendTextVisual(messaging.CategoryHitNaturalBlunt,
			`A concussive burst of force slams outward, staggering everything nearby!`)
	}
	return stunned > 0
}
```

Notes for the implementer:
- Buff 84 is the 1-round submission stagger (`internal/combat/submission_outcome.go:279`). `stun_rounds` param: apply the buff N times only if the buff spec supports duration stacking; if not (TriggerCount-based), applying once is correct — keep params for future tuning but don't fake duration. Check `_datafiles/world/dogmud/buffs/84-*.yaml` for its shape first.
- `room.GetMobs()` returns ALL mobs; charmed/companion mobs may belong to players. Check how companions are excluded elsewhere (grep `IsCharmed` in internal/mobs) and skip mobs charmed by the owner or their party (`parties.Get(userId)` → `GetMembers()`). If the owner is a mob (mob wielding the shield), skip the party logic.
- Messaging category: match the nearest room-wide combat effect precedent (grep `CategoryHitNaturalBlunt` in hooks; if absent use the category the taunt/stomp room messages use).

- [ ] **Step 4: Wire on_block dispatch**

In `handleCombatRound`, next to the on_hit dispatch from Task 4 — the
attack result carries `DefenseUsed` (`sendDefenseMessages` sets
`result.DefenseUsed = DefenseType(best.defenseType)` at
combat_helpers.go:859):

```go
	// Pinnacle item procs: defender's shield on_block.
	if res != nil && !res.Hit && string(res.DefenseUsed) == characters.DefenseBlock {
		dispatchItemProcs("on_block", def.GetCharacter(), atk.GetCharacter(), room, 0)
	}
```

(Verify the miss/`DefenseUsed` semantics: a blocked attack may register
as `Hit == false` with `DefenseUsed == "block"`, or as a mitigated hit —
read `runBestOfAllDefense` (combat_helpers.go:498) and
`sendDefenseMessages` (:857) to confirm which fields mark a successful
block, and gate on exactly that.)

- [ ] **Step 5: Run tests, then commit**

Run: `go test ./internal/hooks/ -run "TestProcAoeStun" -v` → PASS
Run: `go test ./internal/hooks/ ./internal/combat/` → green

```bash
git add internal/hooks/
git commit -m "feat(pinnacle): on_block dispatch + party-safe aoe_stun proc effect"
```

---

### Task 6: on_grapple + apply_condition (bleed)

**Files:**
- Modify: `internal/hooks/item_procs.go` (replace `procApplyCondition` stub)
- Modify: `internal/hooks/Position_GrappleTick.go:254` (`processGrapplePair`)
- Test: `internal/hooks/item_procs_test.go` (append)

- [ ] **Step 1: Write the failing test**

```go
func TestProcApplyCondition_Bleed(t *testing.T) {
	defer seedAllRegistries()()
	target := characters.New()
	target.Stats.Strength.ValueAdj = 100

	ok := procApplyCondition(target, map[string]float64{
		"condition": 1, // 1 = bleeding (see conditionFromParam)
		"duration":  6,
		"magnitude": 12,
	})
	if !ok {
		t.Fatal("apply_condition should execute")
	}
	if !target.HasCondition(characters.ConditionBleeding) {
		t.Fatal("target should be bleeding")
	}
}
```

(Verify the condition-query method name: `AddCondition` is confirmed at
`combat_drain.go:116` — `target.Char.AddCondition(characters.ConditionBleeding, 4, float64(mag), "drain")`.
Find the matching reader — grep `func (c \*Character) HasCondition` or
inspect the conditions API in internal/characters; adjust the assert.)

- [ ] **Step 2: Run to verify it fails** → FAIL (stub returns false)

- [ ] **Step 3: Implement**

```go
// procApplyCondition applies a condition to the target. Params:
// condition (1=bleeding), duration (rounds), magnitude (per-tick).
func procApplyCondition(target *characters.Character, params map[string]float64) bool {
	if target == nil {
		return false
	}
	dur := int(params["duration"])
	if dur < 1 {
		dur = 4
	}
	mag := params["magnitude"]
	if mag < 1 {
		mag = 2
	}
	switch int(params["condition"]) {
	case 1:
		target.AddCondition(characters.ConditionBleeding, dur, mag, "itemproc")
		return true
	}
	return false
}
```

- [ ] **Step 4: Wire on_grapple**

In `processGrapplePair` (Position_GrappleTick.go:254), after the outcome
switch (line ~293-309) — fire for BOTH participants every resolved
grapple round (spec: "either direction"); each side's body armor procs
against the other:

```go
	// Pinnacle item procs: spiked armor while grappling (both directions).
	dispatchItemProcs("on_grapple", controller, controlled, nil, 0)
	dispatchItemProcs("on_grapple", controlled, controller, nil, 0)
```

The proc's own `cooldown_rounds` + `chance` keep this from firing every
round — the Thornwall Harness YAML (Stage 2) tunes that.

- [ ] **Step 5: Run + commit**

Run: `go test ./internal/hooks/ -run TestProcApplyCondition -v` → PASS
Run: `go test ./internal/hooks/` → green

```bash
git add internal/hooks/
git commit -m "feat(pinnacle): on_grapple dispatch + apply_condition bleed proc"
```

---

### Task 7: on_spell_hit + steal_pool (conviction vampirism)

**Files:**
- Modify: `internal/hooks/item_procs.go` (replace `procStealPool` stub)
- Modify: `internal/hooks/spell_resolution.go` (dispatch at damage-apply sites)
- Test: `internal/hooks/item_procs_test.go` (append)

- [ ] **Step 1: Write the failing test**

```go
func TestProcStealPool_Conviction(t *testing.T) {
	defer seedAllRegistries()()
	caster := characters.New()
	caster.ConvictionMax.Value = 100
	caster.Conviction = 40
	target := characters.New()
	target.ConvictionMax.Value = 100
	target.Conviction = 50

	ok := procStealPool(caster, target, map[string]float64{"pool": 3, "amount_pct": 0.10})
	if !ok {
		t.Fatal("steal_pool should execute")
	}
	if target.Conviction != 40 { // 10% of target's max = 10 stolen
		t.Fatalf("target conviction expected 40, got %d", target.Conviction)
	}
	if caster.Conviction != 50 {
		t.Fatalf("caster conviction expected 50, got %d", caster.Conviction)
	}

	// clamps: target at 0, caster at max
	target.Conviction = 3
	caster.Conviction = 95
	procStealPool(caster, target, map[string]float64{"pool": 3, "amount_pct": 0.10})
	if target.Conviction != 0 || caster.Conviction != 98 {
		t.Fatalf("clamps wrong: target=%d caster=%d", target.Conviction, caster.Conviction)
	}
}
```

- [ ] **Step 2: Run to verify it fails** → FAIL

- [ ] **Step 3: Implement** (direct field writes + clamp — the taunt
pattern, `combat_taunt.go:220-222`; steal is capped by what the target
actually has)

```go
// procStealPool drains a pool from the target into the owner. Params:
// pool (1=health 2=stamina 3=conviction), amount_pct (fraction of the
// TARGET's pool max, capped by what they have).
func procStealPool(owner, target *characters.Character, params map[string]float64) bool {
	if owner == nil || target == nil {
		return false
	}
	pct := params["amount_pct"]
	if pct <= 0 {
		return false
	}
	switch int(params["pool"]) {
	case 3: // conviction
		amt := int(float64(target.ConvictionMax.Value) * pct)
		if amt < 1 {
			amt = 1
		}
		if amt > target.Conviction {
			amt = target.Conviction
		}
		if amt <= 0 {
			return false
		}
		target.Conviction -= amt
		owner.Conviction += amt
		if owner.Conviction > owner.ConvictionMax.Value {
			owner.Conviction = owner.ConvictionMax.Value
		}
		return true
	}
	return false
}
```

(Health/stamina variants: YAGNI — add cases only when an item needs
them. The switch is the extension point.)

- [ ] **Step 4: Wire on_spell_hit**

In `spell_resolution.go`, the damage-applying `applyMobEffect_*` funcs
each compute `dmg := calcSpellDamageForCharacter(...)` then subtract
target health (e.g. `:434-447`). Rather than touching every applier, add
the dispatch inside a shared helper IF one exists where damage lands; if
each applier writes health directly (they do), add after the damage
write in each HARM applier (grep `calcSpellDamageForCharacter`
call sites in spell_resolution.go — wire each caster-is-player site):

```go
	// Pinnacle item procs: caster's weapon on_spell_hit.
	if dmg > 0 {
		dispatchItemProcs("on_spell_hit", casterChar, &mob.Character, nil, dmg)
	}
```

(Variable names differ per applier — `user.Character` vs `casterChar`,
`mob.Character` vs player target; match each site. Cover both
mob-target and player-target appliers.)

- [ ] **Step 5: Run + commit**

Run: `go test ./internal/hooks/ -run TestProcStealPool -v` → PASS
Run: `go test ./internal/hooks/` → green

```bash
git add internal/hooks/
git commit -m "feat(pinnacle): on_spell_hit dispatch + steal_pool conviction vampirism"
```

---

### Task 8: require_own_components recipe flag

**Files:**
- Modify: `internal/crafting/crafting.go` (RecipeSpec + check helper)
- Modify: `internal/usercommands/craft.go` (call the check before starting the craft)
- Test: `internal/crafting/crafting_own_components_test.go`

- [ ] **Step 1: Write the failing test**

```go
package crafting

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

func TestRequireOwnComponents(t *testing.T) {
	defer items.SeedItemSpecsForTest(map[int]*items.ItemSpec{
		777701: {ItemId: 777701, Name: "hungering guard", Type: items.Object, ComponentTag: "hungering_guard", IsComponent: true},
	})()

	recipe := &RecipeSpec{
		RecipeId: "test-assembly", Name: "test assembly", Skill: "blacksmithing", SkillMinimum: 65,
		RequireOwnComponents: true,
		Ingredients:          []RecipeIngredient{{ItemTag: "hungering_guard", Quantity: 1}},
	}

	mine := items.New(777701)
	mine.MakerName = "Megalomania"
	theirs := items.New(777701)
	theirs.MakerName = "SomeoneElse"
	unmade := items.New(777701) // no maker at all

	if err := CheckOwnComponents(recipe, []items.Item{mine}, nil, "Megalomania"); err != nil {
		t.Fatalf("own component rejected: %v", err)
	}
	if err := CheckOwnComponents(recipe, []items.Item{theirs}, nil, "Megalomania"); err == nil {
		t.Fatal("foreign component accepted")
	}
	if err := CheckOwnComponents(recipe, []items.Item{unmade}, nil, "Megalomania"); err == nil {
		t.Fatal("maker-less component accepted")
	}

	// flag off → no restriction
	recipe.RequireOwnComponents = false
	if err := CheckOwnComponents(recipe, []items.Item{theirs}, nil, "Megalomania"); err != nil {
		t.Fatalf("flag off should not restrict: %v", err)
	}
}
```

(Check the real field names on ItemSpec for `ComponentTag`/`IsComponent`
— CLAUDE.md documents `is_component`; grep the yaml tags in itemspec.go
and match. If the crafting package's test file `crafting_test.go:10` has
its own seeding pattern, follow it instead of `SeedItemSpecsForTest`.)

- [ ] **Step 2: Run to verify it fails** → FAIL (field + func undefined)

- [ ] **Step 3: Implement**

In RecipeSpec (crafting.go:33-44), after `SkillMinimum`:

```go
	RequireOwnComponents bool `yaml:"require_own_components,omitempty"` // crafted-component ingredients must carry the crafter's MakerName
```

New function in crafting.go:

```go
// CheckOwnComponents enforces require_own_components: every ingredient that
// is itself a crafted component (carries a MakerName slot) must have been
// made by the crafter. Ingredients are matched by item_tag the same way
// HasIngredients matches them. Returns a player-presentable error.
func CheckOwnComponents(recipe *RecipeSpec, inv, componentInv []items.Item, crafterName string) error {
	if !recipe.RequireOwnComponents {
		return nil
	}
	pool := append(append([]items.Item{}, inv...), componentInv...)
	for _, ing := range recipe.Ingredients {
		for _, itm := range pool {
			spec := itm.GetSpec()
			if spec == nil || !tagMatches(spec, ing.ItemTag) {
				continue
			}
			if !spec.IsComponent {
				continue // bulk materials are exempt; only crafted components checked
			}
			if itm.MakerName != crafterName {
				return fmt.Errorf("the %s must be your own work — this one bears another maker's mark", spec.Name)
			}
		}
	}
	return nil
}
```

IMPORTANT implementation notes:
- `tagMatches` — reuse however `HasIngredients`/`ConsumeIngredients`
  (crafting.go:187) match `item_tag` against a spec (a `ComponentTag`
  field or tag list). Extract/share that matcher rather than
  reimplementing; if it's inline, factor a small `specMatchesTag` helper
  both paths call.
- This simple form rejects if ANY matching-tag item in inventory is
  foreign, even when the player also carries their own copy.
  Acceptable for v1 (pinnacle components are not farmed in bulk), but
  add a `// NOTE:` comment; the consume path picks first-match, so the
  strict check prevents the consume path silently using the foreign one.
- MakerName is only stamped at craft skill ≥ 30 and NOT on
  `IsComponent` outputs (`NewRound_UserRoundTick.go:423` —
  `!newSpec.IsComponent` excludes components!). **This means component
  outputs currently get NO MakerName — the flag cannot work without
  removing that exclusion.** Modify `NewRound_UserRoundTick.go:423` to
  also stamp components:

```go
	if newItem.CraftSkill >= 30 && newSpec.Type != items.Object {
		newItem.MakerName = user.Character.Name
	}
```

  (i.e. drop the `!newSpec.IsComponent` clause — verify with the user's
  intent if display side-effects appear; MakerName on components is
  harmless and enables provenance.)

In `craft.go`, right where ingredients are confirmed (after
`HasIngredients` at :141, before the activity starts):

```go
	if err := crafting.CheckOwnComponents(recipe, user.Character.Items, user.Character.ComponentItems, user.Character.Name); err != nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="red">%s</ansi>`, err.Error()))
		return true, nil
	}
```

- [ ] **Step 4: Run + commit**

Run: `go test ./internal/crafting/ -run TestRequireOwnComponents -v` → PASS
Run: `go test ./internal/crafting/ ./internal/usercommands/ ./internal/hooks/` → green (the MakerName stamping change can affect existing tests — fix any that asserted components have no maker)

```bash
git add internal/crafting/ internal/usercommands/craft.go internal/hooks/NewRound_UserRoundTick.go
git commit -m "feat(pinnacle): require_own_components recipe flag + MakerName on components"
```

---

### Task 9: Rarity-floored mutations + remort drink case

**Files:**
- Modify: `internal/mutations/mutations.go` (floored pool)
- Modify: `internal/characters/character.go` (rare grant helper)
- Modify: `internal/usercommands/drink.go` (remort case)
- Test: `internal/mutations/mutations_floor_test.go`, `internal/characters/character_mutation_rare_test.go`, extend `internal/usercommands/drink_scour_test.go`

- [x] **Step 0: Allocate the phial item ID**

Run: `python tools/id_inventory.py --alloc items 1`
Record the returned ID; it is used for `phialOfSecondBirthItemId` below
and MUST be the ID Stage 2 uses for the item YAML. Write it into this
plan file (replace `<PHIAL_ID>` everywhere) and into the spec's section
5.9 as a note. **ALLOCATED: 40181.**

- [ ] **Step 1: Write failing tests**

`internal/mutations/mutations_floor_test.go`:

```go
package mutations

import "testing"

func TestGetWeightedPoolWithFloor(t *testing.T) {
	defer SeedMutationsForTest(map[string]*MutationSpec{
		"common-1": {MutationId: "common-1", Name: "C1", Rarity: 2},
		"rare-1":   {MutationId: "rare-1", Name: "R1", Rarity: 7},
		"rare-2":   {MutationId: "rare-2", Name: "R2", Rarity: 8},
	})()

	pool := GetWeightedPoolWithFloor(map[string]int{}, nil, 5)
	for _, id := range pool {
		if id == "common-1" {
			t.Fatal("rarity floor 5 should exclude rarity-2 mutations")
		}
	}
	if len(pool) == 0 {
		t.Fatal("rare mutations should remain in the floored pool")
	}

	// floor 0 behaves like the unfloored pool
	full := GetWeightedPoolWithFloor(map[string]int{}, nil, 0)
	foundCommon := false
	for _, id := range full {
		if id == "common-1" {
			foundCommon = true
		}
	}
	if !foundCommon {
		t.Fatal("floor 0 should include commons")
	}
}
```

`internal/characters/character_mutation_rare_test.go`:

```go
package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mutations"
)

func TestGrantRandomMutationRare(t *testing.T) {
	defer seedScourTestMutations()() // existing helper: 3 commons rarity 2, 3 rares rarity 7
	c := New()
	c.SpeciesId = 1
	c.Mutations = map[string]int{}

	id := c.GrantRandomMutationRare(5)
	if id == "" {
		t.Fatal("expected a mutation granted")
	}
	if spec := mutations.GetMutation(id); spec == nil || spec.Rarity < 5 {
		t.Fatalf("granted %q below rarity floor", id)
	}
	if c.Mutations[id] != 1 {
		t.Fatal("mutation not recorded at level 1")
	}
}
```

- [ ] **Step 2: Run to verify both fail** (funcs undefined)

Run: `go test ./internal/mutations/ ./internal/characters/ -run "Floor|MutationRare" -v` → FAIL

- [ ] **Step 3: Implement**

`internal/mutations/mutations.go` — refactor `GetWeightedPool` (line
219) into a floored core; keep the old signature delegating:

```go
// GetWeightedPoolWithFloor is GetWeightedPool restricted to mutations at or
// above minRarity (0 = no floor). Used by the pinnacle remort potion and
// the Seething Prism's worn mutation tick.
func GetWeightedPoolWithFloor(owned map[string]int, sp *species.Species, minRarity int) []string {
	rarityBonus := calcRarityBonus(owned)
	pool := make([]string, 0, len(allMutations)*5)
	for id, spec := range allMutations {
		if spec.Rarity < minRarity {
			continue
		}
		if _, has := owned[id]; has {
			continue
		}
		if HasConflict(owned, id) {
			continue
		}
		if !spec.CanApplyTo(sp) {
			continue
		}
		weight := 11 - spec.Rarity - rarityBonus
		if weight < 1 {
			weight = 1
		}
		for i := 0; i < weight; i++ {
			pool = append(pool, id)
		}
	}
	return pool
}

func GetWeightedPool(owned map[string]int, sp *species.Species) []string {
	return GetWeightedPoolWithFloor(owned, sp, 0)
}
```

(Delete the old body — one implementation, delegated. Check
`CanApplyTo(nil)` is safe — it is: fail-open at mutations.go:249.)

`internal/characters/character.go` — next to `GrantRandomMutation`
(:792):

```go
// GrantRandomMutationRare grants one mutation from the rarity-floored
// weighted pool. Returns the granted id, or "" if none qualify.
func (c *Character) GrantRandomMutationRare(minRarity int) string {
	sp := species.GetSpecies(c.SpeciesId)
	pool := mutations.GetWeightedPoolWithFloor(c.Mutations, sp, minRarity)
	if len(pool) == 0 {
		return ""
	}
	mutId := mutations.RollAcquisition(pool)
	if mutId == "" {
		return ""
	}
	if c.Mutations == nil {
		c.Mutations = make(map[string]int)
	}
	c.Mutations[mutId] = 1
	c.Validate()
	return mutId
}
```

`internal/usercommands/drink.go` — next to the catalyst constants (:36):

```go
	phialOfSecondBirthItemId = <PHIAL_ID> // pinnacle remort potion (Stage 2 item)
	phialRarityFloor         = 5
```

Case beside the catalyst case (:266-277):

```go
	if itemSpec.ItemId == phialOfSecondBirthItemId {
		user.Character.ScourMutations(0)
		granted := user.Character.GrantRandomMutationRare(phialRarityFloor)
		if granted != "" {
			if spec := mutations.GetMutation(granted); spec != nil {
				user.SendText(messaging.CategoryWarning, fmt.Sprintf(
					`<ansi fg="magenta">Your flesh unwrites itself — every change the Chrysalis ever made dissolves. Then, from the stillness, something singular takes root: <ansi fg="yellow">%s</ansi>.</ansi>`, spec.Name))
			}
		} else {
			user.SendText(messaging.CategoryWarning,
				`<ansi fg="magenta">Your flesh unwrites itself — every change dissolves. The stillness holds; nothing new takes root.</ansi>`)
		}
	}
```

- [ ] **Step 4: Extend the drink test**

In `drink_scour_test.go`, add a test following
`TestDrink_CatalystOfUnmakingScoursMutations` (:20) — same fixture
approach with a spec whose ItemId is `phialOfSecondBirthItemId`; assert
prior mutations cleared AND exactly one new mutation present with
`Rarity >= 5`.

- [ ] **Step 5: Run + commit**

Run: `go test ./internal/mutations/ ./internal/characters/ ./internal/usercommands/ -run "Floor|MutationRare|SecondBirth|Scour" -v` → PASS

```bash
git add internal/mutations/ internal/characters/ internal/usercommands/ docs/superpowers/
git commit -m "feat(pinnacle): rarity-floored mutation pool + remort phial drink path"
```

---

### Task 10: Sentient item voices — loader + line selection

**Files:**
- Create: `internal/itemvoices/itemvoices.go`
- Create: `internal/itemvoices/itemvoices_test.go`
- Create: `_datafiles/world/dogmud/itemvoices/test-voice.yaml` (NO — see note: loader tests seed in-memory; the real voice files ship in Stage 2. Create the directory with a `.gitkeep` only.)
- Modify: the datafile-loading startup path (grep `LoadDataFiles` call sequence — e.g. where `mutations`/`enchantments` loaders are invoked at boot; likely `internal/gomud.go` or `world` bootstrap; copy the enchantments registration)

- [ ] **Step 1: Write the failing test**

```go
package itemvoices

import "testing"

func TestVoiceLineSelection(t *testing.T) {
	defer SeedVoicesForTest(map[string]*VoiceSpec{
		"blackrazor": {
			VoiceId: "blackrazor",
			Lines: map[string][]string{
				"on_kill":  {"Yes... YES. Another.", "It drinks well tonight."},
				"on_idle":  {"*a low obsidian hum*"},
				"on_hunger_warning": {"I hunger, bearer."},
			},
		},
	})()

	v := GetVoice("blackrazor")
	if v == nil {
		t.Fatal("voice not found")
	}
	line := v.Line("on_kill")
	if line != "Yes... YES. Another." && line != "It drinks well tonight." {
		t.Fatalf("unexpected line %q", line)
	}
	if v.Line("on_equip") != "" {
		t.Fatal("missing event should return empty string")
	}
	if GetVoice("nope") != nil {
		t.Fatal("unknown voice should be nil")
	}
}

func TestVoiceValidation(t *testing.T) {
	bad := &VoiceSpec{VoiceId: ""}
	if err := bad.Validate(); err == nil {
		t.Fatal("empty voiceid accepted")
	}
	badEvent := &VoiceSpec{VoiceId: "x", Lines: map[string][]string{"on_sneeze": {"a"}}}
	if err := badEvent.Validate(); err == nil {
		t.Fatal("unknown event key accepted")
	}
}
```

- [ ] **Step 2: Run to verify it fails** (package doesn't exist)

Run: `go test ./internal/itemvoices/ -v` → FAIL/no package

- [ ] **Step 3: Implement `internal/itemvoices/itemvoices.go`**

Follow the fileloader pattern from `internal/enchantments/enchantments.go`
(EnchantmentDef implements `Id() string`, `Filepath() string`,
`Validate() error`; a `LoadEnchantmentFiles`-style loader fills a
package map):

```go
// Package itemvoices loads sentient-item voice definitions: authored line
// pools keyed by event, one YAML per voice at
// _datafiles/world/dogmud/itemvoices/<voiceid>.yaml.
package itemvoices

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/util"
)

var validVoiceEvents = map[string]bool{
	"on_equip": true, "on_unequip": true, "on_kill": true, "on_idle": true,
	"on_hunger_warning": true, "on_hunger_feeding": true, "on_taunt": true, "on_grudge": true,
}

type VoiceSpec struct {
	VoiceId string              `yaml:"voiceid"`
	Lines   map[string][]string `yaml:"lines"`
}

func (v *VoiceSpec) Id() string       { return v.VoiceId }
func (v *VoiceSpec) Filepath() string { return util.FilePath(v.VoiceId + ".yaml") }

func (v *VoiceSpec) Validate() error {
	if v.VoiceId == "" {
		return fmt.Errorf("voiceid cannot be empty")
	}
	for event := range v.Lines {
		if !validVoiceEvents[event] {
			return fmt.Errorf("voice %q: unknown event %q", v.VoiceId, event)
		}
	}
	return nil
}

// Line returns a random line for the event, or "" if none authored.
func (v *VoiceSpec) Line(event string) string {
	pool := v.Lines[event]
	if len(pool) == 0 {
		return ""
	}
	return pool[util.Rand(len(pool))]
}

var allVoices map[string]*VoiceSpec

func GetVoice(id string) *VoiceSpec {
	if allVoices == nil {
		return nil
	}
	return allVoices[id]
}

// SeedVoicesForTest swaps the registry; returns a restore func.
func SeedVoicesForTest(m map[string]*VoiceSpec) func() {
	old := allVoices
	allVoices = m
	return func() { allVoices = old }
}

// LoadDataFiles loads all voice YAMLs. Wire into server boot beside the
// enchantments loader.
func LoadDataFiles() {
	loaded, err := fileloader.LoadAllFlatFiles[string, *VoiceSpec](voicesPath())
	if err != nil {
		panic(err)
	}
	allVoices = loaded
}
```

IMPORTANT: copy the EXACT loader invocation from
`internal/enchantments` (`LoadAllFlatFiles` generic signature, the
config path helper for the datafiles dir, and the `mudlog.Info(...
loadedCount ...)` line) — the snippet above is directional; the real
fileloader API must be matched from that file. Add the boot call where
`enchantments.LoadDataFiles()` (or equivalent) is invoked, and create
`_datafiles/world/dogmud/itemvoices/` (empty dir; a `.gitkeep` so git
tracks it — real voices ship in Stage 2).

Also add ItemSpec cross-validation: a `voice_id` naming a missing voice
should fail LOUDLY at boot. Voices load after items, so put the check in
`itemvoices.LoadDataFiles` (iterate `items.GetAllItemSpecs()` if such an
accessor exists — grep for how other cross-registry validators do it,
e.g. the schedule validator pattern; if no accessor exists, add
`ValidateVoiceRefs(func(voiceId string) bool)` called from boot after
both loads).

- [ ] **Step 4: Run + commit**

Run: `go test ./internal/itemvoices/ -v` → PASS
Run: `go build ./...` → clean

```bash
git add internal/itemvoices/ _datafiles/world/dogmud/itemvoices/
git commit -m "feat(pinnacle): itemvoices package — sentient item line pools + loader"
```

---

### Task 11: Pinnacle per-round tick (hunger, chatter, ambience, aging freeze, mutation tick)

The single per-player hook that drives all always-on item behavior.
Gated by `PinnacleItemsEnabled`. Cost discipline: the tick early-outs
unless the character wears at least one pinnacle-flagged item — cheap
spec lookups only (the always-on ferry/warehouse ticks proved the
per-round budget, but items are per-player so keep it lean).

**Files:**
- Create: `internal/hooks/pinnacle_tick.go`
- Modify: `internal/hooks/NewRound_UserRoundTick.go` (call site in the per-player loop, after the buff trigger block at :145)
- Test: `internal/hooks/pinnacle_tick_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func TestHungerDrain(t *testing.T) {
	defer seedAllRegistries()()
	defer items.SeedItemSpecsForTest(map[int]*items.ItemSpec{
		999930: {ItemId: 999930, Name: "hungry blade", Type: items.Weapon, Hands: 2,
			HungerRounds: 50, HungerDrainPct: 0.01, VoiceId: ""},
	})()

	c := characters.New()
	c.HealthMax.Value = 400
	c.Health = 400
	c.Equipment.Weapon = items.New(999930)
	now := util.GetRoundCount()

	// anchor just set (fresh kill): no drain
	c.SetMiscData("pinnacle_hunger_anchor", now)
	tickHunger(c, nil, now)
	if c.Health != 400 {
		t.Fatalf("not hungry yet — expected 400, got %d", c.Health)
	}

	// 60 rounds since anchor: overdue by 10 → drain 1% of 400 = 4 (x1 escalation band)
	c.SetMiscData("pinnacle_hunger_anchor", now-60)
	tickHunger(c, nil, now)
	if c.Health >= 400 {
		t.Fatal("expected hunger drain")
	}
	if c.Health < 400-12 { // escalation cap 3x → at most 12 here
		t.Fatalf("drain too large: %d", c.Health)
	}
}

func TestAgingFreeze(t *testing.T) {
	defer seedAllRegistries()()
	defer items.SeedItemSpecsForTest(map[int]*items.ItemSpec{
		999940: {ItemId: 999940, Name: "vitalis bandolier", Type: items.Belt,
			IsBandolier: true, BandolierCapacity: 4, PreservesContents: true},
		999941: {ItemId: 999941, Name: "test potion", Type: items.Potion},
	})()

	c := characters.New()
	c.Equipment.Belt = items.New(999940)
	p := items.New(999941)
	p.CraftedRound = 1000
	c.PotionItems = append(c.PotionItems, p)

	tickPreserveContents(c)
	if c.PotionItems[0].CraftedRound != 1001 {
		t.Fatalf("expected CraftedRound advanced to 1001 (freezing elapsed), got %d", c.PotionItems[0].CraftedRound)
	}
}
```

(Verify `items.Belt`/`items.Potion` type constant names against
itemspec.go; verify `IsBandolier`/`BandolierCapacity` field names —
itemspec.go:290-291.)

- [ ] **Step 2: Run to verify it fails** → FAIL

- [ ] **Step 3: Implement `internal/hooks/pinnacle_tick.go`**

```go
package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/itemvoices"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// pinnacleUserTick runs once per player per round from UserRoundTick.
// It early-outs unless the player wears something pinnacle-flagged.
func pinnacleUserTick(user *users.UserRecord, room *rooms.Room) {
	if !bool(configs.GetConfig().GamePlay.PinnacleItemsEnabled) {
		return
	}
	c := user.Character
	now := util.GetRoundCount()

	tickPreserveContents(c)
	tickAmbientPotions(user, now)
	tickHunger(c, user, now)
	tickMutationItems(user, now)
	tickVoices(user, room, now)
}

// tickPreserveContents freezes aging for potions inside a
// preserves_contents bandolier by advancing CraftedRound 1:1 with the
// round counter (elapsed = now - CraftedRound stays constant).
func tickPreserveContents(c *characters.Character) {
	belt := c.Equipment.Belt
	spec := belt.GetSpec()
	if spec == nil || !spec.PreservesContents {
		return
	}
	for i := range c.PotionItems {
		c.PotionItems[i].CraftedRound++
	}
}

// tickHunger drains the wielder when a hunger-flagged weapon has gone too
// long without a kill. Escalation: 1x at first, growing with overdue time,
// capped at 3x. Never kills outright — clamps at 1 HP (the sword wants a
// living larder, and outright item-suicide is bad UX).
func tickHunger(c *characters.Character, user *users.UserRecord, now uint64) {
	spec := c.Equipment.Weapon.GetSpec()
	if spec == nil || spec.HungerRounds <= 0 || spec.HungerDrainPct <= 0 {
		return
	}
	anchorRaw := c.GetMiscData("pinnacle_hunger_anchor")
	anchor, ok := readMiscRound(anchorRaw)
	if !ok {
		// first time we see the weapon: start the clock now
		c.SetMiscData("pinnacle_hunger_anchor", now)
		return
	}
	if killRaw := c.GetMiscData("pinnacle_last_kill_round"); killRaw != nil {
		if kill, ok := readMiscRound(killRaw); ok && kill > anchor {
			anchor = kill
			c.SetMiscData("pinnacle_hunger_anchor", kill)
		}
	}
	elapsed := now - anchor
	if elapsed <= uint64(spec.HungerRounds) {
		return
	}
	overdue := elapsed - uint64(spec.HungerRounds)
	escalation := 1.0 + float64(overdue)/float64(spec.HungerRounds)
	if escalation > 3.0 {
		escalation = 3.0
	}
	drain := int(float64(c.HealthMax.Value) * spec.HungerDrainPct * escalation)
	if drain < 1 {
		drain = 1
	}
	if c.Health-drain < 1 {
		drain = c.Health - 1
	}
	if drain <= 0 {
		return
	}
	c.Health -= drain
	if user != nil {
		emitVoiceLine(user, nil, spec, "on_hunger_feeding",
			`<ansi fg="red">The blade feeds on you — a cold pull beneath your grip.</ansi>`)
	}
}

// tickMutationItems rolls worn mutation-tick items (the Seething Prism).
func tickMutationItems(user *users.UserRecord, now uint64) {
	c := user.Character
	for _, itm := range c.GetAllWornItems() {
		spec := itm.GetSpec()
		if spec == nil || spec.MutationTickInterval <= 0 {
			continue
		}
		if now%uint64(spec.MutationTickInterval) != 0 {
			continue
		}
		if util.Rand(100) >= spec.MutationTickChance {
			continue
		}
		if granted := c.GrantRandomMutationRare(spec.MutationRarityFloor); granted != "" {
			name := granted
			if ms := mutations.GetMutation(granted); ms != nil {
				name = ms.Name
			}
			user.SendText(messaging.CategoryWarning, fmt.Sprintf(
				`<ansi fg="magenta">Something stirs beneath your skin... <ansi fg="yellow">%s</ansi> takes root.</ansi>`, name))
		}
	}
}

// tickVoices lets sentient items speak (idle chatter + taunts), paced by
// SentientChatterCooldownRounds.
func tickVoices(user *users.UserRecord, room *rooms.Room, now uint64) {
	c := user.Character
	nextRaw := c.GetMiscData("pinnacle_voice_next_round")
	if next, ok := readMiscRound(nextRaw); ok && now < next {
		return
	}
	cool := uint64(configs.GetBalanceConfig().SentientChatterCooldownRounds)
	for _, itm := range c.GetAllWornItems() {
		spec := itm.GetSpec()
		if spec == nil || spec.VoiceId == "" {
			continue
		}
		v := itemvoices.GetVoice(spec.VoiceId)
		if v == nil {
			continue
		}
		event := "on_idle"
		if c.Aggro != nil {
			event = "on_taunt"
		} else if spec.HungerRounds > 0 {
			// hungry weapons warn before they feed
			if anchor, ok := readMiscRound(c.GetMiscData("pinnacle_hunger_anchor")); ok {
				if now-anchor > uint64(spec.HungerRounds)*3/4 {
					event = "on_hunger_warning"
				}
			}
		}
		line := v.Line(event)
		if line == "" {
			continue
		}
		// Low fire chance per eligible round so chatter feels occasional.
		if util.Rand(100) >= 15 {
			continue
		}
		user.SendText(messaging.CategorySystem, fmt.Sprintf(
			`<ansi fg="item">%s</ansi> says, "<ansi fg="yellow">%s</ansi>"`, spec.Name, line))
		if room != nil {
			room.SendTextVisual(messaging.CategorySystem, fmt.Sprintf(
				`<ansi fg="item">%s</ansi>'s %s mutters, "<ansi fg="yellow">%s</ansi>"`,
				c.Name, spec.Name, line), user.UserId)
		}
		c.SetMiscData("pinnacle_voice_next_round", now+cool)
		break // one line per round across all sentient items
	}
}

// emitVoiceLine sends an authored voice line for an event, falling back to
// the provided default flavor when the item has no voice or no lines.
func emitVoiceLine(user *users.UserRecord, room *rooms.Room, spec *items.ItemSpec, event, fallback string) {
	line := fallback
	if spec.VoiceId != "" {
		if v := itemvoices.GetVoice(spec.VoiceId); v != nil {
			if l := v.Line(event); l != "" {
				line = fmt.Sprintf(`<ansi fg="item">%s</ansi> says, "<ansi fg="yellow">%s</ansi>"`, spec.Name, l)
			}
		}
	}
	user.SendText(messaging.CategorySystem, line)
	_ = room
}
```

`tickAmbientPotions` — same file:

```go
// tickAmbientPotions keeps slotted potion buffs active while an
// ambient_potions bandolier is worn and attuned. Buff application uses
// AddBuffScaled at Peak potency (1.30). Buffs applied this way are
// recorded so removal can revoke them.
func tickAmbientPotions(user *users.UserRecord, now uint64) {
	c := user.Character
	belt := c.Equipment.Belt
	spec := belt.GetSpec()

	appliedRaw, _ := c.GetMiscData("pinnacle_bandolier_buffs").([]any)

	if spec == nil || !spec.AmbientPotions {
		revokeAmbient(c, appliedRaw)
		return
	}
	if attune, ok := readMiscRound(c.GetMiscData("pinnacle_bandolier_attune_round")); ok && now < attune {
		revokeAmbient(c, appliedRaw)
		return
	}

	current := map[int]bool{}
	for _, p := range c.PotionItems {
		pSpec := p.GetSpec()
		if pSpec == nil {
			continue
		}
		for _, buffId := range pSpec.BuffIds {
			current[buffId] = true
			if !c.Buffs.HasBuff(buffId) {
				c.AddBuffScaled(buffId, 1.30)
			}
		}
	}
	// revoke buffs whose potion left the bandolier
	for _, v := range appliedRaw {
		if id, ok := v.(int); ok && !current[id] {
			c.RemoveBuff(id)
		}
	}
	// persist current set
	ids := make([]any, 0, len(current))
	for id := range current {
		ids = append(ids, id)
	}
	c.SetMiscData("pinnacle_bandolier_buffs", ids)
}

func revokeAmbient(c *characters.Character, appliedRaw []any) {
	for _, v := range appliedRaw {
		if id, ok := v.(int); ok {
			c.RemoveBuff(id)
		}
	}
	if len(appliedRaw) > 0 {
		c.SetMiscData("pinnacle_bandolier_buffs", []any{})
	}
}
```

Verify against real APIs: `AddBuffScaled(buffId int, scale float64)`
signature (characters/buffs.go:112 per exploration — confirm), buff
permanence semantics (ambient buffs re-apply next tick if they expire —
that's the design: refresh, not permanent), `c.Buffs.HasBuff` (:172),
`c.RemoveBuff` — pick whichever exists on Character vs Buffs. The
attunement round is SET where bandolier contents change: find the
removal path (`internal/usercommands/` — whatever command pulls potions
out of PotionItems; grep `PotionItems` writers: `StoreItem`, removal in
`get`/`remove`/`drink`) and set:

```go
	c.SetMiscData("pinnacle_bandolier_attune_round",
		util.GetRoundCount()+uint64(configs.GetBalanceConfig().BandolierAttuneRounds))
```

in the removal/unequip paths ONLY (drinking a slotted potion counts as
removal; StoreItem addition also re-attunes — spec: "can't be drunk or
removed without breaking the ambient effect"). Do this in the same
task; list each write site you touched in the commit body.

- [ ] **Step 4: Call site**

In `NewRound_UserRoundTick.go`, in the per-player loop after the buff
trigger block (:145):

```go
		// Pinnacle item upkeep (procs are event-driven; this is the always-on layer).
		pinnacleUserTick(user, room)
```

- [ ] **Step 5: Run + commit**

Run: `go test ./internal/hooks/ -run "TestHungerDrain|TestAgingFreeze" -v` → PASS
Run: `go test ./internal/hooks/ ./internal/characters/ ./internal/usercommands/` → green

```bash
git add internal/hooks/ internal/usercommands/
git commit -m "feat(pinnacle): per-round tick — hunger, ambient potions, aging freeze, mutation tick, voices"
```

---

### Task 12: Boot test, full suite, docs

- [ ] **Step 1: Full test suite**

Run: `go test -timeout 300s ./...`
Expected: all green. Fix anything the new fields/stamping change broke.

- [ ] **Step 2: Boot test (pre-push SOP applies to merges too)**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go build -o gomud.exe . && ./gomud.exe
```

Watch for: `mobs.LoadDataFiles() loadedCount=...`, itemvoices loader
line (0 voices is fine — directory is empty until Stage 2), zero panics,
server reaches listening state. Kill the server afterward (do not leave
test servers running).

- [ ] **Step 3: Schema docs**

Create `docs/schemas/pinnacle-items.md` documenting: the `procs` yaml
shape (triggers, effects, params per effect), reserve pcts,
`preserves_contents`/`ambient_potions`, `mutation_tick_*`, `voice_id` +
the itemvoices file format + valid events, `hunger_rounds`/
`hunger_drain_pct`, `require_own_components`, the MiscData keys, and the
two config toggles. One page; Stage 2 content authors work from this.

- [ ] **Step 4: Commit + hand off**

```bash
git add docs/schemas/pinnacle-items.md
git commit -m "docs(pinnacle): Stage 1 primitive schemas for content authors"
```

Then follow `superpowers:finishing-a-development-branch` — merge to
master with `--no-ff` after review. Stage 2 (item YAMLs + voices) gets
its own plan.

---

## Explicitly OUT of Stage 1 (later stages)

- The nine item YAMLs, voice files, worn-buff YAMLs (Stage 2 — including
  the permanent-haste worn buff, which is pure buff YAML + WornBuffIds).
- Staff skill statmods — no engine work at all (`GetSkillLevel` already
  reads statmods; `spell_damage_multiplier` already applied).
- 18 reagent items + drop/forage placement (Stage 3).
- Veyra, rooms, dialogue, commission quests, recipes (Stage 4).
- The `aoe_stun` "party grudge" refinements, health/stamina steal_pool
  variants — YAGNI until an item needs them.
