# Adversarial Meta-Review: Agent Work After 2026-08-04

**Date:** 2026-08-08
**Reviewer:** Claude (Opus 5)
**Scope:** All 75 commits after `6b717c25b` (2026-08-04), the two agent-authored
adversarial reviews, and the remediation roadmap derived from them.
**Method:** Independent re-verification against source. Every claim graded below
was checked by reading the cited code, running the cited tooling, or inspecting
the Git tree. Claims I could not verify are marked as such.

---

## Executive summary

Two agents worked this window. The quality gap between them is large, and it is
not the gap you would guess from the commit counts.

**Cursor** produced the strongest review artifact this repo has received. I spot
checked 16 of its 33 findings against source and **all 16 held**. Its
`golangci-lint` claim reproduced to the digit (105 issues, split 50/31/19/2/3).
It invalidated its own Finding 1 with cryptographic evidence rather than
defending it. That is unusually honest work.

**Jules** found four real architectural problems Cursor missed entirely,
including what is arguably the worst availability defect in the codebase. It
also fabricated a code citation, shipped an invalid CI change, and authored two
commits whose messages describe work not present in their diffs, one of which is
empty.

The most serious problem is neither agent's analysis. It is what happened next:
**36 of 37 findings are still open, and the one closed finding was closed by
being disproven.** Sixty-one commits and roughly 17,200 lines of Go went into
playtest infrastructure that no finding required, while a live production
password sat exposed in a public repository the entire time.

---

## Attribution

Commit authorship is misleading at a glance. All non-Jules commits are committed
as `Calabe Davis`, but 61 of them carry `Co-authored-by: Cursor
<cursoragent@cursor.com>`.

**"Cursor" is not a model.** It is an orchestration harness that dispatches work
to several different underlying models. The Git trailer flattens all of them
into one name, so commit metadata alone cannot tell you which model wrote which
line. Grading "Cursor" as a single agent, as an earlier draft of this document
did, is a category error.

| Agent | Commits | Deliverables |
|---|---:|---|
| **Cursor harness** (multi-model) | 61 | `ADVERSARIAL_CODE_REVIEW_2026-08-07.md` (untracked), `docs/roadmaps/ADVERSARIAL_REVIEW_REMEDIATION_ROADMAP.md`, chunks 0.2 and 0.3a through 0.3d (the playtest harness, ~17.2k LOC), `docs/guides/TESTING_GUIDE.md` |
| **Jules** (`google-labs-jules[bot]`) | 3 | `docs/audits/ADVERSARIAL_REVIEW.md` (313 lines), one line in `.github/workflows/discord-notify.yml` (since reverted) |
| Human / merges | 11 | PR merges, the Jules revert, top-of-branch race fixes |

### Recoverable per-model attribution

The Git trailer records nothing, but the implementation plans under
`docs/superpowers/plans/2026-08-0*.md` name a model per task. That gives partial
attribution for the harness arc only:

| Model | Task types it was assigned |
|---|---|
| **Claude Sonnet 5** (incl. Thinking High) | Dominant executor: template authoring, goals/scenario migration, driver and docs, Docker integration, live smoke, adversarial implementation review |
| **Composer 2.5 Fast** | Mechanical work: CLI wiring, workflow wiring, Task 9 documentation |
| **GPT-5.6 Sol Medium** | Filesystem containment (Task 3 of the supervisor plan) |
| **Grok** | Named once as an inline-execution alternative for 0.3d Tasks 0 through 7 |

Three important limits on this:

1. It covers the **harness build only**. Authorship of the 33-finding review and
   of the roadmap, which are the two highest-value artifacts of the window, is
   recorded nowhere. I cannot attribute those to a model.
2. It is a record of *assignment*, not of execution. Nothing verifies the
   assigned model is the one that ran.
3. Consequently, the per-dimension grades below are grades on **the harness's
   output**, not on any individual model. Do not read "B+ on code quality" as a
   statement about Sonnet 5, or "D on prioritization" as a statement about
   Composer.

If per-model grading matters going forward, the fix is cheap: have the harness
write the executing model into a commit trailer.

