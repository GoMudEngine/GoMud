# Guild treasury + vault Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax.

**Goal:** A shared guild gold treasury + item vault: any member deposits/donates; the leader (and, if delegated, officers) withdraw/take. Persisted in the guild YAML.

**Architecture:** New `Guild` fields + `CanWithdraw` helper + registry ops (deposit/withdraw/donate/take/delegate), all tested; command handlers are thin and route through the registry. Gold moves member-bank↔treasury; item `take` is loss-guarded via `StoreItem`.

**Spec:** `docs/superpowers/specs/2026-07-15-guild-treasury-vault-design.md`

---

## Task 1: Model fields + `CanWithdraw` + config

**Files:** `internal/guilds/guilds.go`, `internal/guilds/guilds_test.go`, `internal/configs/config.balance.go`, `config.balance.misc.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/guilds/guilds_test.go`:

```go
func TestCanWithdraw(t *testing.T) {
	g := &Guild{LeaderUserId: 1, Members: []GuildMember{
		{UserId: 1, Rank: RankLeader}, {UserId: 2, Rank: RankOfficer}, {UserId: 3, Rank: RankMember},
	}}
	if !g.CanWithdraw(1) {
		t.Error("leader always can withdraw")
	}
	if g.CanWithdraw(2) || g.CanWithdraw(3) {
		t.Error("officer/member cannot withdraw when not delegated")
	}
	g.TreasuryDelegated = true
	if !g.CanWithdraw(2) {
		t.Error("delegated officer should be able to withdraw")
	}
	if g.CanWithdraw(3) {
		t.Error("member still cannot withdraw even when delegated")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/guilds/ -run TestCanWithdraw -v` → FAIL (fields/method undefined).

- [ ] **Step 3: Add fields + helper**

In `internal/guilds/guilds.go`, add the `items` import and the fields to `Guild`:

```go
	Treasury          int          `yaml:"treasury,omitempty"`
	Vault             []items.Item `yaml:"vault,omitempty"`
	TreasuryDelegated bool         `yaml:"treasurydelegated,omitempty"` // officers may withdraw when true
```
(import `"github.com/GoMudEngine/GoMud/internal/items"`.)

Add the helper:
```go
// CanWithdraw reports whether userId may withdraw gold / take vault items: the
// leader always, and officers when treasury access is delegated.
func (g *Guild) CanWithdraw(userId int) bool {
	if g.IsLeader(userId) {
		return true
	}
	return g.TreasuryDelegated && g.CanManage(userId)
}
```

- [ ] **Step 4: Config knob**

`config.balance.go` (near `GuildFoundingCost`):
```go
	GuildVaultCapacity          ConfigInt   `yaml:"GuildVaultCapacity"`                 // Max items a guild vault holds (default 100).
```
`config.balance.misc.go` validator:
```go
	if b.GuildVaultCapacity <= 0 {
		b.GuildVaultCapacity = 100
	}
```

- [ ] **Step 5: Run test + build + commit**

Run: `go test ./internal/guilds/ -run TestCanWithdraw -v` → PASS; `go build ./internal/guilds/ ./internal/configs/` → success.
```bash
git add internal/guilds/guilds.go internal/guilds/guilds_test.go internal/configs/config.balance.go internal/configs/config.balance.misc.go
git commit -m "feat(guilds): treasury/vault fields + CanWithdraw + GuildVaultCapacity"
```

---

## Task 2: Registry ops

**Files:** `internal/guilds/registry.go`, `internal/guilds/registry_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/guilds/registry_test.go`:

```go
func TestRegistry_Treasury(t *testing.T) {
	defer SetDataDirForTest(t.TempDir())()
	resetRegistry()
	if _, err := Create("TR", "Treasurers", 1, "L"); err != nil {
		t.Fatal(err)
	}

	if err := DepositGold("TR", 500); err != nil {
		t.Fatal(err)
	}
	g, _ := Get("TR")
	if g.Treasury != 500 {
		t.Fatalf("treasury = %d, want 500", g.Treasury)
	}
	if err := WithdrawGold("TR", 600); err == nil {
		t.Error("withdrawing more than the treasury should fail")
	}
	if g.Treasury != 500 {
		t.Error("failed withdraw must not mutate treasury")
	}
	if err := WithdrawGold("TR", 200); err != nil {
		t.Fatal(err)
	}
	if g.Treasury != 300 {
		t.Errorf("treasury = %d, want 300", g.Treasury)
	}

	// Vault.
	if err := DonateItem("TR", items.Item{ItemId: 1}, 2); err != nil {
		t.Fatal(err)
	}
	DonateItem("TR", items.Item{ItemId: 2}, 2)
	if err := DonateItem("TR", items.Item{ItemId: 3}, 2); err == nil {
		t.Error("donating past capacity should fail")
	}
	if len(g.Vault) != 2 {
		t.Fatalf("vault len = %d, want 2", len(g.Vault))
	}
	it, err := TakeItem("TR", 0)
	if err != nil || it.ItemId != 1 {
		t.Fatalf("take[0] = %+v err=%v, want ItemId 1", it, err)
	}
	if len(g.Vault) != 1 || g.Vault[0].ItemId != 2 {
		t.Errorf("vault after take = %+v", g.Vault)
	}
	if _, err := TakeItem("TR", 5); err == nil {
		t.Error("bad index should error")
	}

	if err := SetTreasuryDelegated("TR", true); err != nil {
		t.Fatal(err)
	}
	if !g.TreasuryDelegated {
		t.Error("delegation not set")
	}
}
```

