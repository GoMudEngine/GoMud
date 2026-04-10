# Instance Difficulty Scaling Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Scale instance mob difficulty based on gold paid — linear stat pool scaling with template multipliers preserving relative mob power.

**Architecture:** Add ~15 lines to `CreateZoneInstance()` that multiply each spawned mob's template stat pool by the gold paid. Add a config cap. Then create new mob content for both instance zones with multiplier-based stat pools (1 = trash, 2 = tough, 3 = boss).

**Tech Stack:** Go, YAML data files

**Spec:** `docs/superpowers/specs/2026-04-10-instance-difficulty-scaling-design.md`

---

### Task 1: Add InstanceStatPoolCap to Balance Config

**Files:**
- Modify: `internal/configs/config.balance.go`

- [ ] **Step 1: Add the config field**

In `internal/configs/config.balance.go`, add to the `BalanceConfig` struct:

```go
InstanceStatPoolCap ConfigInt `yaml:"InstanceStatPoolCap"` // Max stat pool per mob in instances (default 50000, 0=uncapped)
```

In the validation/defaults section, add:

```go
if b.InstanceStatPoolCap < 1 {
    b.InstanceStatPoolCap = 50000
}
```

- [ ] **Step 2: Compile and commit**

Run: `go build ./...`

```bash
git add internal/configs/config.balance.go
git commit -m "feat(instances): add InstanceStatPoolCap balance config"
```

---

### Task 2: Add Stat Pool Scaling to CreateZoneInstance

**Files:**
- Modify: `internal/rooms/instances.go`
- Test: `internal/rooms/instances_test.go`

- [ ] **Step 1: Write test for scaling logic**

Add to `internal/rooms/instances_test.go`:

```go
func TestScaleSpawnInfo(t *testing.T) {
	// Template stat pool acts as multiplier
	spawns := []SpawnInfo{
		{MobId: 1, StatPool: 1},  // trash: 1x
		{MobId: 2, StatPool: 2},  // tough: 2x
		{MobId: 3, StatPool: 3},  // boss: 3x
		{MobId: 4, StatPool: 0},  // unset: defaults to 1x
	}

	ScaleSpawnStatPools(spawns, 500, 50000)

	assert.Equal(t, 500, spawns[0].StatPool)
	assert.Equal(t, 1000, spawns[1].StatPool)
	assert.Equal(t, 1500, spawns[2].StatPool)
	assert.Equal(t, 500, spawns[3].StatPool)
}

func TestScaleSpawnInfo_Cap(t *testing.T) {
	spawns := []SpawnInfo{
		{MobId: 1, StatPool: 3},
	}

	ScaleSpawnStatPools(spawns, 20000, 50000)

	assert.Equal(t, 50000, spawns[0].StatPool) // 60000 capped to 50000
}

func TestScaleSpawnInfo_NoCap(t *testing.T) {
	spawns := []SpawnInfo{
		{MobId: 1, StatPool: 3},
	}

	ScaleSpawnStatPools(spawns, 20000, 0) // 0 = uncapped

	assert.Equal(t, 60000, spawns[0].StatPool)
}
```

- [ ] **Step 2: Run tests — should fail**

Run: `go test ./internal/rooms/ -run TestScaleSpawnInfo -v`

- [ ] **Step 3: Implement ScaleSpawnStatPools**

Add to `internal/rooms/instances.go`:

```go
// ScaleSpawnStatPools multiplies each spawn's StatPool by goldPaid.
// Template stat pools act as multipliers (1=trash, 2=tough, 3=boss).
// Spawns with StatPool 0 default to 1x. Cap of 0 means uncapped.
func ScaleSpawnStatPools(spawns []SpawnInfo, goldPaid int, cap int) {
	for i := range spawns {
		mult := spawns[i].StatPool
		if mult < 1 {
			mult = 1
		}
		scaled := goldPaid * mult
		if cap > 0 && scaled > cap {
			scaled = cap
		}
		spawns[i].StatPool = scaled
	}
}
```

- [ ] **Step 4: Run tests — should pass**

Run: `go test ./internal/rooms/ -run TestScaleSpawnInfo -v`

- [ ] **Step 5: Wire into CreateZoneInstance**

In `internal/rooms/instances.go`, in `CreateZoneInstance()`, add
the scaling loop after the return portal creation (step 5) and
before the temp data stamping (step 6). Find the comment
`// 6. Stamp instance metadata` and add before it:

```go
	// 5b. Scale mob stat pools based on gold paid.
	cap := int(configs.GetBalanceConfig().InstanceStatPoolCap)
	for _, ephId := range roomIdMap {
		if room := LoadRoom(ephId); room != nil {
			ScaleSpawnStatPools(room.SpawnInfo, goldPaid, cap)
		}
	}
```