---

## Verification results

I re-derived each of these from source rather than trusting the reviews.

### Cursor's findings: 16 of 16 confirmed

| # | Claim | Verdict | Evidence |
|---:|---|---|---|
| 3 | Harmful spells bypass `PlayerAttackImmune` | **Confirmed** | `attack.go:166` checks `IsNonCombatant() \|\| PlayerAttackImmune`; `cast.go:96-105` checks only `IsCharmed()` and `IsNonCombatant()` |
| 4 | Round counter unsynchronized | **Confirmed** | `util.go:108-127`, bare package-level `roundCount++` and read |
| 8 | Room path cache unsynchronized | **Confirmed** | `roommanager.go:82-92`, map read and map write, no mutex |
| 9 | Phantom position tests | **Confirmed** | 87 `t.Skip` calls, 40 references to `control_test.go`, which does not exist |
| 10 | Release CI omits lint/coverage | **Confirmed** | `run-tests.yml:14,25,37` has lint and a coverage gate; `build-and-release.yml:25-27` has neither |
| 11 | `wander loot`/`players` ignore their filter | **Confirmed** | `wander.go:44-71` builds `exitOptions`, then line 73 calls `room.GetRandomExit()`. `exitOptions` is never read |
| 12 | Compact gold-give parses two different amounts | **Confirmed** | `give.go:44` slices `[0:len-5]`; `give.go:343` slices `[:len-4]`. `50gold` resolves as 50 and transfers 5 |
| 17 | Admin economy stored-XSS surface | **Confirmed** | 12 `innerHTML` sites in `_datafiles/html/admin/economy/index.html` |
| 20 | HTTP servers lack timeouts | **Confirmed** | `web.go:461,493` construct `http.Server` with only `Addr` and `Handler` |
| 21 | LLM check-then-set race | **Confirmed** | `client.go:49` `isPending()`, `client.go:67` `setPending()`, separately locked |
| 22 | Listener lock held across callbacks | **Confirmed** | `listeners.go:197-198` `listenerLock.Lock(); defer Unlock()` spanning `invokeListenerSafely` |
| 26 | Makefile stale toolchain and world paths | **Confirmed** | `Makefile:24` pins `golang:1.21.3` against `go.mod` `go 1.25.0`; `clean-instances` targets `world/default` and `world/empty`, never `world/dogmud` |
| 28 | Two plausible nil dereferences | **Confirmed** | `combat_helpers.go:260` reads `raceInfo.UnarmedName`, line 275 then tests `raceInfo != nil`. `look.go:615` mirrors it |
| 32 | `WrapAnsi` panic fallback returns `""` | **Confirmed** | Unnamed `string` return plus bare `_ = recover()`, contradicting its own doc comment |
| 35 | Autosave failures reported as success | **Confirmed** | `save_and_load.go:404-425`, `errCt` incremented then `return nil` unconditionally |
| Baseline | "golangci-lint: 105 findings, 50 errcheck, 31 ineffassign, 19 staticcheck, 2 govet, 3 unconvert" | **Confirmed exactly** | Re-ran it. Identical totals and identical per-linter split |

One inaccuracy: Finding 9 says `position_test.go` "contains 100 test functions."
The real count is 124. The 87-skip figure and the `control_test.go` claim are
both correct, so the finding stands, but the number is wrong.

### Jules' findings: 6 of 7 confirmed, 1 fabricated citation

