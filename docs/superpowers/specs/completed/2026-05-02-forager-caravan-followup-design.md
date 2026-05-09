# Forager + Caravan Followup — Design

**Date:** 2026-05-02
**Status:** Draft (review pending)
**Supersedes / extends:** Stages 3.0–3.4 forager + caravan stack already
on `master`.

## Goal

Fix five reported issues in the now-shipped forager + caravan systems
and replace the abstract "presence-flag" Kessa↔caravan handoff with a
concrete world-object handoff. After this work the foragers + caravan
should run autonomously from server boot, never deadlock, never visit
off-route NPCs, and physically move materials between sanctuaries,
roadside crates, wagons, and town vendors.

## Background — what already shipped

Stage 3.4 (real item transfer) is **already on `master`** as of commit
`860b1a33`. Marsh and Steppe foragers physically move items into vendor
stocks during their delivery cycle. Caravan exists, runs Thornwall ↔
Stillwater, and visits 17 vendor rooms. A "rest extension" feature was
added to keep foragers home in saturated economies (when their satchel
remains heavy because vendors are full).

What is **still flag-based** (not item-based):

- The Kessa ↔ caravan handoff at North Road room 4038. The caravan,
  on dwell, scans for Kessa's mob-id presence and sets a
  `caravan_load: ["fernway"]` state flag if she's there. Kessa's
  satchel never drains; the caravan never receives items. The
  "fernway" flag is later used at vendor stops to top up
  fernway-bucket items.

What is **broken**:

