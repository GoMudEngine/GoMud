# QOL Batch + Grenades Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement 7 independent QOL improvements: sort-to-bandolier, tank taunt, auto-eject spoiled potions, food spoiling + grenades, sell from bags, companion pool fix, and rhetoric shouts.

**Architecture:** Each feature is self-contained. Tasks 1-3 are small isolated changes. Task 4 is config-only. Task 5 is moderate (taunt aggro pull). Task 6 adds two new commands + buffs. Task 7 is the largest (food aging, throw command, grenade items/recipes). All share existing patterns (AoE from sonic shout, aging from potions, cooldowns from special moves).

**Tech Stack:** Go (testify for tests), YAML data files, ANSI template help files

**Spec:** `docs/superpowers/specs/2026-04-09-qol-batch-grenades-design.md`

---

### Task 1: Sort Command — Bandolier Support

**Files:**
- Modify: `internal/characters/character.go` (near `SortComponentItems` ~line 3626)
- Modify: `internal/usercommands/sort.go`
- Modify: `_datafiles/world/dogmud/templates/help/sort.template`
- Test: `internal/characters/character_test.go`

- [ ] **Step 1: Write test for SortPotionItems**

In `internal/characters/character_test.go`, add:

```go
func TestSortPotionItems(t *testing.T) {
	// Setup: character with a bandolier equipped and drinkable items in backpack
	c := characters.Character{}
	c.Equipment.Belt = items.Item{ItemId: 99} // fake bandolier

	// Add a drinkable item to backpack
	potion := items.Item{ItemId: 30036} // healing salve (Drinkable subtype)
	c.Items = append(c.Items, potion)

	// Add a non-drinkable item to backpack
	sword := items.Item{ItemId: 1} // not drinkable
	c.Items = append(c.Items, sword)

	moved := c.SortPotionItems()

	assert.Equal(t, 1, moved, "should move 1 potion")
	assert.Equal(t, 1, len(c.PotionItems), "bandolier should have 1 item")
	assert.Equal(t, 1, len(c.Items), "backpack should have 1 item (sword)")
}
```

Note: This test may need adjustment based on how `items.GetSpec()` works in the test environment (item specs must be loaded). Check existing tests in `character_test.go` for initialization patterns and adapt. If specs aren't loadable in tests, test the logic by checking that the method exists and returns 0 when no bandolier is equipped.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/characters/ -run TestSortPotionItems -v`
Expected: FAIL — `SortPotionItems` does not exist yet

- [ ] **Step 3: Implement SortPotionItems**

In `internal/characters/character.go`, add the new method near `SortComponentItems` (~line 3648):

```go
// SortPotionItems moves Drinkable-subtype items from the backpack into the
// equipped potion bandolier. Returns the number of items moved.
func (c *Character) SortPotionItems() int {
	belt := c.Equipment.Belt
	if belt.ItemId < 1 {
		return 0
	}
	beltSpec := belt.GetSpec()
	if !beltSpec.IsBandolier || beltSpec.BandolierCapacity <= 0 {
		return 0
	}

	moved := 0
	remaining := make([]items.Item, 0, len(c.Items))
	for _, item := range c.Items {
		spec := item.GetSpec()
		if spec.Subtype == items.Drinkable && len(c.PotionItems) < beltSpec.BandolierCapacity {
			c.PotionItems = append(c.PotionItems, item)
			moved++
		} else {
			remaining = append(remaining, item)
		}
	}
	c.Items = remaining
	return moved
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/characters/ -run TestSortPotionItems -v`
Expected: PASS (or adjust test if item spec loading is needed)

- [ ] **Step 5: Update sort.go command handler**

Replace the contents of `internal/usercommands/sort.go` with:

```go
package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Sort(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	hasBag := user.Character.Equipment.ComponentBag.ItemId > 0
	hasBandolier := false
	if user.Character.Equipment.Belt.ItemId > 0 {
		beltSpec := user.Character.Equipment.Belt.GetSpec()
		hasBandolier = beltSpec.IsBandolier
	}

	if !hasBag && !hasBandolier {
		user.SendText(`You don't have a component bag or bandolier equipped.`)
		return true, nil
	}

	var parts []string

	if hasBag {
		moved := user.Character.SortComponentItems()
		if moved > 0 {
			parts = append(parts, fmt.Sprintf(
				`%d material(s) into your %s`,
				moved, user.Character.Equipment.ComponentBag.DisplayName()))
		}
	}

	if hasBandolier {
		moved := user.Character.SortPotionItems()
		if moved > 0 {
			parts = append(parts, fmt.Sprintf(
				`%d potion(s) into your %s`,
				moved, user.Character.Equipment.Belt.DisplayName()))
		}
	}

	if len(parts) == 0 {
		user.SendText(`No items found to sort.`)
		return true, nil
	}

	user.SendText(fmt.Sprintf(
		`<ansi fg="green">Sorted %s.</ansi>`,
		strings.Join(parts, " and ")))

	return true, nil
}
```

- [ ] **Step 6: Update help file**

Replace `_datafiles/world/dogmud/templates/help/sort.template` with:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">sort</ansi>

The <ansi fg="command">sort</ansi> command organizes loose items from your
backpack into your equipped storage containers.

<ansi fg="yellow">Usage: </ansi>

  <ansi fg="command">sort</ansi>     Sort materials and potions into bags.

<ansi fg="yellow">What gets sorted:</ansi>

  <ansi fg="stat">Crafting materials</ansi> move into your component bag
  (ores, herbs, leather, binding paste, gemstones, etc.)

  <ansi fg="stat">Potions</ansi> move into your bandolier (any drinkable
  item currently in your backpack)

You must have the appropriate container equipped. Materials
need a component bag in the Components slot. Potions need a
bandolier in the Belt slot.

When you pick up or buy these items, they automatically go
into the right container if one is equipped and has space.
The <ansi fg="command">sort</ansi> command catches anything that was already
in your backpack before you equipped the container.

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help inventory</ansi>, <ansi fg="command">help craft</ansi>, <ansi fg="command">help drink</ansi>
```

- [ ] **Step 7: Compile and verify**

Run: `go build ./...`
Expected: Clean build

- [ ] **Step 8: Commit**

```bash
git add internal/characters/character.go internal/characters/character_test.go \
  internal/usercommands/sort.go \
  _datafiles/world/dogmud/templates/help/sort.template
git commit -m "feat: sort command moves potions into bandolier"
```

---

### Task 2: Sell from Bandolier / Component Bag

**Files:**
- Modify: `internal/usercommands/sell.go`
- Modify: `_datafiles/world/dogmud/templates/help/sell.template`

- [ ] **Step 1: Modify sell.go to search all storage**

In `internal/usercommands/sell.go`, replace lines 19-24:

```go
	item, found := user.Character.FindInBackpack(rest)

	if !found {
		user.SendText("You don't have that item.")
		return true, nil
	}
```

