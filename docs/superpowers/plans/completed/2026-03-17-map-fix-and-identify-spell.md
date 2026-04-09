# Map Fix + Identify Spell Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development
> (if subagents available) or superpowers:executing-plans to implement this plan.
> Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix legacy stat scaling in the map command and replace the broken
inspect command with an Identify spell that uses descriptive language.

**Architecture:** Two independent changes. (1) Map: threshold swap in one
file. (2) Identify: delete inspect command + files, add three template helper
functions, create new identify spell YAML/JS, add Go-side identify effect
handler in spell_resolution.go, rework the shared template, update appraise
to use the new template.

**Tech Stack:** Go, Go templates, JS (goja scripting engine), YAML spell defs

**Spec:** `docs/superpowers/specs/2026-03-17-map-fix-and-identify-spell-design.md`

---

## Chunk 1: Map Threshold Fix

### Task 1: Fix Map Perception Thresholds

**Files:**
- Modify: `internal/usercommands/skill.map.go:31-39`

- [ ] **Step 1: Update thresholds**

Change the three Perception breakpoints from 25/50/75 to 110/135/175:

```go
perceptionAdj := user.Character.Stats.Perception.ValueAdj
skillLevel := 1
if perceptionAdj >= 175 {
    skillLevel = 4
} else if perceptionAdj >= 135 {
    skillLevel = 3
} else if perceptionAdj >= 110 {
    skillLevel = 2
}
```

Note: the order must be highest-first (175, 135, 110) since these are
`if/else if` checks.

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: clean build, no errors

- [ ] **Step 3: Commit**

```bash
git add internal/usercommands/skill.map.go
git commit -m "fix: rescale map perception thresholds for 100-baseline stats

Old thresholds (25/50/75) gave every character max zoom with
100-baseline stats. New thresholds (110/135/175) restore meaningful
progression."
```

---

## Chunk 2: Template Helper Functions

### Task 2: Add damageQuality Template Function

**Files:**
- Modify: `internal/templates/templatesfunctions.go` (in the `funcMap` var)
- Modify: `internal/templates/templatesfunctions_test.go`

- [ ] **Step 1: Write failing tests**

Add to `templatesfunctions_test.go`:

```go
func TestDamageQuality(t *testing.T) {
	// Extract the function from funcMap
	fn := funcMap["damageQuality"].(func(float64) string)

	tests := []struct {
		name string
		mult float64
		want string
	}{
		{"zero", 0.0, "negligible striking power"},
		{"feeble low", 0.3, "feeble striking power"},
		{"feeble mid", 0.5, "feeble striking power"},
		{"light low", 0.6, "light striking power"},
		{"light high", 0.99, "light striking power"},
		{"moderate low", 1.0, "moderate striking power"},
		{"moderate high", 1.49, "moderate striking power"},
		{"strong low", 1.5, "strong striking power"},
		{"strong high", 2.49, "strong striking power"},
		{"devastating low", 2.5, "devastating striking power"},
		{"devastating high", 3.99, "devastating striking power"},
		{"legendary", 4.0, "legendary striking power"},
		{"legendary high", 10.0, "legendary striking power"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, fn(tt.mult))
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/templates/ -run TestDamageQuality -v`
Expected: FAIL — `damageQuality` key not in funcMap

- [ ] **Step 3: Implement damageQuality**

Add to `funcMap` in `templatesfunctions.go`, after the existing
`mitigationQuality` entry (around line 296):

```go
"damageQuality": func(mult float64) string {
    switch {
    case mult < 0.3:
        return "negligible striking power"
    case mult < 0.6:
        return "feeble striking power"
    case mult < 1.0:
        return "light striking power"
    case mult < 1.5:
        return "moderate striking power"
    case mult < 2.5:
        return "strong striking power"
    case mult < 4.0:
        return "devastating striking power"
    default:
        return "legendary striking power"
    }
},
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/templates/ -run TestDamageQuality -v`
Expected: PASS

### Task 3: Add spellDamageQuality Template Function

