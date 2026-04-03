# Necromancy System — Phase 3 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Six undead companion types raised from corpses, corpse assessment command, vampire bite special attack, flesh golem corpse absorption, companion cast-interrupt on owner movement.

**Architecture:** 6 new species YAML files, 6 new mob templates, 6 new raise spells (YAML + JS), assess command, bite combat command, cast-interrupt in go.go follow logic.

**Tech Stack:** Go, JS scripting, YAML data files

---

## File Structure

| File | Responsibility |
|------|---------------|
| `_datafiles/world/dogmud/species/30-skeleton.yaml` | Skeleton species |
| `_datafiles/world/dogmud/species/31-zombie.yaml` | Zombie species |
| `_datafiles/world/dogmud/species/32-wraith.yaml` | Wraith species |
| `_datafiles/world/dogmud/species/33-spectre.yaml` | Spectre species |
| `_datafiles/world/dogmud/species/34-vampire.yaml` | Vampire species |
| `_datafiles/world/dogmud/species/35-flesh_golem.yaml` | Flesh Golem species |
| `_datafiles/world/dogmud/mobs/summons/300-skeleton.yaml` | Skeleton mob template |
| `_datafiles/world/dogmud/mobs/summons/301-zombie.yaml` | Zombie mob template |
| `_datafiles/world/dogmud/mobs/summons/302-wraith.yaml` | Wraith mob template |
| `_datafiles/world/dogmud/mobs/summons/303-spectre.yaml` | Spectre mob template |
| `_datafiles/world/dogmud/mobs/summons/304-vampire.yaml` | Vampire mob template |
| `_datafiles/world/dogmud/mobs/summons/305-flesh_golem.yaml` | Flesh Golem mob template |
| `_datafiles/world/dogmud/spells/raise-skeleton.yaml` | Raise Skeleton spell |
| `_datafiles/world/dogmud/spells/raise-skeleton.js` | Raise Skeleton script |
| `_datafiles/world/dogmud/spells/raise-zombie.yaml` | Raise Zombie spell |
| `_datafiles/world/dogmud/spells/raise-zombie.js` | Raise Zombie script |
| `_datafiles/world/dogmud/spells/raise-wraith.yaml` | Raise Wraith spell |
| `_datafiles/world/dogmud/spells/raise-wraith.js` | Raise Wraith script |
| `_datafiles/world/dogmud/spells/raise-spectre.yaml` | Raise Spectre spell |
| `_datafiles/world/dogmud/spells/raise-spectre.js` | Raise Spectre script |
| `_datafiles/world/dogmud/spells/raise-vampire.yaml` | Raise Vampire spell |
| `_datafiles/world/dogmud/spells/raise-vampire.js` | Raise Vampire script |
| `_datafiles/world/dogmud/spells/raise-golem.yaml` | Raise Flesh Golem spell |
| `_datafiles/world/dogmud/spells/raise-golem.js` | Raise Flesh Golem script |
| `internal/usercommands/assess.go` | Assess command |
| `internal/usercommands/usercommands.go` | Register assess |
| `internal/mobcommands/bite.go` | Vampire bite attack |
| `internal/mobcommands/mobcommands.go` | Register bite |
| `internal/usercommands/go.go` | Cast-interrupt on follow |
| `internal/scripting/room_func.go` | Add GetCorpses / RemoveCorpse JS API |
| `_datafiles/world/dogmud/templates/help/assess.template` | Help file |

---

### Task 1: Corpse JS Scripting API

The raise spells need to find and consume corpses in the room.
Currently the JS scripting layer has no corpse access.

**Files:**
- Modify: `internal/scripting/room_func.go`

- [ ] **Step 1: Read room_func.go and rooms/corpse.go**

Understand how corpses are stored (`room.Corpses []Corpse`) and
the Corpse struct fields. Read ScriptRoom methods for the pattern.

- [ ] **Step 2: Add corpse JS methods**

Add to `ScriptRoom`:

