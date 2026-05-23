package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
)

const (
	activeTrackingBuff = 86
	shadowingBuff      = 87
)

// MobDeathTrackingCleanup clears tracking/shadow state on any character
// (player or mob) that was tracking or shadowing the now-dead mob. Pairs
// with PlayerDespawn_TrackingCleanup for the symmetric logoff path.
//
// State cleared per pointing character:
//   - tracking-mob misc data (string match on CharacterName)
//   - tracking-display-count misc data (cleared alongside tracking-mob)
//   - shadow-target-mob misc data (int match on InstanceId)
//   - buff 86 (Active Tracking) — only if tracking-mob pointed at this mob
//   - buff 87 (Shadowing) — only if shadow-target-mob pointed at this mob
func MobDeathTrackingCleanup(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.MobDeath)
	if !ok {
		return events.Continue
	}

	// CharacterName is populated at death time on the event payload;
	// no need to fetch the instance or template.
	dyingName := evt.CharacterName
	dyingInstanceId := evt.InstanceId

	clearPointers := func(c interface {
		GetMiscData(string) any
		SetMiscData(string, any)
		RemoveBuff(int)
	}) {
		// Tracking by name.
		if dyingName != "" {
			if v := c.GetMiscData("tracking-mob"); v != nil {
				if s, ok := v.(string); ok && s == dyingName {
					c.SetMiscData("tracking-mob", nil)
					c.SetMiscData("tracking-display-count", nil)
					c.RemoveBuff(activeTrackingBuff)
				}
			}
		}
		// Shadow by InstanceId.
		if v := c.GetMiscData("shadow-target-mob"); v != nil {
			if id, ok := v.(int); ok && id == dyingInstanceId {
				c.SetMiscData("shadow-target-mob", nil)
				c.RemoveBuff(shadowingBuff)
			}
		}
	}

	// Walk all online users.
	for _, u := range users.GetAllActiveUsers() {
		if u == nil {
			continue
		}
		clearPointers(u.Character)
	}

	// Walk all active mob instances.
	for _, instId := range mobs.GetAllMobInstanceIds() {
		if instId == dyingInstanceId {
			continue
		}
		m := mobs.GetInstance(instId)
		if m == nil {
			continue
		}
		clearPointers(&m.Character)
	}

	return events.Continue
}