with:

```go
	item, found := user.Character.FindInBackpack(rest)
	if !found {
		item, found = user.Character.FindInPotions(rest)
	}
	if !found {
		item, found = user.Character.FindInComponents(rest)
	}

	if !found {
		user.SendText("You don't have that item.")
		return true, nil
	}
```

Note: Verify that `FindInComponents` exists on Character. If not, it follows the same pattern as `FindInPotions` but searches `ComponentItems`. Check `character.go` for the method name — it may be named differently. Use grep to find the correct method name before implementing.

- [ ] **Step 2: Update help file**

Replace `_datafiles/world/dogmud/templates/help/sell.template` with:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">sell</ansi>

The <ansi fg="command">sell</ansi> command sells an item to a merchant in
your current room.

<ansi fg="yellow">Usage: </ansi>

  <ansi fg="command">sell <item></ansi>     Sell an item to a merchant.

The command searches your backpack first, then your bandolier,
then your component bag. Quest items cannot be sold.

The price offered depends on the merchant's current stock and
gold reserves. Your <ansi fg="skill">Bartering</ansi> skill can increase
the sale price by up to 15%.

Find out more about referring to items by name by typing
<ansi fg="command">help item-names</ansi>.

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help buy</ansi>, <ansi fg="command">help appraise</ansi>, <ansi fg="command">help list</ansi>
```

- [ ] **Step 3: Compile and verify**

Run: `go build ./...`
Expected: Clean build. If `FindInComponents` doesn't exist, you'll need to add it to `character.go` following the `FindInPotions` pattern but searching `ComponentItems`.

- [ ] **Step 4: Commit**

```bash
git add internal/usercommands/sell.go \
  _datafiles/world/dogmud/templates/help/sell.template
git commit -m "feat: sell searches bandolier and component bag as fallbacks"
```

---

### Task 3: Auto-Eject Spoiled Potions from Bandolier

**Files:**
- Modify: `internal/hooks/NewRound_AutoHeal.go` (in the user regen loop)
- Uses: `internal/items/aging.go` (`GetAgingPhase`)

- [ ] **Step 1: Add spoiled potion ejection to AutoHeal**

In `internal/hooks/NewRound_AutoHeal.go`, inside the user loop, after the toxicity decay block (~line 97, after the `if user.Character.Toxicity > 0` block), add:

```go
		// Auto-eject spoiled potions from bandolier
		if len(user.Character.PotionItems) > 0 {
			currentRound := util.GetRoundCount()
			ejected := 0
			remaining := make([]items.Item, 0, len(user.Character.PotionItems))
			for _, pot := range user.Character.PotionItems {
				potSpec := pot.GetSpec()
				if potSpec.Aging.HasAging() && pot.CraftedRound > 0 {
					phase, _ := items.GetAgingPhase(pot, currentRound)
					if phase == items.PhaseSpoiled {
						user.Character.StoreItem(pot)
						user.SendText(fmt.Sprintf(
							`<ansi fg="yellow">Your <ansi fg="itemname">%s</ansi> has spoiled and falls out of your bandolier.</ansi>`,
							pot.DisplayName()))
						ejected++
						continue
					}
				}
				remaining = append(remaining, pot)
			}
			if ejected > 0 {
				user.Character.PotionItems = remaining
			}
		}
```

Note: You'll need to add `"github.com/GoMudEngine/GoMud/internal/items"` to the imports if not already present. Also add `"github.com/GoMudEngine/GoMud/internal/util"` if needed. Check existing imports first. Also verify that `StoreItem` won't re-route the potion back into the bandolier — if it does, use direct `c.Items = append(c.Items, pot)` instead.

**Important:** `StoreItem` auto-routes drinkable items back into the bandolier. Instead, append directly to `user.Character.Items` to force it into the backpack.

- [ ] **Step 2: Compile and verify**

Run: `go build ./...`
Expected: Clean build

- [ ] **Step 3: Commit**

```bash
git add internal/hooks/NewRound_AutoHeal.go
git commit -m "feat: auto-eject spoiled potions from bandolier to backpack"
```

---

### Task 4: Companion SP/CP Pool Fix (Config Changes)

**Files:**
- Modify: `_datafiles/config.yaml` (Balance section)
- Modify: `internal/configs/config.balance.go` (default values)

- [ ] **Step 1: Update balance config defaults**

In `internal/configs/config.balance.go`, find the default values for:
- `MobStaminaRegenPct` — change default from `0.01` to `0.02`
- `MobConvictionRegenPct` — change default from `0.01` to `0.02`
- `ManifestStatScaleChaFactor` — change default from `200` to `150`

Use grep to find the exact lines with these defaults and update them.

- [ ] **Step 2: Update config.yaml if it overrides these values**

Check `_datafiles/config.yaml` for explicit overrides of these three fields. If present, update them. If not present, the new defaults from Step 1 take effect automatically.

- [ ] **Step 3: Compile and verify**

Run: `go build ./...`
Expected: Clean build

- [ ] **Step 4: Commit**

```bash
git add internal/configs/config.balance.go _datafiles/config.yaml
git commit -m "fix: raise mob regen rates and companion stat scaling"
```

---

### Task 5: Taunt Pulls Aggro (Tank Taunt)

**Files:**
- Modify: `internal/actions/combat_taunt.go` (add `AggroPulled` to result, add aggro switch logic)
- Modify: `internal/usercommands/taunt.go` (add aggro-pull messaging)
- Modify: `internal/mobcommands/taunt.go` (add aggro-pull messaging for mob taunts)
- Modify: `_datafiles/world/dogmud/mobs/summons/305-flesh_golem.yaml` (add taunt to combat commands)
- Modify: `_datafiles/world/dogmud/templates/help/taunt.template`
- Test: `internal/actions/combat_taunt_test.go` (new file)

- [ ] **Step 1: Write test for aggro pull**

Create `internal/actions/combat_taunt_test.go`:

```go
package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/stretchr/testify/assert"
)

func init() {
	mudlog.SetupLogger(nil, "", "", false)
}

func TestTauntResultHasAggroPulledField(t *testing.T) {
	r := TauntResult{
		Hit:         true,
		AggroPulled: true,
	}
	assert.True(t, r.AggroPulled)
	assert.True(t, r.Hit)
}
```

Note: Full integration testing of `ExecuteTaunt` requires a mock Actor with aggro state, which may be complex. Start with a struct-level test to ensure the field exists, then verify behavior manually in-game.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/actions/ -run TestTauntResultHasAggroPulledField -v`
Expected: FAIL — `AggroPulled` field doesn't exist yet

- [ ] **Step 3: Add AggroPulled to TauntResult and implement aggro switch**

In `internal/actions/combat_taunt.go`:

Add to `TauntResult` struct (after `CritDeflected` field):

