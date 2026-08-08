package playtestprofiles

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

func TestSanitizeTemplateRejectsPassword(t *testing.T) {
	u := &users.UserRecord{
		Role:      users.RoleUser,
		Password:  "nope",
		Character: &characters.Character{Name: "X"},
	}
	err := SanitizeTemplate("fresh", u)
	require.Error(t, err)
	require.Contains(t, err.Error(), "password")
}

func TestSanitizeTemplateRejectsInbox(t *testing.T) {
	u := &users.UserRecord{
		Role:      users.RoleUser,
		Inbox:     users.Inbox{{Message: "hi"}},
		Character: &characters.Character{Name: "X"},
	}
	err := SanitizeTemplate("fresh", u)
	require.Error(t, err)
	require.Contains(t, err.Error(), "inbox")
}

func TestSanitizeTemplateRejectsAdminRoleOnNonAdmin(t *testing.T) {
	u := &users.UserRecord{
		Role:      users.RoleAdmin,
		Character: &characters.Character{Name: "X"},
	}
	err := SanitizeTemplate("fresh", u)
	require.Error(t, err)
	require.Contains(t, err.Error(), "role admin")
}

func TestSanitizeTemplateRequiresCharacter(t *testing.T) {
	u := &users.UserRecord{Role: users.RoleUser}
	err := SanitizeTemplate("fresh", u)
	require.Error(t, err)
	require.Contains(t, err.Error(), "character")
}

func TestLoadTemplateFreshFixture(t *testing.T) {
	u, err := LoadTemplate("testdata", "fresh")
	require.NoError(t, err)
	require.Equal(t, "Fresh Recruit", u.Character.Name)
	require.Equal(t, users.RoleUser, u.Role)
}

func TestLoadTemplateRejectsPasswordFixture(t *testing.T) {
	dir := t.TempDir()
	data, err := os.ReadFile(filepath.Join("testdata", "bad-password.yaml"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "fresh.yaml"), data, 0o644))
	_, err = LoadTemplate(dir, "fresh")
	require.Error(t, err)
	require.Contains(t, err.Error(), "password")
}
