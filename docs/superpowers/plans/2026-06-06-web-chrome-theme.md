# Web Chrome Theme Pass — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Retheme the public web chrome (nav/header/footer, landing hero, info pages) from the `Press Start 2P` indigo-pixel look to a warm-dark cartographer palette with brass accents, cohesive with the leather web mapper.

**Architecture:** One shared stylesheet (`gomud.css`) drives every public page via `:root` tokens + component rules. Pages are Go `html/template` files that share `_header.html`/`_footer.html`. Reskin is CSS-first; only `index.html` gets new markup (the hero), and three info pages get light class touch-ups. No new image assets (background becomes a CSS gradient). Admin pages and the in-game client interior are out of scope.

**Tech Stack:** Go `html/template`, vanilla CSS. No build step — templates are parsed at server start (a template syntax error panics at boot, which is the verification gate).

**Spec:** `docs/superpowers/specs/2026-06-06-web-chrome-theme-design.md`. Visual source of truth: `docs/superpowers/specs/2026-06-06-web-chrome-mockups/`.

**Branch:** `feature/web-chrome-theme` (already created; spec committed).

**Verification note (all tasks):** there is no unit-test framework for CSS/templates. Each task's verification is: build, boot the server locally, and load the affected page(s) in a browser. Boot is the real gate — a template parse error fails server start. Static assets (`gomud.css`) are served live, so CSS-only changes need only a hard-refresh once the server is up. Suggested boot: `go run . 2>&1 | head -40` from repo root (watch for clean startup, no template panic), or reuse a running server on `:55555` and hard-refresh.

---

### Task 1: Theme foundation — rewrite `gomud.css` + `_header.html`

The entire public look lives in one stylesheet, so it is replaced as a single cohesive unit. Also drop the pixel font and swap the background to a CSS gradient in the shared header.

**Files:**
- Modify (full replace): `_datafiles/html/public/static/css/gomud.css`
- Modify: `_datafiles/html/public/_header.html` (lines ~8 and ~10-13)

- [ ] **Step 1: Replace `gomud.css` entirely with the new theme**

Write this exact content to `_datafiles/html/public/static/css/gomud.css`:

```css
:root {
  /* ── Surfaces ── */
  --bg-base: #191310;
  --panel-bg: #201913;
  --panel-bg-alt: #211a14;
  --panel-border: #3a2a18;
  --panel-border-soft: #2e251c;
  --bar-top: #241813;
  --bar-bottom: #1a110b;
  --gold-rule: #c9a86a;

  /* ── Ink / accents ── */
  --ink-gold: #c9a86a;
  --title-gold: #e8d2a0;
  --text-primary: #d2c3a4;
  --text-secondary: #9a8a6a;
  --text-tagline: #b9a983;
  --online-green: #7fae6a;

  /* ── Brass (shared with the mapper controls) ── */
  --brass-grad: radial-gradient(circle at 34% 26%, #f4dd92, #cb9f42 46%, #8a6620);
  --brass-border: #5e431a;
  --brass-text: #3b2a10;

  /* ── Tables ── */
  --table-row-alt: rgba(201, 168, 106, 0.05);

  /* ── Fonts ── */
  --font-serif: Georgia, 'Times New Roman', serif;
  --font-mono: 'DejaVu Sans Mono', monospace;

  /* Legacy table tokens kept so any stray references resolve */
  --table-border-color: #000;
  --table-cell-text-color: var(--text-primary);
  --table-header-text-color: var(--ink-gold);
}

body {
  margin: 0;
  padding: 0;
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  color: var(--text-primary);
  font-family: var(--font-serif);
  background-color: var(--bg-base);
  /* --background-image is set in _header.html (CSS gradient); falls back here */
  background: var(--background-image, radial-gradient(circle at 50% 22%, #251a12, #110b07)) fixed;
}

a { text-decoration: none; color: inherit; }

/* ── Header bar + MUD-name title ── */
header {
  background: linear-gradient(var(--bar-top), var(--bar-bottom));
  border-bottom: 2px solid var(--gold-rule);
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 24px;
}
.gomud-btn {
  font-family: var(--font-serif);
  font-style: italic;
  font-weight: bold;
  font-size: 1.6rem;
  letter-spacing: 1px;
  color: var(--title-gold);
  text-shadow: 0 1px 0 #000;
  padding: 4px 6px;
  background: none;
  box-shadow: none;
  border-radius: 4px;
  transition: color 0.2s ease;
}
.gomud-btn:hover { color: #fff; background: none; }

/* ── Brass component (buttons, nav pills, CTAs) ── */
.brass {
  display: inline-block;
  font-family: var(--font-serif);
  font-weight: bold;
  color: var(--brass-text);
  background: var(--brass-grad);
  border: 1px solid var(--brass-border);
  border-radius: 6px;
  text-shadow: 0 1px 0 rgba(255, 244, 206, 0.55);
  box-shadow: 0 2px 3px rgba(0, 0, 0, 0.5),
              inset 0 1px 1px rgba(255, 246, 212, 0.7),
              inset 0 -2px 3px rgba(74, 52, 16, 0.55);
  transition: filter 0.08s ease, transform 0.08s ease;
  cursor: pointer;
}
.brass:hover { filter: brightness(1.08); }
.brass:active {
  transform: translateY(1px);
  box-shadow: 0 1px 1px rgba(0, 0, 0, 0.4),
              inset 0 2px 4px rgba(74, 52, 16, 0.7),
              inset 0 -1px 1px rgba(255, 246, 212, 0.4);
}

/* ── Nav bar (brass pills) ── */
nav {
  background: linear-gradient(var(--bar-top), var(--bar-bottom));
  border-bottom: 1px solid var(--panel-border);
  padding: 10px 0;
}
.nav-container {
  margin: 0 auto;
  display: flex;
  justify-content: center;
  gap: 0.9rem;
}
.nav-container a {
  font-family: var(--font-serif);
  font-size: 0.95rem;
  color: var(--brass-text);
  background: var(--brass-grad);
  border: 1px solid var(--brass-border);
  border-radius: 5px;
  padding: 6px 14px;
  text-shadow: 0 1px 0 rgba(255, 244, 206, 0.55);
  box-shadow: 0 2px 3px rgba(0, 0, 0, 0.5),
              inset 0 1px 1px rgba(255, 246, 212, 0.7);
  transition: filter 0.08s ease;
}
.nav-container a:hover { filter: brightness(1.08); }
.nav-container a.selected {
  filter: brightness(0.95);
  box-shadow: inset 0 2px 4px rgba(74, 52, 16, 0.75),
              inset 0 -1px 1px rgba(255, 246, 212, 0.4);
}

/* ── Mobile nav toggle ── */
.nav-toggle { display: none; flex-direction: column; cursor: pointer; }
.nav-toggle div {
  width: 30px; height: 4px; border-radius: 2px;
  background: var(--ink-gold); margin: 5px;
}

/* ── Content + panels ── */
.content-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: flex-start;
  align-items: center;
  padding: 24px;
}
.content-container[data-path="/"] { justify-content: center; }

.overlay, .underlay {
  width: 90%;
  max-width: 1100px;
  background: var(--panel-bg);
  border: 1px solid var(--panel-border);
  border-radius: 8px;
  padding: 20px;
  text-align: center;
  box-shadow: inset 0 1px 0 rgba(201, 168, 106, 0.10),
              0 6px 16px rgba(0, 0, 0, 0.5);
}
.overlay { overflow-y: auto; }
.underlay { height: auto; max-height: 200px; }
.overlay h3, .underlay h3 {
  font-family: var(--font-serif);
  font-style: italic;
  color: var(--title-gold);
  margin: 0 0 12px;
}

/* ── Hero (landing) ── */
.hero { text-align: center; padding: 30px 26px; }
.hero-title {
  font-family: var(--font-serif);
  font-style: italic;
  font-weight: bold;
  color: var(--title-gold);
  font-size: 2.4rem;
  margin: 0;
  text-shadow: 0 2px 4px #000;
}
.hero-tagline {
  font-family: var(--font-serif);
  font-style: italic;
  color: var(--text-tagline);
  font-size: 1.05rem;
  margin: 8px 0 22px;
}
.hero-cta {
  font-size: 1.15rem;
  letter-spacing: 1px;
  padding: 14px 34px;
  text-decoration: none;
}
.hero-sub {
  margin-top: 14px;
  color: var(--text-secondary);
  font-family: var(--font-serif);
  font-size: 0.95rem;
}
.hero-sub b { color: var(--text-primary); }
.online-stat {
  display: inline-block;
  margin-top: 16px;
  padding: 6px 14px;
  border: 1px solid var(--panel-border);
  border-radius: 20px;
  background: #1d1610;
  color: var(--online-green);
  font-family: var(--font-serif);
  font-size: 0.9rem;
}

/* ── Footer ── */
footer {
  background: linear-gradient(var(--bar-bottom), #120c07);
  border-top: 1px solid var(--panel-border);
  text-align: center;
  color: var(--text-secondary);
  font-family: var(--font-serif);
  padding: 16px 0;
  font-size: 0.9rem;
}
footer a { color: var(--ink-gold); }
footer a:hover { text-decoration: underline; }

/* ── Tables ── */
table { width: 100%; border-collapse: collapse; }
table th {
  background: linear-gradient(var(--bar-top), var(--bar-bottom));
  color: var(--ink-gold);
  font-family: var(--font-serif);
  text-transform: uppercase;
  letter-spacing: 1px;
  padding: 0.7em;
  text-align: left;
  border-bottom: 1px solid var(--gold-rule);
}
table tr { background: var(--panel-bg-alt); }
table tr:nth-child(even) { background: var(--table-row-alt); }
table td {
  font-family: var(--font-serif);
  font-size: 1.05rem;
  padding: 0.4em 0.7em;
  text-align: left;
  color: var(--text-primary);
}
/* Config dump values read better monospaced */
table.config td { font-family: var(--font-mono); font-size: 0.95rem; }

pre, code { font-family: var(--font-mono); }

/* ── Responsive ── */
@media (max-width: 768px) {
  header { padding: 8px 12px; }
  .gomud-btn { font-size: 1.1rem; letter-spacing: 1px; }
  .hero-title { font-size: 1.7rem; }
  .nav-container {
    display: none;
    flex-direction: column;
    text-align: center;
    gap: 0.5rem;
    padding: 12px 0;
  }
  .nav-container a { display: block; padding: 10px; }
  .nav-toggle { display: flex; }
  .content-container { padding: 12px; }
  .overlay, .underlay { width: 95%; padding: 12px; }
  footer { font-size: 0.8rem; padding: 12px 0; }
  table th { padding: 0.5em; font-size: 0.8em; }
  table td { padding: 0.3em; font-size: 3.2vw; }
}

@media (max-width: 480px) {
  .gomud-btn { font-size: 0.95rem; }
  .overlay, .underlay { padding: 8px; }
  .hero-cta { padding: 12px 24px; font-size: 1rem; }
}
```

