# Manifestation + Unified Companion System — Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the manifestation skill (charisma-based), unified companion system merging pets and charmed mobs, party integration with bidirectional autoassist, companion persistence across restarts, vitals display, dismiss command, and spell routing for manifestation-school spells.

**Architecture:** New `CompanionInfo` struct persisted on Character. Manifestation skill registered alongside existing skills. Spell routing uses spell school to determine stat+skill pair. Companion combat uses existing charmed mob assist pattern (not party struct). New `companion` and `dismiss` user commands.

**Tech Stack:** Go, testify/assert, existing spell/buff/party/combat systems

---

## File Structure

| File | Responsibility |
|------|---------------|
| `internal/skills/skills.go` | Add Manifestation skill constant + registration |
| `internal/characters/companions.go` | CompanionInfo struct, companion management methods |
| `internal/characters/character.go` | Add Companions field, migration from Pet/CharmedMobs |
| `internal/characters/casting.go` | Generalize CalcFoldsPerRound for school-based routing |
| `internal/spells/spells.go` | Add SchoolManifestation, quest_required field |
| `internal/spells/discovery.go` | School-aware spell discovery |
| `internal/usercommands/companion.go` | `companion` command (vitals, details, assist toggle) |
| `internal/usercommands/dismiss.go` | `dismiss` command (betrayal mechanic) |
| `internal/usercommands/usercommands.go` | Register companion + dismiss commands |
| `internal/users/userrecord.prompt.go` | Add {pet_hp}, {pet_sp}, {pet_cp} tokens |
| `internal/hooks/NewRound_MobRoundTick.go` | Companion respawn on login |
| `internal/hooks/PlayerDespawn_HandleLeave.go` | Companion save on logout |
| `internal/hooks/NewRound_DoCombat_helpers.go` | Bidirectional autoassist |
| `internal/actions/cast.go` | Route spell school to correct stat+skill |
| `internal/usercommands/party.go` | Companion join/leave with party |
| `_datafiles/config.yaml` | Manifestation progression multiplier + scaling config |
| `_datafiles/world/dogmud/templates/help/manifestation.template` | Help file |
| `_datafiles/world/dogmud/templates/help/companion.template` | Help file |
| `internal/characters/companions_test.go` | Tests |
| `internal/actions/cast_test.go` | Updated cast tests |

---

### Task 1: Manifestation Skill Registration

**Files:**
- Modify: `internal/skills/skills.go`
- Modify: `_datafiles/config.yaml`

- [ ] **Step 1: Read skills.go to find the skill constants and registration**

Read `internal/skills/skills.go` fully. Find:
- The `SkillTag` constants block
- The `SkillPrimaryStats` map
- The `GetSkillTier()` function with `totalSkills` const
- The `SkillProgressionMultipliers` map (if it exists in code)

- [ ] **Step 2: Add Manifestation skill constant**

In the `SkillTag` constants block, add:
```go
Manifestation SkillTag = "manifestation"
```

In the `SkillPrimaryStats` map, add:
```go
"manifestation": "charisma",
```

Update `totalSkills` in `GetSkillTier()` to include the new skill (increment by 1).

- [ ] **Step 3: Add config entry**

In `_datafiles/config.yaml` under `SkillProgressionMultipliers`, add:
```yaml
    manifestation: 0.30  # Companion management — moderate use frequency
```

- [ ] **Step 4: Verify build + test**

Run: `go build ./...`
Run: `go test ./internal/skills/ -v -count=1`

- [ ] **Step 5: Commit**

```bash
git commit -m "feat: add manifestation skill (charisma-based)"
```

---

### Task 2: CompanionInfo Struct + Character Integration

**Files:**
- Create: `internal/characters/companions.go`
- Modify: `internal/characters/character.go`

- [ ] **Step 1: Create companions.go**

