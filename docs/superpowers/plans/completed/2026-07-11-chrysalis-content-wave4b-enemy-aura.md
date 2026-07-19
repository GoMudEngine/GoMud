# Chrysalis Content — Wave 4b: enemy-debuff aura (Dissonance Organ)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Complete the aura recipe (P7) in the *other* direction — an **enemy-debuff** aura — proven with **Dissonance Organ** (Weaver): your presence disrupts nearby foes.

**Architecture:** Mirrors Wave 4's ally aura. A new mutation effect `type: aura_enemy_debuff` (value = buff id); a per-room round-tick pass (`applyRoomEnemyAuras`) finds in-combat owners who project an enemy-aura and applies the debuff buff to the **in-combat mobs** in the room, via `mob.AddBuff` (event queue → start text + GMCP; silent on refresh). Short-lived buff → lapses when the owner leaves or the fight ends. This reuses the exact structure of `applyRoomAllyAuras`, differing only in target set (`room.GetMobs()` instead of party players) and effect (a debuff instead of a boon).

**Tech Stack:** Go — `internal/mutations`, `internal/hooks` (aura pass), `internal/mobs` + `internal/rooms` (mob targets); YAML buff + mutation + help; testify.

**Spec:** `docs/superpowers/specs/2026-07-11-chrysalis-cluster-content-design.md` (Wave 4b — the enemy-aura half of §9 Wave 4). Builds on Wave 4 (merged).

**Re-split note.** Wave 4b was originally billed as "enemy aura + companions (P8)." The companion subsystem (Brood Sac / Hive Mind / Brood Mother) is a sizeable extension of the mature `Character.Companions` + `CompanionInfo` + `actSummonCompanion` machinery (spawn, respawn, gear persistence, charm) and needs a brood-spawn mob authored — so it is split into its **own Wave 4c** plan. Wave 4b finishes the aura recipe cleanly.

