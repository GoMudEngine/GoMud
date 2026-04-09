# Inventory Rework Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Overhaul inventory UX: diku-style item disambiguation, inventory stacking, carry capacity reduction with descriptive encumbrance, enchanting targeting, multi-buy, look direction fix, and worn item targeting.

**Architecture:** Bottom-up. Extend the parser first (`GetMatchNumber`), then build features on top. Each layer is independently testable and committable. Storage is unchanged — stacking is display-only.

**Tech Stack:** Go, Go templates, YAML config

---

## Task 1: Parser — N.item and all.item Support

**Files:**
- Modify: `internal/util/util.go:288-304`
- Modify: `internal/util/util_test.go` (if exists, add tests)

- [ ] **Step 1: Add tests for new parsing formats**

Find or create tests for `GetMatchNumber` in `internal/util/util_test.go`. Add these test cases:

```go
func TestGetMatchNumber_DikuFormat(t *testing.T) {
	// N.item format
	name, num := GetMatchNumber("3.dagger")
	if name != "dagger" || num != 3 {
		t.Errorf("expected (dagger, 3), got (%s, %d)", name, num)
	}

	// all.item format
	name, num = GetMatchNumber("all.dagger")
	if name != "dagger" || num != -1 {
		t.Errorf("expected (dagger, -1), got (%s, %d)", name, num)
	}

	// Existing hash format still works
	name, num = GetMatchNumber("dagger#2")
	if name != "dagger" || num != 2 {
		t.Errorf("expected (dagger, 2), got (%s, %d)", name, num)
	}

	// Non-numeric prefix with dot (not a match number)
	name, num = GetMatchNumber("st.elmo")
	if name != "st.elmo" || num != 1 {
		t.Errorf("expected (st.elmo, 1), got (%s, %d)", name, num)
	}

	// Plain item (no format)
	name, num = GetMatchNumber("dagger")
	if name != "dagger" || num != 1 {
		t.Errorf("expected (dagger, 1), got (%s, %d)", name, num)
	}

	// 1.item (explicit first)
	name, num = GetMatchNumber("1.sword")
	if name != "sword" || num != 1 {
		t.Errorf("expected (sword, 1), got (%s, %d)", name, num)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/util/ -run TestGetMatchNumber_DikuFormat -v`
Expected: FAIL (new formats not yet parsed)

- [ ] **Step 3: Implement N.item and all.item parsing**

In `internal/util/util.go`, replace the `GetMatchNumber` function (lines 288-304):

```go
// GetMatchNumber accepts an input and extracts a match number.
// Supports three formats:
//   - "N.item"   (diku-style) → ("item", N)
//   - "all.item"              → ("item", -1)  sentinel for "all matches"
//   - "item#N"   (existing)   → ("item", N)
//   - plain "item"            → ("item", 1)
func GetMatchNumber(input string) (string, int) {
	input = strings.TrimSpace(strings.ToLower(input))

	// Check for N.item or all.item prefix
	if dotIdx := strings.IndexByte(input, '.'); dotIdx > 0 {
		prefix := input[:dotIdx]
		rest := input[dotIdx+1:]
		if len(rest) > 0 {
			if prefix == "all" {
				return rest, -1
			}
			if n, err := strconv.Atoi(prefix); err == nil && n >= 1 {
				return rest, n
			}
		}
	}

	// Check for item#N suffix (existing logic)
	if strings.Contains(input, "#") {
		parts := strings.Split(input, "#")
		input = parts[0]
		inputNumber, _ := strconv.Atoi(strings.Join(parts[1:], "#"))
		if inputNumber < 1 {
			inputNumber = 1
		}
		return input, inputNumber
	}

	return input, 1
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/util/ -run TestGetMatchNumber -v`
Expected: PASS

- [ ] **Step 5: Build full project**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 6: Commit**