```go
package characters

// CompanionSourceType describes how the companion was acquired.
type CompanionSourceType string

const (
    CompanionSummoned CompanionSourceType = "summoned"
    CompanionConjured CompanionSourceType = "conjured"
    CompanionCharmed  CompanionSourceType = "charmed"
    CompanionRaised   CompanionSourceType = "raised"
    CompanionPet      CompanionSourceType = "pet"
)

// CompanionInfo stores the persistent state of a companion.
// Runtime mob instance state (aggro, buffs, casting) is NOT saved —
// companions respawn fresh on login.
type CompanionInfo struct {
    MobId            int                 `yaml:"mobid"`
    InstanceId       int                 `yaml:"-"` // runtime only
    SourceType       CompanionSourceType `yaml:"source_type"`
    Name             string              `yaml:"name"`
    AutoAssist       bool                `yaml:"auto_assist"`
    // Persisted progression
    StatTraining     map[string]int      `yaml:"stat_training,omitempty"`
    Skills           map[string]int      `yaml:"skills,omitempty"`
    SkillUseCount    map[string]int      `yaml:"skill_use_count,omitempty"`
    Mutations        map[string]int      `yaml:"mutations,omitempty"`
    SpellBook        map[string]int      `yaml:"spellbook,omitempty"`
    MutationProgress float64             `yaml:"mutation_progress,omitempty"`
}

// GetCompanion returns the CompanionInfo for a companion by name
// (case-insensitive partial match). Returns nil if not found.
func (c *Character) GetCompanion(name string) *CompanionInfo {
    nameLower := strings.ToLower(name)
    for i := range c.Companions {
        if strings.Contains(
            strings.ToLower(c.Companions[i].Name), nameLower) {
            return &c.Companions[i]
        }
    }
    return nil
}

// AddCompanion adds a companion. Returns false if at max capacity.
func (c *Character) AddCompanion(info CompanionInfo) bool {
    if len(c.Companions) >= c.GetMaxCompanions() {
        return false
    }
    info.AutoAssist = true // default on
    c.Companions = append(c.Companions, info)
    return true
}

// RemoveCompanion removes a companion by instance ID.
// Returns the removed CompanionInfo or nil.
func (c *Character) RemoveCompanion(instanceId int) *CompanionInfo {
    for i, comp := range c.Companions {
        if comp.InstanceId == instanceId {
            removed := c.Companions[i]
            c.Companions = append(
                c.Companions[:i], c.Companions[i+1:]...)
            return &removed
        }
    }
    return nil
}

// GetMaxCompanions returns the max companions based on
// manifestation skill. Minimum 1 if any manifestation spell known.
func (c *Character) GetMaxCompanions() int {
    skill := c.GetSkillLevel(skills.Manifestation)
    max := skill / 19 // 0-18=0, 19-37=1, 38-56=2, 57-75=3, 76+=4
    if max > 4 {
        max = 4
    }
    // Minimum 1 if player knows any manifestation spell
    if max < 1 && c.KnowsManifestationSpell() {
        max = 1
    }
    return max
}

// KnowsManifestationSpell returns true if the character knows
// at least one spell with the manifestation school.
func (c *Character) KnowsManifestationSpell() bool {
    for spellId := range c.SpellBook {
        sp := spells.GetSpell(spellId)
        if sp != nil && sp.HasSchool(spells.SchoolManifestation) {
            return true
        }
    }
    return false
}

// GetActiveCompanionMob returns the mob instance for a companion,
// or nil if the companion isn't active.
func (c *Character) GetActiveCompanionMob(comp *CompanionInfo) *mobs.Mob {
    if comp.InstanceId == 0 {
        return nil
    }
    return mobs.GetInstance(comp.InstanceId)
}
```

Add `"strings"` import and any needed imports (`skills`, `spells`, `mobs`).
Note: check for import cycles — `characters` importing `spells` or `mobs`
may cause issues. If so, use interface or move the spell-check helper
to a different package.

- [ ] **Step 2: Add Companions field to Character**

In `internal/characters/character.go`, add to the Character struct:

```go
Companions []CompanionInfo `yaml:"companions,omitempty"`
```

Place it near the existing `Pet` field (line ~108).

- [ ] **Step 3: Verify build**

Run: `go build ./...`

If there are import cycle issues between characters → spells or
characters → mobs, resolve by:
- Moving `KnowsManifestationSpell` to a helper in `internal/actions/`
- Or using string comparison instead of `spells.GetSpell`

- [ ] **Step 4: Commit**

```bash
git commit -m "feat: CompanionInfo struct + Character.Companions field"
```

---

### Task 3: Spell School Routing

