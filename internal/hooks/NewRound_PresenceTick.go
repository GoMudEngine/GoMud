// Round ticks for Presence transitions.
//
// Chunk 5 (2026-05-19): centralized timeout-driven transitions for both
// players (Active → Idle → AFK → Disconnected) and mobs (Active →
// Dormant → Despawning). Replaces scattered AFK checks in userrecord.go
// and the BoredomCounter increment path in lookfortrouble.go.
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/presence"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// PresenceTick walks all active characters once per round and fires
// timeout-driven Presence transitions. Ordering: must run AFTER
// DoCombat (so attack-driven Dormant→Active fires before this checks
// timeouts) and BEFORE IdleMobs (so Despawning mobs get their
// terminal-tick removal in the same round).
func PresenceTick(e events.Event) events.ListenerReturn {
	evt := e.(events.NewRound)
	srvCfg := configs.GetServerConfig()
	roundNow := evt.RoundNumber

	// === Player presence transitions ===
	idleAfter := uint64(srvCfg.PresenceIdleAfterRounds)
	afkAfter := uint64(srvCfg.PresenceAFKAfterRounds)
	disconnectAfter := uint64(srvCfg.PresenceDisconnectAfterRounds)

	for _, user := range users.GetAllActiveUsers() {
		if user.Character == nil || user.Character.Presence == nil {
			continue
		}
		li := user.GetLastInputRound()
		if li == 0 || roundNow < li {
			continue
		}
		elapsed := roundNow - li

		switch user.Character.Presence.State() {
		case presence.Active:
			if elapsed >= idleAfter {
				_ = user.Character.Presence.TransitionTo(presence.Idle,
					state.TransitionReason{Trigger: presence.TriggerTimeoutIdle})
			}
		case presence.Idle:
			if elapsed >= afkAfter {
				_ = user.Character.Presence.TransitionTo(presence.AFK,
					state.TransitionReason{Trigger: presence.TriggerTimeoutAFK})
			}
		case presence.AFK:
			if elapsed >= disconnectAfter {
				_ = user.Character.Presence.TransitionTo(presence.Disconnected,
					state.TransitionReason{Trigger: presence.TriggerTimeoutDisconnect})
			}
		}
	}

	// === Mob presence transitions ===
	dormantAfter := uint64(srvCfg.PresenceMobDormantAfterRounds)
	despawnAfter := uint64(srvCfg.PresenceMobDespawnAfterRounds)

	for _, mobId := range mobs.GetAllMobInstanceIds() {
		mob := mobs.GetInstance(mobId)
		if mob == nil || mob.Character.Presence == nil {
			continue
		}
		switch mob.Character.Presence.State() {
		case presence.Spawning:
			// Same-tick resolution → Active.
			_ = mob.Character.Presence.TransitionTo(presence.Active,
				state.TransitionReason{Trigger: presence.TriggerSpawnTickResolve})

		case presence.Active:
			// Bored check: rounds since last target found.
			ltf := mob.Character.LastTargetFoundRound
			if ltf > 0 && roundNow-ltf >= dormantAfter {
				_ = mob.Character.Presence.TransitionTo(presence.Dormant,
					state.TransitionReason{Trigger: presence.TriggerBored})
			}

		case presence.Dormant:
			// Despawn check: rounds since entering Dormant.
			// Use LastDormantEntryRound on Character; if 0, set it now.
			if mob.Character.LastDormantEntryRound == 0 {
				mob.Character.LastDormantEntryRound = roundNow
				continue
			}
			if roundNow-mob.Character.LastDormantEntryRound >= despawnAfter {
				_ = mob.Character.Presence.TransitionTo(presence.Despawning,
					state.TransitionReason{Trigger: presence.TriggerDormantTooLong})
			}
		}
	}

	return events.Continue
}
