# Phase 4b: Mob Script Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate 7 mob JS scripts to behavior trees, implement perception-scaled reaction delays, upgrade two boss mobs with improved AI, and add cross-mob interaction for Dal.

**Architecture:** Three tiers — (1) migrate easy quest NPCs using existing engine features, (2) add new events/conditions/actions/reaction delays to the engine, (3) create upgraded boss AI and Dal patrol heckling using the new features.

**Tech Stack:** Go, YAML, existing behaviortree/hooks/mobs/characters packages

**Spec:** `docs/superpowers/specs/2026-04-14-phase4b-mob-script-migration-design.md`

---

## File Structure

```
internal/behaviortree/
├── conditions.go     — add 4 new conditions
├── actions.go        — add ~10 new actions
├── engine.go         — add negative caching (noTree map)
├── helpers.go        — check noTree before os.Stat
├── context.md        — full package documentation (NEW)

internal/hooks/
├── companion_summon.go — extract reusable companion spawn helper
├── MobIdle_HandleIdleMobs.go — (already wired, no changes)

internal/mobcommands/
├── suicide.go        — wire mob_die behavior tree event
├── flee.go           — wire mob_flee behavior tree event

internal/usercommands/
├── go.go             — wire player_enter behavior tree event

_datafiles/world/dogmud/behaviors/
├── dustwalk_road/83-road_warden_tessara.yaml    (NEW)
├── thornwall_city/99-records_clerk_pell.yaml     (NEW)
├── ironwind_steppe/242-geomancer_rhett.yaml      (NEW)
├── ironwind_steppe/241-windwarden_sylara.yaml     (NEW)
├── tutorial/58-training_dummy.yaml                (NEW)
├── marches_spur_road/275-old_edrin.yaml           (NEW)
├── thornwall_city/272-chrysalis_phantom.yaml      (NEW)
├── thornwall_city/117-barmaid_dal.yaml            (MODIFY)

_datafiles/world/dogmud/mobs/
├── marches_spur_road/275-old_edrin.yaml           (MODIFY — add skills)
├── thornwall_city/272-chrysalis_phantom.yaml      (MODIFY — add search)
```

---

### Task 1: Migrate Quest NPCs — Tessara, Pell, Rhett, Sylara

**Files:**
- Create: `_datafiles/world/dogmud/behaviors/dustwalk_road/83-road_warden_tessara.yaml`
- Create: `_datafiles/world/dogmud/behaviors/thornwall_city/99-records_clerk_pell.yaml`
- Create: `_datafiles/world/dogmud/behaviors/ironwind_steppe/242-geomancer_rhett.yaml`
- Create: `_datafiles/world/dogmud/behaviors/ironwind_steppe/241-windwarden_sylara.yaml`
- Rename: 4 JS files to `.js.bak`

These mobs have dialogue YAMLs for conversation. The behavior trees handle
`player_give` events (item rejection, edge cases). Quest engine triggers
already handle core quest advancement for most of these.

- [ ] **Step 1: Read existing JS and dialogue for all 4 mobs**

Read these files to understand the exact behavior:
- `_datafiles/world/dogmud/mobs/dustwalk_road/scripts/83-road_warden_tessara.js`
- `_datafiles/world/dogmud/mobs/thornwall_city/scripts/99-records_clerk_pell.js`
- `_datafiles/world/dogmud/mobs/ironwind_steppe/scripts/242-geomancer_rhett.js`
- `_datafiles/world/dogmud/mobs/ironwind_steppe/scripts/241-windwarden_sylara.js`
- `_datafiles/world/dogmud/dialogue/dustwalk_road/83.yaml`
- `_datafiles/world/dogmud/dialogue/thornwall_city/99.yaml`
- `_datafiles/world/dogmud/dialogue/ironwind_steppe/242.yaml`
- `_datafiles/world/dogmud/dialogue/ironwind_steppe/241.yaml`

Also read the quest engine triggers for Q4, Q6, Q9, Q11 to understand
what the quest engine already handles vs. what the behavior tree needs
to add:
- `_datafiles/world/dogmud/quests/4-*.yaml`
- `_datafiles/world/dogmud/quests/6-*.yaml`
- `_datafiles/world/dogmud/quests/9-*.yaml`
- `_datafiles/world/dogmud/quests/11-*.yaml`

- [ ] **Step 2: Create Tessara behavior tree**

Create `_datafiles/world/dogmud/behaviors/dustwalk_road/83-road_warden_tessara.yaml`.

Tree handles `player_give` event:
- Sequence: `item_matches(16)` + `player_has_quest("4-start")` +
  `player_missing_quest("4-end")` → respond with flavor text. (Check
  whether the quest engine trigger already grants Q4-end — if so, the
  tree just adds flavor. If not, add `grant_quest("4-end")`.)
- Default: say rejection, return item via `give_item` to player.

- [ ] **Step 3: Create Pell behavior tree**

Create `_datafiles/world/dogmud/behaviors/thornwall_city/99-records_clerk_pell.yaml`.

Tree handles `player_give` event with item discrimination:
- Sequence for item 31 (bridge report): `item_matches(31)` +
  `player_has_quest("6-start")` + `player_missing_quest("6-report")` →
  respond, grant Q6-report.
- Sequence for item 31 already filed: `item_matches(31)` +
  `player_has_quest("6-report")` → say "already filed".
- Sequence for item 29 (tithe ledger) + Q9 active: `item_matches(29)` +
  `player_has_quest("9-start")` + `player_missing_quest("9-end")` →
  redirect to Olen, return ledger via `give_item(29)`.
