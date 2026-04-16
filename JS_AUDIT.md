# JavaScript Scripting Audit — DOGMud Codebase

**Date:** 2026-04-11
**Scope:** All `.js` files under `_datafiles/` (excluding `html/` frontend assets)
**Purpose:** Assess what JS scripting does, identify pain points, and evaluate
the cost/benefit of replacing it with native Go.

---

## 1. Inventory Summary

| Metric | Value |
|--------|-------|
| Total JS files | 255 |
| Total lines of code | 10,565 |
| Average lines per file | 41 |
| Largest file | 212 lines (sample quest mob) |
| Go scripting bridge | 3,830 lines (14 files in `internal/scripting/`) |

### Files by World

| World | Files | Notes |
|-------|-------|-------|
| `world/dogmud/` | 128 | Our custom content — primary concern |
| `world/default/` | 100 | Upstream GoMud content |
| `world/empty/` | 19 | Minimal template world |
| `sample-scripts/` | 8 | Developer reference templates |

### Files by Category

| Category | Files | Lines (est.) | Complexity Range |
|----------|-------|-------------|-----------------|
| Spells | ~82 | ~3,500 | Trivial → High |
| Buffs | ~68 | ~1,500 | Trivial → Medium |
| Room scripts | ~65 | ~2,100 | Trivial → High |
| Mob scripts | ~26 | ~2,400 | Low → Very High |
| Item scripts | ~12 | ~300 | Trivial → Low |
| Samples | 8 | ~450 | Reference only |

---

## 2. What the JS Does

### 2.1 Spells (82 files)

Spells use a three-phase hook system: `onCast` → `onWait` → `onMagic`.

**Flavor-only spells (35+ files, 9-16 lines each):** The majority of spell
scripts exist solely to provide cast/channel/resolution flavor text. The
actual mechanical effects (damage, healing, buff application, hit/miss)
are resolved entirely in the Go backend's `spell_resolution.go`. These
scripts are boilerplate — three functions, each containing one
`SendUserMessage` and one `SendRoomMessage` call.

Example pattern (repeated 35+ times with different text):
```js
function onCast(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), "You channel energy...");
    SendRoomMessage(sourceActor.GetRoomId(), "Player channels energy...",
                    sourceActor.UserId());
}
function onWait(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), "Energy gathers...");
}
function onMagic(sourceActor, targetActor) {
    // Empty — Go handles the actual effect
}
```

**Companion summoning spells (14 files, 46-132 lines):** The most complex
spell category. These handle companion cap validation, component item
consumption (backpack iteration + TakeItem), stat pool scaling
(`BASE_STAT_POOL * (1 + charisma/divisor + manifestation*0.02)`), mob
spawning via `SpawnMobScaled`, charm registration, and companion tracking.
The six necromancy raise spells (`raise-skeleton.js` through
`raise-vampire.js`) are near-identical — they differ only in 4 constants
(mob ID, base pool, minimum corpse pool, flavor text) but each is 97-99
lines of duplicated logic.

**Teleportation spells (2 files):** `fold-anchor.js` (29 lines) writes a
room ID to character persistent data; `fold-recall.js` (70 lines) reads
it back, validates the target room allows recall, clears combat, and
teleports.

**Charm spell (1 file, 132 lines):** The most complex single spell. Performs
opposed roll calculation with stat scaling, aggro state penalties,
conditional messaging, and companion registration.

### 2.2 Buffs (68 files)

Buffs use lifecycle hooks: `onStart` → `onTrigger` (per tick) → `onEnd`.

**Flavor-only buffs (~40 files, 9-12 lines):** Send a message on start
and end. `onTrigger` is empty. All mechanical effects (stat mods, flags,
duration) live in the buff YAML definition and Go backend.

**Healing/damage-over-time buffs (~15 files, 14-25 lines):** Calculate a
percentage of max HP/SP/CP and apply it per tick. Pattern:
`Math.floor(actor.GetHealthMax() * 0.08)` with a minimum of 1.
Damage DoTs add a random variance component.

**Utility buffs (~5 files):** Minor antidote removes poison buffs.
Death recovery rolls dice for resource restoration. Tutorial buffs
gate command access.

### 2.3 Mob Scripts (26 files)

The most sophisticated category. These define NPC behavior and are the
scripts that benefit most from JS flexibility.