(Needs `"github.com/GoMudEngine/GoMud/internal/items"` in the test imports.)

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/guilds/ -run TestRegistry_Treasury -v` → FAIL (ops undefined).

- [ ] **Step 3: Implement the ops**

Add to `internal/guilds/registry.go` (add the `items` import), each mirroring the existing
mutators (lock, mutate, unlock, `Save`):

```go
func DepositGold(tag string, amount int) error {
	g, ok := Get(tag)
	if !ok {
		return fmt.Errorf("no such guild")
	}
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	registryMu.Lock()
	g.Treasury += amount
	registryMu.Unlock()
	return Save(g)
}

func WithdrawGold(tag string, amount int) error {
	g, ok := Get(tag)
	if !ok {
		return fmt.Errorf("no such guild")
	}
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	registryMu.Lock()
	if amount > g.Treasury {
		registryMu.Unlock()
		return fmt.Errorf("the treasury only holds %d gold", g.Treasury)
	}
	g.Treasury -= amount
	registryMu.Unlock()
	return Save(g)
}

func DonateItem(tag string, it items.Item, capacity int) error {
	g, ok := Get(tag)
	if !ok {
		return fmt.Errorf("no such guild")
	}
	registryMu.Lock()
	if capacity > 0 && len(g.Vault) >= capacity {
		registryMu.Unlock()
		return fmt.Errorf("the guild vault is full")
	}
	g.Vault = append(g.Vault, it)
	registryMu.Unlock()
	return Save(g)
}

func TakeItem(tag string, index int) (items.Item, error) {
	g, ok := Get(tag)
	if !ok {
		return items.Item{}, fmt.Errorf("no such guild")
	}
	registryMu.Lock()
	if index < 0 || index >= len(g.Vault) {
		registryMu.Unlock()
		return items.Item{}, fmt.Errorf("no such vault item")
	}
	it := g.Vault[index]
	g.Vault = append(g.Vault[:index], g.Vault[index+1:]...)
	registryMu.Unlock()
	return it, Save(g)
}

func SetTreasuryDelegated(tag string, on bool) error {
	g, ok := Get(tag)
	if !ok {
		return fmt.Errorf("no such guild")
	}
	registryMu.Lock()
	g.TreasuryDelegated = on
	registryMu.Unlock()
	return Save(g)
}
```

- [ ] **Step 4: Run test + commit**

Run: `go test ./internal/guilds/ -run TestRegistry_Treasury -v` → PASS.
```bash
git add internal/guilds/registry.go internal/guilds/registry_test.go
git commit -m "feat(guilds): treasury/vault registry ops"
```

---

## Task 3: Command subcommands + `findVaultItem`

**Files:** `internal/usercommands/guild.go`, `internal/usercommands/guild_test.go`, help templates

- [ ] **Step 1: Write the failing test (findVaultItem)**

Add to `internal/usercommands/guild_test.go`:

```go
func TestFindVaultItem(t *testing.T) {
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		101: {ItemId: 101, Name: "iron sword"},
		102: {ItemId: 102, Name: "healing potion"},
	})()
	vault := []items.Item{items.New(101), items.New(102)}
	if idx, ok := findVaultItem(vault, "healing potion"); !ok || idx != 1 {
		t.Errorf("find = %d,%v, want 1,true", idx, ok)
	}
	if _, ok := findVaultItem(vault, "nonexistent"); ok {
		t.Error("miss should return ok=false")
	}
	if _, ok := findVaultItem(nil, "x"); ok {
		t.Error("empty vault -> not found")
	}
}
```

(Add `"github.com/GoMudEngine/GoMud/internal/items"` to the test imports if not present.)

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/usercommands/ -run TestFindVaultItem -v` → FAIL (`findVaultItem` undefined).

- [ ] **Step 3: Implement `findVaultItem` + subcommands**

Add `findVaultItem` to `internal/usercommands/guild.go` (guild.go already imports `items`? add
if not):

