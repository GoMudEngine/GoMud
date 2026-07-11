# Corpse-Based Loot Redesign

**Date:** 2026-07-07
**Status:** Design approved (brainstorm), pending plan.
**Scope:** Route mob loot into a lootable **corpse container** (never the room
floor), gate it by **ownership**, add **party loot modes**, and add a **shared
party gold pool**. Mob corpses only — forever.

## Motivation

Surfaced during the 5.1c arrest smoke (2026-05-29): a killed guard's gear dropped
loose to the **room floor**, and a passing caravan NPC's ground-pickup/equip-if-
better behavior grabbed the dead guard's weapons. The real takeaway: **loot
shouldn't hit the ground in the first place.** Putting mob loot inside an owned
corpse container makes the NPC-grabs-dropped-gear oddity fall out naturally — no
loose ground drops exist to grab — and opens the door to proper party loot rules.

## Scope

**In scope (v1, the full system):**
1. Mob loot drops into a **corpse container** (items + gold), not the floor.
2. **Ownership gating** — the corpse is lootable only by the killer / their party
   until a short timeout, then free-for-all, then decay.
3. **Party loot modes** — free-for-all, round-robin, leader-hold.
4. **Shared party gold pool** — corpse gold pools at the party level and
   settle-splits on membership changes.

**Out of scope — permanently:**
- **PvP / player-corpse looting.** Player death stays exactly as today: a corpse
  *snapshot* (`Death_PlayerCorpse.go`), no inventory drop. Players never loot each
  other; mobs never loot players. (PvP, if ever added, is arena-only and does not
  touch this system.)

**Deferred (not v1):**
- **DKP-auction** loot mode — revisit when guilds exist (needs a persistent,
  cross-session points economy that ephemeral pickup parties don't provide).
- **Need/greed** roll mode — a possible simpler future addition alongside DKP.

## Current engine state (verified 2026-07-07)

- **`Corpse`** (`internal/rooms/corpse.go`) — holds `UserId`, `MobId`, a
  `Character` snapshot, `RoundCreated`, `Prunable`, `WasCharmed`, and optional
  `CorpseName`/`CorpseDescription`. **No item/gold fields yet.** Added via
  `room.AddCorpse`; gated by `config.Death.CorpsesEnabled`; decays per
  `config.Death.CorpseDecayTime` (default 1 week), pruned in `UpdateCorpses`.
  Non-pickup-able (`get.go`).
- **`Container`** (`internal/rooms/container.go`) — full `Lock` / `Items` /
  `Gold` / `Hidden` / `DespawnRound` / `Recipes` with `AddItem`/`RemoveItem`;
  get/put already wired in `get.go`. This is the loot-holding primitive to reuse.
- **Mob loot today** (`internal/hooks/Death_MobLoot.go`,
  `dropMobLootAndSetCorpse`): backpack items → `room.AddItem`, equipped items
  (skip `NeverDrops`) → `room.AddItem`, gold → `room.Gold +=`. The `PermaGear`
  buff flag suppresses the whole drop block.
- **Kill credit** — `Character.PlayerDamage map[int]int` (per-user damage),
  snapshotted onto `events.MobDeath.PlayerDamage`; highest-damager "killer"
  pattern at `MobDeath_BountyClaim.go`; same-room-party-expansion pattern at
  `MobDeath_FactionRep.go`.
- **Parties** (`internal/parties/parties.go`) — in-memory only, registry-keyed by
  userId; **no `PartyId` on Character, no settings store, no gold/loot sharing**.
  Commands in `internal/usercommands/party.go`
  (create/invite/accept/leave/disband/kick/promote).

## Architecture — the corpse holds an embedded Container

Embed the existing `Container` struct into `Corpse` as a `Loot` field, reusing the
chest get/put wiring players already understand. (Rejected alternatives: raw
`Items`+`Gold` fields on `Corpse` re-implement `Container`; a separate hidden room
`Container` alongside the corpse is two objects to keep in sync through decay and
ownership.)

```go
type Corpse struct {
    // ... existing fields ...
    Loot         Container   // embedded loot container (items + gold)
    OwnerUserIds []int       // who may loot before the free-for-all timeout
    LootMode     LootMode    // snapshot of the party's mode at spawn (ffa default)
    RoundOwnedUntil uint64   // round at which ownership opens to free-for-all
    // round-robin cursor / per-item assignment as needed by Layer 3
}
```

`Corpse.HasLoot()` = `len(Loot.Items) > 0 || Loot.Gold > 0` — the gate used by
salvage/raise and the drop-on-decay path.

## Layer 1 — Corpse becomes a lootable container

`Death_MobLoot.go` stops dropping to the floor. Instead
`dropMobLootAndSetCorpse` fills the new corpse's `Loot` container:
- backpack items → `corpse.Loot.AddItem`
- equipped items (still skipping `NeverDrops`) → `corpse.Loot.AddItem`
- gold → `corpse.Loot.Gold += ...`
- **`PermaGear` suppression still applies** (no loot block at all).

Player commands:
- `look <corpse>` / `examine <corpse>` — shows the loot contents (like a chest).
- `loot <corpse>` — take everything the viewer is entitled to (per mode).
- `get <item> from <corpse>` / `get gold from <corpse>` — take a specific thing.

No more `room.AddItem` / `room.Gold +=` for mob loot. `get.go` grows a corpse-as-
container branch mirroring the existing `Container` branch, plus the ownership +
mode checks from Layers 2–3.

## Layer 2 — Ownership + timeout

At corpse spawn (mob death), stamp the owner set and the timeout:
- **Solo kill** → `OwnerUserIds = [killerUserId]`.
- **Party kill** → the party's members (the **leader is the default holder**, per
  the mode). Owner set is computed from the killer's party membership at death,
  using the existing `MobDeath.PlayerDamage` + same-room-party credit pattern.
