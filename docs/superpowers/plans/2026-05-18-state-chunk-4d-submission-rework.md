# Combat State — Chunk 4d: Submission Rework Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the legacy player-typed `submit` special-attack with a symmetric, opportunistic, per-round submission system that piggybacks on the chunk-4b control-axis drift roll. When either side of a grapple pair wins the drift roll by a margin exceeding an alpha threshold (or the defender crits their defense roll), a submission window opens; a separate sub-roll resolves into one of four tiers (bad / neutral / success / crit). Successful subs apply the attempter's pre-set `SubmissionPolicy` (mercy / subdue / cripple / lethal) — defender's `SurrenderPolicy` is honored only by mercy controllers, matching realism. Subdue + cripple reuse the existing Life FSM death cascade with a new `NoDeprogression` flag, so the defender wakes up at the temple without stat decay (cripple additionally applies a duration-based broken-limb buff). Sunset the legacy command + `AttemptSubmission` / `ApplySubmissionSuccess` / `ApplySubmissionFailure` helpers at the end.

**Architecture:** One new per-round observer (`Position_SubmissionTick.go`) that runs immediately after the chunk-4b drift tick, reads the (margin, z-score) the drift tick stashed per side, and calls into the new `internal/combat/submission.go` package for roll + outcome resolution. Two new policy enums (`SubmissionPolicy`, `SurrenderPolicy`) on Character + per-archetype defaults. New position → submission mapping in `internal/state/position/submissions.go` split into top-attack and bottom-attack subs per position. Two new buffs (broken-limb 900-round duration, submission-stunned 1-round). `NoDeprogression` + `GoldLossFraction` fields added to `life.DeadData` with two flag-guarded branches in the existing `Death_PlayerCleanup` + `Death_PlayerCorpse` observers. Hard replacement (not parallel-write) of the legacy command — sunset is its own task.

**Tech Stack:** Go 1.21+, existing chunk-0 state framework, existing `internal/state/position/` (4a) + control axis (4b) + reach utility (4c), existing buff system, existing Life cascade (chunk 2).

**Spec:** `docs/superpowers/specs/2026-05-18-state-chunk-4d-submission-rework-design.md`

**Branch:** Stay on `feature/mob-aliveness-1.3-crimes` per the single-feature-branch SOP.

**Doc scope:** Comprehensive per user SOP. T19 surveys helpfiles + context.md files for stale references to the legacy `submit` command + new doc surface needed for submission policies / broken-limb buff / position-sub mapping. T20 applies helpfile updates. T21 applies context.md updates. The doc surface is wider than chunk 4c because 4d ships brand-new player-facing concepts (policies, broken-limb debuff) AND removes a legacy command.

---

## File structure

| File | Status | Responsibility |
|------|--------|----------------|
| `internal/state/position/submissions.go` | NEW (T1) | `SubmissionType` enum (Armbar / RNC / Triangle / Americana / Kimura / Omoplata / Anaconda); `TopSubmissionsForPosition(s State) []SubmissionType` + `BottomSubmissionsForPosition(s State) []SubmissionType`; `IsTopSubEligible(s State, cl ControlLevel) bool` + `IsBottomSubEligible(s State, cl ControlLevel) bool` predicates; `CrippleBodyPart(t SubmissionType) string` (returns "arm" / "shoulder" / "" for chokes); `SubmissionToVerb(t SubmissionType) string` for narration |
| `internal/state/position/submissions_test.go` | NEW (T1) | Mapping coverage tests across all 14 states × all ControlLevels for both roles |
| `internal/characters/submission_policy.go` | NEW (T2) | `SubmissionPolicy` enum (PolicyMercy / PolicySubdue / PolicyCripple / PolicyLethal); `SurrenderPolicy` struct (Mode enum + HpPctThreshold); `Character.SubmissionPolicy` + `Character.SurrenderPolicy` fields w/ yaml tags + accessor methods; `ParseSubmissionPolicy(s string)` + `ParseSurrenderPolicy(s string)` for `set` command parsing; `DefaultSubmissionPolicyForArchetype(archetype string)` lookup; per-character "last submission attempted" round-robin tracker for varied narration |
| `internal/characters/submission_policy_test.go` | NEW (T2) | Enum parsing, default-by-archetype lookup, round-robin tracker reset on respawn |
| `internal/configs/config.balance.go` | MODIFY (T3) | Add `SubmissionAttemptAlpha`, `SubmissionAttemptCritZ`, `SubSkillWeight`, `SubBadZThreshold`, `SubCritZThreshold`, `SubGoldLossFraction`, `BrokenLimbBuffDuration` knobs with defaults |
| `_datafiles/config.yaml` | MODIFY (T3) | Surface the 7 new knobs with comments |
| `internal/configs/config.balance.combat.go` | MODIFY (T3) | Validation defaults for the 7 new knobs in `validateCombat()` |
| `internal/combat/submission.go` | NEW (T4) | `SubmissionAttemptResult` struct (Tier enum + Type + Margin + ZScore); `RollSubmissionAttempt(attempter, recipient *characters.Character, attemptType SubmissionType) SubmissionAttemptResult` |
| `internal/combat/submission_test.go` | NEW (T4) | Tier boundary tests (bad / neutral / success / crit), Strength + UnarmedCombat weighting verification |
| `internal/hooks/Position_GrappleTick.go` | MODIFY (T5) | Stash per-side drift roll (`margin`, `zScore`) on `Character.LastDriftRoll` so Position_SubmissionTick can read this round's drift outcome |
| `internal/characters/character.go` | MODIFY (T5) | Add `LastDriftRoll DriftRollSnapshot` (yaml:"-") field on Character; `DriftRollSnapshot{Round uint64, MarginAttacker float64, ZScoreAttacker float64, MarginDefender float64, ZScoreDefender float64}` struct |
| `internal/hooks/Position_SubmissionTick.go` | NEW (T6) | Per-round observer fired AFTER Position_GrappleTick (event listener priority); iterates active grapple pairs; for each pair, checks both sides for sub-attempt eligibility using `LastDriftRoll` + position eligibility; calls `RollSubmissionAttempt` + `ResolveSubmissionOutcome` on success |
| `internal/hooks/Position_SubmissionTick_test.go` | NEW (T6) | Integration tests: top-attack eligibility, bottom-attack eligibility, crit-shortcut, tier resolution for each policy |
| `internal/combat/submission_outcome.go` | NEW (T7) | `ResolveSubmissionOutcome(attempter, recipient *characters.Character, subType SubmissionType, attemptResult SubmissionAttemptResult, role Role)` — applies the attempter's policy + emits messaging; bad tier knocks attempter Prone; success tier branches into mercy / subdue / cripple / lethal cleanup |
| `internal/combat/submission_outcome_test.go` | NEW (T7) | Policy resolution matrix tests (mercy + tap honored / mercy + no-tap / subdue / cripple-with-arm-sub / cripple-degraded-to-subdue-for-choke / lethal) |
| `internal/state/life/life.go` | MODIFY (T8) | Add `NoDeprogression bool` + `GoldLossFraction float64` fields to `DeadData` |
| `internal/hooks/Death_PlayerCleanup.go` | MODIFY (T8) | Read `DeadData.NoDeprogression`; skip the stat-decay step when true |
| `internal/hooks/Death_PlayerCorpse.go` | MODIFY (T8) | Read `DeadData.GoldLossFraction`; when > 0 transfer that fraction of gold from defender to Killer and skip full corpse creation |
| `internal/hooks/Death_PlayerCleanup_test.go` (or sibling) | MODIFY (T8) | Tests for the no-deprogression branch + partial-gold transfer |
| `internal/buffs/buffspec.go` | (read-only; verify buff loader spec) | n/a |
| `_datafiles/world/dogmud/buffs/83-broken_limb.yaml` | NEW (T9) | Broken-limb buff YAML: -25% accuracy stat-mod, -10% defense, -5% stamina max, duration 900 rounds, persists across rest |
| `_datafiles/world/dogmud/buffs/84-submission_stunned.yaml` | NEW (T10) | 1-round stun buff: cannot attack, defense penalty |
| `_datafiles/world/dogmud/grapple-messages.yaml` | MODIFY (T11) | Add per-tier submission messages (opening / escape-bad / neutral / success-per-policy / crit-flag) for each `SubmissionType` |
| `internal/hooks/Position_Messaging.go` | MODIFY (T11) | Add `fireSubmissionAttemptMessage` / `fireSubmissionResolutionMessage` helpers that load from the new YAML and broadcast to attempter / recipient / room |
| `internal/mobs/mobs.go` | MODIFY (T12) | Add `MobSpec.SubmissionPolicy` + `MobSpec.SurrenderPolicy` yaml fields; `newMobByIdInternal` copies them to the spawned Character with archetype-default fallback |
| `internal/mobs/mobs_test.go` | MODIFY (T12) | Test YAML round-trip + archetype-default fallback |
| `internal/behaviortree/conditions_submission.go` | NEW (T13) | `mob_can_submit_top` (controller in sub-eligible pos w/ InControl), `mob_can_submit_bottom` (controlled side w/ bottom-attack subs available at position), `mob_submission_policy_is <policy>` |
| `internal/behaviortree/conditions_submission_test.go` | NEW (T13) | Per-primitive tests |
| `_datafiles/world/dogmud/mobs/**/*.yaml` | MODIFY (T14) | Selective overrides on bosses (cripple/lethal), civilians (mercy + always-tap), guards (subdue + never-tap). Most mobs inherit archetype default. |
| `internal/usercommands/set.go` | MODIFY (T15) | Add `set submission <mode>` + `set surrender <mode>` subcommands; first-time lethal confirmation prompt |
| `internal/usercommands/set_test.go` (if exists, else new) | MODIFY (T15) | Tests for the two new subcommands |
| `internal/usercommands/status.go` | MODIFY (T16) | Add lines for current submission/surrender policy + broken-limb buff with remaining duration |
| `_datafiles/world/dogmud/templates/help/submission.template` | NEW (T17) | Player-facing helpfile: policy ladder, defender-side surrender, narration model |
| `_datafiles/world/dogmud/templates/help/surrender.template` | NEW (T17) | Defender-side policy explainer |
| `_datafiles/world/dogmud/templates/help/submit.template` | DELETE (T18) | Legacy helpfile |
| `_datafiles/world/dogmud/templates/help/submit.md` (if exists) | DELETE (T18) | Legacy helpfile (md variant) |
| `internal/usercommands/submit.go` | DELETE (T18) | Legacy command |
| `internal/mobcommands/submit.go` | DELETE (T18) | Legacy mob command |
| `internal/usercommands/usercommands.go` | MODIFY (T18) | Remove `submit` from command registry |
| `internal/mobcommands/mobcommands.go` | MODIFY (T18) | Same |
| `internal/combat/grapple.go` | MODIFY (T18) | DELETE `AttemptSubmission`, `ApplySubmissionSuccess`, `ApplySubmissionFailure`, `SubmissionResult` |
| Behavior Matrix tests | NEW (T19) | PB-301..PB-341 across the various test files |
| `tools/testing/audits/2026-05-18-chunk-4d-doc-helpfile-audit.md` | NEW (T20) | Doc audit deliverable |
| Helpfiles + context.md (per audit) | MODIFY (T21) | Apply audit findings |
| `COMBAT_STATE_ROADMAP.md` | MODIFY (T22) | Mark chunk 4d Done; add "Chunk 4d — Shipped" section |

---

## Task 1: Position → submission mapping + role-split eligibility

**Files:**
- Create: `internal/state/position/submissions.go`
- Create: `internal/state/position/submissions_test.go`

Pure-data layer. No behavior — just enums + lookup tables + predicates. Foundation for everything downstream. The split-by-role (top vs bottom) is the key new structure compared to legacy.

- [ ] **Step 1: Write the failing test**

Create `internal/state/position/submissions_test.go`:

```go
package position_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/stretchr/testify/assert"
)

func TestTopSubmissionsForPosition_CoreMappings(t *testing.T) {
	cases := map[position.State][]position.SubmissionType{
		position.Standing:     nil,
		position.Prone:        nil,
		position.Supine:       nil,
		position.Turtle:       nil,
		position.Clinch:       nil,
		position.BackStanding: {position.SubRNC},
		position.Mount:        {position.SubAmericana, position.SubTriangle, position.SubArmbar},
		position.SideControl:  {position.SubKimura, position.SubAmericana},
		position.KneeOnBelly:  {position.SubArmbar},
		position.NorthSouth:   {position.SubKimura, position.SubAnaconda},
		position.Crucifix:     {position.SubArmbar},
		position.BackGround:   {position.SubRNC},
		position.HalfGuard:    {position.SubKimura},
		position.Guard:        {position.SubTriangle, position.SubArmbar, position.SubOmoplata},
	}
	for state, want := range cases {
		got := position.TopSubmissionsForPosition(state)
		assert.Equal(t, want, got, "TopSubmissionsForPosition(%s)", state)
	}
}

func TestBottomSubmissionsForPosition_CoreMappings(t *testing.T) {
	cases := map[position.State][]position.SubmissionType{
		position.Standing:     nil,
		position.Prone:        nil,
		position.Supine:       nil,
		position.Turtle:       nil,
		position.Clinch:       nil,
		position.BackStanding: nil,
		position.Mount:        {position.SubTriangle, position.SubArmbar},
		position.SideControl:  {position.SubKimura},
		position.KneeOnBelly:  nil,
		position.NorthSouth:   {position.SubKimura},
		position.Crucifix:     nil,
		position.BackGround:   nil,
		position.HalfGuard:    {position.SubKimura},
		position.Guard:        nil,
		position.Turtle:       nil,
	}
	for state, want := range cases {
		got := position.BottomSubmissionsForPosition(state)
		assert.Equal(t, want, got, "BottomSubmissionsForPosition(%s)", state)
	}
}

func TestIsTopSubEligible(t *testing.T) {
	assert.True(t, position.IsTopSubEligible(position.Mount, position.InControl))
	assert.True(t, position.IsTopSubEligible(position.Mount, position.LosingControl))
	assert.False(t, position.IsTopSubEligible(position.Mount, position.Neutral))
	assert.False(t, position.IsTopSubEligible(position.Mount, position.BecomingControlled))
	assert.False(t, position.IsTopSubEligible(position.Mount, position.Controlled))
	assert.False(t, position.IsTopSubEligible(position.Standing, position.InControl), "no top subs at Standing")
}

func TestIsBottomSubEligible(t *testing.T) {
	assert.True(t, position.IsBottomSubEligible(position.Mount, position.Controlled))
	assert.True(t, position.IsBottomSubEligible(position.Mount, position.BecomingControlled))
	assert.False(t, position.IsBottomSubEligible(position.Mount, position.Neutral))
	assert.False(t, position.IsBottomSubEligible(position.Mount, position.InControl))
	assert.False(t, position.IsBottomSubEligible(position.KneeOnBelly, position.Controlled), "no bottom subs at KOB")
}

func TestCrippleBodyPart(t *testing.T) {
	assert.Equal(t, "arm", position.CrippleBodyPart(position.SubArmbar))
	assert.Equal(t, "arm", position.CrippleBodyPart(position.SubOmoplata))
	assert.Equal(t, "shoulder", position.CrippleBodyPart(position.SubKimura))
	assert.Equal(t, "shoulder", position.CrippleBodyPart(position.SubAmericana))
	// Chokes have no body part — they don't break limbs
	assert.Equal(t, "", position.CrippleBodyPart(position.SubRNC))
	assert.Equal(t, "", position.CrippleBodyPart(position.SubTriangle))
	assert.Equal(t, "", position.CrippleBodyPart(position.SubAnaconda))
}
```

- [ ] **Step 2: Run test to verify it fails**

```
go test ./internal/state/position/... -run TestTopSubmissions
```
Expected: FAIL with "undefined: position.SubmissionType" (or similar).

- [ ] **Step 3: Implement `submissions.go`**

Create `internal/state/position/submissions.go`:

```go
package position

// SubmissionType enumerates the named submissions available in
// chunk 4d. Each position-state has 0..N submissions available to
// each role (top-attack from the controller side, bottom-attack from
// the controlled side); the per-round submission tick picks which to
// attempt via a round-robin tracker on Character.
type SubmissionType int

const (
	SubNone SubmissionType = iota
	SubArmbar
	SubRNC // Rear Naked Choke
	SubTriangle
	SubAmericana
	SubKimura
	SubOmoplata
	SubAnaconda
)

func (t SubmissionType) String() string {
	switch t {
	case SubArmbar:
		return "armbar"
	case SubRNC:
		return "rear-naked choke"
	case SubTriangle:
		return "triangle choke"
	case SubAmericana:
		return "americana"
	case SubKimura:
		return "kimura"
	case SubOmoplata:
		return "omoplata"
	case SubAnaconda:
		return "anaconda choke"
	default:
		return "submission"
	}
}

// TopSubmissionsForPosition returns the canonical top-attack subs
// available to the InControl side of the given position. Nil for
// positions with no top-attack subs (Standing, Prone, Supine,
// Turtle, Clinch).
func TopSubmissionsForPosition(s State) []SubmissionType {
	switch s {
	case BackStanding:
		return []SubmissionType{SubRNC}
	case Mount:
		return []SubmissionType{SubAmericana, SubTriangle, SubArmbar}
	case SideControl:
		return []SubmissionType{SubKimura, SubAmericana}
	case KneeOnBelly:
		return []SubmissionType{SubArmbar}
	case NorthSouth:
		return []SubmissionType{SubKimura, SubAnaconda}
	case Crucifix:
		return []SubmissionType{SubArmbar}
	case BackGround:
		return []SubmissionType{SubRNC}
	case HalfGuard:
		return []SubmissionType{SubKimura}
	case Guard:
		// In the FSM the Guard-bottom is the controller; the
		// "top-attack" subs from Guard are the bottom-game subs
		// (Triangle / Armbar / Omoplata).
		return []SubmissionType{SubTriangle, SubArmbar, SubOmoplata}
	default:
		return nil
	}
}

// BottomSubmissionsForPosition returns the canonical bottom-attack
// subs available to the Controlled side of the given position. The
// "reversal sub" set — sparser than top-attack because being
// controlled is supposed to be disadvantageous.
func BottomSubmissionsForPosition(s State) []SubmissionType {
	switch s {
	case Mount:
		// Mount-bottom: hipping up to catch the attacking arm.
		return []SubmissionType{SubTriangle, SubArmbar}
	case SideControl:
		// SideControl-bottom: snatching the wrist for a kimura.
		return []SubmissionType{SubKimura}
	case NorthSouth:
		return []SubmissionType{SubKimura}
	case HalfGuard:
		// HalfGuard-bottom: kimura on the trapped arm.
		return []SubmissionType{SubKimura}
	default:
		return nil
	}
}

// IsTopSubEligible reports whether the controller side of a pair
// can attempt a submission at the given position + control level.
// Requires top-attack subs available AND ControlLevel of InControl
// or LosingControl.
func IsTopSubEligible(s State, cl ControlLevel) bool {
	if len(TopSubmissionsForPosition(s)) == 0 {
		return false
	}
	return cl == InControl || cl == LosingControl
}

// IsBottomSubEligible reports whether the controlled side of a pair
// can attempt a reversal submission at the given position + control
// level. Requires bottom-attack subs available AND ControlLevel of
// Controlled or BecomingControlled.
func IsBottomSubEligible(s State, cl ControlLevel) bool {
	if len(BottomSubmissionsForPosition(s)) == 0 {
		return false
	}
	return cl == Controlled || cl == BecomingControlled
}

// CrippleBodyPart returns the body part broken by a successful
// cripple-policy outcome for the given sub type. Chokes return "" —
// they don't break body parts, and cripple-policy degrades to
// subdue when the sub is a choke (see ResolveSubmissionOutcome).
func CrippleBodyPart(t SubmissionType) string {
	switch t {
	case SubArmbar, SubOmoplata:
		return "arm"
	case SubKimura, SubAmericana:
		return "shoulder"
	default:
		// Chokes (RNC, Triangle, Anaconda) — no body part
		return ""
	}
}

// SubmissionToVerb returns the third-person verb phrase for
// narration ("isolates and hyper-extends your arm" for an armbar,
// etc.). Used by Position_Messaging.go.
func SubmissionToVerb(t SubmissionType) string {
	switch t {
	case SubArmbar:
		return "isolates and hyper-extends the arm"
	case SubRNC:
		return "wraps an arm around the neck"
	case SubTriangle:
		return "wraps the legs around the neck and arm"
	case SubAmericana:
		return "cranks the shoulder backwards"
	case SubKimura:
		return "twists the shoulder into an unnatural angle"
	case SubOmoplata:
		return "traps the shoulder with the leg"
	case SubAnaconda:
		return "rolls and wraps the neck"
	default:
		return "applies a submission"
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

```
go test ./internal/state/position/... -run "TestTopSub|TestBottomSub|TestIsTop|TestIsBottom|TestCripple"
```
Expected: PASS.

- [ ] **Step 5: Build full module**

```
go build ./...
```
Expected: clean (no errors).

- [ ] **Step 6: Commit**

```
git add internal/state/position/submissions.go internal/state/position/submissions_test.go
git commit -m "feat(position): T1 — submission types + top/bottom mapping + eligibility predicates

Chunk-4d foundation. SubmissionType enum (7 named subs: Armbar /
RNC / Triangle / Americana / Kimura / Omoplata / Anaconda) +
TopSubmissionsForPosition + BottomSubmissionsForPosition mapping
tables (split by role per the symmetric-tick design — top-attack
subs from the controller side, bottom-attack reversal subs from
the controlled side). IsTopSubEligible / IsBottomSubEligible
predicates gate by position + ControlLevel. CrippleBodyPart
returns 'arm' / 'shoulder' / '' (chokes don't break limbs — those
degrade to subdue in outcome resolution). SubmissionToVerb provides
narration phrases for Position_Messaging in T11.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: Submission + surrender policy enums + Character fields

**Files:**
- Create: `internal/characters/submission_policy.go`
- Create: `internal/characters/submission_policy_test.go`
- Modify: `internal/characters/character.go` (add yaml-tagged fields + LastDriftRoll struct deferred to T5)

- [ ] **Step 1: Write the failing test**

Create `internal/characters/submission_policy_test.go`:

```go
package characters_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/stretchr/testify/assert"
)

func TestParseSubmissionPolicy(t *testing.T) {
	cases := map[string]characters.SubmissionPolicy{
		"mercy":   characters.PolicyMercy,
		"subdue":  characters.PolicySubdue,
		"cripple": characters.PolicyCripple,
		"lethal":  characters.PolicyLethal,
		"MERCY":   characters.PolicyMercy, // case-insensitive
		"":        characters.PolicySubdue, // default
	}
	for input, want := range cases {
		got, ok := characters.ParseSubmissionPolicy(input)
		assert.True(t, ok || input != "", "parse %q", input)
		assert.Equal(t, want, got)
	}
	_, ok := characters.ParseSubmissionPolicy("eviscerate")
	assert.False(t, ok, "unknown mode rejected")
}

func TestParseSurrenderPolicy(t *testing.T) {
	tests := []struct {
		input      string
		wantMode   characters.SurrenderMode
		wantThresh int
	}{
		{"never", characters.SurrenderNever, 0},
		{"always", characters.SurrenderAlways, 0},
		{"auto-tap-below 15", characters.SurrenderAutoTap, 15},
		{"auto-tap-below 50", characters.SurrenderAutoTap, 50},
	}
	for _, tc := range tests {
		p, ok := characters.ParseSurrenderPolicy(tc.input)
		assert.True(t, ok, "parse %q", tc.input)
		assert.Equal(t, tc.wantMode, p.Mode)
		assert.Equal(t, tc.wantThresh, p.HpPctThreshold)
	}
	_, ok := characters.ParseSurrenderPolicy("bogus")
	assert.False(t, ok)
	_, ok = characters.ParseSurrenderPolicy("auto-tap-below 150")
	assert.False(t, ok, "out-of-range HP pct rejected")
}

func TestDefaultSubmissionPolicyForArchetype(t *testing.T) {
	cases := map[string]characters.SubmissionPolicy{
		"predator":         characters.PolicySubdue,
		"bandit":           characters.PolicySubdue,
		"guard":            characters.PolicySubdue,
		"defensive_caster": characters.PolicyMercy,
		"tank_taunter":     characters.PolicySubdue,
		"leader":           characters.PolicyCripple,
		"generic_fighter":  characters.PolicySubdue,
		"civilian":         characters.PolicyMercy,
		"lookout":          characters.PolicySubdue,
		"":                 characters.PolicySubdue, // unknown → default
		"some_unrecognized": characters.PolicySubdue,
	}
	for arch, want := range cases {
		got := characters.DefaultSubmissionPolicyForArchetype(arch)
		assert.Equal(t, want, got, "archetype %q", arch)
	}
}

func TestDefaultSurrenderPolicyForArchetype(t *testing.T) {
	predator := characters.DefaultSurrenderPolicyForArchetype("predator")
	assert.Equal(t, characters.SurrenderNever, predator.Mode)

	civilian := characters.DefaultSurrenderPolicyForArchetype("civilian")
	assert.Equal(t, characters.SurrenderAlways, civilian.Mode)

	generic := characters.DefaultSurrenderPolicyForArchetype("generic_fighter")
	assert.Equal(t, characters.SurrenderAutoTap, generic.Mode)
	assert.Equal(t, 10, generic.HpPctThreshold)
}
```

- [ ] **Step 2: Run test to verify it fails**

```
go test ./internal/characters/... -run "TestParseSubmission|TestParseSurrender|TestDefault.*Archetype"
```
Expected: FAIL with "undefined: characters.SubmissionPolicy".

- [ ] **Step 3: Implement `submission_policy.go`**

Create `internal/characters/submission_policy.go`:

```go
package characters

import (
	"fmt"
	"strconv"
	"strings"
)

// SubmissionPolicy is the controller-side disposition that resolves
// a successful submission outcome. Set ahead of combat via
// `set submission <policy>`; consulted by the engine when a sub
// locks. No per-round prompts.
type SubmissionPolicy int

const (
	PolicySubdue  SubmissionPolicy = iota // default — knock unconscious, take some gold, leave alive
	PolicyMercy                           // release cleanly, brief recovery debuff
	PolicyCripple                         // break limb (per sub type), take gold, leave alive
	PolicyLethal                          // finishing damage drain → full death path
)

func (p SubmissionPolicy) String() string {
	switch p {
	case PolicyMercy:
		return "mercy"
	case PolicySubdue:
		return "subdue"
	case PolicyCripple:
		return "cripple"
	case PolicyLethal:
		return "lethal"
	default:
		return "unknown"
	}
}

// ParseSubmissionPolicy converts a user-input string ("mercy" / "subdue"
// / "cripple" / "lethal") to a policy value. Case-insensitive. Empty
// string returns the default (subdue, true).
func ParseSubmissionPolicy(s string) (SubmissionPolicy, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return PolicySubdue, true
	case "mercy":
		return PolicyMercy, true
	case "subdue":
		return PolicySubdue, true
	case "cripple":
		return PolicyCripple, true
	case "lethal":
		return PolicyLethal, true
	default:
		return PolicySubdue, false
	}
}

// SurrenderMode is the defender-side disposition for sending a tap
// signal when a sub locks in. Only honored by controllers with mercy
// policy (per the spec's realism framing).
type SurrenderMode int

const (
	SurrenderAutoTap SurrenderMode = iota // default — tap when HP below threshold
	SurrenderNever
	SurrenderAlways
)

// SurrenderPolicy combines the mode with an HP-percentage threshold
// (only meaningful for SurrenderAutoTap).
type SurrenderPolicy struct {
	Mode           SurrenderMode `yaml:"mode"`
	HpPctThreshold int           `yaml:"hp_pct_threshold"` // 1-100; ignored unless Mode == SurrenderAutoTap
}

func (p SurrenderPolicy) String() string {
	switch p.Mode {
	case SurrenderNever:
		return "never"
	case SurrenderAlways:
		return "always"
	case SurrenderAutoTap:
		return fmt.Sprintf("auto-tap-below %d", p.HpPctThreshold)
	default:
		return "unknown"
	}
}

// ParseSurrenderPolicy parses "never" / "always" / "auto-tap-below <N>".
// Returns (policy, true) on success; (zero, false) on parse failure or
// out-of-range threshold.
func ParseSurrenderPolicy(s string) (SurrenderPolicy, bool) {
	t := strings.ToLower(strings.TrimSpace(s))
	switch {
	case t == "never":
		return SurrenderPolicy{Mode: SurrenderNever}, true
	case t == "always":
		return SurrenderPolicy{Mode: SurrenderAlways}, true
	case strings.HasPrefix(t, "auto-tap-below"):
		rest := strings.TrimSpace(strings.TrimPrefix(t, "auto-tap-below"))
		n, err := strconv.Atoi(rest)
		if err != nil || n < 1 || n > 100 {
			return SurrenderPolicy{}, false
		}
		return SurrenderPolicy{Mode: SurrenderAutoTap, HpPctThreshold: n}, true
	default:
		return SurrenderPolicy{}, false
	}
}

// DefaultSubmissionPolicyForArchetype returns the default sub policy
// for a mob archetype. Used during mob spawn (`newMobByIdInternal`)
// when the YAML doesn't provide a `submission_policy:` value.
func DefaultSubmissionPolicyForArchetype(archetype string) SubmissionPolicy {
	switch archetype {
	case "leader":
		return PolicyCripple
	case "defensive_caster", "civilian", "merchant":
		return PolicyMercy
	default:
		// predator, bandit, guard, tank_taunter, generic_fighter,
		// lookout, ambusher, and unrecognized → subdue
		return PolicySubdue
	}
}

// DefaultSurrenderPolicyForArchetype mirrors the above for defender-
// side defaults.
func DefaultSurrenderPolicyForArchetype(archetype string) SurrenderPolicy {
	switch archetype {
	case "civilian", "merchant":
		return SurrenderPolicy{Mode: SurrenderAlways}
	case "predator", "bandit", "guard", "tank_taunter", "leader":
		return SurrenderPolicy{Mode: SurrenderNever}
	case "defensive_caster":
		return SurrenderPolicy{Mode: SurrenderAutoTap, HpPctThreshold: 25}
	case "lookout", "ambusher":
		return SurrenderPolicy{Mode: SurrenderAutoTap, HpPctThreshold: 20}
	default:
		// generic_fighter, unknown
		return SurrenderPolicy{Mode: SurrenderAutoTap, HpPctThreshold: 10}
	}
}
```

- [ ] **Step 4: Add Character fields**

Open `internal/characters/character.go`. Find the `Character` struct definition. Add two fields near the existing `Aggro` / combat-state fields (search for `Aggro *Aggro` to anchor):

```go
// Chunk 4d: submission policy fields. Set via `set submission`
// and `set surrender` commands. Defaults are PolicySubdue and
// SurrenderPolicy{Mode: SurrenderAutoTap, HpPctThreshold: 15}
// for players (applied by characters.New()); mobs inherit from
// archetype defaults at spawn (see DefaultSubmissionPolicyForArchetype).
SubmissionPolicy SubmissionPolicy `yaml:"submission_policy,omitempty"`
SurrenderPolicy  SurrenderPolicy  `yaml:"surrender_policy,omitempty"`

// LastSubmissionAttempted tracks the most recent sub type the
// character attempted (per role). Used by Position_SubmissionTick
// for round-robin sub-type selection so multi-sub positions don't
// hammer the same sub every round.
LastSubmissionAttempted int `yaml:"-"` // index into TopSubmissionsForPosition / BottomSubmissionsForPosition
```