```go
// findVaultItem fuzzy-matches a vault item by name and returns its index.
func findVaultItem(vault []items.Item, name string) (int, bool) {
	if strings.TrimSpace(name) == "" || len(vault) == 0 {
		return 0, false
	}
	closeMatch, fullMatch := items.FindMatchIn(name, vault...)
	target := fullMatch
	if target.ItemId == 0 {
		target = closeMatch
	}
	if target.ItemId == 0 {
		return 0, false
	}
	for i, it := range vault {
		if it.Equals(target) {
			return i, true
		}
	}
	return 0, false
}
```

Add the subcommands and wire them into the `Guild` dispatcher switch (`deposit`, `withdraw`,
`donate`, `take`, `treasury`):

- `guildDeposit(user, remainder)`: `guilds.GetByUser`; parse amount (`all` → `Bank`); reject
  `<=0` or `> Bank`; `Bank -= amt` + `EquipmentChange{BankChange:-amt}`; `guilds.DepositGold`;
  confirm + `announceGuild`.
- `guildWithdraw(user, remainder)`: `g.CanWithdraw(user.UserId)` else refuse; parse amount (`all`
  → `Treasury`); `guilds.WithdrawGold` (errors if > treasury); `Bank += amt` + event; confirm +
  announce.
- `guildDonate(user, remainder)`: resolve `user.Character.FindInBackpack(name)`;
  `guilds.DonateItem(g.Tag, item, int(configs.GetBalanceConfig().GuildVaultCapacity))` (errors if
  full); on success `user.Character.RemoveItem(item)`; confirm + announce.
- `guildTake(user, remainder)`: `g.CanWithdraw` else refuse; `idx, ok := findVaultItem(g.Vault,
  name)`; **item-loss guard**: `it := g.Vault[idx]`; if `!user.Character.StoreItem(it)` → "your
  pack is too full" and DO NOT remove from vault; else `guilds.TakeItem(g.Tag, idx)` +
  `ItemOwnership` event; confirm + announce.
- `guildTreasury(user, remainder)`: if `remainder` starts with `delegate` → leader-only
  `guilds.SetTreasuryDelegated(on/off)`; else show `Treasury` gold, numbered vault contents,
  `len(Vault)/GuildVaultCapacity`, and delegation state.

Dispatcher cases:
```go
	case "deposit":
		guildDeposit(user, remainder)
	case "withdraw":
		guildWithdraw(user, remainder)
	case "donate":
		guildDonate(user, remainder)
	case "take":
		guildTake(user, remainder)
	case "treasury", "bank", "vault":
		guildTreasury(user, remainder)
```

- [ ] **Step 4: Help**

Add the treasury subcommands to `guild.template` (both copies): deposit/withdraw/donate/take/
treasury/treasury delegate, with `(leader/officer)` annotations.

- [ ] **Step 5: Run test + build + full usercommands test + commit**

Run: `go test ./internal/usercommands/ -run TestFindVaultItem -v` → PASS
Run: `go build ./... && go test ./internal/usercommands/` → PASS.
```bash
git add internal/usercommands/guild.go internal/usercommands/guild_test.go _datafiles/world/dogmud/templates/help/guild.template _datafiles/world/default/templates/help/guild.template
git commit -m "feat(guilds): treasury/vault commands (deposit/withdraw/donate/take/treasury)"
```

---

## Task 4: Boot smoke test + docs

- [ ] **Step 1: Build + touched tests**

Run: `go build ./... && go test ./internal/guilds/ ./internal/usercommands/ ./internal/configs/`
Expected: all `ok`.

- [ ] **Step 2: Boot smoke test**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```
Run `go run .`; confirm `Server Ready`, no panic; stop.

- [ ] **Step 3: Docs**

- `PATCH_NOTES.md`: brief entry (guild treasury + vault; deposit/donate, leader withdraws, delegate to officers).
- `docs/PATH_TO_1.0.md` §3 guilds line: note treasury done; ranks-polish remains.

- [ ] **Step 4: Commit**

```bash
git add PATCH_NOTES.md docs/PATH_TO_1.0.md
git commit -m "docs(guilds): patch notes + roadmap for guild treasury + vault"
```

---

## Notes for the implementer

- **Verify before coding:** `guild.go` imports (add `items`, `configs` already there),
  `Character.FindInBackpack`/`RemoveItem`/`StoreItem`/`Bank`, `events.ItemOwnership`, the
  `strconv`/`all`-amount parsing (mirror `bank.go`).
- **Item-loss guard on `take`:** attempt `StoreItem` FIRST; only remove from the vault if it
  succeeded — a full pack never loses a vault item.
- **Value conservation:** gold leaves the member's bank only after (or atomically with) landing
  in the treasury; a rejected op mutates nothing. Same for items.
- **Rank gates BEFORE mutation** on withdraw/take/delegate; deposit/donate are open to members.
- No emoji; ANSI only; numeric gold in notices is fine (bank convention).
```
