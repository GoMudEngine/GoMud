# Devtools Context

## Purpose

`internal/devtools` is world-building automation for authors: generate a grid of
rooms, link two zones together, and check a zone's geometry — driven either from
Go or over a JSON API so external tooling can call in.

It writes room YAML directly to disk. It is an authoring tool, not a runtime
system.

## Files

- **gridgen.go** — `GenerateGrid` and the room-writing helpers.
- **linkzones.go** — `LinkRooms`.
- **consistency.go** — `CheckZoneConsistency`.
- **api.go** — the JSON request/response wrapper.

## API

```go
func GenerateGrid(zoneName string, width, height int) (firstId, lastId int, err error)
func LinkRooms(zoneA string, roomIdA int, direction string, zoneB string, roomIdB int) error
func CheckZoneConsistency(zoneName string) (report string, issueCount int, err error)

type APIRequest struct  { /* op + args */ }
type APIResponse struct { /* result or error */ }
func HandleJSON(input string) string
```

`GenerateGrid` returns the id range it allocated — record it, because that is
the only report of which ids are now taken.

## Gotchas

- **These functions write files.** Run them against a checkout you are willing
  to diff, and inspect the result before committing. There is no dry-run.
- **`GenerateGrid` allocates ids itself.** Cross-check against
  `python tools/id_inventory.py` before and after; the script is the project's
  authority on id ranges, and two authors generating grids in parallel will
  collide.
- **`LinkRooms` creates the link you asked for.** It does not verify the result
  stays Cartesian-consistent — run `CheckZoneConsistency`, or the in-game
  `cartcheck`, afterwards.
- **`CheckZoneConsistency` reads from disk**, so it reports on the authored
  files, not on the loaded world. The running server's `cartcheck` command is
  the runtime equivalent and can disagree if instance data differs.
- **`HandleJSON` returns errors as JSON, not as Go errors.** A caller that only
  checks for a non-empty string will treat a failure as success.

## Dependencies

`rooms`, `exit`, `configs`, `util`, plus direct filesystem access.

## Consumers

The admin web building tools and external authoring scripts.
