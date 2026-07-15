# Player-to-player mudmail

**Date:** 2026-07-15
**Status:** Design (approved decisions; pending spec review)
**Roadmap:** `docs/PATH_TO_1.0.md` §1 — the last substantive econ-arc item.
**Depends on:** nothing new — reuses the existing `users.Message`/`Inbox` infra and the
`inbox` read command.

---

## 1. Problem / goal

The inbox and mail *receive* path already exist: `inbox.go` credits an attached message's
gold to the reader's **bank** and stores an attached item in their backpack when they read
unread mail. The only *sender* is the admin `mudmail`, which **mass-mails everyone** with
**conjured** gold and a **copied (non-consumed)** item — god-mode, not usable for players.

Goal: a player-facing **`mail`** command that sends a message + optional gold + optional item
to **one named recipient** (online or offline), paying real value out of the sender.

---

## 2. Approved design decisions

1. **New `mail` command**, player-usable, **interactive prompts** (message → gold → item →
   confirm), mirroring the admin `mudmail` flow. `inbox` stays the read command; admin
   `mudmail` (mass-send) is untouched.
2. **One named recipient**, resolved by **character name**, online *or* offline. No self-mail.
3. **Gold is paid from the sender's on-hand purse** (`Character.Gold`), not the bank. (The
   recipient still *receives* gold into their **bank** via the existing read path — a
   deliberate asymmetry: send from your pocket, it arrives secure.)
4. **Item is removed from the sender's backpack** (consumed), unlike the admin copy.
5. **Free** — no postage fee.
6. **Per-sender send cooldown** (minimal anti-spam): a character may send at most one mail
   every `MailSendCooldownRounds` (Balance config, default **10** rounds). Round-based, like
   the engine's other cooldowns; the last-sent round is a persisted `Character` field.
7. **Fix the receive-side item-loss bug**: `inbox.go` ignores `StoreItem`'s return, so an
   over-capacity recipient loses the attached item. Defer such a message (keep it unread)
   instead of destroying the item.

---

## 3. The `mail` command — `internal/usercommands/mail.go` (new)

Registered in `usercommands.go` as `` `mail`: {Mail, true, true, false} `` (non-admin, next
to `inbox`).

### Flow (interactive, via `user.StartPrompt`)

`mail <recipient>` — the recipient name is the command argument (required). Then prompts:

0. **Cooldown gate** (at command entry, before `StartPrompt`): if the sender sent mail within
   the last `MailSendCooldownRounds`, reject: "You must wait a while before sending more mail."
   The check reads `sender.Character.LastMailSentRound` vs `util.GetRoundCount()`. Because the
   last-sent round is only stamped on a *successful* send (§ commit), a user mid-compose (who
   hasn't sent yet) is never blocked on prompt re-entry — the cooldown only bites right after
   an actual send.
1. **Resolve recipient first** (before prompting for content — fail fast):
   - Online: `users.GetByCharacterName(name)` → live record.
   - Else offline: `users.CharacterNameSearch(name)` → `(userId, username)`; `userId == 0`
     means no such character → reject: "No adventurer by that name has ever been recorded."
   - Self-mail guard: recipient's `UserId == sender.UserId` → reject: "You can't mail
     yourself."
2. **Message?** — required, non-empty (reject empty; abort on blank like the admin flow).
3. **Attach how much gold?** — parsed int, `>= 0`. Must be `<= sender.Character.Gold`
   (on-hand). If more than they carry → reject the response with "You aren't carrying that
   much gold."
4. **Item name (or "none") to attach from your backpack?** — if not "none",
   `FindInBackpack`; not found → reject the response.
5. **Confirm** — `Send this <mail> to <recipient>?` [Yes/No], showing a preview
   (`mail/message` template). No → cancel, nothing charged.

### Commit (all-or-nothing — only after every validation passes)

```
// Deduct from the sender (on-hand gold + consume the item) BEFORE delivery.
sender.Character.Gold -= gold
emit EquipmentChange{UserId: sender, GoldChange: -gold}   // prompt/GMCP refresh
if item attached: remove it from sender.Character.Items (RemoveItem)

msg := users.Message{
    FromName: sender.Character.Name,   // real name, for accountability
    Message:  <text>,
    Gold:     gold,
    Item:     <&item or nil>,
    DateSent: time.Now(),
}

deliver(recipient, msg)   // §4
sender.Character.LastMailSentRound = util.GetRoundCount()  // start the cooldown
sender is told: "Your mail to <recipient> is on its way."
```

Because gold/item are removed from the sender at send time and delivered into the message,
the value is conserved (it lives in the recipient's inbox until they read it — exactly like
the auction unsold-return / seized-lot paths).

---

## 4. Delivery — `deliver(recipient, msg)`

- **Online recipient** (`GetByCharacterName` returned a live record): `u.Inbox.Add(msg)`;
  notify with `u.Command("inbox check")` (same as admin mudmail) so they see "You have new
  mail." No `SaveUser` needed (active users persist on the normal cycle; optionally SaveUser
  for durability).
