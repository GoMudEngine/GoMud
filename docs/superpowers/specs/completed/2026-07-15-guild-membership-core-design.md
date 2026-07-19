# Guild Membership Core

**Date:** 2026-07-15
**Status:** Design (approved decisions; pending spec review)
**Roadmap:** `docs/PATH_TO_1.0.md` §3 Retention/stickiness — the second item (social guilds).
**Arc:** Social guilds, decomposed into sub-projects. **This spec = sub-project #1, Membership
Core** (create/join/ranks/roster/MOTD/persistence). Later sub-projects: guild chat, guild
treasury/vault, guild perks. **Dropped** (PvE game): territory/zone-control, upkeep-or-disband,
open applications.

---

## 1. Goal

A persistent, social organization players join to find community and a reason to log back in:
found a guild (for a gold fee), invite friends, organize by rank, see the roster and a
message-of-the-day. This slice is the durable membership foundation everything else builds on.

---

## 2. Approved decisions

1. **Social guilds** — no territory, no upkeep, no PvP. Invite-only joining.
2. **Ranks: member → officer → leader.** Leader: invite/kick/promote/demote/transfer/disband/
   set-MOTD. Officer: invite/kick members, set MOTD. Member: none.
3. **Account-level membership** keyed by `UserId` (a user is in at most one guild; all their
   alts share it — matches the party model). The member's character name is stored for display.
4. **Founding fee** `GuildFoundingCost` (default **5000g**) paid from the founder's **bank**.
5. **MOTD** included: `guild motd <text>` (officer+), shown in `guild info` and on member login.
6. **Persistence:** durable per-guild YAML (the `internal/shops` pattern), a cached registry,
   saved on mutation. Registry is authoritative; **no `Character.GuildTag` field** (avoids
   drift + a characters↔guilds import cycle — display reads the registry from the command layer).

---

## 3. Package + persistence — `internal/guilds/`

A fresh `guilds` package. The unused `internal/clans/clans.go` stub (referenced nowhere) is
**removed** (superseded).

- **Storage:** one file per guild at `_datafiles/world/dogmud/guilds/<tag>.yaml` (tag
  lower-cased for the filename). Mirror `internal/shops/persistence.go`: `Save(g)` writes the
  file, `LoadDataFiles()` at boot walks the dir into the registry, `Delete(tag)` removes the
  file. **Not** an instance-save — this is living state, excluded from the instance-cleanup SOP
  (like `shops/`). Add to the "do NOT wipe" list in CLAUDE.md.
- **Registry** (package-level, guarded by a mutex — mutated from command handlers on the main
  loop, but a mutex keeps it safe and matches shops):
  - `byTag map[string]*Guild`, `byUser map[int]string` (userId → tag).
  - `Get(tag string) (*Guild, bool)`, `GetByUser(userId int) (*Guild, bool)`,
    `TagForUser(userId int) string` (empty if none), `All() []*Guild`, `TagExists`,
    `NameExists`.
  - `Create(tag, name string, leaderUserId int, leaderName string) (*Guild, error)` — validates
    uniqueness + tag/name rules, builds the guild with the leader as sole member, saves.
  - Membership mutations (`AddMember`, `RemoveMember`, `SetRank`, `TransferLeader`, invites) all
    update both maps + the guild's `Members`/`PendingInvites` and persist.

---

## 4. Data model

```go
type GuildRank string
const (
    RankMember  GuildRank = "member"
    RankOfficer GuildRank = "officer"
    RankLeader  GuildRank = "leader"
)

type GuildMember struct {
    UserId        int       `yaml:"userid"`
    CharacterName string    `yaml:"charactername"` // display name at join (refreshed on rank ops)
    Rank          GuildRank `yaml:"rank"`
    Joined        time.Time `yaml:"joined"`
}

type Guild struct {
    Tag            string        `yaml:"tag"`   // 2-4 alphanumeric, unique (case-insensitive); UPPERCASE for display
    Name           string        `yaml:"name"`  // 3-40 chars, unique (case-insensitive)
    LeaderUserId   int           `yaml:"leaderuserid"`
    Members        []GuildMember `yaml:"members"`
    PendingInvites []int         `yaml:"pendinginvites,omitempty"` // userIds with an outstanding invite
    Motd           string        `yaml:"motd,omitempty"`
    Created        time.Time     `yaml:"created"`
}
```

Helpers on `*Guild`: `MemberRank(userId) (GuildRank, bool)`, `IsMember(userId)`,
`HasInvite(userId)`, `CanManage(userId) bool` (officer+), `IsLeader(userId)`.

---

## 5. Rules / validation

- **Tag:** 2–4 chars, `[A-Za-z0-9]` only, unique case-insensitively; stored/displayed uppercase,
  filename lowercased.
- **Name:** 3–40 chars, unique case-insensitively; printable, no leading/trailing space.
- A user already in a guild can't create or accept into another (leave/disband first).
- Founding requires `Bank >= GuildFoundingCost`; the fee is deducted on success.

---

## 6. Commands — `guild <subcommand>` (`internal/usercommands/guild.go`)

Registered `` `guild`: {Guild, true, true, false} `` + help template. Subcommands:

- **`guild`** / **`guild info`** — your guild's name/tag, MOTD, and roster grouped by rank
  (leader, officers, members) with each member's name + joined; total members. Not in a guild →
  a hint to `guild create` or get invited.
