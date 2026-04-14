package mobcommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Craft lets a mob start a crafting operation via `craft <recipe-name>`.
// Useful for craftsperson NPC archetypes (blacksmiths, alchemists, etc.)
// that should visibly work on items during roleplay or scripted scenes.
//
// Completion is handled each round by the MobRoundTick crafting block.
func Craft(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return true, nil
	}

	actor := &actions.MobActor{Mob: mob, Room: room}
	result := actions.InitiateCraft(actor, rest)

	switch {
	case result.AlreadyCrafting:
		// Mob is already working; ignore silently (scripts may call too eagerly)
		return true, nil

	case result.RecipeNotFound, result.RecipeNotKnown:
		// Unknown or unavailable recipe — no-op for mobs
		return true, nil

	case result.SkillTooLow:
		// Mob's skill is insufficient; no-op
		return true, nil

	case result.WrongStation:
		// Not at the right station; no-op
		return true, nil

	case result.MissingIngredients:
		// Mob lacks materials; no-op
		return true, nil

	case result.ImmediateComplete:
		room.SendTextVisual(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi> works quickly and produces something.`,
			mob.Character.Name))
		craftBonus := 1.0 + float64(result.SkillMinimum)*float64(configs.GetBalanceConfig().CraftDifficultyProgressionScale)
		mob.Character.OnSkillUseScaled(result.SkillName, 0, craftBonus)
		return true, nil

	case result.Initiated:
		room.SendTextVisual(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi> begins working on something.`,
			mob.Character.Name))
		return true, nil
	}

	return true, nil
}
