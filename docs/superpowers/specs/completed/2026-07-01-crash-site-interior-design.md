# Crash Site Interior (#22) — design

*Date: 2026-07-01. Phase 7 — The Eastern Road (Endgame Approach), leg 3 — THE FINALE.*
*Canon: `docs/ZONE_EXPANSION.md` §Zone 7.3; `world.md` §The Crash Site / §The Moons.*

**Sub-project scope.** #22 decomposes into two sub-projects, each its own
spec→plan→build:
1. **The Disc Questline** (the on-ramp — a cross-zone quest tying the
   Pothole Coulee disc + the reckoning-bone + Maren/Ashwick into a usable
   disc key). **Separate spec — NOT this document.**
2. **The Crash Site Interior** — the finale zone itself. **THIS document.**
   Depends on sub-project 1 delivering a usable disc key.

## Purpose

The finale: the interior of the buried colony ship, where the game's
central mystery finally opens. A disc-gated, gold-scaled **instance**
(hybrid: a one-time story revelation layered over a repeatable
party-endgame trap-dungeon). It is the hardest content in the game, the
canonical source of the very-rare legendary-craft reagents, and the hook
into the future moons expansion. The truth is delivered here — and knowing
it marks the player.

## 1. The hybrid structure + entry

The interior is an **instance** (like the Elemental Oasis — repeatable,
party-scaled, resets each run, ephemeral rooms), reached through the
Eastern Highlands disc-door (6372).

- **The disc is the reusable admission key** (earned in the disc
  questline; NOT consumed — you can re-run). **Gold buy-in sets the tier**
  (difficulty + loot scaling, via the existing instance mechanism:
  `ScaleSpawnStatPools × goldPaid` + `GenerateAffixedItem`). Disc = *you
  belong here*; gold = *how deep the Keeper opens*.
- **The one-time revelation rides a `truth-known` character flag.** The
  records archive (7.3b) holds the truth. The FIRST time a character
  reaches and reads it, the revelation fires (the big scripted beat + sets
  the flag + the seeded consequence). On later runs the records remain as
  re-readable flavor; the one-time beat + story reward do not re-fire.
- **A run:** talk to the Threshold-Keeper → give the disc + pay → the
  Keeper opens a scaled instance → trap-crawl 7.3a → reach 7.3b + the
  records → (first time) the revelation → per-run loot/reagents drop.

## 2. The Threshold-Keeper (the gold buy-in, diegetic)

A single **scavenger who has set up at the crash-site door** — the one
person greedy or cursed enough to live off the forbidden place (the lone
exception to #21's cursed/near-zero-NPC desolation; a persistent overworld
NPC at the threshold, at/just outside 6372). The party gives the Keeper
**the disc + a payment**; the Keeper **operates the ship's threshold
systems to "open the way" to a depth proportional to the coin** — more
gold rouses more of the dormant ship (more automated defenses online =
harder; more intact tech reached = richer loot + the rare reagents).

- Diegetically the gold is the **Keeper's price** for their hard-won
  knowledge and the risk of waking the deep wards — never "paying an
  ancient ship."
- A **morally-grey, memorable character**: sends parties into the
  deadliest place in the world for coin, and not all come back. Hooks: the
  Keeper likely watched **Maren's father** go in and never return (ties to
  the disc questline / the #21 cleared section).
- Mechanically the Keeper runs the existing instance-create action
  (`create_instance` behavior action / `CreateInstancesFromZone` script)
  with `gold_paid` = the payment.

## 3. The revelation + its consequence (seeded, system deferred)

The deep interior (7.3b) holds the records — the ship's log and the
**oracle-stones** (computer cores), all from a **non-techie POV** (glowing
runes, a wall of light showing "the sky-before," stones that speak).
Piecing them together delivers the game's central truth:

> This is a crashed **colony vessel from another world**; the **Chrysalis
> is native to Gaius** and infected the colonists on arrival; the
> **bloodline's "divine" immunity was genetic chance**, not selection; and
> **three more ships wait in orbit — the "moons."**

