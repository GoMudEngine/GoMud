package goals

import (
	"sort"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// loadOrLazyInit returns the cached MobGoals for mobId, loading from
// disk on first access. Returns a fresh empty MobGoals if neither
// cache nor disk has data. Mirrors the chunk-1.3 double-check pattern.
func loadOrLazyInit(mobId int, namesimple string) *MobGoals {
	cacheMu.RLock()
	if mg, ok := cache[mobId]; ok {
		cacheMu.RUnlock()
		return mg
	}
	cacheMu.RUnlock()

	cacheMu.Lock()
	defer cacheMu.Unlock()
	// Double-check after upgrading to write lock.
	if mg, ok := cache[mobId]; ok {
		return mg
	}
	mg := loadFromDisk(mobId, namesimple)
	if mg == nil {
		mg = &MobGoals{MobId: mobId, NextGoalId: 1, Goals: nil}
	}
	cache[mobId] = mg
	nameByMobId[mobId] = namesimple
	return mg
}

// GoalsOf returns the mob's goals in priority-desc, then id-asc order
// (stable for admin output and any future selection layer). Lazy
// loads from disk on first call.
//
// The returned slice is a copy — callers can sort or slice it freely
// without affecting the cache.
func GoalsOf(mobId int, namesimple string) []*Goal {
	mg := loadOrLazyInit(mobId, namesimple)
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	out := make([]*Goal, len(mg.Goals))
	copy(out, mg.Goals)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Priority != out[j].Priority {
			return out[i].Priority > out[j].Priority
		}
		return out[i].Id < out[j].Id
	})
	return out
}

// IsSatisfied looks up the predicate registered for g.Type and
// returns its result. Returns false if no predicate is registered —
// safe default: a goal we don't know how to evaluate stays alive.
func IsSatisfied(g *Goal, mob *mobs.Mob) bool {
	meta, ok := lookupMeta(g.Type)
	if !ok || meta.Predicate == nil {
		return false
	}
	return meta.Predicate(g, mob)
}

// IsExpired is a pure time check. Goals with ExpiresAt.IsZero()
// never expire.
func IsExpired(g *Goal, now time.Time) bool {
	if g.ExpiresAt.IsZero() {
		return false
	}
	return !now.Before(g.ExpiresAt)
}