```bash
git add internal/util/util.go internal/util/util_test.go
git commit -m "$(cat <<'EOF'
feat: support N.item and all.item disambiguation formats

GetMatchNumber now parses diku-style "3.dagger" and "all.dagger"
in addition to existing "dagger#3". Returns -1 as sentinel for
"all matches". Non-numeric prefixes like "st.elmo" are not affected.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Inventory Stacking Display

**Files:**
- Modify: `internal/usercommands/inventory.go:115-146`
- Modify: `_datafiles/world/dogmud/templates/character/inventory.template`

- [ ] **Step 1: Add stacking logic to inventory command**

In `internal/usercommands/inventory.go`, replace the item name building loop (lines 115-128) with stacking logic. Replace from `for _, item := range itemList {` through the end of that loop (line 128 `}`):

```go
	// Build stack keys and group identical items
	type stackEntry struct {
		name          string
		nameFormatted string
		count         int
	}
	stackOrder := []string{}            // preserve display order
	stacks := map[string]*stackEntry{}

	for _, item := range itemList {
		iSpec := item.GetSpec()

		// Stack key: ItemId + enchant state + uses (for consumables)
		stackKey := fmt.Sprintf("%d|%s|%d|%d", item.ItemId, item.EnchantType, item.EnchantTier, item.Uses)

		if entry, exists := stacks[stackKey]; exists {
			entry.count++
			continue
		}

		iName := item.Name()
		iNameFormatted := fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, item.DisplayName())

		if iSpec.Subtype == items.Drinkable || iSpec.Subtype == items.Edible || iSpec.Subtype == items.Usable || iSpec.Type == items.Lockpicks {
			if iSpec.Uses > 0 {
				iName = fmt.Sprintf(`%s (%d)`, iName, item.Uses)
				iNameFormatted = fmt.Sprintf(`%s <ansi fg="uses-left">(%d)</ansi>`, iNameFormatted, item.Uses)
			}
		}

		stacks[stackKey] = &stackEntry{name: iName, nameFormatted: iNameFormatted, count: 1}
		stackOrder = append(stackOrder, stackKey)
	}

	for _, key := range stackOrder {
		entry := stacks[key]
		if entry.count > 1 {
			itemNames = append(itemNames, fmt.Sprintf(`%s (x%d)`, entry.name, entry.count))
			itemNamesFormatted = append(itemNamesFormatted, fmt.Sprintf(`%s <ansi fg="uses-left">(x%d)</ansi>`, entry.nameFormatted, entry.count))
		} else {
			itemNames = append(itemNames, entry.name)
			itemNamesFormatted = append(itemNamesFormatted, entry.nameFormatted)
		}
	}
```

- [ ] **Step 2: Build and verify**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 3: Commit**

```bash
git add internal/usercommands/inventory.go
git commit -m "$(cat <<'EOF'
feat: stack identical items in inventory display

Items with same ItemId, enchantment, and uses are grouped with (xN)
count. Storage is unchanged — stacking is display-only.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Carry Capacity Config + Reduction

**Files:**
- Modify: `internal/configs/config.balance.go:211-212` (add field) and Validate section
- Modify: `internal/characters/character.go:338-341`

- [ ] **Step 1: Add CarryCapacityMultiplier to Balance struct**

In `internal/configs/config.balance.go`, find the end of the Balance struct (line 211, before the closing `}`):

```go
	MoonStatModMax ConfigFloat `yaml:"MoonStatModMax"` // Max fractional stat modifier from moon phases, e.g. 0.05 = ±5% (default 0.05)
}
```

Replace with:

```go
	MoonStatModMax         ConfigFloat `yaml:"MoonStatModMax"`         // Max fractional stat modifier from moon phases, e.g. 0.05 = ±5% (default 0.05)
	CarryCapacityMultiplier ConfigFloat `yaml:"CarryCapacityMultiplier"` // Strength multiplier for carry capacity in lbs (default 0.65)
}
```

Then find the `Validate()` function and add validation. Find the last validation block before the closing `}` of Validate and add:

```go
	// ── CARRY CAPACITY ──────────────────────────────────────────────────────
	if b.CarryCapacityMultiplier < 0.1 || b.CarryCapacityMultiplier > 10.0 {
		b.CarryCapacityMultiplier = 0.65
	}
```

- [ ] **Step 2: Use config multiplier in CarryCapacity**

In `internal/characters/character.go`, replace lines 338-341:

```go
// CarryCapacity returns weight capacity in pounds (Strength × 3)
func (c *Character) CarryCapacity() float64 {
	return float64(c.Stats.Strength.ValueAdj) * 3.0
}
```

With:

```go
// CarryCapacity returns weight capacity in pounds (Strength × config multiplier)
func (c *Character) CarryCapacity() float64 {
	bal := configs.GetBalanceConfig()
	return float64(c.Stats.Strength.ValueAdj) * float64(bal.CarryCapacityMultiplier)
}
```

Ensure `configs` is imported. Check the existing imports in `character.go` — it likely already imports `configs`. If not, add:

```go
	"github.com/GoMudEngine/GoMud/internal/configs"
```

- [ ] **Step 3: Build and verify**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 4: Commit**

