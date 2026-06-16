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
	"github.com/GoMudEngine/GoMud/internal/questengine"
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

	// --- Pre-fire target resolution & guards ---
	// ExecuteFire applies damage the instant it is called, so every guard that
	// must PREVENT a shot (friendly-fire, PvP legality, self-target) has to run
	// here, before firing. We re-resolve the would-be target (mirroring
	// ExecuteFire's parse) without applying damage. The duplicated resolution is
	// the deliberate price of safe pre-fire gating.
	tUserId, tMobId, tRoom, crossRoom := resolveShootTarget(room, rest)

	// Issue 3 (self-target): room.FindByName can resolve the shooter.
	if tUserId > 0 && tUserId == user.UserId {
		user.SendText(messaging.CategorySystem, `You can't shoot yourself.`)
		return true, nil
	}

	// Player-target guards: party friendly-fire + PvP legality.
	if tUserId > 0 {
		if p := users.GetByUserId(tUserId); p != nil {
			if partyInfo := parties.Get(user.UserId); partyInfo != nil && partyInfo.IsMember(tUserId) {
				user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="username">%s</ansi> is in your party!`, p.Character.Name))
				return true, nil
			}
			// Issue 1 (PvP bypass): gate BEFORE firing, exactly as melee's
			// attack.go does via room.CanPvp. CanPvp's limited-mode branch keys
			// off the RECEIVER room's IsPvp(), so for a cross-room shot we also
			// require the TARGET's room to permit PvP — you must not be able to
			// snipe into a sanctuary from a lawless room outside it.
			if pvpErr := room.CanPvp(user, p); pvpErr != nil {
				user.SendText(messaging.CategorySystem, pvpErr.Error())
				return true, nil
			}
			if tRoom != nil && tRoom != room {
				if pvpErr := tRoom.CanPvp(user, p); pvpErr != nil {
					user.SendText(messaging.CategorySystem, pvpErr.Error())
					return true, nil
				}
			}
		}
	}

	// Issue 2 (free opening shot): RecordAndWait inside ExecuteFire only charges
	// the attacker a combat round when Aggro != nil. Kick sets aggro BEFORE its
	// core resolver for exactly this reason; shoot used to set it AFTER, so an
	// out-of-combat opener consumed no round (free shot + a full melee swing the
	// next round). Mirror kick: for a same-room opener, engage now so the round
	// is charged. Cross-room stays aggro-free by design (the sniper isn't in the
	// round loop). If ExecuteFire then refuses the shot (charm / non-combatant /
	// no-target → !Executed) we roll the speculative aggro back below so a
	// rejected shot never starts combat.
	aggroPreset := false
	if !crossRoom && user.Character.Aggro == nil {
		if tMobId > 0 {
			user.Character.SetAggro(0, tMobId, characters.DefaultAttack)
			aggroPreset = true
		} else if tUserId > 0 {
			user.Character.SetAggro(tUserId, 0, characters.DefaultAttack)
			aggroPreset = true
		}
	}

	result := actions.ExecuteFire(&actions.UserActor{User: user, Room: room}, rest)

	// Roll back the speculative opening-shot aggro if the shot was refused
	// (any !Executed early return below: charm, non-combatant, no-target, etc.).
	if aggroPreset && !result.Executed {
		user.Character.EndAggro()
	}

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

	if !result.Executed {
		return true, nil
	}

	// --- Messaging ---
	sendShootMessages(user, room, result)

	hit := result.MoveResult.Hit

	// Issue 5 (cross-room hidden reveal): same-room shots reveal the shooter
	// through the normal aggro→combat path (CancelCombatBuffs in the combat
	// round handler, set up by the pre-fire aggro above). A cross-room shooter
	// never enters that loop, so a hidden sniper would stay hidden forever.
	// Mirror melee's reveal-on-engage: a cross-room shot that LANDS drops
	// stealth. A clean cross-room miss stays hidden — one free hit, then you're
	// exposed. (Same-room shooter aggro is set pre-fire; see Issue 2 above.)
	if result.CrossRoom && hit {
		user.Character.CancelCombatBuffs()
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

	// Notify the quest engine of the `shoot` command so quests can gate a step
	// on the act of shooting (Spoke G's "shoot the practice butt" / "shoot the
	// foe across the canyon" beats). Mirrors forage/drink/throw/cast. RoomId is
	// the SHOOTER's room — for a cross-room shot the quest gates on the room the
	// player shoots FROM, not the target's room.
	bridge := questengine.NewGameBridge(user, room.RoomId)
	questengine.GetEngine().Notify("command", questengine.EventDetails{
		UserId:  user.UserId,
		RoomId:  room.RoomId,
		Command: "shoot",
	}, bridge, bridge)

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

// resolveShootTarget mirrors ExecuteFire's target parsing to determine what a
// shoot command WOULD hit, without firing or applying damage. It powers the
// pre-fire guards (self-target, party friendly-fire, PvP legality) and the
// opening-shot aggro preset. Returns the resolved player id, mob instance id,
// the room the target stands in, and whether the shot is cross-room. Any id may
// be 0 when unresolved; targetRoom is the shooter's room for same-room / absent
// targets.
func resolveShootTarget(room *rooms.Room, rest string) (userId, mobInstanceId int, targetRoom *rooms.Room, crossRoom bool) {
	if room == nil {
		return 0, 0, nil, false
	}
	args := strings.Fields(rest)
	if len(args) < 1 {
		return 0, 0, room, false
	}
	targetRoom = room
	targetWords := args
	if len(args) >= 2 {
		if name, roomId := room.FindExitByName(args[len(args)-1]); name != "" {
			if adj := rooms.LoadRoom(roomId); adj != nil {
				crossRoom = true
				targetRoom = adj
				targetWords = args[:len(args)-1]
			}
		}
	}
	userId, mobInstanceId = targetRoom.FindByName(strings.Join(targetWords, " "))
	if userId == 0 && mobInstanceId == 0 && crossRoom {
		// The trailing word may have been part of the target name after all;
		// retry as a same-room shot using the full argument string.
		crossRoom = false
		targetRoom = room
		userId, mobInstanceId = room.FindByName(strings.Join(args, " "))
	}
	return userId, mobInstanceId, targetRoom, crossRoom
}