```go
// GetCorpses returns all corpses in the room as script-accessible objects.
func (r ScriptRoom) GetCorpses() []ScriptCorpse

// RemoveCorpse removes a corpse by index. Used when a raise spell consumes it.
func (r ScriptRoom) RemoveCorpse(index int) bool
```

Add a `ScriptCorpse` wrapper struct that exposes:
- `Name() string` — corpse character name
- `MobId() int` — original mob ID (0 for player corpses)
- `IsPlayerCorpse() bool`
- `GetStatTrainingTotal() int` — sum of all 6 stat training values (the "corpse statpool")
- `Index() int` — position in the corpses slice (for RemoveCorpse)

- [ ] **Step 3: Verify build + commit**

```bash
git commit -m "feat: corpse JS scripting API for necromancy spells"
```

---

### Task 2: Species Definitions (6 files)

**Files:**
- Create 6 species YAML files

- [ ] **Step 1: Create all 6 species**

Follow the format of existing species files (e.g., `2-canine.yaml`).
Read one to understand the structure.

**`_datafiles/world/dogmud/species/30-skeleton.yaml`:**
```yaml
speciesid: 30
name: Skeleton
description: An animated skeletal frame, quick and relentless.
stats:
  strength:
    base: 100
  dexterity:
    base: 150
  perception:
    base: 80
  vitality:
    base: 60
  willpower:
    base: 15
  charisma:
    base: 5
damage:
  basedamage: 4
  variance: 2
```

Create the remaining 5 species with these exact base stats:

| File | Str | Dex | Per | Vit | Wil | Cha | basedamage |
|------|-----|-----|-----|-----|-----|-----|------------|
| `31-zombie.yaml` | 130 | 50 | 40 | 200 | 15 | 5 | 5 |
| `32-wraith.yaml` | 40 | 160 | 150 | 35 | 170 | 30 | 2 |
| `33-spectre.yaml` | 30 | 150 | 130 | 30 | 150 | 170 | 2 |
| `34-vampire.yaml` | 120 | 140 | 120 | 110 | 110 | 160 | 5 |
| `35-flesh_golem.yaml` | 220 | 65 | 40 | 240 | 15 | 5 | 8 |

- [ ] **Step 2: Commit**

```bash
git commit -m "feat: 6 undead species definitions for necromancy"
```

---

### Task 3: Mob Templates (6 files)

**Files:**
- Create 6 mob YAML files in `_datafiles/world/dogmud/mobs/summons/`

- [ ] **Step 1: Create all 6 mob templates**

Follow the pattern of `243-steppe_spirit_wolf.yaml`. Each needs:
- `mobid`, `zone: Summons`, `archetype`, `statpool` (base pool from spec)
- `hostile: false`, `maxwander: 0`
- `groups: [summon, undead]`
- `idlecommands` with flavor emotes
- `combatcommands` with flavor emotes
- Character: name, description (80-char wrapped), speciesid, level: 1

**Mob ID assignments:** 300-305

For caster types (wraith, spectre, vampire), add starting spells
to the character's spellbook:

**Wraith (302):**
```yaml
  spells:
    chill-touch: 1
    minor-shield: 1
```

**Spectre (303):**
```yaml
  spells:
    conviction-spike: 1
    conviction-ward: 1
```

**Vampire (304):**
```yaml
  spells:
    ward: 1
    conviction-surge: 1
```

Check that these spell IDs exist in `_datafiles/world/dogmud/spells/`.
Use actual existing spell IDs. Read the spells directory to find
appropriate harm/buff spells for each type.

For vampire, add `bite` to combatcommands.
For flesh golem, add `consume` to combatcommands.

- [ ] **Step 2: Commit**

```bash
git commit -m "feat: 6 undead mob templates for necromancy"
```

---

### Task 4: Raise Spells (6 YAML + 6 JS)

**Files:**
- Create 12 files (6 pairs of .yaml + .js)

- [ ] **Step 1: Create spell YAMLs**

All raise spells follow the same pattern. Create all 6 with
these exact values:

