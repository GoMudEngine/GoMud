# Conjure Elementals — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 5 elemental conjure spells (water, earth, air, fire, magma) with return damage mechanic for fire/magma. High conviction cost, no components.

**Architecture:** 5 new species with NaturalBash/ReturnDamage flags (already added to species struct), 5 mob templates, 5 conjure spells, return damage hook in combat resolution, help files.

**Tech Stack:** Go, JS scripting, YAML data files

---

## Reference Tables

### Elemental Stats

| Species | ID | Str | Dex | Per | Vit | Wil | Cha | basedmg | NaturalBash | ReturnDamage |
|---------|-----|-----|-----|-----|-----|-----|-----|---------|-------------|-------------|
| Water | 36 | 100 | 50 | 60 | 200 | 40 | 5 | 4 | no | 0 |
| Earth | 37 | 160 | 45 | 50 | 220 | 30 | 5 | 6 | yes | 0 |
| Air | 38 | 40 | 200 | 160 | 30 | 60 | 5 | 2 | no | 0 |
| Fire | 39 | 60 | 180 | 140 | 35 | 80 | 5 | 3 | no | 25 |
| Magma | 40 | 180 | 70 | 80 | 180 | 60 | 5 | 7 | yes | 25 |

### Spell Constants

| Spell ID | Name | Gate | Folds | Cost | Mob ID | Base Pool |
|----------|------|------|-------|------|--------|-----------|
| conjure-water | Conjure Water Elemental | 3 | 4 | 150 | 310 | 80 |
| conjure-earth | Conjure Earth Elemental | 10 | 6 | 200 | 311 | 90 |
| conjure-air | Conjure Air Elemental | 18 | 8 | 280 | 312 | 70 |
| conjure-fire | Conjure Fire Elemental | 25 | 10 | 350 | 313 | 85 |
| conjure-magma | Conjure Magma Elemental | 60 | 14 | 450 | 314 | 130 |

---

### Task 1: Return Damage Hook in Combat

**Files:**
- Modify: `internal/hooks/NewRound_DoCombat_helpers.go`

The return damage mechanic: when a melee attacker hits a defender
that has `return_damage > 0` (from species or stat mods), the
attacker takes a percentage of their own damage back.

- [ ] **Step 1: Read combat resolution**

Read `handlePlayerVsMob` and `handleMobVsPlayer` in
`NewRound_DoCombat_helpers.go`. Find where damage is applied to
the defender. After damage is applied, add the return damage check.

Also check `handleMobVsMob` for the same.

- [ ] **Step 2: Add return damage to all combat handlers**

After damage is applied to the defender, add:

```go
// Return damage (fire elemental, battlerager armor, etc.)
if roundResult.Hit && roundResult.DamageToTarget > 0 {
    returnPct := defender.Character.StatMod("return_damage")
    // Also check species return damage
    if sp := species.GetSpecies(defender.Character.SpeciesId); sp != nil {
        returnPct += sp.ReturnDamage
    }
    if returnPct > 0 {
        returnDmg := int(float64(roundResult.DamageToTarget) * float64(returnPct) / 100.0)
        if returnDmg > 0 {
            attacker.Character.Health -= returnDmg
            // Messaging
            room.SendText(fmt.Sprintf(
                `<ansi fg="red">%s takes damage from striking %s!</ansi>`,
                attackerName, defenderName))
        }
    }
}
```

Add this to: `handlePlayerVsMob`, `handleMobVsPlayer`, `handleMobVsMob`.

IMPORTANT: Return damage must NOT trigger more return damage.
The check is on the defender — since we're reducing the attacker's
HP directly (not via another combat round), there's no recursion
risk. But add a comment noting this.

- [ ] **Step 3: Verify build + commit**

```bash
git commit -m "feat: return damage mechanic — species + equipment damage reflection"
```

---

### Task 2: Species Definitions (5 files)

**Files:**
- Create 5 species YAML files in `_datafiles/world/dogmud/species/`

- [ ] **Step 1: Create all 5 species**

Read an existing species file for format (e.g., `2-canine.yaml`).

Create from the reference table above. Earth and magma get
`naturalbash: true`. Fire and magma get `return_damage: 25`.

