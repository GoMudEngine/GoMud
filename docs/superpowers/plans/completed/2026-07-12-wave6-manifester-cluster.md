# Wave 6a — Manifester Cluster Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Author the Manifester cluster (the summoner/broodmaster playstyle) on the *companion-economy* design — NOT the stale content-spec §3 "Brood Sac respawn" roster, which was superseded.

**Architecture:** The two headline mechanics already exist from the Wave-5 companion economy — the reserve-reducer (`companion_reserve_reduction` → `GetCompanionReserveRank` → `CalcCompanionReserve`) and the soft-cap raise (`companion-cap-raise` flag → `GetMaxCompanions`). So the identity keystone + Hive Mind are pure YAML. The one new mechanic is **companion empowerment** — a Manifester passive that makes the owner's companions hit harder (a dedicated buff applied per-round to charmed companions in the room), plus the apex's **floor-companion** respawn (reuses the Homunculus tick pattern). The Brood Mother apex composes the reducer + cap + floor companion.

**⚠ Double-count caution (user-flagged):** rally and warcry **already fan their buffs out to companions** (`applyRallyToCompanions`, `rally.go:74`, applies buff 80; warcry likewise). So companion empowerment must be its **own dedicated effect** — it must NOT copy/mirror the owner's transient buff list, or it would double-apply shout buffs. This plan grants companions a distinct "Empowered by Bond" buff, never a copy of owner buffs.

**Reconciled roster** (companion-economy §6, not content-spec §3):
| Keystone | Role | Mechanic |
|---|---|---|
| Sustaining Presence (name TBD) | entry/core — identity | `companion_reserve_reduction` (BUILT — YAML only) |
| Symbiotic Bond | core | `companion_empowerment` (NEW mechanic, this plan) |
| Hive Mind | core | `flag: companion-cap-raise` (BUILT — YAML only) |
| Brood Mother ★ | apex | reducer-max + cap + floor-companion (NEW tick) — prereq Sustaining Presence + Hive Mind |
| Spirit Tether (⇄ethereal) | bridge | `companion_empowerment` variant (YAML) |
| Beast Bond (⇄zealot) | bridge | `companion_empowerment` variant (YAML) |

**Deferred (note, don't build):** companion *coordination* (Hive Mind's "act together"), regen-bleed flavor on empowerment (statmods-only MVP), per-rank magnitude tuning.

---

## Verified context (do not re-discover)

- **Reducer already wired:** `mutations.GetCompanionReserveRank` (mutations.go:538) reads `companion_reserve_reduction`; `Character.CalcCompanionReserve` (companions.go:174) applies it with config caps (24% at rank 4). A keystone with that effect just works.
- **Cap-raise already wired:** `Character.GetMaxCompanions` (companions.go:139) checks `HasMutationFlag(c.Mutations, "companion-cap-raise")` → `CompanionSoftCapApex` (7). A keystone with that flag just works.
- **Effect readers pattern:** `sumEffects(owned, type, target)` + a `GetX` helper (mutations pkg); flags via `HasMutationFlag`. `DescribeEffect`/`flagPhrase` in describe.go need a case per new type/flag or `TestDescribeEffect`/help tests fail.
- **Companion fan-out pattern** (`applyRallyToCompanions`, rally.go:74): `for _, id := range owner.Character.GetCharmIds() { mob := mobs.GetInstance(id); if mob==nil || mob.Character.RoomId != owner.Character.RoomId { continue }; mob.Character.AddBuff(...) }`.
- **Round-tick seam:** `tickHomunculus(user, room)` is called per-user in `NewRound_UserRoundTick.go` (~line 236, right after `pinnacleUserTick`). Add sibling ticks there.
- **Homunculus floor-spawn pattern** (`hooks/chrysifier_homunculus.go`): `spawnHomunculus` = NewMobByIdFresh(pool) → AddMob → charm → TrackCharmed → CompanionInfo{ConvictionReserve} → AddCompanion (check return) → RecalculateStats; `tickHomunculus` gates on flag + no-live-companion + cooldown. Mirror for the Brood Mother floor companion.
- **Buff YAML** (`buffs/101-emboldened.yaml`): `buffid`, `name`, `description`, `triggerrate`, `triggercount`, `statmods:`, `start_user_text`, `end_user_text`. **Next free buff id: 105.**
- **Mutation YAML:** `clusters: [manifester]`, `pole: belief` (Manifester is Belief pole), `prerequisites: [{id, min_level}]`, `pros:`. Bridges use `clusters: [manifester, <neighbor>]`. Boot `ValidateGraph` panics on unknown cluster (manifester IS in KnownClusters) or missing prereq id. Help template per mutation required.