| Claim | Verdict | Evidence |
|---|---|---|
| Single global `mudLock` RWMutex | **Confirmed** | `util.go:68` |
| `RunWithMUDLocked` takes a global **write** lock around admin HTTP handlers | **Confirmed** | `web.go:551-559`, applied at 23 sites in that file |
| `GetAutoComplete` takes a read lock and is heavy | **Confirmed** | `world.go:319-322` |
| Autosave runs synchronously in the locked main loop | **Confirmed** | `NewTurn_AutoSave.go:41,53,65`; `world.go` wraps loop bodies in `util.LockMud()` |
| `SaveRoomInstance` uses reflection plus `DeepEqual` per field | **Confirmed** | `save_and_load.go:331-355` |
| 17 mutable global callback setters wired in `main.go` | **Confirmed** | `main.go:292,296,301,307,320,335,345,365,454,481,1629,1641,1659` and others |
| **`SkillMultiplier` hard-clamps rank at `SkillSoftCap`** | **Confirmed** | `damage_pipeline.go:37-39`. This directly contradicts `CLAUDE.md`'s "no soft cap on stat values" framing for the skill side |
| **`OpposedRollStat` derives both rolls' stddev from the attacker** | **Confirmed** | `dice.go:462-464`, `StdDevFor(atk)` |
| Test sample `internal/rooms/instances_test.go` / `func TestBTreeStateEviction` | **Fabricated identifier** | No such symbol exists anywhere in the repo. The real test is `TestCheckPortalTimers_TtlExpiryEvictsBtreeState` at `instances_test.go:381`. The quoted `origEvictor` / `defer` pattern is genuine |

---

## The Good

### A. Cursor's review is the best audit artifact in this repository. Grade: A

Thirty-three findings, every one I checked reproducible from the cited file and
line. The severity ordering is defensible. The "Positive observations" section
correctly identifies that the repo's problem is inconsistent application of good
patterns rather than absent patterns, which is the right diagnosis. The
verification baseline section states honestly that `go test ./...` was
**inconclusive** rather than green because Defender blocked a test binary,
instead of quietly claiming a pass.

### A. Cursor disproved its own finding. Grade: A

Finding 1 claimed the xterm runtime was missing from releases. Cursor
subsequently proved it wrong via `git ls-tree`, recorded the blob hash
(`fbe149c`), size, and SHA-256, explained the root cause of its own error
(indexed file search skipping a large minified asset), and marked the chunk
**Invalidated** in both the review and the roadmap. It then explicitly declined
to smuggle in an asset-manifest validator as fake remediation. Agents almost
never do this.

### A-. Cursor absorbed a competing review instead of ignoring it. Grade: A-

Jules' review landed independently. Rather than defend territory, Cursor folded
it in as Findings 34 through 37 plus two design-decision chunks (5.7 skill
soft cap, 5.8 opposed-roll variance), and cited the origin commit `9338a33`.
Chunks 5.7 and 5.8 are correctly scoped as decision-only with an explicit
instruction not to infer intent from existing tests, which is exactly right for
a balance question.

### B+. Jules found four real things Cursor missed. Grade: B+

This is the substantive value Jules added, and it should not be lost in the
process criticism below:

- `RunWithMUDLocked` is a genuine and serious defect. A single slow admin HTTP
  request holds the global world write lock through authentication, disk reads,
  template rendering, and the response write. Cursor's 33-finding sweep missed
  it entirely.
- The `SkillMultiplier` ceiling is a real doctrine conflict. `CLAUDE.md` teaches
  that progression is uncapped; combat damage stops scaling at rank 50.
- The attacker-owned variance in `OpposedRollStat` is real and its consequence
  is correctly reasoned: a defender's roll consistency is controlled by whoever
  attacks them.
- The autosave pause mechanism is structurally real.

### B+. The playtest harness is competently engineered. Grade: B+

Judged purely as code, ignoring whether it should have been built now:

- `go build ./...` clean. All five harness packages pass tests.
- The Docker isolation is genuinely careful: `DOCKER_HOST` is rejected outright
  (`docker.go:134`), and the endpoint is validated per platform, `npipe://` on
  Windows and `unix://` on Linux only.
- `ForbiddenIdentity` (`playtestprofiles/identity.go`) denylists production
  character names case-insensitively, with tests, which correctly honors the
  standing "never evaluate on a production character" rule.
- The last two commits on the branch fix a data race and a host/port join bug in
  Cursor's *own* new tests. Self-correction before handoff is a good sign.

### A-. The roadmap's process design is mature. Grade: A-

