# Town Justice 5.1c — Arrest Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Guards arrest wanted players (surrender → jail + decaying fine; resist → lethal combat) instead of killing on sight; serving or paying clears the crime, withdraws the faction bounty, and resets reputation to low neutral.

**Architecture:** `Verdict` downshifts wanted signals from `SeverityAttack` to `SeverityArrest`; the enforcement tick forks on the player's `ArrestPolicy`. A new `internal/justice/arrest.go` holds the pure fine math + `ExecuteArrest`/`ResolveDetention`. Jail is a `no-go` buff + a new cell room with `allow_recall:false`. Two jail commands + a `set arrest` policy toggle + helpfiles + context.md updates.

**Tech Stack:** Go; reuses 5.1b `bountyGold` (same package), 5.1a guard tick + `guardSayFn` seam; `crimes`/`bounties`/`factions`/`buffs`/`rooms`; seam-based tests.

**Spec:** `docs/superpowers/specs/2026-05-29-town-justice-5.1c-arrest-design.md`

**Branch:** `feature/town-justice-5.1c-arrest` (forked off 5.1b — `bountyGold` present).

**Verified facts (confirm any file:line by reading before editing — the codebase moves):**
- `set surrender` is the pattern to mirror for `set arrest`: handler in `internal/usercommands/set.go:435-452`; parser `characters.ParseSurrenderPolicy` in `internal/characters/submission_policy.go:91`. `SubmissionPolicy`/`SurrenderPolicy` fields live on `characters.Character` (~character.go:145).
- Command registry: `userCommands map[string]CommandAccess` at `internal/usercommands/usercommands.go:52`; `RegisterCommand(...)` at :556.
- Helpfiles: `_datafiles/world/dogmud/templates/help/<cmd>.template`, ANSI-tag format (see `bounty.template` as the model — header line + Usage block).
- Guard mobs: `106-city_guard.yaml`, `94-guard_captain_velk.yaml` (and 92) are `behavior_archetype: noncombat_questgiver`, `hostile: false`. They do NOT set `non_combatant: true`. `mobs.Validate()` (`mobs.go:1047`) only sets `Character.NonCombatant` from the explicit YAML `non_combatant` field. Attack-block is at `usercommands/attack.go:166` (`IsNonCombatant() || PlayerAttackImmune`). So guards may ALREADY be attackable — Task 1 verifies empirically.
- `no-go` flag = `buffs.NoMovement` (`buffs/buffspec.go:37`), enforced at `usercommands/go.go:89` (`HasBuffFlag(buffs.NoMovement)`).
- Recall block: `validateFoldRecall` (`hooks/spell_foldrecall.go:25`) checks the room's `allow_recall` TempData.
- `rooms.MoveToRoom(userId int, toRoomId int, isSpawn ...bool) error` (`rooms/roommanager.go:267`).
- Room 473 (Guard Barracks) exits: `south→460`, `up→5104`; **`down` is free**. Velk (94) spawns there. Description already references back-of-room cells.
- 5.1b unexported, same-package (callable from `arrest.go`): `bountyGold(powerBase int, isMurder bool, rep int, murderMult, repMultMax float64) int`, seams `bDefaultGoldFn`/`bRepFn`/`bMurderMultFn`/`bRepMultMaxFn`/`bNowFn`, `existingFactionBounty`. `bounties.Withdraw(id int)`, `bounties.OpenAgainstPlayer(userId)`, `crimes.Resolve(...)` — confirm `crimes.Resolve` signature before use.
- `internal/justice/` has NO context.md yet (create one). `internal/usercommands/context.md` and `internal/buffs/context.md` exist.

---

## File Structure

