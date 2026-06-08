# Chunk 5.4 — NPC Market Participation Smoke Test Report

## Metadata

| Field         | Value                                                           |
|---------------|-----------------------------------------------------------------|
| Date          | 2026-06-02                                                      |
| Target        | local (localhost:55555)                                         |
| Role          | feature-tester                                                  |
| Character     | smoketester (AI, admin role)                                    |
| Goals file    | tools/testing/goals/5.4-npc-market-participation.yaml          |
| Duration      | ~35 minutes                                                     |
| Commands sent | ~80 (estimated)                                                 |

---

## Session Summary

Server booted cleanly in ~3.6s. Bridge connected. Character was already
positioned at Brindle's Smithy (Stillwater room 4106) from a prior session.

Early in the session, Guard Captain Velk (Thornwall City) appeared and attacked
the smoketester — apparently a carry-over crime record from a prior session.
This cost ~15 commands fleeing and navigating back. Used `teleport` admin
command to return to Stillwater and Thornwall City areas for testing.

All primary sell-path goals were successfully tested. Two bugs were identified:
one messaging regression (wrong error message for quest items) and one UX
concern (same misleading "no merchant here" message for overstocked or
wrong-category merchants).

No server crash, no panic, no "looks a little confused" emote observed.

---

## Goal Results

### Player sell-path regression (PRIMARY)

- [x] **PASS — sell-single**: Sold one clay flask to Apothecary Ilsa (room 4125,
  craft_support: alchemy). Response: "You sell a clay flask for 2 gold." Correct
  format. Also tested with Brindle (blacksmithing): "You sell a iron ingot for 2
  gold." Both pass.

- [x] **PASS — sell-multi**: Tested `sell 2 clay flask` → "You sell 2 clay flasks
  for 4 gold." Tested `sell all clay flask` (3 flasks) → "You sell 3 clay flasks
  for 6 gold." Tested `sell all iron ingot` (5 ingots) → "You sell 5 iron ingots
  for 10 gold." Tested diku-style `sell all.iron ingot` → "You sell 2 iron ingots
  for 4 gold." All correct. Bartering skill advancement fired on multi-sell.

- [x] **FAIL — sell-quest-item**: Obtained tally stick (item 40035, questtoken:
  "14-explore") via `item spawn 40035`. Attempted `sell tally stick` with both
  Brindle (room 4106) and Ilsa (room 4125) present. Response in both cases was
  "There's no merchant here." — NOT the expected "Quest items cannot be sold!"
  See BUG-1 below.

- [x] **PASS — sell-no-merchant**: In the Temple Sanctuary (room near 4125, Priest
  Seren present but no shop), `sell clay flask` → "There's no merchant here."
  Correct.

- [x] **PASS — sell-then-buy-back**: After selling clay flasks to Ilsa, `list`
  showed clay flask count increased in stock. After selling iron ingots to Brindle,
  iron ingot count in stock increased. After selling clay flask and raw meat to
  Fence Dealer Siv, both items appeared in Siv's `list` as new entries (qty 1,
  price 5 each). Sold items consistently entered merchant stock.

### Stability

- [x] **PASS — no-crash**: No server crash, no disconnect, no panic in server log.
  No "looks a little confused (sell all)" emote observed from any merchant or mob.
  Server ran cleanly for the full session.

### Mob-market observation (opportunistic)

- [x] **PASS — observe-mob-merchant**: Brindle's Smithy stock changed between first
  visit (10:00ish) and second visit (~10:20): iron ingot 80→100, wooden plank
  80→100, leather strip 68→74, steel ingot 72→80, coal dust 72→80. This is
  direct evidence that forager mobs are restocking Brindle's smithy during the
  session — the NPC market participation pipeline is running. Fence Dealer Siv
  was also observed (craft_support: general). Siv's stock absorbed a clay flask
  and raw meat sold by the player.

- [ ] **BLOCKED — observe-thief**: No thief-archetype mobs (Thornwall Highwayman
  mob 90, smugglers) were spawned during the session. `locate highwayman`,
  `locate thug`, and `locate smuggler` all returned no results. Could not observe
  mob-side sell behavior or fence-routing. No "looks a little confused" emote
  seen — absence of spawn means absence of the failure mode, not a pass.

