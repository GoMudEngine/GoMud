# Agent Runner (per-agent role in a scenario)

You are ONE tester in a multi-agent scenario. The conductor gave you: your
**agent id**, your **role** (a personality), your **target**, your **assignment**
(group goals + any per-agent goals + your choreography lines), the **blackboard
path** (`bb`), and your private `mudagent` bridge files. Follow your personality
(`tools/playtest/personalities/<role>.md`) and the engine profile
(`tools/playtest/engine-profile.yaml`) throughout. If you are on a non-`fresh`
synthetic profile, also read `tools/playtest/profiles/context.md`.

Resolve the harness directory before issuing any `ptorch` commands:
`HARNESS="${GOMUD_HARNESS_DIR:-../gomud-playtest-harness}"` (relative to the
DOGMud repo root). Use `(cd "$HARNESS" && go run ./cmd/ptorch ...)` or the
pre-built binary (`$HARNESS/ptorch.exe` on Windows, `$HARNESS/ptorch` elsewhere).
The blackboard path `$BB` was given to you as an absolute path by the conductor.

## 1. Connect and enter the world

Start your `mudagent` exactly as the single-agent driver does
(`.claude/commands/playtest.md` step 2), using your target's host/port and creds.
Your private bridge files are under `tools/playtest/.run/$RUN/<id>/` (the
conductor gave you the exact paths). With blank creds, create a character via the
new-player flow. Poll your events file until
`{"type":"status","state":"logged_in"}`.

**Ensure an ASCII charset (DOGMud).** After `logged_in`, `set charset` is a
*toggle* with no "get current state" query — a session can start in either mode.
Converge to ASCII:
- Send `set charset`, read the response line:
  - "Charset mode set to ASCII." → done (ASCII confirmed).
  - "Charset mode set to UTF-8." → you were in ASCII and just flipped to UTF-8;
    send `set charset` once more to return to ASCII, and confirm the ASCII line.
- Stop once ASCII is confirmed (at most 2 sends). Box-drawing/emoji arrive as
  mojibake otherwise.

### Onboarding mode

New characters spawn as a pre-tutorial **ghost** in the Void. Honor your assigned
`onboarding` value:
- `auto` (default): advance past the ghost quickly — `start` → choose a race →
  enter a name → confirm → skip the tutorial — so you reach a full character in
  the start room, ready to play.
- `full`: deliberately go through the **real new-player flow** and (if you're a
  feel-tester) grade the experience as you go — it's part of what you're testing.

## 2. Join the lobby barrier

Mark yourself ready, then wait for the conductor to start the run:
```sh
(cd "$HARNESS" && go run ./cmd/ptorch bb ready "$BB" --id "$AGENT_ID")
# wait until the run is RUNNING (poll; the conductor flips it once ALL are ready)
until [ "$(cd "$HARNESS" && go run ./cmd/ptorch bb phase "$BB")" = "running" ]; do sleep 1; done
```

## 3. Play your assignment

Pursue your role + group goals + per-agent goals, interacting **in the game**
(party invites, attacks, trades happen through `mudagent`, not the blackboard).
Pace on the per-round `Playtest.Round` beacon, as in the single-agent loop
(`.claude/commands/playtest.md` step 5).

- **Emit signals** when your choreography says you've reached a cue (use the
  current beacon round):
  ```sh
  (cd "$HARNESS" && go run ./cmd/ptorch bb signal "$BB" --name "$AGENT_ID.ready" --round "$ROUND")
  ```
- **Wait on another agent's cue** (a `choreography.after`):
  ```sh
  until (cd "$HARNESS" && go run ./cmd/ptorch bb dump "$BB") | grep -q '"other.ready"'; do sleep 1; done
  ```

## 4. Record findings

Whenever you find something, drop it on the blackboard so it reaches the combined
report:
```sh
(cd "$HARNESS" && go run ./cmd/ptorch bb finding "$BB" --agent "$AGENT_ID" --type BUG --title "short title" --round "$ROUND")
```

## 5. Finish

When your goals are met (or an exit condition from `.claude/commands/playtest.md`
step 6 is hit), write your per-agent report to
`tools/playtest/.run/$RUN/$AGENT_ID-report.md` following the session report
format from `.claude/commands/playtest.md` step 7. Then quit your `mudagent`
(`.claude/commands/playtest.md` step 8). The conductor aggregates once all agents
finish.
