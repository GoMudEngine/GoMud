package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Assess lets a player evaluate a corpse to determine its essence and what
// undead forms it could support. It is the primary way to decide which
// raise spell to use before animating a corpse.
func Assess(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	rest = strings.TrimSpace(rest)
	if rest == `` {
		user.SendText(`Assess what?`)
		return true, nil
	}

	corpse, found := room.FindCorpse(rest)
	if !found {
		user.SendText(`You don't see those remains here.`)
		return true, nil
	}

	if corpse.Character.IsCharmed() {
		user.SendText(`These remains were bound to a master. The essence is spent — there is nothing left to raise.`)
		return true, nil
	}

	if !user.Character.TryCooldown(`assess`, "6 rounds") {
		user.SendText(
			fmt.Sprintf("You need to wait %d more rounds before you can assess again.", user.Character.GetCooldown(`assess`)),
		)
		return true, nil
	}

	// Sum all stat training values as a proxy for the creature's total power.
	stats := corpse.Character.Stats
	totalTraining := stats.Strength.Training +
		stats.Dexterity.Training +
		stats.Perception.Training +
		stats.Vitality.Training +
		stats.Willpower.Training +
		stats.Charisma.Training

	// Describe the power level without exposing raw numbers.
	var essenceDesc string
	switch {
	case totalTraining >= 500:
		essenceDesc = `overwhelming essence — death barely contains it`
	case totalTraining >= 300:
		essenceDesc = `immense essence — a truly mighty creature`
	case totalTraining >= 200:
		essenceDesc = `powerful essence — formidable in life`
	case totalTraining >= 120:
		essenceDesc = `substantial essence — a strong life force`
	case totalTraining >= 60:
		essenceDesc = `moderate essence`
	case totalTraining >= 30:
		essenceDesc = `faint residual essence`
	default:
		essenceDesc = `barely a trace of essence`
	}

	user.SendText(`You study the remains of <ansi fg="mob-corpse">` + corpse.Character.Name + `</ansi>.`)
	user.SendText(`You sense ` + essenceDesc + ` within.`)

	// List which undead types this corpse could support.
	var supported []string
	if totalTraining >= 30 {
		supported = append(supported, `skeleton`)
	}
	if totalTraining >= 60 {
		supported = append(supported, `zombie`)
	}
	if totalTraining >= 120 {
		supported = append(supported, `wraith`)
	}
	if totalTraining >= 200 {
		supported = append(supported, `spectre`)
	}
	if totalTraining >= 300 {
		supported = append(supported, `vampire`)
	}
	if totalTraining >= 500 {
		supported = append(supported, `golem`)
	}

	if len(supported) == 0 {
		user.SendText(`The essence is too faint to animate any form.`)
	} else {
		user.SendText(`It could sustain: ` + strings.Join(supported, `, `) + `.`)
	}

	// Trigger manifestation skill progression.
	user.Character.OnSkillUse(string(skills.Manifestation), user.UserId)

	return true, nil
}
