# Mutation Art + Reveal (5d) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Per-mutation emblem art (all 62 + generic fallback) and a reveal event that upgrades mutation acquisition into a chrysalis splash ceremony on terminal clients and a toast→card reveal on the web client.

**Architecture:** One new event (`mutations.Gained`) emitted from every player grant site; an `internal/hooks` listener renders the terminal side (splash for new, flourish for deepen); a `modules/gmcp` listener pushes `Char.Mutation` for the web toast/card. Art is generated via image-gen MCP on the panel leather tint (`#201913`), postprocessed to 256px PNGs.

**Tech Stack:** Go (events/listeners/templates), text/template splash scene, vanilla JS in `webclient-pure.html`, `image-gen-mcp` (gpt-image-2, **quality `low` ONLY**), Python/Pillow postprocess.

**Spec:** `docs/superpowers/specs/2026-07-29-mutation-art-reveal-design.md` — read it first. Key constraint discovered in planning: `events` → `skills` → `mutations`, so **`internal/mutations` must NOT import `internal/events`**. The event struct satisfies `events.Event` structurally.

**Verified facts** (2026-07-29, don't re-derive):
- `MutationSpec` fields: `MutationId`, `Name`, `Description`, `Rarity`, `MaxRank` (`internal/mutations/mutations.go:30`); `GetMutation(id) *MutationSpec` at `:151`.
- `mutations.SeedMutationsForTest(map[string]*MutationSpec) func()` exists (`test_helpers.go:6`).
- Splash: `splash.Splash{SceneId, Caption, Target, UserId, Data}`; constants `TargetGlobal/TargetZone/TargetUser`; templates at `_datafiles/world/dogmud/templates/generic/splash/<sceneId>.template`; delivery listener `internal/hooks/Splash_Deliver.go` skips everything when `GamePlay.SplashesEnabled` is false.
- GMCP send: `events.AddToQueue(GMCPOut{UserId, Module, Payload})` (see `modules/gmcp/gmcp.Comm.go`); modules self-register in `func init()` with `plugins.New`.
- Web client GMCP routing: `GMCPUpdateHandlers` map keyed by module name in `_datafiles/html/public/webclient-pure.html` (~line 632); payloads shallow-merge into `GMCPStructs`.
- Template func `splitstring` = `util.SplitStringNL` (registered in `internal/templates/templatesfunctions.go:117`).
- CSS tokens: `--panel-bg #201913`, `--panel-border #3a2a18`, `--ink-gold #c9a86a`.
- `messaging.CategoryMutation` exists and is what current announce lines use.
- Config: `Balance.MutationMaxLevel` (default 3).

---

## Part 1 — Server: event + listeners

### Task 1: `mutations.Gained` event struct

**Files:**
- Create: `internal/mutations/reveal.go`
- Test: `internal/mutations/reveal_test.go` (external test package)

- [ ] **Step 1: Write the failing test**

```go
package mutations_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mutations"
)

// Compile-time proof Gained satisfies events.Event WITHOUT mutations
// importing events (events→skills→mutations would cycle; the external
// test package may import both).
var _ events.Event = mutations.Gained{}

func TestGainedType(t *testing.T) {
	if got := (mutations.Gained{}).Type(); got != `MutationGained` {
		t.Errorf(`Gained.Type() = %q, want "MutationGained"`, got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/mutations/ -run TestGainedType -v`
Expected: FAIL to build — `undefined: mutations.Gained`

- [ ] **Step 3: Write minimal implementation**

```go
package mutations

// Gained is the player-facing mutation-reveal event (spec 2026-07-29).
// It deliberately does NOT import internal/events — events → skills →
// mutations would cycle — and satisfies events.Event structurally via
// Type(). Emit with events.AddToQueue(mutations.Gained{...}) from call
// sites (they all already import events).
type Gained struct {
	UserId     int
	MutationId string
	Rank       int  // new rank after the change (1 for acquisitions)
	IsNew      bool // false = an owned mutation deepened
}

func (Gained) Type() string { return `MutationGained` }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/mutations/ -run TestGainedType -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/mutations/reveal.go internal/mutations/reveal_test.go
git commit -m "feat(mutations): Gained reveal event (cycle-safe, structural events.Event)"
```

### Task 2: Terminal listener (`internal/hooks`)

**Files:**
- Create: `internal/hooks/MutationGained_Reveal.go`
- Test: `internal/hooks/MutationGained_Reveal_test.go`
- Modify: `internal/hooks/hooks.go` (register listener; add `mutations` import)

- [ ] **Step 1: Write the failing tests** (pure-helper style, mirrors `Splash_Deliver_test.go`)

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/stretchr/testify/assert"
)

