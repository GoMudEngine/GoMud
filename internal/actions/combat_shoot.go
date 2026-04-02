package actions

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// ShootResult holds the outcome of a shoot setup attempt. Unlike the other
// combat actions, shoot does not fire a projectile immediately — it resolves
// a target in an adjacent room and sets remote aggro on the actor so the
// normal attack loop fires on the next round.
type ShootResult struct {
	// ExitName is the direction the actor is shooting toward.
	ExitName string

	// TargetName is the display name of the resolved target.
	TargetName string

	// TargetUserId is the player ID of the target (0 if mob).
	TargetUserId int

	// TargetMobInstanceId is the mob instance ID of the target (0 if player).
	TargetMobInstanceId int

	// IsTargetMob is true when the resolved target is a mob.
	IsTargetMob bool

	// IsSneaking is true when the actor is hidden at the moment of shooting.
	IsSneaking bool

	// Ready is true when remote aggro was successfully set. False for all
	// early-exit conditions below.
	Ready bool

	// NoWeapon is true when the actor does not have a shooting weapon equipped.
	NoWeapon bool

	// BadSyntax is true when fewer than 2 args were provided (need target + direction).
	BadSyntax bool

	// NoExit is true when the direction arg did not resolve to a valid exit.
	NoExit bool

	// ExitLocked is true when the exit exists but its lock is closed.
	ExitLocked bool

	// NoTarget is true when no matching entity was found in the adjacent room.
	NoTarget bool

	// IsCharmed is true when the target mob is charmed by this actor (friendly
	// fire prevention). Only set for mob targets.
	IsCharmed bool
}

// ExecuteShoot performs the core shoot target-resolution shared between player
// and mob callers. It handles:
//   - Shooting weapon check
//   - Argument parsing (target words + direction)
//   - Exit lookup and lock check
//   - Adjacent room load + target search
//   - Charmed mob check (charmed by this actor → IsCharmed)
//   - SetAggroRemote on the actor's character
//
// Callers are responsible for:
//   - Party-member friendly-fire check (player only)
//   - All player-facing and room messaging
//   - Sneaking suppression of room messages
func ExecuteShoot(actor Actor, rest string) ShootResult {
	char := actor.GetCharacter()

	// Shooting weapon required.
	if char.Equipment.Weapon.GetSpec().Subtype != items.Shooting {
		return ShootResult{NoWeapon: true}
	}

	// Parse args: everything before the last word is the target name;
	// the last word is the direction.
	args := util.SplitButRespectQuotes(rest)
	if len(args) < 2 {
		return ShootResult{BadSyntax: true}
	}

	direction := args[len(args)-1]
	targetWords := args[:len(args)-1]

	// Resolve the exit.
	room := actor.GetRoom()
	exitName, attackRoomId := room.FindExitByName(direction)
	if exitName == "" {
		return ShootResult{NoExit: true}
	}

	// Check for a locked exit.
	exitInfo, _ := room.GetExitInfo(exitName)
	if exitInfo.Lock.IsLocked() {
		return ShootResult{ExitName: exitName, ExitLocked: true}
	}

	// Load the adjacent room and find the target.
	adjacentRoom := rooms.LoadRoom(attackRoomId)
	if adjacentRoom == nil {
		return ShootResult{NoTarget: true}
	}

	attackPlayerId, attackMobInstanceId := adjacentRoom.FindByName(strings.Join(targetWords, " "))
	if attackPlayerId == 0 && attackMobInstanceId == 0 {
		return ShootResult{NoTarget: true}
	}

	// Determine sneaking state.
	isSneaking := char.HasBuffFlag(buffs.Hidden)

	if attackMobInstanceId > 0 {
		m := mobs.GetInstance(attackMobInstanceId)
		if m == nil {
			return ShootResult{NoTarget: true}
		}

		// Charmed-mob friendly-fire prevention.
		// Player actors charm by userId; mob actors charm by instanceId.
		charmerKey := actor.GetUserId()
		if charmerKey == 0 {
			charmerKey = actor.GetMobInstanceId()
		}
		if m.Character.IsCharmed(charmerKey) {
			return ShootResult{
				ExitName:            exitName,
				TargetName:          m.Character.Name,
				TargetMobInstanceId: attackMobInstanceId,
				IsTargetMob:         true,
				IsCharmed:           true,
			}
		}

		char.SetAggroRemote(exitName, 0, attackMobInstanceId, characters.Shooting)

		return ShootResult{
			ExitName:            exitName,
			TargetName:          m.Character.Name,
			TargetMobInstanceId: attackMobInstanceId,
			IsTargetMob:         true,
			IsSneaking:          isSneaking,
			Ready:               true,
		}
	}

	// Player target.
	p := users.GetByUserId(attackPlayerId)
	if p == nil {
		return ShootResult{NoTarget: true}
	}

	char.SetAggroRemote(exitName, attackPlayerId, 0, characters.Shooting)

	return ShootResult{
		ExitName:     exitName,
		TargetName:   p.Character.Name,
		TargetUserId: attackPlayerId,
		IsTargetMob:  false,
		IsSneaking:   isSneaking,
		Ready:        true,
	}
}
