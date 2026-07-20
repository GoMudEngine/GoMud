package conversations

import (
	"sync"

	"github.com/GoMudEngine/GoMud/internal/relationships"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// ConversationLine is one line in an exchange. Speaker is "A" or "B"
// (the conceptual initiator vs partner; actual NPC role-mapping happens
// per-conversation when the exchange starts).
type ConversationLine struct {
	Speaker string `yaml:"speaker"`
	Text    string `yaml:"text"`
}

// Exchange is a single 2-4 line conversation script.
type Exchange struct {
	Lines []ConversationLine `yaml:"lines"`
}

// Pool is a per-relationship-type collection of generic exchanges, with
// optional subtype sub-pools for richer variation.
type Pool struct {
	Id          string             `yaml:"id"` // must match a relationships.Type
	Description string             `yaml:"description,omitempty"`
	Exchanges   []Exchange         `yaml:"exchanges"`
	Subtypes    map[string]Subpool `yaml:"subtypes,omitempty"`
}

// Subpool is a relationship-subtype-keyed addendum to a Pool's exchanges.
type Subpool struct {
	Exchanges []Exchange `yaml:"exchanges"`
}

// PairOverride adds exchanges specific to a single NPC pair on top of
// (not replacing) the type pool.
type PairOverride struct {
	Id        string     `yaml:"id"`
	MobA      int        `yaml:"mob_a"`
	MobB      int        `yaml:"mob_b"`
	Exchanges []Exchange `yaml:"exchanges"`
}

// pairKey is the registry key for PairOverride. Always normalized so
// (a, b) and (b, a) map to the same entry.
type pairKey struct {
	LowId  int
	HighId int
}

func makePairKey(a, b int) pairKey {
	if a > b {
		a, b = b, a
	}
	return pairKey{LowId: a, HighId: b}
}

// Package-level registries, populated by Load() at startup.
var (
	poolsMu sync.RWMutex
	pools   = map[relationships.Type]*Pool{}

	pairsMu sync.RWMutex
	pairs   = map[pairKey]*PairOverride{}
)

// GetPool returns the type pool for the given relationship type, or nil
// if no pool is registered for that type.
func GetPool(t relationships.Type) *Pool {
	if t == "" {
		return nil
	}
	poolsMu.RLock()
	defer poolsMu.RUnlock()
	return pools[t]
}

// GetPairOverride returns the pair override for the given mob ids,
// normalized so the order doesn't matter. Returns nil if no override
// is registered.
func GetPairOverride(mobA, mobB int) *PairOverride {
	pairsMu.RLock()
	defer pairsMu.RUnlock()
	return pairs[makePairKey(mobA, mobB)]
}

// Test-only registry helpers.
func registerTestPool(p *Pool) {
	poolsMu.Lock()
	defer poolsMu.Unlock()
	pools[relationships.Type(p.Id)] = p
}

func unregisterTestPool(id string) {
	poolsMu.Lock()
	defer poolsMu.Unlock()
	delete(pools, relationships.Type(id))
}

func registerTestPairOverride(po *PairOverride) {
	pairsMu.Lock()
	defer pairsMu.Unlock()
	pairs[makePairKey(po.MobA, po.MobB)] = po
}

func unregisterTestPairOverride(a, b int) {
	pairsMu.Lock()
	defer pairsMu.Unlock()
	delete(pairs, makePairKey(a, b))
}

// Exported test helpers for cross-package tests (hooks package will
// use these in T7's tests).
func RegisterTestPool(p *Pool)                  { registerTestPool(p) }
func UnregisterTestPool(id string)              { unregisterTestPool(id) }
func RegisterTestPairOverride(po *PairOverride) { registerTestPairOverride(po) }
func UnregisterTestPairOverride(a, b int)       { unregisterTestPairOverride(a, b) }

// pickExchange picks an exchange uniformly from the union of:
//   - the relationship-type pool's default exchanges (if registered)
//   - the matching subtype sub-pool's exchanges (if subtype matches)
//   - the pair override's exchanges (if registered for this mob pair)
//
// Returns (Exchange{}, false) if no exchanges are eligible.
func pickExchange(poolId, subtype string, mobA, mobB int) (Exchange, bool) {
	var eligible []Exchange

	if p := GetPool(relationships.Type(poolId)); p != nil {
		eligible = append(eligible, p.Exchanges...)
		if subtype != "" {
			if sub, ok := p.Subtypes[subtype]; ok {
				eligible = append(eligible, sub.Exchanges...)
			}
		}
	}

	if po := GetPairOverride(mobA, mobB); po != nil {
		eligible = append(eligible, po.Exchanges...)
	}

	if len(eligible) == 0 {
		return Exchange{}, false
	}

	idx := util.Rand(len(eligible))
	return eligible[idx], true
}
