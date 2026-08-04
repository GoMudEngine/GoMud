# Archify Diagrams Tab — Design

**Date:** 2026-08-04
**Status:** Approved (design); implementation not started
**Related:** `project_archify_diagrams_tab` memory, `project_web_overhaul_sequence`

## Goal

Add a public-facing "Under the Hood" page to the DOGMud web site presenting a
curated gallery of six interactive technical diagrams of the engine's more
distinctive systems.

**Audience: technical peers** — MUD hobbyists, Go developers, potential
contributors, recruiters. The framing is "here is how this is built," not "here
is what you experience as a player." Diagram content is engineering-grade and
uses real type and function names.

**Non-goals.** This is not internal documentation (Mermaid-in-markdown remains
the tool there), not a player-facing help system, and not auto-generated. Every
diagram is hand-authored against verified code and manually refreshed when the
architecture moves.

## Tooling

The `tt-a1i/archify` Claude Code skill, installed globally at
`C:\Users\Calabe Davis\.agents\skills\archify` (note: **not** under
`~/.claude/skills`). Node-only, fully offline; local Node v24 confirmed.

Archify is not a code analyzer. The agent performs the analysis, hand-authors a
typed JSON specification (one of five types), and the bundled Node CLI validates
layout and renders a self-contained interactive HTML file. Each delivered
artifact is roughly 625 KB and carries its own viewer runtime: pan/zoom, search,
theming, focus panels, guided chapters, and export.

## Diagram roster

Six diagrams: two `architecture`, two `sequence`, one `dataflow`, one
`lifecycle`. Archify's fifth type, `workflow`, is unused — no subject on this
roster is a process or runbook, and inventing one to fill the slot would be
padding.

| # | Diagram | Type | Subject |
|---|---------|------|---------|
| 1 | Engine Overview | `architecture` | Telnet/WebSocket front ends → input handlers → world tick loop → rooms/mobs/users → YAML data layer → modules (GMCP, web, playtest, achievements). The anchor diagram; every other diagram zooms into one box on this one. |
| 2 | Mob Aliveness Stack | `architecture` | Schedules, patrols, NPC↔NPC conversations, behavior trees, and idle command pools — and how they arbitrate for a single mob's tick. |
| 3 | Combat Round Resolution | `sequence` | One swing end to end: encumbrance → swing count, opposed `dice.RollStat`, best-of-all defense resolution, three-channel damage, mitigation caps, descriptive damage text out. |
| 4 | Template → Instance → Runtime | `sequence` | YAML template loads, instance save overlays it, `instance:"skip"`-tagged fields are restored back from the template by `restoreSkipTaggedFields`. |
| 5 | GMCP → Web Client | `dataflow` | Server state → GMCP packet types → JS handlers → SVG leather mapper, with fog of war filtered server-side by `Character.VisitedRooms`. |
| 6 | Use-Based Progression Loop | `lifecycle` | No XP, no levels: `OnStatUse` / `OnSkillUse` → probabilistic advancement → `StatProgressionSoftCap` as a brake rather than a ceiling. |

**Phasing.** Diagrams 1–3 ship together with the index page and nav tab, proving
the hosting story end to end. Diagrams 4–6 follow as a second pass. Each diagram
is a real authoring session; phasing yields a live page sooner rather than six
half-validated specifications.

**Considered and rejected:** an AI-playtest-harness diagram. It is arguably the
most memorable item available for this audience, but it documents the test
harness rather than the engine. Dropped in favour of the progression loop.

## Hosting and routing

### Files

```
_datafiles/html/public/
  architecture.html                 templated index page (site header/nav/footer)
  diagrams/                         NOTE: must not share a name with the index
    engine-overview.html            raw archify artifact, ~625 KB
    mob-aliveness.html
    combat-round.html
    data-load.html
    gmcp-webclient.html
    progression-loop.html

tools/archify/specs/
  *.json                            authored source specifications (committed)

internal/web/
  architecture_test.go              guard test
```

### How each URL serves

`/architecture` — `serveTemplate` (`internal/web/web.go:64`) stats the path,
finds nothing, appends `.html`, finds `architecture.html`, globs `_header.html`
and `_footer.html` into the template set, and executes with the standard
`templateData` map. Full site chrome.

