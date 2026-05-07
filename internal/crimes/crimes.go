package crimes

import (
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Test-only seam — overrides util.GetRoundCount(). Production
// never sets this.
var roundForTest func() uint64

// staleAfterForTest overrides Balance.CrimeStaleAfterRounds for
// tests. Production never sets this. T8 adds the production
// config field; until then, currentStaleAfter() returns 0 in
// production (PruneStale is a no-op without the test seam).
var staleAfterForTest func() uint64

func currentRound() uint64 {
	if roundForTest != nil {
		return roundForTest()
	}
	return util.GetRoundCount()
}

func currentStaleAfter() uint64 {
	if staleAfterForTest != nil {
		return staleAfterForTest()
	}
	// T8 wires this to configs.GetBalanceConfig().CrimeStaleAfterRounds.
	// Until then, production returns 0 (PruneStale early-returns).
	return 0
}

// loadOrLazyInit returns the cached *FactionCrimes for factionId,
// loading from disk on first access. If neither cache nor disk
// has data, an empty FactionCrimes is created and cached.
func loadOrLazyInit(factionId string) *FactionCrimes {
	crimeCacheMu.RLock()
	if fc, ok := crimeCache[factionId]; ok {
		crimeCacheMu.RUnlock()
		return fc
	}
	crimeCacheMu.RUnlock()

	if fc := loadCrimesFromDisk(factionId); fc != nil {
		crimeCacheMu.Lock()
		// Recheck after acquiring write lock; another goroutine may
		// have loaded concurrently.
		if cached, ok := crimeCache[factionId]; ok {
			crimeCacheMu.Unlock()
			return cached
		}
		crimeCache[factionId] = fc
		crimeCacheMu.Unlock()
		return fc
	}

	fc := &FactionCrimes{
		FactionId: factionId,
		Crimes:    []*Crime{},
		nextId:    1,
	}
	crimeCacheMu.Lock()
	// Recheck after acquiring write lock; another goroutine may have
	// created and cached a new entry.
	if cached, ok := crimeCache[factionId]; ok {
		crimeCacheMu.Unlock()
		return cached
	}
	crimeCache[factionId] = fc
	crimeCacheMu.Unlock()
	return fc
}

// Record creates a new crime row on each affected faction's log.
// Returns the new crime IDs (parallel to factionIds order).
// Persists synchronously per-faction.
func Record(
	factionIds []string,
	kind Kind,
	perp Perpetrator,
	victim *mobs.Mob,
	instanceId int,
	roomId int,
	zone string,
) []int {
	if victim == nil || len(factionIds) == 0 {
		return nil
	}
	now := currentRound()
	out := make([]int, 0, len(factionIds))

	for _, fid := range factionIds {
		fc := loadOrLazyInit(fid)

		crimeCacheMu.Lock()
		c := &Crime{
			Id:               fc.nextId,
			Kind:             kind,
			Zone:             zone,
			RoomId:           roomId,
			Round:            now,
			VictimMobId:      int(victim.MobId),
			VictimInstanceId: instanceId,
			Perpetrator:      perp,
		}
		fc.nextId++
		fc.Crimes = append(fc.Crimes, c)
		crimeCacheMu.Unlock()

		if err := saveCrimesToDisk(fid); err != nil {
			mudlog.Warn("crimes.Record: saveCrimesToDisk", "factionId", fid, "error", err)
		}
		out = append(out, c.Id)
	}
	return out
}

// Resolve marks a specific crime as resolved. Idempotent — re-
// resolving is a no-op (preserves original resolved_round and
// resolved_by).
func Resolve(factionId string, crimeId int, resolvedBy string) {
	fc := loadOrLazyInit(factionId)
	now := currentRound()

	crimeCacheMu.Lock()
	mutated := false
	for _, c := range fc.Crimes {
		if c.Id == crimeId && c.ResolvedRound == 0 {
			c.ResolvedRound = now
			c.ResolvedBy = resolvedBy
			mutated = true
			break
		}
	}
	crimeCacheMu.Unlock()

	if mutated {
		if err := saveCrimesToDisk(factionId); err != nil {
			mudlog.Warn("crimes.Resolve: saveCrimesToDisk", "factionId", factionId, "crimeId", crimeId, "error", err)
		}
	}
}

// AllForFaction returns crimes against the given faction. Pass
// includeResolved=false to skip cleared records.
func AllForFaction(factionId string, includeResolved bool) []*Crime {
	fc := loadOrLazyInit(factionId)
	crimeCacheMu.RLock()
	defer crimeCacheMu.RUnlock()
	out := make([]*Crime, 0, len(fc.Crimes))
	for _, c := range fc.Crimes {
		if !includeResolved && c.ResolvedRound != 0 {
			continue
		}
		out = append(out, c)
	}
	return out
}

// AllForPlayer returns crimes naming this userId as the identified
// perpetrator, across all factions. Walks the cache; does not
// load from disk for factions that haven't been touched. (Admin
// command may want a separate disk-walking helper later if it
// matters.)
func AllForPlayer(userId int, includeResolved bool) []*Crime {
	crimeCacheMu.RLock()
	defer crimeCacheMu.RUnlock()
	out := make([]*Crime, 0)
	for _, fc := range crimeCache {
		for _, c := range fc.Crimes {
			if c.Perpetrator.Type != PerpPlayer || c.Perpetrator.Id != userId {
				continue
			}
			if !includeResolved && c.ResolvedRound != 0 {
				continue
			}
			out = append(out, c)
		}
	}
	return out
}

// WitnessesInRoom returns the list of mob instance IDs in the
// given room whose mob template's Groups overlap any of factionIds.
// Pass excludeInstanceId = victim's instance for murder (victim is
// dead, not a self-witness); pass 0 for assault and theft (victim
// is alive and a self-witness).
func WitnessesInRoom(factionIds []string, room *rooms.Room, excludeInstanceId int) []int {
	if room == nil || len(factionIds) == 0 {
		return nil
	}
	wantSet := make(map[string]struct{}, len(factionIds))
	for _, fid := range factionIds {
		wantSet[fid] = struct{}{}
	}

	out := make([]int, 0)
	for _, instId := range room.GetMobs() {
		if instId == excludeInstanceId {
			continue
		}
		mob := mobs.GetInstance(instId)
		if mob == nil {
			continue
		}
		// FactionsForMob would re-walk the registry; we just need
		// "does mob.Groups overlap factionIds" — cheap to do inline.
		for _, g := range mob.Groups {
			if _, hit := wantSet[g]; hit {
				if factions.GetDefinition(g) != nil {
					out = append(out, instId)
					break
				}
			}
		}
	}
	return out
}

// IdentifiedPerp returns PerpPlayer if witnesses is non-empty,
// otherwise PerpUnknown.
func IdentifiedPerp(userId int, witnesses []int) Perpetrator {
	if len(witnesses) == 0 {
		return Perpetrator{Type: PerpUnknown}
	}
	return Perpetrator{Type: PerpPlayer, Id: userId}
}

// FindRecentAssault returns the most recent unresolved assault
// crime committed by `userId` against `factionId` within
// `lookbackRounds`. Used by the combat-death hookup to upgrade
// in place when a fight escalates from assault to murder.
//
// Returns nil if no match. Murder rows and resolved rows are
// ignored. Searches in reverse order so the most recent matching
// assault wins.
func FindRecentAssault(factionId string, userId int, lookbackRounds uint64) *Crime {
	fc := loadOrLazyInit(factionId)
	now := currentRound()
	if now < lookbackRounds {
		// avoid uint underflow
		lookbackRounds = now
	}
	cutoff := now - lookbackRounds

	crimeCacheMu.RLock()
	defer crimeCacheMu.RUnlock()
	for i := len(fc.Crimes) - 1; i >= 0; i-- {
		c := fc.Crimes[i]
		if c.Kind != KindAssault {
			continue
		}
		if c.ResolvedRound != 0 {
			continue
		}
		if c.Perpetrator.Type != PerpPlayer || c.Perpetrator.Id != userId {
			continue
		}
		if c.Round < cutoff {
			break // log is append-order, anything older is older
		}
		return c
	}
	return nil
}

// PruneStale resolves all unresolved crimes older than
// Balance.CrimeStaleAfterRounds with reason "stale". Returns the
// number of rows resolved. Persists once per call (not once per
// row) by mutating in-cache then calling saveCrimesToDisk after
// the loop.
//
// Safety net for indefinite-storage growth — primary expiry is
// consumer-driven (town justice fines, redemption quests).
func PruneStale(factionId string) int {
	fc := loadOrLazyInit(factionId)
	now := currentRound()
	threshold := currentStaleAfter()
	if threshold == 0 || now < threshold {
		return 0
	}
	cutoff := now - threshold

	crimeCacheMu.Lock()
	count := 0
	for _, c := range fc.Crimes {
		if c.ResolvedRound != 0 {
			continue
		}
		if c.Round >= cutoff {
			continue
		}
		c.ResolvedRound = now
		c.ResolvedBy = "stale"
		count++
	}
	crimeCacheMu.Unlock()

	if count > 0 {
		if err := saveCrimesToDisk(factionId); err != nil {
			mudlog.Warn("crimes.PruneStale: saveCrimesToDisk", "factionId", factionId, "error", err)
		}
	}
	return count
}
