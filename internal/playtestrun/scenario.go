package playtestrun

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultScenarioWallClock = 45 * time.Minute
	defaultMaxAIConnections  = 20
	onActorStopContinue      = "continue"
	onActorStopAbort         = "abort"
)

var rosterIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ScenarioParseOpts configures ParseScenario fail-closed checks.
type ScenarioParseOpts struct {
	// Force bypasses only the MaxAIConnections roster-size check.
	Force bool
	// MaxAIConnections overrides the checkout config probe when > 0.
	MaxAIConnections int
	// Checkout is used to read Network.MaxAIConnections when MaxAIConnections is 0.
	Checkout string
}

// ScenarioFile is a validated multi-agent scenario.
type ScenarioFile struct {
	Name        string
	Mode        string
	Summary     string
	OnActorStop string
	WallClock   time.Duration
	Requires    map[string]any
	Roster      []ScenarioActor
	GroupGoals  []yaml.Node
	Path        string // absolute scenario path
}

// ScenarioActor is one roster entry with resolved goals + binding.
type ScenarioActor struct {
	ID          string
	Personality string
	GoalsPath   string
	Binding     EphemeralBinding
}

type scenarioYAML struct {
	Name        string             `yaml:"name"`
	Mode        string             `yaml:"mode"`
	Summary     string             `yaml:"summary"`
	OnActorStop string             `yaml:"on_actor_stop"`
	Budgets     *scenarioBudgets   `yaml:"budgets"`
	Requires    map[string]any     `yaml:"requires"`
	Roster      []scenarioActorRaw `yaml:"roster"`
	GroupGoals  []yaml.Node        `yaml:"group_goals"`
}

type scenarioBudgets struct {
	WallClock string `yaml:"wall_clock"`
}

// scenarioActorRaw keeps Goals as a Node so we can reject inline lists with a
// clear error before KnownFields decoding of peers.
type scenarioActorRaw struct {
	ID          string    `yaml:"id"`
	Personality string    `yaml:"personality"`
	Goals       yaml.Node `yaml:"goals"`
}

