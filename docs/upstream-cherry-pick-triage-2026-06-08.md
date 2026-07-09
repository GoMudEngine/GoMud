# Upstream (GoMudEngine/GoMud) cherry-pick triage — 2026-06-08

## Scope

Review `upstream/master` (`256c981d`) for work worth bringing into
`pruuk/DOGMud`, per [[project_upstream_rereview_webclient]]. Rule (from
`github_guide.md`/CLAUDE.md): **never merge upstream wholesale, never push to
it** — selective only.

- **Merge-base:** `d21fa121` (2025-11-04) — ~7 months back.
- **Upstream commits since:** **234** (158 touch web/frontend/admin).
- **PR-level feature units:** ~110 squash-merges (`#460`–`#603`).

## Headline finding: divergence makes `git cherry-pick` impractical

DOGMud has rewritten large swaths of the engine and UI since the merge-base
(combat/stat pipeline, levels/XP removed, factions/schedules/aliveness, a custom
web dashboard, files moved). The divergence is also **bidirectional** — some
upstream "fixes" were *ported from DOGMud* (e.g. PR **#603** web-client
password-leak, GHSA-m8fw-4ccp-94jw, says "Ported from the validated DOGMud fix";
DOGMud already has it).

Concretely, every high-value backend fix sampled touches files DOGMud has
**diverged on or removed**:

| Upstream PR | files | exist in DOGMud | DOGMud diverged on |
|---|---|---|---|
| #603 password-leak | 1 | 1 | 1 (DOGMud already fixed) |
| #460 nil-deref crashes | 8 | 5 | 5 |
| #461 type-assert crash | 4 | 4 | 4 |
| #462 bcrypt + perms | 5 | 4 | 4 |
| #463 system-cmd authz | 3 | 1 | 1 |
| #474 copyover | 60 | 22 | 20 |
| #593 world backups | 18 | 6 | 4 |

**Implication:** treat upstream as a source of *fixes and ideas to hand-port*,
not a branch to cherry-pick from. The memory's earlier hope ("tons of
web-client work to pull") is over-optimistic given the architectural drift.

### Frontend is architecturally incompatible

Upstream refactored the web client into a component/window system
(`webclient-core.js` + `static/js/windows/window-*.js`: map/character/comm/
vitals/gear/status). DOGMud went a different way (dashboard panels + monolithic
`gmcp.js` + winbox + leather mapper). Same *filenames* (`webclient-pure.html`),
**different architectures** → upstream frontend PRs won't apply; they're idea
sources only. DOGMud is arguably *ahead* on look/feel here.

### Admin is also structurally diverged

Upstream uses flat `admin/mobs.html`; DOGMud uses sectioned
`admin/<area>/index.html` and has its own sections (economy, combatstats,
progression, species). Upstream admin overhauls (#518, #520, #531/#533 themes,
#552 color picker, #527 syntax highlighting) won't drop in cleanly.

## Recommendation: tiered hand-port, not cherry-pick

### Tier 1 — Security / stability parity pass (do first; small, high value)
Hand-port the ones DOGMud genuinely lacks (verify each — DOGMud already has some,
e.g. #603). Read the upstream diff, re-apply the equivalent to DOGMud's diverged
file.
- **#460** four nil-pointer dereferences that crash the server
- **#461** comma-ok type assertion for UserObject in connection handlers
- **#462** bcrypt password hashing + file-permission hardening
- **#463** internal authorization check on system commands
- **#565** require admin role for web basic auth
- **#469** plaintext passwords require a password change
- (**#603** password echo — already in DOGMud; verify only)

Each is small and security/crash-relevant — worth the manual port despite
conflicts. **First step per item: confirm DOGMud doesn't already have it.**

### Tier 2 — Ops/infra features (evaluate; medium surface)
- **#474** copyover (hot-restart without dropping connections) — valuable for
  prod, but 60-file surface, heavy adaptation.
- **#593** automated world backups — useful; moderate surface.
- **module manager** system (#511/#512/#526 + module-manager commits) — infra
  for the GoMud-Modules registry; relevant to the planned playtest-harness
  module ([[project_playtest_harness]]).
- **#506** SSH connection support, **#474**/**#570** optimizations — optional.

### Tier 3 — Frontend / admin: harvest ideas, re-implement in DOGMud's structure
Not cherry-pickable. Worth mining specific UX wins and rebuilding natively:
- mapper improvements (#563/#566/#510/#485) — but DOGMud's leather mapper is its
  own thing; only mine ideas.
- admin **yaml viewer**, **config wizard** (#592), **color themes/picker**
  (#552/#531) — nice-to-haves for DOGMud's admin if desired.
- Cross-check [[project_webclient_vitals_reserved_pool_viz]] — not helped by
  upstream; still a DOGMud-native build.

### Tier 4 — Skip (design mismatch with DOGMud)
DOGMud removed levels/XP and has its own combat/stat/faction design:
- **#590** configurable level & stat/XP gain charts — N/A (no levels/XP).
- elections (#573/#575/#582), **#580** alignment decay, **#554** vitality,
  **#548** combat changes, **#591** elite mobs, pet overhaul (#556/#583/#584) —
  conflict with DOGMud systems.
- Optional *new* modules — fishing (#588), gambling (#505), MudMail (#513),
  fast-travel (#571), zombie mode (#509): could be added later as fresh modules
  if wanted, but that's feature work, not a cherry-pick.

## Suggested next action

Run a **Tier-1 security/stability parity pass** on a feature branch: for each of
the ~6 candidates, verify DOGMud lacks it, then hand-port + test. Small, bounded,
and the highest-value slice. Tiers 2–3 are separate, larger efforts to schedule
deliberately; Tier 4 is closed.
