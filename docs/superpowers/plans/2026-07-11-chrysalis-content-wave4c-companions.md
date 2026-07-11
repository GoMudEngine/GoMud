# Chrysalis Content — Wave 4c: companion subsystem (Brood Sac)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Establish the **passive-companion** mechanic (P8) — a mutation that keeps a bonded creature at your side — proven with **Brood Sac** (Manifester): you always have one companion; if it dies, you birth another.

**Architecture:** The engine already has a full player-companion subsystem — `resolveCompanionSummon` (`internal/hooks/companion_summon.go`) does spawn → `room.AddMob` → `Charm(user.UserId, …)` → `TrackCharmed` → `AddCompanion`; `GetMaxCompanions` caps it; `MobDeath_CompanionCleanup` auto-removes a dead companion from `Character.Companions`. Brood Sac reuses all of this: **(1)** extract the spawn-and-register steps into a reusable `spawnCharmedCompanion(user, mobId, room)` helper (and DRY `resolveCompanionSummon` onto it), then **(2)** a per-round `tickBroodSac` that, for a brood-sac owner in a room with **zero** living companions and room under cap, calls the helper to birth its spawn. Death cleanup + the cap are already handled, so respawn is emergent (companion dies → cleanup empties `Companions` → next tick re-births).

**Tech Stack:** Go — `internal/hooks` (spawn helper + tick), `internal/mutations` (flag + describe), `internal/characters` (companions); a brood-spawn mob + the mutation + help YAML; testify.

**Spec:** `docs/superpowers/specs/2026-07-11-chrysalis-cluster-content-design.md` (Wave 4c — P8, the companion half of §9 Wave 4). Builds on Waves 1–4b.

**Scope — Brood Sac only (one always-on companion).** Hive Mind (extra slots) and Brood Mother (apex swarm) are deferred to Wave 6 per-cluster authoring; they layer on the same helper (more slots / more spawns). Symbiotic Bond (buffs bleed to companions) is a later passive.

**Wave 1–4 reminders:** mutation needs a `templates/help/{id}.template` + `DescribeEffect`/`flagPhrase` entry; apply/spawn via the existing wrappers.

---

## File Structure

**Modify:**
- `internal/hooks/companion_summon.go` — extract `spawnCharmedCompanion`; refactor `resolveCompanionSummon` step 5 to call it
- `internal/mutations/describe.go` — `flagPhrase("brood-spawn")` case
- `internal/hooks/NewRound_UserRoundTick.go` — call `tickBroodSac(user, room)` in the per-player loop

**Create:**
- `internal/hooks/mutation_brood.go` — `tickBroodSac` + `broodSpawnMobId` const + test `mutation_brood_test.go`
- The brood-spawn mob: **reuse an existing low-tier creature** (see Task 3) — set `broodSpawnMobId` to its id. (Avoids authoring/validating a new mob; a dedicated brood-spawnling can replace it in Wave 6.)
- `_datafiles/world/dogmud/mutations/brood-sac.yaml` + `.../help/brood-sac.template`

---

## Phase 1 — Reusable companion spawn helper

### Task 1: Extract `spawnCharmedCompanion`

**Files:** Modify `internal/hooks/companion_summon.go`; Test `internal/hooks/companion_summon_test.go` (new, light)

- [ ] **Step 1: Add the helper**

In `internal/hooks/companion_summon.go`, add a helper that captures the existing "spawn + charm + register" block (steps 5 of `resolveCompanionSummon`, lines ~131-162), parameterized by mob id and stat pool:

```go
// spawnCharmedCompanion spawns mob `mobId` in `room`, charms it permanently to
// `user`, and registers it as a companion. Returns the new mob instance id, or
// 0 on failure (caller checks the cap first). Shared by the summon spells and
// the Brood Sac mutation.
func spawnCharmedCompanion(user *users.UserRecord, mobId int, pool int, source characters.CompanionSourceType) int {
	room := rooms.LoadRoom(user.Character.RoomId)
	if room == nil {
		return 0
	}
	mob := mobs.NewMobByIdFresh(mobs.MobId(mobId), room.RoomId, pool)
	if mob == nil {
		return 0
	}
	room.AddMob(mob.InstanceId)
	mob.Character.Charm(user.UserId, 99999, "")
	mob.Character.EndAggro()
	user.Character.TrackCharmed(mob.InstanceId, true)
	info := characters.CompanionInfo{
		MobId:      int(mob.MobId),
		InstanceId: mob.InstanceId,
		SourceType: source,
		Name:       mob.Character.Name,
		BaseName:   mob.Character.Name,
		AutoAssist: true,
	}
	if !user.Character.AddCompanion(info) {
		return 0
	}
	return mob.InstanceId
}
```

- [ ] **Step 2: Refactor `resolveCompanionSummon` step 5 onto the helper**

Replace the spawn/charm/register block (lines ~131-162, up to but not including the "Clear aggro from existing companions" block) with:

