package playtestprofiles

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestForbiddenIdentityExactStem(t *testing.T) {
	require.Error(t, ForbiddenIdentity("Meirok"))
	require.Error(t, ForbiddenIdentity("meirok"))
	require.Error(t, ForbiddenIdentity("MEIROK"))
}

func TestForbiddenIdentityPtPrefix(t *testing.T) {
	require.Error(t, ForbiddenIdentity("pt_meirok"))
	require.Error(t, ForbiddenIdentity("pt-Meirok"))
	require.Error(t, ForbiddenIdentity("ptMeirok"))
}

func TestForbiddenIdentityDigitsAroundStem(t *testing.T) {
	require.Error(t, ForbiddenIdentity("meirok12"))
	require.Error(t, ForbiddenIdentity("99meirok"))
	require.Error(t, ForbiddenIdentity("1meirok2"))
}

func TestForbiddenIdentityLevenshtein1(t *testing.T) {
	// Meirok len=6 ≥ 4 → distance-1 banned.
	require.Error(t, ForbiddenIdentity("Meirox"))  // substitute
	require.Error(t, ForbiddenIdentity("Meirokk")) // insert
	require.Error(t, ForbiddenIdentity("Merok"))   // delete
}

func TestForbiddenIdentitySubstring(t *testing.T) {
	// stem len ≥ 5 → contiguous substring match bans.
	require.Error(t, ForbiddenIdentity("xxmeirokyy"))
	require.Error(t, ForbiddenIdentity("CaptainMeirok"))
}

func TestForbiddenIdentityAllowsUnrelated(t *testing.T) {
	require.NoError(t, ForbiddenIdentity("FreshRecruit"))
	require.NoError(t, ForbiddenIdentity("pt_early_ab12cd"))
	require.NoError(t, ForbiddenIdentity("Zorblax"))
	require.NoError(t, ForbiddenIdentity(""))
	require.NoError(t, ForbiddenIdentity("   "))
}
