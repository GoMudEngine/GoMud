# Town Justice 5.1b — Crime → Auto-Bounty Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Town factions auto-post kill-bounties on players who commit an identified murder or whose rep crosses Hostile, and those bounties resolve when the target dies (guard kill → pays the guard; player kill → pays the player; else expires).

**Architecture:** A `justice.MaybeDeclareBounty` helper fired from the crime-recording sites declares scaled bounties via the existing `bounties` engine; a `PlayerDeath_BountyResolve` Life observer closes/claims them on death. Pure reward + attribution helpers carry the logic; package function-seams keep it unit-testable (the 5.1a pattern).

**Tech Stack:** Go; reads `factions`/`crimes`/`bounties`/`knowledge`/`opinions`/`state/life`; seam-based tests.

**Spec:** `docs/superpowers/specs/2026-05-29-town-justice-5.1b-auto-bounty-design.md`

**Verified facts (use; the env was flaky during plan-prep — confirm any file:line by reading before editing):**
- `state.ActorRef{ UserId int; MobInstanceId int }` with helpers `IsZero()`, `IsPlayer()`, `IsMob()` (NO `Kind`/`InstanceId`/`Name` fields).
- `life.DeadData{ Killer state.ActorRef; DamageMap map[int]int; NoDeprogression bool }`; obtained in the player-death observer via `c.Life.DeadData()`.
- Player death observer pattern: `Death_PlayerCleanup.go` wires `wirePlayerDeathCleanup(c *characters.Character)` via `characters.OnCharacterCreated`, using `c.Life.Inner().AfterTransition(name, func(from,to life.State, r state.TransitionReason){ if from!=life.Alive||to!=life.Dead {return}; if c.GetUserId()==0 {return}; ... })`.
- `bounties.computeDefaultGold(statpool int) int` (reads `goldMultiplier()`/`goldFloor()` internally — takes ONLY statpool); `bounties.statpoolFor(target knowledge.Subject) int`; test seam `statpoolForTest`.
- `bounties.Declare(issuer Issuer, target knowledge.Subject, condition Condition, expiryRound uint64, opts DeclareOpts) (int, error)`; `DeclareOpts{ GoldOverride int; RepOverride int; DeclaredReason string }`; `bounties.FactionIssuer(slug)`, `bounties.ConditionKill`, `bounties.IssuerFaction`.
- `bounties.OpenAgainstPlayer(userId) []*Bounty`; `bounties.TryClaim(id int, claimer knowledge.Subject) (*Bounty, bool)`; `bounties.MarkExpired(id int)`; `Bounty{ Id int; Issuer Issuer; GoldReward int; RepReward int; ... }`.
- `knowledge.PlayerSubject(userId) Subject`, `knowledge.MobSubject(templateId) Subject`.
- `factions.TierFor(faction string, userId int) opinions.Tier`, `factions.GetRep(faction, userId) int`, `factions.FactionsForMob(mob *mobs.Mob) []string`, `factions.BumpRep(faction, userId, delta)`; `opinions.TierHostile`.
- `mobs.GetInstance(instanceId) *mobs.Mob`; `mob.MobId` (template id, type `mobs.MobId`); `mob.Character.Gold int`.
- `util.GetRoundCount() uint64`. Config: `ConfigInt`/`ConfigFloat` fields in `config.balance.go`, defaults in `config.balance.mobs.go`'s validate.
- `MobDeath_BountyClaim.go` is the claim/payout reference (`TryClaim` → `user.Character.Gold += claimed.GoldReward` → `factions.BumpRep(claimed.Issuer.Id, killerUserId, claimed.RepReward)`).
- Assault recorded by `recordAssaultCrime` (`internal/usercommands/attack.go:332`); rep bumped at `attack.go:351` `factions.BumpRep(fid, user.UserId, delta)` inside `if perp.Type == crimes.PerpPlayer` within the `for _, fid` loop. Murder in `internal/hooks/MobDeath_FactionRep.go` (identified-perp branches that call `factions.BumpRep(fid, userId, ...)`). **Theft is NOT wired in 5.1b** — it lives in `internal/actions/steal.go`/`plant.go`, and `internal/justice` imports `internal/actions` (5.1a `guardSay`), so `actions→justice` would be an import cycle; deferred.

---

## File Structure

