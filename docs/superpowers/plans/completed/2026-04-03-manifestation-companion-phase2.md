# Manifestation + Unified Companion System — Phase 2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Expose the companion system to JS scripting, add scaled mob spawning, rework existing summon spells to use the new companion system with stat scaling, and quest-gate the wolf summon spell via Sylara's quest.

**Architecture:** New JS scripting methods bridge `CompanionInfo` to spell scripts. `SpawnMobScaled` allows caster-stat-based statpool override. Summon spells become manifestation-school with `quest_required` gating.

**Tech Stack:** Go, JS scripting (goja), existing spell/companion systems

---

## File Structure

| File | Responsibility |
|------|---------------|
| `internal/scripting/actor_func.go` | Add companion JS API methods |
| `internal/scripting/room_func.go` | Add SpawnMobScaled |
| `internal/mobs/mobs.go` | Support statpool override in NewMobById |
| `_datafiles/world/dogmud/spells/summon-steppe-spirit.yaml` | Add manifestation school, quest_required |
| `_datafiles/world/dogmud/spells/summon-steppe-spirit.js` | Rewrite to use companion system + scaling |
| `_datafiles/world/dogmud/spells/summon-hive-swarm.yaml` | Add manifestation school |
| `_datafiles/world/dogmud/spells/summon-hive-swarm.js` | Rewrite to use companion system + scaling |
| `internal/characters/companions.go` | Add CalcCompanionStatPool helper |

---

### Task 1: Companion JS Scripting API

**Files:**
- Modify: `internal/scripting/actor_func.go`

Expose the companion system to JS scripts by adding these methods
to `ScriptActor`:

- [ ] **Step 1: Read actor_func.go**

Read `internal/scripting/actor_func.go` to understand the pattern
for exposing Go methods to JS. Find existing `CharmSet`, `HasCharmedMobs`,
etc. for the pattern to follow.

- [ ] **Step 2: Add companion methods**

```go
// AddCompanion registers a mob as a companion of this actor.
// sourceType: "summoned", "conjured", "charmed", "raised", "pet"
// Returns true on success, false if at companion cap.
func (a ScriptActor) AddCompanion(mobInstanceId int, sourceType string, name string) bool

// HasCompanion returns true if this actor has any active companions.
func (a ScriptActor) HasCompanion() bool

// GetCompanionCount returns the number of active companions.
func (a ScriptActor) GetCompanionCount() int

// GetMaxCompanionCount returns the max companions this actor can have.
func (a ScriptActor) GetMaxCompanionCount() int

// RemoveCompanion removes a companion by instance ID.
func (a ScriptActor) RemoveCompanion(mobInstanceId int) bool
```

`AddCompanion` should:
1. Create a `CompanionInfo` with the given params
2. Set `InstanceId = mobInstanceId`
3. Call `a.characterRecord.AddCompanion(info)` — returns false if at cap
4. Also call the existing charm tracking: `a.characterRecord.TrackCharmed(mobInstanceId, true)`

`HasCompanion` checks `len(a.characterRecord.Companions) > 0`.

- [ ] **Step 3: Verify build**

Run: `go build ./...`

- [ ] **Step 4: Commit**

```bash
git commit -m "feat: expose companion system to JS scripting API"
```

---

### Task 2: Scaled Mob Spawning

**Files:**
- Modify: `internal/scripting/room_func.go`
- Modify: `internal/mobs/mobs.go`
- Modify: `internal/characters/companions.go`

- [ ] **Step 1: Add CalcCompanionStatPool**

In `internal/characters/companions.go`, add:

```go
// CalcCompanionStatPool computes the scaled statpool for a
// summoned/conjured/raised companion based on caster stats.
// Formula: baseStatPool × (1.0 + charisma/chaFactor + manifestation×skillFactor)
func CalcCompanionStatPool(baseStatPool int, charisma int, manifestationSkill int) int {
    cfg := configs.GetBalanceConfig()
    chaFactor := float64(cfg.ManifestStatScaleChaFactor)
    skillFactor := float64(cfg.ManifestStatScaleSkillFactor)
    scale := 1.0 + float64(charisma)/chaFactor + float64(manifestationSkill)*skillFactor
    return int(math.Round(float64(baseStatPool) * scale))
}
```