---

## File Structure
- `internal/mutations/manifester.go` (+ `_test.go`) — `GetCompanionEmpowerment` reader.
- `internal/mutations/describe.go` — cases for `companion_empowerment` + any new flag.
- `internal/hooks/manifester_companions.go` (+ `_test.go`) — `tickCompanionEmpowerment` + `tickBroodMotherFloor` + pure helpers.
- `internal/hooks/NewRound_UserRoundTick.go` — register the two ticks.
- `_datafiles/world/dogmud/buffs/105-empowered_by_bond.yaml`.
- `_datafiles/world/dogmud/mobs/summons/9613-brood_spawn.yaml` — the apex floor companion.
- `_datafiles/world/dogmud/mutations/` — sustaining-presence, symbiotic-bond, hive-mind, brood-mother, spirit-tether, beast-bond (+ help templates).

---

### Task 1: Companion-empowerment effect reader + descriptor

**Files:** `internal/mutations/manifester.go`, `_test.go`; `internal/mutations/describe.go`

- [ ] **Step 1: Failing test** — seed a mutation with `companion_empowerment` (value 0.15) and assert `GetCompanionEmpowerment(owned)` returns 0.15; unrelated mutation returns 0.

- [ ] **Step 2: verify fails.**

- [ ] **Step 3: Implement** (`manifester.go`):

```go
package mutations

// GetCompanionEmpowerment returns the net companion-empowerment magnitude
// (Symbiotic Bond + the Manifester bridges) — how much the owner's companions
// are strengthened. This is a DEDICATED effect, distinct from the owner's own
// buffs; companion empowerment is applied as its own buff (see the round-tick),
// never by copying the owner's transient buffs — which would double-count the
// rally/warcry buffs those shouts already fan out to companions.
func GetCompanionEmpowerment(owned map[string]int) float64 {
	return sumEffects(owned, "companion_empowerment", "")
}
```

Add `DescribeEffect` case:

```go
	case "companion_empowerment":
		return "The bond you share with your companions makes them fight harder at your side."
```

- [ ] **Step 4: verify passes** (incl. `TestDescribeEffect`). **Step 5: commit** `feat(manifester): companion_empowerment effect reader + descriptor`.

---

### Task 2: "Empowered by Bond" buff (105)

**Files:** `_datafiles/world/dogmud/buffs/105-empowered_by_bond.yaml`

- [ ] **Step 1: Author** — a short-lived buff (refreshed each round by the tick) that makes a companion hit harder + tougher. Statmods only for the MVP (regen-bleed deferred). Number-free text.

```yaml
buffid: 105
name: Empowered by Bond
description: The bond with your manifester lends this companion greater strength and resilience.
triggerrate: 1 round
triggercount: 3
statmods:
  strength: 10
  vitality: 10
start_user_text: You feel your bond-master's will surge through you.
end_user_text: The bond's strength ebbs away.
```

- [ ] **Step 2: Boot-load check** (buffs validate at load — a quick boot, or fold into a later boot). **Step 3: commit** `content(manifester): Empowered by Bond buff (105)`.

---

### Task 3: `tickCompanionEmpowerment` round-tick

**Files:** `internal/hooks/manifester_companions.go`, `_test.go`; `internal/hooks/NewRound_UserRoundTick.go`

- [ ] **Step 1: Implement the tick** — mirrors `applyRallyToCompanions`, but passive (fires while the owner holds a `companion_empowerment` mutation) and applies the DEDICATED buff 105, never a copy of the owner's buffs:

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

const empoweredByBondBuff = 105

