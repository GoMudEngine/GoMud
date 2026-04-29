# Corpse Salvage — Design (Stage 3.0e of Caravan/Economy Effort)

**Date:** 2026-04-28
**Status:** Approved (brainstorming complete, ready for implementation plan)

## Goal

Extend the existing `salvage` command to accept room-resident corpses
as targets, so players can recover cloth, leather, and a new sinew
material from the mobs they kill. Add `sinew` as a new animal-tendon
mat with two recipe homes for cross-school demand. Reclassify cloth
strip and leather strip from "deferred" (the 3.0b empty-slot state) to
"corpse-salvage-sourced." This closes the supply gap left by 3.0b's
vendor reshuffle.

## Multi-stage context

This spec is **Stage 3.0e** of the multi-stage caravan/economy effort.
Earlier stages: 1 (NPC parties), 2 (caravan), 3.0b (mat region split)
all sit unmerged on the `development` branch. Per user direction,
nothing ships to prod (`master`) until the entire economy stack lands.

Stage 3.0e is independent of 3.0a (west-of-Stillwater zone build),
3.0c (Fernway south expansion), 3.0d (NPC fold-recall), and 3.1
(forager NPCs). It can ship anytime relative to the other prereqs.

## Architecture

The existing salvage skill (`internal/crafting/salvage.go`) handles
multi-round activity, sqrt-curve skill check, and per-ingredient roll
through the `SalvageReturns` field on `ItemSpec`. We extend it to
corpses:

1. **Parse:** `salvage <noun>` first checks inventory items via the
   existing `FindItem` path; if nothing matches, checks the current
   room's `Corpses` slice for a corpse whose mob name contains the
   noun.
2. **Resolve:** corpse → mob spec → groups list. The mob's `groups:`
   field is the lookup key.
3. **Lookup:** new `LookupCorpseSalvage(groups []string) []SalvageReturn`
   helper in `internal/crafting/corpse_salvage.go` returns the salvage
   returns slice for the most-specific matching group (v1 just keys
   on `animal` and `humanoid`; future expansion can layer more keys).
4. **Roll:** existing per-ingredient skill chance applies. Each
   material rolls independently with the sqrt curve (min 0.15, max
   0.85, softCap 50, governed by Perception + salvage skill).
5. **Drop:** materials go into player inventory (or ground if backpack
   full, like other loot).
6. **Consume the corpse:** on successful completion of the salvage
   activity, the corpse is removed from the room. Matches the
   existing tagged-item salvage behavior (which fully consumes the
   target, even on a fully-failed roll). Cleaner room state, no
   "picked clean" clutter, no Salvaged-flag bookkeeping. If the
   activity is interrupted (player moves, combat starts), the
   corpse stays untouched per the existing salvage cancellation
   path.

**Salvage table** (single Go map):

| Group key | Drops (each rolled per skill curve) |
|---|---|
| `animal` | leather-strip ×2, sinew ×1 |
| `humanoid` | cloth-strip ×2, leather-strip ×1 |

If a corpse's groups don't match any key, salvage fails cleanly with
"nothing useful to recover here." This intentionally leaves
elementals, summons, and chrysalis-touched mobs as out-of-scope for
v1; option C (richer drops with feathers/chitin/bone) can layer in
later when zones building those mob types ship.

**Salvage kit required** — same as existing tagged-item salvage. The
kit is sold by Fence Dealer Siv (1g). Field-skinning at the kill site
doesn't get a free pass. This keeps corpse salvage on the same
discoverability footing as existing salvage.

## New material: sinew

`_datafiles/world/dogmud/items/materials-40000/40068-sinew.yaml`
(item ID confirmed available; reconfirm during plan task 0).

```yaml
itemid: 40068
name: sinew
namesimple: sinew
description: A length of dried tendon stripped clean from an
  animal carcass and stretched between two pegs to cure. Tough
  enough to bind a haft against generations of use; supple
  enough to draw a hunting bow without splintering. Sewers
  reach for it when a needle and thread won't hold the seam
  through a hard winter.
type: object
subtype: mundane
component_tag: sinew
weight: 0.05
value: 25
is_component: true
```

