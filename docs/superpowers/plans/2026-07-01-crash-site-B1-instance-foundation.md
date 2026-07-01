# Crash Site #22 — Plan B1: Instance Foundation + Threshold-Keeper

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up the crash-site interior as a disc-gated, gold-scaled instance a party enters through the Threshold-Keeper at the Eastern Highlands door — a minimal but end-to-end-working entry (stub rooms), so Plan B2 can flesh out the 20-room content on a proven foundation.

**Architecture:** An `instanced: true` zone (`crash_site_interior`) with a few stub rooms; a Threshold-Keeper NPC placed in Eastern Highlands 6372 whose behavior tree (modeled on Sable, mob 315) gates on the Attuned Disc (item 40168) and runs `open_instance_portal` — charging gold, cloning a scaled instance, and wiring a temporary portal from the door into it. No engine code (all data — the instance machinery already exists).

**Tech Stack:** GoMud YAML (zone-config/rooms/mobs/behaviors); local boot; the mudagent harness for the entry test. This is **B1 of the #22 finale**; **B2** = the full 20-room content + suppression + traps + wardens + revelation + loot + scour potion, built on this.

**Reference spec:** `docs/superpowers/specs/2026-07-01-crash-site-interior-design.md` §1–2. The Attuned Disc (40168) comes from the disc questline (Quest 76, built).

**Verified mechanism (do not re-derive):**
- **Instanced zone-config** (`rooms/instance_planar_oasis/zone-config.yaml`): `instanced: true`, `entry_room: <id>`, `death_policy: rejoin|ejected`, `portal_duration: "30 real minutes"`, `allow_recall: true|false`, optional `non_cartesian: true`.
- **The broker pattern** (`behaviors/thornwall_city/315-sable.yaml`): a mob behavior tree with `player_ask` sequences; the purchase branch runs `do: open_instance_portal` with `min_gold`, `exit_expires`, `zones: {<keyword>: "<Zone Name>"}`. The action parses the ask text "`<keyword> <gold>`", validates gold ≥ min, charges it, clones + scales the instance (`ScaleSpawnStatPools × gold`), and adds a temporary exit from the NPC's room to the instance entry.
- **Gate on an item:** behavior condition `check: player_has_item` (`internal/behaviortree/conditions.go:13` → `condPlayerHasItem`); params `{itemid: N}`.
- **create_instance** authorizes the whole **party** (owner + party members) — parties enter together.
- IDs: rooms **6373+**, mobs **9553** (Threshold-Keeper), folder **`crash_site_interior`**. Attuned Disc = **40168**. The door = Eastern Highlands **6372**.

**Conventions:** zone folder = `crash_site_interior`; no colon-space in prose values; no semicolons in NPC text; `|` blocks for long text; mob name canonical `Title()`; explicit git pathspecs.

---

## Task 1: The instanced zone scaffold (stub rooms)

**Files:** Create `rooms/crash_site_interior/zone-config.yaml` + `6373.yaml`, `6374.yaml`, `6375.yaml` (stubs — B2 replaces/extends to 20).

- [ ] **Step 1: Branch + baseline.** `git checkout -b feature/crash-site-b1`; nuke instance saves; boot the current tree once, confirm clean (mobs 579, errors=0 mode=panic). Kill server.
- [ ] **Step 2: zone-config** (`rooms/crash_site_interior/zone-config.yaml`):
```yaml
name: Crash Site Interior
instanced: true
death_policy: rejoin
portal_duration: "30 real minutes"
entry_room: 6373
allow_recall: false
region: The Eastern Reach
defaultbiome: cave
```
(`allow_recall: false` — the deadly finale; `death_policy: rejoin` — a fallen party member can re-enter. Both tunable in B2.)
- [ ] **Step 3: Three stub rooms** — 6373 the entry (the breach), 6374 a corridor, 6375 a dead-end chamber. Biome `cave` (a placeholder for the grey-corridor interior; B2 refines). Each `zone: Crash Site Interior`, a short non-techy description (grey seamless corridors, cold blue-white light — threshold-only, never "ship/technology"), and exits: 6373 E→6374; 6374 W→6373, E→6375; 6375 W→6374. **6373 is the entry** — the portal lands here; it needs NO exit back to the overworld (the portal is the temporary exit, managed by the engine). Give 6373 a `mapsymbol` if desired. Keep coords simple/consistent OR set `non_cartesian: true` in zone-config if you don't want to coord these stubs (instances can be non_cartesian like the Oasis).
- [ ] **Step 4: Validate YAML + boot** — confirm `rooms.loadAllRoomZones zoneCount` rose by 1; the instance zone loads (instanced zones load their template rooms). `ValidateZoneConsistency errors=0` (or set `non_cartesian: true` to skip coord checks for the stub). **Step 5: Commit.**

