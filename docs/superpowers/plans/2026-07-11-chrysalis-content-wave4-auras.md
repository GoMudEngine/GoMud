# Chrysalis Content — Wave 4: aura recipe (Commanding Presence)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Establish the **aura** mechanic (P7) — a mutation that, each combat round, applies a buff to *other* characters in the room — proven with **Commanding Presence** (Zealot): an ally-empowering presence.

**Architecture:** An aura is a new mutation effect `type: aura_ally_buff` (value = buff id). A once-per-room pass in the existing user round-tick loop finds room occupants who own an ally-aura and, while that owner is in combat, applies the aura buff to the *other* players in the room via their `AddBuff` wrapper (event queue → start text + GMCP; the `ApplyBuffs` handler already guards start-text on refresh, so re-application each round is silent). The aura buff is short-lived, so it lapses a round or two after the owner leaves or the fight ends.

**Tech Stack:** Go — `internal/mutations`, `internal/hooks` (user round tick), `internal/rooms` (room membership); YAML buff + mutation + help; testify.

**Spec:** `docs/superpowers/specs/2026-07-11-chrysalis-cluster-content-design.md` (Wave 4 of §9). Builds on Waves 1–3 (merged).

**Scope — this wave is the ally-buff aura only.** Deferred to **Wave 4b**: the *enemy-debuff* aura variant (Dissonance Organ — same room-scan tick, targets hostile mobs, applies a debuff) and the whole **companion/brood subsystem (P8** — Brood Sac / Hive Mind / Brood Mother), which extends the sizeable existing `Character.Companions` + `actSummonCompanion` machinery and warrants its own plan. One clean ally-aura exemplar proves the P7 room-scan recipe; the debuff variant is a small delta on it.

**Wave 1–3 reminders:** every mutation needs a `templates/help/{id}.template` (completeness test) + a `DescribeEffect` case per effect it uses; apply buffs via the `AddBuff` wrapper (not raw `Character.AddBuff`) so start-text + GMCP fire (Wave 2/3 lesson).

---

## File Structure

**Create:**
- `internal/mutations/aura.go` — `GetAllyAuraBuffs` helper + test `aura_test.go`
- `internal/hooks/mutation_aura.go` — `applyRoomAllyAuras(room)` + test `mutation_aura_test.go`
- `_datafiles/world/dogmud/buffs/101-emboldened.yaml` — the ally aura buff
- `_datafiles/world/dogmud/mutations/commanding-presence.yaml` + `.../help/commanding-presence.template`

**Modify:**
- `internal/mutations/describe.go` — `aura_ally_buff` describe case
- `internal/hooks/NewRound_UserRoundTick.go` — call `applyRoomAllyAuras(room)` in the room loop

---

## Phase 1 — The aura declaration

### Task 1: `GetAllyAuraBuffs` helper + describe

**Files:** Create `internal/mutations/aura.go`, `internal/mutations/aura_test.go`; Modify `internal/mutations/describe.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/mutations/aura_test.go
package mutations

import "testing"

func TestGetAllyAuraBuffs(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"commanding-presence": {MutationId: "commanding-presence", Name: "Commanding Presence", Rarity: 4,
			Pros: []MutationEffect{{Type: "aura_ally_buff", Value: 101}}},
		"plain": {MutationId: "plain", Name: "Plain", Rarity: 2,
			Pros: []MutationEffect{{Type: "stat_flat", Target: "charisma", Value: 5}}},
	})
	defer cleanup()

	got := GetAllyAuraBuffs(map[string]int{"commanding-presence": 1, "plain": 1})
	if len(got) != 1 || got[0] != 101 {
		t.Fatalf("GetAllyAuraBuffs = %v, want [101]", got)
	}
	if len(GetAllyAuraBuffs(map[string]int{})) != 0 {
		t.Fatal("no mutations → no auras")
	}
}

func TestDescribeEffect_AuraAllyBuff(t *testing.T) {
	if DescribeEffect(MutationEffect{Type: "aura_ally_buff", Value: 101}) == "" {
		t.Fatal("aura_ally_buff must have a non-empty description")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/mutations/ -run "TestGetAllyAuraBuffs|TestDescribeEffect_AuraAllyBuff" -v`
Expected: FAIL — `GetAllyAuraBuffs` undefined; describe returns "".

- [ ] **Step 3: Implement**

In `internal/mutations/aura.go`:

```go
package mutations

// GetAllyAuraBuffs returns the buff ids that owned mutations project onto
// nearby allies (effect type "aura_ally_buff", Value = buff id).
func GetAllyAuraBuffs(owned map[string]int) []int {
	var out []int
	for id := range owned {
		spec := GetMutation(id)
		if spec == nil {
			continue
		}
		for _, p := range spec.Pros {
			if p.Type == "aura_ally_buff" && p.Value > 0 {
				out = append(out, int(p.Value))
			}
		}
	}
	return out
}
```

