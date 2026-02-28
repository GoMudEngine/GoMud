package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Grapple(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Must be in combat to use grapple
	if mob.Character.Aggro == nil {
		return true, nil
	}

	// Check shared special move cooldown
	cfg := configs.GetGamePlayConfig()
	if !mob.Character.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		return true, nil
	}

	// Resolve target
	targetPlayerId := mob.Character.Aggro.UserId
	targetMobId := mob.Character.Aggro.MobInstanceId

	var targetChar *users.UserRecord
	var targetMob *mobs.Mob
	var targetName string
	var defender *characters.Character

	if targetMobId > 0 {
		targetMob = mobs.GetInstance(targetMobId)
		if targetMob == nil {
			return true, nil
		}
		targetName = targetMob.Character.Name
		defender = &targetMob.Character
	} else if targetPlayerId > 0 {
		targetChar = users.GetByUserId(targetPlayerId)
		if targetChar == nil {
			return true, nil
		}
		targetName = targetChar.Character.Name
		defender = targetChar.Character
	} else {
		return true, nil
	}

	// Execute grapple move (attackerId=0 for mobs)
	result := combat.ExecuteGrappleMove(&mob.Character, defender, 0, room)

	// Send messages based on result
	if result.Success {
		if targetChar != nil {
			targetChar.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> <ansi fg="yellow-bold">grapples</ansi> you, transitioning to <ansi fg="cyan">%s</ansi> position!`, mob.Character.Name, result.PositionDesc))
		}
		room.SendText(
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> <ansi fg="yellow-bold">grapples</ansi> <ansi fg="username">%s</ansi> into <ansi fg="cyan">%s</ansi> position!`, mob.Character.Name, targetName, result.PositionDesc),
			targetPlayerId,
		)

		// Disarm messaging
		if result.DisarmResult != nil {
			if targetChar != nil {
				targetChar.SendText(result.DisarmResult.TargetMsg)
			}
			room.SendText(result.DisarmResult.RoomMessage, targetPlayerId)
		}
	} else {
		if targetChar != nil {
			targetChar.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> tries to grapple you, but you slip away!`, mob.Character.Name))
		}
		room.SendText(
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> tries to grapple <ansi fg="username">%s</ansi>, but fails!`, mob.Character.Name, targetName),
			targetPlayerId,
		)

		// Critical failure messaging
		if result.CritFailure != nil {
			if targetChar != nil {
				targetChar.SendText(result.CritFailure.TargetMessage)
			}
			room.SendText(result.CritFailure.RoomMessage, targetPlayerId)
		}
	}

	// Record combat analytics
	tgtType := combat.Mob
	if targetMob == nil {
		tgtType = combat.User
	}
	combat.RecordSpecialMove(combat.Mob, tgtType, "grapple", result.Success, 0, &mob.Character, defender, util.GetRoundCount())

	// Grapple costs the current combat round
	if mob.Character.Aggro != nil {
		mob.Character.Aggro.RoundsWaiting = 1
	}

	return true, nil
}
