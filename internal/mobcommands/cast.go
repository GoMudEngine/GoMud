package mobcommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/textutil"
	"github.com/GoMudEngine/GoMud/internal/users"
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
		// Surface the early-exit reason at debug level so tactics-cast
		// races (higher-priority cast losing CastingState slot to a
		// lower-priority one queued in the same tick) can be diagnosed.
		// See project_tactics_cast_preemption.md.
		switch {
		case result.AlreadyCasting:
			inProgress := ""
			if mob.Character.CastingState != nil {
				inProgress = mob.Character.CastingState.SpellId
			}
			mudlog.Debug("mob.Cast",
				"mob", mob.Character.Name,
				"requested_spell", spellName,
				"reason", "already casting",
				"in_progress", inProgress)
		case result.OnCooldown:
			mudlog.Debug("mob.Cast",
				"mob", mob.Character.Name,
				"requested_spell", spellName,
				"reason", "on cooldown")
		case result.InvalidSpell:
			mudlog.Debug("mob.Cast",
				"mob", mob.Character.Name,
				"requested_spell", spellName,
				"reason", "invalid spell")
		case result.NoTarget:
			mudlog.Debug("mob.Cast",
				"mob", mob.Character.Name,
				"requested_spell", spellName,
				"reason", "no target")
		}
		return true, nil
	}

	spellInfo := result.SpellInfo

	// Send YAML cast text (if defined).
	if spellInfo.CastUserText != "" || spellInfo.CastRoomText != "" {
		castRoom := rooms.LoadRoom(mob.Character.RoomId)
		tCtx := textutil.TokenContext{
			SourceName:      mob.Character.GetCharacterName(true),
			SourcePlainName: mob.Character.GetCharacterName(false),
		}
		if len(result.TargetUserIds) > 0 {
			if tUser := users.GetByUserId(result.TargetUserIds[0]); tUser != nil {
				tCtx.TargetName = tUser.Character.GetCharacterName(true)
				tCtx.TargetPlainName = tUser.Character.GetCharacterName(false)
			}
		} else if len(result.TargetMobInstanceIds) > 0 {
			if tMob := mobs.GetInstance(result.TargetMobInstanceIds[0]); tMob != nil {
				tCtx.TargetName = tMob.Character.GetCharacterName(true)
				tCtx.TargetPlainName = tMob.Character.GetCharacterName(false)
			}
		}
		cfg := textutil.SendTextConfig{
			RoomSendFunc: func(msg string, skip ...int) {
				if castRoom != nil {
					castRoom.SendText(msg, skip...)
				}
			},
		}
		textutil.SendPhaseText("", spellInfo.CastRoomText, tCtx, "pink", cfg)
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
		if !mob.Character.IsInCombat() && len(result.TargetUserIds) > 0 {
			mob.Character.SetAggro(result.TargetUserIds[0], 0, characters.DefaultAttack)
		}
	}

	return true, nil
}
