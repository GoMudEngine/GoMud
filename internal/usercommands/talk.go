package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dialogue"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/llm"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Talk(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	args := util.SplitButRespectQuotes(rest)

	if len(args) == 0 {
		user.SendText(`Talk to whom?`)
		return true, nil
	}

	// Support "talk to <npc>" as well as "talk <npc>"
	if strings.ToLower(args[0]) == `to` {
		if len(args) < 2 {
			user.SendText(`Talk to whom?`)
			return true, nil
		}
		args = args[1:]
	}

	searchName := args[0]

	_, mobId := room.FindByName(searchName)

	if mobId <= 0 {
		user.SendText(`Talk to whom?`)
		return true, nil
	}

	mob := mobs.GetInstance(mobId)
	if mob == nil {
		user.SendText(`Talk to whom?`)
		return true, nil
	}

	room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> approaches <ansi fg="mobname">%s</ansi> for a conversation.`, user.Character.Name, mob.Character.Name), user.UserId)

	// Try JS onAsk with empty string first for backward compatibility
	jsHandled := false
	if handled, err := scripting.TryMobScriptEvent(`onAsk`, mobId, user.UserId, `user`, map[string]any{"askText": ``}); err == nil && handled {
		jsHandled = true
	}

	// LLM greeting path: fires if JS didn't handle it and the mob has an LLM profile.
	if !jsHandled && mob.LLMProfile != nil && bool(configs.GetLLMConfig().Enabled) {
		cfg := configs.GetLLMConfig()
		mem := dialogue.GetMemory(mobId, user.UserId)
		llmCtx := llm.ConversationContext{
			MobName:      mob.Character.Name,
			ZoneName:     mob.Zone,
			PlayerName:   user.Character.Name,
			CurrentMood:  string(dialogue.GetMood(mobId, mob.LLMProfile.DefaultMood)),
			RecentTopics: mem.RecentTopics,
		}
		mob.Command(`emote pauses thoughtfully.`)
		mobIdCopy := mobId
		llm.AskAsync(mob.LLMProfile, string(cfg.Endpoint), int(cfg.Timeout),
			mobIdCopy, llmCtx, `greet the player`,
			func(response string) {
				m := mobs.GetInstance(mobIdCopy)
				if m != nil {
					m.Command(`say ` + response)
				}
			},
			func() {
				// LLM unavailable — fall through to YAML greeting.
				m := mobs.GetInstance(mobIdCopy)
				if m == nil {
					return
				}
				df := dialogue.Load(int(m.MobId), m.Zone)
				if df != nil {
					if greetText, hints, ok := dialogue.Greet(df, mobIdCopy, user.UserId); ok {
						m.Command(`say ` + greetText)
						if hints != `` {
							m.Command(`say ` + hints)
						}
					} else if response, moodChange, ok := dialogue.Match(df, mobIdCopy, ``); ok {
						m.Command(`say ` + response)
						dialogue.ShiftMood(mobIdCopy, moodChange, df.DefaultMood)
					} else {
						m.Command(`emote nods.`)
					}
				} else {
					m.Command(`emote nods.`)
				}
			},
		)
		jsHandled = true
	}

	if !jsHandled {
		df := dialogue.Load(int(mob.MobId), mob.Zone)
		if df != nil {
			if greetText, hints, ok := dialogue.Greet(df, mobId, user.UserId); ok {
				mob.Command(`say ` + greetText)
				if hints != `` {
					mob.Command(`say ` + hints)
				}
			} else if response, moodChange, ok := dialogue.Match(df, mobId, ``); ok {
				// no tree — try a greeting pattern match with empty topic
				mob.Command(`say ` + response)
				dialogue.ShiftMood(mobId, moodChange, df.DefaultMood)
			} else {
				mob.Command(`emote nods.`)
			}
		} else {
			mob.Command(`emote nods.`)
		}
	}

	// Track charisma use when initiating a conversation
	user.Character.OnStatUse("charisma", user.UserId)

	room.SendTextToExits(`You hear someone talking.`, true)

	return true, nil
}
