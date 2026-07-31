# Markdown Context

## Purpose

`internal/markdown` parses a small Markdown subset into an AST and renders it
through a pluggable `Formatter`. It exists so one authored help file can be
served both to a telnet client (as ANSI tags) and to the web client (as HTML)
without maintaining two copies.

## Files

- **parser.go** — the recursive-descent parser.
- **ast.go** — `Node`, `NodeType`, `baseNode`, and the formatter hookup.
- **formatter.go** — the `Formatter` interface.
- **formatter_ansitags.go** — ANSI-tag output for the MUD client.
- **formatter_html.go** — HTML output for the web client.
- **formatter_remarkdown.go** — round-trips back to Markdown.

## API

```go
func NewParser(input string) *Parser
func (p *Parser) Parse() Node
func SetFormatter(newFormatter Formatter)

type Node interface {
    Type() NodeType
    Children() []Node
    String(depth int) string
}
```

Rendering happens in `Node.String(depth)`, which delegates to the **currently
installed** formatter.

## Supported subset

Document, paragraph, heading, horizontal line, hard break, list, list item,
text, strong, emphasis, and `Special`. That is all — no tables, no code fences,
no links, no images. Authored help files must stay inside it.

`ReMarkdown` exists mostly as a fidelity check: parse, re-render, and compare
to see what the parser dropped.

## Gotchas

- **`SetFormatter` is global mutable state.** There is one formatter for the
  whole process, so rendering the same document as HTML for the web and as ANSI
  for telnet means swapping it around the call. That is not concurrency-safe —
  do not render both formats from different goroutines.
- **Unsupported syntax degrades silently** to plain text rather than erroring.
  A table in a help file simply loses its structure, and nothing warns.
- **`Special` is the escape hatch** each formatter interprets differently;
  check the specific formatter before relying on it.

## Dependencies

Standard library, plus the ansitags conventions the ANSI formatter emits.

## Consumers

The help system and `internal/web`.