```bash
git add internal/configs/config.balance.go internal/characters/character.go
git commit -m "$(cat <<'EOF'
feat: configurable carry capacity multiplier (default 0.65)

Reduces baseline carry capacity from Str×3.0 to Str×0.65 (~78%
reduction). Tunable via Balance.CarryCapacityMultiplier in config.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Encumbrance Tier Display

**Files:**
- Modify: `internal/usercommands/inventory.go:139-146`
- Modify: `_datafiles/world/dogmud/templates/character/inventory.template`

- [ ] **Step 1: Add encumbrance tier helper**

In `internal/usercommands/inventory.go`, add this function before the `Inventory` function:

```go
// encumbranceTier returns a descriptive label and ANSI color for the
// player's current weight-to-capacity ratio.
func encumbranceTier(weight, capacity float64) (label, color string) {
	if capacity <= 0 {
		return "crushed", "magenta-bold"
	}
	ratio := weight / capacity
	switch {
	case ratio <= 0.25:
		return "light", "green"
	case ratio <= 0.50:
		return "moderate", "yellow"
	case ratio <= 0.75:
		return "heavy", "red"
	case ratio <= 1.00:
		return "overburdened", "red-bold"
	default:
		return "crushed", "magenta-bold"
	}
}
```

- [ ] **Step 2: Replace numeric count with tier label in template data**

In `internal/usercommands/inventory.go`, find the invData map (lines 139-147):

```go
	invData := map[string]any{
		`Equipment`:          &user.Character.Equipment,
		`ItemNames`:          itemNames,
		`ItemNamesFormatted`: itemNamesFormatted,
		`AttackDamage`:       diceRoll,
		`RaceInfo`:           raceInfo,
		`Searching`:          len(rest) > 0,
		`Count`:              fmt.Sprintf(`(%.1f/%.0f lbs)`, user.Character.GetCarriedWeight(), user.Character.CarryCapacity()),
	}
```

Replace with:

```go
	encLabel, encColor := encumbranceTier(user.Character.GetCarriedWeight(), user.Character.CarryCapacity())

	invData := map[string]any{
		`Equipment`:          &user.Character.Equipment,
		`ItemNames`:          itemNames,
		`ItemNamesFormatted`: itemNamesFormatted,
		`AttackDamage`:       diceRoll,
		`RaceInfo`:           raceInfo,
		`Searching`:          len(rest) > 0,
		`EncumbranceLabel`:   encLabel,
		`EncumbranceColor`:   encColor,
	}
```

- [ ] **Step 3: Update inventory template**

Replace the full contents of `_datafiles/world/dogmud/templates/character/inventory.template`:

```
{{ if not .Searching }} ┌─ <ansi fg="black-bold">.:</ansi><ansi fg="20">Equipment</ansi> ──────────────────────────────────────────────────────────────┐
{{ if not .Equipment.Weapon.IsDisabled }}   <ansi fg="yellow">Weapon:  </ansi><ansi fg="itemname">{{ .Equipment.Weapon.NameComplex  }}</ansi>
{{ end -}}
{{- if not .Equipment.Offhand.IsDisabled }}   <ansi fg="yellow">Offhand: </ansi><ansi fg="itemname">{{ .Equipment.Offhand.NameComplex }}</ansi>
{{ end -}}
{{- if not .Equipment.ExtraArm1.IsDisabled }}   <ansi fg="yellow">Arm 3:   </ansi><ansi fg="itemname">{{ .Equipment.ExtraArm1.NameComplex }}</ansi>
{{ end -}}
{{- if not .Equipment.ExtraArm2.IsDisabled }}   <ansi fg="yellow">Arm 4:   </ansi><ansi fg="itemname">{{ .Equipment.ExtraArm2.NameComplex }}</ansi>
{{ end -}}
{{- if not .Equipment.Head.IsDisabled }}   <ansi fg="yellow">Head:    </ansi><ansi fg="itemname">{{ .Equipment.Head.NameComplex    }}</ansi>
{{ end -}}
{{- if not .Equipment.Neck.IsDisabled }}   <ansi fg="yellow">Neck:    </ansi><ansi fg="itemname">{{ .Equipment.Neck.NameComplex    }}</ansi>
{{ end -}}
{{- if not .Equipment.Body.IsDisabled }}   <ansi fg="yellow">Body:    </ansi><ansi fg="itemname">{{ .Equipment.Body.NameComplex    }}</ansi>
{{ end -}}
{{- if not .Equipment.Belt.IsDisabled }}   <ansi fg="yellow">Belt:    </ansi><ansi fg="itemname">{{ .Equipment.Belt.NameComplex    }}</ansi>
{{ end -}}
{{- if not .Equipment.Gloves.IsDisabled }}   <ansi fg="yellow">Gloves:  </ansi><ansi fg="itemname">{{ .Equipment.Gloves.NameComplex  }}</ansi>
{{ end -}}
{{- if not .Equipment.Ring.IsDisabled }}   <ansi fg="yellow">Ring:    </ansi><ansi fg="itemname">{{ .Equipment.Ring.NameComplex    }}</ansi>
{{ end -}}
{{- if not .Equipment.Legs.IsDisabled }}   <ansi fg="yellow">Legs:    </ansi><ansi fg="itemname">{{ .Equipment.Legs.NameComplex    }}</ansi>
{{ end -}}
{{- if not .Equipment.Feet.IsDisabled }}   <ansi fg="yellow">Feet:    </ansi><ansi fg="itemname">{{ .Equipment.Feet.NameComplex    }}</ansi>
{{ end }} └────────────────────────────────────────────────────────────────────────────┘
 {{ $itemCt := len .ItemNames -}}{{ $formattedNames := .ItemNamesFormatted -}}{{- $strlen := 0 -}}{{- $lineCt := 1 -}}{{- $encTag := printf "Encumbrance: [%s]" .EncumbranceLabel -}}
 Carrying: {{ range $index, $name := .ItemNames -}}{{ $proposedLength := (add 2 (add $strlen (len $name))) }}{{- if gt $proposedLength 68 -}}{{- $strlen = 0 -}}{{- $lineCt = (add 1 $lineCt) -}}{{ if eq $lineCt 2 }}{{- printf "\n           " -}}{{ else }}{{- printf "\n           " -}}{{ end }}{{- end -}}{{ index $formattedNames $index }}{{- if ne $index (sub $itemCt 1) }}, {{ $strlen = (add 2 (add $strlen (len $name))) }}{{ end }}{{ end }}
 <ansi fg="{{ .EncumbranceColor }}">{{ $encTag }}</ansi>
{{ else }}
{{ $itemCt := len .ItemNames -}}{{ $formattedNames := .ItemNamesFormatted -}}
 Found in your bag: {{ range $index, $name := .ItemNames -}}{{  index $formattedNames $index }}
                   {{ end -}}
{{ end }}
```

- [ ] **Step 4: Build and verify**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 5: Commit**

```bash
git add internal/usercommands/inventory.go \
        _datafiles/world/dogmud/templates/character/inventory.template
