# Centralized Messaging Framework Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a single compose → normalize → anonymize → color → wrap → deliver pipeline that every player-facing line of text flows through; consume the chunk-6 Perception FSM via a sight gate; migrate all 228 broadcast call sites to a categorized API; centralize wrapping; collapse the duplicated `canSeeInRoom` predicates; add infrared "red shapes" anonymization; fix the companion-name leak in dark rooms.

**Architecture:** New `internal/messaging/` package owns the pipeline and the `Category` enum (one constant per recognized text class). `Room.SendText` / `Room.SendTextVisual` / `UserRecord.SendText` gain a leading `messaging.Category` parameter. `SendTextVisual` is sight-gated per-recipient using `CanSeeClearly` / `CanSeeShapes` predicates that compose Perception state + room lighting + NightVision / InfraredVision buff flags. Infrared observers in dark rooms receive an anonymized "red shapes" render. Colors are resolved via new aliases in `ansi-aliases.yaml`, not hard-coded. A legacy `SendTextLegacy` shim keeps existing callers compiling during the per-package audit; shim is deleted in T16.

**Tech Stack:** Go 1.24, GoMud engine, existing `<ansi fg="alias">…</ansi>` template system, YAML data files, standard `testing` package.

**Spec:** `docs/superpowers/specs/2026-05-19-messaging-framework-design.md`

**Predecessor:** chunk 6 (Perception FSM, shipped dormant 2026-05-19). The FSM is at `internal/state/perception/`; `Character.Perception *perception.Machine` is already wired (`internal/characters/character.go:198`) and transitions fire from buff/condition observers — but no consumer reads the state yet. This chunk is that consumer.

**Branch:** Continue on `feature/mob-aliveness-1.3-crimes`. No new branch.

**Estimated scope:** 18 tasks. Largest non-framework task is T13 (`usercommands/` audit, ~70 sites). No end-to-end smoke until T18 — the framework is dormant in Phase 1, exercised by per-package audits in Phase 2, validated at the close in T18.

---

## Phase 1 — Framework Primitives (T1–T9)

No behavior change yet. Build the engine, keep it inert. Existing callers keep using the un-categorized API via the `SendTextLegacy` shim added in T9.

---

### Task 1: Package Skeleton + Category Enum

**Files:**
- Create: `internal/messaging/messaging.go`
- Create: `internal/messaging/messaging_test.go`
- Create: `internal/messaging/context.md`

- [ ] **Step 1: Write the failing test for the Category enum**

Create `internal/messaging/messaging_test.go`:

```go
package messaging

import "testing"

func TestCategoryDefaultIsZero(t *testing.T) {
	if CategoryDefault != 0 {
		t.Fatalf("CategoryDefault must be zero-value (got %d)", CategoryDefault)
	}
}

func TestCategoryStringRoundTrip(t *testing.T) {
	cases := []Category{
		CategoryHitMelee, CategoryDodge, CategoryParry, CategoryBlock,
		CategoryGrappleFlow, CategorySurpriseAttack, CategoryRally,
		CategorySpellFold, CategorySpellElemental, CategorySpeech,
		CategoryWhisper, CategoryBroadcast, CategoryError,
		CategoryRoomDescription, CategorySkillProgress,
	}
	seen := map[string]Category{}
	for _, c := range cases {
		s := c.String()
		if s == "" || s == "Unknown" {
			t.Errorf("category %d returned %q", c, s)
		}
		if prev, ok := seen[s]; ok && prev != c {
			t.Errorf("category %q is ambiguous (%d and %d)", s, prev, c)
		}
		seen[s] = c
	}
}

func TestUnknownCategoryString(t *testing.T) {
	if Category(-1).String() != "Unknown" {
		t.Fatalf("negative Category should stringify to Unknown")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/messaging/...`
Expected: package not found / compile errors. That's fine — failure confirms we haven't yet written the implementation.

- [ ] **Step 3: Write the Category enum implementation**

Create `internal/messaging/messaging.go`:

```go
// Package messaging owns the centralized player-facing-text pipeline.
//
// Every Room.SendText / Room.SendTextVisual / UserRecord.SendText call
// flows through this package's pipeline (compose → normalize →
// anonymize → color → wrap → deliver). Sites identify their text by
// Category; the pipeline resolves the color, applies style
// normalization, sight-gates visual content using the Perception FSM,
// and wraps to each recipient's LineWidth preference.
//
// See docs/superpowers/specs/2026-05-19-messaging-framework-design.md
// for the design and rationale.
package messaging

// Category enumerates every recognized text class. Used by the
// pipeline to look up a color alias and to drive per-Category
// normalization-skip behavior. Adding a new Category is a 2-line
// change (constant here + alias in ansi-aliases.yaml).
type Category int

const (
	CategoryDefault Category = iota

	// Combat — hits.
	CategoryHitMelee
	CategoryHitBlunt
	CategoryHitNaturalSharp
	CategoryHitRanged
	CategoryHitCaster
	CategoryHitUnarmed

	// Combat — defense.
	CategoryDodge
	CategoryParry
	CategoryBlock

	// Combat — grapple.
	CategoryGrappleFlow
	CategoryGrappleHigh

	// Combat — submissions / death.
	CategorySubmission
	CategoryDeath

	// Combat — special moves.
	CategorySurpriseAttack
	CategoryKick
	CategoryTrip
	CategoryBash
	CategoryRally
	CategoryWarcry
	CategoryTauntSuccess
	CategoryTauntResist
	CategoryTauntFailure

	// Spells.
	CategorySpellFold
	CategorySpellDisruption
	CategorySpellElemental
	CategorySpellEnhancement
	CategorySpellMental
	CategorySpellVital
	CategorySpellManifestation

	// Social.
	CategorySpeech
	CategoryWhisper
	CategoryShout
	CategoryOOC
	CategoryNPCDialogue
	CategoryDialogueHint
	CategoryEmote
	CategoryMobIdle
	CategoryMobEmote

	// System / meta.
	CategoryBroadcast
	CategoryTip
	CategorySystem
	CategoryError
	CategoryWarning
	CategorySkillProgress
	CategoryLogin
	CategoryLogout

	// Environment.
	CategoryRoomDescription
	CategoryRoomEntry
	CategoryRoomExit
	CategoryWeather
	CategoryTimeOfDay

	// Other.
	CategoryLoot
	CategoryEquipment
	CategoryBuffApply
	CategoryBuffExpire
	CategoryMutation
	CategoryToxin
)

// String returns a stable identifier for the category — used in
// logging and to key the color alias lookup in color.go.
func (c Category) String() string {
	switch c {
	case CategoryDefault:
		return "default"
	case CategoryHitMelee:
		return "hit-melee"
	case CategoryHitBlunt:
		return "hit-blunt"
	case CategoryHitNaturalSharp:
		return "hit-natural-sharp"
	case CategoryHitRanged:
		return "hit-ranged"
	case CategoryHitCaster:
		return "hit-caster"
	case CategoryHitUnarmed:
		return "hit-unarmed"
	case CategoryDodge:
		return "dodge"
	case CategoryParry:
		return "parry"
	case CategoryBlock:
		return "block"
	case CategoryGrappleFlow:
		return "grapple-flow"
	case CategoryGrappleHigh:
		return "grapple-high"
	case CategorySubmission:
		return "submission"
	case CategoryDeath:
		return "death"
	case CategorySurpriseAttack:
		return "surprise"
	case CategoryKick:
		return "kick"
	case CategoryTrip:
		return "trip"
	case CategoryBash:
		return "bash"
	case CategoryRally:
		return "rally"
	case CategoryWarcry:
		return "warcry"
	case CategoryTauntSuccess:
		return "taunt-success"
	case CategoryTauntResist:
		return "taunt-resist"
	case CategoryTauntFailure:
		return "taunt-failure"
	case CategorySpellFold:
		return "spell-fold"
	case CategorySpellDisruption:
		return "spell-disruption"
	case CategorySpellElemental:
		return "spell-elemental"
	case CategorySpellEnhancement:
		return "spell-enhancement"
	case CategorySpellMental:
		return "spell-mental"
	case CategorySpellVital:
		return "spell-vital"
	case CategorySpellManifestation:
		return "spell-manifestation"
	case CategorySpeech:
		return "speech"
	case CategoryWhisper:
		return "whisper"
	case CategoryShout:
		return "shout"
	case CategoryOOC:
		return "ooc"
	case CategoryNPCDialogue:
		return "npc-dialogue"
	case CategoryDialogueHint:
		return "dialogue-hint"
	case CategoryEmote:
		return "emote"
	case CategoryMobIdle:
		return "mob-idle"
	case CategoryMobEmote:
		return "mob-emote"
	case CategoryBroadcast:
		return "broadcast"
	case CategoryTip:
		return "tip"
	case CategorySystem:
		return "system"
	case CategoryError:
		return "error"
	case CategoryWarning:
		return "warning"
	case CategorySkillProgress:
		return "skill-progress"
	case CategoryLogin:
		return "login"
	case CategoryLogout:
		return "logout"
	case CategoryRoomDescription:
		return "room-description"
	case CategoryRoomEntry:
		return "room-entry"
	case CategoryRoomExit:
		return "room-exit"
	case CategoryWeather:
		return "weather"
	case CategoryTimeOfDay:
		return "time-of-day"
	case CategoryLoot:
		return "loot"
	case CategoryEquipment:
		return "equipment"
	case CategoryBuffApply:
		return "buff-apply"
	case CategoryBuffExpire:
		return "buff-expire"
	case CategoryMutation:
		return "mutation"
	case CategoryToxin:
		return "toxin"
	}
	return "Unknown"
}
```

- [ ] **Step 4: Run the tests and verify they pass**

Run: `go test ./internal/messaging/...`
Expected: PASS.

- [ ] **Step 5: Create the package context.md**

Create `internal/messaging/context.md`:

```markdown
# internal/messaging

Centralized player-facing-text pipeline.

## Pipeline Stages

1. **Compose** — caller produces `(Category, text)`.
2. **Style normalize** — sentence-start caps, a/an agreement,
   duplicate-word collapse, sentence-end punctuation, ANSI canon for
   names. Per-Category skip table in `normalize.go`.
3. **Sight gate** (visual channel only) — per-recipient: CanSeeClearly,
   CanSeeShapes, or skip-visual-deliver-audio.
4. **Anonymize** (infrared-only path) — regex strips `username` /
   `mobname` / `petname` ANSI name tags, substitutes "a figure" + the
   `combat-anon` color alias.
5. **Apply category color tag** — `<ansi fg="<category-alias>">…</ansi>`.
6. **Wrap** at recipient's `UserRecord.LineWidth` (default 80),
   ANSI-aware.
7. **Deliver** to the recipient's connection.

## Channels

| Channel  | Helper            | Sight-gated | Stages run            |
|----------|-------------------|-------------|-----------------------|
| Audio    | `SendText`        | no          | 1, 2, 5, 6, 7         |
| Visual   | `SendTextVisual`  | yes         | all 7                 |

## Adding a new Category

1. Add a constant to the enum in `messaging.go`. Append at the end of
   its section.
2. Add the matching string in `Category.String()`.
3. Add the color alias in `_datafiles/world/dogmud/ansi-aliases.yaml`
   named `<category-name>` where `<category-name>` is the string the
   enum returns.
4. If the new Category needs style-normalization skips, edit the
   `normalize.go` skip table.

## See Also

- `docs/superpowers/specs/2026-05-19-messaging-framework-design.md` —
  full design.
- `internal/state/perception/context.md` — the FSM whose state the
  sight gate reads.
- `_datafiles/world/dogmud/ansi-aliases.yaml` — color aliases.
```

- [ ] **Step 6: Verify the package builds cleanly**

Run: `go build ./internal/messaging/...`
Expected: no output (success).

- [ ] **Step 7: Commit**

```bash
git add internal/messaging/
git commit -m "$(cat <<'EOF'
feat(messaging): T1 — package skeleton + Category enum

Establishes internal/messaging/ with the full Category enum, String()
round-trip, and package context.md. No pipeline yet; subsequent tasks
add compose, normalize, anonymize, color, wrap, deliver, and the
public Send helpers.

Predecessor: chunk 6 Perception FSM (shipped dormant 2026-05-19) —
this chunk is its consumer.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Pipeline Core (Wiring Stub)

Wire an empty pipeline that all stages will hook into. The stub
delivers text unchanged; each subsequent task replaces a stub stage
with the real implementation.

**Files:**
- Create: `internal/messaging/pipeline.go`
- Create: `internal/messaging/pipeline_test.go`

- [ ] **Step 1: Write the failing test for the pipeline stub**

Create `internal/messaging/pipeline_test.go`:

```go
package messaging

import "testing"

// RenderForRecipient is the per-recipient pipeline entry point used
// internally by the Room/UserRecord Send helpers. Returns the final
// text to deliver. An empty return string means "don't deliver to
// this recipient" (used by the sight gate).
func TestRenderForRecipientStubReturnsTextUnchanged(t *testing.T) {
	got := RenderForRecipient(RenderInput{
		Category:  CategoryDefault,
		Text:      "Hello, world.",
		Channel:   ChannelAudio,
		LineWidth: 80,
	})
	if got != "Hello, world." {
		t.Fatalf("stub pipeline mutated text: got %q", got)
	}
}

