package justice

import (
	"strconv"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/bounties"
	"github.com/GoMudEngine/GoMud/internal/characters"
)

// ---------------------------------------------------------------------------
// Pure-helper tests (Step 1 — must fail before arrest.go exists)
// ---------------------------------------------------------------------------

func TestCurrentFine(t *testing.T) {
	if g := currentFine(100, 0, 5); g != 100 {
		t.Errorf("served=0 got %d want 100", g)
	}
	if g := currentFine(100, 10, 5); g != 50 {
		t.Errorf("served=10 got %d want 50", g)
	}
	if g := currentFine(100, 30, 5); g != 0 {
		t.Errorf("served past sentence got %d want 0 (floored)", g)
	}
}

func TestSentenceRounds(t *testing.T) {
	if r := sentenceRounds(100, 5); r != 20 {
		t.Errorf("got %d want 20", r)
	}
	if r := sentenceRounds(3, 5); r != 1 {
		t.Errorf("tiny fine should floor to 1 round, got %d", r)
	}
}

// ---------------------------------------------------------------------------
// ResolveDetention seam test (Step 4)
// ---------------------------------------------------------------------------

func TestResolveDetention_Seams(t *testing.T) {
	// Save and restore all seams.
	origDecay := aDecayFn
	origRepReset := aRepResetFn
	origResolve := aResolveCrimeFn
	origOpenBounties := aOpenBountiesFn
	origWithdraw := aWithdrawFn
	origGetRep := aGetRepFn
	origSetRep := aSetRepFn
	origMove := aMoveFn
	defer func() {
		aDecayFn = origDecay
		aRepResetFn = origRepReset
		aResolveCrimeFn = origResolve
		aOpenBountiesFn = origOpenBounties
		aWithdrawFn = origWithdraw
		aGetRepFn = origGetRep
		aSetRepFn = origSetRep
		aMoveFn = origMove
	}()

	// Build a character with a fake jail record stamped in MiscData.
	ch := &characters.Character{}
	ch.SetMiscData(keyJailUntilRound, uint64(9999))
	ch.SetMiscData(keyJailFineOriginal, 100)
	ch.SetMiscData(keyJailDecayPerRound, 5)
	ch.SetMiscData(keyJailFaction, "thornwall_guards")
	ch.SetMiscData(keyJailCellRoom, 5105)
	// Two crime ids: 7 and 12.
	ch.SetMiscData(keyJailCrimeIds, "7,12")

	// Wire seams.
	aDecayFn = func() int { return 5 }
	aRepResetFn = func() int { return -10 }

	var resolvedFaction string
	var resolvedIds []int
	aResolveCrimeFn = func(factionId string, crimeId int, resolvedBy string) {
		resolvedFaction = factionId
		resolvedIds = append(resolvedIds, crimeId)
	}

	// One matching bounty from thornwall_guards, one from another faction.
	aOpenBountiesFn = func(userId int) []*bounties.Bounty {
		return []*bounties.Bounty{
			{Id: 42, Issuer: bounties.Issuer{Type: bounties.IssuerFaction, Id: "thornwall_guards"}},
			{Id: 99, Issuer: bounties.Issuer{Type: bounties.IssuerFaction, Id: "other_faction"}},
		}
	}

	var withdrawnIds []int
	aWithdrawFn = func(bountyId int) {
		withdrawnIds = append(withdrawnIds, bountyId)
	}

	// Rep below floor — should be reset.
	aGetRepFn = func(factionId string, userId int) int { return -50 }
	var setRepFaction string
	var setRepValue int
	aSetRepFn = func(factionId string, userId int, rep int) {
		setRepFaction = factionId
		setRepValue = rep
	}

	var movedTo int
	aMoveFn = func(userId int, toRoomId int, isSpawn ...bool) error {
		movedTo = toRoomId
		return nil
	}

	userId := 42
	ok := ResolveDetention(ch, userId)

	if !ok {
		t.Fatal("ResolveDetention returned false; expected true")
	}

	// Both crimes should be resolved.
	if resolvedFaction != "thornwall_guards" {
		t.Errorf("resolvedFaction=%q want thornwall_guards", resolvedFaction)
	}
	if len(resolvedIds) != 2 || resolvedIds[0] != 7 || resolvedIds[1] != 12 {
		t.Errorf("resolvedIds=%v want [7 12]", resolvedIds)
	}

	// Only the thornwall_guards bounty should be withdrawn.
	if len(withdrawnIds) != 1 || withdrawnIds[0] != 42 {
		t.Errorf("withdrawnIds=%v want [42]", withdrawnIds)
	}

	// Rep should be reset to floor (-10) because current (-50) < floor (-10).
	if setRepFaction != "thornwall_guards" {
		t.Errorf("setRepFaction=%q want thornwall_guards", setRepFaction)
	}
	if setRepValue != -10 {
		t.Errorf("setRepValue=%d want -10", setRepValue)
	}

	// Player should be moved to barracks room 473.
	if movedTo != 473 {
		t.Errorf("movedTo=%d want 473", movedTo)
	}

	// Jail MiscData keys should be cleared.
	info, jailed := JailInfo(ch)
	if jailed {
		t.Errorf("JailInfo should return false after ResolveDetention, got %+v", info)
	}

	// Sanity: crime id parsing helper round-trip.
	ids := parseCrimeIds("7,12")
	if len(ids) != 2 || ids[0] != 7 || ids[1] != 12 {
		t.Errorf("parseCrimeIds round-trip failed: %v", ids)
	}
	_ = strconv.Itoa(0) // suppress import-unused if parseCrimeIds is unexported
	_ = strings.Join(nil, "")
}

