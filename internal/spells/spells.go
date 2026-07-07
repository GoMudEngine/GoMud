package spells

import (
	"fmt"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/casing"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/textutil"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/pkg/errors"
)

type SpellType string

type SpellData struct {
	SpellId     string     `yaml:"spellid,omitempty"`
	Name        string     `yaml:"name,omitempty"`
	Aliases     []string   `yaml:"aliases,omitempty"` // short single-word invocation forms (primary first)
	Description string     `yaml:"description,omitempty"`
	Type        SpellType  `yaml:"type,omitempty"`
	Schools     []string   `yaml:"schools,omitempty"`    // Can have multiple school tags
	Categories  []string   `yaml:"categories,omitempty"` // AI categorization: self_defense, self_offense, etc. Free-form strings.
	Cost        int        `yaml:"cost,omitempty"`    // Conviction cost
	HealthCost  int        `yaml:"healthcost,omitempty"` // Optional Health cost for life-force magic
	WaitRounds  int        `yaml:"waitrounds,omitempty"`
	Difficulty  int        `yaml:"difficulty,omitempty"` // Augments final success chance by this %
	PrimaryStat         string `yaml:"primarystat,omitempty"`          // Stat used for spell rolls and progression
	BaseFolds           int    `yaml:"base_folds,omitempty"`           // 0 = default to 4
	TargetDefenseType   string `yaml:"target_defense_type,omitempty"`  // "physical", "mental", "" = none
	ComponentTag        string `yaml:"component_tag,omitempty"`        // Required item component tag (e.g. "stone")
	EffectType          string  `yaml:"effect_type,omitempty"`          // "damage"|"heal"|"buff"|"shield"|"dot"|"knockdown"|"charm"|"drain_area" (mob-cast only: area life-drain + self-heal, see resolveMobDrainArea)
	EffectMagnitude     int     `yaml:"effect_magnitude,omitempty"`     // Legacy: base damage/heal amount
	DamageMultiplier    float64 `yaml:"damage_multiplier,omitempty"`   // Spell damage multiplier for new pipeline (Stage 34)
	EffectDuration      int    `yaml:"effect_duration,omitempty"`      // DoT tick count (default 0 = use 3)
	BuffIds             []int  `yaml:"buff_ids,omitempty"`             // Buff IDs to apply (for "buff" effect type)
	QuestRequired       string `yaml:"quest_required,omitempty"`       // Quest token required before spell can be discovered
	NoDamageInterrupt   bool   `yaml:"no_damage_interrupt,omitempty"`  // Telegraphed casts: skip damage/position concentration-break (still interrupted by the disruptor system)
	IgnoreMoveCooldown  bool   `yaml:"ignore_move_cooldown,omitempty"` // Scripted boss abilities: bypass the shared special-move cast cooldown (btree controls cadence)

	// Companion summoning fields — replaces JS onMagic for summon spells
	SummonMobId          int    `yaml:"summon_mob_id,omitempty"`
	SummonBasePool       int    `yaml:"summon_base_pool,omitempty"`
	SummonScalingDivisor int    `yaml:"summon_scaling_divisor,omitempty"`
	SummonComponentId    int    `yaml:"summon_component_id,omitempty"`
	SummonRequiresCorpse bool   `yaml:"summon_requires_corpse,omitempty"`
	SummonMinCorpsePool  int    `yaml:"summon_min_corpse_pool,omitempty"`

	// YAML text fields — flavor text sent by the engine (replaces JS messaging)
	CastUserText  string `yaml:"cast_user_text,omitempty"`
	CastRoomText  string `yaml:"cast_room_text,omitempty"`
	WaitUserText  string `yaml:"wait_user_text,omitempty"`
	WaitRoomText  string `yaml:"wait_room_text,omitempty"`
	MagicUserText string `yaml:"magic_user_text,omitempty"`
	MagicRoomText string `yaml:"magic_room_text,omitempty"`
}

const (
	WaitRoundsDefault = 3

	Neutral    SpellType = "neutral"    // Neutral, no expected actor target, use on
	HarmSingle SpellType = "harmsingle" // Harmful, defaults to current aggro - magic missile etc
	HarmMulti  SpellType = "harmmulti"  // Harmful, defaults to all aggro mobs - chain lightning etc
	HelpSingle SpellType = "helpsingle" // Helpful, defaults on self - heal etc
	HelpMulti  SpellType = "helpmulti"  // Helpful, defaults on party - mass heal etc
	HarmArea   SpellType = "harmarea"   // Hits everyone in the room, even if hidden or friendly
	HelpArea   SpellType = "helparea"   // Hits everyone in the room, even if hidden

	// DOG Spell Schools
	SchoolElemental     = "elemental"     // Fire, ice, lightning, earth, wind - offensive elemental magic
	SchoolEnhancement   = "enhancement"   // Buffs, shields, enchantments - augmentation magic
	SchoolMental        = "mental"        // Illusions, charms, telepathy - mind-affecting magic (Psionics skill)
	SchoolVital         = "vital"         // Healing, curing, life/death manipulation - vital force magic
	SchoolManifestation = "manifestation" // Companion summoning, charming, binding - uses Charisma+manifestation
)

