package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// chrysifier_homunculus.go — the Chrysifier apex companion. A "crafted" copy of
// the player: a boss-tier body whose statpool scales off how much the owner has
// crafted, inheriting their non-crafting skills + a preset physical-mutation
// loadout, reserving heavy Conviction (usually the owner's only companion).

const homunculusMobId = 9612

// homunculusCraftSkills are the skills whose summed levels drive the homunculus
// statpool (and which are NOT inherited by it — it's the bruiser, not the maker).
var homunculusCraftSkills = []string{
	"blacksmithing", "alchemy", "tailoring", "cooking",
	"jewelcrafting", "enchanting", "salvage",
}

// homunculusStatPool = round(craftSum * scale), floored so a novice still gets a
// real (if modest) body.
func homunculusStatPool(craftSum int, scale float64) int {
	if craftSum < 1 {
		craftSum = 1
	}
	return int(float64(craftSum)*scale + 0.5)
}

// homunculusCraftSum sums the owner's crafting-skill levels. Uses the raw
// Skills map (not GetSkillLevel, which floors unset skills to 1) so only actual
// investment drives the homunculus's power.
func homunculusCraftSum(c *characters.Character) int {
	sum := 0
	for _, s := range homunculusCraftSkills {
		sum += c.Skills[s]
	}
	return sum
}

// inheritedHomunculusSkills returns the owner's NON-crafting skills, so the
// homunculus fights the way its maker does (crafting skills are excluded — the
// homunculus doesn't craft, it fights).
func inheritedHomunculusSkills(c *characters.Character) map[string]int {
	craftSet := make(map[string]bool, len(homunculusCraftSkills))
	for _, s := range homunculusCraftSkills {
		craftSet[s] = true
	}
	out := make(map[string]int)
	for sk, lvl := range c.Skills {
		if lvl > 0 && !craftSet[sk] {
			out[sk] = lvl
		}
	}
	return out
}

// spawnHomunculus forges the owner's homunculus companion into `room` and
// registers it (with its snapshotted Conviction reservation). Returns the mob,
// or nil on failure.
func spawnHomunculus(user *users.UserRecord, room *rooms.Room) *mobs.Mob {
	ch := user.Character
	cfg := configs.GetBalanceConfig()

	pool := homunculusStatPool(homunculusCraftSum(ch), float64(cfg.HomunculusCraftScale))
	mob := mobs.NewMobByIdFresh(mobs.MobId(homunculusMobId), room.RoomId, pool)
	if mob == nil {
		return nil
	}
	room.AddMob(mob.InstanceId)

	// A copy of its maker: same name-root, same non-crafting skills.
	mob.Character.Name = ch.Name + "'s Homunculus"
	if mob.Character.Skills == nil {
		mob.Character.Skills = make(map[string]int)
	}
	for sk, lvl := range inheritedHomunculusSkills(ch) {
		mob.Character.Skills[sk] = lvl
	}
	// A preset physical-cluster (Colossus) loadout — the bruiser the crafter
	// themselves is not. First-pass set; tune in playtest.
	mob.Character.Mutations = map[string]int{"titan-growth": 2}
	mob.Character.RecalculateStats()

	// Bind as a permanent companion of the owner.
	mob.Character.Charm(user.UserId, 99999, "")
	mob.Character.EndAggro()
	ch.TrackCharmed(mob.InstanceId, true)

	reserve := ch.CalcCompanionReserve(int(cfg.HomunculusConvictionReserve))
	info := characters.CompanionInfo{
		MobId:             homunculusMobId,
		InstanceId:        mob.InstanceId,
		SourceType:        characters.CompanionSummoned,
		Name:              mob.Character.Name,
		BaseName:          mob.Character.Name,
		AutoAssist:        true,
		ConvictionReserve: reserve,
	}
	ch.AddCompanion(info)
	ch.RecalculateStats() // apply the new reservation immediately
	return mob
}
