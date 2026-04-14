// Package enchantments implements the Chrysalis enchanting system for DOGMud.
// Enchantments are living mutations bound to objects — they grow through use,
// feed on the wearer's pool reserves, and visually mutate items over time.
// Adding new enchantments requires only a YAML file in the enchantments/ data directory.
package enchantments

import (
	"fmt"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/statmods"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/pkg/errors"
)

// TierDef describes a single enchantment tier's effects and appearance.
type TierDef struct {
	Tier              int            `yaml:"tier"`
	ReservePct        float64        `yaml:"reserve_pct"`        // fraction of pool max reserved (e.g. 0.02 = 2%)
	Effects           map[string]int `yaml:"effects"`            // damage_bonus, dr_bonus, statmod keys
	Adjective         string         `yaml:"adjective"`          // visual adjective applied to item
	DescriptionSuffix string         `yaml:"description_suffix"` // appended to item description
	TierUpMessage     string         `yaml:"tier_up_message"`    // message sent on tier advancement
}

// EnchantmentDef is the data-driven definition of an enchantment loaded from YAML.
type EnchantmentDef struct {
	EnchantId   string    `yaml:"enchantid"`
	Name        string    `yaml:"name"`
	ReservePool string    `yaml:"reserve_pool"` // health|stamina|conviction
	TargetType  string    `yaml:"target_type"`  // weapon|body|head|neck|ring|legs|feet
	Tiers       []TierDef `yaml:"tiers"`
}

// Id implements fileloader.Loadable.
func (e *EnchantmentDef) Id() string { return e.EnchantId }

// Filepath implements fileloader.Loadable.
func (e *EnchantmentDef) Filepath() string {
	return util.FilePath(e.EnchantId + ".yaml")
}

// Validate implements fileloader.Loadable.
func (e *EnchantmentDef) Validate() error {
	if e.EnchantId == "" {
		return fmt.Errorf("enchantid cannot be empty")
	}
	if e.Name == "" {
		return fmt.Errorf("enchantment %q: name cannot be empty", e.EnchantId)
	}
	if e.ReservePool == "" {
		return fmt.Errorf("enchantment %q: reserve_pool cannot be empty", e.EnchantId)
	}
	if len(e.Tiers) == 0 {
		return fmt.Errorf("enchantment %q: must have at least one tier", e.EnchantId)
	}
	return nil
}

// Package-level registry, populated by LoadEnchantmentFiles.
var allEnchantments map[string]*EnchantmentDef

// LoadEnchantmentFiles reads all YAML files from the enchantments/ data directory
// and populates the in-memory registry. Called once at startup from main.go.
func LoadEnchantmentFiles() {
	start := time.Now()

	dataPath := string(configs.GetFilePathsConfig().DataFiles) + `/enchantments`
	tmpAll, err := fileloader.LoadAllFlatFiles[string, *EnchantmentDef](dataPath)
	if err != nil {
		panic(errors.Wrap(err, `filepath: `+dataPath))
	}

	allEnchantments = tmpAll
	mudlog.Info("enchantments.LoadEnchantmentFiles()", "loadedCount", len(allEnchantments), "Time Taken", time.Since(start))
}

// GetEnchantment returns the EnchantmentDef for a given id, or nil if not found.
func GetEnchantment(id string) *EnchantmentDef {
	if allEnchantments == nil {
		return nil
	}
	return allEnchantments[id]
}

// GetAll returns the full enchantment registry map.
func GetAll() map[string]*EnchantmentDef {
	return allEnchantments
}

// GetTierReservePct returns the reserve_pct for the given enchantment at the
// given tier. If hands >= 2 (two-handed weapon), the reserve is doubled.
// Returns 0 if the enchantment or tier is not found.
func GetTierReservePct(enchantType string, tier int, hands ...int) float64 {
	def := GetEnchantment(enchantType)
	if def == nil {
		return 0
	}
	if tier < 0 || tier >= len(def.Tiers) {
		return 0
	}
	pct := def.Tiers[tier].ReservePct
	if len(hands) > 0 && hands[0] >= 2 {
		pct *= 2.0
	}
	return pct
}

