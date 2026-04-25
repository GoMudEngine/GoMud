package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// setupCompanionValidationConfig seeds NameSizeMin/Max so ValidateActorName
// doesn't reject everything due to zero-value defaults.
func setupCompanionValidationConfig(t *testing.T) {
	t.Helper()
	_ = configs.AddOverlayOverrides(map[string]any{
		"Validation.NameSizeMin": 1,
		"Validation.NameSizeMax": 20,
	})
}

// seedMobNameForTest registers a single mob spec with the given name for the
// duration of a test, returning a cleanup function.
func seedMobNameForTest(t *testing.T, name string) func() {
	t.Helper()
	m := &mobs.Mob{
		MobId: 9901,
		Character: characters.Character{
			Name: name,
		},
	}
	return mobs.SeedMobsForTest(map[int]*mobs.Mob{9901: m}, nil)
}

func TestValidateCompanionName_BlocksMobNames(t *testing.T) {
	setupCompanionValidationConfig(t)
	cleanup := seedMobNameForTest(t, "Goblin")
	defer cleanup()

	err := validateCompanionName("Goblin")
	if err == nil {
		t.Fatal("expected mob-name collision rejection, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "in use") {
		t.Fatalf("expected error to mention 'in use', got %v", err)
	}
}

func TestValidateCompanionName_AllowsCleanName(t *testing.T) {
	setupCompanionValidationConfig(t)
	cleanup := seedMobNameForTest(t, "Goblin")
	defer cleanup()

	err := validateCompanionName("Fluffington")
	if err != nil {
		t.Fatalf("expected clean name to pass, got: %v", err)
	}
}

func TestValidateCompanionName_RejectsShortName(t *testing.T) {
	err := validateCompanionName("X")
	if err == nil {
		t.Fatal("expected short-name rejection, got nil")
	}
}

func TestValidateCompanionName_RejectsNonLetters(t *testing.T) {
	err := validateCompanionName("Bob123")
	if err == nil {
		t.Fatal("expected non-letter rejection, got nil")
	}
}
