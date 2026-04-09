# Thornwall Bank + Storage Fees — Design Spec

**Date**: 2026-04-07
**Status**: Approved

## Problem

Players have no secure long-term item storage with unlimited
capacity. The existing `IsStorage` rooms cap at 20 items. There is
also no gold sink tied to storage — players can hoard indefinitely
with no cost. A bank provides both a thematic home for gold
management and a natural economic drain through storage fees.

## Solution

Create a bank building in Thornwall City with unlimited item storage
and a monthly fee of 1 gold per stored item, automatically deducted
from the player's bank balance. Players who cannot pay lose their
cheapest items first after receiving a warning.

## New Room: Thornwall Bank (Room 510)

**Coordinates**: (5, -1, 0) — south of Market Square Center (465).

**Room YAML** (`_datafiles/world/dogmud/rooms/thornwall_city/510.yaml`):
- Title: "Thornwall Bank"
- `isbank: true` — enables `bank deposit/withdraw` commands
- `isstorage: true` — enables `storage add/remove` commands
- Exit north to room 465

**Room 465 update**: Add `south: { roomid: 510 }` exit.

**Description**: A solid stone building with iron-banded doors and
narrow windows. Inside, a long counter separates the public lobby
from the vault area. Heavy ledgers line the shelves behind the
counter, and the faint clink of counted coins drifts from a back
room. The air smells of ink and old paper.

## New NPC: Bank Clerk

**Mob YAML** (`_datafiles/world/dogmud/mobs/thornwall_city/120-bank_clerk.yaml`):
- `non_combatant: true`
- `hostile: false`
- `charm_immune: true`
- `maxwander: 0`
- Spawns in room 510
- Idle emotes: counts coins, adjusts ledger, examines a seal

No shop, no dialogue tree needed beyond basic keyword patterns
(hello, bank, storage, fees, gold).

## Unlimited Bank Storage

### Current Storage System

The existing `IsStorage` rooms use `user.ItemStorage` which has a
hard cap of 20 items (checked in `storage.go`). The bank needs to
bypass this cap.

### Approach

Add a `StorageCapacity` field to the room YAML (integer, 0 = use
global default of 20). The bank room sets this to a high value (e.g.
1000) which is effectively unlimited. The `storage add` command
reads the room's capacity instead of the hardcoded constant.

This avoids special-casing "bank rooms" in storage logic — any
future storage room can set its own capacity.

### Room Field

```yaml
storagecapacity: 1000  # 0 = use default (20)
```

### Storage Command Change

In `internal/usercommands/storage.go`, replace the hardcoded 20-item
check with:

```go
cap := room.StorageCapacity
if cap <= 0 {
    cap = 20 // default
}
if len(user.ItemStorage.Items) >= cap {
    user.SendText("Storage is full.")
    return
}
```

## Monthly Storage Fee

### Rate

1 gold per stored item per game month, deducted from
`Character.Bank`.

### Config

New balance knob:

| Key                   | Default | Description                         |
|-----------------------|---------|-------------------------------------|
| `StorageFeePerItem`   | 1       | Gold charged per stored item per game month |

### Timing

The fee is assessed once per game month. A new field on the
character tracks when the last fee was charged:

```go
StorageFeeLastMonth int `yaml:"storagefee_lastmonth,omitempty"`
```

This stores the game-month number when fees were last charged.
On each round tick (for online players) and on login (for offline
players returning), check if the current game month differs from
`StorageFeeLastMonth`. If so, process the fee.

### Fee Processing Logic

For each character where `currentGameMonth != StorageFeeLastMonth`:

1. `itemCount := len(user.ItemStorage.Items)`
2. If `itemCount == 0`: update `StorageFeeLastMonth`, done.
3. `fee := itemCount * StorageFeePerItem`
4. If `Character.Bank >= fee`:
   - Deduct fee from `Character.Bank`
   - Update `StorageFeeLastMonth`
   - Done (no message needed for successful payment)
5. If `Character.Bank > 0` but `< fee`:
   - Deduct whatever is available
   - Calculate shortfall items:
     `shortfall := fee - Character.Bank`
     `itemsToRemove := shortfall / StorageFeePerItem` (round up)
   - Sort stored items by `spec.Value` ascending (cheapest first)
   - Delete the cheapest `itemsToRemove` items
   - Send inbox message listing forfeited items
   - Update `StorageFeeLastMonth`
6. If `Character.Bank == 0`:
   - `itemsToRemove := itemCount` worth of items that can't be
     covered (all of them if bank is 0)
   - Same sort-and-delete as above
   - Send inbox forfeiture message
   - Update `StorageFeeLastMonth`

### Warning Message

Sent via inbox at fee time when the player CAN pay this month but
their remaining bank balance won't cover next month:

> "Thornwall Bank Notice: Your monthly storage fee of {fee}g has
> been collected. You have {remaining}g remaining in your account.
> Next month's fee will be {fee}g — please deposit additional gold
> or retrieve items to avoid forfeiture."

Only sent when `remainingBank < fee` after this month's deduction.

### Forfeiture Message

Sent via inbox when items are deleted:

> "Thornwall Bank Notice: Insufficient funds for storage fees.
> The following items were forfeited: {item list}. Your remaining
> {count} items are secure."

### Where the Logic Lives

New file: `internal/hooks/StorageFee_MonthlyCharge.go`

This registers as a listener on a round tick event. It checks all
online users each tick (cheap — just compares two integers). For
offline users, the check runs on login via the existing
`Validate()` or user-load path.

## Help Files

### bank.template

Explains the banking system: where the bank is, how to deposit
and withdraw gold, how item storage works, the monthly fee, and
what happens if you can't pay.

Key points to cover:
- Location: south of Market Square in Thornwall City
- `bank deposit <amount>` / `bank withdraw <amount>` for gold
- `storage add <item>` / `storage remove <item>` for items
- Monthly fee: 1g per stored item per game month
- Fee deducted automatically from bank balance
- Warning sent if balance is low
- Cheapest items forfeited if you can't pay
- Most valuable items kept first

### storage.template (update)

The existing storage help file should mention the bank as a
location with higher capacity and note the monthly fee that
applies there.

## Files to Create

- `_datafiles/world/dogmud/rooms/thornwall_city/510.yaml` — bank room
- `_datafiles/world/dogmud/mobs/thornwall_city/120-bank_clerk.yaml` — NPC
- `internal/hooks/StorageFee_MonthlyCharge.go` — fee logic
- `_datafiles/world/dogmud/templates/help/bank.template` — bank help file

## Files to Modify

- `_datafiles/world/dogmud/rooms/thornwall_city/465.yaml` — add south exit
- `internal/rooms/rooms.go` — add `StorageCapacity` field
- `internal/usercommands/storage.go` — use room capacity instead of hardcoded 20
- `_datafiles/world/dogmud/templates/help/storage.template` — mention bank storage + fees
- `internal/characters/character.go` — add `StorageFeeLastMonth` field
- `internal/configs/config.balance.go` — add `StorageFeePerItem`
- `_datafiles/config.yaml` — add `StorageFeePerItem: 1`

## Scope Exclusions

- Auction house (separate project)
- Interest on gold deposits
- Multiple bank branches
- Guild/shared bank accounts
- Storage fee for items in non-bank storage rooms (only bank
  storage charges fees)
