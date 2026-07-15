package usercommands

import (
	"fmt"
	"strconv"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

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
// the command wires users.GetByCharacterName + users.CharacterNameSearch. On an
// offline hit it returns the account username for the loader.
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

	// Recipient is the initial argument (or prompted for if omitted).
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
		if recipientName != `` && users.GetByCharacterName(recipientName) != nil {
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
		if found, foundOk := user.Character.FindInBackpack(question.Response); foundOk {
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
