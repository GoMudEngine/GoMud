# Multi-Agent (Scenario) Report Format

The conductor writes ONE combined report per scenario run, plus the usual
per-agent reports (matching the format from `.claude/commands/playtest.md` step 7)
for each roster id. Combined report shape:

```markdown
# Multi-Agent Playtest Report: <scenario name>

**Date:** <YYYY-MM-DD>
**Scenario:** <name> (mode: <mode>)
**Agents:** <id> (<role>), <id> (<role>), ...
**Server:** <target> (AI port)

## Summary
<2-4 sentences on the run arc across agents.>

## Group Goal Results
- [x] <goal id>: <do> — PASS: <cross-agent evidence>
- [ ] <goal id>: <do> — FAIL: <observed vs. expected, which agents>

## Per-Agent Outcomes
- <id> (<role>): <one-line outcome; link/reference its per-agent report>

## Findings
(Merged from all agents' blackboard findings, deduped, tagged by agent. Keep the
BUG/CONCERN/OBSERVATION/PASS/FAIL/BLOCKED categories from per-agent reports.)
### BUG: <title> (<agent id>)
<repro: where, what was typed, what happened, what was expected>

## Stats
- Agents: <N>
- Group goals: <P>/<T> PASS
- Bugs / Concerns / Observations: <N> / <N> / <N>
```

## Conventions
- Name it `tools/playtest/reports/<date>-<scenario>.md`.
- Group-goal evidence should cite which agents observed what (e.g., "leader's GMCP
  party shows member; member's shows leader").
- Findings come from each agent's blackboard `findings` entries (the conductor
  reads them with `ptorch bb dump`), merged and deduped, each tagged by agent.
