package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// PlayerDespawnTrackingCleanup clears tracking/shadow state on any
// character (player or mob) that was tracking or shadowing the now-
// despawning player. Pairs with MobDeath_TrackingCleanup for the
// symmetric mob-death path.
//
// State cleared per pointing character:
//   - tracking-user misc (string match on CharacterName)
//   - tracking-display-count misc (cleared alongside tracking-user)
//   - shadow-target-user misc (int match on UserId)
//   - buff 86 (Active Tracking) — only if tracking-user state was on this char
//   - buff 87 (Shadowing) — only if shadow-target-user state was on this char
func PlayerDespawnTrackingCleanup(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.PlayerDespawn)
	if !ok {
		return events.Continue
	}

	leavingName := evt.CharacterName
	leavingUserId := evt.UserId

	clearPointers := func(c interface {
		GetMiscData(string) any
		SetMiscData(string, any)
		RemoveBuff(int)
	}) {
		// Tracking by name.
		if leavingName != "" {
			if v := c.GetMiscData("tracking-user"); v != nil {
				if s, ok := v.(string); ok && s == leavingName {
					c.SetMiscData("tracking-user", nil)
					c.SetMiscData("tracking-display-count", nil)
					c.RemoveBuff(activeTrackingBuff)
				}
			}
		}
		// Shadow by UserId.
		if v := c.GetMiscData("shadow-target-user"); v != nil {
			if id, ok := v.(int); ok && id == leavingUserId {
				c.SetMiscData("shadow-target-user", nil)
				c.RemoveBuff(shadowingBuff)
			}
		}
	}

	// Walk all other online users.
	for _, u := range users.GetAllActiveUsers() {
		if u == nil || u.UserId == leavingUserId {
			continue
		}
		clearPointers(u.Character)
	}

	// Walk all active mob instances.
	for _, instId := range mobs.GetAllMobInstanceIds() {
		m := mobs.GetInstance(instId)
		if m == nil {
			continue
		}
		clearPointers(&m.Character)
	}

	return events.Continue
}
