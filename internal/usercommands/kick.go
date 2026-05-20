package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Kick(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	if user.Character.IsActing() {
		user.SendTextLegacy(`<ansi fg="red">You can't kick while focused on your work. Finish or be interrupted first.</ansi>`)
		return true, nil
	}

	// Must be in combat or specify a target to use kick
	if !user.Character.IsInCombat() {
		if rest == "" {
			user.SendTextLegacy("Kick whom?")
			return true, nil
		}

		target, err := actions.ResolveTargetActor(room, rest, actions.ResolveTargetOptions{
			ExcludeUserId: user.UserId,
		})
		if err != nil {
			// Self-exclusion collapses to NotFound; pre-check for self-targeting message.
			if pId, _ := room.FindByName(rest); pId == user.UserId {
				user.SendTextLegacy("You can't kick yourself.")
				return true, nil
			}
			user.SendTextLegacy("You don't see them here.")
			return true, nil
		}

		if !target.IsPlayer() {
			mob := target.(*actions.MobActor).Mob
			if mob.IsNonCombatant() || mob.PlayerAttackImmune {
				user.SendTextLegacy(fmt.Sprintf(`You can't attack <ansi fg="mobname">%s</ansi>.`, mob.Character.Name))
				return true, nil
			}
			user.Character.SetAggro(0, mob.InstanceId, characters.DefaultAttack)
		} else {
			p := target.(*actions.UserActor).User
			if pvpErr := room.CanPvp(user, p); pvpErr != nil {
				user.SendTextLegacy(pvpErr.Error())
				return true, nil
			}
			user.Character.SetAggro(p.UserId, 0, characters.DefaultAttack)
		}
	}

	// Delegate core resolution to the shared action.
	res := actions.ExecuteKick(&actions.UserActor{User: user, Room: room})

	if res.Crafting {
		// Safety net — should have been caught by the pre-reject above.
		user.SendTextLegacy(`<ansi fg="red">You can't kick while focused on your work. Finish or be interrupted first.</ansi>`)
		return true, nil
	}

	if res.OnCooldown {
		user.SendTextLegacy("You need a moment to recover before attempting another special move.")
		return true, nil
	}
	if res.NoTarget {
		user.SendTextLegacy("You have no target!")
		return true, nil
	}

	// Build variant-specific message arrays.
	var kickMsgs, kickTargetMsgs, kickRoomMsgs []string
	var knockdownMsgs, knockdownTargetMsgs, knockdownRoomMsgs []string
	var missMsgs, missTargetMsgs, missRoomMsgs []string

	targetName := res.Target.Name

	switch res.Variant {
	case actions.KickStomp:
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

	case actions.KickKnee:
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

	default: // KickStandard
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

	// Resolve player target for direct messaging.
	var targetChar *users.UserRecord
	if res.Target.UserId > 0 {
		targetChar = users.GetByUserId(res.Target.UserId)
	}

	// Send messages.
	dmgDesc := combat.GetDamageDescription(res.MoveResult.Damage, res.MoveResult.TargetMaxHP)

	if res.MoveResult.Hit {
		if res.MoveResult.KnockedDown && len(knockdownMsgs) > 0 {
			user.SendTextLegacy(fmt.Sprintf(knockdownMsgs[util.Rand(len(knockdownMsgs))], targetName, dmgDesc))
			if targetChar != nil {
				targetChar.SendTextLegacy(fmt.Sprintf(knockdownTargetMsgs[util.Rand(len(knockdownTargetMsgs))], user.Character.Name, dmgDesc))
			}
			room.SendTextVisualLegacy(fmt.Sprintf(knockdownRoomMsgs[util.Rand(len(knockdownRoomMsgs))], user.Character.Name, targetName), user.UserId, res.Target.UserId)
		} else {
			user.SendTextLegacy(fmt.Sprintf(kickMsgs[util.Rand(len(kickMsgs))], targetName, dmgDesc))
			if targetChar != nil {
				targetChar.SendTextLegacy(fmt.Sprintf(kickTargetMsgs[util.Rand(len(kickTargetMsgs))], user.Character.Name, dmgDesc))
			}
			room.SendTextVisualLegacy(fmt.Sprintf(kickRoomMsgs[util.Rand(len(kickRoomMsgs))], user.Character.Name, targetName), user.UserId, res.Target.UserId)
		}
	} else {
		user.SendTextLegacy(fmt.Sprintf(missMsgs[util.Rand(len(missMsgs))], targetName))
		if targetChar != nil {
			targetChar.SendTextLegacy(fmt.Sprintf(missTargetMsgs[util.Rand(len(missTargetMsgs))], user.Character.Name))
		}
		room.SendTextVisualLegacy(fmt.Sprintf(missRoomMsgs[util.Rand(len(missRoomMsgs))], user.Character.Name, targetName), user.UserId, res.Target.UserId)
	}

	return true, nil
}
