package hooks

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/quests"
	"github.com/GoMudEngine/GoMud/internal/questengine"
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

// statGrant is one parsed stat reward (stat name + additive amount).
type statGrant struct {
	stat   string
	amount int
}

// parseStatGrants parses a quest's stat_info reward into zero or more
// grants. Format: a comma-separated list of "stat:amount" entries, e.g.
// "strength:5" or "strength:3,vitality:2".
// Valid stat names (lowercase): strength, dexterity, perception,
// vitality, willpower, charisma.
// Malformed entries (no colon, unparseable amount, empty name) are skipped.
func parseStatGrants(statInfo string) []statGrant {
	if statInfo == `` {
		return nil
	}
	var grants []statGrant
	for _, entry := range strings.Split(statInfo, `,`) {
		details := strings.Split(strings.TrimSpace(entry), `:`)
		if len(details) <= 1 {
			continue
		}
		statName := strings.ToLower(strings.TrimSpace(details[0]))
		amount, err := strconv.Atoi(strings.TrimSpace(details[1]))
		if err != nil || statName == `` {
			continue
		}
		grants = append(grants, statGrant{stat: statName, amount: amount})
	}
	return grants
}

// parseRecipeGrants parses a quest's recipe_info reward into zero or more
// recipe IDs. Format: a comma-separated list of recipe IDs, e.g.
// "iron-dagger" or "iron-dagger,iron-buckler".
// Empty entries (from trailing commas, double-commas, etc.) are skipped.
func parseRecipeGrants(recipeInfo string) []string {
	if recipeInfo == `` {
		return nil
	}
	var ids []string
	for _, entry := range strings.Split(recipeInfo, `,`) {
		id := strings.TrimSpace(entry)
		if id == `` {
			continue
		}
		ids = append(ids, id)
	}
	return ids
}

// itemGrant is one parsed item-stockpile reward (item id + quantity).
type itemGrant struct {
	itemId int
	qty    int
}

// parseItemGrants parses a quest's item_info reward into zero or more item
// grants. Format: a comma-separated list of "itemid[:qty]" entries, e.g.
// "30036:3,30028:2,30058" (three healing salves, two antidotes, one firebomb).
// A bare itemid with no colon defaults to a quantity of one. Malformed entries
// (unparseable id, non-positive qty) are skipped. This backs multi-item
// "starter kit / stockpile" rewards, mirroring parseSkillGrants/parseStatGrants.
func parseItemGrants(itemInfo string) []itemGrant {
	if itemInfo == `` {
		return nil
	}
	var grants []itemGrant
	for _, entry := range strings.Split(itemInfo, `,`) {
		entry = strings.TrimSpace(entry)
		if entry == `` {
			continue
		}
		details := strings.Split(entry, `:`)
		itemId, err := strconv.Atoi(strings.TrimSpace(details[0]))
		if err != nil || itemId <= 0 {
			continue
		}
		qty := 1
		if len(details) > 1 {
			q, qErr := strconv.Atoi(strings.TrimSpace(details[1]))
			if qErr != nil || q <= 0 {
				continue
			}
			qty = q
		}
		grants = append(grants, itemGrant{itemId: itemId, qty: qty})
	}
	return grants
}