- **`guild create <TAG> <Name…>`** — validate not-already-in-guild, tag/name rules + uniqueness,
  `Bank >= GuildFoundingCost`; deduct fee (bank), create the guild with the caller as leader,
  announce. Interactive `Yes/No` confirm of the fee (StartPrompt), mirroring other spend commands.
- **`guild invite <player>`** (officer+) — target must be an online real player, not already in a
  guild, not already invited; add to `PendingInvites`, notify the target ("X has invited you to
  <Name>; `guild accept` / `guild decline`").
- **`guild accept`** — the caller has a pending invite (search guilds for one); not already in a
  guild; move them from invites to `Members` as `member`, announce to the guild.
- **`guild decline`** — remove the caller's pending invite(s).
- **`guild leave`** — remove the caller. **Leader guard:** a leader with other members must
  `guild transfer` or `guild disband` first (rejected with guidance); a sole-member leader's
  leave = disband (Yes/No confirm).
- **`guild kick <player>`** (officer+) — target is a member of the caller's guild with rank
  **strictly below** the caller's (can't kick peers/superiors; officers can't kick officers or
  the leader); remove + notify.
- **`guild promote <player>`** / **`guild demote <player>`** (leader) — move a member between
  member↔officer (not to/from leader; leadership changes via `transfer`).
- **`guild transfer <player>`** (leader, Yes/No confirm) — the target member becomes leader, the
  caller becomes officer.
- **`guild disband`** (leader, Yes/No confirm) — notify all members, clear their membership,
  delete the guild file.
- **`guild motd <text>`** (officer+) — set/clear the MOTD.
- **`guild list`** — all guilds: tag, name, member count, leader name (paged/simple list).

Every management subcommand checks the caller's rank via the guild helpers and emits a clear
refusal on insufficient permission. All mutations persist via the registry.

---

## 7. MOTD on login — `internal/hooks/`

On player spawn (reuse the existing `PlayerSpawn` hook path, e.g. alongside
`PlayerSpawn_HandleJoin.go`): if `guilds.GetByUser(userId)` has a non-empty `Motd`, send it to
the player (ANSI-styled, e.g. `<ansi fg="cyan">[<Name>] <motd></ansi>`). No emoji.

---

## 8. Config

`internal/configs/config.balance.go` + validator: `GuildFoundingCost` (ConfigInt, default
**5000**; validator `if <= 0 { = 5000 }`).

---

## 9. Display integration

- **`who`** shows a guild tag next to guilded players: `[TAG] Name`. The `who` command imports
  `guilds` and calls `TagForUser(userId)`; empty → no prefix. (Kept in the command layer to
  avoid a characters↔guilds cycle.)
- Deeper prompt/formattedname integration is a follow-up (not this slice).

---

## 10. Boot wiring

Add `guilds.LoadDataFiles()` to `main.go` in the data-load sequence (after users are loadable /
alongside the other loaders); a malformed guild file logs + skips (don't panic the server over
one corrupt guild — unlike authored content, guild files are runtime-generated). Log a
`guilds.LoadDataFiles() loadedCount=…` line.

---

## 11. Testing

- **Registry (pure-ish, temp dir or in-memory):** create enforces tag/name rules + uniqueness;
  add/remove member updates both maps; rank changes; transfer swaps leader/officer; delete
  removes from maps. `TagForUser`/`GetByUser` correctness.
- **Validation helpers:** `validGuildTag`, `validGuildName` (length, charset, boundaries).
- **Permission helpers:** `CanManage`, `IsLeader`, kick-rank rule (can't kick ≥ own rank).
- **Founding fee:** create with `Bank < cost` rejected, nothing mutated; with enough, bank
  debited once and guild created.
- Command handlers are integration (boot/manual); keep the logic in tested registry/helper
  functions so the handlers stay thin.
- Full suite green + boot clean.

---

## 12. Out of scope (later sub-projects / dropped)

- **Guild chat** (next sub-project — reuse `internal/channels`).
- **Guild treasury / item vault / donations** (sub-project after chat).
- **Guild perks / guild leaderboard / guild achievements** (ties into the achievements system).
- **Territory/zone-control, upkeep-or-disband, open applications** — dropped (PvE, social focus).
- Deeper name/prompt integration; guild ranks with custom titles; guild alliances.

---

## 13. Files touched

- `internal/guilds/guilds.go` (types + helpers), `registry.go` (registry + Create/mutations),
  `persistence.go` (Save/LoadDataFiles/Delete), `*_test.go`.
- Remove `internal/clans/clans.go` (superseded stub).
- `internal/usercommands/guild.go` (+ help template ×2) + `usercommands.go` registration +
  `internal/actions/divergences.go` allowlist.
- `internal/usercommands/who.go` (guild-tag prefix).
- `internal/hooks/` — guild-MOTD-on-login.
- `internal/configs/config.balance.go` + `config.balance.misc.go` — `GuildFoundingCost`.
- `main.go` — `guilds.LoadDataFiles()`.
- `CLAUDE.md` — note `guilds/` is durable state (do-not-wipe), like `shops/`.
- `PATCH_NOTES.md`, `docs/PATH_TO_1.0.md` (mark guilds membership-core done; note remaining
  sub-projects).