// tickCompanionEmpowerment refreshes the "Empowered by Bond" buff on the owner's
// in-room companions each round, while the owner holds a companion_empowerment
// mutation (Symbiotic Bond / Spirit Tether / Beast Bond). This is a DEDICATED
// empowerment — it does NOT mirror the owner's own buffs, so it cannot
// double-count the rally/warcry buffs those shouts already fan out to companions.
func tickCompanionEmpowerment(user *users.UserRecord, room *rooms.Room) {
	if room == nil {
		return
	}
	if mutations.GetCompanionEmpowerment(user.Character.Mutations) <= 0 {
		return
	}
	for _, id := range user.Character.GetCharmIds() {
		mob := mobs.GetInstance(id)
		if mob == nil || mob.Character.RoomId != user.Character.RoomId {
			continue
		}
		mob.Character.AddBuff(empoweredByBondBuff, false)
	}
}
```

> Confirm the exact `Character.AddBuff` signature used by `applyRallyToCompanions` (`mob.Character.AddBuff(80, false)`) and match it. If the codebase prefers the event-safe actor path for companions, note that rally uses the raw `Character.AddBuff` here — match rally's approach for consistency.

- [ ] **Step 2: Register** in `NewRound_UserRoundTick.go` next to `tickHomunculus`:

```go
				tickHomunculus(user, room)
				tickCompanionEmpowerment(user, room)
```

- [ ] **Step 3: Test** — a small test asserting the gate: owner WITHOUT a `companion_empowerment` mutation → `GetCompanionEmpowerment` is 0 (tick no-ops); with it → >0. (Full buff-application needs a live mob/room; the gate + reader are the unit-testable part — mirror how the Wave-4 aura tests scope this.)

- [ ] **Step 4: build + test. Step 5: commit** `feat(manifester): companion empowerment round-tick (no shout double-count)`.

---

### Task 4: Sustaining Presence (reducer keystone) + Hive Mind — pure YAML

**Files:** `_datafiles/world/dogmud/mutations/{sustaining-presence,hive-mind}.yaml` + help

- [ ] **Sustaining Presence** (entry/core, the identity) — `clusters: [manifester]`, `pole: belief`, rarity ~4, effect `companion_reserve_reduction` (already consumed by the economy — this is what the economy was built to receive). Value per the economy's per-rank intent (rank drives the 6%/rank via LevelMultiplier — confirm the value so rank-4 lands near the 24% cap; check `LevelMultiplier` + `CalcCompanionReserve`'s `CompanionReserveMutPctPerRank` interaction and set the effect value accordingly).

```yaml
mutationid: sustaining-presence
name: Sustaining Presence
description: |
  Your very presence sustains the things you call forth. The conviction it
  takes to keep a companion at your side eases the deeper this grows -- a
  competent summoner becomes a broodmaster.
rarity: 4
clusters: [manifester]
pole: belief
pros:
  - type: companion_reserve_reduction
    value: 1
```

> **Verify the reduction math:** `GetCompanionReserveRank` returns the owned *level*; `CalcCompanionReserve` computes `mutRank * CompanionReserveMutPctPerRank` (0.06). So the effect `value` is irrelevant to the reduction (rank drives it) — `value: 1` is just a presence marker. Confirm this against the code so the keystone actually reduces cost per rank; adjust if the reader changes.

- [ ] **Hive Mind** (core) — `clusters: [manifester]`, `pole: belief`, rarity ~6, `flag: companion-cap-raise` (already consumed by GetMaxCompanions). Prereq Sustaining Presence.

```yaml
mutationid: hive-mind
name: Hive Mind
description: |
  A shared nerve-web binds you to your brood; you can hold more of them in
  your thoughts at once. (They do not yet act in true concert -- that is a
  deeper attunement still to come.)
rarity: 6
clusters: [manifester]
pole: belief
prerequisites:
  - id: sustaining-presence
    min_level: 1
pros:
  - type: flag
    target: companion-cap-raise
    value: 1