| File | Change |
|------|--------|
| `internal/configs/config.balance.go` + `config.balance.mobs.go` | 3 knobs + defaults: `ArrestResistGraceRounds`(3), `JusticeFineDecayPerRound`(5), `JusticeArrestRepReset`(-10) |
| `internal/justice/justice.go` | `Verdict`: wanted `SeverityAttack` returns → `SeverityArrest` |
| `internal/justice/justice_test.go` | amend verdict tests (Attack→Arrest) |
| `internal/justice/arrest.go` (new) | `currentFine` (pure), `computeFine`/`sentenceRounds`, `ExecuteArrest`, `ResolveDetention`, jail-record helpers, faction→cell map, seams |
| `internal/justice/arrest_test.go` (new) | fine math + ResolveDetention seam tests |
| `internal/justice/enforce.go` | Arrest branch in `RunGuardEnforcement` (policy fork + grace window) |
| `internal/justice/enforce_test.go` | fork + window tests |
| `internal/justice/context.md` (new) | document the justice package incl. arrest |
| `internal/characters/arrest_policy.go` (new) | `ArrestPolicy` type + `ParseArrestPolicy` |
| `internal/characters/character.go` | `ArrestPolicy` field (default surrender) |
| `internal/usercommands/set.go` | `set arrest` subcommand |
| `internal/usercommands/jail.go` (new) | `fine` + `pay fine` commands, `IsJailed` gate |
| `internal/usercommands/usercommands.go` | register `fine`, `payfine` (+ `arrest` alias if used) |
| `internal/usercommands/context.md` | note jail commands + set arrest |
| `internal/hooks/Jail_BuffExpiryRelease.go` (new) | release on Jailed-buff natural expiry |
| `_datafiles/world/dogmud/buffs/<id>-jailed.yaml` (new) | Jailed buff, `no-go` flag |
| `_datafiles/world/dogmud/rooms/thornwall_city/<id>.yaml` (new) | cellblock, `down` from 473, `allow_recall:false` |
| `_datafiles/world/dogmud/rooms/thornwall_city/473.yaml` | add `down` exit to the new cell |
| `_datafiles/world/dogmud/templates/help/{fine,payfine,arrest,justice}.template` (new) | helpfiles |

---

### Task 1: Verify guard attackability (prerequisite — investigation)

The resist path requires a wanted player to be able to attack a guard. This was a deliberate pre-5.1c safety; confirm the actual current behavior before building resist.

- [ ] **Step 1** — Read `internal/usercommands/attack.go:160-175`, `internal/mobs/mobs.go:188-200` (`IsNonCombatant`), and the three guard YAMLs (`106-city_guard.yaml`, `94-guard_captain_velk.yaml`, `92-*`). Determine: with `non_combatant` absent and `hostile:false`, does `attack <guard>` currently succeed or get blocked? Check for any OTHER block (archetype rule, `PlayerAttackImmune`, a peaceful-faction gate in attack.go).

- [ ] **Step 2** — Write findings to the task report. Two outcomes:
  - **(A) Guards are already attackable** → no code change; note it and proceed. Resist works out of the box.
  - **(B) Guards are blocked** (some mechanism prevents it) → the fix is to make a guard attackable *by a wanted player* (or once arrest is declared). Minimal approach: gate the block so it does NOT apply when `justice.Verdict(factions.FactionsForMob(guard), attackerUserId) >= SeverityArrest`. Implement in `attack.go` next to the existing block, importing `justice` (justice must not import usercommands — one-way, fine). Add a focused test.

- [ ] **Step 3** — If (B), commit:
```bash
git add internal/usercommands/attack.go internal/usercommands/attack_test.go
git commit -m "feat(justice): wanted players may attack guards (enables 5.1c resist)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```
If (A), no commit; record the finding in the report so later tasks rely on it. **If the mechanism is ambiguous or a change here risks letting players grief peaceful guards, STOP and report — this is a design-sensitive gate.**

---

### Task 2: Config knobs

- [ ] **Step 1** — In `internal/configs/config.balance.go`, after the 5.1b `JusticeBountyRepMultMax` field, add:
```go
	// ArrestResistGraceRounds is the window between a guard declaring an arrest
	// and hauling the player off, during which the player may fight back. 5.1c.
	ArrestResistGraceRounds ConfigInt `yaml:"ArrestResistGraceRounds"`

	// JusticeFineDecayPerRound is gold the arrest fine drops per round served;
	// sentence length = fine / this. 5.1c.
	JusticeFineDecayPerRound ConfigInt `yaml:"JusticeFineDecayPerRound"`

	// JusticeArrestRepReset is the reputation floor restored with the issuing
	// faction when a sentence is served/paid (only raises, never lowers). 5.1c.
	JusticeArrestRepReset ConfigInt `yaml:"JusticeArrestRepReset"`
```

