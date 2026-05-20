package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Title(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	char := user.Character

	// Compute title parts
	fullTitle := skills.GetTitle(char.Mutations, char.GetAllSkillRanks(), char.Stats)
	mutTier := skills.GetMutationTier(char.Mutations)
	mutLoad := mutations.GetMutationLoad(char.Mutations)
	skillTier := skills.GetSkillTier(char.GetAllSkillRanks())
	archetype := skills.GetStatArchetype(char.Stats)

	// Skill completion — descriptive instead of percentage
	allRanks := char.GetAllSkillRanks()
	totalRanks := 0
	for _, r := range allRanks {
		totalRanks += r
	}
	const maxTotal = 17 * 50
	skillPct := float64(totalRanks) / float64(maxTotal) * 100
	var skillProgress string
	switch {
	case skillPct < 10:
		skillProgress = "fledgling"
	case skillPct < 25:
		skillProgress = "developing"
	case skillPct < 50:
		skillProgress = "seasoned"
	case skillPct < 75:
		skillProgress = "accomplished"
	case skillPct < 90:
		skillProgress = "masterful"
	default:
		skillProgress = "legendary"
	}

	// Mutation load — descriptive instead of number
	var mutLoadDesc string
	switch {
	case mutLoad <= 0:
		mutLoadDesc = "untouched"
	case mutLoad < 2:
		mutLoadDesc = "faint"
	case mutLoad < 4:
		mutLoadDesc = "noticeable"
	case mutLoad < 7:
		mutLoadDesc = "heavy"
	default:
		mutLoadDesc = "overwhelming"
	}

	// Mutation tier progress
	mutTierDisplay := mutTier
	if mutTierDisplay == "" {
		mutTierDisplay = "none"
	}

	// Stat lean - show top 3 stats
	type statBar struct {
		Name  string
		Value int
		Bar   string
	}

	statVals := []statBar{
		{"Strength", char.Stats.Strength.Value, ""},
		{"Dexterity", char.Stats.Dexterity.Value, ""},
		{"Perception", char.Stats.Perception.Value, ""},
		{"Vitality", char.Stats.Vitality.Value, ""},
		{"Willpower", char.Stats.Willpower.Value, ""},
		{"Charisma", char.Stats.Charisma.Value, ""},
	}

	// Sort descending by value
	for i := 0; i < len(statVals); i++ {
		for j := i + 1; j < len(statVals); j++ {
			if statVals[j].Value > statVals[i].Value {
				statVals[i], statVals[j] = statVals[j], statVals[i]
			}
		}
	}

	// Generate bars for top 3
	maxStat := statVals[0].Value
	if maxStat < 1 {
		maxStat = 1
	}
	for i := 0; i < 3 && i < len(statVals); i++ {
		pct := float64(statVals[i].Value) / float64(maxStat)
		barFull, barEmpty := util.ProgressBar(pct, 20)
		statVals[i].Bar = fmt.Sprintf(`<ansi fg="green">%s</ansi><ansi fg="black-bold">%s</ansi>`, barFull, barEmpty)
	}

	type TitleDisplay struct {
		FullTitle string
		MutTier   string
		MutLoad   string
		SkillTier string
		SkillPct  string
		Archetype string
		TopStats  []statBar
	}

	data := TitleDisplay{
		FullTitle: fullTitle,
		MutTier:   mutTierDisplay,
		MutLoad:   mutLoadDesc,
		SkillTier: skillTier,
		SkillPct:  skillProgress,
		Archetype: archetype,
		TopStats:  statVals[:3],
	}

	titleTxt, _ := templates.Process("character/title", data, user.UserId)
	user.SendText(messaging.CategorySystem, titleTxt)

	return true, nil
}