Find `func New() *Character` (it's near the top of the file). Find the existing field initializations. Add to the literal:

```go
SubmissionPolicy: PolicySubdue,
SurrenderPolicy:  SurrenderPolicy{Mode: SurrenderAutoTap, HpPctThreshold: 15},
```

- [ ] **Step 5: Run test to verify it passes**

```
go test ./internal/characters/... -run "TestParseSubmission|TestParseSurrender|TestDefault.*Archetype" -v
```
Expected: PASS.

- [ ] **Step 6: Build full module**

```
go build ./...
```
Expected: clean.

- [ ] **Step 7: Commit**

```
git add internal/characters/submission_policy.go internal/characters/submission_policy_test.go internal/characters/character.go
git commit -m "feat(characters): T2 — SubmissionPolicy + SurrenderPolicy enums + Character fields + archetype defaults

Chunk-4d policy substrate. SubmissionPolicy (Mercy/Subdue/Cripple/
Lethal) is the controller-side disposition; SurrenderPolicy
(Never/Always/AutoTap with HP-pct threshold) is the defender-side.
Both serialize to YAML, default for players (Subdue + auto-tap
at 15% HP), and fall through to DefaultSubmissionPolicyForArchetype
/ DefaultSurrenderPolicyForArchetype for mobs at spawn.

ParseSubmissionPolicy / ParseSurrenderPolicy parse user input from
the upcoming 'set submission' / 'set surrender' commands (T15).
Round-robin tracker (LastSubmissionAttempted) prevents the
sub tick from hammering the same sub every round at multi-sub
positions like Mount.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Balance config knobs

**Files:**
- Modify: `internal/configs/config.balance.go`
- Modify: `internal/configs/config.balance.combat.go`
- Modify: `_datafiles/config.yaml`

The seven new knobs the spec defines. Pure additive — no behavior change yet (knobs are dormant until T6 reads them).

- [ ] **Step 1: Add fields to `config.balance.go`**

Find the existing `Balance` struct (search for `ReachStandingGrappleRadius` to anchor — chunk-4c's knobs live next to where these go). Add a new block:

```go
// Chunk 4d: submission tick knobs. See
// docs/superpowers/specs/2026-05-18-state-chunk-4d-submission-rework-design.md
SubmissionAttemptAlpha   ConfigFloat `yaml:"submission_attempt_alpha"`     // Min drift-margin (std devs) that opens a sub window (either side)
SubmissionAttemptCritZ   ConfigFloat `yaml:"submission_attempt_crit_z"`    // Defender-side shortcut: drift z >= this opens a bottom-sub window regardless of margin
SubSkillWeight           ConfigFloat `yaml:"sub_skill_weight"`             // Unarmed-combat skill contribution multiplier in the sub roll
SubBadZThreshold         ConfigFloat `yaml:"sub_bad_z_threshold"`          // Z-score below which the sub roll's bad-tier (attempter falls prone) fires
SubCritZThreshold        ConfigFloat `yaml:"sub_crit_z_threshold"`         // Z-score at or above which the sub roll's crit-tier (recipient stunned) fires
SubGoldLossFraction      ConfigFloat `yaml:"sub_gold_loss_fraction"`       // Fraction of carried gold transferred to the aggressor on subdue/cripple
BrokenLimbBuffDuration   ConfigInt   `yaml:"broken_limb_buff_duration"`    // Duration in rounds for the broken-limb buff; expires naturally via standard buff tick
```

- [ ] **Step 2: Add defaults in `config.balance.combat.go`**

Find the existing `validateCombat()` function (it sets defaults via `Set*` calls). Add the seven defaults:

```go
b.SubmissionAttemptAlpha.Set(1.0)
b.SubmissionAttemptCritZ.Set(2.0)
b.SubSkillWeight.Set(1.5)
b.SubBadZThreshold.Set(-1.0)
b.SubCritZThreshold.Set(2.0)
b.SubGoldLossFraction.Set(0.20)
b.BrokenLimbBuffDuration.Set(900)
```

- [ ] **Step 3: Surface in `_datafiles/config.yaml`**

Find the Balance section (search for `ReachUtilityFloor` to anchor — chunk-4c's knobs cluster here). Add a new sub-block after the reach knobs:

```yaml
# ── Chunk 4d: Submission tick ─────────────────────────────────
# Per-round opportunistic submission attempts gated on the chunk-4b
# control-axis drift roll. See spec
# docs/superpowers/specs/2026-05-18-state-chunk-4d-submission-rework-design.md
submission_attempt_alpha: 1.0          # Drift margin (std devs) that opens a sub window for the winning side
submission_attempt_crit_z: 2.0         # Defender-crit shortcut for bottom-sub windows
sub_skill_weight: 1.5                  # Unarmed-combat skill weight in the sub roll
sub_bad_z_threshold: -1.0              # Sub roll z < this → attempter falls prone, pair breaks
sub_crit_z_threshold: 2.0              # Sub roll z >= this → recipient stunned next round
sub_gold_loss_fraction: 0.20           # Fraction of carried gold transferred on subdue/cripple
broken_limb_buff_duration: 900         # Rounds (~60 min of play at 4s rounds); expires naturally
```

- [ ] **Step 4: Build + verify**

```
go build ./... && go test ./internal/configs/...
```
Expected: clean.

- [ ] **Step 5: Boot smoke (defensive — verify YAML loads)**

Boot the server briefly (background, kill after Server Ready), check no panic on config load:

```
go build -o /tmp/dogmud-t3.exe . && /tmp/dogmud-t3.exe > /tmp/dogmud-t3.log 2>&1 &
PID=$!
until grep -qE "Server Ready|panic" /tmp/dogmud-t3.log; do sleep 2; done
grep -E "Server Ready|panic" /tmp/dogmud-t3.log | head -3
taskkill //IM dogmud-t3.exe //F
rm -f /tmp/dogmud-t3.exe /tmp/dogmud-t3.log
```
Expected: "Server Ready", no panic.

- [ ] **Step 6: Commit**

```
git add internal/configs/config.balance.go internal/configs/config.balance.combat.go _datafiles/config.yaml
git commit -m "feat(configs): T3 — 7 balance knobs for chunk-4d submission tick

SubmissionAttemptAlpha (1.0) gates the drift-roll margin required
to open a sub window for the winning side. SubmissionAttemptCritZ
(2.0) is the defender-crit shortcut for bottom-sub windows.
SubSkillWeight (1.5) controls how much unarmed-combat skill
contributes to the separate sub roll vs raw strength. SubBadZThreshold
(-1.0) / SubCritZThreshold (2.0) define the bad/crit tiers of the
sub roll. SubGoldLossFraction (0.20) is the wallet-tax on
subdue/cripple. BrokenLimbBuffDuration (900) is the duration of the
broken-limb buff in rounds (~60 min of play at 4s rounds; expires
naturally via standard buff tick).

Knobs are dormant — no behavior change until T6 reads them.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: Submission roll + tier resolution

**Files:**
- Create: `internal/combat/submission.go`
- Create: `internal/combat/submission_test.go`

Pure stateless function that takes attempter + recipient + sub type, runs a fresh opposed roll, classifies into a tier. No side effects — outcome application is T7.

- [ ] **Step 1: Write the failing test**

Create `internal/combat/submission_test.go`:

```go
package combat_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/stretchr/testify/assert"
)

func setupBalanceForSubmissionTests(t *testing.T) {
	// Ensure balance config is loaded so the sub roll picks up the
	// T3 knobs.
	cfg := configs.GetBalanceConfig()
	if cfg.SubmissionAttemptAlpha.Float() == 0 {
		t.Helper()
		t.Skip("balance config not initialized for tests")
	}
}

func newCharFor(t *testing.T, str, vit, unarmedSkill int) *characters.Character {
	t.Helper()
	c := characters.New()
	c.Stats.Strength.Base = str
	c.Stats.Strength.ValueAdj = str
	c.Stats.Vitality.Base = vit
	c.Stats.Vitality.ValueAdj = vit
	if c.Skills == nil {
		c.Skills = map[string]int{}
	}
	c.Skills[string(skills.UnarmedCombat)] = unarmedSkill
	return c
}

func TestRollSubmissionAttempt_Structure(t *testing.T) {
	setupBalanceForSubmissionTests(t)
	atk := newCharFor(t, 120, 100, 30)
	def := newCharFor(t, 100, 100, 10)
	res := combat.RollSubmissionAttempt(atk, def, position.SubArmbar)
	assert.Equal(t, position.SubArmbar, res.SubType)
	assert.NotZero(t, res.AttackerScore)
	assert.NotZero(t, res.DefenderScore)
	// Tier is one of the four enum values
	assert.True(t, res.Tier >= combat.SubTierBad && res.Tier <= combat.SubTierCrit)
}

func TestRollSubmissionAttempt_BadTierOnVeryLowZ(t *testing.T) {
	// Force a bad-tier by feeding a synthetic AttemptResult.
	// We can't easily make dice.OpposedRollStat deterministic in unit
	// tests, but we CAN verify ClassifySubmissionTier directly.
	setupBalanceForSubmissionTests(t)
	cfg := configs.GetBalanceConfig()
	assert.Equal(t, combat.SubTierBad,
		combat.ClassifySubmissionTier(false, cfg.SubBadZThreshold.Float()-0.5))
}

func TestClassifySubmissionTier_BoundaryConditions(t *testing.T) {
	setupBalanceForSubmissionTests(t)
	cfg := configs.GetBalanceConfig()
	bad := cfg.SubBadZThreshold.Float()
	crit := cfg.SubCritZThreshold.Float()

	// Strictly less than bad threshold → Bad (atk roll didn't succeed
	// here either way; we use bad-tier when z < bad)
	assert.Equal(t, combat.SubTierBad, combat.ClassifySubmissionTier(false, bad-0.1))
	// Equal to bad threshold and failed → Neutral
	assert.Equal(t, combat.SubTierNeutral, combat.ClassifySubmissionTier(false, bad))
	// Failed and above bad threshold → Neutral
	assert.Equal(t, combat.SubTierNeutral, combat.ClassifySubmissionTier(false, 0.0))
	// Success below crit threshold → Success
	assert.Equal(t, combat.SubTierSuccess, combat.ClassifySubmissionTier(true, crit-0.1))
	// Success at or above crit threshold → Crit
	assert.Equal(t, combat.SubTierCrit, combat.ClassifySubmissionTier(true, crit))
	assert.Equal(t, combat.SubTierCrit, combat.ClassifySubmissionTier(true, crit+1.0))
}
```

- [ ] **Step 2: Run test to verify it fails**

```
go test ./internal/combat/... -run "TestRollSubmission|TestClassifySubmission"
```
Expected: FAIL with "undefined: combat.RollSubmissionAttempt".

- [ ] **Step 3: Implement `submission.go`**

Create `internal/combat/submission.go`:

```go
package combat

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

// SubmissionTier classifies the outcome of a per-round submission
// attempt roll into one of four bands. See spec
// "Per-round submission tick mechanics".
type SubmissionTier int

const (
	SubTierBad     SubmissionTier = iota // Attempter overcommits; falls Prone, pair breaks to Standing
	SubTierNeutral                       // Failed but no consequence; pair stays
	SubTierSuccess                       // Sub locks; outcome resolves via attempter's SubmissionPolicy
	SubTierCrit                          // Sub locks AND recipient is Stunned next round (T10 buff)
)

func (t SubmissionTier) String() string {
	switch t {
	case SubTierBad:
		return "bad"
	case SubTierNeutral:
		return "neutral"
	case SubTierSuccess:
		return "success"
	case SubTierCrit:
		return "crit"
	default:
		return "unknown"
	}
}

// SubmissionAttemptResult is the output of RollSubmissionAttempt.
// All four fields are useful downstream: SubType drives narration +
// body-part mapping, Tier drives the outcome branch, AttackerScore /
// DefenderScore are exposed for analytics + debugging, and
// AttackerZScore is the basis for the tier classification.
type SubmissionAttemptResult struct {
	SubType         position.SubmissionType
	Tier            SubmissionTier
	AttackerScore   float64
	DefenderScore   float64
	AttackerZScore  float64
	DefenderZScore  float64
	Margin          float64 // attacker margin (positive = attacker won)
}

// RollSubmissionAttempt rolls a fresh opposed Strength + Unarmed-
// combat-skill check between attempter and recipient. This is a
// SEPARATE roll from the chunk-4b drift roll — drift gates the
// opportunity, this roll resolves the attempt.
//
// Formula:
//
//	attackerScore = attempter.Strength
//	              + attempter.UnarmedCombatSkill * SubSkillWeight
//	defenderScore = recipient.Strength
//	              + recipient.Vitality
//	              + recipient.UnarmedCombatSkill * SubSkillWeight
func RollSubmissionAttempt(
	attempter *characters.Character,
	recipient *characters.Character,
	subType position.SubmissionType,
) SubmissionAttemptResult {
	cfg := configs.GetBalanceConfig()
	skillWeight := cfg.SubSkillWeight.Float()

	atkScore := float64(attempter.Stats.Strength.ValueAdj) +
		float64(attempter.GetSkillLevel(skills.UnarmedCombat))*skillWeight
	defScore := float64(recipient.Stats.Strength.ValueAdj) +
		float64(recipient.Stats.Vitality.ValueAdj) +
		float64(recipient.GetSkillLevel(skills.UnarmedCombat))*skillWeight

	success, margin, atkRoll, defRoll := dice.OpposedRollStat(atkScore, defScore)

	return SubmissionAttemptResult{
		SubType:         subType,
		Tier:            ClassifySubmissionTier(success, atkRoll.ZScore),
		AttackerScore:   atkScore,
		DefenderScore:   defScore,
		AttackerZScore:  atkRoll.ZScore,
		DefenderZScore:  defRoll.ZScore,
		Margin:          margin,
	}
}

// ClassifySubmissionTier maps (success, attacker z-score) to a tier
// per the spec table. Exposed for unit testing of the boundary
// conditions independently from the dice roll.
func ClassifySubmissionTier(success bool, attackerZ float64) SubmissionTier {
	cfg := configs.GetBalanceConfig()
	bad := cfg.SubBadZThreshold.Float()
	crit := cfg.SubCritZThreshold.Float()

	if success {
		if attackerZ >= crit {
			return SubTierCrit
		}
		return SubTierSuccess
	}
	if attackerZ < bad {
		return SubTierBad
	}
	return SubTierNeutral
}
```

- [ ] **Step 4: Run test to verify it passes**

```
go test ./internal/combat/... -run "TestRollSubmission|TestClassifySubmission" -v
```
Expected: PASS.

- [ ] **Step 5: Commit**

```
git add internal/combat/submission.go internal/combat/submission_test.go
git commit -m "feat(combat): T4 — RollSubmissionAttempt + tier classification

Pure stateless sub roll. Opposed Strength + Unarmed-combat-skill
check between attempter and recipient (defender gets bonus Vitality
in their score — toughness matters when you're being submitted).
ClassifySubmissionTier branches into 4 bands per the spec:
  - Bad (z < SubBadZThreshold, default -1.0) — attempter overcommits
  - Neutral (failed but not catastrophic) — pair unchanged
  - Success (passed below crit threshold) — sub locks, outcome resolves
  - Crit (z >= SubCritZThreshold, default 2.0) — sub locks + stun

Side-effect-free. T6 (Position_SubmissionTick) gates the opportunity
via the chunk-4b drift result; T7 (ResolveSubmissionOutcome) applies
the tier's downstream effect.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: Drift-roll snapshot on Character

**Files:**
- Modify: `internal/characters/character.go` (add `LastDriftRoll` struct + field)
- Modify: `internal/hooks/Position_GrappleTick.go` (write to LastDriftRoll after the roll)
- Modify: `internal/hooks/Position_GrappleTick_test.go` (verify the snapshot is written)

So Position_SubmissionTick (T6) can read the (margin, z-score) the drift tick produced this round. Stash on the Character with a round-number guard so stale data doesn't fire.

- [ ] **Step 1: Add struct + field to `character.go`**

Open `internal/characters/character.go`. Near the bottom of the file (after the existing struct definitions) add:

```go
// DriftRollSnapshot captures the chunk-4b grapple-tick drift roll
// result for the most recent round, so that the chunk-4d
// Position_SubmissionTick observer can read it without re-rolling.
// The two sides are stored separately because both can be checked
// for sub-attempt eligibility per round.
type DriftRollSnapshot struct {
	Round           uint64  // round number this snapshot was taken
	MarginAttacker  float64 // attacker-side margin (positive = attacker won)
	AttackerZScore  float64
	DefenderZScore  float64
}
```

Add to the `Character` struct (near `LastSubmissionAttempted` from T2):

```go
LastDriftRoll DriftRollSnapshot `yaml:"-"` // chunk 4d: read by Position_SubmissionTick
```

- [ ] **Step 2: Write the snapshot in `Position_GrappleTick.go`**

Open `internal/hooks/Position_GrappleTick.go`. Find `processGrapplePair` (the function that rolls the opposed drift check and applies ControlLevel drift). Locate the line that calls `dice.OpposedRollStat` (or `dice.OpposedRoll` — look for the actual roll function). Immediately after the roll, add:

```go
// Chunk 4d: stash this round's drift result on both characters so
// Position_SubmissionTick can read it without re-rolling.
currentRound := util.GetRoundCount()
controller.LastDriftRoll = characters.DriftRollSnapshot{
	Round:          currentRound,
	MarginAttacker: margin,
	AttackerZScore: attackerRoll.ZScore,
	DefenderZScore: defenderRoll.ZScore,
}
controlled.LastDriftRoll = characters.DriftRollSnapshot{
	Round:          currentRound,
	MarginAttacker: margin, // same value — read sign-correctly per side in T6
	AttackerZScore: attackerRoll.ZScore,
	DefenderZScore: defenderRoll.ZScore,
}
```

The variable names (`margin`, `attackerRoll`, `defenderRoll`, `controller`, `controlled`) should match whatever the existing code uses — read the surrounding context first and adapt. Add the `util` import if not already present.

- [ ] **Step 3: Add a test verifying the snapshot is written**

Open `internal/hooks/Position_GrappleTick_test.go`. Find an existing test that exercises a drift cycle (search for `processGrapplePair`). Add a new test:

```go
func TestProcessGrapplePair_StashesDriftSnapshot(t *testing.T) {
	// Setup: two characters in a Mount grapple (use the existing
	// setCombatPositionParallel helper or transition the FSM
	// directly).
	a := characters.New()
	b := characters.New()
	// ... position setup (mirror an existing test's setup pattern) ...
	roundBefore := util.GetRoundCount()
	processGrapplePair(a, b)
	roundAfter := util.GetRoundCount()
	// Verify both characters got a snapshot for a current round
	assert.GreaterOrEqual(t, a.LastDriftRoll.Round, roundBefore)
	assert.LessOrEqual(t, a.LastDriftRoll.Round, roundAfter)
	assert.Equal(t, a.LastDriftRoll.Round, b.LastDriftRoll.Round, "both sides see same round")
	assert.NotZero(t, a.LastDriftRoll.AttackerZScore, "z-score populated")
}
```

If the existing test file doesn't import the symbols you need, mirror the imports + helper setup from an existing nearby test.

- [ ] **Step 4: Build + test**

```
go test ./internal/characters/... ./internal/hooks/... -run "TestProcessGrapplePair_StashesDriftSnapshot|TestProcess"
go build ./...
```
Expected: PASS + clean.

- [ ] **Step 5: Commit**

```
git add internal/characters/character.go internal/hooks/Position_GrappleTick.go internal/hooks/Position_GrappleTick_test.go
git commit -m "feat(position): T5 — stash drift-roll snapshot on Character for sub tick

Adds DriftRollSnapshot struct + Character.LastDriftRoll field
(yaml:'-' — runtime only). Position_GrappleTick writes both sides
after the opposed roll so Position_SubmissionTick (T6) can read the
(margin, z-score) without re-rolling and getting a different result.
Round-numbered so T6 can detect stale data from a prior round.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 6: Per-round submission tick observer

**Files:**
- Create: `internal/hooks/Position_SubmissionTick.go`
- Create: `internal/hooks/Position_SubmissionTick_test.go`

The new per-round observer. Fires AFTER Position_GrappleTick so it can read the freshly-stashed drift snapshot. For each active grapple pair, checks both sides for sub-attempt opportunity, rolls a sub attempt, and (in T7) calls outcome resolution. For now T6 wires the observer and the eligibility check; outcome application is stubbed.

- [ ] **Step 1: Write the failing test**

Create `internal/hooks/Position_SubmissionTick_test.go`:

```go
package hooks_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
)

// Setup helpers: put `a` and `b` in a Mount grapple with `a` as
// controller (InControl), `b` as Controlled.
func setupMountPair(t *testing.T) (controller, controlled *characters.Character) {
	t.Helper()
	a := characters.New()
	b := characters.New()
	// ... mirror an existing Mount setup pattern from
	// internal/state/position/ or internal/hooks/ tests ...
	return a, b
}

func TestEvaluateSubAttempt_ControllerEligible(t *testing.T) {
	a, b := setupMountPair(t)
	// Fake a drift roll where controller won big
	a.LastDriftRoll = characters.DriftRollSnapshot{
		Round:          util.GetRoundCount(),
		MarginAttacker: 2.0, // controller won by 2 std devs
		AttackerZScore: 1.5,
	}
	b.LastDriftRoll = a.LastDriftRoll
	role, eligible := hooks.EvaluateSubAttempt(a, b)
	assert.True(t, eligible)
	assert.Equal(t, combat.RoleTop, role)
}

func TestEvaluateSubAttempt_DefenderEligibleViaMargin(t *testing.T) {
	a, b := setupMountPair(t)
	a.LastDriftRoll = characters.DriftRollSnapshot{
		Round:          util.GetRoundCount(),
		MarginAttacker: -2.0, // defender won by 2 std devs
		AttackerZScore: -1.5,
		DefenderZScore: 1.5,
	}
	b.LastDriftRoll = a.LastDriftRoll
	role, eligible := hooks.EvaluateSubAttempt(a, b)
	assert.True(t, eligible)
	assert.Equal(t, combat.RoleBottom, role)
}

func TestEvaluateSubAttempt_DefenderEligibleViaCritShortcut(t *testing.T) {
	a, b := setupMountPair(t)
	// Defender crit on defense but margin not above alpha
	a.LastDriftRoll = characters.DriftRollSnapshot{
		Round:          util.GetRoundCount(),
		MarginAttacker: 0.0,
		AttackerZScore: 0.0,
		DefenderZScore: 2.5, // CRIT
	}
	b.LastDriftRoll = a.LastDriftRoll
	role, eligible := hooks.EvaluateSubAttempt(a, b)
	assert.True(t, eligible)
	assert.Equal(t, combat.RoleBottom, role)
}

func TestEvaluateSubAttempt_NeitherEligible(t *testing.T) {
	a, b := setupMountPair(t)
	a.LastDriftRoll = characters.DriftRollSnapshot{
		Round:          util.GetRoundCount(),
		MarginAttacker: 0.3, // not enough on either side
		AttackerZScore: 0.3,
		DefenderZScore: -0.3,
	}
	b.LastDriftRoll = a.LastDriftRoll
	_, eligible := hooks.EvaluateSubAttempt(a, b)
	assert.False(t, eligible)
}

func TestEvaluateSubAttempt_StaleSnapshotIgnored(t *testing.T) {
	a, b := setupMountPair(t)
	// Snapshot is from a prior round → ignore
	a.LastDriftRoll = characters.DriftRollSnapshot{
		Round:          util.GetRoundCount() - 1,
		MarginAttacker: 5.0, // would normally fire, but stale
		AttackerZScore: 3.0,
	}
	b.LastDriftRoll = a.LastDriftRoll
	_, eligible := hooks.EvaluateSubAttempt(a, b)
	assert.False(t, eligible)
}
```

- [ ] **Step 2: Run test to verify it fails**

```
go test ./internal/hooks/... -run "TestEvaluateSubAttempt"
```
Expected: FAIL with "undefined: hooks.EvaluateSubAttempt" + "undefined: combat.RoleTop / RoleBottom".

- [ ] **Step 3: Add Role enum to `internal/combat/submission.go`**

Append to `internal/combat/submission.go`:

```go
// Role discriminates whether a sub attempt comes from the top
// (controller) side of a grapple or the bottom (controlled) side.
// Used by Position_SubmissionTick + Position_Messaging to pick the
// right submission pool and narration.
type Role int

const (
	RoleTop Role = iota
	RoleBottom
)

func (r Role) String() string {
	switch r {
	case RoleTop:
		return "top"
	case RoleBottom:
		return "bottom"
	default:
		return "unknown"
	}
}
```

- [ ] **Step 4: Implement `Position_SubmissionTick.go`**

Create `internal/hooks/Position_SubmissionTick.go`:

```go
// Position_SubmissionTick.go fires once per round per active grapple
// pair, after Position_GrappleTick has stashed the drift snapshot.
// Checks each side of the pair for sub-attempt eligibility (top
// attack from the controller, bottom-attack reversal from the
// controlled side); if eligible, rolls a fresh opposed sub roll and
// applies the outcome via combat.ResolveSubmissionOutcome (T7).
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// processSubmissionTick is the NewRound event listener. Iterates
// active players + mobs; for each pair where a sub attempt is
// eligible, fires the sub roll. Runs AFTER processGrappleTick so
// LastDriftRoll is fresh.
func processSubmissionTick(e events.Event) events.ListenerReturn {
	for _, u := range users.GetAllActiveUsers() {
		if u == nil || u.Character == nil {
			continue
		}
		processSubmissionTickForChar(u.Character)
	}
	for _, mobInstId := range mobs.GetAllMobInstanceIds() {
		m := mobs.GetInstance(mobInstId)
		if m == nil {
			continue
		}
		processSubmissionTickForChar(&m.Character)
	}
	return events.Continue
}

// processSubmissionTickForChar handles a single character's
// sub-tick opportunity (whether top or bottom). Iterates the
// controller-side only — the controlled side is handled when its
// controller is processed (so each pair is processed exactly once).
func processSubmissionTickForChar(c *characters.Character) {
	if c == nil || c.Position == nil {
		return
	}
	if !c.IsController() {
		return
	}
	partner := resolvePartner(c)
	if partner == nil {
		return
	}
	role, eligible := EvaluateSubAttempt(c, partner)
	if !eligible {
		return
	}
	var attempter, recipient *characters.Character
	var subPool []position.SubmissionType
	switch role {
	case combat.RoleTop:
		attempter, recipient = c, partner
		subPool = position.TopSubmissionsForPosition(c.Position.State())
	case combat.RoleBottom:
		attempter, recipient = partner, c
		subPool = position.BottomSubmissionsForPosition(c.Position.State())
	}
	if len(subPool) == 0 {
		return
	}
	subType := pickSubmissionRoundRobin(attempter, subPool)
	result := combat.RollSubmissionAttempt(attempter, recipient, subType)
	mudlog.Debug("SubmissionTick",
		"role", role, "attempter", attempter.Name, "recipient", recipient.Name,
		"sub", subType, "tier", result.Tier, "atkZ", result.AttackerZScore)
	// T7: combat.ResolveSubmissionOutcome(attempter, recipient, result, role)
	// (Stub for T6 — full outcome resolution lands in T7. For now
	// the tick fires, logs, and returns — the actual outcome plumbing
	// is added once the resolver exists.)
}

// EvaluateSubAttempt checks whether a sub attempt is eligible for
// either side of a grapple pair based on the chunk-4b drift snapshot
// stashed on Character.LastDriftRoll this round. Returns the role of
// the attempter (top = controller, bottom = controlled) and whether
// eligible.
//
// Per the spec: at most one side passes per round because the drift
// roll has one winner. If both sides happen to qualify (e.g., wide
// alpha + crit shortcut), prefer the side with the larger absolute
// z-score.
func EvaluateSubAttempt(controller, controlled *characters.Character) (combat.Role, bool) {
	if controller == nil || controller.Position == nil ||
		controlled == nil || controlled.Position == nil {
		return combat.RoleTop, false
	}
	currentRound := util.GetRoundCount()
	snap := controller.LastDriftRoll
	if snap.Round != currentRound {
		return combat.RoleTop, false // stale or missing snapshot
	}

	cfg := configs.GetBalanceConfig()
	alpha := cfg.SubmissionAttemptAlpha.Float()
	critZ := cfg.SubmissionAttemptCritZ.Float()

	posState := controller.Position.State()
	cl, _ := controller.Position.ControlLevel()

	// Top eligibility: controller won drift roll big AND top subs available
	topOK := false
	if position.IsTopSubEligible(posState, cl) && snap.MarginAttacker > alpha {
		topOK = true
	}

	// Bottom eligibility: defender won drift roll big OR crit-defended,
	// AND bottom subs available
	bottomOK := false
	bottomMargin := -snap.MarginAttacker // defender margin = inverse
	if position.IsBottomSubEligible(posState, cl) {
		if bottomMargin > alpha || snap.DefenderZScore >= critZ {
			bottomOK = true
		}
	}

	switch {
	case topOK && !bottomOK:
		return combat.RoleTop, true
	case bottomOK && !topOK:
		return combat.RoleBottom, true
	case topOK && bottomOK:
		// Tiebreak by larger absolute z-score
		if snap.AttackerZScore >= snap.DefenderZScore {
			return combat.RoleTop, true
		}
		return combat.RoleBottom, true
	default:
		return combat.RoleTop, false
	}
}

// pickSubmissionRoundRobin advances the attempter's
// LastSubmissionAttempted index and returns the next sub from the
// pool. Wraps modulo the pool length. Empty pool returns SubNone
// (caller checks).
func pickSubmissionRoundRobin(c *characters.Character, pool []position.SubmissionType) position.SubmissionType {
	if len(pool) == 0 {
		return position.SubNone
	}
	c.LastSubmissionAttempted = (c.LastSubmissionAttempted + 1) % len(pool)
	return pool[c.LastSubmissionAttempted]
}

func init() {
	// Register AFTER processGrappleTick. The exact mechanism for
	// event-listener ordering in DOGMud uses NewRoundPhase or a
	// numeric priority — read internal/events/listener.go to confirm.
	// For now: events.RegisterListener("NewRound", processSubmissionTick, events.PriorityAfter(processGrappleTick))
	events.RegisterListener(events.NewRound{}, processSubmissionTick)
}
```

The exact `events.RegisterListener` API is whatever the codebase uses
(check by reading `internal/events/` + the existing
`processGrappleTick` registration). The key invariant: this listener
MUST fire AFTER `processGrappleTick` so `LastDriftRoll` is fresh.

- [ ] **Step 5: Verify event ordering**

Search for how other observers register in priority order:

```
grep -r "PriorityAfter\|RegisterListener" internal/hooks/ | head -10
```

Mirror the existing pattern (might be plain registration in init-order, or a `Priority` field, or `RunAfter`). Adjust the init() in Position_SubmissionTick.go to match.

- [ ] **Step 6: Run tests**

```
go test ./internal/hooks/... -run "TestEvaluateSubAttempt"
go test ./internal/combat/... -run "TestRollSubmission|TestClassify"
go build ./...
```
Expected: PASS + clean.

- [ ] **Step 7: Commit**

```
git add internal/hooks/Position_SubmissionTick.go internal/hooks/Position_SubmissionTick_test.go internal/combat/submission.go
git commit -m "feat(hooks): T6 — Position_SubmissionTick observer + EvaluateSubAttempt

Per-round observer that fires after Position_GrappleTick. For each
active grapple pair, EvaluateSubAttempt reads the freshly-stashed
drift snapshot (Character.LastDriftRoll, written by T5) and decides
whether either side has a sub-attempt opportunity this round:
  - Top (controller): drift margin > alpha AND top-attack subs at position
  - Bottom (controlled): defender drift margin > alpha OR defender
    z-score >= critZ AND bottom-attack subs at position
  - Tiebreak on larger absolute z-score

Picks a sub type via per-character round-robin (no hammering of the
same sub each round). Rolls via combat.RollSubmissionAttempt (T4).
Outcome application is stubbed in this task — full resolution lands
in T7.

Adds combat.Role enum (RoleTop / RoleBottom) shared by T6 + T7
+ Position_Messaging.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 7: Outcome resolver + policy matrix

**Files:**
- Create: `internal/combat/submission_outcome.go`
- Create: `internal/combat/submission_outcome_test.go`
- Modify: `internal/hooks/Position_SubmissionTick.go` (call resolver)

The policy matrix translation. Given (attempter, recipient, sub type, tier, role, attempter's SubmissionPolicy, recipient's SurrenderPolicy), apply the outcome:
- Bad tier → attempter falls Prone, pair breaks to Standing (regardless of policy)
- Neutral → no-op
- Success / Crit → apply attempter's policy via the matrix below

This is the meatiest task. Implements the four-tier ladder + handles choke-degradation + emits the death-cascade triggers.

- [ ] **Step 1: Write the failing test**

Create `internal/combat/submission_outcome_test.go`:

```go
package combat_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/stretchr/testify/assert"
)

