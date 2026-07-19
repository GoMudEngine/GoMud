# Player-to-player mudmail Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A player `mail` command that sends a message + on-hand gold + a backpack item to one named recipient (online or offline), with a per-sender send cooldown; plus a fix so an over-capacity reader never loses a mailed item.

**Architecture:** New `mail.go` command reusing `user.StartPrompt` (like admin `mudmail`), with pure testable helpers (`mailOnCooldown`, `resolveMailRecipient`, `applyMailReceipt`). Gold leaves the sender's purse and the item leaves their backpack at send; the existing `inbox` read path credits gold→bank and item→backpack on receipt. A new `Character.LastMailSentRound` field + `MailSendCooldownRounds` config drive the cooldown.

**Tech Stack:** Go, GoMud user/command layer, testify.

**Spec:** `docs/superpowers/specs/2026-07-15-player-mudmail-design.md`

---

## File Structure

- **Modify** `internal/configs/config.balance.go` — `MailSendCooldownRounds` field.
- **Modify** `internal/configs/config.balance.misc.go` — validator default (10).
- **Modify** `internal/characters/character.go` — `LastMailSentRound uint64` field.
- **Create** `internal/usercommands/mail.go` — `Mail` command + helpers.
- **Create** `internal/usercommands/mail_test.go` — helper tests.
- **Modify** `internal/usercommands/usercommands.go` — register `mail`.
- **Modify** `internal/usercommands/inbox.go` — item-loss fix (use `applyMailReceipt`).
- **Modify** `PATCH_NOTES.md`, `docs/PATH_TO_1.0.md` — at the end.

---

## Task 1: Config knob + Character field

**Files:**
- Modify: `internal/configs/config.balance.go` (add field near other misc knobs)
- Modify: `internal/configs/config.balance.misc.go` (`validateMisc` / equivalent — the file with `ForagerRestDurationRounds` default)
- Modify: `internal/characters/character.go` (add field)
- Test: `internal/configs/config.balance.misc_test.go` may not exist; add a focused test file `internal/configs/config.balance.mail_test.go`

- [ ] **Step 1: Write the failing config test**

Create `internal/configs/config.balance.mail_test.go`:

```go
package configs

import "testing"

func TestValidate_MailSendCooldownRoundsDefault(t *testing.T) {
	b := &Balance{}
	b.validateMisc()
	if int(b.MailSendCooldownRounds) != 10 {
		t.Errorf("MailSendCooldownRounds default = %d, want 10", int(b.MailSendCooldownRounds))
	}
}
```

