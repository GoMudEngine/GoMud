package characters

import (
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// TrackSkillUse increments the usage counter for a specific skill.
// This data will be used by the progression system (Stage 3.2+) to
// determine skill advancement chances.
func (c *Character) TrackSkillUse(skillName string) {
	if c.SkillUseCount == nil {
		c.SkillUseCount = make(map[string]int)
	}
	c.SkillUseCount[skillName]++
}

// TrackStatUse increments the usage counter for a specific stat.
// This data will be used by the progression system (Stage 3.2+) to
// determine stat advancement chances.
func (c *Character) TrackStatUse(statName string) {
	if c.StatUseCount == nil {
		c.StatUseCount = make(map[string]int)
	}
	c.StatUseCount[statName]++
}

// OnSkillUse is called whenever a player uses a skill in gameplay.
// Currently just tracks usage. In Stage 3.2, this will also trigger
// progression chance calculations.
func (c *Character) OnSkillUse(skillName string) {
	c.TrackSkillUse(skillName)
	mudlog.Debug("Progression", "event", "skill_use", "skill", skillName, "character", c.Name)
}

// OnCriticalSuccess is called when a player lands a critical hit or
// achieves a critical success on a skill check.
// Currently just tracks the event. In Stage 3.2, critical successes
// will have a higher chance of triggering skill/stat progression.
func (c *Character) OnCriticalSuccess(context string) {
	c.TrackSkillUse("critical_success")
	mudlog.Debug("Progression", "event", "critical_success", "context", context, "character", c.Name)
}

// OnCriticalFailure is called when a player critically fails a skill
// check or combat action. In Stage 3.2, critical failures will also
// have a chance of triggering progression (learning from mistakes).
func (c *Character) OnCriticalFailure(context string) {
	c.TrackSkillUse("critical_failure")
	mudlog.Debug("Progression", "event", "critical_failure", "context", context, "character", c.Name)
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