// TestResolveDetention_NoJailRecord verifies the guard: returns false when no
// record is stamped.
func TestResolveDetention_NoJailRecord(t *testing.T) {
	ch := &characters.Character{}
	if ResolveDetention(ch, 1) {
		t.Error("expected false for un-jailed character")
	}
}

// TestResolveDetention_RepAboveFloor verifies that rep is NOT reset when
// it is already at or above the floor.
func TestResolveDetention_RepAboveFloor(t *testing.T) {
	origDecay := aDecayFn
	origRepReset := aRepResetFn
	origResolve := aResolveCrimeFn
	origOpenBounties := aOpenBountiesFn
	origWithdraw := aWithdrawFn
	origGetRep := aGetRepFn
	origSetRep := aSetRepFn
	origMove := aMoveFn
	defer func() {
		aDecayFn = origDecay
		aRepResetFn = origRepReset
		aResolveCrimeFn = origResolve
		aOpenBountiesFn = origOpenBounties
		aWithdrawFn = origWithdraw
		aGetRepFn = origGetRep
		aSetRepFn = origSetRep
		aMoveFn = origMove
	}()

	ch := &characters.Character{}
	ch.SetMiscData(keyJailUntilRound, uint64(9999))
	ch.SetMiscData(keyJailFineOriginal, 50)
	ch.SetMiscData(keyJailDecayPerRound, 5)
	ch.SetMiscData(keyJailFaction, "thornwall_guards")
	ch.SetMiscData(keyJailCellRoom, 5105)
	ch.SetMiscData(keyJailCrimeIds, "")

	aDecayFn = func() int { return 5 }
	aRepResetFn = func() int { return -10 }
	aResolveCrimeFn = func(string, int, string) {}
	aOpenBountiesFn = func(int) []*bounties.Bounty { return nil }
	aWithdrawFn = func(int) {}
	// Rep is above floor — should NOT trigger SetRep.
	aGetRepFn = func(string, int) int { return 0 }
	setRepCalled := false
	aSetRepFn = func(string, int, int) { setRepCalled = true }
	aMoveFn = func(int, int, ...bool) error { return nil }

	ResolveDetention(ch, 1)

	if setRepCalled {
		t.Error("SetRep should not be called when rep >= floor")
	}
}
