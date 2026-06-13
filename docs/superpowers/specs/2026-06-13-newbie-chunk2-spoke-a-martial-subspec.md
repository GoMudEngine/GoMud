# Newbie Rework — Chunk 2 Sub-Spec: Spoke A (Martial)

> Parent spec: `docs/superpowers/specs/2026-05-27-newbie-area-rework-design.md`
> (§6.3 spoke roster, §6.4 ring structure, §7.2 reward table). Chunk 1
> (the hub) is complete and user-approved. This is the first spoke.

## 1. Scope

The combat tutorial spoke. A dry training canyon climbing **east** from
the already-built entry stub **5220 (Dry Coulee Mouth)**, up through a
contested wash to an **abandoned watch-post / ruined watchtower** held by
a bandit crew. Three concentric rings teach combat in escalating risk:

| Ring | Risk | Teaches | Cert reward |
|---|---|---|---|
| **Inner** (drill yard) | None — sanctuary | attack, consider, kick/trip, taunt, combatverbosity | combat-skill seeds + a basic weapon |
| **Middle** (the wash) | Real — can KO you → wake at the Mending Hut | flee, defense (dodge/parry/block), the death→home loop | weapon-combat rank bump + an armor piece |
| **Outer** (the watchtower) | Genuine challenge — crew + captain | special moves under pressure, taunt to hold aggro | Str **or** Dex bump + a notable weapon + weapon-combat rank |

**Combat arc (user-chosen 2026-06-13):** risk-free training **dummy** →
**weak live bandits** that can knock you out → **real boss** (the bandit
captain). The middle ring is deliberately the first place a newbie can
die — the death→wake-at-the-Mending-Hut loop is a taught lesson, and it
pays off the `sethome coulee` beat from chunk 1.

Out of scope: ranged (Spoke G), grappling depth, mounted combat. No new
combat *systems* — every command used here already ships.

## 2. ID allocations (verified `tools/id_inventory.py`, 2026-06-13)

| Type | Global next-free | Spoke A block |
|---|---|---|
| Rooms | 5227 | **5227–5249** (≈17 used; rest spare for the spoke) |
| Mobs | 9108 | **9108–9119** |
| Items | 40069 | **41000–41019** (zone block reserved in ch.1) |
| Quests | 32 | **32–34** (+ repeatables share the ids) |
| Dialogue | (by mobid) | files `9108.yaml`+ keyed by mob id |

Coordinate reserve: the spoke lays out east of the hub, **x 49–59,
y −6…+6, z 0–1** (inside the Pothole reserve x[30..59]; clear of the hub
x 42–48 and the connector corridor x[19..29]). `tools/coord_inventory.py`
gates every chunk at 0 collisions.

## 3. Room manifest (~17 rooms, 3 rings)

> 5220 (Dry Coulee Mouth, **already built**, sanctuary) is the threshold.
> Its east exit currently dead-ends; this chunk wires it to the drill yard.

### Inner ring — the drill yard (sanctuary; trivial) — 5 rooms
| Id | Title | Coord | Exits | Notes |
|---|---|---|---|---|
| 5227 | The Drill Yard | (49,0,0) | W→5220, E→5228, N→5229 | Drillmaster NPC + training dummy spawn; combat-cert quest start |
| 5228 | Weapon Rack Lean-to | (50,0,0) | W→5227 | the basic-weapon handout flavor; a second dummy |
| 5229 | Sparring Circle | (49,-1,0) | S→5227, E→5230 | repeatable "spar" beat; consider/kick/trip practice |
| 5230 | Yard Overlook | (50,-1,0) | W→5229, E→5231 | sightline up the wash to the watchtower (foreshadow boss) |
| 5231 | The Last Safe Step | (51,-1,0) | W→5230, E→5232 | sanctuary ENDS here; NPC warns "past here it's real" + teaches sethome |

### Middle ring — the contested wash (NO sanctuary; can KO) — 6 rooms
| Id | Title | Coord | Exits | Notes |
|---|---|---|---|---|
| 5232 | Lower Wash | (52,-1,0) | W→5231, E→5233, S→5234 | first live foe (weak bandit scout) |
| 5233 | Gravel Bend | (53,-1,0) | W→5232, E→5236 | scout pair; flee lesson if overmatched |
| 5234 | Squatter's Hollow | (52,0,0) | N→5232, E→5235 | middle-ring quest NPC (a held-up caravan guard) |
| 5235 | Cracked Cistern | (53,0,0) | W→5234, N→5236 | armor-drop foe; defense lesson room |
| 5236 | Upper Wash | (54,-1,0) | W→5233, S→5235, E→5237 | repeatable bandit bounty hub |
| 5237 | Tower Approach | (55,-1,0) | W→5236, E→5238 | the climb to the outer ring; last middle room |

