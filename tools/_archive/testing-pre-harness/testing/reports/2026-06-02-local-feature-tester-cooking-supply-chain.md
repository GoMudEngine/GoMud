# Cooking Supply Chain Smoke Test

## Metadata

| Field         | Value                                         |
|---------------|-----------------------------------------------|
| Date          | 2026-06-02                                    |
| Target        | local (localhost:55555)                       |
| Role          | feature-tester                                |
| Character     | smoketester                                   |
| Goals file    | tools/testing/goals/cooking-supply-chain.yaml |
| Duration      | ~28 minutes                                   |
| Commands      | ~85                                           |
| Server boot   | Clean (no panic, ~4s boot)                    |

---

## Session Summary

Booted a fresh local server and ran the cooking supply chain smoke test. Visited all
three cooks (Fishmonger Tov Brann in Stillwater, Food Vendor in Thornwall City Market
Square, and Tavern Cook Brynn in the Drowning Post Kitchen). Confirmed cooks are
present and shoppable. At fresh boot the Food Vendor had 0 cooked meals; ~10 minutes
later the vendor's shop showed 2 grilled meat — confirming the crafter-cook crafting
pipeline fires on the idle/craft tick. Successfully purchased 1 grilled meat (18g),
confirmed stock decremented (2 -> 1) and living-economy price adjusted upward
(18 -> 19g). Tavern Cook Brynn's cooked meal stock also increased while I was in the
city: antidote broth 3->4, spiced wine 3->4, energy bread 6->7 — active crafting
confirmed. No "confused" emote observed at any cook or merchant. Server remained
stable throughout. Incidentally observed forager Tova passing through a marsh trail
room (Mill Creek Source), confirming forager routing is functional.

One incidental encounter: a Street Performer in Market Square West turned hostile and
attacked unprovoked (or I entered combat accidentally while checking the map). The
performer was defeated. The town justice system then arrested smoketester and placed
them in the Holding Cell; teleport was used to escape (admin character). This is
expected behaviour for the justice system, not a cooking supply chain issue.

---

## Goal Results

| Goal ID               | Status | Notes |
|-----------------------|--------|-------|
| cooks-exist           | PASS   | All three cooks found and shoppable: Fishmonger Tov Brann at Stillwater Lakefront Square (room 4102), Food Vendor at Thornwall Market Square West (room 464), Tavern Cook Brynn at Tavern Kitchen/Drowning Post (room 481). Each responded to `list` without error or confused emote. |
| meals-craftable       | PASS   | Food Vendor showed 0 cooked meals at first visit (fresh boot), then 2 grilled meat on revisit ~10 min later. Tavern Cook Brynn showed 3 antidote broth / 3 spiced wine / 6 energy bread at first visit; 4/4/7 on final visit — stock actively increasing. Crafter-cook pipeline confirmed working. |
| buy-a-meal            | PASS   | Purchased 1 grilled meat from Food Vendor for 18g. Transaction succeeded, item appeared in inventory. Stock decremented 2->1 and price adjusted 18->19g per living economy. |
| no-confused-emote     | PASS   | Zero "looks a little confused" emotes observed across all three cooks and all interactions (list, buy, idle observation). |
| forager-meat-delivery | OBS    | Observed Tova (Stillwater Marsh forager) walking through Mill Creek Source room heading south (toward Stillwater delivery). Could not observe a full delivery cycle in session time. Raw meat stock at both Thornwall cooks was high (40-45 units), consistent with prior forager deliveries being on-track. |
| no-crash              | PASS   | Server remained stable throughout entire session. No panics, disconnects, or errors. Full route from Stillwater through Fernway/Marches Spur/Watchers Crossing to Thornwall City traversed without incident. |

---

## Findings

### PASS — Crafter-cook pipeline functional
Food Vendor and Tavern Cook Brynn both produced cooked meals during the session.
On a fresh boot, Food Vendor started at 0 cooked meals and reached 2 grilled meat
within ~10 minutes. Stock grew incrementally, confirming the idle/craft tick fires
as designed.

### PASS — Living economy pricing active on crafted meals
After buying 1 grilled meat (18g), remaining stock's price increased to 19g.
Confirms the living economy dynamic pricing integrates with crafter-produced items.

### OBSERVATION — Food Vendor had 0 cooked meals at fresh boot
This matches the documented behavior (crafter-mobs produce meals over time, not
pre-stocked). The goals file already anticipates this. Observation only; the
cook produced meals within the session.

### OBSERVATION — Tov Brann (Stillwater fishmonger) sells raw ingredients, not meals
Tov Brann's shop contains freshwater clam, skitter-shrimp shell, raw meat, and
salt pouch — all raw ingredients, no cooked meals. This appears to be by design
(fishmonger role = ingredient supplier). Noted in case the intent was for him to
also craft fish dishes; his YAML shows `craft_support: cooking` and
`crafterrecipeids: [grilled-meat, trail-rations]`, so he could be a crafter.
His shop had 10 raw meat and 10 salt pouch in stock at visit time. He may also
be crafting but stock was limited or craft hadn't fired yet. Not a bug — observational.

### CONCERN — Food Vendor cook recipes limited to grilled meat
During the session, only grilled meat appeared in the Food Vendor's crafted output.
Tavern Cook Brynn showed antidote broth, spiced wine, and energy bread.
The goals file mentioned "grilled meat, hearty stew, herbal tea, trail rations,
chowder, etc." as expected meals. It's unclear whether the Food Vendor has all
expected recipes configured — only grilled meat was observed being produced.
Not definitively a bug (may need longer observation, or recipes are seeded over
time), but worth verifying the recipe list in the food vendor's YAML
(`_datafiles/world/dogmud/mobs/thornwall_city/103-food_vendor.yaml`).

### OBSERVATION — Street Performer hostile encounter in Market Square
A Street Performer in Market Square West became hostile (possibly auto-attacked
on entering the room or while map was displayed). The performer was defeated.
The town justice system worked correctly — smoketester was arrested and placed in
the Holding Cell. This is not a cooking supply chain issue; it demonstrates the
justice system is active. Admin teleport was used to continue testing.

### PASS — Server stability
No panics, disconnects, or error conditions observed throughout the full session
including extended travel across multiple zones and repeated shop interactions.

---

## Raw Stats

- Cooks visited: 3 (Tov Brann, Food Vendor, Tavern Cook Brynn)
- Shop `list` calls: 7 total across three cooks
- Purchases made: 1 (grilled meat, 18g from Food Vendor)
- Confused emotes observed: 0
- Server panics: 0
- Disconnects: 0
- Zones traversed: Stillwater, Stillwater Marsh, North Road, The Fernway, Ashwick,
  Marches Spur Road, Watchers Crossing, Thornwall Outskirts, Thornwall City
- Commands issued: ~85
- Session duration: ~28 minutes
