package mobcommands

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func LookForTrouble(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Already aggroed, skip.
	if mob.Character.IsInCombat() {
		return true, nil
	}

	// Non-combatants (merchants, quest NPCs) never pick fights — even if
	// their group has been flagged hostile by a player attacking a
	// combatant in the same faction.
	if mob.IsNonCombatant() {
		return true, nil
	}

	// Make a list of all players this gorup is hostile to in this room.
	isCharmed := mob.Character.IsCharmed()

	allPotentialTargets := []int{}
	nonDownedUserTargets := []int{}
	possibleMobTargets := []int{}

	//mudlog.Info("lookgfortrouble", "mobname", fmt.Sprintf(`%s#%d`, mob.Character.Name, mob.InstanceId))

	if !isCharmed {

		allPlayerIds := room.GetPlayers(rooms.FindAll)

		for _, playerId := range allPlayerIds {

			user := users.GetByUserId(playerId)
			if user == nil {
				continue
			}

			// Respawn grace: grace-protected players are invisible to mob
			// targeting entirely, not just un-settable via SetAggro. Without
			// this, mobs whose group is flagged hostile re-issue attack
			// commands every idle tick. The commands bounce at SetAggro but
			// the "prepares to fight" message is still sent each time, and
			// the mob starts fighting the instant grace expires.
			if user.Character.HasBuffFlag(buffs.NoAggroTarget) {
				continue
			}

			raceInfo := species.GetSpecies(user.Character.SpeciesId)
			if raceInfo == nil {
				mudlog.Error("SpeciesError", "Not Found", user.Character.SpeciesId)
				continue
			}

			// Once a player is downed mobs stop considering them a target
			// They don't see players that are sneaking...
			ignoreUser := false

			if user.Character.Health < 1 {
				ignoreUser = true
			} else if user.Character.IsHidden() {
				ignoreUser = true
			}

			entries := 1
			if party := parties.Get(playerId); party != nil {
				entries += party.ChanceToBeTargetted(playerId)
			}

			// Stage 12.1: Pheromone Glands — predatory mobs prefer this player
			if aggroMagnet := mutations.GetAggroMagnet(user.Character.Mutations); aggroMagnet > 0 {
				for _, g := range mob.Groups {
					if g == "predatory" {
						entries = int(math.Round(float64(entries) * aggroMagnet))
						if entries < 1 {
							entries = 1
						}
						break
					}
				}
			}

			// Faction-rep peace gate: skip players who are at TierWarm
			// or higher with at least one of the mob's defined factions.
			if factions.IsPeacefulToward(mob, user.UserId) {
				continue
			}

			if mob.AutoAggro { // Does it always attack players?

				allPotentialTargets = append(allPotentialTargets, playerId)

				if !ignoreUser {
					for i := 0; i < entries; i++ {
						nonDownedUserTargets = append(nonDownedUserTargets, playerId)
					}
				}
				continue
			}

			// Does this specific mob hate this player?
			if mob.HatesSpecies(raceInfo.Name) {

				allPotentialTargets = append(allPotentialTargets, playerId)

				if !ignoreUser {
					for i := 0; i < entries; i++ {
						nonDownedUserTargets = append(nonDownedUserTargets, playerId)
					}
				}
				continue
			}

		}

		// Still nothing, look for trouble in mobs they hate
		for _, considerMobInstanceId := range room.GetMobs() {
			if mob.InstanceId == considerMobInstanceId {
				continue
			}

			considerMob := mobs.GetInstance(considerMobInstanceId)
			if considerMob == nil {
				continue
			}

			raceInfo := species.GetSpecies(mob.Character.SpeciesId)

			if mob.HatesMob(considerMob) || mob.HatesSpecies(raceInfo.Name) {
				possibleMobTargets = append(possibleMobTargets, considerMobInstanceId)
				continue
			}

			if considerMob.Character.IsCharmed() {
				for _, uid := range allPotentialTargets { // Only consider charmed mobs if they are charmed by a player this mob wants to fight
					if considerMob.Character.IsCharmed(uid) {
						possibleMobTargets = append(possibleMobTargets, considerMobInstanceId)
						break
					}
				}
			}

		}
	}

	targetUserId := 0
	targetMobInstanceId := 0

	userCt := len(nonDownedUserTargets)
	mobCt := len(possibleMobTargets)

	// Prefer player targets over companions/mobs. Only fall back to mob
	// targets when no eligible players are available.
	if userCt > 0 {
		// Chunk 4e §6: tiebreaker bias toward grappled-controlled targets.
		// When a candidate who is being controlled in a grapple (i.e., pinned)
		// has a pool weight within 10% of the top-weight candidate, boost them
		// to match the top weight. Does NOT override a clear primary preference.
		nonDownedUserTargets = applyGrappledControlledTiebreak(nonDownedUserTargets)
		targetUserId = nonDownedUserTargets[util.Rand(len(nonDownedUserTargets))]
	} else if mobCt > 0 {
		targetMobInstanceId = possibleMobTargets[util.Rand(mobCt)]
	}

	if targetUserId > 0 {
		// Chunk 5 (Presence): stamp LastTargetFoundRound so PresenceTick
		// can reset the "bored" timer when a target is found.
		mob.Character.LastTargetFoundRound = util.GetRoundCount()
		mob.Command(fmt.Sprintf("attack @%d", targetUserId)) // @ denotes a specific player id
	} else if targetMobInstanceId > 0 {
		mob.Character.LastTargetFoundRound = util.GetRoundCount()
		mob.Command(fmt.Sprintf("attack #%d", targetMobInstanceId)) // # denotes a specific mob id
	}

	return true, nil
}

