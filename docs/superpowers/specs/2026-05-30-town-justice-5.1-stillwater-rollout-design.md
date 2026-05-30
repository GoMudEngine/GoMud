# Town Justice 5.1 — Stillwater Rollout + Quest-Rep Audit (Design)

**Date:** 2026-05-30
**Chunk:** Town Justice 5.1 (cross-cut) — Stillwater rollout slice
**Status:** Design — approved, plan pending
**Depends on:** 5.1a (guard enforcement), 5.1b (auto-bounty), 5.1c (arrest),
guard-combat-capability precursor — all shipped 2026-05-29.

---

## Summary

Chunk 5.1c pulled pay-fine + rep-reset forward and left "5.1d quest-based
redemption" as an optional rump. Rather than build that rump, this slice does
the higher-value work the 5.1c followups flagged: **extend the working
Thornwall justice framework to the town of Stillwater**, and **wire faction
reputation rewards onto the existing civic quests** in both towns (only quest
2 currently grants rep).

Nothing here is new justice *mechanics* — it is a second-town rollout of the
shipped framework, plus one infrastructure refactor (data-driven holding-cell
registry) chosen deliberately because the zone-expansion plan
(`docs/ZONE_EXPANSION.md`) commits to ~23 zones and many future towns that
will each want their own watch + lockup. Moving the cell registry from
hardcoded Go to faction data now — while there is exactly one entry to
migrate — removes a per-town Go-edit bottleneck for that pipeline.

This slice does **not** touch Stillwater town-flavor (schedules, dialogue,
relationships) — that remains chunk 6.1. The only NPC changes here are faction
`groups:` tags (justice infrastructure) and Constable Drunn's combat profile.

---

## Component summary

