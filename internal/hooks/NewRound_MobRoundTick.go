package hooks

import (
	"fmt"
	"math"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/species"
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

	// Stage 42.8: Pack roaming — coordinate alpha election and pack state
	if mobs.PackRoamingEnabled() {
		mobs.TickPackRoaming()
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
				mName := mobDisplayName(mob, room, 0)
				if success {
					room.SendText(mName + " clambers to their feet in a rushed panic.")
				} else {
					room.SendText(mName + " attempts to stand, but slips and falls in the chaos of battle.")
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
					// Decide: deepen existing mutation or acquire new one
					doDeepen := false
					if canAcquire && canDeepen {
						if util.Rand(100) < int(mb.MutationDeepenChance*100) {
							doDeepen = true
						}
					} else if canDeepen && !canAcquire {
						doDeepen = true
					}

					if doDeepen {
						if mutId := mutations.RollDeepening(mob.Character.Mutations); mutId != "" {
							mob.Character.Mutations[mutId]++
							newLevel := mob.Character.Mutations[mutId]
							if spec := mutations.GetMutation(mutId); spec != nil {
								if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
									room.SendText(fmt.Sprintf(
										`<ansi fg="magenta">The mutation in <ansi fg="mobname">%s</ansi> intensifies.</ansi>`,
										mob.Character.Name))
								}
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
					} else if canAcquire {
						var specDisabledSlots []string
						if specInfo := species.GetSpecies(mob.Character.SpeciesId); specInfo != nil {
							specDisabledSlots = specInfo.DisabledSlots
						}
						pool := mutations.GetWeightedPool(mob.Character.Mutations, specDisabledSlots)
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

		// Charm duration tick — tick CompanionInfo.CharmDuration and re-roll
		// contested Charisma vs Willpower when it expires.
		if mob.Character.IsCharmed() {
			if charmedUserId := mob.Character.GetCharmedUserId(); charmedUserId > 0 {
				if owner := users.GetByUserId(charmedUserId); owner != nil {
					if comp := owner.Character.GetCompanionByInstanceId(mob.InstanceId); comp != nil &&
						comp.SourceType == characters.CompanionCharmed &&
						comp.CharmDuration > 0 {

						comp.CharmDuration--
						if comp.CharmDuration == 0 {
							// Build attacker score: Charisma + Manifestation skill bonus
							manifestSkill := owner.Character.GetSkillLevel(skills.Manifestation)
							attackScore := float64(owner.Character.Stats.Charisma.ValueAdj) +
								float64(manifestSkill)*25.0

							// Build defender score: Willpower + 10% of total training (proxy for experience)
							targetPool := mob.Character.Stats.Strength.Training +
								mob.Character.Stats.Dexterity.Training +
								mob.Character.Stats.Perception.Training +
								mob.Character.Stats.Vitality.Training +
								mob.Character.Stats.Willpower.Training +
								mob.Character.Stats.Charisma.Training
							defenseScore := float64(mob.Character.Stats.Willpower.ValueAdj) +
								float64(targetPool)*0.10

							// Diminishing effectiveness per re-roll (quadratic penalty, floor 50%)
							effectiveness := 1.0 - float64(comp.CharmRerolls)*0.01*float64(comp.CharmRerolls)
							if effectiveness < 0.50 {
								effectiveness = 0.50
							}
							attackScore *= effectiveness

							success, _, _, _ := dice.OpposedRollStat(attackScore, defenseScore)

							if success {
								// Charm holds — reset duration based on caster stats
								newDuration := 50 + owner.Character.Stats.Charisma.ValueAdj/2 +
									manifestSkill*3
								comp.CharmDuration = newDuration
								comp.CharmRerolls++

								owner.SendText(fmt.Sprintf(
									`<ansi fg="cyan">Your hold on %s wavers... but you reassert your will.</ansi>`,
									comp.Name))
								if comp.CharmRerolls >= 5 {
									owner.SendText(fmt.Sprintf(
										`<ansi fg="red">%s's eyes flash with defiance. Your control is slipping...</ansi>`,
										comp.Name))
								} else if comp.CharmRerolls >= 3 {
									owner.SendText(fmt.Sprintf(
										`<ansi fg="yellow">You sense %s's will straining against your bond...</ansi>`,
										comp.Name))
								}
							} else {
								// Charm breaks — mob turns hostile
								owner.SendText(fmt.Sprintf(
									`<ansi fg="red-bold">%s breaks free of your control!</ansi>`, comp.Name))
								if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
									room.SendText(fmt.Sprintf(
										`<ansi fg="red">%s snarls and turns on %s!</ansi>`,
										mob.Character.Name, owner.Character.Name), owner.UserId)
								}
								mob.Character.RemoveCharm()
								owner.Character.TrackCharmed(mob.InstanceId, false)
								owner.Character.RemoveCompanion(mob.InstanceId)
								mob.Character.SetAggro(owner.UserId, 0, characters.DefaultAttack)
							}
						}
					}
				}
			}
		}

		// Crafting tick — advance or complete active crafting for mob NPCs.
		// Mobs are interrupted by combat (same behaviour as players).
		if mob.Character.CraftingState != nil {
			if mob.Character.Aggro != nil {
				mob.Character.CraftingState = nil
			} else {
				cs := mob.Character.CraftingState
				cs.RoundsComplete++
				if cs.RoundsComplete >= cs.RoundsTotal {
					recipe := crafting.GetRecipe(cs.RecipeId)
					mob.Character.CraftingState = nil
					if recipe != nil {
						sl := mob.Character.Skills[recipe.Skill]
						chance := crafting.CalcSuccessChance(sl, recipe.SkillMinimum)
						roll := util.Rand(100)
						util.LogRoll("MobCraft", roll, chance)
						if roll < chance {
							mob.Character.Items, mob.Character.ComponentItems =
								crafting.ConsumeIngredients(
									mob.Character.Items,
									mob.Character.ComponentItems,
									recipe)
							newItem := items.New(recipe.Output.ItemId)
							mob.Character.StoreItem(newItem)
							mob.Character.OnSkillUse(recipe.Skill, 0)
							if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
								room.SendText(fmt.Sprintf(
									`<ansi fg="mobname">%s</ansi> finishes their work.`,
									mob.Character.Name))
							}
						} else {
							mob.Character.Items, mob.Character.ComponentItems =
								crafting.ConsumeIngredients(
									mob.Character.Items,
									mob.Character.ComponentItems,
									recipe)
						}
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
