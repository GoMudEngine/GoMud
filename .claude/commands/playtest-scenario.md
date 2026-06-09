---
description: Run a multi-agent (party / adversarial / parallel / scenario) playtest
argument-hint: <scenario-name>
---

# /playtest-scenario `<scenario-name>`

The DOGMud conductor for **multi-agent** runs. It reads a scenario file, spawns
one independent agent per roster entry, coordinates them via the game + a small
shared blackboard, and writes a combined report. Auto-discovered from the repo —
no install. (Single-agent runs still use `/playtest`.)

> ⚠️ **Cost:** each roster agent is an independent LLM loop. **N agents cost
> roughly N× a single `/playtest` run** in tokens and local processing. Start with
> 2 agents, watch your usage rate, and keep rosters small. The server also caps AI
> clients at `Network.MaxAIConnections` (default 20).

## 0. Resolve the harness

Resolve the harness directory:
`HARNESS="${GOMUD_HARNESS_DIR:-../gomud-playtest-harness}"` (relative to the
DOGMud repo root).

If `$HARNESS` does **not** exist, STOP and tell the user:
"playtest harness not found at $HARNESS — set GOMUD_HARNESS_DIR or clone
GoMudEngine/GoMud-Module-Playtest-Harness next to the DOGMud repo."

Resolve the `ptorch` binary (try in order):
- `$HARNESS/ptorch.exe` (Windows pre-built)
- `$HARNESS/ptorch` (Linux/macOS pre-built)
- `go run ./cmd/ptorch` compiled on the fly, **run from inside `$HARNESS`**

Throughout this driver, every `ptorch` call runs **from inside `$HARNESS`** (via
`(cd "$HARNESS" && ...)` subshell or the pre-built binary), and any path
arguments that are relative to DOGMud (scenarios, blackboard) are passed as
**absolute paths** (use `$(pwd)/...` from the DOGMud repo root).

## 1. Load and check the scenario

- The scenario file is `tools/playtest/scenarios/<scenario-name>.yaml`. Get its
  machine-readable plan:
  ```sh
  SCENARIO="$(pwd)/tools/playtest/scenarios/<scenario-name>.yaml"
  (cd "$HARNESS" && go run ./cmd/ptorch scenario plan "$SCENARIO")
  # or: "$HARNESS/ptorch.exe" scenario plan "$SCENARIO"  (pre-built on Windows)
  ```
  This emits JSON: `name`, `mode`, `max_connections`, `roster` (id/role/target),
  `group_goals`, `requires`, and `warnings`. If the command exits non-zero, the
  file is invalid — show the error and stop.
- **Surface every `warnings` entry to the user** (over-limit roster, cost). If
  the roster exceeds `max_connections`, stop and tell the user to raise
  `Network.MaxAIConnections` in `_datafiles/config.yaml` (or shrink the roster)
  before continuing.
- **Surface `requires` as preconditions to confirm** — the conductor does NOT
  change server config. DOGMud-specific interpretation of `requires` keys:
  - `max_connections`: roster cap; raise `Network.MaxAIConnections` (default 20)
    in `_datafiles/config.yaml`.
  - `pvp`: DOGMud uses `GamePlay.PVP` (`enabled`/`limited`/`disabled`; default
    `disabled`). Tell the user to set the flag and restart before a PvP scenario
    runs.
  - `min_skill_ranks` / progression floor: **DOGMud has no levels.** Surface
    `GamePlay.PVPMinimumSkillRanks` (default 15) instead; tell the user to lower
    it or rank the test characters up first.
  - `permadeath` / `perma_death_protection`: **N/A in DOGMud** — defeat causes
    a respawn, not permanent loss. Note this to the user; permadeath-keyed
    `requires` entries are irrelevant for DOGMud runs.

## 2. Seed the blackboard

```sh
RUN="<scenario-name>-<date>"      # date passed in by you; do not invent timestamps in code
BB="$(pwd)/tools/playtest/.run/$RUN/blackboard.json"
mkdir -p "$(dirname "$BB")"
(cd "$HARNESS" && go run ./cmd/ptorch bb init "$BB" --run "$RUN" --ids "<comma-separated roster ids>")
```

## 3. Spawn one agent per roster entry (background, independent)

For each roster entry, dispatch a **background subagent** whose instructions are
`tools/playtest/agent-runner.md`, parameterized with: that entry's `id`, `role`,
`target`, the relevant `group_goals` + per-agent `goals` + any `choreography`
lines naming it, the blackboard path `$BB`, a private bridge dir
`tools/playtest/.run/$RUN/<id>/`, and the roster entry's `onboarding` value (from
the plan JSON) so the agent knows whether to auto-advance past the ghost or drive
the full new-player flow. Each agent connects, creates/logs in its character, and
marks itself ready.

(Other agent runtimes can spawn OS processes instead — the scenario file +
blackboard CLI are the engine-agnostic contract; subagents are just the reference.)

## 4. Readiness barrier

Wait for all agents to be present, then start the run:
```sh
until (cd "$HARNESS" && go run ./cmd/ptorch bb allready "$BB"); do sleep 1; done   # exit 0 = all ready
(cd "$HARNESS" && go run ./cmd/ptorch bb phase "$BB" --set running)
```

## 5. Let agents run; wait for completion

Agents now play their assignments, interacting in-game and via signals. Wait for
all background subagents to finish (each writes its per-agent report and appends
its findings to the blackboard), then:
```sh
(cd "$HARNESS" && go run ./cmd/ptorch bb phase "$BB" --set done)
```

## 6. Aggregate the combined report

Read the final blackboard and each per-agent report:
```sh
(cd "$HARNESS" && go run ./cmd/ptorch bb dump "$BB")
```
Write the combined report per `tools/playtest/multi-agent-report-format.md` to
`tools/playtest/reports/<date>-<scenario-name>.md`: scenario summary, group-goal
results with cross-agent evidence, per-agent outcomes, and the merged/deduped
findings (already deduped per agent+title on the blackboard).

## 7. Clean up

Quit any still-running `mudagent`s (each agent does this on finish; clean up
strays as in `.claude/commands/playtest.md` step 8). Report the combined-report
path to the user.
