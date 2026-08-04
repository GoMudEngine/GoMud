# Archify Diagrams Tab Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a public "Under the Hood" page at `/architecture` presenting six hand-authored interactive technical diagrams of the DOGMud engine, aimed at a technical-peer audience.

**Architecture:** Each diagram is a hand-authored JSON specification (committed to `tools/archify/specs/`) rendered by the archify CLI into a self-contained ~625 KB interactive HTML artifact (committed to `_datafiles/html/public/architecture/`). A templated index page links them as cards. No engine change is needed to host the artifacts — `internal/web/web.go` uses `text/template` and archify output contains no `{{` — so a Go test guards that property instead.

**Tech Stack:** Go 1.x (`internal/web`), Go `text/template` HTML pages, vanilla CSS using the site's existing custom properties, and the `archify` Node CLI at `~/.agents/skills/archify` (Node v24, fully offline).

**Branch:** `feature/archify-diagrams-tab` (already created; the design spec is committed there as `b9a23e3ba`).

**Spec:** `docs/superpowers/specs/2026-08-04-archify-diagrams-tab-design.md`

---

## Phasing

- **Phase A (Tasks 1–7)** — toolchain check, diagrams 1–3, the index page, the nav tab, the guard test. Ends with a working page the user reviews in a browser.
- **Phase B (Tasks 8–12)** — diagrams 4–6, patch notes, final verification.

Phase A must be reviewed by the user before Phase B starts.

---

## File Structure

| File | Responsibility |
|---|---|
| `tools/archify/specs/*.json` | **Create.** Six authored diagram specifications. These are the source of truth; artifacts are regenerable from them. |
| `_datafiles/html/public/architecture.html` | **Create.** Templated index page: intro copy + card grid + a scoped `<style>` block. |
| `_datafiles/html/public/architecture/*.html` | **Create (generated).** Six frozen archify artifacts. Never hand-edited. |
| `internal/web/web.go:147` | **Modify.** One line added to the core nav slice. |
| `internal/web/architecture_test.go` | **Create.** Two guard tests: no `{{` in artifacts, and every card link resolves. |
| `docs/PATCH_NOTES.md` | **Modify.** Dated entry, per the Pre-Push SOP. |

**Why `tools/archify/specs/` and not alongside the artifacts:** the specs are build inputs, not served content. Anything under `_datafiles/html/public/` is web-reachable; a JSON specification sitting next to its artifact would be silently downloadable.

---

## A note on the diagram-authoring tasks

Tasks 2, 5, 6, 8, 9 and 10 each author one diagram. **They deliberately do not contain the finished JSON.** This is not a placeholder omission — archify's authoring contract requires the opposite of pre-planning:

> "Artifact first: the next tool action must write the candidate. Write the candidate before inspecting renderer internals. **Do not plan exact coordinates in prose.**"

Node positions and sizes are laid out iteratively against validator diagnostics. What each task *does* specify completely: the codegraph queries that establish ground truth, the exact node roster the diagram must contain, the guided chapters, the commands to run, and the acceptance bar. That is everything the engineer needs; the coordinates are discovered, not designed.

**Every authoring task follows this same five-step loop.** It is repeated in full in each task rather than cross-referenced, because tasks may be read out of order.

---

## Task 1: Verify the archify toolchain and resolve the open spec item

