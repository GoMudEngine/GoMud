package hooks

import (
	"fmt"
	"math"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/worldevents"
)

func MobRoundTick(e events.Event) events.ListenerReturn {

	// Stage 38.4: Periodic mob instance saves
	roundCount := util.GetRoundCount()
	saveInterval := uint64(configs.GetBalanceConfig().MobSaveIntervalRounds)
	if saveInterval > 0 && roundCount%saveInterval == 0 {
		for _, mobInstanceId := range mobs.GetAllMobInstanceIds() {
			if mob := mobs.GetInstance(mobInstanceId); mob != nil {
				if err := mobs.SaveMobInstance(mob); err != nil {
					mudlog.Error("MobRoundTick.SaveMobInstance", "mob", mob.Character.Name, "error", err)
				}
			}
		}
	}

	//
	// Reduce existing hostility (if any)
	//
	mobs.ReduceHostility()

	// Stage 38.5.3: Pack scaling — award bonuses and emit events
	for _, bonus := range mobs.TickPackSurvival() {
		sig := worldevents.Local
		if bonus.ReachedMax {
			sig = worldevents.Regional
		}
		if len(bonus.MemberIds) > 0 {
			if firstMob := mobs.GetInstance(bonus.MemberIds[0]); firstMob != nil {
				zone := firstMob.Character.Zone
				region := ""
				if zCfg := rooms.GetZoneConfig(zone); zCfg != nil {
					region = zCfg.Region
				}
				if room := rooms.LoadRoom(firstMob.Character.RoomId); room != nil {
					room.SendText(fmt.Sprintf(
						`The <ansi fg="mobname">%s</ansi> pack moves with renewed coordination.`,
						bonus.GroupTag))
				}
				worldevents.EmitWorldEvent(worldevents.WorldEvent{
					Type:         worldevents.PackStrengthened,
					Significance: sig,
					ZoneName:     zone,
					RegionName:   region,
					MobName:      bonus.GroupTag,
					Description: fmt.Sprintf("The %s pack grows stronger through coordinated survival.",
						bonus.GroupTag),
				})
			}
		}
	}

	//
	// Do mob round maintenance
	//
	mb := configs.GetBalanceConfig()
	for _, mobInstanceId := range mobs.GetAllMobInstanceIds() {

		mob := mobs.GetInstance(mobInstanceId)

		if mob == nil {
			continue
		}

		// Roundtick any cooldowns
		mob.Character.Cooldowns.RoundTick()

		// Stage 7.5: Attempt automatic recovery from prone (uses DEX)
		if attemptMade, success := mob.Character.AttemptRecovery(mob.Character.Stats.Dexterity.ValueAdj); attemptMade {
			// Send messages to the room so players can see NPCs trying to recover
			if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
				if success {
					room.SendText("<ansi fg=\"mobname\">" + mob.Character.Name + "</ansi> clambers to their feet in a rushed panic.")
				} else {
					room.SendText("<ansi fg=\"mobname\">" + mob.Character.Name + "</ansi> attempts to stand, but slips and falls in the chaos of battle.")
				}
			}
		}

		if mob.Character.Charmed != nil && mob.Character.Charmed.RoundsRemaining > 0 {
			mob.Character.Charmed.RoundsRemaining--
		}

		if triggeredBuffs := mob.Character.Buffs.Trigger(); len(triggeredBuffs) > 0 {

			//
			// Fire onTrigger for buff script
			//
			triggeredBuffIds := []int{}
			for _, buff := range triggeredBuffs {
				scripting.TryBuffScriptEvent(`onTrigger`, 0, mobInstanceId, buff.BuffId)
				triggeredBuffIds = append(triggeredBuffIds, buff.BuffId)
			}

			events.AddToQueue(events.BuffsTriggered{MobInstanceId: mobInstanceId, BuffIds: triggeredBuffIds})
		}

		// Stage 38.5.2: Mob mutation acquisition during combat
		if bool(mb.MobMutationEnabled) && mob.Character.Aggro != nil {
			canAcquire := len(mob.Character.Mutations) < int(mb.MutationMaxCount)
			canDeepen := mutations.CanDeepen(mob.Character.Mutations)
			if canAcquire || canDeepen {
				mob.Character.MutationProgress += float64(mb.MutationProgressGainPerRound) * float64(mb.MobMutationRate)
				load := mutations.GetMutationLoad(mob.Character.Mutations)
				threshold := float64(mb.MutationBaseProgress) *
					math.Pow(float64(mb.MutationProgressScale), load)
				if mob.Character.MutationProgress >= threshold {
					mob.Character.MutationProgress = 0
					if canAcquire {
						pool := mutations.GetWeightedPool(mob.Character.Mutations)
						if mutId := mutations.RollAcquisition(pool); mutId != "" {
							if mob.Character.Mutations == nil {
								mob.Character.Mutations = make(map[string]int)
							}
							mob.Character.Mutations[mutId] = 1
							if spec := mutations.GetMutation(mutId); spec != nil {
								if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
									room.SendText(fmt.Sprintf(
										`<ansi fg="magenta">Something shifts in <ansi fg="mobname">%s</ansi>. %s</ansi>`,
										mob.Character.Name, spec.Visual))
								}
								// Emit world event with significance based on rarity
								sig := worldevents.Local
								if spec.Rarity >= 8 {
									sig = worldevents.Global
								} else if spec.Rarity >= 5 {
									sig = worldevents.Regional
								}
								zone := mob.Character.Zone
								region := ""
								if zCfg := rooms.GetZoneConfig(zone); zCfg != nil {
									region = zCfg.Region
								}
								worldevents.EmitWorldEvent(worldevents.WorldEvent{
									Type:         worldevents.MobMutationGained,
									Significance: sig,
									ZoneName:     zone,
									RegionName:   region,
									MobName:      mob.Character.Name,
									Description: fmt.Sprintf("%s has manifested a mutation: %s",
										mob.Character.Name, spec.Name),
								})
							}
						}
					} else if canDeepen {
						if mutId := mutations.RollDeepening(mob.Character.Mutations); mutId != "" {
							mob.Character.Mutations[mutId]++
							newLevel := mob.Character.Mutations[mutId]
							if spec := mutations.GetMutation(mutId); spec != nil {
								if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
									room.SendText(fmt.Sprintf(
										`<ansi fg="magenta">The mutation in <ansi fg="mobname">%s</ansi> intensifies.</ansi>`,
										mob.Character.Name))
								}
								// Deepening significance: bump one tier if level 3
								sig := worldevents.Local
								if spec.Rarity >= 5 {
									sig = worldevents.Regional
								}
								if newLevel >= int(mb.MutationMaxLevel) {
									if sig < worldevents.Global {
										sig++
									}
								}
								zone := mob.Character.Zone
								region := ""
								if zCfg := rooms.GetZoneConfig(zone); zCfg != nil {
									region = zCfg.Region
								}
								worldevents.EmitWorldEvent(worldevents.WorldEvent{
									Type:         worldevents.MobMutationAdvanced,
									Significance: sig,
									ZoneName:     zone,
									RegionName:   region,
									MobName:      mob.Character.Name,
									Description: fmt.Sprintf("%s's %s deepens to level %d",
										mob.Character.Name, spec.Name, newLevel),
								})
							}
						}
					}
				}
			}
		}

		// Do charm cleanup
		if mob.Character.IsCharmed() && mob.Character.Charmed.RoundsRemaining == 0 {
			cmd := mob.Character.Charmed.ExpiredCommand
			if charmedUserId := mob.Character.RemoveCharm(); charmedUserId > 0 {
				if charmedUser := users.GetByUserId(charmedUserId); charmedUser != nil {
					charmedUser.Character.TrackCharmed(mob.InstanceId, false)
				}
			}
			if cmd != `` {
				cmds := strings.Split(cmd, `;`)
				for _, cmd := range cmds {
					cmd = strings.TrimSpace(cmd)
					if len(cmd) > 0 {
						mob.Command(cmd)
					}
				}
			}
		}

		// Stage 9.8: Tick all combat conditions (decrements Duration, removes expired)
		mob.Character.TickConditions()

		// Recalculate all stats at the end of the round tick
		mob.Character.Validate()

		if mob.Character.Health <= 0 {
			// Mob died
			mob.Command(`suicide`)
		}

	}

	return events.Continue
}
