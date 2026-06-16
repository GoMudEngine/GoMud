package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Drink(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Chunk 4e: can't drink while grappled — both hands committed.
	if user.Character.Position != nil && user.Character.Position.IsGrappling() {
		user.SendText(messaging.CategorySystem, `<ansi fg="red">Your hands are committed to the grapple — you can't reach for that.</ansi>`)
		return true, nil
	}

	// Search bandolier first (oldest first), then backpack
	fromBandolier := false
	matchItem, found := user.Character.FindInPotions(rest)
	if found {
		fromBandolier = true
	} else {
		matchItem, found = user.Character.FindInBackpack(rest)
	}

	if !found {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You don't have a "%s" to drink.`, rest))
		return true, nil
	}

	itemSpec := matchItem.GetSpec()

	if itemSpec.Subtype != items.Drinkable {
		user.SendText(messaging.CategorySystem, 
			fmt.Sprintf(`You can't drink <ansi fg="itemname">%s</ansi>.`, matchItem.DisplayName()),
		)
		return true, nil
	}

	// Compute aging phase if the potion has aging data
	var phase items.AgingPhase
	var potencyMult float64 = 1.0
	hasAging := itemSpec.Aging.HasAging() && matchItem.CraftedRound > 0

	if hasAging {
		elapsed := util.GetRoundCount() - matchItem.CraftedRound
		bottleMult := matchItem.BottleMultiplier
		if bottleMult <= 0 {
			bottleMult = itemSpec.BottleAgingMultiplier
		}
		effSpeed := items.CalcEffectiveAgingSpeed(bottleMult, matchItem.CraftSkill)
		phase, potencyMult = items.GetAgingPhase(elapsed, itemSpec.Aging, effSpeed)
	}

	// Handle spoiled potions
	if hasAging && phase == items.PhaseSpoiled {
		// Spoiled potions apply 3x toxicity
		spoiledTox := float64(itemSpec.Toxicity) * 3.0
		user.Character.Toxicity += spoiledTox

		user.Character.CancelBuffsWithFlag(buffs.Hidden)

		// Consume the item
		if fromBandolier {
			user.Character.UseItemFromPotions(matchItem)
		} else {
			user.Character.UseItem(matchItem)
		}

		user.SendText(messaging.CategorySystem, fmt.Sprintf(
			`You drink the <ansi fg="itemname">%s</ansi>...`, matchItem.DisplayName()))
		user.SendText(messaging.CategorySystem, 
			`<ansi fg="red">The potion has gone bad! You retch as the foul liquid burns your throat.</ansi>`)
		room.SendTextVisual(messaging.CategoryMobEmote, fmt.Sprintf(
			`<ansi fg="username">%s</ansi> drinks something and immediately gags.`,
			user.Character.Name), user.UserId)

		// Apply nausea debuff (buff 75)
		user.Character.AddBuffScaled(75, 1.0)

		// Recipe discovery chance: 10% + (alchemySkill * 0.5)%
		alchSkill := user.Character.GetSkillLevel(skills.Alchemy)
		discoveryChance := 10.0 + float64(alchSkill)*0.5
		if float64(util.Rand(100)) < discoveryChance {
			user.SendText(messaging.CategorySystem, 
				`<ansi fg="yellow">The foul taste teaches you something about how the ingredients interact...</ansi>`)
		}

		return true, nil
	}

	// Check toxicity before consuming
	if itemSpec.Toxicity > 0 {
		toxCost := float64(itemSpec.Toxicity)
		if user.Character.Toxicity+toxCost > user.Character.GetToxicityMax() {
			user.SendText(messaging.CategorySystem, 
				`<ansi fg="red">Your body rejects the potion — too much toxicity.</ansi>`)
			return true, nil
		}
	}

	user.Character.CancelBuffsWithFlag(buffs.Hidden)

	// Consume the item
	if fromBandolier {
		user.Character.UseItemFromPotions(matchItem)
	} else {
		user.Character.UseItem(matchItem)
	}

	// Apply toxicity
	if itemSpec.Toxicity > 0 {
		user.Character.Toxicity += float64(itemSpec.Toxicity)
	}

	// Quest engine: command notification — a successful drink advances
	// "drink a potion" quest steps (e.g. the Spoke C alchemy cert).
	questBridge := questengine.NewGameBridge(user, room.RoomId)
	questengine.GetEngine().Notify("command", questengine.EventDetails{
		UserId:  user.UserId,
		RoomId:  room.RoomId,
		Command: "drink",
	}, questBridge, questBridge)

	user.SendText(messaging.CategorySystem, fmt.Sprintf(
		`You drink the <ansi fg="itemname">%s</ansi>.`, matchItem.DisplayName()))
	room.SendTextVisual(messaging.CategoryMobEmote, fmt.Sprintf(
		`<ansi fg="username">%s</ansi> drinks <ansi fg="itemname">%s</ansi>.`,
		user.Character.Name, matchItem.DisplayName()), user.UserId)

	// Aging quality message
	if hasAging {
		switch phase {
		case items.PhaseFresh:
			user.SendText(messaging.CategorySystem, `The potion is freshly brewed — it should do the job.`)
		case items.PhaseFermented:
			user.SendText(messaging.CategorySystem, `The potion has fermented nicely — you feel it working stronger than expected.`)
		case items.PhasePeak:
			user.SendText(messaging.CategorySystem, `<ansi fg="green">The potion is at its peak — you feel its full potency.</ansi>`)
		case items.PhaseDeclining:
			user.SendText(messaging.CategorySystem, `The potion tastes a bit stale — its effects are diminished.`)
		}
	}

	// Calculate final duration multiplier:
	// potencyMult (from aging phase) * craftSkill scaling
	durationMult := potencyMult
	if matchItem.CraftSkill > 0 {
		durationMult *= 1.0 + float64(matchItem.CraftSkill)/100.0
	}

	// Apply buffs with scaled duration
	for _, buffId := range itemSpec.BuffIds {
		if durationMult != 1.0 {
			user.Character.AddBuffScaled(buffId, durationMult)
		} else {
			user.AddBuff(buffId, `drink`)
		}
		// Compute tick snapshot for config-driven buffs (no stat scaling for potions)
		if buffSpec := buffs.GetBuffSpec(buffId); buffSpec != nil && buffSpec.TickPool != "" {
			var maxPool int
			switch buffSpec.TickPool {
			case "health":
				maxPool = user.Character.HealthMax.Value
			case "stamina":
				maxPool = user.Character.StaminaMax.Value
			case "conviction":
				maxPool = user.Character.ConvictionMax.Value
			}
			tickAmt := buffs.ComputeTickAmount(maxPool, buffSpec.TickPercent, buffSpec.TickVariance, buffSpec.TickMin, 1.0)
			user.Character.Buffs.SetTickAmount(buffId, tickAmt)
		}
	}

	return true, nil
}
