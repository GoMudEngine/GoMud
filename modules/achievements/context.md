# Achievements Module Context

## Purpose

`modules/achievements` is the wiring around `internal/achievements`. The
internal package defines and evaluates; this module registers the plugin,
listens for the events that could change an achievement's state, records
unlocks on the character, announces them, and renders the web page.

The split is deliberate: evaluation stays a pure function of a character, and
everything with a side effect lives here.

## Files

- **achievements.go** — the module, the web data provider, and the display
  shaping (`webAchievement`, `webCategory`).

## Web data

```go
func (m *module) webData(r *http.Request) map[string]any
```

Groups definitions into `webCategory` buckets for the dashboard, with per-user
unlock state and progress folded in. Registered through the plugin's
`WebConfig`.

## Gotchas

- **Announcement belongs here, not in `internal/achievements`.** If an
  achievement unlocks silently, the missing piece is a listener in this module,
  not a bug in `Evaluate`.
- **Progress comes from `Progress(t, c)`**, which returns `numeric = false` for
  triggers with no partial state. The web view must not render a progress bar
  for those.
- **Category order comes from the definitions**, so a new category needs a
  matching entry in the internal package's validated set or the file will not
  load at all.

## Dependencies

`plugins`, `internal/achievements`, `characters`, `users`, `events`.

## Consumers

Registered as a plugin; serves the `achievements` command and the web page.