- Sequence for item 29 without Q9: `item_matches(29)` → reject, return.
- Default: bureaucratic rejection.

- [ ] **Step 4: Create Rhett behavior tree**

Create `_datafiles/world/dogmud/behaviors/ironwind_steppe/242-geomancer_rhett.yaml`.

Tree handles `player_give` event:
- Sequence: `item_matches(40032)` + `player_has_quest("11-start")` +
  `player_missing_quest("11-end")` → respond, grant Q11-end.
- Sequence: `item_matches(40032)` + `player_has_quest("11-end")` →
  "already have what I need".
- Default: reject, return item.

- [ ] **Step 5: Create Sylara behavior tree**

Create `_datafiles/world/dogmud/behaviors/ironwind_steppe/241-windwarden_sylara.yaml`.

Two event branches:

`player_give`:
- Sequence: `item_matches(40033)` + `player_has_quest("11-start")` +
  `player_missing_quest("11-end")` → respond, grant Q11-end.
- Sequence: `item_matches(40033)` + `player_has_quest("11-end")` →
  "totem is where it belongs".
- Default: reject, return item.

`player_ask` (fetish dispensing — needs new conditions/actions from
Tier 2, but the tree structure can be authored now with the action
names that will exist after Tier 2):
- Keyword match: fetish, spirit, summon, component, more
- Gate: `player_has_quest("12-end")` OR `player_has_spell("summon-steppe-spirit")`
  (needs `player_has_spell` from Tier 2 — use a selector with both checks)
- First-time bonus: `player_has_misc_data("sylara-bonus-fetishes-given", "1")`
  inverted → `give_item_multiple(40031, 4)` + `set_misc_data(...)`.
  (These actions are added in Tier 2.)
- Already carrying: `player_has_item(40031)` → "you already carry one".
- Subsequent: `give_item(40031)`.

NOTE: The `player_has_spell`, `player_has_misc_data`, `give_item_multiple`,
and `set_misc_data` actions don't exist yet. Author the YAML now with
these names — they'll be implemented in Task 3. The tree won't load
cleanly until Task 3 is done. Alternatively, defer the `player_ask`
branch entirely to after Task 3 and only implement `player_give` now.

- [ ] **Step 6: Rename JS files to .bak**

```bash
mv _datafiles/world/dogmud/mobs/dustwalk_road/scripts/83-road_warden_tessara.js \
   _datafiles/world/dogmud/mobs/dustwalk_road/scripts/83-road_warden_tessara.js.bak
mv _datafiles/world/dogmud/mobs/thornwall_city/scripts/99-records_clerk_pell.js \
   _datafiles/world/dogmud/mobs/thornwall_city/scripts/99-records_clerk_pell.js.bak
mv _datafiles/world/dogmud/mobs/ironwind_steppe/scripts/242-geomancer_rhett.js \
   _datafiles/world/dogmud/mobs/ironwind_steppe/scripts/242-geomancer_rhett.js.bak
mv _datafiles/world/dogmud/mobs/ironwind_steppe/scripts/241-windwarden_sylara.js \
   _datafiles/world/dogmud/mobs/ironwind_steppe/scripts/241-windwarden_sylara.js.bak
```

- [ ] **Step 7: Verify build and commit**

```bash
go build ./...
git add _datafiles/world/dogmud/behaviors/ _datafiles/world/dogmud/mobs/
git commit -m "feat: migrate 4 quest NPCs to behavior trees

Tessara (Q4), Pell (Q6/Q9), Rhett (Q11), Sylara (Q11).
Behavior trees handle onGive item rejection and edge cases.
Dialogue YAMLs retained for conversation. JS renamed to .bak.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: Wire New Events (mob_die, mob_flee, player_enter)

**Files:**
- Modify: `internal/mobcommands/suicide.go`
- Modify: `internal/mobcommands/flee.go`
- Modify: `internal/usercommands/go.go`

- [ ] **Step 1: Wire mob_die event**

In `internal/mobcommands/suicide.go`, find the `onDie` JS dispatch
(around line 132, inside the `for uId := range mob.Character.PlayerDamage`
loop). The JS fires once per attacker — the behavior tree should fire
ONCE, before the loop, using the first attacker's userId.

Add BEFORE the attacker loop (before `for uId := range mob.Character.PlayerDamage`
around line 123):

```go
// Behavior tree: fire mob_die once with primary killer
if len(mob.Character.PlayerDamage) > 0 {
    // Pick first attacker as the killer for the event
    var killerUserId int
    for uId := range mob.Character.PlayerDamage {
        killerUserId = uId
        break
    }
    behaviortree.TryMobBehavior(mob.InstanceId, behaviortree.EventContext{
        EventType: "mob_die",
        UserId:    killerUserId,
        RoomId:    room.RoomId,
    })
}
```

Add import: `"github.com/GoMudEngine/GoMud/internal/behaviortree"`

- [ ] **Step 2: Wire mob_flee event**

In `internal/mobcommands/flee.go`, after the successful `Go(exitName, mob, room)`
call at line 98, add behavior tree dispatch. The mob has now moved to a
new room. Read the mob's current room after the move:

```go
// Behavior tree: notify mob it fled successfully
behaviortree.TryMobBehavior(mob.InstanceId, behaviortree.EventContext{
    EventType: "mob_flee",
    RoomId:    mob.Character.RoomId, // new room after flee
})
```

Add import: `"github.com/GoMudEngine/GoMud/internal/behaviortree"`

- [ ] **Step 3: Wire player_enter event**

In `internal/usercommands/go.go`, find where the player arrives in the
destination room. The mob hostile-reaction loop is around lines 493-531.
BEFORE that loop (or just before the `onEnter` room script at line 541),
add behavior tree dispatch for each mob in the room:

```go
// Behavior tree: notify mobs that a player entered
if !isSneaking {
    for _, mobInstId := range destRoom.GetMobs(rooms.FindAll) {
        mob := mobs.GetInstance(mobInstId)
        if mob == nil || mob.Character.IsCharmed() {
            continue
        }
        behaviortree.TryMobBehavior(mobInstId, behaviortree.EventContext{
            EventType: "player_enter",
            UserId:    user.UserId,
            RoomId:    destRoom.RoomId,
        })
    }
}
```

Add imports if not already present: `behaviortree`, `mobs`.

Check if `mobs` is already imported in `go.go` — it may be. Also check
if `isSneaking` is available at this point (search for where it's set
in the function).

- [ ] **Step 4: Verify build and commit**

```bash
go build ./...
git commit -m "feat: wire mob_die, mob_flee, player_enter events for behavior trees

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: Add New Conditions

