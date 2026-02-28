package characters

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// skillNameMap maps progression context names to actual skill tags.
// Can be used to alias legacy names to new skill tags.
var skillNameMap = map[string]string{}

// MobStatGainMessages contains room-visible messages when a mob gains a stat
// through use-based progression. Format verbs: %s = mob name (pre-wrapped in ansi).
var MobStatGainMessages = map[string]string{
	"strength":   `<ansi fg="mobname">%s</ansi> seems to grow more powerful.`,
	"dexterity":  `<ansi fg="mobname">%s</ansi> moves with increasing swiftness.`,
	"perception": `<ansi fg="mobname">%s</ansi>'s eyes sharpen with awareness.`,
	"vitality":   `<ansi fg="mobname">%s</ansi> looks tougher than before.`,
	"willpower":  `<ansi fg="mobname">%s</ansi> radiates a more focused presence.`,
	"charisma":   `<ansi fg="mobname">%s</ansi> projects a more commanding aura.`,
}

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
// If the roll succeeds, the skill level is increased. Returns true if
// progression fired (skill actually gained), false otherwise.
// bonusMultiplier scales the chance (e.g. 2.0 for critical successes).
func (c *Character) CheckSkillProgression(skillName string, userId int, bonusMultiplier float64) bool {
	b := configs.GetBalanceConfig()

	// Mob-specific gating
	if c.IsMob {
		if !bool(b.MobProgressionEnabled) {
			return false
		}
		actualSkill := resolveSkillName(skillName)
		if lvl, ok := c.Skills[actualSkill]; ok && lvl >= int(b.MobSkillCap) {
			return false // hard cap
		}
		bonusMultiplier *= float64(b.MobProgressionRate)
	}

	virtualRank := c.GetSkillUseCount(skillName) / int(b.UsesPerRank)
	// If the actual skill level exceeds the soft cap, use it as a floor for the virtual rank.
	// This prevents characters with artificially high skills (e.g. admin accounts) from
	// exploiting the low use-count portion of the progression curve.
	actualSkillName := resolveSkillName(skillName)
	if skillLevel, ok := c.Skills[actualSkillName]; ok && skillLevel > int(b.SkillSoftCap) && skillLevel > virtualRank {
		virtualRank = skillLevel
	}
	// Phase 24.2: Apply mutation skill progression multiplier
	mutSkillMult := 1.0 + mutations.GetSkillProgressionMultiplier(c.Mutations)
	// Phase 25.3: Skill Attunement buff doubles skill progression chance
	buffSkillMult := 1.0
	if c.HasBuffFlag(buffs.SkillProgress) {
		buffSkillMult = 2.0
	}
	chance := CalculateProgressionChance(virtualRank, int(b.SkillSoftCap)) * bonusMultiplier * skills.GetProgressionMultiplier(skillName) * mutSkillMult * buffSkillMult
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
				if userId > 0 {
					newLevel := c.Skills[actualSkill]
					msg := fmt.Sprintf(`<ansi fg="magenta">***</ansi> Your <ansi fg="yellow">%s</ansi> skill reaches <ansi fg="yellow-bold">%s</ansi>! <ansi fg="magenta">***</ansi>`, actualSkill, skills.GetSkillRankDescription(newLevel))
					events.AddToQueue(events.Message{UserId: userId, Text: msg + "\n"})
				}
			} else {
				if userId > 0 {
					msg := fmt.Sprintf(`<ansi fg="magenta">***</ansi> You feel your <ansi fg="yellow">%s</ansi> skills sharpening! <ansi fg="magenta">***</ansi>`, actualSkill)
					events.AddToQueue(events.Message{UserId: userId, Text: msg + "\n"})
				}
			}
		} else {
			if userId > 0 {
				msg := fmt.Sprintf(`<ansi fg="magenta">***</ansi> You feel your <ansi fg="yellow">%s</ansi> skills sharpening! <ansi fg="magenta">***</ansi>`, skillName)
				events.AddToQueue(events.Message{UserId: userId, Text: msg + "\n"})
			}
		}
		return true
	}
	return false
}

