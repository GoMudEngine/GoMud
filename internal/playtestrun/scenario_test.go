package playtestrun

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func setupScenarioPlaytestRoot(t *testing.T) (playtestRoot string, leaderGoals, joinerGoals string) {
	t.Helper()
	playtestRoot = t.TempDir()
	goalsDir := filepath.Join(playtestRoot, "goals", "scenarios", "party-formation")
	require.NoError(t, os.MkdirAll(goalsDir, 0o755))

	leaderGoals = filepath.Join(goalsDir, "leader.yaml")
	joinerGoals = filepath.Join(goalsDir, "joiner.yaml")
	require.NoError(t, os.WriteFile(leaderGoals, []byte(`
ephemeral:
  profile: early
  start_room: 5200
  budgets:
    wall_clock: 10m
goals:
  - invite
`), 0o644))
	require.NoError(t, os.WriteFile(joinerGoals, []byte(`
ephemeral:
  profile: fresh
  start_room: 5200
goals:
  - accept
`), 0o644))
	return playtestRoot, leaderGoals, joinerGoals
}

func writeScenario(t *testing.T, playtestRoot, body string) string {
	t.Helper()
	path := filepath.Join(playtestRoot, "scenarios", "party-formation.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	return path
}

func TestParseScenario_HappyPartyFormationShape(t *testing.T) {
	root, _, _ := setupScenarioPlaytestRoot(t)
	path := writeScenario(t, root, `
name: party-formation
mode: party
summary: form a party
on_actor_stop: continue
budgets:
  wall_clock: 30m
requires:
  max_connections: 20
roster:
  - id: leader
    personality: feature-tester
    goals: goals/scenarios/party-formation/leader.yaml
  - id: joiner
    personality: feel-tester
    goals: goals/scenarios/party-formation/joiner.yaml
group_goals:
  - id: party-formed
    do: invite and accept
`)
	got, err := ParseScenario(path, root, ScenarioParseOpts{MaxAIConnections: 20})
	require.NoError(t, err)
	require.Equal(t, "party-formation", got.Name)
	require.Equal(t, "party", got.Mode)
	require.Equal(t, onActorStopContinue, got.OnActorStop)
	require.Equal(t, 30*time.Minute, got.WallClock)
	require.Len(t, got.Roster, 2)
	require.Equal(t, "leader", got.Roster[0].ID)
	require.Equal(t, "feature-tester", got.Roster[0].Personality)
	require.Equal(t, "early", got.Roster[0].Binding.Profile)
	require.Equal(t, 10*time.Minute, got.Roster[0].Binding.WallClock) // parsed, ignored by supervisor later
	require.Equal(t, "joiner", got.Roster[1].ID)
	require.Equal(t, "fresh", got.Roster[1].Binding.Profile)
	require.NotEmpty(t, got.GroupGoals)
}

func TestParseScenario_DefaultOnActorStopAndWallClock(t *testing.T) {
	root, _, _ := setupScenarioPlaytestRoot(t)
	path := writeScenario(t, root, `
name: party-formation
roster:
  - id: leader
    personality: feature-tester
    goals: goals/scenarios/party-formation/leader.yaml
  - id: joiner
    personality: feel-tester
    goals: goals/scenarios/party-formation/joiner.yaml
`)
	got, err := ParseScenario(path, root, ScenarioParseOpts{MaxAIConnections: 20})
	require.NoError(t, err)
	require.Equal(t, onActorStopContinue, got.OnActorStop)
	require.Equal(t, defaultScenarioWallClock, got.WallClock)
}

func TestParseScenario_UnknownScenarioKey(t *testing.T) {
	root, _, _ := setupScenarioPlaytestRoot(t)
	path := writeScenario(t, root, `
name: x
target: local
roster:
  - id: leader
    personality: feature-tester
    goals: goals/scenarios/party-formation/leader.yaml
`)
	_, err := ParseScenario(path, root, ScenarioParseOpts{MaxAIConnections: 20})
	require.Error(t, err)
}

func TestParseScenario_UnknownRosterKey(t *testing.T) {
	root, _, _ := setupScenarioPlaytestRoot(t)
	path := writeScenario(t, root, `
name: x
roster:
  - id: leader
    personality: feature-tester
    role: feature-tester
    goals: goals/scenarios/party-formation/leader.yaml
`)
	_, err := ParseScenario(path, root, ScenarioParseOpts{MaxAIConnections: 20})
	require.Error(t, err)
}

func TestParseScenario_DuplicateID(t *testing.T) {
	root, _, _ := setupScenarioPlaytestRoot(t)
	path := writeScenario(t, root, `
name: x
roster:
  - id: leader
    personality: feature-tester
    goals: goals/scenarios/party-formation/leader.yaml
  - id: leader
    personality: feel-tester
    goals: goals/scenarios/party-formation/joiner.yaml
`)
	_, err := ParseScenario(path, root, ScenarioParseOpts{MaxAIConnections: 20})
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate")
}

func TestParseScenario_EmptyRoster(t *testing.T) {
	root, _, _ := setupScenarioPlaytestRoot(t)
	path := writeScenario(t, root, `
name: x
roster: []
`)
	_, err := ParseScenario(path, root, ScenarioParseOpts{MaxAIConnections: 20})
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-empty")
}

func TestParseScenario_InvalidRosterID(t *testing.T) {
	root, _, _ := setupScenarioPlaytestRoot(t)
	path := writeScenario(t, root, `
name: x
roster:
  - id: "bad id!"
    personality: feature-tester
    goals: goals/scenarios/party-formation/leader.yaml
`)
	_, err := ParseScenario(path, root, ScenarioParseOpts{MaxAIConnections: 20})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid id")
}

func TestParseScenario_AdminProfileRejected(t *testing.T) {
	root, _, _ := setupScenarioPlaytestRoot(t)
	adminGoals := filepath.Join(root, "goals", "admin.yaml")
	require.NoError(t, os.WriteFile(adminGoals, []byte(`
ephemeral:
  profile: admin
  start_room: 1
`), 0o644))
	path := writeScenario(t, root, `
name: x
roster:
  - id: adminish
    personality: feature-tester
    goals: goals/admin.yaml
`)
	_, err := ParseScenario(path, root, ScenarioParseOpts{MaxAIConnections: 20})
	require.Error(t, err)
	require.Contains(t, err.Error(), "admin")
}

func TestParseScenario_LegacyInlineGoalsRejected(t *testing.T) {
	root, _, _ := setupScenarioPlaytestRoot(t)
	path := writeScenario(t, root, `
name: x
roster:
  - id: leader
    personality: feature-tester
    goals:
      - id: invite
        do: party invite
`)
	_, err := ParseScenario(path, root, ScenarioParseOpts{MaxAIConnections: 20})
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "inline")
}

