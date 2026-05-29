package justice

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/bounties"
	"github.com/GoMudEngine/GoMud/internal/crimes"
	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/opinions"
)

func TestBountyGold_MurderDominant(t *testing.T) {
	// powerBase 100, murderMult 2.0, rep -50 (repMult 1.0) -> max=2.0 -> 200
	if g := bountyGold(100, true, -50, 2.0, 2.0); g != 200 {
		t.Errorf("got %d, want 200", g)
	}
}

func TestBountyGold_RepDominant(t *testing.T) {
	// powerBase 100, not murder (crimeMult 1.0), rep -100 -> repMult 2.0 -> 200
	if g := bountyGold(100, false, -100, 2.0, 2.0); g != 200 {
		t.Errorf("got %d, want 200", g)
	}
}

func TestBountyGold_NeitherDominant_BaseFloor(t *testing.T) {
	// not murder, rep -50 (Hostile boundary, repMult 1.0) -> 100
	if g := bountyGold(100, false, -50, 2.0, 2.0); g != 100 {
		t.Errorf("got %d, want 100", g)
	}
}

func TestShouldDeclare(t *testing.T) {
	if !shouldDeclare(crimes.KindMurder, opinions.TierNeutral, false) {
		t.Error("identified murder should declare")
	}
	if !shouldDeclare(crimes.KindAssault, opinions.TierHostile, false) {
		t.Error("Hostile rep should declare")
	}
	if shouldDeclare(crimes.KindAssault, opinions.TierCold, false) {
		t.Error("Cold rep, no murder -> no declare")
	}
	if shouldDeclare(crimes.KindMurder, opinions.TierHostile, true) {
		t.Error("already-open bounty -> no declare (dedup)")
	}
}

func TestMaybeDeclareBounty_MurderDeclares(t *testing.T) {
	o1, o2, o3, o4, o5, o6, o7, o8 :=
		bTierFn, bRepFn, bDefaultGoldFn, bDeclareFn, bNowFn, bMurderMultFn, bRepMultMaxFn, bExistingFn
	defer func() {
		bTierFn, bRepFn, bDefaultGoldFn, bDeclareFn, bNowFn, bMurderMultFn, bRepMultMaxFn, bExistingFn =
			o1, o2, o3, o4, o5, o6, o7, o8
	}()
	bTierFn = func(string, int) opinions.Tier { return opinions.TierNeutral }
	bRepFn = func(string, int) int { return 0 }
	bDefaultGoldFn = func(knowledge.Subject) int { return 100 }
	bMurderMultFn = func() float64 { return 2.0 }
	bRepMultMaxFn = func() float64 { return 2.0 }
	bNowFn = func() uint64 { return 1000 }
	bExistingFn = func(string, int) bool { return false }
	var gotGold int
	var gotIssuer bounties.Issuer
	var gotExpiry uint64
	bDeclareFn = func(issuer bounties.Issuer, _ knowledge.Subject, _ bounties.Condition, expiry uint64, opts bounties.DeclareOpts) (int, error) {
		gotIssuer, gotGold, gotExpiry = issuer, opts.GoldOverride, expiry
		return 1, nil
	}
	MaybeDeclareBounty("thornwall_guards", 99017, crimes.KindMurder)
	if gotGold != 200 || gotIssuer.Id != "thornwall_guards" {
		t.Fatalf("declared gold=%d issuer=%q; want 200 / thornwall_guards", gotGold, gotIssuer.Id)
	}
	_ = gotExpiry // (expiry = now+expiryRounds; exercised, not asserted)
}

func TestMaybeDeclareBounty_DedupSkips(t *testing.T) {
	o1, o2 := bTierFn, bExistingFn
	defer func() { bTierFn, bExistingFn = o1, o2 }()
	bTierFn = func(string, int) opinions.Tier { return opinions.TierHostile }
	bExistingFn = func(string, int) bool { return true } // already open
	called := false
	od := bDeclareFn
	defer func() { bDeclareFn = od }()
	bDeclareFn = func(bounties.Issuer, knowledge.Subject, bounties.Condition, uint64, bounties.DeclareOpts) (int, error) {
		called = true
		return 0, nil
	}
	MaybeDeclareBounty("thornwall_guards", 99018, crimes.KindMurder)
	if called {
		t.Error("declare should be skipped when a faction bounty is already open")
	}
}
