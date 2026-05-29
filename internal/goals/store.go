package goals

import (
	"fmt"
	"sort"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// loadOrLazyInit returns the cached MobGoals for mobId, loading from
// disk on first access. Returns a fresh empty MobGoals if neither
// cache nor disk has data. Mirrors the chunk-1.3 double-check pattern.
//
// For truly fresh mobs (no file on disk), calls seedFromArchetype after
// releasing the write lock — so that Add re-entrancy inside the seed is
// safe (cache entry already present, mutex free). Chunk 4.3.
func loadOrLazyInit(mobId int, namesimple string) *MobGoals {
	cacheMu.RLock()
	if mg, ok := cache[mobId]; ok {
		cacheMu.RUnlock()
		return mg
	}
	cacheMu.RUnlock()

	cacheMu.Lock()
	// Double-check after upgrading to write lock.
	if mg, ok := cache[mobId]; ok {
		cacheMu.Unlock()
		return mg
	}
	mg := loadFromDisk(mobId, namesimple)
	freshMob := mg == nil
	if freshMob {
		mg = &MobGoals{MobId: mobId, NextGoalId: 1, Goals: nil}
	}
	cache[mobId] = mg
	nameByMobId[mobId] = namesimple
	cacheMu.Unlock() // release BEFORE seedFromArchetype so Add re-entrancy is safe

	if freshMob {
		// Chunk 4.3: seed archetype defaults. Runs outside the lock.
		// seedFromArchetype checks mg.SeededFromArchetype for idempotency.
		seedFromArchetype(mobId, namesimple, mg)
	}
	return mg
}

// GoalsOf returns the mob's goals in priority-desc, then id-asc order
// (stable for admin output and any future selection layer). Lazy
// loads from disk on first call.
//
// The returned slice is a copy — callers can sort or slice it freely
// without affecting the cache.
//
// The returned slice is a fresh copy of the POINTERS — the *Goal values
// still share the cache's backing objects. Fields mutated by Prune under
// cacheMu (e.g. DormantSinceRound) may be torn if read from these pointers
// without holding cacheMu; treat such fields as lock-guarded.
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

	// Chunk 4.3: validate params against the registered type's schema.
	if meta, ok := lookupMeta(g.Type); ok {
		if err := ValidateParams(g, meta.Params); err != nil {
			return AddResult{}, err
		}
	}

	cacheMu.Lock()

	// Detect conflicting existing goals. "Same type" always conflicts
	// (no AllowMultiple opt-in in 4.1). Cross-type uses the registered
	// ConflictsWith list, with a symmetry safety net checking both
	// directions.
	newMeta, _ := lookupMeta(g.Type)
	var newKey string
	if newMeta.AllowMultiple && newMeta.DedupKey != nil {
		newKey = invokeDedupKey(newMeta.DedupKey, g)
	}
	var conflicting []*Goal
	for _, e := range mg.Goals {
		if g.Type == e.Type {
			// Chunk 4.3: same-type pair. AllowMultiple=true + matching
			// DedupKey collides; AllowMultiple=false keeps 4.1's
			// "same type always conflicts" semantics.
			if newMeta.AllowMultiple {
				if newMeta.DedupKey != nil && newKey != "" {
					existingKey := invokeDedupKey(newMeta.DedupKey, e)
					if existingKey == newKey {
						conflicting = append(conflicting, e)
					}
				}
				// AllowMultiple + no key match → coexist (do nothing).
				continue
			}
			conflicting = append(conflicting, e)
			continue
		}
		// Cross-type: 4.1's type-name-based ConflictsWith lookup.
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
	// Eager Recompute so callers see consistent CurrentGoalOf state
	// immediately (next tick will re-Recompute too — idempotent).
	// Best-effort: if no instance is loaded for this template, we
	// still want the file's CurrentGoalId to reflect the new top.
	// Pass nil mob — ContextScore funcs that need mob state will
	// score the goal at 1.0 (the panic-recovered default).
	Recompute(mobId, namesimple, instanceForRecompute(mobId), util.GetRoundCount())
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
//
// If the removed goal was current (chunk 4.2 selection state),
// clear CurrentGoalId / round fields under the same write lock so
// the eager Recompute that follows starts fresh.
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
	if mg.CurrentGoalId == goalId {
		mg.CurrentGoalId = ""
		mg.CurrentSinceRound = 0
		mg.LastSwitchRound = 0
	}
	nameByMobId[mobId] = namesimple
	cacheMu.Unlock()

	if err := saveToDisk(mobId, namesimple); err != nil {
		mudlog.Warn("goals.Remove: save failed", "mob_id", mobId, "error", err)
	}
	Recompute(mobId, namesimple, instanceForRecompute(mobId), util.GetRoundCount())
	return nil
}

// Clear removes every goal from the mob, resets NextGoalId to 1, and
// zeros the chunk-4.2 selection state. Admin-only — intentionally
// heavy-hand for resetting a mob's goal state to defaults.
//
// SeededFromArchetype is intentionally preserved: the sentinel records
// that the archetype-seed already ran for this mob template, so a
// subsequent loadOrLazyInit does NOT re-seed. Chunk 4.3.
func Clear(mobId int, namesimple string) error {
	mg := loadOrLazyInit(mobId, namesimple)
	cacheMu.Lock()
	mg.Goals = nil
	mg.NextGoalId = 1
	mg.CurrentGoalId = ""
	mg.CurrentSinceRound = 0
	mg.LastSwitchRound = 0
	// SeededFromArchetype is NOT cleared here — see doc comment above.
	nameByMobId[mobId] = namesimple
	cacheMu.Unlock()

	if err := saveToDisk(mobId, namesimple); err != nil {
		mudlog.Warn("goals.Clear: save failed", "mob_id", mobId, "error", err)
	}
	// No eager Recompute needed — there are no goals to select.
	return nil
}

// CurrentGoalOf returns the cached current goal for the mob template,
// or nil if there is no current goal or the cached id is stale (the
// referenced goal was removed). Lazy-loads MobGoals on first access
// (matches GoalsOf semantics). Cheap accessor — chunk 4.4 will read
// this from the btree.
func CurrentGoalOf(mobId int, namesimple string) *Goal {
	mg := loadOrLazyInit(mobId, namesimple)
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	if mg.CurrentGoalId == "" {
		return nil
	}
	for _, g := range mg.Goals {
		if g.Id == mg.CurrentGoalId {
			return g
		}
	}
	return nil // stale id (goal removed since last Recompute)
}

// Recompute runs the chunk-4.2 selection pipeline for the mob:
//  1. Snapshot the goal list and selection state under the read lock.
//  2. Call the pure Select with archetype weights resolved via the
//     registered WeightsLookupFn.
//  3. On a switch, update CurrentGoalId / CurrentSinceRound /
//     LastSwitchRound under the write lock, persist the file, and
//     emit a debug-level structured log line.
//  4. On no switch, do not rewrite the file (avoid per-tick churn).
//
// Called by the per-round tick hook (Task 8) and eagerly from
// Add/Remove/Clear (Task 7). Safe to call with a nil mob — registered
// ContextScore funcs that need mob state should defend themselves.
func Recompute(mobId int, namesimple string, mob *mobs.Mob, nowRound uint64) {
	mg := loadOrLazyInit(mobId, namesimple)

	// Snapshot under read lock.
	cacheMu.RLock()
	goalsSnap := make([]*Goal, len(mg.Goals))
	copy(goalsSnap, mg.Goals)
	prevId := mg.CurrentGoalId
	currentSince := mg.CurrentSinceRound
	lastSwitch := mg.LastSwitchRound
	cacheMu.RUnlock()

	var prev *Goal
	for _, g := range goalsSnap {
		if g.Id == prevId {
			prev = g
			break
		}
	}
	weights := resolveWeights(mob)

	current, switched, reason := Select(goalsSnap, weights, mob, prev,
		currentSince, lastSwitch, nowRound)

	if !switched {
		return // no file write, no log
	}

	// Apply switch under write lock + persist.
	cacheMu.Lock()
	if current == nil {
		mg.CurrentGoalId = ""
		mg.CurrentSinceRound = 0
		mg.LastSwitchRound = 0
	} else {
		mg.CurrentGoalId = current.Id
		mg.CurrentSinceRound = nowRound
		mg.LastSwitchRound = nowRound
	}
	cacheMu.Unlock()

	if err := saveToDisk(mobId, namesimple); err != nil {
		mudlog.Warn("goals.Recompute: save failed", "mob_id", mobId, "error", err)
	}

	// Structured switch log line.
	fromStr := "none"
	if prev != nil {
		fromStr = fmt.Sprintf("%s(%s,%d)", prev.Id, prev.Type, prev.Priority)
	}
	toStr := "none"
	if current != nil {
		toStr = fmt.Sprintf("%s(%s,%d)", current.Id, current.Type, current.Priority)
	}
	mudlog.Debug("goals.switch",
		"mob_id", mobId,
		"from", fromStr,
		"to", toStr,
		"reason_kind", reason.Kind,
		"reason_detail", reason.Detail,
		"round", nowRound)

	// Chunk 4.4: invoke registered plan-state cleanup so the new goal's
	// planner starts with fresh MiscData. Best-effort; nil callback is
	// fine (tests, unboot).
	invokePlanStateClear(mob)
}

// instanceForRecompute returns the first loaded mob instance for the
// given template id, or nil if none. The goals package can't import
// behaviortree but CAN import mobs; we use the latter to give
// Recompute a real *mobs.Mob whenever possible so registered
// ContextScore hooks can read live state.
func instanceForRecompute(mobId int) *mobs.Mob {
	for _, instId := range mobs.GetAllMobInstanceIds() {
		inst := mobs.GetInstance(instId)
		if inst != nil && int(inst.MobId) == mobId {
			return inst
		}
	}
	return nil
}

// seedFromArchetype runs the chunk-4.3 lazy seed for a fresh MobGoals.
// Idempotent under the SeededFromArchetype sentinel. Always flips the
// sentinel and persists once at the end, regardless of seed outcome —
// so the file records "seeding was attempted" even if no defaults are
// registered. Add failures are logged at warn level and skipped —
// partial seeding is preferable to bailing entirely.
//
// Must be called OUTSIDE any cache lock — it calls Add which calls
// loadOrLazyInit which acquires the read lock.
func seedFromArchetype(mobId int, namesimple string, mg *MobGoals) {
	if mg.SeededFromArchetype {
		return
	}
	mob := instanceForRecompute(mobId)
	defaults := resolveArchetypeDefaults(mob)
	for _, d := range defaults {
		g := &Goal{Type: d.Type, Priority: d.Priority, Params: d.Params}
		if _, err := Add(mobId, namesimple, g); err != nil {
			mudlog.Warn("goals.seedFromArchetype: Add failed (skipping)",
				"mob_id", mobId, "type", d.Type, "error", err)
		}
	}
	cacheMu.Lock()
	mg.SeededFromArchetype = true
	cacheMu.Unlock()
	if err := saveToDisk(mobId, namesimple); err != nil {
		mudlog.Warn("goals.seedFromArchetype: save failed",
			"mob_id", mobId, "error", err)
	}
}

// invokeDedupKey calls a registered DedupKey func under panic recovery.
// A panic logs a single-line warning and returns "" (collapses to
// "no key" — same-type goals fall through to coexist freely under
// AllowMultiple semantics). Mirrors how invokeContextScore handles
// panics. Chunk 4.3.
func invokeDedupKey(fn func(g *Goal) string, g *Goal) (key string) {
	defer func() {
		if r := recover(); r != nil {
			mudlog.Warn("goals.dedup_key panic",
				"type", g.Type,
				"goal_id", g.Id,
				"panic", fmt.Sprintf("%v", r))
			key = ""
		}
	}()
	return fn(g)
}
