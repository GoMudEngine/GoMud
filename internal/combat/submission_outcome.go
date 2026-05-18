package combat

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

// ResolveSubmissionOutcome applies the attempt result to the
// attempter + recipient based on the attempter's SubmissionPolicy.
// Side effects per tier:
//
//   - Bad tier:   attempter knocked Prone, pair breaks to Standing.
//   - Neutral:    no-op (caller handles messaging for the miss).
//   - Success:    outcome resolved per attempter's SubmissionPolicy.
//   - Crit:       same as Success; also applies a 1-round Stunned
//     buff on the recipient when policy is mercy (T10 stub).
//
// Policy routing on Success/Crit:
//
//   - Mercy:    clean release; both return to Standing.
//   - Subdue:   death cascade, NoDeprogression (T8), gold transfer
//     fraction (T8).
//   - Cripple:  same as Subdue + broken-limb buff (T9) when the sub
//     type has a body-part target. Choke subs (RNC, Triangle,
//     Anaconda) degrade to Subdue because they don't break limbs.
//   - Lethal:   full death cascade (deprogression intact; T8
//     distinguishes this from Subdue/Cripple via flag).
//
// The defender's SurrenderPolicy is intentionally not consulted here
// — per the design spec, only a mercy-policy attempter honours the
// tap signal, and that check happens inside applyMercyRelease.
// Other policies ignore the tap as a realism call.
func ResolveSubmissionOutcome(
	attempter *characters.Character,
	recipient *characters.Character,
	result SubmissionAttemptResult,
	role Role,
) {
	switch result.Tier {
	case SubTierBad:
		applyBadTier(attempter, recipient)
	case SubTierNeutral:
		return // no-op; caller owns messaging
	case SubTierSuccess, SubTierCrit:
		applySuccessByPolicy(attempter, recipient, result)
	}
}

// applyBadTier handles the Sub-TierBad outcome: break the grapple to
// Standing, then knock the attempter Prone for at least 2 rounds.
func applyBadTier(attempter, recipient *characters.Character) {
	if err := position.TransitionPair(
		attempter, recipient, position.Standing,
		state.TransitionReason{Trigger: position.TriggerGrappleBreak},
	); err != nil {
		mudlog.Warn("submission bad-tier: TransitionPair → Standing failed",
			"attempter", attempter.Name, "err", err)
		return
	}
	_ = attempter.Position.TransitionToProne(
		position.ProneData{MinRecoveryRounds: 2},
		state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward},
	)
}

// applySuccessByPolicy dispatches to the correct outcome path based
// on the attempter's SubmissionPolicy. Also applies the Crit-tier
// Stunned buff stub (T10) when policy is mercy.
func applySuccessByPolicy(
	attempter *characters.Character,
	recipient *characters.Character,
	result SubmissionAttemptResult,
) {
	policy := attempter.SubmissionPolicy

	// Choke degradation: cripple on a choke-class sub becomes subdue
	// because chokes don't break body parts.
	bodyPart := position.CrippleBodyPart(result.SubType)
	if policy == characters.PolicyCripple && bodyPart == "" {
		policy = characters.PolicySubdue
	}

	// Crit: apply Stunned buff on recipient only when mercy keeps
	// the recipient in play. For subdue/cripple/lethal the recipient
	// enters the death cascade and the buff would be a no-op.
	if result.Tier == SubTierCrit && policy == characters.PolicyMercy {
		applyStunnedBuff(recipient)
	}

	switch policy {
	case characters.PolicyMercy:
		applyMercyRelease(attempter, recipient)
	case characters.PolicySubdue:
		applyDeathCascade(attempter, recipient,
			true /* noDeprogression */, false /* brokenLimb */, "")
	case characters.PolicyCripple:
		applyDeathCascade(attempter, recipient,
			true /* noDeprogression */, true /* brokenLimb */, bodyPart)
	case characters.PolicyLethal:
		applyDeathCascade(attempter, recipient,
			false /* deprogression normally */, false, "")
	}
}

// applyMercyRelease executes the mercy-policy path: clean grapple
// break to Standing. Neither combatant takes damage. A brief
// post-mercy recovery debuff (T9/4f) is stubbed below.
func applyMercyRelease(attempter, recipient *characters.Character) {
	if err := position.TransitionPair(
		attempter, recipient, position.Standing,
		state.TransitionReason{Trigger: position.TriggerGrappleBreak},
	); err != nil {
		mudlog.Warn("submission mercy release: TransitionPair → Standing failed",
			"attempter", attempter.Name, "err", err)
	}
	// T9/4f stub: optional post-mercy stamina drag on the recipient.
	// Placeholder — apply via recipient.AddBuff(<id>) once a
	// recovery-debuff buff YAML is registered.
}

// applyDeathCascade calls the existing Die() entry point and stubs
// the T8/T9 extensions clearly.
//
// noDeprogression — when true (subdue/cripple), the death cascade
// should skip stat/skill rollback. T8 extends DeadData with this flag
// and threads it through Death_PlayerCleanup + Death_PlayerCorpse.
// Until T8 lands, the cascade runs as a normal death.
//
// brokenLimb + brokenBodyPart — when true (cripple with a joint sub),
// a broken-limb buff is applied after the cascade. T9 wires the
// actual buff id + YAML; for T7 the stub is a no-op.
func applyDeathCascade(
	killer *characters.Character,
	victim *characters.Character,
	noDeprogression bool,
	brokenLimb bool,
	brokenBodyPart string,
) {
	cfg := configs.GetBalanceConfig()
	// goldFrac is recorded here for T8; the actual gold transfer lands
	// when DeadData gains the GoldLossFraction field.
	goldFrac := 0.0
	if noDeprogression {
		goldFrac = float64(cfg.SubGoldLossFraction)
	}

	killerRef := state.ActorRef{
		UserId:        killer.GetUserId(),
		MobInstanceId: killer.MobInstanceId,
	}

	// TriggerSubmission lets T8 observers distinguish this cascade from
	// a normal combat-damage kill. Until T8 extends DeadData, downstream
	// observers treat it the same as TriggerHealthZero.
	victim.Die(killerRef, life.TriggerSubmission)

	// T8 stubs: extend DeadData with NoDeprogression + GoldLossFraction
	// and modify Death_PlayerCleanup / Death_PlayerCorpse to read them.
	_ = noDeprogression // T8: pass into DeadData
	_ = goldFrac        // T8: pass into DeadData

	if brokenLimb {
		applyBrokenLimbBuff(victim, brokenBodyPart)
	}
}

// applyBrokenLimbBuff applies the chunk-4d broken-limb buff (id 83).
// Stub for T7 — full implementation lands in T9 after the buff YAML
// is registered.
func applyBrokenLimbBuff(victim *characters.Character, bodyPart string) {
	// T9: victim.AddBuff(83) — buff id 83 = broken_limb
	// The bodyPart string ("arm" / "shoulder") gates which stat
	// penalties the buff applies. Thread it into AddBuff context
	// once the buff YAML supports per-part parameterisation.
	_ = victim
	_ = bodyPart
}

// applyStunnedBuff applies the 1-round Stunned buff (id 84) defined
// in T10. Stub for T7 — full implementation lands in T10 after the
// buff YAML is registered.
func applyStunnedBuff(c *characters.Character) {
	// T10: c.AddBuff(84) — buff id 84 = submission-stunned (T10 yaml)
	_ = c
}
