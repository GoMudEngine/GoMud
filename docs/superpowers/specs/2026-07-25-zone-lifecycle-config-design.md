# Zone Lifecycle & Config Editor (Admin Web-Building) — Design

**Date:** 2026-07-25
**Status:** Approved design, pre-implementation
**Epic:** Admin web-building. Absorbs the deferred **2.5 instanced-zone
authoring** item and the 1b follow-up "zone create/delete + zone-config".
**Predecessor specs:** `2026-07-22-admin-builder-page-1b-design.md`,
`2026-07-23-item-authoring-2-design.md`, `2026-07-23-mob-authoring-3-design.md`

## Goal

Create, delete, rename, and configure **zones** from the `/build` admin page,
without hand-editing `zone-config.yaml` or logging into the game.

Zone lifecycle is the last structural gap in the builder: 1b edits rooms
*within* existing zones, but a new zone still requires the in-game
`build zone "The Arctic"` command, and every zone-config field
(biome, region, music, instancing) is hand-edited YAML.

Scope decisions (user-confirmed):

- **Full zone-config parity, including the instanced block.** This folds in
  deferred sub-project 2.5 — `Instanced`, `EntryRoom`, `PortalDuration`,
  `DeathPolicy`, `AllowRecall` are ZoneConfig fields, so building a separate
  editor panel for them later would duplicate this one.
- **Delete refuses unless the zone is empty**, reporting exactly what blocks
  it. No cascade delete, no archive tree.
- **Rename is real** (moves all 10 directories, rewrites `zone:` everywhere)
  but is **phased separately** — see Phasing below.
- **Fog-of-war is NOT migrated on rename.** Players silently re-explore.
- **Rename refuses while any player is in the zone.**

## Phasing

**Phase 1 — create / delete / config.** Independently useful and low-risk.
Ships on its own.

**Phase 2 — rename.** Roughly half the risk of the sub-project. Built and
shipped only after Phase 1 lands, so a problem in rename cannot leave the
zone editor half-landed.

Each phase gets its own implementation plan.

## Background: what a zone actually owns

A zone is not one directory. `stillwater` occupies **ten**:

```
rooms/<zone>/            rooms.instances/<zone>/
mobs/<zone>/             mobs.instances/<zone>/
dialogue/<zone>/         behaviors/<zone>/
schedules/<zone>/        caravans/<zone>/
foragers/<zone>/         shops/<zone>/
```

Folder names are the **sanitized** zone name (`ZoneNameSanitize` /
`ZoneToFolder` — underscores, not hyphens; see CLAUDE.md "Data File Naming").
The display name lives in `zone-config.yaml`'s `name:` and on every room's
and mob's `zone:` field.

Two of those ten (`shops/`, and the two `.instances/` trees) are **living
state**, not authored content — see the instance-save and shop-persistence
SOPs. Delete removes them along with the rest; that is correct, because the
zone is going away.

## 1. Core seam — `internal/rooms`

Existing and reused unchanged (all verified present):

- `CreateZone(zoneName) (roomId, error)` (`roommanager.go:728`) — makes the
  folder, writes `zone-config.yaml`, makes the `rooms.instances/` folder, and
  creates one starter room. Already backs in-game `build zone`.
- `SaveZoneConfig` (`save_and_load.go:411`) — persists a `ZoneConfig`.
- `GetZoneConfig` (`:602`), `GetAllZoneNames` (`:240`).
- `ValidateZoneName` (`:636`) — character-set check (letters, numbers,
  spaces, underscores). **Reuse it; do not write a second name validator.**
  Note it returns nil for the empty string, so an emptiness check is the
  caller's job.
- `ZoneNameSanitize` (`:620`) / `ZoneToFolder` (`:630`) — the display-name →
  folder mapping.

Net-new:

- `DeleteZone(zoneName) error` — guarded; see §3.
- `ZoneDeletionBlockers(zoneName) []ZoneBlocker` — the reference scan, split
  out so the GMCP layer can report without attempting a delete.
- `ValidateZoneConfig(cfg) error` — **field** validation (biome, death policy,
  entry room, durations) shared by the GMCP handler and any future caller.
  Name validation stays in `ValidateZoneName`; this does not duplicate it.