- [ ] **Step 2: Support statpool override in NewMobById**

Read `internal/mobs/mobs.go` and find `NewMobById`. It may already
accept a variadic `forceStatPool` parameter — check the signature.
If it does, we just need to use it. If not, add one:

```go
func NewMobById(mobId MobId, roomId int, forceStatPool ...int) *Mob
```

When `forceStatPool` is provided and > 0, use it instead of the
template's `StatPool` when rolling stats.

- [ ] **Step 3: Add SpawnMobScaled to room scripting**

In `internal/scripting/room_func.go`, add:

```go
// SpawnMobScaled spawns a mob with an overridden statpool.
// Used for companions that scale with caster stats.
func (r ScriptRoom) SpawnMobScaled(mobId int, statPool int) *ScriptActor {
    if mob := mobs.NewMobById(mobs.MobId(mobId), r.roomId, statPool); mob != nil {
        r.roomRecord.AddMob(mob.InstanceId)
        return GetMob(mob.InstanceId)
    }
    return nil
}
```

- [ ] **Step 4: Verify build**

Run: `go build ./...`

- [ ] **Step 5: Commit**

```bash
git commit -m "feat: scaled mob spawning for companion stat scaling"
```

---

### Task 3: Rework Steppe Spirit Wolf Spell

**Files:**
- Modify: `_datafiles/world/dogmud/spells/summon-steppe-spirit.yaml`
- Modify: `_datafiles/world/dogmud/spells/summon-steppe-spirit.js`

- [ ] **Step 1: Update spell YAML**

Change the spell to manifestation school and add quest gating:

```yaml
schools:
  - manifestation
quest_required: "12-end"
```

Keep `type: helpsingle`, `cost`, `base_folds`, etc. unchanged.

- [ ] **Step 2: Rewrite onMagic in JS**

The current `onMagic` handler:
1. Checks `HasCharmedMobs()` for duplicate
2. Consumes component item
3. `room.SpawnMob(243)` with no scaling
4. `wolf.CharmSet(userId, 99999)`

Rewrite to use the companion system:

```javascript
function onMagic(sourceActor, targetActor, spellAggro) {
    var user = sourceActor;
    
    // Check companion cap
    if (user.GetCompanionCount() >= user.GetMaxCompanionCount()) {
        user.SendText("You cannot maintain another companion bond.");
        return true;
    }
    
    // Consume component (Spirit Fetish, item 40031)
    var items = user.GetBackpackItems();
    var consumed = false;
    for (var i = 0; i < items.length; i++) {
        if (items[i].ItemId() == 40031) {
            user.TakeItem(items[i]);
            consumed = true;
            break;
        }
    }
    if (!consumed) {
        user.SendText("The spell fizzles — you need a Spirit Fetish.");
        return true;
    }
    
    // Calculate scaled statpool
    var charisma = user.GetStatValue("charisma");
    var manifestSkill = user.GetSkillLevel("manifestation");
    var basePool = 120; // wolf base statpool
    // Scale formula matches Go: base × (1 + cha/200 + skill×0.02)
    var scale = 1.0 + charisma / 200.0 + manifestSkill * 0.02;
    var scaledPool = Math.round(basePool * scale);
    
    // Spawn scaled wolf
    var room = GetRoom(user.GetRoomId());
    var wolf = room.SpawnMobScaled(243, scaledPool);
    if (!wolf) {
        user.SendText("The summoning fails — the spirit cannot take form here.");
        return true;
    }
    
    // Charm and register as companion
    wolf.CharmSet(user.UserId(), 99999);
    user.AddCompanion(wolf.MobInstanceId(), "summoned", "Steppe Spirit Wolf");
    
    // Flavor text
    user.SendText("A spectral wolf materializes beside you, bound to your will.");
    room.SendText(user.GetCharacterName(true) + " summons a spectral wolf.");
    
    return true;
}
```

Read the existing JS to understand the exact API calls used
(e.g., `sourceActor.UserId()`, `GetBackpackItems()`, `TakeItem()`).
Match the exact method names.

Also check: does `user.GetStatValue("charisma")` exist? Or is it
`user.GetStat("charisma")`? Read `actor_func.go` for the exact
stat access method.