**Files:**
- Modify: `internal/templates/templatesfunctions.go`
- Modify: `internal/templates/templatesfunctions_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestSpellDamageQuality(t *testing.T) {
	fn := funcMap["spellDamageQuality"].(func(float64) string)

	tests := []struct {
		name string
		mult float64
		want string
	}{
		{"zero", 0.0, "negligible arcane resonance"},
		{"faint", 0.5, "faint arcane resonance"},
		{"mild", 0.8, "mild arcane resonance"},
		{"moderate", 1.2, "moderate arcane resonance"},
		{"strong", 1.6, "strong arcane resonance"},
		{"intense", 2.5, "intense arcane resonance"},
		{"legendary", 4.0, "legendary arcane resonance"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, fn(tt.mult))
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/templates/ -run TestSpellDamageQuality -v`
Expected: FAIL

- [ ] **Step 3: Implement spellDamageQuality**

Add to `funcMap` after `damageQuality`:

```go
"spellDamageQuality": func(mult float64) string {
    switch {
    case mult < 0.5:
        return "negligible arcane resonance"
    case mult < 0.8:
        return "faint arcane resonance"
    case mult < 1.2:
        return "mild arcane resonance"
    case mult < 1.6:
        return "moderate arcane resonance"
    case mult < 2.5:
        return "strong arcane resonance"
    case mult < 4.0:
        return "intense arcane resonance"
    default:
        return "legendary arcane resonance"
    }
},
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/templates/ -run TestSpellDamageQuality -v`
Expected: PASS

### Task 4: Add statModDescription Template Function

**Files:**
- Modify: `internal/templates/templatesfunctions.go`
- Modify: `internal/templates/templatesfunctions_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestStatModDescription(t *testing.T) {
	fn := funcMap["statModDescription"].(func(string, int) string)

	tests := []struct {
		name     string
		stat     string
		value    int
		want     string
	}{
		{"large positive", "strength", 25, "greatly bolsters your strength"},
		{"medium positive", "dexterity", 15, "bolsters your dexterity"},
		{"small positive", "perception", 3, "slightly bolsters your perception"},
		{"large negative", "vitality", -25, "greatly saps your vitality"},
		{"medium negative", "willpower", -15, "saps your willpower"},
		{"small negative", "charisma", -3, "slightly saps your charisma"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, fn(tt.stat, tt.value))
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/templates/ -run TestStatModDescription -v`
Expected: FAIL

- [ ] **Step 3: Implement statModDescription**

Add to `funcMap` after `spellDamageQuality`:

```go
"statModDescription": func(statName string, value int) string {
    var prefix string
    switch {
    case value >= 20:
        prefix = "greatly bolsters"
    case value >= 10:
        prefix = "bolsters"
    case value >= 1:
        prefix = "slightly bolsters"
    case value <= -20:
        prefix = "greatly saps"
    case value <= -10:
        prefix = "saps"
    case value <= -1:
        prefix = "slightly saps"
    default:
        return ""
    }
    return prefix + " your " + statName
},
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/templates/ -run TestStatModDescription -v`
Expected: PASS

- [ ] **Step 5: Commit all three helper functions**

```bash
git add internal/templates/templatesfunctions.go internal/templates/templatesfunctions_test.go
git commit -m "feat: add damageQuality, spellDamageQuality, statModDescription template helpers

Descriptive text bands for item properties. Used by the new identify
spell template and appraise command."
```

---

## Chunk 3: Delete Inspect, Create Identify Template

**Important ordering note:** Task 7 (update appraise) must happen
BEFORE Task 6 (delete old templates) to avoid breaking appraise
between steps. The sequence is: delete inspect command (Task 5) →
update appraise (Task 7) → create new template + delete old ones
(Task 6).

### Task 5: Delete Inspect Command and Help Files

**Files:**
- Delete: `internal/usercommands/skill.inspect.go`
- Modify: `internal/usercommands/usercommands.go` (remove line 98)
- Delete: `_datafiles/world/default/templates/help/inspect.template`
- Delete: `_datafiles/world/dogmud/templates/help/inspect.template`
- Delete: `_datafiles/world/empty/templates/help/inspect.template`

- [ ] **Step 1: Remove inspect registration from usercommands.go**

In `internal/usercommands/usercommands.go`, delete this line (around
line 98):

```go
		`inspect`:     {Inspect, false, true, false},
```

- [ ] **Step 2: Delete skill.inspect.go**

```bash
rm internal/usercommands/skill.inspect.go
```

- [ ] **Step 3: Delete inspect help templates**