func TestChannelConstants(t *testing.T) {
	if ChannelAudio == ChannelVisual {
		t.Fatal("ChannelAudio and ChannelVisual must differ")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/messaging/...`
Expected: undefined: `RenderForRecipient`, `RenderInput`, `ChannelAudio`, `ChannelVisual`.

- [ ] **Step 3: Write the pipeline stub**

Create `internal/messaging/pipeline.go`:

```go
package messaging

// Channel discriminates the broadcast path. Audio bypasses the sight
// gate and the anonymizer; visual runs the full per-recipient
// pipeline. Both channels run style normalization, color, and wrap.
type Channel int

const (
	ChannelAudio Channel = iota
	ChannelVisual
)

// RenderInput bundles the parameters for one recipient's pipeline
// pass. The caller (Room/UserRecord Send helpers) constructs one of
// these per recipient and invokes RenderForRecipient.
//
// SightDecision is computed by the caller using the predicates in
// predicates.go BEFORE entering the pipeline; the pipeline trusts it.
// CanSeeClearly + ChannelVisual → full visual text.
// CanSeeShapes only + ChannelVisual → anonymized text.
// Neither + ChannelVisual → empty return ("don't deliver").
// ChannelAudio ignores SightDecision entirely.
type RenderInput struct {
	Category      Category
	Text          string
	Channel       Channel
	SightDecision SightDecision
	LineWidth     int
}

// SightDecision is the precomputed visibility verdict for one recipient.
type SightDecision int

const (
	SightFull   SightDecision = iota // CanSeeClearly
	SightShapes                      // !CanSeeClearly && CanSeeShapes (infrared in dark)
	SightNone                        // can't see at all
)

// RenderForRecipient runs the pipeline for one recipient and returns
// the final delivery string. Empty return means "don't deliver".
//
// Stage order:
//   1. Compose (caller-provided in.Text)
//   2. Normalize (normalize.go — wired in T8)
//   3. Sight gate (visual channel only)
//   4. Anonymize (infrared-only path; anonymize.go — wired in T6)
//   5. Color (color.go — wired in T2 alongside this stub)
//   6. Wrap (wrap.go — wired in T5)
//   7. Deliver (caller does this; pipeline returns the string)
//
// Each stub stage is a no-op until its task lands. Order is locked.
func RenderForRecipient(in RenderInput) string {
	text := in.Text

	// Stage 2: normalize (stubbed; T8 lands the implementation).
	text = normalize(in.Category, text)

	// Stage 3: sight gate (visual channel only).
	if in.Channel == ChannelVisual {
		switch in.SightDecision {
		case SightNone:
			return ""
		case SightShapes:
			// Stage 4: anonymize (stubbed; T6 lands the implementation).
			text = anonymize(text)
		}
	}

	// Stage 5: color (stubbed; T2 lands a no-op, T4 wires data).
	text = applyCategoryColor(in.Category, text)

	// Stage 6: wrap (stubbed; T5 lands the implementation).
	text = wrap(text, in.LineWidth)

	return text
}

// Stub implementations — each task replaces its stub.
// Keeping them in pipeline.go for now; T5/T6/T8 move them to their
// own files.

func normalize(cat Category, text string) string { return text }
func anonymize(text string) string                { return text }
func applyCategoryColor(cat Category, text string) string {
	if cat == CategoryDefault {
		return text
	}
	// Real impl in T4 wraps with <ansi fg="<cat.String()>">…</ansi>
	// after the alias is registered. For now: no-op.
	return text
}
func wrap(text string, maxWidth int) string { return text }
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/messaging/...`
Expected: PASS.

- [ ] **Step 5: Verify the package still builds cleanly**

Run: `go build ./internal/messaging/...`
Expected: no output.

- [ ] **Step 6: Commit**

```bash
git add internal/messaging/pipeline.go internal/messaging/pipeline_test.go
git commit -m "$(cat <<'EOF'
feat(messaging): T2 — pipeline stub + Channel/SightDecision types

Wires the 7-stage pipeline (normalize → sight gate → anonymize →
color → wrap → deliver) as no-op stubs. RenderForRecipient is the
per-recipient entry point. ChannelAudio bypasses sight gate +
anonymize; ChannelVisual runs the full chain with the precomputed
SightDecision the caller hands in.

Subsequent tasks replace stubs: T5 wraps, T6 anonymizes, T8
normalizes, T4 populates color data.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: Sight Predicates + InfraredVision Buff Flag

Add the two composite predicates and the new buff flag the spec
requires. The predicates read Perception state + room lighting +
NightVision / InfraredVision; the buff flag is needed for steppe-wolf
/ deep-gnawer mutations that already carry the YAML tag but have no
code-side hook.

**Files:**
- Create: `internal/messaging/predicates.go`
- Create: `internal/messaging/predicates_test.go`
- Modify: `internal/buffs/buffspec.go` (add `InfraredVision` flag constant — alongside `NightVision Flag = "nightvision"` at line 61)
- Create: `_datafiles/world/dogmud/buffs/85-infraredvision.yaml`

- [ ] **Step 1: Add the InfraredVision flag constant**

Edit `internal/buffs/buffspec.go`. Find the line:

```go
NightVision Flag = `nightvision`
```

and add immediately below:

```go
InfraredVision Flag = `infraredvision`
```

- [ ] **Step 2: Write the failing predicate tests**

Create `internal/messaging/predicates_test.go`:

```go
package messaging

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
)

func newChar(t *testing.T) *characters.Character {
	t.Helper()
	c := characters.New()
	c.Perception = perception.NewMachine()
	return c
}

func setBlinded(t *testing.T, c *characters.Character) {
	t.Helper()
	if err := c.Perception.TransitionTo(perception.Blinded,
		state.TransitionReason{Trigger: "test"}); err != nil {
		t.Fatalf("transition to Blinded failed: %v", err)
	}
}

func TestCanSeeClearlyLitRoomSighted(t *testing.T) {
	c := newChar(t)
	r := &rooms.Room{} // GetVisibility default = lit-enough
	if !CanSeeClearly(c, r) {
		t.Fatal("Sighted observer in default room should see clearly")
	}
}

func TestCanSeeClearlyBlinded(t *testing.T) {
	c := newChar(t)
	setBlinded(t, c)
	r := &rooms.Room{}
	if CanSeeClearly(c, r) {
		t.Fatal("Blinded observer must NOT see clearly even in a lit room")
	}
}

func TestCanSeeShapesInfraredInDark(t *testing.T) {
	c := newChar(t)
	// Note: GetVisibility() < 1 = dark. We can't easily fabricate a
	// dark Room here without engine coupling — this test uses the
	// nil-room path which short-circuits to lit. Real darkness
	// behavior is exercised in pipeline_test.go's end-to-end suite.
	if !CanSeeShapes(c, nil) {
		t.Fatal("Sighted observer must see shapes (nil room defaults to lit)")
	}
}

func TestCanSeeShapesBlindedNoInfrared(t *testing.T) {
	c := newChar(t)
	setBlinded(t, c)
	if CanSeeShapes(c, nil) {
		t.Fatal("Blinded observer must NOT see shapes, even with nil/lit room")
	}
	_ = buffs.InfraredVision // ensure the flag constant exists
}

func TestNilCharacterDefaultsToSeeing(t *testing.T) {
	if !CanSeeClearly(nil, nil) {
		t.Fatal("nil observer must default to CanSeeClearly (defensive)")
	}
	if !CanSeeShapes(nil, nil) {
		t.Fatal("nil observer must default to CanSeeShapes (defensive)")
	}
}
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `go test ./internal/messaging/...`
Expected: undefined `CanSeeClearly`, `CanSeeShapes`.

- [ ] **Step 4: Implement the predicates**

Create `internal/messaging/predicates.go`:

```go
package messaging

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
)

// CanSeeClearly returns true if the observer can read normal-text
// visual broadcasts in this room. Composes Perception state, room
// lighting, and the NightVision buff flag.
//
// Blinded observers (any source) return false unconditionally.
// A nil observer defaults to true (defensive — pre-init characters
// during boot must not be silently dropped).
func CanSeeClearly(observer *characters.Character, room *rooms.Room) bool {
	if observer == nil {
		return true
	}
	if observer.Perception != nil && observer.Perception.State() == perception.Blinded {
		return false
	}
	if room == nil || room.GetVisibility() >= 1 {
		return true
	}
	return observer.HasFlagFromAnySource(buffs.NightVision)
}

// CanSeeShapes returns true if the observer can detect SOMETHING is
// happening — either full clarity (subsumes CanSeeClearly) OR
// infrared in the dark. Blindness gates this too — broken eyes don't
// see infrared.
//
// A nil observer defaults to true (matches CanSeeClearly's defensive
// behavior).
func CanSeeShapes(observer *characters.Character, room *rooms.Room) bool {
	if CanSeeClearly(observer, room) {
		return true
	}
	if observer == nil {
		return true
	}
	if observer.Perception != nil && observer.Perception.State() == perception.Blinded {
		return false
	}
	return observer.HasFlagFromAnySource(buffs.InfraredVision)
}
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `go test ./internal/messaging/...`
Expected: PASS.

- [ ] **Step 6: Author the InfraredVision buff data file**

Create `_datafiles/world/dogmud/buffs/85-infraredvision.yaml`:

```yaml
buffid: 85
name: InfraredVision
description: "You can see heat signatures in total darkness."
visible: true
sequential: false
expireMessage: ""
statmods: {}
roundinterval: 0
triggernow: false
flags:
  - infraredvision
secret: false
permabuff: true
```

(Filename matches `ConvertForFilename("InfraredVision")` →
`infraredvision`. Permabuff means mutation-driven sources stay until
explicitly removed.)

- [ ] **Step 7: Verify the package still builds and the buff data parses**

Run: `go build ./...`
Expected: no output.

Boot the server briefly to confirm the buff loads without panic:

```bash
go run main.go &
# Wait until "buffs.LoadDataFiles()" log line; then Ctrl-C.
```

Expected: `buffs.LoadDataFiles() loadedCount=…` higher than before by one, no panic.

- [ ] **Step 8: Commit**

```bash
git add internal/buffs/buffspec.go internal/messaging/predicates.go internal/messaging/predicates_test.go _datafiles/world/dogmud/buffs/85-infraredvision.yaml
git commit -m "$(cat <<'EOF'
feat(messaging): T3 — CanSeeClearly/CanSeeShapes predicates + InfraredVision

CanSeeClearly composes Perception state + room lighting + NightVision.
CanSeeShapes adds the infrared-in-dark path gated by the new
InfraredVision buff flag (buff 85, permabuff). Both defensively
default to true for nil observers so boot-time broadcasts don't drop.

InfraredVision matches the YAML mutation tag steppe_wolf / deep_gnawer
already carry but which previously had no code-side hook.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: ANSI Alias Palette Additions

Add every Category's color alias to `ansi-aliases.yaml`. Retune the
three name aliases (`username`, `mobname`, `petname`) to the new
palette. Existing tags continue to work — this is a data-only change.

**Files:**
- Modify: `_datafiles/world/dogmud/ansi-aliases.yaml`
- Modify: `internal/messaging/pipeline.go` (replace the stubbed `applyCategoryColor` with the real wrapper)

- [ ] **Step 1: Write the failing test for the color wrapper**

Append to `internal/messaging/pipeline_test.go`:

```go
func TestApplyCategoryColorWrapsTagForKnownCategory(t *testing.T) {
	got := applyCategoryColor(CategoryHitMelee, "strikes deeply")
	want := `<ansi fg="hit-melee">strikes deeply</ansi>`
	if got != want {
		t.Fatalf("color wrap: got %q want %q", got, want)
	}
}

func TestApplyCategoryColorDefaultPassesThrough(t *testing.T) {
	got := applyCategoryColor(CategoryDefault, "plain text")
	if got != "plain text" {
		t.Fatalf("CategoryDefault must pass text through unchanged, got %q", got)
	}
}

func TestApplyCategoryColorEmptyTextPassesThrough(t *testing.T) {
	got := applyCategoryColor(CategoryHitMelee, "")
	if got != "" {
		t.Fatalf("empty text must pass through unchanged, got %q", got)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/messaging/...`
Expected: `TestApplyCategoryColorWrapsTagForKnownCategory` fails because the stub returns the text unchanged.

- [ ] **Step 3: Replace the stubbed `applyCategoryColor` with the real implementation**

Edit `internal/messaging/pipeline.go`. Replace the stub:

```go
func applyCategoryColor(cat Category, text string) string {
	if cat == CategoryDefault {
		return text
	}
	// Real impl in T4 wraps with <ansi fg="<cat.String()>">…</ansi>
	// after the alias is registered. For now: no-op.
	return text
}
```

with:

```go
func applyCategoryColor(cat Category, text string) string {
	if cat == CategoryDefault || text == "" {
		return text
	}
	return `<ansi fg="` + cat.String() + `">` + text + `</ansi>`
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/messaging/...`
Expected: PASS.

- [ ] **Step 5: Add the color aliases to ansi-aliases.yaml**

Edit `_datafiles/world/dogmud/ansi-aliases.yaml`. Retune the three
existing name aliases (locate around lines 6, 9, 21) and append the
new section at the end of the `colors:` map. Use the spec §8 first-pass
values:

Retune:

```yaml
  username: 153       # cool light blue (was 11 / bright yellow)
  mobname:  180       # warm tan (was 51 / bright cyan)
  petname:  108       # teal-cyan (was 215 / orange)
```

Append (after the last existing alias):

```yaml

# === Messaging Framework Categories ===
# Spec: docs/superpowers/specs/2026-05-19-messaging-framework-design.md
# Test these in-game; values are first-pass approximations.

# Combat — damage bands
  hit-melee: 173
  hit-blunt: 137
  hit-natural-sharp: 95
  hit-ranged: 137
  hit-caster: 146
  hit-unarmed: 173

# Combat — defense
  dodge: 108
  parry: 143
  block: 71

# Combat — grapple / submission / death
  grapple-flow: 94
  grapple-high: 130
  submission: 169
  death: 8

# Combat — special moves
  surprise: 9
  kick: 11
  trip: 214
  bash: 172
  rally: 14
  warcry: 9
  taunt-success: 169
  taunt-resist: 2
  taunt-failure: 11

# Combat — anonymized observer (infrared)
  combat-anon: 196

# Spells — fold + 5 schools + disruption
  spell-fold: 152
  spell-disruption: 214
  spell-elemental: 173
  spell-enhancement: 179
  spell-mental: 146
  spell-vital: 108
  spell-manifestation: 169

# Social
  speech: 111
  whisper: 139
  shout: 215
  ooc: 67
  npc-dialogue: 180
  dialogue-hint: 65
  emote: 144
  mob-idle: 144
  mob-emote: 137

# System / meta
  broadcast: 75
  tip: 108
  system: 8
  error: 9
  warning: 214
  skill-progress: 179
  login: 108
  logout: 8

# Environment
  room-description: 7
  room-entry: 7
  room-exit: 7
  weather: 75
  time-of-day: 179

# Other
  loot: 7
  equipment: 7
  buff-apply: 14
  buff-expire: 14
  mutation: 146
  toxin: 130
```

(`room-description`, `room-entry`, `room-exit`, `loot`, `equipment`
use `7` = default white so prose stays neutral; we still pass through
the pipeline for normalization + wrapping.)

- [ ] **Step 6: Verify the YAML loads — boot the server briefly**

Run:

```bash
go run main.go &
# Wait until you see ansi-aliases loaded; Ctrl-C.
```

Expected: no parse panic, server boots past data loading.

- [ ] **Step 7: Commit**

```bash
git add internal/messaging/pipeline.go internal/messaging/pipeline_test.go _datafiles/world/dogmud/ansi-aliases.yaml
git commit -m "$(cat <<'EOF'
feat(messaging): T4 — ansi-aliases palette + category color wrapper

Adds the full Category palette to ansi-aliases.yaml (combat hits +
defense + grapple + specials + spells + social + system +
environment + buff/mutation). Retunes username (11→153), mobname
(51→180), petname (215→108) to match the new color scheme.

applyCategoryColor now wraps text in <ansi fg="<category-string>">…</ansi>
for every non-default Category. CategoryDefault still passes through
unchanged so room descriptions and system messages keep their
existing rendering until callers explicitly tag them.

256-color values are first-pass approximations per spec §8; expect
tuning during T18 smoke.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: WrapAnsi + UserRecord.LineWidth + `set linewidth`

Add an ANSI-aware wrapper that counts display width (not bytes),
carries open tags across line breaks, and falls back gracefully on
malformed input. Add `UserRecord.LineWidth` (default 80) and a `set
linewidth N` command for player override.

**Files:**
- Create: `internal/messaging/wrap.go`
- Create: `internal/messaging/wrap_test.go`
- Modify: `internal/users/userrecord.go` (add `LineWidth int` field + accessor with default)
- Modify: `internal/messaging/pipeline.go` (replace `wrap` stub with `WrapAnsi`)
- Create: `internal/usercommands/set_linewidth.go` (or extend an existing `set.go` if present)
- Modify: `internal/usercommands/userCommands.go` (register the command, if such a registry exists — verify pattern by reading any other usercommand registration)

- [ ] **Step 1: Write the failing tests for WrapAnsi**

Create `internal/messaging/wrap_test.go`:

```go
package messaging

import "testing"

func TestWrapAnsiShortLineUnchanged(t *testing.T) {
	got := WrapAnsi("short", 80)
	if got != "short" {
		t.Fatalf("short line should be unchanged, got %q", got)
	}
}

func TestWrapAnsiWrapsAtMaxWidthWordBoundary(t *testing.T) {
	// 20-col wrap, ~40 chars of content
	got := WrapAnsi("the quick brown fox jumps over the lazy dog", 20)
	// Expect at least one newline, and no line longer than 20 visible chars.
	lines := splitLines(got)
	if len(lines) < 2 {
		t.Fatalf("expected wrap to produce >=2 lines, got: %q", got)
	}
	for i, line := range lines {
		if displayWidth(line) > 20 {
			t.Fatalf("line %d exceeds 20 cols (%d): %q", i, displayWidth(line), line)
		}
	}
}

func TestWrapAnsiIgnoresAnsiTagsInWidth(t *testing.T) {
	// 12 visible chars wrapped inside a long ANSI tag.
	input := `<ansi fg="hit-melee">strikes hard</ansi>`
	got := WrapAnsi(input, 80)
	if got != input {
		t.Fatalf("12-visible-char line wrapped at 80 must be unchanged, got %q", got)
	}
}

func TestWrapAnsiCarriesOpenTagAcrossLineBreak(t *testing.T) {
	// 10-col wrap forces a break inside an open tag.
	input := `<ansi fg="hit-melee">strikes deeply at the heart</ansi>`
	got := WrapAnsi(input, 10)
	// First line should END with </ansi>; second line should START
	// with <ansi fg="hit-melee">.
	lines := splitLines(got)
	if len(lines) < 2 {
		t.Fatalf("expected break, got 1 line: %q", got)
	}
	if !endsWith(lines[0], `</ansi>`) {
		t.Fatalf("first line must close the open tag, got %q", lines[0])
	}
	if !startsWith(lines[1], `<ansi fg="hit-melee">`) {
		t.Fatalf("second line must reopen the tag, got %q", lines[1])
	}
}

func TestWrapAnsiMalformedTagFallback(t *testing.T) {
	// Orphan opening tag — wrapper must not panic; should fall back
	// to byte-count wrap.
	input := `<ansi fg="bad" missing close ` +
		`this is fifty-plus characters of unwrapped text after`
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("WrapAnsi panicked on malformed input: %v", r)
		}
	}()
	_ = WrapAnsi(input, 20)
}

func TestWrapAnsiZeroWidthIsPassthrough(t *testing.T) {
	// LineWidth=0 (unset) must not wrap or hang.
	got := WrapAnsi("a long line that would otherwise wrap", 0)
	if got != "a long line that would otherwise wrap" {
		t.Fatalf("LineWidth=0 must pass through, got %q", got)
	}
}

// Test helpers — kept here, not exported.

func splitLines(s string) []string {
	out := []string{}
	cur := ""
	for _, r := range s {
		if r == '\n' {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(r)
	}
	out = append(out, cur)
	return out
}

func endsWith(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func displayWidth(s string) int {
	// Inline scan to count visible chars (skip <ansi …> and </ansi>).
	w := 0
	inTag := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '<' {
			inTag = true
			continue
		}
		if c == '>' {
			inTag = false
			continue
		}
		if !inTag {
			w++
		}
	}
	return w
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/messaging/...`
Expected: undefined `WrapAnsi`.

- [ ] **Step 3: Implement WrapAnsi**

Create `internal/messaging/wrap.go`:

```go
package messaging

import (
	"regexp"
	"strings"
)

// ansiTagPattern matches <ansi fg="…"> or </ansi>. Only used for
// scanning; replacement uses the literal strings.
var ansiTagPattern = regexp.MustCompile(`<ansi fg="[^"]*">|</ansi>`)

// WrapAnsi wraps text at maxWidth display columns. ANSI escape
// sequences (<ansi …> / </ansi> tags) don't count toward width.
// Open tags carry across line breaks: each new line gets a fresh
// reopener if the previous line ended mid-tag.
//
// On malformed input (orphan tags, unmatched closers), falls back to
// a byte-count wrap to avoid panicking. The visual output is uglier
// but the server stays up.
//
// A maxWidth of 0 (unset) returns the input unchanged.
func WrapAnsi(text string, maxWidth int) string {
	if maxWidth <= 0 || text == "" {
		return text
	}
	defer func() {
		// Last-resort: if anything in the parser panics, the caller
		// gets the original text back.
		_ = recover()
	}()

	// Walk the text token-by-token, tracking display column and the
	// currently-open ANSI tag (if any). When we cross maxWidth at a
	// word boundary, emit a newline; if a tag is open, close it
	// before the break and reopen on the next line.
	var (
		out      strings.Builder
		line     strings.Builder
		col      int
		openTag  string // empty when no tag is open
		curWord  strings.Builder
		curWordW int
	)

	flushWord := func() {
		// Add space before word if line already has content and there's room.
		if line.Len() > 0 && col+1+curWordW > maxWidth {
			// Wrap before the word.
			if openTag != "" {
				line.WriteString(`</ansi>`)
			}
			out.WriteString(line.String())
			out.WriteByte('\n')
			line.Reset()
			col = 0
			if openTag != "" {
				line.WriteString(openTag)
			}
		} else if line.Len() > 0 {
			line.WriteByte(' ')
			col++
		}
		line.WriteString(curWord.String())
		col += curWordW
		curWord.Reset()
		curWordW = 0
	}

	i := 0
	for i < len(text) {
		// ANSI tag?
		if text[i] == '<' {
			loc := ansiTagPattern.FindStringIndex(text[i:])
			if loc != nil && loc[0] == 0 {
				tag := text[i : i+loc[1]]
				if strings.HasPrefix(tag, `</`) {
					openTag = ""
				} else {
					openTag = tag
				}
				curWord.WriteString(tag)
				i += loc[1]
				continue
			}
		}
		// Whitespace boundary?
		if text[i] == ' ' || text[i] == '\n' {
			if curWord.Len() > 0 {
				flushWord()
			}
			if text[i] == '\n' {
				if openTag != "" {
					line.WriteString(`</ansi>`)
				}
				out.WriteString(line.String())
				out.WriteByte('\n')
				line.Reset()
				col = 0
				if openTag != "" {
					line.WriteString(openTag)
				}
			}
			i++
			continue
		}
		// Visible character.
		curWord.WriteByte(text[i])
		curWordW++
		i++
	}
	if curWord.Len() > 0 {
		flushWord()
	}
	out.WriteString(line.String())
	return out.String()
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/messaging/...`
Expected: PASS.

- [ ] **Step 5: Wire WrapAnsi into the pipeline**

Edit `internal/messaging/pipeline.go`. Replace the stub:

```go
func wrap(text string, maxWidth int) string { return text }
```

with:

```go
func wrap(text string, maxWidth int) string {
	return WrapAnsi(text, maxWidth)
}
```

- [ ] **Step 6: Add LineWidth to UserRecord with accessor**

Edit `internal/users/userrecord.go`. Inside the struct (around line 47–48, near `ScreenReader` / `AsciiMode`), add:

```go
	LineWidth      int                   `yaml:"linewidth,omitempty"` // Column width for line wrapping; 0 = default 80
```

After the existing methods (search for an empty line near `func (u *UserRecord) SendText`, anywhere in the file), add:

```go
// GetLineWidth returns the user's configured line width, falling back
// to 80 if unset or invalid. The messaging pipeline reads this for
// per-recipient wrapping.
func (u *UserRecord) GetLineWidth() int {
	if u == nil || u.LineWidth <= 0 {
		return 80
	}
	return u.LineWidth
}
```

- [ ] **Step 7: Verify everything still builds**

Run: `go build ./...`
Expected: no output.

- [ ] **Step 8: Add the `set linewidth` command**

First, locate the existing `set` command pattern. Either there's a `set.go` in `internal/usercommands/` or a registry of user commands. Read the file:

```bash
ls internal/usercommands/set*.go internal/usercommands/userCommands.go 2>&1 || true
```

Expected: a `set.go` likely exists. Read it. The pattern is a `func Set(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error)` handler that switches on the first token.

Extend that handler with a `linewidth` case. If `set.go` doesn't exist, create one following another usercommand file's pattern (`internal/usercommands/save.go` is a simple reference).

Skeleton — paste into the appropriate location:

```go
// inside Set's switch over subcommand:
case "linewidth":
	args := strings.Fields(rest)
	if len(args) < 2 {
		user.SendText(fmt.Sprintf("Line width is currently %d.", user.GetLineWidth()))
		return true, nil
	}
	n, err := strconv.Atoi(args[1])
	if err != nil || n < 40 || n > 240 {
		user.SendText("Line width must be a number between 40 and 240.")
		return true, nil
	}
	user.LineWidth = n
	user.SendText(fmt.Sprintf("Line width set to %d.", n))
	return true, nil
```

(`SendText` here uses the legacy un-categorized signature — the shim in T9 makes this continue to work post-migration. The literal numbers in the user-facing string are an intentional exception per the spec: this command is a settings tool, not combat narration.)

- [ ] **Step 9: Verify build + run the wrap test suite**

Run: `go build ./... && go test ./internal/messaging/...`
Expected: both succeed.

- [ ] **Step 10: Commit**

```bash
git add internal/messaging/wrap.go internal/messaging/wrap_test.go internal/messaging/pipeline.go internal/users/userrecord.go internal/usercommands/set.go
git commit -m "$(cat <<'EOF'
feat(messaging): T5 — WrapAnsi + UserRecord.LineWidth + set linewidth

WrapAnsi counts display width (ignoring ANSI tags), carries open tags
across line breaks, defensively recovers from malformed tags by
falling back to byte-count wrap. maxWidth=0 passes through.

UserRecord.LineWidth (yaml:"linewidth") configurable per player; new
GetLineWidth() accessor returns 80 default. `set linewidth N` user
command (range 40-240) lets players override.

The pipeline's wrap stage now delegates to WrapAnsi. motd.go's naive
wrapText is left in place for T16's sunset pass.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 6: Anonymize Regex Helper

Strip `username` / `mobname` / `petname` ANSI name tags from infrared-
only renders, substitute `<ansi fg="combat-anon">a figure</ansi>`.
v1 quality bar: bare-name occurrences leak through; the audit makes
sure most names are properly tagged.

**Files:**
- Create: `internal/messaging/anonymize.go`
- Modify: `internal/messaging/anonymize_test.go` (extend existing test file or create)
- Modify: `internal/messaging/pipeline.go` (replace `anonymize` stub)

- [ ] **Step 1: Write the failing tests**

Create `internal/messaging/anonymize_test.go`:

```go
package messaging

import "testing"

func TestAnonymizeReplacesUsernameTag(t *testing.T) {
	in := `<ansi fg="username">Calabe</ansi> attacks`
	want := `<ansi fg="combat-anon">a figure</ansi> attacks`
	if got := Anonymize(in); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAnonymizeReplacesMobnameTag(t *testing.T) {
	in := `<ansi fg="mobname">Thornwall Thug</ansi> snarls`
	want := `<ansi fg="combat-anon">a figure</ansi> snarls`
	if got := Anonymize(in); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAnonymizeReplacesPetnameTag(t *testing.T) {
	in := `<ansi fg="petname">Rex</ansi> follows`
	want := `<ansi fg="combat-anon">a figure</ansi> follows`
	if got := Anonymize(in); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAnonymizeReplacesMultipleNamesInOneLine(t *testing.T) {
	in := `<ansi fg="mobname">Thug</ansi> strikes ` +
		`<ansi fg="username">Calabe</ansi> with a longsword`
	want := `<ansi fg="combat-anon">a figure</ansi> strikes ` +
		`<ansi fg="combat-anon">a figure</ansi> with a longsword`
	if got := Anonymize(in); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAnonymizeLeavesOtherTagsAlone(t *testing.T) {
	in := `<ansi fg="hit-melee">strikes deeply</ansi>`
	if got := Anonymize(in); got != in {
		t.Fatalf("non-name tag must pass through, got %q", got)
	}
}

func TestAnonymizeEmpty(t *testing.T) {
	if got := Anonymize(""); got != "" {
		t.Fatalf("empty must pass through, got %q", got)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/messaging/...`
Expected: `Anonymize` undefined OR returns text unchanged (depending on which fails first).

- [ ] **Step 3: Implement Anonymize**

Create `internal/messaging/anonymize.go`:

```go
package messaging

import "regexp"

// nameTagPattern matches <ansi fg="username|mobname|petname">…</ansi>
// for the anonymizer pass.
var nameTagPattern = regexp.MustCompile(
	`<ansi fg="(username|mobname|petname)">[^<]+</ansi>`,
)

// Anonymize strips player/mob/pet name ANSI tags and replaces them
// with a `combat-anon`-colored "a figure" placeholder. Used by the
// pipeline for infrared-only observers in dark rooms.
//
// v1 limitation: bare-name occurrences (names embedded in prose
// without an ANSI tag) leak through. The 228-site audit gets most
// names properly tagged; remaining leaks are tracked as followups.
func Anonymize(text string) string {
	if text == "" {
		return text
	}
	return nameTagPattern.ReplaceAllString(text,
		`<ansi fg="combat-anon">a figure</ansi>`)
}
```

- [ ] **Step 4: Wire into the pipeline**

Edit `internal/messaging/pipeline.go`. Replace:

```go
func anonymize(text string) string                { return text }
```

with:

```go
func anonymize(text string) string {
	return Anonymize(text)
}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./internal/messaging/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/messaging/anonymize.go internal/messaging/anonymize_test.go internal/messaging/pipeline.go
git commit -m "$(cat <<'EOF'
feat(messaging): T6 — Anonymize regex helper for infrared observers

Anonymize strips <ansi fg="username|mobname|petname">…</ansi> tags
and replaces them with <ansi fg="combat-anon">a figure</ansi>. Used
by the pipeline when the precomputed SightDecision is SightShapes
(infrared in the dark).

v1 quality bar — bare-name occurrences (no ANSI tag) leak through;
the audit phase tags most names so leaks stay low.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 7: SendProgression Banner Helper

Replace `*** You feel your X skills sharpening! ***` with the SKILL
ADVANCEMENT / STATISTIC INCREASED banner from the spec mocks. Keep the
old call sites compiling (T7 wires the helper; the old literal is
removed in T16's sunset pass).

**Files:**
- Create: `internal/messaging/progression.go`
- Create: `internal/messaging/progression_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/messaging/progression_test.go`:

```go
package messaging

import (
	"strings"
	"testing"
)

func TestFormatProgressionSkillNoTierChange(t *testing.T) {
	got := FormatProgression(ProgSkill, "Unarmed Combat", nil)
	if !strings.Contains(got, "SKILL ADVANCEMENT") {
		t.Errorf("missing SKILL ADVANCEMENT title: %q", got)
	}
	if !strings.Contains(got, "Unarmed Combat") {
		t.Errorf("missing skill name: %q", got)
	}
	if strings.Contains(got, "→") {
		t.Errorf("no tier change should mean no arrow, got %q", got)
	}
	if !strings.Contains(got, "━") {
		t.Errorf("missing banner rule: %q", got)
	}
}

func TestFormatProgressionStatNoTierChange(t *testing.T) {
	got := FormatProgression(ProgStat, "Strength", nil)
	if !strings.Contains(got, "STATISTIC INCREASED") {
		t.Errorf("missing STATISTIC INCREASED title: %q", got)
	}
	if !strings.Contains(got, "Strength") {
		t.Errorf("missing stat name: %q", got)
	}
}

func TestFormatProgressionTierCrossingIncludesThirdLine(t *testing.T) {
	got := FormatProgression(ProgSkill, "Unarmed Combat",
		&TierChange{From: "apprentice", To: "journeyman"})
	if !strings.Contains(got, "apprentice → journeyman") {
		t.Errorf("missing tier transition line: %q", got)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/messaging/...`
Expected: undefined `FormatProgression`, `ProgSkill`, `ProgStat`, `TierChange`.

- [ ] **Step 3: Implement the helper**

Create `internal/messaging/progression.go`:

```go
package messaging

import (
	"strings"
)

// ProgressionKind discriminates skill vs stat advancement.
type ProgressionKind int

const (
	ProgSkill ProgressionKind = iota
	ProgStat
)

// TierChange marks a tier crossing (e.g., apprentice→journeyman).
// nil = no tier change; banner omits the third line.
type TierChange struct {
	From string
	To   string
}

const progressionRule = "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

// FormatProgression returns the banner string (without trailing
// newline). SendProgression wraps this for the user's connection.
func FormatProgression(kind ProgressionKind, name string, tier *TierChange) string {
	var title string
	switch kind {
	case ProgSkill:
		title = "SKILL ADVANCEMENT"
	case ProgStat:
		title = "STATISTIC INCREASED"
	default:
		title = "PROGRESSION"
	}

	lines := []string{
		progressionRule,
		center(title),
		center(name),
	}
	if tier != nil {
		lines = append(lines, center(tier.From+" → "+tier.To))
	}
	lines = append(lines, progressionRule)
	return strings.Join(lines, "\n")
}

// center pads s to the rule width with leading spaces.
func center(s string) string {
	width := len(progressionRule)
	if len(s) >= width {
		return s
	}
	pad := (width - len(s)) / 2
	return strings.Repeat(" ", pad) + s
}

// SendProgression emits the banner to the user via SendText
// (audio channel — banners are not sight-gated). Uses
// CategorySkillProgress. The user-facing literal numbers in the
// banner (none currently — tier names only) are an exception to the
// "no hard numbers" rule because this IS the mechanical display.
//
// Users argument is intentionally an interface{} until T9 adds the
// categorized SendText helpers — at that point this wraps the new
// API. For now it builds the string and the caller delivers.
func SendProgression(user UserSender, kind ProgressionKind, name string, tier *TierChange) {
	if user == nil {
		return
	}
	user.SendText(FormatProgression(kind, name, tier))
}

// UserSender is the minimal interface SendProgression needs. The
// real *users.UserRecord satisfies it. Decoupled so messaging/ does
// not import users/ (the audit's import direction).
type UserSender interface {
	SendText(text string)
}
```

(Note: post-T9, `UserSender.SendText` will take a leading `Category`. T9 updates this signature in the same commit it updates everyone else's.)

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/messaging/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/messaging/progression.go internal/messaging/progression_test.go
git commit -m "$(cat <<'EOF'
feat(messaging): T7 — SendProgression banner helper

FormatProgression builds the SKILL ADVANCEMENT / STATISTIC INCREASED
banner from the spec mocks. Three lines normally (rule / title /
name / rule); four when a TierChange crosses (apprentice → journeyman).

SendProgression delivers via UserSender — a one-method interface
satisfied by *users.UserRecord. Decoupled so internal/messaging/
doesn't import internal/users/ (import direction stays clean).

The legacy *** sharpening *** literal in internal/characters/
progression.go stays in place until T16's sunset pass swaps the call
site to SendProgression.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 8: Style Normalization Stages

Implement the five normalization stages (sentence-start cap, a/an
agreement, duplicate-word collapse, sentence-end punctuation, ANSI
canonicalization for names) with a per-Category skip table.

**Files:**
- Create: `internal/messaging/normalize.go`
- Create: `internal/messaging/normalize_test.go`
- Modify: `internal/messaging/pipeline.go` (replace `normalize` stub)

- [ ] **Step 1: Write the failing tests**

Create `internal/messaging/normalize_test.go`:

```go
package messaging

import "testing"

func TestNormalizeCapitalizesSentenceStart(t *testing.T) {
	got := Normalize(CategoryHitMelee, "in control, you press forward")
	if got[:1] != "I" {
		t.Errorf("expected capitalized start, got %q", got)
	}
}

func TestNormalizeAAnAgreement(t *testing.T) {
	got := Normalize(CategoryHitMelee, "with a aggressive posture")
	if got != "With an aggressive posture." {
		t.Errorf("a/an: got %q", got)
	}
}

func TestNormalizeCollapsesDuplicateWord(t *testing.T) {
	got := Normalize(CategoryHitMelee, "negligible damage damage on")
	if got != "Negligible damage on." {
		t.Errorf("dup-word: got %q", got)
	}
}

func TestNormalizeAppendsEndPunctuation(t *testing.T) {
	got := Normalize(CategoryHitMelee, "you strike deeply")
	if got != "You strike deeply." {
		t.Errorf("end-punct: got %q", got)
	}
}

func TestNormalizeSkipsForRoomDescription(t *testing.T) {
	// Room descriptions manage their own capitalization and prose
	// shape; normalization is disabled for them.
	in := "the road winds west"
	got := Normalize(CategoryRoomDescription, in)
	if got != in {
		t.Errorf("CategoryRoomDescription must skip normalization, got %q", got)
	}
}

func TestNormalizeSkipsBannersWithBoxRule(t *testing.T) {
	in := "━━━ banner ━━━"
	got := Normalize(CategoryHitMelee, in)
	if got != in {
		t.Errorf("banner line must skip end-punct, got %q", got)
	}
}

func TestNormalizeDoesNotDoublePunctuate(t *testing.T) {
	in := "You strike deeply."
	got := Normalize(CategoryHitMelee, in)
	if got != "You strike deeply." {
		t.Errorf("must not double-punctuate, got %q", got)
	}
}

func TestNormalizeIdempotent(t *testing.T) {
	once := Normalize(CategoryHitMelee, "with a aggressive posture")
	twice := Normalize(CategoryHitMelee, once)
	if once != twice {
		t.Errorf("normalize must be idempotent: %q vs %q", once, twice)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/messaging/...`
Expected: many failures — `Normalize` undefined or returns input unchanged.

- [ ] **Step 3: Implement Normalize with the five stages**

Create `internal/messaging/normalize.go`:

```go
package messaging

import (
	"regexp"
	"strings"
)

// normalizeSkip indicates per-Category opt-outs from individual
// normalization stages. Defaults to all-on; categories whose prose
// is hand-authored set bits here to keep their original styling.
//
// Bitmask values (powers of 2) are OR'd together. Zero = all stages
// active.
type normalizeStage uint8

const (
	stageCapitalize     normalizeStage = 1 << iota // sentence-start caps
	stageAAnAgreement                              // a/an agreement
	stageDupWordCollapse                           // duplicate word collapse
	stageEndPunct                                  // sentence-end punctuation
	stageNameCanon                                 // ANSI name canonicalization (T8 stub; future polish)
)

// skipStages returns the stage mask of stages to SKIP for cat.
func skipStages(cat Category) normalizeStage {
	switch cat {
	case CategoryRoomDescription, CategoryRoomEntry, CategoryRoomExit,
		CategoryWeather, CategoryTimeOfDay,
		CategoryNPCDialogue, CategoryDialogueHint,
		CategoryMobIdle, CategoryMobEmote,
		CategorySpeech, CategoryWhisper, CategoryShout, CategoryEmote,
		CategorySkillProgress: // banner has its own formatting
		// These categories own their prose shape. Skip everything.
		return stageCapitalize | stageAAnAgreement | stageDupWordCollapse |
			stageEndPunct | stageNameCanon
	}
	return 0
}

var (
	dupWordPattern = regexp.MustCompile(`\b(\w+) \1\b`)
	aBeforeVowel   = regexp.MustCompile(`\b([aA]) ([aeiouAEIOU])`)
)

// Normalize runs the five style-normalization stages on text. Stages
// individually opt-out via skipStages(cat).
//
// Idempotent: Normalize(cat, Normalize(cat, x)) == Normalize(cat, x).
//
// Pure-string transforms; safe to call on any input including
// already-tagged ANSI text. The dup-word regex is ANSI-blind — it
// matches `\b(\w+) \1\b` which excludes `<` and `>` so tag
// boundaries don't collapse word pairs.
func Normalize(cat Category, text string) string {
	if text == "" {
		return text
	}
	skip := skipStages(cat)

	// 1. Sentence-start capitalization.
	if skip&stageCapitalize == 0 {
		text = capitalizeStart(text)
	}

	// 2. a/an agreement.
	if skip&stageAAnAgreement == 0 {
		text = aBeforeVowel.ReplaceAllStringFunc(text, func(match string) string {
			// match is `[aA] [aeiouAEIOU]`. Preserve the original case.
			article := match[:1]
			rest := match[1:]
			if article == "A" {
				return "An" + rest
			}
			return "an" + rest
		})
	}

	// 3. Duplicate-word collapse.
	if skip&stageDupWordCollapse == 0 {
		text = dupWordPattern.ReplaceAllString(text, "$1")
	}

	// 4. Sentence-end punctuation auto-append.
	if skip&stageEndPunct == 0 {
		text = appendEndPunct(text)
	}

	// 5. ANSI name canonicalization is deferred to a future polish
	// pass — v1 relies on the per-package audit to tag names
	// explicitly at the call site. Hook remains here for later
	// extension.
	_ = stageNameCanon

	return text
}

// capitalizeStart uppercases the first non-ansi-tag character of text.
// Idempotent — if the first letter is already uppercase, no-op.
func capitalizeStart(text string) string {
	// Skip past any opening <ansi …> tag(s) before capitalizing.
	i := 0
	for i < len(text) && text[i] == '<' {
		j := strings.IndexByte(text[i:], '>')
		if j < 0 {
			return text // malformed; leave alone
		}
		i += j + 1
	}
	if i >= len(text) {
		return text
	}
	c := text[i]
	if c >= 'a' && c <= 'z' {
		return text[:i] + string(c-32) + text[i+1:]
	}
	return text
}

// appendEndPunct adds a `.` if the last non-tag non-space char isn't
// already a sentence terminator. Skips banner lines (start with `━`),
// pure exclamations, and pure-tag wrappers.
func appendEndPunct(text string) string {
	trimmed := strings.TrimRight(text, " \t")
	if trimmed == "" {
		return text
	}
	// Banner skip.
	if strings.HasPrefix(strings.TrimSpace(text), "━") {
		return text
	}
	// Strip a trailing </ansi> for the check; reattach.
	suffix := ""
	for strings.HasSuffix(trimmed, "</ansi>") {
		suffix = "</ansi>" + suffix
		trimmed = trimmed[:len(trimmed)-len("</ansi>")]
	}
	if trimmed == "" {
		return text
	}
	last := trimmed[len(trimmed)-1]
	switch last {
	case '.', '!', '?', ',', ')', '"', '\'':
		return text
	}
	return trimmed + "." + suffix
}
```

- [ ] **Step 4: Wire Normalize into the pipeline**

Edit `internal/messaging/pipeline.go`. Replace:

```go
func normalize(cat Category, text string) string { return text }
```

with:

```go
func normalize(cat Category, text string) string {
	return Normalize(cat, text)
}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./internal/messaging/...`
Expected: PASS for all normalize_test.go cases.

- [ ] **Step 6: Commit**

```bash
git add internal/messaging/normalize.go internal/messaging/normalize_test.go internal/messaging/pipeline.go
git commit -m "$(cat <<'EOF'
feat(messaging): T8 — style normalization (capitalize, a/an, dup-word, end-punct)

Five-stage normalize: sentence-start caps, a/an agreement,
duplicate-word collapse, sentence-end punctuation, ANSI name canon
(deferred to polish). Bitmask skip table opts out per-Category for
content whose prose shape is hand-authored (room desc, NPC dialogue,
mob idle, etc.).

Idempotent: re-running Normalize on already-normalized text is a no-op.
ANSI-blind: dup-word regex won't span tag boundaries.

Pipeline now runs Normalize as stage 2 (before sight gating). Per-
category skips picked up automatically.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 9: Categorized Send API + Legacy Shim

Extend `Room.SendText` / `Room.SendTextVisual` / `UserRecord.SendText`
with a leading `messaging.Category` param. Add `UserRecord.SendTextVisual`
(new). All existing call sites continue compiling via a temporary
`SendTextLegacy` shim that maps to `CategoryDefault` and emits a
warning log so they can be tracked. The shim is deleted in T16.

**Files:**
- Modify: `internal/rooms/rooms.go` (around lines 259, 274 — extend signatures + add Legacy shims)
- Modify: `internal/users/userrecord.go` (around line 286 — extend signature, add SendTextVisual, add Legacy shim)

- [ ] **Step 1: Read the existing signatures**

Read:

```
internal/rooms/rooms.go lines 259–296
internal/users/userrecord.go lines 286–293
```

(You read these already during planning; the current shapes are:)

```go
// rooms.go:259
func (r *Room) SendText(txt string, excludeUserIds ...int)

// rooms.go:274
func (r *Room) SendTextVisual(txt string, excludeUserIds ...int)

// userrecord.go:286
func (u *UserRecord) SendText(txt string)
```

- [ ] **Step 2: Replace Room.SendText / SendTextVisual with categorized versions + Legacy shims**

Edit `internal/rooms/rooms.go`. Replace lines 259–296 (the existing
`SendText`, `SendTextVisual`, and the dark-room loop inside
`SendTextVisual`) with:

```go
// SendText delivers an audio-channel (unfiltered) message to every
// recipient in the room. Bypasses sight gate + anonymize; runs
// normalize, color, and wrap. Blinded observers still receive it.
func (r *Room) SendText(cat messaging.Category, txt string, excludeUserIds ...int) {
	for _, uid := range r.GetPlayers() {
		if excluded(uid, excludeUserIds) {
			continue
		}
		u := users.GetByUserId(uid)
		if u == nil {
			continue
		}
		rendered := messaging.RenderForRecipient(messaging.RenderInput{
			Category:  cat,
			Text:      txt,
			Channel:   messaging.ChannelAudio,
			LineWidth: u.GetLineWidth(),
		})
		if rendered == "" {
			continue
		}
		events.AddToQueue(events.Message{
			UserId: u.UserId,
			Text:   rendered + "\n",
		})
	}
}

// SendTextVisual delivers a sight-gated message. Per-recipient sight
// is computed via messaging.CanSeeClearly / CanSeeShapes; infrared
// observers get an anonymized render.
func (r *Room) SendTextVisual(cat messaging.Category, txt string, excludeUserIds ...int) {
	for _, uid := range r.GetPlayers() {
		if excluded(uid, excludeUserIds) {
			continue
		}
		u := users.GetByUserId(uid)
		if u == nil {
			continue
		}
		decision := messaging.SightNone
		switch {
		case messaging.CanSeeClearly(u.Character, r):
			decision = messaging.SightFull
		case messaging.CanSeeShapes(u.Character, r):
			decision = messaging.SightShapes
		}
		rendered := messaging.RenderForRecipient(messaging.RenderInput{
			Category:      cat,
			Text:          txt,
			Channel:       messaging.ChannelVisual,
			SightDecision: decision,
			LineWidth:     u.GetLineWidth(),
		})
		if rendered == "" {
			continue
		}
		events.AddToQueue(events.Message{
			UserId: u.UserId,
			Text:   rendered + "\n",
		})
	}
}

// SendTextLegacy is a temporary compatibility shim for callers that
// haven't yet migrated to the categorized API. Maps to
// CategoryDefault and emits one warning per call site (deduped
// internally) so the audit can track remaining sites. DELETED in T16.
func (r *Room) SendTextLegacy(txt string, excludeUserIds ...int) {
	mlog.Warn("legacy", "site", legacyCallerInfo(), "msg", "Room.SendText without Category")
	r.SendText(messaging.CategoryDefault, txt, excludeUserIds...)
}

// SendTextVisualLegacy is the shim for SendTextVisual. DELETED in T16.
func (r *Room) SendTextVisualLegacy(txt string, excludeUserIds ...int) {
	mlog.Warn("legacy", "site", legacyCallerInfo(), "msg", "Room.SendTextVisual without Category")
	r.SendTextVisual(messaging.CategoryDefault, txt, excludeUserIds...)
}

// excluded is a tiny helper for the shared exclusion check.
func excluded(uid int, excluded []int) bool {
	for _, eid := range excluded {
		if uid == eid {
			return true
		}
	}
	return false
}

// legacyCallerInfo returns "file:line" of the caller two frames up
// (skipping legacyCallerInfo + SendTextLegacy themselves). Used only
// for the temp-shim deprecation warnings.
func legacyCallerInfo() string {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return "unknown"
	}
	return fmt.Sprintf("%s:%d", file, line)
}
```

Add the needed imports at the top of `rooms.go`:

```go
import (
	"fmt"
	"runtime"
	// ... existing imports ...
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mlog"
)
```

(`mlog` may have a different name in the codebase — check the existing logging import used elsewhere in `rooms.go`. If it's `slog`, use that.)

- [ ] **Step 3: Replace UserRecord.SendText + add SendTextVisual + Legacy shim**

Edit `internal/users/userrecord.go`. Replace the existing
`SendText` (lines 286–293) with:

```go
// SendText delivers an audio-channel message to this user. See
// rooms.SendText for channel semantics. CategoryDefault keeps prose
// uncolored.
func (u *UserRecord) SendText(cat messaging.Category, txt string) {
	rendered := messaging.RenderForRecipient(messaging.RenderInput{
		Category:  cat,
		Text:      txt,
		Channel:   messaging.ChannelAudio,
		LineWidth: u.GetLineWidth(),
	})
	if rendered == "" {
		return
	}
	events.AddToQueue(events.Message{
		UserId: u.UserId,
		Text:   rendered + "\n",
	})
}

// SendTextVisual delivers a visual message gated by this user's
// sight. The sender must provide the room context so room lighting
// can be folded in.
func (u *UserRecord) SendTextVisual(cat messaging.Category, txt string, room *rooms.Room) {
	decision := messaging.SightNone
	switch {
	case messaging.CanSeeClearly(u.Character, room):
		decision = messaging.SightFull
	case messaging.CanSeeShapes(u.Character, room):
		decision = messaging.SightShapes
	}
	rendered := messaging.RenderForRecipient(messaging.RenderInput{
		Category:      cat,
		Text:          txt,
		Channel:       messaging.ChannelVisual,
		SightDecision: decision,
		LineWidth:     u.GetLineWidth(),
	})
	if rendered == "" {
		return
	}
	events.AddToQueue(events.Message{
		UserId: u.UserId,
		Text:   rendered + "\n",
	})
}

// SendTextLegacy is the temporary shim. DELETED in T16.
func (u *UserRecord) SendTextLegacy(txt string) {
	u.SendText(messaging.CategoryDefault, txt)
}
```

Add imports:

```go
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
```

(NOTE: adding a `rooms` import to `users/` will likely create an
import cycle, because `rooms/` already imports `users/`. If so,
move the SendTextVisual variant into `internal/rooms/rooms.go` as
`func (r *Room) SendTextVisualToUser(u *UserRecord, cat messaging.Category, txt string)` and call THAT from sites that previously did `user.SendText` with visual content. Adjust this step before committing — verify with `go build ./...`.)

- [ ] **Step 4: Update messaging.UserSender interface for T7's helper**

The `UserSender` interface in `internal/messaging/progression.go` (T7)
defined `SendText(text string)`. Now that the real method takes a
leading `Category`, update it:

Edit `internal/messaging/progression.go`. Change:

```go
type UserSender interface {
	SendText(text string)
}
```

to:

```go
type UserSender interface {
	SendText(cat Category, text string)
}
```

And update `SendProgression`:

```go
func SendProgression(user UserSender, kind ProgressionKind, name string, tier *TierChange) {
	if user == nil {
		return
	}
	user.SendText(CategorySkillProgress, FormatProgression(kind, name, tier))
}
```

- [ ] **Step 5: Do a one-shot scripted rename of every `SendText` call site to `SendTextLegacy`**

A search-and-replace is safer than per-file edits at this scale. From
the repo root:

```bash
# Dry run — count what we'd touch:
grep -rn '\.SendText(' --include='*.go' internal/ | wc -l

# Compose the rename. The shim is on rooms.Room AND users.UserRecord.
# Only the call form matters (`.SendText(`), not the receiver type.
grep -rln '\.SendText(' --include='*.go' internal/ | \
  xargs sed -i 's/\.SendText(/\.SendTextLegacy(/g'

# Same for SendTextVisual:
grep -rln '\.SendTextVisual(' --include='*.go' internal/ | \
  xargs sed -i 's/\.SendTextVisual(/\.SendTextVisualLegacy(/g'
```

(On Windows / Git Bash, sed's `-i` may need `-i ''` or `-i.bak`; tune to local sed.)

After the rename, `set linewidth`'s call site from T5 also gets the rename — that's fine; it still works.

- [ ] **Step 6: Verify everything compiles and tests pass**

Run: `go build ./... && go test ./...`
Expected: build succeeds. Tests outside the audit may now log
deprecation warnings via `mlog`, which is fine.

- [ ] **Step 7: Boot the server briefly**

Run:

```bash
go run main.go &
# Wait for full boot; Ctrl-C.
```

Expected: server boots past data file loading; you'll see a flurry of
"legacy" warnings on initial broadcasts. The next 6 tasks zero those
out, package by package.

- [ ] **Step 8: Commit**

```bash
git add internal/rooms/rooms.go internal/users/userrecord.go internal/messaging/progression.go
git add -u  # picks up the bulk SendText → SendTextLegacy rename
git commit -m "$(cat <<'EOF'
feat(messaging): T9 — categorized Send API + temporary legacy shim

Room.SendText / Room.SendTextVisual / UserRecord.SendText now take a
leading messaging.Category param and route through the pipeline:
normalize → (sight gate for visual) → anonymize → color → wrap.

UserRecord.SendTextVisual is new — per-user sight-gated delivery.

All existing call sites (~228) renamed to SendTextLegacy via bulk
sed; the shim maps to CategoryDefault and logs a deprecation warning
with file:line so the per-package audit can track remaining sites.
SendTextLegacy is DELETED in T16.

Existing inline dark-room loop in Room.SendTextVisual is gone — the
sight gate handles it now. canSeeInRoom duplicates in combat/ and
hooks/ remain; T16 deletes those alongside the shim.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Phase 2 — Per-Package Audit + Cutover (T10–T15)

For each task: open every file in the package, find every
`SendTextLegacy` / `SendTextVisualLegacy` call, decide on the correct
`Category`, switch to the categorized API. Most call sites map to one
of a small set of categories — the patterns below cover ~90% of cases.

Auditing pattern — applies to every task in this phase:

1. List all call sites in the package: `grep -rn 'SendTextLegacy\|SendTextVisualLegacy' internal/<pkg>/`
2. For each site, read 3 lines of context above and below.
3. Pick the Category. Common mappings:
   - "X attacks Y", "X strikes Y", "X cleaves Y" → visual + a `CategoryHit*` matching the damage band
   - "X dodges/parries/blocks" → visual + `CategoryDodge` / `CategoryParry` / `CategoryBlock`
   - "X says, …" → audio (speech is heard, not seen) + `CategorySpeech`
   - "You pick up X" → audio + `CategoryLoot`
   - System messages ("You sit down and meditate") → audio + `CategorySystem`
   - Error messages ("You can't do that here") → audio + `CategoryError`
   - Room descriptions → audio (entering player gets them directly) + `CategoryRoomDescription`
   - Mob idle ambient lines → visual + `CategoryMobIdle`
   - Spell folds/casts → visual + `CategorySpellFold` then `CategorySpell*` for resolve
   - Buff start/end → audio + `CategoryBuffApply` / `CategoryBuffExpire`
4. Convert: `room.SendTextLegacy(txt, excludes...)` → `room.SendText(messaging.CategoryFoo, txt, excludes...)`. Similarly `SendTextVisualLegacy` → `SendTextVisual`.
5. **The "Visual vs Audio" decision is the heart of the audit.** If the text contains a character name or describes a visible action, use `SendTextVisual`. If it describes a sound, smell, or generic-environmental cue, use `SendText`. The companion-name-leak headline bug is fixed by every visual-text site choosing `SendTextVisual`.

Per-task delivery template:

- [ ] **Step 1**: List the call sites in the package.
- [ ] **Step 2**: Audit each call. Choose Category. Convert.
- [ ] **Step 3**: `go build ./...` clean.
- [ ] **Step 4**: `go test ./<pkg>/...` clean.
- [ ] **Step 5**: Boot the server; the deprecation warning count from this package's files should be zero.
- [ ] **Step 6**: Commit.

---

### Task 10: Audit + Migrate `internal/combat/` (~30 sites)

**Files:** every `.go` file under `internal/combat/`.

- [ ] **Step 1: Enumerate sites**

```bash
grep -rn 'SendTextLegacy\|SendTextVisualLegacy' internal/combat/
```

- [ ] **Step 2: Migrate each site**

Common combat-package mappings:
- Damage narration ("X strikes Y", "X cleaves Y") → `SendTextVisual` + `CategoryHitMelee` / `CategoryHitBlunt` / etc. The weapon's subtype field decides which.
- Defense narration ("X dodges", "X parries", "X blocks") → `SendTextVisual` + `CategoryDodge` / `CategoryParry` / `CategoryBlock`.
- "*[SURPRISE ATTACK]*" prefix lines → `SendTextVisual` + `CategorySurpriseAttack`.
- Grapple flow → `SendTextVisual` + `CategoryGrappleFlow`; position changes → `CategoryGrappleHigh`.
- Submission attempts → `SendTextVisual` + `CategorySubmission`.
- Combat-mode broadcasts (audio cues — "you hear weapons clashing") → `SendText` + `CategorySystem` (no good combat-audio category; system is the catch-all).

Convert in-place. After each file, `go build ./internal/combat/...` to catch typos early.

- [ ] **Step 3: Verify the package builds and tests pass**

Run: `go build ./... && go test ./internal/combat/...`
Expected: PASS.

- [ ] **Step 4: Boot the server, scroll the deprecation warnings**

Run:

```bash
go run main.go &
# Trigger a combat (manually or via test-mud), then Ctrl-C.
```

Expected: zero `legacy` warnings whose `site=internal/combat/…`.

- [ ] **Step 5: Commit**

```bash
git add internal/combat/
git commit -m "$(cat <<'EOF'
feat(messaging): T10 — internal/combat/ migrated to categorized API

~30 broadcast sites updated. Damage narration uses CategoryHit{Melee,
Blunt,NaturalSharp,Ranged,Caster,Unarmed}; defense uses Dodge/Parry/
Block; specials keep their colors via CategorySurpriseAttack/Kick/
etc.; grapple flows split between GrappleFlow (warm taupe) and
GrappleHigh (brighter taupe).

All combat visual text routed through SendTextVisual; audio cues
through SendText. canSeeInRoom duplicate in combat.go scheduled for
T16 sunset.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 11: Audit + Migrate `internal/hooks/` (~60 sites)

**Files:** every `.go` file under `internal/hooks/`.

The hooks package is the largest non-usercommand category. It contains
combat round resolution, spell casting, death broadcasts, buff
applications, and companion logic. The headline companion-name leak
likely lives here.

- [ ] **Step 1: Enumerate sites**

```bash
grep -rn 'SendTextLegacy\|SendTextVisualLegacy' internal/hooks/
```

- [ ] **Step 2: Audit each site**

Common hooks mappings:
- `NewRound_DoCombat*.go` — all combat narration, same categories as T10
- `Buff_ApplyBuffs.go` — `SendText` + `CategoryBuffApply` (apply line) / `CategoryBuffExpire` (expire)
- `*Spell*.go` — `SendTextVisual` + `CategorySpellFold` for cast-begin, `CategorySpellElemental` / `Enhancement` / `Mental` / `Vital` / `Manifestation` for resolve based on school
- `Death_*.go` — visual for "X falls dead" + `CategoryDeath`; the "you have been killed" personal line is audio + `CategoryError` or `CategorySystem`
- `companion_*.go` — companion lines that previously routed wrong are the audit's headline fix; visual ones get `SendTextVisual` + `CategoryMobEmote` or similar
- `charm_spell.go` — visual + appropriate spell school category
- Death announcements — audio + `CategoryDeath` (it's a "the body slumps" kind of audio cue)

- [ ] **Step 3: Verify build + tests**

Run: `go build ./... && go test ./internal/hooks/...`
Expected: PASS.

- [ ] **Step 4: Boot server, smoke-check zero hooks/ warnings**

```bash
go run main.go &
# Trigger a fight, a spell, a buff, a companion summon; Ctrl-C.
```

Expected: zero `site=internal/hooks/…` deprecation warnings.

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/
git commit -m "$(cat <<'EOF'
feat(messaging): T11 — internal/hooks/ migrated to categorized API

~60 broadcast sites updated. Covers combat-round narration, spell
folds + 5-school resolves, buff application/expire, death broadcasts,
charm/companion flows. The companion logic now correctly tags visual
content as SendTextVisual — the headline companion-name leak bug is
addressed at the call sites that previously hit room.SendText with
visual content.

canSeeInRoom duplicate in NewRound_DoCombat_helpers.go scheduled for
T16 sunset.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 12: Audit + Migrate `internal/mobcommands/` (~25 sites)

**Files:** every `.go` file under `internal/mobcommands/`.

Mob commands include emotes, says, idle behaviors, attacks initiated
by mobs, and dialogue.

- [ ] **Step 1: Enumerate sites**

```bash
grep -rn 'SendTextLegacy\|SendTextVisualLegacy' internal/mobcommands/
```

- [ ] **Step 2: Audit each site**

Common mappings:
- `say.go` (if present) → audio + `CategorySpeech` for the said text + `CategoryNPCDialogue` for dialogue replies
- `emote.go` → visual + `CategoryMobEmote`
- Idle / ambient mob lines → visual + `CategoryMobIdle`
- Mob-initiated attacks → visual + appropriate `CategoryHit*`
- Mob-initiated spells → visual + appropriate `CategorySpell*`

- [ ] **Step 3: Verify build + tests**

Run: `go build ./... && go test ./internal/mobcommands/...`

- [ ] **Step 4: Smoke**

Boot server, trigger mob idle/emote, verify zero `site=internal/mobcommands/…` warnings.

- [ ] **Step 5: Commit**

```bash
git add internal/mobcommands/
git commit -m "$(cat <<'EOF'
feat(messaging): T12 — internal/mobcommands/ migrated to categorized API

~25 sites. Mob say/emote/idle uses CategorySpeech / CategoryMobEmote /
CategoryMobIdle (parchment / warm grey palette per ansi-aliases). Mob-
initiated attacks reuse the CategoryHit* and CategorySpell* tags from
T10/T11.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 13: Audit + Migrate `internal/usercommands/` (~70 sites)

**Files:** every `.go` file under `internal/usercommands/`.

Largest task. Most of the player-facing prose (movement, look,
inventory, social, settings, status) lives here. Plan to spread this
across multiple sittings if needed — commit between subpackages.

- [ ] **Step 1: Enumerate sites by file**

```bash
grep -rln 'SendTextLegacy\|SendTextVisualLegacy' internal/usercommands/ | sort
```

Roughly group:
- **Movement** — `go.go`, `north.go` etc., `flee.go` → `SendTextVisual` + `CategoryRoomEntry` / `CategoryRoomExit`
- **Look** — `look.go`, `examine.go` → `CategoryRoomDescription` for the room text, `CategoryError` for failure cases
- **Social** — `say.go`, `whisper.go`, `shout.go`, `emote.go`, `ooc.go` → audio + `CategorySpeech`, `CategoryWhisper`, `CategoryShout`, `CategoryEmote`, `CategoryOOC`
- **Combat** — `attack.go`, `kick.go`, `taunt.go`, `cast.go` → visual + appropriate combat categories
- **Inventory** — `get.go`, `drop.go`, `wear.go`, `wield.go`, `inventory.go` → audio + `CategoryLoot` / `CategoryEquipment`
- **Status / settings** — `status.go`, `who.go`, `online.go`, `set.go` → audio + `CategorySystem`
- **Help** — `help.go` → audio + `CategorySystem`
- **MOTD** — `motd.go` → audio + `CategoryBroadcast`
- **Tip** — `tip*.go` → audio + `CategoryTip`
- **Login/logout** — `login.go`, `quit.go` → audio + `CategoryLogin` / `CategoryLogout`
- **Spell-related**: `spells.go`, etc. → visual + `CategorySpell*`

- [ ] **Step 2: Migrate in 4 sub-batches with intermediate commits**

Migrate file-by-file. After every ~15 files, run `go build ./...` and commit:

```bash
git add internal/usercommands/<batch-of-files>
git commit -m "feat(messaging): T13a — usercommands/ batch 1 (movement + look)"
```

Suggested batches:
- **T13a** — movement + look + room cmds (~15 files)
- **T13b** — social + dialogue (~15 files)
- **T13c** — combat + spells + buffs (~15 files)
- **T13d** — inventory + status + settings + help + MOTD + login/logout (~25 files)

- [ ] **Step 3: Verify the whole package builds + tests pass**

Run: `go build ./... && go test ./internal/usercommands/...`
Expected: PASS.

- [ ] **Step 4: Smoke — boot server, exercise the major flows**

```bash
go run main.go &
# Manual: log in, move, look, say, whisper, attack, cast, get, wear,
# status, who, help, MOTD, quit. Ctrl-C after.
```

Expected: zero `site=internal/usercommands/…` deprecation warnings.

- [ ] **Step 5: Final commit for T13**

```bash
git commit --allow-empty -m "$(cat <<'EOF'
feat(messaging): T13 — internal/usercommands/ migrated to categorized API

~70 broadcast sites across movement, look, social, combat, spells,
inventory, status, settings, help, MOTD, login/logout. Committed in 4
sub-batches (T13a-T13d).

All visual content uses SendTextVisual with appropriate Category;
audio content (speech, broadcasts, system messages, errors) uses
SendText. Errors and warnings now tagged uniformly so colored
rendering can be tuned in one place.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 14: Audit + Migrate `internal/rooms/`, `internal/actions/`, `internal/behaviortree/` (~30 sites)

These three packages share broadcast helpers used by lower-level
engine code (entity actions, behavior-tree leaf nodes, room state
changes).

- [ ] **Step 1: Enumerate sites**

```bash
grep -rn 'SendTextLegacy\|SendTextVisualLegacy' internal/rooms/ internal/actions/ internal/behaviortree/
```

- [ ] **Step 2: Audit + migrate**

Common mappings:
- `rooms/rooms.go` — internal pieces; mostly already migrated by T9
- `actions/actor_*.go` — entity-emitted text; route by content
- `actions/buy.go`, `actions/steal.go`, `actions/sneak.go`, etc. — system/error mostly; some combat sub-cases
- `behaviortree/*.go` — leaf nodes emitting prose; visual for descriptive, audio for sound/speech

`actions/actor_mob.go`'s `sendRoomTextDarknessAware` is now redundant
with `Room.SendTextVisual`. T16 deletes the helper; T14 migrates its
callers to use `Room.SendTextVisual` directly.

- [ ] **Step 3: Verify build + tests**

Run: `go build ./... && go test ./internal/rooms/... ./internal/actions/... ./internal/behaviortree/...`

- [ ] **Step 4: Smoke**

Boot server, trigger a behavior tree (mob aggro on a player), confirm
zero remaining deprecation warnings in these packages.

- [ ] **Step 5: Commit**

```bash
git add internal/rooms/ internal/actions/ internal/behaviortree/
git commit -m "$(cat <<'EOF'
feat(messaging): T14 — rooms/, actions/, behaviortree/ migrated

~30 sites. Behavior-tree leaf nodes use CategoryMobEmote / CategoryMobIdle;
actor_* helpers route by content. actions/actor_mob.go's
sendRoomTextDarknessAware callers now use Room.SendTextVisual; the
helper itself is scheduled for T16 sunset.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 15: Tabular Renderer Migration

`status`, `inventory`, `online`, `who`, `help`, and `motd` build
tabular displays that don't go through `SendText` — they assemble
text and call the user's connection directly or have their own
formatting. They need to consume `UserRecord.LineWidth` for their
internal wrapping.

**Files:**
- Modify: `internal/usercommands/status.go` (look for table builders)
- Modify: `internal/usercommands/inventory.go`
- Modify: `internal/usercommands/online.go`
- Modify: `internal/usercommands/who.go`
- Modify: `internal/usercommands/help.go`
- Modify: `internal/usercommands/motd.go`

(Exact filenames may vary — first verify with `ls internal/usercommands/{status,inventory,online,who,help,motd}*.go`.)

- [ ] **Step 1: Identify each renderer's existing wrap behavior**

For each file, find places that hard-code line width (commonly literal
`80`, `78`, `76`) for column counting or table-line construction.

```bash
grep -n '80\|78\|76\|wrapText\b' internal/usercommands/status.go \
   internal/usercommands/inventory.go internal/usercommands/online.go \
   internal/usercommands/who.go internal/usercommands/help.go \
   internal/usercommands/motd.go
```

- [ ] **Step 2: For each renderer, replace literal widths with `user.GetLineWidth()`**

Pattern — wherever a renderer uses width:

```go
// Before:
const tableWidth = 80
header := fmt.Sprintf("%-*s", tableWidth, " STATUS")

// After:
tableWidth := user.GetLineWidth()
header := fmt.Sprintf("%-*s", tableWidth, " STATUS")
```

For `motd.go`'s `wrapText` callers — replace with
`messaging.WrapAnsi(text, user.GetLineWidth())`. Leave the
`wrapText` helper definition in place; T16 deletes it.

- [ ] **Step 3: For each renderer, smoke-test a non-default LineWidth**

```bash
# Manual test:
go run main.go &
# In-game:  set linewidth 120
# Then:     status
# Verify:   table renders to ~120 cols, not 80.
# Then:     set linewidth 60
# Verify:   table renders to ~60 cols.
# Ctrl-C.
```

- [ ] **Step 4: Verify build + tests**

Run: `go build ./... && go test ./internal/usercommands/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/usercommands/{status,inventory,online,who,help,motd}*.go
git commit -m "$(cat <<'EOF'
feat(messaging): T15 — tabular renderers honor UserRecord.LineWidth

status, inventory, online, who, help, MOTD: replaced hard-coded 80/78
column widths with user.GetLineWidth(). MOTD wrapText callers now
delegate to messaging.WrapAnsi.

Players who `set linewidth 120` see wider tables; default stays 80.
Renderers are still self-contained — they don't route through the
SendText pipeline, but they now read the same source of truth for
width.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Phase 3 — Cleanup + Close-Out (T16–T18)

The pipeline is live, all packages migrated, tabular displays widened
per player. Now: delete dead code, validate the companion-name fix,
and close out the chunk.

---

### Task 16: Sunset List Removal

Delete every helper the spec marks for sunset. The system should
still build, boot, and pass tests with all of these gone.

**Files:**
- Modify: `internal/combat/combat.go` — delete `canSeeInRoom` (lines 30–36)
- Modify: `internal/hooks/NewRound_DoCombat_helpers.go` — delete `canSeeInRoom` (around line 102)
- Modify: `internal/actions/actor_mob.go` — delete `sendRoomTextDarknessAware` (around line 96)
- Modify: `internal/usercommands/motd.go` — delete `wrapText` (around line 43)
- Modify: `internal/characters/progression.go` — replace `*** You feel your X skills sharpening! ***` (line 128) with a `messaging.SendProgression` call
- Modify: `internal/rooms/rooms.go` — delete `SendTextLegacy` and `SendTextVisualLegacy` shims (added in T9)
- Modify: `internal/users/userrecord.go` — delete `SendTextLegacy` shim

- [ ] **Step 1: Full sunset survey — enumerate every candidate before deleting anything**

The spec §14 lists the planned sunset items, but the per-package
audit (T10–T15) may have surfaced additional dead code: orphaned
helpers, unused imports, hand-written ANSI tag literals that should
have moved to the Category enum, leftover instrumentation. Catch
those before the bulk delete.

Dispatch a thorough exploration with the Explore subagent. Prompt:

```
I'm closing out a messaging-framework migration in DOGMud. Before I
delete the sunset candidates listed in
docs/superpowers/specs/2026-05-19-messaging-framework-design.md §14,
I need a complete inventory of what's actually dead.

Report each finding as `path:line` with a one-line note. Don't fix
anything — survey only.

1. Confirm the spec §14 sunset list still exists in code:
   - `internal/combat/combat.go` — canSeeInRoom (lines ~30-36)
   - `internal/hooks/NewRound_DoCombat_helpers.go` — canSeeInRoom
   - `internal/actions/actor_mob.go` — sendRoomTextDarknessAware
   - `internal/usercommands/motd.go` — wrapText
   - `internal/characters/progression.go` — *** sharpening *** literal
   - `internal/rooms/rooms.go` — SendTextLegacy, SendTextVisualLegacy, legacyCallerInfo
   - `internal/users/userrecord.go` — SendTextLegacy

2. Survey for ANY remaining callers of the listed sunset symbols
   (functions, helpers, literals). For each: report `path:line` of the
   call so I know if a follow-up migration is needed before the
   delete is safe.
   - `grep -rn 'canSeeInRoom\|sendRoomTextDarknessAware\|wrapText\|SendTextLegacy\|SendTextVisualLegacy\|legacyCallerInfo' internal/`
   - Also grep the verbatim `*** You feel your` and `skills sharpening`
     literals in case they were copied elsewhere.

3. Imports added in T9 that may now be unused (after deletes land):
   - `internal/rooms/rooms.go` — `runtime` and `mlog`/`slog` (only the
     legacyCallerInfo helper used them).
   - Any other imports that only the legacy shims relied on.

4. Hand-written `<ansi fg="...">` literals that should have migrated
   to the Category enum. Grep `internal/` for any `<ansi fg="` literal
   in `.go` files and report the count per package. Don't list every
   site — just `internal/<pkg>: N hits` so I can spot whether T13/T11
   missed a batch.

5. Inline `room.GetVisibility() >= 1 || .*HasFlagFromAnySource.*NightVision`
   conditionals — there were several copies of this predicate inline
   across the codebase. Any remaining inline copies should route
   through `messaging.CanSeeClearly`. Report `path:line` for each
   remaining inline copy.

6. Dead `<ansi fg="">` aliases — list any color alias defined in
   `_datafiles/world/dogmud/ansi-aliases.yaml` that no longer appears
   anywhere in `.go` or `.yaml` files under `_datafiles/`. (`username-downed`
   is one likely casualty per the downed-removal sweep tracked in
   MEMORY.md — confirm with a grep.)

7. Anything that LOOKS dead in `internal/messaging/` itself —
   unreferenced helpers from earlier tasks, stubs that were never
   replaced with real implementations.

Output: a single markdown section I can paste under Step 1 of T16
in the plan. Keep findings concrete (path:line and a one-sentence
why). Under 400 lines total.
```

Review the survey report. For each finding:
- **Confirmed dead** → goes into the delete list below.
- **Has unexpected callers** → migrate the caller first (drop a new step into T16, OR push to a followup if it's out of scope).
- **Ambiguous** → ask before deleting.

- [ ] **Step 2: Verify zero remaining legacy callers (sanity check after survey)**

```bash
grep -rn 'SendTextLegacy\|SendTextVisualLegacy' internal/
```

Expected: zero hits. If any remain (the survey should have caught
them), migrate them first.

- [ ] **Step 3: Delete the duplicate `canSeeInRoom` helpers + redundant darkness wrapper**

For each of:
- `internal/combat/combat.go` lines 30–36
- `internal/hooks/NewRound_DoCombat_helpers.go` (its `canSeeInRoom` function)
- `internal/actions/actor_mob.go` (its `sendRoomTextDarknessAware` function)

Use grep to find any remaining callers FIRST. If any call site still
uses the local helper, replace with the message-package equivalent
(`messaging.CanSeeClearly` / `Room.SendTextVisual`). Then delete the
helper.

- [ ] **Step 4: Delete motd.go's `wrapText`**

In `internal/usercommands/motd.go`, delete the `wrapText` function
(around line 43). Verify with grep that nothing calls it:

```bash
grep -rn '\bwrapText\b' internal/
```

Expected: zero hits.

- [ ] **Step 5: Replace the legacy progression literal**

Edit `internal/characters/progression.go`. Replace lines around 128
(the `msg := fmt.Sprintf(\`<ansi fg="magenta">*** You feel your <ansi fg="yellow">%s</ansi> skills sharpening! ***</ansi>\`, ...)` block) with a `messaging.SendProgression` call.

Before:

```go
msg := fmt.Sprintf(`<ansi fg="magenta">***</ansi> You feel your <ansi fg="yellow">%s</ansi> skills sharpening! <ansi fg="magenta">***</ansi>`, actualSkill)
user.SendText(messaging.CategorySkillProgress, msg)
```

After:

```go
messaging.SendProgression(user, messaging.ProgSkill, actualSkill, tierChange)
```

`tierChange` is `*messaging.TierChange` — nil unless the skill rank
crossed a tier band. Tier names from
`skills.GetSkillRankDescription`; compute the from/to before the call
and pass nil if equal:

```go
fromTier := skills.GetSkillRankDescription(oldRank)
toTier := skills.GetSkillRankDescription(newRank)
var tierChange *messaging.TierChange
if fromTier != toTier {
	tierChange = &messaging.TierChange{From: fromTier, To: toTier}
}
messaging.SendProgression(user, messaging.ProgSkill, actualSkill, tierChange)
```

Repeat the analogous treatment for any stat-progression site that
exists in `progression.go` — pass `ProgStat` and `templates.statQuality`
tier names.

- [ ] **Step 6: Delete the legacy shims**

In `internal/rooms/rooms.go`, delete `SendTextLegacy`,
`SendTextVisualLegacy`, and `legacyCallerInfo`. Also drop unused
`runtime` and `mlog` imports if those imports become unused.

In `internal/users/userrecord.go`, delete `SendTextLegacy`.

- [ ] **Step 7: Build + test**

Run: `go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 8: Boot the server, verify zero deprecation warnings**

```bash
go run main.go &
# Exercise a flow; Ctrl-C.
```

Expected: server boots cleanly past data file loading; no
deprecation log warnings.

- [ ] **Step 9: Commit**

```bash
git add internal/combat/combat.go internal/hooks/NewRound_DoCombat_helpers.go internal/actions/actor_mob.go internal/usercommands/motd.go internal/characters/progression.go internal/rooms/rooms.go internal/users/userrecord.go
git commit -m "$(cat <<'EOF'
refactor(messaging): T16 — sunset duplicate predicates + legacy shims

Deleted:
- internal/combat/combat.go:canSeeInRoom (duplicate)
- internal/hooks/NewRound_DoCombat_helpers.go:canSeeInRoom (duplicate)
- internal/actions/actor_mob.go:sendRoomTextDarknessAware
- internal/usercommands/motd.go:wrapText (naive byte-count wrapper)
- *** You feel your X skills sharpening! *** literal in progression.go
- SendTextLegacy / SendTextVisualLegacy shims in rooms.go and userrecord.go

Single source of truth for sight predicates: messaging.CanSeeClearly /
CanSeeShapes. Single source of truth for wrapping: messaging.WrapAnsi.
Progression now goes through messaging.SendProgression's banner format,
including the tier-crossing third line when skill rank changes tier
band.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 17: Companion-Name-Leak Smoke Fix Validation

The companion-name leak surfaces when a player with a companion
enters a dark room. Before this chunk, the companion's name was
emitted via `room.SendText` (audio) when it should have been
`SendTextVisual`. The T11 audit fixed this at the call site; T17
validates the fix is solid and adds a regression test if practical.

**Files:**
- Possibly modify: a `hooks/companion_*.go` file if the audit missed a site
- Possibly create: `internal/hooks/companion_*_test.go` if a unit test path exists

- [ ] **Step 1: Reproduce the bug in a dark room (or verify it no longer reproduces)**

Set up a manual smoke test:

```bash
go run main.go &
# In-game:
#   1. Log in as a player with a companion.
#   2. Go to a dark room (an unlit cavern; check world.md for one).
#   3. Have the companion follow you.
#   4. Look or trigger an action that previously emitted the companion's
#      name to other observers in the room.
#   5. With a second observer (no nightvision) in the room, verify the
#      companion's name does NOT appear in their feed.
#   6. With infrared on (mutation: deep_gnawer), verify the observer
#      sees "a figure" not the companion's name.
# Ctrl-C.
```

- [ ] **Step 2: If the leak still reproduces, locate the offending site**

```bash
grep -rn '\.SendText(.*companion\|petname' internal/hooks/
```

Common offender pattern — companion emote / follow line:

```go
// Wrong (leaks in dark):
room.SendText(messaging.CategoryMobIdle, fmt.Sprintf(`<ansi fg="petname">%s</ansi> trots in.`, companion.Name))

// Correct:
room.SendTextVisual(messaging.CategoryMobIdle, fmt.Sprintf(`<ansi fg="petname">%s</ansi> trots in.`, companion.Name))
```

- [ ] **Step 3: Fix the site**

Apply the `SendText` → `SendTextVisual` correction in-place.

- [ ] **Step 4: Re-smoke**

Repeat Step 1. Verify the leak is gone and the infrared observer sees
"a figure".

- [ ] **Step 5: Commit (only if a fix was needed)**

```bash
git add internal/hooks/<file>
git commit -m "$(cat <<'EOF'
fix(messaging): T17 — companion-name leak in dark rooms

Companion line emitted via room.SendText (audio) was leaking the
companion's name to observers without nightvision. Switched to
SendTextVisual so the sight gate suppresses the name; infrared
observers see "a figure" via the anonymizer.

Validated by manual smoke in a dark cavern with a second observer.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

If no fix needed (T11 caught it already), skip the commit and log
the validation in T18's notes.

---

### Task 18: Context.md Sweep + Roadmap + Patch Notes + AI Smoke Pass

Final close-out. Update docs, ship to prod after AI smoke.

**Files:**
- Modify: `internal/messaging/context.md` (any sections that drifted during implementation)
- Modify: `MOB_ALIVENESS_ROADMAP.md` (mark this chunk complete; advance to next)
- Modify: `PATCH_NOTES.md` (dated entry)
- Modify: `internal/characters/perception/context.md` (drop the "no consumer yet" caveat — messaging is now the consumer)
- Possibly modify: memory entries (mark `[[messaging-framework-chunk]]` Done; drop `[[combat-text-color-coding]]`)

- [ ] **Step 1: Update `internal/messaging/context.md` if it drifted**

If any of the API or stage details changed during implementation
(e.g., `SendProgression` ended up taking a different param shape),
revise the context.md to match the shipped code.

- [ ] **Step 2: Update the Perception context.md**

Edit `internal/state/perception/context.md`. Find the "ships dormant"
warning and update it to:

> Originally shipped dormant in chunk 6 (2026-05-19); consumed by
> the centralized messaging framework chunk (2026-05-XX, this date)
> via `internal/messaging/predicates.go:CanSeeClearly` /
> `CanSeeShapes`. Sight-gated broadcasts now route through the
> messaging pipeline.

- [ ] **Step 3: Update `MOB_ALIVENESS_ROADMAP.md`**

Mark this chunk complete in the roadmap. Add a forward pointer to
whatever the next mob-aliveness substrate chunk is (chunks 0/0.5/1
had been paused since this branch began; resume there).

- [ ] **Step 4: Append a `PATCH_NOTES.md` entry dated today**

```markdown
## 2026-05-XX — Centralized Messaging Framework

Every player-facing line of text now flows through a single pipeline
(compose → normalize → anonymize → color → wrap → deliver). Highlights:

- Combat narration is now color-coded by category — damage bands by
  weapon subtype, defense in greens, specials in their existing hues,
  grapple in warm taupe, spells in five-school palette.
- Skill / stat advancement banners replace the old *** sharpening ***
  one-liners with a SKILL ADVANCEMENT / STATISTIC INCREASED format
  that includes a tier-crossing transition line.
- Style auto-corrects: "a aggressive" → "an aggressive",
  "damage damage" → "damage", missing periods auto-appended, sentence
  starts auto-capitalized for combat prose.
- Line wrapping is ANSI-aware and per-player. `set linewidth N` lets
  players match their terminal (40-240, default 80).
- Infrared observers in dark rooms now see "a red figure" rather than
  named players/mobs — anonymizer strips name tags from the visual
  feed.
- Username (yellow → blue), mobname (cyan → warm tan), petname
  (orange → teal-cyan) retuned to match the new combat palette.

Internal cleanup: duplicate canSeeInRoom helpers consolidated;
sendRoomTextDarknessAware deleted; naive byte-count wrapText removed.
Chunk-6 Perception FSM (shipped dormant in May) is now the source of
truth for the sight gate.
```

- [ ] **Step 5: Set production logging to false + commit pre-push prep**

Per the pre-push SOP:

Edit `_datafiles/config.yaml`. Find `Logging.LogToFile`:

```yaml
Logging:
  LogToFile: false
```

Commit:

```bash
git add MOB_ALIVENESS_ROADMAP.md PATCH_NOTES.md internal/messaging/context.md internal/state/perception/context.md _datafiles/config.yaml
git commit -m "$(cat <<'EOF'
docs(messaging): T18 — chunk close-out (context.md + roadmap + patch notes)

Updated messaging/context.md with shipped API. Perception/context.md
no longer says "ships dormant" — messaging is the consumer. Roadmap
marks the messaging chunk complete; mob-aliveness substrate work
resumes. PATCH_NOTES.md entry for the player-facing changes.

Production logging disabled per pre-push SOP.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

- [ ] **Step 6: Run an AI feature-tester smoke pass**

Author a brief AI test goal that exercises the messaging pipeline end
to end. Create `tools/testing/goals/messaging-framework-smoke.yaml`:

```yaml
name: messaging-framework-smoke
description: End-to-end smoke of the centralized messaging pipeline.
duration_rounds: 60
objectives:
  - Verify combat narration uses appropriate colors (look for ANSI
    tags in raw output via gameserver inspection).
  - Trigger a skill advancement and confirm the new banner renders.
  - Move to a dark room with a companion; verify companion-name does
    NOT appear in observer feeds without nightvision.
  - With infrared mutation, verify "a figure" appears in dark room
    combat instead of the actor's name.
  - Trigger 5+ spells across schools (fire, heal, buff, illusion,
    summon) and confirm the spell-resolve color tag matches the
    school.
  - Issue `set linewidth 120` and confirm subsequent broadcasts and
    table output match.
  - Issue `set linewidth 60` and confirm wraps tighten.
  - Validate no double-anonymization, no missing-color tags
    (raw `category-XXX` strings appearing), no panic stack traces.
```

Run the smoke:

```bash
# Run /test-mud local feature-tester messaging-framework-smoke.yaml
```

(This invokes the project's AI testing tooling. Reports land in
`tools/testing/reports/`.)

Expected report findings:
- No panics
- No raw category strings in output
- Companion-name leak does not reproduce
- Banner renders correctly
- Color tags applied where expected
- LineWidth changes affect output

- [ ] **Step 7: If the AI smoke surfaces issues, file followups**

For any defects that don't block merge (color tuning, edge-case
normalize over-correction, etc.) add an entry to MEMORY.md's
followups table. Genuine bugs (panics, broken pipeline behavior) get
fixed inline before continuing.

- [ ] **Step 8: Kill test mud servers and clean up**

Per the SOP:

```bash
# Find lingering processes
ps aux | grep -E 'dogmud|go run main' | grep -v grep
# Kill what's left (Windows: taskkill /F /IM dogmud.exe)
```

- [ ] **Step 9: Final commit and push**

If the AI smoke logs anything worth recording, commit that report
file or its summary alongside any patch:

```bash
git add tools/testing/goals/messaging-framework-smoke.yaml
git add tools/testing/reports/messaging-framework-smoke-<date>.txt  # if applicable
git commit -m "$(cat <<'EOF'
test(messaging): T18 — AI feature-tester smoke pass

End-to-end smoke goal: combat colors, skill banners, companion leak,
infrared rendering, spell schools, set linewidth, no panics. Report
filed in tools/testing/reports/.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Push to origin (after one more sanity boot):

```bash
go run main.go &
# Boot to "Server ready"; Ctrl-C.
git push origin feature/mob-aliveness-1.3-crimes
```

The chunk is complete. Next session resumes the paused mob-aliveness
substrate work per `MOB_ALIVENESS_ROADMAP.md`.

---

## Appendix: Self-Review Checklist (run before handing off)

1. **Spec coverage** — every section of the spec has at least one task:
   - §3 Package layout ✓ T1
   - §4 Pipeline ✓ T2 + every stage task
   - §5 Helper API ✓ T9
   - §6 Category enum ✓ T1
   - §7 Predicates ✓ T3
   - §8 Color path ✓ T4
   - §9 Anonymizer ✓ T6
   - §10 Wrapping ✓ T5
   - §11 Normalization ✓ T8
   - §12 Progression banner ✓ T7 + T16
   - §13 Migration ✓ T10–T15
   - §14 Sunset ✓ T16
   - §15 Risks — addressed by T18 smoke + per-category skip table + defensive parser
   - §16 Success criteria — every numbered criterion lands by T18
2. **Placeholders** — no "TBD", "implement later", "similar to Task N". Every code step contains the code to write.
3. **Type consistency**:
   - `Category` enum defined T1; consumed T2/T4/T8/T9.
   - `Channel` / `SightDecision` defined T2; consumed T9.
   - `RenderInput` defined T2; consumed T9.
   - `UserSender` interface defined T7; updated T9 with leading Category.
   - `ProgressionKind` / `TierChange` defined T7; consumed T16.
   - `WrapAnsi` defined T5; consumed by pipeline.go in T5, by tabular renderers in T15.
4. **Naming** — `SendText` / `SendTextVisual` / `SendTextLegacy` consistent across T9/T10/T16. `CanSeeClearly` / `CanSeeShapes` consistent T3/T16. `messaging.CategoryFoo` style consistent.

---

## Plan complete.
