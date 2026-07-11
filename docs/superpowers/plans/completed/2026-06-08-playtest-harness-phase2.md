# Playtest Harness Adoption — Phase 2 (group / multi-agent testing) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add multi-agent / group playtest scenarios to DOGMud's harness adoption — a DOGMud `/playtest-scenario` conductor driving `ptorch` + multiple `mudagent` sessions, plus DOGMud scenario content (parallel, party, adversarial/PvP).

**Architecture:** Approach A — the harness (`$GOMUD_HARNESS_DIR`, default `../gomud-playtest-harness`) stays engine-agnostic and provides the `ptorch` orchestrator + blackboard (`go run ./cmd/ptorch` from there). DOGMud owns a self-contained overlay: the conductor command, the per-agent runner, the report format, and DOGMud scenario files under `tools/playtest/`. Builds on Phase 1 (merged): the `playtest` module's per-round beacons (pacing) and the single-agent `/playtest` driver.

**Tech Stack:** YAML (scenarios) + Markdown (conductor/runner). The `ptorch` Go binary and `agent-runner.md` are reference sources at `$GOMUD_HARNESS_DIR`. No DOGMud Go changes.

**Spec:** `docs/superpowers/specs/2026-06-08-playtest-harness-adoption-design.md` (Phase 2).
**Reference (harness):** `cmd/ptorch/` (CLI: `ptorch scenario plan <file>`, `ptorch bb init …`), `framework/scenarios/{SCHEMA.md,template.yaml,examples/*}`, `framework/agent-runner.md`, `framework/multi-agent-report-format.md`, `.claude/commands/playtest-scenario.md` (conductor to adapt), `docs/pr/2026-06-07-multi-agent-testing-pr.md`.

