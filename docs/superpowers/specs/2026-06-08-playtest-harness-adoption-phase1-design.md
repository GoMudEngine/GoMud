# Adopt the GoMud Playtest Harness for DOGMud — Phase 1 (adapter + content)

**Date:** 2026-06-08
**Status:** Design — approved for planning
**Author:** Calabe Davis + Claude

## Problem

DOGMud's AI testing runs on a fragile, bespoke stack: `tools/mud_bridge.py`
(a file-IPC telnet bridge — writes `mud_output.txt`, reads `mud_cmd.txt`, paced
by chained `sleep 2` calls and ANSI-stripping), driven by the `/test-mud`
command, plus a standalone `tools/ai_player.py` (Anthropic-API autonomous
player). We have since built a far better, engine-agnostic harness
(`~/workspace/gomud-playtest-harness`, the published
`GoMudEngine/GoMud-Module-Playtest-Harness`): a `mudagent` adapter that speaks a
clean line-in / JSON-line-out protocol (`output`/`gmcp`/`status`/`beacon`
events) over the AI port, a reference `/playtest` driver, engine-agnostic
content schemas (personalities, goals, scenarios), and a multi-agent
orchestrator (`ptorch`).

This work replaces DOGMud's file-IPC testing path with the harness's adapter +
driver, keeping the harness repo pristine and engine-agnostic.

## Goals

- Drive DOGMud playtests through the built, engine-agnostic `mudagent` adapter
  (robust JSON/GMCP transport, real round pacing) instead of `mud_bridge.py`.
- Carry DOGMud-specific test config (engine profile, targets, personalities,
  goals) as a small **overlay inside the DOGMud repo** (`tools/playtest/`) so the
  harness repo stays clean and updatable.
- Replace `/test-mud` with a DOGMud `/playtest` command adapted from the
  harness's reference driver, reading only the DOGMud overlay + spawning
  `mudagent`.
- Retire the superseded pieces (`mud_bridge.py`, `ai_player.py`, `/test-mud`,
  the old `tools/testing/` content) into an archive, preserving history.

## Non-goals (explicit — Phase 2 / later)

- **The `playtest` server module** (`module/playtest/`: per-round beacons,
  structural safe mode, death protection, sandbox zone, `ai-flag`/`ai-list`).
  Deferred to Phase 2 — it requires DOGMud server-Go changes against DOGMud's
  diverged justice/jail/downed + config (`AIPort` vs `Network.AI.*`). Without it,
  `mudagent` paces on GMCP/prompt + a timing fallback rather than `beacon`
  events. Phase 1 must not depend on it.
- **Multi-agent scenarios** (`ptorch`, PvP/party/parallel). The harness supports
  them; DOGMud can use them later. Phase 1 targets the single-agent `/playtest`
  path.
- Modifying the harness repo. Approach A keeps it engine-agnostic.

## Constraints / context (verified 2026-06-08)

- `mudagent` is engine-agnostic: it takes only `--target/--user/--password`
  (+ optional `--manifest`); all engine specifics come from the **driver**
  reading `framework/*`. So DOGMud needs no harness code changes — only an
  overlay the DOGMud driver reads.
- DOGMud already has the AI port (`AIPort: 55555`, `AICommandsPerRound: 2`) and
  the `gmcp` module — the adapter connects + negotiates GMCP today.
- DOGMud's GMCP payloads are diverged/custom (its own `Char`/`Comm`/`Zone.Map`),
  but the adapter passes GMCP through generically as `gmcp` events; the driver
  reads them descriptively. No adapter change needed.
- The harness lives at `~/workspace/gomud-playtest-harness` with a prebuilt
  `mudagent.exe`. The DOGMud driver locates it via an env var
  `GOMUD_HARNESS_DIR`, defaulting to `../gomud-playtest-harness` relative to the
  DOGMud repo root; it runs the prebuilt `mudagent`/`mudagent.exe` if present,
  else `go run ./cmd/mudagent` from that dir.

## Architecture

Three units, all in the DOGMud repo; the only external dependency is the
engine-agnostic `mudagent` binary from the harness checkout.

### 1. DOGMud playtest overlay — `tools/playtest/`
The DOGMud-specific content the driver reads (mirrors the harness `framework/`
layout so the driver logic is a thin adaptation):

- **`engine-profile.yaml`** — the one place DOGMud specifics live:
  - `engine: DOGMud`
  - `setup_commands: ["set charset"]` (force ASCII — prevents Unicode/box-draw
    corruption; the old `/test-mud` did this manually).
  - `commands:` DOGMud verbs — `look`, movement `[north,south,east,west,up,down]`,
    `inventory`, `status`, `get/drop`, `attack <target>`, `flee`, `say`,
    `recover: ["cast <healspell> me", "drink <potion>"]`, `help`, plus DOGMud
    extras worth surfacing (`map`, `cast`, `drink`, `eat`, `consider`,
    special moves `kick/bash/taunt`).
  - `world:` DOGMud starts new characters in **Sanctum Basin** (not Frostfang);
    `onboarding:` DOGMud's char-creation prompt sequence; `orientation:`
    compass + `look`/`look <noun>` + `map`.
  - `mechanics:` **three pools — Health (HP), Stamina (SP), Conviction (CP)**
    (not GoMud's two); **no levels/XP** (use-based stat progression on the six
    100-centered stats); `death:` DOGMud uses a **justice/jail** system (arrest,
    bounties) — not permadeath; bleedout/downed removed; `combat:`
    `attack`/`flee`, **4-second rounds**, special moves + `cast` spells.
  - `notes:` AI-port output is ANSI-stripped; don't file display artifacts as
    bugs; NPC targets are exact room-description keywords.
