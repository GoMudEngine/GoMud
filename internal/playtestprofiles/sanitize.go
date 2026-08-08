package playtestprofiles

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/users"
)

// SanitizeTemplate checks that a loaded profile template is safe to commit and
// materialize.
func SanitizeTemplate(profileID string, u *users.UserRecord) error {
	if u == nil {
		return fmt.Errorf("playtestprofiles: template %q is nil", profileID)
	}
	if !IsKnownTemplateID(profileID) {
		return fmt.Errorf("playtestprofiles: unknown template id %q", profileID)
	}
	if strings.TrimSpace(u.Password) != "" {
		return fmt.Errorf("playtestprofiles: template %q must not contain a password", profileID)
	}
	if len(u.Inbox) > 0 {
		return fmt.Errorf("playtestprofiles: template %q must not contain inbox mail", profileID)
	}
	if strings.TrimSpace(u.EmailAddress) != "" {
		return fmt.Errorf("playtestprofiles: template %q must not contain emailaddress", profileID)
	}
	if len(u.Macros) > 0 || len(u.Aliases) > 0 || len(u.Ticks) > 0 || len(u.Triggers) > 0 {
		return fmt.Errorf("playtestprofiles: template %q must not contain macros/aliases/ticks/triggers", profileID)
	}
	role := strings.TrimSpace(u.Role)
	if role == "" {
		role = users.RoleUser
		u.Role = role
	}
	switch role {
	case users.RoleUser:
		if profileID == "admin" {
			return fmt.Errorf("playtestprofiles: template %q must use role admin", profileID)
		}
	case users.RoleAdmin:
		if profileID != "admin" {
			return fmt.Errorf("playtestprofiles: template %q must not use role admin", profileID)
		}
	default:
		return fmt.Errorf("playtestprofiles: template %q has unsupported role %q", profileID, role)
	}
	if u.Character == nil {
		return fmt.Errorf("playtestprofiles: template %q missing character block", profileID)
	}
	if strings.TrimSpace(u.Character.Name) == "" {
		return fmt.Errorf("playtestprofiles: template %q character.name is required", profileID)
	}
	// Design-reference prod identities must never ship as template usernames
	// or character names (archive is offline authoring only).
	if err := ForbiddenIdentity(u.Username); err != nil {
		return fmt.Errorf("playtestprofiles: template %q must not use prod identity: %w", profileID, err)
	}
	if err := ForbiddenIdentity(u.Character.Name); err != nil {
		return fmt.Errorf("playtestprofiles: template %q must not use prod identity: %w", profileID, err)
	}
	return nil
}
