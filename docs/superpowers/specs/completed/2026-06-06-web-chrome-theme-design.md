# Web Chrome Theme Pass — Design

**Date:** 2026-06-06
**Status:** Approved (brainstorm) — ready for implementation plan
**Author:** brainstormed with the visual companion

## Goal

Retheme the public web **chrome** — nav/header/footer, landing page, and
info pages — away from the retro `Press Start 2P` indigo-pixel look toward a
warm-dark cartographer palette that is cohesive with the antique tooled-leather
web mapper shipped on 2026-06-06. This is the first of two sequenced web
efforts; the in-game client interior (terminal + WinBox panels → a north-star
three-column dashboard) is a separate follow-on cycle.

## Context — current state

- **`gomud.css`** (`_datafiles/html/public/static/css/`) defines the whole
  public look via `:root` tokens: indigo header `#0e0e20`, nav `#1a1a3e`,
  buttons `#3b3b78` with hard pixel box-shadows, body font `Press Start 2P`,
  body background `web_bg.png`.
- **`_header.html`** renders `<header>` (MUD-name `.gomud-btn` + mobile
  `.nav-toggle`) and `<nav>` (`.nav-container` of links ranged from `.NAV`).
  It also injects the `--background-image` CSS var from config and links the
  `Press Start 2P` Google font.
- **`_footer.html`** renders `<footer>` + the `toggleMenu()` mobile script.
- **Public pages:** `index.html` (Play-button image + telnet-port line),
  `online.html` (Users-Online table), `viewconfig.html` (config dump),
  `404.html`. The game client `webclient.html` iframes `webclient-pure.html`
  and is wrapped by this same chrome.
- **Nav items** (`internal/web/web.go:142`): `Home /`, `Who's Online /online`,
  `Web Client /webclient`, `See Configuration /viewconfig`, plus any plugin
  nav links appended at runtime. **There is no standalone leaderboards page**
  (leaderboards are in-game only).

## Locked decisions (from the brainstorm)

1. **Theme direction: Hybrid** — a warm-dark dashboard base (à la
   `sample_mud_frontend.png`) with leather/brass nav and serif-gold headings
   layered on top. Chosen over full-leather (too ornate for content pages) and
   plain warm-dark (less cohesive with the mapper).
2. **Landing page: Hero** — a real front door (title + tagline + prominent
   Play CTA + telnet line + live online count), chosen over a faithful
   minimal reskin.

Visual source of truth: `docs/superpowers/specs/2026-06-06-web-chrome-mockups/`
(`chrome-theme.html`, `landing.html`).

## Theme system — palette & type tokens

These become the new `:root` custom properties in `gomud.css`. Values are the
exact ones validated in the mockups and shared with the mapper's brass/leather
recipe.

### Surfaces
| Token | Value | Use |
|-------|-------|-----|
| `--bg-base` | `#191310` | page background base (warm near-black) |
| `--panel-bg` | `#201913` | content panels (`.overlay`, `.underlay`, cards) |
| `--panel-bg-alt` | `#211a14` | alt panel / table cell base |
| `--panel-border` | `#3a2a18` | panel borders |
| `--panel-border-soft` | `#2e251c` | subtle inner borders / row rules |
| `--bar-gradient` | `linear-gradient(#241813, #1a110b)` | header + nav bar |
| `--gold-rule` | `#c9a86a` | the gold underline beneath the bar |

### Ink / accents
| Token | Value | Use |
|-------|-------|-----|
| `--ink-gold` | `#c9a86a` | accents, table headers, links |
| `--title-gold` | `#e8d2a0` | headings / MUD-name title |
| `--text-primary` | `#d2c3a4` | body text |
| `--text-secondary` | `#9a8a6a` | muted/secondary text |
| `--text-tagline` | `#b9a983` | hero tagline |
| `--online-green` | `#7fae6a` | "online now" / connected dot |

### Brass (buttons / CTAs / nav pills) — shared with the mapper controls
- background: `radial-gradient(circle at 34% 26%, #f4dd92, #cb9f42 46%, #8a6620)`
- border: `1px solid #5e431a`
- text: `#3b2a10` (engraved-dark), `text-shadow: 0 1px 0 rgba(255,244,206,.55)`
- shadow: `0 2px 3px rgba(0,0,0,.5), inset 0 1px 1px rgba(255,246,212,.7), inset 0 -2px 3px rgba(74,52,16,.55)`
- `:hover` → `filter: brightness(1.08)`; `:active` → `translateY(1px)` + inverted inset shadow
- Encapsulate as a `.brass` class so buttons, nav pills, and the Play CTA all
  share one definition.

