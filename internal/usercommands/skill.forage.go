package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/forager"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Forage(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	biome := room.GetBiome()
	if _, ok := forager.ForageYields[biome.BiomeId]; !ok {
		user.SendText(messaging.CategorySystem, `There is nothing here worth foraging. Try an outdoor area.`)
		return true, nil
	}

	if !user.Character.TryCooldown(`forage`, "6 rounds") {
		user.SendText(messaging.CategorySystem, 
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

	user.SendText(messaging.CategorySystem, `You crouch low and begin searching the ground carefully...`)
	room.SendTextVisual(messaging.CategoryMobEmote, 
		fmt.Sprintf(`<ansi fg="username">%s</ansi> is searching the ground for something.`, user.Character.Name),
		user.UserId,
	)

	result := forager.ForageCore(forager.ForageAttempt{
		Biome:       biome.BiomeId,
		SearchScore: searchScore,
		AtNight:     gametime.IsNight(),
	})

	if !result.Found {
		user.SendText(messaging.CategorySystem, `You find nothing of use this time.`)
		return true, nil
	}

	newItem := items.New(result.ItemId)
	if !newItem.IsValid() {
		user.SendText(messaging.CategorySystem, `You find something, but it crumbles in your hands.`)
		return true, nil
	}

	user.Character.StoreItem(newItem)
	events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: newItem, Gained: true})
	user.Character.CheckSkillProgression(string(skills.Search), user.UserId, 1.0)

	user.SendText(messaging.CategorySystem, fmt.Sprintf(`You find a <ansi fg="itemname">%s</ansi>.`, newItem.DisplayName()))

	return true, nil
}