// ApplyTier rewrites the item's override spec based on the enchantment tier.
// It updates damage bonuses, DR bonuses, stat mods, adjectives, and description.
func ApplyTier(item *items.Item, def *EnchantmentDef, tier int) {
	if tier < 0 || tier >= len(def.Tiers) {
		return
	}

	tierDef := def.Tiers[tier]

	// Detect 2-handed weapons — double effects and reserve
	twoHandMult := 1
	baseSpec := items.GetItemSpec(item.ItemId)
	if baseSpec != nil && baseSpec.Hands >= 2 {
		twoHandMult = 2
	}

	// Ensure we have an override spec to work with
	var newSpec items.ItemSpec
	if item.Spec != nil {
		newSpec = *item.Spec
	} else {
		if baseSpec == nil {
			return
		}
		newSpec = *baseSpec
	}

	// Reset numeric fields to base spec to avoid stacking from previous tiers.
	// For StatMods: start from base, then merge in any affix bonuses from the
	// item's override spec (instanced zone random affixes, etc.) that aren't
	// in the base. This preserves affix bonuses while preventing enchant stacking.
	if baseSpec != nil {
		newSpec.Damage = baseSpec.Damage
		newSpec.DamageReduction = baseSpec.DamageReduction
		newSpec.DamageMultiplier = baseSpec.DamageMultiplier
		newSpec.PhysicalMitigation = baseSpec.PhysicalMitigation
		newSpec.MagicalMitigation = baseSpec.MagicalMitigation
		newSpec.ConvictionMitigation = baseSpec.ConvictionMitigation

		// Preserve affix stat bonuses: start from base, add any extra mods
		// that the item's override had beyond the base spec.
		baseMods := copyStatMods(baseSpec.StatMods)
		if baseMods == nil {
			baseMods = make(statmods.StatMods)
		}
		if item.Spec != nil && len(item.Spec.StatMods) > 0 {
			for k, v := range item.Spec.StatMods {
				baseVal := 0
				if baseSpec.StatMods != nil {
					baseVal = baseSpec.StatMods.Get(k)
				}
				extra := v - baseVal
				if extra != 0 {
					baseMods.Add(k, extra)
				}
			}
		}
		newSpec.StatMods = baseMods
	}

	// Apply tier effects (doubled for 2H weapons)
	for effectKey, effectVal := range tierDef.Effects {
		scaledVal := effectVal * twoHandMult
		switch effectKey {
		case "damage_bonus":
			if newSpec.Damage.BaseDamage > 0 {
				newSpec.Damage.BaseDamage += scaledVal
			} else {
				newSpec.Damage.BonusDamage += scaledVal
			}
		case "damage_multiplier_bonus":
			// Int value interpreted as hundredths: 10 = +0.10
			newSpec.DamageMultiplier += float64(scaledVal) / 100.0
		case "dr_bonus":
			newSpec.DamageReduction += scaledVal
		case "physical_mitigation_bonus":
			newSpec.PhysicalMitigation += scaledVal
		case "magical_mitigation_bonus":
			newSpec.MagicalMitigation += scaledVal
		case "conviction_mitigation_bonus":
			newSpec.ConvictionMitigation += scaledVal
		default:
			// Treat as a stat mod (e.g. "willpower_statmod", "return_damage", "lifesteal_pct")
			newSpec.StatMods.Add(effectKey, scaledVal)
		}
	}

	if newSpec.Damage.BaseDamage == 0 && newSpec.Damage.DiceRoll != "" {
		newSpec.Damage.FormatDiceRoll()
	}
	newSpec.AutoCalculateValue()

	item.Spec = &newSpec

	// Update adjective: remove any previous enchant adjective, add current
	cleanAdjectives := make([]string, 0, len(item.Adjectives))
	for _, adj := range item.Adjectives {
		if !isEnchantAdjective(adj, def) {
			cleanAdjectives = append(cleanAdjectives, adj)
		}
	}
	if tierDef.Adjective != "" {
		cleanAdjectives = append(cleanAdjectives, tierDef.Adjective)
	}
	item.Adjectives = cleanAdjectives
}

// isEnchantAdjective checks if an adjective belongs to any tier of the given enchantment.
func isEnchantAdjective(adj string, def *EnchantmentDef) bool {
	for _, t := range def.Tiers {
		if t.Adjective == adj {
			return true
		}
	}
	return false
}

// copyStatMods creates a shallow copy of a StatMods map.
func copyStatMods(src statmods.StatMods) statmods.StatMods {
	if src == nil {
		return nil
	}
	dst := make(statmods.StatMods, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// StripEnchantment removes all enchantment data from an item, restoring it to base state.
func StripEnchantment(item *items.Item) {
	item.EnchantType = ""
	item.EnchantTier = 0
	item.EnchantUses = 0
	item.ReservePool = ""
	item.Spec = nil

	// Remove enchant adjectives — since we lost the def, just clear all adjectives
	// that aren't the base item adjectives. Simplest: clear them all since base items
	// typically have no adjectives.
	item.Adjectives = nil
}
