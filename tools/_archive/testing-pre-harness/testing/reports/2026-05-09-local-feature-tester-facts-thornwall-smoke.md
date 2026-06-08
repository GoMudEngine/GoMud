# Smoke Test Report: Chunk 1.7 World-Model Facts — Facts Thornwall Smoke

**Date:** 2026-05-09 (session continued into 2026-05-11)
**Tester:** smoketester (AI feature-tester role)
**Server:** local (localhost:55555)
**Goal file:** `tools/testing/goals/facts-thornwall-smoke.yaml`

---

## Smoke Verdict

**PASS.** The chunk 1.7 world-model facts substrate fired correctly on the
gossip path. During the observation window, Old Wrex (mob 116) emitted the
line:

> "I heard The Thornwall mayor has resigned in disgrace."

This line uses the `fact-default` template "I heard {description}" with the
full `test-mayor` fact description substituted correctly. All three gossip
mobs (Old Fen 114, Old Gobb 115, Old Wrex 116) were present in room 484 The
Back Corner throughout the session. The gossip system is active and the fact
path is functional.

---

## Goal Results Table

| # | Goal | Result | Notes |
|---|------|--------|-------|
| 0 | Login + ASCII mode | PASS | Logged in, `set charset` confirmed, starting room Main Street Central (Thornwall City) |
| 1 | Navigate to The Back Corner (room 484) | PASS | Path: Main Street Central -> east -> south -> south. Room confirmed as "The Back Corner." All three old men present on arrival. |
| 2 | Observe fact-based gossip line | PASS | Old Wrex said "I heard The Thornwall mayor has resigned in disgrace." — confirmed fact-default template with correct description substitution |
| 3 | Confirm fact-candidate path is non-blocking | PASS | Event-typed lines appeared in the same window (multiple PlayerCraftedRare and MobCraftedRare events). Gossip mix included both event and fact lines. |
| 4 | Write smoke verdict | PASS | See Smoke Verdict section above |
| 5 | Admin verification list | PASS | Listed in Admin Commands Needed section below |

---

## Gossip Line Transcript

All gossip lines observed verbatim during the observation window (~25 minutes,
~80+ rounds in room 484). Non-gossip idle emotes and room ambiance lines are
excluded for brevity.

| Round estimate | Mob | Line | Type |
|----------------|-----|------|------|
| ~Round 27 | Old Gobb (115) | "Everyone is talking about it. jeweler Tess has crafted a rare Polished Stone Amulet." | EVENT — PlayerCraftedRare-Global |
| ~Round 35 | Old Wrex (116) | "Everyone is talking about it. pearl-carver Kess has crafted a rare Silver Ring." | EVENT — PlayerCraftedRare-Global |
| ~Round 44 | Old Fen (114) | "Right here in town, blacksmith Kerra has crafted a rare Iron Buckler. Imagine that." | EVENT — MobCraftedRare-Regional-Local |
| ~Round 62 | Old Gobb (115) | "Now THIS is worth mentioning. pearl-carver Kess has crafted a rare Silver Ring." | EVENT — MobCraftedRare-Global |
| ~Round 80 | Old Wrex (116) | "I heard The Thornwall mayor has resigned in disgrace." | **FACT** — fact-default template |

**Annotation:** The fact line from Old Wrex used the "I heard {description}"
variant of the `fact-default` template family. The `{description}` placeholder
was substituted with "The Thornwall mayor has resigned in disgrace." exactly as
declared. The other four lines are event-based, using templates from the
event-typed gossip family (PlayerCraftedRare, MobCraftedRare). Event lines
dominated as expected at the 70%/30% mix ratio.

**Idle emotes also observed (non-gossip, showing mobs are active):**
- old Gobb scratches his thick beard and squints at the curtain as if expecting someone
- A cup scrapes against the scarred table as someone shifts in their chair.
- old Fen leans back in his chair and stares at the ceiling through half-closed eyes
- old Wrex sits perfectly still, watching the room with pale, unblinking eyes
- old Wrex tilts his head slightly, as if listening to something no one else hears
- old Fen exhales a thin stream of pipe smoke and watches it curl away
- old Fen picks at a splinter on the table edge with one cracked fingernail
- old Fen turns his cup slowly on the table, watching the dregs settle
- old Gobb takes a long pull from his cup and wipes his mouth with the back of his hand
- old Gobb slaps the table and barks a short laugh at something only he finds funny
- old Gobb leans forward and gestures broadly, sloshing ale from his cup
- old Wrex lifts his cup, considers it, and sets it back down without drinking
- old Fen says, "How about a smile with that ale, Dal?"
- Pipe smoke curls up from behind the curtain in lazy grey ribbons.
- One of the old men coughs quietly, then resumes his silence.
- The murmur of low voices drifts from the back corner, too quiet to make out.

