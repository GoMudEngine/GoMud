package mobs

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
)

// Pack scaling tracks how long groups of mobs survive together and awards
// small Training bonuses when they hit consecutive-round thresholds.

var (
	packSurvivalCounters = map[string]int{} // groupTag → consecutive rounds survived together
	packLastMembers      = map[string]int{} // groupTag → member count last tick
)

// PackBonus describes a single pack bonus award for external handling.
type PackBonus struct {
	GroupTag    string
	MemberIds  []int  // Instance IDs of mobs that received bonuses
	ReachedMax bool   // True if any member hit PackMaxBonus
}

// TickPackSurvival checks all mob groups for pack survival bonuses.
// Returns a list of pack bonuses awarded this tick (for the caller to
// emit world events and room messages without creating import cycles).
func TickPackSurvival() []PackBonus {
	b := configs.GetBalanceConfig()
	if !bool(b.PackScalingEnabled) {
		return nil
	}

	// Build map of groupTag → list of live mob instance IDs
	groupMembers := map[string][]int{}
	for _, instId := range GetAllMobInstanceIds() {
		mob := GetInstance(instId)
		if mob == nil {
			continue
		}
		for _, grp := range mob.Groups {
			groupMembers[grp] = append(groupMembers[grp], instId)
		}
	}

	var bonuses []PackBonus

	// Process each group
	for tag, members := range groupMembers {
		if len(members) < 2 {
			delete(packSurvivalCounters, tag)
			delete(packLastMembers, tag)
			continue
		}

		lastCount, existed := packLastMembers[tag]
		packLastMembers[tag] = len(members)

		if existed && len(members) < lastCount {
			// A member died — reset counter
			packSurvivalCounters[tag] = 0
			continue
		}

		packSurvivalCounters[tag]++

		if packSurvivalCounters[tag] >= int(b.PackSurvivalRounds) {
			packSurvivalCounters[tag] = 0

			bonus := PackBonus{
				GroupTag: tag,
			}

			for _, instId := range members {
				mob := GetInstance(instId)
				if mob == nil {
					continue
				}
				if mob.PackBonusTotal >= int(b.PackMaxBonus) {
					continue
				}

				statName := packStatForArchetype(mob.Archetype)
				mob.Character.IncreaseStat(statName, int(b.PackBonusTrainingPts))
				mob.PackBonusTotal += int(b.PackBonusTrainingPts)
				bonus.MemberIds = append(bonus.MemberIds, instId)

				if mob.PackBonusTotal >= int(b.PackMaxBonus) {
					bonus.ReachedMax = true
				}
			}

			if len(bonus.MemberIds) > 0 {
				bonuses = append(bonuses, bonus)
			}
		}
	}

	// Clean up stale entries for groups that no longer exist
	for tag := range packSurvivalCounters {
		if _, exists := groupMembers[tag]; !exists {
			delete(packSurvivalCounters, tag)
			delete(packLastMembers, tag)
		}
	}

	return bonuses
}

// packStatForArchetype returns the stat name to boost based on mob archetype.
func packStatForArchetype(archetype string) string {
	switch archetype {
	case "fighting":
		return "strength"
	case "casting":
		return "willpower"
	default:
		return "vitality"
	}
}