First read → the scripted revelation beat + a permanent **`truth-known`**
flag.

**Consequence — SEED, don't build the system now.** Set the flag; give a
few existing NPCs a reactive line ("you have a heretic's eyes"); deliver a
clear ominous in-world warning that this knowledge is dangerous. **Defer** a
full faction-hostility / bounty-hunter "you are hunted" subsystem to its
own follow-up (otherwise the consequence balloons into a subsystem).

## 4. The signal array + the moons (deferred hook)

The canon signal array is present in 7.3b — "activated by the disc, it
calls the ships that have waited." It is the revelation's **final beat but
NOT yet activatable**: the player reaches it, understands *this is how you
call them*, sees the disc could do it — but it is **deferred** (damaged /
needs what the moons expansion will provide). The finale's payload is the
**truth**; the signal is the deliberate hook.

**Far-future vision (documented, NOT built here):** the signal → a
**shuttle** the player can ride to **one of the three moons**, each its own
**very large, very difficult endgame instanced zone** with **gear matching
that moon's lore/flavor**. Three future mega-zones. The signal array +
shuttle bay are seeded here as the on-ramp.

## 5. The trap-dungeon + party combat (true party endgame)

Instanced, gold-scaled, **30 rooms / 3 stages**, paced by **two boss-tier
warden fights** so a coordinated party spends 2–3 hours on a full run.