git commit -m "$(cat <<'EOF'
feat: replace numeric weight display with colored encumbrance tiers

Inventory now shows colored labels (light/moderate/heavy/overburdened/
crushed) instead of raw lbs numbers. Removes internal values from
player-visible output.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Encumbrance Prompt Token

**Files:**
- Modify: `internal/users/userrecord.prompt.go`
- Modify: `_datafiles/world/dogmud/templates/help/set-prompt.template`

- [ ] **Step 1: Add {enc} token to ProcessPromptString**

In `internal/users/userrecord.prompt.go`, find the `ProcessPromptString` function's switch statement. Find a suitable location in the `Other` section of the switch. Look for a case like `case `{g}`:` (gold) and add the `{enc}` case nearby:

```go
		case `{enc}`:
			weight := u.Character.GetCarriedWeight()
			capacity := u.Character.CarryCapacity()
			var encLabel, encColor string
			if capacity <= 0 {
				encLabel, encColor = "crushed", "magenta-bold"
			} else {
				ratio := weight / capacity
				switch {
				case ratio <= 0.25:
					encLabel, encColor = "light", "green"
				case ratio <= 0.50:
					encLabel, encColor = "moderate", "yellow"
				case ratio <= 0.75:
					encLabel, encColor = "heavy", "red"
				case ratio <= 1.00:
					encLabel, encColor = "overburdened", "red-bold"
				default:
					encLabel, encColor = "crushed", "magenta-bold"
				}
			}
			promptOut.WriteString(fmt.Sprintf(`<ansi fg="%s">%s</ansi>`, encColor, encLabel))
```

Note: The encumbrance logic is duplicated from `inventory.go`'s `encumbranceTier` function. This is acceptable since the prompt file is a separate package (`users`) and extracting a shared utility would create a cross-package dependency for a 10-line switch. If this bothers you, extract the tier logic to `internal/characters/character.go` as a method on Character.

- [ ] **Step 2: Document {enc} in help file**

In `_datafiles/world/dogmud/templates/help/set-prompt.template`, find the `Other:` section (near the bottom). Find:

```
    <ansi fg="magenta">{casting}</ansi> While casting: spell name and fold progress. Empty when idle.
    <ansi fg="magenta">{\n}</ansi>    New Line
```

Replace with:

```
    <ansi fg="magenta">{enc}</ansi>   Encumbrance tier (light/moderate/heavy/overburdened/crushed)
    <ansi fg="magenta">{casting}</ansi> While casting: spell name and fold progress. Empty when idle.
    <ansi fg="magenta">{\n}</ansi>    New Line
```

- [ ] **Step 3: Build and verify**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 4: Commit**

```bash
git add internal/users/userrecord.prompt.go \
        _datafiles/world/dogmud/templates/help/set-prompt.template
git commit -m "$(cat <<'EOF'
feat: add {enc} prompt token for encumbrance tier display

Players can add {enc} to their prompt to see colored encumbrance
status inline. Off by default.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Look Direction Priority Fix

**Files:**
- Modify: `internal/usercommands/look.go:240-252`

- [ ] **Step 1: Add early-out for direction aliases**

In `internal/usercommands/look.go`, find the exit check block ending around line 284 (after `return true, nil` closing the `if exitName != ""` block). Right after it, before the backpack check comment, add:

```go
	// If the input is a recognized direction alias but no exit exists,
	// stop here — never fall through to item/mob matching.
	if alias := keywords.TryDirectionAlias(lookAt); alias != lookAt {
		user.SendText("There is no exit in that direction.")
		return true, nil
	}
```

This goes between the exit block's closing `}` and the `// Check for anything in their backpack` comment.

- [ ] **Step 2: Build and verify**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 3: Commit**

```bash
git add internal/usercommands/look.go
git commit -m "$(cat <<'EOF'
fix: prevent look direction aliases from matching inventory items

When a player types 'l n' and no north exit exists, the command
now shows 'no exit in that direction' instead of fuzzy-matching
inventory items starting with 'n'.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Unified FindItem for Worn Item Targeting

**Files:**
- Modify: `internal/characters/character.go` (add FindItem method)
- Modify: `internal/usercommands/look.go:288-320`
- Modify: `internal/usercommands/identify.go` (or wherever `id` command lives)

- [ ] **Step 1: Add FindItem method to Character**

In `internal/characters/character.go`, add this method after `FindOnBody` (around line 1463):

```go
// FindItem searches backpack and equipped items as a single pool for
// disambiguation. Returns the item, a source description, and whether found.
// The source is "in your backpack" or a slot name like "wielded", "worn - body", etc.
func (c *Character) FindItem(itemName string) (items.Item, string, bool) {
	if itemName == "" {
		return items.Item{}, "", false
	}

	// Build a single pool: backpack items + equipped items with source labels
	type candidate struct {
		item   items.Item
		source string
	}
	pool := []candidate{}

	for _, item := range c.Items {
		if item.ItemId > 0 {
			pool = append(pool, candidate{item, "in your backpack"})
		}
	}

	slotItems := []struct {
		item   items.Item
		source string
	}{
		{c.Equipment.Weapon, "wielded"},
		{c.Equipment.Offhand, "offhand"},
		{c.Equipment.ExtraArm1, "extra arm"},
		{c.Equipment.ExtraArm2, "extra arm"},
		{c.Equipment.Head, "worn - head"},
		{c.Equipment.Neck, "worn - neck"},
		{c.Equipment.Body, "worn - body"},
		{c.Equipment.Belt, "worn - belt"},
		{c.Equipment.Gloves, "worn - gloves"},
		{c.Equipment.Ring, "worn - ring"},
		{c.Equipment.Legs, "worn - legs"},
		{c.Equipment.Feet, "worn - feet"},
	}
	for _, slot := range slotItems {
		if slot.item.ItemId > 0 {
			pool = append(pool, candidate{slot.item, slot.source})
		}
	}

	// Extract just the items for FindMatchIn
	poolItems := make([]items.Item, len(pool))
	for i, c := range pool {
		poolItems[i] = c.item
	}

	closeMatch, fullMatch := items.FindMatchIn(itemName, poolItems...)

	if fullMatch.ItemId != 0 {
		// Find which pool entry matched
		for _, c := range pool {
			if c.item.ItemId == fullMatch.ItemId && c.item.UUID() == fullMatch.UUID() {
				return fullMatch, c.source, true
			}
		}
		return fullMatch, "in your backpack", true
	}

	if closeMatch.ItemId != 0 {
		for _, c := range pool {
			if c.item.ItemId == closeMatch.ItemId && c.item.UUID() == closeMatch.UUID() {
				return closeMatch, c.source, true
			}
		}
		return closeMatch, "in your backpack", true
	}

	return items.Item{}, "", false
}
```

Note: Check that `items.Item` has a `UUID()` method or similar unique identifier. If not, match by index position in the pool instead.

- [ ] **Step 2: Update look command to use FindItem**

In `internal/usercommands/look.go`, find the backpack/body check block (around lines 288-294):

```go
	lookItem, foundItem := user.Character.FindInBackpack(lookAt)
	lookDestination := `in your backpack`
	if !foundItem {
		// Check for any equipment they are wearing they might want to look at
		lookItem, foundItem = user.Character.FindOnBody(lookAt)
		lookDestination = `you are wearing`
	}
