# Charm Spell — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Single charm spell that converts hostile mobs into companions via opposed roll with diminishing re-rolls. Charm-immune flag for merchants/quest NPCs.

**Architecture:** Spell YAML+JS for the initial charm, Go code for duration tick and re-roll in mob round tick, charm_immune field on Mob struct, CharmDuration/CharmRerolls fields on CompanionInfo.

**Tech Stack:** Go, JS scripting, YAML data files

---

## Reference

| Field | Value |
|-------|-------|
| Spell ID | charm |
| Base Folds | 36 |
| Cost | 120 |
| Gate | 20 manifestation |
| Type | harmsingle |
| School | manifestation |

**Opposed Roll:**
- Attacker: `Charisma + (manifestation × SkillMultiplier × 25)`
- Defender: `Willpower + (statpool × 0.10)`
- Aggro penalty: 75% if target aggro'd on caster, 85% if aggro'd on someone else

**Duration:** `50 + Charisma/2 + manifestation × 3`

**Re-roll decay:** `effectiveness = 1.0 - (rerollCount × 0.01) × rerollCount`

---

### Task 1: CharmImmune Flag + CompanionInfo Duration Fields

**Files:**
- Modify: `internal/mobs/mobs.go` — add `CharmImmune bool` field
- Modify: `internal/characters/companions.go` — add `CharmDuration int`, `CharmRerolls int`
- Modify: merchant/quest NPC mob YAMLs — add `charm_immune: true`

- [ ] **Step 1: Add CharmImmune to Mob struct**

Read `internal/mobs/mobs.go`, find the Mob struct. Add:
```go
CharmImmune bool `yaml:"charm_immune,omitempty"`
```

- [ ] **Step 2: Add duration fields to CompanionInfo**

Read `internal/characters/companions.go`, find CompanionInfo. Add:
```go
CharmDuration int `yaml:"charm_duration,omitempty"` // rounds remaining
CharmRerolls  int `yaml:"charm_rerolls,omitempty"`  // re-roll count
```

- [ ] **Step 3: Flag merchant and quest NPCs**

Add `charm_immune: true` to all mobs that should be immune. Search
for mobs with `groups: [merchant]` or key quest NPCs. At minimum:
- All mobs in `_datafiles/world/dogmud/mobs/` that have a `shop:` field
- Key quest NPCs: Sylara (241), Rhett (242), Hermit Kael (240),
  temple priest Olen (95), guard captain Velk (94)
- Trainers, dialogue-only NPCs

Use grep to find all mobs with `shop:` and add the flag to each.

- [ ] **Step 4: Build + commit**

```bash
git commit -m "feat: CharmImmune mob flag + CompanionInfo duration fields"
```

---

### Task 2: Charm Duration Tick + Re-Roll

**Files:**
- Modify: `internal/hooks/NewRound_MobRoundTick.go`

- [ ] **Step 1: Read MobRoundTick**

Find where charmed mob processing happens. The charm expiry tick
is already there (decrements `Charmed.RoundsRemaining`). We need
to add companion-specific charm duration ticking.

- [ ] **Step 2: Add charm duration tick for companions**

In the mob round tick, for each mob that is charmed:
1. Find the CompanionInfo on the owner's character
2. If `comp.CharmDuration > 0`: decrement
3. If `comp.CharmDuration == 0` and `comp.SourceType == CompanionCharmed`:
   - Fire re-roll
   - Attacker score: `owner.Charisma + manifestSkill × SkillMult × 25`
   - Defender score: `mob.Willpower + totalTraining × 0.10`
   - Effectiveness: `1.0 - (float64(comp.CharmRerolls) * 0.01) * float64(comp.CharmRerolls)`
   - Multiply attacker score by effectiveness
   - Use `dice.OpposedRollStat(attackScore, defenseScore)`
   - On WIN: reset duration, increment rerolls, send warning messages
   - On LOSE: break charm (betrayal), remove CompanionInfo

Warning messages based on reroll count:
- rerolls >= 3: "You sense [name]'s will straining against your bond..."
- rerolls >= 5: "[name]'s eyes flash with defiance. Your control is slipping..."

Re-roll success: "Your hold on [name] wavers... but you reassert your will."
Re-roll failure: "[name] breaks free of your control!" + betrayal

For betrayal: same as dismiss — `mob.Character.RemoveCharm()`,
set aggro on owner, remove CompanionInfo, send room message.

- [ ] **Step 3: Build + commit**

```bash
git commit -m "feat: charm duration tick with diminishing re-roll mechanic"
```

---

### Task 3: Charm Spell (YAML + JS)

**Files:**
- Create: `_datafiles/world/dogmud/spells/charm.yaml`
- Create: `_datafiles/world/dogmud/spells/charm.js`

- [ ] **Step 1: Create spell YAML**

```yaml
spellid: charm
name: Charm
description: |
  You reach into the mind of a hostile creature and bend its
  will to yours, turning it into a loyal companion. The charm
  requires intense focus and is much harder against creatures
  already in combat. Stronger creatures resist more fiercely.
cost: 120
base_folds: 36
type: harmsingle
schools:
  - manifestation
```

