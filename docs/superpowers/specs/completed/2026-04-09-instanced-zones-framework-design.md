# Instanced Zones Framework Design

**Date:** 2026-04-09
**Scope:** Sub-project 1 of 3 — Instance framework only. Difficulty
scaling and randomized loot are separate sub-projects.

---

## Overview

Player pays an NPC a variable gold amount. The NPC opens a time-limited
portal scoped to the payer + their current party + companions. The portal
clones a pre-authored zone template into ephemeral rooms via the existing
ephemeral system. Gold amount is stored on the instance and passed to the
difficulty scaling system (sub-project 2). No quests, no win conditions —
just combat and loot.

---

## Instance Zone Template

Each instanced zone is a standard set of rooms authored in
`_datafiles/world/dogmud/rooms/<zone>/` with zone config fields marking
it as an instance template:

```yaml
# zone-config.yaml additions for instanced zones
instanced: true
death_policy: rejoin       # "rejoin" or "ejected"
portal_duration: "30m"     # how long the entry portal stays open
entry_room: <roomId>       # room the portal drops players into
allow_recall: true         # whether recall spell works inside
```

Rooms, exits, mob spawns, containers — all authored normally using
existing tools. The framework clones them into ephemeral copies. No
special fields needed on individual rooms. Wrapping exits (e.g., a
planar grid whose edges connect to the opposite side) are defined in
the room exit data as normal room-to-room references.

Teleport entry is always blocked for instanced zones — no config flag
needed. The only way in is through the purchased portal.

---

## Portal Lifecycle

### 1. Purchase

Player interacts with an instance vendor NPC. Dialogue flow:

1. Player asks NPC about available zones
2. NPC describes zones and asks how much gold to invest
3. Player specifies gold amount (minimum per zone, no maximum)
4. NPC confirms cost and lists who will be admitted:
   `"The portal will admit you and your current party: [names]"`
5. Gold is deducted. NPC creates a `TemporaryRoomExit` (portal) in the
   current room, pointing to the ephemeral entry room.

The portal is tagged with an **access list**: a snapshot of authorized
userIds captured at creation time.

### 2. Entry

- Authorized players walk through the portal like any exit.
- Companions follow their owner automatically (standard movement).
- Unauthorized players get a rejection message:
  `"The portal's energy pushes you back. It wasn't opened for you."`
- Teleport spells targeting rooms inside an active instance are blocked:
  `"A ward prevents your magic from reaching that place."`

### 3. Timer Expiration

When `portal_duration` elapses:
- The entry portal disappears from the overworld room.
- No new entries are possible.
- A **return portal** in the instance entry room remains active. Players
  inside can leave at any time by walking through it.

### 4. Soft Close

The instance stays alive as long as any player is inside. The return
portal in the entry room is always available.

### 5. AFK Boot

Players who go AFK inside an instance are ejected to the overworld
room where the portal was created, after the standard AFK timeout.

### 6. Cleanup

When the last player leaves (or is ejected), the ephemeral zone is
destroyed via the existing ephemeral cleanup system. All remaining
items on the ground, mob corpses, etc. are lost.

---

## Death Handling

Configurable per zone via `death_policy`:

### `rejoin` (default)

- Player dies, respawns at the death recovery room (normal flow).
- If the entry portal is still open, they can re-enter and rejoin
  their party.
- Instance state persists — mobs stay dead/alive as they were.
- Items dropped on death remain on the ground inside the instance.

### `ejected`

- Player dies, respawns at the death recovery room.
- They are removed from the instance's authorized list. The portal
  no longer admits them.
- Items dropped on death are lost (instance may despawn).
- Remaining party members can continue without them.

---

## Access Control

Portal stores a frozen snapshot of authorized userIds at creation:

- Payer's userId (the portal owner)
- All party member userIds at time of creation
- Companions are not tracked separately — they follow their owner
  through normal movement

