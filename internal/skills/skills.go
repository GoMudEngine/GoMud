package skills

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

type SkillTag string

func (s SkillTag) String(subtag ...string) string {
	result := string(s)
	if len(subtag) > 0 {
		result += `:` + strings.Join(subtag, `:`)
	}
	return result
}

func (s SkillTag) Sub(subtag string) SkillTag {
	return SkillTag(string(s) + subtag)
}

const (
	// Cast is stubbed — full implementation in Phase 11 (fold-based magic system)
	Cast SkillTag = `cast`

	// DOG combat & magic skills
	WeaponCombat SkillTag = `weapon-combat`  // Melee attack & defense with weapons
	UnarmedCombat SkillTag = `unarmed-combat` // Fist/body attacks & defense, grappling
	RangedCombat  SkillTag = `ranged-combat`  // Bows, crossbows, thrown weapons
	Spellcasting  SkillTag = `spellcasting`   // All magic — offense & defense
	Rhetoric      SkillTag = `rhetoric`      // Conviction attacks — taunt, demoralize (Stage 34)

	// DOG non-combat skills
	FirstAid  SkillTag = `first-aid`  // Healing others, treating wounds, stabilizing
	Stealth   SkillTag = `stealth`    // Sneaking, hiding, avoiding detection
	Tracking  SkillTag = `tracking`   // Finding creatures/players, reading trails
	Bartering SkillTag = `bartering`  // Trade prices, negotiation, appraisal
	Foraging      SkillTag = `foraging`      // Gathering resources — herbs, wood, ore, food
	Blacksmithing SkillTag = `blacksmithing` // Metal weapons, armor, tools
	Alchemy       SkillTag = `alchemy`       // Potions, salves, medicines
	Tailoring     SkillTag = `tailoring`     // Cloth and leather goods
	Cooking       SkillTag = `cooking`       // Food preparation, buffs from meals
	Jewelcrafting SkillTag = `jewelcrafting` // Rings, pendants, gemwork
	Enchanting    SkillTag = `enchanting`   // Imbuing items with magic (31.6)
)

var (
	allSkillNames = []SkillTag{}

	Professions = map[string][]SkillTag{
		"warrior": {
			WeaponCombat,
			UnarmedCombat,
		},
		"ranger": {
			RangedCombat,
			Tracking,
		},
		"mage": {
			Spellcasting,
		},
		"healer": {
			Spellcasting,
			FirstAid,
		},
		"rogue": {
			Stealth,
			WeaponCombat,
		},
		"merchant": {
			Bartering,
		},
		"survivalist": {
			Foraging,
			Tracking,
		},
		"smith": {
			Blacksmithing,
		},
		"alchemist": {
			Alchemy,
		},
		"tailor": {
			Tailoring,
		},
		"cook": {
			Cooking,
		},
		"artificer": {
			Jewelcrafting,
			Enchanting,
		},
		"orator": {
			Rhetoric,
			Bartering,
		},
	}
)

type ProfessionRank struct {
	Profession       string
	ExperienceTitle  string
	TotalPointsSpent float64
	PointsToMax      float64
	Completion       float64
	Skills           []string
}

func SkillExists(sk string) bool {
	for _, skTag := range allSkillNames {
		if sk == string(skTag) {
			return true
		}
	}
	return false
}

func GetAllSkillNames() []SkillTag {
	return append([]SkillTag{}, allSkillNames...)
}

func GetProfessionRanks(allRanks map[string]int) []ProfessionRank {

	professionList := []ProfessionRank{}

	for professionName, skills := range Professions {

		ranking := ProfessionRank{Profession: professionName}

		for _, skillName := range skills {

			skillLevel := 0
			if rankVal, ok := allRanks[string(skillName)]; ok {
				skillLevel = rankVal
			}

			// DOG skill system: Skills can progress to ~50 (soft cap)
			// Profession completion is based on progress toward soft cap
			const skillSoftCap = 50.0

			ranking.PointsToMax += skillSoftCap // Each skill can reach ~50
			ranking.TotalPointsSpent += float64(skillLevel)
			ranking.Skills = append(ranking.Skills, string(skillName))
		}

		ranking.Completion = ranking.TotalPointsSpent / ranking.PointsToMax
		ranking.ExperienceTitle = GetExperienceLevel(ranking.Completion)

		professionList = append(professionList, ranking)
	}

	return professionList
}

