package ferry

import (
	"fmt"
	"unicode"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// BoardResult reports what Board did (for tests/telemetry; player
// messaging is handled inside).
type BoardResult int

const (
	BoardOk BoardResult = iota
	BoardNotAtPort
	BoardNotDocked
	BoardNoGold
	BoardError
)

// Board runs the full paid-boarding flow: the asking user, the agent mob
// (speaks refusals/quotes), the room the ask happened in (must be one of
// the route's docks), and the resolved route. Mirrors the Threshold-Keeper
// gold-charge pattern (actOpenInstancePortal).
func Board(user *users.UserRecord, mob *mobs.Mob, roomId int, routeId string) BoardResult {
	r, ok := RouteFor(routeId)
	if !ok {
		return BoardNotAtPort
	}
	portIdx := -1
	for i, p := range r.Ports {
		if p.DockRoom == roomId {
			portIdx = i
		}
	}
	if portIdx < 0 {
		return BoardNotAtPort
	}

	if !bool(configs.GetGamePlayConfig().FerriesEnabled) {
		mob.Command(`say No sailings today. The line is suspended.`)
		// BoardNotDocked is deliberately overloaded for the disabled case —
		// keeps the result enum small.
		return BoardNotDocked
	}

	rpd := int(configs.GetTimingConfig().RoundsPerDay)
	now := util.GetRoundCount()
	s := StateAt(r, now, rpd)

	if !s.Docked || s.PortIdx != portIdx {
		arriveRound := NextDockedRound(r, portIdx, now, rpd)
		gd := gametime.GetDate(arriveRound)
		mob.Command(`say ` + formatNotDockedQuote(r.Name, gd.Hour, gd.AmPm))
		return BoardNotDocked
	}

	if user.Character.Gold < r.Fare {
		mob.Command(fmt.Sprintf(`say Passage on %s is %d gold. Come back when you have it.`, r.Name, r.Fare))
		return BoardNoGold
	}

	user.Character.Gold -= r.Fare

	dockRoom := rooms.LoadRoom(roomId)

	if err := rooms.MoveToRoom(user.UserId, r.DeckRoom); err != nil {
		user.Character.Gold += r.Fare // refund
		mob.Command(`say Trouble at the gangplank. Your coin is returned.`)
		return BoardError
	}

	if dockRoom != nil {
		dockRoom.SendTextVisual(messaging.CategoryRoomExit,
			fmt.Sprintf(`<ansi fg="username">%s</ansi> pays the fare and crosses the gangplank aboard %s.`, user.Character.Name, r.Name),
			user.UserId)
	}
	user.SendText(messaging.CategoryRoomDescription,
		fmt.Sprintf(`You pay %d gold and cross the gangplank aboard %s.`, r.Fare, r.Name))

	// Show the destination room. MoveToRoom does not auto-describe (same
	// engine trap actMovePlayer documents) — queue a look so arrival on
	// deck is never silent.
	user.Command("look")

	return BoardOk
}

// formatNotDockedQuote is pure so it can be unit tested without the
// gametime/config globals.
func formatNotDockedQuote(vesselName string, hour int, amPm string) string {
	return fmt.Sprintf(`%s is out on the water. She ties up here again around %d %s.`, capitalizeFirst(vesselName), hour, amPm)
}

// capitalizeFirst upper-cases the first rune for sentence-initial use of
// lowercase-article vessel names ("the Lakewind Packet" → "The Lakewind Packet").
func capitalizeFirst(s string) string {
	if s == `` {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
