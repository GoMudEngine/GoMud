package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dialogue"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/llm"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"

	"github.com/GoMudEngine/GoMud/internal/users"
)

// deliverDialogue executes the YAML dialogue lookup and has the mob respond.
// It is called both by the normal (non-LLM) path and the LLM-unavailable fallback.
func deliverDialogue(df *dialogue.DialogueFile, mob *mobs.Mob, mobInstanceId int, userId int, topic string, ps *dialogue.PlayerState) {
	if df != nil {
		if nodeText, hints, moodChange, ok := dialogue.TreeAdvance(df, mobInstanceId, userId, topic, ps); ok {
			mob.Command(`say ` + nodeText)
			if hints != `` {
				if u := users.GetByUserId(userId); u != nil {
					u.SendText(fmt.Sprintf(`<ansi fg="181">  [%s]</ansi>`, hints))
				}
			}
			dialogue.ShiftMood(mobInstanceId, moodChange, df.DefaultMood)
		} else if response, moodChange, ok := dialogue.Match(df, mobInstanceId, topic, ps); ok {
			mob.Command(`say ` + response)
			dialogue.ShiftMood(mobInstanceId, moodChange, df.DefaultMood)
		} else {
			mob.Command(`emote shakes their head.`)
		}
	} else {
		mob.Command(`emote shakes their head.`)
	}
}

func Ask(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	args := util.SplitButRespectQuotes(rest)

	if len(args) < 2 {

		for _, mId := range room.GetMobs(rooms.FindCharmed) {
			mob := mobs.GetInstance(mId)
			if mob == nil {
				continue
			}
			if mob.Character.IsCharmed(user.UserId) {
				mob.Command(`emote regards you blankly. It doesn't seem to understand.`)
				return true, nil
			}
		}

		user.SendText(`You must <ansi fg="command">ask</ansi> <ansi fg="mobname">someone</ansi> <ansi fg="yellow">something</ansi>`)
		return true, nil
	}

	searchName := args[0]

	// Only ask charmed players or mobs to do stuff
	target, err := actions.ResolveTargetActor(room, searchName)
	if err != nil {
		user.SendText(`ask who what?`)
		return true, nil
	}
	if target.IsPlayer() {
		user.SendText(`You can't ask another player.`)
		return true, nil
	}

	mob := target.(*actions.MobActor).Mob
	mobId := mob.InstanceId

	args = args[1:]

	if !mob.Character.IsCharmed() {
		room.SendTextVisual(fmt.Sprintf(`<ansi fg="username">%s</ansi> asks <ansi fg="mobname">%s</ansi> about "%s"`, user.Character.Name, mob.Character.Name, strings.Join(args, ` `)), user.UserId)
	}

	// players may type "ask <mob> to <do something>"
	if len(args) > 1 && strings.ToLower(args[0]) == `to` {
		args = args[1:]
	}
	if len(args) > 1 && strings.ToLower(args[0]) == `about` {
		args = args[1:]
	}

	// Companions can't be ordered around — they act on their own
	if mob.Character.IsCharmed(user.UserId) {
		mob.Command(`emote regards you blankly. It doesn't seem to understand.`)
		return true, nil
	}

	rest = strings.Join(args, ` `)

	// Quest engine: dialogue notification
	bridge := questengine.NewGameBridge(user, room.RoomId)
	questengine.GetEngine().Notify("dialogue", questengine.EventDetails{
		UserId: user.UserId,
		RoomId: room.RoomId,
		MobId:  int(mob.MobId),
		Topic:  rest,
	}, bridge, bridge)

	// Build PlayerState for quest/item gating in dialogue
	ps := buildPlayerState(user)

	// Behavior tree: try before JS
	if behaviortree.TryMobBehavior(mobId, behaviortree.EventContext{
		EventType: "player_ask",
		UserId:    user.UserId,
		Text:      rest,
		RoomId:    room.RoomId,
	}) {
		return true, nil
	}

	jsHandled := false

	// LLM path: fires if JS didn't handle it and the mob has an LLM profile configured.
	if !jsHandled && mob.LLMProfile != nil && bool(configs.GetLLMConfig().Enabled) {
		cfg := configs.GetLLMConfig()
		mem := dialogue.GetMemory(mobId, user.UserId)
		llmCtx := llm.ConversationContext{
			MobName:          mob.Character.Name,
			ZoneName:         mob.Zone,
			PlayerName:       user.Character.Name,
			CurrentMood:      string(dialogue.GetMood(mobId, mob.LLMProfile.DefaultMood)),
			RecentTopics:     mem.RecentTopics,
			QuestContext:     buildQuestContext(user, int(mob.MobId)),
			PlayerCondition:  buildPlayerCondition(user),
			TutorialProgress: buildTutorialContext(user),
		}
		mob.Command(`emote pauses thoughtfully.`)
		mobIdCopy := mobId
		userIdCopy := user.UserId
		restCopy := rest
		llm.AskAsync(mob.LLMProfile, string(cfg.Endpoint), int(cfg.Timeout),
			mobIdCopy, llmCtx, restCopy,
			func(response string) {
				m := mobs.GetInstance(mobIdCopy)
				if m != nil {
					m.Command(`say ` + response)
					dialogue.UpdateMemory(mobIdCopy, userIdCopy, "", nil, restCopy)
				}
			},
			func() {
				// LLM unavailable — fall through to YAML dialogue.
				m := mobs.GetInstance(mobIdCopy)
				if m != nil {
					df := dialogue.Load(int(m.MobId), m.Zone)
					deliverDialogue(df, m, mobIdCopy, user.UserId, restCopy, ps)
				}
			},
		)
		jsHandled = true // prevent double-response from YAML block below
	}

	if !jsHandled {
		df := dialogue.Load(int(mob.MobId), mob.Zone)
		deliverDialogue(df, mob, mobId, user.UserId, rest, ps)
	}

	room.SendTextToExits(`You hear someone talking.`, true)

	return true, nil
}
