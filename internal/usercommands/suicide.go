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

	// Dedupe double-fire: the combat loop can queue `suicide` multiple
	// times before the first one executes (damage from multiple
	// sources in the same round, or rapid re-checks). Skip the second
	// and subsequent fires in the same round to avoid stacking stat
	// decay + skill rust.
	if user.Character.LastSuicideRound == currentRound {
		return true, nil
	}
	user.Character.LastSuicideRound = currentRound

	if user.Character.HasBuffFlag(buffs.ReviveOnDeath) {

		user.Character.Health = user.Character.HealthMax.Value

		user.SendText(`You are revived in a shower of magical sparks!`)
		room.SendTextVisual(`<ansi fg="username">`+user.Character.Name+`</ansi> is suddenly revived in a shower of sparks!`, user.UserId)

		user.Character.CancelBuffsWithFlag(buffs.ReviveOnDeath)

		return true, nil
	}

	// Send a death msg to everyone in the room.
	room.SendTextVisual(
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

	// Emit a regional gossip event for PvE deaths (not PvP)
	if dmgCt == 0 {
		causeOfDeath := ""
		// Check if fighting a mob
		if user.Character.IsInCombat() && user.Character.EngagedTarget().MobInstanceId > 0 {
			if mob := mobs.GetInstance(user.Character.EngagedTarget().MobInstanceId); mob != nil {
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
			Significance: worldevents.Global,
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

	user.SendText(`<ansi fg="yellow">You feel weakened by the brush with death.</ansi>`)

	user.Character.CancelBuffsWithFlag(buffs.All)
	user.Character.Conditions = nil // Fix B: wipe all combat conditions (poison, bleeding, rally, etc.)
	user.Character.EndAggro()
	user.Character.CastingState = nil

	// Fix A: clear inbound aggro (mobs in the pre-respawn room
	// targeting this player) and companion aggro. Cleared BEFORE
	// MoveToRoom so companions arrive in home room with a blank
	// slate — they follow via TransportCompanions.
	for _, mobInstId := range room.GetMobs(rooms.FindFighting) {
		mob := mobs.GetInstance(mobInstId)
		if mob == nil || !mob.Character.IsInCombat() {
			continue
		}
		// Standard aggro targeting this player (UserId shape).
		if mob.Character.EngagedTarget().UserId == user.UserId {
			mob.Character.EndAggro()
			continue
		}
		// Spell-cast-shape aggro: check SpellInfo.TargetUserIds if
		// it contains this player. Abort the in-flight cast.
		if mob.Character.Aggro != nil && mob.Character.Aggro.Type == characters.SpellCast {
			for _, tid := range mob.Character.Aggro.SpellInfo.TargetUserIds {
				if tid == user.UserId {
					mob.Character.EndAggro()
					break
				}
			}
		}
	}
	for _, compInstId := range user.Character.GetCharmIds() {
		comp := mobs.GetInstance(compInstId)
		if comp == nil {
			continue
		}
		comp.Character.EndAggro()
	}

	// Respawn at a fraction of max pools. Acts as a brake on "death run"
	// strategies — the player has to recover before their next attempt at
	// whatever killed them. Default 0.05 (5%); configurable via
	// Death.RespawnPoolFraction. Minimum 1 in each pool so the respawn
	// doesn't immediately re-trigger the death check.
	respawnPct := float64(config.Death.RespawnPoolFraction)
	respawnHealth := int(float64(user.Character.HealthMax.Value) * respawnPct)
	if respawnHealth < 1 {
		respawnHealth = 1
	}
	respawnStamina := int(float64(user.Character.StaminaMax.Value) * respawnPct)
	if respawnStamina < 1 {
		respawnStamina = 1
	}
	respawnConviction := int(float64(user.Character.ConvictionMax.Value) * respawnPct)
	if respawnConviction < 1 {
		respawnConviction = 1
	}
	user.Character.Health = respawnHealth
	user.Character.Stamina = respawnStamina
	user.Character.Conviction = respawnConviction
	events.AddToQueue(events.CharacterVitalsChanged{UserId: user.UserId})

	clear(user.Character.PlayerDamage)

	// Check if player died in an instanced zone with ejected death policy
	if rooms.IsEphemeralRoomId(user.Character.RoomId) {
		if inst := rooms.GetInstanceRegistry().FindByRoomId(user.Character.RoomId); inst != nil {
			if inst.DeathPolicy == "ejected" {
				inst.RevokeAccess(user.UserId)
				user.SendText(`<ansi fg="red">You have been expelled from the instance. There is no return.</ansi>`)
			}
		}
	}

	user.SendText(`<ansi fg="yellow">Darkness swallows you. When you open your eyes, you are somewhere safe.</ansi>`)

	// Fix C: apply respawn grace buff. Prevents mobs from acquiring
	// aggro on the respawning player for N rounds. Knob:
	// Death.RespawnGraceRounds (default 3; 0 = disabled).
	if int(config.Death.RespawnGraceRounds) > 0 {
		user.Character.AddBuff(81, false)
	}

	// Resolve home room from player settings, falling back to default.
	rooms.MoveToRoom(user.UserId, user.Character.ResolveRespawnRoom())

	// Belt-and-suspenders: re-clear aggro after room move in case any
	// code path (e.g., mob combat round processing) assigned aggro
	// between our initial EndAggro above and the room move.
	user.Character.EndAggro()

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
