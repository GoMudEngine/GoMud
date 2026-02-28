package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/conversations"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/worldevents"
)

//
// Handles default mob idle behavior
//

func HandleIdleMobs(e events.Event) events.ListenerReturn {

	evt := e.(events.MobIdle)

	mob := mobs.GetInstance(evt.MobInstanceId)
	if mob == nil {
		return events.Cancel
	}

	isCharmed := mob.Character.IsCharmed()

	// if a mob shouldn't be allowed to leave their area (via wandering)
	// but has somehow been displaced, such as pulling through combat, spells, or otherwise
	// tell them to path back home
	if mob.MaxWander == 0 && mob.Character.RoomId != mob.HomeRoomId {
		if !isCharmed {
			mob.Command("pathto home")
		}
	}

	if conversations.HasConverseFile(int(mob.MobId), mob.Character.Zone) && util.Rand(100) < int(configs.GetGamePlayConfig().MobConverseChance) {
		if mobRoom := rooms.LoadRoom(mob.Character.RoomId); mobRoom != nil {
			mobcommands.Converse(``, mob, mobRoom) // Execute this directly so that target mob doesn't leave the room before this command executes
		}
	}

	// Stage 38.5.4: Crafter mob tick — background activity alongside normal idle
	if result := mobs.TickMobCraft(mob); result != nil {
		if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
			if result.Success {
				room.SendText(fmt.Sprintf(
					`<ansi fg="mobname">%s</ansi> finishes crafting and sets a new item on the shelf.`,
					mob.Character.Name))
			} else {
				room.SendText(fmt.Sprintf(
					`<ansi fg="mobname">%s</ansi> frowns at a failed attempt and discards the ruined materials.`,
					mob.Character.Name))
			}
		}
		// Emit world event for rare crafts
		if result.Success {
			b := configs.GetBalanceConfig()
			rareThreshold := int(b.CrafterRareThreshold)
			if result.SkillMinimum >= rareThreshold {
				sig := worldevents.Regional
				if result.SkillMinimum >= rareThreshold*2 {
					sig = worldevents.Global
				}
				zone := result.Zone
				region := ""
				if zCfg := rooms.GetZoneConfig(zone); zCfg != nil {
					region = zCfg.Region
				}
				worldevents.EmitWorldEvent(worldevents.WorldEvent{
					Type:         worldevents.MobCraftedRare,
					Significance: sig,
					ZoneName:     zone,
					RegionName:   region,
					MobName:      result.MobName,
					Description: fmt.Sprintf("%s has crafted a rare %s.",
						result.MobName, result.RecipeName),
				})
			}
		}
	}

	// If they have idle commands, maybe do one of them?
	handled, _ := scripting.TryMobScriptEvent("onIdle", mob.InstanceId, 0, ``, nil)
	if !handled {

		if isCharmed {
			// Only some mobs can apply first aid
			// If a charmed mob can aid someone, try.
			if mob.Character.KnowsFirstAid() {
				mob.Command(`lookforaid`)
			}

			return events.Continue
		}

		if mob.MaxWander > -1 && mob.WanderCount > mob.MaxWander {
			// Not charmed and far from home, and should never leave home.
			// So go home.
			mob.Command(`pathto home`)
			return events.Continue
		}

		//
		// Look for trouble
		//
		idleCmd := `lookfortrouble`
		if util.Rand(100) < mob.ActivityLevel {
			idleCmd = mob.GetIdleCommand()
			if idleCmd == `` {
				idleCmd = `lookfortrouble`
			}
		}
		mob.Command(idleCmd)

	}

	return events.Continue
}
