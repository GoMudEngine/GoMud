package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// calcSneakScore is a package-level shim that delegates to the shared
// actions.CalcSneakScore helper. All callers within usercommands use this
// wrapper so they don't need to reference the actions package directly.
func calcSneakScore(c *characters.Character) float64 {
	return actions.CalcSneakScore(c)
}

// checkStealthDetection rolls each observer in the room against the hidden
// user. Returns (spotted, spotterName). Party members of the hidden user are
// excluded from the check.
func checkStealthDetection(
	hiddenUser *users.UserRecord,
	room *rooms.Room,
) (bool, string) {
	sneakScore := calcSneakScore(hiddenUser.Character)

	// Build party exclusion set
	partyIds := make(map[int]bool)
	if p := parties.Get(hiddenUser.UserId); p != nil {
		for _, uid := range p.GetMembers() {
			partyIds[uid] = true
		}
	}

	// Check players
	for _, pId := range room.GetPlayers() {
		if pId == hiddenUser.UserId || partyIds[pId] {
			continue
		}
		p := users.GetByUserId(pId)
		if p == nil {
			continue
		}
		observerScore := actions.CalcSearchScore(p.Character)
		success, _, _, _ := dice.OpposedRollStat(sneakScore, observerScore)
		if !success {
			p.SendText(fmt.Sprintf(
				`<ansi fg="username">%s</ansi> slips into the room but you notice them.`,
				hiddenUser.Character.Name))
			return true, p.Character.Name
		}
	}

	// Check mobs
	for _, mId := range room.GetMobs() {
		mob := mobs.GetInstance(mId)
		if mob == nil {
			continue
		}
		observerScore := actions.CalcSearchScore(&mob.Character)
		success, _, _, _ := dice.OpposedRollStat(sneakScore, observerScore)
		if !success {
			return true, mob.Character.Name
		}
	}

	return false, ""
}