- *(Phase 2)* `RenameZone(old, new) error` and `ZoneRenameBlockers(...)`.

A **folder-collision guard** is added to `CreateZone` as part of Phase 1.
Today it checks only whether the display name is a known zone, so
`Amber_Valley` alongside an existing `Amber Valley` would `os.Mkdir` onto the
same folder and fail with a raw filesystem error. Same check rename needs
(§4.1), so it is written once and used by both.

`ZoneConfig.Validate()` already defaults `DeathPolicy` and `PortalDuration`;
`ValidateZoneConfig` is the stricter authoring-time check (see §5).

## 2. GMCP protocol — `Build.Zone.*`

Admin-gated exactly as `Build.Room.*` / `Build.Item.*` / `Build.Mob.*`: a
non-admin gets **no response at all** (the `requireAdmin` `break`), not an
error. All mutating verbs travel as a `GMCPBuildOp` to MainWorker, because
zone mutation touches `roomManager` and the mapper caches.

Client → server:

| Verb | Payload | Notes |
|------|---------|-------|
| `Build.Zone.List` | — | zone names + room counts + instanced flag |
| `Build.Zone.Get` | `{zone}` | full config detail + enums |
| `Build.Zone.Create` | `{name}` | delegates to `CreateZone` |
| `Build.Zone.Update` | full config, minus name | validate → `SaveZoneConfig` |
| `Build.Zone.Delete` | `{zone}` | guarded; returns blockers on refusal |
| `Build.Zone.Rename` | `{zone, newName}` | **Phase 2** |

Server → client: `Build.Zones` (list), `Build.Zone` (detail + enums),
and the shared `BuildResult` (`{ok, error, refs}`) already used by item and
mob delete.

Handlers sit behind a `zoneDeps` struct — `load`, `save`, `create`, `del`,
`blockers`, `zoneExists`, `roomsInZone`, `playersInZone` — so every handler
is unit-testable against a fake world with no filesystem, matching
`itemDeps`/`mobDeps`.

## 3. Delete — refuse unless empty

`Build.Zone.Delete` refuses and reports, in the same `refs` shape item and
mob delete already return. Blockers scanned, each reported with a `kind` and
a human-readable id:

1. **`room`** — any room in the zone other than its root (`ZoneConfig.RoomId`).
2. **`mob`** — any template under `mobs/<zone>/`.
3. **`content`** — any file under `dialogue/`, `behaviors/`, `schedules/`,
   `caravans/`, `foragers/` for the zone.
4. **`inbound-exit`** — any room in **another** zone with an exit whose
   target room lives in this zone. This is the one a human would miss, and
   it is exactly what leaves the mapper with a dangling edge.
5. **`player`** — any player currently in the zone.

Shops and the two `.instances/` trees are deliberately **not** blockers —
they are regenerable living state, not authored work.

A zone with only its root room and none of the above is deletable: remove
the ten directories (those that exist), drop it from `roomManager.zones`,
and clear the mapper cache.

**The root room goes with the zone.** Deleting a zone deletes its root room;
that room is not orphan-able, since a room's file path is derived from its
zone.

## 4. Rename (Phase 2)

Guard: refuses if any player is in the zone. Mobs are not a blocker — they
respawn from templates.

Executed on MainWorker as: **validate → dry-run → move → rewrite → rekey.**

1. **Validate** — new name ≥2 chars, passes `ValidateZoneName`, is not
   already a zone, and its **sanitized folder name** does not collide with an
   existing zone's folder. That last check is not paranoia:
   `ZoneNameSanitize` only lowercases and turns spaces into underscores, so
   `Amber Valley`, `amber valley`, and `Amber_Valley` all map to
   `amber_valley`. Renaming to a display name that is "different" but
   sanitizes onto a live zone's folder would collide on disk. **Create must
   apply the identical check** — the same hole exists there today.
2. **Dry run** — confirm every one of the ten target paths is free and the
   source is writable. Build a move manifest. Abort with no changes on any
   problem.
3. **Move** — rename the directories that exist, recording each success.
4. **Rewrite** — `name:` in `zone-config.yaml`; `zone:` on every room file in
   the zone; `zone:` on every mob file in the zone; any caravan / forager /
   schedule file naming the zone.
