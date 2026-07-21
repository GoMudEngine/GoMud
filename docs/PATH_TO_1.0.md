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
- 🟡 **Econ loop — a 4-part living-marketplace arc** (bigger than the original "NPC bidders +
  mudmail verify"; the exploration found mudmail's player-to-player send doesn't exist and the
  auction house had a gold-faucet bug). Sub-projects, in order:
  - ✅ **#1 Auction mechanics core** — DONE 2026-07-14 (local master, unpushed; spec+plan
    `2026-07-14-auction-mechanics-core-*`). Escrowed bidding (winner pays, outbid refunds,
    affordability check — fixes the gold-faucet), seller buyout + derived 25% reserve + buy-it-now,
    house commission (gold sink), reliable unsold-return to bank storage/inbox (+ fixed offline
    item-loss & a sold-seller item-dup bug). Unit-tested + suite 91/91 + boot clean.
  - 🟡 **#2 Living-world NPC buyers** — 5 archetypes, ~5 substages (spec+plan per substage):
    - ✅ **#2.1 Foundation + Collector** — DONE 2026-07-14 (local master, unpushed; spec+plan
      `2026-07-14-npc-auction-buyers-foundation-*`). The engine: `NpcBuyer` framework (Interested/
      MaxBid/Wallet) in `modules/auctions/npc_buyers.go`, non-user sentinel bidder (`HighestBidIsNPC`
      + `npcBid` + unified `refundPreviousBidder`) threaded through #1's escrow, per-tick bid-decision
      (incremental, capped at valuation → players always outbid), Collector archetype (prestige gear,
      regenerating persisted wallet, item-sink flavor). Unit-tested + suite 91/91 + boot clean.
      Provisional numbers (min-value 500, premium 1.0, wallet 10k, bid-chance 35%) — tune in playtest.
    - ✅ **#2.2 Craftsperson + #2.3 Adventurer** — DONE 2026-07-14 (batched; spec+plan
      `2026-07-14-npc-buyers-crafter-adventurer-*`). Craftsperson (buys `IsComponent` mats ≥ value),
      adventurer (buys `isEquipment` + `StatMods` gear ≥ value), + `NpcBuyer.Flavor()` for distinct
      per-archetype win broadcasts. Unit-tested + suite 91/91 + boot clean.
    - ✅ **#2.4 Shopkeeper (+ #3 relisting, folded in)** — DONE 2026-07-14 (spec+plan
      `2026-07-14-npc-buyer-shopkeeper-*`). A single dynamic "The Merchants' Guild" persona: per lot it
      scans `shops.AllShops()`, delegates valuation to the existing `shops.EvaluateBuyRules`
      (VendorCategories ↔ CraftSupport match + dynamic buy price + overstock + gold-reserve gate), and
      bids from the best-matching shop's **real gold** (hybrid wallet via a widened
      `CanAfford/Spend/Refund` seam on `NpcBuyer`; `Wallet()` returns nil so it's skipped by the regen/
      persistence loops). A win **relists the item into that shop's stock** (`AddAffixedStock`,
      instance-safe) — folding #3 in so a shop never spends gold for nothing. `AuctionShopkeeperEnabled`
      toggle. Unit-tested + suite green + build clean.
  - ✅ **Min-auction-value floor on listings** — DONE 2026-07-15 (local, unpushed). `AuctionMinListValue`
    plugin config (default 100g); the `auction` command rejects items whose intrinsic `spec.Value` is
    below the floor (gates on value, not the seller's buyout), keeping trivia off the block and out of
    the shopkeeper's relist. `tooTrivialToAuction` helper, unit-tested + boot clean.
    - ✅ **#2.5 Official** — DONE 2026-07-15 (local, unpushed; spec+plan
      `2026-07-15-npc-auction-buyer-official-*`). "The Crown Assessor" — a fifth `NpcBuyer`
      that bids a premium (1.25) from a deep 25k purse on items carrying the new
      `ItemSpec.Restricted` flag, and **sinks** them (no `auctionWinReceiver`). Gated by
      `AuctionOfficialEnabled` + `OfficialPremium`. Seeded the flag onto the six crash-ship
      warden crafting components (40169/40171/40174/40191/40195/40196) so it's live on day one.
      Unit-tested + full boot clean. **This closes the #2 NPC-buyer sub-arc** (all five
      archetypes shipped).
  - ✅ **#3 Shopkeeper relisting** — DONE (folded into #2.4 above): a shopkeeper win routes the item
    into the bound shop's resale stock.
  - ✅ **#4 Bank-storage → auction** — DONE 2026-07-15 (local, unpushed; spec+plan
    `2026-07-15-storage-seizure-auction-*`). When a player can't pay bank-storage rent, the fee hook
    no longer deletes the cheapest slots — it seizes them (lien model). A slot whose aggregate value
    (`spec.Value × Count`) clears `StorageSeizureMinValue` (default 250g) emits a `StorageItemSeized`
    event; the auctions module enqueues it (persisted `SeizedQueue`) and drains one lot onto the block
    per free round, listed **anonymously**. Existing NPC buyers (#2) bid on it for free. On a win the
    house recoups the (tiny) owed rent and the surplus returns to the ex-owner; the winner receives all
    `Count` units. A seized lot that draws no bids is **disposed** (not returned to storage — breaks the
    seize→unsold→re-seize churn), and sub-floor junk is disposed at seizure without ever hitting the
    block. The single floor knob doubles as the kill-switch (set very high ⇒ dispose all ⇒ old behavior).
    Unit-tested (configs/events/hooks/auctions) + full boot clean.
- ✅ **Player-to-player mudmail** — DONE 2026-07-15 (local, unpushed; spec+plan
  `2026-07-15-player-mudmail-*`). New player `mail <recipient>` command (interactive prompts):
  resolves a recipient by character name online *or* offline (`CharacterNameSearch`), deducts
  **on-hand** gold + consumes an attached backpack item, delivers to the recipient's inbox
  (online notify / offline load+save). The existing `inbox` read path credits gold→bank and
  item→backpack. Per-sender **send cooldown** (`MailSendCooldownRounds`, default 10) is the
  minimal anti-spam guard. Also fixed a latent inbox bug: an over-capacity reader used to
  silently lose an attached item — now the message defers unread (no partial gold credit) until
  they free space. Free postage. Pure-helper unit tests + boot clean. **This is the last
  substantive econ-arc item — the living-marketplace arc is complete.**

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
- ✅ 🎯 **Global player channels (chat / newbie / trade)** — **DONE 2026-07-14** (local master,
  unpushed; spec+plan `2026-07-14-global-channels-*`). The reframe: `broadcast` was ALREADY a
  global channel — the real gaps were tunability, a newbie/help channel, and separating player
  chat from the system-announcement firehose. Built three tunable channels on the existing
  broadcast fan-out: new `internal/channels` package (registry + default-on rule), `ChannelMessage`
  event + per-recipient toggle-filtered fan-out (`ChannelMessage_SendToAll`), `chat`/`newbie`/`trade`
  commands + a `channels` manager (toggles via `GetConfigOption`/`SetConfigOption`), `broadcast`
  re-pointed to chat. **Web accounted for both sides:** `gmcp.Comm.go onComm` routes the new
  CommTypes (toggle-filtered) + `webclient-pure.html` Comms tabs. Verified live with a TWO-client
  cross-delivery + toggle test (A→B delivery, B toggles newbie off → stops receiving, broadcast→chat
  confirmed). Minor: usage text uses the `&lt;name&gt;` convention (shared by 4 other commands) —
  eyeball on the real web client.
- ✅ 🎯 **Copyover / hot-reboot** — **DONE 2026-07-14** (local master, unpushed; spec+plan
  `2026-07-14-copyover-*`). Ported upstream GoMud's copyover: live restart without dropping
  players (telnet FDs handed off across the `exec`; web clients auto-relog via one-time tokens).
  New `internal/copyover` pkg + contributors (connections/users/util/gametime/tokens), `copyover`
  admin command + SIGUSR1, main.go wiring (register/restore/resume/skip-steps + websocket relog),
  web-client relog JS, + a DOGMud living-economy flush (shops/forage/caravan/opinions) so nothing
  rewinds. **Compiles to a no-op on Windows; both-platform build + suite + boot-smoke green here.**
  ⚠ **The actual hot-reboot is UNTESTED — validate on the Linux droplet** (checklist below): connect
  a telnet + web client, run `copyover`, confirm telnet survives + web auto-relogs + no economy
  rewind. A failed `copyover.Restore` exits(1) (players drop) → treat as a normal cold restart.

---

## 3. Retention / stickiness

- 🟡 **Player guilds / clans** — **membership core DONE 2026-07-15** (local, unpushed;
  spec+plan `2026-07-15-guild-membership-core-*`). Social guilds (no territory/upkeep/PvP —
  dropped the stub's territorial vision). `internal/guilds` durable per-guild YAML registry;
  member/officer/leader ranks; invite-only join; `guild` command
  (create/info/list/invite/accept/decline/leave/kick/promote/demote/transfer/disband/motd);
  5000g founding fee from bank; MOTD greeting on login. Superseded the `internal/clans` stub.
  **Guild chat + who-tag DONE 2026-07-15** (spec+plan `2026-07-15-guild-chat-whotag-*`):
  `guild chat`/`gc` members-only broadcast (party-chat pattern + Communication event) and a
  `[TAG]` prefix on guilded players in the room "also here" line (`rooms.GetDetails`).
  **Guild treasury + vault DONE 2026-07-15** (spec+plan `2026-07-15-guild-treasury-vault-*`):
  shared gold treasury + item vault persisted in the guild YAML; any member deposits gold (from
  bank) / donates items; leader withdraws/takes, and can `guild treasury delegate` to let officers
  too. `CanWithdraw` gate; `GuildVaultCapacity` config (default 100); item-loss-guarded `take`.
  **Ranks polish (custom rank titles) DONE 2026-07-15** (spec+plan `2026-07-15-guild-rank-titles-*`):
  leader `guild title <rank> <name>` renames the three ranks per-guild (validated, reset to
  default with no name); shown in `guild info` + promote/demote notices. **✅ SOCIAL GUILDS ARC
  COMPLETE** (membership + chat + who-tag + treasury/vault + custom rank titles). Guild
  perks/leaderboard/achievements were not selected.
- ✅ **Achievements / accolades** — DONE 2026-07-15 (local, unpushed; spec+plan
  `2026-07-15-achievements-*`). YAML-authored, boot-validated definitions (`internal/
  achievements`, fixed trigger vocabulary incl. `item_rarity` "acquire a pinnacle item");
  a `NewRound` poll (`internal/hooks`) evaluates online players and privately announces
  unlocks; `achievements` command (earned + locked with progress); `Character.Achievements`
  storage; an achievement-points board on `modules/leaderboards`; a web catalog page
  (`modules/achievements`); and a 22-item starter set across five categories. Poll-based
  (all triggers state-derivable). Unit-tested + boot clean. Event-driven instant unlocks,
  hidden/tiered achievements, and a `/new-achievement` command are future enhancements.

---

## 4. Trust / moderation (once strangers interact)

- ✅ **Player-vs-player report + moderation queue** — **SHIPPED 2026-07-21**
  (spec+plan `2026-07-21-moderation-reporting-enforcement-*`). Investigation found
  the `report` command was a **vital-bar utility**, not moderation — there was no
  reporting path, queue, ban, or admin kick-player. Built a new `internal/moderation`
  package (durable `petitions.yaml` + `bans.yaml`, gitignored living state): a
  player `petition <message>` command → durable admin-reviewable queue + online-staff
  ping; admin `petitions` (list/detail/resolve), `boot` (global disconnect), `ban`/
  `unban` (perm account + optional IP); account/IP ban rejection at login; and
  `mute`/`deafen` extended to global-by-name targeting. Unit-tested + reviewed.
- 🟡 **Link-death / reconnection grace mid-combat** — zombie handling exists;
  verify a dropped connection in a fight is graceful. **Investigated 2026-07-21:
  NOT graceful** — a mid-fight disconnect leaves a "zombie" that stays in combat,
  keeps taking hits, and can die link-dead (Presence stays Active, so the existing
  Disconnected-target veto never fires). This is sub-project B of §4, deferred to a
  follow-up build (reconnect itself is clean).

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
- ✅ **Duplicate/respawning quest "notice" items** — **already resolved** (2026-07-21 retest):
  the duplicate source was the respawning floor copy in room 472, removed in content (see the
  comment in `rooms/thornwall_city/472.yaml`); the notice is now handed out only by Marek's
  dialogue. Residual corner case (not fixed, no clean fix): the `lost_notice` recovery node
  re-gives on a keyword re-ask without checking possession — dialogue has `requiresItem`
  (require+consume) but no "hide if player already has item" condition. Low risk (requires
  deliberately re-asking about the notice while carrying one).
- ✅ **First Blood "kick/trip dummy" step nearly unhittable** — **fixed 2026-07-21**
  (`quests/32-first_blood.yaml`). The `32-special` (kick/trip) and `32-consider` triggers now
  use `command_issued` (credits the typed command) instead of `command` (needs a landed hit on
  a live target), so the fragile one-shot-able dummy no longer closes the window. Playtest-
  verified end to end: `kick dummy` on a dead/respawning dummy returned "You don't see them
  here" yet still advanced the quest; full chain (strike→special→consider→verbosity→end)
  completes. Report: `tools/playtest/reports/2026-07-21-local-feature-tester-first-blood.md`.
  Minor cosmetic residual (not blocking): the miss line shows one line above "made progress".
- ✅ **Ephemeral room id `1000000000` for The Mending Hut** leaking onto `Zone.Map` — **fixed
  2026-07-21 at the root (feeder) sites.** The render-layer symptom was already guarded
  (`mapper.snapshot.go` ephemeral skip, `35851764a`), but the underlying defect was that map/fog
  feeders recorded the *raw instance id* instead of translating to the template id via
  `rooms.OriginalRoomId` (as the quest engine already does at `go.go:306`). Two feeders fixed:
  `go.go:314` `MarkRoomVisited` now stores the template id (was permanently baking instance ids
  into saved `Character.VisitedRooms`), and the `gmcp.Zone.go` party-array now translates a party
  member's room id (latent leak when a member stands in an instance). Build + full suite green;
  existing `TestSnapshotSkipsEphemeralRooms` guard retained as defense-in-depth.
- ~~ASCII charset "gap"~~ — **NOT a game bug.** `set charset` is a client-mode toggle;
  the testers just didn't converge to ASCII (a harness-driver step, documented in the
  engine profile). No server change needed; ensure future testers converge.
- ✅ **Inconsistent item-name capitalization** (2026-07-14, Meirok screenshot) — **fixed
  2026-07-21.** Root cause: affixed/boss-drop instances snapshot the template `Name` into a
  per-instance `overrides.name` at mint time (`affixgen.go`), and the one-time casing sweep
  (`320592539`) Title-Cased item *templates* but skipped the `users/` tree — so pre-sweep
  instances kept `drowned claws` while templates + newer instances hold `Drowned Claws`. (The
  `(Masterwork)` adjective lives separately in `Adjectives`, hence it rides on both casings.) Fix:
  `Item.DisplayName()` now routes the bare `spec.Name` through `casing.Title` (the single source
  of truth for display casing) — idempotent → no-op on canonical templates, self-heals every stale
  snapshot (present + future template renames), touches no player saves; authored `DisplayName`
  overrides remain verbatim. Failing test written first (`TestItem_DisplayName_CasingNormalization`);
  full `internal/items` suite green.
- ⬜ **GMCP `Char.Stats` transient spike** (LOW, DEFERRED) — right after "STATISTIC INCREASED"
  the wire feed briefly reports a stale `ValueAdj`, reverting next round. ASCII `status` is fine;
  only a rich/web client flashes it. **2026-07-21 investigation: the "ordering" hypothesis is
  DISPROVEN.** The softcap recompute is *eager* — it runs inside `IncreaseStat` →
  `Validate` → `RecalculateStats` → `StatInfo.Recalculate()` (`stats.go`) and strictly precedes
  both the "STATISTIC INCREASED" message and the `CharacterStatsChanged` event that drives the
  GMCP push, so the progression-path GMCP read (`gmcp.Char.go:485`) is always current. The only
  real seam is a *different* Char/Char.Stats push (e.g. from `UserRoundTick`/`MobRoundTick`, which
  run before `DoCombat` in `hooks.go`) reading live `ValueAdj` earlier in the same round drain —
  a value that is correct at that instant and merely about to change. That is a low-value cosmetic
  timing artifact, not a stale read; a responsible fix needs a live repro + instrumentation to
  confirm the emitter and whether it's worth coalescing round-drain Char pushes. Left deferred
  rather than blind-patched.
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