- [ ] **Step 2: Drop the pixel font and switch the background in `_header.html`**

Remove this line (the Google-font link, currently ~line 8):

```html
  <link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Press+Start+2P&display=swap" />
```

Replace the `--background-image` declaration in the inline `<style>` (currently ~lines 10-13):

```html
    :root {
      /* Setting CSS var here so we can prepend WebCDNLocation */
      --background-image: url('{{ .CONFIG.FilePaths.WebCDNLocation }}/static/images/web_bg.png') center center / cover no-repeat fixed;
    }
```

with:

```html
    :root {
      /* Warm-dark cartographer background (CSS gradient; no image asset) */
      --background-image: radial-gradient(circle at 50% 22%, #251a12, #110b07);
    }
```

- [ ] **Step 3: Verify**

Boot the server (`go run . 2>&1 | head -40` from repo root — watch for clean startup, no template panic). Load `http://localhost:<port>/` and `/online`. Confirm: warm-dark gradient background, no pixel font (serif everywhere), brass nav pills with the current page's pill showing the pressed `.selected` state, gold-headed table on `/online`. Hard-refresh if a server is already running (CSS is a live static asset).

- [ ] **Step 4: Commit**

```bash
git add _datafiles/html/public/static/css/gomud.css _datafiles/html/public/_header.html
git commit -m "feat(web): hybrid warm-dark + brass chrome theme (gomud.css, header)"
```

---

### Task 2: Landing hero (`index.html`)

**Files:**
- Modify: `_datafiles/html/public/index.html`

- [ ] **Step 1: Replace the Play-button block with the hero**

Write this exact content to `_datafiles/html/public/index.html`:

```html
{{template "header" .}}

  <div class="hero">
    <h1 class="hero-title">{{ .CONFIG.Server.MudName }}</h1>
    <p class="hero-tagline">A living world of use-based growth, factions, and consequence.</p>

    <a class="brass hero-cta" href="/webclient">&#9654; Play in Browser</a>

    <p class="hero-sub">or connect by telnet &mdash;
      <b>Port{{ if gt (len .CONFIG.Network.TelnetPort) 1 }}s{{ end }}: {{ join .CONFIG.Network.TelnetPort ", " }}</b>
    </p>

    <div>
      <span class="online-stat">&#9679;
        {{ len .STATS.OnlineUsers }} adventurer{{ if ne (len .STATS.OnlineUsers) 1 }}s{{ end }} online
      </span>
    </div>
  </div>

{{template "footer" .}}
```

