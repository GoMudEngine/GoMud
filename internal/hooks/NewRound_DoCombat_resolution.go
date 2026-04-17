package hooks

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// File: NewRound_DoCombat_resolution.go
//
// Phase helpers for the per-combatant round handlers (handlePlayerVsMob
// and handleMobVsPlayer). Extracted during 1.2a god-function refactor.

// applyCombatDamageBonuses_PvM applies the Conviction Surge, Adrenaline
// Surge, return damage, and lifesteal stages for a player-vs-mob swing.
// Mutation order matches the original inline block: bonus damage is added
// before return damage and lifesteal are computed.
func applyCombatDamageBonuses_PvM(roundResult *combat.AttackResult, user *users.UserRecord, defMob *mobs.Mob, uRoom *rooms.Room, defRoom *rooms.Room) {
	// Phase 25.3: Conviction Surge buff
	if roundResult.Hit && roundResult.DamageToTarget > 0 && user.Character.HasBuffFlag(buffs.DamageBonus) {
		bonusDmg := int(math.Round(float64(roundResult.DamageToTarget) * 0.15))
		if bonusDmg < 1 {
			bonusDmg = 1
		}
		defMob.Character.Health -= bonusDmg
		roundResult.DamageToTarget += bonusDmg
	}

	// Stage 12.2: Adrenaline Surge
	if roundResult.Hit && roundResult.DamageToTarget > 0 {
		if mutations.IsAdrenalSurgeActive(user.Character.Mutations, user.Character.Health, user.Character.HealthMax.Value) {
			if surgeBonus := mutations.GetAdrenalSurgeBonus(user.Character.Mutations); surgeBonus > 0 {
				bonusDmg := int(math.Round(float64(roundResult.DamageToTarget) * surgeBonus))
				if bonusDmg < 1 {
					bonusDmg = 1
				}
				defMob.Character.Health -= bonusDmg
				roundResult.DamageToTarget += bonusDmg
			}
		}
	}

	// Return damage — fire elementals, battlerager armor, etc.
	// Direct HP reduction; does NOT trigger another combat round (no recursion risk).
	if roundResult.Hit && roundResult.DamageToTarget > 0 {
		returnPct := defMob.Character.StatMod("return_damage")
		if sp := species.GetSpecies(defMob.Character.SpeciesId); sp != nil {
			returnPct += sp.ReturnDamage
		}
		if returnPct > 0 {
			returnDmg := int(float64(roundResult.DamageToTarget) * float64(returnPct) / 100.0)
			if returnDmg > 0 {
				user.Character.Health -= returnDmg
				dmgDesc := combat.GetDamageDescription(returnDmg, user.Character.HealthMax.Value)
				defMobName := mobDisplayName(defMob, defRoom, user.UserId)
				sendVisualRoomText(uRoom, fmt.Sprintf(
					`<ansi fg="red">%s recoils from striking %s! (%s)</ansi>`,
					user.Character.Name, defMobName, dmgDesc), user.UserId)
				user.SendText(fmt.Sprintf(
					`<ansi fg="red">You recoil from striking %s! (%s)</ansi>`,
					defMobName, dmgDesc))
			}
		}
	}

	// Lifesteal — Hungering Touch enchantment heals attacker on hit
	if roundResult.Hit && roundResult.DamageToTarget > 0 {
		lifestealPct := user.Character.StatMod("lifesteal_pct")
		if lifestealPct > 0 {
			healAmt := int(float64(roundResult.DamageToTarget) * float64(lifestealPct) / 100.0)
			if healAmt > 0 {
				user.Character.Health += healAmt
				if user.Character.Health > user.Character.HealthMax.Value {
					user.Character.Health = user.Character.HealthMax.Value
				}
				healDesc := combat.GetHealDescription(healAmt, user.Character.HealthMax.Value)
				user.SendText(fmt.Sprintf(
					`<ansi fg="green">Your weapon feeds on the blow! (%s)</ansi>`,
					healDesc))
			}
		}
	}
}

// applyCombatDamageBonuses_MvP applies the Conviction Surge, Adrenaline
// Surge, return damage, and lifesteal stages for a mob-vs-player swing.
// Minor Shield reduction is NOT handled here — it remains inline in the
// parent because ConditionShield is player-defender-only.
func applyCombatDamageBonuses_MvP(roundResult *combat.AttackResult, mob *mobs.Mob, defUser *users.UserRecord, mobRoom *rooms.Room) {
	// Conviction Surge buff: +15% damage bonus (mob attacker)
	if roundResult.Hit && roundResult.DamageToTarget > 0 && mob.Character.HasBuffFlag(buffs.DamageBonus) {
		bonusDmg := int(math.Round(float64(roundResult.DamageToTarget) * 0.15))
		if bonusDmg < 1 {
			bonusDmg = 1
		}
		defUser.Character.Health -= bonusDmg
		roundResult.DamageToTarget += bonusDmg
	}

	// Stage 12.2: Adrenaline Surge
	if roundResult.Hit && roundResult.DamageToTarget > 0 {
		if mutations.IsAdrenalSurgeActive(mob.Character.Mutations, mob.Character.Health, mob.Character.HealthMax.Value) {
			if surgeBonus := mutations.GetAdrenalSurgeBonus(mob.Character.Mutations); surgeBonus > 0 {
				bonusDmg := int(math.Round(float64(roundResult.DamageToTarget) * surgeBonus))
				if bonusDmg < 1 {
					bonusDmg = 1
				}
				defUser.Character.Health -= bonusDmg
				roundResult.DamageToTarget += bonusDmg
			}
		}
	}

	// Return damage — fire elementals, battlerager armor, etc.
	// Direct HP reduction; does NOT trigger another combat round (no recursion risk).
	if roundResult.Hit && roundResult.DamageToTarget > 0 {
		returnPct := defUser.Character.StatMod("return_damage")
		if sp := species.GetSpecies(defUser.Character.SpeciesId); sp != nil {
			returnPct += sp.ReturnDamage
		}
		if returnPct > 0 {
			returnDmg := int(float64(roundResult.DamageToTarget) * float64(returnPct) / 100.0)
			if returnDmg > 0 {
				mob.Character.Health -= returnDmg
				dmgDesc := combat.GetDamageDescription(returnDmg, mob.Character.HealthMax.Value)
				mvpMobName := mobDisplayName(mob, mobRoom, defUser.UserId)
				defUser.SendText(fmt.Sprintf(
					`<ansi fg="red">%s recoils from striking you! (%s)</ansi>`,
					mvpMobName, dmgDesc))
				sendVisualRoomText(mobRoom, fmt.Sprintf(
					`<ansi fg="red">%s recoils from striking %s! (%s)</ansi>`,
					mvpMobName, defUser.Character.Name, dmgDesc), defUser.UserId)
			}
		}
	}

	// Lifesteal — mob enchantment heals attacker on hit
	if roundResult.Hit && roundResult.DamageToTarget > 0 {
		lifestealPct := mob.Character.StatMod("lifesteal_pct")
		if lifestealPct > 0 {
			healAmt := int(float64(roundResult.DamageToTarget) * float64(lifestealPct) / 100.0)
			if healAmt > 0 {
				mob.Character.Health += healAmt
				if mob.Character.Health > mob.Character.HealthMax.Value {
					mob.Character.Health = mob.Character.HealthMax.Value
				}
			}
		}
	}
}
