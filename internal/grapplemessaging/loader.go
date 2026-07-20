// Package grapplemessaging loads and renders flavor templates for
// grapple outcomes (advance, degrade, reverse, escape, hold,
// striking apex). Templates live in
// _datafiles/world/dogmud/messaging/grapple_outcomes.yaml.
//
// Consumer is internal/hooks/Position_GrappleTick.go via the
// RenderOutcome function (T9).
package grapplemessaging

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// TemplateTriad holds the three speaker-variant template lists for
// a single outcome key. controller is shown to the controller side
// (second-person "you"); controlled is shown to the controlled side
// (second-person "you", from their POV); observers is broadcast to
// everyone else in the room (third-person).
type TemplateTriad struct {
	Controller []string `yaml:"controller"`
	Controlled []string `yaml:"controlled"`
	Observers  []string `yaml:"observers"`
}

// GradientTriad holds the three speaker-variant template lists for
// a gradient (ControlLevel boundary-crossing) event. Self is shown
// to the character whose state changed; partner is shown to the
// other side of the grapple; observers is broadcast to the room.
//
// Different from TemplateTriad's controller/controlled/observers
// semantics — gradients fire per-character (self), not per-role.
type GradientTriad struct {
	Self      []string `yaml:"self"`
	Partner   []string `yaml:"partner"`
	Observers []string `yaml:"observers"`
}

// Library is the parsed in-memory template store. Keys for each map
// follow spec §7.1 conventions:
//   - Advancements:  "<source>_to_<target>" (e.g. "clinch_to_mount")
//   - Degradations:  "<source>_to_<target>" (e.g. "mount_to_sidecontrol")
//   - Reversals:     "<source>_reverse" for realism-exception sources;
//     "generic_reverse" as fallback.
//   - Escapes:       "generic_escape" (only key for now)
//   - Holds:         "<context>_hold" (e.g. "clinch_hold",
//     "ground_hold_generic")
//   - StrikingApex:  Single-speaker (no triad); "mount_strike_flavor"
//     is the only key currently.
//   - Gradients:     Boundary-crossing (ControlLevel transitions);
//     "upper_boundary_down/up", "lower_boundary_down/up"
type Library struct {
	Advancements map[string]TemplateTriad `yaml:"advancements"`
	Degradations map[string]TemplateTriad `yaml:"degradations"`
	Reversals    map[string]TemplateTriad `yaml:"reversals"`
	Escapes      map[string]TemplateTriad `yaml:"escapes"`
	Holds        map[string]TemplateTriad `yaml:"holds"`
	StrikingApex map[string][]string      `yaml:"striking_apex"`
	Gradients    map[string]GradientTriad `yaml:"gradients"`
}

// Load reads and parses the grapple outcome template file. Returns
// an error if the file is missing, unreadable, or malformed YAML.
// Empty / partial libraries are valid — callers may apply their own
// completeness validation via ValidateCompleteness (T8).
func Load(path string) (*Library, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("grapplemessaging.Load: read %q: %w", path, err)
	}
	lib := &Library{}
	if err := yaml.Unmarshal(data, lib); err != nil {
		return nil, fmt.Errorf("grapplemessaging.Load: parse %q: %w", path, err)
	}
	// Initialize nil maps so consumers can index safely.
	if lib.Advancements == nil {
		lib.Advancements = map[string]TemplateTriad{}
	}
	if lib.Degradations == nil {
		lib.Degradations = map[string]TemplateTriad{}
	}
	if lib.Reversals == nil {
		lib.Reversals = map[string]TemplateTriad{}
	}
	if lib.Escapes == nil {
		lib.Escapes = map[string]TemplateTriad{}
	}
	if lib.Holds == nil {
		lib.Holds = map[string]TemplateTriad{}
	}
	if lib.StrikingApex == nil {
		lib.StrikingApex = map[string][]string{}
	}
	if lib.Gradients == nil {
		lib.Gradients = map[string]GradientTriad{}
	}
	return lib, nil
}

// Minimum templates per triad-speaker variant (spec §7.4).
const MinTemplatesPerSpeaker = 3

// Minimum templates for the Mount strike-apex single-speaker list.
const MinStrikingApexTemplates = 5

// RequiredAdvancementKeys lists the (source, target) pairs that the
// advancement messaging must cover. Derived from the spec §6.1 table
// — every cell in the table that produces an actual transition (not
// Hold) needs its own template triad.
var RequiredAdvancementKeys = []string{
	// Clinch posture variants (1-step)
	"clinch_to_mount",
	"clinch_to_sidecontrol",
	"clinch_to_background",
	"clinch_to_backstanding",

	// BackStanding all tiers
	"backstanding_to_background",

	// Mount 3-step only (1/2-step are Hold)
	"mount_to_background",

	// SideControl 1/2-step → Mount, 3-step → BackGround
	"sidecontrol_to_mount",
	"sidecontrol_to_background",

	// KneeOnBelly 1/2 → Mount; 3 → BackGround
	"kob_to_mount",
	"kob_to_background",

	// NorthSouth 1 → SC, 2 → Mount, 3 → BackGround
	"ns_to_sidecontrol",
	"ns_to_mount",
	"ns_to_background",

	// HalfGuard 1 → SC, 2 → Mount, 3 → BackGround
	"halfguard_to_sidecontrol",
	"halfguard_to_mount",
	"halfguard_to_background",

	// Guard 1 → SC, 2 → Mount, 3 → BackGround (controller bottom passes)
	"guard_to_sidecontrol",
	"guard_to_mount",
	"guard_to_background",

	// Turtle all tiers → BackGround
	"turtle_to_background",
}

