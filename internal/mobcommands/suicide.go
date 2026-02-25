package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Suicide(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	currentRound := util.GetRoundCount()

	if rest != `vanish` && mob.Character.HasBuffFlag(buffs.ReviveOnDeath) {

		mob.Character.Health = mob.Character.HealthMax.Value

		room.SendText(`<ansi fg="mobname">` + mob.Character.Name + `</ansi> is suddenly revived in a shower of sparks!`)

		mob.Character.CancelBuffsWithFlag(buffs.ReviveOnDeath)

		return true, nil
	}

	// Useful to know sometimes
	mobs.TrackRecentDeath(mob.InstanceId)

	mudlog.Debug(`Mob Death`, `name`, mob.Character.Name, `rest`, rest)

	// Make sure to clean up any charm stuff if it's being removed
	if charmedUserId := mob.Character.RemoveCharm(); charmedUserId > 0 {
		if charmedUser := users.GetByUserId(charmedUserId); charmedUser != nil {
			charmedUser.Character.TrackCharmed(mob.InstanceId, false)
		}
	}

	// vanish is meant to remove the mob without any rewards/drops/etc.
	if rest == `vanish` {

		// Destroy any record of this mob.
		mobs.DestroyInstance(mob.InstanceId)

		// Clean up mob from room...
		if r := rooms.LoadRoom(mob.HomeRoomId); r != nil {
			r.CleanupMobSpawns(false)
		}

		// Remove from current room
		room.RemoveMob(mob.InstanceId)

		return true, nil
	}

	// Send a death msg to everyone in the room.
	room.SendText(
		fmt.Sprintf(`<ansi fg="mobname">%s</ansi> has died.`, mob.Character.Name),
	)

	// Special handling of "The Guide"
	// Mark this moment to prevent an immediate respawn
	if mob.MobId == 38 {
		if mob.Character.Charmed != nil {
			if tmpU := users.GetByUserId(mob.Character.Charmed.UserId); tmpU != nil {
				tmpU.SetTempData(`lastGuideRound`, currentRound)
			}
		}
	}

	events.AddToQueue(events.MobDeath{
		MobId:         int(mob.MobId),
		InstanceId:    mob.InstanceId,
		RoomId:        room.RoomId,
		CharacterName: mob.Character.Name,
		Level:         mob.Character.Level,
		PlayerDamage:  mob.Character.PlayerDamage,
	})

	// Stage 3.5: No XP awarded. Progression is skill-based via combat hooks.
	// Still track kills and taming.

	if len(mob.Character.PlayerDamage) > 0 {

		attackerCt := len(mob.Character.PlayerDamage)

		for uId := range mob.Character.PlayerDamage {
			if user := users.GetByUserId(uId); user != nil {

				if user.Character.Aggro != nil {
					if user.Character.Aggro.MobInstanceId == mob.InstanceId {
						user.Character.Aggro = nil
					}
				}

				scripting.TryMobScriptEvent(`onDie`, mob.InstanceId, uId, `user`, map[string]any{`attackerCount`: attackerCt})

				if mob.Character.Zone != `Training` { // Don't track any kills in the training zone
					user.Character.KD.AddMobKill(int(mob.MobId))
					// Check for first kill of this mob type
					if user.Character.KD.GetMobKills(int(mob.MobId)) == 1 {
						user.Character.OnFirstMobKill(user.UserId)
					}
				}

				// Chance to learn to tame the creature (skill-based, no level comparison)
				skillsDelta := int((float64(user.Character.Stats.Charisma.ValueAdj-mob.Character.Stats.Charisma.ValueAdj) + float64(user.Character.Stats.Perception.ValueAdj-mob.Character.Stats.Perception.ValueAdj)) / 2)
				if skillsDelta < 0 {
					skillsDelta = 0
				}
				targetNumber := skillsDelta + user.Character.GetCombatSkillLevel()*2
				if targetNumber < 1 {
					targetNumber = 1
				}

				mudlog.Debug("Tame Chance", "skillsDelta", skillsDelta, "targetNumber", targetNumber)

				if util.Rand(1000) < targetNumber {
					if mob.IsTameable() && user.Character.GetSkillLevel(skills.Spellcasting) > 0 {

						currentSkill := user.Character.MobMastery.GetTame(int(mob.MobId))
						if currentSkill < 50 {
							user.Character.MobMastery.SetTame(int(mob.MobId), currentSkill+1)
							if currentSkill == -1 {
								user.SendText(fmt.Sprintf(`<ansi fg="magenta">***</ansi> You've learned how to tame a <ansi fg="mobname">%s</ansi>! <ansi fg="magenta">***</ansi>`, mob.Character.Name))
							} else {
								user.SendText(fmt.Sprintf(`<ansi fg="magenta">***</ansi> Your <ansi fg="mobname">%s</ansi> taming skills get a little better! <ansi fg="magenta">***</ansi>`, mob.Character.Name))
							}
						}

					}
				}

			}
		}

	}

	// Stage 3.5: Party members also get kill credit (no XP to split)
	// Give kill tracking to party members who didn't directly attack
	partyMembersHandled := map[int]bool{}
	for uId := range mob.Character.PlayerDamage {
		partyMembersHandled[uId] = true
		if p := parties.Get(uId); p != nil {
			for _, memberId := range p.GetMembers() {
				if partyMembersHandled[memberId] {
					continue
				}
				partyMembersHandled[memberId] = true
				if user := users.GetByUserId(memberId); user != nil {
					if mob.Character.Zone != `Training` {
						user.Character.KD.AddMobKill(int(mob.MobId))
						if user.Character.KD.GetMobKills(int(mob.MobId)) == 1 {
							user.Character.OnFirstMobKill(user.UserId)
						}
					}
				}
			}
		}
	}

	if !mob.Character.HasBuffFlag(buffs.PermaGear) {

		// Check for any dropped loot...
		for _, item := range mob.Character.Items {
			msg := fmt.Sprintf(`<ansi fg="item">%s</ansi> drops to the ground.`, item.DisplayName())
			room.SendText(msg)
			room.AddItem(item, false)
		}

		allWornItems := mob.Character.Equipment.GetAllItems()

		for _, item := range allWornItems {

			roll := util.Rand(100)

			util.LogRoll(`Drop Item`, roll, mob.ItemDropChance)

			if roll >= mob.ItemDropChance {
				continue
			}

			msg := fmt.Sprintf(`<ansi fg="item">%s</ansi> drops to the ground.`, item.DisplayName())
			room.SendText(msg)
			room.AddItem(item, false)
		}

		if mob.Character.Gold > 0 {
			msg := fmt.Sprintf(`<ansi fg="yellow-bold">%d gold</ansi> drops to the ground.`, mob.Character.Gold)
			room.SendText(msg)
			room.Gold += mob.Character.Gold
		}

	}

	// Destroy any record of this mob.
	mobs.DestroyInstance(mob.InstanceId)

	// Clean up mob from room...
	if r := rooms.LoadRoom(mob.HomeRoomId); r != nil {
		r.CleanupMobSpawns(false)
	}

	// Remove from current room
	room.RemoveMob(mob.InstanceId)

	config := configs.GetGamePlayConfig()

	if config.Death.CorpsesEnabled {
		room.AddCorpse(rooms.Corpse{
			MobId:        int(mob.MobId),
			Character:    mob.Character,
			RoundCreated: currentRound,
		})
	}

	return true, nil
}
