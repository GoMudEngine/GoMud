# Manifestation Skill + Unified Companion System — Phase 1 Design

**Date:** 2026-04-03
**Goal:** New manifestation skill (charisma-based), unified companion
system merging pets and charmed mobs, party integration with
bidirectional autoassist, companion persistence, and spell system
integration for manifestation-school spells.

## Manifestation Skill

New skill added to the skill system:

| Field | Value |
|-------|-------|
| Name | manifestation |
| Primary stat | Charisma |
| Progression multiplier | 0.30 |
| Soft cap | 50 (standard) |

### Companion Cap

Maximum companions = `floor(manifestation / 18.75)` capped at 4:

| Manifestation Rank | Max Companions |
|--------------------|---------------|
| 0 (knows a spell) | 1 |
| 1-18 | 1 |
| 19-37 | 2 |
| 38-56 | 3 |
| 57-75 | 4 |

If the player knows at least one manifestation-school spell,
minimum cap is 1 regardless of skill rank. If they know zero
manifestation spells, cap is 0 (no companions).

### Spell Routing

Spells tagged with `school: manifestation` use Charisma +
manifestation skill for:
- Fold accumulation rate (`CalcFoldsPerRound`)
- Spell discovery thresholds
- Conviction cost (still conviction — manifestation is still magic)
- Success/opposed rolls where applicable

The `cast` command is unchanged from the player's perspective.
Under the hood, `InitiateCast` reads the spell's school to select
the correct stat+skill pair.

Implementation: add a `GetSpellStat` and `GetSpellSkill` helper
that reads the spell school and returns the appropriate stat name
and skill tag. `CalcFoldsPerRound` takes these as parameters
instead of hardcoding willpower+spellcasting.

### Quest-Gated Spells

Spells can have a `quest_required: "<questtoken>"` field. When set:
- The spell NEVER appears in skill-based discovery
- The spell can only be learned via the quest engine's `teach_spell`
  action or a dialogue `teachesSpell` field
- The spell still appears in the player's spell list once learned
- Example: Sylara's wolf summon spell requires completing her quest

## Unified Companion System

### CompanionInfo Struct

```
CompanionInfo:
  MobId          int       // template mob ID
  InstanceId     int       // runtime mob instance (0 if offline)
  SourceType     string    // "summoned" | "charmed" | "raised" | "pet"
  Name           string    // display name (can be renamed)
  AutoAssist     bool      // default true
  // Persisted progression state
  StatTraining   map[string]int  // stat training values
  Skills         map[string]int  // skill ranks
  SkillUseCount  map[string]int  // skill use counters
  Mutations      map[string]int  // mutations
  SpellBook      map[string]int  // known spells
  MutationProgress float64
```

Stored on `Character.Companions []CompanionInfo` — persisted to
disk as part of the character YAML.

### Source Types

| Type | Origin | Stat Scaling | On Dismiss |
|------|--------|-------------|------------|
| summoned | Cast a summon spell | Charisma + manifestation | Turns hostile, despawns after combat |
| conjured | Cast a conjure spell | Charisma + manifestation | Turns hostile, despawns after combat |
| charmed | Cast charm on existing mob | Keeps original stats | Turns hostile permanently (reverts to original mob) |
| raised | Necromancy on a corpse | Charisma + manifestation | Turns hostile, despawns after combat |
| pet | Purchased/quest reward | Keeps defined stats | Turns hostile, despawns after combat |

### Companion Lifecycle

**Creation (summon/conjure/raise):**
1. Spell resolves → `SpawnMob(mobId)` with scaled statpool
2. Mob charmed to player with permanent duration
3. `CompanionInfo` created and added to `Character.Companions`
4. Companion auto-joins party

**Creation (charm):**
1. Charm spell succeeds → existing mob charmed
2. `CompanionInfo` created from mob's current state
3. Companion auto-joins party
4. Mob retains its current stats (no scaling)

**Login/restart:**
1. For each `CompanionInfo` in `Character.Companions`:
2. Spawn mob from `MobId` template
3. Apply saved stat training, skills, mutations, spells
4. Set to full HP/SP/CP, clean combat state
5. Place in player's current room
6. Charm to player with permanent duration
7. Set `InstanceId` on the `CompanionInfo`

**Logout/restart:**
1. For each active companion:
2. Save current stat training, skills, mutations, spells to `CompanionInfo`
3. Despawn mob instance
4. Set `InstanceId = 0`

**Death:**
1. Companion mob dies in combat
2. `CompanionInfo` removed from `Character.Companions`
3. Player notified: "Your [name] has fallen."
4. Summoned/conjured/raised: gone permanently (re-summon for a new one)
5. Charmed: mob corpse remains as normal
6. Pet: gone permanently (must acquire a new pet)

