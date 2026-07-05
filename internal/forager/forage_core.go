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
	"forest":    {40004, 40004, 40005, 40005, 40049, 40049, 40067, 40063, 40066},        // 40067 pine pitch (tapped from forest pines, e.g. Fernway South); +40063 shadowcap, +40066 blood-moss (cooking chunk)
	"land":      {40004, 40005, 40049, 40047, 40121, 40122, 40151},                      // +40121 wild grapes, +40122 windfall fruit (warm-country produce, e.g. Amber Valley); +40151 gleaned grain (dry wheat plateau, e.g. East Road); NOTE: wild plums (40150) are deliberately farmland-only (moister hedgerow, not dry plateau)
	"farmland":  {40004, 40004, 40005, 40007, 40121, 40121, 40122, 40122, 40150, 40151}, // cultivated land yields more orchard/vine produce; +40150 wild plums, +40151 gleaned grain (East Road wheat country)
	"swamp":     {40005, 40005, 40004, 40055, 40055, 40056, 40057, 40057},
	"shore":     {40004, 40058},
	"water":     {40058, 40058, 40058, 40058, 40058, 40059, 40123, 40124}, // +40123 watercress, +40124 freshwater mussels (river country, e.g. River Road)
	"mountains": {40001, 40004, 40005, 40020, 40024, 40025, 40069, 40069}, // +40069 basalt-iron ore (foraged from the scablands basalt, e.g. the Pothole Coulee Forge-spoke talus slope)
	"cliffs":    {40005, 40020, 40024},
	"cave":      {40001, 40001, 40020, 40020, 40005, 40024, 40025, 40026, 40027, 40029, 40011}, // 40011 hive fragment (crystallized hive matter, e.g. Ironwind Steppe caves)
}

// NightForageYields are appended to the yield table when it's night.
var NightForageYields = map[string][]int{
	"forest":    {40046},
	"mountains": {40046},
	"cave":      {40046},
	"land":      {40046},
}

// ZoneForageYields adds zone-exclusive forageables (keyed by zone display
// name). Appended only when the player forages in that exact zone. Used for
// the pinnacle ultra-rare reagents (single entry among many biome commons =
// rarest outcome). Player-forage only — NOT applied to NPC foragers.
var ZoneForageYields = map[string][]int{
	"Stillwater Marsh":         {40198, 40202}, // Still-Glass Rosette, First-Bloom Nectar
	"Ironwind Steppe":          {40199},        // Mockingbird Amber
	"The Fernway South":        {40200},        // Ironwood Thorn-Heart
	"Labyrinth of Low Tunnels": {40201},        // Bloom-Saturated Geode
	"The Confluence":           {40203},        // Chorus-Shard
}

// StormForageYields adds weather-gated forageables (keyed by biome), appended
// only when the zone's current weather is "storm". Player-forage only.
var StormForageYields = map[string][]int{
	"mountains": {40204}, // Stormfront Residue — highland storms
}

// buildForagePool assembles the candidate yield slice for a forage attempt:
// the biome base + night overlay + zone overlay + storm overlay. Duplicate
// entries raise probability; a single appended ultra-rare is the rarest
// possible outcome.
func buildForagePool(biome, zone, weather string, atNight bool) []int {
	base := ForageYields[biome]
	pool := append([]int{}, base...)
	if atNight {
		pool = append(pool, NightForageYields[biome]...)
	}
	if zone != "" {
		pool = append(pool, ZoneForageYields[zone]...)
	}
	if weather == "storm" {
		pool = append(pool, StormForageYields[biome]...)
	}
	return pool
}

// ForageAttempt holds the inputs needed to run one forage roll. Used
// by both the player Forage command and NPC forager routines. Zone and
// Weather are player-forage-only fields — the NPC forager path
// (behaviortree/actions_forager.go) deliberately leaves them empty so
// zone/weather-gated ultra-rare reagents can never be foraged by NPCs
// and leaked into vendor stock.
type ForageAttempt struct {
	Biome       string
	SearchScore float64 // perception + skill multiplier
	AtNight     bool
	Zone        string // player-forage only; see ZoneForageYields
	Weather     string // player-forage only; see StormForageYields
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
	if _, ok := ForageYields[a.Biome]; !ok {
		return ForageResult{}
	}
	pool := buildForagePool(a.Biome, a.Zone, a.Weather, a.AtNight)
	if len(pool) == 0 {
		return ForageResult{}
	}
	difficulty := ForageDifficulty[a.Biome]
	if difficulty == 0 {
		difficulty = 130
	}
	roll := dice.RollStat(a.SearchScore)
	if roll.Value < difficulty {
		return ForageResult{}
	}
	return ForageResult{Found: true, ItemId: pool[util.Rand(len(pool))]}
}
