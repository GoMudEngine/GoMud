package hooks

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/quests"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// skillGrant is one parsed skill reward (skill tag + target level).
type skillGrant struct {
	skill string
	level int
}

// parseSkillGrants parses a quest's skill_info reward into zero or more
// grants. Format: a comma-separated list of "skill:level" entries, e.g.
// "weapon-combat:1,unarmed-combat:1" or the legacy single "map:1".
// Malformed entries (no colon, unparseable level) are skipped.
func parseSkillGrants(skillInfo string) []skillGrant {
	if skillInfo == `` {
		return nil
	}
	var grants []skillGrant
	for _, entry := range strings.Split(skillInfo, `,`) {
		details := strings.Split(strings.TrimSpace(entry), `:`)
		if len(details) <= 1 {
			continue
		}
		skillName := strings.ToLower(strings.TrimSpace(details[0]))
		level, err := strconv.Atoi(strings.TrimSpace(details[1]))
		if err != nil || skillName == `` {
			continue
		}
		grants = append(grants, skillGrant{skill: skillName, level: level})
	}
	return grants
}

//
// Handles quest progress
//

func HandleQuestUpdate(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.Quest)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "Quest", "Actual Type", e.Type())
		return events.Cancel
	}

	//mudlog.Debug(`Event`, `type`, evt.Type(), `UserId`, evt.UserId, `QuestToken`, evt.QuestToken)

	// Give them a token
	remove := false
	if evt.QuestToken[0:1] == `-` {
		remove = true
		evt.QuestToken = evt.QuestToken[1:]
	}

	questInfo := quests.GetQuest(evt.QuestToken)
	if questInfo == nil {
		return events.Continue
	}

	questUser := users.GetByUserId(evt.UserId)
	if questUser == nil {
		return events.Continue
	}

	if remove {
		questUser.Character.ClearQuestToken(evt.QuestToken)
		return events.Continue
	}
	// Try to advance the quest. If it fails, check whether the quest engine
	// already set this token (GrantQuest sets it synchronously for chain
	// evaluation, then fires this event for rewards). In that case, proceed
	// with reward processing.
	if !questUser.Character.GiveQuestToken(evt.QuestToken) {
		// Already at this step? The quest engine pre-set it. Continue to rewards.
		if !questUser.Character.HasQuest(evt.QuestToken) {
			return events.Continue
		}
	}

	_, stepName := quests.TokenToParts(evt.QuestToken)
	if stepName == `start` {
		if !questInfo.Secret {

			questUser.EventLog.Add(`quest`, fmt.Sprintf(`Given a new quest: <ansi fg="questname">%s</ansi>`, questInfo.Name))

			questUpTxt, _ := templates.Process("character/questup", fmt.Sprintf(`You have been given a new quest: <ansi fg="questname">%s</ansi>!`, questInfo.Name), questUser.UserId)
			questUser.SendText(messaging.CategorySystem, questUpTxt)
		}
	} else if stepName == `end` {

		if !questInfo.Secret {

			questUser.EventLog.Add(`quest`, fmt.Sprintf(`Completed a quest: <ansi fg="questname">%s</ansi>`, questInfo.Name))

			questUpTxt, _ := templates.Process("character/questup", fmt.Sprintf(`You have completed the quest: <ansi fg="questname">%s</ansi>!`, questInfo.Name), questUser.UserId)
			questUser.SendText(messaging.CategorySystem, questUpTxt)
		}

		// Message to player?
		if len(questInfo.Rewards.PlayerMessage) > 0 {
			questUser.SendText(messaging.CategorySystem, questInfo.Rewards.PlayerMessage)
		}
		// Message to room?
		if len(questInfo.Rewards.RoomMessage) > 0 {
			if room := rooms.LoadRoom(questUser.Character.RoomId); room != nil {
				sendVisualRoomText(room, messaging.CategoryEmote, questInfo.Rewards.RoomMessage, questUser.UserId)
			}
		}
		// New quest to start?
		if len(questInfo.Rewards.QuestId) > 0 {

			events.AddToQueue(events.Quest{
				UserId:     questUser.UserId,
				QuestToken: questInfo.Rewards.QuestId,
			})

		}
		// Gold reward?
		if questInfo.Rewards.Gold > 0 {
			questUser.SendText(messaging.CategoryLoot, fmt.Sprintf(`You receive <ansi fg="gold">%d gold</ansi>!`, questInfo.Rewards.Gold))
			questUser.Character.Gold += questInfo.Rewards.Gold

			events.AddToQueue(events.EquipmentChange{
				UserId:     questUser.UserId,
				GoldChange: questInfo.Rewards.Gold,
			})

		}
		// Item reward?
		if questInfo.Rewards.ItemId > 0 {
			newItm := items.New(questInfo.Rewards.ItemId)
			questUser.SendText(messaging.CategoryLoot, fmt.Sprintf(`You receive <ansi fg="itemname">%s</ansi>!`, newItm.NameSimple()))
			questUser.Character.StoreItem(newItm)

			iSpec := newItm.GetSpec()
			if iSpec.QuestToken != `` {

				events.AddToQueue(events.Quest{
					UserId:     questUser.UserId,
					QuestToken: iSpec.QuestToken,
				})

			}
		}
		// Buff reward?
		if questInfo.Rewards.BuffId > 0 {
			questUser.AddBuff(questInfo.Rewards.BuffId, `quest`)
		}
		// Stage 3.5: XP rewards removed. Progression is skill-based.
		// Skill reward? Supports one OR several skills (see parseSkillGrants).
		// Each grant is a floor-raise — it never downgrades a player already
		// above the target (so a veteran replaying a newbie spoke keeps rank).
		for _, grant := range parseSkillGrants(questInfo.Rewards.SkillInfo) {
			currentLevel := questUser.Character.GetSkillLevel(skills.SkillTag(grant.skill))
			if currentLevel < grant.level {
				newLevel := questUser.Character.TrainSkill(grant.skill, grant.level)

				skillData := struct {
					SkillName  string
					SkillLevel int
				}{
					SkillName:  grant.skill,
					SkillLevel: newLevel,
				}
				skillUpTxt, _ := templates.Process("character/skillup", skillData, questUser.UserId)
				questUser.SendText(messaging.CategorySkillProgress, skillUpTxt)
			}
		}
		// Spell reward?
		if questInfo.Rewards.SpellId != "" {
			if questUser.Character.LearnSpell(questInfo.Rewards.SpellId) {
				if spellData := spells.GetSpell(questInfo.Rewards.SpellId); spellData != nil {
					questUser.SendText(messaging.CategorySkillProgress, fmt.Sprintf(
						`<ansi fg="magenta-bold">You have learned the spell: <ansi fg="cyan-bold">%s</ansi></ansi>`,
						spellData.Name))
				}
			}
		}
		// Move them to another room/area?
		if questInfo.Rewards.RoomId > 0 {
			questUser.SendText(messaging.CategorySystem, `You are suddenly moved to a new place!`)

			if room := rooms.LoadRoom(questUser.Character.RoomId); room != nil {
				sendVisualRoomText(room, messaging.CategoryEmote, fmt.Sprintf(`<ansi fg="username">%s</ansi> is suddenly moved to a new place!`, questUser.Character.Name), questUser.UserId)
			}

			rooms.MoveToRoom(questUser.UserId, questInfo.Rewards.RoomId)
		}
		// Faction reputation reward?
		if questInfo.Rewards.RepFaction != "" && questInfo.Rewards.RepAmount != 0 {
			factions.BumpRep(questInfo.Rewards.RepFaction, questUser.UserId, questInfo.Rewards.RepAmount)
		}
	} else {
		if !questInfo.Secret {

			questUser.EventLog.Add(`quest`, fmt.Sprintf(`Made progress on a quest: <ansi fg="questname">%s</ansi>`, questInfo.Name))

			questUpTxt, _ := templates.Process("character/questup", fmt.Sprintf(`You've made progress on the quest: <ansi fg="questname">%s</ansi>!`, questInfo.Name), questUser.UserId)
			questUser.SendText(messaging.CategorySystem, questUpTxt)
		}
	}

	return events.Continue
}
