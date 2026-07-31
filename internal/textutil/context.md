# Text Utility Context

## Purpose

`internal/textutil` handles the two-audience problem in spell and ability text:
the actor sees one sentence, everyone else in the room sees another, and both
are built from the same authored template with `{token}` substitution.

## Files

- **tokens.go** — `TokenContext`, `SubstituteTokens`, `ValidateTokens`.
- **spelltext.go** — `SendTextConfig` and `SendPhaseText`.

## API

```go
type TokenContext struct { /* actor, target, spell, and related names */ }

func SubstituteTokens(text string, ctx TokenContext) string
func ValidateTokens(text string) []string

type SendTextConfig struct { /* delivery options */ }
func SendPhaseText(userText, roomText string, ctx TokenContext, colorName string, cfg SendTextConfig)
```

`SendPhaseText` is the normal entry point: give it the actor's line and the
room's line and it substitutes, colours, and delivers both.

`ValidateTokens` returns the tokens it could not resolve — it is a
content-validation tool, meant to run over authored spell text at load or in
tests, not on a hot path.

## Gotchas

- **A misspelled token substitutes to nothing and does not error.** The line
  reaches the player with a hole in it. `ValidateTokens` exists to catch that
  before it ships; use it.
- **`roomText` must be written for observers**, in the third person, and must
  not leak information the actor alone should have.
- **No raw numbers.** Spell and combat text uses descriptive language; the
  substitution layer will happily interpolate a number if you give it one.

## Dependencies

`messaging`, `colorpatterns`.

## Consumers

`internal/hooks` (spell resolution) and `internal/usercommands`.