// RequiredDegradationKeys lists the (source, target) pairs from spec
// §6.2 that produce actual transitions (Hold sources omitted).
var RequiredDegradationKeys = []string{
	"backstanding_to_clinch",
	"mount_to_sidecontrol",
	"sidecontrol_to_halfguard",
	"kob_to_sidecontrol",
	"ns_to_sidecontrol",
	"crucifix_to_background",
	"background_to_mount",
	"halfguard_to_guard",
}

// RequiredReversalKeys covers the 2 realism-exception reversals
// (Mount→Guard, BackGround→Mount) plus the generic fallback.
var RequiredReversalKeys = []string{
	"mount_reverse",      // → Guard with role swap
	"background_reverse", // → Mount with role swap
	"generic_reverse",    // any other source: same position, role swap
}

// RequiredEscapeKeys: single generic key for the always-to-Standing
// escape.
var RequiredEscapeKeys = []string{
	"generic_escape",
}

// RequiredHoldKeys: per-context hold flavor. Sparse — only fires
// every ~3-4 rounds via cooldown.
var RequiredHoldKeys = []string{
	"clinch_hold",
	"ground_hold_generic",
	"guard_hold",
	"turtle_hold",
	"backstanding_hold",
}

// RequiredStrikingApexKeys: Mount-only.
var RequiredStrikingApexKeys = []string{
	"mount_strike_flavor",
}

// RequiredGradientKeys: 4 boundary-direction keys.
var RequiredGradientKeys = []string{
	"upper_boundary_down", // Controlling → Neutral
	"upper_boundary_up",   // Neutral → Controlling
	"lower_boundary_down", // Neutral → Controlled
	"lower_boundary_up",   // Controlled → Neutral
}

// ValidateCompleteness checks that every required key is present
// AND each triad has at least MinTemplatesPerSpeaker templates per
// speaker variant. StrikingApex keys need MinStrikingApexTemplates
// in the single-speaker list. Returns a slice of all violations
// (caller decides whether to fail-loud at boot or just log).
func ValidateCompleteness(lib *Library) []error {
	var errs []error

	check := func(category string, keys []string, m map[string]TemplateTriad) {
		for _, key := range keys {
			triad, ok := m[key]
			if !ok {
				errs = append(errs, fmt.Errorf("%s: missing key %q", category, key))
				continue
			}
			if len(triad.Controller) < MinTemplatesPerSpeaker {
				errs = append(errs, fmt.Errorf("%s.%s.controller: %d templates, need >= %d",
					category, key, len(triad.Controller), MinTemplatesPerSpeaker))
			}
			if len(triad.Controlled) < MinTemplatesPerSpeaker {
				errs = append(errs, fmt.Errorf("%s.%s.controlled: %d templates, need >= %d",
					category, key, len(triad.Controlled), MinTemplatesPerSpeaker))
			}
			if len(triad.Observers) < MinTemplatesPerSpeaker {
				errs = append(errs, fmt.Errorf("%s.%s.observers: %d templates, need >= %d",
					category, key, len(triad.Observers), MinTemplatesPerSpeaker))
			}
		}
	}

	check("advancements", RequiredAdvancementKeys, lib.Advancements)
	check("degradations", RequiredDegradationKeys, lib.Degradations)
	check("reversals", RequiredReversalKeys, lib.Reversals)
	check("escapes", RequiredEscapeKeys, lib.Escapes)
	check("holds", RequiredHoldKeys, lib.Holds)

	for _, key := range RequiredStrikingApexKeys {
		templates, ok := lib.StrikingApex[key]
		if !ok {
			errs = append(errs, fmt.Errorf("striking_apex: missing key %q", key))
			continue
		}
		if len(templates) < MinStrikingApexTemplates {
			errs = append(errs, fmt.Errorf("striking_apex.%s: %d templates, need >= %d",
				key, len(templates), MinStrikingApexTemplates))
		}
	}

	for _, key := range RequiredGradientKeys {
		triad, ok := lib.Gradients[key]
		if !ok {
			errs = append(errs, fmt.Errorf("gradients: missing key %q", key))
			continue
		}
		if len(triad.Self) < MinTemplatesPerSpeaker {
			errs = append(errs, fmt.Errorf("gradients.%s.self: %d templates, need >= %d",
				key, len(triad.Self), MinTemplatesPerSpeaker))
		}
		if len(triad.Partner) < MinTemplatesPerSpeaker {
			errs = append(errs, fmt.Errorf("gradients.%s.partner: %d templates, need >= %d",
				key, len(triad.Partner), MinTemplatesPerSpeaker))
		}
		if len(triad.Observers) < MinTemplatesPerSpeaker {
			errs = append(errs, fmt.Errorf("gradients.%s.observers: %d templates, need >= %d",
				key, len(triad.Observers), MinTemplatesPerSpeaker))
		}
	}

	return errs
}