## DOGMud facts (verified 2026-06-08)
- **Concurrent AI:** `Network.MaxAIConnections: 20` (separate AI-port cap) — multi-agent rosters of 2–3 are well within it.
- **PvP:** `GamePlay.PVP` (`enabled`|`limited`|`disabled`, default **disabled**) + `GamePlay.PVPMinimumSkillRanks` (default **15**). DOGMud gates PvP by **skill ranks, not levels** (no levels exist). So the scenario schema's `requires.minimum_level` is **N/A** for DOGMud → surface `PVPMinimumSkillRanks` instead.
- **No permadeath** in DOGMud → `requires.permadeath`/`perma_death_protection` are **N/A** (defeat→respawn always; the module's `DeathProtection` is a no-op, per Phase 1).
- **Party:** `internal/usercommands/party.go` exists (a `party` command with subcommands) — cooperative party mode is feasible. Exact subcommands are verified in Task 3.

---

## File Structure
- `.claude/commands/playtest-scenario.md` — **new** DOGMud conductor (adapted from harness).
- `tools/playtest/agent-runner.md` — **new** per-agent runner (copied/adapted; DOGMud overlay paths).
- `tools/playtest/multi-agent-report-format.md` — **new** combined-report spec (copied).
- `tools/playtest/scenarios/parallel-coverage.yaml` — **new** (independent agents).
- `tools/playtest/scenarios/party-formation.yaml` — **new** (cooperative party).
- `tools/playtest/scenarios/adversarial-contest.yaml` — **new** (PvP/contested; server-config-gated).

Build order: conductor + runner + report-format first (the plumbing), then the three scenarios, then the live smoke.

---

## Task 1: DOGMud multi-agent conductor + runner + report format

**Files:**
- Create: `.claude/commands/playtest-scenario.md` (from harness `.claude/commands/playtest-scenario.md`)
- Create: `tools/playtest/agent-runner.md` (from harness `framework/agent-runner.md`)
- Create: `tools/playtest/multi-agent-report-format.md` (from harness `framework/multi-agent-report-format.md`)

- [ ] **Step 1: Copy the three reference files**
```bash
HARNESS="${GOMUD_HARNESS_DIR:-../gomud-playtest-harness}"
cp "$HARNESS/.claude/commands/playtest-scenario.md" .claude/commands/playtest-scenario.md
cp "$HARNESS/framework/agent-runner.md" tools/playtest/agent-runner.md
cp "$HARNESS/framework/multi-agent-report-format.md" tools/playtest/multi-agent-report-format.md
```
Read all three fully before editing.

- [ ] **Step 2: Adapt the conductor (`.claude/commands/playtest-scenario.md`)**
Apply exactly these DOGMud changes; keep all other conductor logic:
1. **Scenario path:** `framework/scenarios/<name>.yaml` (and `examples/`) → `tools/playtest/scenarios/<name>.yaml`.
2. **ptorch invocation:** resolve the harness via `HARNESS="${GOMUD_HARNESS_DIR:-../gomud-playtest-harness}"` and run `go run ./cmd/ptorch` **from `$HARNESS`** (or a prebuilt `$HARNESS/ptorch[.exe]` if present), passing the **DOGMud-absolute** scenario path. If `$HARNESS` is missing, STOP with the same message the `/playtest` driver uses.
3. **Run/bridge dir:** `.playtest/$RUN/` → `tools/playtest/.run/$RUN/` (blackboard + per-agent bridge dirs).
4. **Per-agent runner:** point each spawned subagent at `tools/playtest/agent-runner.md` (not `framework/agent-runner.md`), and have it read the DOGMud overlay (`tools/playtest/{engine-profile,targets,personalities,goals}`) — same as the `/playtest` driver.
5. **DOGMud `requires` surfacing** (the conductor surfaces preconditions, never changes config). Replace the stock-GoMud key guidance with DOGMud's:
   - `max_connections` → DOGMud `Network.MaxAIConnections` (default 20). If the roster exceeds it, stop and tell the user to raise it or shrink the roster.
   - `pvp` → DOGMud `GamePlay.PVP` (`enabled`/`limited`/`disabled`). For a PvP scenario, tell the user to set it (default is `disabled`) and restart.
   - **No `minimum_level`** — surface `GamePlay.PVPMinimumSkillRanks` instead (default 15; testers may not meet it — tell the user to lower it or rank the testers up).
   - **No permadeath** in DOGMud — note `permadeath`/`perma_death_protection` are N/A; outcomes are defeat→respawn.
6. **Charset:** each agent ensures ASCII via the same toggle-until-ASCII logic the `/playtest` driver uses (the runner inherits it — confirm the runner includes it; if not, add the same note).
7. **Usage line:** `/playtest-scenario <scenario-name>`.

- [ ] **Step 3: Adapt the runner (`tools/playtest/agent-runner.md`)**
Change any `framework/` paths to `tools/playtest/`, any `.playtest/` to `tools/playtest/.run/`, and ensure it reads the DOGMud engine-profile/personalities/targets. Confirm it includes the **charset-ensure** step (toggle-until-ASCII, matching `/playtest`); add it if missing.

- [ ] **Step 4: Verify no stale references**
Run: `grep -nE "framework/|\.playtest/|minimum_level|Network\.AI\.|MinimumLevel" .claude/commands/playtest-scenario.md tools/playtest/agent-runner.md`
Expected: no `framework/` or bare `.playtest/` paths; PvP guidance references `GamePlay.PVP`/`PVPMinimumSkillRanks`/`MaxAIConnections` (not `Network.AI.*`/`minimum_level`).

- [ ] **Step 5: Commit**
```bash
git add .claude/commands/playtest-scenario.md tools/playtest/agent-runner.md tools/playtest/multi-agent-report-format.md
git commit -m "feat(test): DOGMud multi-agent conductor + runner + report format (Phase 2)"
```

---

## Task 2: parallel-coverage scenario (independent agents)

**Files:**
- Create: `tools/playtest/scenarios/parallel-coverage.yaml`

- [ ] **Step 1: Author the scenario**
```yaml
name: parallel-coverage
mode: parallel
summary: Two testers independently exercise different DOGMud areas/systems at once — concurrency + coverage, no interaction expected.
requires:
  # DOGMud: no permadeath; defeat -> respawn. max_connections = Network.MaxAIConnections (20).
  max_connections: 20
roster:
  - id: explorer
    role: feel-tester
    target: local
    onboarding: auto
    goals:
      - id: wander-stillwater
        do: >-
          Explore the Stillwater area — move through several rooms reading each,
          note the town's NPCs and flavor.
        verify: >-
          Visited 5+ distinct rooms (beacon room_id changes), descriptions read
          coherently, no movement errors. Note overall feel.
  - id: shopper
    role: feature-tester
    target: local
    onboarding: auto
    goals:
      - id: shop-thornwall
        do: >-
          In Thornwall City, find a merchant, list wares, and buy then sell one item.
        verify: >-
          A shop listing renders; gold decreases on buy and increases on sell
          (check status/Char.Vitals gold and the item entering/leaving inventory);
          no 'looks confused' or generic error.
group_goals:
  - id: concurrency
    do: Both agents play simultaneously for the session.
    verify: >-
      Both ran concurrently with no crash/disconnect; the server accepted two
      AI connections at once (no AI-pool-full rejection).
```

- [ ] **Step 2: Validate with ptorch**
Run: `(cd "${GOMUD_HARNESS_DIR:-../gomud-playtest-harness}" && go run ./cmd/ptorch scenario plan "$(pwd)/../DOGMud/tools/playtest/scenarios/parallel-coverage.yaml")`
(or pass the DOGMud-absolute path). Expected: exit 0, JSON with `mode: parallel`, a 2-entry roster, and `max_connections`. If it errors, fix the YAML to the schema.

- [ ] **Step 3: Commit**
```bash
git add tools/playtest/scenarios/parallel-coverage.yaml
git commit -m "feat(test): DOGMud parallel-coverage scenario"
```

---

## Task 3: party-formation scenario (cooperative)

**Files:**
- Create: `tools/playtest/scenarios/party-formation.yaml`

- [ ] **Step 1: Verify DOGMud party commands**
Read `internal/usercommands/party.go` and note the exact subcommands for **inviting** and **accepting** a party (e.g. `party invite <name>` / `party accept`, or `party join`). Use the real command strings in the scenario's `do` lines. Also confirm how two characters end up in the same room to party (both at `local`; they'll need to meet — pick a known shared start room, e.g. the Sanctum Basin or Thornwall square).

