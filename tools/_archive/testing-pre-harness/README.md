# Archived: pre-harness AI testing (2026-06-08)

Superseded by the GoMud playtest harness adoption (Phase 1):

- `mud_bridge.py` / `test-mud.md` → the `mudagent` adapter + `/playtest` driver
  (`.claude/commands/playtest.md`) + the `tools/playtest/` overlay.
- `ai_player.py` (standalone Anthropic-API autonomous player) → retired; the
  harness's bring-your-own-agent model replaces it.
- `testing/` (roles, goals, reports, audits, targets.yaml) → roles became
  `tools/playtest/personalities/`; a curated goal subset was migrated to
  `tools/playtest/goals/`; the rest is kept here for historical reference.

Kept for history only. Do not wire back up.
