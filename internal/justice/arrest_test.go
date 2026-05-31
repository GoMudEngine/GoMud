package justice

import (
	"fmt"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/bounties"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/crimes"
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
	origCrimesForFaction := aCrimesForFactionFn
	origAllies := alliesFn
	origOpenBounties := aOpenBountiesFn
	origWithdraw := aWithdrawFn
	origGetRep := aGetRepFn
	origSetRep := aSetRepFn
	origMove := aMoveFn
	defer func() {
		aDecayFn = origDecay
		aRepResetFn = origRepReset
		aResolveCrimeFn = origResolve
		aCrimesForFactionFn = origCrimesForFaction
		alliesFn = origAllies
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

	// Wire seams.
	aDecayFn = func() int { return 5 }
	aRepResetFn = func() int { return -10 }

	// The arresting faction has one ally — release must clear the player's
	// record across BOTH (the BUG-03 fix). Crime 7 is against the guards,
	// crime 12 against the allied citizens; an other-faction crime (99) and an
	// other-player crime (13) must NOT be resolved.
	alliesFn = func(faction string) []string {
		if faction == "thornwall_guards" {
			return []string{"thornwall_citizens"}
		}
		return nil
	}
	userId := 42
	aCrimesForFactionFn = func(factionId string, includeResolved bool) []*crimes.Crime {
		switch factionId {
		case "thornwall_guards":
			return []*crimes.Crime{
				{Id: 7, Perpetrator: crimes.Perpetrator{Type: crimes.PerpPlayer, Id: userId}},
				{Id: 13, Perpetrator: crimes.Perpetrator{Type: crimes.PerpPlayer, Id: 99}}, // other player
			}
		case "thornwall_citizens":
			return []*crimes.Crime{
				{Id: 12, Perpetrator: crimes.Perpetrator{Type: crimes.PerpPlayer, Id: userId}},
			}
		}
		return nil
	}

	type resolved struct {
		faction string
		id      int
	}
	var resolvedList []resolved
	aResolveCrimeFn = func(factionId string, crimeId int, resolvedBy string) {
		resolvedList = append(resolvedList, resolved{factionId, crimeId})
	}

	// Bounties: one from the arresting faction, one from the ally, one from an
	// unrelated faction (must be left alone).
	aOpenBountiesFn = func(userId int) []*bounties.Bounty {
		return []*bounties.Bounty{
			{Id: 42, Issuer: bounties.Issuer{Type: bounties.IssuerFaction, Id: "thornwall_guards"}},
			{Id: 43, Issuer: bounties.Issuer{Type: bounties.IssuerFaction, Id: "thornwall_citizens"}},
			{Id: 99, Issuer: bounties.Issuer{Type: bounties.IssuerFaction, Id: "other_faction"}},
		}
	}

	var withdrawnIds []int
	aWithdrawFn = func(bountyId int) {
		withdrawnIds = append(withdrawnIds, bountyId)
	}

	// Rep below floor — should be reset for each faction in the set.
	aGetRepFn = func(factionId string, userId int) int { return -50 }
	repSet := map[string]int{}
	aSetRepFn = func(factionId string, userId int, rep int) {
		repSet[factionId] = rep
	}

	var movedTo int
	aMoveFn = func(userId int, toRoomId int, isSpawn ...bool) error {
		movedTo = toRoomId
		return nil
	}

	ok := ResolveDetention(ch, userId)

	if !ok {
		t.Fatal("ResolveDetention returned false; expected true")
	}

	// The player's crimes against the guards (7) AND the ally citizens (12)
	// must both be resolved; the other-player crime (13) must not.
	gotResolved := map[string]bool{}
	for _, r := range resolvedList {
		gotResolved[fmt.Sprintf("%s:%d", r.faction, r.id)] = true
	}
	if !gotResolved["thornwall_guards:7"] {
		t.Errorf("guards crime 7 should be resolved; got %+v", resolvedList)
	}
	if !gotResolved["thornwall_citizens:12"] {
		t.Errorf("ally citizens crime 12 should be resolved (BUG-03); got %+v", resolvedList)
	}
	if gotResolved["thornwall_guards:13"] {
		t.Errorf("other player's crime 13 must NOT be resolved; got %+v", resolvedList)
	}

	// Both faction-set bounties withdrawn; the unrelated one left alone.
	gotWithdrawn := map[int]bool{}
	for _, id := range withdrawnIds {
		gotWithdrawn[id] = true
	}
	if !gotWithdrawn[42] || !gotWithdrawn[43] {
		t.Errorf("faction-set bounties 42,43 should be withdrawn; got %v", withdrawnIds)
	}
	if gotWithdrawn[99] {
		t.Errorf("unrelated bounty 99 must NOT be withdrawn; got %v", withdrawnIds)
	}

	// Rep reset to floor (-10) for each faction in the set (current -50 < -10).
	if repSet["thornwall_guards"] != -10 || repSet["thornwall_citizens"] != -10 {
		t.Errorf("rep should be reset to -10 for both factions; got %v", repSet)
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
}

// TestResolveDetention_NoJailRecord verifies the guard: returns false when no
// record is stamped.
func TestResolveDetention_NoJailRecord(t *testing.T) {
	ch := &characters.Character{}
	if ResolveDetention(ch, 1) {
		t.Error("expected false for un-jailed character")
	}
}

// ---------------------------------------------------------------------------
// ExecuteArrest seam tests (Task 1 — holding-cell room via faction data)
// ---------------------------------------------------------------------------

func TestExecuteArrest_NoCellForFaction_NoOps(t *testing.T) {
	origCell := cellRoomFn
	origMove := aMoveFn
	t.Cleanup(func() { cellRoomFn = origCell; aMoveFn = origMove })

	moved := false
	aMoveFn = func(int, int, ...bool) error { moved = true; return nil }
	cellRoomFn = func(faction string) int { return 0 } // faction has no jail

	ch := &characters.Character{}
	ok := ExecuteArrest(ch, 1, "stillwater_citizens", false)

	if ok {
		t.Fatalf("ExecuteArrest should return false when faction has no cell")
	}
	if moved {
		t.Fatalf("player must not be moved when faction has no cell")
	}
}

func TestExecuteArrest_UsesFactionCellRoom(t *testing.T) {
	origCell := cellRoomFn
	origMove := aMoveFn
	origDecay := aDecayFn
	origNow := bNowFn
	t.Cleanup(func() {
		cellRoomFn = origCell
		aMoveFn = origMove
		aDecayFn = origDecay
		bNowFn = origNow
	})

	var movedTo int
	aMoveFn = func(userId int, toRoomId int, isSpawn ...bool) error { movedTo = toRoomId; return nil }
	cellRoomFn = func(faction string) int { return 5106 }
	aDecayFn = func() int { return 5 }
	bNowFn = func() uint64 { return 100 }

	ch := &characters.Character{}
	ok := ExecuteArrest(ch, 1, "stillwater_guards", false)

	if !ok {
		t.Fatalf("ExecuteArrest should succeed when faction has a cell")
	}
	if movedTo != 5106 {
		t.Fatalf("player hauled to %d, want 5106", movedTo)
	}
}

// TestResolveDetention_UsesPerFactionReleaseRoom verifies that ResolveDetention
// moves the player to the per-faction release room, not the hardcoded barracks.
func TestResolveDetention_UsesPerFactionReleaseRoom(t *testing.T) {
	origDecay := aDecayFn
	origRepReset := aRepResetFn
	origResolve := aResolveCrimeFn
	origCrimesForFaction := aCrimesForFactionFn
	origAllies := alliesFn
	origOpenBounties := aOpenBountiesFn
	origWithdraw := aWithdrawFn
	origGetRep := aGetRepFn
	origSetRep := aSetRepFn
	origMove := aMoveFn
	origReleaseRoom := releaseRoomFn
	defer func() {
		aDecayFn = origDecay
		aRepResetFn = origRepReset
		aResolveCrimeFn = origResolve
		aCrimesForFactionFn = origCrimesForFaction
		alliesFn = origAllies
		aOpenBountiesFn = origOpenBounties
		aWithdrawFn = origWithdraw
		aGetRepFn = origGetRep
		aSetRepFn = origSetRep
		aMoveFn = origMove
		releaseRoomFn = origReleaseRoom
	}()

	ch := &characters.Character{}
	ch.SetMiscData(keyJailUntilRound, uint64(9999))
	ch.SetMiscData(keyJailFineOriginal, 50)
	ch.SetMiscData(keyJailDecayPerRound, 5)
	ch.SetMiscData(keyJailFaction, "stillwater_guards")
	ch.SetMiscData(keyJailCellRoom, 5106)

	aDecayFn = func() int { return 5 }
	aRepResetFn = func() int { return -10 }
	aResolveCrimeFn = func(string, int, string) {}
	aCrimesForFactionFn = func(string, bool) []*crimes.Crime { return nil }
	alliesFn = func(faction string) []string { return nil }
	aOpenBountiesFn = func(int) []*bounties.Bounty { return nil }
	aWithdrawFn = func(int) {}
	aGetRepFn = func(string, int) int { return 0 }
	aSetRepFn = func(string, int, int) {}

	// Stillwater release room is 4110.
	releaseRoomFn = func(faction string) int {
		if faction == "stillwater_guards" {
			return 4110
		}
		return 0
	}

	var movedTo int
	aMoveFn = func(userId int, toRoomId int, isSpawn ...bool) error {
		movedTo = toRoomId
		return nil
	}

	ok := ResolveDetention(ch, 42)
	if !ok {
		t.Fatal("ResolveDetention returned false; expected true")
	}
	if movedTo != 4110 {
		t.Errorf("player moved to room %d, want 4110 (Stillwater release room)", movedTo)
	}
}

// TestResolveDetention_FallsBackToBarracksWhenNoReleaseRoom verifies that when
// the faction defines no release room (returns 0), the hardcoded barracksRoomId
// is used as the fallback.
func TestResolveDetention_FallsBackToBarracksWhenNoReleaseRoom(t *testing.T) {
	origDecay := aDecayFn
	origRepReset := aRepResetFn
	origResolve := aResolveCrimeFn
	origCrimesForFaction := aCrimesForFactionFn
	origAllies := alliesFn
	origOpenBounties := aOpenBountiesFn
	origWithdraw := aWithdrawFn
	origGetRep := aGetRepFn
	origSetRep := aSetRepFn
	origMove := aMoveFn
	origReleaseRoom := releaseRoomFn
	defer func() {
		aDecayFn = origDecay
		aRepResetFn = origRepReset
		aResolveCrimeFn = origResolve
		aCrimesForFactionFn = origCrimesForFaction
		alliesFn = origAllies
		aOpenBountiesFn = origOpenBounties
		aWithdrawFn = origWithdraw
		aGetRepFn = origGetRep
		aSetRepFn = origSetRep
		aMoveFn = origMove
		releaseRoomFn = origReleaseRoom
	}()

	ch := &characters.Character{}
	ch.SetMiscData(keyJailUntilRound, uint64(9999))
	ch.SetMiscData(keyJailFineOriginal, 50)
	ch.SetMiscData(keyJailDecayPerRound, 5)
	ch.SetMiscData(keyJailFaction, "some_faction")
	ch.SetMiscData(keyJailCellRoom, 999)

	aDecayFn = func() int { return 5 }
	aRepResetFn = func() int { return -10 }
	aResolveCrimeFn = func(string, int, string) {}
	aCrimesForFactionFn = func(string, bool) []*crimes.Crime { return nil }
	alliesFn = func(faction string) []string { return nil }
	aOpenBountiesFn = func(int) []*bounties.Bounty { return nil }
	aWithdrawFn = func(int) {}
	aGetRepFn = func(string, int) int { return 0 }
	aSetRepFn = func(string, int, int) {}

	// Faction returns 0 (no release room configured).
	releaseRoomFn = func(faction string) int { return 0 }

	var movedTo int
	aMoveFn = func(userId int, toRoomId int, isSpawn ...bool) error {
		movedTo = toRoomId
		return nil
	}

	ResolveDetention(ch, 42)
	if movedTo != barracksRoomId {
		t.Errorf("player moved to room %d, want %d (barracks fallback)", movedTo, barracksRoomId)
	}
}

// ---------------------------------------------------------------------------
// ClearFactionRecord tests (Task 2)
// ---------------------------------------------------------------------------

// TestClearFactionRecord_ResolvesCrimesWithdrawsBountiesResetsRep verifies that
// ClearFactionRecord resolves crimes, withdraws bounties, and resets rep for
// the given faction and its allies, without touching unrelated factions/players.
func TestClearFactionRecord_ResolvesCrimesWithdrawsBountiesResetsRep(t *testing.T) {
	// Save and restore all seams touched by ClearFactionRecord.
	origAllies := alliesFn
	origCrimesForFaction := aCrimesForFactionFn
	origResolve := aResolveCrimeFn
	origOpenBounties := aOpenBountiesFn
	origWithdraw := aWithdrawFn
	origGetRep := aGetRepFn
	origSetRep := aSetRepFn
	origRepReset := aRepResetFn
	t.Cleanup(func() {
		alliesFn = origAllies
		aCrimesForFactionFn = origCrimesForFaction
		aResolveCrimeFn = origResolve
		aOpenBountiesFn = origOpenBounties
		aWithdrawFn = origWithdraw
		aGetRepFn = origGetRep
		aSetRepFn = origSetRep
		aRepResetFn = origRepReset
	})

	const userId = 7

	// "f" has one ally "f_ally".
	alliesFn = func(faction string) []string {
		if faction == "f" {
			return []string{"f_ally"}
		}
		return nil
	}

	// Crime 1 against faction "f" by player 7 (should be resolved).
	// Crime 2 against faction "f_ally" by player 7 (should be resolved).
	// Crime 3 against faction "f" by another player (must NOT be resolved).
	aCrimesForFactionFn = func(factionId string, includeResolved bool) []*crimes.Crime {
		switch factionId {
		case "f":
			return []*crimes.Crime{
				{Id: 1, Perpetrator: crimes.Perpetrator{Type: crimes.PerpPlayer, Id: userId}},
				{Id: 3, Perpetrator: crimes.Perpetrator{Type: crimes.PerpPlayer, Id: 99}},
			}
		case "f_ally":
			return []*crimes.Crime{
				{Id: 2, Perpetrator: crimes.Perpetrator{Type: crimes.PerpPlayer, Id: userId}},
			}
		}
		return nil
	}

	type resolvedCall struct {
		faction string
		id      int
	}
	var resolvedList []resolvedCall
	aResolveCrimeFn = func(factionId string, crimeId int, resolvedBy string) {
		resolvedList = append(resolvedList, resolvedCall{factionId, crimeId})
	}

	// Bounty 10 from faction "f", bounty 11 from "f_ally", bounty 99 from
	// unrelated faction (must be left alone).
	aOpenBountiesFn = func(uid int) []*bounties.Bounty {
		return []*bounties.Bounty{
			{Id: 10, Issuer: bounties.Issuer{Type: bounties.IssuerFaction, Id: "f"}},
			{Id: 11, Issuer: bounties.Issuer{Type: bounties.IssuerFaction, Id: "f_ally"}},
			{Id: 99, Issuer: bounties.Issuer{Type: bounties.IssuerFaction, Id: "other"}},
		}
	}

	var withdrawnIds []int
	aWithdrawFn = func(bountyId int) {
		withdrawnIds = append(withdrawnIds, bountyId)
	}

	// Rep below floor — should be reset for both factions.
	aRepResetFn = func() int { return -10 }
	aGetRepFn = func(factionId string, uid int) int { return -50 }

	repSet := map[string]int{}
	aSetRepFn = func(factionId string, uid int, rep int) {
		repSet[factionId] = rep
	}

	// Execute the function under test.
	ClearFactionRecord("f", userId, "served sentence")

	// Crime 1 (f, player 7) and crime 2 (f_ally, player 7) must be resolved.
	gotResolved := map[string]bool{}
	for _, r := range resolvedList {
		gotResolved[fmt.Sprintf("%s:%d", r.faction, r.id)] = true
	}
	if !gotResolved["f:1"] {
		t.Errorf("crime 1 against faction f should be resolved; got %+v", resolvedList)
	}
	if !gotResolved["f_ally:2"] {
		t.Errorf("crime 2 against faction f_ally should be resolved; got %+v", resolvedList)
	}
	// Crime 3 belongs to another player — must NOT be resolved.
	if gotResolved["f:3"] {
		t.Errorf("crime 3 (other player) must NOT be resolved; got %+v", resolvedList)
	}

	// Bounties 10 and 11 withdrawn; unrelated bounty 99 left alone.
	gotWithdrawn := map[int]bool{}
	for _, id := range withdrawnIds {
		gotWithdrawn[id] = true
	}
	if !gotWithdrawn[10] || !gotWithdrawn[11] {
		t.Errorf("faction-set bounties 10,11 should be withdrawn; got %v", withdrawnIds)
	}
	if gotWithdrawn[99] {
		t.Errorf("unrelated bounty 99 must NOT be withdrawn; got %v", withdrawnIds)
	}

	// Rep reset to floor (-10) for both factions (current -50 < -10).
	if repSet["f"] != -10 {
		t.Errorf("rep for f should be reset to -10; got %v", repSet)
	}
	if repSet["f_ally"] != -10 {
		t.Errorf("rep for f_ally should be reset to -10; got %v", repSet)
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
