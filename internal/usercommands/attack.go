package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crimes"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/opinions"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Attack(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	attackPlayerId := 0
	attackMobInstanceId := 0

	if rest == `` {
		partyInfo := parties.Get(user.UserId)

		// If no argument supplied, attack whoever is attacking the player currently.
		for _, mId := range room.GetMobs(rooms.FindFightingPlayer) {
			m := mobs.GetInstance(mId)
			if m == nil || !m.Character.IsInCombat() {
				continue
			}

			if m.Character.EngagedTarget().UserId == user.UserId {
				attackMobInstanceId = m.InstanceId
				break
			}

			if partyInfo != nil {
				if partyInfo.IsMember(m.Character.EngagedTarget().UserId) {
					attackMobInstanceId = m.InstanceId
					break
				}
			}
		}

		if attackMobInstanceId == 0 {
			for _, uId := range room.GetPlayers(rooms.FindFightingPlayer) {
				u := users.GetByUserId(uId)
				if !u.Character.IsInCombat() {
					continue
				}

				if u.Character.EngagedTarget().UserId == user.UserId {
					attackPlayerId = u.UserId
					break
				}

				if partyInfo != nil {
					if partyInfo.IsMember(u.Character.EngagedTarget().UserId) {
						attackPlayerId = u.UserId
						break
					}
				}
			}
		}

		// Finally, if still no targets, check if any party members are aggroed and just glom onto that
		if attackMobInstanceId == 0 && attackPlayerId == 0 {
			if partyInfo != nil {
				for uId := range partyInfo.GetMembers() {
					if partyUser := users.GetByUserId(uId); partyUser != nil {
						if !partyUser.Character.IsInCombat() {
							continue
						}

						if partyUser.Character.EngagedTarget().MobInstanceId > 0 {
							attackMobInstanceId = partyUser.Character.EngagedTarget().MobInstanceId
							break
						}

						if partyUser.Character.EngagedTarget().UserId > 0 {
							attackPlayerId = partyUser.Character.EngagedTarget().UserId
							break
						}

					}
				}
			}
		}

	} else {
		// Wildcard and named-target resolution delegated to shared helper.
		t := actions.FindAttackTarget(rest, room, user.UserId, 0)
		attackPlayerId = t.UserId
		attackMobInstanceId = t.MobInstanceId
	}

	if attackMobInstanceId == 0 && attackPlayerId == 0 {
		user.SendText(messaging.CategorySystem, "You attack the darkness!")
		return true, nil
	}

	isSneaking := user.Character.IsHidden()

	/*
		combatAddlWaitRounds := user.Character.Equipment.Weapon.GetSpec().WaitRounds + user.Character.Equipment.Weapon.GetSpec().WaitRounds
		attkType := characters.DefaultAttack
		if user.Character.Equipment.Weapon.GetSpec().Subtype == items.Shooting {
			attkType = characters.Shooting
		}
	*/

	// --- TARGET SWITCHING LOGIC (Stage 7.4) ---
	// If already in combat and trying to attack a different target, use target switching
	if user.Character.IsInCombat() {
		currentTargetUserId := user.Character.CurrentCombatTarget().UserId
		currentTargetMobId := user.Character.CurrentCombatTarget().MobInstanceId

		isDifferentTarget := false
		if attackMobInstanceId > 0 && attackMobInstanceId != currentTargetMobId {
			isDifferentTarget = true
		}
		if attackPlayerId > 0 && attackPlayerId != currentTargetUserId {
			isDifferentTarget = true
		}

		// If switching targets, use the Target command logic instead
		if isDifferentTarget && (user.Character.Aggro.Type == characters.DefaultAttack || user.Character.Aggro.Type == characters.Shooting) {
			// Build target name for the Target command
			targetName := rest
			if targetName == "" || targetName[0] == '*' {
				// For empty or random targets, find the actual name
				if attackMobInstanceId > 0 {
					if m := mobs.GetInstance(attackMobInstanceId); m != nil {
						targetName = m.Character.Name
					}
				} else if attackPlayerId > 0 {
					if p := users.GetByUserId(attackPlayerId); p != nil {
						targetName = p.Character.Name
					}
				}
			}
			// Delegate to Target command for proper switching logic
			return Target(targetName, user, room, flags)
		}
	}
	// --- END TARGET SWITCHING LOGIC ---

	if attackMobInstanceId > 0 {

		m := mobs.GetInstance(attackMobInstanceId)

		if m != nil {
			dupIdx := room.GetMobDuplicateIndex(m.InstanceId)
			mName := m.Character.GetMobNameIndexed(user.UserId, dupIdx).String()

			if m.Character.IsCharmed() {
				user.SendText(messaging.CategorySystem, fmt.Sprintf(`%s is someone's companion!`, mName))
				return true, nil
			}

			if m.IsNonCombatant() || m.PlayerAttackImmune {
				user.SendText(messaging.CategorySystem, fmt.Sprintf(`You can't attack <ansi fg="mobname">%s</ansi>.`, m.Character.Name))
				mobs.FireAttackRejected(m, user.UserId)
				return true, nil
			}

			if party := parties.Get(user.UserId); party != nil {
				for _, id := range party.UserIds {
					if id == user.UserId {
						continue
					}
					if partyUser := users.GetByUserId(id); partyUser != nil {
						if partyUser.Character.RoomId == user.Character.RoomId &&
							partyUser.Character.GetSetting("autoattack") != "off" &&
							!partyUser.Character.IsInCombat() {
							// Surprise attack for hidden party members before they join combat
							if targetMob := mobs.GetInstance(attackMobInstanceId); targetMob != nil {
								partyActor := actions.NewUserActorInRoom(partyUser, room)
								targetActor := actions.NewMobActorInRoom(targetMob, room)
								actions.SurpriseAttack(partyActor, actions.SurpriseAttackOpts{Target: targetActor})
							}
							partyUser.Command(fmt.Sprintf(`attack #%d`, attackMobInstanceId))
						}
					}
				}
			}

			// Surprise attack from stealth — fires before normal combat begins.
			// SurpriseAttack gates on IsHidden() internally; call unconditionally.
			if targetMob := mobs.GetInstance(attackMobInstanceId); targetMob != nil {
				actor := actions.NewUserActorInRoom(user, room)
				targetActor := actions.NewMobActorInRoom(targetMob, room)
				actions.SurpriseAttack(actor, actions.SurpriseAttackOpts{Target: targetActor})
			}

			// Detect "fresh aggression" before SetAggro overwrites prior state:
			// either no prior aggro, or aggro on a different target.
			// NOTE: Keep Aggro read here for Task 12 — CombatPhase is not
			// populated by writers until Task 15; EngagedTarget() would
			// return zero and cause double-bumps until the sunset in Task 18.
			isFreshAggro := user.Character.Aggro == nil ||
				user.Character.Aggro.MobInstanceId != attackMobInstanceId

			user.Character.SetAggro(0, attackMobInstanceId, characters.DefaultAttack)

			// Chunk 4.5: notify seeders that a player engaged a mob.
			// Fires on every attack commitment, not just fresh aggro,
			// so rules 6 + 9 see repeated aggressive actions too.
			// Only fires for mob targets (player-vs-player handled above).
			events.AddToQueue(events.PlayerAttackedMob{
				UserId:        user.UserId,
				MobInstanceId: attackMobInstanceId,
			})

			if isFreshAggro {
				if mob := mobs.GetInstance(attackMobInstanceId); mob != nil {
					// Per-NPC opinion (chunk 1.1).
					opinions.Bump(int(mob.MobId), user.UserId,
						int(configs.GetBalanceConfig().OpinionAttackBump))
					// Per-faction crime + rep (chunk 1.3).
					recordAssaultCrime(user, mob, room)
				}
			}

			user.SendText(messaging.CategoryHitMelee,
				fmt.Sprintf(`You prepare to enter into mortal combat with %s.`, mName),
			)

			if !isSneaking {
				room.SendTextVisual(messaging.CategoryHitMelee,
					fmt.Sprintf(`<ansi fg="username">%s</ansi> prepares to fight %s.`, user.Character.Name, mName),
					user.UserId,
				)
			}

			for _, instId := range room.GetMobs(rooms.FindCharmed) {
				if m := mobs.GetInstance(instId); m != nil {
					if !m.Character.IsInCombat() && m.Character.IsCharmed(user.UserId) {
						// Only auto-assist if the companion has AutoAssist enabled
						comp := user.Character.GetCompanionByInstanceId(instId)
						if comp != nil && comp.AutoAssist {
							m.Command(fmt.Sprintf(`attack #%d`, attackMobInstanceId))
						}
					}
				}
			}

		}

	} else if attackPlayerId > 0 {

		if p := users.GetByUserId(attackPlayerId); p != nil {

			if pvpErr := room.CanPvp(user, p); pvpErr != nil {
				user.SendText(messaging.CategorySystem, pvpErr.Error())
				return true, nil
			}

			if partyInfo := parties.Get(user.UserId); partyInfo != nil {
				if partyInfo.IsMember(attackPlayerId) {
					user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="username">%s</ansi> is in your party!`, p.Character.Name))
					return true, nil
				}
			}

			if party := parties.Get(user.UserId); party != nil {
				if party.IsLeader(user.UserId) {
					for _, id := range party.GetAutoAttackUserIds() {
						if id == user.UserId {
							continue
						}
						if partyUser := users.GetByUserId(id); partyUser != nil {
							if partyUser.Character.RoomId == user.Character.RoomId {
								partyUser.Command(fmt.Sprintf(`attack @%d`, attackPlayerId)) // # denotes a specific mob instanceId
							}
						}
					}
				}
			}

			// Surprise attack from stealth — fires before normal combat begins.
			// SurpriseAttack gates on IsHidden() internally; call unconditionally.
			if targetUser := users.GetByUserId(attackPlayerId); targetUser != nil {
				actor := actions.NewUserActorInRoom(user, room)
				targetActor := actions.NewUserActorInRoom(targetUser, room)
				actions.SurpriseAttack(actor, actions.SurpriseAttackOpts{Target: targetActor})
			}

			user.Character.SetAggro(attackPlayerId, 0, characters.DefaultAttack)

			user.SendText(messaging.CategoryHitMelee,
				fmt.Sprintf(`You prepare to enter into mortal combat with <ansi fg="username">%s</ansi>.`, p.Character.Name),
			)

			if !isSneaking {

				p.SendText(messaging.CategoryHitMelee,
					fmt.Sprintf(`<ansi fg="username">%s</ansi> prepares to fight you!`, user.Character.Name),
				)

				room.SendTextVisual(messaging.CategoryHitMelee,
					fmt.Sprintf(`<ansi fg="username">%s</ansi> prepares to fight <ansi fg="mobname">%s</ansi>.`, user.Character.Name, p.Character.Name),
					user.UserId, attackPlayerId)
			}

			for _, instId := range room.GetMobs(rooms.FindCharmed) {
				if m := mobs.GetInstance(instId); m != nil {
					if !m.Character.IsInCombat() && m.Character.IsCharmed(user.UserId) {
						comp := user.Character.GetCompanionByInstanceId(instId)
						if comp != nil && comp.AutoAssist {
							m.Command(fmt.Sprintf(`attack @%d`, attackPlayerId))
						}
					}
				}
			}

		}

	}

	return true, nil
}

// recordAssaultCrime records an assault crime against each defined
// faction the mob belongs to, and bumps player rep with each
// (only when perpetrator identified). Shared between attack.go and
// target.go's bumpOpinionOnTargetSwitch.
func recordAssaultCrime(user *users.UserRecord, mob *mobs.Mob, room *rooms.Room) {
	factionIds := factions.FactionsForMob(mob)
	if len(factionIds) == 0 {
		return
	}
	// All witnesses including the victim (excludeInstanceId=0) drive
	// perp/rep determination (victim is alive and a self-witness).
	witnesses := crimes.WitnessesInRoom(factionIds, room, 0)
	perp := crimes.IdentifiedPerp(user.UserId, witnesses)
	// External witnesses: same call but exclude the victim — used to
	// set HadExternalWitness so the murder-upgrade path knows whether
	// the assault was seen by someone other than the victim.
	externalWitnesses := crimes.WitnessesInRoom(factionIds, room, mob.InstanceId)
	hadExternal := len(externalWitnesses) > 0
	delta := int(configs.GetBalanceConfig().CrimeRepDeltaAssault)
	for _, fid := range factionIds {
		crimeIds := crimes.Record([]string{fid}, crimes.KindAssault, perp,
			mob, mob.InstanceId, room.RoomId, mob.Character.Zone, hadExternal)
		if perp.Type == crimes.PerpPlayer {
			factions.BumpRep(fid, user.UserId, delta)
			// Knowledge: each witness records the player as the perp of
			// these crimes.
			subject := knowledge.PlayerSubject(user.UserId)
			for _, witnessInstId := range witnesses {
				w := mobs.GetInstance(witnessInstId)
				if w == nil {
					continue
				}
				for _, crimeId := range crimeIds {
					knowledge.RecordCrimeWitnessed(int(w.MobId), subject, crimeId)
				}
				knowledge.RecordMet(int(w.MobId), subject, room.RoomId,
					knowledge.SourceWitnessed)
			}
		}
	}
}