```go
	// AggroPulled is true when the taunt forced the target to switch aggro
	// to the taunter (target was fighting someone else).
	AggroPulled bool
```

Then, in the `if attackSuccess` block, after the conviction damage is applied and before the `return TauntResult{...}` at ~line 221, add aggro-pull logic:

```go
		// Tank taunt: force target to switch aggro to the taunter.
		agroPulled := false
		if target.MobInstanceId > 0 {
			mob := mobs.GetInstance(target.MobInstanceId)
			if mob != nil && mob.Character.Aggro != nil {
				currentTargetUserId := mob.Character.Aggro.UserId
				attackerUserId := 0
				if actor.IsPlayer() {
					attackerUserId = actor.GetUserId()
				}
				// Only pull if mob is targeting someone else
				if attackerUserId > 0 && currentTargetUserId != attackerUserId {
					mob.Character.SetAggro(attackerUserId, 0, characters.DefaultAttack)
					agroPulled = true
				}
			}
		}
```

You'll need to add imports for `"github.com/GoMudEngine/GoMud/internal/mobs"` and `"github.com/GoMudEngine/GoMud/internal/characters"`.

Also update the return statement to include `AggroPulled: agroPulled`.

Note: Check if `Actor` interface has a `GetUserId()` method. If not, you may need to add one, or extract the user ID differently. Check the Actor interface definition in `internal/actions/` for available methods.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/actions/ -run TestTauntResultHasAggroPulledField -v`
Expected: PASS

- [ ] **Step 5: Add aggro-pull messaging to taunt.go**

In `internal/usercommands/taunt.go`, after the `sendTauntMessages` call for `Hit` and `Crit` cases (inside the `case result.Hit && result.Crit:` and `case result.Hit:` blocks), add:

```go
		if result.AggroPulled {
			aggroPullMessages := []string{
				fmt.Sprintf(`Your words cut deep -- <ansi fg="mobname">%s</ansi> turns away from its prey and fixes its fury on you!`, targetName),
				fmt.Sprintf(`The insult lands true. <ansi fg="mobname">%s</ansi> abandons its quarry and lunges toward you!`, targetName),
				fmt.Sprintf(`Your mockery is unbearable -- <ansi fg="mobname">%s</ansi> wheels around to face you!`, targetName),
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> snarls and shifts its full attention to you.`, targetName),
			}
			user.SendText(aggroPullMessages[util.Rand(len(aggroPullMessages))])

			if room != nil {
				roomPullMessages := []string{
					fmt.Sprintf(`<ansi fg="mobname">%s</ansi> breaks off and turns its fury on <ansi fg="username">%s</ansi>!`, targetName, sourceName),
					fmt.Sprintf(`Enraged by <ansi fg="username">%s</ansi>'s taunts, <ansi fg="mobname">%s</ansi> shifts its attention!`, sourceName, targetName),
					fmt.Sprintf(`<ansi fg="mobname">%s</ansi> abandons its quarry and charges at <ansi fg="username">%s</ansi>!`, targetName, sourceName),
				}
				room.SendText(roomPullMessages[util.Rand(len(roomPullMessages))], user.UserId)
			}
		}
```

Add `"github.com/GoMudEngine/GoMud/internal/util"` to imports.

- [ ] **Step 6: Add taunt to flesh golem combat commands**

In `_datafiles/world/dogmud/mobs/summons/305-flesh_golem.yaml`, update `combatcommands` to include taunt:

```yaml
combatcommands:
  - 'emote brings both fists down in a crushing overhead blow'
  - 'taunt'
  - 'emote drives a shoulder into its target with the force of falling stone'
  - ''
```

- [ ] **Step 7: Update taunt help file**

Replace `_datafiles/world/dogmud/templates/help/taunt.template` with:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">taunt</ansi>

The <ansi fg="command">taunt</ansi> command launches a verbal assault against your
opponent, dealing <ansi fg="stat">conviction damage</ansi> and forcing them
to focus their attacks on you.

<ansi fg="yellow">Requirements:</ansi>

  - Cannot be used if special move cooldown is active

<ansi fg="yellow">Usage:</ansi>

  <ansi fg="command">taunt</ansi>            Taunt your current combat target
  <ansi fg="command">taunt <target></ansi>   Initiate combat by taunting a target

<ansi fg="yellow">Mechanics:</ansi>

  <ansi fg="stat">Hit Check:</ansi> Opposed roll
    - Attacker: Charisma + Rhetoric skill
    - Defender: Willpower + Rhetoric skill
  <ansi fg="stat">Damage:</ansi> Based on Charisma, Rhetoric rank, and target's
    conviction mitigation
  <ansi fg="stat">Damage Type:</ansi> Conviction (mental/social)
  <ansi fg="stat">Cooldown:</ansi> Shared with bash, trip, and kick
  <ansi fg="stat">Skill:</ansi> Progresses Rhetoric

<ansi fg="yellow">Aggro Pull:</ansi>

  When your taunt lands against an enemy that is fighting
  someone else (a companion, party member, or other player),
  the target is forced to switch its attacks to you. This
  makes taunt essential for protecting allies in group combat.

When a target's conviction reaches zero, they enter a
<ansi fg="red">downed</ansi> state -- overwhelmed and unable to fight.

<ansi fg="yellow">See Also:</ansi>

  <ansi fg="command">help rhetoric</ansi>, <ansi fg="command">help combat</ansi>, <ansi fg="command">help bash</ansi>
```

- [ ] **Step 8: Add taunt to keywords.yaml combat list**

In `_datafiles/world/dogmud/keywords.yaml`, check if `taunt` is already in the `combat:` section under `help: command:`. If not, add it:

```yaml
    combat:
      ...
      - taunt
```

- [ ] **Step 9: Compile and verify**

Run: `go build ./...`
Expected: Clean build

- [ ] **Step 10: Commit**

```bash
git add internal/actions/combat_taunt.go internal/actions/combat_taunt_test.go \
  internal/usercommands/taunt.go \
  _datafiles/world/dogmud/mobs/summons/305-flesh_golem.yaml \
  _datafiles/world/dogmud/templates/help/taunt.template \
  _datafiles/world/dogmud/keywords.yaml
git commit -m "feat: taunt pulls aggro on hit, golem companion taunts"
```

---

### Task 6: Rhetoric Shouts — Warcry + Rally

**Files:**
- Create: `internal/usercommands/warcry.go`
- Create: `internal/usercommands/rally.go`
- Create: `_datafiles/world/dogmud/buffs/79-warcry.yaml`
- Create: `_datafiles/world/dogmud/buffs/80-rally.yaml`
- Create: `_datafiles/world/dogmud/templates/help/warcry.template`
- Create: `_datafiles/world/dogmud/templates/help/rally.template`
- Modify: `internal/usercommands/usercommands.go` (register commands)
- Modify: `_datafiles/world/dogmud/keywords.yaml` (add to combat help)
- Test: `internal/usercommands/shout_test.go` (magnitude curve test)

