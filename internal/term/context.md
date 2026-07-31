# Terminal Protocol Context

## Purpose

`internal/term` is the wire-protocol vocabulary for telnet and ANSI. It is
almost entirely **declarative**: a large table of `TerminalCommand` values
naming the byte sequences the server sends and recognises, plus a small set of
matcher and parser helpers. It performs no I/O and holds no state — the
connection layer (`internal/connections`) owns the socket and asks this package
"is this byte run a command, and if so which one?"

The package also carries the MSP (sound) and MSSP (MUD server status) protocol
fragments, since both ride on the same telnet subnegotiation framing.

## Files

- **term.go** — `TerminalCommand`, the master ANSI/telnet command table, and the
  `Matches` recogniser.
- **telnet.go** — telnet option bytes (`TELNET_IAC`, `TELNET_DO`, …) and the
  `TelnetWILL`/`WONT`/`DO`/`DONT` sequence builders.
- **ansi.go** — ANSI byte constants and payload parsers for mouse clicks,
  scroll wheel, and screen-size reports.
- **msp.go** — MSP (MUD Sound Protocol) command bytes and `IsMSPCommand`.
- **mssp.go** — MSSP field encoding for MUD-listing crawlers.

## Core Type

```go
type TerminalCommand struct {
    Chars    []byte // leading byte sequence
    EndChars []byte // trailing byte sequence (may be empty)
}
```

A command is a *prefix* and an optional *suffix*; anything between them is the
payload. That single shape covers both fixed sequences (`AnsiCursorHide`, no
payload) and variable ones (`AnsiColor8BitFG`, where the payload is the colour
index).

Methods: `BytesWithPayload(payload []byte) []byte`, `ExtractBody(input []byte)
[]byte`, `String() string`, `StringWithPayload(payload string) string`,
`DebugString() string`.

## Public API

### Recognition

```go
func Matches(input []byte, cmd TerminalCommand) (ok bool, payload []byte)
func IsTelnetCommand(b []byte) bool   // b[0] == TELNET_IAC
func IsAnsiCommand(b []byte) bool     // b[0] == ANSI_ESC
func IsMSPCommand(b []byte) bool
func IsMSSPCommand(b []byte) bool
```

`Matches` is the workhorse: it checks the prefix, checks the suffix, and returns
whatever sat between them as the payload. Callers switch over the command table
by calling it repeatedly.

### Telnet sequence builders

```go
func TelnetWILL(what IACByte) []IACByte
func TelnetWONT(what IACByte) []IACByte
func TelnetDO(what IACByte) []IACByte
func TelnetDONT(what IACByte) []IACByte
```

### Payload parsers

```go
func TelnetParseScreenSizePayload(info []byte) (width, height int, err error)
func AnsiParseScreenSizePayload(info []byte) (width, height int, err error)
func AnsiParseMouseClickPayload(info []byte) (xPos, yPos int, err error)
func AnsiParseMouseWheelScroll(info []byte) (xPos, yPos int, err error)
```

Two screen-size parsers exist because clients report dimensions two different
ways: NAWS subnegotiation (telnet) and a cursor-position report (ANSI). The
ANSI path is the hacky fallback — `AnsiRequestResolution` saves the cursor,
drives it to `999;999`, asks where it landed, and restores it.

### MSSP encoding

```go
type MSSPField struct { /* name/value pair */ }
func EncodeMSSPPayload(fields []MSSPField) []byte
```

Escapes embedded `IAC` bytes so a field value cannot terminate the
subnegotiation early.

### Debug helpers

`TelnetCommandToString`, `AnsiCommandToString`, `BytesString` — render raw byte
runs readably in logs. Not used on hot paths.

## Constants

ASCII: `ASCII_NULL`, `ASCII_BACKSPACE`, `ASCII_SPACE`, `ASCII_DELETE`,
`ASCII_TAB`, `ASCII_CR`, `ASCII_LF`.

Byte sequences: `CRLF`/`CRLFStr`, `BELL`/`BELLStr`, `BACKSPACE_SEQUENCE` (move
back, print space, move back again — the classic destructive backspace).

The command table itself is grouped by prefix in `term.go`: `Telnet*` for
option negotiation and charset handshaking, `Ansi*` for colour (4/8/24-bit),
cursor movement, screen clearing, alt-screen mode, mouse reporting, function
keys, and window title/bell control.

## Gotchas

- **`TerminalCommand` values are compared by matching, never by `==`.** Use
  `Matches`; two commands can share a prefix and be distinguished only by their
  suffix (`AnsiClientMouseDown` vs `AnsiClientMouseUp` differ solely in `M` vs
  `m`).
- **`ExtractBody` assumes the input already matched.** It slices by prefix and
  suffix length without re-validating; calling it on unmatched input will panic
  or return garbage.
- **The charset handshake is four-way** (`TelnetRequestChangeCharset` →
  `TelnetAgreeChangeCharset` → `TelnetCharset` → accepted/rejected). The
  sequence is documented inline above the constants in `term.go`. A client that
  never converges to UTF-8 is behaving legally, not buggily.
- **`DebugString` indexes `Chars[0]`** and will panic on a zero-length command.

## Dependencies

Standard library only (`fmt`). This package deliberately imports nothing from
the rest of the engine, which is what lets `connections` sit low in the import
graph.

## Consumers

`connections`, `inputhandlers`, `users`, `rooms`, `templates`, `usercommands`,
`mobcommands`, `hooks`, `util`, and `modules/gmcp`.