In `internal/mutations/describe.go`, add a case to the `DescribeEffect` switch:

```go
	case "aura_ally_buff":
		return "Your presence steadies and emboldens allies who fight beside you."
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/mutations/ -run "TestGetAllyAuraBuffs|TestDescribeEffect_AuraAllyBuff" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/mutations/aura.go internal/mutations/aura_test.go internal/mutations/describe.go
git commit -m "feat(mutations): aura_ally_buff effect type — GetAllyAuraBuffs + describe"
```

---

### Task 2: The Emboldened ally buff

**Files:** Create `_datafiles/world/dogmud/buffs/101-emboldened.yaml`

Confirm 101 is free first: `python tools/id_inventory.py --type buffs` (this plan uses 101 — 100 is Blood Frenzy from Wave 3).

- [ ] **Step 1: Write the buff (model statmods on `25-deaths_shadow.yaml`)**

```yaml
buffid: 101
name: Emboldened
description: A commanding presence at your side steadies your hand and stiffens your spine.
triggerrate: 1 round
triggercount: 2
statmods:
  strength: 5
  willpower: 5
start_user_text: You stand a little taller — someone at your side is worth following.
end_user_text: The steadying presence fades, and the fight feels lonelier.
```

> `triggercount: 2` gives a 2-round life so it lapses once the aura stops refreshing it (owner left / fight ended). Modest statmods (first-pass, tunable) rather than the strong `damage-bonus` flag, since an aura hits every ally every round.

- [ ] **Step 2: Commit**

```bash
git add _datafiles/world/dogmud/buffs/101-emboldened.yaml
git commit -m "content(buffs): Emboldened ally-aura buff"
```

---

## Phase 2 — The aura tick

### Task 3: `applyRoomAllyAuras`

**Files:** Create `internal/hooks/mutation_aura.go`, `internal/hooks/mutation_aura_test.go`

- [ ] **Step 1: Write the failing test (the pure recipient-selection helper)**

```go
// internal/hooks/mutation_aura_test.go
package hooks

import "testing"

// auraRecipients returns the ids to buff: everyone in the room except the owner.
func TestAuraRecipients(t *testing.T) {
	got := auraRecipients([]int{1, 2, 3}, 2)
	if len(got) != 2 || got[0] != 1 || got[1] != 3 {
		t.Fatalf("auraRecipients = %v, want [1 3] (owner 2 excluded)", got)
	}
	if len(auraRecipients([]int{5}, 5)) != 0 {
		t.Fatal("a lone owner buffs nobody")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/hooks/ -run TestAuraRecipients -v`
Expected: FAIL — `auraRecipients` undefined.

- [ ] **Step 3: Implement**

In `internal/hooks/mutation_aura.go`:

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// auraRecipients returns the player ids in a room that an aura owner projects
// onto: everyone present except the owner.
func auraRecipients(playerIds []int, ownerId int) []int {
	out := make([]int, 0, len(playerIds))
	for _, id := range playerIds {
		if id != ownerId {
			out = append(out, id)
		}
	}
	return out
}