- [ ] **Step 1: Write magnitude curve test**

Create `internal/usercommands/shout_test.go`:

```go
package usercommands

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func calcShoutBonus(rhetoric int, charisma int) float64 {
	raw := 0.05 + 0.15*math.Sqrt((float64(rhetoric)/75.0)*(float64(charisma)/175.0))
	if raw < 0.05 {
		return 0.05
	}
	if raw > 0.20 {
		return 0.20
	}
	return raw
}

func TestShoutBonusCurve(t *testing.T) {
	tests := []struct {
		rhetoric int
		charisma int
		minBonus float64
		maxBonus float64
	}{
		{1, 100, 0.05, 0.07},    // low end: ~5.5%
		{25, 120, 0.10, 0.13},   // mid-low
		{50, 150, 0.14, 0.18},   // mid-high
		{75, 175, 0.19, 0.21},   // cap: 20%
		{0, 100, 0.05, 0.06},    // zero rhetoric: floor
		{100, 200, 0.19, 0.21},  // over cap: still 20%
	}

	for _, tc := range tests {
		bonus := calcShoutBonus(tc.rhetoric, tc.charisma)
		assert.GreaterOrEqual(t, bonus, tc.minBonus,
			"rhetoric=%d cha=%d bonus=%.3f below min=%.3f", tc.rhetoric, tc.charisma, bonus, tc.minBonus)
		assert.LessOrEqual(t, bonus, tc.maxBonus,
			"rhetoric=%d cha=%d bonus=%.3f above max=%.3f", tc.rhetoric, tc.charisma, bonus, tc.maxBonus)
	}
}
```

- [ ] **Step 2: Run test to verify it passes**

Run: `go test ./internal/usercommands/ -run TestShoutBonusCurve -v`
Expected: PASS (the function is defined in the test file itself)

- [ ] **Step 3: Create buff YAML files**

Create `_datafiles/world/dogmud/buffs/79-warcry.yaml`:

```yaml
buffid: 79
name: Warcry
description: A rallying battle cry bolsters your fighting spirit.
triggerrate: 1 round
triggercount: 25
```

Create `_datafiles/world/dogmud/buffs/80-rally.yaml`:

```yaml
buffid: 80
name: Rally
description: An inspiring shout steadies your defenses.
triggerrate: 1 round
triggercount: 25
```

Note: The statmods are applied dynamically at cast time via `AddBuffScaled` with the calculated magnitude. The YAML defines the base duration; the command handler overrides the statmods when applying. Check how existing buff application works with dynamic statmods — you may need to set statmod values in code when adding the buff rather than in the YAML. Look at how shield spells or heal spells apply dynamic buff values for the pattern to follow.

- [ ] **Step 4: Create warcry.go command**

Create `internal/usercommands/warcry.go`:

```go
package usercommands

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

const warcryBuffId = 79

func Warcry(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	cfg := configs.GetBalanceConfig()
	if !user.Character.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		user.SendText("You need a moment to recover before attempting another special move.")
		return true, nil
	}

	rhetoric := user.Character.GetSkillLevel(skills.Rhetoric)
	charisma := user.Character.Stats.Charisma.ValueAdj

	bonus := calcWarcryBonus(rhetoric, charisma)

	user.Character.CancelBuffsWithFlag(buffs.Hidden)

	// Apply to self
	user.AddBuff(warcryBuffId, "warcry")
	// TODO: set dynamic statmod physical_damage_multiplier on the buff
	// Check how AddBuffScaled or similar works for dynamic buff values

	user.SendText(`<ansi fg="yellow-bold">You let loose a thundering warcry!</ansi>`)
	room.SendText(
		fmt.Sprintf(`<ansi fg="yellow-bold"><ansi fg="username">%s</ansi> lets loose a thundering warcry that stirs your blood!</ansi>`,
			user.Character.Name),
		user.UserId,
	)

	// Apply to all friendly targets in room: party members + companions
	for _, userId := range room.GetPlayers() {
		if userId == user.UserId {
			continue
		}
		if otherUser := users.GetByUserId(userId); otherUser != nil {
			// Check if in same party or allied
			if user.Character.IsPartyMember(userId) {
				otherUser.AddBuff(warcryBuffId, "warcry")
				otherUser.SendText(
					fmt.Sprintf(`<ansi fg="yellow-bold"><ansi fg="username">%s</ansi>'s warcry stirs your blood!</ansi>`,
						user.Character.Name))
			}
		}
	}

	// Apply to companions in room
	for _, mobInstId := range room.GetMobs(rooms.FindAll) {
		mob := mobs.GetInstance(mobInstId)
		if mob == nil {
			continue
		}
		if mob.OwnerId == user.UserId || user.Character.IsPartyMember(mob.OwnerId) {
			mob.Character.AddBuff(warcryBuffId, "warcry")
		}
	}

	// Progression: rhetoric skill + charisma stat
	// Higher chance in combat, 50% chance out of combat
	inCombat := user.Character.Aggro != nil
	user.Character.OnSkillUse(string(skills.Rhetoric), user.UserId)
	if inCombat {
		user.Character.OnStatUse("charisma", user.UserId)
	} else if util.Rand(100) < 50 {
		user.Character.OnStatUse("charisma", user.UserId)
	}

	if user.Character.Aggro != nil {
		user.Character.Aggro.RoundsWaiting = 1
	}

	_ = bonus // Will be used when dynamic buff application is wired up

	return true, nil
}

func calcWarcryBonus(rhetoric int, charisma int) float64 {
	raw := 0.05 + 0.15*math.Sqrt((float64(rhetoric)/75.0)*(float64(charisma)/175.0))
	if raw < 0.05 {
		return 0.05
	}
	if raw > 0.20 {
		return 0.20
	}
	return raw
}
```

**Important implementation note:** The exact mechanism for applying dynamic statmods to a buff needs investigation. Search for how `AddBuffScaled` works, or how shield spells set `effect_magnitude` dynamically. The buff YAML defines the template, but the actual magnitude is set at runtime. The implementer MUST:
1. Look at `AddBuffScaled()` in `character.go` to understand how it sets dynamic values
2. Look at how shield spells apply their buff with a variable magnitude
3. Wire the `bonus` value into the buff's statmods accordingly
4. The `calcWarcryBonus` function is shared — move it to a shared location if both warcry and rally need it, or duplicate it in `rally.go` as `calcRallyBonus` (same formula).

- [ ] **Step 5: Create rally.go command**

Create `internal/usercommands/rally.go` — same structure as `warcry.go` but:
- Uses buff ID 80 instead of 79
- Applies avoidance modifiers (dodge, parry, block) instead of physical_damage_multiplier
- Different messaging:
  - Caster: `<ansi fg="cyan-bold">You shout words of encouragement to your allies!</ansi>`
  - Room: `<ansi fg="cyan-bold"><username> shouts words of encouragement -- you feel steadier on your feet!</ansi>`
  - Buff recipient: `<ansi fg="cyan-bold"><username>'s rallying cry steadies your nerves!</ansi>`