Add `"github.com/GoMudEngine/GoMud/internal/configs"` to imports
if not already present.

- [ ] **Step 6: Compile and commit**

Run: `go build ./...`

```bash
git add internal/rooms/instances.go internal/rooms/instances_test.go
git commit -m "feat(instances): scale mob stat pools by gold paid"
```

---

### Task 3: Update Instance Zone Templates with Multiplier Stat Pools

**Files:**
- Modify: `_datafiles/world/dogmud/rooms/instance_arena/5002.yaml`
- Modify: `_datafiles/world/dogmud/rooms/instance_planar_oasis/5004.yaml`

- [ ] **Step 1: Update arena floor spawn info**

In `_datafiles/world/dogmud/rooms/instance_arena/5002.yaml`, update
the spawn info to use multiplier-based stat pools:

```yaml
spawninfo:
  - mobid: 73
    statpool: 1
    respawnrate: "15m"
  - mobid: 73
    statpool: 1
    respawnrate: "15m"
```

Note: mob 73 is a placeholder. Task 4 will create proper arena mobs
and update these mob IDs.

- [ ] **Step 2: Update oasis sands spawn info**

In `_datafiles/world/dogmud/rooms/instance_planar_oasis/5004.yaml`:

```yaml
spawninfo:
  - mobid: 311
    statpool: 1
    respawnrate: "15m"
  - mobid: 312
    statpool: 1
    respawnrate: "15m"
```

Note: mobs 311/312 are placeholders. Task 5 will create proper
oasis mobs and update these.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/rooms/instance_arena/5002.yaml \
  _datafiles/world/dogmud/rooms/instance_planar_oasis/5004.yaml
git commit -m "feat(instances): use multiplier stat pools in zone templates"
```

---

### Task 4: Create Arena Mobs

**Files:**
- Create: `_datafiles/world/dogmud/mobs/instance_arena/316-arena_brute.yaml`
- Create: `_datafiles/world/dogmud/mobs/instance_arena/317-arena_cutthroat.yaml`
- Modify: `_datafiles/world/dogmud/rooms/instance_arena/5002.yaml`

**IMPORTANT:** Before creating files, verify the next available mob
ID by running:
```
find _datafiles/world/dogmud/mobs -name "*.yaml" | sed 's/.*\///' | sed 's/-.*//' | sort -n | tail -5
```
Use IDs above the highest found (currently 315, so 316+).

**IMPORTANT:** The mob folder must be `instance_arena` (matching
`ConvertForFilename("Instance Arena")`).

- [ ] **Step 1: Create arena brute (trash mob)**

Create `_datafiles/world/dogmud/mobs/instance_arena/316-arena_brute.yaml`:

```yaml
mobid: 316
zone: Instance Arena
archetype: fighting
statpool: 1
hostile: true
groups:
  - humanoid
activitylevel: 20
idlecommands:
  - 'emote cracks knuckles and paces the pit'
  - ''
  - 'emote slams a fist into an open palm'
  - 'wander'
combatcommands:
  - 'emote throws a savage haymaker'
  - ''
  - 'emote lunges forward with reckless aggression'
  - ''
character:
  name: arena brute
  description: >-
    A massive figure scarred from countless pit fights. Thick
    arms corded with muscle, hands wrapped in stained cloth
    bindings. Moves with the confidence of someone who has
    beaten everything put in front of them — or the ignorance
    of someone who has not yet met their match.
  speciesid: 1
  level: 1
  gold: 0
```

- [ ] **Step 2: Create arena cutthroat (trash mob, different style)**

Create `_datafiles/world/dogmud/mobs/instance_arena/317-arena_cutthroat.yaml`:

```yaml
mobid: 317
zone: Instance Arena
archetype: fighting
statpool: 1
hostile: true
groups:
  - humanoid
activitylevel: 20
idlecommands:
  - 'emote flips a blade between fingers with practiced ease'
  - ''
  - 'emote watches the shadows, waiting'
  - ''
combatcommands:
  - 'emote darts in low, blade seeking soft flesh'
  - ''
  - 'emote feints left and strikes right'
  - ''
character:
  name: arena cutthroat
  description: >-
    Lean and quick, built for speed over power. A pair of
    short blades hang from a belt of cracked leather. The
    eyes are flat and assessing — the look of someone who
    counts wounds rather than kills, and prefers the ones
    that bleed slowly.
  speciesid: 1
  level: 1
  gold: 0
  stats:
    dexterity:
      training: 5
    perception:
      training: 3
```

- [ ] **Step 3: Update arena floor to use new mobs**

Update `_datafiles/world/dogmud/rooms/instance_arena/5002.yaml`:

```yaml
spawninfo:
  - mobid: 316
    statpool: 1
    respawnrate: "15m"
  - mobid: 317
    statpool: 1
    respawnrate: "15m"
