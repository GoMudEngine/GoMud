package skills

import (
	"strings"
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
	Cast        SkillTag = `cast`        // [LVL 1-4] Frostfang Magic Academy - ROOM 879
	DualWield   SkillTag = `dual-wield`  // [LVL 1-4] Fishermans house - ROOM 758
	Map         SkillTag = `map`         // [LVL 1-4] Frostwarden Rangers - ROOM 74
	Enchant     SkillTag = `enchant`     // TODO
	Peep        SkillTag = `peep`        // TODO
	Inspect     SkillTag = `inspect`     // TODO
	Portal      SkillTag = `portal`      // [LVL 1] Touch the obelisk in ROOOM 871
	Search      SkillTag = `search`      // [LVL 1-4] Frostwarden Rangers - ROOM 74
	Track       SkillTag = `track`       // [LVL 1-4] Frostwarden Rangers - ROOM 74
	Skulduggery SkillTag = `skulduggery` // [LVL 1-4] Thieves Den - ROOM 491
	Brawling    SkillTag = `brawling`    // [LVL 1-4] Soldiers Training Yard - ROOM 829
	Scribe      SkillTag = `scribe`      // [LVL 1-4] Dark Acolyte's Chamber - ROOM 160
	Protection  SkillTag = `protection`  // TODO
	Tame        SkillTag = `tame`        // [LVL 1-4] Give mushroom to fairie in ROOM 558, train in ROOM 830
	Trading     SkillTag = `trading`     // TODO

	// DOG combat & magic skills
	WeaponCombat  SkillTag = `weapon-combat`  // Melee attack & defense with weapons
	UnarmedCombat SkillTag = `unarmed-combat` // Fist/body attacks & defense, grappling
	RangedCombat  SkillTag = `ranged-combat`  // Bows, crossbows, thrown weapons
	Spellcasting  SkillTag = `spellcasting`   // All magic — offense & defense
	Psionics      SkillTag = `psionics`       // Mental powers — offense & defense

	// DOG non-combat skills
	FirstAid  SkillTag = `first-aid`  // Healing others, treating wounds, stabilizing
	Stealth   SkillTag = `stealth`    // Sneaking, hiding, avoiding detection
	Tracking  SkillTag = `tracking`   // Finding creatures/players, reading trails
	Bartering SkillTag = `bartering`  // Trade prices, negotiation, appraisal
	Foraging  SkillTag = `foraging`   // Gathering resources — herbs, wood, ore, food
)

var (
	allSkillNames = []SkillTag{}

	Professions = map[string][]SkillTag{
		"treasure hunter": {
			Map,
			Search,
			Peep,
			Inspect,
			Trading,
		},
		"assassin": {
			Skulduggery,
			DualWield,
			Track,
		},
		"explorer": {
			Map,
			Portal,
			Scribe,
		},
		"arcane scholar": {
			Enchant,
			Scribe,
			Inspect,
		},
		"warrior": {
			Brawling,
			DualWield,
		},
		"paladin": {
			Protection,
			Brawling,
		},
		"ranger": {
			Map,
			Search,
			Track,
		},
		"monster hunter": {
			Tame,
			Track,
		},
		"sorcerer": {
			Cast,
			Enchant,
		},
		"merchant": {
			Peep,
			Trading,
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
			if skillLevel > 4 {
				skillLevel = 4
			}
			totalSkill := (skillLevel * (skillLevel + 1)) / 2

			ranking.PointsToMax += 10.0 // Each skill has 4 levels, so possible 10 points per skill
			ranking.TotalPointsSpent += float64(totalSkill)
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
	//var highestSpend float64 = 0
	chosenProfessions := []string{}
	experienceName := ``

	for _, pRank := range rankData {

		if pRank.Completion == 0 {
			continue
		}

		if pRank.Completion > highestCompletion {
			highestCompletion = pRank.Completion
			//highestSpend = pRank.TotalPointsSpent
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

	// Register DOG combat, magic, and non-combat skills directly
	for _, sk := range []SkillTag{
		WeaponCombat, UnarmedCombat, RangedCombat, Spellcasting, Psionics,
		FirstAid, Stealth, Tracking, Bartering, Foraging,
	} {
		if _, ok := skillNameSet[sk]; !ok {
			skillNameSet[sk] = struct{}{}
			allSkillNames = append(allSkillNames, sk)
		}
	}

}