## Task 2: The Threshold-Keeper mob

**Files:** Create `mobs/crash_site_interior/9553-the_threshold_keeper.yaml`

Wait — the Keeper stands at the Eastern Highlands **door (6372)**, an overworld room, NOT inside the instance. So the mob's `zone:` should be **Eastern Highlands** (she lives at the door), placed via 6372's spawninfo (Task 4). Author her in `mobs/eastern_highlands/9553-the_threshold_keeper.yaml` to match her zone.

- [ ] **Step 1: Author the Keeper mob** (`mobs/eastern_highlands/9553-the_threshold_keeper.yaml`). `non_combatant: true` (a broker, not a fight — but see the spec: morally-grey, watched). Canonical `Title()` name **"The Threshold-Keeper"** → filename `9553-the_threshold_keeper.yaml`. A distinct mutation; `speciesid: 1`; a rich description (the lone scavenger who lives off the forbidden door — gaunt, watchful, unbothered by the dead country; she has learned to work the door's deep systems; she may have watched Maren's father go in). `behavior_archetype: noncombat_passive` (she takes gold via her behavior tree, not a shop). Idle emotes (tending a fire at the threshold, watching the door, weighing coin). NO combat.
- [ ] **Step 2: Validate + boot** (mob loads, no casing/Filepath panic). **Step 3: Commit.**

## Task 3: The Keeper's behavior — disc-gated gold-scaled portal

**Files:** Create `behaviors/eastern_highlands/9553-the_threshold_keeper.yaml`; reference it from the mob (`behavior_file:` or the behavior-loader convention — confirm how mob 315 links its behavior).

Model on `behaviors/thornwall_city/315-sable.yaml`. The tree (selector of `player_ask` sequences):

- [ ] **Step 1: Confirm the mob→behavior link.** Read how mob 315 (Sable) references its behavior file (a `behavior_file:` field, or filename convention `behaviors/<zone>/<mobid>-*.yaml` auto-loaded). Wire the Keeper the same way.
- [ ] **Step 2: Author the behavior tree:**
```yaml
# The Threshold-Keeper — mob 9553 — opens the way into the buried thing.
# Modeled on Sable (315). Gates the portal on the Attuned Disc (40168);
# gold sets the tier. The disc is NOT consumed (reusable key).
tree:
  type: selector
  children:
    # ── greetings / help ──
    - type: sequence
      event: player_ask
      children:
        - type: condition
          check: keyword_match
          keywords: [hello, hi, help, who, what, quest, keeper]
        - type: action
          do: say
          text: I keep the threshold. Others come to look at the door and go home. You have the look of one who means to go in.
        - type: action
          do: say
          text: I can open the way as deep as your coin buys — but not without the key. Ask me about the way, or the key.

    # ── info: the way / the key ──
    - type: sequence
      event: player_ask
      children:
        - type: condition
          check: keyword_match
          keywords: [way, deep, portal, door, price, gold, key, disc]
        - type: action
          do: say
          text: The door answers the disc — the woken one, warm along its lines. No disc, no way in. That is not mine to give — it was made, and lost, and found again by someone. Come back with it.
        - type: action
          do: say
          text: With the key in hand, tell me how deep to open — the more you spend, the more of the deep wards I rouse, and the more the place has kept for those who reach it.
        - type: action
          do: send_user_text
          text: '<ansi fg="181">  [With the disc: ask keeper crash 300]</ansi>'

    # ── PURCHASE: gated on the Attuned Disc, then open the scaled portal ──
    # MUST be above nothing that catches "crash <gold>" first; the portal
    # action parses "crash 300" and fails-through if the text does not match.
    - type: sequence
      event: player_ask
      children:
        - type: condition
          check: player_has_item
          item_id: 40168
        - type: action
          do: open_instance_portal
          min_gold: 200
          exit_expires: "30 real minutes"
          zones:
            crash: "Crash Site Interior"

    # ── fallback: asked to open but has no disc ──
    - type: sequence
      event: player_ask
      children:
        - type: condition
          check: keyword_match
          keywords: [crash, open, enter, in]
        - type: action
          do: say
          text: Not without the key. The woken disc, or nothing. Bring it, and we will talk of how deep.
```
(`min_gold: 200` per the spec's "≥200g to make drops BIS" note; tune in B2. The disc is required but NOT consumed.)
- [ ] **Step 3: Validate + boot** — behavior loads, no parse error. **Step 4: Commit.**

## Task 4: Place the Keeper at the door + wire the door prose

**Files:** Modify `rooms/eastern_highlands/6372.yaml` (The Disc-Door)

- [ ] **Step 1: Add the Keeper's spawn** to 6372:
```yaml
spawninfo:
  - mobid: 9553
    respawnrate: "30 real minutes"
```
- [ ] **Step 2: Update 6372's prose** — the door is no longer a pure dead terminus; a lone figure keeps a fire at the threshold. Add a sentence to the description + a `keeper` noun (the scavenger who lives off the door). Keep the disc-door depression/symbol prose. Do NOT add an `east` exit (the Keeper's portal is the way in, created on purchase).
- [ ] **Step 3: Validate + boot** — 6372 loads, Keeper spawns there, `ValidateZoneConsistency errors=0`. **Step 4: Commit.**