```bash
rm _datafiles/world/default/templates/help/inspect.template
rm _datafiles/world/dogmud/templates/help/inspect.template
rm _datafiles/world/empty/templates/help/inspect.template
```

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: clean build (no references to `Inspect` remain — `appraise.go`
doesn't reference the function, only the shared template)

- [ ] **Step 5: Commit**

```bash
git add -A internal/usercommands/skill.inspect.go internal/usercommands/usercommands.go
git add _datafiles/world/default/templates/help/inspect.template _datafiles/world/dogmud/templates/help/inspect.template _datafiles/world/empty/templates/help/inspect.template
git commit -m "refactor: remove inspect command

Replaced by the identify spell. Appraise at merchants remains as
the non-magical alternative."
```

### Task 6: Update Appraise to Use New Template

**Files:**
- Modify: `internal/usercommands/appraise.go`

- [ ] **Step 1: Update appraise to use identify template**

In `appraise.go`, replace the `inspectDetails` struct and template call.

Replace lines 47-57:
```go
		type inspectDetails struct {
			InspectLevel int
			Item         *items.Item
			ItemSpec     *items.ItemSpec
		}

		details := inspectDetails{
			InspectLevel: 3,
			Item:         &item,
			ItemSpec:     &itemSpec,
		}
```

With:
```go
		type identifyDetails struct {
			Item     *items.Item
			ItemSpec *items.ItemSpec
		}

		details := identifyDetails{
			Item:     &item,
			ItemSpec: &itemSpec,
		}
```

Also change line 79 from:
```go
		inspectTxt, _ := templates.Process("descriptions/inspect", details, user.UserId)
```
To:
```go
		inspectTxt, _ := templates.Process("descriptions/identify", details, user.UserId)
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: clean build

- [ ] **Step 3: Commit**

```bash
git add internal/usercommands/appraise.go
git commit -m "refactor: update appraise to use new identify template

Removes InspectLevel gating — merchant appraisal now shows full
descriptive output."
```

### Task 7: Create Identify Template (Replaces Inspect Template)

**Files:**
- Create: `_datafiles/world/dogmud/templates/descriptions/identify.template`
  (replaces the old `inspect.template`)
- Delete: `_datafiles/world/dogmud/templates/descriptions/inspect.template`
- Delete: `_datafiles/world/default/templates/descriptions/inspect.template`
- Delete: `_datafiles/world/empty/templates/descriptions/inspect.template`

- [ ] **Step 1: Create the new identify template**

Write `_datafiles/world/dogmud/templates/descriptions/identify.template`:

```
 ┌─ <ansi fg="black-bold">.:</ansi><ansi fg="20">Basic Info</ansi> ──────────────────────────────────────────────────────────────┐
   <ansi fg="yellow">Name:</ansi>        {{ padRight 53 (uc .Item.Name) }}
   <ansi fg="yellow">Description:</ansi> {{ splitstring .ItemSpec.Description 61 "                " }}
   <ansi fg="yellow">Type:</ansi>        {{ uc .ItemSpec.Type.String }} ({{ uc .ItemSpec.Subtype.String }})
   <ansi fg="yellow">Value:</ansi>       {{ padRight 53 ( printf "%d gold" .ItemSpec.Value ) }}
 └─────────────────────────────────────────────────────────────────────────────┘
 ┌─ <ansi fg="black-bold">.:</ansi><ansi fg="20">Combat Properties</ansi> ──────────────────────────────────────────────────────┐
   <ansi fg="yellow">Striking Power:</ansi>  {{ if ne .ItemSpec.Type.String "weapon" }}{{ padRight 47 "N/A" }}{{ else }}{{ padRight 47 (damageQuality .ItemSpec.DamageMultiplier) }}{{ end }}
   <ansi fg="yellow">Arcane Resonance:</ansi>{{ if eq .ItemSpec.SpellDamageMultiplier 0.0 }}{{ padRight 47 " N/A" }}{{ else }}{{ padRight 47 (printf " %s" (spellDamageQuality .ItemSpec.SpellDamageMultiplier)) }}{{ end }}
   <ansi fg="yellow">Physical:</ansi>    {{ if eq .ItemSpec.PhysicalMitigation 0 }}{{ padRight 53 "N/A" }}{{ else }}{{ padRight 53 (mitigationQuality (divFloat .ItemSpec.PhysicalMitigation 100)) }}{{ end }}
   <ansi fg="yellow">Magical:</ansi>     {{ if eq .ItemSpec.MagicalMitigation 0 }}{{ padRight 53 "N/A" }}{{ else }}{{ padRight 53 (mitigationQuality (divFloat .ItemSpec.MagicalMitigation 100)) }}{{ end }}
   <ansi fg="yellow">Conviction:</ansi>  {{ if eq .ItemSpec.ConvictionMitigation 0 }}{{ padRight 53 "N/A" }}{{ else }}{{ padRight 53 (mitigationQuality (divFloat .ItemSpec.ConvictionMitigation 100)) }}{{ end }}
 └─────────────────────────────────────────────────────────────────────────────┘
 ┌─ <ansi fg="black-bold">.:</ansi><ansi fg="20">Modifiers</ansi> ───────────────────────────────────────────────────────────────┐
{{- $hasMods := false }}
{{- range $statName, $qty := .ItemSpec.StatMods }}{{ $hasMods = true }}
   {{ statModDescription $statName $qty }}
{{- end }}
{{- if gt (len .ItemSpec.BuffIds) 0 }}{{ $hasMods = true }}
{{- range $idx, $buffId := .ItemSpec.BuffIds }}
   <ansi fg="yellow">Applies:</ansi>     <ansi fg="spellname">{{ buffname $buffId }}</ansi> - {{ buffduration $buffId }}
{{- end }}{{ end }}
{{- if not $hasMods }}
   None detected.
{{- end }}
 └─────────────────────────────────────────────────────────────────────────────┘
 ┌─ <ansi fg="black-bold">.:</ansi><ansi fg="20">Magical Effects</ansi> ─────────────────────────────────────────────────────────┐
{{- $hasMagic := false }}
{{- if .Item.IsCursed }}{{ $hasMagic = true }}
   It's <ansi fg="red-bold">CURSED!</ansi>{{ end }}
{{- if gt (len .ItemSpec.Element.String) 0 }}{{ $hasMagic = true }}
   <ansi fg="yellow">Element:</ansi>     {{ padRight 53 (uc .ItemSpec.Element.String) }}{{ end }}
{{- if gt (len .ItemSpec.Damage.CritBuffIds) 0 }}{{ $hasMagic = true }}
{{- range $idx, $buffId := .ItemSpec.Damage.CritBuffIds }}
   <ansi fg="yellow">Crits Apply:</ansi> <ansi fg="spellname">{{ buffname $buffId }}</ansi> - {{ buffduration $buffId }}
{{- end }}{{ end }}
{{- if not $hasMagic }}
   None detected.
{{- end }}
 └─────────────────────────────────────────────────────────────────────────────┘

```

- [ ] **Step 2: Delete old inspect description templates**

```bash
rm _datafiles/world/dogmud/templates/descriptions/inspect.template
rm _datafiles/world/default/templates/descriptions/inspect.template
rm _datafiles/world/empty/templates/descriptions/inspect.template
```

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/templates/descriptions/identify.template
git add _datafiles/world/dogmud/templates/descriptions/inspect.template _datafiles/world/default/templates/descriptions/inspect.template _datafiles/world/empty/templates/descriptions/inspect.template
git commit -m "feat: add descriptive identify template, remove old inspect templates

All item properties now use descriptive language instead of raw
numbers. Shared by the identify spell and appraise command."
```

---

## Chunk 4: Create Spell, Add Handler

### Task 8: Create Identify Spell YAML and JS

**Files:**
- Create: `_datafiles/world/dogmud/spells/identify.yaml`
- Create: `_datafiles/world/dogmud/spells/identify.js`

- [ ] **Step 1: Create spell YAML**

Write `_datafiles/world/dogmud/spells/identify.yaml`:

```yaml
spellid: identify
name: Identify
description: >
  Reach out with your mind to sense the hidden properties
  of an item you are carrying or wearing.
type: neutral
schools:
  - mental
cost: 20
waitrounds: 30
primarystat: willpower
base_folds: 3
difficulty: 0
effect_type: identify
```

- [ ] **Step 2: Create spell JS (minimal — casting flavor text only)**

Write `_datafiles/world/dogmud/spells/identify.js`:

```javascript
function onCast(actor, itemName) {
    actor.SendText(
        "You focus your mind, reaching out to sense the "
        + "item's essence..."
    );
    return true;
}
```

The `onMagic` callback is not needed — the Go-side `identify` effect
handler in `spell_resolution.go` handles resolution.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/spells/identify.yaml _datafiles/world/dogmud/spells/identify.js
git commit -m "feat: add identify spell definition

Mental school, neutral type. 20 conviction, 3 folds, 30-round
cooldown. Resolution handled Go-side in spell_resolution.go."
```

### Task 9: Add Identify Effect Handler in spell_resolution.go

**Files:**
- Modify: `internal/hooks/spell_resolution.go`

The identify effect must be handled in `resolveSpell()` BEFORE the
mob/player target loops, since neutral spells have no targets — they
operate on the caster's own items.

- [ ] **Step 1: Add identify handler**

In `spell_resolution.go`, at the top of `resolveSpell()` (after
line 26, before the HarmArea population block on line 29), add:

```go
	// --- Identify: resolve against caster's item, no targets ---
	if spellData.EffectType == "identify" {
		resolveIdentify(user, cs.SpellRest, room)
		return
	}
```

Then add the `resolveIdentify` function at the bottom of the file:

```go
// resolveIdentify finds the named item on the caster and renders
// the identify template with descriptive item properties.
func resolveIdentify(user *users.UserRecord, itemName string, room *rooms.Room) {

	if itemName == "" {
		user.SendText("Identify what? (Usage: cast identify <item>)")
		return
	}

	// Search backpack first, then equipped slots
	matchItem, found := user.Character.FindInBackpack(itemName)
	if !found {
		matchItem, found = user.Character.FindOnBody(itemName)
	}

	if !found {
		user.SendText("You can't seem to identify that.")
		return
	}

	iSpec := matchItem.GetSpec()

	type identifyDetails struct {
		Item     *items.Item
		ItemSpec *items.ItemSpec
	}

	details := identifyDetails{
		Item:     &matchItem,
		ItemSpec: &iSpec,
	}

	user.SendText(
		fmt.Sprintf(`You concentrate on the <ansi fg="item">%s</ansi>...`,
			matchItem.DisplayName()),
	)
	room.SendText(
		fmt.Sprintf(
			`<ansi fg="username">%s</ansi> concentrates on their <ansi fg="item">%s</ansi>...`,
			user.Character.Name, matchItem.DisplayName()),
		user.UserId,
	)

	identifyTxt, _ := templates.Process("descriptions/identify", details, user.UserId)
	user.SendText(identifyTxt)
}
```

Make sure the imports at the top of the file include
`"github.com/GoMudEngine/GoMud/internal/templates"` — check whether
it's already imported. If not, add it.

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: clean build

- [ ] **Step 3: Commit**

```bash
git add internal/hooks/spell_resolution.go
git commit -m "feat: add identify effect handler in spell resolution

Handles neutral-type identify spell: searches backpack then equipped
items, renders descriptive identify template to caster."
```

### Task 10: Create Identify Help File

**Files:**
- Create: `_datafiles/world/dogmud/templates/help/identify.template`

- [ ] **Step 1: Create help template**

Write `_datafiles/world/dogmud/templates/help/identify.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="spellname">identify</ansi> (spell)

The <ansi fg="spellname">identify</ansi> spell reveals the hidden
properties of an item you are carrying or wearing.

<ansi fg="yellow">Usage:</ansi>

  <ansi fg="command">cast identify [item_name]</ansi>

Requires the <ansi fg="skill">spellcasting</ansi> skill. Costs
conviction to cast.

For a non-magical alternative, visit a merchant and use the
<ansi fg="command">appraise</ansi> command.
```

- [ ] **Step 2: Commit**

```bash
git add _datafiles/world/dogmud/templates/help/identify.template
git commit -m "docs: add identify spell help file"
```

### Task 11: Final Verification

- [ ] **Step 1: Full build**

Run: `go build ./...`
Expected: clean build

- [ ] **Step 2: Run all template tests**

Run: `go test ./internal/templates/ -v`
Expected: all tests pass (including new helper function tests)

- [ ] **Step 3: Run full test suite**

Run: `go test ./...`
Expected: no regressions (any inspect-related test failures should
have been caught by now — if they exist, delete or update them)
