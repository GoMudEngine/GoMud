package goals

import (
	"fmt"
	"sort"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
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

// Add appends a goal to the mob's list, resolving conflicts by
// priority. Returns *ConflictError if any conflicting existing goal
// has priority >= the new goal's priority. Persists to disk under the
// write mutex.
//
// g.Id is ignored on entry and assigned by Add; g.OwnerMobId is set
// from mobId; g.CreatedAt is stamped to time.Now().UTC() if zero.
func Add(mobId int, namesimple string, g *Goal) (AddResult, error) {
	mg := loadOrLazyInit(mobId, namesimple)

	cacheMu.Lock()

	// Detect conflicting existing goals. "Same type" always conflicts
	// (no AllowMultiple opt-in in 4.1). Cross-type uses the registered
	// ConflictsWith list, with a symmetry safety net checking both
	// directions.
	newMeta, _ := lookupMeta(g.Type)
	var conflicting []*Goal
	for _, e := range mg.Goals {
		if isConflict(g.Type, e.Type, newMeta) {
			conflicting = append(conflicting, e)
		}
	}

	// Priority resolution: every conflicting existing goal must have
	// strictly lower priority for the new goal to win.
	for _, e := range conflicting {
		if g.Priority <= e.Priority {
			cacheMu.Unlock()
			return AddResult{}, &ConflictError{
				BlockerGoalId: e.Id,
				BlockerType:   e.Type,
				BlockerPrio:   e.Priority,
			}
		}
	}

	// Displace lower-priority conflicting goals in place.
	displaced := make([]string, 0, len(conflicting))
	if len(conflicting) > 0 {
		mg.Goals = removeGoals(mg.Goals, conflicting)
		for _, e := range conflicting {
			displaced = append(displaced, e.Id)
		}
	}

	// Assign id, owner, timestamp, and append.
	g.Id = fmt.Sprintf("g%d", mg.NextGoalId)
	g.OwnerMobId = mobId
	if g.CreatedAt.IsZero() {
		g.CreatedAt = time.Now().UTC()
	}
	mg.NextGoalId++
	mg.Goals = append(mg.Goals, g)
	nameByMobId[mobId] = namesimple
	cacheMu.Unlock()

	if err := saveToDisk(mobId, namesimple); err != nil {
		mudlog.Warn("goals.Add: save failed", "mob_id", mobId, "error", err)
		// Cache is still authoritative; caller treats as success.
	}
	return AddResult{Added: g, Displaced: displaced}, nil
}

// isConflict reports whether existingType conflicts with newType per
// the registry. Same-type is always a conflict in 4.1. Cross-type
// checks newMeta.ConflictsWith and also looks up the existing type's
// metadata as a symmetry safety net (so a one-sided declaration still
// catches the conflict).
func isConflict(newType, existingType string, newMeta GoalTypeMeta) bool {
	if newType == existingType {
		return true
	}
	if sliceContains(newMeta.ConflictsWith, existingType) {
		return true
	}
	if existingMeta, ok := lookupMeta(existingType); ok {
		if sliceContains(existingMeta.ConflictsWith, newType) {
			return true
		}
	}
	return false
}

// removeGoals returns goals with the items in drop removed, preserving
// order. O(n*m) but n and m are small (goals-per-mob ≤ ~10 in practice).
func removeGoals(goals []*Goal, drop []*Goal) []*Goal {
	dropIds := make(map[string]bool, len(drop))
	for _, d := range drop {
		dropIds[d.Id] = true
	}
	out := goals[:0:0]
	for _, g := range goals {
		if !dropIds[g.Id] {
			out = append(out, g)
		}
	}
	return out
}

// Remove deletes a goal by id. Returns ErrGoalNotFound if the id is
// not present on the mob. NextGoalId is NOT decremented — ids are
// never reused within the lifetime of a mob's file.
func Remove(mobId int, namesimple, goalId string) error {
	mg := loadOrLazyInit(mobId, namesimple)
	cacheMu.Lock()
	found := false
	out := mg.Goals[:0:0]
	for _, g := range mg.Goals {
		if g.Id == goalId {
			found = true
			continue
		}
		out = append(out, g)
	}
	if !found {
		cacheMu.Unlock()
		return ErrGoalNotFound
	}
	mg.Goals = out
	nameByMobId[mobId] = namesimple
	cacheMu.Unlock()

	if err := saveToDisk(mobId, namesimple); err != nil {
		mudlog.Warn("goals.Remove: save failed", "mob_id", mobId, "error", err)
	}
	return nil
}

// Clear removes every goal from the mob and resets NextGoalId to 1.
// Admin-only — intentionally heavy-hand for resetting a mob's goal
// state to defaults.
func Clear(mobId int, namesimple string) error {
	mg := loadOrLazyInit(mobId, namesimple)
	cacheMu.Lock()
	mg.Goals = nil
	mg.NextGoalId = 1
	nameByMobId[mobId] = namesimple
	cacheMu.Unlock()

	if err := saveToDisk(mobId, namesimple); err != nil {
		mudlog.Warn("goals.Clear: save failed", "mob_id", mobId, "error", err)
	}
	return nil
}