```go
	instanceId := spawnCharmedCompanion(user, spellData.SummonMobId, pool, func() characters.CompanionSourceType {
		if spellData.SummonRequiresCorpse {
			return characters.CompanionRaised
		}
		return characters.CompanionSummoned
	}())
	if instanceId == 0 {
		user.SendText(messaging.CategorySpellDisruption, "The summoning fails — something is wrong.")
		return false
	}
	mob := mobs.GetInstance(instanceId)
	if mob == nil {
		return false
	}
```

(Everything after — the "clear aggro from existing companions toward the new mob" and owner-aggro-clear blocks — stays as-is, using `mob`.)

- [ ] **Step 3: Build + run the existing summon/companion tests (no behavior change)**

Run: `go build ./... && go test ./internal/hooks/... ./internal/characters/...`
Expected: clean + PASS — the refactor is behavior-preserving; existing summon-spell tests still pass.

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/companion_summon.go
git commit -m "refactor(hooks): extract spawnCharmedCompanion from resolveCompanionSummon"
```

---

## Phase 2 — The Brood Sac tick

### Task 2: `tickBroodSac` + describe

**Files:** Create `internal/hooks/mutation_brood.go`, `internal/hooks/mutation_brood_test.go`; Modify `internal/mutations/describe.go`

- [ ] **Step 1: Add the flag describe case**

In `internal/mutations/describe.go`, inside `flagPhrase`'s switch:

```go
	case "brood-spawn":
		return "You host and birth a bonded creature — you always have a companion at your side."
```

- [ ] **Step 2: Write the failing test (the pure eligibility predicate)**

```go
// internal/hooks/mutation_brood_test.go
package hooks

import "testing"

func TestShouldBirthBrood(t *testing.T) {
	// has flag, alive, 0 companions, under cap → birth
	if !shouldBirthBrood(true, true, 0, 3) {
		t.Fatal("flag + alive + no companions + under cap → should birth")
	}
	if shouldBirthBrood(false, true, 0, 3) {
		t.Fatal("no flag → never")
	}
	if shouldBirthBrood(true, false, 0, 3) {
		t.Fatal("dead owner → never")
	}
	if shouldBirthBrood(true, true, 1, 3) {
		t.Fatal("already has a companion → don't add another")
	}
	if shouldBirthBrood(true, true, 3, 3) {
		t.Fatal("at cap → never")
	}
}
```

- [ ] **Step 3: Run to verify it fails**

Run: `go test ./internal/hooks/ -run TestShouldBirthBrood -v`
Expected: FAIL — `shouldBirthBrood` undefined.

- [ ] **Step 4: Implement**

In `internal/hooks/mutation_brood.go`:

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// broodSpawnMobId is the creature Brood Sac births. TODO(execution): set to a
// suitable low-tier existing mob id (see Task 3); a dedicated brood-spawnling
// can replace it in Wave 6.
const broodSpawnMobId = 0 // <-- set during Task 3

// broodBasePool is the base stat pool for a brood spawn before charisma/skill scaling.
const broodBasePool = 40

// shouldBirthBrood reports whether a brood-sac bearer should birth a spawn now:
// has the brood flag, is alive, has zero companions, and is under the cap.
func shouldBirthBrood(hasFlag, alive bool, companionCount, maxCompanions int) bool {
	return hasFlag && alive && companionCount == 0 && companionCount < maxCompanions
}

// tickBroodSac keeps a bonded spawn at a brood-sac owner's side. Respawn is
// emergent: MobDeath_CompanionCleanup empties Companions when the spawn dies,
// and the next tick re-births it.
func tickBroodSac(user *users.UserRecord) {
	ch := user.Character
	if !shouldBirthBrood(
		mutations.HasMutationFlag(ch.Mutations, "brood-spawn"),
		ch.IsAlive(),
		len(ch.Companions),
		ch.GetMaxCompanions(),
	) {
		return
	}
	if rooms.LoadRoom(ch.RoomId) == nil {
		return
	}
	pool := characters.CalcCompanionStatPool(broodBasePool, ch.Stats.Charisma.ValueAdj, ch.GetSkillLevel(skillsManifestation))
	spawnCharmedCompanion(user, broodSpawnMobId, pool, characters.CompanionSummoned)
}
```

> `skillsManifestation` — use the existing `skills.Manifestation` constant (add the `skills` import); named here only to avoid a bare string. Confirm the import alias matches the file's convention.

- [ ] **Step 5: Run to verify it passes**

Run: `go test ./internal/hooks/ -run TestShouldBirthBrood -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/hooks/mutation_brood.go internal/hooks/mutation_brood_test.go internal/mutations/describe.go
git commit -m "feat(hooks): Brood Sac tick (tickBroodSac) + describe"
```

---

### Task 3: Choose the brood mob + wire the tick

**Files:** Modify `internal/hooks/mutation_brood.go` (set `broodSpawnMobId`), `internal/hooks/NewRound_UserRoundTick.go`

- [ ] **Step 1: Pick a brood mob**

Scan for a thematically-appropriate low-tier creature to reuse as the spawn: `python tools/id_inventory.py --type mobs` and/or grep `_datafiles/world/dogmud/mobs/` for a small beast/vermin (rat, spiderling, imp, etc.). Set `broodSpawnMobId` in `mutation_brood.go` to its id. (A dedicated brood-spawnling mob can be authored in Wave 6.)