// statGainDescriptions maps lowercase stat names to a flavourful gain phrase.
// No hard numbers — per the project no-hard-numbers rule.
var statGainDescriptions = map[string]string{
	"strength":   "You feel notably stronger — your muscles carry a new permanence.",
	"dexterity":  "Your movements sharpen; a new quickness settles into your limbs.",
	"perception": "The world comes into slightly sharper focus, details you once missed now clear.",
	"vitality":   "A deep resilience takes root — your body feels more enduring than before.",
	"willpower":  "Your mind firms, a quiet resolve that wasn't there before now anchoring you.",
	"charisma":   "Something in your bearing shifts; others may find you more compelling.",
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

	// Repeatable-quest cooldown gate: refuse to (re)start a repeatable quest
	// whose post-completion cooldown has not yet elapsed. Parse the step
	// up-front so we can check before advancing the token.
	{
		_, incomingStep := quests.TokenToParts(evt.QuestToken)
		if incomingStep == `start` && questInfo.Repeatable &&
			questUser.Character.QuestCooldownActive(questInfo.QuestId) {
			questUser.SendText(messaging.CategorySystem,
				`You have put in the work recently. Rest a while before taking this on again.`)
			return events.Continue
		}
	}

	// Try to advance the quest. If it fails, check whether the quest engine
	// already set this token (GrantQuest sets it synchronously for chain
	// evaluation, then fires this event for rewards). In that case, proceed
	// with reward processing.
	freshlyGranted := questUser.Character.GiveQuestToken(evt.QuestToken)
	if !freshlyGranted {
		// Already at this step? The quest engine pre-set it. Continue to rewards.
		if !questUser.Character.HasQuest(evt.QuestToken) {
			return events.Continue
		}
	}

	// Bridge legacy/dialogue token grants into the questengine's quest_granted
	// triggers. The questengine fires quest_granted only from its own internal
	// grant chain, so a token granted by DIALOGUE (grantsQuest) or any other
	// legacy path would otherwise never fire a questengine quest_granted trigger.
	// Notify ONLY on a fresh grant: a questengine-initiated grant pre-sets the
	// token synchronously (GiveQuestToken returns false above), so it is skipped
	// here — preventing a double-fire of the same trigger.
	if freshlyGranted {
		qbridge := questengine.NewGameBridge(questUser, questUser.Character.RoomId)
		questengine.GetEngine().Notify("quest_granted", questengine.EventDetails{
			UserId:     questUser.UserId,
			RoomId:     questUser.Character.RoomId,
			QuestToken: evt.QuestToken,
		}, qbridge, qbridge)
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
		// Item-stockpile reward? Grants N of each listed item (see
		// parseItemGrants). Routes through StoreItem so potions auto-fill a
		// bandolier, components a component bag, etc. Reusable "starter kit"
		// reward, mirroring the skill/stat/recipe multi-grants above.
		for _, grant := range parseItemGrants(questInfo.Rewards.ItemInfo) {
			var firstName string
			granted := 0
			for i := 0; i < grant.qty; i++ {
				newItm := items.New(grant.itemId)
				if newItm.ItemId == 0 {
					break // unknown item id — skip the whole grant
				}
				firstName = newItm.NameSimple()
				questUser.Character.StoreItem(newItm)
				granted++
			}
			if granted == 0 {
				mudlog.Warn("Quest ItemReward", "token", evt.QuestToken,
					"itemId", grant.itemId, "error", "unknown item id — skipped")
				continue
			}
			if granted > 1 {
				questUser.SendText(messaging.CategoryLoot, fmt.Sprintf(`You receive <ansi fg="itemname">%s</ansi> (x%d)!`, firstName, granted))
			} else {
				questUser.SendText(messaging.CategoryLoot, fmt.Sprintf(`You receive <ansi fg="itemname">%s</ansi>!`, firstName))
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
		// Stat reward? Additive — permanently increases the named stat's
		// Training value. Supports one OR several stats (see parseStatGrants).
		// Unlike skill grants these are NOT floor-raises; each grant always
		// adds its amount (quests are one-shot linear, so re-grant isn't
		// a concern).
		for _, grant := range parseStatGrants(questInfo.Rewards.StatInfo) {
			if questUser.Character.IncreaseStat(grant.stat, grant.amount) {
				msg, ok := statGainDescriptions[grant.stat]
				if !ok {
					msg = "You feel a subtle but permanent improvement wash over you."
				}
				statUpTxt, _ := templates.Process("character/statup", msg, questUser.UserId)
				questUser.SendText(messaging.CategorySkillProgress, statUpTxt)
			} else {
				mudlog.Warn("Quest StatReward", "token", evt.QuestToken,
					"stat", grant.stat, "error", "unknown stat name — skipped")
			}
		}
		// Recipe reward? Grants each recipe ID to the player's KnownRecipes.
		// Silently skips recipes the player already knows (idempotent).
		for _, recipeId := range parseRecipeGrants(questInfo.Rewards.RecipeInfo) {
			if questUser.Character.LearnRecipe(recipeId) {
				name := recipeId // fallback: use the id if the recipe spec isn't found
				if spec := crafting.GetRecipe(recipeId); spec != nil {
					name = spec.Name
				}
				questUser.SendText(messaging.CategorySkillProgress, fmt.Sprintf(
					`<ansi fg="cyan">You have learned the recipe for </ansi><ansi fg="itemname">%s</ansi><ansi fg="cyan">.</ansi>`,
					name))
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

		// Repeatable quest: reset progress so it can be taken again, and stamp
		// a cooldown so it cannot be farmed back-to-back. (Non-repeatable
		// quests are unaffected — Repeatable defaults false.)
		if questInfo.Repeatable {
			questUser.Character.ClearQuestToken(evt.QuestToken)
			questUser.Character.SetQuestCooldown(questInfo.QuestId, uint64(questInfo.CooldownRounds))
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
