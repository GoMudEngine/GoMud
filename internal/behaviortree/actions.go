package behaviortree

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/textutil"
	"github.com/GoMudEngine/GoMud/internal/users"
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
	actionRegistry["take_item"] = actTakeItem
	actionRegistry["give_gold"] = actGiveGold
	actionRegistry["take_gold"] = actTakeGold
	actionRegistry["move"] = actMove
	actionRegistry["attack"] = actAttack
	actionRegistry["flee"] = actFlee
	actionRegistry["cast"] = actCast
	actionRegistry["spawn_mob"] = actSpawnMob
	actionRegistry["add_temp_exit"] = actAddTempExit
	actionRegistry["set_state"] = actSetState
	actionRegistry["command"] = actCommand
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

func (n *ActionNode) Evaluate(ctx *EvalContext) Result {
	return n.Fn(n.Params, ctx)
}

// --- action implementations ---

func actRespond(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}

	tokenCtx := textutil.TokenContext{
		SourceName:      fmt.Sprintf(`<ansi fg="mobname">%s</ansi>`, mob.Character.Name),
		SourcePlainName: mob.Character.Name,
		TargetName:      user.Character.Name,
		TargetPlainName: user.Character.Name,
	}

	userText := getStringParam(params, "user_text")
	if userText != "" {
		user.SendText(textutil.SubstituteTokens(userText, tokenCtx))
	}

	roomText := getStringParam(params, "room_text")
	if roomText != "" {
		room := rooms.LoadRoom(ctx.RoomId)
		if room != nil {
			room.SendTextVisual(textutil.SubstituteTokens(roomText, tokenCtx), ctx.Event.UserId)
		}
	}

	hints := getStringParam(params, "hints")
	if hints != "" {
		user.SendText(fmt.Sprintf(`<ansi fg="181">  [%s]</ansi>`, hints))
	}

	return Success
}

func actSay(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	text := getStringParam(params, "text")
	if text == "" {
		return Failure
	}
	mob.Command("say " + text)
	return Success
}

func actEmote(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	text := getStringParam(params, "text")
	if text == "" {
		return Failure
	}
	mob.Command("emote " + text)
	return Success
}

func actGrantQuest(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	quest := getStringParam(params, "quest")
	if quest == "" {
		return Failure
	}
	user.Character.GiveQuestToken(quest)
	return Success
}

func actSetQuestFlag(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	key := getStringParam(params, "flag_key")
	value := getStringParam(params, "flag_value")
	if key == "" {
		return Failure
	}
	user.Character.SetQuestFlag(key, value)
	return Success
}

func actGiveItem(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	itemId := getIntParam(params, "item_id")
	if itemId == 0 {
		return Failure
	}
	item := items.New(itemId)
	if !user.Character.StoreItem(item) {
		return Failure
	}
	user.SendText(fmt.Sprintf("You receive a %s.\n", item.Name()))
	return Success
}

func actTakeItem(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	itemId := getIntParam(params, "item_id")
	if itemId == 0 {
		return Failure
	}
	for _, item := range user.Character.Items {
		if item.ItemId == itemId {
			user.Character.RemoveItem(item)
			return Success
		}
	}
	return Failure
}

func actGiveGold(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	amount := getIntParam(params, "amount")
	if amount <= 0 {
		return Failure
	}
	user.Character.Gold += amount
	user.SendText(fmt.Sprintf("You receive %d gold.\n", amount))
	return Success
}

func actTakeGold(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	amount := getIntParam(params, "amount")
	if amount <= 0 {
		return Failure
	}
	user.Character.Gold -= amount
	if user.Character.Gold < 0 {
		user.Character.Gold = 0
	}
	return Success
}

func actMove(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	direction := getStringParam(params, "direction")
	if direction == "" {
		return Failure
	}
	mob.Command("go " + direction)
	return Success
}

func actAttack(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if ctx.Event.UserId == 0 {
		return Failure
	}
	mob.Character.SetAggro(ctx.Event.UserId, 0, characters.DefaultAttack)
	return Success
}

func actFlee(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	mob.Command("flee")
	return Success
}

func actCast(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	spell := getStringParam(params, "spell")
	if spell == "" {
		return Failure
	}
	mob.Command("cast " + spell)
	return Success
}

func actSpawnMob(params map[string]any, ctx *EvalContext) Result {
	mobId := getIntParam(params, "mob_id")
	if mobId == 0 {
		return Failure
	}
	roomId := getIntParam(params, "room_id")
	if roomId == 0 {
		roomId = ctx.RoomId
	}
	spawned := mobs.NewMobById(mobs.MobId(mobId), roomId)
	if spawned == nil {
		return Failure
	}
	return Success
}

func actAddTempExit(params map[string]any, ctx *EvalContext) Result {
	room := rooms.LoadRoom(ctx.RoomId)
	if room == nil {
		return Failure
	}
	exitName := getStringParam(params, "exit_name")
	if exitName == "" {
		return Failure
	}
	roomId := getIntParam(params, "room_id")
	if roomId == 0 {
		return Failure
	}
	tempExit := exit.TemporaryRoomExit{
		RoomId:  roomId,
		Title:   getStringParam(params, "title"),
		UserId:  ctx.Event.UserId,
		Expires: getStringParam(params, "expires"),
	}
	room.AddTemporaryExit(exitName, tempExit)
	return Success
}

func actSetState(params map[string]any, ctx *EvalContext) Result {
	if ctx.MobState == nil {
		return Failure
	}
	key := getStringParam(params, "key")
	if key == "" {
		return Failure
	}
	value := params["value"]
	ctx.MobState.Set(key, value)
	return Success
}

func actCommand(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	cmd := getStringParam(params, "cmd")
	if cmd == "" {
		return Failure
	}
	mob.Command(cmd)
	return Success
}
