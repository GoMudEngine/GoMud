package characters

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

// Go test binaries run with CWD set to their own package directory, so a
// test that reads the real config must chdir to the repo root first.
// Precedent: internal/web/auth_test.go.
//
// Unlike that precedent (and internal/dialogue/save_test.go), this package
// also has tests (progression_test.go) that depend on the code-default
// Balance values never being overwritten by a load of the real
// _datafiles/config.yaml — which sets several progression fields
// (BaseProgressionChance, SkillProgressionMultipliers, ...) to different
// tuned values. configs.ReloadConfig() mutates package-global state that
// persists for the rest of the test binary, so merely reloading "the real
// config" again in cleanup (as the precedent does) is not enough; we must
// snapshot and restore whatever config state existed before this test ran.
func withRepoRoot(t *testing.T) {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root := filepath.Join(cwd, "..", "..")
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir to repo root: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	// SetConfigForTest snapshots the current config and self-registers the
	// restore; ReloadConfig() then mutates it to the real config.yaml for
	// the duration of this test only.
	configs.SetConfigForTest(t, configs.GetConfig())

	if err := configs.ReloadConfig(); err != nil {
		t.Fatalf("reload config: %v", err)
	}
}

// poolsFor builds a character with the six stats set and returns its
// derived pool maxima.
func poolsFor(t *testing.T, str, dex, per, vit, wil, cha int) (int, int, int) {
	t.Helper()
	c := New()
	c.Stats.Strength.Base = str
	c.Stats.Dexterity.Base = dex
	c.Stats.Perception.Base = per
	c.Stats.Vitality.Base = vit
	c.Stats.Willpower.Base = wil
	c.Stats.Charisma.Base = cha
	c.RecalculateStats()
	return c.HealthMax.Value, c.StaminaMax.Value, c.ConvictionMax.Value
}

// TestPoolMaxFormula pins the intended pool design:
//
//	HealthMax     = 5 + Vitality*3 + Strength*1
//	StaminaMax    = 5 + Vitality*3 + Willpower*1
//	ConvictionMax = 5 + Charisma*3 + Willpower*1
//
// Expected values are exact — every term is integer arithmetic on raw
// stats, so there is no reason to accept a tolerance here.
func TestPoolMaxFormula(t *testing.T) {
	withRepoRoot(t)

	cases := []struct {
		name                   string
		str, dex, per          int
		vit, wil, cha          int
		wantHP, wantSP, wantCP int
	}{
		// 5 + vit*3 + str, 5 + vit*3 + wil, 5 + cha*3 + wil
		{"median", 105, 105, 105, 105, 105, 105, 425, 425, 425},
		{"Duard", 104, 87, 175, 134, 195, 111, 511, 602, 533},
		{"meirok", 136, 110, 152, 117, 148, 123, 492, 504, 522},
		{"fyttyn_frozen", 128, 123, 182, 280, 243, 140, 973, 1088, 668},
	}

	for _, tc := range cases {
		hp, sp, cp := poolsFor(t, tc.str, tc.dex, tc.per, tc.vit, tc.wil, tc.cha)
		if hp != tc.wantHP {
			t.Errorf("%s HealthMax = %d, want %d", tc.name, hp, tc.wantHP)
		}
		if sp != tc.wantSP {
			t.Errorf("%s StaminaMax = %d, want %d", tc.name, sp, tc.wantSP)
		}
		if cp != tc.wantCP {
			t.Errorf("%s ConvictionMax = %d, want %d", tc.name, cp, tc.wantCP)
		}
	}
}

// Strength must not contribute to stamina, and willpower must be the
// secondary (not equal) contributor to conviction. Both were true before
// 2026-08-02 and are corrected here, so pin them against regression.
func TestPoolMaxStatRoles(t *testing.T) {
	withRepoRoot(t)

	baseHP, baseSP, baseCP := poolsFor(t, 100, 100, 100, 100, 100, 100)

	// +10 Strength: HealthMax +10, StaminaMax unchanged.
	hp, sp, _ := poolsFor(t, 110, 100, 100, 100, 100, 100)
	if hp != baseHP+10 {
		t.Errorf("+10 Str: HealthMax %d, want %d", hp, baseHP+10)
	}
	if sp != baseSP {
		t.Errorf("+10 Str: StaminaMax %d, want %d (strength must not feed stamina)", sp, baseSP)
	}

	// +10 Charisma: ConvictionMax +30 (primary), HealthMax unchanged.
	hp2, _, cp := poolsFor(t, 100, 100, 100, 100, 100, 110)
	if cp != baseCP+30 {
		t.Errorf("+10 Cha: ConvictionMax %d, want %d (charisma is primary)", cp, baseCP+30)
	}
	if hp2 != baseHP {
		t.Errorf("+10 Cha: HealthMax %d, want %d", hp2, baseHP)
	}
}