// Note: this test focuses on the policy-resolution logic, not the
// full Life cascade integration (which lands in T8 + integration
// tests). It uses fake hooks injected via ResolveSubmissionOutcome
// callbacks (or, if the resolver instead emits events, asserts on
// the event queue).

func TestResolveSubmissionOutcome_BadTierKnocksAttempterProne(t *testing.T) {
	atk := characters.New()
	def := characters.New()
	// Place them in a Mount grapple (set up FSM, mirror existing setup helper)
	// ...
	result := combat.SubmissionAttemptResult{Tier: combat.SubTierBad, SubType: position.SubArmbar}
	combat.ResolveSubmissionOutcome(atk, def, result, combat.RoleTop)
	// Bad tier should knock attempter prone + break the pair
	assert.True(t, atk.IsProne(), "attempter should be prone after bad-tier sub")
	assert.False(t, atk.IsGrappling(), "pair should have broken")
}

func TestResolveSubmissionOutcome_NeutralTierNoOp(t *testing.T) {
	atk := characters.New()
	def := characters.New()
	// ... setup Mount ...
	preState := atk.Position.State()
	result := combat.SubmissionAttemptResult{Tier: combat.SubTierNeutral, SubType: position.SubArmbar}
	combat.ResolveSubmissionOutcome(atk, def, result, combat.RoleTop)
	assert.Equal(t, preState, atk.Position.State(), "position unchanged on neutral")
}

func TestResolveSubmissionOutcome_SuccessMercyReleases(t *testing.T) {
	atk := characters.New()
	atk.SubmissionPolicy = characters.PolicyMercy
	def := characters.New()
	// ... setup Mount ...
	result := combat.SubmissionAttemptResult{Tier: combat.SubTierSuccess, SubType: position.SubArmbar}
	combat.ResolveSubmissionOutcome(atk, def, result, combat.RoleTop)
	// Mercy: clean release, both back to Standing, both alive
	assert.False(t, atk.IsGrappling())
	assert.False(t, def.IsGrappling())
	assert.True(t, def.IsAlive(), "defender alive on mercy")
}

