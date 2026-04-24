// Round ticks for players
package hooks

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/enchantments"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/textutil"
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
			behaviortree.TryRoomBehavior(roomId, behaviortree.EventContext{
				EventType: "room_idle",
				RoomId:    roomId,
			})

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
								wrappedMsg := util.SplitStringNL(msg, 80)
								if room.GetVisibility() < 1 {
									// Idle flavor text is visual — only nightvision players see it
									for _, uid := range room.GetPlayers() {
										u := users.GetByUserId(uid)
										if u != nil && u.Character.HasFlagFromAnySource(buffs.NightVision) {
											u.SendText(wrappedMsg)
										}
									}
								} else {
									sendVisualRoomText(room, wrappedMsg)
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
							sendVisualRoomText(room, "<ansi fg=\"username\">"+user.Character.Name+"</ansi> clambers to their feet in a rushed panic.", user.UserId)
						}
					} else {
						user.SendText("You attempt to stand, but slip back down in the chaos of battle!")
						if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
							sendVisualRoomText(room, "<ansi fg=\"username\">"+user.Character.Name+"</ansi> attempts to stand, but slips and falls in the chaos of battle.", user.UserId)
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

						// Send YAML trigger text (if defined).
						trigBuffSpec := buffs.GetBuffSpec(buff.BuffId)
						if trigBuffSpec != nil && (trigBuffSpec.TriggerUserText != "" || trigBuffSpec.TriggerRoomText != "") {
							tCtx := textutil.TokenContext{
								SourceName:      user.Character.GetCharacterName(true),
								SourcePlainName: user.Character.GetCharacterName(false),
							}
							cfg := textutil.SendTextConfig{
								UserSendFunc: func(msg string) { user.SendText(msg) },
								RoomSendFunc: func(msg string, skip ...int) {
									if r := rooms.LoadRoom(user.Character.RoomId); r != nil {
										r.SendText(msg, skip...)
									}
								},
								ExcludeId: user.UserId,
							}
							textutil.SendPhaseText(trigBuffSpec.TriggerUserText, trigBuffSpec.TriggerRoomText, tCtx, "cyan", cfg)
						}

						// Apply config-driven tick amount (if set)
						if buff.TickAmount != 0 {
							if trigBuffSpec == nil {
								trigBuffSpec = buffs.GetBuffSpec(buff.BuffId)
							}
							if trigBuffSpec != nil {
								switch trigBuffSpec.TickPool {
								case "health":
									user.Character.Heal(buff.TickAmount)
								case "stamina":
									user.Character.Stamina += buff.TickAmount
									if user.Character.Stamina > user.Character.StaminaMax.Value {
										user.Character.Stamina = user.Character.StaminaMax.Value
									} else if user.Character.Stamina < 0 {
										user.Character.Stamina = 0
									}
								case "conviction":
									user.Character.Conviction += buff.TickAmount
									if user.Character.Conviction > user.Character.ConvictionMax.Value {
										user.Character.Conviction = user.Character.ConvictionMax.Value
									} else if user.Character.Conviction < 0 {
										user.Character.Conviction = 0
									}
								}
							}
						}

						triggeredBuffIds = append(triggeredBuffIds, buff.BuffId)

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
							// Decide: deepen existing mutation or acquire new one
							doDeepen := false
							if canAcquire && canDeepen {
								// Both possible — coin flip weighted toward deepening
								if util.Rand(100) < int(mb.MutationDeepenChance*100) {
									doDeepen = true
								}
							} else if canDeepen && !canAcquire {
								// At max count — must deepen
								doDeepen = true
							}
							// else: canAcquire && !canDeepen — acquire new (doDeepen stays false)

							if doDeepen {
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
							} else if canAcquire {
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
							progressMsg := cs.RecipeId
							if len(cs.RecipeId) > 8 && cs.RecipeId[:8] == "salvage:" {
								progressMsg = "salvaging"
							}
							user.SendText(fmt.Sprintf(
								`<ansi fg="yellow">You continue working on %s... (%d/%d)</ansi>`,
								progressMsg, cs.RoundsComplete, cs.RoundsTotal))
						} else if len(cs.RecipeId) > 8 && cs.RecipeId[:8] == "salvage:" {
							// Salvage completion
							itemIdStr := cs.RecipeId[8:]
							user.Character.CraftingState = nil
							resolveSalvage(user, itemIdStr)
						} else {
							recipe := crafting.GetRecipe(cs.RecipeId)
							enchantTargetSlot := cs.TargetSlot
							user.Character.CraftingState = nil
							if recipe != nil {
								sl := user.Character.Skills[recipe.Skill]
								chance := crafting.CalcSuccessChance(sl, recipe.SkillMinimum)
								roll := util.Rand(100)
								util.LogRoll("Craft", roll, chance)
								if roll < chance {
									// Before consuming, find bottle aging multiplier if recipe uses a bottle
									var bottleAgingMult float64
									for _, ing := range recipe.Ingredients {
										if ing.ItemTag == "bottle" {
											for _, itm := range user.Character.Items {
												if itm.GetSpec().ComponentTag == "bottle" && itm.GetSpec().BottleAgingMultiplier > 0 {
													bottleAgingMult = itm.GetSpec().BottleAgingMultiplier
													break
												}
											}
											if bottleAgingMult == 0 {
												for _, itm := range user.Character.ComponentItems {
													if itm.GetSpec().ComponentTag == "bottle" && itm.GetSpec().BottleAgingMultiplier > 0 {
														bottleAgingMult = itm.GetSpec().BottleAgingMultiplier
														break
													}
												}
											}
											break
										}
									}

									if crafting.IsEnchantingRecipe(recipe) {
										// Enchanting: use the stored slot label to find the target
										targetItem := user.Character.Equipment.GetSlotPointer(enchantTargetSlot)
										if targetItem == nil || targetItem.ItemId < 1 {
											user.SendText(`<ansi fg="red">The item is no longer equipped. The enchanting fails, but your materials are returned.</ansi>`)
										} else {
											user.Character.Items, user.Character.ComponentItems = crafting.ConsumeIngredients(user.Character.Items, user.Character.ComponentItems, recipe)
											eDef := enchantments.GetEnchantment(recipe.EnchantType)
											if eDef != nil {
												targetItem.EnchantType = recipe.EnchantType
												targetItem.EnchantTier = 0
												targetItem.EnchantUses = 0
												targetItem.ReservePool = eDef.ReservePool
												enchantments.ApplyTier(targetItem, eDef, 0)
											}
										}
									} else {
										user.Character.Items, user.Character.ComponentItems = crafting.ConsumeIngredients(user.Character.Items, user.Character.ComponentItems, recipe)
										// Normal crafting: produce output item
										newItem := items.New(recipe.Output.ItemId)
										newItem.CraftedRound = util.GetRoundCount()
										newItem.CraftSkill = user.Character.GetSkillLevel(skills.SkillTag(recipe.Skill))
										if bottleAgingMult > 0 {
											newItem.BottleMultiplier = bottleAgingMult
										}
										// Maker's mark for skilled crafters on non-material items
										newSpec := newItem.GetSpec()
										if newItem.CraftSkill >= 30 && !newSpec.IsComponent && newSpec.Type != items.Object {
											newItem.MakerName = user.Character.Name
										}
										user.Character.StoreItem(newItem)
										events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: newItem, Gained: true})
									}
									craftBonus := 1.0 + float64(recipe.SkillMinimum)*float64(configs.GetBalanceConfig().CraftDifficultyProgressionScale)
									user.Character.OnSkillUseScaled(recipe.Skill, user.UserId, craftBonus)
									user.SendText(fmt.Sprintf(`<ansi fg="green">%s</ansi>`, recipe.SuccessMessage))

									// Stage 31.1: Recipe discovery roll
									bal := configs.GetBalanceConfig()
									knownCount := len(user.Character.KnownRecipes)
									craftSkillLevel := user.Character.GetSkillLevel(skills.SkillTag(recipe.Skill))
									discChance := configs.DiscoveryChance(configs.DiscoveryParams{
										Base:       float64(bal.RecipeDiscoveryBaseChance),
										Decay:      float64(bal.RecipeDiscoveryDecayRate),
										Known:      knownCount,
										Perception: user.Character.Stats.Perception.ValueAdj,
										Skill:      craftSkillLevel,
									})
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

				// Stage 31.6: Chrysalis enchantment ticking (combat only)
				if user.Character.Aggro != nil {
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
				}

				// Recalculate all stats at the end of the round tick
				user.Character.Validate()

	
			}

		}

	}

	return events.Continue
}