- [ ] **Step 2: Create spell JS**

Read the existing summon/raise spell JS for API patterns.

**onCast(sourceActor, targetActor, spellAggro):**
1. Check companion cap
2. Check target is a mob (not a player)
3. Check target is not charm_immune — need to check this. Read
   how the JS can access the target mob's data. `targetActor`
   should be available. Check if there's a `targetActor.MobInstanceId()`
   or similar to get the mob, then check the mob template's
   CharmImmune field. If no JS API exists for CharmImmune, add
   a `IsCharmImmune()` method to ScriptActor.
4. Return true to proceed, false to cancel

**onWait(sourceActor, targetActor, spellAggro):**
Cycling flavor text:
- "You lock eyes with [name], your will pressing against theirs..."
- "The air crackles with psychic tension..."
- "You feel the resistance wavering..."

**onMagic(sourceActor, targetActor, spellAggro):**
1. Re-check companion cap
2. Re-check charm immunity
3. Calculate opposed roll:
   - `attackScore = sourceActor.GetStat("charisma") + sourceActor.GetSkillLevel("manifestation") * 25`
   - Need target's willpower + statpool. Get via `targetActor.GetStat("willpower")` and sum target's stat training
   - `defenseScore = targetWillpower + Math.round(targetStatpool * 0.10)`
4. Aggro penalty: check if target has aggro
   - `targetActor.IsAggro()` or check aggro state
   - If aggro on caster: `attackScore *= 0.75`
   - If aggro on other: `attackScore *= 0.85`
5. Opposed roll: need a JS-accessible dice roll. Check if
   `dice.OpposedRollStat` is exposed to JS. If not, do a simple
   comparison: `if (attackScore > defenseScore)` with some
   randomness via `Math.random()`.
6. On success:
   - `targetActor.CharmSet(sourceActor.UserId(), 99999)` (permanent charm, duration managed in Go)
   - `sourceActor.AddCompanion(targetActor.InstanceId(), "charmed", targetActor.GetCharacterName(false))`
   - Set initial CharmDuration on the CompanionInfo (need JS API or set in Go)
   - Strip companion chains
   - End target's aggro
   - Success messaging
7. On failure:
   - Failure messaging
   - Target stays hostile

NOTE: The charm duration and re-roll count need to be set on the
CompanionInfo. The JS `AddCompanion` call creates the entry but
doesn't set duration. Either:
- Add a `SetCompanionCharmDuration(instanceId, duration, rerolls)` JS method
- Or set it in Go after the companion is created
- Simplest: add optional params to `AddCompanion` in the Go scripting API

- [ ] **Step 3: Add any needed JS API methods**

If needed, add to `internal/scripting/actor_func.go`:
- `IsCharmImmune()` on ScriptActor — checks mob template's CharmImmune
- `SetCompanionCharmDuration(instanceId, duration)` — sets duration on CompanionInfo
- Or extend `AddCompanion` to accept duration

Also need: a way to do opposed rolls from JS. Check if any dice
functions are exposed. If not, add `RollOpposed(attackScore, defenseScore) bool`.

- [ ] **Step 4: Build + commit**

```bash
git commit -m "feat: charm spell with opposed roll and aggro penalty"
```

---

### Task 4: Help File + Keywords

**Files:**
- Create: `_datafiles/world/dogmud/templates/help/charm.template`
- Modify: `_datafiles/world/dogmud/keywords.yaml`

- [ ] **Step 1: Create help file**

Cover: what charm does, manifestation school, high conviction cost,
harder against combat targets, charm duration and re-rolls
(described as "your hold weakens over time"), no hard numbers.

- [ ] **Step 2: Add keyword alias**

Charm is already covered by the manifestation help alias. But add
`charm` specifically to the help-aliases if not present:
```yaml
  charm:            [mesmerize, dominate, control, bewitch]
```

- [ ] **Step 3: Commit**

```bash
git commit -m "feat: charm help file + keyword aliases"
```

---

### Task 5: Tests + Verification

- [ ] **Step 1: Full build + tests**

Run: `go build ./...`
Run: `go test ./... -count=1 -timeout 300s`

- [ ] **Step 2: Grant spell to test character**

Add `charm: 1` to quester9's spellbook (the user save file is
gitignored, just edit locally).

- [ ] **Step 3: Manual smoke test**

- Cast charm on a weak mob → should succeed, mob becomes companion
- Cast charm on a merchant → "cannot be charmed" message
- Cast charm at companion cap → "cannot maintain another bond"
- Wait for charm duration to expire → re-roll message
- Eventually charm breaks → betrayal, mob turns hostile
- Cast charm while target is fighting you → harder roll
- `companion` shows charmed mob with correct stats
- `dismiss` the charmed mob → betrayal as normal

- [ ] **Step 4: Commit any fixups**
