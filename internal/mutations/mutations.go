// Package mutations implements the Chrysalis mutation system for DOGMud (Stage 12.1).
// Mutations are acquired through sustained combat and provide both a pro and a con effect.
// Adding new mutations requires only a YAML file in the mutations/ data directory.
package mutations

import (
	"fmt"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Constants governing the mutation acquisition system.
const (
	MutationBaseProgress  = 50.0 // progress needed for first acquisition attempt
	MutationProgressScale = 1.5  // each subsequent mutation needs BaseProgress * Scale^n more
	MutationMaxCount      = 5    // max mutations per character
	MutationMaxLevel      = 3    // maximum level a single mutation can reach (Stage 12.2)
	MutationProgressGain  = 1.0  // progress added per combat round
)

// MutationEffect describes a single pro or con effect on a mutation.
type MutationEffect struct {
	Type   string  `yaml:"type"`   // effect type (see Effect Types in the project docs)
	Target string  `yaml:"target"` // stat name or "" where not applicable
	Value  float64 `yaml:"value"`
}

// MutationSpec is the data-driven definition of a single mutation loaded from YAML.
type MutationSpec struct {
	MutationId  string         `yaml:"mutationid"`
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Rarity      int            `yaml:"rarity"`  // 1=common … 10=very rare
	Visual      string         `yaml:"visual"`  // appended to character look desc (Stage 12.2)
	Pro         MutationEffect `yaml:"pro"`
	Con         MutationEffect `yaml:"con"`
}

// Id implements fileloader.Loadable.
func (m *MutationSpec) Id() string { return m.MutationId }

// Filepath implements fileloader.Loadable. Returns just the filename (no directory).
func (m *MutationSpec) Filepath() string {
	return util.FilePath(m.MutationId + ".yaml")
}

// Validate implements fileloader.Loadable.
func (m *MutationSpec) Validate() error {
	if m.MutationId == "" {
		return fmt.Errorf("mutationid cannot be empty")
	}
	if m.Name == "" {
		return fmt.Errorf("mutation %q: name cannot be empty", m.MutationId)
	}
	if m.Rarity < 1 || m.Rarity > 10 {
		return fmt.Errorf("mutation %q: rarity must be 1–10, got %d", m.MutationId, m.Rarity)
	}
	return nil
}

// Package-level registry, populated by LoadMutationFiles.
var allMutations map[string]*MutationSpec

// LoadMutationFiles reads all YAML files from the mutations/ data directory and
// populates the in-memory registry.  Called once at startup from main.go.
func LoadMutationFiles() {
	start := time.Now()

	tmpAll, err := fileloader.LoadAllFlatFiles[string, *MutationSpec](
		string(configs.GetFilePathsConfig().DataFiles) + `/mutations`,
	)
	if err != nil {
		panic(err)
	}

	allMutations = tmpAll
	mudlog.Info("mutations.LoadMutationFiles()", "loadedCount", len(allMutations), "Time Taken", time.Since(start))
}

// GetMutation returns the MutationSpec for a given id, or nil if not found.
func GetMutation(id string) *MutationSpec {
	if allMutations == nil {
		return nil
	}
	return allMutations[id]
}

// GetAll returns the full mutation registry map.
func GetAll() map[string]*MutationSpec {
	return allMutations
}

// GetWeightedPool builds a weighted slice of mutation IDs suitable for random selection.
// Each mutation appears (11 - Rarity) times so rarer mutations are less likely.
// Mutations already in owned are excluded.
func GetWeightedPool(owned map[string]int) []string {
	pool := make([]string, 0, len(allMutations)*5)
	for id, spec := range allMutations {
		if _, has := owned[id]; has {
			continue
		}
		weight := 11 - spec.Rarity
		if weight < 1 {
			weight = 1
		}
		for i := 0; i < weight; i++ {
			pool = append(pool, id)
		}
	}
	return pool
}

// RollAcquisition picks a random mutation ID from the weighted pool.
// Returns "" if pool is empty.
func RollAcquisition(pool []string) string {
	if len(pool) == 0 {
		return ""
	}
	return pool[util.Rand(len(pool))]
}

// ─── Effect helper functions ───────────────────────────────────────────────────
// All accept the character's mutations map (mutationId → level).

// LevelMultiplier returns the effect scaling factor for a given mutation level.
// L1 → 1.0×, L2 → 1.5×, L3 → 2.0×. Any other value defaults to 1.0.
func LevelMultiplier(level int) float64 {
	switch level {
	case 2:
		return 1.5
	case 3:
		return 2.0
	default:
		return 1.0
	}
}

// TotalMutationEvents returns the sum of all mutation levels owned.
// This is the "events so far" value used by the deepening threshold curve.
func TotalMutationEvents(owned map[string]int) int {
	total := 0
	for _, level := range owned {
		total += level
	}
	return total
}

// CanDeepen returns true if any owned mutation is below MutationMaxLevel.
func CanDeepen(owned map[string]int) bool {
	for _, level := range owned {
		if level < MutationMaxLevel {
			return true
		}
	}
	return false
}

// RollDeepening picks a random mutation id that is below MutationMaxLevel.
// Returns "" if all mutations are already at max level or owned is empty.
func RollDeepening(owned map[string]int) string {
	var candidates []string
	for id, level := range owned {
		if level < MutationMaxLevel {
			candidates = append(candidates, id)
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	return candidates[util.Rand(len(candidates))]
}

// GetAdrenalSurgeBonus returns the level-scaled conditional_damage_low_hp bonus.
// Returns 0 if adrenaline-surge is not owned.
func GetAdrenalSurgeBonus(owned map[string]int) float64 {
	return sumEffects(owned, "conditional_damage_low_hp", "")
}

// sumEffects totals all matching pro and con effects across owned mutations,
// scaling each value by LevelMultiplier for the mutation's current level.
// If target is "" it matches effects regardless of their Target field.
func sumEffects(owned map[string]int, effectType, target string) float64 {
	var total float64
	for id, level := range owned {
		spec := GetMutation(id)
		if spec == nil {
			continue
		}
		mult := LevelMultiplier(level)
		if spec.Pro.Type == effectType && (target == "" || spec.Pro.Target == target) {
			total += spec.Pro.Value * mult
		}
		if spec.Con.Type == effectType && (target == "" || spec.Con.Target == target) {
			total += spec.Con.Value * mult
		}
	}
	return total
}

// GetStatMultiplier returns the net stat_multiplier for a given stat name.
// Apply as: adj = int(float64(adj) * (1.0 + GetStatMultiplier(m, "strength")))
func GetStatMultiplier(owned map[string]int, stat string) float64 {
	return sumEffects(owned, "stat_multiplier", stat)
}

// GetStatFlat returns the net stat_flat bonus for a given stat name.
// Apply to Mods before Recalculate().
func GetStatFlat(owned map[string]int, stat string) int {
	return int(sumEffects(owned, "stat_flat", stat))
}

// GetNaturalArmor returns the total natural_armor bonus (physical damage reduction).
func GetNaturalArmor(owned map[string]int) int {
	return int(sumEffects(owned, "natural_armor", ""))
}

// GetHealthMultiplier returns the net health_multiplier.
// Apply as: hpMax = int(float64(hpMax) * (1.0 + GetHealthMultiplier(m)))
func GetHealthMultiplier(owned map[string]int) float64 {
	return sumEffects(owned, "health_multiplier", "")
}

// GetStaminaRegenMultiplier returns the net stamina_regen_multiplier.
// Apply as: base = int(float64(base) * (1.0 + GetStaminaRegenMultiplier(m)))
func GetStaminaRegenMultiplier(owned map[string]int) float64 {
	return sumEffects(owned, "stamina_regen_multiplier", "")
}

// GetNaturalWeaponBonus returns the total natural_weapon flat damage bonus.
func GetNaturalWeaponBonus(owned map[string]int) float64 {
	return sumEffects(owned, "natural_weapon", "")
}

// GetMagicalResistance returns the total magical_damage_reduction fraction (0.0–1.0).
// Apply as: dmg = int(float64(dmg) * (1.0 - GetMagicalResistance(m)))
func GetMagicalResistance(owned map[string]int) float64 {
	return sumEffects(owned, "magical_damage_reduction", "")
}

// GetConvictionCostMultiplier returns the net conviction_cost_multiplier.
// Apply as: cost = int(float64(cost) * (1.0 + GetConvictionCostMultiplier(m)))
func GetConvictionCostMultiplier(owned map[string]int) float64 {
	return sumEffects(owned, "conviction_cost_multiplier", "")
}

// GetAggroMagnet returns the total aggro_magnet multiplier.
// Used by predatory mobs to weight target selection toward this character.
func GetAggroMagnet(owned map[string]int) float64 {
	return sumEffects(owned, "aggro_magnet", "")
}

// HasMutation returns true if the character owns the given mutation.
func HasMutation(owned map[string]int, id string) bool {
	_, ok := owned[id]
	return ok
}

// GetMutationLevel returns the level of a mutation (0 if not owned).
func GetMutationLevel(owned map[string]int, id string) int {
	return owned[id]
}

// IsAdrenalSurgeActive returns true when the character has the adrenaline-surge
// mutation and their current HP is below 25% of max.
func IsAdrenalSurgeActive(owned map[string]int, currentHP, maxHP int) bool {
	if !HasMutation(owned, "adrenaline-surge") {
		return false
	}
	if maxHP <= 0 {
		return false
	}
	return currentHP*4 < maxHP
}
