package species

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/stats"
	"github.com/GoMudEngine/GoMud/internal/util"
	pkgerrors "github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Size string

var (
	allSpecies map[int]*Species = map[int]*Species{}
)

const (
	Small  Size = "small"  // Something like a mouse, dog
	Medium Size = "medium" // Something like a human
	Large  Size = "large"  // Something like a troll, ogre, dragon, kraken, or leviathan (or bigger).
)

type Species struct {
	SpeciesId        int              `yaml:"speciesid"`
	Name             string
	Description      string
	BuffIds          []int // Permabuffs this species always has
	Size             Size
	UnarmedName      string
	// NaturalAttack is the combat-message subtype an unarmed member of this
	// species uses for BASIC attacks (e.g. items.Bite, items.Claws). Empty =>
	// humanoid default (Unarmed -> generic). Must be a known items.ItemSubType
	// with a loaded combat-message file (validated at load in a later task).
	NaturalAttack    items.ItemSubType `yaml:"natural_attack,omitempty"`
	Tameable         bool
	Damage           items.Damage
	DamageMultiplier float64          `yaml:"damage_multiplier,omitempty"` // Natural weapon power (0=use config default)
	Selectable       bool
	AngryCommands    []string         // randomly chosen to queue when they are angry/entering combat.
	KnowsFirstAid    bool             // Whether they can apply aid to other players.
	Stats            stats.Statistics // Base stats for this species.
	NaturalArmor     int              `yaml:"naturalarmor,omitempty"` // Innate damage reduction (chitin, thick hide, etc.)
	DisabledSlots    []string         `yaml:"disabledslots,omitempty"`
	NaturalBash      bool             `yaml:"naturalbash,omitempty"`    // Can bash without a shield (elementals, golems)
	LifeDrain        bool             `yaml:"lifedrain,omitempty"`      // Drains life on its `drain` special (vampires, parasites)
	ReturnDamage     int              `yaml:"return_damage,omitempty"`  // % of melee damage returned to attacker (fire elemental, etc.)
	GrappleImmune    bool             `yaml:"grapple_immune,omitempty"` // Cannot be grappled (ethereal, fire, etc.)
	BodyParts        []string         `yaml:"body_parts,omitempty"`     // Set of canonical body-part tags this species has
	IntrinsicMutations map[string]int `yaml:"intrinsic_mutations,omitempty"` // Maps mutation id -> baseline rank
	// MutationImmune marks a species that never gains mutations — neither
	// intrinsic kits nor in-combat acquisition. Machines/constructs (the
	// Chrysalis is a biological phenomenon; it does not touch the mechanical).
	MutationImmune bool `yaml:"mutation_immune,omitempty"`
}

func GetAllSpecies() []Species {
	ret := []Species{}
	for _, s := range allSpecies {
		ret = append(ret, *s)
	}
	return ret
}

func GetSpecies(speciesId int) *Species {
	return allSpecies[speciesId]
}

func FindSpecies(name string) (Species, bool) {

	name = strings.ToLower(name)

	closeMatch := -1
	for idx, s := range allSpecies {
		testName := strings.ToLower(s.Name)
		if strings.HasPrefix(testName, name) {
			return *s, true
		} else if strings.Contains(testName, name) {
			closeMatch = idx
		}
	}
	// close matches
	if closeMatch > -1 {
		return *allSpecies[closeMatch], true
	}

	return Species{}, false
}

func (s *Species) Id() int {
	return s.SpeciesId
}

func (s *Species) Validate() error {
	if s.Name == "" {
		return errors.New("species has no name")
	}
	if s.Description == "" {
		return errors.New("species has no description")
	}
	if s.Size == "" {
		return errors.New("species has no size")
	}
	s.Size = Size(strings.ToLower(string(s.Size))) // Sometimes a mismatching CaSe value is provided.

	// Recalculate stats, based on level one because this is actually the baseline for the species
	s.Stats.Strength.Recalculate()
	s.Stats.Dexterity.Recalculate()
	s.Stats.Perception.Recalculate()
	s.Stats.Vitality.Recalculate()
	s.Stats.Willpower.Recalculate()
	s.Stats.Charisma.Recalculate()

	if s.Damage.Attacks < 1 && s.Damage.DiceCount > 0 && s.Damage.SideCount > 0 {
		s.Damage.Attacks = 1
	}

	// If a diceroll was specified, absorb that into the damage struct
	if s.Damage.DiceRoll != `` {
		s.Damage.InitDiceRoll(s.Damage.DiceRoll)
		s.Damage.FormatDiceRoll()
	}

	return nil
}

func (s Species) GetEnabledSlots() []string {

	ret := []string{}
	slots := []string{
		string(items.Weapon),
		string(items.Offhand),
		string(items.Head),
		string(items.Neck),
		string(items.Body),
		string(items.Belt),
		string(items.Gloves),
		string(items.Ring),
		string(items.Legs),
		string(items.Feet),
	}

	for _, slotName := range slots {
		add := true
		for _, slot := range s.DisabledSlots {
			if slotName == slot {
				add = false
				break
			}
		}
		if add {
			ret = append(ret, slotName)
		}
	}

	return ret
}

