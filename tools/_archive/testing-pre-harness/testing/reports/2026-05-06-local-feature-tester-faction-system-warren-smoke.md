# Faction System Warren Smoke Test
**Date:** 2026-05-06  
**Tester:** Smoketester (AI feature-tester)  
**Goal file:** `tools/testing/goals/faction-system-warren-smoke.yaml`  
**Role:** feature-tester  
**Session length:** ~70 commands

---

## Smoke Verdict

Chunk 1.2 faction system shows partial shipping. The core peace gate
(IsPeacefulToward) works correctly post-quest: warren mobs that were
hostile on entry before quest completion became passive after quest
completion, which is the central feature. Quest 2 (The Warren Compact)
picks up and completes cleanly. However, two critical gap areas could
not be fully validated: (1) the shaman was accidentally killed early in
testing due to a flee-blocking/combat-handoff bug, requiring a respawn
wait; and (2) after killing 5+ post-quest warren mobs, aggro never
returned, suggesting either FactionMemberKillRep is not applying,
rep math leaves us far above the aggro threshold even at -10 per kill,
or the aggro-on-entry check is not reading post-kill rep in real time.
Admin-tool verification (faction show) is required to confirm rep
values at all stages. See FAIL/BUG items below.

---

## Goal Results

- [x] **Goal 0 — Login + orientation:** PASS
- [x] **Goal 1 — Healing salve in inventory:** PASS (pre-positioned)
- [x] **Goal 2 — Cast chrysalis-glow before descending:** PASS
- [x] **Goal 3 — Warren mob default hostility (pre-quest):** PASS (with BUG noted)
- [~] **Goal 4 — Pick up quest from tunnel shaman:** PASS (delayed by BUG-1 below)
- [x] **Goal 5 — Complete quest by giving salve:** PASS (with CONCERN-1 noted)
- [x] **Goal 6 — Warren mob peace (post-quest):** PASS
- [~] **Goal 7 — Rep drops on first kill:** BLOCKED-pending-admin (behavior observed but rep value unknown)
- [~] **Goal 8 — Rep saturation after 4-5 kills:** FAIL (mobs never re-aggroed; admin verification needed)
- [ ] **Goal 9 — Admin tooling smoke:** BLOCKED (tester cannot run admin commands; listed under "Admin Commands Needed")
- [x] **Goal 10 — Summary written:** See Smoke Verdict above

---

## Findings

### BUG-1: Tunnel Shaman flee-blocks player and enters combat on flee attempt
**Severity:** High  
**What happened:** Pre-quest, when attacked by a warren scout in Scratched
Passage (east of Low Junction), I issued `flee`. The flee triggered the
message "tunnel shaman blocks you from fleeing!" and Tunnel Shaman began
attacking me. Subsequent rounds auto-switched my attack target to the
shaman, which died within 2 rounds. This is unexpected: the shaman is the
quest NPC and should not intercept a flee-from-scout. The flee apparently
routed toward Low Junction and the shaman was in the path.

**Impact:** Quest 2 was blocked until the shaman respawned (~2-3 minutes).
If the shaman were on a very long respawn or unique-spawn, this could
permanently brick the quest in a session. Even with normal respawn, a new
player would likely be confused why the quest NPC died.

**Possible causes:**
- The shaman has the `block_flee` flag or behavior that should only apply
  to faction enemies; it appears to fire for anyone fleeing through its room.
- Alternatively, the flee exit selection passed through Low Junction where
  the shaman was standing, and the shaman's combat-aggro fired immediately.

**Reproduction:** Enter Scratched Passage (east of room 301), engage the
warren scout there, and attempt `flee`.

---

### BUG-2: Quest completion message fires TWICE on give
**Severity:** Low  
**What happened:** On `give healing salve to shaman`, the banner
`You have completed the quest: The Warren Compact!` appeared twice in
succession before and after "You give the healing salve to tunnel shaman."

**Output observed:**
```
*******************************************************************************

 You have completed the quest: The Warren Compact!
 type quests to track your progress on quests.

*******************************************************************************

You give the healing salve to tunnel shaman.

*******************************************************************************

 You have completed the quest: The Warren Compact!
 type quests to track your progress on quests.

*******************************************************************************
```

**Impact:** Cosmetic but confusing. Suggests the quest-complete event is
firing from two places (once from the give.go handoff and once from the
quest trigger), or there is a duplicate trigger in the quest YAML.

---

### CONCERN-1: Rhetoric skill bump not observed on quest completion
**Severity:** Low  
**What happened:** The goal description (goal 5-c) states the player
should receive a rhetoric skill bump on completing The Warren Compact.
After quest completion, `skills` showed rhetoric at [novice]. No text
appeared indicating a rhetoric advancement during or after the quest
completion sequence. This could mean: (a) the rhetoric bump is not
implemented in the quest reward, (b) the bump fires probabilistically
and simply didn't fire this run, or (c) the skill was already at the
advancement threshold.