| File | Change |
|------|--------|
| `internal/configs/config.balance.go` | 3 knobs (`JusticeBountyExpiryRounds`, `JusticeBountyMurderMult`, `JusticeBountyRepMultMax`) |
| `internal/configs/config.balance.mobs.go` | defaults (5000, 2.0, 2.0) |
| `internal/bounties/bounties.go` | new exported `DefaultGoldFor` |
| `internal/bounties/bounties_test.go` | `DefaultGoldFor` test |
| `internal/justice/bounty.go` (new) | `bountyGold` (pure), `shouldDeclare` (pure), `MaybeDeclareBounty` + seams |
| `internal/justice/bounty_test.go` (new) | reward/gate/declare tests |
| `internal/hooks/MobDeath_FactionRep.go` | call `MaybeDeclareBounty` on identified murder |
| `internal/usercommands/attack.go` | call `MaybeDeclareBounty` after the assault rep-bump (theft deferred — cycle) |
| `internal/hooks/PlayerDeath_BountyResolve.go` (new) | death-resolution observer + pure `attributeBountyKill` |
| `internal/hooks/PlayerDeath_BountyResolve_test.go` (new) | attribution tests |

---

### Task 1: Config knobs + defaults

- [ ] **Step 1** — In `internal/configs/config.balance.go`, after the `JusticeCrimeLookbackRounds` field, add:
```go
	// JusticeBountyExpiryRounds is how long an auto-declared town-faction
	// bounty stays open before lapsing. 5.1b.
	JusticeBountyExpiryRounds ConfigInt `yaml:"JusticeBountyExpiryRounds"`

	// JusticeBountyMurderMult scales an auto-bounty's gold when triggered by
	// an identified murder. 5.1b.
	JusticeBountyMurderMult ConfigFloat `yaml:"JusticeBountyMurderMult"`

	// JusticeBountyRepMultMax is the max gold multiplier at maximum hostility
	// (rep -100) for rep-triggered auto-bounties. 5.1b.
	JusticeBountyRepMultMax ConfigFloat `yaml:"JusticeBountyRepMultMax"`
```

- [ ] **Step 2** — In `internal/configs/config.balance.mobs.go`, after the `JusticeCrimeLookbackRounds` default guard, add:
```go
	if b.JusticeBountyExpiryRounds < 1 {
		b.JusticeBountyExpiryRounds = 5000
	}
	if b.JusticeBountyMurderMult < 1 {
		b.JusticeBountyMurderMult = 2.0
	}
	if b.JusticeBountyRepMultMax < 1 {
		b.JusticeBountyRepMultMax = 2.0
	}
```

- [ ] **Step 3** — `go build ./internal/configs/` (expect clean).

- [ ] **Step 4** — Commit:
```bash
git add internal/configs/config.balance.go internal/configs/config.balance.mobs.go
git commit -m "feat(config): JusticeBounty{ExpiryRounds,MurderMult,RepMultMax}

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: `bounties.DefaultGoldFor`

- [ ] **Step 1** — Append to `internal/bounties/bounties_test.go`:
```go
func TestDefaultGoldFor_MatchesComputeDefault(t *testing.T) {
	restore := statpoolForTest
	statpoolForTest = func(knowledge.Subject) int { return 600 }
	defer func() { statpoolForTest = restore }()
	got := DefaultGoldFor(knowledge.PlayerSubject(17))
	want := computeDefaultGold(600)
	if got != want {
		t.Errorf("DefaultGoldFor = %d, want %d", got, want)
	}
}
```
(If `statpoolForTest` is not the exact seam name, read `bounties.go` for the existing statpool seam and use it.)

- [ ] **Step 2** — Run `go test ./internal/bounties/ -run TestDefaultGoldFor` → FAIL (undefined `DefaultGoldFor`).

- [ ] **Step 3** — In `internal/bounties/bounties.go`, add:
```go
// DefaultGoldFor returns the statpool-derived default gold reward for a target,
// exposing the same computation Declare uses (so callers like internal/justice
// can scale it without duplicating the formula).
func DefaultGoldFor(target knowledge.Subject) int {
	return computeDefaultGold(statpoolFor(target))
}
```

- [ ] **Step 4** — Run the test → PASS; then `go test ./internal/bounties/` (no regressions).

- [ ] **Step 5** — Commit:
```bash
git add internal/bounties/bounties.go internal/bounties/bounties_test.go
git commit -m "feat(bounties): export DefaultGoldFor for justice reward scaling

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: `justice.MaybeDeclareBounty` + reward/gate helpers

