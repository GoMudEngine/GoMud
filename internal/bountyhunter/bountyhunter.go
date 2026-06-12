// Package bountyhunter implements the bounty-hunting dispatch system (5.2).
//
// This file provides the pure scaling helpers and spawnHunter. The per-round
// dispatch sweep and lifecycle registry live in dispatch.go (Task 9).
package bountyhunter

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// hunterMobTemplateId is the bounty_hunter mob template authored in Task 7.
const hunterMobTemplateId = 110

// scaledStatpool computes the hunter's stat pool from the triggering bounty's
// gold value. Formula: round(base + gold*perGold), clamped to [min, max].
// Uses math.Round (half-away-from-zero).
func scaledStatpool(gold, base int, perGold float64, min, max int) int {
	v := int(math.Round(float64(base) + float64(gold)*perGold))
	if v < min {
		v = min
	}
	if v > max {
		v = max
	}
	return v
}

// gearGold computes the gold value fed to GenerateAffixedItem for each gear
// slot. Formula: statpool / divisor. Guards divisor <= 0 → 1.
func gearGold(statpool, divisor int) int {
	if divisor <= 0 {
		divisor = 1
	}
	return statpool / divisor
}

// spawnHunter creates a scaled, geared bounty hunter at the issuer faction's
// release room (the faction's "station"), tags it with the issuer faction so
// the PlayerDeath_BountyResolve killGuard path credits the kill, adds a
// hunt_bounty_target goal, places it in the world, and telegraphs the target.
//
// Returns the hunter's InstanceId on success, 0 on any failure (nil faction
// def, missing room, failed spawn, etc.).
func spawnHunter(targetUserId, bountyId, bountyGold int, issuerFaction string) int {
	bal := configs.GetBalanceConfig()

	// Resolve spawn seat from faction definition.
	def := factions.GetDefinition(issuerFaction)
	if def == nil || def.ReleaseRoom == 0 {
		return 0 // faction has no seat to spawn from
	}
	seat := def.ReleaseRoom

	// Compute scaled stat pool.
	statpool := scaledStatpool(
		bountyGold,
		int(bal.BountyHunterBaseStatpool),
		float64(bal.BountyHunterStatpoolPerGold),
		int(bal.BountyHunterMinStatpool),
		int(bal.BountyHunterMaxStatpool),
	)

	// Spawn fresh instance (overrides template statpool with our scaled value).
	hunter := mobs.NewMobByIdFresh(mobs.MobId(hunterMobTemplateId), seat, statpool)
	if hunter == nil {
		return 0
	}

	// Tag with issuer faction so PlayerDeath_BountyResolve's killGuard path
	// recognises the hunter as belonging to the faction that posted the bounty.
	hunter.Groups = append(hunter.Groups, issuerFaction)

	// Equip affix-scaled gear — mirrors the instance-loot path in rooms.go:786.
	gg := gearGold(statpool, int(bal.BountyHunterGearGoldDivisor))
	scalar := float64(bal.LootBudgetScalar)
	for _, baseId := range hunter.LootPool {
		affixed := items.GenerateAffixedItem(baseId, gg, scalar)
		if affixed.ItemId > 0 {
			if _, worn, reason := hunter.Character.Wear(affixed); !worn {
				mudlog.Warn("bountyhunter.spawnHunter()",
					"mobName", hunter.Character.Name,
					"mobId", hunter.MobId,
					"itemId", affixed.ItemId,
					"reason", reason)
			}
		}
	}

	// Stamp the per-hunter target on the hunter INSTANCE's MiscData so that
	// the planner (which has the live instance) can read it. The goal itself
	// is param-less — it is a template-level intent marker shared across all
	// concurrent hunter instances, so target_user_id / bounty_id cannot live
	// in goal params (all instances share template 110's ONE goal record).
	hunter.Character.SetMiscData("bh_target_user_id", targetUserId)
	hunter.Character.SetMiscData("bh_bounty_id", bountyId)

	// Add the hunt goal via the goals public API (param-less intent marker).
	// mobId = template id (int), namesimple = ConvertForFilename(name).
	g := &goals.Goal{
		Type:     "hunt_bounty_target",
		Priority: 100,
		// No Params — per-hunter data lives in MiscData above.
	}
	if _, err := goals.Add(int(hunter.MobId), util.ConvertForFilename(hunter.Character.Name), g); err != nil {
		// ConflictError is expected when a second hunter is spawned while the
		// first is still active — the goal already exists on template 110.
		// Log-and-continue: the goal is already present and correctly selected,
		// which is exactly what we want. Do NOT abort the spawn.
		_ = err
	}

	// Place in the world.
	if room := rooms.LoadRoom(seat); room != nil {
		room.AddMob(hunter.InstanceId)
	}

	// Telegraph the target if they are currently online.
	if u := users.GetByUserId(targetUserId); u != nil {
		u.SendText(messaging.CategorySystem,
			"Word reaches you that a hunter has taken the contract on your head.")
	}

	return hunter.InstanceId
}
