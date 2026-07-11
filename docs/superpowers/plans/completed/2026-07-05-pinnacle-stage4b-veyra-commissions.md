# Pinnacle Items Stage 4b: Veyra & The Commissions (THE FINALE) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the nine pinnacle items obtainable — Veyra's dialogue, the intro
quest "The Convergence" (masterwork-gated), and 9 commission quests that teach
the recipes (`learn_recipe`), charge staged gold, and let one player run one
commission at a time. Plus the two engine additions this needs. On completion
the whole pinnacle arc is content-complete and the full craft loop is
live-playable.

**Architecture:** Two small engine additions — (A) a `charge_gold` quest action
+ `has_gold` condition (staged fees; `give_gold` only adds today), (B) a
masterwork entry gate (a `Character.HasOwnMasterwork(skillMin)` helper checking
the player carries an item with `MakerName == self && CraftSkill >= N`, exposed
as a dialogue `masterworkRequired` gate + a quest `has_masterwork` condition).
Everything else is content: Veyra's dialogue tree, quest 78 (intro), quests
79-87 (9 commissions). Each commission: ask Veyra (gated on known + no active
commission) → charge first-half gold → teach 2 component recipes + 1 assembly
recipe → player gathers reagents + crafts components + assembles the pinnacle
item → item_gain of the pinnacle item charges the second half + completes.

**Tech Stack:** Go (2 engine tasks in questengine/dialogue/characters), YAML
(quests, dialogue), existing quest+dialogue engines.

**Spec:** `docs/superpowers/specs/2026-07-04-pinnacle-chase-items-design.md`
(§3.1-3.3). **Refs:** `docs/schemas/dialogue.md`, `docs/schemas/pinnacle-items.md`
(the "Stage 4a" recipe-slug tables — the exact slugs to teach), quest 35
(`quests/35-first_heat.yaml`) + Rusk dialogue (`dialogue/pothole_coulee/9116.yaml`)
as the commission template.

**Branch:** `feature/pinnacle-stage4b-veyra-commissions` off `master`.

---

## IDs: quests 78-87 (78 intro + 79-87 the 9 commissions). No new item/mob/room/buff IDs — everything else exists (Veyra 9584, recipes, items 40181-40224).

### Commission → quest id → item → recipe slugs to teach (from docs/schemas/pinnacle-items.md "Stage 4a")

| quest | flag value | item (id) | component slugs | assembly slug | primary skill |
|---|---|---|---|---|---|
| 79 | bandolier | Vitalis Bandolier (40182) | reinforced-harness, preservation-runes | assemble-vitalis-bandolier | alchemy |
| 80 | blackrazor | The Blackrazor (40183) | hungering-guard, obsidian-edge-resin | assemble-the-blackrazor | blacksmithing |
| 81 | wayfarer | Wayfarer's Pack (40184) | reinforced-frame, spatial-stitching | assemble-wayfarers-pack | tailoring |
| 82 | aegis | Aegis of Mockery (40185) | voice-amber-housing, resonance-lacquer | assemble-aegis-of-mockery | blacksmithing |
| 83 | thornwall | Thornwall Harness (40186) | barbed-spike-plates, anti-corrosion-quench | assemble-thornwall-harness | tailoring |
| 84 | prism | Seething Prism (40187) | containment-lattice, nutrient-suspension | assemble-seething-prism | jewelcrafting |
| 85 | zephyr | Zephyr Treads (40188) | quicksilver-soles, windlace-bindings | assemble-zephyr-treads | tailoring |
| 86 | choir | Hollow Choir Staff (40189) | conductor-core, choir-focus-gems | assemble-hollow-choir-staff | enchanting |
| 87 | phial | Phial of Second Birth (40181) | reduction-base | assemble-phial-of-second-birth | alchemy |

(Verify each slug/item-id against the Stage-4a recipe files at build time.)
**Gold fees** (staged half/half, from spec §6 per-item totals): bandolier 35k
(17.5k+17.5k), blackrazor 50k (25k+25k), wayfarer 25k, aegis 40k, thornwall
30k, prism 40k, zephyr 25k, choir 45k, phial 30k. Use these.

