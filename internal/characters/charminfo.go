package characters

import (
	"github.com/GoMudEngine/GoMud/internal/skills"
	"slices"
)

const (
	CharmPermanent      = -1 // Never expires due to time
	CharmExpiredDespawn = `emote bows and bids you farewell, disappearing into the scenery;despawn charmed mob expired`
	CharmExpiredRevert  = `emote reverts to its old ways`
)

var ()

type CharmInfo struct {
	UserId          int    // Charmed or serving a player?
	RoundsRemaining int    // If -1, never expires
	ExpiredCommand  string // Any valid mob commands such as `emote bows and waves farewell;despawn`
}

func NewCharm(userId int, rounds int, expireCommand string) *CharmInfo {
	return &CharmInfo{UserId: userId, RoundsRemaining: rounds, ExpiredCommand: expireCommand}
}

func (ch *CharmInfo) Expire() {
	ch.RoundsRemaining = 0
}

func (c *Character) TrackCharmed(mobId int, add bool) {
	for pos, mobInstanceId := range c.CharmedMobs {
		if mobInstanceId == mobId {
			if !add {
				c.CharmedMobs = slices.Delete(c.CharmedMobs, pos, pos+1)
			}
			return
		}
	}
	c.CharmedMobs = append(c.CharmedMobs, mobId)
}

func (c *Character) GetCharmIds() []int {
	return append([]int{}, c.CharmedMobs...)
}

func (c *Character) Charm(userId int, rounds int, expireCommand string) {
	c.SetAdjective(`charmed`, true)
	c.EverCharmed = true
	c.Charmed = NewCharm(userId, rounds, expireCommand)
	if c.Aggro != nil && c.Aggro.UserId == userId {
		c.EndAggro()
	}
}

func (c *Character) GetCharmedUserId() int {
	if c.Charmed != nil {
		return c.Charmed.UserId
	}
	return 0
}

func (c *Character) IsCharmed(userId ...int) bool {

	if c.Charmed == nil {
		return false
	}

	if len(userId) == 0 {
		return c.Charmed != nil
	}

	if c.Charmed == nil {
		return false
	}
	return slices.Contains(userId, c.Charmed.UserId)
}

// Returns userId of whoever had charmed them
func (c *Character) RemoveCharm() int {
	charmUserId := 0
	c.SetAdjective(`charmed`, false)
	if c.Charmed != nil {
		charmUserId = c.Charmed.UserId
		c.Charmed = nil
	}
	return charmUserId
}

func (c *Character) GetMaxCharmedCreatures() int {
	// Taming is now handled via spellcasting; base charm capacity from spellcasting skill
	lvl := c.GetSkillLevel(skills.Spellcasting)
	return lvl + 1
}