// ParseScenario loads and validates a multi-agent scenario YAML.
// playtestRoot is typically <checkout>/tools/playtest and resolves relative goals paths.
func ParseScenario(scenarioPath, playtestRoot string, opts ScenarioParseOpts) (ScenarioFile, error) {
	scenarioPath = filepath.Clean(strings.TrimSpace(scenarioPath))
	if scenarioPath == "" {
		return ScenarioFile{}, fmt.Errorf("playtestrun: scenario path is required")
	}
	data, err := os.ReadFile(scenarioPath)
	if err != nil {
		return ScenarioFile{}, fmt.Errorf("playtestrun: read scenario %s: %w", scenarioPath, err)
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var raw scenarioYAML
	if err := dec.Decode(&raw); err != nil {
		return ScenarioFile{}, fmt.Errorf("playtestrun: parse scenario: %w", err)
	}

	name := strings.TrimSpace(raw.Name)
	if name == "" {
		return ScenarioFile{}, fmt.Errorf("playtestrun: scenario: name is required")
	}

	onStop := strings.TrimSpace(raw.OnActorStop)
	if onStop == "" {
		onStop = onActorStopContinue
	}
	switch onStop {
	case onActorStopContinue, onActorStopAbort:
	default:
		return ScenarioFile{}, fmt.Errorf("playtestrun: scenario: unknown on_actor_stop %q", onStop)
	}

	if err := rejectPvPRequires(raw.Requires); err != nil {
		return ScenarioFile{}, err
	}

	wallClock, err := parseScenarioWallClock(raw.Budgets)
	if err != nil {
		return ScenarioFile{}, err
	}

	if len(raw.Roster) == 0 {
		return ScenarioFile{}, fmt.Errorf("playtestrun: scenario: roster must be non-empty")
	}

	maxAI := opts.MaxAIConnections
	if maxAI <= 0 {
		maxAI = readMaxAIConnections(opts.Checkout)
	}
	if len(raw.Roster) > maxAI && !opts.Force {
		return ScenarioFile{}, fmt.Errorf("playtestrun: scenario: roster size %d exceeds MaxAIConnections %d (use --force to bypass)", len(raw.Roster), maxAI)
	}

	seen := make(map[string]struct{}, len(raw.Roster))
	actors := make([]ScenarioActor, 0, len(raw.Roster))
	for i, entry := range raw.Roster {
		actor, err := parseScenarioActor(entry, i, playtestRoot)
		if err != nil {
			return ScenarioFile{}, err
		}
		if _, dup := seen[actor.ID]; dup {
			return ScenarioFile{}, fmt.Errorf("playtestrun: scenario: duplicate roster id %q", actor.ID)
		}
		seen[actor.ID] = struct{}{}
		if !actor.Binding.CreationFlow && actor.Binding.Profile == "admin" {
			return ScenarioFile{}, fmt.Errorf("playtestrun: scenario: actor %q: admin profile is not allowed in multi-agent scenarios", actor.ID)
		}
		actors = append(actors, actor)
	}

	absScenario, err := filepath.Abs(scenarioPath)
	if err != nil {
		absScenario = scenarioPath
	}

	return ScenarioFile{
		Name:        name,
		Mode:        strings.TrimSpace(raw.Mode),
		Summary:     strings.TrimSpace(raw.Summary),
		OnActorStop: onStop,
		WallClock:   wallClock,
		Requires:    raw.Requires,
		Roster:      actors,
		GroupGoals:  raw.GroupGoals,
		Path:        absScenario,
	}, nil
}

func parseScenarioActor(entry scenarioActorRaw, index int, playtestRoot string) (ScenarioActor, error) {
	id := strings.TrimSpace(entry.ID)
	if id == "" {
		return ScenarioActor{}, fmt.Errorf("playtestrun: scenario: roster[%d]: id is required", index)
	}
	if !rosterIDPattern.MatchString(id) {
		return ScenarioActor{}, fmt.Errorf("playtestrun: scenario: roster[%d]: invalid id %q (want [a-zA-Z0-9_-]+)", index, id)
	}
	personality := strings.TrimSpace(entry.Personality)
	if personality == "" {
		return ScenarioActor{}, fmt.Errorf("playtestrun: scenario: roster[%d]: personality is required", index)
	}
	if entry.Goals.Kind == 0 {
		return ScenarioActor{}, fmt.Errorf("playtestrun: scenario: roster[%d]: goals path is required", index)
	}
	if entry.Goals.Kind == yaml.SequenceNode {
		return ScenarioActor{}, fmt.Errorf("playtestrun: scenario: roster[%d]: inline goals lists are not allowed; use a goals file path", index)
	}
	if entry.Goals.Kind != yaml.ScalarNode {
		return ScenarioActor{}, fmt.Errorf("playtestrun: scenario: roster[%d]: goals must be a file path string", index)
	}
	goalsRel := strings.TrimSpace(entry.Goals.Value)
	if goalsRel == "" {
		return ScenarioActor{}, fmt.Errorf("playtestrun: scenario: roster[%d]: goals path is required", index)
	}

	goalsPath := goalsRel
	if !filepath.IsAbs(goalsPath) {
		if strings.TrimSpace(playtestRoot) == "" {
			return ScenarioActor{}, fmt.Errorf("playtestrun: scenario: playtest root is required to resolve relative goals paths")
		}
		goalsPath = filepath.Join(playtestRoot, goalsRel)
	}
	goalsPath = filepath.Clean(goalsPath)
	if _, err := os.Stat(goalsPath); err != nil {
		return ScenarioActor{}, fmt.Errorf("playtestrun: scenario: roster[%d]: goals file %s: %w", index, goalsPath, err)
	}

	binding, err := ParseGoalsEphemeral(goalsPath)
	if err != nil {
		return ScenarioActor{}, fmt.Errorf("playtestrun: scenario: roster[%d] (%s): %w", index, id, err)
	}

	absGoals, err := filepath.Abs(goalsPath)
	if err != nil {
		absGoals = goalsPath
	}
	return ScenarioActor{
		ID:          id,
		Personality: personality,
		GoalsPath:   absGoals,
		Binding:     binding,
	}, nil
}

func parseScenarioWallClock(budgets *scenarioBudgets) (time.Duration, error) {
	if budgets == nil || strings.TrimSpace(budgets.WallClock) == "" {
		return defaultScenarioWallClock, nil
	}
	d, err := time.ParseDuration(strings.TrimSpace(budgets.WallClock))
	if err != nil {
		return 0, fmt.Errorf("playtestrun: scenario: budgets.wall_clock: %w", err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("playtestrun: scenario: budgets.wall_clock must be > 0")
	}
	return d, nil
}

func rejectPvPRequires(requires map[string]any) error {
	if requires == nil {
		return nil
	}
	for k, v := range requires {
		key := strings.ToLower(strings.TrimSpace(k))
		if key == "pvp" {
			if isTruthyRequire(v) {
				return fmt.Errorf("playtestrun: scenario: requires.pvp is not supported yet (PvP ephemeral overrides deferred)")
			}
		}
	}
	return nil
}

func isTruthyRequire(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		return s != "" && s != "false" && s != "0" && s != "no" && s != "off"
	case int:
		return t != 0
	case int64:
		return t != 0
	case float64:
		return t != 0
	default:
		return v != nil
	}
}

func readMaxAIConnections(checkout string) int {
	if strings.TrimSpace(checkout) == "" {
		return defaultMaxAIConnections
	}
	path := filepath.Join(checkout, "_datafiles", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultMaxAIConnections
	}
	var doc struct {
		Network struct {
			MaxAIConnections int `yaml:"MaxAIConnections"`
		} `yaml:"Network"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return defaultMaxAIConnections
	}
	if doc.Network.MaxAIConnections < 1 {
		return defaultMaxAIConnections
	}
	return doc.Network.MaxAIConnections
}
