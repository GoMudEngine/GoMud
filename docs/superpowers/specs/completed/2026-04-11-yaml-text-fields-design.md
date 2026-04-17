# YAML Text Fields for Spells and Buffs

**Date:** 2026-04-11
**Status:** Draft
**Phase:** JS Audit Phase 1 — Move static flavor text from JS scripts into
YAML data files

## Goal

Eliminate ~75 trivially duplicated JS files (flavor-only spells and buffs)
by moving their message text into YAML fields on the existing spell/buff
definitions. Complex scripts retain their JS for logic but shed their
boilerplate flavor functions.

## Scope

**In scope:**
- New YAML text fields on `SpellData` and `BuffSpec` structs
- Token substitution helper for actor name interpolation
- Engine resolution changes to send YAML text before calling JS hooks
- Load-time validation for missing text and unknown tokens
- Migration of all flavor-only spell/buff scripts to YAML
- Migration of flavor text from complex spell/buff scripts
- Deletion of ~30 stub room scripts (empty hook functions)
- Update content generation slash commands

**Out of scope:**
- Healing/DoT buff logic (Phase 2 — config-driven `heal_pct`/`damage_pct`)
- Companion spell consolidation (Phase 2 — parameterized Go function)
- Item/room script migration (Phase 3)
- Mob AI / quest script replacement (Phase 4)
- ANSI-aware line wrapping fix (future — architecture enables it)
- AoE damage distribution rework (separate project)

## Design

### YAML Schema Changes

Six new optional string fields on `SpellData`:

```yaml
cast_user_text: "You channel your conviction into a crackling bolt."
cast_room_text: "{source} channels conviction into a crackling bolt."
wait_user_text: "Energy crackles between your fingertips."
wait_room_text: "Energy crackles around {source}."
magic_user_text: "You hurl the bolt at {target}."
magic_room_text: "{source} hurls a bolt of crackling energy at {target}."
```

Six new optional string fields on `BuffSpec`:

```yaml
start_user_text: "Conviction surges through you."
start_room_text: "A fierce light burns in {source}'s eyes."
trigger_user_text: ""
trigger_room_text: ""
end_user_text: "The surge of conviction fades."
end_room_text: "The fierce light in {source}'s eyes dims."
```

All fields optional. Missing or empty string = no message sent for that
phase/audience combination.

### Token Substitution

A helper function replaces known tokens before sending:

| Token | Resolves To | Notes |
|-------|------------|-------|
| `{source}` | Caster/holder's tagged name | ANSI-colored display name |
| `{target}` | Target's tagged name | ANSI-colored display name |
| `{source_plain}` | Caster's plain name | For possessives, e.g. `{source_plain}'s` |
| `{target_plain}` | Target's plain name | For possessives |

Function signature:

```go
// TokenContext holds the actor names for substitution.
// Built at the call site from whatever actor types are available.
type TokenContext struct {
    SourceName      string // tagged/ANSI name
    SourcePlainName string // plain name
    TargetName      string // tagged/ANSI name (empty if no target)
    TargetPlainName string // plain name (empty if no target)
}

func SubstituteTokens(text string, ctx TokenContext) string
```

- Fields in `TokenContext` default to empty string when there is no
  target (buffs, area spells). Empty token = empty replacement.
- Centralizing text through this function creates a single point where
  ANSI-aware line wrapping can be added later.

**Load-time validation:** During `Validate()`, scan text fields for `{...}`
patterns. Log a warning on unknown tokens (catches typos like `{soruce}`).
Warning, not fatal — forward-compatible with future tokens.

### Engine Resolution Flow

**Spell resolution** — YAML text sends first, then JS runs:

```
Cast phase:
  1. Send cast_user_text (if non-empty) → substitute tokens, send to caster
  2. Send cast_room_text (if non-empty) → substitute tokens, send to room
  3. Call JS onCast() if script exists

Wait phase (each round):
  1. Send wait_user_text → to caster
  2. Send wait_room_text → to room
  3. Call JS onWait() if script exists

Magic phase:
  1. Send magic_user_text → to caster
  2. Send magic_room_text → to room
  3. Call JS onMagic() if script exists
