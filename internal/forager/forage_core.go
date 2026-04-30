package forager

// forage_core.go — pure dice-roll + yield-table data for the forage
// command. Lives in the leaf forager package so both usercommands
// (player Forage command) and behaviortree (forager_step NPC action)
// can use the same data without an import cycle.

import (
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// ForageDifficulty maps biome IDs to gaussian roll difficulty targets.
// Lower values are easier to forage in.
var ForageDifficulty = map[string]float64{
	"farmland":  110,
	"forest":    120,
	"land":      125,
	"swamp":     130,
	"shore":     135,
	"water":     135,
	"cave":      135,
	"mountains": 140,
	"cliffs":    145,
}

// ForageYields maps biome IDs to lists of item IDs that can be found.
// Duplicate entries increase the probability of that item appearing.
var ForageYields = map[string][]int{
	"forest":    {40004, 40004, 40005, 40005, 40049, 40049},
	"land":      {40004, 40005, 40049, 40047},
	"farmland":  {40004, 40004, 40005, 40007},
	"swamp":     {40005, 40005, 40004, 40055, 40055, 40056, 40057, 40057},
	"shore":     {40004, 40058},
	"water":     {40058, 40058, 40058, 40058, 40058, 40059},
	"mountains": {40001, 40004, 40005, 40020, 40024, 40025},
	"cliffs":    {40005, 40020, 40024},
	"cave":      {40001, 40001, 40020, 40020, 40005, 40024, 40025, 40026, 40027, 40029},
}

// NightForageYields are appended to the yield table when it's night.
var NightForageYields = map[string][]int{
	"forest":    {40046},
	"mountains": {40046},
	"cave":      {40046},
	"land":      {40046},
}

// ForageAttempt holds the inputs needed to run one forage roll. Used
// by both the player Forage command and NPC forager routines.
type ForageAttempt struct {
	Biome       string
	SearchScore float64 // perception + skill multiplier
	AtNight     bool
}

// ForageResult is the outcome of a single attempt. Caller is responsible
// for actually creating and storing the item; ForageCore is pure.
type ForageResult struct {
	Found  bool
	ItemId int
}

// ForageCore runs the dice roll for one forage attempt. Pure: no
// side effects, no character mutation, no event publication. Caller
// handles cooldowns, item creation, inventory storage, and any
// quest-engine notifications.
//
// Returns Found=false (and ItemId=0) if the biome is unknown or the
// roll missed difficulty.
func ForageCore(a ForageAttempt) ForageResult {
	yields, ok := ForageYields[a.Biome]
	if !ok || len(yields) == 0 {
		return ForageResult{}
	}
	if a.AtNight {
		if night, hasNight := NightForageYields[a.Biome]; hasNight {
			yields = append(append([]int{}, yields...), night...)
		}
	}
	difficulty := ForageDifficulty[a.Biome]
	if difficulty == 0 {
		difficulty = 130
	}
	roll := dice.RollStat(a.SearchScore)
	if roll.Value < difficulty {
		return ForageResult{}
	}
	return ForageResult{Found: true, ItemId: yields[util.Rand(len(yields))]}
}