- Same progression logic (rhetoric + charisma, 50% out-of-combat)
- Same magnitude curve (`calcRallyBonus` — identical formula)

- [ ] **Step 6: Register commands in usercommands.go**

In `internal/usercommands/usercommands.go`, add to the `userCommands` map:

```go
		`warcry`:       {Warcry, false, true, false},
		`rally`:        {Rally, false, true, false},
```

Note: Both are `AllowedInCombat: true` and can also be used out of combat (pre-buffing).

- [ ] **Step 7: Create help files**

Create `_datafiles/world/dogmud/templates/help/warcry.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">warcry</ansi>

The <ansi fg="command">warcry</ansi> command lets loose a thundering battle
cry that bolsters the offensive power of all allies in the
room.

<ansi fg="yellow">Usage:</ansi>

  <ansi fg="command">warcry</ansi>     Buff all friendly targets in the room.

<ansi fg="yellow">Effect:</ansi>

  Increases physical damage dealt by you, your companions,
  and party members in the room. The bonus scales with your
  <ansi fg="skill">Rhetoric</ansi> skill and <ansi fg="stat">Charisma</ansi>.

<ansi fg="yellow">Mechanics:</ansi>

  <ansi fg="stat">Duration:</ansi> Approximately 25 rounds
  <ansi fg="stat">Cooldown:</ansi> Shared with bash, trip, kick, and taunt
  <ansi fg="stat">Skill:</ansi> Progresses Rhetoric and Charisma

Can be used before combat to pre-buff your group. Rhetoric
skill and Charisma both progress on use, with a higher
chance of improvement when used in active combat.

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help rally</ansi>, <ansi fg="command">help taunt</ansi>, <ansi fg="command">help rhetoric</ansi>
```

Create `_datafiles/world/dogmud/templates/help/rally.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">rally</ansi>

The <ansi fg="command">rally</ansi> command shouts words of encouragement
that bolster the defenses of all allies in the room.

<ansi fg="yellow">Usage:</ansi>

  <ansi fg="command">rally</ansi>     Buff all friendly targets in the room.

<ansi fg="yellow">Effect:</ansi>

  Improves the ability to dodge, parry, and block attacks
  for you, your companions, and party members in the room.
  The bonus scales with your <ansi fg="skill">Rhetoric</ansi> skill and
  <ansi fg="stat">Charisma</ansi>.

<ansi fg="yellow">Mechanics:</ansi>

  <ansi fg="stat">Duration:</ansi> Approximately 25 rounds
  <ansi fg="stat">Cooldown:</ansi> Shared with bash, trip, kick, and taunt
  <ansi fg="stat">Skill:</ansi> Progresses Rhetoric and Charisma

Can be used before combat to pre-buff your group. Rhetoric
skill and Charisma both progress on use, with a higher
chance of improvement when used in active combat.

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help warcry</ansi>, <ansi fg="command">help taunt</ansi>, <ansi fg="command">help rhetoric</ansi>
```

- [ ] **Step 8: Add to keywords.yaml**

In `_datafiles/world/dogmud/keywords.yaml`, add `warcry` and `rally` under `combat:`:

```yaml
    combat:
      ...
      - rally
      - warcry
```

- [ ] **Step 9: Compile and run all tests**

Run: `go build ./... && go test ./internal/usercommands/ -run TestShoutBonusCurve -v`
Expected: Clean build and PASS

- [ ] **Step 10: Commit**

```bash
git add internal/usercommands/warcry.go internal/usercommands/rally.go \
  internal/usercommands/shout_test.go \
  internal/usercommands/usercommands.go \
  _datafiles/world/dogmud/buffs/79-warcry.yaml \
  _datafiles/world/dogmud/buffs/80-rally.yaml \
  _datafiles/world/dogmud/templates/help/warcry.template \
  _datafiles/world/dogmud/templates/help/rally.template \
  _datafiles/world/dogmud/keywords.yaml
git commit -m "feat: add warcry and rally rhetoric-based AoE buff shouts"
```

---

### Task 7: Food Spoiling + Putrid Residue + Grenades + Throw Command

This is the largest task. It has four sub-parts that build on each other.

**Files:**
- Modify: `internal/usercommands/eat.go` (spoiled food check)
- Modify: `internal/hooks/NewRound_UserRoundTick.go` (crafted food aging stamp — verify this is where crafting completes)
- Create: `_datafiles/world/dogmud/items/materials-40000/40050-putrid_residue.yaml`
- Create: `_datafiles/world/dogmud/items/alchemy-30000/30057-flashbang.yaml`
- Create: `_datafiles/world/dogmud/items/alchemy-30000/30058-firebomb.yaml`
- Create: `_datafiles/world/dogmud/items/alchemy-30000/30059-toxic_flask.yaml`
- Create: `_datafiles/world/dogmud/recipes/alchemy/flashbang.yaml`
- Create: `_datafiles/world/dogmud/recipes/alchemy/firebomb.yaml`
- Create: `_datafiles/world/dogmud/recipes/alchemy/toxic-flask.yaml`
- Create: `_datafiles/world/dogmud/buffs/77-flashbang_blindness.yaml`
- Create: `_datafiles/world/dogmud/buffs/78-toxic_cloud.yaml`
- Create: `internal/usercommands/throw.go`
- Create: `_datafiles/world/dogmud/templates/help/throw.template`
- Modify: `internal/usercommands/usercommands.go` (register throw)
- Modify: `_datafiles/world/dogmud/keywords.yaml` (add throw to combat)
- Test: `internal/usercommands/throw_test.go`

#### Sub-part 7a: Food Spoiling in eat.go

- [ ] **Step 1: Add spoiled food check to eat.go**

In `internal/usercommands/eat.go`, after the `Subtype != items.Edible` check (~line 24), add a spoiled food check:

```go
		// Check if food has spoiled
		if itemSpec.Aging.HasAging() && matchItem.CraftedRound > 0 {
			phase, _ := items.GetAgingPhase(matchItem, util.GetRoundCount())
			if phase == items.PhaseSpoiled {
				user.SendText(
					`<ansi fg="red">The food has gone bad! It reeks of decay ` +
						`and is clearly inedible.</ansi>`)
				return true, nil
			}
		}
```

Add imports: `"github.com/GoMudEngine/GoMud/internal/items"` (if not present) and `"github.com/GoMudEngine/GoMud/internal/util"`.

- [ ] **Step 2: Verify CraftedRound stamping covers food**

In `internal/hooks/NewRound_UserRoundTick.go` at ~line 329, the crafting output code already stamps `CraftedRound` on ALL crafted items unconditionally:

```go
newItem.CraftedRound = util.GetRoundCount()
```

The `bottleAgingMult > 0` check only gates `BottleMultiplier`, not `CraftedRound`. So crafted food already gets `CraftedRound` stamped. Verify this by reading the code. If the stamping is conditional on potion subtype, broaden the condition.

- [ ] **Step 3: Compile and verify**

Run: `go build ./...`
Expected: Clean build

- [ ] **Step 4: Commit**

```bash
git add internal/usercommands/eat.go
git commit -m "feat: block eating spoiled food with descriptive message"
```

#### Sub-part 7b: Putrid Residue Material

- [ ] **Step 5: Create putrid residue item**

Create `_datafiles/world/dogmud/items/materials-40000/40050-putrid_residue.yaml`:

```yaml
itemid: 40050
name: putrid residue
namesimple: residue
description: >-
  A vile, viscous paste that forms when organic matter breaks
  down. The smell alone could peel paint. Alchemists prize it
  as a volatile base for explosive mixtures.
type: object
subtype: mundane
component_tag: putrid-residue
weight: 0.1
value: 1
is_component: true
```

- [ ] **Step 6: Add salvage_returns to existing food items that have aging**

Find food items that will have `aging:` thresholds added (this is a content decision — which foods spoil). For each spoilable food item YAML, add:

```yaml
salvage_returns:
  - item_tag: putrid-residue
    quantity: 1
```

Also add `salvage_returns` to spoilable potion items. Search `_datafiles/world/dogmud/items/alchemy-30000/` for potion items and add the same salvage returns to each.

Note: Not all food or potions need salvage returns — only ones that have `aging:` thresholds. For existing food that doesn't have aging yet, you'll need to add `aging:` fields. This is a content decision — the implementer should check which food items exist and add aging thresholds to appropriate ones. Start with crafted food items from cooking recipes.

- [ ] **Step 7: Commit**

```bash
git add _datafiles/world/dogmud/items/materials-40000/40050-putrid_residue.yaml
git commit -m "feat: add putrid residue salvage material"
```

#### Sub-part 7c: Grenade Items, Buffs, and Recipes

- [ ] **Step 8: Create flashbang blindness buff**

Create `_datafiles/world/dogmud/buffs/77-flashbang_blindness.yaml`:

```yaml
buffid: 77
name: Flashbang Blindness
description: >-
  A blinding flash has seared your vision. You can barely
  see and your movements are clumsy.
triggerrate: 1 round
triggercount: 3
flags:
  - no-combat
statmods:
  perception: -30
  dexterity: -15
```

Note: The `no-combat` flag prevents the affected mob from attacking for 1 round (the flag is consumed, but the perception/dexterity penalties persist for the full duration). Verify that `no-combat` works this way by checking existing uses of the flag. If it blocks ALL combat actions for the entire duration, reduce `triggercount` or remove the flag and rely on stat penalties only.

- [ ] **Step 9: Create toxic cloud buff**

Create `_datafiles/world/dogmud/buffs/78-toxic_cloud.yaml`:

```yaml
buffid: 78
name: Toxic Cloud
description: >-
  A noxious cloud clings to you, burning your lungs and
  sapping your strength with every breath.
triggerrate: 1 round
triggercount: 5
```

This buff needs a JS script to deal DoT damage each round. Create `_datafiles/world/dogmud/buffs/78-toxic_cloud.js`:

```javascript
function onTrigger(actor, triggersLeft) {
    var maxHp = actor.GetHealthMax();
    var dmg = Math.floor(maxHp * 0.02); // 2% max HP per tick
    if (dmg < 1) dmg = 1;
    actor.ApplyHealthDamage(dmg);
    SendUserMessage(actor.UserId(), '<ansi fg="green">The toxic fumes burn your lungs!</ansi>');
}
```

Note: Check what scripting API methods are available on `actor` for applying damage. It might be `actor.TakeDamage()` or direct health manipulation. Look at existing buff scripts (like poison or bleed) for the correct pattern. The buff 75 (nausea) uses statmods only, but check buff scripts in `_datafiles/world/dogmud/buffs/` for DoT examples.

- [ ] **Step 10: Create grenade item YAMLs**

Create `_datafiles/world/dogmud/items/alchemy-30000/30057-flashbang.yaml`:

```yaml
itemid: 30057
name: flashbang
namesimple: flashbang
description: >-
  A small glass sphere filled with a volatile mixture that
  ignites on impact, producing a blinding flash. Handle
  with care.
type: object
subtype: throwable
uses: 1
value: 15
weight: 0.2
buffids:
  - 77
```

Create `_datafiles/world/dogmud/items/alchemy-30000/30058-firebomb.yaml`:

```yaml
itemid: 30058
name: firebomb
namesimple: firebomb
description: >-
  A sealed flask of unstable alchemical reagents that
  explode violently on impact. The heat alone can singe
  nearby flesh.
type: object
subtype: throwable
uses: 1
value: 20
weight: 0.3
damage_multiplier: 0.30
```

Create `_datafiles/world/dogmud/items/alchemy-30000/30059-toxic_flask.yaml`:

```yaml
itemid: 30059
name: toxic flask
namesimple: flask
description: >-
  A thin-walled vial of concentrated poison that shatters
  easily, releasing a cloud of choking fumes over a wide
  area.
type: object
subtype: throwable
uses: 1
value: 18
weight: 0.2
buffids:
  - 78
```

- [ ] **Step 11: Create alchemy recipes**

Create `_datafiles/world/dogmud/recipes/alchemy/flashbang.yaml`:

```yaml
id: flashbang
name: Flashbang
skill: alchemy
skill_minimum: 15
station: alchemy_bench
time_rounds: 3
ingredients:
  - item_tag: putrid-residue
    quantity: 1
  - item_tag: mineral-salt
    quantity: 1
  - item_tag: bottle
    quantity: 1
output:
  item_id: 30057
  quantity: 1
success_message: >-
  The mixture crystallizes into a volatile sphere that
  glows faintly. A flashbang!
failure_message: >-
  The compounds fizzle and separate. The materials are
  wasted.
```

Create `_datafiles/world/dogmud/recipes/alchemy/firebomb.yaml`:

```yaml
id: firebomb
name: Firebomb
skill: alchemy
skill_minimum: 25
station: alchemy_bench
time_rounds: 4
ingredients:
  - item_tag: putrid-residue
    quantity: 1
  - item_tag: flask-of-oil
    quantity: 1
  - item_tag: bottle
    quantity: 1
output:
  item_id: 30058
  quantity: 1
success_message: >-
  The oil ignites briefly then stabilizes inside the flask,
  swirling with barely-contained fury. A firebomb!
failure_message: >-
  The mixture sputters and goes inert. The materials are lost.
```