// applyRoomAllyAuras applies each in-combat ally-aura owner's buff to the other
// players in the room. Buffs go through the user AddBuff wrapper (start text +
// GMCP; silent on refresh), and are short-lived so they lapse when the aura
// owner leaves or the fight ends.
func applyRoomAllyAuras(room *rooms.Room) {
	playerIds := room.GetPlayers()
	if len(playerIds) < 2 {
		return // an aura needs an owner and at least one ally
	}
	for _, ownerId := range playerIds {
		owner := users.GetByUserId(ownerId)
		if owner == nil || !owner.Character.IsInCombat() {
			continue
		}
		buffIds := mutations.GetAllyAuraBuffs(owner.Character.Mutations)
		if len(buffIds) == 0 {
			continue
		}
		for _, rid := range auraRecipients(playerIds, ownerId) {
			ally := users.GetByUserId(rid)
			if ally == nil {
				continue
			}
			for _, buffId := range buffIds {
				ally.AddBuff(buffId, "aura")
			}
		}
	}
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/hooks/ -run TestAuraRecipients -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/mutation_aura.go internal/hooks/mutation_aura_test.go
git commit -m "feat(hooks): ally-aura room pass (applyRoomAllyAuras)"
```

---

### Task 4: Wire the aura pass into the room loop

**Files:** Modify `internal/hooks/NewRound_UserRoundTick.go`

- [ ] **Step 1: Add the call**

In `NewRound_UserRoundTick.go`, inside the `for _, roomId := range roomsWithPlayers` block, right after `room.RoundTick()` (grep the file for `room.RoundTick()` — it's the top of the per-room work), add:

```go
			applyRoomAllyAuras(room)
```

- [ ] **Step 2: Build + hooks suite**

Run: `go build ./... && go test ./internal/hooks/...`
Expected: clean + PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/hooks/NewRound_UserRoundTick.go
git commit -m "feat(hooks): drive ally auras from the room round tick"
```

---

## Phase 3 — Content

### Task 5: `commanding-presence.yaml` + help

**Files:** Create the mutation YAML + help template.

- [ ] **Step 1: Write `_datafiles/world/dogmud/mutations/commanding-presence.yaml`**

```yaml
mutationid: commanding-presence
name: Commanding Presence
description: |
  Something in your bearing — a scent, a stillness, a weight to your voice —
  makes those who fight alongside you stand straighter and swing surer. You
  are the kind of thing others follow into a bad situation.
rarity: 4
clusters: [zealot]
pole: belief
visual: They hold themselves with an unhurried certainty that makes a room orient toward them.
pros:
  - type: aura_ally_buff
    value: 101
```

- [ ] **Step 2: Write `_datafiles/world/dogmud/templates/help/commanding-presence.template`**

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="yellow">Commanding Presence</ansi> mutation

Something in your bearing makes those who fight alongside you stand
straighter and swing surer.

<ansi fg="yellow">Type:</ansi>     Passive (aura)
<ansi fg="yellow">Rarity:</ansi>   Uncommon

<ansi fg="yellow">Benefits:</ansi>
  While you fight, allies in the room with you are emboldened — their
  bodies and wills are steadied by your presence

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help mutations</ansi>
```

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/mutations/commanding-presence.yaml _datafiles/world/dogmud/templates/help/commanding-presence.template
git commit -m "content(mutations): commanding-presence (Zealot ally-aura keystone)"
```

---

## Phase 4 — Verification

### Task 6: Build, full suite, boot smoke, manual smoke

- [ ] **Step 1: Build + affected suites**

Run: `go build ./... && go test ./internal/mutations/... ./internal/hooks/... ./internal/devtools/...`
Expected: clean, all PASS (devtools help-completeness passes with the new help file).

- [ ] **Step 2: Full suite** (catches cross-package completeness)

Run: `go test ./...`
Expected: 0 failing packages.

- [ ] **Step 3: Boot smoke**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . 2>&1 | grep -mE1 "mutations.LoadMutationFiles|buffSpec.LoadDataFiles|panic:"
```
Expected: mutations + buffs load (mutation count = 50 + 1 = 51), no panic. Ctrl-C after world load.

- [ ] **Step 4: Manual smoke (party of two)**

Two test characters in one room; grant one `commanding-presence`. Enter combat: confirm the OTHER character receives the Emboldened buff ("You stand a little taller…") and it shows in their conditions/GMCP; confirm the owner does NOT buff themselves; confirm it lapses a round or two after combat ends.

- [ ] **Step 5: Commit** (only if a fix was needed)

---

## Self-Review (completed during authoring)

- **Spec coverage:** Wave 4 P7 (aura) via Commanding Presence, end-to-end (effect type → helper → buff → room-scan tick → content). Enemy-debuff auras and the companion subsystem (P8) explicitly deferred to Wave 4b with rationale (companions extend a large existing subsystem).
- **Placeholder scan:** buff id 101 flagged to-confirm-free; one wiring spot ("grep for `room.RoundTick()`") is a local-anchor instruction; every other step carries complete code. Statmods are first-pass/tunable per the spec.
- **Type consistency:** `GetAllyAuraBuffs(owned) []int`, `auraRecipients(playerIds, ownerId) []int`, `applyRoomAllyAuras(room *rooms.Room)` consistent across impl/test/call site; `aura_ally_buff` effect string consistent across YAML, helper, describe. Buff applied via `ally.AddBuff` wrapper (start text + GMCP). Owner-exclusion + in-combat gate handled. New mutation has help + describe.

## Follow-on

- **Wave 4b:** enemy-debuff aura (Dissonance Organ — reuse the room pass, target `room.GetMobs()`, apply a debuff buff) + the companion subsystem (Brood Sac / Hive Mind / Brood Mother, extending `Character.Companions` + `actSummonCompanion`).
- **Wave 5:** the two actives (Venom Coat, Cocoon) + Winged Flight.
- **Wave 6:** full per-cluster authoring (cores + apexes with prereq spines), migration/re-bloom, `archetype_pull`→cluster re-curation, balance pass.
