package ferry

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

const gangplankExitName = `gangplank`

// ambiance lines shown on deck while at sea, rotated deterministically.
var atSeaAmbiance = []string{
	`Water slides past the hull in a long, unhurried hiss.`,
	`The deck lifts and settles. Somewhere below, cargo shifts and goes quiet again.`,
	`A gull keeps pace off the rail for a while, decides you have no food, and banks away.`,
	`Spray flicks over the bow. The shoreline is a low smudge, too far to name.`,
}

// ambianceEveryNRounds paces the at-sea flavor (~every 92s at 4s rounds).
const ambianceEveryNRounds = 23

// Tick reconciles every vessel to its clock-derived state. Called once per
// round from the NewRound hook. Idempotent: transition emotes fire only on
// the round the phase changes; the gangplank ensure/remove runs every round.
func Tick() {
	if !bool(configs.GetGamePlayConfig().FerriesEnabled) {
		return
	}
	rpd := int(configs.GetTimingConfig().RoundsPerDay)
	now := util.GetRoundCount()

	for _, r := range AllRoutes() {
		cur := StateAt(r, now, rpd)
		if now > 0 {
			prev := StateAt(r, now-1, rpd)
			if prev.Docked && !cur.Docked {
				emitDeparture(r, prev.PortIdx)
			}
			if !prev.Docked && cur.Docked {
				emitArrival(r, cur.PortIdx)
			}
		}
		reconcileGangplank(r, cur)
		phase := now + uint64(r.PhaseOffsetRounds)
		if !cur.Docked && phase%ambianceEveryNRounds == 0 {
			emitAmbiance(r, phase)
		}
	}

	tickFactors(rpd, now)
}

// reconcileGangplank ensures the deck's gangplank temp exit matches the
// vessel state: present and pointing at the berth while docked, absent at
// sea. Self-heals if anything else pruned or altered it.
func reconcileGangplank(r Route, s VesselState) {
	deck := rooms.LoadRoom(r.DeckRoom)
	if deck == nil {
		return
	}
	if !s.Docked {
		if t, ok := deck.ExitsTemp[gangplankExitName]; ok {
			deck.RemoveTemporaryExit(t)
		}
		return
	}
	wantDock := r.Ports[s.PortIdx].DockRoom
	if t, ok := deck.ExitsTemp[gangplankExitName]; ok {
		if t.RoomId == wantDock {
			return // already correct
		}
		deck.RemoveTemporaryExit(t)
	}
	deck.AddTemporaryExit(gangplankExitName, exit.TemporaryRoomExit{
		RoomId:  wantDock,
		Title:   gangplankExitName,
		UserId:  0,       // anyone may disembark
		Expires: `1 day`, // effectively never; controller removes it explicitly
	})
}

func emitDeparture(r Route, fromPortIdx int) {
	if dock := rooms.LoadRoom(r.Ports[fromPortIdx].DockRoom); dock != nil {
		dock.SendText(messaging.CategoryRoomDescription,
			fmt.Sprintf(`Lines come off the bollards and %s stands out into open water.`, r.Name))
	}
	if deck := rooms.LoadRoom(r.DeckRoom); deck != nil {
		deck.SendText(messaging.CategoryRoomDescription,
			`The gangplank comes up, the lines go over, and the deck heels as she comes about. The shore begins to slide away.`)
	}
}

func emitArrival(r Route, atPortIdx int) {
	if dock := rooms.LoadRoom(r.Ports[atPortIdx].DockRoom); dock != nil {
		dock.SendText(messaging.CategoryRoomDescription,
			fmt.Sprintf(`%s noses up to the berth and makes fast. The gangplank rattles down.`, capitalizeFirst(r.Name)))
	}
	if deck := rooms.LoadRoom(r.DeckRoom); deck != nil {
		deck.SendText(messaging.CategoryRoomDescription,
			`Fenders squeal against timber as she comes alongside. The gangplank goes down — you can walk ashore.`)
	}
}

func emitAmbiance(r Route, phase uint64) {
	deck := rooms.LoadRoom(r.DeckRoom)
	if deck == nil {
		return
	}
	line := atSeaAmbiance[int(phase/ambianceEveryNRounds)%len(atSeaAmbiance)]
	deck.SendText(messaging.CategoryRoomDescription, line)
}
