package behaviortree

// actions_dialogue.go — dialogue/text output actions:
// actRespond, actSay, actEmote, actSendUserText, actSendRoomText,
// actMobSay, actMobEmote

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/textutil"
	"github.com/GoMudEngine/GoMud/internal/users"
)

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
		user.SendText(messaging.CategoryNPCDialogue, textutil.SubstituteTokens(userText, tokenCtx))
	}

	roomText := getStringParam(params, "room_text")
	if roomText != "" {
		room := rooms.LoadRoom(ctx.RoomId)
		if room != nil {
			room.SendTextVisual(messaging.CategoryNPCDialogue, textutil.SubstituteTokens(roomText, tokenCtx), ctx.Event.UserId)
		}
	}

	hints := getStringParam(params, "hints")
	if hints != "" {
		user.SendText(messaging.CategoryDialogueHint, fmt.Sprintf(`<ansi fg="181">  [%s]</ansi>`, hints))
	}

	return Success
}

func actSay(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	// A sleeping mob stays silent — no ambient or player-triggered chatter.
	if mob.Character.HasBuffFlag(buffs.Sleeping) {
		return Success
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
	// A sleeping mob doesn't emote — suppress idle flavor and the
	// player_enter greetings that fire from the noncombat archetypes.
	if mob.Character.HasBuffFlag(buffs.Sleeping) {
		return Success
	}
	text := getStringParam(params, "text")
	if text == "" {
		return Failure
	}
	mob.Command("emote " + text)
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
	user.SendText(messaging.CategoryNPCDialogue, text)
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
	room.SendText(messaging.CategoryMobEmote, text)
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