func TestResolveSubmissionOutcome_SuccessCrippleArmbar(t *testing.T) {
	atk := characters.New()
	atk.SubmissionPolicy = characters.PolicyCripple
	def := characters.New()
	// ... setup Mount, defender has gold ...
	def.Gold = 100
	result := combat.SubmissionAttemptResult{Tier: combat.SubTierSuccess, SubType: position.SubArmbar}
	combat.ResolveSubmissionOutcome(atk, def, result, combat.RoleTop)
	// Cripple with arm sub: death cascade with NoDeprogression,
	// broken-limb buff applied (T9), some gold transferred (T8)
	// Defender should be in the dying/dead state
	assert.False(t, def.IsAlive(), "defender dies via no-deprogression cascade")
	assert.Less(t, def.Gold, 100, "some gold transferred away")
	// Broken-limb buff assertion lands in T9; for T7 just verify
	// the cascade fired.
}

func TestResolveSubmissionOutcome_SuccessCrippleChokeDegradesToSubdue(t *testing.T) {
	atk := characters.New()
	atk.SubmissionPolicy = characters.PolicyCripple
	def := characters.New()
	// ... setup BackGround (RNC available) ...
	result := combat.SubmissionAttemptResult{Tier: combat.SubTierSuccess, SubType: position.SubRNC}
	combat.ResolveSubmissionOutcome(atk, def, result, combat.RoleTop)
	// Chokes don't break — should degrade to subdue (same death-
	// cascade-with-no-deprogression, but no broken-limb buff)
	assert.False(t, def.IsAlive())
}

func TestResolveSubmissionOutcome_SuccessSubdueDefenderTapsWithMercyController(t *testing.T) {
	atk := characters.New()
	atk.SubmissionPolicy = characters.PolicyMercy
	def := characters.New()
	def.SurrenderPolicy = characters.SurrenderPolicy{Mode: characters.SurrenderAlways}
	// ... setup Mount ...
	result := combat.SubmissionAttemptResult{Tier: combat.SubTierSuccess, SubType: position.SubArmbar}
	combat.ResolveSubmissionOutcome(atk, def, result, combat.RoleTop)
	// Mercy + always-tap: clean release
	assert.True(t, def.IsAlive())
}

func TestResolveSubmissionOutcome_SuccessSubdueDefenderTapsWithSubdueController(t *testing.T) {
	atk := characters.New()
	atk.SubmissionPolicy = characters.PolicySubdue
	def := characters.New()
	def.SurrenderPolicy = characters.SurrenderPolicy{Mode: characters.SurrenderAlways}
	// ... setup Mount ...
	result := combat.SubmissionAttemptResult{Tier: combat.SubTierSuccess, SubType: position.SubArmbar}
	combat.ResolveSubmissionOutcome(atk, def, result, combat.RoleTop)
	// Subdue: tap is IGNORED (the realism point — only mercy honors tap)
	assert.False(t, def.IsAlive(), "subdue ignores tap; defender enters death cascade")
}
```

- [ ] **Step 2: Run test to verify it fails**

```
go test ./internal/combat/... -run "TestResolveSubmissionOutcome"
```
Expected: FAIL with "undefined: combat.ResolveSubmissionOutcome".

- [ ] **Step 3: Implement `submission_outcome.go`**

Create `internal/combat/submission_outcome.go`:

```go
package combat

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

// ResolveSubmissionOutcome applies the attempt result to the
// attempter + recipient based on the attempter's SubmissionPolicy
// and the recipient's SurrenderPolicy. Side effects:
//   - Bad tier: attempter knocked Prone, pair breaks to Standing
//   - Neutral tier: no-op (caller short-circuits messaging)
//   - Success/Crit + mercy: clean release; if Crit, also applies the
//     1-round Stunned buff on the recipient (T10)
//   - Success/Crit + subdue: death cascade with NoDeprogression +
//     gold transfer
//   - Success/Crit + cripple + non-choke sub: death cascade with
//     NoDeprogression + gold transfer + broken-limb buff (T9)
//   - Success/Crit + cripple + choke sub: degrades to subdue
//   - Success/Crit + lethal: full death cascade (NoDeprogression
//     false) with full corpse + deprogression
//
// Defender's SurrenderPolicy is consulted only when attempter's
// policy is mercy. Other policies ignore the tap signal (realism).
func ResolveSubmissionOutcome(
	attempter *characters.Character,
	recipient *characters.Character,
	result SubmissionAttemptResult,
	role Role,
) {
	switch result.Tier {
	case SubTierBad:
		applyBadTier(attempter, recipient)
	case SubTierNeutral:
		return // no-op
	case SubTierSuccess, SubTierCrit:
		applySuccessByPolicy(attempter, recipient, result, role)
	}
}

func applyBadTier(attempter, recipient *characters.Character) {
	// Break the pair → Standing, then knock attempter Prone for 2 rounds
	if err := position.TransitionPair(
		attempter, recipient, position.Standing,
		state.TransitionReason{Trigger: position.TriggerGrappleBreak},
	); err != nil {
		mudlog.Warn("submission bad-tier: TransitionPair → Standing failed", "err", err)
		return
	}
	_ = attempter.Position.TransitionToProne(
		position.ProneData{MinRecoveryRounds: 2},
		state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward},
	)
}

func applySuccessByPolicy(
	attempter, recipient *characters.Character,
	result SubmissionAttemptResult,
	role Role,
) {
	policy := attempter.SubmissionPolicy

	// Choke degradation: cripple on a choke-class sub becomes subdue
	// (chokes don't break limbs).
	bodyPart := position.CrippleBodyPart(result.SubType)
	if policy == characters.PolicyCripple && bodyPart == "" {
		policy = characters.PolicySubdue
	}

	// Crit-tier applies the 1-round Stunned buff on the recipient
	// ONLY when the outcome keeps the recipient in combat (mercy).
	// For subdue/cripple/lethal the recipient enters the death
	// cascade and the stun would be a no-op.
	if result.Tier == SubTierCrit && policy == characters.PolicyMercy {
		applyStunnedBuff(recipient)
	}

	switch policy {
	case characters.PolicyMercy:
		applyMercyRelease(attempter, recipient)
	case characters.PolicySubdue:
		applyDeathCascade(attempter, recipient, true /*noDeprogression*/, false /*noBrokenLimb*/, "")
	case characters.PolicyCripple:
		applyDeathCascade(attempter, recipient, true /*noDeprogression*/, true /*brokenLimb*/, bodyPart)
	case characters.PolicyLethal:
		applyDeathCascade(attempter, recipient, false /*deprogression normally*/, false, "")
	}
}

func applyMercyRelease(attempter, recipient *characters.Character) {
	// Break the pair → Standing; no damage, no debuff
	if err := position.TransitionPair(
		attempter, recipient, position.Standing,
		state.TransitionReason{Trigger: position.TriggerGrappleBreak},
	); err != nil {
		mudlog.Warn("submission mercy release: TransitionPair → Standing failed", "err", err)
	}
	// T9: optional brief recovery debuff. Stub for now — apply via
	// AddBuff if the recovery-debuff buff is registered (lands in T9
	// alongside the broken-limb buff or punted to 4f).
}

func applyDeathCascade(
	killer, victim *characters.Character,
	noDeprogression bool,
	brokenLimb bool,
	brokenBodyPart string,
) {
	cfg := configs.GetBalanceConfig()
	goldFrac := 0.0
	if noDeprogression {
		// subdue/cripple — partial gold transfer
		goldFrac = cfg.SubGoldLossFraction.Float()
	}
	victim.Die(
		state.ActorRef{}, // Killer ref — for now use empty; combat layer wires through actor in T7 sub-step
		life.TriggerHealthZero, // or a new TriggerSubmission constant
	)
	// NoDeprogression + GoldLossFraction propagation: the existing
	// Die() takes a trigger string; the new fields live on DeadData
	// per T8. Until T8 lands, this resolver writes a sentinel
	// trigger-flag on a side-channel that Death_PlayerCleanup reads.
	// (T8 replaces this with a proper DeadData extension.)
	_ = brokenLimb       // T9: applies broken-limb buff after the cascade
	_ = brokenBodyPart   // T9: passed into broken-limb buff context
	_ = goldFrac         // T8: stored on DeadData for Death_PlayerCorpse
	_ = killer           // analytic + corpse-loot recipient
}

// applyStunnedBuff applies the 1-round Stunned buff defined in T10.
// Stub for T7 — full implementation lands in T10 after the buff YAML
// is registered.
func applyStunnedBuff(c *characters.Character) {
	// T10: c.AddBuff(84) — buff id 84 = submission-stunned (T10 yaml)
	_ = c
}
```

Notes on stubs: T7 wires the resolver shape but T8 (Life cascade extension) and T9 (broken-limb buff) need to land before the stubs can be removed. Mark each stub clearly so the implementer of T8/T9 knows where to plug in.

- [ ] **Step 4: Wire the resolver into the sub tick**

Open `internal/hooks/Position_SubmissionTick.go`. Find the comment `// T7: combat.ResolveSubmissionOutcome(...)`. Replace with the actual call:

```go
combat.ResolveSubmissionOutcome(attempter, recipient, result, role)
```

- [ ] **Step 5: Run tests**

```
go test ./internal/combat/... -run "TestResolveSubmissionOutcome"
go test ./internal/hooks/... -run "TestEvaluateSubAttempt"
go build ./...
```
Expected: PASS + clean. Some of the outcome tests may SKIP or PARTIAL-pass if they depend on T8/T9 plumbing — accept those as known-deferred for now and document in commit.

- [ ] **Step 6: Commit**

```
git add internal/combat/submission_outcome.go internal/combat/submission_outcome_test.go internal/hooks/Position_SubmissionTick.go
git commit -m "feat(combat): T7 — submission outcome resolver + policy matrix

ResolveSubmissionOutcome applies the four-tier resolution per spec:
  - Bad tier: attempter knocked Prone, pair breaks to Standing
  - Neutral: no-op
  - Success/Crit + mercy: clean release (+ Stunned on Crit)
  - Success/Crit + subdue: no-deprogression death + gold transfer
  - Success/Crit + cripple + arm/shoulder sub: no-deprogression
    death + gold transfer + broken-limb buff (T9 wires the buff)
  - Success/Crit + cripple + choke: degrades to subdue
  - Success/Crit + lethal: full death cascade

T8 (DeadData extension) + T9 (broken-limb buff) + T10 (Stunned
buff) are stubbed and clearly marked in comments. The resolver's
shape is final; the stubs become real calls in those tasks.

Wires the resolver into Position_SubmissionTick (T6 stub replaced).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 8: NoDeprogression + GoldLossFraction on Life cascade

**Files:**
- Modify: `internal/state/life/life.go` (extend DeadData)
- Modify: `internal/hooks/Death_PlayerCleanup.go` (skip deprogression when flag set)
- Modify: `internal/hooks/Death_PlayerCorpse.go` (partial gold transfer when fraction > 0)
- Modify: `internal/hooks/Death_PlayerCleanup_test.go` or sibling — tests for the flag-guarded branches
- Modify: `internal/combat/submission_outcome.go` (replace the Die() call with the new flag-aware variant)

- [ ] **Step 1: Extend DeadData**

Open `internal/state/life/life.go`. Find `type DeadData struct`. Add the two new fields:

```go
type DeadData struct {
	Killer            state.ActorRef
	DamageMap         map[int]int
	NoDeprogression   bool    // chunk 4d: skip stat-decay step
	GoldLossFraction  float64 // chunk 4d: 0 = full corpse loot; > 0 = transfer this fraction of gold to Killer, skip corpse
}
```

- [ ] **Step 2: Guard the deprogression branch in Death_PlayerCleanup**

Open `internal/hooks/Death_PlayerCleanup.go`. Find the existing deprogression call (search for "deprogression" or "Decay" or "stat-decay"). Wrap it:

```go
// chunk 4d: skip deprogression for no-stat-loss death triggers
// (submission subdue / cripple outcomes)
if !data.NoDeprogression {
	applyDeprogression(c)
}
```

(The exact variable name `data` should match the DeadData receiver in the existing function — look at the function signature first.)

- [ ] **Step 3: Guard corpse creation in Death_PlayerCorpse**

Open `internal/hooks/Death_PlayerCorpse.go`. Find the existing corpse-creation logic. Add a partial-gold branch BEFORE it:

```go
if data.GoldLossFraction > 0 && !data.Killer.IsZero() {
	// chunk 4d: partial gold transfer — no full corpse
	transferPartialGold(c, data.Killer, data.GoldLossFraction)
	return
}
// fall through to existing full corpse creation
```

Add the helper at the bottom of the file:

```go
// transferPartialGold moves a fraction of the dying character's
// gold to the killer (player or mob). Used by chunk-4d subdue and
// cripple outcomes that intentionally leave the victim alive but
// poorer. No corpse is created in this branch.
func transferPartialGold(victim *characters.Character, killerRef state.ActorRef, fraction float64) {
	if fraction <= 0 || fraction > 1.0 || victim.Gold <= 0 {
		return
	}
	loss := int(float64(victim.Gold) * fraction)
	if loss < 1 {
		loss = 1
	}
	victim.Gold -= loss
	// Deposit to killer
	switch {
	case killerRef.UserId > 0:
		if u := users.GetByUserId(killerRef.UserId); u != nil {
			u.Character.Gold += loss
		}
	case killerRef.MobInstanceId > 0:
		if m := mobs.GetInstance(killerRef.MobInstanceId); m != nil {
			m.Character.Gold += loss
		}
	}
}
```

(Adjust imports as needed.)

- [ ] **Step 4: Add tests for the flag-guarded branches**

Find or create `internal/hooks/Death_PlayerCleanup_test.go` (search for any existing). Add:

```go
func TestDeathCleanup_NoDeprogressionSkipsStatDecay(t *testing.T) {
	c := characters.New()
	// Capture baseline stats
	baselineStr := c.Stats.Strength.Training
	// Trigger death with NoDeprogression true
	c.Life.TransitionToDead(
		life.DeadData{NoDeprogression: true},
		state.TransitionReason{Trigger: life.TriggerHealthZero},
	)
	assert.Equal(t, baselineStr, c.Stats.Strength.Training, "training unchanged with NoDeprogression")
}

func TestDeathCorpse_PartialGoldTransfersToKillerSkipsCorpse(t *testing.T) {
	killer := characters.New()
	victim := characters.New()
	victim.Gold = 100
	// Trigger death with partial gold
	victim.Life.TransitionToDead(
		life.DeadData{
			Killer:           state.ActorRef{UserId: 1}, // pseudo
			GoldLossFraction: 0.20,
		},
		state.TransitionReason{Trigger: life.TriggerHealthZero},
	)
	assert.Equal(t, 80, victim.Gold, "20% transferred away")
	// (Asserting receipt on Killer requires registering the killer
	// in users — out of scope for unit test; verified via integration
	// in T20 matrix.)
	_ = killer
}
```

- [ ] **Step 5: Replace stub in submission_outcome.go**

Open `internal/combat/submission_outcome.go`. Find `applyDeathCascade`. Replace the `victim.Die(...)` call with one that passes the flags via DeadData:

```go
func applyDeathCascade(
	killer, victim *characters.Character,
	noDeprogression bool,
	brokenLimb bool,
	brokenBodyPart string,
) {
	cfg := configs.GetBalanceConfig()
	goldFrac := 0.0
	if noDeprogression {
		goldFrac = cfg.SubGoldLossFraction.Float()
	}

	// Construct a killer ActorRef from the controller character.
	killerRef := state.ActorRef{UserId: killer.GetUserId(), MobInstanceId: killer.MobInstanceId}

	_ = victim.Life.TransitionToDead(
		life.DeadData{
			Killer:           killerRef,
			NoDeprogression:  noDeprogression,
			GoldLossFraction: goldFrac,
		},
		state.TransitionReason{Trigger: life.TriggerHealthZero, Actor: killerRef},
	)

	if brokenLimb {
		// T9: apply broken-limb buff after the cascade
		// (the buff persists across respawn — see T9 implementation)
		applyBrokenLimbBuff(victim, brokenBodyPart)
	}
}
```

`applyBrokenLimbBuff` is added in T9; for T8 leave it as a stub call.

- [ ] **Step 6: Run tests**

```
go test ./internal/hooks/... -run "TestDeathCleanup_NoDeprogression|TestDeathCorpse_PartialGold"
go test ./internal/combat/... -run "TestResolve"
go build ./...
```
Expected: PASS + clean.

- [ ] **Step 7: Commit**

```
git add internal/state/life/life.go internal/hooks/Death_PlayerCleanup.go internal/hooks/Death_PlayerCorpse.go internal/hooks/Death_PlayerCleanup_test.go internal/combat/submission_outcome.go
git commit -m "feat(life): T8 — NoDeprogression + GoldLossFraction on DeadData

Two new fields on life.DeadData drive the chunk-4d submission
outcome severity ladder via the existing Life cascade:

  - NoDeprogression: when true, Death_PlayerCleanup skips the
    stat-decay step. Subdue + cripple outcomes set this so the
    defender wakes up at the temple without losing training.
  - GoldLossFraction: when > 0, Death_PlayerCorpse skips the full
    corpse and instead transfers that fraction of the victim's gold
    to the killer. Subdue + cripple use this to model 'robbed and
    knocked out, alive but poorer.'