// CheckStatProgression rolls against the progression chance for a stat.
// If the roll succeeds, the stat's Training value is increased by 1.
// Returns true if progression fired (stat actually gained), false otherwise.
func (c *Character) CheckStatProgression(statName string, userId int, bonusMultiplier float64) bool {
	b := configs.GetBalanceConfig()

	// Mob-specific gating
	if c.IsMob {
		if !bool(b.MobProgressionEnabled) {
			return false
		}
		if c.GetStatValue(statName) >= int(b.MobStatCap) {
			return false // hard cap
		}
		bonusMultiplier *= float64(b.MobProgressionRate)
	}

	virtualRank := c.GetStatUseCount(statName) / int(b.UsesPerRank)
	// If the actual stat value exceeds the soft cap, use it as a floor for the virtual rank.
	// This prevents characters with artificially high stats (e.g. admin accounts) from
	// exploiting the low use-count portion of the progression curve.
	if statVal := c.GetStatValue(statName); statVal > int(b.StatSoftCap) && statVal > virtualRank {
		virtualRank = statVal
	}
	// Phase 24.2: Apply mutation stat progression multiplier
	mutStatMult := 1.0 + mutations.GetStatProgressionMultiplier(c.Mutations)
	// Phase 39.1: Per-stat progression multiplier from config
	statMult := b.GetStatProgressionMultiplier(statName)
	chance := CalculateProgressionChance(virtualRank, int(b.StatSoftCap)) * bonusMultiplier * mutStatMult * statMult
	if chance > 1.0 {
		chance = 1.0
	}

	threshold := int(chance * 10000)
	roll := util.Rand(10000)

	mudlog.Debug("Progression", "check", "stat", "stat", statName, "rank", virtualRank, "chance", fmt.Sprintf("%.2f%%", chance*100), "roll", roll, "threshold", threshold, "character", c.Name)

	if roll < threshold {
		if c.IncreaseStat(statName, 1) {
			if userId > 0 {
				msg := fmt.Sprintf(`<ansi fg="magenta">***</ansi> Your <ansi fg="yellow">%s</ansi> grows stronger! <ansi fg="magenta">***</ansi>`, statName)
				events.AddToQueue(events.Message{UserId: userId, Text: msg + "\n"})
			}
			return true
		}
	}
	return false
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

// OnStatUse is called whenever a character uses a stat in gameplay.
// Tracks usage and, if progression is enabled, rolls for stat advancement.
// Returns true if the stat actually increased.
func (c *Character) OnStatUse(statName string, userId int) bool {
	c.TrackStatUse(statName)
	mudlog.Debug("Progression", "event", "stat_use", "stat", statName, "character", c.Name)

	if configs.GetGamePlayConfig().UseSkillProgression {
		return c.CheckStatProgression(statName, userId, 1.0)
	}
	return false
}

// OnSkillUse is called whenever a character uses a skill in gameplay.
// Tracks usage and, if progression is enabled, rolls for skill advancement.
// Also auto-tracks and progresses the skill's primary governing stat.
// Returns true if the skill actually increased.
func (c *Character) OnSkillUse(skillName string, userId int) bool {
	c.TrackSkillUse(skillName)
	mudlog.Debug("Progression", "event", "skill_use", "skill", skillName, "character", c.Name)

	gained := false
	if configs.GetGamePlayConfig().UseSkillProgression {
		gained = c.CheckSkillProgression(skillName, userId, 1.0)
	}

	// Auto-track and progress the skill's primary governing stat
	if primaryStat := skills.GetSkillPrimaryStat(skillName); primaryStat != "" {
		c.OnStatUse(primaryStat, userId)
	}
	return gained
}

// OnCriticalSuccess is called when a character lands a critical hit or
// achieves a critical success. Triggers progression checks with a
// 2x bonus multiplier for both the skill and related stats.
func (c *Character) OnCriticalSuccess(context string, userId int) {
	c.TrackSkillUse("critical_success")
	mudlog.Debug("Progression", "event", "critical_success", "context", context, "character", c.Name)

	if configs.GetGamePlayConfig().UseSkillProgression {
		if userId > 0 {
			msg := fmt.Sprintf(`<ansi fg="magenta">***</ansi> A moment of brilliance! Your <ansi fg="yellow">%s</ansi> technique improves! <ansi fg="magenta">***</ansi>`, context)
			events.AddToQueue(events.Message{UserId: userId, Text: msg + "\n"})
		}
		c.CheckSkillProgression(context, userId, 2.0)
	}
}

// OnCriticalFailure is called when a character critically fails a skill
// check or combat action. Learning from mistakes — standard progression chance.
func (c *Character) OnCriticalFailure(context string, userId int) {
	c.TrackSkillUse("critical_failure")
	mudlog.Debug("Progression", "event", "critical_failure", "context", context, "character", c.Name)

	if configs.GetGamePlayConfig().UseSkillProgression {
		if userId > 0 {
			msg := fmt.Sprintf(`<ansi fg="red">!!!</ansi> You learn from your mistake! Your <ansi fg="yellow">%s</ansi> understanding deepens. <ansi fg="red">!!!</ansi>`, context)
			events.AddToQueue(events.Message{UserId: userId, Text: msg + "\n"})
		}
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
