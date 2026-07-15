# Guild custom rank titles Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development or executing-plans. Steps use checkbox (`- [ ]`).

**Goal:** A guild leader can rename their guild's three ranks; titles show in `guild info` and rank-change notices. Cosmetic only.

**Architecture:** `RankTitles map[GuildRank]string` on `Guild` + `RankTitle` helper + `validRankTitle` + registry `SetRankTitle` + a `title` subcommand. Display swaps raw rank for `g.RankTitle(rank)`.

**Spec:** `docs/superpowers/specs/2026-07-15-guild-rank-titles-design.md`

---

## Task 1: Model field + `RankTitle` + `validRankTitle`

**Files:** `internal/guilds/guilds.go`, `internal/guilds/guilds_test.go`

- [ ] **Step 1: Failing tests** — add to `guilds_test.go`:
```go
func TestRankTitle(t *testing.T) {
	g := &Guild{}
	if g.RankTitle(RankOfficer) != "officer" {
		t.Errorf("nil map should fall back to default")
	}
	g.RankTitles = map[GuildRank]string{RankOfficer: "Lieutenant", RankMember: ""}
	if g.RankTitle(RankOfficer) != "Lieutenant" {
		t.Errorf("custom title not returned")
	}
	if g.RankTitle(RankMember) != "member" {
		t.Errorf("empty-string title should fall back to default")
	}
}

func TestValidRankTitle(t *testing.T) {
	good := []string{"Lieutenant", "Storm Warden", "R2"}
	for _, s := range good {
		if err := validRankTitle(s); err != nil {
			t.Errorf("%q should be valid: %v", s, err)
		}
	}
	bad := []string{"A", "this title is far too long to accept", "Bad: Title", "semi;colon", "<ansi>x</ansi>", "   "}
	for _, s := range bad {
		if err := validRankTitle(s); err == nil {
			t.Errorf("%q should be invalid", s)
		}
	}
}
```

- [ ] **Step 2: Run → FAIL** — `go test ./internal/guilds/ -run 'TestRankTitle|TestValidRankTitle'`

- [ ] **Step 3: Implement** in `guilds.go` — add field to `Guild` struct (after `TreasuryDelegated`):
```go
	RankTitles map[GuildRank]string `yaml:"ranktitles,omitempty"` // per-guild custom rank names
```
Add helper + validator (import `unicode` + `unicode/utf8`):
```go
// RankTitle returns the guild's custom title for rank, or the default rank name.
func (g *Guild) RankTitle(rank GuildRank) string {
	if t, ok := g.RankTitles[rank]; ok && t != "" {
		return t
	}
	return string(rank)
}

// validRankTitle enforces a short, single-line, markup-free title: 2-20 runes of
// letters, digits, and spaces only (excludes ':' and ';' — YAML/command gotchas —
// and ANSI markup).
func validRankTitle(title string) error {
	title = strings.TrimSpace(title)
	if n := utf8.RuneCountInString(title); n < 2 || n > 20 {
		return fmt.Errorf("a rank title must be 2-20 characters")
	}
	for _, r := range title {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ') {
			return fmt.Errorf("a rank title may only contain letters, digits, and spaces")
		}
	}
	return nil
}
```

- [ ] **Step 4: Run → PASS**; `go build ./internal/guilds/`.

- [ ] **Step 5: Commit**
```bash
git add internal/guilds/guilds.go internal/guilds/guilds_test.go
git commit -m "feat(guilds): RankTitles field + RankTitle helper + validRankTitle"
```

---

## Task 2: Registry `SetRankTitle`

**Files:** `internal/guilds/registry.go`, `internal/guilds/registry_test.go`

- [ ] **Step 1: Failing test** — add to `registry_test.go`:
```go
func TestRegistry_RankTitle(t *testing.T) {
	defer SetDataDirForTest(t.TempDir())()
	resetRegistry()
	if _, err := Create("RT", "Ranktitlers", 1, "L"); err != nil {
		t.Fatal(err)
	}
	if err := SetRankTitle("RT", RankOfficer, "Lieutenant"); err != nil {
		t.Fatal(err)
	}
	if g, _ := Get("RT"); g.RankTitle(RankOfficer) != "Lieutenant" {
		t.Errorf("title not set")
	}
	// Reset (empty) removes the key -> default.
	if err := SetRankTitle("RT", RankOfficer, ""); err != nil {
		t.Fatal(err)
	}
	if g, _ := Get("RT"); g.RankTitle(RankOfficer) != "officer" {
		t.Errorf("reset should restore default")
	}
}
```

- [ ] **Step 2: Run → FAIL** — `go test ./internal/guilds/ -run TestRegistry_RankTitle`

- [ ] **Step 3: Implement** in `registry.go` (near `SetTreasuryDelegated`):
```go
// SetRankTitle sets (title != "") or clears (title == "") the custom title for
// rank and persists. Caller validates non-empty titles first.
func SetRankTitle(tag string, rank GuildRank, title string) error {
	g, ok := Get(tag)
	if !ok {
		return fmt.Errorf("no such guild")
	}
	registryMu.Lock()
	if title == "" {
		delete(g.RankTitles, rank)
	} else {
		if g.RankTitles == nil {
			g.RankTitles = map[GuildRank]string{}
		}
		g.RankTitles[rank] = title
	}
	registryMu.Unlock()
	return Save(g)
}
```

