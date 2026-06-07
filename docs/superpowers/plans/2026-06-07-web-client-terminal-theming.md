# Web Client Terminal Theming Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Theme the in-game web client's xterm terminal to the leather aesthetic — harmonized 16-color ANSI palette + warm-dark surface + self-hosted IBM Plex Mono — with a font-aware refit so the 80-column grid stays aligned.

**Architecture:** Pure client-side edits to three web assets (`webclient-pure.html`, `dashboard.css`, a new `static/fonts/` dir). No Go/server/GMCP changes, no server rebuild, no front-end build step (assets are served raw). Parked locally — NOT pushed to prod.

**Tech Stack:** xterm.js 4.19 (`theme` config object + `setOption`), `@font-face` self-hosted woff2 (IBM Plex Mono, SIL OFL 1.1), `document.fonts.load` for font-ready refit, CSS specificity overrides.

---

## File Structure

- **Create:** `_datafiles/html/public/static/fonts/IBMPlexMono-Regular.woff2` — self-hosted font (weight 400)
- **Create:** `_datafiles/html/public/static/fonts/IBMPlexMono-Bold.woff2` — self-hosted font (weight 700)
- **Create:** `_datafiles/html/public/static/fonts/IBMPlexMono-LICENSE.txt` — SIL OFL 1.1 license text
- **Modify:** `_datafiles/html/public/static/css/dashboard.css` — `@font-face` declarations (Task 2) + `#terminal` cold-black overrides (Task 5)
- **Modify:** `_datafiles/html/public/webclient-pure.html` — xterm `theme` object (Task 3) + `fontFamily` and font-ready refit (Task 4)

Reference: spec at `docs/superpowers/specs/2026-06-07-web-client-terminal-theming-design.md`.

**Verification note:** This is front-end HTML/CSS/JS in a Go repo with no JS test runner. "Tests" here are concrete structural checks (file exists, valid woff2 magic bytes, grep the inserted code) plus a final browser smoke (Task 6). Each task commits its own change. `git add` ONLY the named files — the working tree has unrelated uncommitted world-state files that must not be staged.

---

### Task 1: Self-host the IBM Plex Mono woff2 files + license

**Files:**
- Create: `_datafiles/html/public/static/fonts/IBMPlexMono-Regular.woff2`
- Create: `_datafiles/html/public/static/fonts/IBMPlexMono-Bold.woff2`
- Create: `_datafiles/html/public/static/fonts/IBMPlexMono-LICENSE.txt`

- [ ] **Step 1: Create the fonts directory**

Run (Bash tool):
```bash
mkdir -p "_datafiles/html/public/static/fonts"
```

- [ ] **Step 2: Download the two woff2 files + license from an authoritative OFL source (jsDelivr npm mirror of `@ibm/plex-mono`)**

Run (Bash tool):
```bash
base="https://cdn.jsdelivr.net/npm/@ibm/plex-mono@latest/fonts/complete/woff2"
curl -fL -o "_datafiles/html/public/static/fonts/IBMPlexMono-Regular.woff2" "$base/IBMPlexMono-Regular.woff2"
curl -fL -o "_datafiles/html/public/static/fonts/IBMPlexMono-Bold.woff2"    "$base/IBMPlexMono-Bold.woff2"
curl -fL -o "_datafiles/html/public/static/fonts/IBMPlexMono-LICENSE.txt"   "https://cdn.jsdelivr.net/npm/@ibm/plex-mono@latest/LICENSE.txt"
```

If any URL 404s, retry with a pinned major (`@ibm/plex-mono@6` in place of `@latest`); if `LICENSE.txt` is missing from the package root, fetch the canonical OFL from `https://raw.githubusercontent.com/IBM/plex/master/LICENSE.txt`.

- [ ] **Step 3: Verify the woff2 files are valid (magic bytes = `wOF2`, non-trivial size)**

Run (Bash tool):
```bash
for f in Regular Bold; do
  p="_datafiles/html/public/static/fonts/IBMPlexMono-$f.woff2"
  printf '%s: ' "$f"; head -c4 "$p"; printf ' '; wc -c < "$p"
done
```
Expected: each line begins with `wOF2` and a byte count over ~20000 (e.g. `Regular: wOF2 34xxx`). If a file shows HTML/text or a tiny size, the download failed — re-fetch per Step 2's fallback.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/html/public/static/fonts/IBMPlexMono-Regular.woff2 _datafiles/html/public/static/fonts/IBMPlexMono-Bold.woff2 _datafiles/html/public/static/fonts/IBMPlexMono-LICENSE.txt
git commit -m "feat(web): self-host IBM Plex Mono woff2 (OFL) for terminal

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: Declare the `@font-face` rules in dashboard.css