```

Replace with:

```go
	lookItem, lookDestination, foundItem := user.Character.FindItem(lookAt)
```

- [ ] **Step 3: Update identify command**

Find the identify command (search for `identify.go` or the function handling `id`). Locate where it calls `FindInBackpack` and/or `FindOnBody` and replace with `FindItem` using the same pattern as Step 2.

- [ ] **Step 4: Build and verify**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 5: Commit**

```bash
git add internal/characters/character.go \
        internal/usercommands/look.go \
        internal/usercommands/identify.go
git commit -m "$(cat <<'EOF'
feat: unified FindItem searches backpack + equipped as single pool

Players can now target equipped items with look/identify even when
the same base item exists in backpack. Uses N.item or item#N
disambiguation across the combined pool.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Multi-Buy

**Files:**
- Modify: `internal/usercommands/buy.go:20-40`

- [ ] **Step 1: Add quantity parsing to Buy function**

In `internal/usercommands/buy.go`, find the beginning of the `Buy` function after the empty check (around lines 26-30). Find:

```go
	targetMobInstanceId := 0
	targetUserId := 0

	itemname := rest
```

Replace with:

```go
	targetMobInstanceId := 0
	targetUserId := 0

	// Parse optional leading quantity: "buy 5 iron ingot"
	quantity := 1
	itemname := rest
	args0 := strings.SplitN(strings.TrimSpace(rest), " ", 2)
	if len(args0) == 2 {
		if n, err := strconv.Atoi(args0[0]); err == nil && n >= 1 {
			quantity = n
			itemname = args0[1]
			rest = args0[1] // Update rest so "from" parsing works on the remainder
		}
	}
```

Ensure `strconv` is imported. Check existing imports — add if missing:

```go
	"strconv"
```

Then find the section where purchases are attempted. The current code has single `tryPurchase` calls followed by `return true, nil`. We need to wrap the purchase logic in a loop. Find the first merchant purchase block (around lines 72-78):

```go
		if success = tryPurchase(itemname, user, room, nil, shopUser); success {
			user.Character.OnStatUse("charisma", user.UserId)
			return true, nil
		}
```

Replace with:

```go
		if tryPurchase(itemname, user, room, nil, shopUser) {
			purchased := 1
			for purchased < quantity {
				if !tryPurchase(itemname, user, room, nil, shopUser) {
					break
				}
				purchased++
			}
			if quantity > 1 && purchased < quantity {
				user.SendText(fmt.Sprintf(`<ansi fg="yellow">Purchased %d of %d before running short.</ansi>`, purchased, quantity))
			}
			success = true
			user.Character.OnStatUse("charisma", user.UserId)
			return true, nil
		}
```

Apply the same pattern to the mob merchant block (around lines 96-101):

```go
		if success = tryPurchase(itemname, user, room, shopMob, nil); success {
			user.Character.OnStatUse("charisma", user.UserId)
			// Stage 38.5.5: Merchant mob gains charisma from trade interactions
			shopMob.Character.OnStatUse("charisma", 0)
			return true, nil
		}
```

Replace with:

```go
		if tryPurchase(itemname, user, room, shopMob, nil) {
			purchased := 1
			for purchased < quantity {
				if !tryPurchase(itemname, user, room, shopMob, nil) {
					break
				}
				purchased++
			}
			if quantity > 1 && purchased < quantity {
				user.SendText(fmt.Sprintf(`<ansi fg="yellow">Purchased %d of %d before running short.</ansi>`, purchased, quantity))
			}
			success = true
			user.Character.OnStatUse("charisma", user.UserId)
			shopMob.Character.OnStatUse("charisma", 0)
			return true, nil
		}
```

- [ ] **Step 2: Build and verify**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 3: Commit**

