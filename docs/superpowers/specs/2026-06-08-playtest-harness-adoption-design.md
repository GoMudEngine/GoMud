# Adopt the GoMud Playtest Harness for DOGMud (comprehensive)

**Date:** 2026-06-08
**Status:** Design — approved for planning
**Author:** Calabe Davis + Claude
**Plans:** two — Phase 1 (foundation + server module), Phase 2 (group testing).

## Problem

DOGMud's AI testing runs on a fragile, bespoke stack: `tools/mud_bridge.py`
(file-IPC telnet bridge paced by chained `sleep 2`, ANSI-stripped) driven by
`/test-mud`, plus a standalone `tools/ai_player.py`. We have since built a far
better, engine-agnostic harness (`~/workspace/gomud-playtest-harness` =
`GoMudEngine/GoMud-Module-Playtest-Harness`): a `mudagent` adapter (line-in /
JSON-line-out: `output`/`gmcp`/`status`/`beacon` events), a `playtest` **server
module** (per-round beacons, safe mode, death protection, sandbox), a `ptorch`
**multi-agent orchestrator** with a blackboard + scenario schema, and a
reference `/playtest` driver.

This work adopts the harness for DOGMud **in full** — not just the transport
layer. The point is to gain the capabilities we *don't* have today: reliable
beacon-paced/structured verification, server-side test safety, and **group /
multi-agent testing**. Adopting only the adapter would be mere parity and isn't
worth doing.

## Goals

- Drive DOGMud playtests through the engine-agnostic `mudagent` adapter.
- Run DOGMud's `playtest` **server module** (beacons, safe mode, death
  protection, sandbox) so single-agent runs are safe and beacon-paced with
  structured per-round state — beyond what we have now.
- Run **multi-agent / group scenarios** (`ptorch`: party, parallel-coverage,
  adversarial/PvP) against DOGMud — a brand-new capability.
- Keep DOGMud's test config as a self-contained **overlay in the DOGMud repo**
  (`tools/playtest/`); keep the harness repo pristine/engine-agnostic. The only
  external dependencies are the `mudagent` + `ptorch` binaries.
- Retire the superseded stack (`mud_bridge.py`, `ai_player.py`, `/test-mud`, old
  `tools/testing/`) into an archive, preserving history.

## Approach (A)

The harness stays engine-agnostic; DOGMud owns a thin overlay + driver and
vendors only the server module. The `mudagent`/`ptorch` binaries are
engine-agnostic and consume DOGMud's overlay via the driver — no harness code
changes. The harness location resolves via `GOMUD_HARNESS_DIR` (default
`../gomud-playtest-harness`); the driver prefers a prebuilt binary, else
`go run ./cmd/<tool>` from there.

## Phasing (two implementation plans)

Every phase ships differentiating value — never parity. Phase 1 includes the
server module because beacons/safe-mode/death-protection are the foundation
group testing relies on (reliable pacing; survivable adversarial/PvP runs).

---

# Phase 1 — Foundation + server module

**Deliverable:** safe, beacon-paced, structured single-agent DOGMud playtests via
`mudagent` + the vendored `playtest` module + a DOGMud `/playtest` driver, with
the old stack retired.

## 1A. DOGMud playtest overlay — `tools/playtest/`
Mirrors the harness `framework/` layout so driver logic is a thin adaptation:

- **`engine-profile.yaml`** — the one place DOGMud specifics live:
  - `engine: DOGMud`; `setup_commands: ["set charset"]` (force ASCII).
  - `commands:` DOGMud verbs (`look`, movement, `inventory`, `status`,
    `get/drop`, `attack`, `flee`, `say`, `map`, `consider`, special moves
    `kick/bash/taunt`, `recover: ["cast <heal> me", "drink <potion>"]`, `help`).
  - `world:` starts in **Sanctum Basin**; DOGMud char-creation onboarding;
    compass + `look`/`look <noun>` + `map` orientation.
  - `mechanics:` **three pools — Health (HP) / Stamina (SP) / Conviction (CP)**;
    **no levels/XP** (use-based stat progression, six 100-centered stats);
    `death:` **justice/jail** (arrest/bounty), not permadeath; bleedout/downed
    removed; `combat:` `attack`/`flee`, **4-second rounds**, special moves +
    `cast`.
  - `notes:` AI-port output is ANSI-stripped; don't file display artifacts;
    NPC targets are exact room keywords.
