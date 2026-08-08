package playtestprofiles

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

func TestGenerateCredentialsShapeAndHash(t *testing.T) {
	_ = configs.AddOverlayOverrides(map[string]any{
		"Validation.PasswordSizeMin": 4,
		"Validation.PasswordSizeMax": 16,
		"Validation.NameSizeMin":     1,
		"Validation.NameSizeMax":     80,
	})
	u := &users.UserRecord{
		Role:      users.RoleUser,
		Character: &characters.Character{Name: "Tester"},
	}
	username, password, err := GenerateCredentials(u, "specialist-caster")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(username, "pt-specialist-caster-"))
	require.GreaterOrEqual(t, len(password), 4)
	require.LessOrEqual(t, len(password), 16)
	require.Equal(t, username, u.Username)
	require.NotEqual(t, password, u.Password)
	require.True(t, u.PasswordMatches(password))
}

func TestSanitizeProfileToken(t *testing.T) {
	require.Equal(t, "specialist-caster", sanitizeProfileToken("specialist-caster"))
	require.Equal(t, "mid", sanitizeProfileToken("mid"))
}