**Quest NPCs (8-10 files, 100-188 lines):** Multi-stage dialogue trees
with keyword matching (`UtilFindMatchIn`), quest state checks
(`user.HasQuest`), item/gold transactions, party member handling, and
timed command sequences (`mob.Command("say ...", delaySeconds)`).

**AI behaviors (4-6 files, 42-154 lines):** Patrol routes via `onIdle`
with round-modulo timing, state machines using `SetTempData`/`GetTempData`,
hit-and-run boss tactics with sneak/hide buff management, and timed
aggro systems (bandit leader negotiation countdown).

**NPC routines (2-3 files, 109-121 lines):** Barmaid with room cycling
schedules, economy NPCs that create instanced zones, social interactions
between NPCs.

**Companion AI (1 file, 188 lines):** Player guide with hint system,
lost-player tracking, and portal creation.

### 2.4 Room Scripts (65 files)

**Tutorial rooms (8-10 files, 150-175 lines):** Command restriction and
progressive unlocking. The tutorial is the single most complex room
scripting use case, with whitelisted commands, NPC spawn triggers, and
multi-step progression.

**Map rooms (1 file, 47 lines):** Procedural map generation with caching.

**Quest trigger rooms (15-20 files, 11-50 lines):** `onEnter` hooks that
check quest state and spawn NPCs, add temporary exits, or send messages.

**Stub rooms (~30 files, 5-11 lines):** Empty or near-empty scripts that
exist only because the room YAML references a script file.

### 2.5 Item Scripts (12 files)

Mostly simple `onCommand_use` handlers: grant buffs, spawn items (newbie
kit), train skills (recipe page), toggle light sources. The sleeping bag
and newbie kit are the most complex at 16-34 lines.

---

## 3. The Scripting Bridge (Go Side)

### 3.1 Runtime

- **Engine:** Goja (`github.com/dop251/goja`) — pure Go ES5.1+ runtime
- **VM model:** One VM per unique script definition, cached in type-specific
  maps (`roomVMCache`, `mobVMCache`, `buffVMCache`, `spellVMCache`,
  `itemVMCache`)
- **Timeouts:** 1000ms for script load, 50ms for hook execution
- **Error handling:** Exceptions caught and logged, never crash the server
- **Function caching:** `VMWrapper` caches `goja.Callable` lookups

### 3.2 Exposed API Surface

The Go bridge exposes **80+ methods** across these namespaces:

| Namespace | Methods | Go Source |
|-----------|---------|-----------|
| Actor (user/mob) | ~60+ | `actor_func.go` (1,047 lines) |
| Room | ~20+ | `room_func.go` (619 lines) |
| Utilities | ~15 | `util_func.go` (225 lines) |
| Messaging | 4 | `messaging_func.go` (67 lines) |
| Items | ~8 | `item_func.go` (129 lines) |
| Console | 5 | `console.go` (35 lines) |

### 3.3 Bridge Overhead

The `internal/scripting/` package totals **3,830 lines of Go** — all of
which exists solely to marshal data between Go and the JS runtime. This
is 36% the size of the JS it serves. Key files:

- `actor_func.go` (1,047 lines) — the largest, wrapping every character
  method for JS access
- `room_func.go` (619 lines) — room object wrapper
- `mob.go` (343 lines) / `room.go` (388 lines) — VM lifecycle management
- Per-type files (`buff.go`, `spell.go`, `item.go`) — 200-250 lines each

---

## 4. Pain Points

### 4.1 Massive Duplication

**The #1 problem.** The codebase contains enormous amounts of copy-paste:

- **35+ flavor-only spell scripts** are functionally identical (same 3
  functions, same API calls, different string literals)
- **6 necromancy raise spells** share 95% of their logic, differing only
  in mob ID, base stat pool, minimum corpse pool, and flavor text
- **5 elemental conjure spells** are identical except for mob ID, base
  stat pool, and flavor text
- **~40 flavor-only buff scripts** follow the same 9-12 line template
- **~30 stub room scripts** are effectively empty files

This duplication means:
- Bug fixes must be applied N times (e.g., if companion cap logic changes)
- New patterns can't be introduced without touching dozens of files
- The file count is inflated — 255 files sounds large but ~120 are trivial

### 4.2 No Code Sharing Between Scripts