(Verified: the validator is `func (b *Balance) validateMisc()` in `internal/configs/config.balance.misc.go`, where `ForagerRestDurationRounds` is defaulted.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/configs/ -run TestValidate_MailSendCooldownRoundsDefault -v`
Expected: FAIL — `MailSendCooldownRounds` undefined.

- [ ] **Step 3: Add the config field**

In `internal/configs/config.balance.go`, add near a misc knob (e.g., after `StorageSeizureMinValue`):

```go
	MailSendCooldownRounds      ConfigInt   `yaml:"MailSendCooldownRounds"`             // Rounds a player must wait between sending mail (anti-spam; default 10). Set to 1 for effectively no cooldown.
```

- [ ] **Step 4: Add the validator default**

In the validator file that defaults `ForagerRestDurationRounds` (grep to confirm — likely `config.balance.misc.go`), add:

```go
	if b.MailSendCooldownRounds <= 0 {
		b.MailSendCooldownRounds = 10
	}
```

- [ ] **Step 5: Add the Character field**

In `internal/characters/character.go`, add to the `Character` struct (near other tracking fields like `StorageFeeLastMonth`):

```go
	LastMailSentRound uint64 `yaml:"lastmailsentround,omitempty"` // round of the character's last sent mail (mail send cooldown)
```

- [ ] **Step 6: Run test + build**

Run: `go test ./internal/configs/ -run TestValidate_MailSendCooldownRoundsDefault -v` → PASS
Run: `go build ./internal/characters/ ./internal/configs/` → success

- [ ] **Step 7: Commit**

```bash
git add internal/configs/config.balance.go internal/configs/config.balance.misc.go internal/configs/config.balance.mail_test.go internal/characters/character.go
git commit -m "feat(mudmail): MailSendCooldownRounds config + Character.LastMailSentRound"
```

---

## Task 2: Pure helpers — cooldown + recipient resolution

**Files:**
- Create: `internal/usercommands/mail.go` (helpers only in this task; the command in Task 4)
- Test: `internal/usercommands/mail_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/usercommands/mail_test.go`:

```go
package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func TestMailOnCooldown(t *testing.T) {
	// lastSent 100, cooldown 10 -> blocked until round 110.
	if !mailOnCooldown(100, 105, 10) {
		t.Error("round 105 is within 100+10; should be on cooldown")
	}
	if mailOnCooldown(100, 110, 10) {
		t.Error("round 110 == 100+10; cooldown elapsed")
	}
	if mailOnCooldown(100, 200, 10) {
		t.Error("well past cooldown should be clear")
	}
	if mailOnCooldown(0, 5, 10) {
		t.Error("never-sent (lastSent 0) should never be on cooldown")
	}
	if mailOnCooldown(100, 101, 0) {
		t.Error("cooldown 0 (disabled) should never block")
	}
}

func TestResolveMailRecipient(t *testing.T) {
	online := &users.UserRecord{UserId: 7, Character: &characters.Character{Name: "Onlinerecipient"}}

	onlineByName := func(name string) *users.UserRecord {
		if name == "Onlinerecipient" {
			return online
		}
		return nil
	}
	offlineSearch := func(name string) (int, string) {
		if name == "Offlinerecipient" {
			return 9, "offlineacct"
		}
		return 0, ""
	}

	// Online hit.
	rec, _, ok := resolveMailRecipient("Onlinerecipient", 1, onlineByName, offlineSearch)
	if !ok || !rec.online || rec.userId != 7 {
		t.Errorf("online resolve = %+v ok=%v, want {7, online}", rec, ok)
	}

	// Offline hit.
	rec, uname, ok := resolveMailRecipient("Offlinerecipient", 1, onlineByName, offlineSearch)
	if !ok || rec.online || rec.userId != 9 || uname != "offlineacct" {
		t.Errorf("offline resolve = %+v uname=%q ok=%v, want {9, offline, offlineacct}", rec, uname, ok)
	}

	// Not found.
	if _, _, ok := resolveMailRecipient("Nobody", 1, onlineByName, offlineSearch); ok {
		t.Error("unknown recipient should not resolve")
	}

	// Self-mail (online).
	if _, _, ok := resolveMailRecipient("Onlinerecipient", 7, onlineByName, offlineSearch); ok {
		t.Error("self-mail (online) must be rejected")
	}
	// Self-mail (offline).
	if _, _, ok := resolveMailRecipient("Offlinerecipient", 9, onlineByName, offlineSearch); ok {
		t.Error("self-mail (offline) must be rejected")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/usercommands/ -run 'TestMailOnCooldown|TestResolveMailRecipient' -v`
Expected: FAIL — `mailOnCooldown` / `resolveMailRecipient` / `mailRecipient` undefined.

- [ ] **Step 3: Implement the helpers**

Create `internal/usercommands/mail.go` with (command comes in Task 4):

```go
package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/users"
)

// mailOnCooldown reports whether a sender who last sent at lastSent is still
// within the cooldown window at round now. Disabled when cooldown <= 0 or the
// sender has never sent (lastSent == 0).
func mailOnCooldown(lastSent, now, cooldown uint64) bool {
	if cooldown == 0 || lastSent == 0 {
		return false
	}
	return now < lastSent+cooldown
}

// mailRecipient identifies a resolved mail target.
type mailRecipient struct {
	userId int
	online bool
}

// resolveMailRecipient resolves a recipient by character name (online first, then
// offline), rejecting a send to oneself. Lookups are injected for testability;
// the command wires users.GetByCharacterName + users.CharacterNameSearch.
// On an offline hit it returns the account username for the loader.
func resolveMailRecipient(
	name string,
	senderUserId int,
	onlineByName func(string) *users.UserRecord,
	offlineSearch func(string) (int, string),
) (rec mailRecipient, username string, ok bool) {

	if u := onlineByName(name); u != nil {
		if u.UserId == senderUserId {
			return mailRecipient{}, "", false // no self-mail
		}
		return mailRecipient{userId: u.UserId, online: true}, "", true
	}

	if uid, uname := offlineSearch(name); uid != 0 {
		if uid == senderUserId {
			return mailRecipient{}, "", false // no self-mail
		}
		return mailRecipient{userId: uid, online: false}, uname, true
	}

	return mailRecipient{}, "", false
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/usercommands/ -run 'TestMailOnCooldown|TestResolveMailRecipient' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/usercommands/mail.go internal/usercommands/mail_test.go
git commit -m "feat(mudmail): pure cooldown + recipient-resolution helpers"
```

---

## Task 3: Item-loss fix in inbox receive (`applyMailReceipt`)

**Files:**
- Modify: `internal/usercommands/mail.go` (add `applyMailReceipt`)
- Modify: `internal/usercommands/inbox.go` (use it in the read loop)
- Test: `internal/usercommands/mail_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/usercommands/mail_test.go`:

```go
import "github.com/GoMudEngine/GoMud/internal/items" // add to the existing import block

func TestApplyMailReceipt_GoldOnly(t *testing.T) {
	u := &users.UserRecord{UserId: 1, Character: &characters.Character{}}
	msg := &users.Message{Gold: 250}
	if !applyMailReceipt(u, msg) {
		t.Fatal("gold-only receipt should commit")
	}
	if u.Character.Bank != 250 {
		t.Errorf("bank = %d, want 250", u.Character.Bank)
	}
}

func TestApplyMailReceipt_ItemFits(t *testing.T) {
	items.RegisterTestItemSpec(&items.ItemSpec{ItemId: 8001, Name: "ring", Type: items.Ring, Value: 100}) // weightless -> fits
	u := &users.UserRecord{UserId: 1, Character: &characters.Character{}}
	itm := items.New(8001)
	msg := &users.Message{Gold: 50, Item: &itm}
	if !applyMailReceipt(u, msg) {
		t.Fatal("fitting item should commit")
	}
	if u.Character.Bank != 50 {
		t.Errorf("bank = %d, want 50", u.Character.Bank)
	}
	if len(u.Character.Items) != 1 {
		t.Errorf("backpack items = %d, want 1", len(u.Character.Items))
	}
}

func TestApplyMailReceipt_OverCapacityDefers(t *testing.T) {
	// A very heavy item can't fit a zero-stat character -> StoreItem fails.
	items.RegisterTestItemSpec(&items.ItemSpec{ItemId: 8002, Name: "anvil", Type: items.Weapon, Value: 100, Weight: 100000})
	u := &users.UserRecord{UserId: 1, Character: &characters.Character{}}
	itm := items.New(8002)
	msg := &users.Message{Gold: 999, Item: &itm}

	if applyMailReceipt(u, msg) {
		t.Fatal("over-capacity item receipt must defer (return false)")
	}
	// Nothing partial: no gold credited, no item stored.
	if u.Character.Bank != 0 {
		t.Errorf("bank = %d, want 0 (gold not credited when deferred)", u.Character.Bank)
	}
	if len(u.Character.Items) != 0 {
		t.Errorf("backpack items = %d, want 0", len(u.Character.Items))
	}
}
```

Update the test file's import block to include `"github.com/GoMudEngine/GoMud/internal/items"`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/usercommands/ -run TestApplyMailReceipt -v`
Expected: FAIL — `applyMailReceipt` undefined.

- [ ] **Step 3: Implement `applyMailReceipt`**

Add to `internal/usercommands/mail.go` (and add `"github.com/GoMudEngine/GoMud/internal/events"` to its imports):

```go
// applyMailReceipt credits an unread message's gold (to the reader's bank) and
// stores its attached item (to the backpack). Returns false WITHOUT any mutation
// when an attached item won't fit, so the message can stay unread and nothing is
// lost or partially credited. Already-read messages are a no-op (return true).
func applyMailReceipt(user *users.UserRecord, msg *users.Message) bool {
	if msg.Read {
		return true
	}
	// Try the item first: on failure, defer the whole message (no gold credit).
	if msg.Item != nil {
		if !user.Character.StoreItem(*msg.Item) {
			return false
		}
	}
	if msg.Gold > 0 {
		user.Character.Bank += msg.Gold
		events.AddToQueue(events.EquipmentChange{
			UserId:     user.UserId,
			BankChange: msg.Gold,
		})
	}
	return true
}
```

- [ ] **Step 4: Wire it into `inbox.go`**

In `internal/usercommands/inbox.go`, replace the read-side block:

```go
		if !msg.Read {
			if msg.Gold > 0 {
				user.Character.Bank += msg.Gold

				events.AddToQueue(events.EquipmentChange{
					UserId:     user.UserId,
					BankChange: msg.Gold,
				})

			}
			if msg.Item != nil {
				user.Character.StoreItem(*msg.Item)
			}
		}

		user.Inbox[idx].Read = true
```

with:

```go
		if !msg.Read {
			if !applyMailReceipt(user, &user.Inbox[idx]) {
				// Attached item won't fit — keep the message unread so nothing is
				// lost; the reader frees space and checks mail again. (The message
				// body + border were already printed above this block.)
				user.SendText(messaging.CategorySystem,
					fmt.Sprintf(`Your pack is too full to receive the <ansi fg="item">%s</ansi> — free some space and check your mail again.`, msg.Item.DisplayName()))
				continue
			}
		}

		user.Inbox[idx].Read = true
```

(The `else` keeps already-read messages marked read on an `inbox old` view. `fmt` and `messaging` are already imported in inbox.go.)

- [ ] **Step 5: Run tests + build**

Run: `go test ./internal/usercommands/ -run TestApplyMailReceipt -v` → PASS
Run: `go build ./internal/usercommands/` → success
Run: `go test ./internal/usercommands/` → PASS (existing inbox behavior intact)

- [ ] **Step 6: Commit**

```bash
git add internal/usercommands/mail.go internal/usercommands/inbox.go internal/usercommands/mail_test.go
git commit -m "fix(inbox): defer mail receipt when pack is full instead of losing the item"
```

---

## Task 4: The `mail` command + delivery + registration

**Files:**
- Modify: `internal/usercommands/mail.go` (add `Mail` command + `deliverMail`)
- Modify: `internal/usercommands/usercommands.go` (register)

- [ ] **Step 1: Add the command + delivery helper**

Append to `internal/usercommands/mail.go`. Update its import block to:

```go
import (
	"fmt"
	"strconv"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)
```

Command + delivery:

```go
// Mail sends a message + on-hand gold + an optional backpack item to one named
// recipient (online or offline). Gold leaves the sender's purse and the item
// leaves their backpack at send; the inbox read path credits gold->bank and
// item->backpack on receipt.
func Mail(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Cooldown gate (anti-spam). Only bites right after a successful send.
	cooldown := uint64(configs.GetBalanceConfig().MailSendCooldownRounds)
	if mailOnCooldown(user.Character.LastMailSentRound, util.GetRoundCount(), cooldown) {
		user.SendText(messaging.CategorySystem, `You must wait a while before sending more mail.`)
		return true, nil
	}

	cmdPrompt, _ := user.StartPrompt(`mail`, rest)

	// Recipient is the initial argument.
	recipientName := rest
	if recipientName == `` {
		question := cmdPrompt.Ask(`Send mail to whom?`, []string{})
		if !question.Done {
			return true, nil
		}
		recipientName = question.Response
	}

	rec, offlineUsername, ok := resolveMailRecipient(
		recipientName, user.UserId,
		users.GetByCharacterName,
		users.CharacterNameSearch,
	)
	if !ok {
		if recipientName != `` && (users.GetByCharacterName(recipientName) != nil) {
			user.SendText(messaging.CategorySystem, `You can't mail yourself.`)
		} else {
			user.SendText(messaging.CategorySystem, `No adventurer by that name has ever been recorded.`)
		}
		user.ClearPrompt()
		return true, nil
	}

	msg := users.Message{
		FromName: user.Character.Name,
		DateSent: time.Now(),
	}

	// Message body.
	question := cmdPrompt.Ask(`Message?`, []string{})
	if !question.Done {
		return true, nil
	}
	if question.Response == `` {
		user.SendText(messaging.CategorySystem, `A message is required.`)
		user.ClearPrompt()
		return true, nil
	}
	msg.Message = question.Response

	// Gold (from on-hand purse).
	question = cmdPrompt.Ask(`Attach how much gold?`, []string{})
	if !question.Done {
		return true, nil
	}
	gold, _ := strconv.Atoi(question.Response)
	if gold < 0 {
		gold = 0
	}
	if gold > user.Character.Gold {
		user.SendText(messaging.CategorySystem, `You aren't carrying that much gold.`)
		question.RejectResponse()
		return true, nil
	}
	msg.Gold = gold

	// Optional item.
	question = cmdPrompt.Ask(`Item name (or "none") to attach from your backpack?`, []string{})
	if !question.Done {
		return true, nil
	}
	var attached *items.Item
	if question.Response != `none` && question.Response != `` {
		if found, ok := user.Character.FindInBackpack(question.Response); ok {
			itemCopy := found
			attached = &itemCopy
			msg.Item = attached
		} else {
			user.SendText(messaging.CategorySystem, `Could not find item: `+question.Response)
			question.RejectResponse()
			return true, nil
		}
	}

	// Confirm.
	question = cmdPrompt.Ask(fmt.Sprintf(`Send this mail to <ansi fg="username">%s</ansi>?`, recipientName), []string{`Yes`, `No`}, `No`)
	if !question.Done {
		tplTxt, _ := templates.Process("mail/message", msg, user.UserId)
		user.SendText(messaging.CategorySystem, tplTxt)
		return true, nil
	}
	user.ClearPrompt()
	if len(question.Response) == 0 || question.Response[0:1] != `Y` {
		user.SendText(messaging.CategorySystem, `Cancelling the mail.`)
		return true, nil
	}

	// Commit: debit the sender, then deliver.
	if gold > 0 {
		user.Character.Gold -= gold
		events.AddToQueue(events.EquipmentChange{UserId: user.UserId, GoldChange: -gold})
	}
	if attached != nil {
		user.Character.RemoveItem(*attached)
	}

	deliverMail(rec, offlineUsername, msg)
	user.Character.LastMailSentRound = util.GetRoundCount()

	user.SendText(messaging.CategorySystem, fmt.Sprintf(`Your mail to <ansi fg="username">%s</ansi> is on its way.`, recipientName))
	return true, nil
}

