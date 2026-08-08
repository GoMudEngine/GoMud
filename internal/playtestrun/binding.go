package playtestrun

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/GoMudEngine/GoMud/internal/playtestenv"
	"github.com/GoMudEngine/GoMud/internal/playtestprofiles"
)

const defaultWallClock = 30 * time.Minute

// EphemeralBinding is the goals-file contract for a single local playtestrun.
type EphemeralBinding struct {
	Profile           string // empty if creation flow
	StartRoom         int
	Overlays          playtestenv.ProfileOverlays
	CreationFlow      bool
	CreationRationale string
	WallClock         time.Duration // default 30m
}

type ephemeralYAML struct {
	Profile           *string                     `yaml:"profile"`
	StartRoom         *int                        `yaml:"start_room"`
	Overlays          *playtestenv.ProfileOverlays `yaml:"overlays"`
	CreationFlow      bool                        `yaml:"creation_flow"`
	CreationRationale string                      `yaml:"creation_rationale"`
	Budgets           *ephemeralBudgetsYAML       `yaml:"budgets"`
}

type ephemeralBudgetsYAML struct {
	WallClock string `yaml:"wall_clock"`
}

// ParseGoalsEphemeral reads a goals YAML file and returns the validated
// ephemeral binding. Unknown keys under ephemeral: fail closed (KnownFields).
// Legacy session.* soft hints elsewhere in the file are ignored.
func ParseGoalsEphemeral(goalsPath string) (EphemeralBinding, error) {
	data, err := os.ReadFile(goalsPath)
	if err != nil {
		return EphemeralBinding{}, fmt.Errorf("playtestrun: read goals %s: %w", goalsPath, err)
	}

	var root map[string]yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return EphemeralBinding{}, fmt.Errorf("playtestrun: parse goals %s: %w", goalsPath, err)
	}
	ephemNode, ok := root["ephemeral"]
	if !ok || ephemNode.Kind == 0 {
		return EphemeralBinding{}, fmt.Errorf("playtestrun: goals %s: missing ephemeral block", goalsPath)
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&ephemNode); err != nil {
		_ = enc.Close()
		return EphemeralBinding{}, fmt.Errorf("playtestrun: encode ephemeral: %w", err)
	}
	if err := enc.Close(); err != nil {
		return EphemeralBinding{}, fmt.Errorf("playtestrun: encode ephemeral: %w", err)
	}

	dec := yaml.NewDecoder(&buf)
	dec.KnownFields(true)
	var raw ephemeralYAML
	if err := dec.Decode(&raw); err != nil {
		return EphemeralBinding{}, fmt.Errorf("playtestrun: parse ephemeral: %w", err)
	}

	return validateEphemeral(raw)
}

func validateEphemeral(raw ephemeralYAML) (EphemeralBinding, error) {
	wallClock, err := parseWallClock(raw.Budgets)
	if err != nil {
		return EphemeralBinding{}, err
	}

	hasProfile := raw.Profile != nil && strings.TrimSpace(*raw.Profile) != ""
	hasCreation := raw.CreationFlow

	if hasProfile && hasCreation {
		return EphemeralBinding{}, fmt.Errorf("playtestrun: ephemeral: profile and creation_flow are mutually exclusive")
	}
	if !hasProfile && !hasCreation {
		return EphemeralBinding{}, fmt.Errorf("playtestrun: ephemeral: require profile+start_room or creation_flow")
	}

	if hasCreation {
		if raw.Profile != nil || raw.StartRoom != nil || raw.Overlays != nil {
			return EphemeralBinding{}, fmt.Errorf("playtestrun: ephemeral: creation_flow forbids profile, start_room, and overlays")
		}
		rationale := strings.TrimSpace(raw.CreationRationale)
		if rationale == "" {
			return EphemeralBinding{}, fmt.Errorf("playtestrun: ephemeral: creation_rationale is required when creation_flow is true")
		}
		return EphemeralBinding{
			CreationFlow:      true,
			CreationRationale: rationale,
			WallClock:         wallClock,
		}, nil
	}

	profile := strings.TrimSpace(*raw.Profile)
	if !playtestprofiles.IsKnownTemplateID(profile) {
		return EphemeralBinding{}, fmt.Errorf("playtestrun: ephemeral: unknown profile %q", profile)
	}
	if raw.StartRoom == nil {
		return EphemeralBinding{}, fmt.Errorf("playtestrun: ephemeral: start_room is required with profile")
	}
	if *raw.StartRoom <= 0 {
		return EphemeralBinding{}, fmt.Errorf("playtestrun: ephemeral: start_room must be > 0")
	}

	out := EphemeralBinding{
		Profile:   profile,
		StartRoom: *raw.StartRoom,
		WallClock: wallClock,
	}
	if raw.Overlays != nil {
		out.Overlays = *raw.Overlays
	}
	return out, nil
}

func parseWallClock(budgets *ephemeralBudgetsYAML) (time.Duration, error) {
	if budgets == nil || strings.TrimSpace(budgets.WallClock) == "" {
		return defaultWallClock, nil
	}
	d, err := time.ParseDuration(strings.TrimSpace(budgets.WallClock))
	if err != nil {
		return 0, fmt.Errorf("playtestrun: ephemeral: budgets.wall_clock: %w", err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("playtestrun: ephemeral: budgets.wall_clock must be > 0")
	}
	return d, nil
}