> **The artifact directory must NOT be named `architecture/`.** Corrected
> 2026-08-04 after Task 3 hit this for real. `serveTemplate` stats the
> extension-less path *before* appending `.html` (`web.go:78-85`); if a
> directory of that name exists the stat succeeds, the `info.IsDir()` branch
> takes over, and the handler looks for `architecture/index.html` — which does
> not exist — and returns **404**. Verified live: `/architecture` → 404,
> `/architecture.html` → 200. Hence `diagrams/`. The general rule: **an index
> page's URL path may never equal a sibling directory's name**, unless that
> directory actually contains the `index.html` you want served.

`/diagrams/mob-aliveness.html` — goes through the **same** templating path.
It serves byte-identical because:

- `web.go:16` imports **`text/template`**, not `html/template`, so there is no
  contextual escaping to mangle inline SVG, CSS, or JavaScript.
- Archify output contains **zero `{{` sequences** (verified across all five
  bundled example artifacts), so the file parses as a single literal text node.
- The `_*.html` glob of the request directory (`web.go:230`) finds nothing in
  `public/diagrams/`.

**No engine change is required for hosting.** Keeping the generated artifacts in
their own directory also gives the guard test a clean invariant — *everything*
under `public/diagrams/` is frozen generated output, with no templated-file
exception that could rot as pages are added.

### Navigation

One entry appended to the hardcoded core slice at `internal/web/web.go:147`:

```go
{`Architecture`, `/architecture`},
```

placed after `Web Client`. Plugin-supplied nav items sort *after* the core slice
(`coreCount`, `web.go:158`), so Achievements / Leaderboards / Help keep their
current order.

### The fragile seam, and the guard

Byte-identical pass-through holds only while the artifact contains no `{{`. A
future archify version, or an authored node label containing `{{`, would turn a
625 KB static file into a template parse error and an HTTP 500.

**Decision: ship-time check, not runtime bypass.** A Go test asserts the
property in CI (see Verification). The alternative — extending `web.go` with a
static-passthrough prefix served by `http.ServeFile` regardless of extension —
would make the guarantee structural rather than conventional, but touches shared
engine code for a content feature. Revisit only if third-party HTML beyond
archify output is later placed under `public/`.

## The index page

`gomud.css` is thin (10 KB: `.overlay`, `.hero-*`, `.brass`, `.gomud-btn`) and
has no grid or card classes. The page therefore carries a scoped `<style>`
block, matching the existing pattern in `_header.html`. `gomud.css` is not
modified.

Structure, following `online.html`:

```
{{template "header" .}}
  <div class="overlay">
    <h1>Under the Hood</h1>
    <p>Two or three sentences of framing.</p>

    <div class="diagram-grid">              CSS grid, auto-fit minmax(300px, 1fr)
      <a class="diagram-card"
         href="/diagrams/engine-overview.html"
         target="_blank" rel="noopener">
        <span class="diagram-kind">ARCHITECTURE</span>
        <h3>Engine Overview</h3>
        <p>What it shows — one sentence.</p>
        <p class="diagram-why">Why it is interesting — one sentence.</p>
        <span class="brass">View →</span>
      </a>
      ... x6
    </div>
  </div>
{{template "footer" .}}
```

Visual language follows the site's existing warm-dark cartographer palette and
brass accents. The type badge (`ARCHITECTURE` / `SEQUENCE` / `DATAFLOW` /
`LIFECYCLE`) is a brass chip.

**No thumbnails.** Archify's PNG and 1200×630 Share Card exports are *viewer
runtime* features that run in the browser; `archify --help` exposes no headless
export command. Generating thumbnails would mean opening each artifact by hand
and saving an image, producing a second asset that drifts independently of the
diagram it depicts. Text cards with type badges also suit the site's spartan
look better than six screenshot tiles. Images remain an additive polish pass if
wanted later.

**Cards open in a new tab** (`target="_blank" rel="noopener"`). The delivered
artifact is frozen after validation and must not be edited, so it cannot be
given a "back to gallery" affordance. A new tab keeps the gallery in place and
gives the diagram a clean full viewport, which is what archify's viewer is
designed for.

### Card copy

Twelve sentences total (what / why for each diagram). Drafted alongside each
diagram's specification rather than upfront — the finished diagram determines
its own headline. Target reader: a MUD-hobbyist developer who should think
"that is a good idea" rather than "I see, a box diagram."

## Readability within a diagram

Every generated artifact already ships these reader capabilities; they require
no authoring work:

- **Reading Depth** — detail adapts to zoom. MAP below 100%, READ at the default
  100%, FULL detail at 175%.
- **Focus + Semantic Passport** — clicking a node focuses it and opens a panel
  of authored upstream/downstream facts with a copyable deep link. Closes on
  Escape or outside activation. Never enters canonical export.