## Task 5: End-to-end entry test + finish

- [ ] **Step 1: Boot + harness test.** With a test char holding the **Attuned Disc (40168)** (grant via the questline or admin-spawn item 40168) and enough gold, teleport to 6372:
  - `ask keeper way` → the info + disc requirement.
  - Without the disc: `ask keeper crash 300` → refused ("not without the key").
  - With the disc + gold: `ask keeper crash 300` → gold charged, a **portal exit appears** in 6372, and going through it lands in an instance of `crash_site_interior` (entry room 6373, a scaled clone). Confirm: the instance rooms exist, the disc is NOT consumed (still in inventory), and a second `ask keeper crash 300` opens another run (repeatable).
  - Confirm a **party** entering together works (owner + a member authorized) if a second test char is available; else note it (the create_instance party-authorization is engine-provided).
  - Confirm gold below `min_gold` (200) is refused.
- [ ] **Step 2: Full boot verify** — `ValidateZoneConsistency errors=0 warnings=0 mode=panic`, zone/mob counts clean, 0 panics.
- [ ] **Step 3: Merge** `feature/crash-site-b1` → master `--no-ff`.
- [ ] **Step 4: Docs + memory** — note B1 done (the instance foundation + Keeper work end-to-end); B2 = the 20-room content + all systems, built on this. Record any instance-authoring gotchas.

---

## Self-review notes
- **Spec coverage (B1 slice):** instanced disc-gated gold-scaled entry (T1 zone-config + T3 Keeper) ✓; Threshold-Keeper diegetic broker (T2/T3) ✓; disc as reusable key + gold as tier (T3 — player_has_item gate, no consume, min_gold) ✓; party entry (engine-provided, T5 confirm) ✓; placed at the door (T4) ✓.
- **Deferred to B2:** the full 20 rooms + biome/coords; the one-time revelation + truth-known flag + seeded consequence; the Chrysalis-suppression aura (data-only buff+mutator); the trap-dungeon (hazard mutators + construct wardens); the instance-scaled tech-relic + legendary-reagent loot; the mutation-scour potion delivery (wiring Plan A's ScourMutations); the signal-array + shuttle-bay stub.
- **Type/id consistency:** zone `Crash Site Interior` / folder `crash_site_interior`, entry_room 6373, stubs 6373-6375, Keeper mob 9553 (in eastern_highlands zone, placed at 6372), Attuned Disc 40168, portal zone keyword `crash`, min_gold 200 — consistent.
- **Resolved (was risk):** mob→behavior is **convention-linked** — a behavior file at `behaviors/<zone>/<mobid>-<name>.yaml` auto-loads for that mob (Sable's mob YAML has no behavior field); so `behaviors/eastern_highlands/9553-the_threshold_keeper.yaml` links automatically. The `player_has_item` param key is **`item_id`** (verified `condPlayerHasItem` → `getIntParam(params, "item_id")`), and it checks carried `Character.Items` (the Attuned Disc is carried). T3 Step 1's "confirm the link" is now just a sanity check.
```
