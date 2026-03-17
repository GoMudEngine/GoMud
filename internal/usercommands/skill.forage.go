package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
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
	"cave":      135,
	"mountains": 140,
	"cliffs":    145,
}

// forageYields maps biome IDs to lists of item IDs that can be found.
// Duplicate entries increase the probability of that item appearing.
var forageYields = map[string][]int{
	"forest":    {40004, 40004, 40005, 40005},
	"land":      {40004, 40005},
	"farmland":  {40004, 40004, 40005, 40007},
	"swamp":     {40005, 40005, 40004},
	"shore":     {40004},
	"mountains": {40004, 40005, 40020, 40024, 40025},
	"cliffs":    {40005, 40020, 40024},
	"cave":      {40020, 40020, 40005, 40024, 40025, 40026, 40027, 40029},
}

func Forage(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	biome := room.GetBiome()
	yields, ok := forageYields[biome.BiomeId]
	if !ok || len(yields) == 0 {
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

	difficulty := forageDifficulty[biome.BiomeId]
	if difficulty == 0 {
		difficulty = 130 // fallback for unknown biomes
	}

	user.SendText(`You crouch low and begin searching the ground carefully...`)
	room.SendText(
		fmt.Sprintf(`<ansi fg="username">%s</ansi> is searching the ground for something.`, user.Character.Name),
		user.UserId,
	)

	roll := dice.RollStat(searchScore)
	if roll.Value < difficulty {
		user.SendText(`You find nothing of use this time.`)
		return true, nil
	}

	itemId := yields[util.Rand(len(yields))]
	newItem := items.New(itemId)
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
