package bounties

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/knowledge"
)

func TestComputeDefaultGold(t *testing.T) {
	defer func() { goldMultiplierForTest = nil; goldFloorForTest = nil }()
	goldMultiplierForTest = func() float64 { return 0.5 }
	goldFloorForTest = func() int { return 50 }

	cases := []struct {
		statpool int
		want     int
	}{
		{600, 300},  // 600 * 0.5 = 300
		{1000, 500}, // 1000 * 0.5 = 500
		{50, 50},    // floor wins (25 -> 50)
		{0, 50},     // floor wins
		{200, 100},  // 200 * 0.5 = 100
	}
	for _, c := range cases {
		if got := computeDefaultGold(c.statpool); got != c.want {
			t.Errorf("statpool=%d: got %d, want %d", c.statpool, got, c.want)
		}
	}
}

func TestComputeDefaultRep(t *testing.T) {
	cases := []struct {
		statpool int
		want     int
	}{
		{600, 6},    // 600 / 100 = 6
		{100, 1},    // 100 / 100 = 1
		{50, 1},     // floor of 1 (50 / 100 = 0 -> 1)
		{0, 1},      // floor of 1
		{1000, 10},  // 1000 / 100 = 10
	}
	for _, c := range cases {
		if got := computeDefaultRep(c.statpool); got != c.want {
			t.Errorf("statpool=%d: got %d, want %d", c.statpool, got, c.want)
		}
	}
}

func TestDeclare_DefaultRewards(t *testing.T) {
	resetCache()
	defer func() {
		roundForTest = nil
		goldMultiplierForTest = nil
		goldFloorForTest = nil
		statpoolForTest = nil
	}()
	roundForTest = func() uint64 { return 100 }
	goldMultiplierForTest = func() float64 { return 0.5 }
	goldFloorForTest = func() int { return 50 }
	// Stub the statpool lookup so the test doesn't need a real mob fixture.
	statpoolForTest = func(target knowledge.Subject) int { return 600 }

	id, err := Declare(
		FactionIssuer("thornwall_guards"),
		knowledge.PlayerSubject(17),
		ConditionKill,
		1000,
		DeclareOpts{},
	)
	if err != nil {
		t.Fatalf("Declare returned error: %v", err)
	}
	if id != 1 {
		t.Errorf("expected first id=1, got %d", id)
	}

	b := Get(id)
	if b == nil {
		t.Fatalf("Get returned nil for id %d", id)
	}
	if b.GoldReward != 300 {
		t.Errorf("default gold should be 300 (600 * 0.5), got %d", b.GoldReward)
	}
	if b.RepReward != 6 {
		t.Errorf("default rep should be 6 (600/100), got %d", b.RepReward)
	}
	if b.Status != StatusOpen {
		t.Errorf("status should be open, got %s", b.Status)
	}
	if b.DeclaredRound != 100 {
		t.Errorf("DeclaredRound mismatch: %d", b.DeclaredRound)
	}
}

func TestDeclare_Overrides(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil; statpoolForTest = nil }()
	roundForTest = func() uint64 { return 100 }
	statpoolForTest = func(_ knowledge.Subject) int { return 600 }

	id, _ := Declare(
		FactionIssuer("thornwall_guards"),
		knowledge.PlayerSubject(17),
		ConditionKill,
		1000,
		DeclareOpts{
			GoldOverride:   1500,
			RepOverride:    25,
			DeclaredReason: "High-value target",
		},
	)
	b := Get(id)
	if b.GoldReward != 1500 {
		t.Errorf("override gold not honored: %d", b.GoldReward)
	}
	if b.RepReward != 25 {
		t.Errorf("override rep not honored: %d", b.RepReward)
	}
	if b.DeclaredReason != "High-value target" {
		t.Errorf("reason not stored: %q", b.DeclaredReason)
	}
}