Create `_datafiles/world/dogmud/recipes/alchemy/toxic-flask.yaml`:

```yaml
id: toxic-flask
name: Toxic Flask
skill: alchemy
skill_minimum: 20
station: alchemy_bench
time_rounds: 4
ingredients:
  - item_tag: putrid-residue
    quantity: 1
  - item_tag: venom-sac
    quantity: 1
  - item_tag: bottle
    quantity: 1
output:
  item_id: 30059
  quantity: 1
success_message: >-
  The venom dissolves into a sickly green cloud that swirls
  inside the sealed flask. A toxic flask!
failure_message: >-
  The venom crystallizes uselessly. The materials are wasted.
```

Note: Verify that `mineral-salt`, `flask-of-oil`, and `venom-sac` are valid `component_tag` values on existing items. Grep `_datafiles/world/dogmud/items/` for these tags. If they don't exist, either create the material items or substitute with existing tags that make thematic sense.

- [ ] **Step 12: Commit item and recipe files**

```bash
git add _datafiles/world/dogmud/items/alchemy-30000/30057-flashbang.yaml \
  _datafiles/world/dogmud/items/alchemy-30000/30058-firebomb.yaml \
  _datafiles/world/dogmud/items/alchemy-30000/30059-toxic_flask.yaml \
  _datafiles/world/dogmud/buffs/77-flashbang_blindness.yaml \
  _datafiles/world/dogmud/buffs/78-toxic_cloud.yaml \
  _datafiles/world/dogmud/buffs/78-toxic_cloud.js \
  _datafiles/world/dogmud/recipes/alchemy/flashbang.yaml \
  _datafiles/world/dogmud/recipes/alchemy/firebomb.yaml \
  _datafiles/world/dogmud/recipes/alchemy/toxic-flask.yaml
git commit -m "feat: add grenade items, buffs, and alchemy recipes"
```

#### Sub-part 7d: Throw Command

- [ ] **Step 13: Write throw command test**

Create `internal/usercommands/throw_test.go`:

```go
package usercommands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestThrowDamageCalc(t *testing.T) {
	// Firebomb damage: Dex * SkillMult * 0.30 (physical channel)
	// At dex=100, skill=0 (mult=1.0): raw = 100 * 1.0 * 0.30 = 30
	// This validates the formula matches the spec
	dex := 100
	skillMult := 1.0
	channelScale := 0.30
	raw := float64(dex) * skillMult * channelScale
	assert.InDelta(t, 30.0, raw, 0.01)

	// At dex=150, skill=25 (mult ~1.71): raw = 150 * 1.71 * 0.30 = 76.95
	dex = 150
	skillMult = 1.0 + 2.0*math.Sqrt(25.0/50.0) // SkillMultiplier formula
	raw = float64(dex) * skillMult * channelScale
	assert.InDelta(t, 76.95, raw, 1.0)
}
```

Add `"math"` to imports.

- [ ] **Step 14: Run test**

Run: `go test ./internal/usercommands/ -run TestThrowDamageCalc -v`
Expected: PASS

- [ ] **Step 15: Create throw.go command**

Create `internal/usercommands/throw.go`:

```go
package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Throw(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if rest == "" {
		user.SendText("Throw what?")
		return true, nil
	}

	// Must be in combat
	if user.Character.Aggro == nil {
		user.SendText("You must be in combat to throw things!")
		return true, nil
	}

	// Check special-move cooldown
	cfg := configs.GetBalanceConfig()
	if !user.Character.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		user.SendText("You need a moment to recover before attempting another special move.")
		return true, nil
	}

	// Find throwable item in backpack
	matchItem, found := user.Character.FindInBackpack(rest)
	if !found {
		user.SendText(fmt.Sprintf(`You don't have a "%s" to throw.`, rest))
		return true, nil
	}

	itemSpec := matchItem.GetSpec()
	if itemSpec.Subtype != items.Throwable {
		user.SendText(fmt.Sprintf(
			`You can't throw <ansi fg="itemname">%s</ansi>.`,
			matchItem.DisplayName()))
		return true, nil
	}

	user.Character.CancelBuffsWithFlag(buffs.Hidden)

	// Consume the item
	if usesLeft := user.Character.UseItem(matchItem); usesLeft < 1 {
		events.AddToQueue(events.ItemOwnership{
			UserId: user.UserId,
			Item:   matchItem,
			Gained: false,
		})
	}

	// Attack scores
	skullduggery := user.Character.GetSkillLevel(skills.Skullduggery)
	skillWeight := float64(cfg.SkillWeight)
	attackScore := float64(user.Character.Stats.Dexterity.ValueAdj) +
		float64(skullduggery)*skillWeight

	// Flavor message
	user.SendText(fmt.Sprintf(
		`<ansi fg="red-bold">You hurl a <ansi fg="itemname">%s</ansi> into the fray!</ansi>`,
		matchItem.DisplayName()))
	room.SendText(fmt.Sprintf(
		`<ansi fg="red-bold"><ansi fg="username">%s</ansi> hurls a <ansi fg="itemname">%s</ansi> into the fray!</ansi>`,
		user.Character.Name, matchItem.DisplayName()),
		user.UserId,
	)

	// AoE resolution: opposed roll per hostile mob in room
	for _, mobInstId := range room.GetMobs(rooms.FindAll) {
		mob := mobs.GetInstance(mobInstId)
		if mob == nil {
			continue
		}

		// Skip friendly companions
		if mob.OwnerId > 0 {
			continue
		}

		defenseScore := float64(mob.Character.Stats.Dexterity.ValueAdj) +
			float64(mob.Character.GetSkillLevel(skills.Perception))*skillWeight

		attackSuccess, _, atkRoll, _ := dice.OpposedRollStat(attackScore, defenseScore)

		// Fumble: effect hits the thrower
		if atkRoll.ZScore <= -2.0 {
			user.SendText(
				`<ansi fg="red">The throw goes horribly wrong -- it explodes in your hands!</ansi>`)
			applyGrenadeEffect(itemSpec, user, nil, true)
			// Only fumble once, not per mob
			break
		}

		if attackSuccess {
			applyGrenadeEffect(itemSpec, user, mob, false)
		}

		combat.RecordSpecialMove(combat.User, combat.Mob, "throw",
			attackSuccess, 0, user.Character, &mob.Character, util.GetRoundCount())
	}

	// Progression
	user.Character.OnSkillUse(string(skills.Skullduggery), user.UserId)
	user.Character.OnStatUse("dexterity", user.UserId)

	if user.Character.Aggro != nil {
		user.Character.Aggro.RoundsWaiting = 1
	}

	return true, nil
}