- **Offline recipient** (found via `CharacterNameSearch`): load the detached disk record
  `ou, err := users.LoadUser(username)` (loads from disk, does **not** activate them),
  `ou.Inbox.Add(msg)`, `users.SaveUser(*ou)`. Mirrors the admin mass-mail offline path.

Resolve the recipient **once up front** (§3 step 1) and remember whether they were online, so
delivery uses the right path without re-searching.

---

## 5. Receive side — existing `inbox.go`, plus the item-loss fix

The read loop already does the right thing for gold (→ bank) and items (→ backpack). Replace
the unconditional item store + mark-read with a deferring version so a full pack never
destroys a mailed item:

```go
if !msg.Read {
    // An attached item that won't fit defers the WHOLE message (gold + item stay
    // pending) so nothing is lost — the reader frees space and checks mail again.
    if msg.Item != nil && !user.Character.StoreItem(*msg.Item) {
        user.SendText(messaging.CategorySystem,
            fmt.Sprintf(`Your pack is too full to receive the <ansi fg="item">%s</ansi> — free some space and check your mail again.`, msg.Item.DisplayName()))
        continue // leave unread; do NOT credit gold or mark read
    }
    if msg.Gold > 0 {
        user.Character.Bank += msg.Gold
        events.AddToQueue(events.EquipmentChange{UserId: user.UserId, BankChange: msg.Gold})
    }
    user.Inbox[idx].Read = true
}
```

Key ordering: attempt the item store first; on failure `continue` (no gold credit, message
stays unread) — atomic, no partial/double credit. On success the item is already stored, so
gold is credited and the message marked read together. (This also hardens admin mass-mail and
the auction/seizure inbox items against the same loss.)

---

## 6. Limits / edge cases (minimal for v1)

- Message required (non-empty).
- Gold `>= 0` and `<= sender's on-hand gold`.
- Item optional; must be in the sender's backpack.
- No self-mail; recipient must be a real character (online or offline).
- **Per-sender send cooldown** (`MailSendCooldownRounds`, default 10) is the only anti-spam
  guard. No postage fee, no per-mail gold cap in v1 (free was a deliberate choice — easy to
  add a `MailPostageFee` later if needed).
- Player-facing text stays no-hard-numbers where it's *effect* description; gold amounts in a
  mail preview / "not carrying that much" are fine (money is explicitly numeric, like the
  bank/auction notices).

---

## 7. Testing

- **Recipient resolution:** online name → live record; offline name → `CharacterNameSearch`
  hit; unknown name → rejected; self-name → rejected. (Unit-test a small pure
  `resolveMailRecipient` helper that returns `(userId, online bool, ok bool)` over injectable
  lookups, so the command handler stays thin.)
- **Cooldown:** a small pure helper `mailOnCooldown(lastSent, now, cooldownRounds) bool` —
  true when `now < lastSent + cooldown` (and `lastSent > 0`, `cooldown > 0`); false when the
  cooldown has elapsed or is disabled (0). The command gates on it and stamps
  `LastMailSentRound` only on a successful send.
- **Send debits sender:** on send, sender's on-hand gold drops by the attached amount and the
  attached item leaves their backpack; a zero-gold / no-item mail debits nothing.
- **Insufficient funds / missing item:** rejected, nothing mutated.
- **Delivery:** the recipient's inbox gains exactly one message with the right FromName /
  gold / item.
- **Item-loss fix (`inbox.go`):** reading a mail with an item while over carry capacity leaves
  the message **unread** and does **not** credit its gold (no partial state); reading again
  after freeing space delivers both. A gold-only mail always reads cleanly.

Full suite green + boot clean.

---

## 8. Out of scope / deferred

- Postage fee / gold sink, per-mail gold caps (v1 is free by choice; the send cooldown is the
  only guard).
- Making the recipient *receive* gold into their purse (kept as bank per the existing read
  path; the send-from-purse / receive-to-bank asymmetry is intentional).
- Block/ignore lists, mail deletion of specific messages (only `inbox clear` exists), COD /
  return-to-sender, multi-recipient player mail.

---

## 9. Files touched

- `internal/usercommands/mail.go` (new) — the `mail` command + `resolveMailRecipient` +
  `deliverMail` + `mailOnCooldown` helpers.
- `internal/usercommands/usercommands.go` — register `` `mail` ``.
- `internal/usercommands/inbox.go` — the item-loss-on-full-pack fix.
- `internal/characters/character.go` — new persisted `LastMailSentRound uint64` field.
- `internal/configs/config.balance.go` + a validator file — `MailSendCooldownRounds` (default 10).
- `internal/usercommands/mail_test.go` (new) — resolution + cooldown + delivery + item-loss tests.
- `PATCH_NOTES.md`, `docs/PATH_TO_1.0.md` (mark mudmail done) — at the end.