- [ ] **Step 2** — In `internal/configs/config.balance.mobs.go`, after the 5.1b `JusticeBountyRepMultMax` default guard, add:
```go
	if b.ArrestResistGraceRounds < 1 {
		b.ArrestResistGraceRounds = 3
	}
	if b.JusticeFineDecayPerRound < 1 {
		b.JusticeFineDecayPerRound = 5
	}
	if b.JusticeArrestRepReset == 0 {
		b.JusticeArrestRepReset = -10
	}
```
(`JusticeArrestRepReset` uses `== 0` not `< 1` because the intended default is negative; a designer setting it to exactly 0 — true neutral — should re-pick, acceptable.)

- [ ] **Step 3** — `go build ./internal/configs/`; `gofmt -l` clean. Commit:
```bash
git add internal/configs/config.balance.go internal/configs/config.balance.mobs.go
git commit -m "feat(config): 5.1c arrest knobs (grace, fine decay, rep reset)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: Verdict reshape (Attack → Arrest)

- [ ] **Step 1** — In `internal/justice/justice_test.go`, find the `Verdict` tests asserting `SeverityAttack` for wanted signals (open bounty, Hostile rep, unresolved crime). Change those expectations to `SeverityArrest`. Leave Cold→`Warn` and Neutral→`None` as-is. Run `go test ./internal/justice/ -run Verdict` → FAIL (still returns Attack).

- [ ] **Step 2** — In `internal/justice/justice.go` `Verdict`, change every `return SeverityAttack` / `worst = SeverityAttack` (the bounty, Hostile-rep, and crime branches) to `SeverityArrest`. Do NOT touch the enum definition or the `SeverityWarn` path.

- [ ] **Step 3** — `go test ./internal/justice/` → PASS; `go build ./...`. Commit:
```bash
git add internal/justice/justice.go internal/justice/justice_test.go
git commit -m "feat(justice): Verdict yields Arrest (not Attack) for wanted signals

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: ArrestPolicy + `set arrest`

- [ ] **Step 1 — failing test** — Create `internal/characters/arrest_policy_test.go`:
```go
package characters

import "testing"

func TestParseArrestPolicy(t *testing.T) {
	cases := map[string]ArrestPolicy{
		"surrender": ArrestSurrender,
		"resist":    ArrestResist,
	}
	for in, want := range cases {
		got, ok := ParseArrestPolicy(in)
		if !ok || got != want {
			t.Errorf("ParseArrestPolicy(%q) = %v,%v; want %v,true", in, got, ok, want)
		}
	}
	if _, ok := ParseArrestPolicy("bogus"); ok {
		t.Error("bogus should not parse")
	}
}
```
Run → FAIL (undefined).

- [ ] **Step 2** — Create `internal/characters/arrest_policy.go`:
```go
package characters

// ArrestPolicy is the player's pre-decided response when a town guard attempts
// an arrest (Town Justice 5.1c). Surrender (default) = be jailed; Resist = the
// guard drops to lethal combat.
type ArrestPolicy string

const (
	ArrestSurrender ArrestPolicy = "surrender"
	ArrestResist    ArrestPolicy = "resist"
)

// ParseArrestPolicy parses a player-supplied policy string.
func ParseArrestPolicy(s string) (ArrestPolicy, bool) {
	switch ArrestPolicy(s) {
	case ArrestSurrender, ArrestResist:
		return ArrestPolicy(s), true
	}
	return "", false
}
```

- [ ] **Step 3** — In `internal/characters/character.go`, add an `ArrestPolicy ArrestPolicy` field near `SurrenderPolicy`. Ensure it defaults to `ArrestSurrender` wherever `SurrenderPolicy` gets its default (find that init site; if empty-string on load, treat empty as surrender at read time in Task 7 rather than forcing a migration). Run `go test ./internal/characters/ -run ArrestPolicy` → PASS.