| Spell ID | Name | Folds | Cost | Mob ID | Base Pool | Min Corpse |
|----------|------|-------|------|--------|-----------|------------|
| raise-skeleton | Raise Skeleton | 4 | 20 | 300 | 60 | 30 |
| raise-zombie | Raise Zombie | 6 | 30 | 301 | 80 | 60 |
| raise-wraith | Raise Wraith | 8 | 45 | 302 | 70 | 120 |
| raise-spectre | Raise Spectre | 10 | 60 | 303 | 90 | 200 |
| raise-vampire | Raise Vampire | 12 | 80 | 304 | 100 | 300 |
| raise-golem | Raise Flesh Golem | 16 | 100 | 305 | 120 | 500 |

All are `type: neutral`, `schools: [manifestation]`, no `quest_required`.

Example YAML:

```yaml
spellid: raise-skeleton
name: Raise Skeleton
description: |
  Binds lingering death energy into a skeletal frame, raising
  an animated warrior from the remains of the fallen.
cost: 20
base_folds: 4
type: neutral
schools:
  - manifestation
```

- [ ] **Step 2: Create spell JS scripts**

All raise scripts follow the same pattern with type-specific values.
Read the existing `summon-steppe-spirit.js` for the API pattern.

Each `onMagic` handler:
1. Check companion cap
2. Find corpses in room via `GetRoom(caster.GetRoomId()).GetCorpses()`
3. If `rest` arg provided, match corpse by name; otherwise use first corpse
4. Check corpse statpool vs minimum threshold for this type
5. If too weak: random failure flavor text, consume partial conviction, return
6. Calculate raised pool: `(CalcCompanionStatPool(typeBase, cha, skill) * 0.5) + (corpsePool * 0.5)`
7. `room.SpawnMobScaled(mobId, raisedPool)`
8. `mob.CharmSet(caster.UserId(), 99999)`
9. `caster.AddCompanion(mob.InstanceId(), "raised", typeName)`
10. `room.RemoveCorpse(corpseIndex)`
11. Flavor text: "Dark energy flows from your hands into the remains of [name]..."

The `onCast` handler checks for companion cap and corpse presence.
The `onWait` handler shows dark ritual flavor text.

Create a shared failure text array used by all 6 scripts for the
"corpse too weak" case. Each script also needs skill-too-low failure
text (checked in onCast).

- [ ] **Step 3: Commit**

```bash
git commit -m "feat: 6 raise spells with corpse scaling and failure flavor"
```

---

### Task 5: Assess Command

**Files:**
- Create: `internal/usercommands/assess.go`
- Modify: `internal/usercommands/usercommands.go`

- [ ] **Step 1: Create assess command**

```go
func Assess(rest string, user *users.UserRecord,
    room *rooms.Room, flags events.EventFlag) (bool, error)
```

Flow:
1. If no arg: "Assess what?"
2. Find corpse in room by name match (search `room.Corpses`)
3. If not found: "You don't see that corpse here."
4. Calculate corpse statpool (sum training values)
5. Describe the corpse's power tier descriptively
6. List which undead types it could support
7. Describe freshness (based on rounds since creation)
8. Fire `OnSkillUse("manifestation")` for skill progression

Output example:
```
You study the remains of Guard Captain Velk...
The residual essence feels substantial — a strong life force
lingers here. These remains could sustain a wraith or perhaps
even a spectre. The flesh is still relatively fresh.
```

Register as: `"assess": {Assess, true, true, false}`

- [ ] **Step 2: Create help file**

`_datafiles/world/dogmud/templates/help/assess.template`

- [ ] **Step 3: Commit**

```bash
git commit -m "feat: assess command for corpse evaluation"
```

---

### Task 6: Vampire Bite Attack

**Files:**
- Create: `internal/mobcommands/bite.go`
- Modify: `internal/mobcommands/mobcommands.go`

- [ ] **Step 1: Create bite command**

`bite` is a special melee attack for vampire companions that
drains life.