// applyGrenadeEffect applies the grenade's effect based on its item spec.
// If selfHit is true, the effect is applied to the thrower (fumble).
func applyGrenadeEffect(spec items.ItemSpec, user *users.UserRecord, mob *mobs.MobInstance, selfHit bool) {
	// Firebomb: direct physical damage
	if spec.DamageMultiplier > 0 {
		skullduggery := user.Character.GetSkillLevel(skills.Skullduggery)
		rawDmg := combat.CalcRawDamage(
			user.Character.Stats.Dexterity.ValueAdj,
			skullduggery,
			spec.DamageMultiplier,
			combat.ChannelPhysical,
		)

		if selfHit {
			dmgRoll := dice.RollStat(rawDmg)
			dmg := int(dmgRoll.Value)
			if dmg < 1 {
				dmg = 1
			}
			user.Character.Health -= dmg
			if user.Character.Health < -10 {
				user.Character.Health = -10
			}
			user.SendText(fmt.Sprintf(
				`<ansi fg="red">The explosion sears you! (%s)</ansi>`,
				combat.GetDamageDescription(dmg, user.Character.HealthMax.Value)))
		} else if mob != nil {
			mitigPct := mob.Character.GetPhysicalMitigation()
			cap := combat.MitigationCap(combat.ChannelPhysical)
			dmgMean := combat.ApplyMitigation(rawDmg, mitigPct, cap)
			dmgRoll := dice.RollStat(dmgMean)
			dmg := int(dmgRoll.Value)
			if dmg < 1 {
				dmg = 1
			}
			mob.Character.Health -= dmg
			if mob.Character.Health < 0 {
				mob.Character.Health = 0
			}
			user.SendText(fmt.Sprintf(
				`The explosion engulfs <ansi fg="mobname">%s</ansi>! (%s)`,
				mob.Character.Name,
				combat.GetDamageDescription(dmg, mob.Character.HealthMax.Value)))
		}
		return
	}

	// Debuff grenades: apply buff IDs
	for _, buffId := range spec.BuffIds {
		if selfHit {
			user.AddBuff(buffId, "grenade")
		} else if mob != nil {
			mob.Character.AddBuff(buffId, "grenade")
			user.SendText(fmt.Sprintf(
				`<ansi fg="mobname">%s</ansi> is caught in the blast!`,
				mob.Character.Name))
		}
	}
}
```

**Important implementation notes:**
1. Check that `items.Throwable` is defined as a subtype constant. It was noted in research as existing but verify.
2. The `applyGrenadeEffect` function pattern may need adjustment based on how `mob.Character.AddBuff` works vs `user.AddBuff`.
3. Check if `mob.Character.Health` can be set directly or needs a method call.
4. The companion skip uses `mob.OwnerId > 0` — verify this is the correct way to identify companions. Also check for party members' companions.

- [ ] **Step 16: Register throw command**

In `internal/usercommands/usercommands.go`, add:

```go
		`throw`:        {Throw, false, true, false},
```

- [ ] **Step 17: Create throw help file**

Create `_datafiles/world/dogmud/templates/help/throw.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">throw</ansi>

The <ansi fg="command">throw</ansi> command hurls a throwable item at your
enemies, affecting all hostile targets in the room.

<ansi fg="yellow">Requirements:</ansi>

  - Must be in active combat
  - Must have a throwable item in your backpack
  - Cannot be used if special move cooldown is active

<ansi fg="yellow">Usage:</ansi>

  <ansi fg="command">throw <item></ansi>     Throw an item at all enemies.

<ansi fg="yellow">Throwable Items:</ansi>

  <ansi fg="itemname">Flashbang</ansi>    Blinds all enemies, impairing their
                 combat ability for a short time.
  <ansi fg="itemname">Firebomb</ansi>     Deals physical damage to all enemies
                 based on your Dexterity.
  <ansi fg="itemname">Toxic Flask</ansi>  Poisons all enemies, dealing damage
                 over time for several rounds.

<ansi fg="yellow">Mechanics:</ansi>

  <ansi fg="stat">Hit Check:</ansi> Opposed roll per target
    - Attacker: Dexterity + Skullduggery skill
    - Defender: Dexterity + Perception skill
  <ansi fg="stat">Fumble:</ansi> The grenade explodes in your hands!
  <ansi fg="stat">Cooldown:</ansi> Shared with bash, trip, kick, and taunt
  <ansi fg="stat">Skill:</ansi> Progresses Skullduggery and Dexterity

Throwable items are crafted via <ansi fg="skill">Alchemy</ansi> using
putrid residue and other reagents.

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help craft</ansi>, <ansi fg="command">help alchemy</ansi>, <ansi fg="command">help combat</ansi>
```

- [ ] **Step 18: Add to keywords.yaml**

In `_datafiles/world/dogmud/keywords.yaml`, add `throw` under `combat:`:

```yaml
    combat:
      ...
      - throw
```

- [ ] **Step 19: Update eat help file**

Replace `_datafiles/world/dogmud/templates/help/eat.template` to mention spoiling:

Read the current eat.template first, then update it to add a note about food spoiling. Add a section like:

```
<ansi fg="yellow">Spoiled Food:</ansi>

  Crafted food spoils over time, similar to potions. You
  cannot eat spoiled food. Spoiled food can be salvaged for
  <ansi fg="itemname">putrid residue</ansi>, a crafting material used
  in alchemy.
```

- [ ] **Step 20: Compile and run all tests**

Run: `go build ./... && go test ./internal/usercommands/ -v`
Expected: Clean build and all tests pass

- [ ] **Step 21: Commit**

```bash
git add internal/usercommands/throw.go internal/usercommands/throw_test.go \
  internal/usercommands/usercommands.go \
  _datafiles/world/dogmud/templates/help/throw.template \
  _datafiles/world/dogmud/templates/help/eat.template \
  _datafiles/world/dogmud/keywords.yaml
git commit -m "feat: add throw command for AoE grenade combat"
```

---

### Task 8: Final Integration — Full Test Suite + Build Verification

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -v 2>&1 | tail -50`
Expected: All tests pass

- [ ] **Step 2: Verify clean build**

Run: `go build ./...`
Expected: No errors

- [ ] **Step 3: Verify all new help files render**

Check that all new template files exist and have correct naming:
- `sort.template` (updated)
- `sell.template` (updated)
- `taunt.template` (updated)
- `eat.template` (updated)
- `warcry.template` (new)
- `rally.template` (new)
- `throw.template` (new)

- [ ] **Step 4: Verify keywords.yaml has all new entries**

Grep keywords.yaml for: `taunt`, `warcry`, `rally`, `throw` — all should appear under `combat:`.

- [ ] **Step 5: Verify all data files load correctly**

Check that all YAML files have valid syntax:
- 3 new buff YAMLs (77, 78, 79, 80)
- 4 new item YAMLs (30057, 30058, 30059, 40050)
- 3 new recipe YAMLs
- Updated golem mob YAML

- [ ] **Step 6: Final commit if any fixups needed**

```bash
git add -A
git commit -m "chore: integration fixups for QOL batch"
```