Theme: tough animal tendon for binding, bowstring, heavy-duty sewing
where thread is too weak. Value 25g sits between leather strip (1g)
and shadowcap mushroom (40g) — mid-low tier, reflecting the salvage
gating.

## Recipe demand wiring

Two existing recipes get sinew added as an ingredient, one each from
tailoring and blacksmithing for cross-school coverage. Implementation
reads the recipes folder during plan task 0 and picks the cleanest
insertion points — likely:

- **Tailoring**: a mid-tier strap/satchel/cloak recipe whose binding
  component would plausibly use animal sinew rather than thread
  (heavy-duty seam).
- **Blacksmithing**: a mid-tier haft-wrapped weapon recipe (hook-spear,
  hammer, pole-arm) whose haft binding takes sinew over leather strip
  (more durable wrap).

Two recipes is enough to give the salvage skill a real reason to
exist; we can layer more sinew recipes in later if needed.

**No new recipes invented.** Strictly ingredient additions to existing
recipes within the existing 6 craft schools.

## Corpse consumption on salvage

On successful completion of the salvage activity, the corpse is
removed from the room (via `room.RemoveCorpse` or equivalent — exact
API picked during plan task 0). No `Salvaged` flag, no remaining
"picked clean" husk in the room — the salvage cleanly takes the
corpse with it.

This mirrors how the existing tagged-item salvage works on inventory
items: the item is fully consumed by the salvage activity, even when
all rolls miss. Same intent here — the activity has cost regardless
of outcome.

If the activity is **interrupted** (player moves out of the room,
enters combat, etc.) the corpse stays untouched per the existing
salvage cancellation path. Only successful completion consumes it.

Side-effect: a player wanting to use the corpse for the manifestation
skill (`assess <corpse>` for undead animation) needs to use that path
BEFORE running salvage. Choosing salvage over manifestation is a
meaningful in-fiction tradeoff.

## Group precedence

The `LookupCorpseSalvage(groups []string)` helper accepts the full
groups slice and iterates. v1 only has two recognized keys (`animal`
and `humanoid`), so precedence doesn't matter yet. The API shape is
designed so future expansion (option C richer drops with `bird`,
`insect`, `chrysalis`, etc.) doesn't need a rewrite — just add more
keys to the table.

If a mob has both `animal` and `humanoid` (shouldn't happen, but
theoretical): the helper takes the first matching key in the table
iteration order. For deterministic behavior, the table is a slice of
`{key, returns}` pairs rather than a map, iterated in declaration
order.

## Audit matrix update

`docs/economy/mat-audit-matrix.md` (created in 3.0b) gets two updates:

- New row for `40068 sinew` — Mid-tier overlap (animals are universal,
  but the Stage 3.1 forager/3.4 caravan flow may favor specific
  regions later)
- `40002 leather strip` and `40007 cloth strip` reclassified from
  "Defer to 3.0e" to **Mid-tier overlap (corpse-salvage sourced)** —
  source pipeline now decided.

`40012 thread spool`, `40013 bone needle`, `40055 cattail down`,
`40052 drowned-hunter hide` all stay as-is per the existing audit:
- Thread spool / bone needle: Base (manufactured, not corpse-sourced)
- Cattail down: Stillwater (forager-territory water plant — flagged for
  Stage 3.1)
- Drowned-hunter hide: stays as a unique/quest drop, not in the
  group-salvage table

## Vendor inventory implications

3.0b dropped cloth/leather slots from caravan-served vendors. **3.0e
does not re-add them.** Players salvage corpses for cloth, leather,
and sinew. Stage 3.1 (forager NPCs) and Stage 3.4 (real item transfer)
will revisit whether vendor inventories should also stock these mats.

This means between 3.0e ship and 3.1 ship, the only source of cloth/
leather/sinew is corpse salvage. Players who don't fight much will
struggle to tailor — that's intentional pressure for v1; if it
proves too restrictive in playtest, vendor stocking can be added back
in a follow-up commit.

