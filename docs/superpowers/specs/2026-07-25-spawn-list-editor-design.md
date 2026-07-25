# Spawn-List Editor (Admin Web-Building Sub-Project 4) — Design

**Date:** 2026-07-25
**Status:** Approved design, pre-implementation
**Epic:** Admin web-building (sub-project 4 of 5; follows 1a coords, 1b room
builder, 2 items, 3 mobs, and the zone editor)
**Predecessor specs:** `2026-07-22-admin-builder-page-1b-design.md`,
`2026-07-23-mob-authoring-3-design.md`,
`2026-07-25-zone-lifecycle-config-design.md`

## Goal

Author a room's **spawn list** from the `/build` admin page: which mobs, items
and gold appear in a room, how often they come back, and what overrides apply
to each — without hand-editing `spawninfo:` in YAML.

This is the last structural gap in the builder. Rooms, items, mobs and zones
can all be authored from the web; placing a mob into a room still cannot.

Scope decisions (user-confirmed):

- **Lives in the Rooms-tab inspector**, not a fifth tab. A spawn list belongs
  to a room, so it is edited on the room already selected on the map.
- **Full field parity**, with the ten unused per-spawn overrides behind a
  collapsible Advanced drawer — matching the item and mob editors.
- **The 59 inert `cooldown:` lines are deleted, not converted.** Zero
  gameplay change. See §6.

## Background: what a spawn list actually is

`rooms.SpawnInfo` (`internal/rooms/spawninfo.go`) is a per-room list. Measured
across the live world (721 rooms carry one; 595 spawn entries total):

| field | authored uses | notes |
|-------|--------------|-------|
| `mobid` | 582 | mob template to spawn |
| `respawnrate` | 90 | e.g. `5 real minutes`; **unset means 15 real minutes** |
| `message` | 75 | overrides the default spawn message |
| `cooldown` | 59 | **NOT A FIELD — inert.** See §6 |
| `itemid` | 13 | item template to spawn |
| `container` | 4 | routes the item/gold into a container instead of the floor |
| `gold` | 0 | |
| `name`, `forcehostile`, `maxwander`, `idlecommands`, `scripttag`, `questflags`, `buffids`, `statpool`, `statpoolmod` | 0 each | per-spawn overrides of the mob template |

Ten fields exist and have never been authored once. The working assumption is
that this reflects their invisibility in YAML rather than a lack of need —
surfacing them is part of the point of the editor.

`InstanceId` and `DespawnedRound` are `yaml:"-"` runtime tracking and are not
editable.

### Spawn lists are NOT shadowed by instance saves

`Room.SpawnInfo` carries the struct tag `instance:"skip"`
(`internal/rooms/rooms.go:103`), and `restoreSkipTaggedFields`
(`internal/rooms/save_and_load.go:137`) copies every such field from the
**template** over the instance overlay after loading, explicitly so
"template-owned fields are not corrupted by stale data in pre-fix instance
files."

So a spawn-list edit takes effect on the next room load without wiping
`rooms.instances/`. **This contradicts CLAUDE.md's instance-save SOP**, which
still lists "room spawn lists" among the things stale instance saves silently
shadow. That line is stale and should be corrected separately — it is the
document people consult when a change "isn't taking effect", so a false entry
there costs real debugging time.

## 1. Home: the Rooms inspector

No new mode. The Rooms-tab inspector gains a **Spawns** section listing the
room's entries, each expandable, plus an add control.

Server side:

- `buildRoomDetail` (`modules/gmcp/gmcp.Build.go`) gains a `spawns` array so
  `Build.Room.Get` carries the list.
- `roomUpdateReq` gains `spawns`, so the existing `Build.Room.Update` verb
  saves it. **No new GMCP verb.** Spawns are part of the room, and splitting
  them into their own verb would mean two round trips and two failure modes
  for one Save button.

The existing `buildDeps` seam is reused; spawn validation is pure and testable
against the fake world exactly like the room field validation already there.

## 2. Entry model — one kind per entry

A spawn entry is a **mob**, an **item**, or **gold**. `mobid`, `itemid` and
`gold` are mutually exclusive: the editor asks for the kind first and shows
only that kind's target field, rather than presenting three fields a user
could contradictorily fill.

`container` is orthogonal and applies to item and gold entries: set, the
spawned item or gold goes into that container instead of onto the floor.
A mob entry with a container set is a validation error.

Server-side validation rejects an entry with more than one of
`mobid`/`itemid`/`gold` set, so a hand-edited YAML that contradicts itself is
caught at save rather than behaving unpredictably at spawn time.

## 3. Fields

**Always visible** (the six the world actually uses):

| Field | Control | Validation |
|-------|---------|------------|
| kind | radio: mob / item / gold | exactly one |
| target | mob or item **picker** | must exist |
| gold | number | kind=gold only, > 0 |
| container | text, hinted with the room's container nouns | must match a container noun in this room |
| respawn rate | text | must parse as an engine period; hint states unset = 15 real minutes |
| message | text | — |

**Advanced drawer** (collapsed; the ten currently-unused overrides):
`name`, `forcehostile`, `maxwander`, `idlecommands` (list), `scripttag`,
`questflags` (list), `buffids` (list), `statpool`, `statpoolmod`.

