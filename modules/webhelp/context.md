# Web Help Module Context

## Purpose

`modules/webhelp` serves the in-game help files as browsable web pages: a
category index and a per-command page, rendered from the same help sources the
telnet `help` command uses.

One authored help file, two audiences — the shared rendering comes from
`internal/markdown`.

## API

```go
func (w *WebHelpModule) getHelpCategories(r *http.Request) map[string]any
func (w *WebHelpModule) getHelpCommand(r *http.Request) map[string]any
```

Both are registered as plugin web pages through `WebConfig.WebPage`.

## Gotchas

- **Help content is authored for an 80-column terminal.** It renders acceptably
  as HTML, but tables and alignment done with spaces will not survive; keep
  help files to the supported Markdown subset.
- **`internal/markdown.SetFormatter` is process-global.** Rendering help as
  HTML here while a telnet client renders ANSI is a real interleaving hazard —
  see that package's context.
- **A missing help file yields an empty page, not a 404.** The handlers return
  data maps; the template decides what an empty result looks like.

## Dependencies

`plugins`, `internal/markdown`, the help file loader.

## Consumers

Registered as a plugin; served by `internal/web`.