### Outer ring — the ruined watchtower (challenge + boss) — 6 rooms
| Id | Title | Coord | Exits | Notes |
|---|---|---|---|---|
| 5238 | Broken Gate | (56,-1,0) | W→5237, E→5239, N→5240 | crew fight (2 bandits); cert quest's outer beat |
| 5239 | Collapsed Barracks | (57,-1,0) | W→5238 | loot + a lieutenant-tier foe |
| 5240 | Tower Base | (56,-2,0) | S→5238, U→5241 | stairs up; environmental note (loose stone) |
| 5241 | Tower Stair | (56,-2,1) | D→5240, U→5242 | vertical exit (z+1); tension beat |
| 5242 | The Watch Room | (56,-2,2) | D→5241, E→5243 | the **bandit captain** boss |
| 5243 | Tower Top | (57,-2,2) | W→5242 | post-boss reward vista; lateral-connector stub for adjacent spoke (future) |

Lateral outer-ring connector to the next-built adjacent spoke is left as
a stubbed exit on 5243 (parent §6.1 `══`), wired when that spoke lands.

All rooms: biome `shore`/`mountains` as fits the climb; nouns per the
discoverability rule (every painted feature look-able — the playtest
caught a missing `water` noun in ch.1, so the manifest checker's
noun-token rule is enforced here too).

## 4. Mob manifest (9108–9119)

| Id | Name (clears novel + roster — VERIFY at build) | Room | Archetype | Hostile | Role |
|---|---|---|---|---|---|
| 9108 | Drillmaster (TBD name) | 5227 | noncombat_questgiver | no | teaches all combat cmds; grants quest 32; hands basic weapon |
| 9109 | Training Dummy | 5227/5228 | combat_passive | no (attackable, statpool ~1) | risk-free first combat |
| 9110 | Bandit Scout | 5232/5233 | (weak fighter) | yes | first live foe; low statpool, can KO a fresh newbie but loseable |
| 9111 | Bandit Squatter | 5235/5236 | (weak fighter) | yes | armor-drop foe |
| 9112 | Caravan Guard (held up) | 5234 | noncombat_questgiver | no | middle-ring quest giver (clear the wash) |
| 9113 | Bandit Bruiser | 5238/5239 | (fighter, tankier) | yes | outer crew |
| 9114 | Bandit Lieutenant | 5239 | (fighter) | yes | mini-boss before the captain |
| 9115 | **Bandit Captain** | 5242 | (boss fighter; taunt-using) | yes | the spoke boss |
| 9116–9119 | spare (extra scouts / flavor) | — | — | — | reserve |

Hostile combat mobs reuse existing combat archetypes
(`behaviors/archetypes/`) — no new btrees unless a boss wants a scripted
beat. Statpools tuned so the inner ring cannot kill an attentive newbie,
the middle ring *can* (the lesson), and the captain is a real fight but
beatable with the gear+ranks the spoke has handed out.

## 5. Quest manifest

**Quest 32 — "First Blood" (inner, cert).** Drillmaster grants on
`ask <drillmaster> train`/`quest`. Beats (dialogue + command/mob_death
triggers): strike the dummy (`attack dummy`), try a special move
(`kick`/`trip` the dummy), `consider` a foe, set `combatverbosity` once.
Reward: combat-skill seeds (`skill_info`) + a basic weapon (`item_id`).
Closes by teaching `sethome coulee` and pointing east to the wash.

**Quest 33 — "Hold the Wash" (middle, rank).** Caravan Guard (9112)
grants. Defeat N bandit scouts/squatters; the quest TEXT explicitly tells
the player it's okay to `flee` and that if they fall they'll wake at the
Mending Hut. Reward: weapon-combat rank bump + an armor piece (slot
filler). + a repeatable bandit-bounty (kill-count) keyed off 9110/9111.

**Quest 34 — "Take the Tower" (outer, cert).** Granted on entering the
outer ring / from the Guard after 33. Culminates in defeating the Bandit
Captain (9115). Reward (user-approved 2026-06-13): **Str or Dex bump** +
a **notable weapon** + **a rank of BOTH weapon-combat and unarmed-combat**
(`skill_info: "weapon-combat:1,unarmed-combat:1"` — see §6, the reward
engine was extended to grant multiple skills with a floor-raise guard).
This replaces the vaguer "special-move proficiency" of §7.2 — special
moves are commands available to all, so the reward deepens skill rather
than unlocking a verb.

Inner repeatable: "spar" with the dummy (skill-use, no gold). Middle
repeatable: bandit bounty. No re-grant bugs — every `grantsQuest` node
carries the `{id}-end` token in `questExcluded` (SOP).

## 6. Engine touches — one small, done up front