1. Caravan tries to restock Whisper at room 507 (off-route NPC in the
   phantom's zone).
2. Foragers show "(not active)" on the `/admin/economy/` dashboard
   until a player physically walks to their room, despite commits
   `4cc54e7b` and `40a7d3f2` aimed at this.
3. There is no boot-time mechanism to start caravan or foragers —
   they spawn lazily on first player visit to their anchor rooms
   (4042, 4123, 3040, 4197). Wilderness anchors rarely get visited,
   so foragers stay offline indefinitely.
4. Foragers can stop foraging mid-life. Stage 3.4's "rest extension"
   intentionally parks them at sanctuary when satchel carry exceeds
   `ForagerRestCarryThreshold` (default 0.5). With no draining
   mechanism for unwanted surplus, this becomes effectively
   permanent in saturated economies.
5. Kessa never delivers anything to the caravan (see flag-based
   handoff above).

## Root causes (confirmed)

| # | Issue | Root cause | Evidence |
|---|---|---|---|
| 1 | Whisper restock | Hard-coded list literal includes room 507 | `internal/caravan/routes.go:29-40` |
| 2 | Dashboard "not active" | Forager state machine initializes lazily on first `forager_step` btree tick; the boot loop's `LoadRoom()` does not call `room.Prepare()` so spawninfo never fires | `main.go:1187-1206`, `internal/rooms/save_and_load.go:78-97`, `internal/rooms/rooms.go:585-588` |
| 3 | No boot loader | Same as #2 | Same |
| 4 | "Stop foraging" | Surplus accumulates in satchel; rest-extension gate never releases | `internal/behaviortree/actions_forager.go:149-160` (commit `860b1a33`) |
| 5 | Kessa→caravan no delivery | Handoff is flag-only, not item-based; cycle timing also requires both at room 4038 in the same tick window | `internal/behaviortree/actions_caravan.go:209-246`, `internal/behaviortree/actions_forager.go:283-300` |

## Design decisions

### Decision 1 — Whisper off the caravan rotation

Remove room 507 (Whisper) from `thornwallVendorRooms` in
`internal/caravan/routes.go`. Whisper is a quest-relevant NPC in the
phantom's zone, not a standard merchant. Confirm `stillwaterVendorRooms`
has no similar mistake.

Trivially scoped; no new mechanism.

### Decision 2 — Boot-spawn the system NPCs

Add a fourth boot step after the existing shop prewarm (around
`main.go:1207`) that calls `room.Prepare(false)` on a small,
explicitly-named set of "system NPC anchor rooms":

- `4042` — North Road Crossroads Village Square (caravan master 281)
- `4123` — Stillwater Marsh sanctuary (Marsh forager 371)
- `3040` — Ironwind Steppe sanctuary (Steppe forager 372)
- `4197` — Forager's Camp, Fernway South (Fernway forager 373)

The set is explicit, not derived. New entries get added by hand when
new system NPCs ship. We avoid auto-discovery (e.g., "any room whose
template includes a forager-grouped mob") because the set is small and
the cost of a boot panic from auto-discovery getting it wrong is
higher than the cost of one extra hand-edit.

Implementation note for the plan: this can ship as a list constant or
a small `[]int` config knob (`Balance.SystemNPCAnchorRooms`). Lean
toward the constant — config is overkill for four IDs.

### Decision 3 — Sanctuary lockboxes (replaces rest extension)

Each forager gets a **persistent, lockable container** at her
sanctuary room. When the forager arrives home from delivery, she
dumps her remaining satchel contents into the sanctuary lockbox
**before** entering the Resting state. Her satchel always returns to
empty between cycles; the carry-ratio gate becomes vestigial and we
remove it.

The Stage 3.4 rest extension is **superseded**: foragers always
re-cycle out of Resting after the dwell elapses, regardless of recent
vendor saturation. Surplus accumulates in the lockbox instead of in
the satchel, which is observable, drainable by players, and bounded.

#### Lockbox properties

| Property | Value | Why |
|---|---|---|
| Container type | Existing room `Container` system | Picklock + lock + look already wired |
| Visibility | Visible (not `Hidden`) | Players who trek to wilderness should see "a sturdy lockbox sits beside the firepit" without searching |
| Lock difficulty | 10 (hard) | High enough that low-skill picklockers have a meaningful failure rate and burn lockpicks; targets mid-to-high skullduggery |
| Lock trap | None | Players who reach a sanctuary already paid the wilderness travel cost; no need to gate further with a debuff |
| Persistence | Yes — write to `_datafiles/world/dogmud/foragers/<zone>/<mobid>-room<roomid>.yaml` (mirrors shop persistence) | Surplus must survive reboots so player progress on picking the lock isn't destroyed by a server restart |
| Capacity | 500 items | Large enough to absorb many cycles of surplus across long uninterrupted soaks; if the box hits cap the forager restores Stage 3.4 rest-extension behavior temporarily as a backstop |
| Re-lock cadence | Locks again after each forager dump (cycle-end) | "Make players work for the mats each time" per user direction |
| Keyring bypass | Disabled — players must repeat the picklock minigame each cycle | Default keyring system says "you can pick this any time you carry lockpicks"; for this lockbox we need to invalidate that |

#### Re-lock + keyring-bypass semantics

The existing `gamelock.Lock` has `RelockInterval` (auto-relock after
elapsed game-time) but the picklock keyring still short-circuits the
minigame on a known sequence. To require fresh picking each cycle,
extend `Lock` with a rotation seed:

```go
type Lock struct {
    Difficulty     uint8
    UnlockedRound  uint64
    RelockInterval string
    TrapBuffIds    []int
    RotationSeed   uint64 // NEW — bumped on SetLocked. Mixed into util.GetLockSequence so each rotation has a fresh combination, invalidating any cached keyring sequence.
}
```

`SetLocked()` increments `RotationSeed`. `util.GetLockSequence(lockId,
strength, serverSeed, rotationSeed)` mixes the rotation seed into the
combination derivation. The forager calls `SetLocked()` every time she
finishes dumping items into the box. Players' keyring entries for that
lockId still exist, but the combination has changed, so picklock falls
through to the minigame again.

This is a **generic engine extension**: any room container in the
game can opt into per-cycle fresh-locking by bumping its rotation
seed when desired. We won't enable it on existing chests; only the
three new forager lockboxes.

#### Forager state machine integration

In `tickForagerRecalling` (or a new `tickForagerStashing` that fires
between Recalling and Resting): when the forager arrives at sanctuary
with a non-empty satchel, drain matching items into the room's
lockbox container, fire `SetLocked()` to rotate the combination, and
proceed to Resting with an empty satchel. Drain logic walks the
satchel in reverse (RemoveItem index-safety pattern, mirrors
`npcVisitVendorsInRoom`).

#### Lockbox content rules

- Forager only deposits items whose `economy.BucketFor(itemId)`
  matches her own `Buckets` whitelist. Anything else is dropped to
  the room (legitimate corpse-salvage byproducts unrelated to her
  mat focus).
- Lockbox can hold any item type (no bucket filter on retrieval).
  Players who pick it get whatever happens to be inside.

### Decision 4 — Roadside shipping crate at North Road 4038

Replace the Kessa-presence flag handoff with a persistent **shipping
crate** at room 4038. Properties:

| Property | Value | Why |
|---|---|---|
| Type | New room object (special crate, not the standard Container with a Hidden flag) | Players must NOT be able to interact via standard `get`, `open`, `look in`, or `picklock` |
| Visibility | Visible — appears in room description and `look` | World-fiction; players passing by should see the crate |
| Player interaction | Read-only — `look crate` describes it as the caravan's; all other commands ("get crate", "open crate", "look in crate", "put X in crate") return a flavor message ("The crate is sealed and bound for the caravan; you can't get into it.") |
| Capacity | 2000 items | Per user direction; effectively unbounded for normal gameplay but capped to prevent disk/memory pathology |
| Persistence | Yes — `_datafiles/world/dogmud/crates/4038-fernway_shipment.yaml` |
| Locking | None (player access is denied at the command-handler layer, not via lockpick) |
| Auto-purge | If the crate is full and Kessa arrives with a load, she dumps overflow to the floor at her sanctuary lockbox (not the crate) so the system never wedges |

Implementation route: introduce a new lightweight engine concept,
`SealedCrate`, separate from `Container`. A `SealedCrate` lives on a
`Room` (one per room, keyed by room ID) and is mutated only by
`internal/caravan/...` and `internal/behaviortree/actions_forager.go`
code paths. Player command handlers (`get`, `look`, `put`, `open`,
`picklock`) get a small case in their dispatch to recognize the crate
noun and emit the locked-out flavor.

This is intentionally a **separate type** from `Container`. Reasons:

1. `Container` is built around player interaction — its lock,
   trap, keyring, and capacity are all tuned for the picklock loop.
   We don't want to fight that machinery.
2. A `SealedCrate` is a delivery primitive, not a stash. Future
   stages may want crates between other endpoints; a clean type now
   pays off then.
3. Keeps the player-vs-caravan boundary obvious in code review.

#### Kessa's new flow (cycle outline)

1. **Foraging** — same as today.
2. **TravelingToDropoff** — pathto room 4038.
3. **DeliveringFernway** — on arrival, dump fernway-bucket satchel
   contents into the room's `SealedCrate`. Emit room flavor:
   *"Kessa hauls a satchel up to the crate, latches it shut, and
   turns for home."* Transition immediately to `Recalling` (no more
   150-round wait timer).
4. **Recalling** — same as today, plus dump any remaining (non-
   fernway) satchel contents into her sanctuary lockbox per
   Decision 3.

#### Caravan's new flow at the pickup substate

- `tickFernwayPickup` no longer scans for Kessa's presence. On
  arrival at 4038, drain everything from the room's `SealedCrate`
  into the caravan wagon's inventory. Emit room flavor: *"The caravan
  pulls up to the roadside crate, breaks the seal, and loads its
  contents into the wagon."*
- The `caravan_load: ["fernway"]` flag goes away entirely. Real
  items are now in the wagon and will be distributed at vendor
  stops by the existing Stage 3.4 transfer logic.
- If the crate is empty when the caravan arrives, skip the flavor
  and proceed silently to next state.

#### Migration

The `caravan_load` flag is read at vendor stops to decide whether to
top up fernway-bucket items. Once item-based, this read goes away.
Plan task should grep all `caravanLoad{Get,Set,Append}` callsites and
delete them as part of the change.

## Architecture changes (summary)

| Layer | File | Change |
|---|---|---|
| Engine | `internal/gamelock/gamelock.go` | Add `RotationSeed uint64`; bump on `SetLocked` |
| Engine | `internal/util/lock_sequence.go` (or wherever `GetLockSequence` lives) | Mix `RotationSeed` into combination derivation |
| Engine (new) | `internal/sealedcrate/` | New package: `SealedCrate` type, persistence, room binding |
| Engine | `internal/rooms/rooms.go` | Add `SealedCrate *sealedcrate.Crate` field on `Room` |
| Engine | Boot loop in `main.go` | Add `room.Prepare(false)` step for system anchor rooms |
| Engine | `internal/behaviortree/actions_forager.go` | Drop rest-extension carry-ratio gate; add lockbox dump on Recalling→Resting; rewrite `tickForagerDeliveringFernway` to dump to crate |
| Engine | `internal/behaviortree/actions_caravan.go` | Rewrite `tickFernwayPickup` to drain crate into wagon; delete `caravanLoadAppend` calls everywhere |
| Engine | `internal/usercommands/{get,look,put,picklock,open}.go` | Recognize sealed-crate noun and emit flavor |
| Content | `internal/caravan/routes.go` | Remove room 507 from `thornwallVendorRooms` |
| Config | `internal/configs/config.balance.misc.go`, `_datafiles/config.yaml`, `internal/configs/config.balance_test.go` | `CaravanDepotDwellRounds` 720 → 360 |
| Content | `_datafiles/world/dogmud/rooms/{stillwater_marsh/4123,ironwind_steppe/3040,the_fernway_south/4197}.yaml` | Add lockbox container with `RotationSeed` set, visible, difficulty 6 |
| Content | `_datafiles/world/dogmud/rooms/north_road/4038.yaml` | Add sealed-crate room object + flavor noun |
| Persistence (new) | `_datafiles/world/dogmud/foragers/`, `_datafiles/world/dogmud/crates/` | New persistence directories |

### Decision 5 — Halve the caravan depot dwell

`Balance.CaravanDepotDwellRounds` was bumped from 360 to 720 in Stage
3.1 to make foragers the day-to-day supply pipeline and the caravan a
once-per-game-day event. With foragers now actually running from boot
(Decision 2) and never deadlocking (Decision 3), the foragers will
dominate throughput regardless. The caravan should be more visible
than realistic.

Drop `CaravanDepotDwellRounds` from 720 to 360. Update the inline
config comment in `_datafiles/config.yaml` and the engine default in
`internal/configs/config.balance.misc.go`. The Stage 3.1
config-default test (`internal/configs/config.balance_test.go:8`)
that pins 720 must be updated to 360.

This roughly doubles caravan arrival cadence in each town. Foragers
remain the steady throughput source; the caravan becomes the visible
event-style delivery the user prefers.

## Open questions

All previously-open questions resolved by user direction
(2026-05-02):

- Lockbox capacity → 500 items.
- Lockbox lock difficulty → 10 (hard).
- `SealedCrate` as a new type → confirmed.
- Caravan depot dwell → 360 rounds (was 720).

No remaining blockers.

## Out of scope

- Caravan reset / wipe behavior (already handled by Stage 3.4
  `distribute_cargo_to_hostiles`).
- Forager combat overhaul (separate roadmap item).
- Mob aliveness work — explicitly deferred per user direction;
  these fixes precede that effort.
- Additional foragers, additional crate routes, or any non-Fernway
  caravan handoff.
- Cosmetic crate variants (icebox, strongbox, etc.).

## Success criteria

After this work ships:

1. **Issue #1**: After a clean boot, the caravan reaches Whisper's
   room never. Verified by `grep -n 507` in `routes.go` returning no
   match and a server-log audit of one full caravan cycle.
2. **Issue #2 + #3**: After a clean boot with no player connections,
   `/admin/economy/` shows all three foragers with non-`(not active)`
   states (e.g., `resting`, `foraging`) within ~30 game rounds.
3. **Issue #4**: Foragers re-cycle out of Resting after the dwell
   regardless of vendor saturation. A long server soak (1000+ rounds)
   shows each forager completing ≥3 full cycles.
4. **Issue #5**: One full caravan cycle in which Kessa visits room
   4038 dumps her satchel into the sealed crate, and the next
   caravan pickup at 4038 drains the crate into the wagon. Items
   from Kessa's pickup land in town vendor stocks at later vendor
   stops.
5. **Sanctuary lockbox cheese path**: A player with lockpicks +
   skullduggery can pick the Marsh / Steppe / Fernway sanctuary
   lockbox, retrieve materials, and (after the next forager cycle)
   must re-do the minigame to pick it again — keyring entry no
   longer matches.

## Risk register

| Risk | Severity | Mitigation |
|---|---|---|
| `RotationSeed` change to `Lock` accidentally invalidates existing chests' keyring entries | High | Default `RotationSeed = 0`, mix-in is a no-op when zero. Only the new lockboxes set non-zero rotation seeds. Add a test pinning combination stability for `RotationSeed=0`. |
| `SealedCrate` collides with existing `Container` lookups | Medium | Separate type, separate map on Room. Player commands that match a noun against both (`look <crate>`) check `SealedCrate` first if present, fall through to `Container`. |
| Boot prewarm of system anchor rooms triggers spawn ordering bugs (e.g., a forager spawns before its zone's mob template loads) | Medium | Run the new boot step **after** all `LoadDataFiles` calls (mobs, items, rooms, mutators) — i.e., late in the existing boot sequence. Plan must verify ordering. |
| Sanctuary lockbox capacity 150 saturates before vendors clear, re-introducing the deadlock | Low | Backstop: if lockbox is full, fall back to Stage 3.4 rest-extension behavior temporarily. Forager stays home until lockbox drained (player picks) or vendors clear. |
| Removing `caravan_load` flag breaks tests / dashboards that read it | Low-Medium | Plan task includes a `grep -rn caravan_load` pass and removes/refactors all reads. |