## Files affected

| Action | File | Purpose |
|---|---|---|
| CREATE | `internal/crafting/corpse_salvage.go` | Group → salvage_returns table + `LookupCorpseSalvage` helper |
| CREATE | `internal/crafting/corpse_salvage_test.go` | Unit tests for the table lookup |
| MODIFY | `internal/usercommands/salvage.go` | Parser extension: room corpses if inventory match fails; remove corpse on successful completion |
| CREATE | `_datafiles/world/dogmud/items/materials-40000/40068-sinew.yaml` | New animal-tendon mat |
| MODIFY | `_datafiles/world/dogmud/recipes/tailoring/{recipe}.yaml` | Add sinew to one existing recipe |
| MODIFY | `_datafiles/world/dogmud/recipes/blacksmithing/{recipe}.yaml` | Add sinew to one existing recipe |
| MODIFY | `docs/economy/mat-audit-matrix.md` | Add sinew row + reclassify leather/cloth strip |
| MODIFY | `docs/schemas/mob.md` | One-line note that `groups:` drives corpse salvage |
| MODIFY | `PATCH_NOTES.md` | Stage 3.0e dev-only entry |

## Verification

**Phase 1 — unit tests:**
- `go test ./internal/crafting/...` — table lookup correct for animal,
  humanoid, no-match, multi-group corpses

**Phase 2 — boot test:**
- `go build ./...` clean
- Server boots without panic
- `items` loadedCount +1 (sinew)
- Recipes loadedCount unchanged (97 — editing existing, not adding)
- Mobs loadedCount unchanged

**Phase 3 — in-game smoke test:**
1. Kill a wild dog or wolf → corpse on ground
2. `salvage corpse` (or `salvage dog`) → multi-round activity → leather strip + maybe sinew drop into inventory; corpse removed from room
3. `look` confirms the corpse is gone
4. Kill a bandit → `salvage corpse` → cloth strip + maybe leather strip drops; corpse removed
5. Kill a chrysalis-touched mob (no `animal` / `humanoid` group) → `salvage corpse` fails with "nothing useful to recover"
6. Confirm the 2 new sinew recipes craft when player has all ingredients
7. Confirm salvage skill progresses on use (existing OnSkillUse flow)

## Out of scope (explicitly)

- Per-mob `salvage_returns` overrides — group baseline is sufficient
  for v1; if a future spec needs a unique-mob drop, layer it in then
- Bird/insect/elemental drop tables (option C from brainstorming) —
  defer until specific zones build creates demand
- New cloth/leather/sinew recipes — strictly recipe edits in 3.0e, no
  new recipes invented
- Re-adding cloth/leather slots to caravan-served vendor inventories —
  Stages 3.1/3.4 territory; salvage is the v1 source
- Tanning/curing/processing tiers (raw hide → tanned → leather strip)
  — flat one-step salvage in v1
- Skinning skill as a separate skill — reuses existing salvage skill
- Field-station free-skinning vs salvage-kit-required — salvage kit
  required for corpses, matching existing tagged-item salvage
- Corpse drag/move mechanics — corpses stay where they died
- Multi-stage salvage (skin once for hide, butcher again for meat) —
  single salvage exhausts the corpse
- Forager NPCs salvaging corpses for the player economy — Stage 3.1
  decides whether foragers extend to corpse-salvage gathering
- Salvage failure refund — failed all-miss roll still consumes the
  attempt (matches existing tagged-item salvage behavior)

## Implementation order (preview for the plan stage)

Approximate ordering. Plan task 0 will refine.

1. Add sinew mat YAML (small)
2. Create `corpse_salvage.go` with the group table + lookup + tests (small)
3. Extend `salvage` command to handle corpses + remove on completion (medium — touches user-facing parsing + activity wire-up)
4. Wire 2 sinew recipe edits (small, one per school)
5. Update audit matrix + schema docs + PATCH_NOTES (small)
6. Verification + in-game smoke test (manual)

~6 tasks total. Smaller spec than 3.0b.