---

### Task 1: Engine — `charge_gold` action + `has_gold` condition

**Files:**
- Modify: `internal/questengine/types.go` (ActionDef + Conditions fields)
- Modify: `internal/questengine/actions.go` (ChargeGold dispatch + ActionContext interface)
- Modify: `internal/questengine/conditions.go` (HasGold eval + PlayerState interface)
- Modify: `internal/questengine/bridge.go` (ChargeGold + GetGold impls)
- Modify: the questengine test mocks (`actions_test.go`, `conditions_test.go`, `engine_test.go` — add the new interface methods so they compile)
- Test: `internal/questengine/charge_gold_test.go` (new)

- [ ] **Step 1: Write the failing test**

```go
package questengine

import "testing"

func TestChargeGold_And_HasGold(t *testing.T) {
	// Use the package's existing mock ActionContext/PlayerState (extend it with
	// a gold field). Study actions_test.go / conditions_test.go for the mock
	// shape first — match it.
	// (a) charge_gold deducts, clamped at 0:
	//   a player with 100 gold, ExecuteAction{ChargeGold: 30} → gold 70.
	//   a player with 20 gold, ChargeGold: 50 → gold 0 (clamped, not negative).
	// (b) has_gold condition:
	//   EvalConditions{HasGold: 50} with player gold 60 → true; with 40 → false.
}
```
Write it concretely against the real mock (the mock ActionContext + PlayerState
already exist in the test files — add a gold field + the ChargeGold/GetGold
methods to them). Run → FAIL.

- [ ] **Step 2: Implement** (per the recon — exact edits):
  - `types.go`: `ActionDef` += `ChargeGold int \`yaml:"charge_gold,omitempty"\``; `Conditions` += `HasGold int \`yaml:"has_gold,omitempty"\``.
  - `actions.go`: add `ChargeGold(amount int)` to the `ActionContext` interface (beside `GiveGold`); add the dispatch block after GiveGold: `if a.ChargeGold > 0 { ctx.ChargeGold(a.ChargeGold); return nil }`.
  - `conditions.go`: add `GetGold() int` to the `PlayerState` interface; in `EvalConditions`, before `return true`: `if c.HasGold > 0 && p.GetGold() < c.HasGold { return false }`.
  - `bridge.go`: implement `func (b *GameBridge) ChargeGold(amount int)` (clamp `amount = min(amount, Gold)`; `Gold -= amount`; SendText "You pay N gold"; emit EquipmentChange{GoldChange: -amount}); `func (b *GameBridge) GetGold() int { return b.user.Character.Gold }`.
  - Test mocks: add `ChargeGold`/`GetGold` (+ a gold field) to the mock types in actions_test.go/conditions_test.go/engine_test.go so the interfaces still satisfy.

- [ ] **Step 3:** `go test ./internal/questengine/ -count=1` green; `go build ./...` clean.
- [ ] **Step 4: Commit**

```bash
git add internal/questengine/
git commit -m "feat(pinnacle): charge_gold action + has_gold condition (staged commission fees)"
```

---

### Task 2: Engine — the masterwork gate

A shared `Character.HasOwnMasterwork(skillMin)` helper, exposed as a dialogue
`masterworkRequired` gate AND a quest `has_masterwork` condition.

**Files:**
- Modify: `internal/characters/` (add the helper — e.g. in inventory.go or a new small file)
- Modify: `internal/dialogue/types.go` (MasterworkRequired on Pattern/TreeNode/QuestGreeting + HasOwnMasterwork on PlayerState)
- Modify: `internal/dialogue/engine.go` (checkQuestGate + the 3 call sites)
- Modify: `internal/usercommands/talk.go` (buildPlayerState — wire HasOwnMasterwork)
- Modify: `internal/questengine/types.go` (Conditions += HasMasterwork), `conditions.go` (eval + PlayerState.HasOwnMasterwork), `bridge.go` (impl)
- Test: `internal/characters/masterwork_test.go` (new) + a dialogue gate test

