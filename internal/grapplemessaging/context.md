# Grapple Messaging Context

## Purpose

`internal/grapplemessaging` holds the prose library for the grappling system —
the per-position, per-transition lines describing what two entangled fighters
are doing — and picks a line without repeating itself.

Grappling has many states and many transitions between them, so the message
count is large enough that (a) the pool must be validated for completeness at
load, and (b) selection must avoid immediate repeats or the fight reads like a
stuck record.

## Files

- **loader.go** — `Library`, `TemplateTriad`, `GradientTriad`, `Load`,
  `ValidateCompleteness`.
- **render.go** — `RenderTemplate`, `PickTemplate`.

## Types

```go
type TemplateTriad struct  { /* the three viewpoints of one event */ }
type GradientTriad struct  { /* intensity-graded variants */ }
type Library struct        { /* all pools, keyed by position/transition */ }
```

A **triad** is the same event told three ways — to the controller, to the
controlled, and to the room. Storing them together is what stops the three
drifting apart when someone edits one.

## API

```go
func Load(path string) (*Library, error)
func ValidateCompleteness(lib *Library) []error

func RenderTemplate(template, controllerName, controlledName string) string
func PickTemplate(pool []string, cooldowns map[string]bool, keyPrefix string) string
```

## Gotchas

- **`ValidateCompleteness` returns a slice of errors, not one.** Run it at load
  and report all of them; a partially-covered library produces silent blanks
  mid-fight, which is far harder to diagnose than a startup complaint.
- **`PickTemplate` needs the caller to own the cooldown map.** It reads and
  marks; it does not age entries. A caller that never clears the map eventually
  runs out of eligible lines.
- **`keyPrefix` namespaces the cooldowns.** Two pools sharing a prefix will
  suppress each other's lines.
- **Both names are substituted positionally** — `RenderTemplate` does not know
  which is the player. Getting controller and controlled the wrong way round
  produces text that is grammatical and completely wrong.

## Dependencies

`configs`, `mudlog`, plus YAML.

## Consumers

`internal/combat` and `internal/hooks` on the grapple resolution path.