Submission outcome resolver (T7) is updated to use the new fields.
Lethal outcome leaves both flags at default (full death + corpse +
deprogression).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 9: Broken-limb buff

**Files:**
- Create: `_datafiles/world/dogmud/buffs/83-broken_limb.yaml`
- Modify: `internal/buffs/` (if a code-side registration is required — check existing pattern)
- Modify: `internal/combat/submission_outcome.go` (implement `applyBrokenLimbBuff`)
- Modify: `internal/combat/submission_outcome_test.go` (verify buff applies + has correct duration)

- [ ] **Step 1: Read an existing buff YAML to mirror the pattern**

Read `_datafiles/world/dogmud/buffs/82-steady_hand.yaml` (or any other recent buff) to see the schema. Note the field names: `buffid`, `name`, `description`, `roundinterval`, `roundscompleted`, `statmods`, `flags`, etc.

- [ ] **Step 2: Create the buff YAML**

Create `_datafiles/world/dogmud/buffs/83-broken_limb.yaml`:

```yaml
buffid: 83
name: Broken Limb
description: |
  A grapple submission cranked the joint past its limit. Combat
  effectiveness reduced until the bone sets and heals.
roundinterval: 0           # no per-tick action (just stat penalty)
roundscompleted: 0
roundscomplete: 900        # default duration; matches BrokenLimbBuffDuration config knob
statmods:
  - stat: combatmodifier   # adjust to match actual stat name in your codebase
    amount: -25            # interpret as percentage in the combat math layer
  - stat: defense
    amount: -10
  - stat: staminamax
    amount: -5
flags:
  - persistsafterdeath     # chunk 4d: broken limb survives the no-deprogression respawn
messages:
  startfor: |
    You feel the wrench of a broken limb. Sharp pain will follow every motion until it heals.
  endfor: |
    The ache in your limb finally fades. You can move at full strength again.
```

The exact field names (`statmods`, `flags`, `persistsafterdeath`) MUST match the existing buff schema — read `internal/buffs/buffspec.go` to confirm the field tags before authoring. Adjust names as needed.

- [ ] **Step 3: Implement `applyBrokenLimbBuff` in submission_outcome.go**

Open `internal/combat/submission_outcome.go`. Replace the stub:

```go
func applyBrokenLimbBuff(victim *characters.Character, bodyPart string) {
	if victim == nil || bodyPart == "" {
		return
	}
	// Buff 83 = broken_limb (T9 yaml)
	victim.AddBuff(83)
	// bodyPart is currently flavor only — narration uses it via
	// Position_Messaging in T11. Future per-arm tracking could
	// drive weapon-specific accuracy penalties.
	_ = bodyPart
}
```

- [ ] **Step 4: Verify the buff loads at boot**

Build + boot smoke:

```
go build ./...
go build -o /tmp/dogmud-t9.exe . && /tmp/dogmud-t9.exe > /tmp/dogmud-t9.log 2>&1 &
until grep -qE "Server Ready|panic" /tmp/dogmud-t9.log; do sleep 2; done
grep -E "buffSpec.LoadDataFiles|panic" /tmp/dogmud-t9.log | head -3
taskkill //IM dogmud-t9.exe //F
rm -f /tmp/dogmud-t9.exe /tmp/dogmud-t9.log
```

Expected: `buffSpec.LoadDataFiles() loadedCount=67` (was 66 before adding broken_limb). No panic.

- [ ] **Step 5: Add integration test**

In `internal/combat/submission_outcome_test.go`, add:

```go
func TestApplyBrokenLimbBuff_AppliesBuffWithCorrectId(t *testing.T) {
	c := characters.New()
	combat.ApplyBrokenLimbBuff(c, "arm")
	assert.True(t, c.HasBuff(83), "broken-limb buff applied")
}
```

(If `applyBrokenLimbBuff` is unexported, capitalize the exported name `ApplyBrokenLimbBuff`.)

- [ ] **Step 6: Run tests**

```
go test ./internal/combat/... -run "TestApplyBrokenLimbBuff"
go build ./...
```
Expected: PASS + clean.

- [ ] **Step 7: Commit**

```
git add _datafiles/world/dogmud/buffs/83-broken_limb.yaml internal/combat/submission_outcome.go internal/combat/submission_outcome_test.go
git commit -m "feat(buffs): T9 — broken-limb buff (id 83) + outcome resolver wiring

New buff applied by the cripple-policy submission outcome when the
sub type maps to a body part (Armbar/Omoplata=arm,
Kimura/Americana=shoulder). Stat penalties: -25% combat modifier,
-10% defense, -5% stamina max. Duration 900 rounds (~60 min of
play at 4s rounds; matches BrokenLimbBuffDuration config knob).
Flagged persistsafterdeath so it survives the no-deprogression
temple respawn.

ApplyBrokenLimbBuff helper in submission_outcome.go replaces the
T7 stub.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 10: Submission-stunned buff (1-round crit-tier stun)

**Files:**
- Create: `_datafiles/world/dogmud/buffs/84-submission_stunned.yaml`
- Modify: `internal/combat/submission_outcome.go` (implement `applyStunnedBuff`)

- [ ] **Step 1: Create the buff YAML**

Create `_datafiles/world/dogmud/buffs/84-submission_stunned.yaml`:

```yaml
buffid: 84
name: Stunned
description: |
  A particularly brutal submission left you reeling — you can't
  mount any meaningful attack or defense this round.
roundinterval: 0
roundscompleted: 0
roundscomplete: 1          # 1-round only
statmods:
  - stat: defense
    amount: -50
  - stat: combatmodifier
    amount: -75
flags: []                  # standard tick-based expiry
messages:
  startfor: |
    The submission's pressure leaves you stunned, unable to react.
  endfor: |
    You shake off the stun and regain your composure.
```

- [ ] **Step 2: Implement `applyStunnedBuff` in submission_outcome.go**

Replace the T7 stub:

```go
func applyStunnedBuff(c *characters.Character) {
	if c == nil {
		return
	}
	c.AddBuff(84) // buff 84 = submission-stunned (T10 yaml)
}
```

- [ ] **Step 3: Boot smoke**

```
go build -o /tmp/dogmud-t10.exe . && /tmp/dogmud-t10.exe > /tmp/dogmud-t10.log 2>&1 &
until grep -qE "Server Ready|panic" /tmp/dogmud-t10.log; do sleep 2; done
grep -E "buffSpec.LoadDataFiles|panic" /tmp/dogmud-t10.log | head -3
taskkill //IM dogmud-t10.exe //F
rm -f /tmp/dogmud-t10.exe /tmp/dogmud-t10.log
```

Expected: `loadedCount=68` (66 → +broken_limb → +submission_stunned).

- [ ] **Step 4: Commit**

```
git add _datafiles/world/dogmud/buffs/84-submission_stunned.yaml internal/combat/submission_outcome.go
git commit -m "feat(buffs): T10 — submission-stunned buff (id 84) for crit-tier outcomes

Applied to the recipient when a sub roll crits AND the attempter's
policy is mercy (other policies enter the death cascade so the
stun is a no-op). 1-round duration; -50% defense, -75% combat
modifier — defender gets hit for free next round.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 11: Submission attempt + resolution messaging

**Files:**
- Modify: `_datafiles/world/dogmud/grapple-messages.yaml` (add submission section)
- Modify: `internal/hooks/Position_Messaging.go` (add `fireSubmissionAttemptMessage` + `fireSubmissionResolutionMessage`)
- Modify: `internal/combat/submission_outcome.go` (call the new messaging helpers)

The narration layer. For each tier × policy × sub type, fire appropriate messages to attempter, recipient, and the rest of the room.

- [ ] **Step 1: Read the existing grapple-messages YAML structure**

Read the existing `_datafiles/world/dogmud/grapple-messages.yaml` to mirror the format. Chunk 4b shipped this with gradient / transition / stamina-warning sections. T11 adds a `submission:` section.

- [ ] **Step 2: Add submission section to grapple-messages.yaml**

Append to `_datafiles/world/dogmud/grapple-messages.yaml`:

```yaml
# ── Chunk 4d: submission tick messages ──────────────────────────
submission:
  opening:
    armbar:
      attacker: "You isolate {target}'s arm and crank — they feel the joint go past its limit!"
      target: "{attacker} isolates your arm and cranks — pain shoots through the elbow!"
      room: "{attacker} isolates {target}'s arm in a brutal armbar."
    rnc:
      attacker: "You wrap an arm around {target}'s neck and squeeze."
      target: "{attacker}'s arm tightens around your neck. The world begins to dim."
      room: "{attacker} sinks a rear-naked choke on {target}."
    triangle:
      attacker: "Your legs lock around {target}'s neck and arm — the squeeze is on."
      target: "{attacker}'s legs trap your neck and one arm. You can barely breathe."
      room: "{attacker} locks a triangle choke around {target}'s neck."
    kimura:
      attacker: "You twist {target}'s shoulder into an unnatural angle."
      target: "{attacker} twists your shoulder. The joint screams."
      room: "{attacker} cranks a kimura on {target}'s shoulder."
    americana:
      attacker: "You crank {target}'s shoulder backwards with savage pressure."
      target: "{attacker} cranks your shoulder. The joint is about to give."
      room: "{attacker} cranks an americana on {target}'s shoulder."
    omoplata:
      attacker: "You trap {target}'s shoulder with your leg — the angle is impossible."
      target: "{attacker}'s leg traps your shoulder at an impossible angle."
      room: "{attacker} locks an omoplata on {target}'s shoulder."
    anaconda:
      attacker: "You roll and snake your arms around {target}'s neck."
      target: "{attacker} rolls and your neck disappears into a vise."
      room: "{attacker} sinks an anaconda choke on {target}."

  escape_bad:
    attacker: "Your submission attempt overcommits — {target} exploits the opening and breaks free!"
    target: "You feel {attacker} overcommit on the submission. You twist free and they hit the ground!"
    room: "{attacker} overcommits on the submission — {target} escapes the grapple entirely."

  neutral:
    attacker: "Your submission attempt fails to lock in. {target} survives the round."
    target: "You feel {attacker} threaten a submission but it doesn't fully lock."
    room: "{attacker}'s submission attempt on {target} fails to fully lock in."

  outcome_mercy:
    attacker: "You ease off and let {target} up. They live to fight another day."
    target: "{attacker} eases off the submission and lets you up. You owe them your life."
    room: "{attacker} releases {target} from the submission. Mercy."

  outcome_subdue:
    attacker: "You hold the choke until {target}'s body goes limp. They're done."
    target: "{attacker} holds the choke until everything goes black."
    room: "{attacker} holds the submission until {target} loses consciousness."

  outcome_cripple_arm:
    attacker: "You crank {target}'s arm until you hear bone crack. They scream and pass out."
    target: "You hear the bone in your arm crack. The pain takes you under."
    room: "Bone cracks. {target} screams and goes limp in {attacker}'s grip."

  outcome_cripple_shoulder:
    attacker: "You crank {target}'s shoulder until it pops out of the socket."
    target: "Your shoulder pops out of the socket. The pain takes you under."
    room: "{target}'s shoulder pops sickeningly. They go limp."

  outcome_lethal:
    attacker: "You hold the lock until {target} stops twitching. They will not be getting up."
    target: "Everything fades to black. Cold."
    room: "{attacker} holds the submission past survival. {target} is dead."

  crit_flag:
    attacker: "With brutal precision — "
    target: ""
    room: ""
```

- [ ] **Step 3: Add messaging helpers in Position_Messaging.go**

Open `internal/hooks/Position_Messaging.go`. Add two new functions:

```go
// fireSubmissionAttemptMessage sends the "opening" message for a
// submission when its window first fires (before the roll resolves).
// Picks the right phrase from grapple-messages.yaml under
// submission.opening.<subtype>.
func fireSubmissionAttemptMessage(
	attempter, recipient *characters.Character,
	subType position.SubmissionType,
) {
	key := strings.ToLower(strings.ReplaceAll(subType.String(), " ", "_"))
	// Lookup template; broadcast to attempter / recipient / room
	// ... (mirror existing fire*Message patterns in this file) ...
}

// fireSubmissionResolutionMessage sends the outcome message after
// the sub resolves: escape_bad / neutral / outcome_<policy> /
// outcome_cripple_<bodypart>. Crit-flag prefix is prepended when
// result.Tier == SubTierCrit.
func fireSubmissionResolutionMessage(
	attempter, recipient *characters.Character,
	subType position.SubmissionType,
	tier combat.SubmissionTier,
	policy characters.SubmissionPolicy,
	bodyPart string,
) {
	// Pick the right yaml key based on tier + policy + bodypart
	// ... (mirror existing fire*Message patterns) ...
}
```

The exact implementation mirrors the existing fireGradientMessages /
fireTransitionMessages helpers in the same file — same load-yaml-on-init,
same broadcast-to-attempter-recipient-room pattern, same token
substitution. Reference the existing code for the precise mechanics.

- [ ] **Step 4: Wire messaging into the outcome resolver**

