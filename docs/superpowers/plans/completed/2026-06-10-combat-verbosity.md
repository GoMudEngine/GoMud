# Combat Text Verbosity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Per-user `combatverbosity` setting (full/medium/light) gating auto-attack round narration, with spectated fights one step lower and a light-mode compact per-round tally — per `docs/superpowers/specs/2026-06-10-combat-verbosity-design.md`.

**Architecture:** Verbosity primitives live in `internal/messaging` (types + suppression table + a new `CategoryCombatSummary`). The gate is applied at the combat round's message drain (`dispatchCritAndMessaging` in `internal/hooks/NewRound_DoCombat_unified.go`), where recipient role, line category, and the full `AttackResult` are all in hand. A per-round tally aggregator in a new `internal/hooks/combat_verbosity.go` accumulates suppressed-to-aggregate fights and flushes one compact line per fight pair at the end of `DoCombat`. The messaging pipeline itself is untouched.

**Tech Stack:** Go, existing messaging framework (`internal/messaging`), combat hooks (`internal/hooks`), `combat.AttackResult`/`SwingEvents`, `set` user command.

**Branch:** `feature/combat-verbosity` off `master`.

**Verified API facts (do not re-derive):**
- Drain point: `dispatchCritAndMessaging(atk, def actions.Actor, res *combat.AttackResult)` — `internal/hooks/NewRound_DoCombat_unified.go:464`. Participant drains at lines ~514-523 (`res.MessagesToSource` → `atk.SendText` if player; `res.MessagesToTarget` → `def.SendText` if player). Spectator drains at ~527-532: `sendVisualRoomText(atkRoom, msg.Category, msg.Text, excludes...)` for `MessagesToSourceRoom`, same with `defRoom` for `MessagesToTargetRoom`, where `excludes := playerExcludeIds(atk, def)`.
- `sendVisualRoomText` (NewRound_DoCombat_helpers.go:71) wraps `room.SendTextVisual(cat, msg, excludeUserIds...)`. Per-recipient single-user variant exists: `room.SendTextVisualToUser(u *users.UserRecord, cat, txt)` (internal/rooms/rooms.go:330).
- Round entry: `DoCombat(e events.Event)` — `internal/hooks/NewRound_DoCombat.go:23`; runs `handlePlayerCombat`, `handleMobCombat`, `handleAffected`, retarget pass, then returns. Flush goes at the end, before `return`.
- Wait-round drain (`handleCombatWaitRound`, NewRound_DoCombat_resolution.go) carries "you wait" text, not hit/defense lines — left ungated.
- `combat.AttackResult` (internal/combat/attackresult.go): `MessagesTo*` are `[]TaggedMessage{Category, Text}`; `SwingEvents []SwingEvent{Hit, Crit, Fumble, Damage, ...}`; `Hit bool`, `DamageToTarget int`. `TaggedMessage` categories: hits = `CategoryHit{Melee,Blunt,NaturalSharp,Ranged,Caster,Unarmed}` (from weapon subtype via `CategoryForWeaponSubtype`), defenses = `CategoryDodge/Parry/Block` (via `CategoryForDefenseVerb`). Dark-room substitutes preserve these tags. `MessagesToRoomOld` exists on the struct but is NOT drained in `dispatchCritAndMessaging` — leave whatever drains it (if anything) untouched.
- Crit-effect messages (riposte/sweep/bash, lines ~487-496) and `sendDarkRoomCombatFallback` are separate sends — status-changing / awareness floor → untouched.
- `asUser(actor)` helper exists in the hooks package. `actorSourceTarget`, `playerExcludeIds` exist.
- `combat.GetDamageDescription(damage, targetMaxHP) string` — internal/combat/descriptions.go:17.
- Character pools follow the `<Pool>Max.Value` shape (e.g. `StaminaMax.Value` in combat_helpers.go:104) — use `Character.HealthMax.Value` for max HP (verify the exact field name in internal/characters when first compiling; the stat-pool shape is confirmed).
- Settings pattern: `Set` command switch in `internal/usercommands/set.go:18` (case `linewidth` → `cmdSetLineWidth(user, args)` at set.go:480: validates, assigns `user.LineWidth`, confirms via SendText, emits `events.UserSettingChanged{UserId, Name}`). `UserRecord.LineWidth int` with yaml tag + `GetLineWidth()` fallback getter (internal/users/userrecord.go:53,443) is the field pattern to mirror.
- `messaging.Category` constants in internal/messaging/messaging.go:20-104 with `String()` switch starting line 108; color aliases in `_datafiles/world/dogmud/ansi-aliases.yaml` (e.g. `weather: 75`); prose-normalization skip set in internal/messaging/normalize.go (~line 28).
- Help system: template at `_datafiles/world/dogmud/templates/help/<name>.template` + a `- <name>` entry in `_datafiles/world/dogmud/keywords.yaml` (commands → appropriate category) for the `help` listing. See `weather.template` (added 2026-06-10) for the house format.
- Hook-level test scaffolding exists: `internal/hooks/NewRound_DoCombat_routing_test.go` and `hooks_test.go` (`dummyAttackResult` helper at hooks_test.go:2683).

---

### Task 0: Branch

- [ ] **Step 1:**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
git checkout -b feature/combat-verbosity master
```

(Working tree has unrelated runtime artifacts — never `git add -A` in any task.)

---

### Task 1: Verbosity primitives + CategoryCombatSummary (messaging package)

**Files:**
- Create: `internal/messaging/verbosity.go`
- Modify: `internal/messaging/messaging.go` (Category enum + String())
- Modify: `internal/messaging/normalize.go` (skip set — see step 5)
- Modify: `_datafiles/world/dogmud/ansi-aliases.yaml`
- Test: `internal/messaging/verbosity_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/messaging/verbosity_test.go`:

```go
package messaging

import "testing"

