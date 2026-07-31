# Auctions Module Context

## Purpose

`modules/auctions` is the global auction house. Players list items; other
players **and a panel of NPC buyers** bid against each other in real gold. It
also receives lots seized from unpaid storage, which is how abandoned goods
re-enter the economy.

The NPC bidders are what make the house feel alive on a small player
population — a listing with no human interest still finds a price.

## Files

- **auctions.go** — the module, `AuctionManager`, `AuctionItem`, the `auction`
  command, persistence, and the round handler.
- **npc_buyers.go** — the `NpcBuyer` interface, `NpcWallet`, and each buyer
  archetype.

## Types

```go
type AuctionItem struct     { /* item, seller, bids, buyout, reserve, end round */ }
type PastAuctionItem struct { /* completed record */ }
type SeizedLot struct       { /* storage-seizure origin */ }
type AuctionManager struct  { /* active + history */ }

type AuctionUpdate struct   { /* event payload; Type() == "AuctionUpdate" */ }
```

## NPC buyers

```go
type NpcBuyer interface {
    Name() string
    Interested(item items.Item) bool
    MaxBid(item items.Item) int
    CanAfford(n int) bool
    Spend(n int)
    Refund(n int)
    Wallet() *NpcWallet
    Flavor() string
}
```

Archetypes — collector, craftsperson, adventurer, merchants' guild, crown
assessor — differ in what they want (`Interested`) and what they will pay
(`MaxBid`). `Flavor()` supplies the reason shown to players ("for their
collection"), which is what stops NPC bids reading as a faceless bot.

`NpcWallet` is a real balance: `CanAfford`/`Spend`/`Refund`/`Regen`. A buyer
that has been spending cannot keep bidding, and outbid gold is refunded.

## Gotchas

- **Wallets regenerate; they are not infinite.** A test that lists many lots
  quickly will see NPC interest dry up. That is the design, not a bug.
- **`Refund` must be called on every outbid path.** A missed refund silently
  drains an NPC's wallet over days of uptime.
- **`tooTrivialToAuction` filters junk** before it ever reaches the house. If a
  listing is refused, check there first.
- **`restoreNpcBinding` re-attaches buyers after load** — a persisted auction
  holds a buyer id, not a live object. Anything reconstructing auctions must
  call it or NPC bids stop mid-auction.
- **`reserveFrom(buyout, pct)` derives the reserve from the buyout.** Changing
  the percentage retroactively changes nothing already listed.
- **Auction state is persisted by the module itself** (`load`/`save`), not by
  the engine's world save.

## Dependencies

`plugins`, `events`, `items`, `users`, `rooms`, `mudlog`, `configs`.

## Consumers

Registered as a plugin; drives the `auction` command, the storage-seizure
handler, and the Discord auction mirror.
