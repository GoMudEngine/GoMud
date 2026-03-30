package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// calcSneakScore computes the stealth score for a character, applying
// an illumination penalty if the character is emitting light.
func calcSneakScore(c *characters.Character) float64 {
	score := float64(c.Stats.Dexterity.ValueAdj) +
		combat.SkillMultiplier(c.GetSkillLevel(skills.Skullduggery))*25.0

	// Emitting light makes it much harder to hide
	if c.HasBuffFlag(buffs.EmitsLight) {
		score *= 0.5
	}

	return score
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
		observerScore := float64(p.Character.Stats.Perception.ValueAdj) +
			combat.SkillMultiplier(p.Character.GetSkillLevel(skills.Search))*25.0
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
		observerScore := float64(mob.Character.Stats.Perception.ValueAdj) +
			combat.SkillMultiplier(mob.Character.GetSkillLevel(skills.Search))*25.0
		success, _, _, _ := dice.OpposedRollStat(sneakScore, observerScore)
		if !success {
			return true, mob.Character.Name
		}
	}

	return false, ""
}