**Files:** Create `internal/justice/bounty.go`, `internal/justice/bounty_test.go`

- [ ] **Step 1 — failing tests** (`internal/justice/bounty_test.go`):
```go
package justice

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/crimes"
	"github.com/GoMudEngine/GoMud/internal/opinions"
)

func TestBountyGold_MurderDominant(t *testing.T) {
	// powerBase 100, murderMult 2.0, rep -50 (repMult 1.0) -> max=2.0 -> 200
	if g := bountyGold(100, true, -50, 2.0, 2.0); g != 200 {
		t.Errorf("got %d, want 200", g)
	}
}

func TestBountyGold_RepDominant(t *testing.T) {
	// powerBase 100, not murder (crimeMult 1.0), rep -100 -> repMult 2.0 -> 200
	if g := bountyGold(100, false, -100, 2.0, 2.0); g != 200 {
		t.Errorf("got %d, want 200", g)
	}
}

func TestBountyGold_NeitherDominant_BaseFloor(t *testing.T) {
	// not murder, rep -50 (Hostile boundary, repMult 1.0) -> 100
	if g := bountyGold(100, false, -50, 2.0, 2.0); g != 100 {
		t.Errorf("got %d, want 100", g)
	}
}

func TestShouldDeclare(t *testing.T) {
	if !shouldDeclare(crimes.KindMurder, opinions.TierNeutral, false) {
		t.Error("identified murder should declare")
	}
	if !shouldDeclare(crimes.KindAssault, opinions.TierHostile, false) {
		t.Error("Hostile rep should declare")
	}
	if shouldDeclare(crimes.KindAssault, opinions.TierCold, false) {
		t.Error("Cold rep, no murder -> no declare")
	}
	if shouldDeclare(crimes.KindMurder, opinions.TierHostile, true) {
		t.Error("already-open bounty -> no declare (dedup)")
	}
}
```

- [ ] **Step 2** — `go test ./internal/justice/ -run 'BountyGold|ShouldDeclare'` → FAIL.