- [ ] **Step 4** — In `internal/usercommands/set.go`, add a `set arrest` subcommand mirroring `set surrender` (~line 435): no-arg shows current policy + usage `set arrest <surrender|resist>`; arg parses via `characters.ParseArrestPolicy`, sets the field, confirms. Wire it into `set`'s subcommand dispatch the same way `surrender` is wired.

- [ ] **Step 5** — `go build ./...`; `gofmt -l` clean; `go test ./internal/characters/ ./internal/usercommands/`. Commit:
```bash
git add internal/characters/arrest_policy.go internal/characters/arrest_policy_test.go internal/characters/character.go internal/usercommands/set.go
git commit -m "feat(justice): ArrestPolicy + set arrest command

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: Jailed buff + cellblock room (content)

- [ ] **Step 1** — Run `python tools/id_inventory.py --type buffs` and `--type rooms` (or `--zone thornwall`) to get the next free buff ID and a free Thornwall City room ID. Record both.

- [ ] **Step 2** — Create `_datafiles/world/dogmud/buffs/<buffid>-jailed.yaml` (filename via `ConvertForFilename("Jailed")` → `jailed`):
```yaml
buffid: <buffid>
name: Jailed
description: You are locked in a holding cell. You cannot leave until your
  sentence is served or your fine is paid.
triggerrate: 1 round
triggercount: 1   # duration is overwritten per-arrest via AddBuff with explicit rounds
flags:
  - no-go
start_user_text: "The cell door clangs shut behind you. You are held until your sentence is served."
end_user_text: "Your sentence is served."
```
(Confirm the buff schema fields against an existing simple buff, e.g. `36-*.yaml`/psychic-anchor; match `triggerrate`/`triggercount` conventions. Duration is applied at arrest time via the buff-add call with the computed sentence rounds — verify the engine supports a runtime duration override; if not, note it and have `ExecuteArrest` manage the timer via the jail record + a per-round check instead.)

- [ ] **Step 3** — Create the cellblock room `_datafiles/world/dogmud/rooms/thornwall_city/<roomid>.yaml`. Model fields on a neighboring Thornwall City room. Required: `roomid`, `zone: Thornwall City`, an evocative `title` (e.g. "Holding Cell") and 80-col-wrapped `description` (iron bars, straw, the barracks above), `exits:` with `up: {roomid: 473}` (lore: stairs back up), and a top-level `allow_recall: false` (confirm this is a room-YAML field or set via tempdata default — grep an instanced zone room that uses `allow_recall`). Add cell-flavor nouns (door, bars, cot, slop bucket — mirror Stillwater 4110's noun style).

- [ ] **Step 4** — In `_datafiles/world/dogmud/rooms/thornwall_city/473.yaml`, add to `exits:`:
```yaml
  down:
    roomid: <new cell roomid>
```
(Optionally gate it so only guards/jailed move through, but the Jailed buff's `no-go` already blocks the player from walking out via `up`; a normal player walking `down` into the cell voluntarily is harmless — leave open or add a locked exit if trivial. Default: leave open.)

- [ ] **Step 5** — Boot smoke this content early: wipe instances (`rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*`), `go run .` background, confirm the new buff + room load with no panic (`Server Ready`), then kill. Commit:
```bash
git add _datafiles/world/dogmud/buffs/<buffid>-jailed.yaml _datafiles/world/dogmud/rooms/thornwall_city/<roomid>.yaml _datafiles/world/dogmud/rooms/thornwall_city/473.yaml
git commit -m "content(justice): Thornwall holding cell + Jailed buff

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: `arrest.go` — fine math, ExecuteArrest, ResolveDetention

**Files:** Create `internal/justice/arrest.go`, `internal/justice/arrest_test.go`

- [ ] **Step 1 — failing tests** (`arrest_test.go`):
```go
package justice

import "testing"

func TestCurrentFine(t *testing.T) {
	// original 100, decay 5/round
	if g := currentFine(100, 0, 5); g != 100 {
		t.Errorf("served=0 got %d want 100", g)
	}
	if g := currentFine(100, 10, 5); g != 50 {
		t.Errorf("served=10 got %d want 50", g)
	}
	if g := currentFine(100, 30, 5); g != 0 {
		t.Errorf("served past sentence got %d want 0 (floored)", g)
	}
}

func TestSentenceRounds(t *testing.T) {
	if r := sentenceRounds(100, 5); r != 20 {
		t.Errorf("got %d want 20", r)
	}
	if r := sentenceRounds(3, 5); r != 1 {
		t.Errorf("tiny fine should floor to 1 round, got %d", r)
	}
}
```
Run → FAIL.