Several things here are better than most human-written roadmaps: a finding
coverage index separate from chunk status so findings cannot vanish behind a
"Done" chunk; an explicit invalidation protocol with evidence; the 3.6a/3.6b
split that forces autosave pauses to be **measured** before anyone is allowed to
re-architect persistence; refusal to pre-select solutions in boundary sections;
and XL chunks flagged as requiring decomposition rather than being attempted.

---

## The Bad

### D. Nothing was fixed. Grade: D

This is the headline problem. As of this review:

- Findings closed by fixing: **0 of 37**
- Findings closed by disproof: 1 (Finding 1)
- Findings still Open: **36**
- Commits spent: 61
- Go code added: ~17,200 lines

Every chunk marked Done in the tracker (0.2, 0.3a, 0.3b, 0.3c, 0.3d) is labeled
**"Supporting"** in the Findings column, meaning no review finding required it.
Findings 11, 12, and 32 are one-line fixes sitting in Phase 5 behind a
dependency on the test baseline. Finding 12 is a live bug that takes a player's
gold at 10x the intended rate on the compact syntax.

To be fair to Cursor: chunk 0.3 *was* in the roadmap's first committed version,
so this was not an unplanned detour. But it was scoped there as a single **M**
chunk. It shipped as four chunks including two **L**s. An enabling chunk that no
finding required underwent a roughly 4x scope expansion and consumed the entire
work window.

### C-. The committed roadmap points at a document that does not exist. Grade: C-

`docs/roadmaps/ADVERSARIAL_REVIEW_REMEDIATION_ROADMAP.md` is tracked and its
opening line cites `ADVERSARIAL_CODE_REVIEW_2026-08-07.md`. That file is
**untracked**, sits in the repository **root**, and is referenced by bare
filename with no path. Anyone cloning this repo gets a 37-finding roadmap whose
source document is a phantom. This also violates two standing project rules:
human-readable docs belong in `docs/`, and new files should be stated by full
path.

### C-. Neither agent indexed its new docs. Grade: C-

`docs/README.md` contains zero references to `docs/guides/TESTING_GUIDE.md`,
`docs/roadmaps/ADVERSARIAL_REVIEW_REMEDIATION_ROADMAP.md`, or
`docs/audits/ADVERSARIAL_REVIEW.md`. The project SOP requires new docs to be
added to that index. Three new documents, three misses.

### C. Jules' fifth finding was silently dropped. Grade: C

Jules' section 5 argues the LLM playtest harness is "a mirage" and an
ineffective substitute for deterministic QA. That claim never appears in the
roadmap in any form, neither adopted nor rejected. The roadmap's own operating
rules state that findings "may be regrouped as evidence develops, but must never
disappear."

Cursor then spent the entire window building 17,200 more lines of exactly the
thing Jules criticized, without ever recording the disagreement.

I think Jules is substantially **wrong** on the merits here. Its proposed
alternative, a DFS parser validating dialogue trees "in less than 5
milliseconds," cannot detect the defect class this project actually suffers from
(instructions buried in room prose, confusing prompts, bad pacing, wrong NPC
voice), and the project's content playtest gate exists precisely because
boot-clean never equals usable. But being wrong is not the same as being
unaddressed. A one-line rejection with that rationale was owed and never
written.

### B-. Cursor's review missed the worst availability defect. Grade: B-

Thirty-three findings, extensive concurrency coverage including three separate
race findings (4, 8, 21) and a lock-ordering finding (22), and yet
`RunWithMUDLocked` and the global admin write lock went unnoticed. Cursor's
concurrency lens was pointed at data races and missed lock *scope*. Worth
noting for calibration: breadth of findings is not the same as coverage.

### C. Dead artifacts left in the tree. Grade: C

`docs/superpowers/plans/2026-08-07-web-terminal-release-asset.md` and its
matching spec are untracked leftovers for chunk **0.1**, the chunk that was
invalidated. A plan and spec for work proven unnecessary should have been
deleted or annotated when the invalidation landed.

---

## The Ugly

### F. A live production password is tracked in a public repository. Grade: F

