package justice

import (
	"fmt"
	"strings"
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

// ---------------------------------------------------------------------------
// ExecuteArrest instanced-cell tests (Task 3)
// ---------------------------------------------------------------------------

func TestExecuteArrest_UsesInstancedCell(t *testing.T) {
	origCell := cellRoomFn
	origMove := aMoveFn
	origCreate := aCreateCellFn
	origDesc := aSetCellDescFn
	origDecay := aDecayFn
	origNow := bNowFn
	defer func() {
		cellRoomFn = origCell
		aMoveFn = origMove
		aCreateCellFn = origCreate
		aSetCellDescFn = origDesc
		aDecayFn = origDecay
		bNowFn = origNow
	}()

	cellRoomFn = func(string) int { return 473 }
	aDecayFn = func() int { return 5 }
	bNowFn = func() uint64 { return 100 }

	var movedTo int
	aMoveFn = func(userId int, roomId int, isSpawn ...bool) error { movedTo = roomId; return nil }

	var createdFor int
	aCreateCellFn = func(prisonerUserId, releaseRoomId int) (int, bool) {
		createdFor = prisonerUserId
		return 60001, true
	}

	descSet := ""
	aSetCellDescFn = func(roomId int, desc string) { descSet = desc }

	ch := &characters.Character{}
	ok := ExecuteArrest(ch, 42, "thornwall_guards", false)
	if !ok {
		t.Fatalf("arrest should succeed")
	}
	if createdFor != 42 {
		t.Fatalf("expected cell created for 42, got %d", createdFor)
	}
	rec, jailed := JailInfo(ch)
	if !jailed || rec.InstanceId != 60001 {
		t.Fatalf("expected InstanceId 60001, got %+v", rec)
	}
	if movedTo != 60001 {
		t.Fatalf("expected move to 60001, got %d", movedTo)
	}
	if descSet == "" {
		t.Fatalf("expected faction-flavored cell description set")
	}
	if !strings.Contains(descSet, "thornwall_guards") {
		t.Fatalf("expected cell description to contain faction name %q, got: %s", "thornwall_guards", descSet)
	}
}

// ---------------------------------------------------------------------------
// ResolveDetention instanced-cell teardown tests (Task 4)
// ---------------------------------------------------------------------------

func TestResolveDetention_TearsDownInstance(t *testing.T) {
	origMove, origTeardown := aMoveFn, aTeardownCellFn
	origResolve, origCrimes := aResolveCrimeFn, aCrimesForFactionFn
	origAllies := alliesFn
	origReleaseRoom := releaseRoomFn
	origRepReset := aRepResetFn
	origGetRep := aGetRepFn
	origSetRep := aSetRepFn
	origOpenBounties := aOpenBountiesFn
	origWithdraw := aWithdrawFn
	defer func() {
		aMoveFn, aTeardownCellFn = origMove, origTeardown
		aResolveCrimeFn, aCrimesForFactionFn = origResolve, origCrimes
		alliesFn = origAllies
		releaseRoomFn = origReleaseRoom
		aRepResetFn = origRepReset
		aGetRepFn = origGetRep
		aSetRepFn = origSetRep
		aOpenBountiesFn = origOpenBounties
		aWithdrawFn = origWithdraw
	}()
	aMoveFn = func(int, int, ...bool) error { return nil }
	aCrimesForFactionFn = func(string, bool) []*crimes.Crime { return nil }
	aResolveCrimeFn = func(string, int, string) {}
	alliesFn = func(string) []string { return nil }
	releaseRoomFn = func(string) int { return 0 }
	aRepResetFn = func() int { return -10 }
	aGetRepFn = func(string, int) int { return 0 }
	aSetRepFn = func(string, int, int) {}
	aOpenBountiesFn = func(int) []*bounties.Bounty { return nil }
	aWithdrawFn = func(int) {}
	var torndown int
	aTeardownCellFn = func(entryRoomId int) { torndown = entryRoomId }

	ch := &characters.Character{}
	ch.SetMiscData(keyJailUntilRound, uint64(100))
	ch.SetMiscData(keyJailFaction, "thornwall_guards")
	ch.SetMiscData(keyJailInstanceId, 60001)

	if !ResolveDetention(ch, 42) {
		t.Fatalf("resolve should succeed")
	}
	if torndown != 60001 {
		t.Fatalf("expected instance 60001 torn down, got %d", torndown)
	}
	if _, jailed := JailInfo(ch); jailed {
		t.Fatalf("jail record should be cleared")
	}
}

func TestResolveDetention_LegacyStaticCellNoTeardown(t *testing.T) {
	origMove, origTeardown, origCrimes := aMoveFn, aTeardownCellFn, aCrimesForFactionFn
	origResolve := aResolveCrimeFn
	origAllies := alliesFn
	origReleaseRoom := releaseRoomFn
	origRepReset := aRepResetFn
	origGetRep := aGetRepFn
	origSetRep := aSetRepFn
	origOpenBounties := aOpenBountiesFn
	origWithdraw := aWithdrawFn
	defer func() {
		aMoveFn, aTeardownCellFn, aCrimesForFactionFn = origMove, origTeardown, origCrimes
		aResolveCrimeFn = origResolve
		alliesFn = origAllies
		releaseRoomFn = origReleaseRoom
		aRepResetFn = origRepReset
		aGetRepFn = origGetRep
		aSetRepFn = origSetRep
		aOpenBountiesFn = origOpenBounties
		aWithdrawFn = origWithdraw
	}()
	aMoveFn = func(int, int, ...bool) error { return nil }
	aCrimesForFactionFn = func(string, bool) []*crimes.Crime { return nil }
	aResolveCrimeFn = func(string, int, string) {}
	alliesFn = func(string) []string { return nil }
	releaseRoomFn = func(string) int { return 0 }
	aRepResetFn = func() int { return -10 }
	aGetRepFn = func(string, int) int { return 0 }
	aSetRepFn = func(string, int, int) {}
	aOpenBountiesFn = func(int) []*bounties.Bounty { return nil }
	aWithdrawFn = func(int) {}
	teardownCalled := false
	aTeardownCellFn = func(int) { teardownCalled = true }

	ch := &characters.Character{}
	ch.SetMiscData(keyJailUntilRound, uint64(100))
	ch.SetMiscData(keyJailFaction, "thornwall_guards")
	ch.SetMiscData(keyJailInstanceId, 0)

	ResolveDetention(ch, 42)
	if teardownCalled {
		t.Fatalf("legacy static-cell release must NOT call instance teardown")
	}
}

// ---------------------------------------------------------------------------
// HandleJailedDespawn tests (Task 5)
// ---------------------------------------------------------------------------

func TestHandleJailedDespawn_TearsDownKeepsRecordRewritesRoom(t *testing.T) {
	origTeardown, origCell := aTeardownCellFn, cellRoomFn
	defer func() { aTeardownCellFn, cellRoomFn = origTeardown, origCell }()
	var torndown int
	aTeardownCellFn = func(id int) { torndown = id }
	cellRoomFn = func(string) int { return 473 }

	ch := &characters.Character{}
	ch.SetMiscData(keyJailUntilRound, uint64(9999))
	ch.SetMiscData(keyJailFaction, "thornwall_guards")
	ch.SetMiscData(keyJailInstanceId, 60001)

	HandleJailedDespawn(ch)

	if torndown != 60001 {
		t.Fatalf("expected instance torn down, got %d", torndown)
	}
	if _, jailed := JailInfo(ch); !jailed {
		t.Fatalf("jail record must survive logout")
	}
	if instId, _ := miscDataInt(ch.MiscData, keyJailInstanceId); instId != 0 {
		t.Fatalf("InstanceId must be cleared, got %d", instId)
	}
	if ch.RoomId != 473 {
		t.Fatalf("expected saved RoomId 473, got %d", ch.RoomId)
	}
}

func TestHandleJailedDespawn_NotJailedNoOp(t *testing.T) {
	origTeardown := aTeardownCellFn
	defer func() { aTeardownCellFn = origTeardown }()
	called := false
	aTeardownCellFn = func(int) { called = true }
	ch := &characters.Character{}
	HandleJailedDespawn(ch)
	if called {
		t.Fatalf("non-jailed despawn must be a no-op")
	}
}

// TestHandleJailedDespawn_StaticCellNoTeardown verifies the static-cell path:
// when instId == 0 the teardown seam must NOT be called, the jail record must
// survive, and RoomId must still be rewritten to the faction's fallback cell.
func TestHandleJailedDespawn_StaticCellNoTeardown(t *testing.T) {
	origTeardown, origCell := aTeardownCellFn, cellRoomFn
	defer func() { aTeardownCellFn, cellRoomFn = origTeardown, origCell }()

	teardownCalled := false
	aTeardownCellFn = func(int) { teardownCalled = true }
	cellRoomFn = func(string) int { return 473 }

	ch := &characters.Character{}
	ch.SetMiscData(keyJailUntilRound, uint64(9999))
	ch.SetMiscData(keyJailFaction, "thornwall_guards")
	ch.SetMiscData(keyJailInstanceId, 0)

	HandleJailedDespawn(ch)

	if teardownCalled {
		t.Fatalf("static-cell despawn must NOT call aTeardownCellFn (instId == 0)")
	}
	if _, jailed := JailInfo(ch); !jailed {
		t.Fatalf("jail record must survive logout for static-cell player")
	}
	if ch.RoomId != 473 {
		t.Fatalf("expected saved RoomId 473 (fallback cell), got %d", ch.RoomId)
	}
}

// ---------------------------------------------------------------------------
// RestoreJailOnLogin tests (Task 6)
// ---------------------------------------------------------------------------

func TestRestoreJailOnLogin_ReleasesWhenSentenceServedOffline(t *testing.T) {
	origNow, origMove, origCrimes := bNowFn, aMoveFn, aCrimesForFactionFn
	origAllies := alliesFn
	origOpenBounties, origWithdraw := aOpenBountiesFn, aWithdrawFn
	origGetRep, origSetRep, origRepReset := aGetRepFn, aSetRepFn, aRepResetFn
	origRelease := releaseRoomFn
	defer func() {
		bNowFn, aMoveFn, aCrimesForFactionFn = origNow, origMove, origCrimes
		alliesFn = origAllies
		aOpenBountiesFn, aWithdrawFn = origOpenBounties, origWithdraw
		aGetRepFn, aSetRepFn, aRepResetFn = origGetRep, origSetRep, origRepReset
		releaseRoomFn = origRelease
	}()
	bNowFn = func() uint64 { return 200 }
	aMoveFn = func(int, int, ...bool) error { return nil }
	aCrimesForFactionFn = func(string, bool) []*crimes.Crime { return nil }
	alliesFn = func(string) []string { return nil }
	aOpenBountiesFn = func(int) []*bounties.Bounty { return nil }
	aWithdrawFn = func(int) {}
	aGetRepFn = func(string, int) int { return 0 }
	aSetRepFn = func(string, int, int) {}
	aRepResetFn = func() int { return -10 }
	releaseRoomFn = func(string) int { return 473 }

	ch := &characters.Character{}
	ch.SetMiscData(keyJailUntilRound, uint64(100)) // expired: 100 <= 200
	ch.SetMiscData(keyJailFaction, "thornwall_guards")
	ch.SetMiscData(keyJailInstanceId, 0)

	RestoreJailOnLogin(ch, 42)
	if _, jailed := JailInfo(ch); jailed {
		t.Fatalf("expired sentence should be released on login")
	}
}

func TestRestoreJailOnLogin_ReInstancesWhenStillServing(t *testing.T) {
	origNow, origMove, origCreate, origRelease := bNowFn, aMoveFn, aCreateCellFn, releaseRoomFn
	origDesc := aSetCellDescFn
	defer func() {
		bNowFn, aMoveFn, aCreateCellFn, releaseRoomFn = origNow, origMove, origCreate, origRelease
		aSetCellDescFn = origDesc
	}()
	bNowFn = func() uint64 { return 50 } // still serving (until 100)
	releaseRoomFn = func(string) int { return 473 }
	var movedTo int
	aMoveFn = func(_, roomId int, _ ...bool) error { movedTo = roomId; return nil }
	aCreateCellFn = func(int, int) (int, bool) { return 60002, true }
	aSetCellDescFn = func(int, string) {}

	ch := &characters.Character{}
	ch.SetMiscData(keyJailUntilRound, uint64(100))
	ch.SetMiscData(keyJailFaction, "thornwall_guards")
	ch.SetMiscData(keyJailInstanceId, 0) // stale (instance gone on logout)

	RestoreJailOnLogin(ch, 42)
	rec, jailed := JailInfo(ch)
	if !jailed || rec.InstanceId != 60002 {
		t.Fatalf("expected fresh instance 60002, got %+v", rec)
	}
	if movedTo != 60002 {
		t.Fatalf("expected placed in fresh cell 60002, got %d", movedTo)
	}
}

func TestRestoreJailOnLogin_NotJailedNoOp(t *testing.T) {
	ch := &characters.Character{}
	RestoreJailOnLogin(ch, 42) // must not panic / must no-op
	if _, jailed := JailInfo(ch); jailed {
		t.Fatalf("non-jailed login must stay non-jailed")
	}
}

func TestExecuteArrest_FallsBackToStaticCellOnInstanceFailure(t *testing.T) {
	origCell := cellRoomFn
	origMove := aMoveFn
	origCreate := aCreateCellFn
	origDecay := aDecayFn
	origNow := bNowFn
	defer func() {
		cellRoomFn = origCell
		aMoveFn = origMove
		aCreateCellFn = origCreate
		aDecayFn = origDecay
		bNowFn = origNow
	}()

	cellRoomFn = func(string) int { return 473 }
	aDecayFn = func() int { return 5 }
	bNowFn = func() uint64 { return 100 }

	var movedTo int
	aMoveFn = func(userId int, roomId int, isSpawn ...bool) error { movedTo = roomId; return nil }
	aCreateCellFn = func(int, int) (int, bool) { return 0, false }

	ch := &characters.Character{}
	ok := ExecuteArrest(ch, 42, "thornwall_guards", false)
	if !ok {
		t.Fatalf("arrest should succeed via static fallback")
	}
	rec, _ := JailInfo(ch)
	if rec.InstanceId != 0 || rec.CellRoom != 473 || movedTo != 473 {
		t.Fatalf("expected static fallback (cell 473, InstanceId 0), got %+v movedTo=%d", rec, movedTo)
	}
}