- **`targets.yaml`** — `local` (`localhost:55555`, `user: smoketester`,
  `password: smoke123test`) and `prod` (`dogmud.org:55555`, `user: aitester`,
  `password: testpass123`). Existing DOGMud test accounts (already AI-flagged).
- **`personalities/`** — copies of the three role files (`bug-finder.md`,
  `feature-tester.md`, `feel-tester.md`); identical to ours/the harness's, with
  light DOGMud flavor (justice/jail awareness, three-pool survival, charset
  note). Copied (not referenced) so the DOGMud overlay is self-contained and the
  only harness dependency is the `mudagent` binary.
- **`goals/`** — a **curated subset** of currently-relevant DOGMud goals
  migrated to the harness goal schema (see Goal migration). Not all ~50 — most
  are stale chunk-smokes that move to the archive.
- **`reports/`** — output dir (gitignored except `.gitkeep`).

### 2. DOGMud driver — `.claude/commands/playtest.md`
Adapted from the harness's `.claude/commands/playtest.md`, with DOGMud changes:
reads `tools/playtest/{targets,engine-profile,personalities,goals}` (not
`framework/`); resolves the harness dir via `GOMUD_HARNESS_DIR`
(default `../gomud-playtest-harness`) and bridges `mudagent`'s stdio via files
under `tools/playtest/.run/` (`commands.txt` / `events.jsonl`, gitignored);
drives the JSON event loop (poll `events.jsonl` for `status: logged_in`, send
commands by appending to `commands.txt`, judge goals from `output`/`gmcp`),
writes a report to `tools/playtest/reports/`. Usage:
`/playtest <local|prod> <personality> [goals-file]`.

### 3. Archive — `tools/_archive/testing-pre-harness/`
Moves the superseded stack out of the way, preserving git history (`git mv`):
`mud_bridge.py`, `ai_player.py`, `.claude/commands/test-mud.md`, and the old
`tools/testing/` tree (roles/, the un-migrated goals/, reports/, audits/). A
short `README.md` in the archive notes it's replaced by the harness overlay +
`/playtest`, dated, for historical reference.

## Goal migration

Schema delta (mechanical):

| DOGMud goal (old) | Harness goal (new) |
|---|---|
| `description` | `name` + `summary` |
| `goals[].id` | `goals[].id` (same) |
| `goals[].text` | `goals[].do` (the attempt) + `goals[].verify` (split out the success check, or a light "observe/confirm ..." line) |
| `exit_criteria` | `pass_criteria` (list) |

Only a **curated, currently-useful subset** is migrated (e.g. economy/shop,
supply-chain, crime/justice, messaging-smoke, a feel pass) — the implementation
plan will name the exact list. Stale per-chunk smokes are archived, not
converted. Migration is hand-done (the `text`→`do`/`verify` split needs a little
judgment), not scripted.

## Data flow

`/playtest local feel-tester [goal]` → driver reads `tools/playtest/*` →
spawns `mudagent --target localhost:55555 [--user smoketester --password ...]`
(stdio bridged to `tools/playtest/.run/`) → adapter connects + negotiates GMCP →
driver waits for `status: logged_in` (or drives char-creation) → loop: append
command → read new `events.jsonl` lines (`output`/`gmcp`) → judge goals → write
`tools/playtest/reports/<dated>.md`.

## Error handling

- **Harness not found** (`GOMUD_HARNESS_DIR` missing / no `mudagent`): driver
  fails fast with a clear message (where it looked, how to set the env var / clone
  the harness).
- **Adapter exits / disconnect** (`status: disconnected` or process gone):
  driver reports progress-so-far and stops, like the old bridge.
- **Account doesn't exist** (blank creds or repeated login prompt): driver drives
  the new-player creation flow (a real feel-tester check), per the harness driver.
- **GMCP divergence**: descriptive `verify` over text scraping where possible;
  unknown gmcp packages are ignored, not errors.

## Testing / acceptance

- **Unit:** none (overlay is YAML/markdown; the binary is upstream-tested).
- **Acceptance (manual):** boot DOGMud locally (AI port 55555); run
  `/playtest local feel-tester` — confirm: adapter connects, `status: logged_in`
  (or clean char-creation), `set charset` ran, several rounds of
  command→`output`/`gmcp` play, and a report lands in `tools/playtest/reports/`.
  Then run one migrated goal (e.g. shop-economy) with `feature-tester` and
  confirm goal-by-goal PASS/FAIL judging. Confirm a `bug-finder` exploratory run
  works without a goals file.
- **Regression:** `go build ./...` (no DOGMud Go changed — sanity only). Confirm
  the archived files are gone from their old paths and `/test-mud` no longer
  resolves.

## Files touched

- **new** `tools/playtest/engine-profile.yaml`, `tools/playtest/targets.yaml`,
  `tools/playtest/personalities/{bug-finder,feature-tester,feel-tester}.md`,
  `tools/playtest/goals/*.yaml` (migrated subset), `tools/playtest/reports/.gitkeep`
- **new** `.claude/commands/playtest.md` (DOGMud driver)
- **new** `tools/_archive/testing-pre-harness/README.md`
- **moved (git mv → archive)** `tools/mud_bridge.py`, `tools/ai_player.py`,
  `.claude/commands/test-mud.md`, `tools/testing/**`
- **edit** `.gitignore` — ignore `tools/playtest/.run/` and
  `tools/playtest/reports/*` (keep `.gitkeep`)

## Out of scope (later)

- Phase 2: vendor `module/playtest/` into `modules/playtest/` (beacons, safe
  mode, death protection, sandbox) adapted to DOGMud's config + justice systems.
- Multi-agent scenarios (`ptorch`) for DOGMud.
- Auto-discovery/versioning of the harness binary beyond `GOMUD_HARNESS_DIR`.
