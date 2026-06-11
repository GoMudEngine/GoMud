package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/opinions"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Shoot fires a loaded ranged weapon at a target. The shot resolves
// immediately (one shot per command) — same-room `shoot <target>` or
// cross-room `shoot <target> <direction>`. Firing unloads the weapon;
// callers must `reload` to fire again.
//
// This is the loaded-weapon model (ranged-weapons T6). It REPLACES the
// legacy remote-aggro behavior where a single `shoot` set up a continuous
// round-loop auto-attack into the adjacent room.
func Shoot(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// --- Party friendly-fire guard (PRE-fire) ---
	// ExecuteFire applies damage the moment it's called, so a post-hoc party
	// check would be too late to prevent the hit. We cheaply re-resolve the
	// would-be player target here and block before firing. The duplicated
	// resolution is the deliberate price of safe friendly-fire prevention.
	if partyInfo := parties.Get(user.UserId); partyInfo != nil {
		if pId := wouldBeShootTargetUserId(room, rest); pId > 0 && partyInfo.IsMember(pId) {
			if p := users.GetByUserId(pId); p != nil {
				user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="username">%s</ansi> is in your party!`, p.Character.Name))
				return true, nil
			}
		}
	}

	result := actions.ExecuteFire(&actions.UserActor{User: user, Room: room}, rest)

	// --- Early exits (no shot fired) ---
	if result.Crafting {
		user.SendText(messaging.CategorySystem, `<ansi fg="red">You can't shoot while focused on your work. Finish or be interrupted first.</ansi>`)
		return true, nil
	}
	if result.NoWeapon {
		user.SendText(messaging.CategorySystem, `You don't have a ranged weapon equipped.`)
		return true, nil
	}
	if result.NotLoaded {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`Your <ansi fg="itemname">%s</ansi> isn't loaded. Try <ansi fg="command">reload</ansi>.`, result.WeaponName))
		return true, nil
	}
	if result.BadSyntax {
		user.SendText(messaging.CategorySystem, `Syntax: <ansi fg="command">shoot &lt;target&gt; [direction]</ansi>`)
		return true, nil
	}
	if result.ExitLocked {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`The <ansi fg="exit">%s</ansi> exit is locked.`, result.ExitName))
		return true, nil
	}
	if result.NoTarget {
		user.SendText(messaging.CategorySystem, `Could not find your target.`)
		return true, nil
	}
	if result.IsCharmed {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is your friend!`, result.TargetName))
		return true, nil
	}
	if result.IsNonCombatant {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You can't attack <ansi fg="mobname">%s</ansi>.`, result.TargetName))
		return true, nil
	}

	// PvP gate for player targets — resolved AFTER ExecuteFire (damage is
	// already applied; the party guard above is the pre-fire protection).
	// CanPvp is informational here; we still completed the shot.
	if !result.IsTargetMob && result.TargetUserId > 0 {
		if p := users.GetByUserId(result.TargetUserId); p != nil {
			if pvpErr := room.CanPvp(user, p); pvpErr != nil {
				user.SendText(messaging.CategorySystem, pvpErr.Error())
				// Shot already fired; fall through to messaging/aggro so the
				// world stays consistent.
			}
		}
	}

	if !result.Executed {
		return true, nil
	}

	// --- Messaging ---
	sendShootMessages(user, room, result)

	hit := result.MoveResult.Hit

	// --- Shooter aggro (same-room only; cross-room is one-shot, no chase) ---
	if !result.CrossRoom && user.Character.Aggro == nil {
		if result.IsTargetMob {
			user.Character.SetAggro(0, result.TargetMobInstanceId, characters.DefaultAttack)
		} else if result.TargetUserId > 0 {
			user.Character.SetAggro(result.TargetUserId, 0, characters.DefaultAttack)
		}
	}

	// --- Retaliation + crime (mob targets only) ---
	if result.IsTargetMob {
		// The mob (and its faction) only react if the shot was noticed: a
		// hit always reveals the shooter; a clean miss from stealth does
		// not. A visible shooter is always noticed.
		noticed := hit || !result.IsSneaking
		if m := mobs.GetInstance(result.TargetMobInstanceId); m != nil && noticed {

			if hit {
				m.Character.TrackPlayerDamage(user.UserId, result.MoveResult.Damage)
				// mob_hurt behavior tree — same trigger the unified combat
				// handler fires (fireDefenderBehaviorTrigger).
				behaviortree.TryMobBehavior(m.InstanceId, behaviortree.EventContext{
					EventType: "mob_hurt",
					RoomId:    m.Character.RoomId,
					UserId:    user.UserId,
				})
			}

			if result.CrossRoom {
				// Aggro alone won't make a mob chase cross-room — the unified
				// combat handler drops a mob attacker's aggro the moment its
				// target isn't in the room. Cross-room pursuit instead runs
				// through the revenge-mob GOAL: the PlayerAttackedMob event
				// (below) seeds it, and CombatMemory is what lets that goal's
				// context score see the shooter as "recently seen" so the
				// goal planner emits `pathto` toward the shooter's room.
				m.CombatMemory = mobs.SetCombatMemory(user.UserId, 0, user.Character.RoomId, util.GetRoundCount())
			} else if m.Character.Aggro == nil {
				m.Character.SetAggro(user.UserId, 0, characters.DefaultAttack)
			}

			// Seed revenge + disposition + faction crime — mirrors the melee
			// fresh-aggression block in attack.go. PlayerAttackedMob drives
			// the revenge-mob seeder (the cross-room pursuit machinery);
			// recordAssaultCrime lets IdentifiedPerp attribute the shot.
			events.AddToQueue(events.PlayerAttackedMob{
				UserId:        user.UserId,
				MobInstanceId: m.InstanceId,
			})
			opinions.Bump(int(m.MobId), user.UserId, int(configs.GetBalanceConfig().OpinionAttackBump))
			recordAssaultCrime(user, m, rooms.LoadRoom(result.TargetRoomId))
		}
	}

	// --- Progression: perception always, ranged-combat on hit (mirror melee) ---
	user.Character.OnStatUse("perception", user.UserId)
	if hit {
		user.Character.OnSkillUse(string(skills.RangedCombat), user.UserId)
	}

	return true, nil
}