- [ ] **Step 2** — Create `internal/justice/arrest.go` with the pure helpers + the jail-record/seam scaffolding:
```go
package justice

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/bounties"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crimes"
	"github.com/GoMudEngine/GoMud/internal/knowledge"
)

// currentFine = max(0, original - served*decay). Pure.
func currentFine(original, roundsServed, decayPerRound int) int {
	f := original - roundsServed*decayPerRound
	if f < 0 {
		return 0
	}
	return f
}

// sentenceRounds = max(1, fine / decayPerRound). Pure.
func sentenceRounds(fine, decayPerRound int) int {
	if decayPerRound < 1 {
		decayPerRound = 1
	}
	r := fine / decayPerRound
	if r < 1 {
		return 1
	}
	return r
}

// computeFine reuses 5.1b bountyGold so the fine equals the price on the
// player's head for this faction/crime.
func computeFine(faction string, userId int, isMurder bool) int {
	return bountyGold(
		bDefaultGoldFn(knowledge.PlayerSubject(userId)),
		isMurder,
		bRepFn(faction, userId),
		bMurderMultFn(), bRepMultMaxFn(),
	)
}

// jailCellFor maps a guard faction to its holding-cell room id. Thornwall now;
// Stillwater (4110) when stillwater_guards lands.
var jailCellFor = map[string]int{
	"thornwall_guards": <CELL_ROOMID>, // fill from Task 5
}

// MiscData keys for the jail record.
const (
	jailUntilKey   = "jail_until_round"
	jailFineKey    = "jail_fine_original"
	jailDecayKey   = "jail_decay_per_round"
	jailFactionKey = "jail_faction"
	jailCrimesKey  = "jail_crime_ids"
	jailCellKey    = "jail_cell_room"
)

// Release-path seams (tests override).
var (
	aResolveCrimeFn  = crimes.Resolve
	aWithdrawFn      = bounties.Withdraw
	aOpenBountiesFn  = bounties.OpenAgainstPlayer
	aRepResetFloorFn = func() int { return configs.GetBalanceConfig().JusticeArrestRepReset }
)
```
NOTE: `<CELL_ROOMID>` is filled from Task 5's allocated room id. Confirm `crimes.Resolve`'s real signature (it may take `[]int` crime ids and return something) and adapt `aResolveCrimeFn`'s type + the call in ResolveDetention. Confirm `bounties.Withdraw`'s signature.

- [ ] **Step 3** — Run `go test ./internal/justice/ -run 'CurrentFine|SentenceRounds'` → PASS.

- [ ] **Step 4** — Add `ExecuteArrest` and `ResolveDetention` to `arrest.go`. These touch the live `mobs.Mob`/`characters.Character` types — model the move + buff-apply + MiscData stamping on existing call patterns (grep `MoveToRoom`, `AddBuff`/`AddBuffScaled`, `SetMiscData`). Signatures:
```go
// ExecuteArrest hauls a surrendered player to the faction's cell, applies the
// Jailed buff for the sentence duration, stamps the jail record, and plays
// drag-to-jail flavor. Returns false (no-op) if the faction has no cell.
func ExecuteArrest(guard *mobs.Mob, player *characters.Character, userId int, faction string, crimeIds []int, isMurder bool) bool

// ResolveDetention ends a detention (timer expiry OR fine paid): clears crimes,
// withdraws the issuing faction's open bounty, resets rep to the floor (only if
// below), removes the Jailed buff, and moves the player back to the barracks.
func ResolveDetention(player *characters.Character, userId int) bool
```
For these, ADD seams as needed for `MoveToRoom`, buff add/remove, rep get/set, and `now` (reuse `bNowFn`) so the behavior is unit-testable; mirror the 5.1a/5.1b seam style. Flavor lines: drag-off broadcast at the origin room ("`<guard>` hauls `<player>` off toward the cells, ignoring every protest.") via `guardSayFn`-style room send, and an arrival line in the cell; release line on resolve. Keep all flavor first-person/evocative per the spec; no raw numbers except the fine (gold is currency, allowed — surfaced via the `fine` command).

