# Input Handlers Guide

## Scope

- Use this file for login flows, system command handling, terminal-protocol input handling, and interactive input processing in `internal/inputhandlers`.
- This package sits on the boundary between raw client input and account/session state.

## Working Rules

- Be careful with login and account-creation flow changes. Small handler changes can break reconnects, prompt flow, or auth behavior.
- Preserve the distinction between system commands, prompt handling, protocol sanitization, and normal command processing.
- If a change touches telnet or ANSI parsing, inspect the matching terminal/connection behavior together rather than patching only one side.
- Avoid adding gameplay-specific command policy here when it belongs in `internal/usercommands`.
- `ClientInput.Cursor` is a byte offset into `Buffer` and is the editing position. `CleanserInputHandler` owns buffer mutation: it inserts at the cursor and deletes the rune before it; `AnsiHandler` moves it with the arrow/Home/End/Delete keys. Any code that resizes `Buffer` (history recall in `term_ansi.go`, signal shortcuts in `signals.go`, login submit, prompt redraws in `main.go`) must reset/keep `Cursor` consistent or `clampCursor` will paper over it. Display (echo, erase, cursor moves) is width-aware via `runewidth` so CJK/wide characters occupy 2 columns; it is only emitted for server-side-echo, non-masked input — local-echo clients (Mudlet/web) and masked (password) fields manage their own line and stay linear.

## Verification

- Run targeted package tests for login or input-processing changes.
- If the change affects prompt or login flow, verify the exact user interaction path you changed.
- Call out any live terminal or multi-step prompt behavior that was not exercised directly.

## Documentation

- Keep this file about input and login guardrails, not a walkthrough of the full login stack.
