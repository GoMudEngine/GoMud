package mobcommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Cast(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) < 1 {
		return true, nil
	}

	spellName := args[0]
	args = args[1:]

	if len(args) > 1 {
		if args[0] == `on` {
			args = args[1:]
		}
	}

	spellArg := strings.Join(args, ` `)

	// Delegate shared setup to the unified action.
	actor := &actions.MobActor{Mob: mob, Room: room}
	result := actions.InitiateCast(actor, spellName, spellArg)

	if !result.Initiated {
		// Any early exit (invalid spell, no target, cooldown) — silently bail.
		return true, nil
	}

	spellInfo := result.SpellInfo

	// Build aggro info for the onCast script.
	spellAggro := characters.SpellAggroInfo{
		SpellId:              spellInfo.SpellId,
		SpellRest:            result.SpellRest,
		TargetUserIds:        result.TargetUserIds,
		TargetMobInstanceIds: result.TargetMobInstanceIds,
	}

	if allowContinueCasting, err := scripting.TrySpellScriptEvent(`onCast`, 0, mob.InstanceId, spellAggro); err != nil || !allowContinueCasting {
		return true, nil
	}

	// First-round conviction slice — mob pays a portion up-front.
	firstRoundCost := spellInfo.Cost / result.FoldsNeeded
	if firstRoundCost < 1 {
		firstRoundCost = 1
	}
	mob.Character.Conviction -= firstRoundCost

	// Commit CastingState, recording the first-round payment.
	result.CastingState.ConvictionSpent = firstRoundCost
	mob.Character.CastingState = result.CastingState

	sendRoomText(room, fmt.Sprintf(
		`<ansi fg="mobname">%s</ansi> begins weaving a spell.`, mob.Character.Name))

	// Initiate combat aggro immediately when targeting a player with an offensive spell.
	// This ensures the mob enters the combat loop so the player is flagged as in combat.
	// The casting block in the combat tick safely handles CastingState and skips melee.
	switch spellInfo.Type {
	case spells.HarmSingle, spells.HarmMulti, spells.HarmArea:
		if mob.Character.Aggro == nil && len(result.TargetUserIds) > 0 {
			mob.Character.SetAggro(result.TargetUserIds[0], 0, characters.DefaultAttack)
		}
	}

	return true, nil
}