`buffids` are validated against known buff ids; `questflags` are free-form
(they are set on the mob, not resolved against the quest registry).

### Pickers, not typing

Mob and item targets are chosen from server-supplied lists — the same rule
held since sub-project 2: a reference that can brick a boot or dangle must not
be typeable. `Build.Mob.List` and `Build.Item.List` already exist and are
already admin-gated; the room inspector consumes them rather than adding
parallel plumbing.

The `respawnrate` field carries a hint naming the default, because "unset
means 15 real minutes" is currently invisible and is the single most likely
thing for an author to get wrong.

## 4. Ordering and removal

Entries are a list, and order is preserved on save. The editor supports
add, remove, and reorder. Removal is immediate — a spawn entry has no
downstream references, so unlike item/mob/zone deletion there is nothing to
scan for and no blocker report.

## 5. What this does NOT edit

- **`Room.Stash` / ground items.** The loot SOP is mobs, chests and quest
  rewards — never the ground — so an editor that made ground-loot easy would
  work against it.
- **Mob templates.** Overrides here apply to *this spawn*; editing the
  creature itself remains sub-project 3's job. The panel links across rather
  than duplicating the form.
- **Respawn pacing as a tuning exercise.** See §6.

## 6. Migration: the 59 inert `cooldown:` lines

`cooldown` is not a field on `SpawnInfo`. The real field is `respawnrate`.
The boot smoke test already baselines it as a known silently-ignored key
(`boot_smoke_test.go`, `"cooldown|rooms.SpawnInfo"`) with the note "authored
values doing nothing."

**Correction to that note, twice over.** It says 118×. The first measurement
here said **36** — also wrong. Spawn lists are authored at two indentation
depths, flush (`- mobid:`, keys at 2 spaces) and indented (`  - mobid:`, keys
at 4), and a depth-specific grep silently sees only one of them. The true
count is **59** lines across room templates and **0** in instance saves.

The drift gate is what caught the shortfall: after the first pass it still
reported `cooldown|rooms.SpawnInfo` as present. Any future key cleanup should
match indentation-agnostically and let the gate confirm, rather than trusting
a grep.

Distribution of the first-found 36 — all early-game, not dungeons:
`stillwater` 20, `the_fernway_south` 7, `stillwater_marsh` 7. The further 23
sit in `greenford`, `hartcharn`, `new_plymouth_*`, `pothole_coulee` and
`the_confluence`.

Values are in **rounds** (600×19, 300×11, 1800×2, 1200×2, 900×1, 2400×1),
whereas every live `respawnrate` in the world is in **real minutes** (2–15).
At `RoundSeconds: 4`, converting verbatim would mean:

| authored | real time | vs today's effective default |
|----------|-----------|------------------------------|
| 300 rounds | 20 min | 1.3× slower |
| 600 rounds | 40 min | 2.7× slower |
| 1200–2400 rounds | 80–160 min | 5–11× slower |

**Decision: delete the lines, do not convert.** These spawns have always run
on the 15-minute default; the world has been balanced and playtested against
that for months. Converting would quietly make those starter-area spawn points
1.3–11× slower as a side effect of a cleanup nobody asked to change gameplay.

The authored values are recorded in this table so a deliberate respawn-pacing
pass can use them later — see the existing zone-spawn-pacing backlog item —
rather than inheriting them by accident.

After the strip, delete `"cooldown|rooms.SpawnInfo"` from
`knownSilentlyIgnoredKeys`. The drift gate then proves the cleanup: the
distinct-ignored-key count must **drop**, and a regression would fail the gate.

## 7. Testing & verification gate

**Unit** (`modules/gmcp`, fake world): an entry with two of
mob/item/gold is rejected; a mob entry with a container is rejected; an
unknown mob or item id is rejected; a container not present in the room is
rejected; an unparseable respawn rate is rejected; a valid list round-trips
through `Build.Room.Update`; order is preserved; a non-admin is silently
refused (inherited from the existing gate, asserted once).

**Migration**: a test proving the strip removes only `cooldown:` lines and
leaves every other byte of those 36 files untouched — same discipline as the
zone rename's line rewrite, and for the same reason: a YAML re-marshal would
reflow every description in the file.

**Boot + gates**: clean boot under `MapConsistencyEnforce: panic`;
`TestSmoke_NoNewSilentlyIgnoredYAMLKeys` passing with a **reduced** count and
the baseline entry gone.

**Browser gate**: a human drives `/build` → Rooms → a room with spawns.
Add a mob spawn, set a respawn rate, open Advanced, remove an entry, save,
and confirm the mob actually appears in game.

## 8. Doc fix (small, included)

Correct the CLAUDE.md instance-save SOP: remove "room spawn lists" from the
list of things stale instance saves shadow, since `instance:"skip"` +
`restoreSkipTaggedFields` make that false. Leave the rest of the SOP intact —
it is accurate for the fields that genuinely are shadowed.

## Out of scope

- Ground items / `Room.Stash` (loot SOP).
- Bulk spawn queries ("where does mob 336 spawn?") — a Rooms-inspector
  section is per-room by construction. If world-wide spawn search is wanted
  later it is its own piece.
- Respawn-rate tuning (§6).
- Sub-project 5 (dialogue / behavior trees / quests) remains deferred.