**Files:**
- Modify: `_datafiles/html/public/static/css/dashboard.css` (append at end of file)

- [ ] **Step 1: Append the two `@font-face` blocks**

Add to the end of `dashboard.css`:
```css

/* ── IBM Plex Mono (self-hosted, OFL) — used by the game terminal ────────── */
@font-face {
  font-family: 'IBM Plex Mono';
  font-style: normal; font-weight: 400; font-display: swap;
  src: url('../fonts/IBMPlexMono-Regular.woff2') format('woff2');
}
@font-face {
  font-family: 'IBM Plex Mono';
  font-style: normal; font-weight: 700; font-display: swap;
  src: url('../fonts/IBMPlexMono-Bold.woff2') format('woff2');
}
```

- [ ] **Step 2: Verify the rules and the relative paths resolve to the files from Task 1**

Run (Grep tool): pattern `IBMPlexMono-(Regular|Bold)\.woff2` in `_datafiles/html/public/static/css/dashboard.css`, output_mode `content`.
Expected: both `url('../fonts/IBMPlexMono-Regular.woff2')` and `...Bold.woff2` present. (`dashboard.css` is in `static/css/`, so `../fonts/` resolves to `static/fonts/` — where Task 1 wrote the files.)

- [ ] **Step 3: Commit**

```bash
git add _datafiles/html/public/static/css/dashboard.css
git commit -m "feat(web): @font-face IBM Plex Mono for the game terminal

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: Add the xterm `theme` object (B leather palette)

**Files:**
- Modify: `_datafiles/html/public/webclient-pure.html:328-333` (the `new window.Terminal({...})` config)

- [ ] **Step 1: Add the `theme` object to the Terminal config**

Replace:
```js
        const term = new window.Terminal({
            cols: 80,
            rows: 60,
            cursorBlink: true,
            fontSize: 20
        });
```
with:
```js
        const term = new window.Terminal({
            cols: 80,
            rows: 60,
            cursorBlink: true,
            fontSize: 20,
            // Leather palette (harmonized "option B"). Cross-ref gomud.css :root
            // tokens: --panel-bg #201913, warm-dark #191310, --ink-gold #c9a86a,
            // title-gold #e8d2a0, --ink-deep #2b231d.
            theme: {
                background:   '#191310',
                foreground:   '#e0d2b2',
                cursor:       '#e8d2a0',
                cursorAccent: '#191310',
                selection:    'rgba(201,168,106,0.30)',
                black:   '#3a2f25', brightBlack:   '#6b5a48',
                red:     '#d6694e', brightRed:     '#e8745a',
                green:   '#8a9a4e', brightGreen:   '#b3c06a',
                yellow:  '#d9a441', brightYellow:  '#f0cf72',
                blue:    '#6f8fae', brightBlue:    '#93b0c9',
                magenta: '#a8678f', brightMagenta: '#c890b0',
                cyan:    '#5f9a93', brightCyan:    '#84c0b6',
                white:   '#d8c8a8', brightWhite:   '#f2e6c8'
            }
        });
```

- [ ] **Step 2: Verify the theme is present and well-formed**

Run (Grep tool): pattern `theme:|background:\s*'#191310'|brightWhite:\s*'#f2e6c8'` in `webclient-pure.html`, output_mode `content`.
Expected: the `theme:` key, the `#191310` background, and the `#f2e6c8` brightWhite all present (confirms the block opened and closed around all 16 colors).

- [ ] **Step 3: Commit**

```bash
git add _datafiles/html/public/webclient-pure.html
git commit -m "feat(web): leather ANSI palette theme for the xterm terminal

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: Point xterm at IBM Plex Mono + refit once the font loads

**Files:**
- Modify: `_datafiles/html/public/webclient-pure.html` — add `fontFamily` to the Terminal config (Task 3 block); add a font-ready refit just after the initial sizing calls (~line 377, after `resizeTerminal();`)

- [ ] **Step 1: Add `fontFamily` to the Terminal config**

In the `new window.Terminal({...})` config edited in Task 3, add a `fontFamily` line directly under `fontSize: 20,` (before the `theme:` comment):
```js
            fontSize: 20,
            fontFamily: '"IBM Plex Mono", "Courier New", Courier, monospace',
```

- [ ] **Step 2: Add the font-ready refit after the initial sizing**

Find the initial sizing calls (~lines 376-377):
```js
        requestAnimationFrame(resizeTerminal); // fit once after first grid paint
        resizeTerminal();