```

- [ ] Help templates for both (80-col, number-free). **Boot smoke** (prereqs + cluster resolve). **Commit** `content(manifester): Sustaining Presence + Hive Mind (reuse built economy hooks)`.

---

### Task 5: Symbiotic Bond + the two bridges — YAML

**Files:** `_datafiles/world/dogmud/mutations/{symbiotic-bond,spirit-tether,beast-bond}.yaml` + help

- [ ] **Symbiotic Bond** (core) — `clusters: [manifester]`, `pole: belief`, rarity ~5, prereq Sustaining Presence, effect `companion_empowerment` (value ~0.15).
- [ ] **Spirit Tether** (bridge) — `clusters: [ethereal, manifester]`, `pole: belief`, rarity 6, `companion_empowerment` (flavored: a bonded-spirit organ). No prereq (bridges draw from either neighbor).
- [ ] **Beast Bond** (bridge) — `clusters: [manifester, zealot]`, `pole: belief`, rarity 6, `companion_empowerment` (flavored: extend command to companions). No prereq.
- [ ] Help templates. **Commit** `content(manifester): Symbiotic Bond + Spirit Tether + Beast Bond bridges`.

> All three reuse `companion_empowerment` → buff 105 via Task 3's tick. Differentiate by flavor + magnitude in playtest; the mechanic is shared and does not double-count shouts.

---

### Task 6: Brood Mother apex — floor-companion tick + mob + mutation

**Files:** `internal/hooks/manifester_companions.go` (add `tickBroodMotherFloor`); `_datafiles/world/dogmud/mobs/summons/9613-brood_spawn.yaml`; `_datafiles/world/dogmud/mutations/brood-mother.yaml` + help

- [ ] **Floor-companion mob** (`9613-brood_spawn.yaml`) — a small, cheap brood spawn (a weak humanoid/beast; low statpool). Mirror the steppe-spirit template.

- [ ] **`tickBroodMotherFloor(user, room)`** — mirror `tickHomunculus`: if the owner holds the Brood Mother apex (a dedicated flag, e.g. `brood-mother`, OR reuse a check on the mutation id) AND has NO live companion at all AND a respawn cooldown has elapsed → spawn a cheap brood spawn (low `ConvictionReserve`) so a deep Manifester is never petless. Reuse `spawnHomunculus`'s structure (NewMobByIdFresh(low pool) → charm → TrackCharmed → CompanionInfo{low reserve} → AddCompanion check → RecalculateStats). Register the tick in UserRoundTick.

> "No live companion at all" — reuse the `hasLiveHomunculus` scan generalized to "any live companion" (iterate `Companions`, check `mobs.GetInstance`). Only the FLOOR (petless) triggers it, so it never competes with the Manifester's real brood.

- [ ] **Brood Mother apex mutation** (`brood-mother.yaml`) — `clusters: [manifester]`, `pole: belief`, rarity 9, prereq `sustaining-presence` + `hive-mind`. Carries: `companion_reserve_reduction` (stacks the reducer toward max at apex), the `companion-cap-raise` flag (keeps the raised cap), and a `flag: brood-mother` that `tickBroodMotherFloor` reads.

- [ ] Help template. **Build + boot smoke** (apex prereqs + brood mob load). **Commit** `feat(manifester): Brood Mother apex + floor-companion respawn`.

---

### Task 7: Coverage + suite + patch notes + boot

- [ ] `go test ./... -run 'TestHelpFileCompleteness_Mutations|TestDescribeEffect'` — fix any gap.
- [ ] `go test ./...` — exit 0.
- [ ] `PATCH_NOTES.md` — dated, player-facing: the Manifester path — companions cost less to sustain the deeper your bond, you can hold more of them, they fight harder at your side, and the truest broodmasters are never without one.
- [ ] Nuke instances, full boot smoke (clean load, mapper errors=0, Server Ready, no panic).
- [ ] **Commit** `docs(manifester): patch notes` + any coverage fixes.

---

## Out of scope / follow-on
- **Companion coordination** (Hive Mind's "act in concert") — a combat-AI behavior pass, deferred.
- **Regen-bleed** on empowerment (statmods-only MVP).
- **Per-rank magnitudes** (empowerment value, reducer curve validation) — Wave-6 playtest.
- **Final keystone names** (Sustaining Presence in particular) — pin at authoring.
- The remaining 6a clusters (Ironhide, Trickster, apexes, bridges) — separate chunks.