**Files:**
- None (investigation task; findings recorded in the task's commit message)

- [ ] **Step 1: Confirm the CLI runs**

```bash
node "$HOME/.agents/skills/archify/bin/archify.mjs" doctor
```

Expected: a report confirming the Node version and that renderers/schemas are present, exit code 0. If this fails, stop — nothing else in this plan can proceed.

- [ ] **Step 2: Confirm the CLI works with the repo as CWD**

The skill's documentation shows commands run from inside the skill directory. This plan runs them from the DOGMud repo root, so confirm relative input paths resolve correctly:

```bash
cd "$HOME/workspace/DOGMud"
mkdir -p tools/archify/specs
node "$HOME/.agents/skills/archify/bin/archify.mjs" demo /tmp/archify-smoke
ls -la /tmp/archify-smoke
```

Expected: demo artifacts written to `/tmp/archify-smoke`, exit code 0.

If it fails because the CLI resolves paths relative to its own directory, the fallback for every later command is to `cd "$HOME/.agents/skills/archify"` and pass **absolute** paths for both input and output. Record which form works — every subsequent task depends on it.

- [ ] **Step 3: Resolve the spec's open item — what does `--repo-root` actually do?**

The design spec flags this as explicitly unverified. Find out:

```bash
grep -n -A 20 -i "repository evidence\|repo-root" "$HOME/.agents/skills/archify/references/authoring-contract.md"
```

Answer these three questions:
1. Does `--repo-root` let a node cite a real file path that the validator checks against the filesystem?
2. If a cited path stops existing, does `validate` fail?
3. Is the citation rendered visibly in the artifact, or is it validation-only metadata?

- [ ] **Step 4: Decide and record**

If the answer to (1) and (2) is yes, this is a genuine anti-drift lever: a moved or deleted file would break the diagram's validation, turning a silent staleness problem into a loud one. **In that case, add `--repo-root .` to every `validate` and `deliver` command in this plan and cite real paths on nodes.**

If it is validation-only metadata or does not check the filesystem, do not use it — it adds authoring cost for no drift protection.

- [ ] **Step 5: Commit the finding**

```bash
cd "$HOME/workspace/DOGMud"
touch tools/archify/specs/.gitkeep
git add tools/archify/specs/.gitkeep
git commit -m "chore(archify): scaffold spec dir; record toolchain + repo-evidence findings

doctor: <pass/fail>
CWD form that works: <repo root | skill dir + absolute paths>
--repo-root: <what it actually does, and the decision on using it>"
```

---

## Task 2: Diagram 1 — Engine Overview (`architecture`)

The anchor diagram. Every other diagram zooms into one box on this one.

**Files:**
- Create: `tools/archify/specs/engine-overview.architecture.json`
- Create (generated): `_datafiles/html/public/architecture/engine-overview.html`

- [ ] **Step 1: Establish ground truth with codegraph**

Run these and read the results before writing anything:

```
codegraph_context "server boot, main world tick loop, and connection handling"
codegraph_node "Listen" includeCode:true
codegraph_files "internal/connections"
codegraph_files "modules"
```

Confirm before authoring: the real name of the main loop function, how telnet and websocket connections converge, and which module packages actually exist under `modules/`.

- [ ] **Step 2: Write the candidate specification**

Create `tools/archify/specs/engine-overview.architecture.json`. Required top-level keys are `schema_version` (1), `diagram_type` (`"architecture"`), `meta`, `components`. Optional: `boundaries`, `connections`, `cards`, `layout`.

`meta` must include:
```json
"quality_profile": "showcase",
"title": "DOGMud Engine Overview",
"subtitle": "A Go MUD engine with a YAML-authored world",
"output": "engine-overview.html"
```

**Node roster — at most 12 primary components.** Use `type` values from the enum `frontend | backend | database | cloud | security | messagebus | external`:

| Node | Suggested type | Notes |
|---|---|---|
| Players (telnet) | `external` | |
| Players (browser) | `external` | |
| Telnet listener | `frontend` | real port numbers from config |
| HTTP / WebSocket server | `frontend` | `internal/web` |
| Input handlers | `backend` | `internal/inputhandlers` |
| World tick loop | `backend` | the emphasis node — `variant: "emphasis"` |
| Rooms | `backend` | |
| Mobs | `backend` | |
| Users / Characters | `backend` | |
| YAML world data | `database` | `_datafiles/world/dogmud/` |
| Instance saves | `database` | the overlay layer — diagram 4 zooms in here |
| Modules | `messagebus` | GMCP, web, playtest, achievements |

**Guided chapters — `meta.views`, at most 5.** Each is `{id, label, focus: [node ids], note}` where `label` ≤ 48 chars and `note` ≤ 140 chars:

1. `connection-path` — "How a player connects" — both player nodes, both listeners, input handlers
2. `the-tick` — "The world tick" — input handlers, tick loop, rooms, mobs
3. `world-state` — "Live world state" — rooms, mobs, users
4. `data-layer` — "Where the world is authored" — YAML data, instance saves, rooms, mobs
5. `modules` — "Pluggable modules" — tick loop, modules, websocket server

Keep one obvious main path (players → listener → input handlers → tick loop → world state → data). Start with automatic routes. **Do not add `via`, `channelX`, `channelY`, or `labelAt` until a validator diagnostic asks for one**, and then add only the single control it diagnosed.

- [ ] **Step 3: Validate, repairing until clean**

```bash
cd "$HOME/workspace/DOGMud"
node "$HOME/.agents/skills/archify/bin/archify.mjs" validate architecture \
  tools/archify/specs/engine-overview.architecture.json --quality showcase --json
```

Expected on success: a receipt reporting **all 9 artifact checks, 0 composition errors, 0 warnings.**

A receipt showing only 4 checks means `meta.quality_profile` is missing or misspelled — fix that before touching geometry. On failure, change only the diagnosed `subject`, verify the `evidence`, and pick from `supportedFixes`. Continue while the objective error count reaches a new minimum; **if two consecutive rounds do not improve the best count, stop and report the unresolved diagnostics truthfully rather than forcing it.**

- [ ] **Step 4: Deliver**

```bash
node "$HOME/.agents/skills/archify/bin/archify.mjs" deliver architecture \
  tools/archify/specs/engine-overview.architecture.json \
  _datafiles/html/public/architecture/engine-overview.html \
  --quality showcase --json
```

Expected: exit code 0 and a receipt with SHA-256 plus byte counts for both the specification and the artifact. **A non-zero exit is never success.** After this succeeds the artifact is frozen — never hand-edit it. To change anything, edit the JSON and re-deliver.

- [ ] **Step 5: Run the hosting guard manually**

```bash
grep -c '{{' _datafiles/html/public/architecture/engine-overview.html
```

Expected: `0`. Any other number means the artifact would 500 when served — see Task 4.

- [ ] **Step 6: Commit**

```bash
git add tools/archify/specs/engine-overview.architecture.json \
        _datafiles/html/public/architecture/engine-overview.html
git commit -m "feat(diagrams): engine overview architecture diagram

Showcase validation clean (9/9 checks, 0 errors, 0 warnings).
Spec committed alongside so refreshes are an edit + re-deliver."
```

---

## Task 3: The index page

**Files:**
- Create: `_datafiles/html/public/architecture.html`

- [ ] **Step 1: Create the page**

Create `_datafiles/html/public/architecture.html` with exactly this content. Note it follows `online.html`'s shape: header template, content inside `.overlay`, footer template. The `<style>` block sits in the body — the same practical pattern the site already uses, and browsers apply it fine.

```html
{{template "header" .}}

<style>
  .diagrams-title {
    color: var(--title-gold);
    font-family: var(--font-serif);
    margin-bottom: 0.25rem;
  }
  .diagrams-intro {
    color: var(--text-tagline);
    max-width: 62ch;
    line-height: 1.55;
    margin: 0 0 1.75rem;
  }
  .diagram-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 1.1rem;
  }
  .diagram-card {
    display: flex;
    flex-direction: column;
    gap: 0.45rem;
    padding: 1.1rem 1.15rem 1rem;
    background: var(--panel-bg-alt);
    border: 1px solid var(--panel-border);
    border-radius: 4px;
    text-decoration: none;
    color: var(--text-primary);
    transition: border-color 0.18s ease, transform 0.18s ease;
  }
  .diagram-card:hover,
  .diagram-card:focus-visible {
    border-color: var(--gold-rule);
    transform: translateY(-2px);
  }
  .diagram-kind {
    font-family: var(--font-mono);
    font-size: 0.68rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: var(--antique-gold);
  }
  .diagram-card h3 {
    margin: 0;
    color: var(--title-gold);
    font-family: var(--font-serif);
    font-size: 1.15rem;
  }
  .diagram-what {
    margin: 0;
    line-height: 1.5;
  }
  .diagram-why {
    margin: 0;
    color: var(--text-secondary);
    font-style: italic;
    line-height: 1.5;
  }
  .diagram-cta {
    margin-top: auto;
    padding-top: 0.5rem;
    color: var(--ink-gold);
    font-size: 0.9rem;
  }
  @media (prefers-reduced-motion: reduce) {
    .diagram-card {
      transition: none;
    }
    .diagram-card:hover,
    .diagram-card:focus-visible {
      transform: none;
    }
  }
</style>

<div class="overlay">
  <h1 class="diagrams-title">Under the Hood</h1>
  <p class="diagrams-intro">
    {{ .CONFIG.Server.MudName }} runs on a Go MUD engine with a world authored
    entirely in YAML &mdash; rooms, mobs, quests, dialogue, schedules and
    patrols are all data, not code. These diagrams are hand-drawn against the
    running source and checked for layout, not generated from it. Each one
    opens full screen with search, click-to-focus, and a guided tour.
  </p>

  <div class="diagram-grid">

    <a class="diagram-card" href="/architecture/engine-overview.html" target="_blank" rel="noopener">
      <span class="diagram-kind">Architecture</span>
      <h3>Engine Overview</h3>
      <p class="diagram-what">
        Every player connection, telnet or browser, funnels into one world tick
        loop that drives rooms, mobs and characters over a YAML data layer.
      </p>
      <p class="diagram-why">
        Start here &mdash; every other diagram is a close-up of one box on this one.
      </p>
      <span class="diagram-cta">View diagram &rarr;</span>
    </a>

  </div>
</div>

{{template "footer" .}}
```

- [ ] **Step 2: Verify it renders**

The local server is run by the user, not by you. **Never start, restart or kill it.** If a server is already running, fetch the page:

```bash
curl -s -o /dev/null -w "%{http_code}\n" http://localhost/architecture
```

Expected: `200`. If no server is running, skip this step — Task 7 covers browser verification.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/html/public/architecture.html
git commit -m "feat(web): Under the Hood diagram index page

Card grid over the existing warm-dark palette; scoped style block so
gomud.css stays untouched. Cards open the artifact in a new tab because
delivered archify HTML is frozen and cannot carry a back link."
```

---

## Task 4: The guard test

**On ordering:** the usual red-then-green cycle is inverted here, deliberately. What this test asserts is a property of *data that already exists* (the artifacts from Task 2, the page from Task 3), so writing it earlier would only prove that missing files are missing. The red proof is Step 3, which injects the real defect and confirms the test catches it. A guard that has never been seen to fail is not a guard.

**Files:**
- Create: `internal/web/architecture_test.go`

- [ ] **Step 1: Write the guard test**

Create `internal/web/architecture_test.go`:

```go
package web

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Go test binaries run with CWD set to their own package directory, so any
// test that touches repo files has to climb back out. Same pattern as
// auth_test.go.
func diagramsRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

const (
	diagramsDirRel   = "_datafiles/html/public/architecture"
	diagramsIndexRel = "_datafiles/html/public/architecture.html"
)

// TestDiagramArtifactsHaveNoTemplateDelimiters guards the hosting seam.
//
// serveTemplate parses EVERY .html file it serves as a text/template, including
// the generated archify artifacts. Those artifacts survive that round trip only
// because they contain no "{{" for the parser to latch onto. If a future
// archify version, or an authored node label, ever introduces one, the file
// stops being a 625 KB diagram and becomes an HTTP 500. Catch it here rather
// than on the droplet.
func TestDiagramArtifactsHaveNoTemplateDelimiters(t *testing.T) {
	dir := filepath.Join(diagramsRepoRoot(t), filepath.FromSlash(diagramsDirRel))

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%q): %v", dir, err)
	}

	checked := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
			continue
		}
		checked++

		body, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatalf("ReadFile(%q): %v", entry.Name(), err)
		}

		if idx := strings.Index(string(body), "{{"); idx >= 0 {
			t.Errorf("%s contains a Go template delimiter at byte %d: serveTemplate "+
				"would fail to parse it and return HTTP 500. Re-author the offending "+
				"label in the spec and re-deliver the artifact.", entry.Name(), idx)
		}
	}

	if checked == 0 {
		t.Fatalf("no .html artifacts found under %s", diagramsDirRel)
	}
}