- [ ] **Step 5** — Add a `ResolveDetention` seam test (override `aResolveCrimeFn`/`aWithdrawFn`/`aOpenBountiesFn`/rep seams + move/buff seams; assert: crimes resolved with the stamped ids, open faction bounty withdrawn, rep raised to floor only when below, buff removed, move-home called). Run `go test ./internal/justice/` → PASS; `go build ./...`; `gofmt -l` clean.

- [ ] **Step 6** — Commit:
```bash
git add internal/justice/arrest.go internal/justice/arrest_test.go
git commit -m "feat(justice): arrest execution + detention resolution + fine math

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 7: Enforcement fork (Arrest branch)

- [ ] **Step 1 — failing test** (`enforce_test.go`): add cases — (a) `surrender` policy + Arrest verdict, first tick → declares (stamps `justice_arrest_pending_<uid>`, says intent), no move; (b) within grace → no-op; (c) past grace → calls ExecuteArrest (via a seam) and clears the pending stamp; (d) `resist` policy + Arrest verdict → issues `attack @uid`, no pending stamp. Use the existing enforce-test seam style; add an `executeArrestFn` seam to `enforce.go` so the test asserts the call without live mobs/rooms. Run → FAIL.

- [ ] **Step 2** — In `internal/justice/enforce.go` `RunGuardEnforcement`, add the `case SeverityArrest:` branch (between Warn and Attack handling). Logic per spec §2:
  - Read player ArrestPolicy (empty string → treat as `ArrestSurrender`).
  - `resist` → `mob.Command(fmt.Sprintf("attack @%d", uid))`, record an `EnforceAction{uid, SeverityAttack, false}`.
  - `surrender` → read `justice_arrest_pending_<uid>` via `miscDataRound`:
    - absent → `guardSayFn` intent line, stamp `nowRound`, record `EnforceAction{uid, SeverityArrest, false}`.
    - present & `nowRound - pending < ArrestResistGraceRounds` → no-op.
    - present & past grace → `executeArrestFn(mob, player, uid, faction, crimeIds, isMurder)`, clear the stamp, record `EnforceAction{uid, SeverityArrest, true}` (Escalated=true marks the haul-off).
  - Add a `graceRoundsArrestFn` reading `ArrestResistGraceRounds` (with inline default 3), mirroring `warnGraceRounds`.
  - The faction + isMurder + crimeIds for the arrest: derive faction from the matched guard faction set; for crimeIds/isMurder, query the player's unresolved crimes against the faction (reuse whatever `Verdict`'s crime branch uses). If precise crime-id threading is heavy, ExecuteArrest may re-query inside — keep the enforce branch thin and let `arrest.go` gather crime ids; adjust signatures to match. Confirm against the actual `crimes` query API.

- [ ] **Step 3** — `go test ./internal/justice/` → PASS; `go build ./...`; `gofmt -l` clean. Commit:
```bash
git add internal/justice/enforce.go internal/justice/enforce_test.go
git commit -m "feat(justice): guard enforcement arrest fork (surrender/resist + grace)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 8: Jail commands (`fine`, `pay fine`)

**Files:** Create `internal/usercommands/jail.go`; modify `usercommands.go`.