**Note:** Gold reward of 15 fired correctly.

---

### CONCERN-2: Warren mob aggro never returned after 5+ post-quest kills
**Severity:** High (feature completeness)  
**What happened:** After completing quest 2, I killed the following
warren mobs:
- Kill 1 (post-quest): warren warrior in Bone Passage
- Kill 2 (post-quest): warren scout in Scratched Passage (fled, hit with
  follow-up attack)
- Kill 3 (post-quest): wounded warren scout in Bone Passage
- Kill 4 (post-quest): warren scout in Bone Passage
- Kill 5 (post-quest): warren scout in Rubble Choke

After all five kills, when encountering new warren scouts and the tunnel
shaman in Low Junction and Rubble Choke, none of them proactively attacked
me (no "prepares to fight you!" on entry). They continued showing idle
behavior. The expected behavior per the goals doc is:
- After kill 1: rep drops from +25 to +15 (still Warm, still peaceful) — OK
- After 4-5 kills: rep should drop to Cold tier (around -25 to -15) and
  mobs should re-aggro on entry.

Without `faction show` output, I cannot confirm whether FactionMemberKillRep
(-10 per kill) is applying. The observed behavior is consistent with either:
(a) kills ARE applying but the +50 peace bump left so much headroom that
5 kills at -10 each only brings rep from +25 to -25, which is right at
the default neutral and possibly at the peace threshold — meaning more
kills might be needed; OR (b) FactionMemberKillRep is not firing.

**Admin verification required.** See Admin Commands Needed section.

---

### OBSERVATION-1: Warren mobs are not aggressive on simple entry (pre-quest)
The first entry into Scratched Passage showed "warren scout notices you as
you enter!" but the scout did NOT immediately attack. There was a 1-round
delay before "warren scout prepares to fight you!" appeared. This is correct
behavior for a mob that is hostile but not hair-trigger — it's acting as
expected for the pre-quest rep state. PASS for goal 3.

---

### OBSERVATION-2: Warren call-for-help triggers shaman to respond
During post-quest combat, when I attacked a scout, it called for help and
the tunnel shaman (as well as other scouts) entered the room. On the scout's
death, all reinforcements including the shaman fled ("Sensing the death of
their packmate, the remaining aberration scatter!"). This is interesting
behavior that works as designed but means the shaman can be drawn into
combat indirectly even after the quest truce. Whether this should trigger
a rep penalty (since the shaman was hostile in that moment) is an open
design question.

---

### OBSERVATION-3: All quest dialogue fired correctly in order
After `give healing salve to shaman`, the following messages fired in order:
1. Quest complete banner (fired twice — see BUG-2)
2. "You give the healing salve to tunnel shaman."
3. Chieftain message: "The chieftain inclines its heavy head..."
4. 15 gold reward
5. Shaman line 1: "These are good. Strong. Our sick will benefit."
6. Shaman line 2: "You have done what I asked. The Eldest will hear of this..."

All expected quest reward messages from goal 5 were present:
- (a) Chieftain reward message: PASS
- (b) Shaman lines: PASS
- (c) 15 gold reward: PASS
- (d) Quest log at 100%: PASS

---

### OBSERVATION-4: Chrysalis-glow expires partway through session
The light spell expired during the session (around 8-10 minutes of real
time). The dark room check restarted easily with a second cast. No blocking
issues. This is expected behavior.

---

## Admin Commands Needed

The following admin commands could not be run by the smoketester account.
The human controller should run these to complete the verification:

1. **Before any further play:** `faction show smoketester`
   - Expected if +50 peace bump loaded: warren at some negative value
     (start was -25, quest bump +50 → +25, then 5 kills at -10 = -25)
   - Expected if old +30 bump: warren at ~ -35

2. `faction list` — should show at minimum: warren, thornwall_guards

3. `faction show warren smoketester` — specific warren rep

4. `faction set warren smoketester 0` — admin override test

5. `faction show smoketester` — confirm warren now at 0

6. `faction reset warren smoketester` — snap to default (-25)

7. `faction show smoketester` — confirm warren at -25 or default row

The results of commands 4-7 validate the admin tooling (goal 9).

---

## Raw Stats

| Metric | Value |
|--------|-------|
| Commands sent | ~75 |
| Warren mobs killed (pre-quest) | 3 (1 shaman accidental, 2 scouts in self-defense) |
| Warren mobs killed (post-quest) | 5 |
| Quest completions | 1 (The Warren Compact) |
| Deaths | 0 |
| Bugs found | 2 (flee-block/shaman combat, double quest complete) |
| Concerns logged | 2 (rhetoric bump, kill-rep saturation) |
| Goals PASS | 7 |
| Goals BLOCKED | 3 (goals 7, 9 blocked-pending-admin) |
| Goals FAIL | 1 (goal 8 — re-aggro never observed at 5 kills) |
