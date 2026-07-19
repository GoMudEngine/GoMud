# Guild treasury + vault

**Date:** 2026-07-15
**Status:** Design (approved; pending review)
**Arc:** Social guilds — sub-project #3 (treasury). After membership core + chat/who-tag.
**Depends on:** `internal/guilds` (registry + durable persistence, shipped).

---

## 1. Goal

A shared guild bank: members pool gold and gear; the leader (and, if delegated, officers)
manage the funds. A guild's economic backbone and a real reason to stick with one.

---

## 2. Approved decisions

1. **Deposit is open** (any member); **withdrawal is leader-only by default**, with a
   **leader-toggled delegation** that lets officers withdraw too (`TreasuryDelegated bool`).
   Unified over gold + items for v1.
2. **Gold moves member-bank ↔ guild-treasury** (deposit from bank; withdraw to bank), matching
   the founding fee.
3. **Vault** = a flat `[]items.Item` on the guild, capped by `GuildVaultCapacity` (config).
4. Withdrawal of an item respects the taker's carry capacity — if it won't fit, it stays in
   the vault (no item loss; same lesson as the mail/auction fixes).

---

## 3. Data model — `internal/guilds/guilds.go`

Add to `Guild`:
```go
	Treasury         int          `yaml:"treasury,omitempty"`
	Vault            []items.Item `yaml:"vault,omitempty"`
	TreasuryDelegated bool        `yaml:"treasurydelegated,omitempty"` // officers may withdraw when true
```
(`guilds` importing `items` is cycle-safe — `items` imports nothing from `guilds`.)

Permission helper on `*Guild`:
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

---

## 4. Registry ops — `internal/guilds/registry.go` (all persist)

- `DepositGold(tag string, amount int) error` — `Treasury += amount`.
- `WithdrawGold(tag string, amount int) error` — errors if `amount > Treasury`; else `Treasury -= amount`.
- `DonateItem(tag string, it items.Item, cap int) error` — errors if `len(Vault) >= cap`; else append.
- `TakeItem(tag string, index int) (items.Item, error)` — remove + return the vault item at index
  (index-based, so the command resolves the item to an index first).
- `SetTreasuryDelegated(tag string, on bool) error`.

Guarded by `registryMu` like the existing mutators; each `Save`s.

---

## 5. Commands — `internal/usercommands/guild.go` (new subcommands)

- **`guild deposit <amount|all>`** (any member): `amount` from the depositor's **bank**
  (`Character.Bank`); `all` = whole bank; reject if `Bank < amount` or `amount <= 0`. On success:
  `Bank -= amount` + `EquipmentChange{BankChange:-amount}`; `guilds.DepositGold`; confirm +
  announce to the guild.
- **`guild withdraw <amount|all>`** (`CanWithdraw`): reject if `amount > Treasury`; else
  `guilds.WithdrawGold`; `Character.Bank += amount` + event; confirm + announce.
- **`guild donate <item>`** (any member): resolve an item in the donor's backpack
  (`FindInBackpack`); reject if vault full (`GuildVaultCapacity`); `guilds.DonateItem` then
  `Character.RemoveItem`; confirm + announce.
- **`guild take <item>`** (`CanWithdraw`): resolve the item in the vault by name → index; check
  the taker's carry (`StoreItem` returns false when over ~2× capacity) — attempt `StoreItem`
  FIRST; only on success `guilds.TakeItem(index)` to remove it from the vault (so a full pack
  never loses the item). Confirm + announce.
- **`guild treasury`** (any member): show `Treasury` gold, the vault contents (numbered list),
  vault count / capacity, and whether officer withdrawal is delegated.
- **`guild treasury delegate <on|off>`** (leader only): set `TreasuryDelegated`; confirm.

Every withdrawal/take/delegate gates on rank BEFORE mutating; deposit/donate are open to members.
All gold/item amounts in bank-style notices (numeric gold is fine, like the bank command).

Item resolution: a helper `findVaultItem(g, name) (index int, ok bool)` fuzzy-matches the vault
by item name (reuse `items.FindMatchIn` over the vault's names), returning the index for `TakeItem`.

---

## 6. Config — `internal/configs/config.balance.go` + validator

`GuildVaultCapacity` (ConfigInt, default **100** slots; validator `if <= 0 { = 100 }`).

---

## 7. Testing

- **`CanWithdraw`**: leader true always; officer true only when delegated; member always false.
- **Registry ops** (temp-dir): deposit/withdraw gold (withdraw > treasury rejected, no mutation);
  donate respects capacity (full vault rejected); `TakeItem` removes + returns the right index,
  bad index errors; `SetTreasuryDelegated` persists.
- **`findVaultItem`**: fuzzy match returns the right index; miss returns ok=false.
- Command handlers are integration (boot/manual) — keep logic in the tested registry/helpers.
- Value conservation (unit or careful manual): gold deposited leaves the member's bank exactly
  once and lands in the treasury; withdraw is the inverse; a rejected op mutates nothing. Item
  donate removes from backpack only after it's in the vault; `take` removes from vault only after
  it's in the backpack.
- Full suite green + boot clean.

---

## 8. Out of scope (later)

- Per-member donation/withdrawal audit log (the stub's `Donations` idea) — nice for trust, defer.
- Split gold-vs-item delegation, per-member treasurer grants, withdrawal limits/quotas.
- Vault item stacking (flat list is fine for v1).

---

## 9. Files touched

- `internal/guilds/guilds.go` — `Treasury`/`Vault`/`TreasuryDelegated` fields + `CanWithdraw`.
- `internal/guilds/registry.go` — deposit/withdraw/donate/take/delegate ops.
- `internal/guilds/*_test.go` — the tests above.
- `internal/usercommands/guild.go` — the new subcommands + `findVaultItem`.
- `internal/configs/config.balance.go` + `config.balance.misc.go` — `GuildVaultCapacity`.
- `_datafiles/world/dogmud/templates/help/guild.template` (+ default) — new subcommands.
- `PATCH_NOTES.md`, `docs/PATH_TO_1.0.md`.