- [ ] **Step 1: The helper (TDD).** `internal/characters/masterwork_test.go`:
```go
package characters

import (
	"testing"
	"github.com/GoMudEngine/GoMud/internal/items"
)

func TestHasOwnMasterwork(t *testing.T) {
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		999970: {ItemId: 999970, Name: "masterwork blade", Type: items.Weapon},
	})()
	c := New()
	c.Name = "Megalomania"
	// no masterwork yet
	if c.HasOwnMasterwork(50) { t.Fatal("no items — should be false") }
	// carry an item I crafted at skill 65
	itm := items.New(999970)
	itm.MakerName = "Megalomania"
	itm.CraftSkill = 65
	c.StoreItem(itm) // or the real add-to-backpack method — match reality
	if !c.HasOwnMasterwork(50) { t.Fatal("own skill-65 item should pass a 50 gate") }
	// a foreign-made item does NOT count
	c2 := New(); c2.Name = "Megalomania"
	foreign := items.New(999970); foreign.MakerName = "SomeoneElse"; foreign.CraftSkill = 65
	c2.StoreItem(foreign)
	if c2.HasOwnMasterwork(50) { t.Fatal("foreign-made item must not count") }
	// my own but low-skill item does NOT count
	c3 := New(); c3.Name = "Megalomania"
	low := items.New(999970); low.MakerName = "Megalomania"; low.CraftSkill = 40
	c3.StoreItem(low)
	if c3.HasOwnMasterwork(50) { t.Fatal("own but skill<gate must not count") }
}
```
(Adapt `StoreItem`/the add-to-backpack call + `GetAllBackpackItems` to reality — read inventory.go. The bandolier auto-routes potions, so use a non-potion test item.)

Implement:
```go
// HasOwnMasterwork reports whether the character carries any item they
// personally crafted (MakerName == their name) at craft-skill >= skillMin.
// The pinnacle "show me a masterwork" entry gate.
func (c *Character) HasOwnMasterwork(skillMin int) bool {
	for _, itm := range c.GetAllBackpackItems() {
		if itm.MakerName == c.Name && itm.CraftSkill >= skillMin {
			return true
		}
	}
	return false
}
```
Run the test → green.

- [ ] **Step 2: Dialogue gate.** `dialogue/types.go`: add `MasterworkRequired int \`yaml:"masterworkRequired,omitempty"\`` to Pattern, TreeNode, QuestGreeting; add `HasOwnMasterwork func(skillMin int) bool` to the PlayerState struct. `dialogue/engine.go`: in `checkQuestGate`, add (guarded for backward-compat):
```go
	if masterworkRequired > 0 && ps.HasOwnMasterwork != nil && !ps.HasOwnMasterwork(masterworkRequired) {
		return false
	}
```
Thread `masterworkRequired` through the 3 call sites (patterns ~:120, tree nodes ~:206, root variants ~:246) — pass the node's MasterworkRequired. `talk.go` `buildPlayerState`: wire `HasOwnMasterwork: func(m int) bool { return user.Character.HasOwnMasterwork(m) }`.

- [ ] **Step 3: Quest condition.** `questengine/types.go`: `Conditions` += `HasMasterwork int \`yaml:"has_masterwork,omitempty"\``. `conditions.go`: PlayerState interface += `HasOwnMasterwork(skillMin int) bool`; eval: `if c.HasMasterwork > 0 && !p.HasOwnMasterwork(c.HasMasterwork) { return false }`. `bridge.go`: `func (b *GameBridge) HasOwnMasterwork(m int) bool { return b.user.Character.HasOwnMasterwork(m) }`. Update the questengine test mocks.

- [ ] **Step 4: A dialogue-gate test** — construct a PlayerState with HasOwnMasterwork returning false→true and assert a masterworkRequired node is hidden/shown accordingly (mirror an existing checkQuestGate test in internal/dialogue).

- [ ] **Step 5:** `go test ./internal/characters/ ./internal/dialogue/ ./internal/questengine/ ./internal/usercommands/ -count=1` green; `go build ./...` clean.
- [ ] **Step 6: Commit**

