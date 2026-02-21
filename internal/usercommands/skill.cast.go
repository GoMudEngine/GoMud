package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Cast initiates fold-based spellcasting (Stage 11.2).
// Each round the combat hook adds folds until FoldsNeeded is reached,
// then Stage 11.4 resolves the actual spell effect.
func Cast(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// 1. Spellcasting skill required
	skillLevel := user.Character.GetSkillLevel(skills.Spellcasting)
	if skillLevel == 0 {
		user.SendText(`<ansi fg="red">You have no spellcasting skill.</ansi>`)
		return true, nil
	}

	// 2. Parse spell name and optional target from rest
	rest = strings.TrimSpace(rest)
	if rest == `` {
		user.SendText(`<ansi fg="red">Cast what? (Usage: cast <spell> [target])</ansi>`)
		return true, nil
	}

	parts := strings.SplitN(rest, ` `, 2)
	spellName := strings.ToLower(parts[0])
	targetName := ``
	if len(parts) > 1 {
		targetName = strings.TrimSpace(parts[1])
	}

	// 3. Verify spell exists and user knows it
	// Try spell ID first (e.g. "mm"), then fall back to display-name prefix match (e.g. "magic missile")
	spellInfo := spells.GetSpell(spellName)
	if spellInfo == nil {
		spellInfo = spells.FindSpellByName(spellName)
	}
	if spellInfo == nil {
		user.SendText(fmt.Sprintf(`<ansi fg="red">No spell found for "%s". Use the spell ID (e.g. <ansi fg="cyan-bold">mm</ansi>, <ansi fg="cyan-bold">heal</ansi>). Type <ansi fg="cyan-bold">spells</ansi> to list what you know.</ansi>`, spellName))
		return true, nil
	}
	if !user.Character.HasSpell(spellInfo.SpellId) {
		user.SendText(fmt.Sprintf(`<ansi fg="red">You haven't learned the spell "%s".</ansi>`, spellInfo.Name))
		return true, nil
	}

	// 4. Already casting?
	if user.Character.CastingState != nil {
		cs := user.Character.CastingState
		user.SendText(fmt.Sprintf(
			`<ansi fg="cyan">You are already casting %s (%d/%d folds). Type <ansi fg="cyan-bold">cancel</ansi> to stop.</ansi>`,
			cs.SpellId, cs.FoldsAccumulated, cs.FoldsNeeded))
		return true, nil
	}

	// 5. Conviction check — must have enough for the full cast
	// Stage 12.1: Apply conviction_cost_multiplier from Magical Resistance mutation
	convMult := 1.0 + mutations.GetConvictionCostMultiplier(user.Character.Mutations)
	totalConvictionCost := spellInfo.GetTotalConvictionCost(convMult)
	if totalConvictionCost > 0 && user.Character.Conviction < totalConvictionCost {
		user.SendText(fmt.Sprintf(
			`<ansi fg="red">You don't have enough conviction to cast %s. (Need %d, have %d)</ansi>`,
			spellInfo.Name, totalConvictionCost, user.Character.Conviction))
		return true, nil
	}

	// 6. Check initiation cooldown (blocks if a prior attempt failed)
	if user.Character.GetCooldown(`cast-init`) > 0 {
		user.SendText(fmt.Sprintf(
			`<ansi fg="red">Your mind is still recovering. (%d round(s) remaining)</ansi>`,
			user.Character.GetCooldown(`cast-init`)))
		return true, nil
	}

	// Initiation roll
	initiationChance := characters.CalcInitiationChance(user.Character.Stats.Willpower.ValueAdj, skillLevel)
	roll := util.Rand(100)
	util.LogRoll(`Spell Initiation`, roll, initiationChance)

	if roll >= initiationChance {
		// Failed — apply 2-round cooldown and inform user
		user.Character.TryCooldown(`cast-init`, `2 rounds`)
		user.SendText(fmt.Sprintf(
			`<ansi fg="red">You reach for the folds of %s but your concentration slips. (Rolled %d on %d%% chance)</ansi>`,
			spellInfo.Name, roll, initiationChance))
		room.SendText(fmt.Sprintf(
			`<ansi fg="username">%s</ansi> <ansi fg="red">loses their concentration.</ansi>`,
			user.Character.Name), user.UserId)
		return true, nil
	}

	// 7. Resolve folds needed and rate
	baseFolds := spellInfo.BaseFolds
	if baseFolds == 0 {
		baseFolds = 4
	}
	foldsNeeded := characters.NextPowerOfTwo(baseFolds)

	// Stage 17.2: The Eye modulates Perception → folds-per-round for mutated casters.
	perForCast := user.Character.Stats.Perception.ValueAdj
	if len(user.Character.Mutations) > 0 {
		eyeFrac := (gametime.GetEyePhase() - 0.5) * 2 * float64(configs.GetBalanceConfig().MoonStatModMax)
		perForCast += int(float64(perForCast) * eyeFrac)
	}
	foldsPerRound := characters.CalcFoldsPerRound(perForCast, skillLevel)

	// 8. Resolve targets
	targetUserIds := []int{}
	targetMobInstanceIds := []int{}
	spellRest := ``

	switch spellInfo.Type {
	case spells.HarmSingle:
		if targetName != `` {
			pId, mId := room.FindByName(targetName)
			if mId > 0 {
				targetMobInstanceIds = append(targetMobInstanceIds, mId)
			} else if pId > 0 {
				targetUserIds = append(targetUserIds, pId)
			} else {
				user.SendText(fmt.Sprintf(`<ansi fg="red">You don't see "%s" here.</ansi>`, targetName))
				return true, nil
			}
		} else if user.Character.Aggro != nil && user.Character.Aggro.MobInstanceId > 0 {
			targetMobInstanceIds = append(targetMobInstanceIds, user.Character.Aggro.MobInstanceId)
		} else if user.Character.Aggro != nil && user.Character.Aggro.UserId > 0 {
			targetUserIds = append(targetUserIds, user.Character.Aggro.UserId)
		} else {
			user.SendText(`<ansi fg="red">You need a target to cast that spell.</ansi>`)
			return true, nil
		}

	case spells.HarmMulti:
		if targetName != `` {
			pId, mId := room.FindByName(targetName)
			if mId > 0 {
				targetMobInstanceIds = append(targetMobInstanceIds, mId)
			} else if pId > 0 {
				targetUserIds = append(targetUserIds, pId)
			}
		} else if user.Character.Aggro != nil && user.Character.Aggro.MobInstanceId > 0 {
			targetMobInstanceIds = append(targetMobInstanceIds, user.Character.Aggro.MobInstanceId)
		} else if user.Character.Aggro != nil && user.Character.Aggro.UserId > 0 {
			targetUserIds = append(targetUserIds, user.Character.Aggro.UserId)
		}

	case spells.HelpSingle:
		if targetName != `` && targetName != user.Character.Name {
			pId, _ := room.FindByName(targetName)
			if pId > 0 {
				targetUserIds = append(targetUserIds, pId)
			} else {
				user.SendText(fmt.Sprintf(`<ansi fg="red">You don't see "%s" here.</ansi>`, targetName))
				return true, nil
			}
		} else {
			targetUserIds = append(targetUserIds, user.UserId) // default to self
		}

	case spells.HelpMulti:
		targetUserIds = append(targetUserIds, user.UserId) // self; spell script expands to party

	case spells.HarmArea, spells.HelpArea, spells.Neutral:
		spellRest = targetName // pass through for spell script
	}

	// 8.5. Component check — must have the required item in inventory before committing
	if spellInfo.ComponentTag != "" {
		found := false
		for _, itm := range user.Character.Items {
			if itm.GetSpec().ComponentTag == spellInfo.ComponentTag {
				found = true
				break
			}
		}
		if !found {
			user.SendText(fmt.Sprintf(
				`<ansi fg="red">%s requires a %s in your inventory.</ansi>`,
				spellInfo.Name, spellInfo.ComponentTag))
			return true, nil
		}
	}

	// 9. Cooldown gate — casting shares the special-move slot (prevents cast+bash same round)
	cfg := configs.GetGamePlayConfig()
	if !user.Character.TryCooldown(`special-move`, fmt.Sprintf(`%d rounds`, cfg.SpecialMoveCooldown)) {
		remaining := user.Character.GetCooldown(`special-move`)
		roundWord := "round"
		if remaining > 1 {
			roundWord = "rounds"
		}
		user.SendText(fmt.Sprintf(
			`You need a moment before you can do that! (%d %s remaining)`, remaining, roundWord))
		return true, nil
	}

	// 10. Set CastingState — folds accumulate each combat round
	user.Character.CastingState = &characters.CastingState{
		SpellId:              spellInfo.SpellId,
		FoldsNeeded:          foldsNeeded,
		FoldsAccumulated:     0,
		FoldsPerRound:        foldsPerRound,
		TotalConvictionCost:  totalConvictionCost,
		ConvictionSpent:      0,
		TargetUserIds:        targetUserIds,
		TargetMobInstanceIds: targetMobInstanceIds,
		SpellRest:            spellRest,
	}

	// 10. Announce and fire skill-used event
	events.AddToQueue(events.SkillUsed{
		UserId:  user.UserId,
		Skill:   skills.Spellcasting,
		Details: spellInfo.Name,
	})

	user.SendText(fmt.Sprintf(
		`<ansi fg="cyan">You gather your will and form an image of <ansi fg="cyan-bold">%s</ansi>... (0/%d folds)</ansi>`,
		spellInfo.Name, foldsNeeded))
	room.SendText(fmt.Sprintf(
		`<ansi fg="username">%s</ansi> closes their eyes in concentration.`,
		user.Character.Name), user.UserId)

	return true, nil
}