5. **Rekey** — `roomManager.zones` under the new key, `mapper.ClearCache()`.

On failure mid-flight, reverse the recorded manifest best-effort and report
what was and was not undone. A rename that cannot be fully reversed must say
so loudly rather than reporting success.

**Fog-of-war is not migrated.** `Character.VisitedRooms` is keyed by zone
display name and lives in every player save, including offline players.
Rewriting player data is out of scope by decision; the visible effect is that
players re-explore the renamed zone on the web map. This is an intentional
divergence, recorded here so it is not later mistaken for a bug.

## 5. Config editing & validation

Every `ZoneConfig` field is editable except `Name`, which is rename's job and
renders read-only with a pointer to the rename action.

| Field | Control | Validation |
|-------|---------|------------|
| `RoomId` (root) | room picker, zone-scoped | must be a room in this zone |
| `DefaultBiome` | dropdown | server-supplied biome list |
| `Region` | text + datalist of existing regions | free-form allowed |
| `MusicFile` | text | — |
| `IdleMessages` | list editor | — |
| `Mutators` | list editor | known mutator ids |
| `NonCartesian` | checkbox + warning | disables Cartesian enforcement |
| `DefaultPlane` | number | — |
| `Instanced` | checkbox, reveals block below | — |
| `EntryRoom` | room picker, zone-scoped | must be a room in this zone |
| `PortalDuration` | text | parseable duration |
| `DeathPolicy` | dropdown | `rejoin` \| `ejected` |
| `AllowRecall` | checkbox | — |

Dangerous references are picked from server-supplied lists rather than typed,
the same rule the mob editor follows — a typo must not be able to brick the
next boot.

`NonCartesian` gets an explicit in-panel warning: it marks the zone's plane
non-Euclidean and skips collision/reciprocity enforcement. It is the one
field here that can silently degrade world integrity.

## 6. Web UI

A fourth mode button — **Zones** — beside Rooms / Items / Mobs in the `/build`
toolbar, following the existing `Builder.<X>Panel` module shape
(`zones.js`, matching `items.js` / `mobs.js`).

Left: zone list with room count and an instanced badge. Right: the config
form, with the instanced block collapsed until `Instanced` is ticked.
Create takes a name. Delete and Rename surface their blocker lists in-panel,
listing each blocker rather than a bare "cannot delete".

## 7. Testing & verification gate

**Unit** (`modules/gmcp`, fake world): delete refuses per blocker kind and
reports each; delete succeeds on a clean zone; config round-trips through
`Update`; invalid biome / death policy / non-zone entry room rejected;
non-admin `Build.Zone.*` silently refused. *(Phase 2)* rename refuses with a
player present, refuses on folder-name collision, aborts cleanly when the
dry run fails.

**Integration** (real filesystem, temp zone): create → populate → attempt
delete (blocked, correct blockers) → empty it → delete (all ten directories
gone). *(Phase 2)* create → populate → rename → assert directories moved,
`zone:` rewritten on every room and mob, `roomManager.zones` rekeyed.

**Boot + cartcheck**: boot clean under `MapConsistencyEnforce: panic` after a
delete and after a rename; `cartcheck` clean world-wide.

**E2E**: headless WS driver, reusing the `mob_e2e.mjs` TEXTMASK login-bridge
pattern from sub-project 3.

**Browser gate**: as with 1b/2/3, a human must drive the real page — form
usability, blocker messaging, and the instanced-block reveal cannot be
verified headlessly.

The content adversarial-playtest gate (CLAUDE.md) does **not** apply: this
authors world structure, not player-facing content. The browser gate above
replaces it.

## Out of scope

- **Migrating player fog-of-war on rename** (decided above).
- **Cascade delete** and any archive/soft-delete tree.
- **Moving rooms between zones** — `rooms.MoveToZone` exists and is
  unexposed; a separate follow-up.
- **Zone-level spawn lists** — sub-project 4.
- **Region authoring** — regions are a free-form string here; a region
  registry is its own piece if ever wanted.
- **The in-game `build zone` command** — unchanged, keeps working.
