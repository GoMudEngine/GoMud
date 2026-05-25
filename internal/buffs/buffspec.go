package buffs

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/statmods"
	"github.com/GoMudEngine/GoMud/internal/textutil"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/pkg/errors"
)

// Something temporarily attached to a character
// That modifies some aspect of their status
/*
Examples:
Fast Healing - increased natural health recovery for 10 rounds
Poison - add -10 health every round for 5 rounds
*/

type Flag string

const (
	//
	// All Flags must be lowercase
	//
	All Flag = ``

	// Behavioral flags
	NoCombat       Flag = `no-combat`
	NoMovement     Flag = `no-go`
	NoFlee         Flag = `no-flee`
	NoAggroTarget  Flag = `no-aggro-target` // grace-period protection: mobs cannot acquire aggro on the bearer
	CancelIfCombat Flag = `cancel-on-combat`
	CancelOnAction Flag = `cancel-on-action`
	CancelOnDamage Flag = `cancel-on-damage` // chunk 3.3: cancels on any damage event
	CancelOnWater  Flag = `cancel-on-water`

	// Death preventing
	ReviveOnDeath Flag = `revive-on-death`

	// Gear related
	PermaGear   Flag = `perma-gear`
	RemoveCurse Flag = `remove-curse`

	// Harmful flags
	Poison Flag = `poison`
	Drunk  Flag = `drunk`

	// Useful flags
	Hidden         Flag = `hidden`
	Sleeping       Flag = `sleeping` // chunk 3.3: bearer is asleep
	Accuracy       Flag = `accuracy`
	Blink          Flag = `blink`
	EmitsLight     Flag = `lightsource`
	SuperHearing   Flag = `superhearing`
	NightVision    Flag = `nightvision`
	InfraredVision Flag = `infraredvision`
	Warmed         Flag = `warmed`
	Hydrated       Flag = `hydrated`
	Thirsty        Flag = `thirsty`

	// Phase 25 spell buff flags
	Haste         Flag = `haste`
	DamageBonus   Flag = `damage-bonus`
	Slow          Flag = `slow`
	SkillProgress Flag = `skill-progress`
	MutationRate  Flag = `mutation-rate`

	// Flags that reveal things
	SeeHidden Flag = `see-hidden`
	SeeNouns  Flag = `see-nouns`

	// Arbitrarily chosen round for calculating trigger round counts
	validationRound = 1000000
)

var (
	buffs map[int]*BuffSpec = make(map[int]*BuffSpec)

	validationCalculator = gametime.GetDate(validationRound)
)

type BuffSpec struct {
	BuffId        int               // Unique identifier for this buff spec
	Name          string            // The name of the buff
	Description   string            // A description of the buff
	Secret        bool              // Whether or not the buff is secret (not displayed to the user)
	TriggerNow    bool              `yaml:"triggernow,omitempty"`    // if true, buff triggers once right when it is applied
	TriggerRate   string            `yaml:"triggerrate,omitempty"`   // How often should it trigger? (time string)
	RoundInterval int               `yaml:"roundinterval,omitempty"` // triggers every x rounds
	TriggerCount  int               `yaml:"triggercount,omitempty"`  // How many times it triggers before it is removed
	StatMods      statmods.StatMods `yaml:"statmods,omitempty"`      // stat mods for the duration of the buff
	Flags         []Flag            `yaml:"flags,omitempty"`         // A list of actions and such that this buff prevents or enables

	// YAML text fields — flavor text sent by the engine (replaces JS messaging)
	StartUserText   string `yaml:"start_user_text,omitempty"`
	StartRoomText   string `yaml:"start_room_text,omitempty"`
	TriggerUserText string `yaml:"trigger_user_text,omitempty"`
	TriggerRoomText string `yaml:"trigger_room_text,omitempty"`
	EndUserText     string `yaml:"end_user_text,omitempty"`
	EndRoomText     string `yaml:"end_room_text,omitempty"`

	// Config-driven tick fields — replaces JS onTrigger for heal/DoT buffs
	TickPool         string  `yaml:"tick_pool,omitempty"`          // "health", "stamina", "conviction"
	TickPercent      float64 `yaml:"tick_percent,omitempty"`       // Base % of max pool. Positive=heal, negative=damage
	TickVariance     float64 `yaml:"tick_variance,omitempty"`      // Random variance added to percent
	TickMin          int     `yaml:"tick_min,omitempty"`           // Minimum absolute tick amount (default 1)
	StartRemoveBuffs []int   `yaml:"start_remove_buffs,omitempty"` // Buff IDs to remove when this buff starts
}

