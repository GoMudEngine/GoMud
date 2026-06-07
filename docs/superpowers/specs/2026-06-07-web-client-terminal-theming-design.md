# Web Client Terminal Theming — Design

**Date:** 2026-06-07
**Status:** Approved (brainstorm) — ready for implementation plan
**Author:** brainstormed with the visual companion

## Goal

Theme the in-game web client's game-output terminal (xterm.js) to the antique
tooled-leather aesthetic: a warm-dark background with a harmonized 16-color ANSI
palette, and **IBM Plex Mono** as the typeface. Today the terminal has no
`theme` and no `fontFamily`, so xterm falls back to cold pure-black with the
stock Tango palette and Courier New — a cold rectangle that clashes with the
brass-and-leather panels around it.

This is the **terminal-theming** project, the user-prioritized next step of the
web overhaul (see [[project_web_overhaul_sequence]]), landing before
sub-project #3 (the Triggers/Tick/Macros card). Like the rest of the overhaul,
it stays **parked locally — NOT pushed to prod**.

## Context

- The terminal is constructed in `_datafiles/html/public/webclient-pure.html`
  (~line 328): `new window.Terminal({ cols:80, rows:60, cursorBlink:true,
  fontSize:20 })` — **no `theme`, no `fontFamily`**. It's opened on
  `document.getElementById('terminal')` (~line 336).
- Font sizing is handled by `resizeTerminal()` (~line 342): it runs
  `fitAddon.fit()` and shrinks/grows `fontSize` (10–20) until ~80 columns fit
  the feed panel. It is called on load, on `window.resize`, and from a
  `ResizeObserver` on `#panel-feed`. **All MUD content is fixed-80-column**
  (room text + inline ASCII minimap), so glyph width must be correct before
  sizing is computed.
- xterm 4.19 is vendored at `static/js/xterm.4.19.0.js`; its stylesheet
  `static/css/xterm.css` **hardcodes `#000`** in two places the JS `theme`
  object does NOT reach: `.xterm .composition-view` (`background:#000`,
  line 81) and `.xterm .xterm-viewport` (`background-color:#000`, line 95 — the
  surface behind the scrollbar and below short content).
- The leather palette tokens live in `static/css/gomud.css` `:root`:
  `--panel-bg #201913`, warm-dark base `#191310`, `--ink-gold #c9a86a`,
  title-gold `#e8d2a0`, `--ink-deep #2b231d`. Dashboard panel styling is in
  `static/css/dashboard.css`.
- Body/chrome type is Georgia serif. The terminal **must remain monospace**
  (fixed character grid) — so the font choice is "which monospace pairs with
  the serif chrome," not "match the serif." Chosen: IBM Plex Mono (humanist
  monospace with subtle serifs, designed to pair with serif body text, legible
  at the small auto-fit sizes).

## Locked decisions (from the brainstorm)

1. **Palette = option B (harmonized leather).** Warm-dark background + a full
   16-color retint where each hue keeps its identity but is pulled toward the
   warm/parchment world, so the terminal reads as one piece with the UI.
2. **Font = IBM Plex Mono**, self-hosted (not a runtime CDN dependency), with
   Courier New → generic monospace as fallback.
3. **Client-side only.** No Go/server/GMCP changes; no rebuild. Pure
   HTML/CSS/JS edits to the web client assets.
4. **Scope = the one game terminal.** No per-message recoloring, no change to
   the input box styling beyond what the terminal theme already covers.

Visual source of truth (mockups):
`.superpowers/brainstorm/.../content/terminal-palette.html` and
`terminal-font.html` (companion session — not committed; palette + font values
are reproduced below as the authority).

## Part A — xterm `theme` object (the B palette)

Add a `theme` object to the `new window.Terminal({…})` config in
`webclient-pure.html`. Surface + cursor + selection colors, then the 16 ANSI
entries. Values hardcoded (xterm theme is JS, fixed at construct time) with a
comment cross-referencing the `gomud.css` leather tokens so they don't drift.

```js
theme: {
  // Leather palette (harmonized). Cross-ref gomud.css :root tokens.
  background:    '#191310',  // warm-dark base
  foreground:    '#e0d2b2',  // parchment ink
  cursor:        '#e8d2a0',  // title-gold
  cursorAccent:  '#191310',
  selection:     'rgba(201,168,106,0.30)', // ink-gold @ 30%
  black:   '#3a2f25', brightBlack:   '#6b5a48',
  red:     '#d6694e', brightRed:     '#e8745a',
  green:   '#8a9a4e', brightGreen:   '#b3c06a',
  yellow:  '#d9a441', brightYellow:  '#f0cf72',
  blue:    '#88a9cf', brightBlue:    '#aecce8',  // brightened post-smoke for splash legibility
  magenta: '#a8678f', brightMagenta: '#c890b0',
  cyan:    '#5f9a93', brightCyan:    '#84c0b6',
  white:   '#d8c8a8', brightWhite:   '#f2e6c8'
}
```