**Files:**
- Modify: `internal/spells/spells.go`
- Modify: `internal/characters/casting.go`
- Modify: `internal/actions/cast.go`

- [ ] **Step 1: Add SchoolManifestation + quest_required to SpellData**

In `internal/spells/spells.go`, add the school constant:
```go
SchoolManifestation = "manifestation"
```

Add `QuestRequired` field to SpellData:
```go
QuestRequired string `yaml:"quest_required,omitempty"`
```

Add a helper method:
```go
func (s *SpellData) HasSchool(school string) bool {
    for _, sc := range s.Schools {
        if sc == school {
            return true
        }
    }
    return false
}
```

- [ ] **Step 2: Generalize CalcFoldsPerRound**

Read the current `CalcFoldsPerRound` in `internal/characters/casting.go`.
The function takes `(perception, spellcastingLevel int)`. These are
positional — the function doesn't care what they represent. The callers
need to resolve which stat+skill to pass based on spell school.

Rename the parameters for clarity:
```go
func CalcFoldsPerRound(primaryStat, skillLevel int) int {
    // ... same body, just renamed params
}
```

- [ ] **Step 3: Add spell school → stat+skill resolver**

In `internal/actions/cast.go`, add a helper that the `InitiateCast`
function calls to determine which stat and skill to use:

```go
// GetSpellStatAndSkill returns the primary stat value and skill
// level for a spell based on its school.
func GetSpellStatAndSkill(char *characters.Character, spellData *spells.SpellData) (statValue int, skillLevel int) {
    if spellData.HasSchool(spells.SchoolManifestation) {
        return char.Stats.Charisma.ValueAdj,
            char.GetSkillLevel(skills.Manifestation)
    }
    // Default: willpower + spellcasting
    return char.Stats.Willpower.ValueAdj,
        char.GetSkillLevel(skills.Spellcasting)
}
```

Update `InitiateCast` to use this helper instead of hardcoding
willpower+spellcasting for fold calculation.

- [ ] **Step 4: Update spell discovery**

Read where spell discovery fires (in `handlePlayerFoldCasting` and
`handleMobFoldCasting`). Currently it calls
`spells.GetEligibleSpells(spellBook, skillLevel)` with spellcasting
skill hardcoded.

Add a school filter to `GetEligibleSpells`:
```go
func GetEligibleSpells(spellBook map[string]int, skillLevel int, school string) []string
```

The function should only return spells matching the given school.
Call it twice in the fold handler — once for spellcasting spells,
once for manifestation spells (if the caster has manifestation rank).

Also: spells with `QuestRequired != ""` should NEVER appear in
discovery results regardless of skill level.

- [ ] **Step 5: Verify build + tests**

Run: `go build ./...`
Run: `go test ./internal/actions/ -run Cast -v -count=1`

- [ ] **Step 6: Commit**

```bash
git commit -m "feat: spell school routing — manifestation uses charisma+manifestation"
```

---

### Task 4: Companion Command

**Files:**
- Create: `internal/usercommands/companion.go`
- Modify: `internal/usercommands/usercommands.go`

- [ ] **Step 1: Create companion command**

`internal/usercommands/companion.go`:

The command handles three forms:
- `companion` — list all companions with vitals bars
- `companion <name>` — show detailed info for one companion
- `companion <name> assist on/off` — toggle autoassist

For the vitals display, read how the `status` command or the
vitals web panel formats HP/SP/CP bars. Use a similar pattern
with colored bars.

Output format for `companion` (no args):
```
━━━ Companions ━━━
  Steppe Spirit Wolf [summoned] (assist: on)
    HP: ████████░░  SP: ██████████  CP: ██████████

  Bandit Scout [charmed] (assist: off)
    HP: ██████████  SP: ████████░░  CP: ██████████
```

For `companion <name>`:
```
━━━ Steppe Spirit Wolf [summoned] ━━━
  HP: 45/50  SP: 30/30  CP: 25/25
  Strength: 110  Dexterity: 95  Perception: 80
  Vitality: 100  Willpower: 70  Charisma: 60
  Auto-Assist: on
```

For the assist toggle, find the companion by name, flip
`AutoAssist`, send confirmation.

