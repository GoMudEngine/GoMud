# NPC Buyers — Craftsperson & Adventurer Implementation Plan (Econ #2.2 + #2.3)

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development or superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Two more `NpcBuyer` archetypes (craftsperson, adventurer) plus a `Flavor()` interface method for distinct win broadcasts.

**Architecture:** All in `modules/auctions/npc_buyers.go` (+ a one-line resolution wiring in `auctions.go`). Plugs into the #2.1 framework — no engine changes.

**Spec:** `docs/superpowers/specs/2026-07-14-npc-buyers-crafter-adventurer-design.md`

---

## Task 1: `Flavor()` interface method + resolution wiring

**Files:** Modify `modules/auctions/npc_buyers.go`, `modules/auctions/auctions.go`

- [ ] **Step 1: Add `Flavor()` to the interface + collector**

In `npc_buyers.go`, add to the `NpcBuyer` interface:
```go
	Flavor() string // trailing phrase in the win broadcast, e.g. "for their collection"
```
Add to `collector`:
```go
func (c *collector) Flavor() string { return "for their collection" }
```

- [ ] **Step 2: Route the resolution broadcast through `Flavor()`**

In `auctions.go`, in the NPC-win broadcast block (the `if auctionNow.HighestBidIsNPC {` loop added in
#2.1), replace the hardcoded "for their collection" with the buyer's flavor:
```go
			flavor := "for their collection"
			if b := buyerByName(auctionNow.HighestBidderName); b != nil {
				flavor = b.Flavor()
			}
			for _, uid := range users.GetOnlineUserIds() {
				if u := users.GetByUserId(uid); u != nil {
					if on := u.GetConfigOption(`auction`); on == nil || on.(bool) {
						u.SendText(messaging.CategoryBroadcast, fmt.Sprintf(`<ansi fg="yellow"><ansi fg="username">%s</ansi> has acquired the <ansi fg="item">%s</ansi> %s.</ansi>`, auctionNow.HighestBidderName, auctionNow.ItemData.DisplayName(), flavor))
					}
				}
			}
```

- [ ] **Step 3: Build (existing tests still pass)**

Run: `go build ./modules/auctions/ && go test ./modules/auctions/`
Expected: build clean, all #2.1 tests still green.

- [ ] **Step 4: Commit**

```bash
git add modules/auctions/npc_buyers.go modules/auctions/auctions.go
git commit -m "feat(auctions): NpcBuyer.Flavor() for per-archetype win broadcasts"
```

---

## Task 2: Craftsperson + Adventurer archetypes

**Files:** Modify `modules/auctions/npc_buyers.go`, `modules/auctions/npc_buyers_test.go`, `modules/auctions/auctions.go` (config)

- [ ] **Step 1: Write the failing tests**

Append to `npc_buyers_test.go`:
```go
func TestCraftsperson(t *testing.T) {
	craftMinValue = 50
	craftPremium = 1.0
	c := &craftsperson{name: "Ordwin", wallet: &NpcWallet{Balance: 5000, Cap: 5000}}
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		301: {ItemId: 301, Name: "rare ore", IsComponent: true, Value: 200},
		302: {ItemId: 302, Name: "junk scrap", IsComponent: true, Value: 5},
		303: {ItemId: 303, Name: "sword", Type: items.Weapon, Value: 900},
	})()
	if !c.Interested(items.New(301)) {
		t.Error("crafter should want a valuable component")
	}
	if c.Interested(items.New(302)) {
		t.Error("crafter should NOT want a near-worthless component")
	}
	if c.Interested(items.New(303)) {
		t.Error("crafter should NOT want a non-component weapon")
	}
	if c.MaxBid(items.New(301)) != 200 {
		t.Errorf("MaxBid=%d want 200", c.MaxBid(items.New(301)))
	}
	if c.Flavor() != "for their workshop" {
		t.Errorf("flavor=%q", c.Flavor())
	}
}

func TestAdventurer(t *testing.T) {
	advMinValue = 300
	advPremium = 0.9
	a := &adventurer{name: "Kest", wallet: &NpcWallet{Balance: 5000, Cap: 5000}}
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		311: {ItemId: 311, Name: "enchanted blade", Type: items.Weapon, Value: 1000, StatMods: map[string]int{"strength": 5}},
		312: {ItemId: 312, Name: "plain blade", Type: items.Weapon, Value: 1000},
		313: {ItemId: 313, Name: "ore", IsComponent: true, Value: 1000, StatMods: map[string]int{"strength": 5}},
	})()
	if !a.Interested(items.New(311)) {
		t.Error("adventurer should want stat-bearing gear")
	}
	if a.Interested(items.New(312)) {
		t.Error("adventurer should NOT want plain gear (no statmods)")
	}
	if a.Interested(items.New(313)) {
		t.Error("adventurer should NOT want a non-equipment component")
	}
	if a.MaxBid(items.New(311)) != 900 { // 1000 * 0.9
		t.Errorf("MaxBid=%d want 900", a.MaxBid(items.New(311)))
	}
	if a.Flavor() != "to gear up" {
		t.Errorf("flavor=%q", a.Flavor())
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./modules/auctions/ -run 'TestCraftsperson|TestAdventurer' -v`
Expected: FAIL — types undefined.

- [ ] **Step 3: Add the archetypes + knobs + registry**

In `npc_buyers.go`, add the knobs to the `var (...)` block:
```go
	craftMinValue = 50
	craftPremium  = 1.0
	advMinValue   = 300
	advPremium    = 0.9
```
Add the two structs:
```go
// ── Craftsperson archetype: buys valuable crafting materials ──
type craftsperson struct {
	name   string
	wallet *NpcWallet
}

func (c *craftsperson) Name() string { return c.name }
func (c *craftsperson) Interested(item items.Item) bool {
	spec := item.GetSpec()
	return spec.IsComponent && spec.Value >= craftMinValue
}
func (c *craftsperson) MaxBid(item items.Item) int { return int(float64(item.GetSpec().Value) * craftPremium) }
func (c *craftsperson) Wallet() *NpcWallet         { return c.wallet }
func (c *craftsperson) Flavor() string             { return "for their workshop" }

// ── Adventurer archetype: buys usable gear upgrades (stat-bearing equipment) ──
type adventurer struct {
	name   string
	wallet *NpcWallet
}

func (a *adventurer) Name() string { return a.name }
func (a *adventurer) Interested(item items.Item) bool {
	spec := item.GetSpec()
	return isEquipment(spec.Type) && len(spec.StatMods) > 0 && spec.Value >= advMinValue
}
func (a *adventurer) MaxBid(item items.Item) int { return int(float64(item.GetSpec().Value) * advPremium) }
func (a *adventurer) Wallet() *NpcWallet         { return a.wallet }
func (a *adventurer) Flavor() string             { return "to gear up" }
```
Add them to the `npcBuyers` registry:
```go
	&craftsperson{name: "Master Ordwin", wallet: &NpcWallet{Balance: 6000, Cap: 6000}},
	&adventurer{name: "Sellsword Kest", wallet: &NpcWallet{Balance: 6000, Cap: 6000}},
```

- [ ] **Step 4: Add config knobs** in `auctions.go` `load()` (alongside the collector knobs):
```go
	if v, ok := mod.plug.Config.Get(`CraftspersonMinValue`).(int); ok && v > 0 {
		craftMinValue = v
	}
	if v, ok := mod.plug.Config.Get(`CraftspersonPremium`).(float64); ok && v > 0 {
		craftPremium = v
	}
	if v, ok := mod.plug.Config.Get(`AdventurerMinValue`).(int); ok && v > 0 {
		advMinValue = v
	}
	if v, ok := mod.plug.Config.Get(`AdventurerPremium`).(float64); ok && v > 0 {
		advPremium = v
	}
```

- [ ] **Step 5: Run tests + build**

Run: `go test ./modules/auctions/ -run 'TestCraftsperson|TestAdventurer' -v && go build ./modules/auctions/`
Expected: PASS + build clean.

- [ ] **Step 6: Commit**

```bash
git add modules/auctions/npc_buyers.go modules/auctions/npc_buyers_test.go modules/auctions/auctions.go
git commit -m "feat(auctions): craftsperson + adventurer NPC buyers"
```

---

## Task 3: Verify + patch notes

**Files:** Modify `PATCH_NOTES.md`

- [ ] **Step 1: Full build + suite + boot-smoke**

```bash
go build ./... && go test ./... 2>&1 | grep -E "^(FAIL|---)" | head   # expect none
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run .   # wait for "Server Ready", no panic
```

- [ ] **Step 2: Patch notes**

Prepend to `PATCH_NOTES.md`:
```markdown
## 2026-07-14 — More auction buyers: crafters and adventurers

Two more kinds of NPC now watch the auction block: a master craftsperson who
buys up worthwhile crafting materials for their workshop, and a wandering
adventurer looking to buy a real gear upgrade. Like the collectors, they bid
only up to what a piece is worth to them — so you can always outbid them for
something you want — and they spend their own limited coin.
```

- [ ] **Step 3: Commit**

```bash
git add PATCH_NOTES.md
git commit -m "docs(auctions): patch notes for craftsperson + adventurer buyers"
```

---

## Final verification

- [ ] `go build ./...` clean; `go test ./modules/auctions/` + full suite pass.
- [ ] Boot-smoke clean.