Example for fire:
```yaml
speciesid: 39
name: Fire Elemental
description: A living column of flame and rage.
stats:
  strength:
    base: 60
  dexterity:
    base: 180
  perception:
    base: 140
  vitality:
    base: 35
  willpower:
    base: 80
  charisma:
    base: 5
damage:
  basedamage: 3
  variance: 2
return_damage: 25
```

- [ ] **Step 2: Commit**

```bash
git commit -m "feat: 5 elemental species definitions"
```

---

### Task 3: Mob Templates (5 files)

**Files:**
- Create 5 mob YAML files in `_datafiles/world/dogmud/mobs/summons/`

- [ ] **Step 1: Create all 5 mob templates**

All share: `zone: Summons`, `hostile: false`, `maxwander: 0`,
`groups: [summon, elemental]`, `archetype: fighting`, `level: 1`.

| File | Mob ID | Species | Statpool | Special combatcommands |
|------|--------|---------|----------|-----------------------|
| `310-water_elemental.yaml` | 310 | 36 | 80 | standard melee |
| `311-earth_elemental.yaml` | 311 | 37 | 90 | `bash` + crushing |
| `312-air_elemental.yaml` | 312 | 38 | 70 | fast striking |
| `313-fire_elemental.yaml` | 313 | 39 | 85 | burning/searing |
| `314-magma_elemental.yaml` | 314 | 40 | 130 | `bash` + molten |

Each needs 3-5 idlecommands and 2-4 combatcommands with elemental
flavor. Descriptions wrap at 80 chars.

activitylevel: 40-50 for all.

- [ ] **Step 2: Commit**

```bash
git commit -m "feat: 5 elemental mob templates"
```

---

### Task 4: Conjure Spells (5 YAML + 5 JS)

**Files:**
- Create 10 files in `_datafiles/world/dogmud/spells/`

- [ ] **Step 1: Create spell YAMLs**

All conjure spells: `type: neutral`, `schools: [manifestation]`,
no `quest_required`. Use costs/folds from the reference table.

- [ ] **Step 2: Create spell JS scripts**

Follow the summon spell JS pattern (`summon-steppe-spirit.js`).
Conjure spells are simpler — no component check needed.

Each `onMagic`:
1. Check companion cap
2. Calculate scaled statpool using caster charisma + manifestation
3. `room.SpawnMobScaled(mobId, scaledPool)`
4. `mob.CharmSet(caster.UserId(), 99999)`
5. `caster.AddCompanion(mob.InstanceId(), "conjured", name)`
6. Flavor text (elemental-themed summoning visuals)

Each `onCast`:
1. Check companion cap
2. Return true (no component to check)

Each `onWait`:
- Elemental-themed channeling flavor text

- [ ] **Step 3: Commit**

```bash
git commit -m "feat: 5 conjure elemental spells with high conviction costs"
```

---

### Task 5: Help Files

**Files:**
- Create 5 help templates in `_datafiles/world/dogmud/templates/help/`

- [ ] **Step 1: Create help files**

One per conjure spell. Follow existing raise spell help format.
Cover: what it conjures, school, conviction cost described as
"immense" / "devastating" etc. (no raw numbers), elemental's
combat style, see also links.

- [ ] **Step 2: Update keywords.yaml**

Add conjure aliases if not already covered by the manifestation
help alias we added earlier (which includes "conjure").

- [ ] **Step 3: Commit**

```bash
git commit -m "feat: help files for 5 conjure elemental spells"
```

---

### Task 6: Tests + Verification

- [ ] **Step 1: Test return damage**

Add to `internal/characters/companions_test.go` or a new test file:
- Test that species with ReturnDamage > 0 reflects damage
- Test that return damage doesn't trigger on non-melee

- [ ] **Step 2: Full build + tests**

Run: `go build ./...`
Run: `go test ./... -count=1 -timeout 300s`

- [ ] **Step 3: Manual smoke test**

- Cast `conjure water` → water elemental appears, high HP
- Cast `conjure earth` → earth elemental bashes (crushing slam)
- Cast `conjure fire` → fire elemental, attackers take return damage
- Cast `conjure magma` → magma does both bash + return damage
- Check conviction costs are punishing
- `companion` shows elemental with correct stats

- [ ] **Step 4: Commit any fixups**