- **`targets.yaml`** — `local` (`localhost:55555`, `user: smoketester`) and
  `prod` (`dogmud.org:55555`, `user: aitester`) with existing AI-flagged creds.
- **`personalities/`** — copies of `bug-finder.md`/`feature-tester.md`/
  `feel-tester.md`, lightly DOGMud-flavored (justice/jail awareness, three-pool
  survival, charset note). Copied so the overlay is self-contained.
- **`goals/`** — a **curated subset** of currently-relevant DOGMud goals migrated
  to the harness schema (the plan names the exact list; stale chunk-smokes are
  archived). Schema delta: `description`→`name`+`summary`; `goals[].text`→`do`+
  `verify`; `exit_criteria`→`pass_criteria`.
- **`reports/`** (gitignored except `.gitkeep`), `.run/` (stdio bridge,
  gitignored).

## 1B. Vendor the server module — `modules/playtest/`
Vendor `module/playtest/` into DOGMud's `modules/` (matching how DOGMud bundles
`auctions`/`cleanup`/`follow`/`gmcp`), registered via `init()` against
`internal/plugins`. Module config (`Enabled`/`SafeMode`/`SandboxZoneTag`/
`DeathProtection`/`Beacons`) comes from the module's own
`files/data-overlays/config.yaml` via the plugin config API — **independent of**
DOGMud's `AIPort` server config (the earlier `Network.AI.*` concern was a red
herring). **Known DOGMud adaptation surface (the plan verifies + adapts each):**

- **AI-connection detection.** Module uses `connections.ConnType() ==
  connections.ConnAI`. Verify DOGMud tags AI-port connections as `ConnAI`; if
  not, adapt `isAIConnection`.
