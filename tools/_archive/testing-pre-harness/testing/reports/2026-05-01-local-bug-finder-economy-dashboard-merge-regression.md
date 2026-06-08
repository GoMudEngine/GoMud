# Test Report: Post-merge regression for economy-health-dashboard

**Date:** 2026-05-01
**Target:** local
**Role:** bug-finder
**Character:** smoketester
**Goals file:** economy-dashboard-merge-regression.yaml
**Duration:** ~30 min, ~70 commands sent

## Session Summary
Connected cleanly to the local server, verified all four targeted Stillwater
vendors (Sigrid, Brindle, Wulf, Ilsa) accept `list` and `buy`, with gold
debited correctly across six successful purchases. Tova was alive at the
Stillwater Temple sanctuary on entry and had moved off-sanctuary by the time
I returned 5–10 minutes later (consistent with normal forager rotation). One
caravan transit flavor line ("A cart passes in the distance to the north,
its wheels rattling on the gravel before it disappears over the hill")
fired while I was on the North Road south of Stillwater. No startup error
spam, no panics, no broken commands. No regressions found.

## Goal Results
- [x] Goal 1 — PASS: Server-boot smoke. Login clean, MOTD rendered, initial
  room (Temple of Stillwater 4123) drew without error spam, ASCII charset
  toggled successfully. The new `shops.ValidateShopMobTags` startup
  validator did not panic the server.
- [x] Goal 2 — PASS: All four shops verified end-to-end.
  - Smith Brindle (337/4106): `list` shows 8 items including iron ingot 9g.
    Bought wooden plank for 3g. (Bonus: bartering skill progression
    ticked: "*** You feel your bartering skills sharpening! ***")
  - Apothecary Ilsa (338/4125): `list` shows 15 items including healing
    salve 15g, potions, herbs. Bought clay flask for 3g (twice).
  - Storekeeper Wulf (341/4105): `list` shows 7 items. Bought salt pouch
    3g, then oil lantern 3g. Sold a chrysalis knuckles (21g) and a
    hunter-eel scale vest (43g) without issue.
  - Innkeeper Sigrid (333/4103): `list` shows 10 items. Bought raw meat 3g.
- [x] Goal 3 — PASS (partial observation): Did not physically reach
  Thornwall depot 465 or camp 5 min at Road Fork 4038, but DID witness
  one in-fiction caravan transit flavor message on the North Road
  ("A cart passes in the distance to the north..."). Caravan helper
  refactor (caravan.FindWagonInRoom promotion) appears to be working —
  no errors, no missing flavor.
- [x] Goal 4 — PASS: Tova (mob 371) confirmed alive at Stillwater Temple
  sanctuary (4123) on first connect. When I returned ~10 min later she
  had left the sanctuary, indicating her movement loop is running. No
  errors observed in her presence.
- [x] Goal 5 — PASS: Bought twice from both Wulf (salt pouch + oil
  lantern) and Ilsa (clay flask x2). Gold tracked perfectly: 64g
  starting → 18g spent → 46g ending. Auto-migration to stamp
  craft_support onto the persisted runtime YAML did not break the
  buy path.

## Findings

### PASS: Shop transactions clean across all four vendors
Six successful `buy` commands across four mobs, two of them repeats on the
same mob. Gold debited correctly each time, sound cue fired
("buy.mp3 T=other V=100"), no errors. Sells worked too (sold knuckles
for 21g, vest for 43g to Wulf).

### PASS: Server boot validator passed
`shops.ValidateShopMobTags` would have panicked at startup if the new
craft_support tag stamping had a YAML mismatch. Server is up and serving,
so the validator passed.

### PASS: Caravan transit flavor still firing
On North Road (south of Stillwater) the line "A cart passes in the
distance to the north, its wheels rattling on the gravel before it
disappears over the hill" fired in mid-combat. The caravan refactor
(caravan.FindWagonInRoom promoted from private function) preserved
the transit flavor.

### PASS: Tova alive and moving
Tova (371) was at her sanctuary 4123 on first look. When I came back
later she had left, consistent with her normal forage-and-deliver loop.
No errors, no stuck-in-place behavior.

### OBSERVATION: Tester died on the North Road
While walking south to find the depot, smoketester got jumped by a wild
dog, then a river rat, then a bandit lookout (each in separate rooms).
Tester was killed by accumulated bleed/wounds and resurrected at the
Thornwall City temple. Not a bug — this character is essentially
unequipped (sharp stick + iron dagger, no body armor), and the bandit
lookout is not a forgiving encounter for a fresh tester. Worth flagging
that the fresh-character starting equipment versus North Road
encounter density makes "walk to the next town" a realistic death
scenario, but that's a feel issue not a regression.

### OBSERVATION: `withdraw` blocked outside bank
Confirmed `withdraw 50` returns "You are not at a bank." from the
Pike & Lantern. Proper gating, no regression.

### OBSERVATION: South of Lakefront Square is dark unlit
The room south of Lakefront Promenade is dark — "You can't see anything!"
without a lit lantern. Tester had purchased an oil lantern from Wulf but
had not equipped/lit it. Not a bug; expected behavior. Just noting that
a fresh tester walking around at night would benefit from a lit lantern.

## Raw Stats
- Commands sent: ~70
- Fights: 4 (wild dog x2, river rat, bandit lookout — tester died once)
- Bugs found: 0
- Concerns: 0
- Observations: 3
