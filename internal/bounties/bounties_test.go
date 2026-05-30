package bounties

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
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
		{600, 6},   // 600 / 100 = 6
		{100, 1},   // 100 / 100 = 1
		{50, 1},    // floor of 1 (50 / 100 = 0 -> 1)
		{0, 1},     // floor of 1
		{1000, 10}, // 1000 / 100 = 10
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

func TestWithdraw(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil; statpoolForTest = nil }()
	roundForTest = func() uint64 { return 100 }
	statpoolForTest = func(_ knowledge.Subject) int { return 600 }

	id, _ := Declare(FactionIssuer("thornwall_guards"),
		knowledge.PlayerSubject(17), ConditionKill, 1000, DeclareOpts{})

	Withdraw(id)
	b := Get(id)
	if b.Status != StatusWithdrawn {
		t.Errorf("expected withdrawn, got %s", b.Status)
	}

	// Idempotent on already-withdrawn.
	Withdraw(id)
	b = Get(id)
	if b.Status != StatusWithdrawn {
		t.Errorf("status should still be withdrawn: %s", b.Status)
	}
}

func TestPruneExpired(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil; statpoolForTest = nil }()
	statpoolForTest = func(_ knowledge.Subject) int { return 600 }

	roundForTest = func() uint64 { return 100 }
	idA, _ := Declare(FactionIssuer("thornwall_guards"),
		knowledge.PlayerSubject(17), ConditionKill, 200, DeclareOpts{}) // expires at 200
	idB, _ := Declare(FactionIssuer("thornwall_guards"),
		knowledge.PlayerSubject(18), ConditionKill, 0, DeclareOpts{}) // never expires
	idC, _ := Declare(FactionIssuer("thornwall_guards"),
		knowledge.PlayerSubject(19), ConditionKill, 5000, DeclareOpts{}) // far future

	roundForTest = func() uint64 { return 300 }
	count := PruneExpired()
	if count != 1 {
		t.Errorf("expected 1 pruned (idA), got %d", count)
	}
	if Get(idA).Status != StatusExpired {
		t.Errorf("idA should be expired")
	}
	if Get(idB).Status != StatusOpen {
		t.Errorf("idB should still be open (never expires)")
	}
	if Get(idC).Status != StatusOpen {
		t.Errorf("idC should still be open (future expiry)")
	}
}

func TestTryClaim_HappyPath(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil; statpoolForTest = nil }()
	roundForTest = func() uint64 { return 100 }
	statpoolForTest = func(_ knowledge.Subject) int { return 600 }

	id, _ := Declare(FactionIssuer("thornwall_guards"),
		knowledge.MobSubject(101), ConditionKill, 1000, DeclareOpts{})

	roundForTest = func() uint64 { return 200 }
	b, ok := TryClaim(id, knowledge.PlayerSubject(17))
	if !ok {
		t.Fatalf("TryClaim should have succeeded")
	}
	if b.Status != StatusClaimed {
		t.Errorf("status should be claimed, got %s", b.Status)
	}
	if b.ClaimedBy != knowledge.PlayerSubject(17) {
		t.Errorf("ClaimedBy mismatch: %+v", b.ClaimedBy)
	}
	if b.ClaimedRound != 200 {
		t.Errorf("ClaimedRound mismatch: %d", b.ClaimedRound)
	}
}

func TestTryClaim_AlreadyClaimedReturnsFalse(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil; statpoolForTest = nil }()
	roundForTest = func() uint64 { return 100 }
	statpoolForTest = func(_ knowledge.Subject) int { return 600 }

	id, _ := Declare(FactionIssuer("thornwall_guards"),
		knowledge.MobSubject(101), ConditionKill, 1000, DeclareOpts{})

	TryClaim(id, knowledge.PlayerSubject(17))
	b, ok := TryClaim(id, knowledge.PlayerSubject(99))
	if ok {
		t.Errorf("second TryClaim should have returned false")
	}
	if b != nil {
		t.Errorf("second TryClaim should have returned nil bounty")
	}
}

func TestAllRows_IncludesNonOpen(t *testing.T) {
	bounitiesPath := filepath.Join(string(configs.GetFilePathsConfig().DataFiles), "bounties.yaml")
	os.Remove(bounitiesPath)
	resetCache()
	defer func() { roundForTest = nil; statpoolForTest = nil }()
	roundForTest = func() uint64 { return 100 }
	statpoolForTest = func(_ knowledge.Subject) int { return 600 }

	idA, _ := Declare(FactionIssuer("thornwall_guards"),
		knowledge.MobSubject(101), ConditionKill, 1000, DeclareOpts{})
	idB, _ := Declare(FactionIssuer("thornwall_guards"),
		knowledge.MobSubject(102), ConditionKill, 1000, DeclareOpts{})
	Withdraw(idB)

	rows := AllRows()
	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}
	_ = idA
}

func TestDefaultGoldFor_MatchesComputeDefault(t *testing.T) {
	restore := statpoolForTest
	statpoolForTest = func(knowledge.Subject) int { return 600 }
	defer func() { statpoolForTest = restore }()
	got := DefaultGoldFor(knowledge.PlayerSubject(17))
	want := computeDefaultGold(600)
	if got != want {
		t.Errorf("DefaultGoldFor = %d, want %d", got, want)
	}
}

func TestReadAPI(t *testing.T) {
	// Clean the bounties file to start fresh (other tests accumulate entries)
	bounitiesPath := filepath.Join(string(configs.GetFilePathsConfig().DataFiles), "bounties.yaml")
	os.Remove(bounitiesPath)

	resetCache()
	defer func() { roundForTest = nil; statpoolForTest = nil }()
	roundForTest = func() uint64 { return 100 }
	statpoolForTest = func(_ knowledge.Subject) int { return 600 }

	idA, _ := Declare(FactionIssuer("thornwall_guards"),
		knowledge.PlayerSubject(17), ConditionKill, 1000, DeclareOpts{})
	idB, _ := Declare(FactionIssuer("thornwall_citizens"),
		knowledge.PlayerSubject(17), ConditionKill, 1000, DeclareOpts{})
	idC, _ := Declare(FactionIssuer("thornwall_guards"),
		knowledge.MobSubject(101), ConditionKill, 1000, DeclareOpts{})
	Withdraw(idC)

	// AllOpen returns A and B (C is withdrawn).
	open := AllOpen()
	if len(open) != 2 {
		t.Errorf("AllOpen: got %d, want 2", len(open))
	}

	// OpenForTarget(player 17) returns A and B.
	tgt := OpenForTarget(knowledge.PlayerSubject(17))
	if len(tgt) != 2 {
		t.Errorf("OpenForTarget: got %d, want 2", len(tgt))
	}

	// OpenForIssuer(thornwall_guards) returns only A (C withdrawn).
	iss := OpenForIssuer(FactionIssuer("thornwall_guards"))
	if len(iss) != 1 || iss[0].Id != idA {
		t.Errorf("OpenForIssuer: got %v", iss)
	}

	// OpenAgainstPlayer convenience.
	pl := OpenAgainstPlayer(17)
	if len(pl) != 2 {
		t.Errorf("OpenAgainstPlayer: got %d, want 2", len(pl))
	}

	// AllForTarget with includeNonOpen=true sees the withdrawn one.
	all := AllForTarget(knowledge.MobSubject(101), true)
	if len(all) != 1 || all[0].Id != idC {
		t.Errorf("AllForTarget(include): got %v", all)
	}
	open101 := AllForTarget(knowledge.MobSubject(101), false)
	if len(open101) != 0 {
		t.Errorf("AllForTarget(open only): got %v", open101)
	}

	_ = idB
}
