package usercommands

import (
	"errors"
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/colorpatterns"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/worldevents"
)

func Suicide(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	config := configs.GetGamePlayConfig()
	currentRound := util.GetRoundCount()

	if user.Character.Zone == `Shadow Realm` {
		user.SendText(`You're already dead!`)
		return true, errors.New(`already dead`)
	}

	if user.Character.HasBuffFlag(buffs.ReviveOnDeath) {

		user.Character.Health = user.Character.HealthMax.Value

		user.SendText(`You are revived in a shower of magical sparks!`)
		room.SendText(`<ansi fg="username">`+user.Character.Name+`</ansi> is suddenly revived in a shower of sparks!`, user.UserId)

		user.Character.CancelBuffsWithFlag(buffs.ReviveOnDeath)

		return true, nil
	}

	// Send a death msg to everyone in the room.
	room.SendText(
		fmt.Sprintf(`<ansi fg="username">%s</ansi> has died.`, user.Character.Name),
		user.UserId,
	)

	i := 0
	dmgCt := len(user.Character.PlayerDamage)

	if dmgCt > 0 {
		user.Character.KD.AddPvpDeath()
	} else {
		user.Character.KD.AddMobDeath()
	}

	killedByUserIds := []int{}
	killedBy := ``
	for uid, _ := range user.Character.PlayerDamage {

		if u := users.GetByUserId(uid); u != nil {

			// Update PK stats
			user.Character.KD.AddPlayerDeath(u.UserId, u.Character.Name)
			u.Character.KD.AddPlayerKill(user.UserId, user.Character.Name)

			if i > 0 {
				if i < dmgCt-1 {
					killedBy += ` and `
				} else {
					killedBy += `, `
				}
			}
			killedBy += `<ansi fg="username">` + u.Character.Name + `</ansi>`
			i++
		}

		killedByUserIds = append(killedByUserIds, uid)
	}

	msg := fmt.Sprintf(`<ansi fg="magenta-bold">***</ansi> <ansi fg="username">%s</ansi> has <ansi fg="red-bold">DIED!</ansi> <ansi fg="magenta-bold">***</ansi>%s`, user.Character.Name, term.CRLFStr)
	if killedBy != `` {
		msg = fmt.Sprintf(`<ansi fg="magenta-bold">***</ansi> <ansi fg="username">%s</ansi> has <ansi fg="red-bold">DIED!</ansi> (killed by %s) <ansi fg="magenta-bold">***</ansi>%s`, user.Character.Name, killedBy, term.CRLFStr)
	}

	events.AddToQueue(events.Broadcast{
		Text: msg,
	})

	allowPenalties := user.Character.GetTotalSkillRanks() > int(config.Death.ProtectionSkillRanks)

	events.AddToQueue(events.PlayerDeath{
		UserId:        user.UserId,
		RoomId:        user.Character.RoomId,
		Username:      user.Username,
		CharacterName: user.Character.Name,
		Permanent:     allowPenalties && bool(config.Death.PermaDeath) && user.Character.ExtraLives == 0,
		KilledByUsers: killedByUserIds,
	})

	// Emit a local gossip event for PvE deaths (not PvP)
	if dmgCt == 0 {
		causeOfDeath := ""
		// Check if fighting a mob
		if user.Character.Aggro != nil && user.Character.Aggro.MobInstanceId > 0 {
			if mob := mobs.GetInstance(user.Character.Aggro.MobInstanceId); mob != nil {
				causeOfDeath = mob.Character.Name
			}
		}
		// If not fighting a mob, check for lethal conditions
		if causeOfDeath == "" {
			if user.Character.HasCondition(characters.ConditionPoisoned) {
				causeOfDeath = "poison"
			} else if user.Character.HasCondition(characters.ConditionBleeding) {
				causeOfDeath = "bleeding out"
			}
		}
		if causeOfDeath == "" {
			causeOfDeath = "their own foolishness"
		}

		zone := user.Character.Zone
		region := ""
		if zCfg := rooms.GetZoneConfig(zone); zCfg != nil {
			region = zCfg.Region
		}
		worldevents.EmitWorldEvent(worldevents.WorldEvent{
			Type:         worldevents.PlayerDiedPvE,
			Significance: worldevents.Local,
			ZoneName:     zone,
			RegionName:   region,
			Description:  causeOfDeath,
		})
	}

	// If permadeath is enabled, do some extra bookkeeping
	if allowPenalties && bool(config.Death.PermaDeath) {

		if user.Character.ExtraLives > 0 {

			user.Character.ExtraLives--

		} else {

			user.EventLog.Add(`death`, fmt.Sprintf(`<ansi fg="username">%s</ansi> has <ansi fg="red-bold">PERMA-DIED</ansi>`, user.Character.Name))

			// Perma-died!!!
			textOut, _ := templates.Process("character/permadeath", nil, user.UserId)
			user.SendText(colorpatterns.ApplyColorPattern(textOut, `red`))

			// Unequip everything
			for _, itm := range user.Character.GetAllWornItems() {
				Remove(itm.Name(), user, room, flags)
			}
			// drop all items / gold
			Drop("all", user, room, flags)

			rooms.MoveToRoom(user.UserId, -1)

			user.Character = characters.New()

			return true, nil
		}

	}

	user.EventLog.Add(`death`, fmt.Sprintf(`<ansi fg="username">%s</ansi> has <ansi fg="red-bold">DIED</ansi>`, user.Character.Name))

	// Only apply penalties if they were above the threshold
	if allowPenalties {
		applyStatDecay(user, config)
		applySkillRust(user, config)
	}

	user.SendText(`<ansi fg="yellow">You feel weakened by the brush with death. (Type <ansi fg="command">help death</ansi> to learn more.)</ansi>`)

	user.Character.CancelBuffsWithFlag(buffs.All)
	user.Character.Aggro = nil
	user.Character.CastingState = nil

	// Set all pools to 5% of max so the player can regen up in the shadow realm
	// instead of arriving deep in the negatives and getting stuck.
	user.Character.Health = user.Character.HealthMax.Value / 20
	if user.Character.Health < 1 {
		user.Character.Health = 1
	}
	user.Character.Stamina = user.Character.StaminaMax.Value / 20
	if user.Character.Stamina < 1 {
		user.Character.Stamina = 1
	}
	user.Character.Conviction = user.Character.ConvictionMax.Value / 20
	if user.Character.Conviction < 1 {
		user.Character.Conviction = 1
	}
	events.AddToQueue(events.CharacterVitalsChanged{UserId: user.UserId})

	clear(user.Character.PlayerDamage)

	rooms.MoveToRoom(user.UserId, int(configs.GetSpecialRoomsConfig().DeathRecoveryRoom))

	if config.Death.CorpsesEnabled {
		room.AddCorpse(rooms.Corpse{
			UserId:       user.UserId,
			Character:    *user.Character,
			RoundCreated: currentRound,
		})
	}

	return true, nil
}

