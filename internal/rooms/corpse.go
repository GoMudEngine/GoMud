package rooms

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/gametime"
)

type Corpse struct {
	UserId       int
	MobId        int
	Character    characters.Character
	RoundCreated uint64
	Prunable     bool // Whether it can be removed
	WasCharmed   bool // True if the mob was a charmed companion when it died

	// Stage 3.4: optional overrides for special-mob corpses (wagons,
	// statues, etc.). Stamped from the dying mob's YAML overrides at
	// corpse creation time.
	CorpseName        string
	CorpseDescription string
}

// DisplayName returns the rendered corpse name. If CorpseName is set
// (Stage 3.4 special mobs), returns it directly. Otherwise returns
// the standard "<Name> corpse" form.
func (c Corpse) DisplayName() string {
	if c.CorpseName != "" {
		return c.CorpseName
	}
	return c.Character.Name + " corpse"
}

func (c *Corpse) Update(roundNow uint64, decayRate string) {

	if c.Prunable {
		return
	}

	if decayRate == `` {
		decayRate = `1 week`
	}

	gd := gametime.GetDate(c.RoundCreated)
	decayRound := gd.AddPeriod(decayRate)

	// Has enough time passed to do the respawn?
	if roundNow >= decayRound {
		c.Prunable = true
	}

}