**Scope:** First-cut Dissonance Organ applies a **stat debuff** (dulled will/perception → weaker casting + aim). The literal "chance to fumble spells/shouts" is a distinct interrupt mechanic deferred to the balance/deepening pass. Target set is "in-combat mobs in the room" (a Weaver's dissonance fills the room); tightening to only the owner's aggressors is a later refinement.

**Wave 1–4 reminders:** every mutation needs a `templates/help/{id}.template` + a `DescribeEffect` case; apply buffs via the `AddBuff` wrapper.

---

## File Structure

**Modify:**
- `internal/mutations/aura.go` — `GetEnemyAuraBuffs` helper
- `internal/mutations/describe.go` — `aura_enemy_debuff` case
- `internal/hooks/mutation_aura.go` — `applyRoomEnemyAuras`
- `internal/hooks/NewRound_UserRoundTick.go` — call it in the room loop (next to `applyRoomAllyAuras`)

**Create:**
- `_datafiles/world/dogmud/buffs/102-disrupted.yaml`
- `_datafiles/world/dogmud/mutations/dissonance-organ.yaml` + `.../help/dissonance-organ.template`
- Tests appended to `internal/mutations/aura_test.go`

---

## Phase 1 — The enemy-aura declaration

### Task 1: `GetEnemyAuraBuffs` + describe

**Files:** Modify `internal/mutations/aura.go`, `internal/mutations/describe.go`; append to `internal/mutations/aura_test.go`

- [ ] **Step 1: Write the failing test (append to aura_test.go)**

```go
func TestGetEnemyAuraBuffs(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"dissonance-organ": {MutationId: "dissonance-organ", Name: "Dissonance Organ", Rarity: 5,
			Pros: []MutationEffect{{Type: "aura_enemy_debuff", Value: 102}}},
	})
	defer cleanup()
	got := GetEnemyAuraBuffs(map[string]int{"dissonance-organ": 1})
	if len(got) != 1 || got[0] != 102 {
		t.Fatalf("GetEnemyAuraBuffs = %v, want [102]", got)
	}
	if len(GetEnemyAuraBuffs(map[string]int{})) != 0 {
		t.Fatal("no mutations → no enemy auras")
	}
}

func TestDescribeEffect_AuraEnemyDebuff(t *testing.T) {
	if DescribeEffect(MutationEffect{Type: "aura_enemy_debuff", Value: 102}) == "" {
		t.Fatal("aura_enemy_debuff must have a non-empty description")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/mutations/ -run "TestGetEnemyAuraBuffs|TestDescribeEffect_AuraEnemyDebuff" -v`
Expected: FAIL — `GetEnemyAuraBuffs` undefined; describe "".

- [ ] **Step 3: Implement**

In `internal/mutations/aura.go`, add:

```go
// GetEnemyAuraBuffs returns the debuff ids that owned mutations project onto
// nearby enemies (effect type "aura_enemy_debuff", Value = buff id).
func GetEnemyAuraBuffs(owned map[string]int) []int {
	var out []int
	for id := range owned {
		spec := GetMutation(id)
		if spec == nil {
			continue
		}
		for _, p := range spec.Pros {
			if p.Type == "aura_enemy_debuff" && p.Value > 0 {
				out = append(out, int(p.Value))
			}
		}
	}
	return out
}
```

In `internal/mutations/describe.go`, add a case (near `aura_ally_buff`):

```go
	case "aura_enemy_debuff":
		return "Your presence rattles nearby foes — their aim and focus falter."
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/mutations/ -run "TestGetEnemyAuraBuffs|TestDescribeEffect_AuraEnemyDebuff" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/mutations/aura.go internal/mutations/describe.go internal/mutations/aura_test.go
git commit -m "feat(mutations): aura_enemy_debuff effect type — GetEnemyAuraBuffs + describe"
```

---

### Task 2: The Disrupted enemy debuff

**Files:** Create `_datafiles/world/dogmud/buffs/102-disrupted.yaml`

Confirm 102 free: `python tools/id_inventory.py --type buffs` (101 = Emboldened, Wave 4).

- [ ] **Step 1: Write the buff (model on `25-deaths_shadow.yaml`)**

```yaml
buffid: 102
name: Disrupted
description: A grinding dissonance fills your skull — your aim wanders and your focus slips.
triggerrate: 1 round
triggercount: 2
statmods:
  willpower: -10
  perception: -10
start_room_text: "{source} shudders as a grinding resonance sets their teeth on edge."
end_room_text: "The dissonance around {source} fades."
```

> `triggercount: 2` gives a 2-round life so it lapses once the aura stops refreshing it. Debuffs will/perception (weaker casting + aim); first-pass magnitudes, tunable. Room-text (not user-text) since the primary bearers are mobs.

- [ ] **Step 2: Commit**

```bash
git add _datafiles/world/dogmud/buffs/102-disrupted.yaml
git commit -m "content(buffs): Disrupted enemy-aura debuff"
```

---

## Phase 2 — The enemy-aura tick

### Task 3: `applyRoomEnemyAuras` + wiring

**Files:** Modify `internal/hooks/mutation_aura.go`, `internal/hooks/NewRound_UserRoundTick.go`

- [ ] **Step 1: Implement the enemy pass**

In `internal/hooks/mutation_aura.go`, add (the file already imports `mutations`, `rooms`, `users`; add `mobs`):

```go
// applyRoomEnemyAuras applies each in-combat enemy-aura owner's debuff to the
// in-combat mobs in the room. Buffs go through the mob AddBuff wrapper (room
// text + GMCP; silent on refresh) and are short-lived so they lapse when the
// owner leaves or the fight ends.
func applyRoomEnemyAuras(room *rooms.Room) {
	playerIds := room.GetPlayers()
	if len(playerIds) == 0 {
		return
	}
	// Collect the debuffs projected by in-combat owners in the room.
	var debuffs []int
	for _, ownerId := range playerIds {
		owner := users.GetByUserId(ownerId)
		if owner == nil || !owner.Character.IsInCombat() {
			continue
		}
		debuffs = append(debuffs, mutations.GetEnemyAuraBuffs(owner.Character.Mutations)...)
	}
	if len(debuffs) == 0 {
		return
	}
	for _, mid := range room.GetMobs() {
		mob := mobs.GetInstance(mid)
		if mob == nil || !mob.Character.IsInCombat() {
			continue
		}
		for _, buffId := range debuffs {
			mob.AddBuff(buffId, "aura")
		}
	}
}
```

> Add `"github.com/GoMudEngine/GoMud/internal/mobs"` to the import block.

- [ ] **Step 2: Wire into the room loop**

In `internal/hooks/NewRound_UserRoundTick.go`, right after the existing `applyRoomAllyAuras(room)` call, add:

```go
			applyRoomEnemyAuras(room)
```

- [ ] **Step 3: Build + hooks suite**

Run: `go build ./... && go test ./internal/hooks/...`
Expected: clean + PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/mutation_aura.go internal/hooks/NewRound_UserRoundTick.go
git commit -m "feat(hooks): enemy-debuff aura room pass (applyRoomEnemyAuras)"
```

---

## Phase 3 — Content

### Task 4: `dissonance-organ.yaml` + help

**Files:** Create the mutation YAML + help template.

- [ ] **Step 1: Write `_datafiles/world/dogmud/mutations/dissonance-organ.yaml`**

(Weaver is a hybrid cluster → `pole: ""` via omission, so it never triggers the pole opposition.)

```yaml
mutationid: dissonance-organ
name: Dissonance Organ
description: |
  A resonating organ grows behind your sternum, emitting a subsonic grind that
  no one else can quite place. It sets teeth on edge and scatters concentration
  — spells slip, blows go wide — for every foe unlucky enough to fight near you.
rarity: 5
clusters: [weaver]
visual: A faint, teeth-itching hum surrounds them, felt more than heard.
pros:
  - type: aura_enemy_debuff
    value: 102
```

- [ ] **Step 2: Write `_datafiles/world/dogmud/templates/help/dissonance-organ.template`**

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="yellow">Dissonance Organ</ansi> mutation

A resonating organ behind your sternum emits a subsonic grind that
scatters the concentration of anyone fighting near you.

<ansi fg="yellow">Type:</ansi>     Passive (aura)
<ansi fg="yellow">Rarity:</ansi>   Rare

<ansi fg="yellow">Benefits:</ansi>
  While you fight, enemies in the room with you are disrupted — their
  aim wanders and their spellcraft falters

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help mutations</ansi>
```

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/mutations/dissonance-organ.yaml _datafiles/world/dogmud/templates/help/dissonance-organ.template
git commit -m "content(mutations): dissonance-organ (Weaver enemy-aura keystone)"
```

---

## Phase 4 — Verification

### Task 5: Build, full suite, boot smoke, manual smoke

- [ ] **Step 1: Build + affected suites**

Run: `go build ./... && go test ./internal/mutations/... ./internal/hooks/... ./internal/devtools/...`
Expected: clean, all PASS.

- [ ] **Step 2: Full suite**

Run: `go test ./...`
Expected: 0 failing packages.

- [ ] **Step 3: Boot smoke**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . 2>&1 | grep -mE1 "mutations.LoadMutationFiles|buffSpec.LoadDataFiles|panic:"
```
Expected: mutations + buffs load (mutation count = 51 + 1 = 52), no panic.

- [ ] **Step 4: Manual smoke**

Grant a test character `dissonance-organ`, fight a mob: confirm the mob gains the Disrupted debuff ("{source} shudders…") each round of the fight and it lapses ~2 rounds after combat ends; confirm the character does not debuff itself or allies.

- [ ] **Step 5: Commit** (only if a fix was needed)

---

## Self-Review (completed during authoring)

- **Spec coverage:** completes P7 (aura) with the enemy-debuff direction via Dissonance Organ, mirroring Wave 4's ally aura. The literal spell-fumble mechanic and the companion subsystem (P8 → Wave 4c) are explicitly deferred with rationale.
- **Placeholder scan:** buff id 102 flagged to-confirm-free; every other step carries complete code. Debuff magnitudes first-pass/tunable.
- **Type consistency:** `GetEnemyAuraBuffs(owned) []int` consistent across impl/test; `applyRoomEnemyAuras(room)` parallels `applyRoomAllyAuras`; `aura_enemy_debuff` string consistent across YAML/helper/describe; buffs applied via `mob.AddBuff` wrapper; owner-in-combat + mob-in-combat gates. Weaver mutation is `pole: ""` (hybrid) with help + describe.

## Follow-on

- **Wave 4c:** the companion subsystem (P8) — Brood Sac (passively maintain a bonded spawn: author a brood-spawn mob, respawn-if-missing round-tick, integrate with `Character.Companions`/`AddCompanion`), then Hive Mind (slots) + Brood Mother (apex swarm).
- **Wave 5:** the two actives (Venom Coat, Cocoon) + Winged Flight.
- **Wave 6:** full per-cluster authoring (cores + apexes with prereq spines), migration/re-bloom, `archetype_pull`→cluster re-curation, balance pass.
