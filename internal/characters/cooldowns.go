package characters

import (
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/util"
	"maps"
)

type Cooldowns map[string]int

func (cd *Cooldowns) RoundTick() {
	if cd == nil || *cd == nil {
		return
	}
	for trackingTag := range *cd {
		(*cd)[trackingTag] = (*cd)[trackingTag] - 1
	}
}

func (cd *Cooldowns) Prune() {
	if cd == nil || *cd == nil {
		return
	}
	for trackingTag, cooldownRounds := range *cd {
		if cooldownRounds <= 0 {
			delete(*cd, trackingTag)
		}
	}
}

func (cd *Cooldowns) Try(trackingTag string, cooldownPeriod string) bool {
	if cd == nil || *cd == nil {
		if cd != nil {
			*cd = make(Cooldowns)
		} else {
			// If cd is nil pointer, can't initialize - this shouldn't happen
			return true
		}
	}

	cd.Prune()

	cooldownRounds := int(gametime.GetDate(1000000).AddPeriod(cooldownPeriod) - 1000000)

	if cooldownRounds < 1 {
		return true
	}

	if _, ok := (*cd)[trackingTag]; ok {
		if (*cd)[trackingTag] > 0 {
			return false
		}
	}

	(*cd)[trackingTag] = cooldownRounds
	return true
}

func (c *Character) PruneCooldowns() {
	if len(c.Cooldowns) == 0 {
		return
	}

	c.Cooldowns.Prune()
}

func (c *Character) GetCooldown(trackingTag string) int {
	if c.Cooldowns == nil {
		c.Cooldowns = make(Cooldowns)
	}
	return c.Cooldowns[trackingTag]
}

func (c *Character) GetAllCooldowns() map[string]int {

	ret := map[string]int{}

	if c.Cooldowns == nil {
		return ret
	}

	maps.Copy(ret, c.Cooldowns)

	return ret
}

func (c *Character) TryCooldown(trackingTag string, cooldownTime string) bool {
	if c.Cooldowns == nil {
		c.Cooldowns = make(Cooldowns)
	}

	return c.Cooldowns.Try(trackingTag, cooldownTime)
}

func (c *Character) TimerSet(name, period string) {
	if c.Timers == nil {
		c.Timers = map[string]gametime.RoundTimer{}
	}
	c.Timers[name] = gametime.RoundTimer{
		RoundStart: util.GetRoundCount(),
		Period:     period,
	}
}

func (c *Character) TimerExpired(name string) bool {
	if c.Timers == nil {
		return true
	}

	t, ok := c.Timers[name]

	if !ok {
		return true
	}

	if t.Expired() {
		delete(c.Timers, name)
		return true
	}

	return false
}

func (c *Character) TimerExists(name string) bool {
	if c.Timers == nil {
		return false
	}

	_, ok := c.Timers[name]
	return ok
}
