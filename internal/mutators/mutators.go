package mutators

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

var (
	allMutators = map[string]*MutatorSpec{}
)

type TextBehavior string

const (
	TextPrepend TextBehavior = `prepend`
	TextAppend  TextBehavior = `append` // Default behavior is replace
	TextReplace TextBehavior = `replace`
	TextDefault TextBehavior = TextReplace
)

func (tb TextBehavior) IsValid() bool {
	if tb == TextPrepend || tb == TextAppend || tb == TextReplace {
		return true
	}
	return false
}

type MutatorList []Mutator

type Mutator struct {
	MutatorId      string `yaml:"mutatorid,omitempty"`      // Short text that will uniquely identify this modifier ("dusty")
	SpawnedRound   uint64 `yaml:"spawnedround,omitempty"`   // Tracks when this mutator was created (useful for decay)
	DespawnedRound uint64 `yaml:"despawnedround,omitempty"` // Track when it decayed to nothing.
}

type TextModifier struct {
	Behavior     TextBehavior `yaml:"behavior,omitempty"`     // prepend, append or replace?
	Text         string       `yaml:"text,omitempty"`         // The text that will be injected somehow
	ColorPattern string       `yaml:"colorpattern,omitempty"` // An optional color pattern name to apply
}

type PvpOverride struct {
	Enabled  bool `yaml:"enabled,omitempty"`  // force enabled?
	Disabled bool `yaml:"disabled,omitempty"` // force disabled?
}

type MutatorSpec struct {
	MutatorId string `yaml:"mutatorid,omitempty"` // Short text that will uniquely identify this modifier ("dusty")
	// Text based changes
	NameModifier        *TextModifier `yaml:"namemodifier,omitempty"`
	DescriptionModifier *TextModifier `yaml:"descriptionmodifier,omitempty"`
	AlertModifier       *TextModifier `yaml:"alertmodifier,omitempty"` // These can only append.
	// End text based changes
	DecayIntoId     string                   `yaml:"decayintoid,omitempty"`     // Id of another Mutator that replaces this one when it decays. This can be a circular behavior.
	PlayerBuffIds   []int                    `yaml:"playerbuffids,omitempty"`   // buffId's that apply conditionally TO PLAYERS AND PLAYER FOLLOWERS
	MobBuffIds      []int                    `yaml:"mobbuffids,omitempty"`      // buffId's that apply conditionally TO MOBS
	NativeBuffIds   []int                    `yaml:"nativebuffids,omitempty"`   // buffId's that apply conditionally TO MOBS THAT SPAWNED IN THIS ROOM
	DecayRate       string                   `yaml:"decayrate,omitempty"`       // how long until it is gone
	RespawnRate     string                   `yaml:"respawnrate,omitempty"`     // daily, weekly, 1 day, 3 day, monthly, etc.
	LightMod        int                      `yaml:"lightmod,omitempty"`        //  -2 to 2 (change). If result is 0 = none. 1 = can see this room. 2 = can see this room and all exits
	RegenMultiplier float64                  `yaml:"regenmultiplier,omitempty"` // multiplies HP/SP/CP regen for any actor in the room (1.0 / 0 = no bonus)
	Exits           map[string]exit.RoomExit `yaml:"exits,omitempty"`           // name/roomId pairs of exits only available while mutator is live.
	Pvp             PvpOverride              `yaml:"pvp,omitempty"`             // optionally force room pvp attributes.
	OutdoorOnly     bool                     `yaml:"outdooronly,omitempty"`     // skip this mutator's effects in indoor-biome rooms (weather etc.)
}

func GetAllMutatorSpecs() []MutatorSpec {
	mutSpec := []MutatorSpec{}
	for _, spec := range allMutators {
		mutSpec = append(mutSpec, *spec)
	}
	return mutSpec
}

func GetMutatorSpec(mutatorId string) *MutatorSpec {
	mutSpec := allMutators[mutatorId]
	return mutSpec
}

func GetAllMutatorIds() []string {
	allNames := []string{}
	for _, m := range allMutators {
		allNames = append(allNames, m.MutatorId)
	}
	return allNames
}

func IsMutator(mutName string) bool {
	_, ok := allMutators[mutName]
	return ok
}

func (ml *MutatorList) Has(mutName string) bool {
	if _, ok := allMutators[mutName]; !ok {
		return false
	}

	for _, mut := range *ml {
		if mut.MutatorId == mutName {
			return true
		}
	}
	return false
}

func (ml *MutatorList) Add(mutName string) bool {

	if _, ok := allMutators[mutName]; !ok {
		return false
	}

	for i, mut := range *ml {
		if mut.MutatorId == mutName {
			if !mut.Live() {
				mut.DespawnedRound = 0
				mut.SpawnedRound = 0
				mut.Update(util.GetRoundCount())
				(*ml)[i] = mut
				return true
			}
		}
	}
	*ml = append(*ml, Mutator{MutatorId: mutName})
	return true
}

func (ml *MutatorList) Remove(mutName string) bool {
	for i, mut := range *ml {
		if mut.MutatorId == mutName {
			if mut.Live() {
				rNow := util.GetRoundCount()
				mut.DespawnedRound = rNow
				mut.SpawnedRound = 0
				(*ml)[i] = mut
				(*ml).Update(rNow)
				return true
			}
		}
	}

	return false
}

