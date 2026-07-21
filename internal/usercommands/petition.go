package usercommands

import (
	"fmt"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/moderation"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

var (
	petitionCooldownMu sync.Mutex
	lastPetitionRound  = map[int]uint64{} // userId -> round of last petition
)

// Petition lets a player contact staff about anything (harassment, grief,
// stuck, request). Free-text; reporter, time, and room are captured
// automatically. Filed petitions land in a durable admin-reviewable queue and
// ping any online staff.
func Petition(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	rest = strings.TrimSpace(rest)
	if rest == "" {
		user.SendText(messaging.CategorySystem, `<ansi fg="yellow">Usage: <ansi fg="cyan-bold">petition <message></ansi> — send a message to the staff (harassment, a stuck quest, anything).</ansi>`)
		return true, nil
	}

	gp := configs.GetGamePlayConfig()

	if maxLen := int(gp.PetitionMaxLen); maxLen > 0 && len(rest) > maxLen {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="yellow">That's a bit long — please keep your petition under %d characters.</ansi>`, maxLen))
		return true, nil
	}

	// Anti-spam cooldown (checked here, recorded only after a successful file
	// so a failed Add doesn't lock the player out of retrying).
	cd := uint64(gp.PetitionCooldownRounds)
	nowRound := util.GetRoundCount()
	petitionCooldownMu.Lock()
	last, seen := lastPetitionRound[user.UserId]
	onCooldown := seen && cd > 0 && nowRound-last < cd
	petitionCooldownMu.Unlock()
	if onCooldown {
		user.SendText(messaging.CategorySystem, `<ansi fg="yellow">You've just sent a petition — give the staff a moment before sending another.</ansi>`)
		return true, nil
	}

	p, err := moderation.Add(user.Username, user.Character.RoomId, user.Character.Zone, rest)
	if err != nil {
		user.SendText(messaging.CategorySystem, `<ansi fg="red">Your petition could not be filed. Please try again or find an admin.</ansi>`)
		return true, nil
	}

	petitionCooldownMu.Lock()
	lastPetitionRound[user.UserId] = nowRound
	petitionCooldownMu.Unlock()

	user.SendText(messaging.CategorySystem, `<ansi fg="green">Your petition has been sent to the staff. They'll review it as soon as they can.</ansi>`)

	// Notify online staff.
	alert := fmt.Sprintf(`<ansi fg="alert-5">[PETITION #%d]</ansi> <ansi fg="username">%s</ansi> (%s): %s <ansi fg="black-bold">— type 'petitions' to review.</ansi>`,
		p.Id, user.Username, room.Title, rest)
	for _, staff := range users.GetAllActiveUsers() {
		if staff.Role != users.RoleUser {
			staff.SendText(messaging.CategorySystem, alert)
		}
	}

	return true, nil
}
