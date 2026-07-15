# Guild custom rank titles — Design

**Retention #2 (guilds), final sub-project: ranks polish → custom rank titles.**

## Goal
Let a guild leader rename their guild's three ranks (member / officer / leader) to
flavourful per-guild titles, shown in `guild info` and rank-change notices. Purely
cosmetic; no mechanical effect on permissions.

## Scope
- IN: custom titles for the three existing ranks; leader-only set/reset; validation;
  display in `guild info` roster + promote/demote/transfer notices.
- OUT (user did not pick): member cap, founded-date/stats sheet, roster sorting/QoL.
  Those remain future ranks-polish work.

## Model
`Guild` gains:
```go
RankTitles map[GuildRank]string `yaml:"ranktitles,omitempty"`
```
Helper (value-receiver-safe, nil-map-safe):
```go
// RankTitle returns the guild's custom title for rank, or the default rank name.
func (g *Guild) RankTitle(rank GuildRank) string {
    if t, ok := g.RankTitles[rank]; ok && t != "" {
        return t
    }
    return string(rank)
}
```
Unset / empty always falls back to the default (`"member"`/`"officer"`/`"leader"`),
so existing guilds and reset titles render exactly as today.

## Registry op
```go
// SetRankTitle sets (title != "") or clears (title == "") the custom title for rank.
func SetRankTitle(tag string, rank GuildRank, title string) error
```
Mirrors the existing mutators (Get → lock → mutate map (lazy-init) → unlock → Save).
Clearing deletes the map key so `omitempty` keeps the YAML clean.

## Command
`guild title <rank> [name...]` — **leader-only** (`IsLeader`).
- `guild title officer Lieutenant` → sets officer title.
- `guild title officer` (no name) → resets officer to default.
- `<rank>` must resolve (case-insensitively) to one of member / officer / leader; else
  usage error listing the three.
- On success: confirm to the leader; no guild-wide announce (low-noise, matches motd).

## Validation (`validRankTitle`)
Reject with a clear message; applied before mutation:
- length 2–20 runes;
- single line;
- letters, digits, spaces only (`unicode.IsLetter`/`IsDigit`/space) — this **excludes
  colons and semicolons** (YAML-colon parse gotcha + `;` command separator) and ANSI
  `<...>` markup;
- not all-whitespace (TrimSpace non-empty), and stored trimmed.
(Reset path — empty name — skips validation.)

## Display changes (`guild.go`)
- `guildInfo` roster line: use `g.RankTitle(m.Rank)` instead of raw `m.Rank`.
- `guildSetRank` promote/demote notices and `guildTransfer` notice: use the guild's
  title for the new rank (e.g. "You promote X to Lieutenant.").
- The `[TAG]` who-prefix is unchanged (it is the tag, not a rank).

## Persistence / safety
- Titles live in the existing per-guild YAML (`guilds/<tag>.yaml`) — durable living
  state, already excluded from instance-cleanup.
- `RankTitles` is only ever mutated on the single event goroutine under `registryMu`,
  consistent with every other registry op.

## Tests
- `TestRankTitle`: nil map → defaults; custom set → custom; empty string value → default.
- `TestSetRankTitle`: set then Get shows custom; reset (empty) deletes key → default.
- `TestValidRankTitle`: accepts "Lieutenant", "Storm Warden", "R2"; rejects "" via
  caller, "A", 21-char, "Bad: Title", "semi;colon", "<ansi>x</ansi>".
- Command rank-arg parsing covered by a small `parseGuildRank` unit test.

## Files
- `internal/guilds/guilds.go` — field + `RankTitle` + `validRankTitle`.
- `internal/guilds/guilds_test.go` — RankTitle + validRankTitle tests.
- `internal/guilds/registry.go` — `SetRankTitle`.
- `internal/guilds/registry_test.go` — SetRankTitle test.
- `internal/usercommands/guild.go` — `title` dispatcher case, `guildSetTitle`,
  `parseGuildRank`, display swaps.
- `internal/usercommands/guild_test.go` — `parseGuildRank` test.
- `_datafiles/world/{dogmud,default}/templates/help/guild.template` — help line.
- `PATCH_NOTES.md`, `docs/PATH_TO_1.0.md` — docs.