Goja VMs are isolated. There is no `require()` or `import`. Every script
is self-contained. This means:
- The companion summoning pattern (cap check → component consume →
  scale stats → spawn → charm → register) is copy-pasted across 14 files
- Common formulas (stat scaling, percentage calculations) are duplicated
- Helper functions cannot be shared across scripts

### 4.3 Weak Typing and Silent Failures

JavaScript's loose typing means:
- Calling a non-existent method returns `undefined` instead of a compile error
- Passing wrong argument types silently produces wrong results
- No IDE support for the custom API (no TypeScript definitions exist)

### 4.4 Two-Language Maintenance Burden

Every new game mechanic requires changes in two places:
1. Go backend (the actual mechanic)
2. JS scripts (the flavor text / validation / orchestration)

Plus the bridge layer (`actor_func.go` etc.) must be updated any time a
new Go method needs JS access. This triple-touch pattern slows development.

### 4.5 Testing Gap

JS scripts have no test coverage. The Go scripting bridge has a single
benchmark test (`scripting_test.go`, 75 lines). There is no way to unit
test a spell or buff script in isolation.

### 4.6 Performance (Minor)

Goja is fast enough for this use case — 50ms timeout is generous for
the work these scripts do. But each VM carries memory overhead, and
the marshaling between Go and JS adds latency compared to native Go
function calls. For a MUD with dozens of concurrent users, this is not
a bottleneck. At hundreds of users it could become measurable.

---

## 5. What Replacing JS with Go Would Look Like

### 5.1 Architecture Options

**Option A: Go Hook Functions (Compiled)**

Replace each JS script with a Go function registered by name. Spells,
buffs, mobs, rooms, and items would reference a Go function name in
their YAML instead of a `.js` file path.

```yaml
# spell YAML
script: "go:summon_hive_swarm"  # instead of: script: summon-hive-swarm.js
```

```go
// internal/scripts/spells/summon_hive_swarm.go
func SummonHiveSwarm(phase string, source, target *character.Character,
                      room *rooms.Room) bool {
    switch phase {
    case "cast":
        source.SendText("You reach into the ether...")
        room.SendText("Player reaches into the ether...", source.UserId())
    case "magic":
        if source.GetCompanionCount() >= source.GetMaxCompanionCount() {
            source.SendText("You have too many companions.")
            return false
        }
        // ... rest of logic, calling Go APIs directly
    }
    return true
}
```

**Pros:** Full type safety, IDE support, testable, no bridge overhead,
access to all Go internals.
**Cons:** Requires recompilation for any text change. Loses hot-reload.

**Option B: Data-Driven Flavor + Go Logic**

Move flavor text into the YAML definitions (new fields like `cast_text`,
`wait_text`, `magic_text`) and keep only non-trivial logic in Go.

```yaml
# spell YAML
cast_user_text: "You reach into the ether..."
cast_room_text: "{name} reaches into the ether..."
wait_user_text: "Energy gathers around your hands..."
# No script needed — Go handles the rest
```

For spells/buffs that need logic (summoning, teleportation, charm),
use Go hook functions (Option A).

**Pros:** Eliminates 70%+ of scripts entirely. Text is editable without
recompilation. Logic is in Go with full type safety.
**Cons:** Requires YAML schema changes. Upstream divergence.

**Option C: Hybrid — Keep JS for Content, Go for Mechanics**

Keep JS for mob scripts (quest NPCs, AI behaviors) where the flexibility
and hot-reload matter most. Move spells, buffs, and simple room/item
scripts to Go or data-driven YAML.

**Pros:** Best of both worlds — type safety where it matters, flexibility
where it's needed.
**Cons:** Two systems to maintain (though much smaller JS surface).

### 5.2 Migration Effort Estimates

| Category | Files | Approach | Effort |
|----------|-------|----------|--------|
| Flavor-only spells | ~35 | YAML text fields | Low — mechanical migration |
| Flavor-only buffs | ~40 | YAML text fields | Low — mechanical migration |
| Stub room scripts | ~30 | Delete (no-ops) | Trivial |
| Healing/DoT buffs | ~15 | Go functions or YAML config | Low-Medium |
| Companion spells | 14 | 1 Go function + config table | Medium — consolidates 14→1 |
| Charm spell | 1 | Go function | Medium |
| Teleport spells | 2 | Go functions | Low |
| Item scripts | 12 | Go functions or YAML fields | Low |
| Quest mob scripts | ~10 | Keep as JS (or new Go DSL) | High if migrated |
| AI mob scripts | ~6 | Keep as JS (or new Go DSL) | High if migrated |
| Tutorial rooms | ~8 | Keep as JS or Go functions | Medium-High |
| Quest trigger rooms | ~15 | Go functions or YAML config | Low-Medium |