Similarly check `user.GetSkillLevel("manifestation")` — verify the
exact JS method name.

- [ ] **Step 3: Keep onCast and onWait handlers**

The `onCast` handler currently checks for the component item and
for existing charmed mobs. Update the duplicate check to use
`GetCompanionCount() >= GetMaxCompanionCount()` instead of
`HasCharmedMobs()`.

Keep `onWait` flavor text unchanged.

- [ ] **Step 4: Commit**

```bash
git commit -m "feat: rework steppe spirit wolf to use companion system + scaling"
```

---

### Task 4: Rework Hive Swarm Spell

**Files:**
- Modify: `_datafiles/world/dogmud/spells/summon-hive-swarm.yaml`
- Modify: `_datafiles/world/dogmud/spells/summon-hive-swarm.js`

- [ ] **Step 1: Update spell YAML**

```yaml
schools:
  - manifestation
# No quest_required — hive swarm is discoverable
```

- [ ] **Step 2: Rewrite JS**

Same pattern as the wolf spell but with hive swarm mob ID (111)
and base statpool (18). No quest gating — this spell can be
discovered normally through manifestation skill.

Remove the `MiscCharacterData('hive-swarm-active')` hack — use
the companion system's cap check instead.

- [ ] **Step 3: Commit**

```bash
git commit -m "feat: rework hive swarm to use companion system + scaling"
```

---

### Task 5: Verify Quest Gating

- [ ] **Step 1: Check quest 12 rewards**

Read the quest 12 YAML (`_datafiles/world/dogmud/quests/12-*.yaml`)
to verify it grants `spellid: summon-steppe-spirit`. This is the
existing quest reward — it should still work with the new
`quest_required` field since the spell is taught by the quest, not
discovered.

- [ ] **Step 2: Verify discovery exclusion**

The `GetEligibleSpells` function (modified in Phase 1) excludes
spells where `QuestRequired != ""`. Verify this by checking the
Go code — the wolf spell with `quest_required: "12-end"` should
never appear in skill-based discovery.

The hive swarm (no `quest_required`) should still be discoverable
when manifestation skill reaches the threshold.

- [ ] **Step 3: Commit if any fixes needed**

---

### Task 6: Tests

**Files:**
- Create or modify: `internal/characters/companions_test.go`
- Create or modify: `internal/scripting/` tests if feasible

- [ ] **Step 1: Test CalcCompanionStatPool**

```go
func TestCalcCompanionStatPool(t *testing.T) {
    // Base 120, Cha 100, Manifest 0 → 120 × 1.5 = 180
    result := CalcCompanionStatPool(120, 100, 0)
    assert.Equal(t, 180, result)
    
    // Base 120, Cha 100, Manifest 25 → 120 × 2.0 = 240
    result = CalcCompanionStatPool(120, 100, 25)
    assert.Equal(t, 240, result)
    
    // Base 18 (hive swarm), Cha 100, Manifest 0 → 18 × 1.5 = 27
    result = CalcCompanionStatPool(18, 100, 0)
    assert.Equal(t, 27, result)
}
```

- [ ] **Step 2: Test spell quest_required exclusion**

Verify spells with `QuestRequired` are excluded from
`GetEligibleSpells`. Seed a test spell with `QuestRequired` set
and verify it never appears in results.

- [ ] **Step 3: Run tests + commit**

```bash
git commit -m "test: companion scaling + quest-gated spell tests"
```

---

### Task 7: Final Verification

- [ ] **Step 1: Full build + tests**

Run: `go build ./...`
Run: `go test ./... -count=1 -timeout 300s`

- [ ] **Step 2: Manual smoke test**

Test:
- Complete Quest 12 (Sylara's path) → learn summon-steppe-spirit
- Cast the wolf spell → wolf spawns with scaled stats
- `companion` → wolf appears in list with vitals
- `companion wolf` → descriptive stat comparison
- `dismiss wolf` → wolf turns hostile
- Log out → log in → wolf respawns with saved state
- Level up manifestation → wolf spawns stronger next time
- Hive swarm: cast → swarm spawns scaled, appears in companion list
- Check `spells` → manifestation section shows learned spells
- Verify wolf spell doesn't appear in discovery for characters who
  haven't completed Q12

- [ ] **Step 3: Commit any fixups**