- [ ] **Step 1** — Create `internal/usercommands/jail.go` with an `IsJailed(user)` helper (true when the character has the Jailed buff — `HasBuffFlag(buffs.NoMovement)` is too broad; check buff id presence or a dedicated `buffs.Jailed` flag — simplest: check the jail record's `jail_until_round` MiscData key is set). Then:
  - `Fine(rest string, user *users.UserRecord, ...) (bool, error)` — if not jailed → "You're not in a cell." Else read jail record, compute `currentFine(original, nowRound-... , decay)` using `util.GetRoundCount`, and show current fine (gold figure allowed), rounds remaining, and `pay fine` hint. (Export a small read accessor from `justice` for the record, or read MiscData keys directly via shared key constants — prefer a `justice.JailInfo(player)` accessor returning a struct to avoid leaking key names.)
  - `PayFine(...)` — if not jailed → reject. Compute current fine; deduct from `user.Character.Gold` first, then bank (find the bank-balance/withdraw API — grep `bank`); if total insufficient → flavor refusal, stay jailed. On success → `justice.ResolveDetention(user.Character, user.UserId)` and a "you buy your freedom" line.

- [ ] **Step 2** — Register in `internal/usercommands/usercommands.go`'s `userCommands` map: `"fine"` and `"payfine"` (and decide `pay fine` UX — either a `payfine` command, or extend an existing `pay` command; mirror how multi-word commands are handled elsewhere — confirm whether the parser supports `pay fine` as `pay` + arg. Simplest: register `payfine`; document `pay fine` as the helpfile phrasing only if a `pay` dispatcher exists). Match the `CommandAccess` tuple fields used by neighbors (disabledWhenDowned, allowedInCombat, isAdminOnly=false).

- [ ] **Step 3** — `go build ./...`; `gofmt -l` clean; `go test ./internal/usercommands/`. Manual-ish: the real exercise is the boot smoke + the user's in-game test. Commit:
```bash
git add internal/usercommands/jail.go internal/usercommands/usercommands.go
git commit -m "feat(justice): jail commands (fine, pay fine)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 9: Timed-release on buff expiry

- [ ] **Step 1** — Create `internal/hooks/Jail_BuffExpiryRelease.go`. When the Jailed buff expires naturally (sentence served), `ResolveDetention` must run so the timer-release path clears crime/bounty/rep and moves the player home — not just a silent buff drop. Find the engine's buff-expiry hook/event (grep how other buffs fire end effects — `end_user_text` is cosmetic; look for an `onEnd`/buff-removed observer, or a per-round check). Two viable wirings (pick whichever the engine supports cleanly):
  - **(a)** Subscribe to a buff-removed/expired event filtered to the Jailed buff id → call `justice.ResolveDetention`.
  - **(b)** A per-round check (in the existing justice/mob round tick) that, for any character with a jail record whose `jail_until_round <= now` AND still jailed, calls `ResolveDetention`. This also self-heals if a buff is dropped by other means.
  Prefer (b) if the buff event surface is awkward — it's robust and matches the 5.1a/b per-round-tick idiom. Confirm by reading `internal/hooks/NewRound_MobRoundTick.go` (where 5.1a/b ticks live) and the buff system.

- [ ] **Step 2** — Add a test for the chosen mechanism (pure-ish: a character past `jail_until_round` triggers ResolveDetention exactly once; not-yet-expired does not). Run it.

- [ ] **Step 3** — `go build ./...`; `gofmt -l` clean. Commit:
```bash
git add internal/hooks/Jail_BuffExpiryRelease.go internal/hooks/<test>.go
git commit -m "feat(justice): release jailed player when sentence expires

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 10: Helpfiles + context.md (REQUIRED — no command ships without docs)

- [ ] **Step 1 — `set arrest` / arrest policy helpfile.** Add coverage to the help system. If `set.template` documents subcommands, add an `arrest` entry there; ALSO create `_datafiles/world/dogmud/templates/help/arrest.template` describing the policy (surrender default → jail; resist → guards fight you; how to set: `set arrest <surrender|resist>`; what jail entails). Use the `bounty.template` ANSI header/Usage format. Keep prose ≤80 cols.

- [ ] **Step 2 — jail command helpfiles.** Create `_datafiles/world/dogmud/templates/help/fine.template` and `payfine.template` (or fold `pay fine` into `fine.template` if registered as one command): what the fine is, that it decays as you serve, paying from person-then-bank, and that serving OR paying clears your record with that faction. ≤80 cols, ANSI format.

- [ ] **Step 3 — `justice` overview helpfile.** Create `_datafiles/world/dogmud/templates/help/justice.template`: a player-facing overview of town justice — committing crimes lowers faction rep and gets you wanted; guards warn, then arrest; surrender vs resist; jail + fine + redemption; cross-reference `help bounty`, `help arrest`, `help fine`. ≤80 cols.

- [ ] **Step 4 — Verify help discoverability.** Confirm the help system auto-discovers `.template` files by name (grep how `help <cmd>` resolves templates) — if a manifest/index must list new help topics, update it. Boot smoke later confirms `help fine` / `help arrest` / `help justice` render.

- [ ] **Step 5 — context.md updates.**
  - Create `internal/justice/context.md` documenting the package: `Verdict` (now emits Arrest), `RunGuardEnforcement` (warn/arrest/resist fork + grace), `bounty.go` (5.1b auto-bounty), `arrest.go` (fine math, ExecuteArrest, ResolveDetention, jail record/cell map), the `guardSayFn` seam, and the test-seam pattern. Model structure on a neighboring `context.md` (e.g. `internal/bounties/context.md`).
  - Update `internal/usercommands/context.md` — add the `fine`/`payfine` jail commands and the `set arrest` subcommand to its command inventory.
  - Update `internal/buffs/context.md` if it enumerates buffs/flags — note the Jailed buff uses `no-go`.

- [ ] **Step 6** — Commit:
```bash
git add _datafiles/world/dogmud/templates/help/arrest.template _datafiles/world/dogmud/templates/help/fine.template _datafiles/world/dogmud/templates/help/payfine.template _datafiles/world/dogmud/templates/help/justice.template internal/justice/context.md internal/usercommands/context.md internal/buffs/context.md
git commit -m "docs(justice): helpfiles (arrest/fine/payfine/justice) + context.md

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 11: Verification + boot smoke

- [ ] **Step 1** — `go build ./...`; `go vet ./internal/justice/ ./internal/characters/ ./internal/usercommands/ ./internal/hooks/`; `gofmt -l` over all touched `.go` files (clean).

- [ ] **Step 2** — `go test ./internal/justice/ ./internal/characters/ ./internal/usercommands/ ./internal/configs/ ./internal/hooks/` — all pass (note the pre-existing flaky `TestHandlePlayerFoldCasting_*` if it appears; not ours).

- [ ] **Step 3** — Boot smoke (SOP): wipe instances (`rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*`), build + run, confirm: the three config knobs log (`ArrestResistGraceRounds`, `JusticeFineDecayPerRound`, `JusticeArrestRepReset`), the Jailed buff + new cell room load, no panic, `Server Ready`. Then `help justice` / `help arrest` / `help fine` exist (if a quick check is feasible) — otherwise leave for the user's in-game smoke. Kill the server.

- [ ] **Step 4** — Report completion. In-game functional smoke (commit a crime → get arrested → serve/pay → confirm release + cleared record) is deferred to the user per the project's manual-smoke convention.

---

## Notes for the implementer

- **Guard attackability (Task 1) is the keystone** — if guards turn out to be hard-blocked and the gate fix is non-trivial or risky, STOP and escalate before building the resist path.
- **Same-package reuse:** `arrest.go` is in package `justice`, so it calls 5.1b's unexported `bountyGold` + `b*Fn` seams directly. No new cross-package wiring for the fine.
- **No new `actions` import in `justice`:** reuse the `guardSayFn` seam (5.1b decouple) for all arrest speech.
- **Gold figures are allowed** in the `fine`/`pay fine` output (currency, not a combat/balance number) — but all OTHER arrest/jail flavor must avoid raw numbers per CLAUDE.md (describe the sentence as "your time," not "20 rounds").
- **Rep reset only raises:** `ResolveDetention` sets rep to the floor only when current rep is below it — never launders an already-good reputation.
- **Crime/signature confirmations the implementer must make before wiring:** `crimes.Resolve` signature + how to query a player's unresolved crime ids against a faction; `bounties.Withdraw` signature; the bank balance/withdraw API; the buff runtime-duration-override capability (Task 5/9); whether `allow_recall` is a room-YAML field or tempdata; whether the help system needs an index entry for new topics.
- **Followups to log at finish** (MEMORY): arrest-pending MiscData stamp pruning (same shape as 5.1a warn-stamp followup); Stillwater cell wiring when `stillwater_guards` exists; whether 5.1d still has scope after pay-fine + rep-reset landed here.
