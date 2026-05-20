package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/worldevents"
)

// wireMobDeathBroadcast subscribes to Life Alive→Dead transitions on
// mob characters and handles:
//   1. Room broadcast "X has died."  (respects dark rooms / night vision)
//   2. Guide mob (MobId 38) tempdata for respawn throttle
//   3. worldevents.MobKilledByPlayer emission when a player got the kill
//
// Migrated from internal/mobcommands/suicide.go lines 66-114 as
// part of chunk-2 Task 10.
func wireMobDeathBroadcast(c *characters.Character) {
	c.Life.Inner().AfterTransition("mob_death_broadcast",
		func(from, to life.State, r state.TransitionReason) {
			if from != life.Alive || to != life.Dead {
				return
			}
			// Only fire for mob characters.
			if c.MobInstanceId == 0 {
				return
			}
			m := mobs.GetInstance(c.MobInstanceId)
			if m == nil {
				return
			}
			room := rooms.LoadRoom(m.Character.RoomId)
			if room == nil {
				return
			}

			d, _ := c.Life.DeadData()

			// 1. Room broadcast — replicate sendMovementMessage behaviour.
			//    In lit rooms use SendTextVisual; in dark rooms respect
			//    per-player night vision, falling back to the audible cue.
			deathMsg := fmt.Sprintf(
				`<ansi fg="mobname">%s</ansi> has died.`,
				m.Character.Name,
			)
			soundMsg := `You hear something collapse to the ground.`
			if room.GetVisibility() >= 1 {
				room.SendTextVisualLegacy(deathMsg)
			} else {
				for _, uid := range room.GetPlayers() {
					u := users.GetByUserId(uid)
					if u == nil {
						continue
					}
					if u.Character.HasFlagFromAnySource(buffs.NightVision) {
						u.SendTextLegacy(deathMsg)
					} else {
						u.SendTextLegacy(soundMsg)
					}
				}
			}

			// 2. Guide special-case: mark lastGuideRound to prevent an
			//    immediate respawn.  MobId 38 = "The Guide".
			if m.MobId == 38 && m.Character.Charmed != nil {
				currentRound := util.GetRoundCount()
				if tmpU := users.GetByUserId(m.Character.Charmed.UserId); tmpU != nil {
					tmpU.SetTempData(`lastGuideRound`, currentRound)
				}
			}

			// 3. worldevents emission when a player scored the kill.
			if len(d.DamageMap) > 0 && m.Character.Zone != `Training` {
				killerName := "an adventurer"
				for uid := range d.DamageMap {
					if u := users.GetByUserId(uid); u != nil {
						killerName = u.Character.Name
						break
					}
				}
				sig := worldevents.Local
				region := ""
				if zCfg := rooms.GetZoneConfig(m.Character.Zone); zCfg != nil {
					region = zCfg.Region
				}
				worldevents.EmitWorldEvent(worldevents.WorldEvent{
					Type:         worldevents.MobKilledByPlayer,
					Significance: sig,
					ZoneName:     m.Character.Zone,
					RegionName:   region,
					MobName:      m.Character.Name,
					PlayerName:   killerName,
					Description: fmt.Sprintf(
						"%s slew %s out in %s.",
						killerName, m.Character.Name, m.Character.Zone,
					),
				})
			}
		})
}

func init() {
	characters.OnCharacterCreated(wireMobDeathBroadcast)
}