```

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/mobs/instance_arena/ \
  _datafiles/world/dogmud/rooms/instance_arena/5002.yaml
git commit -m "feat(instances): add arena brute and cutthroat mobs"
```

---

### Task 5: Create Oasis Mobs + Boss Variants

**Files:**
- Create: `_datafiles/world/dogmud/mobs/instance_planar_oasis/318-sand_elemental.yaml`
- Create: `_datafiles/world/dogmud/mobs/instance_planar_oasis/319-storm_elemental.yaml`
- Create: `_datafiles/world/dogmud/mobs/instance_planar_oasis/320-elemental_king.yaml`
- Create: `_datafiles/world/dogmud/mobs/instance_planar_oasis/321-elemental_queen.yaml`
- Create: `_datafiles/world/dogmud/mobs/instance_planar_oasis/322-elemental_prince.yaml`
- Modify: `_datafiles/world/dogmud/rooms/instance_planar_oasis/5004.yaml`
- Create: new boss room `_datafiles/world/dogmud/rooms/instance_planar_oasis/5005.yaml`
- Modify: zone config and entry room exits as needed

**IMPORTANT:** Verify next available mob IDs and room IDs before
creating. Mob IDs should be 316+ (check Task 4 didn't exceed 317).
Room ID 5005 should be free (verify with find command).

**IMPORTANT:** The mob folder must be `instance_planar_oasis`
(matching `ConvertForFilename("Instance Planar Oasis")`).

- [ ] **Step 1: Create sand elemental (trash)**

Create `_datafiles/world/dogmud/mobs/instance_planar_oasis/318-sand_elemental.yaml`:

```yaml
mobid: 318
zone: Instance Planar Oasis
archetype: fighting
statpool: 1
hostile: true
groups:
  - elemental
activitylevel: 25
idlecommands:
  - 'emote shifts and reforms, sand cascading in slow spirals'
  - 'wander'
  - 'emote rises from the ground in a column of whirling grit'
  - ''
combatcommands:
  - 'emote slams a fist of compacted sand forward'
  - ''
  - 'emote hurls a spray of razor-sharp crystals'
  - ''
character:
  name: sand elemental
  description: >-
    A vaguely humanoid shape of swirling sand and crystalline
    grit. It moves in stuttering lurches, reforming after
    each step as if the wind cannot decide what shape it
    should take. Granules of glass glint within the mass
    like embedded teeth.
  speciesid: 10
  level: 1
  gold: 0
```

Note: `speciesid: 10` should be the elemental species. Check what
species ID the existing elementals (311, 312) use and match it.

- [ ] **Step 2: Create storm elemental (trash, casting variant)**

Create `_datafiles/world/dogmud/mobs/instance_planar_oasis/319-storm_elemental.yaml`:

```yaml
mobid: 319
zone: Instance Planar Oasis
archetype: casting
statpool: 1
hostile: true
groups:
  - elemental
activitylevel: 25
idlecommands:
  - 'emote crackles with arcs of static discharge'
  - 'wander'
  - 'emote swirls in a tight vortex of wind and lightning'
  - ''
combatcommands:
  - 'emote hurls a bolt of compressed air'
  - ''
  - 'emote erupts in a burst of electrical fury'
  - ''
character:
  name: storm elemental
  description: >-
    A roiling mass of dark cloud and flickering lightning,
    compressed into a roughly bipedal shape. The air around
    it tastes of copper and ozone. Miniature thunderclaps
    sound with each movement, and the hair on your arms
    stands on end in its presence.
  speciesid: 10
  level: 1
  gold: 0
```

- [ ] **Step 3: Create elemental king (boss, fighter archetype)**

Create `_datafiles/world/dogmud/mobs/instance_planar_oasis/320-elemental_king.yaml`:

```yaml
mobid: 320
zone: Instance Planar Oasis
archetype: fighting
statpool: 3
hostile: true
groups:
  - elemental
activitylevel: 30
tactic_preset: aggressive_melee
tactical_discipline: 0.90
combatcommands:
  - 'emote brings both fists down in a thunderous overhead strike'
  - ''
  - 'emote sweeps a massive arm in a wide arc of stone and fire'
  - 'bash'
  - 'emote roars, the ground cracking beneath its feet'
  - ''
character:
  name: elemental king
  description: >-
    A towering figure of fused stone and molten earth, twice
    the height of a man. A crude crown of obsidian juts from
    its head like a volcanic ridge. Each step sends tremors
    through the ground. The heat radiating from its core
    warps the air into shimmering curtains. This is not a
    creature to be reasoned with. It is a force of nature
    wearing a shape, and it has noticed you.
  speciesid: 10
  level: 1
  gold: 0
  stats:
    strength:
      training: 10
    vitality:
      training: 10
```

- [ ] **Step 4: Create elemental queen (boss, caster archetype)**

Create `_datafiles/world/dogmud/mobs/instance_planar_oasis/321-elemental_queen.yaml`:

```yaml
mobid: 321
zone: Instance Planar Oasis
archetype: casting
statpool: 3
hostile: true
groups:
  - elemental
activitylevel: 30
tactic_preset: caster_backline
tactical_discipline: 0.95
combatcommands:
  - 'emote gestures and the air crystallizes into razor shards'
  - ''
  - 'emote draws moisture from the air into a crushing wave'
  - ''
  - 'emote whispers and the sand rises in choking spirals'
  - ''
character:
  name: elemental queen
  description: >-
    A slender figure of translucent crystal and flowing
    water, moving with an alien grace that is almost
    beautiful until you see the eyes — two points of cold
    blue light that regard you with the dispassionate
    curiosity of something that has never needed to feel
    fear. The temperature drops noticeably in her presence.
  speciesid: 10
  level: 1
  gold: 0
  stats:
    willpower:
      training: 10
    charisma:
      training: 8
  skills:
    spellcasting: 5
```

- [ ] **Step 5: Create elemental prince (boss, rogue archetype)**

Create `_datafiles/world/dogmud/mobs/instance_planar_oasis/322-elemental_prince.yaml`:

```yaml
mobid: 322
zone: Instance Planar Oasis
archetype: fighting
statpool: 3
hostile: true
groups:
  - elemental
activitylevel: 30
tactic_preset: aggressive_melee
tactical_discipline: 0.85
combatcommands:
  - 'emote flickers and strikes from an impossible angle'
  - ''
  - 'emote dissolves into mist and reforms behind its target'
  - 'trip'
  - 'emote lashes out with whip-thin tendrils of superheated air'
  - ''
character:
  name: elemental prince
  description: >-
    A lithe shape of smoke and flickering embers, constantly
    shifting between solid and vapor. It moves too fast to
    track cleanly — your eyes keep sliding off it as if it
    exists slightly out of phase with reality. When it
    strikes, it is already somewhere else.
  speciesid: 10
  level: 1
  gold: 0
  stats:
    dexterity:
      training: 12
    perception:
      training: 8
  skills:
    skullduggery: 3
```

- [ ] **Step 6: Add a boss room to the oasis zone**

Create `_datafiles/world/dogmud/rooms/instance_planar_oasis/5005.yaml`:

**IMPORTANT:** Verify room ID 5005 is free first:
```
find _datafiles/world/dogmud/rooms -name "5005.yaml"
```

```yaml
roomid: 5005
zone: Instance Planar Oasis
title: The Oasis Heart
description: >-
  The dark water of the oasis stretches before you, utterly
  still. The sky above has settled into a permanent twilight.
  Something vast moves beneath the surface — a shadow that
  covers the entire pool. The crystalline sand at the water's
  edge has been fused into glass by repeated exposure to
  tremendous heat. Whatever rules this place is here.
exits:
  south:
    roomid: 5004
    mapdirection: south
spawninfo:
  - mobid: 320
    statpool: 3
    respawnrate: "30m"
```

Note: Only one boss spawns. To randomize which boss appears, you
could add all three with very low spawn rates and use the engine's
existing spawn mechanics — or handle it in a future content pass.
For now, the king spawns by default.

- [ ] **Step 7: Update Shimmering Sands to connect to boss room**

In `_datafiles/world/dogmud/rooms/instance_planar_oasis/5004.yaml`,
add a north exit:

```yaml
exits:
  south:
    roomid: 5003
    mapdirection: south
  north:
    roomid: 5005
    mapdirection: north
```

- [ ] **Step 8: Commit**

```bash
git add _datafiles/world/dogmud/mobs/instance_planar_oasis/ \
  _datafiles/world/dogmud/rooms/instance_planar_oasis/
git commit -m "feat(instances): add oasis elementals and boss variants"
```

---

### Task 6: Integration Test

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -v 2>&1 | tail -20`

- [ ] **Step 2: Build**

Run: `go build ./...`

- [ ] **Step 3: Manual test at multiple gold levels**

Start server, nuke instance saves, test:

1. `ask sable arena 200` — enter, verify mobs have weak stats
   (use `consider` or `assess` if available)
2. `ask sable arena 1000` — enter, verify mobs are noticeably
   tougher
3. `ask sable oasis 500` — enter, verify trash elementals and
   boss room with elemental king
4. Verify boss has 3x the stat pool of trash mobs

- [ ] **Step 4: Commit any fixups**

```bash
git add -A
git commit -m "chore: integration fixups for difficulty scaling"
```