// resolveSalvage handles salvage completion when CraftingState finishes.
func resolveSalvage(user *users.UserRecord, itemIdStr string) {
	var itemId int
	fmt.Sscanf(itemIdStr, "%d", &itemId)

	spec := items.GetItemSpec(itemId)
	if spec == nil {
		user.SendText(`<ansi fg="red">Something went wrong with your salvage attempt.</ansi>`)
		return
	}

	// Find the specific item in backpack by UUID
	uuidStr, _ := user.Character.GetMiscData("salvage_item_uuid").(string)
	usesKit, _ := user.Character.GetMiscData("salvage_uses_kit").(bool)
	user.Character.SetMiscData("salvage_item_uuid", nil)
	user.Character.SetMiscData("salvage_uses_kit", nil)

	// Consume salvage kit if used
	if usesKit {
		for _, itm := range user.Character.Items {
			if itm.GetSpec().ComponentTag == "salvage-kit" {
				user.Character.RemoveItem(itm)
				break
			}
		}
	}

	found := false
	var targetItem items.Item
	for _, itm := range user.Character.Items {
		if itm.UUID.String() == uuidStr && itm.ItemId == itemId {
			targetItem = itm
			found = true
			break
		}
	}

	if !found {
		user.SendText(`<ansi fg="red">The item you were salvaging is no longer in your backpack.</ansi>`)
		return
	}

	// Calculate salvage chance from skill
	bal := configs.GetBalanceConfig()
	salvageSkill := user.Character.GetSkillLevel(skills.Salvage)
	chance := crafting.CalcSalvageChance(salvageSkill,
		float64(bal.SalvageMinChance), float64(bal.SalvageMaxChance),
		int(bal.SalvageSoftCap))

	// Roll returns from recipe, tagged salvage_returns, or spoiled potion
	var recovered []crafting.RecipeIngredient
	isSpoiledPotion, _ := user.Character.GetMiscData("salvage_spoiled_potion").(bool)
	user.Character.SetMiscData("salvage_spoiled_potion", nil)

	if isSpoiledPotion {
		// Spoiled/declining potions always return 1-2 binding paste
		qty := 1
		if chance > 0.5 {
			qty = 2
		}
		recovered = []crafting.RecipeIngredient{
			{ItemTag: "binding-paste", Quantity: qty},
		}
	} else {
		recipe := crafting.GetRecipeByOutputItemId(itemId)
		if recipe != nil {
			recovered = crafting.RollSalvageReturns(recipe.Ingredients, chance)
		} else if len(spec.SalvageReturns) > 0 {
			recovered = crafting.RollSalvageReturnsFromSpec(spec.SalvageReturns, chance)
		}
	}

	// Destroy the item (always consumed)
	user.Character.RemoveItem(targetItem)

	// Give recovered materials
	if len(recovered) > 0 {
		var parts []string
		for _, ing := range recovered {
			for i := 0; i < ing.Quantity; i++ {
				matSpec := items.FindSpecByComponentTag(ing.ItemTag)
				if matSpec != nil {
					newItem := items.New(matSpec.ItemId)
					user.Character.StoreItem(newItem)
				}
			}
			parts = append(parts, fmt.Sprintf("%dx %s",
				ing.Quantity, ing.ItemTag))
		}
		user.SendText(fmt.Sprintf(
			`<ansi fg="green">You salvage the <ansi fg="itemname">%s</ansi> and recover: %s.</ansi>`,
			targetItem.DisplayName(),
			strings.Join(parts, ", ")))
	} else {
		user.SendText(fmt.Sprintf(
			`<ansi fg="red">You attempt to salvage the <ansi fg="itemname">%s</ansi> but recover nothing useful.</ansi>`,
			targetItem.DisplayName()))
	}

	// Track skill use for progression
	user.Character.OnSkillUse("salvage", user.UserId)
}
