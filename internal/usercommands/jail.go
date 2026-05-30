package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/justice"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// currentJailFine returns (current fine, rounds remaining, jail record, true)
// for a jailed player, or (..., false) if not jailed. The fine decays toward 0
// as the sentence is served: it equals decay-per-round × rounds-remaining.
func currentJailFine(user *users.UserRecord) (int, uint64, justice.JailRecord, bool) {
	rec, ok := justice.JailInfo(user.Character)
	if !ok {
		return 0, 0, justice.JailRecord{}, false
	}
	now := util.GetRoundCount()
	var remaining uint64
	if rec.UntilRound > now {
		remaining = rec.UntilRound - now
	}
	fine := int(remaining) * rec.DecayPerRound
	if fine < 0 {
		fine = 0
	}
	return fine, remaining, rec, true
}

// Fine shows a jailed player their current (decaying) fine and how to pay it.
func Fine(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	fine, _, _, ok := currentJailFine(user)
	if !ok {
		user.SendText(messaging.CategorySystem, "You aren't being held anywhere.")
		return true, nil
	}
	if fine <= 0 {
		user.SendText(messaging.CategorySystem, "Your sentence is all but served — you'll be released shortly.")
		return true, nil
	}
	user.SendText(messaging.CategorySystem, fmt.Sprintf(
		`Your fine to walk free now is <ansi fg="gold">%d gold</ansi>.`+
			` It drops the longer you sit. Pay it with <ansi fg="command">payfine</ansi>.`, fine))
	return true, nil
}

// PayFine lets a jailed player buy their freedom early (person gold first, then
// bank). On success the detention resolves immediately.
func PayFine(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	fine, _, _, ok := currentJailFine(user)
	if !ok {
		user.SendText(messaging.CategorySystem, "You aren't being held anywhere.")
		return true, nil
	}
	if fine <= 0 {
		// Effectively free already — just release.
		justice.ResolveDetention(user.Character, user.UserId)
		return true, nil
	}
	total := user.Character.Gold + user.Character.Bank
	if total < fine {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(
			`You can't cover the <ansi fg="gold">%d gold</ansi> fine, even counting your bank.`+
				` You'll have to sit it out.`, fine))
		return true, nil
	}
	// On-person gold first, bank as fallback.
	remaining := fine
	if user.Character.Gold >= remaining {
		user.Character.Gold -= remaining
	} else {
		remaining -= user.Character.Gold
		user.Character.Gold = 0
		user.Character.Bank -= remaining
	}
	// Send the payment line BEFORE ResolveDetention so it reads before the
	// buff's release flavor ("The cell door swings open..."). Don't mention the
	// door here — the Jailed buff's end_user_text owns that line (avoids the
	// double door message; 5.1c smoke BUG-01).
	user.SendText(messaging.CategorySystem, fmt.Sprintf(
		`You count out <ansi fg="gold">%d gold</ansi> and settle your fine with the guards.`, fine))
	justice.ResolveDetention(user.Character, user.UserId)
	return true, nil
}