Read `internal/mobs/mobs.go` to understand how to get mob
vitals from the instance ID stored in CompanionInfo.

- [ ] **Step 2: Register command**

In `internal/usercommands/usercommands.go`, add:
```go
`companion`: {Companion, true, true, false},
```

- [ ] **Step 3: Verify build**

Run: `go build ./...`

- [ ] **Step 4: Commit**

```bash
git commit -m "feat: companion command — vitals display + assist toggle"
```

---

### Task 5: Dismiss Command

**Files:**
- Create: `internal/usercommands/dismiss.go`
- Modify: `internal/usercommands/usercommands.go`

- [ ] **Step 1: Create dismiss command**

`internal/usercommands/dismiss.go`:

```go
func Dismiss(rest string, user *users.UserRecord,
    room *rooms.Room, flags events.EventFlag) (bool, error)
```

The command:
1. Parse companion name from `rest`
2. Find companion in `user.Character.Companions` by name
3. Get the mob instance via `mobs.GetInstance(comp.InstanceId)`
4. Remove charm from mob: `mob.Character.RemoveCharm()`
5. Set mob aggro on the player: `mob.Character.SetAggro(user.UserId, 0, characters.DefaultAttack)`
6. Remove from `user.Character.Companions`
7. Remove from `user.Character.CharmedMobs` (legacy field)
8. Send messages:
   - To player: "You sever the bond with [name].\n[name] turns on you with fury!"
   - To room: "[player] dismisses [name]!\n[name] turns hostile!"
9. For summoned/conjured/raised: set a despawn timer on the mob
   (after combat ends or 5 minute timeout). Read how `despawn`
   works in `internal/mobcommands/despawn.go`.

- [ ] **Step 2: Register command**

```go
`dismiss`: {Dismiss, false, true, false},
```

- [ ] **Step 3: Verify build**

Run: `go build ./...`

- [ ] **Step 4: Commit**

```bash
git commit -m "feat: dismiss command — full betrayal mechanic"
```

---

### Task 6: Companion Persistence (Login/Logout)

**Files:**
- Modify: `internal/hooks/PlayerDespawn_HandleLeave.go`
- Modify: `internal/hooks/` (find login handler)

- [ ] **Step 1: Save companions on logout**

Find where player logout is handled — read
`internal/hooks/PlayerDespawn_HandleLeave.go`. Find where charmed
mobs are currently expired on logout.

Replace/augment the existing charm cleanup with companion save:
1. For each companion in `user.Character.Companions`:
2. Get mob instance via `mobs.GetInstance(comp.InstanceId)`
3. Save current stat training, skills, mutations, spellbook from
   `mob.Character` back to `CompanionInfo`
4. Despawn the mob instance
5. Set `comp.InstanceId = 0`

The `CompanionInfo` persists as part of Character YAML automatically
since the field has `yaml:"companions,omitempty"`.

- [ ] **Step 2: Respawn companions on login**

Find where player login is handled — search for `OnLoginCommands`
or the login hook in `internal/hooks/`. May be in
`PlayerSpawn_HandleJoin.go` or similar.

Add companion respawn logic:
1. For each companion in `user.Character.Companions`:
2. Spawn mob: `mobs.NewMobById(comp.MobId, user.Character.RoomId)`
3. Apply saved state: copy StatTraining, Skills, Mutations, SpellBook
   from CompanionInfo to the mob's Character
4. Set full HP/SP/CP on the mob
5. Charm the mob to the player: `mob.Character.Charm(userId, -1, "")`
   where -1 = permanent
6. Register in `user.Character.CharmedMobs` via `TrackCharmed`
7. Set `comp.InstanceId = mob.InstanceId`
8. Add mob to room

- [ ] **Step 3: Verify build**

Run: `go build ./...`

- [ ] **Step 4: Commit**

```bash
git commit -m "feat: companion persistence — save on logout, respawn on login"
```

---

### Task 7: Bidirectional Autoassist

**Files:**
- Modify: `internal/hooks/NewRound_DoCombat_helpers.go`
- Modify: `internal/usercommands/attack.go`

- [ ] **Step 1: Read existing autoassist code**