**Files:**
- Modify: `internal/behaviortree/conditions.go`
- Modify: `internal/behaviortree/conditions_test.go`

- [ ] **Step 1: Add new conditions to registry**

In `internal/behaviortree/conditions.go`, add to the `init()` function:

```go
conditionRegistry["mob_has_buff"] = condMobHasBuff
conditionRegistry["player_has_spell"] = condPlayerHasSpell
conditionRegistry["player_has_misc_data"] = condPlayerHasMiscData
conditionRegistry["state_greater_than"] = condStateGreaterThan
conditionRegistry["multiple_enemies"] = condMultipleEnemies
```

- [ ] **Step 2: Implement condition functions**

Add to `internal/behaviortree/conditions.go`:

```go
func condMobHasBuff(params map[string]any, ctx *EvalContext) Result {
    mob := mobs.GetInstance(ctx.InstanceId)
    if mob == nil {
        return Failure
    }
    buffId := getIntParam(params, "buff_id")
    if mob.Character.HasBuff(buffId) {
        return Success
    }
    return Failure
}

func condPlayerHasSpell(params map[string]any, ctx *EvalContext) Result {
    user := users.GetByUserId(ctx.Event.UserId)
    if user == nil {
        return Failure
    }
    spell := getStringParam(params, "spell")
    if user.Character.HasSpell(spell) {
        return Success
    }
    return Failure
}

func condPlayerHasMiscData(params map[string]any, ctx *EvalContext) Result {
    user := users.GetByUserId(ctx.Event.UserId)
    if user == nil {
        return Failure
    }
    key := getStringParam(params, "key")
    value := getStringParam(params, "value")
    actual, _ := user.Character.GetMiscData(key).(string)
    if actual == value {
        return Success
    }
    return Failure
}

func condStateGreaterThan(params map[string]any, ctx *EvalContext) Result {
    key := getStringParam(params, "key")
    threshold := getIntParam(params, "value")
    if ctx.MobState.GetInt(key) > threshold {
        return Success
    }
    return Failure
}

func condMultipleEnemies(params map[string]any, ctx *EvalContext) Result {
    room := rooms.LoadRoom(ctx.RoomId)
    if room == nil {
        return Failure
    }
    // Count players + their companions in the room
    count := len(room.GetPlayers())
    // Each player's companions are also enemies from mob's perspective
    for _, uid := range room.GetPlayers() {
        user := users.GetByUserId(uid)
        if user != nil {
            count += len(user.Character.GetCharmIds())
        }
    }
    if count > 1 {
        return Success
    }
    return Failure
}
```

Check: `user.Character.GetCharmIds()` — verify this method exists. If
not, search for how companion counts are tracked (likely a method on
Character). Also verify `user.Character.GetMiscData(key)` returns `any`
(the research found it as `GetMiscData` not `GetMiscCharacterData`).

- [ ] **Step 3: Add tests for new conditions**

Add to `internal/behaviortree/conditions_test.go`:

```go
func TestCondStateGreaterThan_Above(t *testing.T) {
    state := NewBehaviorState()
    state.Set("counter", 5)
    ctx := &EvalContext{MobState: state}
    fn := LookupCondition("state_greater_than")
    result := fn(map[string]any{"key": "counter", "value": 3}, ctx)
    if result != Success {
        t.Error("expected Success when 5 > 3")
    }
}

func TestCondStateGreaterThan_Equal(t *testing.T) {
    state := NewBehaviorState()
    state.Set("counter", 3)
    ctx := &EvalContext{MobState: state}
    fn := LookupCondition("state_greater_than")
    result := fn(map[string]any{"key": "counter", "value": 3}, ctx)
    if result != Failure {
        t.Error("expected Failure when 3 == 3 (not greater)")
    }
}

func TestCondAllNewRegistered(t *testing.T) {
    for _, name := range []string{
        "mob_has_buff", "player_has_spell",
        "player_has_misc_data", "state_greater_than",
        "multiple_enemies",
    } {
        if LookupCondition(name) == nil {
            t.Errorf("condition %q not registered", name)
        }
    }
}
```

- [ ] **Step 4: Run tests and commit**

