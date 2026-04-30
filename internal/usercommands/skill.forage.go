package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// forageDifficulty maps biome IDs to gaussian roll difficulty targets.
// Lower values are easier to forage in.
var forageDifficulty = map[string]float64{
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

// forageYields maps biome IDs to lists of item IDs that can be found.
// Duplicate entries increase the probability of that item appearing.
var forageYields = map[string][]int{
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

// nightForageYields are appended to the yield table when it's night.
var nightForageYields = map[string][]int{
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
	yields, ok := forageYields[a.Biome]
	if !ok || len(yields) == 0 {
		return ForageResult{}
	}
	if a.AtNight {
		if night, hasNight := nightForageYields[a.Biome]; hasNight {
			yields = append(append([]int{}, yields...), night...)
		}
	}
	difficulty := forageDifficulty[a.Biome]
	if difficulty == 0 {
		difficulty = 130
	}
	roll := dice.RollStat(a.SearchScore)
	if roll.Value < difficulty {
		return ForageResult{}
	}
	return ForageResult{Found: true, ItemId: yields[util.Rand(len(yields))]}
}

func Forage(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	biome := room.GetBiome()
	if _, ok := forageYields[biome.BiomeId]; !ok {
		user.SendText(`There is nothing here worth foraging. Try an outdoor area.`)
		return true, nil
	}

	if !user.Character.TryCooldown(`forage`, "6 rounds") {
		user.SendText(
			fmt.Sprintf("You need to wait %d more rounds before you can forage again.", user.Character.GetCooldown(`forage`)),
		)
		return true, fmt.Errorf("you're doing that too often")
	}

	searchRank := user.Character.GetSkillLevel(skills.Search)
	searchScore := float64(user.Character.Stats.Perception.ValueAdj) + combat.SkillMultiplier(searchRank)*25.0

	bridge := questengine.NewGameBridge(user, room.RoomId)
	questengine.GetEngine().Notify("command", questengine.EventDetails{
		UserId:  user.UserId,
		RoomId:  room.RoomId,
		Command: "forage",
	}, bridge, bridge)

	user.SendText(`You crouch low and begin searching the ground carefully...`)
	room.SendTextVisual(
		fmt.Sprintf(`<ansi fg="username">%s</ansi> is searching the ground for something.`, user.Character.Name),
		user.UserId,
	)

	result := ForageCore(ForageAttempt{
		Biome:       biome.BiomeId,
		SearchScore: searchScore,
		AtNight:     gametime.IsNight(),
	})

	if !result.Found {
		user.SendText(`You find nothing of use this time.`)
		return true, nil
	}

	newItem := items.New(result.ItemId)
	if !newItem.IsValid() {
		user.SendText(`You find something, but it crumbles in your hands.`)
		return true, nil
	}

	user.Character.StoreItem(newItem)
	events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: newItem, Gained: true})
	user.Character.CheckSkillProgression(string(skills.Search), user.UserId, 1.0)

	user.SendText(fmt.Sprintf(`You find a <ansi fg="itemname">%s</ansi>.`, newItem.DisplayName()))

	return true, nil
}
