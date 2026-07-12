// Package mutations implements the Chrysalis mutation system for DOGMud.
// Mutations are acquired through sustained combat and provide pro and con effects.
// Adding new mutations requires only a YAML file in the mutations/ data directory.
//
// Phase 24 adds multi-effect mutations (Pros/Cons lists), a points-budget acquisition
// system based on mutation load, and a conflict system to prevent incompatible combos.
package mutations

import (
	"fmt"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/pkg/errors"
)

// MutationEffect describes a single pro or con effect on a mutation.
type MutationEffect struct {
	Type   string  `yaml:"type"`   // effect type (see Effect Types in the project docs)
	Target string  `yaml:"target"` // stat name or "" where not applicable
	Value  float64 `yaml:"value"`
}

// MutationSpec is the data-driven definition of a single mutation loaded from YAML.
type MutationSpec struct {
	MutationId  string `yaml:"mutationid"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Rarity      int    `yaml:"rarity"` // 1=common … 10=very rare
	Visual      string `yaml:"visual"` // appended to character look desc

	// Legacy single-effect fields (backward compat — migrated into Pros/Cons during Validate)
	Pro MutationEffect `yaml:"pro"`
	Con MutationEffect `yaml:"con"`

	// Multi-effect fields (Phase 24). Rarer mutations can have multiple pros/cons.
	Pros []MutationEffect `yaml:"pros,omitempty"`
	Cons []MutationEffect `yaml:"cons,omitempty"`

	// Conflicts lists mutation IDs that cannot coexist with this mutation.
	// If a character owns any conflicting mutation, this one is excluded from the pool.
	Conflicts []string `yaml:"conflicts,omitempty"`

	// MoonFlavor gates a mutation to a moon "bucket" (1–4): the bloom pool only
	// offers it when the current moon bucket matches. 0 = always eligible. Used
	// by the Reflect Skin family so the moon at acquisition picks which flavor
	// you receive.
	MoonFlavor int `yaml:"moon_flavor,omitempty"`

	// RequiresBodyParts lists canonical body-part tags from
	// species.CanonicalBodyParts. Empty/nil = body-agnostic.
	// Boot-time validation is added in chunk-2.5 Task 6 (or see
	// species.IsCanonicalBodyPart for the canonical set).
	RequiresBodyParts []string `yaml:"requires_body_parts,omitempty"`


	// Clusters lists the design-side playstyle clusters this mutation
	// belongs to (empty = universal/generalist). Steers acquisition drift.
	Clusters []string `yaml:"clusters,omitempty"`

	// Pole is "body", "belief", or "" (neutral). Drives the opposition:
	// deep Body shrinks the Conviction pool; deep Belief degrades gear.
	Pole string `yaml:"pole,omitempty"`

	// Prerequisites lists mutations (with min level) that must be owned
	// before this one can be acquired. Gates apex/spine mutations.
	Prerequisites []MutationPrereq `yaml:"prerequisites,omitempty"`
}

// Id implements fileloader.Loadable.
func (m *MutationSpec) Id() string { return m.MutationId }

// Filepath implements fileloader.Loadable. Returns just the filename (no directory).
func (m *MutationSpec) Filepath() string {
	return util.FilePath(m.MutationId + ".yaml")
}

// Validate implements fileloader.Loadable.
// Migrates legacy singular Pro/Con into Pros/Cons slices for backward compatibility.
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

	// Migrate legacy single Pro into Pros list if present and not already in Pros
	if m.Pro.Type != "" {
		found := false
		for _, p := range m.Pros {
			if p.Type == m.Pro.Type && p.Target == m.Pro.Target {
				found = true
				break
			}
		}
		if !found {
			m.Pros = append(m.Pros, m.Pro)
		}
	}

	// Migrate legacy single Con into Cons list if present and not already in Cons
	if m.Con.Type != "" {
		found := false
		for _, c := range m.Cons {
			if c.Type == m.Con.Type && c.Target == m.Con.Target {
				found = true
				break
			}
		}
		if !found {
			m.Cons = append(m.Cons, m.Con)
		}
	}

	return nil
}

// Package-level registry, populated by LoadMutationFiles.
var allMutations map[string]*MutationSpec

// LoadMutationFiles reads all YAML files from the mutations/ data directory and
// populates the in-memory registry.  Called once at startup from main.go.
func LoadMutationFiles() {
	start := time.Now()

	dataPath := string(configs.GetFilePathsConfig().DataFiles) + `/mutations`
	tmpAll, err := fileloader.LoadAllFlatFiles[string, *MutationSpec](dataPath)
	if err != nil {
		panic(errors.Wrap(err, `filepath: `+dataPath))
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

// AllSpecs returns every loaded mutation spec. Read-only convenience
// for boot-time validation passes (e.g. archetype_pull validation);
// callers must not mutate the returned specs.
func AllSpecs() []*MutationSpec {
	out := make([]*MutationSpec, 0, len(allMutations))
	for _, spec := range allMutations {
		out = append(out, spec)
	}
	return out
}

// ─── Load & Conflict system (Phase 24.1) ───────────────────────────────────────

// GetMutationLoad returns the total "weight" of all owned mutations.
// Load = sum(rarity * level) for each owned mutation.
// Used by the acquisition formula: higher load → slower acquisition.
func GetMutationLoad(owned map[string]int) float64 {
	var load float64
	for id, level := range owned {
		spec := GetMutation(id)
		if spec == nil {
			continue
		}
		load += float64(spec.Rarity) * float64(level)
	}
	return load
}

// HasConflict returns true if candidateId conflicts with any mutation in owned.
// Checks both directions: candidate's Conflicts list AND each owned mutation's
// Conflicts list.
func HasConflict(owned map[string]int, candidateId string) bool {
	candidate := GetMutation(candidateId)
	if candidate == nil {
		return false
	}

	// Check if any owned mutation is listed in the candidate's conflicts
	for _, conflictId := range candidate.Conflicts {
		if _, has := owned[conflictId]; has {
			return true
		}
	}

	// Check if the candidate is listed in any owned mutation's conflicts
	for ownedId := range owned {
		ownedSpec := GetMutation(ownedId)
		if ownedSpec == nil {
			continue
		}
		for _, conflictId := range ownedSpec.Conflicts {
			if conflictId == candidateId {
				return true
			}
		}
	}

	return false
}

// calcRarityBonus computes a pool weight reduction based on the player's
// average mutation level. Higher average levels shift new discoveries
// toward rarer mutations.
//
//	avgLevel 1 → bonus 0 (normal weights)
//	avgLevel 2 → bonus 1
//	avgLevel 3 → bonus 2
//	avgLevel 4 → bonus 3
func calcRarityBonus(owned map[string]int) int {
	if len(owned) == 0 {
		return 0
	}
	totalLevels := 0
	for _, level := range owned {
		totalLevels += level
	}
	avgLevel := totalLevels / len(owned) // integer division = floor
	bonus := avgLevel - 1
	if bonus < 0 {
		bonus = 0
	}
	return bonus
}

// GetWeightedPool builds a weighted slice of mutation IDs suitable for random selection.
// Each mutation appears (11 - Rarity) times so rarer mutations are less likely.
// Mutations already owned or conflicting with owned mutations are excluded.
// sp filters out mutations whose body-part requirements the species does not meet.
// Pass nil for sp to disable body-part filtering (fail-open for players/unknown species).
//
// Returns mutation IDs rather than spec pointers to preserve
// RollAcquisition compatibility.
func GetWeightedPool(owned map[string]int, sp *species.Species) []string {
	return GetWeightedPoolWithFloor(owned, sp, 0)
}

// GetWeightedPoolWithFloor is GetWeightedPool restricted to mutations at or
// above minRarity (0 = no floor). Used by the pinnacle remort potion and
// the Seething Prism's worn mutation tick.
func GetWeightedPoolWithFloor(owned map[string]int, sp *species.Species, minRarity int) []string {
	// Rarity uplift: reduce common mutation weights for advanced players
	rarityBonus := calcRarityBonus(owned)
	moonBucket := gametime.CurrentMoonFlavorBucket()

	pool := make([]string, 0, len(allMutations)*5)
	for id, spec := range allMutations {
		if spec.Rarity < minRarity {
			continue
		}
		if _, has := owned[id]; has {
			continue
		}
		if HasConflict(owned, id) {
			continue
		}
		if !spec.CanApplyTo(sp) {
			continue
		}
		// Moon-gated flavor (Reflect Skin): only the current-bucket variant.
		if spec.MoonFlavor != 0 && spec.MoonFlavor != moonBucket {
			continue
		}
		weight := 11 - spec.Rarity - rarityBonus
		if weight < 1 {
			weight = 1
		}
		for i := 0; i < weight; i++ {
			pool = append(pool, id)
		}
	}
	return pool
}

// CanApplyTo reports whether this mutation's body-part requirements
// are satisfied by the given species. Empty requirements pass for
// any embodied or unembodied species.
func (s *MutationSpec) CanApplyTo(sp *species.Species) bool {
	if sp == nil {
		return true // defensive fail-open
	}
	return sp.HasAllBodyParts(s.RequiresBodyParts)
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
// L1 → 1.0×, L2 → MutationLevel2Multiplier×, L3 → MutationLevel3Multiplier×,
// L4 → MutationLevel4Multiplier×. Any other value defaults to 1.0.
func LevelMultiplier(level int) float64 {
	b := configs.GetBalanceConfig()
	switch level {
	case 2:
		return float64(b.MutationLevel2Multiplier)
	case 3:
		return float64(b.MutationLevel3Multiplier)
	case 4:
		return float64(b.MutationLevel4Multiplier)
	default:
		return 1.0
	}
}

// LevelDescription renders a mutation depth as descriptive language —
// player-facing text never shows raw level numbers (no-hard-numbers
// rule). Single source of truth for the status template's
// mutationLevel helper and the mutations command.
func LevelDescription(level int) string {
	switch level {
	case 1:
		return "minor"
	case 2:
		return "moderate"
	case 3:
		return "major"
	case 4:
		return "extreme"
	default:
		return "unknown"
	}
}

// TotalMutationEvents returns the sum of all mutation levels owned.
// Kept for backward compatibility; GetMutationLoad is preferred for acquisition.
func TotalMutationEvents(owned map[string]int) int {
	total := 0
	for _, level := range owned {
		total += level
	}
	return total
}

// CanDeepen returns true if any owned mutation is below MutationMaxLevel.
func CanDeepen(owned map[string]int) bool {
	maxLevel := int(configs.GetBalanceConfig().MutationMaxLevel)
	for _, level := range owned {
		if level < maxLevel {
			return true
		}
	}
	return false
}

// RollDeepening picks a random mutation id that is below MutationMaxLevel.
// Returns "" if all mutations are already at max level or owned is empty.
func RollDeepening(owned map[string]int) string {
	maxLevel := int(configs.GetBalanceConfig().MutationMaxLevel)
	var candidates []string
	for id, level := range owned {
		if level < maxLevel {
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

// DampenBonus pulls a mutation MULTIPLIER's bonus toward the 1.0 baseline by
// `factor` (0 = remove the bonus entirely, 1 = untouched). Values at or below
// 1.0 (no bonus / a penalty) are returned unchanged. Used by the #22 crash-site
// suppression aura at the mutation call sites where the bearer is Dampened.
func DampenBonus(raw, factor float64) float64 {
	if raw <= 1.0 {
		return raw
	}
	return 1.0 + (raw-1.0)*factor
}

// sumEffects totals all matching pro and con effects across owned mutations,
// scaling each value by LevelMultiplier for the mutation's current level.
// If target is "" it matches effects regardless of their Target field.
// Iterates Pros and Cons slices (which include migrated legacy Pro/Con).
func sumEffects(owned map[string]int, effectType, target string) float64 {
	var total float64
	for id, level := range owned {
		spec := GetMutation(id)
		if spec == nil {
			continue
		}
		mult := LevelMultiplier(level)
		for _, p := range spec.Pros {
			if p.Type == effectType && (target == "" || p.Target == target) {
				total += p.Value * mult
			}
		}
		for _, c := range spec.Cons {
			if c.Type == effectType && (target == "" || c.Target == target) {
				total += c.Value * mult
			}
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

// GetConvictionResistance returns the total conviction_damage_reduction fraction (0.0–1.0).
// Used by GetConvictionMitigation() — converted to percentage points (* 100).
func GetConvictionResistance(owned map[string]int) float64 {
	return sumEffects(owned, "conviction_damage_reduction", "")
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

// ─── Stage 24.4: Flag system ────────────────────────────────────────────────

// GetMutationFlags returns all active flags granted by owned mutations.
// Flags are derived from effects with type "flag" where Target is the flag name.
func GetMutationFlags(owned map[string]int) map[string]bool {
	flags := make(map[string]bool)
	for id, level := range owned {
		spec := GetMutation(id)
		if spec == nil {
			continue
		}
		_ = LevelMultiplier(level) // flags don't scale, but keep the pattern
		for _, p := range spec.Pros {
			if p.Type == "flag" && p.Target != "" {
				flags[p.Target] = true
			}
		}
		for _, c := range spec.Cons {
			if c.Type == "flag" && c.Target != "" {
				flags[c.Target] = true
			}
		}
	}
	return flags
}

// HasMutationFlag returns true if any owned mutation grants the specified flag.
func HasMutationFlag(owned map[string]int, flag string) bool {
	for id := range owned {
		spec := GetMutation(id)
		if spec == nil {
			continue
		}
		for _, p := range spec.Pros {
			if p.Type == "flag" && p.Target == flag {
				return true
			}
		}
		for _, c := range spec.Cons {
			if c.Type == "flag" && c.Target == flag {
				return true
			}
		}
	}
	return false
}

// GetCompanionReserveRank returns the highest owned rank among mutations that
// carry a "companion_reserve_reduction" effect (0 if none). The reduction
// magnitude is computed by the caller from config knobs (linear per rank) —
// NOT via sumEffects — because companion reservation reduction is a
// character-economy knob, not a stat effect scaled by the mutation-level curve.
func GetCompanionReserveRank(owned map[string]int) int {
	rank := 0
	for id, level := range owned {
		spec := GetMutation(id)
		if spec == nil {
			continue
		}
		for _, p := range spec.Pros {
			if p.Type == "companion_reserve_reduction" && level > rank {
				rank = level
			}
		}
	}
	return rank
}

// GetConditionalHealthRegen returns the health_regen_if_lit value if biomeIsLit is true, else 0.
func GetConditionalHealthRegen(owned map[string]int, biomeIsLit bool) int {
	if !biomeIsLit {
		return 0
	}
	return int(sumEffects(owned, "health_regen_if_lit", ""))
}

// GetHealthRegenMultiplier returns the net health_regen_multiplier bonus.
// Apply as: regen = int(float64(regen) * (1.0 + GetHealthRegenMultiplier(m)))
func GetHealthRegenMultiplier(owned map[string]int) float64 {
	return sumEffects(owned, "health_regen_multiplier", "")
}

// GetConditionalHealthRegenMultiplier returns the health_regen_if_lit_multiplier
// bonus if the biome is lit, else 0.
func GetConditionalHealthRegenMultiplier(owned map[string]int, biomeIsLit bool) float64 {
	if !biomeIsLit {
		return 0
	}
	return sumEffects(owned, "health_regen_if_lit_multiplier", "")
}

// ─── Stage 24.2: New effect helpers ─────────────────────────────────────────

// GetDodgeModifier returns the flat dodge score bonus/penalty from mutations.
// Apply by adding to the dodge defense score in combat.
func GetDodgeModifier(owned map[string]int) float64 {
	return sumEffects(owned, "dodge_modifier", "")
}

// GetStealthBonus returns the flat bonus to sneak score from mutations.
// Applied additively to the sneak score before opposed rolls.
func GetStealthBonus(owned map[string]int) float64 {
	return sumEffects(owned, "stealth_bonus", "")
}

// GetDamageMultiplier returns the net outgoing physical damage multiplier.
// Apply as: dmg = int(float64(dmg) * (1.0 + GetDamageMultiplier(m)))
func GetDamageMultiplier(owned map[string]int) float64 {
	return sumEffects(owned, "damage_multiplier", "")
}

// GetSpellPowerMultiplier returns the net spell_power bonus from mutations
// (Ether Gland, Corvid Brain, …). Apply as:
//
//	dmg = dmg * (1.0 + GetSpellPowerMultiplier(m))
func GetSpellPowerMultiplier(owned map[string]int) float64 {
	return sumEffects(owned, "spell_power", "")
}

// GetOnHitBuffs returns the buff ids that owned mutations apply to a struck
// target on a landed melee/natural hit (effect type "on_hit_buff", Value = buff id).
func GetOnHitBuffs(owned map[string]int) []int {
	var out []int
	for id := range owned {
		spec := GetMutation(id)
		if spec == nil {
			continue
		}
		for _, p := range spec.Pros {
			if p.Type == "on_hit_buff" && p.Value > 0 {
				out = append(out, int(p.Value))
			}
		}
	}
	return out
}

// GetReflectRiderBuffs returns the buff ids that owned mutations apply to an
// attacker when this defender's reflect fires (effect type "on_reflect_buff",
// Value = buff id). Mirrors GetOnHitBuffs but the buff lands on whoever struck
// you — the Ironhide Reflect Skin flavors (Molten burn, Frostbite chill,
// Voltaic shock) carry these.
func GetReflectRiderBuffs(owned map[string]int) []int {
	var out []int
	for id := range owned {
		spec := GetMutation(id)
		if spec == nil {
			continue
		}
		for _, p := range spec.Pros {
			if p.Type == "on_reflect_buff" && p.Value > 0 {
				out = append(out, int(p.Value))
			}
		}
	}
	return out
}

// GetShoutAmp returns the total shout amplification fraction from owned
// mutations (effect type "shout_amp"). Rally/war-cry magnitude and duration are
// scaled by (1.0 + this). Booming Lungs (Zealot) is the source.
func GetShoutAmp(owned map[string]int) float64 {
	return sumEffects(owned, "shout_amp", "")
}

// GetMovementSpeedModifier returns the net movement speed modifier.
// Negative = faster (reduced stamina cost), positive = slower.
// Apply as: cost = int(float64(cost) * (1.0 - GetMovementSpeedModifier(m)))
func GetMovementSpeedModifier(owned map[string]int) float64 {
	return sumEffects(owned, "movement_speed", "")
}

// GetHealthRegen returns the passive HP regen per auto-heal tick.
func GetHealthRegen(owned map[string]int) int {
	return int(sumEffects(owned, "health_regen", ""))
}

// GetSkillProgressionMultiplier returns the net skill progression multiplier.
// Apply as: chance *= (1.0 + GetSkillProgressionMultiplier(m))
func GetSkillProgressionMultiplier(owned map[string]int) float64 {
	return sumEffects(owned, "skill_progression_multiplier", "")
}

// GetStatProgressionMultiplier returns the net stat progression multiplier.
// Apply as: chance *= (1.0 + GetStatProgressionMultiplier(m))
func GetStatProgressionMultiplier(owned map[string]int) float64 {
	return sumEffects(owned, "stat_progression_multiplier", "")
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

// ─── Stage 2.2a: Incorporeal mutation helpers ────────────────────────────────

// GetGearEffectivenessLoss returns the total fraction (0.0–1.0)
// by which all equipment effects (stat mods, weapon damage,
// mitigation values) should be reduced for this character.
// Computed with RAW level multiplication — does NOT call
// LevelMultiplier. This produces a linear progression across
// ranks 1-4 of 0.25/0.50/0.75/1.00, matching the design intent
// for percentage-loss effects. Clamped to [0.0, 1.0].
func GetGearEffectivenessLoss(owned map[string]int) float64 {
	loss := 0.0
	for id, level := range owned {
		spec := GetMutation(id)
		if spec == nil {
			continue
		}
		// Either side of the pros/cons split can declare this
		// effect; the type name implies a downside but we sum
		// both for completeness.
		for _, c := range spec.Cons {
			if c.Type == "gear_effectiveness_loss" {
				loss += c.Value * float64(level)
			}
		}
		for _, p := range spec.Pros {
			if p.Type == "gear_effectiveness_loss" {
				loss += p.Value * float64(level)
			}
		}
	}
	if loss < 0 {
		loss = 0
	} else if loss > 1 {
		loss = 1
	}
	return loss
}

// GearEffectivenessMultiplier returns the multiplier consumers apply to
// gear-derived values (1.0 = full effectiveness, 0.0 = none). Combines the
// per-mutation gear_effectiveness_loss (incorporeal) with the Belief-pole
// opposition decay (deep casters render worn gear ornamental).
func GearEffectivenessMultiplier(owned map[string]int) float64 {
	return (1.0 - GetGearEffectivenessLoss(owned)) * BeliefGearScale(owned)
}

// GetPhysicalDefenseBonus returns the total bonus added to the
// defender's roll margin for physical-channel attacks. Uses
// standard LevelMultiplier scaling.
func GetPhysicalDefenseBonus(owned map[string]int) float64 {
	return sumEffects(owned, "physical_defense_bonus", "")
}

// HasSpec returns true if a mutation with the given id is loaded.
// Used by cross-package validation.
func HasSpec(id string) bool {
	_, ok := allMutations[id]
	return ok
}

// ValidateBodyPartTags scans all loaded mutation specs and panics
// on any unknown body-part tag in RequiresBodyParts.
func ValidateBodyPartTags() {
	for _, spec := range allMutations {
		for _, tag := range spec.RequiresBodyParts {
			if !species.IsCanonicalBodyPart(tag) {
				panic(fmt.Sprintf(
					"mutation %q: unknown requires_body_parts tag %q (canonical: %v)",
					spec.MutationId, tag, species.CanonicalBodyParts))
			}
		}
	}
}
