package justice

// arrest.go — fine math, ExecuteArrest, ResolveDetention, and JailInfo.
//
// Package justice must NOT import internal/actions or internal/usercommands.
// Player messages use users.GetByUserId + UserRecord.SendText directly.

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/bounties"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crimes"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// ---------------------------------------------------------------------------
// MiscData key constants for the jail record.
// ---------------------------------------------------------------------------

const (
	keyJailUntilRound    = "jail_until_round"
	keyJailFineOriginal  = "jail_fine_original"
	keyJailDecayPerRound = "jail_decay_per_round"
	keyJailFaction       = "jail_faction"
	keyJailCellRoom      = "jail_cell_room"
	keyJailCrimeIds      = "jail_crime_ids"
)

// barracksRoomId is the release destination after a sentence is served.
const barracksRoomId = 473

// jailedBuffId is the buff applied to jailed players (88-jailed.yaml).
const jailedBuffId = 88

// ---------------------------------------------------------------------------
// Config-reader seams (tests override).
// ---------------------------------------------------------------------------

// aDecayFn returns JusticeFineDecayPerRound (gold the fine drops per served
// round). Default 5 if the knob is unset.
var aDecayFn = func() int {
	v := int(configs.GetBalanceConfig().JusticeFineDecayPerRound)
	if v < 1 {
		return 5
	}
	return v
}

// aRepResetFn returns JusticeArrestRepReset (rep floor restored on release).
// Default -10 if the knob is unset.
var aRepResetFn = func() int {
	v := int(configs.GetBalanceConfig().JusticeArrestRepReset)
	if v == 0 {
		return -10
	}
	return v
}

// ---------------------------------------------------------------------------
// Release-path seams (tests override).
// ---------------------------------------------------------------------------

var aResolveCrimeFn = crimes.Resolve
var aCrimesForFactionFn = crimes.AllForFaction
var aOpenBountiesFn = bounties.OpenAgainstPlayer
var aWithdrawFn = bounties.Withdraw
var aGetRepFn = factions.GetRep
var aSetRepFn = factions.SetRep
var aMoveFn = rooms.MoveToRoom

// ---------------------------------------------------------------------------
// Pure helpers (unit-tested in arrest_test.go).
// ---------------------------------------------------------------------------

// currentFine returns the outstanding fine after roundsServed rounds.
// Floors at zero so the fine never goes negative.
func currentFine(original, roundsServed, decayPerRound int) int {
	remaining := original - roundsServed*decayPerRound
	if remaining < 0 {
		return 0
	}
	return remaining
}

// sentenceRounds returns how many rounds a sentence lasts for the given fine
// and per-round decay rate. Floors at 1 so every sentence has at least one
// round.
func sentenceRounds(fine, decayPerRound int) int {
	if decayPerRound < 1 {
		decayPerRound = 1
	}
	r := fine / decayPerRound
	if r < 1 {
		return 1
	}
	return r
}

// computeFine calculates the initial fine for arresting a player, reusing the
// 5.1b bounty-gold formula so the fine is proportional to the bounty the
// player attracted.
func computeFine(faction string, userId int, isMurder bool) int {
	return bountyGold(
		bDefaultGoldFn(knowledge.PlayerSubject(userId)),
		isMurder,
		bRepFn(faction, userId),
		bMurderMultFn(),
		bRepMultMaxFn(),
	)
}

// ---------------------------------------------------------------------------
// MiscData helpers.
// ---------------------------------------------------------------------------

// miscDataInt reads an int value stored under key in MiscData, tolerating the
// various numeric kinds a YAML round-trip may produce.
func miscDataInt(misc map[string]any, key string) (int, bool) {
	v, ok := misc[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case uint64:
		return int(n), true
	case float64:
		return int(n), true
	}
	return 0, false
}

