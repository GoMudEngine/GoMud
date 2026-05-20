package characters

import (
	"fmt"
	"strconv"
	"strings"
)

// SubmissionPolicy is the controller-side disposition that resolves
// a successful submission outcome. Set ahead of combat via
// `set submission <policy>`; consulted by the engine when a sub
// locks. No per-round prompts.
type SubmissionPolicy int

const (
	PolicySubdue  SubmissionPolicy = iota // default — knock unconscious, take some gold, leave alive
	PolicyMercy                           // release cleanly, brief recovery debuff
	PolicyCripple                         // break limb (per sub type), take gold, leave alive
	PolicyLethal                          // finishing damage drain → full death path
)

func (p SubmissionPolicy) String() string {
	switch p {
	case PolicyMercy:
		return "mercy"
	case PolicySubdue:
		return "subdue"
	case PolicyCripple:
		return "cripple"
	case PolicyLethal:
		return "lethal"
	default:
		return "unknown"
	}
}

// ParseSubmissionPolicy converts a user-input string ("mercy" / "subdue"
// / "cripple" / "lethal") to a policy value. Case-insensitive. Empty
// string returns the default (subdue, true).
func ParseSubmissionPolicy(s string) (SubmissionPolicy, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return PolicySubdue, true
	case "mercy":
		return PolicyMercy, true
	case "subdue":
		return PolicySubdue, true
	case "cripple":
		return PolicyCripple, true
	case "lethal":
		return PolicyLethal, true
	default:
		return PolicySubdue, false
	}
}

// SurrenderMode is the defender-side disposition for sending a tap
// signal when a sub locks in. Only honored by controllers with mercy
// policy (per the spec's realism framing).
type SurrenderMode int

const (
	SurrenderAutoTap SurrenderMode = iota // default — tap when HP below threshold
	SurrenderNever
	SurrenderAlways
)

// SurrenderPolicy combines the mode with an HP-percentage threshold
// (only meaningful for SurrenderAutoTap).
type SurrenderPolicy struct {
	Mode           SurrenderMode `yaml:"mode"`
	HpPctThreshold int           `yaml:"hp_pct_threshold"` // 1-100; ignored unless Mode == SurrenderAutoTap
}

func (p SurrenderPolicy) String() string {
	switch p.Mode {
	case SurrenderNever:
		return "never"
	case SurrenderAlways:
		return "always"
	case SurrenderAutoTap:
		return fmt.Sprintf("auto-tap-below %d", p.HpPctThreshold)
	default:
		return "unknown"
	}
}

// ParseSurrenderPolicy parses "never" / "always" / "auto-tap-below <N>".
// Returns (policy, true) on success; (zero, false) on parse failure or
// out-of-range threshold.
func ParseSurrenderPolicy(s string) (SurrenderPolicy, bool) {
	t := strings.ToLower(strings.TrimSpace(s))
	switch {
	case t == "never":
		return SurrenderPolicy{Mode: SurrenderNever}, true
	case t == "always":
		return SurrenderPolicy{Mode: SurrenderAlways}, true
	case strings.HasPrefix(t, "auto-tap-below"):
		rest := strings.TrimSpace(strings.TrimPrefix(t, "auto-tap-below"))
		n, err := strconv.Atoi(rest)
		if err != nil || n < 1 || n > 100 {
			return SurrenderPolicy{}, false
		}
		return SurrenderPolicy{Mode: SurrenderAutoTap, HpPctThreshold: n}, true
	default:
		return SurrenderPolicy{}, false
	}
}

// DefaultSubmissionPolicyForArchetype returns the default sub policy
// for a mob archetype. Used during mob spawn (`newMobByIdInternal`)
// when the YAML doesn't provide a `submission_policy:` value.
func DefaultSubmissionPolicyForArchetype(archetype string) SubmissionPolicy {
	switch archetype {
	case "leader":
		return PolicyCripple
	case "defensive_caster", "civilian", "merchant":
		return PolicyMercy
	default:
		// predator, bandit, guard, tank_taunter, generic_fighter,
		// lookout, ambusher, and unrecognized → subdue
		return PolicySubdue
	}
}

// DefaultSurrenderPolicyForArchetype mirrors the above for defender-
// side defaults.
func DefaultSurrenderPolicyForArchetype(archetype string) SurrenderPolicy {
	switch archetype {
	case "civilian", "merchant":
		return SurrenderPolicy{Mode: SurrenderAlways}
	case "predator", "bandit", "guard", "tank_taunter", "leader":
		return SurrenderPolicy{Mode: SurrenderNever}
	case "defensive_caster":
		return SurrenderPolicy{Mode: SurrenderAutoTap, HpPctThreshold: 25}
	case "lookout", "ambusher":
		return SurrenderPolicy{Mode: SurrenderAutoTap, HpPctThreshold: 20}
	default:
		// generic_fighter, unknown
		return SurrenderPolicy{Mode: SurrenderAutoTap, HpPctThreshold: 10}
	}
}
