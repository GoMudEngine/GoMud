# Official NPC Auction Buyer + `restricted` flag (econ #2.5) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the fifth/last auction NPC buyer — "The Crown Assessor" — which bids a premium on `restricted` items and sinks them, plus the new `ItemSpec.Restricted` flag and a six-item content seed.

**Architecture:** A new `official` `NpcBuyer` archetype mirroring the collector (regen wallet, sink), gated by a per-buyer `officialEnabled` flag inside `Interested()` and the existing global `npcBuyersEnabled`. Interest keys on a new `ItemSpec.Restricted bool`. Registered in the static `npcBuyers` slice; persistence/regen are automatic via the existing wallet loops.

**Tech Stack:** Go, GoMud auctions plugin, testify, YAML data files.

**Spec:** `docs/superpowers/specs/2026-07-15-npc-auction-buyer-official-design.md`

---

## File Structure

- **Modify** `internal/items/itemspec.go` — add `Restricted bool` field.
- **Modify** `modules/auctions/npc_buyers.go` — `official` struct, package vars, registry entry.
- **Modify** `modules/auctions/auctions.go` — two config reads in `load()`.
- **Create** `modules/auctions/official_test.go` — archetype unit tests.
- **Modify** six `_datafiles/world/dogmud/items/materials-40000/40{169,171,174,191,195,196}-*.yaml` — add `restricted: true`.
- **Modify** `PATCH_NOTES.md`, `docs/PATH_TO_1.0.md` — at the end.

---

## Task 1: `ItemSpec.Restricted` flag

**Files:**
- Modify: `internal/items/itemspec.go` (bool-flag block, near `NeverDrops` ~line 328)
- Test: reuse `modules/auctions/official_test.go` in Task 2 (the flag is exercised there); no standalone test — it's a plain optional bool with no logic.

- [ ] **Step 1: Add the field**

In `internal/items/itemspec.go`, immediately after the `NeverDrops` line:

```go
	NeverDrops            bool              `yaml:"never_drops,omitempty"`             // Equipped-only: this item is skipped entirely by mob death-loot drops (boss-only gear that must never reach players). Does not affect carried Character.Items — use a separate mechanism (loot_pool / character.items) for guaranteed loot on the same mob.
	Restricted            bool              `yaml:"restricted,omitempty"`              // Contraband: bid on by the auction Official (The Crown Assessor, econ #2.5). Interest tag only — no other mechanics.
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/items/`
Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add internal/items/itemspec.go
git commit -m "feat(items): Restricted flag for auction Official interest (econ #2.5)"
```

---

## Task 2: `official` archetype + config wiring + registry

**Files:**
- Modify: `modules/auctions/npc_buyers.go` (var block ~line 41; new struct after the adventurer ~line 129; registry ~line 260)
- Modify: `modules/auctions/auctions.go` (`load()`, after the shopkeeper knob ~line 178)
- Test: `modules/auctions/official_test.go` (create)

- [ ] **Step 1: Write the failing tests**

Create `modules/auctions/official_test.go`:

```go
package auctions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

func TestOfficial_InterestAndMaxBid(t *testing.T) {
	officialEnabled = true
	officialPremium = 1.25
	o := &official{name: "The Crown Assessor", wallet: &NpcWallet{Balance: 25000, Cap: 25000}}

	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		401: {ItemId: 401, Name: "warden core", Type: items.Object, IsComponent: true, Value: 1400, Restricted: true},
		402: {ItemId: 402, Name: "iron ore", Type: items.Object, IsComponent: true, Value: 1400}, // not restricted
		403: {ItemId: 403, Name: "prize blade", Type: items.Weapon, Value: 5000},                  // valuable but not restricted
	})()

	if !o.Interested(items.New(401)) {
		t.Error("official should want a restricted item")
	}
	if o.Interested(items.New(402)) {
		t.Error("official should NOT want a non-restricted component")
	}
	if o.Interested(items.New(403)) {
		t.Error("official should NOT want valuable non-restricted gear")
	}
	if got := o.MaxBid(items.New(401)); got != 1750 {
		t.Errorf("MaxBid=%d want 1750 (Value 1400 * 1.25)", got)
	}
}

func TestOfficial_DisabledDeclinesEverything(t *testing.T) {
	officialEnabled = false
	defer func() { officialEnabled = true }()
	o := &official{name: "The Crown Assessor", wallet: &NpcWallet{Balance: 25000, Cap: 25000}}
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		401: {ItemId: 401, Name: "warden core", Type: items.Object, IsComponent: true, Value: 1400, Restricted: true},
	})()
	if o.Interested(items.New(401)) {
		t.Error("disabled official must not be interested in anything")
	}
}