- [ ] **Step 2: Wire the tick into the per-player loop**

In `internal/hooks/NewRound_UserRoundTick.go`, in the `for _, uId := range room.GetPlayers()` block (near the other per-player mutation effects), add:

```go
					tickBroodSac(user)
```

- [ ] **Step 3: Build + hooks suite**

Run: `go build ./... && go test ./internal/hooks/...`
Expected: clean + PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/mutation_brood.go internal/hooks/NewRound_UserRoundTick.go
git commit -m "feat(hooks): drive Brood Sac from the user round tick (brood mob wired)"
```

---

## Phase 3 — Content

### Task 4: `brood-sac.yaml` + help

**Files:** Create the mutation YAML + help template.

- [ ] **Step 1: Write `_datafiles/world/dogmud/mutations/brood-sac.yaml`**

```yaml
mutationid: brood-sac
name: Brood Sac
description: |
  A pulsing sac grows against your ribs, and something lives in it. It quickens,
  it births, it follows — a bonded creature that fights at your side and is
  replaced the moment it falls. You are never quite alone.
rarity: 4
clusters: [manifester]
pole: belief
visual: A translucent, faintly-moving sac swells beneath the skin of their flank.
pros:
  - type: flag
    target: brood-spawn
```

- [ ] **Step 2: Write `_datafiles/world/dogmud/templates/help/brood-sac.template`**

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="yellow">Brood Sac</ansi> mutation

A pulsing sac against your ribs quickens and births a bonded creature
that fights at your side.

<ansi fg="yellow">Type:</ansi>     Passive (companion)
<ansi fg="yellow">Rarity:</ansi>   Uncommon

<ansi fg="yellow">Benefits:</ansi>
  You always have a bonded companion creature — if it falls, you
  birth another before long

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help mutations</ansi>
```

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/mutations/brood-sac.yaml _datafiles/world/dogmud/templates/help/brood-sac.template
git commit -m "content(mutations): brood-sac (Manifester passive-companion keystone)"
```

---

## Phase 4 — Verification

### Task 5: Build, full suite, boot smoke, manual smoke

- [ ] **Step 1: Build + affected suites**

Run: `go build ./... && go test ./internal/hooks/... ./internal/characters/... ./internal/mutations/... ./internal/devtools/...`
Expected: clean, all PASS.

- [ ] **Step 2: Full suite**

Run: `go test ./...`
Expected: 0 failing packages.

- [ ] **Step 3: Boot smoke**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . 2>&1 | grep -mE1 "mutations.LoadMutationFiles|panic:"
```
Expected: mutations load (count = 52 + 1 = 53), no panic.

- [ ] **Step 4: Manual smoke (the important one — companion lifecycle)**

Grant a test character `brood-sac`, walk around: confirm a brood companion spawns and follows; kill it (or let it die in a fight): confirm it is birthed again a round later; confirm you never exceed one brood (and never exceed `GetMaxCompanions`); log out and back in: confirm no duplicate/orphan spawns; confirm summon-spell companions still work (the refactor).

- [ ] **Step 5: Commit** (only if a fix was needed)

---

## Self-Review (completed during authoring)

- **Spec coverage:** Wave 4c P8 (passive companion) via Brood Sac, reusing the existing companion subsystem (spawn flow, cap, death-cleanup). Hive Mind / Brood Mother / Symbiotic Bond deferred to Wave 6 (they layer on `spawnCharmedCompanion`).
- **Placeholder scan:** `broodSpawnMobId = 0` is an explicit execution-time choice (Task 3, mob scan); the `skills.Manifestation` import alias is a "match the file convention" note. Every other step carries complete code. Base pool 40 is first-pass/tunable.
- **Type consistency:** `spawnCharmedCompanion(user, mobId, pool, source) int` used by both `resolveCompanionSummon` and `tickBroodSac`; `shouldBirthBrood(hasFlag, alive, count, max) bool` pure + tested; `brood-spawn` flag consistent across mutation YAML, describe, and the tick's `HasMutationFlag` check. Cap + alive + zero-companion gates prevent over-spawning; death cleanup already exists so respawn is emergent.

## Risk notes (verify in manual smoke)

- **No over-spawn:** the `companionCount == 0` gate plus the cap means at most one brood, and only when the owner has no companions at all. If the owner has a *summoned* companion, the brood won't spawn (intended — one is enough).
- **Cleanup/logout:** dead companions are pruned by `MobDeath_CompanionCleanup`; confirm logout doesn't strand an orphan mob or double-spawn on relogin (the tick only fires for in-room players).

## Follow-on

- **Wave 5:** the two actives (Venom Coat, Cocoon) + Winged Flight.
- **Wave 6:** full per-cluster authoring (cores + apexes with prereq spines — Hive Mind, Brood Mother, Symbiotic Bond here; a dedicated brood-spawnling mob), migration/re-bloom, `archetype_pull`→cluster re-curation, balance pass.
