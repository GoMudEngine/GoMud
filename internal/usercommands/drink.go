package usercommands

import (
	"fmt"
	"math/rand"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// bloomWaferItemId is the item ID for the Bloom Wafer (40108).
// The wafer's effect (Communion buff, addiction tick, mutation roll) is
// handled as a special case in Drink rather than through the generic buffids
// path, because it also needs to stamp BloomLastDoseRound and call
// BloomAdvanceMutation.
const bloomWaferItemId = 40108

// ysoldesPurgeItemId is the item ID for Ysolde's Purge (40109).
// The purge carries a heavy toxicity load (45) and buff 93 (Bloom Detox)
// through the normal drink path. Its special-case here drives the addiction
// step-down. It also bypasses the toxicity pre-check — addicts presenting
// for detox are expected to already have elevated toxicity, and the flood is
// the mechanism, not a mistake.
const ysoldesPurgeItemId = 40109

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
		user.Character.AddToxicity(spoiledTox)

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

	// Check toxicity before consuming.
	// Exception: Ysolde's Purge bypasses this cap -- the toxicity flood is
	// intentional and the detox must be drinkable even at high toxicity.
	if itemSpec.Toxicity > 0 && itemSpec.ItemId != ysoldesPurgeItemId {
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
		user.Character.AddToxicity(float64(itemSpec.Toxicity))
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

	// ── Ysolde's Purge special-case ──────────────────────────────────────────
	// Toxicity (45) and the detox debuff (buff 93) are applied by the normal
	// drink path above. Here we drive the addiction step-down -- the brutal-
	// fast path to clean.
	if itemSpec.ItemId == ysoldesPurgeItemId {
		user.Character.AddBloomAddiction(-5)
		user.SendText(messaging.CategoryWarning,
			`The purge takes hold -- your body convulses as it expels the `+
				`Bloom. It is violent, and it is fast.`)
	}

	// ── Bloom Wafer special-case ──────────────────────────────────────────────
	// The wafer has no buffids in its YAML; all Bloom effects are wired here.
	// Toxicity (35) was already applied by the normal path above — don't
	// apply it again. The order relative to the buff loop above doesn't matter
	// since the loop is empty for this item.
	if itemSpec.ItemId == bloomWaferItemId {
		bal := configs.GetBalanceConfig()

		// Communion high. Buff 90's YAML baseline is 30 rounds; scale it by the
		// BloomCommunionRounds knob so config actually tunes the duration.
		communionMult := float64(bal.BloomCommunionRounds) / 30.0
		if communionMult <= 0 {
			communionMult = 1.0
		}
		_ = user.Character.AddBuffScaled(90, communionMult)

		// Tick addiction counter.
		user.Character.AddBloomAddiction(int(bal.BloomAddictionPerDose))

		// Stamp the dose round for withdrawal / decay timing.
		user.Character.BloomLastDoseRound = util.GetRoundCount()

		// Mutation acceleration. First roll the (small) BloomNewMutationChance to
		// push a brand-new change even if the user already has mutations — Bloom's
		// "occasionally something wholly new" variety. Otherwise roll the (larger)
		// BloomMutationAdvanceChance to deepen the strongest existing mutation
		// (which falls through to seeding when the user has none / all are capped).
		var mutId string
		if rand.Float64() < float64(bal.BloomNewMutationChance) {
			mutId, _ = user.Character.BloomSeedNewMutation(nil)
		} else if rand.Float64() < float64(bal.BloomMutationAdvanceChance) {
			mutId, _ = user.Character.BloomAdvanceMutation(nil)
		}
		if mutId != "" {
			user.SendText(messaging.CategoryWarning,
				`Something under your skin shifts and settles differently.`)
		}

		// Euphoric onset message — replaces the generic "you drink" that was
		// already sent above. Sent last so it reads as the climax of the
		// consume sequence.
		user.SendText(messaging.CategoryWarning,
			`The wafer dissolves to nothing on your tongue and the world goes `+
				`warm and wide — communion.`)
	}

	return true, nil
}