func (ml *MutatorList) Update(roundNow uint64) {

	if ml == nil {
		ml = &MutatorList{}
	}

	removeIdx := []int{}
	for idx := range *ml {
		(*ml)[idx].Update(roundNow)
		if (*ml)[idx].Removable() {
			removeIdx = append(removeIdx, idx)
		}
	}

	for i := len(removeIdx) - 1; i >= 0; i-- {
		(*ml) = append((*ml)[:removeIdx[i]], (*ml)[removeIdx[i]+1:]...)
	}
}

// Returns a new list containing only active mutators
func (ml *MutatorList) GetActive() MutatorList {
	activeMuts := MutatorList{}
	for _, mut := range *ml {
		if !mut.Live() {
			continue
		}
		activeMuts = append(activeMuts, mut)
	}
	return activeMuts
}

func (m *Mutator) Live() bool {
	return m.DespawnedRound == 0
}

// Returns true if mutator can be removed since it won't become anything or respawn
func (m *Mutator) Removable() bool {
	// If currently in play, don't remove
	if m.Live() {
		return false
	}
	// If it might respawn, don't remove
	if m.GetSpec().RespawnRate != `` {
		return false
	}

	return true
}

func (m *Mutator) GetSpec() *MutatorSpec {
	return allMutators[m.MutatorId]
}

// Checks whether it decays or respawns
// Returns true if it has changed somehow?
func (m *Mutator) Update(currentRound uint64) {
	spec := m.GetSpec()

	//
	// If it hasn't been initialized yet
	//
	if m.SpawnedRound == 0 && m.DespawnedRound == 0 {

		// If it's a special period, don't allow it to auto-initialize.
		// Treat it as expired and now waiting for the initialization
		if strings.HasSuffix(spec.RespawnRate, `noon`) || strings.HasSuffix(spec.RespawnRate, `noons`) ||
			strings.HasSuffix(spec.RespawnRate, `midnight`) || strings.HasSuffix(spec.RespawnRate, `midnights`) ||
			strings.HasSuffix(spec.RespawnRate, `sunrise`) || strings.HasSuffix(spec.RespawnRate, `sunrises`) ||
			strings.HasSuffix(spec.RespawnRate, `sunset`) || strings.HasSuffix(spec.RespawnRate, `sunsets`) {
			m.DespawnedRound = currentRound
		} else {
			m.SpawnedRound = currentRound
		}
	}

	//
	// If it is currently despawned, check whether we should respawn it.
	//
	if spec.RespawnRate != `` {
		if m.DespawnedRound != 0 {

			gd := gametime.GetDate(m.DespawnedRound)
			respawnRound := gd.AddPeriod(spec.RespawnRate)

			// Has enough time passed to do the respawn?
			if currentRound >= respawnRound {
				m.DespawnedRound = 0
				m.SpawnedRound = respawnRound

			}

			return
		}
	}

	//
	// It isn't despawned, so check whether we should despawn it.
	//

	if spec.DecayRate != `` {
		gd := gametime.GetDate(m.SpawnedRound)
		despawnRound := gd.AddPeriod(spec.DecayRate)

		// Has enough time passed to do the despawn?
		if currentRound >= despawnRound {
			if spec.DecayIntoId != `` {

				m.MutatorId = spec.DecayIntoId
				m.SpawnedRound = currentRound
				m.DespawnedRound = 0

			} else {
				m.DespawnedRound = currentRound
			}

			return
		}
	}

}

func (m *MutatorSpec) Filename() string {
	filename := util.ConvertForFilename(m.MutatorId)
	return fmt.Sprintf("%s.yaml", filename)
}

func (m *MutatorSpec) Filepath() string {
	return m.Filename()
}

func (m *MutatorSpec) Save() error {
	fileName := strings.ToLower(m.MutatorId)

	bytes, err := yaml.Marshal(m)
	if err != nil {
		return err
	}

	saveFilePath := util.FilePath(configs.GetFilePathsConfig().DataFiles.String(), `/`, `mutators`, `/`, fmt.Sprintf("%s.yaml", fileName))

	err = os.WriteFile(saveFilePath, bytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (m *MutatorSpec) Id() string {
	return m.MutatorId
}

func (m *MutatorSpec) Validate() error {

	if m.NameModifier != nil && !m.NameModifier.Behavior.IsValid() {
		m.NameModifier.Behavior = TextDefault
	}

	if m.DescriptionModifier != nil && !m.DescriptionModifier.Behavior.IsValid() {
		m.DescriptionModifier.Behavior = TextDefault
	}

	if m.AlertModifier != nil && !m.AlertModifier.Behavior.IsValid() {
		m.AlertModifier.Behavior = TextDefault
	}

	return nil
}

// file self loads due to init()
func LoadDataFiles() {

	start := time.Now()

	dataPath := configs.GetFilePathsConfig().DataFiles.String() + `/mutators`
	tmpMutators, err := fileloader.LoadAllFlatFiles[string, *MutatorSpec](dataPath)
	if err != nil {
		panic(errors.Wrap(err, `filepath: `+dataPath))
	}

	allMutators = tmpMutators

	mudlog.Info("mutators.LoadDataFiles()", "loadedCount", len(allMutators), "Time Taken", time.Since(start))
}