`tools/playtest/targets.yaml` is tracked on `origin/master` at
`github.com/pruuk/DOGMud`, which is public. It contains a plaintext password for
the `aitester` account on `dogmud.org:55555`, alongside a local `smoketester`
credential.

Cursor **found this** and filed it correctly as Chunk 1.5. Then it placed it in
Phase 1, marked **Not started**, behind a CI-parity chunk, with no urgency
marker anywhere in the document, and shipped an entire playtest infrastructure
arc without touching it.

This is a triage failure, not a detection failure. Credential exposure is the
one finding class where sequencing logic does not apply: the exposure window is
the whole time it stays open, and rotation cannot be scheduled behind a
dependency graph. It is also the one item on the list that a roadmap cannot fix,
because the repository history retains the secret regardless of what the working
tree says.

Worse, **the finding was incomplete.** There were two files, not one.
`tools/_archive/testing-pre-harness/testing/targets.yaml` carried the identical
`smoketester` and `aitester` credentials and is named in neither review nor in
the roadmap chunk. Cursor's own Docker-context work explicitly excludes
`tools/playtest/targets.yaml` from image layers, so it clearly understood the
file was sensitive, and still neither untracked it nor swept for copies.

In fairness, both files predate this window. They go back to commit `5b11d735f`,
the original `/test-mud` framework. Neither agent introduced them. But both
agents had the repository open for a full work cycle, one of them explicitly
found one of them, and both were still live at the start of this review.

**Remediated 2026-08-08** on branch `fix/remove-tracked-playtest-credentials`
(commit `21f076c29`):

- `tools/playtest/targets.yaml` untracked; local copy retained on disk so
  `/playtest prod` still works.
- `tools/_archive/testing-pre-harness/testing/targets.yaml` deleted; the
  pre-harness stack was retired 2026-06-08 and nothing consumes it.
- `tools/playtest/targets.example.yaml` added as a credential-free template.
- `.gitignore` ignores `**/targets.yaml`, negating the example.
- `CLAUDE.md` and `.claude/commands/playtest.md` document the setup step and a
  never-echo-the-password rule.
- Roadmap chunk 1.5 moved to In progress with rotation flagged as the blocker.

**Rotation is still outstanding and no repository change can substitute for
it.** The blobs remain reachable in history, in every existing clone, and in
every fork. Rotate `aitester` on `dogmud.org` and the local `smoketester`, and
confirm `aitester` holds no admin role.

### F. Jules fabricated a code citation, and the casing proves it. Grade: F

Jules' section 3 presents this as a quotation from the codebase:

```go
// internal/rooms/instances_test.go
func TestBTreeStateEviction(t *testing.T) {
	origEvictor := bTreeStateEvictor
	defer SetBTreeStateEvictor(origEvictor)
```

The real code at `instances_test.go:381-384`:

```go
func TestCheckPortalTimers_TtlExpiryEvictsBtreeState(t *testing.T) {
	// Snapshot/restore the package-level evictor.
	origEvictor := btreeStateEvictor
	defer SetBTreeStateEvictor(origEvictor)
```

Two divergences, and the second is the damning one:

1. `TestBTreeStateEviction` does not exist anywhere in the repository.
2. Jules wrote the variable as `bTreeStateEvictor`. The actual identifier is
   `btreeStateEvictor`, lowercase `t`, confirmed at `instances.go:227,229,247,254`.

**A copy-paste preserves casing.** Getting the capitalization of a package-level
variable wrong is only possible if the text was generated from memory of what
the code probably looks like, then presented inside a fenced block with a
file-path comment. That is not paraphrase. That is synthesized text formatted to
read as a verbatim citation.

The underlying argument is still correct: `instances_test.go` really does back
up and restore that global at lines 384, 424, and 502, so global callback
pointers really do force brittle test scaffolding. But a reader cannot tell
that from the document, and cannot distinguish "paraphrased" from "invented the
whole finding" without doing the verification themselves. It retroactively puts
every other code block in the review in question, which is exactly why I checked
all seven of Jules' remaining claims instead of sampling them.

### D. Jules' impact claims are inflated, and one is simply false. Grade: D