func GetProfession(allRanks map[string]int) string {

	rankData := GetProfessionRanks(allRanks)

	var highestCompletion float64 = 0
	chosenProfessions := []string{}
	experienceName := ``

	for _, pRank := range rankData {

		if pRank.Completion == 0 {
			continue
		}

		if pRank.Completion > highestCompletion {
			highestCompletion = pRank.Completion
			chosenProfessions = []string{}
		}

		if pRank.Completion == highestCompletion {
			experienceName = pRank.ExperienceTitle
			chosenProfessions = append(chosenProfessions, pRank.Profession)
		}
	}

	if len(chosenProfessions) < 1 {
		return `scrub`
	}

	if len(experienceName) > 0 {
		experienceName = experienceName + ` `
	}

	if len(chosenProfessions) == len(Professions) {
		return experienceName + `demigod`
	}

	extra := ``
	if len(chosenProfessions) > 3 {
		chosenProfessions = chosenProfessions[0:3]
		extra = ` (and more)`
	}

	return experienceName + strings.Join(chosenProfessions, `/`) + extra
}

// Possible value is something like 1-10
func GetExperienceLevel(percentage float64) string {

	if percentage >= .9 { // avg level ~4
		return `expert`
	}

	if percentage >= .6 { // avg level 3
		return `journeyman`
	}

	if percentage >= .3 { // avg level 2
		return `apprentice`
	}

	if percentage >= .1 { // avg level 1
		return `novice`
	}

	return `scrub`
}

// SkillPrimaryStats maps each DOG skill to its primary governing stat.
// This stat is auto-tracked and progressed whenever the skill is used.
var SkillPrimaryStats = map[string]string{
	"weapon-combat":  "dexterity",
	"unarmed-combat": "dexterity",
	"ranged-combat":  "perception",
	"spellcasting":   "willpower",
	"first-aid":      "perception",
	"stealth":        "dexterity",
	"tracking":       "perception",
	"rhetoric":       "charisma",
	"bartering":      "charisma",
	"foraging":       "perception",
	"blacksmithing":  "strength",
	"alchemy":        "perception",
	"tailoring":      "dexterity",
	"cooking":        "perception",
	"jewelcrafting":  "dexterity",
	"enchanting":     "perception",
}

// GetSkillPrimaryStat returns the primary governing stat for a skill,
// or an empty string if none is defined.
func GetSkillPrimaryStat(skillName string) string {
	return SkillPrimaryStats[skillName]
}

// SkillProgressionMultipliers controls how fast each skill progresses.
// Combat skills fire many times per round, so they get a low multiplier.
// Utility skills are used less often, so they get a high multiplier.
var SkillProgressionMultipliers = map[SkillTag]float64{
	// Combat skills — fire multiple times per round
	WeaponCombat:  0.3,
	UnarmedCombat: 0.3,
	RangedCombat:  0.3,
	// Magic skills — moderate frequency
	Spellcasting: 0.5,
	// Social combat — moderate frequency
	Rhetoric: 0.5,
	Cast:         0.5,
	// Utility skills — used infrequently
	Tracking:  2.0,
	Bartering: 2.0,
	Foraging:      2.0,
	FirstAid:      2.0,
	Stealth:       2.0,
	Blacksmithing: 2.0,
	Alchemy:       2.0,
	Tailoring:     2.0,
	Cooking:       2.0,
	Jewelcrafting: 2.0,
	Enchanting:    2.0,
}

// GetSkillRankDescription converts a numeric skill level (1–50) to a qualitative tier name.
func GetSkillRankDescription(level int) string {
	switch {
	case level <= 0:
		return "unknown"
	case level <= 1:
		return "novice"
	case level <= 9:
		return "apprentice"
	case level <= 19:
		return "journeyman"
	case level <= 34:
		return "adept"
	case level <= 49:
		return "expert"
	default:
		return "master"
	}
}

// GetProgressionMultiplier returns the progression speed multiplier for a skill.
// Config overrides take priority; falls back to the hardcoded SkillProgressionMultipliers map.
// Returns 1.0 for any skill not in either source.
func GetProgressionMultiplier(skillName string) float64 {
	b := configs.GetBalanceConfig()
	if mult, ok := b.GetSkillProgressionMultiplier(skillName); ok {
		return mult
	}
	if mult, ok := SkillProgressionMultipliers[SkillTag(skillName)]; ok {
		return mult
	}
	return 1.0
}

func init() {

	skillNameSet := map[SkillTag]struct{}{}

	for _, skills := range Professions {
		for _, skillName := range skills {

			if _, ok := skillNameSet[skillName]; ok {
				continue
			}

			skillNameSet[skillName] = struct{}{}
			allSkillNames = append(allSkillNames, skillName)
		}
	}

	// Register all DOG skills directly (ensures cast and any not in professions are included)
	for _, sk := range []SkillTag{
		Cast,
		WeaponCombat, UnarmedCombat, RangedCombat, Spellcasting, Rhetoric,
		FirstAid, Stealth, Tracking, Bartering, Foraging,
		Blacksmithing, Alchemy, Tailoring, Cooking, Jewelcrafting, Enchanting,
	} {
		if _, ok := skillNameSet[sk]; !ok {
			skillNameSet[sk] = struct{}{}
			allSkillNames = append(allSkillNames, sk)
		}
	}

}