// applyStatDecay reduces Training on 1 random core stat as a permanent death penalty.
func applyStatDecay(user *users.UserRecord, config configs.GamePlay) {

	type statEntry struct {
		name     string
		desc     string
		training *int
	}

	stats := []statEntry{
		{`Strength`, `physical might`, &user.Character.Stats.Strength.Training},
		{`Dexterity`, `nimbleness`, &user.Character.Stats.Dexterity.Training},
		{`Perception`, `keen senses`, &user.Character.Stats.Perception.Training},
		{`Vitality`, `endurance`, &user.Character.Stats.Vitality.Training},
		{`Willpower`, `mental fortitude`, &user.Character.Stats.Willpower.Training},
		{`Charisma`, `force of personality`, &user.Character.Stats.Charisma.Training},
	}

	pick := stats[util.Rand(len(stats))]

	decayMin := int(config.Death.StatDecayMin)
	decayMax := int(config.Death.StatDecayMax)
	amount := decayMin
	if decayMax > decayMin {
		amount = decayMin + util.Rand(decayMax-decayMin+1)
	}

	*pick.training -= amount
	if *pick.training < 0 {
		*pick.training = 0
	}

	user.Character.Validate()

	user.SendText(fmt.Sprintf(`<ansi fg="red">The shadow of death saps your %s.</ansi>`, pick.desc))
	user.EventLog.Add(`death`, fmt.Sprintf(`Lost some <ansi fg="yellow">%s</ansi> training on death`, pick.name))
}

// applySkillRust decays ranks on up to SkillRustCount skills that haven't been used recently.
func applySkillRust(user *users.UserRecord, config configs.GamePlay) {

	recencyThreshold := int(config.Death.SkillRecencyThreshold)
	rustCount := int(config.Death.SkillRustCount)
	rustAmount := int(config.Death.SkillRustAmount)

	// Build list of eligible (unprotected) skills
	eligible := []string{}
	for skillName, rank := range user.Character.Skills {
		if rank <= 1 {
			continue // never reduce below 1
		}
		useCount := user.Character.GetSkillUseCount(skillName)
		if useCount >= recencyThreshold {
			continue // recently used — protected
		}
		eligible = append(eligible, skillName)
	}

	if len(eligible) == 0 {
		return
	}

	// Shuffle and pick up to rustCount
	for i := len(eligible) - 1; i > 0; i-- {
		j := util.Rand(i + 1)
		eligible[i], eligible[j] = eligible[j], eligible[i]
	}

	if rustCount > len(eligible) {
		rustCount = len(eligible)
	}

	for _, skillName := range eligible[:rustCount] {
		oldRank := user.Character.Skills[skillName]
		newRank := oldRank - rustAmount
		if newRank < 1 {
			newRank = 1
		}
		user.Character.Skills[skillName] = newRank

		user.SendText(fmt.Sprintf(`<ansi fg="red">Your %s feels rusty and diminished.</ansi>`, skillName))
		user.EventLog.Add(`death`, fmt.Sprintf(`Lost some <ansi fg="yellow">%s</ansi> skill on death`, skillName))
	}
}
