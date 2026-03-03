package skills

import (
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/stats"
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

// GetMutationTier returns the mutation tier prefix based on total mutation load.
func GetMutationTier(owned map[string]int) string {
	load := mutations.GetMutationLoad(owned)
	switch {
	case load >= 50:
		return "Exalted"
	case load >= 30:
		return "Ascendant"
	case load >= 15:
		return "Evolved"
	case load >= 1:
		return "Awakened"
	default:
		return ""
	}
}

// GetSkillTier returns the skill tier based on aggregate completion across all skills.
func GetSkillTier(allRanks map[string]int) string {
	const totalSkills = 17
	const softCap = 50.0
	maxTotal := totalSkills * softCap

	total := 0.0
	for _, rank := range allRanks {
		total += float64(rank)
	}

	pct := total / maxTotal
	switch {
	case pct >= 0.56:
		return "master"
	case pct >= 0.31:
		return "expert"
	case pct >= 0.16:
		return "journeyman"
	case pct >= 0.06:
		return "apprentice"
	case pct >= 0.01:
		return "novice"
	default:
		return "scrub"
	}
}

// statEntry is used for sorting stats by value.
type statEntry struct {
	name  string
	value int
}

// GetStatArchetype determines the character's archetype based on stat distribution.
func GetStatArchetype(s stats.Statistics) string {
	entries := []statEntry{
		{"Strength", s.Strength.Value},
		{"Dexterity", s.Dexterity.Value},
		{"Perception", s.Perception.Value},
		{"Vitality", s.Vitality.Value},
		{"Willpower", s.Willpower.Value},
		{"Charisma", s.Charisma.Value},
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].value > entries[j].value
	})

	top := entries[0]
	second := entries[1]
	third := entries[2]

	// Check if top stat is >10% above second → pure archetype
	threshold := 0.10
	if top.value > 0 && float64(top.value-second.value)/float64(top.value) > threshold {
		return pureArchetype(top.name)
	}

	// Check if top 2 are within 10% of each other AND both >10% above third → hybrid
	if top.value > 0 && float64(top.value-second.value)/float64(top.value) <= threshold {
		if third.value == 0 || float64(second.value-third.value)/float64(second.value) > threshold {
			return hybridArchetype(top.name, second.name)
		}
	}

	return "generalist"
}

func pureArchetype(stat string) string {
	switch stat {
	case "Strength":
		return "warrior"
	case "Dexterity":
		return "rogue"
	case "Perception":
		return "scout"
	case "Vitality":
		return "guardian"
	case "Willpower":
		return "channeler"
	case "Charisma":
		return "orator"
	default:
		return "generalist"
	}
}

func hybridArchetype(stat1, stat2 string) string {
	// Create a pair key that works regardless of order
	pair := stat1 + "+" + stat2
	// Also check reversed
	pairRev := stat2 + "+" + stat1

	hybrids := map[string]string{
		"Strength+Willpower":  "paladin",
		"Strength+Vitality":   "juggernaut",
		"Strength+Dexterity":  "duelist",
		"Dexterity+Perception": "ranger",
		"Willpower+Charisma":  "sage",
		"Perception+Willpower": "seer",
		"Vitality+Willpower":  "stoic",
	}

	if name, ok := hybrids[pair]; ok {
		return name
	}
	if name, ok := hybrids[pairRev]; ok {
		return name
	}
	// Unmapped hybrid pair → use pure archetype of the top stat
	return pureArchetype(stat1)
}

// GetTitle returns the three-part title: "<MutationTier> <SkillTier> <StatArchetype>".
// E.g., "Awakened expert paladin" or "scrub warrior".
func GetTitle(owned map[string]int, allRanks map[string]int, s stats.Statistics) string {
	mutTier := GetMutationTier(owned)
	skillTier := GetSkillTier(allRanks)
	archetype := GetStatArchetype(s)

	parts := []string{}
	if mutTier != "" {
		parts = append(parts, mutTier)
	}
	parts = append(parts, skillTier)
	parts = append(parts, archetype)

	return strings.Join(parts, " ")
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