```bash
git add internal/usercommands/buy.go
git commit -m "$(cat <<'EOF'
feat: support multi-buy with quantity prefix

Players can now type 'buy 5 iron ingot' to purchase multiple copies
in one command. Stops early if funds or capacity run out, with a
partial purchase message.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Enchanting Disambiguation

**Files:**
- Modify: `internal/crafting/crafting.go:206-219`
- Modify: `internal/usercommands/craft.go:70-78`
- Modify: `internal/hooks/NewRound_UserRoundTick.go:264-277`

- [ ] **Step 1: Update FindTargetItem to support specifier and equipment**

In `internal/crafting/crafting.go`, replace the `FindTargetItem` function (lines 206-219):

```go
// FindTargetItem searches the player's inventory and equipment for items
// matching the recipe's target_type. If specifier is non-empty, filters
// by item name. Returns all matching items with their source info.
func FindTargetItems(inv []items.Item, equipment *characters.Worn, targetType string, specifier string) []TargetCandidate {
	var candidates []TargetCandidate

	// Search backpack
	for i, item := range inv {
		if item.ItemId < 1 {
			continue
		}
		spec := item.GetSpec()
		if string(spec.Type) != targetType {
			continue
		}
		if specifier != "" && !strings.Contains(strings.ToLower(item.DisplayName()), strings.ToLower(specifier)) {
			continue
		}
		candidates = append(candidates, TargetCandidate{
			BackpackIdx: i,
			Item:        item,
			Source:      "backpack",
			SourceLabel: "",
		})
	}

	// Search equipment slots
	if equipment != nil {
		slots := []struct {
			item  items.Item
			label string
		}{
			{equipment.Weapon, "wielded"},
			{equipment.Offhand, "offhand"},
			{equipment.Head, "worn - head"},
			{equipment.Neck, "worn - neck"},
			{equipment.Body, "worn - body"},
			{equipment.Belt, "worn - belt"},
			{equipment.Gloves, "worn - gloves"},
			{equipment.Ring, "worn - ring"},
			{equipment.Legs, "worn - legs"},
			{equipment.Feet, "worn - feet"},
		}
		for _, slot := range slots {
			if slot.item.ItemId < 1 {
				continue
			}
			spec := slot.item.GetSpec()
			if string(spec.Type) != targetType {
				continue
			}
			if specifier != "" && !strings.Contains(strings.ToLower(slot.item.DisplayName()), strings.ToLower(specifier)) {
				continue
			}
			candidates = append(candidates, TargetCandidate{
				BackpackIdx: -1,
				Item:        slot.item,
				Source:      "equipped",
				SourceLabel: slot.label,
			})
		}
	}

	return candidates
}

// TargetCandidate represents a potential enchanting target.
type TargetCandidate struct {
	BackpackIdx int        // Index in backpack slice, or -1 if equipped
	Item        items.Item
	Source      string     // "backpack" or "equipped"
	SourceLabel string     // e.g. "wielded", "worn - body" (empty for backpack)
}

// FindTargetItem is a backward-compatible wrapper. Returns the first match.
func FindTargetItem(inv []items.Item, targetType string) (int, bool) {
	candidates := FindTargetItems(inv, nil, targetType, "")
	if len(candidates) > 0 {
		return candidates[0].BackpackIdx, true
	}
	return -1, false
}
```

Ensure `strings` is imported in `crafting.go`. Add `items` import if not present:

```go
	"github.com/GoMudEngine/GoMud/internal/items"
```

And add the `characters` import for the `Worn` type:

```go
	"github.com/GoMudEngine/GoMud/internal/characters"
```

- [ ] **Step 2: Update craft.go to parse specifier and show disambiguation**

In `internal/usercommands/craft.go`, find the enchanting target check (lines 70-78):

```go
	// Enchanting: target item check
	if crafting.IsEnchantingRecipe(recipe) {
		_, found := crafting.FindTargetItem(user.Character.Items, recipe.TargetType)
		if !found {
			user.SendText(fmt.Sprintf(
				`<ansi fg="red">You need a %s in your inventory to enchant.</ansi>`,
				strings.ReplaceAll(recipe.TargetType, "_", " ")))
			return true, nil
		}
	}
