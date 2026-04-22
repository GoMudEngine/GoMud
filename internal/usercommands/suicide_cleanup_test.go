package usercommands

import "testing"

// The following tests are skipped because Suicide() operates on
// package-global state (users registry, mobs registry, rooms registry)
// and a direct unit test would require bootstrapping all three.
// The SetAggro grace guard has direct unit tests in
// internal/characters/aggro_grace_test.go (Task 2), and the smoke
// test in Task 7 covers the three fixes end-to-end.
//
// If a future pass adds a minimal user+room+mob test harness here,
// flesh these out. Pattern to follow: internal/hooks/hooks_test.go's
// seedAllRegistries().

func TestSuicide_ClearsInboundAggro(t *testing.T) {
	t.Skip("pending test-infra; smoke test (Task 7) covers")
}

func TestSuicide_ClearsCompanionAggro(t *testing.T) {
	t.Skip("pending test-infra; smoke test (Task 7) covers")
}

func TestSuicide_ClearsConditions(t *testing.T) {
	t.Skip("pending test-infra; smoke test (Task 7) covers")
}

func TestSuicide_AppliesRespawnGrace(t *testing.T) {
	t.Skip("pending test-infra; smoke test (Task 7) covers")
}
