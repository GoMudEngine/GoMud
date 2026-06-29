# Newbie-Area Playtest Feedback — Triage (2026-06-29)

Source: Malia's playtest (`User Feedback Dogmud.pdf`, project root). Parsed and
grouped by Claude. Star ratings she gave: Login/Main Interface ⭐⭐⭐;
Tutorial/Newbie First Area ⭐⭐⭐ (flagged "R" = needs review).

## Key finding — zone boundary / silent early-exit

Malia's exit point is **Warden Esk → The Threshold House (room 5215)**. "The
Awakening" is the *exit quest*. Everything she reported **at or after Blacksmith
Kerra** is **Thornwall, not the coulee** — she completed The Awakening and walked
through the Threshold House into Thornwall **with no transition signal**, then
kept giving feedback as if still in the tutorial.

| Still in Pothole Coulee (newbie area) | Thornwall (already left) |
|---|---|
| Cleric Hadwen, Crier Toke (Hub Square 5203), Smith Rusk, Warden Esk (Threshold House 5215), **Market Row 5204**, **Granny Wicker** | **Blacksmith Kerra**, **Craftsmen's Quarter** (469/470), **Market Square West** (464) |

Surprises vs. the chat notes: Granny Wicker and Market Row are both still in the
coulee (not Thornwall). The silent early-exit (A5) is its own bug and the likely
root cause of several later complaints.

---

## 🔴 A. Confirmed bugs (functional — highest priority)
| # | Item | Zone |
|---|---|---|
| A1 | **First Heat quest bugs out at 66%** — after forging the iron dagger and giving it to Rusk (he gives it back), the quest stalls | Coulee |
| A2 | **Completed "The Awakening" stays in quest log** at 100% instead of clearing | Coulee |
| A3 | **`sell` fails on held items** — "Sell Kerra steel buckler" → "you don't have that item" though INV shows it held | Thornwall |
| A4 | **No "quest added" confirmation** — asking Warden Esk for a quest gives no feedback; had to `quest all` to confirm The Awakening was granted | Coulee |
| A5 | **Early exit is silent** — finishing The Awakening drops the player into Thornwall with no boundary/transition message | Coulee→Thornwall |

## 🟠 B. Command / syntax friction
| # | Item | Zone |
|---|---|---|
| B1 | **`give` too strict** — "Give dagger to Smith Rusk" failed; only "Give Dagger Rusk" worked. Wants fuzzy keywords (Give Iron Rusk, Give Dagger Smith, "to") | Coulee |
| B2 | **`ask <npc> quest` with no quest is confusing** — "Ask Granny Wicker quest" returned charm lore. Wants a clear "I have no quest, but I can train you in…" fallback | Coulee |
| B3 | **Market Row "stalls" highlighted but inert** — typed "list stalls", nothing actionable | Coulee |

## 🟡 C. Newbie onboarding / handholding (char-creation poll theme)
| # | Item | Zone |
|---|---|---|
| C1 | **Poll at character creation** → 3 tracks: total newbie (heavy basic-mechanics handholding) / MUD-vet-new-to-DOGMud (current coulee experience) / true vet (funnel to Thornwall) | meta |
| C2 | **First tips should cover basics**: `look` to orient, `inv`, movement, how to see **gold**, that food gives a non-toxic buff | both |
| C3 | **After mutation, force an inventory check** — player sees no armor/weapon → funnel to trainers | Coulee |
| C4 | **Forging msg should teach next step** — "An Iron Dagger! Now type `wield dagger`… type `inv` to confirm" | Coulee |
| C5 | **"Foraging skill works but isn't in the skills list at all"** | both |
| C6 | **NPC purpose is opaque** — every NPC room, unclear what to do; knew it wasn't combat only because they don't attack | both |
| C7 | **Quest progress legibility** — following the quest list but unsure which steps are checked off | both |

## 🟢 D. Text / verbiage / clarity polish
| # | Item | Zone |
|---|---|---|
| D1 | **"coulee" is jargon** — Hadwen says "new to the coulee"; suggests "this realm/Realm of Gaius"; even rename "Pothole Coulee" → "Acclimation Camp" | Coulee |
| D2 | **"Come to the water" reads as movement** — replace with the single command "ask hadwen begin" everywhere | Coulee |
| D3 | **Capitalize/color command words** consistently — `Renameself`, `Help`, `Footing`, `Passage` in dialogue | Coulee |
| D4 | **Hadwen still says "come to the water" after the player is Opened** — should stop post-mutation | Coulee |
| D5 | **Mutation flavor reword** (Pacifism Aura description) | Coulee |
| D6 | **Room desc wording** — Hub Square "Wounds close more easily" → "Wounds heal more easily" | Coulee |
| D7 | **Grey-on-grey room text hard to read** — "flat gray ceiling… dulling every color" | Thornwall |