- **Node Finder** (searches labels and stable IDs), **Semantic Lens**
  (summarizes selected node/relationship kinds without changing geometry),
  **Route Probe** (resolves exactly two endpoints over authored directed
  relationships), **Semantic Radar** (viewport minimap).

**Authored addition: `meta.views` on all six diagrams.** Up to five curated
chapters per diagram, each an array of stable node IDs. That single array drives
the Named Chapter Rail, Story Beat Navigator, and follow camera — a guided tour
where each beat frames one region while the rest recedes. This is the primary
mechanism making a dense diagram self-explanatory, at roughly 20% additional
authoring cost per diagram.

Constraints from the skill's contract: chapters may reference only authored
nodes, and story transitions classify only the exact relationship between
adjacent authored stops. Never imply a transitive edge, verb, or causality from
story order or proximity.

`meta.animation` is left at its static default. Motion is opt-in and is not
needed here.

## Authoring loop

Per diagram:

1. **Establish ground truth with codegraph.** `codegraph_context` on the
   subsystem, then one `codegraph_explore` over the symbols it surfaces. The
   diagram must name real types and real call paths — a showcase diagram that
   misstates the architecture is worse than no diagram, because this audience
   will notice.
2. **Hand-author the JSON specification.** At most 12 primary nodes, one obvious
   main path, short side branches, sparse labels. `meta.quality_profile:
   "showcase"`. `meta.views` with at most five chapters. Artifact-first: write
   the candidate before inspecting renderer internals. Do not add `via`,
   `channelX`, `channelY`, or `labelAt` before a diagnostic calls for one.
3. **Validate to zero.**
   `node bin/archify.mjs validate <type> <spec>.json --quality showcase --json`,
   repaired until all **9 artifact checks report 0 composition errors and 0
   warnings**. A receipt listing only 4 checks is basic validation, never
   showcase acceptance.
4. **Deliver once.**
   `node bin/archify.mjs deliver <type> <spec>.json <output>.html --quality showcase --json`.
   Delivery freezes the exact specification bytes into a private snapshot,
   renders and checks that snapshot, atomically commits the HTML, and reports
   SHA-256 plus byte counts. A non-zero exit is never reported as success.
5. **Guard.** `grep -c '{{'` on the delivered artifact must return 0.

A passing final validation freezes the candidate. **The artifact is never edited
afterward.** If a change is needed, edit the specification and re-deliver.

### Specifications are source

The authored `.json` files are committed to `tools/archify/specs/`. Without
them, refreshing a drifted diagram means re-authoring from scratch; with them it
is an edit plus a re-`deliver`. This is the difference between these artifacts
aging like helpfiles and aging like code.

### Open item

Archify accepts a `--repo-root` flag and its authoring contract describes a
"repository evidence" mode that may allow nodes to cite real files. If it does
what it appears to, it is a meaningful anti-drift lever. **Unverified.** Check
while authoring diagram #1 and report findings rather than designing around a
guess.

## Verification

**Automated —`internal/web/architecture_test.go`:**

- Walks `_datafiles/html/public/diagrams/*.html` and fails if any file
  contains `{{`. This is the guard for the templating seam.
- Parses `architecture.html` for `href="/diagrams/..."` targets and fails if
  a referenced file does not exist, catching typo'd card links.
- Go test binaries run with CWD set to their package directory, so this test
  must chdir to the repository root first, following the existing pattern in
  `internal/web/auth_test.go`.

**Agent smoke pass:** confirm the page renders, all six links resolve, and the
guard test passes. The local server is started and stopped by the user only —
never by the agent.

**Human verification is the user's.** The user views the diagrams on the local
web page and judges legibility, accuracy, and whether the copy lands. The
CLAUDE.md playtest-harness gate does not literally apply here — `mudagent`
drives telnet and cannot see a web page — but the principle behind it does: a
clean boot and a passing test prove nothing about whether a diagram is readable.

## Known limitations

- **Drift.** These are hand-authored snapshots with no CI check tying them to
  the code they depict. They will go stale as the architecture moves, the same
  failure class as the helpfile drift found in the 2026-08-03 audit. Committed
  specifications reduce the cost of a refresh but do not detect the need for
  one.
- **Repository weight.** Six artifacts at roughly 625 KB each is about 3.7 MB
  committed and deployed to the droplet.
- **Manual refresh.** Updating a diagram is a deliberate authoring session, not
  a build step.