**Dismiss:**
1. Player types `dismiss <companion>`
2. Companion turns hostile (aggro on player)
3. Charm removed
4. `CompanionInfo` removed from `Character.Companions`
5. Summoned/conjured/raised: mob stays in world until killed or despawn timer
6. Charmed: mob reverts to original behavior permanently
7. ALL companion types turn hostile on dismiss — full betrayal

### Stat Scaling (Summon/Conjure/Raise)

For companions that scale with caster stats:

```
scaledStatPool = baseStatPool × (1.0 + (charisma / 200) + (manifestation × 0.02))
```

At charisma 100, manifestation 0: `base × 1.5`
At charisma 100, manifestation 25: `base × 2.0`
At charisma 150, manifestation 50: `base × 2.75`

The base statpool comes from the mob's YAML definition. The scaled
value is used when spawning. This means the same wolf spell produces
a stronger wolf as the caster grows.

Config knobs: `ManifestStatScaleChaFactor` (default 200),
`ManifestStatScaleSkillFactor` (default 0.02).

## Party Integration

### Companion Party Membership

Companions are treated as party members for combat purposes:

- Companions appear in the party vitals display
- Companions count for party-based mechanics (aggro splitting, etc.)
- Companions do NOT count against player party size limits

### Bidirectional Auto-Assist

**Owner → Companion:** When a companion with `AutoAssist=true` is
attacked, the owner (and other party members with autoassist on)
automatically attack the aggressor. This uses the existing party
autoassist system.

**Companion → Owner:** When the owner enters combat, companions
with `AutoAssist=true` attack the same target. This already works
via the existing charmed mob assist code in `attack.go` and
`NewRound_DoCombat_helpers.go`.

**Party Member → Companion:** Other party members with autoassist
also defend companions, same as they would any party member.

### Party Merge Behavior

When a player with companions is invited to a party:
- `party invite <player>` — invites the player AND all their
  companions join the party automatically
- No separate invite needed for each companion

When a player with companions leaves a party:
- `party leave` — the player AND all their companions leave
- A new party is formed with just the player + their companions
- Remaining party members keep their party intact

This means two players with companions can party up seamlessly:
Player A (with wolf) invites Player B (with swarm) → party of
4: A, wolf, B, swarm. If B leaves → B+swarm form their own
party, A+wolf stay in the original party.

### Auto-Assist Toggle

`companion <name> assist on` / `companion <name> assist off`

Default is ON. Stored in `CompanionInfo.AutoAssist`. When off:
- Companion doesn't attack when owner enters combat
- Owner doesn't auto-defend companion
- Companion still defends itself if directly attacked

## Companion Vitals Display

### Companion Command

`companion` (no args) — shows all companions:

```
━━━ Companions ━━━
  Steppe Spirit Wolf [summoned] (assist: on)
    HP: ████████░░  SP: ██████████  CP: ██████████
  
  Bandit Scout [charmed] (assist: off)
    HP: ██████████  SP: ████████░░  CP: ██████████
```

`companion <name>` — shows detailed info for one companion
(stats, skills, mutations — like a mini `status`).

### Prompt Tokens

`{pet_hp}` — HP percentage of first companion (for prompt display)
`{pet_sp}` — SP percentage of first companion
`{pet_cp}` — CP percentage of first companion

## Dismiss Command

`dismiss <companion>` — removes companion, triggers betrayal.

```
You sever the bond with Steppe Spirit Wolf.
Steppe Spirit Wolf turns on you with fury!
```

The dismissed companion:
1. Charm removed immediately
2. Aggro set on the former owner
3. `CompanionInfo` removed from character
4. Summoned/conjured/raised: despawns when combat ends or after
   a timeout (5 minutes)
5. Charmed: stays in world permanently as a hostile mob

## What This Phase Does NOT Include

- Necromancy (raise dead, corpse assessment) — Phase 3
- Companion AI improvements (spellcaster casting, self-buff) — Phase 4
- Companion progression (stat/skill advancement, mutation) — Phase 5
- Reworking existing summon spells to use new system — Phase 2
- Quest-gated spell implementation (quest engine changes) — Phase 2

Phase 1 builds the foundation: skill, companion data model, party
integration, vitals display, dismiss, and spell routing. Phase 2
ports the existing summon spells and adds quest gating.

## Migration

Existing charmed mobs (steppe spirit wolf, hive swarm) will need
migration code:
- On first login after update, if player has `CharmedMobs` entries,
  convert them to `CompanionInfo` structs
- Old `Pet` field migrates to a `CompanionInfo` with type "pet"
- The old `CharmedMobs []int` field becomes deprecated

## Testing Strategy

- Manifestation skill progression via OnSkillUse
- Companion cap calculation at various skill ranks
- Spell routing: manifestation spells use charisma, traditional use willpower
- CompanionInfo persistence: save, restart, companions respawn
- Party autoassist: companion attacked → owner assists, owner attacked → companion assists
- Dismiss: companion turns hostile, CompanionInfo removed
- Stat scaling formula produces expected values
- Migration: old charmed mobs → new CompanionInfo
