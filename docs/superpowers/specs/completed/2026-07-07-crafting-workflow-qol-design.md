# Crafting Workflow QOL — smarter craft menu + storage integration

**Date:** 2026-07-07
**Status:** Design approved (brainstorm), pending plan.
**Scope:** Make crafting less tedious: (1) a smarter `craft` menu that surfaces
what you can actually make, (2) `craft` auto-pulls missing components from the
room's storage when you're at the station, and (3) a short `stow` command to
deposit all crafting components at once. Ergonomics on top of existing plumbing —
no new data, no save migration.

## Motivation

Crafting today works but is tedious: `craft`/`craft list` dumps every known
recipe (with red `[X]` + reason for the ones you can't make), so "what can I make
right now?" is buried. And moving components between the component bag and room
storage is per-item by name; `storage add all` over-dumps (gear + consumables
too). Since you can only craft at a station, and storage often sits at the same
bank/workshop room, crafting from stored mats should just work.

## Current state (verified 2026-07-07)

- **`craft` menu** (`internal/usercommands/craft.go`, `craftList` at :221):
  filters `crafting.GetAll()` to KNOWN recipes (`Character.HasRecipe`), groups by
  Skill, annotates each with `recipeStatus` (skill min / station match /
  ingredients via `crafting.HasIngredients(Items, ComponentItems, recipe)`) plus
  ingredient list, station, time. Shows ALL known recipes regardless of
  craftability. Bare `craft` and `craft list` both call `craftList`.
- **Craft resolve path** (`craft.go` `Craft`): a recipe name routes to the craft
  resolver; result codes include `WrongStation`, `MissingIngredients`
  (`crafting.HasIngredients` over `Items` + `ComponentItems`), `SkillTooLow`, etc.
  Ingredients are consumed from `Character.Items` + `Character.ComponentItems`.
- **Storage** (`internal/usercommands/storage.go`, gated on `room.IsStorage`):
  `storage` (list), `storage add all` (deposits backpack + `ComponentItems`),
  `storage add [N] <item>`, `storage add all <item>`, `storage remove [N|all]
  [item]`. Backing store `UserRecord.ItemStorage` (`internal/users/storage.go`:
  stack-aware `Storage`/`StorageSlot`). Per-room cap `room.StorageCapacity`
  (default 20). Retrieval routes components back to the component bag via
  `Character.StoreItem`.
- **Bank** (`internal/usercommands/bank.go`) is **gold-only** (`deposit`/
  `withdraw`); item storage is the separate `storage`/`IsStorage` system. A room
  can be both. **`deposit` is taken by the gold bank** — hence a distinct verb for
  components.
- Command names `stow` / `stock` / `mats` are unused → `stow` is free.

## Design

### 1. Smarter craft menu

- **Bare `craft`** (no args) → a **"what can I make right now"** view: only
  recipes where the player knows it, meets the skill, **the current room
  satisfies the recipe's station**, and has the ingredients in **pack +
  component bag + this room's storage** (storage counted only when
  `room.IsStorage`; see §2). Empty state: a friendly "nothing you can craft here
  right now — try `craft list`."
- **Station-aware**: the current room's station gates which recipes qualify for
  the bare view. At an alchemy bench, only alchemy-bench recipes appear; a recipe
  with `station: ""` (no station needed) always qualifies.
- **`craft list`** (and `craft all`) → the **full known-recipe view**, keeping
  today's per-recipe annotations, but **sectioned** by actionability:
  1. **Ready to craft** (have everything, here) — green
  2. **Missing ingredients** (skill + station OK, lack mats) — with the missing tags
  3. **Locked** (wrong/absent station, or skill too low) — with the reason
  Grouped-by-skill can stay *within* each section, or sections can be flat —
  decide in the plan; the requirement is actionable-first.

### 2. Craft auto-pulls components from storage (at the station)

When `craft <recipe>` runs in a room that is **both a storage room
(`room.IsStorage`) and satisfies the recipe's station**, any ingredients missing
from pack + component bag are **auto-pulled from that room's storage** (exact
quantities the recipe needs, nothing more), then consumed by the craft.

- Transparent: emit e.g. `You draw 3 iron ore from storage.` before the craft
  message, for each pulled component.
- If, after checking storage, ingredients are still missing → the normal
  `You are missing: <tags>.` error (and nothing is pulled/consumed — all-or-
  nothing so a failed craft doesn't strand mats in the pack).
- Only the player's own per-character storage; only the recipe's ingredients.
- Salvage is unaffected (it needs no station and produces components; not a
  consumer). This applies to the `craft` consume path only.

### 3. `stow` — deposit all crafting components

A new short command **`stow`** (gated on `room.IsStorage`): deposits **only
crafting components** (from the backpack + the component bag) into the room's
storage, leaving gear, consumables, and quest items in the pack. The "dump my
mats after a forage/farm run" button, versus `storage add all` which dumps
everything.

- "Component" = an item with the component classification the component bag /
  `storage add all` already uses (reuse that predicate; see `ComponentItems`
  routing + `is_component`).
- Respects `room.StorageCapacity`; reports what was stowed / what didn't fit.
- **Discoverability**: a `stow` helpfile; mention `stow` in the `storage` command
  output/help and in the bank/storage room's command hints so players find it.

## Where it lives

- `internal/usercommands/craft.go` — menu sectioning + the bare-craft craftable-
  now filter (station + storage aware); the auto-pull in the resolve path
  (before `MissingIngredients`).
- `internal/usercommands/storage.go` (or a new `stow.go`) — the `stow` command +
  the component predicate; register `stow` in `usercommands.go`.
- Reuse `crafting.HasIngredients`, `UserRecord.ItemStorage`, the component-bag
  routing. No new data files, no save migration.
- Help templates for `stow` (+ update `storage`/bank help text).

## Out of scope
- Shared/account vaults (storage stays per-character).
- Changing gold banking, or salvage (station-free, produces components).
- Auto-pull for anything other than the `craft` consume path.
- New recipe data or recipe-discovery changes.

## Open items for the plan
- Exact section layout of `craft list` (flat sections vs skill-groups within
  sections) + colors.
- The precise "is this a component" predicate to reuse for `stow` and the storage-
  aware craftability check.
- Whether the bare-craft storage check should also count the component bag of
  *other* stored stacks vs only what the recipe needs (perf: check per-recipe on
  demand, not a full storage scan per render).
- Final `stow` wording + help-text placement (bank vs storage room hint).
