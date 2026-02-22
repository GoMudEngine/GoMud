package characters

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// skillNameMap maps progression context names to actual skill tags.
// Can be used to alias legacy names to new skill tags.
var skillNameMap = map[string]string{}

// resolveSkillName maps a progression context name to the actual skill tag.
func resolveSkillName(name string) string {
	if mapped, ok := skillNameMap[name]; ok {
		return mapped
	}
	return name
}

// CalculateProgressionChance returns the probability (0.0–1.0) that a
// skill/stat progression event fires at the given virtual rank.
// The curve is exponential decay: ~30% at rank 0, ~1.5% near the soft cap,
// and very small above the soft cap.
func CalculateProgressionChance(currentRank int, softCap int) float64 {
	b := configs.GetBalanceConfig()
	if softCap <= 0 {
		softCap = 1
	}
	base := float64(b.BaseProgressionChance)
	decayBelow := float64(b.ProgressionDecayBelowCap)
	decayAbove := float64(b.ProgressionDecayAboveCap)
	if currentRank <= 0 {
		return base
	}
	ratio := float64(currentRank) / float64(softCap)
	if currentRank <= softCap {
		return base * math.Exp(-decayBelow*ratio)
	}
	// Above soft cap: very hard, continues exponential decay
	aboveCapFloor := base * math.Exp(-decayBelow) // value at exactly the cap
	return aboveCapFloor * math.Exp(-decayAbove*(ratio-1.0))
}

// CheckSkillProgression rolls against the progression chance for a skill.
// If the roll succeeds and DualProgressionMode is enabled, the skill level
// is actually increased (capped at 4). Otherwise, only a notification is sent.
// bonusMultiplier scales the chance (e.g. 2.0 for critical successes).
func (c *Character) CheckSkillProgression(skillName string, userId int, bonusMultiplier float64) {
	b := configs.GetBalanceConfig()
	virtualRank := c.GetSkillUseCount(skillName) / int(b.UsesPerRank)
	chance := CalculateProgressionChance(virtualRank, int(b.SkillSoftCap)) * bonusMultiplier * skills.GetProgressionMultiplier(skillName)
	if chance > 1.0 {
		chance = 1.0
	}

	// Roll: chance is 0.0–1.0, convert to 0–10000 for integer roll
	threshold := int(chance * 10000)
	roll := util.Rand(10000)

	mudlog.Debug("Progression", "check", "skill", "skill", skillName, "rank", virtualRank, "chance", fmt.Sprintf("%.2f%%", chance*100), "roll", roll, "threshold", threshold, "character", c.Name)

	if roll < threshold {
		actualSkill := resolveSkillName(skillName)

		if skills.SkillExists(actualSkill) {
			if c.IncreaseSkill(actualSkill) {
				newLevel := c.Skills[actualSkill]
				msg := fmt.Sprintf(`<ansi fg="magenta">***</ansi> Your <ansi fg="yellow">%s</ansi> skill improves to rank <ansi fg="yellow-bold">%d</ansi>! <ansi fg="magenta">***</ansi>`, actualSkill, newLevel)
				events.AddToQueue(events.Message{UserId: userId, Text: msg + "\n"})
			} else {
				msg := fmt.Sprintf(`<ansi fg="magenta">***</ansi> You feel your <ansi fg="yellow">%s</ansi> skills sharpening! <ansi fg="magenta">***</ansi>`, actualSkill)
				events.AddToQueue(events.Message{UserId: userId, Text: msg + "\n"})
			}
		} else {
			msg := fmt.Sprintf(`<ansi fg="magenta">***</ansi> You feel your <ansi fg="yellow">%s</ansi> skills sharpening! <ansi fg="magenta">***</ansi>`, skillName)
			events.AddToQueue(events.Message{UserId: userId, Text: msg + "\n"})
		}
	}
}

// CheckStatProgression rolls against the progression chance for a stat.
// If the roll succeeds and DualProgressionMode is enabled, the stat's
// Training value is increased by 1. Otherwise, only a notification is sent.
func (c *Character) CheckStatProgression(statName string, userId int, bonusMultiplier float64) {
	b := configs.GetBalanceConfig()
	virtualRank := c.GetStatUseCount(statName) / int(b.UsesPerRank)
	// If the actual stat value exceeds the soft cap, use it as a floor for the virtual rank.
	// This prevents characters with artificially high stats (e.g. admin accounts) from
	// exploiting the low use-count portion of the progression curve.
	if statVal := c.GetStatValue(statName); statVal > int(b.StatSoftCap) && statVal > virtualRank {
		virtualRank = statVal
	}
	chance := CalculateProgressionChance(virtualRank, int(b.StatSoftCap)) * bonusMultiplier
	if chance > 1.0 {
		chance = 1.0
	}

	threshold := int(chance * 10000)
	roll := util.Rand(10000)

	mudlog.Debug("Progression", "check", "stat", "stat", statName, "rank", virtualRank, "chance", fmt.Sprintf("%.2f%%", chance*100), "roll", roll, "threshold", threshold, "character", c.Name)

	if roll < threshold {
		if c.IncreaseStat(statName, 1) {
			msg := fmt.Sprintf(`<ansi fg="magenta">***</ansi> Your <ansi fg="yellow">%s</ansi> grows stronger! <ansi fg="magenta">***</ansi>`, statName)
			events.AddToQueue(events.Message{UserId: userId, Text: msg + "\n"})
		}
	}
}

