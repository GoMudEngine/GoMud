# DOGMud — Path to 1.0

**Goal of 1.0:** the point at which we're comfortable advertising DOGMud heavily on
the MUD listing sites (TMC, MUDConnect, Grapevine, mudstats, r/MUD, etc.) — i.e. a
new player who arrives from an ad gets a stable, coherent, community-feeling
first hour and a reason to come back.

**Status legend:** ✅ done · 🟡 partial / needs finishing · ⬜ not started ·
🎯 advertising-critical

Living document. Started 2026-07-13.

---

## 0. Where we already are (the good news)

Two things on the original wishlist are largely already built, and the newbie
experience just passed a full feel-test — so the hill is shorter than it looks.

- ✅ **MUD mail with gold + items** — the `mudmail` command already sends a
  message plus attached gold plus an attached item; on receipt gold lands in the
  recipient's **bank** and the item in their backpack (`internal/usercommands/
  inbox.go`, `admin.mudmail.go`). Bank tie-in exists. → Reclassified to "verify +
  polish," not "build."
- 🟡 **Auction house** — `modules/auctions/` is a working player auction system
  (list lots, bid, winner + sales-history tracking). → The net-new work is your
  "**NPCs bid with their gold**" behavior + confirming it's enabled/tuned, not
  building the house.
- ✅ **Landing page / "play in browser" front door** — the web client's default
  page at https://www.dogmud.org already is this. Removed from the list.
- ✅ **Newbie experience** — full feel-test 2026-07-13 (Fernling): Awakening
  Ceremony reads as intentional, blank-slate mutation framing is fully consistent
  (no "you were given a mutation" leak), first mutation emerges at a satisfying
  point on the combat trail, and the tutorial→open-world handoff is smooth. Not
  a 1.0 blocker. (See `tools/playtest/reports/2026-07-13-local-feel-tester-newbie-full.md`.)

---

## 1. Build items (from your list)

- ⬜ **Admin web-building** — replicate in-game admin building via the web/mapper
  UI: create rooms from the mapper, add items / NPCs / behavior trees / dialogue,
  edit exits, etc. Highest-value *build* item — massively raises content velocity
  and lets non-coders contribute. Biggest scope here; worth its own spec.
- 🟡 **Weather / seasons polish** — the systems exist (`modules/weather/`); finish
  and polish. Scope the specific gaps before committing.
- 🟡 **Auction: NPC bidders** — add NPCs that bid on player lots with real gold
  (gold-gated), so the house feels alive even at low population; verify enabled.
- 🟡 **MUD mail: verify + polish** — confirm it's enabled and discoverable for
  players; UX pass on the send flow and inbox.

---

## 2. Advertising-critical gaps 🎯 (do these or the ads underperform)

- ✅ 🎯 **MSSP (MUD Server Status Protocol)** — **DONE 2026-07-14** (local master, unpushed;
  spec `2026-07-14-mssp-design.md`, plan `2026-07-14-mssp.md`). Telnet option 70, mirroring the
  existing MSP option: `internal/term/mssp.go` (byte protocol + encoder), `internal/inputhandlers/
  mssp.go` (field assembler), `term_iac.go` reply + `main.go` WILL-offer, `internal/util` uptime
  accessor, new `Server.MSSP` config block. Rich field set (live player count/uptime + world counts
  + capability flags + descriptive config). Contact/Hostname/Port default empty (privacy — public
  repo). Verified live via `tools/mssp_probe.py`. **Post-deploy:** validate against a public MSSP
  checker + register DOGMud with the directory sites (manual, non-code).
- ⬜ 🎯 **Global player channels (gossip / chat / newbie / OOC)** — **appears
  absent** (we have `say`/`shout`/`whisper`/`broadcast`/`who` + a Discord bridge,
  but no tune-able game-wide chat or newbie-help channel). A newcomer needs to
  *feel* other players and ask for help world-wide. Core community glue and
  retention. (Confirm truly absent before building.)
- ⬜ 🎯 **Copyover / hot-reboot** — **not ported** from upstream (in the backlog).
  During a launch push we iterate constantly; without copyover every update
  disconnects everyone — worst right when we're most active.

---

## 3. Retention / stickiness

- ⬜ **Player guilds / clans** — absent. One of the strongest "bring your friends /
  reason to stay" drivers in the genre. Larger build; a real 1.0-vs-not
  differentiator.
- ⬜ **Achievements / accolades** — goals + bragging rights for new players; plugs
  into the leaderboards we already have.

---

## 4. Trust / moderation (once strangers interact)

- 🟡 **Player-vs-player report + moderation queue** — we have admin `mute` and a
  `report` command; verify it handles harassment reporting into an admin queue for
  a stranger crowd.
- 🟡 **Link-death / reconnection grace mid-combat** — zombie handling exists;
  verify a dropped connection in a fight is graceful.

---

## 5. Balance & polish that gates 1.0 (not features)

### 5a. Mutation balance retune (reopened by the 2026-07-13 veteran feel-test)
The blank-slate + center-out redesign is working; the **rate/gate overshoot in long
fights** got a first-pass fix (2026-07-13, master `9a6606a3d`) — needs live playtest:
- ✅ **Connective-tissue (bridge) mutations reworked to apex-class.** All 9 bridges →
  rarity 8 (just below the r9 cluster apexes) + binary (`max_rank: 1`, no deepen).
  They're apex-powers shared by two clusters; reachable via either cluster.
- ✅ **Depth gate made QUADRATIC** (`threshold = rarity^2 * PerRarity`, PerRarity 6→2)
  + per-action affinity gain halved (PerSkillUse/PerCombatEvent 1.0→0.5). Linear
  scaling had compressed r3→r9 into ~3×, letting Extra Arms emerge in fight #1;
  quadratic spreads it (r3~18, r8 bridge~128, r9 apex~162) so high tiers need
  sustained dedicated drift. **PROVISIONAL — verify in playtest that Extra Arms now
  takes many fights and entry mutations still feel timely.**
- ✅ **Early acquisition cadence dialed back** (master `cdd953a1d`) — `MutationBaseProgress`
  15→30, taking the 6e ~8x down to ~4x of the pre-6e rate (first mutation ~15 rounds, not
  ~7.5). Faster than original but not overboard. PROVISIONAL — re-verify in playtest.
- ⬜ **Early stat progression may be too generous** — both testers noted ~3 stat-ups
  in the first bout. Watch; possibly trim the 6e stat rate slightly.

### 5b. Bug burn-down (surfaced 2026-07-13 + existing backlog)
- ✅ **`read` falls back to item description** (master `9a6606a3d`) — quest notices/
  letters authored as `object` now read; no more "read the notice" dead end.
- ✅ **"You receive a Thornwall Pass!"** (master `9a6606a3d`) — quest reward now uses
  the full display name + article instead of the keyword ("pass").
- ✅ **Drillmaster Vorn flees when you `attack dummy`** (master `7ae6ff68e`) — fixed via
  the broader **witness-response faction gate**: attacking a factionless target (dummy/
  wildlife/monster) no longer triggers bystander alarm/revenge, mirroring the crime record.
- ⬜ **Duplicate/respawning quest "notice" items** (retest) — end up carrying 2-3 copies of
  "Protection Notice" (room pickup + dialogue `givesItem` + a floor respawn). Room-reset-vs-
  givesItem dedup gap.
- ⬜ **First Blood "kick/trip dummy" step nearly unhittable** (retest) — after the quest's own
  grind step, auto-attack 1-2-shots the respawned dummy, so the special-move window closes
  before you can land kick/trip (~10 failed attempts). Dummy scales but not fast enough.
- ⬜ **Ephemeral room id `1000000000` for The Mending Hut** shows alongside the real 5209 on
  `Zone.Map` (minor mapper quirk; instance-room id leaking into the snapshot).
- ~~ASCII charset "gap"~~ — **NOT a game bug.** `set charset` is a client-mode toggle;
  the testers just didn't converge to ASCII (a harness-driver step, documented in the
  engine profile). No server change needed; ensure future testers converge.
- ⬜ **Inconsistent item-name capitalization** (2026-07-14, Meirok screenshot) — the SAME item
  shows both cases in the equipment list (`drowned claws` vs `Drowned Claws`, `storm bracer` vs
  `Storm Bracer`), all with the same `(Masterwork)` adjective. Not a render bug — it's baked
  per-instance: `Item.DisplayName()` returns `spec.Name` merged with per-instance `overrides`,
  and some crafted/affixed copies carry a Title-Cased `name` override while base copies keep the
  lowercase template name. Seam: affix/masterwork generation (`internal/items/affixgen.go`) and
  `Item.Rename`. Cleanest fix is likely to normalize casing at the render layer (`DisplayName`
  Title-cases or sentence-cases consistently) so stored inconsistency stops mattering. LOW pri.
- ⬜ **GMCP `Char.Stats` transient spike** (LOW) — right after "STATISTIC INCREASED"
  the wire feed briefly reports a stale `ValueAdj` (pre-softcap-recompute), reverting
  next round. ASCII `status` is fine; only a rich/web client flashes it. Deferred —
  needs the stat-increase → recompute → GMCP-update ordering tightened.
- Plus existing backlog (Vitalis Bandolier potion-rot, etc.).

### 5c. Content / onboarding polish
- 🟡 **Veteran onboarding starter gear** — both testers hit the "dropped in unarmed/
  unarmored two rooms from a hostile ambush" welcome. Seed minimal gear for the
  veteran path, or telegraph "get equipped first" clearly.
- 🟡 **Help coverage** — verify every system a newbie touches has a discoverable
  help file.
- 🟡 **Owed feel-checks** — newbie-zone aggression not oppressive; endgame
  (#20/#21) tuning.
- 🟡 **Suppress NPC ambient barks during a paced rite** (Hadwen's backstop line
  interleaved into the ceremony once).

### 5d. Presentation / delight polish
Not gating, but directly serves the "coherent, memorable first hour" advertising
goal — cheap wins that make the world feel crafted.
- ⬜ **Mutation-acquisition art + reveal popup** — reuse the eq-icon generation
  pipeline (`image-gen-mcp` low-quality → `tools/strip_icon_bg.py`) to make small
  per-mutation illustrations, and show a richer reveal on acquisition. Two
  independent pieces: **(A) art batch** (no engine risk; lock a shared house style,
  pilot on ~3, MVP = the 13 apexes + keystones first with a per-cluster crest
  fallback, then expand to all 96) → `static/img/mutations/<id>.png`; **(B) reveal
  event** — a GMCP event carrying the mutation id/name at grant time + a web-client
  toast/modal with the illustration + flavor, degrading gracefully to a richer text
  flourish on terminal clients. Brainstorm the style lock + GMCP shape before
  generating anything.
- ⬜ **Celestial splash screens** — improve the moon-phase / sunrise / sunset splash
  art & messaging. The events already fire; this is a presentation pass (better
  ASCII/art, evocative copy, maybe a web-client visual treatment) so day/night and
  the lunar cycle feel like events, not log lines.

---

## 6. Suggested sequencing

1. **Cheap, high-leverage-for-advertising first:** MSSP → global channels →
   copyover. (These three most directly make the advertising pay off.)
2. **Finish the near-done economy loop:** verify/polish mudmail + auction, add NPC
   bidders. (Small, satisfying, player-facing.)
3. **The mutation balance retune** (5a) — needs live playtest iteration; start
   early so it bakes.
4. **Bug burn-down** (5b) — knock these out continuously; several are quick.
5. **Admin web-building** — the big build; spec it, then chunk it.
6. **Retention layer:** guilds/clans, achievements.
7. **Weather/seasons polish, moderation hardening, content/help polish, presentation
   / delight polish (5d)** — parallel / as-capacity. The 5d art batch in particular
   has no engine risk and can start any time.

---

## Appendix — 2026-07-13 feel-test summary

Two autonomous feel-testers via the GoMud playtest harness (`feel-tester`
personality, background subagents):
- **Fernling** (full newbie): newbie arc is in strong shape — see §0. Report:
  `tools/playtest/reports/2026-07-13-local-feel-tester-newbie-full.md`.
- **Rover** (veteran, wider world, ~90 min): surfaced the mutation-rate overshoot
  (§5a) and the `read`/reward-template bugs (§5b); praised combat text, loot, quest
  flow. Report: `tools/playtest/reports/2026-07-13-local-feel-tester-veteran-wider-world.md`.
- Neither died (newbie zone is sanctuary-buffed; veteran never dropped below ~58%),
  so the death→respawn fix wasn't re-verified by these runs — it was verified
  separately (Punchy/Vera). A future tester should deliberately seek a lethal fight.
