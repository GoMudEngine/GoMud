package playtestprofiles

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRepoTemplatesSanitize(t *testing.T) {
	dir := filepath.Join("..", "..", "tools", "playtest", "profiles")
	if _, err := os.Stat(dir); err != nil {
		t.Skip("repo profiles directory not present")
	}
	for _, id := range KnownTemplateIDs {
		t.Run(id, func(t *testing.T) {
			u, err := LoadTemplate(dir, id)
			require.NoError(t, err)
			require.NotNil(t, u.Character)
			require.Empty(t, u.Password)
			if id == "admin" {
				require.Equal(t, "admin", u.Role)
			} else {
				require.Equal(t, "user", u.Role)
			}
		})
	}
}
