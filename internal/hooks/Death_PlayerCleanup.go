package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// wirePlayerDeathCleanup subscribes to Life Dead transitions on
// player characters and runs the existing stat-decay + KD
// tracking + party-notification cleanup that used to live in
// suicide.go's inline death sequence.
//
// Chunk-2 Task 9 thins suicide.go to a ~30-line handler;
// this observer is where those cleanups now live.
func wirePlayerDeathCleanup(c *characters.Character) {
	c.Life.Inner().AfterTransition("player_death_cleanup",
		func(from, to life.State, r state.TransitionReason) {
			if from != life.Alive || to != life.Dead {
				return
			}
			// Only fire for player characters.
			if c.GetUserId() == 0 {
				return
			}
			u := users.GetByUserId(c.GetUserId())
			if u == nil {
				return
			}

			config := configs.GetGamePlayConfig()

			d, _ := c.Life.DeadData()

			// 1. KD stat tracking — mirrors the PlayerDamage-based
			//    logic in suicide.go lines 62-90.
			recordKDDeath(u, d)

			// 2. Stat decay + skill rust — preserved normal-death
			//    penalties. Guard matches suicide.go allowPenalties check.
			allowPenalties := u.Character.GetTotalSkillRanks() > int(config.Death.ProtectionSkillRanks)
			// chunk 4d: skip deprogression for submission outcomes that
			// intentionally leave the victim alive (subdue/cripple).
			if allowPenalties && !d.NoDeprogression {
				applyPlayerStatDecay(u, config)
				applyPlayerSkillRust(u, config)
			}

			// 3. Party notifications — targeted message to party
			//    members who are not in the dead player's room.
			notifyPartyOfPlayerDeath(c)
		})
}

// recordKDDeath mirrors the KD-tracking block from suicide.go
// (lines 62-90). It uses DeadData.DamageMap to identify player
// killers so AddPlayerDeath / AddPlayerKill can be recorded on
// both sides. Falls back to AddMobDeath when no player dealt
// damage.
func recordKDDeath(u *users.UserRecord, d life.DeadData) {
	if len(d.DamageMap) > 0 {
		// At least one player contributed damage — PvP death.
		u.Character.KD.AddPvpDeath()

		for uid := range d.DamageMap {
			killer := users.GetByUserId(uid)
			if killer == nil {
				continue
			}
			u.Character.KD.AddPlayerDeath(killer.UserId, killer.Character.Name)
			killer.Character.KD.AddPlayerKill(u.UserId, u.Character.Name)
		}
	} else {
		// PvE death (mob or environmental).
		u.Character.KD.AddMobDeath()
	}
}

// applyPlayerStatDecay reduces Training on 1 random core stat as a
// permanent death penalty. Ported from suicide.go applyStatDecay().
func applyPlayerStatDecay(u *users.UserRecord, config configs.GamePlay) {

	type statEntry struct {
		name     string
		desc     string
		training *int
	}

	stats := []statEntry{
		{`Strength`, `physical might`, &u.Character.Stats.Strength.Training},
		{`Dexterity`, `nimbleness`, &u.Character.Stats.Dexterity.Training},
		{`Perception`, `keen senses`, &u.Character.Stats.Perception.Training},
		{`Vitality`, `endurance`, &u.Character.Stats.Vitality.Training},
		{`Willpower`, `mental fortitude`, &u.Character.Stats.Willpower.Training},
		{`Charisma`, `force of personality`, &u.Character.Stats.Charisma.Training},
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

	u.Character.Validate()

	u.SendText(messaging.CategoryDeath, fmt.Sprintf(`<ansi fg="red">The shadow of death saps your %s.</ansi>`, pick.desc))
	u.EventLog.Add(`death`, fmt.Sprintf(`Lost some <ansi fg="yellow">%s</ansi> training on death`, pick.name))
}

// applyPlayerSkillRust decays ranks on up to SkillRustCount skills
// that haven't been used recently. Ported from suicide.go
// applySkillRust().
func applyPlayerSkillRust(u *users.UserRecord, config configs.GamePlay) {

	recencyThreshold := int(config.Death.SkillRecencyThreshold)
	rustCount := int(config.Death.SkillRustCount)
	rustAmount := int(config.Death.SkillRustAmount)

	// Build list of eligible (unprotected) skills.
	eligible := []string{}
	for skillName, rank := range u.Character.Skills {
		if rank <= 1 {
			continue // never reduce below 1
		}
		useCount := u.Character.GetSkillUseCount(skillName)
		if useCount >= recencyThreshold {
			continue // recently used — protected
		}
		eligible = append(eligible, skillName)
	}

	if len(eligible) == 0 {
		return
	}

	// Shuffle and pick up to rustCount.
	for i := len(eligible) - 1; i > 0; i-- {
		j := util.Rand(i + 1)
		eligible[i], eligible[j] = eligible[j], eligible[i]
	}

	if rustCount > len(eligible) {
		rustCount = len(eligible)
	}

	for _, skillName := range eligible[:rustCount] {
		oldRank := u.Character.Skills[skillName]
		newRank := oldRank - rustAmount
		if newRank < 1 {
			newRank = 1
		}
		u.Character.Skills[skillName] = newRank

		u.SendText(messaging.CategoryDeath, fmt.Sprintf(`<ansi fg="red">Your %s feels rusty and diminished.</ansi>`, skillName))
		u.EventLog.Add(`death`, fmt.Sprintf(`Lost some <ansi fg="yellow">%s</ansi> skill on death`, skillName))
	}
}

// notifyPartyOfPlayerDeath sends a targeted death notice to party
// members who are NOT in the dead player's room (those in the room
// already see the room-level death message from suicide.go).
func notifyPartyOfPlayerDeath(c *characters.Character) {
	party := parties.Get(c.GetUserId())
	if party == nil {
		return
	}

	msg := fmt.Sprintf(
		`<ansi fg="magenta-bold">***</ansi> Your party member <ansi fg="username">%s</ansi> has <ansi fg="red-bold">DIED!</ansi> <ansi fg="magenta-bold">***</ansi>%s`,
		c.Name,
		term.CRLFStr,
	)

	for _, memberId := range party.GetMembers() {
		if memberId == c.GetUserId() {
			continue
		}
		member := users.GetByUserId(memberId)
		if member == nil {
			continue
		}
		// Skip members in the same room — they see the room-broadcast
		// death message already sent by the death sequence.
		if member.Character.RoomId == c.RoomId {
			continue
		}
		member.SendText(messaging.CategoryDeath, msg)
	}
}

func init() {
	characters.OnCharacterCreated(wirePlayerDeathCleanup)
}