```bash
go test -v ./internal/behaviortree/... -run TestCond
go build ./...
git commit -m "feat: add 5 new behavior tree conditions

mob_has_buff, player_has_spell, player_has_misc_data,
state_greater_than, multiple_enemies.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: Add New Actions

**Files:**
- Modify: `internal/behaviortree/actions.go`

- [ ] **Step 1: Add new actions to registry**

In `internal/behaviortree/actions.go`, add to the `init()` function:

```go
actionRegistry["summon_companion"] = actSummonCompanion
actionRegistry["set_room_locked"] = actSetRoomLocked
actionRegistry["spawn_item_in_room"] = actSpawnItemInRoom
actionRegistry["add_buff"] = actAddBuff
actionRegistry["command_mob"] = actCommandMob
actionRegistry["give_item_multiple"] = actGiveItemMultiple
actionRegistry["set_misc_data"] = actSetMiscData
actionRegistry["increment_state"] = actIncrementState
actionRegistry["decrement_state"] = actDecrementState
```

- [ ] **Step 2: Implement action functions**

Add all action implementations. Key ones:

**summon_companion** — spawns a mob as a companion of the behavior tree's
mob, using the same scaling as player summoning:

```go
func actSummonCompanion(params map[string]any, ctx *EvalContext) Result {
    mob := mobs.GetInstance(ctx.InstanceId)
    if mob == nil {
        return Failure
    }
    mobId := getIntParam(params, "mob_id")
    count := getIntParam(params, "count")
    if count <= 0 {
        count = 1
    }

    room := rooms.LoadRoom(ctx.RoomId)
    if room == nil {
        return Failure
    }

    // Scale off mob's manifestation + willpower (same as player companions)
    manifestSkill := mob.Character.GetSkillLevel(skills.Manifestation)
    willpower := mob.Character.Stats.Willpower.ValueAdj
    basePool := 50 // default base pool for mob-summoned companions

    // Check if a custom base pool was specified
    if bp := getIntParam(params, "base_pool"); bp > 0 {
        basePool = bp
    }

    pool := characters.CalcCompanionStatPool(basePool, willpower, manifestSkill)

    for i := 0; i < count; i++ {
        companion := mobs.NewMobById(mobs.MobId(mobId), room.RoomId, pool)
        if companion == nil {
            continue
        }
        room.AddMob(companion.InstanceId)

        // Charm to the summoning mob (not a player)
        // Use mob's InstanceId as a pseudo-userId for charm tracking
        companion.Character.Charm(0, 99999, "")
        companion.Character.EndAggro()

        // Register as companion
        info := characters.CompanionInfo{
            MobId:      int(companion.MobId),
            InstanceId: companion.InstanceId,
            SourceType: characters.CompanionSummoned,
            Name:       companion.Character.Name,
            BaseName:   companion.Character.Name,
            AutoAssist: true,
        }
        mob.Character.AddCompanion(info)
    }
    return Success
}
```

NOTE: The companion system is designed for player-owned companions. Mob-
to-mob charm may need adjustment. Read `companion_summon.go` lines 127-178
and `Character.Charm()` to understand if `Charm(0, ...)` with userId=0
works for mob owners. If not, you may need to use a different approach —
perhaps `companion.Character.Charm(mob.InstanceId, ...)` where the mob's
instanceId is stored, or set the companions as hostile+auto-target instead.
Test this carefully.

**Other actions** (implement all):

```go
func actSetRoomLocked(params map[string]any, ctx *EvalContext) Result {
    room := rooms.LoadRoom(ctx.RoomId)
    if room == nil {
        return Failure
    }
    direction := getStringParam(params, "direction")
    locked := getStringParam(params, "locked") != "false"
    room.SetExitLock(direction, locked)
    return Success
}

func actSpawnItemInRoom(params map[string]any, ctx *EvalContext) Result {
    itemId := getIntParam(params, "item_id")
    roomId := getIntParam(params, "room_id")
    if roomId == 0 {
        roomId = ctx.RoomId
    }
    chance := getIntParam(params, "chance")
    if chance <= 0 {
        chance = 100
    }
    if util.Rand(100) >= chance {
        return Success // skipped by chance, not a failure
    }
    room := rooms.LoadRoom(roomId)
    if room == nil {
        return Failure
    }
    item := items.New(itemId)
    room.AddItem(item, false)
    return Success
}

func actAddBuff(params map[string]any, ctx *EvalContext) Result {
    mob := mobs.GetInstance(ctx.InstanceId)
    if mob == nil {
        return Failure
    }
    buffId := getIntParam(params, "buff_id")
    mob.Character.AddBuff(buffId)
    return Success
}

func actCommandMob(params map[string]any, ctx *EvalContext) Result {
    targetMobId := getIntParam(params, "mob_id")
    cmd := getStringParam(params, "cmd")
    room := rooms.LoadRoom(ctx.RoomId)
    if room == nil {
        return Failure
    }
    // Find a mob with matching template MobId in the same room
    for _, instId := range room.GetMobs(rooms.FindAll) {
        m := mobs.GetInstance(instId)
        if m != nil && int(m.MobId) == targetMobId {
            m.Command(cmd)
            return Success
        }
    }
    return Failure
}

func actGiveItemMultiple(params map[string]any, ctx *EvalContext) Result {
    user := users.GetByUserId(ctx.Event.UserId)
    if user == nil {
        return Failure
    }
    itemId := getIntParam(params, "item_id")
    count := getIntParam(params, "count")
    if count <= 0 {
        count = 1
    }
    for i := 0; i < count; i++ {
        item := items.New(itemId)
        user.Character.StoreItem(item)
    }
    return Success
}