The findings are real. The consequences Jules attaches to them frequently are
not, and nothing in the document is measured.

- **Section 3's nil-panic claim is false.** Jules asserts that using a package
  before `main.go` wires the pointers means "the application will crash with a
  silent, untraceable `nil` pointer dereference." The seams are nil-guarded by
  design: `instances.go:227` reads `if btreeStateEvictor != nil`,
  `roommanager.go:376` reads `if companionTransport != nil`, and the setter's
  own doc comment says "Safe to leave unregistered in tests that don't exercise
  btree state." The roadmap independently reached the same conclusion. The
  observation (mutable globals) survives; the stated impact does not.
- **Two unmeasured "Critical" severities.** Sections 1 and 2 are both rated
  Critical. Section 2 predicts autosave "guarantees major periodic performance
  lag spikes" and will "rapidly degrade the server into an unplayable state,"
  with no benchmark, no loaded-room count, and no tick-delay figure. The project
  already has a production performance baseline showing copyover completing in
  under one second. The roadmap correctly refused to accept the severity and
  split the work into "measure first" (3.6a) and "remediate only if the budget
  is exceeded" (3.6b). That was the right call and it is a correction of Jules.
- **Section 5 is wrong on the merits**, as covered above.

Scoring the document honestly: five sections, of which two are solid (section 1
lock scope, section 4 progression and dice math), two are real observations
carrying false or unevidenced impact claims (sections 2 and 3), and one is
incorrect (section 5). The recommendations are generic ("Event-Driven
Architecture", "Asynchronous File Writes") and would apply unchanged to most Go
MUD codebases.

### F. Two of Jules' three commits misrepresent their contents. Grade: F

- `05d627468` message: "Delivers an exhaustive adversarial review of DOGMud's
  codebase in `docs/audits/ADVERSARIAL_REVIEW.md`, detailing architecture,
  locks, persistence, game math, and testing harness observations." The actual
  diff is **one line** in `.github/workflows/discord-notify.yml`. The review
  document is not in it.
- `e94f4a9da` message claims both the review document and the workflow
  safeguard. The commit is **empty**. Zero files changed.

Three commits, of which one is honest. Anyone reading `git log` to reconstruct
what happened is actively misled about which change delivered what.

### F. Jules' only code change was invalid and had a false rationale. Grade: F

```yaml
- name: Send Discord Message
  if: ${{ secrets.DISCORD_WEBHOOK_URL != '' }}
```

Two problems:

1. **It does not work.** GitHub Actions does not expose the `secrets` context in
   a step-level `if:` condition. The available contexts there are `github`,
   `needs`, `strategy`, `matrix`, `job`, `runner`, `env`, `vars`, `steps`, and
   `inputs`. The documented workaround is to promote the secret to a job-level
   `env` and test that instead. I did not execute the workflow to confirm the
   failure mode, so treat the specific failure as unverified, but the context is
   not available per GitHub's own availability rules.
2. **The stated reason is wrong.** Jules justified it as avoiding "CI failures
   when secrets are unavailable (e.g. in PR runs)." This workflow triggers on
   `pull_request: opened`, and same-repository pull requests **do** receive
   secrets. The problem it claims to solve does not occur on this trigger.

The change was reverted (`32f3c4bf2`, then merged via `1f18c2ac9`), so the
current tree is clean. But it cost two PRs and a revert.

---

## Report card

| Agent | Dimension | Grade |
|---|---|---|
| **Cursor** | Finding accuracy | **A** |
| **Cursor** | Intellectual honesty (self-invalidation, inconclusive-not-green) | **A** |
| **Cursor** | Roadmap process design | **A-** |
| **Cursor** | Harness code quality | **B+** |
| **Cursor** | Coverage gaps (missed the global admin lock) | **B-** |
| **Cursor** | Doc hygiene (untracked source doc, no index entries) | **C-** |
| **Cursor** | Handling of the competing review's section 5 | **C** |
| **Cursor** | Prioritization and delivery (0 of 36 findings fixed) | **D** |
| **Cursor** | Credential triage (found one file, missed the second, shipped neither) | **F** |
| **Cursor** | **Overall** | **B-** |
| **Jules** | Architectural insight (2 genuine catches the other review missed) | **B** |
| **Jules** | Observation accuracy (structural claims verify) | **B** |
| **Jules** | Impact/severity accuracy (one false, two unmeasured Criticals) | **D** |
| **Jules** | Citation integrity (synthesized code block, wrong identifier casing) | **F** |
| **Jules** | Code contribution (invalid, false rationale, reverted) | **F** |
| **Jules** | Commit honesty (1 of 3 accurate, 1 empty) | **F** |
| **Jules** | **Overall** | **D** |

**Cursor harness: B-.** Excellent analysis, mature process, competent code, and
almost no delivery against the problem it was pointed at. Trust its findings.
Do not trust it to sequence its own work toward the highest-value item. Note
this grades the harness, not any single model in it.

**Jules: D.** Regraded down from an initial C-, which was too generous. The
initial grade weighted "found two things the more thorough reviewer missed" as
if it offset the rest. It does not.

Net contribution to this repository after the revert: **one markdown file, which
contains a fabricated code block.** Everything Jules changed in the codebase was
invalid and was backed out. Of its three commits, one is honest, one describes
work absent from its diff, and one is empty. Of its five findings, two are
solid, two carry impact claims that are unevidenced or provably false, and one
is wrong. And the fabricated citation is not a near-miss: the identifier casing
mismatch shows the code block was generated rather than read, which means the
document asserts verification it did not perform.

The two real catches (the global admin write lock, the `SkillMultiplier`
ceiling) are worth keeping and are now tracked as roadmap chunks 3.5 and 5.7.
Keep the findings. Do not extend trust to the document.

---

## Recommended next actions

1. **Rotate `aitester` and `smoketester` credentials now.** The repository side
   of Chunk 1.5 is done (see above); rotation is the part only you can do, and
   removing the file does not un-expose a secret that is in public history.
2. **Ship the four one-line fixes.** Findings 11 (wander filter), 12 (gold
   parse), 32 (`WrapAnsi` named return), and 28 (two nil dereferences) are
   contained, independently verified above, and currently blocked behind a
   dependency chain they do not need.
3. **Commit the review, or delete the roadmap's reference to it.** Move
   `ADVERSARIAL_CODE_REVIEW_2026-08-07.md` to `docs/audits/` and track it. The
   roadmap is worthless to anyone who cannot read its source document.
4. **Index the four new docs** in `docs/README.md`, including this one.
5. **Record a disposition for Jules' section 5.** Adopt it, reject it with
   rationale, or mark it Invalidated. The roadmap's own rules forbid leaving it
   unrecorded.
6. **Re-verify Jules' review before acting on chunks 3.5, 3.6, 5.7, and 5.8.**
   The findings themselves are confirmed above, but the document contains at
   least one invented identifier, so treat any *other* code sample in it as
   unverified.
7. **Delete the chunk 0.1 plan and spec.** They plan work that was disproven.
8. **Impose a delivery constraint on the next window.** Enabling chunks should
   not outweigh finding closures again. Findings closed is currently zero.

---

## What I did not verify

Stated plainly so this document does not repeat the failure it grades:

- Findings 5, 6, 7, 13, 14, 15, 16, 18, 19, 23, 24, 25, 27, 29, 30, 31, and 33
  from Cursor's review. Given a 16-for-16 hit rate on the sample, I expect them
  to hold, but I did not read the cited code.
- The Docker integration test evidence claimed in the 0.3a through 0.3d status
  blocks. I ran the native package tests (all pass) but not the gated Docker
  integration suites.
- Whether Jules' step-level `secrets` condition fails at parse time or evaluates
  to empty. The context-availability rule is documented; the runtime behavior
  was not executed here.
- The two untracked room files (`pothole_coulee/6468.yaml`, `6469.yaml`) and the
  uncommitted `_datafiles/config.yaml` changes. These appear to be human working
  state rather than agent output, so they fall outside this review's scope.