- [ ] **Step 4: Run → PASS**.

- [ ] **Step 5: Commit**
```bash
git add internal/guilds/registry.go internal/guilds/registry_test.go
git commit -m "feat(guilds): SetRankTitle registry op"
```

---

## Task 3: `title` command + display swaps

**Files:** `internal/usercommands/guild.go`, `internal/usercommands/guild_test.go`, help templates

- [ ] **Step 1: Failing test** — add to `guild_test.go`:
```go
func TestParseGuildRank(t *testing.T) {
	cases := map[string]guilds.GuildRank{
		"member": guilds.RankMember, "Officer": guilds.RankOfficer, "LEADER": guilds.RankLeader,
	}
	for in, want := range cases {
		if got, ok := parseGuildRank(in); !ok || got != want {
			t.Errorf("parseGuildRank(%q) = %v,%v want %v", in, got, ok, want)
		}
	}
	if _, ok := parseGuildRank("captain"); ok {
		t.Error("unknown rank should not parse")
	}
}
```

- [ ] **Step 2: Run → FAIL** — `go test ./internal/usercommands/ -run TestParseGuildRank`

- [ ] **Step 3: Implement** in `guild.go`:

Dispatcher case (after the `treasury` case):
```go
	case "title":
		guildSetTitle(user, remainder)
```

Functions:
```go
// parseGuildRank resolves a case-insensitive rank name to a GuildRank.
func parseGuildRank(s string) (guilds.GuildRank, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "member":
		return guilds.RankMember, true
	case "officer":
		return guilds.RankOfficer, true
	case "leader":
		return guilds.RankLeader, true
	}
	return "", false
}

func guildSetTitle(user *users.UserRecord, remainder string) {
	g, ok := guilds.GetByUser(user.UserId)
	if !ok {
		user.SendText(messaging.CategorySystem, `You are not in a guild.`)
		return
	}
	if !g.IsLeader(user.UserId) {
		user.SendText(messaging.CategorySystem, `Only the guild leader can rename ranks.`)
		return
	}
	fields := strings.Fields(remainder)
	if len(fields) == 0 {
		user.SendText(messaging.CategorySystem, `Usage: <ansi fg="command">guild title <member|officer|leader> [new name]</ansi>  (omit the name to reset).`)
		return
	}
	rank, ok := parseGuildRank(fields[0])
	if !ok {
		user.SendText(messaging.CategorySystem, `Name a rank: <ansi fg="white">member</ansi>, <ansi fg="white">officer</ansi>, or <ansi fg="white">leader</ansi>.`)
		return
	}
	title := strings.TrimSpace(strings.Join(fields[1:], " "))
	if title == "" {
		guilds.SetRankTitle(g.Tag, rank, "")
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You reset the <ansi fg="white">%s</ansi> rank to its default name.`, rank))
		return
	}
	if err := guilds.ValidRankTitle(title); err != nil {
		user.SendText(messaging.CategorySystem, `Invalid title: `+err.Error())
		return
	}
	guilds.SetRankTitle(g.Tag, rank, title)
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`Members of <ansi fg="white">%s</ansi> rank are now titled <ansi fg="cyan">%s</ansi>.`, rank, title))
}
```
> NOTE: `validRankTitle` is unexported in `guilds`. Export a thin wrapper `ValidRankTitle` in `guilds.go` OR move validation into `SetRankTitle` returning the error. **Chosen: export `ValidRankTitle`** (add `func ValidRankTitle(t string) error { return validRankTitle(t) }` in guilds.go) so the command validates before mutating and the internal helper name stays lowercase for tests. (Update Task 1 to add this one-liner.)

Display swaps in `guild.go`:
- In `guildInfo`, the roster line: replace `m.Rank` with `g.RankTitle(m.Rank)`.
- In `guildSetRank`, promote/demote confirmations: replace "to officer"/"to member" literals with `g.RankTitle(guilds.RankOfficer)` / `g.RankTitle(guilds.RankMember)`.

- [ ] **Step 4: Help** — add to both `guild.template` copies under the roster commands:
```
  <ansi fg="command">guild title <rank> <name></ansi>  - (leader) Rename a rank (member/officer/leader)
```

- [ ] **Step 5: Run → PASS + build + full pkg test**
```
go test ./internal/usercommands/ -run TestParseGuildRank
go build ./... && go test ./internal/usercommands/ ./internal/guilds/
```

- [ ] **Step 6: Commit**
```bash
git add internal/usercommands/guild.go internal/usercommands/guild_test.go _datafiles/world/dogmud/templates/help/guild.template _datafiles/world/default/templates/help/guild.template
git commit -m "feat(guilds): guild title command (custom rank names) + display"
```

---

## Task 4: Boot smoke + docs

- [ ] **Step 1** — `go build ./... && go test ./internal/guilds/ ./internal/usercommands/`.
- [ ] **Step 2** — wipe instance saves, `go run .`, confirm `Server Ready`, no panic, stop.
- [ ] **Step 3** — `PATCH_NOTES.md` entry; `docs/PATH_TO_1.0.md` guilds line (ranks-polish/custom titles done → guild arc COMPLETE).
- [ ] **Step 4** — commit docs.