// TrackSkillUse increments the usage counter for a specific skill.
func (c *Character) TrackSkillUse(skillName string) {
	if c.SkillUseCount == nil {
		c.SkillUseCount = make(map[string]int)
	}
	c.SkillUseCount[skillName]++
}

// TrackStatUse increments the usage counter for a specific stat.
func (c *Character) TrackStatUse(statName string) {
	if c.StatUseCount == nil {
		c.StatUseCount = make(map[string]int)
	}
	c.StatUseCount[statName]++
}

// OnStatUse is called whenever a player uses a stat in gameplay.
// Tracks usage and, if progression is enabled, rolls for stat advancement.
func (c *Character) OnStatUse(statName string, userId int) {
	c.TrackStatUse(statName)
	mudlog.Debug("Progression", "event", "stat_use", "stat", statName, "character", c.Name)

	if configs.GetGamePlayConfig().UseSkillProgression {
		c.CheckStatProgression(statName, userId, 1.0)
	}
}

// OnSkillUse is called whenever a player uses a skill in gameplay.
// Tracks usage and, if progression is enabled, rolls for skill advancement.
// Also auto-tracks and progresses the skill's primary governing stat.
func (c *Character) OnSkillUse(skillName string, userId int) {
	c.TrackSkillUse(skillName)
	mudlog.Debug("Progression", "event", "skill_use", "skill", skillName, "character", c.Name)

	if configs.GetGamePlayConfig().UseSkillProgression {
		c.CheckSkillProgression(skillName, userId, 1.0)
	}

	// Auto-track and progress the skill's primary governing stat
	if primaryStat := skills.GetSkillPrimaryStat(skillName); primaryStat != "" {
		c.OnStatUse(primaryStat, userId)
	}
}

// OnCriticalSuccess is called when a player lands a critical hit or
// achieves a critical success. Triggers progression checks with a
// 2x bonus multiplier for both the skill and related stats.
func (c *Character) OnCriticalSuccess(context string, userId int) {
	c.TrackSkillUse("critical_success")
	mudlog.Debug("Progression", "event", "critical_success", "context", context, "character", c.Name)

	if configs.GetGamePlayConfig().UseSkillProgression {
		msg := fmt.Sprintf(`<ansi fg="magenta">***</ansi> A moment of brilliance! Your <ansi fg="yellow">%s</ansi> technique improves! <ansi fg="magenta">***</ansi>`, context)
		events.AddToQueue(events.Message{UserId: userId, Text: msg + "\n"})
		c.CheckSkillProgression(context, userId, 2.0)
	}
}

// OnCriticalFailure is called when a player critically fails a skill
// check or combat action. Learning from mistakes — standard progression chance.
func (c *Character) OnCriticalFailure(context string, userId int) {
	c.TrackSkillUse("critical_failure")
	mudlog.Debug("Progression", "event", "critical_failure", "context", context, "character", c.Name)

	if configs.GetGamePlayConfig().UseSkillProgression {
		msg := fmt.Sprintf(`<ansi fg="red">!!!</ansi> You learn from your mistake! Your <ansi fg="yellow">%s</ansi> understanding deepens. <ansi fg="red">!!!</ansi>`, context)
		events.AddToQueue(events.Message{UserId: userId, Text: msg + "\n"})
		c.CheckSkillProgression(context, userId, 1.0)
	}
}

// OnFirstMobKill is called when a player kills a mob type for the first time.
// Triggers a bonus combat skill progression check.
func (c *Character) OnFirstMobKill(userId int) {
	mudlog.Debug("Progression", "event", "first_mob_kill", "character", c.Name)

	if configs.GetGamePlayConfig().UseSkillProgression {
		msg := `<ansi fg="magenta">***</ansi> Defeating a new foe hones your combat instincts! <ansi fg="magenta">***</ansi>`
		events.AddToQueue(events.Message{UserId: userId, Text: msg + "\n"})
		c.CheckSkillProgression("combat", userId, 2.0)
	}
}

// OnLowResource is called when a resource (health, stamina, conviction)
// drops below 25% of its maximum. Triggers a stat progression check
// for the related stat (e.g. low health → vitality progression).
func (c *Character) OnLowResource(resourceName string, relatedStat string, userId int) {
	mudlog.Debug("Progression", "event", "low_resource", "resource", resourceName, "stat", relatedStat, "character", c.Name)

	if configs.GetGamePlayConfig().UseSkillProgression {
		c.CheckStatProgression(relatedStat, userId, 1.5)
	}
}

// GetSkillUseCount returns how many times a skill has been used.
func (c *Character) GetSkillUseCount(skillName string) int {
	if c.SkillUseCount == nil {
		return 0
	}
	return c.SkillUseCount[skillName]
}

// GetStatUseCount returns how many times a stat has been checked.
func (c *Character) GetStatUseCount(statName string) int {
	if c.StatUseCount == nil {
		return 0
	}
	return c.StatUseCount[statName]
}