type notAGainedEvent struct{}

func (notAGainedEvent) Type() string { return "dummy" }

func TestMutationGainedRevealWrongEventType(t *testing.T) {
	assert.Equal(t, events.Continue, MutationGained_Reveal(notAGainedEvent{}))
}

func TestDeepenFlourishText(t *testing.T) {
	got := deepenFlourishText("Chameleon Skin", 2, 3)
	assert.Contains(t, got, "Chameleon Skin")
	assert.Contains(t, got, "Level 2")
	assert.NotContains(t, got, "fully matured")

	got = deepenFlourishText("Chameleon Skin", 3, 3)
	assert.Contains(t, got, "fully matured")
}

func TestRevealCaption(t *testing.T) {
	got := revealCaption("Chameleon Skin", "Your skin drinks the colors around it.")
	// The caption is the screen-reader / degraded path — plain text, no ansi tags.
	assert.Contains(t, got, "A mutation emerges: Chameleon Skin.")
	assert.Contains(t, got, "Your skin drinks the colors around it.")
	assert.NotContains(t, got, "<ansi")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/hooks/ -run "TestMutationGainedReveal|TestDeepenFlourish|TestRevealCaption" -v`
Expected: FAIL to build — `undefined: MutationGained_Reveal` etc.

- [ ] **Step 3: Write the listener**

`internal/hooks/MutationGained_Reveal.go`:

```go
package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/splash"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// deepenFlourishText is the rank-up announcement (no splash ceremony —
// spec 2026-07-29: deepenings get the lighter reveal).
func deepenFlourishText(name string, rank int, maxLevel int) string {
	levelTag := fmt.Sprintf(`Level %d`, rank)
	if rank >= maxLevel {
		levelTag = `fully matured`
	}
	return fmt.Sprintf(
		`<ansi fg="magenta">The Chrysalis deepens its hold. Your <ansi fg="yellow">%s</ansi> grows stronger (%s).</ansi>`,
		name, levelTag)
}

// revealCaption is the plain-text form: the splash caption for
// screen-reader users, and part of the degraded no-splash path.
func revealCaption(name, description string) string {
	return fmt.Sprintf(`Something stirs beneath your skin. A mutation emerges: %s. %s`, name, description)
}

// MutationGained_Reveal renders the terminal side of a mutation reveal:
// a per-user chrysalis splash ceremony for new acquisitions, a short
// flourish for deepenings. The web card is pushed separately by the gmcp
// module listening to the same event.
func MutationGained_Reveal(e events.Event) events.ListenerReturn {
	evt, ok := e.(mutations.Gained)
	if !ok {
		return events.Continue
	}
	user := users.GetByUserId(evt.UserId)
	if user == nil {
		return events.Continue
	}
	spec := mutations.GetMutation(evt.MutationId)
	if spec == nil {
		return events.Continue
	}

	if !evt.IsNew {
		user.SendText(messaging.CategoryMutation,
			deepenFlourishText(spec.Name, evt.Rank, int(configs.GetBalanceConfig().MutationMaxLevel)))
		return events.Continue
	}

	if !bool(configs.GetGamePlayConfig().SplashesEnabled) {
		// Splash_Deliver drops everything when splashes are disabled, so
		// degrade here to the classic two-line announcement.
		user.SendText(messaging.CategoryMutation, fmt.Sprintf(
			`<ansi fg="magenta">Something stirs beneath your skin. A mutation emerges: <ansi fg="yellow">%s</ansi>.</ansi>`,
			spec.Name))
		user.SendText(messaging.CategoryMutation, fmt.Sprintf(`<ansi fg="magenta">%s</ansi>`, spec.Description))
		return events.Continue
	}

	events.AddToQueue(splash.Splash{
		SceneId: `mutation_reveal`,
		Caption: revealCaption(spec.Name, spec.Description),
		Target:  splash.TargetUser,
		UserId:  evt.UserId,
		Data: map[string]any{
			`name`:        spec.Name,
			`description`: spec.Description,
		},
	})
	return events.Continue
}
```

In `internal/hooks/hooks.go`, after the existing splash registration (line ~17), add:

```go
	// Mutation reveal (terminal ceremony/flourish; web card via gmcp module)
	events.RegisterListener(mutations.Gained{}, MutationGained_Reveal)
```

and add `"github.com/GoMudEngine/GoMud/internal/mutations"` to its imports.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/hooks/ -run "TestMutationGainedReveal|TestDeepenFlourish|TestRevealCaption" -v`
Expected: PASS (3 tests)

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/MutationGained_Reveal.go internal/hooks/MutationGained_Reveal_test.go internal/hooks/hooks.go
git commit -m "feat(hooks): terminal mutation reveal — splash ceremony for new, flourish for deepen"
```

### Task 3: Emit from all player grant sites

**Files:**
- Modify: `internal/hooks/NewRound_UserRoundTick.go:288-320` (drift deepen + acquire branches)
- Modify: `internal/hooks/pinnacle_tick.go:191-215` (`tickMutationItems`; add `events` import)
- Modify: `internal/behaviortree/actions_quest.go:56-63` (`actGrantMutation`; add `mutations` import if absent)
- Modify: `internal/questengine/bridge.go` (`GiveMutation`, ~line 332)

No new unit tests in this task — the listener behavior is covered by Task 2 and existing drift tests must stay green; the full flow is verified by the Task 10 playtest.

- [ ] **Step 1: Drift deepen branch** — in `NewRound_UserRoundTick.go`, replace the block at ~288-302:

```go
							if doDeepen {
								mutId := mutations.RollDeepening(user.Character.Mutations)
								if mutId != "" {
									user.Character.Mutations[mutId]++
									events.AddToQueue(mutations.Gained{
										UserId:     user.UserId,
										MutationId: mutId,
										Rank:       user.Character.Mutations[mutId],
										IsNew:      false,
									})
								}
							} else if canAcquire {
```

(The old `spec := mutations.GetMutation(...)` / `levelTag` / `user.SendText(...)` lines are deleted — the listener owns all text now.)

- [ ] **Step 2: Drift acquire branch** — same file, ~303-320: after `user.Character.Mutations[mutId] = 1`, delete the two `user.SendText(...)` calls (the "Something stirs…" line and the description line) and add the emit **before** the retained `spec != nil` block; the worldevents gossip block stays exactly as-is:

```go
										user.Character.Mutations[mutId] = 1
										events.AddToQueue(mutations.Gained{
											UserId:     user.UserId,
											MutationId: mutId,
											Rank:       1,
											IsNew:      true,
										})
										spec := mutations.GetMutation(mutId)
										if spec != nil {
											// Emit world event for gossip system
											// ... (existing worldevents block, unchanged)
```

- [ ] **Step 3: Pinnacle tick** — in `pinnacle_tick.go` `tickMutationItems`, replace lines 204-213 (grant + name lookup + SendText):

```go
		granted := c.GrantRandomMutationRare(spec.MutationRarityFloor)
		if granted == "" {
			continue
		}
		events.AddToQueue(mutations.Gained{
			UserId:     user.UserId,
			MutationId: granted,
			Rank:       1,
			IsNew:      true,
		})
```

Add `"github.com/GoMudEngine/GoMud/internal/events"` to the file's imports. If `fmt`/`messaging`/`mutations` become unused in this file, remove them (the compiler will say).

- [ ] **Step 4: Behavior tree (Awakening Rite)** — in `actions_quest.go`, `actGrantMutation` (currently grants **silently**):

```go
func actGrantMutation(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	if mutId := user.Character.GrantRandomMutation(); mutId != "" {
		events.AddToQueue(mutations.Gained{
			UserId:     user.UserId,
			MutationId: mutId,
			Rank:       1,
			IsNew:      true,
		})
	}
	return Success
}
```

Add `"github.com/GoMudEngine/GoMud/internal/mutations"` to imports (`events` is already imported).

- [ ] **Step 5: Quest engine** — in `bridge.go`, `GiveMutation` (currently grants **silently**), the tail becomes:

```go
	mutId := mutations.RollAcquisition(pool)
	if mutId == "" {
		return
	}
	if _, exists := b.user.Character.Mutations[mutId]; !exists {
		b.user.Character.Mutations[mutId] = 1
		events.AddToQueue(mutations.Gained{
			UserId:     b.user.UserId,
			MutationId: mutId,
			Rank:       1,
			IsNew:      true,
		})
	}
```

(`events` and `mutations` are already imported here.)

**Bloom stays untouched** — `drink.go` / `bloom_mutation.go` keep their deliberately vague line and emit nothing (spec decision).

- [ ] **Step 6: Build + run the touched packages' suites**

Run: `go build ./... && go test ./internal/hooks/ ./internal/behaviortree/ ./internal/questengine/ ./internal/mutations/`
Expected: build OK, all PASS

- [ ] **Step 7: Commit**

```bash
git add internal/hooks/NewRound_UserRoundTick.go internal/hooks/pinnacle_tick.go internal/behaviortree/actions_quest.go internal/questengine/bridge.go
git commit -m "feat(mutations): emit Gained from all player grant sites (rite + quest grants were silent)"
```

### Task 4: `mutation_reveal` splash template

**Files:**
- Create: `_datafiles/world/dogmud/templates/generic/splash/mutation_reveal.template`

- [ ] **Step 1: Author the scene.** Chrysalis motif (split cocoon), magenta/yellow palette matching `messaging.CategoryMutation`'s existing colors, all lines ≤ 80 cols, frame style copied from `moon_eye_full.template`. `{{ .name }}` and `{{ splitstring .description 73 }}` render the specifics (splitstring = `util.SplitStringNL`, wraps the paragraph):

```
 <ansi fg="239">┌─────────────────────────────────────────────────────────────────────────┐</ansi>

                  <ansi fg="60">˙    ·         ✦         ·    ˙</ansi>
                        <ansi fg="97">░▒▓▒░   ░▒▓▒░</ansi>
                      <ansi fg="133">░▒▓██▓▒░ ░▒▓██▓▒░</ansi>
                     <ansi fg="170">▒▓████▓▒░ ░▒▓████▓▒</ansi>
                     <ansi fg="207">▓█████▓▒</ansi><ansi fg="229">░ ░</ansi><ansi fg="207">▒▓█████▓</ansi>
                     <ansi fg="170">▒▓████▓▒</ansi><ansi fg="229">░ ░</ansi><ansi fg="170">▒▓████▓▒</ansi>
                      <ansi fg="133">░▒▓██▓▒░ ░▒▓██▓▒░</ansi>
                        <ansi fg="97">░▒▓▒░   ░▒▓▒░</ansi>
                  <ansi fg="60">·    ˙         ·         ˙    ·</ansi>

              <ansi fg="253">The chrysalis splits. Something new is true.</ansi>

     <ansi fg="229">{{ .name }}</ansi>
<ansi fg="133">{{ splitstring .description 73 }}</ansi>

 <ansi fg="239">└─────────────────────────────────────────────────────────────────────────┘</ansi>
```

- [ ] **Step 2: Verify it renders** — boot the server locally (`go run .` after the instance-save wipe SOP is NOT needed for a template add, but boot must not error) and watch for template/load errors on startup. Then in a local client, as an admin, trigger a reveal if a quick path exists — otherwise rendering is exercised in Task 10's playtest.

Run: `go run .` (Ctrl+C after clean boot past data-file loading)
Expected: no template errors, `mobs.LoadDataFiles() loadedCount=…` etc. reached

- [ ] **Step 3: Commit**

```bash
git add "_datafiles/world/dogmud/templates/generic/splash/mutation_reveal.template"
git commit -m "content(splash): mutation_reveal chrysalis scene"
```

### Task 5: GMCP module (`Char.Mutation`)

**Files:**
- Create: `modules/gmcp/gmcp.Mutation.go`
- Test: `modules/gmcp/gmcp.Mutation_test.go`

- [ ] **Step 1: Write the failing test**

```go
package gmcp

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mutations"
)

func TestBuildMutationPayload(t *testing.T) {
	restore := mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"chameleon-skin": {
			MutationId:  "chameleon-skin",
			Name:        "Chameleon Skin",
			Description: "Your skin drinks the colors around it.",
		},
	})
	defer restore()

	p, ok := buildMutationPayload(mutations.Gained{
		UserId: 1, MutationId: "chameleon-skin", Rank: 2, IsNew: false,
	})
	if !ok {
		t.Fatal("expected payload for a registered mutation")
	}
	if p.Id != "chameleon-skin" || p.Name != "Chameleon Skin" || p.Rank != 2 || p.IsNew {
		t.Errorf("payload fields wrong: %+v", p)
	}
	if p.Art != "/static/images/mutations/chameleon-skin.png" {
		t.Errorf("art path = %q", p.Art)
	}
	if p.Description == "" {
		t.Error("description must ship")
	}

	if _, ok := buildMutationPayload(mutations.Gained{MutationId: "nope"}); ok {
		t.Error("unknown mutation must not build a payload")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./modules/gmcp/ -run TestBuildMutationPayload -v`
Expected: FAIL to build — `undefined: buildMutationPayload`

- [ ] **Step 3: Write the module** (mirrors `gmcp.Comm.go` registration)

```go
package gmcp

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/plugins"
)

func init() {
	g := GMCPMutationModule{
		plug: plugins.New(`gmcp.Mutation`, `1.0`),
	}
	events.RegisterListener(mutations.Gained{}, g.onMutationGained)
}

type GMCPMutationModule struct {
	plug *plugins.Plugin
}

// GMCPMutationModule_Payload is pushed as "Char.Mutation" when a player
// acquires (isNew) or deepens a mutation. The web client shows a corner
// toast that expands to the ceremonial card. No mechanical numbers beyond
// rank (used only for "deepens" wording client-side).
type GMCPMutationModule_Payload struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Rank        int    `json:"rank"`
	IsNew       bool   `json:"isNew"`
	Description string `json:"description"`
	Art         string `json:"art"`
}

