# Room Instance Load Respects `instance:"skip"` Tag — Design (Fix A)

**Date:** 2026-04-21
**Status:** Approved
**Related memory:** `project_instance_save_exits_corruption.md` (Fix A — remaining half)
**Companion work:** `docs/superpowers/specs/2026-04-21-summons-dont-persist-design.md` (Fix D — landed earlier today)

## Problem

`SaveRoomInstance` in `internal/rooms/save_and_load.go:242` correctly
skips any Room struct field tagged `instance:"skip"`. The tag was
added to structural fields (Title, Description, Exits, Nouns, Zone,
etc.) on 2026-03-03 in commit `10edee33` precisely to prevent
instance saves from overriding template values.

However, `LoadRoomInstance` at `save_and_load.go:116-121` does not
respect the tag. It uses raw `yaml.Unmarshal` onto the
template-loaded Room, which only reads `yaml:` tags — the
`instance:"skip"` annotation is invisible to the YAML library. Every
field serialized into an existing instance file therefore overwrites
the template value on load, regardless of whether the tag says to
skip it.

## Concrete symptom

`_datafiles/world/dogmud/rooms.instances/thornwall_city/472.yaml` is
a pre-`10edee33` file that still contains:

```yaml
description: A low-ceilinged tavern...
exits:
  east:
    roomid: 475
  north:
    roomid: 469
nouns:
  bar: ...
```