---

## Findings

### BUG-1: Quest items produce wrong error message when sold

**Severity**: Medium (incorrect player feedback, but item correctly blocked)

**Reproduction**: `item spawn 40035` (tally stick, questtoken: "14-explore"),
then `sell tally stick` with any merchant present (tested with Brindle and Ilsa).

**Observed**: "There's no merchant here."
**Expected**: "Quest items cannot be sold!"

**Root cause**: `resolveMerchant()` in `internal/actions/sell.go` evaluates
merchant willingness using the probe item. `shops.EvaluateBuyRules` in
`internal/shops/buyrules.go` (line 42-44) checks `QuestToken != ""` and returns
empty BuyOffer. This causes `probeValue == 0`, so `resolveMerchant` returns nil,
and `usercommands/sell.go` emits `SellStopNoMerchant` → "There's no merchant
here." The explicit "Quest items cannot be sold!" path in `sellOneToMerchant`
(line 199-203) is only reached in the legacy-shop path (`shopInv == nil`) and
even then only if VendorCategories match. For all living-economy shops, quest
items are silently rejected by EvaluateBuyRules before reaching the correct
error path.

**Files**: `internal/actions/sell.go`, `internal/shops/buyrules.go`,
`internal/usercommands/sell.go`

---

### CONCERN-1: "There's no merchant here" fires for category-mismatch and overstock

**Severity**: Low (misleading UX, not a regression per se)

**Observation**: When selling clay flask (vendor_categories: alchemy) to Brindle
(craft_support: blacksmithing), the response is "There's no merchant here." — the
same message as when no merchant exists at all. Same message fires when stock is
at MaxStock (100). A merchant IS present in both cases, but won't buy the specific
item. This message has existed prior to 5.4 (same `resolveMerchant` mechanism).

**Suggestion**: `resolveMerchant` could be split into "find any merchant" and
"find a willing merchant" — the former would suppress the misleading "no merchant"
message for category-mismatch/overstock cases. Lower priority.

---

### OBSERVATION-1: Forager-to-vendor pipeline confirmed active

Brindle's stock levels changed materially between session start and ~20 minutes
later (iron ingot +20, wooden plank +20, leather strip +6, steel ingot +8, coal
dust +8), consistent with forager NPCs delivering materials during the session.
This is the core 5.4 mob-market participation feature running correctly.

---

### OBSERVATION-2: "sell all.item" diku-style syntax works

`sell all.iron ingot` correctly sells all matching items with the multi-sell
message. Functional parity with `sell all iron ingot`.

---

### OBSERVATION-3: Guard Captain Velk crime record persisted into session

The smoketester had an active criminal record from a prior session causing Guard
Captain Velk to attack immediately on session start at Brindle's Smithy (room
4106, which is in Stillwater but Velk is Thornwall). This was not a 5.4 issue
but consumed early session commands. The guard pursued the character across
teleports. No gameplay crash or lockup resulted.

---

### PASS: No "looks a little confused (sell all)" emotes

Zero occurrences of the "looks a little confused" emote from any merchant or mob
throughout the session. The primary regression concern for 5.4 (the actions.Sell
lift breaking mob sell-all) did not manifest. Server log also contained no related
errors.

---

## Raw Stats

| Metric                     | Value                    |
|----------------------------|--------------------------|
| Server boot time           | ~3.6 seconds             |
| Session duration           | ~35 minutes              |
| Commands sent              | ~80 (estimated)          |
| Server crashes             | 0                        |
| Server panics              | 0                        |
| "looks a little confused"  | 0                        |
| Sell commands executed     | ~15                      |
| Successful sells to Ilsa   | Single, 2-count, 3-count, all |
| Successful sells to Brindle| Single, 3-count, 5-count, all, all. |
| Successful sells to Siv    | clay flask, raw meat     |
| Quest item sell attempts   | 2 (tally stick at Brindle, tally stick at Ilsa) |
| BUGs filed                 | 1                        |
| CONCERNs filed             | 1                        |
