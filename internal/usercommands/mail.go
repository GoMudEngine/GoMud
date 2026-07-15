package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
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
