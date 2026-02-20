// Round ticks for players
package hooks

import (
	"fmt"
	"math"
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

//
// Player Round Tick
//

func UserRoundTick(e events.Event) events.ListenerReturn {

	evt := e.(events.NewRound)

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
								room.SendText(msg)
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
				if user.Character.Aggro != nil {
					canAcquire := len(user.Character.Mutations) < mutations.MutationMaxCount
					canDeepen := mutations.CanDeepen(user.Character.Mutations)
					if canAcquire || canDeepen {
						user.Character.MutationProgress += mutations.MutationProgressGain
						evts := mutations.TotalMutationEvents(user.Character.Mutations)
						threshold := mutations.MutationBaseProgress *
							math.Pow(mutations.MutationProgressScale, float64(evts))
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
									}
								}
							} else if canDeepen {
								mutId := mutations.RollDeepening(user.Character.Mutations)
								if mutId != "" {
									user.Character.Mutations[mutId]++
									newLevel := user.Character.Mutations[mutId]
									if spec := mutations.GetMutation(mutId); spec != nil {
										levelTag := fmt.Sprintf("Level %d", newLevel)
										if newLevel >= mutations.MutationMaxLevel {
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
									user.Character.Items = crafting.ConsumeIngredients(user.Character.Items, recipe)
									newItem := items.New(recipe.Output.ItemId)
									user.Character.StoreItem(newItem)
									events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: newItem, Gained: true})
									user.Character.OnSkillUse(recipe.Skill, user.UserId)
									user.SendText(fmt.Sprintf(`<ansi fg="green">%s</ansi>`, recipe.SuccessMessage))
								} else {
									user.Character.Items = crafting.ConsumeIngredients(user.Character.Items, recipe)
									user.SendText(fmt.Sprintf(`<ansi fg="red">%s</ansi>`, recipe.FailureMessage))
								}
							}
						}
					}
				}

				// Recalculate all stats at the end of the round tick
				user.Character.Validate()

				// Only do this every 15 rounds to keep spam down.
				if evt.RoundNumber%15 == 0 {

					if !user.DidTip(`status train`) && user.Character.StatPoints > 0 {
						user.SendText(`<ansi fg="alert-5">TIP:</ansi> <ansi fg="tip-text">Type <ansi fg="command">status train</ansi> to use the status points you've earned through leveling.</ansi>`)
						user.SendText(``)
					}

				}

			}

		}

	}

	return events.Continue
}