### Typography
- Drop the `Press Start 2P` Google-font link from `_header.html`.
- Headings + titles: `Georgia, 'Times New Roman', serif` (italic bold for the
  largest titles, e.g. MUD name and hero title).
- Body / prose / nav: same Georgia serif stack for cohesion.
- Monospace (`'DejaVu Sans Mono', monospace`) retained ONLY for the config
  dump (`viewconfig`) and the xterm terminal — not touched here.

### Background
- Replace the `web_bg.png` dependency with a pure-CSS warm radial gradient over
  `--bg-base` (e.g. `radial-gradient(circle at 50% 22%, #251a12, #110b07)`),
  optionally a faint CSS grain via a tiny repeating overlay. No new image asset.
- The `--background-image` var set in `_header.html` is overridden to this
  gradient (keep the var mechanism so config can still override if desired).

## File-by-file changes

### `gomud.css` (primary)
- Rewrite `:root` to the token table above.
- Restyle: `body` (bg gradient + Georgia), `header`, `.gomud-btn` (serif-gold
  title treatment), `nav`/`.nav-container a` (brass pills + brass-pressed
  `.selected`/`:hover`), `footer`, `table`/`th`/`tr:nth-child(even)`/`td`
  (gold headers, faint alt-row tint, serif), `.overlay`/`.underlay`
  (warm-dark panels), `.content-container`, `.play-button`.
- Add `.brass` and hero classes (`.hero`, `.hero-title`, `.hero-tagline`,
  `.hero-cta`, `.hero-sub`, `.online-stat`).
- Preserve all existing responsive `@media` breakpoints; update sizes/letter-
  spacing as needed for the new (non-pixel) font.

### `_header.html`
- Remove the `Press Start 2P` `<link>`.
- Override `--background-image` to the CSS gradient.
- No structural nav change required (links still range from `.NAV`); styling is
  driven by `gomud.css`. The MUD-name `.gomud-btn` becomes the serif-gold title.

### `index.html` (landing → hero)
- Replace the bare Play-button block with the hero structure:
  - `.hero-title` = `{{ .CONFIG.Server.MudName }}` (serif gold).
  - `.hero-tagline` = a one-line tagline (placeholder copy below; trivially
    editable).
  - `.hero-cta` = a `.brass` "▶ Play in Browser" link to `/webclient`.
  - `.hero-sub` = telnet host + `{{ join .CONFIG.Network.TelnetPort ", " }}`
    (pluralize "Port(s)" as the current template does).
  - `.online-stat` = `● {{ len .STATS.OnlineUsers }} adventurers online`
    (singular/plural handled).
- Keep the existing `btn_play.png` asset only if we want an image CTA; default
  is the text `.brass` button (no asset dependency).

### `online.html`, `viewconfig.html`, `404.html`
- Inherit the theme automatically via the restyled `table`/`.overlay` rules.
- Light per-page touch-ups only (heading classes, spacing). `viewconfig`'s
  config dump stays monospace.

### `_footer.html`
- No structural change; footer styling inherited. Verify the mobile
  `toggleMenu()` still toggles `.nav-container` correctly under the new styles.

## Scope / boundaries

- **In scope:** all public pages, nav/header/footer, the shared `gomud.css`
  theme.
- **Out of scope (next cycle):** the in-game client *interior*
  (`webclient-pure.html` terminal + WinBox panels → north-star dashboard). Only
  the surrounding nav chrome (shared `_header.html`) changes here, which fixes
  the client's wrapper for free.
- **Out of scope:** admin pages (`_datafiles/html/admin/` — separate
  `styles.css`).
- **Upstream cherry-picks:** scan upstream GoMud for clean landing/nav
  improvements and fold in only if they fit the hybrid theme; never push to
  upstream. Not a blocker.

## Acceptance / verification

- Every public page renders with no `Press Start 2P` font and no `web_bg.png`
  dependency; build + boot the server and load `/`, `/online`, `/viewconfig`,
  `/404`, and `/webclient` (chrome only) to confirm.
- Nav links show brass pills with a clear `selected` state on the current page.
- Landing hero shows title, tagline, Play CTA (links to `/webclient`), telnet
  line, and a live online count that matches `/online`.
- Tables (online/viewconfig) show gold headers + alternating-row tint, readable
  at mobile breakpoints.
- No regression to the mobile nav-toggle behavior.

## Open items

- **Tagline copy** — placeholder: *"A living world of use-based growth,
  factions, and consequence."* Final wording is the user's call; it lives in
  `index.html` and is a one-line edit.