Open `internal/combat/submission_outcome.go`. Wire the messaging calls (or rather, register hooks the outcome resolver invokes — Position_Messaging shouldn't directly import combat to avoid cycles). Pattern: outcome resolver emits an event, Position_Messaging listens.

Cleanest approach: add a callback registration:

```go
// in submission_outcome.go
var (
	onSubmissionOpening    func(attempter, recipient *characters.Character, subType position.SubmissionType)
	onSubmissionResolution func(attempter, recipient *characters.Character, subType position.SubmissionType, tier SubmissionTier, policy characters.SubmissionPolicy, bodyPart string)
)

func RegisterSubmissionMessaging(
	opening func(attempter, recipient *characters.Character, subType position.SubmissionType),
	resolution func(attempter, recipient *characters.Character, subType position.SubmissionType, tier SubmissionTier, policy characters.SubmissionPolicy, bodyPart string),
) {
	onSubmissionOpening = opening
	onSubmissionResolution = resolution
}

// Then call onSubmissionOpening / onSubmissionResolution from the
// resolver. Position_Messaging registers the callbacks in init().
```

- [ ] **Step 5: Boot smoke**

```
go build -o /tmp/dogmud-t11.exe . && /tmp/dogmud-t11.exe > /tmp/dogmud-t11.log 2>&1 &
until grep -qE "Server Ready|panic" /tmp/dogmud-t11.log; do sleep 2; done
grep -E "Server Ready|panic" /tmp/dogmud-t11.log | head -3
taskkill //IM dogmud-t11.exe //F
rm -f /tmp/dogmud-t11.exe /tmp/dogmud-t11.log
```

Expected: ready, no panic.

- [ ] **Step 6: Commit**

```
git add _datafiles/world/dogmud/grapple-messages.yaml internal/hooks/Position_Messaging.go internal/combat/submission_outcome.go
git commit -m "feat(messaging): T11 — submission attempt + resolution narration

Adds submission-section to grapple-messages.yaml with opening,
neutral, escape_bad, outcome (mercy / subdue / cripple_arm /
cripple_shoulder / lethal), and crit_flag prefix templates. Each
sub type has its own opening phrase. Position_Messaging.go gets
fireSubmissionAttemptMessage + fireSubmissionResolutionMessage
helpers that broadcast to attempter / recipient / room.

Wiring via callback registration in submission_outcome.go to
avoid combat → hooks import cycle (combat exports
RegisterSubmissionMessaging; hooks calls it in init).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 12: MobSpec policy fields + archetype defaults

**Files:**
- Modify: `internal/mobs/mobs.go` (add SubmissionPolicy + SurrenderPolicy yaml fields to MobSpec; copy to Character at spawn with archetype fallback)
- Modify: `internal/mobs/mobs_test.go` (test YAML round-trip + archetype fallback)

- [ ] **Step 1: Add fields to MobSpec**

Open `internal/mobs/mobs.go`. Find the `Mob` struct (or `MobSpec` if separate). Add the two fields near other behavior fields (search for `AIProfile` to anchor):

```go
SubmissionPolicy string `yaml:"submission_policy,omitempty"` // chunk 4d: override archetype default; "mercy"/"subdue"/"cripple"/"lethal"
SurrenderPolicy  string `yaml:"surrender_policy,omitempty"`  // chunk 4d: override archetype default; "never"/"always"/"auto-tap-below <N>"
```

- [ ] **Step 2: Apply defaults in `newMobByIdInternal`**

Find `newMobByIdInternal` (the shallow-copy + initialization function from chunk 4b). After the existing field setup (after `mob.Character.PlayerDamage = make(map[int]int)`), add:

```go
// Chunk 4d: submission policy from YAML override or archetype default
if mob.SubmissionPolicy != "" {
	if p, ok := characters.ParseSubmissionPolicy(mob.SubmissionPolicy); ok {
		mob.Character.SubmissionPolicy = p
	} else {
		mudlog.Warn("MobSpawn", "msg", "invalid submission_policy", "mobId", mob.MobId, "value", mob.SubmissionPolicy)
		mob.Character.SubmissionPolicy = characters.DefaultSubmissionPolicyForArchetype(mob.BehaviorArchetype)
	}
} else {
	mob.Character.SubmissionPolicy = characters.DefaultSubmissionPolicyForArchetype(mob.BehaviorArchetype)
}
if mob.SurrenderPolicy != "" {
	if p, ok := characters.ParseSurrenderPolicy(mob.SurrenderPolicy); ok {
		mob.Character.SurrenderPolicy = p
	} else {
		mudlog.Warn("MobSpawn", "msg", "invalid surrender_policy", "mobId", mob.MobId, "value", mob.SurrenderPolicy)
		mob.Character.SurrenderPolicy = characters.DefaultSurrenderPolicyForArchetype(mob.BehaviorArchetype)
	}
} else {
	mob.Character.SurrenderPolicy = characters.DefaultSurrenderPolicyForArchetype(mob.BehaviorArchetype)
}
```

- [ ] **Step 3: Add tests**

In `internal/mobs/mobs_test.go`, add:

```go
func TestNewMobById_SubmissionPolicyFromArchetypeDefault(t *testing.T) {
	// Load a mob known to be a predator-archetype (e.g., a steppe wolf)
	// ... existing test setup pattern ...
	m := mobs.NewMobById(/* steppe wolf id */, /* room id */)
	assert.Equal(t, characters.PolicySubdue, m.Character.SubmissionPolicy)
}

func TestNewMobById_SubmissionPolicyYAMLOverride(t *testing.T) {
	// Find a mob with submission_policy: lethal in YAML (need to author one
	// in test fixture or use an existing override)
	// ...
}
```

(The second test needs a fixture; defer to T14 which adds the YAML overrides.)

- [ ] **Step 4: Build + test**

```
go test ./internal/mobs/... -run "TestNewMobById_SubmissionPolicy" -v
go build ./...
```
Expected: PASS + clean.

- [ ] **Step 5: Commit**

```
git add internal/mobs/mobs.go internal/mobs/mobs_test.go
git commit -m "feat(mobs): T12 — MobSpec.SubmissionPolicy + SurrenderPolicy + archetype defaults

Two new optional yaml fields on MobSpec (submission_policy,
surrender_policy). newMobByIdInternal parses them via the T2
helpers and falls through to the archetype defaults
(DefaultSubmissionPolicyForArchetype /
DefaultSurrenderPolicyForArchetype) when blank. Invalid values
log a warning and fall through to the default.

Per-mob YAML authoring for outliers (bosses, civilians, etc.)
lands in T14.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 13: Btree primitives

**Files:**
- Create: `internal/behaviortree/conditions_submission.go`
- Create: `internal/behaviortree/conditions_submission_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/behaviortree/conditions_submission_test.go`:

```go
package behaviortree_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/stretchr/testify/assert"
)

func TestMobCanSubmitTop_TrueWhenControllerInSubEligiblePos(t *testing.T) {
	// Setup a mob in Mount with InControl
	mob := setupGrappleMob(t /*, position.Mount, position.InControl*/)
	res := behaviortree.EvalCondition("mob_can_submit_top", nil, mob)
	assert.Equal(t, behaviortree.Success, res)
}

func TestMobCanSubmitTop_FalseAtStanding(t *testing.T) {
	mob := setupCharacterMob(t /* standing, no grapple */)
	res := behaviortree.EvalCondition("mob_can_submit_top", nil, mob)
	assert.Equal(t, behaviortree.Failure, res)
}

func TestMobCanSubmitBottom_TrueWhenControlledInBottomSubEligiblePos(t *testing.T) {
	mob := setupGrappleMob(t /*, position.Mount, position.Controlled*/)
	res := behaviortree.EvalCondition("mob_can_submit_bottom", nil, mob)
	assert.Equal(t, behaviortree.Success, res)
}

func TestMobSubmissionPolicyIs_MatchesEnum(t *testing.T) {
	mob := setupCharacterMob(t)
	mob.SubmissionPolicy = characters.PolicyLethal
	params := map[string]any{"policy": "lethal"}
	res := behaviortree.EvalCondition("mob_submission_policy_is", params, mob)
	assert.Equal(t, behaviortree.Success, res)

	params = map[string]any{"policy": "mercy"}
	res = behaviortree.EvalCondition("mob_submission_policy_is", params, mob)
	assert.Equal(t, behaviortree.Failure, res)
}
```

- [ ] **Step 2: Run test to verify it fails**

```
go test ./internal/behaviortree/... -run "TestMobCanSubmit"
```
Expected: FAIL — primitive not registered.

- [ ] **Step 3: Implement the primitives**

Create `internal/behaviortree/conditions_submission.go`:

```go
package behaviortree

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

func condMobCanSubmitTop(params map[string]any, ctx *EvalContext) Result {
	mob := ctx.Mob
	if mob == nil || mob.Character.Position == nil {
		return Failure
	}
	if !mob.Character.IsController() {
		return Failure
	}
	cl, _ := mob.Character.Position.ControlLevel()
	if position.IsTopSubEligible(mob.Character.Position.State(), cl) {
		return Success
	}
	return Failure
}

func condMobCanSubmitBottom(params map[string]any, ctx *EvalContext) Result {
	mob := ctx.Mob
	if mob == nil || mob.Character.Position == nil {
		return Failure
	}
	if !mob.Character.IsBeingControlled() {
		return Failure
	}
	cl, _ := mob.Character.Position.ControlLevel()
	if position.IsBottomSubEligible(mob.Character.Position.State(), cl) {
		return Success
	}
	return Failure
}

func condMobSubmissionPolicyIs(params map[string]any, ctx *EvalContext) Result {
	mob := ctx.Mob
	if mob == nil {
		return Failure
	}
	want, ok := params["policy"].(string)
	if !ok {
		return Failure
	}
	wantPolicy, ok := characters.ParseSubmissionPolicy(want)
	if !ok {
		return Failure
	}
	if mob.Character.SubmissionPolicy == wantPolicy {
		return Success
	}
	return Failure
}

func init() {
	RegisterCondition("mob_can_submit_top", condMobCanSubmitTop)
	RegisterCondition("mob_can_submit_bottom", condMobCanSubmitBottom)
	RegisterCondition("mob_submission_policy_is", condMobSubmissionPolicyIs)
}
```

(Adjust function signatures to match the existing primitive convention — read another `conditions_*.go` file in `internal/behaviortree/` to see the exact `EvalContext` shape.)

- [ ] **Step 4: Run tests + build**

```
go test ./internal/behaviortree/... -run "TestMobCanSubmit"
go build ./...
```
Expected: PASS + clean.

- [ ] **Step 5: Commit**

```
git add internal/behaviortree/conditions_submission.go internal/behaviortree/conditions_submission_test.go
git commit -m "feat(behaviortree): T13 — 3 chunk-4d btree primitives

mob_can_submit_top — true when mob is the controller of a sub-
eligible grapple. Used in archetype branches like 'don't attempt
trip when about to fire a sub.'

mob_can_submit_bottom — true when mob is the controlled side of a
grapple with bottom-attack subs at the position. Symmetric to the
top variant for reversal-sub awareness.

mob_submission_policy_is <policy> — branch on the mob's
SubmissionPolicy value (mercy/subdue/cripple/lethal). Used in
boss archetypes that want to differ in behavior based on their
preset policy.

Engine fires sub attempts automatically via Position_SubmissionTick
(T6) — these primitives exist only for AI awareness, not as
action nodes.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 14: Mob YAML policy overrides (selective)

**Files:**
- Modify: ~8-12 mob YAMLs across `_datafiles/world/dogmud/mobs/` — selective bosses, civilians, guards

- [ ] **Step 1: Identify candidate mobs**

Run a quick survey to find archetypes that warrant explicit overrides:

```
grep -l "behavior_archetype: leader" _datafiles/world/dogmud/mobs/ -r | head -5
grep -l "behavior_archetype: civilian\|behavior_archetype: merchant" _datafiles/world/dogmud/mobs/ -r | head -5
grep -l "behavior_archetype: guard\|behavior_archetype: city_watch" _datafiles/world/dogmud/mobs/ -r | head -5
```

- [ ] **Step 2: Author overrides**

For each candidate mob, decide whether the archetype default fits or an override is appropriate:

- **Bosses (leader / boss_*):**
  - `edrin` (boss_edrin): `submission_policy: lethal` + `surrender_policy: never` — final boss is lethal
  - Other bosses: keep `submission_policy: cripple` (archetype default) unless they should be lethal
- **Civilians / merchants:**
  - Most: keep `submission_policy: mercy` + `surrender_policy: always` (archetype default — no override needed)
  - Edge cases: a "hardened mercenary" civilian-class might override to `submission_policy: subdue`
- **Guards / city_watch:**
  - Most: keep archetype default
  - Specific named guards may want individual flavor

Pick 8-12 specific mobs and author the YAML edits. Example:

```yaml
# _datafiles/world/dogmud/mobs/.../boss_edrin.yaml
submission_policy: lethal
surrender_policy: never
```

Each edit is small (2 lines).

- [ ] **Step 3: Boot smoke to verify YAML loads**

```
go build -o /tmp/dogmud-t14.exe . && /tmp/dogmud-t14.exe > /tmp/dogmud-t14.log 2>&1 &
until grep -qE "Server Ready|panic" /tmp/dogmud-t14.log; do sleep 2; done
grep -E "MobSpawn.*invalid|panic" /tmp/dogmud-t14.log | head -5
taskkill //IM dogmud-t14.exe //F
rm -f /tmp/dogmud-t14.exe /tmp/dogmud-t14.log
```

Expected: ready, no panic, no "invalid submission_policy" warnings.

- [ ] **Step 4: Commit**

```
git add _datafiles/world/dogmud/mobs/
git commit -m "content(mobs): T14 — selective submission_policy / surrender_policy overrides

Per-mob YAML overrides for mobs where the archetype default isn't
quite right:
  - Boss Edrin: lethal + never-tap (final-boss flavor)
  - [other named bosses per archetype review]
  - [civilian overrides where applicable]
  - [guard overrides where applicable]

Most mobs inherit archetype defaults (T2) — no per-mob override
needed for the bulk of the bestiary.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 15: `set submission` + `set surrender` player commands

**Files:**
- Modify: `internal/usercommands/set.go` (add two subcommands + first-time-lethal confirmation)
- Modify: `internal/usercommands/set_test.go` (if exists, else create)

- [ ] **Step 1: Read existing `set.go` structure**

Open `internal/usercommands/set.go`. Identify the subcommand dispatch pattern (`switch` on the first arg or a map lookup).

- [ ] **Step 2: Add `set submission` subcommand**

Add a new case (mirror the style of existing subcommands like `set surrender` — actually you're ADDING `set surrender`, so just mirror an existing simple subcommand like `set charset`):

```go
case "submission":
	if len(args) < 2 {
		// Display current policy
		user.SendText(fmt.Sprintf("Your submission policy: <ansi fg=\"command\">%s</ansi>",
			user.Character.SubmissionPolicy))
		user.SendText("Set with: <ansi fg=\"command\">set submission &lt;mercy|subdue|cripple|lethal&gt;</ansi>")
		user.SendText("See <ansi fg=\"command\">help submission</ansi> for details.")
		return true, nil
	}
	newPolicy, ok := characters.ParseSubmissionPolicy(args[1])
	if !ok {
		user.SendText(fmt.Sprintf("Unknown submission policy: <ansi fg=\"red\">%s</ansi>", args[1]))
		user.SendText("Valid policies: mercy, subdue, cripple, lethal.")
		return true, nil
	}
	// First-time lethal confirmation
	if newPolicy == characters.PolicyLethal && user.Character.SubmissionPolicy != characters.PolicyLethal {
		// Check a flag/miscdata to see if user has confirmed lethal before
		if user.Character.MiscData["submission_lethal_confirmed"] != "1" {
			user.SendText("<ansi fg=\"yellow-bold\">WARNING:</ansi> Setting your submission policy to <ansi fg=\"red-bold\">lethal</ansi> means your successful submissions will KILL opponents — players included, with full deprogression and corpse loot.")
			user.SendText("If you're sure, run the command again to confirm.")
			user.Character.MiscData["submission_lethal_pending"] = "1"
			return true, nil
		}
		if user.Character.MiscData["submission_lethal_pending"] != "1" {
			user.Character.MiscData["submission_lethal_pending"] = "1"
			user.SendText("<ansi fg=\"yellow-bold\">WARNING:</ansi> Lethal policy will permanently kill opponents. Run again to confirm.")
			return true, nil
		}
		user.Character.MiscData["submission_lethal_confirmed"] = "1"
		delete(user.Character.MiscData, "submission_lethal_pending")
	}
	user.Character.SubmissionPolicy = newPolicy
	user.SendText(fmt.Sprintf("Submission policy set to <ansi fg=\"command\">%s</ansi>.", newPolicy))
	return true, nil

case "surrender":
	if len(args) < 2 {
		user.SendText(fmt.Sprintf("Your surrender policy: <ansi fg=\"command\">%s</ansi>",
			user.Character.SurrenderPolicy))
		user.SendText("Set with: <ansi fg=\"command\">set surrender &lt;never|always|auto-tap-below &lt;N&gt;&gt;</ansi>")
		user.SendText("See <ansi fg=\"command\">help surrender</ansi> for details.")
		return true, nil
	}
	// Join all remaining args (auto-tap-below takes a numeric arg)
	policyStr := strings.Join(args[1:], " ")
	newPolicy, ok := characters.ParseSurrenderPolicy(policyStr)
	if !ok {
		user.SendText(fmt.Sprintf("Unknown surrender policy: <ansi fg=\"red\">%s</ansi>", policyStr))
		user.SendText("Valid policies: never, always, auto-tap-below &lt;1-100&gt;.")
		return true, nil
	}
	user.Character.SurrenderPolicy = newPolicy
	user.SendText(fmt.Sprintf("Surrender policy set to <ansi fg=\"command\">%s</ansi>.", newPolicy))
	return true, nil
```

- [ ] **Step 3: Test the commands manually**

Build + boot the server, log in as smoketester, run:

```
set submission
set submission mercy
set submission lethal      # should warn
set submission lethal      # should confirm
set surrender
set surrender always
set surrender auto-tap-below 25
```

Verify outputs match expectations.

- [ ] **Step 4: Commit**

```
git add internal/usercommands/set.go
git commit -m "feat(usercommands): T15 — set submission / set surrender subcommands

set submission <mercy|subdue|cripple|lethal>
set surrender <never|always|auto-tap-below <N>>

No-args form displays current policy. Lethal requires a two-step
confirmation the first time it's set (stored in MiscData so future
sets bypass the warning). Defaults are PolicySubdue for submission
and SurrenderAutoTap @ 15% HP for surrender (per T2).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 16: Status display additions

**Files:**
- Modify: `internal/usercommands/status.go`

- [ ] **Step 1: Add policy lines to status**

Open `internal/usercommands/status.go`. Find the existing status output (search for "Submission" or look for where buffs are listed). Add two new lines:

```go
user.SendText(fmt.Sprintf(" Submission policy: %s    Surrender policy: %s",
	user.Character.SubmissionPolicy, user.Character.SurrenderPolicy))
```

If the user has the broken-limb buff (id 83), surface it prominently:

```go
if user.Character.HasBuff(83) {
	remaining := user.Character.BuffRoundsRemaining(83)
	user.SendText(fmt.Sprintf(" <ansi fg=\"red-bold\">Broken limb: %d rounds remaining</ansi>", remaining))
}
```

(`BuffRoundsRemaining` may not exist exactly — use whatever the existing buff system exposes for "rounds until expiry".)

- [ ] **Step 2: Verify manually**

Boot server, log in as smoketester, run `status`. Should now show submission/surrender policies.

- [ ] **Step 3: Commit**

```
git add internal/usercommands/status.go
git commit -m "feat(status): T16 — show submission/surrender policy + broken-limb in status

Adds two lines to the status output: current SubmissionPolicy and
SurrenderPolicy values. If broken-limb buff is active, prints a
prominent red-bold line with the remaining duration so the player
sees the cost of past defeats.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 17: New helpfiles (submission + surrender)

**Files:**
- Create: `_datafiles/world/dogmud/templates/help/submission.template`
- Create: `_datafiles/world/dogmud/templates/help/surrender.template`

- [ ] **Step 1: Author `submission.template`**

Create `_datafiles/world/dogmud/templates/help/submission.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">submission</ansi>

When you grapple an opponent on the ground, your character may lock
in a submission — a joint lock or choke that finishes the fight
without further trading of blows. Submissions fire AUTOMATICALLY
when your control is strong enough; you don't type a command.

What HAPPENS when a submission lands is determined by your
<ansi fg="command">submission policy</ansi> — set ahead of time, no
in-round prompts. Combat is too fast for that.

<ansi fg="yellow">━━━ Submission Policies ━━━</ansi>

  <ansi fg="command">mercy</ansi>    — Release the opponent cleanly. They go on living.
            Brief recovery debuff. No persistent damage.

  <ansi fg="command">subdue</ansi>   <ansi fg="white">(default)</ansi> Choke them out. They lose
            consciousness, wake up at the temple. You take some of
            their gold. No persistent injury.

  <ansi fg="command">cripple</ansi>  — Break the joint. They lose consciousness, wake up
            at the temple with a broken arm or shoulder. The debuff
            persists for around an hour of play. You take some gold.

  <ansi fg="command">lethal</ansi>   — Finish them. Full death — including for players,
            with full deprogression and corpse loot. Requires
            confirmation the first time you set it.

<ansi fg="yellow">━━━ Position Matters ━━━</ansi>

Each grapple position has its own submission types. Mount opens
armbar / americana / triangle; back-control opens the rear-naked
choke; the bottom of mount (when YOU are being controlled) opens
reversal submissions like the triangle from below. See
<ansi fg="command">help grapple</ansi> for position dynamics and
<ansi fg="command">help reach</ansi> for how weapon choice
interacts with grapples.

<ansi fg="yellow">━━━ Setting Your Policy ━━━</ansi>

  <ansi fg="command">set submission</ansi>
    Show current policy.

  <ansi fg="command">set submission &lt;mercy|subdue|cripple|lethal&gt;</ansi>
    Set policy. Lethal requires two-step confirmation.

Your policy applies to ALL submissions you successfully lock in,
against all opponents. Choose based on the kind of person your
character is.

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help surrender</ansi>, <ansi fg="command">help grapple</ansi>, <ansi fg="command">help reach</ansi>
```

- [ ] **Step 2: Author `surrender.template`**

Create `_datafiles/world/dogmud/templates/help/surrender.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">surrender</ansi>

When an opponent locks a submission on you, the engine consults
your <ansi fg="command">surrender policy</ansi> to decide whether
to send them a tap signal. This happens automatically — no in-round
prompt.

<ansi fg="yellow">━━━ The Realism Caveat ━━━</ansi>

The tap is a SIGNAL, not a guarantee. Only opponents with
<ansi fg="command">mercy</ansi> submission policy honor it.
Bandits, predators, and most hostile NPCs ignore your tap and apply
their policy anyway. A guard might subdue you instead of crippling.
A wolf will probably eat your face regardless.

<ansi fg="yellow">━━━ Surrender Policies ━━━</ansi>

  <ansi fg="command">never</ansi>                    — Never tap. Take whatever they give you.

  <ansi fg="command">always</ansi>                   — Tap immediately on any successful sub.
                             Useful for civilian / non-combatant characters.

  <ansi fg="command">auto-tap-below &lt;N&gt;</ansi>      <ansi fg="white">(default: 15)</ansi> Tap when your HP is
                             below N%. Best of both worlds — fight on while
                             you can, surrender when you're done for.

<ansi fg="yellow">━━━ Setting Your Policy ━━━</ansi>

  <ansi fg="command">set surrender</ansi>
    Show current policy.

  <ansi fg="command">set surrender never</ansi>
  <ansi fg="command">set surrender always</ansi>
  <ansi fg="command">set surrender auto-tap-below 25</ansi>

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help submission</ansi>, <ansi fg="command">help grapple</ansi>
```

- [ ] **Step 3: Verify rendering**

Boot, log in, run `help submission` and `help surrender`. Verify ansi tags render correctly, line lengths look right (under 80 chars where prose permits).

- [ ] **Step 4: Commit**

```
git add _datafiles/world/dogmud/templates/help/submission.template _datafiles/world/dogmud/templates/help/surrender.template
git commit -m "docs(helpfiles): T17 — help submission + help surrender (chunk 4d)

Player-facing explainers for the new policy system. submission.template
covers the four-tier ladder + how to set policy + position dynamics.
surrender.template covers the defender side with the realism caveat
('tap is a signal, not a guarantee — only mercy controllers honor it').

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 18: Sunset legacy `submit` command + helpers

**Files:**
- DELETE: `internal/usercommands/submit.go`
- DELETE: `internal/mobcommands/submit.go`
- DELETE: `_datafiles/world/dogmud/templates/help/submit.template`
- DELETE: `_datafiles/world/dogmud/templates/help/submit.md` (if exists)
- Modify: `internal/usercommands/usercommands.go` (unregister `submit`)
- Modify: `internal/mobcommands/mobcommands.go` (unregister `submit`)
- Modify: `internal/combat/grapple.go` (DELETE `AttemptSubmission`, `ApplySubmissionSuccess`, `ApplySubmissionFailure`, `SubmissionResult`)

The hard cutover. New system is fully in place (T1-T17), old system goes.

- [ ] **Step 1: Delete the command files**

```
git rm internal/usercommands/submit.go
git rm internal/mobcommands/submit.go
git rm _datafiles/world/dogmud/templates/help/submit.template
# also check for submit.md and remove if present
if [ -f _datafiles/world/dogmud/templates/help/submit.md ]; then
  git rm _datafiles/world/dogmud/templates/help/submit.md
fi
```

- [ ] **Step 2: Unregister from command registries**

Open `internal/usercommands/usercommands.go`. Find the `submit` entry in the command registry. Delete the line.

Same for `internal/mobcommands/mobcommands.go`.

- [ ] **Step 3: Delete the legacy helpers from grapple.go**

Open `internal/combat/grapple.go`. Find and delete:
- `func AttemptSubmission(...)` — the entire function
- `func ApplySubmissionSuccess(...)` — entire function
- `func ApplySubmissionFailure(...)` — entire function
- `type SubmissionResult struct {...}` — the struct definition

- [ ] **Step 4: Fix compile errors**

```
go build ./...
```

Read each error. For each:
- Test files referencing the deleted symbols: delete the tests OR update them to use the new API
- Other callers: most should already be migrated; any remaining ones need to be updated to call the new path (probably nothing — the legacy callers were just the legacy submit commands)

Iterate until clean.

- [ ] **Step 5: Run full test suite**

```
go test ./... -count=1 2>&1 | grep -E "^FAIL" | head -20
```

Expected: zero FAILs. Fix any breakage.

- [ ] **Step 6: Boot smoke**

```
go build -o /tmp/dogmud-t18.exe . && /tmp/dogmud-t18.exe > /tmp/dogmud-t18.log 2>&1 &
until grep -qE "Server Ready|panic" /tmp/dogmud-t18.log; do sleep 2; done
grep -E "Server Ready|panic" /tmp/dogmud-t18.log | head -3
taskkill //IM dogmud-t18.exe //F
rm -f /tmp/dogmud-t18.exe /tmp/dogmud-t18.log
```

Expected: ready, no panic.

- [ ] **Step 7: Verify command unregistered**

(Optional) start the server, log in, type `submit`. Should get "command not recognized."

- [ ] **Step 8: Commit**

```
git add -A
git commit -m "chore(combat): T18 — sunset legacy submit command + helpers

Deletes:
  - internal/usercommands/submit.go
  - internal/mobcommands/submit.go
  - _datafiles/world/dogmud/templates/help/submit.template
  - (submit.md if it existed)
  - combat.AttemptSubmission
  - combat.ApplySubmissionSuccess
  - combat.ApplySubmissionFailure
  - combat.SubmissionResult struct
  - submit entries from usercommands/mobcommands registries

The new opportunistic per-round submission system (T1-T17) is the
sole source of submission attempts. Players no longer type submit;
the engine fires attempts when control + drift conditions align.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 19: Behavior Matrix PB-301..PB-341

**Files:**
- Modify: existing test files OR create new aggregate test file
  - `internal/combat/submission_test.go` (PB-301..305 boundary)
  - `internal/combat/submission_outcome_test.go` (PB-306..313 policy resolution)
  - `internal/hooks/Position_SubmissionTick_test.go` (PB-302..303, PB-315..319, PB-341 eligibility)
  - `internal/mobs/mobs_test.go` (PB-320..321 mob defaults)
  - `internal/usercommands/set_test.go` (PB-322..323 player policy)
  - `internal/buffs/buffs_test.go` (PB-330..332 broken-limb)

Walk the matrix from the spec and verify each row. Most are unit/integration tests of pieces already implemented — this task is the consolidation that confirms each row PASSes.

- [ ] **Step 1: Author the matrix tests row-by-row**

For each row in the spec's Behavior Matrix (PB-301 through PB-341), write a test case that exercises the scenario and asserts the expected outcome. Group by test file as listed above.

Each test follows the same skeleton:
```go
func TestPB_321_BossOverrideLethal(t *testing.T) {
	mob := setupBossWithLethalPolicy(t)
	assert.Equal(t, characters.PolicyLethal, mob.Character.SubmissionPolicy)
	// + integration: a sub fires → outcome is full death cascade
}
```

- [ ] **Step 2: Run + verify**

```
go test ./... -count=1 -run "TestPB_" -v 2>&1 | grep -E "^(--- PASS|--- FAIL|^FAIL)" | head -50
```
Expected: all PB-* tests PASS or SKIP per the spec's mix. Zero FAILs.

- [ ] **Step 3: Commit**

```
git add -A
git commit -m "test(submission): T19 — Behavior Matrix PB-301..PB-341 ship

Per the chunk-4d spec's Behavior Matrix. Coverage:
  - PB-301..305 sub roll boundary tiers
  - PB-306..313 outcome by policy
  - PB-314 round-robin sub-type selection
  - PB-315..319 bottom-sub eligibility + outcome (NEW for 4d)
  - PB-320..321 mob archetype defaults + override
  - PB-322..323 player surrender policy honored only by mercy
  - PB-330..332 broken-limb buff behavior
  - PB-340 legacy submit command unregistered
  - PB-341 KneeOnBelly bottom-sub: no eligibility (no bottom subs)

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 20: Doc + helpfile audit

**Files:**
- Create: `tools/testing/audits/2026-05-18-chunk-4d-doc-helpfile-audit.md`

Same pattern as chunk-4b T22 and chunk-4c T8. Read-only survey.

- [ ] **Step 1: Run audit greps**

```
ls _datafiles/world/dogmud/templates/help/ | grep -iE "submit|surrender|grapple|combat|attack"
git --no-pager grep -nlE "AttemptSubmission|ApplySubmissionSuccess|ApplySubmissionFailure|SubmissionResult|legacy submit" -- '**/*.md' '**/*.go' 'internal/' 'docs/'
git --no-pager grep -nlE "submission|surrender|broken[ -]limb" -- '**/context.md'
```

- [ ] **Step 2: Write the audit doc**

Follow the chunk-4c audit template (`tools/testing/audits/2026-05-16-chunk-4c-doc-helpfile-audit.md`). Sections:
- Header / metadata
- Files reviewed table
- Per-helpfile findings (DELETE / UPDATE / KEEP-AS-IS verdicts + suggested copy)
- Per-context.md findings
- New documentation surface needed (broken-limb buff player-facing? submission system overview?)
- Summary counts

- [ ] **Step 3: Commit**

```
git add tools/testing/audits/2026-05-18-chunk-4d-doc-helpfile-audit.md
git commit -m "docs(audits): chunk-4d doc + helpfile audit

Survey of helpfile + context.md surface affected by chunk-4d
(commits ...). Feeds T21 (apply).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 21: Apply doc + helpfile updates

**Files:**
- Modify: helpfiles + context.md files per audit

Walk the audit and apply each UPDATE.

- [ ] **Step 1: Apply audit findings to helpfiles**

For each helpfile UPDATE flagged in T20:
- Update prose
- Verify ansi tags + 80-char wrap

Likely candidates: `grapple.template` (mention submissions + policy reference), `combat.template` (brief overview line), `attack.template` (cross-reference).

- [ ] **Step 2: Apply audit findings to context.md files**

For each context.md UPDATE flagged in T20:
- `internal/combat/context.md` — add submission system section: roll formula, tier classification, policy matrix, broken-limb interaction
- `internal/state/position/context.md` — add reference to submissions.go + the role-split mapping
- `internal/hooks/context.md` — document Position_SubmissionTick + its dependency on Position_GrappleTick's snapshot
- `internal/characters/context.md` — note SubmissionPolicy + SurrenderPolicy fields + LastDriftRoll snapshot
- `internal/state/life/context.md` — document NoDeprogression + GoldLossFraction DeadData fields

- [ ] **Step 3: Build verify (defensive)**

```
go build ./...
```

- [ ] **Step 4: Commit**

```
git add internal/ _datafiles/world/dogmud/templates/help/
git commit -m "docs: T21 — chunk-4d helpfile + context.md updates

Per T20 audit findings. Updates the helpfile network for submissions
and updates context.md files in combat, position, hooks, characters,
and state/life packages.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 22: Roadmap closeout

**Files:**
- Modify: `COMBAT_STATE_ROADMAP.md`

- [ ] **Step 1: Mark chunk 4d Done**

Open `COMBAT_STATE_ROADMAP.md`. Find the 4d row in the progress table. Change "Not started" → "Done (2026-05-18)" with summary.

- [ ] **Step 2: Add Chunk 4d — Shipped section**

After the existing "Chunk 4c — Shipped" section. Cover:
- Goal restate (one paragraph)
- What shipped (sub tick, policy substrate, 4-tier ladder, death cascade flags, two new buffs, 3 btree primitives, mob policy storage, sunset of legacy submit)
- Behavior Matrix outcome (PB-301..PB-341 tally)
- Doc work (T20/T21)
- "What's next" pointing to chunk 4e (third-party interactions)

- [ ] **Step 3: Update tail**

Change "Next: chunk 4d — Submission rework..." → "Next: chunk 4e — Third-party interaction asymmetries."

- [ ] **Step 4: Commit**

```
git add COMBAT_STATE_ROADMAP.md
git commit -m "$(cat <<'EOF'
docs(roadmap): chunk 4d (Submission Rework) Done

Opportunistic per-round submission system shipped. Submissions
fire automatically when the chunk-4b drift roll favors a side by
margin > alpha (or defender crits defense). Policy-driven outcome
resolution (mercy / subdue / cripple / lethal × never-tap / always-
tap / auto-tap-below) with no per-round prompts. Subdue + cripple
reuse the Life cascade with new NoDeprogression + GoldLossFraction
flags — defender wakes at temple without stat decay, partial gold
loss, cripple adds broken-limb buff (900-round duration, expires
naturally). Crit-tier success applies a 1-round Stunned buff (only
meaningful on mercy outcomes).

Symmetric model: bottom-attack subs (Mount-bottom triangle,
SideControl-bottom kimura, etc.) fire when the defender wins the
drift roll. Position drives sub type via top/bottom-split tables.

3 new btree primitives (mob_can_submit_top, mob_can_submit_bottom,
mob_submission_policy_is). 2 new buffs (broken_limb id 83,
submission_stunned id 84). Legacy submit command + helpers fully
sunset.

Behavior Matrix PB-301..PB-341 PASS. Chunks 0-4c regression clean.
Server boots cleanly.

Next: chunk 4e — Third-party interaction asymmetries.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Spec coverage check

| Spec section | Task(s) |
|---|---|
| Position → submission mapping (top + bottom) | T1 |
| Submission types + body parts + verbs | T1 |
| Eligibility predicates | T1 |
| SubmissionPolicy + SurrenderPolicy enums | T2 |
| Character fields + archetype defaults | T2 |
| Balance config knobs | T3 |
| Submission roll formula | T4 |
| Tier classification | T4 |
| Per-round sub tick observer | T6 |
| Drift snapshot for tick consumption | T5 |
| Policy outcome matrix | T7 |
| NoDeprogression + GoldLossFraction Life cascade | T8 |
| Broken-limb buff (duration-based) | T9 |
| Submission-stunned buff (1-round crit) | T10 |
| Submission attempt + resolution messaging | T11 |
| Mob YAML policy storage + archetype fallback | T12 |
| Btree primitives | T13 |
| Per-mob YAML overrides | T14 |
| `set submission` / `set surrender` UX | T15 |
| Status display | T16 |
| Helpfiles (submission, surrender) | T17 |
| Legacy submit sunset | T18 |
| Behavior Matrix | T19 |
| Doc audit | T20 |
| Doc updates | T21 |
| Roadmap closeout | T22 |

All spec sections covered.

## Known followups (out of chunk 4d)

- Spells / potions that accelerate broken-limb healing (4f flavor)
- Position-specific submission narration flavor variations (4f)
- Voluntary "drag captured opponent" mechanic after subdue/cripple
- Submission-as-progression: separate skill use credits from kills
- PvP cooldown-per-victim to prevent griefing
- Per-position stamina costs varying (Crucifix armbar > Mount americana)
- Standing submission entries (guillotine from clinch)
- Back-take submissions from a turtled-defender attacker
- Admin `mob heal-injury <inst>` shortcut for clearing broken-limb during tests
