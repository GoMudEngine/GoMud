package hooks

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
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
	testingArenaEntrance := 200 // Fast regen room for testing (Arena Entrance)

	onlineIds := users.GetOnlineUserIds()
	for _, userId := range onlineIds {
		user := users.GetByUserId(userId)

		if user.Character.RoomId == deathRecoveryRoomId {
			continue
		}

		// Fast regen in testing area
		regenMultiplier := 1.0
		if user.Character.RoomId == testingArenaEntrance {
			regenMultiplier = 10.0 // 10x faster regen for testing
		}

		// 5x regen in Sanctum Basin tutorial zone (rooms 101–120)
		if user.Character.RoomId >= 101 && user.Character.RoomId <= 120 {
			regenMultiplier = 5.0
		}

		// 5x regen in Thornwall Temple (room 468)
		if user.Character.RoomId == 468 {
			regenMultiplier = 5.0
		}

		inCombat := user.Character.Aggro != nil
		healthStart := user.Character.Health

		// ── Downed state: health bleedout only ───────────────────────
		// Only health causes death. Stamina and conviction bottom out at
		// 0 and recover via normal regen — the resource depletion penalties
		// already provide meaningful gameplay consequences for low pools.
		if user.Character.Health < 1 {

			if user.Character.Health <= -10 {
				user.Command(`suicide`)
				continue
			}
			user.Character.Health--
			user.SendText(`<ansi fg="red">you are bleeding out!</ansi>`)
			if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
				room.SendText(fmt.Sprintf(
					`<ansi fg="username">%s</ansi> is <ansi fg="red">bleeding out</ansi>!`,
					user.Character.Name), user.UserId)
			}

			// If it has changed, send an update
			if user.Character.Health-healthStart != 0 {
				events.AddToQueue(events.RedrawPrompt{UserId: user.UserId, OnlyIfChanged: true}, 100)
				events.AddToQueue(events.CharacterVitalsChanged{UserId: user.UserId})
			}

			continue // Skip all regen while health is depleted
		}

		// ── Not downed: reset downed counter, normal regen ──────────
		user.Character.DownedRounds = 0

		// Regeneration (only heal health if health > 0)
		if user.Character.Health > 0 {

			if !inCombat {
				// Out of combat: base %-regen, then mutation multipliers, then room multiplier
				healthRegen := float64(user.Character.HealthPerRound())

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

			} else {
				// In combat: no base regen, but ConditionRegen (heal spell) still applies
				if user.Character.HasCondition(characters.ConditionRegen) {
					regenMult := user.Character.GetConditionMagnitude(characters.ConditionRegen)
					healAmt := int(math.Floor(float64(user.Character.HealthPerRound()) * regenMult * regenMultiplier))
					if healAmt < 1 {
						healAmt = 1
					}
					user.Character.Heal(healAmt)
				}
			}

			// Phase 24.5: Apply poison DoT damage
			if user.Character.HasCondition(characters.ConditionPoisoned) {
				poisonDmg := int(user.Character.GetConditionMagnitude(characters.ConditionPoisoned))
				if poisonDmg < 1 {
					poisonDmg = 1
				}
				user.Character.Health -= poisonDmg
				if user.Character.Health < -10 {
					user.Character.Health = -10
				}
				user.SendText(`<ansi fg="green">The poison burns through your veins!</ansi>`)
			}

			// Stage 42.7: Apply bleed DoT damage
			if user.Character.HasCondition(characters.ConditionBleeding) {
				bleedDmg := int(user.Character.GetConditionMagnitude(characters.ConditionBleeding))
				if bleedDmg < 1 {
					bleedDmg = 1
				}
				user.Character.Health -= bleedDmg
				if user.Character.Health < -10 {
					user.Character.Health = -10
				}
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
		staminaRegen = int(float64(staminaRegen) * regenMultiplier)
		user.Character.Stamina += staminaRegen
		if user.Character.Stamina > user.Character.StaminaMax.Value {
			user.Character.Stamina = user.Character.StaminaMax.Value
		}

		// Regenerate Conviction (not affected by combat state)
		convictionRegen := int(float64(user.Character.ConvictionPerRound()) * regenMultiplier)
		user.Character.Conviction += convictionRegen
		if user.Character.Conviction > user.Character.ConvictionMax.Value {
			user.Character.Conviction = user.Character.ConvictionMax.Value
		}

		// Regen-based stat progression: smooth chance based on pool depletion
		user.Character.OnRegenTick(
			user.Character.Health, user.Character.HealthMax.Value,
			[]string{"vitality", "willpower"}, user.UserId)
		user.Character.OnRegenTick(
			user.Character.Stamina, user.Character.StaminaMax.Value,
			[]string{"strength", "vitality"}, user.UserId)
		user.Character.OnRegenTick(
			user.Character.Conviction, user.Character.ConvictionMax.Value,
			[]string{"willpower", "charisma"}, user.UserId)

		// If it has changed, send an update
		if user.Character.Health-healthStart != 0 {

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

		mobInCombat := mob.Character.Aggro != nil

		// Health regen (out of combat only, unless heal-spell ConditionRegen)
		if !mobInCombat {
			hpRegen := mob.Character.HealthPerRound()
			// ConditionRegen acts as a multiplier on base regen
			if mob.Character.HasCondition(characters.ConditionRegen) {
				regenMult := mob.Character.GetConditionMagnitude(characters.ConditionRegen)
				if regenMult > 1.0 {
					hpRegen = int(float64(hpRegen) * regenMult)
				}
			}
			mob.Character.Health += hpRegen
			if mob.Character.Health > mob.Character.HealthMax.Value {
				mob.Character.Health = mob.Character.HealthMax.Value
			}
		} else {
			// In combat: only ConditionRegen applies
			if mob.Character.HasCondition(characters.ConditionRegen) {
				regenMult := mob.Character.GetConditionMagnitude(characters.ConditionRegen)
				hpRegen := int(float64(mob.Character.HealthPerRound()) * regenMult)
				if hpRegen < 1 {
					hpRegen = 1
				}
				mob.Character.Health += hpRegen
				if mob.Character.Health > mob.Character.HealthMax.Value {
					mob.Character.Health = mob.Character.HealthMax.Value
				}
			}
		}

		// Stamina regen (1/4 rate in combat)
		spRegen := mob.Character.StaminaPerRound()
		if mobInCombat {
			spRegen = spRegen / 4
			if spRegen < 1 {
				spRegen = 1
			}
		}
		mob.Character.Stamina += spRegen
		if mob.Character.Stamina > mob.Character.StaminaMax.Value {
			mob.Character.Stamina = mob.Character.StaminaMax.Value
		}

		// Conviction regen
		cpRegen := mob.Character.ConvictionPerRound()
		mob.Character.Conviction += cpRegen
		if mob.Character.Conviction > mob.Character.ConvictionMax.Value {
			mob.Character.Conviction = mob.Character.ConvictionMax.Value
		}

		// Regen-based stat progression for mobs (gated inside OnRegenTick)
		mob.Character.OnRegenTick(
			mob.Character.Health, mob.Character.HealthMax.Value,
			[]string{"vitality", "willpower"}, 0)
		mob.Character.OnRegenTick(
			mob.Character.Stamina, mob.Character.StaminaMax.Value,
			[]string{"strength", "vitality"}, 0)
		mob.Character.OnRegenTick(
			mob.Character.Conviction, mob.Character.ConvictionMax.Value,
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