### 5.3 Bridge Cleanup

If all scripts moved to Go, the entire `internal/scripting/` package
(3,830 lines) could be deleted. If we keep JS for mob scripts only,
we could slim it significantly — remove spell/buff/item VM management
and most of `actor_func.go`.

---

## 6. Cost/Benefit Analysis

### 6.1 High-Value Targets (Do First)

**Flavor-only spells and buffs (~75 files):**
- **Benefit:** Eliminates 75 trivially duplicated files. Text moves to
  YAML where it belongs. No more "write 3 JS functions to say 3 strings."
- **Cost:** Requires adding `cast_text`/`wait_text`/`magic_text` fields
  to spell YAML schema and corresponding Go logic to send them. Similar
  for buff `start_text`/`trigger_text`/`end_text`.
- **Risk:** Low. These scripts have zero logic.
- **Upstream impact:** Moderate — changes spell/buff loading in Go engine.

**Stub room scripts (~30 files):**
- **Benefit:** Delete 30 empty files. Cleaner directory tree.
- **Cost:** Remove script references from room YAMLs.
- **Risk:** Near zero.

**Companion summoning consolidation (14 files → 1 Go function + table):**
- **Benefit:** One place to fix companion logic bugs. Currently a bug in
  the cap-check, stat-scaling, or charm-registration pattern must be
  fixed in 14 separate files. The necromancy spells alone are 600 lines
  of near-identical code.
- **Cost:** Write one parameterized Go function (~80 lines) and a config
  table mapping spell→mob ID, base pool, component item, etc.
- **Risk:** Low-Medium. Logic is well-understood and consistent.

### 6.2 Medium-Value Targets

**Healing/DoT buffs (15 files):**
- **Benefit:** Replace percentage-based tick logic with YAML config fields
  (`heal_pct: 0.08`, `damage_pct: 0.05`). The Go buff tick handler
  already exists — just needs to read these fields.
- **Cost:** Low. Add fields to buff schema, update tick handler.

**Quest trigger room scripts (15-20 files):**
- **Benefit:** Many of these are just "onEnter: check quest, spawn mob."
  Could be expressed as YAML conditions.
- **Cost:** Medium. Requires a declarative room-event system in Go.

**Item scripts (12 files):**
- **Benefit:** Most are trivial (use → give buff / spawn item). Could be
  YAML fields (`on_use_buff: 15`, `on_use_spawn_item: [4, 6, 100]`).
- **Cost:** Low. Straightforward schema additions.

### 6.3 Low-Value Targets (Keep as JS or Defer)

**Quest NPCs / AI mob scripts (~16 files):**
- **Benefit of migration:** Type safety, testability.
- **Cost of migration:** Very high. These scripts use the full API surface
  — keyword matching, quest state, inventory manipulation, timed command
  sequences, state machines. Rewriting them in Go means either a complex
  declarative DSL or verbose Go functions.
- **Recommendation:** Keep these in JS. They are the scripts that actually
  benefit from the scripting layer's flexibility. When we add new NPCs,
  JS lets us iterate quickly without recompilation.

**Tutorial rooms (~8 files):**
- **Benefit:** Type safety.
- **Cost:** High. Tutorial logic is complex and tightly coupled to room
  state. Would need significant refactoring.
- **Recommendation:** Defer. These are upstream content and rarely change.

### 6.4 Summary Scorecard

| Target | Files Eliminated | Lines Eliminated | Effort | Priority |
|--------|-----------------|-----------------|--------|----------|
| Flavor spell text → YAML | ~35 | ~450 | Low | **High** |
| Flavor buff text → YAML | ~40 | ~400 | Low | **High** |
| Delete stub rooms | ~30 | ~200 | Trivial | **High** |
| Consolidate companion spells | 14→1 | ~900 | Medium | **High** |
| Healing/DoT → YAML config | ~15 | ~300 | Low | **Medium** |
| Item scripts → YAML | ~12 | ~300 | Low | **Medium** |
| Quest room triggers → YAML | ~15 | ~400 | Medium | **Medium** |
| Quest NPCs | 0 (keep JS) | 0 | — | **Skip** |
| AI mob scripts | 0 (keep JS) | 0 | — | **Skip** |
| Tutorial rooms | 0 (keep JS) | 0 | — | **Skip** |