- **Beacons.** Module calls the gmcp module's exported `SendGMCPEvent(int,
  string, any)` via `plugins.GetPluginRegistry().GetExportedFunction`. Verify
  DOGMud's (diverged) gmcp module exports it; adapt the payload to DOGMud's
  `Character` fields and **add Conviction (CP)** to the beacon
  (`{round,hp,hp_max,sp,sp_max,cp,cp_max,room_id}`) — DOGMud has three pools, the
  stock beacon only carries two.
- **Death protection.** Module's `onPlayerSpawn` grants extra-lives on
  `events.PlayerSpawn`. Map to DOGMud's death model (justice/jail/respawn, no
  bleedout) — confirm what "death protection" means for DOGMud and wire
  accordingly (may be a no-op or a justice-exemption rather than extra-lives).
- **Safe mode / sandbox.** Uses `events.RoomChange` + `rooms.LoadRoom/MoveToRoom`
  + room `Tags` + `plug.ReserveTags`. DOGMud has all of these; define a DOGMud
  **sandbox zone + tag** (or leave `SandboxZoneTag` empty to disable confinement
  initially). Beacons + death-protection are the load-bearing parts; sandbox is
  optional.
- **Admin pages.** Module registers `plug.Web.AdminPage(...)`. Verify DOGMud's
  plugin Web API matches that signature; adapt if diverged.
- **Events/fields.** `events.NewRound.RoundNumber`, `events.RoomChange`,
  `events.PlayerSpawn` — confirm presence/shape in DOGMud (all used elsewhere in
  DOGMud, so expected present).

## 1C. DOGMud driver — `.claude/commands/playtest.md`
Adapted from the harness reference driver: reads `tools/playtest/*` (not
`framework/`), resolves the harness via `GOMUD_HARNESS_DIR`, bridges `mudagent`
stdio through `tools/playtest/.run/`, drives the JSON event loop (wait for
`status: logged_in` or drive char-creation; pace on `beacon` events once the
module is live; judge goals from `output`/`gmcp`/`beacon`), writes a report to
`tools/playtest/reports/`. Usage: `/playtest <local|prod> <personality>
[goals-file]`. Replaces `/test-mud`.

## 1D. Retire the old stack — `tools/_archive/testing-pre-harness/`
`git mv` `tools/mud_bridge.py`, `tools/ai_player.py`,
`.claude/commands/test-mud.md`, and `tools/testing/**` (roles/un-migrated goals/
reports/audits) into the archive with a dated `README.md` noting the
replacement. History preserved.

## Phase 1 testing / acceptance
- **Module unit tests:** the module ships `*_test.go` (beacons/safemode/config);
  keep them green after adaptation (`go test ./modules/playtest/...`).
- **Build/boot:** `go build ./...`; boot DOGMud (AI port 55555) and confirm clean
  start + the module loads (no warning that gmcp `SendGMCPEvent` is missing).
- **Acceptance (manual):** `/playtest local feel-tester` — adapter connects,
  `status: logged_in` (or clean char-creation), `set charset` ran, **`beacon`
  events arrive each round carrying hp/sp/cp/room**, several rounds of play, a
  report lands. Then one migrated goal with `feature-tester` (goal-by-goal
  PASS/FAIL). Confirm `bug-finder` free-form (no goals file) works. Confirm
  safe-mode death-protection lets a tester survive a lethal fight.

---

# Phase 2 — Group / multi-agent testing

**Deliverable:** run multi-agent scenarios against DOGMud — the standout new
capability. Depends on Phase 1 (beacons for pacing; death-protection/safe-mode
for survivable adversarial/PvP runs).

## 2A. Scenario content — `tools/playtest/scenarios/`
DOGMud scenarios authored to the harness scenario schema
(`framework/scenarios/SCHEMA.md`): at least **party-formation** (cooperative
group), **parallel-coverage** (N agents covering different zones/systems at
once), and one **adversarial/PvP** scenario (uses death-protection). Plus the
multi-agent report format.

## 2B. Multi-agent driver — `.claude/commands/playtest-scenario.md`
Adapted from the harness, driving `ptorch` (the orchestrator + blackboard) over
DOGMud's overlay + multiple `mudagent` sessions; resolves the harness via
`GOMUD_HARNESS_DIR`; writes a multi-agent report. Usage:
`/playtest-scenario <scenario>`.

## 2C. DOGMud fit checks (the plan verifies)
- **Multiple AI sessions** on DOGMud's AI port (the module beacons every AI
  connection; confirm `AICommandsPerRound`/`MaxConnections` allow N testers).
- **PvP/adversarial** requires player-vs-player to be possible between testers
  in a sandbox/zone — confirm DOGMud allows it (or stage in a PvP-permitted
  room), and that death-protection keeps both testers alive.
- Blackboard coordination is harness-side (no DOGMud change).

## Phase 2 testing / acceptance
- Run `/playtest-scenario parallel-coverage` (lowest-risk: independent agents) →
  N reports + a merged multi-agent report; confirm N concurrent AI sessions hold.
- Run `/playtest-scenario party-formation` (cooperative) and one adversarial/PvP
  → confirm coordination via blackboard and that death-protection prevents tester
  loss.

---

## Files touched (both phases)

**Phase 1:** new `tools/playtest/{engine-profile.yaml,targets.yaml,
personalities/*.md,goals/*.yaml,reports/.gitkeep}`; new
`.claude/commands/playtest.md`; new `modules/playtest/**` (vendored + adapted);
new `tools/_archive/testing-pre-harness/README.md`; `git mv` of the old stack;
`.gitignore` (ignore `tools/playtest/.run/` + `reports/*`). Module registration
wired into DOGMud's module load (matching other `modules/`).

**Phase 2:** new `tools/playtest/scenarios/*.yaml`; new
`.claude/commands/playtest-scenario.md`.

## Risks

- **Server-module integration (Phase 1B)** is the main risk — beacons (gmcp
  export + Conviction field), AI-connection detection (`ConnAI`), and the death
  model are the points most likely to need real adaptation against DOGMud's
  divergence. Each is independently verifiable; the plan front-loads these
  checks before wiring.
- **gmcp divergence:** DOGMud's gmcp payloads differ; the adapter passes them
  through generically, and beacons use the module's own `Playtest.Round` package,
  so the agent doesn't depend on DOGMud's exact gmcp shapes — but the beacon
  payload must read DOGMud's actual `Character` field names.
- **PvP availability (Phase 2)** may need a designated PvP room/zone.

## Out of scope

- Harness repo changes (stays engine-agnostic).
- Auto-discovery/versioning of the harness binaries beyond `GOMUD_HARNESS_DIR`.
- Upstreaming any DOGMud-specific beacon/death adaptations back to the harness
  (possible later, but not required here).