func (s *Species) Filename() string {
	filename := util.ConvertForFilename(s.Name)
	return fmt.Sprintf("%d-%s.yaml", s.SpeciesId, filename)
}

func (s *Species) Filepath() string {
	return s.Filename()
}

func (s *Species) Save() error {

	bytes, err := yaml.Marshal(s)
	if err != nil {
		return err
	}

	saveFilePath := util.FilePath(configs.GetFilePathsConfig().DataFiles.String(), `/`, `species`, `/`, s.Filename())

	err = os.WriteFile(saveFilePath, bytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

// validNaturalAttacks is the set of subtypes a species may declare for its
// basic unarmed attacks. Each must have a loaded combat-message file of the
// same name in _datafiles/world/dogmud/combat-messages/.
var validNaturalAttacks = map[items.ItemSubType]struct{}{
	items.Bite:  {},
	items.Claws: {},
	items.Slam:  {},
	items.Gore:  {},
	items.Sting: {},
}

// validateGoreHasHorns returns an error if a gore (horned) species does not
// declare the "horns" body part — gore is anatomically a horn strike.
func validateGoreHasHorns(s *Species) error {
	if s.NaturalAttack != items.Gore {
		return nil
	}
	for _, part := range s.BodyParts {
		if part == "horns" {
			return nil
		}
	}
	return fmt.Errorf("species %d (%s): natural_attack gore requires \"horns\" in body_parts", s.SpeciesId, s.Name)
}

func validateNaturalAttack(s *Species) error {
	if s.NaturalAttack == "" || s.NaturalAttack == items.Unarmed {
		return nil
	}
	if _, ok := validNaturalAttacks[s.NaturalAttack]; !ok {
		return fmt.Errorf("species %d (%s): unknown natural_attack %q", s.SpeciesId, s.Name, s.NaturalAttack)
	}
	return nil
}

// file self loads due to init()
func LoadDataFiles() {

	start := time.Now()

	dataPath := configs.GetFilePathsConfig().DataFiles.String() + `/species`
	tmpSpecies, err := fileloader.LoadAllFlatFiles[int, *Species](dataPath)
	if err != nil {
		panic(pkgerrors.Wrap(err, `filepath: `+dataPath))
	}

	allSpecies = tmpSpecies

	for _, s := range allSpecies {
		if err := validateNaturalAttack(s); err != nil {
			panic(err)
		}
		if err := validateGoreHasHorns(s); err != nil {
			panic(err)
		}
	}

	mudlog.Info("species.LoadDataFiles()", "loadedCount", len(allSpecies), "Time Taken", time.Since(start))

}

// CanonicalBodyParts is the exhaustive set of body-part tags that
// can appear in Species.BodyParts and MutationSpec.RequiresBodyParts.
// Boot-time validation rejects any value not in this set.
var CanonicalBodyParts = []string{
	"arms",  // explicit grasping limbs distinct from legs
	"hands", // fingered manipulators on the arms
	"legs",  // distinct locomotion limbs
	"eyes",  // visual organs
	"mouth", // biting/vocal apparatus
	"skin",  // surface coverage
	"tail",  // anatomical tail
	"horns", // bony protrusions used for gore attacks
}

// IsCanonicalBodyPart reports whether the given tag is in the
// canonical set.
func IsCanonicalBodyPart(tag string) bool {
	for _, t := range CanonicalBodyParts {
		if t == tag {
			return true
		}
	}
	return false
}

// HasBodyPart returns true if this species declares the given
// canonical body-part tag, OR if BodyParts is nil (fail-open for
// un-migrated species). An explicit empty slice means "no body
// parts" and returns false for every tag.
func (s *Species) HasBodyPart(part string) bool {
	if s == nil {
		return true // defensive fail-open
	}
	if s.BodyParts == nil {
		return true // un-migrated species fail-open
	}
	for _, p := range s.BodyParts {
		if p == part {
			return true
		}
	}
	return false
}

// HasAllBodyParts returns true if this species has every part in
// the requirements list. Empty requirements list always returns
// true (body-agnostic mutations).
func (s *Species) HasAllBodyParts(required []string) bool {
	for _, part := range required {
		if !s.HasBodyPart(part) {
			return false
		}
	}
	return true
}

// ValidateBodyPartTags scans all loaded species and panics on any
// unknown body-part tag or unknown intrinsic mutation id. Called
// from main after species + mutations are loaded.
//
// mutationIdExists is a callback for cross-package lookup — pass
// mutations.HasSpec or equivalent.
func ValidateBodyPartTags(mutationIdExists func(id string) bool) {
	for _, sp := range allSpecies {
		for _, tag := range sp.BodyParts {
			if !IsCanonicalBodyPart(tag) {
				panic(fmt.Sprintf(
					"species %q (id %d): unknown body_part tag %q (canonical: %v)",
					sp.Name, sp.SpeciesId, tag, CanonicalBodyParts))
			}
		}
		for id := range sp.IntrinsicMutations {
			if !mutationIdExists(id) {
				panic(fmt.Sprintf(
					"species %q (id %d): unknown mutation id in intrinsic_mutations: %q",
					sp.Name, sp.SpeciesId, id))
			}
		}
	}
}