**Net result of high-priority items:** ~120 files eliminated, ~1,950
lines removed, replaced by ~200 lines of Go + YAML schema additions.

**Net result of all items except skipped:** ~160 files eliminated,
~3,050 lines removed. ~95 JS files remain (mob scripts, tutorial,
complex rooms).

---

## 7. Upstream Considerations

The `world/default/` directory (100 JS files) is upstream GoMud content.
We have two options:

1. **Fork the engine scripting layer** to support YAML text fields
   alongside JS. This means our Go engine diverges from upstream, making
   future merges harder. But we already diverge significantly in combat,
   stats, and progression.

2. **Only migrate `world/dogmud/` scripts** (128 files). Leave upstream
   `default/` and `empty/` untouched. This is safer but means we maintain
   two patterns side by side.

**Recommendation:** Option 1. We're already deeply forked. The scripting
changes would be additive (JS still works, YAML text fields are optional),
so upstream merges remain possible via the existing script path.

---

## 8. The Bridge Question

If we move to Option B/C (data-driven flavor + Go logic for complex
scripts, JS only for mob AI), the scripting bridge shrinks dramatically:

| Current Bridge | After Migration |
|---------------|-----------------|
| `actor_func.go` — 1,047 lines | Keep (mob scripts still need it) |
| `room_func.go` — 619 lines | Keep (mob/room scripts need it) |
| `spell.go` — 217 lines | **Delete** (no more spell JS) |
| `buff.go` — 237 lines | **Delete** (no more buff JS) |
| `item.go` — 244 lines | **Delete** (no more item JS) |
| `item_func.go` — 129 lines | **Delete** |
| `spell_func.go` — ? lines | **Delete** |

Estimated bridge reduction: ~800-1,000 lines of Go removed from the
bridge, plus the 3,050 lines of JS. Total cleanup: ~4,000 lines.

---

## 9. Risks and Open Questions

1. **Hot-reload loss for spells/buffs:** Moving flavor text to YAML means
   server restart to change text. Currently JS can be edited and the VM
   cache cleared. How often do we actually hot-reload spell text? (Answer:
   almost never — we restart after every content change anyway.)

2. **Upstream merge friction:** Any changes to the spell/buff loading
   pipeline in upstream GoMud will conflict with our YAML text fields.
   Manageable but adds merge work.

3. **Content generation impact:** Our `/new-mob`, `/new-room` etc.
   slash commands currently generate JS files alongside YAML. If we
   move to YAML text fields, the generation templates need updating.

4. **Mob script complexity ceiling:** If mob AI gets more complex (which
   it will — companion AI, reactive tactics, etc.), JS remains the right
   tool. But we should consider whether a more structured approach
   (behavior trees? state machine DSL?) would serve us better than
   ad-hoc JS.

5. **Sample scripts:** The `sample-scripts/` directory is developer
   documentation for the JS API. If we reduce the JS surface, these
   need updating to reflect what still uses JS vs. what's data-driven.

---

## 10. Recommended Approach

**Phase 1 — Quick Wins (Low effort, high impact)**
- Add `cast_text`, `wait_text`, `magic_text` fields to spell YAML schema
- Add `start_text`, `trigger_text`, `end_text` fields to buff YAML schema
- Go spell/buff resolution sends these texts when present, skips JS
- Migrate ~75 flavor-only scripts to YAML text fields
- Delete ~30 stub room scripts and remove stale script references

**Phase 2 — Consolidation (Medium effort, high impact)**
- Write one parameterized `CompanionSummon()` Go function
- Config table maps spell → mob ID, base pool, component, scaling params
- Replace 14 companion spell scripts with YAML config + Go function
- Write parameterized `HealOverTime()` / `DamageOverTime()` for buff ticks
- Replace 15 healing/DoT buff scripts with YAML config fields