- [ ] **Step 3** — Create `internal/justice/bounty.go`:
```go
package justice

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/bounties"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crimes"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/opinions"
)

const hostileRepFloor = 50 // |rep| at the Hostile tier boundary (rep <= -50)

// bountyGold = powerBase * max(crimeMult, repMult). crimeMult is murderMult on
// an identified murder, else 1.0. repMult ramps 1.0 (rep -50) -> repMultMax
// (rep -100); 1.0 if not yet Hostile.
func bountyGold(powerBase int, isMurder bool, rep int, murderMult, repMultMax float64) int {
	crimeMult := 1.0
	if isMurder {
		crimeMult = murderMult
	}
	repMult := 1.0
	if absRep := -rep; absRep > hostileRepFloor {
		frac := float64(absRep-hostileRepFloor) / float64(100-hostileRepFloor)
		if frac > 1 {
			frac = 1
		}
		repMult = 1.0 + frac*(repMultMax-1.0)
	}
	mult := crimeMult
	if repMult > mult {
		mult = repMult
	}
	return int(math.Round(float64(powerBase) * mult))
}

// shouldDeclare gates auto-bounty creation: skip if one is already open;
// otherwise declare on an identified murder OR Hostile rep.
func shouldDeclare(triggerKind crimes.Kind, tier opinions.Tier, alreadyOpen bool) bool {
	if alreadyOpen {
		return false
	}
	return triggerKind == crimes.KindMurder || tier == opinions.TierHostile
}

// Seams (production wiring; tests override).
var (
	bDefaultGoldFn      = bounties.DefaultGoldFor
	bRepFn              = factions.GetRep
	bTierFn             = factions.TierFor
	bDeclareFn          = bounties.Declare
	bExpiryRoundsFn     = func() uint64 {
		v := configs.GetBalanceConfig().JusticeBountyExpiryRounds
		if v < 1 {
			return 5000
		}
		return uint64(v)
	}
	bMurderMultFn  = func() float64 { return float64(configs.GetBalanceConfig().JusticeBountyMurderMult) }
	bRepMultMaxFn  = func() float64 { return float64(configs.GetBalanceConfig().JusticeBountyRepMultMax) }
	bNowFn         = nowRoundFn // reuse justice.go's round seam
)

// existingFactionBounty reports whether an open bounty issued by `faction`
// already targets the player.
func existingFactionBounty(faction string, userId int) bool {
	for _, b := range bounties.OpenAgainstPlayer(userId) {
		if b.Issuer.Type == bounties.IssuerFaction && b.Issuer.Id == faction {
			return true
		}
	}
	return false
}

// MaybeDeclareBounty posts a town-faction kill-bounty on a player when an
// identified murder or Hostile rep warrants it. Idempotent per (faction,
// player). Called from the crime-recording sites after their rep hit.
func MaybeDeclareBounty(faction string, userId int, triggerKind crimes.Kind) {
	tier := bTierFn(faction, userId)
	if !shouldDeclare(triggerKind, tier, existingFactionBounty(faction, userId)) {
		return
	}
	gold := bountyGold(
		bDefaultGoldFn(knowledge.PlayerSubject(userId)),
		triggerKind == crimes.KindMurder,
		bRepFn(faction, userId),
		bMurderMultFn(), bRepMultMaxFn(),
	)
	reason := fmt.Sprintf("Crimes against %s", faction)
	if triggerKind == crimes.KindMurder {
		reason = fmt.Sprintf("Murder (faction %s)", faction)
	}
	_, _ = bDeclareFn(
		bounties.FactionIssuer(faction),
		knowledge.PlayerSubject(userId),
		bounties.ConditionKill,
		bNowFn()+bExpiryRoundsFn(),
		bounties.DeclareOpts{GoldOverride: gold, DeclaredReason: reason},
	)
}
```
(If `nowRoundFn` from `justice.go` isn't accessible/named as expected, define `bNowFn = util.GetRoundCount` and import util.)

- [ ] **Step 4** — Add a declare-capture test to `bounty_test.go`:
```go
func TestMaybeDeclareBounty_MurderDeclares(t *testing.T) {
	origTier, origRep, origGold, origDeclare, origNow, origMM, origRM :=
		bTierFn, bRepFn, bDefaultGoldFn, bDeclareFn, bNowFn, bMurderMultFn, bRepMultMaxFn
	defer func() {
		bTierFn, bRepFn, bDefaultGoldFn, bDeclareFn, bNowFn, bMurderMultFn, bRepMultMaxFn =
			origTier, origRep, origGold, origDeclare, origNow, origMM, origRM
	}()
	bTierFn = func(string, int) opinions.Tier { return opinions.TierNeutral }
	bRepFn = func(string, int) int { return 0 }
	bDefaultGoldFn = func(knowledge.Subject) int { return 100 }
	bMurderMultFn = func() float64 { return 2.0 }
	bRepMultMaxFn = func() float64 { return 2.0 }
	bNowFn = func() uint64 { return 1000 }
	// existingFactionBounty hits the real bounties registry; ensure none open
	// for this fresh userId in the test process.
	var gotGold int
	var gotIssuer bounties.Issuer
	bDeclareFn = func(issuer bounties.Issuer, _ knowledge.Subject, _ bounties.Condition, _ uint64, opts bounties.DeclareOpts) (int, error) {
		gotIssuer = issuer
		gotGold = opts.GoldOverride
		return 1, nil
	}
	MaybeDeclareBounty("thornwall_guards", 99017, crimes.KindMurder)
	if gotGold != 200 || gotIssuer.Id != "thornwall_guards" {
		t.Fatalf("declared gold=%d issuer=%v; want 200 / thornwall_guards", gotGold, gotIssuer.Id)
	}
}
```

- [ ] **Step 5** — `go test ./internal/justice/` → PASS.

- [ ] **Step 6** — Commit:
```bash
git add internal/justice/bounty.go internal/justice/bounty_test.go
git commit -m "feat(justice): MaybeDeclareBounty (murder/Hostile trigger, scaled reward)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: Fire `MaybeDeclareBounty` from crime sites

**Read each file first** (env was flaky during plan-prep) and place the call immediately after the crime is recorded + rep bumped, once per affected faction id.

- [ ] **Step 1 — murder (`internal/hooks/MobDeath_FactionRep.go`)**: in each branch that records/upgrades an *identified* murder for faction `fid` against player `userId` (the branches that call `factions.BumpRep(fid, userId, ...)` with an identified perp — Case A and the fresh-murder identified path), add after the rep bump:
```go
			justice.MaybeDeclareBounty(fid, userId, crimes.KindMurder)
```
Add the `justice` import. Do NOT add it to the unknown-perp branches (Case C / unknown fresh murder) — there's no identified target.

- [ ] **Step 2 — assault (`internal/usercommands/attack.go`)**: in `recordAssaultCrime`, immediately after `factions.BumpRep(fid, user.UserId, delta)` (≈line 351, inside `if perp.Type == crimes.PerpPlayer` within the `for _, fid := range factionIds` loop), add:
```go
				justice.MaybeDeclareBounty(fid, user.UserId, crimes.KindAssault)
```

- [ ] **Step 3** — Add the `justice` import to both files, then `go build ./...` (expect clean). If in-scope names differ, use the actual ones; if no post-rep anchor exists, STOP and report BLOCKED.

(Theft is intentionally NOT wired — it lives in `internal/actions`, which would form an `actions→justice` import cycle. Murder exercises the murder-trigger; assault + accumulated rep exercise the rep-trigger.)

- [ ] **Step 4** — Commit:
```bash
git add internal/hooks/MobDeath_FactionRep.go internal/usercommands/attack.go
git commit -m "feat(justice): fire MaybeDeclareBounty from murder + assault sites

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: `PlayerDeath_BountyResolve`

**Files:** Create `internal/hooks/PlayerDeath_BountyResolve.go`, `internal/hooks/PlayerDeath_BountyResolve_test.go`

- [ ] **Step 1 — failing test (pure attribution)** (`..._test.go`):
```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state"
)

func TestAttributeBountyKill(t *testing.T) {
	// guard mob of the issuer faction landed the kill
	gk := attributeBountyKill(
		state.ActorRef{MobInstanceId: 5},
		"thornwall_guards", nil,
		func(int) []string { return []string{"thornwall_guards"} }, // killer mob factions
	)
	if gk.kind != killGuard || gk.mobInstanceId != 5 {
		t.Errorf("guard kill: got %+v", gk)
	}
	// third-party player landed the kill
	pk := attributeBountyKill(
		state.ActorRef{UserId: 42},
		"thornwall_guards", nil, nil,
	)
	if pk.kind != killPlayer || pk.userId != 42 {
		t.Errorf("player kill: got %+v", pk)
	}
	// non-issuer mob, but a player damager exists -> player attribution
	dm := attributeBountyKill(
		state.ActorRef{MobInstanceId: 7},
		"thornwall_guards", map[int]int{42: 10},
		func(int) []string { return []string{"warren"} }, // killer mob not in issuer faction
	)
	if dm.kind != killPlayer || dm.userId != 42 {
		t.Errorf("damager fallback: got %+v", dm)
	}
	// nobody eligible -> expire
	ex := attributeBountyKill(state.ActorRef{}, "thornwall_guards", nil, nil)
	if ex.kind != killNone {
		t.Errorf("none: got %+v", ex)
	}
}
```

- [ ] **Step 2** — `go test ./internal/hooks/ -run TestAttributeBountyKill` → FAIL.

- [ ] **Step 3** — Create `internal/hooks/PlayerDeath_BountyResolve.go`:
```go
package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/bounties"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/users"
)

type bountyKillKind int

const (
	killNone   bountyKillKind = iota
	killGuard                 // issuing-faction guard mob landed the kill
	killPlayer                // third-party player (killing blow or top damager)
)

type bountyKill struct {
	kind          bountyKillKind
	mobInstanceId int // when killGuard
	userId        int // when killPlayer
}

// attributeBountyKill decides who, if anyone, claims a bounty when the target
// dies. issuerFaction is the bounty's issuing faction; killerFactions resolves a
// mob instance's faction memberships (nil ok). Pure.
func attributeBountyKill(killer state.ActorRef, issuerFaction string, damageMap map[int]int, killerFactions func(instanceId int) []string) bountyKill {
	if killer.IsMob() && killer.MobInstanceId > 0 && killerFactions != nil {
		for _, f := range killerFactions(killer.MobInstanceId) {
			if f == issuerFaction {
				return bountyKill{kind: killGuard, mobInstanceId: killer.MobInstanceId}
			}
		}
	}
	if killer.IsPlayer() && killer.UserId > 0 {
		return bountyKill{kind: killPlayer, userId: killer.UserId}
	}
	// Fallback: highest player damager (covers player kills not captured as the
	// killing-blow ActorRef).
	top, topDmg := 0, 0
	for uid, dmg := range damageMap {
		if dmg > topDmg {
			top, topDmg = uid, dmg
		}
	}
	if top > 0 {
		return bountyKill{kind: killPlayer, userId: top}
	}
	return bountyKill{kind: killNone}
}

func killerMobFactions(instanceId int) []string {
	inst := mobs.GetInstance(instanceId)
	if inst == nil {
		return nil
	}
	return factions.FactionsForMob(inst)
}

func wirePlayerDeathBountyResolve(c *characters.Character) {
	c.Life.Inner().AfterTransition("player_death_bounty_resolve",
		func(from, to life.State, r state.TransitionReason) {
			if from != life.Alive || to != life.Dead || c.GetUserId() == 0 {
				return
			}
			userId := c.GetUserId()
			open := bounties.OpenAgainstPlayer(userId)
			if len(open) == 0 {
				return
			}
			d, _ := c.Life.DeadData()
			for _, b := range open {
				issuerFaction := ""
				if b.Issuer.Type == bounties.IssuerFaction {
					issuerFaction = b.Issuer.Id
				}
				bk := attributeBountyKill(d.Killer, issuerFaction, d.DamageMap, killerMobFactions)
				switch bk.kind {
				case killGuard:
					inst := mobs.GetInstance(bk.mobInstanceId)
					if inst == nil {
						bounties.MarkExpired(b.Id)
						continue
					}
					if _, ok := bounties.TryClaim(b.Id, knowledge.MobSubject(int(inst.MobId))); ok {
						inst.Character.Gold += b.GoldReward
					}
				case killPlayer:
					killer := users.GetByUserId(bk.userId)
					if killer == nil {
						bounties.MarkExpired(b.Id)
						continue
					}
					if claimed, ok := bounties.TryClaim(b.Id, knowledge.PlayerSubject(bk.userId)); ok {
						killer.Character.Gold += claimed.GoldReward
						if claimed.Issuer.Type == bounties.IssuerFaction {
							factions.BumpRep(claimed.Issuer.Id, bk.userId, claimed.RepReward)
						}
						killer.SendText(messaging.CategoryLoot, fmt.Sprintf("You collect a bounty: %dg.\r\n", claimed.GoldReward))
					}
				default:
					bounties.MarkExpired(b.Id)
				}
			}
		})
}

func init() {
	characters.OnCharacterCreated(wirePlayerDeathBountyResolve)
}
```

- [ ] **Step 4** — `go test ./internal/hooks/ -run TestAttributeBountyKill` → PASS; `go build ./...` clean.

- [ ] **Step 5** — Commit:
```bash
git add internal/hooks/PlayerDeath_BountyResolve.go internal/hooks/PlayerDeath_BountyResolve_test.go
git commit -m "feat(justice): resolve player bounties on death (guard/player/expire)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: Verification

- [ ] **Step 1** — `go build ./...`; `go vet ./internal/justice/ ./internal/bounties/ ./internal/hooks/` (clean).
- [ ] **Step 2** — `go test ./internal/justice/ ./internal/bounties/ ./internal/configs/` and `go test ./internal/hooks/ -run 'AttributeBountyKill|IsGuardMob|RecomputeGoals'` (ok; note the pre-existing flaky `TestHandlePlayerFoldCasting_*`).
- [ ] **Step 3** — Boot smoke (instance wipe per SOP): `rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*`, `go run .` in background; confirm `Config name="Balance.JusticeBountyExpiryRounds" value=5000` (+ the two mults), no panic, `Server Ready`; then `taskkill //IM "GoMud.exe" //F`.

---

## Notes for the implementer

- **Env caveat:** the controller's plan-prep hit transient tool-output errors, so the exact insertion lines in `attack.go` / `steal.go` / `MobDeath_FactionRep.go` were not byte-confirmed. Read each file before editing and place the one-line `MaybeDeclareBounty` call at the documented anchor (after the crime record + rep bump, per faction). Report BLOCKED if no such anchor exists rather than guessing.
- **Dedup uses the live registry** (`existingFactionBounty` → `bounties.OpenAgainstPlayer`), so the murder/declare test uses a fresh userId (99017) to avoid cross-test residue.
- **Death clears the bounty, not the wanted status** — rep/crimes persist; full clearing is 5.1d. Do not add rep/crime resolution here.
- **Guard payout funds 5.3** — paying the killing guard's instance gold is intentional (future equipment-aware shopping spends it). NPC bounty-hunter *pursuit* is 5.2.
- **Forward-compat:** `MaybeDeclareBounty` is the single declaration entry point; 5.1c/5.1d and 5.2 read/clear via the same `bounties` registry.