var diagramHrefPattern = regexp.MustCompile(`href="(/architecture/[^"]+\.html)"`)

// TestDiagramIndexLinksResolve fails on a card pointing at a file that is not
// there, which would otherwise surface as a 404 only when a visitor clicked it.
func TestDiagramIndexLinksResolve(t *testing.T) {
	root := diagramsRepoRoot(t)

	body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(diagramsIndexRel)))
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", diagramsIndexRel, err)
	}

	matches := diagramHrefPattern.FindAllStringSubmatch(string(body), -1)
	if len(matches) == 0 {
		t.Fatalf("%s links no diagrams", diagramsIndexRel)
	}

	for _, match := range matches {
		target := filepath.Join(root, "_datafiles", "html", "public",
			filepath.FromSlash(strings.TrimPrefix(match[1], "/")))
		if _, err := os.Stat(target); err != nil {
			t.Errorf("index card links %s but %s does not exist: %v", match[1], target, err)
		}
	}
}
```

- [ ] **Step 2: Run the tests and confirm they pass**

Both artifacts and the index page already exist from Tasks 2 and 3, so these should pass immediately:

```bash
cd "$HOME/workspace/DOGMud"
go test ./internal/web/ -run 'TestDiagram' -v
```

Expected: `PASS` for both tests, with `TestDiagramArtifactsHaveNoTemplateDelimiters` having checked 1 file.

- [ ] **Step 3: Prove the guard actually guards**

A test that cannot fail is not a guard. Verify it catches the real defect:

```bash
printf '{{' >> _datafiles/html/public/architecture/engine-overview.html
go test ./internal/web/ -run 'TestDiagramArtifactsHaveNoTemplateDelimiters' -v
```

Expected: **FAIL**, naming `engine-overview.html` and a byte offset.

Now restore the artifact exactly — do not hand-edit it back:

```bash
git checkout -- _datafiles/html/public/architecture/engine-overview.html
go test ./internal/web/ -run 'TestDiagram' -v
```

Expected: `PASS`.

- [ ] **Step 4: Format and commit**

```bash
gofmt -l internal/web/architecture_test.go
```

Expected: **no output.** (The 2026-08-03 push failed CI on the gofmt gate; the local flow does not check it automatically.) If the file is listed, run `gofmt -w internal/web/architecture_test.go`.

```bash
git add internal/web/architecture_test.go
git commit -m "test(web): guard the diagram hosting seam