func actSetMiscData(params map[string]any, ctx *EvalContext) Result {
    user := users.GetByUserId(ctx.Event.UserId)
    if user == nil {
        return Failure
    }
    key := getStringParam(params, "key")
    value := getStringParam(params, "value")
    user.Character.SetMiscData(key, value)
    return Success
}

func actIncrementState(params map[string]any, ctx *EvalContext) Result {
    key := getStringParam(params, "key")
    amount := getIntParam(params, "amount")
    if amount == 0 {
        amount = 1
    }
    current := ctx.MobState.GetInt(key)
    ctx.MobState.Set(key, current+amount)
    return Success
}

func actDecrementState(params map[string]any, ctx *EvalContext) Result {
    key := getStringParam(params, "key")
    amount := getIntParam(params, "amount")
    if amount == 0 {
        amount = 1
    }
    current := ctx.MobState.GetInt(key)
    ctx.MobState.Set(key, current-amount)
    return Success
}
```

Add necessary imports: `skills`, `characters`, `items` (check which are
already imported in actions.go).

- [ ] **Step 3: Verify AddBuff API**

Check exact signature of `Character.AddBuff`. It may be
`mob.Character.AddBuff(buffId, "self")` or similar — grep for `AddBuff`
in `internal/characters/` and `internal/buffs/` to find the correct
calling convention. The Phantom JS used `mob.AddBuff(9, 'self')`.

- [ ] **Step 4: Build and commit**

```bash
go build ./...
git commit -m "feat: add 9 new behavior tree actions

summon_companion, set_room_locked, spawn_item_in_room, add_buff,
command_mob, give_item_multiple, set_misc_data, increment_state,
decrement_state.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: Implement Reaction Delays

**Files:**
- Modify: `internal/behaviortree/actions.go`
- Modify: `internal/behaviortree/helpers.go`

- [ ] **Step 1: Add reaction delay helper function**

Add to `internal/behaviortree/helpers.go`:

```go
// calcReactionDelay computes perception-scaled delay in seconds.
func calcReactionDelay(mobInstanceId int) time.Duration {
    mob := mobs.GetInstance(mobInstanceId)
    if mob == nil {
        return 0
    }
    cfg := configs.GetBalanceConfig()
    base := float64(cfg.MobBTreeReactionBase)
    scale := float64(cfg.MobBTreeReactionPerceptionScale)
    minDelay := float64(cfg.MobReactionDelayMin)
    maxDelay := float64(cfg.MobReactionDelayMax)

    if scale <= 0 {
        scale = 100
    }

    perception := float64(mob.Character.Stats.Perception.ValueAdj)
    delay := base - (perception / scale)

    if delay < minDelay {
        delay = minDelay
    }
    if delay > maxDelay {
        delay = maxDelay
    }
    return time.Duration(delay * float64(time.Second))
}
```

Add imports: `"time"`, `"github.com/GoMudEngine/GoMud/internal/configs"`.

- [ ] **Step 2: Create delayed action wrapper**

Add to `internal/behaviortree/actions.go`:

```go
// delayedActions lists action names that use perception-scaled delays.
var delayedActions = map[string]bool{
    "respond":     true,
    "say":         true,
    "emote":       true,
    "attack":      true,
    "flee":        true,
    "cast":        true,
    "move":        true,
    "add_buff":    true,
    "command_mob": true,
}
```

Modify the `ActionNode.Evaluate` method to check if the action is delayed:

Read the current `ActionNode.Evaluate` in actions.go, then replace it:

```go
func (n *ActionNode) Evaluate(ctx *EvalContext) Result {
    if delayedActions[n.Name] {
        delay := calcReactionDelay(ctx.InstanceId)
        if delay > 0 {
            // Capture params and context for delayed execution
            params := n.Params
            fn := n.Fn
            mobId := ctx.InstanceId
            event := ctx.Event
            mobState := ctx.MobState
            evalCtx := &EvalContext{
                Event:      event,
                MobState:   mobState,
                MobId:      ctx.MobId,
                InstanceId: mobId,
                RoomId:     ctx.RoomId,
                MobName:    ctx.MobName,
            }
            GetEngine().QueueDelayed(delay, func() {
                fn(params, evalCtx)
            })
            return Success
        }
    }
    return n.Fn(n.Params, ctx)
}
```

- [ ] **Step 3: Build and test**

```bash
go build ./...
go test -v ./internal/behaviortree/...
```

Existing tests should still pass — the test context has no mob instance,
so `calcReactionDelay` returns 0 and actions fire instantly.

- [ ] **Step 4: Commit**