Notes: `len`, `gt`, `join` are already used elsewhere in these templates; `ne` is a Go `text/template` builtin. If the boot test reports `ne` as undefined (custom funcmap stripped builtins — unlikely), fall back to `{{ if gt (len .STATS.OnlineUsers) 1 }}s{{ end }}` (drops the singular "1 adventurer" nicety only).

- [ ] **Step 2: Verify**

Boot/refresh and load `/`. Confirm: centered hero with serif-gold title (MUD name), tagline, a brass "▶ Play in Browser" button linking to `/webclient`, the telnet port line, and a "● N adventurers online" pill whose count matches `/online`. Click Play → lands on the web client.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/html/public/index.html
git commit -m "feat(web): landing hero — title, tagline, brass Play CTA, telnet + online count"
```

---

### Task 3: Info-page touch-ups (`online`, `viewconfig`, `404`)

These inherit the theme already; the only edits are a mono class on the config table and confirming the panels read well. `404.html` needs no change but is verified.

**Files:**
- Modify: `_datafiles/html/public/viewconfig.html` (add `class="config"` to the table)
- Modify: `_datafiles/html/public/online.html` (style the empty-state text)
- Verify (no change): `_datafiles/html/public/404.html`

- [ ] **Step 1: `viewconfig.html` — mono config values**

Change the opening table tag (currently `<table>` on line 6) to:

```html
        <table class="config">
```

(Leave everything else in the file unchanged.)

- [ ] **Step 2: `online.html` — themed empty state**

Replace the `{{else}}` empty branch (currently `None.` on line 26) so it reads:

```html
        {{else}}
            <p class="hero-sub">No adventurers are online right now.</p>
        {{end}}
```

(Leave the table and surrounding `.overlay` markup unchanged.)

- [ ] **Step 3: Verify**

Boot/refresh and load `/viewconfig` (Name column serif, Value column monospace, gold headers, alt-row tint), `/online` with zero and with ≥1 users if possible (themed empty state vs table), and `/404` (e.g. load `/nope`) — confirm the `.underlay` panel renders themed.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/html/public/viewconfig.html _datafiles/html/public/online.html
git commit -m "feat(web): theme touch-ups for config table + online empty state"
```

---

### Task 4: Full-site smoke + upstream scan + wrap-up

**Files:**
- None (verification + optional notes only)

- [ ] **Step 1: Full boot smoke across every public surface**

Boot the server fresh. Load and eyeball, watching for any template panic at startup and any unstyled/regressed element:
- `/` (hero)
- `/online` (table + empty state)
- `/viewconfig` (config table)
- `/404` (via a bad path)
- `/webclient` (embedded view) — confirm the surrounding nav/header/footer chrome is themed (the client *interior* is intentionally untouched this cycle)
- Mobile width (narrow the window < 768px): confirm the hamburger `.nav-toggle` appears and `toggleMenu()` still shows/hides `.nav-container`.

- [ ] **Step 2: Upstream cherry-pick scan (optional, time-boxed)**

Scan upstream GoMud (`GoMudEngine/GoMud`) for clean landing/nav/web improvements that fit the hybrid theme. Cherry-pick only; NEVER push to upstream. If nothing fits cleanly, note it and move on — not a blocker.

- [ ] **Step 3: Confirm no leftover references to removed assets**

Grep the public templates/CSS for `Press Start`, `web_bg.png`, and `play-button` to confirm nothing still references the dropped font/asset/class:

```bash
grep -rn "Press Start\|web_bg.png\|play-button\|btn_play" _datafiles/html/public/
```

Expected: no matches in `_header.html`, `gomud.css`, or `index.html`. (`btn_play.png` may remain unused on disk — that's fine; just no template should reference it.)

- [ ] **Step 4: Final commit (if Step 2/3 produced changes)**

```bash
git add -- <only the specific files changed>
git commit -m "chore(web): chrome theme smoke fixes + upstream scan notes"
```

---

## Self-review

- **Spec coverage:** theme tokens (Task 1), nav/header/footer (Task 1), landing hero (Task 2), tables/info pages (Tasks 1+3), background gradient (Task 1), scope boundaries respected (admin + client interior untouched), upstream scan (Task 4) — all covered.
- **Placeholders:** none. Tagline is real copy (flagged editable in the spec). Every CSS/HTML block is complete.
- **Type/name consistency:** `.brass`, `.hero*`, `.online-stat`, `.config`, and the `:root` token names are defined in Task 1's `gomud.css` and used verbatim in Tasks 2-3. `--bar-top`/`--bar-bottom` used consistently across header/nav/footer/table.
- **Template funcs:** `len`/`gt`/`join` confirmed in existing templates; `ne` builtin with a documented fallback.
```