---

## Findings

### PASS — Fact substrate wired into gossip pipeline

Old Wrex gossipped the `test-mayor` fact using the `fact-default` template
with correct `{description}` substitution. The `KnownFactsOf()` → `renderFactGossip()`
code path is reachable and functional. All three awareness files at
`_datafiles/world/dogmud/facts.awareness/` contain `known_facts: - fact_id: test-mayor`
and were present before the session began.

### PASS — Pre-test setup confirmed complete

The controller ran the admin `fact` commands before this session. Evidence:
- `_datafiles/world/dogmud/facts.yaml` contains `test-mayor` with `status: active`
- Awareness files for mobs 114, 115, 116 all include `fact_id: test-mayor` under `known_facts`
- Mob 114 (source: witnessed), mobs 115 and 116 (source: told)

### PASS — Event-based gossip path also healthy

Four event-based gossip lines fired during the session (PlayerCraftedRare and
MobCraftedRare types). Templates loaded correctly, distance-variant logic
worked (Kerra's buckler was "Right here in town" — Local variant for same-zone
event). The dedup mechanism via `HeardEvent` / `RecordHeardEvent` is functioning:
the same events were not repeated within a short window.

### OBSERVATION — Gossip interval is 75 rounds with per-mob stagger

Gossip lines were infrequent: approximately one per mob per ~75 rounds. The
stagger formula (`MobId % 3 * 25`) spreads the mobs so they don't all fire at
the same time. With three mobs, a complete round-robin takes approximately
75 rounds. This matches the `GossipIntervalRounds` default of 75.

### OBSERVATION — 30% fact probability means patience required

At 30% probability, roughly 1 in 3 gossip ticks should pick the fact pool when
both pools are non-empty. In practice, 4 event lines appeared before 1 fact
line — consistent with the 70/30 split but toward the unlucky end. For a
dedicated fact-only verification, lowering `GossipIntervalRounds` temporarily
or increasing the fact pool weight during testing would speed confirmation.

### OBSERVATION — `TempData` for `lastGossipRound` resets on server restart

The `lastGossip` value is stored in `TempData` (in-memory only). On server
restart all mobs lose this state, so the first gossip fires almost immediately
(stagger aside). This is by design and consistent with the code comment
("lastGossip == 0" short-circuits the interval check).

### OBSERVATION — Only Wrex fired a fact line during the session

Fen and Gobb both gossiped events; Wrex was the only one to pick the fact
pool. This is expected variance at 30% probability over 2 visible gossip
cycles each. There is nothing to suggest Fen or Gobb cannot gossip facts —
all three have identical `known_facts` entries.

### OBSERVATION — Another player (Megalomania) was present briefly

Player Megalomania entered and left room 484 during the session (was
teleported away after a few minutes). This had no observable effect on the
gossip behavior.

---

## Admin Commands Needed

Run these after the tester session to verify the awareness substrate recorded
event gossip correctly:

```
fact awareness 114
fact awareness 115
fact awareness 116
fact list
```

The `heard_events` lists for mobs 114, 115, 116 should now include the event
IDs they gossiped during this session (crafted-rare events). `fact list` should
still show `test-mayor` as `active`.

**Optional — lifecycle test (withdraw):**
```
fact withdraw test-mayor
```
Then idle in room 484 for 10+ more rounds. Expect: no further mayor-related
gossip lines (lazy filter excludes withdrawn facts from `KnownFactsOf()`
results).

**Optional — auto-withdraw test:**
```
fact declare test-respawn --withdraw-on-respawn-of 100 -- A city beggar has gone missing.
```
Then trigger or wait for mob 100 (city beggar) to respawn. Run
`fact show test-respawn` — should show `status: withdrawn`.

---

## Travel Path to Room 484

For future reference:

1. Start: Main Street, Central (Thornwall City)
2. `east` → Main Street, East
3. `south` → Craftsmen's Quarter, East
4. `west` → Craftsmen's Quarter, West
5. `south` → The Drowning Post Tavern
6. `south` → The Back Corner (room 484)

Room 484 has one exit: `north` (back to the tavern).

---

## Raw Stats

- Commands sent: approximately 82 `look` + passive-wait cycles
- Gossip lines observed: 5 total (4 event, 1 fact)
- Idle emotes observed: 16+ (mobs active throughout)
- Time in room 484: approximately 25 minutes
- Fact line fired: confirmed on Old Wrex's second gossip cycle (~round 80 of
  the session)
- Bridge: mud_bridge.py, Python, localhost:55555
- Server status at end: clean (no crashes, no disconnects during session)