Combat is fully shipped: `attack`, `consider`, `flee`, `kick`/`stomp`/
`knee`, `trip`/`tailsweep`, `taunt`, `combatverbosity` all exist; defense
best-of-all and the smooth-resource penalties are live; death→respawn via
`ResolveRespawnRoom` works. The spoke is otherwise **pure content**.

**DONE (this chunk, committed before Phase R):** the quest `skill_info`
reward now accepts a comma-separated list of `skill:level` entries
(`parseSkillGrants` in `Quest_HandleQuestUpdate.go`), reusing the existing
`currentLevel < level` floor-raise guard so it never downgrades a veteran.
Needed for Q34's two-skill reward; the single `train_skill` quest-trigger
action routes through `SetSkill` (absolute, would downgrade) so it was
NOT usable. Backward-compatible (no comma = legacy single skill); tested
(`Quest_SkillGrants_test.go`).

**One build-branch prep (pull forward from C10):** re-point the
`"default"` HomeLocations key from room 0 to **5209 (The Mending Hut)** so
a newbie who reaches Spoke A *without* having run `sethome` still wakes at
the hut when first killed — otherwise the death lesson dumps them in the
old Sanctum Basin (room 0). This is a C10 task anyway; Spoke A is the
first content that makes it matter. Inner-ring lesson ALSO teaches
`sethome coulee` as belt-and-suspenders. (**Flagged for review** —
touches the shared respawn map, build-branch only until merge.)

## 7. Lesson coverage (Tier-1 combat touch points owned by Spoke A)

| Lesson | Where |
|---|---|
| `attack <target>` + auto-attack rounds | Drillmaster + dummy (Q32) |
| `consider <target>` (read a foe before engaging) | drill yard |
| Special moves: `kick`/`trip` (+ taunt) | drill + reinforced vs. captain |
| `taunt` to hold aggro | outer ring (captain fight) |
| `flee` when overmatched | middle ring (Q33 text) |
| Defense — dodge/parry/block exist passively | middle ring lesson room |
| `combatverbosity full/medium/light` | Q32 beat |
| Death → wake at home; `sethome coulee` | inner-ring close + middle-ring payoff |
| Looting a corpse / equipping a drop | middle ring (armor drop) |

Combat that the hub deliberately did NOT teach (playtest finding) is now
taught here — a player who does Spoke A leaves combat-literate.

## 8. Acceptance criteria

1. Boots clean (mobs/quests loadedCount up by the new counts; zero panics).
2. `coord_inventory.py` 0 collisions; `cartcheck pothole_coulee` clean.
3. Manifest checker extended for Spoke A rooms+mobs (noun-token rule).
4. A fresh character can: enter from the hub → complete Q32 on the dummy →
   take a real hit in the wash → (deliberately) die and wake at the
   Mending Hut → return, finish Q33, gear up → beat the captain → claim
   the stat bump. Verified by the **naive feel-tester playtest**
   (fresh char, zero goal-coaching — see `feedback-naive-newbie-playtest`).
5. Rewards actually land (skill ranks/stat bump/items confirmed on the
   character sheet).

## 9. Build-phase task breakdown (per the user-mandated phase gates)

- **Phase R — rooms + nouns:** 17 rooms (5227–5243), wire 5220 E→5227,
  sanctuary on inner ring only, nouns, coords. Audits + walkthrough →
  **REVIEW gate.**
- **Phase M — mobs + items:** 8 mobs (9108–9115) + the reward items
  (basic weapon, armor piece, notable weapon) + spawninfo + combat
  archetype assignments + statpool tuning. Boot + presence + a combat
  spot-check → **REVIEW gate.**
- **Phase D — dialogue + quests (INLINE per user):** Drillmaster +
  Caravan Guard dialogue (combat-command teaching, voice-matched),
  quests 32–34 + repeatables, the default-home re-point, any boss btree
  beat. Naive playtest verification → **REVIEW gate (= chunk complete).**

## 10. Resolutions & remaining build-time checks

1. **Outer reward — RESOLVED (user 2026-06-13):** a rank of BOTH
   weapon-combat and unarmed-combat (+ Str/Dex bump + notable weapon).
   Engine extended to support it (§6).
2. **Default-home re-point — RESOLVED (approved):** do it now on the
   build branch — `"default"` HomeLocations → 5209 — so the death lesson
   is testable. (Still also a C10 cutover item for prod.)
3. **NPC names (build-time check):** Drillmaster + Caravan Guard names
   must clear the novel (`what_the_moons_keep.md`) and the live mob
   roster (the Maren/Pell rule). Placeholders are TBD; pick at Phase M.
4. **Boss difficulty knob (Phase M spot-check):** the captain must be
   beatable by a player carrying only what Spoke A handed out — tune
   statpool against the granted weapon/ranks during the combat spot-check.
