package hooks

// NewRound_Bloom.go — per-round Bloom drug lifecycle for online players.
//
// Three duties per tick:
//  1. Crash onset:    Communion (buff 90) just expired → apply Crash (buff 91).
//  2. Withdrawal:     addict has abstained past BloomWithdrawalOnsetRounds →
//                     ensure Withdrawal (buff 92) is active; message on first apply.
//  3. Addiction decay: one addiction point removed per BloomAddictionDecayRounds
//                     of abstinence; fully clean players have buff 92 cleared.
//
// The Communion→Crash transition uses the transient Character.BloomHadCommunion
// bool (yaml:"-") so the detection survives logout-proof buff-expiry timing:
// the flag is true while buff 90 is on, and cleared the round it goes away.
//
// BloomLastDoseRound is yaml:"-" (not persisted), so withdrawal and decay timers
// restart fresh after each login.  Addiction level (BloomAddiction) IS persisted.

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// BloomTick is the NewRound listener that drives the Bloom drug lifecycle.
// Registered in hooks.go after AutoHeal.
func BloomTick(e events.Event) events.ListenerReturn {
	evt := e.(events.NewRound)
	currentRound := evt.RoundNumber

	bal := configs.GetBalanceConfig()
	withdrawOnset := uint64(bal.BloomWithdrawalOnsetRounds)
	decayRounds := uint64(bal.BloomAddictionDecayRounds)

	for _, userId := range users.GetOnlineUserIds() {
		user := users.GetByUserId(userId)
		if user == nil {
			continue
		}
		// user.Character is *characters.Character — use directly.
		c := user.Character

		hasCommunion := c.HasBuff(90)

		// ── 1. Crash on Communion end ─────────────────────────────────────────
		// BloomHadCommunion was set to true last tick while buff 90 was active.
		// The moment it goes absent, apply the Crash and emit a grim message.
		// Scale: buff 91 baseline triggercount (75) * BloomCrashRoundsMult (default 2.5)
		// = 187 rounds of crash at default config.  Adjust both knobs together to
		// change the high:crash ratio without touching buff YAML.
		if c.BloomHadCommunion && !hasCommunion {
			_ = c.AddBuffScaled(91, float64(bal.BloomCrashRoundsMult))
			user.SendText(messaging.CategoryWarning,
				`The warmth is gone all at once. Your limbs feel heavy, `+
					`the world grey and much too sharp. You are in the crash.`)
		}
		// Always mirror the current state for the next tick's detection.
		c.BloomHadCommunion = hasCommunion

		// ── 2. Withdrawal ─────────────────────────────────────────────────────
		// Gate: must be addicted AND have a reference dose round (non-zero).
		// BloomLastDoseRound is zero until the player's first Bloom dose this
		// session, so this block is a no-op for freshly logged-in addicts until
		// they take another dose (known limitation of yaml:"-" field).
		if c.BloomAddiction > 0 && c.BloomLastDoseRound > 0 && withdrawOnset > 0 {
			sinceLastDose := currentRound - c.BloomLastDoseRound
			if sinceLastDose >= withdrawOnset {
				// Re-apply only when the buff has expired to avoid constant
				// timer resets that would make withdrawal permanent.
				if !c.HasBuff(92) {
					_ = c.AddBuffScaled(92, 1.0) // baseline 200 rounds
					user.SendText(messaging.CategoryWarning,
						`Your body turns on you. The shakes come first, then the craving `+
							`rises like a tide. You need Bloom.`)
				}
			}
		}

		// ── 3. Addiction decay ────────────────────────────────────────────────
		// One addiction point per BloomAddictionDecayRounds of abstinence.
		// After each decrement, advance BloomLastDoseRound by one period so
		// the next decrement fires exactly BloomAddictionDecayRounds later.
		if c.BloomAddiction > 0 && c.BloomLastDoseRound > 0 && decayRounds > 0 {
			sinceLastDose := currentRound - c.BloomLastDoseRound
			if sinceLastDose >= decayRounds {
				c.AddBloomAddiction(-1)
				// Slide the reference point forward one period.
				c.BloomLastDoseRound += decayRounds
				if c.BloomAddiction == 0 {
					// Fully clean: remove any lingering Withdrawal buff.
					c.RemoveBuff(92)
					user.SendText(messaging.CategoryWarning,
						`The craving has finally loosened its grip. You feel `+
							`clean — or as close to it as you're likely to get.`)
				}
			}
		}
	}

	return events.Continue
}
