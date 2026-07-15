package achievements

// Category is the grouping an achievement belongs to.
const (
	CategoryCombat      = "combat"
	CategoryExploration = "exploration"
	CategoryWealth      = "wealth"
	CategoryProgression = "progression"
	CategoryQuests      = "quests"
)

// validCategories is the allowed set (loader validation).
var validCategories = map[string]bool{
	CategoryCombat: true, CategoryExploration: true, CategoryWealth: true,
	CategoryProgression: true, CategoryQuests: true,
}

// Trigger is the fixed-vocabulary unlock condition for an achievement.
type Trigger struct {
	Type      string `yaml:"type"`
	Threshold int    `yaml:"threshold,omitempty"` // count/value types
	Stat      string `yaml:"stat,omitempty"`      // stat_reached (a primary stat name or "any")
	Skill     string `yaml:"skill,omitempty"`     // skill_reached (a skill name or "any")
	Token     string `yaml:"token,omitempty"`     // quest_completed (a quest token)
}

// Definition is one authored achievement.
type Definition struct {
	Id          string  `yaml:"id"`
	Name        string  `yaml:"name"`
	Description string  `yaml:"description"`
	Category    string  `yaml:"category"`
	Points      int     `yaml:"points"`
	Trigger     Trigger `yaml:"trigger"`
}

// registry holds the loaded definitions, keyed by id, plus a stable ordering.
var (
	registry      = map[string]Definition{}
	registryOrder []string
)

// All returns the loaded definitions in load order.
func All() []Definition {
	out := make([]Definition, 0, len(registryOrder))
	for _, id := range registryOrder {
		out = append(out, registry[id])
	}
	return out
}

// Get returns a definition by id.
func Get(id string) (Definition, bool) { d, ok := registry[id]; return d, ok }

// PointsFor sums the points of the given unlocked achievement ids.
func PointsFor(ids map[string]uint64) int {
	total := 0
	for id := range ids {
		if d, ok := registry[id]; ok {
			total += d.Points
		}
	}
	return total
}