```

**Buff resolution** — same pattern:

```
Start:   send start texts   → call JS onStart() if exists
Trigger: send trigger texts  → call JS onTrigger() if exists
End:     send end texts      → call JS onEnd() if exists
```

**Validation at load time:**
- Spell/buff with NO text fields AND no JS file → log warning
  ("spell X has no messaging")
- Unknown tokens in text fields → log warning

### Migration Strategy

**Order of operations:**

1. **Go changes** — Add struct fields, write substitution helper, modify
   resolution hooks, add validation. Write unit tests for token
   substitution and integration tests for the resolution flow.

2. **Proof of concept** — Migrate one flavor-only spell end-to-end
   (e.g., `conviction-surge`). Verify in-game: correct messages, no
   double-sends, no regressions.

3. **Batch migrate flavor-only spells** (~35 files) — Copy text from JS
   to YAML, delete JS files.

4. **Batch migrate flavor-only buffs** (~40 files) — Same process.

5. **Migrate complex spell flavor text** — Add YAML text fields to
   companion summon, charm, teleport spells. Strip `SendUserMessage`/
   `SendRoomMessage` calls from JS `onCast`/`onWait` functions. Keep
   `onMagic` logic intact.

6. **Migrate complex buff flavor text** — Add start/end YAML text to
   healing/DoT buffs. Keep JS `onTrigger` logic (Phase 2 handles these).

7. **Delete stub room scripts** (~30 files) — Verify they contain only
   empty hooks, then delete. No YAML changes needed (discovery is
   automatic by filename).

8. **Update content generation** — Modify `/new-mob`, `/new-room`, and
   spell/buff generation slash commands to produce YAML text fields
   instead of JS flavor files.

### Testing

**Unit tests:**
- `SubstituteTokens` with all token types, nil actors, empty strings,
  unknown tokens, no tokens
- Validation warns on unknown tokens, warns on no-text-no-script

**Integration tests:**
- Spell with YAML text + no JS → messages sent correctly
- Spell with YAML text + JS logic → YAML text sends, JS logic runs,
  no double-send
- Spell with no text and no JS → warning logged
- Buff lifecycle with YAML text → start/trigger/end messages correct

**Manual smoke test:**
- Cast a migrated flavor spell in-game
- Cast a migrated complex spell (e.g., a companion summon)
- Apply and let a migrated buff expire
- Verify no double messages, correct names, correct room broadcast
  exclusions

## Files Modified

**Go source (new/modified):**
- `internal/spells/spells.go` — Add 6 text fields to `SpellData`
- `internal/buffs/buffspec.go` — Add 6 text fields to `BuffSpec`
- `internal/hooks/spell_resolution.go` — Send YAML text in magic phase
- `internal/hooks/NewRound_DoCombat_helpers.go` — Send YAML text in
  cast/wait phases
- `internal/hooks/Buff_ApplyBuffs.go` — Send YAML text on buff start
- `internal/hooks/NewRound_UserRoundTick.go` — Send YAML text on trigger
- `internal/hooks/NewTurn_PruneBuffs.go` — Send YAML text on buff end
- New file: `internal/textutil/tokens.go` — token substitution helper

**Data files (modified/deleted):**
- ~35 spell JS files deleted
- ~40 buff JS files deleted
- ~30 stub room JS files deleted
- ~75 spell/buff YAML files gain text fields
- Complex spell/buff JS files have messaging calls stripped

**Estimated net change:**
- ~105 JS files deleted (~1,050 lines removed)
- ~75 YAML files gain ~6 lines each (~450 lines added)
- ~200 lines of new Go code (struct fields, substitution, resolution
  hooks, validation, tests)

## Future Work Enabled

- **Phase 2:** Config-driven buff ticks (`heal_pct`, `damage_pct`) and
  companion spell consolidation into parameterized Go functions
- **ANSI wrapping fix:** Centralized text pipeline is the right place to
  add ANSI-aware line wrapping
- **Refactor to TextSpec sub-struct:** If items/rooms gain text fields,
  extract to shared struct with `yaml:",inline"` (no YAML format change)
- **AoE damage distribution:** Separate project, unrelated to text fields
