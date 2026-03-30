// Round ticks for players
package hooks

import (
	"fmt"
	"math"
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/enchantments"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/worldevents"
)

//
// Player Round Tick
//

func UserRoundTick(e events.Event) events.ListenerReturn {

	roomsWithPlayers := rooms.GetRoomsWithPlayers()
	for _, roomId := range roomsWithPlayers {
		// Get rooom
		if room := rooms.LoadRoom(roomId); room != nil {
			room.RoundTick()

			allowIdleMessages := true
			if handled, err := scripting.TryRoomIdleEvent(roomId); err == nil {
				if handled { // For this event, handled represents whether to reject the move.
					allowIdleMessages = false
				}
			}

			if allowIdleMessages {

				chanceIn100 := 5
				if room.RoomId == -1 {
					chanceIn100 = 20
				}

				var idleMsgs []string

				if len(room.IdleMessages) > 0 {
					idleMsgs = room.IdleMessages
				} else {
					if zCfg := rooms.GetZoneConfig(room.Zone); zCfg != nil {
						if len(zCfg.IdleMessages) > 0 {
							idleMsgs = zCfg.IdleMessages
						}
					}
				}

				idleMsgCt := len(idleMsgs)
				if idleMsgCt > 0 && util.Rand(100) < chanceIn100 {

					if targetRoomId, err := strconv.Atoi(idleMsgs[0]); err == nil {
						idleMsgCt = 0
						if tgtRoom := rooms.LoadRoom(targetRoomId); tgtRoom != nil {
							idleMsgs = tgtRoom.IdleMessages
							idleMsgCt = len(idleMsgs)
						}
					}

					if idleMsgCt > 0 {
						// pick a random message
						idleMsgIndex := uint8(util.Rand(idleMsgCt))

						// If it's a repeating message, treat it as a non-message
						// (Unless it's the only one)
						if idleMsgIndex != room.LastIdleMessage || idleMsgCt == 1 {

							room.LastIdleMessage = idleMsgIndex

							msg := idleMsgs[idleMsgIndex]
							if msg != `` {
								wrappedMsg := util.SplitStringNL(msg, 65)
								if room.GetVisibility() < 1 {
									// Idle flavor text is visual — only nightvision players see it
									for _, uid := range room.GetPlayers() {
										u := users.GetByUserId(uid)
										if u != nil && u.Character.HasFlagFromAnySource(buffs.NightVision) {
											u.SendText(wrappedMsg)
										}
									}
								} else {
									room.SendText(wrappedMsg)
								}
							}

						}
					}

				}
			}

			for _, uId := range room.GetPlayers() {

				user := users.GetByUserId(uId)
				if user == nil {
					continue
				}

				if user.Character.HasAdjective(`zombie`) {
					user.Command(`zombieact`)
				}

				// Roundtick any cooldowns
				user.Character.Cooldowns.RoundTick()

				// Stage 7.5: Attempt automatic recovery from prone (uses DEX)
				if attemptMade, success := user.Character.AttemptRecovery(user.Character.Stats.Dexterity.ValueAdj); attemptMade {
					if success {
						user.SendText("You scramble to your feet!")
						if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
							room.SendText("<ansi fg=\"username\">"+user.Character.Name+"</ansi> clambers to their feet in a rushed panic.", user.UserId)
						}
					} else {
						user.SendText("You attempt to stand, but slip back down in the chaos of battle!")
						if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
							room.SendText("<ansi fg=\"username\">"+user.Character.Name+"</ansi> attempts to stand, but slips and falls in the chaos of battle.", user.UserId)
						}
					}
				}

				if user.Character.Charmed != nil && user.Character.Charmed.RoundsRemaining > 0 {
					user.Character.Charmed.RoundsRemaining--
				}

				if triggeredBuffs := user.Character.Buffs.Trigger(); len(triggeredBuffs) > 0 {

					//
					// Fire onTrigger for buff script
					//
					triggeredBuffIds := []int{}
					for _, buff := range triggeredBuffs {

						if buff.Expired() {
							triggeredBuffIds = append(triggeredBuffIds, buff.BuffId)
							continue
						}

						_, err := scripting.TryBuffScriptEvent(`onTrigger`, uId, 0, buff.BuffId)

						if buff.TriggersLeft != buffs.TriggersLeftUnlimited || err != scripting.ErrEventNotFound {
							triggeredBuffIds = append(triggeredBuffIds, buff.BuffId)
						}

					}

					events.AddToQueue(events.BuffsTriggered{UserId: user.UserId, BuffIds: triggeredBuffIds})
				}

				// Stage 9.8: Tick all combat conditions (decrements Duration, removes expired)
				user.Character.TickConditions()

				// Stage 12.2: Mutation progress — accumulates during combat, triggers acquisition or deepening
				// Stage 17.2: The Eye modulates how quickly mutations happen (0.5× at new moon, 1.5× at full)
				if user.Character.Aggro != nil {
					mb := configs.GetBalanceConfig()
					canAcquire := len(user.Character.Mutations) < int(mb.MutationMaxCount)
					canDeepen := mutations.CanDeepen(user.Character.Mutations)
					if canAcquire || canDeepen {
						eyeMult := 0.5 + gametime.GetEyePhase()
						// Phase 25.3: Mutation Catalyst buff doubles mutation progress gain
						mutCatalystMult := 1.0
						if user.Character.HasBuffFlag(buffs.MutationRate) {
							mutCatalystMult = 2.0
						}
						user.Character.MutationProgress += float64(mb.MutationProgressGainPerRound) * eyeMult * mutCatalystMult
						// Phase 24.1: Use rarity-weighted load instead of flat event count
						load := mutations.GetMutationLoad(user.Character.Mutations)
						threshold := float64(mb.MutationBaseProgress) *
							math.Pow(float64(mb.MutationProgressScale), load)
						if user.Character.MutationProgress >= threshold {
							user.Character.MutationProgress = 0
							if canAcquire {
								pool := mutations.GetWeightedPool(user.Character.Mutations)
								if len(pool) > 0 {
									mutId := mutations.RollAcquisition(pool)
									if user.Character.Mutations == nil {
										user.Character.Mutations = make(map[string]int)
									}
									user.Character.Mutations[mutId] = 1
									spec := mutations.GetMutation(mutId)
									if spec != nil {
										user.SendText(fmt.Sprintf(
											`<ansi fg="magenta">Something stirs beneath your skin. A mutation emerges: <ansi fg="yellow">%s</ansi>.</ansi>`,
											spec.Name))
										user.SendText(fmt.Sprintf(`<ansi fg="magenta">%s</ansi>`, spec.Description))

										// Emit world event for gossip system
										sig := worldevents.Regional
										if spec.Rarity >= 8 {
											sig = worldevents.Global
										}
										zone := user.Character.Zone
										region := ""
										if zCfg := rooms.GetZoneConfig(zone); zCfg != nil {
											region = zCfg.Region
										}
										worldevents.EmitWorldEvent(worldevents.WorldEvent{
											Type:         worldevents.PlayerMutationMilestone,
											Significance: sig,
											ZoneName:     zone,
											RegionName:   region,
											PlayerName:   user.Character.Name,
											Description: fmt.Sprintf("%s has undergone a mutation: %s.",
												user.Character.Name, spec.Name),
										})
									}
								}
							} else if canDeepen {
								mutId := mutations.RollDeepening(user.Character.Mutations)
								if mutId != "" {
									user.Character.Mutations[mutId]++
									newLevel := user.Character.Mutations[mutId]
									if spec := mutations.GetMutation(mutId); spec != nil {
										levelTag := fmt.Sprintf("Level %d", newLevel)
										if newLevel >= int(mb.MutationMaxLevel) {
											levelTag = "fully matured"
										}
										user.SendText(fmt.Sprintf(
											`<ansi fg="magenta">The Chrysalis deepens its hold. Your <ansi fg="yellow">%s</ansi> grows stronger (%s).</ansi>`,
											spec.Name, levelTag))
									}
								}
							}
						}
					}
				}

				// Stage 13.1: Crafting tick — advance or complete active crafting
				if user.Character.CraftingState != nil {
					if user.Character.Aggro != nil {
						user.Character.CraftingState = nil
						user.SendText(`<ansi fg="red">Your work is interrupted!</ansi>`)
					} else {
						cs := user.Character.CraftingState
						cs.RoundsComplete++
						if cs.RoundsComplete < cs.RoundsTotal {
							user.SendText(fmt.Sprintf(
								`<ansi fg="yellow">You continue working on %s... (%d/%d)</ansi>`,
								cs.RecipeId, cs.RoundsComplete, cs.RoundsTotal))
						} else {
							recipe := crafting.GetRecipe(cs.RecipeId)
							user.Character.CraftingState = nil
							if recipe != nil {
								sl := user.Character.Skills[recipe.Skill]
								chance := crafting.CalcSuccessChance(sl, recipe.SkillMinimum)
								roll := util.Rand(100)
								util.LogRoll("Craft", roll, chance)
								if roll < chance {
									user.Character.Items, user.Character.ComponentItems = crafting.ConsumeIngredients(user.Character.Items, user.Character.ComponentItems, recipe)

									if crafting.IsEnchantingRecipe(recipe) {
										// Enchanting: find target item, apply enchantment
										eq := user.Character.Equipment
										equipSlots := []crafting.EquipmentSlot{
											{Item: eq.Weapon, Label: "wielded"},
											{Item: eq.Offhand, Label: "offhand"},
											{Item: eq.ExtraArm1, Label: "extra arm 1"},
											{Item: eq.ExtraArm2, Label: "extra arm 2"},
											{Item: eq.ExtraArm3, Label: "extra arm 3"},
											{Item: eq.ExtraArm4, Label: "extra arm 4"},
											{Item: eq.Head, Label: "worn - head"},
											{Item: eq.Neck, Label: "worn - neck"},
											{Item: eq.Shoulders, Label: "worn - shoulders"},
											{Item: eq.Body, Label: "worn - body"},
											{Item: eq.Back, Label: "worn - back"},
											{Item: eq.Belt, Label: "worn - belt"},
											{Item: eq.Wrist1, Label: "worn - wrist"},
											{Item: eq.Wrist2, Label: "worn - wrist"},
											{Item: eq.ExtraWrist1, Label: "extra wrist 1"},
											{Item: eq.ExtraWrist2, Label: "extra wrist 2"},
											{Item: eq.ExtraWrist3, Label: "extra wrist 3"},
											{Item: eq.ExtraWrist4, Label: "extra wrist 4"},
											{Item: eq.Gloves, Label: "worn - gloves"},
											{Item: eq.Ring, Label: "worn - ring"},
											{Item: eq.Ring2, Label: "worn - ring"},
											{Item: eq.Legs, Label: "worn - legs"},
											{Item: eq.Feet, Label: "worn - feet"},
										}
										candidates := crafting.FindTargetItems(user.Character.Items, equipSlots, recipe.TargetType, "")
										if len(candidates) > 0 {
											c := candidates[0]
											var targetItem *items.Item
											if c.BackpackIdx >= 0 {
												targetItem = &user.Character.Items[c.BackpackIdx]
											} else {
												targetItem = user.Character.Equipment.GetSlotPointer(c.SourceLabel)
											}
											if targetItem != nil {
												eDef := enchantments.GetEnchantment(recipe.EnchantType)
												if eDef != nil {
													targetItem.EnchantType = recipe.EnchantType
													targetItem.EnchantTier = 0
													targetItem.EnchantUses = 0
													targetItem.ReservePool = eDef.ReservePool
													enchantments.ApplyTier(targetItem, eDef, 0)
												}
											}
										}
									} else {
										// Normal crafting: produce output item
										newItem := items.New(recipe.Output.ItemId)
										newItem.CraftedRound = util.GetRoundCount()
										newItem.CraftSkill = user.Character.GetSkillLevel(skills.SkillTag(recipe.Skill))
										user.Character.StoreItem(newItem)
										events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: newItem, Gained: true})
									}
									user.Character.OnSkillUse(recipe.Skill, user.UserId)
									user.SendText(fmt.Sprintf(`<ansi fg="green">%s</ansi>`, recipe.SuccessMessage))

									// Stage 31.1: Recipe discovery roll
									bal := configs.GetBalanceConfig()
									knownCount := len(user.Character.KnownRecipes)
									discChance := float64(bal.RecipeDiscoveryBaseChance) /
										(1.0 + float64(knownCount)*float64(bal.RecipeDiscoveryDecayRate))
									if util.Rand(100) < int(discChance) {
										eligible := crafting.GetEligibleRecipes(
											user.Character.KnownRecipes,
											user.Character.Skills,
											recipe.Skill)
										if len(eligible) > 0 {
											pick := eligible[util.Rand(len(eligible))]
											if user.Character.LearnRecipe(pick) {
												if newRecipe := crafting.GetRecipe(pick); newRecipe != nil {
													user.SendText(fmt.Sprintf(
														`<ansi fg="yellow-bold">A new idea takes shape in your mind: %s!</ansi>`, newRecipe.Name))
												}
											}
										}
									}
								} else {
									user.Character.Items, user.Character.ComponentItems = crafting.ConsumeIngredients(user.Character.Items, user.Character.ComponentItems, recipe)
									user.SendText(fmt.Sprintf(`<ansi fg="red">%s</ansi>`, recipe.FailureMessage))
								}
							}
						}
					}
				}

				// Stage 31.6: Chrysalis enchantment ticking
				for _, itemPtr := range user.Character.Equipment.GetAllItemPtrs() {
					if !itemPtr.HasChrysalisEnchantment() {
						continue
					}
					itemPtr.EnchantUses++

					eDef := enchantments.GetEnchantment(itemPtr.EnchantType)
					if eDef == nil {
						continue
					}

					currentTier := itemPtr.EnchantTier
					maxTier := int(configs.GetBalanceConfig().EnchantMaxTier)
					if currentTier >= maxTier || currentTier >= len(eDef.Tiers)-1 {
						continue
					}

					bal := configs.GetBalanceConfig()
					threshold := float64(bal.EnchantTierUsesBase) * math.Pow(float64(bal.EnchantTierUsesScale), float64(currentTier))
					if float64(itemPtr.EnchantUses) >= threshold {
						if util.Rand(100) < int(float64(bal.EnchantTierUpBaseChance)*100) {
							itemPtr.EnchantTier++
							itemPtr.EnchantUses = 0
							enchantments.ApplyTier(itemPtr, eDef, itemPtr.EnchantTier)

							newTier := itemPtr.EnchantTier
							if newTier < len(eDef.Tiers) && eDef.Tiers[newTier].TierUpMessage != "" {
								user.SendText(fmt.Sprintf(`<ansi fg="magenta">%s</ansi>`, eDef.Tiers[newTier].TierUpMessage))
							}
						}
					}
				}

				// Recalculate all stats at the end of the round tick
				user.Character.Validate()

	
			}

		}

	}

	return events.Continue
}