```
Insert immediately AFTER them:
```js

        // The 80-col grid is measured from glyph width, so it must be computed
        // against the real font, not the Courier fallback. font-display:swap
        // renders in Courier until IBM Plex Mono loads; once it does, re-assign
        // fontFamily (forces xterm to rebuild its glyph atlas) and refit.
        if (document.fonts && document.fonts.load) {
            Promise.all([
                document.fonts.load('400 16px "IBM Plex Mono"'),
                document.fonts.load('700 16px "IBM Plex Mono"')
            ]).then(function () {
                term.setOption('fontFamily', '"IBM Plex Mono", "Courier New", Courier, monospace');
                resizeTerminal();
            }).catch(function () { resizeTerminal(); });
        }
```

- [ ] **Step 3: Verify both edits are present**

Run (Grep tool): pattern `fontFamily:|document\.fonts\.load\('700 16px` in `webclient-pure.html`, output_mode `content`.
Expected: the `fontFamily:` config line AND the `document.fonts.load('700 16px "IBM Plex Mono"')` refit line both present.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/html/public/webclient-pure.html
git commit -m "feat(web): IBM Plex Mono on terminal + font-ready refit for 80-col grid

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: Override the two cold-`#000` leaks from xterm.css

**Files:**
- Modify: `_datafiles/html/public/static/css/dashboard.css` (append after the `@font-face` block from Task 2)

- [ ] **Step 1: Append the `#terminal` overrides**

`xterm.css` hardcodes `background-color:#000` on `.xterm .xterm-viewport` (line 95) and `background:#000` on `.xterm .composition-view` (line 81). The JS `theme` does not reach these. Append to `dashboard.css`:
```css

/* Plug xterm.css's hardcoded cold-black surfaces (viewport behind the
   scrollbar / below short content, and the IME composition box). #terminal
   prefix wins on specificity over xterm.css's `.xterm .xterm-viewport`,
   regardless of stylesheet load order, and survives an xterm upgrade. */
#terminal .xterm-viewport   { background-color: #191310; }
#terminal .composition-view { background: #2b231d; color: #e8d2a0; }
```

- [ ] **Step 2: Verify the overrides are present**

Run (Grep tool): pattern `#terminal \.xterm-viewport|#terminal \.composition-view` in `dashboard.css`, output_mode `content`.
Expected: both selectors present with warm-tone backgrounds (`#191310`, `#2b231d`).

- [ ] **Step 3: Commit**

```bash
git add _datafiles/html/public/static/css/dashboard.css
git commit -m "fix(web): warm-tone the xterm viewport + IME box (remove cold black)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: Browser smoke verification

**Files:** none (verification only)

This task is a manual browser smoke against a locally-booted server — there is no automated front-end test harness. Boot the server, open `/webclient`, and confirm each acceptance item. (The reviewer/operator performs the visual checks; the dev server need not be rebuilt since only static/template assets changed — a hard refresh picks them up.)

- [ ] **Step 1: Serve the client and hard-refresh**

The Go server serves these assets directly. If a server is already running, a hard refresh (Ctrl+Shift+R) of `/webclient` is enough. Otherwise start one: `go run . ` (or the project's normal run command) and open the web client URL.

- [ ] **Step 2: Walk the acceptance checklist (from the spec)**

Confirm, with devtools open:
- No console errors on load.
- Terminal background is warm-dark (not cold black); foreground is parchment-toned; the ANSI game colors (room title, exits, speech, combat-hit, gold, prompt bars) render in the leather palette and stay readable/distinct.
- Computed style on `.xterm` shows `font-family` resolving to **IBM Plex Mono** (not the Courier fallback). Network tab shows the woff2 fetched from the app's own `static/fonts/` — **no request to a Google/CDN host** at runtime.
- 80-column grid still aligns after font load: room text wraps correctly and the inline ASCII minimap is not misaligned. Resize the window and pop-out the feed panel to exercise `resizeTerminal()`; the grid stays aligned.
- No cold-black flash behind the scrollbar or below short content.

- [ ] **Step 3: Record the result**

If any color reads muddy or any check fails, note it for a follow-up nudge (the 16 hex values in Task 3 are trivially adjustable). If all pass, the feature is ready for finishing-a-development-branch.

---

## Self-Review

**Spec coverage:** Part A (theme) → Task 3. Part B1 (self-host) → Task 1+2. Part B2 (fontFamily) → Task 4 Step 1. Part B3 (font-ready refit) → Task 4 Step 2. Part C (CSS leaks) → Task 5. Acceptance/verification → Task 6. All spec sections covered.

**Placeholder scan:** No TBD/TODO; every code step shows the exact code; the download fallback is concrete (pinned major + raw GitHub OFL).

**Type/identifier consistency:** `resizeTerminal` (existing fn) reused verbatim in Task 4; `term` and `#terminal` container consistent with webclient-pure.html:336; the `fontFamily` string is identical in the config (Task 4 Step 1) and the `setOption` refit (Task 4 Step 2); woff2 filenames identical across Task 1 (created), Task 2 (`@font-face src`), and Task 3/4 are unaffected.
