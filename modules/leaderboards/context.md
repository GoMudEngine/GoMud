# Leaderboards Module Context

## Purpose

`modules/leaderboards` maintains ranked top-N tables of players and serves them
to both the `leaderboard` command and the web dashboard.

Each board is a fixed-size list rebuilt by offering every character to it, so
adding a board means adding one metric — not a new storage mechanism.

## Types

```go
type leaderboardEntry struct { /* userId, name, value */ }
type leaderboardData struct  { /* one board: entries + size */ }
type LeaderboardModule struct
```

## API

```go
func (l *leaderboardData) Reset(size int)
func (l *leaderboardData) Consider(userId int, char characters.Character, val int)

func (l *LeaderboardModule) Update()
func (l *LeaderboardModule) Reset(maxSize int)
func (l *LeaderboardModule) RefreshConfig()
func (l *LeaderboardModule) getCurrentLeaderboards() []leaderboardData
func (l *LeaderboardModule) loadLBs()
func (l *LeaderboardModule) saveLBs()
```

`Consider` is the whole insertion algorithm: offer a candidate, and the board
keeps it only if it beats the current floor. Rebuilding is therefore
`Reset` + a `Consider` per character.

## Gotchas

- **Boards are rebuilt, not incrementally maintained.** `Update` runs on a
  round handler; a value that changes between updates is not reflected until
  the next rebuild.
- **`RefreshConfig` must be called after a config change** or the board keeps
  its old size.
- **Boards persist** via `loadLBs`/`saveLBs`, so a restart does not blank them —
  but a board whose metric was removed still holds stale entries until the next
  rebuild.
- **Entries store a name snapshot.** A renamed character shows its old name
  until it is reconsidered.
- **`Consider` takes `characters.Character` by value.** It is a large struct;
  do not add per-entry work inside the loop lightly.

## Dependencies

`plugins`, `events`, `characters`, `users`, `configs`.

## Consumers

Registered as a plugin; the `leaderboard` command and the web dashboard.
