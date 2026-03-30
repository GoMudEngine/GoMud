package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Kick(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Must be in combat to use kick
	if user.Character.Aggro == nil {
		user.SendText("You must be in combat to kick!")
		return true, nil
	}

	// Check shared special move cooldown
	cfg := configs.GetBalanceConfig()
	if !user.Character.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		user.SendText("You need a moment to recover before attempting another special move.")
		return true, nil
	}

	// Resolve target
	targetPlayerId := user.Character.Aggro.UserId
	targetMobId := user.Character.Aggro.MobInstanceId

	var targetChar *users.UserRecord
	var targetMob *mobs.Mob
	var targetName string
	var defender *characters.Character

	if targetMobId > 0 {
		targetMob = mobs.GetInstance(targetMobId)
		if targetMob == nil {
			user.SendText("Your target is gone!")
			return true, nil
		}
		targetName = targetMob.Character.Name
		defender = &targetMob.Character
	} else if targetPlayerId > 0 {
		targetChar = users.GetByUserId(targetPlayerId)
		if targetChar == nil {
			user.SendText("Your target is gone!")
			return true, nil
		}
		targetName = targetChar.Character.Name
		defender = targetChar.Character
	} else {
		user.SendText("You have no target!")
		return true, nil
	}

	// Determine kick variant based on combat positions
	variant := "kick"
	damagePercent := float64(cfg.KickDamagePercent)
	knockdownChance := int(cfg.KickKnockdownChance)
	mitigationMult := 1.0

	// Stomp: target is prone
	if defender.CombatPosition == characters.PositionProne {
		variant = "stomp"
		damagePercent = float64(cfg.StompDamagePercent)
		knockdownChance = 0
		mitigationMult = 0.5
	}

	// Knee: attacker is clinched or grounded AND in control
	if (user.Character.CombatPosition == characters.PositionClinched ||
		user.Character.CombatPosition == characters.PositionGrounded) &&
		user.Character.HasCondition(characters.ConditionGrappleController) {
		variant = "knee"
		damagePercent = float64(cfg.KneeDamagePercent)
		knockdownChance = 0
		mitigationMult = 1.0
	}

	// Execute skill move
	result := combat.ExecuteSkillMove(combat.SkillMoveParams{
		Attacker:             user.Character,
		Defender:             defender,
		AttackStat:           user.Character.Stats.Strength.ValueAdj,
		AttackSkill:          user.Character.GetSkillLevel(skills.UnarmedCombat),
		DefenseStat:          defender.Stats.Dexterity.ValueAdj,
		DefenseSkill:         defender.GetCombatSkillLevel(),
		DamagePercent:        damagePercent,
		KnockdownChance:      knockdownChance,
		SkillRank:            user.Character.GetSkillLevel(skills.UnarmedCombat),
		DamageStat:           user.Character.Stats.Strength.ValueAdj,
		MitigationMultiplier: mitigationMult,
	})

	// Build variant-specific message arrays
	var kickMsgs, kickTargetMsgs, kickRoomMsgs []string
	var knockdownMsgs, knockdownTargetMsgs, knockdownRoomMsgs []string
	var missMsgs, missTargetMsgs, missRoomMsgs []string

	switch variant {
	case "stomp":
		kickMsgs = []string{
			`You bring your heel down hard on <ansi fg="mobname">%s</ansi>! (<ansi fg="damage">%s</ansi>)`,
			`You slam your foot into the downed <ansi fg="mobname">%s</ansi>! (<ansi fg="damage">%s</ansi>)`,
			`You drive a vicious stomp into <ansi fg="mobname">%s</ansi> while they scramble! (<ansi fg="damage">%s</ansi>)`,
			`You crush your boot down on <ansi fg="mobname">%s</ansi>! (<ansi fg="damage">%s</ansi>)`,
			`You stamp hard on the prone <ansi fg="mobname">%s</ansi>! (<ansi fg="damage">%s</ansi>)`,
			`You grind your heel into <ansi fg="mobname">%s</ansi> as they try to rise! (<ansi fg="damage">%s</ansi>)`,
		}
		kickTargetMsgs = []string{
			`<ansi fg="username">%s</ansi> stomps on you while you're down! (<ansi fg="damage">%s</ansi>)`,
			`<ansi fg="username">%s</ansi> slams a boot into you! (<ansi fg="damage">%s</ansi>)`,
			`<ansi fg="username">%s</ansi> drives their heel into you! (<ansi fg="damage">%s</ansi>)`,
			`<ansi fg="username">%s</ansi> crushes you underfoot! (<ansi fg="damage">%s</ansi>)`,
			`A vicious stomp from <ansi fg="username">%s</ansi> smashes into you! (<ansi fg="damage">%s</ansi>)`,
			`<ansi fg="username">%s</ansi> grinds a boot into your ribs! (<ansi fg="damage">%s</ansi>)`,
		}
		kickRoomMsgs = []string{
			`<ansi fg="username">%s</ansi> stomps on the downed <ansi fg="mobname">%s</ansi>!`,
			`<ansi fg="username">%s</ansi> slams a boot into <ansi fg="mobname">%s</ansi>!`,
			`<ansi fg="username">%s</ansi> drives their heel into <ansi fg="mobname">%s</ansi>!`,
			`<ansi fg="username">%s</ansi> viciously stomps <ansi fg="mobname">%s</ansi>!`,
			`<ansi fg="username">%s</ansi> crushes <ansi fg="mobname">%s</ansi> underfoot!`,
			`<ansi fg="username">%s</ansi> grinds their heel into <ansi fg="mobname">%s</ansi>!`,
		}
		missMsgs = []string{
			`You try to stomp <ansi fg="mobname">%s</ansi>, but they roll aside!`,
			`Your stomp misses as <ansi fg="mobname">%s</ansi> twists away!`,
			`You slam your foot down but <ansi fg="mobname">%s</ansi> squirms free!`,
		}
		missTargetMsgs = []string{
			`<ansi fg="username">%s</ansi> tries to stomp you, but you roll aside!`,
			`<ansi fg="username">%s</ansi>'s stomp misses as you twist away!`,
		}
		missRoomMsgs = []string{
			`<ansi fg="username">%s</ansi> tries to stomp <ansi fg="mobname">%s</ansi>, but misses!`,
		}
	case "knee":
		kickMsgs = []string{
			`You drive a knee into <ansi fg="mobname">%s</ansi>! (<ansi fg="damage">%s</ansi>)`,
			`You slam your knee into <ansi fg="mobname">%s</ansi>'s body! (<ansi fg="damage">%s</ansi>)`,
			`You ram a sharp knee into <ansi fg="mobname">%s</ansi>! (<ansi fg="damage">%s</ansi>)`,
			`You crack your knee against <ansi fg="mobname">%s</ansi>! (<ansi fg="damage">%s</ansi>)`,
			`You pump a knee strike into <ansi fg="mobname">%s</ansi>'s midsection! (<ansi fg="damage">%s</ansi>)`,
			`You wrench <ansi fg="mobname">%s</ansi> close and knee them hard! (<ansi fg="damage">%s</ansi>)`,
		}
		kickTargetMsgs = []string{
			`<ansi fg="username">%s</ansi> drives a knee into you! (<ansi fg="damage">%s</ansi>)`,
			`<ansi fg="username">%s</ansi> slams a knee into your body! (<ansi fg="damage">%s</ansi>)`,
			`<ansi fg="username">%s</ansi> rams their knee into you! (<ansi fg="damage">%s</ansi>)`,
			`<ansi fg="username">%s</ansi> cracks a knee against you! (<ansi fg="damage">%s</ansi>)`,
			`A sharp knee from <ansi fg="username">%s</ansi> hits your midsection! (<ansi fg="damage">%s</ansi>)`,
			`<ansi fg="username">%s</ansi> wrenches you close and knees you hard! (<ansi fg="damage">%s</ansi>)`,
		}
		kickRoomMsgs = []string{
			`<ansi fg="username">%s</ansi> drives a knee into <ansi fg="mobname">%s</ansi>!`,
			`<ansi fg="username">%s</ansi> slams a knee into <ansi fg="mobname">%s</ansi>!`,
			`<ansi fg="username">%s</ansi> rams their knee into <ansi fg="mobname">%s</ansi>!`,
			`<ansi fg="username">%s</ansi> cracks a knee against <ansi fg="mobname">%s</ansi>!`,
			`<ansi fg="username">%s</ansi> knees <ansi fg="mobname">%s</ansi> hard in the grapple!`,
		}
		missMsgs = []string{
			`You try to knee <ansi fg="mobname">%s</ansi>, but can't find the angle!`,
			`Your knee strike glances off <ansi fg="mobname">%s</ansi>!`,
			`<ansi fg="mobname">%s</ansi> blocks your knee!`,
		}
		missTargetMsgs = []string{
			`<ansi fg="username">%s</ansi> tries to knee you, but you block it!`,
			`<ansi fg="username">%s</ansi>'s knee strike glances off you!`,
		}
		missRoomMsgs = []string{
			`<ansi fg="username">%s</ansi> tries to knee <ansi fg="mobname">%s</ansi> in the grapple, but misses!`,
		}
	default: // kick
		kickMsgs = []string{
			`Your <ansi fg="yellow-bold">kick</ansi> strikes <ansi fg="mobname">%s</ansi>! (<ansi fg="damage">%s</ansi>)`,
			`You land a solid kick on <ansi fg="mobname">%s</ansi>! (<ansi fg="damage">%s</ansi>)`,
			`Your boot connects with <ansi fg="mobname">%s</ansi>! (<ansi fg="damage">%s</ansi>)`,
			`You drive a kick into <ansi fg="mobname">%s</ansi>! (<ansi fg="damage">%s</ansi>)`,
			`You snap a kick at <ansi fg="mobname">%s</ansi> and it lands! (<ansi fg="damage">%s</ansi>)`,
			`Your foot crashes into <ansi fg="mobname">%s</ansi>! (<ansi fg="damage">%s</ansi>)`,
		}
		kickTargetMsgs = []string{
			`<ansi fg="username">%s</ansi> kicks you hard! (<ansi fg="damage">%s</ansi>)`,
			`<ansi fg="username">%s</ansi> lands a solid kick on you! (<ansi fg="damage">%s</ansi>)`,
			`<ansi fg="username">%s</ansi>'s boot connects with you! (<ansi fg="damage">%s</ansi>)`,
			`<ansi fg="username">%s</ansi> drives a kick into you! (<ansi fg="damage">%s</ansi>)`,
			`A sharp kick from <ansi fg="username">%s</ansi> hits you! (<ansi fg="damage">%s</ansi>)`,
		}
		kickRoomMsgs = []string{
			`<ansi fg="username">%s</ansi> kicks <ansi fg="mobname">%s</ansi>!`,
			`<ansi fg="username">%s</ansi> lands a solid kick on <ansi fg="mobname">%s</ansi>!`,
			`<ansi fg="username">%s</ansi> drives a kick into <ansi fg="mobname">%s</ansi>!`,
			`<ansi fg="username">%s</ansi> snaps a kick at <ansi fg="mobname">%s</ansi>!`,
		}
		knockdownMsgs = []string{
			`Your powerful <ansi fg="yellow-bold">kick</ansi> knocks <ansi fg="mobname">%s</ansi> to the ground! (<ansi fg="damage">%s</ansi>)`,
			`Your kick sends <ansi fg="mobname">%s</ansi> sprawling! (<ansi fg="damage">%s</ansi>)`,
			`A devastating kick drops <ansi fg="mobname">%s</ansi> to the floor! (<ansi fg="damage">%s</ansi>)`,
		}
		knockdownTargetMsgs = []string{
			`<ansi fg="username">%s</ansi>'s powerful kick knocks you to the ground! (<ansi fg="damage">%s</ansi>)`,
			`<ansi fg="username">%s</ansi>'s kick sends you sprawling! (<ansi fg="damage">%s</ansi>)`,
		}
		knockdownRoomMsgs = []string{
			`<ansi fg="username">%s</ansi> kicks <ansi fg="mobname">%s</ansi>, knocking them to the ground!`,
			`<ansi fg="username">%s</ansi>'s kick sends <ansi fg="mobname">%s</ansi> sprawling!`,
		}
		missMsgs = []string{
			`Your <ansi fg="yellow-bold">kick</ansi> misses <ansi fg="mobname">%s</ansi>!`,
			`You swing a kick at <ansi fg="mobname">%s</ansi> but miss!`,
			`Your kick sails past <ansi fg="mobname">%s</ansi>!`,
		}
		missTargetMsgs = []string{
			`<ansi fg="username">%s</ansi> attempts to kick you, but misses!`,
			`<ansi fg="username">%s</ansi> swings a kick at you but misses!`,
		}
		missRoomMsgs = []string{
			`<ansi fg="username">%s</ansi> attempts to kick <ansi fg="mobname">%s</ansi>, but misses!`,
			`<ansi fg="username">%s</ansi> swings a kick at <ansi fg="mobname">%s</ansi> but misses!`,
		}
	}

	// Send messages
	dmgDesc := combat.GetDamageDescription(result.Damage, result.TargetMaxHP)

	if result.Hit {
		// Stomp extends prone duration
		if variant == "stomp" && defender.CombatPosition == characters.PositionProne {
			defender.PositionRoundsMin += 1
		}

		if result.KnockedDown && len(knockdownMsgs) > 0 {
			user.SendText(fmt.Sprintf(knockdownMsgs[util.Rand(len(knockdownMsgs))], targetName, dmgDesc))
			if targetChar != nil {
				targetChar.SendText(fmt.Sprintf(knockdownTargetMsgs[util.Rand(len(knockdownTargetMsgs))], user.Character.Name, dmgDesc))
			}
			room.SendText(fmt.Sprintf(knockdownRoomMsgs[util.Rand(len(knockdownRoomMsgs))], user.Character.Name, targetName), user.UserId, targetPlayerId)
		} else {
			user.SendText(fmt.Sprintf(kickMsgs[util.Rand(len(kickMsgs))], targetName, dmgDesc))
			if targetChar != nil {
				targetChar.SendText(fmt.Sprintf(kickTargetMsgs[util.Rand(len(kickTargetMsgs))], user.Character.Name, dmgDesc))
			}
			room.SendText(fmt.Sprintf(kickRoomMsgs[util.Rand(len(kickRoomMsgs))], user.Character.Name, targetName), user.UserId, targetPlayerId)
		}
	} else {
		user.SendText(fmt.Sprintf(missMsgs[util.Rand(len(missMsgs))], targetName))
		if targetChar != nil {
			targetChar.SendText(fmt.Sprintf(missTargetMsgs[util.Rand(len(missTargetMsgs))], user.Character.Name))
		}
		room.SendText(fmt.Sprintf(missRoomMsgs[util.Rand(len(missRoomMsgs))], user.Character.Name, targetName), user.UserId, targetPlayerId)
	}

	// Record combat analytics
	dmgRecorded := 0
	if result.Hit {
		dmgRecorded = result.Damage
	}
	tgtType := combat.Mob
	if targetMob == nil {
		tgtType = combat.User
	}
	combat.RecordSpecialMove(combat.User, tgtType, variant, result.Hit, dmgRecorded, user.Character, defender, util.GetRoundCount())

	// Progress unarmed combat skill
	events.AddToQueue(events.SkillUsed{
		UserId:  user.UserId,
		Skill:   skills.UnarmedCombat,
		Details: variant,
	})

	// Kick costs the current combat round
	if user.Character.Aggro != nil {
		user.Character.Aggro.RoundsWaiting = 1
	}

	return true, nil
}