**Phase 3 — Cleanup (Low effort, medium impact)**
- Migrate simple item scripts to YAML `on_use` fields
- Migrate quest-trigger room scripts to declarative YAML events
- Remove unused bridge code (`spell.go`, `buff.go`, `item.go`)
- Update content generation slash commands

**Phase 4 — Evaluate (Research)**
- Assess whether mob scripts need a more structured approach
- Consider behavior trees or state machine DSL for AI
- Evaluate whether remaining JS scripts warrant keeping Goja at all
  (could a simple Go plugin system replace it?)

---

## Appendix A: Complete File Listing by Category

### Spell Scripts (dogmud) — 66 files
```
_datafiles/world/dogmud/spells/
├── Flavor-only (35): blood-boil, chrysalis-cocoon, chrysalis-glow,
│   chrysalis-haste, chrysalis-regeneration, cleansing-wave,
│   communion-of-flesh, conviction-armor, conviction-barrage,
│   conviction-spike, conviction-surge, conviction-ward,
│   empathic-bond, empathic-shroud, hemorrhagic-burst,
│   hemorrhagic-wave, iron-will, kinetic-hurl, kinetic-shove,
│   mass-mend, mend-all, mend-wounds, mind-fog, mind-spike,
│   mutation-catalyst, nerve-disruption, neural-stun, neural-toxin,
│   psychic-anchor, pyretic-surge, sensory-overload, sensory-veil,
│   skill-attunement, sparks, synaptic-overload, veil-rend,
│   veil-sight, vital-surge
├── Companion summons (14): charm, chrysalis-construct,
│   summon-hive-swarm, summon-steppe-spirit, conjure-water,
│   conjure-earth, conjure-air, conjure-fire, conjure-magma,
│   raise-skeleton, raise-zombie, raise-wraith, raise-spectre,
│   raise-golem, raise-vampire
├── Teleportation (2): fold-anchor, fold-recall
├── Healing (3): chrysalis-aid, heal, purge-affliction
├── Utility (1): identify
```

### Buff Scripts (dogmud) — 28 files
```
_datafiles/world/dogmud/buffs/
├── Flavor-only (~20): conviction-surge, iron-will, chrysalis-haste,
│   nerve-disruption, empathic-shroud, skill-attunement,
│   mutation-catalyst, psychic-anchor, sensory-overload, mind-fog,
│   clarity-tonic, fire-resistance, berserker-elixir, ...
├── Healing/DoT (6): vital-surge, chrysalis-regeneration, venom,
│   spore-toxin, toxic-cloud, minor-antidote
├── Utility (2): death-recovery, stamina/conviction draughts
```

### Mob Scripts (dogmud) — 13 files
```
_datafiles/world/dogmud/mobs/
├── thornwall_city/scripts/ — barmaid, Sable (economy), others
├── marches_spur_road/scripts/ — bandit leader
├── dustwalk_road/scripts/ — various
├── ironwind_steppe/scripts/ — various
├── startland/ — tutorial mobs
├── tutorial/scripts/ — training dummy
```

### Room Scripts (dogmud) — 27 files
```
_datafiles/world/dogmud/rooms/
├── sanctum_basin/ — quest rooms, complex triggers
├── thornwall_city/ — city interactions (if any)
├── startland/ — starter area
├── various zones — entry triggers, quest gates
```

## Appendix B: API Methods Used by JS (Frequency)

| Method | Calls (est.) | Category |
|--------|-------------|----------|
| `SendUserMessage()` | 400+ | Messaging |
| `SendRoomMessage()` | 350+ | Messaging |
| `actor.UserId()` | 300+ | Identity |
| `actor.GetCharacterName()` | 250+ | Identity |
| `actor.GetRoomId()` | 200+ | Location |
| `actor.AddHealth()` | 40+ | Resources |
| `actor.GetHealthMax()` | 30+ | Resources |
| `mob.Command()` | 25+ | AI |
| `UtilFindMatchIn()` | 20+ | Dialogue |
| `actor.GetStat()` | 15+ | Stats |
| `actor.GetSkillLevel()` | 15+ | Skills |
| `actor.GetCompanionCount()` | 14 | Companions |
| `room.SpawnMobScaled()` | 14 | Spawning |
| `actor.SetTempData()` | 12+ | State |
| `user.HasQuest()` | 10+ | Quests |
| `Math.floor()` | 25+ | Math |
| `UtilDiceRoll()` | 10+ | Randomness |
