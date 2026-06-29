package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

type SkillsOptions struct {
	SkillList      map[string]int
	SkillCooldowns map[string]int
	SkillBlurbs    map[string]string
}

func Skills(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	allSkills := user.Character.GetSkills()
	allCooldowns := map[string]int{}
	blurbs := map[string]string{}
	for skillName := range allSkills {
		allCooldowns[skillName] = user.Character.GetCooldown(skillName)
		blurbs[skillName] = skills.SkillBlurb(skillName)
	}

	skillData := SkillsOptions{
		SkillList:      allSkills, // name to level
		SkillCooldowns: allCooldowns,
		SkillBlurbs:    blurbs,
	}

	skillTxt, _ := templates.Process("character/skills", skillData, user.UserId)
	user.SendText(messaging.CategorySystem, skillTxt)

	if rest == `extra` {
		user.SendText(messaging.CategorySystem, `<ansi fg="yellow">Cooldown Tracking:</ansi>`)
		for name, rnds := range user.Character.GetAllCooldowns() {
			user.SendText(messaging.CategorySystem, fmt.Sprintf(` <ansi fg="yellow">%s</ansi>: <ansi fg="red">%d</ansi>`, name, rnds))
		}
	}

	return true, nil
}
