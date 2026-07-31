# Moderation Context

## Purpose

`internal/moderation` holds the two pieces of player-moderation state that must
survive a restart: the **petition queue** (player→staff reports) and the **ban
lists** (account and IP).

Like `guilds/` and `shops/`, this is **living state**. It is `.gitignore`d,
kept on the production droplet, and must never be wiped by the instance-save
smoke-test ritual.

## Files

- **moderation.go** — data-directory resolution, `LoadDataFiles`, test hooks.
- **petitions.go** — the petition queue.
- **bans.go** — account and IP bans.

Both stores live under `_datafiles/world/<world>/moderation/` as
`petitions.yaml` and `bans.yaml`.

## Petitions

```go
type Petition struct { /* id, reporter, roomId, zone, message,
                          resolved-by, note, timestamps */ }

func Add(reporter string, roomId int, zone, message string) (Petition, error)
func ListOpen() []Petition
func ListAll() []Petition
func Get(id int) (Petition, bool)
func Resolve(id int, by, note string) error
```

A petition captures *where* the player was as well as what they said, so staff
can go look. Resolution is non-destructive — resolved petitions stay in the
file with the resolver and note attached.

Config: `GamePlay.PetitionCooldownRounds` (rate limit per player) and
`GamePlay.PetitionMaxLen`.

## Bans

```go
type AccountBan struct { /* username, reason, by, when */ }
type IPBan struct      { /* ip, reason, by, when */ }

func BanAccount(username, reason, by string) error
func Unban(username string) error
func IsAccountBanned(username string) (reason string, banned bool)

func BanIP(ip, reason, by string) error
func UnbanIP(ip string) error
func IsIPBanned(host string) (reason string, banned bool)
```

Account keys are normalised through `normAccountKey`, so bans are
case-insensitive and survive a player retyping their name differently.

**Enforcement happens in `FinalizeLoginOrCreate`**
(`internal/inputhandlers/login.go`) — this package only stores and answers
questions. A new entry point that creates sessions must call the `Is*Banned`
checks itself.

## Public API

```go
func LoadDataFiles()
func SetDataDirForTest(dir string) func()
```

## Gotchas

- **A malformed file logs and skips at boot rather than panicking**, mirroring
  the guilds loader. Authored content panics; runtime-generated state must not.
  The cost is that a corrupt `bans.yaml` silently unbans everyone — check the
  boot log.
- **`IsIPBanned` takes a host, not a `host:port`.** Split the remote address
  before calling.
- **Both stores are whole-file rewrites** (`saveBansLocked`,
  `savePetitionsLocked`). They are small, but do not call them in a loop.
- **Bans are not checked on an existing session.** Banning a connected player
  stops them reconnecting; use `boot` to remove them now.

## Related commands

`petition` (player), `petitions` / `boot` / `ban` / `unban` (admin), plus the
globally-targetable `mute` / `deafen`.

## Dependencies

`configs`, `mudlog`, `util`, plus YAML and the filesystem. Deliberately no
dependency on `users` — the login handler passes strings in.

## Consumers

`internal/inputhandlers` (login-time ban rejection) and
`internal/usercommands` (the moderation command family).