```

Replace with:

```go
	// Enchanting: target item check with disambiguation
	if crafting.IsEnchantingRecipe(recipe) {
		// Parse optional item specifier from remaining args after recipe name
		specifier := ""
		recipeArgs := strings.SplitN(rest, " ", 2)
		if len(recipeArgs) > 1 {
			specifier = strings.TrimSpace(recipeArgs[1])
		}

		candidates := crafting.FindTargetItems(user.Character.Items, &user.Character.Equipment, recipe.TargetType, specifier)

		if len(candidates) == 0 {
			user.SendText(fmt.Sprintf(
				`<ansi fg="red">You need a %s in your inventory or equipment to enchant.</ansi>`,
				strings.ReplaceAll(recipe.TargetType, "_", " ")))
			return true, nil
		}

		if len(candidates) > 1 && specifier == "" {
			// Ambiguous — show numbered list
			user.SendText(`<ansi fg="yellow">Which item do you want to enchant?</ansi>`)
			for i, c := range candidates {
				slotInfo := ""
				if c.SourceLabel != "" {
					slotInfo = fmt.Sprintf(` <ansi fg="yellow">(%s)</ansi>`, c.SourceLabel)
				}
				user.SendText(fmt.Sprintf(`  <ansi fg="white-bold">[%d]</ansi> <ansi fg="itemname">%s</ansi>%s`, i+1, c.Item.DisplayName(), slotInfo))
			}
			user.SendText(`  <ansi fg="white-bold">[0]</ansi> Cancel`)
			user.SendText(`Choose a number, or type the item name:`)
			// Store pending enchant state for next input
			user.Character.SetMiscData("pending-enchant-recipe", recipe.RecipeId)
			user.Character.SetMiscData("pending-enchant-count", len(candidates))
			return true, nil
		}
	}
```

Note: The pending enchant state handling (reading the player's next input as a selection) will require additional integration with the input handler. The implementer should check how other "question/answer" prompts work in the engine and follow that pattern. If no such pattern exists, the simplest approach is to require the specifier inline (`craft enchant keen dagger`) and skip the interactive prompt for now.

- [ ] **Step 3: Update enchanting completion to support equipped items**

In `internal/hooks/NewRound_UserRoundTick.go`, find lines 264-277 and update to use the new `FindTargetItems`:

```go
									if crafting.IsEnchantingRecipe(recipe) {
										candidates := crafting.FindTargetItems(user.Character.Items, &user.Character.Equipment, recipe.TargetType, "")
										if len(candidates) > 0 {
											c := candidates[0]
											eDef := enchantments.GetEnchantment(recipe.EnchantType)
											if eDef != nil {
												var targetItem *items.Item
												if c.BackpackIdx >= 0 {
													targetItem = &user.Character.Items[c.BackpackIdx]
												} else {
													targetItem = user.Character.Equipment.GetSlotPointer(c.SourceLabel)
												}
												if targetItem != nil {
													targetItem.EnchantType = recipe.EnchantType
													targetItem.EnchantTier = 0
													targetItem.EnchantUses = 0
													targetItem.ReservePool = eDef.ReservePool
													enchantments.ApplyTier(targetItem, eDef, 0)
												}
											}
										}
```

Note: Check if `Equipment` has a `GetSlotPointer(slotLabel)` method that returns `*items.Item`. If not, add a helper method to `Worn` in `worn.go` that maps slot label strings to pointers:

```go
func (w *Worn) GetSlotPointer(label string) *items.Item {
	switch label {
	case "wielded":
		return &w.Weapon
	case "offhand":
		return &w.Offhand
	case "worn - head":
		return &w.Head
	case "worn - neck":
		return &w.Neck
	case "worn - body":
		return &w.Body
	case "worn - belt":
		return &w.Belt
	case "worn - gloves":
		return &w.Gloves
	case "worn - ring":
		return &w.Ring
	case "worn - legs":
		return &w.Legs
	case "worn - feet":
		return &w.Feet
	}
	return nil
}
```

- [ ] **Step 4: Build and verify**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 5: Commit**

```bash
git add internal/crafting/crafting.go \
        internal/usercommands/craft.go \
        internal/hooks/NewRound_UserRoundTick.go \
        internal/characters/worn.go
git commit -m "$(cat <<'EOF'
feat: enchanting disambiguation with equipped item support

Players can now specify which item to enchant with 'craft <recipe>
<item-name>'. Enchanting searches both backpack and equipped items.
Shows numbered list when multiple targets match.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: Smoke Test

- [ ] **Step 1: Full build**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 2: Run tests**

Run: `go test ./...`
Expected: All tests pass.

- [ ] **Step 3: Manual verification checklist**

Start the server and verify:
1. **N.item**: `get 2.dagger` picks up second dagger on ground
2. **all.item**: `drop all.potion` drops all potions
3. **Inventory stacking**: Five iron ingots show as `iron ingot (x5)`
4. **Encumbrance display**: Inventory shows colored tier label instead of numbers
5. **{enc} prompt**: `set prompt {enc} >` shows encumbrance tier in prompt
6. **Carry capacity**: Verify reduced capacity (~65 lbs at Str 100)
7. **Look direction**: `l n` with no north exit shows "no exit" not inventory match
8. **Worn item targeting**: `look dagger#2` can target wielded dagger when one is in backpack
9. **Multi-buy**: `buy 5 iron ingot` buys 5 copies
10. **Enchanting**: `craft enchant keen dagger` targets specific dagger; equipped items appear in list
