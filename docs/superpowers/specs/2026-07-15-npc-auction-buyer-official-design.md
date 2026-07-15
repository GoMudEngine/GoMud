# Econ #2.5 — Official NPC auction buyer + `restricted` item flag

**Date:** 2026-07-15
**Status:** Design (approved decisions; pending spec review)
**Roadmap:** `docs/PATH_TO_1.0.md` §1 econ arc, item **#2.5** — the last NPC-buyer archetype.
**Depends on:** #2.1 NPC-buyer foundation (`NpcBuyer` framework, sentinel bidder, regen
wallet, persistence) — merged on master. This substage only adds one archetype + one flag.

---

## 1. Problem / goal

The living-world auction has four buyer archetypes (collector, craftsperson, adventurer,
shopkeeper). The **Official** is the fifth and last: a state authority that buys **restricted**
goods off the block and takes them out of circulation (a sink). It needs a new item flag,
`restricted`, to know what to bid on.

Scope is deliberately minimal (per brainstorm): `restricted` is **only** the Official's
interest tag — no shop-sale block, no carry/equip gating, no guard reactions. Those would be a
separate contraband arc.

---

## 2. Approved design decisions

1. **`restricted` = interest tag only.** A new `ItemSpec.Restricted bool`. No other mechanics.
2. **Official pays a premium from deep pockets.** `MaxBid = Value × OfficialPremium` (default
   **1.25**) from a large wallet, so it reliably outbids others and clears restricted lots.
   (Players can still win by paying above the Official's per-item cap, as with every archetype.)
3. **Official is a sink** (like the collector): it does *not* implement `auctionWinReceiver`,
   so a won item is simply removed from circulation.
4. **Persona:** name **"The Crown Assessor"**, win flavor **"into the crown's vaults"**.
5. **Seed content:** tag the six crash-ship "warden" crafting components `restricted: true`
   (otherworldly salvage the state wants off the market) so the Official is live on day one.

---

## 3. The `restricted` flag — `internal/items/itemspec.go`

Add alongside the existing bool flags (`Cursed`, `NotSalable`, `NeverDrops`):

```go
	Restricted            bool              `yaml:"restricted,omitempty"`              // Contraband: bid on by the auction Official (econ #2.5). No other mechanics.
```

No loader/validator changes — it's a plain optional bool (defaults false).

---

## 4. The `official` archetype — `modules/auctions/npc_buyers.go`

Mirrors the collector (regen wallet, sink). ~18 lines:

```go
// ── Official archetype: buys restricted goods and sinks them (econ #2.5) ──
type official struct {
	name   string
	wallet *NpcWallet
}

func (o *official) Name() string { return o.name }
func (o *official) Interested(item items.Item) bool {
	if !officialEnabled {
		return false
	}
	return item.GetSpec().Restricted
}
func (o *official) MaxBid(item items.Item) int {
	return int(float64(item.GetSpec().Value) * officialPremium)
}
func (o *official) CanAfford(n int) bool { return o.wallet.CanAfford(n) }
func (o *official) Spend(n int)          { o.wallet.Spend(n) }
func (o *official) Refund(n int)         { o.wallet.Refund(n) }
func (o *official) Wallet() *NpcWallet   { return o.wallet }
func (o *official) Flavor() string       { return "into the crown's vaults" }
```

- The per-buyer enable check lives in `Interested()` (same pattern as the shopkeeper's
  `if !shopkeeperEnabled`). The Official is additionally under the global `npcBuyersEnabled`
  gate in `newRoundHandler` (auctions.go:644).
- **Sink:** no `Receive` method → the default sink path in `resolve... / newRoundHandler`
  removes the won item from circulation.
- **Deep pockets:** achieved via a large wallet `Cap` in the registry literal (§6), not a
  separate regen knob — the regen loop (auctions.go:645-648) applies one shared rate to every
  wallet, so the Official regenerates like the others but from a higher ceiling.

### Provisional knobs (`npc_buyers.go` var block, tuned in playtest)

```go
	officialEnabled = true // gated by AuctionOfficialEnabled config
	officialPremium = 1.25 // deep-pockets premium over item value
```

---

## 5. Config wiring — `modules/auctions/auctions.go` `load()`

Read the two plugin-config knobs (same pattern as the shopkeeper/collector knobs):

```go
	if v, ok := mod.plug.Config.Get(`AuctionOfficialEnabled`).(bool); ok {
		officialEnabled = v
	}
	if v, ok := mod.plug.Config.Get(`OfficialPremium`).(float64); ok && v > 0 {
		officialPremium = v
	}
```

Defaults (`AuctionOfficialEnabled: true`, `OfficialPremium: 1.25`) live in the plugin's config
file the same way the other auction knobs do; absent keys fall back to the package-var defaults.

---

## 6. Registry — `modules/auctions/npc_buyers.go`

Add the Official to the `npcBuyers` slice with deep-pocket wallet:

```go
var npcBuyers = []NpcBuyer{
	&collector{name: "Collector Veyd", wallet: &NpcWallet{Balance: 10000, Cap: 10000}},
	&collector{name: "Lady Ashcombe", wallet: &NpcWallet{Balance: 10000, Cap: 10000}},
	&craftsperson{name: "Master Ordwin", wallet: &NpcWallet{Balance: 6000, Cap: 6000}},
	&adventurer{name: "Sellsword Kest", wallet: &NpcWallet{Balance: 6000, Cap: 6000}},
	&shopkeeper{name: "The Merchants' Guild"},
	&official{name: "The Crown Assessor", wallet: &NpcWallet{Balance: 25000, Cap: 25000}},
}
```

Persistence is automatic: the `Wallet() != nil` loops in `save()`/`load()` snapshot and restore
the Official's balance by name (`WalletBalances["The Crown Assessor"]`), exactly like the other
wallet buyers. `buyerByName` finds it for refund/flavor.

---

## 7. Seed content — tag six crash-ship components `restricted: true`

Add `restricted: true` to each of these material YAMLs under
`_datafiles/world/dogmud/items/materials-40000/`:

| ID    | Name                        | Value |
|-------|-----------------------------|-------|
| 40169 | Warden Core                 | 1400  |
| 40171 | Hull Filament               | 950   |
| 40174 | Warden-Prime Casing         | 1600  |
| 40191 | Resonant Vox-Core           | 1300  |
| 40195 | Void-Quenched Obsidian Core  | 1400  |
| 40196 | Warden Chassis-Loom         | 1300  |

(The three related *non-component* warden items — 40225 Guardian Hull Plating, 40228
Servo-Reinforced Warden Housing, 40229 Vent-Shielded Sweeper Drum — are crafted gear/output,
not raw salvage, so they stay untagged. Only the six `is_component` salvage pieces are the
"crafting components from the crashed ship.")

These are high-value (950–1600g) contraband, so the Official (premium 1.25 → bids 1188–2000)
will reliably win them from its deep purse unless a player outbids — a live sink on day one.

Interaction note: these six are also components with `Value ≥ craftMinValue`, so the
**craftsperson** already bids on them. Both archetypes competing on the same lot is intended
(the framework picks the first interested, affordable buyer per tick; the Official's higher cap
means it typically prevails, which is the desired "contraband sinks to the state" outcome).

---

## 8. Testing — `modules/auctions/official_test.go` (create)

- **Interest:** a `restricted` item → `Interested` true; a non-restricted item (even
  high-value gear) → false; `restricted` item with `officialEnabled=false` → false (restore
  the var after).
- **MaxBid:** `= int(Value × officialPremium)` for a restricted item.
- **Escrow seam:** `CanAfford/Spend/Refund` delegate to the wallet (spend then refund restores).
- **Registry:** `buyerByName("The Crown Assessor")` returns a non-nil buyer whose `Wallet()` is
  non-nil (so persistence/regen include it).
- **Sink:** the Official does *not* satisfy `auctionWinReceiver` (type-assert fails) — asserts
  it's a sink, not a relister.

Full suite green + boot clean (the seed YAMLs load without panic; `restricted` is a known field).

---

## 9. Out of scope / deferred

- Any non-interest-tag meaning for `restricted` (shop-sale block, carry/equip gating, guard
  reactions, confiscation) — a separate contraband arc if ever wanted.
- Fine tuning of `OfficialPremium` / wallet cap — provisional, playtest.
- Broader restricted-item content beyond the six-component seed — a normal content pass.

---

## 10. Files touched

- `internal/items/itemspec.go` — `Restricted` bool field.
- `modules/auctions/npc_buyers.go` — `official` struct, package vars, registry entry.
- `modules/auctions/auctions.go` — two config reads in `load()`.
- `modules/auctions/official_test.go` (create) — coverage above.
- Six `_datafiles/world/dogmud/items/materials-40000/40{169,171,174,191,195,196}-*.yaml` —
  add `restricted: true`.
- `PATCH_NOTES.md`, `docs/PATH_TO_1.0.md` (mark #2.5 done) — at the end.