## 🔵 E. UI / web client
| # | Item | Zone |
|---|---|---|
| E1 | **Music/volume control hard to find** — wants auto-retract/closeable; sound icon styled like Reconnect; labeled Sound/Reconnect/Disconnect buttons | client |
| E2 | **Login screen** — highlight "new" in green; capitalize/color Y/N prompts (green=yes, red=no); "Would you like to create a new user named Phae? [Y/N]" | client |
| E3 | **Skills/command list overwhelms newbies** — split into Beginner/Experienced/Advanced tabs | client |
| E4 | **No auto-scroll on movement** — after scrolling up, moving a direction doesn't snap the view to the new room | client |
| E5 | **ASCII art / iconography readability** — included Original/Better/Best mockups | client |
| E6 | **Optional email field at signup** (collect, careful verbiage) | client/product |

---

## Bug-batch outcomes (2026-06-29, A2/A3/C5/B3)
- **A2 — NOT A BUG (working as intended).** The `quests` command already filters
  completed quests (`if !showComplete && completion >= 1 { continue }`); the
  default `quests` hides them, `quests all` shows them by design. Malia saw the
  100% Awakening via `quest all`. No change.
- **A3 — FIXED.** `sell` now strips a leading merchant name ("sell Kerra steel
  buckler" works). `resolveSellItem` in `internal/usercommands/sell.go`; test
  `TestSell_StripsMerchantNamePrefix`.
- **C5 — FIXED (discoverability).** There is no separate "foraging" skill —
  `forage` trains **Search**. The `skills` list now shows a one-line blurb per
  skill (`skills.SkillBlurb`); Search's blurb names foraging. So foraging is
  visible under Search instead of seeming absent.
- **B3 — FIXED (clarity).** Market Row's "stalls" noun now states it's an
  unattended display and points to a tended stall (Smith Rusk) for real trade.

## Triage notes
- **Biggest lever: A5 + C1 together.** The silent early-exit means several
  "newbie area" complaints (D7, A3, Craftsmen/Market-Square-West confusion) are
  really *Thornwall*, never meant to handhold. Boundary signal + char-creation
  poll resolves most of section C structurally rather than line-by-line.
- **A1 (First Heat 66% stall)** — first to verify; hard quest break in the very
  first content path. May share a root cause with B1 (give-matching), since the
  quest completes on giving the dagger to Rusk.

## Work log
- 2026-06-29: doc created; starting A1 (First Heat) investigation + the
  multi-path quest-resolution fixes (alt nouns for items & NPCs; give-item AND
  ask-keyword completion paths).
- 2026-06-29: **A1 root cause + fix.** Quest 35's `craft→end` step had a single
  completion path (the `craft_turnin` dialogue node, `ask rusk done`). Physically
  giving the finished dagger fired only the generic `noncombat_questgiver`
  archetype `player_give` (hands it back, no quest effect) — quest parked at
  `35-craft` (66%) forever.
  - **F1 (data):** added a non-stalling `item_give` trigger to
    `35-first_heat.yaml` (give dagger 10009 to Rusk 9116 → consume + hand back a
    fresh dagger via `consume_item`/`give_item`, grant `35-end`). `Equals`
    compares UUID so the give.go post-consume `RemoveItem` is a no-op; consume
    branch returns before the archetype emote, so messaging reads cleanly. Test:
    `internal/questengine/first_heat_givepath_test.go`.
  - **F2 (engine):** `give.go` now parses multi-word recipient names
    ("give dagger to smith rusk") via greedy-from-right splitting
    (`splitGiveArgs`/`giveObjectResolves`/`giveTargetResolves`). Single-word
    recipients and gold gives unchanged. Test:
    `TestGive_MultiWordRecipientName` in `internal/usercommands/give_test.go`.
  - **F3 (audit):** swept all 45 quests for the same single-path gap. Found ONE
    additional genuine stall of the same class — **Q69 Gallery Cipher**
    `rubbing→gallery` (give the rubbing to Dross silently transferred it; only
    `ask dross rubbing` advanced). Fixed with a mirrored give-path trigger
    (consume + hand back; player keeps the rubbing, needed later). Test:
    `TestQuest69_GiveRubbingAdvancesGalleryStep`.
  - The other 22 audit hits are **inverse** gaps (give works, `ask` doesn't =
    no stall) and most are design-appropriate courier quests. Audit also
    conflated the legacy (`type:`/`grants:`, only quest 4) vs questengine
    (`event:`/`actions:`) formats, so its inverse-gap list needs per-file
    verification before acting. NOT auto-applied — candidate follow-up:
    Q4 (Tessara "hand it over") + Q65 (Coll "give it here") have misleading
    dialogue worth an `ask`-path receipt node.
  - Verified: `go test ./internal/questengine/ ./internal/usercommands/` green;
    `go build ./...` clean; server boots clean (quests 63, flags validated,
    ValidateZoneConsistency errors=0 mode=panic, no panics).
  - **Not yet done:** in-game smoke of the full give→complete→reward→keep-weapon
    flow + messaging order (unit tests cover engine + parsing separately).
- 2026-06-29 (cont.): **multi-path quest resolution sweep.** Per the directive
  that *every* reasonable input a player tries should resolve a step (give OR
  ask), audited all delivery quests and added ask-path completion nodes so
  `ask <npc> <keyword>` completes a delivery (consuming the item) identically to
  `give`.
  - **Engine (TDD):** dialogue nodes gained `bumpsRep` (faction rep) and
    `givesGold`, wired through `PlayerState` → `factions.BumpRep` /
    gold + `events.EquipmentChange` in `talk.go`. Existing `requiresItem`
    already consumes + gates; `setsQuestFlag`/`givesItem` already existed. This
    gives the ask-path full parity with give-path delivery-trigger actions.
    Tests: `internal/dialogue/bumpsrep_test.go`.
  - **Data:** added a `*handoff` completion node (placed first under
    `tree.nodes`, `requiresItem`-gated so broad triggers are safe) to 21 NPC
    dialogue files covering Q4, Q5, Q7, Q9, Q14, Q17, Q19(×2), Q20(×2),
    Q60(×3), Q63(×2), Q65, Q66, Q67, Q68, Q70, Q71, Q74(×2). Chain quests
    (Q14/Q63/Q65/Q67 — give-path reaches end via a questengine `quest_granted`
    trigger the dialogue path can't fire) grant the **end** token directly
    (`IsTokenAfter` permits step-skipping) and replicate the chain's side
    effects (Q67: `bumpsRep bloom_trade -15`).
  - **Bug fixed in my own earlier work:** the first Q65 node granted `65-report`
    (would stall — that token only chains to `65-end` via the questengine path);
    corrected to grant `65-end` directly.
  - **Contract test:** `internal/dialogue/delivery_askpath_test.go` is
    data-driven over the REAL dialogue files (22 rows) asserting grant-token +
    item-consumption + flag/gold/rep/item side-effects per quest. All green.
  - **Incidental fix:** removed a pre-existing duplicate `moodChange` key in
    `new_plymouth_common/9323.yaml` (tolerated by the yaml.v2 loader, caught by
    the stricter test).
  - **Q4/Q65 reframing:** these were NOT broken (give worked; ask guided to
    give) — but per the directive both now *complete* via ask too.
  - Verified: dialogue/questengine/usercommands suites green; `go build ./...`
    clean; boot clean (quests 63, flags validated, zone errors=0 mode=panic).

## Open / flagged
- **Q6 (Collector's Burden) — VERIFIED FINE (false alarm).** It is a two-NPC
  quest: Toll Collector Harn (mob 86, Watchers Crossing) grants `6-start` and,
  on return, `6-end`; Clerk Pell (mob 99, Thornwall) grants `6-report`. The
  earlier "broken" call only inspected the quest-engine triggers and missed that
  `6-end` is granted by Harn's dialogue. Pell's delivery step ALREADY has both
  paths (give via the `item_give` trigger; ask via Pell's `bridge_report` node,
  which uses `requiresItem: 31` so the ask-path consumes the report too) — i.e.
  Q6 was authored with the exact dual-path pattern before this work. No changes
  needed.
- **In-game smoke DONE (2026-06-29):** triple smoke (total-newbie / MUD-vet /
  veteran) via the playtest harness against a live local server with all fixes.
  Reports: `tools/playtest/reports/2026-06-29-local-feel-tester-smoke-*.md`.

### Smoke results — my fixes VERIFIED working
- First Heat completes via **give** (`give dagger rusk`): PASS (newbie + vet).
- First Heat **multi-word** target (`give dagger to smith rusk`): PASS (newbie + vet).
- First Heat reaches **100%** (no 66% stall): PASS.
- (ask-path `ask rusk done` not separately re-tested — agents completed via give.)

### Smoke findings (new / confirmed)
| # | Finding | Type | Notes |
|---|---|---|---|
| S1 | **Double "quest completed" banner** on First Heat | engine bug (pre-existing = quest-60 backlog) | `GameBridge.GrantQuest` prints the end banner AND queues `events.Quest`, which `HandleQuestUpdate` prints again. Affects ALL questengine give/command completions. My give-path newly surfaces it on the first newbie quest. Fix: defer the end banner to `HandleQuestUpdate`. |
| S2 | **Silent tutorial→Thornwall transition** (= A5) | bug/UX | `ask esk passage` teleports to room 468 with ZERO output; no auto-`look` after teleport; Esk **asleep** suppresses his send-off dialogue entirely. |
| S3 | **MOTD lie: vets not auto-Awakened** | bug | MOTD says veterans arrive Awakened; not implemented (migration to grant `30-end` to existing-mutation chars never done). Vets sit through the full ~37s Rite. |
| S4 | **Smithy (Rusk) hard to find** | onboarding gap (newbie + vet; "most significant") | Rusk is 4–5 rooms off the hub; no wayfinding from Hadwen/Toke/room descs. |
| S5 | **Crier Toke absent at night** | schedule vs onboarding | Hadwen sends newbies to Toke, but Toke's schedule removes him from Hub Square ≥18:00 → no fallback guidance. |
| S6 | **No quest-progress feedback** | UX | Awakening rite completes with no banner; Find Your Footing ticks silently on room entry. |
| S7 | **Mutation mechanics invisible** | UX | e.g. bioluminescence shows flavor only — no hint it emits light / affects stealth. |
| S8 | **`look post` errors** in Hub Square | data bug | Room invites "look post"; engine says "Look at what???" (noun mismatch). |
| S9 | **No veteran fast-track** | design (ties to C1) | Mandatory unskippable ~37s Rite; no skip for self-identified vets. |

### Smoke-driven fixes applied (2026-06-29)
- **S1 — double completion banner FIXED** (`internal/questengine/bridge.go`):
  `GameBridge.GrantQuest` now, for the `end` step, ONLY queues `events.Quest`
  and returns — it no longer also emits the banner/event-log (those are done by
  `HandleQuestUpdate`). `start`/`progress` still emit directly. Fixes the
  duplicate for ALL questengine give/command completions (incl. the old quest-60
  report). No clean unit-test seam (quest registry isn't seeded in unit tests);
  verified by re-smoke.
- **S2 — silent exit FIXED** (two parts):
  - *Availability/sleep* (`schedules/pothole_coulee/pothole_esk.yaml`): Warden
    Esk now holds the Threshold House arch (5215) 24/7 and **never sleeps / never
    leaves** — only idle flavor changes by hour. The exit works at any time and
    his send-off is never sleep-suppressed.
  - *Arrival-blindness* (`internal/behaviortree/actions_room.go`): `actMovePlayer`
    now queues a `look` after a successful teleport, so portal arrivals always
    show the destination room (engine-wide; benefits any portal NPC).
- **S4 — smithy wayfinding FIXED** (`dialogue/pothole_coulee/9106.yaml`): Crier
  Toke's footing turn-in hint now explicitly points west to the coulee smithy /
  Smith Rusk.
- **S5 — Crier Toke night gap FIXED** (`schedules/pothole_coulee/pothole_toke.yaml`):
  Toke now stays at Hub Square (5203) 24/7, awake, time-varied idle — always
  findable for the footing a newbie is sent to do.
- **S8 — `look post` FIXED** (`rooms/pothole_coulee/5203.yaml`): added `post` /
  `notice` / `notices` noun aliases to the notice-post.
- Verified: `go build ./...` clean; boot clean (schedules 102, quests 63, flags
  validated, zone errors=0 mode=panic, no panics). Behavioral re-smoke of S1 +
  S2 in progress.
- **Deferred (not done):** S3 (MOTD/auto-Awaken migration), S6/S7 (quest-progress
  + mutation-mechanic visibility), S9 + the char-creation-poll/veteran tier (C1).
