# Multi-Agent (Scenario) Report Format — 0.3d

The conductor writes ONE combined report per `playtestrun scenario` run, plus
optional per-agent notes. Combined report shape:

```markdown
# Multi-Agent Playtest Report: <scenario name>

**Date:** <YYYY-MM-DD>
**Scenario:** <name> (mode: <mode>)
**Checkout:** <path> (commit: <sha>, dirty: <bool>)
**Run ID:** <playtestenv run_id>
**On actor stop:** <continue|abort>
**Wall-clock:** <duration> (hard cut)
**Agents:** <id> (<personality>, profile|creation), ...
**Endpoint:** <host:ai-port> (ephemeral; never paste passwords)

## Summary
<2-4 sentences on the run arc across agents.>

## Group Goal Results
- [x] <goal id>: <do> — PASS: <cross-agent evidence>
- [ ] <goal id>: <do> — FAIL: <observed vs. expected, which agents>

## Per-Agent Outcomes
- <id> (<personality>): <one-line outcome; bridge/goals paths OK>
  - Status: ready|stopped|failed|incomplete|aborted_peer
  - Soft budgets: <token/pacing notes if any — not a Go hard kill>

## Blackboard
- Dir: `tools/playtest/.run/<run_id>/blackboard/`
- Signals observed: <list signal names>
- (Do not dump secrets; cite signal names + relevant `data` keys only.)

## Findings
(Merged from all agents, deduped, tagged by agent. Keep
BUG/CONCERN/OBSERVATION/PASS/FAIL/BLOCKED categories.)
### BUG: <title> (<agent id>)
<repro: where, what was typed, what happened, what was expected>

## Stats
- Agents: <N>
- Group goals: <P>/<T> PASS
- Bugs / Concerns / Observations: <N> / <N> / <N>
- Scenario sidecar status: <ready|stopped|incomplete_wallclock|…>
```

## Conventions

- Name it `tools/playtest/reports/<date>-<scenario>.md`.
- Group-goal evidence should cite which agents observed what (e.g., "leader's
  GMCP party shows member; member's shows leader").
- Findings come from each agent's play notes + blackboard signal payloads —
  **file I/O only** (no `ptorch bb dump`).
- Never embed passwords or full `creds.json` in the report (paths only).
- Echo checkout `commit` + `dirty` in the header.
