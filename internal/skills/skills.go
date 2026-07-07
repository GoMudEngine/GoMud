package skills

import (
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/casing"
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
	// DOG combat & magic skills
	WeaponCombat  SkillTag = `weapon-combat`  // Melee attack & defense with weapons
	UnarmedCombat SkillTag = `unarmed-combat` // Fist/body attacks & defense, grappling
	RangedCombat  SkillTag = `ranged-combat`  // Bows, crossbows, pistols — aimed shots (Perception)
	Spellcasting  SkillTag = `spellcasting`   // All magic — offense & defense
	Rhetoric      SkillTag = `rhetoric`       // Conviction attacks — taunt, demoralize (Stage 34)

	// DOG non-combat skills
	Skullduggery SkillTag = `skullduggery` // Sneaking, stealing, lockpicking, surprise attacks
	Search    SkillTag = `search`     // Finding hidden objects, creatures, and resources
	Bartering SkillTag = `bartering`  // Trade prices, negotiation, appraisal
	Blacksmithing SkillTag = `blacksmithing` // Metal weapons, armor, tools
	Alchemy       SkillTag = `alchemy`       // Potions, salves, medicines
	Tailoring     SkillTag = `tailoring`     // Cloth and leather goods
	Cooking       SkillTag = `cooking`       // Food preparation, buffs from meals
	Jewelcrafting SkillTag = `jewelcrafting` // Rings, pendants, gemwork
	Enchanting    SkillTag = `enchanting`   // Imbuing items with magic (31.6)
	Salvage       SkillTag = `salvage`     // Breaking down items for materials
	Manifestation SkillTag = `manifestation` // Companion summoning, charming, necromancy
)

// skillBlurbs gives a short, player-facing "what it does" line for each skill,
// shown in the skills list so players know what a skill covers. Notably,
// foraging (the `forage` command) trains Search, so Search's blurb says so —
// players who forage were confused that there was no separate "foraging" skill.
var skillBlurbs = map[SkillTag]string{
	WeaponCombat:  "Melee with weapons -- attack and defense.",
	UnarmedCombat: "Fists, grappling, and unarmed defense.",
	RangedCombat:  "Bows, crossbows, and firearms (aimed with Perception).",
	Spellcasting:  "All magic, offense and defense.",
	Rhetoric:      "Conviction attacks -- taunt and demoralize.",
	Skullduggery:  "Sneaking, stealing, lockpicking, and ambush.",
	Search:        "Finding hidden things -- and foraging the wild for resources.",
	Bartering:     "Better prices and appraisal when you trade.",
	Blacksmithing: "Forging metal weapons, armor, and tools.",
	Alchemy:       "Brewing potions, salves, and medicines.",
	Tailoring:     "Crafting cloth and leather goods.",
	Cooking:       "Preparing food and the buffs good meals give.",
	Jewelcrafting: "Rings, pendants, and gemwork.",
	Enchanting:    "Imbuing items with magic.",
	Salvage:       "Breaking items down into materials.",
	Manifestation: "Summoning companions, charming, and necromancy.",
}

// SkillBlurb returns the short description for a skill name, or "" if unknown.
func SkillBlurb(name string) string {
	return skillBlurbs[SkillTag(name)]
}

var (
	allSkillNames = []SkillTag{}

	Professions = map[string][]SkillTag{
		"warrior": {
			WeaponCombat,
			UnarmedCombat,
		},
		"hunter": {
			RangedCombat,
			Search,
		},
		"ranger": {
			Search,
		},
		"mage": {
			Spellcasting,
		},
		"healer": {
			Spellcasting,
		},
		"rogue": {
			Skullduggery,
			WeaponCombat,
		},
		"merchant": {
			Bartering,
		},
		"survivalist": {
			Search,
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
		"scavenger": {
			Search,
			Salvage,
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
		return "exalted"
	case load >= 30:
		return "ascendant"
	case load >= 15:
		return "evolved"
	case load >= 1:
		return "awakened"
	default:
		return ""
	}
}

// GetSkillTier returns the skill tier based on aggregate completion across all skills.
func GetSkillTier(allRanks map[string]int) string {
	// totalSkills is a tuning denominator for the title-threshold curve, not
	// the actual skill count (currently 16 DOG skills). Raise it only when
	// the curve needs rebalancing, not every time a skill is added.
	const totalSkills = 17
	const softCap = 50.0
	maxTotal := totalSkills * softCap

	total := 0.0
	for _, rank := range allRanks {
		total += float64(rank)
	}

	// demigod: every profession mastered (all canonical skills at the master
	// rank, 50 = soft cap). The all-professions-mastered capstone.
	if allProfessionsMastered(allRanks) {
		return "demigod"
	}

	pct := total / maxTotal
	switch {
	case pct >= 0.85:
		return "grandmaster"
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

// allProfessionsMastered reports whether every canonical DOG skill (the keys of
// SkillPrimaryStats) is at the master rank (50 = the soft cap) or above. This is
// the "demigod" all-professions-mastered capstone condition for GetSkillTier.
func allProfessionsMastered(allRanks map[string]int) bool {
	const masterRank = 50
	for skill := range SkillPrimaryStats {
		if allRanks[skill] < masterRank {
			return false
		}
	}
	return true
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

	return casing.Title(strings.Join(parts, " "))
}

// SkillPrimaryStats maps each DOG skill to its primary governing stat.
// This stat is auto-tracked and progressed whenever the skill is used.
var SkillPrimaryStats = map[string]string{
	"weapon-combat":  "dexterity",
	"unarmed-combat": "dexterity",
	"ranged-combat":  "perception",
	"spellcasting":   "willpower",
	"skullduggery":   "dexterity",
	"search":         "perception",
	"rhetoric":       "charisma",
	"bartering":      "charisma",
	"blacksmithing":  "strength",
	"alchemy":        "perception",
	"tailoring":      "dexterity",
	"cooking":        "perception",
	"jewelcrafting":  "dexterity",
	"enchanting":     "perception",
	"salvage":        "perception",
	"manifestation":  "charisma",
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
	// Ranged combat — one aimed shot per action (reload on cooldown), so it
	// fires less often than melee; moderate rate like the other per-action skills.
	RangedCombat: 0.5,
	// Magic skills — moderate frequency
	Spellcasting: 0.5,
	// Social combat — moderate frequency
	Rhetoric: 0.5,
	// Companion management — moderate use frequency
	Manifestation: 0.5,
	// Utility skills — used infrequently
	Search:    2.0,
	Bartering: 2.0,
	Skullduggery:  2.0,
	Blacksmithing: 2.0,
	Alchemy:       2.0,
	Tailoring:     2.0,
	Cooking:       2.0,
	Jewelcrafting: 2.0,
	Enchanting:    2.0,
	Salvage:       2.0,
}

// GetSkillRankDescription converts a numeric skill level to a qualitative
// tier name. Skills soft-cap at 50 (master); grandmaster tier rewards the
// slow progression above the cap.
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
	case level <= 64:
		return "master"
	default:
		return "grandmaster"
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

	// Register all DOG skills directly (ensures any not in professions are included)
	for _, sk := range []SkillTag{
		WeaponCombat, UnarmedCombat, RangedCombat, Spellcasting, Rhetoric,
		Skullduggery, Search, Bartering,
		Blacksmithing, Alchemy, Tailoring, Cooking, Jewelcrafting, Enchanting, Salvage,
		Manifestation,
	} {
		if _, ok := skillNameSet[sk]; !ok {
			skillNameSet[sk] = struct{}{}
			allSkillNames = append(allSkillNames, sk)
		}
	}

}
