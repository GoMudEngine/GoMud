package factions

import (
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/opinions"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Test-only seam — overrides util.GetRoundCount(). Production code
// never sets this.
var roundForTest func() uint64

func currentRound() uint64 {
	if roundForTest != nil {
		return roundForTest()
	}
	return util.GetRoundCount()
}

// loadOrLazyInit returns the cached *FactionRep for factionId,
// loading from disk on first access. If neither cache nor disk
// has data, an empty FactionRep is created and cached. Returns
// nil if the faction is unknown (no definition loaded).
func loadOrLazyInit(factionId string) *FactionRep {
	repCacheMu.RLock()
	if fr, ok := repCache[factionId]; ok {
		repCacheMu.RUnlock()
		return fr
	}
	repCacheMu.RUnlock()

	def := GetDefinition(factionId)
	if def == nil {
		return nil
	}

	if fr := loadRepFromDisk(factionId); fr != nil {
		repCacheMu.Lock()
		repCache[factionId] = fr
		repCacheMu.Unlock()
		return fr
	}

	fr := &FactionRep{
		FactionId: factionId,
		Players:   map[int]*RepEntry{},
	}
	repCacheMu.Lock()
	repCache[factionId] = fr
	repCacheMu.Unlock()
	return fr
}

// GetRep returns the player's reputation with the given faction,
// or the faction's DefaultRep if no row exists. Pure read — does
// not write to disk. Lazy cache priming may add an empty
// FactionRep to the cache on first access. Returns 0 if the
// faction is unknown.
func GetRep(factionId string, userId int) int {
	def := GetDefinition(factionId)
	if def == nil {
		return 0
	}
	fr := loadOrLazyInit(factionId)
	if fr == nil {
		return 0
	}
	repCacheMu.RLock()
	entry, has := fr.Players[userId]
	repCacheMu.RUnlock()
	if !has {
		return def.DefaultRep
	}
	return entry.Rep
}

// SetRep assigns an absolute rep, clamped to [-100, +100], stamps
// last_updated_round, and persists synchronously.
func SetRep(factionId string, userId int, rep int) {
	fr := loadOrLazyInit(factionId)
	if fr == nil {
		return
	}
	clamped := clampRep(rep)
	now := currentRound()

	repCacheMu.Lock()
	if fr.Players == nil {
		fr.Players = map[int]*RepEntry{}
	}
	fr.Players[userId] = &RepEntry{Rep: clamped, LastUpdatedRound: now}
	repCacheMu.Unlock()

	if err := saveRepToDisk(factionId); err != nil {
		mudlog.Warn("factions.SetRep: saveRepToDisk", "factionId", factionId, "userId", userId, "error", err)
	}
}

// BumpRep adds delta to current rep, clamps, re-stamps, persists.
// No-op if the faction is unknown.
func BumpRep(factionId string, userId int, delta int) {
	fr := loadOrLazyInit(factionId)
	if fr == nil {
		mudlog.Warn("factions.BumpRep: unknown faction", "factionId", factionId)
		return
	}
	def := GetDefinition(factionId)
	now := currentRound()

	repCacheMu.Lock()
	if fr.Players == nil {
		fr.Players = map[int]*RepEntry{}
	}
	entry, has := fr.Players[userId]
	var base int
	if has {
		base = entry.Rep
	} else {
		base = def.DefaultRep
	}
	newRep := clampRep(base + delta)
	fr.Players[userId] = &RepEntry{Rep: newRep, LastUpdatedRound: now}
	repCacheMu.Unlock()

	if err := saveRepToDisk(factionId); err != nil {
		mudlog.Warn("factions.BumpRep: saveRepToDisk", "factionId", factionId, "userId", userId, "error", err)
	}
}

// TierFor returns opinions.TierOf(GetRep(factionId, userId)).
// Reuses the opinion package's banding logic — no duplicate enum.
func TierFor(factionId string, userId int) opinions.Tier {
	return opinions.TierOf(GetRep(factionId, userId))
}

func clampRep(r int) int {
	if r < RepMin {
		return RepMin
	}
	if r > RepMax {
		return RepMax
	}
	return r
}
