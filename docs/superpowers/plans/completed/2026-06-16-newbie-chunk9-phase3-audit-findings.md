# Chunk 9 Phase 3 — Audit Findings (Pothole Coulee)

**Date:** 2026-06-16. Final C9 phase: reward-balance consistency sweep +
hint-coverage audit + no-hard-numbers audit. Per the C9 sub-spec §3,
**difficulty/encounter tuning stays deferred** to the evening playtest —
this phase is consistency + discoverability + immersion only.

## 1. Reward-balance consistency sweep (quests 30–59)

Tabulated every `rewards:` block across the 30 newbie quests. The pattern
is consistent by quest position:

| Position | Norm | Notes |
|----------|------|-------|
| Hub | 30 = rite (0g), 31 = 20g | orientation |
| Spoke 1st (intro) | **15g** + skill:1 | |
| Spoke 2nd (mid) | **25g** + skill:1 + item | |
| Spoke capstone | **50g** + stat:3 + item/recipe/spell | |
| Repeatable (53–59) | **6g** + skill:1 | uniform small band |

### Outliers fixed (uniformity, zero difficulty impact)
- **32 First Blood** (A intro): 10g → **15g** (spoke-intro norm).
- **49 The Old Shrine** (F capstone): 40g → **50g** (capstone norm).
- **58 Reading the Stones** (repeatable): 5g → **6g** (repeatable band).

### Intentional asymmetries (left as-is — design, not bugs)
- **47 First Words** (F intro) grants `charisma:1` (a stat) where other
  intros grant a skill. Spoke F is the social/charisma spoke; front-loading
  charisma is thematic. Left.
- **49 The Old Shrine** (F capstone) gives `charisma:3` + gold only — no
  item/recipe/spell, unlike the other capstones (37/40/43/46/52). F is the
  "quiet discovery" spoke (the Orbital Stone lore + a narrative faction nod
  is the reward, not loot). Left intentionally; flagged for the evening
  playtest in case it reads as thin.
- **34 Take the Tower** (A capstone) grants a 2-skill bump
  (`weapon-combat:1,unarmed-combat:1`) and routes its +3 stat through the
  Garve "might/finesse" boon (a `train_stat` dialogue trigger), not a
  `stat_info` reward block. Equivalent to the other capstones' stat:3 by a
  different mechanism. Left.

## 2. Hint-coverage audit — PASS

- Every non-`end` step in the 7 repeatables (53–59) carries a `hint:`;
  `end` steps correctly omit hints (completion text). Verified by count.
- Each repeatable offer-node hints `ask <trainername> work` and includes
  `quest`/`task` in keywords+triggers (Quest NPC Dialogue SOP).
- The 30–52 spokes were hint-gated in their own chunks (manifest checker's
  noun-token rule + per-chunk review).

## 3. No-hard-numbers audit — PASS

- Zero digits in any C9-authored player-facing prose: repeatable quest
  text/hints/messages, the 8 hub schedule `say`/`emote` lines, and the 3
  conversation pair files. (Confirmed by a digit-scan of those fields.)
- Gold reward amounts (`gold: N`) are not prose — they surface via the
  standard "You receive N gold" loot message, the sanctioned numeric
  display (like a price). Not a violation.

## 4. Outcome
C9 Phase 3 = 3 one-line gold normalizations; both audits clean by
construction (the C9 content was built SOP-compliant). C9 (Polish) is
complete after this. Next: **C10 cutover.**