```bash
git commit -m "feat: implement perception-scaled reaction delays

Dialogue, combat, movement, buff, and cross-mob actions are delayed
by perception. Formula: base - (perception / scale), clamped to
[min, max]. Quest/item/state actions remain instant.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: Add Negative Caching

**Files:**
- Modify: `internal/behaviortree/engine.go`
- Modify: `internal/behaviortree/helpers.go`

- [ ] **Step 1: Add noTree map to Engine**

In `internal/behaviortree/engine.go`, add `noTree` field to the Engine
struct and initialize it:

```go
type Engine struct {
    mu     sync.RWMutex
    trees  map[int]Node
    noTree map[int]bool // mob IDs confirmed to have no behavior file
    queue  []DelayedAction
}
```

Update `init()`:
```go
func init() {
    globalEngine = &Engine{
        trees:  make(map[int]Node),
        noTree: make(map[int]bool),
    }
}
```

Clear `noTree` entry in `LoadTree` (in case files are added at runtime):
```go
func (e *Engine) LoadTree(mobId int, path string) error {
    node, err := LoadTreeFromFile(path)
    if err != nil {
        return err
    }
    e.mu.Lock()
    e.trees[mobId] = node
    delete(e.noTree, mobId)
    e.mu.Unlock()
    return nil
}
```

Add a `HasNoTree` method:
```go
func (e *Engine) HasNoTree(mobId int) bool {
    e.mu.RLock()
    defer e.mu.RUnlock()
    return e.noTree[mobId]
}

func (e *Engine) SetNoTree(mobId int) {
    e.mu.Lock()
    e.noTree[mobId] = true
    e.mu.Unlock()
}
```

- [ ] **Step 2: Use noTree in TryMobBehavior**

In `internal/behaviortree/helpers.go`, modify `TryMobBehavior` to check
negative cache before `os.Stat`:

Replace the lazy-load block:
```go
tree := GetEngine().GetTree(mobId)
if tree == nil {
    if GetEngine().HasNoTree(mobId) {
        return false
    }
    path := GetBehaviorPath(mobId, mob.Zone, mob.Character.Name)
    if _, err := os.Stat(path); err != nil {
        GetEngine().SetNoTree(mobId)
        return false
    }
    // ... rest of loading
}
```

- [ ] **Step 3: Build and commit**

```bash
go build ./...
git commit -m "feat: add negative caching for mobs without behavior trees

Avoids repeated os.Stat calls on every idle tick for mobs that
don't have behavior tree YAML files.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 7: Training Dummy + Cleanup

**Files:**
- Create: `_datafiles/world/dogmud/behaviors/tutorial/58-training_dummy.yaml`
- Delete: `_datafiles/world/dogmud/mobs/tutorial/scripts/58-training_dummy.js`
- Delete: `_datafiles/world/dogmud/mobs/startland/3-apprentice_mage.js`

- [ ] **Step 1: Read tutorial quest structure**

Read the tutorial quest YAML to find what token the dummy death should
grant. Search `_datafiles/world/dogmud/quests/` for tutorial-related
quests. Also read the training dummy JS to confirm what it currently does.

- [ ] **Step 2: Create training dummy behavior tree**

Create `_datafiles/world/dogmud/behaviors/tutorial/58-training_dummy.yaml`.

Simple `mob_die` event handler:
```yaml
tree:
  type: sequence
  event: mob_die
  children:
    - type: action
      do: grant_quest_to_user
      quest: "<tutorial-token>"  # fill in from Step 1
```

The `grant_quest_to_user` is functionally the same as `grant_quest` —
both use `ctx.Event.UserId`. Register it as an alias in actions.go init:
```go
actionRegistry["grant_quest_to_user"] = actGrantQuest
```

(Where `actGrantQuest` is the existing grant_quest function.)

- [ ] **Step 3: Delete JS files and commit**

```bash
rm _datafiles/world/dogmud/mobs/tutorial/scripts/58-training_dummy.js
rm _datafiles/world/dogmud/mobs/startland/3-apprentice_mage.js
git add -A
git commit -m "feat: migrate training dummy to behavior tree, delete stub

Dummy death grants tutorial token directly. Teacher mob reacts
via dialogue. Apprentice mage empty stub deleted.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 8: Old Edrin — Multi-Phase Caster Boss

**Files:**
- Create: `_datafiles/world/dogmud/behaviors/marches_spur_road/275-old_edrin.yaml`
- Modify: `_datafiles/world/dogmud/mobs/marches_spur_road/275-old_edrin.yaml`
- Rename: `_datafiles/world/dogmud/mobs/marches_spur_road/scripts/275-old_edrin.js` → `.js.bak`

- [ ] **Step 1: Read Edrin's current mob YAML and JS**

Read both files to understand his current stats, equipment, and behavior.
Also read the companion summoning code in `internal/hooks/companion_summon.go`
to understand how `summon_companion` should work for a mob summoner.

- [ ] **Step 2: Update Edrin's mob YAML**

Add to `_datafiles/world/dogmud/mobs/marches_spur_road/275-old_edrin.yaml`:

```yaml
# Add to skills section:
skills:
  manifestation: 18
  spellcasting: 15

# Add to spellbook:
spellbook:
  conviction-ward: 10
  chrysalis-cocoon: 8