// deliverMail drops the message into the recipient's inbox. Online: notify.
// Offline: load the detached disk record, add, and save (does not activate them).
func deliverMail(rec mailRecipient, offlineUsername string, msg users.Message) {
	if rec.online {
		if u := users.GetByUserId(rec.userId); u != nil {
			u.Inbox.Add(msg)
			u.Command(`inbox check`)
		}
		return
	}
	if ou, err := users.LoadUser(offlineUsername); err == nil && ou != nil {
		ou.Inbox.Add(msg)
		users.SaveUser(*ou)
	}
}
```

Add `"github.com/GoMudEngine/GoMud/internal/items"` to the mail.go import block (used by `attached *items.Item`).

- [ ] **Step 2: Register the command**

In `internal/usercommands/usercommands.go`, add to the `userCommands` map next to `inbox`:

```go
		`mail`:            {Mail, false, false, false},
```

(Not allowed when downed or in combat — mail is a peaceful, prompt-driven activity.)

- [ ] **Step 3: Build + full package test**

Run: `go build ./...`
Expected: success.

Run: `go test ./internal/usercommands/`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/usercommands/mail.go internal/usercommands/usercommands.go
git commit -m "feat(mudmail): player 'mail' command — send message+gold+item to a recipient"
```

---

## Task 5: Boot smoke test + docs