- [ ] **Step 2: Author the scenario** (using the verified party commands; replace `<party-invite>` / `<party-accept>` with the real strings from Step 1)
```yaml
name: party-formation
mode: party
summary: Two testers meet, form a party (invite/accept), group up, and confirm party membership + shared play.
requires:
  max_connections: 20
roster:
  - id: leader
    role: feature-tester
    target: local
    onboarding: auto
    goals:
      - id: invite
        do: >-
          Meet the other tester in a shared room, then invite them to a party
          using the party command (<party-invite>).
        verify: >-
          An invite is sent; once accepted, you are party leader and the roster
          shows two members.
  - id: joiner
    role: feature-tester
    target: local
    onboarding: auto
    goals:
      - id: accept
        do: >-
          Accept the leader's party invite (<party-accept>) and confirm you are in
          the party.
        verify: >-
          You join the party; party status lists both members.
group_goals:
  - id: grouped
    do: Both agents form one party and confirm membership from both sides.
    verify: >-
      Both agents report being in the same party (two members), coordinated via
      the blackboard (leader signals 'invited', joiner signals 'joined'). No crash.
```
NOTE: the agents coordinate timing via the blackboard (leader sets an `invited` signal; joiner waits on it then accepts) — the conductor/runner handle blackboard signals; the scenario just declares the intent in `do`/`verify`.

- [ ] **Step 3: Validate + commit**
```bash
# validate (as in Task 2 Step 2, for party-formation.yaml) -> expect mode: party, 2 roster
git add tools/playtest/scenarios/party-formation.yaml
git commit -m "feat(test): DOGMud party-formation scenario"
```

---

## Task 4: adversarial-contest scenario (PvP — server-config-gated)

DOGMud PvP ships **disabled**; this scenario file + the conductor's precondition surfacing are built now, but a live run requires the user to enable PvP first. Author a defeat→respawn contest (no permadeath in DOGMud).

**Files:**
- Create: `tools/playtest/scenarios/adversarial-contest.yaml`

