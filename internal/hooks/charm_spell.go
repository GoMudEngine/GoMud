package hooks

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// resolveCharmSpell handles the opposed-roll charm logic that was previously
// in charm.js's onMagic function. Called after the spell cast completes.
//
// Attack = charisma + manifestation_skill * 25
// Defense = target willpower + round(target stat training total * 0.10)
// Aggro penalties applied to attack score before the roll.
//
// Returns true on success (mob charmed), false on failure.
func resolveCharmSpell(user *users.UserRecord, targetMob *mobs.Mob, room *rooms.Room) bool {

	ch := user.Character

	// ── 1. Charm immunity ──────────────────────────────────────────────
	if targetMob.CharmImmune {
		user.SendText(`That creature's mind is impervious to charm.`)
		return false
	}

	// ── 2. Companion cap ───────────────────────────────────────────────
	if len(ch.Companions) >= ch.GetMaxCompanions() {
		user.SendText(`You cannot maintain any more companions.`)
		return false
	}

	// ── 3. Build attack score ──────────────────────────────────────────
	attackScore := ch.Stats.Charisma.ValueAdj +
		ch.GetSkillLevel(skills.Manifestation)*25

	// ── 4. Build defense score ─────────────────────────────────────────
	ts := targetMob.Character.Stats
	statTrainingTotal := ts.Strength.Training +
		ts.Dexterity.Training +
		ts.Perception.Training +
		ts.Vitality.Training +
		ts.Willpower.Training +
		ts.Charisma.Training

	defenseScore := ts.Willpower.ValueAdj +
		int(math.Round(float64(statTrainingTotal)*0.10))

	// ── 5. Aggro penalty ───────────────────────────────────────────────
	if targetMob.Character.Aggro != nil {
		if targetMob.Character.Aggro.UserId == user.UserId {
			// Fighting the caster — steepest penalty
			attackScore = int(math.Round(float64(attackScore) * 0.75))
		} else {
			// Fighting someone else — moderate penalty
			attackScore = int(math.Round(float64(attackScore) * 0.85))
		}
	}

	// ── 6. Opposed roll ────────────────────────────────────────────────
	success, _, _, _ := dice.OpposedRollStat(
		float64(attackScore), float64(defenseScore))

	targetName := targetMob.Character.Name

	if success {
		// Charm permanently
		targetMob.Character.Charm(user.UserId, 99999, "")
		targetMob.Character.EndAggro()
		ch.TrackCharmed(targetMob.InstanceId, true)

		// Register as companion
		info := characters.CompanionInfo{
			MobId:      int(targetMob.MobId),
			InstanceId: targetMob.InstanceId,
			SourceType: characters.CompanionCharmed,
			Name:       targetName,
			BaseName:   targetName,
			AutoAssist: true,
		}
		if !ch.AddCompanion(info) {
			user.SendText(`You cannot maintain any more companions.`)
			return false
		}

		// Clear aggro from existing companions toward the new mob
		for _, charmId := range ch.GetCharmIds() {
			if charmId == targetMob.InstanceId {
				continue
			}
			if companion := mobs.GetInstance(charmId); companion != nil {
				if companion.Character.Aggro != nil &&
					companion.Character.Aggro.MobInstanceId == targetMob.InstanceId {
					companion.Character.EndAggro()
				}
			}
		}

		// Clear the owner's own aggro if targeting the new mob
		if ch.Aggro != nil && ch.Aggro.MobInstanceId == targetMob.InstanceId {
			ch.EndAggro()
		}

		user.SendText(fmt.Sprintf(
			`<ansi fg="cyan">%s's eyes glaze as your will takes hold. It is yours.</ansi>`,
			targetName))
		sendVisualRoomText(room, fmt.Sprintf(
			`<ansi fg="cyan"><ansi fg="username">%s</ansi> bends %s to their will!</ansi>`,
			user.Character.Name, targetName), user.UserId)

		return true
	}

	// ── Failure ────────────────────────────────────────────────────────
	user.SendText(fmt.Sprintf(
		`<ansi fg="yellow">You reach for %s's mind, but its will is iron. The spell shatters.</ansi>`,
		targetName))
	sendVisualRoomText(room, fmt.Sprintf(
		`<ansi fg="yellow"><ansi fg="username">%s</ansi>'s charm spell breaks against %s's resolve!</ansi>`,
		user.Character.Name, targetName), user.UserId)

	return false
}