var (
	allSpells     = map[string]*SpellData{}
	spellsByAlias = map[string]*SpellData{}
)

func (s SpellType) HelpOrHarmString() string {
	switch s {
	case Neutral:
		return `Neutral`
	case HelpSingle, HelpMulti, HelpArea:
		return `Helpful`
	case HarmSingle, HarmMulti, HarmArea:
		return `Harmful`
	}
	return `Unknown`
}

func (s SpellType) TargetTypeString(short ...bool) string {
	// Return a short version
	if len(short) > 0 && short[0] {
		switch s {
		case Neutral:
			return `Self`
		case HelpSingle, HarmSingle:
			return `Single`
		case HelpMulti, HarmMulti:
			return `Group`
		case HelpArea, HarmArea:
			return `Area`
		}
		return `Unknown`
	}
	// Regular handling
	switch s {
	case Neutral:
		return `Self`
	case HelpSingle:
		return `Single Target`
	case HarmSingle:
		return `Single Target`
	case HelpMulti:
		return `Group Target`
	case HarmMulti:
		return `Group Target`
	case HelpArea, HarmArea:
		return `Area Target`
	}
	return `Unknown`
}

// Finds a match for a spell by name or id
func FindSpell(spellName string) string {
	if sd, ok := allSpells[spellName]; ok {
		return sd.SpellId
	}
	for _, spellInfo := range allSpells {
		if strings.ToLower(spellInfo.Name) == spellName {
			return spellInfo.SpellId
		}
	}
	return ``
}

func GetSpell(spellId string) *SpellData {
	if sd, ok := allSpells[spellId]; ok {
		return sd
	}
	return nil
}

func FindSpellByName(spellName string) *SpellData {

	var closestMatch *SpellData = nil

	spellName = strings.ToLower(spellName)
	for _, spellData := range allSpells {

		testName := strings.ToLower(spellData.Name)

		if testName == spellName {
			return spellData
		}

		if closestMatch == nil && strings.HasPrefix(strings.ToLower(spellData.Name), spellName) {
			closestMatch = spellData
		}

	}
	return closestMatch
}

// buildSpellAliasIndex (re)builds the alias→spell index from allSpells, panicking
// on a duplicate alias or an alias that collides with a spellid.
func buildSpellAliasIndex() {
	spellsByAlias = make(map[string]*SpellData, len(allSpells))
	for _, s := range allSpells {
		for _, a := range s.Aliases {
			a = strings.ToLower(strings.TrimSpace(a))
			if a == "" {
				continue
			}
			if _, clash := allSpells[a]; clash {
				panic(fmt.Sprintf("spell alias %q (on %s) collides with a spellid", a, s.SpellId))
			}
			if other, dup := spellsByAlias[a]; dup {
				panic(fmt.Sprintf("duplicate spell alias %q on %s and %s", a, other.SpellId, s.SpellId))
			}
			spellsByAlias[a] = s
		}
	}
}

// ResolveSpell resolves a token to a spell by canonical id, then alias, then
// full display name (case-insensitive). Returns nil if none match.
func ResolveSpell(token string) *SpellData {
	token = strings.ToLower(strings.TrimSpace(token))
	if sd, ok := allSpells[token]; ok {
		return sd
	}
	if sd, ok := spellsByAlias[token]; ok {
		return sd
	}
	return FindSpellByName(token)
}

// ResolveSpellId returns the canonical spellid for a token (id/alias/name), or "".
func ResolveSpellId(token string) string {
	if sd := ResolveSpell(token); sd != nil {
		return sd.SpellId
	}
	return ""
}

func GetAllSpells() map[string]*SpellData {
	retSpellBook := make(map[string]*SpellData)
	for k, v := range allSpells {
		retSpellBook[k] = v
	}
	return retSpellBook
}

func (s *SpellData) Id() string {
	return s.SpellId
}

// SpellData implements the Filepath method from the Loadable interface.
func (s *SpellData) Filepath() string {
	return util.FilePath(fmt.Sprintf("%s.yaml", s.SpellId))
}

