package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
)

// Salvage is the mob-side corpse-salvage handler. It is a single-tick direct
// action — no CraftingState machinery (that is player-side UX scaffolding).
//
// When called with rest == "corpse" the handler finds the first eligible corpse
// in the mob's current room, rolls a skill-based yield, stores recovered
// materials in the mob's inventory, removes the corpse, and emits a flavor
// message. Returning handled=true in all cases prevents the engine's
// "looks a little confused" fallback from firing.
func Salvage(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Find first non-prunable mob corpse with salvage returns.
	var target rooms.Corpse
	found := false
	for _, c := range room.Corpses {
		if c.Prunable {
			continue
		}
		// Player corpses are out of scope.
		if c.MobId <= 0 {
			continue
		}
		mobSpec := mobs.GetMobSpec(mobs.MobId(c.MobId))
		if mobSpec == nil {
			continue
		}
		if len(crafting.LookupCorpseSalvage(mobSpec.Groups)) > 0 {
			target = c
			found = true
			break
		}
	}

	// No eligible corpse — return handled=true silently. This prevents the
	// engine's confused-emote fallback (world.go) from firing on a no-op.
	if !found {
		return true, nil
	}

	mobSpec := mobs.GetMobSpec(mobs.MobId(target.MobId))
	returns := crafting.LookupCorpseSalvage(mobSpec.Groups)

	// Transition to Salvaging for data-shape parity with user salvage.
	// Mob salvage stays single-tick at the resolution layer — no per-round
	// messaging for mobs. RoundsTotal=1 reflects the single-tick design.
	// Guard nil Activity — test mobs constructed outside New() may not
	// have the machine initialized yet.
	if mob.Character.Activity != nil {
		_ = mob.Character.Activity.TransitionToSalvaging(
			activity.SalvagingData{
				ItemUuid:    fmt.Sprintf("corpse:%d", target.MobId),
				RoundsTotal: 1,
			},
			state.TransitionReason{
				Trigger: activity.TriggerSalvageBegin,
				Actor:   state.ActorRef{MobInstanceId: mob.InstanceId},
			},
		)
	}

	// Skill-scaled yield chance.
	bal := configs.GetBalanceConfig()
	salvageSkill := mob.Character.GetSkillLevel(skills.Salvage)
	chance := crafting.CalcSalvageChance(salvageSkill,
		float64(bal.SalvageMinChance),
		float64(bal.SalvageMaxChance),
		int(bal.SalvageSoftCap))

	recovered := crafting.RollSalvageReturnsFromSpec(returns, chance)

	// Always remove the corpse regardless of yield — matches the CLAUDE.md
	// rule "item is always consumed, even if no materials recovered."
	room.RemoveCorpse(target)

	// Store recovered materials in mob inventory.
	for _, ing := range recovered {
		for i := 0; i < ing.Quantity; i++ {
			matSpec := items.FindSpecByComponentTag(ing.ItemTag)
			if matSpec != nil {
				newItem := items.New(matSpec.ItemId)
				mob.Character.StoreItem(newItem)
			}
		}
	}

	// Emit flavor message if any players are watching.
	if room.PlayerCt() > 0 {
		sendRoomText(room,
			fmt.Sprintf(
				`<ansi fg="mobname">%s</ansi> kneels by the carcass and cuts`+
					` strips of hide from it.`,
				mob.Character.Name))
	}

	// Return Activity machine to Free — single-tick resolution complete.
	if mob.Character.Activity != nil {
		_ = mob.Character.Activity.TransitionToFree(state.TransitionReason{
			Trigger: activity.TriggerSalvageComplete,
			Actor:   state.ActorRef{MobInstanceId: mob.InstanceId},
		})
	}

	return true, nil
}
