package usercommands

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

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

	forageSkill := int(math.Round(float64(user.Character.GetSkillLevel(skills.Foraging)) * float64(configs.GetBalanceConfig().SkillWeight)))
	perceptionAdj := user.Character.Stats.Perception.ValueAdj
	successOdds := 20 + (forageSkill * 5) + int(math.Ceil(float64(perceptionAdj)/10))
	if successOdds > 90 {
		successOdds = 90
	}

	user.SendText(`You crouch low and begin searching the ground carefully...`)
	room.SendText(
		fmt.Sprintf(`<ansi fg="username">%s</ansi> is searching the ground for something.`, user.Character.Name),
		user.UserId,
	)

	roll := util.Rand(100)
	util.LogRoll(`Forage`, roll, successOdds)

	if roll >= successOdds {
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
	user.Character.OnSkillUse(`foraging`, user.UserId)

	user.SendText(fmt.Sprintf(`You find a <ansi fg="itemname">%s</ansi>.`, newItem.DisplayName()))

	return true, nil
}