func (s *SpellData) Validate() error {

	if s.Difficulty < 0 {
		s.Difficulty = 0
	} else if s.Difficulty > 100 {
		s.Difficulty = 100
	}

	// Validate YAML text tokens
	for _, text := range []string{
		s.CastUserText, s.CastRoomText,
		s.WaitUserText, s.WaitRoomText,
		s.MagicUserText, s.MagicRoomText,
	} {
		for _, w := range textutil.ValidateTokens(text) {
			mudlog.Warn("Spell.Validate", "spellId", s.SpellId, "warning", w)
		}
	}

	// Validate summon fields
	if s.SummonMobId > 0 && s.SummonBasePool == 0 {
		mudlog.Warn("Spell.Validate", "spellId", s.SpellId, "warning", "summon_mob_id set but summon_base_pool is 0")
	}
	if s.SummonRequiresCorpse && s.SummonMinCorpsePool == 0 {
		mudlog.Warn("Spell.Validate", "spellId", s.SpellId, "warning", "summon_requires_corpse set but summon_min_corpse_pool is 0")
	}

	return nil
}

func (s *SpellData) GetDifficulty() int {
	return s.Difficulty
}

// HasSchool returns true if the spell belongs to the given school.
func (s *SpellData) HasSchool(school string) bool {
	for _, sc := range s.Schools {
		if sc == school {
			return true
		}
	}
	return false
}

// GetSchoolsString returns a comma-separated string of spell schools
func (s *SpellData) GetSchoolsString() string {
	if len(s.Schools) == 0 {
		return "Unknown"
	}
	return strings.Join(s.Schools, ", ")
}

// GetTotalConvictionCost returns the conviction cost, optionally scaled by a multiplier
func (s *SpellData) GetTotalConvictionCost(multiplier float64) int {
	if multiplier <= 0 {
		multiplier = 1.0
	}
	return int(float64(s.Cost) * multiplier)
}

// GetTotalHealthCost returns the health cost, optionally scaled by a multiplier
func (s *SpellData) GetTotalHealthCost(multiplier float64) int {
	if multiplier <= 0 {
		multiplier = 1.0
	}
	return int(float64(s.HealthCost) * multiplier)
}

// MaxFoldsForSkill returns the highest base_folds a player can discover at a given casting skill level.
func MaxFoldsForSkill(skillLevel int) int {
	switch {
	case skillLevel >= 80:
		return 32
	case skillLevel >= 70:
		return 28
	case skillLevel >= 60:
		return 24
	case skillLevel >= 50:
		return 20
	case skillLevel >= 40:
		return 16
	case skillLevel >= 30:
		return 12
	case skillLevel >= 20:
		return 10
	case skillLevel >= 10:
		return 8
	case skillLevel >= 5:
		return 6
	default:
		return 4
	}
}

// GetEligibleSpells returns spell IDs the player could discover (not in spellBook, within fold threshold).
// When schools are provided, only spells belonging to at least one of those schools are returned.
// When schools is empty, all non-manifestation spells are returned (backward compat).
// Spells with QuestRequired set are never returned by discovery.
func GetEligibleSpells(spellBook map[string]int, skillLevel int, schools ...string) []string {
	maxFolds := MaxFoldsForSkill(skillLevel)
	var eligible []string
	for id, sp := range allSpells {
		if _, known := spellBook[id]; known {
			continue
		}
		// Quest-gated spells are never discovered organically.
		if sp.QuestRequired != "" {
			continue
		}
		// School filtering.
		if len(schools) > 0 {
			matched := false
			for _, school := range schools {
				if sp.HasSchool(school) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		} else {
			// Backward compat: no school filter means all non-manifestation spells.
			if sp.HasSchool(SchoolManifestation) {
				continue
			}
		}
		folds := sp.BaseFolds
		if folds == 0 {
			folds = 4
		}
		if folds <= maxFolds {
			eligible = append(eligible, id)
		}
	}
	return eligible
}

func LoadSpellFiles() {

	start := time.Now()

	dataPath := string(configs.GetFilePathsConfig().DataFiles) + `/spells`
	tmpAllSpells, err := fileloader.LoadAllFlatFiles[string, *SpellData](dataPath)
	if err != nil {
		panic(errors.Wrap(err, `filepath: `+dataPath))
	}

	for _, s := range tmpAllSpells {
		if s.Name != "" {
			casing.AssertCanonical(s.Name, "spell", s.SpellId)
		}
	}

	allSpells = tmpAllSpells

	buildSpellAliasIndex()

	mudlog.Info("spells.loadAllSpells()", "loadedCount", len(allSpells), "Time Taken", time.Since(start))

}
