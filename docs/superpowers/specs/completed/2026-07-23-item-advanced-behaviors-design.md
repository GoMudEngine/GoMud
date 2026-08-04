# Item Editor — Advanced (Pinnacle) Behaviors

**Date:** 2026-07-23
**Sub-project:** admin web-building 2 (item authoring), final followup
**Status:** approved (design)

## Goal

Let admins author the advanced `ItemSpec` fields that make "pinnacle" items
like the Blackrazor special — combat procs, sentient voice, resource reserves,
hunger, mutation-drip, and worn buffs — directly in the `/build` item editor.
These fields already round-trip untouched (the editor rebuilds the spec from the
loaded copy), but there is no UI to create or change them.

## The fields (all on `ItemSpec`)

| Group | Fields | Constraints (`ItemSpec.Validate`) |
|-------|--------|-----------------------------------|
| Procs | `Procs []ItemProc{Trigger, Chance, CooldownRounds, Effect, Params map[string]float64}` | trigger ∈ {on_hit, on_kill, on_block, on_grapple, on_spell_hit}; effect ∈ {lifesteal, steal_pool, aoe_stun, apply_condition}; chance 1–100 |
| Reserves | `ReserveHealthPct`, `ReserveStaminaPct`, `ReserveConvictionPct` | each 0 ≤ v < 1 |
| Sentient | `VoiceId` (from `itemvoices/*.yaml`), `TauntPull` (bool) | voice must resolve to a file |
| Hunger | `HungerRounds` (int ≥ 0), `HungerDrainPct` | drain 0 ≤ v < 1 |
| Mutation drip | `MutationTickInterval` (≥0), `MutationTickChance` (0–100), `MutationRarityFloor` (0–10) | interval>0 ⇒ chance 1–100 |
| Worn buffs | `WornBuffIds []int` | — |

`ItemSpec.Validate()` runs inside `SaveItemSpec`, so every one of these
constraints is already enforced on save. A bad entry returns an error that the
editor surfaces as a red toast — the same guard pattern as the casing and
vendor-category fixes. No new boot-brick class is introduced.

## The Advanced panel (collapsible, auto-open when populated)

A new inspector section, **"Advanced — sentient & procs"**, rendered for all
item types (these are cross-type equipment behaviors).

- Rendered as a **collapsible disclosure**: a header row with a ▸/▾ chevron;
  clicking it toggles the body.
- **Collapsed by default** so ordinary items aren't cluttered.
- **Auto-expanded** on render when the item already carries any advanced data —
  a non-empty `Procs`, a set `VoiceId`, any reserve > 0, hunger set, any
  mutation-drip field set, `TauntPull`, or non-empty `WornBuffIds`. So opening
  the Blackrazor shows its behaviors immediately; opening a plain sword shows a
  collapsed "Advanced" header the author can expand when needed.
- A `Panel.advancedOpen` boolean tracks the state so a toggle survives a form
  re-render within the same item; it is recomputed from the loaded data each
  time a different item is selected.

### Section contents

- **Procs** — a repeatable row editor (add/remove, like stat-mods):
  - Trigger `<select>` (enum), Effect `<select>` (enum), Chance (int 1–100),
    Cooldown rounds (int).
  - **Params**: a nested generic key→value editor per proc — add/remove rows of
    `name` (text) + `value` (number, float). Matches `map[string]float64`
    exactly; no hardcoded per-effect schema. Hint: "effect-specific, e.g.
    ratio 0.25".
- **Reserves** — three number fields, 0–1 hint each.
- **Sentient** — `voice_id` `<select>` (from the valid voice list + "(none)"),
  `taunt-pull` checkbox.
- **Hunger** — `hunger_rounds` (int), `hunger_drain_pct` (0–1).
- **Mutation drip** — `mutation_tick_interval` (int), `mutation_tick_chance`
  (0–100), `mutation_rarity_floor` (0–10).
- **Worn buffs** — comma-separated buff ids (like the consumable `buffIds`).

## Backend contract

- **`itemUpdateReq`** gains: `procs []procRow`, `reserveHealthPct`,
  `reserveStaminaPct`, `reserveConvictionPct`, `voiceId`, `tauntPull`,
  `hungerRounds`, `hungerDrainPct`, `mutationTickInterval`, `mutationTickChance`,
  `mutationRarityFloor`, `wornBuffIds`. `procRow` = `{trigger, effect string;
  chance, cooldownRounds int; params map[string]float64}`.
- **`specToReq`** maps these out of the loaded spec (so the form shows current
  values); **`reqToSpec`** maps them back (starting from the loaded spec, so
  any not-yet-modeled field still survives).
- **`itemDetail`** gains enum lists `ProcTriggers`, `ProcEffects`, `Voices`
  (like the existing `Types`/`Stats`). New providers:
  - `procTriggerIds()` / `procEffectIds()` — read the valid sets. The
    `validProcTriggers`/`validProcEffects` maps are unexported in
    `internal/items`; expose them via a tiny exported accessor
    (`items.ValidProcTriggers()` / `items.ValidProcEffects()`) returning sorted
    slices.
  - `itemVoiceIds()` — glob `<DataFiles>/itemvoices/*.yaml`, return the sorted
    base names.
- **No new save guard needed**: `SaveItemSpec → Validate()` already rejects
  invalid procs / out-of-range reserves / bad mutation-drip. `buildItemUpdate`
  returns that error; the form toasts it.

## Frontend

- `renderForm` gains an Advanced disclosure section built after the existing
  type-specific sections.
- A `procRow` sub-editor mirrors the stat-mods/salvage pattern: a `<div>` per
  proc holding the trigger/effect selects + chance/cooldown inputs + a nested
  params key→value list, with a "+ proc" button and per-row "✕".
- `gather()` collects `procs` (dropping rows with no trigger/effect), the
  reserve/hunger/mutation numbers, `voiceId`, `tauntPull`, and `wornBuffIds`.
- The number fields reuse the existing `numField` (integer snapping + hints);
  0–1 fields pass a fractional `step` so they're treated as decimals.

## Testing

- `internal/items`: `ValidProcTriggers()`/`ValidProcEffects()` return the
  expected sets (guards against drift if the maps change).
- `modules/gmcp`:
  - `buildItemUpdate` round-trips a proc (trigger/effect/chance/cooldown/params)
    and the reserve/voice/hunger/mutation/worn-buff fields.
  - `buildItemUpdate` rejects an invalid proc (bad trigger, chance 0) — surfaced
    as an error, nothing saved.
  - `buildItemGet` ships the trigger/effect/voice enum lists.
- Live smoke: open the Blackrazor (40183) over WS, confirm its proc + voice +
  reserve + hunger come through in `Build.Item`; save a round-trip and confirm
  persistence + clean boot.

## Out of scope

- Effect-aware param schemas (generic key→value chosen instead).
- Authoring the voice-line YAML itself (`itemvoices/*.yaml`) — the editor picks
  an existing voice id; voice content stays hand-authored for now.
- Bandolier `PreservesContents` / `AmbientPotions` — belong with the existing
  bandolier fields in the consumable section, added opportunistically if cheap.
