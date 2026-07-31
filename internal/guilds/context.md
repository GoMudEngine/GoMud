# Guilds Context

## Purpose

`internal/guilds` owns player guilds: membership, ranks, invites, the shared
treasury, and per-guild persistence. Guilds are **social**, not mechanical —
there are no guild-only skills or combat bonuses; the package provides identity,
a chat channel tag, and a shared bank.

State is **living**, written to `_datafiles/world/<world>/guilds/<tag>.yaml` and
kept on the production droplet. It is not instance-save data and must never be
wiped by the smoke-test cleanup.

## Files

- **guilds.go** — `Guild`, `GuildMember`, `GuildRank`, permission predicates,
  and validation.
- **registry.go** — the in-memory index and every mutating operation.
- **persistence.go** — load, save, delete, and the test data-dir override.

## Types

```go
type GuildRank string      // ordered; rankOrder() gives the comparison
type GuildMember struct { /* userId, char name, rank */ }
type Guild struct {
    /* tag, name, LeaderUserId, members, invites, motd,
       treasury gold + items, rank titles, treasury delegation flag */
}
```

## Permissions

```go
func (g *Guild) MemberRank(userId int) (GuildRank, bool)
func (g *Guild) IsMember(userId int) bool
func (g *Guild) IsLeader(userId int) bool
func (g *Guild) CanManage(userId int) bool
func (g *Guild) CanKick(actorId, targetId int) bool
func (g *Guild) CanWithdraw(userId int) bool
func (g *Guild) HasInvite(userId int) bool
func (g *Guild) RankTitle(rank GuildRank) string
```

`CanKick` takes both actor and target because the rule is relative — you cannot
kick someone at or above your own rank. `CanWithdraw` consults the guild's
treasury-delegation flag, not just rank.

## Registry API

Lookup:

```go
func Get(tag string) (*Guild, bool)
func GetByUser(userId int) (*Guild, bool)
func TagForUser(userId int) string
func TagExists(tag string) bool
func NameExists(name string) bool
func All() []*Guild
func GuildWithInvite(userId int) (*Guild, bool)
```

Membership:

```go
func Create(tag, name string, leaderUserId int, leaderName string) (*Guild, error)
func AddMember(tag string, userId int, charName string) error
func RemoveMember(tag string, userId int) error
func SetRank(tag string, userId int, rank GuildRank) error
func TransferLeader(tag string, newLeaderId int) error
func AddInvite(tag string, userId int) error
func RemoveInvite(tag string, userId int) error
func ClearInvites(userId int)
```

Treasury and settings:

```go
func DepositGold(tag string, amount int) error
func WithdrawGold(tag string, amount int) error
func DonateItem(tag string, item items.Item) error
func TakeItem(tag string, idx int) (items.Item, error)
func SetTreasuryDelegated(tag string, delegated bool) error
func SetMotd(tag, text string) error
func SetRankTitle(tag string, rank GuildRank, title string) error
```

Persistence:

```go
func LoadDataFiles()
func Save(g *Guild) error
func Delete(tag string)
func SetDataDirForTest(dir string) func()
```

## Gotchas

- **A malformed guild file logs and skips at boot; it does not panic.** This is
  the opposite of authored content (mobs, quests, rooms), which panics on a bad
  file. Guild YAML is runtime-generated, so one corrupt file must not take the
  server down — but it also means a broken guild disappears quietly. Check the
  boot log if a guild goes missing.
- **`Create` enforces tag and name validity** via `validGuildTag` /
  `validGuildName`, and `SetRankTitle` via `validRankTitle`. Do not construct a
  `Guild` literal and hand it to `Save` — you will bypass all three.
- **A user belongs to at most one guild.** `GetByUser` and `TagForUser` assume
  it; `AddMember` does not verify it for you.
- **`ClearInvites` sweeps every guild**, not just one. It is the "player joined
  somewhere" cleanup, so calling it from a single-guild flow will silently drop
  unrelated pending invites.
- **`TakeItem` is index-based.** The index refers to the treasury slice at the
  moment of the call; two concurrent takes race unless the caller holds the
  guild.

## Dependencies

`items`, `users`, `configs`, `mudlog`, `util`, plus YAML.

## Consumers

`internal/usercommands` (the `guild` command family), the guild chat channel,
and `internal/web` for the guild roster view.