- **7.3a — The Breached Section (~9 rooms) — the on-ramp.** The torn entry
  seam (6373), grey branching corridors of the made material, sealed storage
  compartments (loot), cold blue-white emergency lighting, and the
  **navigation alcove with the orbital display** (the first "four shapes, one
  damaged" beat). **This is where the Chrysalis-suppression aura is
  introduced** — mutations/spell power visibly fail as you descend, teaching
  the twist early. One **lone warden construct** + one **hazard corridor**
  teach the two threat types. Ends at a lift/bulkhead down.
- **7.3b — The Ruined Decks (~11 rooms) — the trap heart.** The maze: warden
  packs, defusable trapped passages, peak hazard-room density, plus **2–3
  optional risk/reward side rooms** off the path. Two thematic anchor rooms
  live here: the **medical bay** (home of the **mutation-scour chamber** —
  "where the change was studied") and the **fabrication bay** (the
  legendary-reagent-rich loot room). Guarded at its far end by a **mid-boss:
  a Warden-Prime** blocking the way to command.
- **7.3c — The Command Section (~10 rooms) — the climax.** Heaviest defenses;
  the **final boss (the Core Guardian)**; then the **records archive — THE
  REVELATION** (oracle-stones, the wall of light "sky-before," the
  `truth-known` flag), the command deck, the **signal array** (the deferred
  moons hook), and the **sealed shuttle bay** stub.

Net 9 + 11 + 10 = **30 rooms**. Room IDs continue from B1's stubs (6373–6375
already built as 7.3a's opening); allocate a contiguous block for the rest.

Threat = the ship's **degraded automated defenses** — a whole facility of
them, built on #21's now-working machinery, scaled up:
- **Hazard rooms** (mutator `playerbuffids` DoT — stronger discharge buffs
  than #21's) + **defusable trapped passages** (`lock.trapbuffids`,
  skill-gated) — "tons of traps" per canon.
- **Construct "wardens"** — the ship's guardians (the orb-construct
  species, the Sentinel writ large). **NOTE (from the #21 balance critic):
  species 20 (orb) has zero base stats + no `damage_multiplier`, so
  constructs hit soft — give the construct species a real
  `damage_multiplier` (~0.80) OR use debuff-on-hit adds, so the wardens are
  a real party threat.**
- Stacking hazards + traps + construct packs so a coordinated **party**
  (2–3 geared masters + companions) is genuinely needed.

### 5b. Chrysalis suppression inside the hull (a real mechanic)
Canon: *"the Chrysalis cannot reach here; the hull resists it."* This is a
**real mechanic**: inside the hull, Chrysalis-granted **mutations and
belief/spell power are suppressed or weakened** (a zone-wide aura /
debuff). The distinctive endgame twist — the one place your "magic" fails,
forcing gear- and skill-based play. Ties to the game's **"hollow question"**
theme (inside the ship, *unchanged hands operate everything*). Difficulty
lever + thematic payload. (Implementation: a zone/room aura that disables or
scales down mutation effects + spell power — an engine mechanic to design in
the plan; may reuse a suppression buff flag.)

## 6. Loot — tech relics + the legendary-reagent source + the mutation-scour

- **Instance-scaled loot** (affix-rolled by the Keeper's gold buy-in,
  exactly like the Oasis — NOT fixed 100% BIS; heed the #21 economy
  findings). Non-techie descriptions (oracle-shards, glowing medical
  relics, fabrication-tools, warden-cores).
- **THE canonical source of the very-rare legendary-craft reagents** — the
  `40166`-tier hull materials + new deeper ones (warden-core, oracle-shard,
  etc.). #22 finally feeds the **pinnacle-craft economy** the legendary-BIS
  plan called for (`project_legendary_bis_craft_items`). Finished pinnacle
  gear stays **crafted** from these reagents; #22 drops reagents + some
  tech-relic gear at Oasis-parity power.
- **THE MUTATION-SCOUR reward (the "moon-crash remort").** A drop (a
  **potion**) and/or a **chamber** (room interaction) in the ship that
  **scours ALL the player's mutations to nothing** — possible ONLY here,
  where the Chrysalis is held off. After the player **leaves the ship, the
  Chrysalis returns stronger**: better odds for MORE and RARER mutations as
  they re-acquire. A mutation **respec/reroll** — a top-tier endgame
  aspiration (chase the perfect, rarest mutation loadout), and the thematic
  climax of the "hollow question" (become unchanged, then re-form). Ties to
  the mutation system (rarity-weighted acquisition/deepening) + the Bloom
  accel mechanic. (Implementation: an engine mechanic — clear mutations +
  set a "return-stronger" modifier on the next acquisition cycle; design in
  the plan.)

## Deferred / out of scope for #22 (documented hooks)

- **The Disc Questline** (sub-project 1 — the on-ramp; separate spec).
- **The full "hunted heretic" consequence system** (seed only here).
- **The signal-array activation + the shuttle + the three moon mega-zones**
  (the far-future expansion; seeded here).
- **Arc combat calibration** (Cascade #20 + EH #21 numbers) — separate
  owed pass; #22 tuning joins it.

## Novel mechanics this zone introduces (flag for the plan)

This zone is mechanic-heavy — the plan may sequence these as distinct
build phases, each verified before the next:
1. **Hybrid disc-gated gold-scaled instance** (Threshold-Keeper +
   instance-create + `truth-known` one-time flag). Reuses the Oasis
   instance machinery.
2. **Chrysalis suppression aura** (mutation/spell suppression inside the
   hull) — new engine mechanic.
3. **The mutation-scour / return-stronger** ("remort") — new engine
   mechanic.
4. **The revelation delivery** (records/oracle-stones + the scripted beat +
   the seeded consequence) — content + light scripting.
5. **The 30-room / 3-stage trap-dungeon** (rooms + scaled hazards/traps +
   construct warden packs + two boss-tier wardens [Warden-Prime mid,
   Core Guardian final]; construct-species `damage_multiplier` fix) —
   content.

## What #22 deliberately is NOT

- Not a pure story climax (it is a repeatable endgame instance).
- Not the moons (those are the future; only the shuttle-bay hook is here).
- Not fixed-BIS loot (instance-scaled, feeding crafted pinnacle gear).
- Not the full hunted-heretic system (seeded).
- Not the disc questline (separate sub-project, built first or alongside).