Read `handleCharmedMobAssist` in `NewRound_DoCombat_helpers.go`.
This handles companion→owner assist (companion attacks when owner
is attacked). Also read `attack.go` lines 206-214 for the owner→
companion assist (charmed mobs attack when owner attacks).

- [ ] **Step 2: Add owner autoassist when companion is attacked**

In `handleMobVsPlayer` (or equivalent handler where a mob attacks
a player), add a check: if the defender is a charmed mob, look up
the charm owner. If the owner has autoattack on and the companion
has `AutoAssist=true`, have the owner attack the aggressor.

Actually — mobs attacking a charmed mob goes through `handleMobVsMob`.
Find where mob-vs-mob combat fires. When a mob attacks a charmed mob:
1. Look up the charm owner (userId from `mob.Character.GetCharmedUserId()`)
2. Get the owner's user record
3. Check if the companion's `AutoAssist` is true
4. If the owner isn't already fighting, set owner's aggro on the attacker
5. Also trigger other party members' autoassist for the companion

For party member autoassist: when a companion is attacked, treat it
the same as the owner being attacked for autoassist purposes.

- [ ] **Step 3: Verify build**

Run: `go build ./...`

- [ ] **Step 4: Commit**

```bash
git commit -m "feat: bidirectional autoassist — owner defends companion"
```

---

### Task 8: Prompt Tokens

**Files:**
- Modify: `internal/users/userrecord.prompt.go`

- [ ] **Step 1: Add companion vitals prompt tokens**

Read `internal/users/userrecord.prompt.go` and find the
`ProcessPromptString` function's switch statement.

Add cases for `{pet_hp}`, `{pet_sp}`, `{pet_cp}`:

```go
case "{pet_hp}":
    // Get first companion's HP percentage
    if len(u.Character.Companions) > 0 {
        comp := u.Character.Companions[0]
        if mob := mobs.GetInstance(comp.InstanceId); mob != nil {
            pct := mob.Character.Health * 100 / mob.Character.HealthMax.Value
            // Format as colored percentage or bar
            replacement = fmt.Sprintf("%d%%", pct)
        }
    }
```

Same pattern for `{pet_sp}` (stamina) and `{pet_cp}` (conviction).

- [ ] **Step 2: Verify build**

Run: `go build ./...`

- [ ] **Step 3: Commit**

```bash
git commit -m "feat: {pet_hp}, {pet_sp}, {pet_cp} prompt tokens"
```

---

### Task 9: Party Integration (Join/Leave with Companions)

**Files:**
- Modify: `internal/usercommands/party.go`

- [ ] **Step 1: Read party.go**

Read `internal/usercommands/party.go` fully. Understand how
`party invite`, `party join`, and `party leave` work.

- [ ] **Step 2: Augment party join**

When a player joins a party (accepts invite), their companions
should be recognized by the party's autoassist system. Since the
party struct is user-ID-only, we don't add mob IDs to it. Instead,
the autoassist system already scans for charmed mobs in the room.

The main change: when displaying party vitals (the `Party.Vitals`
GMCP data or the party command output), include companion vitals
alongside player vitals. Read how party vitals are sent to clients.

- [ ] **Step 3: Augment party leave**

When a player leaves a party, no special companion handling is
needed — companions stay charmed to the player regardless of
party membership. The autoassist system works off charm ownership,
not party membership.

The key change: ensure that when party autoassist fires for a
player, it also considers defending that player's companions.
This was handled in Task 7.

- [ ] **Step 4: Verify build + commit**

```bash
git commit -m "feat: party integration — companion vitals in party display"
```

---

### Task 10: Help Files + Config

**Files:**
- Create: `_datafiles/world/dogmud/templates/help/manifestation.template`
- Create: `_datafiles/world/dogmud/templates/help/companion.template`
- Create: `_datafiles/world/dogmud/templates/help/dismiss.template`
- Modify: `_datafiles/config.yaml`

- [ ] **Step 1: Create help files**

Follow the existing help file format (see `help/equipment.template`).

**manifestation.template:**
Cover: what the skill does, companion cap formula, how it affects
spellcasting for manifestation-school spells, progression.

**companion.template:**
Cover: `companion` command usage (list, details, assist toggle),
companion types (summoned, charmed, raised, pet), persistence,
dismiss warning.

