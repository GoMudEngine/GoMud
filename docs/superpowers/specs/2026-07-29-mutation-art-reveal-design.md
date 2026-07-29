# Mutation Art + Reveal (Path-to-1.0 §5d) — Design

**Date:** 2026-07-29
**Status:** Approved (brainstorm session 2026-07-29)

## Goal

Make mutation acquisition feel like the milestone it is. Two independent
pieces: **(A)** a per-mutation emblem art batch, and **(B)** a reveal event
that upgrades the acquisition moment on both web and terminal clients.

## Decisions (locked with user)

| Question | Decision |
|----------|----------|
| Art subject | **Symbolic emblem/sigil** per mutation (crest-like; never depicts the player's body/species) |
| Web reveal UX | **Corner toast → click expands to ceremonial card** (combat-safe; no combat-aware hybrid) |
| Terminal reveal UX | **One shared chrysalis ASCII splash scene** via the splash pipeline + name + flavor beneath |
| Art scope | **Style-lock pilot, then all 62** mutations in one batch |
| Rank-ups (deepening) | **Lighter reveal** — toast/flourish with "deepens" wording; no full ceremony |
| Bloom doses | **Stay vague** — keep the authored "Something under your skin shifts and settles differently." line; NO reveal card (naming the mutation would break Bloom's intentional haze) |
| Plumbing | **One event + listeners** (approach A) — grant sites emit; hooks + gmcp listeners deliver |
| Art pipeline | Generate **on the card's leather tint** (2026-07-07 idea — no chroma-key); `strip_icon_bg.py` as fallback |

## Inventory (verified 2026-07-29)

62 mutation YAMLs in `_datafiles/world/dogmud/mutations/`: 10 clusters of
4–7, 9 dual-cluster bridges (rarity 8), 7 rarity-9 apexes, 1 rarity-10.
(The roadmap's "96" was stale.) Relevant YAML fields: `mutationid`, `name`,
`description` (flavor paragraph), `visual`, `rarity`, `clusters`, `pole`.

## Part A — Emblem art batch

### Style lock (user-gated, in browser via visual companion)

1. Generate **3 style variants of one mutation** (candidate house styles,
   e.g. engraved woodcut sigil / brass alchemical emblem / inked tarot
   mark). User picks the house style.
2. Generate **3 more pilots in the winning style** (one common, one apex,
   one bridge) to prove cross-batch consistency. User approves.
3. **Batch all 62** + one `_generic.png` chrysalis fallback emblem.
   User spot-reviews the full grid in the browser.

### Pipeline

- Prompt = shared house-style prefix + per-mutation subject line, all
  checked in at `tools/mutation_art/manifest.yaml`
  (`mutationid → subject`), authored from each YAML's
  name/description/visual. Manifest makes regeneration reproducible.
- `image-gen-mcp` (gpt-image-2) at **quality `low` ONLY** (high/medium
  time out AND still bill — see icon-gap memory). ~$0.35 total for 62.
- Generate on the exact card leather tint (solid hex from the web card
  background) so no background stripping is needed; emblems display in a
  circular frame of the same tint. If tint generation looks bad in the
  pilot, fall back to the proven transparent + `strip_icon_bg.py` route.
- Downscale (LANCZOS) to **256px** → `_datafiles/html/public/static/images/mutations/<mutationid>.png`.

## Part B — Reveal event

### Server plumbing (approach A: one event + listeners)

New event in `internal/mutations`:

```go
type Gained struct {
    UserId     int
    MutationId string
    Rank       int
    IsNew      bool // false = rank-up/deepen
}
func (Gained) Type() string { return "MutationGained" }
// plus a one-line emit helper, e.g. mutations.AnnounceGained(userId, id, rank, isNew)
```

`internal/mutations` gains an `events` import (verified: nothing under
`events` reaches back into `mutations` — no cycle).

**Emitting sites** (each replaces its ad-hoc announce text):

| Site | Today | Change |
|------|-------|--------|
| Combat drift — `internal/hooks/NewRound_UserRoundTick.go` (~:258–320: acquires new OR deepens via `RollDeepening`) | magenta one-liner(s) | emit `Gained{IsNew:true}` on acquire, `Gained{IsNew:false}` on deepen; listener owns all text |
| Pinnacle tick — `internal/hooks/pinnacle_tick.go` (~:213) | "takes root" line | emit |
| Quest engine `give_mutation` — `internal/questengine/bridge.go` (~:346) | **silent (gap)** | emit — fixes the gap |
| Awakening Rite / veteran skip / btree — callers of `Character.GrantRandomMutation(Rare)` | per-site text | emit at call sites (the `characters` method has no user context) |
| Bloom seed/advance — `internal/usercommands/drink.go` | vague line, no name | **keep vague line; do NOT emit** (design decision) |
| Mob mutations | `worldevents.MobMutationGained` | untouched, out of scope |

### Terminal delivery — `internal/hooks` listener on `Gained`

- `IsNew` → queue `splash.Splash{SceneId:"mutation-reveal", Target:User,
  UserId, Data:{name, description}}`. New authored ASCII scene template
  (chrysalis motif, 80-col, no hard numbers) renders with the mutation
  name + flavor paragraph beneath the art.
- Rank-up → short 2–3 line "deepens its hold" flourish (plain text, no
  splash).
- Degradation: if `SplashesEnabled` is off or the template fails, send
  the current-style one-liner instead. Screen-reader users get the
  caption only (existing splash behavior).

### Web delivery — `modules/gmcp` listener on `Gained`

Push `Char.Mutation`:

```json
{"id": "chameleon-skin", "name": "Chameleon Skin", "rank": 1,
 "isNew": true, "description": "<flavor paragraph>",
 "art": "/static/images/mutations/chameleon-skin.png"}
```

Client (`webclient-pure.html` + `gmcp.js`):

- Corner toast: emblem thumb + name + "A mutation emerges — click to
  view" (or "deepens its hold" for rank-ups). Never blocks input.
- Click → ceremonial card: large emblem in circular leather frame,
  "The Chrysalis acts" kicker, name, flavor paragraph; click anywhere
  dismisses.
- Toast auto-fades after ~20s unclicked; simple FIFO queue if multiple
  reveals arrive.
- `onerror` on the art image swaps to `_generic.png`.
- No mechanical numbers anywhere on toast or card (house SOP).

## Out of scope (noted for later)

- A web dashboard "mutations panel" reusing the emblems (none exists
  today; the reveal is the only art consumer for now).
- Per-mutation ASCII art for terminal (one shared scene is the call).
- Mob-side reveal changes.

## Testing & gates

1. Unit: emit-on-grant (drift + quest bridge), gmcp payload shape,
   hooks listener (splash queued for `IsNew`, flourish branch for
   rank-up, disabled-splash degradation).
2. Pre-push SOP boot test (new splash template must load).
3. **Adversarial playtest gate (required, content SOP):** harness run
   driving a fresh character through the Awakening Rite + a combat-drift
   acquisition; read the terminal reveal line-by-line; capture the
   `Char.Mutation` GMCP event.
4. **User gates:** style lock + pilot + 62-grid art review (browser);
   web toast/card eyeball in the real client.
