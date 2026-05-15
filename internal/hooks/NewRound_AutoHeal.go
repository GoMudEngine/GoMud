package hooks

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/mutators"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

//
// Watches the rounds go by
// Applies autohealing where appropriate
//

func AutoHeal(e events.Event) events.ListenerReturn {

	evt := e.(events.NewRound)

	// Every 3 rounds. Else, pass it along.
	if evt.RoundNumber%3 != 0 {
		return events.Continue
	}

	deathRecoveryRoomId := int(configs.GetSpecialRoomsConfig().DeathRecoveryRoom)

	onlineIds := users.GetOnlineUserIds()
	for _, userId := range onlineIds {
		user := users.GetByUserId(userId)

		if user.Character.RoomId == deathRecoveryRoomId {
			continue
		}

		regenMultiplier := roomRegenMultiplier(rooms.LoadRoom(user.Character.RoomId))

		inCombat := user.Character.IsInCombat()
		healthStart := user.Character.Health

		// Death on zero — any path that dropped Health below 1 (damage,
		// DoT, grenade, etc.) gets caught here on the next round tick.
		// DoCombat handles in-combat deaths same-tick; AutoHeal is the
		// catch-all for non-combat deaths (poison, DoT, grenade, etc.).
		// Environmental death has no player killer — empty ActorRef.
		if user.Character.Health < 1 && user.Character.IsAlive() {
			user.Character.Die(state.ActorRef{}, life.TriggerHealthZero)
			continue
		}

		// Toxicity decay
		if user.Character.Toxicity > 0 {
			bal := configs.GetBalanceConfig()
			user.Character.Toxicity -= float64(bal.ToxicityDecayPerTick)
			if user.Character.Toxicity < 0 {
				user.Character.Toxicity = 0
			}
		}

		// Auto-eject spoiled items from bandolier
		if len(user.Character.PotionItems) > 0 {
			currentRound := util.GetRoundCount()
			ejected := 0
			remaining := make([]items.Item, 0, len(user.Character.PotionItems))
			for _, pot := range user.Character.PotionItems {
				potSpec := pot.GetSpec()
				if potSpec.Aging.HasAging() && pot.CraftedRound > 0 {
					elapsed := currentRound - pot.CraftedRound
					bottleMult := pot.BottleMultiplier
					if bottleMult <= 0 {
						bottleMult = potSpec.BottleAgingMultiplier
					}
					effSpeed := items.CalcEffectiveAgingSpeed(bottleMult, pot.CraftSkill)
					phase, _ := items.GetAgingPhase(elapsed, potSpec.Aging, effSpeed)
					if phase == items.PhaseSpoiled {
						if potSpec.Subtype == items.Throwable {
							// Spoiled grenades fall safely to the ground
							if userRoom := rooms.LoadRoom(user.Character.RoomId); userRoom != nil {
								userRoom.AddItem(pot, false)
							}
							user.SendText(fmt.Sprintf(
								`<ansi fg="yellow">Your <ansi fg="itemname">%s</ansi> has destabilized and falls out of your bandolier onto the ground.</ansi>`,
								pot.DisplayName()))
						} else {
							// Spoiled potions go to backpack
							user.Character.Items = append(user.Character.Items, pot)
							user.SendText(fmt.Sprintf(
								`<ansi fg="yellow">Your <ansi fg="itemname">%s</ansi> has spoiled and falls out of your bandolier.</ansi>`,
								pot.DisplayName()))
						}
						ejected++
						continue
					}
				}
				remaining = append(remaining, pot)
			}
			if ejected > 0 {
				user.Character.PotionItems = remaining
			}
		}

		// Toxicity regen penalty (applied to all pool regen below)
		toxRegenMult, _, _ := user.Character.GetToxicityPenalties()

		// Regeneration (only heal health if health > 0)
		if user.Character.Health > 0 {

			if !inCombat {
				// Out of combat: base %-regen, then mutation multipliers, then room multiplier
				healthRegen := float64(user.Character.HealthPerRound()) * toxRegenMult

				// Mutation health regen multiplier (e.g. Healing Gel, Regenerative Tissue)
				if mult := mutations.GetHealthRegenMultiplier(user.Character.Mutations); mult != 0 {
					healthRegen *= (1.0 + mult)
				}

				// Conditional multiplier (e.g. Photosynthetic Skin in lit rooms)
				if userRoom := rooms.LoadRoom(user.Character.RoomId); userRoom != nil {
					biome := userRoom.GetBiome()
					if condMult := mutations.GetConditionalHealthRegenMultiplier(user.Character.Mutations, biome.IsLit()); condMult != 0 {
						healthRegen *= (1.0 + condMult)
					}
				}

				// ConditionRegen from heal spell — multiplier on base regen
				if user.Character.HasCondition(characters.ConditionRegen) {
					regenMult := user.Character.GetConditionMagnitude(characters.ConditionRegen)
					if regenMult > 1.0 {
						healthRegen *= regenMult
					}
				}

				// Room multiplier applied last
				healAmt := int(math.Floor(healthRegen * regenMultiplier))
				if healAmt < 1 {
					healAmt = 1
				}
				user.Character.Heal(healAmt)

				// Emit tick feedback while a heal-spell ConditionRegen is active
				// so players get confirmation their mend-wounds (or similar) is working.
				if user.Character.HasCondition(characters.ConditionRegen) {
					user.SendText(fmt.Sprintf(
						`<ansi fg="green">Your wounds knit closed. (%s)</ansi>`,
						combat.GetHealDescription(healAmt, user.Character.HealthMax.Value)))
				}

			} else {
				// In combat: no base regen, but ConditionRegen (heal spell) still applies
				if user.Character.HasCondition(characters.ConditionRegen) {
					regenMult := user.Character.GetConditionMagnitude(characters.ConditionRegen)
					healAmt := int(math.Floor(float64(user.Character.HealthPerRound()) * toxRegenMult * regenMult * regenMultiplier))
					if healAmt < 1 {
						healAmt = 1
					}
					user.Character.Heal(healAmt)
					user.SendText(fmt.Sprintf(
						`<ansi fg="green">Your wounds knit closed. (%s)</ansi>`,
						combat.GetHealDescription(healAmt, user.Character.HealthMax.Value)))
				}
			}

			// Phase 24.5: Apply poison DoT damage
			if user.Character.HasCondition(characters.ConditionPoisoned) {
				poisonDmg := int(user.Character.GetConditionMagnitude(characters.ConditionPoisoned))
				if poisonDmg < 1 {
					poisonDmg = 1
				}
				user.Character.Health -= poisonDmg
				user.SendText(`<ansi fg="green">The poison burns through your veins!</ansi>`)
			}

			// Stage 42.7: Apply bleed DoT damage
			if user.Character.HasCondition(characters.ConditionBleeding) {
				bleedDmg := int(user.Character.GetConditionMagnitude(characters.ConditionBleeding))
				if bleedDmg < 1 {
					bleedDmg = 1
				}
				user.Character.Health -= bleedDmg
				user.SendText(`<ansi fg="red">Blood seeps from your wounds!</ansi>`)
			}
		}

		// Regenerate Stamina - slower during combat
		var staminaRegen int
		if inCombat {
			// Combat: 1/4 normal regen rate (much slower)
			staminaRegen = user.Character.StaminaPerRound() / 4
			if staminaRegen < 1 {
				staminaRegen = 1 // Minimum 1 stamina per 3 rounds even in combat
			}
		} else {
			// Out of combat: full regen
			staminaRegen = user.Character.StaminaPerRound()
		}
		staminaRegen = int(float64(staminaRegen) * toxRegenMult * regenMultiplier)
		user.Character.Stamina += staminaRegen
		if user.Character.Stamina > user.Character.StaminaMax.Value {
			user.Character.Stamina = user.Character.StaminaMax.Value
		}

		// Regenerate Conviction (not affected by combat state)
		convictionRegen := int(float64(user.Character.ConvictionPerRound()) * toxRegenMult * regenMultiplier)
		user.Character.Conviction += convictionRegen
		if user.Character.Conviction > user.Character.ConvictionMax.Value {
			user.Character.Conviction = user.Character.ConvictionMax.Value
		}

		// Regen-based stat progression: smooth chance based on pool depletion.
		// Subtract Chrysalis-enchant pool reservation from the max — a player
		// at their effective cap (unable to heal higher) should NOT count as
		// "depleted" or the reserved fraction becomes a permanent progression
		// farm. See fyttyn vitality exploit (2026-04-16).
		healthMax := user.Character.HealthMax.Value - user.Character.GetPoolReservation("health", user.Character.HealthMax.Value)
		staminaMax := user.Character.StaminaMax.Value - user.Character.GetPoolReservation("stamina", user.Character.StaminaMax.Value)
		convictionMax := user.Character.ConvictionMax.Value - user.Character.GetPoolReservation("conviction", user.Character.ConvictionMax.Value)
		user.Character.OnRegenTick(
			user.Character.Health, healthMax,
			[]string{"vitality", "willpower"}, user.UserId)
		user.Character.OnRegenTick(
			user.Character.Stamina, staminaMax,
			[]string{"strength", "vitality"}, user.UserId)
		user.Character.OnRegenTick(
			user.Character.Conviction, convictionMax,
			[]string{"willpower", "charisma"}, user.UserId)

		// If it has changed, send an update
		vitalsChanged := user.Character.Health-healthStart != 0
		// Always push vitals if the player has active companions
		// so companion bars in the web client stay up to date.
		if !vitalsChanged && len(user.Character.Companions) > 0 {
			for _, comp := range user.Character.Companions {
				if comp.InstanceId > 0 {
					vitalsChanged = true
					break
				}
			}
		}
		if vitalsChanged {

			// Trigger a redraw, but only if the users prompt has changed.
			events.AddToQueue(events.RedrawPrompt{UserId: user.UserId, OnlyIfChanged: true}, 100)

			events.AddToQueue(events.CharacterVitalsChanged{UserId: user.UserId})

		}

	}

	// ── NPC / Mob regen ─────────────────────────────────────────────────────
	for _, mobInstId := range mobs.GetAllMobInstanceIds() {
		mob := mobs.GetInstance(mobInstId)
		if mob == nil || mob.Character.Health < 1 {
			continue
		}

		mobInCombat := mob.Character.IsInCombat()
		mobRegenMult := roomRegenMultiplier(rooms.LoadRoom(mob.Character.RoomId))

		// Health regen (out of combat only, unless heal-spell ConditionRegen)
		if !mobInCombat {
			hpRegen := float64(mob.Character.HealthPerRound())
			// ConditionRegen acts as a multiplier on base regen
			if mob.Character.HasCondition(characters.ConditionRegen) {
				regenMult := mob.Character.GetConditionMagnitude(characters.ConditionRegen)
				if regenMult > 1.0 {
					hpRegen *= regenMult
				}
			}
			hpRegen *= mobRegenMult
			hpAmt := int(math.Floor(hpRegen))
			if hpAmt < 1 {
				hpAmt = 1
			}
			mob.Character.Health += hpAmt
			if mob.Character.Health > mob.Character.HealthMax.Value {
				mob.Character.Health = mob.Character.HealthMax.Value
			}
		} else {
			// In combat: only ConditionRegen applies
			if mob.Character.HasCondition(characters.ConditionRegen) {
				regenMult := mob.Character.GetConditionMagnitude(characters.ConditionRegen)
				hpAmt := int(math.Floor(float64(mob.Character.HealthPerRound()) * regenMult * mobRegenMult))
				if hpAmt < 1 {
					hpAmt = 1
				}
				mob.Character.Health += hpAmt
				if mob.Character.Health > mob.Character.HealthMax.Value {
					mob.Character.Health = mob.Character.HealthMax.Value
				}
			}
		}

		// Stamina regen (1/4 rate in combat)
		spRegen := float64(mob.Character.StaminaPerRound())
		if mobInCombat {
			spRegen = spRegen / 4
		}
		spRegen *= mobRegenMult
		spAmt := int(math.Floor(spRegen))
		if spAmt < 1 {
			spAmt = 1
		}
		mob.Character.Stamina += spAmt
		if mob.Character.Stamina > mob.Character.StaminaMax.Value {
			mob.Character.Stamina = mob.Character.StaminaMax.Value
		}

		// Conviction regen
		cpAmt := int(math.Floor(float64(mob.Character.ConvictionPerRound()) * mobRegenMult))
		if cpAmt < 1 {
			cpAmt = 1
		}
		mob.Character.Conviction += cpAmt
		if mob.Character.Conviction > mob.Character.ConvictionMax.Value {
			mob.Character.Conviction = mob.Character.ConvictionMax.Value
		}

		// Regen-based stat progression for mobs (gated inside OnRegenTick).
		// Subtract equipment pool reservation (companions can inherit
		// chrysalis-enchanted gear).
		mobHealthMax := mob.Character.HealthMax.Value - mob.Character.GetPoolReservation("health", mob.Character.HealthMax.Value)
		mobStaminaMax := mob.Character.StaminaMax.Value - mob.Character.GetPoolReservation("stamina", mob.Character.StaminaMax.Value)
		mobConvictionMax := mob.Character.ConvictionMax.Value - mob.Character.GetPoolReservation("conviction", mob.Character.ConvictionMax.Value)
		mob.Character.OnRegenTick(
			mob.Character.Health, mobHealthMax,
			[]string{"vitality", "willpower"}, 0)
		mob.Character.OnRegenTick(
			mob.Character.Stamina, mobStaminaMax,
			[]string{"strength", "vitality"}, 0)
		mob.Character.OnRegenTick(
			mob.Character.Conviction, mobConvictionMax,
			[]string{"willpower", "charisma"}, 0)

		// Phase 25.1: Apply poison DoT damage to mobs
		if mob.Character.HasCondition(characters.ConditionPoisoned) {
			poisonDmg := int(mob.Character.GetConditionMagnitude(characters.ConditionPoisoned))
			if poisonDmg < 1 {
				poisonDmg = 1
			}
			mob.Character.Health -= poisonDmg
			if mob.Character.Health < 1 {
				mob.Character.Health = 0
			}
		}

		// Stage 42.7: Apply bleed DoT damage to mobs
		if mob.Character.HasCondition(characters.ConditionBleeding) {
			bleedDmg := int(mob.Character.GetConditionMagnitude(characters.ConditionBleeding))
			if bleedDmg < 1 {
				bleedDmg = 1
			}
			mob.Character.Health -= bleedDmg
			if mob.Character.Health < 1 {
				mob.Character.Health = 0
			}
		}
	}

	return events.Continue
}

// regenMultFromSpecs returns the regen multiplier resulting from a set
// of mutator specs. Specs with RegenMultiplier > 0 contribute;
// multiple specs stack multiplicatively. Used by roomRegenMultiplier
// after iterating a room's active mutators.
func regenMultFromSpecs(specs ...*mutators.MutatorSpec) float64 {
	mult := 1.0
	for _, spec := range specs {
		if spec == nil || spec.RegenMultiplier <= 0 {
			continue
		}
		mult *= spec.RegenMultiplier
	}
	return mult
}

// roomRegenMultiplier returns the regen multiplier applied to HP/SP/CP
// regen for any actor (player or mob) in the given room. 1.0 means no
// bonus. Sourced from the room's active mutators — any mutator with
// RegenMultiplier > 0 contributes; multiple stack multiplicatively.
// Replaces the Stage 2-era hardcoded room-id switch.
func roomRegenMultiplier(room *rooms.Room) float64 {
	if room == nil {
		return 1.0
	}
	var specs []*mutators.MutatorSpec
	for mut := range room.ActiveMutators {
		specs = append(specs, mut.GetSpec())
	}
	return regenMultFromSpecs(specs...)
}
