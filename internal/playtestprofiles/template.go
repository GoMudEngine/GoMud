package playtestprofiles

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/users"
	"gopkg.in/yaml.v3"
)

func rejectArchiveLoadPath(path string) error {
	norm := strings.ToLower(filepath.ToSlash(path))
	if strings.Contains(norm, "/_archive/") || strings.Contains(norm, "/prod-users/") {
		return fmt.Errorf("playtestprofiles: refusing archive/prod-users path %q", path)
	}
	return nil
}

// LoadTemplate reads <profilesDir>/<id>.yaml, unmarshals into a UserRecord,
// and runs SanitizeTemplate.
func LoadTemplate(profilesDir, profileID string) (*users.UserRecord, error) {
	if !IsKnownTemplateID(profileID) {
		return nil, fmt.Errorf("playtestprofiles: unknown template id %q", profileID)
	}
	if profilesDir == "" {
		return nil, fmt.Errorf("playtestprofiles: profiles dir is required")
	}
	path := filepath.Join(profilesDir, profileID+".yaml")
	if err := rejectArchiveLoadPath(path); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("playtestprofiles: read template %s: %w", path, err)
	}

	u := &users.UserRecord{}
	if err := yaml.Unmarshal(data, u); err != nil {
		return nil, fmt.Errorf("playtestprofiles: parse template %s: %w", path, err)
	}
	if err := SanitizeTemplate(profileID, u); err != nil {
		return nil, err
	}
	return u, nil
}