func buildMutationPayload(evt mutations.Gained) (GMCPMutationModule_Payload, bool) {
	spec := mutations.GetMutation(evt.MutationId)
	if spec == nil {
		return GMCPMutationModule_Payload{}, false
	}
	return GMCPMutationModule_Payload{
		Id:          evt.MutationId,
		Name:        spec.Name,
		Rank:        evt.Rank,
		IsNew:       evt.IsNew,
		Description: spec.Description,
		Art:         `/static/images/mutations/` + evt.MutationId + `.png`,
	}, true
}

func (g *GMCPMutationModule) onMutationGained(e events.Event) events.ListenerReturn {
	evt, ok := e.(mutations.Gained)
	if !ok {
		return events.Continue
	}
	payload, ok := buildMutationPayload(evt)
	if !ok {
		return events.Continue
	}
	events.AddToQueue(GMCPOut{
		UserId:  evt.UserId,
		Module:  `Char.Mutation`,
		Payload: payload,
	})
	return events.Continue
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./modules/gmcp/ -run TestBuildMutationPayload -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add modules/gmcp/gmcp.Mutation.go modules/gmcp/gmcp.Mutation_test.go
git commit -m "feat(gmcp): Char.Mutation reveal push"
```

### Task 6: Web client toast → card

**Files:**
- Modify: `_datafiles/html/public/webclient-pure.html` — three additions: CSS (in the existing `<style>` block), DOM (two elements before `</body>`), JS (handler in `GMCPUpdateHandlers` ~line 707 + the queue/render functions near the other helper functions).

No automated test (inline page script, no harness) — gated by the user browser eyeball in Task 10.

- [ ] **Step 1: CSS** — add to the page's `<style>` block:

```css
/* ── Mutation reveal (toast → ceremonial card) ─────────────────── */
#mut-toast {
    position: absolute; top: 14px; right: 14px; z-index: 60;
    display: none; align-items: center; gap: 10px; max-width: 260px;
    background: linear-gradient(160deg, #2b2118, var(--panel-bg, #201913));
    border: 1px solid var(--panel-border, #3a2a18); border-radius: 8px;
    padding: 10px 14px; cursor: pointer;
    box-shadow: 0 4px 18px rgba(0,0,0,.6);
}
#mut-toast.show { display: flex; }
#mut-toast img, #mut-card img {
    border-radius: 50%; border: 1px solid var(--ink-gold, #c9a86a);
    background: var(--panel-bg, #201913); object-fit: cover;
}
#mut-toast img { width: 44px; height: 44px; }
#mut-toast .mt-name { color: #f0e2c8; font-family: Georgia, serif; font-size: 13px; }
#mut-toast .mt-sub  { color: #8d7c64; font-size: 10px; }
#mut-card-overlay {
    position: absolute; inset: 0; z-index: 70; display: none;
    background: rgba(10,7,5,.55); align-items: center; justify-content: center;
    cursor: pointer;
}
#mut-card-overlay.show { display: flex; }
#mut-card {
    background: linear-gradient(160deg, #2b2118, var(--panel-bg, #201913));
    border: 2px solid var(--ink-gold, #c9a86a); border-radius: 10px;
    padding: 22px 34px; text-align: center; max-width: 340px;
    box-shadow: 0 8px 40px rgba(0,0,0,.7);
}
#mut-card img { width: 128px; height: 128px; margin: 0 auto 12px; display: block; }
#mut-card .mc-kicker { color: #d9b36a; font-size: 11px; letter-spacing: .25em; text-transform: uppercase; }
#mut-card .mc-name   { color: #f0e2c8; font-size: 20px; font-family: Georgia, serif; margin: 6px 0 8px; }
#mut-card .mc-desc   { color: #a3937c; font-size: 12px; }
#mut-card .mc-hint   { color: #6e5f4b; font-size: 10px; margin-top: 12px; }
```

- [ ] **Step 2: DOM** — add just before `</body>` (or beside the other overlay elements if the file groups them):

```html
<div id="mut-toast" onclick="mutationRevealExpand()">
    <img id="mut-toast-art" src="" alt="" onerror="this.onerror=null;this.src='/static/images/mutations/_generic.png'">
    <div>
        <div class="mt-name" id="mut-toast-name"></div>
        <div class="mt-sub" id="mut-toast-sub"></div>
    </div>
</div>
<div id="mut-card-overlay" onclick="mutationRevealDismiss()">
    <div id="mut-card">
        <img id="mut-card-art" src="" alt="" onerror="this.onerror=null;this.src='/static/images/mutations/_generic.png'">
        <div class="mc-kicker">The Chrysalis acts</div>
        <div class="mc-name" id="mut-card-name"></div>
        <div class="mc-desc" id="mut-card-desc"></div>
        <div class="mc-hint">click anywhere to continue</div>
    </div>
</div>
```

Both live inside the client's main relative-positioned container so `position:absolute` anchors to the client area, not the page — mirror where the reconnect badge sits; if that container is `document.body`-relative, switch the two outer divs to `position: fixed`.

- [ ] **Step 3: JS** — add near the other helpers (same `<script>` scope as `GMCPUpdateHandlers`):

```js
// ── Mutation reveal: FIFO toast queue → ceremonial card ────────────
var mutRevealQueue = [];
var mutRevealCurrent = null;
var mutRevealFadeTimer = null;
var MUT_TOAST_FADE_MS = 20000;

function mutationRevealEnqueue(m) {
    mutRevealQueue.push(m);
    if (!mutRevealCurrent) { mutationRevealShowNext(); }
}
function mutationRevealShowNext() {
    if (mutRevealFadeTimer) { clearTimeout(mutRevealFadeTimer); mutRevealFadeTimer = null; }
    mutRevealCurrent = mutRevealQueue.shift() || null;
    var toast = document.getElementById('mut-toast');
    if (!mutRevealCurrent) { toast.classList.remove('show'); return; }
    document.getElementById('mut-toast-art').src = mutRevealCurrent.art || '/static/images/mutations/_generic.png';
    document.getElementById('mut-toast-name').textContent = mutRevealCurrent.name || '';
    document.getElementById('mut-toast-sub').textContent = mutRevealCurrent.isNew
        ? 'A mutation emerges — click to view'
        : 'Deepens its hold — click to view';
    toast.classList.add('show');
    mutRevealFadeTimer = setTimeout(function () {
        // Unclicked toast fades; move on to any queued reveal.
        mutationRevealShowNext();
    }, MUT_TOAST_FADE_MS);
}
function mutationRevealExpand() {
    if (!mutRevealCurrent) { return; }
    if (mutRevealFadeTimer) { clearTimeout(mutRevealFadeTimer); mutRevealFadeTimer = null; }
    document.getElementById('mut-toast').classList.remove('show');
    document.getElementById('mut-card-art').src = mutRevealCurrent.art || '/static/images/mutations/_generic.png';
    document.getElementById('mut-card-name').textContent = mutRevealCurrent.name || '';
    document.getElementById('mut-card-desc').textContent = mutRevealCurrent.description || '';
    document.getElementById('mut-card-overlay').classList.add('show');
}
function mutationRevealDismiss() {
    document.getElementById('mut-card-overlay').classList.remove('show');
    mutationRevealShowNext();
}
```

And register the handler inside `GMCPUpdateHandlers` (after `"Char.Automation"`):

```js
            "Char.Mutation": function() {
                var m = (GMCPStructs["Char"] || {}).Mutation;
                if (m && m.name) { mutationRevealEnqueue(m); }
            },
```

- [ ] **Step 4: Smoke it** — boot locally, open the web client, and from the browser dev console simulate: `mutationRevealEnqueue({id:"chameleon-skin",name:"Chameleon Skin",isNew:true,description:"Your skin drinks the colors around it.",art:"/static/images/mutations/chameleon-skin.png"})`. Toast appears (generic art until Task 9 lands), click → card, click → dismissed. Enqueue two back-to-back and confirm the second shows after the first dismisses.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/html/public/webclient-pure.html
git commit -m "feat(webclient): mutation reveal toast + ceremonial card (Char.Mutation)"
```

---

## Part 2 — Art batch (user-gated, main session drives the MCP)

> Image generation is done by the MAIN session calling `image-gen-mcp` tools
> directly (subagents are shell/MCP-constrained on this machine). **Quality
> `low` ONLY** — high/medium time out AND still bill (memory:
> `project_icon_gap_generation`). Generate on the leather tint `#201913`
> (state it in the prompt: "flat solid background, exact hex #201913") so no
> chroma-key strip is needed; the circular CSS frame shows the same tint.
> Fallback if the pilot looks bad: transparent generation + `tools/strip_icon_bg.py`.

### Task 7: Manifest + postprocess script

**Files:**
- Create: `tools/mutation_art/manifest.yaml`
- Create: `tools/mutation_art/postprocess.py`

- [ ] **Step 1: Author the manifest.** One entry per mutation YAML in `_datafiles/world/dogmud/mutations/` (all 62; enumerate with `ls`). Structure:

```yaml
# Mutation emblem art manifest — regeneration-reproducible.
# prompt = style_prefix + " " + subject + " " + style_suffix
style_prefix: >-
  <LOCKED IN TASK 8 — the winning house-style sentence, e.g. "Antique
  engraved emblem in a circular medallion, aged gold and bone-white ink
  on dark leather,">
style_suffix: >-
  centered composition, no text, no letters, flat solid background
  exact hex #201913, single emblem only
emblems:
  apex-predator: "a snarling predator skull fused with a hooked claw, radiating short dread-lines"
  chameleon-skin: "a lizard eye within overlapping color-shifting scales"
  # ... one line per mutation, authored from each YAML's name/description/visual ...
  _generic: "a split chrysalis cocoon with light escaping the seam, three small moons above"
```

Subject lines are authored content — write all 62 by reading each mutation's `name`/`description`/`visual` fields; keep each under ~20 words, concrete nouns, no style words (style lives in the prefix).

- [ ] **Step 2: Write `postprocess.py`** (mirrors the Pillow steps from the icon pipeline, minus background strip):

```python
"""Downscale generated 1024px mutation emblems to 256px PNGs.

Usage: python tools/mutation_art/postprocess.py <in_dir> [--out-dir DIR]
In-dir: raw 1024x1024 PNGs named <mutationid>.png from image-gen.
Out:    _datafiles/html/public/static/images/mutations/<mutationid>.png
"""
import argparse
import pathlib

from PIL import Image

DEFAULT_OUT = pathlib.Path("_datafiles/html/public/static/images/mutations")
SIZE = 256

def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("in_dir", type=pathlib.Path)
    ap.add_argument("--out-dir", type=pathlib.Path, default=DEFAULT_OUT)
    args = ap.parse_args()
    args.out_dir.mkdir(parents=True, exist_ok=True)
    count = 0
    for src in sorted(args.in_dir.glob("*.png")):
        img = Image.open(src).convert("RGB")  # solid-tint bg: no alpha needed
        img = img.resize((SIZE, SIZE), Image.LANCZOS)
        img.save(args.out_dir / src.name, optimize=True)
        count += 1
    print(f"postprocessed {count} emblems -> {args.out_dir}")

if __name__ == "__main__":
    main()
```

- [ ] **Step 3: Commit**

```bash
git add tools/mutation_art/manifest.yaml tools/mutation_art/postprocess.py
git commit -m "tools(mutation-art): prompt manifest + 256px postprocess"
```

### Task 8: Style lock (USER GATE — visual companion)

- [ ] **Step 1:** Generate **3 style variants of `chameleon-skin`** via `image-gen-mcp` at quality `low`, one per candidate house style (same subject line, different prefixes): (a) antique engraved woodcut sigil, (b) brass-and-verdigris alchemical emblem, (c) inked tarot-card mark with a thin border. Each prompt ends with the `style_suffix` from the manifest (solid `#201913` background).
- [ ] **Step 2:** Postprocess all three, show side-by-side in the visual companion (browser) inside the circular leather frame mock so the user judges them in context. **Also judge the tint match here** — if the generated background visibly mismatches `#201913` at the circle edge, fall back to transparent generation + `strip_icon_bg.py` for all later tasks.
- [ ] **Step 3:** User picks the house style → write the winning sentence into `manifest.yaml` `style_prefix`. Commit: `git commit -am "tools(mutation-art): lock house style"`.

### Task 9: Pilot ×3, then the full batch (USER GATES)

- [ ] **Step 1: Pilot.** Generate 3 more in the locked style — one common (rarity 3-4), one bridge (rarity 8), one apex (rarity 9) — postprocess, show in the browser next to the Task 8 winner. User approves consistency (or adjusts the prefix; regenerate, re-gate).
- [ ] **Step 2: Batch.** Generate the remaining 58 + `_generic.png` from the manifest, one MCP call each, saving raw output to a scratch dir; run `postprocess.py`; total new cost ~$0.35. Log any generation failure and retry it once; list any still-missing ids rather than silently skipping.
- [ ] **Step 3: Grid review (USER GATE).** Show all 63 images in a browser grid (id + name captions). User flags rerolls; regenerate flagged ones (tweak subject line, same style prefix) until accepted.
- [ ] **Step 4: Commit**

```bash
git add _datafiles/html/public/static/images/mutations/
git commit -m "assets(mutations): 62 emblem set + generic fallback (house style locked)"
```

---

## Part 3 — Verification

### Task 10: Full verification + playtest gate

- [ ] **Step 1: Full suite + vet**

Run: `go vet ./... && go test ./...`
Expected: clean vet; all packages PASS

- [ ] **Step 2: Boot test per pre-push SOP.** Wipe instance saves first (SOP):

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run .
```

Expected: clean boot past `mobs.LoadDataFiles()` / `quests.LoadDataFiles()`, no template panics. Kill the server after.

- [ ] **Step 3: Adversarial playtest gate (REQUIRED — content SOP).** Run `/playtest local bug-finder` with a goals brief: create a fresh character, complete the Awakening Rite (its `actGrantMutation` now announces — read the reveal output line-by-line: splash scene renders inside 80 cols, name + wrapped description present, no template artifacts, no double-announcement from any old text path we missed), then grind combat until a drift acquisition and, if reachable, a deepening (verify flourish wording). Confirm the `Char.Mutation` GMCP event arrives (harness GMCP capture) with correct id/name/art path. Report every usability defect; fix and re-run until clean.
- [ ] **Step 4: USER GATES (hand over):** (a) web toast/card eyeball in the real browser client — trigger via a rite or admin grant; (b) confirm the emblem art shows (not `_generic.png`) for the granted mutation.
- [ ] **Step 5: PATCH_NOTES.md entry + final commit** (per pre-push SOP; leave `Logging.LogToFile` alone until an actual prod push is prepared).

```bash
git add PATCH_NOTES.md
git commit -m "docs(patch-notes): mutation reveal — emblems, chrysalis splash, web card"
```

---

## Self-review notes

- Spec coverage: art batch (T7-9), Gained event (T1), emit sites incl. silent rite/quest gaps (T3), Bloom exclusion (T3 note), terminal splash + degradation + screen-reader caption (T2/T4), GMCP + toast/card + `_generic` fallback + queue (T5/T6), testing & gates (T10). Out-of-scope items from the spec stay out.
- Type consistency: `mutations.Gained{UserId, MutationId, Rank, IsNew}` used identically in T1/T2/T3/T5; `buildMutationPayload` name consistent T5 test/impl; helper names `deepenFlourishText`/`revealCaption` consistent T2 test/impl.
- Deepen flourish keeps the existing `MutationMaxLevel` comparison (behavior-identical to today); per-spec `MaxRank` awareness deliberately not added (YAGNI).