// miscDataString reads a string value from MiscData under key.
func miscDataString(misc map[string]any, key string) (string, bool) {
	v, ok := misc[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// parseCrimeIds splits a comma-separated string of crime ids into a []int.
// An empty string returns nil (no crimes to resolve).
func parseCrimeIds(raw string) []int {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.Atoi(p)
		if err == nil {
			out = append(out, id)
		}
	}
	return out
}

// crimeIdsForFactionPlayer collects unresolved crime ids logged against
// `faction` that name `userId` as the perpetrator.
func crimeIdsForFactionPlayer(faction string, userId int) []int {
	all := crimes.AllForFaction(faction, false)
	out := make([]int, 0)
	for _, c := range all {
		if c.Perpetrator.Type == crimes.PerpPlayer && c.Perpetrator.Id == userId {
			out = append(out, c.Id)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// JailInfo — exported accessor for the Task 8 `fine` command.
// ---------------------------------------------------------------------------

// JailRecord holds the stamped jail fields for an in-custody player.
type JailRecord struct {
	FineOriginal  int
	DecayPerRound int
	UntilRound    uint64
	Faction       string
	CellRoom      int
}

// JailInfo returns the jail record stamped on player's Character, and whether
// the player is actually jailed. Returns (zero, false) when no record exists.
func JailInfo(player *characters.Character) (JailRecord, bool) {
	if player == nil || player.MiscData == nil {
		return JailRecord{}, false
	}
	untilRound, ok := miscDataRound(player.MiscData, keyJailUntilRound)
	if !ok {
		return JailRecord{}, false
	}
	fine, _ := miscDataInt(player.MiscData, keyJailFineOriginal)
	decay, _ := miscDataInt(player.MiscData, keyJailDecayPerRound)
	faction, _ := miscDataString(player.MiscData, keyJailFaction)
	cell, _ := miscDataInt(player.MiscData, keyJailCellRoom)
	return JailRecord{
		FineOriginal:  fine,
		DecayPerRound: decay,
		UntilRound:    untilRound,
		Faction:       faction,
		CellRoom:      cell,
	}, true
}

// ---------------------------------------------------------------------------
// ExecuteArrest
// ---------------------------------------------------------------------------

// ExecuteArrest hauls a surrendered player to the faction's holding cell,
// applies the Jailed buff (buff 88) for the sentence duration, stamps the
// jail record in MiscData, and sends drag-to-jail flavor to the player.
// Returns false (no-op) if the faction has no registered holding cell.
//
// The Jailed buff duration is set via Character.AddBuffScaled: the buff's
// triggercount (1) is scaled by float64(rounds) so TriggersLeft == rounds.
// The buff will therefore expire naturally after `rounds` ticks. The
// jail_until_round MiscData key provides a parallel time-stamp so Task 9's
// per-round release check can also fire when the buff has already expired.
func ExecuteArrest(player *characters.Character, userId int, faction string, isMurder bool) bool {
	cell := cellRoomFn(faction)
	if cell == 0 {
		return false
	}

	decay := aDecayFn()
	fine := computeFine(faction, userId, isMurder)
	rounds := sentenceRounds(fine, decay)

	// Collect the player's unresolved crime ids for this faction so they can
	// be resolved on release without hitting global state at arrest time.
	ids := crimeIdsForFactionPlayer(faction, userId)
	idsStr := ""
	if len(ids) > 0 {
		parts := make([]string, len(ids))
		for i, id := range ids {
			parts[i] = strconv.Itoa(id)
		}
		idsStr = strings.Join(parts, ",")
	}

	// Stamp the jail record.
	now := bNowFn()
	player.SetMiscData(keyJailUntilRound, now+uint64(rounds))
	player.SetMiscData(keyJailFineOriginal, fine)
	player.SetMiscData(keyJailDecayPerRound, decay)
	player.SetMiscData(keyJailFaction, faction)
	player.SetMiscData(keyJailCellRoom, cell)
	player.SetMiscData(keyJailCrimeIds, idsStr)

	// Apply the Jailed buff scaled to `rounds` triggers so it expires
	// naturally at the end of the sentence. The buff also carries the
	// no-aggro-target flag, which makes the jailed player invisible to ALL mob
	// aggro paths (LookForTrouble, retarget, etc.) — without it, a guard that
	// reaches the cell re-acquires aggro and fights the prisoner (smoke BUG-04;
	// RunGuardEnforcement's own exemption only covered the justice tick, not the
	// legacy group-hostile LookForTrouble path).
	_ = player.AddBuffScaled(jailedBuffId, float64(rounds))

	// Drop any combat the player was in — they're in custody now, not fighting.
	player.EndAggro()

	// Haul the player to the holding cell.
	_ = aMoveFn(userId, cell)

	// Arrival flavor (the buff's start_user_text fires via the buff system;
	// send an additional arrest-context line here).
	if u := users.GetByUserId(userId); u != nil {
		u.SendText(messaging.CategorySystem,
			fmt.Sprintf("A guard seizes you and hauls you to the holding cell. "+
				"You have been placed under arrest by the %s.", faction))
	}

	return true
}

// ---------------------------------------------------------------------------
// ResolveDetention
// ---------------------------------------------------------------------------

// ResolveDetention ends a detention (timer expiry OR fine paid): resolves the
// stamped crimes, withdraws the issuing faction's open bounties, resets rep
// to the floor only if currently below it, removes the Jailed buff, clears
// the jail record, and moves the player to the barracks. Returns false if
// the player has no active jail record.
func ResolveDetention(player *characters.Character, userId int) bool {
	if player == nil || player.MiscData == nil {
		return false
	}

	_, ok := miscDataRound(player.MiscData, keyJailUntilRound)
	if !ok {
		return false
	}

	faction, _ := miscDataString(player.MiscData, keyJailFaction)

	// Serving the sentence answers for the arresting faction AND its allies —
	// the same set Verdict checks. Guards belong to thornwall_guards AND
	// thornwall_citizens, so clearing only the arresting faction left an
	// unresolved crime against the ally that re-triggered arrest the instant
	// the player was released (5.1c smoke BUG-03). Clear comprehensively via a
	// live query rather than the crime ids stamped at arrest time.
	factionSet := map[string]bool{faction: true}
	for _, a := range alliesFn(faction) {
		factionSet[a] = true
	}
	for f := range factionSet {
		for _, c := range aCrimesForFactionFn(f, false) {
			if c.Perpetrator.Type == crimes.PerpPlayer && c.Perpetrator.Id == userId {
				aResolveCrimeFn(f, c.Id, "served sentence")
			}
		}
	}

	// Withdraw any open bounties issued by the faction set against this player.
	for _, b := range aOpenBountiesFn(userId) {
		if b.Issuer.Type == bounties.IssuerFaction && factionSet[b.Issuer.Id] {
			aWithdrawFn(b.Id)
		}
	}

	// Reset rep to the floor for each faction in the set, only where the
	// player's rep is currently below it (never lowers good standing).
	floor := aRepResetFn()
	for f := range factionSet {
		if aGetRepFn(f, userId) < floor {
			aSetRepFn(f, userId, floor)
		}
	}

	// Remove the Jailed buff.
	player.RemoveBuff(jailedBuffId)

	// Clear all jail MiscData keys.
	player.SetMiscData(keyJailUntilRound, nil)
	player.SetMiscData(keyJailFineOriginal, nil)
	player.SetMiscData(keyJailDecayPerRound, nil)
	player.SetMiscData(keyJailFaction, nil)
	player.SetMiscData(keyJailCellRoom, nil)
	player.SetMiscData(keyJailCrimeIds, nil)

	// Move the player to the barracks (release destination).
	_ = aMoveFn(userId, barracksRoomId)

	// No release flavor here: removing the Jailed buff (above) fires its
	// end_user_text ("The cell door swings open. You are free to go."), which
	// covers both the payfine and timer-expiry paths. Sending it again here
	// double-printed the line (5.1c smoke BUG-01).

	return true
}