// applyGrappledControlledTiebreak boosts grappled-controlled candidates in
// the pool when they are within 10% of the top-weight candidate. Returns a
// new pool slice (never modifies the input). This is a TIEBREAKER ONLY: if a
// grappled candidate's pool weight is already below the 10% tie band, the
// original pool is returned unchanged.
//
// "Grappled-controlled" means the player is in a grapple AND is on the
// controlled/losing side (i.e., being mounted or pinned by someone else).
// A mob targeting such a player models predatory opportunism: the pinned
// target is already vulnerable and harder to rescue.
//
// Chunk 4e §6.
func applyGrappledControlledTiebreak(pool []int) []int {
	if len(pool) < 2 {
		return pool
	}

	// Count pool-entry occurrences per user — that's their priority weight.
	weights := make(map[int]int)
	for _, uid := range pool {
		weights[uid]++
	}

	// Find the maximum weight among all candidates.
	maxWeight := 0
	for _, w := range weights {
		if w > maxWeight {
			maxWeight = w
		}
	}

	// Determine the tie band floor (top weight minus 10%).
	// Use integer arithmetic: tieBandFloor = maxWeight - maxWeight/10.
	// A candidate is "in the tie band" if their weight >= tieBandFloor.
	tieBandFloor := maxWeight - maxWeight/10
	if tieBandFloor < 1 {
		tieBandFloor = 1
	}

	// Find any candidate in the tie band who is grappled+controlled.
	// Only boost if at least one such candidate exists AND is not already
	// at maxWeight (otherwise no change needed).
	needsBoost := false
	for uid, w := range weights {
		if w < tieBandFloor {
			continue // outside tie band
		}
		if w >= maxWeight {
			continue // already at top — no boost needed
		}
		user := users.GetByUserId(uid)
		if user == nil || user.Character == nil {
			continue
		}
		if user.Character.IsGrappling() && user.Character.IsBeingControlled() {
			needsBoost = true
			break
		}
	}

	if !needsBoost {
		return pool
	}

	// Rebuild the pool: boost tie-band grappled-controlled candidates up to
	// maxWeight by appending extra entries for them.
	out := make([]int, len(pool))
	copy(out, pool)

	for uid, w := range weights {
		if w < tieBandFloor || w >= maxWeight {
			continue
		}
		user := users.GetByUserId(uid)
		if user == nil || user.Character == nil {
			continue
		}
		if user.Character.IsGrappling() && user.Character.IsBeingControlled() {
			deficit := maxWeight - w
			for i := 0; i < deficit; i++ {
				out = append(out, uid)
			}
		}
	}

	return out
}
