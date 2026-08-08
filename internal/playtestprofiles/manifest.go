package playtestprofiles

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseManifest parses and validates a profiles manifest. Unknown YAML keys
// are rejected. Empty entries is legal (no-op materialization).
func ParseManifest(data []byte) (*Manifest, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	var m Manifest
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("playtestprofiles: parse manifest: %w", err)
	}
	if err := validateManifest(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

// LoadManifest reads a manifest file from disk.
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("playtestprofiles: read manifest %s: %w", path, err)
	}
	return ParseManifest(data)
}

func validateManifest(m *Manifest) error {
	if m == nil {
		return fmt.Errorf("playtestprofiles: nil manifest")
	}
	for i, e := range m.Entries {
		id := strings.TrimSpace(e.Profile)
		if id == "" {
			return fmt.Errorf("playtestprofiles: entries[%d]: profile is required", i)
		}
		if !IsKnownTemplateID(id) {
			return fmt.Errorf("playtestprofiles: entries[%d]: unknown profile %q", i, id)
		}
		if e.StartRoom <= 0 {
			return fmt.Errorf("playtestprofiles: entries[%d]: start_room must be > 0", i)
		}
		m.Entries[i].Profile = id
	}
	return nil
}

// IsKnownTemplateID reports whether id is one of the six tracked templates.
func IsKnownTemplateID(id string) bool {
	for _, known := range KnownTemplateIDs {
		if id == known {
			return true
		}
	}
	return false
}
