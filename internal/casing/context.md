# Casing Context

## Purpose

`internal/casing` is two functions enforcing the project's naming-casing
convention: a title-caser that knows about MUD naming, and a boot-time
assertion that authored names are in canonical form.

## API

```go
func Title(s string) string
func AssertCanonical(name, kind, source string)
```

`Title` upper-cases the first letter of each word. `AssertCanonical` is the
validator: it is called during data loading with the name, what kind of thing it
is, and where it came from, so a violation names the offending file rather than
just complaining.

## Gotchas

- **`AssertCanonical` is a boot-time gate.** It is meant to fail loudly on
  authored content, which is how casing drift gets caught before it reaches
  players. Do not call it on runtime-generated or player-supplied strings.
- **`Title` is not `strings.Title`.** Do not swap in the standard library
  version, which is deprecated and has different behaviour on punctuation and
  Unicode.

## Dependencies

Standard library only.

## Consumers

The data loaders (`mobs`, `items`, `rooms`, `spells`) and the naming-convention
tests.