func TestParseVerbosity(t *testing.T) {
	cases := []struct {
		in   string
		want Verbosity
	}{
		{"", VerbosityFull},
		{"full", VerbosityFull},
		{"FULL", VerbosityFull},
		{"medium", VerbosityMedium},
		{"light", VerbosityLight},
		{"garbage", VerbosityFull}, // unknown → safe default
	}
	for _, c := range cases {
		if got := ParseVerbosity(c.in); got != c.want {
			t.Errorf("ParseVerbosity(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestVerbosityString(t *testing.T) {
	if VerbosityFull.String() != "full" || VerbosityMedium.String() != "medium" || VerbosityLight.String() != "light" {
		t.Errorf("Verbosity String() mismatch: %q %q %q",
			VerbosityFull.String(), VerbosityMedium.String(), VerbosityLight.String())
	}
}

func TestVerbosityOneStepLower(t *testing.T) {
	if VerbosityFull.OneStepLower() != VerbosityMedium {
		t.Error("full should step to medium")
	}
	if VerbosityMedium.OneStepLower() != VerbosityLight {
		t.Error("medium should step to light")
	}
	if VerbosityLight.OneStepLower() != VerbosityLight {
		t.Error("light should stay light")
	}
}

func TestVerbositySuppresses(t *testing.T) {
	hitCats := []Category{CategoryHitMelee, CategoryHitBlunt, CategoryHitNaturalSharp,
		CategoryHitRanged, CategoryHitCaster, CategoryHitUnarmed}
	defenseCats := []Category{CategoryDodge, CategoryParry, CategoryBlock}
	neverSuppressed := []Category{CategoryDeath, CategoryKick, CategoryTrip, CategoryBash,
		CategorySubmission, CategorySystem, CategoryDefault, CategoryCombatSummary}

	for _, c := range hitCats {
		if VerbosityFull.Suppresses(c) {
			t.Errorf("full must not suppress %v", c)
		}
		if VerbosityMedium.Suppresses(c) {
			t.Errorf("medium must not suppress hit category %v", c)
		}
		if !VerbosityLight.Suppresses(c) {
			t.Errorf("light must suppress hit category %v", c)
		}
	}
	for _, c := range defenseCats {
		if VerbosityFull.Suppresses(c) {
			t.Errorf("full must not suppress %v", c)
		}
		if !VerbosityMedium.Suppresses(c) {
			t.Errorf("medium must suppress defense category %v", c)
		}
		if !VerbosityLight.Suppresses(c) {
			t.Errorf("light must suppress defense category %v", c)
		}
	}
	for _, c := range neverSuppressed {
		for _, v := range []Verbosity{VerbosityFull, VerbosityMedium, VerbosityLight} {
			if v.Suppresses(c) {
				t.Errorf("%v must never suppress %v", v, c)
			}
		}
	}
}

func TestCategoryCombatSummaryString(t *testing.T) {
	if CategoryCombatSummary.String() != "combat-summary" {
		t.Errorf("got %q", CategoryCombatSummary.String())
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/messaging/ -run 'TestParseVerbosity|TestVerbosity|TestCategoryCombatSummary' -v`
Expected: FAIL (compile: Verbosity undefined, CategoryCombatSummary undefined)

- [ ] **Step 3: Implement**

Create `internal/messaging/verbosity.go`:

```go
package messaging

import "strings"

// Verbosity is a player's combat-text verbosity preference. Full is the
// engine's historical behavior; Medium shows landed hits only; Light
// suppresses individual lines in favor of a per-round compact tally
// (built by the combat hook — see internal/hooks/combat_verbosity.go).
// Spectated fights render one step lower than the viewer's setting.
type Verbosity int

const (
	VerbosityFull Verbosity = iota
	VerbosityMedium
	VerbosityLight
)

// ParseVerbosity maps a stored/user-typed string to a Verbosity.
// Unknown or empty input is Full — the safe, historical default.
func ParseVerbosity(s string) Verbosity {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "medium":
		return VerbosityMedium
	case "light":
		return VerbosityLight
	}
	return VerbosityFull
}

func (v Verbosity) String() string {
	switch v {
	case VerbosityMedium:
		return "medium"
	case VerbosityLight:
		return "light"
	}
	return "full"
}

// OneStepLower is the spectator tier: fights you're merely watching
// render one level quieter than your setting. Light is the floor.
func (v Verbosity) OneStepLower() Verbosity {
	switch v {
	case VerbosityFull:
		return VerbosityMedium
	default:
		return VerbosityLight
	}
}

// suppressibleAtMedium / suppressibleAtLight are explicit allowlists of
// categories the verbosity gate may drop. Anything not listed always
// passes — new combat text is verbose-by-default (safe).
var suppressibleAtMedium = map[Category]bool{
	CategoryDodge: true,
	CategoryParry: true,
	CategoryBlock: true,
}

var suppressibleAtLight = map[Category]bool{
	CategoryDodge:           true,
	CategoryParry:           true,
	CategoryBlock:           true,
	CategoryHitMelee:        true,
	CategoryHitBlunt:        true,
	CategoryHitNaturalSharp: true,
	CategoryHitRanged:       true,
	CategoryHitCaster:       true,
	CategoryHitUnarmed:      true,
}

// Suppresses reports whether this verbosity level drops lines of the
// given category. Floor rules (damage-to-viewer always shows) are the
// caller's responsibility — this is a pure category table.
func (v Verbosity) Suppresses(cat Category) bool {
	switch v {
	case VerbosityMedium:
		return suppressibleAtMedium[cat]
	case VerbosityLight:
		return suppressibleAtLight[cat]
	}
	return false
}
```

In `internal/messaging/messaging.go`, add to the Category enum in the "Combat — hits" region's vicinity — append after `CategoryWarcry`-style grouping is fine; place it with the combat block, e.g. directly after `CategoryDeath`:

```go
	CategoryCombatSummary // per-round compact tally (light verbosity)
```

(Exact insertion point: anywhere before `categoryMax`; keep it adjacent to the other combat categories for readability.) Add to `String()`:

```go
	case CategoryCombatSummary:
		return "combat-summary"
```

- [ ] **Step 4: Color alias + normalize skip**

In `_datafiles/world/dogmud/ansi-aliases.yaml`, find the category alias block (entries like `hit-melee: <n>`, `weather: 75`) and add a `combat-summary:` entry using a color consistent with the combat palette — read the existing `hit-melee` value and pick a nearby neutral (e.g. the same value as `hit-melee`, or `250` light gray if hits are strongly colored; judgment call, note your choice).

In `internal/messaging/normalize.go`, read the prose-normalization skip set (~line 28). If `CategoryHitMelee` (or the combat categories generally) are members, add `CategoryCombatSummary` alongside them; if they are not members, do nothing.

- [ ] **Step 5: Run tests to verify pass**

Run: `go test ./internal/messaging/ -count=1`
Expected: PASS (including pre-existing tests — `TestShouldWrapDisabledByDefault` iterates the full enum and must still pass; `categoryMax` handles the new constant automatically)

- [ ] **Step 6: Commit**

```bash
git add internal/messaging/ _datafiles/world/dogmud/ansi-aliases.yaml
git commit -m "feat(messaging): combat verbosity primitives + CategoryCombatSummary"
```

---

### Task 2: UserRecord.CombatVerbosity + getter

**Files:**
- Modify: `internal/users/userrecord.go` (field beside LineWidth at ~line 53; getter beside GetLineWidth at ~line 443)
- Test: `internal/users/users_test.go` (or a new `userrecord_verbosity_test.go` if users_test.go is unwieldy — check its size first)

- [ ] **Step 1: Write the failing test**

```go
func TestUserRecord_GetCombatVerbosity(t *testing.T) {
	u := &UserRecord{}
	if got := u.GetCombatVerbosity(); got != messaging.VerbosityFull {
		t.Errorf("empty setting should default to full, got %v", got)
	}
	u.CombatVerbosity = "light"
	if got := u.GetCombatVerbosity(); got != messaging.VerbosityLight {
		t.Errorf("light setting: got %v", got)
	}
	u.CombatVerbosity = "Medium"
	if got := u.GetCombatVerbosity(); got != messaging.VerbosityMedium {
		t.Errorf("case-insensitive medium: got %v", got)
	}
	var nilUser *UserRecord
	if got := nilUser.GetCombatVerbosity(); got != messaging.VerbosityFull {
		t.Errorf("nil receiver should default to full, got %v", got)
	}
}
```

(Add the `messaging` import if the test file lacks it.)

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/users/ -run TestUserRecord_GetCombatVerbosity -v`
Expected: FAIL (compile: CombatVerbosity undefined)

- [ ] **Step 3: Implement**

In the `UserRecord` struct, directly after the `LineWidth` field:

```go
	CombatVerbosity string                `yaml:"combatverbosity,omitempty"` // Combat text level: ""/full, medium (hits only), light (round tally)
```

After `GetLineWidth()`:

```go
// GetCombatVerbosity returns the user's combat-text verbosity, defaulting
// to full for unset/unknown values and nil receivers. The combat round
// hook reads this when draining attack narration.
func (u *UserRecord) GetCombatVerbosity() messaging.Verbosity {
	if u == nil {
		return messaging.VerbosityFull
	}
	return messaging.ParseVerbosity(u.CombatVerbosity)
}
```

- [ ] **Step 4: Run to verify pass**

Run: `go test ./internal/users/ -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/users/
git commit -m "feat(users): CombatVerbosity setting on UserRecord"
```

---

### Task 3: `set combatverbosity` command + helpfile

**Files:**
- Modify: `internal/usercommands/set.go` (switch at line ~57; status display in `displaySetStatus`; new helper near `cmdSetLineWidth` at ~line 480)
- Create: `_datafiles/world/dogmud/templates/help/combatverbosity.template`
- Modify: `_datafiles/world/dogmud/keywords.yaml`

- [ ] **Step 1: Add the switch case**

In `Set()`'s switch, after `case `linewidth``:

```go
		case `combatverbosity`:
			return cmdSetCombatVerbosity(user, args)
```

- [ ] **Step 2: Add the helper** (place next to `cmdSetLineWidth`):

```go
func cmdSetCombatVerbosity(user *users.UserRecord, args []string) (bool, error) {
	if len(args) < 1 {
		user.SendText(messaging.CategorySystem,
			fmt.Sprintf("Combat verbosity is currently <ansi fg=\"yellow-bold\">%s</ansi>.", user.GetCombatVerbosity().String()))
		user.SendText(messaging.CategorySystem, `Options: <ansi fg="command">full</ansi> (everything), <ansi fg="command">medium</ansi> (hits only), <ansi fg="command">light</ansi> (round summaries). See <ansi fg="command">help combatverbosity</ansi>.`)
		return true, nil
	}
	choice := strings.ToLower(args[0])
	if choice != `full` && choice != `medium` && choice != `light` {
		user.SendText(messaging.CategorySystem, `Combat verbosity must be <ansi fg="command">full</ansi>, <ansi fg="command">medium</ansi>, or <ansi fg="command">light</ansi>.`)
		return true, nil
	}
	if choice == `full` {
		user.CombatVerbosity = `` // empty = default = full (omitempty keeps saves clean)
	} else {
		user.CombatVerbosity = choice
	}
	user.SendText(messaging.CategorySystem,
		fmt.Sprintf("Combat verbosity set to <ansi fg=\"yellow-bold\">%s</ansi>.", choice))

	events.AddToQueue(events.UserSettingChanged{
		UserId: user.UserId,
		Name:   `combatverbosity`,
	})

	return true, nil
}
```

- [ ] **Step 3: Show it in `set` status**

In `displaySetStatus` (set.go:69), after the linewidth/wimpy entries (match the surrounding style — read the function's tail first):

```go
	user.SendText(messaging.CategorySystem, `<ansi fg="yellow-bold">combatverbosity:</ansi> `)
	user.SendText(messaging.CategorySystem, user.GetCombatVerbosity().String())
	user.SendText(messaging.CategorySystem, ``)
```

- [ ] **Step 4: Helpfile + keywords**

Create `_datafiles/world/dogmud/templates/help/combatverbosity.template` (mirror `weather.template`'s header conventions exactly; ≤80-char lines):

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">set combatverbosity</ansi>

Controls how much combat text you see during fights.

<ansi fg="yellow">Usage: </ansi>

  <ansi fg="command">set combatverbosity full</ansi>   - Every swing, hit, and dodge. (default)
  <ansi fg="command">set combatverbosity medium</ansi> - Landed hits only; misses and dodges
                                 are skipped.
  <ansi fg="command">set combatverbosity light</ansi>  - One compact summary line per round.

No matter the setting, you always see deaths, moves that change your
footing (knockdowns, grapples, stuns, disarms), and every blow that
lands on you personally.

Fights you are merely watching are shown one step quieter than your
setting, so a crowded room stays readable.

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help set</ansi>
```

In `_datafiles/world/dogmud/keywords.yaml`, add `- combatverbosity` to the same `commands` list that holds `weather` (information category) **or** a more fitting category if one exists for settings (read the file; if there's a `configuration`/`settings` group with `set` in it, use that). Keep alphabetical order within the list.

- [ ] **Step 5: Build + manual sanity**

Run: `go build ./...` — clean. `go vet ./internal/usercommands/` — clean.
(Command behavior is exercised in the Task 6 live smoke; set.go has no existing unit-test scaffolding for these helpers — verify by reading `internal/usercommands/usercommands_test.go` and add a unit test only if a pattern for `cmdSet*` helpers exists.)

- [ ] **Step 6: Commit**

```bash
git add internal/usercommands/set.go _datafiles/world/dogmud/templates/help/combatverbosity.template _datafiles/world/dogmud/keywords.yaml
git commit -m "feat(usercommands): set combatverbosity command + helpfile"
```

---

### Task 4: Round tally aggregator (hooks package, pure logic)

**Files:**
- Create: `internal/hooks/combat_verbosity.go`
- Test: `internal/hooks/combat_verbosity_test.go`

This task builds the aggregator as pure, fully-tested logic. Task 5 wires it.

- [ ] **Step 1: Write the failing tests**

Create `internal/hooks/combat_verbosity_test.go`:

```go
package hooks

import (
	"strings"
	"testing"
)

func mkSwings(hits, misses int, worstDamage int) []swingStat {
	out := []swingStat{}
	for i := 0; i < hits; i++ {
		d := 1
		if i == 0 {
			d = worstDamage
		}
		out = append(out, swingStat{Hit: true, Damage: d})
	}
	for i := 0; i < misses; i++ {
		out = append(out, swingStat{Hit: false})
	}
	return out
}

func TestTally_ParticipantOutgoingHitsAndEnemyWhiffs(t *testing.T) {
	agg := newCombatTallies()
	// Viewer (user 7) attacked the wolf: 2 hits (worst 30), 1 miss. Wolf maxHP 100.
	agg.record(7, fighterRef{Key: "u:7", Name: "You-Ignored", IsMob: false},
		fighterRef{Key: "m:31", Name: "marsh wolf", IsMob: true},
		mkSwings(2, 1, 30), 100)
	// Wolf attacked viewer: 0 hits, 2 misses. Viewer maxHP 200.
	agg.record(7, fighterRef{Key: "m:31", Name: "marsh wolf", IsMob: true},
		fighterRef{Key: "u:7", Name: "You-Ignored", IsMob: false},
		mkSwings(0, 2, 0), 200)

	lines := agg.flushForViewer(7, "u:7")
	if len(lines) != 1 {
		t.Fatalf("expected 1 tally line, got %d: %v", len(lines), lines)
	}
	l := lines[0]
	if !strings.Contains(l, "You strike") || !strings.Contains(l, "marsh wolf") || !strings.Contains(l, "twice") {
		t.Errorf("outgoing segment wrong: %q", l)
	}
	// 30/100 = 30%% → "serious wounds" per GetDamageDescription thresholds
	if !strings.Contains(l, "serious wounds") {
		t.Errorf("expected worst-hit tier 'serious wounds': %q", l)
	}
	if !strings.Contains(l, "fails to land a blow") {
		t.Errorf("expected enemy whiff segment: %q", l)
	}
}

func TestTally_ParticipantIncomingHitsOmitted(t *testing.T) {
	agg := newCombatTallies()
	agg.record(7, fighterRef{Key: "u:7", Name: "x", IsMob: false},
		fighterRef{Key: "m:31", Name: "marsh wolf", IsMob: true},
		mkSwings(1, 0, 10), 100)
	// Wolf LANDED hits on the viewer — those showed in full prose (floor
	// rule), so the tally must NOT re-describe them.
	agg.record(7, fighterRef{Key: "m:31", Name: "marsh wolf", IsMob: true},
		fighterRef{Key: "u:7", Name: "x", IsMob: false},
		mkSwings(2, 0, 40), 200)

	lines := agg.flushForViewer(7, "u:7")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %v", lines)
	}
	if strings.Contains(lines[0], "fails to land") {
		t.Errorf("enemy landed hits; whiff text is wrong: %q", lines[0])
	}
	// Incoming hits already shown in full — no tier for them in the tally.
	if strings.Count(lines[0], "wounds")+strings.Count(lines[0], "injuries")+strings.Count(lines[0], "damage") > 1 {
		t.Errorf("incoming damage should not be re-described: %q", lines[0])
	}
}

func TestTally_WhiffRound(t *testing.T) {
	agg := newCombatTallies()
	agg.record(7, fighterRef{Key: "u:7", Name: "x", IsMob: false},
		fighterRef{Key: "m:31", Name: "marsh wolf", IsMob: true},
		mkSwings(0, 2, 0), 100)
	agg.record(7, fighterRef{Key: "m:31", Name: "marsh wolf", IsMob: true},
		fighterRef{Key: "u:7", Name: "x", IsMob: false},
		mkSwings(0, 1, 0), 200)

	lines := agg.flushForViewer(7, "u:7")
	if len(lines) != 1 || !strings.Contains(lines[0], "neither side draws blood") {
		t.Errorf("whiff round wording: %v", lines)
	}
}

func TestTally_SpectatorBothDirections(t *testing.T) {
	agg := newCombatTallies()
	// Viewer 9 watches Velk (user 4) fight the shambler.
	agg.record(9, fighterRef{Key: "u:4", Name: "Velk", IsMob: false},
		fighterRef{Key: "m:50", Name: "bog shambler", IsMob: true},
		mkSwings(3, 0, 35), 100)
	agg.record(9, fighterRef{Key: "m:50", Name: "bog shambler", IsMob: true},
		fighterRef{Key: "u:4", Name: "Velk", IsMob: false},
		mkSwings(1, 1, 12), 150)

	lines := agg.flushForViewer(9, "")
	if len(lines) != 1 {
		t.Fatalf("expected 1 spectator line, got %v", lines)
	}
	l := lines[0]
	if !strings.Contains(l, "Velk") || !strings.Contains(l, "bog shambler") {
		t.Errorf("names missing: %q", l)
	}
	if !strings.Contains(l, "three times") {
		t.Errorf("expected count word 'three times': %q", l)
	}
	// 35/100 → serious wounds; 12/150 = 8%% → light wounds
	if !strings.Contains(l, "serious wounds") || !strings.Contains(l, "light wounds") {
		t.Errorf("expected both direction tiers: %q", l)
	}
}

func TestTally_MultipleFightsSortedAndCleared(t *testing.T) {
	agg := newCombatTallies()
	agg.record(9, fighterRef{Key: "u:4", Name: "Velk", IsMob: false},
		fighterRef{Key: "m:50", Name: "bog shambler", IsMob: true}, mkSwings(1, 0, 10), 100)
	agg.record(9, fighterRef{Key: "u:5", Name: "Tova", IsMob: false},
		fighterRef{Key: "m:51", Name: "reed viper", IsMob: true}, mkSwings(1, 0, 10), 100)

	lines := agg.flushForViewer(9, "")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %v", lines)
	}
	// Deterministic order (sorted by pair key).
	again := newCombatTallies()
	again.record(9, fighterRef{Key: "u:5", Name: "Tova", IsMob: false},
		fighterRef{Key: "m:51", Name: "reed viper", IsMob: true}, mkSwings(1, 0, 10), 100)
	again.record(9, fighterRef{Key: "u:4", Name: "Velk", IsMob: false},
		fighterRef{Key: "m:50", Name: "bog shambler", IsMob: true}, mkSwings(1, 0, 10), 100)
	lines2 := again.flushForViewer(9, "")
	if lines[0] != lines2[0] || lines[1] != lines2[1] {
		t.Errorf("flush order must be deterministic:\n%v\n%v", lines, lines2)
	}

	// flush clears
	if rem := agg.flushForViewer(9, ""); len(rem) != 0 {
		t.Errorf("second flush must be empty, got %v", rem)
	}
}

func TestCountWord(t *testing.T) {
	cases := map[int]string{1: "", 2: " twice", 3: " three times", 4: " again and again", 7: " again and again"}
	for n, want := range cases {
		if got := countWord(n); got != want {
			t.Errorf("countWord(%d) = %q want %q", n, got, want)
		}
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/hooks/ -run 'TestTally|TestCountWord' -v`
Expected: FAIL (compile: newCombatTallies undefined etc.)

- [ ] **Step 3: Implement**

Create `internal/hooks/combat_verbosity.go`:

```go
package hooks

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/combat"
)

// File: combat_verbosity.go
//
// Light-verbosity round tally for combat narration (spec:
// docs/superpowers/specs/2026-06-10-combat-verbosity-design.md).
// When a viewer's effective verbosity is Light, the per-swing combat
// lines are suppressed at the drain (dispatchCritAndMessaging) and the
// AttackResult's swing data is recorded here instead. flushCombatTallies
// (called at the end of DoCombat) emits one compact line per fight pair
// per viewer. All state is touched only from the game-loop goroutine.

// fighterRef identifies one combatant for tally purposes. Key is a
// stable identity ("u:<userId>" / "m:<mobInstanceId>") so same-named
// mobs don't merge; Name/IsMob drive rendering.
type fighterRef struct {
	Key   string
	Name  string
	IsMob bool
}

// swingStat is the slice of SwingEvent the tally needs.
type swingStat struct {
	Hit    bool
	Damage int
}

// tallyDir accumulates one attack direction within a fight pair.
type tallyDir struct {
	Hits        int
	Misses      int
	WorstHit    int
	TargetMaxHP int
}

func (d *tallyDir) add(swings []swingStat, targetMaxHP int) {
	for _, s := range swings {
		if s.Hit {
			d.Hits++
			if s.Damage > d.WorstHit {
				d.WorstHit = s.Damage
			}
		} else {
			d.Misses++
		}
	}
	if targetMaxHP > 0 {
		d.TargetMaxHP = targetMaxHP
	}
}

// combatTally is one (viewer, fight-pair) accumulator. A/B orientation
// is fixed by whichever direction is recorded first.
type combatTally struct {
	A, B   fighterRef
	AtoB   tallyDir
	BtoA   tallyDir
}

type tallyKey struct {
	viewerId int
	pairKey  string // canonical unordered pair: min(key)+"|"+max(key)
}

type combatTallies struct {
	m map[tallyKey]*combatTally
}

func newCombatTallies() *combatTallies {
	return &combatTallies{m: map[tallyKey]*combatTally{}}
}

func pairKeyFor(a, b string) string {
	if a < b {
		return a + "|" + b
	}
	return b + "|" + a
}

// record adds one AttackResult's swings (attacker → defender) to the
// viewer's tally for that fight pair.
func (ct *combatTallies) record(viewerId int, attacker, defender fighterRef, swings []swingStat, defenderMaxHP int) {
	k := tallyKey{viewerId: viewerId, pairKey: pairKeyFor(attacker.Key, defender.Key)}
	t, ok := ct.m[k]
	if !ok {
		t = &combatTally{A: attacker, B: defender}
		ct.m[k] = t
	}
	if attacker.Key == t.A.Key {
		t.AtoB.add(swings, defenderMaxHP)
	} else {
		t.BtoA.add(swings, defenderMaxHP)
	}
}

// countWord renders a hit count as prose. 1 → "" (the verb carries it),
// per the no-hard-numbers rule everything stays qualitative.
func countWord(n int) string {
	switch {
	case n <= 1:
		return ""
	case n == 2:
		return " twice"
	case n == 3:
		return " three times"
	default:
		return " again and again"
	}
}

// nameToken renders a fighter's name with the engine's standard color
// alias for their kind.
func nameToken(f fighterRef) string {
	if f.IsMob {
		return `<ansi fg="mobname">` + f.Name + `</ansi>`
	}
	return `<ansi fg="username">` + f.Name + `</ansi>`
}

// pronounFor is the subject stand-in for a fighter on second mention.
func pronounFor(f fighterRef) string {
	if f.IsMob {
		return "it"
	}
	return "they"
}

// renderTally builds the tally line for one fight pair from a viewer's
// perspective. viewerKey is "" for spectators, or the viewer's
// fighterRef.Key when they are a participant (their side renders as
// "You" and their incoming hits are omitted — full prose already showed
// them under the floor rule).
func renderTally(t *combatTally, viewerKey string) string {
	// Orient so X = viewer (participant) or t.A (spectator).
	x, y := t.A, t.B
	xOut, yOut := t.AtoB, t.BtoA
	if viewerKey != "" && t.B.Key == viewerKey {
		x, y = t.B, t.A
		xOut, yOut = t.BtoA, t.AtoB
	}
	isParticipant := viewerKey != "" && x.Key == viewerKey

	xSwings := xOut.Hits + xOut.Misses
	ySwings := yOut.Hits + yOut.Misses

	// Whiff round: swings happened, nothing landed either way.
	if xOut.Hits == 0 && yOut.Hits == 0 && (xSwings > 0 || ySwings > 0) {
		if isParticipant {
			return fmt.Sprintf("You trade swings with %s; neither side draws blood.", nameToken(y))
		}
		return fmt.Sprintf("%s and %s trade swings without drawing blood.", nameToken(x), nameToken(y))
	}

	segs := []string{}

	// X's outgoing segment.
	if xOut.Hits > 0 {
		tier := combat.GetDamageDescription(xOut.WorstHit, xOut.TargetMaxHP)
		if isParticipant {
			segs = append(segs, fmt.Sprintf("You strike %s%s (%s)", nameToken(y), countWord(xOut.Hits), tier))
		} else {
			segs = append(segs, fmt.Sprintf("%s strikes %s%s (%s)", nameToken(x), nameToken(y), countWord(xOut.Hits), tier))
		}
	} else if xSwings > 0 {
		if isParticipant {
			segs = append(segs, fmt.Sprintf("You fail to break %s's guard", nameToken(y)))
		} else {
			segs = append(segs, fmt.Sprintf("%s can't get past %s's guard", nameToken(x), nameToken(y)))
		}
	}

	// Y's segment. For participants, landed incoming hits already showed
	// in full prose (floor rule) — only whiffs are worth a mention.
	if yOut.Hits > 0 {
		if !isParticipant {
			tier := combat.GetDamageDescription(yOut.WorstHit, yOut.TargetMaxHP)
			segs = append(segs, fmt.Sprintf("%s lands %s%s (%s)",
				nameToken(y), hitNoun(yOut.Hits), countWord(yOut.Hits), tier))
		}
	} else if ySwings > 0 {
		segs = append(segs, fmt.Sprintf("%s fails to land a blow", pronounOrName(y, isParticipant)))
	}

	if len(segs) == 0 {
		return ""
	}
	return strings.Join(segs, "; ") + "."
}

// hitNoun: "a blow" vs "blows".
func hitNoun(n int) string {
	if n == 1 {
		return "a blow"
	}
	return "blows"
}

// pronounOrName: participants get the pronoun for flow ("it fails to
// land a blow"); spectators get the name so multi-fight rooms stay
// unambiguous.
func pronounOrName(f fighterRef, isParticipant bool) string {
	if isParticipant {
		return pronounFor(f)
	}
	return nameToken(f)
}

// flushForViewer renders and removes all of one viewer's tallies,
// sorted by pair key for deterministic output.
func (ct *combatTallies) flushForViewer(viewerId int, viewerKey string) []string {
	keys := []tallyKey{}
	for k := range ct.m {
		if k.viewerId == viewerId {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].pairKey < keys[j].pairKey })

	lines := []string{}
	for _, k := range keys {
		if line := renderTally(ct.m[k], viewerKey); line != "" {
			lines = append(lines, line)
		}
		delete(ct.m, k)
	}
	return lines
}

// viewerIds returns the distinct viewers with pending tallies.
func (ct *combatTallies) viewerIds() []int {
	seen := map[int]bool{}
	out := []int{}
	for k := range ct.m {
		if !seen[k.viewerId] {
			seen[k.viewerId] = true
			out = append(out, k.viewerId)
		}
	}
	sort.Ints(out)
	return out
}
```

Note the rendering nuance the tests pin down: the participant's whiff-only enemy segment uses a pronoun ("it fails to land a blow"), spectator lines always use names, and a participant's incoming LANDED hits are omitted entirely.

- [ ] **Step 4: Run to verify pass**

Run: `go test ./internal/hooks/ -run 'TestTally|TestCountWord' -v -count=1`
Expected: PASS. Then `go test ./internal/hooks/ -count=1` (full package — slower; confirm no regressions) and `go vet ./internal/hooks/`.

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/combat_verbosity.go internal/hooks/combat_verbosity_test.go
git commit -m "feat(hooks): combat round tally aggregator for light verbosity"
```

---

### Task 5: Wire the gate into the combat drain + flush

**Files:**
- Modify: `internal/hooks/combat_verbosity.go` (add the drain-side glue: package-level aggregator, record-from-AttackResult, flush)
- Modify: `internal/hooks/NewRound_DoCombat_unified.go` (`dispatchCritAndMessaging`, lines ~506-536)
- Modify: `internal/hooks/NewRound_DoCombat.go` (`DoCombat`, add flush before return)
- Test: `internal/hooks/combat_verbosity_wiring_test.go`

- [ ] **Step 1: Add the glue functions**

Append to `internal/hooks/combat_verbosity.go`:

```go
// ── Drain-side glue ────────────────────────────────────────────────────

// roundTallies is the per-round accumulator. Game-loop goroutine only.
var roundTallies = newCombatTallies()

// fighterRefFor builds a tally identity for an Actor.
func fighterRefFor(a actions.Actor) fighterRef {
	if a.IsPlayer() {
		return fighterRef{Key: fmt.Sprintf("u:%d", a.GetUserId()), Name: a.GetCharacter().Name, IsMob: false}
	}
	return fighterRef{Key: fmt.Sprintf("m:%d", a.GetMobInstanceId()), Name: a.GetCharacter().Name, IsMob: true}
}

// swingStatsFor extracts tally stats from an AttackResult. Rounds with
// no per-swing analytics (defensive fallback) degrade to one synthetic
// swing from the top-level Hit/DamageToTarget.
func swingStatsFor(res *combat.AttackResult) []swingStat {
	if len(res.SwingEvents) > 0 {
		out := make([]swingStat, 0, len(res.SwingEvents))
		for _, s := range res.SwingEvents {
			out = append(out, swingStat{Hit: s.Hit, Damage: s.Damage})
		}
		return out
	}
	if res.DefenderWasAttacked || res.Hit {
		return []swingStat{{Hit: res.Hit, Damage: res.DamageToTarget}}
	}
	return nil
}

// recordTallyFor records one AttackResult into a viewer's round tally.
func recordTallyFor(viewerId int, atk, def actions.Actor, res *combat.AttackResult) {
	swings := swingStatsFor(res)
	if len(swings) == 0 {
		return
	}
	roundTallies.record(viewerId, fighterRefFor(atk), fighterRefFor(def), swings,
		def.GetCharacter().HealthMax.Value)
}

// drainParticipantLines sends a participant's combat lines subject to
// their verbosity. incoming=true marks lines describing swings AGAINST
// the viewer: landed hits there are floor-protected (always full prose,
// any level); only defense/miss lines are suppressible.
func drainParticipantLines(u *users.UserRecord, msgs []combat.TaggedMessage, lvl messaging.Verbosity, incoming bool) {
	for _, msg := range msgs {
		if incoming && isHitCategory(msg.Category) {
			u.SendText(msg.Category, msg.Text) // floor: damage to you always shows
			continue
		}
		if lvl.Suppresses(msg.Category) {
			continue
		}
		u.SendText(msg.Category, msg.Text)
	}
}

// isHitCategory reports whether a category is one of the CategoryHit*
// damage bands.
func isHitCategory(cat messaging.Category) bool {
	switch cat {
	case messaging.CategoryHitMelee, messaging.CategoryHitBlunt, messaging.CategoryHitNaturalSharp,
		messaging.CategoryHitRanged, messaging.CategoryHitCaster, messaging.CategoryHitUnarmed:
		return true
	}
	return false
}

// drainSpectatorLines delivers room combat lines per spectator at their
// effective (one-step-lower) verbosity, preserving the sight gate via
// SendTextVisualToUser. Excluded ids are the combatants (they got their
// participant lines already).
func drainSpectatorLines(room *rooms.Room, msgs []combat.TaggedMessage, excludeUserIds []int) {
	if room == nil || len(msgs) == 0 {
		return
	}
	for _, uid := range room.GetPlayers() {
		if isExcludedUser(uid, excludeUserIds) {
			continue
		}
		u := users.GetByUserId(uid)
		if u == nil {
			continue
		}
		lvl := u.GetCombatVerbosity().OneStepLower()
		for _, msg := range msgs {
			if lvl.Suppresses(msg.Category) {
				continue
			}
			room.SendTextVisualToUser(u, msg.Category, msg.Text)
		}
	}
}

// recordSpectatorTallies records this AttackResult for every spectator
// whose effective verbosity is Light. Called once per AttackResult
// (NOT per message batch).
func recordSpectatorTallies(atkRoom, defRoom *rooms.Room, atk, def actions.Actor, res *combat.AttackResult, excludeUserIds []int) {
	seen := map[int]bool{}
	for _, room := range []*rooms.Room{atkRoom, defRoom} {
		if room == nil {
			continue
		}
		for _, uid := range room.GetPlayers() {
			if seen[uid] || isExcludedUser(uid, excludeUserIds) {
				continue
			}
			seen[uid] = true
			u := users.GetByUserId(uid)
			if u == nil {
				continue
			}
			if u.GetCombatVerbosity().OneStepLower() == messaging.VerbosityLight {
				recordTallyFor(uid, atk, def, res)
			}
		}
	}
}

// flushCombatTallies emits every pending tally line and clears the
// accumulator. Called once at the end of DoCombat each round.
func flushCombatTallies() {
	for _, viewerId := range roundTallies.viewerIds() {
		u := users.GetByUserId(viewerId)
		if u == nil {
			// Viewer logged off mid-round; drop their tallies.
			roundTallies.flushForViewer(viewerId, "")
			continue
		}
		viewerKey := fmt.Sprintf("u:%d", viewerId)
		for _, line := range roundTallies.flushForViewer(viewerId, viewerKey) {
			u.SendText(messaging.CategoryCombatSummary, line)
		}
	}
}
```

Add the imports the file now needs (`actions`, `messaging`, `rooms`, `users` from the engine's internal packages — match the hooks package's existing import paths). NOTE on `flushForViewer(viewerId, viewerKey)`: passing the viewer's "u:<id>" key makes participant tallies render in second person while spectator tallies (pairs not containing the viewer) render with names — `renderTally` orients on whether the viewer's key matches a side. Re-check Task 4's tests still pass since they call `flushForViewer(7, "u:7")` / `flushForViewer(9, "")` — spectator test uses "" but production passes the real key; pairs that don't contain the key render identically either way (verify by reading renderTally's orientation logic — `isParticipant` is false when neither side matches).

- [ ] **Step 2: Rewire `dispatchCritAndMessaging`**

In `internal/hooks/NewRound_DoCombat_unified.go`, replace the drain block (currently lines ~506-536, beginning with the `// Direct messages — Divergence #1.` comment and ending with the `sendDarkRoomCombatFallback(defRoom, ...)` call) with:

```go
	// Direct messages — Divergence #1, now verbosity-gated (spec:
	// 2026-06-10-combat-verbosity-design.md). AttackResult.MessagesTo*
	// carry per-line TaggedMessage data (Category + Text); the gate
	// suppresses by category per the viewer's setting. Floor rules:
	// damage-to-viewer lines always pass; categories outside the
	// suppress tables always pass.
	if atk.IsPlayer() {
		u := asUser(atk)
		lvl := u.GetCombatVerbosity()
		drainParticipantLines(u, res.MessagesToSource, lvl, false)
		if lvl == messaging.VerbosityLight {
			recordTallyFor(u.UserId, atk, def, res)
		}
	}
	if def.IsPlayer() {
		u := asUser(def)
		lvl := u.GetCombatVerbosity()
		drainParticipantLines(u, res.MessagesToTarget, lvl, true)
		if lvl == messaging.VerbosityLight {
			recordTallyFor(u.UserId, atk, def, res)
		}
	}

	// Room broadcasts, per-spectator gated one step below their setting.
	excludes := playerExcludeIds(atk, def)
	drainSpectatorLines(atkRoom, res.MessagesToSourceRoom, excludes)
	drainSpectatorLines(defRoom, res.MessagesToTargetRoom, excludes)
	recordSpectatorTallies(atkRoom, defRoom, atk, def, res, excludes)
	sendDarkRoomCombatFallback(atkRoom, excludes...)
	if defRoom != atkRoom {
		sendDarkRoomCombatFallback(defRoom, excludes...)
	}
```

The old `sendVisualRoomText` helper keeps its other callers (wait rounds etc.) — do not delete it.

- [ ] **Step 3: Flush in DoCombat**

In `internal/hooks/NewRound_DoCombat.go`, at the very end of `DoCombat` (immediately before its final `return`):

```go
	// Light-verbosity round tallies (spec: combat-verbosity design).
	flushCombatTallies()
```

Read the end of the function first — place the call after the retarget pass so the tally is the last combat text of the round.

- [ ] **Step 4: Write the wiring test**

Create `internal/hooks/combat_verbosity_wiring_test.go`. Model setup on `NewRound_DoCombat_routing_test.go` (read it first — it shows how the package fakes users/rooms for drain testing). The behaviors to pin:

```go
// Behavior matrix to assert (adapt mechanics to the routing test's
// existing fake/capture style):
//
// 1. Full participant: every MessagesToSource line delivered (hits AND
//    dodges) — current behavior preserved.
// 2. Medium participant (attacker): CategoryHitMelee line delivered,
//    CategoryDodge line suppressed.
// 3. Medium participant (defender): CategoryHit* lines in
//    MessagesToTarget DELIVERED even though medium (floor: damage to
//    you); CategoryParry line suppressed.
// 4. Light participant (attacker): both hit and dodge lines suppressed;
//    after flushCombatTallies() the user receives exactly one
//    CategoryCombatSummary line mentioning the defender's name.
// 5. Spectator with full setting: effective medium — room hit lines
//    delivered via the visual path, room dodge lines suppressed.
// 6. Spectator with medium setting: effective light — all room lines
//    suppressed; after flush, one CategoryCombatSummary line.
// 7. Non-suppressible category (e.g. CategoryDefault) in MessagesToSource
//    passes at light.
```

If the routing test's scaffolding can't capture per-user sends for spectators (it may only cover participant drains), split: assert 1-4+7 at the `dispatchCritAndMessaging` level and 5-6 against `drainSpectatorLines`/`recordSpectatorTallies` with a room fixture from the rooms test helpers (`SeedRoomsForTest` exists per internal/rooms/test_helpers.go). Write real assertions — suppressed means the captured send list contains zero lines with that category/text.

- [ ] **Step 5: Run everything**

```bash
go build ./...
go test ./internal/hooks/ ./internal/messaging/ ./internal/users/ ./internal/usercommands/ -count=1
go vet ./internal/hooks/
```
Expected: clean build, all green. Existing routing/darkness tests in the hooks package must still pass — they exercise the drain; if any fail, the gate broke full-verbosity defaults (every user fixture without CombatVerbosity set must behave exactly as before — that invariant is the point).

- [ ] **Step 6: Commit**

```bash
git add internal/hooks/
git commit -m "feat(hooks): verbosity-gated combat drain + light-mode round tally flush"
```

---

### Task 6: context.md audit, smoke, PATCH_NOTES, merge

**Files:**
- Modify: `PATCH_NOTES.md`
- Audit/modify: `internal/messaging/context.md`, `internal/hooks/context.md`, `internal/users/context.md`, `internal/usercommands/context.md` (whichever exist)

- [ ] **Step 0: context.md sanity audit (touched packages)**

For each package this branch modified (`internal/messaging`, `internal/users`, `internal/usercommands`, `internal/hooks`), check whether a `context.md` exists and read the sections describing what we changed. Update surgically wherever the doc is now stale or silent on something load-bearing:

- `internal/messaging/context.md`: Category list/count if it enumerates categories (new `CategoryCombatSummary`); add a short note that verbosity primitives (`Verbosity`, `ParseVerbosity`, `Suppresses` allowlists) live in `verbosity.go` and that suppression is applied by the combat hooks, not the pipeline.
- `internal/hooks/context.md`: the combat-round messaging description (it documents `DoCombat` and the drain) — note the verbosity gate in `dispatchCritAndMessaging`, the `combat_verbosity.go` tally aggregator, and the `flushCombatTallies()` call at the end of `DoCombat`.
- `internal/users/context.md`: if it lists UserRecord settings fields (LineWidth etc.), add `CombatVerbosity`.
- `internal/usercommands/context.md`: if it enumerates `set` subcommands, add `combatverbosity`.

Keep edits minimal — correct what's wrong/missing, don't rewrite. Commit these with the PATCH_NOTES commit in Step 3.

- [ ] **Step 1: Boot smoke**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
(go run . > /tmp/verbosity-boot.log 2>&1 &)
sleep 40
grep -E "panic|PANIC|error" /tmp/verbosity-boot.log | head -10
```
Expected: no panics (keywords.yaml + helpfile load is exercised at boot). Kill the server after.

- [ ] **Step 2: Live smoke (scripted via AI port 55555, admin account smoketester/smoke123test; pace commands ~3s apart — AI rate limit is 2/round)**

1. `set combatverbosity` → reports "full". `help combatverbosity` renders.
2. Pick a fightable mob (Test Arena, room 200 north of room 1, is a combat training ground — or any aggro-able critter). At **full**: attack; confirm normal per-swing prose.
3. `set combatverbosity medium` → next rounds show landed-hit lines only; no dodge/parry/block lines; hits AGAINST the tester still show.
4. `set combatverbosity light` → per-swing lines stop; one `You strike ... (tier)`-style summary line arrives at round end; incoming landed hits still show in full as they happen.
5. Status-move floor: `kick` (or trip/bash) mid-fight at light — its message still shows.
6. Death floor: kill the mob at light — death message shows.
7. Spectator tiering: second session (e.g. local-fresh AI login) in the same room set to full — sees hits-only for the tester's fight; set to medium → sees only the summary line; set to light → summary line.
8. `set` (no args) shows the combatverbosity row. Reconnect — setting persisted in the user save.
9. Cleanup: reset tester to full, kill server, confirm no processes left, wipe instance saves.

Record verbatim evidence per check. Any failure: fix before merging.

- [ ] **Step 3: PATCH_NOTES entry**

Match the existing format; player-facing copy along the lines of: "New `set combatverbosity` option: `full` (everything, the default), `medium` (landed hits only), or `light` (one compact summary per round). Whatever you choose, you always see deaths, position-changing moves, and every blow that lands on you — and fights you're just watching run one step quieter than your own."

- [ ] **Step 4: Final verification + merge**

```bash
go build ./... && go test ./internal/hooks/ ./internal/messaging/ ./internal/users/ ./internal/usercommands/ -count=1
git checkout master
git merge --no-ff feature/combat-verbosity -m "Merge feature/combat-verbosity: combat text verbosity (full/medium/light, tiered for spectators)"
```
Then re-run the boot smoke once on master. No prod push (end-of-day bundle per SOP).

---

## Self-review notes

- **Spec coverage:** setting+storage+UX (T2+T3), per-line gate with allowlist semantics (T1 `Suppresses` tables + T5 drain), floor rules (incoming-hit bypass in `drainParticipantLines`; deaths/status moves never enter the gated drain — verified untouched paths listed in API facts), spectator one-step tier (T5 `OneStepLower` at both spectator sites), light tally from SwingEvents with worst-tier + pluralization + whiff phrasing + participant-incoming omission (T4), flush at round end (T5 step 3), phase-1 scope guard (only `dispatchCritAndMessaging`'s four drains touched; wait rounds, crit effects, dark fallback, RoomOld untouched), AI accounts default full (empty field ⇒ full), tests incl. hook-level matrix (T5 step 4) and live smoke incl. floor + spectator checks (T6).
- **Type consistency:** `fighterRef`/`swingStat`/`tallyDir`/`combatTally`/`combatTallies` defined T4, used T5; `Verbosity`/`Suppresses`/`OneStepLower`/`CategoryCombatSummary` defined T1, used T2/T5; `GetCombatVerbosity` defined T2, used T3/T5. `flushForViewer(viewerId, viewerKey)` signature consistent T4/T5.
- **Known judgment points left to the implementer:** exact `HealthMax.Value` field spelling (stat-pool shape verified via `StaminaMax.Value`); ansi-aliases color value for `combat-summary`; normalize.go skip-set membership mirroring; wiring-test scaffolding adaptation; keywords.yaml category choice.
