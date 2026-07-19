# NPC Auction Buyers — Craftsperson & Adventurer (Econ #2.2 + #2.3)

**Date:** 2026-07-14
**Status:** Approved (brainstorm), ready for implementation plan
**Goal:** Add two more living-world auction buyers on top of the #2.1 framework — a **Craftsperson**
(buys crafting materials) and an **Adventurer** (buys usable gear upgrades) — plus a small framework
touch so each archetype's win is flavored distinctly.

**Scope:** econ substages #2.2 + #2.3, batched (they are structurally identical to the collector).
Shopkeeper (#2.4, real shop gold), official (#2.5), relisting (#3), storage→auction (#4) are OUT.

---

## 1. Background

#2.1 built the `NpcBuyer` framework (`Interested`/`MaxBid`/`Wallet`), the sentinel non-user bidder,
the per-tick `nextNpcBid` decision, wallet persistence, and the generic NPC-win **sink** (item leaves
the world) with a hardcoded "…for their collection" broadcast. Adding an archetype = a new `NpcBuyer`
struct in the registry. Two new item signals (already present, no new data): `spec.IsComponent`
(materials) and `len(spec.StatMods) > 0` (gear with real stat bonuses).

---

## 2. The two archetypes (regen wallets, like the collector)

```go
type craftsperson struct { name string; wallet *NpcWallet }
func (c *craftsperson) Interested(item items.Item) bool {
	spec := item.GetSpec()
	return spec.IsComponent && spec.Value >= craftMinValue
}
func (c *craftsperson) MaxBid(item items.Item) int { return int(float64(item.GetSpec().Value) * craftPremium) }
func (c *craftsperson) Flavor() string             { return "for their workshop" }

type adventurer struct { name string; wallet *NpcWallet }
func (a *adventurer) Interested(item items.Item) bool {
	spec := item.GetSpec()
	return isEquipment(spec.Type) && len(spec.StatMods) > 0 && spec.Value >= advMinValue
}
func (a *adventurer) MaxBid(item items.Item) int { return int(float64(item.GetSpec().Value) * advPremium) }
func (a *adventurer) Flavor() string             { return "to gear up" }
```
(Each also has `Name()` and `Wallet()` like the collector.) Added to the `npcBuyers` registry with
persistent regenerating wallets.

**Provisional numbers (playtest-tuned):** `craftMinValue` 50 (materials are cheaper than gear),
`craftPremium` 1.0; `advMinValue` 300, `advPremium` 0.9 (an adventurer haggles a bit); wallets
~5000–8000 cap. Reuse the shared `npcBidChancePct` + regen cadence from #2.1.

---

## 3. Framework touch: per-archetype win flavor

Add `Flavor() string` to the `NpcBuyer` interface — the trailing phrase in the win broadcast. The
collector implements it too (`"for their collection"`). The #2.1 resolution's hardcoded broadcast
changes from a literal "for their collection" to:
```go
flavor := "for their collection" // fallback
if b := buyerByName(auctionNow.HighestBidderName); b != nil {
	flavor = b.Flavor()
}
// "%s has acquired the %s %s." (name, item, flavor)
```

---

## 4. Config knobs (plugin config, defaults inline)

`CraftspersonMinValue` (50), `CraftspersonPremium` (1.0), `AdventurerMinValue` (300),
`AdventurerPremium` (0.9). Wallets persist via the existing `WalletBalances` map (they're just more
personas in the registry — no persistence changes needed).

---

## 5. Testing

Unit (no network), mirroring the collector tests:
- **Craftsperson** `Interested`: an `IsComponent` item ≥ min → yes; a cheap component < min → no; a
  non-component (equipment/potion) → no. `MaxBid` = Value × premium.
- **Adventurer** `Interested`: equipment with `StatMods` ≥ min → yes; equipment with **no** StatMods
  (plain) → no; a component/potion → no. `MaxBid`.
- **Flavor:** each archetype returns its distinct phrase; `buyerByName` resolves them.

The bid/resolution paths are already covered by #2.1's framework tests — these archetypes just supply
different predicates into the same engine.

---

## 6. Out of scope

- #2.4 Shopkeeper (real shop gold + buy-low valuation), #2.5 Official (needs a restricted flag),
  #3 relisting, #4 storage→auction. Fine tuning of thresholds/premiums (provisional; playtest).

---

## 7. Success criteria

- A craftsperson bids on valuable materials; an adventurer bids on stat-bearing gear — each up to its
  valuation, gold-gated by its regenerating wallet, never past `MaxBid` (players outbid).
- On a win, the correct archetype flavor fires ("…for their workshop" / "…to gear up").
- Existing collector behavior + all #2.1 tests unchanged.