**Files:**
- Modify: `PATCH_NOTES.md`, `docs/PATH_TO_1.0.md`

- [ ] **Step 1: Build the whole project**

Run: `go build ./...` → success.

- [ ] **Step 2: Boot smoke test (per pre-push SOP)**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```

Run `go run .`, confirm `Server Ready` with no panic, then stop. (Additive config + command; no data files.)

- [ ] **Step 3: Patch note**

Prepend to `PATCH_NOTES.md`:

```markdown
## 2026-07-15 — Send mail to another adventurer

You can now write to another player by name with the new `mail` command — attach a note, some
coin from your purse, and even an item from your pack, and it'll be waiting in their mailbox
the next time they read their mail (the coin lands safely in their bank, the item in their
pack). It reaches them whether they're online or away. There's a short wait between sendings to
keep the mailboxes civil, and if someone's pack is too full for a parcel, it simply waits in
the mail rather than being lost.
```

- [ ] **Step 4: Mark mudmail done in the roadmap**

In `docs/PATH_TO_1.0.md`, change the `🟡 Player-to-player mudmail` line to `✅` with a one-line
completion note dated 2026-07-15 referencing this plan + spec, noting **this is the last
substantive econ-arc item — the marketplace arc is complete.**

- [ ] **Step 5: Commit**

```bash
git add PATCH_NOTES.md docs/PATH_TO_1.0.md
git commit -m "docs(mudmail): patch notes + roadmap for player-to-player mail"
```

---

## Notes for the implementer

- **No self-mail message nuance:** `resolveMailRecipient` returns `ok=false` for both "not
  found" and "self". The command re-checks `GetByCharacterName(name) != nil` to pick the
  "you can't mail yourself" vs "no such adventurer" message. (An offline self-mail is
  effectively impossible — you're online to run the command — so the online re-check suffices.)
- **Prompt re-entry:** `StartPrompt`/`Ask` re-enter the command per response; the cooldown
  check at the top stays clear during composition because `LastMailSentRound` is only stamped
  after a successful send.
- **Gold event sign is cosmetic:** the GMCP `EquipmentChange` handler only checks non-zero to
  trigger a `Char.Worth` refresh; `-gold` on debit / `+gold` on bank credit both just refresh.
- **Offline delivery must not activate the recipient:** `users.LoadUser` returns a detached
  disk record; `SaveUser` persists it. Do not add them to the active user manager.
- **`FindInBackpack` returns a copy:** take its address into a local (`itemCopy := found`)
  before storing the pointer in `msg.Item`, then `RemoveItem(found)` the original.
```