Pattern: follow `internal/actions/combat_bash.go` for the shared
action pattern. Bite uses:
- Attack stat: Strength + unarmed-combat
- Defense: target's combat skill + Dexterity
- Damage: similar to kick (DamagePercent ~0.60)
- On hit: heal vampire for `damage × 0.50` (50% life drain)
- Uses the shared special-move cooldown
- Skill progression: unarmed-combat

The mob command just calls the shared action and formats messages.

Register as: `"bite": {Bite, false}`

- [ ] **Step 2: Commit**

```bash
git commit -m "feat: vampire bite attack with life drain"
```

---

### Task 7: Cast-Interrupt on Owner Movement

**Files:**
- Modify: `internal/usercommands/go.go`

- [ ] **Step 1: Find the charmed-mob follow logic**

In `go.go`, find where charmed mobs follow the player on room
change. This is the code that issues a `go <direction>` command
to charmed mobs.

- [ ] **Step 2: Add cast interruption before follow**

Before issuing the follow command, check if the companion has
`CastingState != nil`. If so, clear it:

```go
if mob.Character.CastingState != nil {
    mob.Character.CastingState = nil
    // Optional: room message "companion breaks off its spell to follow you"
}
```

This ensures caster companions don't get left behind mid-cast.

- [ ] **Step 3: Commit**

```bash
git commit -m "feat: companions interrupt casting to follow owner"
```

---

### Task 8: Flesh Golem Corpse Absorption Enhancement

**Files:**
- Modify: `internal/mobcommands/consume.go`

- [ ] **Step 1: Read existing consume command**

Read `internal/mobcommands/consume.go` to understand how mobs
consume corpses currently.

- [ ] **Step 2: Enhance for flesh golem**

When the consuming mob's species is flesh golem (species 35):
- Instead of the standard regen buff, apply a temporary stat buff
- Buff strength and vitality based on the consumed corpse's stats
- Flavor: "[golem] rips a piece from the fallen [name] and grafts
  it onto itself! Its form grows more massive."
- Duration: ~20 rounds
- Check if the mob has an existing consume buff — if so, stack up
  to a cap (3 max stacks?)

If the consuming mob is NOT a flesh golem, keep the existing
behavior (standard regen buff).

- [ ] **Step 3: Commit**

```bash
git commit -m "feat: flesh golem enhanced corpse absorption"
```

---

### Task 9: Help File Updates

**Files:**
- Create: `_datafiles/world/dogmud/templates/help/assess.template`
- Modify: `_datafiles/world/dogmud/templates/help/manifestation.template`
- Modify: `_datafiles/world/dogmud/keywords.yaml` (add assess alias)

- [ ] **Step 1: Create assess help + update manifestation help**

Add necromancy section to manifestation help. Create assess help
covering usage and what the output means.

Add to keywords.yaml help-aliases:
```yaml
  assess:           [examine corpse, study corpse, evaluate corpse]
```

- [ ] **Step 2: Commit**

```bash
git commit -m "feat: help files for assess + necromancy section in manifestation"
```

---

### Task 10: Tests

- [ ] **Step 1: Test corpse statpool calculation**

Test that summing corpse training values produces the expected
statpool for the scaling formula.

- [ ] **Step 2: Test raise scaling formula**

Test the 50/50 split: `(companionScale * 0.5) + (corpsePool * 0.5)`
at various caster stat / corpse power combinations.

- [ ] **Step 3: Commit**

```bash
git commit -m "test: necromancy scaling + corpse statpool tests"
```

---

### Task 11: Final Verification

- [ ] **Step 1: Full build + tests**

- [ ] **Step 2: Manual smoke test**

- Kill a mob → corpse appears
- `assess corpse` → descriptive evaluation
- `cast raise-skeleton` → skeleton rises from corpse
- `companion` → skeleton in list
- Skeleton fights alongside player
- Kill a stronger mob → `assess` shows wraith-capable
- `cast raise-wraith` → wraith rises, casts spells in combat
- Attempt raise on too-weak corpse → failure flavor text
- Vampire bites enemies and heals itself
- Flesh golem consumes corpses for stat buff
- Move rooms while companion is casting → spell interrupted, follows

- [ ] **Step 3: Commit any fixups**
