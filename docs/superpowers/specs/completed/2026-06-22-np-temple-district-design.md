# New Plymouth — District 5: The Temple Quarter (design)

**Status:** approved 2026-06-22 (brainstorming). Build sequence item #5 of the
capital (Docks ✅ → Common ✅ → Crafting ✅ → Merchant ✅ → **Temple** → Noble →
Old Quarter). Pulls anchors from the city-wide layer
(`2026-06-20-new-plymouth-citywide-design.md` §6 #29–35).

> **PUSH POLICY (user, 2026-06-20):** Do NOT push New Plymouth to prod until the
> WHOLE capital is built and tests well. Accumulate locally; no mid-build push.

## 0. Scope (locked with user 2026-06-22)

IN: rooms + mobs + dialogue + **the `temple_np` faction** (+ the `bloodline_domestic`
**ally** backfill) + **anchor schedules** + a **Grand Temple respawn anchor**
(opt-in, `set home`) + **Dobb visible transit** to the temple gate + an **Archive
deep-lore stub** (Holt's Restricted Collection + Dross/Ept lore breadcrumbs).

OUT (deferred): **the district quest** ("The Doubting Novice" — author Ept's doubt
+ the Ept↔Orin cooperage cross-reference as discoverable dialogue/lore only, no
quest-engine work); FORCED respawn (opt-in only — players `set home`, matching the
Thornwall/Stillwater pattern); the Noble physical seam (stubbed); resolving any
deep-lore (Archive/cipher/orbital-symbol stay sealed).

## 1. Zone, IDs, coordinates

- **Zone folder:** `new_plymouth_temple` (underscores; `ConvertForFilename("New Plymouth Temple")`).
- **Rooms:** 5900–5924 (25). **Mobs:** 9356–9367 (7 anchors 9356–9362 + 5 ambient
  9363–9367). **Dialogue:** by mobid. **Items:** 40106+ only if needed. **Quests:** none.
- **Coordinates — REAL unified frame** (not nominal). Live z0 footprint: Common
  x[−16..−12]/y[82..88], Crafting x[−20..−15]/y[87..92], Merchant x[−15..−9]/y[88..91].
  The Temple occupies the **open ground EAST of the Merchant zone** (x[−8..+6],
  y[85..95] — solved collision-free at build time; `cartcheck errors=0`).

Role (city-wide §2): *faith; respawn; Archive; pilgrim arrival* — the eastern
terminus of the Long Market.

### Connections
- **Primary entry:** Merchant **5823 (The Porters' Dock, at −9/89)** is the east
  end of the Merchant y89 spine. Wire **5823 → east → the temple processional
  road** (5900) + reciprocal `west`. Prose: the road leaves the market bustle and
  climbs toward the temple precinct. (5823's existing exits W→5822, S→5812, N→5819
  stay intact; east is currently free.)
- **Deferred prose-stubs** (gotcha #5): the Long Market terminates at the Temple
  (no further east exit, prose-stub); any Noble-ward seam stubbed. Interiors use
  compass/vertical exits only (gotcha #10).

## 2. Layout & connections

The **Grand Temple** (sanctuary) is the hub and the respawn anchor; clusters:
- **The Grand Temple** — the processional road (5900) → the great sanctuary
  (**5901 = the respawn/home room**) + the high altar; rest, worship, respawn.
- **The Keeper's House** — Yelin (warden) + Father Thane (desk intake); pilgrim/
  keeper processing.
- **The healer's chapel** — Sister Alms.
- **The seminary** — Novice Ept's dormitory.
- **The courtyard + the Archive** — Scholar Dross (courtyard debates); Archivist
  Holt's **Restricted Collection** (a locked deep-lore gate — prose-only,
  impassable, like Horst's house).
- **The canon's cell** — Canon Merid (blessings/regen; Archive access; bloodline-
  aligned).

## 3. Anchors (7 — city-wide §6 #29–35), full life-sheets

Each: home + work room (live-in cells/dormitory where noted), ≥3 discoverable
first-person dialogue topics, the unique mutation in the description, faction via
`groups:`, a 24h `schedule_id` (Stage D). Shop mobs (if any) need `craft_support:`.
Most temple anchors have **no shop** (faith is a service); Canon Merid MAY offer a
minor blessing/component shop (`general` or `enchanting`) — decided at build.

| # | Mob | Name | Mutation | Role | `groups` | Notes |
|---|-----|------|----------|------|----------|-------|
| 29 | 9356 | **Yelin** *(canon)* | hands worn smooth like prayer-stones | lay brother / warden, Keeper's House — NO shop | humanoid, temple_np | 19+ years; gatekeeps visiting keepers; keeps the house in order |
| 30 | 9357 | **Father Thane** *(canon)* | a voice that soothes the anxious | desk intake, Keeper's House — NO shop | humanoid, temple_np | processes keepers; calm, orderly; "everything in its register" |
| 31 | 9358 | **Canon Merid** | a faint dawn-prayer halo-glow | temple canon (blessings/regen) — optional minor shop | humanoid, temple_np | oversees Archive access; **bloodline-aligned** (keeps faith's machinery turning) |
| 32 | 9359 | **Novice Ept** | sees the old orbital symbol everywhere | doubting novice, seminary — NO shop | humanoid, temple_np | **Ept↔Orin cooperage cross-reference** (lore breadcrumb; quest deferred); the orbital-symbol lore |
| 33 | 9360 | **Scholar Dross** | an overlarge, veined cranium | skeptical scholar, courtyard — NO shop | humanoid, temple_np | argues the inscriptions' age; the gallery-cipher link (Noble, later) |
| 34 | 9361 | **Sister Alms** | warm hands that mend | temple healer, the chapel — optional minor healing-goods shop | humanoid, temple_np | the legit counterpart to Ysolde (Common); heals the deserving and not |
| 35 | 9362 | **Archivist Holt** | eyes that catalog at a glance | Restricted Collection gate — NO shop | humanoid, temple_np, **bloodline_domestic** | the institutional intertwine; lets nothing out that shouldn't be (deep-lore gate) |

Plus **~4–5 ambient** (a pilgrim or two, a lay sister, an acolyte, a beggar at the
temple gate) — `non_combatant` (mobs 9363–9367).

**Authoring gotchas:** YAML `": "` → `" — "`; Title-Case names; `ConvertForFilename`
filenames; faction via `groups:`; noun ansi `fg="itemname"` (post-grep); interiors
compass/vertical; **sentence-leading noun tags lowercase** (match the noun key — a
real prior-district defect); avoid short trigger stems that prefix-shadow longer
discoverable words (the `buy`/`buyer` lesson).

## 4. Factions — create `temple_np`, backfill `bloodline_domestic`

- **Create `_datafiles/world/dogmud/factions/temple_np.yaml`:** `default_rep: 0`;
  **allies: [bloodline_domestic]** (the institutional intertwine — bloodline_domestic
  exists, so legal); enemies: []. Members (via `groups:`): all 7 anchors; Holt
  (9362) is in BOTH temple_np + bloodline_domestic.
- **Backfill `bloodline_domestic.yaml`:** add **allies: [temple_np]** (the edge
  deferred from the Merchant build — both factions exist now, so the mutual ally
  reference resolves; gotcha #2 only bites on a faction not built at all). Keep its
  `enemies: [cooperage_circle]`.
- Boot-verify faction-load count (18→19) + no panic.

## 5. Respawn anchor (the canonical Temple role — opt-in)

Add the Grand Temple sanctuary to the respawn/home map in
`internal/characters/respawn_home.go` (the same pattern as the Thornwall/Stillwater
temple anchors):
- `HomeLocations["newplymouth"] = 5901` (the Grand Temple sanctuary).
- `HomeLocationNames["newplymouth"] = "New Plymouth (The Grand Temple)"`.
Players in NP can then `set home` (→ "newplymouth") and respawn at 5901 on death.
**Opt-in only** — no change to default respawn. Small Go change + a confirming boot
(and ideally a quick in-game `set home` + death-respawn check at 5901).

## 6. Dobb visible transit + Archive stub + lore breadcrumbs

- **Dobb transit (patrol YAML only):** extend `np_docks_runner_circuit` one
  waypoint east to the **temple gate** (the processional road 5900 or the gate
  plaza) as a **visible transit stop completing the Long Market spine** — `dwell`
  for visibility, `arrival_event: np_runner_vendor` is harmless (no shop there →
  delivers nothing) OR `""` (no event). **No delivery** (the Temple isn't a craft
  vendor); Temple stays OUT of `CaravanServedZones`. Every new waypoint must be
  pathto-reachable across the Merchant→Temple seam.
- **Archive deep-lore stub:** Archivist Holt guards the **Restricted Collection** —
  a locked gate, prose-only, impassable (no interior exit). Plus discoverable,
  **unresolved** lore: Dross's **gallery-cipher** argument (points at the Noble
  gallery, later), Ept's **orbital-symbol** obsession + his cross-reference to
  Orin's cooperage lore (Crafting). Seeds future content; resolves nothing.

## 7. Anchor schedules (Stage D)

One `schedule_id` per anchor at `schedules/new_plymouth_temple/`, 24h-contiguous
(validators panic on gaps/unreachable). Beats: **matins/dawn offices** at the
sanctuary (the anchors converge to worship); **Sister Alms's chapel hours**;
**Dross's courtyard debates** (midday); **Yelin's warden rounds** (Keeper's House);
**Merid's blessings**; quiet night with a lay vigil. All `pathto` targets within
the built Temple rooms.

## 8. Build staging — feeds ONE plan (each stage boot-verified)

> **Pre-smoke ritual** (CLAUDE.md SOP): wipe instance saves before each boot;
> boot-poll for `ERROR:.*PANIC`/`fatal error:`, not bare "panic" (gotcha #8).

- **Stage A — Grand Temple + entry (rooms):** the processional road (wire Merchant
  5823 `east`), the sanctuary (5901, the respawn room) + altar, the gate plaza.
  `cartcheck`-clean.
- **Stage B — cluster rooms:** Keeper's House, healer's chapel, seminary, courtyard
  + the Archive (Holt's locked gate), the canon's cell. Boot-verify.
- **Stage C — population:** 7 anchors + ambient + dialogue + any shops + `temple_np`
  faction (+ `bloodline_domestic` ally backfill) + room spawns.
- **Stage D — respawn anchor + Dobb transit + Archive/lore + schedules:** §5/§6/§7.
  Boot-verify (faction count, schedule/patrol validators, the respawn map compiles);
  ideally an in-game `set home`/respawn check.
- **Stage E — district harness playtest** (`/playtest local feature-tester`) →
  report → merge `--no-ff` (hold push) → update memory.

## 9. Definition of done

- 25 rooms (5900–5924) boot-clean, `cartcheck` errors=0, the Merchant→Temple entry
  works.
- 7 anchors + ambient live; Holt's Restricted Collection impassable; dialogue ≥3
  topics each.
- `temple_np` faction active + the `bloodline_domestic`↔`temple_np` mutual-ally edge
  live.
- Grand Temple respawn anchor works (opt-in `set home` → respawn at 5901).
- Dobb's circuit visibly reaches the temple gate.
- Ept/Dross/Holt lore breadcrumbs in place + unresolved; Ept↔Orin cross-reference.
- Anchor schedules pass validators.
- Harness-playtested; report committed. Merge to master, hold push.

## 10. Honored gotchas checklist

#2 faction refs (temple_np↔bloodline_domestic mutual is legal — both built;
cooperage_circle enmity already live) · #5 no exits to unbuilt rooms · #6 shop mobs
need `craft_support:` · #7 `fg="itemname"` ansi · #8 boot-poll `ERROR:.*PANIC` ·
#10 cardinal/vertical interiors + REAL unified-frame coords · Plan-2 supply rules
(visible transit, no CaravanServedZones, runner ticks only when player-active) ·
sentence-leading noun tags lowercase · no prefix-shadowing triggers · YAML `": "`
→ `" — "` · Title-Case names.
