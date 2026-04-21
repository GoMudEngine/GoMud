package behaviortree

import (
	"time"
)

// ActionFunc is the signature for all registered action implementations.
type ActionFunc func(params map[string]any, ctx *EvalContext) Result

// actionRegistry maps action names to their implementations.
var actionRegistry = map[string]ActionFunc{}

func init() {
	actionRegistry["respond"] = actRespond
	actionRegistry["say"] = actSay
	actionRegistry["emote"] = actEmote
	actionRegistry["grant_quest"] = actGrantQuest
	actionRegistry["set_quest_flag"] = actSetQuestFlag
	actionRegistry["give_item"] = actGiveItem
	actionRegistry["return_item"] = actReturnItem
	actionRegistry["take_item"] = actTakeItem
	actionRegistry["give_gold"] = actGiveGold
	actionRegistry["take_gold"] = actTakeGold
	actionRegistry["move"] = actMove
	actionRegistry["attack"] = actAttack
	actionRegistry["flee"] = actFlee
	actionRegistry["cast"] = actCast
	actionRegistry["cast_best_in_category"] = actCastBestInCategory
	actionRegistry["spawn_mob"] = actSpawnMob
	actionRegistry["add_temp_exit"] = actAddTempExit
	actionRegistry["set_state"] = actSetState
	actionRegistry["command"] = actCommand
	actionRegistry["command_best_of"] = actCommandBestOf

	// New actions for boss mob / quest NPC behavior
	actionRegistry["summon_companion"] = actSummonCompanion
	actionRegistry["set_room_locked"] = actSetRoomLocked
	actionRegistry["spawn_item_in_room"] = actSpawnItemInRoom
	actionRegistry["add_buff"] = actAddBuff
	actionRegistry["command_mob"] = actCommandMob
	actionRegistry["give_item_multiple"] = actGiveItemMultiple
	actionRegistry["set_misc_data"] = actSetMiscData
	actionRegistry["increment_state"] = actIncrementState
	actionRegistry["decrement_state"] = actDecrementState
	actionRegistry["grant_quest_to_user"] = actGrantQuest // alias for grant_quest

	// New actions for room behavior trees
	actionRegistry["mob_say"] = actMobSay
	actionRegistry["mob_emote"] = actMobEmote
	actionRegistry["grant_mutation"] = actGrantMutation
	actionRegistry["send_user_text"] = actSendUserText
	actionRegistry["send_room_text"] = actSendRoomText
	actionRegistry["intercept"] = actIntercept
	actionRegistry["remove_buff"] = actRemoveBuff
	actionRegistry["move_player"] = actMovePlayer
	actionRegistry["create_instance"] = actCreateInstance
	actionRegistry["open_instance_portal"] = actOpenInstancePortal
}

// LookupAction returns the action function for the given name,
// or nil if not found.
func LookupAction(name string) ActionFunc {
	return actionRegistry[name]
}

// ActionNode wraps a registered action function with its params.
type ActionNode struct {
	Name   string
	Params map[string]any
	Fn     ActionFunc
}

// delayedActions is the set of action names that are subject to
// perception-scaled reaction delays. Internal bookkeeping actions
// (state, quest, item) remain instant.
var delayedActions = map[string]bool{
	"respond":      true,
	"say":          true,
	"emote":        true,
	"attack":       true,
	"flee":         true,
	"cast":         true,
	"move":         true,
	"add_buff":     true,
	"command_mob":  true,
	"mob_say":      true,
	"mob_emote":    true,
}

func (n *ActionNode) Evaluate(ctx *EvalContext) Result {
	// Static delay check first — bypasses perception-scaled timing.
	if staticDelay, ok := n.Params["delay"]; ok {
		var delaySec float64
		switch v := staticDelay.(type) {
		case float64:
			delaySec = v
		case int:
			delaySec = float64(v)
		}
		if delaySec > 0 {
			params := n.Params
			fn := n.Fn
			evalCtx := &EvalContext{
				Event:       ctx.Event,
				MobState:    ctx.MobState,
				MobId:       ctx.MobId,
				InstanceId:  ctx.InstanceId,
				RoomId:      ctx.RoomId,
				MobName:     ctx.MobName,
				Intercepted: ctx.Intercepted,
			}
			dur := time.Duration(delaySec * float64(time.Second))
			GetEngine().QueueDelayed(dur, func() {
				fn(params, evalCtx)
			})
			return Success
		}
	}
	if delayedActions[n.Name] {
		delay := calcReactionDelay(ctx.InstanceId)
		if delay > 0 {
			params := n.Params
			fn := n.Fn
			evalCtx := &EvalContext{
				Event:       ctx.Event,
				MobState:    ctx.MobState,
				MobId:       ctx.MobId,
				InstanceId:  ctx.InstanceId,
				RoomId:      ctx.RoomId,
				MobName:     ctx.MobName,
				Intercepted: ctx.Intercepted,
			}
			GetEngine().QueueDelayed(delay, func() {
				fn(params, evalCtx)
			})
			return Success
		}
	}
	return n.Fn(n.Params, ctx)
}
