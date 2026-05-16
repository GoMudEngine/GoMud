package hooks

import (
	"fmt"
	"math"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/buffs"
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
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/GoMudEngine/GoMud/internal/state/life"
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
					sendVisualRoomText(room, fmt.Sprintf(
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
	activeZones := rooms.SnapshotActiveZones()

	for _, mobInstanceId := range mobs.GetAllMobInstanceIds() {

		mob := mobs.GetInstance(mobInstanceId)

		if mob == nil {
			continue
		}

		room := rooms.LoadRoom(mob.Character.RoomId)
		active := room != nil && activeZones[room.Zone]

		// Idle lane — runs every round regardless of zone activity.
		// Players returning to a cold zone expect timers to have been
		// counting, and DoT ticks can still kill a mob.
		tickMobCooldowns(mob)
		expireMobCombatMemory(mob)
		tickMobCharmDuration(mob)
		tickMobBuffs(mob, mobInstanceId)
		tickMobConditions(mob)

		// Death check always runs — a DoT tick in an idle zone should
		// still kill the mob. Skip the rest of the loop when the mob dies.
		if mob.Character.Health <= 0 && mob.Character.IsAlive() {
			mob.Character.Die(state.ActorRef{}, life.TriggerHealthZero)
			continue
		}

		if !active {
			continue
		}

		// Active-only — skipped entirely in idle zones. Progression,
		// state machines coupled to combat, and player-facing work
		// (crafting) all gate behind at least one player in the zone.
		tickMobProneRecovery(mob)
		tickMobMutationAcquisition(mob, &mb)
		tickMobCharmState(mob)
		tickMobCrafting(mob)
		revalidateMobStats(mob)
	}

	// Drain behavior tree delayed action queue
	behaviortree.GetEngine().DrainQueue()

	return events.Continue
}

// tickMobCooldowns — matches the current inline block at line 97.
func tickMobCooldowns(mob *mobs.Mob) {
	mob.Character.Cooldowns.RoundTick()
}

// expireMobCombatMemory — current inline block at lines 99–105.
func expireMobCombatMemory(mob *mobs.Mob) {
	if mob.CombatMemory != nil {
		if mobs.CombatMemoryExpired(mob.CombatMemory, util.GetRoundCount(),
			int(configs.GetBalanceConfig().CombatMemoryDuration)) {
			mob.CombatMemory = nil
		}
	}
}

// tickMobProneRecovery — current inline block at lines 107–118.
func tickMobProneRecovery(mob *mobs.Mob) {
	if attemptMade, success := mob.Character.AttemptRecovery(mob.Character.Stats.Dexterity.ValueAdj); attemptMade {
		if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
			mName := mobDisplayName(mob, room, 0)
			if success {
				sendVisualRoomText(room, mName+" clambers to their feet in a rushed panic.")
			} else {
				sendVisualRoomText(room, mName+" attempts to stand, but slips and falls in the chaos of battle.")
			}
		}
	}
}

// tickMobCharmDuration — current inline block at lines 120–122. Simple
// rounds decrement; the expiry + re-roll dance is a separate helper.
func tickMobCharmDuration(mob *mobs.Mob) {
	if mob.Character.Charmed != nil && mob.Character.Charmed.RoundsRemaining > 0 {
		mob.Character.Charmed.RoundsRemaining--
	}
}

// tickMobBuffs — current inline block at lines 124–160.
func tickMobBuffs(mob *mobs.Mob, mobInstanceId int) {
	if triggeredBuffs := mob.Character.Buffs.Trigger(); len(triggeredBuffs) > 0 {
		triggeredBuffIds := []int{}
		for _, buff := range triggeredBuffs {
			if buff.TickAmount != 0 {
				if mobBuffSpec := buffs.GetBuffSpec(buff.BuffId); mobBuffSpec != nil {
					switch mobBuffSpec.TickPool {
					case "health":
						mob.Character.Heal(buff.TickAmount)
					case "stamina":
						mob.Character.Stamina += buff.TickAmount
						if mob.Character.Stamina > mob.Character.StaminaMax.Value {
							mob.Character.Stamina = mob.Character.StaminaMax.Value
						} else if mob.Character.Stamina < 0 {
							mob.Character.Stamina = 0
						}
					case "conviction":
						mob.Character.Conviction += buff.TickAmount
						if mob.Character.Conviction > mob.Character.ConvictionMax.Value {
							mob.Character.Conviction = mob.Character.ConvictionMax.Value
						} else if mob.Character.Conviction < 0 {
							mob.Character.Conviction = 0
						}
					}
				}
			}
			triggeredBuffIds = append(triggeredBuffIds, buff.BuffId)
		}
		events.AddToQueue(events.BuffsTriggered{MobInstanceId: mobInstanceId, BuffIds: triggeredBuffIds})
	}
}

// tickMobMutationAcquisition — current inline block at lines 162–260.
func tickMobMutationAcquisition(mob *mobs.Mob, mb *configs.Balance) {
	if !(bool(mb.MobMutationEnabled) && mob.Character.IsInCombat()) {
		return
	}
	canAcquire := len(mob.Character.Mutations) < int(mb.MutationMaxCount)
	canDeepen := mutations.CanDeepen(mob.Character.Mutations)
	if !(canAcquire || canDeepen) {
		return
	}
	mob.Character.MutationProgress += float64(mb.MutationProgressGainPerRound) * float64(mb.MobMutationRate)
	load := mutations.GetMutationLoad(mob.Character.Mutations)
	threshold := float64(mb.MutationBaseProgress) *
		math.Pow(float64(mb.MutationProgressScale), load)
	if mob.Character.MutationProgress < threshold {
		return
	}
	mob.Character.MutationProgress = 0
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
					sendVisualRoomText(room, fmt.Sprintf(
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
		return
	}

	if canAcquire {
		sp := species.GetSpecies(mob.Character.SpeciesId)
		pool := mutations.GetWeightedPool(mob.Character.Mutations, sp)
		if mutId := mutations.RollAcquisition(pool); mutId != "" {
			if mob.Character.Mutations == nil {
				mob.Character.Mutations = make(map[string]int)
			}
			mob.Character.Mutations[mutId] = 1
			if spec := mutations.GetMutation(mutId); spec != nil {
				if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
					sendVisualRoomText(room, fmt.Sprintf(
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

// tickMobCharmState — current inline blocks at lines 262–279 (expiry
// cleanup) + 281–353 (re-roll). Kept together since both depend on
// Charmed state.
func tickMobCharmState(mob *mobs.Mob) {
	// Charm expiry cleanup.
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

	// Re-roll contested Charisma vs Willpower on CharmDuration tick.
	if mob.Character.IsCharmed() {
		if charmedUserId := mob.Character.GetCharmedUserId(); charmedUserId > 0 {
			if owner := users.GetByUserId(charmedUserId); owner != nil {
				if comp := owner.Character.GetCompanionByInstanceId(mob.InstanceId); comp != nil &&
					comp.SourceType == characters.CompanionCharmed &&
					comp.CharmDuration > 0 {

					comp.CharmDuration--
					if comp.CharmDuration == 0 {
						manifestSkill := owner.Character.GetSkillLevel(skills.Manifestation)
						attackScore := float64(owner.Character.Stats.Charisma.ValueAdj) +
							float64(manifestSkill)*25.0

						targetPool := mob.Character.Stats.Strength.Training +
							mob.Character.Stats.Dexterity.Training +
							mob.Character.Stats.Perception.Training +
							mob.Character.Stats.Vitality.Training +
							mob.Character.Stats.Willpower.Training +
							mob.Character.Stats.Charisma.Training
						defenseScore := float64(mob.Character.Stats.Willpower.ValueAdj) +
							float64(targetPool)*0.10

						effectiveness := 1.0 - float64(comp.CharmRerolls)*0.01*float64(comp.CharmRerolls)
						if effectiveness < 0.50 {
							effectiveness = 0.50
						}
						attackScore *= effectiveness

						success, _, _, _ := dice.OpposedRollStat(attackScore, defenseScore)

						if success {
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
							owner.SendText(fmt.Sprintf(
								`<ansi fg="red-bold">%s breaks free of your control!</ansi>`, comp.Name))
							if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
								sendVisualRoomText(room, fmt.Sprintf(
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
}

// tickMobCrafting advances or completes an active mob crafting operation.
// The mob-only combat-cancel block that previously lived here has been
// deleted — Activity_Cascades.go (Task 5) now handles combat-entry cancel
// for both mobs and players via the cascade observer (parity per AC-038).
func tickMobCrafting(mob *mobs.Mob) {
	if mob.Character.CraftingState == nil {
		return
	}
	cs := mob.Character.CraftingState
	cs.RoundsComplete++
	if cs.RoundsComplete < cs.RoundsTotal {
		return
	}
	recipe := crafting.GetRecipe(cs.RecipeId)
	// Fire Activity transition alongside legacy clear.
	if mob.Character.Activity != nil && mob.Character.Activity.IsCrafting() {
		_ = mob.Character.Activity.TransitionToFree(state.TransitionReason{
			Trigger: activity.TriggerCraftComplete,
			Actor:   mob.Character.Activity.Self(),
		})
	}
	mob.Character.CraftingState = nil
	if recipe == nil {
		return
	}
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
		craftBonus := 1.0 + float64(recipe.SkillMinimum)*float64(configs.GetBalanceConfig().CraftDifficultyProgressionScale)
		mob.Character.OnSkillUseScaled(recipe.Skill, 0, craftBonus)
		if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
			sendVisualRoomText(room, fmt.Sprintf(
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

// tickMobConditions — current inline line 399.
func tickMobConditions(mob *mobs.Mob) {
	mob.Character.TickConditions()
}

// revalidateMobStats — current inline line 402.
func revalidateMobStats(mob *mobs.Mob) {
	mob.Character.Validate()
}