// Calculates the value of this buff
func (b *BuffSpec) GetValue() int {
	val := 0

	for _, v := range b.StatMods {
		val += int(math.Abs(float64(v)))
	}

	freqVal := max(5-b.RoundInterval, 0)
	val += freqVal
	val += len(b.Flags) * 5

	if b.TriggerCount > 0 {
		val *= b.TriggerCount
	}

	return val
}

func (b *BuffSpec) VisibleNameDesc() (name, description string) {
	if b.Secret {
		return "Mysterious Affliction", "Unknown"
	}
	return b.Name, b.Description
}

type BuffMessage struct {
	User string
	Room string
}

type BuffMessages struct {
	Start  BuffMessage
	Effect BuffMessage
	End    BuffMessage
}

func GetBuffSpec(buffId int) *BuffSpec {
	if buffId < 0 {
		buffId *= -1
	}

	if buff, ok := buffs[buffId]; ok {
		return buff
	}

	return nil
}

func GetAllBuffIds() []int {

	var results []int = make([]int, 0, len(buffs))
	for _, buff := range buffs {
		results = append(results, buff.BuffId)
	}

	return results
}

// Searches for buffs whose name contains text and returns their Ids
func SearchBuffs(searchTerm string) []int {

	searchTerm = strings.TrimSpace(strings.ToLower(searchTerm))

	var results []int = make([]int, 0, 2)

	for _, buff := range buffs {
		if strings.Contains(strings.ToLower(buff.Name), searchTerm) {
			results = append(results, buff.BuffId)
		} else if strings.Contains(strings.ToLower(buff.Description), searchTerm) {
			results = append(results, buff.BuffId)
		}
	}

	return results
}

// Presumably to ensure the datafile hasn't messed something up.
func (b *BuffSpec) Id() int {
	return b.BuffId
}

// Presumably to ensure the datafile hasn't messed something up.
func (b *BuffSpec) Validate() error {

	// Validate YAML text tokens
	for _, text := range []string{
		b.StartUserText, b.StartRoomText,
		b.TriggerUserText, b.TriggerRoomText,
		b.EndUserText, b.EndRoomText,
	} {
		for _, w := range textutil.ValidateTokens(text) {
			mudlog.Warn("Buff.Validate", "buffId", b.BuffId, "warning", w)
		}
	}

	// Validate tick fields
	if b.TickPool != "" {
		switch b.TickPool {
		case "health", "stamina", "conviction":
			// valid
		default:
			return fmt.Errorf("buffId %d (%s) has invalid tick_pool %q (must be health/stamina/conviction)", b.BuffId, b.Name, b.TickPool)
		}
		if b.TickPercent == 0 {
			mudlog.Warn("Buff.Validate", "buffId", b.BuffId, "warning", "tick_pool set but tick_percent is 0")
		}
	}

	// If this is the quit/meditating buff, override the trigger count
	if b.BuffId == 0 {
		b.TriggerCount = int(configs.GetNetworkConfig().LogoutRounds)
	}

	// If TriggerRate is set, validate RoundInterval and TriggerCount
	if b.TriggerRate != "" {
		b.RoundInterval = int(validationCalculator.AddPeriod(b.TriggerRate) - validationRound)

		if b.TriggerCount < 1 {
			return fmt.Errorf("buffId %d (%s) has a TriggersCount of < 1, must be at least 1", b.BuffId, b.Name)
		}
		if b.RoundInterval < 1 {
			return fmt.Errorf("buffId %d (%s) has a RoundInterval of < 1, must be at least 1. Is %s a valid time string?", b.BuffId, b.Name, b.TriggerRate)
		}
	}

	return nil
}

func (b *BuffSpec) Filename() string {
	filename := util.ConvertForFilename(b.Name)
	return fmt.Sprintf("%d-%s.yaml", b.BuffId, filename)
}

func (b *BuffSpec) Filepath() string {
	return b.Filename()
}

// file self loads due to init()
func LoadDataFiles() {

	start := time.Now()

	dataPath := string(configs.GetFilePathsConfig().DataFiles) + `/buffs`
	tmpBuffs, err := fileloader.LoadAllFlatFiles[int, *BuffSpec](dataPath)
	if err != nil {
		panic(errors.Wrap(err, `filepath: `+dataPath))
	}

	buffs = tmpBuffs

	mudlog.Info("buffSpec.LoadDataFiles()", "loadedCount", len(buffs), "Time Taken", time.Since(start))
}