All three of those fields carry `instance:"skip"` on the Room struct
today. The template's real exits are `north: 469, west: 481,
south: 484`. On boot, `LoadRoomInstance` loads the template, then
overlays this file — destroying `west` and `south` and injecting the
spurious `east: 475` (a pre-fix temp-portal artifact). Result: Sable
access intermittently breaks.

This same asymmetry corrupts every stale instance file that predates
the March fix, across every skip-tagged field.

## Design rule

> **`LoadRoomInstance` must respect `instance:"skip"` symmetrically
> with `SaveRoomInstance`. A field tagged `instance:"skip"` is
> template-owned and must come from the template on every load,
> regardless of what any instance file says.**

## Code changes

### `internal/rooms/save_and_load.go` — modify `LoadRoomInstance`

Current:

```go
if bytes, err := os.ReadFile(filepath); err == nil {
    // Unmarshal onto the default template data, overwriting any set fields in the instance save file
    if err := yaml.Unmarshal(bytes, room); err != nil {
        mudlog.Warn("LoadRoom", "roomId", roomId, "filepath", filepath, "error", err)
    }
}
```

Proposed:

```go
if bytes, err := os.ReadFile(filepath); err == nil {
    if err := yaml.Unmarshal(bytes, room); err != nil {
        mudlog.Warn("LoadRoom", "roomId", roomId, "filepath", filepath, "error", err)
    }
    // yaml.Unmarshal only honors `yaml:` tags; the `instance:"skip"`
    // annotation is invisible to it. Load a fresh template copy and
    // copy every skip-tagged field back onto the room, so
    // template-owned fields (title/description/exits/nouns/zone/etc.)
    // cannot be corrupted by stale data in pre-fix instance files.
    if freshTemplate := LoadRoomTemplate(roomId); freshTemplate != nil {
        restoreSkipTaggedFields(room, freshTemplate)
    }
}
```

### New helper

One reflection helper mirrors the tag-check loop in
`SaveRoomInstance`. Add alongside `LoadRoomInstance` in the same
file:

```go
// restoreSkipTaggedFields copies every exported Room field tagged
// `instance:"skip"` from src onto dst. Used after an instance-file
// overlay unmarshal to ensure template-owned fields are not
// corrupted by stale data in pre-fix instance files.
func restoreSkipTaggedFields(dst, src *Room) {
    srcVal := reflect.ValueOf(*src)
    dstVal := reflect.ValueOf(dst).Elem()
    t := srcVal.Type()
    for i := 0; i < t.NumField(); i++ {
        field := t.Field(i)
        if field.PkgPath != "" {
            continue
        }
        if field.Tag.Get("instance") != "skip" {
            continue
        }
        dstVal.Field(i).Set(srcVal.Field(i))
    }
}
```

**Why reload the template instead of snapshotting the existing
`room` before unmarshal?** A snapshot via `reflect.Value.Interface()`
on map/slice fields captures the header, not a copy; if
`yaml.Unmarshal` happens to mutate the existing map in place rather
than replacing it, the snapshot is invalidated. Reloading via
`LoadRoomTemplate` is trivially cheap (single YAML file read on disk
cache) and produces a fresh Room with independent map backing
storage. Safer and simpler.

### Import additions

`reflect` should already be imported in `save_and_load.go` (used by
`SaveRoomInstance`). Confirm — add if missing.

## What this fixes without any further action

- All pre-`10edee33` instance files stop poisoning reloads.
  Skip-tagged fields come from the template, period.
- Room 472's exits are restored at next boot.
- The 3 currently-corrupt files will **self-clean** on any subsequent
  `SaveRoomInstance` for those rooms — the save's `instanceSaveData`
  map excludes skip-tagged fields, and the existing file-deletion at
  `save_and_load.go:295` (`if len(instanceSaveData) == 0 { os.Remove(...) }`)
  kicks in when only skip-tagged fields diverge.

## Edge cases consciously accepted

- **A zone with no runtime mutations keeps a stale file indefinitely.**
  The self-clean only fires when `SaveRoomInstance` runs. Rooms that
  never receive dynamic state changes (items, gold, containers) keep
  their stale file forever — but those fields are now ignored on
  load, so the file is inert. No game-visible symptom.
- **User/ops cleanup remains the escape hatch.** Prod wipes
  `rooms.instances/` on every patch pull. Local dev can nuke
  manually. Both flows already exist and are undisturbed.

## Out of scope

- **Separate tempExits runtime field (Fix B from the original
  memory).** The load-side fix makes Fix B unnecessary — temp exits
  can still live in the same `Exits` map because the save side
  ignores the tag anyway, and the load side now restores the
  template's exits regardless of what any file says. If future
  requirements need admin-created runtime exits to persist across
  restart, we'd revisit.
- **Per-entry map diff (Fix C from the original memory).** Was
  proposed for `nouns`, `items`, etc. but only `exits` had the
  specific temp-portal pollution problem, and all three now have the
  `instance:"skip"` tag so per-entry diff is overkill.
- **Boot-time cleanup of existing corrupt files.** User's call — prod
  nukes on patch pull, local can nuke manually.

## Testing

### Unit tests (Go)

Location: `internal/rooms/save_and_load_test.go` (create if
absent; otherwise append).

1. **`TestLoadRoomInstance_RespectsSkipTag`** — write a fake
   instance file containing a corrupt `title:` or `exits:` field for
   a known room; call `LoadRoomInstance`; assert the loaded room's
   title/exits come from the template, not the file.
2. **`TestLoadRoomInstance_AppliesNonSkipFields`** — write an instance
   file containing a `gold:` or `items:` field (both are NOT
   skip-tagged); assert those fields land on the loaded room. Control
   case — proves the tag scope is narrow.
3. **`TestRestoreSkipTaggedFields`** — unit-level: build two Rooms
   (one representing the "current" post-unmarshal state with corrupt
   Exits/Title, one representing a fresh template); call
   `restoreSkipTaggedFields(current, template)`; assert skip-tagged
   fields now match template, non-skip fields keep their current
   values.

### Smoke test (local server)

1. With the fix applied and the known-corrupt `thornwall_city/472.yaml`
   present on disk: boot the server, enter room 472, confirm
   exits show `north, south, west` (not `north, east`).
2. Move through room 472 and interact (drop an item, pick it up)
   to trigger `SaveRoomInstance`. Confirm the file is either
   deleted (if no other runtime diff exists) or rewritten without
   the exits/description/nouns keys.