```bash
git add internal/characters/ internal/dialogue/ internal/questengine/ internal/usercommands/talk.go
git commit -m "feat(pinnacle): masterwork entry gate (HasOwnMasterwork — dialogue + quest condition)"
```

---

### Task 3: The intro quest 78 "The Convergence" + the commission flag

**Files:**
- Create: `_datafiles/world/dogmud/quests/78-the_convergence.yaml`

- [ ] **Step 1: Author quest 78.** A minimal "you are now known to Veyra" quest,
granted when the player asks Veyra while carrying a masterwork (the dialogue
node in Task 4 does the grant + the masterwork gate). Declare the shared
`commission` flag here (values = the 9 commission slugs + `none`):
```yaml
questid: 78
name: The Convergence
description: >-
  Veyra Coil-Tongue has taken your measure and found you worth her time.
  The convergence crafts are open to you now, one commission at a time.
steps:
  - id: start
    description: "You have shown Veyra a piece of your own making. She will hear a commission now."
  - id: end
    description: "You are known to Veyra."
flags:
  - key: commission
    values: [bandolier, blackrazor, wayfarer, aegis, thornwall, prism, zephyr, choir, phial, none]
    description: "Which pinnacle commission the player has active (none = free to start one)."
triggers:
  # granted 78-start via Veyra's dialogue (Task 4); auto-advance to 78-end + set commission=none.
  - event: quest_granted
    quest_token: "78-start"
    actions:
      - set_flag: {key: "78-commission", value: "none"}
      - grant: "78-end"
      - npc_say: "Veyra: \"Good. When you want a thing made, ask. One at a time.\""
```
(Verify the trigger event/action shapes against the questengine — `quest_granted` event + `grant`/`set_flag`/`npc_say` actions per the recon. If `quest_granted` self-trigger loops oddly, instead complete 78 via a second dialogue beat — adapt to what the engine supports; the GOAL is: after the intro, the player has 78-end + commission flag = none.)