- `RoundOwnedUntil = now + Death.CorpseLootTimeout` (config; see Config).

Rules:
- A **non-owner cannot loot** while `now < RoundOwnedUntil` ("this isn't your
  kill").
- After the timeout, the corpse is **free-for-all** — anyone in the room may loot.
- The corpse still decays on the existing `CorpseDecayTime` timer.
- **On decay, any remaining loot drops to the room floor** (last-resort fallback
  so long-abandoned loot isn't destroyed). This is well after the ~4-min
  ownership window, so it does not reintroduce the immediate-ground-drop problem.
  *(Follow-up note: review the NPC ground-pickup/equip-if-better behavior so it
  doesn't scoop decayed-corpse loot — low priority, rare edge.)*

## Layer 3 — Party loot modes

`party loot <mode>` (**leader-only**) sets the mode; stored on the in-memory
`Party` struct. Default = **free-for-all**. The corpse snapshots the mode at spawn
(`Corpse.LootMode`) so mid-loot mode changes don't retroactively scramble an open
corpse.

- **free-for-all** (default) — any owner-member `loot`s freely, first-come.
- **round-robin** — each item is assigned to the next member in a rotation; only
  the assigned member may take that item. If they **decline or can't carry**
  (full/over-capacity), the item **auto-passes to the next member**. Rotation
  order = party join order; the cursor persists across corpses in the party.
- **leader-hold** — only the leader may loot the corpse; the leader distributes
  via the normal `give` command (no new distribution verb needed).

Solo (no party) ignores modes entirely — the killer owns and loots freely.

## Layer 4 — Shared party gold pool

Corpse gold does **not** go straight to a purse when a party loots it — it flows
into a **shared party gold pool** on the `Party` struct.

- **Accrue:** looting a corpse's gold adds it to `party.GoldPool`.
- **Settle-split:** on **join / leave / disband**, the pool divides **evenly**
  among the current members (the ones who earned it), pays into their purses, and
  resets to 0 — settled *before* the membership change takes effect (so a new
  member doesn't get a share of pre-join gold, and a leaver gets their fair
  share). Also `party gold split` on demand.
- **Solo** killers have no pool — corpse gold loots straight to the purse.

Settle-split hooks live in `internal/usercommands/party.go`
(leave/disband/kick/promote/accept) + `internal/parties/parties.go`.

## Cross-cutting — salvage / raise gate

A corpse that **still holds loot is ineligible** for:
- `salvage corpse` (the salvage command), and
- raise-zombie / raise-skeleton / raise-wraith (the necromancy corpse-selection
  path).

Both refuse with "pick it clean first" until `Corpse.HasLoot()` is false. This
prevents destroying unlooted loot. Once emptied (looted, or gold+items removed),
the corpse becomes eligible again.

## Config

- **`Death.CorpseLootTimeout`** — real-time duration ownership holds before
  free-for-all. Default **~4 minutes** (user: 3–5 min). Expressed in the same
  duration form as `CorpseDecayTime`.
- Existing `Death.CorpsesEnabled` and `Death.CorpseDecayTime` unchanged.

## Build order (phased, but all v1)

1. **Layer 1** — Corpse-as-container + `Death_MobLoot` reroute + look/loot/get +
   drop-on-decay. (Foundational; solves the ground-loot bug on its own.)
2. **Cross-cutting gate** — salvage/raise refuse a loot-bearing corpse.
3. **Layer 2** — ownership stamping + timeout + free-for-all.
4. **Layer 3** — `party loot <mode>` + FFA/round-robin/leader-hold logic.
5. **Layer 4** — shared party gold pool + settle-split.

## Open / minor items for the plan

- Exact `look`/`loot`/`get from corpse` command surface + help templates.
- Round-robin cursor persistence detail (per-party, survives corpse-to-corpse).
- Whether `loot <corpse>` in round-robin auto-takes only *your* assigned items.
- NPC ground-pickup behavior review (drop-on-decay interaction) — low priority.