Artifacts serve byte-identical only because they contain no {{ for
text/template to parse. Assert that, and assert every index card links
a file that exists. Failure mode verified by injecting a delimiter."
```

---

## Task 5: Diagram 2 — Mob Aliveness Stack (`architecture`)

The most distinctive system in the codebase, and the reason a peer audience stays on the page.

**Files:**
- Create: `tools/archify/specs/mob-aliveness.architecture.json`
- Create (generated): `_datafiles/html/public/architecture/mob-aliveness.html`
- Modify: `_datafiles/html/public/architecture.html`

- [ ] **Step 1: Establish ground truth with codegraph**

```
codegraph_context "mob idle tick: schedules, patrols, conversations, behavior trees"
codegraph_node "TickMobCraft" includeCode:true
codegraph_files "internal/conversations"
codegraph_search "patrol"
```

Confirm before authoring: the real arbitration order on an idle tick, the actual executor entry points, and the `Engine` field names (CLAUDE.md warns these are `trees`/`noTree`, **not** `mobTrees`/`noMobTree` — a stale plan once shipped code against the wrong names).

- [ ] **Step 2: Write the candidate specification**

Create `tools/archify/specs/mob-aliveness.architecture.json` with `schema_version: 1`, `diagram_type: "architecture"`, and `meta.quality_profile: "showcase"`, `meta.title: "Mob Aliveness Stack"`, `meta.output: "mob-aliveness.html"`.

**Node roster — at most 12:**

| Node | Suggested type |
|---|---|
| Idle mob tick | `backend` (`variant: "emphasis"`) |
| Combat / interrupt check | `security` |
| Schedule executor | `backend` |
| Patrol executor | `backend` |
| Conversation engine | `backend` |
| Behavior tree engine | `backend` |
| Idle command pool | `backend` |
| `pathto` movement | `backend` |
| Schedule YAML | `database` |
| Patrol YAML | `database` |
| Conversation library | `database` |
| Relationship edges | `database` |

**Guided chapters — `meta.views`, at most 5:**

1. `idle-gate` — "What runs on an idle tick" — idle tick, combat check, the four executors
2. `scheduled-day` — "A townsperson's day" — schedule executor, schedule YAML, pathto, idle pool
3. `patrol-loop` — "Patrol routes" — patrol executor, patrol YAML, pathto
4. `npc-talk` — "NPCs talking to each other" — conversation engine, conversation library, relationship edges
5. `interrupts` — "What breaks the routine" — combat check, idle tick, patrol executor

Be accurate about gating: conversations fire only when **both** NPCs are fully idle, and combat interrupts patrols with resumption to the same waypoint. Do not draw an edge that does not exist in the code.

- [ ] **Step 3: Validate, repairing until clean**

```bash
cd "$HOME/workspace/DOGMud"
node "$HOME/.agents/skills/archify/bin/archify.mjs" validate architecture \
  tools/archify/specs/mob-aliveness.architecture.json --quality showcase --json
```

Expected: all 9 checks, 0 errors, 0 warnings. Change only the diagnosed subject per round; stop and report if two consecutive rounds do not improve the best error count.

- [ ] **Step 4: Deliver**

```bash
node "$HOME/.agents/skills/archify/bin/archify.mjs" deliver architecture \
  tools/archify/specs/mob-aliveness.architecture.json \
  _datafiles/html/public/architecture/mob-aliveness.html \
  --quality showcase --json
```

Expected: exit 0 with a SHA-256 receipt. Artifact is frozen after this.

- [ ] **Step 5: Add the card to the index**

In `_datafiles/html/public/architecture.html`, insert this immediately after the Engine Overview `</a>` and before `</div>`:

```html
    <a class="diagram-card" href="/architecture/mob-aliveness.html" target="_blank" rel="noopener">
      <span class="diagram-kind">Architecture</span>
      <h3>Mob Aliveness Stack</h3>
      <p class="diagram-what">
        Daily schedules, patrol routes, NPC-to-NPC conversations, behaviour
        trees and idle chatter &mdash; and how they arbitrate for one mob on
        one tick.
      </p>
      <p class="diagram-why">
        Most MUD NPCs stand still until spoken to. These ones keep a calendar.
      </p>
      <span class="diagram-cta">View diagram &rarr;</span>
    </a>
```

- [ ] **Step 6: Run the guard tests**

```bash
grep -c '{{' _datafiles/html/public/architecture/mob-aliveness.html
go test ./internal/web/ -run 'TestDiagram' -v
```

Expected: `0` from grep, `PASS` from both tests (now checking 2 artifacts and 2 links).

- [ ] **Step 7: Commit**

```bash
git add tools/archify/specs/mob-aliveness.architecture.json \
        _datafiles/html/public/architecture/mob-aliveness.html \
        _datafiles/html/public/architecture.html
git commit -m "feat(diagrams): mob aliveness stack architecture diagram

Showcase validation clean (9/9 checks, 0 errors, 0 warnings)."
```

---

## Task 6: Diagram 3 — Combat Round Resolution (`sequence`)

**Files:**
- Create: `tools/archify/specs/combat-round.sequence.json`
- Create (generated): `_datafiles/html/public/architecture/combat-round.html`
- Modify: `_datafiles/html/public/architecture.html`

- [ ] **Step 1: Establish ground truth with codegraph**

```
codegraph_context "combat round resolution: hit roll, defense, damage pipeline"
codegraph_node "handleCombatRound" includeCode:true
codegraph_node "CalcRawDamage" includeCode:true
codegraph_node "ApplyMitigation" includeCode:true
```

Confirm the real call order and the exact function names. CLAUDE.md documents the intended design; verify the code matches before drawing it.

- [ ] **Step 2: Read the sequence schema and example**

This is the first `sequence` diagram in the plan, so read the field shape first — participants and messages differ from architecture's components and connections:

```bash
cat "$HOME/.agents/skills/archify/schemas/sequence.schema.json"
cat "$HOME/.agents/skills/archify/examples/cache-miss-request.sequence.json"
```

Use the example for field shape only, never for facts.

- [ ] **Step 3: Write the candidate specification**

Create `tools/archify/specs/combat-round.sequence.json` with `diagram_type: "sequence"`, `meta.quality_profile: "showcase"`, `meta.title: "Combat Round Resolution"`, `meta.output: "combat-round.html"`.

**Participants (at most 8 for a readable sequence):** Attacker, Combat round handler, Resource/encumbrance check, Dice (`dice.OpposedRollStat`), Defender, Defense resolution (best-of-all), Damage pipeline, Room broadcast.

**The message chain must show, in order:**
1. Stamina and encumbrance reduce the number of swings this round
2. Opposed stat roll for to-hit — with the note that **armour never enters this roll**
3. Defender rolls dodge, parry and block **all** and keeps whichever won by the widest margin, with a 15% floor (`MinDefenseChance`)
4. Raw damage = `stat × SkillMultiplier(rank) × itemMult × ChannelScale`
5. `ApplyMitigation` against the matching channel's percentage, capped at 75%
6. `dice.RollStat` adds variance
7. Output as a *description* (`GetDamageDescription`), never a number

**Guided chapters — `meta.views`, at most 5:**

1. `swing-budget` — "How many swings you get"
2. `to-hit` — "The attack roll"
3. `defense` — "Best-of-all defence"
4. `damage` — "Three-channel damage"
5. `message` — "Why you never see a number"

Point 7 and chapter 5 matter: never showing raw numbers to players is a deliberate design rule, and it is exactly the sort of decision this audience notices.

- [ ] **Step 4: Validate, repairing until clean**

```bash
cd "$HOME/workspace/DOGMud"
node "$HOME/.agents/skills/archify/bin/archify.mjs" validate sequence \
  tools/archify/specs/combat-round.sequence.json --quality showcase --json
```

Expected: all 9 checks, 0 errors, 0 warnings.

- [ ] **Step 5: Deliver**

```bash
node "$HOME/.agents/skills/archify/bin/archify.mjs" deliver sequence \
  tools/archify/specs/combat-round.sequence.json \
  _datafiles/html/public/architecture/combat-round.html \
  --quality showcase --json
```

Expected: exit 0 with a SHA-256 receipt.

- [ ] **Step 6: Add the card to the index**

Insert before the closing `</div>` of `.diagram-grid` in `_datafiles/html/public/architecture.html`:

```html
    <a class="diagram-card" href="/architecture/combat-round.html" target="_blank" rel="noopener">
      <span class="diagram-kind">Sequence</span>
      <h3>Combat Round Resolution</h3>
      <p class="diagram-what">
        One swing end to end: how many attacks your stamina buys, the opposed
        roll to land one, a defence picked from dodge, parry and block, and
        damage through one of three channels.
      </p>
      <p class="diagram-why">
        Armour never touches the to-hit roll &mdash; it only ever reduces damage.
      </p>
      <span class="diagram-cta">View diagram &rarr;</span>
    </a>
```

- [ ] **Step 7: Run the guard tests and commit**

```bash
grep -c '{{' _datafiles/html/public/architecture/combat-round.html
go test ./internal/web/ -run 'TestDiagram' -v
git add tools/archify/specs/combat-round.sequence.json \
        _datafiles/html/public/architecture/combat-round.html \
        _datafiles/html/public/architecture.html
git commit -m "feat(diagrams): combat round resolution sequence diagram

Showcase validation clean (9/9 checks, 0 errors, 0 warnings)."
```

---

## Task 7: Wire the nav tab and hand Phase A to the user

**Files:**
- Modify: `internal/web/web.go:147-151`

- [ ] **Step 1: Add the nav entry**

In `internal/web/web.go`, change the core nav slice from:

```go
		"NAV": []WebNav{
			{`Home`, `/`},
			{`Who's Online`, `/online`},
			{`Web Client`, `/webclient`},
		},
```

to:

```go
		"NAV": []WebNav{
			{`Home`, `/`},
			{`Who's Online`, `/online`},
			{`Web Client`, `/webclient`},
			{`Architecture`, `/architecture`},
		},
```

Plugin-supplied nav items sort after the core slice via `coreCount` a few lines below, so Achievements / Leaderboards / Help keep their existing order. Do not touch `navOrder`.

- [ ] **Step 2: Build and format**

```bash
cd "$HOME/workspace/DOGMud"
go build ./...
gofmt -l internal/web/web.go
```

Expected: build succeeds; `gofmt -l` prints nothing.

- [ ] **Step 3: Run the full web package test suite**

```bash
go test ./internal/web/ -v
```

Expected: PASS. Run the whole package, not just `TestDiagram` — the nav change touches shared template data.

- [ ] **Step 4: Commit**

```bash
git add internal/web/web.go
git commit -m "feat(web): Architecture tab in the site nav

Appended to the core slice, so plugin nav items still sort after it."
```

- [ ] **Step 5: Hand off to the user for browser verification**

Do **not** start the server — the user runs it. Ask the user to restart their local server and check:

1. The **Architecture** tab appears in the nav, after Web Client and before Achievements.
2. `/architecture` renders with the site header, footer and the three cards.
3. Each card opens its diagram full screen in a new tab.
4. In each diagram: zooming past 175% reveals more detail; clicking a node opens the focus panel; the chapter rail steps through the guided tour.
5. The copy on each card is accurate and worth reading.

**Stop here. Phase B does not start until the user has reviewed Phase A.** Their read on whether the diagrams are legible and the copy lands determines whether diagrams 4–6 follow the same template or need a different approach — authoring three more before finding that out wastes the work.

---

## Task 8: Diagram 4 — Template → Instance → Runtime (`sequence`)

**Files:**
- Create: `tools/archify/specs/data-load.sequence.json`
- Create (generated): `_datafiles/html/public/architecture/data-load.html`
- Modify: `_datafiles/html/public/architecture.html`

- [ ] **Step 1: Establish ground truth with codegraph**

```
codegraph_context "room and mob loading: YAML templates, instance saves, skip-tagged fields"
codegraph_node "restoreSkipTaggedFields" includeCode:true
codegraph_node "SaveRoomInstance" includeCode:true
```

Confirm which fields carry `instance:"skip"` and the exact order of template load, instance overlay, and restore. CLAUDE.md notes room spawn lists were **wrongly** documented as shadowed until 2026-07-25 — check the struct tags, do not trust prose.

- [ ] **Step 2: Write the candidate specification**

Create `tools/archify/specs/data-load.sequence.json` with `diagram_type: "sequence"`, `meta.quality_profile: "showcase"`, `meta.title: "Template, Instance, Runtime"`, `meta.output: "data-load.html"`.

**Participants (at most 8):** Boot loader, YAML template file, Room/Mob struct, Instance save file, `restoreSkipTaggedFields`, Live world object, `SaveRoomInstance`.

**The message chain must show:**
1. Template YAML loads into the struct
2. If an instance save exists, it overlays the struct — silently shadowing template edits
3. Fields tagged `instance:"skip"` are copied **back** from the template, so a stale save cannot override them
4. The object goes live
5. On save, `SaveRoomInstance` omits the skip-tagged fields entirely

**Guided chapters — `meta.views`, at most 5:**

1. `template-load` — "Authored YAML loads first"
2. `instance-overlay` — "Saved state overlays it"
3. `skip-restore` — "What a save cannot override"
4. `runtime` — "The live object"
5. `write-back` — "What gets written back"

The story here is the round trip: an overlay that is deliberately *incomplete*, with a restore pass that puts authored intent back on top. That asymmetry is the whole point of the diagram.

- [ ] **Step 3: Validate, repairing until clean**

```bash
cd "$HOME/workspace/DOGMud"
node "$HOME/.agents/skills/archify/bin/archify.mjs" validate sequence \
  tools/archify/specs/data-load.sequence.json --quality showcase --json
```

Expected: all 9 checks, 0 errors, 0 warnings.

- [ ] **Step 4: Deliver**

```bash
node "$HOME/.agents/skills/archify/bin/archify.mjs" deliver sequence \
  tools/archify/specs/data-load.sequence.json \
  _datafiles/html/public/architecture/data-load.html \
  --quality showcase --json
```

- [ ] **Step 5: Add the card to the index**

Insert before the closing `</div>` of `.diagram-grid`:

```html
    <a class="diagram-card" href="/architecture/data-load.html" target="_blank" rel="noopener">
      <span class="diagram-kind">Sequence</span>
      <h3>Template, Instance, Runtime</h3>
      <p class="diagram-what">
        Authored YAML loads first, a saved instance overlays it, and then a
        restore pass copies certain fields back from the template so a stale
        save can never override them.
      </p>
      <p class="diagram-why">
        A deliberately incomplete overlay &mdash; authored intent wins on the
        fields that matter.
      </p>
      <span class="diagram-cta">View diagram &rarr;</span>
    </a>
```

- [ ] **Step 6: Run the guard tests and commit**

```bash
grep -c '{{' _datafiles/html/public/architecture/data-load.html
go test ./internal/web/ -run 'TestDiagram' -v
git add tools/archify/specs/data-load.sequence.json \
        _datafiles/html/public/architecture/data-load.html \
        _datafiles/html/public/architecture.html
git commit -m "feat(diagrams): template/instance/runtime sequence diagram

Showcase validation clean (9/9 checks, 0 errors, 0 warnings)."
```

---

## Task 9: Diagram 5 — GMCP to Web Client (`dataflow`)

**Files:**
- Create: `tools/archify/specs/gmcp-webclient.dataflow.json`
- Create (generated): `_datafiles/html/public/architecture/gmcp-webclient.html`
- Modify: `_datafiles/html/public/architecture.html`

- [ ] **Step 1: Establish ground truth with codegraph**

```
codegraph_context "GMCP zone map snapshot and web client rendering"
codegraph_node "Snapshot" includeCode:true
codegraph_node "MarkRoomVisited" includeCode:true
codegraph_search "SnapshotExit"
```

Also read the client renderer directly — it is JavaScript, so codegraph does not index it:

```bash
grep -n "RoomGridSVG" -A 30 _datafiles/html/public/static/js/gmcp.js | head -60
```

- [ ] **Step 2: Read the dataflow schema and example**

```bash
cat "$HOME/.agents/skills/archify/schemas/dataflow.schema.json"
cat "$HOME/.agents/skills/archify/examples/product-analytics.dataflow.json"
```

- [ ] **Step 3: Write the candidate specification**

Create `tools/archify/specs/gmcp-webclient.dataflow.json` with `diagram_type: "dataflow"`, `meta.quality_profile: "showcase"`, `meta.title: "GMCP to Web Client"`, `meta.output: "gmcp-webclient.html"`.

**The pipeline must show, at most 12 stages:** player move in `go.go` → `MarkRoomVisited` writes `Character.VisitedRooms` → `mapper.Snapshot(visited)` filters unvisited rooms **and their exits** → `Zone.Map` GMCP payload (rooms, exits with kind and flags, party room IDs) → websocket → `gmcp.js` handler → `RoomGridSVG` → the leather-styled SVG surface, with per-exit-type connection styling and party figures.

**Guided chapters — `meta.views`, at most 5:**

1. `move-triggers` — "Every move sends a map"
2. `fog-of-war` — "Fog is enforced on the server"
3. `transport` — "The GMCP payload"
4. `render` — "Drawing the map"
5. `overlays` — "Party markers and exit types"

Chapter 2 carries the interesting claim and must be accurate: unvisited rooms are filtered out **server-side** before the payload is built, so the client is never sent map data the player has not earned. Verify that in `Snapshot` before asserting it.

- [ ] **Step 4: Validate, repairing until clean**

```bash
cd "$HOME/workspace/DOGMud"
node "$HOME/.agents/skills/archify/bin/archify.mjs" validate dataflow \
  tools/archify/specs/gmcp-webclient.dataflow.json --quality showcase --json
```

Expected: all 9 checks, 0 errors, 0 warnings.

- [ ] **Step 5: Deliver**

```bash
node "$HOME/.agents/skills/archify/bin/archify.mjs" deliver dataflow \
  tools/archify/specs/gmcp-webclient.dataflow.json \
  _datafiles/html/public/architecture/gmcp-webclient.html \
  --quality showcase --json
```

- [ ] **Step 6: Add the card to the index**

Insert before the closing `</div>` of `.diagram-grid`:

```html
    <a class="diagram-card" href="/architecture/gmcp-webclient.html" target="_blank" rel="noopener">
      <span class="diagram-kind">Data Flow</span>
      <h3>GMCP to Web Client</h3>
      <p class="diagram-what">
        Every step you take rebuilds a map snapshot, ships it over GMCP, and
        redraws the browser client's tooled-leather map with exits, party
        markers and fog.
      </p>
      <p class="diagram-why">
        Fog of war is filtered on the server &mdash; the client is never sent
        rooms you have not walked.
      </p>
      <span class="diagram-cta">View diagram &rarr;</span>
    </a>
```

- [ ] **Step 7: Run the guard tests and commit**

```bash
grep -c '{{' _datafiles/html/public/architecture/gmcp-webclient.html
go test ./internal/web/ -run 'TestDiagram' -v
git add tools/archify/specs/gmcp-webclient.dataflow.json \
        _datafiles/html/public/architecture/gmcp-webclient.html \
        _datafiles/html/public/architecture.html
git commit -m "feat(diagrams): GMCP to web client dataflow diagram

Showcase validation clean (9/9 checks, 0 errors, 0 warnings)."
```

---

## Task 10: Diagram 6 — Use-Based Progression Loop (`lifecycle`)

**Files:**
- Create: `tools/archify/specs/progression-loop.lifecycle.json`
- Create (generated): `_datafiles/html/public/architecture/progression-loop.html`
- Modify: `_datafiles/html/public/architecture.html`

- [ ] **Step 1: Establish ground truth with codegraph**

```
codegraph_context "use-based stat and skill progression"
codegraph_node "CheckStatProgression" includeCode:true
codegraph_node "CheckSkillProgression" includeCode:true
```

Confirm: the probability curve, the role of `StatProgressionSoftCap` (a *virtual rank* where progression slows, plus an anti-exploit floor — **not** a ceiling on stat values), and the roughly-25-uses cadence on skills.

- [ ] **Step 2: Read the lifecycle schema and example**

```bash
cat "$HOME/.agents/skills/archify/schemas/lifecycle.schema.json"
cat "$HOME/.agents/skills/archify/examples/agent-run.lifecycle.json"
```

Note the layout rule from the skill: phase columns `0..4` occupy the main rail, and event/outcome columns `0..2` align beneath later phases.

- [ ] **Step 3: Write the candidate specification**

Create `tools/archify/specs/progression-loop.lifecycle.json` with `diagram_type: "lifecycle"`, `meta.quality_profile: "showcase"`, `meta.title: "Use-Based Progression"`, `meta.output: "progression-loop.html"`.

**States and transitions must show:**
- A stat is *used* in play → `OnStatUse` fires
- `CheckStatProgression` rolls probabilistically — most uses change nothing, and that is the normal path, not a failure
- On success the stat advances; the loop returns to "in play"
- Past `StatProgressionSoftCap` (default 150) the advance probability drops sharply — a brake, not a wall. **There is no cap on stat values.**
- The parallel skill loop: `OnSkillUse` → `CheckSkillProgression`, roughly every 25 uses, soft cap 50
- No XP node and no level node anywhere — their absence is the point

**Guided chapters — `meta.views`, at most 5:**

1. `use-it` — "Growth starts with use"
2. `the-roll` — "A roll, not a counter"
3. `advance` — "When a stat moves"
4. `the-brake` — "What slows runaway stats"
5. `skills` — "Skills work the same way"

- [ ] **Step 4: Validate, repairing until clean**

```bash
cd "$HOME/workspace/DOGMud"
node "$HOME/.agents/skills/archify/bin/archify.mjs" validate lifecycle \
  tools/archify/specs/progression-loop.lifecycle.json --quality showcase --json
```

Expected: all 9 checks, 0 errors, 0 warnings.

- [ ] **Step 5: Deliver**

```bash
node "$HOME/.agents/skills/archify/bin/archify.mjs" deliver lifecycle \
  tools/archify/specs/progression-loop.lifecycle.json \
  _datafiles/html/public/architecture/progression-loop.html \
  --quality showcase --json
```

- [ ] **Step 6: Add the card to the index**

Insert before the closing `</div>` of `.diagram-grid`:

```html
    <a class="diagram-card" href="/architecture/progression-loop.html" target="_blank" rel="noopener">
      <span class="diagram-kind">Lifecycle</span>
      <h3>Use-Based Progression</h3>
      <p class="diagram-what">
        No experience points and no levels. Using a stat rolls for a chance to
        improve it, and a soft cap bends the curve rather than capping the
        number.
      </p>
      <p class="diagram-why">
        The thing worth noticing is what is missing from this diagram.
      </p>
      <span class="diagram-cta">View diagram &rarr;</span>
    </a>
```

- [ ] **Step 7: Run the guard tests and commit**

```bash
grep -c '{{' _datafiles/html/public/architecture/progression-loop.html
go test ./internal/web/ -run 'TestDiagram' -v
git add tools/archify/specs/progression-loop.lifecycle.json \
        _datafiles/html/public/architecture/progression-loop.html \
        _datafiles/html/public/architecture.html
git commit -m "feat(diagrams): use-based progression lifecycle diagram

Showcase validation clean (9/9 checks, 0 errors, 0 warnings)."
```

---

## Task 11: Patch notes

**Files:**
- Modify: `docs/PATCH_NOTES.md`

- [ ] **Step 1: Read the existing format**

```bash
head -40 docs/PATCH_NOTES.md
```

Match the existing dated-entry style exactly rather than inventing one.

- [ ] **Step 2: Add the entry**

Add a `2026-08-04` entry describing the Under the Hood page: six interactive technical diagrams covering the engine overview, mob aliveness, combat resolution, the data-load round trip, GMCP-to-client dataflow, and use-based progression; reachable from the new Architecture tab.

Keep it player-readable — patch notes are public. Do not list file paths or function names.

- [ ] **Step 3: Commit**

```bash
git add docs/PATCH_NOTES.md
git commit -m "docs(patch-notes): Under the Hood diagrams page"
```

---

## Task 12: Final verification

- [ ] **Step 1: Full build, format and test**

```bash
cd "$HOME/workspace/DOGMud"
go build ./...
gofmt -l internal/ modules/
go test ./internal/web/ -v
```

Expected: build succeeds, `gofmt -l` prints **nothing**, all web tests PASS.

The gofmt check is not optional — the 2026-08-03 push failed CI on exactly this gate because the local flow never checks it.

- [ ] **Step 2: Confirm all six artifacts are present and clean**

```bash
ls -la _datafiles/html/public/architecture/
grep -c '{{' _datafiles/html/public/architecture/*.html
du -sh _datafiles/html/public/architecture/
```

Expected: six `.html` files, `0` from every grep, total size around 3.7 MB.

- [ ] **Step 3: Confirm the specs are committed**

```bash
ls -la tools/archify/specs/
git status --short
```

Expected: six `.json` specifications; a clean working tree. If any artifact is committed without its specification, refreshing that diagram later means re-authoring from scratch — fix it before finishing.

- [ ] **Step 4: Confirm nothing unwanted is staged for the droplet**

```bash
git log --stat --oneline "$(git merge-base master HEAD)"..HEAD | grep -E "instances|shops/|guilds/|moderation/" || echo "clean"
```

Expected: `clean`. Instance saves, shop state, guild files and moderation state must never be committed.

- [ ] **Step 5: Hand off to the user**

Ask the user to restart their local server and verify in a browser:

1. All six cards render and open their diagrams.
2. Each diagram's guided chapters step through sensibly.
3. Nothing on any diagram is factually wrong about the engine.
4. The card copy reads well to a developer who has never seen the project.

Point (3) is the one that matters most and the one automation cannot check. A validated diagram is a well-laid-out diagram, not a true one.

**Do not merge to master until the user has approved.** Merging to master is shipping.

---

## Notes for whoever executes this

- **Never start, restart or kill the local server.** The user runs it. The PID owning the ports is a live session.
- **Never hand-edit a delivered artifact.** Validation freezes it. Edit the JSON specification and re-deliver.
- **A non-zero exit from `deliver` is never success**, regardless of what got written to disk.
- **Stop rather than force a diagram through.** If two consecutive repair rounds do not improve the objective error count, report the unresolved diagnostics truthfully. A diagram that barely passes is worse than one honestly reported as blocked.
- **Leave `meta.animation` unset on every diagram.** Static is the default and is what this page wants; `"trace"` motion is for demos and presentations.
- **Diagram accuracy outranks diagram beauty.** This page is aimed at people who will read the source. A pretty diagram that misstates the architecture is a liability.