- [ ] **Step 2: Boot smoke** — quests load (+1 → 68... actually 67+1), `ValidateAllFlags` OK (the commission flag is declared + will be referenced by Task 4/5 dialogue+quests — if those don't exist yet, references come later; 78 alone must declare cleanly), zero panics, killed.
- [ ] **Step 3: Commit**

```bash
git add "_datafiles/world/dogmud/quests/78-the_convergence.yaml"
git commit -m "content(pinnacle): The Convergence intro quest + commission flag"
```

---

### Task 4: Veyra's dialogue tree

**Files:**
- Create: `_datafiles/world/dogmud/dialogue/the_confluence/9584.yaml`

- [ ] **Step 1: Author the tree** (model on `dialogue/pothole_coulee/9116.yaml`).
Structure (all `text:` first-person as Veyra; `hints:` narrator; every
`grantsQuest` node includes `"quest"`+`"task"` in triggers per the SOP; end
tokens in questExcluded):
- **Root variants**:
  - Not-yet-known + NO masterwork: flavor/legend lines (she's dismissive; a legend to chase). No offer.
  - Not-yet-known + HAS masterwork (`masterworkRequired: 50`): she takes your measure; a node (triggers: quest/task/convergence/work/commission) `grantsQuest: 78-start`, `questExcluded: [78-start, 78-end]`, `masterworkRequired: 50`.
  - Known (`questRequired: [78-end]`): the working greeting; she'll hear a commission.
- **The 9 commission-offer nodes** (each `questRequired: [78-end]`,
  `questFlagRequired: {"78-commission": "none"}`, `questExcluded: [{id}-start, {id}-end]`,
  triggers include the item's discoverable keywords + quest/task):
  - `grantsQuest: {id}-start`, `setsQuestFlag: {key: "78-commission", value: <slug>}`.
  - Veyra names the price (the reagents + the gold) in `text:`; `hints:` steer the player.
  - (The charge-first-half gold + learn_recipe happen in the QUEST's quest_granted trigger — Task 5 — not the dialogue, so the dialogue stays declarative.)
- **A "collect/paid" acknowledgment** per commission is optional — the second
  gold charge + completion fire on item_gain (Task 5), so dialogue just needs
  the offer. Add a "mid-commission" root variant (`questFlagRequired:
  {"78-commission": <slug>}`) where Veyra asks how the work goes (flavor).
- **Truth-knower flavor** (optional, cheap): a variant line gated on quest 77's
  end token for players who know the Crash Site truth (reward, not gate).

- [ ] **Step 2: Boot smoke** — dialogue loads, `ValidateAllFlags` OK (the
`78-commission` flag references resolve against quest 78's declaration), zero
panics, killed. (A dialogue flag reference to an undeclared flag/value PANICS —
so the slug values must exactly match quest 78's declared values.)
- [ ] **Step 3: Commit**

```bash
git add "_datafiles/world/dogmud/dialogue/the_confluence/9584.yaml"
git commit -m "content(pinnacle): Veyra's dialogue tree (masterwork gate + 9 commission offers)"
```

---

### Task 5: Commission quests 79-83 (5 of 9)

**Files:** Create `quests/79-*.yaml` … `83-*.yaml` (bandolier, blackrazor,
wayfarer, aegis, thornwall).

- [ ] **Step 1: Author each commission** (uniform structure; use quest 35 as the
craft-quest template). Per commission (example: 80 Blackrazor):
```yaml
questid: 80
name: Commission - The Blackrazor
description: >-
  Veyra has agreed to forge The Blackrazor. Gather what she named, craft the
  parts by your own hand, and forge the blade at her stations.
steps:
  - id: start
    description: "Veyra has taken the commission. Gather the reagents and forge the parts."
  - id: end
    description: "The Blackrazor is made."
triggers:
  # On acceptance (granted via Veyra's dialogue node): charge first half + teach recipes.
  - event: quest_granted
    quest_token: "80-start"
    conditions: {has_gold: 25000}   # gate: only if affordable (the dialogue should also indicate the price)
    actions:
      - charge_gold: 25000
      - learn_recipe: {recipe: "hungering-guard"}
      - learn_recipe: {recipe: "obsidian-edge-resin"}
      - learn_recipe: {recipe: "assemble-the-blackrazor"}
      - npc_say: "Veyra: \"Half now. Bring me the rest when it is done. The recipes are in your hands.\""
  # On forging the item (the player assembles it at a station): charge second half + complete.
  - event: item_gain
    item: 40183
    actions:
      - charge_gold: 25000
      - set_flag: {key: "78-commission", value: "none"}
      - grant: "80-end"
      - npc_say: "Veyra: \"...it holds. Balance owed. Go, before it gets ideas.\""
```
IMPORTANT nuances to verify/handle at build time:
  - **Affordability**: if `has_gold` on the quest_granted trigger fails (can't
    afford the first half), the grant shouldn't strand the player. Best: gate
    the DIALOGUE offer node on `has_masterwork`+flag, and put the price in the
    text; the quest_granted charge assumes they accepted knowing the price. If
    the engine allows a `has_gold` GATE on the dialogue grantsQuest node,
    prefer that (no quest condition needed). Investigate + pick the cleaner
    path; document. A player who accepts without gold: the charge_gold clamps
    (pays what they have) — acceptable, or add the has_gold guard.
  - **Second charge on item_gain**: fires when the player first obtains the
    pinnacle item. Since they CRAFT it (not receive it), item_gain fires on the
    craft output. Confirm item_gain triggers on a crafted item (not just
    given/looted) — if it only fires on give/loot, use a `command: craft` +
    output check or a dialogue turn-in instead. VERIFY and adapt.
  - Each commission clears `78-commission` → `none` on completion (frees the
    next commission).
Author the 5 with their correct ids/items/slugs/gold from the table.

- [ ] **Step 2: Boot smoke** — quests load (+5), ValidateAllFlags OK (the
`78-commission` sets resolve), zero panics, killed.
- [ ] **Step 3: Commit**

```bash
git add "_datafiles/world/dogmud/quests/"
git commit -m "content(pinnacle): commission quests 79-83 (bandolier, blackrazor, wayfarer, aegis, thornwall)"
```

---

### Task 6: Commission quests 84-87 (the other 4)

**Files:** Create `quests/84-*.yaml` … `87-*.yaml` (prism, zephyr, choir, phial).

- [ ] **Step 1: Author** the same structure as Task 5, per the table (ids
84-87, correct items 40187/40188/40189/40181, slugs, gold). **Phial (87)** is
special: repeatable (`repeatable: true` + `cooldown_rounds`) since the Phial is
a consumable the player may commission again; it teaches only 1 component
(reduction-base) + the assembly (its other ingredient is the shared bottle +
reagents). Verify the repeatable shape (quest 54 pattern).

- [ ] **Step 2: Boot smoke** — quests load (+4 → 76 total real quests +
generic), ValidateAllFlags OK, zero panics, killed.
- [ ] **Step 3: Commit**

```bash
git add "_datafiles/world/dogmud/quests/"
git commit -m "content(pinnacle): commission quests 84-87 (prism, zephyr, choir, phial)"
```

---

### Task 7: Full-suite + boot + world-critic + THE FULL LIVE CRAFT

- [ ] **Step 1:** `go test -timeout 300s -count=1 ./...` → green (note the
pre-existing grapple flake if it surfaces).
- [ ] **Step 2:** Instance-wipe + boot: quests load (78-87 present),
`ValidateAllFlags` OK (all `78-commission` references across quests 78-87 +
Veyra's dialogue resolve against quest 78's declared values — a mismatch
PANICS), Veyra's dialogue loads, zero panics, Server Ready. Kill+verify+delete.
- [ ] **Step 3: World-critic** — Veyra's dialogue (her voice: quasi-legal,
exacting, few words) + the 10 quest descriptions against world.md. Fix inline;
re-boot if changed.
- [ ] **Step 4: THE FULL LIVE CRAFT (the payoff — now finally possible).** Via
the harness/admin: an admin character with a self-crafted skill-50+ item (to
pass the masterwork gate — craft any starter recipe at skill 50, OR admin-spawn
+ set MakerName/CraftSkill... prefer a real craft). Then: talk to Veyra → she
offers (masterwork gate passes) → accept a commission (e.g. Wayfarer's Pack,
cheapest/tailoring) → confirm gold charged (first half) + the 3 recipes learned
→ admin-spawn the reagents (folded-space-silk, warden-chassis-loom) + component
raw materials + bulk → craft the 2 components at their stations (confirm they
carry the player's MakerName) → assemble the Pack at the loom (confirm
require_own_components PASSES with own components) → confirm the Pack is
produced + the second gold charge fires + the commission completes + the
commission flag frees. ALSO confirm: a SECOND commission can't start while one
is active (flag gate); a player WITHOUT a masterwork gets only flavor from
Veyra (gate). This is the end-to-end proof of the entire pinnacle arc.
- [ ] **Step 5: Schema-doc + spec close-out** — append a "Stage 4b: commissions"
section to `docs/schemas/pinnacle-items.md` (quest ids, the commission flow, the
2 engine additions). Update the spec's §11 open items as resolved.
- [ ] **Step 6: Commit**

```bash
git add _datafiles/ docs/
git commit -m "content(pinnacle): Stage 4b world-critic polish + commission registry + arc close-out"
```

---

## After Stage 4b (the arc is content-complete)

- Arc-wide **geared-party combat calibration** (the #20/#21/#22 + pinnacle
  statpools/hazards are starting values — a geared non-godlike party pass).
- A full **live craft playtest** of a 2nd-3rd item to confirm the loop across
  disciplines.
- The **prod push** (pre-push SOP: PATCH_NOTES, LogToFile:false, boot-test,
  droplet deploy + perf datapoint) — the whole pinnacle arc (Stages 1-4)
  ships together.

## Notes / build-time decisions
- **item_gain on crafted items**: verify it fires on craft output (Task 5) —
  if not, use `command: craft`+output or a dialogue turn-in for the completion.
- **has_gold on the dialogue offer vs quest_granted**: prefer gating the
  dialogue offer if supported (cleaner); else the quest_granted charge clamps.
- **the intro quest completion**: ensure the player ends with 78-end +
  commission=none; adapt the mechanism to what the engine cleanly supports.
