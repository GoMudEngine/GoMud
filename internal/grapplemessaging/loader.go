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

// Library is the parsed in-memory template store. Keys for each map
// follow spec §7.1 conventions:
//   - Advancements:  "<source>_to_<target>" (e.g. "clinch_to_mount")
//   - Degradations:  "<source>_to_<target>" (e.g. "mount_to_sidecontrol")
//   - Reversals:     "<source>_reverse" for realism-exception sources;
//                    "generic_reverse" as fallback.
//   - Escapes:       "generic_escape" (only key for now)
//   - Holds:         "<context>_hold" (e.g. "clinch_hold",
//                    "ground_hold_generic")
//   - StrikingApex:  Single-speaker (no triad); "mount_strike_flavor"
//                    is the only key currently.
type Library struct {
	Advancements map[string]TemplateTriad `yaml:"advancements"`
	Degradations map[string]TemplateTriad `yaml:"degradations"`
	Reversals    map[string]TemplateTriad `yaml:"reversals"`
	Escapes      map[string]TemplateTriad `yaml:"escapes"`
	Holds        map[string]TemplateTriad `yaml:"holds"`
	StrikingApex map[string][]string      `yaml:"striking_apex"`
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
	return lib, nil
}
