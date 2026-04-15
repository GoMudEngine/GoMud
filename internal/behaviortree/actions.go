package behaviortree

import (
	"fmt"
	"time"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/textutil"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
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
	actionRegistry["spawn_mob"] = actSpawnMob
	actionRegistry["add_temp_exit"] = actAddTempExit
	actionRegistry["set_state"] = actSetState
	actionRegistry["command"] = actCommand

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
	"respond":     true,
	"say":         true,
	"emote":       true,
	"attack":      true,
	"flee":        true,
	"cast":        true,
	"move":        true,
	"add_buff":    true,
	"command_mob": true,
	"mob_say":     true,
	"mob_emote":   true,
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

// actReturnItem gives a fresh copy of the triggering event's item back
// to the player. Used in player_give handlers to reject/return items.
// Unlike give_item, no item_id param is needed — the item comes from
// ctx.Event.ItemId.
func actReturnItem(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	itemId := ctx.Event.ItemId
	if itemId == 0 {
		return Failure
	}
	item := items.New(itemId)
	if !user.Character.StoreItem(item) {
		return Failure
	}
	user.SendText(fmt.Sprintf("%s hands back the %s.\n", ctx.MobName, item.Name()))
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
	targetUserId := ctx.Event.UserId
	// If no specific target, pick a random player in the room
	if targetUserId == 0 {
		room := rooms.LoadRoom(ctx.RoomId)
		if room == nil {
			return Failure
		}
		players := room.GetPlayers()
		if len(players) == 0 {
			return Failure
		}
		targetUserId = players[util.Rand(len(players))]
	}
	mob.Character.SetAggro(targetUserId, 0, characters.DefaultAttack)
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

// actSummonCompanion spawns a mob and registers it as a companion of the
// acting mob (mob-owned companion). Uses the same Charm mechanism as the
// player manifestation system, with userId=0 to indicate mob ownership.
func actSummonCompanion(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	mobId := getIntParam(params, "mob_id")
	if mobId == 0 {
		return Failure
	}
	count := getIntParam(params, "count")
	if count <= 0 {
		count = 1
	}
	room := rooms.LoadRoom(ctx.RoomId)
	if room == nil {
		return Failure
	}
	manifestSkill := mob.Character.GetSkillLevel(skills.Manifestation)
	charisma := mob.Character.Stats.Charisma.ValueAdj
	basePool := getIntParam(params, "base_pool")
	if basePool <= 0 {
		basePool = 50
	}
	pool := characters.CalcCompanionStatPool(basePool, charisma, manifestSkill)
	for i := 0; i < count; i++ {
		companion := mobs.NewMobById(mobs.MobId(mobId), room.RoomId, pool)
		if companion == nil {
			continue
		}
		room.AddMob(companion.InstanceId)
		companion.Character.Charm(0, 99999, "")
		companion.Character.EndAggro()
		info := characters.CompanionInfo{
			MobId:      int(companion.MobId),
			InstanceId: companion.InstanceId,
			SourceType: characters.CompanionSummoned,
			Name:       companion.Character.Name,
			BaseName:   companion.Character.Name,
			AutoAssist: true,
		}
		mob.Character.AddCompanion(info)
	}
	return Success
}

// actSetRoomLocked locks or unlocks a named exit in the current room.
// params: direction (string), locked ("true"/"false", default true)
func actSetRoomLocked(params map[string]any, ctx *EvalContext) Result {
	room := rooms.LoadRoom(ctx.RoomId)
	if room == nil {
		return Failure
	}
	direction := getStringParam(params, "direction")
	if direction == "" {
		return Failure
	}
	locked := getStringParam(params, "locked") != "false"
	room.SetExitLock(direction, locked)
	return Success
}

// actSpawnItemInRoom drops an item on the floor of a room.
// params: item_id (int), room_id (int, default ctx.RoomId), chance (int 1-100, default 100)
func actSpawnItemInRoom(params map[string]any, ctx *EvalContext) Result {
	itemId := getIntParam(params, "item_id")
	if itemId == 0 {
		return Failure
	}
	roomId := getIntParam(params, "room_id")
	if roomId == 0 {
		roomId = ctx.RoomId
	}
	chance := getIntParam(params, "chance")
	if chance <= 0 {
		chance = 100
	}
	if util.Rand(100) >= chance {
		return Success // skipped by chance — not a failure
	}
	room := rooms.LoadRoom(roomId)
	if room == nil {
		return Failure
	}
	item := items.New(itemId)
	room.AddItem(item, false)
	return Success
}

// actAddBuff applies a buff to the acting mob.
// params: buff_id (int)
func actAddBuff(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	buffId := getIntParam(params, "buff_id")
	if buffId == 0 {
		return Failure
	}
	mob.AddBuff(buffId, "behaviortree")
	return Success
}

// actCommandMob finds the first mob in the room with the given mob_id and
// issues it a command string. Returns Failure if no matching mob is found.
// params: mob_id (int), cmd (string)
func actCommandMob(params map[string]any, ctx *EvalContext) Result {
	targetMobId := getIntParam(params, "mob_id")
	if targetMobId == 0 {
		return Failure
	}
	cmd := getStringParam(params, "cmd")
	if cmd == "" {
		return Failure
	}
	room := rooms.LoadRoom(ctx.RoomId)
	if room == nil {
		return Failure
	}
	for _, instId := range room.GetMobs(rooms.FindAll) {
		m := mobs.GetInstance(instId)
		if m != nil && int(m.MobId) == targetMobId {
			m.Command(cmd)
			return Success
		}
	}
	return Failure
}

// actGiveItemMultiple gives one or more copies of an item to the triggering user.
// params: item_id (int), count (int, default 1)
func actGiveItemMultiple(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	itemId := getIntParam(params, "item_id")
	if itemId == 0 {
		return Failure
	}
	count := getIntParam(params, "count")
	if count <= 0 {
		count = 1
	}
	for i := 0; i < count; i++ {
		item := items.New(itemId)
		user.Character.StoreItem(item)
	}
	return Success
}

// actSetMiscData stores an arbitrary string value on the triggering user's
// character. Useful for quest-branch tracking via misc data keys.
// params: key (string), value (string)
func actSetMiscData(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	key := getStringParam(params, "key")
	if key == "" {
		return Failure
	}
	value := getStringParam(params, "value")
	user.Character.SetMiscData(key, value)
	return Success
}

// actIncrementState adds amount (default 1) to a numeric MobState key.
// params: key (string), amount (int, default 1)
func actIncrementState(params map[string]any, ctx *EvalContext) Result {
	if ctx.MobState == nil {
		return Failure
	}
	key := getStringParam(params, "key")
	if key == "" {
		return Failure
	}
	amount := getIntParam(params, "amount")
	if amount == 0 {
		amount = 1
	}
	ctx.MobState.Set(key, ctx.MobState.GetInt(key)+amount)
	return Success
}

// actDecrementState subtracts amount (default 1) from a numeric MobState key.
// params: key (string), amount (int, default 1)
func actDecrementState(params map[string]any, ctx *EvalContext) Result {
	if ctx.MobState == nil {
		return Failure
	}
	key := getStringParam(params, "key")
	if key == "" {
		return Failure
	}
	amount := getIntParam(params, "amount")
	if amount == 0 {
		amount = 1
	}
	ctx.MobState.Set(key, ctx.MobState.GetInt(key)-amount)
	return Success
}

// actMobSay finds the first mob in the room with the given mob_id and makes
// it speak.
// params: mob_id (int), text (string)
func actMobSay(params map[string]any, ctx *EvalContext) Result {
	mobId := getIntParam(params, "mob_id")
	text := getStringParam(params, "text")
	room := rooms.LoadRoom(ctx.RoomId)
	if room == nil {
		return Failure
	}
	for _, instId := range room.GetMobs(rooms.FindAll) {
		m := mobs.GetInstance(instId)
		if m != nil && int(m.MobId) == mobId {
			m.Command("say " + text)
			return Success
		}
	}
	return Failure
}

// actMobEmote finds the first mob in the room with the given mob_id and makes
// it emote.
// params: mob_id (int), text (string)
func actMobEmote(params map[string]any, ctx *EvalContext) Result {
	mobId := getIntParam(params, "mob_id")
	text := getStringParam(params, "text")
	room := rooms.LoadRoom(ctx.RoomId)
	if room == nil {
		return Failure
	}
	for _, instId := range room.GetMobs(rooms.FindAll) {
		m := mobs.GetInstance(instId)
		if m != nil && int(m.MobId) == mobId {
			m.Command("emote " + text)
			return Success
		}
	}
	return Failure
}

// actGrantMutation rolls and grants a random mutation to the triggering player
// from the weighted acquisition pool. Returns Success even if no mutations are
// available (no eligible mutations is not an error).
func actGrantMutation(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	pool := mutations.GetWeightedPool(user.Character.Mutations)
	if len(pool) == 0 {
		return Success // no mutations available, but not an error
	}
	mutId := mutations.RollAcquisition(pool)
	if mutId == "" {
		return Success
	}
	if user.Character.Mutations == nil {
		user.Character.Mutations = make(map[string]int)
	}
	user.Character.Mutations[mutId] = 1
	user.Character.Validate()
	return Success
}

// actSendUserText sends a text message to the triggering player.
// params: text (string)
func actSendUserText(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	text := getStringParam(params, "text")
	user.SendText(text)
	return Success
}

// actSendRoomText sends a text message to everyone in the room.
// params: text (string)
func actSendRoomText(params map[string]any, ctx *EvalContext) Result {
	room := rooms.LoadRoom(ctx.RoomId)
	if room == nil {
		return Failure
	}
	text := getStringParam(params, "text")
	room.SendText(text)
	return Success
}

// actIntercept marks the event as intercepted, preventing the default command
// handler from processing it.
func actIntercept(params map[string]any, ctx *EvalContext) Result {
	ctx.Intercepted = true
	return Success
}

// actRemoveBuff removes a buff from the triggering player by buff ID.
// params: buff_id (int)
func actRemoveBuff(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	buffId := getIntParam(params, "buff_id")
	user.Character.RemoveBuff(buffId)
	return Success
}

// actMovePlayer teleports the triggering player to a target room.
// params: room_id (int)
func actMovePlayer(params map[string]any, ctx *EvalContext) Result {
	roomId := getIntParam(params, "room_id")
	if roomId == 0 {
		return Failure
	}
	rooms.MoveToRoom(ctx.Event.UserId, roomId)
	return Success
}