# Ensure archetype is casting:
archetype: casting
```

Read the existing YAML first to merge cleanly with existing fields.

- [ ] **Step 3: Create Edrin behavior tree**

Create `_datafiles/world/dogmud/behaviors/marches_spur_road/275-old_edrin.yaml`.

Four-phase tree:

```yaml
tree:
  type: selector
  children:

    # Phase 0: Preemptive — self-buff + summon on player entry
    - type: sequence
      event: player_enter
      children:
        - type: decorator
          mod: cooldown
          rounds: 100  # don't re-summon if player leaves and returns
          child:
            type: sequence
            children:
              - type: action
                do: cast
                spell: conviction-ward
              - type: action
                do: summon_companion
                mob_id: 313  # fire elemental
                base_pool: 60
              - type: action
                do: summon_companion
                mob_id: 311  # earth elemental
                base_pool: 60
              - type: action
                do: summon_companion
                mob_id: 310  # water elemental
                base_pool: 60

    # Phase 1: Reveal — first time hurt
    - type: sequence
      event: mob_hurt
      children:
        - type: decorator
          mod: invert
          child:
            type: condition
            check: state_equals
            key: revealed
            value: "true"
        - type: action
          do: set_state
          key: revealed
          value: "true"
        - type: action
          do: emote
          text: >
            straightens slowly, the stoop vanishing from his spine.
            His milky eyes clear to sharp, pale blue. The tremor in
            his hands stops.
        - type: action
          do: say
          text: You should not have done that.

    # Phase 2: Tactical combat — AoE vs single target
    - type: selector
      event: mob_idle
      children:
        # Only when revealed and in combat
        - type: sequence
          children:
            - type: condition
              check: state_equals
              key: revealed
              value: "true"
            - type: condition
              check: mob_in_combat
            - type: selector
              children:
                # Re-buff if ward dropped
                - type: sequence
                  children:
                    - type: decorator
                      mod: invert
                      child:
                        type: condition
                        check: mob_has_buff
                        buff_id: 36  # conviction-ward buff ID — verify
                    - type: action
                      do: cast
                      spell: conviction-ward
                # AoE if multiple enemies
                - type: sequence
                  children:
                    - type: condition
                      check: multiple_enemies
                    - type: action
                      do: cast
                      spell: cleansing-wave  # or whatever AoE he knows
                # Single target spell
                - type: action
                  do: cast
                  spell: conviction-spike

    # Phase 3: Death — unlock room, spawn loot
    - type: sequence
      event: mob_die
      children:
        - type: action
          do: set_room_locked
          direction: west
          locked: "false"
        - type: action
          do: spawn_item_in_room
          item_id: 40010
          room_id: 4037
          chance: 75
        - type: action
          do: spawn_item_in_room
          item_id: 40027
          room_id: 4037
          chance: 75
        - type: action
          do: spawn_item_in_room
          item_id: 40004
          room_id: 4037
          chance: 75
        - type: action
          do: spawn_item_in_room
          item_id: 40009
          room_id: 4037
          chance: 75
```

NOTE: Verify buff IDs (conviction-ward buff ID), spell names (AoE spell),
and elemental mob IDs (310, 311, 313) from the existing JS and game data.

- [ ] **Step 4: Rename JS and commit**

```bash
mv _datafiles/world/dogmud/mobs/marches_spur_road/scripts/275-old_edrin.js \
   _datafiles/world/dogmud/mobs/marches_spur_road/scripts/275-old_edrin.js.bak
go build ./...
git commit -m "feat: migrate Old Edrin to behavior tree with upgraded AI

Multi-phase caster boss: self-buffs on player entry, summons 3
scaled elemental companions, reveals on first hit, uses AoE vs
single-target spells tactically, unlocks back room on death.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 9: Chrysalis Phantom — Hit-and-Run Assassin

**Files:**
- Create: `_datafiles/world/dogmud/behaviors/thornwall_city/272-chrysalis_phantom.yaml`
- Modify: `_datafiles/world/dogmud/mobs/thornwall_city/272-chrysalis_phantom.yaml`
- Rename: `_datafiles/world/dogmud/mobs/thornwall_city/scripts/272-chrysalis_phantom.js` → `.js.bak`

- [ ] **Step 1: Read Phantom's current mob YAML**

Check current stats, buffs, and skills. Also check what the hidden buff
ID is (should be 9) and what the sneak/hide cooldown mechanics are.

- [ ] **Step 2: Update Phantom's mob YAML**

Add search skill for tracking:
```yaml
skills:
  search: 22
```

Verify `buffids: [9]` is set (hidden on spawn). Ensure high perception
and dexterity for fast reactions and flee success.

- [ ] **Step 3: Create Phantom behavior tree**

Create `_datafiles/world/dogmud/behaviors/thornwall_city/272-chrysalis_phantom.yaml`.

The full surprise-strike loop:

```yaml
tree:
  type: selector
  children:

    # Flee event: start recovery countdown
    - type: sequence
      event: mob_flee
      children:
        - type: action
          do: set_state
          key: recovery
          value: "3"

    # Hurt event: remember target name
    - type: sequence
      event: mob_hurt
      children:
        - type: action
          do: set_state
          key: engaged
          value: "true"

    # Idle: main behavior loop
    - type: selector
      event: mob_idle
      children:

        # Recovery countdown (after fleeing)
        - type: sequence
          children:
            - type: condition
              check: state_greater_than
              key: recovery
              value: 0
            - type: selector
              children:
                # Recovery tick 2: re-sneak and re-hide
                - type: sequence
                  children:
                    - type: condition
                      check: state_equals
                      key: recovery
                      value: "2"
                    - type: action
                      do: command
                      cmd: sneak
                    - type: action
                      do: add_buff
                      buff_id: 9
                    - type: action
                      do: decrement_state
                      key: recovery
                # Recovery tick 1: wait for cooldown
                - type: sequence
                  children:
                    - type: condition
                      check: state_equals
                      key: recovery
                      value: "1"
                    - type: action
                      do: decrement_state
                      key: recovery
                # Recovery tick 0 or below: track target
                - type: sequence
                  children:
                    - type: action
                      do: decrement_state
                      key: recovery
                    - type: action
                      do: command
                      cmd: track  # uses search skill to find target

        # Suppress idle emotes while hidden
        - type: sequence
          children:
            - type: condition
              check: mob_has_buff
              buff_id: 9
            - type: condition
              check: players_in_room
            # Hidden + players present = attack (surprise strike)
            - type: action
              do: attack

        # Engaged but not fighting — re-engage
        - type: sequence
          children:
            - type: condition
              check: state_equals
              key: engaged
              value: "true"
            - type: condition
              check: players_in_room
            - type: decorator
              mod: invert
              child:
                type: condition
                check: mob_in_combat
            - type: action
              do: attack

        # In combat after striking — flee immediately
        - type: sequence
          children:
            - type: condition
              check: mob_in_combat
            - type: action
              do: command
              cmd: flee
```