- [ ] **Step 1: Author the scenario**
```yaml
name: adversarial-contest
mode: adversarial
summary: >-
  Two testers contest each other (a PvP duel and/or a contested resource) to
  exercise PvP combat resolution. Defeat -> respawn (DOGMud has no permadeath).
requires:
  # DOGMud PvP is OFF by default. To run live, a human must set on the server:
  #   GamePlay.PVP: enabled   (or 'limited' + meet in a PvP-flagged room)
  #   GamePlay.PVPMinimumSkillRanks: <low enough that the testers qualify>
  # then restart. DOGMud has NO permadeath, so outcomes are defeat -> respawn.
  pvp: enabled
  max_connections: 20
roster:
  - id: red
    role: bug-finder
    target: local
    onboarding: auto
    goals:
      - id: engage
        do: >-
          Meet the other tester in a shared (PvP-permitted) room and attempt to
          engage them in combat (attack), playing to win but fairly.
        verify: >-
          Combat resolves between the two players (swings/defenses shown); a
          defeat results in a respawn, not a permanent loss. No crash or
          'looks confused'.
  - id: blue
    role: bug-finder
    target: local
    onboarding: auto
    goals:
      - id: defend
        do: Defend / fight back when engaged; use flee or a heal if losing.
        verify: >-
          Defenses and counter-swings resolve; if defeated, you respawn and can
          continue. PvP rules behave consistently.
group_goals:
  - id: pvp-resolves
    do: A PvP encounter between the two testers occurs and resolves cleanly.
    verify: >-
      At least one PvP combat round resolves between the two players, with a
      consistent outcome (defeat->respawn), coordinated via the blackboard
      ('ready-to-fight' signal). No crash, no rule contradiction.
```

- [ ] **Step 2: Validate + commit**
```bash
# validate adversarial-contest.yaml with ptorch -> expect mode: adversarial, requires.pvp surfaced
git add tools/playtest/scenarios/adversarial-contest.yaml
git commit -m "feat(test): DOGMud adversarial-contest (PvP) scenario (server-config-gated)"
```
(The live adversarial run is deferred until the user enables PvP on the server; the conductor will surface the `requires.pvp` precondition.)

---

## Task 5: Acceptance — live multi-agent smoke (interactive)

**Files:** none (verification). **This is interactive + token-heavy (2 concurrent LLM agents) — it lands with the user.**

- [ ] **Step 1: Boot the server** (AI port on; `Modules.playtest.Enabled: true` already). Confirm `Network.MaxAIConnections >= 2`.
- [ ] **Step 2: Validate all scenarios parse**
Run `ptorch scenario plan` on each of the three scenario files; all exit 0 with the expected `mode`.
- [ ] **Step 3: Run the cheapest multi-agent scenario**
`/playtest-scenario parallel-coverage` (2 independent agents — lowest cost, no PvP/party setup). Confirm: the conductor surfaces requires/cost; `ptorch bb init` seeds the blackboard; **two `mudagent` sessions connect concurrently** (no AI-pool-full); each agent plays its goals; per-agent reports + a **combined multi-agent report** land under `tools/playtest/reports/`.
- [ ] **Step 4 (optional, user-driven):** `party-formation` (cooperative) once party commands are confirmed; and `adversarial-contest` only after the user enables `GamePlay.PVP` + lowers `PVPMinimumSkillRanks` + restarts.
- [ ] **Step 5: Record results / file findings** as a Phase-2 followup list (e.g. concurrency limits, party-command friction, PvP behavior).

---

## Self-Review notes (addressed)

- **Spec coverage (Phase 2):** conductor + runner + report format (T1); parallel-coverage (T2), party-formation (T3), adversarial/PvP (T4) scenarios; acceptance smoke (T5). All Phase-2 spec items covered.
- **DOGMud divergence handled:** no `minimum_level` → `PVPMinimumSkillRanks`; no permadeath → N/A; `max_connections` → `Network.MaxAIConnections` (20); PvP off by default → adversarial run is config-gated; party commands verified from `party.go` before authoring (T3 Step 1).
- **Placeholder scan:** `<party-invite>`/`<party-accept>` in T3 are explicit verify-then-fill tokens (Step 1 resolves them from `party.go`) — not plan placeholders. No TODO/TBD.
- **Type/path consistency:** all overlay paths `tools/playtest/...`; conductor + runner both read the same overlay; ptorch resolved via `GOMUD_HARNESS_DIR` exactly as the `/playtest` driver resolves `mudagent`.
- **Cost honesty:** T5 calls out the N× token cost of multi-agent runs and starts with the cheapest (2-agent parallel).