These exact values are the approved B palette from the mockup. They are easy to
nudge during the smoke if any color reads muddy — the retint amount is the one
thing flagged for in-game confirmation.

## Part B — IBM Plex Mono (self-hosted) + font-aware refit

**B1 — Self-host the font.** IBM Plex Mono is SIL OFL 1.1 (open). Place the
woff2 files under a new `static/fonts/` directory and declare them via
`@font-face` in `dashboard.css`:

- `IBMPlexMono-Regular.woff2` (weight 400)
- `IBMPlexMono-Bold.woff2` (weight 700)

```css
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

Two weights only — the terminal uses 400 for normal text and 700 for bold/bright
emphasis; medium is not needed. Include the OFL license text alongside the font
files (e.g. `static/fonts/IBMPlexMono-LICENSE.txt`).

**B2 — Point xterm at it.** Add to the `Terminal` config:

```js
fontFamily: '"IBM Plex Mono", "Courier New", Courier, monospace',
```

(Leave `fontWeightBold` at xterm's default of `'bold'` → 700.)

**B3 — Refit after the font loads (critical for the 80-col grid).** xterm
measures glyph width when it opens and on each `fit()`. If `resizeTerminal()`
runs before IBM Plex Mono has loaded, sizing is computed against the Courier
fallback and the grid is wrong until the next fit. After `term.open(...)` and
the initial sizing, wait for the font then force a glyph re-measure + refit:

```js
if (document.fonts && document.fonts.load) {
  Promise.all([
    document.fonts.load('400 16px "IBM Plex Mono"'),
    document.fonts.load('700 16px "IBM Plex Mono"')
  ]).then(function () {
    // Re-assigning fontFamily forces xterm to rebuild its glyph atlas with the
    // now-loaded font; then re-fit so 80 columns are measured correctly.
    term.setOption('fontFamily', '"IBM Plex Mono", "Courier New", Courier, monospace');
    resizeTerminal();
  }).catch(function () { resizeTerminal(); });
}
```

This is additive — the existing `resizeTerminal()` calls stay as-is; the
font-ready handler just runs one more refit once glyphs are final.

## Part C — Plug the two cold-black CSS leaks

The JS `theme` object does not control the viewport surface or the IME
composition box; both hardcode `#000` in vendored `xterm.css`. Override them in
`dashboard.css` (NOT by editing vendored `xterm.css`, so an xterm upgrade can't
clobber the fix). Use `#terminal …` selectors so the override wins by
specificity regardless of stylesheet load order:

```css
#terminal .xterm-viewport   { background-color: #191310; }
#terminal .composition-view { background: #2b231d; color: #e8d2a0; }
```

(`#terminal` is the container xterm is opened on.) Optionally style the
viewport scrollbar to brass/leather in the same block for polish, but the
`#000` removal is the required part.

## Scope / boundaries

- **In:** the xterm `theme` object (Part A), self-hosted IBM Plex Mono +
  font-aware refit (Part B), the two CSS leak overrides (Part C).
- **Out:** server/GMCP changes (none); per-message/category recoloring; font
  changes to the rest of the chrome (stays Georgia); the Triggers card (#3).
- **No server rebuild.** Verification is browser-only.

## Acceptance / verification

- Web client loads with no console errors; the terminal background is warm-dark
  (not cold black), text is parchment-toned, and the 16 ANSI game colors
  (title, exits, speech, combat-hit, gold, prompt bars) render in the B palette
  and stay readable/distinct.
- Terminal text is IBM Plex Mono (confirm the font actually loaded — not the
  Courier fallback — via devtools computed style on `.xterm`).
- The 80-column grid still aligns after font load: room text wraps correctly
  and the inline ASCII minimap is not misaligned (resize the window / pop-out
  the feed panel to exercise `resizeTerminal()`).
- No cold-black flashes behind the scrollbar or below short content; the IME
  composition box (if triggered) is leather-toned.
- The font is served from the app's own `static/fonts/` (no request to a Google
  CDN at runtime — check the network tab).

## Risks / open items

- **Font load race / FOUT.** Mitigated by `font-display: swap` (renders in
  Courier until Plex is ready) + the `document.fonts.load(...)` refit. Worst
  case is a brief Courier flash before the refit; acceptable.
- **Retint amount.** The exact 16 hex values are the one thing to eyeball
  in-game; nudging them is a trivial follow-up edit if any color reads muddy.
- **Stylesheet load order.** Avoided by using `#terminal`-prefixed selectors
  (higher specificity than xterm.css's `.xterm .xterm-viewport`).
- **woff2 acquisition.** The plan must fetch the two IBM Plex Mono woff2 files
  from an authoritative OFL source (IBM Plex GitHub release / Google Fonts) and
  commit them with the license; no runtime CDN.
