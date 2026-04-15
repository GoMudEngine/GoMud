package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/worldevents"
)

func Suicide(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	currentRound := util.GetRoundCount()

	if rest != `vanish` && mob.Character.HasBuffFlag(buffs.ReviveOnDeath) {

		mob.Character.Health = mob.Character.HealthMax.Value

		room.SendTextVisual(`<ansi fg="mobname">` + mob.Character.Name + `</ansi> is suddenly revived in a shower of sparks!`)

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

		// Stage 38.4: Delete saved instance on vanish too
		mobs.DeleteMobInstance(mob.MobId, mob.Zone, mob.Character.Name, mob.HomeRoomId)

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
	sendMovementMessage(room,
		fmt.Sprintf(`<ansi fg="mobname">%s</ansi> has died.`, mob.Character.Name),
		`You hear something collapse to the ground.`)

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
		PlayerDamage:  mob.Character.PlayerDamage,
	})

	// Emit world event for gossip system when a player kills a mob
	if len(mob.Character.PlayerDamage) > 0 && mob.Character.Zone != `Training` {
		// Find the first attacker name for the event description
		killerName := "an adventurer"
		for uId := range mob.Character.PlayerDamage {
			if user := users.GetByUserId(uId); user != nil {
				killerName = user.Character.Name
				break
			}
		}
		sig := worldevents.Local
		region := ""
		if zCfg := rooms.GetZoneConfig(mob.Character.Zone); zCfg != nil {
			region = zCfg.Region
		}
		worldevents.EmitWorldEvent(worldevents.WorldEvent{
			Type:         worldevents.MobKilledByPlayer,
			Significance: sig,
			ZoneName:     mob.Character.Zone,
			RegionName:   region,
			MobName:      mob.Character.Name,
			PlayerName:   killerName,
			Description: fmt.Sprintf("%s slew %s out in %s.",
				killerName, mob.Character.Name, mob.Character.Zone),
		})
	}

	// Stage 3.5: No XP awarded. Progression is skill-based via combat hooks.
	// Still track kills and taming.

	// Behavior tree: fire mob_die once with primary killer
	if len(mob.Character.PlayerDamage) > 0 {
		var killerUserId int
		for uId := range mob.Character.PlayerDamage {
			killerUserId = uId
			break
		}
		behaviortree.TryMobBehavior(mob.InstanceId, behaviortree.EventContext{
			EventType: "mob_die",
			UserId:    killerUserId,
			RoomId:    room.RoomId,
		})
	}

	if len(mob.Character.PlayerDamage) > 0 {

		attackerCt := len(mob.Character.PlayerDamage)

		for uId := range mob.Character.PlayerDamage {
			if user := users.GetByUserId(uId); user != nil {

				if user.Character.Aggro != nil {
					if user.Character.Aggro.MobInstanceId == mob.InstanceId {
						user.Character.EndAggro()
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

		lootDropped := false

		// Check for any dropped loot...
		for _, item := range mob.Character.Items {
			msg := fmt.Sprintf(`<ansi fg="item">%s</ansi> drops to the ground.`, item.DisplayName())
			sendMovementMessage(room, msg, ``)
			room.AddItem(item, false)
			lootDropped = true
		}

		allWornItems := mob.Character.Equipment.GetAllItems()

		for _, item := range allWornItems {

			roll := util.Rand(100)

			util.LogRoll(`Drop Item`, roll, mob.ItemDropChance)

			if roll >= mob.ItemDropChance {
				continue
			}

			msg := fmt.Sprintf(`<ansi fg="item">%s</ansi> drops to the ground.`, item.DisplayName())
			sendMovementMessage(room, msg, ``)
			room.AddItem(item, false)
			lootDropped = true
		}

		if mob.Character.Gold > 0 {
			msg := fmt.Sprintf(`<ansi fg="yellow-bold">%d gold</ansi> drops to the ground.`, mob.Character.Gold)
			sendMovementMessage(room, msg, ``)
			room.Gold += mob.Character.Gold
			lootDropped = true
		}

		// One-time sound fallback for loot drops in dark rooms
		if lootDropped && room.GetVisibility() < 1 {
			sendMovementMessage(room, ``, `You hear something clatter to the ground.`)
		}

	}

	// Stage 38.4: Delete saved instance so respawn starts fresh from template
	mobs.DeleteMobInstance(mob.MobId, mob.Zone, mob.Character.Name, mob.HomeRoomId)

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
			WasCharmed:   mob.Character.IsCharmed() || mob.Character.EverCharmed,
		})
	}

	return true, nil
}
