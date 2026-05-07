package crimes

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Test-only seam — overrides util.GetRoundCount(). Production
// never sets this.
var roundForTest func() uint64

func currentRound() uint64 {
	if roundForTest != nil {
		return roundForTest()
	}
	return util.GetRoundCount()
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