**dismiss.template:**
Cover: what happens when you dismiss (full betrayal), warning that
summoned/raised companions despawn after, charmed mobs revert.

- [ ] **Step 2: Add config knobs**

In `_datafiles/config.yaml` under Balance, add:
```yaml
  ManifestStatScaleChaFactor: 200    # Charisma divisor for companion stat scaling
  ManifestStatScaleSkillFactor: 0.02 # Manifestation skill multiplier for scaling
```

- [ ] **Step 3: Commit**

```bash
git commit -m "feat: help files for manifestation, companion, dismiss + config knobs"
```

---

### Task 11: Anti-Recursion — Strip Companions on Charm

**Files:**
- Modify: `internal/characters/character.go` (in Charm method or companions.go)

When a mob becomes a player's companion (charmed/summoned), any
companions that mob had are stripped. No recursive companion chains.

- [ ] **Step 1: Add companion stripping to Charm**

In the `Charm()` method on Character (or in the companion creation
flow), when a mob is charmed:
1. Check if the mob has any charmed mobs of its own (`CharmedMobs`)
2. For each, remove the charm and despawn them
3. Clear the mob's `CharmedMobs` list

Also: mobs with the manifestation skill can use it for their own
summoning via mob AI (a necromancer mob raising minions in combat).
The skill works identically for mobs via the shared cast system.
But when that mob gets charmed by a player, its minions are gone.

- [ ] **Step 2: Verify build + commit**

```bash
git commit -m "feat: strip companion chains when mob becomes a companion (no recursion)"
```

---

### Task 12: Migration (Pet + CharmedMobs → Companions)


**Files:**
- Modify: `internal/characters/character.go` (in Validate or a migration hook)

- [ ] **Step 1: Add migration in Validate**

In `Character.Validate()`, add a migration block that runs once:
1. If `Pet.Exists()` and no companion with type "pet" exists:
   - Create `CompanionInfo` from Pet data
   - Add to `Companions`
   - Clear `Pet` field
2. Log the migration

Note: `CharmedMobs` is runtime-only (`yaml:"-"`), so there's nothing
to migrate from disk for charmed mobs. The migration is only for
the persisted Pet field.

- [ ] **Step 2: Verify build + test**

- [ ] **Step 3: Commit**

```bash
git commit -m "feat: migrate Pet field to Companions on first load"
```

---

### Task 13: Tests

**Files:**
- Create: `internal/characters/companions_test.go`

- [ ] **Step 1: Write companion tests**

Tests needed:
1. `TestGetMaxCompanions_Ranks` — verify companion cap at various
   manifestation skill levels (0, 18, 19, 37, 38, 56, 57, 75)
2. `TestAddCompanion_AtCap` — adding when at max returns false
3. `TestRemoveCompanion_ByInstanceId` — removes correct companion
4. `TestGetCompanion_PartialMatch` — name search is case-insensitive
5. `TestCompanionInfo_Persistence` — marshal/unmarshal CompanionInfo
   to YAML and verify all fields survive
6. `TestGetSpellStatAndSkill_Manifestation` — manifestation school
   returns charisma+manifestation
7. `TestGetSpellStatAndSkill_Default` — non-manifestation returns
   willpower+spellcasting
8. `TestQuestRequiredSpell_NotInDiscovery` — spell with
   QuestRequired set never appears in GetEligibleSpells

- [ ] **Step 2: Run tests**

Run: `go test ./internal/characters/ ./internal/actions/ ./internal/spells/ -v -count=1`

- [ ] **Step 3: Commit**

```bash
git commit -m "test: companion system + spell routing tests"
```

---

### Task 14: Final Verification

- [ ] **Step 1: Full build + all tests**

Run: `go build ./...`
Run: `go test ./... -count=1 -timeout 300s`

- [ ] **Step 2: Manual smoke test**

Test:
- `companion` command with no companions → empty display
- Summon a spirit wolf → appears in `companion` list
- `companion wolf assist off` → toggles assist
- `dismiss wolf` → wolf turns hostile, attacks
- Log out with companion → log back in → companion respawns
- `{pet_hp}` in prompt shows companion HP
- Cast a manifestation spell → uses charisma for fold rate
- Check `skills` → manifestation appears

- [ ] **Step 3: Commit any fixups**
