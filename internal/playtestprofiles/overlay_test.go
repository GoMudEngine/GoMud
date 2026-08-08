package playtestprofiles

import (
	"fmt"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

func testWorld() WorldChecks {
	return WorldChecks{
		RoomExists: func(id int) bool { return id == 100 || id == 462 },
		SpellOK:    func(id string) bool { return id == "heal" || id == "identify" },
		ItemOK:     func(id int) bool { return id == 1 || id == 2 },
		FlagOK: func(key, value string) error {
			if key == "11-branch" && (value == "rhett" || value == "sylara") {
				return nil
			}
			return fmt.Errorf("undeclared quest flag %q", key)
		},
	}
}

func baseUser() *users.UserRecord {
	return &users.UserRecord{
		Role: users.RoleUser,
		Character: &characters.Character{
			Name:      "Tester",
			SpellBook: map[string]int{},
			Skills:    map[string]int{},
		},
	}
}

func TestApplyOverlaysGrantsSpellSkillGoldRoom(t *testing.T) {
	u := baseUser()
	gold := 50
	err := ApplyOverlays(u, 100, Overlays{
		GrantSpells: map[string]int{"heal": 3},
		GrantSkills: map[string]int{"salvage": 10},
		SetGold:     &gold,
	}, testWorld())
	require.NoError(t, err)
	require.Equal(t, 100, u.Character.RoomId)
	require.Equal(t, 3, u.Character.SpellBook["heal"])
	require.Equal(t, 10, u.Character.Skills["salvage"])
	require.Equal(t, 50, u.Character.Gold)
}

func TestApplyOverlaysRejectsUnknownSpell(t *testing.T) {
	u := baseUser()
	err := ApplyOverlays(u, 100, Overlays{
		GrantSpells: map[string]int{"no-such-spell": 1},
	}, testWorld())
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown spell")
}

func TestApplyOverlaysRejectsBadRoom(t *testing.T) {
	u := baseUser()
	err := ApplyOverlays(u, 99999, Overlays{}, testWorld())
	require.Error(t, err)
	require.Contains(t, err.Error(), "start_room")
}

func TestApplyOverlaysSetsQuestFlag(t *testing.T) {
	u := baseUser()
	err := ApplyOverlays(u, 100, Overlays{
		SetQuestFlags: map[string]string{"11-branch": "rhett"},
	}, testWorld())
	require.NoError(t, err)
	require.Equal(t, "rhett", u.Character.GetQuestFlag("11-branch"))
}
