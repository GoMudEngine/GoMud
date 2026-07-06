package usercommands

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// maybeInterruptOnThrow cancels a mob's in-progress fold-cast if the thrown
// item id is a configured boss-interrupt disruptor
// (Balance.BossInterruptItemIds) AND the mob is currently casting
// (Character.IsCasting()). Generic (non-allowlisted) thrown items never
// interrupt a cast, even on a hit. Reuses the shared InterruptTargetCast
// primitive (conviction refund + TriggerCastCancel) rather than reimplementing
// cast cancellation here. Returns true if a cast was actually interrupted.
func maybeInterruptOnThrow(mob *mobs.Mob, thrownItemId int, by state.ActorRef) bool {
	if mob == nil {
		return false
	}
	if !configs.GetBalanceConfig().IsBossInterruptItem(thrownItemId) {
		return false
	}
	if !mob.Character.IsCasting() {
		return false
	}
	return actions.InterruptTargetCast(&mob.Character, by)
}

func Throw(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if rest == "" {
		user.SendText(messaging.CategorySystem, "Throw what? Specify a throwable item.")
		return true, nil
	}

	// Must be in combat
	if !user.Character.IsInCombat() {
		user.SendText(messaging.CategorySystem, "You must be in combat to throw something!")
		return true, nil
	}

	// Special-move cooldown
	cfg := configs.GetBalanceConfig()
	if !user.Character.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		user.SendText(messaging.CategorySystem, "You need a moment to recover before attempting another special move.")
		return true, nil
	}

	// Find throwable in backpack or bandolier
	matchItem, found := user.Character.FindInBackpack(rest)
	if !found {
		matchItem, found = user.Character.FindInPotions(rest)
	}
	if !found {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You don't have a "%s" to throw.`, rest))
		return true, nil
	}

	spec := matchItem.GetSpec()
	if spec.Subtype != items.Throwable {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="itemname">%s</ansi> isn't something you can throw.`, matchItem.DisplayName()))
		return true, nil
	}

	// Check if grenade has spoiled — auto-detonates in hand
	if spec.Aging.HasAging() && matchItem.CraftedRound > 0 {
		currentRound := util.GetRoundCount()
		var elapsed uint64
		if currentRound >= matchItem.CraftedRound {
			elapsed = currentRound - matchItem.CraftedRound
		}
		effSpeed := items.CalcEffectiveAgingSpeed(matchItem.BottleMultiplier, matchItem.CraftSkill)
		phase, _ := items.GetAgingPhase(elapsed, spec.Aging, effSpeed)
		if phase == items.PhaseSpoiled {
			user.Character.UseItem(matchItem)
			user.SendText(messaging.CategorySystem, fmt.Sprintf(
				`<ansi fg="red-bold">You pull out the <ansi fg="itemname">%s</ansi> and it crumbles apart in your hand -- it's gone bad!</ansi>`,
				matchItem.DisplayName()))
			return true, nil
		}
	}

	// Consume the item
	usesLeft := user.Character.UseItem(matchItem)
	if usesLeft < 1 {
		events.AddToQueue(events.ItemOwnership{
			UserId: user.UserId,
			Item:   matchItem,
			Gained: false,
		})
	}

	// Quest engine: command notification — a successful throw advances
	// "throw a grenade" quest steps (e.g. the Spoke C grenade lesson).
	questBridge := questengine.NewGameBridge(user, room.RoomId)
	questengine.GetEngine().Notify("command", questengine.EventDetails{
		UserId:  user.UserId,
		RoomId:  room.RoomId,
		Command: "throw",
	}, questBridge, questBridge)

	// AoE resolution: opposed roll per hostile mob in room
	skillWeight := float64(cfg.SkillWeight)
	skullduggery := user.Character.GetSkillLevel(skills.Skullduggery)
	dexterity := user.Character.GetEffectiveDexterity()

	attackerScore := float64(dexterity) + float64(skullduggery)*skillWeight

	user.SendText(messaging.CategorySystem, fmt.Sprintf(
		`<ansi fg="yellow-bold">You hurl the <ansi fg="itemname">%s</ansi> into the fray!</ansi>`,
		matchItem.DisplayName()))
	room.SendTextVisual(messaging.CategoryHitRanged, fmt.Sprintf(
		`<ansi fg="yellow-bold"><ansi fg="username">%s</ansi> hurls a <ansi fg="itemname">%s</ansi> into the fray!</ansi>`,
		user.Character.Name, matchItem.DisplayName()),
		user.UserId)

	hasDamage := spec.DamageMultiplier > 0
	hasBuffs := len(spec.BuffIds) > 0
	hitCount := 0
	fumbled := false

	for _, mobInstId := range room.GetMobs(rooms.FindAll) {
		mob := mobs.GetInstance(mobInstId)
		if mob == nil {
			continue
		}

		// Skip all companions
		if mob.Character.IsCharmed() {
			continue
		}

		// Skip non-combatants and player-attack-immune mobs
		if mob.IsNonCombatant() || mob.PlayerAttackImmune {
			continue
		}

		defenderScore := float64(mob.Character.GetEffectiveDexterity()) +
			float64(mob.Character.GetEffectivePerception())*skillWeight*0.5

		attackSuccess, _, atkRoll, _ := dice.OpposedRollStat(attackerScore, defenderScore)

		// Fumble check: effect hits thrower instead
		if atkRoll.ZScore <= -2.0 {
			fumbled = true
			user.SendText(messaging.CategorySystem, `<ansi fg="red-bold">Your throw goes horribly wrong — the projectile detonates in your hand!</ansi>`)
			room.SendTextVisual(messaging.CategoryHitRanged, fmt.Sprintf(
				`<ansi fg="red"><ansi fg="username">%s</ansi>'s throw backfires spectacularly!</ansi>`,
				user.Character.Name), user.UserId)

			// Apply effects to thrower
			if hasDamage {
				rawDmg := combat.CalcRawDamage(dexterity, skullduggery, spec.DamageMultiplier, combat.ChannelPhysical)
				dmg := int(math.Round(rawDmg))
				if dmg < 1 {
					dmg = 1
				}
				user.Character.Health -= dmg
				if user.Character.Health < 0 {
					user.Character.Health = 0
				}
				dmgDesc := combat.GetDamageDescription(dmg, user.Character.HealthMax.Value)
				user.SendText(messaging.CategorySystem, fmt.Sprintf(
					`<ansi fg="red">The explosion sears you! (%s)</ansi>`, dmgDesc))
			}
			if hasBuffs {
				for _, buffId := range spec.BuffIds {
					user.AddBuff(buffId, `grenade-fumble`)
				}
			}
			break // Fumble ends the AoE loop
		}

		// Boss-interrupt: a configured disruptor thrown at a mid-fold-cast mob
		// cancels the cast whether or not the throw wins its damage roll — the
		// interrupt is the disruptor's purpose, not a side effect of a hit. Fires
		// before the attack-success gate so a tanky boss can't simply dodge the
		// interrupt. (Fumbles break above, so a botched throw still can't cancel.)
		maybeInterruptOnThrow(mob, matchItem.ItemId, state.ActorRef{UserId: user.UserId})

		if !attackSuccess {
			continue
		}

		hitCount++

		// Apply effects to mob
		if hasDamage {
			rawDmg := combat.CalcRawDamage(dexterity, skullduggery, spec.DamageMultiplier, combat.ChannelPhysical)
			mitPct := mob.Character.GetPhysicalMitigation()
			mitCap := combat.MitigationCap(combat.ChannelPhysical)
			mitigated := combat.ApplyMitigation(rawDmg, mitPct, mitCap)
			finalRoll := dice.RollStat(mitigated)
			dmg := int(math.Round(finalRoll.Value))
			if dmg < 1 {
				dmg = 1
			}
			mob.Character.Health -= dmg
			if mob.Character.Health < 0 {
				mob.Character.Health = 0
			}
			dmgDesc := combat.GetDamageDescription(dmg, mob.Character.HealthMax.Value)
			user.SendText(messaging.CategorySystem, fmt.Sprintf(
				`The explosion catches <ansi fg="mobname">%s</ansi>! (<ansi fg="damage">%s</ansi>)`,
				mob.Character.Name, dmgDesc))
		}

		if hasBuffs {
			for _, buffId := range spec.BuffIds {
				mob.AddBuff(buffId, `grenade`)
			}
			user.SendText(messaging.CategorySystem, fmt.Sprintf(
				`<ansi fg="mobname">%s</ansi> is caught in the blast!`,
				mob.Character.Name))
		}

		combat.RecordSpecialMove(combat.User, combat.Mob, "throw",
			true, 0, user.Character, &mob.Character, util.GetRoundCount())
	}

	if !fumbled {
		if hitCount == 0 {
			user.SendText(messaging.CategorySystem, "The projectile misses everything and shatters harmlessly.")
		} else if hitCount == 1 {
			user.SendText(messaging.CategorySystem, fmt.Sprintf(
				`<ansi fg="green">Your throw strikes true!</ansi>`))
		} else {
			user.SendText(messaging.CategorySystem, fmt.Sprintf(
				`<ansi fg="green">Your throw catches multiple targets!</ansi>`))
		}
	}

	// Progress skullduggery skill (also advances dexterity as primary stat)
	user.Character.OnSkillUse(string(skills.Skullduggery), user.UserId)

	if user.Character.Aggro != nil {
		user.Character.Aggro.RoundsWaiting = 1
	}

	return true, nil
}
