package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Taunt(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	if actions.AcquireMeleeTarget(user, room, rest, actions.MeleeTargetOpts{
		Verb: "taunt",
	}) {
		return true, nil
	}

	actor := &actions.UserActor{User: user, Room: room}
	result := actions.ExecuteTaunt(actor)

	if result.Crafting {
		// Safety net — should have been caught by the pre-reject above.
		user.SendText(messaging.CategorySystem, `<ansi fg="red">You can't taunt while focused on your work. Finish or be interrupted first.</ansi>`)
		return true, nil
	}

	if result.OnCooldown {
		user.SendText(messaging.CategorySystem, "You need a moment to recover before attempting another special move.")
		return true, nil
	}

	if result.NoTarget {
		user.SendText(messaging.CategorySystem, "You have no target!")
		return true, nil
	}

	// Chunk 4.5: notify seeders that a player engaged a mob via taunt.
	// Taunt is symmetric to attack for rules 6 + 9 — it is also a
	// player-initiated combat engagement. Only fire for mob targets.
	if result.Target.MobInstanceId > 0 {
		events.AddToQueue(events.PlayerAttackedMob{
			UserId:        user.UserId,
			MobInstanceId: result.Target.MobInstanceId,
		})
	}

	targetType := "mob"
	if result.Target.UserId > 0 {
		targetType = "user"
	}

	var targetPlayer *users.UserRecord
	if result.Target.UserId > 0 {
		targetPlayer = users.GetByUserId(result.Target.UserId)
	}

	sourceName := user.Character.Name
	targetName := result.Target.Name

	switch {
	case result.Fumble:
		sendTauntMessages(combat.TauntFumble, "", sourceName, targetName,
			"username", targetType, user, targetPlayer, room, result.Target.UserId)

	case result.Hit && result.Crit:
		sendTauntMessages(combat.TauntCritical, result.DmgDesc, sourceName, targetName,
			"username", targetType, user, targetPlayer, room, result.Target.UserId)

		if result.AggroPulled {
			sendAggroPullMessages(user, room, sourceName, targetName)
		}

	case result.Hit:
		sendTauntMessages(combat.TauntHit, result.DmgDesc, sourceName, targetName,
			"username", targetType, user, targetPlayer, room, result.Target.UserId)

		if result.AggroPulled {
			sendAggroPullMessages(user, room, sourceName, targetName)
		}

		// Stoic resolve messaging
		if result.CritDeflected {
			if targetPlayer != nil {
				targetPlayer.SendText(messaging.CategorySystem,
					`<ansi fg="green">The words wash over you harmlessly — you are unmoved.</ansi>`)
			}
			user.SendText(messaging.CategorySystem,
				`<ansi fg="yellow">Your words have no effect — your target is completely unmoved!</ansi>`)
		} else if result.Deflected {
			if targetPlayer != nil {
				targetPlayer.SendText(messaging.CategorySystem,
					`<ansi fg="green">You steel yourself against the barrage of words.</ansi>`)
			}
			user.SendText(messaging.CategorySystem,
				`<ansi fg="yellow">Your words fail to fully penetrate your target's resolve!</ansi>`)
		}

	default:
		sendTauntMessages(combat.TauntMiss, "", sourceName, targetName,
			"username", targetType, user, targetPlayer, room, result.Target.UserId)
	}

	// Quest engine: command notification — the player executed a taunt (the
	// newbie Lore spoke's "talk down the belligerent" rhetoric beat). Fires
	// after a real taunt (past the cooldown/no-target guards). Mirrors cast.
	bridge := questengine.NewGameBridge(user, room.RoomId)
	questengine.GetEngine().Notify("command", questengine.EventDetails{
		UserId:  user.UserId,
		RoomId:  room.RoomId,
		Command: "taunt",
	}, bridge, bridge)

	return true, nil
}

// sendAggroPullMessages notifies the taunter and the room that the mob
// switched its aggro to the taunter.
func sendAggroPullMessages(user *users.UserRecord, room *rooms.Room, sourceName, targetName string) {
	pullMsgs := []string{
		`Your words cut deep -- <ansi fg="mobname">%s</ansi> turns away from its prey and fixes its fury on you!`,
		`The insult lands true. <ansi fg="mobname">%s</ansi> abandons its quarry and lunges toward you!`,
		`Your mockery is unbearable -- <ansi fg="mobname">%s</ansi> wheels around to face you!`,
		`<ansi fg="mobname">%s</ansi> snarls and shifts its full attention to you.`,
	}
	user.SendText(messaging.CategorySystem, fmt.Sprintf(pullMsgs[util.Rand(len(pullMsgs))], targetName))

	roomPullMsgs := []string{
		`<ansi fg="mobname">%s</ansi> breaks off and turns its fury on <ansi fg="username">%s</ansi>!`,
		`Enraged by <ansi fg="username">%s</ansi>'s taunts, <ansi fg="mobname">%s</ansi> shifts its attention!`,
		`<ansi fg="mobname">%s</ansi> abandons its quarry and charges at <ansi fg="username">%s</ansi>!`,
	}
	idx := util.Rand(len(roomPullMsgs))
	var roomMsg string
	if idx == 1 {
		roomMsg = fmt.Sprintf(roomPullMsgs[idx], sourceName, targetName)
	} else {
		roomMsg = fmt.Sprintf(roomPullMsgs[idx], targetName, sourceName)
	}
	room.SendTextVisual(messaging.CategoryTauntSuccess, roomMsg, user.UserId)
}

// sendTauntMessages sends data-driven taunt messages to attacker, defender,
// and room. Falls back to generic text if no YAML messages are loaded.
func sendTauntMessages(intensity combat.TauntIntensity, dmgDesc, source, target, sourceType, targetType string,
	user *users.UserRecord, targetPlayer *users.UserRecord, room *rooms.Room, targetPlayerId int) {

	atkMsg := combat.GetTauntMessage(intensity, "toattacker", source, target, sourceType, targetType, dmgDesc)
	defMsg := combat.GetTauntMessage(intensity, "todefender", source, target, sourceType, targetType, dmgDesc)
	roomMsg := combat.GetTauntMessage(intensity, "toroom", source, target, sourceType, targetType, dmgDesc)

	// Fallback if no messages loaded
	if atkMsg == "" {
		switch intensity {
		case combat.TauntHit, combat.TauntCritical:
			atkMsg = fmt.Sprintf(`You hurl a scathing taunt at %s! (%s)`, target, dmgDesc)
		case combat.TauntMiss:
			atkMsg = fmt.Sprintf(`Your taunt falls flat against %s.`, target)
		case combat.TauntFumble:
			atkMsg = fmt.Sprintf(`You stumble over your words trying to taunt %s!`, target)
		}
	}

	user.SendText(messaging.CategorySystem, atkMsg)

	if targetPlayer != nil && defMsg != "" {
		targetPlayer.SendText(messaging.CategorySystem, defMsg)
	}

	if roomMsg != "" {
		room.SendTextVisual(messaging.CategoryTauntSuccess, roomMsg, user.UserId, targetPlayerId)
	}
}