func TestParseScenario_MissingGoalsFile(t *testing.T) {
	root, _, _ := setupScenarioPlaytestRoot(t)
	path := writeScenario(t, root, `
name: x
roster:
  - id: leader
    personality: feature-tester
    goals: goals/scenarios/missing.yaml
`)
	_, err := ParseScenario(path, root, ScenarioParseOpts{MaxAIConnections: 20})
	require.Error(t, err)
}

func TestParseScenario_UnknownOnActorStop(t *testing.T) {
	root, _, _ := setupScenarioPlaytestRoot(t)
	path := writeScenario(t, root, `
name: x
on_actor_stop: explode
roster:
  - id: leader
    personality: feature-tester
    goals: goals/scenarios/party-formation/leader.yaml
`)
	_, err := ParseScenario(path, root, ScenarioParseOpts{MaxAIConnections: 20})
	require.Error(t, err)
	require.Contains(t, err.Error(), "on_actor_stop")
}

func TestParseScenario_RequiresPvPRejected(t *testing.T) {
	root, _, _ := setupScenarioPlaytestRoot(t)
	path := writeScenario(t, root, `
name: x
requires:
  pvp: true
roster:
  - id: leader
    personality: feature-tester
    goals: goals/scenarios/party-formation/leader.yaml
`)
	_, err := ParseScenario(path, root, ScenarioParseOpts{MaxAIConnections: 20})
	require.Error(t, err)
	require.Contains(t, err.Error(), "pvp")
}

func TestParseScenario_RequiresNonPvPOpaqueOK(t *testing.T) {
	root, _, _ := setupScenarioPlaytestRoot(t)
	path := writeScenario(t, root, `
name: x
requires:
  foo: bar
  max_connections: 20
roster:
  - id: leader
    personality: feature-tester
    goals: goals/scenarios/party-formation/leader.yaml
`)
	got, err := ParseScenario(path, root, ScenarioParseOpts{MaxAIConnections: 20})
	require.NoError(t, err)
	require.Equal(t, "bar", got.Requires["foo"])
}

func TestParseScenario_RosterExceedsMaxAI(t *testing.T) {
	root, _, _ := setupScenarioPlaytestRoot(t)
	path := writeScenario(t, root, `
name: x
roster:
  - id: a
    personality: feature-tester
    goals: goals/scenarios/party-formation/leader.yaml
  - id: b
    personality: feel-tester
    goals: goals/scenarios/party-formation/joiner.yaml
`)
	_, err := ParseScenario(path, root, ScenarioParseOpts{MaxAIConnections: 1})
	require.Error(t, err)
	require.Contains(t, err.Error(), "MaxAIConnections")

	got, err := ParseScenario(path, root, ScenarioParseOpts{MaxAIConnections: 1, Force: true})
	require.NoError(t, err)
	require.Len(t, got.Roster, 2)
}
