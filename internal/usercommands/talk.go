package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dialogue"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/llm"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/quests"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Talk(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	args := util.SplitButRespectQuotes(rest)

	if len(args) == 0 {
		user.SendText(messaging.CategorySystem, `Talk to whom?`)
		return true, nil
	}

	// Support "talk to <npc>" as well as "talk <npc>"
	if strings.ToLower(args[0]) == `to` {
		if len(args) < 2 {
			user.SendText(messaging.CategorySystem, `Talk to whom?`)
			return true, nil
		}
		args = args[1:]
	}

	searchName := args[0]

	target, err := actions.ResolveTargetActor(room, searchName)
	if err != nil || target.IsPlayer() {
		user.SendText(messaging.CategorySystem, `Talk to whom?`)
		return true, nil
	}
	mob := target.(*actions.MobActor).Mob
	mobId := mob.InstanceId

	room.SendTextVisual(messaging.CategoryMobEmote, fmt.Sprintf(`<ansi fg="username">%s</ansi> approaches <ansi fg="mobname">%s</ansi> for a conversation.`, user.Character.Name, mob.Character.Name), user.UserId)

	// Build PlayerState for quest/item gating in dialogue
	ps := buildPlayerState(user)

	jsHandled := false

	// LLM greeting path: fires if JS didn't handle it and the mob has an LLM profile.
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
		llm.AskAsync(mob.LLMProfile, string(cfg.Endpoint), int(cfg.Timeout),
			mobIdCopy, llmCtx, `greet the player`,
			func(response string) {
				m := mobs.GetInstance(mobIdCopy)
				if m != nil {
					// Untrusted LLM output — see the matching comment in
					// ask.go. Authored dialogue below is left unescaped
					// because its <ansi> markup is legitimate.
					m.Command(`say ` + util.EscapeAnsiTags(response))
					dialogue.UpdateMemory(mobIdCopy, userIdCopy, "", nil, "greet")
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
					if greetText, hints, ok := dialogue.Greet(df, mobIdCopy, user.UserId, ps); ok {
						m.Command(`say ` + greetText)
						if hints != `` {
							if u := users.GetByUserId(user.UserId); u != nil {
								u.SendText(messaging.CategoryDialogueHint, fmt.Sprintf(`<ansi fg="181">  [%s]</ansi>`, hints))
							}
						}
					} else if response, moodChange, ok := dialogue.Match(df, mobIdCopy, ``, ps); ok {
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
			if greetText, hints, ok := dialogue.Greet(df, mobId, user.UserId, ps); ok {
				mob.Command(`say ` + greetText)
				if hints != `` {
					user.SendText(messaging.CategoryDialogueHint, fmt.Sprintf(`<ansi fg="181">  [%s]</ansi>`, hints))
				}
			} else if response, moodChange, ok := dialogue.Match(df, mobId, ``, ps); ok {
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

	room.SendTextToExits(`You hear someone talking.`, true)

	return true, nil
}

// buildPlayerState creates a PlayerState from the user's character for
// quest/item gating in dialogue calls.
func buildPlayerState(user *users.UserRecord) *dialogue.PlayerState {
	return &dialogue.PlayerState{
		HasQuest: user.Character.HasQuest,
		HasItem: func(itemId int) bool {
			for _, itm := range user.Character.GetAllBackpackItems() {
				if itm.ItemId == itemId {
					return true
				}
			}
			return false
		},
		RemoveItem: func(itemId int) bool {
			for _, itm := range user.Character.GetAllBackpackItems() {
				if itm.ItemId == itemId {
					return user.Character.RemoveItem(itm)
				}
			}
			return false
		},
		GiveQuest: func(token string) {
			events.AddToQueue(events.Quest{
				UserId:     user.UserId,
				QuestToken: token,
			})
		},
		GiveItem: func(itemId int) bool {
			newItem := items.New(itemId)
			if newItem.ItemId <= 0 {
				// Data bug (givesItem points at no item). Report failure so
				// the engine also withholds grantsQuest — granting a quest
				// whose item can never exist is the soft-lock, loudly.
				mudlog.Error("talk.GiveItem", "msg", "dialogue givesItem references invalid item",
					"userId", user.UserId, "itemId", itemId)
				return false
			}
			// StoreItem refuses past twice carry capacity (the "crushed"
			// encumbrance tier). Claiming delivery anyway — and worse,
			// queueing ItemOwnership{Gained:true}, which fires item_gain
			// quest triggers and auto-grants QuestTokens — advanced quest
			// state on an item the player never received.
			if !user.Character.StoreItem(newItem) {
				user.SendText(messaging.CategorySystem, fmt.Sprintf(
					`You are carrying too much to take the <ansi fg="itemname">%s</ansi>. `+
						`Put something down, then ask again.`, newItem.DisplayName()))
				mudlog.Warn("talk.GiveItem", "msg", "player could not carry dialogue item",
					"userId", user.UserId, "itemId", itemId)
				return false
			}
			user.SendText(messaging.CategoryLoot, fmt.Sprintf(`You receive a <ansi fg="itemname">%s</ansi>.`, newItem.DisplayName()))
			events.AddToQueue(events.ItemOwnership{
				UserId: user.UserId,
				Item:   newItem,
				Gained: true,
			})
			return true
		},
		GetQuestFlag: func(key string) string {
			return user.Character.GetQuestFlag(key)
		},
		SetQuestFlag: func(key, value string) {
			user.Character.SetQuestFlag(key, value)
		},
		BumpRep: func(faction string, delta int) {
			factions.BumpRep(faction, user.UserId, delta)
		},
		GiveGold: func(amount int) {
			user.Character.Gold += amount
			user.SendText(messaging.CategoryLoot, fmt.Sprintf(`You receive <ansi fg="gold">%d gold</ansi>.`, amount))
			events.AddToQueue(events.EquipmentChange{
				UserId:     user.UserId,
				GoldChange: amount,
			})
		},
		HasOwnMasterwork: func(skillMin int) bool {
			return user.Character.HasOwnMasterwork(skillMin)
		},
	}
}

// buildPlayerCondition returns a short description of the player's health and
// death history for injection into LLM conversation context.
func buildPlayerCondition(user *users.UserRecord) string {
	c := user.Character
	pct := 100
	if c.HealthMax.Value > 0 {
		pct = int(float64(c.Health) / float64(c.HealthMax.Value) * 100)
	}
	deaths := c.KD.GetMobDeaths()
	var cond string
	switch {
	case pct <= 15:
		cond = "near death"
	case pct <= 50:
		cond = "seriously wounded"
	case pct <= 80:
		cond = "lightly wounded"
	default:
		cond = "healthy"
	}
	if deaths == 1 {
		return cond + ", has died once"
	} else if deaths > 1 {
		return fmt.Sprintf("%s, has died %d times", cond, deaths)
	}
	return cond
}

// buildTutorialContext checks the player's Sanctum Trials quest progress and
// returns a structured summary string for LLM context injection.
func buildTutorialContext(user *users.UserRecord) string {
	const questId = 1 // The Sanctum Trials
	steps := []string{
		"start", "mutation",
		"shopping_arrive", "shopping_buy", "shopping_equip",
		"combat_arrive", "combat_defeat",
		"crafting_arrive", "crafting_craft",
		"alchemy_arrive", "alchemy_craft",
		"wilderness_arrive", "wilderness_forage", "wilderness_track",
		"magic_arrive", "magic_cast",
		"cave", "warden", "end",
	}

	progress := user.Character.GetQuestProgress()
	currentStep, hasQuest := progress[questId]
	if !hasQuest {
		return "TUTORIAL PROGRESS: Player has not started the tutorial."
	}
	if currentStep == "end" {
		return "TUTORIAL PROGRESS: Player has completed all tutorial steps."
	}

	var completed, remaining []string
	found := false
	for _, s := range steps {
		if s == currentStep {
			found = true
			continue
		}
		if !found {
			completed = append(completed, s)
		} else {
			remaining = append(remaining, s)
		}
	}

	compStr := "none"
	if len(completed) > 0 {
		compStr = strings.Join(completed, ", ")
	}
	remStr := "none"
	if len(remaining) > 0 {
		remStr = strings.Join(remaining, ", ")
	}
	return fmt.Sprintf(
		"TUTORIAL PROGRESS: Current step: %s. Completed: %s. Remaining: %s.",
		currentStep, compStr, remStr,
	)
}

// buildQuestContext returns human-readable quest summaries for quests that
// are relevant to the given mob (matched by quest data's mob references or
// simply all active non-secret quests for now).
func buildQuestContext(user *users.UserRecord, mobId int) []string {
	progress := user.Character.GetQuestProgress()
	if len(progress) == 0 {
		return nil
	}

	var ctx []string
	for questId, step := range progress {
		if step == "end" {
			continue
		}
		token := quests.PartsToToken(questId, step)
		q := quests.GetQuest(token)
		if q == nil || q.Secret {
			continue
		}
		ctx = append(ctx, fmt.Sprintf("%s (step: %s)", q.Name, step))
	}
	return ctx
}
