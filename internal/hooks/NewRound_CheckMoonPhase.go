package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
)

//
// Watches the rounds go by.
// Fires a MoonPhase event when any of the three Witnesses (moons)
// crosses a new-moon or full-moon boundary.
//

type moonSpec struct {
	name   string
	period float64 // multiplier against RoundsPerDay
}

var witnesses = []moonSpec{
	{"Swiftmoon", 4.7},
	{"The Wanderer", 10.6},
	{"The Eye", 21.1},
}

func CheckMoonPhase(e events.Event) events.ListenerReturn {
	evt := e.(events.NewRound)

	roundNum := evt.RoundNumber
	if roundNum == 0 {
		return events.Continue
	}

	rpd := uint64(configs.GetTimingConfig().RoundsPerDay)

	for _, moon := range witnesses {
		cycleLen := uint64(moon.period * float64(rpd))
		if cycleLen == 0 {
			continue
		}

		prevPhase := (roundNum - 1) % cycleLen
		currPhase := roundNum % cycleLen

		// New moon: cycle just wrapped to zero
		if currPhase == 0 {
			events.AddToQueue(events.MoonPhase{
				MoonName:  moon.name,
				PhaseName: "new",
				IsNew:     true,
				IsFull:    false,
			})
			continue
		}

		// Full moon: crossed the halfway point
		halfCycle := cycleLen / 2
		if prevPhase < halfCycle && currPhase >= halfCycle {
			events.AddToQueue(events.MoonPhase{
				MoonName:  moon.name,
				PhaseName: "full",
				IsFull:    true,
				IsNew:     false,
			})
		}
	}

	return events.Continue
}
