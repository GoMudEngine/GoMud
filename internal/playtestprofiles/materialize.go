package playtestprofiles

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/users"
	"gopkg.in/yaml.v3"
)

// MaterializeOptions configures a materialization pass.
type MaterializeOptions struct {
	ProfilesDir  string
	World        WorldChecks
	CredsOutPath string
	RunID        string
}

// Materialize loads templates, applies overlays, generates credentials, and
// persists offline users. Returns plaintext creds for the run artifact.
func Materialize(m *Manifest, opts MaterializeOptions) ([]PlayerCreds, error) {
	if m == nil {
		return nil, fmt.Errorf("playtestprofiles: nil manifest")
	}
	if err := validateManifest(m); err != nil {
		return nil, err
	}
	if opts.ProfilesDir == "" {
		return nil, fmt.Errorf("playtestprofiles: ProfilesDir is required")
	}
	world := opts.World
	if world.RoomExists == nil && world.SpellOK == nil && world.ItemOK == nil {
		world = DefaultWorldChecks()
	}

	out := make([]PlayerCreds, 0, len(m.Entries))
	for i, entry := range m.Entries {
		u, err := LoadTemplate(opts.ProfilesDir, entry.Profile)
		if err != nil {
			return nil, fmt.Errorf("playtestprofiles: entries[%d]: %w", i, err)
		}
		u, err = cloneUser(u)
		if err != nil {
			return nil, err
		}
		if err := ApplyOverlays(u, entry.StartRoom, entry.Overlays, world); err != nil {
			return nil, fmt.Errorf("playtestprofiles: entries[%d]: %w", i, err)
		}
		if err := ForbiddenIdentity(u.Character.Name); err != nil {
			return nil, fmt.Errorf("playtestprofiles: entries[%d]: %w", i, err)
		}
		if err := u.Character.Validate(true); err != nil {
			return nil, fmt.Errorf("playtestprofiles: entries[%d]: character validate: %w", i, err)
		}
		username, password, err := GenerateCredentials(u, entry.Profile)
		if err != nil {
			return nil, fmt.Errorf("playtestprofiles: entries[%d]: %w", i, err)
		}
		if err := PersistOfflineUser(u); err != nil {
			return nil, fmt.Errorf("playtestprofiles: entries[%d]: %w", i, err)
		}
		out = append(out, PlayerCreds{
			Profile:  entry.Profile,
			Username: username,
			Password: password,
			UserID:   u.UserId,
			RoomID:   u.Character.RoomId,
		})
	}

	if opts.CredsOutPath != "" {
		if err := writeCredsFile(opts.CredsOutPath, opts.RunID, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// MaterializeFromConfig runs materialization when Playtest.ProfilesManifest is set.
// Empty/absent path → no-op (nil, nil).
func MaterializeFromConfig() ([]PlayerCreds, error) {
	cfg := configs.GetPlaytestConfig()
	manifestPath := string(cfg.ProfilesManifest)
	if manifestPath == "" {
		return nil, nil
	}
	m, err := LoadManifest(manifestPath)
	if err != nil {
		return nil, err
	}
	credsPath := filepath.Join(filepath.Dir(manifestPath), "creds.json")
	return Materialize(m, MaterializeOptions{
		ProfilesDir:  string(cfg.ProfilesDir),
		World:        DefaultWorldChecks(),
		CredsOutPath: credsPath,
	})
}

func writeCredsFile(path, runID string, players []PlayerCreds) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("playtestprofiles: create creds dir: %w", err)
	}
	payload := CredsFile{RunID: runID, Players: players}
	buf, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("playtestprofiles: marshal creds: %w", err)
	}
	if err := os.WriteFile(path, buf, 0o600); err != nil {
		return fmt.Errorf("playtestprofiles: write creds: %w", err)
	}
	return nil
}

func cloneUser(u *users.UserRecord) (*users.UserRecord, error) {
	data, err := yaml.Marshal(u)
	if err != nil {
		return nil, fmt.Errorf("playtestprofiles: clone marshal: %w", err)
	}
	out := &users.UserRecord{}
	if err := yaml.Unmarshal(data, out); err != nil {
		return nil, fmt.Errorf("playtestprofiles: clone unmarshal: %w", err)
	}
	return out, nil
}
