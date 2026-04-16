# YAML Text Fields Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move static flavor text from JS scripts into YAML data fields on spells and buffs, eliminating ~105 JS files.

**Architecture:** Add 6 optional text fields each to `SpellData` and `BuffSpec` Go structs. A new `internal/textutil` package handles `{source}`/`{target}` token substitution. Hook call sites send YAML text before invoking JS scripts (which may no longer exist for flavor-only spells/buffs). Color wrapping uses `colorpatterns.ApplyColorPattern` directly.

**Tech Stack:** Go (structs, string replacement), YAML (data files), existing colorpatterns package

**Spec:** `docs/superpowers/specs/2026-04-11-yaml-text-fields-design.md`

---

### Task 1: Token Substitution Package

**Files:**
- Create: `internal/textutil/tokens.go`
- Create: `internal/textutil/tokens_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/textutil/tokens_test.go`:

```go
package textutil

import "testing"

func TestSubstituteTokens_AllTokens(t *testing.T) {
	ctx := TokenContext{
		SourceName:      `<ansi fg="yellow">Kael</ansi>`,
		SourcePlainName: `Kael`,
		TargetName:      `<ansi fg="red">Goblin</ansi>`,
		TargetPlainName: `Goblin`,
	}
	input := `{source} hurls a bolt at {target}. {source_plain}'s eyes glow. {target_plain} staggers.`
	expected := `<ansi fg="yellow">Kael</ansi> hurls a bolt at <ansi fg="red">Goblin</ansi>. Kael's eyes glow. Goblin staggers.`
	result := SubstituteTokens(input, ctx)
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestSubstituteTokens_EmptyTarget(t *testing.T) {
	ctx := TokenContext{
		SourceName:      `Kael`,
		SourcePlainName: `Kael`,
	}
	input := `{source} channels energy at {target}.`
	expected := `Kael channels energy at .`
	result := SubstituteTokens(input, ctx)
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestSubstituteTokens_NoTokens(t *testing.T) {
	ctx := TokenContext{SourceName: `Kael`}
	input := `Energy crackles in the air.`
	result := SubstituteTokens(input, ctx)
	if result != input {
		t.Errorf("got %q, want %q", result, input)
	}
}

func TestSubstituteTokens_EmptyString(t *testing.T) {
	ctx := TokenContext{}
	result := SubstituteTokens("", ctx)
	if result != "" {
		t.Errorf("got %q, want empty", result)
	}
}

func TestValidateTokens_KnownTokens(t *testing.T) {
	warnings := ValidateTokens(`{source} attacks {target}`)
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
}

func TestValidateTokens_UnknownToken(t *testing.T) {
	warnings := ValidateTokens(`{source} attacks {targat}`)
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
	if warnings[0] != `unknown token: {targat}` {
		t.Errorf("got %q", warnings[0])
	}
}

func TestValidateTokens_EmptyString(t *testing.T) {
	warnings := ValidateTokens("")
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd internal/textutil && go test -v`
Expected: compilation failure — package and types don't exist yet

- [ ] **Step 3: Write the implementation**

Create `internal/textutil/tokens.go`:

```go
package textutil

import (
	"regexp"
	"strings"
)

// TokenContext holds actor names for substitution in YAML text fields.
type TokenContext struct {
	SourceName      string // ANSI-tagged display name
	SourcePlainName string // Plain name (for possessives)
	TargetName      string // ANSI-tagged display name (empty if no target)
	TargetPlainName string // Plain name (empty if no target)
}

// SubstituteTokens replaces known tokens in text with values from ctx.
// Unknown tokens are left as-is. Empty string input returns empty string.
func SubstituteTokens(text string, ctx TokenContext) string {
	if text == "" {
		return ""
	}
	r := strings.NewReplacer(
		`{source}`, ctx.SourceName,
		`{target}`, ctx.TargetName,
		`{source_plain}`, ctx.SourcePlainName,
		`{target_plain}`, ctx.TargetPlainName,
	)
	return r.Replace(text)
}

var tokenPattern = regexp.MustCompile(`\{[a-z_]+\}`)

var knownTokens = map[string]bool{
	`{source}`:       true,
	`{target}`:       true,
	`{source_plain}`: true,
	`{target_plain}`: true,
}

// ValidateTokens scans text for {token} patterns and returns warnings
// for any that are not in the known set.
func ValidateTokens(text string) []string {
	if text == "" {
		return nil
	}
	var warnings []string
	matches := tokenPattern.FindAllString(text, -1)
	for _, m := range matches {
		if !knownTokens[m] {
			warnings = append(warnings, "unknown token: "+m)
		}
	}
	return warnings
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd internal/textutil && go test -v`
Expected: all 7 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/textutil/tokens.go internal/textutil/tokens_test.go
git commit -m "feat: add textutil package for YAML text token substitution

Supports {source}, {target}, {source_plain}, {target_plain} tokens.
Includes ValidateTokens for load-time typo detection.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: Add Text Fields to SpellData

**Files:**
- Modify: `internal/spells/spells.go:18-38` (SpellData struct)
- Modify: `internal/spells/spells.go:166-175` (Validate method)

- [ ] **Step 1: Add 6 text fields to SpellData struct**

In `internal/spells/spells.go`, after the `QuestRequired` field (line 38), add:

```go
	// YAML text fields — flavor text sent by the engine (replaces JS messaging)
	CastUserText  string `yaml:"cast_user_text,omitempty"`
	CastRoomText  string `yaml:"cast_room_text,omitempty"`
	WaitUserText  string `yaml:"wait_user_text,omitempty"`
	WaitRoomText  string `yaml:"wait_room_text,omitempty"`
	MagicUserText string `yaml:"magic_user_text,omitempty"`
	MagicRoomText string `yaml:"magic_room_text,omitempty"`
```

- [ ] **Step 2: Add token validation to Validate()**

In `internal/spells/spells.go`, add import for `textutil` and update `Validate()`:

Add to imports:
```go
"github.com/GoMudEngine/GoMud/internal/textutil"
"github.com/GoMudEngine/GoMud/internal/mudlog"
```

Replace the `Validate()` method (lines 166-175) with:

```go
func (s *SpellData) Validate() error {

	if s.Difficulty < 0 {
		s.Difficulty = 0
	} else if s.Difficulty > 100 {
		s.Difficulty = 100
	}

	// Validate YAML text tokens
	for _, text := range []string{
		s.CastUserText, s.CastRoomText,
		s.WaitUserText, s.WaitRoomText,
		s.MagicUserText, s.MagicRoomText,
	} {
		for _, w := range textutil.ValidateTokens(text) {
			mudlog.Warn("Spell.Validate", "spellId", s.SpellId, "warning", w)
		}
	}

	return nil
}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/spells/...`
Expected: compiles without errors

- [ ] **Step 4: Commit**

```bash
git add internal/spells/spells.go
git commit -m "feat: add YAML text fields to SpellData struct

Six optional fields (cast/wait/magic × user/room) for flavor text.
Token validation warns on unknown {tokens} at load time.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: Add Text Fields to BuffSpec

**Files:**
- Modify: `internal/buffs/buffspec.go:86-97` (BuffSpec struct)
- Modify: `internal/buffs/buffspec.go:182-198` (Validate method)

- [ ] **Step 1: Add 6 text fields to BuffSpec struct**

In `internal/buffs/buffspec.go`, after the `Flags` field (line 97), add:

```go
	// YAML text fields — flavor text sent by the engine (replaces JS messaging)
	StartUserText   string `yaml:"start_user_text,omitempty"`
	StartRoomText   string `yaml:"start_room_text,omitempty"`
	TriggerUserText string `yaml:"trigger_user_text,omitempty"`
	TriggerRoomText string `yaml:"trigger_room_text,omitempty"`
	EndUserText     string `yaml:"end_user_text,omitempty"`
	EndRoomText     string `yaml:"end_room_text,omitempty"`
```

- [ ] **Step 2: Add token validation to Validate()**

Add imports for `textutil` and `mudlog`, then insert token validation
at the top of `Validate()` (after the opening brace, before the
`BuffId == 0` check):

```go
	// Validate YAML text tokens
	for _, text := range []string{
		b.StartUserText, b.StartRoomText,
		b.TriggerUserText, b.TriggerRoomText,
		b.EndUserText, b.EndRoomText,
	} {
		for _, w := range textutil.ValidateTokens(text) {
			mudlog.Warn("Buff.Validate", "buffId", b.BuffId, "warning", w)
		}
	}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/buffs/...`
Expected: compiles without errors

- [ ] **Step 4: Commit**

```bash
git add internal/buffs/buffspec.go
git commit -m "feat: add YAML text fields to BuffSpec struct

Six optional fields (start/trigger/end × user/room) for flavor text.
Token validation warns on unknown {tokens} at load time.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: Spell Text Sending Helper

**Files:**
- Create: `internal/textutil/spelltext.go`

This helper sends YAML text with token substitution and color wrapping,
callable from any hook site. It avoids import cycles by accepting
primitive send functions rather than importing `users`/`rooms` directly.

- [ ] **Step 1: Create the spell/buff text sender**

Create `internal/textutil/spelltext.go`:

```go
package textutil

import (
	"github.com/GoMudEngine/GoMud/internal/colorpatterns"
)

// SendTextConfig holds the messaging functions for a text send operation.
// This avoids import cycles — callers pass in their send functions.
type SendTextConfig struct {
	UserSendFunc func(msg string)             // sends to the acting user/mob owner
	RoomSendFunc func(msg string, skip ...int) // sends to the room, excluding user
	ExcludeId    int                           // user ID to exclude from room messages
}

// SendPhaseText substitutes tokens, applies color wrapping, and sends
// user/room text for a spell or buff phase. Empty text = no message.
func SendPhaseText(userText, roomText string, ctx TokenContext, colorName string, cfg SendTextConfig) {
	if userText != "" && cfg.UserSendFunc != nil {
		msg := SubstituteTokens(userText, ctx)
		if colorName != "" {
			msg = colorpatterns.ApplyColorPattern(msg, colorName, colorpatterns.Stretch)
		}
		cfg.UserSendFunc(msg)
	}
	if roomText != "" && cfg.RoomSendFunc != nil {
		msg := SubstituteTokens(roomText, ctx)
		if colorName != "" {
			msg = colorpatterns.ApplyColorPattern(msg, colorName, colorpatterns.Stretch)
		}
		cfg.RoomSendFunc(msg, cfg.ExcludeId)
	}
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/textutil/...`
Expected: compiles without errors

- [ ] **Step 3: Commit**

```bash
git add internal/textutil/spelltext.go
git commit -m "feat: add SendPhaseText helper for YAML text dispatch

Handles token substitution, color wrapping, and user/room message
dispatch in one call. Uses callback functions to avoid import cycles.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: Wire Spell Cast Phase Text

**Files:**
- Modify: `internal/usercommands/skill.cast.go:216-226`
- Modify: `internal/mobcommands/cast.go:46-56`
- Modify: `internal/mobcommands/aid.go:52-57`

The cast phase has three call sites: player cast, mob cast, and mob aid.
YAML text sends before the JS `onCast` call. If `onCast` returns false
(cancels the cast), the text was already sent — this is acceptable because
the text describes the attempt ("You channel energy..."), not the result.

- [ ] **Step 1: Wire player cast (skill.cast.go)**

In `internal/usercommands/skill.cast.go`, add import:
```go
"github.com/GoMudEngine/GoMud/internal/textutil"
```

Before the `onCast` script call (line 216), insert YAML text sending.
Find the existing block:

```go
	// 13. Fire onCast spell script (if present) — can cancel the cast.
	spellAggro := characters.SpellAggroInfo{
```

Insert before it:

```go
	// 12b. Send YAML cast text (if defined).
	if spellInfo.CastUserText != "" || spellInfo.CastRoomText != "" {
		room := rooms.LoadRoom(user.Character.RoomId)
		tCtx := textutil.TokenContext{
			SourceName:      user.Character.GetCharacterName(true),
			SourcePlainName: user.Character.GetCharacterName(false),
		}
		// Resolve target name for single-target spells
		if len(result.TargetUserIds) > 0 {
			if tUser := users.GetByUserId(result.TargetUserIds[0]); tUser != nil {
				tCtx.TargetName = tUser.Character.GetCharacterName(true)
				tCtx.TargetPlainName = tUser.Character.GetCharacterName(false)
			}
		} else if len(result.TargetMobInstanceIds) > 0 {
			if tMob := mobs.GetInstance(result.TargetMobInstanceIds[0]); tMob != nil {
				tCtx.TargetName = tMob.Character.GetCharacterName(true)
				tCtx.TargetPlainName = tMob.Character.GetCharacterName(false)
			}
		}
		cfg := textutil.SendTextConfig{
			UserSendFunc: func(msg string) { user.SendText(msg) },
			RoomSendFunc: func(msg string, skip ...int) {
				if room != nil {
					room.SendText(msg, skip...)
				}
			},
			ExcludeId: user.UserId,
		}
		textutil.SendPhaseText(spellInfo.CastUserText, spellInfo.CastRoomText, tCtx, "pink", cfg)
	}

```

- [ ] **Step 2: Wire mob cast (cast.go)**

In `internal/mobcommands/cast.go`, add import for `textutil`.

Before the `onCast` script call (line 46 area), insert YAML text sending:

```go
	// Send YAML cast text (if defined).
	if spellInfo.CastUserText != "" || spellInfo.CastRoomText != "" {
		room := rooms.LoadRoom(mob.Character.RoomId)
		tCtx := textutil.TokenContext{
			SourceName:      mob.Character.GetCharacterName(true),
			SourcePlainName: mob.Character.GetCharacterName(false),
		}
		if len(result.TargetUserIds) > 0 {
			if tUser := users.GetByUserId(result.TargetUserIds[0]); tUser != nil {
				tCtx.TargetName = tUser.Character.GetCharacterName(true)
				tCtx.TargetPlainName = tUser.Character.GetCharacterName(false)
			}
		} else if len(result.TargetMobInstanceIds) > 0 {
			if tMob := mobs.GetInstance(result.TargetMobInstanceIds[0]); tMob != nil {
				tCtx.TargetName = tMob.Character.GetCharacterName(true)
				tCtx.TargetPlainName = tMob.Character.GetCharacterName(false)
			}
		}
		cfg := textutil.SendTextConfig{
			RoomSendFunc: func(msg string, skip ...int) {
				if room != nil {
					room.SendText(msg, skip...)
				}
			},
		}
		textutil.SendPhaseText("", spellInfo.CastRoomText, tCtx, "pink", cfg)
	}
```

Note: mob casts only send room text (no `UserSendFunc` — mobs don't
receive player messages).

- [ ] **Step 3: Wire mob aid (aid.go)**

In `internal/mobcommands/aid.go`, add import for `textutil`.
Apply the same pattern as mob cast, before the `onCast` script call
(line 55 area). Same code as Step 2 — mob casts only send room text.

- [ ] **Step 4: Verify compilation**

Run: `go build ./internal/usercommands/... ./internal/mobcommands/...`
Expected: compiles without errors

- [ ] **Step 5: Commit**

```bash
git add internal/usercommands/skill.cast.go internal/mobcommands/cast.go internal/mobcommands/aid.go
git commit -m "feat: wire YAML text sending for spell cast phase

Sends cast_user_text/cast_room_text before JS onCast at all three
call sites (player cast, mob cast, mob aid).

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: Wire Spell Wait and Magic Phase Text

**Files:**
- Modify: `internal/hooks/NewRound_DoCombat_helpers.go:263-270, 349-356`
- Modify: `internal/hooks/spell_resolution.go:134-141`

- [ ] **Step 1: Wire wait phase text (two call sites)**

In `internal/hooks/NewRound_DoCombat_helpers.go`, add import for `textutil`.

There are two `onWait` call sites. Before each `scripting.TrySpellScriptEvent("onWait", ...)` call, insert YAML text sending.

The pattern is the same at both sites. Before each `onWait` block, insert:

```go
		// Send YAML wait text (if defined).
		if spellInfo.WaitUserText != "" || spellInfo.WaitRoomText != "" {
			tCtx := textutil.TokenContext{
				SourceName:      user.Character.GetCharacterName(true),
				SourcePlainName: user.Character.GetCharacterName(false),
			}
			cfg := textutil.SendTextConfig{
				UserSendFunc: func(msg string) { user.SendText(msg) },
				RoomSendFunc: func(msg string, skip ...int) {
					room := rooms.LoadRoom(user.Character.RoomId)
					if room != nil {
						room.SendText(msg, skip...)
					}
				},
				ExcludeId: user.UserId,
			}
			textutil.SendPhaseText(spellInfo.WaitUserText, spellInfo.WaitRoomText, tCtx, "pink", cfg)
		}
```

Note: Check that `spellInfo` is accessible at each site. It should be —
the `cs` (CastingState) provides the spell ID and `spells.GetSpell(cs.SpellId)`
is already called nearby. If `spellInfo` isn't in scope, add:
```go
		spellInfo := spells.GetSpell(cs.SpellId)
```

- [ ] **Step 2: Wire magic phase text**

In `internal/hooks/spell_resolution.go`, add import for `textutil`.

Before the `onMagic` script call (line 134), insert:

```go
	// Send YAML magic text (if defined).
	spellInfo := spells.GetSpell(cs.SpellId)
	if spellInfo != nil && (spellInfo.MagicUserText != "" || spellInfo.MagicRoomText != "") {
		tCtx := textutil.TokenContext{
			SourceName:      user.Character.GetCharacterName(true),
			SourcePlainName: user.Character.GetCharacterName(false),
		}
		// Resolve target for single-target spells
		if len(cs.TargetUserIds) > 0 {
			if tUser := users.GetByUserId(cs.TargetUserIds[0]); tUser != nil {
				tCtx.TargetName = tUser.Character.GetCharacterName(true)
				tCtx.TargetPlainName = tUser.Character.GetCharacterName(false)
			}
		} else if len(cs.TargetMobInstanceIds) > 0 {
			if tMob := mobs.GetInstance(cs.TargetMobInstanceIds[0]); tMob != nil {
				tCtx.TargetName = tMob.Character.GetCharacterName(true)
				tCtx.TargetPlainName = tMob.Character.GetCharacterName(false)
			}
		}
		cfg := textutil.SendTextConfig{
			UserSendFunc: func(msg string) { user.SendText(msg) },
			RoomSendFunc: func(msg string, skip ...int) {
				room := rooms.LoadRoom(user.Character.RoomId)
				if room != nil {
					room.SendText(msg, skip...)
				}
			},
			ExcludeId: user.UserId,
		}
		textutil.SendPhaseText(spellInfo.MagicUserText, spellInfo.MagicRoomText, tCtx, "pink", cfg)
	}
```

Note: Check whether `spellInfo` is already in scope from earlier in the
function. If so, use that variable instead of re-fetching.

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/hooks/...`
Expected: compiles without errors

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/NewRound_DoCombat_helpers.go internal/hooks/spell_resolution.go
git commit -m "feat: wire YAML text sending for spell wait and magic phases

Sends wait/magic text before JS onWait/onMagic calls. Target names
resolved for single-target spells in magic phase.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 7: Wire Buff Lifecycle Text

**Files:**
- Modify: `internal/hooks/Buff_ApplyBuffs.go:62-66`
- Modify: `internal/hooks/NewRound_UserRoundTick.go:140-162`
- Modify: `internal/hooks/NewTurn_PruneBuffs.go:38-47, 66-69`

Buff text uses `{source}` for the buff holder (no `{target}` — buffs
don't have targets). Color is `cyan` (matching JS buff text wrapping).

- [ ] **Step 1: Wire buff start text**

In `internal/hooks/Buff_ApplyBuffs.go`, add import for `textutil` and
`buffs` (if not already imported).

Before the `onStart` script call (line 62), insert:

```go
		// Send YAML start text (if defined).
		buffSpec := buffs.GetBuffSpec(evt.BuffId)
		if buffSpec != nil && (buffSpec.StartUserText != "" || buffSpec.StartRoomText != "") {
			var charName, charPlainName string
			var sendFunc func(string)
			var roomId, excludeId int

			if evt.UserId != 0 {
				if u := users.GetByUserId(evt.UserId); u != nil {
					charName = u.Character.GetCharacterName(true)
					charPlainName = u.Character.GetCharacterName(false)
					roomId = u.Character.RoomId
					excludeId = u.UserId
					sendFunc = func(msg string) { u.SendText(msg) }
				}
			} else if evt.MobInstanceId != 0 {
				if m := mobs.GetInstance(evt.MobInstanceId); m != nil {
					charName = m.Character.GetCharacterName(true)
					charPlainName = m.Character.GetCharacterName(false)
					roomId = m.Character.RoomId
				}
			}

			if charName != "" {
				tCtx := textutil.TokenContext{
					SourceName:      charName,
					SourcePlainName: charPlainName,
				}
				cfg := textutil.SendTextConfig{
					UserSendFunc: sendFunc,
					RoomSendFunc: func(msg string, skip ...int) {
						if r := rooms.LoadRoom(roomId); r != nil {
							r.SendText(msg, skip...)
						}
					},
					ExcludeId: excludeId,
				}
				textutil.SendPhaseText(buffSpec.StartUserText, buffSpec.StartRoomText, tCtx, "cyan", cfg)
			}
		}
```

- [ ] **Step 2: Wire buff trigger text**

In `internal/hooks/NewRound_UserRoundTick.go`, add import for `textutil`
and `buffs`.

Inside the triggered buffs loop (after `buff.Expired()` check, before
the `TryBuffScriptEvent("onTrigger", ...)` call on line 153), insert:

```go
		// Send YAML trigger text (if defined).
		buffSpec := buffs.GetBuffSpec(buff.BuffId)
		if buffSpec != nil && (buffSpec.TriggerUserText != "" || buffSpec.TriggerRoomText != "") {
			tCtx := textutil.TokenContext{
				SourceName:      user.Character.GetCharacterName(true),
				SourcePlainName: user.Character.GetCharacterName(false),
			}
			cfg := textutil.SendTextConfig{
				UserSendFunc: func(msg string) { user.SendText(msg) },
				RoomSendFunc: func(msg string, skip ...int) {
					if r := rooms.LoadRoom(user.Character.RoomId); r != nil {
						r.SendText(msg, skip...)
					}
				},
				ExcludeId: user.UserId,
			}
			textutil.SendPhaseText(buffSpec.TriggerUserText, buffSpec.TriggerRoomText, tCtx, "cyan", cfg)
		}
```

- [ ] **Step 3: Wire buff end text (player and mob)**

In `internal/hooks/NewTurn_PruneBuffs.go`, add import for `textutil`
and `buffs`.

**Player buffs** — inside the player prune loop (line 39), before the
existing `TryBuffScriptEvent("onEnd", ...)` call:

```go
		// Send YAML end text (if defined).
		buffSpec := buffs.GetBuffSpec(buffInfo.BuffId)
		if buffSpec != nil && (buffSpec.EndUserText != "" || buffSpec.EndRoomText != "") {
			tCtx := textutil.TokenContext{
				SourceName:      user.Character.GetCharacterName(true),
				SourcePlainName: user.Character.GetCharacterName(false),
			}
			cfg := textutil.SendTextConfig{
				UserSendFunc: func(msg string) { user.SendText(msg) },
				RoomSendFunc: func(msg string, skip ...int) {
					if r := rooms.LoadRoom(user.Character.RoomId); r != nil {
						r.SendText(msg, skip...)
					}
				},
				ExcludeId: user.UserId,
			}
			textutil.SendPhaseText(buffSpec.EndUserText, buffSpec.EndRoomText, tCtx, "cyan", cfg)
		}
```

**Mob buffs** — inside the mob prune loop (line 67), before the existing
`TryBuffScriptEvent("onEnd", ...)` call:

```go
		// Send YAML end text (if defined).
		buffSpec := buffs.GetBuffSpec(buffInfo.BuffId)
		if buffSpec != nil && (buffSpec.EndRoomText != "") {
			tCtx := textutil.TokenContext{
				SourceName:      mob.Character.GetCharacterName(true),
				SourcePlainName: mob.Character.GetCharacterName(false),
			}
			cfg := textutil.SendTextConfig{
				RoomSendFunc: func(msg string, skip ...int) {
					if r := rooms.LoadRoom(mob.Character.RoomId); r != nil {
						r.SendText(msg, skip...)
					}
				},
			}
			textutil.SendPhaseText("", buffSpec.EndRoomText, tCtx, "cyan", cfg)
		}
```

- [ ] **Step 4: Verify compilation**

Run: `go build ./internal/hooks/...`
Expected: compiles without errors

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/Buff_ApplyBuffs.go internal/hooks/NewRound_UserRoundTick.go internal/hooks/NewTurn_PruneBuffs.go
git commit -m "feat: wire YAML text sending for buff start/trigger/end phases

Sends buff text before JS callbacks at all lifecycle points.
Handles both player and mob buff text. Color: cyan.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 8: Proof of Concept — Migrate Conviction Surge

**Files:**
- Modify: `_datafiles/world/dogmud/spells/conviction-surge.yaml`
- Delete: `_datafiles/world/dogmud/spells/conviction-surge.js`
- Modify: `_datafiles/world/dogmud/buffs/26-conviction_surge.yaml`
- Delete: `_datafiles/world/dogmud/buffs/26-conviction_surge.js`

- [ ] **Step 1: Add text fields to spell YAML**

Edit `_datafiles/world/dogmud/spells/conviction-surge.yaml`, appending:

```yaml
cast_user_text: "You channel conviction into empowering energy."
cast_room_text: "{source} gathers conviction, a fierce glow building."
```

(No wait or magic text — the original JS had empty `onWait` and `onMagic`.)

- [ ] **Step 2: Delete spell JS file**

```bash
rm _datafiles/world/dogmud/spells/conviction-surge.js
```

- [ ] **Step 3: Add text fields to buff YAML**

Edit `_datafiles/world/dogmud/buffs/26-conviction_surge.yaml`, appending:

```yaml
start_user_text: "Conviction surges through your limbs, empowering your strikes."
end_user_text: "The surge of conviction fades from your limbs."
```

(No trigger or room text — the original JS had empty `onTrigger` and
no room messages.)

- [ ] **Step 4: Delete buff JS file**

```bash
rm _datafiles/world/dogmud/buffs/26-conviction_surge.js
```

- [ ] **Step 5: Manual smoke test**

Start the server, log in, and:
1. Cast `conviction surge` on self or target
2. Verify cast text appears (pink color, correct name substitution)
3. Verify buff start text appears (cyan color)
4. Wait for buff to expire
5. Verify buff end text appears
6. Verify no errors in server log
7. Verify no double messages

- [ ] **Step 6: Commit**

```bash
git add _datafiles/world/dogmud/spells/conviction-surge.yaml _datafiles/world/dogmud/buffs/26-conviction_surge.yaml
git rm _datafiles/world/dogmud/spells/conviction-surge.js _datafiles/world/dogmud/buffs/26-conviction_surge.js
git commit -m "feat: migrate conviction-surge spell+buff to YAML text fields

Proof of concept — first spell/buff migrated from JS to YAML text.
Deletes both JS files, adds cast/start/end text to YAML definitions.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 9: Batch Migrate Flavor-Only Spells

**Files:**
- Modify: 36 spell YAML files (add text fields)
- Delete: 36 spell JS files

For each JS file, extract the text strings from `onCast`, add them as
`cast_user_text` / `cast_room_text` to the YAML, then delete the JS.

**Spell list** (all in `_datafiles/world/dogmud/spells/`):

```
blood-boil, chrysalis-cocoon, chrysalis-glow, chrysalis-haste,
chrysalis-regeneration, cleansing-wave, communion-of-flesh,
conviction-armor, conviction-barrage, conviction-spike,
conviction-ward, empathic-bond, empathic-shroud, hemorrhagic-burst,
hemorrhagic-wave, iron-will, kinetic-hurl, kinetic-shove,
mass-mend, mend-all, mend-wounds, mind-fog, mind-spike,
mutation-catalyst, nerve-disruption, neural-stun, neural-toxin,
psychic-anchor, pyretic-surge, sensory-overload, sensory-veil,
skill-attunement, sparks, synaptic-overload, veil-rend, veil-sight,
vital-surge
```

- [ ] **Step 1: For each spell, read the JS, extract text, add to YAML**

Pattern for each file:
1. Read the JS file — find the `SendUserMessage(...)` and
   `SendRoomMessage(...)` string arguments in `onCast`
2. Convert room text: replace `sourceActor.GetCharacterName(true)+' ...'`
   with `{source} ...`
3. Add `cast_user_text:` and `cast_room_text:` to the YAML file
4. If `onWait` has text (rare), add `wait_user_text:` / `wait_room_text:`
5. Delete the JS file

Example transformation (blood-boil.js):
```
JS:  SendUserMessage(sourceActor.UserId(), 'You focus on the target\'s blood, willing it to boil.');
     SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' extends a trembling hand, eyes burning with intent.', sourceActor.UserId());

YAML: cast_user_text: "You focus on the target's blood, willing it to boil."
      cast_room_text: "{source} extends a trembling hand, eyes burning with intent."
```

- [ ] **Step 2: Delete all 36 JS files**

```bash
cd _datafiles/world/dogmud/spells
rm blood-boil.js chrysalis-cocoon.js chrysalis-glow.js chrysalis-haste.js \
   chrysalis-regeneration.js cleansing-wave.js communion-of-flesh.js \
   conviction-armor.js conviction-barrage.js conviction-spike.js \
   conviction-ward.js empathic-bond.js empathic-shroud.js \
   hemorrhagic-burst.js hemorrhagic-wave.js iron-will.js kinetic-hurl.js \
   kinetic-shove.js mass-mend.js mend-all.js mend-wounds.js mind-fog.js \
   mind-spike.js mutation-catalyst.js nerve-disruption.js neural-stun.js \
   neural-toxin.js psychic-anchor.js pyretic-surge.js sensory-overload.js \
   sensory-veil.js skill-attunement.js sparks.js synaptic-overload.js \
   veil-rend.js veil-sight.js vital-surge.js
```

- [ ] **Step 3: Verify server starts without errors**

Run server, check logs for any missing script warnings or YAML parse
errors. Cast 2-3 of the migrated spells to verify text appears.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/spells/*.yaml
git rm _datafiles/world/dogmud/spells/blood-boil.js _datafiles/world/dogmud/spells/chrysalis-cocoon.js [... all 36 files]
git commit -m "feat: migrate 36 flavor-only spells from JS to YAML text fields

Batch migration of all DOGMud spells that only contained flavor text.
JS files deleted, cast text moved to YAML cast_user_text/cast_room_text.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 10: Batch Migrate Flavor-Only Buffs

**Files:**
- Modify: 11 buff YAML files (add text fields)
- Delete: 11 buff JS files

**Buff list** (all in `_datafiles/world/dogmud/buffs/`):

```
26-conviction_surge (already done in Task 8)
27-iron_will, 28-chrysalis_haste, 30-nerve_disruption,
31-empathic_shroud, 34-skill_attunement, 35-mutation_catalyst,
36-psychic_anchor, 37-sensory_overload, 38-conviction_armor,
41-mind_fog
```

- [ ] **Step 1: For each buff, read JS, extract text, add to YAML**

Pattern for each file:
1. Read `onStart` — extract `SendUserMessage` text → `start_user_text`
2. Read `onStart` — extract `SendRoomMessage` text (if any) → `start_room_text`
3. Read `onEnd` — extract text → `end_user_text` / `end_room_text`
4. Convert `actor.GetCharacterName(true)` → `{source}`
5. Delete JS file

- [ ] **Step 2: Delete all 10 JS files** (26 already done)

```bash
cd _datafiles/world/dogmud/buffs
rm 27-iron_will.js 28-chrysalis_haste.js 30-nerve_disruption.js \
   31-empathic_shroud.js 34-skill_attunement.js 35-mutation_catalyst.js \
   36-psychic_anchor.js 37-sensory_overload.js 38-conviction_armor.js \
   41-mind_fog.js
```

- [ ] **Step 3: Verify server starts, apply a buff, verify text**

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/buffs/*.yaml
git rm _datafiles/world/dogmud/buffs/27-iron_will.js [... all 10 files]
git commit -m "feat: migrate 10 flavor-only buffs from JS to YAML text fields

Batch migration of all DOGMud buffs with empty onTrigger.
JS deleted, start/end text moved to YAML fields.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 11: Migrate Complex Spell Flavor Text

**Files:**
- Modify: ~19 complex spell YAML + JS files

Complex spells keep their JS for logic but gain YAML text for cast/wait
phases, and their JS `onCast`/`onWait` `SendUserMessage`/`SendRoomMessage`
calls get removed.

**Spell list:**
```
chrysalis-aid, chrysalis-construct, charm,
raise-skeleton, raise-zombie, raise-wraith, raise-spectre,
raise-golem, raise-vampire,
summon-hive-swarm, summon-steppe-spirit,
conjure-water, conjure-earth, conjure-air, conjure-fire, conjure-magma,
fold-anchor, fold-recall, identify, purge-affliction
```

- [ ] **Step 1: For each spell, extract cast/wait text to YAML**

1. Read the JS `onCast` function
2. Extract `SendUserMessage` / `SendRoomMessage` text
3. Add as `cast_user_text` / `cast_room_text` to YAML
4. Remove the `Send*` calls from JS `onCast` (keep any logic like
   companion cap checks, component validation)
5. If `onCast` is now empty (only had messaging), delete the function
6. Repeat for `onWait` if it has text

**Important:** Do NOT touch `onMagic` — it contains the actual logic.

- [ ] **Step 2: Verify compilation and test**

Start server, cast one of each type:
- A companion summon (verify cap check still works)
- Charm (verify opposed roll still works)
- Fold recall (verify teleport still works)

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/spells/*.yaml _datafiles/world/dogmud/spells/*.js
git commit -m "feat: extract flavor text from complex spells to YAML

Cast/wait text moved to YAML fields. JS scripts retain onMagic logic
(companion spawning, charm rolls, teleport, etc).

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 12: Migrate Complex Buff Flavor Text

**Files:**
- Modify: ~10 complex buff YAML + JS files

Buffs with `onTrigger` logic keep their JS but gain YAML text for
start/end phases.

**Buff list** (buffs with trigger logic in `_datafiles/world/dogmud/buffs/`):
```
32-vital_surge, 33-chrysalis_regeneration,
39-venom, 40-spore_toxin, 78-toxic_cloud
```

Also check `_datafiles/world/default/buffs/` for any flavor-only buffs
we maintain (only if they're DOGMud-modified — skip pure upstream).

- [ ] **Step 1: For each buff, extract start/end text to YAML**

1. Read `onStart` — extract `SendUserMessage` text → `start_user_text`
2. Read `onEnd` — extract text → `end_user_text`
3. Remove `Send*` calls from JS `onStart` / `onEnd`
4. If `onStart`/`onEnd` is now empty, delete the function from JS
5. Keep `onTrigger` intact

- [ ] **Step 2: Verify and test**

Apply a healing buff and a DoT, verify:
- Start text appears (from YAML)
- Trigger text appears (from JS, with computed values)
- End text appears (from YAML)

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/buffs/*.yaml _datafiles/world/dogmud/buffs/*.js
git commit -m "feat: extract start/end text from complex buffs to YAML

Buffs with onTrigger logic retain JS for computed tick messaging.
Start/end flavor text moved to YAML fields.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 13: Delete Stub Room Scripts

**Files:**
- Delete: ~30 stub room JS files

Stub scripts contain only empty hook functions (`onEnter`, `onExit`,
`onLoad` with empty bodies). Since script discovery is automatic by
filename, deleting the file means the engine simply skips scripting
for that room — no behavior change.

- [ ] **Step 1: Identify all stub scripts**

Search for room JS files in `_datafiles/world/dogmud/rooms/` that
contain ONLY empty functions. A stub has no `SendUserMessage`,
`SendRoomMessage`, `mob.Command`, or any other API call.

```bash
# Find JS files with no actual API calls
for f in _datafiles/world/dogmud/rooms/*/*.js; do
  if ! grep -qE 'Send|Command|MoveRoom|GiveQuest|HasQuest|SpawnMob|GetMob|AddTemporaryExit' "$f"; then
    echo "STUB: $f"
  fi
done
```

- [ ] **Step 2: Verify each stub is truly empty, then delete**

For each identified stub, read it to confirm it's only empty functions.
Then delete.

- [ ] **Step 3: Commit**

```bash
git rm [list of verified stub files]
git commit -m "chore: delete empty stub room scripts

These files contained only empty hook functions and had no effect.
Script discovery is automatic by filename — deletion is transparent.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 14: Update Content Generation Slash Commands

**Files:**
- Modify: `.claude/plugins/` or skill files for `/new-mob`, `/new-room`,
  and any spell/buff generation commands

- [ ] **Step 1: Identify which slash commands generate JS files**

Search for the skill definitions that handle spell and buff content
generation. These need to generate YAML text fields instead of (or in
addition to) JS files.

- [ ] **Step 2: Update spell generation to produce YAML text**

The spell generation template should:
- Always produce `cast_user_text` and `cast_room_text` with `{source}` tokens
- For help/harm spells, include `magic_user_text` / `magic_room_text`
  with `{target}` tokens
- Only generate a `.js` file if the spell needs logic (companion summon,
  teleport, etc.)

- [ ] **Step 3: Update buff generation to produce YAML text**

The buff generation template should:
- Always produce `start_user_text` and `end_user_text`
- Include room text variants if the buff has visible effects
- Only generate a `.js` file if the buff needs trigger logic

- [ ] **Step 4: Test by generating a new spell**

Use the updated slash command to generate a test spell. Verify the YAML
contains text fields and no unnecessary JS file is created.

- [ ] **Step 5: Commit**

```bash
git add [modified skill files]
git commit -m "feat: update content generation to use YAML text fields

Spell and buff generators now produce YAML text fields instead of
flavor-only JS scripts. JS files only generated when logic is needed.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```
