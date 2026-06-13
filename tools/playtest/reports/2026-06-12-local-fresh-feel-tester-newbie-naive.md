# Playtest Report — Naive New-Player Run, Pothole Coulee Newbie Hub (Chunk 1)

- **Date:** 2026-06-12
- **Target:** local-fresh (localhost:55555), brand-new character via new-player flow
- **Personality:** feel-tester
- **Goals file:** newbie-naive.yaml ("never played a MUD; rely only on what the
  game tells you")
- **Character:** Brannel (Novice Generalist, Human), created fresh this session
- **Charset:** ASCII (converged via `set charset`, confirmed)

## Session Summary (in order)

1. **Account/character creation.** Connected, chose `new`, walked the prompts:
   username (Brannel) -> password x2 -> email (skipped) -> screen-reader (n) ->
   confirm (y). Smooth, standard, no friction. Dropped directly in-world at
   **The Awakening Pool** (Pothole Coulee, room 5200) — no separate ghost/limbo
   step.
2. **Converged charset to ASCII.** `set charset` -> "Charset mode set to ASCII."
   HP/SP/CP bars switched from block glyphs to `#`. Confirmed once, did not need
   to repeat.
3. **The Awakening rite.** Cleric Hadwen greeted me and offered two paths in his
   first line: `ask hadwen about rite` (lore) or `ask hadwen begin` (action),
   with a bracketed UI hint. I asked about the rite (good lore on belief/
   mutation), then `ask hadwen begin`. The rite is a **scripted, auto-paced**
   sequence (fires on a timer regardless of input) that adjusts stats, delivers
   worldbuilding, and ends by naming **two clearly-signposted roads with exact
   commands**: `ask toke footing` (new players) or `ask esk passage` (the way
   out). It also name-dropped `renameself` and `help`.
4. **Find Your Footing quest.** `ask toke footing` granted the quest: three
   stops (Chrysalis School Hall, Coulee Provisions, The Drowned Lantern), walk
   in order, return to Toke. Quest tracker (`quests`) showed clear next-step
   text at each stage.
5. **Toured the hub.** North up the Basalt Stair / School Shelf to the School
   Hall (read the notice-board — diegetic `help`/`set linewidth` teaching;
   peeked the Cleric's Study dead-end). Quest auto-advanced to 40% **on room
   entry**. South to Market Row -> Coulee Provisions; saw the **"TIP: type list"**
   prompt, listed wares, **bought + wielded a Crude Short Blade and bought +
   wore a Cotton Shirt** (started with empty equipment). Buying advanced the
   quest to 60%. West to The Drowned Lantern (inn) -> 80%.
6. **Completed the quest.** Returned to Toke, `ask toke footing` again ->
   "Quest complete," **+20 gold**, and the **"seven mouths"** forward-hook
   (seven spoke paths: fighting / smithing / brewing / wilds / folding / lore /
   ranged). `ask toke seven mouths` + `map` for orientation.
7. **Explored the rest of the hub.** Stilt-House Walk (excellent mutation-
   acceptance flavor) -> Scrub Draw (one of the "mouths" — a graceful
   under-construction dead-end). Strongbox House (bank, Ledger-Keeper Croup).
   The Threshold House (Warden Esk + the humming basalt arch).
8. **Tested the exit.** `ask esk passage` — Esk activated the arch and
   **transported me to Thornwall City (Temple Interior, room 468)**. This works
   as designed and as Hadwen advertised. Ended the session here (Thornwall is
   the existing main city, outside chunk-1 scope).

## Findings

### PASS

- **PASS — Character creation.** Clear, quick, standard prompt chain. A first-
  timer would not get stuck.
- **PASS — The Awakening opening is a model onboarding beat.** Hadwen offers
  lore-or-action up front, bracketed hints reinforce, the rite is atmospheric
  and self-pacing, and it ends by handing the player two explicit, command-
  labeled next steps. Strong.
- **PASS — Find Your Footing is forgiving and legible.** Steps auto-advance on
  room entry and on purchase; the `quests` tracker always states the next stop;
  completion gives gold + a clear forward-hook. A naive player cannot easily
  fail or get lost on this arc.
- **PASS — Newbie shop (Coulee Provisions).** "TIP: type list" surfaces the
  command; stock is cheap, basic, and complete (weapon/armor/food/potions/
  bandages/sling+shot, 1-10g; I had 250g). `list`/`buy`/`wield`/`wear` all work
  with text confirmation. A new player can fully kit out for a few gold.
- **PASS — Prose & worldbuilding.** Consistently high quality and on-theme
  (the Opened / mutation / belief-shapes-the-world). Stilt-House Walk's casual
  acceptance of mutated bodies is a standout immersive touch.
- **PASS — Seven mouths soft-gate.** Unbuilt spoke zones are signposted
  in-fiction ("country no map has caught up to yet", "someone is building the
  marker higher") rather than dead-ending abruptly — a player reads "come back
  later," not "broken."
- **PASS — Exit to Thornwall.** `ask esk passage` (the command Hadwen taught)
  transports cleanly through the arch into the main city. Progression out of the
  newbie zone is not blocked.

### CONCERN

- **CONCERN (minor, non-blocking) — "Look into the water" is not parseable.**
  During the rite Hadwen says "Look into the water," but `look water` returns
  **"Look at what???"** (neither `water` nor `pool` is a lookable keyword in
  5200). The rite advances on its own timer so it doesn't block, but the first
  concrete instruction the game gives teaches the new player that the obvious
  command fails. Repro: at The Awakening Pool during/after the rite, `look
  water` or `look pool`.

### OBSERVATION

- **OBSERVATION (polish) — NPC dialogue final lines drop terminal punctuation.**
  Crier Toke ("...Walk any of them") and Warden Esk ("...no count and no
  grudge") both end mid-sentence with a close-quote and no period. Reads like
  truncated strings. Two confirmed instances; likely a content pattern worth a
  sweep.
- **OBSERVATION (formatting seam) — name glued to narration in the rite.** One
  rite line rendered as **"Cleric Hadwen The light lets you go..."** — the NPC
  name is prepended to third-person narration with no "says"/punctuation. Looks
  like an emote/forced-narration formatting seam. Repro: run the rite (`ask
  hadwen begin`) and watch the scripted lines.
- **OBSERVATION (low confidence, possible client artifact) — leftover UTF-8 in
  ASCII mode + a garbled prompt tail.** After `set charset` ASCII, box-drawing
  and the HP/SP/CP bars converted correctly, but the **time-of-day glyph
  (sun/moon emoji), the inline room mini-map glyphs (≈ ⌂ ▼ @), and the `map`
  command's legend/body** stayed UTF-8. The status prompt also consistently ends
  in a garbled two-byte `??`/replacement-char sequence after the weather glyph.
  The mini-map/legend leakage is plausibly an ASCII-mode coverage gap (it only
  converts box-drawing); the trailing garbled bytes look like a genuine encoding
  glitch. Flagging low-confidence on a headless client — **confirm in a rich
  web/Mudlet client** before treating as a bug.
- **OBSERVATION — `ask esk passage` transports immediately.** Esk says "Walk
  through," implying a follow-up manual step, but the ask itself sends you to
  Thornwall. Minor expectation mismatch; Hadwen did teach `ask esk passage` as
  THE action, so it's discoverable.
- **OBSERVATION — `enter` is not a recognized command.** A naive player told to
  "walk through" an arch might try `enter arch` -> "Enter not recognized. Type
  help for commands." The arch only responds to the NPC-ask path, not to
  `enter`/`go arch`/`through` as a room feature. (Didn't matter here because the
  ask had already transported me.)

## What I Never Discovered (suspect exists, was never led to)

- **Combat — never taught.** Completing BOTH questlines, the tutorial never put
  a single enemy in front of me and never surfaced `attack`/`consider`/`flee`/
  specials. I armed myself only on my own initiative. A brand-new player who
  finishes the hub and steps through the arch into Thornwall is **combat-naive**
  — presumably the "fighting mouth" spoke handles this, but it isn't built yet.
- **The seven spoke zones** (fighting/smithing/brewing/wilds/folding/lore/
  ranged) — signposted, not yet walkable (expected for chunk 1).
- **Resting / the inn loft.** The inn has an `up` exit to a sleeping loft, but
  the tutorial never teaches `sleep`/rest or what the loft is for.
- **Banking.** Strongbox House + Ledger-Keeper Croup exist, but Find Your Footing
  never visits the bank or teaches deposit/withdraw. A new player wouldn't learn
  banking from in-game text alone.
- **`renameself` / `help`.** Surfaced only obliquely (Hadwen's farewell + the
  notice-board's flavor riddle). A literal first-timer might not connect "put
  the question plainly to the world" to the `help` command.

## Verdict — Core Question

**Could a brand-new player get through the questline(s) without help? YES.**

The two chunk-1 questlines (The Awakening, Find Your Footing) are completable
relying solely on in-game text. Guidance is genuinely excellent: every step
names the exact command, the quest tracker states the next objective, bracketed
hints reinforce, and quest steps auto-advance forgivingly. The hub is content-
complete, well-connected (clean Cartesian loop), and the prose is a cut above.
No blockers, no dead-ends that read as broken.

Two caveats keep this from a flawless grade: (1) the tutorial teaches navigation,
shopping, and quest-flow but **not combat, resting, or banking** — those are
deferred to unbuilt spokes, so a player who clears the hub and takes the arch to
Thornwall arrives game-literate about movement but naive about fighting; and
(2) a handful of minor polish nits (the unparseable "look water," missing
sentence-ending punctuation on a couple NPC lines, a name/narration formatting
seam, and ASCII-mode glyph leakage / a garbled prompt tail to confirm in a rich
client).

## Recommendations

1. Make `water`/`pool` lookable in The Awakening Pool, or reword Hadwen's line
   so the instruction matches a parseable command (the rite is on a timer, so
   the "look" is purely flavor — either honor it or don't ask for it).
2. Sweep NPC dialogue for final lines missing terminal punctuation (Toke, Esk
   confirmed).
3. Fix the "Cleric Hadwen The light lets you go" formatting seam in the rite
   script (missing `says`/emote framing).
4. Confirm the ASCII-mode glyph leakage (mini-map, `map` legend, weather glyph)
   and the garbled prompt-tail bytes in a rich client; if reproducible there,
   extend ASCII conversion past box-drawing or strip the stray bytes.
5. Consider a light combat primer somewhere in the hub (or at least a "you have
   no spells/no foe yet — the fighting mouth will teach you" pointer), so the
   arch to Thornwall doesn't dump a combat-naive newbie into a full city.
6. Optional: have Esk's `ask esk passage` either pause for a confirm/explicit
   step, or reword "Walk through" so the immediate transport isn't a surprise.

---
*Driver: /playtest local-fresh feel-tester newbie-naive.yaml. Session reached a
natural "done" after both questlines completed, the hub was fully explored, and
the arch transported out to Thornwall (main game, out of scope).*