func TestOfficial_EscrowSeam(t *testing.T) {
	o := &official{name: "The Crown Assessor", wallet: &NpcWallet{Balance: 1000, Cap: 1000}}
	if !o.CanAfford(400) {
		t.Fatal("should afford 400 of 1000")
	}
	o.Spend(400)
	if o.CanAfford(700) {
		t.Error("should not afford 700 after spending 400 (600 left)")
	}
	o.Refund(400)
	if !o.CanAfford(1000) {
		t.Error("refund should restore to 1000")
	}
}

func TestOfficial_RegisteredAndIsSink(t *testing.T) {
	b := buyerByName("The Crown Assessor")
	if b == nil {
		t.Fatal("The Crown Assessor must be in the npcBuyers registry")
	}
	if b.Wallet() == nil {
		t.Error("official must expose a wallet so persistence/regen include it")
	}
	// The official is a SINK: it must NOT implement auctionWinReceiver (only the
	// shopkeeper relists). A sink lets the won item leave circulation.
	if _, ok := b.(auctionWinReceiver); ok {
		t.Error("official must be a sink, not an auctionWinReceiver")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./modules/auctions/ -run TestOfficial -v`
Expected: FAIL — `official` / `officialEnabled` / `officialPremium` undefined (compile error), and `buyerByName("The Crown Assessor")` returns nil.

- [ ] **Step 3: Add the package vars**

In `modules/auctions/npc_buyers.go`, in the provisional-knobs `var (...)` block (after the `shopkeeperEnabled` line ~53):

```go
	shopkeeperEnabled = true // gated by AuctionShopkeeperEnabled config

	officialEnabled = true // gated by AuctionOfficialEnabled config
	officialPremium = 1.25 // deep-pockets premium over item value for restricted goods
```

- [ ] **Step 4: Add the `official` archetype**

In `modules/auctions/npc_buyers.go`, after the adventurer archetype (after its `Flavor()` ~line 129):

```go
// ── Official archetype: buys restricted goods and sinks them (econ #2.5) ──
// "The Crown Assessor" — a state authority that buys contraband off the block
// at a premium from a deep purse and takes it out of circulation.
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

Note: no `Receive` method — the Official is a sink (a won lot leaves circulation via the default NPC-win path).

- [ ] **Step 5: Add the registry entry**

In `modules/auctions/npc_buyers.go`, add the Official to the `npcBuyers` slice (after the shopkeeper):

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

- [ ] **Step 6: Wire the config reads**

In `modules/auctions/auctions.go`, in `load()`, after the `AuctionShopkeeperEnabled` block (~line 178):

```go
	if v, ok := mod.plug.Config.Get(`AuctionShopkeeperEnabled`).(bool); ok {
		shopkeeperEnabled = v
	}
	if v, ok := mod.plug.Config.Get(`AuctionOfficialEnabled`).(bool); ok {
		officialEnabled = v
	}
	if v, ok := mod.plug.Config.Get(`OfficialPremium`).(float64); ok && v > 0 {
		officialPremium = v
	}
```

- [ ] **Step 7: Run tests to verify they pass**

Run: `go test ./modules/auctions/ -run TestOfficial -v`
Expected: PASS (all four).

- [ ] **Step 8: Run the whole auctions package (catch any registry-length assumptions)**

Run: `go test ./modules/auctions/`
Expected: PASS (ok) — existing buyer/auction tests still green with the added registry entry.

- [ ] **Step 9: Commit**

```bash
git add modules/auctions/npc_buyers.go modules/auctions/auctions.go modules/auctions/official_test.go
git commit -m "feat(auctions): Official NPC buyer 'The Crown Assessor' for restricted goods (econ #2.5)"
```

---

## Task 3: Seed the six crash-ship components

**Files:**
- Modify: `_datafiles/world/dogmud/items/materials-40000/40169-warden_core.yaml`
- Modify: `_datafiles/world/dogmud/items/materials-40000/40171-hull_filament.yaml`
- Modify: `_datafiles/world/dogmud/items/materials-40000/40174-warden_prime_casing.yaml`
- Modify: `_datafiles/world/dogmud/items/materials-40000/40191-resonant_vox_core.yaml`
- Modify: `_datafiles/world/dogmud/items/materials-40000/40195-void_quenched_obsidian_core.yaml`
- Modify: `_datafiles/world/dogmud/items/materials-40000/40196-warden_chassis_loom.yaml`

- [ ] **Step 1: Add `restricted: true` to each file**

For each of the six YAMLs, add a top-level `restricted: true` line. Place it next to the existing `is_component: true` line (top-level key, no indentation). Example for `40196-warden_chassis_loom.yaml` — the file has:

```yaml
is_component: true
```

Add immediately after it:

```yaml
is_component: true
restricted: true
```

Do the same for the other five files (each already has an `is_component: true` line; add `restricted: true` right after it). If a file's exact surrounding lines differ, just add `restricted: true` as its own top-level line anywhere among the top-level keys (YAML key order is irrelevant).

- [ ] **Step 2: Verify the field is recognized (no unknown-field issues) + all six tagged**

Run: `grep -rl "restricted: true" _datafiles/world/dogmud/items/materials-40000/ | wc -l`
Expected: `6`

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/items/materials-40000/40169-warden_core.yaml \
        _datafiles/world/dogmud/items/materials-40000/40171-hull_filament.yaml \
        _datafiles/world/dogmud/items/materials-40000/40174-warden_prime_casing.yaml \
        _datafiles/world/dogmud/items/materials-40000/40191-resonant_vox_core.yaml \
        _datafiles/world/dogmud/items/materials-40000/40195-void_quenched_obsidian_core.yaml \
        _datafiles/world/dogmud/items/materials-40000/40196-warden_chassis_loom.yaml
git commit -m "content(items): tag six crash-ship warden components restricted (econ #2.5)"
```

---

## Task 4: Full build, boot smoke test, docs

**Files:**
- Modify: `PATCH_NOTES.md` (new dated entry at top)
- Modify: `docs/PATH_TO_1.0.md` (mark #2.5 done)

- [ ] **Step 1: Build the whole project**

Run: `go build ./...`
Expected: no output (success).

- [ ] **Step 2: Run touched-package tests**

Run: `go test ./internal/items/ ./modules/auctions/`
Expected: all `ok`.

- [ ] **Step 3: Boot smoke test (per pre-push SOP)**

Nuke instance saves (do NOT touch `shops/`), then boot and confirm a clean load past
data-file loading — the six seeded YAMLs must load without a panic (the `restricted` field is
now a known key):

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```

Run the server (`go run .`), confirm it reaches `Server Ready` with
`itemspec.LoadDataFiles() itemLoadedCount=...` and no panic, then stop it.

- [ ] **Step 4: Add the patch note**

Prepend to `PATCH_NOTES.md` (under `# DOGMud Patch Notes`):

```markdown
## 2026-07-15 — Auction buyers: the Crown takes an interest

A new bidder watches the auction block: an agent of the Crown who buys up restricted goods —
the strange salvage recovered from the wreck in the wastes among them — and spends freely to
get them off the open market. Bring such a find to auction and you'll usually find a ready
buyer, though a determined bidder can still outbid the Crown for something they mean to keep.
```

- [ ] **Step 5: Mark #2.5 done in the roadmap**

In `docs/PATH_TO_1.0.md`, change the econ arc **#2.5 Official** line from `⬜` to `✅` with a
one-line completion note dated 2026-07-15 referencing this plan + spec, in the same style as
the #2.1–#2.4 entries above it. (Optionally note the econ NPC-buyer sub-arc #2 is now fully done.)

- [ ] **Step 6: Commit**

```bash
git add PATCH_NOTES.md docs/PATH_TO_1.0.md
git commit -m "docs(auctions): patch notes + roadmap for Official buyer (econ #2.5)"
```

---

## Notes for the implementer

- **Test-var hygiene:** the archetype tests mutate package vars (`officialEnabled`,
  `officialPremium`). `TestOfficial_DisabledDeclinesEverything` restores `officialEnabled` with
  a deferred closure — follow that pattern for any var you flip.
- **Sink mechanism:** the Official deliberately has no `Receive` method. In `newRoundHandler`'s
  NPC-win path, only a buyer implementing `auctionWinReceiver` takes custody; everyone else's
  won item is not re-homed (it sinks). Do not add a `Receive` to the Official.
- **Registry persistence is automatic:** `save()`/`load()` iterate `npcBuyers` and snapshot/
  restore any buyer whose `Wallet()` is non-nil, keyed by `Name()`. The Official's wallet is
  non-nil, so `WalletBalances["The Crown Assessor"]` persists with zero extra wiring.
- **Deep pockets = large Cap, not a regen knob:** the regen loop (auctions.go:645-648) applies
  one shared rate to every wallet. The Official's depth comes from its 25000 Cap literal; there
  is intentionally no per-Official regen knob.
- **No-hard-numbers rule** applies to any player-facing text you add — the patch note and win
  flavor are descriptive, not numeric.
```