Rules:
- New party members added AFTER portal creation are NOT authorized
- If the portal owner disconnects, the portal and instance persist
  on their timer. Party leadership transfers normally. The owner can
  reconnect and re-enter (they're on the access list).
- Party disbanding does not affect portal access — the access list
  is independent of the party object.

### Teleport Blocking

Any spell, command, or mechanic that moves a player to a room must
check: is the target room in an ephemeral instance? If so, is the
player on that instance's authorized list? Block if not.

This is a single gatekeeper check on room entry, not per-spell
patching. The check applies to:
- Teleport spells
- Any future movement mechanic that bypasses normal exits

The recall spell is handled separately via `allow_recall` on the
zone config. If false, recall is blocked inside the instance:
`"Something about this place prevents you from recalling."`

---

## Instance Data

Each active instance is tracked in a runtime registry (not persisted
to disk — instances are ephemeral):

```
InstanceId        int       // ephemeral chunk ID
TemplateZone      string    // zone name that was cloned
GoldPaid          int       // gold amount (for scaling system)
AuthorizedUsers   []int     // userId snapshot at creation
OwnerUserId       int       // who paid
CreatedRound      uint64    // for timer tracking
PortalDuration    string    // from zone config
DeathPolicy       string    // from zone config
AllowRecall       bool      // from zone config
OverworldRoomId   int       // room where the portal was created
OverworldExitName string    // exit name in that room (for cleanup)
EntryRoomId       int       // ephemeral entry room ID
RoomIdMap         map[int]int // original → ephemeral room ID mapping
```

---

## NPC Interface

Instance vendor NPCs use a script-based interaction (not YAML dialogue,
to avoid quest engine complexity). The script handles:

1. Listing available instanced zones
2. Accepting gold amount input from the player
3. Validating minimum gold requirement per zone
4. Creating the ephemeral zone clone
5. Creating the portal (TemporaryRoomExit) with access list
6. Creating the return portal in the instance entry room
7. Sending confirmation messages with zone name, duration, and
   authorized party members

---

## Player-Facing Messaging

Clear communication at every step:

**Purchase:**
- `"<NPC> says, 'For <X> gold, I'll open a portal to <Zone>.
  It will remain open for <duration>.'"
- `"The portal will admit: <player>, <party member 1>, <party member 2>"`
- `"Party up before purchasing! Only current party members will
  be able to enter."`

**Portal description** (visible via `look`):
- `"A shimmering portal pulses with unstable energy. It leads to
  <Zone>. Authorized: <player list>. Time remaining: <duration>."`

**Entry rejection:**
- `"The portal's energy pushes you back. It wasn't opened for you."`

**Teleport block:**
- `"A ward prevents your magic from reaching that place."`

**Recall block (if disabled):**
- `"Something about this place prevents you from recalling."`

**Timer warnings (broadcast to players inside):**
- At 5 minutes remaining: `"The portal flickers — it won't hold
  much longer."`
- At 1 minute remaining: `"The portal is barely a shimmer now.
  Leave soon or find your own way out."`
- On expiration: `"The entry portal collapses. The return portal
  in the entry chamber still glows steadily."`

**AFK eject:**
- `"You've been idle too long. The unstable magic of this place
  expels you."`

---

## Existing Systems Leveraged

- **Ephemeral rooms** (`internal/rooms/ephemeral.go`): Clone zone
  templates, rewrite exits, auto-cleanup on empty.
- **TemporaryRoomExit**: Time-limited exits already supported on rooms.
- **Party system** (`internal/parties/`): Membership queries, leader
  transfer on disconnect.
- **Mob spawning** (`Room.Prepare()`): Standard spawn system works in
  ephemeral rooms — mobs spawn when a player first enters.
- **AFK system**: Existing AFK detection + timeout.

---

## Help Files

New help topics needed:
- `help instances` — overview of the instanced zones system
- `help portal` — how portals work, access rules, timer
- Update vendor NPC dialogue/help if applicable

---

## What This Spec Does NOT Cover

- Difficulty scaling math (sub-project 2)
- Randomized loot / affixes (sub-project 3)
- Procedural zone generation (future framework)
- Specific zone content (arena rooms, planar grid rooms)
- Traps or environmental hazards (future enhancement)