| Component | What |
|-----------|------|
| Cell registry | Move holding-cell room id from hardcoded `jailCellFor` map onto the faction definition YAML (`holding_cell_room:`); `justice` reads it from faction data. |
| Factions | New `stillwater_guards` (with cell room) + `stillwater_citizens`, mirroring the Thornwall guards/citizens pair. |
| Constable Drunn (335) | Flip to combat-capable lone enforcer: `guard_captain` hybrid archetype, statpool 240, plus faction tags. |
| Cell room 5106 | New Stillwater holding cell, `down` from the Constabulary (4110), mirroring Thornwall 5105. |
| Citizen tagging | Tag Stillwater town NPCs, two foragers, one Ironwind forager, and the Ketil caravan crew with the appropriate citizen faction(s). |
| Quest-rep audit | Add `bump_rep` (flat +15) completion actions to seven civic quests across both towns. |
| Warn-stamp pruning | Clear stale `justice_warned_*` MiscData on detention release (5.1 followup #2). |
| Docs | Package context.md updates, roadmap mini-brief, resolve followup #1. |

---

## 1. Data-driven holding-cell registry (refactor)

**Today.** `internal/justice/arrest.go:38`:

```go
var jailCellFor = map[string]int{
    "thornwall_guards": 5105,
}
```

`ExecuteArrest` (`arrest.go:243`) does `cell, ok := jailCellFor[faction]; if !ok { return false }`.

**Change.**

- Add a field to `factions.Definition` (`internal/factions/types.go`):

  ```go
  HoldingCellRoom int `yaml:"holding_cell_room"`
  ```

  Optional. `0` / omitted means "this faction has no jail" — correct for
  citizen and warren factions. Only guard factions set it.

- `ExecuteArrest` reads the cell from faction data instead of the map:

  ```go
  def := factions.GetDefinition(faction)
  if def == nil || def.HoldingCellRoom == 0 {
      return false
  }
  cell := def.HoldingCellRoom
  ```

  This preserves the existing graceful no-op: a guard whose faction has no
  cell simply cannot arrest (same as today's `!ok`).

- **Delete** the `jailCellFor` map. Move its single entry into
  `thornwall_guards.yaml` as `holding_cell_room: 5105`.

- **Faction-selection robustness fix** (`enforce.go:178-180`). Today the arrest
  fork uses `faction := guardFactions[0]` — the guard's *first* faction. With
  multi-faction guards (Drunn carries `stillwater_guards` **and**
  `stillwater_citizens`), a citizen faction sorting first would make arrest
  no-op. Change to pick the first guard-faction whose `HoldingCellRoom` is
  non-zero:

  ```go
  faction := firstFactionWithCell(guardFactions) // first with HoldingCellRoom != 0
  if faction == "" {
      // no arresting faction has a cell — fall through (no arrest)
  }
  ```

  Group-ordering then no longer matters for correctness.

- **Boot validation.** Where rooms are guaranteed loaded (justice init / the
  existing 5.1c config-validation path), validate that every faction with a
  non-zero `HoldingCellRoom` resolves to a real loaded room; `panic` with a
  clear message on a dangling reference, mirroring the faction ally/enemy
  panic style in `registry.go:81-89`. (Validation runs only for set cells, so
  citizen/warren factions are unaffected.)

**Why data-driven now.** The zone plan commits to New Plymouth (6 districts +
sewers, dock constabulary referenced), The Confluence, Greenford, Amber
Valley, and crossroads villages. Each plausibly wants a watch + lockup. A
hardcoded map forces a Go edit + recompile per town; faction-data cells let a
content author stand up a new town's justice purely in data. Migration cost is
trivial today (one entry).

---

## 2. Faction definitions

Two new files mirroring the Thornwall pair.

`_datafiles/world/dogmud/factions/stillwater_guards.yaml`:

```yaml
faction_id: stillwater_guards
display_name: "Stillwater Constabulary"
description: |
  The lone constable's office of Stillwater — a quiet lake town
  that keeps order with one tough lawman rather than a garrison.
default_rep: 0
allies: [stillwater_citizens]
enemies: []
holding_cell_room: 5106
```

`_datafiles/world/dogmud/factions/stillwater_citizens.yaml`:

```yaml
faction_id: stillwater_citizens
display_name: "Stillwater Townsfolk"
description: |
  The fishers, traders, and craftspeople of Stillwater. They keep
  the lake town alive and look out for one another.
default_rep: 0
allies: [stillwater_guards]
enemies: []
```

No cross-town enemy edges — Stillwater has no warren-equivalent antagonist
faction yet. (Exact display names / descriptions are author-polish; the
structural fields are what matter.)

---

## 3. Constable Drunn (335) → combat-capable lone enforcer

Current: `behavior_archetype: noncombat_questgiver`, statpool 80, stats
8/8/5/5, `groups: [humanoid, guard]`, home room 4110, `charm_immune: true`,
`hostile: false`.

Changes:

- `behavior_archetype: noncombat_questgiver` → `guard_captain` (the hybrid
  questgiver+combat archetype Velk 94 uses — preserves Drunn's quest-giving
  for quest 19 while letting the justice tick drive warn/arrest/attack).
- `statpool: 80` → `240`, tank-leaning stat distribution matching the
  Thornwall city/gate guards bumped by the guard-combat precursor
  (`archetype: fighting`, tank stats). Keep `charm_immune: true`.
- Keep `hostile: false` — Drunn reacts to crimes via `RunGuardEnforcement`,
  not blanket aggression.
- `groups:` gains `stillwater_guards` (listed first, so it is the primary /
  cell-bearing faction) and `stillwater_citizens`. Final:
  `[humanoid, guard, stillwater_guards, stillwater_citizens]`.

The `guard` group is already present, so `isGuardMob` already returns true and
`RunGuardEnforcement` already ticks for Drunn — it currently no-ops only
because `FactionsForMob` returns empty (no faction group). Adding the faction
flips him on; the archetype + statpool bump lets him actually win.

---

## 4. Stillwater holding cell — new room 5106

New `_datafiles/world/dogmud/rooms/stillwater/5106.yaml`, mirroring Thornwall's
cell (5105):

- `zone: Stillwater`, `title: Holding Cell`.
- Reached **`down` from 4110** (the Constabulary already describes a barred
  cell behind the office as flavor — this realizes it as a room).
- `coord: { x: -17, y: 5, z: -1 }` — directly beneath 4110 (which is at
  x −17, y 5, z 0). Zero x/y overlap; z-axis only, exactly like Thornwall's
  cell sitting `down` from the barracks. Verify the z=-1 coord is unoccupied
  during build (almost certainly free).
- `allow_recall: false` (blocks recall-class spells out of the cell).
- Exits: `up` → 4110 on the cell; add a reciprocal `down` → 5106 exit on 4110.
  Lockdown is enforced by the Jailed buff's `no-go` flag (blocks walk/flee) +
  `allow_recall: false`, identical to Thornwall — the `up` exit existing is
  fine and matches the 5105 pattern.
- 2+ examinable nouns (bars, pallet, bucket) per the `docs/ZONE_EXPANSION.md`
  quality bar; 80-col wrapping.

Room id 5106 is the next globally-free id (max was 5105) and clusters with the
existing justice/special-room block (Thornwall cell 5105, Stillwater bank
5100). Run `python tools/id_inventory.py` before authoring to reconfirm.

---

## 5. Citizen faction tagging

Citizens are crime **witnesses**, not enforcers: `isGuardMob`
(`NewRound_MobRoundTick.go:468`) gates `RunGuardEnforcement` to mobs in the
literal `guard` group, so tagging a civilian with a citizen faction never
makes them attack a wanted player. The citizen tag's effect is that the mob
satisfies `crimes.WitnessesInRoom` for that faction — a crime committed in
their presence is identified and recorded against the faction, and that
faction's *guards* react later.

Governing principle: **an NPC is a citizen of the community it belongs to,
tagged only when that town's faction exists in this chunk.**

Tagging list:

| Mob(s) | Tag(s) added |
|--------|--------------|
| All ~25 Stillwater town NPCs (330-356) | `stillwater_citizens` |
| Tova (371, Stillwater Marsh forager) | `stillwater_citizens` |
| Kessa (373, Fernway South forager) | `stillwater_citizens` |
| Halix (372, Ironwind Steppe forager) | `thornwall_citizens` |
| Ketil (357), Marta (358), Lars (359) — caravan humans | **both** `thornwall_citizens` + `stillwater_citizens` |
| Hob (375), Bran (376) — caravan draft animals | *none* (animals, not citizens) |

Notes:

- **Drunn (335)** is in the 330-356 range but is a guard; he gets the guard +
  citizen tags per §3, not just citizen.
- **Caravan crew dual citizenship** reflects that the Ketil family is the
  literal trade link between the two towns; a crime witnessed by the caravan
  reports to *both* towns. (The crew was added in the 3.8 caravan work, after
  chunk 1.3 tagged Thornwall citizens, so they were previously untagged — this
  also closes that gap.)
- **Halix → Thornwall, Kessa/Tova → Stillwater** per the world-author's
  delivery/association mapping (foragers report wilderness crimes to the town
  they supply).
- Tagging is pure `groups:` line edits — no behavior or stat changes to these
  NPCs.

---

## 6. Quest-rep audit

Only quest 2 (`bump_rep: {faction: warren, delta: 50}`) currently rewards
reputation. Add a `bump_rep` completion action — flat **+15** — to each civic
quest below, mapped by the giver's town and role. `bump_rep` is single-faction
with no ally cascade, so one line per faction.

| Quest | Name | Giver (town/role) | Faction | Delta |
|-------|------|-------------------|---------|-------|
| 8 | The City Watch's Missing Person | Capt. Velk (Thornwall, guard) | `thornwall_guards` | +15 |
| 7 | The Fallow Field | Farmer Dorn (Thornwall Outskirts) | `thornwall_citizens` | +15 |
| 9 | The Temple's Tithe Audit | Priest Olen (Thornwall) | `thornwall_citizens` | +15 |
| 10 | The Drowning Post's Debt | Tavern Keeper Marek (Thornwall) | `thornwall_citizens` | +15 |
| 14 | The Undertow | Tavern Keeper Marek (Thornwall) | `thornwall_citizens` | +15 |
| 19 | The Lake-Caves Bounty | Constable Drunn (Stillwater, guard) | `stillwater_guards` | +15 |
| 20 | Ulla's Silence | Ulla (Stillwater) | `stillwater_citizens` | +15 |

- The `bump_rep` action goes on each quest's terminal completion node, beside
  the existing `grant: <quest>-end` (the quest-2 pattern at
  `2-the_warren_compact.yaml:37-39`).
- Quest 2 is left untouched.
- Non-town quests (Ironwind 11-13, Dustwalk 4, Sanctum 1/3, Watchers 5/6,
  Ashwick 16, North Road 15/17/18) are **out of scope** — their factions do
  not exist yet (chunk 6.5a).
- +15 is tunable; a handful of civic quests earns "liked" standing without any
  single quest dominating the −100..+100 band the way quest 2's +50 does.

---

## 7. Warn-stamp pruning (5.1 followup #2)

**Context — which stamp, and why release-time is the wrong seam.** Post-5.1c,
`Verdict` returns only Arrest / Warn / None (never Attack). The enforcement
switch (`enforce.go:146-184`) writes two distinct per-(guard, player) MiscData
stamps:

- `justice_arrest_pending_<userId>` — Arrest tier (open bounty, Hostile rep, or
  unresolved crime). **Already self-cleaned** — cleared on haul
  (`enforce.go:181`). No work needed.
- `justice_warned_<userId>` — Warn tier, which fires **only for TierCold rep**
  (a disliked-but-not-criminal player). After the grace window it escalates to
  attack (`warnOutcomeAttack`), but the stamp itself is **never cleared**.

The accumulation in followup #2 is specifically the `justice_warned_*` stamp.
A cold-rep player who keeps getting warned is **never jailed** (cold rep alone
yields Warn, not Arrest), so they never pass through `ResolveDetention`.
Hanging the prune off the release path would therefore miss the exact players
who accumulate the stamp. Once a warned player's rep recovers above Cold, the
Warn branch stops running and never revisits the stale key on its own.

**Fix — guard-side sweep.** Prune the guard's own `justice_warned_*` entries
that are older than a staleness threshold (reuse the crime-lookback window,
`JusticeCrimeLookbackRounds`, or a dedicated knob) during guard save / a
bounded per-guard sweep. This is keyed on the guard (where the stamps live),
independent of whether any particular warned player is ever arrested. Leave the
already-self-cleaning `justice_arrest_pending_*` stamps untouched. The plan
will pick the exact sweep seam (guard save hook vs. a cheap in-tick sweep
bounded to the guard's existing stamps).

---

## 8. Documentation

- `internal/justice/context.md` — note the holding-cell room moved from the
  hardcoded `jailCellFor` map to faction data (`HoldingCellRoom`), and the
  cell-selection-by-first-faction-with-cell rule.
- `internal/factions/context.md` — document the new `holding_cell_room:` field
  and its boot validation.
- `MOB_ALIVENESS_ROADMAP.md` — update the 5.1 mini-brief: Stillwater rollout
  slice shipped; note guards/citizens factions now exist for two towns.
- Memory `project_town_justice_5_1_followups.md` — mark followup #1 (Drunn
  faction) and followup #2 (warn-stamp pruning) resolved.

---

## Testing

Unit:

- `ExecuteArrest` reads cell from faction data: returns the configured room
  for a guard faction with a cell; returns `false` (no-op) for a faction with
  `HoldingCellRoom == 0` or an unknown faction.
- `firstFactionWithCell` picks the first guard-faction with a non-zero cell,
  skipping a leading citizen faction.
- Faction loader: `HoldingCellRoom` round-trips from YAML; boot validation
  panics on a dangling cell-room reference and passes for `0`/unset.
- Quest-rep: completion nodes carry the expected `bump_rep` faction+delta
  (extend existing quest-load assertions where present).

Boot smoke (instance-wipe per SOP):

- Server boots clean past data-file load: two new factions load (ally
  validation passes), new cell room 5106 loads, Drunn's `guard_captain`
  archetype + faction tags load, all retagged mobs load without panic, all
  seven edited quests load.
- `factions.LoadAllDefinitions` logs both new factions; no unknown-ally panic.

In-game smoke (deferred to user, per chunk precedent):

- Commit a crime in Stillwater within Drunn's sight → warn → arrest → haul to
  cell 5106 → `pay fine` / serve → release clears crime + bounty + resets rep.
- Confirm a crime witnessed only by a tagged Stillwater civilian (no guard
  present) records against `stillwater_citizens` and Drunn reacts on his next
  pass.
- Complete quests 8 and 20 → confirm `faction show` reflects +15 with the
  mapped faction.
- Verify a guard's stale `justice_warned_*` stamps (cold-rep warnings whose
  owner's rep has since recovered) are swept after the staleness threshold,
  while fresh stamps and any `justice_arrest_pending_*` stamps are untouched.

---

## Out of scope

- **6.1 Stillwater town-flavor** — schedules, dialogue topics, NPC↔NPC
  relationships for Stillwater NPCs. This slice only adds faction `groups:`.
- **New quests.**
- **Combat-subdue arrests** (sap/net) — still deferred per 5.1c.
- **Deputies / additional Stillwater guard mobs** — Stillwater enforces with
  one constable by design.
- **Other-zone factions** (Ironwind, Dustwalk, Ashwick, etc.) — chunk 6.5a.
- **Gossip propagation of crimes across towns** — if "the caravan carried word
  to the other town" is later wanted as distinct from dual citizenship, that
  is the 1.7 world-facts substrate, not faction membership.