// sendShootMessages emits the shooter line, room broadcasts, and the target's
// direct line for an executed shot. No raw numbers — damage is described via
// combat.GetDamageDescription.
func sendShootMessages(user *users.UserRecord, room *rooms.Room, result actions.FireResult) {

	hit := result.MoveResult.Hit
	tier := combat.GetDamageDescription(result.MoveResult.Damage, result.MoveResult.TargetMaxHP)

	// Color the target name by type.
	targetColored := fmt.Sprintf(`<ansi fg="mobname">%s</ansi>`, result.TargetName)
	if !result.IsTargetMob {
		targetColored = fmt.Sprintf(`<ansi fg="username">%s</ansi>`, result.TargetName)
	}

	// Shooter's own feedback.
	if hit {
		user.SendText(messaging.CategoryHitRanged, fmt.Sprintf(`Your shot takes %s (<ansi fg="damage">%s</ansi>)!`, targetColored, tier))
	} else {
		user.SendText(messaging.CategoryDodge, fmt.Sprintf(`Your shot goes wide of %s!`, targetColored))
	}

	// Direct line to a player target.
	if !result.IsTargetMob && result.TargetUserId > 0 {
		if p := users.GetByUserId(result.TargetUserId); p != nil {
			shooter := fmt.Sprintf(`<ansi fg="username">%s</ansi>`, user.Character.Name)
			if result.IsSneaking {
				shooter = `Someone`
			}
			if hit {
				p.SendText(messaging.CategoryHitRanged, fmt.Sprintf(`%s's shot strikes you (<ansi fg="damage">%s</ansi>)!`, shooter, tier))
			} else {
				p.SendText(messaging.CategoryHitRanged, fmt.Sprintf(`%s's shot narrowly misses you!`, shooter))
			}
		}
	}

	weapon := fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, result.WeaponName)
	shooterName := fmt.Sprintf(`<ansi fg="username">%s</ansi>`, user.Character.Name)

	if !result.CrossRoom {
		// Same-room broadcast (exclude shooter + player target). Suppressed
		// when sneaking.
		if !result.IsSneaking {
			room.SendTextVisual(messaging.CategoryHitRanged,
				fmt.Sprintf(`%s fires their %s at %s!`, shooterName, weapon, targetColored),
				user.UserId, result.TargetUserId)
		}
		return
	}

	// Cross-room. Shooter's room sees the shot leave (suppressed when sneaking).
	if !result.IsSneaking {
		room.SendTextVisual(messaging.CategoryHitRanged,
			fmt.Sprintf(`%s fires their %s %sward.`, shooterName, weapon, result.ExitName),
			user.UserId)
	}

	// Target's room sees the shot arrive (exclude the player target — they
	// already got the direct line above).
	if tr := rooms.LoadRoom(result.TargetRoomId); tr != nil {
		fromDir := tr.FindExitTo(room.RoomId)
		origin := `from somewhere nearby`
		if fromDir != "" {
			origin = fmt.Sprintf(`from beyond the <ansi fg="exit">%s</ansi>`, fromDir)
		}
		if hit {
			tr.SendTextVisual(messaging.CategoryHitRanged,
				fmt.Sprintf(`A shot streaks in %s and strikes %s!`, origin, targetColored),
				result.TargetUserId)
		} else {
			tr.SendTextVisual(messaging.CategoryHitRanged,
				fmt.Sprintf(`A shot streaks in %s and narrowly misses %s!`, origin, targetColored),
				result.TargetUserId)
		}
	}
}

// wouldBeShootTargetUserId mirrors ExecuteFire's target parsing to find the
// player (if any) a shoot command would hit, WITHOUT applying damage. Used
// only for the pre-fire party friendly-fire guard. Returns 0 when the target
// is a mob, absent, or unresolvable.
func wouldBeShootTargetUserId(room *rooms.Room, rest string) int {
	if room == nil {
		return 0
	}
	args := strings.Fields(rest)
	if len(args) < 1 {
		return 0
	}
	targetRoom := room
	targetWords := args
	if len(args) >= 2 {
		if name, roomId := room.FindExitByName(args[len(args)-1]); name != "" {
			if adj := rooms.LoadRoom(roomId); adj != nil {
				targetRoom = adj
				targetWords = args[:len(args)-1]
			}
		}
	}
	uId, _ := targetRoom.FindByName(strings.Join(targetWords, " "))
	if uId == 0 && targetRoom != room {
		// Trailing word may have been part of the name; retry same-room.
		uId, _ = room.FindByName(strings.Join(args, " "))
	}
	return uId
}