NOTE: The `track` command behavior — verify that `mob.Command("track")`
works for mobs and uses the search skill. If `track` needs a target name,
the tree needs to store the target name and use it:
`do: command, cmd: "track <targetname>"`. This requires the target name
to be stored in state on first engagement. Adjust the `mob_hurt` handler
to store the player's name if needed.

- [ ] **Step 4: Rename JS and commit**

```bash
mv _datafiles/world/dogmud/mobs/thornwall_city/scripts/272-chrysalis_phantom.js \
   _datafiles/world/dogmud/mobs/thornwall_city/scripts/272-chrysalis_phantom.js.bak
go build ./...
git commit -m "feat: migrate Chrysalis Phantom to behavior tree with upgraded AI

Hit-and-run assassin: surprise strike from hidden, immediate flee,
re-sneak during recovery, track target via search skill, repeat.
Suppresses idle emotes while hidden.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 10: Upgrade Barmaid Dal — Patron Heckling

**Files:**
- Modify: `_datafiles/world/dogmud/behaviors/thornwall_city/117-barmaid_dal.yaml`

- [ ] **Step 1: Read current Dal behavior tree**

Read `_datafiles/world/dogmud/behaviors/thornwall_city/117-barmaid_dal.yaml`
to understand the current structure.

- [ ] **Step 2: Add patron heckling to back-room arrival**

In the idle event handler, after Dal moves to the back room (south), add
a 40% chance interaction sequence using `command_mob`:

The interaction should fire when Dal is NOT at home (she's in the back
room). Add a new branch after the "move to back room" cooldown:

```yaml
# At back room: chance of patron heckling
- type: sequence
  children:
    - type: decorator
      mod: invert
      child:
        type: condition
        check: mob_at_home
    - type: decorator
      mod: cooldown
      rounds: 20
      child:
        type: decorator
          mod: random
          percent: 40
          child:
            type: selector
            children:
              # Interaction 1
              - type: decorator
                mod: random
                percent: 25
                child:
                  type: sequence
                  children:
                    - type: action
                      do: command_mob
                      mob_id: 114
                      cmd: "emote nudges his cup forward with one finger, grinning."
                    - type: action
                      do: emote
                      text: "smacks the cup back without looking. \"Ask nicer.\""
              # Interaction 2
              - type: decorator
                mod: random
                percent: 33
                child:
                  type: sequence
                  children:
                    - type: action
                      do: command_mob
                      mob_id: 115
                      cmd: "say Dal, you get prettier every year."
                    - type: action
                      do: emote
                      text: >
                        does not look up from the tray. "And you get
                        older. Funny how that works."
              # Interaction 3
              - type: decorator
                mod: random
                percent: 50
                child:
                  type: sequence
                  children:
                    - type: action
                      do: command_mob
                      mob_id: 116
                      cmd: "emote reaches for the bread and accidentally brushes her arm."
                    - type: action
                      do: emote
                      text: "pulls her arm back and gives him a look that could curdle milk."
              # Interaction 4 (fallback)
              - type: sequence
                children:
                  - type: action
                    do: command_mob
                    mob_id: 114
                    cmd: "say How about a smile with that ale, Dal?"
                  - type: action
                    do: emote
                    text: "sets the cup down hard enough to slosh. \"How about a tip?\""
```

Watch YAML indentation carefully. All `command_mob` and `emote` actions
will be perception-delayed, creating natural back-and-forth pacing.

- [ ] **Step 3: Commit**

```bash
git commit -m "feat: add patron heckling to Dal's behavior tree

40% chance of NPC-NPC interaction when Dal arrives in the back
corner. 4 heckle/response pairs with perception-scaled delays.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 11: Documentation — Behavior Tree Context

**Files:**
- Create: `internal/behaviortree/context.md`
- Modify: `docs/schemas/behavior.md` (update with new features)

- [ ] **Step 1: Create context.md**

Create `internal/behaviortree/context.md` with full package documentation:

- Package overview and architecture
- All condition types (original 15 + 5 new) with params
- All action types (original 17 + 9 new) with params
- All decorator types (5) with params
- Event types: `player_ask`, `player_give`, `mob_idle`, `mob_hurt`,
  `mob_die`, `mob_flee`, `player_enter`
- Instant vs. delayed action table
- Reaction delay formula
- YAML format guide with examples
- BehaviorState patterns (counters, flags, timers)
- File path convention: `_datafiles/world/dogmud/behaviors/{zone}/{mobId}-{name}.yaml`

- [ ] **Step 2: Update schema doc**

Update `docs/schemas/behavior.md` with the new conditions, actions, events,
and reaction delay information from Phase 4b.

- [ ] **Step 3: Commit**

```bash
git commit -m "docs: update behavior tree docs with Phase 4b features

Full context.md for package, updated schema reference with new
events, conditions, actions, reaction delay system.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```
